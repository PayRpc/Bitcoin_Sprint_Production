package performance

import (
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/PayRpc/Bitcoin-Sprint/internal/config"
	"github.com/PayRpc/Bitcoin-Sprint/internal/entropy"
	"go.uber.org/zap"
)

// OptimizationLevel represents different performance optimization levels
type OptimizationLevel int

const (
	LevelStandard OptimizationLevel = iota // Standard performance
	LevelHigh                              // High performance (default)
	LevelMaximum                           // Maximum performance (99.9% SLA)
)

// PerformanceManager handles system-level performance optimizations
type PerformanceManager struct {
	cfg    config.Config
	logger *zap.Logger
	level  OptimizationLevel

	// Enhanced buffer pool for memory management
	bufferPool *BufferPool

	// Pipeline and worker management for flat latency
	workerPool    *WorkerPool
	backpressure  *BackpressureController
	pipelineStats *PipelineStats
}

// WorkerPool manages a pool of workers with 2×NumCPU for optimal latency
type WorkerPool struct {
	numWorkers int
	workers    []*Worker
	taskChan   chan Task
	quitChan   chan struct{}
	wg         sync.WaitGroup
}

// Worker represents a single worker in the pool
type Worker struct {
	id         int
	taskChan   <-chan Task
	quitChan   <-chan struct{}
	processing bool
	lastTask   time.Time
}

// Task represents a unit of work for the worker pool
type Task struct {
	ID       string
	Payload  interface{}
	Priority int
	Deadline time.Time
	Handler  func(interface{}) error
}

// BackpressureController manages backpressure to prevent latency spikes
type BackpressureController struct {
	mu                 sync.RWMutex
	queueDepth         int
	maxQueueDepth      int
	backpressureLevel  int
	backpressureEvents int64
	lastAdjustment     time.Time
	samplingInterval   time.Duration
}

// PipelineStats tracks pipeline performance metrics for flat latency monitoring
type PipelineStats struct {
	mu                 sync.RWMutex
	workerUtilization  []float64
	queueDepth         int
	processingLatency  []time.Duration
	backpressureEvents int64
	lastUpdate         time.Time
}

// BufferPool manages reusable memory buffers
type BufferPool struct {
	pools   map[int]*sync.Pool
	mu      sync.RWMutex
	maxSize int
}

// New creates a new performance manager with automatic optimization level detection
func New(cfg config.Config, logger *zap.Logger) *PerformanceManager {
	level := LevelHigh // Default to high performance

	// Determine optimization level based on tier
	switch cfg.Tier {
	case config.TierTurbo, config.TierEnterprise:
		level = LevelMaximum // 99.9% SLA compliance
	case config.TierPro, config.TierBusiness:
		level = LevelHigh // High performance
	default:
		level = LevelStandard // Standard performance
	}

	return &PerformanceManager{
		cfg:           cfg,
		logger:        logger,
		level:         level,
		bufferPool:    NewBufferPool(),
		workerPool:    NewWorkerPool(),
		backpressure:  NewBackpressureController(),
		pipelineStats: NewPipelineStats(),
	}
}

// NewBufferPool creates a new buffer pool for memory management
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pools:   make(map[int]*sync.Pool),
		maxSize: 1024 * 1024, // 1MB max buffer size
	}
}

// NewWorkerPool creates a new worker pool with 2×NumCPU workers for optimal latency
func NewWorkerPool() *WorkerPool {
	numCPU := runtime.NumCPU()
	numWorkers := numCPU * 2 // 2×NumCPU for optimal queue balancing

	wp := &WorkerPool{
		numWorkers: numWorkers,
		workers:    make([]*Worker, numWorkers),
		taskChan:   make(chan Task, numWorkers*4), // Buffer for 4 tasks per worker
		quitChan:   make(chan struct{}),
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		worker := &Worker{
			id:       i,
			taskChan: wp.taskChan,
			quitChan: wp.quitChan,
		}
		wp.workers[i] = worker
		wp.wg.Add(1)
		go worker.start(&wp.wg)
	}

	return wp
}

// NewBackpressureController creates a new backpressure controller
func NewBackpressureController() *BackpressureController {
	return &BackpressureController{
		maxQueueDepth:    1000,
		samplingInterval: 100 * time.Millisecond,
		lastAdjustment:   time.Now(),
	}
}

