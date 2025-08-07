package audit

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Maximum audit queue size to prevent memory issues
	MaxAuditQueueSize = 10000
	// Default batch size for processing audit events
	DefaultBatchSize = 100
	// Default flush interval for batched processing
	DefaultFlushInterval = 10 * time.Second
	// Audit format version for schema evolution
	AuditFormatVersion = "1.0"
)

var (
	// Metrics for audit system monitoring
	auditEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nephoran_audit_events_total",
		Help: "Total number of audit events processed by severity",
	}, []string{"severity", "event_type", "component"})

	auditProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nephoran_audit_processing_duration_seconds",
		Help:    "Time spent processing audit events",
		Buckets: prometheus.DefBuckets,
	}, []string{"backend", "operation"})

	auditQueueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nephoran_audit_queue_size",
		Help: "Current size of audit event queue",
	}, []string{"backend"})

	auditErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nephoran_audit_errors_total",
		Help: "Total number of audit processing errors",
	}, []string{"backend", "error_type"})
)

// AuditSystemConfig holds the configuration for the audit system
type AuditSystemConfig struct {
	// Enabled controls whether audit logging is active
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// LogLevel controls the minimum severity level for audit events
	LogLevel Severity `json:"log_level" yaml:"log_level"`
	
	// BatchSize controls how many events to process in a batch
	BatchSize int `json:"batch_size" yaml:"batch_size"`
	
	// FlushInterval controls how often to flush batched events
	FlushInterval time.Duration `json:"flush_interval" yaml:"flush_interval"`
	
	// MaxQueueSize controls the maximum number of events to queue
	MaxQueueSize int `json:"max_queue_size" yaml:"max_queue_size"`
	
	// EnableIntegrity controls whether log integrity protection is enabled
	EnableIntegrity bool `json:"enable_integrity" yaml:"enable_integrity"`
	
	// ComplianceMode controls additional compliance-specific features
	ComplianceMode []ComplianceStandard `json:"compliance_mode" yaml:"compliance_mode"`
	
	// Backends configuration for different output destinations
	Backends []BackendConfig `json:"backends" yaml:"backends"`
}

// DefaultAuditConfig returns a default configuration for the audit system
func DefaultAuditConfig() *AuditSystemConfig {
	return &AuditSystemConfig{
		Enabled:         true,
		LogLevel:        SeverityInfo,
		BatchSize:       DefaultBatchSize,
		FlushInterval:   DefaultFlushInterval,
		MaxQueueSize:    MaxAuditQueueSize,
		EnableIntegrity: true,
		ComplianceMode:  []ComplianceStandard{ComplianceSOC2, ComplianceISO27001},
		Backends:        []BackendConfig{},
	}
}

// AuditSystem is the main audit logging system
type AuditSystem struct {
	config   *AuditSystemConfig
	backends []Backend
	
	// Event processing
	eventQueue    chan *AuditEvent
	batchBuffer   []*AuditEvent
	batchMutex    sync.RWMutex
	flushTimer    *time.Timer
	
	// System state
	logger         logr.Logger
	tracer         trace.Tracer
	running        atomic.Bool
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	
	// Integrity protection
	integrityChain *IntegrityChain
	
	// Compliance features
	retentionManager *RetentionManager
	complianceLogger *ComplianceLogger
	
	// Metrics and monitoring
	lastFlush      time.Time
	eventsReceived int64
	eventsDropped  int64
}

