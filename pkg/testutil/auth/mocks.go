package authtestutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// MockOAuthProvider provides a mock implementation of the OAuthProvider interface.

type MockOAuthProvider struct {
	mock.Mock

	Name string
}

// NewMockOAuthProvider performs newmockoauthprovider operation.

func NewMockOAuthProvider(name string) *MockOAuthProvider {

	return &MockOAuthProvider{Name: name}

}

// GetProviderName performs getprovidername operation.

func (m *MockOAuthProvider) GetProviderName() string {

	args := m.Called()

	if m.Name != "" {

		return m.Name

	}

	return args.String(0)

}

// GetAuthorizationURL performs getauthorizationurl operation.

func (m *MockOAuthProvider) GetAuthorizationURL(state, redirectURI string, options ...providers.AuthOption) (string, *providers.PKCEChallenge, error) {

	args := m.Called(state, redirectURI, options)

	return args.String(0), args.Get(1).(*providers.PKCEChallenge), args.Error(2)

}

// ExchangeCodeForToken performs exchangecodefortoken operation.

func (m *MockOAuthProvider) ExchangeCodeForToken(ctx context.Context, code, redirectURI string, challenge *providers.PKCEChallenge) (*providers.TokenResponse, error) {

	args := m.Called(ctx, code, redirectURI, challenge)

	return args.Get(0).(*providers.TokenResponse), args.Error(1)

}

// RefreshToken performs refreshtoken operation.

func (m *MockOAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*providers.TokenResponse, error) {

	args := m.Called(ctx, refreshToken)

	return args.Get(0).(*providers.TokenResponse), args.Error(1)

}

// GetUserInfo performs getuserinfo operation.

func (m *MockOAuthProvider) GetUserInfo(ctx context.Context, accessToken string) (*providers.UserInfo, error) {

	args := m.Called(ctx, accessToken)

	return args.Get(0).(*providers.UserInfo), args.Error(1)

}

// ValidateToken performs validatetoken operation.

func (m *MockOAuthProvider) ValidateToken(ctx context.Context, accessToken string) (*providers.TokenValidation, error) {

	args := m.Called(ctx, accessToken)

	return args.Get(0).(*providers.TokenValidation), args.Error(1)

}

// RevokeToken performs revoketoken operation.

func (m *MockOAuthProvider) RevokeToken(ctx context.Context, token string) error {

	args := m.Called(ctx, token)

	return args.Error(0)

}

// SupportsFeature performs supportsfeature operation.

func (m *MockOAuthProvider) SupportsFeature(feature providers.ProviderFeature) bool {

	args := m.Called(feature)

	return args.Bool(0)

}

// GetConfiguration performs getconfiguration operation.

func (m *MockOAuthProvider) GetConfiguration() *providers.ProviderConfig {

	args := m.Called()

	return args.Get(0).(*providers.ProviderConfig)

}

// MockOIDCProvider extends MockOAuthProvider with OIDC functionality.

type MockOIDCProvider struct {
	*MockOAuthProvider
}

// NewMockOIDCProvider performs newmockoidcprovider operation.

func NewMockOIDCProvider(name string) *MockOIDCProvider {

	return &MockOIDCProvider{

		MockOAuthProvider: NewMockOAuthProvider(name),
	}

}

// DiscoverConfiguration performs discoverconfiguration operation.

func (m *MockOIDCProvider) DiscoverConfiguration(ctx context.Context) (*providers.OIDCConfiguration, error) {

	args := m.Called(ctx)

	return args.Get(0).(*providers.OIDCConfiguration), args.Error(1)

}

// ValidateIDToken performs validateidtoken operation.

func (m *MockOIDCProvider) ValidateIDToken(ctx context.Context, idToken string) (*providers.IDTokenClaims, error) {

	args := m.Called(ctx, idToken)

	return args.Get(0).(*providers.IDTokenClaims), args.Error(1)

}

// GetJWKS performs getjwks operation.

func (m *MockOIDCProvider) GetJWKS(ctx context.Context) (*providers.JWKS, error) {

	args := m.Called(ctx)

	return args.Get(0).(*providers.JWKS), args.Error(1)

}

// GetUserInfoFromIDToken performs getuserinfofromidtoken operation.

