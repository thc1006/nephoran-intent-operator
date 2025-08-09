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

package parallel

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
)

// ParallelProcessingEngine manages concurrent intent processing with worker pools
type ParallelProcessingEngine struct {
	// Core configuration
	config *ProcessingEngineConfig
	logger logr.Logger

	// Worker pools
	intentPool     *WorkerPool
	llmPool        *WorkerPool
	ragPool        *WorkerPool
	resourcePool   *WorkerPool
	manifestPool   *WorkerPool
	gitopsPool     *WorkerPool
	deploymentPool *WorkerPool

	// Task queues and scheduling
	intentQueue     *PriorityQueue
	taskScheduler   *TaskScheduler
	dependencyGraph *DependencyGraph
	loadBalancer    *LoadBalancer

	// State management
	processingStates sync.Map // map[string]*ProcessingState
	metrics          *ProcessingMetrics

	// Resource management
	resourceLimiter     *ResourceLimiter
	backpressureManager *BackpressureManager

	// Lifecycle
	mutex        sync.RWMutex
	started      bool
	stopChan     chan struct{}
	healthTicker *time.Ticker
}

// ProcessingEngineConfig holds configuration for the parallel processing engine
type ProcessingEngineConfig struct {
	// Worker pool configurations
	IntentWorkers     int `json:"intentWorkers"`
	LLMWorkers        int `json:"llmWorkers"`
	RAGWorkers        int `json:"ragWorkers"`
	ResourceWorkers   int `json:"resourceWorkers"`
	ManifestWorkers   int `json:"manifestWorkers"`
	GitOpsWorkers     int `json:"gitopsWorkers"`
	DeploymentWorkers int `json:"deploymentWorkers"`

	// Queue configurations
	MaxQueueSize int           `json:"maxQueueSize"`
	QueueTimeout time.Duration `json:"queueTimeout"`

	// Performance tuning
	MaxConcurrentIntents int           `json:"maxConcurrentIntents"`
	ProcessingTimeout    time.Duration `json:"processingTimeout"`
	HealthCheckInterval  time.Duration `json:"healthCheckInterval"`

	// Resource limits
	MaxMemoryPerWorker int64   `json:"maxMemoryPerWorker"`
	MaxCPUPerWorker    float64 `json:"maxCpuPerWorker"`

	// Backpressure settings
	BackpressureEnabled   bool    `json:"backpressureEnabled"`
	BackpressureThreshold float64 `json:"backpressureThreshold"`

	// Load balancing
	LoadBalancingStrategy string `json:"loadBalancingStrategy"`

	// Retry and circuit breaker
	MaxRetries            int  `json:"maxRetries"`
	CircuitBreakerEnabled bool `json:"circuitBreakerEnabled"`

	// Monitoring
	MetricsEnabled  bool `json:"metricsEnabled"`
	DetailedLogging bool `json:"detailedLogging"`
}

// WorkerPool manages a pool of workers for specific processing tasks
type WorkerPool struct {
	name        string
	workers     []*Worker
	taskQueue   chan *Task
	resultQueue chan *TaskResult
	workerCount int

	// State tracking
	activeWorkers  int32
	processedTasks int64
	failedTasks    int64
	averageLatency time.Duration

	// Resource management
	memoryUsage int64
	cpuUsage    float64

	mutex    sync.RWMutex
	logger   logr.Logger
	stopChan chan struct{}
}

// Worker represents an individual worker in a pool
type Worker struct {
	id          int
	poolName    string
	taskQueue   chan *Task
	resultQueue chan *TaskResult
	processor   TaskProcessor

	// State
	currentTask    *Task
	lastActivity   time.Time
	totalProcessed int64
	totalErrors    int64
	averageLatency time.Duration

	// Resource tracking
	memoryUsage int64
	cpuUsage    float64

	mutex    sync.RWMutex
	logger   logr.Logger
	stopChan chan struct{}
}

