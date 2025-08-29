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

	"math"

	"os"

	"path/filepath"

	"sort"

	"sync"

	"time"



	"github.com/go-logr/logr"



	ctrl "sigs.k8s.io/controller-runtime"

)



// RetryPolicy defines retry behavior for failed operations.

type RetryPolicy struct {

	MaxRetries      int           `json:"maxRetries"`

	BackoffBase     time.Duration `json:"backoffBase"`

	BackoffMax      time.Duration `json:"backoffMax"`

	BackoffJitter   bool          `json:"backoffJitter"`

	RetryableErrors []string      `json:"retryableErrors,omitempty"`

}



// Execute executes a function with retry logic.

func (rp *RetryPolicy) Execute(fn func() error) error {

	var lastErr error



	for attempt := 0; attempt <= rp.MaxRetries; attempt++ {

		if attempt > 0 {

			backoff := rp.calculateBackoff(attempt)

			time.Sleep(backoff)

		}



		if err := fn(); err != nil {

			lastErr = err



			// Check if error is retryable.

			if !rp.isRetryableError(err) {

				return err

			}



			continue

		}



		return nil // Success

	}



	return fmt.Errorf("operation failed after %d attempts: %w", rp.MaxRetries+1, lastErr)

}



func (rp *RetryPolicy) calculateBackoff(attempt int) time.Duration {

	backoff := time.Duration(math.Pow(2, float64(attempt))) * rp.BackoffBase



	if backoff > rp.BackoffMax {

		backoff = rp.BackoffMax

	}



	// Add jitter if enabled.

	if rp.BackoffJitter && attempt > 0 {

		jitter := time.Duration(float64(backoff) * 0.1 * (2*float64(time.Now().UnixNano()%100)/100 - 1))

		backoff += jitter

	}



	return backoff

}



func (rp *RetryPolicy) isRetryableError(err error) bool {

	if len(rp.RetryableErrors) == 0 {

		return true // Retry all errors by default

	}



	errStr := err.Error()

	for _, retryableErr := range rp.RetryableErrors {

		if errStr == retryableErr {

			return true

		}

	}



	return false

}



// DeadLetterQueue handles failed events.

type DeadLetterQueue struct {

	enabled bool

	events  []DeadLetterEvent

	maxSize int

	mutex   sync.RWMutex

	logger  logr.Logger

}



// DeadLetterEvent represents a failed event.

type DeadLetterEvent struct {

	Event        ProcessingEvent `json:"event"`

	Error        string          `json:"error"`

	Timestamp    time.Time       `json:"timestamp"`

	AttemptCount int             `json:"attemptCount"`

	LastAttempt  time.Time       `json:"lastAttempt"`

}



// NewDeadLetterQueue creates a new dead letter queue.

func NewDeadLetterQueue(enabled bool) *DeadLetterQueue {

	return &DeadLetterQueue{

		enabled: enabled,

		events:  make([]DeadLetterEvent, 0),

		maxSize: 1000,

		logger:  ctrl.Log.WithName("dead-letter-queue"),

	}

}



// Add adds a failed event to the dead letter queue.

func (dlq *DeadLetterQueue) Add(event ProcessingEvent, err error) {

	if !dlq.enabled {

		return

	}



	dlq.mutex.Lock()

	defer dlq.mutex.Unlock()



	deadEvent := DeadLetterEvent{

		Event:        event,

		Error:        err.Error(),

		Timestamp:    time.Now(),

		AttemptCount: 1,

		LastAttempt:  time.Now(),

	}



	dlq.events = append(dlq.events, deadEvent)



	// Cleanup old events if necessary.

	if len(dlq.events) > dlq.maxSize {

		dlq.events = dlq.events[dlq.maxSize/10:]

	}



	dlq.logger.Error(err, "Event added to dead letter queue",

		"eventType", event.Type, "intentID", event.IntentID)

}



// GetEvents returns all dead letter events.

func (dlq *DeadLetterQueue) GetEvents() []DeadLetterEvent {

	dlq.mutex.RLock()

	defer dlq.mutex.RUnlock()



	result := make([]DeadLetterEvent, len(dlq.events))

	copy(result, dlq.events)



	return result

}



// Clear clears all dead letter events.

func (dlq *DeadLetterQueue) Clear() {

	dlq.mutex.Lock()

	defer dlq.mutex.Unlock()



	dlq.events = dlq.events[:0]

}



