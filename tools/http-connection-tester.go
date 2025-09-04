// Package main provides an HTTP server connectivity test tool
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	// Define command-line flags
	host := flag.String("host", "127.0.0.1", "Host to connect to")
	port := flag.Int("port", 9000, "Port to connect to")
	timeout := flag.Int("timeout", 5, "Connection timeout in seconds")
	method := flag.String("method", "GET", "HTTP method")
	path := flag.String("path", "/health", "HTTP path")
	retries := flag.Int("retries", 3, "Number of connection retries")
	verbose := flag.Bool("v", false, "Verbose output")
	useTLS := flag.Bool("tls", false, "Use HTTPS (TLS)")
	showHeaders := flag.Bool("headers", false, "Show response headers")
	showContent := flag.Bool("content", false, "Show response content")
	flag.Parse()

	// Configure HTTP transport with detailed timeouts
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(*timeout) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Allow self-signed certificates
		},
	}

	// Create HTTP client
	client := &http.Client{
		Timeout:   time.Duration(*timeout) * time.Second,
		Transport: transport,
	}

	// Build URL
	scheme := "http"
	if *useTLS {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d%s", scheme, *host, *port, *path)

	fmt.Printf("Bitcoin Sprint HTTP Connection Test\n")
	fmt.Printf("==================================\n")
	fmt.Printf("Target URL: %s\n", url)
	fmt.Printf("Method: %s\n", *method)
	fmt.Printf("Timeout: %d seconds\n", *timeout)
	fmt.Printf("Retries: %d\n", *retries)
	fmt.Printf("\nRunning test...\n\n")

	// Try raw TCP connection first to see if port is open
	testTCPConnection(*host, *port, time.Duration(*timeout)*time.Second)

	// Attempt HTTP connection with retries
	var lastErr error
	var resp *http.Response
	success := false

	for attempt := 1; attempt <= *retries; attempt++ {
		startTime := time.Now()
		fmt.Printf("[Attempt %d/%d] Connecting to %s... ", attempt, *retries, url)

		// Create request with context
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
		req, err := http.NewRequestWithContext(ctx, *method, url, nil)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			cancel()
			lastErr = err
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
			continue
		}

		// Add some standard headers
		req.Header.Set("User-Agent", "BitcoinSprint-HTTP-Tester/1.0")
		req.Header.Set("Accept", "*/*")

		// Perform the HTTP request
		resp, err = client.Do(req)
		elapsed := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Printf("FAILED (%dms): %v\n", elapsed, err)
			if *verbose {
				if netErr, ok := err.(net.Error); ok {
					fmt.Printf("  Network error: timeout=%v, temporary=%v\n", netErr.Timeout(), netErr.Temporary())
				}
				if strings.Contains(err.Error(), "connection refused") {
					fmt.Printf("  Connection refused: The server is not running or is not accepting connections on this port.\n")
				} else if strings.Contains(err.Error(), "timeout") {
					fmt.Printf("  Timeout: The server did not respond within %d seconds.\n", *timeout)
				}
			}
			cancel()
			lastErr = err
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
			continue
		}

		// Success!
		fmt.Printf("SUCCESS (%dms): Status %s\n", elapsed, resp.Status)
		success = true
		
		// Show response details
		if *showHeaders {
			fmt.Printf("\nResponse Headers:\n")
			for name, values := range resp.Header {
				fmt.Printf("  %s: %s\n", name, strings.Join(values, ", "))
			}
		}
		
		if *showContent {
			fmt.Printf("\nResponse Content:\n")
			buf := make([]byte, 8192) // 8KB buffer
			n, err := resp.Body.Read(buf)
			if err != nil && err.Error() != "EOF" {
				fmt.Printf("Error reading response body: %v\n", err)
			} else {
				fmt.Printf("%s\n", string(buf[:n]))
			}
		}

		resp.Body.Close()
		cancel()
		break
	}

	if !success {
		fmt.Printf("\nAll connection attempts failed.\n")
		fmt.Printf("Last error: %v\n", lastErr)
		fmt.Printf("\nDiagnostic information:\n")
		fmt.Printf("1. Check if the server is running\n")
		fmt.Printf("2. Verify the server is listening on %s:%d\n", *host, *port)
		fmt.Printf("3. Check for any firewall or network policy blocking the connection\n")
		fmt.Printf("4. Ensure the server's HTTP handler for %s is properly registered\n", *path)
		os.Exit(1)
	}
}

func testTCPConnection(host string, port int, timeout time.Duration) {
	fmt.Printf("Testing TCP connection to %s:%d... ", host, port)
	startTime := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	elapsed := time.Since(startTime).Milliseconds()
	
	if err != nil {
		fmt.Printf("FAILED (%dms): %v\n", elapsed, err)
		fmt.Printf("  TCP connection failed - the port may not be open or accessible.\n")
	} else {
		fmt.Printf("SUCCESS (%dms): Connection established\n", elapsed)
		fmt.Printf("  TCP port is open and accepting connections.\n")
		conn.Close()
	}
	fmt.Println()
}
