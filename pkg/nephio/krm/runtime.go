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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// Runtime provides secure, isolated execution of KRM functions
// with comprehensive performance optimization and observability
type Runtime struct {
	config         *RuntimeConfig
	resourcePool   *ResourcePool
	executorPool   *ExecutorPool
	securityPolicy *RuntimeSecurityPolicy
	metrics        *RuntimeMetrics
	tracer         trace.Tracer
	mu             sync.RWMutex
}

// RuntimeConfig defines configuration for KRM function execution
type RuntimeConfig struct {
	// Resource limits
	MaxCPU           string        `json:"maxCpu" yaml:"maxCpu"`
	MaxMemory        string        `json:"maxMemory" yaml:"maxMemory"`
	MaxExecutionTime time.Duration `json:"maxExecutionTime" yaml:"maxExecutionTime"`
	MaxDiskSpace     string        `json:"maxDiskSpace" yaml:"maxDiskSpace"`

	// Concurrency settings
	MaxConcurrentFunctions int `json:"maxConcurrentFunctions" yaml:"maxConcurrentFunctions"`
	WorkerPoolSize         int `json:"workerPoolSize" yaml:"workerPoolSize"`
	QueueSize              int `json:"queueSize" yaml:"queueSize"`

	// Security settings
	SandboxEnabled      bool     `json:"sandboxEnabled" yaml:"sandboxEnabled"`
	AllowedImages       []string `json:"allowedImages" yaml:"allowedImages"`
	BlockedCapabilities []string `json:"blockedCapabilities" yaml:"blockedCapabilities"`
	NetworkAccess       bool     `json:"networkAccess" yaml:"networkAccess"`

	// Performance settings
	EnableCaching     bool          `json:"enableCaching" yaml:"enableCaching"`
	CacheSize         int           `json:"cacheSize" yaml:"cacheSize"`
	CacheTTL          time.Duration `json:"cacheTtl" yaml:"cacheTtl"`
	EnableCompression bool          `json:"enableCompression" yaml:"enableCompression"`

	// Observability settings
	EnableMetrics   bool `json:"enableMetrics" yaml:"enableMetrics"`
	EnableTracing   bool `json:"enableTracing" yaml:"enableTracing"`
	EnableProfiling bool `json:"enableProfiling" yaml:"enableProfiling"`
	VerboseLogging  bool `json:"verboseLogging" yaml:"verboseLogging"`

	// Workspace settings
	WorkspaceDir     string        `json:"workspaceDir" yaml:"workspaceDir"`
	CleanupInterval  time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"`
	MaxWorkspaceSize string        `json:"maxWorkspaceSize" yaml:"maxWorkspaceSize"`
}

// ExecutionContext provides context for function execution
type ExecutionContext struct {
	ID              string
	FunctionImage   string
	Resources       []porch.KRMResource
	Config          porch.FunctionConfig
	Timeout         time.Duration
	WorkspaceDir    string
	Environment     map[string]string
	SecurityContext *SecurityContext
	ParentSpan      trace.Span
	StartTime       time.Time
}

// SecurityContext defines security constraints for function execution
type SecurityContext struct {
	UserID              int
	GroupID             int
	AllowedCapabilities []string
	DropCapabilities    []string
	ReadOnlyRootFS      bool
	NoNewPrivileges     bool
	SELinuxContext      string
	SeccompProfile      string
	NetworkIsolation    bool
	FileSystemIsolation bool
}

// ExecutionResult contains the result of function execution
type ExecutionResult struct {
	Resources   []porch.KRMResource     `json:"resources"`
	Results     []*porch.FunctionResult `json:"results,omitempty"`
	Logs        string                  `json:"logs,omitempty"` // Changed from []string to string for compatibility
	Err         error                   `json:"error,omitempty"` // Changed from Error to Err to match expected interface
	Duration    time.Duration           `json:"duration"`
	MemoryUsage int64                   `json:"memoryUsage"`
	CPUUsage    float64                 `json:"cpuUsage"`
	ExitCode    int                     `json:"exitCode"`
	
	// Legacy compatibility - keep Error field but mark as deprecated
	Error       *porch.FunctionError    `json:"-"` // Hidden from JSON serialization
}

