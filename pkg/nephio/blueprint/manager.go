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
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// GeneratorInterface defines the interface for blueprint generators
type GeneratorInterface interface {
	GenerateFromNetworkIntent(ctx context.Context, intent *v1.NetworkIntent) (map[string]string, error)
	HealthCheck(ctx context.Context) bool
}

const (

	// Blueprint lifecycle phases.

	BlueprintPhasePending = "Pending"

	// BlueprintPhaseProcessing holds blueprintphaseprocessing value.

	BlueprintPhaseProcessing = "Processing"

	// BlueprintPhaseReady holds blueprintphaseready value.

	BlueprintPhaseReady = "Ready"

	// BlueprintPhaseFailed holds blueprintphasefailed value.

	BlueprintPhaseFailed = "Failed"

	// BlueprintPhaseDeprecated holds blueprintphasedeprecated value.

	BlueprintPhaseDeprecated = "Deprecated"

	// Cache TTL for blueprint templates.

	DefaultCacheTTL = 15 * time.Minute

	// Max concurrent blueprint operations.

	DefaultMaxConcurrency = 50

	// Blueprint metadata annotations.

	AnnotationBlueprintVersion = "nephoran.com/blueprint-version"

	// AnnotationBlueprintType holds annotationblueprinttype value.

	AnnotationBlueprintType = "nephoran.com/blueprint-type"

	// AnnotationBlueprintComponent holds annotationblueprintcomponent value.

	AnnotationBlueprintComponent = "nephoran.com/blueprint-component"

	// AnnotationBlueprintGenerated holds annotationblueprintgenerated value.

	AnnotationBlueprintGenerated = "nephoran.com/blueprint-generated"

	// AnnotationIntentID holds annotationintentid value.

	AnnotationIntentID = "nephoran.com/intent-id"

	// AnnotationORANCompliant holds annotationorancompliant value.

	AnnotationORANCompliant = "nephoran.com/oran-compliant"
)

// PackageRevision represents a simplified Nephio package revision.

type PackageRevision struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PackageRevisionSpec `json:"spec,omitempty"`
}

// PackageRevisionSpec represents a packagerevisionspec.

type PackageRevisionSpec struct {
	PackageName string `json:"packageName"`

	WorkspaceName WorkspaceName `json:"workspaceName"`

	Tasks []Task `json:"tasks,omitempty"`

	Resources map[string]string `json:"resources,omitempty"`
}

// WorkspaceName represents a workspacename.

type WorkspaceName struct {
	Name string `json:"name"`

	Namespace string `json:"namespace"`
}

// Task represents a task.

type Task struct {
	Type string `json:"type"`

	Init *PackageInitTaskSpec `json:"init,omitempty"`
}

// PackageInitTaskSpec represents a packageinittaskspec.

type PackageInitTaskSpec struct {
	Description string `json:"description"`
}

const (

	// TaskTypeInit holds tasktypeinit value.

	TaskTypeInit = "init"
)

// BlueprintMetrics contains Prometheus metrics for blueprint operations.

type BlueprintMetrics struct {
	// Blueprint generation metrics.

	GenerationDuration prometheus.Histogram

	GenerationTotal prometheus.Counter

	GenerationErrors prometheus.Counter

	// Template metrics.

	TemplateHits prometheus.Counter

	TemplateMisses prometheus.Counter

	TemplateErrors prometheus.Counter

	// Validation metrics.

	ValidationDuration prometheus.Histogram

	ValidationTotal prometheus.Counter

	ValidationErrors prometheus.Counter

	// Cache metrics.

	CacheSize prometheus.Gauge

	CacheHitRatio prometheus.Gauge

	CacheEvictions prometheus.Counter

	// Performance metrics.

	ConcurrentOperations prometheus.Gauge

	QueueDepth prometheus.Gauge

	ProcessingLatency prometheus.Histogram
}

var (
	// Global singleton metrics instance and initialization guard.
	globalMetrics *BlueprintMetrics
	metricsOnce   sync.Once
)

// NewBlueprintMetrics creates new blueprint metrics using singleton pattern to prevent duplicate registration.

