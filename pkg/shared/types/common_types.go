package types

import (
	"time"
)

// CacheEntry represents a unified cached item with comprehensive metadata
// This type consolidates all CacheEntry definitions across the codebase
type CacheEntry struct {
	// Core fields
	Key           string      `json:"key"`
	Value         interface{} `json:"value"`
	Response      string      `json:"response,omitempty"` // For LLM responses
	OriginalValue interface{} `json:"-"`                  // Keep reference to avoid re-serialization

	// Timestamps
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastAccess   time.Time `json:"last_access"`
	LastAccessed time.Time `json:"last_accessed"`
	LastModified time.Time `json:"last_modified"`

	// Access patterns and metrics  
	AccessCount     int64         `json:"access_count"`
	HitCount        int64         `json:"hit_count"`
	AccessFrequency float64       `json:"access_frequency"`
	AccessPattern   interface{}   `json:"access_pattern,omitempty"` // Can be different types

	// Size and performance
	Size            int64         `json:"size"`
	TTL             time.Duration `json:"ttl"`
	Level           int           `json:"level"` // 1 for L1, 2 for L2
	Compressed      bool          `json:"compressed"`

	// Semantic and similarity
	Keywords        []string `json:"keywords,omitempty"`
	SimilarityScore float64  `json:"similarity_score"`

	// Generic metadata for extension
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Document represents a unified document type for RAG and performance testing
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Title    string                 `json:"title,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Vector   []float32              `json:"vector,omitempty"`
	
	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Classification
	Category string   `json:"category,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	
	// Performance metrics
	ProcessingTime time.Duration `json:"processing_time,omitempty"`
	Size           int64         `json:"size,omitempty"`
}

// BatchProcessor interface for processing operations in batches
type BatchProcessorInterface interface {
	ProcessBatch(items []interface{}) error
	GetStats() *BatchProcessorStats
	Start() error
	Stop() error
}

// BatchProcessorStats represents statistics for batch processing operations
type BatchProcessorStats struct {
	ProcessedBatches    int64         `json:"processed_batches"`
	TotalRequests       int64         `json:"total_requests"`
	ActiveBatches       int64         `json:"active_batches"`
	AverageBatchSize    float64       `json:"average_batch_size"`
	AverageProcessTime  time.Duration `json:"average_process_time"`
	QueueLength         int64         `json:"queue_length"`
	PriorityQueueLength int64         `json:"priority_queue_length,omitempty"`
	WorkerCount         int64         `json:"worker_count"`
	SuccessRate         float64       `json:"success_rate"`
	ErrorCount          int64         `json:"error_count"`
}

