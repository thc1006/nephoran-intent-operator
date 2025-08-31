// Package testutil provides testing utilities and helpers for auth package.

package testutil

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// Import types from main auth package to avoid circular dependencies.

type NephoranJWTClaims struct {
	jwt.RegisteredClaims

	Email string `json:"email"`

	EmailVerified bool `json:"email_verified"`

	Name string `json:"name"`

	PreferredName string `json:"preferred_username"`

	Picture string `json:"picture"`

	Groups []string `json:"groups"`

	Roles []string `json:"roles"`

	Permissions []string `json:"permissions"`

	Organizations []string `json:"organizations"`

	Provider string `json:"provider"`

	ProviderID string `json:"provider_id"`

	TenantID string `json:"tenant_id,omitempty"`

	SessionID string `json:"session_id"`

	TokenType string `json:"token_type"`

	Scope string `json:"scope,omitempty"`

	IPAddress string `json:"ip_address,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// TokenInfo represents stored token information.

type TokenInfo struct {
	TokenID string `json:"token_id"`

	UserID string `json:"user_id"`

	SessionID string `json:"session_id"`

	TokenType string `json:"token_type"`

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

// Note: Role and Permission types have been moved to contracts.go as TestRole and TestPermission.

// to avoid circular dependencies and type conflicts.

// Role represents a role with associated permissions (alias to TestRole).

type Role = TestRole

// Permission represents a specific permission (alias to TestPermission).

type Permission = TestPermission

// AccessRequest represents an access control request (test copy).

type AccessRequest struct {
	UserID string `json:"user_id"`

	Resource string `json:"resource"`

	Action string `json:"action"`

	Context map[string]interface{} `json:"context,omitempty"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`

	IPAddress string `json:"ip_address,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	RequestID string `json:"request_id,omitempty"`
}

// AccessDecision represents the result of an access control evaluation (test copy).

type AccessDecision struct {
	Allowed bool `json:"allowed"`

	Reason string `json:"reason"`

	AppliedPolicies []string `json:"applied_policies"`

	RequiredRoles []string `json:"required_roles,omitempty"`

	MissingPermissions []string `json:"missing_permissions,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`

	EvaluatedAt time.Time `json:"evaluated_at"`

	TTL time.Duration `json:"ttl,omitempty"`
}

// TokenOption represents token generation options.

type TokenOption func(*TokenOptions)

// TokenOptions contains token generation options.

type TokenOptions struct {
	TTL time.Duration

	Scope string

	IPAddress string

	UserAgent string
}

// Helper functions for token options.

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

// JWTManagerMock provides mock JWT functionality with interface compatibility.

type JWTManagerMock struct {
	privateKey *rsa.PrivateKey

	keyID string

	blacklistedTokens map[string]bool

	tokenStore map[string]*TokenInfo

	mutex sync.RWMutex
}

// NewJWTManagerMock creates a new JWT manager mock.

func NewJWTManagerMock() *JWTManagerMock {

	return &JWTManagerMock{

		blacklistedTokens: make(map[string]bool),

		tokenStore: make(map[string]*TokenInfo),
	}

}

// GenerateAccessToken generates an access token for a user (matches real interface).

func (j *JWTManagerMock) GenerateAccessToken(ctx context.Context, userInfo *providers.UserInfo, sessionID string, options ...TokenOption) (string, *TokenInfo, error) {

	j.mutex.Lock()

	defer j.mutex.Unlock()

	if j.privateKey == nil {

		return "", nil, fmt.Errorf("private key not set")

	}

	opts := &TokenOptions{}

	for _, opt := range options {

		opt(opts)

	}

	now := time.Now()

	tokenID := fmt.Sprintf("token-%d", time.Now().UnixNano())

	ttl := time.Hour

	if opts.TTL > 0 {

		ttl = opts.TTL

	}

	claims := &NephoranJWTClaims{

		RegisteredClaims: jwt.RegisteredClaims{

			ID: tokenID,

			Subject: userInfo.Subject,

			Audience: jwt.ClaimStrings{"test-audience"},

			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),

			NotBefore: jwt.NewNumericDate(now),

			IssuedAt: jwt.NewNumericDate(now),

			Issuer: "test-issuer",
		},

		Email: userInfo.Email,

		EmailVerified: userInfo.EmailVerified,

		Name: userInfo.Name,

		PreferredName: userInfo.PreferredName,

		Picture: userInfo.Picture,

		Groups: userInfo.Groups,

		Roles: userInfo.Roles,

		Permissions: userInfo.Permissions,

		Provider: userInfo.Provider,

		ProviderID: userInfo.ProviderID,

		SessionID: sessionID,

		TokenType: "access",

		Scope: opts.Scope,

		IPAddress: opts.IPAddress,

		UserAgent: opts.UserAgent,

		Attributes: userInfo.Attributes,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	token.Header["kid"] = j.keyID

	tokenString, err := token.SignedString(j.privateKey)

	if err != nil {

		return "", nil, err

	}

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

	j.tokenStore[tokenID] = tokenInfo

	return tokenString, tokenInfo, nil

}

