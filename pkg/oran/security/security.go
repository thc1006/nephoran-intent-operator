package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecurityManager manages security configurations and operations
type SecurityManager struct {
	tlsManager   *TLSManager
	oauthManager *OAuthManager
	rbacManager  *RBACManager
	certManager  *CertificateManager
}

// TLSManager manages mutual TLS configurations
type TLSManager struct {
	mu           sync.RWMutex
	tlsConfigs   map[string]*tls.Config
	certPaths    map[string]*CertificatePaths
	autoReload   bool
	reloadTicker *time.Ticker
}

// OAuthManager manages OAuth2/OIDC authentication
type OAuthManager struct {
	mu         sync.RWMutex
	providers  map[string]*OIDCProvider
	tokenCache map[string]*TokenInfo
	httpClient *http.Client
}

// RBACManager manages role-based access control
type RBACManager struct {
	mu       sync.RWMutex
	policies map[string]*RBACPolicy
	roles    map[string]*Role
	users    map[string]*User
}

// CertificateManager manages certificate lifecycle
type CertificateManager struct {
	mu               sync.RWMutex
	certificates     map[string]*CertificateInfo
	caPool           *x509.CertPool
	autoRotate       bool
	rotationInterval time.Duration
}

// CertificatePaths holds paths to certificate files
type CertificatePaths struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// CertificateInfo holds certificate information
type CertificateInfo struct {
	Certificate  *x509.Certificate
	PrivateKey   interface{}
	NotBefore    time.Time
	NotAfter     time.Time
	Subject      string
	Issuer       string
	SerialNumber string
}

// OIDCProvider represents an OIDC provider configuration
type OIDCProvider struct {
	Name         string            `json:"name"`
	IssuerURL    string            `json:"issuer_url"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	RedirectURL  string            `json:"redirect_url"`
	Scopes       []string          `json:"scopes"`
	ExtraParams  map[string]string `json:"extra_params"`
	JWKSCache    *JWKSCache        `json:"-"`
}

// JWKSCache caches JWKS for token validation
type JWKSCache struct {
	mu          sync.RWMutex
	keys        interface{}
	lastUpdated time.Time
	ttl         time.Duration
}

// TokenInfo holds token information
type TokenInfo struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	TokenType    string                 `json:"token_type"`
	ExpiresIn    int64                  `json:"expires_in"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Claims       map[string]interface{} `json:"claims"`
	Subject      string                 `json:"subject"`
	Issuer       string                 `json:"issuer"`
	Audience     []string               `json:"audience"`
}

// RBACPolicy represents an RBAC policy
type RBACPolicy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Rules       []*PolicyRule          `json:"rules"`
	Subjects    []*Subject             `json:"subjects"`
	Resources   []*Resource            `json:"resources"`
	Conditions  map[string]interface{} `json:"conditions"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PolicyRule defines a policy rule
type PolicyRule struct {
	ID         string      `json:"id"`
	Effect     string      `json:"effect"` // ALLOW, DENY
	Actions    []string    `json:"actions"`
	Resources  []string    `json:"resources"`
	Conditions []Condition `json:"conditions"`
	Priority   int         `json:"priority"`
}

// Subject represents a policy subject (user, group, service)
type Subject struct {
	Type       string            `json:"type"` // user, group, service
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

// Resource represents a protected resource
type Resource struct {
	Type       string            `json:"type"` // api, vnf, policy
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

// Condition represents a policy condition
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, in, not_in, regex
	Value    interface{} `json:"value"`
}

// Role represents an RBAC role
type Role struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Permissions []string          `json:"permissions"`
	Attributes  map[string]string `json:"attributes"`
	CreatedAt   time.Time         `json:"created_at"`
}

// User represents a user
type User struct {
	ID         string            `json:"id"`
	Username   string            `json:"username"`
	Email      string            `json:"email"`
	Roles      []string          `json:"roles"`
	Groups     []string          `json:"groups"`
	Attributes map[string]string `json:"attributes"`
	Active     bool              `json:"active"`
	CreatedAt  time.Time         `json:"created_at"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	TLS   *TLSConfig   `json:"tls"`
	OAuth *OAuthConfig `json:"oauth"`
	RBAC  *RBACConfig  `json:"rbac"`
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled        bool                         `json:"enabled"`
	MutualTLS      bool                         `json:"mutual_tls"`
	Certificates   map[string]*CertificatePaths `json:"certificates"`
	CABundle       string                       `json:"ca_bundle"`
	MinVersion     string                       `json:"min_version"`
	CipherSuites   []string                     `json:"cipher_suites"`
	AutoReload     bool                         `json:"auto_reload"`
	ReloadInterval string                       `json:"reload_interval"`
}

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	Enabled        bool                     `json:"enabled"`
	Providers      map[string]*OIDCProvider `json:"providers"`
	DefaultScopes  []string                 `json:"default_scopes"`
	TokenTTL       string                   `json:"token_ttl"`
	RefreshEnabled bool                     `json:"refresh_enabled"`
	CacheEnabled   bool                     `json:"cache_enabled"`
	CacheTTL       string                   `json:"cache_ttl"`
}

