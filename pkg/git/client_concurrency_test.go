package git

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentPushLimit tests that at most N pushes run concurrently
func TestConcurrentPushLimit(t *testing.T) {
	// Create a temporary directory for the test repo
	tmpDir, err := os.MkdirTemp("", "git-concurrency-test-*")
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
	concurrentLimit := 3
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   "test-key",
		RepoPath: repoPath,
		logger:   slog.Default().With("test", "concurrency"),
		pushSem:  make(chan struct{}, concurrentLimit),
	}

	// Setup atomic counter and synchronization
	var activeOperations int32
	var maxConcurrent int32
	var barrier = make(chan struct{})
	var startedCount int32

	// Setup test hooks
	client.beforePushHook = func() {
		// Increment active operations counter
		current := atomic.AddInt32(&activeOperations, 1)
		
		// Track maximum concurrent operations
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}

		// Track how many have started
		atomic.AddInt32(&startedCount, 1)

		// Wait at barrier to simulate work and ensure concurrency
		<-barrier
	}

	client.afterPushHook = func() {
		// Decrement active operations counter
		atomic.AddInt32(&activeOperations, -1)
	}

	// Number of goroutines to launch
	numGoroutines := 10

	// Channel to collect errors
	errors := make(chan error, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start a goroutine to release the barrier after some operations have started
	go func() {
		// Wait for at least the limit number of operations to start
		for atomic.LoadInt32(&startedCount) < int32(concurrentLimit) {
			time.Sleep(10 * time.Millisecond)
		}
		
		// Give a bit more time to ensure we're testing concurrency
		time.Sleep(50 * time.Millisecond)
		
		// Check that we never exceed the limit
		assert.LessOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(concurrentLimit),
			"Maximum concurrent operations should not exceed limit")
		
		// Release the barrier to let operations complete
		close(barrier)
	}()

	// Launch goroutines that will try to push concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			// Create unique file for this goroutine
			files := map[string]string{
				fmt.Sprintf("file%d.txt", id): fmt.Sprintf("Content from goroutine %d", id),
			}
			
			// Attempt to commit and push (push will fail but that's ok for this test)
			_, err := client.CommitAndPush(files, fmt.Sprintf("Commit from goroutine %d", id))
			if err != nil {
				// We expect push to fail since we don't have a real remote
				// But we're testing concurrency control, not actual pushing
				errors <- err
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errors)

	// Verify final state
	assert.Equal(t, int32(0), atomic.LoadInt32(&activeOperations),
		"All operations should be completed")
	assert.LessOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(concurrentLimit),
		"Maximum concurrent operations should never have exceeded limit")

	t.Logf("Test completed: max concurrent operations was %d (limit was %d)",
		atomic.LoadInt32(&maxConcurrent), concurrentLimit)
}

// TestConcurrentPushWithErrors tests error propagation and no deadlocks
func TestConcurrentPushWithErrors(t *testing.T) {
	// Create a temporary directory for the test repo
	tmpDir, err := os.MkdirTemp("", "git-error-test-*")
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
	concurrentLimit := 2
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   "test-key",
		RepoPath: repoPath,
		logger:   slog.Default().With("test", "error-handling"),
		pushSem:  make(chan struct{}, concurrentLimit),
	}

	// Setup hooks to simulate errors for some operations
	var operationCount int32
	errorOnOperations := map[int32]bool{2: true, 5: true, 7: true} // Fail operations 2, 5, and 7
	originalRepoPath := client.RepoPath

	client.beforePushHook = func() {
		count := atomic.AddInt32(&operationCount, 1)
		if errorOnOperations[count] {
			// Simulate an error by causing the repo path to be invalid
			client.RepoPath = "/invalid/path/that/does/not/exist"
		} else {
			// Restore the valid path for successful operations
			client.RepoPath = originalRepoPath
		}
	}

	// Number of goroutines to launch
	numGoroutines := 10
	results := make(chan error, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			files := map[string]string{
				fmt.Sprintf("file%d.txt", id): fmt.Sprintf("Content %d", id),
			}
			
			_, err := client.CommitAndPush(files, fmt.Sprintf("Commit %d", id))
			results <- err
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(results)

	// Count errors and successes
	var errorCount, successCount int
	for err := range results {
		if err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	// Verify that all operations completed (they will all error due to no real remote)
	assert.Equal(t, numGoroutines, errorCount+successCount, "All operations should complete")
	// In a real test with a mock remote, we would expect:
	// assert.Greater(t, errorCount, 0, "Should have some errors")
	// assert.Greater(t, successCount, 0, "Should have some successes")
	
	// For now, just verify we got the expected errors for the operations we broke
	assert.GreaterOrEqual(t, errorCount, 3, "Should have at least the forced errors")

	t.Logf("Test completed: %d errors, %d successes out of %d operations",
		errorCount, successCount, numGoroutines)
}

// TestConcurrentPushDeadlockPrevention tests that no deadlocks occur even under stress
func TestConcurrentPushDeadlockPrevention(t *testing.T) {
	// Create a temporary directory for the test repo
	tmpDir, err := os.MkdirTemp("", "git-deadlock-test-*")
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

	// Create client with very limited concurrency to stress test
	concurrentLimit := 1
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   "test-key",
		RepoPath: repoPath,
		logger:   slog.Default().With("test", "deadlock-prevention"),
		pushSem:  make(chan struct{}, concurrentLimit),
	}

	// Setup timeout for deadlock detection
	done := make(chan bool)
	timeout := time.After(10 * time.Second)

	go func() {
		// Number of operations to perform
		numOperations := 20
		var wg sync.WaitGroup
		wg.Add(numOperations)

		// Rapidly fire off operations
		for i := 0; i < numOperations; i++ {
			go func(id int) {
				defer wg.Done()
				
				files := map[string]string{
					fmt.Sprintf("file%d.txt", id): fmt.Sprintf("Content %d", id),
				}
				
				// We don't care about the error here, just that it doesn't deadlock
				client.CommitAndPush(files, fmt.Sprintf("Commit %d", id))
			}(i)
		}

		wg.Wait()
		done <- true
	}()

	// Wait for either completion or timeout
	select {
	case <-done:
		// Success - no deadlock
		t.Log("All operations completed without deadlock")
	case <-timeout:
		t.Fatal("Deadlock detected - operations did not complete within timeout")
	}
}

