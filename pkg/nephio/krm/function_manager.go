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

package krm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/krm/functions"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// FunctionManager provides comprehensive KRM function management
type FunctionManager struct {
	config           *FunctionManagerConfig
	runtime          *Runtime
	containerRuntime *ContainerRuntime
	registry         *Registry
	functionCache    *FunctionCache
	metrics          *FunctionManagerMetrics
	tracer           trace.Tracer
	nativeFunctions  map[string]functions.KRMFunction
	mu               sync.RWMutex
}

// FunctionManagerConfig defines configuration for function management
type FunctionManagerConfig struct {
	// Function execution settings
	ExecutionMode      string        `json:"executionMode" yaml:"executionMode"`           // native, container, hybrid
	DefaultTimeout     time.Duration `json:"defaultTimeout" yaml:"defaultTimeout"`         // Default function timeout
	MaxConcurrentExecs int           `json:"maxConcurrentExecs" yaml:"maxConcurrentExecs"` // Max concurrent executions

	// Native function settings
	EnableNativeFunctions bool     `json:"enableNativeFunctions" yaml:"enableNativeFunctions"` // Enable native functions
	NativeFunctionPaths   []string `json:"nativeFunctionPaths" yaml:"nativeFunctionPaths"`     // Paths to native functions

	// Container function settings
	EnableContainerFunctions bool   `json:"enableContainerFunctions" yaml:"enableContainerFunctions"` // Enable container functions
	DefaultRegistry          string `json:"defaultRegistry" yaml:"defaultRegistry"`                   // Default container registry
	ImagePullSecret          string `json:"imagePullSecret" yaml:"imagePullSecret"`                   // Image pull secret

	// Caching settings
	EnableFunctionCache bool          `json:"enableFunctionCache" yaml:"enableFunctionCache"` // Enable function caching
	CacheSize           int64         `json:"cacheSize" yaml:"cacheSize"`                     // Cache size in bytes
	CacheTTL            time.Duration `json:"cacheTtl" yaml:"cacheTtl"`                       // Cache TTL

	// Security settings
	EnableSandboxing  bool     `json:"enableSandboxing" yaml:"enableSandboxing"`   // Enable sandboxing
	AllowedRegistries []string `json:"allowedRegistries" yaml:"allowedRegistries"` // Allowed registries
	TrustedImages     []string `json:"trustedImages" yaml:"trustedImages"`         // Trusted images

	// Observability settings
	EnableMetrics   bool `json:"enableMetrics" yaml:"enableMetrics"`     // Enable metrics
	EnableTracing   bool `json:"enableTracing" yaml:"enableTracing"`     // Enable tracing
	EnableProfiling bool `json:"enableProfiling" yaml:"enableProfiling"` // Enable profiling
	DetailedLogging bool `json:"detailedLogging" yaml:"detailedLogging"` // Enable detailed logging
}

// FunctionExecutionRequest represents a function execution request
type FunctionExecutionRequest struct {
	// Function identification
	FunctionName   string                 `json:"functionName"`
	FunctionImage  string                 `json:"functionImage,omitempty"`
	FunctionConfig map[string]interface{} `json:"functionConfig,omitempty"`

	// Input resources
	Resources    []*porch.KRMResource    `json:"resources"`
	ResourceList *functions.ResourceList `json:"resourceList,omitempty"`

	// Execution context
	Context       *functions.ExecutionContext `json:"context,omitempty"`
	Timeout       time.Duration               `json:"timeout,omitempty"`
	ExecutionMode string                      `json:"executionMode,omitempty"`

	// Options
	EnableCaching   bool `json:"enableCaching,omitempty"`
	EnableProfiling bool `json:"enableProfiling,omitempty"`
	EnableTracing   bool `json:"enableTracing,omitempty"`

	// Metadata
	RequestID   string            `json:"requestId,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// FunctionExecutionResponse represents a function execution response
type FunctionExecutionResponse struct {
	// Results
	Resources    []*porch.KRMResource    `json:"resources"`
	ResourceList *functions.ResourceList `json:"resourceList,omitempty"`
	Results      []*porch.FunctionResult `json:"results,omitempty"`

	// Execution metadata
	ExecutionID   string        `json:"executionId"`
	FunctionName  string        `json:"functionName"`
	ExecutionMode string        `json:"executionMode"`
	Duration      time.Duration `json:"duration"`
	StartTime     time.Time     `json:"startTime"`
	EndTime       time.Time     `json:"endTime"`

	// Performance metrics
	CPUUsage    float64 `json:"cpuUsage,omitempty"`
	MemoryUsage int64   `json:"memoryUsage,omitempty"`
	CacheHit    bool    `json:"cacheHit,omitempty"`

	// Error information
	Error        error  `json:"error,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorDetails string `json:"errorDetails,omitempty"`

	// Audit information
	SecurityEvents []*SecurityEvent `json:"securityEvents,omitempty"`
	AuditLogs      []string         `json:"auditLogs,omitempty"`
}