// GenerateRefreshToken generates a refresh token.

func (j *JWTManagerMock) GenerateRefreshToken(ctx context.Context, userInfo *providers.UserInfo, sessionID string, options ...TokenOption) (string, *TokenInfo, error) {

	j.mutex.Lock()

	defer j.mutex.Unlock()

	if j.privateKey == nil {

		return "", nil, fmt.Errorf("private key not set")

	}

	opts := &TokenOptions{}

	for _, opt := range options {

		opt(opts)

	}

	now := time.Now()

	tokenID := fmt.Sprintf("refresh-%d", time.Now().UnixNano())

	ttl := 24 * time.Hour

	if opts.TTL > 0 {

		ttl = opts.TTL

	}

	claims := &NephoranJWTClaims{

		RegisteredClaims: jwt.RegisteredClaims{

			ID: tokenID,

			Subject: userInfo.Subject,

			Audience: jwt.ClaimStrings{"test-audience"},

			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),

			NotBefore: jwt.NewNumericDate(now),

			IssuedAt: jwt.NewNumericDate(now),

			Issuer: "test-issuer",
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

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	token.Header["kid"] = j.keyID

	tokenString, err := token.SignedString(j.privateKey)

	if err != nil {

		return "", nil, err

	}

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

	j.tokenStore[tokenID] = tokenInfo

	return tokenString, tokenInfo, nil

}

// ValidateToken validates a JWT token and returns claims (matches real interface).

func (j *JWTManagerMock) ValidateToken(ctx context.Context, tokenString string) (*NephoranJWTClaims, error) {

	j.mutex.RLock()

	defer j.mutex.RUnlock()

	// Check if token is blacklisted.

	if j.blacklistedTokens != nil && j.blacklistedTokens[tokenString] {

		return nil, fmt.Errorf("token has been revoked")

	}

	// Parse and validate token.

	token, err := jwt.ParseWithClaims(tokenString, &NephoranJWTClaims{}, func(token *jwt.Token) (interface{}, error) {

		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {

			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])

		}

		return &j.privateKey.PublicKey, nil

	})

	if err != nil {

		return nil, fmt.Errorf("failed to parse token: %w", err)

	}

	if !token.Valid {

		return nil, fmt.Errorf("invalid token")

	}

	claims, ok := token.Claims.(*NephoranJWTClaims)

	if !ok {

		return nil, fmt.Errorf("invalid token claims")

	}

	// Update token usage if found in store.

	if j.tokenStore != nil {

		if tokenInfo, exists := j.tokenStore[claims.ID]; exists {

			tokenInfo.LastUsed = time.Now()

			tokenInfo.UseCount++

		}

	}

	return claims, nil

}

// RefreshAccessToken generates a new access token using a refresh token.

func (j *JWTManagerMock) RefreshAccessToken(ctx context.Context, refreshTokenString string, options ...TokenOption) (string, *TokenInfo, error) {

	// Validate refresh token.

	claims, err := j.ValidateToken(ctx, refreshTokenString)

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

	return j.GenerateAccessToken(ctx, userInfo, claims.SessionID, options...)

}

// RevokeToken revokes a token by adding it to the blacklist.

func (j *JWTManagerMock) RevokeToken(ctx context.Context, tokenString string) error {

	j.mutex.Lock()

	defer j.mutex.Unlock()

	// Parse token to get ID.

	token, err := jwt.ParseWithClaims(tokenString, &NephoranJWTClaims{}, func(token *jwt.Token) (interface{}, error) {

		return &j.privateKey.PublicKey, nil

	})

	if err != nil && !strings.Contains(err.Error(), "token is expired") {

		return fmt.Errorf("failed to parse token for revocation: %w", err)

	}

	if claims, ok := token.Claims.(*NephoranJWTClaims); ok {

		// Add to blacklist.

		j.blacklistedTokens[tokenString] = true

		// Remove from token store.

		delete(j.tokenStore, claims.ID)

	}

	return nil

}

// RevokeUserTokens revokes all tokens for a specific user.

func (j *JWTManagerMock) RevokeUserTokens(ctx context.Context, userID string) error {

	j.mutex.Lock()

	defer j.mutex.Unlock()

	tokensToRevoke := []string{}

	for tokenID, tokenInfo := range j.tokenStore {

		if tokenInfo.UserID == userID {

			tokensToRevoke = append(tokensToRevoke, tokenID)

		}

	}

	for _, tokenID := range tokensToRevoke {

		delete(j.tokenStore, tokenID)

	}

	return nil

}

