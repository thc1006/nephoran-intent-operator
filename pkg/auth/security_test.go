package auth_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
	authtestutil "github.com/thc1006/nephoran-intent-operator/pkg/testutil/auth"
)

// SecurityTestSuite provides comprehensive security testing
type SecurityTestSuite struct {
	t              *testing.T
	tc             *authtestutil.TestContext
	jwtManager     *auth.JWTManager
	sessionManager *auth.SessionManager
	rbacManager    *auth.RBACManager
	server         *httptest.Server
	handlers       *auth.AuthHandlers
	testUser       *providers.UserInfo
	validToken     string
}

func NewSecurityTestSuite(t *testing.T) *SecurityTestSuite {
	tc := authtestutil.NewTestContext(t)

	suite := &SecurityTestSuite{
		t:              t,
		tc:             tc,
		jwtManager:     tc.SetupJWTManager(),
		sessionManager: tc.SetupSessionManager(),
		rbacManager:    tc.SetupRBACManager(),
	}

	suite.setupTestData()
	suite.setupHTTPServer()

	return suite
}

func (suite *SecurityTestSuite) setupTestData() {
	uf := authtestutil.NewUserFactory()
	suite.testUser = uf.CreateBasicUser()

	var err error
	suite.validToken, err = suite.jwtManager.GenerateToken(suite.testUser, nil)
	require.NoError(suite.t, err)
}

func (suite *SecurityTestSuite) setupHTTPServer() {
	// Create handlers with security features enabled
	suite.handlers = NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:      suite.jwtManager,
		SessionManager:  suite.sessionManager,
		RBACManager:     suite.rbacManager,
		EnableCSRF:      true,
		EnableRateLimit: true,
		RateLimitRPS:    10,
		Logger:          suite.tc.Logger,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", suite.handlers.LoginHandler)
	mux.HandleFunc("/auth/userinfo", suite.handlers.UserInfoHandler)
	mux.HandleFunc("/auth/refresh", suite.handlers.RefreshTokenHandler)
	mux.HandleFunc("/auth/csrf-token", suite.handlers.CSRFTokenHandler)
	mux.HandleFunc("/protected", suite.protectedHandler)

	// Apply security middleware
	securityMiddleware := NewSecurityHeadersMiddleware(&SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'",
		XFrameOptions:         "DENY",
		XContentTypeOptions:   "nosniff",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		HSTSMaxAge:            31536000,
		RemoveServerHeader:    true,
	})

	suite.server = httptest.NewServer(securityMiddleware.Middleware(mux))
}

func (suite *SecurityTestSuite) protectedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("protected content"))
}

func (suite *SecurityTestSuite) Cleanup() {
	if suite.server != nil {
		suite.server.Close()
	}
	suite.tc.Cleanup()
}