// FunctionRegistration represents a function registration
type FunctionRegistration struct {
	// Function metadata
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Type        functions.FunctionType `json:"type"`

	// Function source
	NativeFunction  functions.KRMFunction    `json:"-"`                         // Native Go function
	ContainerImage  string                   `json:"containerImage,omitempty"`  // Container image
	ContainerConfig *ContainerFunctionConfig `json:"containerConfig,omitempty"` // Container configuration

	// Function metadata
	Metadata *functions.FunctionMetadata  `json:"metadata,omitempty"`
	Schema   *functions.FunctionSchema    `json:"schema,omitempty"`
	Examples []*functions.FunctionExample `json:"examples,omitempty"`

	// Registration metadata
	RegisteredAt time.Time `json:"registeredAt"`
	RegisteredBy string    `json:"registeredBy,omitempty"`
	Source       string    `json:"source,omitempty"`
	Verified     bool      `json:"verified"`

	// Statistics
	ExecutionCount int64         `json:"executionCount"`
	LastExecuted   *time.Time    `json:"lastExecuted,omitempty"`
	AverageLatency time.Duration `json:"averageLatency"`
	ErrorRate      float64       `json:"errorRate"`
}

// ContainerFunctionConfig represents container function configuration
type ContainerFunctionConfig struct {
	Image           string            `json:"image"`
	Tag             string            `json:"tag,omitempty"`
	Command         []string          `json:"command,omitempty"`
	Args            []string          `json:"args,omitempty"`
	Environment     map[string]string `json:"environment,omitempty"`
	WorkingDir      string            `json:"workingDir,omitempty"`
	CPULimit        string            `json:"cpuLimit,omitempty"`
	MemoryLimit     string            `json:"memoryLimit,omitempty"`
	Timeout         time.Duration     `json:"timeout,omitempty"`
	Mounts          []*VolumeMount    `json:"mounts,omitempty"`
	SecurityContext *SecurityConfig   `json:"securityContext,omitempty"`
}

// SecurityConfig represents security configuration for container functions
type SecurityConfig struct {
	RunAsUser       *int64                `json:"runAsUser,omitempty"`
	RunAsGroup      *int64                `json:"runAsGroup,omitempty"`
	ReadOnlyRootFS  bool                  `json:"readOnlyRootFS,omitempty"`
	Privileged      bool                  `json:"privileged,omitempty"`
	Capabilities    *SecurityCapabilities `json:"capabilities,omitempty"`
	AppArmorProfile string                `json:"appArmorProfile,omitempty"`
	SeccompProfile  string                `json:"seccompProfile,omitempty"`
}

// SecurityCapabilities represents security capabilities
type SecurityCapabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

