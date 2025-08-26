//go:build !cgo
// +build !cgo

package secure

// SecureChannelManager stub for non-CGO builds
type SecureChannelManager struct {
	enabled   bool
	isRunning bool
}

// NewSecureChannelManager creates a new SecureChannelManager (stub for non-CGO builds)
func NewSecureChannelManager() *SecureChannelManager {
	return &SecureChannelManager{
		enabled:   true, // Always enabled in Go mode
		isRunning: true, // Always considered running
	}
}

// Initialize initializes the secure channel (stub for non-CGO builds)
func (scm *SecureChannelManager) Initialize() error {
	scm.enabled = true
	scm.isRunning = true
	return nil // Always succeed in Go-native mode
}

// InitializeWithEndpoint initializes with custom endpoint (stub for non-CGO builds)
func (scm *SecureChannelManager) InitializeWithEndpoint(endpoint string) error {
	scm.enabled = true
	scm.isRunning = true
	return nil // Always succeed in Go-native mode
}

// Start starts the secure channel (stub for non-CGO builds)
func (scm *SecureChannelManager) Start() error {
	scm.isRunning = true
	return nil // Always succeed in Go-native mode
}

// Stop stops the secure channel (stub for non-CGO builds)
func (scm *SecureChannelManager) Stop() error {
	scm.isRunning = false
	return nil
}

// IsRunning returns whether the secure channel is running (stub for non-CGO builds)
func (scm *SecureChannelManager) IsRunning() bool {
	return scm.isRunning
}

// GetStatus returns the secure channel status (stub for non-CGO builds)
func (scm *SecureChannelManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":      scm.isRunning,
		"backend":      "Go-native",
		"mode":         "compatibility",
		"available":    true,
		"metrics_port": 9191,
		"endpoint":     "http://127.0.0.1:9191",
	}
}

// GetMetricsPort returns the metrics server port (stub for non-CGO builds)
func (scm *SecureChannelManager) GetMetricsPort() uint16 {
	return 9191 // Return unified default port in Go mode
}
