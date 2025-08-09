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

package porch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// functionRunner implements the FunctionRunner interface
type functionRunner struct {
	// Parent client for API operations
	client *Client

	// Logger for function operations
	logger logr.Logger

	// Metrics for function operations
	metrics *FunctionRunnerMetrics

	// Function registry for available functions
	registry FunctionRegistry

	// Execution engine for running functions
	executionEngine ExecutionEngine

	// Resource quota manager
	quotaManager ResourceQuotaManager

	// Security validator for function execution
	securityValidator FunctionSecurityValidator

	// Pipeline orchestrator for chaining functions
	pipelineOrchestrator PipelineOrchestrator

	// O-RAN compliance validator
	oranValidator ORANComplianceValidator

	// Performance profiler
	profiler PerformanceProfiler

	// Cache for function results
	cache FunctionCache

	// Concurrent execution limiter
	semaphore chan struct{}

	// Active executions tracking
	activeExecutions map[string]*executionContext
	executionMutex   sync.RWMutex
}

// executionContext tracks active function executions
type executionContext struct {
	id           string
	functionName string
	startTime    time.Time
	cancel       context.CancelFunc
	resources    *ResourceUsage
	status       ExecutionStatus
	mutex        sync.RWMutex
}

// ExecutionStatus represents the current status of function execution
type ExecutionStatus string

const (
	ExecutionStatusQueued    ExecutionStatus = "Queued"
	ExecutionStatusRunning   ExecutionStatus = "Running"
	ExecutionStatusCompleted ExecutionStatus = "Completed"
	ExecutionStatusFailed    ExecutionStatus = "Failed"
	ExecutionStatusCancelled ExecutionStatus = "Cancelled"
	ExecutionStatusTimeout   ExecutionStatus = "Timeout"
)

// ResourceUsage tracks resource consumption during execution
type ResourceUsage struct {
	CPU       float64
	Memory    int64
	Storage   int64
	Network   int64
	Duration  time.Duration
	ExitCode  int
	StartTime time.Time
	EndTime   time.Time
}

// FunctionRunnerMetrics defines Prometheus metrics for function operations
type FunctionRunnerMetrics struct {
	functionExecutions *prometheus.CounterVec
	executionDuration  *prometheus.HistogramVec
	executionErrors    *prometheus.CounterVec
	activeExecutions   *prometheus.GaugeVec
	resourceUsage      *prometheus.GaugeVec
	cacheHitRate       *prometheus.CounterVec
	pipelineExecutions *prometheus.CounterVec
	validationResults  *prometheus.CounterVec
}

// FunctionRegistry manages available KRM functions
type FunctionRegistry interface {
	RegisterFunction(ctx context.Context, info *FunctionInfo) error
	UnregisterFunction(ctx context.Context, name string) error
	GetFunction(ctx context.Context, name string) (*FunctionInfo, error)
	ListFunctions(ctx context.Context) ([]*FunctionInfo, error)
	SearchFunctions(ctx context.Context, query string) ([]*FunctionInfo, error)
	ValidateFunction(ctx context.Context, name string) (*FunctionValidation, error)
	GetFunctionSchema(ctx context.Context, name string) (*FunctionSchema, error)
}

// ExecutionEngine handles the actual execution of KRM functions
type ExecutionEngine interface {
	ExecuteFunction(ctx context.Context, req *FunctionExecutionRequest) (*FunctionExecutionResult, error)
	GetSupportedRuntimes() []string
	IsRuntimeAvailable(runtime string) bool
	GetResourceLimits() *FunctionResourceLimits
	SetResourceLimits(limits *FunctionResourceLimits)
}

// FunctionExecutionRequest extends FunctionRequest with execution-specific details
type FunctionExecutionRequest struct {
	*FunctionRequest
	ExecutionID    string
	Runtime        string
	ResourceLimits *FunctionResourceLimits
	Timeout        time.Duration
	Environment    map[string]string
	WorkingDir     string
	NetworkAccess  bool
	Privileged     bool
}

// FunctionExecutionResult extends FunctionResponse with execution details
type FunctionExecutionResult struct {
	*FunctionResponse
	ExecutionID   string
	ResourceUsage *ResourceUsage
	ExitCode      int
	StartTime     time.Time
	EndTime       time.Time
	Runtime       string
}

