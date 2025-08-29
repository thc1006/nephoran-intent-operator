package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nephio-project/nephoran-intent-operator/pkg/auth/providers"
)

// JWTManager manages JWT token creation, validation, and rotation.

type JWTManager struct {

	// Signing keys.

	signingKey *rsa.PrivateKey

	verifyingKey *rsa.PublicKey

	keyID string

	keyRotationPeriod time.Duration

	// Token settings.

	issuer string

	defaultTTL time.Duration

	refreshTTL time.Duration

	// Token storage and blacklist.

	tokenStore TokenStore

	blacklist TokenBlacklist

	// Security settings.

	requireSecureCookies bool

	cookieDomain string

	cookiePath string

	// Monitoring.

	logger *slog.Logger

	metrics *JWTMetrics

	mutex sync.RWMutex
}

// JWTConfig represents JWT configuration.

type JWTConfig struct {
	Issuer string `json:"issuer"`

	SigningKeyPath string `json:"signing_key_path,omitempty"`

	SigningKey string `json:"signing_key,omitempty"`

	KeyRotationPeriod time.Duration `json:"key_rotation_period"`

	DefaultTTL time.Duration `json:"default_ttl"`

	RefreshTTL time.Duration `json:"refresh_ttl"`

	RequireSecureCookies bool `json:"require_secure_cookies"`

	CookieDomain string `json:"cookie_domain"`

	CookiePath string `json:"cookie_path"`

	Algorithm string `json:"algorithm"`
}

// NephoranJWTClaims extends standard JWT claims with Nephoran-specific fields.

type NephoranJWTClaims struct {
	jwt.RegisteredClaims

	// User information.

	Email string `json:"email"`

	EmailVerified bool `json:"email_verified"`

	Name string `json:"name"`

	PreferredName string `json:"preferred_username"`

	Picture string `json:"picture"`

	// Authorization.

	Groups []string `json:"groups"`

	Roles []string `json:"roles"`

	Permissions []string `json:"permissions"`

	Organizations []string `json:"organizations"`

	// Provider information.

	Provider string `json:"provider"`

	ProviderID string `json:"provider_id"`

	// Nephoran-specific.

	TenantID string `json:"tenant_id,omitempty"`

	SessionID string `json:"session_id"`

	TokenType string `json:"token_type"` // "access" or "refresh"

	Scope string `json:"scope,omitempty"`

	// Security.

	IPAddress string `json:"ip_address,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	// Custom attributes.

	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// TokenStore interface for storing and retrieving tokens.

type TokenStore interface {

	// Store a token with expiration.

	StoreToken(ctx context.Context, tokenID string, token *TokenInfo) error

	// Get token info.

	GetToken(ctx context.Context, tokenID string) (*TokenInfo, error)

	// Update token info.

	UpdateToken(ctx context.Context, tokenID string, token *TokenInfo) error

	// Delete token.

	DeleteToken(ctx context.Context, tokenID string) error

	// List tokens for a user.

	ListUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error)

	// Cleanup expired tokens.

	CleanupExpired(ctx context.Context) error
}

// TokenBlacklist interface for managing revoked tokens.

type TokenBlacklist interface {

	// Add token to blacklist.

	BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error

	// Check if token is blacklisted.

	IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error)

	// Remove expired entries.

	CleanupExpired(ctx context.Context) error
}

// TokenInfo represents stored token information.

type TokenInfo struct {
	TokenID string `json:"token_id"`

	UserID string `json:"user_id"`

	SessionID string `json:"session_id"`

	TokenType string `json:"token_type"` // "access" or "refresh"

	IssuedAt time.Time `json:"issued_at"`

	ExpiresAt time.Time `json:"expires_at"`

	Provider string `json:"provider"`

	Scope string `json:"scope,omitempty"`

	IPAddress string `json:"ip_address,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`

	LastUsed time.Time `json:"last_used"`

	UseCount int64 `json:"use_count"`
}

// JWTMetrics contains JWT-related metrics.

type JWTMetrics struct {
	TokensIssued int64 `json:"tokens_issued"`

	TokensValidated int64 `json:"tokens_validated"`

	TokensRevoked int64 `json:"tokens_revoked"`

	TokensExpired int64 `json:"tokens_expired"`

	ValidationFailures int64 `json:"validation_failures"`

	KeyRotations int64 `json:"key_rotations"`
}

