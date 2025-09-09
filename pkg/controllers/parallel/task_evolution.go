package parallel

import (
	"encoding/json"
	"fmt"
	"time"
)

<<<<<<< HEAD
=======

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// TaskBuilder provides Go 1.24+ struct evolution patterns for Task construction
type TaskBuilder struct {
	task *Task
}

// NewTaskBuilder creates a new task builder with Go 1.24+ evolution patterns
func NewTaskBuilder(id, intentID string) *TaskBuilder {
	return &TaskBuilder{
		task: &Task{
			ID:        id,
			IntentID:  intentID,
			Status:    TaskStatusPending,
			CreatedAt: time.Now(),
			Version:   2, // Mark as evolved version
		},
	}
}

// WithCorrelationID sets the correlation ID for tracing (Go 1.24+ migration)
func (tb *TaskBuilder) WithCorrelationID(correlationID string) *TaskBuilder {
	tb.task.CorrelationID = correlationID
	return tb
}

// WithType sets the task type
func (tb *TaskBuilder) WithType(taskType TaskType) *TaskBuilder {
	tb.task.Type = taskType
	return tb
}

// WithPriority sets the task priority
func (tb *TaskBuilder) WithPriority(priority int) *TaskBuilder {
	tb.task.Priority = priority
	return tb
}

// WithTimeout sets the task timeout
func (tb *TaskBuilder) WithTimeout(timeout time.Duration) *TaskBuilder {
	tb.task.Timeout = timeout
	return tb
}

// WithDependencies sets task dependencies
func (tb *TaskBuilder) WithDependencies(dependencies []string) *TaskBuilder {
	tb.task.Dependencies = dependencies
	return tb
}

// WithInputData sets input data
func (tb *TaskBuilder) WithInputData(inputData map[string]interface{}) *TaskBuilder {
	if data, err := json.Marshal(inputData); err == nil {
		tb.task.InputData = json.RawMessage(data)
	}
	return tb
}

// WithRetryConfig sets advanced retry configuration (Go 1.24+ evolution)
func (tb *TaskBuilder) WithRetryConfig(config *TaskRetryConfig) *TaskBuilder {
	tb.task.RetryConfig = config
	return tb
}

// WithMetadata sets task metadata
func (tb *TaskBuilder) WithMetadata(metadata map[string]string) *TaskBuilder {
	tb.task.Metadata = metadata
	return tb
}

// Build returns the constructed task
func (tb *TaskBuilder) Build() *Task {
	return tb.task
}

// TaskEvolution provides backward compatibility and migration helpers
type TaskEvolution struct{}

// MigrateFromV1 migrates a legacy task to new schema (Go 1.24+ migration)
func (te *TaskEvolution) MigrateFromV1(legacyTask map[string]interface{}) *Task {
	task := &Task{
		Version: 2, // Mark as migrated
	}

	// Map legacy fields to new schema
	if id, ok := legacyTask["id"].(string); ok {
		task.ID = id
	}
	if intentID, ok := legacyTask["intent_id"].(string); ok {
		task.IntentID = intentID
	}
	if correlationID, ok := legacyTask["correlation_id"].(string); ok {
		task.CorrelationID = correlationID
	}
	if timeout, ok := legacyTask["timeout"].(time.Duration); ok {
		task.Timeout = timeout
	}
	if priority, ok := legacyTask["priority"].(int); ok {
		task.Priority = priority
	}

	return task
}

// GetCorrelationID returns the correlation ID with fallback logic
func (t *Task) GetCorrelationID() string {
	if t.CorrelationID != "" {
		return t.CorrelationID
	}
	// Generate one if missing for backward compatibility
	if t.ID != "" && t.IntentID != "" {
		return fmt.Sprintf("%s-%s", t.IntentID, t.ID)
	}
	return ""
}

// IsEvolved returns true if task uses the new schema
func (t *Task) IsEvolved() bool {
	return t.Version >= 2
}

// GetEffectiveTimeout returns the timeout with backward compatibility
func (t *Task) GetEffectiveTimeout() time.Duration {
	if t.Timeout > 0 {
		return t.Timeout
	}
	return DefaultTimeout
}

// GetEffectiveRetryConfig returns retry config with backward compatibility
func (t *Task) GetEffectiveRetryConfig() *TaskRetryConfig {
	if t.RetryConfig != nil {
		return t.RetryConfig
	}

	// Create default retry config based on legacy RetryCount
	return &TaskRetryConfig{
		MaxAttempts:   max(t.RetryCount, 3),
		InitialDelay:  500 * time.Millisecond,
		BackoffFactor: 2.0,
		MaxDelay:      5 * time.Second,
	}
}

// max helper function for Go versions that don't have built-in max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TaskCompatibilityLayer provides methods for zero-downtime migration
type TaskCompatibilityLayer struct{}

// AdaptLegacyTask adapts a task created with legacy patterns
func (tcl *TaskCompatibilityLayer) AdaptLegacyTask(task *Task) *Task {
	if task.Version == 0 {
		// Mark as migrated and ensure required fields
		task.Version = 2

		// Ensure correlation ID is present
		if task.CorrelationID == "" && task.GetCorrelationID() != "" {
			task.CorrelationID = task.GetCorrelationID()
		}

		// Migrate retry configuration if needed
		if task.RetryConfig == nil && task.RetryCount > 0 {
			task.RetryConfig = &TaskRetryConfig{
				MaxAttempts:   task.RetryCount,
				InitialDelay:  500 * time.Millisecond,
				BackoffFactor: 1.5,
				MaxDelay:      3 * time.Second,
			}
		}
	}

	return task
}

// ValidateTaskSchema validates task schema evolution
func (tcl *TaskCompatibilityLayer) ValidateTaskSchema(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if task.IntentID == "" {
		return fmt.Errorf("task IntentID is required")
	}
	if task.Type == "" {
		return fmt.Errorf("task Type is required")
	}

	// Evolution-specific validations
	if task.IsEvolved() {
		if task.CorrelationID == "" {
			task.CorrelationID = task.GetCorrelationID()
		}
	}

	return nil
}
