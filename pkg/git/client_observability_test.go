package git

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObservabilityLogging tests that the correct debug logs are emitted
func TestObservabilityLogging(t *testing.T) {
	// Create a buffer to capture logs
	var logBuffer bytes.Buffer
	
	// Create a JSON handler that writes to our buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create a temporary directory for the test repo
	tmpDir, err := os.MkdirTemp("", "git-observability-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	// Create an initial commit
	w, err := repo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(testFile, []byte("Initial content"), 0644)
	require.NoError(t, err)

	_, err = w.Add("README.md")
	require.NoError(t, err)

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Create client with limited concurrency and custom logger
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   "test-key",
		RepoPath: repoPath,
		logger:   logger,
		pushSem:  make(chan struct{}, 2), // Limit of 2
	}

	// Test acquiring and releasing semaphore
	client.acquireSemaphore("test-operation-1")
	client.acquireSemaphore("test-operation-2")
	
	// This one should wait (in a goroutine to not block)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.acquireSemaphore("test-operation-3")
		client.releaseSemaphore("test-operation-3")
	}()
	
	// Give it time to log the waiting message
	time.Sleep(50 * time.Millisecond)
	
	// Release one to let the waiting operation proceed
	client.releaseSemaphore("test-operation-1")
	client.releaseSemaphore("test-operation-2")
	
	wg.Wait()

	// Parse and verify logs
	logs := strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
	
	foundAcquired := false
	foundWaiting := false
	foundReleased := false
	foundInFlight := false
	foundLimit := false

	for _, logLine := range logs {
		if logLine == "" {
			continue
		}
		
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(logLine), &logEntry)
		require.NoError(t, err, "Failed to parse log: %s", logLine)
		
		msg, ok := logEntry["msg"].(string)
		if !ok {
			continue
		}
		
		// Check for expected log messages
		if strings.Contains(msg, "git push: acquired semaphore") {
			foundAcquired = true
			
			// Verify in_flight and limit fields exist
			if _, ok := logEntry["in_flight"]; ok {
				foundInFlight = true
			}
			if _, ok := logEntry["limit"]; ok {
				foundLimit = true
			}
		}
		
		if strings.Contains(msg, "git push: waiting on semaphore") {
			foundWaiting = true
			
			// Verify in_flight and limit fields exist in waiting log too
			if _, ok := logEntry["in_flight"]; ok {
				foundInFlight = true
			}
			if _, ok := logEntry["limit"]; ok {
				foundLimit = true
			}
		}
		
		if strings.Contains(msg, "git push: released semaphore") {
			foundReleased = true
		}
	}

	// Verify all expected logs were found
	assert.True(t, foundAcquired, "Should have logged semaphore acquisition")
	assert.True(t, foundWaiting, "Should have logged semaphore waiting")
	assert.True(t, foundReleased, "Should have logged semaphore release")
	assert.True(t, foundInFlight, "Should have logged in_flight field")
	assert.True(t, foundLimit, "Should have logged limit field")
}

// TestObservabilityMetrics tests that metrics are properly updated
func TestObservabilityMetrics(t *testing.T) {
	// Create a test registry
	registry := prometheus.NewRegistry()
	
	// Initialize metrics with our test registry
	InitMetrics(registry)
	
	// Create a temporary directory for the test repo
	tmpDir, err := os.MkdirTemp("", "git-metrics-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	// Create an initial commit
	w, err := repo.Worktree()
	require.NoError(t, err)

	testFile := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(testFile, []byte("Initial content"), 0644)
	require.NoError(t, err)

	_, err = w.Add("README.md")
	require.NoError(t, err)

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Create client with limited concurrency
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   "test-key",
		RepoPath: repoPath,
		logger:   slog.Default(),
		pushSem:  make(chan struct{}, 3), // Limit of 3
	}

	// Verify initial metric value
	assert.Equal(t, float64(0), testutil.ToFloat64(gitPushInFlightGauge))

	// Acquire semaphores and check metrics
	client.acquireSemaphore("operation-1")
	assert.Equal(t, float64(1), testutil.ToFloat64(gitPushInFlightGauge))

	client.acquireSemaphore("operation-2")
	assert.Equal(t, float64(2), testutil.ToFloat64(gitPushInFlightGauge))

	client.acquireSemaphore("operation-3")
	assert.Equal(t, float64(3), testutil.ToFloat64(gitPushInFlightGauge))

	// Release and check metrics decrease
	client.releaseSemaphore("operation-1")
	assert.Equal(t, float64(2), testutil.ToFloat64(gitPushInFlightGauge))

	client.releaseSemaphore("operation-2")
	assert.Equal(t, float64(1), testutil.ToFloat64(gitPushInFlightGauge))

	client.releaseSemaphore("operation-3")
	assert.Equal(t, float64(0), testutil.ToFloat64(gitPushInFlightGauge))
}

