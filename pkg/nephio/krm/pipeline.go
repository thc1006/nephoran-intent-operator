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
	"math"
	"strings"
	"sync"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/pkg/nephio/porch"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Pipeline orchestrates the execution of multiple KRM functions in sequence or parallel.

type Pipeline struct {
	config *PipelineConfig

	runtime *Runtime

	registry *Registry

	stateManager *StateManager

	scheduler *FunctionScheduler

	metrics *PipelineMetrics

	tracer trace.Tracer

	mu sync.RWMutex
}

// PipelineConfig defines configuration for pipeline execution.

type PipelineConfig struct {

	// Execution settings.

	MaxConcurrentStages int `json:"maxConcurrentStages" yaml:"maxConcurrentStages"`

	StageTimeout time.Duration `json:"stageTimeout" yaml:"stageTimeout"`

	PipelineTimeout time.Duration `json:"pipelineTimeout" yaml:"pipelineTimeout"`

	RetryAttempts int `json:"retryAttempts" yaml:"retryAttempts"`

	RetryDelay time.Duration `json:"retryDelay" yaml:"retryDelay"`

	// Error handling.

	ContinueOnError bool `json:"continueOnError" yaml:"continueOnError"`

	RollbackOnFailure bool `json:"rollbackOnFailure" yaml:"rollbackOnFailure"`

	ErrorThreshold int `json:"errorThreshold" yaml:"errorThreshold"`

	FailureMode string `json:"failureMode" yaml:"failureMode"` // fail-fast, continue, rollback

	// State management.

	PersistentState bool `json:"persistentState" yaml:"persistentState"`

	StateStorage string `json:"stateStorage" yaml:"stateStorage"` // memory, file, database

	CheckpointInterval int `json:"checkpointInterval" yaml:"checkpointInterval"`

	// Performance optimization.

	EnableParallelism bool `json:"enableParallelism" yaml:"enableParallelism"`

	OptimizeExecution bool `json:"optimizeExecution" yaml:"optimizeExecution"`

	ResourcePooling bool `json:"resourcePooling" yaml:"resourcePooling"`

	// Observability.

	EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics"`

	EnableTracing bool `json:"enableTracing" yaml:"enableTracing"`

	DetailedLogging bool `json:"detailedLogging" yaml:"detailedLogging"`
}

// PipelineStage represents a stage in the pipeline.

type PipelineStage struct {

	// Basic properties.

	Name string `json:"name" yaml:"name"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	Type string `json:"type" yaml:"type"` // function, parallel, sequential, conditional

	// Execution properties.

	Functions []*StageFunction `json:"functions,omitempty" yaml:"functions,omitempty"`

	Stages []*PipelineStage `json:"stages,omitempty" yaml:"stages,omitempty"` // For nested stages

	Conditions []*Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`

	// Control flow.

	DependsOn []string `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`

	RunAfter []string `json:"runAfter,omitempty" yaml:"runAfter,omitempty"`

	RunBefore []string `json:"runBefore,omitempty" yaml:"runBefore,omitempty"`

	// Error handling.

	OnError *ErrorHandler `json:"onError,omitempty" yaml:"onError,omitempty"`

	OnSuccess *SuccessHandler `json:"onSuccess,omitempty" yaml:"onSuccess,omitempty"`

	// Timing.

	Timeout *time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	Retry *RetryPolicy `json:"retry,omitempty" yaml:"retry,omitempty"`

	// Variables and state.

	Variables map[string]Variable `json:"variables,omitempty" yaml:"variables,omitempty"`

	Outputs []string `json:"outputs,omitempty" yaml:"outputs,omitempty"`
}

// StageFunction represents a function within a stage.

