package sla

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// CircularBuffer represents a circular buffer for storing historical data.

type CircularBuffer struct {
	buffer []interface{}

	size int

	head int

	tail int

	count int

	mutex sync.RWMutex
}

// NewCircularBuffer creates a new circular buffer.

func NewCircularBuffer(size int) *CircularBuffer {

	return &CircularBuffer{

		buffer: make([]interface{}, size),

		size: size,

		mutex: sync.RWMutex{},
	}

}

// Add adds an item to the circular buffer.

func (cb *CircularBuffer) Add(item interface{}) {

	cb.mutex.Lock()

	defer cb.mutex.Unlock()

	cb.buffer[cb.head] = item

	cb.head = (cb.head + 1) % cb.size

	if cb.count < cb.size {

		cb.count++

	} else {

		cb.tail = (cb.tail + 1) % cb.size

	}

}

// Tracker provides end-to-end intent processing tracking with distributed tracing correlation.

type Tracker struct {
	config *TrackerConfig

	logger *logging.StructuredLogger

	started atomic.Bool

	// Active intent tracking.

	activeIntents map[string]*IntentExecution

	intentsMu sync.RWMutex

	// Component tracking.

	componentLatency map[string]*ComponentLatencyTracker

	componentMu sync.RWMutex

	// Error tracking and categorization.

	errorCategorizer *ErrorCategorizer

	impactWeighter *ImpactWeighter

	// Queue depth monitoring.

	queueMonitor *QueueDepthMonitor

	// Performance metrics.

	metrics *TrackerMetrics

	trackingCount atomic.Uint64

	completionCount atomic.Uint64

	errorCount atomic.Uint64

	timeoutCount atomic.Uint64

	// Background processing.

	cleanupTicker *time.Ticker

	stopCh chan struct{}

	wg sync.WaitGroup
}

// TrackerConfig holds configuration for the intent tracker.

type TrackerConfig struct {

	// Tracking limits.

	MaxActiveIntents int `yaml:"max_active_intents"`

	TrackingTimeout time.Duration `yaml:"tracking_timeout"`

	CompletionThreshold time.Duration `yaml:"completion_threshold"`

	// Cleanup configuration.

	CleanupInterval time.Duration `yaml:"cleanup_interval"`

	RetentionPeriod time.Duration `yaml:"retention_period"`

	// Component timeout thresholds.

	ComponentTimeouts map[string]time.Duration `yaml:"component_timeouts"`

	// Error impact weights.

	ErrorImpactWeights map[string]float64 `yaml:"error_impact_weights"`

	// Queue monitoring.

	QueueMonitoringEnabled bool `yaml:"queue_monitoring_enabled"`

	QueueDepthThreshold int `yaml:"queue_depth_threshold"`
}

// DefaultTrackerConfig returns optimized default configuration.

func DefaultTrackerConfig() *TrackerConfig {

	return &TrackerConfig{

		MaxActiveIntents: 1000,

		TrackingTimeout: 10 * time.Minute,

		CompletionThreshold: 30 * time.Second,

		CleanupInterval: 1 * time.Minute,

		RetentionPeriod: 1 * time.Hour,

		ComponentTimeouts: map[string]time.Duration{

			"llm-processing": 30 * time.Second,

			"rag-retrieval": 10 * time.Second,

			"package-generation": 2 * time.Minute,

			"gitops-commit": 30 * time.Second,

			"deployment": 5 * time.Minute,
		},

		ErrorImpactWeights: map[string]float64{

			"authentication_failure": 0.9,

			"llm_timeout": 0.8,

			"rag_unavailable": 0.7,

			"git_commit_failed": 0.6,

			"deployment_failed": 0.8,

			"validation_error": 0.3,
		},

		QueueMonitoringEnabled: true,

		QueueDepthThreshold: 100,
	}

}

// IntentExecution tracks the lifecycle of a single intent processing request.

type IntentExecution struct {

	// Basic identification.

	ID string `json:"id"`

	IntentType string `json:"intent_type"`

	UserID string `json:"user_id,omitempty"`

	RequestID string `json:"request_id,omitempty"`

	// Timing information.

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time,omitempty"`

	TotalDuration time.Duration `json:"total_duration"`

	// Distributed tracing.

	TraceID string `json:"trace_id,omitempty"`

	SpanID string `json:"span_id,omitempty"`

	// Component latencies.

	ComponentLatencies map[string]time.Duration `json:"component_latencies"`

	ComponentErrors map[string]error `json:"component_errors,omitempty"`

	ComponentStatus map[string]ComponentStatus `json:"component_status"`

	// Processing stages.

	Stages []*ProcessingStage `json:"stages"`

	CurrentStage string `json:"current_stage"`

	// Status and completion.

	Status IntentStatus `json:"status"`

	Success bool `json:"success"`

	ErrorDetails *IntentError `json:"error_details,omitempty"`

	// Queue information.

	QueuedAt time.Time `json:"queued_at,omitempty"`

	QueueDepth int `json:"queue_depth,omitempty"`

	QueueLatency time.Duration `json:"queue_latency,omitempty"`

	// Business impact.

	BusinessImpact float64 `json:"business_impact"`

	CriticalPath bool `json:"critical_path"`

	// Synchronization.

	mu sync.RWMutex
}

