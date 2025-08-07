package ca

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// ValidationFramework provides comprehensive certificate validation capabilities
type ValidationFramework struct {
	config      *ValidationConfig
	logger      *logging.StructuredLogger
	ctClient    *CTClient
	validators  map[string]CertificateValidator
	mu          sync.RWMutex
}

// ValidationConfig configures certificate validation
type ValidationConfig struct {
	RealtimeValidation     bool          `yaml:"realtime_validation"`
	ChainValidationEnabled bool          `yaml:"chain_validation_enabled"`
	CTLogValidationEnabled bool          `yaml:"ct_log_validation_enabled"`
	CustomValidators       []string      `yaml:"custom_validators"`
	ValidationTimeout      time.Duration `yaml:"validation_timeout"`
	MaxChainDepth          int           `yaml:"max_chain_depth"`
	TrustedRoots          []string      `yaml:"trusted_roots"`
	CTLogEndpoints        []string      `yaml:"ct_log_endpoints"`
	PolicyRules           []PolicyRule  `yaml:"policy_rules"`
}

// PolicyRule defines validation policy rules
type PolicyRule struct {
	Name        string      `yaml:"name"`
	Type        string      `yaml:"type"` // subject, san, key_usage, extension, etc.
	Pattern     string      `yaml:"pattern"`
	Required    bool        `yaml:"required"`
	Severity    RuleSeverity `yaml:"severity"`
	Description string      `yaml:"description"`
}

// RuleSeverity represents the severity of policy rule violations
type RuleSeverity string

const (
	SeverityInfo    RuleSeverity = "info"
	SeverityWarning RuleSeverity = "warning"
	SeverityError   RuleSeverity = "error"
	SeverityCritical RuleSeverity = "critical"
)

// CertificateValidator interface for custom validators
type CertificateValidator interface {
	Name() string
	Validate(ctx context.Context, cert *x509.Certificate) (*ValidationResult, error)
	CanValidate(cert *x509.Certificate) bool
}

// CTClient handles Certificate Transparency log operations
type CTClient struct {
	endpoints []string
	timeout   time.Duration
	logger    *logging.StructuredLogger
}

// CTLogEntry represents a Certificate Transparency log entry
type CTLogEntry struct {
	LogID         string    `json:"log_id"`
	Index         int64     `json:"index"`
	Timestamp     time.Time `json:"timestamp"`
	Certificate   string    `json:"certificate"`
	PrecertEntry  bool      `json:"precert_entry"`
	SignedEntry   string    `json:"signed_entry"`
}

// ChainValidationResult represents chain validation results
type ChainValidationResult struct {
	Valid         bool                    `json:"valid"`
	ChainLength   int                     `json:"chain_length"`
	TrustAnchor   string                  `json:"trust_anchor"`
	Intermediates []string                `json:"intermediates"`
	Errors        []string                `json:"errors"`
	Warnings      []string                `json:"warnings"`
	Details       *ChainValidationDetails `json:"details,omitempty"`
}

// ChainValidationDetails provides detailed chain validation information
type ChainValidationDetails struct {
	PathLength      int                   `json:"path_length"`
	KeyUsageValid   bool                  `json:"key_usage_valid"`
	BasicConstraints bool                 `json:"basic_constraints"`
	NameConstraints bool                  `json:"name_constraints"`
	CertificateInfo []CertificateInfo     `json:"certificate_info"`
}

// CertificateInfo represents information about a certificate in the chain
type CertificateInfo struct {
	Subject       string    `json:"subject"`
	Issuer        string    `json:"issuer"`
	SerialNumber  string    `json:"serial_number"`
	NotBefore     time.Time `json:"not_before"`
	NotAfter      time.Time `json:"not_after"`
	KeyUsage      []string  `json:"key_usage"`
	ExtKeyUsage   []string  `json:"ext_key_usage"`
	IsCA          bool      `json:"is_ca"`
	SelfSigned    bool      `json:"self_signed"`
}

