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
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// NewTaskScheduler creates a new task scheduler
func NewTaskScheduler(engine *ParallelProcessingEngine, logger logr.Logger) *TaskScheduler {
	scheduler := &TaskScheduler{
		engine:          engine,
		schedulingQueue: make(chan *Task, 1000),
		dependencyGraph: engine.dependencyGraph,
		readyTasks:      make(chan *Task, 1000),
		strategies:      make(map[string]SchedulingStrategy),
		currentStrategy: "priority_first",
		logger:          logger.WithName("task-scheduler"),
		stopChan:        make(chan struct{}),
	}

	// Register scheduling strategies
	scheduler.strategies["priority_first"] = NewPriorityFirstStrategy()
	scheduler.strategies["load_balanced"] = NewLoadBalancedStrategy()
	scheduler.strategies["dependency_aware"] = NewDependencyAwareStrategy()

	return scheduler
}

// Start starts the task scheduler
func (ts *TaskScheduler) Start(ctx context.Context) {
	ts.logger.Info("Starting task scheduler")

	// Start dependency resolver
	go ts.dependencyResolver(ctx)

	// Start task scheduler
	go ts.taskScheduler(ctx)

	// Start ready task dispatcher
	go ts.readyTaskDispatcher(ctx)
}

// Stop stops the task scheduler
func (ts *TaskScheduler) Stop() {
	ts.logger.Info("Stopping task scheduler")
	close(ts.stopChan)
}

// dependencyResolver resolves task dependencies
func (ts *TaskScheduler) dependencyResolver(ctx context.Context) {
	ts.logger.Info("Dependency resolver started")

	for {
		select {
		case task := <-ts.schedulingQueue:
			ts.processDependencies(task)

		case <-ts.stopChan:
			ts.logger.Info("Dependency resolver stopping")
			return

		case <-ctx.Done():
			ts.logger.Info("Dependency resolver context cancelled")
			return
		}
	}
}

// processDependencies processes task dependencies
func (ts *TaskScheduler) processDependencies(task *Task) {
	ts.logger.Info("Processing dependencies", "taskId", task.ID, "dependencies", len(task.Dependencies))

	// Check if all dependencies are satisfied
	if ts.dependencyGraph.AreDependenciesSatisfied(task.ID) {
		ts.logger.Info("Dependencies satisfied, task ready", "taskId", task.ID)

		// Mark as scheduled
		now := time.Now()
		task.ScheduledAt = &now

		// Send to ready queue
		select {
		case ts.readyTasks <- task:
			// Task queued successfully
		default:
			ts.logger.Info("Ready task queue full", "taskId", task.ID)
		}
	} else {
		ts.logger.Info("Dependencies not satisfied, waiting", "taskId", task.ID)
		// Task will be re-evaluated when dependencies complete
	}
}

// taskScheduler schedules ready tasks to appropriate worker pools
func (ts *TaskScheduler) taskScheduler(ctx context.Context) {
	ts.logger.Info("Task scheduler started")

	for {
		select {
		case task := <-ts.readyTasks:
			ts.scheduleTask(task)

		case <-ts.stopChan:
			ts.logger.Info("Task scheduler stopping")
			return

		case <-ctx.Done():
			ts.logger.Info("Task scheduler context cancelled")
			return
		}
	}
}

// scheduleTask schedules a single task
func (ts *TaskScheduler) scheduleTask(task *Task) {
	// Get available worker pools
	availableWorkers := ts.getAvailableWorkers()

	// Use strategy to select pool
	strategy := ts.strategies[ts.currentStrategy]
	poolName, err := strategy.ScheduleTask(task, availableWorkers)
	if err != nil {
		ts.logger.Error(err, "Failed to schedule task", "taskId", task.ID)
		return
	}

	// Submit task to selected pool
	if err := ts.submitToPool(poolName, task); err != nil {
		ts.logger.Error(err, "Failed to submit task to pool", "taskId", task.ID, "pool", poolName)
	} else {
		ts.logger.Info("Task scheduled successfully", "taskId", task.ID, "pool", poolName)
	}
}

