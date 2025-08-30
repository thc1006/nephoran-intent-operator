/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

	nephiov1 "github.com/nephio-project/nephoran-intent-operator/api/v1"


	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecurityScanner performs comprehensive security scans on network services and applications.

type SecurityScanner struct {
	client.Client

	logger *slog.Logger

	config SecurityScannerConfig

	httpClient *http.Client

	results ScanResults

	mutex sync.RWMutex

	ctx context.Context

	cancel context.CancelFunc

	// Add concurrent scanning support

	semaphore chan struct{}
}

// SecurityScannerConfig defines configuration options for the security scanner.

type SecurityScannerConfig struct {
	// Maximum number of concurrent scans

	MaxConcurrency int `json:"maxConcurrency"`

	// Timeout for individual scan operations

	ScanTimeout time.Duration `json:"scanTimeout"`

	// HTTP request timeout

	HTTPTimeout time.Duration `json:"httpTimeout"`

	// Enable/disable specific scan types

	EnablePortScan     bool `json:"enablePortScan"`

	EnableVulnScan     bool `json:"enableVulnScan"`

	EnableTLSScan      bool `json:"enableTlsScan"`

	EnableHeaderScan   bool `json:"enableHeaderScan"`

	EnableInjectionScan bool `json:"enableInjectionScan"`

	// Port ranges for scanning

	PortRanges []PortRange `json:"portRanges"`

	// Custom headers for HTTP requests

	CustomHeaders map[string]string `json:"customHeaders,omitempty"`

	// User agent for HTTP requests

	UserAgent string `json:"userAgent,omitempty"`

	// Service name for metrics

	ServiceName string `json:"serviceName"`
}

// PortRange defines a range of ports to scan.

type PortRange struct {
	Start int `json:"start"`

	End int `json:"end"`
}

// ScanResults contains the results of a comprehensive security scan.

type ScanResults struct {
	Timestamp time.Time `json:"timestamp"`

	Target string `json:"target"`

	// Port scan results

	OpenPorts []PortInfo `json:"openPorts"`

	// Vulnerability scan results

	Vulnerabilities []Vulnerability `json:"vulnerabilities"`

	// TLS/SSL findings

	TLSFindings []TLSFinding `json:"tlsFindings"`

	// HTTP header security findings

	HeaderFindings []HeaderFinding `json:"headerFindings"`

	// Injection vulnerability findings

	InjectionFindings []InjectionFinding `json:"injectionFindings"`

	// Summary statistics

	Summary ScanSummary `json:"summary"`

	// Scan duration

	Duration time.Duration `json:"duration"`
}

// PortInfo contains information about an open port.

type PortInfo struct {
	Port     int    `json:"port"`

	Protocol string `json:"protocol"`

	Service  string `json:"service"`

	Banner   string `json:"banner,omitempty"`

	State    string `json:"state"`
}

// Vulnerability represents a security vulnerability found during scanning.

type Vulnerability struct {
	ID          string `json:"id"`

	Title       string `json:"title"`

	Description string `json:"description"`

	Severity    string `json:"severity"` // Critical, High, Medium, Low

	CVE         string `json:"cve,omitempty"`

	CVSS        string `json:"cvss,omitempty"`

	Solution    string `json:"solution,omitempty"`

	References  []string `json:"references,omitempty"`

	Port        int    `json:"port,omitempty"`

	Service     string `json:"service,omitempty"`
}

// TLSFinding represents TLS/SSL security findings.

type TLSFinding struct {
	Issue       string `json:"issue"`

	Severity    string `json:"severity"`

	Description string `json:"description"`

	Protocol    string `json:"protocol,omitempty"`

	Cipher      string `json:"cipher,omitempty"`

	Certificate string `json:"certificate,omitempty"`

	Expiry      string `json:"expiry,omitempty"`
}

// HeaderFinding represents HTTP security header findings.

type HeaderFinding struct {
	Header      string `json:"header"`

	Issue       string `json:"issue"`

	Severity    string `json:"severity"`

	Description string `json:"description"`

	Recommendation string `json:"recommendation"`

	Present     bool `json:"present"`

	Value       string `json:"value,omitempty"`
}

// InjectionFinding represents injection vulnerability findings.

type InjectionFinding struct {
	Type        string `json:"type"` // SQL, XSS, Command, etc.

	URL         string `json:"url"`

	Parameter   string `json:"parameter,omitempty"`

	Payload     string `json:"payload"`

	Method      string `json:"method"`

	Severity    string `json:"severity"`

	Description string `json:"description"`

	Evidence    string `json:"evidence,omitempty"`
}

// ScanSummary provides a high-level summary of scan results.

type ScanSummary struct {
	TotalPorts        int `json:"totalPorts"`

	OpenPorts         int `json:"openPorts"`

	TotalVulns        int `json:"totalVulns"`

	CriticalVulns     int `json:"criticalVulns"`

	HighVulns         int `json:"highVulns"`

	MediumVulns       int `json:"mediumVulns"`

	LowVulns          int `json:"lowVulns"`

	TLSIssues         int `json:"tlsIssues"`

	HeaderIssues      int `json:"headerIssues"`

	InjectionIssues   int `json:"injectionIssues"`

	SecurityScore     int `json:"securityScore"` // 0-100

	RiskLevel         string `json:"riskLevel"`   // Low, Medium, High, Critical
}

// NewSecurityScanner creates a new SecurityScanner instance.