// EventLog provides persistent event logging.

type EventLog struct {

	dir           string

	rotationSize  int64

	retentionDays int

	currentFile   *os.File

	currentSize   int64

	mutex         sync.Mutex

	logger        logr.Logger

}



// NewEventLog creates a new event log.

func NewEventLog(dir string, rotationSize int64, retentionDays int) *EventLog {

	return &EventLog{

		dir:           dir,

		rotationSize:  rotationSize,

		retentionDays: retentionDays,

		logger:        ctrl.Log.WithName("event-log"),

	}

}



// WriteEvent writes an event to the log.

func (el *EventLog) WriteEvent(event ProcessingEvent) error {

	el.mutex.Lock()

	defer el.mutex.Unlock()



	// Open current log file if needed.

	if el.currentFile == nil {

		if err := el.openCurrentFile(); err != nil {

			return fmt.Errorf("failed to open log file: %w", err)

		}

	}



	// Serialize event.

	eventData, err := json.Marshal(event)

	if err != nil {

		return fmt.Errorf("failed to serialize event: %w", err)

	}



	// Add newline.

	eventData = append(eventData, '\n')



	// Write to file.

	n, err := el.currentFile.Write(eventData)

	if err != nil {

		return fmt.Errorf("failed to write event: %w", err)

	}



	el.currentSize += int64(n)



	// Check if rotation is needed.

	if el.currentSize >= el.rotationSize {

		if err := el.rotate(); err != nil {

			el.logger.Error(err, "Failed to rotate log file")

		}

	}



	return nil

}



// GetEventsByIntentID retrieves events for a specific intent.

func (el *EventLog) GetEventsByIntentID(intentID string) ([]ProcessingEvent, error) {

	el.mutex.Lock()

	defer el.mutex.Unlock()



	var events []ProcessingEvent



	// Read from all log files.

	files, err := el.getLogFiles()

	if err != nil {

		return nil, err

	}



	for _, filename := range files {

		fileEvents, err := el.readEventsFromFile(filename, func(event ProcessingEvent) bool {

			return event.IntentID == intentID

		})

		if err != nil {

			el.logger.Error(err, "Failed to read events from file", "file", filename)

			continue

		}



		events = append(events, fileEvents...)

	}



	return events, nil

}



// GetEventsByType retrieves events by type.

func (el *EventLog) GetEventsByType(eventType string, limit int) ([]ProcessingEvent, error) {

	el.mutex.Lock()

	defer el.mutex.Unlock()



	var events []ProcessingEvent



	files, err := el.getLogFiles()

	if err != nil {

		return nil, err

	}



	// Read files in reverse order (newest first).

	for i := len(files) - 1; i >= 0 && len(events) < limit; i-- {

		fileEvents, err := el.readEventsFromFile(files[i], func(event ProcessingEvent) bool {

			return event.Type == eventType

		})

		if err != nil {

			el.logger.Error(err, "Failed to read events from file", "file", files[i])

			continue

		}



		events = append(events, fileEvents...)



		if len(events) > limit {

			events = events[:limit]

		}

	}



	return events, nil

}



// Close closes the event log.

func (el *EventLog) Close() error {

	el.mutex.Lock()

	defer el.mutex.Unlock()



	if el.currentFile != nil {

		return el.currentFile.Close()

	}



	return nil

}



// Internal methods for EventLog.



func (el *EventLog) openCurrentFile() error {

	filename := filepath.Join(el.dir, fmt.Sprintf("events-%d.log", time.Now().Unix()))



	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)

	if err != nil {

		return err

	}



	// Get current file size.

	stat, err := file.Stat()

	if err != nil {

		file.Close()

		return err

	}



	el.currentFile = file

	el.currentSize = stat.Size()



	return nil

}



func (el *EventLog) rotate() error {

	if el.currentFile != nil {

		el.currentFile.Close()

		el.currentFile = nil

		el.currentSize = 0

	}



	// Clean up old files.

	el.cleanupOldFiles()



	// Open new file.

	return el.openCurrentFile()

}



func (el *EventLog) getLogFiles() ([]string, error) {

	files, err := os.ReadDir(el.dir)

	if err != nil {

		return nil, err

	}



	var logFiles []string

	for _, file := range files {

		if !file.IsDir() && filepath.Ext(file.Name()) == ".log" {

			logFiles = append(logFiles, filepath.Join(el.dir, file.Name()))

		}

	}



	sort.Strings(logFiles)

	return logFiles, nil

}



