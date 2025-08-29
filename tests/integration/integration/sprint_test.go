package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	apiHost     = getEnv("API_HOST", "127.0.0.1")
	apiPort     = getEnv("API_PORT", "8080")
	rpcUser     = getEnv("BTC_RPC_USER", "sprint")
	rpcPass     = getEnv("BTC_RPC_PASS", "integration")
	rpcHost     = getEnv("BTC_RPC_HOST", "127.0.0.1:8332")
	testAddress = ""
)

func getEnv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

// RPCRequest for bitcoind
type rpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  interface{}     `json:"error"`
	ID     string          `json:"id"`
}

// callBitcoinRPC makes a direct RPC call to bitcoind
func callBitcoinRPC(method string, params ...interface{}) ([]byte, error) {
	reqBody, _ := json.Marshal(rpcRequest{
		Jsonrpc: "1.0",
		ID:      "sprint-test",
		Method:  method,
		Params:  params,
	})

	url := fmt.Sprintf("http://%s:%s@%s/", rpcUser, rpcPass, rpcHost)
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("rpc post error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var res rpcResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("rpc decode error: %w", err)
	}
	if res.Error != nil {
		return nil, fmt.Errorf("rpc error: %+v", res.Error)
	}
	return res.Result, nil
}

// getLatestBlock calls Sprint /latest endpoint
func getLatestBlock(t *testing.T) map[string]interface{} {
	url := fmt.Sprintf("http://%s:%s/latest", apiHost, apiPort)
	resp, err := http.Get(url)
	require.NoError(t, err, "GET /latest failed")
	defer resp.Body.Close()

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err, "decode /latest response failed")

	return data
}

// getHealth calls Sprint /health endpoint
func getHealth(t *testing.T) map[string]interface{} {
	url := fmt.Sprintf("http://%s:%s/health", apiHost, apiPort)
	resp, err := http.Get(url)
	require.NoError(t, err, "GET /health failed")
	defer resp.Body.Close()

	var data map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	require.NoError(t, err, "decode /health response failed")

	return data
}

func TestSprintIntegration(t *testing.T) {
	// 1. Get a new address
	addrBytes, err := callBitcoinRPC("getnewaddress")
	require.NoError(t, err)
	json.Unmarshal(addrBytes, &testAddress)
	require.NotEmpty(t, testAddress, "expected valid regtest address")

	// 2. Generate 1 block to that address
	_, err = callBitcoinRPC("generatetoaddress", 1, testAddress)
	require.NoError(t, err, "failed to mine block in regtest")

	// 3. Wait up to 10s for Sprint to detect
	var latest map[string]interface{}
	found := false
	for i := 0; i < 20; i++ {
		latest = getLatestBlock(t)
		if latest["hash"] != nil && latest["height"] != nil {
			found = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	require.True(t, found, "Sprint did not detect new block in time")

	t.Logf("Sprint detected block: hash=%s height=%v", latest["hash"], latest["height"])

	// 4. Check health endpoint
	health := getHealth(t)
	require.Equal(t, "ok", health["status"], "Sprint /health should be ok")
}