// Task represents a processing task
type Task struct {
	ID       string                     `json:"id"`
	Type     TaskType                   `json:"type"`
	Priority int                        `json:"priority"`
	IntentID string                     `json:"intentId"`
	Phase    interfaces.ProcessingPhase `json:"phase"`

	// Task data
	Intent    *nephoranv1.NetworkIntent `json:"-"`
	InputData map[string]interface{}    `json:"inputData,omitempty"`
	Context   context.Context           `json:"-"`

	// Dependencies
	Dependencies []string `json:"dependencies,omitempty"`
	Dependents   []string `json:"dependents,omitempty"`

	// Execution control
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"maxRetries"`
	RetryCount int           `json:"retryCount"`

	// Timing
	CreatedAt   time.Time  `json:"createdAt"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Callbacks
	OnSuccess  func(*TaskResult)   `json:"-"`
	OnFailure  func(*TaskResult)   `json:"-"`
	OnProgress func(*TaskProgress) `json:"-"`
}

// TaskType defines the type of processing task
type TaskType string

const (
	TaskTypeIntentProcessing   TaskType = "intent_processing"
	TaskTypeLLMProcessing      TaskType = "llm_processing"
	TaskTypeRAGRetrieval       TaskType = "rag_retrieval"
	TaskTypeResourcePlanning   TaskType = "resource_planning"
	TaskTypeManifestGeneration TaskType = "manifest_generation"
	TaskTypeGitOpsCommit       TaskType = "gitops_commit"
	TaskTypeDeploymentVerify   TaskType = "deployment_verify"
)

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID   string        `json:"taskId"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	WorkerID int           `json:"workerId"`
	PoolName string        `json:"poolName"`

	// Result data
	OutputData       map[string]interface{}       `json:"outputData,omitempty"`
	ProcessingResult *interfaces.ProcessingResult `json:"processingResult,omitempty"`

	// Error information
	Error        error  `json:"-"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	ErrorCode    string `json:"errorCode,omitempty"`

	// Performance metrics
	MemoryUsed int64   `json:"memoryUsed"`
	CPUUsed    float64 `json:"cpuUsed"`

	// Timing details
	QueueTime      time.Duration `json:"queueTime"`
	ProcessingTime time.Duration `json:"processingTime"`

	CompletedAt time.Time `json:"completedAt"`
}

// TaskProgress represents progress information for long-running tasks
type TaskProgress struct {
	TaskID         string     `json:"taskId"`
	Phase          string     `json:"phase"`
	Progress       float64    `json:"progress"` // 0.0 to 1.0
	Message        string     `json:"message"`
	EstimatedETA   *time.Time `json:"estimatedEta,omitempty"`
	CurrentStep    string     `json:"currentStep"`
	TotalSteps     int        `json:"totalSteps"`
	CompletedSteps int        `json:"completedSteps"`
}

// TaskProcessor defines the interface for task processing
type TaskProcessor interface {
	ProcessTask(ctx context.Context, task *Task) (*TaskResult, error)
	GetProcessorType() TaskType
	HealthCheck(ctx context.Context) error
	GetMetrics() map[string]interface{}
}

// PriorityQueue manages task prioritization
type PriorityQueue struct {
	tasks       []*Task
	mutex       sync.RWMutex
	notEmpty    *sync.Cond
	maxSize     int
	currentSize int
}

// TaskScheduler manages task scheduling and dependencies
type TaskScheduler struct {
	engine          *ParallelProcessingEngine
	schedulingQueue chan *Task

	// Dependency management
	dependencyGraph *DependencyGraph
	readyTasks      chan *Task

	// Scheduling strategies
	strategies      map[string]SchedulingStrategy
	currentStrategy string

	logger   logr.Logger
	stopChan chan struct{}
}

// SchedulingStrategy defines task scheduling strategies
type SchedulingStrategy interface {
	ScheduleTask(task *Task, availableWorkers map[string]int) (string, error) // Returns pool name
	GetStrategyName() string
	GetMetrics() map[string]float64
}

// DependencyGraph manages task dependencies
type DependencyGraph struct {
	nodes map[string]*DependencyNode
	mutex sync.RWMutex
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	TaskID       string
	Dependencies []*DependencyNode
	Dependents   []*DependencyNode
	Completed    bool
	Failed       bool
	mutex        sync.RWMutex
}

