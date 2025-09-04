package automation

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// LearningEngine learns from remediation outcomes and improves strategies.
// Enhanced with 2025 AI/ML capabilities for intelligent automation.
type LearningEngine struct {
	mu sync.RWMutex

	logger *slog.Logger

	strategies map[string]*RemediationStrategy

	outcomes []*RemediationOutcome

	accuracyMetrics map[string]*AccuracyMetric

	modelUpdates chan *ModelUpdate

	// 2025 AI enhancements
	aiFeatures map[string]bool
	predictionCache map[string]*CachedPrediction
	cacheTTL time.Duration
}

// CachedPrediction represents a cached ML prediction
type CachedPrediction struct {
	Prediction float64 `json:"prediction"`
	Timestamp  time.Time `json:"timestamp"`
	Features   map[string]float64 `json:"features"`
	Confidence float64 `json:"confidence"`
}

// RemediationOutcome represents the outcome of a remediation action.

type RemediationOutcome struct {
	SessionID string `json:"session_id"`

	Component string `json:"component"`

	Strategy string `json:"strategy"`

	Success bool `json:"success"`

	Duration time.Duration `json:"duration"`

	Timestamp time.Time `json:"timestamp"`

	Metrics map[string]float64 `json:"metrics"`

	Context json.RawMessage `json:"context"`

	ErrorDetails string `json:"error_details,omitempty"`
}

// AccuracyMetric tracks the accuracy of prediction models.

type AccuracyMetric struct {
	Component string `json:"component"`

	ModelType string `json:"model_type"`

	TotalPredictions int `json:"total_predictions"`

	CorrectPredictions int `json:"correct_predictions"`

	Accuracy float64 `json:"accuracy"`

	LastUpdated time.Time `json:"last_updated"`
}

// ModelUpdate represents an update to a prediction model.

type ModelUpdate struct {
	Component string `json:"component"`

	ModelType string `json:"model_type"`

	Parameters json.RawMessage `json:"parameters"`

	Accuracy float64 `json:"accuracy"`

	Timestamp time.Time `json:"timestamp"`
}

// RollbackManager manages rollback operations.

type RollbackManager struct {
	mu sync.RWMutex

	logger *slog.Logger

	rollbacks map[string]*RollbackPlan
}

// RollbackPlan defines steps to rollback a failed remediation.

type RollbackPlan struct {
	ID string `json:"id"`

	SessionID string `json:"session_id"`

	Component string `json:"component"`

	Steps []RollbackStep `json:"steps"`

	Status string `json:"status"`

	CreatedAt time.Time `json:"created_at"`

	ExecutedAt *time.Time `json:"executed_at,omitempty"`

	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Metadata json.RawMessage `json:"metadata"`
}

// RollbackStep represents a single step in a rollback plan.