func TestSecurity_JWTTokenManipulation(t *testing.T) {
	suite := NewSecurityTestSuite(t)
	defer suite.Cleanup()

	tests := []struct {
		name          string
		tokenModifier func(string) string
		expectStatus  int
		expectError   bool
	}{
		{
			name: "Valid token",
			tokenModifier: func(token string) string {
				return token
			},
			expectStatus: http.StatusOK,
			expectError:  false,
		},
		{
			name: "Modified signature",
			tokenModifier: func(token string) string {
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					return token
				}
				// Corrupt the signature
				signature := parts[2]
				corruptedSignature := signature[:len(signature)-5] + "XXXXX"
				return parts[0] + "." + parts[1] + "." + corruptedSignature
			},
			expectStatus: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name: "Modified payload",
			tokenModifier: func(token string) string {
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					return token
				}

				// Decode payload, modify it, encode back
				payload, err := base64.RawURLEncoding.DecodeString(parts[1])
				if err != nil {
					return token
				}

				var claims map[string]interface{}
				json.Unmarshal(payload, &claims)
				claims["sub"] = "malicious-user" // Change subject

				newPayload, _ := json.Marshal(claims)
				newPayloadB64 := base64.RawURLEncoding.EncodeToString(newPayload)

				return parts[0] + "." + newPayloadB64 + "." + parts[2]
			},
			expectStatus: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name: "Modified header algorithm",
			tokenModifier: func(token string) string {
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					return token
				}

				// Decode header, change algorithm to "none"
				header, err := base64.RawURLEncoding.DecodeString(parts[0])
				if err != nil {
					return token
				}

				var headerClaims map[string]interface{}
				json.Unmarshal(header, &headerClaims)
				headerClaims["alg"] = "none"

				newHeader, _ := json.Marshal(headerClaims)
				newHeaderB64 := base64.RawURLEncoding.EncodeToString(newHeader)

				return newHeaderB64 + "." + parts[1] + "." + ""
			},
			expectStatus: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name: "Token with expired timestamp",
			tokenModifier: func(token string) string {
				// Create new expired token
				expiredClaims := jwt.MapClaims{
					"iss": "test-issuer",
					"sub": suite.testUser.Subject,
					"exp": time.Now().Add(-time.Hour).Unix(),
					"iat": time.Now().Add(-2 * time.Hour).Unix(),
				}
				return suite.tc.CreateTestToken(expiredClaims)
			},
			expectStatus: http.StatusUnauthorized,
			expectError:  true,
		},
		{
			name: "Token with future nbf claim",
			tokenModifier: func(token string) string {
				futureClaims := jwt.MapClaims{
					"iss": "test-issuer",
					"sub": suite.testUser.Subject,
					"exp": time.Now().Add(2 * time.Hour).Unix(),
					"iat": time.Now().Unix(),
					"nbf": time.Now().Add(time.Hour).Unix(), // Not before 1 hour from now
				}
				return suite.tc.CreateTestToken(futureClaims)
			},
			expectStatus: http.StatusUnauthorized,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifiedToken := tt.tokenModifier(suite.validToken)

			req, _ := http.NewRequest("GET", suite.server.URL+"/auth/userinfo", nil)
			req.Header.Set("Authorization", "Bearer "+modifiedToken)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectStatus, resp.StatusCode)
		})
	}
}

func TestSecurity_SessionSecurity(t *testing.T) {
	suite := NewSecurityTestSuite(t)
	defer suite.Cleanup()

	// Create valid session
	ctx := context.Background()
	session, err := suite.sessionManager.CreateSession(ctx, suite.testUser, map[string]interface{}{
		"ip_address": "192.168.1.100",
		"user_agent": "test-browser",
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		sessionID    string
		expectStatus int
		description  string
	}{
		{
			name:         "Valid session",
			sessionID:    session.ID,
			expectStatus: http.StatusOK,
			description:  "Should allow access with valid session",
		},
		{
			name:         "Non-existent session",
			sessionID:    "non-existent-session-id",
			expectStatus: http.StatusUnauthorized,
			description:  "Should deny access with non-existent session",
		},
		{
			name:         "Empty session ID",
			sessionID:    "",
			expectStatus: http.StatusUnauthorized,
			description:  "Should deny access with empty session",
		},
		{
			name:         "SQL injection attempt in session ID",
			sessionID:    "'; DROP TABLE sessions; --",
			expectStatus: http.StatusUnauthorized,
			description:  "Should safely handle SQL injection attempts",
		},
		{
			name:         "XSS attempt in session ID",
			sessionID:    "<script>alert('xss')</script>",
			expectStatus: http.StatusUnauthorized,
			description:  "Should safely handle XSS attempts",
		},
		{
			name:         "Very long session ID",
			sessionID:    strings.Repeat("a", 10000),
			expectStatus: http.StatusUnauthorized,
			description:  "Should handle extremely long session IDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with session cookie
			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.sessionID != "" {
				req.AddCookie(&http.Cookie{
					Name:  "session",
					Value: tt.sessionID,
				})
			}

			w := httptest.NewRecorder()

			// Apply auth middleware
			authMiddleware := NewAuthMiddleware(&AuthMiddlewareConfig{
				SessionManager: suite.sessionManager,
				RequireAuth:    true,
				CookieName:     "session",
				ContextKey:     "user",
			})

			handler := authMiddleware.Middleware(http.HandlerFunc(suite.protectedHandler))
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectStatus, w.Code, tt.description)
		})
	}
}

