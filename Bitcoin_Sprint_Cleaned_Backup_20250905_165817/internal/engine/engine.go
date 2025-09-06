// package engine implements a general-purpose processing engine with a Bitcoin plugin example.
package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	lru "github.com/decred/dcrd/lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ----------------------------- Generic Types & Interfaces -----------------------------

// Message is the generic envelope for tasks/results.
type Message struct {
	ID        string                 `json:"id"`
	Topic     string                 `json:"topic"`
	Payload   []byte                 `json:"payload"` // arbitrary payload bytes
	Data      map[string]interface{} `json:"data"`    // parsed JSON if applicable
	Timestamp time.Time              `json:"timestamp"`
	Meta      map[string]string      `json:"meta,omitempty"`
	BlockHash string                 `json:"blockhash,omitempty"`
}

// Task is a unit of work. Implement Execute to perform the work.
// The engine will call Task.Execute(ctx, engineHelpers).
type Task interface {
	// ID returns a unique task identifier
	ID() string
	// Execute runs the task; may produce zero or more Messages as results.
	Execute(ctx context.Context, helpers EngineHelpers) ([]Message, error)
}

// EngineHelpers exposes services available to tasks (cache, state, metrics).
type EngineHelpers struct {
	Cache      *ResultCache
	State      StateStore
	Seen       SeenStore
	HTTPClient *http.Client
	Metrics    *engineMetrics
}

// ----------------------------- Engine Implementation -----------------------------

// Engine runs tasks using a bounded worker pool.
type Engine struct {
	workerCount int
	queue       chan Task
	cache       *ResultCache
	state       StateStore
	seen        SeenStore
	httpClient  *http.Client
	metrics     *engineMetrics
	stopCtx     context.Context
	stopCancel  context.CancelFunc
	wg          sync.WaitGroup
	httpStop    func()
}

// NewEngine creates a new Engine.
func NewEngine(workerCount int, queueSize int, cacheSize int, state StateStore, seen SeenStore) (*Engine, error) {
	if workerCount <= 0 {
		workerCount = 8
	}
	if queueSize <= 0 {
		queueSize = 1024
	}
	cache, err := NewResultCache(cacheSize)
	if err != nil {
		return nil, err
	}
	if state == nil || seen == nil {
		return nil, errors.New("state and seen stores required")
	}
	m := newEngineMetrics()
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		workerCount: workerCount,
		queue:       make(chan Task, queueSize),
		cache:       cache,
		state:       state,
		seen:        seen,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		metrics:    m,
		stopCtx:    ctx,
		stopCancel: cancel,
	}, nil
}

// Start launches worker goroutines and optional HTTP server.
func (e *Engine) Start(enableHTTP bool, httpAddr string) error {
	// Start workers
	for i := 0; i < e.workerCount; i++ {
		e.wg.Add(1)
		go e.worker(i)
	}
	// Start HTTP server for metrics if requested
	if enableHTTP {
		stopFn, err := StartMetricsServer(httpAddr, func() bool { return true })
		if err != nil {
			return err
		}
		e.httpStop = stopFn
	}
	log.Printf("Engine started: workers=%d queue=%d cache=%d", e.workerCount, cap(e.queue), e.cache.Size())
	return nil
}

// Stop gracefully shuts down the engine.
func (e *Engine) Stop() {
	e.stopCancel()
	if e.httpStop != nil {
		e.httpStop()
	}
	close(e.queue) // allow workers to drain
	e.wg.Wait()
}

// Submit enqueues a task. Returns error if engine stopping.
func (e *Engine) Submit(task Task) error {
	select {
	case <-e.stopCtx.Done():
		return errors.New("engine stopping")
	default:
	}
	select {
	case e.queue <- task:
		e.metrics.tasksQueued.Inc()
		return nil
	default:
		// queue full
		e.metrics.queueDrops.Inc()
		return errors.New("task queue full")
	}
}

