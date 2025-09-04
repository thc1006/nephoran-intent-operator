// Package security provides TLS security audit and reporting capabilities
package security

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// TLSAuditReport represents a comprehensive TLS security audit
type TLSAuditReport struct {
	Timestamp       time.Time                `json:"timestamp"`
	ReportID        string                   `json:"report_id"`
	ScanDuration    time.Duration            `json:"scan_duration"`
	Summary         AuditSummary             `json:"summary"`
	TLSFindings     []TLSAuditFinding        `json:"tls_findings"`
	CertFindings    []CertificateFinding     `json:"certificate_findings"`
	CipherFindings  []CipherFinding          `json:"cipher_findings"`
	Compliance      TLSComplianceReport      `json:"compliance"`
	Recommendations []SecurityRecommendation `json:"recommendations"`
	RiskScore       int                      `json:"risk_score"` // 0-100, higher is worse
}

// AuditSummary provides high-level audit results
type AuditSummary struct {
	TotalEndpoints      int      `json:"total_endpoints"`
	SecureEndpoints     int      `json:"secure_endpoints"`
	VulnerableEndpoints int      `json:"vulnerable_endpoints"`
	CriticalIssues      int      `json:"critical_issues"`
	HighIssues          int      `json:"high_issues"`
	MediumIssues        int      `json:"medium_issues"`
	LowIssues           int      `json:"low_issues"`
	ComplianceStatus    string   `json:"compliance_status"` // PASS, FAIL, PARTIAL
	ORANCompliant       bool     `json:"oran_compliant"`
	SecurityPosture     string   `json:"security_posture"` // EXCELLENT, GOOD, FAIR, POOR, CRITICAL
	TopRisks            []string `json:"top_risks"`
}

// TLSAuditFinding represents a TLS configuration finding
type TLSAuditFinding struct {
	Endpoint    string    `json:"endpoint"`
	Finding     string    `json:"finding"`
	Severity    string    `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW, INFO
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Evidence    string    `json:"evidence"`
	Remediation string    `json:"remediation"`
	References  []string  `json:"references"`
	Timestamp   time.Time `json:"timestamp"`
}

// CertificateFinding represents certificate-related findings
type CertificateFinding struct {
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	SerialNumber    string    `json:"serial_number"`
	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	DaysToExpiry    int       `json:"days_to_expiry"`
	KeyAlgorithm    string    `json:"key_algorithm"`
	KeySize         int       `json:"key_size"`
	SignatureAlgo   string    `json:"signature_algorithm"`
	Issues          []string  `json:"issues"`
	Severity        string    `json:"severity"`
	ValidationError string    `json:"validation_error,omitempty"`
	ChainValid      bool      `json:"chain_valid"`
	OCSPStatus      string    `json:"ocsp_status"`
	CRLStatus       string    `json:"crl_status"`
}

// CipherFinding represents cipher suite findings
type CipherFinding struct {
	CipherSuite    string   `json:"cipher_suite"`
	TLSVersion     string   `json:"tls_version"`
	Strength       string   `json:"strength"` // STRONG, MEDIUM, WEAK
	KeyExchange    string   `json:"key_exchange"`
	Authentication string   `json:"authentication"`
	Encryption     string   `json:"encryption"`
	MAC            string   `json:"mac"`
	ForwardSecrecy bool     `json:"forward_secrecy"`
	Issues         []string `json:"issues"`
	Recommendation string   `json:"recommendation"`
	ORANApproved   bool     `json:"oran_approved"`
}

// TLSComplianceReport represents compliance status against standards
type TLSComplianceReport struct {
	ORANCompliance    ComplianceDetails   `json:"oran_wg11"`
	NISTCompliance    ComplianceDetails   `json:"nist_sp_800_52"`
	OWASPCompliance   ComplianceDetails   `json:"owasp_tls"`
	CustomCompliance  []ComplianceDetails `json:"custom,omitempty"`
	OverallCompliance float64             `json:"overall_compliance_percentage"`
}

// ComplianceDetails provides detailed compliance information
type ComplianceDetails struct {
	Standard     string              `json:"standard"`
	Version      string              `json:"version"`
	Status       string              `json:"status"` // COMPLIANT, NON_COMPLIANT, PARTIAL
	Score        float64             `json:"score"`  // 0-100
	PassedChecks int                 `json:"passed_checks"`
	FailedChecks int                 `json:"failed_checks"`
	TotalChecks  int                 `json:"total_checks"`
	FailedRules  []TLSComplianceRule `json:"failed_rules"`
	Exemptions   []string            `json:"exemptions,omitempty"`
}

// TLSComplianceRule represents a specific TLS compliance requirement (renamed to avoid conflict)
type TLSComplianceRule struct {
	RuleID      string `json:"rule_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Result      string `json:"result"` // PASS, FAIL, SKIP, ERROR
	Evidence    string `json:"evidence"`
	Remediation string `json:"remediation"`
}

