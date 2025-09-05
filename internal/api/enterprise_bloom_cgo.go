//go:build cgo
// +build cgo

package api

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/PayRpc/Bitcoin-Sprint/internal/securebuf"
    "go.uber.org/zap"
)

// BloomFilterRequest represents a request to create a Bitcoin bloom filter
type BloomFilterRequest struct {
    NumBits       int    `json:"num_bits"`
    HashFunctions int    `json:"hash_functions"`
    Tweak         uint32 `json:"tweak"`
    Flags         uint8  `json:"flags"`
    MaxAge        int    `json:"max_age"`
    BatchSize     int    `json:"batch_size"`
}

// BloomFilterResponse represents the response for bloom filter operations
type BloomFilterResponse struct {
    FilterID  string    `json:"filter_id"`
    NumBits   int       `json:"num_bits"`
    HashFns   int       `json:"hash_functions"`
    Timestamp time.Time `json:"timestamp"`
}

// handleNewBloomFilter creates a new Bitcoin bloom filter (cgo build only)
func (esm *EnterpriseSecurityManager) handleNewBloomFilter(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        esm.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
        return
    }

    var req BloomFilterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        esm.jsonError(w, http.StatusBadRequest, "Invalid request body")
        return
    }

    // Set defaults if not provided
    if req.NumBits == 0 {
        req.NumBits = 1000000
    }
    if req.HashFunctions == 0 {
        req.HashFunctions = 7
    }
    if req.MaxAge == 0 {
        req.MaxAge = 3600
    }
    if req.BatchSize == 0 {
        req.BatchSize = 1000
    }

    // Create bloom filter via FFI
    filter, err := securebuf.NewBitcoinBloomFilter(
        uint64(req.NumBits), uint8(req.HashFunctions), req.Tweak,
        req.Flags, uint64(req.MaxAge), uint64(req.BatchSize),
    )
    if err != nil {
        esm.logger.Error("Failed to create Bitcoin bloom filter", zap.Error(err))
        esm.jsonError(w, http.StatusInternalServerError, "Failed to create bloom filter")
        return
    }
    defer filter.Free()

    response := BloomFilterResponse{
        FilterID:  fmt.Sprintf("bloom_%d", time.Now().UnixNano()),
        NumBits:   req.NumBits,
        HashFns:   req.HashFunctions,
        Timestamp: time.Now(),
    }

    esm.jsonResponse(w, http.StatusOK, response)
}