type StageFunction struct {
	Name string `json:"name" yaml:"name"`

	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`

	Selectors []ResourceSelector `json:"selectors,omitempty" yaml:"selectors,omitempty"`

	InputMapping map[string]string `json:"inputMapping,omitempty" yaml:"inputMapping,omitempty"`

	OutputMapping map[string]string `json:"outputMapping,omitempty" yaml:"outputMapping,omitempty"`

	Conditions []*Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`

	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`

	ResourceFilter *ResourceFilter `json:"resourceFilter,omitempty" yaml:"resourceFilter,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// ResourceSelector defines criteria for resource selection.

type ResourceSelector struct {
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`

	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`

	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	Fields map[string]string `json:"fields,omitempty" yaml:"fields,omitempty"`
}

// ResourceFilter defines resource filtering criteria.

type ResourceFilter struct {
	Include []*ResourceSelector `json:"include,omitempty" yaml:"include,omitempty"`

	Exclude []*ResourceSelector `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}

// Variable represents a pipeline variable.

type Variable struct {
	Type string `json:"type" yaml:"type"` // string, number, boolean, object, array

	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`

	Default interface{} `json:"default,omitempty" yaml:"default,omitempty"`

	Required bool `json:"required,omitempty" yaml:"required,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	Secret bool `json:"secret,omitempty" yaml:"secret,omitempty"`
}

// Condition defines conditional execution logic.

type Condition struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"` // expression, resource-exists, variable-equals

	Expression string `json:"expression,omitempty" yaml:"expression,omitempty"`

	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	Negate bool `json:"negate,omitempty" yaml:"negate,omitempty"`
}

// ErrorHandler defines error handling behavior.

type ErrorHandler struct {
	Action string `json:"action" yaml:"action"` // continue, retry, rollback, fail

	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty" yaml:"retryPolicy,omitempty"`

	Conditions []*Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`

	Handler string `json:"handler,omitempty" yaml:"handler,omitempty"` // Reference to handler function

	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// SuccessHandler defines success handling behavior.

type SuccessHandler struct {
	Action string `json:"action" yaml:"action"` // continue, checkpoint, notification

	Handler string `json:"handler,omitempty" yaml:"handler,omitempty"`

	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// RetryPolicy defines retry behavior.

type RetryPolicy struct {
	MaxAttempts int `json:"maxAttempts" yaml:"maxAttempts"`

	InitialDelay time.Duration `json:"initialDelay" yaml:"initialDelay"`

	MaxDelay time.Duration `json:"maxDelay" yaml:"maxDelay"`

	Multiplier float64 `json:"multiplier" yaml:"multiplier"`

	RetryOn []string `json:"retryOn,omitempty" yaml:"retryOn,omitempty"` // Error types to retry on

}

// ExecutionSettings defines execution configuration.

type ExecutionSettings struct {
	Mode string `json:"mode" yaml:"mode"` // sequential, parallel, dag

	MaxConcurrency int `json:"maxConcurrency,omitempty" yaml:"maxConcurrency,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	Priority int `json:"priority,omitempty" yaml:"priority,omitempty"`

	ResourceLimits *ResourceLimits `json:"resourceLimits,omitempty" yaml:"resourceLimits,omitempty"`
}

// RollbackSettings defines rollback configuration.

type RollbackSettings struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	Strategy string `json:"strategy" yaml:"strategy"` // reverse-order, custom

	OnFailure bool `json:"onFailure" yaml:"onFailure"`

	OnCancel bool `json:"onCancel" yaml:"onCancel"`

	CustomStages []string `json:"customStages,omitempty" yaml:"customStages,omitempty"`
}

// TelecomProfile defines telecom-specific pipeline settings.

type TelecomProfile struct {
	Profile string `json:"profile" yaml:"profile"` // 5g-core, o-ran, edge

	Standards []string `json:"standards,omitempty" yaml:"standards,omitempty"`

	Interfaces []string `json:"interfaces,omitempty" yaml:"interfaces,omitempty"`

	NetworkSlices []string `json:"networkSlices,omitempty" yaml:"networkSlices,omitempty"`

	Compliance *ComplianceSettings `json:"compliance,omitempty" yaml:"compliance,omitempty"`
}

// ComplianceSettings defines compliance requirements.

type ComplianceSettings struct {
	Required bool `json:"required" yaml:"required"`

	Standards []string `json:"standards" yaml:"standards"`

	Validations []string `json:"validations" yaml:"validations"`

	Auditing bool `json:"auditing" yaml:"auditing"`
}

// PipelineExecution represents an execution of a pipeline.

type PipelineExecution struct {

	// Metadata.

	ID string `json:"id"`

	Name string `json:"name"`

	Pipeline *PipelineDefinition `json:"pipeline"`

	// Execution state.

	Status ExecutionStatus `json:"status"`

	StartTime time.Time `json:"startTime"`

	EndTime *time.Time `json:"endTime,omitempty"`

	Duration time.Duration `json:"duration"`

	// Stage tracking.

	Stages map[string]*StageExecution `json:"stages"`

	CurrentStage string `json:"currentStage,omitempty"`

	// Results.

	Resources []porch.KRMResource `json:"resources"`

	OutputResources []*porch.KRMResource `json:"outputResources"`

	Results []*ExecutionResult `json:"results"`

	Errors []ExecutionError `json:"errors"`

	// State management.

	Variables map[string]interface{} `json:"variables"`

	Checkpoints []*ExecutionCheckpoint `json:"checkpoints"`

	// Context.

	Context map[string]interface{} `json:"context"`

	Metadata map[string]string `json:"metadata"`
}

// StageExecution represents execution of a pipeline stage.

type StageExecution struct {
	Name string `json:"name"`

	Status ExecutionStatus `json:"status"`

	StartTime time.Time `json:"startTime"`

	EndTime *time.Time `json:"endTime,omitempty"`

	Duration time.Duration `json:"duration"`

	Functions map[string]*FunctionExecution `json:"functions"`

	Dependencies []*DependencyStatus `json:"dependencies"`

	Error *ExecutionError `json:"error,omitempty"`

	Retries int `json:"retries"`

	Output map[string]interface{} `json:"output"`
}

// FunctionExecution represents execution of a function within a stage.

type FunctionExecution struct {
	Name string `json:"name"`

	Image string `json:"image"`

	Status ExecutionStatus `json:"status"`

	StartTime time.Time `json:"startTime"`

	EndTime *time.Time `json:"endTime,omitempty"`

	Duration time.Duration `json:"duration"`

	Input []porch.KRMResource `json:"input"`

	InputCount int `json:"inputCount"`

	Output []porch.KRMResource `json:"output"`

	OutputCount int `json:"outputCount"`

	Results []*porch.FunctionResult `json:"results"`

	CacheHit bool `json:"cacheHit"`

	Error *ExecutionError `json:"error,omitempty"`

	Retries int `json:"retries"`

	Logs []string `json:"logs"`

	Context map[string]interface{} `json:"context"`
}