func (m *MockOIDCProvider) GetUserInfoFromIDToken(idToken string) (*providers.UserInfo, error) {

	args := m.Called(idToken)

	return args.Get(0).(*providers.UserInfo), args.Error(1)

}

// MockEnterpriseProvider extends MockOAuthProvider with enterprise features.

type MockEnterpriseProvider struct {
	*MockOAuthProvider
}

// NewMockEnterpriseProvider performs newmockenterpriseprovider operation.

func NewMockEnterpriseProvider(name string) *MockEnterpriseProvider {

	return &MockEnterpriseProvider{

		MockOAuthProvider: NewMockOAuthProvider(name),
	}

}

// GetGroups performs getgroups operation.

func (m *MockEnterpriseProvider) GetGroups(ctx context.Context, accessToken string) ([]string, error) {

	args := m.Called(ctx, accessToken)

	return args.Get(0).([]string), args.Error(1)

}

// GetRoles performs getroles operation.

func (m *MockEnterpriseProvider) GetRoles(ctx context.Context, accessToken string) ([]string, error) {

	args := m.Called(ctx, accessToken)

	return args.Get(0).([]string), args.Error(1)

}

// CheckGroupMembership performs checkgroupmembership operation.

func (m *MockEnterpriseProvider) CheckGroupMembership(ctx context.Context, accessToken string, groups []string) ([]string, error) {

	args := m.Called(ctx, accessToken, groups)

	return args.Get(0).([]string), args.Error(1)

}

// GetOrganizations performs getorganizations operation.

func (m *MockEnterpriseProvider) GetOrganizations(ctx context.Context, accessToken string) ([]providers.Organization, error) {

	args := m.Called(ctx, accessToken)

	return args.Get(0).([]providers.Organization), args.Error(1)

}

// ValidateUserAccess performs validateuseraccess operation.

func (m *MockEnterpriseProvider) ValidateUserAccess(ctx context.Context, accessToken string, requiredLevel providers.AccessLevel) error {

	args := m.Called(ctx, accessToken, requiredLevel)

	return args.Error(0)

}

// MockLDAPProvider provides a mock implementation of the LDAPProvider interface.

type MockLDAPProvider struct {
	mock.Mock
}

// NewMockLDAPProvider performs newmockldapprovider operation.

func NewMockLDAPProvider() *MockLDAPProvider {

	return &MockLDAPProvider{}

}

// Connect performs connect operation.

func (m *MockLDAPProvider) Connect(ctx context.Context) error {

	args := m.Called(ctx)

	return args.Error(0)

}

// Authenticate performs authenticate operation.

func (m *MockLDAPProvider) Authenticate(ctx context.Context, username, password string) (*providers.UserInfo, error) {

	args := m.Called(ctx, username, password)

	return args.Get(0).(*providers.UserInfo), args.Error(1)

}

// SearchUser performs searchuser operation.

func (m *MockLDAPProvider) SearchUser(ctx context.Context, username string) (*providers.UserInfo, error) {

	args := m.Called(ctx, username)

	return args.Get(0).(*providers.UserInfo), args.Error(1)

}

// GetUserGroups performs getusergroups operation.

func (m *MockLDAPProvider) GetUserGroups(ctx context.Context, username string) ([]string, error) {

	args := m.Called(ctx, username)

	return args.Get(0).([]string), args.Error(1)

}

// GetUserRoles performs getuserroles operation.

func (m *MockLDAPProvider) GetUserRoles(ctx context.Context, username string) ([]string, error) {

	args := m.Called(ctx, username)

	return args.Get(0).([]string), args.Error(1)

}

// ValidateUserAttributes performs validateuserattributes operation.

func (m *MockLDAPProvider) ValidateUserAttributes(ctx context.Context, username string, requiredAttrs map[string]string) error {

	args := m.Called(ctx, username, requiredAttrs)

	return args.Error(0)

}

// Close performs close operation.

func (m *MockLDAPProvider) Close() error {

	args := m.Called()

	return args.Error(0)

}

// MockLDAPServer provides an in-memory LDAP server for testing.

type MockLDAPServer struct {
	users map[string]*LDAPUser

	groups map[string]*LDAPGroup

	binds map[string]string // username -> password

	mutex sync.RWMutex

	closed bool
}

// LDAPUser represents a ldapuser.

