package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	testutil "github.com/thc1006/nephoran-intent-operator/pkg/testutil/auth"
)

func TestSessionManager_CreateSession(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	tests := []struct {
		name         string
		userInfo     interface{}
		metadata     map[string]interface{}
		expectError  bool
		checkSession func(*testing.T, *Session)
	}{
		{
			name:     "Valid session creation",
			userInfo: uf.CreateBasicUser(),
			metadata: map[string]interface{}{
				"ip_address":   "192.168.1.1",
				"user_agent":   "Mozilla/5.0...",
				"login_method": "oauth2",
			},
			expectError: false,
			checkSession: func(t *testing.T, session *Session) {
				assert.NotEmpty(t, session.ID)
				assert.NotEmpty(t, session.UserID)
				assert.NotZero(t, session.CreatedAt)
				assert.True(t, session.ExpiresAt.After(time.Now()))
				assert.Equal(t, "192.168.1.1", session.IPAddress)
				assert.Contains(t, session.Metadata, "login_method")
			},
		},
		{
			name:        "Session with minimal metadata",
			userInfo:    uf.CreateBasicUser(),
			metadata:    nil,
			expectError: false,
			checkSession: func(t *testing.T, session *Session) {
				assert.NotNil(t, session.Metadata)
				assert.NotEmpty(t, session.ID)
			},
		},
		{
			name:        "Nil user info",
			userInfo:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			session, err := manager.CreateSession(ctx, tt.userInfo, tt.metadata)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, session)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, session)
			if tt.checkSession != nil {
				tt.checkSession(t, session)
			}
		})
	}
}

func TestSessionManager_GetSession(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()
	sf := testutil.NewSessionFactory()

	// Create a valid session
	user := uf.CreateBasicUser()
	validSession, err := manager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	// Create expired session manually
	expiredSession := sf.CreateExpiredSession(user.Subject)

	tests := []struct {
		name         string
		sessionID    string
		expectError  bool
		expectNil    bool
		checkSession func(*testing.T, *Session)
	}{
		{
			name:        "Valid session retrieval",
			sessionID:   validSession.ID,
			expectError: false,
			expectNil:   false,
			checkSession: func(t *testing.T, session *Session) {
				assert.Equal(t, validSession.ID, session.ID)
				assert.Equal(t, validSession.UserID, session.UserID)
			},
		},
		{
			name:        "Non-existent session",
			sessionID:   "non-existent-session-id",
			expectError: false,
			expectNil:   true,
		},
		{
			name:        "Empty session ID",
			sessionID:   "",
			expectError: true,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			session, err := manager.GetSession(ctx, tt.sessionID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNil {
				assert.Nil(t, session)
			} else {
				assert.NotNil(t, session)
				if tt.checkSession != nil {
					tt.checkSession(t, session)
				}
			}
		})
	}
}

func TestSessionManager_ValidateSession(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()
	sf := testutil.NewSessionFactory()

	user := uf.CreateBasicUser()

	// Create valid session
	validSession, err := manager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	// Create expired session manually
	expiredSession := sf.CreateExpiredSession(user.Subject)

	tests := []struct {
		name        string
		sessionID   string
		expectError bool
		errorType   string
	}{
		{
			name:        "Valid session",
			sessionID:   validSession.ID,
			expectError: false,
		},
		{
			name:        "Non-existent session",
			sessionID:   "non-existent-id",
			expectError: true,
			errorType:   "not found",
		},
		{
			name:        "Empty session ID",
			sessionID:   "",
			expectError: true,
			errorType:   "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			isValid, err := manager.ValidateSession(ctx, tt.sessionID)

			if tt.expectError {
				assert.Error(t, err)
				assert.False(t, isValid)
				// Could add specific error type checking
			} else {
				assert.NoError(t, err)
				assert.True(t, isValid)
			}
		})
	}
}

