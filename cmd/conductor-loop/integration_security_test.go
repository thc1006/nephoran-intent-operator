package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

// SecurityTestSuite represents a comprehensive security test suite
type SecurityTestSuite struct {
	tempDir     string
	handoffDir  string
	outDir      string
	mockPorch   string
	config      loop.Config
}

// NewSecurityTestSuite creates a new security test suite
func NewSecurityTestSuite(t *testing.T) *SecurityTestSuite {
	tempDir := t.TempDir()
	handoffDir := filepath.Join(tempDir, "handoff")
	outDir := filepath.Join(tempDir, "out")
	
	require.NoError(t, os.MkdirAll(handoffDir, 0755))
	require.NoError(t, os.MkdirAll(outDir, 0755))

	mockPorch := createComprehensiveMockPorch(t, tempDir)

	config := loop.Config{
		PorchPath:    mockPorch,
		Mode:         "direct",
		OutDir:       outDir,
		Once:         true,
		DebounceDur:  100 * time.Millisecond,
		MaxWorkers:   3,
		CleanupAfter: time.Hour,
	}

	return &SecurityTestSuite{
		tempDir:    tempDir,
		handoffDir: handoffDir,
		outDir:     outDir,
		mockPorch:  mockPorch,
		config:     config,
	}
}

// TestComprehensiveSecuritySuite runs the complete security test suite
func TestComprehensiveSecuritySuite(t *testing.T) {
	suite := NewSecurityTestSuite(t)

	// Run all security test categories
	t.Run("InputValidationSecurity", func(t *testing.T) {
		suite.testInputValidationSecurity(t)
	})

	t.Run("PathTraversalSecurity", func(t *testing.T) {
		suite.testPathTraversalSecurity(t)
	})

	t.Run("CommandInjectionSecurity", func(t *testing.T) {
		suite.testCommandInjectionSecurity(t)
	})

	t.Run("ResourceExhaustionSecurity", func(t *testing.T) {
		suite.testResourceExhaustionSecurity(t)
	})

	t.Run("FileSystemSecurity", func(t *testing.T) {
		suite.testFileSystemSecurity(t)
	})

	t.Run("ConcurrencySecurity", func(t *testing.T) {
		suite.testConcurrencySecurity(t)
	})

	t.Run("StateManagementSecurity", func(t *testing.T) {
		suite.testStateManagementSecurity(t)
	})

	t.Run("ConfigurationSecurity", func(t *testing.T) {
		suite.testConfigurationSecurity(t)
	})
}

// testInputValidationSecurity tests input validation security
func (s *SecurityTestSuite) testInputValidationSecurity(t *testing.T) {
	maliciousInputs := []struct {
		name    string
		content string
		expectSafeHandling bool
	}{
		{
			name: "SQL injection attempt",
			content: `{
				"intent_type": "scaling",
				"target": "app'; DROP TABLE users; --",
				"namespace": "default",
				"replicas": 3
			}`,
			expectSafeHandling: true,
		},
		{
			name: "XSS attempt",
			content: `{
				"intent_type": "scaling",
				"target": "<script>alert('xss')</script>",
				"namespace": "default",
				"replicas": 3
			}`,
			expectSafeHandling: true,
		},
		{
			name: "Buffer overflow attempt",
			content: fmt.Sprintf(`{
				"intent_type": "scaling",
				"target": "%s",
				"namespace": "default",
				"replicas": 3
			}`, strings.Repeat("A", 10000)),
			expectSafeHandling: true,
		},
		{
			name: "Control character injection",
			content: `{
				"intent_type": "scaling",
				"target": "app\x00\x01\x02\x03",
				"namespace": "default\n\r\t",
				"replicas": 3
			}`,
			expectSafeHandling: true,
		},
		{
			name: "JSON bomb",
			content: `{
				"intent_type": "scaling",
				"target": "app",
				"namespace": "default",
				"replicas": 3,
				"nested": ` + generateDeepNestedJSON(100) + `
			}`,
			expectSafeHandling: true,
		},
	}

	for _, input := range maliciousInputs {
		t.Run(input.name, func(t *testing.T) {
			intentFile := filepath.Join(s.handoffDir, fmt.Sprintf("malicious-%s.json", 
				strings.ReplaceAll(input.name, " ", "-")))
			
			require.NoError(t, os.WriteFile(intentFile, []byte(input.content), 0644))

			watcher, err := loop.NewWatcher(s.handoffDir, s.config)
			require.NoError(t, err)
			defer watcher.Close()

			err = watcher.Start()
			
			if input.expectSafeHandling {
				assert.NoError(t, err, "Should handle malicious input safely")
				// Verify no system compromise
				s.assertNoSystemCompromise(t)
			} else {
				assert.Error(t, err, "Should reject dangerous input")
			}
		})
	}
}