// NewJWTManager creates a new JWT manager.

func NewJWTManager(config *JWTConfig, tokenStore TokenStore, blacklist TokenBlacklist, logger *slog.Logger) (*JWTManager, error) {

	manager := &JWTManager{

		issuer: config.Issuer,

		defaultTTL: config.DefaultTTL,

		refreshTTL: config.RefreshTTL,

		keyRotationPeriod: config.KeyRotationPeriod,

		requireSecureCookies: config.RequireSecureCookies,

		cookieDomain: config.CookieDomain,

		cookiePath: config.CookiePath,

		tokenStore: tokenStore,

		blacklist: blacklist,

		logger: logger,

		metrics: &JWTMetrics{},
	}

	// Initialize signing key.

	if err := manager.initializeSigningKey(config); err != nil {

		return nil, fmt.Errorf("failed to initialize signing key: %w", err)

	}

	// Start background tasks.

	go manager.keyRotationLoop()

	go manager.cleanupLoop()

	return manager, nil

}

// GenerateAccessToken generates an access token for a user.

func (jm *JWTManager) GenerateAccessToken(ctx context.Context, userInfo *providers.UserInfo, sessionID string, options ...TokenOption) (string, *TokenInfo, error) {

	if userInfo == nil {

		return "", nil, fmt.Errorf("user info cannot be nil")

	}

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	opts := applyTokenOptions(options...)

	now := time.Now()

	tokenID := generateTokenID()

	claims := &NephoranJWTClaims{

		RegisteredClaims: jwt.RegisteredClaims{

			ID: tokenID,

			Subject: userInfo.Subject,

			Audience: jwt.ClaimStrings{jm.issuer},

			ExpiresAt: jwt.NewNumericDate(now.Add(jm.defaultTTL)),

			NotBefore: jwt.NewNumericDate(now),

			IssuedAt: jwt.NewNumericDate(now),

			Issuer: jm.issuer,
		},

		Email: userInfo.Email,

		EmailVerified: userInfo.EmailVerified,

		Name: userInfo.Name,

		PreferredName: userInfo.PreferredName,

		Picture: userInfo.Picture,

		Groups: userInfo.Groups,

		Roles: userInfo.Roles,

		Permissions: userInfo.Permissions,

		Organizations: extractOrganizationNames(userInfo.Organizations),

		Provider: userInfo.Provider,

		ProviderID: userInfo.ProviderID,

		SessionID: sessionID,

		TokenType: "access",

		Scope: opts.Scope,

		IPAddress: opts.IPAddress,

		UserAgent: opts.UserAgent,

		Attributes: userInfo.Attributes,
	}

	// Apply custom TTL if specified.

	if opts.TTL > 0 {

		claims.ExpiresAt = jwt.NewNumericDate(now.Add(opts.TTL))

	}

	// Sign token.

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	token.Header["kid"] = jm.keyID

	tokenString, err := token.SignedString(jm.signingKey)

	if err != nil {

		return "", nil, fmt.Errorf("failed to sign access token: %w", err)

	}

	// Store token info.

	tokenInfo := &TokenInfo{

		TokenID: tokenID,

		UserID: userInfo.Subject,

		SessionID: sessionID,

		TokenType: "access",

		IssuedAt: now,

		ExpiresAt: claims.ExpiresAt.Time,

		Provider: userInfo.Provider,

		Scope: opts.Scope,

		IPAddress: opts.IPAddress,

		UserAgent: opts.UserAgent,

		Attributes: userInfo.Attributes,

		LastUsed: now,

		UseCount: 0,
	}

	if err := jm.tokenStore.StoreToken(ctx, tokenID, tokenInfo); err != nil {

		jm.logger.Warn("Failed to store token info", "error", err)

	}

	jm.metrics.TokensIssued++

	return tokenString, tokenInfo, nil

}

// GenerateRefreshToken generates a refresh token.