// DependencyStatus represents dependency status.

type DependencyStatus struct {
	From string `json:"from"`

	To string `json:"to"`

	Type string `json:"type"`

	Status string `json:"status"`

	Satisfied bool `json:"satisfied"`

	CheckedAt time.Time `json:"checkedAt"`
}

// ExecutionStatus represents the status of execution.

type ExecutionStatus string

const (

	// StatusPending holds statuspending value.

	StatusPending ExecutionStatus = "pending"

	// StatusRunning holds statusrunning value.

	StatusRunning ExecutionStatus = "running"

	// StatusSucceeded holds statussucceeded value.

	StatusSucceeded ExecutionStatus = "succeeded"

	// StatusFailed holds statusfailed value.

	StatusFailed ExecutionStatus = "failed"

	// StatusCancelled holds statuscancelled value.

	StatusCancelled ExecutionStatus = "cancelled"

	// StatusRolledBack holds statusrolledback value.

	StatusRolledBack ExecutionStatus = "rolledback"

	// StatusPaused holds statuspaused value.

	StatusPaused ExecutionStatus = "paused"
)

// ExecutionError represents an execution error.

type ExecutionError struct {
	Code string `json:"code"`

	Message string `json:"message"`

	Stage string `json:"stage,omitempty"`

	Function string `json:"function,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Recoverable bool `json:"recoverable"`

	Details string `json:"details,omitempty"`
}

// ExecutionResult represents an execution result.

type ExecutionResult struct {
	Stage string `json:"stage"`

	Function string `json:"function,omitempty"`

	Type string `json:"type"`

	Message string `json:"message"`

	Data map[string]interface{} `json:"data,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// ExecutionCheckpoint represents a pipeline checkpoint.

type ExecutionCheckpoint struct {
	ID string `json:"id"`

	Stage string `json:"stage"`

	Timestamp time.Time `json:"timestamp"`

	Variables map[string]interface{} `json:"variables"`

	Resources []porch.KRMResource `json:"resources"`

	Metadata map[string]string `json:"metadata"`
}

// StateManager manages pipeline execution state.

type StateManager struct {
	storage StateStorage

	checkpoints map[string]*ExecutionCheckpoint

	mu sync.RWMutex
}

// StateStorage interface for state persistence.

type StateStorage interface {
	Save(ctx context.Context, key string, data interface{}) error

	Load(ctx context.Context, key string, data interface{}) error

	Delete(ctx context.Context, key string) error

	List(ctx context.Context, prefix string) ([]string, error)
}

// FunctionScheduler manages function execution scheduling.

type FunctionScheduler struct {
	maxConcurrency int

	semaphore chan struct{}

	queue chan *ScheduledFunction

	workers int

	ctx context.Context

	cancel context.CancelFunc

	wg sync.WaitGroup
}

// ScheduledFunction represents a function scheduled for execution.

type ScheduledFunction struct {
	Function *StageFunction

	Stage string

	Resources []porch.KRMResource

	Context context.Context

	ResultChan chan *FunctionExecutionResult

	Priority int

	ScheduledAt time.Time
}

// FunctionExecutionResult represents the result of function execution.

type FunctionExecutionResult struct {
	Resources []porch.KRMResource

	Results []*porch.FunctionResult

	Error error

	Duration time.Duration

	Logs []string
}

// PipelineMetrics provides comprehensive metrics for pipeline execution.

type PipelineMetrics struct {
	PipelineExecutions *prometheus.CounterVec

	ExecutionDuration *prometheus.HistogramVec

	StageExecutions *prometheus.CounterVec

	StageDuration *prometheus.HistogramVec

	FunctionExecutions *prometheus.CounterVec

	FunctionDuration *prometheus.HistogramVec

	ErrorRate *prometheus.CounterVec

	RetryCount *prometheus.CounterVec

	QueueDepth prometheus.Gauge

	ActiveExecutions prometheus.Gauge

	CheckpointOperations *prometheus.CounterVec
}

// Default pipeline configuration.

var DefaultPipelineConfig = &PipelineConfig{

	MaxConcurrentStages: 10,

	StageTimeout: 15 * time.Minute,

	PipelineTimeout: 1 * time.Hour,

	RetryAttempts: 3,

	RetryDelay: 5 * time.Second,

	ContinueOnError: false,

	RollbackOnFailure: true,

	ErrorThreshold: 5,

	FailureMode: "fail-fast",

	PersistentState: true,

	StateStorage: "memory",

	CheckpointInterval: 5,

	EnableParallelism: true,

	OptimizeExecution: true,

	ResourcePooling: true,

	EnableMetrics: true,

	EnableTracing: true,

	DetailedLogging: false,
}

// NewPipeline creates a new KRM function pipeline with comprehensive orchestration.

func NewPipeline(config *PipelineConfig, runtime *Runtime, registry *Registry) (*Pipeline, error) {

	if config == nil {

		config = DefaultPipelineConfig

	}

	// Validate configuration.

	if err := validatePipelineConfig(config); err != nil {

		return nil, fmt.Errorf("invalid pipeline configuration: %w", err)

	}

	// Initialize metrics.

	metrics := &PipelineMetrics{

		PipelineExecutions: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_pipeline_executions_total",

				Help: "Total number of pipeline executions",
			},

			[]string{"pipeline", "status"},
		),

		ExecutionDuration: promauto.NewHistogramVec(

			prometheus.HistogramOpts{

				Name: "krm_pipeline_execution_duration_seconds",

				Help: "Duration of pipeline executions",

				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},

			[]string{"pipeline"},
		),

		StageExecutions: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_pipeline_stage_executions_total",

				Help: "Total number of stage executions",
			},

			[]string{"pipeline", "stage", "status"},
		),

		StageDuration: promauto.NewHistogramVec(

			prometheus.HistogramOpts{

				Name: "krm_pipeline_stage_duration_seconds",

				Help: "Duration of stage executions",

				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
			},

			[]string{"pipeline", "stage"},
		),

		FunctionExecutions: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_pipeline_function_executions_total",

				Help: "Total number of function executions in pipelines",
			},

			[]string{"pipeline", "stage", "function", "status"},
		),

		FunctionDuration: promauto.NewHistogramVec(

			prometheus.HistogramOpts{

				Name: "krm_pipeline_function_duration_seconds",

				Help: "Duration of function executions in pipelines",

				Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
			},

			[]string{"pipeline", "stage", "function"},
		),

		ErrorRate: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_pipeline_errors_total",

				Help: "Total number of pipeline execution errors",
			},

			[]string{"pipeline", "stage", "error_type"},
		),

		RetryCount: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_pipeline_retries_total",

				Help: "Total number of execution retries",
			},

			[]string{"pipeline", "stage", "function"},
		),

		QueueDepth: promauto.NewGauge(

			prometheus.GaugeOpts{

				Name: "krm_pipeline_queue_depth",

				Help: "Current depth of function execution queue",
			},
		),

		ActiveExecutions: promauto.NewGauge(

			prometheus.GaugeOpts{

				Name: "krm_pipeline_active_executions",

				Help: "Number of active pipeline executions",
			},
		),

		CheckpointOperations: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "krm_pipeline_checkpoint_operations_total",

				Help: "Total number of checkpoint operations",
			},

			[]string{"operation", "status"},
		),
	}

	// Initialize state manager.

	var storage StateStorage = &MemoryStateStorage{

		data: make(map[string][]byte),
	}

	stateManager := &StateManager{

		storage: storage,

		checkpoints: make(map[string]*ExecutionCheckpoint),
	}

	// Initialize function scheduler.

	ctx, cancel := context.WithCancel(context.Background())

	scheduler := &FunctionScheduler{

		maxConcurrency: config.MaxConcurrentStages,

		semaphore: make(chan struct{}, config.MaxConcurrentStages),

		queue: make(chan *ScheduledFunction, 1000),

		workers: config.MaxConcurrentStages,

		ctx: ctx,

		cancel: cancel,
	}

	pipeline := &Pipeline{

		config: config,

		runtime: runtime,

		registry: registry,

		stateManager: stateManager,

		scheduler: scheduler,

		metrics: metrics,

		tracer: otel.Tracer("krm-pipeline"),
	}

	// Start scheduler workers.

	pipeline.startScheduler()

	return pipeline, nil

}