// RBACConfig holds RBAC configuration
type RBACConfig struct {
	Enabled       bool     `json:"enabled"`
	PolicyPath    string   `json:"policy_path"`
	DefaultPolicy string   `json:"default_policy"` // ALLOW, DENY
	AdminUsers    []string `json:"admin_users"`
	AdminRoles    []string `json:"admin_roles"`
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityConfig) (*SecurityManager, error) {
	tlsManager, err := NewTLSManager(config.TLS)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS manager: %w", err)
	}

	oauthManager, err := NewOAuthManager(config.OAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth manager: %w", err)
	}

	rbacManager, err := NewRBACManager(config.RBAC)
	if err != nil {
		return nil, fmt.Errorf("failed to create RBAC manager: %w", err)
	}

	certManager := NewCertificateManager()

	return &SecurityManager{
		tlsManager:   tlsManager,
		oauthManager: oauthManager,
		rbacManager:  rbacManager,
		certManager:  certManager,
	}, nil
}

// NewTLSManager creates a new TLS manager
func NewTLSManager(config *TLSConfig) (*TLSManager, error) {
	if config == nil || !config.Enabled {
		return &TLSManager{
			tlsConfigs: make(map[string]*tls.Config),
			certPaths:  make(map[string]*CertificatePaths),
		}, nil
	}

	manager := &TLSManager{
		tlsConfigs: make(map[string]*tls.Config),
		certPaths:  make(map[string]*CertificatePaths),
		autoReload: config.AutoReload,
	}

	// Load certificates
	for name, certPath := range config.Certificates {
		tlsConfig, err := manager.loadTLSConfig(certPath, config)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config for %s: %w", name, err)
		}
		manager.tlsConfigs[name] = tlsConfig
		manager.certPaths[name] = certPath
	}

	// Start auto-reload if enabled
	if config.AutoReload {
		interval := 5 * time.Minute
		if config.ReloadInterval != "" {
			if d, err := time.ParseDuration(config.ReloadInterval); err == nil {
				interval = d
			}
		}
		manager.startAutoReload(interval)
	}

	return manager, nil
}

// loadTLSConfig loads TLS configuration from certificate files
func (m *TLSManager) loadTLSConfig(certPath *CertificatePaths, config *TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath.CertFile, certPath.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Set minimum TLS version
	if config.MinVersion != "" {
		switch config.MinVersion {
		case "1.0":
			tlsConfig.MinVersion = tls.VersionTLS10
		case "1.1":
			tlsConfig.MinVersion = tls.VersionTLS11
		case "1.2":
			tlsConfig.MinVersion = tls.VersionTLS12
		case "1.3":
			tlsConfig.MinVersion = tls.VersionTLS13
		}
	}

	// Configure mutual TLS
	if config.MutualTLS {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

		// Load CA bundle for client certificate verification
		if certPath.CAFile != "" {
			caCert, err := os.ReadFile(certPath.CAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA file: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}

			tlsConfig.ClientCAs = caCertPool
		}
	}

	return tlsConfig, nil
}

// startAutoReload starts automatic certificate reloading
func (m *TLSManager) startAutoReload(interval time.Duration) {
	m.reloadTicker = time.NewTicker(interval)
	go func() {
		for range m.reloadTicker.C {
			m.reloadCertificates()
		}
	}()
}