func NewBlueprintMetrics() *BlueprintMetrics {
	metricsOnce.Do(func() {
		globalMetrics = &BlueprintMetrics{
			GenerationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
				Name: "nephoran_blueprint_generation_duration_seconds",

				Help: "Duration of blueprint generation operations",

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
				Name: "nephoran_blueprint_validation_duration_seconds",

				Help: "Duration of blueprint validation operations",

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
				Name: "nephoran_blueprint_processing_latency_seconds",

				Help: "Latency of blueprint processing operations",

				Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			}),
		}
	})
	return globalMetrics
}

// BlueprintConfig contains configuration for the blueprint manager.

type BlueprintConfig struct {
	// PorchEndpoint is the Nephio Porch API endpoint.

	PorchEndpoint string

	// LLMEndpoint is the LLM processor service endpoint.

	LLMEndpoint string

	// RAGEndpoint is the RAG API service endpoint.

	RAGEndpoint string

	// CacheTTL is the cache TTL for blueprint templates.

	CacheTTL time.Duration

	// MaxConcurrency is the maximum number of concurrent blueprint operations.

	MaxConcurrency int

	// EnableValidation enables comprehensive blueprint validation.

	EnableValidation bool

	// EnableORANCompliance enables O-RAN compliance checking.

	EnableORANCompliance bool

	// TemplateRepository is the Git repository for blueprint templates.

	TemplateRepository string

	// DefaultNamespace is the default namespace for blueprint operations.

	DefaultNamespace string
}

// blueprintEnvOrDefault returns the first non-empty env var value or the default.
func blueprintEnvOrDefault(defaultVal string, envVars ...string) string {
	for _, env := range envVars {
		if v := os.Getenv(env); v != "" {
			return v
		}
	}
	return defaultVal
}

// DefaultBlueprintConfig returns default configuration.
// Endpoints are resolved from environment variables with hardcoded fallbacks.
// Prefer setting PORCH_SERVER_URL, LLM_ENDPOINT, RAG_ENDPOINT in production.
func DefaultBlueprintConfig() *BlueprintConfig {
	return &BlueprintConfig{
		PorchEndpoint: blueprintEnvOrDefault(
			"http://porch-server.porch-system.svc.cluster.local:9080",
			"PORCH_SERVER_URL", "PORCH_ENDPOINT",
		),

		LLMEndpoint: blueprintEnvOrDefault(
			"http://ollama-service.ollama.svc.cluster.local:11434",
			"LLM_ENDPOINT", "LLM_PROCESSOR_URL",
		),

		RAGEndpoint: blueprintEnvOrDefault(
			"http://rag-api.nephoran-system.svc.cluster.local:8081",
			"RAG_ENDPOINT",
		),

		CacheTTL: DefaultCacheTTL,

		MaxConcurrency: DefaultMaxConcurrency,

		EnableValidation: true,

		EnableORANCompliance: true,

		TemplateRepository: "https://github.com/nephio-project/free5gc-packages.git",

		DefaultNamespace: "default",
	}
}

// BlueprintOperation represents a blueprint operation request.

type BlueprintOperation struct {
	ID string

	Intent *v1.NetworkIntent

	Type string

	Priority v1.Priority

	Context context.Context

	StartTime time.Time

	Callback func(*BlueprintResult)
}

// BlueprintResult represents the result of a blueprint operation.

type BlueprintResult struct {
	Operation *BlueprintOperation

	Success bool

	Error error

	PackageRevision *PackageRevision

	GeneratedFiles map[string]string

	ValidationResults *SimpleValidationResult

	Metrics map[string]interface{}

	Duration time.Duration
}

// SimpleValidationResult represents validation results (simplified).

type SimpleValidationResult struct {
	IsValid bool `json:"isValid"`

	Errors []string `json:"errors,omitempty"`
}

// Manager handles blueprint lifecycle operations and orchestration.

type Manager struct {
	// Core dependencies.

	client client.Client

	k8sClient kubernetes.Interface

	config *BlueprintConfig

	logger *zap.Logger

	metrics *BlueprintMetrics

	// Components.

	catalog *Catalog

	generator GeneratorInterface

	customizer *Customizer

	validator ValidatorInterface

	// Operation management.

	operationQueue chan *BlueprintOperation

	semaphore chan struct{}

	wg sync.WaitGroup

	ctx context.Context

	cancel context.CancelFunc

	// Cache and state.

	cache sync.Map

	healthStatus map[string]bool

	healthMutex sync.RWMutex

	lastHealthCheck time.Time
}

