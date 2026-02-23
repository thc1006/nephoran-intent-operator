package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecurityManager manages security configurations and operations.

type SecurityManager struct {
	tlsManager *TLSManager

	oauthManager *OAuthManager

	rbacManager *RBACManager

	certManager *CertificateManager
}

// TLSManager manages mutual TLS configurations.

type TLSManager struct {
	mu sync.RWMutex

	tlsConfigs map[string]*tls.Config

	certPaths map[string]*CertificatePaths

	autoReload bool

	reloadTicker *time.Ticker
}

// OAuthManager manages OAuth2/OIDC authentication.

type OAuthManager struct {
	mu sync.RWMutex

	providers map[string]*OIDCProvider

	tokenCache map[string]*TokenInfo

	httpClient *http.Client
}

// RBACManager manages role-based access control.

type RBACManager struct {
	mu sync.RWMutex

	policies map[string]*RBACPolicy

	roles map[string]*Role

	users map[string]*User
}

// CertificateManager manages certificate lifecycle.

type CertificateManager struct {
	mu sync.RWMutex

	certificates map[string]*CertificateInfo

	caPool *x509.CertPool

	autoRotate bool

	rotationInterval time.Duration
}

// CertificatePaths holds paths to certificate files.

type CertificatePaths struct {
	CertFile string

	KeyFile string

	CAFile string
}

// CertificateInfo holds certificate information.

type CertificateInfo struct {
	Certificate *x509.Certificate

	PrivateKey interface{}

	NotBefore time.Time

	NotAfter time.Time

	Subject string

	Issuer string

	DNSNames []string

	IPAddresses []string
}

// OIDCProvider represents an OIDC provider configuration.

type OIDCProvider struct {
	Name string

	IssuerURL string

	ClientID string

	ClientSecret string

	RedirectURL string

	Scopes []string

	JWKSCache *JWKSCache
}

// JWKSCache caches JWKS keys.

type JWKSCache struct {
	mu sync.RWMutex

	keys map[string]interface{}

	lastUpdated time.Time

	ttl time.Duration
}

// TokenInfo holds token information.

type TokenInfo struct {
	AccessToken string `json:"access_token"`

	RefreshToken string `json:"refresh_token"`

	TokenType string `json:"token_type"`

	ExpiresIn int64 `json:"expires_in"`

	ExpiresAt time.Time `json:"expires_at"`

	Claims json.RawMessage `json:"claims"`

	Subject string `json:"subject"`

	Issuer string `json:"issuer"`

	Audience []string `json:"audience"`
}

// RBACPolicy represents an RBAC policy.

type RBACPolicy struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Rules []RBACRule `json:"rules"`
}

// RBACRule represents an RBAC rule.

type RBACRule struct {
	Resources []string `json:"resources"`

	Actions []string `json:"actions"`

	Effect string `json:"effect"`

	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// Role represents a user role.

type Role struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Policies []string `json:"policies"`

	Permissions []Permission `json:"permissions"`
}

// Permission represents a specific permission.

type Permission struct {
	Resource string `json:"resource"`

	Actions []string `json:"actions"`

	Conditions map[string]interface{} `json:"conditions,omitempty"`
}

// User represents a user.

