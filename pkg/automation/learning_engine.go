package automation

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// LearningEngine learns from remediation outcomes and improves strategies
type LearningEngine struct {
	mu              sync.RWMutex
	logger          *slog.Logger
	strategies      map[string]*RemediationStrategy
	outcomes        []*RemediationOutcome
	accuracyMetrics map[string]*AccuracyMetric
	modelUpdates    chan *ModelUpdate
}

// RemediationOutcome represents the outcome of a remediation action
type RemediationOutcome struct {
	SessionID    string                 `json:"session_id"`
	Component    string                 `json:"component"`
	Strategy     string                 `json:"strategy"`
	Success      bool                   `json:"success"`
	Duration     time.Duration          `json:"duration"`
	Timestamp    time.Time              `json:"timestamp"`
	Metrics      map[string]float64     `json:"metrics"`
	Context      map[string]interface{} `json:"context"`
	ErrorDetails string                 `json:"error_details,omitempty"`
}

// AccuracyMetric tracks the accuracy of prediction models
type AccuracyMetric struct {
	Component          string    `json:"component"`
	ModelType          string    `json:"model_type"`
	TotalPredictions   int       `json:"total_predictions"`
	CorrectPredictions int       `json:"correct_predictions"`
	Accuracy           float64   `json:"accuracy"`
	LastUpdated        time.Time `json:"last_updated"`
}

// ModelUpdate represents an update to a prediction model
type ModelUpdate struct {
	Component  string                 `json:"component"`
	ModelType  string                 `json:"model_type"`
	Parameters map[string]interface{} `json:"parameters"`
	Accuracy   float64                `json:"accuracy"`
	Timestamp  time.Time              `json:"timestamp"`
}

// RollbackManager manages rollback operations
type RollbackManager struct {
	mu        sync.RWMutex
	logger    *slog.Logger
	rollbacks map[string]*RollbackPlan
}

// RollbackPlan defines steps to rollback a failed remediation
type RollbackPlan struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Component   string                 `json:"component"`
	Steps       []RollbackStep         `json:"steps"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	ExecutedAt  *time.Time             `json:"executed_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RollbackStep represents a single step in a rollback plan
type RollbackStep struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // RESTORE, SCALE, RESTART, etc.
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Status      string                 `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED
	StartTime   *time.Time             `json:"start_time,omitempty"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// NewLearningEngine creates a new learning engine
func NewLearningEngine(logger *slog.Logger) *LearningEngine {
	return &LearningEngine{
		logger:          logger,
		strategies:      make(map[string]*RemediationStrategy),
		outcomes:        make([]*RemediationOutcome, 0),
		accuracyMetrics: make(map[string]*AccuracyMetric),
		modelUpdates:    make(chan *ModelUpdate, 100),
	}
}

// RecordRemediation records the outcome of a remediation action
func (le *LearningEngine) RecordRemediation(session *RemediationSession, strategy *RemediationStrategy, success bool) {
	le.mu.Lock()
	defer le.mu.Unlock()

	outcome := &RemediationOutcome{
		SessionID: session.ID,
		Component: session.Component,
		Strategy:  session.Strategy,
		Success:   success,
		Duration:  time.Since(session.StartTime),
		Timestamp: time.Now(),
		Metrics:   make(map[string]float64),
		Context:   session.Results,
	}

	if !success && len(session.Actions) > 0 {
		// Find the first failed action for error details
		for _, action := range session.Actions {
			if action.Status == "FAILED" && action.Error != "" {
				outcome.ErrorDetails = action.Error
				break
			}
		}
	}

	le.outcomes = append(le.outcomes, outcome)

	// Update strategy success rate
	if strategy != nil {
		strategy.Total++
		if success {
			strategy.Success++
		}
		strategy.SuccessRate = float64(strategy.Success) / float64(strategy.Total)
		le.strategies[strategy.Name] = strategy
	}

	le.logger.Info("Recorded remediation outcome",
		"session_id", outcome.SessionID,
		"component", outcome.Component,
		"strategy", outcome.Strategy,
		"success", outcome.Success,
		"duration", outcome.Duration)
}

// UpdateModelAccuracy updates the accuracy metrics for a prediction model
func (le *LearningEngine) UpdateModelAccuracy(component, modelType string, correct bool) {
	le.mu.Lock()
	defer le.mu.Unlock()

	key := component + ":" + modelType
	metric := le.accuracyMetrics[key]
	if metric == nil {
		metric = &AccuracyMetric{
			Component: component,
			ModelType: modelType,
		}
		le.accuracyMetrics[key] = metric
	}

	metric.TotalPredictions++
	if correct {
		metric.CorrectPredictions++
	}
	metric.Accuracy = float64(metric.CorrectPredictions) / float64(metric.TotalPredictions)
	metric.LastUpdated = time.Now()
}

