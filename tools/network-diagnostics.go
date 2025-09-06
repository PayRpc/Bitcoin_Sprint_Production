// network-diagnostics.go
// A tool to diagnose network connectivity issues with Bitcoin Sprint.

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

func main() {
	fmt.Println("Bitcoin Sprint Network Diagnostics")
	fmt.Println("=================================")

	// Test HTTP server
	checkHTTPServer()

	// Test relay endpoints
	checkRelayEndpoints()

	// Test DNS resolution
	checkDNSResolution()

	// Test loopback interfaces
	checkLoopback()

	fmt.Println("\nDiagnostics completed. Check the results above.")
}

func checkHTTPServer() {
	fmt.Println("\n1. Testing HTTP Server Bindings:")

	// Test if we can connect to the server on all interfaces
	ports := []int{9000}
	interfaces := []string{"127.0.0.1", "localhost", "0.0.0.0"}

	for _, port := range ports {
		fmt.Printf("\n  Testing port %d:\n", port)

		for _, iface := range interfaces {
			addr := fmt.Sprintf("%s:%d", iface, port)

			// Try to create a listener to see if the port is available
			l, err := net.Listen("tcp", addr)
			if err != nil {
				fmt.Printf("  ✗ %s - Port in use: %v\n", addr, err)

				// Try to connect to it as a client
				client := http.Client{Timeout: 3 * time.Second}
				url := fmt.Sprintf("http://%s/health", addr)
				resp, err := client.Get(url)
				if err != nil {
					fmt.Printf("    - Cannot connect as client: %v\n", err)
				} else {
					fmt.Printf("    - Connected as client successfully! Status: %d\n", resp.StatusCode)
					resp.Body.Close()
				}
			} else {
				fmt.Printf("  ✓ %s - Port is available\n", addr)
				l.Close()
			}
		}
	}
}

func checkRelayEndpoints() {
	fmt.Println("\n2. Testing WebSocket Relay Endpoints:")

	endpoints := []string{
		"wss://ethereum.publicnode.com",
		"wss://cloudflare-eth.com",
		"wss://rpc.ankr.com/eth/ws",
		"wss://api.mainnet-beta.solana.com",
		"wss://solana.publicnode.com",
		"wss://rpc.ankr.com/solana",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("\n  Testing %s:\n", endpoint)

		// Parse host and port from endpoint
		host := endpoint[6:] // Remove "wss://"
		if host[len(host)-1] == '/' {
			host = host[:len(host)-1] // Remove trailing slash
		}

		// Default to 443 if no port specified
		port := "443"

		// Try to resolve DNS
		fmt.Printf("    - DNS Lookup: ")
		ips, err := net.LookupHost(host)
		if err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
		} else {
			fmt.Printf("✓ Success, resolved to %v\n", ips)
		}

		// Try to establish TCP connection
		fmt.Printf("    - TCP Connection: ")
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := dialer.DialContext(ctx, "tcp", host+":"+port)
		if err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
		} else {
			fmt.Printf("✓ Connected successfully\n")
			conn.Close()
		}

		// Try to establish TLS connection
		fmt.Printf("    - TLS Handshake: ")
		tlsConn, err := tls.Dial("tcp", host+":"+port, &tls.Config{
			InsecureSkipVerify: false,
		})
		if err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
		} else {
			fmt.Printf("✓ TLS handshake successful\n")
			tlsConn.Close()
		}
	}
}

func checkDNSResolution() {
	fmt.Println("\n3. Testing DNS Resolution:")

	domains := []string{
		"ethereum.publicnode.com",
		"cloudflare-eth.com",
		"rpc.ankr.com",
		"api.mainnet-beta.solana.com",
		"solana.publicnode.com",
	}

	for _, domain := range domains {
		fmt.Printf("  • %s: ", domain)
		ips, err := net.LookupHost(domain)
		if err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
		} else {
			fmt.Printf("✓ %v\n", ips)
		}
	}
}

func checkLoopback() {
	fmt.Println("\n4. Testing Loopback Interfaces:")

	interfaces := []string{"127.0.0.1", "::1", "localhost"}

	for _, iface := range interfaces {
		fmt.Printf("  • %s: ", iface)

		// Create a temporary TCP listener on a random port
		l, err := net.Listen("tcp", iface+":0")
		if err != nil {
			fmt.Printf("✗ Cannot bind: %v\n", err)
			continue
		}

		port := l.Addr().(*net.TCPAddr).Port

		// Create a simple HTTP server
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "OK")
		})

		server := &http.Server{
			Handler: mux,
		}

		go func() {
			server.Serve(l)
		}()

		// Test connecting to it
		time.Sleep(100 * time.Millisecond)
		client := http.Client{Timeout: 1 * time.Second}
		_, err = client.Get(fmt.Sprintf("http://%s:%d/", iface, port))
		if err != nil {
			fmt.Printf("✗ Cannot connect: %v\n", err)
		} else {
			fmt.Printf("✓ Interface is working correctly\n")
		}

		// Cleanup
		server.Shutdown(context.Background())
	}
}
