package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"bitcoin-sprint/internal/circuitbreaker"
)

// CircuitBreakerMonitor provides real-time monitoring of circuit breakers
type CircuitBreakerMonitor struct {
	breakers  map[string]*circuitbreaker.EnterpriseCircuitBreaker
	mu        sync.RWMutex
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	broadcast chan MonitorMessage
	stopChan  chan struct{}
}

// MonitorMessage represents a message sent to monitoring clients
type MonitorMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// CircuitBreakerStatus represents the current status of a circuit breaker
type CircuitBreakerStatus struct {
	Name            string                                `json:"name"`
	State           string                                `json:"state"`
	Metrics         *circuitbreaker.CircuitBreakerMetrics `json:"metrics"`
	Health          float64                               `json:"health"`
	LastStateChange time.Time                             `json:"last_state_change"`
	Configuration   CircuitBreakerConfig                  `json:"configuration"`
}

// CircuitBreakerConfig represents configuration summary
type CircuitBreakerConfig struct {
	MaxFailures      int           `json:"max_failures"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
	FailureThreshold float64       `json:"failure_threshold"`
	EnableAdaptive   bool          `json:"enable_adaptive"`
	EnableHealth     bool          `json:"enable_health"`
}

// AlertMessage represents an alert condition
type AlertMessage struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Breaker   string                 `json:"breaker"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func main() {
	var (
		port       = flag.String("port", "8090", "Monitor server port")
		configFile = flag.String("config", "", "Configuration file path")
		interval   = flag.Duration("interval", time.Second*5, "Monitoring interval")
	)
	flag.Parse()

	monitor := NewCircuitBreakerMonitor()

	// Load configuration if provided
	if *configFile != "" {
		if err := monitor.LoadConfiguration(*configFile); err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	}

	// Start monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	monitor.Start(ctx, *interval)

	// Setup HTTP server
	router := mux.NewRouter()

	// API endpoints
	router.HandleFunc("/api/breakers", monitor.handleGetBreakers).Methods("GET")
	router.HandleFunc("/api/breakers/{name}", monitor.handleGetBreaker).Methods("GET")
	router.HandleFunc("/api/breakers/{name}/metrics", monitor.handleGetMetrics).Methods("GET")
	router.HandleFunc("/api/breakers/{name}/state", monitor.handleSetState).Methods("POST")
	router.HandleFunc("/api/breakers/{name}/reset", monitor.handleReset).Methods("POST")
	router.HandleFunc("/api/alerts", monitor.handleGetAlerts).Methods("GET")

	// WebSocket endpoint for real-time updates
	router.HandleFunc("/ws", monitor.handleWebSocket)

	// Static file serving for web interface
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/monitor/")))

	server := &http.Server{
		Addr:         ":" + *port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Circuit Breaker Monitor starting on port %s", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down monitor...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	monitor.Stop()
	log.Println("Monitor stopped")
}

// NewCircuitBreakerMonitor creates a new monitor instance
func NewCircuitBreakerMonitor() *CircuitBreakerMonitor {
	return &CircuitBreakerMonitor{
		breakers: make(map[string]*circuitbreaker.EnterpriseCircuitBreaker),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan MonitorMessage, 100),
		stopChan:  make(chan struct{}),
	}
}

// RegisterBreaker registers a circuit breaker for monitoring
func (m *CircuitBreakerMonitor) RegisterBreaker(name string, breaker *circuitbreaker.EnterpriseCircuitBreaker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.breakers[name] = breaker
	log.Printf("Registered circuit breaker: %s", name)
}

// Start begins monitoring operations
func (m *CircuitBreakerMonitor) Start(ctx context.Context, interval time.Duration) {
	go m.monitoringLoop(ctx, interval)
	go m.broadcastLoop()
}

// Stop stops all monitoring operations
func (m *CircuitBreakerMonitor) Stop() {
	close(m.stopChan)

	// Close all WebSocket connections
	m.clientsMu.Lock()
	for client := range m.clients {
		client.Close()
	}
	m.clientsMu.Unlock()
}

// monitoringLoop continuously monitors circuit breakers
func (m *CircuitBreakerMonitor) monitoringLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectAndBroadcastStatus()

		case <-ctx.Done():
			return

		case <-m.stopChan:
			return
		}
	}
}

// collectAndBroadcastStatus collects status from all breakers and broadcasts updates
func (m *CircuitBreakerMonitor) collectAndBroadcastStatus() {
	m.mu.RLock()
	statuses := make(map[string]CircuitBreakerStatus)

	for name, breaker := range m.breakers {
		metrics := breaker.GetMetrics()

		status := CircuitBreakerStatus{
			Name:            name,
			State:           breaker.State().String(),
			Metrics:         metrics,
			Health:          metrics.HealthScore,
			LastStateChange: metrics.LastStateChange,
			Configuration: CircuitBreakerConfig{
				MaxFailures:      10, // This would come from breaker config
				ResetTimeout:     time.Minute,
				FailureThreshold: 0.5,
				EnableAdaptive:   true,
				EnableHealth:     true,
			},
		}

		statuses[name] = status

		// Check for alert conditions
		m.checkAlerts(name, status)
	}
	m.mu.RUnlock()

	// Broadcast status update
	message := MonitorMessage{
		Type:      "status_update",
		Timestamp: time.Now(),
		Data:      statuses,
	}

	select {
	case m.broadcast <- message:
	default:
		// Channel full, skip this update
	}
}

