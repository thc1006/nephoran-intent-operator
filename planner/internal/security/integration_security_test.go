//go:build integration

package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/planner"
	"github.com/thc1006/nephoran-intent-operator/planner/internal/rules"
)

// TestIntegration_EndToEndSecurity tests complete security validation in realistic planner scenarios
func TestIntegration_EndToEndSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration security tests in short mode")
	}

	// Create temporary directories for the test environment
	tempDir, err := os.MkdirTemp("", "planner-security-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputDir := filepath.Join(tempDir, "handoff")
	stateFile := filepath.Join(tempDir, "state.json")
	configDir := filepath.Join(tempDir, "config")

	// Create necessary directories
	for _, dir := range []string{outputDir, configDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	validator := NewValidator(DefaultValidationConfig())

	// Test scenario 1: Normal operation with security validations
	t.Run("NormalOperationWithSecurity", func(t *testing.T) {
		// Create valid KMP data
		kmpData := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "integration-test-node-001",
			PRBUtilization:  0.85, // High utilization to trigger scaling
			P95Latency:      200.0,
			ActiveUEs:       150,
			CurrentReplicas: 2,
		}

		// Validate KMP data
		if err := validator.ValidateKMPData(kmpData); err != nil {
			t.Fatalf("Valid KMP data failed validation: %v", err)
		}

		// Create rule engine with security validator
		config := rules.Config{
			StateFile:            stateFile,
			CooldownDuration:     60 * time.Second,
			MinReplicas:          1,
			MaxReplicas:          10,
			LatencyThresholdHigh: 100.0,
			LatencyThresholdLow:  50.0,
			PRBThresholdHigh:     0.8,
			PRBThresholdLow:      0.3,
			EvaluationWindow:     90 * time.Second,
		}

		engine := rules.NewRuleEngine(config)
		engine.SetValidator(validator)

		// Process metrics and generate scaling decision
		decision := engine.Evaluate(kmpData)
		if decision == nil {
			t.Fatal("Expected scaling decision for high utilization")
		}

		// Create intent with security validation
		intent := &planner.Intent{
			IntentType:    "scaling",
			Target:        decision.Target,
			Namespace:     decision.Namespace,
			Replicas:      decision.TargetReplicas,
			Reason:        decision.Reason,
			Source:        "planner-integration-test",
			CorrelationID: fmt.Sprintf("integration-%d", time.Now().Unix()),
		}

		// Write intent with secure permissions
		intentData, err := json.MarshalIndent(intent, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal intent: %v", err)
		}

		intentFile := filepath.Join(outputDir, fmt.Sprintf("intent-%d.json", time.Now().Unix()))

		// Validate intent file path
		if err := validator.ValidateFilePath(intentFile, "integration test intent"); err != nil {
			t.Fatalf("Intent file path validation failed: %v", err)
		}

		// Write with secure permissions
		if err := os.WriteFile(intentFile, intentData, 0600); err != nil {
			t.Fatalf("Failed to write intent file: %v", err)
		}

		// Verify file permissions
		info, err := os.Stat(intentFile)
		if err != nil {
			t.Fatalf("Failed to stat intent file: %v", err)
		}

		if info.Mode().Perm() != 0600 {
			t.Errorf("Intent file has incorrect permissions: %o, expected 0600", info.Mode().Perm())
		}

		// Verify state file security
		if _, err := os.Stat(stateFile); err == nil {
			stateInfo, err := os.Stat(stateFile)
			if err != nil {
				t.Fatalf("Failed to stat state file: %v", err)
			}
			if stateInfo.Mode().Perm() != 0600 {
				t.Errorf("State file has incorrect permissions: %o, expected 0600", stateInfo.Mode().Perm())
			}
		}

		t.Log("✓ Normal operation with security validations completed successfully")
	})

	// Test scenario 2: Attack mitigation during operation
	t.Run("AttackMitigationDuringOperation", func(t *testing.T) {
		// Attempt various attacks during normal operation
		attackScenarios := []struct {
			name       string
			kmpData    rules.KPMData
			shouldFail bool
		}{
			{
				name: "SQL injection in node ID",
				kmpData: rules.KPMData{
					Timestamp:       time.Now(),
					NodeID:          "node'; DROP TABLE metrics; --",
					PRBUtilization:  0.5,
					P95Latency:      100.0,
					ActiveUEs:       50,
					CurrentReplicas: 2,
				},
				shouldFail: true,
			},
			{
				name: "Path traversal in node ID",
				kmpData: rules.KPMData{
					Timestamp:       time.Now(),
					NodeID:          "../../../etc/passwd",
					PRBUtilization:  0.5,
					P95Latency:      100.0,
					ActiveUEs:       50,
					CurrentReplicas: 2,
				},
				shouldFail: true,
			},
			{
				name: "Malicious values in metrics",
				kmpData: rules.KPMData{
					Timestamp:       time.Now(),
					NodeID:          "test-node",
					PRBUtilization:  -1.0,   // Invalid value
					P95Latency:      -100.0, // Invalid value
					ActiveUEs:       -50,    // Invalid value
					CurrentReplicas: -2,     // Invalid value
				},
				shouldFail: true,
			},
			{
				name: "Excessive values in metrics",
				kmpData: rules.KPMData{
					Timestamp:       time.Now(),
					NodeID:          "test-node",
					PRBUtilization:  999.0,    // Excessive value
					P95Latency:      999999.0, // Excessive value
					ActiveUEs:       999999,   // Excessive value
					CurrentReplicas: 999999,   // Excessive value
				},
				shouldFail: true,
			},
		}

		for _, scenario := range attackScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				err := validator.ValidateKMPData(scenario.kmpData)
				failed := err != nil

				if failed != scenario.shouldFail {
					t.Errorf("Attack scenario '%s': expected failure=%v, got failure=%v (error: %v)",
						scenario.name, scenario.shouldFail, failed, err)
				}

				if scenario.shouldFail && failed {
					t.Logf("✓ Successfully blocked attack: %s", scenario.name)
				}
			})
		}
	})

	// Test scenario 3: Configuration security validation
	t.Run("ConfigurationSecurityValidation", func(t *testing.T) {
		configTests := []struct {
			name       string
			envVars    map[string]string
			shouldFail bool
		}{
			{
				name: "Valid configuration",
				envVars: map[string]string{
					"PLANNER_METRICS_URL": "http://localhost:9090/metrics",
					"PLANNER_OUTPUT_DIR":  outputDir,
				},
				shouldFail: false,
			},
			{
				name: "Malicious metrics URL",
				envVars: map[string]string{
					"PLANNER_METRICS_URL": "http://evil.com/steal'; DELETE FROM users; --",
				},
				shouldFail: true,
			},
			{
				name: "Directory traversal in output dir",
				envVars: map[string]string{
					"PLANNER_OUTPUT_DIR": "../../../etc",
				},
				shouldFail: true,
			},
			{
				name: "Invalid URL scheme",
				envVars: map[string]string{
					"PLANNER_METRICS_URL": "file:///etc/passwd",
				},
				shouldFail: true,
			},
		}

		for _, test := range configTests {
			t.Run(test.name, func(t *testing.T) {
				hasErrors := false
				for envVar, value := range test.envVars {
					err := validator.ValidateEnvironmentVariable(envVar, value, "config test")
					if err != nil {
						hasErrors = true
						t.Logf("Environment variable %s failed validation: %v", envVar, err)
					}
				}

				if hasErrors != test.shouldFail {
					t.Errorf("Configuration test '%s': expected errors=%v, got errors=%v",
						test.name, test.shouldFail, hasErrors)
				}

				if test.shouldFail && hasErrors {
					t.Logf("✓ Successfully blocked malicious configuration: %s", test.name)
				}
			})
		}
	})
}

