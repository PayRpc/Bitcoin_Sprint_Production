//go:build !sprintd_min
// +build !sprintd_min

package rpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/engine"
	"go.uber.org/zap"
)

// EnhancedRPCService provides high-performance RPC operations using the processing engine
type EnhancedRPCService struct {
	cfg      config.Config
	logger   *zap.Logger
	engine   *engine.Engine
	state    engine.StateStore
	seen     engine.SeenStore
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
	metrics  *RPCServiceMetrics
}

// RPCServiceMetrics tracks RPC service performance
type RPCServiceMetrics struct {
	TasksSubmitted    int64
	TasksCompleted    int64
	MessagesProcessed int64
	Errors            int64
	LastActivity      time.Time
	mu                sync.RWMutex
}

// NewEnhancedRPCService creates a new enhanced RPC service
func NewEnhancedRPCService(cfg config.Config, logger *zap.Logger) (*EnhancedRPCService, error) {
	// Create persistent stores
	state := engine.NewFileStateStore("data/rpc_state.txt", "data/rpc_failed.ndjson")
	seen := engine.NewFileSeenStore("data/rpc_seen.ndjson", 100000)

	// Create engine with optimized settings
	eng, err := engine.NewEngine(
		cfg.RPCBatchSize, // workers
		4096,             // queue size
		8192,             // cache size
		state,
		seen,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	return &EnhancedRPCService{
		cfg:      cfg,
		logger:   logger,
		engine:   eng,
		state:    state,
		seen:     seen,
		stopChan: make(chan struct{}),
		metrics:  &RPCServiceMetrics{},
	}, nil
}

// Start begins the enhanced RPC service
func (s *EnhancedRPCService) Start(ctx context.Context) error {
	if !s.cfg.RPCEnabled {
		s.logger.Info("Enhanced RPC service disabled")
		return nil
	}

	s.running = true
	s.logger.Info("Starting enhanced RPC service",
		zap.String("rpc_url", s.cfg.RPCURL),
		zap.Int("workers", s.cfg.RPCBatchSize),
		zap.Duration("timeout", s.cfg.RPCTimeout))

	// Start engine with metrics server
	if err := s.engine.Start(true, ":9091"); err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}

	// Start background task submission
	s.wg.Add(1)
	go s.taskScheduler(ctx)

	s.logger.Info("Enhanced RPC service started successfully")
	return nil
}

// Stop gracefully shuts down the RPC service
func (s *EnhancedRPCService) Stop() {
	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)

	// Stop engine
	s.engine.Stop()

	s.wg.Wait()
	s.logger.Info("Enhanced RPC service stopped")
}

// SubmitBackfillTask submits a new backfill task to the engine
func (s *EnhancedRPCService) SubmitBackfillTask(taskID string, maxBlocks int) error {
	if !s.running {
		return fmt.Errorf("RPC service not running")
	}

	cfg := engine.BitcoinRPCConfig{
		URL:           s.cfg.RPCURL,
		Username:      s.cfg.RPCUsername,
		Password:      s.cfg.RPCPassword,
		Timeout:       s.cfg.RPCTimeout,
		MaxBlocks:     maxBlocks,
		MaxTxPerBlock: 1000, // Configurable
		BatchSize:     50,
		Topic:         "bitcoin_backfill",
		RetryAttempts: s.cfg.RPCRetryAttempts,
		RetryMaxWait:  s.cfg.RPCRetryMaxWait,
		SkipMempool:   s.cfg.RPCSkipMempool,
	}

	task := engine.NewBitcoinTask(taskID, cfg, s.state, s.seen)

	if err := s.engine.Submit(task); err != nil {
		s.metrics.mu.Lock()
		s.metrics.Errors++
		s.metrics.mu.Unlock()
		return fmt.Errorf("failed to submit task: %w", err)
	}

	s.metrics.mu.Lock()
	s.metrics.TasksSubmitted++
	s.metrics.LastActivity = time.Now()
	s.metrics.mu.Unlock()

	s.logger.Info("Backfill task submitted",
		zap.String("task_id", taskID),
		zap.Int("max_blocks", maxBlocks))

	return nil
}

// taskScheduler periodically submits maintenance tasks
func (s *EnhancedRPCService) taskScheduler(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			// Submit periodic health check task
			if err := s.SubmitBackfillTask(fmt.Sprintf("health-%d", time.Now().Unix()), 1); err != nil {
				s.logger.Error("Failed to submit health check task", zap.Error(err))
			}
		}
	}
}

// GetMetrics returns current service metrics
func (s *EnhancedRPCService) GetMetrics() RPCServiceMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()
	return *s.metrics
}

// UpdateMetrics updates metrics from engine results
func (s *EnhancedRPCService) UpdateMetrics(tasksCompleted, messagesProcessed int64) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()
	s.metrics.TasksCompleted += tasksCompleted
	s.metrics.MessagesProcessed += messagesProcessed
	s.metrics.LastActivity = time.Now()
}