// ResourcePool manages computational resources for function execution
type ResourcePool struct {
	cpuSemaphore    chan struct{}
	memorySemaphore chan struct{}
	diskSemaphore   chan struct{}
	config          *RuntimeConfig
	mu              sync.Mutex
	activeJobs      map[string]*ExecutionContext
}

// ExecutorPool manages a pool of function executors
type ExecutorPool struct {
	workers chan *Executor
	config  *RuntimeConfig
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	metrics *RuntimeMetrics
}

// Executor handles individual function execution
type Executor struct {
	id          string
	runtime     *Runtime
	workspace   string
	containerID string
	isOccupied  bool
	lastUsed    time.Time
	mu          sync.Mutex
}

// RuntimeSecurityPolicy enforces security constraints for runtime execution
type RuntimeSecurityPolicy struct {
	allowedImages       map[string]bool
	blockedCapabilities map[string]bool
	maxResourceLimits   ResourceLimits
	networkPolicy       *NetworkPolicy
	fileSystemPolicy    *FileSystemPolicy
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	CPU      string
	Memory   string
	Disk     string
	Network  string
	FileDesc int
}

// NetworkPolicy defines network access constraints
type NetworkPolicy struct {
	AllowInternet    bool
	AllowedHosts     []string
	BlockedPorts     []int
	AllowLoopback    bool
	AllowServiceMesh bool
}

// FileSystemPolicy defines file system access constraints
type FileSystemPolicy struct {
	ReadOnlyPaths []string
	WritablePaths []string
	BlockedPaths  []string
	MaxFileSize   int64
	MaxTotalSize  int64
}

// RuntimeMetrics provides comprehensive metrics for function execution
type RuntimeMetrics struct {
	FunctionExecutions  prometheus.CounterVec
	ExecutionDuration   prometheus.HistogramVec
	ResourceUtilization prometheus.GaugeVec
	ErrorRate           prometheus.CounterVec
	QueueDepth          prometheus.Gauge
	ActiveExecutors     prometheus.Gauge
	CacheHitRate        prometheus.Counter
	SecurityViolations  prometheus.CounterVec
}

// Default configuration
var DefaultRuntimeConfig = &RuntimeConfig{
	MaxCPU:                 "2000m",
	MaxMemory:              "2Gi",
	MaxExecutionTime:       5 * time.Minute,
	MaxDiskSpace:           "1Gi",
	MaxConcurrentFunctions: 50,
	WorkerPoolSize:         20,
	QueueSize:              100,
	SandboxEnabled:         true,
	AllowedImages:          []string{},
	BlockedCapabilities:    []string{"SYS_ADMIN", "NET_ADMIN", "DAC_OVERRIDE"},
	NetworkAccess:          false,
	EnableCaching:          true,
	CacheSize:              1000,
	CacheTTL:               1 * time.Hour,
	EnableCompression:      true,
	EnableMetrics:          true,
	EnableTracing:          true,
	EnableProfiling:        false,
	VerboseLogging:         false,
	WorkspaceDir:           "/tmp/krm-runtime",
	CleanupInterval:        10 * time.Minute,
	MaxWorkspaceSize:       "100Mi",
}