func (jm *JWTManager) GenerateRefreshToken(ctx context.Context, userInfo *providers.UserInfo, sessionID string, options ...TokenOption) (string, *TokenInfo, error) {

	if userInfo == nil {

		return "", nil, fmt.Errorf("user info cannot be nil")

	}

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	opts := applyTokenOptions(options...)

	now := time.Now()

	tokenID := generateTokenID()

	claims := &NephoranJWTClaims{

		RegisteredClaims: jwt.RegisteredClaims{

			ID: tokenID,

			Subject: userInfo.Subject,

			Audience: jwt.ClaimStrings{jm.issuer},

			ExpiresAt: jwt.NewNumericDate(now.Add(jm.refreshTTL)),

			NotBefore: jwt.NewNumericDate(now),

			IssuedAt: jwt.NewNumericDate(now),

			Issuer: jm.issuer,
		},

		Email: userInfo.Email,

		Name: userInfo.Name,

		Provider: userInfo.Provider,

		ProviderID: userInfo.ProviderID,

		SessionID: sessionID,

		TokenType: "refresh",

		IPAddress: opts.IPAddress,

		UserAgent: opts.UserAgent,
	}

	// Apply custom TTL if specified.

	if opts.TTL > 0 {

		claims.ExpiresAt = jwt.NewNumericDate(now.Add(opts.TTL))

	}

	// Sign token.

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	token.Header["kid"] = jm.keyID

	tokenString, err := token.SignedString(jm.signingKey)

	if err != nil {

		return "", nil, fmt.Errorf("failed to sign refresh token: %w", err)

	}

	// Store token info.

	tokenInfo := &TokenInfo{

		TokenID: tokenID,

		UserID: userInfo.Subject,

		SessionID: sessionID,

		TokenType: "refresh",

		IssuedAt: now,

		ExpiresAt: claims.ExpiresAt.Time,

		Provider: userInfo.Provider,

		IPAddress: opts.IPAddress,

		UserAgent: opts.UserAgent,

		LastUsed: now,

		UseCount: 0,
	}

	if err := jm.tokenStore.StoreToken(ctx, tokenID, tokenInfo); err != nil {

		jm.logger.Warn("Failed to store refresh token info", "error", err)

	}

	jm.metrics.TokensIssued++

	return tokenString, tokenInfo, nil

}

// ValidateToken validates a JWT token and returns claims.

func (jm *JWTManager) ValidateToken(ctx context.Context, tokenString string) (*NephoranJWTClaims, error) {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	// Parse and validate token.

	token, err := jwt.ParseWithClaims(tokenString, &NephoranJWTClaims{}, func(token *jwt.Token) (interface{}, error) {

		// Verify signing method.

		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {

			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])

		}

		// Verify key ID.

		if kidInterface, exists := token.Header["kid"]; exists {

			if kid, ok := kidInterface.(string); ok && kid != jm.keyID {

				// Check if this is from a previous key rotation.

				// In production, you'd maintain multiple keys for rotation.

				jm.logger.Warn("Token signed with different key ID", "token_kid", kid, "current_kid", jm.keyID)

			}

		}

		return jm.verifyingKey, nil

	})

	if err != nil {

		jm.metrics.ValidationFailures++

		return nil, fmt.Errorf("failed to parse token: %w", err)

	}

	if !token.Valid {

		jm.metrics.ValidationFailures++

		return nil, fmt.Errorf("invalid token")

	}

	claims, ok := token.Claims.(*NephoranJWTClaims)

	if !ok {

		jm.metrics.ValidationFailures++

		return nil, fmt.Errorf("invalid token claims")

	}

	// Check if token is blacklisted.

	if blacklisted, err := jm.blacklist.IsTokenBlacklisted(ctx, claims.ID); err != nil {

		jm.logger.Warn("Failed to check token blacklist", "error", err)

	} else if blacklisted {

		jm.metrics.ValidationFailures++

		return nil, fmt.Errorf("token is blacklisted")

	}

	// Update token usage.

	if tokenInfo, err := jm.tokenStore.GetToken(ctx, claims.ID); err == nil && tokenInfo != nil {

		tokenInfo.LastUsed = time.Now()

		tokenInfo.UseCount++

		if err := jm.tokenStore.UpdateToken(ctx, claims.ID, tokenInfo); err != nil {

			jm.logger.Warn("Failed to update token usage", "error", err)

		}

	}

	jm.metrics.TokensValidated++

	return claims, nil

}

// RefreshAccessToken generates a new access token using a refresh token.

