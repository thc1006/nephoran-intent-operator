package shared

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultHTTPTransportConfig(t *testing.T) {
	config := DefaultHTTPTransportConfig()

	if config.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns=100, got %d", config.MaxIdleConns)
	}
	if config.MaxIdleConnsPerHost != 50 {
		t.Errorf("Expected MaxIdleConnsPerHost=50, got %d", config.MaxIdleConnsPerHost)
	}
	if config.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected IdleConnTimeout=90s, got %v", config.IdleConnTimeout)
	}
	if config.DisableKeepAlives {
		t.Error("Expected DisableKeepAlives=false for connection pooling")
	}
}

func TestNewOptimizedHTTPTransport(t *testing.T) {
	tests := []struct {
		name   string
		config *HTTPTransportConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &HTTPTransportConfig{
				MaxIdleConns:        200,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     120 * time.Second,
			},
		},
		{
			name: "partial config fills defaults",
			config: &HTTPTransportConfig{
				MaxIdleConns: 150,
				// Other fields should use defaults
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewOptimizedHTTPTransport(tt.config)

			if transport == nil {
				t.Fatal("Expected non-nil transport")
			}

			// Verify transport has reasonable settings
			if transport.MaxIdleConns < 100 {
				t.Errorf("MaxIdleConns too low: %d", transport.MaxIdleConns)
			}
			if transport.MaxIdleConnsPerHost < 50 {
				t.Errorf("MaxIdleConnsPerHost too low: %d", transport.MaxIdleConnsPerHost)
			}
			if transport.IdleConnTimeout < 60*time.Second {
				t.Errorf("IdleConnTimeout too low: %v", transport.IdleConnTimeout)
			}

			// Verify connection pooling is enabled
			if transport.DisableKeepAlives {
				t.Error("Connection pooling should be enabled (DisableKeepAlives=false)")
			}
		})
	}
}

func TestNewOptimizedHTTPClient(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		transportConfig *HTTPTransportConfig
		expectedTimeout time.Duration
	}{
		{
			name:            "default timeout",
			timeout:         0,
			transportConfig: nil,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "custom timeout",
			timeout:         60 * time.Second,
			transportConfig: nil,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:    "custom transport config",
			timeout: 45 * time.Second,
			transportConfig: &HTTPTransportConfig{
				MaxIdleConns: 200,
			},
			expectedTimeout: 45 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOptimizedHTTPClient(tt.timeout, tt.transportConfig)

			if client == nil {
				t.Fatal("Expected non-nil client")
			}

			if client.Timeout != tt.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tt.expectedTimeout, client.Timeout)
			}

			// Verify transport is configured
			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Fatal("Expected *http.Transport")
			}

			if transport.MaxIdleConns < 100 {
				t.Errorf("MaxIdleConns too low: %d", transport.MaxIdleConns)
			}
			if transport.MaxIdleConnsPerHost < 50 {
				t.Errorf("MaxIdleConnsPerHost too low: %d", transport.MaxIdleConnsPerHost)
			}
		})
	}
}

func TestHTTPTransportDefaults(t *testing.T) {
	// Test that zero values in config get replaced with defaults
	config := &HTTPTransportConfig{} // All zero values

	transport := NewOptimizedHTTPTransport(config)

	if transport.MaxIdleConns != 100 {
		t.Errorf("Expected default MaxIdleConns=100, got %d", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 50 {
		t.Errorf("Expected default MaxIdleConnsPerHost=50, got %d", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("Expected default IdleConnTimeout=90s, got %v", transport.IdleConnTimeout)
	}
	if transport.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("Expected default TLSHandshakeTimeout=10s, got %v", transport.TLSHandshakeTimeout)
	}
}

func BenchmarkNewOptimizedHTTPClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewOptimizedHTTPClient(30*time.Second, nil)
	}
}

func BenchmarkNewOptimizedHTTPTransport(b *testing.B) {
	config := DefaultHTTPTransportConfig()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewOptimizedHTTPTransport(config)
	}
}