// FunctionCache provides caching for function execution results
type FunctionCache struct {
	cache    map[string]*CacheEntry
	size     int64
	maxSize  int64
	ttl      time.Duration
	mu       sync.RWMutex
	metrics  *CacheMetrics
	cacheDir string
	items    map[string]*CacheItem
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	Key         string                     `json:"key"`
	Response    *FunctionExecutionResponse `json:"response"`
	Size        int64                      `json:"size"`
	CreatedAt   time.Time                  `json:"createdAt"`
	ExpiresAt   time.Time                  `json:"expiresAt"`
	AccessCount int64                      `json:"accessCount"`
	LastAccess  time.Time                  `json:"lastAccess"`
}

// CacheMetrics provides cache performance metrics
type CacheMetrics struct {
	Hits      prometheus.Counter
	Misses    prometheus.Counter
	Evictions prometheus.Counter
	Size      prometheus.Gauge
	Entries   prometheus.Gauge
	ItemCount prometheus.Gauge
}

// FunctionManagerMetrics provides comprehensive metrics
type FunctionManagerMetrics struct {
	FunctionExecutions    *prometheus.CounterVec
	ExecutionDuration     *prometheus.HistogramVec
	FunctionRegistrations prometheus.Gauge
	ExecutionErrors       *prometheus.CounterVec
	CachePerformance      *prometheus.HistogramVec
	SecurityViolations    *prometheus.CounterVec
	ResourceUtilization   *prometheus.GaugeVec
}

// Default configuration
var DefaultFunctionManagerConfig = &FunctionManagerConfig{
	ExecutionMode:            "hybrid",
	DefaultTimeout:           5 * time.Minute,
	MaxConcurrentExecs:       50,
	EnableNativeFunctions:    true,
	EnableContainerFunctions: true,
	DefaultRegistry:          "gcr.io/kpt-fn",
	EnableFunctionCache:      true,
	CacheSize:                1024 * 1024 * 1024, // 1GB
	CacheTTL:                 1 * time.Hour,
	EnableSandboxing:         true,
	EnableMetrics:            true,
	EnableTracing:            true,
	DetailedLogging:          false,
}

// NewFunctionManager creates a new function manager
func NewFunctionManager(config *FunctionManagerConfig, runtime *Runtime, containerRuntime *ContainerRuntime, registry *Registry) (*FunctionManager, error) {
	if config == nil {
		config = DefaultFunctionManagerConfig
	}

	// Validate configuration
	if err := validateFunctionManagerConfig(config); err != nil {
		return nil, fmt.Errorf("invalid function manager configuration: %w", err)
	}

	// Initialize metrics
	metrics := &FunctionManagerMetrics{
		FunctionExecutions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_function_manager_executions_total",
				Help: "Total number of function executions",
			},
			[]string{"function", "mode", "status"},
		),
		ExecutionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "krm_function_manager_execution_duration_seconds",
				Help:    "Duration of function executions",
				Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
			},
			[]string{"function", "mode"},
		),
		FunctionRegistrations: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "krm_function_manager_registrations_total",
				Help: "Total number of registered functions",
			},
		),
		ExecutionErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_function_manager_errors_total",
				Help: "Total number of execution errors",
			},
			[]string{"function", "error_type"},
		),
		CachePerformance: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "krm_function_manager_cache_operation_duration_seconds",
				Help:    "Duration of cache operations",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			},
			[]string{"operation", "result"},
		),
		SecurityViolations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_function_manager_security_violations_total",
				Help: "Total number of security violations",
			},
			[]string{"violation_type", "severity"},
		),
		ResourceUtilization: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "krm_function_manager_resource_utilization",
				Help: "Resource utilization during function execution",
			},
			[]string{"resource_type", "function"},
		),
	}

	// Initialize cache
	var functionCache *FunctionCache
	if config.EnableFunctionCache {
		cacheMetrics := &CacheMetrics{
			Hits: promauto.NewCounter(prometheus.CounterOpts{
				Name: "krm_function_cache_hits_total",
				Help: "Total number of cache hits",
			}),
			Misses: promauto.NewCounter(prometheus.CounterOpts{
				Name: "krm_function_cache_misses_total",
				Help: "Total number of cache misses",
			}),
			Evictions: promauto.NewCounter(prometheus.CounterOpts{
				Name: "krm_function_cache_evictions_total",
				Help: "Total number of cache evictions",
			}),
			Size: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "krm_function_cache_size_bytes",
				Help: "Current cache size in bytes",
			}),
			Entries: promauto.NewGauge(prometheus.GaugeOpts{
				Name: "krm_function_cache_entries_total",
				Help: "Current number of cache entries",
			}),
		}

		functionCache = &FunctionCache{
			cache:   make(map[string]*CacheEntry),
			maxSize: config.CacheSize,
			ttl:     config.CacheTTL,
			metrics: cacheMetrics,
		}
	}

	manager := &FunctionManager{
		config:           config,
		runtime:          runtime,
		containerRuntime: containerRuntime,
		registry:         registry,
		functionCache:    functionCache,
		metrics:          metrics,
		tracer:           otel.Tracer("krm-function-manager"),
		nativeFunctions:  make(map[string]functions.KRMFunction),
	}

	// Register built-in native functions
	if err := manager.registerBuiltinFunctions(); err != nil {
		return nil, fmt.Errorf("failed to register builtin functions: %w", err)
	}

	return manager, nil
}