// NewPipelineStats creates a new pipeline statistics tracker
func NewPipelineStats() *PipelineStats {
	numCPU := runtime.NumCPU()
	return &PipelineStats{
		workerUtilization: make([]float64, numCPU*2),
		processingLatency: make([]time.Duration, numCPU*2),
		lastUpdate:        time.Now(),
	}
}

// start begins the worker's processing loop
func (w *Worker) start(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case task := <-w.taskChan:
			w.processing = true
			start := time.Now()

			// Execute task with error handling
			if err := task.Handler(task.Payload); err != nil {
				// Log error but continue processing
				fmt.Printf("Worker %d: task %s failed: %v\n", w.id, task.ID, err)
			}

			w.lastTask = time.Now()
			w.processing = false

			// Update pipeline stats
			latency := time.Since(start)
			_ = latency // TODO: Update pipeline stats with latency

		case <-w.quitChan:
			return
		}
	}
}

// SubmitTask submits a task to the worker pool with backpressure handling
func (wp *WorkerPool) SubmitTask(task Task) bool {
	select {
	case wp.taskChan <- task:
		return true
	default:
		// Channel is full, backpressure engaged
		return false
	}
}

// SubmitTaskBlocking submits a task to the worker pool, blocking if necessary
func (wp *WorkerPool) SubmitTaskBlocking(task Task) {
	wp.taskChan <- task
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.quitChan)
	wp.wg.Wait()
}

// GetQueueDepth returns the current queue depth
func (wp *WorkerPool) GetQueueDepth() int {
	return len(wp.taskChan)
}

// GetWorkerStats returns statistics about worker utilization
func (wp *WorkerPool) GetWorkerStats() []bool {
	stats := make([]bool, len(wp.workers))
	for i, worker := range wp.workers {
		stats[i] = worker.processing
	}
	return stats
}

// ShouldThrottle determines if backpressure should be applied
func (bc *BackpressureController) ShouldThrottle() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.backpressureLevel > 0
}

// UpdateQueueDepth updates the current queue depth and adjusts backpressure
func (bc *BackpressureController) UpdateQueueDepth(depth int) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.queueDepth = depth
	now := time.Now()

	// Adjust backpressure based on queue depth
	if depth > bc.maxQueueDepth*8/10 { // 80% of max
		bc.backpressureLevel = 3 // High backpressure
	} else if depth > bc.maxQueueDepth*6/10 { // 60% of max
		bc.backpressureLevel = 2 // Medium backpressure
	} else if depth > bc.maxQueueDepth*4/10 { // 40% of max
		bc.backpressureLevel = 1 // Light backpressure
	} else {
		bc.backpressureLevel = 0 // No backpressure
	}

	bc.lastAdjustment = now
}

// GetBackpressureLevel returns the current backpressure level
func (bc *BackpressureController) GetBackpressureLevel() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.backpressureLevel
}

// GetStats returns backpressure statistics
func (bc *BackpressureController) GetStats() (int, int, int64) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.queueDepth, bc.backpressureLevel, bc.backpressureEvents
}

// UpdateLatency updates processing latency for a worker
func (ps *PipelineStats) UpdateLatency(workerID int, latency time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if workerID < len(ps.processingLatency) {
		ps.processingLatency[workerID] = latency
	}
	ps.lastUpdate = time.Now()
}

// UpdateUtilization updates worker utilization
func (ps *PipelineStats) UpdateUtilization(workerID int, utilization float64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if workerID < len(ps.workerUtilization) {
		ps.workerUtilization[workerID] = utilization
	}
	ps.lastUpdate = time.Now()
}

// UpdateQueueDepth updates the current queue depth
func (ps *PipelineStats) UpdateQueueDepth(depth int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.queueDepth = depth
	ps.lastUpdate = time.Now()
}