// IntentStatus represents the current status of intent processing.

type IntentStatus string

const (

	// IntentStatusReceived holds intentstatusreceived value.

	IntentStatusReceived IntentStatus = "received"

	// IntentStatusQueued holds intentstatusqueued value.

	IntentStatusQueued IntentStatus = "queued"

	// IntentStatusProcessing holds intentstatusprocessing value.

	IntentStatusProcessing IntentStatus = "processing"

	// IntentStatusCompleted holds intentstatuscompleted value.

	IntentStatusCompleted IntentStatus = "completed"

	// IntentStatusFailed holds intentstatusfailed value.

	IntentStatusFailed IntentStatus = "failed"

	// IntentStatusTimeout holds intentstatustimeout value.

	IntentStatusTimeout IntentStatus = "timeout"

	// IntentStatusCancelled holds intentstatuscancelled value.

	IntentStatusCancelled IntentStatus = "cancelled"
)

// ComponentStatus represents the status of a component in the processing pipeline.

type ComponentStatus string

const (

	// ComponentStatusPending holds componentstatuspending value.

	ComponentStatusPending ComponentStatus = "pending"

	// ComponentStatusProcessing holds componentstatusprocessing value.

	ComponentStatusProcessing ComponentStatus = "processing"

	// ComponentStatusCompleted holds componentstatuscompleted value.

	ComponentStatusCompleted ComponentStatus = "completed"

	// ComponentStatusFailed holds componentstatusfailed value.

	ComponentStatusFailed ComponentStatus = "failed"

	// ComponentStatusSkipped holds componentstatusskipped value.

	ComponentStatusSkipped ComponentStatus = "skipped"
)

// ProcessingStage represents a stage in the intent processing pipeline.

