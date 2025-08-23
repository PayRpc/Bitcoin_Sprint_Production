//go:build bench
// +build bench

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"time"
)

type rpcResp struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type chainInfo struct {
	BestHash string `json:"bestblockhash"`
	Height   int    `json:"blocks"`
}

func callRPC(ctx context.Context, c *http.Client, url, user, pass string) (chainInfo, error) {
	body := []byte(`{"jsonrpc":"1.0","id":"bench","method":"getblockchaininfo","params":[]}`)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return chainInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return chainInfo{}, fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	var env rpcResp
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return chainInfo{}, err
	}
	if env.Error != nil {
		return chainInfo{}, fmt.Errorf("rpc error: %d %s", env.Error.Code, env.Error.Message)
	}
	var out chainInfo
	if err := json.Unmarshal(env.Result, &out); err != nil {
		var nested struct {
			Result chainInfo `json:"result"`
		}
		if err2 := json.Unmarshal(env.Result, &nested); err2 != nil {
			return chainInfo{}, fmt.Errorf("unexpected result shape: %v / %v", err, err2)
		}
		return nested.Result, nil
	}
	return out, nil
}

func main() {
	var (
		rpcURL  = flag.String("rpc-url", "http://127.0.0.1:8332", "Bitcoin Core RPC URL")
		rpcUser = flag.String("rpc-user", "", "RPC username")
		rpcPass = flag.String("rpc-pass", "", "RPC password")
		sMs     = flag.Int("sprint-interval-ms", 1000, "Sprint poll interval (ms)")
		bMs     = flag.Int("baseline-interval-ms", 5000, "Baseline poll interval (ms)")
		count   = flag.Int("count", 25, "Number of blocks to sample")
	)
	flag.Parse()

	client := &http.Client{Timeout: 5 * time.Second}
	ctx := context.Background()

	ci, err := callRPC(ctx, client, *rpcURL, *rpcUser, *rpcPass)
	if err != nil {
		log.Fatalf("RPC check failed: %v", err)
	}
	log.Printf("Start height=%d hash=%s", ci.Height, hashPrefix(ci.BestHash))

	type det struct {
		hash   string
		height int
		t      time.Time
	}
	sCh := make(chan det, 10)
	bCh := make(chan det, 10)

	startPoller := func(interval time.Duration, out chan det) {
		last := ""
		for {
			info, err := callRPC(ctx, client, *rpcURL, *rpcUser, *rpcPass)
			if err == nil && info.BestHash != "" && info.BestHash != last {
				last = info.BestHash
				out <- det{hash: info.BestHash, height: info.Height, t: time.Now()}
			}
			time.Sleep(interval)
		}
	}

	go startPoller(time.Duration(*sMs)*time.Millisecond, sCh)
	go startPoller(time.Duration(*bMs)*time.Millisecond, bCh)

	type sample struct {
		h    int
		hash string
		s, b time.Time
	}
	seen := map[string]*sample{}
	results := make([]sample, 0, *count)

	for len(results) < *count {
		select {
		case a := <-sCh:
			v := seen[a.hash]
			if v == nil {
				v = &sample{h: a.height, hash: a.hash}
				seen[a.hash] = v
			}
			if v.s.IsZero() {
				v.s = a.t
			}
		case a := <-bCh:
			v := seen[a.hash]
			if v == nil {
				v = &sample{h: a.height, hash: a.hash}
				seen[a.hash] = v
			}
			if v.b.IsZero() {
				v.b = a.t
			}
		}
		for k, v := range seen {
			if !v.s.IsZero() && !v.b.IsZero() {
				results = append(results, *v)
				delete(seen, k)
				lead := v.b.Sub(v.s).Milliseconds()
				log.Printf("h=%d %s lead=%dms", v.h, k[:8], lead)
				break
			}
		}
	}

	leads := make([]int, 0, len(results))
	for _, v := range results {
		leads = append(leads, int(v.b.Sub(v.s).Milliseconds()))
	}
	sort.Ints(leads)
	median := leads[len(leads)/2]
	p95 := leads[int(math.Ceil(float64(len(leads))*0.95))-1]
	log.Printf("Summary: n=%d median=%dms p95=%dms", len(leads), median, p95)
}

func hashPrefix(h string) string {
	if len(h) < 8 {
		return h
	}
	return h[:8]
}