// LoadBalancer distributes tasks across worker pools
type LoadBalancer struct {
	strategies map[string]LoadBalancingStrategy
	current    string

	// Pool monitoring
	poolMetrics map[string]*PoolMetrics

	mutex  sync.RWMutex
	logger logr.Logger
}

// LoadBalancingStrategy defines load balancing strategies
type LoadBalancingStrategy interface {
	SelectPool(pools map[string]*WorkerPool, task *Task) (string, error)
	GetStrategyName() string
	UpdateMetrics(poolName string, metrics *PoolMetrics)
}

// PoolMetrics tracks metrics for each worker pool
type PoolMetrics struct {
	ActiveWorkers  int32         `json:"activeWorkers"`
	QueueLength    int           `json:"queueLength"`
	AverageLatency time.Duration `json:"averageLatency"`
	SuccessRate    float64       `json:"successRate"`
	MemoryUsage    int64         `json:"memoryUsage"`
	CPUUsage       float64       `json:"cpuUsage"`
	ThroughputRate float64       `json:"throughputRate"`
	LastUpdated    time.Time     `json:"lastUpdated"`
}

// ResourceLimiter manages resource constraints
type ResourceLimiter struct {
	maxMemory     int64
	maxCPU        float64
	currentMemory int64
	currentCPU    float64

	memoryWaiters []chan struct{}
	cpuWaiters    []chan struct{}

	mutex  sync.RWMutex
	logger logr.Logger
}

// BackpressureManager handles system backpressure
type BackpressureManager struct {
	config      *BackpressureConfig
	currentLoad float64
	thresholds  map[string]float64
	actions     map[string]BackpressureAction

	metrics *BackpressureMetrics

	mutex  sync.RWMutex
	logger logr.Logger
}

// BackpressureConfig configures backpressure management
type BackpressureConfig struct {
	Enabled          bool                          `json:"enabled"`
	LoadThreshold    float64                       `json:"loadThreshold"`
	Actions          map[string]BackpressureAction `json:"actions"`
	EvaluationWindow time.Duration                 `json:"evaluationWindow"`
}

// BackpressureAction defines actions to take under backpressure
type BackpressureAction struct {
	Name       string                 `json:"name"`
	Threshold  float64                `json:"threshold"`
	Action     string                 `json:"action"` // throttle, reject, shed_load, degrade
	Parameters map[string]interface{} `json:"parameters"`
}

// BackpressureMetrics tracks backpressure metrics
type BackpressureMetrics struct {
	CurrentLoad       float64   `json:"currentLoad"`
	ThrottledRequests int64     `json:"throttledRequests"`
	RejectedRequests  int64     `json:"rejectedRequests"`
	ShedRequests      int64     `json:"shedRequests"`
	DegradedRequests  int64     `json:"degradedRequests"`
	LastAction        string    `json:"lastAction"`
	LastActionTime    time.Time `json:"lastActionTime"`
}

// ProcessingState tracks the state of intent processing
type ProcessingState struct {
	IntentID     string                     `json:"intentId"`
	CurrentPhase interfaces.ProcessingPhase `json:"currentPhase"`
	Progress     float64                    `json:"progress"`

	// Task tracking
	ActiveTasks    map[string]*Task       `json:"activeTasks"`
	CompletedTasks map[string]*TaskResult `json:"completedTasks"`
	FailedTasks    map[string]*TaskResult `json:"failedTasks"`

	// Timing
	StartedAt           time.Time  `json:"startedAt"`
	LastUpdated         time.Time  `json:"lastUpdated"`
	EstimatedCompletion *time.Time `json:"estimatedCompletion,omitempty"`

	// Dependencies
	BlockedTasks []string `json:"blockedTasks"`
	ReadyTasks   []string `json:"readyTasks"`

	mutex sync.RWMutex
}