// NewAuditSystem creates a new audit system with the provided configuration
func NewAuditSystem(config *AuditSystemConfig) (*AuditSystem, error) {
	if config == nil {
		config = DefaultAuditConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	system := &AuditSystem{
		config:       config,
		eventQueue:   make(chan *AuditEvent, config.MaxQueueSize),
		batchBuffer:  make([]*AuditEvent, 0, config.BatchSize),
		logger:       log.Log.WithName("audit-system"),
		ctx:          ctx,
		cancel:       cancel,
		lastFlush:    time.Now(),
	}
	
	// Initialize integrity protection if enabled
	if config.EnableIntegrity {
		var err error
		system.integrityChain, err = NewIntegrityChain()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize integrity chain: %w", err)
		}
	}
	
	// Initialize retention manager
	retentionConfig := &RetentionConfig{
		ComplianceMode: config.ComplianceMode,
	}
	system.retentionManager = NewRetentionManager(retentionConfig)
	
	// Initialize compliance logger
	system.complianceLogger = NewComplianceLogger(config.ComplianceMode)
	
	// Initialize backends
	for _, backendConfig := range config.Backends {
		backend, err := NewBackend(backendConfig)
		if err != nil {
			system.logger.Error(err, "Failed to initialize backend", "type", backendConfig.Type)
			continue
		}
		system.backends = append(system.backends, backend)
	}
	
	return system, nil
}

// Start begins processing audit events
func (as *AuditSystem) Start() error {
	if !as.config.Enabled {
		as.logger.Info("Audit system is disabled")
		return nil
	}
	
	if as.running.Load() {
		return fmt.Errorf("audit system is already running")
	}
	
	as.logger.Info("Starting audit system",
		"batch_size", as.config.BatchSize,
		"flush_interval", as.config.FlushInterval,
		"max_queue_size", as.config.MaxQueueSize,
		"backends", len(as.backends))
	
	as.running.Store(true)
	
	// Start the main processing goroutine
	as.wg.Add(1)
	go as.processingLoop()
	
	// Start flush timer
	as.flushTimer = time.NewTimer(as.config.FlushInterval)
	as.wg.Add(1)
	go as.timerLoop()
	
	// Start retention manager
	as.wg.Add(1)
	go as.retentionManager.Start(as.ctx, &as.wg)
	
	return nil
}

// Stop gracefully shuts down the audit system
func (as *AuditSystem) Stop() error {
	if !as.running.Load() {
		return nil
	}
	
	as.logger.Info("Stopping audit system")
	as.running.Store(false)
	
	// Cancel context to signal shutdown
	as.cancel()
	
	// Stop flush timer
	if as.flushTimer != nil {
		as.flushTimer.Stop()
	}
	
	// Wait for all goroutines to finish
	as.wg.Wait()
	
	// Flush any remaining events
	as.flushBatch()
	
	// Close backends
	for _, backend := range as.backends {
		if err := backend.Close(); err != nil {
			as.logger.Error(err, "Failed to close backend", "type", backend.Type())
		}
	}
	
	as.logger.Info("Audit system stopped")
	return nil
}

// LogEvent submits an audit event for processing
func (as *AuditSystem) LogEvent(event *AuditEvent) error {
	if !as.config.Enabled || !as.running.Load() {
		return nil
	}
	
	// Check minimum severity level
	if event.Severity < as.config.LogLevel {
		return nil
	}
	
	// Enrich event with system metadata
	as.enrichEvent(event)
	
	// Validate event structure
	if err := event.Validate(); err != nil {
		auditErrorsTotal.WithLabelValues("system", "validation_error").Inc()
		return fmt.Errorf("invalid audit event: %w", err)
	}
	
	// Apply integrity protection
	if as.integrityChain != nil {
		if err := as.integrityChain.ProcessEvent(event); err != nil {
			as.logger.Error(err, "Failed to apply integrity protection to audit event")
		}
	}
	
	// Try to enqueue the event
	select {
	case as.eventQueue <- event:
		atomic.AddInt64(&as.eventsReceived, 1)
		auditEventsTotal.WithLabelValues(
			event.Severity.String(),
			string(event.EventType),
			event.Component,
		).Inc()
		return nil
	default:
		// Queue is full, drop the event
		atomic.AddInt64(&as.eventsDropped, 1)
		auditErrorsTotal.WithLabelValues("system", "queue_full").Inc()
		return fmt.Errorf("audit queue is full, dropping event")
	}
}

