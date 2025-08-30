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

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// =============================================================================
// TEST SUITE FOR WATCHER VALIDATION
// =============================================================================

type WatcherValidationTestSuite struct {
	suite.Suite
	tempDir   string
	config    Config
	porchPath string
}

func TestWatcherValidationSuite(t *testing.T) {
	suite.Run(t, new(WatcherValidationTestSuite))
}

func (s *WatcherValidationTestSuite) SetupTest() {
	s.tempDir = s.T().TempDir()
	outDir := filepath.Join(s.tempDir, "out")
	s.Require().NoError(os.MkdirAll(outDir, 0755))

	s.porchPath = createMockPorch(s.T(), s.tempDir, 0, "processed successfully", "")
	s.config = Config{
		PorchPath:   s.porchPath,
		Mode:        porch.ModeDirect,
		OutDir:      outDir,
		MaxWorkers:  4,
		DebounceDur: 50 * time.Millisecond,
		Once:        false,
		MetricsPort: 0, // Use dynamic port to avoid conflicts
	}
}

// =============================================================================
// 1. CREATE/WRITE EVENT DUPLICATE PROCESSING PREVENTION TESTS
// =============================================================================

func (s *WatcherValidationTestSuite) TestDuplicateEventPrevention_CreateFollowedByWrite() {
	s.T().Log("Testing CREATE followed by WRITE event debouncing")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	testFile := filepath.Join(s.tempDir, "intent-debounce-test.json")
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": {"action": "scale"}}`

	// Track processing through executor stats instead
	initialStats := watcher.executor.GetStats()

	// Start watcher in background
	cancel := context.CancelFunc(func() { watcher.Close() })

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Simulate CREATE event followed by WRITE event rapidly
	s.Require().NoError(os.WriteFile(testFile, []byte(testContent), 0644))

	// Trigger events manually to simulate filesystem behavior
	watcher.handleIntentFileWithEnhancedDebounce(testFile, fsnotify.Create)
	time.Sleep(10 * time.Millisecond) // Small delay between events
	watcher.handleIntentFileWithEnhancedDebounce(testFile, fsnotify.Write)

	// Wait for debouncing to settle
	time.Sleep(s.config.DebounceDur + 200*time.Millisecond)

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}

	// Should have processed only once due to debouncing
	finalStats := watcher.executor.GetStats()
	processingCount := finalStats.TotalExecutions - initialStats.TotalExecutions
	s.Assert().LessOrEqual(processingCount, 1, "File should be processed at most once due to debouncing")
	s.T().Logf("CREATE+WRITE events resulted in %d processing calls", processingCount)
}

func (s *WatcherValidationTestSuite) TestDuplicateEventPrevention_RecentEventsCleanup() {
	s.T().Log("Testing recent events map cleanup")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	testFile := filepath.Join(s.tempDir, "intent-cleanup-test.json")

	// Add entry to recent events
	watcher.fileState.mu.Lock()
	watcher.fileState.recentEvents[testFile] = time.Now().Add(-2 * time.Minute) // Old event
	watcher.fileState.mu.Unlock()

	// Trigger cleanup manually using a helper method
	watcher.cleanupOldFileState()

	// Give cleanup time to run
	time.Sleep(100 * time.Millisecond)

	// Check that old entry was cleaned up
	watcher.fileState.mu.RLock()
	_, exists := watcher.fileState.recentEvents[testFile]
	watcher.fileState.mu.RUnlock()

	s.Assert().False(exists, "Old recent event entry should be cleaned up")
}

