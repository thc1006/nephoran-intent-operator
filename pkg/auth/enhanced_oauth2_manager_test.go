package auth

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnhancedOAuth2Manager_ConfigureAllRoutes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name               string
		config             *EnhancedOAuth2ManagerConfig
		expectedRoutes     []string
		unexpectedRoutes   []string
		publicAccess       map[string]bool // route -> should be publicly accessible
	}{
		{
			name: "Auth disabled - all routes public",
			config: &EnhancedOAuth2ManagerConfig{
				Enabled:                false,
				RequireAuth:            false,
				HealthEndpointsEnabled: true,
				ExposeMetricsPublicly:  true,
				StreamingEnabled:       true,
			},
			expectedRoutes: []string{
				"/healthz", "/readyz", "/metrics", "/process", "/status", "/circuit-breaker/status", "/stream",
			},
			publicAccess: map[string]bool{
				"/healthz":                    true,
				"/readyz":                     true,
				"/metrics":                    true,
				"/process":                    true,
				"/status":                     true,
				"/circuit-breaker/status":     true,
				"/stream":                     true,
			},
		},
		{
			name: "Auth enabled with public metrics",
			config: &EnhancedOAuth2ManagerConfig{
				Enabled:                true,
				RequireAuth:            true,
				JWTSecretKey:           "test-secret-key",
				AuthConfigFile:         "", // Will be mocked
				HealthEndpointsEnabled: true,
				ExposeMetricsPublicly:  true,
				StreamingEnabled:       false, // Streaming disabled
			},
			expectedRoutes: []string{
				"/healthz", "/readyz", "/metrics", 
				"/auth/login/{provider}", "/auth/callback/{provider}", "/auth/refresh", "/auth/logout", "/auth/userinfo",
			},
			unexpectedRoutes: []string{
				"/stream", // Should not exist when streaming disabled
			},
			publicAccess: map[string]bool{
				"/healthz":  true,
				"/readyz":   true, 
				"/metrics":  true,
				"/process":  false, // Protected
				"/status":   false, // Protected admin endpoint
			},
		},
		{
			name: "Metrics with IP allowlist",
			config: &EnhancedOAuth2ManagerConfig{
				Enabled:                false,
				HealthEndpointsEnabled: true,
				ExposeMetricsPublicly:  false,
				MetricsAllowedCIDRs:    []string{"127.0.0.0/8", "192.168.0.0/16"},
			},
			expectedRoutes: []string{
				"/healthz", "/readyz", "/metrics",
			},
			publicAccess: map[string]bool{
				"/healthz": true,
				"/readyz":  true,
				"/metrics": false, // IP restricted
			},
		},
		{
			name: "Health endpoints disabled", 
			config: &EnhancedOAuth2ManagerConfig{
				Enabled:                false,
				HealthEndpointsEnabled: false, // Disabled
				ExposeMetricsPublicly:  true,
			},
			expectedRoutes: []string{
				"/metrics", "/process", "/status", "/circuit-breaker/status",
			},
			unexpectedRoutes: []string{
				"/healthz", "/readyz", // Should not exist
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create manager (skip OAuth2 setup for test simplicity)
			manager := &EnhancedOAuth2Manager{
				config: tt.config,
				logger: logger,
				// authMiddleware will be nil for most tests
			}

			// Create test router
			router := mux.NewRouter()

			// Create mock handlers
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			}

			publicHandlers := &PublicRouteHandlers{
				Health:  mockHandler,
				Ready:   mockHandler,
				Metrics: mockHandler,
			}

			protectedHandlers := &ProtectedRouteHandlers{
				ProcessIntent:        mockHandler,
				Status:               mockHandler,
				CircuitBreakerStatus: mockHandler,
				StreamingHandler:     mockHandler,
			}

			// Configure all routes
			err := manager.ConfigureAllRoutes(router, publicHandlers, protectedHandlers)
			assert.NoError(t, err)

			// Test that expected routes exist
			for _, route := range tt.expectedRoutes {
				t.Run("route_exists_"+route, func(t *testing.T) {
					req := httptest.NewRequest("GET", route, nil)
					if route == "/process" || route == "/stream" || route == "/auth/refresh" || route == "/auth/logout" {
						req = httptest.NewRequest("POST", route, nil)
					}
					
					rr := httptest.NewRecorder()
					router.ServeHTTP(rr, req)
					
					// Route should exist (not 404)
					assert.NotEqual(t, http.StatusNotFound, rr.Code, 
						"Route %s should exist but returned 404", route)
				})
			}

			// Test that unexpected routes don't exist
			for _, route := range tt.unexpectedRoutes {
				t.Run("route_not_exists_"+route, func(t *testing.T) {
					req := httptest.NewRequest("GET", route, nil)
					rr := httptest.NewRecorder()
					router.ServeHTTP(rr, req)
					
					// Route should not exist (404)
					assert.Equal(t, http.StatusNotFound, rr.Code,
						"Route %s should not exist but was found", route)
				})
			}

			// Test public access (simplified - would need more complex auth setup for full test)
			for route, shouldBePublic := range tt.publicAccess {
				t.Run("access_"+route, func(t *testing.T) {
					method := "GET"
					if route == "/process" || route == "/stream" {
						method = "POST"
					}
					
					req := httptest.NewRequest(method, route, nil)
					rr := httptest.NewRecorder()
					router.ServeHTTP(rr, req)
					
					if shouldBePublic {
						// Public routes should return OK (200) or at least not 401/403 for auth
						assert.NotContains(t, []int{http.StatusUnauthorized, http.StatusForbidden}, 
							rr.Code, "Public route %s should be accessible", route)
					}
					// Note: Testing protected routes would require setting up proper auth middleware
				})
			}
		})
	}
}