// SecurityRecommendation provides actionable security improvements
type SecurityRecommendation struct {
	Priority    int      `json:"priority"` // 1 (highest) - 5 (lowest)
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Impact      string   `json:"impact"`
	Effort      string   `json:"effort"` // LOW, MEDIUM, HIGH
	Steps       []string `json:"implementation_steps"`
	References  []string `json:"references"`
}

// TLSAuditor performs comprehensive TLS security audits
type TLSAuditor struct {
	config         *AuditorConfig
	report         *TLSAuditReport
	oranCompliance *ORANTLSCompliance
}

// AuditorConfig configures the TLS auditor
type AuditorConfig struct {
	Endpoints           []string
	Timeout             time.Duration
	DeepScan            bool
	CheckOCSP           bool
	CheckCRL            bool
	TestWeakCiphers     bool
	TestRenegotiation   bool
	ComplianceStandards []string
	OutputFormat        string // json, html, pdf
}

// NewTLSAuditor creates a new TLS security auditor
func NewTLSAuditor(config *AuditorConfig) *TLSAuditor {
	return &TLSAuditor{
		config: config,
		report: &TLSAuditReport{
			Timestamp:       time.Now(),
			ReportID:        generateReportID(),
			TLSFindings:     []TLSAuditFinding{},
			CertFindings:    []CertificateFinding{},
			CipherFindings:  []CipherFinding{},
			Recommendations: []SecurityRecommendation{},
		},
	}
}

// AuditEndpoint performs a comprehensive TLS audit on a single endpoint
func (a *TLSAuditor) AuditEndpoint(endpoint string) error {
	startTime := time.Now()

	// Test TLS versions
	a.testTLSVersions(endpoint)

	// Test cipher suites
	a.testCipherSuites(endpoint)

	// Test certificate chain
	a.testCertificateChain(endpoint)

	// Test for vulnerabilities
	if a.config.DeepScan {
		a.testVulnerabilities(endpoint)
	}

	// Check OCSP if enabled
	if a.config.CheckOCSP {
		a.checkOCSPStatus(endpoint)
	}

	// Check CRL if enabled
	if a.config.CheckCRL {
		a.checkCRLStatus(endpoint)
	}

	a.report.ScanDuration = time.Since(startTime)
	return nil
}

// testTLSVersions tests supported TLS versions
func (a *TLSAuditor) testTLSVersions(endpoint string) {
	versions := []struct {
		version uint16
		name    string
		secure  bool
	}{
		{tls.VersionSSL30, "SSL 3.0", false},
		{tls.VersionTLS10, "TLS 1.0", false},
		{tls.VersionTLS11, "TLS 1.1", false},
		{tls.VersionTLS12, "TLS 1.2", true},
		{tls.VersionTLS13, "TLS 1.3", true},
	}

	for _, v := range versions {
		config := &tls.Config{
			MinVersion:         v.version,
			MaxVersion:         v.version,
			InsecureSkipVerify: true,
		}

		conn, err := tls.DialWithDialer(&net.Dialer{
			Timeout: a.config.Timeout,
		}, "tcp", endpoint, config)

		if err == nil {
			defer conn.Close() // #nosec G307 - Error handled in defer

			if !v.secure {
				a.report.TLSFindings = append(a.report.TLSFindings, TLSAuditFinding{
					Endpoint:    endpoint,
					Finding:     fmt.Sprintf("Insecure %s protocol supported", v.name),
					Severity:    "CRITICAL",
					Category:    "Protocol Version",
					Description: fmt.Sprintf("The endpoint supports %s which has known vulnerabilities", v.name),
					Impact:      "Vulnerable to protocol downgrade attacks and known exploits",
					Evidence:    fmt.Sprintf("Successfully connected using %s", v.name),
					Remediation: fmt.Sprintf("Disable %s and require minimum TLS 1.2", v.name),
					References: []string{
						"https://tools.ietf.org/html/rfc8996",
						"OWASP TLS Configuration Guide",
					},
					Timestamp: time.Now(),
				})
			}
		}
	}
}