// Execute executes a pipeline definition with comprehensive orchestration.

func (p *Pipeline) Execute(ctx context.Context, definition *PipelineDefinition, resources []porch.KRMResource) (*PipelineExecution, error) {

	ctx, span := p.tracer.Start(ctx, "krm-pipeline-execute")

	defer span.End()

	// Create execution instance.

	execution := &PipelineExecution{

		ID: generateExecutionID(),

		Name: definition.Name,

		Pipeline: definition,

		Status: StatusPending,

		StartTime: time.Now(),

		Stages: make(map[string]*StageExecution),

		Resources: resources,

		Results: []*ExecutionResult{},

		Errors: []ExecutionError{},

		Variables: p.convertAndInitializeVariables(definition.Variables),

		Checkpoints: []*ExecutionCheckpoint{},

		Context: make(map[string]interface{}),

		Metadata: make(map[string]string),
	}

	// Add execution context to span.

	span.SetAttributes(

		attribute.String("pipeline.name", definition.Name),

		attribute.String("pipeline.version", definition.Version),

		attribute.String("execution.id", execution.ID),

		attribute.Int("stages.count", len(definition.Stages)),

		attribute.Int("resources.count", len(resources)),
	)

	logger := log.FromContext(ctx).WithName("krm-pipeline").WithValues(

		"pipeline", definition.Name,

		"execution", execution.ID,
	)

	logger.Info("Starting pipeline execution",

		"stages", len(definition.Stages),

		"resources", len(resources),
	)

	// Update metrics.

	p.metrics.ActiveExecutions.Inc()

	defer p.metrics.ActiveExecutions.Dec()

	// Set execution status.

	execution.Status = StatusRunning

	// Create execution context with timeout.

	execCtx, cancel := context.WithTimeout(ctx, p.config.PipelineTimeout)

	defer cancel()

	// Execute pipeline with error handling.

	var execErr error

	defer func() {

		execution.EndTime = &time.Time{}

		*execution.EndTime = time.Now()

		execution.Duration = execution.EndTime.Sub(execution.StartTime)

		status := "success"

		if execErr != nil {

			execution.Status = StatusFailed

			status = "failed"

			logger.Error(execErr, "Pipeline execution failed")

		} else {

			execution.Status = StatusSucceeded

			logger.Info("Pipeline execution completed successfully", "duration", execution.Duration)

		}

		p.metrics.PipelineExecutions.WithLabelValues(definition.Name, status).Inc()

		p.metrics.ExecutionDuration.WithLabelValues(definition.Name).Observe(execution.Duration.Seconds())

	}()

	// Execute stages based on execution mode.

	if definition.ExecutionMode == "parallel" {

		execErr = p.executeStagesParallel(execCtx, execution)

	} else if definition.ExecutionMode == "dag" {

		execErr = p.executeStagesDAG(execCtx, execution)

	} else {

		execErr = p.executeStagesSequential(execCtx, execution)

	}

	// Handle rollback on failure.

	if execErr != nil && p.config.RollbackOnFailure {

		if rollbackErr := p.rollbackExecution(execCtx, execution); rollbackErr != nil {

			logger.Error(rollbackErr, "Rollback failed")

			execution.Status = StatusFailed

		} else {

			execution.Status = StatusRolledBack

		}

	}

	if execErr != nil {

		span.RecordError(execErr)

		span.SetStatus(codes.Error, "pipeline execution failed")

		return execution, execErr

	}

	span.SetStatus(codes.Ok, "pipeline execution completed")

	return execution, nil

}

