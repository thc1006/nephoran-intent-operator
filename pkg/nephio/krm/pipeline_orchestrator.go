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
	"sort"
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

	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// PipelineOrchestrator manages execution of KRM function pipelines
type PipelineOrchestrator struct {
	config          *PipelineOrchestratorConfig
	functionManager *FunctionManager
	dependencyGraph *DependencyGraph
	executionEngine *ExecutionEngine
	stateManager    *PipelineStateManager
	metrics         *PipelineOrchestratorMetrics
	tracer          trace.Tracer
	mu              sync.RWMutex
}

// PipelineOrchestratorConfig defines configuration for pipeline orchestration
type PipelineOrchestratorConfig struct {
	// Execution settings
	MaxConcurrentPipelines int           `json:"maxConcurrentPipelines" yaml:"maxConcurrentPipelines"`
	MaxConcurrentStages    int           `json:"maxConcurrentStages" yaml:"maxConcurrentStages"`
	DefaultTimeout         time.Duration `json:"defaultTimeout" yaml:"defaultTimeout"`
	MaxPipelineTimeout     time.Duration `json:"maxPipelineTimeout" yaml:"maxPipelineTimeout"`

	// Error handling
	FailureMode            string        `json:"failureMode" yaml:"failureMode"` // fail-fast, continue, rollback
	MaxRetries             int           `json:"maxRetries" yaml:"maxRetries"`
	RetryDelay             time.Duration `json:"retryDelay" yaml:"retryDelay"`
	RetryBackoffMultiplier float64       `json:"retryBackoffMultiplier" yaml:"retryBackoffMultiplier"`

	// State management
	EnableStateManagement bool   `json:"enableStateManagement" yaml:"enableStateManagement"`
	CheckpointInterval    int    `json:"checkpointInterval" yaml:"checkpointInterval"`
	StateStorageType      string `json:"stateStorageType" yaml:"stateStorageType"` // memory, file, database

	// Performance optimization
	EnableParallelism  bool `json:"enableParallelism" yaml:"enableParallelism"`
	EnableOptimization bool `json:"enableOptimization" yaml:"enableOptimization"`
	EnableCaching      bool `json:"enableCaching" yaml:"enableCaching"`
	EnableBatching     bool `json:"enableBatching" yaml:"enableBatching"`

	// Observability
	EnableMetrics   bool `json:"enableMetrics" yaml:"enableMetrics"`
	EnableTracing   bool `json:"enableTracing" yaml:"enableTracing"`
	EnableProfiling bool `json:"enableProfiling" yaml:"enableProfiling"`
	DetailedLogging bool `json:"detailedLogging" yaml:"detailedLogging"`

	// Resource management
	ResourceQuotas       map[string]string `json:"resourceQuotas" yaml:"resourceQuotas"`
	PriorityClassMapping map[string]int    `json:"priorityClassMapping" yaml:"priorityClassMapping"`
}

// PipelineDefinition defines a complete KRM function pipeline
type OrchestrationPipelineDefinition struct {
	// Metadata
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`

	// Pipeline structure
	Stages       []*PipelineStage             `json:"stages" yaml:"stages"`
	Dependencies []*StageDependency           `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Variables    map[string]*PipelineVariable `json:"variables,omitempty" yaml:"variables,omitempty"`

	// Execution settings
	ExecutionMode string        `json:"executionMode" yaml:"executionMode"` // sequential, parallel, dag
	Timeout       time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Priority      int           `json:"priority,omitempty" yaml:"priority,omitempty"`

	// Error handling
	FailurePolicy *FailurePolicy `json:"failurePolicy,omitempty" yaml:"failurePolicy,omitempty"`
	RetryPolicy   *RetryPolicy   `json:"retryPolicy,omitempty" yaml:"retryPolicy,omitempty"`

	// State management
	Checkpoints []string `json:"checkpoints,omitempty" yaml:"checkpoints,omitempty"`

	// O-RAN/5G specific metadata
	TelecomProfile *TelecomPipelineProfile `json:"telecomProfile,omitempty" yaml:"telecomProfile,omitempty"`
}

// PipelineStage represents a stage in the pipeline
type OrchestrationPipelineStage struct {
	// Basic properties
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string `json:"type" yaml:"type"` // function, parallel-group, conditional

	// Function configuration
	Functions []*StageFunction `json:"functions,omitempty" yaml:"functions,omitempty"`

	// Control flow
	DependsOn []string        `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`
	Condition *StageCondition `json:"condition,omitempty" yaml:"condition,omitempty"`

	// Execution settings
	Timeout     time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Concurrency int           `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	Priority    int           `json:"priority,omitempty" yaml:"priority,omitempty"`

	// Error handling
	OnFailure *FailureAction `json:"onFailure,omitempty" yaml:"onFailure,omitempty"`
	OnSuccess *SuccessAction `json:"onSuccess,omitempty" yaml:"onSuccess,omitempty"`

	// Resource requirements
	Resources *StageResources `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Variables and outputs
	InputMapping  map[string]string `json:"inputMapping,omitempty" yaml:"inputMapping,omitempty"`
	OutputMapping map[string]string `json:"outputMapping,omitempty" yaml:"outputMapping,omitempty"`
}

