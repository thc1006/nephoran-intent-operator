package audit

import (
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

// AuditConfig is an alias for controller compatibility
type AuditConfig = AuditSystemConfig

// StorageConfig defines storage backend configuration
type StorageConfig struct {
	Type          string                 `json:"type"` // "elasticsearch", "s3", "local"
	Elasticsearch *ElasticsearchConfig   `json:"elasticsearch,omitempty"`
	S3            *S3Config              `json:"s3,omitempty"`
	Local         *LocalStorageConfig    `json:"local,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// ElasticsearchConfig defines Elasticsearch storage settings
type ElasticsearchConfig struct {
	URL      string `json:"url"`
	Index    string `json:"index"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	TLS      bool   `json:"tls,omitempty"`
}

// S3Config defines S3 storage settings
type S3Config struct {
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
}

// LocalStorageConfig defines local file storage settings
type LocalStorageConfig struct {
	Path       string `json:"path"`
	MaxSize    int64  `json:"max_size,omitempty"`
	Rotate     bool   `json:"rotate,omitempty"`
	Compress   bool   `json:"compress,omitempty"`
	Retentions int    `json:"retentions,omitempty"`
}

// Event is an alias for controller compatibility
type Event = types.AuditEvent

// Metrics represents audit system metrics
type Metrics struct {
	EventsProcessed   int64     `json:"events_processed"`
	EventsDropped     int64     `json:"events_dropped"`
	LastEventTime     time.Time `json:"last_event_time"`
	StorageUsed       int64     `json:"storage_used"`
	TotalEvents       int64     `json:"total_events"`
	ErrorCount        int64     `json:"error_count"`
	BackendsHealthy   int       `json:"backends_healthy"`
	BackendsTotal     int       `json:"backends_total"`
	QueueSize         int       `json:"queue_size"`
	ProcessingLatency float64   `json:"processing_latency_ms"`
}

// SearchCriteria defines search parameters for audit events
type SearchCriteria struct {
	// Time range
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`

	// Event filtering
	EventTypes []types.EventType `json:"event_types,omitempty"`
	Severities []types.Severity  `json:"severities,omitempty"`
	Categories []string          `json:"categories,omitempty"`

	// User and session filtering
	UserIDs      []string `json:"user_ids,omitempty"`
	SessionIDs   []string `json:"session_ids,omitempty"`
	ServiceNames []string `json:"service_names,omitempty"`

	// Resource filtering
	ResourceTypes []string `json:"resource_types,omitempty"`
	ResourceIDs   []string `json:"resource_ids,omitempty"`
	Namespaces    []string `json:"namespaces,omitempty"`

	// Network filtering
	SourceIPs  []string `json:"source_ips,omitempty"`
	UserAgents []string `json:"user_agents,omitempty"`

	// Content filtering
	TextSearch string                 `json:"text_search,omitempty"`
	Filters    map[string]interface{} `json:"filters,omitempty"`

	// Result options
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}