func NewSecurityScanner(client client.Client, logger *slog.Logger, config SecurityScannerConfig) (*SecurityScanner, error) {

	ctx, cancel := context.WithCancel(context.Background())

	// Set default configuration values

	if config.MaxConcurrency == 0 {

		config.MaxConcurrency = 10

	}

	if config.ScanTimeout == 0 {

		config.ScanTimeout = 5 * time.Minute

	}

	if config.HTTPTimeout == 0 {

		config.HTTPTimeout = 30 * time.Second

	}

	if config.UserAgent == "" {

		config.UserAgent = "Nephoran-Security-Scanner/1.0"

	}

	if len(config.PortRanges) == 0 {

		config.PortRanges = []PortRange{

			{Start: 80, End: 80},

			{Start: 443, End: 443},

			{Start: 8080, End: 8080},

			{Start: 8443, End: 8443},

		}

	}

	// Create HTTP client with timeout

	httpClient := &http.Client{

		Timeout: config.HTTPTimeout,

		Transport: &http.Transport{

			TLSClientConfig: &tls.Config{

				MinVersion: tls.VersionTLS12, // Enforce minimum TLS 1.2

			},

		},
	}

	scanner := &SecurityScanner{

		Client: client,

		logger: logger,

		config: config,

		httpClient: httpClient,

		ctx:    ctx,

		cancel: cancel,

		semaphore: make(chan struct{}, config.MaxConcurrency),
	}

	// Initialize semaphore

	for i := 0; i < config.MaxConcurrency; i++ {

		scanner.semaphore <- struct{}{}

	}

	return scanner, nil

}

// ScanTarget performs a comprehensive security scan on the specified target.

func (ss *SecurityScanner) ScanTarget(ctx context.Context, target string) (*ScanResults, error) {

	start := time.Now()

	ss.logger.Info("Starting security scan", "target", target)

	// Initialize results

	ss.mutex.Lock()

	ss.results = ScanResults{

		Timestamp: start,

		Target:    target,

		OpenPorts:         []PortInfo{},

		Vulnerabilities:   []Vulnerability{},

		TLSFindings:       []TLSFinding{},

		HeaderFindings:    []HeaderFinding{},

		InjectionFindings: []InjectionFinding{},
	}

	ss.mutex.Unlock()

	// Parse target

	host, port, err := ss.parseTarget(target)

	if err != nil {

		return nil, fmt.Errorf("failed to parse target: %w", err)

	}

	var wg sync.WaitGroup

	// Perform port scan

	if ss.config.EnablePortScan {

		wg.Add(1)

		go func() {

			defer wg.Done()

			ss.scanPorts(ctx, host)

		}()

	}

	// Perform TLS scan

	if ss.config.EnableTLSScan && port != "" {

		wg.Add(1)

		go func() {

			defer wg.Done()

			ss.testTLSConfiguration(ctx, host, port)

			ss.testCertificate(ctx, host, port)

		}()

	}

	// Perform HTTP header scan

	if ss.config.EnableHeaderScan {

		wg.Add(1)

		go func() {

			defer wg.Done()

			ss.scanHTTPHeaders(ctx, target)

		}()

	}

	// Perform injection scan

	if ss.config.EnableInjectionScan {

		wg.Add(1)

		go func() {

			defer wg.Done()

			ss.scanInjectionVulns(ctx, target)

		}()

	}

	// Perform vulnerability scan

	if ss.config.EnableVulnScan {

		wg.Add(1)

		go func() {

			defer wg.Done()

			ss.scanVulnerabilities(ctx, host, port)

		}()

	}

	// Wait for all scans to complete

	wg.Wait()

	duration := time.Since(start)

	// Finalize results

	ss.mutex.Lock()

	ss.results.Duration = duration

	ss.results.Summary = ss.calculateSummary()

	results := ss.results

	ss.mutex.Unlock()

	ss.logger.Info("Security scan completed",

		"target", target,

		"duration", duration,

		"openPorts", results.Summary.OpenPorts,

		"vulnerabilities", results.Summary.TotalVulns,

		"securityScore", results.Summary.SecurityScore,

	)

	return &results, nil

}

// parseTarget extracts host and port from target string.

func (ss *SecurityScanner) parseTarget(target string) (string, string, error) {

	// Handle URLs

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {

		u, err := url.Parse(target)

		if err != nil {

			return "", "", err

		}

		host, port, err := net.SplitHostPort(u.Host)

		if err != nil {

			// No port specified, use default

			if u.Scheme == "https" {

				return u.Host, "443", nil

			}

			return u.Host, "80", nil

		}

		return host, port, nil

	}

	// Handle host:port format

	host, port, err := net.SplitHostPort(target)

	if err != nil {

		// No port specified, assume it's just a host

		return target, "", nil

	}

	return host, port, nil

}

// scanPorts scans for open ports on the target host.

func (ss *SecurityScanner) scanPorts(ctx context.Context, host string) {

	ss.logger.Debug("Starting port scan", "host", host)

	var wg sync.WaitGroup

	for _, portRange := range ss.config.PortRanges {

		for port := portRange.Start; port <= portRange.End; port++ {

			wg.Add(1)

			go func(p int) {

				defer wg.Done()

				// Acquire semaphore

				<-ss.semaphore

				// Scan port and release semaphore
				func() {
					defer func() { ss.semaphore <- struct{}{} }()
					ss.scanPort(ctx, host, p)
				}()

			}(port)

		}

	}

	wg.Wait()

}

