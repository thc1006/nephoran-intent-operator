// Package security implements comprehensive security features for the A1 Policy Management Service
// compliant with O-RAN WG11 security specifications and industry best practices
package security

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// AuthProvider defines the authentication provider interface
type AuthProvider interface {
	Authenticate(ctx context.Context, credentials interface{}) (*AuthResult, error)
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error)
	RevokeToken(ctx context.Context, token string) error
	GetPublicKeys(ctx context.Context) (jwk.Set, error)
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled                bool               `json:"enabled"`
	Type                   AuthType           `json:"type"`
	JWTConfig              *JWTConfig         `json:"jwt_config,omitempty"`
	OAuth2Config           *OAuth2Config      `json:"oauth2_config,omitempty"`
	ServiceAuthConfig      *ServiceAuthConfig `json:"service_auth_config,omitempty"`
	RBACConfig             *RBACConfig        `json:"rbac_config,omitempty"`
	TokenExpiry            time.Duration      `json:"token_expiry"`
	RefreshTokenExpiry     time.Duration      `json:"refresh_token_expiry"`
	MaxTokenLifetime       time.Duration      `json:"max_token_lifetime"`
	EnableRevocation       bool               `json:"enable_revocation"`
	RequireSecureTransport bool               `json:"require_secure_transport"`
}

// AuthType represents the authentication type
type AuthType string

const (
	AuthTypeJWT            AuthType = "jwt"
	AuthTypeOAuth2         AuthType = "oauth2"
	AuthTypeServiceAccount AuthType = "service_account"
	AuthTypeMutualTLS      AuthType = "mtls"
	AuthTypeAPIKey         AuthType = "api_key"
)

// JWTConfig holds JWT-specific configuration
type JWTConfig struct {
	Issuers           []string          `json:"issuers"`
	Audiences         []string          `json:"audiences"`
	SigningAlgorithm  string            `json:"signing_algorithm"`
	PublicKeyPath     string            `json:"public_key_path"`
	PrivateKeyPath    string            `json:"private_key_path"`
	JWKSEndpoint      string            `json:"jwks_endpoint"`
	JWKSCacheDuration time.Duration     `json:"jwks_cache_duration"`
	RequiredClaims    map[string]string `json:"required_claims"`
	ClockSkew         time.Duration     `json:"clock_skew"`
}

// OAuth2Config holds OAuth2/OIDC configuration
type OAuth2Config struct {
	Providers       map[string]*OAuth2Provider `json:"providers"`
	DefaultProvider string                     `json:"default_provider"`
	StateStore      StateStore                 `json:"-"`
	PKCERequired    bool                       `json:"pkce_required"`
	NonceRequired   bool                       `json:"nonce_required"`
}

// OAuth2Provider represents an OAuth2/OIDC provider
type OAuth2Provider struct {
	Name             string            `json:"name"`
	ClientID         string            `json:"client_id"`
	ClientSecret     string            `json:"client_secret"`
	AuthURL          string            `json:"auth_url"`
	TokenURL         string            `json:"token_url"`
	UserInfoURL      string            `json:"user_info_url"`
	JWKSEndpoint     string            `json:"jwks_endpoint"`
	Scopes           []string          `json:"scopes"`
	RedirectURLs     []string          `json:"redirect_urls"`
	ResponseType     string            `json:"response_type"`
	AdditionalParams map[string]string `json:"additional_params"`
}

// ServiceAuthConfig holds service account authentication configuration
type ServiceAuthConfig struct {
	Enabled          bool                       `json:"enabled"`
	ServiceAccounts  map[string]*ServiceAccount `json:"service_accounts"`
	TokenRotation    bool                       `json:"token_rotation"`
	RotationInterval time.Duration              `json:"rotation_interval"`
	MaxKeyAge        time.Duration              `json:"max_key_age"`
	AllowedServices  []string                   `json:"allowed_services"`
}