// TestObservabilityUnderLoad tests observability during concurrent operations
func TestObservabilityUnderLoad(t *testing.T) {
	// Create a buffer to capture logs
	var logBuffer bytes.Buffer
	
	// Create a JSON handler that writes to our buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create client with very limited concurrency to force waiting
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   "test-key",
		RepoPath: "/tmp/test-repo",
		logger:   logger,
		pushSem:  make(chan struct{}, 2), // Very limited
	}

	// Launch multiple operations concurrently
	numOperations := 10
	var wg sync.WaitGroup
	wg.Add(numOperations)

	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			operation := fmt.Sprintf("operation-%d", id)
			
			client.acquireSemaphore(operation)
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			client.releaseSemaphore(operation)
		}(i)
	}

	wg.Wait()

	// Parse logs and count different message types
	logs := strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
	
	var acquiredCount, waitingCount, releasedCount int
	var maxInFlight float64

	for _, logLine := range logs {
		if logLine == "" {
			continue
		}
		
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(logLine), &logEntry)
		if err != nil {
			continue
		}
		
		msg, _ := logEntry["msg"].(string)
		
		switch {
		case strings.Contains(msg, "git push: acquired semaphore"):
			acquiredCount++
			if inFlight, ok := logEntry["in_flight"].(float64); ok && inFlight > maxInFlight {
				maxInFlight = inFlight
			}
		case strings.Contains(msg, "git push: waiting on semaphore"):
			waitingCount++
		case strings.Contains(msg, "git push: released semaphore"):
			releasedCount++
		}
	}

	// Verify expectations
	assert.Equal(t, numOperations, acquiredCount, "Should acquire semaphore for each operation")
	assert.Equal(t, numOperations, releasedCount, "Should release semaphore for each operation")
	assert.Greater(t, waitingCount, 0, "Should have some operations waiting due to limited concurrency")
	assert.LessOrEqual(t, maxInFlight, float64(2), "In-flight should never exceed limit of 2")

	t.Logf("Load test summary: %d acquired, %d waited, %d released, max in-flight: %.0f",
		acquiredCount, waitingCount, releasedCount, maxInFlight)
}

// TestMetricsRegistrationOnce tests that metrics are only registered once
func TestMetricsRegistrationOnce(t *testing.T) {
	// Create multiple registries
	registry1 := prometheus.NewRegistry()
	registry2 := prometheus.NewRegistry()

	// Reset the metricsOnce for this test (in a real scenario this wouldn't be needed)
	// Note: This is just for testing, normally sync.Once can't be reset
	metricsOnce = sync.Once{}
	gitPushInFlightGauge = nil

	// First initialization should succeed
	assert.NotPanics(t, func() {
		InitMetrics(registry1)
	}, "First metrics initialization should not panic")

	// Verify metric was created
	assert.NotNil(t, gitPushInFlightGauge, "Metric should be created after first init")

	// Second initialization should be a no-op (not panic due to duplicate registration)
	assert.NotPanics(t, func() {
		InitMetrics(registry2)
	}, "Second metrics initialization should not panic")
}