// scanPort scans a single port.

func (ss *SecurityScanner) scanPort(ctx context.Context, host string, port int) {

	address := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.DialTimeout("tcp", address, 3*time.Second)

	if err != nil {

		return // Port is closed or filtered

	}

	defer conn.Close()

	// Port is open

	portInfo := PortInfo{

		Port:     port,

		Protocol: "tcp",

		Service:  ss.identifyService(port),

		State:    "open",
	}

	// Try to grab banner

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	buffer := make([]byte, 1024)

	n, err := conn.Read(buffer)

	if err == nil && n > 0 {

		portInfo.Banner = string(buffer[:n])

	}

	ss.mutex.Lock()

	ss.results.OpenPorts = append(ss.results.OpenPorts, portInfo)

	ss.mutex.Unlock()

	ss.logger.Debug("Found open port",

		"host", host,

		"port", port,

		"service", portInfo.Service,

	)

}

// identifyService attempts to identify the service running on a port.

func (ss *SecurityScanner) identifyService(port int) string {

	services := map[int]string{

		21:   "ftp",

		22:   "ssh",

		23:   "telnet",

		25:   "smtp",

		53:   "dns",

		80:   "http",

		110:  "pop3",

		143:  "imap",

		443:  "https",

		993:  "imaps",

		995:  "pop3s",

		3306: "mysql",

		5432: "postgresql",

		6379: "redis",

		8080: "http-alt",

		8443: "https-alt",

		9200: "elasticsearch",
	}

	if service, exists := services[port]; exists {

		return service

	}

	return "unknown"

}

func (ss *SecurityScanner) testTLSConfiguration(ctx context.Context, host, port string) {

	// Test for weak TLS versions.

	weakVersions := []uint16{

		tls.VersionTLS10,

		tls.VersionTLS11,
	}

	for _, version := range weakVersions {
		// #nosec G402 -- This is intentional for security scanning to detect weak TLS versions
		// The security scanner needs to test for weak TLS configurations on target servers
		// This is not used for production connections but only for vulnerability assessment
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second},

			"tcp", host+":"+port, &tls.Config{

				MaxVersion: version,

				// SECURITY SCANNER EXCEPTION: InsecureSkipVerify is intentionally set to true
				// for security scanning purposes. This allows the scanner to test for weak
				// TLS configurations on target servers. This is NOT used for production
				// connections but only for vulnerability assessment.
				InsecureSkipVerify: true, // #nosec G402

			})

		if err == nil {

			conn.Close()

			finding := TLSFinding{

				Issue: "WEAK_TLS_VERSION",

				Severity: "High",

				Description: fmt.Sprintf("Weak TLS version supported: %s", getTLSVersionName(version)),

				Protocol: getTLSVersionName(version),
			}

			ss.mutex.Lock()

			ss.results.TLSFindings = append(ss.results.TLSFindings, finding)

			ss.mutex.Unlock()

		}

	}

}

func (ss *SecurityScanner) testCertificate(ctx context.Context, host, port string) {
	// Security fix: Set minimum TLS version (G402)
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second},

		"tcp", host+":"+port, &tls.Config{
			MinVersion: tls.VersionTLS12, // Enforce minimum TLS 1.2
		})

	if err != nil {

		return

	}

	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates

	if len(certs) == 0 {

		return

	}

	cert := certs[0]

	// Check certificate expiration.

	now := time.Now()

	if cert.NotAfter.Before(now) {

		finding := TLSFinding{

			Issue: "EXPIRED_CERTIFICATE",

			Severity: "Critical",

			Description: fmt.Sprintf("Certificate expired on %s", cert.NotAfter.Format("2006-01-02")),

			Certificate: cert.Subject.CommonName,

			Expiry: cert.NotAfter.Format("2006-01-02 15:04:05 UTC"),
		}

		ss.mutex.Lock()

		ss.results.TLSFindings = append(ss.results.TLSFindings, finding)

		ss.mutex.Unlock()

	} else if cert.NotAfter.Before(now.Add(30 * 24 * time.Hour)) {

		// Certificate expires within 30 days

		finding := TLSFinding{

			Issue: "CERTIFICATE_EXPIRING_SOON",

			Severity: "Medium",

			Description: fmt.Sprintf("Certificate expires soon: %s", cert.NotAfter.Format("2006-01-02")),

			Certificate: cert.Subject.CommonName,

			Expiry: cert.NotAfter.Format("2006-01-02 15:04:05 UTC"),
		}

		ss.mutex.Lock()

		ss.results.TLSFindings = append(ss.results.TLSFindings, finding)

		ss.mutex.Unlock()

	}

	// Check for weak signature algorithms

	if cert.SignatureAlgorithm == x509.MD5WithRSA || cert.SignatureAlgorithm == x509.SHA1WithRSA {

		finding := TLSFinding{

			Issue: "WEAK_SIGNATURE_ALGORITHM",

			Severity: "High",

			Description: fmt.Sprintf("Certificate uses weak signature algorithm: %s", cert.SignatureAlgorithm),

			Certificate: cert.Subject.CommonName,
		}

		ss.mutex.Lock()

		ss.results.TLSFindings = append(ss.results.TLSFindings, finding)

		ss.mutex.Unlock()

	}

	// Check key length

	if cert.PublicKeyAlgorithm == x509.RSA {

		if rsaKey, ok := cert.PublicKey.(*interface{}); ok {

			_ = rsaKey // We would check key size here

		}

	}

}

