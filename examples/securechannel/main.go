// Bitcoin Sprint SecureChannel Integration Example
// This example demonstrates professional integration of the SecureChannelPool
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bitcoin-sprint/pkg/secure"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize professional SecureChannel service
	config := &secure.ServiceConfig{
		RustPoolURL:      getEnv("RUST_POOL_URL", "http://127.0.0.1:9191"),
		CacheTimeout:     30 * time.Second,
		HealthTimeout:    5 * time.Second,
		MonitorInterval:  15 * time.Second,
		EnableMetrics:    true,
		MetricsNamespace: "bitcoin_sprint_secure_channel",
		LogLevel:         "info",
	}

	secureService, err := secure.NewService(config)
	if err != nil {
		log.Fatalf("Failed to create SecureChannel service: %v", err)
	}

	// Start the service
	ctx := context.Background()
	if err := secureService.Start(ctx); err != nil {
		log.Fatalf("Failed to start SecureChannel service: %v", err)
	}

	// Setup HTTP router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Add CORS middleware for professional API
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Register SecureChannel routes
	secureService.RegisterRoutes(router)

	// Add Bitcoin Sprint main status endpoint with SecureChannel integration
	router.GET("/status", func(c *gin.Context) {
		enhancedStatus, err := secureService.GetEnhancedStatus(c.Request.Context())
		if err != nil {
			log.Printf("Failed to get enhanced status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get system status",
			})
			return
		}
		c.JSON(http.StatusOK, enhancedStatus)
	})

	// Health check endpoint for Kubernetes/Docker
	router.GET("/health", func(c *gin.Context) {
		healthy, err := secureService.IsHealthy(c.Request.Context())
		if err != nil || !healthy {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  err,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// Readiness check for Kubernetes
	router.GET("/ready", func(c *gin.Context) {
		// Check if SecureChannel pool is responsive
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		_, err := secureService.GetHealthStatus(ctx)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})

	// API documentation endpoint
	router.GET("/api/docs", func(c *gin.Context) {
		docs := map[string]interface{}{
			"title":       "Bitcoin Sprint SecureChannel API",
			"version":     "1.0.0",
			"description": "Professional API for Bitcoin Sprint secure connection management",
			"endpoints": map[string]interface{}{
				"GET /status":                                "Complete system status including SecureChannel pool",
				"GET /health":                                "Health check endpoint",
				"GET /ready":                                 "Readiness check for Kubernetes",
				"GET /api/v1/secure-channel/status":          "Pool status with connection details",
				"GET /api/v1/secure-channel/health":          "Health status of the connection pool",
				"GET /api/v1/secure-channel/connections":     "List all connections",
				"GET /api/v1/secure-channel/connections/:id": "Get specific connection details",
				"GET /api/v1/secure-channel/stats":           "Aggregated connection statistics",
				"GET /api/v1/secure-channel/metrics":         "Prometheus metrics from pool",
				"GET /metrics":                               "Service-level Prometheus metrics",
			},
			"examples": map[string]interface{}{
				"pool_status": map[string]interface{}{
					"endpoint":            "relay.bitcoin-sprint.inc:443",
					"active_connections":  5,
					"total_reconnects":    12,
					"total_errors":        3,
					"pool_p95_latency_ms": 45,
				},
				"health_status": map[string]interface{}{
					"status":             "healthy",
					"pool_healthy":       true,
					"active_connections": 5,
				},
			},
		}
		c.JSON(http.StatusOK, docs)
	})

	// Start HTTP server
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Bitcoin Sprint SecureChannel API starting on port %s", port)
		log.Printf("API Documentation: http://localhost:%s/api/docs", port)
		log.Printf("Health Check: http://localhost:%s/health", port)
		log.Printf("System Status: http://localhost:%s/status", port)
		log.Printf("Metrics: http://localhost:%s/metrics", port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Bitcoin Sprint SecureChannel API...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Bitcoin Sprint SecureChannel API stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Example of using the SecureChannel client directly
func exampleDirectClientUsage() {
	// Create client for direct API access
	config := &secure.ClientConfig{
		BaseURL:       "http://127.0.0.1:9191",
		Timeout:       10 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		UserAgent:     "BitcoinSprint-Client/1.0.0",
	}

	client := secure.NewClient(config)
	ctx := context.Background()

	// Get pool status
	poolStatus, err := client.GetPoolStatus(ctx)
	if err != nil {
		log.Printf("Failed to get pool status: %v", err)
		return
	}

	log.Printf("Pool has %d active connections to %s",
		poolStatus.ActiveConnections, poolStatus.Endpoint)

	// Monitor pool health
	monitor := secure.NewMonitor(client, 30*time.Second)
	monitor.Start(ctx, func(pool *secure.PoolStatus, health *secure.HealthStatus, err error) {
		if err != nil {
			log.Printf("Monitor error: %v", err)
			return
		}

		if health != nil && health.Status == "healthy" {
			log.Printf("Pool is healthy with %d connections", health.ActiveConnections)
		}
	})

	// Wait for pool to be healthy
	if err := client.WaitForHealthy(ctx, 30*time.Second); err != nil {
		log.Printf("Pool did not become healthy: %v", err)
	} else {
		log.Println("Pool is now healthy!")
	}
}
