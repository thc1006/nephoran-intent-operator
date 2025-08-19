package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/testutil"
)

func TestAuthMiddleware(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	// Create test user and tokens
	user := uf.CreateBasicUser()
	validToken, err := jwtManager.GenerateToken(user, nil)
	require.NoError(t, err)

	validSession, err := sessionManager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	// Create middleware
	middleware := NewAuthMiddleware(&AuthMiddlewareConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RequireAuth:    true,
		AllowedPaths:   []string{"/health", "/public"},
		HeaderName:     "Authorization",
		CookieName:     "session",
		ContextKey:     "user",
	})

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectStatus  int
		expectUser    bool
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Valid JWT token",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/protected", nil)
				req.Header.Set("Authorization", "Bearer "+validToken)
				return req
			},
			expectStatus: http.StatusOK,
			expectUser:   true,
		},
		{
			name: "Valid session cookie",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/protected", nil)
				req.AddCookie(&http.Cookie{
					Name:  "session",
					Value: validSession.ID,
				})
				return req
			},
			expectStatus: http.StatusOK,
			expectUser:   true,
		},
		{
			name: "Missing authentication",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/protected", nil)
			},
			expectStatus: http.StatusUnauthorized,
			expectUser:   false,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Missing authentication", response["error"])
			},
		},
		{
			name: "Invalid JWT token",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/protected", nil)
				req.Header.Set("Authorization", "Bearer invalid-token")
				return req
			},
			expectStatus: http.StatusUnauthorized,
			expectUser:   false,
		},
		{
			name: "Malformed authorization header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/protected", nil)
				req.Header.Set("Authorization", "NotBearer "+validToken)
				return req
			},
			expectStatus: http.StatusUnauthorized,
			expectUser:   false,
		},
		{
			name: "Public path allowed",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/public", nil)
			},
			expectStatus: http.StatusOK,
			expectUser:   false,
		},
		{
			name: "Health check allowed",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/health", nil)
			},
			expectStatus: http.StatusOK,
			expectUser:   false,
		},
		{
			name: "Expired JWT token",
			setupRequest: func() *http.Request {
				// Create expired token
				expiredToken := tc.CreateTestToken(map[string]interface{}{
					"iss": "test-issuer",
					"sub": user.Subject,
					"exp": time.Now().Add(-time.Hour).Unix(),
					"iat": time.Now().Add(-2 * time.Hour).Unix(),
				})

				req := httptest.NewRequest("GET", "/protected", nil)
				req.Header.Set("Authorization", "Bearer "+expiredToken)
				return req
			},
			expectStatus: http.StatusUnauthorized,
			expectUser:   false,
		},
	}

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userCtx := r.Context().Value("user")
		if userCtx != nil {
			user := userCtx.(*UserContext)
			response := map[string]string{
				"message": "success",
				"user_id": user.UserID,
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("public access"))
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			// Apply middleware
			handler := middleware.Middleware(testHandler)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			if tt.expectUser && tt.expectStatus == http.StatusOK {
				// Verify user context was set
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "success", response["message"])
				assert.NotEmpty(t, response["user_id"])
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestRBACMiddleware(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	rbacManager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()
	pf := testutil.NewPermissionFactory()

	// Setup RBAC data
	ctx := context.Background()

	// Create permissions
	readPerm := pf.CreateResourcePermissions("api", []string{"read"})[0]
	createdReadPerm, err := rbacManager.CreatePermission(ctx, readPerm)
	require.NoError(t, err)

	writePerm := pf.CreateResourcePermissions("api", []string{"write"})[0]
	createdWritePerm, err := rbacManager.CreatePermission(ctx, writePerm)
	require.NoError(t, err)

	adminPerm := pf.CreateResourcePermissions("admin", []string{"*"})[0]
	createdAdminPerm, err := rbacManager.CreatePermission(ctx, adminPerm)
	require.NoError(t, err)

	// Create roles
	readerRole := rf.CreateRoleWithPermissions([]string{createdReadPerm.ID})
	readerRole.Name = "reader"
	createdReaderRole, err := rbacManager.CreateRole(ctx, readerRole)
	require.NoError(t, err)

	writerRole := rf.CreateRoleWithPermissions([]string{createdReadPerm.ID, createdWritePerm.ID})
	writerRole.Name = "writer"
	createdWriterRole, err := rbacManager.CreateRole(ctx, writerRole)
	require.NoError(t, err)

	adminRole := rf.CreateRoleWithPermissions([]string{createdAdminPerm.ID})
	adminRole.Name = "admin"
	createdAdminRole, err := rbacManager.CreateRole(ctx, adminRole)
	require.NoError(t, err)

	// Assign roles to users
	err = rbacManager.AssignRoleToUser(ctx, "reader-user", createdReaderRole.ID)
	require.NoError(t, err)

	err = rbacManager.AssignRoleToUser(ctx, "writer-user", createdWriterRole.ID)
	require.NoError(t, err)

	err = rbacManager.AssignRoleToUser(ctx, "admin-user", createdAdminRole.ID)
	require.NoError(t, err)

	// Create RBAC middleware
	middleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager: rbacManager,
		ResourceExtractor: func(r *http.Request) string {
			if strings.HasPrefix(r.URL.Path, "/admin") {
				return "admin"
			}
			return "api"
		},
		ActionExtractor: func(r *http.Request) string {
			switch r.Method {
			case http.MethodGet:
				return "read"
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				return "write"
			case http.MethodDelete:
				return "delete"
			default:
				return "read"
			}
		},
		UserIDExtractor: func(r *http.Request) string {
			if userCtx := r.Context().Value("user"); userCtx != nil {
				return userCtx.(*UserContext).UserID
			}
			return ""
		},
	})

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectStatus  int
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Reader can read API",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/data", nil)
				req = req.WithContext(context.WithValue(req.Context(), "user", &UserContext{
					UserID: "reader-user",
				}))
				return req
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Reader cannot write to API",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/api/data", nil)
				req = req.WithContext(context.WithValue(req.Context(), "user", &UserContext{
					UserID: "reader-user",
				}))
				return req
			},
			expectStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Insufficient permissions", response["error"])
			},
		},
		{
			name: "Writer can write to API",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/api/data", nil)
				req = req.WithContext(context.WithValue(req.Context(), "user", &UserContext{
					UserID: "writer-user",
				}))
				return req
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Writer cannot access admin endpoints",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/admin/users", nil)
				req = req.WithContext(context.WithValue(req.Context(), "user", &UserContext{
					UserID: "writer-user",
				}))
				return req
			},
			expectStatus: http.StatusForbidden,
		},
		{
			name: "Admin can access admin endpoints",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/admin/users", nil)
				req = req.WithContext(context.WithValue(req.Context(), "user", &UserContext{
					UserID: "admin-user",
				}))
				return req
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "No user context",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/api/data", nil)
			},
			expectStatus: http.StatusForbidden,
		},
	}

	// Test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authorized"))
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			handler := middleware.Middleware(testHandler)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	middleware := NewCORSMiddleware(&CORSConfig{
		AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders:   []string{"X-Total-Count", "X-Page-Count"},
		AllowCredentials: true,
		MaxAge:           3600,
	})

	tests := []struct {
		name         string
		setupRequest func() *http.Request
		expectStatus int
		checkHeaders func(*testing.T, http.Header)
	}{
		{
			name: "Simple CORS request",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/data", nil)
				req.Header.Set("Origin", "https://example.com")
				return req
			},
			expectStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, headers http.Header) {
				assert.Equal(t, "https://example.com", headers.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "true", headers.Get("Access-Control-Allow-Credentials"))
				assert.Equal(t, "X-Total-Count,X-Page-Count", headers.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "Preflight CORS request",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("OPTIONS", "/api/data", nil)
				req.Header.Set("Origin", "https://app.example.com")
				req.Header.Set("Access-Control-Request-Method", "POST")
				req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")
				return req
			},
			expectStatus: http.StatusNoContent,
			checkHeaders: func(t *testing.T, headers http.Header) {
				assert.Equal(t, "https://app.example.com", headers.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "GET,POST,PUT,DELETE,OPTIONS", headers.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Content-Type,Authorization,X-Requested-With", headers.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "3600", headers.Get("Access-Control-Max-Age"))
			},
		},
		{
			name: "Disallowed origin",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/data", nil)
				req.Header.Set("Origin", "https://malicious.com")
				return req
			},
			expectStatus: http.StatusForbidden,
		},
		{
			name: "No origin header",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/api/data", nil)
			},
			expectStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, headers http.Header) {
				assert.Empty(t, headers.Get("Access-Control-Allow-Origin"))
			},
		},
		{
			name: "Preflight with disallowed method",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("OPTIONS", "/api/data", nil)
				req.Header.Set("Origin", "https://example.com")
				req.Header.Set("Access-Control-Request-Method", "PATCH")
				return req
			},
			expectStatus: http.StatusForbidden,
		},
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			handler := middleware.Middleware(testHandler)
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)

			if tt.checkHeaders != nil {
				tt.checkHeaders(t, w.Header())
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	middleware := NewRateLimitMiddleware(&RateLimitConfig{
		RequestsPerMinute: 5,
		BurstSize:         2,
		KeyGenerator: func(r *http.Request) string {
			// Use IP address as key
			return r.RemoteAddr
		},
		OnLimitExceeded: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded",
			})
		},
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name           string
		requestCount   int
		expectStatuses []int
	}{
		{
			name:         "Within rate limit",
			requestCount: 2,
			expectStatuses: []int{
				http.StatusOK,
				http.StatusOK,
			},
		},
		{
			name:         "Exceed burst limit",
			requestCount: 4,
			expectStatuses: []int{
				http.StatusOK,
				http.StatusOK,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.Middleware(testHandler)

			for i := 0; i < tt.requestCount; i++ {
				req := httptest.NewRequest("GET", "/api/data", nil)
				req.RemoteAddr = "192.168.1.100:12345" // Same IP for all requests
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)
				assert.Equal(t, tt.expectStatuses[i], w.Code, "Request %d status mismatch", i+1)
			}
		})
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	middleware := NewSecurityHeadersMiddleware(&SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'",
		XFrameOptions:         "DENY",
		XContentTypeOptions:   "nosniff",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
		RemoveServerHeader:    true,
		CustomHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "nginx/1.20") // Should be removed
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler := middleware.Middleware(testHandler)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()
	assert.Equal(t, "default-src 'self'; script-src 'self' 'unsafe-inline'", headers.Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))
	assert.Equal(t, "max-age=31536000; includeSubDomains; preload", headers.Get("Strict-Transport-Security"))
	assert.Equal(t, "custom-value", headers.Get("X-Custom-Header"))
	assert.Empty(t, headers.Get("Server")) // Should be removed
}

