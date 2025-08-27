package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name      string
		envVars   map[string]string
		checkFunc func(*Config, *testing.T)
	}{
		{
			name: "ENABLE_NETWORK_INTENT=false",
			envVars: map[string]string{
				"ENABLE_NETWORK_INTENT": "false",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.EnableNetworkIntent != false {
					t.Errorf("Expected EnableNetworkIntent to be false, got %v", cfg.EnableNetworkIntent)
				}
			},
		},
		{
			name: "ENABLE_NETWORK_INTENT=true",
			envVars: map[string]string{
				"ENABLE_NETWORK_INTENT": "true",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.EnableNetworkIntent != true {
					t.Errorf("Expected EnableNetworkIntent to be true, got %v", cfg.EnableNetworkIntent)
				}
			},
		},
		{
			name: "ENABLE_LLM_INTENT=true",
			envVars: map[string]string{
				"ENABLE_LLM_INTENT": "true",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.EnableLLMIntent != true {
					t.Errorf("Expected EnableLLMIntent to be true, got %v", cfg.EnableLLMIntent)
				}
			},
		},
		{
			name: "ENABLE_LLM_INTENT=false",
			envVars: map[string]string{
				"ENABLE_LLM_INTENT": "false",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.EnableLLMIntent != false {
					t.Errorf("Expected EnableLLMIntent to be false, got %v", cfg.EnableLLMIntent)
				}
			},
		},
		{
			name: "LLM_TIMEOUT_SECS=30",
			envVars: map[string]string{
				"LLM_TIMEOUT_SECS": "30",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				expected := 30 * time.Second
				if cfg.LLMTimeout != expected {
					t.Errorf("Expected LLMTimeout to be %v, got %v", expected, cfg.LLMTimeout)
				}
			},
		},
		{
			name: "LLM_MAX_RETRIES=5",
			envVars: map[string]string{
				"LLM_MAX_RETRIES": "5",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.LLMMaxRetries != 5 {
					t.Errorf("Expected LLMMaxRetries to be 5, got %v", cfg.LLMMaxRetries)
				}
			},
		},
		{
			name: "LLM_CACHE_MAX_ENTRIES=1024",
			envVars: map[string]string{
				"LLM_CACHE_MAX_ENTRIES": "1024",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.LLMCacheMaxEntries != 1024 {
					t.Errorf("Expected LLMCacheMaxEntries to be 1024, got %v", cfg.LLMCacheMaxEntries)
				}
			},
		},
		{
			name: "HTTP_MAX_BODY=2097152",
			envVars: map[string]string{
				"HTTP_MAX_BODY": "2097152",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				expected := int64(2097152)
				if cfg.HTTPMaxBody != expected {
					t.Errorf("Expected HTTPMaxBody to be %v, got %v", expected, cfg.HTTPMaxBody)
				}
			},
		},
		{
			name: "METRICS_ENABLED=true",
			envVars: map[string]string{
				"METRICS_ENABLED": "true",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.MetricsEnabled != true {
					t.Errorf("Expected MetricsEnabled to be true, got %v", cfg.MetricsEnabled)
				}
			},
		},
		{
			name: "METRICS_ALLOWED_IPS with single IP",
			envVars: map[string]string{
				"METRICS_ALLOWED_IPS": "192.168.1.100",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				expected := []string{"192.168.1.100"}
				if len(cfg.MetricsAllowedIPs) != 1 || cfg.MetricsAllowedIPs[0] != expected[0] {
					t.Errorf("Expected MetricsAllowedIPs to be %v, got %v", expected, cfg.MetricsAllowedIPs)
				}
			},
		},
		{
			name: "METRICS_ALLOWED_IPS with multiple IPs",
			envVars: map[string]string{
				"METRICS_ALLOWED_IPS": "192.168.1.100,10.0.0.1,127.0.0.1",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				expected := []string{"192.168.1.100", "10.0.0.1", "127.0.0.1"}
				if len(cfg.MetricsAllowedIPs) != len(expected) {
					t.Errorf("Expected MetricsAllowedIPs length to be %d, got %d", len(expected), len(cfg.MetricsAllowedIPs))
					return
				}
				for i, expectedIP := range expected {
					if cfg.MetricsAllowedIPs[i] != expectedIP {
						t.Errorf("Expected MetricsAllowedIPs[%d] to be %s, got %s", i, expectedIP, cfg.MetricsAllowedIPs[i])
					}
				}
			},
		},
		{
			name: "METRICS_ENABLED=true with METRICS_ALLOWED_IPS=*",
			envVars: map[string]string{
				"METRICS_ENABLED":     "true",
				"METRICS_ALLOWED_IPS": "*",
			},
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.MetricsEnabled != true {
					t.Errorf("Expected MetricsEnabled to be true, got %v", cfg.MetricsEnabled)
				}
				expected := []string{"*"}
				if len(cfg.MetricsAllowedIPs) != 1 || cfg.MetricsAllowedIPs[0] != "*" {
					t.Errorf("Expected MetricsAllowedIPs to be %v, got %v", expected, cfg.MetricsAllowedIPs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test environment variables
			for key, value := range tt.envVars {
				// Ignore setenv error in test
				_ = os.Setenv(key, value)
			}

			// Set required OPENAI_API_KEY for validation
			// Ignore setenv error in test
			_ = os.Setenv("OPENAI_API_KEY", "test-key")

			// Clean up after test
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue == "" {
						if err := os.Unsetenv(key); err != nil {
							t.Logf("failed to unset %s: %v", key, err)
						}
					} else {
						// Ignore setenv error in test
						_ = os.Setenv(key, originalValue)
					}
				}
				if err := os.Unsetenv("OPENAI_API_KEY"); err != nil {
					t.Logf("failed to unset OPENAI_API_KEY: %v", err)
				} // Clean up test API key
			}()

			// Load configuration
			cfg, err := LoadFromEnv()
			if err != nil {
				t.Fatalf("LoadFromEnv() failed: %v", err)
			}

			// Run check function
			tt.checkFunc(cfg, t)
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	// Clear all relevant environment variables to test defaults
	envVars := []string{
		"ENABLE_NETWORK_INTENT",
		"ENABLE_LLM_INTENT",
		"LLM_TIMEOUT_SECS",
		"LLM_MAX_RETRIES",
		"LLM_CACHE_MAX_ENTRIES",
		"HTTP_MAX_BODY",
		"METRICS_ENABLED",
		"METRICS_ALLOWED_IPS",
	}

	// Save original environment
	originalEnv := make(map[string]string)
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Logf("failed to unset %s: %v", key, err)
		}
	}

	// Clean up after test
	defer func() {
		for key, originalValue := range originalEnv {
			if originalValue == "" {
				if err := os.Unsetenv(key); err != nil {
					t.Logf("failed to unset %s: %v", key, err)
				}
			} else {
				// Ignore setenv error in test
				_ = os.Setenv(key, originalValue)
			}
		}
	}()

	cfg := DefaultConfig()

	// Test defaults
	if cfg.EnableNetworkIntent != true {
		t.Errorf("Expected default EnableNetworkIntent to be true, got %v", cfg.EnableNetworkIntent)
	}

	if cfg.EnableLLMIntent != false {
		t.Errorf("Expected default EnableLLMIntent to be false, got %v", cfg.EnableLLMIntent)
	}

	if cfg.LLMTimeout != 15*time.Second {
		t.Errorf("Expected default LLMTimeout to be 15s, got %v", cfg.LLMTimeout)
	}

	if cfg.LLMMaxRetries != 2 {
		t.Errorf("Expected default LLMMaxRetries to be 2, got %v", cfg.LLMMaxRetries)
	}

	if cfg.LLMCacheMaxEntries != 512 {
		t.Errorf("Expected default LLMCacheMaxEntries to be 512, got %v", cfg.LLMCacheMaxEntries)
	}

	if cfg.HTTPMaxBody != 1048576 {
		t.Errorf("Expected default HTTPMaxBody to be 1048576, got %v", cfg.HTTPMaxBody)
	}

	if cfg.MetricsEnabled != false {
		t.Errorf("Expected default MetricsEnabled to be false, got %v", cfg.MetricsEnabled)
	}

	if len(cfg.MetricsAllowedIPs) != 0 {
		t.Errorf("Expected default MetricsAllowedIPs to be empty, got %v", cfg.MetricsAllowedIPs)
	}
}

