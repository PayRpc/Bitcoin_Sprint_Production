package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewRecorder(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(100, logger)

	assert.NotNil(t, recorder)
	assert.Equal(t, 100, recorder.maxEvents)
	assert.NotNil(t, recorder.events)
	assert.NotNil(t, recorder.stats)
	assert.True(t, recorder.isRunning)
}

func TestRecordEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	defer recorder.Close()

	ctx := context.Background()
	event := &DiagnosticEvent{
		EventType: "test_event",
		Message:   "Test message",
		Severity:  SeverityInfo,
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	err := recorder.RecordEvent(ctx, event)
	assert.NoError(t, err)

	// Check that event was recorded
	events, err := recorder.GetEvents(ctx, 10, SeverityDebug)
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "test_event", events[0].EventType)
	assert.Equal(t, "Test message", events[0].Message)
	assert.Equal(t, SeverityInfo, events[0].Severity)
	assert.NotZero(t, events[0].Timestamp)
}

func TestRecordEventNilEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	defer recorder.Close()

	ctx := context.Background()
	err := recorder.RecordEvent(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event cannot be nil")
}

func TestRecordEventAfterClose(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	recorder.Close()

	ctx := context.Background()
	event := &DiagnosticEvent{
		EventType: "test_event",
		Message:   "Test message",
		Severity:  SeverityInfo,
	}

	err := recorder.RecordEvent(ctx, event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recorder is not running")
}

func TestGetEvents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Record multiple events
	events := []*DiagnosticEvent{
		{
			EventType: "event1",
			Message:   "Message 1",
			Severity:  SeverityInfo,
		},
		{
			EventType: "event2",
			Message:   "Message 2",
			Severity:  SeverityWarning,
		},
		{
			EventType: "event3",
			Message:   "Message 3",
			Severity:  SeverityError,
		},
	}

	for _, event := range events {
		err := recorder.RecordEvent(ctx, event)
		require.NoError(t, err)
	}

	// Test getting all events
	allEvents, err := recorder.GetEvents(ctx, 10, SeverityDebug)
	assert.NoError(t, err)
	assert.Len(t, allEvents, 3)

	// Test filtering by severity
	warningAndAbove, err := recorder.GetEvents(ctx, 10, SeverityWarning)
	assert.NoError(t, err)
	assert.Len(t, warningAndAbove, 2) // Warning and Error

	// Test limit
	limited, err := recorder.GetEvents(ctx, 1, SeverityDebug)
	assert.NoError(t, err)
	assert.Len(t, limited, 1)
}

func TestGetStats(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Record some events
	events := []*DiagnosticEvent{
		{
			EventType: "connection",
			Message:   "Peer connected",
			Severity:  SeverityInfo,
		},
		{
			EventType: "message",
			Message:   "Message received",
			Severity:  SeverityDebug,
		},
		{
			EventType: "error",
			Message:   "Connection failed",
			Severity:  SeverityError,
		},
	}

	for _, event := range events {
		err := recorder.RecordEvent(ctx, event)
		require.NoError(t, err)
	}

	stats, err := recorder.GetStats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(3), stats.TotalEvents)
	assert.Equal(t, int64(1), stats.EventsByType["connection"])
	assert.Equal(t, int64(1), stats.EventsByType["message"])
	assert.Equal(t, int64(1), stats.EventsByType["error"])
	assert.Equal(t, int64(1), stats.EventsBySeverity[SeverityInfo])
	assert.Equal(t, int64(1), stats.EventsBySeverity[SeverityDebug])
	assert.Equal(t, int64(1), stats.EventsBySeverity[SeverityError])
	assert.NotNil(t, stats.FirstEvent)
	assert.NotNil(t, stats.LastEvent)
}

func TestClearEvents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Record some events
	event := &DiagnosticEvent{
		EventType: "test",
		Message:   "Test message",
		Severity:  SeverityInfo,
	}

	err := recorder.RecordEvent(ctx, event)
	require.NoError(t, err)

	// Verify event exists
	events, err := recorder.GetEvents(ctx, 10, SeverityDebug)
	assert.NoError(t, err)
	assert.Len(t, events, 1)

	// Clear events
	err = recorder.ClearEvents(ctx)
	assert.NoError(t, err)

	// Verify events are cleared
	events, err = recorder.GetEvents(ctx, 10, SeverityDebug)
	assert.NoError(t, err)
	assert.Len(t, events, 0)

	// Verify stats are reset
	stats, err := recorder.GetStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEvents)
}

func TestMaxEventsLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	maxEvents := 3
	recorder := NewRecorder(maxEvents, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Record more events than the limit
	for i := 0; i < maxEvents+2; i++ {
		event := &DiagnosticEvent{
			EventType: "test",
			Message:   fmt.Sprintf("Message %d", i),
			Severity:  SeverityInfo,
		}
		err := recorder.RecordEvent(ctx, event)
		require.NoError(t, err)
	}

	// Should only have maxEvents events
	events, err := recorder.GetEvents(ctx, 10, SeverityDebug)
	assert.NoError(t, err)
	assert.Len(t, events, maxEvents)

	// The oldest events should be removed
	assert.Equal(t, "Message 2", events[0].Message) // Should start from the 3rd event
}

func TestConcurrentAccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(100, logger)
	defer recorder.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 10
	eventsPerGoroutine := 100

	// Start multiple goroutines recording events
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := &DiagnosticEvent{
					EventType: "concurrent_test",
					Message:   fmt.Sprintf("Goroutine %d, Event %d", id, j),
					Severity:  SeverityInfo,
				}
				err := recorder.RecordEvent(ctx, event)
				assert.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all events were recorded
	events, err := recorder.GetEvents(ctx, numGoroutines*eventsPerGoroutine, SeverityDebug)
	assert.NoError(t, err)
	assert.Len(t, events, numGoroutines*eventsPerGoroutine)

	// Verify stats
	stats, err := recorder.GetStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(numGoroutines*eventsPerGoroutine), stats.TotalEvents)
}

func TestHelperMethods(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Test RecordPeerConnection
	err := recorder.RecordPeerConnection(ctx, "peer1", "127.0.0.1:8333")
	assert.NoError(t, err)

	// Test RecordPeerDisconnection
	err = recorder.RecordPeerDisconnection(ctx, "peer1", "127.0.0.1:8333", "timeout")
	assert.NoError(t, err)

	// Test RecordMessage
	err = recorder.RecordMessage(ctx, "peer1", "version", "outbound", 100)
	assert.NoError(t, err)

	// Test RecordError
	testErr := errors.New("test error")
	err = recorder.RecordError(ctx, "peer1", testErr, "handshake")
	assert.NoError(t, err)

	// Verify events were recorded
	events, err := recorder.GetEvents(ctx, 10, SeverityDebug)
	assert.NoError(t, err)
	assert.Len(t, events, 4)

	// Check event types
	eventTypes := make(map[string]bool)
	for _, event := range events {
		eventTypes[event.EventType] = true
	}

	assert.True(t, eventTypes["peer_connected"])
	assert.True(t, eventTypes["peer_disconnected"])
	assert.True(t, eventTypes["message"])
	assert.True(t, eventTypes["error"])
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityDebug, "DEBUG"},
		{SeverityInfo, "INFO"},
		{SeverityWarning, "WARNING"},
		{SeverityError, "ERROR"},
		{SeverityCritical, "CRITICAL"},
		{Severity(999), "UNKNOWN"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.severity.String())
	}
}

func TestCleanupRoutine(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	defer recorder.Close()

	ctx := context.Background()

	// Record an old event (simulate old timestamp)
	oldTime := time.Now().Add(-25 * time.Hour) // Older than cleanup threshold
	event := &DiagnosticEvent{
		EventType: "old_event",
		Message:   "Old message",
		Severity:  SeverityInfo,
		Timestamp: oldTime,
	}

	// Manually add the event to bypass timestamp setting
	recorder.eventsMu.Lock()
	recorder.events = append(recorder.events, event)
	recorder.eventsMu.Unlock()

	// Record a new event
	newEvent := &DiagnosticEvent{
		EventType: "new_event",
		Message:   "New message",
		Severity:  SeverityInfo,
	}

	err := recorder.RecordEvent(ctx, newEvent)
	require.NoError(t, err)

	// Wait a bit for cleanup to run (cleanup runs every hour, but we can trigger it manually)
	recorder.cleanupOldEvents()

	// The old event should be removed
	events, err := recorder.GetEvents(ctx, 10, SeverityDebug)
	assert.NoError(t, err)

	// Should only have the new event
	foundOld := false
	foundNew := false
	for _, e := range events {
		if e.EventType == "old_event" {
			foundOld = true
		}
		if e.EventType == "new_event" {
			foundNew = true
		}
	}

	assert.False(t, foundOld, "Old event should have been cleaned up")
	assert.True(t, foundNew, "New event should still exist")
}

func TestGetStatsAfterClose(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	recorder.Close()

	ctx := context.Background()
	_, err := recorder.GetStats(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recorder is not running")
}

func TestGetEventsAfterClose(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	recorder.Close()

	ctx := context.Background()
	_, err := recorder.GetEvents(ctx, 10, SeverityDebug)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recorder is not running")
}

func TestClearEventsAfterClose(t *testing.T) {
	logger := zaptest.NewLogger(t)
	recorder := NewRecorder(10, logger)
	recorder.Close()

	ctx := context.Background()
	err := recorder.ClearEvents(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "recorder is not running")
}
