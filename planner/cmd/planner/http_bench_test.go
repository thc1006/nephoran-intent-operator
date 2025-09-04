package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/planner/internal/security"
)

// BenchmarkHTTPClientConnectionReuse benchmarks HTTP client performance with connection reuse
func BenchmarkHTTPClientConnectionReuse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"timestamp":"2023-01-01T00:00:00Z","node_id":"test","prb_utilization":0.5,"p95_latency":25.0,"active_ues":100,"current_replicas":2}`))
	}))
	defer server.Close() // #nosec G307 - Error handled in defer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := httpClient.Get(server.URL)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}

// BenchmarkHTTPClientWithoutConnectionReuse benchmarks a new client for each request
func BenchmarkHTTPClientWithoutConnectionReuse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"timestamp":"2023-01-01T00:00:00Z","node_id":"test","prb_utilization":0.5,"p95_latency":25.0,"active_ues":100,"current_replicas":2}`))
	}))
	defer server.Close() // #nosec G307 - Error handled in defer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a new client for each request (no connection reuse)
		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()
	}
}

// BenchmarkFetchKPMMetrics benchmarks the actual fetchKPMMetrics function
func BenchmarkFetchKPMMetrics(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"timestamp":"2023-01-01T00:00:00Z","node_id":"test","prb_utilization":0.5,"p95_latency":25.0,"active_ues":100,"current_replicas":2}`))
	}))
	defer server.Close() // #nosec G307 - Error handled in defer

	validator := security.NewValidator(security.DefaultValidationConfig())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fetchKPMMetrics(server.URL, validator)
		if err != nil {
			b.Fatalf("fetchKPMMetrics failed: %v", err)
		}
	}
}

// BenchmarkConfigLoading benchmarks configuration loading performance
func BenchmarkConfigLoading(b *testing.B) {
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

	tmpDir := b.TempDir()
	configFile := tmpDir + "/bench-config.yaml"
	err := writeTestFile(configFile, yamlContent)
	if err != nil {
		b.Fatalf("Failed to create test config file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := &Config{
			MetricsURL:      "http://localhost:9090/metrics/kmp",
			EventsURL:       "http://localhost:9091/events/ves",
			OutputDir:       "./handoff",
			IntentEndpoint:  "http://localhost:8080/intent",
			PollingInterval: 30 * time.Second,
			SimMode:         false,
		}

		validator := security.NewValidator(security.DefaultValidationConfig())
		err := loadConfig(configFile, cfg, validator)
		if err != nil {
			b.Fatalf("loadConfig failed: %v", err)
		}
	}
}

// writeTestFile is a helper function for benchmarks
func writeTestFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}
