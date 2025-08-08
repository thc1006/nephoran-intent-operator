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

package blueprint

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/thc1006/nephoran-intent-operator/api/v1"
)

const (
	// Blueprint lifecycle phases
	BlueprintPhasePending    = "Pending"
	BlueprintPhaseProcessing = "Processing"
	BlueprintPhaseReady      = "Ready"
	BlueprintPhaseFailed     = "Failed"
	BlueprintPhaseDeprecated = "Deprecated"

	// Cache TTL for blueprint templates
	DefaultCacheTTL = 15 * time.Minute

	// Max concurrent blueprint operations
	DefaultMaxConcurrency = 50

	// Blueprint metadata annotations
	AnnotationBlueprintVersion    = "nephoran.com/blueprint-version"
	AnnotationBlueprintType       = "nephoran.com/blueprint-type"
	AnnotationBlueprintComponent  = "nephoran.com/blueprint-component"
	AnnotationBlueprintGenerated  = "nephoran.com/blueprint-generated"
	AnnotationIntentID            = "nephoran.com/intent-id"
	AnnotationORANCompliant       = "nephoran.com/oran-compliant"
)

// PackageRevision represents a simplified Nephio package revision
type PackageRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec PackageRevisionSpec `json:"spec,omitempty"`
}

type PackageRevisionSpec struct {
	PackageName   string            `json:"packageName"`
	WorkspaceName WorkspaceName     `json:"workspaceName"`
	Tasks         []Task            `json:"tasks,omitempty"`
	Resources     map[string]string `json:"resources,omitempty"`
}

type WorkspaceName struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Task struct {
	Type string          `json:"type"`
	Init *PackageInitTaskSpec `json:"init,omitempty"`
}

type PackageInitTaskSpec struct {
	Description string `json:"description"`
}

const (
	TaskTypeInit = "init"
)

// BlueprintMetrics contains Prometheus metrics for blueprint operations
type BlueprintMetrics struct {
	// Blueprint generation metrics
	GenerationDuration prometheus.Histogram
	GenerationTotal    prometheus.Counter
	GenerationErrors   prometheus.Counter

	// Template metrics
	TemplateHits    prometheus.Counter
	TemplateMisses  prometheus.Counter
	TemplateErrors  prometheus.Counter

	// Validation metrics
	ValidationDuration prometheus.Histogram
	ValidationTotal    prometheus.Counter
	ValidationErrors   prometheus.Counter

	// Cache metrics
	CacheSize      prometheus.Gauge
	CacheHitRatio  prometheus.Gauge
	CacheEvictions prometheus.Counter

	// Performance metrics
	ConcurrentOperations prometheus.Gauge
	QueueDepth           prometheus.Gauge
	ProcessingLatency    prometheus.Histogram
}

// NewBlueprintMetrics creates new blueprint metrics
func NewBlueprintMetrics() *BlueprintMetrics {
	return &BlueprintMetrics{
		GenerationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_blueprint_generation_duration_seconds",
			Help:    "Duration of blueprint generation operations",
			Buckets: prometheus.DefBuckets,
		}),
		GenerationTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_generation_total",
			Help: "Total number of blueprint generation operations",
		}),
		GenerationErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_generation_errors_total",
			Help: "Total number of blueprint generation errors",
		}),
		TemplateHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_template_hits_total",
			Help: "Total number of blueprint template cache hits",
		}),
		TemplateMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_template_misses_total",
			Help: "Total number of blueprint template cache misses",
		}),
		TemplateErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_template_errors_total",
			Help: "Total number of blueprint template errors",
		}),
		ValidationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_blueprint_validation_duration_seconds",
			Help:    "Duration of blueprint validation operations",
			Buckets: prometheus.DefBuckets,
		}),
		ValidationTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_validation_total",
			Help: "Total number of blueprint validation operations",
		}),
		ValidationErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_validation_errors_total",
			Help: "Total number of blueprint validation errors",
		}),
		CacheSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_blueprint_cache_size",
			Help: "Current size of blueprint template cache",
		}),
		CacheHitRatio: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_blueprint_cache_hit_ratio",
			Help: "Cache hit ratio for blueprint templates",
		}),
		CacheEvictions: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_blueprint_cache_evictions_total",
			Help: "Total number of blueprint template cache evictions",
		}),
		ConcurrentOperations: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_blueprint_concurrent_operations",
			Help: "Current number of concurrent blueprint operations",
		}),
		QueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_blueprint_queue_depth",
			Help: "Current depth of blueprint operation queue",
		}),
		ProcessingLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_blueprint_processing_latency_seconds",
			Help:    "Latency of blueprint processing operations",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}),
	}
}