type LDAPUser struct {
	DN string `json:"dn"`

	Username string `json:"username"`

	Email string `json:"email"`

	Name string `json:"name"`

	GivenName string `json:"given_name"`

	Surname string `json:"surname"`

	Groups []string `json:"groups"`

	Attributes map[string]string `json:"attributes"`
}

// LDAPGroup represents a ldapgroup.

type LDAPGroup struct {
	DN string `json:"dn"`

	Name string `json:"name"`

	Description string `json:"description"`

	Members []string `json:"members"`
}

// NewMockLDAPServer performs newmockldapserver operation.

func NewMockLDAPServer() *MockLDAPServer {

	server := &MockLDAPServer{

		users: make(map[string]*LDAPUser),

		groups: make(map[string]*LDAPGroup),

		binds: make(map[string]string),
	}

	// Add default test users.

	server.AddUser(&LDAPUser{

		DN: "cn=testuser,ou=users,dc=example,dc=com",

		Username: "testuser",

		Email: "testuser@example.com",

		Name: "Test User",

		GivenName: "Test",

		Surname: "User",

		Groups: []string{"users", "developers"},

		Attributes: map[string]string{

			"department": "engineering",

			"title": "software engineer",
		},
	})

	server.SetPassword("testuser", "password123")

	server.AddUser(&LDAPUser{

		DN: "cn=admin,ou=users,dc=example,dc=com",

		Username: "admin",

		Email: "admin@example.com",

		Name: "Admin User",

		GivenName: "Admin",

		Surname: "User",

		Groups: []string{"users", "admins"},

		Attributes: map[string]string{

			"department": "operations",

			"title": "system administrator",
		},
	})

	server.SetPassword("admin", "admin123")

	// Add default test groups.

	server.AddGroup(&LDAPGroup{

		DN: "cn=users,ou=groups,dc=example,dc=com",

		Name: "users",

		Description: "All users",

		Members: []string{"testuser", "admin"},
	})

	server.AddGroup(&LDAPGroup{

		DN: "cn=developers,ou=groups,dc=example,dc=com",

		Name: "developers",

		Description: "Software developers",

		Members: []string{"testuser"},
	})

	server.AddGroup(&LDAPGroup{

		DN: "cn=admins,ou=groups,dc=example,dc=com",

		Name: "admins",

		Description: "System administrators",

		Members: []string{"admin"},
	})

	return server

}

// AddUser performs adduser operation.

func (s *MockLDAPServer) AddUser(user *LDAPUser) {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	s.users[user.Username] = user

}

// AddGroup performs addgroup operation.

func (s *MockLDAPServer) AddGroup(group *LDAPGroup) {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	s.groups[group.Name] = group

}

// SetPassword performs setpassword operation.

func (s *MockLDAPServer) SetPassword(username, password string) {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	s.binds[username] = password

}

// GetUser performs getuser operation.

func (s *MockLDAPServer) GetUser(username string) *LDAPUser {

	s.mutex.RLock()

	defer s.mutex.RUnlock()

	return s.users[username]

}

// GetGroup performs getgroup operation.

func (s *MockLDAPServer) GetGroup(name string) *LDAPGroup {

	s.mutex.RLock()

	defer s.mutex.RUnlock()

	return s.groups[name]

}

// Authenticate performs authenticate operation.

func (s *MockLDAPServer) Authenticate(username, password string) bool {

	s.mutex.RLock()

	defer s.mutex.RUnlock()

	expectedPassword, exists := s.binds[username]

	return exists && expectedPassword == password

}

// Close performs close operation.

func (s *MockLDAPServer) Close() error {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	s.closed = true

	return nil

}

// IsClosed performs isclosed operation.

func (s *MockLDAPServer) IsClosed() bool {

	s.mutex.RLock()

	defer s.mutex.RUnlock()

	return s.closed

}

// OAuth2 Mock Server for different providers.

type OAuth2MockServer struct {
	Server *httptest.Server

	Provider string

	Users map[string]*providers.UserInfo

	Tokens map[string]*providers.TokenResponse

	InvalidTokens map[string]bool

	mutex sync.RWMutex
}

// NewOAuth2MockServer performs newoauth2mockserver operation.

