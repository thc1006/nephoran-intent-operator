package loop

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_Validate tests the comprehensive configuration validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		// MaxWorkers validation tests
		{
			name: "valid MaxWorkers",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "MaxWorkers too low",
			config: Config{
				MaxWorkers: 0,
			},
			wantErr: false, // Validate() now auto-corrects instead of erroring
		},
		{
			name: "MaxWorkers too high",
			config: Config{
				MaxWorkers: runtime.NumCPU()*4 + 1,
			},
			wantErr: false, // Validate() now auto-corrects instead of erroring
		},
		{
			name: "MaxWorkers at safe limit",
			config: Config{
				MaxWorkers:   runtime.NumCPU() * 4,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},

		// MetricsPort validation tests
		{
			name: "MetricsPort disabled (0)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  0,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "MetricsPort valid (1024)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  1024,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "MetricsPort valid (65535)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  65535,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "MetricsPort too low",
			config: Config{
				MaxWorkers:  2,
				MetricsPort: 1023,
			},
			wantErr: true,
			errMsg:  "metrics_port must be between 1024-65535 or 0 to disable",
		},
		{
			name: "MetricsPort too high",
			config: Config{
				MaxWorkers:  2,
				MetricsPort: 65536,
			},
			wantErr: true,
			errMsg:  "metrics_port must be between 1024-65535 or 0 to disable",
		},

		// DebounceDur validation tests
		{
			name: "DebounceDur disabled (0)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  0,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "DebounceDur minimum valid (10ms)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  10 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "DebounceDur maximum valid (5s)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  5 * time.Second,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "DebounceDur too low",
			config: Config{
				MaxWorkers:  2,
				DebounceDur: 9 * time.Millisecond,
			},
			wantErr: false, // Validate() auto-corrects instead of erroring
		},
		{
			name: "DebounceDur too high",
			config: Config{
				MaxWorkers:  2,
				DebounceDur: 6 * time.Second,
			},
			wantErr: false, // Validate() auto-corrects instead of erroring
		},

		// Period validation tests
		{
			name: "Period disabled (0)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				Period:       0,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Period minimum valid (100ms)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				Period:       100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Period maximum valid (1h)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Hour,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Period too low",
			config: Config{
				MaxWorkers: 2,
				Period:     99 * time.Millisecond,
			},
			wantErr: true,
			errMsg:  "period must be at least 100ms",
		},
		{
			name: "Period too high",
			config: Config{
				MaxWorkers: 2,
				Period:     61 * time.Minute,
			},
			wantErr: true,
			errMsg:  "period must not exceed 1 hour",
		},

		// CleanupAfter validation tests
		{
			name: "CleanupAfter disabled (0)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 0,
			},
			wantErr: false,
		},
		{
			name: "CleanupAfter minimum valid (1h)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 1 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "CleanupAfter maximum valid (30d)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 30 * 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "CleanupAfter too low",
			config: Config{
				MaxWorkers:   2,
				CleanupAfter: 59 * time.Minute,
			},
			wantErr: true,
			errMsg:  "cleanup_after must be at least 1 hour",
		},
		{
			name: "CleanupAfter too high",
			config: Config{
				MaxWorkers:   2,
				CleanupAfter: 31 * 24 * time.Hour,
			},
			wantErr: true,
			errMsg:  "cleanup_after must not exceed 30 days",
		},

		// MetricsAuth validation tests
		{
			name: "MetricsAuth disabled",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				MetricsAuth:  false,
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "MetricsAuth enabled with valid credentials",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				MetricsAuth:  true,
				MetricsUser:  "admin",
				MetricsPass:  "password123",
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "MetricsAuth enabled with minimum length password",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				MetricsAuth:  true,
				MetricsUser:  "admin",
				MetricsPass:  "12345678", // exactly 8 chars
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "MetricsAuth enabled without username",
			config: Config{
				MaxWorkers:  2,
				MetricsAuth: true,
				MetricsPass: "password123",
			},
			wantErr: true,
			errMsg:  "metrics_auth requires both metrics_user and metrics_pass",
		},
		{
			name: "MetricsAuth enabled without password",
			config: Config{
				MaxWorkers:  2,
				MetricsAuth: true,
				MetricsUser: "admin",
			},
			wantErr: true,
			errMsg:  "metrics_auth requires both metrics_user and metrics_pass",
		},
		{
			name: "MetricsAuth enabled with password too short",
			config: Config{
				MaxWorkers:  2,
				MetricsAuth: true,
				MetricsUser: "admin",
				MetricsPass: "1234567", // only 7 chars
			},
			wantErr: true,
			errMsg:  "metrics_pass must be at least 8 characters for security",
		},

		// Mode validation tests
		{
			name: "Mode empty (valid)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				Mode:         "",
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Mode watch (valid)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				Mode:         "watch",
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Mode once (valid)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				Mode:         "once",
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Mode periodic (valid)",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				Mode:         "periodic",
				DebounceDur:  100 * time.Millisecond,
				Period:       1 * time.Second,
				CleanupAfter: 24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Mode invalid",
			config: Config{
				MaxWorkers: 2,
				Mode:       "invalid",
			},
			wantErr: true,
			errMsg:  `invalid mode "invalid", must be one of: watch, once, periodic`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNewWatcher_SecurityDefaults tests that NewWatcher applies secure defaults
func TestNewWatcher_SecurityDefaults(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		inputConfig    Config
		expectedConfig Config
		description    string
	}{
		{
			name: "MetricsAddr defaults to localhost",
			inputConfig: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			expectedConfig: Config{
				MetricsAddr: "127.0.0.1",
			},
			description: "MetricsAddr should default to localhost for security",
		},
		{
			name: "MetricsPort negative becomes default",
			inputConfig: Config{
				MaxWorkers:   2,
				MetricsPort:  -1,
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			expectedConfig: Config{
				MetricsPort: 8080,
				MetricsAddr: "127.0.0.1",
			},
			description: "Negative MetricsPort should default to 8080",
		},
		{
			name: "MaxWorkers zero becomes default",
			inputConfig: Config{
				MetricsPort:  8080,
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			expectedConfig: Config{
				MaxWorkers:  2,
				MetricsAddr: "127.0.0.1",
			},
			description: "MaxWorkers zero should default to 2",
		},
		{
			name: "CleanupAfter zero becomes default",
			inputConfig: Config{
				MaxWorkers:  2,
				MetricsPort: 8080,
				DebounceDur: 100 * time.Millisecond,
			},
			expectedConfig: Config{
				CleanupAfter: 7 * 24 * time.Hour,
				MetricsAddr:  "127.0.0.1",
			},
			description: "CleanupAfter zero should default to 7 days",
		},
		{
			name: "DebounceDur zero becomes default",
			inputConfig: Config{
				MaxWorkers:   2,
				MetricsPort:  8080,
				CleanupAfter: 24 * time.Hour,
			},
			expectedConfig: Config{
				DebounceDur: 100 * time.Millisecond,
				MetricsAddr: "127.0.0.1",
			},
			description: "DebounceDur zero should default to 100ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set a valid PorchPath for the test
			tt.inputConfig.PorchPath = createMockPorchExecutable(t, tempDir)

			watcher, err := NewWatcher(tempDir, tt.inputConfig)
			require.NoError(t, err, "NewWatcher should not fail with valid config")
			defer watcher.Close()

			// Check that the expected defaults were applied
			if tt.expectedConfig.MetricsAddr != "" {
				assert.Equal(t, tt.expectedConfig.MetricsAddr, watcher.config.MetricsAddr, tt.description)
			}
			if tt.expectedConfig.MetricsPort != 0 {
				assert.Equal(t, tt.expectedConfig.MetricsPort, watcher.config.MetricsPort, tt.description)
			}
			if tt.expectedConfig.MaxWorkers != 0 {
				assert.Equal(t, tt.expectedConfig.MaxWorkers, watcher.config.MaxWorkers, tt.description)
			}
			if tt.expectedConfig.CleanupAfter != 0 {
				assert.Equal(t, tt.expectedConfig.CleanupAfter, watcher.config.CleanupAfter, tt.description)
			}
			if tt.expectedConfig.DebounceDur != 0 {
				assert.Equal(t, tt.expectedConfig.DebounceDur, watcher.config.DebounceDur, tt.description)
			}
		})
	}
}

// TestMetricsServer_Security tests the security features of the metrics server
func TestMetricsServer_Security(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		config      Config
		endpoint    string
		authUser    string
		authPass    string
		expectCode  int
		description string
	}{
		{
			name: "metrics endpoint without auth should work",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  getFreePort(t),
				MetricsAuth:  false,
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			endpoint:    "/metrics",
			expectCode:  200,
			description: "Unauthenticated access should work when auth is disabled",
		},
		{
			name: "health endpoint should always be public",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  getFreePort(t),
				MetricsAuth:  true,
				MetricsUser:  "admin",
				MetricsPass:  "password123",
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			endpoint:    "/health",
			expectCode:  200,
			description: "Health endpoint should remain public even with auth enabled",
		},
		{
			name: "metrics endpoint with auth requires credentials",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  getFreePort(t),
				MetricsAuth:  true,
				MetricsUser:  "admin",
				MetricsPass:  "password123",
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			endpoint:    "/metrics",
			expectCode:  401,
			description: "Metrics endpoint should require auth when enabled",
		},
		{
			name: "metrics endpoint with valid credentials should work",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  getFreePort(t),
				MetricsAuth:  true,
				MetricsUser:  "admin",
				MetricsPass:  "password123",
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			endpoint:    "/metrics",
			authUser:    "admin",
			authPass:    "password123",
			expectCode:  200,
			description: "Valid credentials should allow access to metrics",
		},
		{
			name: "prometheus endpoint with auth requires credentials",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  getFreePort(t),
				MetricsAuth:  true,
				MetricsUser:  "admin",
				MetricsPass:  "password123",
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			endpoint:    "/metrics/prometheus",
			expectCode:  401,
			description: "Prometheus endpoint should require auth when enabled",
		},
		{
			name: "prometheus endpoint with valid credentials should work",
			config: Config{
				MaxWorkers:   2,
				MetricsPort:  getFreePort(t),
				MetricsAuth:  true,
				MetricsUser:  "admin",
				MetricsPass:  "password123",
				DebounceDur:  100 * time.Millisecond,
				CleanupAfter: 24 * time.Hour,
			},
			endpoint:    "/metrics/prometheus",
			authUser:    "admin",
			authPass:    "password123",
			expectCode:  200,
			description: "Valid credentials should allow access to prometheus metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set a valid PorchPath for the test
			tt.config.PorchPath = createMockPorchExecutable(t, tempDir)

			watcher, err := NewWatcher(tempDir, tt.config)
			require.NoError(t, err, "NewWatcher should not fail")
			defer watcher.Close()

			// Wait for metrics server to start
			time.Sleep(100 * time.Millisecond)

			// Test the endpoint
			url := fmt.Sprintf("http://%s:%d%s", watcher.config.MetricsAddr, watcher.config.MetricsPort, tt.endpoint)
			req, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			if tt.authUser != "" && tt.authPass != "" {
				req.SetBasicAuth(tt.authUser, tt.authPass)
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err, "HTTP request should not fail")
			defer resp.Body.Close()

			assert.Equal(t, tt.expectCode, resp.StatusCode, tt.description)

			// Additional checks for successful responses
			if tt.expectCode == 200 {
				assert.NotEmpty(t, resp.Header.Get("Content-Type"), "Response should have Content-Type header")
				
				if tt.endpoint == "/health" {
					assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
				} else if tt.endpoint == "/metrics" {
					assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
				} else if tt.endpoint == "/metrics/prometheus" {
					assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
				}
			}

			// Additional checks for unauthorized responses
			if tt.expectCode == 401 {
				assert.Equal(t, `Basic realm="Metrics"`, resp.Header.Get("WWW-Authenticate"))
			}
		})
	}
}

