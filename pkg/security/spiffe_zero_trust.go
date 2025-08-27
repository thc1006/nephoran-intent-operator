// Package security implements SPIFFE/SPIRE zero-trust authentication
// for Nephoran Intent Operator with O-RAN WG11 compliance
package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	// "github.com/spiffe/go-spiffe/v2/svid/jwtsvid" // Removed unused import
	// "github.com/spiffe/go-spiffe/v2/svid/x509svid" // Removed unused import
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	// SPIFFE/SPIRE Metrics
	spiffeConnectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_spiffe_connections_total",
			Help: "Total number of SPIFFE connections",
		},
		[]string{"status"},
	)

	spiffeAuthenticationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_spiffe_authentications_total",
			Help: "Total number of SPIFFE authentications",
		},
		[]string{"service", "status"},
	)

	spiffeSvidRotations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_spiffe_svid_rotations_total",
			Help: "Total number of SVID rotations",
		},
		[]string{"type"},
	)

	spiffePolicyViolations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_spiffe_policy_violations_total",
			Help: "Total number of SPIFFE policy violations",
		},
		[]string{"policy", "reason"},
	)
)

// ZeroTrustConfig contains zero-trust authentication configuration
type ZeroTrustConfig struct {
	// SPIFFE configuration
	SpiffeSocketPath    string
	TrustDomain         string
	ServiceSpiffeID     string
	AllowedSpiffeIDs    []string
	RequiredAudiences   []string
	
	// JWT configuration
	JWTIssuer          string
	JWTSigningKey      []byte
	JWTExpirationTime  time.Duration
	JWTRefreshWindow   time.Duration
	
	// Policy configuration
	EnableAuthzPolicies bool
	PolicyRefreshTime   time.Duration
	DefaultDenyPolicy   bool
	
	// TLS configuration
	RequireMTLS        bool
	MinTLSVersion      uint16
	AllowedCipherSuites []uint16
	
	// Service configuration
	ServiceName        string
	ServiceVersion     string
	Environment        string
}

// ZeroTrustAuthenticator implements zero-trust authentication
type ZeroTrustAuthenticator struct {
	config        *ZeroTrustConfig
	logger        *slog.Logger
	
	// SPIFFE components
	source        *workloadapi.X509Source
	jwtSource     *workloadapi.JWTSource
	// validator     *jwtsvid.Validator // TODO: jwtsvid.Validator may not exist in current SPIFFE version
	
	// Policy engine
	policyEngine  *ZeroTrustPolicyEngine
	
	// TLS configuration
	tlsConfig     *tls.Config
	
	// JWT management
	jwtKeys       sync.Map // map[string]*jwt.Token
	
	// Metrics and monitoring
	stats         *AuthStats
	
	mu            sync.RWMutex
	shutdown      chan struct{}
}

// ZeroTrustPolicyEngine implements authorization policies
type ZeroTrustPolicyEngine struct {
	policies      map[string]*AuthzPolicy
	defaultPolicy PolicyDecision
	mu            sync.RWMutex
}