// ProcessingMetrics tracks overall processing metrics
type ProcessingMetrics struct {
	// Throughput metrics
	TotalIntentsProcessed int64   `json:"totalIntentsProcessed"`
	SuccessfulIntents     int64   `json:"successfulIntents"`
	FailedIntents         int64   `json:"failedIntents"`
	IntentsPerSecond      float64 `json:"intentsPerSecond"`

	// Latency metrics
	AverageProcessingTime time.Duration `json:"averageProcessingTime"`
	P95ProcessingTime     time.Duration `json:"p95ProcessingTime"`
	P99ProcessingTime     time.Duration `json:"p99ProcessingTime"`

	// Resource metrics
	TotalMemoryUsage int64   `json:"totalMemoryUsage"`
	TotalCPUUsage    float64 `json:"totalCpuUsage"`

	// Pool metrics
	PoolMetrics map[string]*PoolMetrics `json:"poolMetrics"`

	// Task metrics
	TaskMetrics map[TaskType]*TaskTypeMetrics `json:"taskMetrics"`

	// Queue metrics
	QueueDepth       int           `json:"queueDepth"`
	AverageQueueTime time.Duration `json:"averageQueueTime"`

	// Error metrics
	ErrorRate    float64          `json:"errorRate"`
	ErrorsByType map[string]int64 `json:"errorsByType"`

	// Last updated
	LastUpdated time.Time `json:"lastUpdated"`

	mutex sync.RWMutex
}

// TaskTypeMetrics tracks metrics for each task type
type TaskTypeMetrics struct {
	TotalTasks      int64         `json:"totalTasks"`
	SuccessfulTasks int64         `json:"successfulTasks"`
	FailedTasks     int64         `json:"failedTasks"`
	AverageLatency  time.Duration `json:"averageLatency"`
	TasksPerSecond  float64       `json:"tasksPerSecond"`
	LastUpdated     time.Time     `json:"lastUpdated"`
}

// NewParallelProcessingEngine creates a new parallel processing engine
func NewParallelProcessingEngine(config *ProcessingEngineConfig, logger logr.Logger) *ParallelProcessingEngine {
	if config == nil {
		config = getDefaultProcessingEngineConfig()
	}

	engine := &ParallelProcessingEngine{
		config:          config,
		logger:          logger,
		intentQueue:     NewPriorityQueue(config.MaxQueueSize),
		dependencyGraph: NewDependencyGraph(),
		metrics:         NewProcessingMetrics(),
		stopChan:        make(chan struct{}),
		healthTicker:    time.NewTicker(config.HealthCheckInterval),
	}

	// Initialize worker pools
	engine.initializeWorkerPools()

	// Initialize management components
	engine.taskScheduler = NewTaskScheduler(engine, logger)
	engine.loadBalancer = NewLoadBalancer(logger)
	engine.resourceLimiter = NewResourceLimiter(config.MaxMemoryPerWorker, config.MaxCPUPerWorker, logger)

	if config.BackpressureEnabled {
		engine.backpressureManager = NewBackpressureManager(&BackpressureConfig{
			Enabled:          true,
			LoadThreshold:    config.BackpressureThreshold,
			EvaluationWindow: 30 * time.Second,
		}, logger)
	}

	return engine
}

// initializeWorkerPools initializes all worker pools
func (pe *ParallelProcessingEngine) initializeWorkerPools() {
	// Intent processing pool
	pe.intentPool = NewWorkerPool("intent", pe.config.IntentWorkers,
		NewIntentProcessor(pe.logger), pe.logger)

	// LLM processing pool
	pe.llmPool = NewWorkerPool("llm", pe.config.LLMWorkers,
		NewLLMProcessor(pe.logger), pe.logger)

	// RAG processing pool
	pe.ragPool = NewWorkerPool("rag", pe.config.RAGWorkers,
		NewRAGProcessor(pe.logger), pe.logger)

	// Resource planning pool
	pe.resourcePool = NewWorkerPool("resource", pe.config.ResourceWorkers,
		NewResourceProcessor(pe.logger), pe.logger)

	// Manifest generation pool
	pe.manifestPool = NewWorkerPool("manifest", pe.config.ManifestWorkers,
		NewManifestProcessor(pe.logger), pe.logger)

	// GitOps pool
	pe.gitopsPool = NewWorkerPool("gitops", pe.config.GitOpsWorkers,
		NewGitOpsProcessor(pe.logger), pe.logger)

	// Deployment verification pool
	pe.deploymentPool = NewWorkerPool("deployment", pe.config.DeploymentWorkers,
		NewDeploymentProcessor(pe.logger), pe.logger)
}