// BlueprintConfig contains configuration for the blueprint manager
type BlueprintConfig struct {
	// PorchEndpoint is the Nephio Porch API endpoint
	PorchEndpoint string

	// LLMEndpoint is the LLM processor service endpoint
	LLMEndpoint string

	// RAGEndpoint is the RAG API service endpoint
	RAGEndpoint string

	// CacheTTL is the cache TTL for blueprint templates
	CacheTTL time.Duration

	// MaxConcurrency is the maximum number of concurrent blueprint operations
	MaxConcurrency int

	// EnableValidation enables comprehensive blueprint validation
	EnableValidation bool

	// EnableORANCompliance enables O-RAN compliance checking
	EnableORANCompliance bool

	// TemplateRepository is the Git repository for blueprint templates
	TemplateRepository string

	// DefaultNamespace is the default namespace for blueprint operations
	DefaultNamespace string
}

// DefaultBlueprintConfig returns default configuration
func DefaultBlueprintConfig() *BlueprintConfig {
	return &BlueprintConfig{
		PorchEndpoint:        "http://porch-server.porch-system.svc.cluster.local:9080",
		LLMEndpoint:          "http://llm-processor.nephoran-system.svc.cluster.local:8080",
		RAGEndpoint:          "http://rag-api.nephoran-system.svc.cluster.local:8081",
		CacheTTL:             DefaultCacheTTL,
		MaxConcurrency:       DefaultMaxConcurrency,
		EnableValidation:     true,
		EnableORANCompliance: true,
		TemplateRepository:   "https://github.com/nephio-project/free5gc-packages.git",
		DefaultNamespace:     "default",
	}
}

// BlueprintOperation represents a blueprint operation request
type BlueprintOperation struct {
	ID        string
	Intent    *v1.NetworkIntent
	Type      string
	Priority  v1.Priority
	Context   context.Context
	StartTime time.Time
	Callback  func(*BlueprintResult)
}

// BlueprintResult represents the result of a blueprint operation
type BlueprintResult struct {
	Operation         *BlueprintOperation
	Success           bool
	Error             error
	PackageRevision   *PackageRevision
	GeneratedFiles    map[string]string
	ValidationResults *ValidationResult
	Metrics           map[string]interface{}
	Duration          time.Duration
}

// ValidationResult represents validation results (simplified)
type ValidationResult struct {
	IsValid bool     `json:"isValid"`
	Errors  []string `json:"errors,omitempty"`
}

// Manager handles blueprint lifecycle operations and orchestration
type Manager struct {
	// Core dependencies
	client     client.Client
	k8sClient  kubernetes.Interface
	config     *BlueprintConfig
	logger     *zap.Logger
	metrics    *BlueprintMetrics

	// Components
	catalog     *Catalog
	generator   *Generator
	customizer  *Customizer
	validator   *Validator

	// Operation management
	operationQueue chan *BlueprintOperation
	semaphore      chan struct{}
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc

	// Cache and state
	cache          sync.Map
	healthStatus   map[string]bool
	healthMutex    sync.RWMutex
	lastHealthCheck time.Time
}

