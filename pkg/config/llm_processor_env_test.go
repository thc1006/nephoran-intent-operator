package config

import (
	"os"
	"testing"
	"time"
)

func TestLLMProcessorConfigEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name      string
		envVars   map[string]string
		checkFunc func(*LLMProcessorConfig, *testing.T)
		wantError bool
	}{
		{
			name: "LLM_TIMEOUT_SECS=30",
			envVars: map[string]string{
				"LLM_TIMEOUT_SECS": "30",
			},
			checkFunc: func(cfg *LLMProcessorConfig, t *testing.T) {
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
			checkFunc: func(cfg *LLMProcessorConfig, t *testing.T) {
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
			checkFunc: func(cfg *LLMProcessorConfig, t *testing.T) {
				if cfg.CacheMaxEntries != 1024 {
					t.Errorf("Expected CacheMaxEntries to be 1024, got %v", cfg.CacheMaxEntries)
				}
			},
		},
		{
			name: "HTTP_MAX_BODY=2097152",
			envVars: map[string]string{
				"HTTP_MAX_BODY": "2097152",
			},
			checkFunc: func(cfg *LLMProcessorConfig, t *testing.T) {
				expected := int64(2097152)
				if cfg.MaxRequestSize != expected {
					t.Errorf("Expected MaxRequestSize to be %v, got %v", expected, cfg.MaxRequestSize)
				}
			},
		},
		{
			name: "METRICS_ENABLED=true",
			envVars: map[string]string{
				"METRICS_ENABLED": "true",
			},
			checkFunc: func(cfg *LLMProcessorConfig, t *testing.T) {
				if cfg.MetricsEnabled != true {
					t.Errorf("Expected MetricsEnabled to be true, got %v", cfg.MetricsEnabled)
				}
			},
		},
		{
			name: "METRICS_ALLOWED_IPS=127.0.0.1,192.168.1.1",
			envVars: map[string]string{
				"METRICS_ALLOWED_IPS": "127.0.0.1,192.168.1.1",
			},
			checkFunc: func(cfg *LLMProcessorConfig, t *testing.T) {
				expected := []string{"127.0.0.1", "192.168.1.1"}
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
			name: "Invalid LLM_TIMEOUT_SECS",
			envVars: map[string]string{
				"LLM_TIMEOUT_SECS": "invalid",
			},
			wantError: true,
		},
		{
			name: "Negative LLM_TIMEOUT_SECS",
			envVars: map[string]string{
				"LLM_TIMEOUT_SECS": "-5",
			},
			wantError: true,
		},
		{
			name: "Zero LLM_TIMEOUT_SECS",
			envVars: map[string]string{
				"LLM_TIMEOUT_SECS": "0",
			},
			wantError: true,
		},
		{
			name: "Invalid LLM_MAX_RETRIES",
			envVars: map[string]string{
				"LLM_MAX_RETRIES": "invalid",
			},
			wantError: true,
		},
		{
			name: "Negative LLM_MAX_RETRIES",
			envVars: map[string]string{
				"LLM_MAX_RETRIES": "-1",
			},
			wantError: true,
		},
		{
			name: "Invalid LLM_CACHE_MAX_ENTRIES",
			envVars: map[string]string{
				"LLM_CACHE_MAX_ENTRIES": "invalid",
			},
			wantError: true,
		},
		{
			name: "Zero LLM_CACHE_MAX_ENTRIES",
			envVars: map[string]string{
				"LLM_CACHE_MAX_ENTRIES": "0",
			},
			wantError: true,
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
				os.Setenv(key, value)
			}
			
			// Set required environment variables for tests that don't want errors
			if !tt.wantError {
				os.Setenv("AUTH_ENABLED", "false")    // Disable auth to skip JWT requirement
				os.Setenv("CORS_ENABLED", "false")    // Disable CORS to skip origins requirement
			}

			// Clean up after test
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, originalValue)
					}
				}
				// Clean up test-specific environment variables
				if !tt.wantError {
					os.Unsetenv("AUTH_ENABLED")
					os.Unsetenv("CORS_ENABLED")
				}
			}()

			// Load configuration
			cfg, err := LoadLLMProcessorConfig()

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadLLMProcessorConfig() failed: %v", err)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(cfg, t)
			}
		})
	}
}