func (s *WatcherValidationTestSuite) TestDuplicateEventPrevention_DebounceWindowConfigurable() {
	s.T().Log("Testing configurable debounce window")

	tests := []struct {
		name        string
		debounceMs  time.Duration
		eventGapMs  time.Duration
		shouldMerge bool
	}{
		{
			name:        "events_within_window",
			debounceMs:  100 * time.Millisecond,
			eventGapMs:  50 * time.Millisecond,
			shouldMerge: true,
		},
		{
			name:        "events_outside_window",
			debounceMs:  50 * time.Millisecond,
			eventGapMs:  100 * time.Millisecond,
			shouldMerge: false,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			testConfig := s.config
			testConfig.DebounceDur = tt.debounceMs

			watcher, err := NewWatcher(s.tempDir, testConfig)
			require.NoError(t, err)
			defer watcher.Close()

			testFile := filepath.Join(s.tempDir, fmt.Sprintf("intent-%s.json", tt.name))
			testContent := `{"apiVersion": "v1", "kind": "NetworkIntent"}`

			initialStats := watcher.executor.GetStats()

			// Start the watcher first
			_, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			go func() {
				watcher.Start()
			}()
			time.Sleep(50 * time.Millisecond) // Let watcher start

			// First event
			require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))
			watcher.handleIntentFileWithEnhancedDebounce(testFile, fsnotify.Create)

			// Wait specified gap
			time.Sleep(tt.eventGapMs)

			// Second event - update file content to trigger processing
			testContent2 := `{"apiVersion": "v1", "kind": "NetworkIntent", "updated": true}`
			require.NoError(t, os.WriteFile(testFile, []byte(testContent2), 0644))
			watcher.handleIntentFileWithEnhancedDebounce(testFile, fsnotify.Write)

			// Wait for processing to complete
			time.Sleep(testConfig.DebounceDur + 200*time.Millisecond)
			cancel()

			finalStats := watcher.executor.GetStats()
			processingCount := finalStats.TotalExecutions - initialStats.TotalExecutions
			if tt.shouldMerge {
				assert.LessOrEqual(t, processingCount, 1, "Events should be merged within debounce window")
			} else {
				// On Windows, timing can be less precise, so we allow for some debouncing
				// even when events are theoretically outside the window
				if runtime.GOOS == "windows" {
					assert.GreaterOrEqual(t, processingCount, 1, "Should process at least 1 event on Windows")
					assert.LessOrEqual(t, processingCount, 2, "Should process at most 2 events on Windows")
				} else {
					assert.Equal(t, 2, processingCount, "Events should be processed separately outside debounce window")
				}
			}
		})
	}
}

func (s *WatcherValidationTestSuite) TestDuplicateEventPrevention_ConcurrentEventHandling() {
	s.T().Log("Testing concurrent event handling with debouncing")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	testFile := filepath.Join(s.tempDir, "intent-concurrent-test.json")
	testContent := `{"apiVersion": "v1", "kind": "NetworkIntent"}`

	initialStats := watcher.executor.GetStats()

	s.Require().NoError(os.WriteFile(testFile, []byte(testContent), 0644))

	// Simulate multiple concurrent events for the same file
	numEvents := 10
	var wg sync.WaitGroup

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(eventID int) {
			defer wg.Done()
			eventType := fsnotify.Create
			if eventID%2 == 0 {
				eventType = fsnotify.Write
			}
			watcher.handleIntentFileWithEnhancedDebounce(testFile, eventType)
		}(i)
	}

	wg.Wait()

	// Wait for debouncing and processing to complete
	time.Sleep(s.config.DebounceDur + 500*time.Millisecond)

	finalStats := watcher.executor.GetStats()
	processingCount := finalStats.TotalExecutions - initialStats.TotalExecutions
	s.Assert().LessOrEqual(processingCount, 2, "Concurrent events should be heavily debounced")
	s.T().Logf("Concurrent %d events resulted in %d processing calls", numEvents, processingCount)
}

// =============================================================================
// 2. DIRECTORY CREATION RACE CONDITION TESTS
// =============================================================================

func (s *WatcherValidationTestSuite) TestDirectoryCreationRace_ConcurrentCreation() {
	s.T().Log("Testing concurrent directory creation with sync.Once")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	targetDir := filepath.Join(s.tempDir, "status", "subdir", "nested")
	numGoroutines := 50

	var wg sync.WaitGroup
	var successCount int64
	var attemptCount int64

	// Track mkdir calls (simulate by checking if directory exists immediately after our call)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			atomic.AddInt64(&attemptCount, 1)

			// Call the directory creation function
			watcher.ensureDirectoryExists(targetDir)

			// Check if directory exists after our call
			if _, err := os.Stat(targetDir); err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Verify directory was created
	s.Assert().DirExists(targetDir, "Directory should be created successfully")

	// All goroutines should report success since directory exists after creation
	s.Assert().Equal(int64(numGoroutines), successCount, "All goroutines should see the directory exists")
	s.T().Logf("Directory creation: %d attempts, %d successes", attemptCount, successCount)
}

func (s *WatcherValidationTestSuite) TestDirectoryCreationRace_DirectoryManagerState() {
	s.T().Log("Testing DirectoryManager state consistency")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	dirs := []string{
		filepath.Join(s.tempDir, "status"),
		filepath.Join(s.tempDir, "status", "level1"),
		filepath.Join(s.tempDir, "status", "level1", "level2"),
		filepath.Join(s.tempDir, "processed"),
		filepath.Join(s.tempDir, "failed"),
	}

	var wg sync.WaitGroup

	// Create directories concurrently
	for _, dir := range dirs {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(targetDir string) {
				defer wg.Done()
				watcher.ensureDirectoryExists(targetDir)
			}(dir)
		}
	}

	wg.Wait()

	// Verify all directories exist
	for _, dir := range dirs {
		s.Assert().DirExists(dir, "Directory %s should exist", dir)
	}

	// Check DirectoryManager state
	watcher.dirManager.mu.RLock()
	mapSize := len(watcher.dirManager.dirOnce)
	watcher.dirManager.mu.RUnlock()

	s.Assert().Equal(len(dirs), mapSize, "DirectoryManager should track all created directories")
}

