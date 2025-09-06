//go:build cgo
// +build cgo

package securechan

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewChannel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	tests := []struct {
		name        string
		endpoint    string
		config      *ChannelConfig
		expectError bool
		errorType   string
	}{
		{
			name:        "valid endpoint with default config",
			endpoint:    "tcp://localhost:8080",
			config:      nil,
			expectError: false,
		},
		{
			name:        "valid endpoint with custom config",
			endpoint:    "tcp://localhost:8080",
			config:      DefaultChannelConfig(),
			expectError: false,
		},
		{
			name:        "empty endpoint",
			endpoint:    "",
			config:      DefaultChannelConfig(),
			expectError: true,
			errorType:   "*securechan.SecureChannelError",
		},
		{
			name:        "valid endpoint with nil logger",
			endpoint:    "tcp://localhost:8080",
			config:      DefaultChannelConfig(),
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := NewChannel(tt.endpoint, tt.config, logger)
			
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, channel)
				if tt.errorType != "" {
					assert.IsType(t, &SecureChannelError{}, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, channel)
				assert.Equal(t, tt.endpoint, channel.GetEndpoint())
				assert.Equal(t, StateDisconnected, channel.GetState())
				assert.False(t, channel.IsConnected())
				
				// Cleanup
				err = channel.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestChannelState(t *testing.T) {
	logger := zaptest.NewLogger(t)
	channel, err := NewChannel("tcp://localhost:8080", DefaultChannelConfig(), logger)
	require.NoError(t, err)
	defer func() {
		err := channel.Close()
		assert.NoError(t, err)
	}()
	
	// Test initial state
	assert.Equal(t, StateDisconnected, channel.GetState())
	assert.False(t, channel.IsConnected())
	
	// Test state string representation
	states := []ChannelState{
		StateDisconnected,
		StateConnecting,
		StateConnected,
		StateError,
		StateStopping,
	}
	
	expectedStrings := []string{
		"disconnected",
		"connecting",
		"connected",
		"error",
		"stopping",
	}
	
	for i, state := range states {
		assert.Equal(t, expectedStrings[i], state.String())
	}
	
	// Test unknown state
	unknownState := ChannelState(999)
	assert.Equal(t, "unknown", unknownState.String())
}

func TestChannelMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	channel, err := NewChannel("tcp://localhost:8080", DefaultChannelConfig(), logger)
	require.NoError(t, err)
	defer func() {
		err := channel.Close()
		assert.NoError(t, err)
	}()
	
	// Test initial metrics
	metrics := channel.GetMetrics()
	assert.Equal(t, int64(0), metrics.ConnectionAttempts)
	assert.Equal(t, int64(0), metrics.SuccessfulConnects)
	assert.Equal(t, int64(0), metrics.FailedConnects)
	assert.Equal(t, int64(0), metrics.BytesSent)
	assert.Equal(t, int64(0), metrics.BytesReceived)
	assert.Equal(t, int64(0), metrics.ErrorCount)
	assert.Equal(t, time.Duration(0), metrics.TotalUptime)
}

func TestChannelConfiguration(t *testing.T) {
	config := DefaultChannelConfig()
	
	// Test default configuration values
	assert.Equal(t, 30*time.Second, config.ConnectionTimeout)
	assert.Equal(t, 10*time.Second, config.ReadTimeout)
	assert.Equal(t, 10*time.Second, config.WriteTimeout)
	assert.Equal(t, 60*time.Second, config.KeepAliveInterval)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.RetryDelay)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
	assert.Equal(t, 5*time.Second, config.MaxRetryDelay)
	assert.Equal(t, 8192, config.SendBufferSize)
	assert.Equal(t, 8192, config.ReceiveBufferSize)
	assert.Equal(t, 1024*1024, config.MaxMessageSize)
	assert.True(t, config.EnableEncryption)
	assert.True(t, config.EnableMetrics)
	assert.Equal(t, 30*time.Second, config.MetricsInterval)
}