// NewManager creates a new blueprint manager
func NewManager(mgr manager.Manager, config *BlueprintConfig, logger *zap.Logger) (*Manager, error) {
	if config == nil {
		config = DefaultBlueprintConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	k8sClient, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		client:         mgr.GetClient(),
		k8sClient:      k8sClient,
		config:         config,
		logger:         logger,
		metrics:        NewBlueprintMetrics(),
		operationQueue: make(chan *BlueprintOperation, config.MaxConcurrency*2),
		semaphore:      make(chan struct{}, config.MaxConcurrency),
		ctx:            ctx,
		cancel:         cancel,
		healthStatus:   make(map[string]bool),
	}

	// Initialize components
	if err := m.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Start background workers
	m.startWorkers()

	logger.Info("Blueprint manager initialized successfully",
		zap.String("porch_endpoint", config.PorchEndpoint),
		zap.String("llm_endpoint", config.LLMEndpoint),
		zap.Duration("cache_ttl", config.CacheTTL),
		zap.Int("max_concurrency", config.MaxConcurrency))

	return m, nil
}

// initializeComponents initializes all blueprint manager components
func (m *Manager) initializeComponents() error {
	var err error

	// Initialize catalog
	m.catalog, err = NewCatalog(m.config, m.logger.Named("catalog"))
	if err != nil {
		return fmt.Errorf("failed to initialize catalog: %w", err)
	}

	// Initialize generator
	m.generator, err = NewGenerator(m.config, m.logger.Named("generator"))
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Initialize customizer
	m.customizer, err = NewCustomizer(m.config, m.logger.Named("customizer"))
	if err != nil {
		return fmt.Errorf("failed to initialize customizer: %w", err)
	}

	// Initialize validator if enabled
	if m.config.EnableValidation {
		m.validator, err = NewValidator(m.config, m.logger.Named("validator"))
		if err != nil {
			return fmt.Errorf("failed to initialize validator: %w", err)
		}
	}

	return nil
}

// startWorkers starts background worker goroutines
func (m *Manager) startWorkers() {
	// Start operation processor workers
	for i := 0; i < m.config.MaxConcurrency/2; i++ {
		m.wg.Add(1)
		go m.operationWorker()
	}

	// Start health check worker
	m.wg.Add(1)
	go m.healthCheckWorker()

	// Start metrics updater
	m.wg.Add(1)
	go m.metricsWorker()

	// Start cache cleanup worker
	m.wg.Add(1)
	go m.cacheCleanupWorker()
}

// ProcessNetworkIntent processes a NetworkIntent and generates blueprint packages
func (m *Manager) ProcessNetworkIntent(ctx context.Context, intent *v1.NetworkIntent) (*BlueprintResult, error) {
	startTime := time.Now()
	m.metrics.GenerationTotal.Inc()
	m.metrics.ConcurrentOperations.Inc()
	defer m.metrics.ConcurrentOperations.Dec()

	defer func() {
		duration := time.Since(startTime)
		m.metrics.GenerationDuration.Observe(duration.Seconds())
		m.metrics.ProcessingLatency.Observe(duration.Seconds())
	}()

	m.logger.Info("Processing NetworkIntent for blueprint generation",
		zap.String("intent_name", intent.Name),
		zap.String("intent_type", string(intent.Spec.IntentType)),
		zap.String("priority", string(intent.Spec.Priority)))

	// Create operation context
	operation := &BlueprintOperation{
		ID:        fmt.Sprintf("%s-%d", intent.Name, time.Now().UnixNano()),
		Intent:    intent,
		Type:      "process_intent",
		Priority:  intent.Spec.Priority,
		Context:   ctx,
		StartTime: startTime,
	}

	// Process operation synchronously for immediate response
	result := m.processOperationSync(operation)

	if result.Error != nil {
		m.metrics.GenerationErrors.Inc()
		m.logger.Error("Failed to process NetworkIntent",
			zap.String("intent_name", intent.Name),
			zap.Error(result.Error))
	} else {
		m.logger.Info("Successfully processed NetworkIntent",
			zap.String("intent_name", intent.Name),
			zap.Duration("duration", result.Duration),
			zap.Int("generated_files", len(result.GeneratedFiles)))
	}

	return result, result.Error
}

