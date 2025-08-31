//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/planner/internal/rules"
	"github.com/nephio-project/nephoran-intent-operator/planner/internal/security"
	"gopkg.in/yaml.v3"
)

// TestEndToEndScenario tests the complete flow from metrics fetching to intent generation
func TestEndToEndScenario(t *testing.T) {
	// Create a test server that provides KMP metrics
	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kmpData := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "test-node",
			PRBUtilization:  0.9,   // High utilization to trigger scale-out
			P95Latency:      150.0, // High latency to trigger scale-out
			ActiveUEs:       200,
			CurrentReplicas: 2,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(kmpData)
	}))
	defer metricsServer.Close()

	// Create temporary directories for output and state
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "handoff")
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create configuration
	cfg := &Config{
		MetricsURL:      metricsServer.URL,
		EventsURL:       "http://localhost:9091/events/ves",
		OutputDir:       outputDir,
		IntentEndpoint:  "http://localhost:8080/intent",
		PollingInterval: 1 * time.Second, // Short interval for testing
		SimMode:         false,
		StateFile:       stateFile,
	}

	// Ensure output directory exists
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create rule engine with test configuration
	ruleConfig := createRuleEngineConfig(cfg, nil)
	engine := rules.NewRuleEngine(ruleConfig)

	// Process metrics multiple times to meet the minimum data points requirement (3)
	for i := 0; i < 4; i++ {
		processMetrics(cfg, engine, security.NewValidator(security.DefaultValidationConfig()))
		time.Sleep(100 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// Check if intent file was created
	files, err := filepath.Glob(filepath.Join(outputDir, "intent-*.json"))
	if err != nil {
		t.Fatalf("Failed to list intent files: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No intent files were created after multiple polling cycles")
	}

	// Read and validate the intent file
	intentData, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("Failed to read intent file: %v", err)
	}

	var intent map[string]interface{}
	err = json.Unmarshal(intentData, &intent)
	if err != nil {
		t.Fatalf("Failed to parse intent JSON: %v", err)
	}

	// Validate intent content
	if intent["intent_type"] != "scaling" {
		t.Errorf("Expected intent_type 'scaling', got %v", intent["intent_type"])
	}
	if intent["target"] != "test-node" {
		t.Errorf("Expected target 'test-node', got %v", intent["target"])
	}

	replicas, ok := intent["replicas"].(float64)
	if !ok || replicas != 3 { // Should scale from 2 to 3
		t.Errorf("Expected replicas 3, got %v", intent["replicas"])
	}
}