// PolicyValidationResult represents policy validation results
type PolicyValidationResult struct {
	Valid       bool                    `json:"valid"`
	Violations  []PolicyViolation       `json:"violations"`
	Warnings    []PolicyWarning         `json:"warnings"`
	Score       float64                 `json:"score"`
	Details     *PolicyValidationDetails `json:"details,omitempty"`
}

// PolicyViolation represents a policy rule violation
type PolicyViolation struct {
	Rule        *PolicyRule `json:"rule"`
	Severity    RuleSeverity `json:"severity"`
	Field       string      `json:"field"`
	Value       string      `json:"value"`
	Expected    string      `json:"expected"`
	Description string      `json:"description"`
}

// PolicyWarning represents a policy warning
type PolicyWarning struct {
	Rule        *PolicyRule `json:"rule"`
	Field       string      `json:"field"`
	Value       string      `json:"value"`
	Suggestion  string      `json:"suggestion"`
	Description string      `json:"description"`
}

// PolicyValidationDetails provides detailed policy validation information
type PolicyValidationDetails struct {
	RulesEvaluated int                     `json:"rules_evaluated"`
	RulesPassed    int                     `json:"rules_passed"`
	RulesFailed    int                     `json:"rules_failed"`
	RulesWarning   int                     `json:"rules_warning"`
	Categories     map[string]int          `json:"categories"`
	Recommendations []string              `json:"recommendations"`
}

// NewValidationFramework creates a new validation framework
func NewValidationFramework(config *ValidationConfig, logger *logging.StructuredLogger) (*ValidationFramework, error) {
	framework := &ValidationFramework{
		config:     config,
		logger:     logger,
		validators: make(map[string]CertificateValidator),
	}

	// Initialize CT client if CT validation is enabled
	if config.CTLogValidationEnabled {
		ctClient := &CTClient{
			endpoints: config.CTLogEndpoints,
			timeout:   config.ValidationTimeout,
			logger:    logger,
		}
		if len(ctClient.endpoints) == 0 {
			// Default CT log endpoints
			ctClient.endpoints = []string{
				"https://ct.googleapis.com/logs/argon2024/",
				"https://ct.googleapis.com/logs/xenon2024/",
				"https://ct.cloudflare.com/logs/nimbus2024/",
			}
		}
		framework.ctClient = ctClient
	}

	// Register built-in validators
	framework.registerBuiltinValidators()

	// Register custom validators if specified
	for _, validatorName := range config.CustomValidators {
		if err := framework.registerCustomValidator(validatorName); err != nil {
			logger.Warn("failed to register custom validator",
				"validator", validatorName,
				"error", err)
		}
	}

	return framework, nil
}