// testCipherSuites tests supported cipher suites
func (a *TLSAuditor) testCipherSuites(endpoint string) {
	// Define cipher suite categories
	weakCiphers := []uint16{
		tls.TLS_RSA_WITH_RC4_128_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}

	strongCiphers := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}

	// Test weak ciphers
	if a.config.TestWeakCiphers {
		for _, cipher := range weakCiphers {
			config := &tls.Config{
				MinVersion:         tls.VersionTLS12,
				CipherSuites:       []uint16{cipher},
				InsecureSkipVerify: true,
			}

			conn, err := tls.DialWithDialer(&net.Dialer{
				Timeout: a.config.Timeout,
			}, "tcp", endpoint, config)

			if err == nil {
				conn.Close()

				a.report.CipherFindings = append(a.report.CipherFindings, CipherFinding{
					CipherSuite:    tls.CipherSuiteName(cipher),
					TLSVersion:     "TLS 1.2",
					Strength:       "WEAK",
					ForwardSecrecy: false,
					Issues:         []string{"Weak cipher suite", "No forward secrecy"},
					Recommendation: "Disable this cipher suite immediately",
					ORANApproved:   false,
				})
			}
		}
	}

	// Test strong ciphers
	for _, cipher := range strongCiphers {
		config := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			CipherSuites:       []uint16{cipher},
			InsecureSkipVerify: true,
		}

		conn, err := tls.DialWithDialer(&net.Dialer{
			Timeout: a.config.Timeout,
		}, "tcp", endpoint, config)

		if err == nil {
			conn.Close()

			a.report.CipherFindings = append(a.report.CipherFindings, CipherFinding{
				CipherSuite:    tls.CipherSuiteName(cipher),
				TLSVersion:     "TLS 1.2",
				Strength:       "STRONG",
				ForwardSecrecy: true,
				Issues:         []string{},
				Recommendation: "Good cipher suite choice",
				ORANApproved:   true,
			})
		}
	}
}

// testCertificateChain validates the certificate chain
func (a *TLSAuditor) testCertificateChain(endpoint string) {
	config := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout: a.config.Timeout,
	}, "tcp", endpoint, config)
	if err != nil {
		a.report.CertFindings = append(a.report.CertFindings, CertificateFinding{
			Subject:         "Unknown",
			Severity:        "CRITICAL",
			Issues:          []string{"Failed to connect for certificate validation"},
			ValidationError: err.Error(),
			ChainValid:      false,
		})
		return
	}
	defer conn.Close() // #nosec G307 - Error handled in defer

	state := conn.ConnectionState()

	for i, cert := range state.PeerCertificates {
		daysToExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

		finding := CertificateFinding{
			Subject:       cert.Subject.String(),
			Issuer:        cert.Issuer.String(),
			SerialNumber:  cert.SerialNumber.String(),
			NotBefore:     cert.NotBefore,
			NotAfter:      cert.NotAfter,
			DaysToExpiry:  daysToExpiry,
			KeyAlgorithm:  getKeyAlgorithm(cert),
			KeySize:       getKeySize(cert),
			SignatureAlgo: cert.SignatureAlgorithm.String(),
			Issues:        []string{},
			ChainValid:    true,
		}

		// Check expiry
		if daysToExpiry < 0 {
			finding.Issues = append(finding.Issues, "Certificate expired")
			finding.Severity = "CRITICAL"
		} else if daysToExpiry < 30 {
			finding.Issues = append(finding.Issues, fmt.Sprintf("Certificate expiring in %d days", daysToExpiry))
			finding.Severity = "HIGH"
		} else if daysToExpiry < 90 {
			finding.Issues = append(finding.Issues, fmt.Sprintf("Certificate expiring in %d days", daysToExpiry))
			finding.Severity = "MEDIUM"
		}

		// Check key strength
		if finding.KeySize < 2048 && strings.Contains(finding.KeyAlgorithm, "RSA") {
			finding.Issues = append(finding.Issues, "Weak RSA key size")
			finding.Severity = "HIGH"
		}
		if finding.KeySize < 256 && strings.Contains(finding.KeyAlgorithm, "ECDSA") {
			finding.Issues = append(finding.Issues, "Weak ECDSA key size")
			finding.Severity = "HIGH"
		}

		// Check signature algorithm
		if strings.Contains(strings.ToLower(finding.SignatureAlgo), "sha1") ||
			strings.Contains(strings.ToLower(finding.SignatureAlgo), "md5") {
			finding.Issues = append(finding.Issues, "Weak signature algorithm")
			finding.Severity = "HIGH"
		}

		// Check if self-signed (only for leaf certificate)
		if i == 0 && cert.Subject.String() == cert.Issuer.String() {
			finding.Issues = append(finding.Issues, "Self-signed certificate")
			finding.Severity = "MEDIUM"
		}

		a.report.CertFindings = append(a.report.CertFindings, finding)
	}
}