// Start starts the parallel processing engine
func (pe *ParallelProcessingEngine) Start(ctx context.Context) error {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	if pe.started {
		return fmt.Errorf("parallel processing engine already started")
	}

	pe.logger.Info("Starting parallel processing engine")

	// Start worker pools
	pools := []*WorkerPool{
		pe.intentPool, pe.llmPool, pe.ragPool,
		pe.resourcePool, pe.manifestPool,
		pe.gitopsPool, pe.deploymentPool,
	}

	for _, pool := range pools {
		if err := pool.Start(ctx); err != nil {
			return fmt.Errorf("failed to start pool %s: %w", pool.name, err)
		}
	}

	// Start task scheduler
	go pe.taskScheduler.Start(ctx)

	// Start health monitoring
	go pe.healthMonitor(ctx)

	// Start metrics collector
	if pe.config.MetricsEnabled {
		go pe.metricsCollector(ctx)
	}

	// Start backpressure manager if enabled
	if pe.backpressureManager != nil {
		go pe.backpressureManager.Start(ctx)
	}

	pe.started = true
	pe.logger.Info("Parallel processing engine started successfully")

	return nil
}

// Stop stops the parallel processing engine
func (pe *ParallelProcessingEngine) Stop() error {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	if !pe.started {
		return nil
	}

	pe.logger.Info("Stopping parallel processing engine")

	// Signal stop to all components
	close(pe.stopChan)
	pe.healthTicker.Stop()

	// Stop worker pools
	pools := []*WorkerPool{
		pe.intentPool, pe.llmPool, pe.ragPool,
		pe.resourcePool, pe.manifestPool,
		pe.gitopsPool, pe.deploymentPool,
	}

	for _, pool := range pools {
		pool.Stop()
	}

	// Stop task scheduler
	pe.taskScheduler.Stop()

	// Stop backpressure manager
	if pe.backpressureManager != nil {
		pe.backpressureManager.Stop()
	}

	pe.started = false
	pe.logger.Info("Parallel processing engine stopped")

	return nil
}

// ProcessIntent processes an intent using parallel processing
func (pe *ParallelProcessingEngine) ProcessIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) error {
	if !pe.started {
		return fmt.Errorf("parallel processing engine not started")
	}

	// Check backpressure
	if pe.backpressureManager != nil && pe.backpressureManager.ShouldReject() {
		return errors.NewProcessingError("ENGINE_OVERLOAD",
			"System under high load, intent processing rejected",
			"processing_engine", "process_intent",
			errors.CategoryCapacity, errors.SeverityHigh)
	}

	// Create processing state
	state := &ProcessingState{
		IntentID:       intent.Name,
		CurrentPhase:   interfaces.PhaseIntentReceived,
		Progress:       0.0,
		ActiveTasks:    make(map[string]*Task),
		CompletedTasks: make(map[string]*TaskResult),
		FailedTasks:    make(map[string]*TaskResult),
		StartedAt:      time.Now(),
		LastUpdated:    time.Now(),
		BlockedTasks:   make([]string, 0),
		ReadyTasks:     make([]string, 0),
	}

	pe.processingStates.Store(intent.Name, state)

	// Create processing pipeline tasks
	tasks := pe.createProcessingPipeline(ctx, intent)

	// Submit tasks for scheduling
	for _, task := range tasks {
		if err := pe.SubmitTask(task); err != nil {
			pe.logger.Error(err, "Failed to submit task",
				"taskId", task.ID, "taskType", task.Type)
			return err
		}
	}

	pe.logger.Info("Intent processing started",
		"intentId", intent.Name,
		"totalTasks", len(tasks))

	return nil
}

// SubmitTask submits a task for processing
func (pe *ParallelProcessingEngine) SubmitTask(task *Task) error {
	// Add to priority queue
	if err := pe.intentQueue.Enqueue(task); err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	// Update processing state
	if state, exists := pe.processingStates.Load(task.IntentID); exists {
		s := state.(*ProcessingState)
		s.mutex.Lock()
		s.ActiveTasks[task.ID] = task
		s.LastUpdated = time.Now()
		s.mutex.Unlock()
	}

	// Signal task scheduler
	select {
	case pe.taskScheduler.schedulingQueue <- task:
		// Task queued for scheduling
	default:
		return fmt.Errorf("task scheduler queue full")
	}

	return nil
}

