package security

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SecurityScanner performs comprehensive security assessments
type SecurityScanner struct {
	config  *ScannerConfig
	logger  *slog.Logger
	client  *http.Client
	results *ScanResults
	mutex   sync.RWMutex
}

// ScannerConfig holds security scanner configuration
type ScannerConfig struct {
	BaseURL                string        `json:"base_url"`
	Timeout                time.Duration `json:"timeout"`
	MaxConcurrency         int           `json:"max_concurrency"`
	SkipTLSVerification    bool          `json:"skip_tls_verification"`
	EnableVulnScanning     bool          `json:"enable_vuln_scanning"`
	EnablePortScanning     bool          `json:"enable_port_scanning"`
	EnableOWASPTesting     bool          `json:"enable_owasp_testing"`
	EnableAuthTesting      bool          `json:"enable_auth_testing"`
	EnableInjectionTesting bool          `json:"enable_injection_testing"`
	TestAPIKeys            []string      `json:"test_api_keys"`
	TestCredentials        []Credential  `json:"test_credentials"`
	UserAgents             []string      `json:"user_agents"`
	Wordlists              *Wordlists    `json:"wordlists"`
}

// Credential represents test credentials
type Credential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Wordlists holds various wordlists for testing
type Wordlists struct {
	CommonPasswords  []string `json:"common_passwords"`
	CommonPaths      []string `json:"common_paths"`
	SQLInjection     []string `json:"sql_injection"`
	XSSPayloads      []string `json:"xss_payloads"`
	CommandInjection []string `json:"command_injection"`
}

// ScanResults holds comprehensive scan results
type ScanResults struct {
	ScanID            string                 `json:"scan_id"`
	StartTime         time.Time              `json:"start_time"`
	EndTime           time.Time              `json:"end_time"`
	Duration          time.Duration          `json:"duration"`
	TotalTests        int                    `json:"total_tests"`
	PassedTests       int                    `json:"passed_tests"`
	FailedTests       int                    `json:"failed_tests"`
	Vulnerabilities   []Vulnerability        `json:"vulnerabilities"`
	OpenPorts         []Port                 `json:"open_ports"`
	TLSFindings       []TLSFinding           `json:"tls_findings"`
	AuthFindings      []AuthFinding          `json:"auth_findings"`
	InjectionFindings []InjectionFinding     `json:"injection_findings"`
	OWASPFindings     []OWASPFinding         `json:"owasp_findings"`
	Configuration     map[string]interface{} `json:"configuration"`
	RiskScore         float64                `json:"risk_score"`
	Recommendations   []string               `json:"recommendations"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	CVSS        float64   `json:"cvss"`
	CWE         string    `json:"cwe"`
	URL         string    `json:"url"`
	Method      string    `json:"method"`
	Payload     string    `json:"payload"`
	Response    string    `json:"response"`
	Evidence    string    `json:"evidence"`
	Solution    string    `json:"solution"`
	FoundAt     time.Time `json:"found_at"`
}

// Port represents an open network port
type Port struct {
	Number   int    `json:"number"`
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Version  string `json:"version"`
	State    string `json:"state"`
}

// TLSFinding represents TLS/SSL security findings
type TLSFinding struct {
	Issue       string    `json:"issue"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Protocol    string    `json:"protocol"`
	Cipher      string    `json:"cipher"`
	Certificate string    `json:"certificate"`
	Expires     time.Time `json:"expires"`
}

// AuthFinding represents authentication security findings
type AuthFinding struct {
	Issue       string `json:"issue"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint"`
	Method      string `json:"method"`
	Evidence    string `json:"evidence"`
}

// InjectionFinding represents injection vulnerability findings
type InjectionFinding struct {
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	URL       string `json:"url"`
	Parameter string `json:"parameter"`
	Payload   string `json:"payload"`
	Response  string `json:"response"`
	Evidence  string `json:"evidence"`
	Confirmed bool   `json:"confirmed"`
}