func NewOAuth2MockServer(provider string) *OAuth2MockServer {

	ms := &OAuth2MockServer{

		Provider: provider,

		Users: make(map[string]*providers.UserInfo),

		Tokens: make(map[string]*providers.TokenResponse),

		InvalidTokens: make(map[string]bool),
	}

	// Setup default test data.

	ms.setupDefaultData()

	// Create HTTP server.

	mux := http.NewServeMux()

	ms.setupEndpoints(mux)

	ms.Server = httptest.NewServer(mux)

	return ms

}

func (ms *OAuth2MockServer) setupDefaultData() {

	// Add test users.

	testUser := &providers.UserInfo{

		Subject: "test-user-123",

		Email: "testuser@example.com",

		EmailVerified: true,

		Name: "Test User",

		GivenName: "Test",

		FamilyName: "User",

		Username: "testuser",

		Provider: ms.Provider,

		ProviderID: fmt.Sprintf("%s-test-user-123", ms.Provider),

		Groups: []string{"users", "testers"},

		Roles: []string{"viewer"},
	}

	adminUser := &providers.UserInfo{

		Subject: "admin-user-456",

		Email: "admin@example.com",

		EmailVerified: true,

		Name: "Admin User",

		GivenName: "Admin",

		FamilyName: "User",

		Username: "admin",

		Provider: ms.Provider,

		ProviderID: fmt.Sprintf("%s-admin-user-456", ms.Provider),

		Groups: []string{"users", "admins"},

		Roles: []string{"admin"},
	}

	ms.Users["test-access-token"] = testUser

	ms.Users["admin-access-token"] = adminUser

	// Add token responses.

	ms.Tokens["test-auth-code"] = &providers.TokenResponse{

		AccessToken: "test-access-token",

		RefreshToken: "test-refresh-token",

		TokenType: "Bearer",

		ExpiresIn: 3600,

		Scope: "openid email profile",

		IssuedAt: time.Now(),
	}

	ms.Tokens["admin-auth-code"] = &providers.TokenResponse{

		AccessToken: "admin-access-token",

		RefreshToken: "admin-refresh-token",

		TokenType: "Bearer",

		ExpiresIn: 3600,

		Scope: "openid email profile",

		IssuedAt: time.Now(),
	}

}

func (ms *OAuth2MockServer) setupEndpoints(mux *http.ServeMux) {

	// Authorization endpoint.

	mux.HandleFunc("/oauth/authorize", ms.handleAuthorize)

	mux.HandleFunc("/auth", ms.handleAuthorize) // Alternative path

	// Token endpoint.

	mux.HandleFunc("/oauth/token", ms.handleToken)

	mux.HandleFunc("/token", ms.handleToken) // Alternative path

	// User info endpoint.

	mux.HandleFunc("/user", ms.handleUserInfo)

	mux.HandleFunc("/userinfo", ms.handleUserInfo) // OIDC standard

	// OIDC endpoints.

	mux.HandleFunc("/.well-known/openid_configuration", ms.handleOIDCDiscovery)

	mux.HandleFunc("/.well-known/jwks.json", ms.handleJWKS)

	// Token revocation.

	mux.HandleFunc("/oauth/revoke", ms.handleRevoke)

}

func (ms *OAuth2MockServer) handleAuthorize(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	redirectURI := query.Get("redirect_uri")

	state := query.Get("state")

	if redirectURI == "" {

		http.Error(w, "Missing redirect_uri", http.StatusBadRequest)

		return

	}

	// Generate mock authorization code.

	code := "test-auth-code"

	if query.Get("error") == "access_denied" {

		code = "denied-auth-code"

	}

	// Redirect with code.

	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
		return
	}

	q := redirectURL.Query()

	q.Set("code", code)

	if state != "" {

		q.Set("state", state)

	}

	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)

}

func (ms *OAuth2MockServer) handleToken(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return

	}

	if err := r.ParseForm(); err != nil {

		http.Error(w, "Invalid form data", http.StatusBadRequest)

		return

	}

	grantType := r.FormValue("grant_type")

	switch grantType {

	case "authorization_code":

		ms.handleAuthorizationCodeGrant(w, r)

	case "refresh_token":

		ms.handleRefreshTokenGrant(w, r)

	default:

		http.Error(w, "Unsupported grant type", http.StatusBadRequest)

	}

}