// GetTokenInfo retrieves token information.

func (j *JWTManagerMock) GetTokenInfo(ctx context.Context, tokenID string) (*TokenInfo, error) {

	j.mutex.RLock()

	defer j.mutex.RUnlock()

	tokenInfo, exists := j.tokenStore[tokenID]

	if !exists {

		return nil, fmt.Errorf("token not found")

	}

	return tokenInfo, nil

}

// ListUserTokens lists all active tokens for a user.

func (j *JWTManagerMock) ListUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {

	j.mutex.RLock()

	defer j.mutex.RUnlock()

	var tokens []*TokenInfo

	for _, tokenInfo := range j.tokenStore {

		if tokenInfo.UserID == userID {

			tokens = append(tokens, tokenInfo)

		}

	}

	return tokens, nil

}

// SetSigningKey sets the signing key for the mock.

func (j *JWTManagerMock) SetSigningKey(privateKey *rsa.PrivateKey, keyID string) error {

	j.mutex.Lock()

	defer j.mutex.Unlock()

	j.privateKey = privateKey

	j.keyID = keyID

	return nil

}

// Deprecated: GenerateToken is deprecated, use GenerateAccessToken instead.

func (j *JWTManagerMock) GenerateToken(user *providers.UserInfo, customClaims map[string]interface{}) (string, error) {

	// Convert to new interface.

	tokenString, _, err := j.GenerateAccessToken(context.Background(), user, "test-session")

	return tokenString, err

}

// GenerateTokenWithTTL performs generatetokenwithttl operation.

func (j *JWTManagerMock) GenerateTokenWithTTL(user *providers.UserInfo, customClaims map[string]interface{}, ttl time.Duration) (string, error) {

	tokenString, _, err := j.GenerateAccessToken(context.Background(), user, "test-session", WithTTL(ttl))

	return tokenString, err

}

// GenerateTokenPair performs generatetokenpair operation.

func (j *JWTManagerMock) GenerateTokenPair(user *providers.UserInfo, customClaims map[string]interface{}) (string, string, error) {

	accessToken, _, err := j.GenerateAccessToken(context.Background(), user, "test-session")

	if err != nil {

		return "", "", err

	}

	refreshToken, _, err := j.GenerateRefreshToken(context.Background(), user, "test-session")

	if err != nil {

		return "", "", err

	}

	return accessToken, refreshToken, nil

}

// Deprecated: ValidateToken with *jwt.Token return is deprecated.

func (j *JWTManagerMock) ValidateTokenLegacy(tokenString string) (*jwt.Token, error) {

	// Check if token is blacklisted.

	if j.blacklistedTokens != nil && j.blacklistedTokens[tokenString] {

		return nil, fmt.Errorf("token has been revoked")

	}

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

		return &j.privateKey.PublicKey, nil

	})

}

// RefreshToken performs refreshtoken operation.

func (j *JWTManagerMock) RefreshToken(refreshTokenString string) (string, string, error) {

	accessToken, _, err := j.RefreshAccessToken(context.Background(), refreshTokenString)

	if err != nil {

		return "", "", err

	}

	return accessToken, refreshTokenString, nil

}

// BlacklistToken performs blacklisttoken operation.

func (j *JWTManagerMock) BlacklistToken(tokenString string) error {

	return j.RevokeToken(context.Background(), tokenString)

}

// CleanupBlacklist performs cleanupblacklist operation.

func (j *JWTManagerMock) CleanupBlacklist() error {

	// Mock implementation - in real scenario would clean up expired tokens.

	return nil

}

// Close performs close operation.

func (j *JWTManagerMock) Close() {

	// Mock implementation.

}

// RBACManagerMock provides mock RBAC functionality with interface compatibility.

type RBACManagerMock struct {
	roles map[string][]string // userID -> roles

	permissions map[string][]string // role -> permissions

	roleStore map[string]*TestRole // roleID -> role

	permissionStore map[string]*TestPermission // permissionID -> permission

	mutex sync.RWMutex
}

// NewRBACManagerMock creates a new RBAC manager mock.

func NewRBACManagerMock() *RBACManagerMock {

	return &RBACManagerMock{

		roles: make(map[string][]string),

		permissions: make(map[string][]string),

		roleStore: make(map[string]*TestRole),

		permissionStore: make(map[string]*TestPermission),
	}

}

// GrantRoleToUser assigns a role to a user (matches real interface).

func (r *RBACManagerMock) GrantRoleToUser(ctx context.Context, userID, roleID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	// Verify role exists.

	if _, exists := r.roleStore[roleID]; !exists {

		return fmt.Errorf("role %s does not exist", roleID)

	}

	// Add role to user.

	userRoles := r.roles[userID]

	for _, existingRole := range userRoles {

		if existingRole == roleID {

			return nil // Already has role

		}

	}

	r.roles[userID] = append(userRoles, roleID)

	return nil

}