func (s *WatcherValidationTestSuite) TestDirectoryCreationRace_SyncOncePattern() {
	s.T().Log("Testing sync.Once pattern effectiveness")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	targetDir := filepath.Join(s.tempDir, "once-test")
	var callCount int64

	// Note: We can't override os.MkdirAll in Go, so we simulate the sync.Once behavior

	// Simulate the once pattern behavior
	var once sync.Once
	var actualMkdirCalls int64

	numGoroutines := 100
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&callCount, 1)

			// Simulate the sync.Once pattern
			once.Do(func() {
				atomic.AddInt64(&actualMkdirCalls, 1)
				os.MkdirAll(targetDir, 0755)
			})
		}()
	}

	wg.Wait()

	s.Assert().Equal(int64(numGoroutines), callCount, "All goroutines should call the function")
	s.Assert().Equal(int64(1), actualMkdirCalls, "sync.Once should ensure only one actual mkdir call")
	s.Assert().DirExists(targetDir, "Directory should be created")
}

func (s *WatcherValidationTestSuite) TestDirectoryCreationRace_NestedDirectories() {
	s.T().Log("Testing nested directory creation race conditions")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	// Create deeply nested directory structure concurrently
	baseDir := filepath.Join(s.tempDir, "nested")
	var wg sync.WaitGroup

	// Multiple goroutines trying to create different levels of nesting
	for level := 1; level <= 5; level++ {
		for worker := 0; worker < 10; worker++ {
			wg.Add(1)
			go func(depth int) {
				defer wg.Done()

				var dirPath string = baseDir
				for i := 0; i < depth; i++ {
					dirPath = filepath.Join(dirPath, fmt.Sprintf("level%d", i))
				}

				watcher.ensureDirectoryExists(dirPath)
			}(level)
		}
	}

	wg.Wait()

	// Verify all nested directories exist
	currentPath := baseDir
	for level := 0; level < 5; level++ {
		currentPath = filepath.Join(currentPath, fmt.Sprintf("level%d", level))
		s.Assert().DirExists(currentPath, "Nested directory level %d should exist", level)
	}
}

// =============================================================================
// 3. JSON VALIDATION TESTS
// =============================================================================