// GetStats returns audit system statistics
func (as *AuditSystem) GetStats() AuditStats {
	return AuditStats{
		EventsReceived:   atomic.LoadInt64(&as.eventsReceived),
		EventsDropped:    atomic.LoadInt64(&as.eventsDropped),
		QueueSize:        len(as.eventQueue),
		BackendCount:     len(as.backends),
		LastFlushTime:    as.lastFlush,
		IntegrityEnabled: as.config.EnableIntegrity,
		ComplianceMode:   as.config.ComplianceMode,
	}
}

// processingLoop is the main event processing loop
func (as *AuditSystem) processingLoop() {
	defer as.wg.Done()
	
	for {
		select {
		case event := <-as.eventQueue:
			as.batchMutex.Lock()
			as.batchBuffer = append(as.batchBuffer, event)
			shouldFlush := len(as.batchBuffer) >= as.config.BatchSize
			as.batchMutex.Unlock()
			
			if shouldFlush {
				as.flushBatch()
			}
			
		case <-as.ctx.Done():
			as.logger.Info("Processing loop shutting down")
			return
		}
	}
}

// timerLoop handles periodic flushing based on time intervals
func (as *AuditSystem) timerLoop() {
	defer as.wg.Done()
	
	for {
		select {
		case <-as.flushTimer.C:
			as.flushBatch()
			as.flushTimer.Reset(as.config.FlushInterval)
			
		case <-as.ctx.Done():
			as.logger.Info("Timer loop shutting down")
			return
		}
	}
}

// flushBatch sends accumulated events to all backends
func (as *AuditSystem) flushBatch() {
	as.batchMutex.Lock()
	if len(as.batchBuffer) == 0 {
		as.batchMutex.Unlock()
		return
	}
	
	// Copy and clear buffer
	events := make([]*AuditEvent, len(as.batchBuffer))
	copy(events, as.batchBuffer)
	as.batchBuffer = as.batchBuffer[:0]
	as.batchMutex.Unlock()
	
	// Update queue size metric
	auditQueueSize.WithLabelValues("batch_buffer").Set(float64(len(as.batchBuffer)))
	
	// Process events with each backend
	for _, backend := range as.backends {
		if err := as.processEventsWithBackend(backend, events); err != nil {
			as.logger.Error(err, "Failed to process events with backend", "type", backend.Type())
			auditErrorsTotal.WithLabelValues(backend.Type(), "processing_error").Inc()
		}
	}
	
	// Update compliance logging
	for _, event := range events {
		as.complianceLogger.ProcessEvent(event)
	}
	
	as.lastFlush = time.Now()
	as.logger.V(1).Info("Flushed audit batch", "count", len(events))
}

// processEventsWithBackend sends events to a specific backend with timing metrics
func (as *AuditSystem) processEventsWithBackend(backend Backend, events []*AuditEvent) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		auditProcessingDuration.WithLabelValues(backend.Type(), "batch_write").Observe(duration.Seconds())
	}()
	
	return backend.WriteEvents(as.ctx, events)
}

// enrichEvent adds system metadata to audit events
func (as *AuditSystem) enrichEvent(event *AuditEvent) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	
	if event.Version == "" {
		event.Version = AuditFormatVersion
	}
	
	// Add system context
	if event.SystemContext == nil {
		event.SystemContext = &SystemContext{}
	}
	
	event.SystemContext.Hostname = getHostname()
	event.SystemContext.ProcessID = getProcessID()
	event.SystemContext.ThreadID = getGoroutineID()
	
	// Add compliance metadata based on configured standards
	for _, standard := range as.config.ComplianceMode {
		switch standard {
		case ComplianceSOC2:
			as.addSOC2Metadata(event)
		case ComplianceISO27001:
			as.addISO27001Metadata(event)
		case CompliancePCIDSS:
			as.addPCIDSSMetadata(event)
		}
	}
}

// addSOC2Metadata enriches events with SOC2-specific fields
func (as *AuditSystem) addSOC2Metadata(event *AuditEvent) {
	if event.ComplianceMetadata == nil {
		event.ComplianceMetadata = make(map[string]interface{})
	}
	
	event.ComplianceMetadata["soc2_control_id"] = as.getSOC2ControlID(event.EventType)
	event.ComplianceMetadata["soc2_trust_service"] = as.getSOC2TrustService(event.EventType)
}

