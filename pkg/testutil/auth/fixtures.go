// Package auth provides authentication test fixtures and utilities for the Nephoran Intent Operator.

// This package contains test data and helper functions for testing authentication scenarios.

package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestUser represents a test user for authentication scenarios.

type TestUser struct {
	Username string            `json:"username"`
	Password string            `json:"password"`
	Email    string            `json:"email"`
	Roles    []string          `json:"roles"`
	Claims   map[string]interface{} `json:"claims"`
	Enabled  bool              `json:"enabled"`
}

// TestCertificate represents a test certificate for mTLS scenarios.

type TestCertificate struct {
	Name        string `json:"name"`
	CommonName  string `json:"common_name"`
	Certificate []byte `json:"certificate"`
	PrivateKey  []byte `json:"private_key"`
	CA          []byte `json:"ca,omitempty"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// TestAuthConfig represents test authentication configuration.

type TestAuthConfig struct {
	JWTSecret     string                 `json:"jwt_secret"`
	TokenDuration time.Duration          `json:"token_duration"`
	Issuer        string                 `json:"issuer"`
	Audience      []string               `json:"audience"`
	Claims        map[string]interface{} `json:"claims"`
}

// AuthFixtures contains all authentication test fixtures.

type AuthFixtures struct {
	Users        []TestUser       `json:"users"`
	Roles        []TestRole       `json:"roles"` // Uses TestRole from contracts.go
	Certificates []TestCertificate `json:"certificates"`
	Config       TestAuthConfig   `json:"config"`
}

// DefaultAuthFixtures returns standard authentication fixtures for testing.

func DefaultAuthFixtures() *AuthFixtures {
	return &AuthFixtures{
		Users: []TestUser{
			{
				Username: "admin",
				Password: "admin123",
				Email:    "admin@nephoran.local",
				Roles:    []string{"cluster-admin", "network-admin"},
				Claims: map[string]interface{}{
					"department": "system",
					"clearance":  "top-secret",
				},
				Enabled: true,
			},
			{
				Username: "operator",
				Password: "operator123",
				Email:    "operator@nephoran.local",
				Roles:    []string{"network-operator"},
				Claims: map[string]interface{}{
					"department": "operations",
					"clearance":  "secret",
				},
				Enabled: true,
			},
			{
				Username: "viewer",
				Password: "viewer123",
				Email:    "viewer@nephoran.local",
				Roles:    []string{"network-viewer"},
				Claims: map[string]interface{}{
					"department": "monitoring",
					"clearance":  "confidential",
				},
				Enabled: true,
			},
			{
				Username: "service-account",
				Password: "service123",
				Email:    "service@nephoran.local",
				Roles:    []string{"api-client"},
				Claims: map[string]interface{}{
					"type":    "service",
					"service": "intent-operator",
				},
				Enabled: true,
			},
		},
		Roles: []TestRole{
			{
				ID:   "cluster-admin",
				Name: "cluster-admin",
				Permissions: []string{
					"*:*:*",
				},
				Description: "Full cluster administration access",
				CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
				UpdatedAt:   time.Now(),
			},
			{
				ID:   "network-admin",
				Name: "network-admin",
				Permissions: []string{
					"networkintents:*:*",
					"e2nodesets:*:*",
					"networkfunctions:*:*",
				},
				Description: "Network function administration",
				CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
				UpdatedAt:   time.Now(),
			},
			{
				ID:   "network-operator",
				Name: "network-operator",
				Permissions: []string{
					"networkintents:read:*",
					"networkintents:update:*",
					"e2nodesets:read:*",
					"networkfunctions:read:*",
				},
				Description: "Network function operations",
				CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
				UpdatedAt:   time.Now(),
			},
			{
				ID:   "network-viewer",
				Name: "network-viewer",
				Permissions: []string{
					"networkintents:read:*",
					"e2nodesets:read:*",
					"networkfunctions:read:*",
				},
				Description: "Read-only network access",
				CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
				UpdatedAt:   time.Now(),
			},
			{
				ID:   "api-client",
				Name: "api-client",
				Permissions: []string{
					"api:read:*",
					"api:write:*",
				},
				Description: "API client access for services",
				CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
				UpdatedAt:   time.Now(),
			},
		},
		Config: TestAuthConfig{
			JWTSecret:     "nephoran-test-secret-2025",
			TokenDuration: 24 * time.Hour,
			Issuer:        "nephoran-intent-operator",
			Audience:      []string{"nephoran-api", "nephoran-ui"},
			Claims: map[string]interface{}{
				"version": "1.0",
				"env":     "test",
			},
		},
	}
}

// GetUserByUsername returns a test user by username.

func (af *AuthFixtures) GetUserByUsername(username string) (*TestUser, bool) {
	for _, user := range af.Users {
		if user.Username == username {
			return &user, true
		}
	}
	return nil, false
}

// GetRoleByName returns a test role by name.

func (af *AuthFixtures) GetRoleByName(roleName string) (*TestRole, bool) {
	for _, role := range af.Roles {
		if role.Name == roleName {
			return &role, true
		}
	}
	return nil, false
}

// ValidateUserCredentials validates user credentials against fixtures.

func (af *AuthFixtures) ValidateUserCredentials(username, password string) bool {
	user, exists := af.GetUserByUsername(username)
	if !exists || !user.Enabled {
		return false
	}
	return user.Password == password
}

// GenerateJWTToken generates a JWT token for a test user.

func (af *AuthFixtures) GenerateJWTToken(username string) (string, error) {
	user, exists := af.GetUserByUsername(username)
	if !exists {
		return "", fmt.Errorf("user not found: %s", username)
	}

	// Create claims
	claims := jwt.MapClaims{
		"sub":      user.Username,
		"email":    user.Email,
		"roles":    user.Roles,
		"iss":      af.Config.Issuer,
		"aud":      af.Config.Audience,
		"exp":      time.Now().Add(af.Config.TokenDuration).Unix(),
		"iat":      time.Now().Unix(),
		"nbf":      time.Now().Unix(),
	}

	// Add custom claims
	for key, value := range user.Claims {
		claims[key] = value
	}

	// Add config claims
	for key, value := range af.Config.Claims {
		claims[key] = value
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := token.SignedString([]byte(af.Config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}

	return tokenString, nil
}

// ValidateJWTToken validates a JWT token against test configuration.

func (af *AuthFixtures) ValidateJWTToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(af.Config.JWTSecret), nil
	})
}

// GenerateTestCertificates generates test certificates for mTLS scenarios.

func (af *AuthFixtures) GenerateTestCertificates() error {
	// Generate CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Nephoran Test CA", Organization: []string{"Nephoran Test"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate CA key: %v", err)
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %v", err)
	}

	// Encode CA certificate
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	// Generate client certificates
	clientCerts := []struct {
		name       string
		commonName string
	}{
		{"admin-client", "admin@nephoran.local"},
		{"operator-client", "operator@nephoran.local"},
		{"service-client", "service@nephoran.local"},
	}

	for _, certInfo := range clientCerts {
		// Generate client private key
		clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("failed to generate client key for %s: %v", certInfo.name, err)
		}

		// Create client certificate template
		clientTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(int64(len(af.Certificates)) + 2),
			Subject:      pkix.Name{CommonName: certInfo.commonName, Organization: []string{"Nephoran Test"}},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().Add(365 * 24 * time.Hour),
			KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		// Create client certificate
		clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
		if err != nil {
			return fmt.Errorf("failed to create client certificate for %s: %v", certInfo.name, err)
		}

		// Encode client certificate
		clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})

		// Encode client private key
		clientKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)})

		// Add to fixtures
		af.Certificates = append(af.Certificates, TestCertificate{
			Name:        certInfo.name,
			CommonName:  certInfo.commonName,
			Certificate: clientCertPEM,
			PrivateKey:  clientKeyPEM,
			CA:          caCertPEM,
			ExpiresAt:   time.Now().Add(365 * 24 * time.Hour),
		})
	}

	return nil
}

// GetCertificateByName returns a test certificate by name.

func (af *AuthFixtures) GetCertificateByName(name string) (*TestCertificate, bool) {
	for _, cert := range af.Certificates {
		if cert.Name == name {
			return &cert, true
		}
	}
	return nil, false
}

// HasPermission checks if a user has a specific permission.

func (af *AuthFixtures) HasPermission(username, permission string) bool {
	user, exists := af.GetUserByUsername(username)
	if !exists || !user.Enabled {
		return false
	}

	for _, roleName := range user.Roles {
		role, roleExists := af.GetRoleByName(roleName)
		if !roleExists {
			continue
		}

		for _, perm := range role.Permissions {
			if perm == "*:*:*" || perm == permission {
				return true
			}
			// Check wildcard permissions
			if af.matchesWildcard(perm, permission) {
				return true
			}
		}
	}

	return false
}

// matchesWildcard checks if a permission matches a wildcard pattern.

func (af *AuthFixtures) matchesWildcard(pattern, permission string) bool {
	// Simple wildcard matching for resource:action:namespace format
	// This is a simplified implementation for testing
	patternParts := af.splitPermission(pattern)
	permParts := af.splitPermission(permission)

	if len(patternParts) != len(permParts) {
		return false
	}

	for i, patternPart := range patternParts {
		if patternPart != "*" && patternPart != permParts[i] {
			return false
		}
	}

	return true
}

// splitPermission splits a permission string into parts.

func (af *AuthFixtures) splitPermission(permission string) []string {
	// Split by colon for resource:action:namespace format
	parts := make([]string, 0)
	current := ""
	for _, char := range permission {
		if char == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// ToJSON converts the fixtures to JSON format.

func (af *AuthFixtures) ToJSON() ([]byte, error) {
	return json.MarshalIndent(af, "", "  ")
}

// FromJSON loads fixtures from JSON data.

func (af *AuthFixtures) FromJSON(data []byte) error {
	return json.Unmarshal(data, af)
}

// CreateOAuth2Fixtures creates OAuth2-specific test fixtures.

func CreateOAuth2Fixtures() *OAuth2Fixtures {
	return &OAuth2Fixtures{
		Clients: []OAuth2Client{
			{
				ClientID:     "nephoran-ui",
				ClientSecret: "ui-secret-2025",
				Name:         "Nephoran Web UI",
				RedirectURIs: []string{
					"http://localhost:3000/auth/callback",
					"http://nephoran-ui:3000/auth/callback",
				},
				Scopes: []string{"openid", "profile", "email", "network:read", "network:write"},
				GrantTypes: []string{"authorization_code", "refresh_token"},
			},
			{
				ClientID:     "nephoran-api",
				ClientSecret: "api-secret-2025",
				Name:         "Nephoran API Client",
				RedirectURIs: []string{},
				Scopes: []string{"network:read", "network:write", "system:admin"},
				GrantTypes: []string{"client_credentials"},
			},
			{
				ClientID:     "e2-simulator",
				ClientSecret: "e2-secret-2025",
				Name:         "E2 Node Simulator",
				RedirectURIs: []string{},
				Scopes: []string{"e2:register", "e2:report"},
				GrantTypes: []string{"client_credentials"},
			},
		},
		AuthorizationCodes: make(map[string]OAuth2AuthCode),
		AccessTokens: make(map[string]OAuth2AccessToken),
		RefreshTokens: make(map[string]OAuth2RefreshToken),
	}
}

// OAuth2Fixtures contains OAuth2-specific test fixtures.

type OAuth2Fixtures struct {
	Clients            []OAuth2Client                      `json:"clients"`
	AuthorizationCodes map[string]OAuth2AuthCode           `json:"authorization_codes"`
	AccessTokens       map[string]OAuth2AccessToken        `json:"access_tokens"`
	RefreshTokens      map[string]OAuth2RefreshToken       `json:"refresh_tokens"`
}

// OAuth2Client represents an OAuth2 client.

type OAuth2Client struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Name         string   `json:"name"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
	GrantTypes   []string `json:"grant_types"`
}