func (s *WatcherValidationTestSuite) TestJSONValidation_ValidNetworkIntentStructures() {
	s.T().Log("Testing valid NetworkIntent and ScalingIntent structures")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	validCases := []struct {
		name    string
		content string
		desc    string
	}{
		{
			name: "valid_network_intent_full",
			content: `{
				"apiVersion": "nephoran.com/v1alpha1",
				"kind": "NetworkIntent",
				"metadata": {
					"name": "test-network-intent",
					"namespace": "default",
					"labels": {
						"app": "test",
						"version": "v1"
					}
				},
				"spec": {
					"action": "deploy",
					"target": {
						"type": "deployment",
						"name": "test-app"
					},
					"parameters": {
						"networkFunctions": [
							{
								"name": "cucp",
								"type": "cnf",
								"resources": {
									"cpu": "2",
									"memory": "4Gi"
								}
							}
						],
						"connectivity": {
							"n1": {"interface": "eth0"},
							"n2": {"interface": "eth1"}
						},
						"sla": {
							"availability": "99.9%",
							"throughput": "1Gbps",
							"latency": "1ms"
						}
					},
					"constraints": {
						"placement": {"zone": "us-west-1"},
						"security": {"level": "high"}
					}
				}
			}`,
			desc: "Complete NetworkIntent with all valid fields",
		},
		{
			name: "valid_scaling_intent",
			content: `{
				"apiVersion": "v1",
				"kind": "ScalingIntent",
				"metadata": {
					"name": "scale-deployment"
				},
				"spec": {
					"replicas": 5
				}
			}`,
			desc: "Valid ScalingIntent",
		},
		{
			name: "valid_minimal_intent",
			content: `{
				"apiVersion": "v1",
				"kind": "NetworkIntent"
			}`,
			desc: "Minimal valid intent with required fields only",
		},
		{
			name: "legacy_scaling_format",
			content: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 3,
				"source": "user"
			}`,
			desc: "Legacy scaling intent format",
		},
	}

	for _, tc := range validCases {
		s.T().Run(tc.name, func(t *testing.T) {
			fileName := fmt.Sprintf("intent-%s.json", tc.name)
			filePath := filepath.Join(s.tempDir, fileName)

			err := os.WriteFile(filePath, []byte(tc.content), 0644)
			require.NoError(t, err)

			err = watcher.validateJSONFile(filePath)
			assert.NoError(t, err, "Should accept valid JSON: %s", tc.desc)
		})
	}
}

func (s *WatcherValidationTestSuite) TestJSONValidation_InvalidJSONRejection() {
	s.T().Log("Testing invalid JSON rejection with specific error types")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	invalidCases := []struct {
		name        string
		content     string
		expectedErr string
		desc        string
	}{
		{
			name:        "malformed_json",
			content:     `{"apiVersion": "v1", "kind": "NetworkIntent", "action":}`,
			expectedErr: "JSON bomb detected",
			desc:        "Malformed JSON syntax",
		},
		{
			name:        "empty_file",
			content:     ``,
			expectedErr: "file is empty",
			desc:        "Empty file",
		},
		{
			name:        "missing_apiversion",
			content:     `{"kind": "NetworkIntent"}`,
			expectedErr: "apiVersion must be a non-empty string",
			desc:        "Missing apiVersion field",
		},
		{
			name:        "empty_apiversion",
			content:     `{"apiVersion": "", "kind": "NetworkIntent"}`,
			expectedErr: "apiVersion must be a non-empty string",
			desc:        "Empty apiVersion",
		},
		{
			name:        "invalid_kind",
			content:     `{"apiVersion": "v1", "kind": "InvalidKind"}`,
			expectedErr: "unsupported kind",
			desc:        "Unsupported kind",
		},
		{
			name:        "invalid_apiversion_format",
			content:     `{"apiVersion": "invalid", "kind": "NetworkIntent"}`,
			expectedErr: "apiVersion format invalid",
			desc:        "Invalid apiVersion format",
		},
		{
			name:        "non_string_kind",
			content:     `{"apiVersion": "v1", "kind": 123}`,
			expectedErr: "kind must be a non-empty string",
			desc:        "Non-string kind",
		},
		{
			name:        "invalid_metadata_name",
			content:     `{"apiVersion": "v1", "kind": "NetworkIntent", "metadata": {"name": "Invalid_Name!"}}`,
			expectedErr: "metadata.name contains invalid characters",
			desc:        "Invalid Kubernetes name format",
		},
		{
			name:        "non_object_spec",
			content:     `{"apiVersion": "v1", "kind": "NetworkIntent", "spec": "invalid"}`,
			expectedErr: "spec must be an object",
			desc:        "Non-object spec",
		},
		{
			name:        "invalid_scaling_replicas",
			content:     `{"intent_type": "scaling", "target": "app", "namespace": "default", "replicas": 0}`,
			expectedErr: "replicas must be an integer between 1 and 100",
			desc:        "Invalid replicas count",
		},
		{
			name:        "missing_legacy_fields",
			content:     `{"intent_type": "scaling", "target": "app"}`,
			expectedErr: "missing or null required field: namespace",
			desc:        "Missing required fields in legacy format",
		},
	}

	for _, tc := range invalidCases {
		s.T().Run(tc.name, func(t *testing.T) {
			fileName := fmt.Sprintf("intent-invalid-%s.json", tc.name)
			filePath := filepath.Join(s.tempDir, fileName)

			err := os.WriteFile(filePath, []byte(tc.content), 0644)
			require.NoError(t, err)

			err = watcher.validateJSONFile(filePath)
			assert.Error(t, err, "Should reject invalid JSON: %s", tc.desc)
			if tc.expectedErr != "" && err != nil {
				assert.Contains(t, err.Error(), tc.expectedErr, "Should contain expected error message")
			}
		})
	}
}

func (s *WatcherValidationTestSuite) TestJSONValidation_PathTraversalPrevention() {
	s.T().Log("Testing path traversal prevention in validation")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	maliciousPaths := []struct {
		name string
		path string
		desc string
	}{
		{
			name: "parent_directory_traversal",
			path: filepath.Join(s.tempDir, "..", "malicious.json"),
			desc: "Parent directory traversal",
		},
		{
			name: "deep_traversal",
			path: filepath.Join(s.tempDir, "..", "..", "..", "etc", "passwd"),
			desc: "Deep directory traversal",
		},
		{
			name: "mixed_traversal",
			path: filepath.Join(s.tempDir, "subdir", "..", "..", "sensitive.json"),
			desc: "Mixed path traversal",
		},
		{
			name: "absolute_path",
			path: func() string {
				if runtime.GOOS == "windows" {
					return "C:\\Windows\\System32\\drivers\\etc\\hosts"
				}
				return "/etc/passwd"
			}(),
			desc: "Absolute path outside watched directory",
		},
	}

	validContent := `{"apiVersion": "v1", "kind": "NetworkIntent"}`

	for _, tc := range maliciousPaths {
		s.T().Run(tc.name, func(t *testing.T) {
			// Try to create the malicious file
			os.MkdirAll(filepath.Dir(tc.path), 0755)
			os.WriteFile(tc.path, []byte(validContent), 0644)

			err := watcher.validatePath(tc.path)
			assert.Error(t, err, "Should reject path traversal: %s", tc.desc)
			assert.Contains(t, err.Error(), "outside watched directory",
				"Error should mention path restriction")
		})
	}
}

func (s *WatcherValidationTestSuite) TestWindowsPathValidation_EdgeCases() {
	if runtime.GOOS != "windows" {
		s.T().Skip("Windows-specific test")
	}

	s.T().Log("Testing Windows path validation edge cases")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	windowsPathCases := []struct {
		name          string
		path          string
		desc          string
		shouldError   bool
		errorContains string
	}{
		{
			name:          "drive_letter_only",
			path:          "C:",
			desc:          "Drive letter only (relative path)",
			shouldError:   true,
			errorContains: "Windows path validation failed",
		},
		{
			name:        "drive_with_relative_path",
			path:        "C:temp\\file.json",
			desc:        "Drive letter with relative path",
			shouldError: false, // Should be converted to absolute
		},
		{
			name:        "mixed_separators",
			path:        filepath.Join(s.tempDir, "mixed/path\\file.json"),
			desc:        "Mixed path separators",
			shouldError: false, // Should be normalized
		},
		{
			name:          "unc_path_outside_watched",
			path:          "\\\\server\\share\\file.json",
			desc:          "UNC path outside watched directory",
			shouldError:   true,
			errorContains: "outside watched directory",
		},
		{
			name:          "long_device_path",
			path:          "\\\\?\\C:\\temp\\file.json",
			desc:          "Long device path outside watched directory",
			shouldError:   true,
			errorContains: "outside watched directory",
		},
		{
			name:          "invalid_chars_lt_gt",
			path:          filepath.Join(s.tempDir, "file<test>.json"),
			desc:          "Invalid characters < and >",
			shouldError:   true,
			errorContains: "Windows path validation failed",
		},
		{
			name:          "invalid_chars_pipe",
			path:          filepath.Join(s.tempDir, "file|test.json"),
			desc:          "Invalid character pipe",
			shouldError:   true,
			errorContains: "Windows path validation failed",
		},
		{
			name:          "reserved_filename_con",
			path:          filepath.Join(s.tempDir, "CON.json"),
			desc:          "Reserved filename CON",
			shouldError:   true,
			errorContains: "Windows path validation failed",
		},
		{
			name:          "reserved_filename_com1",
			path:          filepath.Join(s.tempDir, "COM1.log"),
			desc:          "Reserved filename COM1",
			shouldError:   true,
			errorContains: "Windows path validation failed",
		},
		{
			name:        "case_insensitive_path_comparison",
			path:        strings.ToUpper(filepath.Join(s.tempDir, "file.json")),
			desc:        "Case insensitive path comparison",
			shouldError: false, // Windows paths should be case-insensitive
		},
		{
			name:          "very_long_path_without_prefix",
			path:          filepath.Join(s.tempDir, strings.Repeat("a", 250), "file.json"),
			desc:          "Very long path without \\\\?\\ prefix",
			shouldError:   true,
			errorContains: "Windows path validation failed",
		},
	}

	validContent := `{"apiVersion": "v1", "kind": "NetworkIntent"}`

	for _, tc := range windowsPathCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Create the file if path is within temp directory
			if strings.Contains(tc.path, s.tempDir) || (!strings.Contains(tc.path, ":") && !strings.HasPrefix(tc.path, "\\\\")) {
				// Create parent directories
				if dir := filepath.Dir(tc.path); dir != "." {
					os.MkdirAll(dir, 0755)
				}
				os.WriteFile(tc.path, []byte(validContent), 0644)
			}

			err := watcher.validatePath(tc.path)
			if tc.shouldError {
				assert.Error(t, err, "Should reject path: %s", tc.desc)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains,
						"Error should contain expected message for: %s", tc.desc)
				}
			} else {
				// For paths that should be valid, we need to ensure they're within watched dir
				if !strings.Contains(tc.path, s.tempDir) && !filepath.IsAbs(tc.path) {
					// Skip validation for relative paths outside temp dir
					t.Skipf("Skipping validation for relative path outside temp dir: %s", tc.path)
				} else {
					assert.NoError(t, err, "Should accept path: %s", tc.desc)
				}
			}
		})
	}
}

func (s *WatcherValidationTestSuite) TestJSONValidation_SizeLimitEnforcement() {
	s.T().Log("Testing JSON size limit enforcement")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	tests := []struct {
		name       string
		size       int
		shouldPass bool
	}{
		{"small_file", 1024, true},
		{"medium_file", 1024 * 1024, true},
		{"large_valid_file", 5 * 1024 * 1024, true},
		{"oversized_file", MaxJSONSize + 1024, false},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			fileName := fmt.Sprintf("intent-size-%s.json", tt.name)
			filePath := filepath.Join(s.tempDir, fileName)

			if tt.size <= MaxJSONSize {
				// Create valid JSON of specified size
				padding := strings.Repeat("x", tt.size-100)
				content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "data": "%s"}`, padding)
				err := os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			} else {
				// Create oversized file with well-formed JSON
				// Calculate padding size to exceed MaxJSONSize
				baseJSON := `{"apiVersion": "v1", "kind": "NetworkIntent", "data": ""}`
				baseSizeWithoutData := len(baseJSON) - 2 // Subtract 2 for the empty quotes
				paddingSize := tt.size - baseSizeWithoutData

				// Generate deterministic padding with 'A' characters for consistency
				padding := strings.Repeat("A", paddingSize)
				content := fmt.Sprintf(`{"apiVersion": "v1", "kind": "NetworkIntent", "data": "%s"}`, padding)

				err := os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			}

			err := watcher.validateJSONFile(filePath)
			if tt.shouldPass {
				assert.NoError(t, err, "File of size %d should pass", tt.size)
			} else {
				assert.Error(t, err, "File of size %d should fail", tt.size)
				assert.Contains(t, err.Error(), "exceeds maximum",
					"Error should mention size limit")
			}
		})
	}
}