// TestConfigurationWithComplexYAML tests loading complex YAML configurations
func TestConfigurationWithComplexYAML(t *testing.T) {
	complexYaml := `# Complex configuration with comments and various data types
planner:
  # HTTP endpoints
  metrics_url: "http://prometheus.monitoring.svc.cluster.local:9090/metrics/kpm"
  events_url: "http://ves-collector.logging.svc.cluster.local:9091/events/ves"
  output_dir: "/var/lib/nephoran/intents"
  intent_endpoint: "http://nephio-intent-api.nephio-system.svc.cluster.local:8080/intent"
  
  # Timing configuration
  polling_interval: 2m30s  # 2 minutes 30 seconds
  
  # Simulation settings
  sim_mode: false
  sim_data_file: "/opt/nephoran/examples/kpm-production.json"
  state_file: "/var/lib/nephoran/planner-state.json"

# Scaling behavior configuration
scaling_rules:
  # Cooldown between scaling decisions
  cooldown_duration: 5m
  
  # Replica limits
  min_replicas: 3
  max_replicas: 50
  
  # Evaluation window for metrics
  evaluation_window: 3m
  
  # Performance thresholds
  thresholds:
    latency:
      scale_out: 250.0  # milliseconds
      scale_in: 75.0   # milliseconds
    prb_utilization:
      scale_out: 0.85  # 85%
      scale_in: 0.25   # 25%

# Logging configuration
logging:
  level: "info"
  format: "json"
  
# Additional metadata (should be ignored by current parser)
metadata:
  version: "1.2.3"
  environment: "production"
  cluster: "ran-cluster-01"
  region: "us-west-2"`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "complex-config.yaml")
	err := os.WriteFile(configFile, []byte(complexYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to create complex config file: %v", err)
	}

	cfg := &Config{
		MetricsURL:      "http://localhost:9090/metrics/kmp",
		EventsURL:       "http://localhost:9091/events/ves",
		OutputDir:       "./handoff",
		IntentEndpoint:  "http://localhost:8080/intent",
		PollingInterval: 30 * time.Second,
		SimMode:         true,
		SimDataFile:     "examples/planner/kmp-sample.json",
		StateFile:       filepath.Join(os.TempDir(), "planner-state.json"),
	}

	err = loadConfig(configFile, cfg, security.NewValidator(security.DefaultValidationConfig()))
	if err != nil {
		t.Fatalf("Failed to load complex config: %v", err)
	}

	// Verify complex configuration was parsed correctly
	expectedURL := "http://prometheus.monitoring.svc.cluster.local:9090/metrics/kpm"
	if cfg.MetricsURL != expectedURL {
		t.Errorf("Expected MetricsURL %s, got %s", expectedURL, cfg.MetricsURL)
	}

	expectedInterval := 2*time.Minute + 30*time.Second
	if cfg.PollingInterval != expectedInterval {
		t.Errorf("Expected PollingInterval %v, got %v", expectedInterval, cfg.PollingInterval)
	}

	expectedOutputDir := "/var/lib/nephoran/intents"
	if cfg.OutputDir != expectedOutputDir {
		t.Errorf("Expected OutputDir %s, got %s", expectedOutputDir, cfg.OutputDir)
	}

	// Test rule engine configuration with complex YAML
	yamlConfig := &YAMLConfig{}
	yamlData, _ := os.ReadFile(configFile)
	err = yaml.Unmarshal(yamlData, yamlConfig)
	if err != nil {
		t.Fatalf("Failed to parse YAML for rule config: %v", err)
	}

	ruleConfig := createRuleEngineConfig(cfg, yamlConfig)

	expectedCooldown := 5 * time.Minute
	if ruleConfig.CooldownDuration != expectedCooldown {
		t.Errorf("Expected CooldownDuration %v, got %v", expectedCooldown, ruleConfig.CooldownDuration)
	}

	if ruleConfig.MinReplicas != 3 {
		t.Errorf("Expected MinReplicas 3, got %d", ruleConfig.MinReplicas)
	}

	if ruleConfig.MaxReplicas != 50 {
		t.Errorf("Expected MaxReplicas 50, got %d", ruleConfig.MaxReplicas)
	}

	if ruleConfig.LatencyThresholdHigh != 250.0 {
		t.Errorf("Expected LatencyThresholdHigh 250.0, got %f", ruleConfig.LatencyThresholdHigh)
	}
}

// TestHTTPClientErrorRecovery tests how the HTTP client handles various error conditions
func TestHTTPClientErrorRecovery(t *testing.T) {
	tests := []struct {
		name           string
		serverBehavior func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			"Server returns 500",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal server error"))
			},
			true,
		},
		{
			"Server returns 404",
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Not found"))
			},
			true,
		},
		{
			"Server returns invalid JSON",
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json{"))
			},
			true,
		},
		{
			"Server returns empty response",
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(""))
			},
			true,
		},
		{
			"Server closes connection immediately",
			func(w http.ResponseWriter, r *http.Request) {
				// This will cause the connection to be closed
				if hijacker, ok := w.(http.Hijacker); ok {
					conn, _, _ := hijacker.Hijack()
					conn.Close()
				}
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverBehavior))
			defer server.Close()

			_, err := fetchKPMMetrics(server.URL, security.NewValidator(security.DefaultValidationConfig()))

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
		})
	}
}

// TestSimModeIntegration tests simulation mode with actual file loading
func TestSimModeIntegration(t *testing.T) {
	// Create test KMP data file
	testKMPData := rules.KPMData{
		Timestamp:       time.Now(),
		NodeID:          "sim-test-node",
		PRBUtilization:  0.7,
		P95Latency:      80.0,
		ActiveUEs:       150,
		CurrentReplicas: 3,
	}

	tmpDir := t.TempDir()
	simDataFile := filepath.Join(tmpDir, "sim-kmp-data.json")

	data, err := json.MarshalIndent(testKMPData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test KMP data: %v", err)
	}

	err = os.WriteFile(simDataFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write sim data file: %v", err)
	}

	// Test loading sim data
	loadedData, err := loadSimData(simDataFile, security.NewValidator(security.DefaultValidationConfig()))
	if err != nil {
		t.Fatalf("Failed to load sim data: %v", err)
	}

	// Verify loaded data
	if loadedData.NodeID != testKMPData.NodeID {
		t.Errorf("Expected NodeID %s, got %s", testKMPData.NodeID, loadedData.NodeID)
	}
	if loadedData.PRBUtilization != testKMPData.PRBUtilization {
		t.Errorf("Expected PRBUtilization %f, got %f", testKMPData.PRBUtilization, loadedData.PRBUtilization)
	}
	if loadedData.CurrentReplicas != testKMPData.CurrentReplicas {
		t.Errorf("Expected CurrentReplicas %d, got %d", testKMPData.CurrentReplicas, loadedData.CurrentReplicas)
	}
}