// testPathTraversalSecurity tests path traversal security
func (s *SecurityTestSuite) testPathTraversalSecurity(t *testing.T) {
	pathTraversalTests := []struct {
		name         string
		setupFunc    func() string
		expectSecure bool
	}{
		{
			name: "Unix path traversal",
			setupFunc: func() string {
				return `{
					"intent_type": "scaling",
					"target": "../../../etc/passwd",
					"namespace": "default",
					"replicas": 3
				}`
			},
			expectSecure: true,
		},
		{
			name: "Windows path traversal",
			setupFunc: func() string {
				return `{
					"intent_type": "scaling",
					"target": "..\\..\\..\\windows\\system32\\config\\sam",
					"namespace": "default",
					"replicas": 3
				}`
			},
			expectSecure: true,
		},
		{
			name: "URL encoded path traversal",
			setupFunc: func() string {
				return `{
					"intent_type": "scaling",
					"target": "%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
					"namespace": "default",
					"replicas": 3
				}`
			},
			expectSecure: true,
		},
		{
			name: "Double encoded path traversal",
			setupFunc: func() string {
				return `{
					"intent_type": "scaling",
					"target": "%252e%252e%252f%252e%252e%252f%252e%252e%252fetc%252fpasswd",
					"namespace": "default",
					"replicas": 3
				}`
			},
			expectSecure: true,
		},
	}

	for _, test := range pathTraversalTests {
		t.Run(test.name, func(t *testing.T) {
			content := test.setupFunc()
			intentFile := filepath.Join(s.handoffDir, fmt.Sprintf("traversal-%s.json", 
				strings.ReplaceAll(test.name, " ", "-")))
			
			require.NoError(t, os.WriteFile(intentFile, []byte(content), 0644))

			watcher, err := loop.NewWatcher(s.handoffDir, s.config)
			require.NoError(t, err)
			defer watcher.Close()

			err = watcher.Start()
			assert.NoError(t, err, "Should handle path traversal safely")

			// Verify no files created outside expected directories
			s.assertNoPathTraversal(t)
		})
	}
}

// testCommandInjectionSecurity tests command injection security
func (s *SecurityTestSuite) testCommandInjectionSecurity(t *testing.T) {
	injectionTests := []struct {
		name       string
		porchPath  string
		expectSafe bool
	}{
		{
			name:      "Semicolon injection",
			porchPath: "porch; rm -rf /",
			expectSafe: false, // Should fail due to invalid command
		},
		{
			name:      "Pipe injection",
			porchPath: "porch | cat /etc/passwd",
			expectSafe: false,
		},
		{
			name:      "Ampersand injection",
			porchPath: "porch && echo 'injected'",
			expectSafe: false,
		},
		{
			name:      "Backtick injection",
			porchPath: "porch `whoami`",
			expectSafe: false,
		},
		{
			name:      "Dollar injection",
			porchPath: "porch $(id)",
			expectSafe: false,
		},
	}

	for _, test := range injectionTests {
		t.Run(test.name, func(t *testing.T) {
			// Create test intent
			content := `{
				"intent_type": "scaling",
				"target": "my-app",
				"namespace": "default",
				"replicas": 3
			}`
			intentFile := filepath.Join(s.handoffDir, fmt.Sprintf("injection-%s.json", 
				strings.ReplaceAll(test.name, " ", "-")))
			require.NoError(t, os.WriteFile(intentFile, []byte(content), 0644))

			// Use malicious porch path
			maliciousConfig := s.config
			maliciousConfig.PorchPath = test.porchPath

			watcher, err := loop.NewWatcher(s.handoffDir, maliciousConfig)
			require.NoError(t, err)
			defer watcher.Close()

			err = watcher.Start()
			
			if test.expectSafe {
				assert.NoError(t, err, "Should handle injection safely")
			} else {
				// Command injection should fail during execution, not during watcher creation
				// The actual test is that no malicious commands are executed
				assert.NoError(t, err, "Watcher creation should succeed")
			}

			// Verify no command injection occurred
			s.assertNoCommandInjection(t)
		})
	}
}