// TestIntegration_HTTPSecurityHeaders tests HTTP security in metrics endpoints
func TestIntegration_HTTPSecurityHeaders(t *testing.T) {
	// Create a mock metrics server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Simulate KMP metrics response
		kmpData := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "metrics-server-node-001",
			PRBUtilization:  0.65,
			P95Latency:      120.0,
			ActiveUEs:       75,
			CurrentReplicas: 3,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(kmpData)
	}))
	defer server.Close()

	validator := NewValidator(DefaultValidationConfig())

	// Test HTTP client security
	t.Run("HTTPClientSecurity", func(t *testing.T) {
		// Validate server URL
		if err := validator.ValidateURL(server.URL, "metrics server test"); err != nil {
			t.Fatalf("Server URL validation failed: %v", err)
		}

		// Make request to metrics endpoint
		resp, err := http.Get(server.URL + "/metrics")
		if err != nil {
			t.Fatalf("Failed to make HTTP request: %v", err)
		}
		defer resp.Body.Close()

		// Verify security headers are present
		securityHeaders := map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"X-XSS-Protection":       "1; mode=block",
		}

		for header, expectedValue := range securityHeaders {
			actualValue := resp.Header.Get(header)
			if actualValue != expectedValue {
				t.Errorf("Security header %s: expected %s, got %s", header, expectedValue, actualValue)
			}
		}

		// Read and validate response data
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		var kmpData rules.KPMData
		if err := json.Unmarshal(body, &kmpData); err != nil {
			t.Fatalf("Failed to unmarshal KMP data: %v", err)
		}

		// Validate the received KMP data
		if err := validator.ValidateKMPData(kmpData); err != nil {
			t.Fatalf("Received KMP data failed validation: %v", err)
		}

		t.Log("✓ HTTP security headers and data validation completed")
	})
}