type RollbackStep struct {
	ID string `json:"id"`

	Type string `json:"type"` // RESTORE, SCALE, RESTART, etc.

	Description string `json:"description"`

	Parameters json.RawMessage `json:"parameters"`

	Status string `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED

	StartTime *time.Time `json:"start_time,omitempty"`

	EndTime *time.Time `json:"end_time,omitempty"`

	Error string `json:"error,omitempty"`
}

// NewLearningEngine creates a new learning engine.

func NewLearningEngine(logger *slog.Logger) *LearningEngine {
	return &LearningEngine{
		logger: logger,

		strategies: make(map[string]*RemediationStrategy),

		outcomes: make([]*RemediationOutcome, 0),

		accuracyMetrics: make(map[string]*AccuracyMetric),

		modelUpdates: make(chan *ModelUpdate, 100),

		// 2025 AI enhancements
		aiFeatures: map[string]bool{
			FeatureAIPoweredRemediation: true,
			FeatureAnomalyDetection:     true,
			FeaturePredictiveScaling:    true,
			FeatureSmartAlerts:          true,
		},
		predictionCache: make(map[string]*CachedPrediction),
		cacheTTL:        5 * time.Minute,
	}
}

// RecordRemediation records the outcome of a remediation action.

func (le *LearningEngine) RecordRemediation(session *RemediationSession, strategy *RemediationStrategy, success bool) {
	le.mu.Lock()

	defer le.mu.Unlock()

	outcome := &RemediationOutcome{
		SessionID: session.ID,

		Component: session.Component,

		Strategy: session.Strategy,

		Success: success,

		Duration: time.Since(session.StartTime),

		Timestamp: time.Now(),

		Metrics: make(map[string]float64),

		Context: session.Results,
	}

	if !success && len(session.Actions) > 0 {
		// Find the first failed action for error details.

		for _, action := range session.Actions {
			if action.Status == "FAILED" && action.Error != "" {

				outcome.ErrorDetails = action.Error

				break

			}
		}
	}

	le.outcomes = append(le.outcomes, outcome)

	// Update strategy success rate.

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

// UpdateModelAccuracy updates the accuracy metrics for a prediction model.

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

// GetStrategySuccessRate returns the success rate for a strategy.

func (le *LearningEngine) GetStrategySuccessRate(strategyName string) float64 {
	le.mu.RLock()

	defer le.mu.RUnlock()

	if strategy, exists := le.strategies[strategyName]; exists {
		return strategy.SuccessRate
	}

	return 0.0
}

// GetModelAccuracy returns the accuracy for a prediction model.

func (le *LearningEngine) GetModelAccuracy(component, modelType string) float64 {
	le.mu.RLock()

	defer le.mu.RUnlock()

	key := component + ":" + modelType

	if metric, exists := le.accuracyMetrics[key]; exists {
		return metric.Accuracy
	}

	return 0.0
}

// NewRollbackManager creates a new rollback manager.

func NewRollbackManager(logger *slog.Logger) *RollbackManager {
	return &RollbackManager{
		logger: logger,

		rollbacks: make(map[string]*RollbackPlan),
	}
}

// CreateRollbackPlan creates a rollback plan for a failed remediation.

func (rm *RollbackManager) CreateRollbackPlan(session *RemediationSession) *RollbackPlan {
	rm.mu.Lock()

	defer rm.mu.Unlock()

	plan := &RollbackPlan{
		ID: "rollback-" + session.ID,

		SessionID: session.ID,

		Component: session.Component,

		Status: "CREATED",

		CreatedAt: time.Now(),

		Steps: make([]RollbackStep, 0),

		Metadata: json.RawMessage(`{}`),
	}

	// Generate rollback steps based on the failed actions.

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

// createRollbackStep creates a rollback step for a completed action.

func (rm *RollbackManager) createRollbackStep(action *RemediationAction) RollbackStep {
	step := RollbackStep{
		ID:     "step-" + action.Type + "-rollback",
		Status: "PENDING",
	}

	// Build parameters map
	params := map[string]interface{}{
		"target": action.Target,
	}

	switch action.Type {
	case "SCALE":
		step.Type = "RESTORE_SCALE"
		step.Description = "Restore original scaling configuration"

	case "RESTART":
		step.Type = "RESTORE_STATE"
		step.Description = "Restore service to previous state"

	case "REDEPLOY":
		step.Type = "RESTORE_DEPLOYMENT"
		step.Description = "Restore previous deployment version"

	default:

		step.Type = "MANUAL_INTERVENTION"

		step.Description = "Manual intervention required for rollback"

	}

	// Marshal parameters to JSON
	if paramBytes, err := json.Marshal(params); err == nil {
		step.Parameters = paramBytes
	} else {
		step.Parameters = json.RawMessage(`{}`)
	}

	return step
}

// ExecuteRollback executes a rollback plan.

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

	// Execute each step.

	for i := range plan.Steps {

		step := &plan.Steps[i]

		rm.executeRollbackStep(step)

	}

	plan.Status = "COMPLETED"

	completedAt := time.Now()

	plan.CompletedAt = &completedAt

	return nil
}

// executeRollbackStep executes a single rollback step.

func (rm *RollbackManager) executeRollbackStep(step *RollbackStep) {
	step.Status = "RUNNING"

	now := time.Now()

	step.StartTime = &now

	rm.logger.Info("Executing rollback step", "step_id", step.ID, "type", step.Type)

	// In a real implementation, this would perform the actual rollback operation.

	// For now, simulate the execution.

	time.Sleep(100 * time.Millisecond)

	step.Status = "COMPLETED"

	endTime := time.Now()

	step.EndTime = &endTime
}

// 2025 Modern Automation Methods

// EnableAIFeature enables a specific AI feature
func (le *LearningEngine) EnableAIFeature(feature string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.aiFeatures[feature] = true
	le.logger.Info("AI feature enabled", "feature", feature)
}

// DisableAIFeature disables a specific AI feature
func (le *LearningEngine) DisableAIFeature(feature string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.aiFeatures[feature] = false
	le.logger.Info("AI feature disabled", "feature", feature)
}

// IsAIFeatureEnabled checks if an AI feature is enabled
func (le *LearningEngine) IsAIFeatureEnabled(feature string) bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.aiFeatures[feature]
}

// GetCachedPrediction retrieves a cached prediction if valid
func (le *LearningEngine) GetCachedPrediction(key string) (*CachedPrediction, bool) {
	le.mu.RLock()
	defer le.mu.RUnlock()
	
	if cached, exists := le.predictionCache[key]; exists {
		if time.Since(cached.Timestamp) < le.cacheTTL {
			return cached, true
		}
		// Cache expired, remove it
		delete(le.predictionCache, key)
	}
	return nil, false
}

// SetCachedPrediction stores a prediction in cache
func (le *LearningEngine) SetCachedPrediction(key string, prediction float64, confidence float64, features map[string]float64) {
	le.mu.Lock()
	defer le.mu.Unlock()
	
	le.predictionCache[key] = &CachedPrediction{
		Prediction: prediction,
		Timestamp:  time.Now(),
		Features:   features,
		Confidence: confidence,
	}
}

// PredictRemediationSuccess uses AI to predict remediation success probability
func (le *LearningEngine) PredictRemediationSuccess(component, strategy string, currentMetrics map[string]float64) float64 {
	if !le.IsAIFeatureEnabled(FeatureAIPoweredRemediation) {
		// Fallback to simple historical success rate
		return le.GetStrategySuccessRate(strategy)
	}
	
	cacheKey := fmt.Sprintf("%s:%s", component, strategy)
	if cached, found := le.GetCachedPrediction(cacheKey); found {
		if le.metricsMatch(cached.Features, currentMetrics, 0.1) {
			return cached.Prediction
		}
	}
	
	// Simple ML prediction based on historical data
	prediction := le.calculateAIPrediction(component, strategy, currentMetrics)
	confidence := le.calculatePredictionConfidence(component, strategy)
	
	le.SetCachedPrediction(cacheKey, prediction, confidence, currentMetrics)
	return prediction
}

// metricsMatch checks if two metric sets are similar within tolerance
func (le *LearningEngine) metricsMatch(cached, current map[string]float64, tolerance float64) bool {
	for key, cachedVal := range cached {
		if currentVal, exists := current[key]; exists {
			if abs(cachedVal-currentVal) > tolerance {
				return false
			}
		}
	}
	return true
}

// calculateAIPrediction performs AI-based prediction (simplified implementation)
func (le *LearningEngine) calculateAIPrediction(component, strategy string, metrics map[string]float64) float64 {
	le.mu.RLock()
	defer le.mu.RUnlock()
	
	// Get historical success rate as base
	baseRate := le.GetStrategySuccessRate(strategy)
	
	// Adjust based on current metrics (simplified ML logic)
	adjustment := 0.0
	if cpuUsage, exists := metrics["cpu_usage"]; exists {
		if cpuUsage > 0.8 {
			adjustment -= 0.2 // High CPU usage reduces success probability
		} else if cpuUsage < 0.3 {
			adjustment += 0.1 // Low CPU usage increases success probability
		}
	}
	
	if errorRate, exists := metrics["error_rate"]; exists {
		if errorRate > 0.05 {
			adjustment -= 0.3 // High error rate reduces success probability
		}
	}
	
	result := baseRate + adjustment
	if result < 0 {
		result = 0
	} else if result > 1 {
		result = 1
	}
	
	return result
}

// calculatePredictionConfidence calculates confidence level for predictions
func (le *LearningEngine) calculatePredictionConfidence(component, strategy string) float64 {
	le.mu.RLock()
	defer le.mu.RUnlock()
	
	// Confidence based on number of historical samples
	if strat, exists := le.strategies[strategy]; exists {
		if strat.Total < 10 {
			return 0.3 // Low confidence with few samples
		} else if strat.Total < 50 {
			return 0.7 // Medium confidence
		} else {
			return 0.9 // High confidence with many samples
		}
	}
	return 0.1 // Very low confidence for unknown strategies
}

// abs returns absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