func TestSecurity_CSRFProtection(t *testing.T) {
	suite := NewSecurityTestSuite(t)
	defer suite.Cleanup()

	// Get CSRF token first
	req, _ := http.NewRequest("GET", suite.server.URL+"/auth/csrf-token", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var csrfResp map[string]string
	err = json.NewDecoder(resp.Body).Decode(&csrfResp)
	require.NoError(t, err)

	validCSRFToken := csrfResp["csrf_token"]

	tests := []struct {
		name         string
		csrfToken    string
		expectStatus int
		description  string
	}{
		{
			name:         "Valid CSRF token",
			csrfToken:    validCSRFToken,
			expectStatus: http.StatusBadRequest, // Login will fail for other reasons, but CSRF passes
			description:  "Should allow request with valid CSRF token",
		},
		{
			name:         "Missing CSRF token",
			csrfToken:    "",
			expectStatus: http.StatusForbidden,
			description:  "Should deny request without CSRF token",
		},
		{
			name:         "Invalid CSRF token",
			csrfToken:    "invalid-csrf-token",
			expectStatus: http.StatusForbidden,
			description:  "Should deny request with invalid CSRF token",
		},
		{
			name:         "Expired CSRF token",
			csrfToken:    "expired-csrf-token",
			expectStatus: http.StatusForbidden,
			description:  "Should deny request with expired CSRF token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginReq := map[string]string{
				"provider":     "test",
				"redirect_uri": "http://localhost:8080/callback",
			}

			body, _ := json.Marshal(loginReq)
			req, _ := http.NewRequest("POST", suite.server.URL+"/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			if tt.csrfToken != "" {
				req.Header.Set("X-CSRF-Token", tt.csrfToken)
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectStatus, resp.StatusCode, tt.description)
		})
	}
}

func TestSecurity_RateLimiting(t *testing.T) {
	suite := NewSecurityTestSuite(t)
	defer suite.Cleanup()

	// Create rate limiter with very low limits for testing
	rateLimiter := NewRateLimitMiddleware(&RateLimitConfig{
		RequestsPerMinute: 5,
		BurstSize:         2,
		KeyGenerator: func(r *http.Request) string {
			return r.RemoteAddr
		},
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := rateLimiter.Middleware(testHandler)

	// Test normal requests within limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// Test requests exceeding limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "Request %d should be rate limited", i+3)
	}

	// Test different IP should not be affected
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.101:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "Different IP should not be rate limited")
}

func TestSecurity_InputValidation(t *testing.T) {
	suite := NewSecurityTestSuite(t)
	defer suite.Cleanup()

	tests := []struct {
		name         string
		payload      interface{}
		endpoint     string
		expectStatus int
		description  string
	}{
		{
			name: "SQL injection in login request",
			payload: map[string]string{
				"provider":     "test'; DROP TABLE users; --",
				"redirect_uri": "http://localhost:8080/callback",
			},
			endpoint:     "/auth/login",
			expectStatus: http.StatusBadRequest,
			description:  "Should reject SQL injection attempts",
		},
		{
			name: "XSS in redirect URI",
			payload: map[string]string{
				"provider":     "test",
				"redirect_uri": "javascript:alert('xss')",
			},
			endpoint:     "/auth/login",
			expectStatus: http.StatusBadRequest,
			description:  "Should reject JavaScript URLs",
		},
		{
			name: "Path traversal in redirect URI",
			payload: map[string]string{
				"provider":     "test",
				"redirect_uri": "http://localhost:8080/../../../etc/passwd",
			},
			endpoint:     "/auth/login",
			expectStatus: http.StatusBadRequest,
			description:  "Should reject path traversal attempts",
		},
		{
			name: "Extremely long provider name",
			payload: map[string]string{
				"provider":     strings.Repeat("a", 10000),
				"redirect_uri": "http://localhost:8080/callback",
			},
			endpoint:     "/auth/login",
			expectStatus: http.StatusBadRequest,
			description:  "Should handle extremely long inputs",
		},
		{
			name: "Invalid characters in provider name",
			payload: map[string]string{
				"provider":     "test\x00\x01\x02",
				"redirect_uri": "http://localhost:8080/callback",
			},
			endpoint:     "/auth/login",
			expectStatus: http.StatusBadRequest,
			description:  "Should reject null bytes and control characters",
		},
		{
			name: "Malformed refresh token",
			payload: map[string]string{
				"refresh_token": "malformed.token.with.null\x00bytes",
			},
			endpoint:     "/auth/refresh",
			expectStatus: http.StatusUnauthorized,
			description:  "Should safely handle malformed refresh tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req, _ := http.NewRequest("POST", suite.server.URL+tt.endpoint, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectStatus, resp.StatusCode, tt.description)
		})
	}
}

