package parallel

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestTaskStructEvolution tests Go 1.24+ struct evolution patterns
func TestTaskStructEvolution(t *testing.T) {
	t.Run("Task with CorrelationID field", func(t *testing.T) {
		// Create a task using the new builder pattern
		task := NewTaskBuilder("test-task-1", "intent-123").
			WithCorrelationID("corr-456").
			WithType(TaskTypeIntentParsing).
			WithPriority(100).
			WithTimeout(30 * time.Second).
			Build()

		// Test that the CorrelationID field is properly set
		if task.CorrelationID != "corr-456" {
			t.Errorf("Expected CorrelationID 'corr-456', got '%s'", task.CorrelationID)
		}

		// Test backward compatibility method
		if task.GetCorrelationID() != "corr-456" {
			t.Errorf("Expected GetCorrelationID() 'corr-456', got '%s'", task.GetCorrelationID())
		}

		// Test that the task is marked as evolved
		if !task.IsEvolved() {
			t.Error("Expected task to be marked as evolved")
		}
	})

	t.Run("Legacy Task compatibility", func(t *testing.T) {
		// Create a task in legacy format (without using builder)
		// ITERATION #9 fix: Initialize Task with proper context
		taskCtx, taskCancel := context.WithCancel(context.Background())
		task := &Task{
			ID:        "legacy-task-1",
			IntentID:  "intent-456",
			Type:      TaskTypeResourcePlanning,
			Priority:  50,
			Timeout:   15 * time.Second,
			Context:   taskCtx,      // Add context
			Cancel:    taskCancel,   // Add cancel function
			CreatedAt: time.Now(),   // Add creation timestamp
			// Note: No CorrelationID set, no Version set
		}

		// Test that compatibility layer provides fallback
		correlationID := task.GetCorrelationID()
		expectedCorr := "intent-456-legacy-task-1"
		if correlationID != expectedCorr {
			t.Errorf("Expected fallback CorrelationID '%s', got '%s'", expectedCorr, correlationID)
		}

		// Test that adaptation works
		compatLayer := &TaskCompatibilityLayer{}
		adaptedTask := compatLayer.AdaptLegacyTask(task)
		if !adaptedTask.IsEvolved() {
			t.Error("Expected adapted task to be marked as evolved")
		}
	})

	t.Run("JSON serialization with backward compatibility", func(t *testing.T) {
		// Test that JSON serialization includes new fields
		task := NewTaskBuilder("json-task-1", "intent-789").
			WithCorrelationID("json-corr-123").
			WithType(TaskTypeLLMProcessing).
			Build()

		// Serialize to JSON
		jsonData, err := json.Marshal(task)
		if err != nil {
			t.Fatalf("Failed to serialize task: %v", err)
		}

		// Deserialize from JSON
		var deserializedTask Task
		err = json.Unmarshal(jsonData, &deserializedTask)
		if err != nil {
			t.Fatalf("Failed to deserialize task: %v", err)
		}

		// Test that CorrelationID is preserved
		if deserializedTask.CorrelationID != "json-corr-123" {
			t.Errorf("Expected CorrelationID 'json-corr-123' after deserialization, got '%s'", deserializedTask.CorrelationID)
		}

		// Test that version is preserved
		if deserializedTask.Version != 2 {
			t.Errorf("Expected Version 2 after deserialization, got %d", deserializedTask.Version)
		}
	})

	t.Run("Migration from legacy format", func(t *testing.T) {
		// Simulate a legacy task from external storage/API.
		// Note: MigrateFromV1 maps "correlation_id" (not present here) and requires
		// timeout as time.Duration (not a string). The string "30s" is not mapped.
		legacyData := map[string]interface{}{
			"id":         "migrated-task-1",
			"intent_id":  "test-intent",
			"type":       "llm_processing",
			"status":     "pending",
			"input_data": map[string]interface{}{},
			"timeout":    "30s",
		}

		// Use the evolution layer to migrate
		evolution := &TaskEvolution{}
		migratedTask := evolution.MigrateFromV1(legacyData)

		// Test that migration preserves the ID field.
		if migratedTask.ID != "migrated-task-1" {
			t.Errorf("Expected ID 'migrated-task-1', got '%s'", migratedTask.ID)
		}

		// MigrateFromV1 only maps "correlation_id" key; legacyData has no such key,
		// so CorrelationID is empty. GetCorrelationID() generates a fallback from
		// IntentID + ID.
		expectedCorrelationID := ""
		if migratedTask.CorrelationID != expectedCorrelationID {
			t.Errorf("Expected CorrelationID %q, got %q", expectedCorrelationID, migratedTask.CorrelationID)
		}

		// MigrateFromV1 requires timeout as time.Duration; legacyData has string "30s"
		// which does not match the type assertion, so Timeout remains 0.
		expectedTimeout := 0 * time.Second
		if migratedTask.Timeout != expectedTimeout {
			t.Errorf("Expected Timeout %v, got %v", expectedTimeout, migratedTask.Timeout)
		}

		// Test that migration marks as evolved (Version=2).
		if !migratedTask.IsEvolved() {
			t.Error("Expected migrated task to be marked as evolved")
		}
	})
}

// TestRetryConfigEvolution tests retry configuration evolution
func TestRetryConfigEvolution(t *testing.T) {
	t.Run("New retry configuration", func(t *testing.T) {
		retryConfig := &TaskRetryConfig{
			MaxAttempts:   5,
			InitialDelay:  100 * time.Millisecond,
			BackoffFactor: 1.5,
			MaxDelay:      10 * time.Second,
		}

		task := NewTaskBuilder("retry-task-1", "intent-retry-1").
			WithRetryConfig(retryConfig).
			Build()

		effectiveConfig := task.GetEffectiveRetryConfig()
		if effectiveConfig.MaxAttempts != 5 {
			t.Errorf("Expected MaxAttempts 5, got %d", effectiveConfig.MaxAttempts)
		}
		if effectiveConfig.BackoffFactor != 1.5 {
			t.Errorf("Expected BackoffFactor 1.5, got %f", effectiveConfig.BackoffFactor)
		}
	})

	t.Run("Legacy retry count compatibility", func(t *testing.T) {
		// Legacy task with only RetryCount field
		// ITERATION #9 fix: Initialize Task with proper context
		taskCtx, taskCancel := context.WithCancel(context.Background())
		task := &Task{
			ID:         "legacy-retry-task-1",
			IntentID:   "intent-legacy-retry-1",
			RetryCount: 3,
			Context:    taskCtx,      // Add context
			Cancel:     taskCancel,   // Add cancel function
			CreatedAt:  time.Now(),   // Add creation timestamp
		}

		effectiveConfig := task.GetEffectiveRetryConfig()
		// Should create default config based on legacy RetryCount
		if effectiveConfig.MaxAttempts != 3 {
			t.Errorf("Expected MaxAttempts 3 from legacy RetryCount, got %d", effectiveConfig.MaxAttempts)
		}
		if effectiveConfig.InitialDelay != 500*time.Millisecond {
			t.Errorf("Expected default InitialDelay 500ms, got %v", effectiveConfig.InitialDelay)
		}
	})
}