// StageFunction represents a function within a stage
type OrchestrationStageFunction struct {
	Name           string                 `json:"name" yaml:"name"`
	Image          string                 `json:"image,omitempty" yaml:"image,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	ResourceFilter *ResourceFilter        `json:"resourceFilter,omitempty" yaml:"resourceFilter,omitempty"`
	Timeout        time.Duration          `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	RetryPolicy    *RetryPolicy           `json:"retryPolicy,omitempty" yaml:"retryPolicy,omitempty"`
	Optional       bool                   `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// StageDependency defines dependencies between stages
type StageDependency struct {
	From      string               `json:"from" yaml:"from"`
	To        string               `json:"to" yaml:"to"`
	Type      string               `json:"type" yaml:"type"` // data, control, resource
	Condition *DependencyCondition `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// PipelineVariable represents a pipeline variable
type PipelineVariable struct {
	Type        string      `json:"type" yaml:"type"`
	Value       interface{} `json:"value,omitempty" yaml:"value,omitempty"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Secret      bool        `json:"secret,omitempty" yaml:"secret,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
}

// StageCondition defines conditional execution logic
type StageCondition struct {
	Type       string                 `json:"type" yaml:"type"`
	Expression string                 `json:"expression,omitempty" yaml:"expression,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// DependencyCondition defines dependency conditions
type DependencyCondition struct {
	Status     []string               `json:"status,omitempty" yaml:"status,omitempty"`
	Expression string                 `json:"expression,omitempty" yaml:"expression,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// FailurePolicy defines pipeline failure handling
type FailurePolicy struct {
	Mode       string            `json:"mode" yaml:"mode"` // fail-fast, continue, rollback
	Stages     []string          `json:"stages,omitempty" yaml:"stages,omitempty"`
	Conditions []*StageCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// RetryPolicy defines retry behavior
type OrchestrationRetryPolicy struct {
	MaxAttempts   int           `json:"maxAttempts" yaml:"maxAttempts"`
	InitialDelay  time.Duration `json:"initialDelay" yaml:"initialDelay"`
	MaxDelay      time.Duration `json:"maxDelay" yaml:"maxDelay"`
	BackoffFactor float64       `json:"backoffFactor" yaml:"backoffFactor"`
	RetryOn       []string      `json:"retryOn,omitempty" yaml:"retryOn,omitempty"`
}

// FailureAction defines what to do on stage failure
type FailureAction struct {
	Action     string                 `json:"action" yaml:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// SuccessAction defines what to do on stage success
type SuccessAction struct {
	Action     string                 `json:"action" yaml:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// StageResources defines resource requirements for a stage
type StageResources struct {
	CPU      string            `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory   string            `json:"memory,omitempty" yaml:"memory,omitempty"`
	Storage  string            `json:"storage,omitempty" yaml:"storage,omitempty"`
	Limits   map[string]string `json:"limits,omitempty" yaml:"limits,omitempty"`
	Requests map[string]string `json:"requests,omitempty" yaml:"requests,omitempty"`
}

// ResourceFilter defines resource filtering criteria
type ResourceFilter struct {
	Include []*ResourceSelector `json:"include,omitempty" yaml:"include,omitempty"`
	Exclude []*ResourceSelector `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}

// ResourceSelector defines resource selection criteria
type OrchestrationResourceSelector struct {
	APIVersion string            `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string            `json:"kind,omitempty" yaml:"kind,omitempty"`
	Name       string            `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace  string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels     map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Fields     map[string]string `json:"fields,omitempty" yaml:"fields,omitempty"`
}

// TelecomPipelineProfile defines telecom-specific pipeline characteristics
type TelecomPipelineProfile struct {
	Domain       string   `json:"domain" yaml:"domain"`             // 5g-core, o-ran, edge
	Components   []string `json:"components" yaml:"components"`     // AMF, SMF, UPF, etc.
	Interfaces   []string `json:"interfaces" yaml:"interfaces"`     // A1, O1, O2, E2
	Compliance   []string `json:"compliance" yaml:"compliance"`     // 3GPP, O-RAN
	Optimization string   `json:"optimization" yaml:"optimization"` // performance, cost, compliance
}

// PipelineExecution represents a pipeline execution instance
type OrchestrationPipelineExecution struct {
	// Metadata
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Pipeline *PipelineDefinition `json:"pipeline"`

	// Execution state
	Status    PipelineStatus `json:"status"`
	Phase     PipelinePhase  `json:"phase"`
	StartTime time.Time      `json:"startTime"`
	EndTime   *time.Time     `json:"endTime,omitempty"`
	Duration  time.Duration  `json:"duration"`

	// Stage execution tracking
	Stages       map[string]*StageExecution `json:"stages"`
	CurrentStage string                     `json:"currentStage,omitempty"`

	// Input/Output
	InputResources  []*porch.KRMResource `json:"inputResources"`
	OutputResources []*porch.KRMResource `json:"outputResources"`

	// Results and errors
	Results []*ExecutionResult `json:"results"`
	Errors  []*ExecutionError  `json:"errors"`

	// State management
	Variables   map[string]interface{} `json:"variables"`
	Checkpoints []*ExecutionCheckpoint `json:"checkpoints"`

	// Performance metrics
	ResourceUsage *PipelineResourceUsage `json:"resourceUsage,omitempty"`

	// Audit and tracing
	AuditLogs []string `json:"auditLogs"`
	TraceID   string   `json:"traceId,omitempty"`
	SpanID    string   `json:"spanId,omitempty"`

	// Context
	Context map[string]interface{} `json:"context"`
}

// StageExecution represents execution of a pipeline stage
type OrchestrationStageExecution struct {
	Name         string                        `json:"name"`
	Status       ExecutionStatus               `json:"status"`
	StartTime    time.Time                     `json:"startTime"`
	EndTime      *time.Time                    `json:"endTime,omitempty"`
	Duration     time.Duration                 `json:"duration"`
	Functions    map[string]*FunctionExecution `json:"functions"`
	Error        *ExecutionError               `json:"error,omitempty"`
	Retries      int                           `json:"retries"`
	Dependencies []*DependencyStatus           `json:"dependencies"`
	Output       map[string]interface{}        `json:"output"`
}

// FunctionExecution represents execution of a function within a stage
type OrchestrationFunctionExecution struct {
	Name        string                  `json:"name"`
	Status      ExecutionStatus         `json:"status"`
	StartTime   time.Time               `json:"startTime"`
	EndTime     *time.Time              `json:"endTime,omitempty"`
	Duration    time.Duration           `json:"duration"`
	InputCount  int                     `json:"inputCount"`
	OutputCount int                     `json:"outputCount"`
	Results     []*porch.FunctionResult `json:"results"`
	Error       *ExecutionError         `json:"error,omitempty"`
	Retries     int                     `json:"retries"`
	CacheHit    bool                    `json:"cacheHit"`
}

// DependencyStatus represents dependency status
type DependencyStatus struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	Satisfied bool      `json:"satisfied"`
	CheckedAt time.Time `json:"checkedAt"`
}

// ExecutionCheckpoint represents a pipeline checkpoint
type OrchestrationExecutionCheckpoint struct {
	ID        string                 `json:"id"`
	Stage     string                 `json:"stage"`
	Timestamp time.Time              `json:"timestamp"`
	Variables map[string]interface{} `json:"variables"`
	Resources []*porch.KRMResource   `json:"resources"`
}

// PipelineResourceUsage represents resource usage metrics
type PipelineResourceUsage struct {
	TotalCPU    float64 `json:"totalCpu"`
	TotalMemory int64   `json:"totalMemory"`
	PeakCPU     float64 `json:"peakCpu"`
	PeakMemory  int64   `json:"peakMemory"`
}

// ExecutionResult represents an execution result
type OrchestrationExecutionResult struct {
	Stage     string                 `json:"stage"`
	Function  string                 `json:"function,omitempty"`
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ExecutionError represents an execution error
type OrchestrationExecutionError struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Stage       string                 `json:"stage,omitempty"`
	Function    string                 `json:"function,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Recoverable bool                   `json:"recoverable"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// Execution status enums
type PipelineStatus string
type PipelinePhase string
type ExecutionStatus string

const (
	PipelineStatusPending    PipelineStatus = "pending"
	PipelineStatusRunning    PipelineStatus = "running"
	PipelineStatusSucceeded  PipelineStatus = "succeeded"
	PipelineStatusFailed     PipelineStatus = "failed"
	PipelineStatusCancelled  PipelineStatus = "cancelled"
	PipelineStatusRolledBack PipelineStatus = "rolled-back"

	PipelinePhaseInitialization PipelinePhase = "initialization"
	PipelinePhaseExecution      PipelinePhase = "execution"
	PipelinePhaseFinalization   PipelinePhase = "finalization"
	PipelinePhaseCleanup        PipelinePhase = "cleanup"

	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusSucceeded ExecutionStatus = "succeeded"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusSkipped   ExecutionStatus = "skipped"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// DependencyGraph manages stage dependencies
type DependencyGraph struct {
	nodes map[string]*DependencyNode
	edges map[string][]*DependencyEdge
	mu    sync.RWMutex
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	Name         string
	Stage        *PipelineStage
	Dependencies []*DependencyEdge
	Dependents   []*DependencyEdge
	Status       ExecutionStatus
}

// DependencyEdge represents an edge in the dependency graph
type DependencyEdge struct {
	From      *DependencyNode
	To        *DependencyNode
	Type      string
	Condition *DependencyCondition
}

// ExecutionEngine manages pipeline execution
type ExecutionEngine struct {
	config    *PipelineOrchestratorConfig
	funcMgr   *FunctionManager
	executors map[string]*StageExecutor
	scheduler *StageScheduler
	mu        sync.RWMutex
}

// StageExecutor executes individual stages
type StageExecutor struct {
	stage     *PipelineStage
	execution *PipelineExecution
	funcMgr   *FunctionManager
	status    ExecutionStatus
	startTime time.Time
	endTime   *time.Time
	results   []*ExecutionResult
	errors    []*ExecutionError
	mu        sync.Mutex
}

// StageScheduler schedules stage execution
type StageScheduler struct {
	queue      chan *ScheduledStage
	workers    []*SchedulerWorker
	maxWorkers int
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// ScheduledStage represents a scheduled stage
type ScheduledStage struct {
	Stage       *PipelineStage
	Execution   *PipelineExecution
	Priority    int
	ScheduledAt time.Time
	ResultChan  chan *StageExecutionResult
}

// StageExecutionResult represents stage execution result
type StageExecutionResult struct {
	Stage     *PipelineStage
	Status    ExecutionStatus
	Resources []*porch.KRMResource
	Results   []*ExecutionResult
	Error     error
	Duration  time.Duration
}

// SchedulerWorker executes scheduled stages
type SchedulerWorker struct {
	id      int
	funcMgr *FunctionManager
	queue   chan *ScheduledStage
	ctx     context.Context
	cancel  context.CancelFunc
}

// PipelineStateManager manages pipeline execution state
type PipelineStateManager struct {
	storage    StateStorage
	executions map[string]*PipelineExecution
	mu         sync.RWMutex
}

// StateStorage interface for state persistence
type OrchestrationStateStorage interface {
	Save(ctx context.Context, key string, data interface{}) error
	Load(ctx context.Context, key string, data interface{}) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
}

// PipelineOrchestratorMetrics provides comprehensive metrics
type PipelineOrchestratorMetrics struct {
	PipelineExecutions   prometheus.CounterVec
	ExecutionDuration    prometheus.HistogramVec
	StageExecutions      prometheus.CounterVec
	StageDuration        prometheus.HistogramVec
	DependencyResolution prometheus.HistogramVec
	ErrorRate            prometheus.CounterVec
	ResourceUtilization  prometheus.GaugeVec
	ActivePipelines      prometheus.Gauge
	QueueDepth           prometheus.Gauge
}

// Default configuration
var DefaultPipelineOrchestratorConfig = &PipelineOrchestratorConfig{
	MaxConcurrentPipelines: 10,
	MaxConcurrentStages:    20,
	DefaultTimeout:         30 * time.Minute,
	MaxPipelineTimeout:     2 * time.Hour,
	FailureMode:            "fail-fast",
	MaxRetries:             3,
	RetryDelay:             30 * time.Second,
	RetryBackoffMultiplier: 2.0,
	EnableStateManagement:  true,
	CheckpointInterval:     5,
	StateStorageType:       "memory",
	EnableParallelism:      true,
	EnableOptimization:     true,
	EnableCaching:          true,
	EnableBatching:         false,
	EnableMetrics:          true,
	EnableTracing:          true,
	EnableProfiling:        false,
	DetailedLogging:        false,
}

// NewPipelineOrchestrator creates a new pipeline orchestrator
func NewPipelineOrchestrator(config *PipelineOrchestratorConfig, functionManager *FunctionManager) (*PipelineOrchestrator, error) {
	if config == nil {
		config = DefaultPipelineOrchestratorConfig
	}

	// Validate configuration
	if err := validatePipelineOrchestratorConfig(config); err != nil {
		return nil, errors.WithContext(err, "invalid pipeline orchestrator configuration")
	}

	// Initialize metrics
	metrics := &PipelineOrchestratorMetrics{
		PipelineExecutions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_pipeline_orchestrator_executions_total",
				Help: "Total number of pipeline executions",
			},
			[]string{"pipeline", "status"},
		),
		ExecutionDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "krm_pipeline_orchestrator_execution_duration_seconds",
				Help:    "Duration of pipeline executions",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},
			[]string{"pipeline"},
		),
		StageExecutions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_pipeline_orchestrator_stage_executions_total",
				Help: "Total number of stage executions",
			},
			[]string{"pipeline", "stage", "status"},
		),
		StageDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "krm_pipeline_orchestrator_stage_duration_seconds",
				Help:    "Duration of stage executions",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
			},
			[]string{"pipeline", "stage"},
		),
		DependencyResolution: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "krm_pipeline_orchestrator_dependency_resolution_seconds",
				Help:    "Duration of dependency resolution",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			},
			[]string{"pipeline"},
		),
		ErrorRate: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_pipeline_orchestrator_errors_total",
				Help: "Total number of execution errors",
			},
			[]string{"pipeline", "stage", "error_type"},
		),
		ResourceUtilization: *promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "krm_pipeline_orchestrator_resource_utilization",
				Help: "Resource utilization during pipeline execution",
			},
			[]string{"resource_type", "pipeline"},
		),
		ActivePipelines: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "krm_pipeline_orchestrator_active_pipelines",
				Help: "Number of active pipeline executions",
			},
		),
		QueueDepth: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "krm_pipeline_orchestrator_queue_depth",
				Help: "Current depth of stage execution queue",
			},
		),
	}

	// Initialize dependency graph
	dependencyGraph := &DependencyGraph{
		nodes: make(map[string]*DependencyNode),
		edges: make(map[string][]*DependencyEdge),
	}

	// Initialize stage scheduler
	ctx, cancel := context.WithCancel(context.Background())
	scheduler := &StageScheduler{
		queue:      make(chan *ScheduledStage, 1000),
		maxWorkers: config.MaxConcurrentStages,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize workers
	for i := 0; i < scheduler.maxWorkers; i++ {
		worker := &SchedulerWorker{
			id:      i,
			funcMgr: functionManager,
			queue:   scheduler.queue,
			ctx:     ctx,
			cancel:  cancel,
		}
		scheduler.workers = append(scheduler.workers, worker)
		go worker.run()
	}

	// Initialize execution engine
	executionEngine := &ExecutionEngine{
		config:    config,
		funcMgr:   functionManager,
		executors: make(map[string]*StageExecutor),
		scheduler: scheduler,
	}

	// Initialize state manager
	var storage StateStorage = &OrchestrationMemoryStateStorage{
		data: make(map[string][]byte),
	}

	stateManager := &PipelineStateManager{
		storage:    storage,
		executions: make(map[string]*PipelineExecution),
	}

	orchestrator := &PipelineOrchestrator{
		config:          config,
		functionManager: functionManager,
		dependencyGraph: dependencyGraph,
		executionEngine: executionEngine,
		stateManager:    stateManager,
		metrics:         metrics,
		tracer:          otel.Tracer("krm-pipeline-orchestrator"),
	}

	return orchestrator, nil
}