// NewRuntime creates a new KRM function runtime with comprehensive capabilities
func NewRuntime(config *RuntimeConfig) (*Runtime, error) {
	if config == nil {
		config = DefaultRuntimeConfig
	}

	// Validate configuration
	if err := validateRuntimeConfig(config); err != nil {
		return nil, fmt.Errorf("invalid runtime configuration: %w", err)
	}

	// Initialize metrics
	metrics := &RuntimeMetrics{
		FunctionExecutions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_function_executions_total",
				Help: "Total number of KRM function executions",
			},
			[]string{"function", "status", "image"},
		),
		ExecutionDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "krm_function_execution_duration_seconds",
				Help:    "Duration of KRM function executions",
				Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
			},
			[]string{"function", "image"},
		),
		ResourceUtilization: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "krm_function_resource_utilization",
				Help: "Resource utilization during function execution",
			},
			[]string{"resource_type", "function"},
		),
		ErrorRate: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_function_errors_total",
				Help: "Total number of KRM function execution errors",
			},
			[]string{"function", "error_type"},
		),
		QueueDepth: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "krm_function_queue_depth",
				Help: "Current depth of function execution queue",
			},
		),
		ActiveExecutors: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "krm_function_active_executors",
				Help: "Number of active function executors",
			},
		),
		CacheHitRate: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "krm_function_cache_hits_total",
				Help: "Total number of function cache hits",
			},
		),
		SecurityViolations: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_function_security_violations_total",
				Help: "Total number of security violations during function execution",
			},
			[]string{"violation_type", "function"},
		),
	}

	// Create tracer
	tracer := otel.Tracer("krm-runtime")

	// Initialize resource pool
	resourcePool := &ResourcePool{
		cpuSemaphore:    make(chan struct{}, config.MaxConcurrentFunctions),
		memorySemaphore: make(chan struct{}, config.MaxConcurrentFunctions),
		diskSemaphore:   make(chan struct{}, config.MaxConcurrentFunctions),
		config:          config,
		activeJobs:      make(map[string]*ExecutionContext),
	}

	// Initialize executor pool
	ctx, cancel := context.WithCancel(context.Background())
	executorPool := &ExecutorPool{
		workers: make(chan *Executor, config.WorkerPoolSize),
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		metrics: metrics,
	}

	// Initialize security policy
	securityPolicy := &RuntimeSecurityPolicy{
		allowedImages:       make(map[string]bool),
		blockedCapabilities: make(map[string]bool),
		maxResourceLimits: ResourceLimits{
			CPU:      config.MaxCPU,
			Memory:   config.MaxMemory,
			Disk:     config.MaxDiskSpace,
			FileDesc: 1024,
		},
		networkPolicy: &NetworkPolicy{
			AllowInternet:    config.NetworkAccess,
			AllowLoopback:    true,
			AllowServiceMesh: true,
		},
		fileSystemPolicy: &FileSystemPolicy{
			ReadOnlyPaths: []string{"/bin", "/usr", "/lib", "/etc"},
			WritablePaths: []string{"/tmp", "/workspace"},
			BlockedPaths:  []string{"/proc", "/sys", "/dev"},
			MaxFileSize:   100 * 1024 * 1024,  // 100MB
			MaxTotalSize:  1024 * 1024 * 1024, // 1GB
		},
	}

	// Populate security policy maps
	for _, img := range config.AllowedImages {
		securityPolicy.allowedImages[img] = true
	}
	for _, cap := range config.BlockedCapabilities {
		securityPolicy.blockedCapabilities[cap] = true
	}

	runtime := &Runtime{
		config:         config,
		resourcePool:   resourcePool,
		executorPool:   executorPool,
		securityPolicy: securityPolicy,
		metrics:        metrics,
		tracer:         tracer,
	}

	// Initialize executor pool workers
	if err := runtime.initializeExecutorPool(); err != nil {
		return nil, fmt.Errorf("failed to initialize executor pool: %w", err)
	}

	// Start background cleanup routine
	go runtime.backgroundCleanup()

	return runtime, nil
}