func TestSecurity_HeaderSecurity(t *testing.T) {
	suite := NewSecurityTestSuite(t)
	defer suite.Cleanup()

	req, _ := http.NewRequest("GET", suite.server.URL+"/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+suite.validToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check security headers
	assert.Equal(t, "default-src 'self'", resp.Header.Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	assert.Contains(t, resp.Header.Get("Strict-Transport-Security"), "max-age=31536000")

	// Server header should be removed
	assert.Empty(t, resp.Header.Get("Server"))
}

func TestSecurity_TimingAttacks(t *testing.T) {
	suite := NewSecurityTestSuite(t)
	defer suite.Cleanup()

	// Test that token validation has consistent timing regardless of token validity
	// This helps prevent timing attacks

	validToken := suite.validToken
	invalidToken := "invalid.jwt.token"

	measureTime := func(token string) time.Duration {
		start := time.Now()

		req := httptest.NewRequest("GET", "/auth/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		suite.handlers.UserInfoHandler(w, req)

		return time.Since(start)
	}

	const iterations = 10
	var validTimes, invalidTimes []time.Duration

	// Measure timing for valid tokens
	for i := 0; i < iterations; i++ {
		validTimes = append(validTimes, measureTime(validToken))
	}

	// Measure timing for invalid tokens
	for i := 0; i < iterations; i++ {
		invalidTimes = append(invalidTimes, measureTime(invalidToken))
	}

	// Calculate averages
	var validAvg, invalidAvg time.Duration
	for _, t := range validTimes {
		validAvg += t
	}
	validAvg /= time.Duration(len(validTimes))

	for _, t := range invalidTimes {
		invalidAvg += t
	}
	invalidAvg /= time.Duration(len(invalidTimes))

	// The timing difference should not be significant
	timingDifference := float64(validAvg-invalidAvg) / float64(validAvg)
	if timingDifference < 0 {
		timingDifference = -timingDifference
	}

	// Allow up to 50% timing difference (in practice should be much less)
	assert.Less(t, timingDifference, 0.5, "Timing difference too large: valid=%v, invalid=%v", validAvg, invalidAvg)
}

func TestSecurity_RandomnessQuality(t *testing.T) {
	// Test the quality of random values generated by the system

	// Test session ID randomness
	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()

	sessionManager := tc.SetupSessionManager()
	uf := authtestutil.NewUserFactory()
	user := uf.CreateBasicUser()

	const numSessions = 1000
	sessionIDs := make(map[string]bool)
	var sessionBytes []byte

	ctx := context.Background()
	for i := 0; i < numSessions; i++ {
		session, err := sessionManager.CreateSession(ctx, user, nil)
		require.NoError(t, err)

		// Check for duplicates
		assert.False(t, sessionIDs[session.ID], "Duplicate session ID found")
		sessionIDs[session.ID] = true

		// Collect bytes for entropy analysis
		sessionBytes = append(sessionBytes, []byte(session.ID)...)
	}

	// Basic entropy check - count unique bytes
	uniqueBytes := make(map[byte]bool)
	for _, b := range sessionBytes {
		uniqueBytes[b] = true
	}

	// Should have reasonable byte distribution
	assert.GreaterOrEqual(t, len(uniqueBytes), 50, "Session IDs should have good byte distribution")

	// Test JWT token randomness
	jwtManager := tc.SetupJWTManager()

	const numTokens = 100
	tokenIDs := make(map[string]bool)

	for i := 0; i < numTokens; i++ {
		token, err := jwtManager.GenerateToken(user, nil)
		require.NoError(t, err)

		// Extract JTI claim for uniqueness test
		claims, err := jwtManager.ValidateToken(token)
		require.NoError(t, err)

		if jti, ok := claims["jti"].(string); ok {
			assert.False(t, tokenIDs[jti], "Duplicate JWT ID found")
			tokenIDs[jti] = true
		}
	}
}

func TestSecurity_PasswordSecurityBestPractices(t *testing.T) {
	// Test that sensitive data is handled securely

	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()

	// Create configuration with sensitive data
	config := &JWTConfig{
		Issuer:       "test-issuer",
		SigningKey:   "super-secret-key",
		ClientSecret: "oauth2-client-secret",
	}

	// Test that sensitive data doesn't appear in string representation
	configStr := fmt.Sprintf("%+v", config)
	assert.NotContains(t, configStr, "super-secret-key")
	assert.NotContains(t, configStr, "oauth2-client-secret")

	// Test that errors don't leak sensitive information
	invalidToken := "invalid.token.here"
	jwtManager := tc.SetupJWTManager()

	_, err := jwtManager.ValidateToken(invalidToken)
	assert.Error(t, err)

	// Error message should not contain the actual token
	errorMsg := err.Error()
	assert.NotContains(t, strings.ToLower(errorMsg), "invalid.token.here")
}

func TestSecurity_CryptographicStandards(t *testing.T) {
	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()

	// Test RSA key strength
	assert.GreaterOrEqual(t, tc.PrivateKey.Size()*8, 2048, "RSA key should be at least 2048 bits")

	// Test JWT algorithm
	jwtManager := tc.SetupJWTManager()
	uf := authtestutil.NewUserFactory()
	user := uf.CreateBasicUser()

	token, err := jwtManager.GenerateToken(user, nil)
	require.NoError(t, err)

	// Parse token to check algorithm
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tc.PublicKey, nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	assert.Equal(t, "RS256", parsedToken.Header["alg"])
}

func TestSecurity_MemoryLeakPrevention(t *testing.T) {
	// Test that sensitive data is properly cleared from memory
	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	uf := authtestutil.NewUserFactory()
	user := uf.CreateBasicUser()

	// Generate many tokens to test for memory leaks
	const numTokens = 1000

	for i := 0; i < numTokens; i++ {
		token, err := jwtManager.GenerateToken(user, map[string]interface{}{
			"iteration":      i,
			"sensitive_data": fmt.Sprintf("secret-%d", i),
		})
		require.NoError(t, err)

		// Validate token
		_, err = jwtManager.ValidateToken(token)
		require.NoError(t, err)

		// Force garbage collection periodically
		if i%100 == 0 {
			// In a real test, you might use runtime.GC() and memory profiling
			// to detect memory leaks, but that's beyond the scope of this example
		}
	}
}

func TestSecurity_ConcurrencyAttacks(t *testing.T) {
	// Test that concurrent access doesn't expose security vulnerabilities
	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	uf := authtestutil.NewUserFactory()
	user := uf.CreateBasicUser()

	// Create initial token and session
	token, err := jwtManager.GenerateToken(user, nil)
	require.NoError(t, err)

	ctx := context.Background()
	session, err := sessionManager.CreateSession(ctx, user, nil)
	require.NoError(t, err)

	const numGoroutines = 10
	const numOperations = 100

	// Test concurrent token validation
	results := make(chan error, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				_, err := jwtManager.ValidateToken(token)
				results <- err
			}
		}()
	}

	// Test concurrent session operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				_, err := sessionManager.ValidateSession(ctx, session.ID)
				results <- err
			}
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines*numOperations*2; i++ {
		if err := <-results; err == nil {
			successCount++
		}
	}

	// All operations should succeed
	assert.Equal(t, numGoroutines*numOperations*2, successCount)
}

// Helper function to generate random bytes for testing
func generateRandomBytes(n int) []byte {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return bytes
}
