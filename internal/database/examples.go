package database

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Example usage of database integration
func (db *DB) ExampleUsage(ctx context.Context) error {
	// Example: Get API key
	apiKey, err := db.GetAPIKey(ctx, "some_key_hash")
	if err != nil {
		db.Logger.Info("API key not found", zap.Error(err))
	} else {
		db.Logger.Info("Found API key",
			zap.String("name", apiKey.Name),
			zap.String("tier", apiKey.Tier))
	}

	// Example: Log a request
	err = db.LogRequest(ctx, "api_key_id", "bitcoin", "GET", "/api/v1/blocks",
		1024, 2048, 150, 200, "192.168.1.1", "Mozilla/5.0")
	if err != nil {
		return fmt.Errorf("failed to log request: %w", err)
	}

	// Example: Get chain status
	chainStatus, err := db.GetChainStatus(ctx, "bitcoin")
	if err != nil {
		db.Logger.Info("Chain status not found", zap.Error(err))
	} else {
		db.Logger.Info("Bitcoin chain status",
			zap.Bool("synced", chainStatus["is_synced"].(bool)),
			zap.Int64("block_height", chainStatus["block_height"].(int64)))
	}

	// Example: Update chain status
	updates := map[string]interface{}{
		"block_height": 850000,
		"is_synced":    true,
		"peer_count":   25,
		"health_score": 98.5,
	}

	err = db.UpdateChainStatus(ctx, "bitcoin", updates)
	if err != nil {
		return fmt.Errorf("failed to update chain status: %w", err)
	}

	db.Logger.Info("Database operations completed successfully")
	return nil
}