// RevokeRoleFromUser removes a role from a user (matches real interface).

func (r *RBACManagerMock) RevokeRoleFromUser(ctx context.Context, userID, roleID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	userRoles := r.roles[userID]

	for i, role := range userRoles {

		if role == roleID {

			// Remove role.

			r.roles[userID] = append(userRoles[:i], userRoles[i+1:]...)

			return nil

		}

	}

	return fmt.Errorf("user %s does not have role %s", userID, roleID)

}

// GetUserRoles returns all roles for a user (matches real interface).

func (r *RBACManagerMock) GetUserRoles(ctx context.Context, userID string) []string {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	roles := r.roles[userID]

	result := make([]string, len(roles))

	copy(result, roles)

	return result

}

// GetUserPermissions returns all permissions for a user (matches real interface).

func (r *RBACManagerMock) GetUserPermissions(ctx context.Context, userID string) []string {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	permissionSet := make(map[string]bool)

	var allPermissions []string

	// Get permissions from user's roles.

	userRoles := r.roles[userID]

	for _, roleID := range userRoles {

		if role, exists := r.roleStore[roleID]; exists {

			for _, permID := range role.Permissions {

				if !permissionSet[permID] {

					allPermissions = append(allPermissions, permID)

					permissionSet[permID] = true

				}

			}

		}

	}

	return allPermissions

}

// CheckPermission checks if a user has a specific permission (matches real interface).

func (r *RBACManagerMock) CheckPermission(ctx context.Context, userID, permission string) bool {

	userPermissions := r.GetUserPermissions(ctx, userID)

	for _, perm := range userPermissions {

		if r.matchesPermission(perm, permission) {

			return true

		}

	}

	return false

}

// CheckAccess evaluates an access request against RBAC policies (matches real interface).

func (r *RBACManagerMock) CheckAccess(ctx context.Context, request *AccessRequest) *AccessDecision {

	startTime := time.Now()

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	decision := &AccessDecision{

		EvaluatedAt: startTime,

		Metadata: make(map[string]interface{}),
	}

	// Get user permissions.

	userPermissions := r.GetUserPermissions(ctx, request.UserID)

	userRoles := r.GetUserRoles(ctx, request.UserID)

	// Check if user has required permission.

	requiredPermission := fmt.Sprintf("%s:%s", request.Resource, request.Action)

	hasPermission := false

	for _, perm := range userPermissions {

		if r.matchesPermission(perm, requiredPermission) {

			hasPermission = true

			break

		}

	}

	if !hasPermission {

		decision.Allowed = false

		decision.Reason = fmt.Sprintf("User lacks required permission: %s", requiredPermission)

		decision.MissingPermissions = []string{requiredPermission}

		return decision

	}

	decision.Allowed = true

	decision.Reason = "Permission granted by RBAC"

	decision.Metadata["user_roles"] = userRoles

	decision.Metadata["checked_permission"] = requiredPermission

	return decision

}

// CreateRole creates a new role (matches real interface).

func (r *RBACManagerMock) CreateRole(ctx context.Context, role *TestRole) (*TestRole, error) {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if role.ID == "" {

		return nil, fmt.Errorf("role ID cannot be empty")

	}

	if _, exists := r.roleStore[role.ID]; exists {

		return nil, fmt.Errorf("role %s already exists", role.ID)

	}

	// Validate permissions exist.

	for _, permID := range role.Permissions {

		if _, exists := r.permissionStore[permID]; !exists {

			return nil, fmt.Errorf("permission %s does not exist", permID)

		}

	}

	now := time.Now()

	role.CreatedAt = now

	role.UpdatedAt = now

	r.roleStore[role.ID] = role

	return role, nil

}

// UpdateRole updates an existing role (matches real interface).

func (r *RBACManagerMock) UpdateRole(ctx context.Context, role *TestRole) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if _, exists := r.roleStore[role.ID]; !exists {

		return fmt.Errorf("role %s does not exist", role.ID)

	}

	// Validate permissions exist.

	for _, permID := range role.Permissions {

		if _, exists := r.permissionStore[permID]; !exists {

			return fmt.Errorf("permission %s does not exist", permID)

		}

	}

	role.UpdatedAt = time.Now()

	r.roleStore[role.ID] = role

	return nil

}

// DeleteRole deletes a role (matches real interface).

func (r *RBACManagerMock) DeleteRole(ctx context.Context, roleID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if _, exists := r.roleStore[roleID]; !exists {

		return fmt.Errorf("role %s does not exist", roleID)

	}

	// Remove role from all users.

	for userID, userRoles := range r.roles {

		for i, role := range userRoles {

			if role == roleID {

				r.roles[userID] = append(userRoles[:i], userRoles[i+1:]...)

				break

			}

		}

	}

	delete(r.roleStore, roleID)

	return nil

}

