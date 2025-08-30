/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// EnhancedEventBus provides enhanced event communication with persistence and routing.

type EnhancedEventBus struct {
	logger logr.Logger

	mutex sync.RWMutex

	// Event handling.

	subscribers map[string][]EventHandler

	eventRoutes map[string][]string // event type -> target components

	eventFilters map[string]EventFilter

	// Event buffering and batching.

	eventBuffer chan ProcessingEvent

	batchBuffer []ProcessingEvent

	batchSize int

	batchTimeout time.Duration

	// Event persistence.

	persistenceDir string

	enablePersistence bool

	eventLog *EventLog

	// Delivery guarantees.

	retryPolicy *RetryPolicy

	deadLetterQueue *DeadLetterQueue

	ackTimeouts map[string]time.Time

	// Event ordering.

	sequenceNumber int64

	ordering EventOrdering

	partitions map[string]*EventPartition

	// Performance optimization.

	enableBatching bool

	enableCompression bool

	maxEventSize int64

	compressionLevel int

	// Monitoring and metrics.

	metrics EventBusMetrics

	healthChecker *EventHealthChecker

	// Configuration.

	config *EventBusConfig

	// Lifecycle management.

	started bool

	stopChan chan bool

	workerWG sync.WaitGroup
}

// EventBusConfig provides comprehensive configuration for the event bus.

type EventBusConfig struct {

	// Buffer configuration.

	BufferSize int `json:"bufferSize"`

	MaxEventSize int64 `json:"maxEventSize"`

	BatchSize int `json:"batchSize"`

	BatchTimeout time.Duration `json:"batchTimeout"`

	// Persistence configuration.

	PersistenceDir string `json:"persistenceDir"`

	EnablePersistence bool `json:"enablePersistence"`

	LogRotationSize int64 `json:"logRotationSize"`

	LogRetentionDays int `json:"logRetentionDays"`

	// Delivery configuration.

	MaxRetries int `json:"maxRetries"`

	RetryBackoff time.Duration `json:"retryBackoff"`

	AckTimeout time.Duration `json:"ackTimeout"`

	DeadLetterEnabled bool `json:"deadLetterEnabled"`

	// Ordering configuration.

	OrderingMode string `json:"orderingMode"` // "none", "global", "partition"

	PartitionStrategy string `json:"partitionStrategy"` // "intent", "component", "hash"

	MaxPartitions int `json:"maxPartitions"`

	// Performance configuration.

	EnableBatching bool `json:"enableBatching"`

	EnableCompression bool `json:"enableCompression"`

	CompressionLevel int `json:"compressionLevel"`

	WorkerCount int `json:"workerCount"`

	// Monitoring configuration.

	EnableMetrics bool `json:"enableMetrics"`

	MetricsInterval time.Duration `json:"metricsInterval"`

	HealthCheckInterval time.Duration `json:"healthCheckInterval"`
}

// DefaultEventBusConfig returns default configuration.

func DefaultEventBusConfig() *EventBusConfig {

	return &EventBusConfig{

		BufferSize: 10000,

		MaxEventSize: 1024 * 1024, // 1MB

		BatchSize: 100,

		BatchTimeout: 1 * time.Second,

		PersistenceDir: "/tmp/nephoran-events",

		EnablePersistence: true,

		LogRotationSize: 100 * 1024 * 1024, // 100MB

		LogRetentionDays: 7,

		MaxRetries: 3,

		RetryBackoff: 1 * time.Second,

		AckTimeout: 30 * time.Second,

		DeadLetterEnabled: true,

		OrderingMode: "partition",

		PartitionStrategy: "intent",

		MaxPartitions: 10,

		EnableBatching: true,

		EnableCompression: false,

		CompressionLevel: 6,

		WorkerCount: 5,

		EnableMetrics: true,

		MetricsInterval: 30 * time.Second,

		HealthCheckInterval: 10 * time.Second,
	}

}

// NewEnhancedEventBus creates a new enhanced event bus.

func NewEnhancedEventBus(config *EventBusConfig) *EnhancedEventBus {

	if config == nil {

		config = DefaultEventBusConfig()

	}

	bus := &EnhancedEventBus{

		logger: ctrl.Log.WithName("enhanced-event-bus"),

		subscribers: make(map[string][]EventHandler),

		eventRoutes: make(map[string][]string),

		eventFilters: make(map[string]EventFilter),

		eventBuffer: make(chan ProcessingEvent, config.BufferSize),

		batchBuffer: make([]ProcessingEvent, 0, config.BatchSize),

		batchSize: config.BatchSize,

		batchTimeout: config.BatchTimeout,

		persistenceDir: config.PersistenceDir,

		enablePersistence: config.EnablePersistence,

		ackTimeouts: make(map[string]time.Time),

		partitions: make(map[string]*EventPartition),

		enableBatching: config.EnableBatching,

		enableCompression: config.EnableCompression,

		maxEventSize: config.MaxEventSize,

		compressionLevel: config.CompressionLevel,

		config: config,

		stopChan: make(chan bool),
	}

	// Initialize components.

	bus.initializeComponents()

	return bus

}