func TestRequestLoggingMiddleware(t *testing.T) {
	var logEntries []string
	mockLogger := func(entry string) {
		logEntries = append(logEntries, entry)
	}

	middleware := NewRequestLoggingMiddleware(&RequestLoggingConfig{
		Logger:           mockLogger,
		LogHeaders:       true,
		LogBody:          true,
		MaxBodySize:      1024,
		SkipPaths:        []string{"/health"},
		SensitiveHeaders: []string{"Authorization", "Cookie"},
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate processing time
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	})

	tests := []struct {
		name         string
		setupRequest func() *http.Request
		expectLogs   bool
		checkLogs    func(*testing.T, []string)
	}{
		{
			name: "Normal request logging",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/api/data", strings.NewReader(`{"key": "value"}`))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer secret-token")
				req.Header.Set("User-Agent", "test-client")
				return req
			},
			expectLogs: true,
			checkLogs: func(t *testing.T, logs []string) {
				assert.NotEmpty(t, logs)
				logStr := strings.Join(logs, " ")
				assert.Contains(t, logStr, "POST /api/data")
				assert.Contains(t, logStr, "200")
				assert.Contains(t, logStr, "User-Agent: test-client")
				assert.Contains(t, logStr, "Authorization: [REDACTED]")
				assert.Contains(t, logStr, `{"key": "value"}`)
			},
		},
		{
			name: "Skip health check",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/health", nil)
			},
			expectLogs: false,
		},
		{
			name: "Large body truncation",
			setupRequest: func() *http.Request {
				largeBody := strings.Repeat("a", 2048)
				req := httptest.NewRequest("POST", "/api/upload", strings.NewReader(largeBody))
				req.Header.Set("Content-Type", "text/plain")
				return req
			},
			expectLogs: true,
			checkLogs: func(t *testing.T, logs []string) {
				logStr := strings.Join(logs, " ")
				assert.Contains(t, logStr, "[TRUNCATED]")
				assert.NotContains(t, logStr, strings.Repeat("a", 2048))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear log entries
			logEntries = logEntries[:0]

			req := tt.setupRequest()
			w := httptest.NewRecorder()

			handler := middleware.Middleware(testHandler)
			handler.ServeHTTP(w, req)

			if tt.expectLogs {
				assert.NotEmpty(t, logEntries)
				if tt.checkLogs != nil {
					tt.checkLogs(t, logEntries)
				}
			} else {
				assert.Empty(t, logEntries)
			}
		})
	}
}