// getTLSVersionName returns the name of a TLS version.

func getTLSVersionName(version uint16) string {

	switch version {

	case tls.VersionTLS10:

		return "TLS 1.0"

	case tls.VersionTLS11:

		return "TLS 1.1"

	case tls.VersionTLS12:

		return "TLS 1.2"

	case tls.VersionTLS13:

		return "TLS 1.3"

	default:

		return fmt.Sprintf("Unknown (%d)", version)

	}

}

// scanHTTPHeaders scans for HTTP security header issues.

func (ss *SecurityScanner) scanHTTPHeaders(ctx context.Context, target string) {

	ss.logger.Debug("Starting HTTP header scan", "target", target)

	// Make sure target is a full URL

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {

		target = "https://" + target

	}

	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)

	if err != nil {

		ss.logger.Error("Failed to create HTTP request", "error", err)

		return

	}

	// Set custom headers

	req.Header.Set("User-Agent", ss.config.UserAgent)

	for key, value := range ss.config.CustomHeaders {

		req.Header.Set(key, value)

	}

	resp, err := ss.httpClient.Do(req)

	if err != nil {

		ss.logger.Debug("HTTP request failed", "error", err, "target", target)

		return

	}

	defer resp.Body.Close()

	// Check for security headers

	ss.checkSecurityHeader(resp.Header, "Strict-Transport-Security", "HSTS", "High",

		"HSTS header is missing. This makes the site vulnerable to SSL stripping attacks.",

		"Add 'Strict-Transport-Security: max-age=31536000; includeSubDomains' header")

	ss.checkSecurityHeader(resp.Header, "Content-Security-Policy", "CSP", "High",

		"Content Security Policy header is missing. This increases XSS attack risk.",

		"Implement a restrictive Content-Security-Policy header")

	ss.checkSecurityHeader(resp.Header, "X-Frame-Options", "X-Frame-Options", "Medium",

		"X-Frame-Options header is missing. Site may be vulnerable to clickjacking.",

		"Add 'X-Frame-Options: DENY' or 'X-Frame-Options: SAMEORIGIN' header")

	ss.checkSecurityHeader(resp.Header, "X-Content-Type-Options", "X-Content-Type-Options", "Medium",

		"X-Content-Type-Options header is missing. Site may be vulnerable to MIME sniffing.",

		"Add 'X-Content-Type-Options: nosniff' header")

	ss.checkSecurityHeader(resp.Header, "Referrer-Policy", "Referrer-Policy", "Low",

		"Referrer-Policy header is missing. May leak sensitive information in referrer.",

		"Add 'Referrer-Policy: strict-origin-when-cross-origin' header")

	// Check for information disclosure headers

	ss.checkInformationDisclosure(resp.Header, "Server", "Server header reveals server information")

	ss.checkInformationDisclosure(resp.Header, "X-Powered-By", "X-Powered-By header reveals technology stack")

}

// checkSecurityHeader checks for the presence and validity of a security header.

func (ss *SecurityScanner) checkSecurityHeader(headers http.Header, headerName, issue, severity, description, recommendation string) {

	values := headers.Values(headerName)

	if len(values) == 0 {

		finding := HeaderFinding{

			Header:         headerName,

			Issue:          "MISSING_" + issue,

			Severity:       severity,

			Description:    description,

			Recommendation: recommendation,

			Present:        false,
		}

		ss.mutex.Lock()

		ss.results.HeaderFindings = append(ss.results.HeaderFindings, finding)

		ss.mutex.Unlock()

	} else {

		// Header is present, could validate its value here

		ss.logger.Debug("Security header present",

			"header", headerName,

			"value", values[0],

		)

	}

}

// checkInformationDisclosure checks for headers that might disclose sensitive information.

func (ss *SecurityScanner) checkInformationDisclosure(headers http.Header, headerName, description string) {

	values := headers.Values(headerName)

	if len(values) > 0 {

		finding := HeaderFinding{

			Header:         headerName,

			Issue:          "INFORMATION_DISCLOSURE",

			Severity:       "Low",

			Description:    description,

			Recommendation: fmt.Sprintf("Remove or obscure the %s header", headerName),

			Present:        true,

			Value:          values[0],
		}

		ss.mutex.Lock()

		ss.results.HeaderFindings = append(ss.results.HeaderFindings, finding)

		ss.mutex.Unlock()

	}

}

// scanInjectionVulns scans for common injection vulnerabilities.

func (ss *SecurityScanner) scanInjectionVulns(ctx context.Context, target string) {

	ss.logger.Debug("Starting injection vulnerability scan", "target", target)

	// Make sure target is a full URL

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {

		target = "https://" + target

	}

	// Common injection payloads

	sqlPayloads := []string{

		"' OR '1'='1",

		"' OR '1'='1' --",

		"'; DROP TABLE users; --",

		"1' UNION SELECT NULL,NULL,NULL--",

	}

	xssPayloads := []string{

		"<script>alert('XSS')</script>",

		"<img src=x onerror=alert('XSS')>",

		"javascript:alert('XSS')",

		"\"><script>alert('XSS')</script>",

	}

	commandPayloads := []string{

		"; ls -la",

		"| whoami",

		"; cat /etc/passwd",

		"`id`",

	}

	// Test SQL injection

	for _, payload := range sqlPayloads {

		ss.testInjection(ctx, target, payload, "SQL_INJECTION")

	}

	// Test XSS

	for _, payload := range xssPayloads {

		ss.testInjection(ctx, target, payload, "XSS")

	}

	// Test command injection

	for _, payload := range commandPayloads {

		ss.testInjection(ctx, target, payload, "COMMAND_INJECTION")

	}

}