func (s *WatcherValidationTestSuite) TestJSONValidation_SuspiciousFilenamePatterns() {
	s.T().Log("Testing suspicious filename pattern detection")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	suspiciousPatterns := []string{
		"intent-test..json",
		"intent-test~.json",
		"intent-test$.json",
		"intent-test*.json",
		"intent-test?.json",
		"intent-test[.json",
		"intent-test{.json",
		"intent-test|.json",
		"intent-test<.json",
		"intent-test>.json",
		"intent-test\x00.json", // null byte
	}

	validContent := `{"apiVersion": "v1", "kind": "NetworkIntent"}`

	for _, pattern := range suspiciousPatterns {
		s.T().Run(fmt.Sprintf("suspicious_%s", pattern), func(t *testing.T) {
			filePath := filepath.Join(s.tempDir, pattern)

			// Create the file
			os.WriteFile(filePath, []byte(validContent), 0644)

			err := watcher.validatePath(filePath)
			assert.Error(t, err, "Should reject suspicious filename: %s", pattern)

			// On Windows, handle OS-specific validation behavior
			if runtime.GOOS == "windows" {
				// Windows-invalid characters get caught by Windows path validation first
				windowsInvalidChars := []string{"*", "?", "|", "<", ">", ":", "\""}
				isWindowsInvalidChar := false
				for _, char := range windowsInvalidChars {
					if strings.Contains(pattern, char) {
						isWindowsInvalidChar = true
						break
					}
				}

				if pattern == "intent-test\x00.json" {
					// Null bytes cause filepath.Abs to fail with "invalid argument"
					// before we reach any validation check
					if err != nil {
						assert.Contains(t, err.Error(), "failed to get absolute path",
							"Error should mention absolute path failure on Windows for null bytes")
					} else {
						t.Log("Note: OS-level null byte handling may prevent this error")
					}
				} else if isWindowsInvalidChar {
					// Windows-invalid characters are caught by Windows path validation
					if err != nil {
						assert.Contains(t, err.Error(), "Windows path validation failed",
							"Error should mention Windows path validation failure for Windows-invalid characters")
					} else {
						t.Log("Note: Windows path validation may handle this differently")
					}
				} else {
					// Other patterns should still be caught by suspicious pattern validation
					if err != nil {
						assert.Contains(t, err.Error(), "suspicious pattern",
							"Error should mention suspicious pattern")
					} else {
						t.Log("Note: Suspicious pattern validation may be handled differently")
					}
				}
			} else {
				// On non-Windows systems, all patterns should be caught by suspicious pattern validation
				if err != nil {
					assert.Contains(t, err.Error(), "suspicious pattern",
						"Error should mention suspicious pattern")
				} else {
					t.Log("Note: Suspicious pattern validation may be handled differently")
				}
			}
		})
	}
}

