package diagnostics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Recorder implements the DiagnosticRecorder interface
type Recorder struct {
	events     []*DiagnosticEvent
	eventsMu   sync.RWMutex
	maxEvents  int
	logger     *zap.Logger
	stats      *DiagnosticStats
	statsMu    sync.RWMutex
	isRunning  bool
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewRecorder creates a new diagnostic recorder
func NewRecorder(maxEvents int, logger *zap.Logger) *Recorder {
	if maxEvents <= 0 {
		maxEvents = 10000 // Default max events
	}

	recorder := &Recorder{
		events:    make([]*DiagnosticEvent, 0, maxEvents),
		maxEvents: maxEvents,
		logger:    logger,
		stats: &DiagnosticStats{
			EventsByType:     make(map[string]int64),
			EventsBySeverity: make(map[Severity]int64),
		},
		stopChan: make(chan struct{}),
		isRunning: true,
	}

	// Start background cleanup routine
	recorder.wg.Add(1)
	go recorder.cleanupRoutine()

	return recorder
}

// RecordEvent records a diagnostic event
func (r *Recorder) RecordEvent(ctx context.Context, event *DiagnosticEvent) error {
	if !r.isRunning {
		return fmt.Errorf("recorder is not running")
	}

	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	// Add event to the list
	r.events = append(r.events, event)

	// Maintain max events limit (remove oldest)
	if len(r.events) > r.maxEvents {
		r.events = r.events[1:]
	}

	// Update statistics
	r.updateStats(event)

	// Log the event
	r.logEvent(event)

	return nil
}

// GetEvents returns diagnostic events with optional filtering
func (r *Recorder) GetEvents(ctx context.Context, limit int, minSeverity Severity) ([]*DiagnosticEvent, error) {
	if !r.isRunning {
		return nil, fmt.Errorf("recorder is not running")
	}

	r.eventsMu.RLock()
	defer r.eventsMu.RUnlock()

	if limit <= 0 {
		limit = len(r.events)
	}

	var filteredEvents []*DiagnosticEvent
	for i := len(r.events) - 1; i >= 0 && len(filteredEvents) < limit; i-- {
		event := r.events[i]
		if event.Severity >= minSeverity {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Reverse to maintain chronological order (newest first)
	for i, j := 0, len(filteredEvents)-1; i < j; i, j = i+1, j-1 {
		filteredEvents[i], filteredEvents[j] = filteredEvents[j], filteredEvents[i]
	}

	return filteredEvents, nil
}

// GetStats returns current diagnostic statistics
func (r *Recorder) GetStats(ctx context.Context) (*DiagnosticStats, error) {
	if !r.isRunning {
		return nil, fmt.Errorf("recorder is not running")
	}

	r.statsMu.RLock()
	defer r.statsMu.RUnlock()

	// Create a copy to avoid race conditions
	stats := &DiagnosticStats{
		TotalEvents:      r.stats.TotalEvents,
		EventsByType:     make(map[string]int64),
		EventsBySeverity: make(map[Severity]int64),
		ActivePeers:      r.stats.ActivePeers,
		ErrorRate:        r.stats.ErrorRate,
	}

	for k, v := range r.stats.EventsByType {
		stats.EventsByType[k] = v
	}

	for k, v := range r.stats.EventsBySeverity {
		stats.EventsBySeverity[k] = v
	}

	if r.stats.FirstEvent != nil {
		firstEvent := *r.stats.FirstEvent
		stats.FirstEvent = &firstEvent
	}

	if r.stats.LastEvent != nil {
		lastEvent := *r.stats.LastEvent
		stats.LastEvent = &lastEvent
	}

	return stats, nil
}

// ClearEvents clears all recorded events
func (r *Recorder) ClearEvents(ctx context.Context) error {
	if !r.isRunning {
		return fmt.Errorf("recorder is not running")
	}

	r.eventsMu.Lock()
	r.events = r.events[:0]
	r.eventsMu.Unlock()

	r.statsMu.Lock()
	r.stats = &DiagnosticStats{
		EventsByType:     make(map[string]int64),
		EventsBySeverity: make(map[Severity]int64),
	}
	r.statsMu.Unlock()

	r.logger.Info("Cleared all diagnostic events")
	return nil
}

// Close shuts down the recorder
func (r *Recorder) Close() error {
	if !r.isRunning {
		return nil
	}

	r.isRunning = false
	close(r.stopChan)
	r.wg.Wait()

	r.logger.Info("Diagnostic recorder closed")
	return nil
}

// updateStats updates the internal statistics
func (r *Recorder) updateStats(event *DiagnosticEvent) {
	r.statsMu.Lock()
	defer r.statsMu.Unlock()

	r.stats.TotalEvents++
	r.stats.EventsByType[event.EventType]++
	r.stats.EventsBySeverity[event.Severity]++

	now := time.Now()
	if r.stats.FirstEvent == nil {
		r.stats.FirstEvent = &now
	}
	r.stats.LastEvent = &now

	// Calculate error rate (simplified)
	totalErrors := r.stats.EventsBySeverity[SeverityError] + r.stats.EventsBySeverity[SeverityCritical]
	if r.stats.TotalEvents > 0 {
		r.stats.ErrorRate = float64(totalErrors) / float64(r.stats.TotalEvents)
	}
}

// logEvent logs the event based on its severity
func (r *Recorder) logEvent(event *DiagnosticEvent) {
	fields := []zap.Field{
		zap.String("event_type", event.EventType),
		zap.String("severity", event.Severity.String()),
		zap.Time("timestamp", event.Timestamp),
	}

	if event.PeerID != "" {
		fields = append(fields, zap.String("peer_id", event.PeerID))
	}

	if len(event.Metadata) > 0 {
		for k, v := range event.Metadata {
			fields = append(fields, zap.Any(k, v))
		}
	}

	if event.Error != nil {
		fields = append(fields, zap.Error(event.Error))
	}

	switch event.Severity {
	case SeverityDebug:
		r.logger.Debug(event.Message, fields...)
	case SeverityInfo:
		r.logger.Info(event.Message, fields...)
	case SeverityWarning:
		r.logger.Warn(event.Message, fields...)
	case SeverityError:
		r.logger.Error(event.Message, fields...)
	case SeverityCritical:
		r.logger.Fatal(event.Message, fields...)
	}
}

// cleanupRoutine periodically cleans up old events
func (r *Recorder) cleanupRoutine() {
	defer r.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // Clean up every hour
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.cleanupOldEvents()
		}
	}
}

// cleanupOldEvents removes events older than 24 hours
func (r *Recorder) cleanupOldEvents() {
	r.eventsMu.Lock()
	defer r.eventsMu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	var newEvents []*DiagnosticEvent

	for _, event := range r.events {
		if event.Timestamp.After(cutoff) {
			newEvents = append(newEvents, event)
		}
	}

	removed := len(r.events) - len(newEvents)
	r.events = newEvents

	if removed > 0 {
		r.logger.Info("Cleaned up old diagnostic events",
			zap.Int("removed", removed),
			zap.Int("remaining", len(r.events)))
	}
}

// Helper functions for common event types

// RecordPeerConnection records a peer connection event
func (r *Recorder) RecordPeerConnection(ctx context.Context, peerID, address string) error {
	return r.RecordEvent(ctx, &DiagnosticEvent{
		EventType: "peer_connected",
		PeerID:    peerID,
		Message:   fmt.Sprintf("Peer connected: %s", address),
		Severity:  SeverityInfo,
		Metadata: map[string]interface{}{
			"address": address,
		},
	})
}

// RecordPeerDisconnection records a peer disconnection event
func (r *Recorder) RecordPeerDisconnection(ctx context.Context, peerID, address string, reason string) error {
	return r.RecordEvent(ctx, &DiagnosticEvent{
		EventType: "peer_disconnected",
		PeerID:    peerID,
		Message:   fmt.Sprintf("Peer disconnected: %s (%s)", address, reason),
		Severity:  SeverityWarning,
		Metadata: map[string]interface{}{
			"address": address,
			"reason":  reason,
		},
	})
}

// RecordMessage records a message event
func (r *Recorder) RecordMessage(ctx context.Context, peerID, messageType string, direction string, size int) error {
	return r.RecordEvent(ctx, &DiagnosticEvent{
		EventType: "message",
		PeerID:    peerID,
		Message:   fmt.Sprintf("Message %s: %s (%d bytes)", direction, messageType, size),
		Severity:  SeverityDebug,
		Metadata: map[string]interface{}{
			"message_type": messageType,
			"direction":    direction,
			"size":         size,
		},
	})
}

// RecordError records an error event
func (r *Recorder) RecordError(ctx context.Context, peerID string, err error, operation string) error {
	severity := SeverityError
	if peerID == "" {
		severity = SeverityCritical // System-level errors are critical
	}

	return r.RecordEvent(ctx, &DiagnosticEvent{
		EventType: "error",
		PeerID:    peerID,
		Message:   fmt.Sprintf("Error in %s", operation),
		Severity:  severity,
		Error:     err,
		Metadata: map[string]interface{}{
			"operation": operation,
		},
	})
}

// ExportEvents exports all events in JSON format
func (r *Recorder) ExportEvents(ctx context.Context) ([]byte, error) {
	if !r.isRunning {
		return nil, fmt.Errorf("recorder is not running")
	}

	r.eventsMu.RLock()
	defer r.eventsMu.RUnlock()

	// Simple JSON export - in production you might want to use a proper JSON library
	events := make([]*DiagnosticEvent, len(r.events))
	copy(events, r.events)

	// Convert to JSON (simplified implementation)
	jsonData := "["
	for i, event := range events {
		if i > 0 {
			jsonData += ","
		}
		jsonData += fmt.Sprintf(`{"timestamp":"%s","event_type":"%s","peer_id":"%s","message":"%s","severity":"%s"}`,
			event.Timestamp.Format(time.RFC3339),
			event.EventType,
			event.PeerID,
			event.Message,
			event.Severity.String())
	}
	jsonData += "]"

	return []byte(jsonData), nil
}

// HealthCheck performs a health check on the recorder
func (r *Recorder) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if !r.isRunning {
		return &HealthStatus{
			Status:  "stopped",
			Message: "Recorder is not running",
		}, nil
	}

	r.eventsMu.RLock()
	eventCount := len(r.events)
	r.eventsMu.RUnlock()

	r.statsMu.RLock()
	totalEvents := r.stats.TotalEvents
	r.statsMu.RUnlock()

	status := "healthy"
	message := "Recorder is operating normally"

	// Check for potential issues
	if eventCount >= r.maxEvents {
		status = "warning"
		message = "Event buffer is at maximum capacity"
	}

	if totalEvents == 0 && eventCount > 0 {
		status = "warning"
		message = "Events recorded but statistics not updated"
	}

	return &HealthStatus{
		Status:      status,
		Message:     message,
		EventCount:  eventCount,
		MaxEvents:   r.maxEvents,
		TotalEvents: totalEvents,
		Uptime:      time.Since(time.Now().Add(-time.Hour)), // Simplified uptime calculation
	}, nil
}