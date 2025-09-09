// Package auth provides authentication mocks and test utilities for the Nephoran Intent Operator.

// This package contains mock implementations for various authentication providers and services.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
)

// MockAuthenticator provides a mock implementation of authentication services.

type MockAuthenticator struct {
	mock.Mock

	fixtures *AuthFixtures

	sessions map[string]*UserSession

	mu sync.RWMutex
}

// UserSession represents an active user session.

type UserSession struct {
	SessionID string    `json:"session_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
}

// NewMockAuthenticator creates a new mock authenticator.

func NewMockAuthenticator() *MockAuthenticator {
	return &MockAuthenticator{
		fixtures: DefaultAuthFixtures(),
		sessions: make(map[string]*UserSession),
	}
}

// WithFixtures sets custom fixtures for the mock authenticator.

func (m *MockAuthenticator) WithFixtures(fixtures *AuthFixtures) *MockAuthenticator {
	m.fixtures = fixtures
	return m
}

// Authenticate performs authentication using username and password.

func (m *MockAuthenticator) Authenticate(ctx context.Context, username, password string) (*UserSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, username, password)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*UserSession), args.Error(1)
	}

	// Default behavior using fixtures
	if !m.fixtures.ValidateUserCredentials(username, password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	sessionID := fmt.Sprintf("session-%s-%d", username, time.Now().UnixNano())
	session := &UserSession{
		SessionID: sessionID,
		Username:  username,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Active:    true,
	}

	m.sessions[sessionID] = session
	return session, nil
}

// ValidateToken validates a JWT token.

func (m *MockAuthenticator) ValidateToken(ctx context.Context, tokenString string) (*jwt.Token, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, tokenString)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*jwt.Token), args.Error(1)
	}

	// Default behavior using fixtures
	return m.fixtures.ValidateJWTToken(tokenString)
}

// GenerateToken generates a JWT token for a user.

func (m *MockAuthenticator) GenerateToken(ctx context.Context, username string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, username)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.String(0), args.Error(1)
	}

	// Default behavior using fixtures
	return m.fixtures.GenerateJWTToken(username)
}

// GetSession retrieves a user session by session ID.

func (m *MockAuthenticator) GetSession(ctx context.Context, sessionID string) (*UserSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, sessionID)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*UserSession), args.Error(1)
	}

	// Default behavior
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if !session.Active || time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// HasPermission checks if a user has a specific permission.

func (m *MockAuthenticator) HasPermission(ctx context.Context, username, permission string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, username, permission)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Bool(0), args.Error(1)
	}

	// Default behavior using fixtures
	return m.fixtures.HasPermission(username, permission), nil
}

// Logout invalidates a user session.

func (m *MockAuthenticator) Logout(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, sessionID)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Error(0)
	}

	// Default behavior
	if session, exists := m.sessions[sessionID]; exists {
		session.Active = false
		session.ExpiresAt = time.Now()
	}

	return nil
}

// MockOAuth2Provider provides a mock OAuth2 provider implementation.

type MockOAuth2Provider struct {
	mock.Mock

	fixtures *OAuth2Fixtures

	mu sync.RWMutex
}

// NewMockOAuth2Provider creates a new mock OAuth2 provider.

func NewMockOAuth2Provider() *MockOAuth2Provider {
	return &MockOAuth2Provider{
		fixtures: CreateOAuth2Fixtures(),
	}
}

// WithOAuth2Fixtures sets custom OAuth2 fixtures.

func (m *MockOAuth2Provider) WithOAuth2Fixtures(fixtures *OAuth2Fixtures) *MockOAuth2Provider {
	m.fixtures = fixtures
	return m
}

// ValidateClientCredentials validates OAuth2 client credentials.

func (m *MockOAuth2Provider) ValidateClientCredentials(ctx context.Context, clientID, clientSecret string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, clientID, clientSecret)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Bool(0), args.Error(1)
	}

	// Default behavior using fixtures
	return m.fixtures.ValidateClientCredentials(clientID, clientSecret), nil
}

// GenerateAuthorizationCode generates an OAuth2 authorization code.

func (m *MockOAuth2Provider) GenerateAuthorizationCode(ctx context.Context, clientID, userID, redirectURI string, scopes []string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, clientID, userID, redirectURI, scopes)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.String(0), args.Error(1)
	}

	// Default behavior
	if !m.fixtures.ValidateRedirectURI(clientID, redirectURI) {
		return "", fmt.Errorf("invalid redirect URI")
	}

	code := fmt.Sprintf("auth-code-%d", time.Now().UnixNano())
	authCode := OAuth2AuthCode{
		Code:        code,
		ClientID:    clientID,
		UserID:      userID,
		RedirectURI: redirectURI,
		Scopes:      scopes,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	m.fixtures.AuthorizationCodes[code] = authCode
	return code, nil
}

// ExchangeCodeForTokens exchanges an authorization code for access tokens.

func (m *MockOAuth2Provider) ExchangeCodeForTokens(ctx context.Context, code, clientID, clientSecret string) (*OAuth2TokenResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, code, clientID, clientSecret)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*OAuth2TokenResponse), args.Error(1)
	}

	// Default behavior
	if !m.fixtures.ValidateClientCredentials(clientID, clientSecret) {
		return nil, fmt.Errorf("invalid client credentials")
	}

	authCode, exists := m.fixtures.AuthorizationCodes[code]
	if !exists || time.Now().After(authCode.ExpiresAt) {
		return nil, fmt.Errorf("invalid or expired authorization code")
	}

	// Generate access token
	accessToken := fmt.Sprintf("access-token-%d", time.Now().UnixNano())
	refreshToken := fmt.Sprintf("refresh-token-%d", time.Now().UnixNano())

	// Store tokens
	m.fixtures.AccessTokens[accessToken] = OAuth2AccessToken{
		Token:     accessToken,
		ClientID:  clientID,
		UserID:    authCode.UserID,
		Scopes:    authCode.Scopes,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	m.fixtures.RefreshTokens[refreshToken] = OAuth2RefreshToken{
		Token:     refreshToken,
		ClientID:  clientID,
		UserID:    authCode.UserID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	// Remove used authorization code
	delete(m.fixtures.AuthorizationCodes, code)

	return &OAuth2TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: refreshToken,
		Scopes:       authCode.Scopes,
	}, nil
}

// ValidateAccessToken validates an OAuth2 access token.

func (m *MockOAuth2Provider) ValidateAccessToken(ctx context.Context, token string) (*OAuth2AccessToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, token)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*OAuth2AccessToken), args.Error(1)
	}

	// Default behavior
	accessToken, exists := m.fixtures.AccessTokens[token]
	if !exists || time.Now().After(accessToken.ExpiresAt) {
		return nil, fmt.Errorf("invalid or expired access token")
	}

	return &accessToken, nil
}

// OAuth2TokenResponse represents an OAuth2 token response.

type OAuth2TokenResponse struct {
	AccessToken  string   `json:"access_token"`
	TokenType    string   `json:"token_type"`
<<<<<<< HEAD
	ExpiresIn    int      `json:"expires_in"`
=======
	ExpiresIn    int64    `json:"expires_in"`
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	RefreshToken string   `json:"refresh_token,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
}