// OAuth2AuthCode represents an OAuth2 authorization code.

type OAuth2AuthCode struct {
	Code        string    `json:"code"`
	ClientID    string    `json:"client_id"`
	UserID      string    `json:"user_id"`
	RedirectURI string    `json:"redirect_uri"`
	Scopes      []string  `json:"scopes"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// OAuth2AccessToken represents an OAuth2 access token.

type OAuth2AccessToken struct {
	Token     string    `json:"token"`
	ClientID  string    `json:"client_id"`
	UserID    string    `json:"user_id"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
}

// OAuth2RefreshToken represents an OAuth2 refresh token.

type OAuth2RefreshToken struct {
	Token     string    `json:"token"`
	ClientID  string    `json:"client_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetClientByID returns an OAuth2 client by ID.

func (of *OAuth2Fixtures) GetClientByID(clientID string) (*OAuth2Client, bool) {
	for _, client := range of.Clients {
		if client.ClientID == clientID {
			return &client, true
		}
	}
	return nil, false
}

// ValidateClientCredentials validates OAuth2 client credentials.

func (of *OAuth2Fixtures) ValidateClientCredentials(clientID, clientSecret string) bool {
	client, exists := of.GetClientByID(clientID)
	if !exists {
		return false
	}
	return client.ClientSecret == clientSecret
}

// SupportsGrantType checks if a client supports a specific grant type.

func (of *OAuth2Fixtures) SupportsGrantType(clientID, grantType string) bool {
	client, exists := of.GetClientByID(clientID)
	if !exists {
		return false
	}

	for _, supportedType := range client.GrantTypes {
		if supportedType == grantType {
			return true
		}
	}
	return false
}

// ValidateRedirectURI validates a redirect URI for a client.

func (of *OAuth2Fixtures) ValidateRedirectURI(clientID, redirectURI string) bool {
	client, exists := of.GetClientByID(clientID)
	if !exists {
		return false
	}

	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			return true
		}
	}
	return false
}

// CreateSAMLFixtures creates SAML-specific test fixtures.

func CreateSAMLFixtures() *SAMLFixtures {
	return &SAMLFixtures{
		IdentityProviders: []SAMLIdentityProvider{
			{
				EntityID:    "https://sso.nephoran.local/saml/idp",
				Name:        "Nephoran SSO",
				SSOService:  "https://sso.nephoran.local/saml/sso",
				SLOService:  "https://sso.nephoran.local/saml/slo",
				Certificate: []byte("-----BEGIN CERTIFICATE-----\nMIIC...example...cert\n-----END CERTIFICATE-----"),
			},
		},
		ServiceProviders: []SAMLServiceProvider{
			{
				EntityID:    "https://api.nephoran.local/saml/sp",
				Name:        "Nephoran API",
				ACSService:  "https://api.nephoran.local/saml/acs",
				SLOService:  "https://api.nephoran.local/saml/slo",
				Certificate: []byte("-----BEGIN CERTIFICATE-----\nMIIC...example...cert\n-----END CERTIFICATE-----"),
				PrivateKey:  []byte("-----BEGIN PRIVATE KEY-----\nMIIE...example...key\n-----END PRIVATE KEY-----"),
			},
		},
	}
}

// SAMLFixtures contains SAML-specific test fixtures.

type SAMLFixtures struct {
	IdentityProviders []SAMLIdentityProvider `json:"identity_providers"`
	ServiceProviders  []SAMLServiceProvider  `json:"service_providers"`
}

// SAMLIdentityProvider represents a SAML identity provider.

type SAMLIdentityProvider struct {
	EntityID    string `json:"entity_id"`
	Name        string `json:"name"`
	SSOService  string `json:"sso_service"`
	SLOService  string `json:"slo_service"`
	Certificate []byte `json:"certificate"`
}

// SAMLServiceProvider represents a SAML service provider.

type SAMLServiceProvider struct {
	EntityID    string `json:"entity_id"`
	Name        string `json:"name"`
	ACSService  string `json:"acs_service"`
	SLOService  string `json:"slo_service"`
	Certificate []byte `json:"certificate"`
	PrivateKey  []byte `json:"private_key"`
}

// GetIDPByEntityID returns a SAML identity provider by entity ID.

func (sf *SAMLFixtures) GetIDPByEntityID(entityID string) (*SAMLIdentityProvider, bool) {
	for _, idp := range sf.IdentityProviders {
		if idp.EntityID == entityID {
			return &idp, true
		}
	}
	return nil, false
}

// GetSPByEntityID returns a SAML service provider by entity ID.

func (sf *SAMLFixtures) GetSPByEntityID(entityID string) (*SAMLServiceProvider, bool) {
	for _, sp := range sf.ServiceProviders {
		if sp.EntityID == entityID {
			return &sp, true
		}
	}
	return nil, false
}

// CreateLDAPFixtures creates LDAP-specific test fixtures.

func CreateLDAPFixtures() *LDAPFixtures {
	return &LDAPFixtures{
		Server: LDAPServerConfig{
			Host:     "ldap.nephoran.local",
			Port:     389,
			BaseDN:   "dc=nephoran,dc=local",
			BindDN:   "cn=admin,dc=nephoran,dc=local",
			BindPass: "admin123",
			UserDN:   "ou=users,dc=nephoran,dc=local",
			GroupDN:  "ou=groups,dc=nephoran,dc=local",
		},
		Users: []LDAPUser{
			{
				DN:           "cn=admin,ou=users,dc=nephoran,dc=local",
				CN:           "admin",
				UID:          "admin",
				Email:        "admin@nephoran.local",
				DisplayName:  "Administrator",
				MemberOf:     []string{"cn=admins,ou=groups,dc=nephoran,dc=local"},
				UserPassword: "admin123",
			},
			{
				DN:           "cn=operator,ou=users,dc=nephoran,dc=local",
				CN:           "operator",
				UID:          "operator",
				Email:        "operator@nephoran.local",
				DisplayName:  "Network Operator",
				MemberOf:     []string{"cn=operators,ou=groups,dc=nephoran,dc=local"},
				UserPassword: "operator123",
			},
		},
		Groups: []LDAPGroup{
			{
				DN:          "cn=admins,ou=groups,dc=nephoran,dc=local",
				CN:          "admins",
				Description: "System Administrators",
				Members:     []string{"cn=admin,ou=users,dc=nephoran,dc=local"},
			},
			{
				DN:          "cn=operators,ou=groups,dc=nephoran,dc=local",
				CN:          "operators",
				Description: "Network Operators",
				Members:     []string{"cn=operator,ou=users,dc=nephoran,dc=local"},
			},
		},
	}
}

// LDAPFixtures contains LDAP-specific test fixtures.

type LDAPFixtures struct {
	Server LDAPServerConfig `json:"server"`
	Users  []LDAPUser       `json:"users"`
	Groups []LDAPGroup      `json:"groups"`
}

// LDAPServerConfig represents LDAP server configuration.

type LDAPServerConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	BaseDN   string `json:"base_dn"`
	BindDN   string `json:"bind_dn"`
	BindPass string `json:"bind_pass"`
	UserDN   string `json:"user_dn"`
	GroupDN  string `json:"group_dn"`
}

// LDAPUser represents an LDAP user entry.

type LDAPUser struct {
	DN           string   `json:"dn"`
	CN           string   `json:"cn"`
	UID          string   `json:"uid"`
	Email        string   `json:"email"`
	DisplayName  string   `json:"display_name"`
	MemberOf     []string `json:"member_of"`
	UserPassword string   `json:"user_password"`
}

// LDAPGroup represents an LDAP group entry.

type LDAPGroup struct {
	DN          string   `json:"dn"`
	CN          string   `json:"cn"`
	Description string   `json:"description"`
	Members     []string `json:"members"`
}

// GetLDAPUserByUID returns an LDAP user by UID.

func (lf *LDAPFixtures) GetLDAPUserByUID(uid string) (*LDAPUser, bool) {
	for _, user := range lf.Users {
		if user.UID == uid {
			return &user, true
		}
	}
	return nil, false
}

// ValidateLDAPCredentials validates LDAP user credentials.

func (lf *LDAPFixtures) ValidateLDAPCredentials(uid, password string) bool {
	user, exists := lf.GetLDAPUserByUID(uid)
	if !exists {
		return false
	}
	return user.UserPassword == password
}

// GetLDAPGroupByCN returns an LDAP group by CN.

func (lf *LDAPFixtures) GetLDAPGroupByCN(cn string) (*LDAPGroup, bool) {
	for _, group := range lf.Groups {
		if group.CN == cn {
			return &group, true
		}
	}
	return nil, false
}

// IsUserMemberOfGroup checks if a user is a member of a specific group.

func (lf *LDAPFixtures) IsUserMemberOfGroup(userDN, groupDN string) bool {
	group, exists := lf.GetLDAPGroupByCN(groupDN)
	if !exists {
		return false
	}

	for _, member := range group.Members {
		if member == userDN {
			return true
		}
	}
	return false
}