// checkAlerts checks for alert conditions and sends alerts
func (m *CircuitBreakerMonitor) checkAlerts(name string, status CircuitBreakerStatus) {
	// High failure rate alert
	if status.Metrics.FailureRate > 0.8 {
		alert := AlertMessage{
			Level:     "critical",
			Message:   fmt.Sprintf("High failure rate: %.2f%%", status.Metrics.FailureRate*100),
			Breaker:   name,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"failure_rate": status.Metrics.FailureRate,
				"state":        status.State,
			},
		}

		m.sendAlert(alert)
	}

	// Circuit open alert
	if status.State == "open" {
		alert := AlertMessage{
			Level:     "warning",
			Message:   "Circuit breaker is open",
			Breaker:   name,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"state":                status.State,
				"consecutive_failures": status.Metrics.ConsecutiveFailures,
			},
		}

		m.sendAlert(alert)
	}

	// Low health score alert
	if status.Health < 0.5 {
		alert := AlertMessage{
			Level:     "warning",
			Message:   fmt.Sprintf("Low health score: %.2f", status.Health),
			Breaker:   name,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"health_score": status.Health,
				"state":        status.State,
			},
		}

		m.sendAlert(alert)
	}
}

// sendAlert broadcasts an alert message
func (m *CircuitBreakerMonitor) sendAlert(alert AlertMessage) {
	message := MonitorMessage{
		Type:      "alert",
		Timestamp: time.Now(),
		Data:      alert,
	}

	select {
	case m.broadcast <- message:
	default:
		// Channel full, skip this alert
	}
}

// broadcastLoop handles broadcasting messages to WebSocket clients
func (m *CircuitBreakerMonitor) broadcastLoop() {
	for {
		select {
		case message := <-m.broadcast:
			m.clientsMu.RLock()
			for client := range m.clients {
				if err := client.WriteJSON(message); err != nil {
					client.Close()
					delete(m.clients, client)
				}
			}
			m.clientsMu.RUnlock()

		case <-m.stopChan:
			return
		}
	}
}

// HTTP Handlers

// handleGetBreakers returns information about all registered circuit breakers
func (m *CircuitBreakerMonitor) handleGetBreakers(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breakers := make(map[string]CircuitBreakerStatus)
	for name, breaker := range m.breakers {
		metrics := breaker.GetMetrics()

		breakers[name] = CircuitBreakerStatus{
			Name:            name,
			State:           breaker.State().String(),
			Metrics:         metrics,
			Health:          metrics.HealthScore,
			LastStateChange: metrics.LastStateChange,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(breakers)
}

// handleGetBreaker returns information about a specific circuit breaker
func (m *CircuitBreakerMonitor) handleGetBreaker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	m.mu.RLock()
	breaker, exists := m.breakers[name]
	m.mu.RUnlock()

	if !exists {
		http.Error(w, "Circuit breaker not found", http.StatusNotFound)
		return
	}

	metrics := breaker.GetMetrics()
	status := CircuitBreakerStatus{
		Name:            name,
		State:           breaker.State().String(),
		Metrics:         metrics,
		Health:          metrics.HealthScore,
		LastStateChange: metrics.LastStateChange,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleGetMetrics returns detailed metrics for a specific circuit breaker
func (m *CircuitBreakerMonitor) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	m.mu.RLock()
	breaker, exists := m.breakers[name]
	m.mu.RUnlock()

	if !exists {
		http.Error(w, "Circuit breaker not found", http.StatusNotFound)
		return
	}

	metrics := breaker.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleSetState sets the state of a circuit breaker
func (m *CircuitBreakerMonitor) handleSetState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var request struct {
		State string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	m.mu.RLock()
	breaker, exists := m.breakers[name]
	m.mu.RUnlock()

	if !exists {
		http.Error(w, "Circuit breaker not found", http.StatusNotFound)
		return
	}

	switch request.State {
	case "open":
		breaker.ForceOpen()
	case "closed":
		breaker.ForceClose()
	case "reset":
		breaker.Reset()
	default:
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleReset resets a circuit breaker
func (m *CircuitBreakerMonitor) handleReset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	m.mu.RLock()
	breaker, exists := m.breakers[name]
	m.mu.RUnlock()

	if !exists {
		http.Error(w, "Circuit breaker not found", http.StatusNotFound)
		return
	}

	breaker.Reset()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reset successful"})
}

// handleGetAlerts returns recent alerts
func (m *CircuitBreakerMonitor) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	// This would typically fetch from a persistent store
	// For now, return empty array
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]AlertMessage{})
}

// handleWebSocket handles WebSocket connections for real-time updates
func (m *CircuitBreakerMonitor) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Register client
	m.clientsMu.Lock()
	m.clients[conn] = true
	m.clientsMu.Unlock()

	// Remove client on disconnect
	defer func() {
		m.clientsMu.Lock()
		delete(m.clients, conn)
		m.clientsMu.Unlock()
	}()

	// Send initial status
	m.collectAndBroadcastStatus()

	// Keep connection alive
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// LoadConfiguration loads circuit breaker configurations from file
func (m *CircuitBreakerMonitor) LoadConfiguration(filename string) error {
	// Implementation would load and create circuit breakers from configuration
	// This is a placeholder for the actual implementation
	log.Printf("Loading configuration from %s", filename)
	return nil
}
