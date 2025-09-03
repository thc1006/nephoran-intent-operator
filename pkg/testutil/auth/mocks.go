// Package auth provides authentication mocks and test utilities for the Nephoran Intent Operator.

// This package contains mock implementations for various authentication providers and services.

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
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
	ExpiresIn    int      `json:"expires_in"`
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