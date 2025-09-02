package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// TestShutdownSequencing verifies that shutdown happens in the correct order:
// 1. Stop accepting new files
// 2. Drain existing queued work
// 3. Stop coordinator and cancel context
func TestShutdownSequencing(t *testing.T) {
	handoffDir := t.TempDir()

	mockValidator := &MockValidator{}
	var processedCount int64
	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
		atomic.AddInt64(&processedCount, 1)
		time.Sleep(50 * time.Millisecond) // Simulate work
		return nil
	}

	processor, err := NewProcessor(&ProcessorConfig{
		HandoffDir:    handoffDir,
		ErrorDir:      handoffDir + "/errors",
		PorchMode:     "direct",
		BatchSize:     3,
		BatchInterval: 100 * time.Millisecond,
		MaxRetries:    1,
		SendTimeout:   1 * time.Second,
		WorkerCount:   2,
	}, mockValidator, mockPorchFunc)
	require.NoError(t, err)

	processor.StartBatchProcessor()

	// Create actual test files and queue work before shutdown
	var wg sync.WaitGroup
	numFiles := 10
	var queueErrors int64

	// Create valid test intent files
	validIntent := ingest.Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "default",
		Replicas:   3,
	}

	intentData, err := json.Marshal(validIntent)
	require.NoError(t, err)

	for i := 0; i < numFiles; i++ {
		// Create actual file on disk
		testFile := filepath.Join(handoffDir, fmt.Sprintf("file-%d.json", i))
		require.NoError(t, os.WriteFile(testFile, intentData, 0o644))

		wg.Add(1)
		go func(id int, filename string) {
			defer wg.Done()
			if err := processor.ProcessFile(filename); err != nil {
				atomic.AddInt64(&queueErrors, 1)
				t.Logf("Queue error for file-%d: %v", id, err)
			}
		}(i, testFile)
	}

	// Wait for files to be queued
	wg.Wait()

	// Now shutdown - this should drain the queue before stopping
	shutdownStart := time.Now()
	processor.Stop()
	shutdownDuration := time.Since(shutdownStart)

	// Verify shutdown timing and results
	finalProcessed := atomic.LoadInt64(&processedCount)
	finalQueueErrors := atomic.LoadInt64(&queueErrors)

	t.Logf("Shutdown completed in %v", shutdownDuration)
	t.Logf("Files processed: %d, Queue errors: %d", finalProcessed, finalQueueErrors)

	// Assertions about shutdown behavior
	assert.GreaterOrEqual(t, finalProcessed, int64(5), "At least some files should be processed before shutdown")
	assert.LessOrEqual(t, finalQueueErrors, int64(5), "Queue errors should be minimal during proper shutdown")
	assert.LessOrEqual(t, shutdownDuration, 10*time.Second, "Shutdown should complete within reasonable time")

	// Test that new files are rejected after shutdown
	err = processor.ProcessFile("after-shutdown.json")
	assert.Error(t, err, "Should reject new files after shutdown")
	assert.Contains(t, err.Error(), "shutting down", "Error should indicate shutdown state")
}
