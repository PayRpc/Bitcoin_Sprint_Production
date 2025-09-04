// Package main provides a diagnostic tool for blockchain relay connections
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Default endpoints for testing
	defaultEthWSEndpoint = "wss://eth-mainnet.g.alchemy.com/v2/demo"
	defaultSolWSEndpoint = "wss://api.mainnet-beta.solana.com"
	defaultEthRPCEndpoint = "https://cloudflare-eth.com"
	defaultSolRPCEndpoint = "https://api.mainnet-beta.solana.com"
)

var (
	// Test cases for each endpoint
	ethRPCPayloads = []string{
		`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`,
		`{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}`,
		`{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`,
	}

	solRPCPayloads = []string{
		`{"jsonrpc":"2.0","method":"getHealth","params":[],"id":1}`,
		`{"jsonrpc":"2.0","method":"getVersion","params":[],"id":1}`,
		`{"jsonrpc":"2.0","method":"getSlot","params":[],"id":1}`,
	}

	ethWSPayloads = []string{
		`{"jsonrpc":"2.0","method":"eth_subscribe","params":["newHeads"],"id":1}`,
	}

	solWSPayloads = []string{
		`{"jsonrpc":"2.0","method":"slotSubscribe","params":[],"id":1}`,
	}
)

type testResult struct {
	Endpoint    string
	Success     bool
	Latency     time.Duration
	Message     string
	StatusCode  int
	ResponseLen int
}

func main() {
	// Parse command line flags
	ethRPC := flag.String("eth-rpc", defaultEthRPCEndpoint, "Ethereum JSON-RPC endpoint")
	solRPC := flag.String("sol-rpc", defaultSolRPCEndpoint, "Solana JSON-RPC endpoint")
	ethWS := flag.String("eth-ws", defaultEthWSEndpoint, "Ethereum WebSocket endpoint")
	solWS := flag.String("sol-ws", defaultSolWSEndpoint, "Solana WebSocket endpoint")
	timeout := flag.Int("timeout", 10, "Timeout in seconds for each test")
	skipRPC := flag.Bool("skip-rpc", false, "Skip RPC tests")
	skipWS := flag.Bool("skip-ws", false, "Skip WebSocket tests")
	showResponse := flag.Bool("show-response", false, "Show full JSON response")
	testLatency := flag.Bool("latency", false, "Test network latency with multiple pings")
	flag.Parse()

	fmt.Println("Bitcoin Sprint Blockchain Network Diagnostics")
	fmt.Println("============================================")
	fmt.Println()

	// Run tests
	var wg sync.WaitGroup
	results := make(chan testResult, 50)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	if !*skipRPC {
		// Test Ethereum RPC
		wg.Add(1)
		go func() {
			defer wg.Done()
			testJSONRPC(ctx, "Ethereum RPC", *ethRPC, ethRPCPayloads, results, *showResponse)
		}()

		// Test Solana RPC
		wg.Add(1)
		go func() {
			defer wg.Done()
			testJSONRPC(ctx, "Solana RPC", *solRPC, solRPCPayloads, results, *showResponse)
		}()
	}

	if !*skipWS {
		// Test Ethereum WebSocket
		wg.Add(1)
		go func() {
			defer wg.Done()
			testWebSocket(ctx, "Ethereum WS", *ethWS, ethWSPayloads, results, *showResponse)
		}()

		// Test Solana WebSocket
		wg.Add(1)
		go func() {
			defer wg.Done()
			testWebSocket(ctx, "Solana WS", *solWS, solWSPayloads, results, *showResponse)
		}()
	}

	if *testLatency {
		// Test network latency
		wg.Add(1)
		go func() {
			defer wg.Done()
			testNetworkLatency(ctx, *ethRPC, *solRPC, results)
		}()
	}

	// Collect and display results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Print results as they come in
	successCount := 0
	totalTests := 0
	for result := range results {
		totalTests++
		if result.Success {
			successCount++
			fmt.Printf("✅ %s: %s (%d ms)\n", result.Endpoint, result.Message, result.Latency.Milliseconds())
		} else {
			fmt.Printf("❌ %s: %s\n", result.Endpoint, result.Message)
		}
	}

	// Print summary
	fmt.Println("\nTest Summary")
	fmt.Println("===========")
	fmt.Printf("Total tests: %d\n", totalTests)
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", totalTests-successCount)
	
	if successCount < totalTests {
		os.Exit(1)
	}
}