// testInjection tests for a specific injection vulnerability.

func (ss *SecurityScanner) testInjection(ctx context.Context, baseURL, payload, injectionType string) {

	// Test various parameter positions

	testURLs := []string{

		baseURL + "?id=" + url.QueryEscape(payload),

		baseURL + "?search=" + url.QueryEscape(payload),

		baseURL + "?q=" + url.QueryEscape(payload),

	}

	for _, testURL := range testURLs {

		ss.testSingleInjection(ctx, testURL, payload, injectionType)

	}

}

// testSingleInjection tests a single injection attempt.

func (ss *SecurityScanner) testSingleInjection(ctx context.Context, testURL, payload, injectionType string) {

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)

	if err != nil {

		return

	}

	req.Header.Set("User-Agent", ss.config.UserAgent)

	resp, err := ss.httpClient.Do(req)

	if err != nil {

		return

	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	ss.analyzeInjectionResponse(testURL, "GET", payload, string(body), injectionType)

}

// analyzeInjectionResponse analyzes the response for signs of injection success.

func (ss *SecurityScanner) analyzeInjectionResponse(testURL, method, payload, responseBody, injectionType string) {

	var indicators []string

	var severity string

	switch injectionType {

	case "SQL_INJECTION":

		indicators = []string{

			"SQL syntax",

			"mysql_fetch",

			"ORA-",

			"PostgreSQL",

			"Warning: pg_",

			"valid MySQL result",

			"MySqlClient",

		}

		severity = "Critical"

	case "XSS":

		indicators = []string{

			"<script>alert('XSS')</script>",

			"alert('XSS')",

			"<img src=x onerror=alert('XSS')>",

		}

		severity = "High"

	case "COMMAND_INJECTION":

		indicators = []string{

			"root:x:",

			"uid=",

			"gid=",

			"groups=",

			"/bin/sh",

			"/bin/bash",

		}

		severity = "Critical"

	}

	for _, indicator := range indicators {

		if strings.Contains(responseBody, indicator) {

			// Extract parameter from URL

			u, err := url.Parse(testURL)

			if err != nil {

				continue

			}

			var parameter string

			for key := range u.Query() {

				parameter = key

				break // Take the first parameter

			}

			finding := InjectionFinding{

				Type:        injectionType,

				URL:         testURL,

				Parameter:   parameter,

				Payload:     payload,

				Method:      method,

				Severity:    severity,

				Description: fmt.Sprintf("Possible %s vulnerability detected", injectionType),

				Evidence:    indicator,
			}

			ss.mutex.Lock()

			ss.results.InjectionFindings = append(ss.results.InjectionFindings, finding)

			ss.mutex.Unlock()

			break

		}

	}

}

// scanVulnerabilities performs vulnerability scanning.

func (ss *SecurityScanner) scanVulnerabilities(ctx context.Context, host, port string) {

	ss.logger.Debug("Starting vulnerability scan", "host", host, "port", port)

	// This would typically integrate with vulnerability databases

	// For now, we'll check for some common vulnerabilities based on service detection

	ss.mutex.RLock()

	openPorts := ss.results.OpenPorts

	ss.mutex.RUnlock()

	for _, portInfo := range openPorts {

		ss.checkCommonVulnerabilities(host, portInfo)

	}

}

// checkCommonVulnerabilities checks for common vulnerabilities based on service type.

func (ss *SecurityScanner) checkCommonVulnerabilities(host string, portInfo PortInfo) {

	switch portInfo.Service {

	case "ssh":

		ss.checkSSHVulnerabilities(host, portInfo)

	case "http", "http-alt":

		ss.checkHTTPVulnerabilities(host, portInfo)

	case "https", "https-alt":

		ss.checkHTTPSVulnerabilities(host, portInfo)

	case "mysql":

		ss.checkMySQLVulnerabilities(host, portInfo)

	case "postgresql":

		ss.checkPostgreSQLVulnerabilities(host, portInfo)

	}

}

// checkSSHVulnerabilities checks for SSH-specific vulnerabilities.

func (ss *SecurityScanner) checkSSHVulnerabilities(host string, portInfo PortInfo) {

	// Check if SSH is running on default port (potential security issue)

	if portInfo.Port == 22 {

		vuln := Vulnerability{

			ID:          "SSH-001",

			Title:       "SSH on default port",

			Description: "SSH is running on the default port 22, making it an easy target for automated attacks",

			Severity:    "Low",

			Solution:    "Consider changing SSH port to a non-standard port",

			Port:        portInfo.Port,

			Service:     "ssh",

			References:  []string{"https://attack.mitre.org/techniques/T1021/"},
		}

		ss.mutex.Lock()

		ss.results.Vulnerabilities = append(ss.results.Vulnerabilities, vuln)

		ss.mutex.Unlock()

	}

}

// checkHTTPVulnerabilities checks for HTTP-specific vulnerabilities.