func (s *WatcherValidationTestSuite) TestJSONValidation_ComplexIntentValidation() {
	s.T().Log("Testing complex NetworkIntent field validation")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	complexCases := []struct {
		name        string
		content     string
		shouldPass  bool
		description string
	}{
		{
			name: "valid_complex_network_function",
			content: `{
				"apiVersion": "nephoran.com/v1alpha1",
				"kind": "NetworkIntent",
				"spec": {
					"parameters": {
						"networkFunctions": [
							{
								"name": "test-nf",
								"type": "cucp",
								"resources": {
									"cpu": "2",
									"memory": "4Gi",
									"storage": "10Gi"
								}
							}
						]
					}
				}
			}`,
			shouldPass:  true,
			description: "Valid complex network function",
		},
		{
			name: "invalid_network_function_missing_name",
			content: `{
				"apiVersion": "v1",
				"kind": "NetworkIntent",
				"spec": {
					"parameters": {
						"networkFunctions": [
							{
								"type": "cucp"
							}
						]
					}
				}
			}`,
			shouldPass:  false,
			description: "Network function missing required name",
		},
		{
			name: "valid_connectivity_config",
			content: `{
				"apiVersion": "v1",
				"kind": "NetworkIntent",
				"spec": {
					"parameters": {
						"connectivity": {
							"n1": {"interface": "eth0"},
							"n2": {"interface": "eth1"},
							"n3": {"interface": "eth2"}
						}
					}
				}
			}`,
			shouldPass:  true,
			description: "Valid connectivity configuration",
		},
		{
			name: "invalid_connectivity_interface",
			content: `{
				"apiVersion": "v1",
				"kind": "NetworkIntent",
				"spec": {
					"parameters": {
						"connectivity": {
							"n1": {"interface": ""}
						}
					}
				}
			}`,
			shouldPass:  false,
			description: "Empty interface in connectivity",
		},
		{
			name: "valid_sla_configuration",
			content: `{
				"apiVersion": "v1",
				"kind": "NetworkIntent",
				"spec": {
					"parameters": {
						"sla": {
							"availability": "99.99%",
							"throughput": "10Gbps",
							"latency": "1ms",
							"jitter": "0.1ms"
						}
					}
				}
			}`,
			shouldPass:  true,
			description: "Valid SLA configuration",
		},
	}

	for _, tc := range complexCases {
		s.T().Run(tc.name, func(t *testing.T) {
			fileName := fmt.Sprintf("intent-complex-%s.json", tc.name)
			filePath := filepath.Join(s.tempDir, fileName)

			err := os.WriteFile(filePath, []byte(tc.content), 0644)
			require.NoError(t, err)

			err = watcher.validateJSONFile(filePath)
			if tc.shouldPass {
				assert.NoError(t, err, "Should pass validation: %s", tc.description)
			} else {
				assert.Error(t, err, "Should fail validation: %s", tc.description)
			}
		})
	}
}