func TestSessionManager_RefreshSession(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()
	originalSession, err := manager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	tests := []struct {
		name         string
		sessionID    string
		expectError  bool
		checkSession func(*testing.T, *Session, *Session)
	}{
		{
			name:        "Valid session refresh",
			sessionID:   originalSession.ID,
			expectError: false,
			checkSession: func(t *testing.T, original, refreshed *Session) {
				assert.Equal(t, original.ID, refreshed.ID)
				assert.Equal(t, original.UserID, refreshed.UserID)
				assert.True(t, refreshed.ExpiresAt.After(original.ExpiresAt))
				assert.True(t, refreshed.UpdatedAt.After(original.UpdatedAt))
			},
		},
		{
			name:        "Non-existent session refresh",
			sessionID:   "non-existent-id",
			expectError: true,
		},
		{
			name:        "Empty session ID",
			sessionID:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			refreshedSession, err := manager.RefreshSession(ctx, tt.sessionID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, refreshedSession)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, refreshedSession)
			if tt.checkSession != nil {
				tt.checkSession(t, originalSession, refreshedSession)
			}
		})
	}
}

func TestSessionManager_RevokeSession(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()
	session, err := manager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	tests := []struct {
		name         string
		sessionID    string
		expectError  bool
		checkRevoked func(*testing.T, string)
	}{
		{
			name:        "Valid session revocation",
			sessionID:   session.ID,
			expectError: false,
			checkRevoked: func(t *testing.T, sessionID string) {
				// Verify session is revoked by trying to validate
				ctx := context.Background()
				isValid, err := manager.ValidateSession(ctx, sessionID)
				assert.False(t, isValid)
				assert.Error(t, err)
			},
		},
		{
			name:        "Non-existent session revocation",
			sessionID:   "non-existent-id",
			expectError: true,
		},
		{
			name:        "Empty session ID",
			sessionID:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := manager.RevokeSession(ctx, tt.sessionID)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkRevoked != nil {
				tt.checkRevoked(t, tt.sessionID)
			}
		})
	}
}

func TestSessionManager_RevokeAllUserSessions(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()

	// Create multiple sessions for the user
	session1, err := manager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	session2, err := manager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	// Create session for different user
	otherUser := uf.CreateBasicUser()
	otherSession, err := manager.CreateSession(context.Background(), otherUser, nil)
	require.NoError(t, err)

	tests := []struct {
		name         string
		userID       string
		expectError  bool
		checkRevoked func(*testing.T, string)
	}{
		{
			name:        "Valid user session revocation",
			userID:      user.Subject,
			expectError: false,
			checkRevoked: func(t *testing.T, userID string) {
				ctx := context.Background()

				// Check that user's sessions are revoked
				isValid1, _ := manager.ValidateSession(ctx, session1.ID)
				assert.False(t, isValid1)

				isValid2, _ := manager.ValidateSession(ctx, session2.ID)
				assert.False(t, isValid2)

				// Check that other user's session is still valid
				isValidOther, err := manager.ValidateSession(ctx, otherSession.ID)
				assert.NoError(t, err)
				assert.True(t, isValidOther)
			},
		},
		{
			name:        "Non-existent user",
			userID:      "non-existent-user",
			expectError: false, // Should succeed even if no sessions exist
		},
		{
			name:        "Empty user ID",
			userID:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := manager.RevokeAllUserSessions(ctx, tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkRevoked != nil {
				tt.checkRevoked(t, tt.userID)
			}
		})
	}
}

func TestSessionManager_CleanupExpiredSessions(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()
	sf := testutil.NewSessionFactory()

	user := uf.CreateBasicUser()

	// Create valid session
	validSession, err := manager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	// Manually create expired sessions (in a real implementation, these would be in the storage)
	expiredSession1 := sf.CreateExpiredSession(user.Subject)
	expiredSession2 := sf.CreateExpiredSession(user.Subject)

	// Store expired sessions in manager (implementation detail depends on storage backend)
	// For this test, we'll assume the cleanup method exists and works

	tests := []struct {
		name              string
		expectError       bool
		checkAfterCleanup func(*testing.T)
	}{
		{
			name:        "Successful cleanup",
			expectError: false,
			checkAfterCleanup: func(t *testing.T) {
				ctx := context.Background()

				// Valid session should still exist
				isValid, err := manager.ValidateSession(ctx, validSession.ID)
				assert.NoError(t, err)
				assert.True(t, isValid)

				// Expired sessions should be removed (would need access to storage to verify)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.CleanupExpiredSessions()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkAfterCleanup != nil {
				tt.checkAfterCleanup(t)
			}
		})
	}
}