// ResourceQuotaManager manages resource quotas for function execution
type ResourceQuotaManager interface {
	CheckQuota(ctx context.Context, req *FunctionExecutionRequest) error
	ReserveResources(ctx context.Context, executionID string, resources *ResourceRequirement) error
	ReleaseResources(ctx context.Context, executionID string) error
	GetQuotaUsage(ctx context.Context) (*QuotaUsage, error)
}

// ResourceRequirement defines resource requirements
type ResourceRequirement struct {
	CPU     float64
	Memory  int64
	Storage int64
	Network int64
}

// QuotaUsage represents current quota usage
type QuotaUsage struct {
	Used      *ResourceRequirement
	Available *ResourceRequirement
	Limit     *ResourceRequirement
}

// FunctionSecurityValidator validates function security requirements
type FunctionSecurityValidator interface {
	ValidateFunction(ctx context.Context, req *FunctionExecutionRequest) error
	ScanImage(ctx context.Context, image string) (*SecurityScanResult, error)
	ValidatePermissions(ctx context.Context, req *FunctionExecutionRequest) error
	CheckCompliance(ctx context.Context, req *FunctionExecutionRequest) error
}

// SecurityScanResult represents security scan results
type SecurityScanResult struct {
	Vulnerabilities  []Vulnerability
	PolicyViolations []PolicyViolation
	RiskScore        int
	Approved         bool
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	Policy      string
	Severity    string
	Description string
	Remediation string
}

// PipelineOrchestrator manages execution of function pipelines
type PipelineOrchestrator interface {
	ExecutePipeline(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error)
	ValidatePipeline(ctx context.Context, pipeline *Pipeline) error
	OptimizePipeline(ctx context.Context, pipeline *Pipeline) (*Pipeline, error)
	GetPipelineMetrics(ctx context.Context, pipelineID string) (*PipelineMetrics, error)
}

// PipelineMetrics represents pipeline execution metrics
type PipelineMetrics struct {
	TotalDuration   time.Duration
	StageExecutions map[string]time.Duration
	ResourceUsage   *ResourceUsage
	SuccessRate     float64
	ErrorRate       float64
}

// ORANComplianceValidator validates O-RAN compliance of function results
type ORANComplianceValidator interface {
	ValidateCompliance(ctx context.Context, resources []KRMResource) (*ComplianceResult, error)
	GetComplianceRules() []ComplianceRule
	ValidateInterface(ctx context.Context, interfaceType string, config map[string]interface{}) error
	ValidateNetworkFunction(ctx context.Context, nf *NetworkFunctionSpec) error
}

// ComplianceResult represents O-RAN compliance validation results
type ComplianceResult struct {
	Compliant  bool
	Violations []ComplianceViolation
	Warnings   []ComplianceWarning
	Score      int
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	Rule        string
	Severity    string
	Description string
	Resource    string
	Field       string
	Remediation string
}

// ComplianceWarning represents a compliance warning
type ComplianceWarning struct {
	Rule        string
	Description string
	Resource    string
	Field       string
	Suggestion  string
}

// NetworkFunctionSpec represents a network function specification
type NetworkFunctionSpec struct {
	Type       string
	Interfaces []ORANInterface
	Resources  map[string]interface{}
	Metadata   map[string]string
}

// PerformanceProfiler profiles function execution performance
type PerformanceProfiler interface {
	StartProfiling(ctx context.Context, executionID string) error
	StopProfiling(ctx context.Context, executionID string) (*ProfileResult, error)
	GetProfile(ctx context.Context, executionID string, profileType string) ([]byte, error)
	AnalyzePerformance(ctx context.Context, executionID string) (*PerformanceAnalysis, error)
}

// ProfileResult represents profiling results
type ProfileResult struct {
	ExecutionID string
	ProfileType string
	Data        []byte
	Analysis    *PerformanceAnalysis
	GeneratedAt time.Time
}

// PerformanceAnalysis represents performance analysis results
type PerformanceAnalysis struct {
	Bottlenecks        []Bottleneck
	Recommendations    []string
	Score              int
	ResourceEfficiency float64
}