// GetAverageLatency returns the average processing latency across all workers
func (ps *PipelineStats) GetAverageLatency() time.Duration {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if len(ps.processingLatency) == 0 {
		return 0
	}

	var total time.Duration
	count := 0
	for _, latency := range ps.processingLatency {
		if latency > 0 {
			total += latency
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}

// GetAverageUtilization returns the average worker utilization
func (ps *PipelineStats) GetAverageUtilization() float64 {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if len(ps.workerUtilization) == 0 {
		return 0
	}

	var total float64
	count := 0
	for _, util := range ps.workerUtilization {
		if util > 0 {
			total += util
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// GetStats returns current pipeline statistics
func (ps *PipelineStats) GetStats() (time.Duration, float64, int) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.GetAverageLatency(), ps.GetAverageUtilization(), ps.queueDepth
}

// Get retrieves a buffer from the pool with optional memory locking
func (bp *BufferPool) Get(size int) []byte {
	if size > bp.maxSize {
		return make([]byte, size)
	}

	bp.mu.RLock()
	pool, exists := bp.pools[size]
	bp.mu.RUnlock()

	if !exists {
		bp.mu.Lock()
		if bp.pools[size] == nil {
			bp.pools[size] = &sync.Pool{
				New: func() interface{} {
					return make([]byte, size)
				},
			}
		}
		pool = bp.pools[size]
		bp.mu.Unlock()
	}

	buf := pool.Get().([]byte)

	// Attempt to lock buffer in memory for sensitive data (best-effort)
	bp.lockBufferInMemory(buf)

	return buf
}

// Put returns a buffer to the pool with secure zeroization
func (bp *BufferPool) Put(buf []byte) {
	size := cap(buf)
	if size > bp.maxSize {
		return
	}

	// Unlock buffer from memory before returning to pool
	bp.unlockBufferFromMemory(buf)

	bp.mu.RLock()
	pool, exists := bp.pools[size]
	bp.mu.RUnlock()

	if exists {
		// Clear buffer before returning to pool
		for i := range buf {
			buf[i] = 0
		}
		pool.Put(buf[:0]) // Reset length but keep capacity
	}
}

// lockBufferInMemory attempts to lock buffer in memory (best-effort)
func (bp *BufferPool) lockBufferInMemory(buf []byte) {
	// On Windows, use VirtualLock for memory locking
	// This is a best-effort operation - failure is not fatal
	defer func() {
		if r := recover(); r != nil {
			// Silently ignore locking failures
		}
	}()

	// For sensitive data buffers, attempt to lock in memory
	// This prevents sensitive data from being paged out
	if len(buf) > 0 {
		// Use Windows VirtualLock via kernel32.dll
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		virtualLock := kernel32.NewProc("VirtualLock")

		if virtualLock != nil {
			addr := uintptr(unsafe.Pointer(&buf[0]))
			size := uintptr(len(buf))
			virtualLock.Call(addr, size)
		}
	}
}

// unlockBufferFromMemory unlocks buffer from memory
func (bp *BufferPool) unlockBufferFromMemory(buf []byte) {
	defer func() {
		if r := recover(); r != nil {
			// Silently ignore unlocking failures
		}
	}()

	if len(buf) > 0 {
		// Use Windows VirtualUnlock via kernel32.dll
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		virtualUnlock := kernel32.NewProc("VirtualUnlock")

		if virtualUnlock != nil {
			addr := uintptr(unsafe.Pointer(&buf[0]))
			size := uintptr(len(buf))
			virtualUnlock.Call(addr, size)
		}
	}
}

// ApplyOptimizations applies all performance optimizations based on configuration
func (pm *PerformanceManager) ApplyOptimizations() error {
	pm.logger.Info("Applying performance optimizations",
		zap.String("level", pm.GetOptimizationLevelName()),
		zap.String("tier", string(pm.cfg.Tier)),
	)

	// 1. Runtime optimizations (always applied)
	pm.applyRuntimeOptimizations()

	// 2. Memory optimizations
	pm.applyMemoryOptimizations()

	// 3. CPU optimizations
	pm.applyCPUOptimizations()

	// 4. System-level optimizations (if enabled)
	if pm.cfg.OptimizeSystem {
		if err := pm.applySystemOptimizations(); err != nil {
			pm.logger.Warn("System optimizations failed", zap.Error(err))
			// Don't fail startup, just log warning
		}
	}

	pm.logger.Info("Performance optimizations applied successfully",
		zap.Int("gomaxprocs", runtime.GOMAXPROCS(0)),
		zap.Int("gc_percent", pm.cfg.GCPercent),
	)

	return nil
}

// GetBufferPool returns the buffer pool for external use
func (pm *PerformanceManager) GetBufferPool() *BufferPool {
	return pm.bufferPool
}

// applyRuntimeOptimizations applies Go runtime optimizations
func (pm *PerformanceManager) applyRuntimeOptimizations() {
	// Lock main thread to OS thread for consistent latency
	if pm.cfg.LockOSThread {
		runtime.LockOSThread()
		pm.logger.Debug("Locked main thread to OS thread")
	}

	// Configure garbage collector
	if pm.cfg.GCPercent > 0 {
		oldPercent := debug.SetGCPercent(pm.cfg.GCPercent)
		pm.logger.Debug("Configured garbage collector",
			zap.Int("old_percent", oldPercent),
			zap.Int("new_percent", pm.cfg.GCPercent),
		)
	}

	// Set CPU core usage
	if pm.cfg.MaxCPUCores > 0 {
		oldProcs := runtime.GOMAXPROCS(pm.cfg.MaxCPUCores)
		pm.logger.Debug("Configured CPU cores",
			zap.Int("old_procs", oldProcs),
			zap.Int("new_procs", pm.cfg.MaxCPUCores),
		)
	} else if pm.cfg.MaxCPUCores == 0 {
		// Auto-detect and use all available cores
		cores := runtime.NumCPU()
		runtime.GOMAXPROCS(cores)
		pm.logger.Debug("Auto-configured CPU cores", zap.Int("cores", cores))
	}
}

// applyMemoryOptimizations applies memory-related optimizations
func (pm *PerformanceManager) applyMemoryOptimizations() {
	if pm.level >= LevelHigh {
		// Pre-allocate commonly used buffers
		if pm.cfg.PreallocBuffers {
			pm.preallocateBuffers()
		}
	}

	if pm.level >= LevelMaximum {
		// Maximum performance: disable GC for ultra-low latency
		debug.SetGCPercent(-1)
		pm.logger.Debug("Disabled garbage collector for maximum performance")
	}
}

// applyCPUOptimizations applies CPU-related optimizations
func (pm *PerformanceManager) applyCPUOptimizations() {
	// CPU affinity and priority optimizations will be applied
	// in applySystemOptimizations() as they require OS-specific calls
}

// applySystemOptimizations applies OS-level optimizations
func (pm *PerformanceManager) applySystemOptimizations() error {
	if pm.cfg.HighPriority {
		if err := pm.setHighPriority(); err != nil {
			return fmt.Errorf("failed to set high priority: %w", err)
		}
	}

	return nil
}

// setHighPriority sets the process to high priority (Windows-specific)
func (pm *PerformanceManager) setHighPriority() error {
	if runtime.GOOS == "windows" {
		return pm.setWindowsHighPriority()
	}

	// For Unix-like systems, we could implement nice() calls here
	pm.logger.Debug("High priority optimization not implemented for this OS")
	return nil
}

// preallocateBuffers pre-allocates commonly used memory buffers
func (pm *PerformanceManager) preallocateBuffers() {
	// Pre-allocate buffers based on tier requirements
	bufferSize := pm.cfg.BlockBufferSize
	if bufferSize == 0 {
		bufferSize = 1024 // Default buffer size
	}

	// Pre-populate the buffer pool with commonly used buffer sizes
	bufferSizes := []int{bufferSize, bufferSize * 2, bufferSize * 4, 4096, 8192}

	totalBuffers := 0
	for _, size := range bufferSizes {
		// Pre-allocate 5 buffers of each size
		for i := 0; i < 5; i++ {
			buf := pm.bufferPool.Get(size)
			pm.bufferPool.Put(buf) // Return to pool immediately
			totalBuffers++
		}
	}

	pm.logger.Debug("Pre-populated buffer pool",
		zap.Int("total_buffers", totalBuffers),
		zap.Int("buffer_sizes", len(bufferSizes)),
		zap.Int("pool_max_size", pm.bufferPool.maxSize),
	)
}

// GetOptimizationLevelName returns the string representation of optimization level
func (pm *PerformanceManager) GetOptimizationLevelName() string {
	switch pm.level {
	case LevelMaximum:
		return "maximum"
	case LevelHigh:
		return "high"
	default:
		return "standard"
	}
}

// GetCurrentStats returns current performance statistics
func (pm *PerformanceManager) GetCurrentStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"optimization_level": pm.GetOptimizationLevelName(),
		"tier":               string(pm.cfg.Tier),
		"runtime": map[string]interface{}{
			"gomaxprocs":     runtime.GOMAXPROCS(0),
			"num_cpu":        runtime.NumCPU(),
			"num_goroutines": runtime.NumGoroutine(),
			"gc_percent":     pm.cfg.GCPercent,
		},
		"memory": map[string]interface{}{
			"alloc_mb":       m.Alloc / 1024 / 1024,
			"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
			"sys_mb":         m.Sys / 1024 / 1024,
			"num_gc":         m.NumGC,
		},
		"config": map[string]interface{}{
			"lock_os_thread":   pm.cfg.LockOSThread,
			"high_priority":    pm.cfg.HighPriority,
			"prealloc_buffers": pm.cfg.PreallocBuffers,
			"optimize_system":  pm.cfg.OptimizeSystem,
		},
	}
}

// SubmitTask submits a task to the worker pool for processing
func (pm *PerformanceManager) SubmitTask(taskID string, payload interface{}, handler func(interface{}) error) bool {
	task := Task{
		ID:      taskID,
		Payload: payload,
		Handler: handler,
	}

	// Try non-blocking submit first
	if pm.workerPool.SubmitTask(task) {
		// Update backpressure controller
		pm.backpressure.UpdateQueueDepth(pm.workerPool.GetQueueDepth())
		return true
	}

	// If backpressure is engaged, check if we should throttle
	if pm.backpressure.ShouldThrottle() {
		pm.logger.Debug("Task submission throttled due to backpressure",
			zap.String("task_id", taskID),
			zap.Int("queue_depth", pm.workerPool.GetQueueDepth()),
		)
		return false
	}

	// Fallback to blocking submit for critical tasks
	pm.workerPool.SubmitTaskBlocking(task)
	pm.backpressure.UpdateQueueDepth(pm.workerPool.GetQueueDepth())
	return true
}

// SubmitTaskBlocking submits a task and blocks until it's accepted
func (pm *PerformanceManager) SubmitTaskBlocking(taskID string, payload interface{}, handler func(interface{}) error) {
	task := Task{
		ID:      taskID,
		Payload: payload,
		Handler: handler,
	}

	pm.workerPool.SubmitTaskBlocking(task)
	pm.backpressure.UpdateQueueDepth(pm.workerPool.GetQueueDepth())
}

// GetPerformanceStats returns current performance statistics
func (pm *PerformanceManager) GetPerformanceStats() map[string]interface{} {
	avgLatency, avgUtilization, queueDepth := pm.pipelineStats.GetStats()
	_, backpressureLevel, backpressureEvents := pm.backpressure.GetStats()
	workerStats := pm.workerPool.GetWorkerStats()

	return map[string]interface{}{
		"average_latency_ms":     avgLatency.Milliseconds(),
		"average_utilization":    avgUtilization,
		"queue_depth":            queueDepth,
		"backpressure_level":     backpressureLevel,
		"backpressure_events":    backpressureEvents,
		"active_workers":         pm.countActiveWorkers(workerStats),
		"total_workers":          len(workerStats),
		"worker_utilization_pct": (float64(pm.countActiveWorkers(workerStats)) / float64(len(workerStats))) * 100,
	}
}

// countActiveWorkers counts how many workers are currently processing tasks
func (pm *PerformanceManager) countActiveWorkers(stats []bool) int {
	count := 0
	for _, active := range stats {
		if active {
			count++
		}
	}
	return count
}

// Shutdown gracefully shuts down all performance components
func (pm *PerformanceManager) Shutdown() {
	pm.logger.Info("Shutting down performance manager")

	// Stop worker pool
	if pm.workerPool != nil {
		pm.workerPool.Stop()
	}

	pm.logger.Info("Performance manager shutdown complete")
}

// RunLatencyBenchmark runs a comprehensive latency benchmark to demonstrate flat latency curves
func (pm *PerformanceManager) RunLatencyBenchmark(duration time.Duration, concurrency int) *LatencyBenchmarkResult {
	pm.logger.Info("Starting flat latency benchmark",
		zap.Duration("duration", duration),
		zap.Int("concurrency", concurrency))

	start := time.Now()
	results := &LatencyBenchmarkResult{
		P50Latencies:  make([]time.Duration, 0),
		P95Latencies:  make([]time.Duration, 0),
		P99Latencies:  make([]time.Duration, 0),
		P999Latencies: make([]time.Duration, 0),
		RawLatencies:  make([]time.Duration, 0),
		StartTime:     start,
		EndTime:       start.Add(duration),
		Concurrency:   concurrency,
	}

	// Create work generator
	workChan := make(chan Task, concurrency*10)
	doneChan := make(chan struct{})

	// Start workers
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			for {
				select {
				case task := <-workChan:
					start := time.Now()
					err := task.Handler(task.Payload)
					latency := time.Since(start)

					results.mu.Lock()
					results.RawLatencies = append(results.RawLatencies, latency)
					results.TotalRequests++
					if err != nil {
						results.Errors++
					}
					results.mu.Unlock()

				case <-doneChan:
					return
				}
			}
		}(i)
	}

	// Generate work load with varying patterns to simulate real-world usage
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Simulate different types of requests
				for i := 0; i < concurrency; i++ {
					task := Task{
						ID: fmt.Sprintf("benchmark-%d", results.TotalRequests),
						Payload: map[string]interface{}{
							"type":    "benchmark",
							"size":    1024,
							"entropy": pm.cfg.Tier == config.TierTurbo,
						},
						Handler: func(payload interface{}) error {
							// Simulate processing with occasional entropy generation
							data := payload.(map[string]interface{})
							if entropyFlag, ok := data["entropy"].(bool); ok && entropyFlag {
								_, err := entropy.FastEntropy()
								if err != nil {
									return err
								}
							}
							// Simulate some processing time
							time.Sleep(time.Duration(100+rand.Intn(200)) * time.Microsecond)
							return nil
						},
					}

					select {
					case workChan <- task:
					default:
						// Backpressure engaged
						results.mu.Lock()
						results.BackpressureEvents++
						results.mu.Unlock()
					}
				}

			case <-doneChan:
				return
			}
		}
	}()

	// Run benchmark for specified duration
	time.Sleep(duration)
	close(doneChan)

	// Calculate percentiles
	results.calculatePercentiles()

	pm.logger.Info("Flat latency benchmark completed",
		zap.Int("total_requests", results.TotalRequests),
		zap.Int("errors", results.Errors),
		zap.Int("backpressure_events", results.BackpressureEvents),
		zap.Duration("p50_latency", results.P50Latency),
		zap.Duration("p95_latency", results.P95Latency),
		zap.Duration("p99_latency", results.P99Latency),
		zap.Duration("p999_latency", results.P999Latency))

	return results
}

