//go:build !sprintd_min
// +build !sprintd_min

package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/engine"
)

// CustomRPCTask demonstrates how to create custom RPC tasks using the engine
type CustomRPCTask struct {
	id          string
	rpcURL      string
	rpcUser     string
	rpcPass     string
	customQuery string
}

// NewCustomRPCTask creates a new custom RPC task
func NewCustomRPCTask(id, rpcURL, rpcUser, rpcPass, customQuery string) *CustomRPCTask {
	return &CustomRPCTask{
		id:          id,
		rpcURL:      rpcURL,
		rpcUser:     rpcUser,
		rpcPass:     rpcPass,
		customQuery: customQuery,
	}
}

func (t *CustomRPCTask) ID() string { return t.id }

// Execute performs a custom RPC operation
func (t *CustomRPCTask) Execute(ctx context.Context, helpers engine.EngineHelpers) ([]engine.Message, error) {
	// This is a simplified example - in practice you'd make actual RPC calls
	messages := []engine.Message{
		{
			ID:        fmt.Sprintf("%s-result-%d", t.id, time.Now().Unix()),
			Topic:     "custom_rpc",
			Data:      map[string]interface{}{"query": t.customQuery, "result": "success"},
			Timestamp: time.Now().UTC(),
		},
	}

	// Update metrics
	helpers.Metrics.messagesProduced.Inc()

	return messages, nil
}

// BatchRPCTask demonstrates batch RPC operations
type BatchRPCTask struct {
	id       string
	rpcURL   string
	rpcUser  string
	rpcPass  string
	queries  []string
}

// NewBatchRPCTask creates a new batch RPC task
func NewBatchRPCTask(id, rpcURL, rpcUser, rpcPass string, queries []string) *BatchRPCTask {
	return &BatchRPCTask{
		id:      id,
		rpcURL:  rpcURL,
		rpcUser: rpcUser,
		rpcPass: rpcPass,
		queries: queries,
	}
}

func (t *BatchRPCTask) ID() string { return t.id }

// Execute performs batch RPC operations
func (t *BatchRPCTask) Execute(ctx context.Context, helpers engine.EngineHelpers) ([]engine.Message, error) {
	messages := make([]engine.Message, 0, len(t.queries))

	for i, query := range t.queries {
		// Check if we've seen this query before
		queryID := fmt.Sprintf("%s-query-%d", t.id, i)
		if has, _ := helpers.Seen.Has(ctx, queryID); has {
			continue // Skip duplicate
		}

		msg := engine.Message{
			ID:        queryID,
			Topic:     "batch_rpc",
			Data:      map[string]interface{}{"query": query, "batch_id": t.id},
			Timestamp: time.Now().UTC(),
		}

		messages = append(messages, msg)
		helpers.Cache.Add(queryID, msg)
		_ = helpers.Seen.Add(ctx, queryID)
		helpers.Metrics.messagesProduced.Inc()
	}

	return messages, nil
}