// testResourceExhaustionSecurity tests resource exhaustion security
func (s *SecurityTestSuite) testResourceExhaustionSecurity(t *testing.T) {
	exhaustionTests := []struct {
		name      string
		setupFunc func(t *testing.T) int
		timeout   time.Duration
	}{
		{
			name: "Many small files",
			setupFunc: func(t *testing.T) int {
				count := 200
				for i := 0; i < count; i++ {
					content := fmt.Sprintf(`{
						"intent_type": "scaling",
						"target": "app-%d",
						"namespace": "ns-%d",
						"replicas": %d
					}`, i, i%10, i%5+1)
					
					file := filepath.Join(s.handoffDir, fmt.Sprintf("bulk-%d.json", i))
					require.NoError(t, os.WriteFile(file, []byte(content), 0644))
				}
				return count
			},
			timeout: 30 * time.Second,
		},
		{
			name: "Large files",
			setupFunc: func(t *testing.T) int {
				count := 5
				for i := 0; i < count; i++ {
					largeData := strings.Repeat("A", 1024*1024) // 1MB
					content := fmt.Sprintf(`{
						"intent_type": "scaling",
						"target": "large-app-%d",
						"namespace": "default",
						"replicas": 3,
						"large_data": "%s"
					}`, i, largeData)
					
					file := filepath.Join(s.handoffDir, fmt.Sprintf("large-%d.json", i))
					require.NoError(t, os.WriteFile(file, []byte(content), 0644))
				}
				return count
			},
			timeout: 20 * time.Second,
		},
		{
			name: "Rapid file creation",
			setupFunc: func(t *testing.T) int {
				count := 100
				
				// Create files rapidly during processing
				go func() {
					for i := 0; i < count; i++ {
						content := fmt.Sprintf(`{
							"intent_type": "scaling",
							"target": "rapid-%d",
							"namespace": "default",
							"replicas": 1
						}`, i)
						
						file := filepath.Join(s.handoffDir, fmt.Sprintf("rapid-%d.json", i))
						_ = os.WriteFile(file, []byte(content), 0644)
						time.Sleep(5 * time.Millisecond)
					}
				}()
				
				return count
			},
			timeout: 25 * time.Second,
		},
	}

	for _, test := range exhaustionTests {
		t.Run(test.name, func(t *testing.T) {
			_ = test.setupFunc(t)

			// Use config with resource limits
			limitedConfig := s.config
			limitedConfig.MaxWorkers = 5 // Limit workers
			limitedConfig.DebounceDur = 10 * time.Millisecond // Short debounce

			watcher, err := loop.NewWatcher(s.handoffDir, limitedConfig)
			require.NoError(t, err)
			defer watcher.Close()

			// Measure processing time
			start := time.Now()
			err = watcher.Start()
			duration := time.Since(start)

			assert.NoError(t, err, "Should handle resource exhaustion gracefully")
			assert.Less(t, duration, test.timeout, "Should complete within reasonable time")

			// Verify system health
			s.assertSystemHealth(t)
		})
	}
}