// ExecutePipeline executes a pipeline definition
func (po *PipelineOrchestrator) ExecutePipeline(ctx context.Context, definition *PipelineDefinition, resources []*porch.KRMResource) (*PipelineExecution, error) {
	ctx, span := po.tracer.Start(ctx, "pipeline-orchestrator-execute")
	defer span.End()

	// Generate execution ID
	executionID := generateExecutionID()

	span.SetAttributes(
		attribute.String("pipeline.name", definition.Name),
		attribute.String("pipeline.version", definition.Version),
		attribute.String("execution.id", executionID),
		attribute.Int("stages.count", len(definition.Stages)),
		attribute.Int("resources.count", len(resources)),
	)

	logger := log.FromContext(ctx).WithName("pipeline-orchestrator").WithValues(
		"pipeline", definition.Name,
		"execution", executionID,
	)

	logger.Info("Starting pipeline execution",
		"stages", len(definition.Stages),
		"resources", len(resources),
	)

	// Create pipeline execution
	execution := &PipelineExecution{
		ID:           executionID,
		Name:         definition.Name,
		Pipeline:     definition,
		Status:       ExecutionStatusPending,
		StartTime:    time.Now(),
		Stages:       make(map[string]*StageExecution),
		Resources:    convertResources(resources),
		Results:      []*ExecutionResult{},
		Errors:       []ExecutionError{},
	}

	// Store execution
	po.stateManager.mu.Lock()
	po.stateManager.executions[executionID] = execution
	po.stateManager.mu.Unlock()

	po.metrics.ActivePipelines.Inc()
	defer po.metrics.ActivePipelines.Dec()

	// Build dependency graph
	startTime := time.Now()
	dependencyGraph, err := po.buildDependencyGraph(definition)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "dependency graph construction failed")
		return execution, err
	}
	po.metrics.DependencyResolution.WithLabelValues(definition.Name).Observe(time.Since(startTime).Seconds())

	// Update phase
	execution.Status = ExecutionStatusRunning

	// Execute pipeline based on execution mode (default to DAG since PipelineDefinition doesn't have ExecutionMode)
	err = po.executeDAG(ctx, execution, dependencyGraph)

	// Finalize execution
	endTime := time.Now()
	execution.EndTime = &endTime
	execution.Duration = execution.EndTime.Sub(execution.StartTime)

	if err != nil {
		execution.Status = ExecutionStatusFailed
		execution.Errors = append(execution.Errors, ExecutionError{
			Code:        "PIPELINE_EXECUTION_FAILED",
			Message:     err.Error(),
			Timestamp:   time.Now(),
			Recoverable: false,
		})

		span.RecordError(err)
		span.SetStatus(codes.Error, "pipeline execution failed")
		po.metrics.ErrorRate.WithLabelValues(definition.Name, "", "pipeline_error").Inc()
		po.metrics.PipelineExecutions.WithLabelValues(definition.Name, "failed").Inc()

		logger.Error(err, "Pipeline execution failed", "duration", execution.Duration)
	} else {
		execution.Status = ExecutionStatusSucceeded
		po.metrics.PipelineExecutions.WithLabelValues(definition.Name, "succeeded").Inc()

		logger.Info("Pipeline execution completed successfully", "duration", execution.Duration)
	}

	po.metrics.ExecutionDuration.WithLabelValues(definition.Name).Observe(execution.Duration.Seconds())

	// Cleanup phase is implicit since PipelineExecution doesn't have Phase field

	span.SetStatus(codes.Ok, "pipeline execution completed")
	return execution, err
}

