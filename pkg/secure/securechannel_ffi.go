//go:build cgo
// +build cgo

package secure

/*
#cgo LDFLAGS: -L. -lsecurebuffer
#include <stdlib.h>
#include <stdbool.h>

// SecureChannel FFI functions
bool secure_channel_init();
bool secure_channel_init_with_endpoint(const char* endpoint);
bool secure_channel_start();
bool secure_channel_stop();
bool secure_channel_is_running();
unsigned short secure_channel_get_metrics_port();
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// SecureChannelManager provides a Go interface to the Rust secure channel implementation
type SecureChannelManager struct {
	endpoint    string
	isRunning   bool
	metricsPort uint16
}

// NewSecureChannelManager creates a new secure channel manager
// Note: keep signature consistent with non-CGO build.
func NewSecureChannelManager() *SecureChannelManager {
	return &SecureChannelManager{
		endpoint:    "http://127.0.0.1:9191",
		isRunning:   false,
		metricsPort: 9191,
	}
}

// Initialize initializes the secure channel with default settings
func (scm *SecureChannelManager) Initialize() error {
	success := bool(C.secure_channel_init())
	if !success {
		return fmt.Errorf("failed to initialize secure channel")
	}
	return nil
}

// InitializeWithEndpoint initializes the secure channel with a specific endpoint
func (scm *SecureChannelManager) InitializeWithEndpoint(endpoint string) error {
	cEndpoint := C.CString(endpoint)
	defer C.free(unsafe.Pointer(cEndpoint))

	success := bool(C.secure_channel_init_with_endpoint(cEndpoint))
	if !success {
		return fmt.Errorf("failed to initialize secure channel with endpoint: %s", endpoint)
	}
	scm.endpoint = endpoint
	return nil
}

// Start starts the secure channel
func (scm *SecureChannelManager) Start() error {
	success := bool(C.secure_channel_start())
	if !success {
		return fmt.Errorf("failed to start secure channel")
	}
	scm.isRunning = true
	return nil
}

// Stop stops the secure channel
func (scm *SecureChannelManager) Stop() error {
	success := bool(C.secure_channel_stop())
	if !success {
		return fmt.Errorf("failed to stop secure channel")
	}
	scm.isRunning = false
	return nil
}

// IsRunning checks if the secure channel is currently running
func (scm *SecureChannelManager) IsRunning() bool {
	isRunning := bool(C.secure_channel_is_running())
	scm.isRunning = isRunning
	return isRunning
}

// GetMetricsPort returns the port where metrics are served
func (scm *SecureChannelManager) GetMetricsPort() uint16 {
	port := uint16(C.secure_channel_get_metrics_port())
	scm.metricsPort = port
	return port
}

// GetEndpoint returns the configured endpoint
func (scm *SecureChannelManager) GetEndpoint() string {
	return scm.endpoint
}

// GetMetricsURL returns the full URL for metrics
func (scm *SecureChannelManager) GetMetricsURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", scm.GetMetricsPort())
}

// GetStatus returns a comprehensive status of the secure channel
func (scm *SecureChannelManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":      scm.IsRunning(),
		"endpoint":     scm.endpoint,
		"metrics_port": scm.GetMetricsPort(),
		"metrics_url":  scm.GetMetricsURL(),
	}
}