// Bottleneck represents a performance bottleneck
type Bottleneck struct {
	Type       string
	Location   string
	Impact     string
	Suggestion string
}

// FunctionCache provides caching for function results
type FunctionCache interface {
	Get(ctx context.Context, key string) (*FunctionResponse, bool)
	Set(ctx context.Context, key string, response *FunctionResponse, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	GetStats() *CacheStats
}

// CacheStats represents cache statistics
type CacheStats struct {
	HitCount  int64
	MissCount int64
	Size      int64
	HitRate   float64
}

// NewFunctionRunner creates a new function runner
func NewFunctionRunner(client *Client) FunctionRunner {
	maxConcurrent := 5
	if client.config.Functions != nil && client.config.Functions.Execution != nil {
		maxConcurrent = client.config.Functions.Execution.MaxConcurrent
	}

	return &functionRunner{
		client:           client,
		logger:           log.Log.WithName("function-runner"),
		metrics:          initFunctionRunnerMetrics(),
		activeExecutions: make(map[string]*executionContext),
		semaphore:        make(chan struct{}, maxConcurrent),
	}
}

// ExecuteFunction executes a single KRM function
func (fr *functionRunner) ExecuteFunction(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	start := time.Now()
	executionID := fmt.Sprintf("exec-%d", time.Now().UnixNano())

	fr.logger.Info("Executing function", "function", req.FunctionConfig.Image, "executionID", executionID)

	defer func() {
		if fr.metrics != nil {
			fr.metrics.executionDuration.WithLabelValues(req.FunctionConfig.Image, "single").Observe(time.Since(start).Seconds())
		}
	}()

	// Check cache first
	if fr.cache != nil {
		cacheKey := fr.generateCacheKey(req)
		if cached, found := fr.cache.Get(ctx, cacheKey); found {
			fr.logger.V(1).Info("Function result found in cache", "executionID", executionID)
			if fr.metrics != nil {
				fr.metrics.cacheHitRate.WithLabelValues("hit").Inc()
			}
			return cached, nil
		}
		if fr.metrics != nil {
			fr.metrics.cacheHitRate.WithLabelValues("miss").Inc()
		}
	}

	// Create execution request
	execReq := &FunctionExecutionRequest{
		FunctionRequest: req,
		ExecutionID:     executionID,
		Runtime:         fr.getRuntime(req),
		Timeout:         fr.getTimeout(req),
		ResourceLimits:  fr.getResourceLimits(req),
		Environment:     fr.getEnvironment(req),
	}

	// Validate security requirements
	if fr.securityValidator != nil {
		if err := fr.securityValidator.ValidateFunction(ctx, execReq); err != nil {
			fr.recordError(executionID, "security_validation", err)
			return nil, fmt.Errorf("security validation failed: %w", err)
		}
	}

	// Check resource quotas
	if fr.quotaManager != nil {
		if err := fr.quotaManager.CheckQuota(ctx, execReq); err != nil {
			fr.recordError(executionID, "quota_exceeded", err)
			return nil, fmt.Errorf("resource quota exceeded: %w", err)
		}
	}

	// Acquire semaphore for concurrent execution control
	select {
	case fr.semaphore <- struct{}{}:
		defer func() { <-fr.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create execution context
	execCtx, cancel := context.WithTimeout(ctx, execReq.Timeout)
	defer cancel()

	execution := &executionContext{
		id:           executionID,
		functionName: req.FunctionConfig.Image,
		startTime:    time.Now(),
		cancel:       cancel,
		status:       ExecutionStatusQueued,
	}

	fr.executionMutex.Lock()
	fr.activeExecutions[executionID] = execution
	fr.executionMutex.Unlock()

	defer func() {
		fr.executionMutex.Lock()
		delete(fr.activeExecutions, executionID)
		fr.executionMutex.Unlock()
	}()

	// Update metrics
	if fr.metrics != nil {
		fr.metrics.functionExecutions.WithLabelValues(req.FunctionConfig.Image, "started").Inc()
		fr.metrics.activeExecutions.WithLabelValues(req.FunctionConfig.Image).Inc()
		defer fr.metrics.activeExecutions.WithLabelValues(req.FunctionConfig.Image).Dec()
	}

	// Reserve resources
	if fr.quotaManager != nil {
		resourceReq := &ResourceRequirement{
			CPU:     fr.parseCPULimit(execReq.ResourceLimits.CPU),
			Memory:  fr.parseMemoryLimit(execReq.ResourceLimits.Memory),
			Storage: fr.parseStorageLimit(execReq.ResourceLimits.Storage),
		}
		if err := fr.quotaManager.ReserveResources(ctx, executionID, resourceReq); err != nil {
			fr.recordError(executionID, "resource_reservation", err)
			return nil, fmt.Errorf("failed to reserve resources: %w", err)
		}
		defer fr.quotaManager.ReleaseResources(ctx, executionID)
	}

	// Start profiling if enabled
	if fr.profiler != nil {
		fr.profiler.StartProfiling(ctx, executionID)
		defer fr.profiler.StopProfiling(ctx, executionID)
	}

	// Update execution status
	execution.mutex.Lock()
	execution.status = ExecutionStatusRunning
	execution.mutex.Unlock()

	// Execute function
	result, err := fr.executionEngine.ExecuteFunction(execCtx, execReq)
	if err != nil {
		execution.mutex.Lock()
		execution.status = ExecutionStatusFailed
		execution.mutex.Unlock()

		fr.recordError(executionID, "execution_failed", err)
		return nil, fmt.Errorf("function execution failed: %w", err)
	}

	// Update execution status
	execution.mutex.Lock()
	execution.status = ExecutionStatusCompleted
	execution.resources = result.ResourceUsage
	execution.mutex.Unlock()

	// Validate O-RAN compliance if configured
	if fr.oranValidator != nil && len(result.Resources) > 0 {
		if compliance, err := fr.oranValidator.ValidateCompliance(ctx, result.Resources); err != nil {
			fr.logger.Error(err, "O-RAN compliance validation failed", "executionID", executionID)
		} else if !compliance.Compliant {
			fr.logger.Warn("Function result is not O-RAN compliant", "executionID", executionID, "violations", len(compliance.Violations))
			if fr.metrics != nil {
				fr.metrics.validationResults.WithLabelValues("oran_compliance", "failed").Inc()
			}
		}
	}

	// Cache result if applicable
	if fr.cache != nil && fr.shouldCacheResult(req, result.FunctionResponse) {
		cacheKey := fr.generateCacheKey(req)
		cacheTTL := 5 * time.Minute
		if err := fr.cache.Set(ctx, cacheKey, result.FunctionResponse, cacheTTL); err != nil {
			fr.logger.Error(err, "Failed to cache function result", "executionID", executionID)
		}
	}

	// Update metrics
	if fr.metrics != nil {
		fr.metrics.functionExecutions.WithLabelValues(req.FunctionConfig.Image, "completed").Inc()
		if result.ResourceUsage != nil {
			fr.metrics.resourceUsage.WithLabelValues(executionID, "cpu").Set(result.ResourceUsage.CPU)
			fr.metrics.resourceUsage.WithLabelValues(executionID, "memory").Set(float64(result.ResourceUsage.Memory))
		}
	}

	fr.logger.Info("Function execution completed", "executionID", executionID, "duration", time.Since(start))
	return result.FunctionResponse, nil
}

// ExecutePipeline executes a pipeline of KRM functions
func (fr *functionRunner) ExecutePipeline(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error) {
	start := time.Now()
	pipelineID := fmt.Sprintf("pipeline-%d", time.Now().UnixNano())

	fr.logger.Info("Executing function pipeline", "pipelineID", pipelineID, "stages", len(req.Pipeline.Mutators)+len(req.Pipeline.Validators))

	defer func() {
		if fr.metrics != nil {
			fr.metrics.pipelineExecutions.WithLabelValues("completed").Inc()
		}
	}()

	// Validate pipeline
	if fr.pipelineOrchestrator != nil {
		if err := fr.pipelineOrchestrator.ValidatePipeline(ctx, &req.Pipeline); err != nil {
			return nil, fmt.Errorf("pipeline validation failed: %w", err)
		}
	}

	// Use pipeline orchestrator if available
	if fr.pipelineOrchestrator != nil {
		return fr.pipelineOrchestrator.ExecutePipeline(ctx, req)
	}

	// Fallback to sequential execution
	return fr.executeSequentialPipeline(ctx, req)
}

// ValidateFunction validates a function's availability and configuration
func (fr *functionRunner) ValidateFunction(ctx context.Context, functionName string) (*FunctionValidation, error) {
	fr.logger.V(1).Info("Validating function", "function", functionName)

	// Check function registry
	if fr.registry != nil {
		return fr.registry.ValidateFunction(ctx, functionName)
	}

	// Basic validation
	validation := &FunctionValidation{
		Valid: true,
	}

	// Validate image format
	if !strings.Contains(functionName, ":") {
		validation.Valid = false
		validation.Errors = append(validation.Errors, ValidationError{
			Message:  "Function image must include a tag",
			Severity: "error",
			Code:     "INVALID_IMAGE_TAG",
		})
	}

	return validation, nil
}

// ListFunctions lists all available functions
func (fr *functionRunner) ListFunctions(ctx context.Context) ([]*FunctionInfo, error) {
	if fr.registry != nil {
		return fr.registry.ListFunctions(ctx)
	}

	// Return default functions if no registry
	return []*FunctionInfo{
		{
			Name:        "gcr.io/kpt-fn/apply-setters",
			Description: "Apply setters to configuration",
			Types:       []string{"mutator"},
		},
		{
			Name:        "gcr.io/kpt-fn/set-namespace",
			Description: "Set namespace for resources",
			Types:       []string{"mutator"},
		},
		{
			Name:        "gcr.io/kpt-fn/kubeval",
			Description: "Validate Kubernetes resources",
			Types:       []string{"validator"},
		},
	}, nil
}

// GetFunctionSchema returns the schema for a function's configuration
func (fr *functionRunner) GetFunctionSchema(ctx context.Context, functionName string) (*FunctionSchema, error) {
	if fr.registry != nil {
		return fr.registry.GetFunctionSchema(ctx, functionName)
	}

	// Return empty schema if no registry
	return &FunctionSchema{}, nil
}

// RegisterFunction registers a new function in the registry
func (fr *functionRunner) RegisterFunction(ctx context.Context, info *FunctionInfo) error {
	if fr.registry != nil {
		return fr.registry.RegisterFunction(ctx, info)
	}

	return fmt.Errorf("function registry not available")
}

// Private helper methods

// executeSequentialPipeline executes a pipeline sequentially
func (fr *functionRunner) executeSequentialPipeline(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error) {
	resources := make([]KRMResource, len(req.Resources))
	copy(resources, req.Resources)

	var allResults []*FunctionResult

	// Execute mutators first
	for i, mutator := range req.Pipeline.Mutators {
		fr.logger.V(1).Info("Executing mutator", "index", i, "image", mutator.Image)

		funcReq := &FunctionRequest{
			FunctionConfig: mutator,
			Resources:      resources,
			Context:        req.Context,
		}

		response, err := fr.ExecuteFunction(ctx, funcReq)
		if err != nil {
			return &PipelineResponse{
				Resources: resources,
				Results:   allResults,
				Error: &FunctionError{
					Message: fmt.Sprintf("Mutator %d failed: %v", i, err),
					Code:    "MUTATOR_FAILED",
				},
			}, nil
		}

		resources = response.Resources
		allResults = append(allResults, response.Results...)
	}

	// Execute validators
	for i, validator := range req.Pipeline.Validators {
		fr.logger.V(1).Info("Executing validator", "index", i, "image", validator.Image)

		funcReq := &FunctionRequest{
			FunctionConfig: validator,
			Resources:      resources,
			Context:        req.Context,
		}

		response, err := fr.ExecuteFunction(ctx, funcReq)
		if err != nil {
			return &PipelineResponse{
				Resources: resources,
				Results:   allResults,
				Error: &FunctionError{
					Message: fmt.Sprintf("Validator %d failed: %v", i, err),
					Code:    "VALIDATOR_FAILED",
				},
			}, nil
		}

		allResults = append(allResults, response.Results...)

		// Check for validation errors
		for _, result := range response.Results {
			if result.Severity == "error" {
				return &PipelineResponse{
					Resources: resources,
					Results:   allResults,
					Error: &FunctionError{
						Message: result.Message,
						Code:    "VALIDATION_FAILED",
					},
				}, nil
			}
		}
	}

	return &PipelineResponse{
		Resources: resources,
		Results:   allResults,
	}, nil
}

// getRuntime determines the runtime for function execution
func (fr *functionRunner) getRuntime(req *FunctionRequest) string {
	if fr.client.config.Functions != nil && fr.client.config.Functions.Execution != nil {
		return fr.client.config.Functions.Execution.Runtime
	}
	return "docker"
}

// getTimeout determines the timeout for function execution
func (fr *functionRunner) getTimeout(req *FunctionRequest) time.Duration {
	if fr.client.config.Functions != nil && fr.client.config.Functions.Execution != nil {
		return fr.client.config.Functions.Execution.DefaultTimeout
	}
	return 5 * time.Minute
}

// getResourceLimits determines resource limits for function execution
func (fr *functionRunner) getResourceLimits(req *FunctionRequest) *FunctionResourceLimits {
	if fr.client.config.Functions != nil && fr.client.config.Functions.Execution != nil {
		return fr.client.config.Functions.Execution.ResourceLimits
	}
	return &FunctionResourceLimits{
		CPU:     "1000m",
		Memory:  "1Gi",
		Timeout: 5 * time.Minute,
	}
}

// getEnvironment determines environment variables for function execution
func (fr *functionRunner) getEnvironment(req *FunctionRequest) map[string]string {
	env := make(map[string]string)

	if fr.client.config.Functions != nil && fr.client.config.Functions.Execution != nil {
		for k, v := range fr.client.config.Functions.Execution.EnvironmentVars {
			env[k] = v
		}
	}

	// Add context-specific environment variables
	if req.Context != nil && req.Context.Environment != nil {
		for k, v := range req.Context.Environment {
			env[k] = v
		}
	}

	return env
}

// generateCacheKey generates a cache key for function request
func (fr *functionRunner) generateCacheKey(req *FunctionRequest) string {
	// Simple cache key based on function config and resource hash
	configJSON, _ := json.Marshal(req.FunctionConfig)
	resourceJSON, _ := json.Marshal(req.Resources)
	return fmt.Sprintf("%x-%x", configJSON, resourceJSON)
}

// shouldCacheResult determines if a result should be cached
func (fr *functionRunner) shouldCacheResult(req *FunctionRequest, resp *FunctionResponse) bool {
	// Cache successful results with no errors
	if resp.Error != nil {
		return false
	}

	// Don't cache if there are error-level results
	for _, result := range resp.Results {
		if result.Severity == "error" {
			return false
		}
	}

	return true
}

// recordError records function execution errors
func (fr *functionRunner) recordError(executionID, errorType string, err error) {
	fr.logger.Error(err, "Function execution error", "executionID", executionID, "errorType", errorType)

	if fr.metrics != nil {
		fr.metrics.executionErrors.WithLabelValues(errorType).Inc()
	}
}

// parseCPULimit parses CPU limit string to float64
func (fr *functionRunner) parseCPULimit(cpu string) float64 {
	// Simple parsing - in production would use k8s resource parsing
	if strings.HasSuffix(cpu, "m") {
		var milliCPU float64
		fmt.Sscanf(cpu, "%f", &milliCPU)
		return milliCPU / 1000.0
	}

	var cores float64
	fmt.Sscanf(cpu, "%f", &cores)
	return cores
}

// parseMemoryLimit parses memory limit string to int64 bytes
func (fr *functionRunner) parseMemoryLimit(memory string) int64 {
	// Simple parsing - in production would use k8s resource parsing
	memory = strings.ToUpper(memory)
	var value int64
	var unit string
	fmt.Sscanf(memory, "%d%s", &value, &unit)

	switch unit {
	case "KI", "K":
		return value * 1024
	case "MI", "M":
		return value * 1024 * 1024
	case "GI", "G":
		return value * 1024 * 1024 * 1024
	default:
		return value
	}
}

// parseStorageLimit parses storage limit string to int64 bytes
func (fr *functionRunner) parseStorageLimit(storage string) int64 {
	return fr.parseMemoryLimit(storage) // Same parsing logic
}

// initFunctionRunnerMetrics initializes Prometheus metrics
func initFunctionRunnerMetrics() *FunctionRunnerMetrics {
	return &FunctionRunnerMetrics{
		functionExecutions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_function_executions_total",
				Help: "Total number of function executions",
			},
			[]string{"function", "status"},
		),
		executionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "porch_function_execution_duration_seconds",
				Help:    "Duration of function executions",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"function", "type"},
		),
		executionErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_function_execution_errors_total",
				Help: "Total number of function execution errors",
			},
			[]string{"error_type"},
		),
		activeExecutions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "porch_function_active_executions",
				Help: "Number of currently active function executions",
			},
			[]string{"function"},
		),
		resourceUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "porch_function_resource_usage",
				Help: "Resource usage during function execution",
			},
			[]string{"execution_id", "resource_type"},
		),
		cacheHitRate: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_function_cache_operations_total",
				Help: "Total number of function cache operations",
			},
			[]string{"type"},
		),
		pipelineExecutions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_function_pipeline_executions_total",
				Help: "Total number of function pipeline executions",
			},
			[]string{"status"},
		),
		validationResults: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "porch_function_validation_results_total",
				Help: "Total number of function validation results",
			},
			[]string{"validator", "result"},
		),
	}
}