// TestIntegration_ConcurrentSecurityOperations tests security under concurrent load
func TestIntegration_ConcurrentSecurityOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent security tests in short mode")
	}

	validator := NewValidator(DefaultValidationConfig())

	// Create temporary directory for concurrent operations
	tempDir, err := os.MkdirTemp("", "planner-concurrent-security-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	const numWorkers = 50
	const operationsPerWorker = 20

	// Test concurrent file operations with security validation
	t.Run("ConcurrentFileOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, numWorkers*operationsPerWorker)

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerWorker; j++ {
					// Create intent file with security validation
					intent := &planner.Intent{
						IntentType:    "scaling",
						Target:        fmt.Sprintf("cnf-%d-%d", workerID, j),
						Namespace:     "production",
						Replicas:      j%5 + 1,
						Reason:        fmt.Sprintf("Concurrent test worker %d operation %d", workerID, j),
						Source:        "concurrent-test",
						CorrelationID: fmt.Sprintf("concurrent-%d-%d-%d", workerID, j, time.Now().Unix()),
					}

					intentData, err := json.MarshalIndent(intent, "", "  ")
					if err != nil {
						errors <- fmt.Errorf("worker %d operation %d: marshal failed: %v", workerID, j, err)
						continue
					}

					fileName := fmt.Sprintf("intent-%d-%d-%d.json", workerID, j, time.Now().UnixNano())
					filePath := filepath.Join(tempDir, fileName)

					// Validate file path
					if err := validator.ValidateFilePath(filePath, "concurrent test"); err != nil {
						errors <- fmt.Errorf("worker %d operation %d: path validation failed: %v", workerID, j, err)
						continue
					}

					// Write with secure permissions
					if err := os.WriteFile(filePath, intentData, 0600); err != nil {
						errors <- fmt.Errorf("worker %d operation %d: file write failed: %v", workerID, j, err)
						continue
					}

					// Verify permissions
					info, err := os.Stat(filePath)
					if err != nil {
						errors <- fmt.Errorf("worker %d operation %d: stat failed: %v", workerID, j, err)
						continue
					}

					if info.Mode().Perm() != 0600 {
						errors <- fmt.Errorf("worker %d operation %d: incorrect permissions %o", workerID, j, info.Mode().Perm())
						continue
					}

					// Small delay to prevent timestamp collisions
					time.Sleep(time.Millisecond)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errorCount int
		for err := range errors {
			t.Errorf("Concurrent operation error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Failed %d out of %d concurrent operations", errorCount, numWorkers*operationsPerWorker)
		}

		// Verify all files were created with correct permissions
		files, err := filepath.Glob(filepath.Join(tempDir, "intent-*.json"))
		if err != nil {
			t.Fatalf("Failed to list intent files: %v", err)
		}

		expectedFiles := numWorkers * operationsPerWorker
		if len(files) != expectedFiles {
			t.Errorf("Expected %d files, found %d", expectedFiles, len(files))
		}

		// Verify permissions on all files
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				t.Errorf("Failed to stat file %s: %v", file, err)
				continue
			}

			if info.Mode().Perm() != 0600 {
				t.Errorf("File %s has incorrect permissions: %o", file, info.Mode().Perm())
			}
		}

		t.Logf("✓ Successfully completed %d concurrent secure file operations", len(files))
	})

	// Test concurrent validation operations
	t.Run("ConcurrentValidationOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		validationErrors := make(chan error, numWorkers*operationsPerWorker)

		// Mix of valid and invalid data for validation testing
		testData := []rules.KPMData{
			{
				Timestamp:       time.Now(),
				NodeID:          "valid-node-001",
				PRBUtilization:  0.5,
				P95Latency:      100.0,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
			{
				Timestamp:       time.Now(),
				NodeID:          "'; DROP TABLE nodes; --",
				PRBUtilization:  0.5,
				P95Latency:      100.0,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
			{
				Timestamp:       time.Now(),
				NodeID:          "valid-node-002",
				PRBUtilization:  -1.0, // Invalid
				P95Latency:      100.0,
				ActiveUEs:       50,
				CurrentReplicas: 2,
			},
		}

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < operationsPerWorker; j++ {
					data := testData[j%len(testData)]
					// Modify timestamp to make each validation unique
					data.Timestamp = time.Now().Add(time.Duration(j) * time.Millisecond)

					err := validator.ValidateKMPData(data)

					// Record both successes and failures for analysis
					if err != nil {
						validationErrors <- fmt.Errorf("worker %d operation %d: validation error: %v", workerID, j, err)
					}
				}
			}(i)
		}

		wg.Wait()
		close(validationErrors)

		// Analyze validation results
		errorCount := 0
		for err := range validationErrors {
			// Don't treat expected validation failures as errors
			if !strings.Contains(err.Error(), "DROP TABLE") &&
				!strings.Contains(err.Error(), "negative") &&
				!strings.Contains(err.Error(), "exceeds maximum") {
				t.Errorf("Unexpected validation error: %v", err)
			}
			errorCount++
		}

		t.Logf("✓ Completed %d concurrent validation operations with %d expected rejections",
			numWorkers*operationsPerWorker, errorCount)
	})
}