// worker processes tasks from the queue.
func (e *Engine) worker(id int) {
	defer e.wg.Done()
	log.Printf("worker-%d started", id)
	for {
		select {
		case <-e.stopCtx.Done():
			return
		case task, ok := <-e.queue:
			if !ok {
				return
			}
			start := time.Now()
			helpers := EngineHelpers{
				Cache:      e.cache,
				State:      e.state,
				Seen:       e.seen,
				HTTPClient: e.httpClient,
				Metrics:    e.metrics,
			}
			results, err := task.Execute(e.stopCtx, helpers)
			// update metrics
			e.metrics.tasksProcessed.Inc()
			e.metrics.taskProcessingTime.Observe(time.Since(start).Seconds())
			if err != nil {
				e.metrics.taskErrors.Inc()
				log.Printf("task %s failed: %v", task.ID(), err)
				continue
			}
			// store results in cache & persist seen
			for _, m := range results {
				if m.ID != "" {
					e.cache.Add(m.ID, m)
					if err := e.seen.Add(context.Background(), m.ID); err != nil {
						log.Printf("seen add error: %v", err)
					}
				}
				// In a real system you'd publish results to an output sink (DB, queue, etc.)
				e.metrics.messagesProduced.Inc()
			}
		}
	}
}

// ----------------------------- Result Cache (LRU) -----------------------------

// ResultCache wraps an LRU cache storing Messages.
type ResultCache struct {
	cache *lru.Cache
	mu    sync.RWMutex
}

func NewResultCache(size int) (*ResultCache, error) {
	if size <= 0 {
		size = 1024
	}
	c := lru.NewCache(size)
	return &ResultCache{cache: c}, nil
}

func (r *ResultCache) Get(key string) (Message, bool) {
	r.mu.RLock()
	v, ok := r.cache.Get(key)
	r.mu.RUnlock()
	if !ok {
		return Message{}, false
	}
	msg, _ := v.(Message)
	return msg, true
}

func (r *ResultCache) Add(key string, msg Message) {
	r.mu.Lock()
	r.cache.Put(key, msg)
	r.mu.Unlock()
}

func (r *ResultCache) Size() int {
	r.mu.RLock()
	n := r.cache.Len()
	r.mu.RUnlock()
	return n
}

// ----------------------------- State & Seen Stores -----------------------------

// StateStore persists last processed position and failed IDs.
type StateStore interface {
	LoadLastID(ctx context.Context) (string, error)
	SaveLastID(ctx context.Context, id string) error
	LoadFailedIDs(ctx context.Context) ([]string, error)
	SaveFailedIDs(ctx context.Context, ids []string) error
}

// SeenStore persists seen IDs for dedup across restarts.
type SeenStore interface {
	Has(ctx context.Context, id string) (bool, error)
	Add(ctx context.Context, id string) error
	Compact(ctx context.Context, keep int) error
}

// FileStateStore implements StateStore using files (NDJSON for failed).
type FileStateStore struct {
	LastIDFile string
	FailedFile string
	mu         sync.Mutex
}

func NewFileStateStore(lastIDFile, failedFile string) *FileStateStore {
	return &FileStateStore{LastIDFile: lastIDFile, FailedFile: failedFile}
}

func (f *FileStateStore) LoadLastID(ctx context.Context) (string, error) {
	data, err := os.ReadFile(f.LastIDFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (f *FileStateStore) SaveLastID(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	tmp := f.LastIDFile + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(id + "\n"); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return os.Rename(tmp, f.LastIDFile)
}

func (f *FileStateStore) LoadFailedIDs(ctx context.Context) ([]string, error) {
	file, err := os.Open(f.FailedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	var ids []string
	sc := bufio.NewScanner(file)
	buf := make([]byte, 64*1024) // 64KB buffer
	sc.Buffer(buf, 1024*1024)    // Max 1MB per line
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			ids = append(ids, line)
		}
	}
	return ids, sc.Err()
}

func (f *FileStateStore) SaveFailedIDs(ctx context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			set[id] = struct{}{}
		}
	}
	tmp := f.FailedFile + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		if fh != nil {
			_ = fh.Close()
		}
	}()

	for id := range set {
		if _, err := fh.WriteString(id + "\n"); err != nil {
			return err
		}
	}
	if err := fh.Sync(); err != nil {
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}
	fh = nil
	return os.Rename(tmp, f.FailedFile)
}

// FileSeenStore: append-only NDJSON of seen IDs with in-memory index.
type FileSeenStore struct {
	Path    string
	MaxKeep int
	mu      sync.RWMutex
	seen    map[string]struct{}
	order   []string
	loaded  bool
}

func NewFileSeenStore(path string, maxKeep int) *FileSeenStore {
	return &FileSeenStore{
		Path:    path,
		MaxKeep: maxKeep,
		seen:    make(map[string]struct{}, maxKeep),
		order:   make([]string, 0, maxKeep),
	}
}