// createProcessingPipeline creates the processing pipeline for an intent
func (pe *ParallelProcessingEngine) createProcessingPipeline(ctx context.Context, intent *nephoranv1.NetworkIntent) []*Task {
	tasks := make([]*Task, 0)

	// Task 1: LLM Processing
	llmTask := &Task{
		ID:         fmt.Sprintf("%s-llm", intent.Name),
		Type:       TaskTypeLLMProcessing,
		Priority:   1,
		IntentID:   intent.Name,
		Phase:      interfaces.PhaseLLMProcessing,
		Intent:     intent,
		InputData:  map[string]interface{}{"intent": intent.Spec.Intent},
		Context:    ctx,
		Timeout:    pe.config.ProcessingTimeout,
		MaxRetries: pe.config.MaxRetries,
		CreatedAt:  time.Now(),
	}
	tasks = append(tasks, llmTask)

	// Task 2: RAG Retrieval (parallel with LLM)
	ragTask := &Task{
		ID:         fmt.Sprintf("%s-rag", intent.Name),
		Type:       TaskTypeRAGRetrieval,
		Priority:   1,
		IntentID:   intent.Name,
		Phase:      interfaces.PhaseLLMProcessing,
		Intent:     intent,
		InputData:  map[string]interface{}{"query": intent.Spec.Intent},
		Context:    ctx,
		Timeout:    pe.config.ProcessingTimeout,
		MaxRetries: pe.config.MaxRetries,
		CreatedAt:  time.Now(),
	}
	tasks = append(tasks, ragTask)

	// Task 3: Resource Planning (depends on LLM and RAG)
	resourceTask := &Task{
		ID:           fmt.Sprintf("%s-resource", intent.Name),
		Type:         TaskTypeResourcePlanning,
		Priority:     2,
		IntentID:     intent.Name,
		Phase:        interfaces.PhaseResourcePlanning,
		Intent:       intent,
		InputData:    map[string]interface{}{},
		Context:      ctx,
		Dependencies: []string{llmTask.ID, ragTask.ID},
		Timeout:      pe.config.ProcessingTimeout,
		MaxRetries:   pe.config.MaxRetries,
		CreatedAt:    time.Now(),
	}
	tasks = append(tasks, resourceTask)

	// Task 4: Manifest Generation (depends on resource planning)
	manifestTask := &Task{
		ID:           fmt.Sprintf("%s-manifest", intent.Name),
		Type:         TaskTypeManifestGeneration,
		Priority:     3,
		IntentID:     intent.Name,
		Phase:        interfaces.PhaseManifestGeneration,
		Intent:       intent,
		InputData:    map[string]interface{}{},
		Context:      ctx,
		Dependencies: []string{resourceTask.ID},
		Timeout:      pe.config.ProcessingTimeout,
		MaxRetries:   pe.config.MaxRetries,
		CreatedAt:    time.Now(),
	}
	tasks = append(tasks, manifestTask)

	// Task 5: GitOps Commit (depends on manifest generation)
	gitopsTask := &Task{
		ID:           fmt.Sprintf("%s-gitops", intent.Name),
		Type:         TaskTypeGitOpsCommit,
		Priority:     4,
		IntentID:     intent.Name,
		Phase:        interfaces.PhaseGitOpsCommit,
		Intent:       intent,
		InputData:    map[string]interface{}{},
		Context:      ctx,
		Dependencies: []string{manifestTask.ID},
		Timeout:      pe.config.ProcessingTimeout,
		MaxRetries:   pe.config.MaxRetries,
		CreatedAt:    time.Now(),
	}
	tasks = append(tasks, gitopsTask)

	// Task 6: Deployment Verification (depends on GitOps)
	deploymentTask := &Task{
		ID:           fmt.Sprintf("%s-deployment", intent.Name),
		Type:         TaskTypeDeploymentVerify,
		Priority:     5,
		IntentID:     intent.Name,
		Phase:        interfaces.PhaseDeploymentVerification,
		Intent:       intent,
		InputData:    map[string]interface{}{},
		Context:      ctx,
		Dependencies: []string{gitopsTask.ID},
		Timeout:      pe.config.ProcessingTimeout,
		MaxRetries:   pe.config.MaxRetries,
		CreatedAt:    time.Now(),
	}
	tasks = append(tasks, deploymentTask)

	// Build dependency graph
	pe.dependencyGraph.AddTask(llmTask.ID, []string{}, []string{resourceTask.ID})
	pe.dependencyGraph.AddTask(ragTask.ID, []string{}, []string{resourceTask.ID})
	pe.dependencyGraph.AddTask(resourceTask.ID, []string{llmTask.ID, ragTask.ID}, []string{manifestTask.ID})
	pe.dependencyGraph.AddTask(manifestTask.ID, []string{resourceTask.ID}, []string{gitopsTask.ID})
	pe.dependencyGraph.AddTask(gitopsTask.ID, []string{manifestTask.ID}, []string{deploymentTask.ID})
	pe.dependencyGraph.AddTask(deploymentTask.ID, []string{gitopsTask.ID}, []string{})

	return tasks
}