type ProcessingStage struct {
	Name string `json:"name"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time,omitempty"`

	Duration time.Duration `json:"duration"`

	Status ComponentStatus `json:"status"`

	Error string `json:"error,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// IntentError provides detailed error information with categorization.

type IntentError struct {
	Type string `json:"type"`

	Message string `json:"message"`

	Component string `json:"component"`

	Stage string `json:"stage"`

	Timestamp time.Time `json:"timestamp"`

	Severity string `json:"severity"`

	Impact float64 `json:"impact"`

	Retryable bool `json:"retryable"`

	StackTrace []string `json:"stack_trace,omitempty"`
}

// TrackerMetrics contains Prometheus metrics for the tracker.

type TrackerMetrics struct {

	// Intent tracking metrics.

	ActiveIntents prometheus.Gauge

	IntentsTracked *prometheus.CounterVec

	IntentDuration *prometheus.HistogramVec

	IntentSuccess *prometheus.CounterVec

	// Component latency metrics.

	ComponentLatency *prometheus.HistogramVec

	ComponentErrors *prometheus.CounterVec

	ComponentStatus *prometheus.GaugeVec

	// Queue metrics.

	QueueDepth *prometheus.GaugeVec

	QueueLatency *prometheus.HistogramVec

	QueueThroughput prometheus.Gauge

	// Error categorization metrics.

	ErrorsByCategory *prometheus.CounterVec

	ErrorImpact *prometheus.HistogramVec

	CriticalPathErrors prometheus.Counter

	// Tracking performance metrics.

	TrackingLatency prometheus.Histogram

	MemoryUsage prometheus.Gauge

	CleanupOperations prometheus.Counter
}

// ComponentLatencyTracker tracks latency for individual components.

type ComponentLatencyTracker struct {
	componentName string

	measurements *CircularBuffer

	quantiles *QuantileEstimator

	violationCount atomic.Uint64

	mu sync.RWMutex
}

// ErrorCategorizer categorizes errors by type and impact.

type ErrorCategorizer struct {
	categories map[string]*ErrorCategory

	impactWeights map[string]float64

	mu sync.RWMutex
}

// ErrorCategory represents a category of errors.

type ErrorCategory struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Severity string `json:"severity"`

	Impact float64 `json:"impact"`

	Retryable bool `json:"retryable"`

	Count atomic.Uint64 `json:"count"`
}

// ImpactWeighter calculates business impact of failures.

type ImpactWeighter struct {
	weights map[string]float64

	mu sync.RWMutex
}

// QueueDepthMonitor tracks queue depths across components.

type QueueDepthMonitor struct {
	queues map[string]*QueueMetrics

	threshold int

	enabled bool

	mu sync.RWMutex
}

// QueueMetrics tracks metrics for a specific queue.

type QueueMetrics struct {
	Name string

	Depth atomic.Int64

	Throughput atomic.Uint64

	Latency *CircularBuffer
}

// NewTracker creates a new intent tracker with the given configuration.

func NewTracker(config *TrackerConfig, logger *logging.StructuredLogger) (*Tracker, error) {

	if config == nil {

		config = DefaultTrackerConfig()

	}

	if logger == nil {

		return nil, fmt.Errorf("logger is required")

	}

	// Initialize metrics.

	metrics := &TrackerMetrics{

		ActiveIntents: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_tracker_active_intents",

			Help: "Number of actively tracked intents",
		}),

		IntentsTracked: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_tracker_intents_tracked_total",

			Help: "Total number of intents tracked",
		}, []string{"intent_type", "user_type"}),

		IntentDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "sla_tracker_intent_duration_seconds",

			Help: "Duration of intent processing",

			Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 0.1s to ~400s

		}, []string{"intent_type", "status", "critical_path"}),

		IntentSuccess: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_tracker_intent_success_total",

			Help: "Total successful intent completions",
		}, []string{"intent_type", "completion_reason"}),

		ComponentLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "sla_tracker_component_latency_seconds",

			Help: "Component processing latency",

			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~30s

		}, []string{"component", "intent_type"}),

		ComponentErrors: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_tracker_component_errors_total",

			Help: "Component processing errors",
		}, []string{"component", "error_type", "severity"}),

		ComponentStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "sla_tracker_component_status",

			Help: "Current component status (0=pending, 1=processing, 2=completed, 3=failed)",
		}, []string{"component", "intent_id"}),

		QueueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "sla_tracker_queue_depth",

			Help: "Current queue depth by component",
		}, []string{"queue_name", "component"}),

		QueueLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "sla_tracker_queue_latency_seconds",

			Help: "Time spent waiting in queue",

			Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to ~4s

		}, []string{"queue_name"}),

		QueueThroughput: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_tracker_queue_throughput",

			Help: "Current queue processing throughput",
		}),

		ErrorsByCategory: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "sla_tracker_errors_by_category_total",

			Help: "Total errors by category",
		}, []string{"category", "severity", "retryable"}),

		ErrorImpact: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "sla_tracker_error_impact",

			Help: "Business impact of errors (0.0-1.0)",

			Buckets: prometheus.LinearBuckets(0, 0.1, 11), // 0.0 to 1.0

		}, []string{"error_type"}),

		CriticalPathErrors: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_tracker_critical_path_errors_total",

			Help: "Total errors on critical path",
		}),

		TrackingLatency: prometheus.NewHistogram(prometheus.HistogramOpts{

			Name: "sla_tracker_tracking_latency_seconds",

			Help: "Latency of tracking operations",

			Buckets: prometheus.ExponentialBuckets(0.00001, 2, 12), // 10μs to ~40ms

		}),

		MemoryUsage: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "sla_tracker_memory_usage_bytes",

			Help: "Memory usage of tracker",
		}),

		CleanupOperations: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "sla_tracker_cleanup_operations_total",

			Help: "Total cleanup operations performed",
		}),
	}

	// Initialize error categorizer.

	errorCategorizer := &ErrorCategorizer{

		categories: map[string]*ErrorCategory{

			"authentication": {

				Name: "Authentication",

				Description: "Authentication and authorization failures",

				Severity: "high",

				Impact: 0.9,

				Retryable: true,
			},

			"llm_timeout": {

				Name: "LLM Timeout",

				Description: "LLM processing timeout",

				Severity: "medium",

				Impact: 0.7,

				Retryable: true,
			},

			"validation": {

				Name: "Validation",

				Description: "Input validation errors",

				Severity: "low",

				Impact: 0.3,

				Retryable: false,
			},

			"infrastructure": {

				Name: "Infrastructure",

				Description: "Infrastructure and dependency failures",

				Severity: "high",

				Impact: 0.8,

				Retryable: true,
			},
		},

		impactWeights: config.ErrorImpactWeights,
	}

	// Initialize impact weighter.

	impactWeighter := &ImpactWeighter{

		weights: config.ErrorImpactWeights,
	}

	// Initialize queue monitor.

	queueMonitor := &QueueDepthMonitor{

		queues: make(map[string]*QueueMetrics),

		threshold: config.QueueDepthThreshold,

		enabled: config.QueueMonitoringEnabled,
	}

	tracker := &Tracker{

		config: config,

		logger: logger.WithComponent("tracker"),

		activeIntents: make(map[string]*IntentExecution),

		componentLatency: make(map[string]*ComponentLatencyTracker),

		errorCategorizer: errorCategorizer,

		impactWeighter: impactWeighter,

		queueMonitor: queueMonitor,

		metrics: metrics,

		stopCh: make(chan struct{}),
	}

	// Initialize component latency trackers.

	for component := range config.ComponentTimeouts {

		tracker.componentLatency[component] = &ComponentLatencyTracker{

			componentName: component,

			measurements: NewCircularBuffer(1000),

			quantiles: NewQuantileEstimator(),
		}

	}

	return tracker, nil

}

// Start begins the intent tracking service.

func (t *Tracker) Start(ctx context.Context) error {

	if t.started.Load() {

		return fmt.Errorf("tracker already started")

	}

	t.logger.InfoWithContext("Starting intent tracker",

		"max_active_intents", t.config.MaxActiveIntents,

		"tracking_timeout", t.config.TrackingTimeout,

		"cleanup_interval", t.config.CleanupInterval,
	)

	// Start cleanup ticker.

	t.cleanupTicker = time.NewTicker(t.config.CleanupInterval)

	// Start background processing.

	t.wg.Add(1)

	go t.runCleanupLoop(ctx)

	t.wg.Add(1)

	go t.updateMetrics(ctx)

	t.started.Store(true)

	t.logger.InfoWithContext("Intent tracker started successfully")

	return nil

}

// Stop gracefully stops the intent tracker.

func (t *Tracker) Stop(ctx context.Context) error {

	if !t.started.Load() {

		return nil

	}

	t.logger.InfoWithContext("Stopping intent tracker")

	// Stop cleanup ticker.

	if t.cleanupTicker != nil {

		t.cleanupTicker.Stop()

	}

	// Signal stop.

	close(t.stopCh)

	// Wait for background goroutines.

	t.wg.Wait()

	t.logger.InfoWithContext("Intent tracker stopped")

	return nil

}

// StartIntent begins tracking a new intent execution.

func (t *Tracker) StartIntent(ctx context.Context, intentType, userID, requestID string) (*IntentExecution, error) {

	start := time.Now()

	defer func() {

		t.metrics.TrackingLatency.Observe(time.Since(start).Seconds())

	}()

	// Check capacity limits.

	t.intentsMu.RLock()

	currentCount := len(t.activeIntents)

	t.intentsMu.RUnlock()

	if currentCount >= t.config.MaxActiveIntents {

		return nil, fmt.Errorf("maximum active intents reached: %d", t.config.MaxActiveIntents)

	}

	// Create new intent execution.

	intentID := uuid.New().String()

	// Extract tracing information.

	var traceID, spanID string

	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {

		traceID = span.SpanContext().TraceID().String()

		spanID = span.SpanContext().SpanID().String()

	}

	intent := &IntentExecution{

		ID: intentID,

		IntentType: intentType,

		UserID: userID,

		RequestID: requestID,

		StartTime: time.Now(),

		TraceID: traceID,

		SpanID: spanID,

		ComponentLatencies: make(map[string]time.Duration),

		ComponentErrors: make(map[string]error),

		ComponentStatus: make(map[string]ComponentStatus),

		Stages: make([]*ProcessingStage, 0),

		Status: IntentStatusReceived,

		BusinessImpact: t.calculateBusinessImpact(intentType, userID),

		CriticalPath: t.isCriticalPath(intentType),
	}

	// Add to active tracking.

	t.intentsMu.Lock()

	t.activeIntents[intentID] = intent

	t.intentsMu.Unlock()

	// Update metrics.

	t.trackingCount.Add(1)

	t.metrics.ActiveIntents.Inc()

	t.metrics.IntentsTracked.WithLabelValues(intentType, t.getUserType(userID)).Inc()

	t.logger.InfoWithContext("Started tracking intent",

		"intent_id", intentID,

		"intent_type", intentType,

		"user_id", userID,

		"request_id", requestID,

		"trace_id", traceID,

		"business_impact", intent.BusinessImpact,

		"critical_path", intent.CriticalPath,
	)

	return intent, nil

}

// CompleteIntent marks an intent as completed successfully.

func (t *Tracker) CompleteIntent(intentID string, success bool, errorDetails *IntentError) error {

	start := time.Now()

	defer func() {

		t.metrics.TrackingLatency.Observe(time.Since(start).Seconds())

	}()

	t.intentsMu.Lock()

	intent, exists := t.activeIntents[intentID]

	if !exists {

		t.intentsMu.Unlock()

		return fmt.Errorf("intent not found: %s", intentID)

	}

	// Update intent completion.

	intent.mu.Lock()

	intent.EndTime = time.Now()

	intent.TotalDuration = intent.EndTime.Sub(intent.StartTime)

	intent.Success = success

	intent.ErrorDetails = errorDetails

	if success {

		intent.Status = IntentStatusCompleted

	} else {

		intent.Status = IntentStatusFailed

	}

	// Complete current stage if any.

	if intent.CurrentStage != "" && len(intent.Stages) > 0 {

		lastStage := intent.Stages[len(intent.Stages)-1]

		if lastStage.Status == ComponentStatusProcessing {

			lastStage.EndTime = time.Now()

			lastStage.Duration = lastStage.EndTime.Sub(lastStage.StartTime)

			if success {

				lastStage.Status = ComponentStatusCompleted

			} else {

				lastStage.Status = ComponentStatusFailed

				if errorDetails != nil {

					lastStage.Error = errorDetails.Message

				}

			}

		}

	}

	intent.mu.Unlock()

	// Remove from active tracking.

	delete(t.activeIntents, intentID)

	t.intentsMu.Unlock()

	// Update metrics.

	t.completionCount.Add(1)

	t.metrics.ActiveIntents.Dec()

	criticalPathStr := "false"

	if intent.CriticalPath {

		criticalPathStr = "true"

	}

	statusStr := "completed"

	if !success {

		statusStr = "failed"

		t.errorCount.Add(1)

		if intent.CriticalPath {

			t.metrics.CriticalPathErrors.Inc()

		}

		if errorDetails != nil {

			t.categorizeError(errorDetails)

		}

	}

	t.metrics.IntentDuration.WithLabelValues(intent.IntentType, statusStr, criticalPathStr).
		Observe(intent.TotalDuration.Seconds())

	t.metrics.IntentSuccess.WithLabelValues(intent.IntentType, statusStr).Inc()

	// Record component latencies.

	for component, latency := range intent.ComponentLatencies {

		t.recordComponentLatency(component, intent.IntentType, latency)

	}

	t.logger.InfoWithContext("Completed tracking intent",

		"intent_id", intentID,

		"intent_type", intent.IntentType,

		"success", success,

		"duration", intent.TotalDuration,

		"critical_path", intent.CriticalPath,
	)

	return nil

}

// StartStage begins tracking a processing stage for an intent.

func (t *Tracker) StartStage(intentID, stageName string, metadata map[string]interface{}) error {

	t.intentsMu.RLock()

	intent, exists := t.activeIntents[intentID]

	t.intentsMu.RUnlock()

	if !exists {

		return fmt.Errorf("intent not found: %s", intentID)

	}

	intent.mu.Lock()

	defer intent.mu.Unlock()

	// Complete previous stage if still processing.

	if len(intent.Stages) > 0 {

		lastStage := intent.Stages[len(intent.Stages)-1]

		if lastStage.Status == ComponentStatusProcessing {

			lastStage.EndTime = time.Now()

			lastStage.Duration = lastStage.EndTime.Sub(lastStage.StartTime)

			lastStage.Status = ComponentStatusCompleted

		}

	}

	// Create new stage.

	stage := &ProcessingStage{

		Name: stageName,

		StartTime: time.Now(),

		Status: ComponentStatusProcessing,

		Metadata: metadata,
	}

	intent.Stages = append(intent.Stages, stage)

	intent.CurrentStage = stageName

	intent.Status = IntentStatusProcessing

	t.logger.DebugWithContext("Started processing stage",

		"intent_id", intentID,

		"stage", stageName,

		"metadata", metadata,
	)

	return nil

}

// CompleteStage marks a processing stage as completed.

func (t *Tracker) CompleteStage(intentID, stageName string, success bool, errorMsg string) error {

	t.intentsMu.RLock()

	intent, exists := t.activeIntents[intentID]

	t.intentsMu.RUnlock()

	if !exists {

		return fmt.Errorf("intent not found: %s", intentID)

	}

	intent.mu.Lock()

	defer intent.mu.Unlock()

	// Find and complete the stage.

	for _, stage := range intent.Stages {

		if stage.Name == stageName && stage.Status == ComponentStatusProcessing {

			stage.EndTime = time.Now()

			stage.Duration = stage.EndTime.Sub(stage.StartTime)

			if success {

				stage.Status = ComponentStatusCompleted

			} else {

				stage.Status = ComponentStatusFailed

				stage.Error = errorMsg

			}

			// Record component latency.

			intent.ComponentLatencies[stageName] = stage.Duration

			if !success {

				intent.ComponentErrors[stageName] = fmt.Errorf("%s", errorMsg)

			}

			break

		}

	}

	// Clear current stage.

	if intent.CurrentStage == stageName {

		intent.CurrentStage = ""

	}

	t.logger.DebugWithContext("Completed processing stage",

		"intent_id", intentID,

		"stage", stageName,

		"success", success,

		"error", errorMsg,
	)

	return nil

}

// RecordQueueMetrics records queue depth and latency metrics.

func (t *Tracker) RecordQueueMetrics(queueName string, depth int, latency time.Duration) {

	if !t.queueMonitor.enabled {

		return

	}

	t.queueMonitor.mu.Lock()

	defer t.queueMonitor.mu.Unlock()

	queue, exists := t.queueMonitor.queues[queueName]

	if !exists {

		queue = &QueueMetrics{

			Name: queueName,

			Latency: NewCircularBuffer(1000),
		}

		t.queueMonitor.queues[queueName] = queue

	}

	queue.Depth.Store(int64(depth))

	queue.Throughput.Add(1)

	queue.Latency.Add(float64(latency.Milliseconds()))

	// Update Prometheus metrics.

	t.metrics.QueueDepth.WithLabelValues(queueName, "default").Set(float64(depth))

	t.metrics.QueueLatency.WithLabelValues(queueName).Observe(latency.Seconds())

	// Check threshold violations.

	if depth > t.queueMonitor.threshold {

		t.logger.WarnWithContext("Queue depth threshold exceeded",

			"queue", queueName,

			"depth", depth,

			"threshold", t.queueMonitor.threshold,
		)

	}

}

// GetActiveIntents returns a snapshot of currently active intents.

func (t *Tracker) GetActiveIntents() map[string]*IntentExecution {

	t.intentsMu.RLock()

	defer t.intentsMu.RUnlock()

	// Return a copy to avoid race conditions.

	result := make(map[string]*IntentExecution, len(t.activeIntents))

	for id, intent := range t.activeIntents {

		intent.mu.RLock()

		intentCopy := IntentExecution{

			ID: intent.ID,

			IntentType: intent.IntentType,

			UserID: intent.UserID,

			RequestID: intent.RequestID,

			StartTime: intent.StartTime,

			EndTime: intent.EndTime,

			TotalDuration: intent.TotalDuration,

			TraceID: intent.TraceID,

			SpanID: intent.SpanID,

			ComponentLatencies: make(map[string]time.Duration),

			ComponentErrors: make(map[string]error),

			ComponentStatus: make(map[string]ComponentStatus),

			Stages: make([]*ProcessingStage, len(intent.Stages)),

			CurrentStage: intent.CurrentStage,

			Status: intent.Status,

			Success: intent.Success,

			ErrorDetails: intent.ErrorDetails,

			QueuedAt: intent.QueuedAt,

			QueueDepth: intent.QueueDepth,

			QueueLatency: intent.QueueLatency,

			BusinessImpact: intent.BusinessImpact,

			CriticalPath: intent.CriticalPath,
		}

		// Deep copy maps and slices.

		for k, v := range intent.ComponentLatencies {

			intentCopy.ComponentLatencies[k] = v

		}

		for k, v := range intent.ComponentErrors {

			intentCopy.ComponentErrors[k] = v

		}

		for k, v := range intent.ComponentStatus {

			intentCopy.ComponentStatus[k] = v

		}

		copy(intentCopy.Stages, intent.Stages)

		intent.mu.RUnlock()

		result[id] = &intentCopy

	}

	return result

}

// GetIntentStatus returns the current status of a specific intent.

func (t *Tracker) GetIntentStatus(intentID string) (*IntentExecution, error) {

	t.intentsMu.RLock()

	defer t.intentsMu.RUnlock()

	intent, exists := t.activeIntents[intentID]

	if !exists {

		return nil, fmt.Errorf("intent not found: %s", intentID)

	}

	// Return a copy.

	intent.mu.RLock()

	defer intent.mu.RUnlock()

	intentCopy := IntentExecution{

		ID: intent.ID,

		IntentType: intent.IntentType,

		UserID: intent.UserID,

		RequestID: intent.RequestID,

		StartTime: intent.StartTime,

		EndTime: intent.EndTime,

		TotalDuration: intent.TotalDuration,

		TraceID: intent.TraceID,

		SpanID: intent.SpanID,

		ComponentLatencies: make(map[string]time.Duration),

		ComponentErrors: make(map[string]error),

		ComponentStatus: make(map[string]ComponentStatus),

		Stages: make([]*ProcessingStage, len(intent.Stages)),

		CurrentStage: intent.CurrentStage,

		Status: intent.Status,

		Success: intent.Success,

		ErrorDetails: intent.ErrorDetails,

		QueuedAt: intent.QueuedAt,

		QueueDepth: intent.QueueDepth,

		QueueLatency: intent.QueueLatency,

		BusinessImpact: intent.BusinessImpact,

		CriticalPath: intent.CriticalPath,
	}

	// Deep copy maps and slices.

	for k, v := range intent.ComponentLatencies {

		intentCopy.ComponentLatencies[k] = v

	}

	for k, v := range intent.ComponentErrors {

		intentCopy.ComponentErrors[k] = v

	}

	for k, v := range intent.ComponentStatus {

		intentCopy.ComponentStatus[k] = v

	}

	copy(intentCopy.Stages, intent.Stages)

	return &intentCopy, nil

}

// GetTrackerStats returns current tracker statistics.

func (t *Tracker) GetTrackerStats() TrackerStats {

	t.intentsMu.RLock()

	activeCount := len(t.activeIntents)

	t.intentsMu.RUnlock()

	return TrackerStats{

		ActiveIntents: activeCount,

		TotalTracked: t.trackingCount.Load(),

		TotalCompleted: t.completionCount.Load(),

		TotalErrors: t.errorCount.Load(),

		TotalTimeouts: t.timeoutCount.Load(),
	}

}

// TrackerStats contains tracker performance statistics.

type TrackerStats struct {
	ActiveIntents int `json:"active_intents"`

	TotalTracked uint64 `json:"total_tracked"`

	TotalCompleted uint64 `json:"total_completed"`

	TotalErrors uint64 `json:"total_errors"`

	TotalTimeouts uint64 `json:"total_timeouts"`
}

// runCleanupLoop performs periodic cleanup of expired intent tracking data.

func (t *Tracker) runCleanupLoop(ctx context.Context) {

	defer t.wg.Done()

	for {

		select {

		case <-ctx.Done():

			return

		case <-t.stopCh:

			return

		case <-t.cleanupTicker.C:

			t.performCleanup()

		}

	}

}

// performCleanup removes expired and timed-out intent tracking data.

func (t *Tracker) performCleanup() {

	start := time.Now()

	defer func() {

		t.metrics.CleanupOperations.Inc()

		t.logger.DebugWithContext("Cleanup completed", "duration", time.Since(start))

	}()

	now := time.Now()

	timeoutThreshold := now.Add(-t.config.TrackingTimeout)

	t.intentsMu.Lock()

	defer t.intentsMu.Unlock()

	expiredIntents := make([]string, 0)

	for intentID, intent := range t.activeIntents {

		// Check for timeout.

		if intent.StartTime.Before(timeoutThreshold) {

			intent.mu.Lock()

			intent.Status = IntentStatusTimeout

			intent.EndTime = now

			intent.TotalDuration = intent.EndTime.Sub(intent.StartTime)

			intent.mu.Unlock()

			expiredIntents = append(expiredIntents, intentID)

			t.timeoutCount.Add(1)

			// Record timeout metrics.

			criticalPathStr := "false"

			if intent.CriticalPath {

				criticalPathStr = "true"

			}

			t.metrics.IntentDuration.WithLabelValues(intent.IntentType, "timeout", criticalPathStr).
				Observe(intent.TotalDuration.Seconds())

			t.logger.WarnWithContext("Intent tracking timed out",

				"intent_id", intentID,

				"intent_type", intent.IntentType,

				"duration", intent.TotalDuration,

				"timeout_threshold", t.config.TrackingTimeout,
			)

		}

	}

	// Remove expired intents.

	for _, intentID := range expiredIntents {

		delete(t.activeIntents, intentID)

		t.metrics.ActiveIntents.Dec()

	}

	if len(expiredIntents) > 0 {

		t.logger.InfoWithContext("Cleaned up expired intents", "count", len(expiredIntents))

	}

}

// updateMetrics updates tracker performance metrics.

func (t *Tracker) updateMetrics(ctx context.Context) {

	defer t.wg.Done()

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-t.stopCh:

			return

		case <-ticker.C:

			t.updatePerformanceMetrics()

		}

	}

}

// updatePerformanceMetrics updates performance and resource usage metrics.

func (t *Tracker) updatePerformanceMetrics() {

	// Update memory usage (simplified calculation).

	t.intentsMu.RLock()

	activeCount := len(t.activeIntents)

	t.intentsMu.RUnlock()

	// Estimate memory usage: ~1KB per active intent.

	estimatedMemory := activeCount * 1024

	t.metrics.MemoryUsage.Set(float64(estimatedMemory))

	// Update queue throughput.

	totalThroughput := 0.0

	t.queueMonitor.mu.RLock()

	for _, queue := range t.queueMonitor.queues {

		totalThroughput += float64(queue.Throughput.Load())

		queue.Throughput.Store(0) // Reset counter

	}

	t.queueMonitor.mu.RUnlock()

	t.metrics.QueueThroughput.Set(totalThroughput / 30.0) // Per second over 30s window

}

// recordComponentLatency records latency for a specific component.

func (t *Tracker) recordComponentLatency(component, intentType string, latency time.Duration) {

	t.componentMu.Lock()

	defer t.componentMu.Unlock()

	tracker, exists := t.componentLatency[component]

	if !exists {

		tracker = &ComponentLatencyTracker{

			componentName: component,

			measurements: NewCircularBuffer(1000),

			quantiles: NewQuantileEstimator(),
		}

		t.componentLatency[component] = tracker

	}

	latencyMs := float64(latency.Milliseconds())

	tracker.measurements.Add(latencyMs)

	tracker.quantiles.AddObservation(latencyMs)

	// Record Prometheus metrics.

	t.metrics.ComponentLatency.WithLabelValues(component, intentType).Observe(latency.Seconds())

	// Check for SLO violations.

	if timeout, exists := t.config.ComponentTimeouts[component]; exists && latency > timeout {

		tracker.violationCount.Add(1)

		t.metrics.ComponentErrors.WithLabelValues(component, "timeout", "medium").Inc()

		t.logger.WarnWithContext("Component latency SLO violation",

			"component", component,

			"intent_type", intentType,

			"latency", latency,

			"slo_target", timeout,
		)

	}

}

// categorizeError categorizes and records an error for impact analysis.

func (t *Tracker) categorizeError(errorDetails *IntentError) {

	t.errorCategorizer.mu.Lock()

	defer t.errorCategorizer.mu.Unlock()

	// Determine error category.

	category := t.determineErrorCategory(errorDetails)

	if cat, exists := t.errorCategorizer.categories[category]; exists {

		cat.Count.Add(1)

		// Update Prometheus metrics.

		retryableStr := "false"

		if cat.Retryable {

			retryableStr = "true"

		}

		t.metrics.ErrorsByCategory.WithLabelValues(category, cat.Severity, retryableStr).Inc()

		t.metrics.ErrorImpact.WithLabelValues(errorDetails.Type).Observe(cat.Impact)

	}

}

// determineErrorCategory determines the category of an error.

func (t *Tracker) determineErrorCategory(errorDetails *IntentError) string {

	errorType := errorDetails.Type

	// Simple categorization logic - in production this would be more sophisticated.

	switch {

	case contains(errorType, "auth", "login", "permission"):

		return "authentication"

	case contains(errorType, "timeout", "deadline"):

		return "timeout"

	case contains(errorType, "validation", "invalid"):

		return "validation"

	case contains(errorType, "network", "connection", "unavailable"):

		return "infrastructure"

	default:

		return "unknown"

	}

}

// calculateBusinessImpact calculates the business impact score for an intent.

func (t *Tracker) calculateBusinessImpact(intentType, userID string) float64 {

	// Simplified business impact calculation.

	baseImpact := 0.5 // Default impact

	// Adjust based on intent type.

	switch intentType {

	case "critical-deployment":

		baseImpact = 0.9

	case "production-config":

		baseImpact = 0.8

	case "monitoring-setup":

		baseImpact = 0.6

	case "development-test":

		baseImpact = 0.3

	}

	// Adjust based on user type (simplified).

	if userID != "" {

		// Premium users have higher impact.

		if userID[0] == 'p' { // Simplified premium user detection

			baseImpact *= 1.2

		}

	}

	return math.Min(baseImpact, 1.0)

}

// isCriticalPath determines if an intent is on the critical path.

func (t *Tracker) isCriticalPath(intentType string) bool {

	criticalTypes := []string{

		"critical-deployment",

		"production-config",

		"security-update",

		"incident-response",
	}

	for _, criticalType := range criticalTypes {

		if intentType == criticalType {

			return true

		}

	}

	return false

}

// getUserType determines the user type for metrics.

func (t *Tracker) getUserType(userID string) string {

	if userID == "" {

		return "anonymous"

	}

	// Simplified user type detection.

	if userID != "" {

		switch userID[0] {

		case 'a':

			return "admin"

		case 'p':

			return "premium"

		case 'd':

			return "developer"

		default:

			return "standard"

		}

	}

	return "unknown"

}

// Helper function to check if a string contains any of the given substrings.

func contains(str string, substrings ...string) bool {

	for _, substring := range substrings {

		if len(str) >= len(substring) {

			for i := 0; i <= len(str)-len(substring); i++ {

				if str[i:i+len(substring)] == substring {

					return true

				}

			}

		}

	}

	return false

}