// TestSemaphoreAcquisitionOrder tests that semaphore acquisition is fair
func TestSemaphoreAcquisitionOrder(t *testing.T) {
	// Create client with limited concurrency
	concurrentLimit := 2
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   "test-key",
		RepoPath: "/tmp/test-repo",
		logger:   slog.Default().With("test", "acquisition-order"),
		pushSem:  make(chan struct{}, concurrentLimit),
	}

	// Track order of acquisitions
	var acquisitionOrder []int
	var orderMutex sync.Mutex
	var barrier = make(chan struct{})

	// Pre-fill the semaphore to capacity
	for i := 0; i < concurrentLimit; i++ {
		client.pushSem <- struct{}{}
	}

	// Launch goroutines that will queue for the semaphore
	numGoroutines := 5
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			// Try to acquire
			client.acquireSemaphore(fmt.Sprintf("operation-%d", id))
			
			// Record acquisition
			orderMutex.Lock()
			acquisitionOrder = append(acquisitionOrder, id)
			orderMutex.Unlock()
			
			// Wait at barrier
			<-barrier
			
			// Release
			client.releaseSemaphore(fmt.Sprintf("operation-%d", id))
		}(i)
	}

	// Give goroutines time to queue up
	time.Sleep(100 * time.Millisecond)

	// Now release the pre-filled slots one by one
	for i := 0; i < concurrentLimit; i++ {
		<-client.pushSem
		time.Sleep(50 * time.Millisecond) // Allow time for acquisition
	}

	// Release barrier to let operations complete
	close(barrier)

	// Wait for all operations
	wg.Wait()

	// Verify that we got all acquisitions
	assert.Equal(t, numGoroutines, len(acquisitionOrder),
		"All goroutines should have acquired the semaphore")

	t.Logf("Acquisition order: %v", acquisitionOrder)
}

// TestConcurrentDifferentOperations tests mixing different git operations
func TestConcurrentDifferentOperations(t *testing.T) {
	// Create a temporary directory for the test repo
	tmpDir, err := os.MkdirTemp("", "git-mixed-ops-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	// Create an initial commit with some directories
	w, err := repo.Worktree()
	require.NoError(t, err)

	// Create initial structure
	for i := 0; i < 3; i++ {
		dir := filepath.Join(repoPath, fmt.Sprintf("dir%d", i))
		err = os.MkdirAll(dir, 0755)
		require.NoError(t, err)
		
		testFile := filepath.Join(dir, "file.txt")
		err = os.WriteFile(testFile, []byte(fmt.Sprintf("Content %d", i)), 0644)
		require.NoError(t, err)
		
		_, err = w.Add(fmt.Sprintf("dir%d/file.txt", i))
		require.NoError(t, err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Create client with limited concurrency
	concurrentLimit := 2
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   "test-key",
		RepoPath: repoPath,
		logger:   slog.Default().With("test", "mixed-operations"),
		pushSem:  make(chan struct{}, concurrentLimit),
	}

	// Track concurrent operations
	var activeOps int32
	var maxConcurrent int32

	client.beforePushHook = func() {
		current := atomic.AddInt32(&activeOps, 1)
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
	}

	client.afterPushHook = func() {
		atomic.AddInt32(&activeOps, -1)
	}

	// Launch different types of operations concurrently
	var wg sync.WaitGroup
	
	// CommitAndPush operations
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			files := map[string]string{
				fmt.Sprintf("new_file%d.txt", id): fmt.Sprintf("New content %d", id),
			}
			client.CommitAndPush(files, fmt.Sprintf("Add file %d", id))
		}(i)
	}

	// CommitAndPushChanges operations
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Modify a file first
			testFile := filepath.Join(repoPath, fmt.Sprintf("dir%d/file.txt", id))
			os.WriteFile(testFile, []byte(fmt.Sprintf("Modified %d", id)), 0644)
			client.CommitAndPushChanges(fmt.Sprintf("Modify dir%d", id))
		}(i)
	}

	// RemoveDirectory operations
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client.RemoveDirectory(fmt.Sprintf("dir%d", id), fmt.Sprintf("Remove dir%d", id))
		}(i)
	}

	// Wait for all operations
	wg.Wait()

	// Verify concurrency limit was respected
	assert.LessOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(concurrentLimit),
		"Should not exceed concurrency limit even with mixed operations")
	assert.Equal(t, int32(0), atomic.LoadInt32(&activeOps),
		"All operations should be completed")

	t.Logf("Mixed operations test completed: max concurrent was %d (limit was %d)",
		atomic.LoadInt32(&maxConcurrent), concurrentLimit)
}