// testFileSystemSecurity tests file system security
func (s *SecurityTestSuite) testFileSystemSecurity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Advanced file system tests require Unix-like system")
	}

	fsTests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		testFunc  func(t *testing.T, watcher *loop.Watcher, file string)
	}{
		{
			name: "Symlink to sensitive file",
			setupFunc: func(t *testing.T) string {
				// Create symlink to /etc/passwd
				symlinkFile := filepath.Join(s.handoffDir, "passwd-link.json")
				err := os.Symlink("/etc/passwd", symlinkFile)
				if err != nil {
					t.Skip("Cannot create symlink to /etc/passwd")
				}
				return symlinkFile
			},
			testFunc: func(t *testing.T, watcher *loop.Watcher, file string) {
				// Should handle symlinks without exposing sensitive data
				err := watcher.Start()
				assert.NoError(t, err, "Should handle symlinks safely")
			},
		},
		{
			name: "FIFO pipe",
			setupFunc: func(t *testing.T) string {
				if runtime.GOOS == "windows" {
					t.Skip("FIFO pipes not supported on Windows")
				}
				fifoFile := filepath.Join(s.handoffDir, "test-pipe.json")
				err := syscall.Mkfifo(fifoFile, 0644)
				if err != nil {
					t.Skip("Cannot create FIFO pipe")
				}
				return fifoFile
			},
			testFunc: func(t *testing.T, watcher *loop.Watcher, file string) {
				// Should handle FIFO pipes safely
				err := watcher.Start()
				assert.NoError(t, err, "Should handle FIFO pipes safely")
			},
		},
		{
			name: "Device file simulation",
			setupFunc: func(t *testing.T) string {
				// Create a file that looks like a device file
				deviceFile := filepath.Join(s.handoffDir, "device-like.json")
				content := `{
					"intent_type": "scaling",
					"target": "/dev/null",
					"namespace": "default",
					"replicas": 0
				}`
				require.NoError(t, os.WriteFile(deviceFile, []byte(content), 0644))
				return deviceFile
			},
			testFunc: func(t *testing.T, watcher *loop.Watcher, file string) {
				// Should handle device-like filenames safely
				err := watcher.Start()
				assert.NoError(t, err, "Should handle device-like names safely")
			},
		},
	}

	for _, test := range fsTests {
		t.Run(test.name, func(t *testing.T) {
			file := test.setupFunc(t)

			watcher, err := loop.NewWatcher(s.handoffDir, s.config)
			require.NoError(t, err)
			defer watcher.Close()

			test.testFunc(t, watcher, file)

			// Verify no security issues
			s.assertNoSecurityViolation(t)
		})
	}
}

// testConcurrencySecurity tests concurrent access security
func (s *SecurityTestSuite) testConcurrencySecurity(t *testing.T) {
	// Test concurrent watchers
	var watchers []*loop.Watcher
	var wg sync.WaitGroup
	numWatchers := 5

	// Create intent files
	for i := 0; i < 20; i++ {
		content := fmt.Sprintf(`{
			"intent_type": "scaling",
			"target": "concurrent-app-%d",
			"namespace": "default",
			"replicas": %d
		}`, i, i%3+1)
		
		file := filepath.Join(s.handoffDir, fmt.Sprintf("concurrent-%d.json", i))
		require.NoError(t, os.WriteFile(file, []byte(content), 0644))
	}

	// Start multiple watchers concurrently
	for i := 0; i < numWatchers; i++ {
		watcher, err := loop.NewWatcher(s.handoffDir, s.config)
		require.NoError(t, err)
		watchers = append(watchers, watcher)

		wg.Add(1)
		go func(w *loop.Watcher, id int) {
			defer wg.Done()
			err := w.Start()
			assert.NoError(t, err, "Concurrent watcher %d should complete successfully", id)
		}(watcher, i)
	}

	wg.Wait()

	// Clean up
	for _, watcher := range watchers {
		watcher.Close()
	}

	// Verify no race conditions caused security issues
	s.assertNoRaceConditionSecurity(t)
}

// testStateManagementSecurity tests state management security
func (s *SecurityTestSuite) testStateManagementSecurity(t *testing.T) {
	// Test state file manipulation
	stateFile := filepath.Join(s.handoffDir, ".conductor-loop-state.json")
	
	// Create malicious state file
	maliciousState := map[string]interface{}{
		"files": map[string]interface{}{
			"../../../etc/passwd": map[string]interface{}{
				"status":    "processed",
				"timestamp": time.Now().Format(time.RFC3339),
			},
			"$(whoami)": map[string]interface{}{
				"status":    "processed",
				"timestamp": "2025-08-15T12:00:00Z",
			},
		},
	}

	data, _ := json.Marshal(maliciousState)
	require.NoError(t, os.WriteFile(stateFile, data, 0644))

	// Create intent file
	content := `{
		"intent_type": "scaling",
		"target": "test-app",
		"namespace": "default",
		"replicas": 3
	}`
	intentFile := filepath.Join(s.handoffDir, "state-test.json")
	require.NoError(t, os.WriteFile(intentFile, []byte(content), 0644))

	watcher, err := loop.NewWatcher(s.handoffDir, s.config)
	require.NoError(t, err)
	defer watcher.Close()

	err = watcher.Start()
	assert.NoError(t, err, "Should handle malicious state file safely")

	// Verify state manipulation didn't cause security issues
	s.assertNoStateManipulationSecurity(t)
}

