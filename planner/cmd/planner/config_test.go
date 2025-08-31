package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/planner/internal/security"
)

func TestLoadConfig(t *testing.T) {
	// Test with valid YAML config
	yamlContent := `planner:
  metrics_url: "http://test:9090/metrics"
  events_url: "http://test:9091/events"
  output_dir: "./test-output"
  intent_endpoint: "http://test:8080/intent"
  polling_interval: 45s
  sim_mode: true
  sim_data_file: "test-file.json"
  state_file: "test-state.json"

scaling_rules:
  cooldown_duration: 120s
  min_replicas: 2
  max_replicas: 20
  evaluation_window: 180s
  
  thresholds:
    latency:
      scale_out: 200.0
      scale_in: 100.0
    prb_utilization:
      scale_out: 0.9
      scale_in: 0.4

logging:
  level: "debug"
  format: "text"`

	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Create config with defaults
	cfg := &Config{
		MetricsURL:      "http://localhost:9090/metrics/kmp",
		EventsURL:       "http://localhost:9091/events/ves",
		OutputDir:       "./handoff",
		IntentEndpoint:  "http://localhost:8080/intent",
		PollingInterval: 30 * time.Second,
		SimMode:         false,
		SimDataFile:     "examples/planner/kmp-sample.json",
		StateFile:       filepath.Join(os.TempDir(), "planner-state.json"),
	}

	// Load configuration
	validator := security.NewValidator(security.DefaultValidationConfig())
	err = loadConfig(configFile, cfg, validator)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify configuration was loaded correctly
	if cfg.MetricsURL != "http://test:9090/metrics" {
		t.Errorf("Expected MetricsURL 'http://test:9090/metrics', got '%s'", cfg.MetricsURL)
	}
	if cfg.EventsURL != "http://test:9091/events" {
		t.Errorf("Expected EventsURL 'http://test:9091/events', got '%s'", cfg.EventsURL)
	}
	if cfg.OutputDir != "./test-output" {
		t.Errorf("Expected OutputDir './test-output', got '%s'", cfg.OutputDir)
	}
	if cfg.IntentEndpoint != "http://test:8080/intent" {
		t.Errorf("Expected IntentEndpoint 'http://test:8080/intent', got '%s'", cfg.IntentEndpoint)
	}
	if cfg.PollingInterval != 45*time.Second {
		t.Errorf("Expected PollingInterval 45s, got %v", cfg.PollingInterval)
	}
	if !cfg.SimMode {
		t.Errorf("Expected SimMode true, got %v", cfg.SimMode)
	}
	if cfg.SimDataFile != "test-file.json" {
		t.Errorf("Expected SimDataFile 'test-file.json', got '%s'", cfg.SimDataFile)
	}
	if !filepath.IsAbs(cfg.StateFile) {
		t.Errorf("Expected StateFile to be absolute path, got '%s'", cfg.StateFile)
	}
}