// reloadCertificates reloads all certificates
func (m *TLSManager) reloadCertificates() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, certPath := range m.certPaths {
		// Check if certificate files have been modified
		certInfo, err := os.Stat(certPath.CertFile)
		if err != nil {
			continue
		}

		keyInfo, err := os.Stat(certPath.KeyFile)
		if err != nil {
			continue
		}

		// Check if either file was modified recently
		now := time.Now()
		if now.Sub(certInfo.ModTime()) < time.Minute || now.Sub(keyInfo.ModTime()) < time.Minute {
			// Reload certificate
			cert, err := tls.LoadX509KeyPair(certPath.CertFile, certPath.KeyFile)
			if err != nil {
				log.Log.Error(err, "failed to reload certificate", "name", name)
				continue
			}

			if tlsConfig, ok := m.tlsConfigs[name]; ok {
				tlsConfig.Certificates = []tls.Certificate{cert}
				log.Log.Info("certificate reloaded successfully", "name", name)
			}
		}
	}
}

// GetTLSConfig returns TLS configuration for a given name
func (m *TLSManager) GetTLSConfig(name string) (*tls.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, ok := m.tlsConfigs[name]
	if !ok {
		return nil, fmt.Errorf("TLS config not found: %s", name)
	}

	return config.Clone(), nil
}

// NewOAuthManager creates a new OAuth manager
func NewOAuthManager(config *OAuthConfig) (*OAuthManager, error) {
	if config == nil || !config.Enabled {
		return &OAuthManager{
			providers:  make(map[string]*OIDCProvider),
			tokenCache: make(map[string]*TokenInfo),
		}, nil
	}

	manager := &OAuthManager{
		providers:  make(map[string]*OIDCProvider),
		tokenCache: make(map[string]*TokenInfo),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Initialize providers
	for name, provider := range config.Providers {
		provider.Name = name
		if err := manager.initializeProvider(provider); err != nil {
			return nil, fmt.Errorf("failed to initialize OAuth provider %s: %w", name, err)
		}
		manager.providers[name] = provider
	}

	// Start token cleanup routine
	go manager.startTokenCleanup()

	return manager, nil
}

// initializeProvider initializes an OIDC provider
func (m *OAuthManager) initializeProvider(provider *OIDCProvider) error {
	// Initialize JWKS cache
	provider.JWKSCache = &JWKSCache{
		ttl: 24 * time.Hour,
	}

	// Fetch and cache JWKS
	return m.refreshJWKS(provider)
}

// refreshJWKS refreshes JWKS from the provider
func (m *OAuthManager) refreshJWKS(provider *OIDCProvider) error {
	jwksURL := provider.IssuerURL + "/.well-known/jwks.json"

	resp, err := m.httpClient.Get(jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS: status %d", resp.StatusCode)
	}

	var jwks interface{}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	provider.JWKSCache.mu.Lock()
	provider.JWKSCache.keys = jwks
	provider.JWKSCache.lastUpdated = time.Now()
	provider.JWKSCache.mu.Unlock()

	return nil
}

// ValidateToken validates a JWT token
func (m *OAuthManager) ValidateToken(ctx context.Context, tokenString, providerName string) (*TokenInfo, error) {
	logger := log.FromContext(ctx)

	// Check cache first
	if tokenInfo, ok := m.getTokenFromCache(tokenString); ok {
		if time.Now().Before(tokenInfo.ExpiresAt) {
			return tokenInfo, nil
		}
		// Remove expired token from cache
		m.removeTokenFromCache(tokenString)
	}

	provider, ok := m.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("OAuth provider not found: %s", providerName)
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return m.getSigningKey(provider, token)
	})

	if err != nil {
		logger.Error(err, "failed to parse JWT token")
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Extract token information
	tokenInfo := &TokenInfo{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		Claims:      claims,
	}

	// Extract standard claims
	if sub, ok := claims["sub"].(string); ok {
		tokenInfo.Subject = sub
	}

	if iss, ok := claims["iss"].(string); ok {
		tokenInfo.Issuer = iss
	}

	if aud, ok := claims["aud"].([]interface{}); ok {
		for _, a := range aud {
			if audStr, ok := a.(string); ok {
				tokenInfo.Audience = append(tokenInfo.Audience, audStr)
			}
		}
	}

	if exp, ok := claims["exp"].(float64); ok {
		tokenInfo.ExpiresAt = time.Unix(int64(exp), 0)
	}

	// Cache token
	m.cacheToken(tokenString, tokenInfo)

	logger.Info("token validated successfully", "subject", tokenInfo.Subject)
	return tokenInfo, nil
}

