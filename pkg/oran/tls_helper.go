package oran

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"
)

// TLSConfig represents O-RAN WG11 compliant TLS configuration
// COMPLIANCE: O-RAN WG11 Interface Security, NIST PR.DS-2, GDPR Art. 32
type TLSConfig struct {
	// Certificate files - COMPLIANCE: X.509 certificate management
	CertFile string `json:"cert_file" validate:"file"`
	KeyFile  string `json:"key_file" validate:"file"`
	CAFile   string `json:"ca_file" validate:"file"`
	
	// Security settings - COMPLIANCE: O-RAN WG11 security requirements
	SkipVerify     bool   `json:"skip_verify" default:"false"`
	ServerName     string `json:"server_name,omitempty"`
	MinTLSVersion  string `json:"min_tls_version" validate:"oneof=TLS1.2 TLS1.3" default:"TLS1.3"`
	MaxTLSVersion  string `json:"max_tls_version" validate:"oneof=TLS1.2 TLS1.3" default:"TLS1.3"`
	
	// O-RAN interface specific settings
	InterfaceType  string `json:"interface_type" validate:"oneof=E2 A1 O1 O2 OpenFronthaul"`
	MutualTLS      bool   `json:"mutual_tls" default:"true"`
	ClientAuth     string `json:"client_auth" validate:"oneof=NoClientCert RequestClientCert RequireAnyClientCert VerifyClientCertIfGiven RequireAndVerifyClientCert" default:"RequireAndVerifyClientCert"`
	
	// FIPS compliance - COMPLIANCE: FIPS 140-3 Level 2
	FIPSMode       bool     `json:"fips_mode" default:"true"`
	AllowedCiphers []string `json:"allowed_ciphers,omitempty"`
	NextProtos     []string `json:"next_protos,omitempty"`
	
	// Security hardening
	SessionTicketsDisabled bool `json:"session_tickets_disabled" default:"true"`
	RenegotiationSupport   int  `json:"renegotiation_support" validate:"oneof=0 1 2" default:"0"` // 0=Never, 1=OnceAsClient, 2=FreelyAsClient
	
	// Compliance tracking
	ComplianceMode string                 `json:"compliance_mode" validate:"oneof=strict permissive" default:"strict"`
	AuditLogging   bool                   `json:"audit_logging" default:"true"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// BuildTLSConfig builds a *tls.Config from the provided TLSConfig
// following O-RAN security requirements and best practices
// COMPLIANCE: O-RAN WG11, NIST PR.DS-2, FIPS 140-3
func BuildTLSConfig(config *TLSConfig) (*tls.Config, error) {
	if config == nil {
		return nil, fmt.Errorf("TLS configuration is required for O-RAN WG11 compliance")
	}

	// COMPLIANCE: Validate configuration first
	if err := ValidateTLSConfig(config); err != nil {
		return nil, fmt.Errorf("TLS configuration validation failed: %w", err)
	}

	// COMPLIANCE: O-RAN WG11 - TLS 1.3 only for enhanced security
	minVersion := tls.VersionTLS13
	maxVersion := tls.VersionTLS13
	
	// Allow TLS 1.2 only in permissive compliance mode
	if config.ComplianceMode == "permissive" && config.MinTLSVersion == "TLS1.2" {
		minVersion = tls.VersionTLS12
	}

	tlsConfig := &tls.Config{
		// COMPLIANCE: Enforce TLS 1.3 for O-RAN security
		MinVersion: uint16(minVersion),
		MaxVersion: uint16(maxVersion),
		
		// COMPLIANCE: O-RAN WG11 - Strict certificate verification
		InsecureSkipVerify: config.SkipVerify && config.ComplianceMode == "permissive",
		ServerName:         config.ServerName,
		
		// COMPLIANCE: Enhanced security measures
		PreferServerCipherSuites: true,
		
		// COMPLIANCE: HTTP/2 support for modern protocols
		NextProtos: getNextProtocolsForInterface(config.InterfaceType),
		
		// COMPLIANCE: Disable session tickets for forward secrecy
		SessionTicketsDisabled: config.SessionTicketsDisabled,
		
		// COMPLIANCE: Renegotiation security
		Renegotiation: tls.RenegotiateNever,
	}

	// COMPLIANCE: Set appropriate renegotiation support
	switch config.RenegotiationSupport {
	case 1:
		tlsConfig.Renegotiation = tls.RenegotiateOnceAsClient
	case 2:
		tlsConfig.Renegotiation = tls.RenegotiateFreelyAsClient
	default:
		tlsConfig.Renegotiation = tls.RenegotiateNever
	}

	// COMPLIANCE: FIPS 140-3 cipher suite configuration
	if config.FIPSMode {
		tlsConfig.CipherSuites = getFIPSCompliantCipherSuites()
	} else if len(config.AllowedCiphers) > 0 {
		cipherSuites, err := parseCipherSuites(config.AllowedCiphers)
		if err != nil {
			return nil, fmt.Errorf("invalid cipher suites: %w", err)
		}
		tlsConfig.CipherSuites = cipherSuites
	}

	// COMPLIANCE: Load client certificate and key for mutual TLS
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate and key: %w", err)
		}
		
		// COMPLIANCE: Validate certificate compliance
		if err := validateCertificateCompliance(&cert, config); err != nil {
			return nil, fmt.Errorf("certificate failed compliance validation: %w", err)
		}
		
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// COMPLIANCE: Load CA certificate for server verification
	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
		
		// COMPLIANCE: Set client CA for mutual authentication
		if config.MutualTLS {
			tlsConfig.ClientCAs = caCertPool
		}
	}

	// COMPLIANCE: Set client authentication mode
	if config.MutualTLS {
		clientAuth := parseClientAuthType(config.ClientAuth)
		tlsConfig.ClientAuth = clientAuth
		
		// COMPLIANCE: O-RAN WG11 requires mutual TLS for most interfaces
		if isInterfaceRequiringMutualTLS(config.InterfaceType) && clientAuth < tls.RequireAndVerifyClientCert {
			return nil, fmt.Errorf("interface %s requires mutual TLS with client certificate verification", config.InterfaceType)
		}
	}

	// COMPLIANCE: Set up certificate verification callback
	if config.ComplianceMode == "strict" {
		tlsConfig.VerifyPeerCertificate = createStrictCertificateVerifier(config)
		tlsConfig.VerifyConnection = createConnectionVerifier(config)
	}

	return tlsConfig, nil
}

// ValidateTLSConfig validates the TLS configuration parameters against compliance requirements
// COMPLIANCE: O-RAN WG11, NIST configuration validation
func ValidateTLSConfig(config *TLSConfig) error {
	if config == nil {
		return fmt.Errorf("TLS configuration cannot be nil")
	}

	// COMPLIANCE: O-RAN interface validation
	if !isValidORANInterface(config.InterfaceType) {
		return fmt.Errorf("invalid O-RAN interface type: %s", config.InterfaceType)
	}

	// COMPLIANCE: Strict mode validations
	if config.ComplianceMode == "strict" {
		// COMPLIANCE: TLS 1.3 requirement in strict mode
		if config.MinTLSVersion != "TLS1.3" {
			return fmt.Errorf("strict compliance mode requires TLS 1.3 minimum version")
		}
		
		// COMPLIANCE: Certificate verification required in strict mode
		if config.SkipVerify {
			return fmt.Errorf("strict compliance mode cannot skip certificate verification")
		}
		
		// COMPLIANCE: Mutual TLS requirement for security-critical interfaces
		if isInterfaceRequiringMutualTLS(config.InterfaceType) && !config.MutualTLS {
			return fmt.Errorf("interface %s requires mutual TLS in strict compliance mode", config.InterfaceType)
		}
	}

	// Check if certificate and key files exist when specified
	if config.CertFile != "" {
		if _, err := os.Stat(config.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("certificate file does not exist: %s", config.CertFile)
		}
	}

	if config.KeyFile != "" {
		if _, err := os.Stat(config.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("key file does not exist: %s", config.KeyFile)
		}
	}

	if config.CAFile != "" {
		if _, err := os.Stat(config.CAFile); os.IsNotExist(err) {
			return fmt.Errorf("CA file does not exist: %s", config.CAFile)
		}
	}

	// COMPLIANCE: Ensure both cert and key are provided together
	if (config.CertFile != "" && config.KeyFile == "") || (config.CertFile == "" && config.KeyFile != "") {
		return fmt.Errorf("both certificate file and key file must be provided for mutual TLS")
	}

	// COMPLIANCE: Validate cipher suites if specified
	if len(config.AllowedCiphers) > 0 {
		_, err := parseCipherSuites(config.AllowedCiphers)
		if err != nil {
			return fmt.Errorf("invalid cipher suite configuration: %w", err)
		}
	}

	return nil
}

// GetRecommendedTLSConfig returns a TLS configuration with O-RAN recommended security settings
// COMPLIANCE: O-RAN WG11 security baseline configuration
func GetRecommendedTLSConfig(certFile, keyFile, caFile, interfaceType string) *TLSConfig {
	return &TLSConfig{
		CertFile:      certFile,
		KeyFile:       keyFile,
		CAFile:        caFile,
		InterfaceType: interfaceType,
		SkipVerify:    false, // COMPLIANCE: Always verify certificates
		MinTLSVersion: "TLS1.3",
		MaxTLSVersion: "TLS1.3",
		MutualTLS:     true,  // COMPLIANCE: Enable mutual TLS by default
		ClientAuth:    "RequireAndVerifyClientCert",
		FIPSMode:      true,  // COMPLIANCE: Enable FIPS mode by default
		SessionTicketsDisabled: true,
		RenegotiationSupport:   0, // Never allow renegotiation
		ComplianceMode: "strict",
		AuditLogging:   true,
		Metadata: map[string]interface{}{
			"created_at":     time.Now().Format(time.RFC3339),
			"compliance":     "O-RAN WG11",
			"security_level": "enhanced",
		},
	}
}

// GetInterfaceSpecificTLSConfig returns TLS configuration tailored for specific O-RAN interfaces
// COMPLIANCE: Interface-specific security requirements
func GetInterfaceSpecificTLSConfig(interfaceType, certFile, keyFile, caFile string) *TLSConfig {
	baseConfig := GetRecommendedTLSConfig(certFile, keyFile, caFile, interfaceType)
	
	switch interfaceType {
	case "E2":
		// COMPLIANCE: E2 interface requires enhanced security
		baseConfig.Metadata["port"] = 36421
		baseConfig.Metadata["protocol"] = "SCTP/TCP"
		baseConfig.Metadata["security_requirements"] = "mutual_tls_mandatory"
		
	case "A1":
		// COMPLIANCE: A1 interface with OAuth2 integration
		baseConfig.NextProtos = []string{"h2", "http/1.1"}
		baseConfig.Metadata["port"] = 9001
		baseConfig.Metadata["protocol"] = "HTTP/2"
		baseConfig.Metadata["additional_auth"] = "oauth2"
		
	case "O1":
		// COMPLIANCE: O1 interface for management
		baseConfig.Metadata["port"] = 830
		baseConfig.Metadata["protocol"] = "NETCONF"
		baseConfig.Metadata["additional_protocols"] = []string{"RESTCONF", "SSH"}
		
	case "O2":
		// COMPLIANCE: O2 interface for infrastructure management
		baseConfig.Metadata["port"] = 443
		baseConfig.Metadata["protocol"] = "HTTPS"
		baseConfig.Metadata["container_runtime"] = "kubernetes"
		
	case "OpenFronthaul":
		// COMPLIANCE: Open Fronthaul interface
		baseConfig.Metadata["ports"] = []int{7777, 8888}
		baseConfig.Metadata["protocol"] = "eCPRI"
		baseConfig.Metadata["timing_requirements"] = "strict"
	}
	
	return baseConfig
}

// =============================================================================
// Private Helper Functions
// =============================================================================

func getFIPSCompliantCipherSuites() []uint16 {
	// COMPLIANCE: FIPS 140-3 approved cipher suites for TLS 1.3
	return []uint16{
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_AES_128_GCM_SHA256,
	}
}

func parseCipherSuites(cipherNames []string) ([]uint16, error) {
	var cipherSuites []uint16
	
	cipherMap := map[string]uint16{
		"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,
		// TLS 1.2 cipher suites (for permissive mode)
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}
	
	for _, name := range cipherNames {
		if suite, exists := cipherMap[name]; exists {
			cipherSuites = append(cipherSuites, suite)
		} else {
			return nil, fmt.Errorf("unknown cipher suite: %s", name)
		}
	}
	
	return cipherSuites, nil
}

func parseClientAuthType(authType string) tls.ClientAuthType {
	switch authType {
	case "NoClientCert":
		return tls.NoClientCert
	case "RequestClientCert":
		return tls.RequestClientCert
	case "RequireAnyClientCert":
		return tls.RequireAnyClientCert
	case "VerifyClientCertIfGiven":
		return tls.VerifyClientCertIfGiven
	case "RequireAndVerifyClientCert":
		return tls.RequireAndVerifyClientCert
	default:
		return tls.RequireAndVerifyClientCert // COMPLIANCE: Default to strongest authentication
	}
}

func getNextProtocolsForInterface(interfaceType string) []string {
	switch interfaceType {
	case "A1":
		return []string{"h2", "http/1.1"} // HTTP/2 preferred for A1
	case "O1":
		return []string{"netconf", "ssh"} // NETCONF over SSH for O1
	case "O2":
		return []string{"h2", "http/1.1"} // HTTP/2 for O2 APIs
	default:
		return []string{"h2", "http/1.1"} // Default to HTTP/2
	}
}

func isValidORANInterface(interfaceType string) bool {
	validInterfaces := []string{"E2", "A1", "O1", "O2", "OpenFronthaul"}
	for _, valid := range validInterfaces {
		if interfaceType == valid {
			return true
		}
	}
	return false
}

func isInterfaceRequiringMutualTLS(interfaceType string) bool {
	// COMPLIANCE: These interfaces require mutual TLS for security
	criticalInterfaces := []string{"E2", "O1", "O2"}
	for _, critical := range criticalInterfaces {
		if interfaceType == critical {
			return true
		}
	}
	return false
}

func validateCertificateCompliance(cert *tls.Certificate, config *TLSConfig) error {
	if len(cert.Certificate) == 0 {
		return fmt.Errorf("certificate chain is empty")
	}
	
	// Parse the leaf certificate
	leafCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse leaf certificate: %w", err)
	}
	
	// COMPLIANCE: Certificate validity checks
	now := time.Now()
	if now.Before(leafCert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid")
	}
	if now.After(leafCert.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}
	
	// COMPLIANCE: Key size validation for security
	switch leafCert.PublicKeyAlgorithm {
	case x509.RSA:
		if rsaKey, ok := leafCert.PublicKey.(*rsa.PublicKey); ok {
			keySize := rsaKey.Size() * 8
			minKeySize := 2048
			if config.FIPSMode {
				minKeySize = 3072 // FIPS requirement
			}
			if keySize < minKeySize {
				return fmt.Errorf("RSA key size %d bits below minimum required %d bits", keySize, minKeySize)
			}
		}
	case x509.ECDSA:
		// ECDSA certificates are acceptable
	default:
		return fmt.Errorf("unsupported public key algorithm: %v", leafCert.PublicKeyAlgorithm)
	}
	
	// COMPLIANCE: Check for critical extensions in strict mode
	if config.ComplianceMode == "strict" {
		if !leafCert.BasicConstraintsValid {
			return fmt.Errorf("certificate missing basic constraints extension")
		}
		
		// Check for key usage extension
		if leafCert.KeyUsage == 0 {
			return fmt.Errorf("certificate missing key usage extension")
		}
	}
	
	return nil
}

func createStrictCertificateVerifier(config *TLSConfig) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if len(verifiedChains) == 0 {
			return fmt.Errorf("no verified certificate chains")
		}
		
		// Additional compliance checks can be added here
		for _, chain := range verifiedChains {
			for _, cert := range chain {
				// COMPLIANCE: Check certificate attributes
				if config.FIPSMode {
					// Verify FIPS-compliant signature algorithms
					if !isFIPSCompliantSignatureAlgorithm(cert.SignatureAlgorithm) {
						return fmt.Errorf("certificate uses non-FIPS compliant signature algorithm: %v", cert.SignatureAlgorithm)
					}
				}
			}
		}
		
		return nil
	}
}

func createConnectionVerifier(config *TLSConfig) func(cs tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		// COMPLIANCE: Verify connection state meets security requirements
		if config.FIPSMode {
			if !isFIPSCompliantCipherSuite(cs.CipherSuite) {
				return fmt.Errorf("connection using non-FIPS compliant cipher suite")
			}
		}
		
		// COMPLIANCE: Verify TLS version
		if cs.Version < tls.VersionTLS12 {
			return fmt.Errorf("connection using insecure TLS version: %d", cs.Version)
		}
		
		if config.ComplianceMode == "strict" && cs.Version < tls.VersionTLS13 {
			return fmt.Errorf("strict mode requires TLS 1.3, got version: %d", cs.Version)
		}
		
		return nil
	}
}

func isFIPSCompliantSignatureAlgorithm(alg x509.SignatureAlgorithm) bool {
	fipsCompliantAlgorithms := []x509.SignatureAlgorithm{
		x509.SHA256WithRSA,
		x509.SHA384WithRSA,
		x509.SHA512WithRSA,
		x509.ECDSAWithSHA256,
		x509.ECDSAWithSHA384,
		x509.ECDSAWithSHA512,
	}
	
	for _, compliant := range fipsCompliantAlgorithms {
		if alg == compliant {
			return true
		}
	}
	
	return false
}

func isFIPSCompliantCipherSuite(cipherSuite uint16) bool {
	fipsCompliantSuites := getFIPSCompliantCipherSuites()
	
	for _, compliant := range fipsCompliantSuites {
		if cipherSuite == compliant {
			return true
		}
	}
	
	// Additional TLS 1.2 FIPS compliant suites
	tls12FIPSCompliant := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}
	
	for _, compliant := range tls12FIPSCompliant {
		if cipherSuite == compliant {
			return true
		}
	}
	
	return false
}

// =============================================================================
// Advanced TLS Configuration Functions
// =============================================================================

// CreateSecureTLSListener creates a TLS listener with O-RAN compliance settings
// COMPLIANCE: O-RAN WG11 server-side security configuration
func CreateSecureTLSListener(network, address string, config *TLSConfig) (net.Listener, error) {
	tlsConfig, err := BuildTLSConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS configuration: %w", err)
	}
	
	listener, err := tls.Listen(network, address, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS listener: %w", err)
	}
	
	return listener, nil
}

// CreateSecureTLSDialer creates a TLS dialer with O-RAN compliance settings
// COMPLIANCE: O-RAN WG11 client-side security configuration
func CreateSecureTLSDialer(config *TLSConfig) (*tls.Dialer, error) {
	tlsConfig, err := BuildTLSConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS configuration: %w", err)
	}
	
	dialer := &tls.Dialer{
		Config: tlsConfig,
		NetDialer: &net.Dialer{
			Timeout: 30 * time.Second, // COMPLIANCE: Reasonable timeout for connections
		},
	}
	
	return dialer, nil
}

// TLSConfigAuditor provides audit capabilities for TLS configurations
type TLSConfigAuditor struct {
	config     *TLSConfig
	auditTrail []TLSAuditEvent
}

type TLSAuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Interface   string                 `json:"interface"`
	Description string                 `json:"description"`
	Compliance  map[string]bool        `json:"compliance"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewTLSConfigAuditor creates a new TLS configuration auditor
func NewTLSConfigAuditor(config *TLSConfig) *TLSConfigAuditor {
	return &TLSConfigAuditor{
		config:     config,
		auditTrail: make([]TLSAuditEvent, 0),
	}
}

// AuditConfiguration performs comprehensive compliance audit
func (auditor *TLSConfigAuditor) AuditConfiguration() (*TLSComplianceReport, error) {
	report := &TLSComplianceReport{
		Timestamp:     time.Now(),
		Interface:     auditor.config.InterfaceType,
		ComplianceMode: auditor.config.ComplianceMode,
		Checks:        make([]ComplianceCheck, 0),
		OverallScore:  0.0,
	}
	
	// Run compliance checks
	checks := []struct {
		name     string
		weight   float64
		checkFunc func() (bool, string)
	}{
		{"TLS Version Compliance", 0.25, auditor.checkTLSVersionCompliance},
		{"Certificate Validation", 0.20, auditor.checkCertificateCompliance},
		{"Cipher Suite Security", 0.20, auditor.checkCipherSuiteCompliance},
		{"Mutual TLS Configuration", 0.15, auditor.checkMutualTLSCompliance},
		{"FIPS Mode Compliance", 0.10, auditor.checkFIPSCompliance},
		{"Interface-Specific Requirements", 0.10, auditor.checkInterfaceCompliance},
	}
	
	totalWeight := 0.0
	scoreSum := 0.0
	
	for _, check := range checks {
		passed, details := check.checkFunc()
		complianceCheck := ComplianceCheck{
			Name:        check.name,
			Passed:      passed,
			Weight:      check.weight,
			Details:     details,
			Timestamp:   time.Now(),
		}
		
		report.Checks = append(report.Checks, complianceCheck)
		
		if passed {
			scoreSum += check.weight
		}
		totalWeight += check.weight
	}
	
	report.OverallScore = (scoreSum / totalWeight) * 100.0
	report.CompliantStatus = report.OverallScore >= 90.0
	
	// Log audit event
	auditor.auditTrail = append(auditor.auditTrail, TLSAuditEvent{
		Timestamp:   time.Now(),
		EventType:   "compliance_audit",
		Interface:   auditor.config.InterfaceType,
		Description: fmt.Sprintf("TLS compliance audit completed with score: %.2f%%", report.OverallScore),
		Compliance: map[string]bool{
			"overall_compliant": report.CompliantStatus,
			"fips_enabled":     auditor.config.FIPSMode,
			"mutual_tls":       auditor.config.MutualTLS,
		},
		Metadata: map[string]interface{}{
			"compliance_mode": auditor.config.ComplianceMode,
			"interface_type": auditor.config.InterfaceType,
			"total_checks":   len(checks),
			"passed_checks":  len(report.Checks),
		},
	})
	
	return report, nil
}

type TLSComplianceReport struct {
	Timestamp      time.Time         `json:"timestamp"`
	Interface      string            `json:"interface"`
	ComplianceMode string            `json:"compliance_mode"`
	Checks         []ComplianceCheck `json:"checks"`
	OverallScore   float64           `json:"overall_score"`
	CompliantStatus bool             `json:"compliant_status"`
}

type ComplianceCheck struct {
	Name      string    `json:"name"`
	Passed    bool      `json:"passed"`
	Weight    float64   `json:"weight"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}

// Compliance check functions
func (auditor *TLSConfigAuditor) checkTLSVersionCompliance() (bool, string) {
	if auditor.config.ComplianceMode == "strict" {
		if auditor.config.MinTLSVersion == "TLS1.3" && auditor.config.MaxTLSVersion == "TLS1.3" {
			return true, "TLS 1.3 enforced in strict compliance mode"
		}
		return false, "Strict compliance mode requires TLS 1.3"
	}
	
	if auditor.config.MinTLSVersion == "TLS1.2" || auditor.config.MinTLSVersion == "TLS1.3" {
		return true, fmt.Sprintf("TLS version %s meets minimum requirements", auditor.config.MinTLSVersion)
	}
	
	return false, "TLS version below minimum security requirements"
}

func (auditor *TLSConfigAuditor) checkCertificateCompliance() (bool, string) {
	if auditor.config.CertFile == "" || auditor.config.KeyFile == "" {
		return false, "Certificate and key files not configured"
	}
	
	if auditor.config.SkipVerify && auditor.config.ComplianceMode == "strict" {
		return false, "Certificate verification disabled in strict compliance mode"
	}
	
	return true, "Certificate configuration meets compliance requirements"
}

func (auditor *TLSConfigAuditor) checkCipherSuiteCompliance() (bool, string) {
	if auditor.config.FIPSMode {
		return true, "FIPS mode enabled with compliant cipher suites"
	}
	
	if len(auditor.config.AllowedCiphers) > 0 {
		// Check if allowed ciphers are secure
		for _, cipher := range auditor.config.AllowedCiphers {
			if !isSecureCipherSuite(cipher) {
				return false, fmt.Sprintf("Insecure cipher suite configured: %s", cipher)
			}
		}
		return true, "Custom cipher suites meet security requirements"
	}
	
	return true, "Default secure cipher suites in use"
}

func (auditor *TLSConfigAuditor) checkMutualTLSCompliance() (bool, string) {
	if isInterfaceRequiringMutualTLS(auditor.config.InterfaceType) {
		if auditor.config.MutualTLS {
			return true, fmt.Sprintf("Mutual TLS enabled for security-critical interface %s", auditor.config.InterfaceType)
		}
		return false, fmt.Sprintf("Mutual TLS required for interface %s", auditor.config.InterfaceType)
	}
	
	return true, "Mutual TLS configuration appropriate for interface type"
}

func (auditor *TLSConfigAuditor) checkFIPSCompliance() (bool, string) {
	if auditor.config.FIPSMode {
		return true, "FIPS 140-3 mode enabled"
	}
	
	if auditor.config.ComplianceMode == "strict" {
		return false, "FIPS mode should be enabled in strict compliance mode"
	}
	
	return true, "FIPS mode not required in permissive compliance mode"
}

func (auditor *TLSConfigAuditor) checkInterfaceCompliance() (bool, string) {
	switch auditor.config.InterfaceType {
	case "E2":
		if auditor.config.MutualTLS && auditor.config.MinTLSVersion == "TLS1.3" {
			return true, "E2 interface security requirements met"
		}
		return false, "E2 interface requires mutual TLS and TLS 1.3"
		
	case "A1":
		if auditor.config.MinTLSVersion == "TLS1.2" || auditor.config.MinTLSVersion == "TLS1.3" {
			return true, "A1 interface security requirements met"
		}
		return false, "A1 interface requires TLS 1.2 or higher"
		
	case "O1":
		if auditor.config.MutualTLS {
			return true, "O1 interface security requirements met"
		}
		return false, "O1 interface requires mutual TLS for management security"
		
	case "O2":
		if auditor.config.MutualTLS && auditor.config.MinTLSVersion == "TLS1.3" {
			return true, "O2 interface security requirements met"
		}
		return false, "O2 interface requires mutual TLS and TLS 1.3"
		
	default:
		return true, "Standard security configuration applied"
	}
}

func isSecureCipherSuite(cipherName string) bool {
	secureCiphers := []string{
		"TLS_AES_128_GCM_SHA256",
		"TLS_AES_256_GCM_SHA384",
		"TLS_CHACHA20_POLY1305_SHA256",
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	}
	
	for _, secure := range secureCiphers {
		if cipherName == secure {
			return true
		}
	}
	
	return false
}

// GetAuditTrail returns the audit trail for compliance tracking
func (auditor *TLSConfigAuditor) GetAuditTrail() []TLSAuditEvent {
	return auditor.auditTrail
}

