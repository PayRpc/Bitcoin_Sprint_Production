// internal/headers/wire.go
package headers

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"golang.org/x/sync/errgroup"
)

/* -------- Domain types -------- */

type Header struct {
	Hash   [32]byte // big-endian
	Height uint32
	Raw    []byte // serialized block header bytes (80B for BTC)
}

type Node interface {
	GetBlockHeader(ctx context.Context, height int) (Header, error)
	// (optional) Add GetBlockCount(ctx) (int, error) if you want a tip poller here.
}

/* -------- FastRead (hedged + quorum) -------- */

type FastRead struct {
	nodes  []Node
	cb     *gobreaker.CircuitBreaker
	hedged time.Duration // when to fan-out to the rest
	q      int           // quorum (e.g. 2 of 3)
}

func NewFastRead(nodes []Node, hedged time.Duration, quorum int) *FastRead {
	if quorum < 1 {
		quorum = 1
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:          "header-read",
		MaxRequests:   5,
		Interval:      30 * time.Second,
		Timeout:       10 * time.Second,
		ReadyToTrip:   func(c gobreaker.Counts) bool { return c.ConsecutiveFailures >= 5 },
		IsSuccessful:  nil,
		OnStateChange: nil,
	})
	return &FastRead{nodes: nodes, cb: cb, hedged: hedged, q: quorum}
}

func (fr *FastRead) GetHeader(ctx context.Context, height int) (Header, error) {
	if len(fr.nodes) == 0 {
		return Header{}, errors.New("no nodes configured")
	}

	g, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	resCh := make(chan Header, len(fr.nodes))
	errCh := make(chan error, len(fr.nodes))

	launch := func(n Node) {
		g.Go(func() error {
			// short timeout with jitter
			d := 800*time.Millisecond + time.Duration(time.Now().UnixNano()%200)*time.Millisecond
			c, cCancel := context.WithTimeout(ctx, d)
			defer cCancel()
			hdrAny, err := fr.cb.Execute(func() (any, error) {
				return n.GetBlockHeader(c, height)
			})
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return nil
			}
			hdr := hdrAny.(Header)
			select {
			case resCh <- hdr:
			default:
			}
			return nil
		})
	}

	// Fire primary (assume nodes are EWMA-ranked externally)
	launch(fr.nodes[0])

	hedgeTimer := time.NewTimer(fr.hedged)
	defer hedgeTimer.Stop()
	deadline := time.NewTimer(1500 * time.Millisecond)
	defer deadline.Stop()

	agree := make(map[[32]byte]int)
	hedged := false

	for {
		select {
		case <-ctx.Done():
			return Header{}, ctx.Err()
		case <-deadline.C:
			return Header{}, context.DeadlineExceeded
		case <-hedgeTimer.C:
			if !hedged {
				for i := 1; i < len(fr.nodes); i++ {
					launch(fr.nodes[i])
				}
				hedged = true
			}
		case hdr := <-resCh:
			agree[hdr.Hash]++
			if agree[hdr.Hash] >= fr.q {
				cancel()
				_ = g.Wait()
				return hdr, nil
			}
		case <-errCh:
			// keep collecting until quorum/timeout
		}
	}
}

/* -------- Snapshots + tail (for /latest and /stream) -------- */

type Snapshot struct {
	BlockHash [32]byte
	Bytes     []byte // raw header bytes
	Height    uint32
}

var (
	snapMu  sync.RWMutex
	current Snapshot

	subMu sync.Mutex
	subs  = map[chan Snapshot]struct{}{}
)

func SetSnapshot(s Snapshot) {
	snapMu.Lock()
	current = s
	snapMu.Unlock()

	// fan out to subscribers (non-blocking)
	subMu.Lock()
	for ch := range subs {
		select {
		case ch <- s:
		default:
		}
	}
	subMu.Unlock()
}

func getSnapshot() Snapshot {
	snapMu.RLock()
	defer snapMu.RUnlock()
	return current
}

func Subscribe() chan Snapshot {
	ch := make(chan Snapshot, 8)
	subMu.Lock()
	subs[ch] = struct{}{}
	subMu.Unlock()
	return ch
}

func Unsubscribe(ch chan Snapshot) {
	subMu.Lock()
	delete(subs, ch)
	subMu.Unlock()
	close(ch)
}

/* -------- HTTP: /headers/latest (cached, compressed) -------- */

func LatestHandler(w http.ResponseWriter, r *http.Request) {
	s := getSnapshot()
	etag := `"` + hex.EncodeToString(s.BlockHash[:]) + `"`

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=30, immutable")
	w.Header().Set("Vary", "Accept-Encoding")

	enc := negotiateEncoding(r.Header.Get("Accept-Encoding"))
	switch enc {
	case "gzip":
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		_, _ = gz.Write(s.Bytes)
	default:
		_, _ = w.Write(s.Bytes)
	}
}

func negotiateEncoding(accept string) string {
	accept = strings.ToLower(accept)
	if strings.Contains(accept, "gzip") {
		return "gzip"
	}
	return "identity"
}

/* -------- HTTP: /headers/stream (SSE) -------- */

func StreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}

	ch := Subscribe()
	defer Unsubscribe(ch)

	// send the current tip immediately
	if cur := getSnapshot(); len(cur.Bytes) > 0 {
		writeHeaderEvent(w, cur)
		flusher.Flush()
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			// SSE comment = heartbeat
			_, _ = io.WriteString(w, ":\n\n")
			flusher.Flush()
		case s := <-ch:
			writeHeaderEvent(w, s)
			flusher.Flush()
		}
	}
}

func writeHeaderEvent(w http.ResponseWriter, s Snapshot) {
	// Tiny payload: base64(raw header)
	fmt.Fprint(w, "event: header\n")
	fmt.Fprintf(w, "id: %d\n", s.Height)
	fmt.Fprintf(w, "data: %s\n\n", base64.StdEncoding.EncodeToString(s.Bytes))
}

/* -------- Helper: compute block hash from raw header -------- */

func DoubleSHA256(b []byte) [32]byte {
	h1 := sha256.Sum256(b)
	h2 := sha256.Sum256(h1[:])
	// Bitcoin displays little-endian as hex-reversed; we keep big-endian wire.
	var out [32]byte
	copy(out[:], h2[:]) // big-endian representation here
	return out
}

/* -------- Background tip updater (plug any source you want) -------- */

// RunTipUpdater is an example loop that you call from main.
// It should:
//   - figure out the next height to fetch,
//   - call fr.GetHeader(ctx, next),
//   - setSnapshot when a new header arrives.
// You can swap this with your own notifier/ZMQ pipeline.
func RunTipUpdater(ctx context.Context, fr *FastRead, startHeight int, every time.Duration) {
	h := startHeight
	ticker := time.NewTicker(every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hdr, err := fr.GetHeader(ctx, h)
			if err != nil {
				continue
			}
			if int(hdr.Height) == h {
				SetSnapshot(Snapshot{
					BlockHash: hdr.Hash,
					Bytes:     hdr.Raw,
					Height:    hdr.Height,
				})
				h++ // advance
			}
		}
	}
}