// initializeComponents initializes all event bus components.

func (eb *EnhancedEventBus) initializeComponents() {

	// Initialize retry policy.

	eb.retryPolicy = &RetryPolicy{

		MaxRetries: eb.config.MaxRetries,

		BackoffBase: eb.config.RetryBackoff,

		BackoffMax: 30 * time.Second,

		BackoffJitter: true,
	}

	// Initialize dead letter queue.

	eb.deadLetterQueue = NewDeadLetterQueue(eb.config.DeadLetterEnabled)

	// Initialize event log.

	if eb.enablePersistence {

		eb.eventLog = NewEventLog(eb.persistenceDir, eb.config.LogRotationSize, eb.config.LogRetentionDays)

	}

	// Initialize metrics.

	if eb.config.EnableMetrics {

		eb.metrics = NewEventBusMetricsImpl()

	}

	// Initialize health checker.

	eb.healthChecker = NewEventHealthChecker(eb.config.HealthCheckInterval)

	// Initialize event ordering.

	eb.ordering = NewEventOrdering(eb.config.OrderingMode, eb.config.PartitionStrategy)

}

// Core event bus interface implementation.

// PublishStateChange performs publishstatechange operation.

func (eb *EnhancedEventBus) PublishStateChange(ctx context.Context, event StateChangeEvent) error {

	// Convert to processing event.

	processingEvent := ProcessingEvent{

		Type: "state.changed",

		Source: "state-manager",

		IntentID: event.IntentName.String(),

		Phase: string(event.NewPhase),

		Success: true,

		Data: map[string]interface{}{

			"oldPhase": event.OldPhase,

			"newPhase": event.NewPhase,

			"version": event.Version,
		},

		Timestamp: time.Now().UnixNano(),

		CorrelationID: fmt.Sprintf("state-%d", time.Now().UnixNano()),

		Metadata: make(map[string]string),
	}

	// Add metadata from event.

	for key, value := range event.Metadata {

		if strValue, ok := value.(string); ok {

			processingEvent.Metadata[key] = strValue

		}

	}

	return eb.Publish(ctx, processingEvent)

}

// Subscribe performs subscribe operation.

func (eb *EnhancedEventBus) Subscribe(eventType string, handler EventHandler) error {

	if handler == nil {

		return fmt.Errorf("handler cannot be nil")

	}

	eb.mutex.Lock()

	defer eb.mutex.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)

	eb.logger.Info("Subscribed to event type", "eventType", eventType,

		"handlerCount", len(eb.subscribers[eventType]))

	return nil

}

// Unsubscribe performs unsubscribe operation.

func (eb *EnhancedEventBus) Unsubscribe(eventType string) error {

	eb.mutex.Lock()

	defer eb.mutex.Unlock()

	delete(eb.subscribers, eventType)

	eb.logger.Info("Unsubscribed from event type", "eventType", eventType)

	return nil

}

// Start performs start operation.

func (eb *EnhancedEventBus) Start(ctx context.Context) error {

	eb.mutex.Lock()

	defer eb.mutex.Unlock()

	if eb.started {

		return fmt.Errorf("event bus already started")

	}

	// Ensure persistence directory exists.

	if eb.enablePersistence {

		if err := os.MkdirAll(eb.persistenceDir, 0o755); err != nil {

			return fmt.Errorf("failed to create persistence directory: %w", err)

		}

	}

	// Start worker goroutines.

	for i := range eb.config.WorkerCount {

		eb.workerWG.Add(1)

		go eb.eventWorker(ctx, i)

	}

	// Start batch processor if enabled.

	if eb.enableBatching {

		eb.workerWG.Add(1)

		go eb.batchProcessor(ctx)

	}

	// Start health checker.

	if eb.healthChecker != nil {

		eb.workerWG.Add(1)

		go eb.healthChecker.Start(ctx, &eb.workerWG)

	}

	// Start metrics collector.

	if eb.metrics != nil {

		eb.workerWG.Add(1)

		go eb.metricsCollector(ctx)

	}

	eb.started = true

	eb.logger.Info("Enhanced event bus started", "workers", eb.config.WorkerCount)

	return nil

}

// Stop performs stop operation.