func (s *FileSeenStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return nil
	}
	f, err := os.Open(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			s.loaded = true
			return nil
		}
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	buf := make([]byte, 64*1024) // 64KB buffer
	sc.Buffer(buf, 1024*1024)    // Max 1MB per line
	for sc.Scan() {
		id := strings.TrimSpace(sc.Text())
		if id == "" {
			continue
		}
		if _, ok := s.seen[id]; !ok {
			s.seen[id] = struct{}{}
			s.order = append(s.order, id)
		}
	}
	s.loaded = true
	return sc.Err()
}

func (s *FileSeenStore) Has(ctx context.Context, id string) (bool, error) {
	if err := s.load(); err != nil {
		return false, err
	}
	s.mu.RLock()
	_, ok := s.seen[id]
	s.mu.RUnlock()
	return ok, nil
}

func (s *FileSeenStore) Add(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	if err := s.load(); err != nil {
		return err
	}
	s.mu.RLock()
	if _, ok := s.seen[id]; ok {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// append to file
	fh, err := os.OpenFile(s.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if fh != nil {
			_ = fh.Close()
		}
	}()

	if _, err := fh.WriteString(id + "\n"); err != nil {
		return err
	}
	if err := fh.Sync(); err != nil {
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}
	fh = nil
	s.seen[id] = struct{}{}
	s.order = append(s.order, id)
	// compact if needed
	if s.MaxKeep > 0 && len(s.order) > s.MaxKeep {
		toKeep := s.order[len(s.order)-s.MaxKeep:]
		tmp := s.Path + ".tmp"
		fh2, err := os.Create(tmp)
		if err == nil {
			defer func() {
				if fh2 != nil {
					_ = fh2.Close()
				}
			}()
			for _, v := range toKeep {
				if _, err := fh2.WriteString(v + "\n"); err != nil {
					return err
				}
			}
			if err := fh2.Sync(); err != nil {
				return err
			}
			if err := fh2.Close(); err != nil {
				return err
			}
			fh2 = nil
			if err := os.Rename(tmp, s.Path); err != nil {
				return err
			}
		}
		// rebuild index
		ns := make(map[string]struct{}, s.MaxKeep)
		for _, v := range toKeep {
			ns[v] = struct{}{}
		}
		s.seen = ns
		s.order = append([]string{}, toKeep...)
	}
	return nil
}

func (s *FileSeenStore) Compact(ctx context.Context, keep int) error {
	if err := s.load(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if keep <= 0 || len(s.order) <= keep {
		return nil
	}
	start := len(s.order) - keep
	tmp := s.Path + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		if fh != nil {
			_ = fh.Close()
		}
	}()

	for _, id := range s.order[start:] {
		if _, err := fh.WriteString(id + "\n"); err != nil {
			return err
		}
	}
	if err := fh.Sync(); err != nil {
		return err
	}
	if err := fh.Close(); err != nil {
		return err
	}
	fh = nil
	if err := os.Rename(tmp, s.Path); err != nil {
		return err
	}
	s.order = append([]string{}, s.order[start:]...)
	m := make(map[string]struct{}, len(s.order))
	for _, id := range s.order {
		m[id] = struct{}{}
	}
	s.seen = m
	return nil
}

// ----------------------------- Metrics & HTTP -----------------------------

type engineMetrics struct {
	tasksQueued        prometheus.Counter
	tasksProcessed     prometheus.Counter
	taskErrors         prometheus.Counter
	taskProcessingTime prometheus.Histogram
	messagesProduced   prometheus.Counter
	tasksDropped       prometheus.Counter
	queueDrops         prometheus.Counter
	tasksTotal         prometheus.Counter
}

func newEngineMetrics() *engineMetrics {
	return &engineMetrics{
		tasksQueued: promauto.NewCounter(prometheus.CounterOpts{
			Name: "engine_tasks_queued_total",
			Help: "Total tasks queued",
		}),
		tasksProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "engine_tasks_processed_total",
			Help: "Total tasks processed",
		}),
		taskErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "engine_task_errors_total",
			Help: "Total task errors",
		}),
		taskProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "engine_task_processing_seconds",
			Help:    "Task processing duration",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
		}),
		messagesProduced: promauto.NewCounter(prometheus.CounterOpts{
			Name: "engine_messages_produced_total",
			Help: "Messages produced by tasks",
		}),
		queueDrops: promauto.NewCounter(prometheus.CounterOpts{
			Name: "engine_queue_drops_total",
			Help: "Tasks dropped due to full queue",
		}),
	}
}