func TestSessionManager_HTTPIntegration(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()

	tests := []struct {
		name          string
		setupRequest  func() (*http.Request, *httptest.ResponseRecorder)
		expectError   bool
		expectCookie  bool
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Set session cookie",
			setupRequest: func() (*http.Request, *httptest.ResponseRecorder) {
				req := httptest.NewRequest("GET", "/", nil)
				w := httptest.NewRecorder()
				return req, w
			},
			expectError:  false,
			expectCookie: true,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				cookies := w.Result().Cookies()
				assert.Len(t, cookies, 1)

				cookie := cookies[0]
				assert.Equal(t, "test-session", cookie.Name)
				assert.NotEmpty(t, cookie.Value)
				assert.True(t, cookie.HttpOnly)
				assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
			},
		},
		{
			name: "Get session from cookie",
			setupRequest: func() (*http.Request, *httptest.ResponseRecorder) {
				// First create a session
				session, err := manager.CreateSession(context.Background(), user, nil)
				require.NoError(t, err)

				req := httptest.NewRequest("GET", "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  "test-session",
					Value: session.ID,
				})
				w := httptest.NewRecorder()
				return req, w
			},
			expectError:  false,
			expectCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, w := tt.setupRequest()

			if tt.expectCookie {
				// Test setting session cookie
				session, err := manager.CreateSession(context.Background(), user, nil)
				if tt.expectError {
					assert.Error(t, err)
					return
				}

				require.NoError(t, err)
				manager.SetSessionCookie(w, session.ID)
			} else {
				// Test getting session from cookie
				sessionID, err := manager.GetSessionFromCookie(req)
				if tt.expectError {
					assert.Error(t, err)
					return
				}

				assert.NoError(t, err)
				assert.NotEmpty(t, sessionID)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestSessionManager_ClearSessionCookie(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()

	w := httptest.NewRecorder()
	manager.ClearSessionCookie(w)

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, "test-session", cookie.Name)
	assert.Empty(t, cookie.Value)
	assert.True(t, cookie.Expires.Before(time.Now()))
	assert.Equal(t, -1, cookie.MaxAge)
}

func TestSessionManager_UpdateSessionMetadata(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()
	session, err := manager.CreateSession(context.Background(), user, map[string]interface{}{
		"initial_key": "initial_value",
	})
	require.NoError(t, err)

	tests := []struct {
		name         string
		sessionID    string
		metadata     map[string]interface{}
		expectError  bool
		checkSession func(*testing.T, *Session)
	}{
		{
			name:      "Update existing session metadata",
			sessionID: session.ID,
			metadata: map[string]interface{}{
				"updated_key":   "updated_value",
				"new_key":       "new_value",
				"login_count":   5,
				"last_activity": time.Now().Format(time.RFC3339),
			},
			expectError: false,
			checkSession: func(t *testing.T, updatedSession *Session) {
				assert.Equal(t, "updated_value", updatedSession.Metadata["updated_key"])
				assert.Equal(t, "new_value", updatedSession.Metadata["new_key"])
				assert.Equal(t, 5, updatedSession.Metadata["login_count"])
				assert.Contains(t, updatedSession.Metadata, "last_activity")

				// Original metadata should still exist unless overwritten
				assert.Contains(t, updatedSession.Metadata, "initial_key")
			},
		},
		{
			name:        "Non-existent session",
			sessionID:   "non-existent-id",
			metadata:    map[string]interface{}{"key": "value"},
			expectError: true,
		},
		{
			name:        "Empty session ID",
			sessionID:   "",
			metadata:    map[string]interface{}{"key": "value"},
			expectError: true,
		},
		{
			name:        "Nil metadata",
			sessionID:   session.ID,
			metadata:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			updatedSession, err := manager.UpdateSessionMetadata(ctx, tt.sessionID, tt.metadata)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, updatedSession)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, updatedSession)
			if tt.checkSession != nil {
				tt.checkSession(t, updatedSession)
			}
		})
	}
}