func (eb *EnhancedEventBus) Stop(ctx context.Context) error {

	eb.mutex.Lock()

	defer eb.mutex.Unlock()

	if !eb.started {

		return nil

	}

	eb.logger.Info("Stopping enhanced event bus")

	// Signal stop.

	close(eb.stopChan)

	// Wait for workers to finish with timeout.

	done := make(chan bool)

	go func() {

		eb.workerWG.Wait()

		done <- true

	}()

	select {

	case <-done:

		eb.logger.Info("All workers stopped gracefully")

	case <-time.After(10 * time.Second):

		eb.logger.Info("Worker shutdown timeout reached")

	}

	// Close resources.

	if eb.eventLog != nil {

		eb.eventLog.Close()

	}

	eb.started = false

	return nil

}

// GetEventHistory performs geteventhistory operation.

func (eb *EnhancedEventBus) GetEventHistory(ctx context.Context, intentID string) ([]ProcessingEvent, error) {

	if !eb.enablePersistence || eb.eventLog == nil {

		return nil, fmt.Errorf("persistence not enabled")

	}

	return eb.eventLog.GetEventsByIntentID(intentID)

}

// GetEventsByType performs geteventsbytype operation.

func (eb *EnhancedEventBus) GetEventsByType(ctx context.Context, eventType string, limit int) ([]ProcessingEvent, error) {

	if !eb.enablePersistence || eb.eventLog == nil {

		return nil, fmt.Errorf("persistence not enabled")

	}

	return eb.eventLog.GetEventsByType(eventType, limit)

}

// Enhanced functionality.

// Publish performs publish operation.

func (eb *EnhancedEventBus) Publish(ctx context.Context, event ProcessingEvent) error {

	if !eb.started {

		return fmt.Errorf("event bus not started")

	}

	// Validate event size.

	eventData, err := json.Marshal(event)

	if err != nil {

		return fmt.Errorf("failed to marshal event: %w", err)

	}

	if int64(len(eventData)) > eb.maxEventSize {

		return fmt.Errorf("event size exceeds maximum allowed size")

	}

	// Assign sequence number for ordering.

	eb.mutex.Lock()

	eb.sequenceNumber++

	event.Data["sequenceNumber"] = eb.sequenceNumber

	eb.mutex.Unlock()

	// Add to partition if ordering is enabled.

	if eb.config.OrderingMode == "partition" {

		partition := eb.getOrCreatePartition(event.IntentID)

		partition.AddEvent(event)

	}

	// Persist event if enabled.

	if eb.enablePersistence && eb.eventLog != nil {

		if err := eb.eventLog.WriteEvent(event); err != nil {

			eb.logger.Error(err, "Failed to persist event", "eventType", event.Type)

		}

	}

	// Update metrics.

	if eb.metrics != nil {

		eb.metrics.RecordEventPublished(event.Type)

	}

	// Send to processing channel.

	select {

	case eb.eventBuffer <- event:

		return nil

	case <-ctx.Done():

		return ctx.Err()

	default:

		return fmt.Errorf("event buffer full")

	}

}

// AddEventRoute performs addeventroute operation.

func (eb *EnhancedEventBus) AddEventRoute(eventType string, targetComponents []string) {

	eb.mutex.Lock()

	defer eb.mutex.Unlock()

	eb.eventRoutes[eventType] = targetComponents

}

// AddEventFilter performs addeventfilter operation.

func (eb *EnhancedEventBus) AddEventFilter(eventType string, filter EventFilter) {

	eb.mutex.Lock()

	defer eb.mutex.Unlock()

	eb.eventFilters[eventType] = filter

}

// Worker methods.

func (eb *EnhancedEventBus) eventWorker(ctx context.Context, workerID int) {

	defer eb.workerWG.Done()

	eb.logger.V(1).Info("Event worker started", "workerID", workerID)

	for {

		select {

		case event := <-eb.eventBuffer:

			eb.processEvent(ctx, event)

		case <-eb.stopChan:

			eb.logger.V(1).Info("Event worker stopping", "workerID", workerID)

			return

		case <-ctx.Done():

			eb.logger.V(1).Info("Event worker cancelled", "workerID", workerID)

			return

		}

	}

}