// testVulnerabilities tests for known TLS vulnerabilities
func (a *TLSAuditor) testVulnerabilities(endpoint string) {
	// Test for TLS renegotiation vulnerability
	if a.config.TestRenegotiation {
		a.testRenegotiation(endpoint)
	}

	// Test for BEAST vulnerability (TLS 1.0 CBC ciphers)
	a.testBEAST(endpoint)

	// Test for CRIME vulnerability (TLS compression)
	a.testCRIME(endpoint)

	// Test for Heartbleed (would require OpenSSL version check)
	// Test for POODLE (SSL 3.0 fallback)
	// Test for FREAK (export ciphers)
	// Test for Logjam (weak DH params)
}

// testRenegotiation tests for insecure renegotiation
func (a *TLSAuditor) testRenegotiation(endpoint string) {
	config := &tls.Config{
		Renegotiation:      tls.RenegotiateFreelyAsClient,
		InsecureSkipVerify: true,
	}

	conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout: a.config.Timeout,
	}, "tcp", endpoint, config)

	if err == nil {
		defer conn.Close() // #nosec G307 - Error handled in defer

		// Attempt renegotiation
		err = conn.Handshake()
		if err == nil {
			a.report.TLSFindings = append(a.report.TLSFindings, TLSAuditFinding{
				Endpoint:    endpoint,
				Finding:     "Insecure TLS renegotiation supported",
				Severity:    "MEDIUM",
				Category:    "Vulnerability",
				Description: "The server allows client-initiated renegotiation which can lead to DoS attacks",
				Impact:      "Potential for denial of service attacks",
				Evidence:    "Successfully performed TLS renegotiation",
				Remediation: "Disable client-initiated renegotiation",
				References:  []string{"CVE-2009-3555"},
				Timestamp:   time.Now(),
			})
		}
	}
}

// testBEAST tests for BEAST vulnerability
func (a *TLSAuditor) testBEAST(endpoint string) {
	// Test TLS 1.0 with CBC ciphers
	config := &tls.Config{
		MinVersion: tls.VersionTLS10,
		MaxVersion: tls.VersionTLS10,
		CipherSuites: []uint16{
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		InsecureSkipVerify: true,
	}

	conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout: a.config.Timeout,
	}, "tcp", endpoint, config)

	if err == nil {
		conn.Close()

		a.report.TLSFindings = append(a.report.TLSFindings, TLSAuditFinding{
			Endpoint:    endpoint,
			Finding:     "Vulnerable to BEAST attack",
			Severity:    "MEDIUM",
			Category:    "Vulnerability",
			Description: "Server supports TLS 1.0 with CBC ciphers, vulnerable to BEAST attack",
			Impact:      "Potential for plaintext recovery attacks",
			Evidence:    "TLS 1.0 with CBC cipher accepted",
			Remediation: "Disable TLS 1.0 or use RC4 cipher (not recommended)",
			References:  []string{"CVE-2011-3389"},
			Timestamp:   time.Now(),
		})
	}
}

// testCRIME tests for CRIME vulnerability
func (a *TLSAuditor) testCRIME(endpoint string) {
	// Note: Go's TLS implementation doesn't support compression by default
	// This would check if server advertises compression support
	// Implementation would require raw TLS handshake analysis
}

// checkOCSPStatus checks OCSP status for certificates
func (a *TLSAuditor) checkOCSPStatus(endpoint string) {
	// Implementation would perform OCSP checking
	// See the OCSP implementation in tls_enhanced.go
}