func (jm *JWTManager) RefreshAccessToken(ctx context.Context, refreshTokenString string, options ...TokenOption) (string, *TokenInfo, error) {

	// Validate refresh token.

	claims, err := jm.ValidateToken(ctx, refreshTokenString)

	if err != nil {

		return "", nil, fmt.Errorf("invalid refresh token: %w", err)

	}

	if claims.TokenType != "refresh" {

		return "", nil, fmt.Errorf("token is not a refresh token")

	}

	// Create user info from refresh token claims.

	userInfo := &providers.UserInfo{

		Subject: claims.Subject,

		Email: claims.Email,

		EmailVerified: claims.EmailVerified,

		Name: claims.Name,

		PreferredName: claims.PreferredName,

		Picture: claims.Picture,

		Groups: claims.Groups,

		Roles: claims.Roles,

		Permissions: claims.Permissions,

		Provider: claims.Provider,

		ProviderID: claims.ProviderID,

		Attributes: claims.Attributes,
	}

	// Generate new access token.

	return jm.GenerateAccessToken(ctx, userInfo, claims.SessionID, options...)

}

// RevokeToken revokes a token by adding it to the blacklist.

func (jm *JWTManager) RevokeToken(ctx context.Context, tokenString string) error {

	// Parse token to get claims without signature validation.

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, _, err := parser.ParseUnverified(tokenString, &NephoranJWTClaims{})

	if err != nil {

		return fmt.Errorf("failed to parse token for revocation: %w", err)

	}

	claims, ok := token.Claims.(*NephoranJWTClaims)

	if !ok {

		return fmt.Errorf("invalid token claims")

	}

	// Add to blacklist.

	if err := jm.blacklist.BlacklistToken(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {

		return fmt.Errorf("failed to blacklist token: %w", err)

	}

	// Remove from token store.

	if err := jm.tokenStore.DeleteToken(ctx, claims.ID); err != nil {

		jm.logger.Warn("Failed to delete token from store", "error", err)

	}

	jm.metrics.TokensRevoked++

	return nil

}

// RevokeUserTokens revokes all tokens for a specific user.

func (jm *JWTManager) RevokeUserTokens(ctx context.Context, userID string) error {

	tokens, err := jm.tokenStore.ListUserTokens(ctx, userID)

	if err != nil {

		return fmt.Errorf("failed to list user tokens: %w", err)

	}

	for _, token := range tokens {

		if err := jm.blacklist.BlacklistToken(ctx, token.TokenID, token.ExpiresAt); err != nil {

			jm.logger.Warn("Failed to blacklist token", "token_id", token.TokenID, "error", err)

			continue

		}

		if err := jm.tokenStore.DeleteToken(ctx, token.TokenID); err != nil {

			jm.logger.Warn("Failed to delete token from store", "token_id", token.TokenID, "error", err)

		}

		jm.metrics.TokensRevoked++

	}

	return nil

}

// GetTokenInfo retrieves token information.

func (jm *JWTManager) GetTokenInfo(ctx context.Context, tokenID string) (*TokenInfo, error) {

	return jm.tokenStore.GetToken(ctx, tokenID)

}

// ListUserTokens lists all active tokens for a user.

func (jm *JWTManager) ListUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {

	return jm.tokenStore.ListUserTokens(ctx, userID)

}

// IsTokenBlacklisted checks if a token is blacklisted.

func (jm *JWTManager) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {

	// Parse token to get token ID without signature validation.

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, _, err := parser.ParseUnverified(tokenString, &NephoranJWTClaims{})

	if err != nil {

		return false, fmt.Errorf("failed to parse token: %w", err)

	}

	claims, ok := token.Claims.(*NephoranJWTClaims)

	if !ok {

		return false, fmt.Errorf("invalid token claims")

	}

	return jm.blacklist.IsTokenBlacklisted(ctx, claims.ID)

}

// GenerateTokenPair generates both access and refresh tokens.

func (jm *JWTManager) GenerateTokenPair(ctx context.Context, userInfo *providers.UserInfo, sessionID string, options ...TokenOption) (string, string, error) {

	accessToken, _, err := jm.GenerateAccessToken(ctx, userInfo, sessionID, options...)

	if err != nil {

		return "", "", err

	}

	refreshToken, _, err := jm.GenerateRefreshToken(ctx, userInfo, sessionID, options...)

	if err != nil {

		return "", "", err

	}

	return accessToken, refreshToken, nil

}

// Deprecated methods for backward compatibility with tests.

// GenerateToken generates an access token (deprecated: use GenerateAccessToken).

func (jm *JWTManager) GenerateToken(userInfo *providers.UserInfo, customClaims map[string]interface{}) (string, error) {

	return jm.generateLegacyToken(userInfo, customClaims, 0)

}

// GenerateTokenWithTTL generates an access token with custom TTL (deprecated: use GenerateAccessToken with options).

func (jm *JWTManager) GenerateTokenWithTTL(userInfo *providers.UserInfo, customClaims map[string]interface{}, ttl time.Duration) (string, error) {

	return jm.generateLegacyToken(userInfo, customClaims, ttl)

}

// GenerateTokenPair with old signature (deprecated).

func (jm *JWTManager) GenerateTokenPairLegacy(userInfo *providers.UserInfo, customClaims map[string]interface{}) (string, string, error) {

	return jm.GenerateTokenPair(context.Background(), userInfo, "legacy-session")

}

// RefreshToken with old signature (deprecated).

func (jm *JWTManager) RefreshToken(refreshTokenString string) (string, string, error) {

	newAccessToken, _, err := jm.RefreshAccessToken(context.Background(), refreshTokenString)

	if err != nil {

		return "", "", err

	}

	return newAccessToken, refreshTokenString, nil

}

// BlacklistToken alias for RevokeToken (deprecated).

func (jm *JWTManager) BlacklistToken(tokenString string) error {

	return jm.RevokeToken(context.Background(), tokenString)

}

// CleanupBlacklist cleans up expired blacklist entries.

func (jm *JWTManager) CleanupBlacklist() error {

	return jm.blacklist.CleanupExpired(context.Background())

}

// ValidateTokenWithContext validates a token with a specific context (alias for ValidateToken).

func (jm *JWTManager) ValidateTokenWithContext(ctx context.Context, tokenString string) (*NephoranJWTClaims, error) {

	return jm.ValidateToken(ctx, tokenString)

}

// generateLegacyToken generates a token with custom claims as top-level fields for backward compatibility.

func (jm *JWTManager) generateLegacyToken(userInfo *providers.UserInfo, customClaims map[string]interface{}, ttl time.Duration) (string, error) {

	if userInfo == nil {

		return "", fmt.Errorf("user info cannot be nil")

	}

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	now := time.Now()

	tokenID := generateTokenID()

	// Create base claims using MapClaims for flexibility with custom claims.

	claims := jwt.MapClaims{

		"jti": tokenID,

		"sub": userInfo.Subject,

		"aud": jm.issuer,

		"iss": jm.issuer,

		"iat": now.Unix(),

		"nbf": now.Unix(),

		// User information.

		"email": userInfo.Email,

		"email_verified": userInfo.EmailVerified,

		"name": userInfo.Name,

		"preferred_username": userInfo.Username,

		"picture": userInfo.Picture,

		// Authorization.

		"groups": userInfo.Groups,

		"roles": userInfo.Roles,

		"permissions": userInfo.Permissions,

		"organizations": extractOrganizationNames(userInfo.Organizations),

		// Provider information.

		"provider": userInfo.Provider,

		"provider_id": userInfo.ProviderID,

		// Nephoran-specific.

		"session_id": "legacy-session",

		"token_type": "access",
	}

	// Set expiration.

	if ttl > 0 {

		claims["exp"] = now.Add(ttl).Unix()

	} else {

		claims["exp"] = now.Add(jm.defaultTTL).Unix()

	}

	// Add custom claims as top-level fields.
	// S1031: Range over maps is safe even if the map is nil, no check needed

	for key, value := range customClaims {

		claims[key] = value

	}

	// Create and sign token.

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	token.Header["kid"] = jm.keyID

	tokenString, err := token.SignedString(jm.signingKey)

	if err != nil {

		jm.metrics.ValidationFailures++

		return "", fmt.Errorf("failed to sign token: %w", err)

	}

	jm.metrics.TokensIssued++

	return tokenString, nil

}

// GetMetrics returns JWT metrics.

func (jm *JWTManager) GetMetrics() *JWTMetrics {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	// Return a copy to avoid race conditions.

	metrics := *jm.metrics

	return &metrics

}

// Private methods.

func (jm *JWTManager) initializeSigningKey(config *JWTConfig) error {

	var privateKey *rsa.PrivateKey

	var err error

	if config.SigningKeyPath != "" {

		// Load key from file.

		privateKey, err = loadRSAPrivateKeyFromFile(config.SigningKeyPath)

		if err != nil {

			return fmt.Errorf("failed to load signing key from file: %w", err)

		}

	} else if config.SigningKey != "" {

		// Parse key from string.

		privateKey, err = parseRSAPrivateKey(config.SigningKey)

		if err != nil {

			return fmt.Errorf("failed to parse signing key: %w", err)

		}

	} else {

		// Generate new key.

		privateKey, err = rsa.GenerateKey(rand.Reader, 2048)

		if err != nil {

			return fmt.Errorf("failed to generate signing key: %w", err)

		}

		jm.logger.Info("Generated new RSA signing key")

	}

	jm.signingKey = privateKey

	jm.verifyingKey = &privateKey.PublicKey

	jm.keyID = generateKeyID()

	return nil

}

func (jm *JWTManager) keyRotationLoop() {

	if jm.keyRotationPeriod == 0 {

		return // Key rotation disabled

	}

	ticker := time.NewTicker(jm.keyRotationPeriod)

	defer ticker.Stop()

	for range ticker.C {

		if err := jm.rotateSigningKey(); err != nil {

			jm.logger.Error("Failed to rotate signing key", "error", err)

		}

	}

}

func (jm *JWTManager) rotateSigningKey() error {

	jm.mutex.Lock()

	defer jm.mutex.Unlock()

	// Generate new key.

	newKey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {

		return fmt.Errorf("failed to generate new signing key: %w", err)

	}

	// Update keys.

	// In production, you'd want to maintain the old key for some time to validate existing tokens.

	jm.signingKey = newKey

	jm.verifyingKey = &newKey.PublicKey

	jm.keyID = generateKeyID()

	jm.metrics.KeyRotations++

	jm.logger.Info("Rotated JWT signing key", "new_key_id", jm.keyID)

	return nil

}

func (jm *JWTManager) cleanupLoop() {

	ticker := time.NewTicker(1 * time.Hour)

	defer ticker.Stop()

	for range ticker.C {

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

		// Cleanup expired tokens.

		if err := jm.tokenStore.CleanupExpired(ctx); err != nil {

			jm.logger.Error("Failed to cleanup expired tokens", "error", err)

		}

		// Cleanup expired blacklist entries.

		if err := jm.blacklist.CleanupExpired(ctx); err != nil {

			jm.logger.Error("Failed to cleanup expired blacklist entries", "error", err)

		}

		cancel()

	}

}

// Token options.

// TokenOption represents a tokenoption.

type TokenOption func(*TokenOptions)

// TokenOptions represents a tokenoptions.

type TokenOptions struct {
	TTL time.Duration

	Scope string

	IPAddress string

	UserAgent string
}

// WithTTL performs withttl operation.

func WithTTL(ttl time.Duration) TokenOption {

	return func(opts *TokenOptions) {

		opts.TTL = ttl

	}

}

// WithScope performs withscope operation.

func WithScope(scope string) TokenOption {

	return func(opts *TokenOptions) {

		opts.Scope = scope

	}

}

// WithIPAddress performs withipaddress operation.

func WithIPAddress(ip string) TokenOption {

	return func(opts *TokenOptions) {

		opts.IPAddress = ip

	}

}

// WithUserAgent performs withuseragent operation.

func WithUserAgent(ua string) TokenOption {

	return func(opts *TokenOptions) {

		opts.UserAgent = ua

	}

}

func applyTokenOptions(options ...TokenOption) *TokenOptions {

	opts := &TokenOptions{}

	for _, option := range options {

		option(opts)

	}

	return opts

}

// Helper functions.

func generateTokenID() string {

	// Generate a random token ID.

	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {

		// Fallback to time-based ID if random generation fails.

		return fmt.Sprintf("%x", time.Now().UnixNano())

	}

	return fmt.Sprintf("%x", b)

}

func generateKeyID() string {

	// Generate a random key ID.

	b := make([]byte, 8)

	if _, err := rand.Read(b); err != nil {

		// Fallback to time-based ID if random generation fails.

		return fmt.Sprintf("key-%x", time.Now().UnixNano())

	}

	return fmt.Sprintf("%x", b)

}

func loadRSAPrivateKeyFromFile(filename string) (*rsa.PrivateKey, error) {

	// Implementation would read PEM file and parse RSA key.

	return nil, fmt.Errorf("key loading from file not implemented")

}

func parseRSAPrivateKey(keyData string) (*rsa.PrivateKey, error) {

	block, _ := pem.Decode([]byte(keyData))

	if block == nil {

		return nil, fmt.Errorf("failed to decode PEM block")

	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)

	if err != nil {

		// Try PKCS8.

		pkcs8Key, err := x509.ParsePKCS8PrivateKey(block.Bytes)

		if err != nil {

			return nil, fmt.Errorf("failed to parse RSA private key: %w", err)

		}

		rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)

		if !ok {

			return nil, fmt.Errorf("key is not an RSA private key")

		}

		return rsaKey, nil

	}

	return key, nil

}