// ExecuteFunction executes a KRM function with comprehensive security and observability
func (r *Runtime) ExecuteFunction(ctx context.Context, req *porch.FunctionRequest) (*porch.FunctionResponse, error) {
	// Start tracing
	ctx, span := r.tracer.Start(ctx, "krm-function-execution")
	defer span.End()

	// Create execution context
	execCtx := &ExecutionContext{
		ID:              generateExecutionID(),
		FunctionImage:   req.FunctionConfig.Image,
		Resources:       req.Resources,
		Config:          req.FunctionConfig,
		Timeout:         r.config.MaxExecutionTime,
		Environment:     make(map[string]string),
		SecurityContext: r.createSecurityContext(),
		ParentSpan:      span,
		StartTime:       time.Now(),
	}

	// Add execution context to span
	span.SetAttributes(
		attribute.String("function.image", execCtx.FunctionImage),
		attribute.String("execution.id", execCtx.ID),
		attribute.Int("resources.count", len(execCtx.Resources)),
	)

	// Validate function security
	if err := r.validateFunctionSecurity(execCtx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "security validation failed")
		r.metrics.SecurityViolations.WithLabelValues("validation_failed", execCtx.FunctionImage).Inc()
		return nil, fmt.Errorf("function security validation failed: %w", err)
	}

	// Acquire resources
	if err := r.resourcePool.acquire(ctx, execCtx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "resource acquisition failed")
		return nil, fmt.Errorf("failed to acquire execution resources: %w", err)
	}
	defer r.resourcePool.release(execCtx)

	// Get executor from pool
	executor, err := r.executorPool.getExecutor(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "executor acquisition failed")
		return nil, fmt.Errorf("failed to get executor: %w", err)
	}
	defer r.executorPool.releaseExecutor(executor)

	// Execute function
	result, err := executor.executeFunction(ctx, execCtx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "function execution failed")
		r.metrics.ErrorRate.WithLabelValues(execCtx.FunctionImage, "execution_failed").Inc()
		r.metrics.FunctionExecutions.WithLabelValues(
			execCtx.FunctionImage, "failed", execCtx.FunctionImage,
		).Inc()
		return nil, fmt.Errorf("function execution failed: %w", err)
	}

	// Record metrics
	duration := time.Since(execCtx.StartTime)
	r.metrics.ExecutionDuration.WithLabelValues(
		execCtx.FunctionImage, execCtx.FunctionImage,
	).Observe(duration.Seconds())
	r.metrics.FunctionExecutions.WithLabelValues(
		execCtx.FunctionImage, "success", execCtx.FunctionImage,
	).Inc()
	r.metrics.ResourceUtilization.WithLabelValues(
		"memory", execCtx.FunctionImage,
	).Set(float64(result.MemoryUsage))
	r.metrics.ResourceUtilization.WithLabelValues(
		"cpu", execCtx.FunctionImage,
	).Set(result.CPUUsage)

	// Convert to Porch response
	var logs []string
	if result.Logs != "" {
		logs = strings.Split(result.Logs, "\n")
	}
	response := &porch.FunctionResponse{
		Resources: result.Resources,
		Results:   result.Results,
		Logs:      logs,
		Error:     result.Error,
	}

	span.SetStatus(codes.Ok, "function executed successfully")
	return response, nil
}

// ExecuteBatch executes multiple functions in parallel with intelligent scheduling
func (r *Runtime) ExecuteBatch(ctx context.Context, requests []*porch.FunctionRequest) ([]*porch.FunctionResponse, error) {
	ctx, span := r.tracer.Start(ctx, "krm-function-batch-execution")
	defer span.End()

	span.SetAttributes(attribute.Int("batch.size", len(requests)))

	if len(requests) == 0 {
		return []*porch.FunctionResponse{}, nil
	}

	// Create result channels
	results := make([]*porch.FunctionResponse, len(requests))
	errors := make([]error, len(requests))

	// Create semaphore to limit concurrent executions
	sem := make(chan struct{}, r.config.MaxConcurrentFunctions)
	var wg sync.WaitGroup

	// Execute functions in parallel
	for i, req := range requests {
		wg.Add(1)
		go func(idx int, request *porch.FunctionRequest) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Execute function
			result, err := r.ExecuteFunction(ctx, request)
			results[idx] = result
			errors[idx] = err
		}(i, req)
	}

	// Wait for all executions to complete
	wg.Wait()

	// Check for errors
	var aggregatedError error
	for i, err := range errors {
		if err != nil {
			if aggregatedError == nil {
				aggregatedError = fmt.Errorf("batch execution failed")
			}
			aggregatedError = fmt.Errorf("%w; function %d: %v", aggregatedError, i, err)
		}
	}

	if aggregatedError != nil {
		span.RecordError(aggregatedError)
		span.SetStatus(codes.Error, "batch execution failed")
		return results, aggregatedError
	}

	span.SetStatus(codes.Ok, "batch execution completed")
	return results, nil
}