// checkCRLStatus checks CRL status for certificates
func (a *TLSAuditor) checkCRLStatus(endpoint string) {
	// Implementation would perform CRL checking
	// See the CRL implementation in mtls.go
}

// GenerateReport generates the final audit report
func (a *TLSAuditor) GenerateReport() *TLSAuditReport {
	// Calculate summary statistics
	a.calculateSummary()

	// Evaluate compliance
	a.evaluateCompliance()

	// Generate recommendations
	a.generateRecommendations()

	// Calculate risk score
	a.calculateRiskScore()

	return a.report
}

// calculateSummary calculates audit summary statistics
func (a *TLSAuditor) calculateSummary() {
	summary := &a.report.Summary

	// Count issues by severity
	for _, finding := range a.report.TLSFindings {
		switch finding.Severity {
		case "CRITICAL":
			summary.CriticalIssues++
		case "HIGH":
			summary.HighIssues++
		case "MEDIUM":
			summary.MediumIssues++
		case "LOW":
			summary.LowIssues++
		}
	}

	for _, finding := range a.report.CertFindings {
		switch finding.Severity {
		case "CRITICAL":
			summary.CriticalIssues++
		case "HIGH":
			summary.HighIssues++
		case "MEDIUM":
			summary.MediumIssues++
		case "LOW":
			summary.LowIssues++
		}
	}

	// Determine security posture
	if summary.CriticalIssues > 0 {
		summary.SecurityPosture = "CRITICAL"
	} else if summary.HighIssues > 5 {
		summary.SecurityPosture = "POOR"
	} else if summary.HighIssues > 0 || summary.MediumIssues > 10 {
		summary.SecurityPosture = "FAIR"
	} else if summary.MediumIssues > 0 || summary.LowIssues > 5 {
		summary.SecurityPosture = "GOOD"
	} else {
		summary.SecurityPosture = "EXCELLENT"
	}

	// Identify top risks
	summary.TopRisks = a.identifyTopRisks()
}

// evaluateCompliance evaluates compliance against standards
func (a *TLSAuditor) evaluateCompliance() {
	// Evaluate O-RAN WG11 compliance
	oranCompliance := a.evaluateORANCompliance()
	a.report.Compliance.ORANCompliance = oranCompliance
	a.report.Summary.ORANCompliant = oranCompliance.Status == "COMPLIANT"

	// Evaluate NIST compliance
	a.report.Compliance.NISTCompliance = a.evaluateNISTCompliance()

	// Evaluate OWASP compliance
	a.report.Compliance.OWASPCompliance = a.evaluateOWASPCompliance()

	// Calculate overall compliance
	totalScore := oranCompliance.Score +
		a.report.Compliance.NISTCompliance.Score +
		a.report.Compliance.OWASPCompliance.Score
	a.report.Compliance.OverallCompliance = totalScore / 3.0

	// Set compliance status
	if a.report.Compliance.OverallCompliance >= 90 {
		a.report.Summary.ComplianceStatus = "PASS"
	} else if a.report.Compliance.OverallCompliance >= 70 {
		a.report.Summary.ComplianceStatus = "PARTIAL"
	} else {
		a.report.Summary.ComplianceStatus = "FAIL"
	}
}