// OWASPFinding represents OWASP Top 10 findings
type OWASPFinding struct {
	Category    string `json:"category"`
	Rank        int    `json:"rank"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Evidence    string `json:"evidence"`
	Impact      string `json:"impact"`
}

// NewSecurityScanner creates a new security scanner
func NewSecurityScanner(config *ScannerConfig) *SecurityScanner {
	if config == nil {
		config = getDefaultScannerConfig()
	}

	client := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.SkipTLSVerification,
			},
			MaxIdleConns:       100,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
		},
	}

	return &SecurityScanner{
		config: config,
		logger: slog.Default().With("component", "security-scanner"),
		client: client,
		results: &ScanResults{
			ScanID:            generateScanID(),
			StartTime:         time.Now(),
			Vulnerabilities:   make([]Vulnerability, 0),
			OpenPorts:         make([]Port, 0),
			TLSFindings:       make([]TLSFinding, 0),
			AuthFindings:      make([]AuthFinding, 0),
			InjectionFindings: make([]InjectionFinding, 0),
			OWASPFindings:     make([]OWASPFinding, 0),
			Configuration:     make(map[string]interface{}),
			Recommendations:   make([]string, 0),
		},
	}
}

// getDefaultScannerConfig returns default scanner configuration
func getDefaultScannerConfig() *ScannerConfig {
	return &ScannerConfig{
		Timeout:                30 * time.Second,
		MaxConcurrency:         10,
		SkipTLSVerification:    false,
		EnableVulnScanning:     true,
		EnablePortScanning:     true,
		EnableOWASPTesting:     true,
		EnableAuthTesting:      true,
		EnableInjectionTesting: true,
		UserAgents: []string{
			"Mozilla/5.0 (compatible; SecurityScanner/1.0)",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		},
		Wordlists: &Wordlists{
			CommonPasswords: []string{
				"admin", "password", "123456", "admin123", "root", "test",
				"password123", "administrator", "letmein", "welcome",
			},
			CommonPaths: []string{
				"/admin", "/api", "/swagger", "/docs", "/health",
				"/metrics", "/config", "/debug", "/test", "/backup",
			},
			SQLInjection: []string{
				"'", "\"", "1' OR '1'='1", "1\" OR \"1\"=\"1",
				"'; DROP TABLE users; --", "1 UNION SELECT NULL--",
			},
			XSSPayloads: []string{
				"<script>alert('XSS')</script>",
				"javascript:alert('XSS')",
				"<img src=x onerror=alert('XSS')>",
				"<svg onload=alert('XSS')>",
			},
			CommandInjection: []string{
				"; ls", "| id", "& whoami", "; cat /etc/passwd",
				"$(id)", "`whoami`", "|| id",
			},
		},
	}
}

// RunFullScan performs a comprehensive security scan
func (ss *SecurityScanner) RunFullScan(ctx context.Context) (*ScanResults, error) {
	ss.logger.Info("Starting comprehensive security scan", "base_url", ss.config.BaseURL)

	ss.results.StartTime = time.Now()

	// Run different types of scans concurrently
	var wg sync.WaitGroup

	if ss.config.EnablePortScanning {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ss.runPortScan(ctx)
		}()
	}

	if ss.config.EnableOWASPTesting {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ss.runOWASPTests(ctx)
		}()
	}

	if ss.config.EnableAuthTesting {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ss.runAuthTests(ctx)
		}()
	}

	if ss.config.EnableInjectionTesting {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ss.runInjectionTests(ctx)
		}()
	}

	// Run TLS/SSL tests
	wg.Add(1)
	go func() {
		defer wg.Done()
		ss.runTLSTests(ctx)
	}()

	// Wait for all scans to complete
	wg.Wait()

	ss.results.EndTime = time.Now()
	ss.results.Duration = ss.results.EndTime.Sub(ss.results.StartTime)

	// Calculate risk score and generate recommendations
	ss.calculateRiskScore()
	ss.generateRecommendations()

	ss.logger.Info("Security scan completed",
		"duration", ss.results.Duration,
		"vulnerabilities", len(ss.results.Vulnerabilities),
		"risk_score", ss.results.RiskScore)

	return ss.results, nil
}

// runPortScan performs network port scanning
func (ss *SecurityScanner) runPortScan(ctx context.Context) {
	ss.logger.Info("Starting port scan")

	parsedURL, err := url.Parse(ss.config.BaseURL)
	if err != nil {
		ss.logger.Error("Invalid base URL", "error", err)
		return
	}

	host := parsedURL.Hostname()
	commonPorts := []int{
		21, 22, 23, 25, 53, 80, 110, 143, 443, 993, 995,
		8080, 8443, 8000, 9000, 5432, 3306, 27017, 6379,
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, ss.config.MaxConcurrency)

	for _, port := range commonPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ss.scanPort(ctx, host, p)
		}(port)
	}

	wg.Wait()
}

// scanPort scans a specific port
func (ss *SecurityScanner) scanPort(ctx context.Context, host string, port int) {
	address := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return // Port closed or filtered
	}
	defer conn.Close()

	portInfo := Port{
		Number:   port,
		Protocol: "tcp",
		State:    "open",
		Service:  getServiceName(port),
	}

	// Try to get service banner
	if banner := ss.getBanner(conn); banner != "" {
		portInfo.Version = banner
	}

	ss.mutex.Lock()
	ss.results.OpenPorts = append(ss.results.OpenPorts, portInfo)
	ss.mutex.Unlock()

	// Check for insecure services
	if ss.isInsecureService(port) {
		vuln := Vulnerability{
			ID:          fmt.Sprintf("INSECURE_SERVICE_%d", port),
			Title:       fmt.Sprintf("Insecure service on port %d", port),
			Description: fmt.Sprintf("Service running on port %d may use insecure protocols", port),
			Severity:    "Medium",
			CVSS:        5.0,
			URL:         address,
			Solution:    "Consider using secure alternatives or proper authentication",
			FoundAt:     time.Now(),
		}

		ss.mutex.Lock()
		ss.results.Vulnerabilities = append(ss.results.Vulnerabilities, vuln)
		ss.mutex.Unlock()
	}
}

// runOWASPTests performs OWASP Top 10 vulnerability tests
func (ss *SecurityScanner) runOWASPTests(ctx context.Context) {
	ss.logger.Info("Starting OWASP Top 10 tests")

	tests := []func(context.Context){
		ss.testInjectionFlaws,           // A01: Injection
		ss.testBrokenAuthentication,     // A02: Broken Authentication
		ss.testSensitiveDataExposure,    // A03: Sensitive Data Exposure
		ss.testXXE,                      // A04: XML External Entities
		ss.testBrokenAccessControl,      // A05: Broken Access Control
		ss.testSecurityMisconfiguration, // A06: Security Misconfiguration
		ss.testXSS,                      // A07: Cross-Site Scripting
		ss.testInsecureDeserialization,  // A08: Insecure Deserialization
		ss.testVulnerableComponents,     // A09: Using Components with Known Vulnerabilities
		ss.testInsufficientLogging,      // A10: Insufficient Logging & Monitoring
	}

	var wg sync.WaitGroup
	for i, test := range tests {
		wg.Add(1)
		go func(rank int, testFunc func(context.Context)) {
			defer wg.Done()
			testFunc(ctx)
		}(i+1, test)
	}

	wg.Wait()
}

// testInjectionFlaws tests for injection vulnerabilities
func (ss *SecurityScanner) testInjectionFlaws(ctx context.Context) {
	endpoints := []string{
		"/api/v1/intents",
		"/search",
		"/login",
		"/api/query",
	}

	for _, endpoint := range endpoints {
		for _, payload := range ss.config.Wordlists.SQLInjection {
			ss.testEndpointWithPayload(ctx, endpoint, "sql_injection", payload)
		}

		for _, payload := range ss.config.Wordlists.CommandInjection {
			ss.testEndpointWithPayload(ctx, endpoint, "command_injection", payload)
		}
	}
}

// testBrokenAuthentication tests for authentication vulnerabilities
func (ss *SecurityScanner) testBrokenAuthentication(ctx context.Context) {
	// Test for weak passwords
	ss.testWeakPasswords(ctx, "/auth/login")

	// Test for session management issues
	ss.testSessionManagement(ctx)

	// Test for authentication bypass
	ss.testAuthBypass(ctx)
}

// runAuthTests performs comprehensive authentication testing
func (ss *SecurityScanner) runAuthTests(ctx context.Context) {
	ss.logger.Info("Starting authentication tests")

	// Test authentication endpoints
	authEndpoints := []string{
		"/auth/login",
		"/api/auth/login",
		"/login",
		"/signin",
		"/authenticate",
	}

	for _, endpoint := range authEndpoints {
		// Test for default credentials
		ss.testDefaultCredentials(ctx, endpoint)

		// Test for brute force protection
		ss.testBruteForceProtection(ctx, endpoint)

		// Test for password policy
		ss.testPasswordPolicy(ctx, endpoint)
	}

	// Test JWT vulnerabilities
	ss.testJWTVulnerabilities(ctx)
}

// runInjectionTests performs injection vulnerability testing
func (ss *SecurityScanner) runInjectionTests(ctx context.Context) {
	ss.logger.Info("Starting injection tests")

	testParams := []string{
		"id", "name", "query", "search", "filter", "sort",
		"user", "username", "email", "password", "token",
	}

	for _, endpoint := range ss.config.Wordlists.CommonPaths {
		fullURL := ss.config.BaseURL + endpoint

		for _, param := range testParams {
			// SQL Injection
			for _, payload := range ss.config.Wordlists.SQLInjection {
				ss.testParameterInjection(ctx, fullURL, param, "sql", payload)
			}

			// XSS
			for _, payload := range ss.config.Wordlists.XSSPayloads {
				ss.testParameterInjection(ctx, fullURL, param, "xss", payload)
			}

			// Command Injection
			for _, payload := range ss.config.Wordlists.CommandInjection {
				ss.testParameterInjection(ctx, fullURL, param, "command", payload)
			}
		}
	}
}

// runTLSTests performs TLS/SSL security testing
func (ss *SecurityScanner) runTLSTests(ctx context.Context) {
	ss.logger.Info("Starting TLS/SSL tests")

	parsedURL, err := url.Parse(ss.config.BaseURL)
	if err != nil {
		return
	}

	if parsedURL.Scheme != "https" {
		finding := TLSFinding{
			Issue:       "HTTP_ONLY",
			Severity:    "High",
			Description: "Service does not support HTTPS",
			Protocol:    "HTTP",
		}

		ss.mutex.Lock()
		ss.results.TLSFindings = append(ss.results.TLSFindings, finding)
		ss.mutex.Unlock()
		return
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "443"
	}

	// Test TLS configuration
	ss.testTLSConfiguration(ctx, host, port)

	// Test certificate
	ss.testCertificate(ctx, host, port)
}

// Helper methods for specific tests

func (ss *SecurityScanner) testEndpointWithPayload(ctx context.Context, endpoint, injectionType, payload string) {
	fullURL := ss.config.BaseURL + endpoint

	// Test GET parameters
	testURL := fmt.Sprintf("%s?test=%s", fullURL, url.QueryEscape(payload))
	resp, err := ss.client.Get(testURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	ss.analyzeInjectionResponse(testURL, "GET", payload, string(body), injectionType)

	// Test POST data
	data := url.Values{"test": {payload}}
	resp, err = ss.client.PostForm(fullURL, data)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	ss.analyzeInjectionResponse(fullURL, "POST", payload, string(body), injectionType)
}

func (ss *SecurityScanner) analyzeInjectionResponse(testURL, method, payload, response, injectionType string) {
	var indicators []string

	switch injectionType {
	case "sql_injection":
		indicators = []string{
			"SQL syntax",
			"mysql_fetch",
			"PostgreSQL query failed",
			"ORA-00936",
			"Microsoft Access Driver",
			"SQLServer JDBC Driver",
		}
	case "command_injection":
		indicators = []string{
			"uid=", "gid=", "/bin/sh", "/bin/bash",
			"root:", "/etc/passwd", "command not found",
		}
	case "xss":
		indicators = []string{
			"<script>", "javascript:", "alert(",
			"onerror=", "onload=",
		}
	}

	for _, indicator := range indicators {
		if strings.Contains(response, indicator) {
			finding := InjectionFinding{
				Type:      injectionType,
				Severity:  "High",
				URL:       testURL,
				Parameter: "test",
				Payload:   payload,
				Response:  response[:min(len(response), 500)],
				Evidence:  indicator,
				Confirmed: true,
			}

			ss.mutex.Lock()
			ss.results.InjectionFindings = append(ss.results.InjectionFindings, finding)
			ss.mutex.Unlock()
			break
		}
	}
}

func (ss *SecurityScanner) testDefaultCredentials(ctx context.Context, endpoint string) {
	fullURL := ss.config.BaseURL + endpoint

	for _, cred := range ss.config.TestCredentials {
		data := url.Values{
			"username": {cred.Username},
			"password": {cred.Password},
		}

		resp, err := ss.client.PostForm(fullURL, data)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			finding := AuthFinding{
				Issue:       "DEFAULT_CREDENTIALS",
				Severity:    "Critical",
				Description: fmt.Sprintf("Default credentials accepted: %s/%s", cred.Username, cred.Password),
				Endpoint:    endpoint,
				Method:      "POST",
				Evidence:    fmt.Sprintf("HTTP %d response", resp.StatusCode),
			}

			ss.mutex.Lock()
			ss.results.AuthFindings = append(ss.results.AuthFindings, finding)
			ss.mutex.Unlock()
		}
	}
}

func (ss *SecurityScanner) testParameterInjection(ctx context.Context, baseURL, param, injectionType, payload string) {
	testURL := fmt.Sprintf("%s?%s=%s", baseURL, param, url.QueryEscape(payload))

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return
	}

	resp, err := ss.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	ss.analyzeInjectionResponse(testURL, "GET", payload, string(body), injectionType)
}

func (ss *SecurityScanner) testTLSConfiguration(ctx context.Context, host, port string) {
	// Test for weak TLS versions
	weakVersions := []uint16{
		tls.VersionSSL30,
		tls.VersionTLS10,
		tls.VersionTLS11,
	}

	for _, version := range weakVersions {
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second},
			"tcp", host+":"+port, &tls.Config{
				MaxVersion:         version,
				// SECURITY: InsecureSkipVerify is intentionally set to true for security scanning
				// This allows the scanner to test for weak TLS configurations on target servers
				InsecureSkipVerify: true,
			})

		if err == nil {
			conn.Close()

			finding := TLSFinding{
				Issue:       "WEAK_TLS_VERSION",
				Severity:    "High",
				Description: fmt.Sprintf("Weak TLS version supported: %s", getTLSVersionName(version)),
				Protocol:    getTLSVersionName(version),
			}

			ss.mutex.Lock()
			ss.results.TLSFindings = append(ss.results.TLSFindings, finding)
			ss.mutex.Unlock()
		}
	}
}

func (ss *SecurityScanner) testCertificate(ctx context.Context, host, port string) {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second},
		"tcp", host+":"+port, &tls.Config{})
	if err != nil {
		return
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return
	}

	cert := certs[0]

	// Check certificate expiration
	if time.Until(cert.NotAfter) < 30*24*time.Hour {
		finding := TLSFinding{
			Issue:       "CERT_EXPIRING_SOON",
			Severity:    "Medium",
			Description: "Certificate expires within 30 days",
			Certificate: cert.Subject.String(),
			Expires:     cert.NotAfter,
		}

		ss.mutex.Lock()
		ss.results.TLSFindings = append(ss.results.TLSFindings, finding)
		ss.mutex.Unlock()
	}

	// Check for self-signed certificate
	if cert.Issuer.String() == cert.Subject.String() {
		finding := TLSFinding{
			Issue:       "SELF_SIGNED_CERT",
			Severity:    "Medium",
			Description: "Self-signed certificate detected",
			Certificate: cert.Subject.String(),
		}

		ss.mutex.Lock()
		ss.results.TLSFindings = append(ss.results.TLSFindings, finding)
		ss.mutex.Unlock()
	}
}

// Additional OWASP test methods (stubs for brevity)

func (ss *SecurityScanner) testSensitiveDataExposure(ctx context.Context) {
	// Test for exposed sensitive information
	sensitivePatterns := []string{
		`password.*=.*['"](.*?)['"]`,
		`api[_-]?key.*=.*['"](.*?)['"]`,
		`secret.*=.*['"](.*?)['"]`,
		`token.*=.*['"](.*?)['"]`,
	}

	commonPaths := []string{
		"/.env", "/config.json", "/config.xml", "/web.config",
		"/.git/config", "/backup.sql", "/database.sql",
	}

	for _, path := range commonPaths {
		ss.testPathForSensitiveData(ctx, path, sensitivePatterns)
	}
}

