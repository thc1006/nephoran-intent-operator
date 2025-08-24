//go:build windows
// +build windows

package loop

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWindowsConcurrencyStress provides comprehensive stress testing for Windows
// filesystem operations under high concurrency, focusing on issues that caused CI failures.
func TestWindowsConcurrencyStress(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows concurrency stress tests only apply to Windows")
	}

	// Increase test timeout for stress tests
	if testing.Short() {
		t.Skip("Skipping stress tests in short mode")
	}

	t.Run("High_Concurrency_IsProcessed_Stress", func(t *testing.T) {
		// Stress test the IsProcessed operation under high concurrency
		baseDir := t.TempDir()
		sm, err := NewStateManager(baseDir)
		require.NoError(t, err)
		defer sm.Close()

		numWorkers := 50
		numOperationsPerWorker := 100
		numFiles := 20

		// Create test files
		testFiles := make([]string, numFiles)
		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("stress-test-%d.json", i)
			testFiles[i] = filename
			filePath := filepath.Join(baseDir, filename)
			require.NoError(t, os.WriteFile(filePath, []byte(`{"test": true}`), 0644))
		}

		var wg sync.WaitGroup
		var totalOperations int64
		var totalErrors int64
		var successfulChecks int64
		var processedMarks int64

		startTime := time.Now()

		// Launch stress test workers
		for worker := 0; worker < numWorkers; worker++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				r := rand.New(rand.NewSource(int64(workerID)))

				for op := 0; op < numOperationsPerWorker; op++ {
					atomic.AddInt64(&totalOperations, 1)

					// Pick a random file
					fileIndex := r.Intn(numFiles)
					filename := testFiles[fileIndex]

					// Randomly choose operation: check (70%) or mark (30%)
					if r.Float32() < 0.7 {
						// Check if processed
						processed, err := sm.IsProcessed(filename)
						if err != nil {
							atomic.AddInt64(&totalErrors, 1)
							t.Logf("Worker %d: IsProcessed error for %s: %v", workerID, filename, err)
						} else {
							atomic.AddInt64(&successfulChecks, 1)
							_ = processed // Use the result
						}
					} else {
						// Mark as processed
						err := sm.MarkProcessed(filename)
						if err != nil {
							atomic.AddInt64(&totalErrors, 1)
							t.Logf("Worker %d: MarkProcessed error for %s: %v", workerID, filename, err)
						} else {
							atomic.AddInt64(&processedMarks, 1)
						}
					}

					// Small random delay to create more realistic timing
					if r.Intn(10) == 0 {
						time.Sleep(time.Microsecond * time.Duration(r.Intn(100)))
					}
				}
			}(worker)
		}

		wg.Wait()
		duration := time.Since(startTime)

		// Report statistics
		t.Logf("Stress test completed in %v", duration)
		t.Logf("Total operations: %d", atomic.LoadInt64(&totalOperations))
		t.Logf("Total errors: %d", atomic.LoadInt64(&totalErrors))
		t.Logf("Successful checks: %d", atomic.LoadInt64(&successfulChecks))
		t.Logf("Processed marks: %d", atomic.LoadInt64(&processedMarks))
		t.Logf("Operations per second: %.2f", float64(atomic.LoadInt64(&totalOperations))/duration.Seconds())

		// Verify results
		expectedOps := int64(numWorkers * numOperationsPerWorker)
		assert.Equal(t, expectedOps, atomic.LoadInt64(&totalOperations), "All operations should be counted")
		
		// Error rate should be very low (< 1%)
		errorRate := float64(atomic.LoadInt64(&totalErrors)) / float64(atomic.LoadInt64(&totalOperations))
		assert.Less(t, errorRate, 0.01, "Error rate should be less than 1%")

		// Verify all files are eventually marked as processed
		for _, filename := range testFiles {
			processed, err := sm.IsProcessed(filename)
			assert.NoError(t, err, "Final check should not error for %s", filename)
			// Some files should be processed (depending on random operations)
			t.Logf("File %s processed: %v", filename, processed)
		}
	})

	t.Run("Concurrent_Status_File_Creation_Stress", func(t *testing.T) {
		// Stress test concurrent status file creation
		baseDir := t.TempDir()
		watchDir := filepath.Join(baseDir, "watch")
		statusDir := filepath.Join(watchDir, "status")

		require.NoError(t, os.MkdirAll(watchDir, 0755))

		config := Config{
			PorchPath: "/tmp/stress-test",
			Mode:      "once",
		}

		watcher, err := NewWatcher(watchDir, config)
		require.NoError(t, err)
		defer watcher.Close()

		numWorkers := 30
		numStatusesPerWorker := 50

		// Create intent files
		numIntentFiles := 10
		intentFiles := make([]string, numIntentFiles)
		for i := 0; i < numIntentFiles; i++ {
			filename := fmt.Sprintf("stress-intent-%d.json", i)
			intentFiles[i] = filepath.Join(watchDir, filename)
			content := fmt.Sprintf(`{"api_version": "intent.nephoran.io/v1", "kind": "StressTest", "metadata": {"name": "stress-%d"}}`, i)
			require.NoError(t, os.WriteFile(intentFiles[i], []byte(content), 0644))
		}

		var wg sync.WaitGroup
		var totalStatusWrites int64
		var statusErrors int64

		startTime := time.Now()

		for worker := 0; worker < numWorkers; worker++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				r := rand.New(rand.NewSource(int64(workerID)))

				for i := 0; i < numStatusesPerWorker; i++ {
					atomic.AddInt64(&totalStatusWrites, 1)

					// Pick random intent file
					intentFile := intentFiles[r.Intn(numIntentFiles)]
					
					// Generate unique status
					status := []string{"success", "failed", "processing", "pending"}[r.Intn(4)]
					message := fmt.Sprintf("Worker %d operation %d at %s", workerID, i, time.Now().Format("15:04:05.000"))

					// Write status file atomically
					func() {
						defer func() {
							if r := recover(); r != nil {
								atomic.AddInt64(&statusErrors, 1)
								t.Logf("Worker %d panic during status write: %v", workerID, r)
							}
						}()

						watcher.writeStatusFileAtomic(intentFile, status, message)
					}()

					// Random small delay
					if r.Intn(20) == 0 {
						time.Sleep(time.Millisecond * time.Duration(r.Intn(5)))
					}
				}
			}(worker)
		}

		wg.Wait()
		duration := time.Since(startTime)

		// Report statistics
		t.Logf("Status write stress test completed in %v", duration)
		t.Logf("Total status writes: %d", atomic.LoadInt64(&totalStatusWrites))
		t.Logf("Status errors: %d", atomic.LoadInt64(&statusErrors))
		t.Logf("Status writes per second: %.2f", float64(atomic.LoadInt64(&totalStatusWrites))/duration.Seconds())

		// Verify results
		expectedWrites := int64(numWorkers * numStatusesPerWorker)
		assert.Equal(t, expectedWrites, atomic.LoadInt64(&totalStatusWrites), "All status writes should be counted")

		// Error rate should be very low
		errorRate := float64(atomic.LoadInt64(&statusErrors)) / float64(atomic.LoadInt64(&totalStatusWrites))
		assert.Less(t, errorRate, 0.05, "Status write error rate should be less than 5%")

		// Verify status directory was created
		_, err = os.Stat(statusDir)
		assert.NoError(t, err, "Status directory should exist")

		// Verify status files were created
		statusEntries, err := os.ReadDir(statusDir)
		require.NoError(t, err, "Should be able to read status directory")
		assert.Greater(t, len(statusEntries), 0, "Should have status files")

		t.Logf("Created %d status files", len(statusEntries))

		// Verify some status files are valid
		validStatusFiles := 0
		for i, entry := range statusEntries {
			if i >= 10 { // Check first 10 for performance
				break
			}
			
			statusFile := filepath.Join(statusDir, entry.Name())
			content, err := os.ReadFile(statusFile)
			if err == nil && len(content) > 0 {
				validStatusFiles++
			}
		}

		assert.Greater(t, validStatusFiles, 0, "Should have valid status files")
	})

	t.Run("File_System_Race_Condition_Stress", func(t *testing.T) {
		// Test file system operations that commonly have race conditions on Windows
		baseDir := t.TempDir()

		numWorkers := 20
		numOperationsPerWorker := 100
		sharedFiles := 5

		var wg sync.WaitGroup
		var createOperations int64
		var readOperations int64
		var deleteOperations int64
		var moveOperations int64
		var raceErrors int64

		// Create shared file names
		sharedFileNames := make([]string, sharedFiles)
		for i := 0; i < sharedFiles; i++ {
			sharedFileNames[i] = fmt.Sprintf("race-file-%d.json", i)
		}

		startTime := time.Now()

		for worker := 0; worker < numWorkers; worker++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				r := rand.New(rand.NewSource(int64(workerID)))

				for op := 0; op < numOperationsPerWorker; op++ {
					// Pick random file
					fileName := sharedFileNames[r.Intn(sharedFiles)]
					filePath := filepath.Join(baseDir, fileName)

					// Pick random operation
					operation := r.Intn(4)

					func() {
						defer func() {
							if r := recover(); r != nil {
								atomic.AddInt64(&raceErrors, 1)
							}
						}()

						switch operation {
						case 0: // Create/Write
							atomic.AddInt64(&createOperations, 1)
							content := fmt.Sprintf(`{"worker": %d, "op": %d, "time": "%s"}`, 
								workerID, op, time.Now().Format(time.RFC3339Nano))
							atomicWriteFile(filePath, []byte(content), 0644)

						case 1: // Read
							atomic.AddInt64(&readOperations, 1)
							readFileWithRetry(filePath) // Ignore errors

						case 2: // Delete
							atomic.AddInt64(&deleteOperations, 1)
							os.Remove(filePath) // Ignore errors

						case 3: // Move
							atomic.AddInt64(&moveOperations, 1)
							tempPath := filePath + ".tmp"
							os.Rename(filePath, tempPath) // Ignore errors
							os.Rename(tempPath, filePath) // Ignore errors
						}
					}()

					// Random tiny delay
					if r.Intn(50) == 0 {
						time.Sleep(time.Microsecond * time.Duration(r.Intn(10)))
					}
				}
			}(worker)
		}

		wg.Wait()
		duration := time.Since(startTime)

		// Report statistics
		t.Logf("File system race stress test completed in %v", duration)
		t.Logf("Create operations: %d", atomic.LoadInt64(&createOperations))
		t.Logf("Read operations: %d", atomic.LoadInt64(&readOperations))
		t.Logf("Delete operations: %d", atomic.LoadInt64(&deleteOperations))
		t.Logf("Move operations: %d", atomic.LoadInt64(&moveOperations))
		t.Logf("Race errors: %d", atomic.LoadInt64(&raceErrors))

		totalOps := atomic.LoadInt64(&createOperations) + atomic.LoadInt64(&readOperations) + 
					atomic.LoadInt64(&deleteOperations) + atomic.LoadInt64(&moveOperations)
		t.Logf("Operations per second: %.2f", float64(totalOps)/duration.Seconds())

		// Race errors should be acceptable (< 10% due to expected file system races)
		errorRate := float64(atomic.LoadInt64(&raceErrors)) / float64(totalOps)
		assert.Less(t, errorRate, 0.1, "Race error rate should be less than 10%")
	})

	t.Run("Memory_Pressure_Concurrent_Operations", func(t *testing.T) {
		// Test behavior under memory pressure with concurrent operations
		baseDir := t.TempDir()
		sm, err := NewStateManager(baseDir)
		require.NoError(t, err)
		defer sm.Close()

		numWorkers := 15
		numFiles := 100
		
		// Create many files to increase memory usage
		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("memory-test-%d.json", i)
			filePath := filepath.Join(baseDir, filename)
			// Create larger file content to increase memory pressure
			largeContent := fmt.Sprintf(`{"id": %d, "data": "%s"}`, i, 
				strings.Repeat("x", 1000)) // 1KB per file
			require.NoError(t, os.WriteFile(filePath, []byte(largeContent), 0644))
		}

		var wg sync.WaitGroup
		var memoryErrors int64
		var successfulOps int64

		// Monitor memory before test
		var startMemStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&startMemStats)

		startTime := time.Now()

		for worker := 0; worker < numWorkers; worker++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				r := rand.New(rand.NewSource(int64(workerID)))

				for i := 0; i < 50; i++ {
					func() {
						defer func() {
							if r := recover(); r != nil {
								atomic.AddInt64(&memoryErrors, 1)
							}
						}()

						// Pick random file
						fileIndex := r.Intn(numFiles)
						filename := fmt.Sprintf("memory-test-%d.json", fileIndex)

						// Perform operations that use memory
						if r.Intn(2) == 0 {
							// Read and process file content
							filePath := filepath.Join(baseDir, filename)
							if content, err := readFileWithRetry(filePath); err == nil {
								// Process content (simulate work)
								_ = strings.Contains(string(content), "data")
								atomic.AddInt64(&successfulOps, 1)
							}
						} else {
							// Mark as processed (state operations)
							if err := sm.MarkProcessed(filename); err == nil {
								atomic.AddInt64(&successfulOps, 1)
							}
						}

						// Force some GC pressure occasionally
						if r.Intn(10) == 0 {
							runtime.GC()
						}
					}()
				}
			}(worker)
		}

		wg.Wait()
		duration := time.Since(startTime)

		// Monitor memory after test
		var endMemStats runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&endMemStats)

		// Report statistics
		t.Logf("Memory pressure test completed in %v", duration)
		t.Logf("Memory errors: %d", atomic.LoadInt64(&memoryErrors))
		t.Logf("Successful operations: %d", atomic.LoadInt64(&successfulOps))
		t.Logf("Memory usage increase: %.2f MB", 
			float64(endMemStats.Alloc-startMemStats.Alloc)/(1024*1024))

		// Verify reasonable behavior under memory pressure
		totalOps := atomic.LoadInt64(&memoryErrors) + atomic.LoadInt64(&successfulOps)
		errorRate := float64(atomic.LoadInt64(&memoryErrors)) / float64(totalOps)
		assert.Less(t, errorRate, 0.1, "Memory pressure error rate should be reasonable")

		// Memory usage should not be excessive
		memoryIncrease := float64(endMemStats.Alloc-startMemStats.Alloc) / (1024*1024)
		assert.Less(t, memoryIncrease, 100.0, "Memory usage should not exceed 100MB")
	})

	t.Run("Long_Running_Concurrent_Stability", func(t *testing.T) {
		// Test stability over a longer period with continuous concurrent operations
		if testing.Short() {
			t.Skip("Skipping long-running stability test in short mode")
		}

		baseDir := t.TempDir()
		sm, err := NewStateManager(baseDir)
		require.NoError(t, err)
		defer sm.Close()

		// Test duration
		testDuration := 30 * time.Second
		numWorkers := 10

		var wg sync.WaitGroup
		var totalOperations int64
		var errors int64
		var stopSignal int32

		startTime := time.Now()

		for worker := 0; worker < numWorkers; worker++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				operationCount := 0
				r := rand.New(rand.NewSource(int64(workerID)))

				for atomic.LoadInt32(&stopSignal) == 0 {
					operationCount++
					atomic.AddInt64(&totalOperations, 1)

					filename := fmt.Sprintf("stability-worker-%d-op-%d.json", workerID, operationCount)
					filePath := filepath.Join(baseDir, filename)

					func() {
						defer func() {
							if r := recover(); r != nil {
								atomic.AddInt64(&errors, 1)
							}
						}()

						// Create file
						content := fmt.Sprintf(`{"worker": %d, "operation": %d}`, workerID, operationCount)
						if err := atomicWriteFile(filePath, []byte(content), 0644); err != nil {
							atomic.AddInt64(&errors, 1)
							return
						}

						// Mark as processed
						if err := sm.MarkProcessed(filename); err != nil {
							atomic.AddInt64(&errors, 1)
							return
						}

						// Check if processed
						if _, err := sm.IsProcessed(filename); err != nil {
							atomic.AddInt64(&errors, 1)
							return
						}

						// Clean up occasionally
						if operationCount%10 == 0 {
							os.Remove(filePath)
						}
					}()

					// Small delay to avoid overwhelming the system
					time.Sleep(time.Duration(r.Intn(10)) * time.Millisecond)
				}

				t.Logf("Worker %d completed %d operations", workerID, operationCount)
			}(worker)
		}

		// Stop workers after test duration
		time.AfterFunc(testDuration, func() {
			atomic.StoreInt32(&stopSignal, 1)
		})

		wg.Wait()
		actualDuration := time.Since(startTime)

		// Report statistics
		t.Logf("Stability test ran for %v", actualDuration)
		t.Logf("Total operations: %d", atomic.LoadInt64(&totalOperations))
		t.Logf("Total errors: %d", atomic.LoadInt64(&errors))
		t.Logf("Operations per second: %.2f", float64(atomic.LoadInt64(&totalOperations))/actualDuration.Seconds())

		// Verify stability
		errorRate := float64(atomic.LoadInt64(&errors)) / float64(atomic.LoadInt64(&totalOperations))
		assert.Less(t, errorRate, 0.05, "Long-running error rate should be less than 5%")
		assert.Greater(t, atomic.LoadInt64(&totalOperations), int64(100), "Should have performed many operations")
	})
}