func extractOrganizationNames(orgs []providers.Organization) []string {

	names := make([]string, len(orgs))

	for i, org := range orgs {

		names[i] = org.Name

	}

	return names

}

// Getter methods for testing.

// GetIssuer returns the issuer for testing purposes.

func (jm *JWTManager) GetIssuer() string {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	return jm.issuer

}

// GetDefaultTTL returns the default TTL for testing purposes.

func (jm *JWTManager) GetDefaultTTL() time.Duration {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	return jm.defaultTTL

}

// GetRefreshTTL returns the refresh TTL for testing purposes.

func (jm *JWTManager) GetRefreshTTL() time.Duration {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	return jm.refreshTTL

}

// GetRequireSecureCookies returns the secure cookies requirement for testing purposes.

func (jm *JWTManager) GetRequireSecureCookies() bool {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	return jm.requireSecureCookies

}

// GetKeyID returns the current key ID for testing purposes.

func (jm *JWTManager) GetKeyID() string {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	return jm.keyID

}

// SetSigningKey sets the signing key for the JWT manager.

func (jm *JWTManager) SetSigningKey(privateKey *rsa.PrivateKey, keyID string) error {

	jm.mutex.Lock()

	defer jm.mutex.Unlock()

	if privateKey == nil {

		return fmt.Errorf("private key cannot be nil")

	}

	if keyID == "" {

		return fmt.Errorf("key ID cannot be empty")

	}

	jm.signingKey = privateKey

	jm.verifyingKey = &privateKey.PublicKey

	jm.keyID = keyID

	return nil

}