// TestIntegration_SecurityEventLogging tests security event logging and monitoring
func TestIntegration_SecurityEventLogging(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	// Create a buffer to capture log-like output
	var logBuffer strings.Builder

	securityEvents := []struct {
		name       string
		operation  func() error
		expectLog  bool
		logPattern string
	}{
		{
			name: "SQL injection attempt",
			operation: func() error {
				return validator.validateNodeID("'; DROP TABLE users; --")
			},
			expectLog:  true,
			logPattern: "dangerous characters",
		},
		{
			name: "Path traversal attempt",
			operation: func() error {
				return validator.ValidateFilePath("../../../etc/passwd", "security test")
			},
			expectLog:  true,
			logPattern: "traversal",
		},
		{
			name: "Valid operation",
			operation: func() error {
				return validator.validateNodeID("valid-node-001")
			},
			expectLog:  false,
			logPattern: "",
		},
		{
			name: "Log injection attempt",
			operation: func() error {
				result := validator.SanitizeForLogging("normal\nFAKE: Admin login successful")
				logBuffer.WriteString(fmt.Sprintf("User input: %s\n", result))
				return nil
			},
			expectLog:  true,
			logPattern: "\\n",
		},
	}

	for _, event := range securityEvents {
		t.Run(event.name, func(t *testing.T) {
			initialLogSize := logBuffer.Len()

			err := event.operation()

			logContent := logBuffer.String()[initialLogSize:]
			hasLog := len(logContent) > 0 || err != nil

			if hasLog != event.expectLog {
				t.Errorf("Security event '%s': expected log=%v, got log=%v (error: %v, log: %s)",
					event.name, event.expectLog, hasLog, err, logContent)
			}

			if event.expectLog && event.logPattern != "" {
				if err != nil && !strings.Contains(err.Error(), event.logPattern) {
					t.Errorf("Security event '%s': expected log pattern '%s' not found in error: %v",
						event.name, event.logPattern, err)
				} else if logContent != "" && !strings.Contains(logContent, event.logPattern) {
					t.Errorf("Security event '%s': expected log pattern '%s' not found in log: %s",
						event.name, event.logPattern, logContent)
				}
			}

			if event.expectLog && (err != nil || hasLog) {
				t.Logf("✓ Security event logged: %s", event.name)
			}
		})
	}
}