// testJSONRPC tests JSON-RPC endpoints
func testJSONRPC(ctx context.Context, name, endpoint string, payloads []string, results chan<- testResult, showResponse bool) {
	// Parse URL
	u, err := url.Parse(endpoint)
	if err != nil {
		results <- testResult{
			Endpoint: name,
			Success:  false,
			Message:  fmt.Sprintf("Invalid URL: %v", err),
		}
		return
	}

	// DNS resolution test
	start := time.Now()
	ips, err := net.LookupIP(u.Hostname())
	dnsLatency := time.Since(start)
	
	if err != nil {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (DNS)", name),
			Success:  false,
			Message:  fmt.Sprintf("DNS lookup failed: %v", err),
		}
	} else {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (DNS)", name),
			Success:  true,
			Latency:  dnsLatency,
			Message:  fmt.Sprintf("Resolved to %d IPs: %v", len(ips), ips[0]),
		}
	}

	// TCP connection test
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	
	start = time.Now()
	conn, err := net.DialTimeout("tcp", u.Hostname()+":"+port, 5*time.Second)
	tcpLatency := time.Since(start)
	
	if err != nil {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (TCP)", name),
			Success:  false,
			Message:  fmt.Sprintf("TCP connection failed: %v", err),
		}
		return
	}
	conn.Close()
	
	results <- testResult{
		Endpoint: fmt.Sprintf("%s (TCP)", name),
		Success:  true,
		Latency:  tcpLatency,
		Message:  "TCP connection successful",
	}

	// HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
			DisableKeepAlives: false,
			IdleConnTimeout:   90 * time.Second,
		},
	}

	// Test each RPC payload
	for i, payload := range payloads {
		start = time.Now()
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(payload))
		if err != nil {
			results <- testResult{
				Endpoint: fmt.Sprintf("%s (Payload %d)", name, i+1),
				Success:  false,
				Message:  fmt.Sprintf("Error creating request: %v", err),
			}
			continue
		}
		
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Bitcoin-Sprint/1.0 Network-Diagnostics")
		
		resp, err := client.Do(req)
		latency := time.Since(start)
		
		if err != nil {
			results <- testResult{
				Endpoint: fmt.Sprintf("%s (Payload %d)", name, i+1),
				Success:  false,
				Message:  fmt.Sprintf("Request failed: %v", err),
				Latency:  latency,
			}
			continue
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		
		if err != nil {
			results <- testResult{
				Endpoint:   fmt.Sprintf("%s (Payload %d)", name, i+1),
				Success:    false,
				Message:    fmt.Sprintf("Error reading response: %v", err),
				StatusCode: resp.StatusCode,
				Latency:    latency,
			}
			continue
		}

		responseMsg := fmt.Sprintf("Response code %d, length %d bytes", resp.StatusCode, len(body))
		if showResponse && len(body) > 0 {
			if len(body) > 500 {
				responseMsg = fmt.Sprintf("%s\n%s... (truncated)", responseMsg, string(body[:500]))
			} else {
				responseMsg = fmt.Sprintf("%s\n%s", responseMsg, string(body))
			}
		}
		
		results <- testResult{
			Endpoint:    fmt.Sprintf("%s (Payload %d)", name, i+1),
			Success:     resp.StatusCode >= 200 && resp.StatusCode < 300,
			Message:     responseMsg,
			StatusCode:  resp.StatusCode,
			ResponseLen: len(body),
			Latency:     latency,
		}
	}
}