// ExecuteFunction executes a KRM function
func (fm *FunctionManager) ExecuteFunction(ctx context.Context, req *FunctionExecutionRequest) (*FunctionExecutionResponse, error) {
	ctx, span := fm.tracer.Start(ctx, "function-execution")
	defer span.End()

	// Generate execution ID
	req.RequestID = generateExecutionID()

	span.SetAttributes(
		attribute.String("function.name", req.FunctionName),
		attribute.String("execution.id", req.RequestID),
		attribute.String("execution.mode", req.ExecutionMode),
		attribute.Int("resources.count", len(req.Resources)),
	)

	logger := log.FromContext(ctx).WithName("function-manager").WithValues(
		"function", req.FunctionName,
		"executionId", req.RequestID,
	)

	logger.Info("Starting function execution")
	startTime := time.Now()

	// Determine execution mode
	executionMode := req.ExecutionMode
	if executionMode == "" {
		executionMode = fm.determineExecutionMode(req.FunctionName)
	}

	// Check cache if enabled
	if fm.config.EnableFunctionCache && req.EnableCaching {
		if cached := fm.checkCache(ctx, req); cached != nil {
			logger.Info("Function execution served from cache")
			span.SetAttributes(attribute.Bool("cache.hit", true))
			fm.metrics.FunctionExecutions.WithLabelValues(req.FunctionName, executionMode, "cached").Inc()
			return cached, nil
		}
	}

	// Execute function
	var response *FunctionExecutionResponse
	var err error

	switch executionMode {
	case "native":
		response, err = fm.executeNativeFunction(ctx, req)
	case "container":
		response, err = fm.executeContainerFunction(ctx, req)
	case "hybrid":
		response, err = fm.executeHybridFunction(ctx, req)
	default:
		err = fmt.Errorf("unsupported execution mode: %s", executionMode)
	}

	duration := time.Since(startTime)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "function execution failed")
		fm.metrics.ExecutionErrors.WithLabelValues(req.FunctionName, "execution_error").Inc()
		fm.metrics.FunctionExecutions.WithLabelValues(req.FunctionName, executionMode, "failed").Inc()
		return nil, err
	}

	// Complete response
	response.ExecutionID = req.RequestID
	response.FunctionName = req.FunctionName
	response.ExecutionMode = executionMode
	response.Duration = duration
	response.StartTime = startTime
	response.EndTime = time.Now()

	// Cache result if enabled
	if fm.config.EnableFunctionCache && req.EnableCaching && err == nil {
		fm.cacheResult(ctx, req, response)
	}

	// Record metrics
	fm.metrics.ExecutionDuration.WithLabelValues(req.FunctionName, executionMode).Observe(duration.Seconds())
	fm.metrics.FunctionExecutions.WithLabelValues(req.FunctionName, executionMode, "success").Inc()

	logger.Info("Function execution completed",
		"duration", duration,
		"mode", executionMode)

	span.SetStatus(codes.Ok, "function execution completed")
	return response, nil
}