// MockSAMLProvider provides a mock SAML provider implementation.

type MockSAMLProvider struct {
	mock.Mock

	fixtures *SAMLFixtures

	mu sync.RWMutex
}

// NewMockSAMLProvider creates a new mock SAML provider.

func NewMockSAMLProvider() *MockSAMLProvider {
	return &MockSAMLProvider{
		fixtures: CreateSAMLFixtures(),
	}
}

// WithSAMLFixtures sets custom SAML fixtures.

func (m *MockSAMLProvider) WithSAMLFixtures(fixtures *SAMLFixtures) *MockSAMLProvider {
	m.fixtures = fixtures
	return m
}

// ValidateSAMLResponse validates a SAML response.

func (m *MockSAMLProvider) ValidateSAMLResponse(ctx context.Context, samlResponse string) (*SAMLAssertion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, samlResponse)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*SAMLAssertion), args.Error(1)
	}

	// Default behavior - simplified SAML validation
	// In a real implementation, this would parse and validate the XML
	return &SAMLAssertion{
		Subject:    "admin@nephoran.local",
		Issuer:     "https://sso.nephoran.local/saml/idp",
		Audience:   "https://api.nephoran.local/saml/sp",
		NotBefore:  time.Now().Add(-5 * time.Minute),
		NotOnOrAfter: time.Now().Add(1 * time.Hour),
		Attributes: map[string][]string{
			"email":      {"admin@nephoran.local"},
			"displayName": {"Administrator"},
			"roles":      {"cluster-admin", "network-admin"},
		},
	}, nil
}

// GenerateSAMLRequest generates a SAML authentication request.

func (m *MockSAMLProvider) GenerateSAMLRequest(ctx context.Context, entityID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, entityID)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.String(0), args.Error(1)
	}

	// Default behavior - return a mock SAML request
	return fmt.Sprintf(`<samlp:AuthnRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="request-%d" Version="2.0" IssueInstant="%s" Destination="https://sso.nephoran.local/saml/sso">
		<saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">%s</saml:Issuer>
		<samlp:NameIDPolicy Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress" AllowCreate="true"/>
	</samlp:AuthnRequest>`, time.Now().UnixNano(), time.Now().Format(time.RFC3339), entityID), nil
}

// SAMLAssertion represents a SAML assertion.

type SAMLAssertion struct {
	Subject      string              `json:"subject"`
	Issuer       string              `json:"issuer"`
	Audience     string              `json:"audience"`
	NotBefore    time.Time           `json:"not_before"`
	NotOnOrAfter time.Time           `json:"not_on_or_after"`
	Attributes   map[string][]string `json:"attributes"`
}

// MockLDAPProvider provides a mock LDAP provider implementation.

type MockLDAPProvider struct {
	mock.Mock

	fixtures *LDAPFixtures

	mu sync.RWMutex
}

// NewMockLDAPProvider creates a new mock LDAP provider.

func NewMockLDAPProvider() *MockLDAPProvider {
	return &MockLDAPProvider{
		fixtures: CreateLDAPFixtures(),
	}
}

// WithLDAPFixtures sets custom LDAP fixtures.

func (m *MockLDAPProvider) WithLDAPFixtures(fixtures *LDAPFixtures) *MockLDAPProvider {
	m.fixtures = fixtures
	return m
}

// Bind performs LDAP bind authentication.

