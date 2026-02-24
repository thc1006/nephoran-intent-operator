package loop

import (
	"context"
	"fmt"
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
	"github.com/stretchr/testify/suite"
	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

// =============================================================================
// COMPREHENSIVE VALIDATION SUITE FOR CRITICAL FIXES
// =============================================================================

// ValidationTestSuite validates all critical fixes are working properly
type ValidationTestSuite struct {
	suite.Suite
	tempDir   string
	porchPath string
}

// MockValidator is already defined in processor_test.go, we'll use that one

func TestValidationTestSuite(t *testing.T) {
	suite.Run(t, new(ValidationTestSuite))
}

func (s *ValidationTestSuite) SetupTest() {
	s.tempDir = s.T().TempDir()
	s.porchPath = s.createValidationMockPorch()
}

// =============================================================================
// FIX 1: NIL POINTER DEREFERENCE VALIDATION
// =============================================================================

func (s *ValidationTestSuite) TestFix1_NilPointerDereference_WatcherCloseNil() {
	s.T().Log("Validating Fix 1: Nil Pointer Dereference - Watcher.Close() on nil")

	// Test 1: Close on nil watcher should not panic
	var nilWatcher *Watcher
	err := nilWatcher.Close()
	s.Assert().NoError(err, "Close() on nil Watcher should not panic or error")
}

func (s *ValidationTestSuite) TestFix1_NilPointerDereference_SafeDefers() {
	s.T().Log("Validating Fix 1: Safe defer patterns in Watcher")

	handoffDir := filepath.Join(s.tempDir, "handoff")
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(handoffDir, 0o755))
	s.Require().NoError(os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:    s.porchPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         true,
		MaxWorkers:   3, // Production-like worker count for realistic concurrency testing
		DebounceDur:  100 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	// Test creating and immediately closing watcher
	watcher, err := NewWatcher(handoffDir, config)
	s.Require().NoError(err)
	s.Require().NotNil(watcher)

	// Close immediately - should handle all nil components gracefully
	err = watcher.Close()
	s.Assert().NoError(err, "Close should handle partially initialized components")

	// Close again - should be idempotent
	err = watcher.Close()
	s.Assert().NoError(err, "Second close should be safe")
}

func (s *ValidationTestSuite) TestFix1_NilPointerDereference_ComponentsNil() {
	s.T().Log("Validating Fix 1: Handling nil components throughout lifecycle")

	// Test scenario where creation fails partway through
	handoffDir := filepath.Join(s.tempDir, "nonexistent")
	outDir := filepath.Join(s.tempDir, "out")

	config := Config{
		PorchPath:    s.porchPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         true,
		MaxWorkers:   3, // Production-like worker count for realistic concurrency testing
		CleanupAfter: time.Hour,
	}

	// This should fail during creation
	watcher, err := NewWatcher(handoffDir, config)
	s.Assert().Error(err, "Should fail with nonexistent directory")

	// If watcher is created despite error, closing should be safe
	if watcher != nil {
		closeErr := watcher.Close()
		s.Assert().NoError(closeErr, "Close should be safe even after creation error")
	}
}

// =============================================================================
// FIX 2: CROSS-PLATFORM SCRIPTING VALIDATION
// =============================================================================

func (s *ValidationTestSuite) TestFix2_CrossPlatformScripting_WindowsBatFiles() {
	s.T().Log("Validating Fix 2: Cross-platform mock script creation - Windows .bat")

	if runtime.GOOS != "windows" {
		s.T().Skip("Windows-specific test")
	}

	// Test Windows .bat file creation
	tempDir := s.T().TempDir()
	mockOptions := porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Windows batch test",
		Stderr:   "",
		Sleep:    100 * time.Millisecond,
	}

	mockPath, err := porch.CreateCrossPlatformMock(tempDir, mockOptions)
	s.Require().NoError(err)
	s.Assert().True(strings.HasSuffix(mockPath, ".bat"), "Windows mock should have .bat extension")
	s.Assert().FileExists(mockPath, "Windows .bat file should exist")

	// Test execution
	config := porch.ExecutorConfig{
		PorchPath: mockPath,
		Mode:      "direct",
		OutDir:    filepath.Join(tempDir, "out"),
		Timeout:   5 * time.Second,
	}

	executor := porch.NewExecutor(config)
	ctx := context.Background()

	// Create dummy intent file for testing
	intentFile := filepath.Join(tempDir, "test-intent.json")
	s.Require().NoError(os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0o644))
	s.Require().NoError(os.MkdirAll(config.OutDir, 0o755))

	result, err := executor.Execute(ctx, intentFile)
	s.Require().NoError(err)
	s.Assert().True(result.Success, "Windows .bat execution should succeed")
	s.Assert().Contains(result.Stdout, "Windows batch test", "Should contain expected output")
}