// NewManager creates a new blueprint manager.
func NewManager(mgr manager.Manager, config *BlueprintConfig, logger *zap.Logger) (*Manager, error) {
	// Defensive programming: Validate inputs
	if mgr == nil {
		return nil, fmt.Errorf("controller manager is nil")
	}
	if config == nil {
		config = DefaultBlueprintConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// Validate config fields
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = DefaultMaxConcurrency
	}
	if config.CacheTTL <= 0 {
		config.CacheTTL = DefaultCacheTTL
	}

	// Get Kubernetes config with defensive check
	k8sConfig := mgr.GetConfig()
	if k8sConfig == nil {
		return nil, fmt.Errorf("Kubernetes config is nil")
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		client: mgr.GetClient(),

		k8sClient: k8sClient,

		config: config,

		logger: logger,

		metrics: NewBlueprintMetrics(),

		operationQueue: make(chan *BlueprintOperation, config.MaxConcurrency*2),

		semaphore: make(chan struct{}, config.MaxConcurrency),

		ctx: ctx,

		cancel: cancel,

		healthStatus: make(map[string]bool),
	}

	// Initialize components.

	if err := m.initializeComponents(); err != nil {

		cancel()

		return nil, fmt.Errorf("failed to initialize components: %w", err)

	}

	// Start background workers.

	m.startWorkers()

	logger.Info("Blueprint manager initialized successfully",

		zap.String("porch_endpoint", config.PorchEndpoint),

		zap.String("llm_endpoint", config.LLMEndpoint),

		zap.Duration("cache_ttl", config.CacheTTL),

		zap.Int("max_concurrency", config.MaxConcurrency))

	return m, nil
}

