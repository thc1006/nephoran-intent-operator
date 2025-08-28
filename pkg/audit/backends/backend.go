package backends

import (
	"context"
	"time"

	audittypes "github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

// Backend represents an audit log destination
type Backend interface {
	// Type returns the backend type identifier
	Type() string

	// Initialize sets up the backend with configuration
	Initialize(config BackendConfig) error

	// WriteEvent writes a single audit event
	WriteEvent(ctx context.Context, event *audittypes.AuditEvent) error

	// WriteEvents writes multiple audit events in a batch
	WriteEvents(ctx context.Context, events []*audittypes.AuditEvent) error

	// Query searches for audit events (optional for compliance reporting)
	Query(ctx context.Context, query *QueryRequest) (*QueryResponse, error)

	// Health checks the backend connectivity and status
	Health(ctx context.Context) error

	// Close gracefully shuts down the backend
	Close() error
}

// BackendType represents the type of audit backend
type BackendType string

const (
	BackendTypeFile          BackendType = "file"
	BackendTypeElasticsearch BackendType = "elasticsearch"
	BackendTypeSplunk        BackendType = "splunk"
	BackendTypeSyslog        BackendType = "syslog"
	BackendTypeKafka         BackendType = "kafka"
	BackendTypeCloudWatch    BackendType = "cloudwatch"
)

// BackendConfig represents configuration for audit backends
type BackendConfig struct {
	Name     string                 `yaml:"name" json:"name"`
	Type     BackendType            `yaml:"type" json:"type"`
	Enabled  bool                   `yaml:"enabled" json:"enabled"`
	Settings map[string]interface{} `yaml:"settings" json:"settings"`

	// Common settings
	BatchSize     int           `yaml:"batch_size" json:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval" json:"flush_interval"`
	BufferSize    int           `yaml:"buffer_size" json:"buffer_size"`

	// Filtering
	Filter FilterConfig `yaml:"filter" json:"filter"`

	// Retry configuration
	RetryConfig RetryConfig `yaml:"retry" json:"retry"`

	// Encryption (for sensitive backends)
	Encryption EncryptionConfig `yaml:"encryption" json:"encryption"`
}

// FilterConfig defines event filtering rules
type FilterConfig struct {
	// Event types to include/exclude
	IncludeEventTypes []string `yaml:"include_event_types" json:"include_event_types"`
	ExcludeEventTypes []string `yaml:"exclude_event_types" json:"exclude_event_types"`

	// Severity levels to include/exclude
	MinSeverity string `yaml:"min_severity" json:"min_severity"`
	MaxSeverity string `yaml:"max_severity" json:"max_severity"`

	// User-based filtering
	IncludeUsers []string `yaml:"include_users" json:"include_users"`
	ExcludeUsers []string `yaml:"exclude_users" json:"exclude_users"`

	// Resource-based filtering
	IncludeResources []string `yaml:"include_resources" json:"include_resources"`
	ExcludeResources []string `yaml:"exclude_resources" json:"exclude_resources"`

	// Custom field filters
	FieldFilters map[string]interface{} `yaml:"field_filters" json:"field_filters"`

	// Compliance-based filtering
	ComplianceOnly bool     `yaml:"compliance_only" json:"compliance_only"`
	Standards      []string `yaml:"standards" json:"standards"`
}

// RetryConfig defines retry behavior for failed writes
type RetryConfig struct {
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	InitialDelay  time.Duration `yaml:"initial_delay" json:"initial_delay"`
	MaxDelay      time.Duration `yaml:"max_delay" json:"max_delay"`
	BackoffFactor float64       `yaml:"backoff_factor" json:"backoff_factor"`
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Algorithm  string `yaml:"algorithm" json:"algorithm"`
	KeyFile    string `yaml:"key_file" json:"key_file"`
	KeyRotation struct {
		Enabled  bool          `yaml:"enabled" json:"enabled"`
		Interval time.Duration `yaml:"interval" json:"interval"`
	} `yaml:"key_rotation" json:"key_rotation"`
}

// QueryRequest represents a query for audit events
type QueryRequest struct {
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	EventTypes   []string               `json:"event_types,omitempty"`
	UserIDs      []string               `json:"user_ids,omitempty"`
	ResourceIDs  []string               `json:"resource_ids,omitempty"`
	Severity     []string               `json:"severity,omitempty"`
	Limit        int                    `json:"limit,omitempty"`
	Offset       int                    `json:"offset,omitempty"`
	Fields       []string               `json:"fields,omitempty"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
	Query        string                 `json:"query,omitempty"`
	SortBy       string                 `json:"sort_by,omitempty"`
	SortOrder    string                 `json:"sort_order,omitempty"`
}

// QueryResponse represents the response to a query
type QueryResponse struct {
	Events       []*audittypes.AuditEvent    `json:"events"`
	Total        int64                       `json:"total"`
	Took         time.Duration               `json:"took"`
	Aggregations map[string]interface{}      `json:"aggregations,omitempty"`
	ScrollID     string                      `json:"scroll_id,omitempty"`
	NextToken    string                      `json:"next_token,omitempty"`
	TotalCount   int                         `json:"total_count"` // For Splunk compatibility
	HasMore      bool                        `json:"has_more"`    // For Splunk compatibility
}

// BackendStats represents statistics for a backend
type BackendStats struct {
	EventsWritten      int64         `json:"events_written"`
	EventsFailed       int64         `json:"events_failed"`
	BatchesProcessed   int64         `json:"batches_processed"`
	AverageWriteTime   time.Duration `json:"average_write_time"`
	LastWriteTime      time.Time     `json:"last_write_time"`
	ConnectionStatus   string        `json:"connection_status"`
	BufferUtilization  float64       `json:"buffer_utilization"`
	RetryAttempts      int64         `json:"retry_attempts"`
	SuccessfulRetries  int64         `json:"successful_retries"`
	HealthCheckStatus  string        `json:"health_check_status"`
	LastHealthCheck    time.Time     `json:"last_health_check"`
}

// BaseBackend provides common functionality for all backends
type BaseBackend struct {
	config BackendConfig
	stats  BackendStats
}

// NewBaseBackend creates a new base backend
func NewBaseBackend(config BackendConfig) *BaseBackend {
	return &BaseBackend{
		config: config,
		stats: BackendStats{
			ConnectionStatus:  "disconnected",
			HealthCheckStatus: "unknown",
		},
	}
}

// GetStats returns backend statistics
func (b *BaseBackend) GetStats() BackendStats {
	return b.stats
}

// UpdateStats updates backend statistics
func (b *BaseBackend) UpdateStats(eventsWritten, eventsFailed int64, writeTime time.Duration) {
	b.stats.EventsWritten += eventsWritten
	b.stats.EventsFailed += eventsFailed
	
	if writeTime > 0 {
		// Calculate running average
		if b.stats.AverageWriteTime == 0 {
			b.stats.AverageWriteTime = writeTime
		} else {
			b.stats.AverageWriteTime = (b.stats.AverageWriteTime + writeTime) / 2
		}
	}
	
	b.stats.LastWriteTime = time.Now()
}

// ShouldProcessEvent checks if an event should be processed based on filters
func (f *FilterConfig) ShouldProcessEvent(event *audittypes.AuditEvent) bool {
	// Check event type filters
	if len(f.IncludeEventTypes) > 0 {
		found := false
		for _, eventType := range f.IncludeEventTypes {
			if string(event.EventType) == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.ExcludeEventTypes) > 0 {
		for _, eventType := range f.ExcludeEventTypes {
			if string(event.EventType) == eventType {
				return false
			}
		}
	}

	// Check user filters
	if len(f.IncludeUsers) > 0 {
		found := false
		userID := getUserID(event)
		for _, user := range f.IncludeUsers {
			if userID == user {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.ExcludeUsers) > 0 {
		userID := getUserID(event)
		for _, user := range f.ExcludeUsers {
			if userID == user {
				return false
			}
		}
	}

	return true
}

// ApplyFieldFilters applies field-level filters to an event
func (f *FilterConfig) ApplyFieldFilters(event *audittypes.AuditEvent) *audittypes.AuditEvent {
	if len(f.FieldFilters) == 0 {
		return event
	}

	// Create a copy of the event to avoid modifying the original
	filteredEvent := *event

	// Apply field filters based on configuration
	// This is a simplified implementation - in practice, you'd want more sophisticated filtering
	for field, filter := range f.FieldFilters {
		switch field {
		case "metadata":
			// Filter metadata fields
			if filteredEvent.Data == nil {
				continue
			}
			
			if filterMap, ok := filter.(map[string]interface{}); ok {
				filteredMetadata := make(map[string]interface{})
				for key, value := range filteredEvent.Data {
					if allowedValue, exists := filterMap[key]; exists {
						if allowedValue == value || allowedValue == "*" {
							filteredMetadata[key] = value
						}
					}
				}
				filteredEvent.Data = filteredMetadata
			}
		}
	}

	return &filteredEvent
}

// DefaultBackendConfig returns a default backend configuration
func DefaultBackendConfig() BackendConfig {
	return BackendConfig{
		Name:          "default",
		Type:          BackendTypeFile,
		Enabled:       true,
		BatchSize:     100,
		FlushInterval: 30 * time.Second,
		BufferSize:    1000,
		Settings:      make(map[string]interface{}),
		RetryConfig: RetryConfig{
			MaxRetries:    3,
			InitialDelay:  time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
		},
		Filter: FilterConfig{
			ComplianceOnly: false,
		},
	}
}

// Helper functions

// getUserID extracts the UserID from the event's UserContext
func getUserID(event *audittypes.AuditEvent) string {
	if event.UserContext != nil {
		return event.UserContext.UserID
	}
	return ""
}