func (ss *SecurityScanner) checkHTTPVulnerabilities(host string, portInfo PortInfo) {

	// HTTP instead of HTTPS

	vuln := Vulnerability{

		ID:          "HTTP-001",

		Title:       "Insecure HTTP Protocol",

		Description: "Service is using HTTP instead of HTTPS, transmitting data in plain text",

		Severity:    "Medium",

		Solution:    "Implement HTTPS/TLS encryption",

		Port:        portInfo.Port,

		Service:     "http",

		References:  []string{"https://owasp.org/www-project-top-ten/2017/A3_2017-Sensitive_Data_Exposure"},
	}

	ss.mutex.Lock()

	ss.results.Vulnerabilities = append(ss.results.Vulnerabilities, vuln)

	ss.mutex.Unlock()

}

// checkHTTPSVulnerabilities checks for HTTPS-specific vulnerabilities.

func (ss *SecurityScanner) checkHTTPSVulnerabilities(host string, portInfo PortInfo) {

	// This would typically check for specific HTTPS vulnerabilities

	// like weak cipher suites, certificate issues, etc.

}

// checkMySQLVulnerabilities checks for MySQL-specific vulnerabilities.

func (ss *SecurityScanner) checkMySQLVulnerabilities(host string, portInfo PortInfo) {

	// Check if MySQL is accessible from external networks

	vuln := Vulnerability{

		ID:          "MYSQL-001",

		Title:       "MySQL Exposed to Network",

		Description: "MySQL database is accessible from external networks, potentially exposing sensitive data",

		Severity:    "High",

		Solution:    "Restrict MySQL access to localhost or specific trusted networks",

		Port:        portInfo.Port,

		Service:     "mysql",

		References:  []string{"https://dev.mysql.com/doc/refman/8.0/en/security-guidelines.html"},
	}

	ss.mutex.Lock()

	ss.results.Vulnerabilities = append(ss.results.Vulnerabilities, vuln)

	ss.mutex.Unlock()

}

// checkPostgreSQLVulnerabilities checks for PostgreSQL-specific vulnerabilities.

func (ss *SecurityScanner) checkPostgreSQLVulnerabilities(host string, portInfo PortInfo) {

	// Check if PostgreSQL is accessible from external networks

	vuln := Vulnerability{

		ID:          "PGSQL-001",

		Title:       "PostgreSQL Exposed to Network",

		Description: "PostgreSQL database is accessible from external networks, potentially exposing sensitive data",

		Severity:    "High",

		Solution:    "Restrict PostgreSQL access to localhost or specific trusted networks",

		Port:        portInfo.Port,

		Service:     "postgresql",

		References:  []string{"https://www.postgresql.org/docs/current/security.html"},
	}

	ss.mutex.Lock()

	ss.results.Vulnerabilities = append(ss.results.Vulnerabilities, vuln)

	ss.mutex.Unlock()

}

// calculateSummary calculates a summary of the scan results.

func (ss *SecurityScanner) calculateSummary() ScanSummary {

	var summary ScanSummary

	// Count vulnerabilities by severity

	for _, vuln := range ss.results.Vulnerabilities {

		summary.TotalVulns++

		switch vuln.Severity {

		case "Critical":

			summary.CriticalVulns++

		case "High":

			summary.HighVulns++

		case "Medium":

			summary.MediumVulns++

		case "Low":

			summary.LowVulns++

		}

	}

	// Count ports

	summary.TotalPorts = len(ss.results.OpenPorts)

	for _, port := range ss.results.OpenPorts {

		if port.State == "open" {

			summary.OpenPorts++

		}

	}

	// Count TLS issues

	summary.TLSIssues = len(ss.results.TLSFindings)

	// Count header issues

	summary.HeaderIssues = len(ss.results.HeaderFindings)

	// Count injection issues

	summary.InjectionIssues = len(ss.results.InjectionFindings)

	// Calculate security score (0-100)

	// Start with 100 and deduct points for issues

	score := 100

	score -= summary.CriticalVulns * 20 // Critical: -20 points each

	score -= summary.HighVulns * 10     // High: -10 points each

	score -= summary.MediumVulns * 5    // Medium: -5 points each

	score -= summary.LowVulns * 2       // Low: -2 points each

	score -= summary.TLSIssues * 5      // TLS issues: -5 points each

	score -= summary.HeaderIssues * 3   // Header issues: -3 points each

	score -= summary.InjectionIssues * 15 // Injection issues: -15 points each

	if score < 0 {

		score = 0

	}

	summary.SecurityScore = score

	// Determine risk level

	switch {

	case summary.CriticalVulns > 0:

		summary.RiskLevel = "Critical"

	case summary.HighVulns > 0 || summary.InjectionIssues > 0:

		summary.RiskLevel = "High"

	case summary.MediumVulns > 0 || summary.TLSIssues > 2:

		summary.RiskLevel = "Medium"

	default:

		summary.RiskLevel = "Low"

	}

	return summary

}

// ScanNetworkIntent scans a NetworkIntent resource for security issues.

func (ss *SecurityScanner) ScanNetworkIntent(ctx context.Context, intent *nephiov1.NetworkIntent) (*ScanResults, error) {

	ss.logger.Info("Scanning NetworkIntent for security issues",

		"intent", intent.Name,

		"namespace", intent.Namespace,

	)

	// Extract target endpoints from the NetworkIntent

	targets := ss.extractTargetsFromIntent(intent)

	if len(targets) == 0 {

		return nil, fmt.Errorf("no scan targets found in NetworkIntent")

	}

	// Perform scanning on the first target (could be extended to scan all)

	results, err := ss.ScanTarget(ctx, targets[0])

	if err != nil {

		return nil, fmt.Errorf("failed to scan NetworkIntent target: %w", err)

	}

	// Add NetworkIntent-specific findings

	ss.addIntentSecurityFindings(intent, results)

	return results, nil

}