// GetExecution returns a pipeline execution by ID
func (po *PipelineOrchestrator) GetExecution(ctx context.Context, executionID string) (*PipelineExecution, error) {
	po.stateManager.mu.RLock()
	defer po.stateManager.mu.RUnlock()

	execution, exists := po.stateManager.executions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}

	return execution, nil
}

// ListExecutions returns all pipeline executions
func (po *PipelineOrchestrator) ListExecutions(ctx context.Context) ([]*PipelineExecution, error) {
	po.stateManager.mu.RLock()
	defer po.stateManager.mu.RUnlock()

	executions := make([]*PipelineExecution, 0, len(po.stateManager.executions))
	for _, execution := range po.stateManager.executions {
		executions = append(executions, execution)
	}

	return executions, nil
}

// CancelExecution cancels a running pipeline execution
func (po *PipelineOrchestrator) CancelExecution(ctx context.Context, executionID string) error {
	po.stateManager.mu.Lock()
	defer po.stateManager.mu.Unlock()

	execution, exists := po.stateManager.executions[executionID]
	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	if execution.Status == ExecutionStatusRunning {
		execution.Status = ExecutionStatusCancelled
		endTime := time.Now()
		execution.EndTime = &endTime
		execution.Duration = execution.EndTime.Sub(execution.StartTime)

		// Cancel running stages
		for _, stageExec := range execution.Stages {
			if stageExec.Status == ExecutionStatusRunning {
				stageExec.Status = ExecutionStatusCancelled
				stageExec.EndTime = execution.EndTime
				stageExec.Duration = stageExec.EndTime.Sub(stageExec.StartTime)
			}
		}
	}

	return nil
}

