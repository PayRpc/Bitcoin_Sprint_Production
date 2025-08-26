package blocks

import "time"

type BlockEvent struct {
	Hash      string    `json:"hash"`
	Height    uint32    `json:"height"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	TxID      string    `json:"txid,omitempty"`
}