// ExecuteStage executes a single pipeline stage.

func (p *Pipeline) ExecuteStage(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, resources []porch.KRMResource) (*StageExecution, error) {

	ctx, span := p.tracer.Start(ctx, "krm-pipeline-execute-stage")

	defer span.End()

	stageExec := &StageExecution{

		Name: stage.Name,

		Status: StatusRunning,

		StartTime: time.Now(),

		Functions: make(map[string]*FunctionExecution),

		Output: make(map[string]interface{}),
	}

	span.SetAttributes(

		attribute.String("stage.name", stage.Name),

		attribute.String("stage.type", stage.Type),

		attribute.Int("functions.count", len(stage.Functions)),
	)

	logger := log.FromContext(ctx).WithValues("stage", stage.Name)

	// Apply stage timeout.

	stageTimeout := p.config.StageTimeout

	if stage.Timeout != nil {

		stageTimeout = *stage.Timeout

	}

	stageCtx, cancel := context.WithTimeout(ctx, stageTimeout)

	defer cancel()

	// Check conditions.

	if !p.evaluateConditions(stage.Conditions, execution.Variables, resources) {

		logger.Info("Stage conditions not met, skipping")

		stageExec.Status = StatusSucceeded

		return stageExec, nil

	}

	var stageErr error

	defer func() {

		stageExec.EndTime = &time.Time{}

		*stageExec.EndTime = time.Now()

		stageExec.Duration = stageExec.EndTime.Sub(stageExec.StartTime)

		status := "success"

		if stageErr != nil {

			stageExec.Status = StatusFailed

			status = "failed"

		} else {

			stageExec.Status = StatusSucceeded

		}

		p.metrics.StageExecutions.WithLabelValues(execution.Name, stage.Name, status).Inc()

		p.metrics.StageDuration.WithLabelValues(execution.Name, stage.Name).Observe(stageExec.Duration.Seconds())

	}()

	// Execute based on stage type.

	switch stage.Type {

	case "parallel":

		stageErr = p.executeStageParallel(stageCtx, execution, stage, stageExec, resources)

	case "conditional":

		stageErr = p.executeStageConditional(stageCtx, execution, stage, stageExec, resources)

	default:

		stageErr = p.executeStageSequential(stageCtx, execution, stage, stageExec, resources)

	}

	if stageErr != nil {

		span.RecordError(stageErr)

		span.SetStatus(codes.Error, "stage execution failed")

		// Handle error based on stage configuration.

		if stage.OnError != nil {

			if handledErr := p.handleStageError(stageCtx, execution, stage, stageExec, stageErr); handledErr == nil {

				stageErr = nil // Error was handled successfully

			}

		}

		return stageExec, stageErr

	}

	span.SetStatus(codes.Ok, "stage execution completed")

	return stageExec, nil

}

// Private helper methods.

func (p *Pipeline) executeStagesSequential(ctx context.Context, execution *PipelineExecution) error {

	currentResources := execution.Resources

	for _, stage := range execution.Pipeline.Stages {

		execution.CurrentStage = stage.Name

		stageExec, err := p.ExecuteStage(ctx, execution, stage, currentResources)

		execution.Stages[stage.Name] = stageExec

		if err != nil {

			if !p.config.ContinueOnError {

				return err

			}

			execution.Errors = append(execution.Errors, ExecutionError{

				Code: "STAGE_EXECUTION_FAILED",

				Message: err.Error(),

				Stage: stage.Name,

				Timestamp: time.Now(),
			})

		}

		// Update resources with stage output.

		if stageExec.Status == StatusSucceeded {

			// Extract resources from stage execution.

			for _, funcExec := range stageExec.Functions {

				if funcExec.Status == StatusSucceeded {

					currentResources = funcExec.Output

				}

			}

		}

		// Create checkpoint.

		if p.config.PersistentState && len(execution.Stages)%p.config.CheckpointInterval == 0 {

			if err := p.createCheckpoint(ctx, execution, stage.Name, currentResources); err != nil {

				log.FromContext(ctx).Error(err, "Failed to create checkpoint")

			}

		}

	}

	// Update final resources.

	execution.Resources = currentResources

	return nil

}

