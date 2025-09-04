package messaging

import (
	"testing"
	"time"
)

func TestBitcoinRPCConfig_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  BitcoinRPCConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: BitcoinRPCConfig{
				URL:           "http://127.0.0.1:8332",
				Username:      "testuser",
				Password:      "testpass",
				Timeout:       30 * time.Second,
				MaxBlocks:     100,
				MaxTxPerBlock: 10000,
				MaxTxWorkers:  10,
				BatchSize:     50,
				RetryAttempts: 3,
				RetryMaxWait:  5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: BitcoinRPCConfig{
				Username: "testuser",
				Password: "testpass",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: BitcoinRPCConfig{
				URL:      "http://127.0.0.1:8332",
				Password: "testpass",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			config: BitcoinRPCConfig{
				URL:      "http://127.0.0.1:8332",
				Username: "testuser",
			},
			wantErr: true,
		},
		{
			name: "zero timeout gets default",
			config: BitcoinRPCConfig{
				URL:      "http://127.0.0.1:8332",
				Username: "testuser",
				Password: "testpass",
				Timeout:  0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("BitcoinRPCConfig.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check that defaults are set
			if tt.config.Timeout == 0 && !tt.wantErr {
				t.Errorf("Timeout should be set to default when zero")
			}
		})
	}
}

func TestGetTxIDs(t *testing.T) {
	tests := []struct {
		name     string
		txs      []interface{}
		maxTx    int
		expected int
	}{
		{
			name:     "empty transactions",
			txs:      []interface{}{},
			maxTx:    100,
			expected: 0,
		},
		{
			name:     "string transaction IDs",
			txs:      []interface{}{"tx1", "tx2", "tx3"},
			maxTx:    100,
			expected: 3,
		},
		{
			name: "map transaction objects",
			txs: []interface{}{
				map[string]interface{}{"txid": "tx1"},
				map[string]interface{}{"txid": "tx2"},
			},
			maxTx:    100,
			expected: 2,
		},
		{
			name:     "mixed types",
			txs:      []interface{}{"tx1", map[string]interface{}{"txid": "tx2"}},
			maxTx:    100,
			expected: 2,
		},
		{
			name:     "respect max limit",
			txs:      []interface{}{"tx1", "tx2", "tx3", "tx4", "tx5"},
			maxTx:    3,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTxIDs(tt.txs, tt.maxTx)
			if len(result) != tt.expected {
				t.Errorf("getTxIDs() = %v, expected %d items, got %d", result, tt.expected, len(result))
			}
		})
	}
}

func TestLoadFailedTxs(t *testing.T) {
	// Test with non-existent file
	result := loadFailedTxs("nonexistent.txt")
	if len(result) != 0 {
		t.Errorf("loadFailedTxs() with nonexistent file should return empty slice, got %v", result)
	}
}

func TestLoadLastID(t *testing.T) {
	// Test with non-existent file
	result := loadLastID("nonexistent.txt")
	if result != "" {
		t.Errorf("loadLastID() with nonexistent file should return empty string, got %q", result)
	}
}