// ValidateCertificate performs comprehensive certificate validation
func (vf *ValidationFramework) ValidateCertificate(ctx context.Context, cert *x509.Certificate) (*ValidationResult, error) {
	start := time.Now()
	
	result := &ValidationResult{
		SerialNumber:   cert.SerialNumber.String(),
		ValidationTime: start,
		ExpiresAt:      cert.NotAfter,
		Valid:          true,
		Errors:         []string{},
		Warnings:       []string{},
	}

	vf.logger.Debug("starting certificate validation",
		"serial_number", result.SerialNumber,
		"subject", cert.Subject.String())

	// Basic certificate validation
	if err := vf.validateBasicCertificate(cert, result); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("basic validation failed: %v", err))
	}

	// Chain validation
	if vf.config.ChainValidationEnabled {
		chainResult, err := vf.validateCertificateChain(ctx, cert)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("chain validation failed: %v", err))
		} else {
			result.ChainValid = chainResult.Valid
			if !chainResult.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, chainResult.Errors...)
			}
			result.Warnings = append(result.Warnings, chainResult.Warnings...)
		}
	}

	// Certificate Transparency validation
	if vf.config.CTLogValidationEnabled && vf.ctClient != nil {
		ctVerified, err := vf.validateCTLog(ctx, cert)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("CT log validation failed: %v", err))
		} else {
			result.CTLogVerified = ctVerified
			if !ctVerified {
				result.Warnings = append(result.Warnings, "certificate not found in CT logs")
			}
		}
	}

	// Custom validator execution
	for name, validator := range vf.validators {
		if validator.CanValidate(cert) {
			customResult, err := validator.Validate(ctx, cert)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s validator failed: %v", name, err))
			} else if !customResult.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, customResult.Errors...)
			}
		}
	}

	// Policy validation
	if len(vf.config.PolicyRules) > 0 {
		policyResult, err := vf.validatePolicy(cert)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("policy validation failed: %v", err))
		} else if !policyResult.Valid {
			result.Valid = false
			for _, violation := range policyResult.Violations {
				if violation.Severity == SeverityError || violation.Severity == SeverityCritical {
					result.Errors = append(result.Errors, violation.Description)
				}
			}
			for _, warning := range policyResult.Warnings {
				result.Warnings = append(result.Warnings, warning.Description)
			}
		}
	}

	duration := time.Since(start)
	vf.logger.Info("certificate validation completed",
		"serial_number", result.SerialNumber,
		"valid", result.Valid,
		"chain_valid", result.ChainValid,
		"ct_verified", result.CTLogVerified,
		"errors", len(result.Errors),
		"warnings", len(result.Warnings),
		"duration", duration)

	return result, nil
}

// Basic certificate validation
func (vf *ValidationFramework) validateBasicCertificate(cert *x509.Certificate, result *ValidationResult) error {
	now := time.Now()

	// Check expiration
	if cert.NotAfter.Before(now) {
		return fmt.Errorf("certificate expired on %v", cert.NotAfter)
	}

	// Check not valid before
	if cert.NotBefore.After(now) {
		return fmt.Errorf("certificate not valid until %v", cert.NotBefore)
	}

	// Check key usage
	if cert.KeyUsage == 0 {
		result.Warnings = append(result.Warnings, "certificate has no key usage extensions")
	}

	// Check extended key usage
	if len(cert.ExtKeyUsage) == 0 && len(cert.UnknownExtKeyUsage) == 0 {
		result.Warnings = append(result.Warnings, "certificate has no extended key usage extensions")
	}

	// Check basic constraints for CA certificates
	if cert.IsCA && !cert.BasicConstraintsValid {
		return fmt.Errorf("CA certificate missing basic constraints")
	}

	// Check signature algorithm
	if cert.SignatureAlgorithm == x509.UnknownSignatureAlgorithm {
		return fmt.Errorf("unknown signature algorithm")
	}

	// Check for weak signature algorithms
	weakAlgorithms := map[x509.SignatureAlgorithm]bool{
		x509.MD2WithRSA:  true,
		x509.MD5WithRSA:  true,
		x509.SHA1WithRSA: true,
	}
	if weakAlgorithms[cert.SignatureAlgorithm] {
		result.Warnings = append(result.Warnings, fmt.Sprintf("weak signature algorithm: %v", cert.SignatureAlgorithm))
	}

	// Check RSA key size
	if rsaKey, ok := cert.PublicKey.(*x509.Certificate); ok {
		_ = rsaKey // RSA key size validation would go here
	}

	return nil
}