// GetProcessingStatus returns the current processing status
func (pe *ParallelProcessingEngine) GetProcessingStatus(intentID string) (*ProcessingState, bool) {
	if state, exists := pe.processingStates.Load(intentID); exists {
		return state.(*ProcessingState), true
	}
	return nil, false
}

// GetMetrics returns processing metrics
func (pe *ParallelProcessingEngine) GetMetrics() *ProcessingMetrics {
	pe.metrics.mutex.RLock()
	defer pe.metrics.mutex.RUnlock()

	// Create a deep copy to avoid concurrent access issues
	metricsCopy := *pe.metrics
	metricsCopy.PoolMetrics = make(map[string]*PoolMetrics)
	metricsCopy.TaskMetrics = make(map[TaskType]*TaskTypeMetrics)
	metricsCopy.ErrorsByType = make(map[string]int64)

	for k, v := range pe.metrics.PoolMetrics {
		metricsCopy.PoolMetrics[k] = v
	}

	for k, v := range pe.metrics.TaskMetrics {
		metricsCopy.TaskMetrics[k] = v
	}

	for k, v := range pe.metrics.ErrorsByType {
		metricsCopy.ErrorsByType[k] = v
	}

	return &metricsCopy
}

// healthMonitor performs periodic health monitoring
func (pe *ParallelProcessingEngine) healthMonitor(ctx context.Context) {
	for {
		select {
		case <-pe.healthTicker.C:
			pe.performHealthCheck()

		case <-pe.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// performHealthCheck performs comprehensive health checking
func (pe *ParallelProcessingEngine) performHealthCheck() {
	pe.logger.Info("Performing health check")

	// Check worker pools
	pools := map[string]*WorkerPool{
		"intent":     pe.intentPool,
		"llm":        pe.llmPool,
		"rag":        pe.ragPool,
		"resource":   pe.resourcePool,
		"manifest":   pe.manifestPool,
		"gitops":     pe.gitopsPool,
		"deployment": pe.deploymentPool,
	}

	for name, pool := range pools {
		health := pool.GetHealth()
		if !health["healthy"].(bool) {
			pe.logger.Warn("Pool unhealthy", "pool", name, "metrics", health)
		}
	}

	// Check resource usage
	memUsage := pe.getCurrentMemoryUsage()
	cpuUsage := pe.getCurrentCPUUsage()

	pe.metrics.mutex.Lock()
	pe.metrics.TotalMemoryUsage = memUsage
	pe.metrics.TotalCPUUsage = cpuUsage
	pe.metrics.LastUpdated = time.Now()
	pe.metrics.mutex.Unlock()

	if memUsage > pe.config.MaxMemoryPerWorker*int64(len(pools)) {
		pe.logger.Warn("High memory usage detected", "usage", memUsage)
	}

	if cpuUsage > pe.config.MaxCPUPerWorker*float64(len(pools)) {
		pe.logger.Warn("High CPU usage detected", "usage", cpuUsage)
	}
}

// metricsCollector collects and updates metrics
func (pe *ParallelProcessingEngine) metricsCollector(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pe.collectMetrics()

		case <-pe.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// collectMetrics collects comprehensive metrics
func (pe *ParallelProcessingEngine) collectMetrics() {
	pe.metrics.mutex.Lock()
	defer pe.metrics.mutex.Unlock()

	// Collect pool metrics
	pools := map[string]*WorkerPool{
		"intent":     pe.intentPool,
		"llm":        pe.llmPool,
		"rag":        pe.ragPool,
		"resource":   pe.resourcePool,
		"manifest":   pe.manifestPool,
		"gitops":     pe.gitopsPool,
		"deployment": pe.deploymentPool,
	}

	for name, pool := range pools {
		metrics := pool.GetMetrics()
		pe.metrics.PoolMetrics[name] = &PoolMetrics{
			ActiveWorkers:  atomic.LoadInt32(&pool.activeWorkers),
			QueueLength:    len(pool.taskQueue),
			AverageLatency: pool.averageLatency,
			SuccessRate:    calculateSuccessRate(pool.processedTasks, pool.failedTasks),
			MemoryUsage:    atomic.LoadInt64(&pool.memoryUsage),
			CPUUsage:       pool.cpuUsage,
			LastUpdated:    time.Now(),
		}
	}

	// Calculate overall throughput
	pe.calculateThroughputMetrics()

	pe.metrics.LastUpdated = time.Now()
}

// calculateThroughputMetrics calculates throughput and latency metrics
func (pe *ParallelProcessingEngine) calculateThroughputMetrics() {
	// This would implement sophisticated throughput calculation
	// For now, using placeholder logic
	totalProcessed := atomic.LoadInt64(&pe.metrics.TotalIntentsProcessed)
	successful := atomic.LoadInt64(&pe.metrics.SuccessfulIntents)
	failed := atomic.LoadInt64(&pe.metrics.FailedIntents)

	if totalProcessed > 0 {
		pe.metrics.IntentsPerSecond = float64(successful+failed) / time.Since(pe.metrics.LastUpdated).Seconds()
	}
}

// getCurrentMemoryUsage returns current memory usage
func (pe *ParallelProcessingEngine) getCurrentMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc)
}

// getCurrentCPUUsage returns current CPU usage (simplified)
func (pe *ParallelProcessingEngine) getCurrentCPUUsage() float64 {
	// This would implement actual CPU usage calculation
	// For now, returning a placeholder
	return 0.15 // 15%
}

// calculateSuccessRate calculates success rate from processed and failed tasks
func calculateSuccessRate(processed, failed int64) float64 {
	if processed == 0 {
		return 1.0
	}
	return float64(processed-failed) / float64(processed)
}

// Helper functions and supporting implementations

// getDefaultProcessingEngineConfig returns default configuration
func getDefaultProcessingEngineConfig() *ProcessingEngineConfig {
	return &ProcessingEngineConfig{
		IntentWorkers:         5,
		LLMWorkers:            3,
		RAGWorkers:            3,
		ResourceWorkers:       4,
		ManifestWorkers:       4,
		GitOpsWorkers:         2,
		DeploymentWorkers:     3,
		MaxQueueSize:          1000,
		QueueTimeout:          5 * time.Minute,
		MaxConcurrentIntents:  100,
		ProcessingTimeout:     10 * time.Minute,
		HealthCheckInterval:   30 * time.Second,
		MaxMemoryPerWorker:    512 * 1024 * 1024, // 512MB
		MaxCPUPerWorker:       0.5,               // 50%
		BackpressureEnabled:   true,
		BackpressureThreshold: 0.8, // 80%
		LoadBalancingStrategy: "round_robin",
		MaxRetries:            3,
		CircuitBreakerEnabled: true,
		MetricsEnabled:        true,
		DetailedLogging:       false,
	}
}

// NewProcessingMetrics creates new processing metrics
func NewProcessingMetrics() *ProcessingMetrics {
	return &ProcessingMetrics{
		PoolMetrics:  make(map[string]*PoolMetrics),
		TaskMetrics:  make(map[TaskType]*TaskTypeMetrics),
		ErrorsByType: make(map[string]int64),
		LastUpdated:  time.Now(),
	}
}