func (p *Pipeline) executeStagesParallel(ctx context.Context, execution *PipelineExecution) error {

	var wg sync.WaitGroup

	var mu sync.Mutex

	var errors []error

	sem := make(chan struct{}, p.config.MaxConcurrentStages)

	for _, stage := range execution.Pipeline.Stages {

		wg.Add(1)

		go func(s *PipelineStage) {

			defer wg.Done()

			// Acquire semaphore.

			sem <- struct{}{}

			defer func() { <-sem }()

			stageExec, err := p.ExecuteStage(ctx, execution, s, execution.Resources)

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

func (p *Pipeline) executeStagesDAG(ctx context.Context, execution *PipelineExecution) error {

	// DAG execution would implement topological sorting and dependency resolution.

	// For now, fall back to sequential execution.

	return p.executeStagesSequential(ctx, execution)

}

func (p *Pipeline) executeStageSequential(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, stageExec *StageExecution, resources []porch.KRMResource) error {

	currentResources := resources

	for _, function := range stage.Functions {

		funcExec := &FunctionExecution{

			Name: function.Name,

			Image: function.Image,

			Status: StatusRunning,

			StartTime: time.Now(),

			Input: currentResources,
		}

		// Execute function.

		result, err := p.executeFunction(ctx, execution, stage, function, currentResources)

		funcExec.EndTime = &time.Time{}

		*funcExec.EndTime = time.Now()

		funcExec.Duration = funcExec.EndTime.Sub(funcExec.StartTime)

		if err != nil {

			funcExec.Status = StatusFailed

			funcExec.Error = &ExecutionError{

				Code: "FUNCTION_EXECUTION_FAILED",

				Message: err.Error(),

				Stage: stage.Name,

				Function: function.Name,

				Timestamp: time.Now(),

				Recoverable: true,
			}

			p.metrics.FunctionExecutions.WithLabelValues(

				execution.Name, stage.Name, function.Name, "failed",
			).Inc()

			if !function.Optional {

				return err

			}

		} else {

			funcExec.Status = StatusSucceeded

			funcExec.Output = result.Resources

			funcExec.Results = result.Results

			funcExec.Logs = result.Logs

			currentResources = result.Resources

			p.metrics.FunctionExecutions.WithLabelValues(

				execution.Name, stage.Name, function.Name, "success",
			).Inc()

		}

		p.metrics.FunctionDuration.WithLabelValues(

			execution.Name, stage.Name, function.Name,
		).Observe(funcExec.Duration.Seconds())

		stageExec.Functions[function.Name] = funcExec

	}

	return nil

}

func (p *Pipeline) executeStageParallel(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, stageExec *StageExecution, resources []porch.KRMResource) error {

	var wg sync.WaitGroup

	var mu sync.Mutex

	var errors []error

	for _, function := range stage.Functions {

		wg.Add(1)

		go func(f *StageFunction) {

			defer wg.Done()

			funcExec := &FunctionExecution{

				Name: f.Name,

				Image: f.Image,

				Status: StatusRunning,

				StartTime: time.Now(),

				Input: resources,
			}

			result, err := p.executeFunction(ctx, execution, stage, f, resources)

			funcExec.EndTime = &time.Time{}

			*funcExec.EndTime = time.Now()

			funcExec.Duration = funcExec.EndTime.Sub(funcExec.StartTime)

			if err != nil {

				funcExec.Status = StatusFailed

				funcExec.Error = &ExecutionError{

					Code: "FUNCTION_EXECUTION_FAILED",

					Message: err.Error(),

					Stage: stage.Name,

					Function: f.Name,

					Timestamp: time.Now(),

					Recoverable: true,
				}

				mu.Lock()

				if !f.Optional {

					errors = append(errors, err)

				}

				mu.Unlock()

			} else {

				funcExec.Status = StatusSucceeded

				funcExec.Output = result.Resources

				funcExec.Results = result.Results

				funcExec.Logs = result.Logs

			}

			mu.Lock()

			stageExec.Functions[f.Name] = funcExec

			mu.Unlock()

		}(function)

	}

	wg.Wait()

	if len(errors) > 0 {

		return fmt.Errorf("parallel stage execution failed with %d errors", len(errors))

	}

	return nil

}

func (p *Pipeline) executeStageConditional(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, stageExec *StageExecution, resources []porch.KRMResource) error {

	// For conditional stages, evaluate conditions for each function.

	return p.executeStageSequential(ctx, execution, stage, stageExec, resources)

}

func (p *Pipeline) executeFunction(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, function *StageFunction, resources []porch.KRMResource) (*porch.FunctionResponse, error) {

	// Filter resources based on selectors.

	selectedResources := p.filterResources(resources, function.Selectors)

	// Create function request.

	request := &porch.FunctionRequest{

		FunctionConfig: porch.FunctionConfig{

			Image: function.Image,

			ConfigMap: function.Config,

			Selectors: convertResourceSelectors(function.Selectors),
		},

		Resources: convertToPorchResources(selectedResources),

		Context: &porch.FunctionContext{

			Package: &porch.PackageReference{

				Repository: "pipeline-execution",

				PackageName: execution.Name,

				Revision: execution.ID,
			},

			Environment: map[string]string{

				"PIPELINE_NAME": execution.Name,

				"EXECUTION_ID": execution.ID,

				"STAGE_NAME": stage.Name,

				"FUNCTION_NAME": function.Name,
			},
		},
	}

	// Execute function using runtime.

	return p.runtime.ExecuteFunction(ctx, request)

}

func (p *Pipeline) rollbackExecution(ctx context.Context, execution *PipelineExecution) error {

	if execution.Pipeline.FailurePolicy == nil || execution.Pipeline.FailurePolicy.Mode != "rollback" {

		return nil

	}

	// Implement rollback logic.

	log.FromContext(ctx).Info("Rolling back pipeline execution", "execution", execution.ID)

	// For now, just mark as rolled back.

	execution.Status = StatusRolledBack

	return nil

}

func (p *Pipeline) handleStageError(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, stageExec *StageExecution, err error) error {

	if stage.OnError == nil {

		return err

	}

	switch stage.OnError.Action {

	case "continue":

		log.FromContext(ctx).Info("Continuing after stage error", "stage", stage.Name, "error", err)

		return nil

	case "retry":

		return p.retryStage(ctx, execution, stage, stageExec)

	case "rollback":

		return p.rollbackExecution(ctx, execution)

	default:

		return err

	}

}

func (p *Pipeline) retryStage(ctx context.Context, execution *PipelineExecution, stage *PipelineStage, stageExec *StageExecution) error {

	retryPolicy := stage.Retry

	if retryPolicy == nil {

		retryPolicy = &RetryPolicy{

			MaxAttempts: p.config.RetryAttempts,

			InitialDelay: p.config.RetryDelay,

			MaxDelay: 5 * time.Minute,

			Multiplier: 2.0,
		}

	}

	if stageExec.Retries >= retryPolicy.MaxAttempts {

		return fmt.Errorf("max retries exceeded for stage %s", stage.Name)

	}

	delay := time.Duration(float64(retryPolicy.InitialDelay) *

		math.Pow(retryPolicy.Multiplier, float64(stageExec.Retries)))

	if delay > retryPolicy.MaxDelay {

		delay = retryPolicy.MaxDelay

	}

	log.FromContext(ctx).Info("Retrying stage",

		"stage", stage.Name,

		"attempt", stageExec.Retries+1,

		"delay", delay,
	)

	time.Sleep(delay)

	stageExec.Retries++

	p.metrics.RetryCount.WithLabelValues(execution.Name, stage.Name, "").Inc()

	// Execute stage again.

	newStageExec, err := p.ExecuteStage(ctx, execution, stage, execution.Resources)

	if err == nil {

		// Replace with successful execution.

		execution.Stages[stage.Name] = newStageExec

	}

	return err

}

func (p *Pipeline) createCheckpoint(ctx context.Context, execution *PipelineExecution, stageName string, resources []porch.KRMResource) error {

	checkpoint := &ExecutionCheckpoint{

		ID: fmt.Sprintf("%s-%s-%d", execution.ID, stageName, len(execution.Checkpoints)),

		Stage: stageName,

		Timestamp: time.Now(),

		Variables: execution.Variables,

		Resources: resources,

		Metadata: map[string]string{

			"execution_id": execution.ID,

			"stage": stageName,
		},
	}

	if err := p.stateManager.SaveCheckpoint(ctx, checkpoint); err != nil {

		return err

	}

	execution.Checkpoints = append(execution.Checkpoints, checkpoint)

	p.metrics.CheckpointOperations.WithLabelValues("create", "success").Inc()

	return nil

}

func (p *Pipeline) startScheduler() {

	for range p.scheduler.workers {

		p.scheduler.wg.Add(1)

		go p.schedulerWorker()

	}

}

func (p *Pipeline) schedulerWorker() {

	defer p.scheduler.wg.Done()

	for {

		select {

		case scheduledFunc := <-p.scheduler.queue:

			p.scheduler.semaphore <- struct{}{}

			p.processScheduledFunction(scheduledFunc)

			<-p.scheduler.semaphore

		case <-p.scheduler.ctx.Done():

			return

		}

	}

}

func (p *Pipeline) processScheduledFunction(scheduled *ScheduledFunction) {

	// Process the scheduled function.

	// Implementation would execute the function and send result to ResultChan.

	result := &FunctionExecutionResult{

		Resources: scheduled.Resources,

		Results: []*porch.FunctionResult{},

		Duration: time.Second,

		Logs: []string{},
	}

	select {

	case scheduled.ResultChan <- result:

	case <-scheduled.Context.Done():

	}

}

// Shutdown gracefully shuts down the pipeline.

func (p *Pipeline) Shutdown(ctx context.Context) error {

	logger := log.FromContext(ctx).WithName("krm-pipeline")

	logger.Info("Shutting down KRM pipeline")

	// Cancel scheduler.

	p.scheduler.cancel()

	// Wait for scheduler workers.

	done := make(chan struct{})

	go func() {

		p.scheduler.wg.Wait()

		close(done)

	}()

	select {

	case <-done:

		logger.Info("Scheduler shutdown completed")

	case <-ctx.Done():

		logger.Info("Scheduler shutdown timeout")

	}

	logger.Info("KRM pipeline shutdown complete")

	return nil

}

// Helper functions and supporting types.

// MemoryStateStorage represents a memorystatestorage.

type MemoryStateStorage struct {
	data map[string][]byte

	mu sync.RWMutex
}

// Save performs save operation.

func (s *MemoryStateStorage) Save(ctx context.Context, key string, data interface{}) error {

	jsonData, err := json.Marshal(data)

	if err != nil {

		return err

	}

	s.mu.Lock()

	s.data[key] = jsonData

	s.mu.Unlock()

	return nil

}

// Load performs load operation.

func (s *MemoryStateStorage) Load(ctx context.Context, key string, data interface{}) error {

	s.mu.RLock()

	jsonData, exists := s.data[key]

	s.mu.RUnlock()

	if !exists {

		return fmt.Errorf("key not found: %s", key)

	}

	return json.Unmarshal(jsonData, data)

}

// Delete performs delete operation.

func (s *MemoryStateStorage) Delete(ctx context.Context, key string) error {

	s.mu.Lock()

	delete(s.data, key)

	s.mu.Unlock()

	return nil

}

// List performs list operation.

func (s *MemoryStateStorage) List(ctx context.Context, prefix string) ([]string, error) {

	s.mu.RLock()

	defer s.mu.RUnlock()

	var keys []string

	for key := range s.data {

		if strings.HasPrefix(key, prefix) {

			keys = append(keys, key)

		}

	}

	return keys, nil

}

// SaveCheckpoint performs savecheckpoint operation.

func (sm *StateManager) SaveCheckpoint(ctx context.Context, checkpoint *ExecutionCheckpoint) error {

	return sm.storage.Save(ctx, checkpoint.ID, checkpoint)

}

// LoadCheckpoint performs loadcheckpoint operation.

func (sm *StateManager) LoadCheckpoint(ctx context.Context, id string) (*ExecutionCheckpoint, error) {

	var checkpoint ExecutionCheckpoint

	err := sm.storage.Load(ctx, id, &checkpoint)

	return &checkpoint, err

}

// Helper functions.

func (p *Pipeline) convertAndInitializeVariables(vars map[string]*Variable) map[string]interface{} {

	result := make(map[string]interface{})

	for name, variable := range vars {

		if variable.Value != nil {

			result[name] = variable.Value

		} else if variable.Default != nil {

			result[name] = variable.Default

		}

	}

	return result

}

func (p *Pipeline) initializeVariables(vars map[string]Variable) map[string]interface{} {

	result := make(map[string]interface{})

	for name, variable := range vars {

		if variable.Value != nil {

			result[name] = variable.Value

		} else if variable.Default != nil {

			result[name] = variable.Default

		}

	}

	return result

}

func (p *Pipeline) evaluateConditions(conditions []*Condition, variables map[string]interface{}, resources []porch.KRMResource) bool {

	if len(conditions) == 0 {

		return true

	}

	for _, condition := range conditions {

		if !p.evaluateCondition(condition, variables, resources) {

			return false

		}

	}

	return true

}

func (p *Pipeline) evaluateCondition(condition *Condition, variables map[string]interface{}, resources []porch.KRMResource) bool {

	// Simple condition evaluation - could be more sophisticated.

	switch condition.Type {

	case "variable-equals":

		if value, exists := condition.Parameters["value"]; exists {

			varName := condition.Parameters["variable"].(string)

			if varValue, ok := variables[varName]; ok {

				return varValue == value

			}

		}

	case "resource-exists":

		// Check if resource exists.

		apiVersion := condition.Parameters["apiVersion"].(string)

		kind := condition.Parameters["kind"].(string)

		name := condition.Parameters["name"].(string)

		for _, resource := range resources {

			if resource.APIVersion == apiVersion &&

				resource.Kind == kind &&

				resource.Metadata["name"] == name {

				return !condition.Negate

			}

		}

		return condition.Negate

	}

	return true

}

func (p *Pipeline) filterResources(resources []porch.KRMResource, selectors []ResourceSelector) []porch.KRMResource {

	if len(selectors) == 0 {

		return resources

	}

	var filtered []porch.KRMResource

	for _, resource := range resources {

		for _, selector := range selectors {

			if p.resourceMatches(resource, selector) {

				filtered = append(filtered, resource)

				break

			}

		}

	}

	return filtered

}

func (p *Pipeline) resourceMatches(resource porch.KRMResource, selector ResourceSelector) bool {

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

	// Check labels.

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

func convertResourceSelectors(selectors []ResourceSelector) []porch.ResourceSelector {

	var result []porch.ResourceSelector

	for _, selector := range selectors {

		result = append(result, porch.ResourceSelector{

			APIVersion: selector.APIVersion,

			Kind: selector.Kind,

			Name: selector.Name,

			Namespace: selector.Namespace,

			Labels: selector.Labels,
		})

	}

	return result

}

func convertToPorchResources(resources []porch.KRMResource) []porch.KRMResource {

	return resources

}

func validatePipelineConfig(config *PipelineConfig) error {

	if config.MaxConcurrentStages <= 0 {

		return fmt.Errorf("maxConcurrentStages must be positive")

	}

	if config.StageTimeout <= 0 {

		return fmt.Errorf("stageTimeout must be positive")

	}

	if config.PipelineTimeout <= 0 {

		return fmt.Errorf("pipelineTimeout must be positive")

	}

	return nil

}