func TestSecureChannelError(t *testing.T) {
	baseErr := assert.AnError
	secErr := &SecureChannelError{
		Operation: "test_operation",
		Endpoint:  "tcp://localhost:8080",
		Err:       baseErr,
	}
	
	expectedMsg := "secure channel test_operation failed for endpoint tcp://localhost:8080: assert.AnError general error for testing"
	assert.Equal(t, expectedMsg, secErr.Error())
}

func TestChannelSendReceiveValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultChannelConfig()
	config.MaxMessageSize = 100 // Small limit for testing
	
	channel, err := NewChannel("tcp://localhost:8080", config, logger)
	require.NoError(t, err)
	defer func() {
		err := channel.Close()
		assert.NoError(t, err)
	}()
	
	ctx := context.Background()
	
	// Test send when not connected
	data := []byte("test data")
	_, err = channel.Send(ctx, data)
	require.Error(t, err)
	assert.IsType(t, &SecureChannelError{}, err)
	
	// Test receive when not connected
	buffer := make([]byte, 1024)
	_, err = channel.Receive(ctx, buffer)
	require.Error(t, err)
	assert.IsType(t, &SecureChannelError{}, err)
	
	// Test send with empty data
	channel.state = StateConnected // Simulate connected state for validation
	n, err := channel.Send(ctx, []byte{})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	
	// Test send with oversized data
	bigData := make([]byte, config.MaxMessageSize+1)
	_, err = channel.Send(ctx, bigData)
	require.Error(t, err)
	assert.IsType(t, &SecureChannelError{}, err)
	
	// Test receive with empty buffer
	_, err = channel.Receive(ctx, []byte{})
	require.Error(t, err)
}

func TestChannelContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	channel, err := NewChannel("tcp://localhost:8080", DefaultChannelConfig(), logger)
	require.NoError(t, err)
	defer func() {
		err := channel.Close()
		assert.NoError(t, err)
	}()
	
	// Test start with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	err = channel.Start(ctx)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestChannelClose(t *testing.T) {
	logger := zaptest.NewLogger(t)
	channel, err := NewChannel("tcp://localhost:8080", DefaultChannelConfig(), logger)
	require.NoError(t, err)
	
	// Test close
	err = channel.Close()
	assert.NoError(t, err)
	
	// Test double close
	err = channel.Close()
	assert.NoError(t, err)
}

func TestChannelRetryLogic(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultChannelConfig()
	config.MaxRetries = 2
	config.RetryDelay = 10 * time.Millisecond
	config.BackoffMultiplier = 2.0
	config.MaxRetryDelay = 100 * time.Millisecond
	
	channel, err := NewChannel("tcp://invalid:99999", config, logger)
	require.NoError(t, err)
	defer func() {
		err := channel.Close()
		assert.NoError(t, err)
	}()
	
	// Calculate expected retry delays
	delays := []time.Duration{
		10 * time.Millisecond,  // First retry
		20 * time.Millisecond,  // Second retry (10ms * 2)
	}
	
	for i, expectedDelay := range delays {
		calculatedDelay := channel.calculateRetryDelay(i)
		assert.Equal(t, expectedDelay, calculatedDelay, "Retry delay mismatch for attempt %d", i)
	}
	
	// Test max delay cap
	longDelay := channel.calculateRetryDelay(10) // Should hit max delay
	assert.Equal(t, config.MaxRetryDelay, longDelay)
}

func BenchmarkChannelCreation(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := DefaultChannelConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		channel, err := NewChannel("tcp://localhost:8080", config, logger)
		if err != nil {
			b.Fatal(err)
		}
		err = channel.Close()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChannelMetrics(b *testing.B) {
	logger := zaptest.NewLogger(b)
	channel, err := NewChannel("tcp://localhost:8080", DefaultChannelConfig(), logger)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		err := channel.Close()
		if err != nil {
			b.Fatal(err)
		}
	}()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channel.GetMetrics()
	}
}