// RegisterNativeFunction registers a native Go function
func (fm *FunctionManager) RegisterNativeFunction(name string, function functions.KRMFunction) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, exists := fm.nativeFunctions[name]; exists {
		return fmt.Errorf("function %s is already registered", name)
	}

	fm.nativeFunctions[name] = function
	fm.metrics.FunctionRegistrations.Inc()

	return nil
}

// RegisterContainerFunction registers a container-based function
func (fm *FunctionManager) RegisterContainerFunction(name string, config *ContainerFunctionConfig) error {
	// This would register a container function in the registry
	// Implementation would store the container configuration and make it available for execution
	registration := &FunctionRegistration{
		Name:            name,
		Version:         "latest",
		Type:            functions.FunctionTypeMutator,
		ContainerImage:  config.Image,
		ContainerConfig: config,
		RegisteredAt:    time.Now(),
		Verified:        false,
	}

	// Store in registry (implementation would persist this)
	_ = registration

	fm.metrics.FunctionRegistrations.Inc()
	return nil
}

// ListFunctions returns all registered functions
func (fm *FunctionManager) ListFunctions(ctx context.Context) ([]*FunctionRegistration, error) {
	var functions []*FunctionRegistration

	// Add native functions
	fm.mu.RLock()
	for name, function := range fm.nativeFunctions {
		metadata := function.GetMetadata()
		registration := &FunctionRegistration{
			Name:         name,
			Version:      metadata.Version,
			Description:  metadata.Description,
			Type:         metadata.Type,
			Metadata:     metadata,
			RegisteredAt: time.Now(),
			Verified:     true,
			Source:       "native",
		}
		functions = append(functions, registration)
	}
	fm.mu.RUnlock()

	// Add container functions from registry
	if containerFunctions, err := fm.getContainerFunctions(ctx); err == nil {
		functions = append(functions, containerFunctions...)
	}

	return functions, nil
}

// GetFunction returns information about a specific function
func (fm *FunctionManager) GetFunction(ctx context.Context, name string) (*FunctionRegistration, error) {
	// Check native functions
	fm.mu.RLock()
	if function, exists := fm.nativeFunctions[name]; exists {
		fm.mu.RUnlock()
		metadata := function.GetMetadata()
		return &FunctionRegistration{
			Name:         name,
			Version:      metadata.Version,
			Description:  metadata.Description,
			Type:         metadata.Type,
			Metadata:     metadata,
			Schema:       function.GetSchema(),
			RegisteredAt: time.Now(),
			Verified:     true,
			Source:       "native",
		}, nil
	}
	fm.mu.RUnlock()

	// Check container functions
	return fm.getContainerFunction(ctx, name)
}

// Private methods

func (fm *FunctionManager) registerBuiltinFunctions() error {
	// Register O-RAN validation functions
	oranValidator := functions.NewORANComplianceValidator()
	if err := fm.RegisterNativeFunction("oran-compliance-validator", oranValidator); err != nil {
		return err
	}

	a1Validator := functions.NewA1PolicyValidator()
	if err := fm.RegisterNativeFunction("a1-policy-validator", a1Validator); err != nil {
		return err
	}

	fiveGValidator := functions.NewFiveGCoreValidator()
	if err := fm.RegisterNativeFunction("5g-core-validator", fiveGValidator); err != nil {
		return err
	}

	// Register optimization functions
	fiveGOptimizer := functions.NewFiveGCoreOptimizer()
	if err := fm.RegisterNativeFunction("5g-core-optimizer", fiveGOptimizer); err != nil {
		return err
	}

	sliceOptimizer := functions.NewNetworkSliceOptimizer()
	if err := fm.RegisterNativeFunction("network-slice-optimizer", sliceOptimizer); err != nil {
		return err
	}

	vendorNormalizer := functions.NewMultiVendorNormalizer()
	if err := fm.RegisterNativeFunction("multi-vendor-normalizer", vendorNormalizer); err != nil {
		return err
	}

	return nil
}