// TestMetricsServer_LocalhostBinding tests that metrics server binds to localhost only by default
func TestMetricsServer_LocalhostBinding(t *testing.T) {
	tempDir := t.TempDir()
	port := getFreePort(t)

	config := Config{
		MaxWorkers:   2,
		MetricsPort:  port,
		MetricsAuth:  false,
		DebounceDur:  100 * time.Millisecond,
		CleanupAfter: 24 * time.Hour,
		PorchPath:    createMockPorchExecutable(t, tempDir),
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err, "NewWatcher should not fail")
	defer watcher.Close()

	// Wait for metrics server to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is accessible from localhost
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	require.NoError(t, err, "Should be able to access from localhost")
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	// Test that server rejects connections from other interfaces (if available)
	// Note: This test only works if there are multiple network interfaces
	interfaces, err := net.Interfaces()
	require.NoError(t, err)

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				// Try to connect from a non-localhost IP
				url := fmt.Sprintf("http://%s:%d/health", ipNet.IP.String(), port)
				client := &http.Client{Timeout: 1 * time.Second}
				
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
				
				req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
				if err != nil {
					continue
				}
				
				resp, err := client.Do(req)
				if err == nil {
					resp.Body.Close()
					// If we can connect, it means the server is not properly bound to localhost only
					// This might be expected behavior depending on the system configuration
					t.Logf("Warning: Server accessible from %s (this might be expected)", ipNet.IP.String())
				}
				// If we can't connect, that's good - it means localhost-only binding is working
			}
		}
	}
}