// testWebSocket tests WebSocket endpoints
func testWebSocket(ctx context.Context, name, endpoint string, payloads []string, results chan<- testResult, showResponse bool) {
	// Parse URL
	u, err := url.Parse(endpoint)
	if err != nil {
		results <- testResult{
			Endpoint: name,
			Success:  false,
			Message:  fmt.Sprintf("Invalid URL: %v", err),
		}
		return
	}

	// DNS resolution test
	start := time.Now()
	ips, err := net.LookupIP(u.Hostname())
	dnsLatency := time.Since(start)
	
	if err != nil {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (DNS)", name),
			Success:  false,
			Message:  fmt.Sprintf("DNS lookup failed: %v", err),
		}
	} else {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (DNS)", name),
			Success:  true,
			Latency:  dnsLatency,
			Message:  fmt.Sprintf("Resolved to %d IPs: %v", len(ips), ips[0]),
		}
	}

	// WebSocket connection test
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}
	
	headers := http.Header{}
	headers.Add("User-Agent", "Bitcoin-Sprint/1.0 Network-Diagnostics")
	
	start = time.Now()
	c, _, err := dialer.DialContext(ctx, endpoint, headers)
	wsLatency := time.Since(start)
	
	if err != nil {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (Connection)", name),
			Success:  false,
			Message:  fmt.Sprintf("WebSocket connection failed: %v", err),
			Latency:  wsLatency,
		}
		return
	}
	defer c.Close()
	
	results <- testResult{
		Endpoint: fmt.Sprintf("%s (Connection)", name),
		Success:  true,
		Message:  "WebSocket connection successful",
		Latency:  wsLatency,
	}

	// Set read deadline
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Test each WebSocket payload
	for i, payload := range payloads {
		start = time.Now()
		err := c.WriteMessage(websocket.TextMessage, []byte(payload))
		if err != nil {
			results <- testResult{
				Endpoint: fmt.Sprintf("%s (Payload %d)", name, i+1),
				Success:  false,
				Message:  fmt.Sprintf("Error sending message: %v", err),
			}
			continue
		}
		
		_, message, err := c.ReadMessage()
		latency := time.Since(start)
		
		if err != nil {
			results <- testResult{
				Endpoint: fmt.Sprintf("%s (Payload %d)", name, i+1),
				Success:  false,
				Message:  fmt.Sprintf("Error receiving message: %v", err),
				Latency:  latency,
			}
			continue
		}

		responseMsg := fmt.Sprintf("Received %d bytes", len(message))
		if showResponse && len(message) > 0 {
			if len(message) > 500 {
				responseMsg = fmt.Sprintf("%s\n%s... (truncated)", responseMsg, string(message[:500]))
			} else {
				responseMsg = fmt.Sprintf("%s\n%s", responseMsg, string(message))
			}
		}
		
		results <- testResult{
			Endpoint:    fmt.Sprintf("%s (Payload %d)", name, i+1),
			Success:     true,
			Message:     responseMsg,
			ResponseLen: len(message),
			Latency:     latency,
		}
	}

	// Test ping/pong
	start = time.Now()
	err = c.WriteMessage(websocket.PingMessage, []byte("ping"))
	if err != nil {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (Ping)", name),
			Success:  false,
			Message:  fmt.Sprintf("Error sending ping: %v", err),
		}
		return
	}
	
	// Read messages until we get a pong or timeout
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	messageType, message, err := c.ReadMessage()
	pingLatency := time.Since(start)
	
	if err != nil {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (Ping)", name),
			Success:  false,
			Message:  fmt.Sprintf("Error receiving pong: %v", err),
			Latency:  pingLatency,
		}
	} else if messageType == websocket.PongMessage {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (Ping)", name),
			Success:  true,
			Message:  "Ping/Pong successful",
			Latency:  pingLatency,
		}
	} else {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (Ping)", name),
			Success:  false,
			Message:  fmt.Sprintf("Expected pong message, got message type %d", messageType),
			Latency:  pingLatency,
		}
	}

	// Close the connection cleanly
	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s (Close)", name),
			Success:  false,
			Message:  fmt.Sprintf("Error closing connection: %v", err),
		}
		return
	}
	
	results <- testResult{
		Endpoint: fmt.Sprintf("%s (Close)", name),
		Success:  true,
		Message:  "Connection closed properly",
	}
}

// testNetworkLatency tests network latency by sending ICMP pings or TCP connection tests
func testNetworkLatency(ctx context.Context, ethEndpoint, solEndpoint string, results chan<- testResult) {
	ethURL, err := url.Parse(ethEndpoint)
	if err != nil {
		results <- testResult{
			Endpoint: "Ethereum Latency",
			Success:  false,
			Message:  fmt.Sprintf("Invalid URL: %v", err),
		}
		return
	}
	
	solURL, err := url.Parse(solEndpoint)
	if err != nil {
		results <- testResult{
			Endpoint: "Solana Latency",
			Success:  false,
			Message:  fmt.Sprintf("Invalid URL: %v", err),
		}
		return
	}

	// Test Ethereum endpoint latency
	testEndpointLatency(ctx, "Ethereum", ethURL.Hostname(), results)
	
	// Test Solana endpoint latency
	testEndpointLatency(ctx, "Solana", solURL.Hostname(), results)
}

func testEndpointLatency(ctx context.Context, name, hostname string, results chan<- testResult) {
	// Perform multiple TCP connection tests to measure latency
	var latencies []time.Duration
	totalTests := 5
	
	for i := 0; i < totalTests; i++ {
		select {
		case <-ctx.Done():
			results <- testResult{
				Endpoint: fmt.Sprintf("%s Latency", name),
				Success:  false,
				Message:  "Latency test cancelled",
			}
			return
		default:
			start := time.Now()
			port := "443" // Default to HTTPS port
			
			conn, err := net.DialTimeout("tcp", hostname+":"+port, 5*time.Second)
			elapsed := time.Since(start)
			
			if err == nil {
				conn.Close()
				latencies = append(latencies, elapsed)
			}
			
			time.Sleep(500 * time.Millisecond) // Wait between tests
		}
	}
	
	if len(latencies) == 0 {
		results <- testResult{
			Endpoint: fmt.Sprintf("%s Latency", name),
			Success:  false,
			Message:  "No successful latency measurements",
		}
		return
	}
	
	// Calculate average latency
	var totalLatency time.Duration
	for _, latency := range latencies {
		totalLatency += latency
	}
	avgLatency := totalLatency / time.Duration(len(latencies))
	
	// Calculate min/max latency
	minLatency := latencies[0]
	maxLatency := latencies[0]
	for _, latency := range latencies {
		if latency < minLatency {
			minLatency = latency
		}
		if latency > maxLatency {
			maxLatency = latency
		}
	}
	
	results <- testResult{
		Endpoint: fmt.Sprintf("%s Latency", name),
		Success:  true,
		Message:  fmt.Sprintf("Avg: %d ms, Min: %d ms, Max: %d ms", 
			avgLatency.Milliseconds(), minLatency.Milliseconds(), maxLatency.Milliseconds()),
		Latency:  avgLatency,
	}
}
