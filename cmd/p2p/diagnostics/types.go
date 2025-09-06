package diagnostics

import (
	"context"
	"time"
)

// DiagnosticEvent represents a diagnostic event
type DiagnosticEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	PeerID    string                 `json:"peer_id,omitempty"`
	Message   string                 `json:"message"`
	Severity  Severity               `json:"severity"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Error     error                  `json:"error,omitempty"`
}

// Severity represents the severity level of a diagnostic event
type Severity int

const (
	SeverityDebug Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
	SeverityCritical
)

// String returns the string representation of severity
func (s Severity) String() string {
	switch s {
	case SeverityDebug:
		return "DEBUG"
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// DiagnosticRecorder defines the interface for recording diagnostic events
type DiagnosticRecorder interface {
	RecordEvent(ctx context.Context, event *DiagnosticEvent) error
	GetEvents(ctx context.Context, limit int, minSeverity Severity) ([]*DiagnosticEvent, error)
	GetStats(ctx context.Context) (*DiagnosticStats, error)
	ClearEvents(ctx context.Context) error
	Close() error
}

// DiagnosticStats represents diagnostic statistics
type DiagnosticStats struct {
	TotalEvents      int64              `json:"total_events"`
	EventsByType     map[string]int64   `json:"events_by_type"`
	EventsBySeverity map[Severity]int64 `json:"events_by_severity"`
	FirstEvent       *time.Time         `json:"first_event,omitempty"`
	LastEvent        *time.Time         `json:"last_event,omitempty"`
	ActivePeers      int                `json:"active_peers"`
	ErrorRate        float64            `json:"error_rate"`
}

// PeerDiagnostic represents diagnostic information for a peer
type PeerDiagnostic struct {
	PeerID           string        `json:"peer_id"`
	Address          string        `json:"address"`
	ConnectedAt      time.Time     `json:"connected_at"`
	LastActivity     time.Time     `json:"last_activity"`
	MessagesSent     int64         `json:"messages_sent"`
	MessagesReceived int64         `json:"messages_received"`
	BytesSent        int64         `json:"bytes_sent"`
	BytesReceived    int64         `json:"bytes_received"`
	Latency          time.Duration `json:"latency"`
	ErrorCount       int64         `json:"error_count"`
	Status           string        `json:"status"`
}

// NetworkDiagnostic represents network-level diagnostic information
type NetworkDiagnostic struct {
	TotalPeers        int               `json:"total_peers"`
	ActivePeers       int               `json:"active_peers"`
	DisconnectedPeers int               `json:"disconnected_peers"`
	NetworkLatency    time.Duration     `json:"network_latency"`
	BlockHeight       int64             `json:"block_height"`
	NetworkHashrate   int64             `json:"network_hashrate"`
	PeerDiagnostics   []*PeerDiagnostic `json:"peer_diagnostics"`
}

// RecorderConfig holds configuration for the diagnostic recorder
type RecorderConfig struct {
	MaxEvents         int           `json:"max_events"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	EventRetention    time.Duration `json:"event_retention"`
	EnableBackground  bool          `json:"enable_background"`
	LogAllEvents      bool          `json:"log_all_events"`
	SeverityThreshold Severity      `json:"severity_threshold"`
}

// DefaultRecorderConfig returns a default configuration
func DefaultRecorderConfig() *RecorderConfig {
	return &RecorderConfig{
		MaxEvents:         10000,
		CleanupInterval:   time.Hour,
		EventRetention:    24 * time.Hour,
		EnableBackground:  true,
		LogAllEvents:      false,
		SeverityThreshold: SeverityDebug,
	}
}

// HealthStatus represents the health status of the recorder
type HealthStatus struct {
	Status      string        `json:"status"`
	Message     string        `json:"message"`
	EventCount  int           `json:"event_count"`
	MaxEvents   int           `json:"max_events"`
	TotalEvents int64         `json:"total_events"`
	Uptime      time.Duration `json:"uptime"`
}
