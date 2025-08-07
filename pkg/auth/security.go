package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PKCEManager manages PKCE (Proof Key for Code Exchange) challenges
type PKCEManager struct {
	challenges map[string]*PKCEChallenge
	mu         sync.RWMutex
	ttl        time.Duration
}

// PKCEChallenge represents a PKCE challenge
type PKCEChallenge struct {
	CodeVerifier  string    `json:"code_verifier"`
	CodeChallenge string    `json:"code_challenge"`
	Method        string    `json:"code_challenge_method"`
	State         string    `json:"state"`
	CreatedAt     time.Time `json:"created_at"`
	ClientID      string    `json:"client_id"`
	RedirectURI   string    `json:"redirect_uri"`
}

// CSRFManager manages CSRF (Cross-Site Request Forgery) tokens
type CSRFManager struct {
	tokens map[string]*CSRFToken
	mu     sync.RWMutex
	ttl    time.Duration
	secret []byte
}

// CSRFToken represents a CSRF token
type CSRFToken struct {
	Token     string    `json:"token"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
}

// SecurityManager provides comprehensive security features
type SecurityManager struct {
	pkceManager *PKCEManager
	csrfManager *CSRFManager
}

// NewPKCEManager creates a new PKCE manager
func NewPKCEManager(ttl time.Duration) *PKCEManager {
	if ttl == 0 {
		ttl = 10 * time.Minute // Default PKCE challenge TTL
	}

	pm := &PKCEManager{
		challenges: make(map[string]*PKCEChallenge),
		ttl:        ttl,
	}

	// Start cleanup goroutine
	go pm.cleanup()

	return pm
}

// GenerateChallenge generates a new PKCE challenge
func (pm *PKCEManager) GenerateChallenge(state, clientID, redirectURI string) (*PKCEChallenge, error) {
	// Generate code verifier (43-128 characters, base64url-encoded)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate code challenge using S256 method
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	challenge := &PKCEChallenge{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
		Method:        "S256",
		State:         state,
		CreatedAt:     time.Now(),
		ClientID:      clientID,
		RedirectURI:   redirectURI,
	}

	pm.mu.Lock()
	pm.challenges[state] = challenge
	pm.mu.Unlock()

	return challenge, nil
}

// ValidateChallenge validates a PKCE challenge
func (pm *PKCEManager) ValidateChallenge(state, codeVerifier, clientID, redirectURI string) (*PKCEChallenge, error) {
	pm.mu.RLock()
	challenge, exists := pm.challenges[state]
	pm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("PKCE challenge not found or expired")
	}

	// Check expiration
	if time.Since(challenge.CreatedAt) > pm.ttl {
		pm.mu.Lock()
		delete(pm.challenges, state)
		pm.mu.Unlock()
		return nil, fmt.Errorf("PKCE challenge expired")
	}

	// Validate client ID and redirect URI
	if challenge.ClientID != clientID {
		return nil, fmt.Errorf("client ID mismatch")
	}
	if challenge.RedirectURI != redirectURI {
		return nil, fmt.Errorf("redirect URI mismatch")
	}

	// Validate code verifier
	hash := sha256.Sum256([]byte(codeVerifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	if challenge.CodeChallenge != expectedChallenge {
		return nil, fmt.Errorf("code verifier validation failed")
	}

	// Remove used challenge
	pm.mu.Lock()
	delete(pm.challenges, state)
	pm.mu.Unlock()

	return challenge, nil
}

// GetChallenge retrieves a PKCE challenge by state
func (pm *PKCEManager) GetChallenge(state string) (*PKCEChallenge, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	challenge, exists := pm.challenges[state]
	return challenge, exists
}

// cleanup removes expired PKCE challenges
func (pm *PKCEManager) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pm.mu.Lock()
		now := time.Now()
		for state, challenge := range pm.challenges {
			if now.Sub(challenge.CreatedAt) > pm.ttl {
				delete(pm.challenges, state)
			}
		}
		pm.mu.Unlock()
	}
}

// NewCSRFManager creates a new CSRF manager
func NewCSRFManager(secret []byte, ttl time.Duration) *CSRFManager {
	if ttl == 0 {
		ttl = 1 * time.Hour // Default CSRF token TTL
	}

	if len(secret) == 0 {
		// Generate a random secret if none provided
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			panic(fmt.Sprintf("failed to generate CSRF secret: %v", err))
		}
	}

	cm := &CSRFManager{
		tokens: make(map[string]*CSRFToken),
		ttl:    ttl,
		secret: secret,
	}

	// Start cleanup goroutine
	go cm.cleanup()

	return cm
}

// GenerateToken generates a new CSRF token for a session
func (cm *CSRFManager) GenerateToken(sessionID string) (string, error) {
	// Generate random token data
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	// Create token with session binding
	tokenData := append(tokenBytes, []byte(sessionID)...)
	hash := sha256.Sum256(append(tokenData, cm.secret...))
	token := hex.EncodeToString(hash[:])

	csrfToken := &CSRFToken{
		Token:     token,
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}

	cm.mu.Lock()
	cm.tokens[token] = csrfToken
	cm.mu.Unlock()

	return token, nil
}

// ValidateToken validates a CSRF token for a session
func (cm *CSRFManager) ValidateToken(token, sessionID string) error {
	if token == "" {
		return fmt.Errorf("CSRF token is required")
	}

	cm.mu.RLock()
	csrfToken, exists := cm.tokens[token]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("CSRF token not found or expired")
	}

	// Check expiration
	if time.Since(csrfToken.CreatedAt) > cm.ttl {
		cm.mu.Lock()
		delete(cm.tokens, token)
		cm.mu.Unlock()
		return fmt.Errorf("CSRF token expired")
	}

	// Validate session binding
	if csrfToken.SessionID != sessionID {
		return fmt.Errorf("CSRF token session mismatch")
	}

	// Mark token as used (optional - for single-use tokens)
	now := time.Now()
	cm.mu.Lock()
	csrfToken.UsedAt = &now
	cm.mu.Unlock()

	return nil
}

// InvalidateToken invalidates a CSRF token
func (cm *CSRFManager) InvalidateToken(token string) {
	cm.mu.Lock()
	delete(cm.tokens, token)
	cm.mu.Unlock()
}

// InvalidateSession invalidates all CSRF tokens for a session
func (cm *CSRFManager) InvalidateSession(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for token, csrfToken := range cm.tokens {
		if csrfToken.SessionID == sessionID {
			delete(cm.tokens, token)
		}
	}
}

// cleanup removes expired CSRF tokens
func (cm *CSRFManager) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cm.mu.Lock()
		now := time.Now()
		for token, csrfToken := range cm.tokens {
			if now.Sub(csrfToken.CreatedAt) > cm.ttl {
				delete(cm.tokens, token)
			}
		}
		cm.mu.Unlock()
	}
}

// NewSecurityManager creates a new security manager with PKCE and CSRF protection
func NewSecurityManager(csrfSecret []byte) *SecurityManager {
	return &SecurityManager{
		pkceManager: NewPKCEManager(10 * time.Minute),
		csrfManager: NewCSRFManager(csrfSecret, 1*time.Hour),
	}
}

// GetPKCEManager returns the PKCE manager
func (sm *SecurityManager) GetPKCEManager() *PKCEManager {
	return sm.pkceManager
}

// GetCSRFManager returns the CSRF manager
func (sm *SecurityManager) GetCSRFManager() *CSRFManager {
	return sm.csrfManager
}

// Enhanced CSRF middleware with double-submit cookie pattern
type CSRFMiddleware struct {
	csrfManager *CSRFManager
	config      *CSRFConfig
}

// CSRFConfig represents CSRF middleware configuration
type CSRFConfig struct {
	TokenHeader    string        `json:"token_header"`
	CookieName     string        `json:"cookie_name"`
	SafeMethods    []string      `json:"safe_methods"`
	RequireHTTPS   bool          `json:"require_https"`
	CookieSecure   bool          `json:"cookie_secure"`
	CookieHTTPOnly bool          `json:"cookie_http_only"`
	CookieSameSite http.SameSite `json:"cookie_same_site"`
	MaxAge         int           `json:"max_age"`
}

// NewCSRFMiddleware creates new CSRF middleware
func NewCSRFMiddleware(csrfManager *CSRFManager, config *CSRFConfig) *CSRFMiddleware {
	if config == nil {
		config = &CSRFConfig{
			TokenHeader:    "X-CSRF-Token",
			CookieName:     "csrf_token",
			SafeMethods:    []string{"GET", "HEAD", "OPTIONS", "TRACE"},
			RequireHTTPS:   true,
			CookieSecure:   true,
			CookieHTTPOnly: false, // Must be false to access via JavaScript
			CookieSameSite: http.SameSiteLaxMode,
			MaxAge:         3600, // 1 hour
		}
	}

	return &CSRFMiddleware{
		csrfManager: csrfManager,
		config:      config,
	}
}

// Middleware returns HTTP middleware for CSRF protection
func (cm *CSRFMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods
		if cm.isSafeMethod(r.Method) {
			// Set CSRF token cookie for safe methods
			cm.setCsrfCookie(w, r)
			next.ServeHTTP(w, r)
			return
		}

		// Require HTTPS in production
		if cm.config.RequireHTTPS && r.TLS == nil && !cm.isLocalhost(r) {
			http.Error(w, "HTTPS required", http.StatusUpgradeRequired)
			return
		}

		// Get session ID
		sessionID := cm.getSessionID(r)
		if sessionID == "" {
			http.Error(w, "Session required for CSRF protection", http.StatusForbidden)
			return
		}

		// Get CSRF token from header or form
		token := r.Header.Get(cm.config.TokenHeader)
		if token == "" {
			token = r.FormValue("csrf_token")
		}

		// Validate CSRF token
		if err := cm.csrfManager.ValidateToken(token, sessionID); err != nil {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Set new CSRF token cookie
		cm.setCsrfCookie(w, r)
		next.ServeHTTP(w, r)
	})
}

// GetToken returns a CSRF token for the current session
func (cm *CSRFMiddleware) GetToken(w http.ResponseWriter, r *http.Request) (string, error) {
	sessionID := cm.getSessionID(r)
	if sessionID == "" {
		return "", fmt.Errorf("no session found")
	}

	token, err := cm.csrfManager.GenerateToken(sessionID)
	if err != nil {
		return "", err
	}

	// Set cookie with token
	http.SetCookie(w, &http.Cookie{
		Name:     cm.config.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cm.config.MaxAge,
		Secure:   cm.config.CookieSecure,
		HttpOnly: cm.config.CookieHTTPOnly,
		SameSite: cm.config.CookieSameSite,
	})

	return token, nil
}

// Private helper methods

func (cm *CSRFMiddleware) isSafeMethod(method string) bool {
	for _, safeMethod := range cm.config.SafeMethods {
		if method == safeMethod {
			return true
		}
	}
	return false
}

func (cm *CSRFMiddleware) isLocalhost(r *http.Request) bool {
	host := r.Host
	if strings.Contains(host, ":") {
		host, _, _ = strings.Cut(host, ":")
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (cm *CSRFMiddleware) getSessionID(r *http.Request) string {
	// Try cookie first
	cookie, err := r.Cookie("nephoran_session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Try header
	return r.Header.Get("X-Session-ID")
}

func (cm *CSRFMiddleware) setCsrfCookie(w http.ResponseWriter, r *http.Request) {
	sessionID := cm.getSessionID(r)
	if sessionID == "" {
		return
	}

	// Check if CSRF token cookie already exists and is valid
	if cookie, err := r.Cookie(cm.config.CookieName); err == nil && cookie.Value != "" {
		if err := cm.csrfManager.ValidateToken(cookie.Value, sessionID); err == nil {
			return // Valid token exists, no need to set new one
		}
	}

	// Generate new CSRF token
	token, err := cm.csrfManager.GenerateToken(sessionID)
	if err != nil {
		return
	}

	// Set new cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cm.config.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cm.config.MaxAge,
		Secure:   cm.config.CookieSecure,
		HttpOnly: cm.config.CookieHTTPOnly,
		SameSite: cm.config.CookieSameSite,
	})
}

// Utility functions for generating secure random values

// GenerateState generates a secure random state parameter
func GenerateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateNonce generates a secure random nonce
func GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateState validates that state parameter is properly formatted
func ValidateState(state string) error {
	if state == "" {
		return fmt.Errorf("state parameter is required")
	}
	
	// Decode to check if it's valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(state); err != nil {
		return fmt.Errorf("invalid state format: %w", err)
	}
	
	// Check minimum length (32 bytes = 43 base64url chars)
	if len(state) < 43 {
		return fmt.Errorf("state parameter too short")
	}
	
	return nil
}

// HashToken creates a secure hash of a token for storage
func HashToken(token string, salt []byte) string {
	data := append([]byte(token), salt...)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SecureCompare performs constant-time comparison of two strings
func SecureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtleCompareBytes([]byte(a), []byte(b))
}

// subtleCompareBytes performs constant-time comparison of byte slices
func subtleCompareBytes(a, b []byte) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare(a, b) == 1
}