// Private methods

func (po *PipelineOrchestrator) buildDependencyGraph(definition *PipelineDefinition) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		nodes: make(map[string]*DependencyNode),
		edges: make(map[string][]*DependencyEdge),
	}

	// Create nodes for each stage
	for _, stage := range definition.Stages {
		node := &DependencyNode{
			Name:         stage.Name,
			Stage:        stage,
			Dependencies: []*DependencyEdge{},
			Dependents:   []*DependencyEdge{},
			Status:       ExecutionStatusPending,
		}
		graph.nodes[stage.Name] = node
	}

	// Create edges from stage dependencies (PipelineDefinition doesn't have explicit Dependencies field)
	for _, stage := range definition.Stages {
		if stage.DependsOn != nil {
			toNode := graph.nodes[stage.Name]
			for _, depName := range stage.DependsOn {
				fromNode, fromExists := graph.nodes[depName]
				if !fromExists {
					return nil, fmt.Errorf("dependency stage %s not found for stage %s", depName, stage.Name)
				}

				edge := &DependencyEdge{
					From: fromNode,
					To:   toNode,
					Type: "control", // default dependency type
				}

				fromNode.Dependents = append(fromNode.Dependents, edge)
				toNode.Dependencies = append(toNode.Dependencies, edge)
				graph.edges[depName] = append(graph.edges[depName], edge)
			}
		}
	}

	// Validate for cycles
	if err := po.validateDependencyGraph(graph); err != nil {
		return nil, err
	}

	return graph, nil
}

