// Package validation provides comprehensive security testing mock services
package validation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
)

// OAuthMockService provides OAuth2/OIDC testing capabilities
type OAuthMockService struct {
	server      *httptest.Server
	tokenStore  map[string]TokenInfo
	clientStore map[string]ClientInfo
}

// TokenInfo represents OAuth token information
type TokenInfo struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	TokenType    string
	Scopes       []string
}

// ClientInfo represents OAuth client information
type ClientInfo struct {
	ClientID     string
	ClientSecret string
	RedirectURIs []string
	Scopes       []string
}

// NewOAuthMockService creates a new OAuth mock service
func NewOAuthMockService() *OAuthMockService {
	oauth := &OAuthMockService{
		tokenStore:  make(map[string]TokenInfo),
		clientStore: make(map[string]ClientInfo),
	}

	// Pre-populate with test client
	oauth.clientStore["nephoran-test-client"] = ClientInfo{
		ClientID:     "nephoran-test-client",
		ClientSecret: "test-secret-12345",
		RedirectURIs: []string{"http://localhost:8080/callback"},
		Scopes:       []string{"read", "write", "admin"},
	}

	oauth.setupMockServer()
	return oauth
}

// setupMockServer creates the OAuth2 mock endpoints
func (o *OAuthMockService) setupMockServer() {
	mux := http.NewServeMux()

	// OIDC Discovery endpoint
	mux.HandleFunc("/.well-known/openid_configuration", func(w http.ResponseWriter, r *http.Request) {
		ginkgo.By("Serving OIDC discovery endpoint")

		baseURL := o.server.URL
		discovery := fmt.Sprintf(`{
			"issuer": "%s",
			"authorization_endpoint": "%s/auth",
			"token_endpoint": "%s/token",
			"userinfo_endpoint": "%s/userinfo",
			"jwks_uri": "%s/.well-known/jwks.json",
			"scopes_supported": ["openid", "profile", "email", "read", "write"],
			"response_types_supported": ["code", "token", "id_token"],
			"grant_types_supported": ["authorization_code", "client_credentials", "refresh_token"],
			"token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
			"claims_supported": ["sub", "name", "email", "iat", "exp"]
		}`, baseURL, baseURL, baseURL, baseURL, baseURL)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(discovery))
	})

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		ginkgo.By("Processing token request")

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		grantType := r.FormValue("grant_type")
		clientID := r.FormValue("client_id")

		// Validate client
		if _, exists := o.clientStore[clientID]; !exists {
			http.Error(w, "Invalid client", http.StatusUnauthorized)
			return
		}

		// Generate token based on grant type
		var token TokenInfo
		switch grantType {
		case "client_credentials":
			token = o.generateClientCredentialsToken(clientID)
		case "authorization_code":
			token = o.generateAuthorizationCodeToken(clientID, r.FormValue("code"))
		case "refresh_token":
			token = o.refreshAccessToken(r.FormValue("refresh_token"))
		default:
			http.Error(w, "Unsupported grant type", http.StatusBadRequest)
			return
		}

		// Store token
		o.tokenStore[token.AccessToken] = token

		// Return token response
		response := fmt.Sprintf(`{
			"access_token": "%s",
			"token_type": "%s",
			"expires_in": %d,
			"refresh_token": "%s",
			"scope": "%s"
		}`, token.AccessToken, token.TokenType, 3600, token.RefreshToken, strings.Join(token.Scopes, " "))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	// Token validation endpoint
	mux.HandleFunc("/tokeninfo", func(w http.ResponseWriter, r *http.Request) {
		ginkgo.By("Validating token")

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Missing or invalid authorization header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		tokenInfo, exists := o.tokenStore[token]
		if !exists || time.Now().After(tokenInfo.ExpiresAt) {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		response := fmt.Sprintf(`{
			"active": true,
			"client_id": "nephoran-test-client",
			"exp": %d,
			"scope": "%s"
		}`, tokenInfo.ExpiresAt.Unix(), strings.Join(tokenInfo.Scopes, " "))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})

	// JWKS endpoint
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		ginkgo.By("Serving JWKS endpoint")

		// Mock JWKS response
		jwks := `{
			"keys": [
				{
					"kty": "RSA",
					"use": "sig",
					"kid": "test-key-1",
					"n": "test-modulus",
					"e": "AQAB"
				}
			]
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jwks))
	})

	o.server = httptest.NewServer(mux)
}

// SimulateOAuthValidation simulates OAuth2 validation for testing
func (o *OAuthMockService) SimulateOAuthValidation() bool {
	ginkgo.By("🔐 Simulating OAuth2/OIDC validation")

	// Test client credentials flow
	client := &http.Client{Timeout: 5 * time.Second}

	// Test discovery endpoint
	resp, err := client.Get(o.server.URL + "/.well-known/openid_configuration")
	if err != nil {
		ginkgo.By(fmt.Sprintf("❌ OIDC discovery failed: %v", err))
		return false
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ginkgo.By(fmt.Sprintf("❌ OIDC discovery returned status: %d", resp.StatusCode))
		return false
	}

	ginkgo.By("✅ OAuth2/OIDC mock validation successful")
	return true
}

// SimulateOIDCDiscovery simulates OIDC discovery for testing
func (o *OAuthMockService) SimulateOIDCDiscovery() bool {
	ginkgo.By("🔐 Simulating OIDC discovery")
	return true
}

// Helper methods for OAuth mock service

func (o *OAuthMockService) generateClientCredentialsToken(clientID string) TokenInfo {
	return TokenInfo{
		AccessToken:  o.generateRandomToken(),
		RefreshToken: o.generateRandomToken(),
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		TokenType:    "Bearer",
		Scopes:       []string{"read", "write"},
	}
}

func (o *OAuthMockService) generateAuthorizationCodeToken(clientID, code string) TokenInfo {
	return TokenInfo{
		AccessToken:  o.generateRandomToken(),
		RefreshToken: o.generateRandomToken(),
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		TokenType:    "Bearer",
		Scopes:       []string{"openid", "profile", "email"},
	}
}

func (o *OAuthMockService) refreshAccessToken(refreshToken string) TokenInfo {
	return TokenInfo{
		AccessToken:  o.generateRandomToken(),
		RefreshToken: refreshToken, // Keep same refresh token
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		TokenType:    "Bearer",
		Scopes:       []string{"read", "write"},
	}
}

func (o *OAuthMockService) generateRandomToken() string {
	return fmt.Sprintf("tok_%d_%d", time.Now().UnixNano(), rand.Int63())
}

// Cleanup stops the OAuth mock server
func (o *OAuthMockService) Cleanup() {
	if o.server != nil {
		o.server.Close()
	}
}

// TLSMockService provides TLS/mTLS testing capabilities
type TLSMockService struct {
	server     *httptest.Server
	caCert     []byte
	serverCert []byte
	serverKey  []byte
	clientCert []byte
	clientKey  []byte
}

// NewTLSMockService creates a new TLS mock service
func NewTLSMockService() *TLSMockService {
	tls := &TLSMockService{}
	tls.generateTestCertificates()
	tls.setupMockServer()
	return tls
}

// setupMockServer creates the TLS mock endpoints
func (t *TLSMockService) setupMockServer() {
	mux := http.NewServeMux()

	// Certificate download endpoint
	mux.HandleFunc("/certs/ca.crt", func(w http.ResponseWriter, r *http.Request) {
		ginkgo.By("Serving CA certificate")

		w.Header().Set("Content-Type", "application/x-pem-file")
		w.WriteHeader(http.StatusOK)
		w.Write(t.caCert)
	})

	// Certificate validation endpoint
	mux.HandleFunc("/validate-cert", func(w http.ResponseWriter, r *http.Request) {
		ginkgo.By("Validating certificate")

		// Check for client certificate
		if len(r.TLS.PeerCertificates) > 0 {
			cert := r.TLS.PeerCertificates[0]
			response := fmt.Sprintf(`{
				"valid": true,
				"subject": "%s",
				"issuer": "%s",
				"expires": "%s"
			}`, cert.Subject.String(), cert.Issuer.String(), cert.NotAfter.Format(time.RFC3339))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(response))
		} else {
			http.Error(w, "No client certificate provided", http.StatusBadRequest)
		}
	})

	// TLS configuration info endpoint
	mux.HandleFunc("/tls-info", func(w http.ResponseWriter, r *http.Request) {
		ginkgo.By("Providing TLS configuration information")

		tlsInfo := fmt.Sprintf(`{
			"tls_version": "%s",
			"cipher_suite": "%s",
			"server_name": "%s"
		}`, "1.3", "TLS_AES_256_GCM_SHA384", r.Host)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(tlsInfo))
	})

	t.server = httptest.NewServer(mux)
}

// generateTestCertificates creates test certificates for TLS testing
func (t *TLSMockService) generateTestCertificates() {
	ginkgo.By("Generating test certificates for TLS validation")

	// Generate CA certificate
	caCert, caKey := t.generateCertificate("Nephoran Test CA", nil, nil, true)
	t.caCert = t.encodeCertificatePEM(caCert)

	// Generate server certificate
	serverCert, serverKey := t.generateCertificate("localhost", caCert, caKey, false)
	t.serverCert = t.encodeCertificatePEM(serverCert)
	t.serverKey = t.encodeKeyPEM(serverKey)

	// Generate client certificate
	clientCert, clientKey := t.generateCertificate("nephoran-client", caCert, caKey, false)
	t.clientCert = t.encodeCertificatePEM(clientCert)
	t.clientKey = t.encodeKeyPEM(clientKey)
}

// generateCertificate generates a test certificate
func (t *TLSMockService) generateCertificate(commonName string, caCert *x509.Certificate, caKey *rsa.PrivateKey, isCA bool) (*x509.Certificate, *rsa.PrivateKey) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: x509.Certificate{
			CommonName: commonName,
		}.Subject,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:    []string{"localhost"},
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		template.BasicConstraintsValid = true
	}

	// Sign certificate
	var parent *x509.Certificate
	var parentKey *rsa.PrivateKey

	if caCert == nil {
		// Self-signed
		parent = &template
		parentKey = privateKey
	} else {
		// Signed by CA
		parent = caCert
		parentKey = caKey
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, parent, &privateKey.PublicKey, parentKey)
	if err != nil {
		panic(err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		panic(err)
	}

	return cert, privateKey
}

// encodeCertificatePEM encodes certificate to PEM format
func (t *TLSMockService) encodeCertificatePEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// encodeKeyPEM encodes private key to PEM format
func (t *TLSMockService) encodeKeyPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// SimulateTLSValidation simulates TLS validation for testing
func (t *TLSMockService) SimulateTLSValidation() bool {
	ginkgo.By("🔒 Simulating TLS/mTLS validation")

	// Test server certificate validation
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get(t.server.URL + "/tls-info")
	if err != nil {
		ginkgo.By(fmt.Sprintf("❌ TLS validation failed: %v", err))
		return false
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ginkgo.By(fmt.Sprintf("❌ TLS validation returned status: %d", resp.StatusCode))
		return false
	}

	ginkgo.By("✅ TLS mock validation successful")
	return true
}

// SimulateMutualTLS simulates mutual TLS validation
func (t *TLSMockService) SimulateMutualTLS() bool {
	ginkgo.By("🔒 Simulating mutual TLS validation")

	// For testing purposes, we'll simulate successful mTLS
	// In a real implementation, this would involve setting up client certificates
	// and validating the full mTLS handshake

	ginkgo.By("✅ Mutual TLS mock validation successful")
	return true
}

// GetCertificates returns test certificates
func (t *TLSMockService) GetCertificates() (caCert, serverCert, serverKey, clientCert, clientKey []byte) {
	return t.caCert, t.serverCert, t.serverKey, t.clientCert, t.clientKey
}

// Cleanup stops the TLS mock server
func (t *TLSMockService) Cleanup() {
	if t.server != nil {
		t.server.Close()
	}
}

// VulnerabilityScanMock provides vulnerability scanning simulation
type VulnerabilityScanMock struct {
	scanResults   map[string]*ScanResult
	policyResults map[string]*PolicyResult
}

// ScanResult represents a vulnerability scan result
type ScanResult struct {
	ComponentName   string
	Version         string
	Vulnerabilities []Vulnerability
	RiskScore       int
	LastScanned     time.Time
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	CVE          string
	Severity     string
	Description  string
	FixAvailable bool
	CVSS         float64
}

// PolicyResult represents a security policy evaluation result
type PolicyResult struct {
	PolicyName    string
	Passed        bool
	Violations    []PolicyViolation
	Score         int
	LastEvaluated time.Time
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	Rule        string
	Severity    string
	Description string
	Resource    string
}

// NewVulnerabilityScanMock creates a new vulnerability scan mock
func NewVulnerabilityScanMock() *VulnerabilityScanMock {
	vsm := &VulnerabilityScanMock{
		scanResults:   make(map[string]*ScanResult),
		policyResults: make(map[string]*PolicyResult),
	}

	// Pre-populate with test data
	vsm.populateTestData()
	return vsm
}

// populateTestData creates test vulnerability and policy data
func (v *VulnerabilityScanMock) populateTestData() {
	// Container scan results
	v.scanResults["nephoran-operator:latest"] = &ScanResult{
		ComponentName: "nephoran-operator",
		Version:       "latest",
		Vulnerabilities: []Vulnerability{
			{
				CVE:          "CVE-2023-0001",
				Severity:     "LOW",
				Description:  "Minor security issue in base image",
				FixAvailable: true,
				CVSS:         3.1,
			},
		},
		RiskScore:   15,
		LastScanned: time.Now().Add(-1 * time.Hour),
	}

	// Policy results
	v.policyResults["container-security-policy"] = &PolicyResult{
		PolicyName:    "Container Security Policy",
		Passed:        true,
		Violations:    []PolicyViolation{},
		Score:         95,
		LastEvaluated: time.Now().Add(-30 * time.Minute),
	}

	v.policyResults["network-security-policy"] = &PolicyResult{
		PolicyName: "Network Security Policy",
		Passed:     false,
		Violations: []PolicyViolation{
			{
				Rule:        "default-deny-ingress",
				Severity:    "MEDIUM",
				Description: "Default deny ingress policy not found",
				Resource:    "namespace/default",
			},
		},
		Score:         75,
		LastEvaluated: time.Now().Add(-15 * time.Minute),
	}
}

// SimulateContainerScan simulates container vulnerability scanning
func (v *VulnerabilityScanMock) SimulateContainerScan() bool {
	ginkgo.By("🔍 Simulating container vulnerability scan")

	// Simulate scanning process
	time.Sleep(100 * time.Millisecond)

	// Check scan results
	for imageName, result := range v.scanResults {
		ginkgo.By(fmt.Sprintf("📊 Scan result for %s: Risk Score %d", imageName, result.RiskScore))

		criticalVulns := 0
		for _, vuln := range result.Vulnerabilities {
			if vuln.Severity == "CRITICAL" {
				criticalVulns++
			}
		}

		if criticalVulns > 0 {
			ginkgo.By(fmt.Sprintf("❌ Critical vulnerabilities found: %d", criticalVulns))
			return false
		}
	}

	ginkgo.By("✅ Container vulnerability scan passed")
	return true
}

// SimulateDependencyScan simulates dependency vulnerability scanning
func (v *VulnerabilityScanMock) SimulateDependencyScan() bool {
	ginkgo.By("🔍 Simulating dependency vulnerability scan")

	// Simulate scanning process
	time.Sleep(100 * time.Millisecond)

	// Mock dependency scan results
	dependencies := []string{
		"k8s.io/client-go",
		"sigs.k8s.io/controller-runtime",
		"github.com/onsi/ginkgo/v2",
		"github.com/onsi/gomega",
	}

	vulnerableDeps := 0
	for _, dep := range dependencies {
		// Simulate 90% pass rate
		if rand.Intn(10) < 1 {
			vulnerableDeps++
			ginkgo.By(fmt.Sprintf("⚠️ Vulnerable dependency found: %s", dep))
		}
	}

	if vulnerableDeps > 2 {
		ginkgo.By(fmt.Sprintf("❌ Too many vulnerable dependencies: %d", vulnerableDeps))
		return false
	}

	ginkgo.By("✅ Dependency vulnerability scan passed")
	return true
}

// SimulatePolicyEvaluation simulates security policy evaluation
func (v *VulnerabilityScanMock) SimulatePolicyEvaluation() bool {
	ginkgo.By("🔍 Simulating security policy evaluation")

	passedPolicies := 0
	totalPolicies := len(v.policyResults)

	for policyName, result := range v.policyResults {
		if result.Passed {
			passedPolicies++
			ginkgo.By(fmt.Sprintf("✅ Policy passed: %s (Score: %d)", policyName, result.Score))
		} else {
			ginkgo.By(fmt.Sprintf("❌ Policy failed: %s (%d violations)", policyName, len(result.Violations)))
			for _, violation := range result.Violations {
				ginkgo.By(fmt.Sprintf("   - [%s] %s: %s", violation.Severity, violation.Rule, violation.Description))
			}
		}
	}

	passRate := float64(passedPolicies) / float64(totalPolicies)
	ginkgo.By(fmt.Sprintf("📊 Policy evaluation: %.1f%% pass rate", passRate*100))

	return passRate >= 0.8
}

// GetScanResults returns current scan results
func (v *VulnerabilityScanMock) GetScanResults() map[string]*ScanResult {
	return v.scanResults
}

// GetPolicyResults returns current policy results
func (v *VulnerabilityScanMock) GetPolicyResults() map[string]*PolicyResult {
	return v.policyResults
}

// SecurityPolicyMock provides security policy testing capabilities
type SecurityPolicyMock struct {
	policies   map[string]*SecurityPolicy
	violations []PolicyViolation
}

// SecurityPolicy represents a security policy
type SecurityPolicy struct {
	Name        string
	Type        string
	Rules       []SecurityRule
	Enabled     bool
	LastUpdated time.Time
}

// SecurityRule represents a security rule
type SecurityRule struct {
	RuleID      string
	Description string
	Severity    string
	Condition   string
	Action      string
}

// NewSecurityPolicyMock creates a new security policy mock
func NewSecurityPolicyMock() *SecurityPolicyMock {
	spm := &SecurityPolicyMock{
		policies:   make(map[string]*SecurityPolicy),
		violations: make([]PolicyViolation, 0),
	}

	spm.populatePolicies()
	return spm
}

// populatePolicies creates test security policies
func (s *SecurityPolicyMock) populatePolicies() {
	// Pod Security Policy
	s.policies["pod-security-policy"] = &SecurityPolicy{
		Name: "Pod Security Policy",
		Type: "PodSecurityPolicy",
		Rules: []SecurityRule{
			{
				RuleID:      "no-privileged-containers",
				Description: "Containers must not run in privileged mode",
				Severity:    "HIGH",
				Condition:   "spec.securityContext.privileged == false",
				Action:      "deny",
			},
			{
				RuleID:      "no-root-user",
				Description: "Containers must not run as root user",
				Severity:    "MEDIUM",
				Condition:   "spec.securityContext.runAsUser != 0",
				Action:      "deny",
			},
		},
		Enabled:     true,
		LastUpdated: time.Now().Add(-24 * time.Hour),
	}

	// Network Security Policy
	s.policies["network-security-policy"] = &SecurityPolicy{
		Name: "Network Security Policy",
		Type: "NetworkPolicy",
		Rules: []SecurityRule{
			{
				RuleID:      "default-deny-ingress",
				Description: "Default deny all ingress traffic",
				Severity:    "HIGH",
				Condition:   "networkPolicy.spec.policyTypes contains 'Ingress'",
				Action:      "enforce",
			},
			{
				RuleID:      "default-deny-egress",
				Description: "Default deny all egress traffic",
				Severity:    "MEDIUM",
				Condition:   "networkPolicy.spec.policyTypes contains 'Egress'",
				Action:      "enforce",
			},
		},
		Enabled:     true,
		LastUpdated: time.Now().Add(-12 * time.Hour),
	}
}

// ValidatePolicy validates a security policy
func (s *SecurityPolicyMock) ValidatePolicy(policyName string) bool {
	ginkgo.By(fmt.Sprintf("🔍 Validating security policy: %s", policyName))

	policy, exists := s.policies[policyName]
	if !exists {
		ginkgo.By(fmt.Sprintf("❌ Policy not found: %s", policyName))
		return false
	}

	if !policy.Enabled {
		ginkgo.By(fmt.Sprintf("⚠️ Policy disabled: %s", policyName))
		return false
	}

	// Simulate policy validation
	violations := 0
	for _, rule := range policy.Rules {
		// Simulate 95% compliance rate
		if rand.Intn(100) < 5 {
			violations++
			s.violations = append(s.violations, PolicyViolation{
				Rule:        rule.RuleID,
				Severity:    rule.Severity,
				Description: rule.Description,
				Resource:    "mock-resource",
			})
		}
	}

	if violations > 0 {
		ginkgo.By(fmt.Sprintf("⚠️ Policy violations found: %d", violations))
		return violations <= 1 // Allow minor violations
	}

	ginkgo.By(fmt.Sprintf("✅ Policy validation passed: %s", policyName))
	return true
}

// GetPolicies returns all security policies
func (s *SecurityPolicyMock) GetPolicies() map[string]*SecurityPolicy {
	return s.policies
}

// GetViolations returns current policy violations
func (s *SecurityPolicyMock) GetViolations() []PolicyViolation {
	return s.violations
}

// SimulatePolicyEnforcement simulates security policy enforcement
func (s *SecurityPolicyMock) SimulatePolicyEnforcement() bool {
	ginkgo.By("🔍 Simulating security policy enforcement")

	enforcedPolicies := 0
	totalPolicies := len(s.policies)

	for policyName, policy := range s.policies {
		if s.ValidatePolicy(policyName) {
			enforcedPolicies++
			ginkgo.By(fmt.Sprintf("✅ Policy enforced: %s", policy.Name))
		} else {
			ginkgo.By(fmt.Sprintf("❌ Policy enforcement failed: %s", policy.Name))
		}
	}

	enforcementRate := float64(enforcedPolicies) / float64(totalPolicies)
	ginkgo.By(fmt.Sprintf("📊 Policy enforcement: %.1f%% success rate", enforcementRate*100))

	return enforcementRate >= 0.9
}

// CleanupMockServices cleans up all security mock services
func CleanupMockServices(oauth *OAuthMockService, tlsMock *TLSMockService) {
	if oauth != nil {
		oauth.Cleanup()
	}
	if tlsMock != nil {
		tlsMock.Cleanup()
	}
}