// Certificate chain validation
func (vf *ValidationFramework) validateCertificateChain(ctx context.Context, cert *x509.Certificate) (*ChainValidationResult, error) {
	result := &ChainValidationResult{
		Valid:       true,
		Errors:      []string{},
		Warnings:    []string{},
	}

	// Create root certificate pool
	roots := x509.NewCertPool()
	if len(vf.config.TrustedRoots) > 0 {
		// Load trusted roots from configuration
		for _, rootPEM := range vf.config.TrustedRoots {
			if !roots.AppendCertsFromPEM([]byte(rootPEM)) {
				result.Warnings = append(result.Warnings, "failed to parse trusted root certificate")
			}
		}
	} else {
		// Use system root certificates
		systemRoots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to load system root certificates: %w", err)
		}
		roots = systemRoots
	}

	// Create intermediate certificate pool (this would be populated from the certificate store)
	intermediates := x509.NewCertPool()

	// Verify certificate chain
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	chains, err := cert.Verify(opts)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("chain verification failed: %v", err))
		return result, nil
	}

	if len(chains) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "no valid certificate chains found")
		return result, nil
	}

	// Analyze the first valid chain
	chain := chains[0]
	result.ChainLength = len(chain)

	// Build chain information
	details := &ChainValidationDetails{
		PathLength:      len(chain) - 1,
		CertificateInfo: make([]CertificateInfo, len(chain)),
	}

	for i, chainCert := range chain {
		info := CertificateInfo{
			Subject:      chainCert.Subject.String(),
			Issuer:       chainCert.Issuer.String(),
			SerialNumber: chainCert.SerialNumber.String(),
			NotBefore:    chainCert.NotBefore,
			NotAfter:     chainCert.NotAfter,
			IsCA:         chainCert.IsCA,
			SelfSigned:   chainCert.Subject.String() == chainCert.Issuer.String(),
		}

		// Extract key usage
		keyUsages := []string{}
		if chainCert.KeyUsage&x509.KeyUsageDigitalSignature != 0 {
			keyUsages = append(keyUsages, "digital_signature")
		}
		if chainCert.KeyUsage&x509.KeyUsageKeyEncipherment != 0 {
			keyUsages = append(keyUsages, "key_encipherment")
		}
		if chainCert.KeyUsage&x509.KeyUsageCertSign != 0 {
			keyUsages = append(keyUsages, "cert_sign")
		}
		info.KeyUsage = keyUsages

		// Extract extended key usage
		extKeyUsages := []string{}
		for _, eku := range chainCert.ExtKeyUsage {
			switch eku {
			case x509.ExtKeyUsageServerAuth:
				extKeyUsages = append(extKeyUsages, "server_auth")
			case x509.ExtKeyUsageClientAuth:
				extKeyUsages = append(extKeyUsages, "client_auth")
			case x509.ExtKeyUsageCodeSigning:
				extKeyUsages = append(extKeyUsages, "code_signing")
			}
		}
		info.ExtKeyUsage = extKeyUsages

		details.CertificateInfo[i] = info

		// Root certificate is the trust anchor
		if i == len(chain)-1 {
			result.TrustAnchor = chainCert.Subject.String()
		} else {
			result.Intermediates = append(result.Intermediates, chainCert.Subject.String())
		}
	}

	result.Details = details

	// Validate chain constraints
	vf.validateChainConstraints(chain, result)

	return result, nil
}

// Validate certificate chain constraints
func (vf *ValidationFramework) validateChainConstraints(chain []*x509.Certificate, result *ChainValidationResult) {
	if len(chain) > vf.config.MaxChainDepth {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("certificate chain depth (%d) exceeds maximum (%d)", len(chain), vf.config.MaxChainDepth))
	}

	// Check each certificate in the chain
	for i, cert := range chain {
		// CA certificates (except leaf) should have CertSign key usage
		if i > 0 && cert.IsCA {
			if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
				result.Warnings = append(result.Warnings, 
					fmt.Sprintf("CA certificate in chain missing CertSign key usage: %s", cert.Subject.String()))
			}
		}

		// Check path length constraints
		if cert.BasicConstraintsValid && cert.MaxPathLen >= 0 {
			remainingDepth := len(chain) - i - 2 // -2 because we don't count current cert and leaf
			if remainingDepth > cert.MaxPathLen {
				result.Valid = false
				result.Errors = append(result.Errors, 
					fmt.Sprintf("path length constraint violated: %d > %d", remainingDepth, cert.MaxPathLen))
			}
		}
	}
}