func TestConfigGetters(t *testing.T) {
	cfg := &Config{
		EnableNetworkIntent: true,
		EnableLLMIntent:     false,
		LLMTimeout:          25 * time.Second,
		LLMMaxRetries:       3,
		LLMCacheMaxEntries:  256,
		HTTPMaxBody:         2048000,
		MetricsEnabled:      true,
		MetricsAllowedIPs:   []string{"10.0.0.1"},
	}

	if cfg.GetEnableNetworkIntent() != true {
		t.Errorf("GetEnableNetworkIntent() = %v, want true", cfg.GetEnableNetworkIntent())
	}

	if cfg.GetEnableLLMIntent() != false {
		t.Errorf("GetEnableLLMIntent() = %v, want false", cfg.GetEnableLLMIntent())
	}

	if cfg.GetLLMTimeout() != 25*time.Second {
		t.Errorf("GetLLMTimeout() = %v, want %v", cfg.GetLLMTimeout(), 25*time.Second)
	}

	if cfg.GetLLMMaxRetries() != 3 {
		t.Errorf("GetLLMMaxRetries() = %v, want 3", cfg.GetLLMMaxRetries())
	}

	if cfg.GetLLMCacheMaxEntries() != 256 {
		t.Errorf("GetLLMCacheMaxEntries() = %v, want 256", cfg.GetLLMCacheMaxEntries())
	}

	if cfg.GetHTTPMaxBody() != 2048000 {
		t.Errorf("GetHTTPMaxBody() = %v, want 2048000", cfg.GetHTTPMaxBody())
	}

	if cfg.GetMetricsEnabled() != true {
		t.Errorf("GetMetricsEnabled() = %v, want true", cfg.GetMetricsEnabled())
	}

	allowedIPs := cfg.GetMetricsAllowedIPs()
	if len(allowedIPs) != 1 || allowedIPs[0] != "10.0.0.1" {
		t.Errorf("GetMetricsAllowedIPs() = %v, want [10.0.0.1]", allowedIPs)
	}
}