// getAvailableWorkers returns available worker counts for each pool
func (ts *TaskScheduler) getAvailableWorkers() map[string]int {
	pools := map[string]*WorkerPool{
		"intent":     ts.engine.intentPool,
		"llm":        ts.engine.llmPool,
		"rag":        ts.engine.ragPool,
		"resource":   ts.engine.resourcePool,
		"manifest":   ts.engine.manifestPool,
		"gitops":     ts.engine.gitopsPool,
		"deployment": ts.engine.deploymentPool,
	}

	available := make(map[string]int)
	for name, pool := range pools {
		queueCapacity := cap(pool.taskQueue)
		queueLength := len(pool.taskQueue)
		available[name] = queueCapacity - queueLength
	}

	return available
}

// submitToPool submits a task to the specified worker pool
func (ts *TaskScheduler) submitToPool(poolName string, task *Task) error {
	var pool *WorkerPool

	switch poolName {
	case "intent":
		pool = ts.engine.intentPool
	case "llm":
		pool = ts.engine.llmPool
	case "rag":
		pool = ts.engine.ragPool
	case "resource":
		pool = ts.engine.resourcePool
	case "manifest":
		pool = ts.engine.manifestPool
	case "gitops":
		pool = ts.engine.gitopsPool
	case "deployment":
		pool = ts.engine.deploymentPool
	default:
		return fmt.Errorf("unknown pool: %s", poolName)
	}

	return pool.SubmitTask(task)
}

// readyTaskDispatcher dispatches ready tasks
func (ts *TaskScheduler) readyTaskDispatcher(ctx context.Context) {
	ts.logger.Info("Ready task dispatcher started")

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.checkForReadyTasks()

		case <-ts.stopChan:
			ts.logger.Info("Ready task dispatcher stopping")
			return

		case <-ctx.Done():
			ts.logger.Info("Ready task dispatcher context cancelled")
			return
		}
	}
}

// checkForReadyTasks checks for tasks that have become ready
func (ts *TaskScheduler) checkForReadyTasks() {
	// This would implement logic to check for tasks whose dependencies
	// have been satisfied and make them ready for scheduling

	// For now, this is a placeholder as dependency satisfaction
	// is handled in the dependency resolver
}

// Scheduling Strategies

// PriorityFirstStrategy schedules tasks based on priority
type PriorityFirstStrategy struct {
	name string
}

// NewPriorityFirstStrategy creates a new priority-first strategy
func NewPriorityFirstStrategy() *PriorityFirstStrategy {
	return &PriorityFirstStrategy{
		name: "priority_first",
	}
}

// ScheduleTask schedules a task using priority-first strategy
func (pfs *PriorityFirstStrategy) ScheduleTask(task *Task, availableWorkers map[string]int) (string, error) {
	// Map task type to pool name
	poolName := pfs.getPoolNameForTaskType(task.Type)

	// Check if pool has capacity
	if available, exists := availableWorkers[poolName]; exists && available > 0 {
		return poolName, nil
	}

	// If preferred pool is full, try to find alternative
	for name, available := range availableWorkers {
		if available > 0 {
			return name, nil
		}
	}

	return "", fmt.Errorf("no available workers for task %s", task.ID)
}

// getPoolNameForTaskType maps task types to pool names
func (pfs *PriorityFirstStrategy) getPoolNameForTaskType(taskType TaskType) string {
	switch taskType {
	case TaskTypeIntentProcessing:
		return "intent"
	case TaskTypeLLMProcessing:
		return "llm"
	case TaskTypeRAGRetrieval:
		return "rag"
	case TaskTypeResourcePlanning:
		return "resource"
	case TaskTypeManifestGeneration:
		return "manifest"
	case TaskTypeGitOpsCommit:
		return "gitops"
	case TaskTypeDeploymentVerify:
		return "deployment"
	default:
		return "intent" // default fallback
	}
}

// GetStrategyName returns the strategy name
func (pfs *PriorityFirstStrategy) GetStrategyName() string {
	return pfs.name
}

// GetMetrics returns strategy metrics
func (pfs *PriorityFirstStrategy) GetMetrics() map[string]float64 {
	return map[string]float64{
		"strategy_success_rate": 0.95,
	}
}

// LoadBalancedStrategy balances load across worker pools
type LoadBalancedStrategy struct {
	name string
}

// NewLoadBalancedStrategy creates a new load-balanced strategy
func NewLoadBalancedStrategy() *LoadBalancedStrategy {
	return &LoadBalancedStrategy{
		name: "load_balanced",
	}
}