// Health returns runtime health status
func (r *Runtime) Health() *RuntimeHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &RuntimeHealth{
		Status:           "healthy",
		ActiveExecutions: len(r.resourcePool.activeJobs),
		PoolUtilization:  float64(len(r.executorPool.workers)) / float64(r.config.WorkerPoolSize),
		ResourceUsage: ResourceUsage{
			CPU:    r.getCurrentCPUUsage(),
			Memory: r.getCurrentMemoryUsage(),
			Disk:   r.getCurrentDiskUsage(),
		},
		LastHealthCheck: time.Now(),
	}
}

// Shutdown gracefully shuts down the runtime
func (r *Runtime) Shutdown(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("krm-runtime")
	logger.Info("Shutting down KRM runtime")

	// Cancel executor pool context
	r.executorPool.cancel()

	// Wait for active executions to complete (with timeout)
	done := make(chan struct{})
	go func() {
		r.executorPool.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All executions completed gracefully")
	case <-ctx.Done():
		logger.Info("Shutdown timeout reached, forcing shutdown")
	}

	// Cleanup resources
	if err := r.cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup runtime resources: %w", err)
	}

	logger.Info("KRM runtime shutdown complete")
	return nil
}

// Private helper methods

func (r *Runtime) initializeExecutorPool() error {
	for i := 0; i < r.config.WorkerPoolSize; i++ {
		executor, err := r.createExecutor()
		if err != nil {
			return fmt.Errorf("failed to create executor: %w", err)
		}
		r.executorPool.workers <- executor
	}
	return nil
}

func (r *Runtime) createExecutor() (*Executor, error) {
	workspace, err := r.createWorkspace()
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return &Executor{
		id:        generateExecutorID(),
		runtime:   r,
		workspace: workspace,
		lastUsed:  time.Now(),
	}, nil
}

func (r *Runtime) createWorkspace() (string, error) {
	workspaceDir := filepath.Join(r.config.WorkspaceDir, generateExecutionID())
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory: %w", err)
	}
	return workspaceDir, nil
}

func (r *Runtime) createSecurityContext() *SecurityContext {
	return &SecurityContext{
		UserID:              1000,
		GroupID:             1000,
		AllowedCapabilities: []string{},
		DropCapabilities:    r.config.BlockedCapabilities,
		ReadOnlyRootFS:      true,
		NoNewPrivileges:     true,
		NetworkIsolation:    !r.config.NetworkAccess,
		FileSystemIsolation: true,
	}
}

func (r *Runtime) validateFunctionSecurity(ctx *ExecutionContext) error {
	// Check if image is allowed
	if len(r.config.AllowedImages) > 0 {
		allowed := false
		for _, allowedImage := range r.config.AllowedImages {
			if ctx.FunctionImage == allowedImage {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("function image %s is not in allowed list", ctx.FunctionImage)
		}
	}

	// Additional security validations would go here
	return nil
}

func (r *Runtime) backgroundCleanup() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.performCleanup()
	}
}

func (r *Runtime) performCleanup() {
	// Cleanup old workspaces
	if entries, err := os.ReadDir(r.config.WorkspaceDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				dirPath := filepath.Join(r.config.WorkspaceDir, entry.Name())
				if info, err := entry.Info(); err == nil {
					if time.Since(info.ModTime()) > r.config.CleanupInterval {
						os.RemoveAll(dirPath)
					}
				}
			}
		}
	}
}