// GetPublicKey returns the public key for a given key ID.

func (jm *JWTManager) GetPublicKey(keyID string) (*rsa.PublicKey, error) {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	if keyID == "" {

		return nil, fmt.Errorf("key ID cannot be empty")

	}

	if keyID != jm.keyID {

		return nil, fmt.Errorf("key not found")

	}

	return jm.verifyingKey, nil

}

// RotateKeys generates new signing keys.

func (jm *JWTManager) RotateKeys() error {

	return jm.rotateSigningKey()

}

// ExtractClaims extracts claims from a token without full validation.

func (jm *JWTManager) ExtractClaims(tokenString string) (jwt.MapClaims, error) {

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})

	if err != nil {

		return nil, fmt.Errorf("failed to parse token: %w", err)

	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok {

		return nil, fmt.Errorf("invalid token claims")

	}

	return claims, nil

}

// GetJWKS returns the JSON Web Key Set for public key verification.

func (jm *JWTManager) GetJWKS() (map[string]interface{}, error) {

	jm.mutex.RLock()

	defer jm.mutex.RUnlock()

	if jm.verifyingKey == nil {

		return nil, fmt.Errorf("verifying key not initialized")

	}

	// Create JWKS response.

	return map[string]interface{}{

		"keys": []interface{}{

			map[string]interface{}{

				"kty": "RSA",

				"use": "sig",

				"kid": jm.keyID,

				"alg": "RS256",

				"n": encodeBase64URL(jm.verifyingKey.N.Bytes()),

				"e": encodeBase64URL(big.NewInt(int64(jm.verifyingKey.E)).Bytes()),
			},
		},
	}, nil

}

// encodeBase64URL encodes bytes to base64 URL encoding without padding.

func encodeBase64URL(data []byte) string {

	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")

}