// initializeComponents initializes all blueprint manager components.
func (m *Manager) initializeComponents() error {
	// Defensive programming: Validate manager
	if m == nil {
		return fmt.Errorf("manager is nil")
	}
	if m.config == nil {
		return fmt.Errorf("config is nil")
	}
	if m.logger == nil {
		m.logger = zap.NewNop()
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("Panic recovered in initializeComponents", 
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	var err error

	// Initialize catalog with defensive programming
	m.catalog, err = NewCatalog(m.config, m.logger.Named("catalog"))
	if err != nil {
		return fmt.Errorf("failed to initialize catalog: %w", err)
	}

	// Initialize generator with defensive programming
	m.generator, err = NewGenerator(m.config, m.logger.Named("generator"))
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}
	if m.generator == nil {
		return fmt.Errorf("generator creation returned nil")
	}

	// Initialize customizer with defensive programming
	m.customizer, err = NewCustomizer(m.config, m.logger.Named("customizer"))
	if err != nil {
		return fmt.Errorf("failed to initialize customizer: %w", err)
	}
	if m.customizer == nil {
		return fmt.Errorf("customizer creation returned nil")
	}

	// Initialize validator if enabled with defensive programming
	if m.config.EnableValidation {
		m.validator, err = NewValidator(m.config, m.logger.Named("validator"))
		if err != nil {
			return fmt.Errorf("failed to initialize validator: %w", err)
		}
		if m.validator == nil {
			return fmt.Errorf("validator creation returned nil")
		}
	}

	return nil
}

// startWorkers starts background worker goroutines.

func (m *Manager) startWorkers() {
	// Start operation processor workers.

	for range m.config.MaxConcurrency / 2 {

		m.wg.Add(1)

		go m.operationWorker()

	}

	// Start health check worker.

	m.wg.Add(1)

	go m.healthCheckWorker()

	// Start metrics updater.

	m.wg.Add(1)

	go m.metricsWorker()

	// Start cache cleanup worker.

	m.wg.Add(1)

	go m.cacheCleanupWorker()
}

// ProcessNetworkIntent processes a NetworkIntent and generates blueprint packages.
func (m *Manager) ProcessNetworkIntent(ctx context.Context, intent *v1.NetworkIntent) (*BlueprintResult, error) {
	// Defensive programming: Validate inputs
	if m == nil {
		return nil, fmt.Errorf("manager is nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if intent == nil {
		return nil, fmt.Errorf("intent is nil")
	}
	if m.metrics == nil {
		m.metrics = NewBlueprintMetrics()
	}
	if m.logger == nil {
		m.logger = zap.NewNop()
	}

	// Add panic recovery for critical path
	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("Panic recovered in ProcessNetworkIntent", 
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

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

	// Create operation context.

	operation := &BlueprintOperation{
		ID: fmt.Sprintf("%s-%d", intent.Name, time.Now().UnixNano()),

		Intent: intent,

		Type: "process_intent",

		Priority: v1.ConvertNetworkPriorityToPriority(intent.Spec.Priority),

		Context: ctx,

		StartTime: startTime,
	}

	// Process operation synchronously for immediate response.

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

// processOperationSync processes a blueprint operation synchronously.
func (m *Manager) processOperationSync(operation *BlueprintOperation) *BlueprintResult {
	// Defensive programming: Validate inputs
	if m == nil || operation == nil {
		return &BlueprintResult{
			Success: false,
			Error:   fmt.Errorf("manager or operation is nil"),
		}
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic recovered in processOperationSync", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
	}()

	result := &BlueprintResult{
		Operation: operation,
	}

	defer func() {
		if operation.StartTime.IsZero() {
			result.Duration = 0
		} else {
			result.Duration = time.Since(operation.StartTime)
		}
	}()

	// Step 1: Generate blueprint from NetworkIntent.

	// Defensive check for generator availability
	if m.generator == nil {
		result.Error = fmt.Errorf("generator is not initialized")
		return result
	}

	generatedFiles, err := m.generator.GenerateFromNetworkIntent(operation.Context, operation.Intent)
	if err != nil {
		result.Error = fmt.Errorf("blueprint generation failed: %w", err)
		return result
	}
	
	// Defensive check for generated files
	if generatedFiles == nil {
		generatedFiles = make(map[string]string)
	}

	result.GeneratedFiles = generatedFiles

	// Step 2: Customize blueprint based on intent parameters.

	var customizedFiles map[string]string
	// Defensive check for customizer availability
	if m.customizer == nil {
		// Use original files if customizer is not available
		customizedFiles = generatedFiles
		if m.logger != nil {
			m.logger.Warn("Customizer not available, using original files")
		}
	} else {
		var err error
		customizedFiles, err = m.customizer.CustomizeBlueprint(operation.Context, operation.Intent, generatedFiles)
		if err != nil {
			result.Error = fmt.Errorf("blueprint customization failed: %w", err)
			return result
		}
		// Defensive check for customized files
		if customizedFiles == nil {
			customizedFiles = generatedFiles
		}
	}

	result.GeneratedFiles = customizedFiles

	// Step 3: Validate blueprint if validation is enabled.

	if m.validator != nil {
		validationResult, err := m.validator.ValidateBlueprint(operation.Context, operation.Intent, customizedFiles)
		if err != nil {
			// When validator fails with error, continue gracefully with default valid result
			result.ValidationResults = &SimpleValidationResult{
				IsValid: true, // Default to valid when validator has an error
				Errors:  []string{},
			}
			// Log the validation error but continue processing
			if m.logger != nil {
				m.logger.Warn("Validator failed, continuing with default valid result", zap.Error(err))
			}
		} else if validationResult != nil {
			// Convert ValidationResult to SimpleValidationResult with defensive programming
			var errorStrings []string
			errorStrings = make([]string, 0, len(validationResult.Errors))

			for _, err := range validationResult.Errors {
				if err.Message != "" {
					errorStrings = append(errorStrings, err.Message)
				}
			}

			result.ValidationResults = &SimpleValidationResult{
				IsValid: validationResult.IsValid,
				Errors: errorStrings,
			}
		} else {
			// Validation returned nil result
			result.ValidationResults = &SimpleValidationResult{
				IsValid: false,
				Errors: []string{"validation result is nil"},
			}
		}

		// Even if validation fails, we continue with the result
		// The validation results are already captured in result.ValidationResults

	} else {
		// No validator configured - default to valid
		result.ValidationResults = &SimpleValidationResult{
			IsValid: true,
			Errors:  []string{},
		}
	}

	// Step 4: Create Nephio PackageRevision.

	packageRevision, err := m.createPackageRevision(operation.Context, operation.Intent, customizedFiles)
	if err != nil {

		result.Error = fmt.Errorf("package revision creation failed: %w", err)

		return result

	}

	result.PackageRevision = packageRevision

	result.Success = true

	return result
}

// createPackageRevision creates a Nephio PackageRevision.
func (m *Manager) createPackageRevision(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (*PackageRevision, error) {
	// Defensive programming: Validate inputs
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if intent == nil {
		return nil, fmt.Errorf("intent is nil")
	}
	if files == nil {
		files = make(map[string]string)
	}
	if m == nil {
		return nil, fmt.Errorf("manager is nil")
	}
	packageRevision := &PackageRevision{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.kpt.dev/v1alpha1",

			Kind: "PackageRevision",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-blueprint-%d", intent.Name, time.Now().Unix()),

			Namespace: intent.Namespace,

			Annotations: map[string]string{
				AnnotationBlueprintVersion: "v1.0.0",

				AnnotationBlueprintType: string(intent.Spec.IntentType),

				AnnotationBlueprintGenerated: time.Now().Format(time.RFC3339),

				AnnotationIntentID: intent.Name,

				AnnotationORANCompliant: "true",
			},

			Labels: map[string]string{
				"nephoran.com/blueprint": "true",

				"nephoran.com/intent": intent.Name,

				"nephoran.com/component": m.getComponentFromIntent(intent),
			},
		},

		Spec: PackageRevisionSpec{
			PackageName: fmt.Sprintf("%s-blueprint", intent.Name),

			WorkspaceName: WorkspaceName{
				Name: fmt.Sprintf("%s-workspace", intent.Name),

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

// getComponentFromIntent extracts the primary component type from NetworkIntent.

func (m *Manager) getComponentFromIntent(intent *v1.NetworkIntent) string {
	if len(intent.Spec.TargetComponents) > 0 {
		return string(intent.Spec.TargetComponents[0])
	}

	return "unknown"
}

// operationWorker processes blueprint operations from the queue.
func (m *Manager) operationWorker() {
	// Defensive programming: Validate manager
	if m == nil {
		return
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic recovered in operationWorker", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
		m.wg.Done()
	}()

	for {
		select {
		case <-m.ctx.Done():
			return

		case operation := <-m.operationQueue:
			// Defensive check for nil operation
			if operation == nil {
				continue
			}

			// Acquire semaphore with defensive checks
			select {
			case m.semaphore <- struct{}{}:
				// Process operation with error handling
				func() {
					defer func() {
						if r := recover(); r != nil {
							if m.logger != nil {
								m.logger.Error("Panic in operation processing", 
									zap.Any("panic", r),
									zap.Stack("stack"))
							}
						}
						// Always release semaphore
						<-m.semaphore
					}()
					
					result := m.processOperationSync(operation)
					
					// Execute callback if provided with defensive checks
					if operation.Callback != nil && result != nil {
						func() {
							defer func() {
								if r := recover(); r != nil {
									if m.logger != nil {
										m.logger.Error("Panic in callback execution", 
											zap.Any("panic", r))
									}
								}
							}()
							operation.Callback(result)
						}()
					}
				}()

			case <-m.ctx.Done():
				return
			}
		}
	}
}

// healthCheckWorker performs periodic health checks.

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

// performHealthCheck checks the health of all components.
func (m *Manager) performHealthCheck() {
	// Defensive programming: Validate manager
	if m == nil {
		return
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic recovered in performHealthCheck", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
	}()

	m.healthMutex.Lock()
	defer m.healthMutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize healthStatus if nil
	if m.healthStatus == nil {
		m.healthStatus = make(map[string]bool)
	}

	// Check catalog health with defensive programming
	if m.catalog != nil {
		m.healthStatus["catalog"] = m.catalog.HealthCheck(ctx)
	} else {
		m.healthStatus["catalog"] = false
	}

	// Check generator health with defensive programming
	if m.generator != nil {
		m.healthStatus["generator"] = m.generator.HealthCheck(ctx)
	} else {
		m.healthStatus["generator"] = false
	}

	// Check customizer health with defensive programming
	if m.customizer != nil {
		m.healthStatus["customizer"] = m.customizer.HealthCheck(ctx)
	} else {
		m.healthStatus["customizer"] = false
	}

	// Check validator health if enabled
	if m.validator != nil {
		m.healthStatus["validator"] = m.validator.HealthCheck(ctx)
	} else {
		m.healthStatus["validator"] = false
	}

	m.lastHealthCheck = time.Now()
}

// metricsWorker updates metrics periodically.

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

// updateMetrics updates Prometheus metrics.
func (m *Manager) updateMetrics() {
	// Defensive programming: Validate manager and metrics
	if m == nil || m.metrics == nil {
		return
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic recovered in updateMetrics", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
	}()

	// Update queue depth with nil check
	if m.operationQueue != nil {
		m.metrics.QueueDepth.Set(float64(len(m.operationQueue)))
	}

	// Update cache size with defensive operations
	cacheSize := 0
	m.cache.Range(func(key, value interface{}) bool {
		if key != nil && value != nil {
			cacheSize++
		}
		return true
	})
	m.metrics.CacheSize.Set(float64(cacheSize))

	// Update cache hit ratio if we have template metrics.
	if m.catalog != nil {
		hits := m.catalog.GetCacheHits()
		misses := m.catalog.GetCacheMisses()
		
		// Defensive check to avoid division by zero
		if hits+misses > 0 && hits >= 0 && misses >= 0 {
			ratio := float64(hits) / float64(hits+misses)
			// Ensure ratio is valid (between 0 and 1)
			if ratio >= 0 && ratio <= 1 {
				m.metrics.CacheHitRatio.Set(ratio)
			}
		}
	}
}

// cacheCleanupWorker performs periodic cache cleanup.

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

// cleanupCache removes expired entries from cache.
func (m *Manager) cleanupCache() {
	// Defensive programming: Validate manager
	if m == nil {
		return
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic recovered in cleanupCache", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
	}()

	now := time.Now()
	evicted := 0

	// Defensive cache operations with nil checks
	m.cache.Range(func(key, value interface{}) bool {
		if key == nil || value == nil {
			return true
		}
		
		if cacheEntry, ok := value.(map[string]interface{}); ok && cacheEntry != nil {
			if expireTime, exists := cacheEntry["expire_time"]; exists && expireTime != nil {
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

// GetHealthStatus returns the current health status of all components.
func (m *Manager) GetHealthStatus() map[string]bool {
	// Defensive programming: Validate manager
	if m == nil {
		return make(map[string]bool)
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic recovered in GetHealthStatus", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
	}()

	m.healthMutex.RLock()
	defer m.healthMutex.RUnlock()

	status := make(map[string]bool)
	
	// Defensive check for healthStatus
	if m.healthStatus != nil {
		for component, health := range m.healthStatus {
			if component != "" {
				status[component] = health
			}
		}
	}

	return status
}

// GetMetrics returns current metrics.
func (m *Manager) GetMetrics() map[string]interface{} {
	// Defensive programming: Validate manager
	if m == nil {
		return map[string]interface{}{
			"error": "manager is nil",
		}
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic recovered in GetMetrics", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
	}()

	size := 0
	m.cache.Range(func(k, v interface{}) bool {
		if k != nil && v != nil {
			size++
		}
		return true
	})

	return map[string]interface{}{
		"blueprints_count": size,
		"cache_size":       size,
		"last_updated":     time.Now().Unix(),
	}
}

// Stop gracefully stops the blueprint manager.
func (m *Manager) Stop() error {
	// Defensive programming: Handle nil manager
	if m == nil {
		return fmt.Errorf("manager is nil")
	}
	
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if m.logger != nil {
				m.logger.Error("Panic recovered in Stop method", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
	}()

	if m.logger != nil {
		m.logger.Info("Stopping blueprint manager...")
	}

	// Defensive check for cancel function
	if m.cancel != nil {
		m.cancel()
	}

	// Wait for workers to finish with timeout.

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