func (m *MockLDAPProvider) Bind(ctx context.Context, bindDN, password string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, bindDN, password)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Error(0)
	}

	// Default behavior - validate against fixtures
	for _, user := range m.fixtures.Users {
		if user.DN == bindDN && user.UserPassword == password {
			return nil
		}
	}

	return fmt.Errorf("invalid credentials")
}

// Search performs LDAP search.

func (m *MockLDAPProvider) Search(ctx context.Context, baseDN, filter string, attributes []string) ([]LDAPSearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, baseDN, filter, attributes)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).([]LDAPSearchResult), args.Error(1)
	}

	// Default behavior - simplified search
	results := make([]LDAPSearchResult, 0)

	// Search users
	for _, user := range m.fixtures.Users {
		if m.matchesFilter(user.DN, filter) {
			result := LDAPSearchResult{
				DN: user.DN,
				Attributes: map[string][]string{
					"cn":          {user.CN},
					"uid":         {user.UID},
					"mail":        {user.Email},
					"displayName": {user.DisplayName},
					"memberOf":    user.MemberOf,
				},
			}
			results = append(results, result)
		}
	}

	// Search groups
	for _, group := range m.fixtures.Groups {
		if m.matchesFilter(group.DN, filter) {
			result := LDAPSearchResult{
				DN: group.DN,
				Attributes: map[string][]string{
					"cn":          {group.CN},
					"description": {group.Description},
					"member":      group.Members,
				},
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// matchesFilter performs simple LDAP filter matching.

func (m *MockLDAPProvider) matchesFilter(dn, filter string) bool {
	// Simplified filter matching for testing
	// In a real implementation, this would parse LDAP filter syntax
	return true // For mock purposes, return all results
}

// LDAPSearchResult represents an LDAP search result.

type LDAPSearchResult struct {
	DN         string              `json:"dn"`
	Attributes map[string][]string `json:"attributes"`
}

// MockCertificateProvider provides mock certificate validation.

type MockCertificateProvider struct {
	mock.Mock

	fixtures *AuthFixtures

	mu sync.RWMutex
}

// NewMockCertificateProvider creates a new mock certificate provider.

func NewMockCertificateProvider() *MockCertificateProvider {
	return &MockCertificateProvider{
		fixtures: DefaultAuthFixtures(),
	}
}

// WithCertificateFixtures sets certificate fixtures.

func (m *MockCertificateProvider) WithCertificateFixtures(fixtures *AuthFixtures) *MockCertificateProvider {
	m.fixtures = fixtures
	return m
}

// ValidateCertificate validates a client certificate.

func (m *MockCertificateProvider) ValidateCertificate(ctx context.Context, certPEM []byte) (*CertificateInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, certPEM)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).(*CertificateInfo), args.Error(1)
	}

	// Default behavior - find matching certificate in fixtures
	for _, cert := range m.fixtures.Certificates {
		if string(cert.Certificate) == string(certPEM) {
			return &CertificateInfo{
				CommonName: cert.CommonName,
				Subject:    cert.CommonName,
				Issuer:     "Nephoran Test CA",
				NotBefore:  time.Now().Add(-24 * time.Hour),
				NotAfter:   cert.ExpiresAt,
				Valid:      true,
			}, nil
		}
	}

	return nil, fmt.Errorf("certificate not found or invalid")
}

// CertificateInfo contains certificate validation information.

type CertificateInfo struct {
	CommonName string    `json:"common_name"`
	Subject    string    `json:"subject"`
	Issuer     string    `json:"issuer"`
	NotBefore  time.Time `json:"not_before"`
	NotAfter   time.Time `json:"not_after"`
	Valid      bool      `json:"valid"`
}

// MockMultiFactorProvider provides mock MFA implementation.

type MockMultiFactorProvider struct {
	mock.Mock

	codes map[string]MFACode

	mu sync.RWMutex
}

// MFACode represents a multi-factor authentication code.

type MFACode struct {
	Code      string    `json:"code"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"` // totp, sms, email
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}

// NewMockMultiFactorProvider creates a new mock MFA provider.

func NewMockMultiFactorProvider() *MockMultiFactorProvider {
	return &MockMultiFactorProvider{
		codes: make(map[string]MFACode),
	}
}

// GenerateMFACode generates a multi-factor authentication code.

func (m *MockMultiFactorProvider) GenerateMFACode(ctx context.Context, userID, codeType string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, userID, codeType)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.String(0), args.Error(1)
	}

	// Default behavior
	code := fmt.Sprintf("mfa-%d", time.Now().UnixNano()%1000000)
	mfaCode := MFACode{
		Code:      code,
		UserID:    userID,
		Type:      codeType,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      false,
	}

	m.codes[code] = mfaCode
	return code, nil
}

// ValidateMFACode validates a multi-factor authentication code.

func (m *MockMultiFactorProvider) ValidateMFACode(ctx context.Context, userID, code string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, userID, code)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Bool(0), args.Error(1)
	}

	// Default behavior
	mfaCode, exists := m.codes[code]
	if !exists {
		return false, fmt.Errorf("invalid MFA code")
	}

	if mfaCode.Used {
		return false, fmt.Errorf("MFA code already used")
	}

	if time.Now().After(mfaCode.ExpiresAt) {
		return false, fmt.Errorf("MFA code expired")
	}

	if mfaCode.UserID != userID {
		return false, fmt.Errorf("MFA code mismatch")
	}

	// Mark code as used
	mfaCode.Used = true
	m.codes[code] = mfaCode

	return true, nil
}