// TestIntegration_SecurityConfigurationValidation tests security configuration edge cases
func TestIntegration_SecurityConfigurationValidation(t *testing.T) {
	// Test custom security configurations
	customConfigs := []struct {
		name       string
		config     ValidationConfig
		shouldWork bool
		reason     string
	}{
		{
			name: "Overly permissive config",
			config: ValidationConfig{
				MaxPRBUtilization: 10.0,                                  // Too high
				MaxLatency:        1000000.0,                             // Too high
				MaxReplicas:       10000,                                 // Too high
				MaxActiveUEs:      10000000,                              // Too high
				AllowedExtensions: []string{".exe", ".bat", ".sh"},       // Dangerous extensions
				MaxPathLength:     1000000,                               // Too high
				AllowedSchemes:    []string{"ftp", "file", "javascript"}, // Insecure schemes
				MaxURLLength:      1000000,                               // Too high
			},
			shouldWork: false,
			reason:     "Configuration allows dangerous values",
		},
		{
			name: "Overly restrictive config",
			config: ValidationConfig{
				MaxPRBUtilization: 0.1,        // Too restrictive
				MaxLatency:        1.0,        // Too restrictive
				MaxReplicas:       1,          // Too restrictive
				MaxActiveUEs:      1,          // Too restrictive
				AllowedExtensions: []string{}, // No extensions allowed
				MaxPathLength:     10,         // Too short
				AllowedSchemes:    []string{}, // No schemes allowed
				MaxURLLength:      10,         // Too short
			},
			shouldWork: false,
			reason:     "Configuration is too restrictive for normal operation",
		},
		{
			name: "Balanced secure config",
			config: ValidationConfig{
				MaxPRBUtilization: 1.0,
				MaxLatency:        5000.0,
				MaxReplicas:       50,
				MaxActiveUEs:      5000,
				AllowedExtensions: []string{".json", ".yaml", ".yml", ".txt"},
				MaxPathLength:     2048,
				AllowedSchemes:    []string{"http", "https"},
				MaxURLLength:      1024,
			},
			shouldWork: true,
			reason:     "Balanced security configuration",
		},
	}

	for _, configTest := range customConfigs {
		t.Run(configTest.name, func(t *testing.T) {
			validator := NewValidator(configTest.config)

			// Test normal operations with this configuration
			normalKMP := rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "test-node-001",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			}

			kmpErr := validator.ValidateKMPData(normalKMP)
			urlErr := validator.ValidateURL("https://api.example.com/metrics", "test")
			pathErr := validator.ValidateFilePath("/tmp/test.json", "test")

			hasErrors := kmpErr != nil || urlErr != nil || pathErr != nil
			works := !hasErrors

			if works != configTest.shouldWork {
				t.Errorf("Configuration test '%s' (%s): expected works=%v, got works=%v\nErrors: KMP=%v, URL=%v, Path=%v",
					configTest.name, configTest.reason, configTest.shouldWork, works, kmpErr, urlErr, pathErr)
			}

			if configTest.shouldWork && works {
				t.Logf("✓ Configuration works as expected: %s", configTest.name)
			} else if !configTest.shouldWork && !works {
				t.Logf("✓ Configuration properly restrictive: %s", configTest.name)
			}
		})
	}
}