// getSigningKey gets the signing key for token validation
func (m *OAuthManager) getSigningKey(provider *OIDCProvider, token *jwt.Token) (interface{}, error) {
	// For simplicity, this implementation returns a dummy key
	// In a real implementation, this would extract the correct key from JWKS
	// based on the token's "kid" (key ID) header

	provider.JWKSCache.mu.RLock()
	defer provider.JWKSCache.mu.RUnlock()

	// Check if JWKS needs refresh
	if time.Since(provider.JWKSCache.lastUpdated) > provider.JWKSCache.ttl {
		go func() {
			m.refreshJWKS(provider)
		}()
	}

	// This is a simplified implementation
	// In reality, you would parse the JWKS and return the appropriate key
	return []byte("your-secret-key"), nil
}

// getTokenFromCache retrieves token from cache
func (m *OAuthManager) getTokenFromCache(tokenString string) (*TokenInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tokenInfo, ok := m.tokenCache[tokenString]
	return tokenInfo, ok
}

// cacheToken caches a token
func (m *OAuthManager) cacheToken(tokenString string, tokenInfo *TokenInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokenCache[tokenString] = tokenInfo
}

// removeTokenFromCache removes token from cache
func (m *OAuthManager) removeTokenFromCache(tokenString string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tokenCache, tokenString)
}

// startTokenCleanup starts token cleanup routine
func (m *OAuthManager) startTokenCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupExpiredTokens()
	}
}

// cleanupExpiredTokens removes expired tokens from cache
func (m *OAuthManager) cleanupExpiredTokens() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for tokenString, tokenInfo := range m.tokenCache {
		if now.After(tokenInfo.ExpiresAt) {
			delete(m.tokenCache, tokenString)
		}
	}
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager(config *RBACConfig) (*RBACManager, error) {
	manager := &RBACManager{
		policies: make(map[string]*RBACPolicy),
		roles:    make(map[string]*Role),
		users:    make(map[string]*User),
	}

	if config != nil && config.Enabled {
		// Load policies from file
		if config.PolicyPath != "" {
			if err := manager.loadPoliciesFromFile(config.PolicyPath); err != nil {
				return nil, fmt.Errorf("failed to load RBAC policies: %w", err)
			}
		}

		// Create admin users and roles
		manager.initializeAdminAccess(config)
	}

	return manager, nil
}