// MockAuthorizationProvider provides mock RBAC authorization.

type MockAuthorizationProvider struct {
	mock.Mock

	fixtures *AuthFixtures

	mu sync.RWMutex
}

// NewMockAuthorizationProvider creates a new mock authorization provider.

func NewMockAuthorizationProvider() *MockAuthorizationProvider {
	return &MockAuthorizationProvider{
		fixtures: DefaultAuthFixtures(),
	}
}

// WithAuthorizationFixtures sets authorization fixtures.

func (m *MockAuthorizationProvider) WithAuthorizationFixtures(fixtures *AuthFixtures) *MockAuthorizationProvider {
	m.fixtures = fixtures
	return m
}

// CheckPermission checks if a user has permission for a specific resource and action.

func (m *MockAuthorizationProvider) CheckPermission(ctx context.Context, userID, resource, action, namespace string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, userID, resource, action, namespace)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Bool(0), args.Error(1)
	}

	// Default behavior using fixtures
	permission := fmt.Sprintf("%s:%s:%s", resource, action, namespace)
	return m.fixtures.HasPermission(userID, permission), nil
}

// GetUserRoles returns the roles for a user.

func (m *MockAuthorizationProvider) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	args := m.Called(ctx, userID)

	// If mock expects are set, return the mock result
	if len(m.ExpectedCalls) > 0 {
		return args.Get(0).([]string), args.Error(1)
	}

	// Default behavior using fixtures
	user, exists := m.fixtures.GetUserByUsername(userID)
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return user.Roles, nil
}

// CreateRole creates a new role (mock implementation)
func (m *MockAuthorizationProvider) CreateRole(ctx context.Context, role interface{}) error {
	// Mock implementation - just return success
	return nil
}

// CreatePermission creates a new permission (mock implementation)
func (m *MockAuthorizationProvider) CreatePermission(ctx context.Context, permission interface{}) error {
	// Mock implementation - just return success
	return nil
}

<<<<<<< HEAD
=======
// CreateRole creates a new role (mock implementation) for RBACManagerMock
func (rbac *RBACManagerMock) CreateRole(ctx context.Context, role interface{}) (interface{}, error) {
	// Mock implementation - return the role and success
	return role, nil
}

// CreatePermission creates a new permission (mock implementation) for RBACManagerMock
func (rbac *RBACManagerMock) CreatePermission(ctx context.Context, permission interface{}) (interface{}, error) {
	// Mock implementation - return the permission and success
	return permission, nil
}

// AssignRoleToUser assigns a role to a user (mock implementation) for RBACManagerMock
func (rbac *RBACManagerMock) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	// Mock implementation - just return success
	return nil
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// ToJSON serializes mock data to JSON for debugging.

func (m *MockAuthenticator) ToJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := map[string]interface{}{
		"fixtures": m.fixtures,
		"sessions": m.sessions,
	}

	return json.MarshalIndent(data, "", "  ")
}

// Reset clears all mock data and expectations.

func (m *MockAuthenticator) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Mock.ExpectedCalls = nil
	m.Mock.Calls = nil
	m.sessions = make(map[string]*UserSession)
}

<<<<<<< HEAD
=======
// Missing Mock Type Definitions

// JWTManagerMock provides a mock JWT manager implementation
type JWTManagerMock struct {
	mock.Mock
	blacklistedTokens map[string]bool
	tokenStore        map[string]interface{}
}

// NewJWTManagerMock creates a new JWT manager mock
func NewJWTManagerMock() *JWTManagerMock {
	return &JWTManagerMock{
		blacklistedTokens: make(map[string]bool),
		tokenStore:        make(map[string]interface{}),
	}
}

// SessionManagerMock provides a mock session manager implementation
type SessionManagerMock struct {
	mock.Mock
	sessions map[string]*SessionResult
	config   *SessionConfig
}

// SessionConfig represents session configuration
type SessionConfig struct {
	SessionTTL time.Duration `json:"session_ttl"`
}

// NewSessionManagerMock creates a new session manager mock
func NewSessionManagerMock() *SessionManagerMock {
	return &SessionManagerMock{
		sessions: make(map[string]*SessionResult),
		config: &SessionConfig{
			SessionTTL: time.Hour, // Default to 1 hour
		},
	}
}

// RBACManagerMock provides a mock RBAC manager implementation
type RBACManagerMock struct {
	mock.Mock
	roles           map[string]interface{}
	permissions     map[string]interface{}
	roleStore       map[string]interface{}
	permissionStore map[string]interface{}
}

// NewRBACManagerMock creates a new RBAC manager mock
func NewRBACManagerMock() *RBACManagerMock {
	return &RBACManagerMock{
		roles:           make(map[string]interface{}),
		permissions:     make(map[string]interface{}),
		roleStore:       make(map[string]interface{}),
		permissionStore: make(map[string]interface{}),
	}
}