func (fm *FunctionManager) determineExecutionMode(functionName string) string {
	// Check if it's a native function
	fm.mu.RLock()
	_, isNative := fm.nativeFunctions[functionName]
	fm.mu.RUnlock()

	if isNative {
		return "native"
	}

	// Check if it's a container function
	if fm.isContainerFunction(functionName) {
		return "container"
	}

	return fm.config.ExecutionMode
}

func (fm *FunctionManager) isContainerFunction(functionName string) bool {
	// This would check if the function is registered as a container function
	// For now, assume functions with certain patterns are container functions
	containerPatterns := []string{
		"gcr.io/",
		"docker.io/",
		"quay.io/",
		"registry.k8s.io/",
	}

	for _, pattern := range containerPatterns {
		if strings.Contains(functionName, pattern) {
			return true
		}
	}

	return false
}

func (fm *FunctionManager) executeNativeFunction(ctx context.Context, req *FunctionExecutionRequest) (*FunctionExecutionResponse, error) {
	fm.mu.RLock()
	function, exists := fm.nativeFunctions[req.FunctionName]
	fm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("native function %s not found", req.FunctionName)
	}

	// Convert request to ResourceList
	resourceList := &functions.ResourceList{
		Items:          convertToKRMResources(req.Resources),
		FunctionConfig: req.FunctionConfig,
		Context:        req.Context,
	}

	// Execute function
	result, err := function.Execute(ctx, resourceList)
	if err != nil {
		return nil, err
	}

	// Convert result to response
	response := &FunctionExecutionResponse{
		Resources:    convertFromKRMResources(result.Items),
		ResourceList: result,
		Results:      result.Results,
	}

	return response, nil
}

func (fm *FunctionManager) executeContainerFunction(ctx context.Context, req *FunctionExecutionRequest) (*FunctionExecutionResponse, error) {
	if fm.containerRuntime == nil {
		return nil, fmt.Errorf("container runtime not available")
	}

	// Get container function configuration
	config, err := fm.getContainerFunctionConfig(req.FunctionName)
	if err != nil {
		return nil, err
	}

	// Prepare input files
	inputFiles := make(map[string][]byte)

	// Create resource list input
	resourceList := &functions.ResourceList{
		Items:          convertToKRMResources(req.Resources),
		FunctionConfig: req.FunctionConfig,
		Context:        req.Context,
	}

	inputData, err := json.Marshal(resourceList)
	if err != nil {
		return nil, err
	}
	inputFiles["input.json"] = inputData

	// Create container request
	containerReq := &ContainerRequest{
		Image:        config.Image,
		Tag:          config.Tag,
		Command:      config.Command,
		Args:         config.Args,
		Environment:  config.Environment,
		WorkingDir:   config.WorkingDir,
		CPULimit:     config.CPULimit,
		MemoryLimit:  config.MemoryLimit,
		Timeout:      config.Timeout,
		InputFiles:   inputFiles,
		Context:      ctx,
		RequestID:    req.RequestID,
		FunctionName: req.FunctionName,
	}

	// Configure security context
	if config.SecurityContext != nil {
		if config.SecurityContext.RunAsUser != nil {
			containerReq.User = fmt.Sprintf("%d", *config.SecurityContext.RunAsUser)
		}
		if config.SecurityContext.RunAsGroup != nil {
			containerReq.Group = fmt.Sprintf("%d", *config.SecurityContext.RunAsGroup)
		}
		containerReq.ReadOnlyRootFS = config.SecurityContext.ReadOnlyRootFS
		containerReq.Privileged = config.SecurityContext.Privileged

		if config.SecurityContext.Capabilities != nil {
			containerReq.Capabilities = config.SecurityContext.Capabilities.Add
		}
	}

	// Execute container
	containerResp, err := fm.containerRuntime.ExecuteContainer(ctx, containerReq)
	if err != nil {
		return nil, err
	}

	// Parse output
	if containerResp.ExitCode != 0 {
		return nil, fmt.Errorf("container function failed with exit code %d: %s",
			containerResp.ExitCode, string(containerResp.Stderr))
	}

	// Parse output files
	var outputResourceList functions.ResourceList
	if outputData, exists := containerResp.OutputFiles["output.json"]; exists {
		if err := json.Unmarshal(outputData, &outputResourceList); err != nil {
			return nil, fmt.Errorf("failed to parse function output: %w", err)
		}
	} else {
		// Fallback to stdout
		if err := json.Unmarshal(containerResp.Stdout, &outputResourceList); err != nil {
			return nil, fmt.Errorf("failed to parse function stdout: %w", err)
		}
	}

	response := &FunctionExecutionResponse{
		Resources:    convertFromKRMResources(outputResourceList.Items),
		ResourceList: &outputResourceList,
		Results:      outputResourceList.Results,
		CPUUsage:     float64(containerResp.CPUUsage.MilliValue()),
		MemoryUsage:  containerResp.MemoryUsage.Value(),
	}

	return response, nil
}