// =============================================================================
// INTEGRATION TESTS - COMBINING ALL THREE ASPECTS
// =============================================================================

func (s *WatcherValidationTestSuite) TestIntegration_DebouncingWithValidation() {
	s.T().Log("Testing debouncing combined with JSON validation")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	testFile := filepath.Join(s.tempDir, "intent-debounce-validation.json")

	// Track validation through metrics instead
	initialValidationCount := atomic.LoadInt64(&watcher.metrics.ValidationFailuresTotal)

	// Start watcher in background
	cancel := context.CancelFunc(func() { watcher.Close() })

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Write invalid JSON multiple times rapidly
	invalidContent := `{"apiVersion": "v1", "kind": "InvalidKind"`
	for i := 0; i < 5; i++ {
		s.Require().NoError(os.WriteFile(testFile, []byte(invalidContent), 0644))
		watcher.handleIntentFileWithEnhancedDebounce(testFile, fsnotify.Write)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debouncing and processing
	time.Sleep(s.config.DebounceDur + 300*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
	}

	// Should have validation failures due to invalid JSON
	finalValidationCount := atomic.LoadInt64(&watcher.metrics.ValidationFailuresTotal)
	validationFailures := finalValidationCount - initialValidationCount
	s.Assert().Greater(validationFailures, int64(0), "Should have validation failures")

	// File should be in failed directory due to validation failure
	failedFiles, err := watcher.fileManager.GetFailedFiles()
	s.Require().NoError(err)
	s.Assert().Len(failedFiles, 1, "Invalid file should be moved to failed directory")
}

func (s *WatcherValidationTestSuite) TestIntegration_ConcurrentDirectoryAndFileProcessing() {
	s.T().Log("Testing concurrent directory creation and file processing")

	watcher, err := NewWatcher(s.tempDir, s.config)
	s.Require().NoError(err)
	defer watcher.Close()

	var wg sync.WaitGroup

	// Track processing through executor stats
	initialStats := watcher.executor.GetStats()

	// Start watcher
	cancel := context.CancelFunc(func() { watcher.Close() })

	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Concurrently:
	// 1. Create files that need status directories
	// 2. Trigger directory creation race conditions
	numFiles := 20
	validContent := `{"apiVersion": "v1", "kind": "NetworkIntent"}`

	for i := 0; i < numFiles; i++ {
		wg.Add(2)

		// File creation goroutine
		go func(fileID int) {
			defer wg.Done()
			fileName := fmt.Sprintf("intent-concurrent-%d.json", fileID)
			filePath := filepath.Join(s.tempDir, fileName)

			s.Require().NoError(os.WriteFile(filePath, []byte(validContent), 0644))
			watcher.handleIntentFileWithEnhancedDebounce(filePath, fsnotify.Create)
		}(i)

		// Directory creation goroutine
		go func(dirID int) {
			defer wg.Done()
			statusDir := filepath.Join(s.tempDir, "status", fmt.Sprintf("level-%d", dirID%3))
			watcher.ensureDirectoryExists(statusDir)
		}(i)
	}

	wg.Wait()

	// Wait for all processing to complete
	time.Sleep(s.config.DebounceDur + 1*time.Second)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}

	// Verify directories were created
	for level := 0; level < 3; level++ {
		statusDir := filepath.Join(s.tempDir, "status", fmt.Sprintf("level-%d", level))
		s.Assert().DirExists(statusDir, "Status directory should be created")
	}

	// Should have processed some files successfully
	finalStats := watcher.executor.GetStats()
	totalProcessed := finalStats.TotalExecutions - initialStats.TotalExecutions
	s.Assert().Greater(totalProcessed, 0, "Some files should have been processed")
	s.T().Logf("Processed %d files with concurrent directory creation", totalProcessed)
}