func TestLLMProcessorConfigDefaults(t *testing.T) {
	// Clear all relevant environment variables to test defaults
	envVars := []string{
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
		os.Unsetenv(key)
	}

	// Clean up after test
	defer func() {
		for key, originalValue := range originalEnv {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}()

	cfg := DefaultLLMProcessorConfig()

	// Test defaults
	if cfg.LLMTimeout != 15*time.Second {
		t.Errorf("Expected default LLMTimeout to be 15s, got %v", cfg.LLMTimeout)
	}

	if cfg.LLMMaxRetries != 2 {
		t.Errorf("Expected default LLMMaxRetries to be 2, got %v", cfg.LLMMaxRetries)
	}

	if cfg.CacheMaxEntries != 512 {
		t.Errorf("Expected default CacheMaxEntries to be 512, got %v", cfg.CacheMaxEntries)
	}

	if cfg.MaxRequestSize != 1048576 {
		t.Errorf("Expected default MaxRequestSize to be 1048576, got %v", cfg.MaxRequestSize)
	}

	if cfg.MetricsEnabled != false {
		t.Errorf("Expected default MetricsEnabled to be false, got %v", cfg.MetricsEnabled)
	}

	if len(cfg.MetricsAllowedIPs) != 0 {
		t.Errorf("Expected default MetricsAllowedIPs to be empty, got %v", cfg.MetricsAllowedIPs)
	}
}

func TestLLMProcessorConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *LLMProcessorConfig
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: &LLMProcessorConfig{
				LLMTimeout:        15 * time.Second,
				LLMMaxRetries:     2,
				CacheMaxEntries:   512,
				MaxRequestSize:    1048576,
				MetricsEnabled:    false,
				MetricsAllowedIPs: []string{},
				// Set required fields to pass validation
				LLMBackendType:     "rag",
				AuthEnabled:        false,
				APIKeyRequired:     false,
				TLSEnabled:        false,
				CORSEnabled:       false,
				AllowedOrigins:    []string{},
			},
		},
		{
			name: "Valid with metrics enabled and allowed IPs",
			config: &LLMProcessorConfig{
				LLMTimeout:        30 * time.Second,
				LLMMaxRetries:     3,
				CacheMaxEntries:   1024,
				MaxRequestSize:    2097152,
				MetricsEnabled:    true,
				MetricsAllowedIPs: []string{"127.0.0.1", "192.168.1.1"},
				// Set required fields
				LLMBackendType:     "rag",
				AuthEnabled:        false,
				APIKeyRequired:     false,
				TLSEnabled:        false,
				CORSEnabled:       false,
				AllowedOrigins:    []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLLMProcessorConfigMetricsAccess(t *testing.T) {
	cfg := &LLMProcessorConfig{
		MetricsEnabled:    true,
		MetricsAllowedIPs: []string{"192.168.1.100", "10.0.0.1"},
	}

	tests := []struct {
		name     string
		clientIP string
		want     bool
	}{
		{
			name:     "Allowed IP",
			clientIP: "192.168.1.100",
			want:     true,
		},
		{
			name:     "Another allowed IP",
			clientIP: "10.0.0.1",
			want:     true,
		},
		{
			name:     "Disallowed IP",
			clientIP: "192.168.1.101",
			want:     false,
		},
		{
			name:     "Invalid IP",
			clientIP: "invalid",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.IsMetricsAccessAllowed(tt.clientIP)
			if got != tt.want {
				t.Errorf("IsMetricsAccessAllowed(%s) = %v, want %v", tt.clientIP, got, tt.want)
			}
		})
	}
}

func TestLLMProcessorConfigMetricsDisabled(t *testing.T) {
	cfg := &LLMProcessorConfig{
		MetricsEnabled: false,
	}

	// When metrics are disabled, all access should be denied
	if cfg.IsMetricsAccessAllowed("127.0.0.1") {
		t.Errorf("IsMetricsAccessAllowed should return false when metrics are disabled")
	}
}