// PerformanceValidator represents a validator for performance metrics
type PerformanceValidator struct {
	Name        string                 `json:"name"`
	Thresholds  map[string]float64     `json:"thresholds"`
	Metrics     map[string]float64     `json:"metrics"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
	LastRun     time.Time             `json:"last_run"`
	Results     []ValidationResult    `json:"results"`
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Passed    bool      `json:"passed"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// ComparisonResult represents the result of comparing performance metrics
type ComparisonResult struct {
	Metric         string    `json:"metric"`
	BaseValue      float64   `json:"base_value"`
	CompareValue   float64   `json:"compare_value"`
	Difference     float64   `json:"difference"`
	PercentChange  float64   `json:"percent_change"`
	Improved       bool      `json:"improved"`
	Timestamp      time.Time `json:"timestamp"`
	Category       string    `json:"category,omitempty"`
	Significance   string    `json:"significance,omitempty"` // "low", "medium", "high"
}

// Bottleneck represents a performance bottleneck detected in the system
type Bottleneck struct {
	ID          string                 `json:"id"`
	Component   string                 `json:"component"`
	Type        string                 `json:"type"` // "cpu", "memory", "io", "network", "database"
	Severity    string                 `json:"severity"` // "low", "medium", "high", "critical"
	Description string                 `json:"description"`
	
	// Metrics
	Impact      float64               `json:"impact"` // 0.0 to 1.0
	Duration    time.Duration         `json:"duration"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     time.Time             `json:"end_time,omitempty"`
	
	// Context
	Context     map[string]interface{} `json:"context,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	
	// Resolution
	Resolved    bool      `json:"resolved"`
	ResolvedAt  time.Time `json:"resolved_at,omitempty"`
	Resolution  string    `json:"resolution,omitempty"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Alloc         uint64        `json:"alloc"`          // bytes allocated and still in use
	TotalAlloc    uint64        `json:"total_alloc"`    // bytes allocated (even if freed)
	Sys           uint64        `json:"sys"`            // bytes obtained from system
	Lookups       uint64        `json:"lookups"`        // number of pointer lookups
	Mallocs       uint64        `json:"mallocs"`        // number of mallocs
	Frees         uint64        `json:"frees"`          // number of frees
	
	// Heap stats
	HeapAlloc    uint64 `json:"heap_alloc"`    // bytes allocated and still in use
	HeapSys      uint64 `json:"heap_sys"`      // bytes obtained from system
	HeapIdle     uint64 `json:"heap_idle"`     // bytes in idle spans
	HeapInuse    uint64 `json:"heap_inuse"`    // bytes in non-idle span
	HeapReleased uint64 `json:"heap_released"` // bytes released to the OS
	HeapObjects  uint64 `json:"heap_objects"`  // total number of allocated objects
	
	// GC stats
	NextGC       uint64        `json:"next_gc"`        // next collection will happen when HeapAlloc ≥ this amount
	LastGC       uint64        `json:"last_gc"`        // end time of last collection (nanoseconds since 1970)
	PauseTotalNs uint64        `json:"pause_total_ns"` // cumulative nanoseconds in GC stop-the-world pauses
	NumGC        uint32        `json:"num_gc"`         // number of completed GC cycles
	NumForcedGC  uint32        `json:"num_forced_gc"`  // number of GC cycles that were forced
	
	// Timing
	Timestamp    time.Time     `json:"timestamp"`
	GCCPUPercent float64       `json:"gc_cpu_percent"` // percentage of CPU time used by GC
}

// ProfileReport represents a profiling report with performance data
type ProfileReport struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "cpu", "memory", "goroutine", "block", "mutex"
	Duration    time.Duration          `json:"duration"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	
	// Stats
	SampleCount int64                  `json:"sample_count"`
	TotalCost   int64                  `json:"total_cost"`
	
	// Top functions/hotspots
	HotSpots    []HotSpot              `json:"hotspots"`
	
	// Summary metrics
	Metrics     map[string]interface{} `json:"metrics"`
	
	// File paths
	ProfilePath string                 `json:"profile_path,omitempty"`
	ReportPath  string                 `json:"report_path,omitempty"`
	
	// Analysis
	Summary     string                 `json:"summary,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
}

// HotSpot represents a performance hotspot in profiling data
type HotSpot struct {
	Function     string        `json:"function"`
	File         string        `json:"file"`
	Line         int           `json:"line"`
	
	// Costs
	SelfCost     int64         `json:"self_cost"`      // exclusive cost
	TotalCost    int64         `json:"total_cost"`     // inclusive cost
	SelfPercent  float64       `json:"self_percent"`   // exclusive percentage
	TotalPercent float64       `json:"total_percent"`  // inclusive percentage
	
	// Counts
	SampleCount  int64         `json:"sample_count"`
	CallCount    int64         `json:"call_count,omitempty"`
	
	// Context
	Package      string        `json:"package,omitempty"`
	Module       string        `json:"module,omitempty"`
	
	// Analysis
	Category     string        `json:"category,omitempty"` // "cpu", "memory", "io", "lock"
	Impact       string        `json:"impact,omitempty"`   // "low", "medium", "high"
	Suggestion   string        `json:"suggestion,omitempty"`
}