// processOperationSync processes a blueprint operation synchronously
func (m *Manager) processOperationSync(operation *BlueprintOperation) *BlueprintResult {
	result := &BlueprintResult{
		Operation: operation,
	}

	defer func() {
		result.Duration = time.Since(operation.StartTime)
	}()

	// Step 1: Generate blueprint from NetworkIntent
	generatedFiles, err := m.generator.GenerateFromNetworkIntent(operation.Context, operation.Intent)
	if err != nil {
		result.Error = fmt.Errorf("blueprint generation failed: %w", err)
		return result
	}
	result.GeneratedFiles = generatedFiles

	// Step 2: Customize blueprint based on intent parameters
	customizedFiles, err := m.customizer.CustomizeBlueprint(operation.Context, operation.Intent, generatedFiles)
	if err != nil {
		result.Error = fmt.Errorf("blueprint customization failed: %w", err)
		return result
	}
	result.GeneratedFiles = customizedFiles

	// Step 3: Validate blueprint if validation is enabled
	if m.validator != nil {
		validationResult, err := m.validator.ValidateBlueprint(operation.Context, operation.Intent, customizedFiles)
		if err != nil {
			result.Error = fmt.Errorf("blueprint validation failed: %w", err)
			return result
		}
		result.ValidationResults = validationResult

		if !validationResult.IsValid {
			result.Error = fmt.Errorf("blueprint validation failed: %v", validationResult.Errors)
			return result
		}
	}

	// Step 4: Create Nephio PackageRevision
	packageRevision, err := m.createPackageRevision(operation.Context, operation.Intent, customizedFiles)
	if err != nil {
		result.Error = fmt.Errorf("package revision creation failed: %w", err)
		return result
	}
	result.PackageRevision = packageRevision

	result.Success = true
	return result
}

// createPackageRevision creates a Nephio PackageRevision
func (m *Manager) createPackageRevision(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (*PackageRevision, error) {
	packageRevision := &PackageRevision{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.kpt.dev/v1alpha1",
			Kind:       "PackageRevision",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-blueprint-%d", intent.Name, time.Now().Unix()),
			Namespace: intent.Namespace,
			Annotations: map[string]string{
				AnnotationBlueprintVersion:   "v1.0.0",
				AnnotationBlueprintType:      string(intent.Spec.IntentType),
				AnnotationBlueprintGenerated: time.Now().Format(time.RFC3339),
				AnnotationIntentID:           intent.Name,
				AnnotationORANCompliant:      "true",
			},
			Labels: map[string]string{
				"nephoran.com/blueprint": "true",
				"nephoran.com/intent":    intent.Name,
				"nephoran.com/component": m.getComponentFromIntent(intent),
			},
		},
		Spec: PackageRevisionSpec{
			PackageName: fmt.Sprintf("%s-blueprint", intent.Name),
			WorkspaceName: WorkspaceName{
				Name:      fmt.Sprintf("%s-workspace", intent.Name),
				Namespace: intent.Namespace,
			},
			Tasks: []Task{
				{
					Type: TaskTypeInit,
					Init: &PackageInitTaskSpec{
						Description: fmt.Sprintf("Blueprint package for NetworkIntent: %s", intent.Name),
					},
				},
			},
			Resources: files,
		},
	}

	m.logger.Info("Created PackageRevision",
		zap.String("name", packageRevision.Name),
		zap.String("namespace", packageRevision.Namespace),
		zap.Int("files", len(files)))

	return packageRevision, nil
}

// getComponentFromIntent extracts the primary component type from NetworkIntent
func (m *Manager) getComponentFromIntent(intent *v1.NetworkIntent) string {
	if len(intent.Spec.TargetComponents) > 0 {
		return string(intent.Spec.TargetComponents[0])
	}
	return "unknown"
}