// GetStrategySuccessRate returns the success rate for a strategy
func (le *LearningEngine) GetStrategySuccessRate(strategyName string) float64 {
	le.mu.RLock()
	defer le.mu.RUnlock()

	if strategy, exists := le.strategies[strategyName]; exists {
		return strategy.SuccessRate
	}
	return 0.0
}

// GetModelAccuracy returns the accuracy for a prediction model
func (le *LearningEngine) GetModelAccuracy(component, modelType string) float64 {
	le.mu.RLock()
	defer le.mu.RUnlock()

	key := component + ":" + modelType
	if metric, exists := le.accuracyMetrics[key]; exists {
		return metric.Accuracy
	}
	return 0.0
}

// NewRollbackManager creates a new rollback manager
func NewRollbackManager(logger *slog.Logger) *RollbackManager {
	return &RollbackManager{
		logger:    logger,
		rollbacks: make(map[string]*RollbackPlan),
	}
}

// CreateRollbackPlan creates a rollback plan for a failed remediation
func (rm *RollbackManager) CreateRollbackPlan(session *RemediationSession) *RollbackPlan {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	plan := &RollbackPlan{
		ID:        "rollback-" + session.ID,
		SessionID: session.ID,
		Component: session.Component,
		Status:    "CREATED",
		CreatedAt: time.Now(),
		Steps:     make([]RollbackStep, 0),
		Metadata:  make(map[string]interface{}),
	}

	// Generate rollback steps based on the failed actions
	for i := len(session.Actions) - 1; i >= 0; i-- {
		action := session.Actions[i]
		if action.Status == "COMPLETED" {
			step := rm.createRollbackStep(action)
			plan.Steps = append(plan.Steps, step)
		}
	}

	rm.rollbacks[plan.ID] = plan
	return plan
}

// createRollbackStep creates a rollback step for a completed action
func (rm *RollbackManager) createRollbackStep(action *RemediationAction) RollbackStep {
	step := RollbackStep{
		ID:         "step-" + action.Type + "-rollback",
		Status:     "PENDING",
		Parameters: make(map[string]interface{}),
	}

	switch action.Type {
	case "SCALE":
		step.Type = "RESTORE_SCALE"
		step.Description = "Restore original scaling configuration"
		step.Parameters["target"] = action.Target
	case "RESTART":
		step.Type = "RESTORE_STATE"
		step.Description = "Restore service to previous state"
		step.Parameters["target"] = action.Target
	case "REDEPLOY":
		step.Type = "RESTORE_DEPLOYMENT"
		step.Description = "Restore previous deployment version"
		step.Parameters["target"] = action.Target
	default:
		step.Type = "MANUAL_INTERVENTION"
		step.Description = "Manual intervention required for rollback"
	}

	return step
}

// ExecuteRollback executes a rollback plan
func (rm *RollbackManager) ExecuteRollback(planID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	plan, exists := rm.rollbacks[planID]
	if !exists {
		return fmt.Errorf("rollback plan %s not found", planID)
	}

	if plan.Status != "CREATED" {
		return fmt.Errorf("rollback plan %s is not in CREATED status", planID)
	}

	plan.Status = "EXECUTING"
	now := time.Now()
	plan.ExecutedAt = &now

	rm.logger.Info("Executing rollback plan", "plan_id", planID, "component", plan.Component)

	// Execute each step
	for i := range plan.Steps {
		step := &plan.Steps[i]
		rm.executeRollbackStep(step)
	}

	plan.Status = "COMPLETED"
	completedAt := time.Now()
	plan.CompletedAt = &completedAt

	return nil
}

// executeRollbackStep executes a single rollback step
func (rm *RollbackManager) executeRollbackStep(step *RollbackStep) {
	step.Status = "RUNNING"
	now := time.Now()
	step.StartTime = &now

	rm.logger.Info("Executing rollback step", "step_id", step.ID, "type", step.Type)

	// In a real implementation, this would perform the actual rollback operation
	// For now, simulate the execution
	time.Sleep(100 * time.Millisecond)

	step.Status = "COMPLETED"
	endTime := time.Now()
	step.EndTime = &endTime
}
