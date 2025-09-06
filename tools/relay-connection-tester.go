// Package main provides a blockchain relay connection benchmarking tool
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/relay"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Parse command-line flags
	testDuration := flag.Int("duration", 30, "Test duration in seconds")
	testEth := flag.Bool("eth", true, "Test Ethereum connections")
	testSol := flag.Bool("sol", true, "Test Solana connections")
	verbose := flag.Bool("v", false, "Verbose logging")
	flag.Parse()

	// Configure logging
	logConfig := zap.NewProductionConfig()
	logConfig.OutputPaths = []string{"stdout"}
	logConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	if *verbose {
		logConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}
	logger, _ := logConfig.Build()
	defer logger.Sync()

	// Create configuration
	cfg := config.NewDefaultConfig()

	// Create context with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*testDuration)*time.Second)
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Track connection status
	var wg sync.WaitGroup
	statusChan := make(chan string, 100)

	go func() {
		for status := range statusChan {
			fmt.Println(status)
		}
	}()

	// Test Ethereum connections if enabled
	if *testEth {
		wg.Add(1)
		go testEthereumConnections(ctx, cfg, logger, statusChan, &wg)
	}

	// Test Solana connections if enabled
	if *testSol {
		wg.Add(1)
		go testSolanaConnections(ctx, cfg, logger, statusChan, &wg)
	}

	// Wait for tests to complete
	fmt.Printf("Running connection tests for %d seconds...\n", *testDuration)
	fmt.Println("Press Ctrl+C to stop early")

	// Wait for completion
	wg.Wait()
	close(statusChan)

	fmt.Println("Connection tests completed")
}

func testEthereumConnections(ctx context.Context, cfg config.Config, logger *zap.Logger,
	statusChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	ethRelay := relay.NewEthereumRelay(cfg, logger)

	// Connect to Ethereum network
	statusChan <- "Connecting to Ethereum network..."
	if err := ethRelay.Connect(ctx); err != nil {
		statusChan <- fmt.Sprintf("ERROR: Failed to start Ethereum connection: %v", err)
		return
	}

	// Check connection status periodically
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			statusChan <- "Ethereum test finished, disconnecting..."
			ethRelay.Disconnect()
			return
		case <-ticker.C:
			// Check connection status
			isConnected := ethRelay.IsConnected()
			statusChan <- fmt.Sprintf("[ETH] Connected: %v", isConnected)

			// Try to get network info
			if isConnected {
				if info, err := ethRelay.GetNetworkInfo(); err == nil {
					statusChan <- fmt.Sprintf("[ETH] Network: %s, Block Height: %d, Peers: %d",
						info.Network, info.BlockHeight, info.PeerCount)
				}

				// Try to get health status
				if health, err := ethRelay.GetHealth(); err == nil {
					statusChan <- fmt.Sprintf("[ETH] Health: %v, State: %s",
						health.IsHealthy, health.ConnectionState)
				}
			}
		}
	}
}

func testSolanaConnections(ctx context.Context, cfg config.Config, logger *zap.Logger,
	statusChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()

	solRelay := relay.NewSolanaRelay(cfg, logger)

	// Connect to Solana network
	statusChan <- "Connecting to Solana network..."
	if err := solRelay.Connect(ctx); err != nil {
		statusChan <- fmt.Sprintf("ERROR: Failed to start Solana connection: %v", err)
		return
	}

	// Check connection status periodically
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			statusChan <- "Solana test finished, disconnecting..."
			solRelay.Disconnect()
			return
		case <-ticker.C:
			// Check connection status
			isConnected := solRelay.IsConnected()
			statusChan <- fmt.Sprintf("[SOL] Connected: %v", isConnected)

			// Try to get network info
			if isConnected {
				if info, err := solRelay.GetNetworkInfo(); err == nil {
					statusChan <- fmt.Sprintf("[SOL] Network: %s, Block Height: %d",
						info.Network, info.BlockHeight)
				}

				// Try to get health status
				if health, err := solRelay.GetHealth(); err == nil {
					statusChan <- fmt.Sprintf("[SOL] Health: %v, State: %s",
						health.IsHealthy, health.ConnectionState)
				}
			}
		}
	}
}