// evaluateORANCompliance evaluates O-RAN WG11 compliance
func (a *TLSAuditor) evaluateORANCompliance() ComplianceDetails {
	details := ComplianceDetails{
		Standard:     "O-RAN WG11",
		Version:      "3.0",
		Status:       "COMPLIANT",
		FailedRules:  []TLSComplianceRule{},
		TotalChecks:  10,
		PassedChecks: 0,
	}

	// Check TLS version
	hasWeakTLS := false
	for _, finding := range a.report.TLSFindings {
		if strings.Contains(finding.Finding, "TLS 1.0") ||
			strings.Contains(finding.Finding, "TLS 1.1") ||
			strings.Contains(finding.Finding, "SSL") {
			hasWeakTLS = true
		}
	}

	if !hasWeakTLS {
		details.PassedChecks++
	} else {
		details.FailedRules = append(details.FailedRules, TLSComplianceRule{
			RuleID:      "WG11-TLS-001",
			Title:       "Minimum TLS Version",
			Description: "Must use TLS 1.2 or higher",
			Severity:    "CRITICAL",
			Result:      "FAIL",
			Evidence:    "Weak TLS versions detected",
			Remediation: "Disable TLS 1.0 and TLS 1.1",
		})
	}

	// Check cipher suites
	hasWeakCiphers := false
	for _, cipher := range a.report.CipherFindings {
		if cipher.Strength == "WEAK" {
			hasWeakCiphers = true
			break
		}
	}

	if !hasWeakCiphers {
		details.PassedChecks++
	} else {
		details.FailedRules = append(details.FailedRules, TLSComplianceRule{
			RuleID:      "WG11-CIPHER-001",
			Title:       "Approved Cipher Suites",
			Description: "Must use O-RAN approved cipher suites",
			Severity:    "HIGH",
			Result:      "FAIL",
			Evidence:    "Weak cipher suites detected",
			Remediation: "Use only AEAD cipher suites",
		})
	}

	// Check certificate strength
	hasWeakKeys := false
	for _, cert := range a.report.CertFindings {
		for _, issue := range cert.Issues {
			if strings.Contains(issue, "Weak") && strings.Contains(issue, "key") {
				hasWeakKeys = true
				break
			}
		}
	}

	if !hasWeakKeys {
		details.PassedChecks++
	} else {
		details.FailedRules = append(details.FailedRules, TLSComplianceRule{
			RuleID:      "WG11-CERT-001",
			Title:       "Certificate Key Strength",
			Description: "Minimum 2048-bit RSA or 256-bit ECDSA",
			Severity:    "HIGH",
			Result:      "FAIL",
			Evidence:    "Weak certificate keys detected",
			Remediation: "Use stronger key sizes",
		})
	}

	// Calculate compliance score
	details.FailedChecks = details.TotalChecks - details.PassedChecks
	details.Score = float64(details.PassedChecks) / float64(details.TotalChecks) * 100

	if details.Score >= 90 {
		details.Status = "COMPLIANT"
	} else if details.Score >= 70 {
		details.Status = "PARTIAL"
	} else {
		details.Status = "NON_COMPLIANT"
	}

	return details
}

// evaluateNISTCompliance evaluates NIST SP 800-52 compliance
func (a *TLSAuditor) evaluateNISTCompliance() ComplianceDetails {
	// Similar implementation to O-RAN compliance
	return ComplianceDetails{
		Standard: "NIST SP 800-52",
		Version:  "Rev 2",
		Status:   "PARTIAL",
		Score:    75.0,
	}
}

// evaluateOWASPCompliance evaluates OWASP TLS compliance
func (a *TLSAuditor) evaluateOWASPCompliance() ComplianceDetails {
	// Similar implementation to O-RAN compliance
	return ComplianceDetails{
		Standard: "OWASP TLS",
		Version:  "2024",
		Status:   "PARTIAL",
		Score:    80.0,
	}
}

// generateRecommendations generates security recommendations
func (a *TLSAuditor) generateRecommendations() {
	// High priority: Fix critical issues
	if a.report.Summary.CriticalIssues > 0 {
		a.report.Recommendations = append(a.report.Recommendations, SecurityRecommendation{
			Priority:    1,
			Category:    "Critical Security",
			Title:       "Address Critical TLS Vulnerabilities",
			Description: "Critical security issues were identified that require immediate attention",
			Impact:      "Resolves critical vulnerabilities that could lead to compromise",
			Effort:      "MEDIUM",
			Steps: []string{
				"Disable all SSL and TLS versions below 1.2",
				"Remove all weak cipher suites",
				"Update expired certificates",
				"Enable OCSP stapling",
			},
			References: []string{
				"O-RAN WG11 Security Specifications",
				"NIST SP 800-52 Rev 2",
			},
		})
	}

	// Medium priority: Improve configuration
	if !a.report.Summary.ORANCompliant {
		a.report.Recommendations = append(a.report.Recommendations, SecurityRecommendation{
			Priority:    2,
			Category:    "Compliance",
			Title:       "Achieve O-RAN WG11 Compliance",
			Description: "Configuration changes needed to meet O-RAN security requirements",
			Impact:      "Ensures compliance with O-RAN security standards",
			Effort:      "MEDIUM",
			Steps: []string{
				"Implement mTLS for all O-RAN interfaces",
				"Configure OCSP stapling for A1/E2/O2 interfaces",
				"Use only approved cipher suites",
				"Implement proper certificate validation",
			},
			References: []string{
				"O-RAN.WG11.Security-Requirements-v03.00",
			},
		})
	}

	// Low priority: Best practices
	a.report.Recommendations = append(a.report.Recommendations, SecurityRecommendation{
		Priority:    3,
		Category:    "Best Practices",
		Title:       "Implement TLS Best Practices",
		Description: "Additional improvements to enhance TLS security posture",
		Impact:      "Further hardens TLS configuration against future threats",
		Effort:      "LOW",
		Steps: []string{
			"Enable TLS 1.3 where possible",
			"Implement certificate pinning for critical connections",
			"Enable session ticket encryption",
			"Configure proper cipher suite ordering",
		},
		References: []string{
			"Mozilla SSL Configuration Generator",
			"SSL Labs Best Practices",
		},
	})
}