// GetFunctionRunnerMetrics returns function runner metrics
func (fr *functionRunner) GetFunctionRunnerMetrics() *FunctionRunnerMetrics {
	return fr.metrics
}

// SetFunctionRegistry sets the function registry
func (fr *functionRunner) SetFunctionRegistry(registry FunctionRegistry) {
	fr.registry = registry
}

// SetExecutionEngine sets the execution engine
func (fr *functionRunner) SetExecutionEngine(engine ExecutionEngine) {
	fr.executionEngine = engine
}

// SetQuotaManager sets the resource quota manager
func (fr *functionRunner) SetQuotaManager(manager ResourceQuotaManager) {
	fr.quotaManager = manager
}

// SetSecurityValidator sets the security validator
func (fr *functionRunner) SetSecurityValidator(validator FunctionSecurityValidator) {
	fr.securityValidator = validator
}

// SetPipelineOrchestrator sets the pipeline orchestrator
func (fr *functionRunner) SetPipelineOrchestrator(orchestrator PipelineOrchestrator) {
	fr.pipelineOrchestrator = orchestrator
}

// SetORANValidator sets the O-RAN compliance validator
func (fr *functionRunner) SetORANValidator(validator ORANComplianceValidator) {
	fr.oranValidator = validator
}

// SetPerformanceProfiler sets the performance profiler
func (fr *functionRunner) SetPerformanceProfiler(profiler PerformanceProfiler) {
	fr.profiler = profiler
}