// ListRoles returns all roles (matches real interface).

func (r *RBACManagerMock) ListRoles(ctx context.Context) []*TestRole {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	roles := make([]*TestRole, 0, len(r.roleStore))

	for _, role := range r.roleStore {

		roles = append(roles, role)

	}

	return roles

}

// GetRole returns a specific role (matches real interface).

func (r *RBACManagerMock) GetRole(ctx context.Context, roleID string) (*TestRole, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	role, exists := r.roleStore[roleID]

	if !exists {

		return nil, fmt.Errorf("role %s does not exist", roleID)

	}

	return role, nil

}

// CreatePermission creates a new permission (matches real interface).

func (r *RBACManagerMock) CreatePermission(ctx context.Context, perm *TestPermission) (*TestPermission, error) {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if perm.ID == "" {

		return nil, fmt.Errorf("permission ID cannot be empty")

	}

	if _, exists := r.permissionStore[perm.ID]; exists {

		return nil, fmt.Errorf("permission %s already exists", perm.ID)

	}

	now := time.Now()

	perm.CreatedAt = now

	perm.UpdatedAt = now

	r.permissionStore[perm.ID] = perm

	return perm, nil

}

// ListPermissions returns all permissions (matches real interface).

func (r *RBACManagerMock) ListPermissions(ctx context.Context) []*TestPermission {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	permissions := make([]*TestPermission, 0, len(r.permissionStore))

	for _, perm := range r.permissionStore {

		permissions = append(permissions, perm)

	}

	return permissions

}

// AssignRolesFromClaims assigns roles based on JWT claims and provider groups.

func (r *RBACManagerMock) AssignRolesFromClaims(ctx context.Context, userInfo *providers.UserInfo) error {

	userID := userInfo.Subject

	// Clear existing roles for fresh assignment.

	r.mutex.Lock()

	r.roles[userID] = []string{}

	r.mutex.Unlock()

	// Map provider groups/roles to Nephoran roles.

	roleMappings := []string{"read-only"} // Default role

	for _, roleID := range roleMappings {

		if err := r.GrantRoleToUser(ctx, userID, roleID); err != nil {

			// If role doesn't exist, create a basic one.

			if strings.Contains(err.Error(), "does not exist") {

				basicRole := &TestRole{

					ID: roleID,

					Name: roleID,

					Description: fmt.Sprintf("Auto-generated role: %s", roleID),

					Permissions: []string{"read:basic"},

					CreatedAt: time.Now(),

					UpdatedAt: time.Now(),
				}

				if _, err := r.CreateRole(ctx, basicRole); err != nil {
					// Log error but continue - this is test utility code
					// Note: Error creating basic role but continuing execution
					_ = err // Acknowledge error but continue
				}

				if err := r.GrantRoleToUser(ctx, userID, roleID); err != nil {
					// Log error but continue - this is test utility code
					// Note: Error granting role to user but continuing execution
					_ = err // Acknowledge error but continue
				}

			}

		}

	}

	return nil

}

// Helper method for permission matching.

func (r *RBACManagerMock) matchesPermission(granted, required string) bool {

	// Handle wildcard permissions.

	if granted == "*" || granted == required {

		return true

	}

	// Handle resource-level wildcards (e.g., "intent:*" matches "intent:read").

	if strings.HasSuffix(granted, ":*") {

		grantedResource := strings.TrimSuffix(granted, ":*")

		requiredParts := strings.SplitN(required, ":", 2)

		if len(requiredParts) == 2 && requiredParts[0] == grantedResource {

			return true

		}

	}

	return false

}

// Deprecated methods for backward compatibility.

func (r *RBACManagerMock) CheckPermissionLegacy(ctx context.Context, userID, resource, action string) (bool, error) {

	permission := fmt.Sprintf("%s:%s", resource, action)

	allowed := r.CheckPermission(ctx, userID, permission)

	return allowed, nil

}

// AssignRole performs assignrole operation.

func (r *RBACManagerMock) AssignRole(ctx context.Context, userID, role string) error {

	return r.GrantRoleToUser(ctx, userID, role)

}

// RevokeRole performs revokerole operation.

func (r *RBACManagerMock) RevokeRole(ctx context.Context, userID, role string) error {

	return r.RevokeRoleFromUser(ctx, userID, role)

}

// GetRolePermissions performs getrolepermissions operation.

func (r *RBACManagerMock) GetRolePermissions(ctx context.Context, role string) ([]string, error) {

	if r.permissions == nil {

		return []string{}, nil

	}

	return r.permissions[role], nil

}