func (r *Runtime) cleanup() error {
	// Cleanup all workspaces
	if err := os.RemoveAll(r.config.WorkspaceDir); err != nil {
		return err
	}
	return nil
}

// Resource pool methods

func (rp *ResourcePool) acquire(ctx context.Context, execCtx *ExecutionContext) error {
	// Acquire resource semaphores
	select {
	case rp.cpuSemaphore <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case rp.memorySemaphore <- struct{}{}:
	case <-ctx.Done():
		<-rp.cpuSemaphore // Release CPU semaphore
		return ctx.Err()
	}

	select {
	case rp.diskSemaphore <- struct{}{}:
	case <-ctx.Done():
		<-rp.cpuSemaphore    // Release CPU semaphore
		<-rp.memorySemaphore // Release memory semaphore
		return ctx.Err()
	}

	// Add to active jobs
	rp.mu.Lock()
	rp.activeJobs[execCtx.ID] = execCtx
	rp.mu.Unlock()

	return nil
}

func (rp *ResourcePool) release(execCtx *ExecutionContext) {
	// Release semaphores
	<-rp.cpuSemaphore
	<-rp.memorySemaphore
	<-rp.diskSemaphore

	// Remove from active jobs
	rp.mu.Lock()
	delete(rp.activeJobs, execCtx.ID)
	rp.mu.Unlock()
}

// Executor pool methods