func (el *EventLog) readEventsFromFile(filename string, filter func(ProcessingEvent) bool) ([]ProcessingEvent, error) {

	data, err := os.ReadFile(filename)

	if err != nil {

		return nil, err

	}



	var events []ProcessingEvent

	lines := splitLines(data)



	for _, line := range lines {

		if len(line) == 0 {

			continue

		}



		var event ProcessingEvent

		if err := json.Unmarshal(line, &event); err != nil {

			el.logger.Error(err, "Failed to unmarshal event", "file", filename)

			continue

		}



		if filter == nil || filter(event) {

			events = append(events, event)

		}

	}



	return events, nil

}



func (el *EventLog) cleanupOldFiles() {

	files, err := el.getLogFiles()

	if err != nil {

		return

	}



	cutoff := time.Now().AddDate(0, 0, -el.retentionDays)



	for _, filename := range files {

		stat, err := os.Stat(filename)

		if err != nil {

			continue

		}



		if stat.ModTime().Before(cutoff) {

			if err := os.Remove(filename); err != nil {

				el.logger.Error(err, "Failed to remove old log file", "file", filename)

			} else {

				el.logger.V(1).Info("Removed old log file", "file", filename)

			}

		}

	}

}



func splitLines(data []byte) [][]byte {

	var lines [][]byte

	start := 0



	for i, b := range data {

		if b == '\n' {

			if i > start {

				lines = append(lines, data[start:i])

			}

			start = i + 1

		}

	}



	// Handle last line without newline.

	if start < len(data) {

		lines = append(lines, data[start:])

	}



	return lines

}



// EventOrdering manages event ordering strategies.

type EventOrdering struct {

	mode     string

	strategy string

}



// NewEventOrdering creates a new event ordering manager.

func NewEventOrdering(mode, strategy string) EventOrdering {

	return EventOrdering{

		mode:     mode,

		strategy: strategy,

	}

}



// EventPartition manages events for a specific partition.

type EventPartition struct {

	id      string

	events  []ProcessingEvent

	maxSize int

	mutex   sync.RWMutex

	lastSeq int64

}



// NewEventPartition creates a new event partition.

func NewEventPartition(id string, maxSize int) *EventPartition {

	return &EventPartition{

		id:      id,

		events:  make([]ProcessingEvent, 0),

		maxSize: maxSize,

	}

}



// AddEvent adds an event to the partition.

func (ep *EventPartition) AddEvent(event ProcessingEvent) {

	ep.mutex.Lock()

	defer ep.mutex.Unlock()



	ep.events = append(ep.events, event)

	ep.lastSeq++



	// Cleanup old events if necessary.

	if len(ep.events) > ep.maxSize {

		ep.events = ep.events[ep.maxSize/10:]

	}

}



// GetEvents returns all events in the partition.

func (ep *EventPartition) GetEvents() []ProcessingEvent {

	ep.mutex.RLock()

	defer ep.mutex.RUnlock()



	result := make([]ProcessingEvent, len(ep.events))

	copy(result, ep.events)



	return result

}



// EventHealthChecker monitors event bus health.

type EventHealthChecker struct {

	interval  time.Duration

	logger    logr.Logger

	lastCheck time.Time

	healthy   bool

	checks    map[string]HealthCheck

}



// HealthCheck represents a health check function.

type HealthCheck struct {

	Name      string

	Check     func() error

	LastRun   time.Time

	LastError error

}



// NewEventHealthChecker creates a new health checker.

func NewEventHealthChecker(interval time.Duration) *EventHealthChecker {

	return &EventHealthChecker{

		interval: interval,

		logger:   ctrl.Log.WithName("event-health-checker"),

		healthy:  true,

		checks:   make(map[string]HealthCheck),

	}

}



// Start starts the health checker.

func (ehc *EventHealthChecker) Start(ctx context.Context, wg *sync.WaitGroup) {

	defer wg.Done()



	ticker := time.NewTicker(ehc.interval)

	defer ticker.Stop()



	for {

		select {

		case <-ticker.C:

			ehc.runHealthChecks()



		case <-ctx.Done():

			return

		}

	}

}



// AddHealthCheck adds a health check.

func (ehc *EventHealthChecker) AddHealthCheck(name string, check func() error) {

	ehc.checks[name] = HealthCheck{

		Name:  name,

		Check: check,

	}

}