// AssignRoleToUser performs assignroletouser operation.

func (r *RBACManagerMock) AssignRoleToUser(ctx context.Context, userID, roleID string) error {

	return r.GrantRoleToUser(ctx, userID, roleID)

}

// Rest of the file continues with SessionManagerMock and TestContext...

// (The rest remains the same as the original file).

// SessionManagerMock provides mock session functionality.

type SessionManagerMock struct {
	sessions map[string]*MockSession

	mutex sync.RWMutex

	config SessionConfig
}

// SessionConfig represents a sessionconfig.

type SessionConfig struct {
	SessionTTL time.Duration
}

// MockSession represents a mocksession.

type MockSession struct {
	ID string

	UserID string

	UserInfo *providers.UserInfo

	CreatedAt time.Time

	ExpiresAt time.Time

	Data map[string]interface{}
}

// NewSessionManagerMock performs newsessionmanagermock operation.

func NewSessionManagerMock() *SessionManagerMock {

	return &SessionManagerMock{

		sessions: make(map[string]*MockSession),

		config: SessionConfig{

			SessionTTL: time.Hour,
		},
	}

}

// CreateSession performs createsession operation.

func (s *SessionManagerMock) CreateSession(ctx context.Context, userInfo *providers.UserInfo, metadata ...map[string]interface{}) (*MockSession, error) {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	session := &MockSession{

		ID: fmt.Sprintf("test-session-%d", time.Now().UnixNano()),

		UserID: userInfo.Subject,

		UserInfo: userInfo,

		CreatedAt: time.Now(),

		ExpiresAt: time.Now().Add(time.Hour),

		Data: make(map[string]interface{}),
	}

	// Add metadata if provided.

	if len(metadata) > 0 {

		for k, v := range metadata[0] {

			session.Data[k] = v

		}

	}

	s.sessions[session.ID] = session

	return session, nil

}

// GetSession performs getsession operation.

func (s *SessionManagerMock) GetSession(ctx context.Context, sessionID string) (*MockSession, error) {

	s.mutex.RLock()

	defer s.mutex.RUnlock()

	session, exists := s.sessions[sessionID]

	if !exists {

		return nil, fmt.Errorf("session not found")

	}

	return session, nil

}

// UpdateSession performs updatesession operation.

func (s *SessionManagerMock) UpdateSession(ctx context.Context, sessionID string, updates map[string]interface{}) error {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	session, exists := s.sessions[sessionID]

	if !exists {

		return fmt.Errorf("session not found")

	}

	for k, v := range updates {

		session.Data[k] = v

	}

	return nil

}

// DeleteSession performs deletesession operation.

func (s *SessionManagerMock) DeleteSession(ctx context.Context, sessionID string) error {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	delete(s.sessions, sessionID)

	return nil

}

// ListUserSessions performs listusersessions operation.

func (s *SessionManagerMock) ListUserSessions(ctx context.Context, userID string) ([]*MockSession, error) {

	s.mutex.RLock()

	defer s.mutex.RUnlock()

	var sessions []*MockSession

	for _, session := range s.sessions {

		if session.UserID == userID {

			sessions = append(sessions, session)

		}

	}

	return sessions, nil

}

// ValidateSession performs validatesession operation.

func (s *SessionManagerMock) ValidateSession(ctx context.Context, sessionID string) (*MockSession, error) {

	session, err := s.GetSession(ctx, sessionID)

	if err != nil {

		return nil, err

	}

	// Check if session is expired.

	if time.Now().After(session.ExpiresAt) {

		return nil, fmt.Errorf("session expired")

	}

	return session, nil

}

// RefreshSession performs refreshsession operation.

func (s *SessionManagerMock) RefreshSession(ctx context.Context, sessionID string) (*MockSession, error) {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	session, exists := s.sessions[sessionID]

	if !exists {

		return nil, fmt.Errorf("session not found")

	}

	// Extend session expiry.

	session.ExpiresAt = time.Now().Add(time.Hour)

	return session, nil

}

// SetSessionCookie performs setsessioncookie operation.

func (s *SessionManagerMock) SetSessionCookie(w http.ResponseWriter, sessionID string) {

	http.SetCookie(w, &http.Cookie{

		Name: "test-session",

		Value: sessionID,

		Path: "/",

		HttpOnly: true,

		Secure: false, // For testing

	})

}

// GetSessionFromRequest performs getsessionfromrequest operation.

func (s *SessionManagerMock) GetSessionFromRequest(r *http.Request) (*MockSession, error) {

	cookie, err := r.Cookie("test-session")

	if err != nil {

		return nil, err

	}

	return s.GetSession(r.Context(), cookie.Value)

}

// CleanupExpiredSessions performs cleanupexpiredsessions operation.