func StartMetricsServer(addr string, readyFn func() bool) (stop func(), err error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if readyFn != nil && !readyFn() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server error: %v", err)
		}
	}()
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(ctx)
		cancel()
	}, nil
}

// ----------------------------- Bitcoin Task (Plugin) -----------------------------

// BitcoinRPCConfig holds connection info for the Bitcoin node; minimal fields here.
// For production, expand as in earlier code.
type BitcoinRPCConfig struct {
	URL           string
	Username      string
	Password      string
	Timeout       time.Duration
	MaxBlocks     int
	MaxTxPerBlock int
	MaxWorkers    int
	BatchSize     int
	Topic         string
	RetryAttempts int
	RetryMaxWait  time.Duration
	SkipMempool   bool
	FailedFile    string
}

// BitcoinTask implements Task to perform a backfill pass.
type BitcoinTask struct {
	id     string
	cfg    BitcoinRPCConfig
	state  StateStore
	seen   SeenStore
	client *http.Client
}

func NewBitcoinTask(id string, cfg BitcoinRPCConfig, state StateStore, seen SeenStore) *BitcoinTask {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 8
	}
	if cfg.MaxBlocks <= 0 {
		cfg.MaxBlocks = 100
	}
	if cfg.Topic == "" {
		cfg.Topic = "bitcoin_tx"
	}
	return &BitcoinTask{
		id:     id,
		cfg:    cfg,
		state:  state,
		seen:   seen,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

func (t *BitcoinTask) ID() string { return t.id }

// Execute performs a backfill pass; uses helpers.Cache/State/Seen where appropriate.
func (t *BitcoinTask) Execute(ctx context.Context, helpers EngineHelpers) ([]Message, error) {
	// Simple integration: load last ID from state, perform backfill like previous implementations.
	// For brevity, we implement a pragmatic approach: fetch mempool (optional) and N blocks.
	// This uses batch RPC calls with retries. On error, we return partial results + error.

	// RPC helpers with backoff
	rpcCall := func(method string, params []interface{}) (json.RawMessage, error) {
		var res json.RawMessage
		operation := func() error {
			reqBody, _ := json.Marshal(map[string]interface{}{
				"jsonrpc": "1.0",
				"id":      fmt.Sprintf("rpc-%d", time.Now().UnixNano()),
				"method":  method,
				"params":  params,
			})
			req, _ := http.NewRequestWithContext(ctx, "POST", t.cfg.URL, bytes.NewReader(reqBody))
			req.SetBasicAuth(t.cfg.Username, t.cfg.Password)
			req.Header.Set("Content-Type", "application/json")
			resp, err := t.client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			var rpcResp struct {
				Result json.RawMessage `json:"result"`
				Error  *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
				return err
			}
			if rpcResp.Error != nil {
				return fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
			}
			res = rpcResp.Result
			return nil
		}
		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = t.cfg.RetryMaxWait
		if err := backoff.Retry(operation, backoff.WithMaxRetries(backoff.WithContext(b, ctx), uint64(t.cfg.RetryAttempts))); err != nil {
			return nil, err
		}
		return res, nil
	}

	// Batch helper (simple)
	batchRpc := func(txIDs []string) (map[string]json.RawMessage, []string) {
		results := map[string]json.RawMessage{}
		failures := []string{}
		if len(txIDs) == 0 {
			return results, failures
		}
		// build batch
		batch := make([]map[string]interface{}, 0, len(txIDs))
		for i, id := range txIDs {
			batch = append(batch, map[string]interface{}{
				"jsonrpc": "1.0",
				"id":      fmt.Sprintf("b-%d-%s", i, id),
				"method":  "getrawtransaction",
				"params":  []interface{}{id, true},
			})
		}
		reqBody, _ := json.Marshal(batch)
		req, _ := http.NewRequestWithContext(ctx, "POST", t.cfg.URL, bytes.NewReader(reqBody))
		req.SetBasicAuth(t.cfg.Username, t.cfg.Password)
		req.Header.Set("Content-Type", "application/json")
		resp, err := t.client.Do(req)
		if err != nil {
			for _, id := range txIDs {
				failures = append(failures, id)
			}
			return results, failures
		}
		defer resp.Body.Close()
		var batchResp []struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			ID string `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
			for _, id := range txIDs {
				failures = append(failures, id)
			}
			return results, failures
		}
		for i, r := range batchResp {
			if r.Error != nil {
				failures = append(failures, txIDs[i])
				continue
			}
			results[txIDs[i]] = r.Result
		}
		return results, failures
	}

	// main flow: load last block id
	lastID, _ := t.state.LoadLastID(ctx)
	messages := []Message{}

	// optionally process mempool (skipped by default)
	if !t.cfg.SkipMempool {
		if raw, err := rpcCall("getrawmempool", nil); err == nil {
			var txs []string
			_ = json.Unmarshal(raw, &txs)
			// append any previous failed IDs from state
			prevFailed, _ := t.state.LoadFailedIDs(ctx)
			all := append(txs, prevFailed...)
			// process batches
			for i := 0; i < len(all); i += t.cfg.BatchSize {
				end := i + t.cfg.BatchSize
				if end > len(all) {
					end = len(all)
				}
				batch := all[i:end]
				results, failures := batchRpc(batch)
				// convert results to Messages
				for id, raw := range results {
					// check cache
					if _, ok := helpers.Cache.Get(id); ok {
						continue
					}
					var tx map[string]interface{}
					_ = json.Unmarshal(raw, &tx)
					msg := Message{
						ID:        id,
						Topic:     t.cfg.Topic,
						Data:      tx,
						Timestamp: time.Now().UTC(),
					}
					messages = append(messages, msg)
					helpers.Cache.Add(id, msg)
					_ = helpers.Seen.Add(ctx, id)
				}
				// persist failures for retry
				if len(failures) > 0 {
					_ = t.state.SaveFailedIDs(ctx, failures)
				}
			}
		}
	}

	// process blocks since lastID (naive: walk forward N blocks starting at lastID or tip)
	startHash := lastID
	if startHash == "" {
		if raw, err := rpcCall("getblockchaininfo", nil); err == nil {
			var info map[string]interface{}
			_ = json.Unmarshal(raw, &info)
			if h, ok := info["bestblockhash"].(string); ok {
				startHash = h
			}
		}
	}
	current := startHash
	blocksProcessed := 0

	for current != "" && blocksProcessed < t.cfg.MaxBlocks {
		select {
		case <-ctx.Done():
			return messages, ctx.Err()
		default:
		}
		rawBlock, err := rpcCall("getblock", []interface{}{current, 2})
		if err != nil {
			break
		}
		var block map[string]interface{}
		_ = json.Unmarshal(rawBlock, &block)
		// collect txids
		if txs, ok := block["tx"].([]interface{}); ok {
			txIDs := make([]string, 0, len(txs))
			for i, v := range txs {
				if t.cfg.MaxTxPerBlock > 0 && i >= t.cfg.MaxTxPerBlock {
					break
				}
				switch x := v.(type) {
				case string:
					txIDs = append(txIDs, x)
				case map[string]interface{}:
					if id, ok := x["txid"].(string); ok {
						txIDs = append(txIDs, id)
					}
				}
			}
			// batch
			for i := 0; i < len(txIDs); i += t.cfg.BatchSize {
				end := i + t.cfg.BatchSize
				if end > len(txIDs) {
					end = len(txIDs)
				}
				batch := txIDs[i:end]
				results, failures := batchRpc(batch)
				// iterate results
				for _, id := range batch {
					if raw, ok := results[id]; ok {
						// check cache/seen
						if _, ok := helpers.Cache.Get(id); ok {
							continue
						}
						if has, _ := helpers.Seen.Has(ctx, id); has {
							continue
						}
						var tx map[string]interface{}
						_ = json.Unmarshal(raw, &tx)
						msg := Message{
							ID:        id,
							Topic:     t.cfg.Topic,
							Data:      tx,
							Timestamp: time.Now().UTC(),
						}
						if bh, ok := tx["blockhash"].(string); ok {
							msg.BlockHash = bh
						}
						messages = append(messages, msg)
						helpers.Cache.Add(id, msg)
						_ = helpers.Seen.Add(ctx, id)
					}
				}
				if len(failures) > 0 {
					_ = t.state.SaveFailedIDs(ctx, failures)
				}
			}
		}
		// advance
		next, _ := block["nextblockhash"].(string)
		if next == "" {
			break
		}
		current = next
		blocksProcessed++
		_ = t.state.SaveLastID(ctx, current)
	}
	return messages, nil
}

// ----------------------------- Utility: signal-run -----------------------------

// RunWithSignals runs fn(ctx) and cancels on SIGINT/SIGTERM.
func RunWithSignals(fn func(ctx context.Context) error) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- fn(ctx) }()
	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}