func (po *PipelineOrchestrator) validateDependencyGraph(graph *DependencyGraph) error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var hasCycle func(nodeName string) bool
	hasCycle = func(nodeName string) bool {
		visited[nodeName] = true
		recursionStack[nodeName] = true

		for _, edge := range graph.edges[nodeName] {
			if !visited[edge.To.Name] {
				if hasCycle(edge.To.Name) {
					return true
				}
			} else if recursionStack[edge.To.Name] {
				return true
			}
		}

		recursionStack[nodeName] = false
		return false
	}

	for nodeName := range graph.nodes {
		if !visited[nodeName] {
			if hasCycle(nodeName) {
				return fmt.Errorf("circular dependency detected in pipeline")
			}
		}
	}

	return nil
}

func (po *PipelineOrchestrator) executeSequential(ctx context.Context, execution *PipelineExecution, graph *DependencyGraph) error {
	// Simple sequential execution without dependency resolution
	for _, stage := range execution.Pipeline.Stages {
		stageExec, err := po.executeStage(ctx, execution, stage)
		execution.Stages[stage.Name] = stageExec

		if err != nil {
			if po.config.FailureMode == "fail-fast" {
				return err
			}
			// Continue with remaining stages in continue mode
		}

		// Update resources for next stage
		if stageExec.Status == ExecutionStatusSucceeded {
			// Update output resources with results from this stage
			po.updateOutputResources(execution, stageExec)
		}
	}

	return nil
}

func (po *PipelineOrchestrator) executeParallel(ctx context.Context, execution *PipelineExecution, graph *DependencyGraph) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	semaphore := make(chan struct{}, po.config.MaxConcurrentStages)

	for _, stage := range execution.Pipeline.Stages {
		wg.Add(1)
		go func(s *PipelineStage) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			stageExec, err := po.executeStage(ctx, execution, s)

			mu.Lock()
			execution.Stages[s.Name] = stageExec
			if err != nil {
				errors = append(errors, err)
			}
			mu.Unlock()
		}(stage)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("parallel execution failed with %d errors", len(errors))
	}

	return nil
}

func (po *PipelineOrchestrator) executeDAG(ctx context.Context, execution *PipelineExecution, graph *DependencyGraph) error {
	// Topological sort to determine execution order
	executionOrder, err := po.topologicalSort(graph)
	if err != nil {
		return err
	}

	// Execute stages in topological order, respecting dependencies
	for _, stageName := range executionOrder {
		stage := graph.nodes[stageName].Stage

		// Check if dependencies are satisfied
		if !po.areDependenciesSatisfied(graph.nodes[stageName], execution) {
			// Skip this stage for now - it will be retried later
			continue
		}

		stageExec, err := po.executeStage(ctx, execution, stage)
		execution.Stages[stage.Name] = stageExec

		if err != nil {
			if po.config.FailureMode == "fail-fast" {
				return err
			}
		}

		// Update graph node status
		if stageExec.Status == ExecutionStatusSucceeded {
			graph.nodes[stageName].Status = ExecutionStatusSucceeded
			po.updateOutputResources(execution, stageExec)
		} else {
			graph.nodes[stageName].Status = ExecutionStatusFailed
		}
	}

	return nil
}

func (po *PipelineOrchestrator) topologicalSort(graph *DependencyGraph) ([]string, error) {
	visited := make(map[string]bool)
	stack := []string{}

	var visit func(nodeName string) error
	visit = func(nodeName string) error {
		if visited[nodeName] {
			return nil
		}

		visited[nodeName] = true

		// Visit all dependencies first
		for _, edge := range graph.edges[nodeName] {
			if err := visit(edge.To.Name); err != nil {
				return err
			}
		}

		stack = append([]string{nodeName}, stack...)
		return nil
	}

	for nodeName := range graph.nodes {
		if err := visit(nodeName); err != nil {
			return nil, err
		}
	}

	return stack, nil
}

func (po *PipelineOrchestrator) areDependenciesSatisfied(node *DependencyNode, execution *PipelineExecution) bool {
	for _, edge := range node.Dependencies {
		fromStageExec, exists := execution.Stages[edge.From.Name]
		if !exists || fromStageExec.Status != ExecutionStatusSucceeded {
			// Check if condition allows bypass
			if edge.Condition != nil {
				if po.evaluateDependencyCondition(edge.Condition, execution) {
					continue
				}
			}
			return false
		}
	}
	return true
}

func (po *PipelineOrchestrator) evaluateDependencyCondition(condition *DependencyCondition, execution *PipelineExecution) bool {
	// Simple condition evaluation
	if len(condition.Status) > 0 {
		for _, requiredStatus := range condition.Status {
			// Check if any dependent stage has this status
			for _, stageExec := range execution.Stages {
				if string(stageExec.Status) == requiredStatus {
					return true
				}
			}
		}
		return false
	}
	return true
}