func (fm *FunctionManager) executeHybridFunction(ctx context.Context, req *FunctionExecutionRequest) (*FunctionExecutionResponse, error) {
	// Try native first, fall back to container
	if fm.isNativeFunction(req.FunctionName) {
		return fm.executeNativeFunction(ctx, req)
	}
	return fm.executeContainerFunction(ctx, req)
}

func (fm *FunctionManager) isNativeFunction(functionName string) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	_, exists := fm.nativeFunctions[functionName]
	return exists
}

func (fm *FunctionManager) getContainerFunctionConfig(functionName string) (*ContainerFunctionConfig, error) {
	// This would retrieve container function configuration from registry
	// For now, provide a default configuration
	return &ContainerFunctionConfig{
		Image:       functionName,
		CPULimit:    "500m",
		MemoryLimit: "512Mi",
		Timeout:     5 * time.Minute,
		SecurityContext: &SecurityConfig{
			ReadOnlyRootFS: true,
			Capabilities: &SecurityCapabilities{
				Drop: []string{"ALL"},
			},
		},
	}, nil
}

func (fm *FunctionManager) getContainerFunctions(ctx context.Context) ([]*FunctionRegistration, error) {
	// This would retrieve container functions from registry
	return []*FunctionRegistration{}, nil
}

func (fm *FunctionManager) getContainerFunction(ctx context.Context, name string) (*FunctionRegistration, error) {
	// This would retrieve a specific container function from registry
	return nil, fmt.Errorf("container function %s not found", name)
}

// Cache methods