// extractTargetsFromIntent extracts scannable targets from a NetworkIntent.
// Note: NetworkIntentSpec contains the following valid fields:
// - Intent, Description, IntentType, Priority
// - TargetComponents ([]ORANComponent), TargetNamespace, TargetCluster
// - NetworkSlice, Region, ResourceConstraints
// - ProcessedParameters (which contains SecurityParameters)
//
// CNFDeployments and Security are NOT direct fields in NetworkIntentSpec.
// Security configurations are available via ProcessedParameters.SecurityParameters.

func (ss *SecurityScanner) extractTargetsFromIntent(intent *nephiov1.NetworkIntent) []string {

	var targets []string

	// Extract targets based on valid NetworkIntentSpec fields

	// Extract targets from cluster information
	if intent.Spec.TargetCluster != "" {
		// Add cluster endpoint as a target for security scanning
		targets = append(targets, intent.Spec.TargetCluster)
	}

	// Add namespace-based targets if specified
	if intent.Spec.TargetNamespace != "" {
		// This could be expanded to scan specific services in the namespace
		targets = append(targets, "namespace:"+intent.Spec.TargetNamespace)
	}

	// Add targets based on O-RAN components
	for _, component := range intent.Spec.TargetComponents {
		// Create service endpoints based on O-RAN component types
		switch component {
		case nephiov1.ORANComponentAMF:
			targets = append(targets, "amf-service:80")
		case nephiov1.ORANComponentSMF:
			targets = append(targets, "smf-service:80")
		case nephiov1.ORANComponentUPF:
			targets = append(targets, "upf-service:80")
		case nephiov1.ORANComponentNearRTRIC:
			targets = append(targets, "near-rt-ric:80")
		default:
			// Generic service endpoint for other components
			targets = append(targets, string(component)+"-service:80")
		}
	}

	// If no specific targets found, add a default placeholder
	if len(targets) == 0 {
		targets = append(targets, "localhost:80")
	}

	return targets

}

// addIntentSecurityFindings adds NetworkIntent-specific security findings.
// This function safely accesses ProcessedParameters.SecurityParameters which is the 
// correct path for security configuration in NetworkIntentSpec.
//
// IMPORTANT: Direct .Security or .CNFDeployments fields do NOT exist in NetworkIntentSpec.
// All security configuration is accessed via ProcessedParameters.SecurityParameters.

func (ss *SecurityScanner) addIntentSecurityFindings(intent *nephiov1.NetworkIntent, results *ScanResults) {

	// Check for insecure configurations in the NetworkIntent
	// Safely check if ProcessedParameters and SecurityParameters exist

	if intent.Spec.ProcessedParameters == nil || intent.Spec.ProcessedParameters.SecurityParameters == nil {

		vuln := Vulnerability{

			ID:          "INTENT-001",

			Title:       "Missing Security Configuration",

			Description: "NetworkIntent does not specify security configuration in ProcessedParameters",

			Severity:    "Medium",

			Solution:    "Add security parameters to NetworkIntent.Spec.ProcessedParameters.SecurityParameters",

			Service:     "NetworkIntent",
		}

		results.Vulnerabilities = append(results.Vulnerabilities, vuln)

		// Log warning for debugging
		ss.logger.Warn("NetworkIntent missing security parameters", 
			"intent", intent.Name, 
			"namespace", intent.Namespace,
			"hasProcessedParams", intent.Spec.ProcessedParameters != nil)

	} else {

		secParams := intent.Spec.ProcessedParameters.SecurityParameters

		// Check specific security settings with safe dereferencing

		if secParams.TLSEnabled == nil || !*secParams.TLSEnabled {

			vuln := Vulnerability{

				ID:          "INTENT-002",

				Title:       "TLS Disabled or Not Configured",

				Description: "NetworkIntent has TLS disabled or not configured, communications may be unencrypted",

				Severity:    "High",

				Solution:    "Enable TLS in NetworkIntent.Spec.ProcessedParameters.SecurityParameters.TLSEnabled",

				Service:     "NetworkIntent",
			}

			results.Vulnerabilities = append(results.Vulnerabilities, vuln)

		}

		// Check if service mesh is disabled (which could indicate lack of security)

		if secParams.ServiceMesh == nil || !*secParams.ServiceMesh {

			vuln := Vulnerability{

				ID:          "INTENT-003",

				Title:       "Service Mesh Disabled",

				Description: "NetworkIntent has service mesh disabled, which may reduce security",

				Severity:    "Medium",

				Solution:    "Enable service mesh in NetworkIntent.Spec.ProcessedParameters.SecurityParameters.ServiceMesh",

				Service:     "NetworkIntent",
			}

			results.Vulnerabilities = append(results.Vulnerabilities, vuln)

		}

		// Check if encryption is properly configured with safe dereferencing

		if secParams.Encryption == nil || secParams.Encryption.Enabled == nil || !*secParams.Encryption.Enabled {

			vuln := Vulnerability{

				ID:          "INTENT-004",

				Title:       "Encryption Disabled",

				Description: "NetworkIntent has encryption disabled or not configured",

				Severity:    "High",

				Solution:    "Enable encryption in NetworkIntent.Spec.ProcessedParameters.SecurityParameters.Encryption.Enabled",

				Service:     "NetworkIntent",
			}

			results.Vulnerabilities = append(results.Vulnerabilities, vuln)

		}

		// Additional validation: Check network policies if available
		if len(secParams.NetworkPolicies) == 0 {
			vuln := Vulnerability{
				ID:          "INTENT-005",
				Title:       "No Network Policies Configured",
				Description: "NetworkIntent does not specify any network security policies",
				Severity:    "Medium",
				Solution:    "Add network policies to SecurityParameters.NetworkPolicies",
				Service:     "NetworkIntent",
			}
			results.Vulnerabilities = append(results.Vulnerabilities, vuln)
		}

	}

}