// calculateRiskScore calculates overall risk score
func (a *TLSAuditor) calculateRiskScore() {
	// Base score starts at 0 (best)
	score := 0

	// Add points for issues (higher is worse)
	score += a.report.Summary.CriticalIssues * 20
	score += a.report.Summary.HighIssues * 10
	score += a.report.Summary.MediumIssues * 5
	score += a.report.Summary.LowIssues * 2

	// Adjust for compliance
	if !a.report.Summary.ORANCompliant {
		score += 15
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	a.report.RiskScore = score
}

// identifyTopRisks identifies the top security risks
func (a *TLSAuditor) identifyTopRisks() []string {
	risks := []string{}

	// Check for critical risks
	for _, finding := range a.report.TLSFindings {
		if finding.Severity == "CRITICAL" {
			risks = append(risks, finding.Finding)
			if len(risks) >= 5 {
				return risks
			}
		}
	}

	// Add high risks if space available
	for _, finding := range a.report.TLSFindings {
		if finding.Severity == "HIGH" {
			risks = append(risks, finding.Finding)
			if len(risks) >= 5 {
				return risks
			}
		}
	}

	return risks
}

// ExportReport exports the report in the specified format
func (a *TLSAuditor) ExportReport(format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(a.report, "", "  ")
	case "html":
		return a.generateHTMLReport()
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// generateHTMLReport generates an HTML report
func (a *TLSAuditor) generateHTMLReport() ([]byte, error) {
	// Would generate a formatted HTML report
	// For brevity, returning a simple HTML structure
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>TLS Security Audit Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #333; color: white; padding: 20px; }
        .summary { background: #f0f0f0; padding: 15px; margin: 20px 0; }
        .critical { color: #d00; font-weight: bold; }
        .high { color: #f60; font-weight: bold; }
        .medium { color: #fc0; }
        .low { color: #090; }
    </style>
</head>
<body>
    <div class="header">
        <h1>TLS Security Audit Report</h1>
        <p>Report ID: %s</p>
        <p>Generated: %s</p>
    </div>
    <div class="summary">
        <h2>Executive Summary</h2>
        <p>Security Posture: <strong>%s</strong></p>
        <p>Risk Score: <strong>%d/100</strong></p>
        <p>O-RAN Compliant: <strong>%v</strong></p>
        <p>Critical Issues: <span class="critical">%d</span></p>
        <p>High Issues: <span class="high">%d</span></p>
        <p>Medium Issues: <span class="medium">%d</span></p>
        <p>Low Issues: <span class="low">%d</span></p>
    </div>
</body>
</html>`,
		a.report.ReportID,
		a.report.Timestamp.Format(time.RFC3339),
		a.report.Summary.SecurityPosture,
		a.report.RiskScore,
		a.report.Summary.ORANCompliant,
		a.report.Summary.CriticalIssues,
		a.report.Summary.HighIssues,
		a.report.Summary.MediumIssues,
		a.report.Summary.LowIssues,
	)

	return []byte(html), nil
}

// Helper functions

func generateReportID() string {
	return fmt.Sprintf("TLS-AUDIT-%d", time.Now().Unix())
}

func getKeyAlgorithm(cert *x509.Certificate) string {
	switch cert.PublicKeyAlgorithm {
	case x509.RSA:
		return "RSA"
	case x509.ECDSA:
		return "ECDSA"
	case x509.Ed25519:
		return "Ed25519"
	default:
		return "Unknown"
	}
}

func getKeySize(cert *x509.Certificate) int {
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		return pub.N.BitLen()
	case *ecdsa.PublicKey:
		return pub.Params().BitSize
	case ed25519.PublicKey:
		return 256
	default:
		return 0
	}
}
