package automation

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AutomatedRemediation performs intelligent remediation actions
type AutomatedRemediation struct {
	mu                     sync.RWMutex
	config                 *SelfHealingConfig
	logger                 *slog.Logger
	k8sClient              kubernetes.Interface
	ctrlClient             client.Client
	activeRemediations     map[string]*RemediationSession
	remediationStrategies  map[string]*RemediationStrategy
	learningEngine         *LearningEngine
	rollbackManager        *RollbackManager
}

// NewAutomatedRemediation creates a new automated remediation system
func NewAutomatedRemediation(config *SelfHealingConfig, k8sClient kubernetes.Interface, ctrlClient client.Client, logger *slog.Logger) (*AutomatedRemediation, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	ar := &AutomatedRemediation{
		config:                config,
		logger:                logger,
		k8sClient:             k8sClient,
		ctrlClient:            ctrlClient,
		activeRemediations:    make(map[string]*RemediationSession),
		remediationStrategies: make(map[string]*RemediationStrategy),
		learningEngine:        NewLearningEngine(logger),
		rollbackManager:       NewRollbackManager(logger),
	}

	// Initialize default remediation strategies
	ar.initializeDefaultStrategies()

	return ar, nil
}

// Start starts the automated remediation system
func (ar *AutomatedRemediation) Start(ctx context.Context) error {
	ar.logger.Info("Starting automated remediation system")
	
	// Start monitoring active remediations
	go ar.monitorActiveRemediations(ctx)
	
	return nil
}

// InitiateRemediation initiates remediation for a component
func (ar *AutomatedRemediation) InitiateRemediation(ctx context.Context, component, reason string) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	// Check if we're already remediating this component
	for _, session := range ar.activeRemediations {
		if session.Component == component && session.Status == "RUNNING" {
			return fmt.Errorf("remediation already in progress for component %s", component)
		}
	}

	// Check concurrent remediation limit
	activeCount := 0
	for _, session := range ar.activeRemediations {
		if session.Status == "RUNNING" {
			activeCount++
		}
	}

	if activeCount >= ar.config.MaxConcurrentRemediations {
		return fmt.Errorf("maximum concurrent remediations (%d) reached", ar.config.MaxConcurrentRemediations)
	}

	// Find suitable strategy
	strategy := ar.findBestStrategy(component, reason)
	if strategy == nil {
		return fmt.Errorf("no suitable remediation strategy found for component %s", component)
	}

	// Create remediation session
	session := &RemediationSession{
		ID:        fmt.Sprintf("remediation-%s-%d", component, time.Now().Unix()),
		Component: component,
		Strategy:  strategy.Name,
		Status:    "PENDING",
		StartTime: time.Now(),
		Actions:   make([]*RemediationAction, 0),
		Results:   make(map[string]interface{}),
	}

	// Create backup if configured
	if ar.config.BackupBeforeRemediation {
		session.BackupID = fmt.Sprintf("backup-%s", session.ID)
		// In real implementation, would create actual backup
	}

	// Create rollback plan if configured
	if ar.config.RollbackOnFailure {
		session.RollbackPlan = ar.rollbackManager.CreateRollbackPlan(session)
	}

	ar.activeRemediations[session.ID] = session

	ar.logger.Info("Initiated remediation",
		"session_id", session.ID,
		"component", component,
		"strategy", strategy.Name,
		"reason", reason)

	// Start remediation execution
	go ar.executeRemediation(ctx, session, strategy)

	return nil
}

// InitiatePreventiveRemediation initiates preventive remediation based on predictions
func (ar *AutomatedRemediation) InitiatePreventiveRemediation(ctx context.Context, component string, probability float64) error {
	reason := fmt.Sprintf("predictive failure probability: %.2f", probability)
	return ar.InitiateRemediation(ctx, component, reason)
}

// GetActiveRemediations returns currently active remediation sessions
func (ar *AutomatedRemediation) GetActiveRemediations() map[string]*RemediationSession {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	result := make(map[string]*RemediationSession)
	for k, v := range ar.activeRemediations {
		if v.Status == "RUNNING" || v.Status == "PENDING" {
			result[k] = v
		}
	}
	return result
}