// TestMetricsDirIntegration tests loading metrics from a directory
func TestMetricsDirIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, "metrics")
	err := os.MkdirAll(metricsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create metrics directory: %v", err)
	}

	// Create multiple KMP files with different timestamps
	files := []struct {
		name string
		data rules.KPMData
		age  time.Duration
	}{
		{
			"kmp-old.json",
			rules.KPMData{
				Timestamp:       time.Now().Add(-2 * time.Hour),
				NodeID:          "old-node",
				PRBUtilization:  0.5,
				P95Latency:      50.0,
				ActiveUEs:       100,
				CurrentReplicas: 2,
			},
			2 * time.Hour,
		},
		{
			"kmp-recent.json",
			rules.KPMData{
				Timestamp:       time.Now().Add(-10 * time.Minute),
				NodeID:          "recent-node",
				PRBUtilization:  0.8,
				P95Latency:      120.0,
				ActiveUEs:       180,
				CurrentReplicas: 4,
			},
			10 * time.Minute,
		},
		{
			"kmp-latest.json",
			rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "latest-node",
				PRBUtilization:  0.9,
				P95Latency:      150.0,
				ActiveUEs:       200,
				CurrentReplicas: 5,
			},
			0,
		},
	}

	for _, f := range files {
		filePath := filepath.Join(metricsDir, f.name)
		data, err := json.MarshalIndent(f.data, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal KMP data for %s: %v", f.name, err)
		}

		err = os.WriteFile(filePath, data, 0644)
		if err != nil {
			t.Fatalf("Failed to write KMP file %s: %v", f.name, err)
		}

		// Set file modification time to simulate age
		modTime := time.Now().Add(-f.age)
		err = os.Chtimes(filePath, modTime, modTime)
		if err != nil {
			t.Fatalf("Failed to set modification time for %s: %v", f.name, err)
		}
	}

	// Test loading latest metrics from directory
	latestData, err := loadLatestMetricsFromDir(metricsDir, security.NewValidator(security.DefaultValidationConfig()))
	if err != nil {
		t.Fatalf("Failed to load latest metrics: %v", err)
	}

	// Should load the most recent file (latest-node)
	if latestData.NodeID != "latest-node" {
		t.Errorf("Expected latest NodeID 'latest-node', got %s", latestData.NodeID)
	}
	if latestData.PRBUtilization != 0.9 {
		t.Errorf("Expected latest PRBUtilization 0.9, got %f", latestData.PRBUtilization)
	}
}

// TestConfigValidationWithWarnings tests configuration validation that logs warnings
func TestConfigValidationWithWarnings(t *testing.T) {
	// Test configuration with invalid duration values that should generate warnings
	warningYaml := `planner:
  polling_interval: 30s
  
scaling_rules:
  cooldown_duration: "invalid-duration"
  evaluation_window: "also-invalid"
  min_replicas: 1
  max_replicas: 10
  
  thresholds:
    latency:
      scale_out: 100.0
      scale_in: 50.0
    prb_utilization:
      scale_out: 0.8
      scale_in: 0.3`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "warning-config.yaml")
	err := os.WriteFile(configFile, []byte(warningYaml), 0644)
	if err != nil {
		t.Fatalf("Failed to create warning config file: %v", err)
	}

	cfg := &Config{
		StateFile: filepath.Join(tmpDir, "state.json"),
	}

	// Load config (should succeed despite warnings)
	err = loadConfig(configFile, cfg, security.NewValidator(security.DefaultValidationConfig()))
	if err != nil {
		t.Fatalf("Config loading should succeed despite warnings: %v", err)
	}

	// Load full YAML config
	yamlData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var yamlConfig YAMLConfig
	err = yaml.Unmarshal(yamlData, &yamlConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Create rule engine config (should use defaults for invalid durations)
	ruleConfig := createRuleEngineConfig(cfg, &yamlConfig)

	// Should use default values when invalid durations are provided
	if ruleConfig.CooldownDuration != 60*time.Second {
		t.Errorf("Expected default CooldownDuration 60s, got %v", ruleConfig.CooldownDuration)
	}
	if ruleConfig.EvaluationWindow != 90*time.Second {
		t.Errorf("Expected default EvaluationWindow 90s, got %v", ruleConfig.EvaluationWindow)
	}
}