type User struct {
	ID string `json:"id"`

	Username string `json:"username"`

	Email string `json:"email"`

	Roles []string `json:"roles"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// NewSecurityManager creates a new security manager.

func NewSecurityManager() *SecurityManager {
	tlsManager := &TLSManager{
		tlsConfigs: make(map[string]*tls.Config),

		certPaths: make(map[string]*CertificatePaths),
	}

	oauthManager := &OAuthManager{
		providers: make(map[string]*OIDCProvider),

		tokenCache: make(map[string]*TokenInfo),

		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
	}

	rbacManager := &RBACManager{
		policies: make(map[string]*RBACPolicy),

		roles: make(map[string]*Role),

		users: make(map[string]*User),
	}

	certManager := &CertificateManager{
		certificates: make(map[string]*CertificateInfo),

		caPool: x509.NewCertPool(),
	}

	return &SecurityManager{
		tlsManager: tlsManager,

		oauthManager: oauthManager,

		rbacManager: rbacManager,

		certManager: certManager,
	}
}

// ConfigureTLS configures TLS for a specific service.

func (sm *SecurityManager) ConfigureTLS(serviceName string, certPaths *CertificatePaths) error {
	cert, err := tls.LoadX509KeyPair(certPaths.CertFile, certPaths.KeyFile)

	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},

		MinVersion: tls.VersionTLS13,

		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,

			tls.TLS_AES_256_GCM_SHA384,

			tls.TLS_CHACHA20_POLY1305_SHA256,
		},

		PreferServerCipherSuites: true,

		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	// Load CA certificate if provided.

	if certPaths.CAFile != "" {
		caCert, err := os.ReadFile(certPaths.CAFile)

		if err != nil {
			return fmt.Errorf("failed to load CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()

		caCertPool.AppendCertsFromPEM(caCert)

		config.ClientCAs = caCertPool
	}

	sm.tlsManager.mu.Lock()

	sm.tlsManager.tlsConfigs[serviceName] = config

	sm.tlsManager.certPaths[serviceName] = certPaths

	sm.tlsManager.mu.Unlock()

	return nil
}

// GetTLSConfig returns the TLS configuration for a service.

func (sm *SecurityManager) GetTLSConfig(serviceName string) (*tls.Config, error) {
	sm.tlsManager.mu.RLock()

	defer sm.tlsManager.mu.RUnlock()

	config, exists := sm.tlsManager.tlsConfigs[serviceName]

	if !exists {
		return nil, fmt.Errorf("TLS configuration not found for service: %s", serviceName)
	}

	return config, nil
}

// EnableTLSAutoReload enables automatic certificate reloading.

func (sm *SecurityManager) EnableTLSAutoReload(interval time.Duration) {
	sm.tlsManager.mu.Lock()

	defer sm.tlsManager.mu.Unlock()

	sm.tlsManager.autoReload = true

	sm.tlsManager.reloadTicker = time.NewTicker(interval)

	go sm.tlsAutoReloadLoop()
}

// tlsAutoReloadLoop runs the TLS auto-reload loop.

func (sm *SecurityManager) tlsAutoReloadLoop() {
	for range sm.tlsManager.reloadTicker.C {
		sm.tlsManager.mu.RLock()

		certPaths := make(map[string]*CertificatePaths)

		for service, paths := range sm.tlsManager.certPaths {
			certPaths[service] = paths
		}

		sm.tlsManager.mu.RUnlock()

		// Reload certificates.

		for serviceName, paths := range certPaths {
			err := sm.ConfigureTLS(serviceName, paths)

			if err != nil {
				log.Log.Error(err, "failed to reload TLS configuration", "service", serviceName)
			}
		}
	}
}

// ConfigureOIDCProvider configures an OIDC provider.

func (sm *SecurityManager) ConfigureOIDCProvider(provider *OIDCProvider) error {
	if provider.JWKSCache == nil {
		provider.JWKSCache = &JWKSCache{
			keys: make(map[string]interface{}),

			ttl: 1 * time.Hour,
		}
	}

	sm.oauthManager.mu.Lock()

	sm.oauthManager.providers[provider.Name] = provider

	sm.oauthManager.mu.Unlock()

	// Initialize JWKS cache.

	return sm.oauthManager.refreshJWKS(provider)
}

// ValidateToken validates a JWT token using the configured OIDC provider.

func (sm *SecurityManager) ValidateToken(providerName, tokenString string) (*TokenInfo, error) {
	sm.oauthManager.mu.RLock()

	provider, exists := sm.oauthManager.providers[providerName]

	sm.oauthManager.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("OIDC provider not found: %s", providerName)
	}

	return sm.oauthManager.validateToken(provider, tokenString)
}

// AuthorizeRequest authorizes a request using RBAC policies.

func (sm *SecurityManager) AuthorizeRequest(userID, resource, action string) (bool, error) {
	return sm.rbacManager.authorize(userID, resource, action)
}

// AddRBACPolicy adds an RBAC policy.

func (sm *SecurityManager) AddRBACPolicy(policy *RBACPolicy) error {
	sm.rbacManager.mu.Lock()

	defer sm.rbacManager.mu.Unlock()

	sm.rbacManager.policies[policy.Name] = policy

	return nil
}

// AddRole adds a role.

func (sm *SecurityManager) AddRole(role *Role) error {
	sm.rbacManager.mu.Lock()

	defer sm.rbacManager.mu.Unlock()

	sm.rbacManager.roles[role.Name] = role

	return nil
}

// AddUser adds a user.

func (sm *SecurityManager) AddUser(user *User) error {
	sm.rbacManager.mu.Lock()

	defer sm.rbacManager.mu.Unlock()

	sm.rbacManager.users[user.ID] = user

	return nil
}

// RegisterCertificate registers a certificate for monitoring and rotation.

func (sm *SecurityManager) RegisterCertificate(name string, cert *x509.Certificate, key interface{}) {
	certInfo := &CertificateInfo{
		Certificate: cert,

		PrivateKey: key,

		NotBefore: cert.NotBefore,

		NotAfter: cert.NotAfter,

		Subject: cert.Subject.String(),

		Issuer: cert.Issuer.String(),

		DNSNames: cert.DNSNames,

		IPAddresses: make([]string, len(cert.IPAddresses)),
	}

	for i, ip := range cert.IPAddresses {
		certInfo.IPAddresses[i] = ip.String()
	}

	sm.certManager.mu.Lock()

	sm.certManager.certificates[name] = certInfo

	sm.certManager.mu.Unlock()
}

// EnableCertificateAutoRotation enables automatic certificate rotation.

func (sm *SecurityManager) EnableCertificateAutoRotation(interval time.Duration) {
	sm.certManager.mu.Lock()

	defer sm.certManager.mu.Unlock()

	sm.certManager.autoRotate = true

	sm.certManager.rotationInterval = interval

	go sm.certAutoRotationLoop()
}

// certAutoRotationLoop runs the certificate auto-rotation loop.

func (sm *SecurityManager) certAutoRotationLoop() {
	ticker := time.NewTicker(sm.certManager.rotationInterval)

	defer ticker.Stop()

	for range ticker.C {
		sm.certManager.mu.RLock()

		certs := make(map[string]*CertificateInfo)

		for name, cert := range sm.certManager.certificates {
			certs[name] = cert
		}

		sm.certManager.mu.RUnlock()

		// Check for certificates that need rotation.

		for name, cert := range certs {
			// Rotate if certificate expires within 30 days.

			if time.Until(cert.NotAfter) < 30*24*time.Hour {
				log.Log.Info("certificate needs rotation", "name", name, "expires", cert.NotAfter)

				// In a real implementation, this would trigger certificate renewal.

				// For now, we just log the event.
			}
		}
	}
}

// validateToken validates a JWT token.

func (m *OAuthManager) validateToken(provider *OIDCProvider, tokenString string) (*TokenInfo, error) {
	logger := log.Log.WithName("oauth-manager")

	// Check cache first.

	if tokenInfo := m.getCachedToken(tokenString); tokenInfo != nil {
		if time.Now().Before(tokenInfo.ExpiresAt) {
			logger.V(1).Info("using cached token")

			return tokenInfo, nil
		}

		// Remove expired token from cache.

		m.removeCachedToken(tokenString)
	}

	// Parse and validate token.

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

	// Marshal claims to JSON for storage in TokenInfo
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Extract token information.

	tokenInfo := &TokenInfo{
		AccessToken: tokenString,

		TokenType: "Bearer",

		Claims: json.RawMessage(claimsJSON),
	}

	// Extract standard claims.

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
		// Safe timestamp conversion with overflow and validity checks

		if exp >= 0 && exp <= float64(1<<62) && exp > float64(time.Now().Unix()) {
			tokenInfo.ExpiresAt = time.Unix(int64(exp), 0)
		} else if exp <= float64(time.Now().Unix()) {
			return nil, fmt.Errorf("token has expired")
		} else {
			return nil, fmt.Errorf("invalid expiration time")
		}
	}

	// Cache token.

	m.cacheToken(tokenString, tokenInfo)

	logger.Info("token validated successfully", "subject", tokenInfo.Subject)

	return tokenInfo, nil
}

// getSigningKey gets the signing key for token validation.

func (m *OAuthManager) getSigningKey(provider *OIDCProvider, token *jwt.Token) (interface{}, error) {
	// For simplicity, this implementation returns a dummy key.

	// In a real implementation, this would extract the correct key from JWKS.

	// based on the token's "kid" (key ID) header.

	provider.JWKSCache.mu.RLock()

	defer provider.JWKSCache.mu.RUnlock()

	// Check if JWKS needs refresh.

	if time.Since(provider.JWKSCache.lastUpdated) > provider.JWKSCache.ttl {
		go func() {
			m.refreshJWKS(provider)
		}()
	}

	// This is a simplified implementation.

	// In reality, you would parse the JWKS and return the appropriate key.

	return []byte("your-secret-key"), nil
}

// refreshJWKS refreshes the JWKS cache.

func (m *OAuthManager) refreshJWKS(provider *OIDCProvider) error {
	// This is a simplified implementation.

	// In reality, you would fetch JWKS from the provider's JWKS endpoint.

	provider.JWKSCache.mu.Lock()

	defer provider.JWKSCache.mu.Unlock()

	// Simulate refreshing JWKS.

	provider.JWKSCache.keys = make(map[string]interface{})

	provider.JWKSCache.keys["default"] = []byte("your-secret-key")

	provider.JWKSCache.lastUpdated = time.Now()

	return nil
}

// getCachedToken gets a cached token.

func (m *OAuthManager) getCachedToken(tokenString string) *TokenInfo {
	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.tokenCache[tokenString]
}

// cacheToken caches a token.

func (m *OAuthManager) cacheToken(tokenString string, tokenInfo *TokenInfo) {
	m.mu.Lock()

	defer m.mu.Unlock()

	m.tokenCache[tokenString] = tokenInfo
}

// removeCachedToken removes a token from cache.

func (m *OAuthManager) removeCachedToken(tokenString string) {
	m.mu.Lock()

	defer m.mu.Unlock()

	delete(m.tokenCache, tokenString)
}

// authorize performs authorization using RBAC.

func (r *RBACManager) authorize(userID, resource, action string) (bool, error) {
	r.mu.RLock()

	defer r.mu.RUnlock()

	user, exists := r.users[userID]

	if !exists {
		return false, fmt.Errorf("user not found: %s", userID)
	}

	// Check user roles and permissions.

	for _, roleName := range user.Roles {
		role, exists := r.roles[roleName]

		if !exists {
			continue
		}

		// Check role permissions.

		for _, permission := range role.Permissions {
			if r.matchesPermission(permission, resource, action) {
				return true, nil
			}
		}

		// Check role policies.

		for _, policyName := range role.Policies {
			policy, exists := r.policies[policyName]

			if !exists {
				continue
			}

			for _, rule := range policy.Rules {
				if r.matchesRule(rule, resource, action) {
					return rule.Effect == "allow", nil
				}
			}
		}
	}

	// Default deny.

	return false, nil
}

// matchesPermission checks if a permission matches the resource and action.

func (r *RBACManager) matchesPermission(permission Permission, resource, action string) bool {
	// Simple wildcard matching.

	if permission.Resource == "*" || permission.Resource == resource {
		for _, allowedAction := range permission.Actions {
			if allowedAction == "*" || allowedAction == action {
				return true
			}
		}
	}

	return false
}

// matchesRule checks if a rule matches the resource and action.

func (r *RBACManager) matchesRule(rule RBACRule, resource, action string) bool {
	// Simple wildcard matching.

	resourceMatch := false

	for _, ruleResource := range rule.Resources {
		if ruleResource == "*" || ruleResource == resource {
			resourceMatch = true

			break
		}
	}

	if !resourceMatch {
		return false
	}

	for _, ruleAction := range rule.Actions {
		if ruleAction == "*" || ruleAction == action {
			return true
		}
	}

	return false
}

// CreateHTTPMiddleware creates HTTP middleware for security.

func (sm *SecurityManager) CreateHTTPMiddleware(providerName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract authorization header.

			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)

				return
			}

			// Check bearer token format.

			parts := strings.SplitN(authHeader, " ", 2)

			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization header format", http.StatusUnauthorized)

				return
			}

			// Validate token.

			tokenInfo, err := sm.ValidateToken(providerName, parts[1])

			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)

				return
			}

			// Add token info to request context.

			ctx := context.WithValue(r.Context(), "tokenInfo", tokenInfo)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}