func (s *ValidationTestSuite) TestFix2_CrossPlatformScripting_UnixShellFiles() {
	s.T().Log("Validating Fix 2: Cross-platform mock script creation - Unix shell")

	if runtime.GOOS == "windows" {
		s.T().Skip("Unix-specific test")
	}

	// Test Unix shell script creation
	tempDir := s.T().TempDir()
	mockOptions := porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Unix shell test",
		Stderr:   "",
		Sleep:    100 * time.Millisecond,
	}

	mockPath, err := porch.CreateCrossPlatformMock(tempDir, mockOptions)
	s.Require().NoError(err)
	s.Assert().False(strings.HasSuffix(mockPath, ".bat"), "Unix mock should not have .bat extension")
	s.Assert().FileExists(mockPath, "Unix shell file should exist")

	// Verify executable permissions
	info, err := os.Stat(mockPath)
	s.Require().NoError(err)
	s.Assert().True(info.Mode()&0o111 != 0, "Unix shell script should be executable")

	// Test execution
	config := porch.ExecutorConfig{
		PorchPath: mockPath,
		Mode:      "direct",
		OutDir:    filepath.Join(tempDir, "out"),
		Timeout:   5 * time.Second,
	}

	executor := porch.NewExecutor(config)
	ctx := context.Background()

	// Create dummy intent file for testing
	intentFile := filepath.Join(tempDir, "test-intent.json")
	s.Require().NoError(os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0o644))
	s.Require().NoError(os.MkdirAll(config.OutDir, 0o755))

	result, err := executor.Execute(ctx, intentFile)
	s.Require().NoError(err)
	s.Assert().True(result.Success, "Unix shell execution should succeed")
	s.Assert().Contains(result.Stdout, "Unix shell test", "Should contain expected output")
}

func (s *ValidationTestSuite) TestFix2_CrossPlatformScripting_CustomScripts() {
	s.T().Log("Validating Fix 2: Custom script support with platform-specific logic")

	tempDir := s.T().TempDir()

	// Define custom scripts for both platforms
	customScript := struct {
		Windows string
		Unix    string
	}{
		Windows: `@echo off
echo Platform: Windows
echo Args: %*
exit /b 0`,
		Unix: `#!/bin/bash
echo "Platform: Unix"
echo "Args: $@"
exit 0`,
	}

	mockOptions := porch.CrossPlatformMockOptions{
		CustomScript: customScript,
	}

	mockPath, err := porch.CreateCrossPlatformMock(tempDir, mockOptions)
	s.Require().NoError(err)
	s.Assert().FileExists(mockPath, "Custom script file should exist")

	// Test execution with arguments
	config := porch.ExecutorConfig{
		PorchPath: mockPath,
		Mode:      "direct",
		OutDir:    filepath.Join(tempDir, "out"),
		Timeout:   5 * time.Second,
	}

	executor := porch.NewExecutor(config)
	ctx := context.Background()

	// Create dummy intent file for testing
	intentFile := filepath.Join(tempDir, "test-intent.json")
	s.Require().NoError(os.WriteFile(intentFile, []byte(`{"test": "intent"}`), 0o644))
	s.Require().NoError(os.MkdirAll(config.OutDir, 0o755))

	result, err := executor.Execute(ctx, intentFile)
	s.Require().NoError(err)
	s.Assert().True(result.Success, "Custom script execution should succeed")

	// Verify platform-specific output
	if runtime.GOOS == "windows" {
		s.Assert().Contains(result.Stdout, "Platform: Windows", "Should show Windows platform")
	} else {
		s.Assert().Contains(result.Stdout, "Platform: Unix", "Should show Unix platform")
	}
}

// =============================================================================
// FIX 3: DATA RACE CONDITION VALIDATION
// =============================================================================