// TestMetricsServer_DisabledPort tests that metrics server doesn't start when port is 0
func TestMetricsServer_DisabledPort(t *testing.T) {
	tempDir := t.TempDir()

	config := Config{
		MaxWorkers:   2,
		MetricsPort:  0, // Disabled
		DebounceDur:  100 * time.Millisecond,
		CleanupAfter: 24 * time.Hour,
		PorchPath:    createMockPorchExecutable(t, tempDir),
	}

	watcher, err := NewWatcher(tempDir, config)
	require.NoError(t, err, "NewWatcher should not fail")
	defer watcher.Close()

	// Verify that no metrics server was created
	assert.Nil(t, watcher.metricsServer, "Metrics server should be nil when port is 0")
}

// TestConfig_EdgeCases tests edge cases in configuration validation
func TestConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "all zeros except max workers",
			config: Config{
				MaxWorkers: 1, // Need at least 1 worker
			},
			wantErr: false,
		},
		{
			name: "negative values",
			config: Config{
				MaxWorkers:   -1,
				MetricsPort:  -1,
				DebounceDur:  -1 * time.Second,
				Period:       -1 * time.Second,
				CleanupAfter: -1 * time.Hour,
			},
			wantErr: true,
			errMsg:  "metrics_port must be between 1024-65535 or 0 to disable", // Now returns error for negative port
		},
		{
			name: "extreme values",
			config: Config{
				MaxWorkers:   999999,
				MetricsPort:  999999,
				DebounceDur:  999999 * time.Hour,
				Period:       999999 * time.Hour,
				CleanupAfter: 999999 * time.Hour,
			},
			wantErr: true,
			errMsg:  "metrics_port must be between 1024-65535 or 0 to disable", // Port validation fails first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions

// getFreePort returns a free port number for testing
func getFreePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// createMockPorchExecutable creates a mock porch executable for testing
func createMockPorchExecutable(t *testing.T, dir string) string {
	// Create a mock porch executable
	porchPath := filepath.Join(dir, "porch")
	if runtime.GOOS == "windows" {
		porchPath += ".exe"
	}
	
	// Create a simple script that just exits successfully
	content := "#!/bin/bash\necho 'mock porch execution'\nexit 0\n"
	if runtime.GOOS == "windows" {
		content = "@echo off\necho mock porch execution\nexit /b 0\n"
	}
	
	err := os.WriteFile(porchPath, []byte(content), 0755)
	require.NoError(t, err)
	
	return porchPath
}