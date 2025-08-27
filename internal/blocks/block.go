package blocks

import "time"

type BlockEvent struct {
	Hash        string    `json:"hash"`
	Height      uint32    `json:"height"`
	Timestamp   time.Time `json:"timestamp"`
	DetectedAt  time.Time `json:"detected_at"`
	RelayTimeMs float64   `json:"relay_time_ms"`
	Source      string    `json:"source"`
	TxID        string    `json:"txid,omitempty"`
	Tier        string    `json:"tier"`
	IsHeader    bool      `json:"is_header,omitempty"`
}