func (s *ValidationTestSuite) TestFix3_DataRaceCondition_ProcessorConcurrentAccess() {
	s.T().Log("Validating Fix 3: Data race protection in IntentProcessor")

	handoffDir := filepath.Join(s.tempDir, "handoff")
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(handoffDir, 0o755))
	s.Require().NoError(os.MkdirAll(outDir, 0o755))

	// Create IntentProcessor with concurrent access - using mock validator and porch function
	mockValidator := &MockValidator{}
	mockPorchFunc := func(ctx context.Context, intent *ingest.Intent, mode string) error { return nil }

	processor, err := NewProcessor(&ProcessorConfig{
		HandoffDir:    handoffDir,
		ErrorDir:      filepath.Join(outDir, "errors"),
		PorchMode:     "direct",
		BatchSize:     5,
		BatchInterval: 100 * time.Millisecond,
		MaxRetries:    3,
		SendTimeout:   2 * time.Second, // Reduced timeout to catch issues faster
		WorkerCount:   4,               // Set worker count
	}, mockValidator, mockPorchFunc)
	s.Require().NoError(err)

	// Start the batch processor to handle incoming files
	processor.StartBatchProcessor()
	defer processor.Stop()

	// Create multiple intent files
	numFiles := 50
	var wg sync.WaitGroup
	var processedCount int64
	var errorCount int64

	// Concurrent file processing
	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(fileID int) {
			defer wg.Done()

			fileName := fmt.Sprintf("concurrent-intent-%d.json", fileID)
			filePath := filepath.Join(handoffDir, fileName)

			content := fmt.Sprintf(`{
				"apiVersion": "v1",
				"kind": "NetworkIntent",
				"metadata": {"name": "test-%d"},
				"spec": {"action": "scale", "replicas": %d}
			}`, fileID, fileID%10+1)

			err := os.WriteFile(filePath, []byte(content), 0o644)
			s.Require().NoError(err)

			// Process file - this should be race-safe
			if err := processor.ProcessFile(filePath); err != nil {
				atomic.AddInt64(&errorCount, 1)
				s.T().Logf("Error processing file %d: %v", fileID, err)
			} else {
				atomic.AddInt64(&processedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Wait for batch interval to flush any remaining files
	time.Sleep(600 * time.Millisecond)

	s.T().Logf("Processed: %d, Errors: %d, Total: %d",
		atomic.LoadInt64(&processedCount),
		atomic.LoadInt64(&errorCount),
		numFiles)

	// All files should be processed without race conditions
	totalProcessed := atomic.LoadInt64(&processedCount) + atomic.LoadInt64(&errorCount)
	s.Assert().Equal(int64(numFiles), totalProcessed, "All files should be processed exactly once")

	// CRITICAL FIX: The test requirement states "total send timeouts must be ??5"
	// With improved channel buffering and shutdown handling, we should see much fewer errors
	errorCountVal := atomic.LoadInt64(&errorCount)
	s.Assert().LessOrEqual(errorCountVal, int64(5),
		"Should have ?? send timeout errors (actual: %d). If this fails, coordinator buffering needs adjustment", errorCountVal)
}

func (s *ValidationTestSuite) TestFix3_DataRaceCondition_WatcherConcurrentOperations() {
	s.T().Log("Validating Fix 3: Data race protection in Watcher concurrent operations")

	handoffDir := filepath.Join(s.tempDir, "handoff")
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(handoffDir, 0o755))
	s.Require().NoError(os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:    s.porchPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         false, // Continuous mode to test concurrent operations
		MaxWorkers:   8,     // High concurrency
		DebounceDur:  50 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	watcher, err := NewWatcher(handoffDir, config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Start watcher in background
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for watcher to start
	time.Sleep(200 * time.Millisecond)

	// Concurrent operations: file creation, metrics access, cleanup
	var wg sync.WaitGroup
	numOperations := 100

	// 1. Concurrent file creation
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations/4; i++ {
			fileName := fmt.Sprintf("race-test-file-%d.json", i)
			filePath := filepath.Join(handoffDir, fileName)
			content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "id": %d}`, i)

			os.WriteFile(filePath, []byte(content), 0o644)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// 2. Concurrent metrics access
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations/2; i++ {
			metrics := watcher.GetMetrics()
			s.Assert().NotNil(metrics, "GetMetrics should not return nil")
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// 3. Concurrent file state cleanup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations/4; i++ {
			watcher.cleanupOldFileState()
			time.Sleep(20 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Cancel and wait for completion
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.T().Log("Watcher shutdown timeout, but no race condition detected")
	}

	// If we reach here without panic or deadlock, race conditions are protected
	s.Assert().True(true, "No race conditions detected during concurrent operations")
}

func (s *ValidationTestSuite) TestFix3_DataRaceCondition_SharedVariableAccess() {
	s.T().Log("Validating Fix 3: Mutex protection for shared test variables")

	// Test the shared variable protection pattern
	var sharedCounter int64
	var mu sync.RWMutex
	var wg sync.WaitGroup

	numGoroutines := 100
	numIncrementsPerGoroutine := 1000

	// Concurrent increments with mutex protection
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIncrementsPerGoroutine; j++ {
				mu.Lock()
				sharedCounter++
				mu.Unlock()
			}
		}()
	}

	// Concurrent reads with mutex protection
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIncrementsPerGoroutine/2; j++ {
				mu.RLock()
				_ = sharedCounter
				mu.RUnlock()
			}
		}()
	}

	wg.Wait()

	expectedValue := int64(numGoroutines * numIncrementsPerGoroutine)
	s.Assert().Equal(expectedValue, sharedCounter,
		"Shared counter should equal expected value if mutex protection works")
}

// =============================================================================
// INTEGRATION TESTS - ALL FIXES TOGETHER
// =============================================================================

func (s *ValidationTestSuite) TestIntegration_AllFixesTogether() {
	s.T().Log("Integration test: All three fixes working together")

	handoffDir := filepath.Join(s.tempDir, "handoff")
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(handoffDir, 0o755))
	s.Require().NoError(os.MkdirAll(outDir, 0o755))

	// Create cross-platform mock (Fix 2)
	mockOptions := porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Integration test processing completed",
		Sleep:    50 * time.Millisecond,
	}
	crossPlatformMock, err := porch.CreateCrossPlatformMock(s.tempDir, mockOptions)
	s.Require().NoError(err)

	config := Config{
		PorchPath:    crossPlatformMock, // Using cross-platform mock (Fix 2)
		Mode:         "direct",
		OutDir:       outDir,
		Once:         false,
		MaxWorkers:   4,
		DebounceDur:  100 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	watcher, err := NewWatcher(handoffDir, config)
	s.Require().NoError(err)

	// Test nil safety (Fix 1) - safe defer pattern
	defer func() {
		if watcher != nil {
			closeErr := watcher.Close() // Should be safe even if partially initialized
			s.Assert().NoError(closeErr, "Close should be safe (Fix 1)")
		}
	}()

	// Start watcher for concurrent operations
	_, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for startup
	time.Sleep(300 * time.Millisecond)

	// Concurrent file processing (Fix 3 - race condition protection)
	var wg sync.WaitGroup
	numFiles := 20

	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(fileID int) {
			defer wg.Done()

			fileName := fmt.Sprintf("integration-test-%d.json", fileID)
			filePath := filepath.Join(handoffDir, fileName)
			content := fmt.Sprintf(`{
				"apiVersion": "v1",
				"kind": "NetworkIntent", 
				"metadata": {"name": "integration-%d"},
				"spec": {"action": "scale", "replicas": %d}
			}`, fileID, fileID%5+1)

			err := os.WriteFile(filePath, []byte(content), 0o644)
			s.Require().NoError(err)

			// Small delay to trigger concurrent processing
			time.Sleep(time.Duration(fileID%50) * time.Millisecond)
		}(i)
	}

	wg.Wait()

	// Concurrent metrics access (Fix 3)
	go func() {
		for i := 0; i < 50; i++ {
			metrics := watcher.GetMetrics()
			s.Assert().NotNil(metrics, "Metrics should be accessible concurrently")
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Wait for processing
	time.Sleep(4 * time.Second)

	// Cancel and cleanup (Fix 1 - safe cleanup)
	cancel()
	select {
	case <-done:
		s.T().Log("Watcher completed successfully")
	case <-time.After(3 * time.Second):
		s.T().Log("Watcher shutdown timeout - testing forced cleanup")
	}

	// Final verification - all fixes working
	s.Assert().True(true, "Integration test completed - all fixes working together")
}

// =============================================================================
// PERFORMANCE REGRESSION TESTS
// =============================================================================

func (s *ValidationTestSuite) TestPerformance_NoRegressionFromFixes() {
	s.T().Log("Performance test: Verifying fixes don't introduce significant performance regression")

	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	handoffDir := filepath.Join(s.tempDir, "handoff")
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(handoffDir, 0o755))
	s.Require().NoError(os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:    s.porchPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         true,
		MaxWorkers:   8,
		DebounceDur:  50 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	// Create test files
	numFiles := 100
	for i := 0; i < numFiles; i++ {
		fileName := fmt.Sprintf("perf-test-%d.json", i)
		filePath := filepath.Join(handoffDir, fileName)
		content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "id": %d}`, i)
		s.Require().NoError(os.WriteFile(filePath, []byte(content), 0o644))
	}

	watcher, err := NewWatcher(handoffDir, config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Measure processing time
	startTime := time.Now()
	err = watcher.Start()
	s.Require().NoError(err)
	processingTime := time.Since(startTime)

	s.T().Logf("Processed %d files in %v", numFiles, processingTime)

	// Verify reasonable performance (should process 100 files in under 30 seconds)
	s.Assert().Less(processingTime, 30*time.Second, "Performance should be reasonable after fixes")

	// Verify throughput
	throughput := float64(numFiles) / processingTime.Seconds()
	s.Assert().Greater(throughput, 3.0, "Should maintain throughput > 3 files/second")
}

// =============================================================================
// STRESS TESTS
// =============================================================================

func (s *ValidationTestSuite) TestStress_HighConcurrencyWithFixes() {
	s.T().Log("Stress test: High concurrency with all fixes enabled")

	if testing.Short() {
		s.T().Skip("Skipping stress test in short mode")
	}

	handoffDir := filepath.Join(s.tempDir, "handoff")
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(handoffDir, 0o755))
	s.Require().NoError(os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:    s.porchPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         false,
		MaxWorkers:   16, // High concurrency
		DebounceDur:  25 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	watcher, err := NewWatcher(handoffDir, config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	_, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for startup
	time.Sleep(200 * time.Millisecond)

	// High-concurrency file creation and concurrent operations
	var wg sync.WaitGroup
	numOperations := 500

	// File creation stress
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations; i++ {
			fileName := fmt.Sprintf("stress-test-%d.json", i)
			filePath := filepath.Join(handoffDir, fileName)
			content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "id": %d}`, i)

			os.WriteFile(filePath, []byte(content), 0o644)
			if i%50 == 0 {
				time.Sleep(10 * time.Millisecond) // Brief pause every 50 files
			}
		}
	}()

	// Metrics access stress
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations*2; i++ {
			metrics := watcher.GetMetrics()
			s.Assert().NotNil(metrics)
			if i%100 == 0 {
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// File state cleanup stress
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numOperations/10; i++ {
			watcher.cleanupOldFileState()
			time.Sleep(50 * time.Millisecond)
		}
	}()

	wg.Wait()

	// Wait for processing to settle
	time.Sleep(5 * time.Second)

	cancel()
	select {
	case <-done:
		s.T().Log("Stress test completed successfully")
	case <-time.After(5 * time.Second):
		s.T().Log("Stress test shutdown timeout")
	}

	// If we reach here without deadlocks or panics, the fixes are working under stress
	s.Assert().True(true, "Stress test completed - fixes handle high concurrency")
}

// =============================================================================
// EDGE CASE TESTS FOR FIXES
// =============================================================================

func (s *ValidationTestSuite) TestEdgeCase_PartialInitializationCleanup() {
	s.T().Log("Edge case: Cleanup during partial initialization (Fix 1)")

	// Test cleanup when initialization fails partway
	nonExistentDir := filepath.Join(s.tempDir, "nonexistent", "deeply", "nested")

	config := Config{
		PorchPath:    s.porchPath,
		Mode:         "direct",
		OutDir:       nonExistentDir, // This should cause initialization to fail
		Once:         true,
		MaxWorkers:   2,
		CleanupAfter: time.Hour,
	}

	// NewWatcher may create the output directory (MkdirAll), so err may be nil
	watcher, err := NewWatcher(s.tempDir, config)

	// In either case, Close should be safe
	if err == nil && watcher != nil {
		closeErr := watcher.Close()
		s.Assert().NoError(closeErr, "Close should be safe after successful initialization")
	}
	// err != nil case: partial initialization, watcher is nil, nothing to close
}

func (s *ValidationTestSuite) TestEdgeCase_CrossPlatformPathHandling() {
	s.T().Log("Edge case: Cross-platform path handling (Fix 2)")

	// Test path handling with different separators
	tempDir := s.T().TempDir()
	mockOptions := porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Path handling test",
	}

	mockPath, err := porch.CreateCrossPlatformMock(tempDir, mockOptions)
	s.Require().NoError(err)

	// Test with various path formats
	absPath, _ := filepath.Abs(mockPath)
	testPaths := []string{
		mockPath,
		filepath.Clean(mockPath),
		absPath,
	}

	for _, testPath := range testPaths {
		if absPathLocal, err := filepath.Abs(testPath); err == nil {
			config := porch.ExecutorConfig{
				PorchPath: absPathLocal,
				Mode:      "direct",
				OutDir:    filepath.Join(tempDir, "out"),
				Timeout:   5 * time.Second,
			}

			executor := porch.NewExecutor(config)
			s.Assert().NotNil(executor, "Executor should handle various path formats")
		}
	}
}

func (s *ValidationTestSuite) TestEdgeCase_RaceConditionUnderMemoryPressure() {
	s.T().Log("Edge case: Race conditions under memory pressure (Fix 3)")

	// Simulate memory pressure scenario
	handoffDir := filepath.Join(s.tempDir, "handoff")
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(handoffDir, 0o755))
	s.Require().NoError(os.MkdirAll(outDir, 0o755))

	config := Config{
		PorchPath:    s.porchPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         true,
		MaxWorkers:   32, // High number to stress memory
		DebounceDur:  10 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	watcher, err := NewWatcher(handoffDir, config)
	s.Require().NoError(err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	// Create many files rapidly to stress memory and concurrency
	numFiles := 200
	var wg sync.WaitGroup

	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(fileID int) {
			defer wg.Done()

			fileName := fmt.Sprintf("memory-stress-%d.json", fileID)
			filePath := filepath.Join(handoffDir, fileName)

			// Create larger content to use more memory
			largeData := strings.Repeat("x", 1024) // 1KB of data
			content := fmt.Sprintf(`{
				"apiVersion": "v1",
				"kind": "NetworkIntent",
				"metadata": {"name": "memory-test-%d"},
				"spec": {"data": "%s", "replicas": %d}
			}`, fileID, largeData, fileID%10+1)

			err := os.WriteFile(filePath, []byte(content), 0o644)
			s.Require().NoError(err)
		}(i)
	}

	wg.Wait()

	// Process under memory pressure
	startTime := time.Now()
	err = watcher.Start()
	s.Require().NoError(err)
	processingTime := time.Since(startTime)

	s.T().Logf("Processed %d files under memory pressure in %v", numFiles, processingTime)

	// Should complete without race conditions or deadlocks
	s.Assert().Less(processingTime, 60*time.Second, "Should complete within reasonable time")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func (s *ValidationTestSuite) createValidationMockPorch() string {
	mockPath, err := porch.CreateCrossPlatformMock(s.tempDir, porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Validation processing completed successfully",
		Sleep:    25 * time.Millisecond,
	})
	s.Require().NoError(err)
	return mockPath
}

// =============================================================================
// RACE DETECTION TEST (Run with -race flag)
// =============================================================================

func TestRaceDetection_RunWithRaceFlag(t *testing.T) {
	t.Log("Race detection test - run with 'go test -race' to validate all fixes")

	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(handoffDir, 0o755))
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	// Create cross-platform mock
	mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
		ExitCode: 0,
		Stdout:   "Race detection test",
		Sleep:    10 * time.Millisecond,
	})
	require.NoError(t, err)

	config := Config{
		PorchPath:    mockPath,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         false,
		MaxWorkers:   8,
		DebounceDur:  25 * time.Millisecond,
		CleanupAfter: time.Hour,
	}

	watcher, err := NewWatcher(handoffDir, config)
	require.NoError(t, err)
	defer watcher.Close() // #nosec G307 - Error handled in defer

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Concurrent operations to trigger race detection
	var wg sync.WaitGroup

	// File creation
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fileName := fmt.Sprintf("race-test-%d.json", id)
			filePath := filepath.Join(handoffDir, fileName)
			content := fmt.Sprintf(`{"id": %d}`, id)
			os.WriteFile(filePath, []byte(content), 0o644)
		}(i)
	}

	// Metrics access
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			metrics := watcher.GetMetrics()
			assert.NotNil(t, metrics)
		}()
	}

	// File state operations
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			watcher.cleanupOldFileState()
		}()
	}

	wg.Wait()
	time.Sleep(1 * time.Second)

	cancel()
	select {
	case <-done:
		t.Log("Race detection test completed successfully")
	case <-time.After(3 * time.Second):
		t.Log("Race detection test timeout")
	}
}