func (eb *EnhancedEventBus) processEvent(ctx context.Context, event ProcessingEvent) {

	start := time.Now()

	// Apply filters.

	if filter, exists := eb.eventFilters[event.Type]; exists {

		if !filter.ShouldProcess(event) {

			eb.logger.V(1).Info("Event filtered out", "eventType", event.Type, "intentID", event.IntentID)

			return

		}

	}

	// Get subscribers.

	eb.mutex.RLock()

	handlers := make([]EventHandler, len(eb.subscribers[event.Type]))

	copy(handlers, eb.subscribers[event.Type])

	// Also get wildcard subscribers.

	wildcardHandlers := eb.subscribers["*"]

	handlers = append(handlers, wildcardHandlers...)

	eb.mutex.RUnlock()

	if len(handlers) == 0 {

		eb.logger.V(1).Info("No subscribers for event type", "eventType", event.Type)

		return

	}

	// Process with each handler.

	successCount := 0

	for i, handler := range handlers {

		if err := eb.executeHandlerWithRetry(ctx, handler, event); err != nil {

			eb.logger.Error(err, "Handler failed permanently",

				"eventType", event.Type, "handlerIndex", i)

			// Send to dead letter queue if enabled.

			if eb.deadLetterQueue != nil {

				eb.deadLetterQueue.Add(event, err)

			}

		} else {

			successCount++

		}

	}

	// Update metrics.

	if eb.metrics != nil {

		eb.metrics.RecordEventProcessed(time.Since(start))

		if successCount < len(handlers) {

			eb.metrics.RecordHandlerFailure()

		}

	}

}

func (eb *EnhancedEventBus) executeHandlerWithRetry(ctx context.Context, handler EventHandler, event ProcessingEvent) error {

	return eb.retryPolicy.Execute(func() error {

		return handler(ctx, event)

	})

}

func (eb *EnhancedEventBus) batchProcessor(ctx context.Context) {

	defer eb.workerWG.Done()

	ticker := time.NewTicker(eb.batchTimeout)

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			eb.processBatch()

		case <-eb.stopChan:

			// Process remaining batch before stopping.

			if len(eb.batchBuffer) > 0 {

				eb.processBatch()

			}

			return

		case <-ctx.Done():

			return

		}

	}

}

func (eb *EnhancedEventBus) processBatch() {

	eb.mutex.Lock()

	if len(eb.batchBuffer) == 0 {

		eb.mutex.Unlock()

		return

	}

	batch := make([]ProcessingEvent, len(eb.batchBuffer))

	copy(batch, eb.batchBuffer)

	eb.batchBuffer = eb.batchBuffer[:0] // Clear buffer

	eb.mutex.Unlock()

	eb.logger.V(1).Info("Processing event batch", "size", len(batch))

	// Process batch events.

	for _, event := range batch {

		eb.processEvent(context.Background(), event)

	}

}

func (eb *EnhancedEventBus) metricsCollector(ctx context.Context) {

	defer eb.workerWG.Done()

	ticker := time.NewTicker(eb.config.MetricsInterval)

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			eb.collectMetrics()

		case <-eb.stopChan:

			return

		case <-ctx.Done():

			return

		}

	}

}

func (eb *EnhancedEventBus) collectMetrics() {

	if eb.metrics == nil {

		return

	}

	// Collect buffer utilization.

	bufferUtilization := float64(len(eb.eventBuffer)) / float64(cap(eb.eventBuffer))

	eb.metrics.SetBufferUtilization(bufferUtilization)

	// Collect partition metrics.

	eb.mutex.RLock()

	partitionCount := len(eb.partitions)

	eb.mutex.RUnlock()

	eb.metrics.SetPartitionCount(partitionCount)

}

func (eb *EnhancedEventBus) getOrCreatePartition(intentID string) *EventPartition {

	eb.mutex.Lock()

	defer eb.mutex.Unlock()

	partition, exists := eb.partitions[intentID]

	if !exists {

		partition = NewEventPartition(intentID, 1000) // Max 1000 events per partition

		eb.partitions[intentID] = partition

	}

	return partition

}

// Event filtering interface.

type EventFilter interface {
	ShouldProcess(event ProcessingEvent) bool
}

// Simple event filter implementation.

type SimpleEventFilter struct {
	AllowedSources []string

	AllowedPhases []string

	MinTimestamp int64
}

// ShouldProcess performs shouldprocess operation.

func (f *SimpleEventFilter) ShouldProcess(event ProcessingEvent) bool {

	// Check source filter.

	if len(f.AllowedSources) > 0 {

		allowed := false

		for _, source := range f.AllowedSources {

			if event.Source == source {

				allowed = true

				break

			}

		}

		if !allowed {

			return false

		}

	}

	// Check phase filter.

	if len(f.AllowedPhases) > 0 {

		allowed := false

		for _, phase := range f.AllowedPhases {

			if event.Phase == phase {

				allowed = true

				break

			}

		}

		if !allowed {

			return false

		}

	}

	// Check timestamp filter.

	if f.MinTimestamp > 0 && event.Timestamp < f.MinTimestamp {

		return false

	}

	return true

}
