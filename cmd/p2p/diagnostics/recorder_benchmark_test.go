package diagnostics

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func BenchmarkRecordEvent(b *testing.B) {
	logger := zaptest.NewLogger(b)
	recorder := NewRecorder(10000, logger)
	defer recorder.Close()

	ctx := context.Background()
	event := &DiagnosticEvent{
		EventType: "benchmark_test",
		Message:   "Benchmark message",
		Severity:  SeverityInfo,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			recorder.RecordEvent(ctx, event)
		}
	})
}

func BenchmarkGetEvents(b *testing.B) {
	logger := zaptest.NewLogger(b)
	recorder := NewRecorder(10000, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Pre-populate with events
	for i := 0; i < 1000; i++ {
		event := &DiagnosticEvent{
			EventType: "benchmark_test",
			Message:   "Benchmark message",
			Severity:  SeverityInfo,
		}
		recorder.RecordEvent(ctx, event)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder.GetEvents(ctx, 100, SeverityDebug)
	}
}

func BenchmarkGetStats(b *testing.B) {
	logger := zaptest.NewLogger(b)
	recorder := NewRecorder(10000, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Pre-populate with events
	for i := 0; i < 1000; i++ {
		event := &DiagnosticEvent{
			EventType: "benchmark_test",
			Message:   "Benchmark message",
			Severity:  SeverityInfo,
		}
		recorder.RecordEvent(ctx, event)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder.GetStats(ctx)
	}
}

func BenchmarkConcurrentOperations(b *testing.B) {
	logger := zaptest.NewLogger(b)
	recorder := NewRecorder(10000, logger)
	defer recorder.Close()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of different operations
			event := &DiagnosticEvent{
				EventType: "concurrent_test",
				Message:   "Concurrent benchmark message",
				Severity:  SeverityInfo,
			}
			recorder.RecordEvent(ctx, event)
			recorder.GetEvents(ctx, 10, SeverityDebug)
			recorder.GetStats(ctx)
		}
	})
}