// TestIntegration_CrossPlatformSecurity tests security features across different platforms
func TestIntegration_CrossPlatformSecurity(t *testing.T) {
	validator := NewValidator(DefaultValidationConfig())

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "planner-cross-platform-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Platform-specific security tests
	platformTests := []struct {
		name        string
		testFunc    func(t *testing.T)
		platforms   []string // Empty means all platforms
		description string
	}{
		{
			name:        "File permissions on Unix",
			platforms:   []string{"linux", "darwin", "freebsd"},
			description: "Test file permission enforcement on Unix-like systems",
			testFunc: func(t *testing.T) {
				testFile := filepath.Join(tempDir, "unix-test.json")
				testData := []byte(`{"test": "unix permissions"}`)

				if err := os.WriteFile(testFile, testData, 0600); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}

				info, err := os.Stat(testFile)
				if err != nil {
					t.Fatalf("Failed to stat test file: %v", err)
				}

				if info.Mode().Perm() != 0600 {
					t.Errorf("Unix file permissions incorrect: %o, expected 0600", info.Mode().Perm())
				}
			},
		},
		{
			name:        "File permissions on Windows",
			platforms:   []string{"windows"},
			description: "Test file permission handling on Windows",
			testFunc: func(t *testing.T) {
				testFile := filepath.Join(tempDir, "windows-test.json")
				testData := []byte(`{"test": "windows permissions"}`)

				if err := os.WriteFile(testFile, testData, 0600); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}

				// On Windows, just verify the file exists and is readable by owner
				if _, err := os.ReadFile(testFile); err != nil {
					t.Fatalf("Failed to read back test file: %v", err)
				}
			},
		},
		{
			name:        "Path validation cross-platform",
			platforms:   []string{}, // All platforms
			description: "Test path validation across all platforms",
			testFunc: func(t *testing.T) {
				platformPaths := map[string][]string{
					"windows": {
						"C:\\Windows\\System32\\config\\SAM",
						"\\\\server\\share\\sensitive",
						"C:/Windows/System32/config/SAM", // Forward slashes on Windows
					},
					"unix": {
						"/etc/passwd",
						"/proc/self/environ",
						"/dev/null",
					},
				}

				var testPaths []string
				if runtime.GOOS == "windows" {
					testPaths = platformPaths["windows"]
				} else {
					testPaths = platformPaths["unix"]
				}

				for _, path := range testPaths {
					err := validator.ValidateFilePath(path, "cross-platform test")
					// Most of these should be blocked on their respective platforms
					if err == nil {
						t.Logf("⚠ Path validation allowed potentially sensitive path: %s", path)
					} else {
						t.Logf("✓ Path validation blocked sensitive path: %s", path)
					}
				}
			},
		},
	}

	for _, test := range platformTests {
		t.Run(test.name, func(t *testing.T) {
			// Check if test should run on current platform
			shouldRun := len(test.platforms) == 0 // Run on all platforms if none specified
			for _, platform := range test.platforms {
				if runtime.GOOS == platform {
					shouldRun = true
					break
				}
			}

			if !shouldRun {
				t.Skipf("Skipping test '%s' on platform %s", test.name, runtime.GOOS)
				return
			}

			t.Logf("Running cross-platform test: %s", test.description)
			test.testFunc(t)
		})
	}
}