func (fm *FunctionManager) checkCache(ctx context.Context, req *FunctionExecutionRequest) *FunctionExecutionResponse {
	if fm.functionCache == nil {
		return nil
	}

	key := fm.generateCacheKey(req)

	fm.functionCache.mu.RLock()
	defer fm.functionCache.mu.RUnlock()

	entry, exists := fm.functionCache.cache[key]
	if !exists {
		fm.functionCache.metrics.Misses.Inc()
		return nil
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		fm.functionCache.metrics.Misses.Inc()
		return nil
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccess = time.Now()

	fm.functionCache.metrics.Hits.Inc()
	return entry.Response
}

func (fm *FunctionManager) cacheResult(ctx context.Context, req *FunctionExecutionRequest, response *FunctionExecutionResponse) {
	if fm.functionCache == nil {
		return
	}

	key := fm.generateCacheKey(req)

	// Calculate entry size
	data, _ := json.Marshal(response)
	size := int64(len(data))

	entry := &CacheEntry{
		Key:         key,
		Response:    response,
		Size:        size,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(fm.functionCache.ttl),
		AccessCount: 1,
		LastAccess:  time.Now(),
	}

	fm.functionCache.mu.Lock()
	defer fm.functionCache.mu.Unlock()

	// Check if we need to evict entries
	if fm.functionCache.size+size > fm.functionCache.maxSize {
		fm.evictCacheEntries(size)
	}

	fm.functionCache.cache[key] = entry
	fm.functionCache.size += size

	fm.functionCache.metrics.Size.Set(float64(fm.functionCache.size))
	fm.functionCache.metrics.Entries.Set(float64(len(fm.functionCache.cache)))
}

func (fm *FunctionManager) generateCacheKey(req *FunctionExecutionRequest) string {
	// Generate a hash of the request for caching
	data, _ := json.Marshal(struct {
		Function  string                 `json:"function"`
		Config    map[string]interface{} `json:"config"`
		Resources string                 `json:"resources"`
	}{
		Function:  req.FunctionName,
		Config:    req.FunctionConfig,
		Resources: fmt.Sprintf("%v", req.Resources),
	})

	return fmt.Sprintf("%x", data)
}

func (fm *FunctionManager) evictCacheEntries(neededSize int64) {
	// Simple LRU eviction
	for fm.functionCache.size+neededSize > fm.functionCache.maxSize && len(fm.functionCache.cache) > 0 {
		var oldestKey string
		var oldestTime time.Time

		for key, entry := range fm.functionCache.cache {
			if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.LastAccess
			}
		}

		if oldestKey != "" {
			entry := fm.functionCache.cache[oldestKey]
			delete(fm.functionCache.cache, oldestKey)
			fm.functionCache.size -= entry.Size
			fm.functionCache.metrics.Evictions.Inc()
		}
	}
}

// Health returns function manager health status
func (fm *FunctionManager) Health() *FunctionManagerHealth {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	health := &FunctionManagerHealth{
		Status:              "healthy",
		RegisteredFunctions: len(fm.nativeFunctions),
		LastHealthCheck:     time.Now(),
	}

	if fm.functionCache != nil {
		fm.functionCache.mu.RLock()
		health.CacheSize = fm.functionCache.size
		health.CacheEntries = len(fm.functionCache.cache)
		fm.functionCache.mu.RUnlock()
	}

	return health
}

// FunctionManagerHealth represents health status
type FunctionManagerHealth struct {
	Status              string    `json:"status"`
	RegisteredFunctions int       `json:"registeredFunctions"`
	CacheSize           int64     `json:"cacheSize"`
	CacheEntries        int       `json:"cacheEntries"`
	LastHealthCheck     time.Time `json:"lastHealthCheck"`
}

// Shutdown gracefully shuts down the function manager
func (fm *FunctionManager) Shutdown(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("function-manager")
	logger.Info("Shutting down function manager")

	// Clear function cache
	if fm.functionCache != nil {
		fm.functionCache.mu.Lock()
		fm.functionCache.cache = make(map[string]*CacheEntry)
		fm.functionCache.size = 0
		fm.functionCache.mu.Unlock()
	}

	logger.Info("Function manager shutdown complete")
	return nil
}

// Helper functions

func validateFunctionManagerConfig(config *FunctionManagerConfig) error {
	if config.DefaultTimeout <= 0 {
		return fmt.Errorf("defaultTimeout must be positive")
	}
	if config.MaxConcurrentExecs <= 0 {
		return fmt.Errorf("maxConcurrentExecs must be positive")
	}
	return nil
}


func convertToKRMResources(resources []*porch.KRMResource) []porch.KRMResource {
	result := make([]porch.KRMResource, len(resources))
	for i, r := range resources {
		result[i] = *r
	}
	return result
}

func convertFromKRMResources(resources []porch.KRMResource) []*porch.KRMResource {
	result := make([]*porch.KRMResource, len(resources))
	for i, r := range resources {
		result[i] = &r
	}
	return result
}
