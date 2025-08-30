package loop

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// TestCoordinatorChannelBackpressure verifies that the coordinator channel
// is properly sized to handle burst load without send timeouts
func TestCoordinatorChannelBackpressure(t *testing.T) {
	handoffDir := t.TempDir()

	mockValidator := &MockValidator{}
	var processedCount int64

	// Slow porch function to create backpressure
	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error {
		atomic.AddInt64(&processedCount, 1)
		time.Sleep(100 * time.Millisecond) // Slow processing
		return nil
	}

	processor, err := NewProcessor(&ProcessorConfig{
		HandoffDir:    handoffDir,
		ErrorDir:      handoffDir + "/errors",
		PorchMode:     "direct",
		BatchSize:     5,
		BatchInterval: 200 * time.Millisecond,
		MaxRetries:    1,
		SendTimeout:   1 * time.Second,
		WorkerCount:   4,
	}, mockValidator, mockPorchFunc)
	require.NoError(t, err)

	processor.StartBatchProcessor()
	defer processor.Stop()

	// Send burst of files rapidly to test channel buffering
	numFiles := 25 // More than typical buffer size
	var sendErrors int64
	var successfulSends int64

	start := time.Now()
	for i := 0; i < numFiles; i++ {
		err := processor.ProcessFile(fmt.Sprintf("burst-file-%d.json", i))
		if err != nil {
			atomic.AddInt64(&sendErrors, 1)
			t.Logf("Send error %d: %v", i, err)
		} else {
			atomic.AddInt64(&successfulSends, 1)
		}
	}
	sendDuration := time.Since(start)

	t.Logf("Burst send completed in %v: %d successful, %d errors",
		sendDuration, successfulSends, sendErrors)

	// With proper channel sizing, most sends should succeed immediately
	assert.GreaterOrEqual(t, successfulSends, int64(20), "Most files should queue successfully")
	assert.LessOrEqual(t, sendErrors, int64(5), "Send errors should be minimal with proper buffering")
}