// operationWorker processes blueprint operations from the queue
func (m *Manager) operationWorker() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case operation := <-m.operationQueue:
			// Acquire semaphore
			select {
			case m.semaphore <- struct{}{}:
				// Process operation
				result := m.processOperationSync(operation)
				
				// Execute callback if provided
				if operation.Callback != nil {
					operation.Callback(result)
				}

				// Release semaphore
				<-m.semaphore

			case <-m.ctx.Done():
				return
			}
		}
	}
}

// healthCheckWorker performs periodic health checks
func (m *Manager) healthCheckWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// performHealthCheck checks the health of all components
func (m *Manager) performHealthCheck() {
	m.healthMutex.Lock()
	defer m.healthMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check catalog health
	m.healthStatus["catalog"] = m.catalog.HealthCheck(ctx)

	// Check generator health
	m.healthStatus["generator"] = m.generator.HealthCheck(ctx)

	// Check customizer health
	m.healthStatus["customizer"] = m.customizer.HealthCheck(ctx)

	// Check validator health if enabled
	if m.validator != nil {
		m.healthStatus["validator"] = m.validator.HealthCheck(ctx)
	}

	m.lastHealthCheck = time.Now()
}

// metricsWorker updates metrics periodically
func (m *Manager) metricsWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateMetrics()
		}
	}
}

// updateMetrics updates Prometheus metrics
func (m *Manager) updateMetrics() {
	// Update queue depth
	m.metrics.QueueDepth.Set(float64(len(m.operationQueue)))

	// Update cache size
	cacheSize := 0
	m.cache.Range(func(_, _ interface{}) bool {
		cacheSize++
		return true
	})
	m.metrics.CacheSize.Set(float64(cacheSize))

	// Update cache hit ratio if we have template metrics
	if m.catalog != nil {
		hits := m.catalog.GetCacheHits()
		misses := m.catalog.GetCacheMisses()
		if hits+misses > 0 {
			ratio := float64(hits) / float64(hits+misses)
			m.metrics.CacheHitRatio.Set(ratio)
		}
	}
}

// cacheCleanupWorker performs periodic cache cleanup
func (m *Manager) cacheCleanupWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CacheTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanupCache()
		}
	}
}

// cleanupCache removes expired entries from cache
func (m *Manager) cleanupCache() {
	now := time.Now()
	evicted := 0

	m.cache.Range(func(key, value interface{}) bool {
		if cacheEntry, ok := value.(map[string]interface{}); ok {
			if expireTime, exists := cacheEntry["expire_time"]; exists {
				if expTime, ok := expireTime.(time.Time); ok && now.After(expTime) {
					m.cache.Delete(key)
					evicted++
				}
			}
		}
		return true
	})

	if evicted > 0 {
		m.metrics.CacheEvictions.Add(float64(evicted))
		m.logger.Debug("Cleaned up expired cache entries", zap.Int("evicted", evicted))
	}
}

// GetHealthStatus returns the current health status of all components
func (m *Manager) GetHealthStatus() map[string]bool {
	m.healthMutex.RLock()
	defer m.healthMutex.RUnlock()

	status := make(map[string]bool)
	for component, health := range m.healthStatus {
		status[component] = health
	}
	
	return status
}

// GetMetrics returns current metrics
func (m *Manager) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"queue_depth":           len(m.operationQueue),
		"concurrent_operations": len(m.semaphore),
		"last_health_check":     m.lastHealthCheck,
		"cache_size": func() int {
			size := 0
			m.cache.Range(func(_, _ interface{}) bool {
				size++
				return true
			})
			return size
		}(),
	}
}

// Stop gracefully stops the blueprint manager
func (m *Manager) Stop() error {
	m.logger.Info("Stopping blueprint manager...")

	// Cancel context to stop workers
	m.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info("Blueprint manager stopped successfully")
	case <-time.After(30 * time.Second):
		m.logger.Warn("Blueprint manager stop timeout reached")
	}

	return nil
}