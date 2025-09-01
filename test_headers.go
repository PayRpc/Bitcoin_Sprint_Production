package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/headers"
)

func main() {
	fmt.Println("Testing Headers Service...")

	// Create mock nodes
	nodes := []headers.Node{
		headers.NewMockNode(800000),
		headers.NewMockNode(800001),
	}

	// Create FastRead
	fr := headers.NewFastRead(nodes, 250*time.Millisecond, 2)

	// Test getting a header
	ctx := context.Background()
	header, err := fr.GetHeader(ctx, 800000)
	if err != nil {
		fmt.Printf("Error getting header: %v\n", err)
		return
	}

	fmt.Printf("Got header: Height=%d, Hash=%x\n", header.Height, header.Hash)

	// Set an initial snapshot for testing
	headers.SetSnapshot(headers.Snapshot{
		BlockHash: header.Hash,
		Bytes:     header.Raw,
		Height:    header.Height,
	})

	// Start background updater with a context that doesn't cancel immediately
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// Don't start the updater for now - just test the endpoints
	// go headers.RunTipUpdater(ctx, fr, 800000, 1*time.Second)

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/headers/latest", headers.LatestHandler)
	mux.HandleFunc("/headers/stream", headers.StreamHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("Starting server on :8080")
	fmt.Println("Test endpoints:")
	fmt.Println("  GET http://localhost:8080/health")
	fmt.Println("  GET http://localhost:8080/headers/latest")
	fmt.Println("  GET http://localhost:8080/headers/stream")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