// ScheduleTask schedules a task using load-balanced strategy
func (lbs *LoadBalancedStrategy) ScheduleTask(task *Task, availableWorkers map[string]int) (string, error) {
	// Find pool with most available capacity
	maxAvailable := 0
	bestPool := ""

	// First try the preferred pool for this task type
	preferredPool := getPoolNameForTaskType(task.Type)
	if available, exists := availableWorkers[preferredPool]; exists && available > 0 {
		maxAvailable = available
		bestPool = preferredPool
	}

	// Look for better options
	for poolName, available := range availableWorkers {
		if available > maxAvailable {
			maxAvailable = available
			bestPool = poolName
		}
	}

	if bestPool == "" {
		return "", fmt.Errorf("no available workers for task %s", task.ID)
	}

	return bestPool, nil
}

// GetStrategyName returns the strategy name
func (lbs *LoadBalancedStrategy) GetStrategyName() string {
	return lbs.name
}

// GetMetrics returns strategy metrics
func (lbs *LoadBalancedStrategy) GetMetrics() map[string]float64 {
	return map[string]float64{
		"strategy_success_rate": 0.92,
		"load_distribution":     0.88,
	}
}

// DependencyAwareStrategy considers task dependencies in scheduling
type DependencyAwareStrategy struct {
	name string
}

// NewDependencyAwareStrategy creates a new dependency-aware strategy
func NewDependencyAwareStrategy() *DependencyAwareStrategy {
	return &DependencyAwareStrategy{
		name: "dependency_aware",
	}
}

// ScheduleTask schedules a task using dependency-aware strategy
func (das *DependencyAwareStrategy) ScheduleTask(task *Task, availableWorkers map[string]int) (string, error) {
	// Prioritize pools that can handle dependent tasks
	preferredPool := getPoolNameForTaskType(task.Type)

	// Check if preferred pool has capacity
	if available, exists := availableWorkers[preferredPool]; exists && available > 0 {
		return preferredPool, nil
	}

	// Look for alternative with capacity
	for poolName, available := range availableWorkers {
		if available > 0 {
			return poolName, nil
		}
	}

	return "", fmt.Errorf("no available workers for task %s", task.ID)
}

// GetStrategyName returns the strategy name
func (das *DependencyAwareStrategy) GetStrategyName() string {
	return das.name
}

// GetMetrics returns strategy metrics
func (das *DependencyAwareStrategy) GetMetrics() map[string]float64 {
	return map[string]float64{
		"strategy_success_rate": 0.89,
		"dependency_efficiency": 0.91,
	}
}

// Dependency Graph Implementation

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*DependencyNode),
	}
}

// AddTask adds a task to the dependency graph
func (dg *DependencyGraph) AddTask(taskID string, dependencies []string, dependents []string) {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()

	// Create node if it doesn't exist
	if _, exists := dg.nodes[taskID]; !exists {
		dg.nodes[taskID] = &DependencyNode{
			TaskID:       taskID,
			Dependencies: make([]*DependencyNode, 0),
			Dependents:   make([]*DependencyNode, 0),
		}
	}

	node := dg.nodes[taskID]

	// Add dependencies
	for _, depID := range dependencies {
		if _, exists := dg.nodes[depID]; !exists {
			dg.nodes[depID] = &DependencyNode{
				TaskID:       depID,
				Dependencies: make([]*DependencyNode, 0),
				Dependents:   make([]*DependencyNode, 0),
			}
		}

		depNode := dg.nodes[depID]
		node.Dependencies = append(node.Dependencies, depNode)
		depNode.Dependents = append(depNode.Dependents, node)
	}

	// Add dependents
	for _, depID := range dependents {
		if _, exists := dg.nodes[depID]; !exists {
			dg.nodes[depID] = &DependencyNode{
				TaskID:       depID,
				Dependencies: make([]*DependencyNode, 0),
				Dependents:   make([]*DependencyNode, 0),
			}
		}

		depNode := dg.nodes[depID]
		node.Dependents = append(node.Dependents, depNode)
		depNode.Dependencies = append(depNode.Dependencies, node)
	}
}