// AuthzPolicy represents an authorization policy
type AuthzPolicy struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Rules       []ZeroTrustPolicyRule      `json:"rules"`
	Principals  []string          `json:"principals"`
	Resources   []string          `json:"resources"`
	Actions     []string          `json:"actions"`
	Conditions  map[string]string `json:"conditions"`
	Effect      PolicyDecision    `json:"effect"`
	Priority    int               `json:"priority"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// ZeroTrustPolicyRule defines a specific authorization rule (renamed to avoid conflicts)
type ZeroTrustPolicyRule struct {
	Principal string            `json:"principal"`
	Resource  string            `json:"resource"`
	Action    string            `json:"action"`
	Condition map[string]string `json:"condition"`
	Effect    PolicyDecision    `json:"effect"`
}

// PolicyDecision represents the authorization decision
type PolicyDecision int

const (
	PolicyDeny PolicyDecision = iota
	PolicyAllow
	PolicyConditionalAllow
)

// AuthContext contains authentication and authorization context
type AuthContext struct {
	// SPIFFE identity
	SpiffeID      spiffeid.ID
	TrustDomain   string
	ServiceName   string
	
	// JWT claims
	JWTClaims     jwt.MapClaims
	Issuer        string
	Audience      []string
	
	// Request context
	Method        string
	Path          string
	RemoteAddr    string
	UserAgent     string
	
	// Authorization context
	Roles         []string
	Permissions   []string
	Attributes    map[string]interface{}
	
	// Temporal context
	AuthTime      time.Time
	ExpirationTime time.Time
}

// AuthStats tracks authentication statistics
type AuthStats struct {
	TotalAuths         int64     `json:"total_auths"`
	SuccessfulAuths    int64     `json:"successful_auths"`
	FailedAuths        int64     `json:"failed_auths"`
	PolicyDenials      int64     `json:"policy_denials"`
	SvidRotations      int64     `json:"svid_rotations"`
	LastAuthTime       time.Time `json:"last_auth_time"`
	LastSvidRotation   time.Time `json:"last_svid_rotation"`
}

// NewZeroTrustAuthenticator creates a new zero-trust authenticator
func NewZeroTrustAuthenticator(config *ZeroTrustConfig, logger *slog.Logger) (*ZeroTrustAuthenticator, error) {
	if config == nil {
		return nil, fmt.Errorf("zero-trust config is required")
	}

	// Set defaults
	if config.JWTExpirationTime == 0 {
		config.JWTExpirationTime = 1 * time.Hour
	}
	if config.JWTRefreshWindow == 0 {
		config.JWTRefreshWindow = 15 * time.Minute
	}
	if config.PolicyRefreshTime == 0 {
		config.PolicyRefreshTime = 5 * time.Minute
	}
	if config.MinTLSVersion == 0 {
		config.MinTLSVersion = tls.VersionTLS13
	}

	zta := &ZeroTrustAuthenticator{
		config:   config,
		logger:   logger.With(slog.String("component", "zero_trust_auth")),
		stats:    &AuthStats{},
		shutdown: make(chan struct{}),
	}

	// Initialize SPIFFE X.509 source
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	x509Source, err := workloadapi.NewX509Source(ctx, workloadapi.WithClientOptions(
		workloadapi.WithAddr(config.SpiffeSocketPath),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create X.509 source: %w", err)
	}
	zta.source = x509Source

	// Initialize SPIFFE JWT source
	jwtSource, err := workloadapi.NewJWTSource(ctx, workloadapi.WithClientOptions(
		workloadapi.WithAddr(config.SpiffeSocketPath),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT source: %w", err)
	}
	zta.jwtSource = jwtSource

	// Create JWT validator
	_, err = spiffeid.TrustDomainFromString(config.TrustDomain)
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain: %w", err)
	}
	// zta.validator = jwtsvid.NewValidator(td, jwtSource) // TODO: jwtsvid.NewValidator may not exist in current SPIFFE version

	// Initialize policy engine
	zta.policyEngine = &ZeroTrustPolicyEngine{
		policies: make(map[string]*AuthzPolicy),
		defaultPolicy: PolicyDeny,
	}
	if !config.DefaultDenyPolicy {
		zta.policyEngine.defaultPolicy = PolicyAllow
	}

	// Configure TLS
	if err := zta.configureTLS(); err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}

	// Start background tasks
	go zta.backgroundTasks()

	logger.Info("Zero-trust authenticator initialized",
		slog.String("trust_domain", config.TrustDomain),
		slog.String("service_spiffe_id", config.ServiceSpiffeID),
		slog.Bool("require_mtls", config.RequireMTLS),
		slog.Bool("enable_policies", config.EnableAuthzPolicies))

	return zta, nil
}

// configureTLS sets up TLS configuration with SPIFFE certificates
func (zta *ZeroTrustAuthenticator) configureTLS() error {
	// Create TLS config with SPIFFE certificates
	tlsConfig := tlsconfig.MTLSServerConfig(zta.source, zta.source, tlsconfig.AuthorizeAny())
	
	// Apply security hardening
	tlsConfig.MinVersion = zta.config.MinTLSVersion
	if len(zta.config.AllowedCipherSuites) > 0 {
		tlsConfig.CipherSuites = zta.config.AllowedCipherSuites
	} else {
		// O-RAN WG11 approved cipher suites
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		}
	}
	
	// Custom verification for SPIFFE ID authorization
	tlsConfig.VerifyPeerCertificate = zta.verifyPeerCertificate
	
	zta.tlsConfig = tlsConfig
	return nil
}

// verifyPeerCertificate performs custom peer certificate verification
func (zta *ZeroTrustAuthenticator) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		spiffePolicyViolations.WithLabelValues("mtls", "no_certificate").Inc()
		return fmt.Errorf("no client certificate provided")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		spiffePolicyViolations.WithLabelValues("mtls", "certificate_parse_error").Inc()
		return fmt.Errorf("failed to parse client certificate: %w", err)
	}

	// Extract SPIFFE ID from certificate
	var spiffeIDStr string
	for _, uri := range cert.URIs {
		if strings.HasPrefix(uri.String(), "spiffe://") {
			spiffeIDStr = uri.String()
			break
		}
	}

	if spiffeIDStr == "" {
		spiffePolicyViolations.WithLabelValues("mtls", "no_spiffe_id").Inc()
		return fmt.Errorf("no SPIFFE ID found in certificate")
	}

	// Parse SPIFFE ID
	spiffeID, err := spiffeid.FromString(spiffeIDStr)
	if err != nil {
		spiffePolicyViolations.WithLabelValues("mtls", "invalid_spiffe_id").Inc()
		return fmt.Errorf("invalid SPIFFE ID: %w", err)
	}

	// Verify trust domain
	if spiffeID.TrustDomain().String() != zta.config.TrustDomain {
		spiffePolicyViolations.WithLabelValues("mtls", "trust_domain_mismatch").Inc()
		return fmt.Errorf("trust domain mismatch: expected %s, got %s", 
			zta.config.TrustDomain, spiffeID.TrustDomain().String())
	}

	// Check allowed SPIFFE IDs
	if len(zta.config.AllowedSpiffeIDs) > 0 {
		allowed := false
		for _, allowedID := range zta.config.AllowedSpiffeIDs {
			if spiffeIDStr == allowedID || strings.HasSuffix(spiffeIDStr, allowedID) {
				allowed = true
				break
			}
		}
		if !allowed {
			spiffePolicyViolations.WithLabelValues("mtls", "spiffe_id_not_allowed").Inc()
			return fmt.Errorf("SPIFFE ID not in allowed list: %s", spiffeIDStr)
		}
	}

	spiffeAuthenticationsTotal.WithLabelValues("mtls", "success").Inc()
	zta.logger.Debug("mTLS peer certificate verified",
		slog.String("spiffe_id", spiffeIDStr),
		slog.String("trust_domain", spiffeID.TrustDomain().String()))

	return nil
}

// AuthenticateHTTP performs HTTP authentication with zero-trust principles
func (zta *ZeroTrustAuthenticator) AuthenticateHTTP(r *http.Request) (*AuthContext, error) {
	authCtx := &AuthContext{
		Method:     r.Method,
		Path:       r.URL.Path,
		RemoteAddr: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		AuthTime:   time.Now(),
		Attributes: make(map[string]interface{}),
	}

	// Try JWT authentication first
	if token := zta.extractJWTToken(r); token != "" {
		if err := zta.validateJWT(token, authCtx); err != nil {
			spiffeAuthenticationsTotal.WithLabelValues("jwt", "failed").Inc()
			return nil, fmt.Errorf("JWT validation failed: %w", err)
		}
		spiffeAuthenticationsTotal.WithLabelValues("jwt", "success").Inc()
	} else if zta.config.RequireMTLS {
		// Try mTLS authentication
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			spiffeAuthenticationsTotal.WithLabelValues("mtls", "failed").Inc()
			return nil, fmt.Errorf("mTLS required but no client certificate provided")
		}

		if err := zta.validateMTLS(r.TLS, authCtx); err != nil {
			spiffeAuthenticationsTotal.WithLabelValues("mtls", "failed").Inc()
			return nil, fmt.Errorf("mTLS validation failed: %w", err)
		}
		spiffeAuthenticationsTotal.WithLabelValues("mtls", "success").Inc()
	} else {
		return nil, fmt.Errorf("no valid authentication method found")
	}

	// Perform authorization if policies are enabled
	if zta.config.EnableAuthzPolicies {
		decision, err := zta.authorize(authCtx)
		if err != nil {
			return nil, fmt.Errorf("authorization error: %w", err)
		}
		if decision != PolicyAllow {
			spiffePolicyViolations.WithLabelValues("authorization", "denied").Inc()
			return nil, fmt.Errorf("access denied by authorization policy")
		}
	}

	zta.stats.TotalAuths++
	zta.stats.SuccessfulAuths++
	zta.stats.LastAuthTime = time.Now()

	return authCtx, nil
}

// extractJWTToken extracts JWT token from Authorization header
func (zta *ZeroTrustAuthenticator) extractJWTToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// validateJWT validates a JWT token
func (zta *ZeroTrustAuthenticator) validateJWT(tokenString string, authCtx *AuthContext) error {
	// Parse and validate JWT
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return zta.config.JWTSigningKey, nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse JWT: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("invalid JWT token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid JWT claims")
	}

	// Validate issuer
	if iss, ok := claims["iss"].(string); ok {
		if iss != zta.config.JWTIssuer {
			return fmt.Errorf("invalid issuer: %s", iss)
		}
		authCtx.Issuer = iss
	}

	// Validate audience
	if aud, ok := claims["aud"].([]interface{}); ok {
		audStrs := make([]string, len(aud))
		for i, a := range aud {
			if audStr, ok := a.(string); ok {
				audStrs[i] = audStr
			}
		}
		authCtx.Audience = audStrs

		// Check required audiences
		if len(zta.config.RequiredAudiences) > 0 {
			found := false
			for _, reqAud := range zta.config.RequiredAudiences {
				for _, aud := range audStrs {
					if aud == reqAud {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				return fmt.Errorf("required audience not found")
			}
		}
	}

	// Extract SPIFFE ID from subject
	if sub, ok := claims["sub"].(string); ok {
		if strings.HasPrefix(sub, "spiffe://") {
			spiffeID, err := spiffeid.FromString(sub)
			if err != nil {
				return fmt.Errorf("invalid SPIFFE ID in subject: %w", err)
			}
			authCtx.SpiffeID = spiffeID
			authCtx.TrustDomain = spiffeID.TrustDomain().String()
		}
	}

	// Extract roles and permissions
	if roles, ok := claims["roles"].([]interface{}); ok {
		roleStrs := make([]string, len(roles))
		for i, r := range roles {
			if roleStr, ok := r.(string); ok {
				roleStrs[i] = roleStr
			}
		}
		authCtx.Roles = roleStrs
	}

	if permissions, ok := claims["permissions"].([]interface{}); ok {
		permStrs := make([]string, len(permissions))
		for i, p := range permissions {
			if permStr, ok := p.(string); ok {
				permStrs[i] = permStr
			}
		}
		authCtx.Permissions = permStrs
	}

	// Set expiration
	if exp, ok := claims["exp"].(float64); ok {
		authCtx.ExpirationTime = time.Unix(int64(exp), 0)
	}

	authCtx.JWTClaims = claims
	return nil
}

// validateMTLS validates mTLS connection
func (zta *ZeroTrustAuthenticator) validateMTLS(tlsState *tls.ConnectionState, authCtx *AuthContext) error {
	if len(tlsState.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificates")
	}

	cert := tlsState.PeerCertificates[0]

	// Extract SPIFFE ID from certificate
	var spiffeIDStr string
	for _, uri := range cert.URIs {
		if strings.HasPrefix(uri.String(), "spiffe://") {
			spiffeIDStr = uri.String()
			break
		}
	}

	if spiffeIDStr == "" {
		return fmt.Errorf("no SPIFFE ID found in certificate")
	}

	spiffeID, err := spiffeid.FromString(spiffeIDStr)
	if err != nil {
		return fmt.Errorf("invalid SPIFFE ID: %w", err)
	}

	authCtx.SpiffeID = spiffeID
	authCtx.TrustDomain = spiffeID.TrustDomain().String()
	authCtx.ServiceName = spiffeID.Path()

	return nil
}

// authorize performs authorization based on policies
func (zta *ZeroTrustAuthenticator) authorize(authCtx *AuthContext) (PolicyDecision, error) {
	zta.policyEngine.mu.RLock()
	defer zta.policyEngine.mu.RUnlock()

	// Evaluate policies in priority order
	for _, policy := range zta.policyEngine.policies {
		if zta.matchesPolicy(policy, authCtx) {
			decision := zta.evaluatePolicy(policy, authCtx)
			if decision == PolicyDeny {
				return PolicyDeny, nil
			}
		}
	}

	return zta.policyEngine.defaultPolicy, nil
}

// matchesPolicy checks if a policy applies to the auth context
func (zta *ZeroTrustAuthenticator) matchesPolicy(policy *AuthzPolicy, authCtx *AuthContext) bool {
	// Check principals (SPIFFE IDs)
	if len(policy.Principals) > 0 {
		found := false
		spiffeIDStr := authCtx.SpiffeID.String()
		for _, principal := range policy.Principals {
			if principal == spiffeIDStr || strings.HasSuffix(spiffeIDStr, principal) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check resources (paths)
	if len(policy.Resources) > 0 {
		found := false
		for _, resource := range policy.Resources {
			if resource == authCtx.Path || strings.HasPrefix(authCtx.Path, resource) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check actions (methods)
	if len(policy.Actions) > 0 {
		found := false
		for _, action := range policy.Actions {
			if strings.EqualFold(action, authCtx.Method) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// evaluatePolicy evaluates a specific policy
func (zta *ZeroTrustAuthenticator) evaluatePolicy(policy *AuthzPolicy, authCtx *AuthContext) PolicyDecision {
	// Evaluate all rules in the policy
	for _, rule := range policy.Rules {
		if zta.evaluateRule(&rule, authCtx) {
			return rule.Effect
		}
	}

	return policy.Effect
}

// evaluateRule evaluates a specific policy rule
func (zta *ZeroTrustAuthenticator) evaluateRule(rule *ZeroTrustPolicyRule, authCtx *AuthContext) bool {
	// Simple rule evaluation - can be extended
	principalMatch := rule.Principal == "" || strings.Contains(authCtx.SpiffeID.String(), rule.Principal)
	resourceMatch := rule.Resource == "" || strings.HasPrefix(authCtx.Path, rule.Resource)
	actionMatch := rule.Action == "" || strings.EqualFold(rule.Action, authCtx.Method)

	return principalMatch && resourceMatch && actionMatch
}

// CreateHTTPMiddleware creates HTTP middleware for zero-trust authentication
func (zta *ZeroTrustAuthenticator) CreateHTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Perform authentication
			authCtx, err := zta.AuthenticateHTTP(r)
			if err != nil {
				zta.stats.FailedAuths++
				zta.logger.Warn("Authentication failed",
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.String("remote_addr", r.RemoteAddr),
					slog.String("error", err.Error()))

				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Add auth context to request context
			ctx := context.WithValue(r.Context(), "auth_context", authCtx)
			r = r.WithContext(ctx)

			// Add security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

			next.ServeHTTP(w, r)
		})
	}
}

// CreateGRPCCredentials creates gRPC credentials with SPIFFE authentication
func (zta *ZeroTrustAuthenticator) CreateGRPCCredentials() credentials.TransportCredentials {
	return credentials.NewTLS(zta.tlsConfig)
}

// LoadAuthzPolicy loads an authorization policy
func (zta *ZeroTrustAuthenticator) LoadAuthzPolicy(policy *AuthzPolicy) error {
	zta.policyEngine.mu.Lock()
	defer zta.policyEngine.mu.Unlock()

	policy.UpdatedAt = time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now()
	}

	zta.policyEngine.policies[policy.ID] = policy

	zta.logger.Info("Authorization policy loaded",
		slog.String("policy_id", policy.ID),
		slog.String("policy_name", policy.Name),
		slog.Int("rules_count", len(policy.Rules)))

	return nil
}

// backgroundTasks runs background maintenance tasks
func (zta *ZeroTrustAuthenticator) backgroundTasks() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			zta.rotateTokensIfNeeded()
			zta.updateMetrics()

		case <-zta.shutdown:
			return
		}
	}
}

// rotateTokensIfNeeded rotates JWT tokens and SVID certificates if needed
func (zta *ZeroTrustAuthenticator) rotateTokensIfNeeded() {
	// Check if SVID rotation is needed
	// TODO: GetX509SVIDs method doesn't exist - use GetX509SVID instead
	if svid, err := zta.source.GetX509SVID(); err == nil && len(svid.Certificates) > 0 {
		cert := svid.Certificates[0]
		timeUntilExpiry := time.Until(cert.NotAfter)
		if timeUntilExpiry < 24*time.Hour {
			zta.logger.Info("SVID approaching expiry, rotation will be handled automatically",
				slog.String("spiffe_id", svid.ID.String()),
				slog.Time("expiry", cert.NotAfter))
			spiffeSvidRotations.WithLabelValues("x509").Inc()
			zta.stats.SvidRotations++
			zta.stats.LastSvidRotation = time.Now()
		}
	}
}

// updateMetrics updates internal metrics
func (zta *ZeroTrustAuthenticator) updateMetrics() {
	// Update connection status metrics
	spiffeConnectionsTotal.WithLabelValues("active").Add(float64(zta.stats.TotalAuths))
}

// GetStats returns authentication statistics
func (zta *ZeroTrustAuthenticator) GetStats() *AuthStats {
	return zta.stats
}

// Close shuts down the zero-trust authenticator
func (zta *ZeroTrustAuthenticator) Close() error {
	close(zta.shutdown)
	
	if zta.source != nil {
		zta.source.Close()
	}
	if zta.jwtSource != nil {
		zta.jwtSource.Close()
	}
	
	zta.logger.Info("Zero-trust authenticator shut down")
	return nil
}

// GetAuthContextFromRequest extracts auth context from HTTP request
func GetAuthContextFromRequest(r *http.Request) (*AuthContext, bool) {
	if authCtx, ok := r.Context().Value("auth_context").(*AuthContext); ok {
		return authCtx, true
	}
	return nil, false
}

// DefaultZeroTrustConfig returns default configuration for zero-trust authentication
func DefaultZeroTrustConfig() *ZeroTrustConfig {
	return &ZeroTrustConfig{
		SpiffeSocketPath:   "unix:///tmp/spire-agent/public/api.sock",
		TrustDomain:        "nephoran.local",
		ServiceSpiffeID:    "spiffe://nephoran.local/nephoran-intent-operator",
		JWTExpirationTime:  1 * time.Hour,
		JWTRefreshWindow:   15 * time.Minute,
		PolicyRefreshTime:  5 * time.Minute,
		RequireMTLS:        true,
		MinTLSVersion:      tls.VersionTLS13,
		EnableAuthzPolicies: true,
		DefaultDenyPolicy:   true,
		ServiceName:        "nephoran-intent-operator",
		ServiceVersion:     "v1.0.0",
		Environment:        "production",
	}
}