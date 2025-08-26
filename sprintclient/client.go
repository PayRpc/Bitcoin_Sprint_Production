package sprintclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SprintClient handles both Sprint API and Core RPC as fallback
type SprintClient struct {
	sprintURL string
	coreURL   string
	rpcUser   string
	rpcPass   string
	client    *http.Client
}

// NewSprintClient returns a new client
func NewSprintClient(sprintURL, coreURL, rpcUser, rpcPass string) *SprintClient {
	return &SprintClient{
		sprintURL: sprintURL,
		coreURL:   coreURL,
		rpcUser:   rpcUser,
		rpcPass:   rpcPass,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetStatus returns a status map. Tries Sprint first then falls back to Core RPC (getblockchaininfo).
func (c *SprintClient) GetStatus() (map[string]any, error) {
	// try Sprint API
	if data, err := c.getJSON(c.sprintURL + "/status"); err == nil {
		return map[string]any{"source": "sprint", "data": data}, nil
	}

	// fallback to core
	res, err := c.CoreRPC("getblockchaininfo", nil)
	if err != nil {
		return nil, err
	}
	// CoreRPC returns a JSON-RPC envelope: { result, error, id }
	if inner, ok := res["result"]; ok {
		if m, ok := inner.(map[string]any); ok {
			return map[string]any{"source": "core", "data": m}, nil
		}
		return map[string]any{"source": "core", "data": inner}, nil
	}
	// if no result field, return the raw envelope under data
	return map[string]any{"source": "core", "data": res}, nil
}

// GetLatest calls Sprint /latest (no core equivalent) and returns raw data or error
func (c *SprintClient) GetLatest() (map[string]any, error) {
	if data, err := c.getJSON(c.sprintURL + "/latest"); err != nil {
		return nil, err
	} else {
		return map[string]any{"source": "sprint", "data": data}, nil
	}
}

// GetMetrics calls Sprint /metrics
func (c *SprintClient) GetMetrics() (map[string]any, error) {
	if data, err := c.getJSON(c.sprintURL + "/metrics"); err != nil {
		return nil, err
	} else {
		return map[string]any{"source": "sprint", "data": data}, nil
	}
}

// CoreRPC performs a JSON-RPC POST to Bitcoin Core
func (c *SprintClient) CoreRPC(method string, params []any) (map[string]any, error) {
	payload := map[string]any{
		"jsonrpc": "1.0",
		"id":      "sprintclient",
		"method":  method,
		"params":  params,
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", c.coreURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.rpcUser, c.rpcPass)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, errors.New(fmt.Sprintf("core rpc http %d: %s", resp.StatusCode, string(b)))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// getJSON performs a simple GET and decodes JSON into map[string]any
func (c *SprintClient) getJSON(url string) (map[string]any, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}
