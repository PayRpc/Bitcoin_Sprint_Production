package runtime

import (
	"runtime"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.Level != OptimizationStandard {
		t.Errorf("Expected OptimizationStandard, got %v", config.Level)
	}
	
	if config.MaxThreads <= 0 {
		t.Errorf("Expected positive MaxThreads, got %d", config.MaxThreads)
	}
	
	if config.GCTargetPercent <= 0 {
		t.Errorf("Expected positive GCTargetPercent, got %d", config.GCTargetPercent)
	}
	
	if config.MemoryLimitPercent <= 0 || config.MemoryLimitPercent > 100 {
		t.Errorf("Expected valid MemoryLimitPercent (1-100), got %d", config.MemoryLimitPercent)
	}
}

func TestEnterpriseConfig(t *testing.T) {
	config := EnterpriseConfig()
	
	if config.Level != OptimizationEnterprise {
		t.Errorf("Expected OptimizationEnterprise, got %v", config.Level)
	}
	
	if !config.EnableCPUPinning {
		t.Error("Expected CPU pinning to be enabled for enterprise config")
	}
	
	if !config.EnableRTPriority {
		t.Error("Expected RT priority to be enabled for enterprise config")
	}
	
	if config.GCTargetPercent >= DefaultConfig().GCTargetPercent {
		t.Error("Expected enterprise config to have lower GC target than default")
	}
}

func TestTurboConfig(t *testing.T) {
	config := TurboConfig()
	
	if config.Level != OptimizationTurbo {
		t.Errorf("Expected OptimizationTurbo, got %v", config.Level)
	}
	
	if config.GCTargetPercent >= EnterpriseConfig().GCTargetPercent {
		t.Error("Expected turbo config to have lower GC target than enterprise")
	}
	
	if config.ThreadStackSize >= DefaultConfig().ThreadStackSize {
		t.Error("Expected turbo config to have smaller stack size for cache efficiency")
	}
}

func TestNewSystemOptimizer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// Test with nil config
	optimizer := NewSystemOptimizer(nil, logger)
	if optimizer == nil {
		t.Fatal("Expected non-nil optimizer")
	}
	if optimizer.config == nil {
		t.Fatal("Expected default config when nil provided")
	}
	
	// Test with custom config
	config := &SystemOptimizationConfig{
		Level:           OptimizationBasic,
		MaxThreads:      4,
		GCTargetPercent: 100,
	}
	optimizer = NewSystemOptimizer(config, logger)
	if optimizer.config.Level != OptimizationBasic {
		t.Errorf("Expected OptimizationBasic, got %v", optimizer.config.Level)
	}
}

func TestSystemOptimizerApply(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.Level = OptimizationBasic // Use basic to avoid aggressive changes
	
	optimizer := NewSystemOptimizer(config, logger)
	
	// Test first application
	err := optimizer.Apply()
	if err != nil {
		t.Errorf("Expected no error on first Apply, got %v", err)
	}
	
	if !optimizer.applied {
		t.Error("Expected applied flag to be true after Apply")
	}
	
	// Test double application
	err = optimizer.Apply()
	if err == nil {
		t.Error("Expected error on second Apply")
	}
}

func TestSystemOptimizerRestore(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.Level = OptimizationBasic
	
	optimizer := NewSystemOptimizer(config, logger)
	
	// Test restore without apply
	err := optimizer.Restore()
	if err == nil {
		t.Error("Expected error when restoring without applying")
	}
	
	// Apply and then restore
	err = optimizer.Apply()
	if err != nil {
		t.Fatalf("Failed to apply optimizations: %v", err)
	}
	
	err = optimizer.Restore()
	if err != nil {
		t.Errorf("Expected no error on restore, got %v", err)
	}
	
	if optimizer.applied {
		t.Error("Expected applied flag to be false after restore")
	}
}

func TestGetStats(t *testing.T) {
	logger := zaptest.NewLogger(t)
	optimizer := NewSystemOptimizer(DefaultConfig(), logger)
	
	stats := optimizer.GetStats()
	
	expectedKeys := []string{
		"applied", "level", "gomaxprocs", "num_cpu", "num_goroutine",
		"heap_alloc_mb", "heap_sys_mb", "num_gc", "gc_cpu_fraction", "go_version",
	}
	
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stats to contain key %s", key)
		}
	}
	
	if stats["applied"] != false {
		t.Error("Expected applied to be false initially")
	}
	
	if stats["level"] != "standard" {
		t.Errorf("Expected level to be 'standard', got %v", stats["level"])
	}
}

func TestApplySystemOptimizations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// This should not panic or error
	ApplySystemOptimizations(logger)
}

func TestApplyEnterpriseOptimizations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	err := ApplyEnterpriseOptimizations(logger)
	if err != nil {
		t.Errorf("Expected no error from ApplyEnterpriseOptimizations, got %v", err)
	}
}

func TestTriggerGC(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// Allocate some memory to make GC meaningful
	data := make([][]byte, 1000)
	for i := range data {
		data[i] = make([]byte, 1024)
	}
	
	// This should not panic
	TriggerGC(logger)
	
	// Clear reference to allow GC
	data = nil
}

func TestGetOptimalGOMAXPROCS(t *testing.T) {
	optimal := GetOptimalGOMAXPROCS()
	numCPU := runtime.NumCPU()
	
	if optimal <= 0 {
		t.Errorf("Expected positive optimal GOMAXPROCS, got %d", optimal)
	}
	
	if optimal > numCPU {
		t.Errorf("Expected optimal GOMAXPROCS <= CPU count, got %d > %d", optimal, numCPU)
	}
}