// SetFunctionCache sets the function cache
func (fr *functionRunner) SetFunctionCache(cache FunctionCache) {
	fr.cache = cache
}

// GetActiveExecutions returns currently active executions
func (fr *functionRunner) GetActiveExecutions() map[string]*executionContext {
	fr.executionMutex.RLock()
	defer fr.executionMutex.RUnlock()

	result := make(map[string]*executionContext)
	for k, v := range fr.activeExecutions {
		result[k] = v
	}
	return result
}

// CancelExecution cancels an active execution
func (fr *functionRunner) CancelExecution(executionID string) error {
	fr.executionMutex.RLock()
	execution, exists := fr.activeExecutions[executionID]
	fr.executionMutex.RUnlock()

	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	execution.mutex.Lock()
	if execution.cancel != nil {
		execution.cancel()
		execution.status = ExecutionStatusCancelled
	}
	execution.mutex.Unlock()

	return nil
}

// Close gracefully shuts down the function runner
func (fr *functionRunner) Close() error {
	fr.logger.Info("Shutting down function runner")

	// Cancel all active executions
	fr.executionMutex.RLock()
	for _, execution := range fr.activeExecutions {
		if execution.cancel != nil {
			execution.cancel()
		}
	}
	fr.executionMutex.RUnlock()

	// Clear active executions
	fr.executionMutex.Lock()
	fr.activeExecutions = make(map[string]*executionContext)
	fr.executionMutex.Unlock()

	fr.logger.Info("Function runner shut down complete")
	return nil
}