func (ep *ExecutorPool) getExecutor(ctx context.Context) (*Executor, error) {
	select {
	case executor := <-ep.workers:
		ep.metrics.ActiveExecutors.Inc()
		return executor, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (ep *ExecutorPool) releaseExecutor(executor *Executor) {
	executor.lastUsed = time.Now()
	ep.workers <- executor
	ep.metrics.ActiveExecutors.Dec()
}

// Executor methods

func (e *Executor) executeFunction(ctx context.Context, execCtx *ExecutionContext) (*ExecutionResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.isOccupied = true
	defer func() { e.isOccupied = false }()

	// Prepare workspace
	execCtx.WorkspaceDir = e.workspace
	if err := e.prepareWorkspace(execCtx); err != nil {
		return nil, fmt.Errorf("failed to prepare workspace: %w", err)
	}

	// Create execution command
	cmd, err := e.createCommand(ctx, execCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution command: %w", err)
	}

	// Execute function
	startTime := time.Now()
	result, err := e.runCommand(ctx, cmd, execCtx)
	if err != nil {
		return nil, fmt.Errorf("function execution failed: %w", err)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (e *Executor) prepareWorkspace(execCtx *ExecutionContext) error {
	// Create input file
	inputFile := filepath.Join(execCtx.WorkspaceDir, "input.yaml")
	inputData, err := json.Marshal(execCtx.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal input resources: %w", err)
	}

	if err := os.WriteFile(inputFile, inputData, 0644); err != nil {
		return fmt.Errorf("failed to write input file: %w", err)
	}

	// Create config file if needed
	if execCtx.Config.ConfigMap != nil {
		configFile := filepath.Join(execCtx.WorkspaceDir, "config.yaml")
		configData, err := json.Marshal(execCtx.Config.ConfigMap)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := os.WriteFile(configFile, configData, 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
	}

	return nil
}

func (e *Executor) createCommand(ctx context.Context, execCtx *ExecutionContext) (*exec.Cmd, error) {
	// For now, implement as a container execution
	// In production, this would use proper container runtime integration
	args := []string{
		"run", "--rm",
		"--cpu-limit", e.runtime.config.MaxCPU,
		"--memory-limit", e.runtime.config.MaxMemory,
		"--user", fmt.Sprintf("%d:%d", execCtx.SecurityContext.UserID, execCtx.SecurityContext.GroupID),
		"--volume", fmt.Sprintf("%s:/workspace", execCtx.WorkspaceDir),
		"--workdir", "/workspace",
	}

	// Add security constraints
	if execCtx.SecurityContext.ReadOnlyRootFS {
		args = append(args, "--read-only")
	}
	if execCtx.SecurityContext.NoNewPrivileges {
		args = append(args, "--security-opt", "no-new-privileges")
	}
	if execCtx.SecurityContext.NetworkIsolation {
		args = append(args, "--network", "none")
	}

	// Add drop capabilities
	for _, cap := range execCtx.SecurityContext.DropCapabilities {
		args = append(args, "--cap-drop", cap)
	}

	// Add function image
	args = append(args, execCtx.FunctionImage)

	cmd := exec.CommandContext(ctx, "docker", args...)
	return cmd, nil
}

func (e *Executor) runCommand(ctx context.Context, cmd *exec.Cmd, execCtx *ExecutionContext) (*ExecutionResult, error) {
	// Set up pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Create result
	result := &ExecutionResult{
		Resources: []porch.KRMResource{},
		Results:   []*porch.FunctionResult{},
		Logs:      "", // String instead of slice
	}

	// Read output and collect logs
	var logLines []string
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			logLines = append(logLines, scanner.Text())
		}
	}()

	// Read stdout (should contain the transformed resources)
	outputBytes, err := io.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	// Wait for command completion
	err = cmd.Wait()
	exitCode := 0
	if exitError, ok := err.(*exec.ExitError); ok {
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			exitCode = status.ExitStatus()
		}
	}

	result.ExitCode = exitCode

	// Parse output
	if len(outputBytes) > 0 {
		if err := json.Unmarshal(outputBytes, &result.Resources); err != nil {
			// If JSON parsing fails, treat as error
			result.Err = fmt.Errorf("failed to parse function output: %w", err)
			// Also set the legacy Error field for backward compatibility
			result.Error = &porch.FunctionError{
				Message: fmt.Sprintf("Failed to parse function output: %v", err),
				Code:    "OUTPUT_PARSE_ERROR",
				Details: string(outputBytes),
			}
		}
	}

	// Combine logs into single string
	if len(logLines) > 0 {
		result.Logs = strings.Join(logLines, "\n")
	}

	// Set error if exit code is non-zero
	if exitCode != 0 && result.Err == nil {
		result.Err = fmt.Errorf("function execution failed with exit code %d", exitCode)
		// Also set the legacy Error field for backward compatibility
		result.Error = &porch.FunctionError{
			Message: fmt.Sprintf("Function execution failed with exit code %d", exitCode),
			Code:    "EXECUTION_ERROR",
		}
	}

	return result, nil
}

// Supporting types and methods

type RuntimeHealth struct {
	Status           string        `json:"status"`
	ActiveExecutions int           `json:"activeExecutions"`
	PoolUtilization  float64       `json:"poolUtilization"`
	ResourceUsage    ResourceUsage `json:"resourceUsage"`
	LastHealthCheck  time.Time     `json:"lastHealthCheck"`
}

type ResourceUsage struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
	Disk   float64 `json:"disk"`
}

// Helper functions

func validateRuntimeConfig(config *RuntimeConfig) error {
	if config.MaxConcurrentFunctions <= 0 {
		return fmt.Errorf("maxConcurrentFunctions must be positive")
	}
	if config.WorkerPoolSize <= 0 {
		return fmt.Errorf("workerPoolSize must be positive")
	}
	if config.MaxExecutionTime <= 0 {
		return fmt.Errorf("maxExecutionTime must be positive")
	}
	return nil
}


func generateExecutorID() string {
	return fmt.Sprintf("executor-%d-%d", time.Now().UnixNano(), runtime.NumGoroutine())
}

func (r *Runtime) getCurrentCPUUsage() float64 {
	// Implementation would get actual CPU usage
	return 0.0
}

func (r *Runtime) getCurrentMemoryUsage() float64 {
	// Implementation would get actual memory usage
	return 0.0
}

func (r *Runtime) getCurrentDiskUsage() float64 {
	// Implementation would get actual disk usage
	return 0.0
}