func TestChainMiddlewares(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	rbacManager := tc.SetupRBACManager()
	uf := testutil.NewUserFactory()

	// Create test data
	user := uf.CreateBasicUser()
	token, err := jwtManager.GenerateToken(user, nil)
	require.NoError(t, err)

	// Setup RBAC
	ctx := context.Background()
	pf := testutil.NewPermissionFactory()
	rf := testutil.NewRoleFactory()

	perm := pf.CreateResourcePermissions("api", []string{"read"})[0]
	createdPerm, err := rbacManager.CreatePermission(ctx, perm)
	require.NoError(t, err)

	role := rf.CreateRoleWithPermissions([]string{createdPerm.ID})
	createdRole, err := rbacManager.CreateRole(ctx, role)
	require.NoError(t, err)

	err = rbacManager.AssignRoleToUser(ctx, user.Subject, createdRole.ID)
	require.NoError(t, err)

	// Create middleware chain
	authMiddleware := NewAuthMiddleware(&AuthMiddlewareConfig{
		JWTManager:  jwtManager,
		RequireAuth: true,
		HeaderName:  "Authorization",
		ContextKey:  "user",
	})

	rbacMiddleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager:       rbacManager,
		ResourceExtractor: func(r *http.Request) string { return "api" },
		ActionExtractor:   func(r *http.Request) string { return "read" },
		UserIDExtractor: func(r *http.Request) string {
			if userCtx := r.Context().Value("user"); userCtx != nil {
				return userCtx.(*UserContext).UserID
			}
			return ""
		},
	})

	corsMiddleware := NewCORSMiddleware(&CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	securityMiddleware := NewSecurityHeadersMiddleware(&SecurityHeadersConfig{
		XFrameOptions:       "DENY",
		XContentTypeOptions: "nosniff",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userCtx := r.Context().Value("user").(*UserContext)
		response := map[string]string{
			"message": "success",
			"user_id": userCtx.UserID,
		}
		json.NewEncoder(w).Encode(response)
	})

	// Chain middlewares
	handler := ChainMiddlewares(testHandler,
		securityMiddleware.Middleware,
		corsMiddleware.Middleware,
		authMiddleware.Middleware,
		rbacMiddleware.Middleware,
	)

	tests := []struct {
		name         string
		setupRequest func() *http.Request
		expectStatus int
		checkHeaders func(*testing.T, http.Header)
	}{
		{
			name: "Complete chain success",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/data", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Origin", "https://example.com")
				return req
			},
			expectStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, headers http.Header) {
				assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
				assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
				assert.Equal(t, "https://example.com", headers.Get("Access-Control-Allow-Origin"))
			},
		},
		{
			name: "Chain stops at auth middleware",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/data", nil)
				req.Header.Set("Origin", "https://example.com")
				// No authorization header
				return req
			},
			expectStatus: http.StatusUnauthorized,
			checkHeaders: func(t *testing.T, headers http.Header) {
				// Security headers should still be set
				assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
				assert.Equal(t, "https://example.com", headers.Get("Access-Control-Allow-Origin"))
			},
		},
		{
			name: "Chain stops at CORS middleware",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/data", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Origin", "https://malicious.com")
				return req
			},
			expectStatus: http.StatusForbidden,
			checkHeaders: func(t *testing.T, headers http.Header) {
				// Security headers should still be set
				assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.checkHeaders != nil {
				tt.checkHeaders(t, w.Header())
			}
		})
	}
}

// Benchmark tests
func BenchmarkAuthMiddleware(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()
	token, err := jwtManager.GenerateToken(user, nil)
	if err != nil {
		b.Fatal(err)
	}

	middleware := NewAuthMiddleware(&AuthMiddlewareConfig{
		JWTManager:  jwtManager,
		RequireAuth: true,
		HeaderName:  "Authorization",
		ContextKey:  "user",
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Middleware(testHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkRBACMiddleware(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	rbacManager := tc.SetupRBACManager()

	middleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager:       rbacManager,
		ResourceExtractor: func(r *http.Request) string { return "api" },
		ActionExtractor:   func(r *http.Request) string { return "read" },
		UserIDExtractor:   func(r *http.Request) string { return "test-user" },
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.Middleware(testHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/data", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user", &UserContext{
			UserID: "test-user",
		}))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// Helper functions
func ChainMiddlewares(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
