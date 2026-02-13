package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// Mock token store for testing
type mockTokenStore struct {
	tokens map[string]*TokenInfo
}

func (m *mockTokenStore) StoreToken(ctx context.Context, tokenID string, token *TokenInfo) error {
	m.tokens[tokenID] = token
	return nil
}

func (m *mockTokenStore) GetToken(ctx context.Context, tokenID string) (*TokenInfo, error) {
	if token, exists := m.tokens[tokenID]; exists {
		return token, nil
	}
	return nil, fmt.Errorf("token not found")
}

func (m *mockTokenStore) UpdateToken(ctx context.Context, tokenID string, token *TokenInfo) error {
	m.tokens[tokenID] = token
	return nil
}

func (m *mockTokenStore) DeleteToken(ctx context.Context, tokenID string) error {
	delete(m.tokens, tokenID)
	return nil
}

func (m *mockTokenStore) ListUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {
	var tokens []*TokenInfo
	for _, token := range m.tokens {
		if token.UserID == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (m *mockTokenStore) CleanupExpired(ctx context.Context) error {
	now := time.Now()
	for id, token := range m.tokens {
		if now.After(token.ExpiresAt) {
			delete(m.tokens, id)
		}
	}
	return nil
}

// Mock token blacklist for testing
type mockTokenBlacklist struct {
	blacklisted map[string]time.Time
}

func (m *mockTokenBlacklist) BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	m.blacklisted[tokenID] = expiresAt
	return nil
}

func (m *mockTokenBlacklist) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	_, exists := m.blacklisted[tokenID]
	return exists, nil
}

func (m *mockTokenBlacklist) CleanupExpired(ctx context.Context) error {
	now := time.Now()
	for id, expiresAt := range m.blacklisted {
		if now.After(expiresAt) {
			delete(m.blacklisted, id)
		}
	}
	return nil
}

// Mock OAuth provider for testing
type mockOAuthProvider struct {
	name     string
	userInfo *providers.UserInfo
	tokens   *providers.TokenResponse
}

func (m *mockOAuthProvider) GetProviderName() string {
	return m.name
}

func (m *mockOAuthProvider) GetAuthorizationURL(state, redirectURI string, options ...providers.AuthOption) (string, *providers.PKCEChallenge, error) {
	challenge, _ := providers.GeneratePKCEChallenge()
	return "https://example.com/auth?state=" + state, challenge, nil
}

func (m *mockOAuthProvider) ExchangeCodeForToken(ctx context.Context, code, redirectURI string, challenge *providers.PKCEChallenge) (*providers.TokenResponse, error) {
	return m.tokens, nil
}

func (m *mockOAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*providers.TokenResponse, error) {
	return m.tokens, nil
}

func (m *mockOAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (*providers.UserInfo, error) {
	return m.userInfo, nil
}

func (m *mockOAuthProvider) ValidateToken(ctx context.Context, accessToken string) (*providers.TokenValidation, error) {
	return &providers.TokenValidation{Valid: true}, nil
}

func (m *mockOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	return nil
}

func (m *mockOAuthProvider) SupportsFeature(feature providers.ProviderFeature) bool {
	return true
}

func (m *mockOAuthProvider) GetConfiguration() *providers.ProviderConfig {
	return &providers.ProviderConfig{
		Name:     m.name,
		Type:     "mock",
		Features: []providers.ProviderFeature{providers.FeaturePKCE, providers.FeatureTokenRefresh},
	}
}

func setupTestSessionManager(t *testing.T) (*SessionManager, *JWTManager, *RBACManager) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Setup JWT manager
	jwtConfig := &JWTConfig{
		Issuer:     "test-issuer",
		DefaultTTL: 1 * time.Hour,
		RefreshTTL: 24 * time.Hour,
	}

	mockStore := &mockTokenStore{tokens: make(map[string]*TokenInfo)}
	mockBlacklist := &mockTokenBlacklist{blacklisted: make(map[string]time.Time)}

	jwtManager, err := NewJWTManager(context.Background(), jwtConfig, mockStore, mockBlacklist, logger)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	// Setup RBAC manager
	rbacManager := NewRBACManager(nil, logger)

	// Setup session manager
	sessionConfig := &SessionConfig{
		SessionTimeout:   24 * time.Hour,
		RefreshThreshold: 15 * time.Minute,
		MaxSessions:      10,
		EnableSSO:        true,
		EnableCSRF:       true,
		StateTimeout:     10 * time.Minute,
		CleanupInterval:  1 * time.Hour,
	}

	sessionManager := NewSessionManager(sessionConfig, jwtManager, rbacManager, logger)

	// Register mock provider
	mockProvider := &mockOAuthProvider{
		name: "mock",
		userInfo: &providers.UserInfo{
			Subject:  "test-user-123",
			Email:    "test@example.com",
			Name:     "Test User",
			Provider: "mock",
		},
		tokens: &providers.TokenResponse{
			AccessToken:  "mock-access-token",
			RefreshToken: "mock-refresh-token",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		},
	}
	sessionManager.RegisterProvider(mockProvider)

	return sessionManager, jwtManager, rbacManager
}