// ExportResults exports scan results in various formats.

func (ss *SecurityScanner) ExportResults(results *ScanResults, format string) ([]byte, error) {

	switch strings.ToLower(format) {

	case "json":

		return json.MarshalIndent(results, "", "  ")

	case "summary":

		return ss.generateSummaryReport(results), nil

	default:

		return nil, fmt.Errorf("unsupported export format: %s", format)

	}

}

// generateSummaryReport generates a human-readable summary report.

func (ss *SecurityScanner) generateSummaryReport(results *ScanResults) []byte {

	var report strings.Builder

	report.WriteString(fmt.Sprintf("Security Scan Report - %s\n", results.Target))

	report.WriteString(fmt.Sprintf("Scan Date: %s\n", results.Timestamp.Format("2006-01-02 15:04:05 UTC")))

	report.WriteString(fmt.Sprintf("Duration: %v\n\n", results.Duration))

	// Summary

	report.WriteString("SUMMARY\n")

	report.WriteString("=======\n")

	report.WriteString(fmt.Sprintf("Security Score: %d/100\n", results.Summary.SecurityScore))

	report.WriteString(fmt.Sprintf("Risk Level: %s\n", results.Summary.RiskLevel))

	report.WriteString(fmt.Sprintf("Open Ports: %d/%d\n", results.Summary.OpenPorts, results.Summary.TotalPorts))

	report.WriteString(fmt.Sprintf("Vulnerabilities: %d (Critical: %d, High: %d, Medium: %d, Low: %d)\n\n",

		results.Summary.TotalVulns, results.Summary.CriticalVulns,

		results.Summary.HighVulns, results.Summary.MediumVulns, results.Summary.LowVulns))

	// Open Ports

	if len(results.OpenPorts) > 0 {

		report.WriteString("OPEN PORTS\n")

		report.WriteString("==========\n")

		for _, port := range results.OpenPorts {

			report.WriteString(fmt.Sprintf("Port %d/%s - %s (%s)\n",

				port.Port, port.Protocol, port.Service, port.State))

		}

		report.WriteString("\n")

	}

	// Critical and High Vulnerabilities

	criticalHighVulns := []Vulnerability{}

	for _, vuln := range results.Vulnerabilities {

		if vuln.Severity == "Critical" || vuln.Severity == "High" {

			criticalHighVulns = append(criticalHighVulns, vuln)

		}

	}

	if len(criticalHighVulns) > 0 {

		report.WriteString("CRITICAL & HIGH SEVERITY VULNERABILITIES\n")

		report.WriteString("=======================================\n")

		for _, vuln := range criticalHighVulns {

			report.WriteString(fmt.Sprintf("[%s] %s\n", vuln.Severity, vuln.Title))

			report.WriteString(fmt.Sprintf("Description: %s\n", vuln.Description))

			report.WriteString(fmt.Sprintf("Solution: %s\n\n", vuln.Solution))

		}

	}

	return []byte(report.String())

}

// Stop stops the security scanner and cleans up resources.

func (ss *SecurityScanner) Stop() {

	ss.cancel()

}

// GetScanHistory returns historical scan results (placeholder for database integration).

func (ss *SecurityScanner) GetScanHistory(ctx context.Context, target string, limit int) ([]ScanResults, error) {

	// This would typically query a database for historical results

	// For now, return empty results

	return []ScanResults{}, nil

}

// ScheduleScan schedules a recurring security scan (placeholder for scheduler integration).

func (ss *SecurityScanner) ScheduleScan(target string, interval time.Duration) error {

	// This would integrate with a job scheduler

	ss.logger.Info("Scheduling recurring scan",

		"target", target,

		"interval", interval,

	)

	return nil

}

// ValidateConfiguration validates the security scanner configuration.

func ValidateConfiguration(config SecurityScannerConfig) error {

	if config.MaxConcurrency <= 0 || config.MaxConcurrency > 100 {

		return fmt.Errorf("MaxConcurrency must be between 1 and 100")

	}

	if config.ScanTimeout <= 0 || config.ScanTimeout > 30*time.Minute {

		return fmt.Errorf("ScanTimeout must be between 1s and 30m")

	}

	if config.HTTPTimeout <= 0 || config.HTTPTimeout > 5*time.Minute {

		return fmt.Errorf("HTTPTimeout must be between 1s and 5m")

	}

	for _, portRange := range config.PortRanges {

		if portRange.Start <= 0 || portRange.Start > 65535 {

			return fmt.Errorf("invalid port range start: %d", portRange.Start)

		}

		if portRange.End <= 0 || portRange.End > 65535 {

			return fmt.Errorf("invalid port range end: %d", portRange.End)

		}

		if portRange.Start > portRange.End {

			return fmt.Errorf("port range start (%d) cannot be greater than end (%d)", portRange.Start, portRange.End)

		}

	}

	return nil

}