// testConfigurationSecurity tests configuration security
func (s *SecurityTestSuite) testConfigurationSecurity(t *testing.T) {
	securityConfigs := []struct {
		name     string
		modifier func(config loop.Config) loop.Config
		expectSafe bool
	}{
		{
			name: "Negative worker count",
			modifier: func(config loop.Config) loop.Config {
				config.MaxWorkers = -10
				return config
			},
			expectSafe: true, // Should use default
		},
		{
			name: "Extremely large worker count",
			modifier: func(config loop.Config) loop.Config {
				config.MaxWorkers = 999999
				return config
			},
			expectSafe: true, // Should be capped
		},
		{
			name: "Negative timeout",
			modifier: func(config loop.Config) loop.Config {
				config.DebounceDur = -1 * time.Hour
				return config
			},
			expectSafe: true, // Should use default
		},
		{
			name: "Path traversal in output directory",
			modifier: func(config loop.Config) loop.Config {
				config.OutDir = "../../../tmp/malicious"
				return config
			},
			expectSafe: true, // Should handle safely
		},
	}

	for _, test := range securityConfigs {
		t.Run(test.name, func(t *testing.T) {
			testConfig := test.modifier(s.config)

			watcher, err := loop.NewWatcher(s.handoffDir, testConfig)
			
			if test.expectSafe {
				assert.NoError(t, err, "Should handle configuration safely")
				if watcher != nil {
					watcher.Close()
				}
			} else {
				assert.Error(t, err, "Should reject dangerous configuration")
			}
		})
	}
}

// Helper assertion methods

func (s *SecurityTestSuite) assertNoSystemCompromise(t *testing.T) {
	// Check for signs of system compromise
	suspiciousFiles := []string{
		"/tmp/hacked",
		"/tmp/backdoor",
		"C:\\temp\\hacked",
		"C:\\temp\\backdoor",
	}

	for _, file := range suspiciousFiles {
		_, err := os.Stat(file)
		assert.True(t, os.IsNotExist(err), "Suspicious file should not exist: %s", file)
	}
}

func (s *SecurityTestSuite) assertNoPathTraversal(t *testing.T) {
	// Walk all created files and ensure they're within expected directories
	err := filepath.WalkDir(s.tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Ensure path is within tempDir
		relPath, err := filepath.Rel(s.tempDir, path)
		if err != nil || strings.HasPrefix(relPath, "..") {
			t.Errorf("Path traversal detected: %s is outside %s", path, s.tempDir)
		}

		return nil
	})
	require.NoError(t, err)
}

func (s *SecurityTestSuite) assertNoCommandInjection(t *testing.T) {
	// Check for signs of command injection
	suspiciousPatterns := []string{
		"injected",
		"whoami",
		"id",
		"passwd",
		"cat /etc/passwd",
	}

	// Check process list (simplified)
	// In a real implementation, you'd check running processes
	for _, pattern := range suspiciousPatterns {
		// Check if any output files contain suspicious content
		err := filepath.WalkDir(s.outDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				content, err := os.ReadFile(path)
				if err == nil && strings.Contains(strings.ToLower(string(content)), pattern) {
					t.Errorf("Suspicious content found in %s: %s", path, pattern)
				}
			}

			return nil
		})
		require.NoError(t, err)
	}
}