// runHealthChecks runs all registered health checks.

func (ehc *EventHealthChecker) runHealthChecks() {

	ehc.lastCheck = time.Now()

	allHealthy := true



	for name, check := range ehc.checks {

		err := check.Check()

		check.LastRun = time.Now()

		check.LastError = err



		if err != nil {

			allHealthy = false

			ehc.logger.Error(err, "Health check failed", "check", name)

		}



		ehc.checks[name] = check

	}



	ehc.healthy = allHealthy



	if !ehc.healthy {

		ehc.logger.Info("Event bus health check failed")

	}

}



// IsHealthy returns the current health status.

func (ehc *EventHealthChecker) IsHealthy() bool {

	return ehc.healthy

}



// EventBusMetricsImpl implements the EventBusMetrics interface.

type EventBusMetricsImpl struct {

	mutex                sync.RWMutex

	totalEventsPublished int64

	totalEventsProcessed int64

	failedHandlers       int64

	processingTimes      []time.Duration

	bufferUtilization    float64

	partitionCount       int

	lastMetricsUpdate    time.Time

}



// NewEventBusMetricsImpl creates a new metrics implementation.

func NewEventBusMetricsImpl() *EventBusMetricsImpl {

	return &EventBusMetricsImpl{

		processingTimes: make([]time.Duration, 0, 1000),

	}

}



// RecordEventPublished performs recordeventpublished operation.

func (m *EventBusMetricsImpl) RecordEventPublished(eventType string) {

	m.mutex.Lock()

	defer m.mutex.Unlock()



	m.totalEventsPublished++

}



// RecordEventProcessed performs recordeventprocessed operation.

func (m *EventBusMetricsImpl) RecordEventProcessed(processingTime time.Duration) {

	m.mutex.Lock()

	defer m.mutex.Unlock()



	m.totalEventsProcessed++

	m.processingTimes = append(m.processingTimes, processingTime)



	// Keep only recent times.

	if len(m.processingTimes) > 1000 {

		m.processingTimes = m.processingTimes[100:]

	}

}



// RecordHandlerFailure performs recordhandlerfailure operation.

func (m *EventBusMetricsImpl) RecordHandlerFailure() {

	m.mutex.Lock()

	defer m.mutex.Unlock()



	m.failedHandlers++

}



// SetBufferUtilization performs setbufferutilization operation.

func (m *EventBusMetricsImpl) SetBufferUtilization(utilization float64) {

	m.mutex.Lock()

	defer m.mutex.Unlock()



	m.bufferUtilization = utilization

}



// SetPartitionCount performs setpartitioncount operation.

func (m *EventBusMetricsImpl) SetPartitionCount(count int) {

	m.mutex.Lock()

	defer m.mutex.Unlock()



	m.partitionCount = count

}



// GetTotalEventsPublished performs gettotaleventspublished operation.

func (m *EventBusMetricsImpl) GetTotalEventsPublished() int64 {

	m.mutex.RLock()

	defer m.mutex.RUnlock()



	return m.totalEventsPublished

}



// GetTotalEventsProcessed performs gettotaleventsprocessed operation.

func (m *EventBusMetricsImpl) GetTotalEventsProcessed() int64 {

	m.mutex.RLock()

	defer m.mutex.RUnlock()



	return m.totalEventsProcessed

}



// GetFailedHandlers performs getfailedhandlers operation.

func (m *EventBusMetricsImpl) GetFailedHandlers() int64 {

	m.mutex.RLock()

	defer m.mutex.RUnlock()



	return m.failedHandlers

}



// GetAverageProcessingTime performs getaverageprocessingtime operation.

func (m *EventBusMetricsImpl) GetAverageProcessingTime() int64 {

	m.mutex.RLock()

	defer m.mutex.RUnlock()



	if len(m.processingTimes) == 0 {

		return 0

	}



	var total time.Duration

	for _, t := range m.processingTimes {

		total += t

	}



	avg := total / time.Duration(len(m.processingTimes))

	return avg.Nanoseconds() / 1000000 // Convert to milliseconds

}



// GetBufferUtilization performs getbufferutilization operation.

func (m *EventBusMetricsImpl) GetBufferUtilization() float64 {

	m.mutex.RLock()

	defer m.mutex.RUnlock()



	return m.bufferUtilization

}