func TestNewSessionManager(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)

	if sessionManager == nil {
		t.Fatal("Expected SessionManager to be created")
	}

	// Verify provider is registered
	if len(sessionManager.providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(sessionManager.providers))
	}

	if _, exists := sessionManager.providers["mock"]; !exists {
		t.Error("Expected mock provider to be registered")
	}
}

func TestInitiateLogin(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	loginRequest := &LoginRequest{
		Provider:    "mock",
		RedirectURI: "http://localhost:8080/callback",
		IPAddress:   "127.0.0.1",
		UserAgent:   "test-agent",
		Options: map[string]string{
			"prompt": "consent",
		},
	}

	response, err := sessionManager.InitiateLogin(ctx, loginRequest)
	if err != nil {
		t.Fatalf("Failed to initiate login: %v", err)
	}

	if response.AuthURL == "" {
		t.Error("Expected auth URL to be set")
	}

	if response.State == "" {
		t.Error("Expected state to be set")
	}

	// Verify state is stored
	if _, exists := sessionManager.stateStore[response.State]; !exists {
		t.Error("Expected auth state to be stored")
	}
}

func TestHandleCallback(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	// First, initiate login to get state
	loginRequest := &LoginRequest{
		Provider:    "mock",
		RedirectURI: "http://localhost:8080/callback",
		IPAddress:   "127.0.0.1",
		UserAgent:   "test-agent",
	}

	loginResponse, err := sessionManager.InitiateLogin(ctx, loginRequest)
	if err != nil {
		t.Fatalf("Failed to initiate login: %v", err)
	}

	// Handle callback
	callbackRequest := &CallbackRequest{
		Provider:    "mock",
		Code:        "test-auth-code",
		State:       loginResponse.State,
		RedirectURI: "http://localhost:8080/callback",
		IPAddress:   "127.0.0.1",
		UserAgent:   "test-agent",
	}

	callbackResponse, err := sessionManager.HandleCallback(ctx, callbackRequest)
	if err != nil {
		t.Fatalf("Failed to handle callback: %v", err)
	}

	if !callbackResponse.Success {
		t.Errorf("Expected callback to succeed, got error: %s", callbackResponse.Error)
	}

	if callbackResponse.SessionID == "" {
		t.Error("Expected session ID to be set")
	}

	if callbackResponse.UserInfo == nil {
		t.Error("Expected user info to be set")
	}

	// Verify session is created
	if _, exists := sessionManager.sessions[callbackResponse.SessionID]; !exists {
		t.Error("Expected session to be created")
	}

	// Verify state is cleaned up
	if _, exists := sessionManager.stateStore[loginResponse.State]; exists {
		t.Error("Expected auth state to be cleaned up")
	}
}