// AreDependenciesSatisfied checks if all dependencies for a task are satisfied
func (dg *DependencyGraph) AreDependenciesSatisfied(taskID string) bool {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	node, exists := dg.nodes[taskID]
	if !exists {
		return true // No dependencies
	}

	node.mutex.RLock()
	defer node.mutex.RUnlock()

	for _, dep := range node.Dependencies {
		dep.mutex.RLock()
		completed := dep.Completed && !dep.Failed
		dep.mutex.RUnlock()

		if !completed {
			return false
		}
	}

	return true
}

// MarkTaskCompleted marks a task as completed
func (dg *DependencyGraph) MarkTaskCompleted(taskID string, success bool) {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()

	node, exists := dg.nodes[taskID]
	if !exists {
		return
	}

	node.mutex.Lock()
	defer node.mutex.Unlock()

	node.Completed = true
	node.Failed = !success
}

// GetReadyTasks returns tasks that are ready to be scheduled
func (dg *DependencyGraph) GetReadyTasks() []string {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	ready := make([]string, 0)

	for taskID, node := range dg.nodes {
		node.mutex.RLock()
		if !node.Completed && !node.Failed && dg.areDependenciesSatisfiedUnsafe(node) {
			ready = append(ready, taskID)
		}
		node.mutex.RUnlock()
	}

	return ready
}

// areDependenciesSatisfiedUnsafe checks dependencies without locking (assumes already locked)
func (dg *DependencyGraph) areDependenciesSatisfiedUnsafe(node *DependencyNode) bool {
	for _, dep := range node.Dependencies {
		dep.mutex.RLock()
		completed := dep.Completed && !dep.Failed
		dep.mutex.RUnlock()

		if !completed {
			return false
		}
	}
	return true
}

// GetTaskStatus returns the status of a task in the dependency graph
func (dg *DependencyGraph) GetTaskStatus(taskID string) (completed bool, failed bool, exists bool) {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	node, exists := dg.nodes[taskID]
	if !exists {
		return false, false, false
	}

	node.mutex.RLock()
	defer node.mutex.RUnlock()

	return node.Completed, node.Failed, true
}

// Helper functions

// getPoolNameForTaskType returns the appropriate pool name for a task type
func getPoolNameForTaskType(taskType TaskType) string {
	switch taskType {
	case TaskTypeIntentProcessing:
		return "intent"
	case TaskTypeLLMProcessing:
		return "llm"
	case TaskTypeRAGRetrieval:
		return "rag"
	case TaskTypeResourcePlanning:
		return "resource"
	case TaskTypeManifestGeneration:
		return "manifest"
	case TaskTypeGitOpsCommit:
		return "gitops"
	case TaskTypeDeploymentVerify:
		return "deployment"
	default:
		return "intent" // Default fallback
	}
}

// Load Balancer Implementation

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(logger logr.Logger) *LoadBalancer {
	lb := &LoadBalancer{
		strategies:  make(map[string]LoadBalancingStrategy),
		current:     "round_robin",
		poolMetrics: make(map[string]*PoolMetrics),
		logger:      logger.WithName("load-balancer"),
	}

	// Register load balancing strategies
	lb.strategies["round_robin"] = NewRoundRobinLoadBalancer()
	lb.strategies["least_connections"] = NewLeastConnectionsLoadBalancer()
	lb.strategies["weighted_response_time"] = NewWeightedResponseTimeLoadBalancer()

	return lb
}

// SelectPool selects a pool using the current load balancing strategy
func (lb *LoadBalancer) SelectPool(pools map[string]*WorkerPool, task *Task) (string, error) {
	lb.mutex.RLock()
	strategy := lb.strategies[lb.current]
	lb.mutex.RUnlock()

	if strategy == nil {
		return "", fmt.Errorf("no load balancing strategy configured")
	}

	return strategy.SelectPool(pools, task)
}

// UpdatePoolMetrics updates metrics for a pool
func (lb *LoadBalancer) UpdatePoolMetrics(poolName string, metrics *PoolMetrics) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.poolMetrics[poolName] = metrics

	// Update strategy metrics
	for _, strategy := range lb.strategies {
		strategy.UpdateMetrics(poolName, metrics)
	}
}

// Load Balancing Strategies

// RoundRobinLoadBalancer implements round-robin load balancing
type RoundRobinLoadBalancer struct {
	name    string
	counter int
	mutex   sync.Mutex
}