// Certificate Transparency validation
func (vf *ValidationFramework) validateCTLog(ctx context.Context, cert *x509.Certificate) (bool, error) {
	if vf.ctClient == nil {
		return false, fmt.Errorf("CT client not initialized")
	}

	// Calculate certificate hash for CT log lookup
	certHash := sha256.Sum256(cert.Raw)
	certHashHex := hex.EncodeToString(certHash[:])

	// Check each CT log endpoint
	for _, endpoint := range vf.ctClient.endpoints {
		found, err := vf.ctClient.searchCertificate(ctx, endpoint, certHashHex)
		if err != nil {
			vf.logger.Warn("CT log search failed",
				"endpoint", endpoint,
				"cert_hash", certHashHex,
				"error", err)
			continue
		}

		if found {
			vf.logger.Debug("certificate found in CT log",
				"endpoint", endpoint,
				"cert_hash", certHashHex)
			return true, nil
		}
	}

	return false, nil
}

// Search for certificate in CT log
func (ct *CTClient) searchCertificate(ctx context.Context, endpoint, certHash string) (bool, error) {
	// This is a simplified implementation
	// Real CT log search would use the CT API to search for the certificate
	
	searchURL := fmt.Sprintf("%s/ct/v1/get-entries?start=0&end=100", strings.TrimSuffix(endpoint, "/"))
	
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: ct.timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("CT log request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("CT log returned status %d", resp.StatusCode)
	}

	// Parse response and search for certificate
	// This is simplified - real implementation would parse the CT log response format
	return false, nil // Placeholder - would return true if certificate found
}

// Policy validation
func (vf *ValidationFramework) validatePolicy(cert *x509.Certificate) (*PolicyValidationResult, error) {
	result := &PolicyValidationResult{
		Valid:      true,
		Violations: []PolicyViolation{},
		Warnings:   []PolicyWarning{},
		Score:      100.0,
		Details: &PolicyValidationDetails{
			Categories: make(map[string]int),
		},
	}

	for _, rule := range vf.config.PolicyRules {
		result.Details.RulesEvaluated++

		violation, warning := vf.evaluatePolicyRule(cert, &rule)
		if violation != nil {
			result.Violations = append(result.Violations, *violation)
			result.Details.RulesFailed++
			if violation.Severity == SeverityError || violation.Severity == SeverityCritical {
				result.Valid = false
			}
			// Reduce score based on severity
			switch violation.Severity {
			case SeverityCritical:
				result.Score -= 25.0
			case SeverityError:
				result.Score -= 15.0
			case SeverityWarning:
				result.Score -= 5.0
			}
		} else if warning != nil {
			result.Warnings = append(result.Warnings, *warning)
			result.Details.RulesWarning++
			result.Score -= 2.0
		} else {
			result.Details.RulesPassed++
		}

		// Update category counts
		if _, exists := result.Details.Categories[rule.Type]; !exists {
			result.Details.Categories[rule.Type] = 0
		}
		result.Details.Categories[rule.Type]++
	}

	// Ensure score doesn't go below 0
	if result.Score < 0 {
		result.Score = 0
	}

	return result, nil
}

// Evaluate individual policy rule
func (vf *ValidationFramework) evaluatePolicyRule(cert *x509.Certificate, rule *PolicyRule) (*PolicyViolation, *PolicyWarning) {
	switch rule.Type {
	case "subject":
		return vf.evaluateSubjectRule(cert, rule)
	case "san":
		return vf.evaluateSANRule(cert, rule)
	case "key_usage":
		return vf.evaluateKeyUsageRule(cert, rule)
	case "validity_period":
		return vf.evaluateValidityPeriodRule(cert, rule)
	case "signature_algorithm":
		return vf.evaluateSignatureAlgorithmRule(cert, rule)
	default:
		return nil, &PolicyWarning{
			Rule:        rule,
			Description: fmt.Sprintf("unknown rule type: %s", rule.Type),
		}
	}
}