func TestSessionManager_GetUserSessions(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()
	otherUser := uf.CreateBasicUser()

	// Create multiple sessions for the user
	session1, err := manager.CreateSession(context.Background(), user, map[string]interface{}{
		"device": "desktop",
	})
	require.NoError(t, err)

	session2, err := manager.CreateSession(context.Background(), user, map[string]interface{}{
		"device": "mobile",
	})
	require.NoError(t, err)

	// Create session for other user
	_, err = manager.CreateSession(context.Background(), otherUser, nil)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        string
		expectError   bool
		expectCount   int
		checkSessions func(*testing.T, []*Session)
	}{
		{
			name:        "Get user sessions",
			userID:      user.Subject,
			expectError: false,
			expectCount: 2,
			checkSessions: func(t *testing.T, sessions []*Session) {
				assert.Len(t, sessions, 2)

				// Verify all sessions belong to the user
				for _, session := range sessions {
					assert.Equal(t, user.Subject, session.UserID)
				}

				// Check that we have both desktop and mobile sessions
				deviceTypes := make([]string, 0, len(sessions))
				for _, session := range sessions {
					if device, ok := session.Metadata["device"]; ok {
						deviceTypes = append(deviceTypes, device.(string))
					}
				}
				assert.Contains(t, deviceTypes, "desktop")
				assert.Contains(t, deviceTypes, "mobile")
			},
		},
		{
			name:        "Get sessions for user with one session",
			userID:      otherUser.Subject,
			expectError: false,
			expectCount: 1,
		},
		{
			name:        "Get sessions for non-existent user",
			userID:      "non-existent-user",
			expectError: false,
			expectCount: 0,
		},
		{
			name:        "Empty user ID",
			userID:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			sessions, err := manager.GetUserSessions(ctx, tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, sessions)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, sessions, tt.expectCount)
			if tt.checkSessions != nil {
				tt.checkSessions(t, sessions)
			}
		})
	}
}

// Benchmark tests
func BenchmarkSessionManager_CreateSession(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.CreateSession(ctx, user, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSessionManager_ValidateSession(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()
	ctx := context.Background()

	session, err := manager.CreateSession(ctx, user, nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ValidateSession(ctx, session.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSessionManager_RefreshSession(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()
	ctx := context.Background()

	session, err := manager.CreateSession(ctx, user, nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.RefreshSession(ctx, session.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test concurrent access to sessions
func TestSessionManager_ConcurrentAccess(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	// Create initial session
	session, err := manager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	// Run concurrent operations
	const numGoroutines = 10
	const numOperations = 100

	errChan := make(chan error, numGoroutines*numOperations)

	// Test concurrent validation
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				_, err := manager.ValidateSession(context.Background(), session.ID)
				errChan <- err
			}
		}()
	}

	// Test concurrent refresh
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				_, err := manager.RefreshSession(context.Background(), session.ID)
				errChan <- err
			}
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines*numOperations*2; i++ {
		err := <-errChan
		assert.NoError(t, err, "Concurrent operation failed")
	}
}

// Test session security features
func TestSessionManager_SecurityFeatures(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupSessionManager()
	uf := testutil.NewUserFactory()

	tests := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "Session ID uniqueness",
			testFunc: func(t *testing.T) {
				user := uf.CreateBasicUser()
				sessionIDs := make(map[string]bool)

				// Create multiple sessions and verify uniqueness
				for i := 0; i < 100; i++ {
					session, err := manager.CreateSession(context.Background(), user, nil)
					require.NoError(t, err)

					assert.False(t, sessionIDs[session.ID], "Duplicate session ID detected")
					sessionIDs[session.ID] = true
				}
			},
		},
		{
			name: "Session ID entropy",
			testFunc: func(t *testing.T) {
				user := uf.CreateBasicUser()
				session, err := manager.CreateSession(context.Background(), user, nil)
				require.NoError(t, err)

				// Session ID should be sufficiently long and random
				assert.GreaterOrEqual(t, len(session.ID), 32, "Session ID should be at least 32 characters")
				assert.NotEqual(t, "00000000000000000000000000000000", session.ID, "Session ID should not be all zeros")
			},
		},
		{
			name: "Session expiration enforcement",
			testFunc: func(t *testing.T) {
				user := uf.CreateBasicUser()

				// Create session with very short TTL
				manager.config.SessionTTL = time.Millisecond
				session, err := manager.CreateSession(context.Background(), user, nil)
				require.NoError(t, err)

				// Wait for expiration
				time.Sleep(10 * time.Millisecond)

				// Session should be invalid
				isValid, err := manager.ValidateSession(context.Background(), session.ID)
				assert.False(t, isValid)
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}