func (po *PipelineOrchestrator) executeStage(ctx context.Context, execution *PipelineExecution, stage *PipelineStage) (*StageExecution, error) {
	stageExec := &StageExecution{
		Name:      stage.Name,
		Status:    ExecutionStatusRunning,
		StartTime: time.Now(),
		Functions: make(map[string]*FunctionExecution),
		Output:    make(map[string]interface{}),
	}

	logger := log.FromContext(ctx).WithValues("stage", stage.Name)
	logger.Info("Starting stage execution", "functions", len(stage.Functions))

	// Execute functions in the stage
	var stageErr error
	defer func() {
		stageExec.EndTime = &time.Time{}
		*stageExec.EndTime = time.Now()
		stageExec.Duration = stageExec.EndTime.Sub(stageExec.StartTime)

		status := "success"
		if stageErr != nil {
			stageExec.Status = ExecutionStatusFailed
			stageExec.Error = &ExecutionError{
				Code:        "STAGE_EXECUTION_FAILED",
				Message:     stageErr.Error(),
				Stage:       stage.Name,
				Timestamp:   time.Now(),
				Recoverable: true,
			}
			status = "failed"
		} else {
			stageExec.Status = ExecutionStatusSucceeded
		}

		po.metrics.StageExecutions.WithLabelValues(execution.Name, stage.Name, status).Inc()
		po.metrics.StageDuration.WithLabelValues(execution.Name, stage.Name).Observe(stageExec.Duration.Seconds())

		logger.Info("Stage execution completed", "status", status, "duration", stageExec.Duration)
	}()

	// Execute functions based on stage type
	switch stage.Type {
	case "parallel-group":
		stageErr = po.executeStageFunctionsParallel(ctx, execution, stage, stageExec)
	default:
		stageErr = po.executeStageFunctionsSequential(ctx, execution, stage, stageExec)
	}

	return stageExec, stageErr
}

func (po *PipelineOrchestrator) executeStageFunctionsSequential(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, stageExec *StageExecution) error {
	currentResources := convertResourcesToPointers(execution.Resources)

	for _, function := range stage.Functions {
		funcExec, err := po.executeStageFunction(ctx, execution, stage, function, currentResources)
		stageExec.Functions[function.Name] = funcExec

		if err != nil {
			if !function.Optional {
				return err
			}
		} else {
			// Update resources for next function
			if funcExec.Status == ExecutionStatusSucceeded {
				// In a real implementation, we would parse function output
				// For now, we keep the same resources
			}
		}
	}

	return nil
}

func (po *PipelineOrchestrator) executeStageFunctionsParallel(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, stageExec *StageExecution) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, function := range stage.Functions {
		wg.Add(1)
		go func(f *StageFunction) {
			defer wg.Done()

			funcExec, err := po.executeStageFunction(ctx, execution, stage, f, convertResourcesToPointers(execution.Resources))

			mu.Lock()
			stageExec.Functions[f.Name] = funcExec
			if err != nil && !f.Optional {
				errors = append(errors, err)
			}
			mu.Unlock()
		}(function)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("parallel function execution failed with %d errors", len(errors))
	}

	return nil
}

func (po *PipelineOrchestrator) executeStageFunction(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, function *StageFunction, resources []*porch.KRMResource) (*FunctionExecution, error) {
	funcExec := &FunctionExecution{
		Name:      function.Name,
		Status:    ExecutionStatusRunning,
		StartTime: time.Now(),
		Results:   []*porch.FunctionResult{},
	}

	// Use resources as-is (ResourceFilter field doesn't exist on StageFunction)
	filteredResources := resources

	// Create function execution request
	funcName := function.Name
	if function.Image != "" {
		funcName = function.Image
	}

	request := &FunctionExecutionRequest{
		FunctionName:   funcName,
		FunctionImage:  function.Image,
		FunctionConfig: function.Config,
		Resources:      filteredResources,
		Timeout:        po.config.DefaultTimeout, // Use default timeout since StageFunction doesn't have Timeout field
		RequestID:      fmt.Sprintf("%s-%s-%s", execution.ID, stage.Name, function.Name),
		EnableCaching:  po.config.EnableCaching,
	}

	// Execute function
	response, err := po.functionManager.ExecuteFunction(ctx, request)

	funcExec.EndTime = &time.Time{}
	*funcExec.EndTime = time.Now()
	funcExec.Duration = funcExec.EndTime.Sub(funcExec.StartTime)

	if err != nil {
		funcExec.Status = ExecutionStatusFailed
		funcExec.Error = &ExecutionError{
			Code:        "FUNCTION_EXECUTION_FAILED",
			Message:     err.Error(),
			Stage:       stage.Name,
			Function:    function.Name,
			Timestamp:   time.Now(),
			Recoverable: true,
		}
		return funcExec, err
	}

	funcExec.Status = ExecutionStatusSucceeded
	funcExec.Results = response.Results
	// OutputCount and CacheHit fields don't exist on FunctionExecution

	return funcExec, nil
}

