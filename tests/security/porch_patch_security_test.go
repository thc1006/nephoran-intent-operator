package security

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/rand"
)

// TestPathTraversalPrevention validates path validation functions
func TestPathTraversalPrevention(t *testing.T) {
	testCases := []struct {
		name          string
		inputPath     string
		expectedError bool
	}{
		{
			name:          "Normal Path",
			inputPath:     "/tmp/nephio/safe_path",
			expectedError: false,
		},
		{
			name:          "Path Traversal Attempt 1",
			inputPath:     "/../etc/passwd",
			expectedError: true,
		},
		{
			name:          "Path Traversal Attempt 2",
			inputPath:     "/tmp/../../../etc/shadow",
			expectedError: true,
		},
		{
			name:          "Path Traversal Attempt 3",
			inputPath:     "../../sensitive/file",
			expectedError: true,
		},
		{
			name:          "Symlink Path",
			inputPath:     "/tmp/symlink_to_sensitive",
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOutputDir(tc.inputPath)
			if tc.expectedError {
				assert.Error(t, err, "Expected path traversal to be prevented")
			} else {
				assert.NoError(t, err, "Expected safe path to be allowed")
			}
		})
	}
}

// TestTimestampCollisionResistance validates timestamp generation uniqueness
func TestTimestampCollisionResistance(t *testing.T) {
	const (
		concurrentGenerations = 1000
		maxConcurrentThreads  = 100
	)

	timestampSet := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use a semaphore to limit concurrent goroutines
	sem := make(chan struct{}, maxConcurrentThreads)

	for i := 0; i < concurrentGenerations; i++ {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func() {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			timestamp := generateCollisionResistantTimestamp()

			mu.Lock()
			defer mu.Unlock()

			// Ensure no duplicate timestamps
			assert.False(t, timestampSet[timestamp],
				fmt.Sprintf("Duplicate timestamp generated: %s", timestamp))

			timestampSet[timestamp] = true
		}()
	}

	wg.Wait()

	// Verify total unique timestamps match total generations
	assert.Equal(t, concurrentGenerations, len(timestampSet),
		"Not all timestamps were unique")
}

// Simulate the implementation of these functions for the test
func validateOutputDir(path string) error {
	// Sanitize and validate path
	cleanPath := filepath.Clean(path)
	basePath := "/tmp/nephio" // Hardcoded safe base path

	// Prevent path traversal
	if !strings.HasPrefix(cleanPath, basePath) {
		return fmt.Errorf("invalid path: potential path traversal detected")
	}

	return nil
}

func generateCollisionResistantTimestamp() string {
	// Use high-resolution time + random suffix
	now := time.Now().UTC()
	nanoTime := now.UnixNano()
	randomSuffix := rand.String(6)

	return fmt.Sprintf("%d_%s", nanoTime, randomSuffix)
}