func TestSessionValidation(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	// Create a session through callback
	loginRequest := &LoginRequest{
		Provider:  "mock",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	loginResponse, _ := sessionManager.InitiateLogin(ctx, loginRequest)

	callbackRequest := &CallbackRequest{
		Provider:  "mock",
		Code:      "test-code",
		State:     loginResponse.State,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	callbackResponse, err := sessionManager.HandleCallback(ctx, callbackRequest)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Validate session
	sessionInfo, err := sessionManager.ValidateSession(ctx, callbackResponse.SessionID)
	if err != nil {
		t.Fatalf("Failed to validate session: %v", err)
	}

	if sessionInfo.UserID != "test-user-123" {
		t.Errorf("Expected user ID 'test-user-123', got '%s'", sessionInfo.UserID)
	}

	if sessionInfo.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got '%s'", sessionInfo.Provider)
	}
}

func TestSessionRefresh(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	// Create a session with short timeout
	sessionManager.config.SessionTimeout = 1 * time.Second
	sessionManager.config.RefreshThreshold = 500 * time.Millisecond

	loginRequest := &LoginRequest{
		Provider:  "mock",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	loginResponse, _ := sessionManager.InitiateLogin(ctx, loginRequest)
	callbackRequest := &CallbackRequest{
		Provider:  "mock",
		Code:      "test-code",
		State:     loginResponse.State,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	callbackResponse, err := sessionManager.HandleCallback(ctx, callbackRequest)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Wait until refresh is needed
	time.Sleep(600 * time.Millisecond)

	// Refresh session
	err = sessionManager.RefreshSession(ctx, callbackResponse.SessionID)
	if err != nil {
		t.Fatalf("Failed to refresh session: %v", err)
	}

	// Verify session is still valid
	_, err = sessionManager.ValidateSession(ctx, callbackResponse.SessionID)
	if err != nil {
		t.Fatalf("Session should be valid after refresh: %v", err)
	}
}

func TestSessionRevocation(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	// Create a session
	loginRequest := &LoginRequest{
		Provider:  "mock",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	loginResponse, _ := sessionManager.InitiateLogin(ctx, loginRequest)
	callbackRequest := &CallbackRequest{
		Provider:  "mock",
		Code:      "test-code",
		State:     loginResponse.State,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	callbackResponse, err := sessionManager.HandleCallback(ctx, callbackRequest)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Revoke session
	err = sessionManager.RevokeSession(ctx, callbackResponse.SessionID)
	if err != nil {
		t.Fatalf("Failed to revoke session: %v", err)
	}

	// Verify session is gone
	_, err = sessionManager.ValidateSession(ctx, callbackResponse.SessionID)
	if err == nil {
		t.Error("Expected session to be invalid after revocation")
	}
}

func TestMultipleUserSessions(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	// Create sessions for different users
	users := []string{"user1", "user2", "user3"}
	sessionIDs := make(map[string]string)

	for _, userID := range users {
		// Mock different user info
		mockProvider := sessionManager.providers["mock"].(*mockOAuthProvider)
		originalUserInfo := mockProvider.userInfo
		mockProvider.userInfo = &providers.UserInfo{
			Subject:  userID,
			Email:    userID + "@example.com",
			Name:     "Test " + userID,
			Provider: "mock",
		}

		loginRequest := &LoginRequest{
			Provider:  "mock",
			IPAddress: "127.0.0.1",
			UserAgent: "test-agent",
		}

		loginResponse, _ := sessionManager.InitiateLogin(ctx, loginRequest)
		callbackRequest := &CallbackRequest{
			Provider:  "mock",
			Code:      "test-code",
			State:     loginResponse.State,
			IPAddress: "127.0.0.1",
			UserAgent: "test-agent",
		}

		callbackResponse, err := sessionManager.HandleCallback(ctx, callbackRequest)
		if err != nil {
			t.Fatalf("Failed to create session for user %s: %v", userID, err)
		}

		sessionIDs[userID] = callbackResponse.SessionID

		// Restore original user info
		mockProvider.userInfo = originalUserInfo
	}

	// Verify all sessions exist
	for userID, sessionID := range sessionIDs {
		sessionInfo, err := sessionManager.ValidateSession(ctx, sessionID)
		if err != nil {
			t.Errorf("Failed to validate session for user %s: %v", userID, err)
		}

		if sessionInfo.UserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, sessionInfo.UserID)
		}
	}

	// List sessions for user1
	sessions, err := sessionManager.ListUserSessions(ctx, "user1")
	if err != nil {
		t.Fatalf("Failed to list sessions for user1: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session for user1, got %d", len(sessions))
	}

	// Revoke all sessions for user1
	err = sessionManager.RevokeUserSessions(ctx, "user1")
	if err != nil {
		t.Fatalf("Failed to revoke user sessions: %v", err)
	}

	// Verify user1's session is gone but others remain
	_, err = sessionManager.ValidateSession(ctx, sessionIDs["user1"])
	if err == nil {
		t.Error("Expected user1's session to be revoked")
	}

	// Verify user2's session still exists
	_, err = sessionManager.ValidateSession(ctx, sessionIDs["user2"])
	if err != nil {
		t.Error("Expected user2's session to remain valid")
	}
}

func TestSessionMetrics(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		loginRequest := &LoginRequest{
			Provider:  "mock",
			IPAddress: "127.0.0.1",
			UserAgent: "test-agent",
		}

		loginResponse, _ := sessionManager.InitiateLogin(ctx, loginRequest)
		callbackRequest := &CallbackRequest{
			Provider:  "mock",
			Code:      "test-code",
			State:     loginResponse.State,
			IPAddress: "127.0.0.1",
			UserAgent: "test-agent",
		}

		_, err := sessionManager.HandleCallback(ctx, callbackRequest)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
	}

	// Get metrics
	metrics := sessionManager.GetSessionMetrics(ctx)

	activeSessions, ok := metrics["active_sessions"].(int)
	if !ok || activeSessions != 3 {
		t.Errorf("Expected 3 active sessions, got %v", activeSessions)
	}

	providerCounts, ok := metrics["provider_counts"].(map[string]int)
	if !ok {
		t.Error("Expected provider_counts to be a map")
	} else {
		if mockCount, exists := providerCounts["mock"]; !exists || mockCount != 3 {
			t.Errorf("Expected 3 mock provider sessions, got %d", mockCount)
		}
	}

	if _, exists := metrics["sso_enabled"]; exists {
		t.Error("Did not expect deprecated sso_enabled metric key")
	}
}

func TestInvalidState(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	// Try callback with invalid state
	callbackRequest := &CallbackRequest{
		Provider:  "mock",
		Code:      "test-code",
		State:     "invalid-state",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	response, err := sessionManager.HandleCallback(ctx, callbackRequest)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response.Success {
		t.Error("Expected callback to fail with invalid state")
	}

	if response.Error == "" {
		t.Error("Expected error message for invalid state")
	}
}

func TestExpiredState(t *testing.T) {
	sessionManager, _, _ := setupTestSessionManager(t)
	ctx := context.Background()

	// Set short state timeout
	sessionManager.config.StateTimeout = 100 * time.Millisecond

	loginRequest := &LoginRequest{
		Provider:  "mock",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	loginResponse, _ := sessionManager.InitiateLogin(ctx, loginRequest)

	// Wait for state to expire
	time.Sleep(200 * time.Millisecond)

	callbackRequest := &CallbackRequest{
		Provider:  "mock",
		Code:      "test-code",
		State:     loginResponse.State,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	response, err := sessionManager.HandleCallback(ctx, callbackRequest)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response.Success {
		t.Error("Expected callback to fail with expired state")
	}
}

func BenchmarkSessionValidation(b *testing.B) {
	sessionManager, _, _ := setupTestSessionManager(&testing.T{})
	ctx := context.Background()

	// Create a session
	loginRequest := &LoginRequest{
		Provider:  "mock",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	loginResponse, _ := sessionManager.InitiateLogin(ctx, loginRequest)
	callbackRequest := &CallbackRequest{
		Provider:  "mock",
		Code:      "test-code",
		State:     loginResponse.State,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	callbackResponse, _ := sessionManager.HandleCallback(ctx, callbackRequest)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionManager.ValidateSession(ctx, callbackResponse.SessionID)
	}
}