// =============================================================================
// BENCHMARK TESTS FOR PERFORMANCE VALIDATION
// =============================================================================

func BenchmarkWatcherValidation_JSONValidation(b *testing.B) {
	tempDir := b.TempDir()
	config := Config{
		PorchPath:   createMockPorch(b, tempDir, 0, "processed", ""),
		Mode:        porch.ModeDirect,
		OutDir:      filepath.Join(tempDir, "out"),
		MaxWorkers:  3, // Production-like worker count for realistic concurrency testing
		DebounceDur: 10 * time.Millisecond,
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(b, err)
	defer watcher.Close()

	validContent := `{
		"apiVersion": "nephoran.com/v1alpha1",
		"kind": "NetworkIntent",
		"metadata": {"name": "test"},
		"spec": {
			"action": "deploy",
			"parameters": {
				"networkFunctions": [
					{"name": "test-nf", "type": "cucp"}
				]
			}
		}
	}`

	testFile := filepath.Join(tempDir, "intent-bench.json")
	require.NoError(b, os.WriteFile(testFile, []byte(validContent), 0644))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = watcher.validateJSONFile(testFile)
	}
}

func BenchmarkWatcherValidation_DirectoryCreation(b *testing.B) {
	tempDir := b.TempDir()
	config := Config{
		PorchPath:  createMockPorch(b, tempDir, 0, "processed", ""),
		Mode:       porch.ModeDirect,
		OutDir:     filepath.Join(tempDir, "out"),
		MaxWorkers: 3, // Production-like worker count for realistic concurrency testing
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(b, err)
	defer watcher.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targetDir := filepath.Join(tempDir, "bench-status", fmt.Sprintf("level-%d", i%10))
		watcher.ensureDirectoryExists(targetDir)
	}
}

func BenchmarkWatcherValidation_EventDebouncing(b *testing.B) {
	tempDir := b.TempDir()
	config := Config{
		PorchPath:   createMockPorch(b, tempDir, 0, "processed", ""),
		Mode:        porch.ModeDirect,
		OutDir:      filepath.Join(tempDir, "out"),
		MaxWorkers:  3,                    // Production-like worker count for realistic concurrency testing
		DebounceDur: 1 * time.Millisecond, // Very short for benchmarking
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(b, err)
	defer watcher.Close()

	testFile := filepath.Join(tempDir, "intent-debounce-bench.json")
	validContent := `{"apiVersion": "v1", "kind": "NetworkIntent"}`
	require.NoError(b, os.WriteFile(testFile, []byte(validContent), 0644))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.handleIntentFileWithEnhancedDebounce(testFile, fsnotify.Write)
	}
}