func TestIsRealTimeCapable(t *testing.T) {
	capable := IsRealTimeCapable()
	
	// Should return a boolean without panic
	_ = capable
	
	// On major platforms, should generally be true
	switch runtime.GOOS {
	case "linux", "windows", "darwin":
		if !capable {
			t.Logf("Real-time capabilities reported as unavailable on %s", runtime.GOOS)
		}
	}
}

func TestGetSystemInfo(t *testing.T) {
	info := GetSystemInfo()
	
	expectedKeys := []string{
		"os", "arch", "go_version", "num_cpu", "gomaxprocs",
		"num_goroutine", "heap_sys_mb", "heap_alloc_mb",
		"pointer_size", "rt_capable", "optimal_threads",
	}
	
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("Expected system info to contain key %s", key)
		}
	}
	
	if info["os"] != runtime.GOOS {
		t.Errorf("Expected OS to be %s, got %v", runtime.GOOS, info["os"])
	}
	
	if info["arch"] != runtime.GOARCH {
		t.Errorf("Expected arch to be %s, got %v", runtime.GOARCH, info["arch"])
	}
	
	if info["num_cpu"] != runtime.NumCPU() {
		t.Errorf("Expected num_cpu to be %d, got %v", runtime.NumCPU(), info["num_cpu"])
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    OptimizationLevel
		expected string
	}{
		{OptimizationBasic, "basic"},
		{OptimizationStandard, "standard"},
		{OptimizationAggressive, "aggressive"},
		{OptimizationEnterprise, "enterprise"},
		{OptimizationTurbo, "turbo"},
	}
	
	logger := zaptest.NewLogger(t)
	
	for _, test := range tests {
		config := &SystemOptimizationConfig{Level: test.level}
		optimizer := NewSystemOptimizer(config, logger)
		
		result := optimizer.getLevelString()
		if result != test.expected {
			t.Errorf("Expected level string %s, got %s", test.expected, result)
		}
	}
}

func TestMemoryEstimation(t *testing.T) {
	estimate := estimateSystemMemory()
	
	if estimate <= 0 {
		t.Errorf("Expected positive memory estimate, got %d", estimate)
	}
	
	// Should be reasonable (at least 100MB)
	if estimate < 100*1024*1024 {
		t.Errorf("Memory estimate seems too low: %d bytes", estimate)
	}
}

func TestBToMb(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected uint64
	}{
		{0, 0},
		{1024 * 1024, 1},
		{1024 * 1024 * 1024, 1024},
		{1023, 0}, // Should round down
	}
	
	for _, test := range tests {
		result := bToMb(test.bytes)
		if result != test.expected {
			t.Errorf("Expected %d MB for %d bytes, got %d", test.expected, test.bytes, result)
		}
	}
}

func TestMonitoringLifecycle(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	config.Level = OptimizationStandard // Enable monitoring
	
	optimizer := NewSystemOptimizer(config, logger)
	
	err := optimizer.Apply()
	if err != nil {
		t.Fatalf("Failed to apply optimizations: %v", err)
	}
	
	// Let monitoring run briefly
	time.Sleep(100 * time.Millisecond)
	
	// Restore should stop monitoring
	err = optimizer.Restore()
	if err != nil {
		t.Errorf("Failed to restore optimizations: %v", err)
	}
	
	// Context should be cancelled
	select {
	case <-optimizer.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be cancelled after restore")
	}
}

func TestConcurrentOperations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	optimizer := NewSystemOptimizer(DefaultConfig(), logger)
	
	// Test concurrent GetStats calls
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			stats := optimizer.GetStats()
			if stats == nil {
				t.Error("Expected non-nil stats")
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestApplyOptimizationTypes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig()
	optimizer := NewSystemOptimizer(config, logger)
	
	// These should not panic
	err := optimizer.applyRuntimeTuning()
	if err != nil {
		t.Errorf("applyRuntimeTuning failed: %v", err)
	}
	
	err = optimizer.applyMemoryOptimizations()
	if err != nil {
		t.Errorf("applyMemoryOptimizations failed: %v", err)
	}
	
	err = optimizer.applyGCTuning()
	if err != nil {
		t.Errorf("applyGCTuning failed: %v", err)
	}
	
	err = optimizer.applyThreadOptimizations()
	if err != nil {
		t.Errorf("applyThreadOptimizations failed: %v", err)
	}
	
	err = optimizer.applyCPUOptimizations()
	if err != nil {
		t.Errorf("applyCPUOptimizations failed: %v", err)
	}
	
	err = optimizer.applyLatencyTuning()
	if err != nil {
		t.Errorf("applyLatencyTuning failed: %v", err)
	}
	
	err = optimizer.applyPlatformSpecific()
	if err != nil {
		t.Errorf("applyPlatformSpecific failed: %v", err)
	}
}

// Benchmark tests
func BenchmarkGetStats(b *testing.B) {
	logger := zap.NewNop()
	optimizer := NewSystemOptimizer(DefaultConfig(), logger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.GetStats()
	}
}

func BenchmarkTriggerGC(b *testing.B) {
	logger := zap.NewNop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TriggerGC(logger)
	}
}

func BenchmarkGetSystemInfo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetSystemInfo()
	}
}