func TestLoadConfigInvalidFile(t *testing.T) {
	cfg := &Config{}
	validator := security.NewValidator(security.DefaultValidationConfig())
	err := loadConfig("nonexistent.yaml", cfg, validator)
	if err == nil {
		t.Errorf("Expected error when loading nonexistent file, got nil")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	invalidYaml := `invalid: yaml: content
  - missing
  [ brackets`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid-config.yaml")
	err := os.WriteFile(configFile, []byte(invalidYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg := &Config{}
	validator := security.NewValidator(security.DefaultValidationConfig())
	err = loadConfig(configFile, cfg, validator)
	if err == nil {
		t.Errorf("Expected error when loading invalid YAML, got nil")
	}
}

func TestCreateRuleEngineConfig(t *testing.T) {
	cfg := &Config{
		StateFile: "/tmp/test-state.json",
	}

	yamlConfig := &YAMLConfig{
		ScalingRules: ScalingRulesConfig{
			CooldownDuration: "120s",
			MinReplicas:      2,
			MaxReplicas:      20,
			EvaluationWindow: "180s",
			Thresholds: ScalingThresholdsConfig{
				Latency: ThresholdConfig{
					ScaleOut: 200.0,
					ScaleIn:  100.0,
				},
				PRBUtilization: ThresholdConfig{
					ScaleOut: 0.9,
					ScaleIn:  0.4,
				},
			},
		},
	}

	ruleConfig := createRuleEngineConfig(cfg, yamlConfig)

	if ruleConfig.StateFile != "/tmp/test-state.json" {
		t.Errorf("Expected StateFile '/tmp/test-state.json', got '%s'", ruleConfig.StateFile)
	}
	if ruleConfig.CooldownDuration != 120*time.Second {
		t.Errorf("Expected CooldownDuration 120s, got %v", ruleConfig.CooldownDuration)
	}
	if ruleConfig.MinReplicas != 2 {
		t.Errorf("Expected MinReplicas 2, got %d", ruleConfig.MinReplicas)
	}
	if ruleConfig.MaxReplicas != 20 {
		t.Errorf("Expected MaxReplicas 20, got %d", ruleConfig.MaxReplicas)
	}
	if ruleConfig.EvaluationWindow != 180*time.Second {
		t.Errorf("Expected EvaluationWindow 180s, got %v", ruleConfig.EvaluationWindow)
	}
	if ruleConfig.LatencyThresholdHigh != 200.0 {
		t.Errorf("Expected LatencyThresholdHigh 200.0, got %f", ruleConfig.LatencyThresholdHigh)
	}
	if ruleConfig.LatencyThresholdLow != 100.0 {
		t.Errorf("Expected LatencyThresholdLow 100.0, got %f", ruleConfig.LatencyThresholdLow)
	}
	if ruleConfig.PRBThresholdHigh != 0.9 {
		t.Errorf("Expected PRBThresholdHigh 0.9, got %f", ruleConfig.PRBThresholdHigh)
	}
	if ruleConfig.PRBThresholdLow != 0.4 {
		t.Errorf("Expected PRBThresholdLow 0.4, got %f", ruleConfig.PRBThresholdLow)
	}
}

func TestCreateRuleEngineConfigDefaults(t *testing.T) {
	cfg := &Config{
		StateFile: "/tmp/test-state.json",
	}

	// Test with nil yaml config (should use defaults)
	ruleConfig := createRuleEngineConfig(cfg, nil)

	if ruleConfig.CooldownDuration != 60*time.Second {
		t.Errorf("Expected default CooldownDuration 60s, got %v", ruleConfig.CooldownDuration)
	}
	if ruleConfig.MinReplicas != 1 {
		t.Errorf("Expected default MinReplicas 1, got %d", ruleConfig.MinReplicas)
	}
	if ruleConfig.MaxReplicas != 10 {
		t.Errorf("Expected default MaxReplicas 10, got %d", ruleConfig.MaxReplicas)
	}
	if ruleConfig.EvaluationWindow != 90*time.Second {
		t.Errorf("Expected default EvaluationWindow 90s, got %v", ruleConfig.EvaluationWindow)
	}
}

// TestHTTPClientConnectionReuse tests HTTP client connection pooling
func TestHTTPClientConnectionReuse(t *testing.T) {
	// Test that the client is configured for connection reuse
	if httpClient.Transport == nil {
		t.Fatal("HTTP client should have a custom transport")
	}

	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("HTTP client transport should be *http.Transport")
	}

	// Verify connection pooling settings
	if transport.MaxIdleConns != 10 {
		t.Errorf("Expected MaxIdleConns 10, got %d", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 5 {
		t.Errorf("Expected MaxIdleConnsPerHost 5, got %d", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout 90s, got %v", transport.IdleConnTimeout)
	}

	// Test that the client actually works
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"timestamp":"2023-01-01T00:00:00Z","node_id":"test","prb_utilization":0.5,"p95_latency":25.0,"active_ues":100,"current_replicas":2}`))
	}))
	defer server.Close()

	// Make multiple requests using the global httpClient
	for i := 0; i < 3; i++ {
		resp, err := httpClient.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		resp.Body.Close()
	}
}

// TestHTTPClientConcurrentUsage tests concurrent HTTP requests
func TestHTTPClientConcurrentUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"timestamp":"2023-01-01T00:00:00Z","node_id":"test","prb_utilization":0.5,"p95_latency":25.0,"active_ues":100,"current_replicas":2}`))
	}))
	defer server.Close()

	const numRequests = 20
	var wg sync.WaitGroup
	errorChan := make(chan error, numRequests)

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			resp, err := httpClient.Get(server.URL)
			if err != nil {
				errorChan <- fmt.Errorf("request %d failed: %w", id, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errorChan <- fmt.Errorf("request %d returned status %d", id, resp.StatusCode)
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	elapsed := time.Since(start)

	// Check for errors
	for err := range errorChan {
		t.Error(err)
	}

	// Verify concurrent execution was efficient (should be much less than sequential)
	expectedSequentialTime := time.Duration(numRequests) * 10 * time.Millisecond
	if elapsed > expectedSequentialTime/2 {
		t.Errorf("Concurrent requests took too long: %v (expected less than %v)", elapsed, expectedSequentialTime/2)
	}
}

// TestHTTPClientTimeouts tests various timeout scenarios
func TestHTTPClientTimeouts(t *testing.T) {
	tests := []struct {
		name          string
		serverDelay   time.Duration
		expectTimeout bool
	}{
		{"Normal response", 50 * time.Millisecond, false},
		{"Slow response within timeout", 5 * time.Second, false},
		{"Response exceeding timeout", 15 * time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.serverDelay)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"test":"data"}`))
			}))
			defer server.Close()

			resp, err := httpClient.Get(server.URL)
			if tt.expectTimeout {
				if err == nil {
					resp.Body.Close()
					t.Error("Expected timeout error, but request succeeded")
				}
				// Check if it's a timeout error
				if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
					t.Errorf("Expected timeout error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					resp.Body.Close()
				}
			}
		})
	}
}

// TestConfigurationValidationEdgeCases tests invalid configuration values
func TestConfigurationValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		description string
	}{
		{
			"Negative replicas",
			`planner:
  polling_interval: 30s
scaling_rules:
  min_replicas: -1
  max_replicas: 5`,
			false, // Currently no validation, but should log warning
			"Negative min_replicas should be handled gracefully",
		},
		{
			"Invalid polling interval",
			`planner:
  polling_interval: "invalid-duration"
scaling_rules:
  min_replicas: 1
  max_replicas: 5`,
			true,
			"Invalid duration format should cause error",
		},
		{
			"Zero max replicas",
			`planner:
  polling_interval: 30s
scaling_rules:
  min_replicas: 1
  max_replicas: 0`,
			false, // Currently no validation
			"Zero max_replicas should use defaults",
		},
		{
			"Negative thresholds",
			`planner:
  polling_interval: 30s
scaling_rules:
  min_replicas: 1
  max_replicas: 5
  thresholds:
    latency:
      scale_out: -100.0
      scale_in: -50.0
    prb_utilization:
      scale_out: -0.8
      scale_in: -0.3`,
			false, // Currently no validation
			"Negative thresholds should use defaults",
		},
		{
			"Extreme timeout values",
			`planner:
  polling_interval: 1ns
scaling_rules:
  cooldown_duration: 1000h
  evaluation_window: 1ns`,
			false,
			"Extreme timeout values should be parsed correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "test-config.yaml")
			err := os.WriteFile(configFile, []byte(tt.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			cfg := &Config{
				MetricsURL:      "http://localhost:9090/metrics/kmp",
				EventsURL:       "http://localhost:9091/events/ves",
				OutputDir:       "./handoff",
				IntentEndpoint:  "http://localhost:8080/intent",
				PollingInterval: 30 * time.Second,
				SimMode:         false,
				SimDataFile:     "examples/planner/kmp-sample.json",
				StateFile:       filepath.Join(os.TempDir(), "planner-state.json"),
			}

			err = loadConfig(configFile, cfg, security.NewValidator(security.DefaultValidationConfig()))
			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.description)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}
		})
	}
}

// TestConcurrentConfigurationLoading tests loading config from multiple goroutines
func TestConcurrentConfigurationLoading(t *testing.T) {
	yamlContent := `planner:
  metrics_url: "http://test:9090/metrics"
  events_url: "http://test:9091/events"
  output_dir: "./test-output"
  intent_endpoint: "http://test:8080/intent"
  polling_interval: 45s
  sim_mode: true
  sim_data_file: "test-file.json"
  state_file: "test-state.json"

scaling_rules:
  cooldown_duration: 120s
  min_replicas: 2
  max_replicas: 20
  evaluation_window: 180s
  
  thresholds:
    latency:
      scale_out: 200.0
      scale_in: 100.0
    prb_utilization:
      scale_out: 0.9
      scale_in: 0.4

logging:
  level: "debug"
  format: "text"`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines)
	resultChan := make(chan *Config, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			cfg := &Config{
				MetricsURL:      "http://localhost:9090/metrics/kmp",
				EventsURL:       "http://localhost:9091/events/ves",
				OutputDir:       "./handoff",
				IntentEndpoint:  "http://localhost:8080/intent",
				PollingInterval: 30 * time.Second,
				SimMode:         false,
				SimDataFile:     "examples/planner/kmp-sample.json",
				StateFile:       filepath.Join(os.TempDir(), "planner-state.json"),
			}

			if err := loadConfig(configFile, cfg, security.NewValidator(security.DefaultValidationConfig())); err != nil {
				errorChan <- fmt.Errorf("goroutine %d failed: %w", id, err)
				return
			}

			resultChan <- cfg
		}(i)
	}

	wg.Wait()
	close(errorChan)
	close(resultChan)

	// Check for errors
	for err := range errorChan {
		t.Error(err)
	}

	// Verify all results are consistent
	var configs []*Config
	for cfg := range resultChan {
		configs = append(configs, cfg)
	}

	if len(configs) != numGoroutines {
		t.Errorf("Expected %d configs, got %d", numGoroutines, len(configs))
	}

	// Verify all configs have the same values
	for i, cfg := range configs {
		if cfg.MetricsURL != "http://test:9090/metrics" {
			t.Errorf("Config %d has wrong MetricsURL: %s", i, cfg.MetricsURL)
		}
		if cfg.PollingInterval != 45*time.Second {
			t.Errorf("Config %d has wrong PollingInterval: %v", i, cfg.PollingInterval)
		}
		if !cfg.SimMode {
			t.Errorf("Config %d has wrong SimMode: %v", i, cfg.SimMode)
		}
	}
}

// TestIntegrationWithRealConfigFiles tests loading actual config files
func TestIntegrationWithRealConfigFiles(t *testing.T) {
	tests := []struct {
		name             string
		configPath       string
		expectedURL      string
		expectedInterval time.Duration
	}{
		{
			"Default config",
			"../../config/config.yaml",
			"http://localhost:9090/metrics/kpm",
			30 * time.Second,
		},
		{
			"Local test config",
			"../../config/local-test.yaml",
			"", // May not exist or have different values
			0,  // May not exist or have different values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if config file exists
			if _, err := os.Stat(tt.configPath); os.IsNotExist(err) {
				t.Skipf("Config file %s does not exist, skipping test", tt.configPath)
				return
			}

			cfg := &Config{
				MetricsURL:      "http://localhost:9090/metrics/kmp",
				EventsURL:       "http://localhost:9091/events/ves",
				OutputDir:       "./handoff",
				IntentEndpoint:  "http://localhost:8080/intent",
				PollingInterval: 30 * time.Second,
				SimMode:         false,
				SimDataFile:     "examples/planner/kmp-sample.json",
				StateFile:       filepath.Join(os.TempDir(), "planner-state.json"),
			}

			// Use a more permissive validator for test files that might have different URL schemes
			testValidatorConfig := security.DefaultValidationConfig()
			testValidatorConfig.AllowedSchemes = append(testValidatorConfig.AllowedSchemes, "file")
			testValidator := security.NewValidator(testValidatorConfig)

			err := loadConfig(tt.configPath, cfg, testValidator)
			if err != nil {
				t.Errorf("Failed to load real config file %s: %v", tt.configPath, err)
				return
			}

			// Only validate expected values if they are set
			if tt.expectedURL != "" && cfg.MetricsURL != tt.expectedURL {
				t.Errorf("Expected MetricsURL %s, got %s", tt.expectedURL, cfg.MetricsURL)
			}
			if tt.expectedInterval != 0 && cfg.PollingInterval != tt.expectedInterval {
				t.Errorf("Expected PollingInterval %v, got %v", tt.expectedInterval, cfg.PollingInterval)
			}

			// Verify basic integrity
			if cfg.MetricsURL == "" {
				t.Error("MetricsURL should not be empty")
			}
			if cfg.PollingInterval <= 0 {
				t.Error("PollingInterval should be positive")
			}
		})
	}
}

// TestYAMLParsingErrorScenarios tests various YAML parsing failure cases
func TestYAMLParsingErrorScenarios(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			"Empty file",
			"",
			false, // Empty YAML is valid
		},
		{
			"Invalid YAML syntax",
			"invalid: yaml: [",
			true,
		},
		{
			"Malformed duration",
			`planner:
  polling_interval: "not-a-duration"`,
			true,
		},
		{
			"Mixed types",
			`planner:
  polling_interval: 123
  sim_mode: "not-a-boolean"`,
			true, // YAML cannot coerce this string to boolean
		},
		{
			"Unicode content",
			`planner:
  metrics_url: "http://测试:9090/metrics"
  output_dir: "./输出目录"`,
			false,
		},
		{
			"Large file content",
			func() string {
				content := "planner:\n"
				for i := 0; i < 1000; i++ {
					content += fmt.Sprintf("  field_%d: value_%d\n", i, i)
				}
				return content
			}(),
			false,
		},
		{
			"Deeply nested structure",
			`planner:
  nested:
    level1:
      level2:
        level3:
          level4:
            value: "deep"`,
			false,
		},
		{
			"Binary data",
			"\x00\x01\x02\x03\x04\x05",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "test-config.yaml")
			err := os.WriteFile(configFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			cfg := &Config{}
			err = loadConfig(configFile, cfg, security.NewValidator(security.DefaultValidationConfig()))

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', but got none", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}
		})
	}
}

// TestFileSystemErrorHandling tests file system related error scenarios
func TestFileSystemErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		expectError bool
	}{
		{
			"Permission denied",
			func(t *testing.T) string {
				// Skip on Windows as permission handling is different
				if os.Getenv("OS") == "Windows_NT" {
					t.Skip("Skipping permission test on Windows")
				}
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "no-permission.yaml")
				err := os.WriteFile(configFile, []byte("planner:\n  test: value"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				// Try to remove read permission (may not work on Windows)
				os.Chmod(configFile, 0000)
				return configFile
			},
			true,
		},
		{
			"Directory instead of file",
			func(t *testing.T) string {
				tmpDir := t.TempDir()
				dirPath := filepath.Join(tmpDir, "config-dir")
				os.Mkdir(dirPath, 0755)
				return dirPath
			},
			true,
		},
		{
			"Non-existent directory",
			func(t *testing.T) string {
				return "/non/existent/path/config.yaml"
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setup(t)

			cfg := &Config{}
			err := loadConfig(configPath, cfg, security.NewValidator(security.DefaultValidationConfig()))

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', but got none", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.name, err)
			}

			// Cleanup: restore permissions if needed
			if tt.name == "Permission denied" {
				os.Chmod(configPath, 0644)
			}
		})
	}
}

// TestEnvironmentVariableOverrides tests environment variable precedence
func TestEnvironmentVariableOverrides(t *testing.T) {
	// Save current environment
	oldMetricsURL := os.Getenv("PLANNER_METRICS_URL")
	oldOutputDir := os.Getenv("PLANNER_OUTPUT_DIR")
	oldSimMode := os.Getenv("PLANNER_SIM_MODE")
	oldMetricsDir := os.Getenv("PLANNER_METRICS_DIR")

	defer func() {
		// Restore environment
		os.Setenv("PLANNER_METRICS_URL", oldMetricsURL)
		os.Setenv("PLANNER_OUTPUT_DIR", oldOutputDir)
		os.Setenv("PLANNER_SIM_MODE", oldSimMode)
		os.Setenv("PLANNER_METRICS_DIR", oldMetricsDir)
	}()

	// Set test environment variables
	os.Setenv("PLANNER_METRICS_URL", "http://env:9090/metrics")
	os.Setenv("PLANNER_OUTPUT_DIR", "./env-output")
	os.Setenv("PLANNER_SIM_MODE", "true")
	os.Setenv("PLANNER_METRICS_DIR", "./env-metrics")

	// Create a config with different values
	yamlContent := `planner:
  metrics_url: "http://config:9090/metrics"
  output_dir: "./config-output"
  sim_mode: false`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg := &Config{
		MetricsURL: "http://default:9090/metrics",
		OutputDir:  "./default-output",
		SimMode:    false,
		MetricsDir: "./default-metrics",
	}

	// Load config (this simulates the main function logic)
	loadConfig(configFile, cfg, security.NewValidator(security.DefaultValidationConfig()))

	// Apply environment variable overrides (from main function)
	if envURL := os.Getenv("PLANNER_METRICS_URL"); envURL != "" {
		cfg.MetricsURL = envURL
	}
	if envDir := os.Getenv("PLANNER_OUTPUT_DIR"); envDir != "" {
		cfg.OutputDir = envDir
	}
	if envSim := os.Getenv("PLANNER_SIM_MODE"); envSim == "true" {
		cfg.SimMode = true
	}
	if envMetricsDir := os.Getenv("PLANNER_METRICS_DIR"); envMetricsDir != "" {
		cfg.MetricsDir = envMetricsDir
	}

	// Verify environment variables override config file values
	if cfg.MetricsURL != "http://env:9090/metrics" {
		t.Errorf("Expected env MetricsURL, got %s", cfg.MetricsURL)
	}
	if cfg.OutputDir != "./env-output" {
		t.Errorf("Expected env OutputDir, got %s", cfg.OutputDir)
	}
	if !cfg.SimMode {
		t.Errorf("Expected env SimMode true, got %v", cfg.SimMode)
	}
	if cfg.MetricsDir != "./env-metrics" {
		t.Errorf("Expected env MetricsDir, got %s", cfg.MetricsDir)
	}
}