func TestEnhancedOAuth2Manager_IPAllowlist(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := &EnhancedOAuth2ManagerConfig{
		Enabled:               false,
		ExposeMetricsPublicly: false,
		MetricsAllowedCIDRs:   []string{"127.0.0.0/8", "192.168.1.0/24"},
	}

	manager := &EnhancedOAuth2Manager{
		config: config,
		logger: logger,
	}

	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("metrics data"))
	}

	// Create IP allowlist handler
	protectedHandler := manager.createIPAllowlistHandler(mockHandler, config.MetricsAllowedCIDRs)

	tests := []struct {
		name           string
		clientIP       string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "Allowed IP - localhost",
			clientIP:       "127.0.0.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Allowed IP - local network", 
			clientIP:       "192.168.1.100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Blocked IP - external",
			clientIP:       "8.8.8.8",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:     "X-Forwarded-For header - allowed",
			clientIP: "8.8.8.8", // This will be overridden by header
			headers: map[string]string{
				"X-Forwarded-For": "127.0.0.1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "X-Real-IP header - blocked",
			clientIP: "127.0.0.1", // This will be overridden by header
			headers: map[string]string{
				"X-Real-IP": "8.8.8.8",
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/metrics", nil)
			req.RemoteAddr = tt.clientIP + ":12345"
			
			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			rr := httptest.NewRecorder()
			protectedHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code,
				"Expected status %d for IP %s, got %d", tt.expectedStatus, tt.clientIP, rr.Code)
		})
	}
}

func TestEnhancedConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *EnhancedOAuth2ManagerConfig
		expectError bool
		errorCode   string
	}{
		{
			name: "Valid config - auth disabled",
			config: &EnhancedOAuth2ManagerConfig{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "Valid config - auth enabled with JWT secret",
			config: &EnhancedOAuth2ManagerConfig{
				Enabled:      true,
				JWTSecretKey: "valid-secret",
			},
			expectError: false,
		},
		{
			name: "Invalid config - auth enabled without JWT secret",
			config: &EnhancedOAuth2ManagerConfig{
				Enabled:      true,
				JWTSecretKey: "",
			},
			expectError: true,
			errorCode:   "missing_jwt_secret",
		},
		{
			name: "Invalid config - bad CIDR",
			config: &EnhancedOAuth2ManagerConfig{
				Enabled:               false,
				ExposeMetricsPublicly: false,
				MetricsAllowedCIDRs:   []string{"invalid-cidr"},
			},
			expectError: true,
			errorCode:   "invalid_metrics_cidr",
		},
		{
			name: "Valid config - good CIDR",
			config: &EnhancedOAuth2ManagerConfig{
				Enabled:               false,
				ExposeMetricsPublicly: false,
				MetricsAllowedCIDRs:   []string{"127.0.0.0/8", "192.168.0.0/16"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				require.Error(t, err)
				if tt.errorCode != "" {
					if authErr, ok := err.(*AuthError); ok {
						assert.Equal(t, tt.errorCode, authErr.Code)
					}
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}