func (ms *OAuth2MockServer) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {

	code := r.FormValue("code")

	ms.mutex.RLock()

	tokenResponse, exists := ms.Tokens[code]

	ms.mutex.RUnlock()

	if !exists {

		http.Error(w, "Invalid authorization code", http.StatusBadRequest)

		return

	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(tokenResponse); err != nil {
		http.Error(w, "Failed to encode token response", http.StatusInternalServerError)
		return
	}

}

func (ms *OAuth2MockServer) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {

	refreshToken := r.FormValue("refresh_token")

	// Find token by refresh token.

	ms.mutex.RLock()

	var tokenResponse *providers.TokenResponse

	for _, token := range ms.Tokens {

		if token.RefreshToken == refreshToken {

			tokenResponse = &providers.TokenResponse{

				AccessToken: token.AccessToken + "-refreshed",

				RefreshToken: token.RefreshToken,

				TokenType: token.TokenType,

				ExpiresIn: 3600,

				Scope: token.Scope,

				IssuedAt: time.Now(),
			}

			break

		}

	}

	ms.mutex.RUnlock()

	if tokenResponse == nil {

		http.Error(w, "Invalid refresh token", http.StatusBadRequest)

		return

	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(tokenResponse); err != nil {
		http.Error(w, "Failed to encode refresh token response", http.StatusInternalServerError)
		return
	}

}

func (ms *OAuth2MockServer) handleUserInfo(w http.ResponseWriter, r *http.Request) {

	authHeader := r.Header.Get("Authorization")

	if !strings.HasPrefix(authHeader, "Bearer ") {

		http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)

		return

	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	ms.mutex.RLock()

	user, exists := ms.Users[token]

	isInvalid := ms.InvalidTokens[token]

	ms.mutex.RUnlock()

	if isInvalid {

		http.Error(w, "Token has been revoked", http.StatusUnauthorized)

		return

	}

	if !exists {

		http.Error(w, "Invalid access token", http.StatusUnauthorized)

		return

	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

}

func (ms *OAuth2MockServer) handleOIDCDiscovery(w http.ResponseWriter, r *http.Request) {

	baseURL := ms.Server.URL

	config := map[string]interface{}{

		"issuer": baseURL,

		"authorization_endpoint": baseURL + "/oauth/authorize",

		"token_endpoint": baseURL + "/oauth/token",

		"userinfo_endpoint": baseURL + "/userinfo",

		"jwks_uri": baseURL + "/.well-known/jwks.json",

		"scopes_supported": []string{

			"openid", "email", "profile",
		},

		"response_types_supported": []string{

			"code",
		},

		"grant_types_supported": []string{

			"authorization_code", "refresh_token",
		},

		"subject_types_supported": []string{

			"public",
		},

		"id_token_signing_alg_values_supported": []string{

			"RS256",
		},

		"code_challenge_methods_supported": []string{

			"S256",
		},
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Failed to encode config", http.StatusInternalServerError)
		return
	}

}

func (ms *OAuth2MockServer) handleJWKS(w http.ResponseWriter, r *http.Request) {

	jwks := map[string]interface{}{

		"keys": []map[string]interface{}{

			{

				"kty": "RSA",

				"kid": "test-key-1",

				"use": "sig",

				"alg": "RS256",

				"n": "test-modulus",

				"e": "AQAB",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(jwks); err != nil {
		http.Error(w, "Failed to encode JWKS", http.StatusInternalServerError)
		return
	}

}

func (ms *OAuth2MockServer) handleRevoke(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return

	}

	if err := r.ParseForm(); err != nil {

		http.Error(w, "Invalid form data", http.StatusBadRequest)

		return

	}

	token := r.FormValue("token")

	if token == "" {

		http.Error(w, "Missing token", http.StatusBadRequest)

		return

	}

	ms.mutex.Lock()

	ms.InvalidTokens[token] = true

	ms.mutex.Unlock()

	w.WriteHeader(http.StatusOK)

}

// AddUser performs adduser operation.

func (ms *OAuth2MockServer) AddUser(token string, user *providers.UserInfo) {

	ms.mutex.Lock()

	defer ms.mutex.Unlock()

	ms.Users[token] = user

}

// InvalidateToken performs invalidatetoken operation.

func (ms *OAuth2MockServer) InvalidateToken(token string) {

	ms.mutex.Lock()

	defer ms.mutex.Unlock()

	ms.InvalidTokens[token] = true

}

// Close performs close operation.

func (ms *OAuth2MockServer) Close() {

	if ms.Server != nil {

		ms.Server.Close()

	}

}