// LatencyBenchmarkResult contains the results of a latency benchmark
type LatencyBenchmarkResult struct {
	mu                 sync.RWMutex
	P50Latencies       []time.Duration
	P95Latencies       []time.Duration
	P99Latencies       []time.Duration
	P999Latencies      []time.Duration
	RawLatencies       []time.Duration
	StartTime          time.Time
	EndTime            time.Time
	Concurrency        int
	TotalRequests      int
	Errors             int
	BackpressureEvents int

	// Calculated percentiles
	P50Latency  time.Duration
	P95Latency  time.Duration
	P99Latency  time.Duration
	P999Latency time.Duration
}

// calculatePercentiles calculates latency percentiles from raw data
func (r *LatencyBenchmarkResult) calculatePercentiles() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.RawLatencies) == 0 {
		return
	}

	// Sort latencies
	sorted := make([]time.Duration, len(r.RawLatencies))
	copy(sorted, r.RawLatencies)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	n := len(sorted)
	r.P50Latency = sorted[n*50/100]
	r.P95Latency = sorted[n*95/100]
	r.P99Latency = sorted[n*99/100]
	if n*999/1000 < n {
		r.P999Latency = sorted[n*999/1000]
	} else {
		r.P999Latency = sorted[n-1]
	}
}

// GetLatencyDistribution returns the latency distribution for analysis
func (r *LatencyBenchmarkResult) GetLatencyDistribution() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"p50_ms":              r.P50Latency.Milliseconds(),
		"p95_ms":              r.P95Latency.Milliseconds(),
		"p99_ms":              r.P99Latency.Milliseconds(),
		"p999_ms":             r.P999Latency.Milliseconds(),
		"total_requests":      r.TotalRequests,
		"errors":              r.Errors,
		"backpressure_events": r.BackpressureEvents,
		"concurrency":         r.Concurrency,
		"duration_seconds":    r.EndTime.Sub(r.StartTime).Seconds(),
		"requests_per_second": float64(r.TotalRequests) / r.EndTime.Sub(r.StartTime).Seconds(),
		"curve_flatness":      r.calculateFlatness(),
	}
}

// calculateFlatness calculates how flat the latency curve is (lower is better)
func (r *LatencyBenchmarkResult) calculateFlatness() float64 {
	if r.P50Latency == 0 {
		return 0
	}
	// Flatness is the ratio of P99 to P50 (lower ratio = flatter curve)
	return float64(r.P99Latency) / float64(r.P50Latency)
}