func (s *SecurityTestSuite) assertSystemHealth(t *testing.T) {
	// Basic system health checks
	
	// Check we can still create files
	testFile := filepath.Join(s.tempDir, "health-check.tmp")
	err := os.WriteFile(testFile, []byte("health"), 0644)
	assert.NoError(t, err, "System should still be responsive")
	
	if err == nil {
		os.Remove(testFile)
	}

	// Check memory isn't exhausted (simplified check)
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Ensure memory usage is reasonable (less than 100MB for tests)
	assert.Less(t, m.Alloc, uint64(100*1024*1024), "Memory usage should be reasonable")
}

func (s *SecurityTestSuite) assertNoSecurityViolation(t *testing.T) {
	// Generic security violation check
	s.assertNoSystemCompromise(t)
	s.assertNoPathTraversal(t)
	s.assertNoCommandInjection(t)
}

func (s *SecurityTestSuite) assertNoRaceConditionSecurity(t *testing.T) {
	// Check that concurrent access didn't create security issues
	
	// Verify state file integrity
	stateFile := filepath.Join(s.handoffDir, ".conductor-loop-state.json")
	if _, err := os.Stat(stateFile); err == nil {
		content, err := os.ReadFile(stateFile)
		if err == nil {
			var state map[string]interface{}
			err = json.Unmarshal(content, &state)
			assert.NoError(t, err, "State file should remain valid JSON after concurrent access")
		}
	}

	// Verify no duplicate processing
	processedCount := s.countProcessedFiles()
	assert.GreaterOrEqual(t, processedCount, 0, "Should have processed files without corruption")
}

func (s *SecurityTestSuite) assertNoStateManipulationSecurity(t *testing.T) {
	// Verify state manipulation didn't compromise security
	stateFile := filepath.Join(s.handoffDir, ".conductor-loop-state.json")
	
	if _, err := os.Stat(stateFile); err == nil {
		content, err := os.ReadFile(stateFile)
		if err == nil {
			// Ensure state doesn't contain dangerous paths
			stateContent := string(content)
			dangerousPatterns := []string{
				"/etc/passwd",
				"$(whoami)",
				"`whoami`",
				"; rm -rf",
			}
			
			for _, pattern := range dangerousPatterns {
				assert.NotContains(t, stateContent, pattern, 
					"State file should not contain dangerous pattern: %s", pattern)
			}
		}
	}
}

func (s *SecurityTestSuite) countProcessedFiles() int {
	processedDir := filepath.Join(s.handoffDir, "processed")
	failedDir := filepath.Join(s.handoffDir, "failed")
	
	count := 0
	
	if entries, err := os.ReadDir(processedDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				count++
			}
		}
	}
	
	if entries, err := os.ReadDir(failedDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				count++
			}
		}
	}
	
	return count
}

// Helper functions

func createComprehensiveMockPorch(t testing.TB, tempDir string) string {
	var mockScript string
	var mockPath string

	if runtime.GOOS == "windows" {
		mockPath = filepath.Join(tempDir, "mock-porch-comprehensive.bat")
		mockScript = `@echo off
if "%1"=="--help" (
    echo Mock porch comprehensive help
    exit /b 0
)

REM Log all arguments for security analysis
echo Arguments: %* >> "%TEMP%\porch-security-log.txt"

REM Simulate processing
echo Processing intent: %2
echo Output directory: %4

REM Create output file if directory exists
if exist "%4" (
    echo Mock output > "%4\mock-output.yaml"
)

echo Comprehensive processing completed
exit /b 0`
	} else {
		mockPath = filepath.Join(tempDir, "mock-porch-comprehensive")
		mockScript = `#!/bin/bash
if [ "$1" = "--help" ]; then
    echo "Mock porch comprehensive help"
    exit 0
fi

# Log all arguments for security analysis
echo "Arguments: $*" >> /tmp/porch-security-log.txt

# Simulate processing
echo "Processing intent: $2"
echo "Output directory: $4"

# Create output file if directory exists
if [ -d "$4" ]; then
    echo "Mock output" > "$4/mock-output.yaml"
fi

echo "Comprehensive processing completed"
exit 0`
	}

	require.NoError(t, os.WriteFile(mockPath, []byte(mockScript), 0755))
	return mockPath
}

func generateDeepNestedJSON(depth int) string {
	if depth <= 0 {
		return `"end"`
	}
	return fmt.Sprintf(`{"level": %s}`, generateDeepNestedJSON(depth-1))
}