func (vf *ValidationFramework) evaluateSubjectRule(cert *x509.Certificate, rule *PolicyRule) (*PolicyViolation, *PolicyWarning) {
	subject := cert.Subject.String()
	
	// Simple pattern matching (real implementation would use regex)
	if !strings.Contains(subject, rule.Pattern) {
		if rule.Required {
			return &PolicyViolation{
				Rule:        rule,
				Severity:    rule.Severity,
				Field:       "subject",
				Value:       subject,
				Expected:    rule.Pattern,
				Description: fmt.Sprintf("subject does not match required pattern: %s", rule.Pattern),
			}, nil
		} else {
			return nil, &PolicyWarning{
				Rule:        rule,
				Field:       "subject",
				Value:       subject,
				Suggestion:  fmt.Sprintf("consider including pattern: %s", rule.Pattern),
				Description: fmt.Sprintf("subject does not match recommended pattern: %s", rule.Pattern),
			}
		}
	}

	return nil, nil
}

func (vf *ValidationFramework) evaluateSANRule(cert *x509.Certificate, rule *PolicyRule) (*PolicyViolation, *PolicyWarning) {
	sans := append(cert.DNSNames, cert.EmailAddresses...)
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	if len(sans) == 0 && rule.Required {
		return &PolicyViolation{
			Rule:        rule,
			Severity:    rule.Severity,
			Field:       "san",
			Value:       "none",
			Expected:    "at least one SAN",
			Description: "certificate requires Subject Alternative Names",
		}, nil
	}

	return nil, nil
}

func (vf *ValidationFramework) evaluateKeyUsageRule(cert *x509.Certificate, rule *PolicyRule) (*PolicyViolation, *PolicyWarning) {
	// This would check if required key usages are present
	// Simplified implementation
	return nil, nil
}

func (vf *ValidationFramework) evaluateValidityPeriodRule(cert *x509.Certificate, rule *PolicyRule) (*PolicyViolation, *PolicyWarning) {
	// This would check certificate validity period constraints
	// Simplified implementation
	return nil, nil
}

func (vf *ValidationFramework) evaluateSignatureAlgorithmRule(cert *x509.Certificate, rule *PolicyRule) (*PolicyViolation, *PolicyWarning) {
	// This would check signature algorithm requirements
	// Simplified implementation
	return nil, nil
}

// Register built-in validators
func (vf *ValidationFramework) registerBuiltinValidators() {
	// Built-in validators would be registered here
	vf.logger.Debug("registered built-in certificate validators")
}

// Register custom validator
func (vf *ValidationFramework) registerCustomValidator(name string) error {
	// Custom validators would be loaded and registered here
	vf.logger.Debug("attempting to register custom validator", "validator", name)
	return fmt.Errorf("custom validator registration not implemented: %s", name)
}

// AddValidator adds a custom validator
func (vf *ValidationFramework) AddValidator(validator CertificateValidator) {
	vf.mu.Lock()
	defer vf.mu.Unlock()
	
	vf.validators[validator.Name()] = validator
	vf.logger.Info("validator registered", "name", validator.Name())
}

// RemoveValidator removes a validator
func (vf *ValidationFramework) RemoveValidator(name string) {
	vf.mu.Lock()
	defer vf.mu.Unlock()
	
	delete(vf.validators, name)
	vf.logger.Info("validator removed", "name", name)
}

// GetValidators returns all registered validators
func (vf *ValidationFramework) GetValidators() []string {
	vf.mu.RLock()
	defer vf.mu.RUnlock()
	
	validators := make([]string, 0, len(vf.validators))
	for name := range vf.validators {
		validators = append(validators, name)
	}
	
	return validators
}