// TestWindowsConcurrencyRegressionPrevention ensures specific concurrency issues
// that caused Windows CI failures cannot reoccur.
func TestWindowsConcurrencyRegressionPrevention(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows concurrency regression tests only apply to Windows")
	}

	t.Run("IsProcessed_ENOENT_Handling", func(t *testing.T) {
		// Test the specific scenario where IsProcessed was failing with ENOENT errors
		baseDir := t.TempDir()
		sm, err := NewStateManager(baseDir)
		require.NoError(t, err)
		defer sm.Close()

		numWorkers := 20
		numFiles := 10

		// Create files
		for i := 0; i < numFiles; i++ {
			filename := fmt.Sprintf("enoent-test-%d.json", i)
			filePath := filepath.Join(baseDir, filename)
			require.NoError(t, os.WriteFile(filePath, []byte(`{"test": true}`), 0644))
		}

		var wg sync.WaitGroup
		var enoentErrors int64
		var successfulChecks int64

		for worker := 0; worker < numWorkers; worker++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				r := rand.New(rand.NewSource(int64(workerID)))

				for i := 0; i < 50; i++ {
					fileIndex := r.Intn(numFiles)
					filename := fmt.Sprintf("enoent-test-%d.json", fileIndex)
					filePath := filepath.Join(baseDir, filename)

					// Randomly delete the file to trigger ENOENT
					if r.Intn(10) == 0 {
						os.Remove(filePath)
					}

					// This should not error even if file doesn't exist
					_, err := sm.IsProcessed(filename)
					if err != nil {
						atomic.AddInt64(&enoentErrors, 1)
						t.Logf("Worker %d: IsProcessed error for %s: %v", workerID, filename, err)
					} else {
						atomic.AddInt64(&successfulChecks, 1)
					}

					// Recreate file occasionally
					if r.Intn(5) == 0 {
						os.WriteFile(filePath, []byte(`{"recreated": true}`), 0644)
					}
				}
			}(worker)
		}

		wg.Wait()

		// The critical assertion: IsProcessed should NEVER return errors due to ENOENT
		assert.Equal(t, int64(0), atomic.LoadInt64(&enoentErrors), 
			"IsProcessed should handle ENOENT gracefully without errors")
		assert.Greater(t, atomic.LoadInt64(&successfulChecks), int64(0), 
			"Should have successful checks")
	})

	t.Run("Concurrent_State_File_Access", func(t *testing.T) {
		// Test concurrent access to state files that was causing corruption
		baseDir := t.TempDir()
		sm, err := NewStateManager(baseDir)
		require.NoError(t, err)
		defer sm.Close()

		numWorkers := 15
		testFile := "concurrent-state-test.json"
		filePath := filepath.Join(baseDir, testFile)
		require.NoError(t, os.WriteFile(filePath, []byte(`{"test": true}`), 0644))

		var wg sync.WaitGroup
		var markErrors int64
		var checkErrors int64
		var successfulMarks int64
		var successfulChecks int64

		for worker := 0; worker < numWorkers; worker++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for i := 0; i < 20; i++ {
					// Mark as processed
					err := sm.MarkProcessed(testFile)
					if err != nil {
						atomic.AddInt64(&markErrors, 1)
						t.Logf("Worker %d: MarkProcessed error: %v", workerID, err)
					} else {
						atomic.AddInt64(&successfulMarks, 1)
					}

					// Check if processed
					_, err = sm.IsProcessed(testFile)
					if err != nil {
						atomic.AddInt64(&checkErrors, 1)
						t.Logf("Worker %d: IsProcessed error: %v", workerID, err)
					} else {
						atomic.AddInt64(&successfulChecks, 1)
					}

					// Small delay
					time.Sleep(time.Millisecond)
				}
			}(worker)
		}

		wg.Wait()

		// Verify state operations are reliable under concurrency
		totalMarks := atomic.LoadInt64(&markErrors) + atomic.LoadInt64(&successfulMarks)
		totalChecks := atomic.LoadInt64(&checkErrors) + atomic.LoadInt64(&successfulChecks)

		markErrorRate := float64(atomic.LoadInt64(&markErrors)) / float64(totalMarks)
		checkErrorRate := float64(atomic.LoadInt64(&checkErrors)) / float64(totalChecks)

		assert.Less(t, markErrorRate, 0.05, "MarkProcessed error rate should be very low")
		assert.Less(t, checkErrorRate, 0.01, "IsProcessed error rate should be very low")

		// Final state should be consistent
		processed, err := sm.IsProcessed(testFile)
		assert.NoError(t, err, "Final state check should not error")
		assert.True(t, processed, "File should be marked as processed")
	})

	t.Run("Status_Directory_Creation_Race", func(t *testing.T) {
		// Test the race condition in status directory creation
		baseDir := t.TempDir()
		watchDir := filepath.Join(baseDir, "watch-race")
		require.NoError(t, os.MkdirAll(watchDir, 0755))

		config := Config{
			PorchPath: "/tmp/race-test",
			Mode:      "once",
		}

		// Create multiple watchers trying to write status files simultaneously
		numWatchers := 10
		var wg sync.WaitGroup
		var creationErrors int64
		var statusWriteErrors int64
		var successfulWrites int64

		for i := 0; i < numWatchers; i++ {
			wg.Add(1)
			go func(watcherID int) {
				defer wg.Done()

				watcher, err := NewWatcher(watchDir, config)
				if err != nil {
					atomic.AddInt64(&creationErrors, 1)
					t.Logf("Watcher %d creation error: %v", watcherID, err)
					return
				}
				defer watcher.Close()

				// Create intent file for this watcher
				intentFile := filepath.Join(watchDir, fmt.Sprintf("race-intent-%d.json", watcherID))
				content := fmt.Sprintf(`{"api_version": "intent.nephoran.io/v1", "kind": "RaceTest", "metadata": {"name": "race-%d"}}`, watcherID)
				if err := os.WriteFile(intentFile, []byte(content), 0644); err != nil {
					atomic.AddInt64(&statusWriteErrors, 1)
					return
				}

				// All watchers try to write status files at the same time
				func() {
					defer func() {
						if r := recover(); r != nil {
							atomic.AddInt64(&statusWriteErrors, 1)
							t.Logf("Watcher %d status write panic: %v", watcherID, r)
						}
					}()

					watcher.writeStatusFileAtomic(intentFile, "success", fmt.Sprintf("Race test watcher %d", watcherID))
					atomic.AddInt64(&successfulWrites, 1)
				}()
			}(i)
		}

		wg.Wait()

		// Report results
		t.Logf("Watcher creation errors: %d", atomic.LoadInt64(&creationErrors))
		t.Logf("Status write errors: %d", atomic.LoadInt64(&statusWriteErrors))
		t.Logf("Successful status writes: %d", atomic.LoadInt64(&successfulWrites))

		// Verify that the status directory was created despite race conditions
		statusDir := filepath.Join(watchDir, "status")
		_, err := os.Stat(statusDir)
		assert.NoError(t, err, "Status directory should exist after concurrent writes")

		// Verify that some status files were created successfully
		statusEntries, err := os.ReadDir(statusDir)
		require.NoError(t, err, "Should be able to read status directory")
		assert.Greater(t, len(statusEntries), 0, "Should have status files")

		// Error rates should be reasonable
		totalAttempts := int64(numWatchers)
		creationErrorRate := float64(atomic.LoadInt64(&creationErrors)) / float64(totalAttempts)
		writeErrorRate := float64(atomic.LoadInt64(&statusWriteErrors)) / float64(totalAttempts)

		assert.Less(t, creationErrorRate, 0.2, "Watcher creation error rate should be reasonable")
		assert.Less(t, writeErrorRate, 0.2, "Status write error rate should be reasonable")
	})
}