func (s *SessionManagerMock) CleanupExpiredSessions() error {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	now := time.Now()

	for id, session := range s.sessions {

		if now.After(session.ExpiresAt) {

			delete(s.sessions, id)

		}

	}

	return nil

}

// Close performs close operation.

func (s *SessionManagerMock) Close() {

	// Mock implementation.

}

// Additional methods needed for comprehensive tests.

// UpdateSessionMetadata performs updatesessionmetadata operation.

func (s *SessionManagerMock) UpdateSessionMetadata(ctx context.Context, sessionID string, metadata map[string]interface{}) (*MockSession, error) {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	session, exists := s.sessions[sessionID]

	if !exists {

		return nil, fmt.Errorf("session not found")

	}

	// S1031: Range over maps is safe even if the map is nil, no check needed
	for k, v := range metadata {

		session.Data[k] = v

	}

	return session, nil

}

// GetUserSessions performs getusersessions operation.

func (s *SessionManagerMock) GetUserSessions(ctx context.Context, userID string) ([]*MockSession, error) {

	return s.ListUserSessions(ctx, userID)

}

// RevokeSession performs revokesession operation.

func (s *SessionManagerMock) RevokeSession(ctx context.Context, sessionID string) error {

	return s.DeleteSession(ctx, sessionID)

}

// RevokeAllUserSessions performs revokeallusersessions operation.

func (s *SessionManagerMock) RevokeAllUserSessions(ctx context.Context, userID string) error {

	s.mutex.Lock()

	defer s.mutex.Unlock()

	toDelete := []string{}

	for id, session := range s.sessions {

		if session.UserID == userID {

			toDelete = append(toDelete, id)

		}

	}

	for _, id := range toDelete {

		delete(s.sessions, id)

	}

	return nil

}

// GetSessionFromCookie performs getsessionfromcookie operation.

func (s *SessionManagerMock) GetSessionFromCookie(r *http.Request) (string, error) {

	cookie, err := r.Cookie("test-session")

	if err != nil {

		return "", err

	}

	return cookie.Value, nil

}

// ClearSessionCookie performs clearsessioncookie operation.

func (s *SessionManagerMock) ClearSessionCookie(w http.ResponseWriter) {

	http.SetCookie(w, &http.Cookie{

		Name: "test-session",

		Value: "",

		Path: "/",

		HttpOnly: true,

		Secure: false,

		MaxAge: -1,

		Expires: time.Now().Add(-time.Hour),
	})

}

// TestContext provides a complete testing environment for auth tests.

type TestContext struct {
	T *testing.T

	Ctx context.Context

	Logger *slog.Logger

	// Keys for testing.

	PrivateKey *rsa.PrivateKey

	PublicKey *rsa.PublicKey

	KeyID string

	// Test servers.

	OAuthServer *httptest.Server

	LDAPServer *MockLDAPServer

	// Mock implementations for testing.

	JWTManager *JWTManagerMock

	RBACManager *RBACManagerMock

	SessionManager *SessionManagerMock

	// Cleanup functions.

	cleanupFuncs []func()

	mutex sync.Mutex
}

// NewTestContext creates a new test context with default configuration.

func NewTestContext(t *testing.T) *TestContext {

	ctx := context.Background()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{

		Level: slog.LevelDebug,
	}))

	// Generate test RSA key pair.

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)

	require.NoError(t, err)

	publicKey := &privateKey.PublicKey

	keyID := "test-key-id"

	tc := &TestContext{

		T: t,

		Ctx: ctx,

		Logger: logger,

		PrivateKey: privateKey,

		PublicKey: publicKey,

		KeyID: keyID,

		JWTManager: NewJWTManagerMock(),

		RBACManager: NewRBACManagerMock(),

		SessionManager: NewSessionManagerMock(),
	}

	return tc

}

// SetupJWTManager initializes JWT manager mock for testing.

func (tc *TestContext) SetupJWTManager() *JWTManagerMock {

	// Set test keys.

	err := tc.JWTManager.SetSigningKey(tc.PrivateKey, tc.KeyID)

	require.NoError(tc.T, err)

	tc.AddCleanup(func() {

		tc.JWTManager.Close()

	})

	return tc.JWTManager

}

// SetupRBACManager initializes RBAC manager mock for testing.

func (tc *TestContext) SetupRBACManager() *RBACManagerMock {

	return tc.RBACManager

}

// SetupSessionManager initializes session manager mock for testing.

func (tc *TestContext) SetupSessionManager() *SessionManagerMock {

	tc.AddCleanup(func() {

		tc.SessionManager.Close()

	})

	return tc.SessionManager

}

// CreateTestToken creates a test JWT token.