// addISO27001Metadata enriches events with ISO 27001-specific fields
func (as *AuditSystem) addISO27001Metadata(event *AuditEvent) {
	if event.ComplianceMetadata == nil {
		event.ComplianceMetadata = make(map[string]interface{})
	}
	
	event.ComplianceMetadata["iso27001_control"] = as.getISO27001Control(event.EventType)
	event.ComplianceMetadata["iso27001_annex"] = as.getISO27001Annex(event.EventType)
}

// addPCIDSSMetadata enriches events with PCI DSS-specific fields
func (as *AuditSystem) addPCIDSSMetadata(event *AuditEvent) {
	if event.ComplianceMetadata == nil {
		event.ComplianceMetadata = make(map[string]interface{})
	}
	
	event.ComplianceMetadata["pci_requirement"] = as.getPCIRequirement(event.EventType)
	event.ComplianceMetadata["pci_data_classification"] = as.getPCIDataClassification(event)
}

// Helper functions for compliance metadata
func (as *AuditSystem) getSOC2ControlID(eventType EventType) string {
	switch eventType {
	case EventTypeAuthentication:
		return "CC6.1"
	case EventTypeAuthorization:
		return "CC6.2"
	case EventTypeDataAccess:
		return "CC6.7"
	default:
		return "CC1.4"
	}
}

func (as *AuditSystem) getSOC2TrustService(eventType EventType) string {
	switch eventType {
	case EventTypeAuthentication, EventTypeAuthorization:
		return "Security"
	case EventTypeDataAccess:
		return "Confidentiality"
	case EventTypeSystemChange:
		return "Processing Integrity"
	default:
		return "Security"
	}
}

func (as *AuditSystem) getISO27001Control(eventType EventType) string {
	switch eventType {
	case EventTypeAuthentication:
		return "A.9.2.1"
	case EventTypeAuthorization:
		return "A.9.2.2"
	case EventTypeDataAccess:
		return "A.12.4.1"
	default:
		return "A.12.1.1"
	}
}

func (as *AuditSystem) getISO27001Annex(eventType EventType) string {
	switch eventType {
	case EventTypeAuthentication, EventTypeAuthorization:
		return "A.9 - Access Control"
	case EventTypeDataAccess:
		return "A.12 - Operations Security"
	default:
		return "A.12 - Operations Security"
	}
}

func (as *AuditSystem) getPCIRequirement(eventType EventType) string {
	switch eventType {
	case EventTypeAuthentication:
		return "8.1.1"
	case EventTypeAuthorization:
		return "7.1.1"
	case EventTypeDataAccess:
		return "10.2.1"
	default:
		return "10.1"
	}
}

func (as *AuditSystem) getPCIDataClassification(event *AuditEvent) string {
	// Check if event involves cardholder data
	if event.Data != nil {
		if _, exists := event.Data["cardholder_data"]; exists {
			return "Cardholder Data"
		}
		if _, exists := event.Data["sensitive_auth_data"]; exists {
			return "Sensitive Authentication Data"
		}
	}
	return "Non-CHD"
}

// AuditStats contains statistics about the audit system
type AuditStats struct {
	EventsReceived   int64                 `json:"events_received"`
	EventsDropped    int64                 `json:"events_dropped"`
	QueueSize        int                   `json:"queue_size"`
	BackendCount     int                   `json:"backend_count"`
	LastFlushTime    time.Time            `json:"last_flush_time"`
	IntegrityEnabled bool                 `json:"integrity_enabled"`
	ComplianceMode   []ComplianceStandard `json:"compliance_mode"`
}

// Helper functions for system metadata
func getHostname() string {
	// Implementation would get actual hostname
	return "nephoran-operator"
}

func getProcessID() int {
	// Implementation would get actual process ID
	return 1
}

func getGoroutineID() int {
	// Implementation would get goroutine ID
	return 1
}