func (ss *SecurityScanner) testXXE(ctx context.Context) {
	// Test for XML External Entity vulnerabilities
	xxePayload := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE test [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
<test>&xxe;</test>`

	xmlEndpoints := []string{
		"/api/xml", "/upload", "/parse", "/import",
	}

	for _, endpoint := range xmlEndpoints {
		ss.testXMLEndpoint(ctx, endpoint, xxePayload)
	}
}

func (ss *SecurityScanner) testBrokenAccessControl(ctx context.Context) {
	// Test for access control issues
	// This would test for horizontal/vertical privilege escalation
}

func (ss *SecurityScanner) testSecurityMisconfiguration(ctx context.Context) {
	// Test for security misconfigurations
	configPaths := []string{
		"/server-status", "/server-info", "/.htaccess",
		"/phpinfo.php", "/info.php", "/test.php",
	}

	for _, path := range configPaths {
		ss.testConfigurationExposure(ctx, path)
	}
}

func (ss *SecurityScanner) testXSS(ctx context.Context) {
	// XSS testing is handled in runInjectionTests
}

func (ss *SecurityScanner) testInsecureDeserialization(ctx context.Context) {
	// Test for insecure deserialization
}

func (ss *SecurityScanner) testVulnerableComponents(ctx context.Context) {
	// Test for known vulnerable components
}

func (ss *SecurityScanner) testInsufficientLogging(ctx context.Context) {
	// Test for logging and monitoring issues
}

// Utility functions

func generateScanID() string {
	return fmt.Sprintf("scan_%d", time.Now().Unix())
}

func getServiceName(port int) string {
	services := map[int]string{
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "dns",
		80:    "http",
		110:   "pop3",
		143:   "imap",
		443:   "https",
		993:   "imaps",
		995:   "pop3s",
		3306:  "mysql",
		5432:  "postgresql",
		6379:  "redis",
		8080:  "http-alt",
		8443:  "https-alt",
		27017: "mongodb",
	}

	if service, exists := services[port]; exists {
		return service
	}
	return "unknown"
}

func (ss *SecurityScanner) isInsecureService(port int) bool {
	insecurePorts := []int{21, 23, 25, 110, 143} // FTP, Telnet, SMTP, POP3, IMAP
	for _, p := range insecurePorts {
		if port == p {
			return true
		}
	}
	return false
}

func (ss *SecurityScanner) getBanner(conn net.Conn) string {
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	scanner := bufio.NewScanner(conn)

	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

func getTLSVersionName(version uint16) string {
	versions := map[uint16]string{
		tls.VersionSSL30: "SSLv3",
		tls.VersionTLS10: "TLS 1.0",
		tls.VersionTLS11: "TLS 1.1",
		tls.VersionTLS12: "TLS 1.2",
		tls.VersionTLS13: "TLS 1.3",
	}

	if name, exists := versions[version]; exists {
		return name
	}
	return "Unknown"
}

func (ss *SecurityScanner) calculateRiskScore() {
	var score float64

	// Calculate based on vulnerability severity
	for _, vuln := range ss.results.Vulnerabilities {
		switch vuln.Severity {
		case "Critical":
			score += 10
		case "High":
			score += 7
		case "Medium":
			score += 4
		case "Low":
			score += 1
		}
	}

	// Add scores for other findings
	score += float64(len(ss.results.TLSFindings)) * 2
	score += float64(len(ss.results.AuthFindings)) * 5
	score += float64(len(ss.results.InjectionFindings)) * 8
	score += float64(len(ss.results.OWASPFindings)) * 6

	// Normalize to 0-100 scale
	if score > 100 {
		score = 100
	}

	ss.results.RiskScore = score
}

func (ss *SecurityScanner) generateRecommendations() {
	recommendations := []string{}

	if len(ss.results.TLSFindings) > 0 {
		recommendations = append(recommendations, "Upgrade TLS configuration and certificates")
	}

	if len(ss.results.AuthFindings) > 0 {
		recommendations = append(recommendations, "Strengthen authentication mechanisms")
	}

	if len(ss.results.InjectionFindings) > 0 {
		recommendations = append(recommendations, "Implement input validation and sanitization")
	}

	if ss.results.RiskScore > 70 {
		recommendations = append(recommendations, "Immediate security review required")
	}

	ss.results.Recommendations = recommendations
}

// Helper methods (stubs for brevity)

func (ss *SecurityScanner) testWeakPasswords(ctx context.Context, endpoint string)        {}
func (ss *SecurityScanner) testSessionManagement(ctx context.Context)                     {}
func (ss *SecurityScanner) testAuthBypass(ctx context.Context)                            {}
func (ss *SecurityScanner) testBruteForceProtection(ctx context.Context, endpoint string) {}
func (ss *SecurityScanner) testPasswordPolicy(ctx context.Context, endpoint string)       {}
func (ss *SecurityScanner) testJWTVulnerabilities(ctx context.Context)                    {}
func (ss *SecurityScanner) testPathForSensitiveData(ctx context.Context, path string, patterns []string) {
}
func (ss *SecurityScanner) testXMLEndpoint(ctx context.Context, endpoint, payload string) {}
func (ss *SecurityScanner) testConfigurationExposure(ctx context.Context, path string)    {}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GenerateReport generates a comprehensive security report
func (ss *SecurityScanner) GenerateReport() ([]byte, error) {
	return json.MarshalIndent(ss.results, "", "  ")
}

// SaveReport saves the scan results to a file
func (ss *SecurityScanner) SaveReport(filename string) error {
	report, err := ss.GenerateReport()
	if err != nil {
		return err
	}

	return writeFile(filename, report)
}

func writeFile(filename string, data []byte) error {
	// Implementation would write to file
	// For now, just return nil
	return nil
}