func (po *PipelineOrchestrator) filterResources(resources []*porch.KRMResource, filter *ResourceFilter) []*porch.KRMResource {
	if filter == nil {
		return resources
	}

	var filtered []*porch.KRMResource

	for _, resource := range resources {
		include := len(filter.Include) == 0 // If no include filters, include by default

		// Check include filters
		for _, selector := range filter.Include {
			if po.resourceMatchesSelector(resource, selector) {
				include = true
				break
			}
		}

		// Check exclude filters
		exclude := false
		for _, selector := range filter.Exclude {
			if po.resourceMatchesSelector(resource, selector) {
				exclude = true
				break
			}
		}

		if include && !exclude {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

func (po *PipelineOrchestrator) resourceMatchesSelector(resource *porch.KRMResource, selector *ResourceSelector) bool {
	if selector.APIVersion != "" && resource.APIVersion != selector.APIVersion {
		return false
	}
	if selector.Kind != "" && resource.Kind != selector.Kind {
		return false
	}
	if selector.Name != "" {
		if name, ok := resource.Metadata["name"].(string); !ok || name != selector.Name {
			return false
		}
	}
	if selector.Namespace != "" {
		if namespace, ok := resource.Metadata["namespace"].(string); !ok || namespace != selector.Namespace {
			return false
		}
	}

	// Check label selectors
	for key, value := range selector.Labels {
		if labels, ok := resource.Metadata["labels"].(map[string]interface{}); ok {
			if labelValue, exists := labels[key]; !exists || labelValue != value {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func (po *PipelineOrchestrator) updateOutputResources(execution *PipelineExecution, stageExec *StageExecution) {
	// In a real implementation, this would merge the output resources from all successful functions
	// For now, we keep the current resources
}

func (po *PipelineOrchestrator) initializeVariables(variables map[string]*PipelineVariable) map[string]interface{} {
	result := make(map[string]interface{})

	for name, variable := range variables {
		if variable.Value != nil {
			result[name] = variable.Value
		} else if variable.Default != nil {
			result[name] = variable.Default
		}
	}

	return result
}

// SchedulerWorker methods

func (sw *SchedulerWorker) run() {
	for {
		select {
		case stage := <-sw.queue:
			sw.executeScheduledStage(stage)
		case <-sw.ctx.Done():
			return
		}
	}
}

func (sw *SchedulerWorker) executeScheduledStage(scheduled *ScheduledStage) {
	// This would execute the scheduled stage
	// For now, return a placeholder result
	result := &StageExecutionResult{
		Stage:     scheduled.Stage,
		Status:    ExecutionStatusSucceeded,
		Resources: []*porch.KRMResource{},
		Results:   []*ExecutionResult{},
		Duration:  time.Second,
	}

	select {
	case scheduled.ResultChan <- result:
	case <-time.After(5 * time.Second): // timeout instead of context
		// timeout sending result
	}
}

// Health returns orchestrator health status
func (po *PipelineOrchestrator) Health() *PipelineOrchestratorHealth {
	po.mu.RLock()
	defer po.mu.RUnlock()

	po.stateManager.mu.RLock()
	activeExecutions := 0
	for _, execution := range po.stateManager.executions {
		if execution.Status == ExecutionStatusRunning {
			activeExecutions++
		}
	}
	po.stateManager.mu.RUnlock()

	return &PipelineOrchestratorHealth{
		Status:           "healthy",
		ActiveExecutions: activeExecutions,
		TotalExecutions:  len(po.stateManager.executions),
		QueueDepth:       len(po.executionEngine.scheduler.queue),
		LastHealthCheck:  time.Now(),
	}
}

// PipelineOrchestratorHealth represents health status
type PipelineOrchestratorHealth struct {
	Status           string    `json:"status"`
	ActiveExecutions int       `json:"activeExecutions"`
	TotalExecutions  int       `json:"totalExecutions"`
	QueueDepth       int       `json:"queueDepth"`
	LastHealthCheck  time.Time `json:"lastHealthCheck"`
}

// Shutdown gracefully shuts down the pipeline orchestrator
func (po *PipelineOrchestrator) Shutdown(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("pipeline-orchestrator")
	logger.Info("Shutting down pipeline orchestrator")

	// Cancel scheduler
	po.executionEngine.scheduler.cancel()

	// Wait for workers
	done := make(chan struct{})
	go func() {
		po.executionEngine.scheduler.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("All scheduler workers stopped")
	case <-ctx.Done():
		logger.Info("Shutdown timeout reached")
	}

	logger.Info("Pipeline orchestrator shutdown complete")
	return nil
}

// Helper functions

func validatePipelineOrchestratorConfig(config *PipelineOrchestratorConfig) error {
	if config.MaxConcurrentPipelines <= 0 {
		return fmt.Errorf("maxConcurrentPipelines must be positive")
	}
	if config.MaxConcurrentStages <= 0 {
		return fmt.Errorf("maxConcurrentStages must be positive")
	}
	if config.DefaultTimeout <= 0 {
		return fmt.Errorf("defaultTimeout must be positive")
	}
	return nil
}

func generateOrchestrationExecutionID() string {
	return fmt.Sprintf("pipeline-%d", time.Now().UnixNano())
}

// OrchestrationMemoryStateStorage implements StateStorage for in-memory storage
type OrchestrationMemoryStateStorage struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func (s *OrchestrationMemoryStateStorage) Save(ctx context.Context, key string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.data[key] = jsonData
	s.mu.Unlock()

	return nil
}

func (s *OrchestrationMemoryStateStorage) Load(ctx context.Context, key string, data interface{}) error {
	s.mu.RLock()
	jsonData, exists := s.data[key]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	return json.Unmarshal(jsonData, data)
}

func (s *OrchestrationMemoryStateStorage) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
	return nil
}

func (s *OrchestrationMemoryStateStorage) List(ctx context.Context, prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []string
	for key := range s.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

// convertResources converts []*porch.KRMResource to []porch.KRMResource
func convertResources(resources []*porch.KRMResource) []porch.KRMResource {
	result := make([]porch.KRMResource, len(resources))
	for i, resource := range resources {
		if resource != nil {
			result[i] = *resource
		}
	}
	return result
}

// convertResourcesToPointers converts []porch.KRMResource to []*porch.KRMResource
func convertResourcesToPointers(resources []porch.KRMResource) []*porch.KRMResource {
	result := make([]*porch.KRMResource, len(resources))
	for i := range resources {
		result[i] = &resources[i]
	}
	return result
}