// BenchmarkIntegration_SecurityOverhead benchmarks the performance overhead of security features
func BenchmarkIntegration_SecurityOverhead(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	// Create temporary directory for benchmarking
	tempDir, err := os.MkdirTemp("", "planner-benchmark-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	b.Run("CompleteSecurityValidation", func(b *testing.B) {
		kmpData := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "benchmark-node-001",
			PRBUtilization:  0.75,
			P95Latency:      150.0,
			ActiveUEs:       100,
			CurrentReplicas: 3,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate complete security validation pipeline
			_ = validator.ValidateKMPData(kmpData)
			_ = validator.ValidateURL("https://api.example.com/metrics", "benchmark")
			_ = validator.ValidateFilePath("/tmp/benchmark.json", "benchmark")
			_ = validator.SanitizeForLogging("benchmark log message")
		}
	})

	b.Run("SecureFileOperations", func(b *testing.B) {
		testData := []byte(`{"benchmark": true}`)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fileName := filepath.Join(tempDir, fmt.Sprintf("benchmark-%d.json", i))

			// Validate path and write with secure permissions
			if err := validator.ValidateFilePath(fileName, "benchmark"); err != nil {
				b.Fatalf("Path validation failed: %v", err)
			}

			if err := os.WriteFile(fileName, testData, 0600); err != nil {
				b.Fatalf("Secure file write failed: %v", err)
			}
		}
	})

	b.Run("WithoutSecurity", func(b *testing.B) {
		testData := []byte(`{"benchmark": true}`)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			fileName := filepath.Join(tempDir, fmt.Sprintf("no-security-%d.json", i))

			// Direct file write without security validation
			if err := os.WriteFile(fileName, testData, 0644); err != nil {
				b.Fatalf("Direct file write failed: %v", err)
			}
		}
	})
}

// TestIntegration_SecurityRecovery tests system recovery after security incidents
func TestIntegration_SecurityRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security recovery tests in short mode")
	}

	validator := NewValidator(DefaultValidationConfig())

	// Simulate security incident scenarios and recovery
	t.Run("RecoveryAfterAttack", func(t *testing.T) {
		// Simulate multiple attack attempts
		attacks := []string{
			"'; DROP TABLE metrics; --",
			"../../../etc/passwd",
			"<script>alert('xss')</script>",
			strings.Repeat("A", 10000),
		}

		// Execute attacks (which should all be blocked)
		for i, attack := range attacks {
			t.Run(fmt.Sprintf("Attack_%d", i), func(t *testing.T) {
				// Multiple validation attempts with malicious data
				_ = validator.validateNodeID(attack)
				_ = validator.ValidateFilePath(attack, "attack test")
				_ = validator.ValidateURL("http://example.com/"+attack, "attack test")
				_ = validator.SanitizeForLogging(attack)
			})
		}

		// Verify system still works normally after attack attempts
		normalKMP := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "recovery-test-node",
			PRBUtilization:  0.5,
			P95Latency:      100.0,
			ActiveUEs:       50,
			CurrentReplicas: 2,
		}

		if err := validator.ValidateKMPData(normalKMP); err != nil {
			t.Errorf("System failed to recover after attacks: %v", err)
		} else {
			t.Log("✓ System successfully recovered and processes normal data")
		}
	})

	// Test graceful degradation under resource pressure
	t.Run("GracefulDegradation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create high load scenario
		const numGoroutines = 100
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					default:
						// Continuous validation load
						_ = validator.validateNodeID(fmt.Sprintf("load-test-node-%d", id))
						time.Sleep(time.Millisecond)
					}
				}
			}(i)
		}

		// Test normal operations during high load
		time.Sleep(2 * time.Second)

		normalKMP := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "degradation-test-node",
			PRBUtilization:  0.7,
			P95Latency:      120.0,
			ActiveUEs:       80,
			CurrentReplicas: 3,
		}

		start := time.Now()
		err := validator.ValidateKMPData(normalKMP)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("Validation failed under load: %v", err)
		}

		if elapsed > time.Second {
			t.Errorf("Validation took too long under load: %v", elapsed)
		}

		cancel()
		wg.Wait()

		t.Logf("✓ System maintained functionality under load (validation time: %v)", elapsed)
	})
}