// SessionResult represents a session creation result
type SessionResult struct {
	ID        string      `json:"id"`
	User      interface{} `json:"user"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// Additional methods for compatibility with security tests

// GenerateToken generates a JWT token for testing
func (jsm *JWTManagerMock) GenerateToken(userInfo interface{}, claims interface{}) (string, error) {
	// Extract user information
	var username string
	switch ui := userInfo.(type) {
	case *TestUser:
		username = ui.Username
	case string:
		username = ui
	default:
		username = "test-user"
	}

	// Create JWT claims
	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"sub": username,
		"iss": "test-issuer",
		"aud": []string{"test-audience"},
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"jti": fmt.Sprintf("test-%d", now.UnixNano()),
	}

	// Add custom claims if provided
	if claims != nil {
		if claimsMap, ok := claims.(map[string]interface{}); ok {
			for k, v := range claimsMap {
				jwtClaims[k] = v
			}
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	// Use a simple secret for testing
	return token.SignedString([]byte("test-secret"))
}

// GenerateTokenPair generates both access and refresh tokens
func (jsm *JWTManagerMock) GenerateTokenPair(userInfo interface{}, claims interface{}) (string, string, error) {
	accessToken, err := jsm.GenerateToken(userInfo, claims)
	if err != nil {
		return "", "", err
	}
	refreshToken := fmt.Sprintf("refresh-%s", accessToken)
	return accessToken, refreshToken, nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (jsm *JWTManagerMock) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	return jsm.blacklistedTokens[token], nil
}

// SetSigningKey sets the signing key (mock implementation)
func (jsm *JWTManagerMock) SetSigningKey(key interface{}, keyID string) error {
	// Mock implementation - just return success
	return nil
}

// ValidateToken validates a JWT token (mock implementation)
func (jsm *JWTManagerMock) ValidateToken(ctx context.Context, tokenString string) (*auth.NephoranJWTClaims, error) {
	// Simple validation for testing
	if tokenString == "invalid" {
		return nil, fmt.Errorf("invalid token")
	}
	
	// Return basic claims for valid tokens
	now := time.Now()
	return &auth.NephoranJWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "test-user",
			Issuer:    "test-issuer",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          "Test User",
		Roles:         []string{"user"},
	}, nil
}

// RefreshToken refreshes a JWT token (mock implementation)
func (jsm *JWTManagerMock) RefreshToken(refreshToken string) (string, string, error) {
	if refreshToken == "" {
		return "", "", fmt.Errorf("empty refresh token")
	}
	
	// Generate a new access token
	accessToken, err := jsm.GenerateToken("test-user", nil)
	if err != nil {
		return "", "", err
	}
	
	// Generate a new refresh token
	newRefreshToken := fmt.Sprintf("refresh-%s", accessToken)
	return accessToken, newRefreshToken, nil
}

// BlacklistToken adds a token to the blacklist
func (jsm *JWTManagerMock) BlacklistToken(tokenString string) error {
	if jsm.blacklistedTokens == nil {
		jsm.blacklistedTokens = make(map[string]bool)
	}
	jsm.blacklistedTokens[tokenString] = true
	return nil
}

// GenerateTokenWithTTL generates a token with custom TTL
func (jsm *JWTManagerMock) GenerateTokenWithTTL(userInfo interface{}, claims interface{}, ttl time.Duration) (string, error) {
	// Extract user information
	var username string
	switch ui := userInfo.(type) {
	case *TestUser:
		username = ui.Username
	case string:
		username = ui
	default:
		username = "test-user"
	}

	// Create JWT claims with custom TTL
	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"sub": username,
		"iss": "test-issuer",
		"aud": []string{"test-audience"},
		"exp": now.Add(ttl).Unix(),
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"jti": fmt.Sprintf("test-%d", now.UnixNano()),
	}

	// Add custom claims if provided
	if claims != nil {
		if claimsMap, ok := claims.(map[string]interface{}); ok {
			for k, v := range claimsMap {
				jwtClaims[k] = v
			}
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	// Use a simple secret for testing
	return token.SignedString([]byte("test-secret"))
}

// GetPublicKey returns a public key by key ID (mock implementation)
func (jsm *JWTManagerMock) GetPublicKey(keyID string) (*rsa.PublicKey, error) {
	// Mock implementation - return a test public key
	if keyID == "" {
		return nil, fmt.Errorf("empty key ID")
	}
	
	// For testing, we'll return a dummy RSA public key
	// In a real implementation, this would retrieve the actual public key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &privateKey.PublicKey, nil
}

// GetJWKS returns the JSON Web Key Set (mock implementation)
func (jsm *JWTManagerMock) GetJWKS() (map[string]interface{}, error) {
	// Mock implementation - return a test JWKS
	return map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": "test-key-id",
				"use": "sig",
				"alg": "RS256",
				"n":   "mock-modulus",
				"e":   "AQAB",
			},
		},
	}, nil
}

// GetKeyID returns the current key ID (mock implementation)
func (jsm *JWTManagerMock) GetKeyID() string {
	return "test-key-id"
}

// RotateKeys rotates the signing keys (mock implementation)
func (jsm *JWTManagerMock) RotateKeys() error {
	// Mock implementation - just return success
	return nil
}

// ExtractClaims extracts claims from a JWT token without validation (mock implementation)
func (jsm *JWTManagerMock) ExtractClaims(tokenString string) (jwt.MapClaims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("empty token")
	}
	
	// Mock implementation - return basic claims
	now := time.Now()
	return jwt.MapClaims{
		"sub": "test-user",
		"iss": "test-issuer",
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
		"email": "test@example.com",
		"roles": []interface{}{"admin", "user"},
	}, nil
}

// CleanupBlacklist removes expired tokens from the blacklist (mock implementation)
func (jsm *JWTManagerMock) CleanupBlacklist() error {
	// Mock implementation - just return success
	// In a real implementation, this would clean up expired blacklisted tokens
	return nil
}

// GetIssuer returns the JWT issuer (mock implementation)
func (jsm *JWTManagerMock) GetIssuer() string {
	return "test-issuer"
}

// GetDefaultTTL returns the default token TTL (mock implementation)
func (jsm *JWTManagerMock) GetDefaultTTL() time.Duration {
	return time.Hour
}

// GetRefreshTTL returns the refresh token TTL (mock implementation)
func (jsm *JWTManagerMock) GetRefreshTTL() time.Duration {
	return 24 * time.Hour
}

// GetRequireSecureCookies returns whether secure cookies are required (mock implementation)
func (jsm *JWTManagerMock) GetRequireSecureCookies() bool {
	return false
}

// Close cleans up the JWT manager (mock implementation)
func (jsm *JWTManagerMock) Close() {
	// Mock implementation - nothing to clean up
}


// CreateSession creates a new session (compatible with both interfaces)
func (ssm *SessionManagerMock) CreateSession(ctx context.Context, userInfo interface{}, metadata ...interface{}) (*SessionResult, error) {
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	session := &SessionResult{
		ID:        sessionID,
		User:      userInfo,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	ssm.sessions[sessionID] = session
	return session, nil
}

<<<<<<< HEAD
=======
// GetSession retrieves a session by ID
func (ssm *SessionManagerMock) GetSession(ctx context.Context, sessionID string) (*SessionResult, error) {
	session, exists := ssm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// ValidateSession validates an existing session
func (ssm *SessionManagerMock) ValidateSession(ctx context.Context, sessionID string) (*SessionResult, error) {
	session, exists := ssm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}
	
	return session, nil
}

<<<<<<< HEAD
=======
// RefreshSession refreshes an existing session
func (ssm *SessionManagerMock) RefreshSession(ctx context.Context, sessionID string) (*SessionResult, error) {
	session, exists := ssm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// Extend the expiration time
	session.ExpiresAt = time.Now().Add(24 * time.Hour)
	ssm.sessions[sessionID] = session
	return session, nil
}

// RevokeSession revokes/deletes a session
func (ssm *SessionManagerMock) RevokeSession(ctx context.Context, sessionID string) error {
	_, exists := ssm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	delete(ssm.sessions, sessionID)
	return nil
}

// RevokeAllUserSessions revokes all sessions for a user
func (ssm *SessionManagerMock) RevokeAllUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("empty user ID")
	}

	// For simplicity, remove all sessions (in a real implementation, we'd check user association)
	for sessionID, session := range ssm.sessions {
		if user, ok := session.User.(*TestUser); ok && user.Subject == userID {
			delete(ssm.sessions, sessionID)
		}
	}
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (ssm *SessionManagerMock) CleanupExpiredSessions() error {
	now := time.Now()
	for sessionID, session := range ssm.sessions {
		if now.After(session.ExpiresAt) {
			delete(ssm.sessions, sessionID)
		}
	}
	return nil
}

// GetUserSessions returns all sessions for a user
func (ssm *SessionManagerMock) GetUserSessions(ctx context.Context, userID string) ([]*SessionResult, error) {
	if userID == "" {
		return nil, fmt.Errorf("empty user ID")
	}

	var userSessions []*SessionResult
	for _, session := range ssm.sessions {
		if user, ok := session.User.(*TestUser); ok && user.Subject == userID {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions, nil
}

// UpdateSessionMetadata updates session metadata
func (ssm *SessionManagerMock) UpdateSessionMetadata(ctx context.Context, sessionID string, metadata map[string]interface{}) (*SessionResult, error) {
	session, exists := ssm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// For simplicity, just return the session (metadata updates would be stored in a real implementation)
	return session, nil
}

// SetSessionCookie sets a session cookie (mock implementation)
func (ssm *SessionManagerMock) SetSessionCookie(w http.ResponseWriter, sessionID string) {
	cookie := http.Cookie{
		Name:     "test-session",
		Value:    sessionID,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
	http.SetCookie(w, &cookie)
}

// ClearSessionCookie clears the session cookie
func (ssm *SessionManagerMock) ClearSessionCookie(w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:     "test-session",
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Path:     "/",
	}
	http.SetCookie(w, &cookie)
}

// GetSessionFromCookie retrieves session ID from cookie
func (ssm *SessionManagerMock) GetSessionFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("test-session")
	if err != nil {
		return "", fmt.Errorf("session cookie not found")
	}
	return cookie.Value, nil
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// NewOAuth2MockServer creates a mock OAuth2 server for testing
func NewOAuth2MockServer(issuer string) *httptest.Server {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"access_token": "mock-access-token",
			"token_type": "Bearer",
			"expires_in": 3600,
		}
		json.NewEncoder(w).Encode(response)
	})
	
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"sub": "test-user",
			"name": "Test User",
			"email": "test@example.com",
		}
		json.NewEncoder(w).Encode(response)
	})
	
	return httptest.NewServer(mux)
}

// MockOAuthProvider provides a testify compatible OAuth provider mock
type MockOAuthProvider struct {
	mock.Mock
}

// NewMockOAuthProvider creates a simple mock OAuth provider
func NewMockOAuthProvider(provider ...string) *MockOAuthProvider {
	return &MockOAuthProvider{}
}

// On provides testify mock functionality
func (m *MockOAuthProvider) On(methodName string, args ...interface{}) *mock.Call {
	return m.Mock.On(methodName, args...)
}

// GetAuthURL returns a mock auth URL
func (m *MockOAuthProvider) GetAuthURL(state string) string {
	args := m.Called(state)
	return args.String(0)
}

// Exchange exchanges code for token
func (m *MockOAuthProvider) Exchange(code string) (interface{}, error) {
	args := m.Called(code)
	return args.Get(0), args.Error(1)
}

// MockTokenStore provides a mock token store implementation
type MockTokenStore struct {
	tokens map[string]*auth.TokenInfo
}

// NewMockTokenStore creates a new mock token store
func NewMockTokenStore() *MockTokenStore {
	return &MockTokenStore{
		tokens: make(map[string]*auth.TokenInfo),
	}
}

// StoreToken stores a token with expiration
func (mts *MockTokenStore) StoreToken(ctx context.Context, tokenID string, token *auth.TokenInfo) error {
	mts.tokens[tokenID] = token
	return nil
}

// GetToken retrieves token info
func (mts *MockTokenStore) GetToken(ctx context.Context, tokenID string) (*auth.TokenInfo, error) {
	token, exists := mts.tokens[tokenID]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}
	return token, nil
}

// UpdateToken updates token info
func (mts *MockTokenStore) UpdateToken(ctx context.Context, tokenID string, token *auth.TokenInfo) error {
	mts.tokens[tokenID] = token
	return nil
}

// DeleteToken deletes a token
func (mts *MockTokenStore) DeleteToken(ctx context.Context, tokenID string) error {
	delete(mts.tokens, tokenID)
	return nil
}

// ListUserTokens lists tokens for a user
func (mts *MockTokenStore) ListUserTokens(ctx context.Context, userID string) ([]*auth.TokenInfo, error) {
	var userTokens []*auth.TokenInfo
	for _, token := range mts.tokens {
		if token.UserID == userID {
			userTokens = append(userTokens, token)
		}
	}
	return userTokens, nil
}

// CleanupExpired removes expired tokens (mock implementation)
func (mts *MockTokenStore) CleanupExpired(ctx context.Context) error {
	// Mock implementation - don't actually clean up for testing
	return nil
}

// MockTokenBlacklist provides a mock token blacklist implementation
type MockTokenBlacklist struct {
	blacklist map[string]time.Time
}

// NewMockTokenBlacklist creates a new mock token blacklist
func NewMockTokenBlacklist() *MockTokenBlacklist {
	return &MockTokenBlacklist{
		blacklist: make(map[string]time.Time),
	}
}

// BlacklistToken adds a token to the blacklist
func (mtb *MockTokenBlacklist) BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	mtb.blacklist[tokenID] = expiresAt
	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (mtb *MockTokenBlacklist) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	_, exists := mtb.blacklist[tokenID]
	return exists, nil
}

// CleanupExpired removes expired entries
func (mtb *MockTokenBlacklist) CleanupExpired(ctx context.Context) error {
	now := time.Now()
	for tokenID, expiresAt := range mtb.blacklist {
		if expiresAt.Before(now) {
			delete(mtb.blacklist, tokenID)
		}
	}
	return nil
}

// NewMockSlogLogger creates a mock slog.Logger for testing
func NewMockSlogLogger() *slog.Logger {
	// Create a no-op logger for testing
	return slog.New(slog.NewTextHandler(io.Discard, nil))
<<<<<<< HEAD
=======
}

// Factory Functions for Test Data Generation

// UserFactory provides factory methods for creating test users
type UserFactory struct{}

// NewUserFactory creates a new user factory
func NewUserFactory() *UserFactory {
	return &UserFactory{}
}

// CreateBasicUser creates a basic test user
func (uf *UserFactory) CreateBasicUser() *TestUser {
	return &TestUser{
		Username:      "testuser",
		Password:      "password123",
		Email:         "test@example.com",
		Roles:         []string{"user"},
		Claims:        map[string]interface{}{"test": true},
		Enabled:       true,
		Subject:       "test-user-123",
		Name:          "Test User",
		EmailVerified: true,
		Provider:      "test",
	}
}

// TokenFactory provides factory methods for creating JWT tokens
type TokenFactory struct {
	issuer string
}

// NewTokenFactory creates a new token factory
func NewTokenFactory(issuer string) *TokenFactory {
	return &TokenFactory{issuer: issuer}
}

// CreateBasicToken creates basic JWT claims
func (tf *TokenFactory) CreateBasicToken(subject string) map[string]interface{} {
	now := time.Now()
	return map[string]interface{}{
		"iss": tf.issuer,
		"sub": subject,
		"aud": []string{"test-audience"},
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"jti": fmt.Sprintf("token-%d", now.UnixNano()),
	}
}

// CreateExpiredToken creates an expired JWT token
func (tf *TokenFactory) CreateExpiredToken(subject string) map[string]interface{} {
	now := time.Now()
	return map[string]interface{}{
		"iss": tf.issuer,
		"sub": subject,
		"aud": []string{"test-audience"},
		"exp": now.Add(-time.Hour).Unix(), // Expired 1 hour ago
		"iat": now.Add(-2 * time.Hour).Unix(),
		"nbf": now.Add(-2 * time.Hour).Unix(),
		"jti": fmt.Sprintf("expired-token-%d", now.UnixNano()),
	}
}

// CreateTokenNotValidYet creates a JWT token not valid yet
func (tf *TokenFactory) CreateTokenNotValidYet(subject string) map[string]interface{} {
	now := time.Now()
	return map[string]interface{}{
		"iss": tf.issuer,
		"sub": subject,
		"aud": []string{"test-audience"},
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
		"nbf": now.Add(time.Hour).Unix(), // Not before 1 hour from now
		"jti": fmt.Sprintf("future-token-%d", now.UnixNano()),
	}
}

// OAuthResponseFactory provides factory methods for OAuth responses
type OAuthResponseFactory struct{}

// NewOAuthResponseFactory creates a new OAuth response factory
func NewOAuthResponseFactory() *OAuthResponseFactory {
	return &OAuthResponseFactory{}
}

// CreateTokenResponse creates a mock OAuth2 token response
func (of *OAuthResponseFactory) CreateTokenResponse() *OAuth2TokenResponse {
	return &OAuth2TokenResponse{
		AccessToken:  fmt.Sprintf("access-token-%d", time.Now().UnixNano()),
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: fmt.Sprintf("refresh-token-%d", time.Now().UnixNano()),
		Scopes:       []string{"read", "write"},
	}
}

// RoleFactory provides factory methods for creating test roles
type RoleFactory struct{}

// NewRoleFactory creates a new role factory
func NewRoleFactory() *RoleFactory {
	return &RoleFactory{}
}

// CreateBasicRole creates a basic test role
func (rf *RoleFactory) CreateBasicRole() map[string]interface{} {
	return map[string]interface{}{
		"name":        "test-role",
		"description": "Test role for testing purposes",
		"permissions": []string{"read", "write"},
	}
}

// CreateRoleWithPermissions creates a role with specific permissions
func (rf *RoleFactory) CreateRoleWithPermissions(name string, permissions []string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"description": fmt.Sprintf("Role %s with specific permissions", name),
		"permissions": permissions,
	}
}

// PermissionFactory provides factory methods for creating test permissions
type PermissionFactory struct{}

// NewPermissionFactory creates a new permission factory
func NewPermissionFactory() *PermissionFactory {
	return &PermissionFactory{}
}

// CreateBasicPermission creates a basic test permission
func (pf *PermissionFactory) CreateBasicPermission() map[string]interface{} {
	return map[string]interface{}{
		"name":        "test-permission",
		"description": "Test permission for testing purposes",
		"resource":    "test-resource",
		"action":      "read",
	}
}

// CreatePermission creates a permission with specific resource and action
func (pf *PermissionFactory) CreatePermission(resource, action, scope string) map[string]interface{} {
	return map[string]interface{}{
		"name":        fmt.Sprintf("%s-%s-%s", resource, action, scope),
		"description": fmt.Sprintf("Permission to %s on %s in %s", action, resource, scope),
		"resource":    resource,
		"action":      action,
		"scope":       scope,
	}
}

// CreateCompleteTestSetup creates all factory instances
func CreateCompleteTestSetup() (*UserFactory, *TokenFactory, *OAuthResponseFactory, interface{}, interface{}, interface{}, interface{}) {
	uf := NewUserFactory()
	tf := NewTokenFactory("test-issuer")
	of := NewOAuthResponseFactory()
	rf := NewRoleFactory()
	pf := NewPermissionFactory()
	// Return placeholder interfaces for the other factories
	var cf, sf interface{}
	cf = &struct{}{}
	sf = &struct{}{}
	return uf, tf, of, cf, rf, pf, sf
}

// CreateTestData creates comprehensive test data
func CreateTestData() map[string]interface{} {
	return map[string]interface{}{
		"users": map[string]interface{}{
			"basic": map[string]interface{}{
				"username":       "basic-user",
				"email":          "basic@nephoran.local",
				"email_verified": true,
			},
			"admin": map[string]interface{}{
				"username":       "admin-user",
				"email":          "admin@nephoran.local",
				"email_verified": true,
				"roles":          []string{"admin"},
			},
			"github": map[string]interface{}{
				"username":       "github-user",
				"email":          "github@example.com",
				"email_verified": true,
				"provider":       "github",
			},
		},
		"tokens": map[string]interface{}{
			"valid":   "valid-test-token",
			"expired": "expired-test-token",
			"invalid": "invalid-test-token",
		},
		"roles": map[string]interface{}{
			"admin": []string{"create", "read", "update", "delete"},
			"user":  []string{"read"},
		},
		"permissions": map[string]interface{}{
			"create": "permission to create resources",
			"read":   "permission to read resources",
			"update": "permission to update resources",
			"delete": "permission to delete resources",
		},
		"sessions": map[string]interface{}{
			"active":  "active-session-id",
			"expired": "expired-session-id",
		},
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}