func (tc *TestContext) CreateTestToken(claims jwt.MapClaims) string {

	if claims == nil {

		claims = jwt.MapClaims{}

	}

	// Set default claims if not provided.

	if claims["iss"] == nil {

		claims["iss"] = "test-issuer"

	}

	if claims["sub"] == nil {

		claims["sub"] = "test-user"

	}

	if claims["exp"] == nil {

		claims["exp"] = time.Now().Add(time.Hour).Unix()

	}

	if claims["iat"] == nil {

		claims["iat"] = time.Now().Unix()

	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	token.Header["kid"] = tc.KeyID

	tokenString, err := token.SignedString(tc.PrivateKey)

	require.NoError(tc.T, err)

	return tokenString

}

// CreateTestUser creates a test user info.

func (tc *TestContext) CreateTestUser(userID string) *providers.UserInfo {

	return &providers.UserInfo{

		Subject: userID,

		Email: fmt.Sprintf("%s@example.com", userID),

		EmailVerified: true,

		Name: fmt.Sprintf("Test %s", cases.Title(language.English).String(userID)),

		GivenName: "Test",

		FamilyName: cases.Title(language.English).String(userID),

		Username: userID,

		Provider: "test",

		ProviderID: fmt.Sprintf("test-%s", userID),

		Groups: []string{"users", "testers"},

		Roles: []string{"viewer"},

		Attributes: map[string]interface{}{

			"department": "engineering",

			"team": "platform",
		},
	}

}

// AddCleanup adds a cleanup function to be called when the test finishes.

func (tc *TestContext) AddCleanup(cleanup func()) {

	tc.mutex.Lock()

	defer tc.mutex.Unlock()

	tc.cleanupFuncs = append(tc.cleanupFuncs, cleanup)

}

// Cleanup performs all cleanup operations.

func (tc *TestContext) Cleanup() {

	tc.mutex.Lock()

	defer tc.mutex.Unlock()

	// Run cleanup functions in reverse order.

	for i := len(tc.cleanupFuncs) - 1; i >= 0; i-- {

		tc.cleanupFuncs[i]()

	}

	tc.cleanupFuncs = nil

}

// Assertion helpers.

func AssertNoError(t *testing.T, err error) {

	assert.NoError(t, err)

}

// AssertError performs asserterror operation.

func AssertError(t *testing.T, err error) {

	assert.Error(t, err)

}

// AssertEqual performs assertequal operation.

func AssertEqual(t *testing.T, expected, actual interface{}) {

	assert.Equal(t, expected, actual)

}

// AssertNotEqual performs assertnotequal operation.

func AssertNotEqual(t *testing.T, expected, actual interface{}) {

	assert.NotEqual(t, expected, actual)

}

// AssertContains performs assertcontains operation.

func AssertContains(t *testing.T, haystack, needle interface{}) {

	assert.Contains(t, haystack, needle)

}

// AssertNotContains performs assertnotcontains operation.

func AssertNotContains(t *testing.T, haystack, needle interface{}) {

	assert.NotContains(t, haystack, needle)

}

// AssertTrue performs asserttrue operation.

func AssertTrue(t *testing.T, value bool) {

	assert.True(t, value)

}

// AssertFalse performs assertfalse operation.

func AssertFalse(t *testing.T, value bool) {

	assert.False(t, value)

}

// AssertNil performs assertnil operation.

func AssertNil(t *testing.T, value interface{}) {

	assert.Nil(t, value)

}

// AssertNotNil performs assertnotnil operation.

func AssertNotNil(t *testing.T, value interface{}) {

	assert.NotNil(t, value)

}

// PEM helpers for key generation in tests.

func GenerateTestKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {

		return nil, nil, err

	}

	return privateKey, &privateKey.PublicKey, nil

}

// PrivateKeyToPEM performs privatekeytopem operation.

func PrivateKeyToPEM(key *rsa.PrivateKey) string {

	keyBytes := x509.MarshalPKCS1PrivateKey(key)

	keyBlock := &pem.Block{

		Type: "RSA PRIVATE KEY",

		Bytes: keyBytes,
	}

	return string(pem.EncodeToMemory(keyBlock))

}

// PublicKeyToPEM performs publickeytopem operation.

func PublicKeyToPEM(key *rsa.PublicKey) (string, error) {

	keyBytes, err := x509.MarshalPKIXPublicKey(key)

	if err != nil {

		return "", err

	}

	keyBlock := &pem.Block{

		Type: "PUBLIC KEY",

		Bytes: keyBytes,
	}

	return string(pem.EncodeToMemory(keyBlock)), nil

}

// Type aliases for backward compatibility.

type (
	MockJWTManager = JWTManagerMock

	// MockSessionManager represents a mocksessionmanager.

	MockSessionManager = SessionManagerMock

	// MockRBACManager represents a mockrbacmanager.

	MockRBACManager = RBACManagerMock
)
