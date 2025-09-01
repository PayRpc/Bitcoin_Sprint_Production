package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// DB represents the database connection
type DB struct {
	Pool   *pgxpool.Pool
	Logger *zap.Logger
}

// Config holds database configuration
type Config struct {
	Type     string
	URL      string
	MaxConns int
	MinConns int
}

// New creates a new database connection
func New(cfg Config, logger *zap.Logger) (*DB, error) {
	if cfg.Type != "postgres" && cfg.Type != "postgresql" {
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.HealthCheckPeriod = 1 * time.Minute
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("type", cfg.Type),
		zap.String("url", cfg.URL),
		zap.Int("max_conns", cfg.MaxConns),
		zap.Int("min_conns", cfg.MinConns))

	return &DB{
		Pool:   pool,
		Logger: logger,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		db.Logger.Info("Database connection closed")
	}
}

// Ping tests the database connection
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// APIKey represents an API key in the database
type APIKey struct {
	ID          string    `json:"id"`
	KeyHash     string    `json:"key_hash"`
	Name        string    `json:"name"`
	Tier        string    `json:"tier"`
	RateLimit   int       `json:"rate_limit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool      `json:"is_active"`
}

// GetAPIKey retrieves an API key by hash
func (db *DB) GetAPIKey(ctx context.Context, keyHash string) (*APIKey, error) {
	query := `
		SELECT id, key_hash, name, tier, rate_limit, created_at, updated_at, expires_at, is_active
		FROM sprint_core.api_keys
		WHERE key_hash = $1 AND is_active = true
		AND (expires_at IS NULL OR expires_at > NOW())`

	var key APIKey
	var expiresAt sql.NullTime

	err := db.Pool.QueryRow(ctx, query, keyHash).Scan(
		&key.ID, &key.KeyHash, &key.Name, &key.Tier, &key.RateLimit,
		&key.CreatedAt, &key.UpdatedAt, &expiresAt, &key.IsActive)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}

	return &key, nil
}

// LogRequest logs an API request
func (db *DB) LogRequest(ctx context.Context, apiKeyID, chain, method, endpoint string, requestSize, responseSize, responseTimeMs, statusCode int, ipAddress, userAgent string) error {
	query := `
		INSERT INTO sprint_core.request_logs (
			api_key_id, chain, method, endpoint, request_size, response_size,
			response_time_ms, status_code, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`

	_, err := db.Pool.Exec(ctx, query,
		apiKeyID, chain, method, endpoint, requestSize, responseSize,
		responseTimeMs, statusCode, ipAddress, userAgent)

	if err != nil {
		db.Logger.Error("Failed to log request", zap.Error(err))
		return fmt.Errorf("failed to log request: %w", err)
	}

	return nil
}

// GetChainStatus retrieves chain status
func (db *DB) GetChainStatus(ctx context.Context, chainName string) (map[string]interface{}, error) {
	query := `
		SELECT chain_name, rpc_endpoint, websocket_endpoint, block_height,
			   is_synced, last_block_time, avg_block_time, peer_count, health_score, last_updated
		FROM sprint_chains.chain_status
		WHERE chain_name = $1`

	var chainNameResult, rpcEndpoint, wsEndpoint sql.NullString
	var blockHeight sql.NullInt64
	var isSynced sql.NullBool
	var lastBlockTime, lastUpdated pq.NullTime
	var avgBlockTime sql.NullFloat64
	var peerCount sql.NullInt32
	var healthScore sql.NullFloat64

	err := db.Pool.QueryRow(ctx, query, chainName).Scan(
		&chainNameResult, &rpcEndpoint, &wsEndpoint, &blockHeight,
		&isSynced, &lastBlockTime, &lastUpdated, &avgBlockTime,
		&peerCount, &healthScore)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("chain not found: %s", chainName)
		}
		return nil, fmt.Errorf("failed to get chain status: %w", err)
	}

	result := map[string]interface{}{
		"chain_name":        chainNameResult.String,
		"rpc_endpoint":      rpcEndpoint.String,
		"websocket_endpoint": wsEndpoint.String,
		"block_height":      blockHeight.Int64,
		"is_synced":         isSynced.Bool,
		"avg_block_time":    avgBlockTime.Float64,
		"peer_count":        peerCount.Int32,
		"health_score":      healthScore.Float64,
	}

	if lastBlockTime.Valid {
		result["last_block_time"] = lastBlockTime.Time
	}
	if lastUpdated.Valid {
		result["last_updated"] = lastUpdated.Time
	}

	return result, nil
}

// UpdateChainStatus updates chain status
func (db *DB) UpdateChainStatus(ctx context.Context, chainName string, updates map[string]interface{}) error {
	setParts := []string{}
	args := []interface{}{}
	argCount := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argCount))
		args = append(args, value)
		argCount++
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE sprint_chains.chain_status
		SET %s, last_updated = NOW()
		WHERE chain_name = $%d`,
		fmt.Sprintf("%s", setParts[0]), argCount)

	args = append(args, chainName)

	_, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update chain status: %w", err)
	}

	return nil
}