// ServiceAccount represents a service account
type ServiceAccount struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	PublicKey     string            `json:"public_key"`
	AllowedScopes []string          `json:"allowed_scopes"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
	LastRotatedAt time.Time         `json:"last_rotated_at"`
	Active        bool              `json:"active"`
}

// RBACConfig holds RBAC configuration
type RBACConfig struct {
	Enabled          bool             `json:"enabled"`
	PolicyEngine     string           `json:"policy_engine"`
	PolicyPath       string           `json:"policy_path"`
	Roles            map[string]*Role `json:"roles"`
	DefaultRole      string           `json:"default_role"`
	DenyByDefault    bool             `json:"deny_by_default"`
	CachePermissions bool             `json:"cache_permissions"`
	CacheTTL         time.Duration    `json:"cache_ttl"`
}

// Role represents an RBAC role
type Role struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []*Permission     `json:"permissions"`
	ParentRoles []string          `json:"parent_roles"`
	Metadata    map[string]string `json:"metadata"`
	Priority    int               `json:"priority"`
}

// Permission represents a permission
type Permission struct {
	Resource   string            `json:"resource"`
	Actions    []string          `json:"actions"`
	Conditions map[string]string `json:"conditions"`
	Effect     PermissionEffect  `json:"effect"`
}

// PermissionEffect represents the effect of a permission
type PermissionEffect string

const (
	PermissionAllow PermissionEffect = "allow"
	PermissionDeny  PermissionEffect = "deny"
)

// AuthResult represents the result of authentication
type AuthResult struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token,omitempty"`
	TokenType    string                 `json:"token_type"`
	ExpiresIn    int64                  `json:"expires_in"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Scope        string                 `json:"scope,omitempty"`
	User         *User                  `json:"user,omitempty"`
	Claims       map[string]interface{} `json:"claims,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
}