// initializeDefaultStrategies initializes default remediation strategies
func (ar *AutomatedRemediation) initializeDefaultStrategies() {
	// Restart strategy
	restartStrategy := &RemediationStrategy{
		Name: "restart-strategy",
		Conditions: []*RemediationCondition{
			{
				Metric:    "health_check_failures",
				Operator:  "GT",
				Threshold: 3,
				Duration:  2 * time.Minute,
			},
		},
		Actions: []*RemediationActionTemplate{
			{
				Type:     "RESTART",
				Template: "restart-deployment",
				Parameters: map[string]interface{}{
					"graceful": true,
					"timeout":  "60s",
				},
				Timeout: 5 * time.Minute,
				RetryPolicy: &RetryPolicy{
					MaxAttempts:       3,
					InitialDelay:      30 * time.Second,
					MaxDelay:          5 * time.Minute,
					BackoffMultiplier: 2.0,
				},
			},
		},
		Priority:    1,
		Success:     0,
		Total:       0,
		SuccessRate: 0.0,
	}

	// Scale strategy
	scaleStrategy := &RemediationStrategy{
		Name: "scale-strategy",
		Conditions: []*RemediationCondition{
			{
				Metric:    "cpu_usage",
				Operator:  "GT",
				Threshold: 80.0,
				Duration:  5 * time.Minute,
			},
			{
				Metric:    "response_time",
				Operator:  "GT",
				Threshold: 1000.0, // ms
				Duration:  3 * time.Minute,
			},
		},
		Actions: []*RemediationActionTemplate{
			{
				Type:     "SCALE",
				Template: "scale-deployment",
				Parameters: map[string]interface{}{
					"scale_factor": 1.5,
					"max_replicas": 10,
				},
				Timeout: 10 * time.Minute,
			},
		},
		Priority:    2,
		Success:     0,
		Total:       0,
		SuccessRate: 0.0,
	}

	ar.remediationStrategies["restart"] = restartStrategy
	ar.remediationStrategies["scale"] = scaleStrategy
}

// findBestStrategy finds the best remediation strategy for a component
func (ar *AutomatedRemediation) findBestStrategy(component, reason string) *RemediationStrategy {
	// Simple strategy selection based on reason
	// In a real implementation, this would use more sophisticated logic
	
	if contains(reason, "health") || contains(reason, "failure") {
		return ar.remediationStrategies["restart"]
	}
	
	if contains(reason, "cpu") || contains(reason, "memory") || contains(reason, "latency") {
		return ar.remediationStrategies["scale"]
	}
	
	// Default to restart strategy
	return ar.remediationStrategies["restart"]
}

// executeRemediation executes a remediation session
func (ar *AutomatedRemediation) executeRemediation(ctx context.Context, session *RemediationSession, strategy *RemediationStrategy) {
	ar.mu.Lock()
	session.Status = "RUNNING"
	ar.mu.Unlock()

	ar.logger.Info("Executing remediation", "session_id", session.ID, "strategy", strategy.Name)

	success := true
	
	// Execute each action in the strategy
	for _, actionTemplate := range strategy.Actions {
		action := &RemediationAction{
			Type:       actionTemplate.Type,
			Target:     session.Component,
			Parameters: actionTemplate.Parameters,
			Status:     "PENDING",
			StartTime:  time.Now(),
		}

		session.Actions = append(session.Actions, action)

		// Execute the action
		success = ar.executeAction(ctx, session, action, actionTemplate)
		if !success {
			break
		}
	}

	// Update session status
	ar.mu.Lock()
	if success {
		session.Status = "COMPLETED"
		session.Results["success"] = true
	} else {
		session.Status = "FAILED"
		session.Results["success"] = false
		
		// Execute rollback if configured and plan exists
		if ar.config.RollbackOnFailure && session.RollbackPlan != nil {
			ar.logger.Info("Executing rollback plan", "session_id", session.ID)
			ar.rollbackManager.ExecuteRollback(session.RollbackPlan.ID)
		}
	}
	
	endTime := time.Now()
	session.EndTime = &endTime
	session.Results["duration"] = endTime.Sub(session.StartTime).String()
	ar.mu.Unlock()

	// Record outcome for learning
	ar.learningEngine.RecordRemediation(session, strategy, success)

	ar.logger.Info("Remediation completed",
		"session_id", session.ID,
		"success", success,
		"duration", endTime.Sub(session.StartTime))
}

// executeAction executes a single remediation action
func (ar *AutomatedRemediation) executeAction(ctx context.Context, session *RemediationSession, action *RemediationAction, template *RemediationActionTemplate) bool {
	action.Status = "RUNNING"
	
	ar.logger.Info("Executing action",
		"session_id", session.ID,
		"action_type", action.Type,
		"target", action.Target)

	// Simulate action execution
	// In a real implementation, this would perform actual operations
	time.Sleep(2 * time.Second)

	// Simulate success/failure
	success := true // In real implementation, would depend on actual operation result

	endTime := time.Now()
	action.EndTime = &endTime
	
	if success {
		action.Status = "COMPLETED"
		action.Result = "success"
	} else {
		action.Status = "FAILED"
		action.Error = "simulated failure"
	}

	return success
}

// monitorActiveRemediations monitors active remediation sessions
func (ar *AutomatedRemediation) monitorActiveRemediations(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ar.cleanupCompletedSessions()
		}
	}
}

// cleanupCompletedSessions removes old completed sessions
func (ar *AutomatedRemediation) cleanupCompletedSessions() {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour) // Keep sessions for 1 hour

	for sessionID, session := range ar.activeRemediations {
		if (session.Status == "COMPLETED" || session.Status == "FAILED") &&
			session.EndTime != nil && session.EndTime.Before(cutoff) {
			delete(ar.activeRemediations, sessionID)
		}
	}
}

// Helper function (reuse from edge package or make it shared)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr ||
		     containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}