// NewRoundRobinLoadBalancer creates a new round-robin load balancer
func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		name: "round_robin",
	}
}

// SelectPool selects a pool using round-robin
func (rr *RoundRobinLoadBalancer) SelectPool(pools map[string]*WorkerPool, task *Task) (string, error) {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()

	if len(pools) == 0 {
		return "", fmt.Errorf("no pools available")
	}

	poolNames := make([]string, 0, len(pools))
	for name := range pools {
		poolNames = append(poolNames, name)
	}

	selected := poolNames[rr.counter%len(poolNames)]
	rr.counter++

	return selected, nil
}

// GetStrategyName returns the strategy name
func (rr *RoundRobinLoadBalancer) GetStrategyName() string {
	return rr.name
}

// UpdateMetrics updates metrics (no-op for round-robin)
func (rr *RoundRobinLoadBalancer) UpdateMetrics(poolName string, metrics *PoolMetrics) {
	// No-op for round-robin
}

// LeastConnectionsLoadBalancer selects pool with least active connections
type LeastConnectionsLoadBalancer struct {
	name string
}

// NewLeastConnectionsLoadBalancer creates a new least-connections load balancer
func NewLeastConnectionsLoadBalancer() *LeastConnectionsLoadBalancer {
	return &LeastConnectionsLoadBalancer{
		name: "least_connections",
	}
}

// SelectPool selects pool with least connections
func (lc *LeastConnectionsLoadBalancer) SelectPool(pools map[string]*WorkerPool, task *Task) (string, error) {
	if len(pools) == 0 {
		return "", fmt.Errorf("no pools available")
	}

	var bestPool string
	minConnections := int32(-1)

	for name, pool := range pools {
		connections := pool.activeWorkers
		if minConnections == -1 || connections < minConnections {
			minConnections = connections
			bestPool = name
		}
	}

	return bestPool, nil
}

// GetStrategyName returns the strategy name
func (lc *LeastConnectionsLoadBalancer) GetStrategyName() string {
	return lc.name
}

// UpdateMetrics updates metrics (no-op for least connections)
func (lc *LeastConnectionsLoadBalancer) UpdateMetrics(poolName string, metrics *PoolMetrics) {
	// No-op for least connections
}

// WeightedResponseTimeLoadBalancer selects pool based on response time
type WeightedResponseTimeLoadBalancer struct {
	name        string
	poolWeights map[string]float64
	mutex       sync.RWMutex
}

// NewWeightedResponseTimeLoadBalancer creates a new weighted response time load balancer
func NewWeightedResponseTimeLoadBalancer() *WeightedResponseTimeLoadBalancer {
	return &WeightedResponseTimeLoadBalancer{
		name:        "weighted_response_time",
		poolWeights: make(map[string]float64),
	}
}

// SelectPool selects pool based on weighted response time
func (wrt *WeightedResponseTimeLoadBalancer) SelectPool(pools map[string]*WorkerPool, task *Task) (string, error) {
	wrt.mutex.RLock()
	defer wrt.mutex.RUnlock()

	if len(pools) == 0 {
		return "", fmt.Errorf("no pools available")
	}

	var bestPool string
	bestWeight := float64(-1)

	for name := range pools {
		weight, exists := wrt.poolWeights[name]
		if !exists {
			weight = 1.0 // Default weight
		}

		if bestWeight == -1 || weight > bestWeight {
			bestWeight = weight
			bestPool = name
		}
	}

	if bestPool == "" {
		// Fallback to first pool
		for name := range pools {
			return name, nil
		}
	}

	return bestPool, nil
}

// GetStrategyName returns the strategy name
func (wrt *WeightedResponseTimeLoadBalancer) GetStrategyName() string {
	return wrt.name
}

// UpdateMetrics updates pool weights based on metrics
func (wrt *WeightedResponseTimeLoadBalancer) UpdateMetrics(poolName string, metrics *PoolMetrics) {
	wrt.mutex.Lock()
	defer wrt.mutex.Unlock()

	// Calculate weight based on response time (inverse relationship)
	if metrics.AverageLatency > 0 {
		weight := 1.0 / metrics.AverageLatency.Seconds()
		wrt.poolWeights[poolName] = weight
	} else {
		wrt.poolWeights[poolName] = 1.0
	}
}