// User represents an authenticated user
type User struct {
	ID          string            `json:"id"`
	Username    string            `json:"username"`
	Email       string            `json:"email,omitempty"`
	Roles       []string          `json:"roles"`
	Permissions []*Permission     `json:"permissions,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	IsService   bool              `json:"is_service"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID      string            `json:"uid,omitempty"`
	Username    string            `json:"username,omitempty"`
	Email       string            `json:"email,omitempty"`
	Roles       []string          `json:"roles,omitempty"`
	Scope       string            `json:"scope,omitempty"`
	ServiceID   string            `json:"sid,omitempty"`
	Permissions []string          `json:"permissions,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// StateStore interface for OAuth2 state management
type StateStore interface {
	Store(ctx context.Context, state string, data interface{}) error
	Retrieve(ctx context.Context, state string) (interface{}, error)
	Delete(ctx context.Context, state string) error
}

// AuthManager manages authentication and authorization
type AuthManager struct {
	config          *AuthConfig
	logger          *logging.StructuredLogger
	providers       map[AuthType]AuthProvider
	tokenStore      TokenStore
	keyCache        *KeyCache
	rbacEngine      RBACEngine
	mu              sync.RWMutex
	publicKey       *rsa.PublicKey
	privateKey      *rsa.PrivateKey
	jwksCache       jwk.Set
	jwksCacheExpiry time.Time
	revokedTokens   map[string]time.Time
	revokedTokensMu sync.RWMutex
}

// TokenStore interface for token storage
type TokenStore interface {
	Store(ctx context.Context, token string, claims *TokenClaims, expiry time.Duration) error
	Get(ctx context.Context, token string) (*TokenClaims, error)
	Delete(ctx context.Context, token string) error
	Exists(ctx context.Context, token string) (bool, error)
}

// KeyCache manages cryptographic key caching
type KeyCache struct {
	mu     sync.RWMutex
	keys   map[string]interface{}
	expiry map[string]time.Time
	ttl    time.Duration
}

// RBACEngine interface for RBAC operations
type RBACEngine interface {
	Authorize(ctx context.Context, user *User, resource string, action string) (bool, error)
	GetUserPermissions(ctx context.Context, user *User) ([]*Permission, error)
	GetRolePermissions(ctx context.Context, role string) ([]*Permission, error)
	AddRole(ctx context.Context, role *Role) error
	RemoveRole(ctx context.Context, roleName string) error
	AssignRole(ctx context.Context, userID string, roleName string) error
	RevokeRole(ctx context.Context, userID string, roleName string) error
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *AuthConfig, logger *logging.StructuredLogger) (*AuthManager, error) {
	if config == nil {
		return nil, errors.New("auth config is required")
	}

	am := &AuthManager{
		config:        config,
		logger:        logger,
		providers:     make(map[AuthType]AuthProvider),
		revokedTokens: make(map[string]time.Time),
		keyCache:      NewKeyCache(30 * time.Minute),
	}

	// Initialize providers based on configuration
	if err := am.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize auth providers: %w", err)
	}

	// Load cryptographic keys if JWT is enabled
	if config.Type == AuthTypeJWT && config.JWTConfig != nil {
		if err := am.loadKeys(); err != nil {
			return nil, fmt.Errorf("failed to load cryptographic keys: %w", err)
		}
	}

	// Initialize RBAC engine if enabled
	if config.RBACConfig != nil && config.RBACConfig.Enabled {
		rbacEngine, err := NewRBACEngine(config.RBACConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize RBAC engine: %w", err)
		}
		am.rbacEngine = rbacEngine
	}

	// Start token cleanup routine
	go am.cleanupExpiredTokens()

	return am, nil
}

// initializeProviders initializes authentication providers
func (am *AuthManager) initializeProviders() error {
	switch am.config.Type {
	case AuthTypeJWT:
		provider, err := NewJWTProvider(am.config.JWTConfig, am.logger)
		if err != nil {
			return err
		}
		am.providers[AuthTypeJWT] = provider

	case AuthTypeOAuth2:
		provider, err := NewOAuth2Provider(am.config.OAuth2Config, am.logger)
		if err != nil {
			return err
		}
		am.providers[AuthTypeOAuth2] = provider

	case AuthTypeServiceAccount:
		provider, err := NewServiceAccountProvider(am.config.ServiceAuthConfig, am.logger)
		if err != nil {
			return err
		}
		am.providers[AuthTypeServiceAccount] = provider

	default:
		return fmt.Errorf("unsupported auth type: %s", am.config.Type)
	}

	return nil
}

// loadKeys loads cryptographic keys for JWT signing/verification
func (am *AuthManager) loadKeys() error {
	if am.config.JWTConfig == nil {
		return errors.New("JWT config is nil")
	}

	// Load public key
	if am.config.JWTConfig.PublicKeyPath != "" {
		pubKeyData, err := readFile(am.config.JWTConfig.PublicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read public key: %w", err)
		}

		pubKey, err := parsePublicKey(pubKeyData)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
		am.publicKey = pubKey
	}

	// Load private key
	if am.config.JWTConfig.PrivateKeyPath != "" {
		privKeyData, err := readFile(am.config.JWTConfig.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key: %w", err)
		}

		privKey, err := parsePrivateKey(privKeyData)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		am.privateKey = privKey
	}

	// Load JWKS if endpoint is configured
	if am.config.JWTConfig.JWKSEndpoint != "" {
		if err := am.refreshJWKS(); err != nil {
			return fmt.Errorf("failed to load JWKS: %w", err)
		}
	}

	return nil
}

// Authenticate performs authentication based on configured provider
func (am *AuthManager) Authenticate(ctx context.Context, r *http.Request) (*AuthResult, error) {
	// Check for secure transport if required
	if am.config.RequireSecureTransport && !isSecureTransport(r) {
		return nil, errors.New("secure transport required")
	}

	// Extract credentials from request
	credentials, authType, err := am.extractCredentials(r)
	if err != nil {
		return nil, fmt.Errorf("failed to extract credentials: %w", err)
	}

	// Get appropriate provider
	provider, ok := am.providers[authType]
	if !ok {
		return nil, fmt.Errorf("no provider for auth type: %s", authType)
	}

	// Perform authentication
	result, err := provider.Authenticate(ctx, credentials)
	if err != nil {
		am.logger.Error("authentication failed",
			slog.String("auth_type", string(authType)),
			slog.Error(err))
		return nil, err
	}

	// Apply RBAC if enabled
	if am.rbacEngine != nil && result.User != nil {
		permissions, err := am.rbacEngine.GetUserPermissions(ctx, result.User)
		if err != nil {
			am.logger.Warn("failed to get user permissions",
				slog.String("user_id", result.User.ID),
				slog.Error(err))
		} else {
			result.User.Permissions = permissions
		}
	}

	// Store token if caching is enabled
	if am.tokenStore != nil {
		claims := &TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   result.User.ID,
				ExpiresAt: jwt.NewNumericDate(result.ExpiresAt),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			UserID:   result.User.ID,
			Username: result.User.Username,
			Email:    result.User.Email,
			Roles:    result.User.Roles,
			Scope:    result.Scope,
		}

		if err := am.tokenStore.Store(ctx, result.AccessToken, claims, am.config.TokenExpiry); err != nil {
			am.logger.Warn("failed to cache token", slog.Error(err))
		}
	}

	am.logger.Info("authentication successful",
		slog.String("user_id", result.User.ID),
		slog.String("auth_type", string(authType)))

	return result, nil
}

// ValidateToken validates an access token
func (am *AuthManager) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
	// Check if token is revoked
	if am.isTokenRevoked(token) {
		return nil, errors.New("token has been revoked")
	}

	// Check token store first if available
	if am.tokenStore != nil {
		if claims, err := am.tokenStore.Get(ctx, token); err == nil {
			return claims, nil
		}
	}

	// Validate with appropriate provider
	provider, ok := am.providers[am.config.Type]
	if !ok {
		return nil, fmt.Errorf("no provider for auth type: %s", am.config.Type)
	}

	claims, err := provider.ValidateToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Check token expiry
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	return claims, nil
}

// Authorize checks if a user has permission for a resource and action
func (am *AuthManager) Authorize(ctx context.Context, user *User, resource, action string) (bool, error) {
	if am.rbacEngine == nil {
		// If RBAC is not configured, allow all authenticated users
		return user != nil, nil
	}

	return am.rbacEngine.Authorize(ctx, user, resource, action)
}

// RefreshToken refreshes an access token using a refresh token
func (am *AuthManager) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	provider, ok := am.providers[am.config.Type]
	if !ok {
		return nil, fmt.Errorf("no provider for auth type: %s", am.config.Type)
	}

	result, err := provider.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// Check max token lifetime
	if am.config.MaxTokenLifetime > 0 {
		maxExpiry := time.Now().Add(am.config.MaxTokenLifetime)
		if result.ExpiresAt.After(maxExpiry) {
			result.ExpiresAt = maxExpiry
			result.ExpiresIn = int64(am.config.MaxTokenLifetime.Seconds())
		}
	}

	return result, nil
}

// RevokeToken revokes an access token
func (am *AuthManager) RevokeToken(ctx context.Context, token string) error {
	if !am.config.EnableRevocation {
		return errors.New("token revocation is not enabled")
	}

	// Add to revoked tokens list
	am.revokedTokensMu.Lock()
	am.revokedTokens[token] = time.Now().Add(am.config.TokenExpiry)
	am.revokedTokensMu.Unlock()

	// Remove from token store if present
	if am.tokenStore != nil {
		if err := am.tokenStore.Delete(ctx, token); err != nil {
			am.logger.Warn("failed to remove token from store", slog.Error(err))
		}
	}

	// Call provider's revoke method if available
	provider, ok := am.providers[am.config.Type]
	if ok {
		if err := provider.RevokeToken(ctx, token); err != nil {
			am.logger.Warn("provider token revocation failed", slog.Error(err))
		}
	}

	am.logger.Info("token revoked successfully", slog.String("token_id", hashToken(token)))
	return nil
}

// extractCredentials extracts authentication credentials from request
func (am *AuthManager) extractCredentials(r *http.Request) (interface{}, AuthType, error) {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			return nil, "", errors.New("invalid authorization header format")
		}

		scheme := strings.ToLower(parts[0])
		credentials := parts[1]

		switch scheme {
		case "bearer":
			return credentials, am.config.Type, nil
		case "basic":
			return credentials, AuthTypeAPIKey, nil
		default:
			return nil, "", fmt.Errorf("unsupported authorization scheme: %s", scheme)
		}
	}

	// Check for API key in header or query
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey, AuthTypeAPIKey, nil
	}

	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
		return apiKey, AuthTypeAPIKey, nil
	}

	// Check for client certificate (mTLS)
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		return r.TLS.PeerCertificates[0], AuthTypeMutualTLS, nil
	}

	return nil, "", errors.New("no authentication credentials found")
}

// isTokenRevoked checks if a token has been revoked
func (am *AuthManager) isTokenRevoked(token string) bool {
	am.revokedTokensMu.RLock()
	expiry, revoked := am.revokedTokens[token]
	am.revokedTokensMu.RUnlock()

	if !revoked {
		return false
	}

	// Check if revocation has expired
	if time.Now().After(expiry) {
		am.revokedTokensMu.Lock()
		delete(am.revokedTokens, token)
		am.revokedTokensMu.Unlock()
		return false
	}

	return true
}

// refreshJWKS refreshes the JWKS cache
func (am *AuthManager) refreshJWKS() error {
	if am.config.JWTConfig.JWKSEndpoint == "" {
		return nil
	}

	// Check if cache is still valid
	if time.Now().Before(am.jwksCacheExpiry) {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	set, err := jwk.Fetch(ctx, am.config.JWTConfig.JWKSEndpoint)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	am.mu.Lock()
	am.jwksCache = set
	am.jwksCacheExpiry = time.Now().Add(am.config.JWTConfig.JWKSCacheDuration)
	am.mu.Unlock()

	return nil
}

// cleanupExpiredTokens periodically removes expired tokens from revocation list
func (am *AuthManager) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		am.revokedTokensMu.Lock()
		now := time.Now()
		for token, expiry := range am.revokedTokens {
			if now.After(expiry) {
				delete(am.revokedTokens, token)
			}
		}
		am.revokedTokensMu.Unlock()
	}
}

// Helper functions

func isSecureTransport(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func hashToken(token string) string {
	// Return first 8 characters for logging (not for security)
	if len(token) > 8 {
		return token[:8] + "..."
	}
	return "***"
}

func readFile(path string) ([]byte, error) {
	// Implementation would read file securely
	return nil, nil
}

func parsePublicKey(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPub, nil
}

func parsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		privInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaPriv, ok := privInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
		return rsaPriv, nil
	}

	return priv, nil
}

// NewKeyCache creates a new key cache
func NewKeyCache(ttl time.Duration) *KeyCache {
	return &KeyCache{
		keys:   make(map[string]interface{}),
		expiry: make(map[string]time.Time),
		ttl:    ttl,
	}
}

// Get retrieves a key from cache
func (kc *KeyCache) Get(keyID string) (interface{}, bool) {
	kc.mu.RLock()
	defer kc.mu.RUnlock()

	if expiry, ok := kc.expiry[keyID]; ok {
		if time.Now().After(expiry) {
			return nil, false
		}
	}

	key, ok := kc.keys[keyID]
	return key, ok
}

// Set stores a key in cache
func (kc *KeyCache) Set(keyID string, key interface{}) {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	kc.keys[keyID] = key
	kc.expiry[keyID] = time.Now().Add(kc.ttl)
}