// loadPoliciesFromFile loads RBAC policies from a file
func (m *RBACManager) loadPoliciesFromFile(policyPath string) error {
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	var policies []*RBACPolicy
	if err := json.Unmarshal(data, &policies); err != nil {
		return fmt.Errorf("failed to parse policy file: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, policy := range policies {
		m.policies[policy.ID] = policy
	}

	return nil
}

// initializeAdminAccess initializes admin users and roles
func (m *RBACManager) initializeAdminAccess(config *RBACConfig) {
	// Create admin role
	adminRole := &Role{
		ID:          "admin",
		Name:        "Administrator",
		Description: "Full system access",
		Permissions: []string{"*"},
		CreatedAt:   time.Now(),
	}
	m.roles[adminRole.ID] = adminRole

	// Create admin users
	for _, username := range config.AdminUsers {
		user := &User{
			ID:        username,
			Username:  username,
			Roles:     []string{"admin"},
			Active:    true,
			CreatedAt: time.Now(),
		}
		m.users[user.ID] = user
	}
}

// CheckPermission checks if a subject has permission for a resource
func (m *RBACManager) CheckPermission(ctx context.Context, subject, action, resource string) (bool, error) {
	logger := log.FromContext(ctx)

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get user information
	user, ok := m.users[subject]
	if !ok {
		logger.Info("user not found", "subject", subject)
		return false, nil
	}

	if !user.Active {
		logger.Info("user is not active", "subject", subject)
		return false, nil
	}

	// Check user roles
	for _, roleID := range user.Roles {
		role, ok := m.roles[roleID]
		if !ok {
			continue
		}

		// Check if role has required permission
		for _, permission := range role.Permissions {
			if permission == "*" || permission == action || m.matchesPattern(permission, action) {
				logger.Info("permission granted", "subject", subject, "action", action, "resource", resource, "via_role", roleID)
				return true, nil
			}
		}
	}

	// Check policies
	for _, policy := range m.policies {
		if m.evaluatePolicy(policy, subject, action, resource) {
			logger.Info("permission granted via policy", "subject", subject, "action", action, "resource", resource, "policy", policy.ID)
			return true, nil
		}
	}

	logger.Info("permission denied", "subject", subject, "action", action, "resource", resource)
	return false, nil
}

// evaluatePolicy evaluates an RBAC policy
func (m *RBACManager) evaluatePolicy(policy *RBACPolicy, subject, action, resource string) bool {
	// Check if subject matches policy subjects
	subjectMatches := false
	for _, policySubject := range policy.Subjects {
		if policySubject.ID == subject || policySubject.Name == subject {
			subjectMatches = true
			break
		}
	}

	if !subjectMatches {
		return false
	}

	// Check policy rules
	for _, rule := range policy.Rules {
		if m.evaluateRule(rule, action, resource) {
			return rule.Effect == "ALLOW"
		}
	}

	return false
}

// evaluateRule evaluates a policy rule
func (m *RBACManager) evaluateRule(rule *PolicyRule, action, resource string) bool {
	// Check actions
	actionMatches := false
	for _, ruleAction := range rule.Actions {
		if ruleAction == "*" || ruleAction == action || m.matchesPattern(ruleAction, action) {
			actionMatches = true
			break
		}
	}

	if !actionMatches {
		return false
	}

	// Check resources
	resourceMatches := false
	for _, ruleResource := range rule.Resources {
		if ruleResource == "*" || ruleResource == resource || m.matchesPattern(ruleResource, resource) {
			resourceMatches = true
			break
		}
	}

	return resourceMatches
}

// matchesPattern checks if a string matches a pattern (supports wildcards)
func (m *RBACManager) matchesPattern(pattern, value string) bool {
	// Simple wildcard matching
	if strings.Contains(pattern, "*") {
		prefix := strings.Split(pattern, "*")[0]
		return strings.HasPrefix(value, prefix)
	}
	return pattern == value
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager() *CertificateManager {
	return &CertificateManager{
		certificates:     make(map[string]*CertificateInfo),
		caPool:           x509.NewCertPool(),
		autoRotate:       true,
		rotationInterval: 30 * 24 * time.Hour, // 30 days
	}
}

// LoadCertificate loads a certificate from file
func (m *CertificateManager) LoadCertificate(name, certFile string) error {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	certInfo := &CertificateInfo{
		Certificate:  cert,
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
	}

	m.mu.Lock()
	m.certificates[name] = certInfo
	m.mu.Unlock()

	return nil
}

// GetCertificateInfo returns certificate information
func (m *CertificateManager) GetCertificateInfo(name string) (*CertificateInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.certificates[name]
	if !ok {
		return nil, fmt.Errorf("certificate not found: %s", name)
	}

	return info, nil
}

// CheckCertificateExpiry checks if certificates are expiring
func (m *CertificateManager) CheckCertificateExpiry(ctx context.Context) {
	logger := log.FromContext(ctx)

	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	warningThreshold := 30 * 24 * time.Hour // 30 days

	for name, cert := range m.certificates {
		timeToExpiry := cert.NotAfter.Sub(now)

		if timeToExpiry < 0 {
			logger.Error(fmt.Errorf("certificate expired"), "certificate", name, "expired_at", cert.NotAfter)
		} else if timeToExpiry < warningThreshold {
			logger.Info("certificate expiring soon", "certificate", name, "expires_at", cert.NotAfter, "days_remaining", int(timeToExpiry.Hours()/24))
		}
	}
}

// StartCertificateMonitoring starts certificate monitoring
func (m *CertificateManager) StartCertificateMonitoring(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.CheckCertificateExpiry(ctx)
		}
	}
}

// CreateMiddleware creates HTTP middleware for security
func (sm *SecurityManager) CreateMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Validate Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Validate token (assuming default provider)
			tokenInfo, err := sm.oauthManager.ValidateToken(ctx, tokenString, "default")
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Check RBAC permissions
			action := r.Method
			resource := r.URL.Path

			allowed, err := sm.rbacManager.CheckPermission(ctx, tokenInfo.Subject, action, resource)
			if err != nil {
				http.Error(w, "Permission check failed", http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			// Add user context
			ctx = context.WithValue(ctx, "user", tokenInfo)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// Start starts the security manager
func (sm *SecurityManager) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("starting security manager")

	// Start certificate monitoring
	go sm.certManager.StartCertificateMonitoring(ctx)

	// Start JWKS refresh for OAuth providers
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, provider := range sm.oauthManager.providers {
					if err := sm.oauthManager.refreshJWKS(provider); err != nil {
						logger.Error(err, "failed to refresh JWKS", "provider", provider.Name)
					}
				}
			}
		}
	}()

	return nil
}

// Stop stops the security manager
func (sm *SecurityManager) Stop() {
	if sm.tlsManager.reloadTicker != nil {
		sm.tlsManager.reloadTicker.Stop()
	}
}
