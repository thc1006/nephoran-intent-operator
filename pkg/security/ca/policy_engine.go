package ca

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// PolicyEngine enforces certificate security policies
type PolicyEngine struct {
	config           *PolicyConfig
	logger           *logging.StructuredLogger
	rules            map[string]PolicyRule
	pinningStore     *CertificatePinningStore
	customValidators map[string]CustomPolicyValidator
	oranValidator    *ORANPolicyValidator
	metrics          *PolicyMetrics
	mu               sync.RWMutex
}

// PolicyConfig configures the policy engine
type PolicyConfig struct {
	Enabled                   bool                      `yaml:"enabled"`
	Rules                     []PolicyRule              `yaml:"rules"`
	CertificatePinning        bool                      `yaml:"certificate_pinning"`
	AlgorithmStrengthCheck    bool                      `yaml:"algorithm_strength_check"`
	MinimumRSAKeySize         int                       `yaml:"minimum_rsa_key_size"`
	AllowedECCurves           []string                  `yaml:"allowed_ec_curves"`
	RequireExtendedValidation bool                      `yaml:"require_extended_validation"`
	ServicePolicies           map[string]*ServicePolicy `yaml:"service_policies"`
	EnforcementMode           string                    `yaml:"enforcement_mode"` // strict, permissive, audit
	CustomValidators          []string                  `yaml:"custom_validators"`
	ORANCompliance            *ORANComplianceConfig     `yaml:"oran_compliance"`
}

// ServicePolicy defines policies for specific services
type ServicePolicy struct {
	ServiceName         string             `yaml:"service_name"`
	Namespace           string             `yaml:"namespace"`
	PinnedCertificates  []string           `yaml:"pinned_certificates"`
	RequiredKeyUsage    []x509.KeyUsage    `yaml:"required_key_usage"`
	RequiredExtKeyUsage []x509.ExtKeyUsage `yaml:"required_ext_key_usage"`
	AllowedDNSNames     []string           `yaml:"allowed_dns_names"`
	AllowedIPAddresses  []string           `yaml:"allowed_ip_addresses"`
	MinimumValidityDays int                `yaml:"minimum_validity_days"`
	MaximumValidityDays int                `yaml:"maximum_validity_days"`
	CustomRules         []PolicyRule       `yaml:"custom_rules"`
}

// ORANComplianceConfig configures O-RAN compliance checking
type ORANComplianceConfig struct {
	Enabled              bool     `yaml:"enabled"`
	RequiredExtensions   []string `yaml:"required_extensions"`
	ComponentValidation  bool     `yaml:"component_validation"`
	NamingConvention     string   `yaml:"naming_convention"`
	MinimumSecurityLevel int      `yaml:"minimum_security_level"`
	ForbiddenAlgorithms  []string `yaml:"forbidden_algorithms"`
	RequireHSMProtection bool     `yaml:"require_hsm_protection"`
}

// CertificatePinningStore manages certificate pins
type CertificatePinningStore struct {
	pins             map[string]*CertificatePin
	dynamicPins      map[string]*DynamicPin
	backupPins       map[string][]*CertificatePin
	rotationSchedule map[string]*PinRotationSchedule
	mu               sync.RWMutex
}

// CertificatePin represents a pinned certificate
type CertificatePin struct {
	ServiceName     string    `json:"service_name"`
	Fingerprint     string    `json:"fingerprint"`
	PublicKeyHash   string    `json:"public_key_hash"`
	Algorithm       string    `json:"algorithm"`
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	AllowSubdomains bool      `json:"allow_subdomains"`
	IncludeBackup   bool      `json:"include_backup"`
}

// DynamicPin represents a dynamically updated pin
type DynamicPin struct {
	Pattern         string        `json:"pattern"`
	UpdateInterval  time.Duration `json:"update_interval"`
	LastUpdate      time.Time     `json:"last_update"`
	CurrentPins     []string      `json:"current_pins"`
	ValidationCount atomic.Uint64
}

// PinRotationSchedule manages pin rotation
type PinRotationSchedule struct {
	ServiceName      string        `json:"service_name"`
	RotationInterval time.Duration `json:"rotation_interval"`
	NextRotation     time.Time     `json:"next_rotation"`
	BackupPinCount   int           `json:"backup_pin_count"`
	AutoRotate       bool          `json:"auto_rotate"`
}

// CustomPolicyValidator interface for custom validators
type CustomPolicyValidator interface {
	Name() string
	Validate(ctx context.Context, cert *x509.Certificate, policy *ServicePolicy) (*PolicyValidationResult, error)
	Priority() int
}

// ORANPolicyValidator validates O-RAN specific policies
type ORANPolicyValidator struct {
	config           *ORANComplianceConfig
	componentRules   map[string]*ComponentRule
	namingValidator  *NamingConventionValidator
	securityAnalyzer *SecurityLevelAnalyzer
	logger           *logging.StructuredLogger
}

// ComponentRule defines rules for O-RAN components
type ComponentRule struct {
	ComponentType  string   `yaml:"component_type"`
	RequiredSANs   []string `yaml:"required_sans"`
	ForbiddenSANs  []string `yaml:"forbidden_sans"`
	MinKeySize     int      `yaml:"min_key_size"`
	RequiredOUs    []string `yaml:"required_ous"`
	ValidityPeriod int      `yaml:"validity_period"`
}

// NamingConventionValidator validates naming conventions
type NamingConventionValidator struct {
	patterns map[string]*regexp.Regexp
	rules    []NamingRule
}

// NamingRule defines naming convention rules
type NamingRule struct {
	Field       string `yaml:"field"`
	Pattern     string `yaml:"pattern"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
}

// SecurityLevelAnalyzer analyzes certificate security level
type SecurityLevelAnalyzer struct {
	criteria map[int][]SecurityCriterion
	weights  map[string]float64
}

// SecurityCriterion defines security level criteria
type SecurityCriterion struct {
	Name        string `yaml:"name"`
	Check       func(*x509.Certificate) bool
	Weight      float64 `yaml:"weight"`
	Required    bool    `yaml:"required"`
	Description string  `yaml:"description"`
}

// PolicyMetrics tracks policy enforcement metrics
type PolicyMetrics struct {
	TotalValidations    atomic.Uint64
	PolicyViolations    atomic.Uint64
	PinningHits         atomic.Uint64
	PinningMisses       atomic.Uint64
	AlgorithmViolations atomic.Uint64
	ORANViolations      atomic.Uint64
	CustomViolations    atomic.Uint64
	EnforcementActions  atomic.Uint64
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(config *PolicyConfig, logger *logging.StructuredLogger) (*PolicyEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("policy config is required")
	}

	engine := &PolicyEngine{
		config:           config,
		logger:           logger,
		rules:            make(map[string]PolicyRule),
		customValidators: make(map[string]CustomPolicyValidator),
		metrics:          &PolicyMetrics{},
	}

	// Initialize rules
	for _, rule := range config.Rules {
		engine.rules[rule.Name] = rule
	}

	// Initialize certificate pinning
	if config.CertificatePinning {
		engine.pinningStore = &CertificatePinningStore{
			pins:             make(map[string]*CertificatePin),
			dynamicPins:      make(map[string]*DynamicPin),
			backupPins:       make(map[string][]*CertificatePin),
			rotationSchedule: make(map[string]*PinRotationSchedule),
		}

		// Load initial pins from service policies
		engine.loadServicePins()
	}

	// Initialize O-RAN validator
	if config.ORANCompliance != nil && config.ORANCompliance.Enabled {
		engine.oranValidator = &ORANPolicyValidator{
			config:           config.ORANCompliance,
			componentRules:   make(map[string]*ComponentRule),
			namingValidator:  engine.createNamingValidator(),
			securityAnalyzer: engine.createSecurityAnalyzer(),
			logger:           logger,
		}
	}

	// Register custom validators
	for _, validatorName := range config.CustomValidators {
		if err := engine.registerCustomValidator(validatorName); err != nil {
			logger.Warn("failed to register custom validator",
				"validator", validatorName,
				"error", err)
		}
	}

	return engine, nil
}

// ValidateCertificate validates a certificate against policies
func (pe *PolicyEngine) ValidateCertificate(ctx context.Context, cert *x509.Certificate) *PolicyValidationResult {
	start := time.Now()
	pe.metrics.TotalValidations.Add(1)

	result := &PolicyValidationResult{
		Valid:      true,
		Violations: []PolicyViolation{},
		Warnings:   []PolicyWarning{},
		Score:      100.0,
		Details: &PolicyValidationDetails{
			Categories:      make(map[string]int),
			Recommendations: []string{},
		},
	}

	// Check algorithm strength
	if pe.config.AlgorithmStrengthCheck {
		pe.validateAlgorithmStrength(cert, result)
	}

	// Check certificate pinning
	if pe.config.CertificatePinning {
		pe.validateCertificatePinning(cert, result)
	}

	// Apply general policy rules
	for _, rule := range pe.rules {
		pe.applyPolicyRule(cert, &rule, result)
	}

	// Check extended validation requirements
	if pe.config.RequireExtendedValidation {
		pe.validateExtendedValidation(cert, result)
	}

	// O-RAN compliance validation
	if pe.oranValidator != nil {
		pe.validateORANCompliance(ctx, cert, result)
	}

	// Apply custom validators
	for _, validator := range pe.customValidators {
		customResult, err := validator.Validate(ctx, cert, nil)
		if err != nil {
			result.Warnings = append(result.Warnings, PolicyWarning{
				Description: fmt.Sprintf("custom validator %s failed: %v", validator.Name(), err),
			})
		} else {
			pe.mergeValidationResults(result, customResult)
		}
	}

	// Calculate final score
	result.Score = pe.calculateScore(result)

	// Record metrics
	if !result.Valid {
		pe.metrics.PolicyViolations.Add(1)
		pe.enforcePolicy(cert, result)
	}

	duration := time.Since(start)
	pe.logger.Debug("policy validation completed",
		"serial_number", cert.SerialNumber.String(),
		"valid", result.Valid,
		"score", result.Score,
		"violations", len(result.Violations),
		"duration", duration)

	return result
}

// ValidateServiceCertificate validates a certificate for a specific service
func (pe *PolicyEngine) ValidateServiceCertificate(ctx context.Context, cert *x509.Certificate, serviceName, namespace string) *PolicyValidationResult {
	// Get service-specific policy
	servicePolicy := pe.getServicePolicy(serviceName, namespace)
	if servicePolicy == nil {
		// Use default validation
		return pe.ValidateCertificate(ctx, cert)
	}

	result := pe.ValidateCertificate(ctx, cert)

	// Apply service-specific policies
	pe.applyServicePolicy(cert, servicePolicy, result)

	// Apply custom validators with service context
	for _, validator := range pe.customValidators {
		customResult, err := validator.Validate(ctx, cert, servicePolicy)
		if err != nil {
			result.Warnings = append(result.Warnings, PolicyWarning{
				Description: fmt.Sprintf("service validator %s failed: %v", validator.Name(), err),
			})
		} else {
			pe.mergeValidationResults(result, customResult)
		}
	}

	return result
}

// Algorithm strength validation

func (pe *PolicyEngine) validateAlgorithmStrength(cert *x509.Certificate, result *PolicyValidationResult) {
	// Check signature algorithm
	weakAlgorithms := map[x509.SignatureAlgorithm]bool{
		x509.MD2WithRSA:    true,
		x509.MD5WithRSA:    true,
		x509.SHA1WithRSA:   true,
		x509.DSAWithSHA1:   true,
		x509.ECDSAWithSHA1: true,
	}

	if weakAlgorithms[cert.SignatureAlgorithm] {
		violation := PolicyViolation{
			Severity:    RuleSeverityCritical,
			Field:       "signature_algorithm",
			Value:       cert.SignatureAlgorithm.String(),
			Expected:    "SHA256 or stronger",
			Description: "weak signature algorithm detected",
		}
		result.Violations = append(result.Violations, violation)
		result.Valid = false
		result.Score -= 25.0
		pe.metrics.AlgorithmViolations.Add(1)
	}

	// Check RSA key size
	if rsaKey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
		keySize := rsaKey.N.BitLen()
		if keySize < pe.config.MinimumRSAKeySize {
			violation := PolicyViolation{
				Severity:    RuleSeverityError,
				Field:       "rsa_key_size",
				Value:       fmt.Sprintf("%d bits", keySize),
				Expected:    fmt.Sprintf(">= %d bits", pe.config.MinimumRSAKeySize),
				Description: "RSA key size below minimum requirement",
			}
			result.Violations = append(result.Violations, violation)
			result.Valid = false
			result.Score -= 20.0
			pe.metrics.AlgorithmViolations.Add(1)
		}
	}

	// Check EC curves
	if ecKey, ok := cert.PublicKey.(*ecdsa.PublicKey); ok {
		curveName := ecKey.Curve.Params().Name
		allowed := false
		for _, allowedCurve := range pe.config.AllowedECCurves {
			if curveName == allowedCurve {
				allowed = true
				break
			}
		}
		if !allowed {
			violation := PolicyViolation{
				Severity:    RuleSeverityError,
				Field:       "ec_curve",
				Value:       curveName,
				Expected:    strings.Join(pe.config.AllowedECCurves, ", "),
				Description: "EC curve not in allowed list",
			}
			result.Violations = append(result.Violations, violation)
			result.Valid = false
			result.Score -= 15.0
			pe.metrics.AlgorithmViolations.Add(1)
		}
	}
}

// Certificate pinning validation

func (pe *PolicyEngine) validateCertificatePinning(cert *x509.Certificate, result *PolicyValidationResult) {
	if pe.pinningStore == nil {
		return
	}

	// Calculate certificate fingerprint
	fingerprint := pe.calculateFingerprint(cert)
	publicKeyHash := pe.calculatePublicKeyHash(cert)

	// Check for exact certificate pin
	pin := pe.pinningStore.GetPin(cert.Subject.CommonName)
	if pin != nil {
		if pin.Fingerprint != fingerprint && pin.PublicKeyHash != publicKeyHash {
			violation := PolicyViolation{
				Severity:    RuleSeverityCritical,
				Field:       "certificate_pin",
				Value:       fingerprint,
				Expected:    pin.Fingerprint,
				Description: "certificate does not match pinned value",
			}
			result.Violations = append(result.Violations, violation)
			result.Valid = false
			result.Score -= 30.0
			pe.metrics.PinningMisses.Add(1)
		} else {
			pe.metrics.PinningHits.Add(1)
			result.Details.Recommendations = append(result.Details.Recommendations,
				"certificate matches pinned value")
		}
	}

	// Check dynamic pins
	pe.checkDynamicPins(cert, result)
}

func (pe *PolicyEngine) calculateFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hash[:])
}

func (pe *PolicyEngine) calculatePublicKeyHash(cert *x509.Certificate) string {
	publicKeyDER, _ := x509.MarshalPKIXPublicKey(cert.PublicKey)
	hash := sha256.Sum256(publicKeyDER)
	return hex.EncodeToString(hash[:])
}

func (pe *PolicyEngine) checkDynamicPins(cert *x509.Certificate, result *PolicyValidationResult) {
	pe.pinningStore.mu.RLock()
	defer pe.pinningStore.mu.RUnlock()

	for pattern, dynamicPin := range pe.pinningStore.dynamicPins {
		matched, _ := regexp.MatchString(pattern, cert.Subject.CommonName)
		if matched {
			dynamicPin.ValidationCount.Add(1)

			fingerprint := pe.calculateFingerprint(cert)
			found := false
			for _, pin := range dynamicPin.CurrentPins {
				if pin == fingerprint {
					found = true
					break
				}
			}

			if !found {
				warning := PolicyWarning{
					Field:       "dynamic_pin",
					Value:       fingerprint,
					Suggestion:  "consider updating dynamic pin list",
					Description: fmt.Sprintf("certificate not in dynamic pin list for pattern %s", pattern),
				}
				result.Warnings = append(result.Warnings, warning)
				result.Score -= 5.0
			}
		}
	}
}

// Extended validation

func (pe *PolicyEngine) validateExtendedValidation(cert *x509.Certificate, result *PolicyValidationResult) {
	// Check for EV OID in certificate policies
	evOIDs := []string{
		"2.23.140.1.1",          // CA/Browser Forum EV OID
		"1.3.6.1.4.1.34697.1.1", // Example vendor EV OID
	}

	hasEV := false
	for _, ext := range cert.Extensions {
		// Certificate Policies extension OID: 2.5.29.32
		if ext.Id.Equal([]int{2, 5, 29, 32}) {
			// Parse certificate policies
			// This is simplified - real implementation would parse ASN.1
			for _, oid := range evOIDs {
				if strings.Contains(string(ext.Value), oid) {
					hasEV = true
					break
				}
			}
		}
	}

	if !hasEV {
		violation := PolicyViolation{
			Severity:    RuleSeverityError,
			Field:       "extended_validation",
			Value:       "not present",
			Expected:    "EV certificate required",
			Description: "certificate lacks extended validation",
		}
		result.Violations = append(result.Violations, violation)
		result.Valid = false
		result.Score -= 20.0
	}

	// Additional EV checks
	pe.validateEVOrganization(cert, result)
	pe.validateEVJurisdiction(cert, result)
}

func (pe *PolicyEngine) validateEVOrganization(cert *x509.Certificate, result *PolicyValidationResult) {
	if cert.Subject.Organization == nil || len(cert.Subject.Organization) == 0 {
		warning := PolicyWarning{
			Field:       "organization",
			Value:       "not present",
			Suggestion:  "EV certificates should include organization",
			Description: "missing organization in EV certificate",
		}
		result.Warnings = append(result.Warnings, warning)
		result.Score -= 5.0
	}
}

func (pe *PolicyEngine) validateEVJurisdiction(cert *x509.Certificate, result *PolicyValidationResult) {
	// Check for jurisdiction extensions
	// OID: 1.3.6.1.4.1.311.60.2.1.3 (jurisdictionCountry)
	// This is simplified - real implementation would check specific OIDs
}

// O-RAN compliance validation

func (pe *PolicyEngine) validateORANCompliance(ctx context.Context, cert *x509.Certificate, result *PolicyValidationResult) {
	if pe.oranValidator == nil {
		return
	}

	// Check required extensions
	for _, requiredOID := range pe.oranValidator.config.RequiredExtensions {
		hasExtension := false
		for _, ext := range cert.Extensions {
			if ext.Id.String() == requiredOID {
				hasExtension = true
				break
			}
		}

		if !hasExtension {
			violation := PolicyViolation{
				Severity:    RuleSeverityError,
				Field:       "oran_extension",
				Value:       "missing",
				Expected:    requiredOID,
				Description: fmt.Sprintf("missing required O-RAN extension: %s", requiredOID),
			}
			result.Violations = append(result.Violations, violation)
			result.Valid = false
			result.Score -= 10.0
			pe.metrics.ORANViolations.Add(1)
		}
	}

	// Validate naming convention
	if pe.oranValidator.namingValidator != nil {
		pe.validateORANNaming(cert, result)
	}

	// Check security level
	if pe.oranValidator.securityAnalyzer != nil {
		securityLevel := pe.oranValidator.securityAnalyzer.AnalyzeSecurityLevel(cert)
		if securityLevel < pe.oranValidator.config.MinimumSecurityLevel {
			violation := PolicyViolation{
				Severity:    RuleSeverityError,
				Field:       "security_level",
				Value:       fmt.Sprintf("%d", securityLevel),
				Expected:    fmt.Sprintf(">= %d", pe.oranValidator.config.MinimumSecurityLevel),
				Description: "certificate security level below O-RAN requirement",
			}
			result.Violations = append(result.Violations, violation)
			result.Valid = false
			result.Score -= 15.0
			pe.metrics.ORANViolations.Add(1)
		}
	}

	// Check forbidden algorithms
	for _, forbidden := range pe.oranValidator.config.ForbiddenAlgorithms {
		if cert.SignatureAlgorithm.String() == forbidden {
			violation := PolicyViolation{
				Severity:    RuleSeverityCritical,
				Field:       "signature_algorithm",
				Value:       cert.SignatureAlgorithm.String(),
				Expected:    "not " + forbidden,
				Description: "forbidden algorithm for O-RAN",
			}
			result.Violations = append(result.Violations, violation)
			result.Valid = false
			result.Score -= 25.0
			pe.metrics.ORANViolations.Add(1)
		}
	}

	// Check HSM protection requirement
	if pe.oranValidator.config.RequireHSMProtection {
		pe.validateHSMProtection(cert, result)
	}
}

func (pe *PolicyEngine) validateORANNaming(cert *x509.Certificate, result *PolicyValidationResult) {
	validator := pe.oranValidator.namingValidator

	for _, rule := range validator.rules {
		var value string
		switch rule.Field {
		case "CN":
			value = cert.Subject.CommonName
		case "O":
			if len(cert.Subject.Organization) > 0 {
				value = cert.Subject.Organization[0]
			}
		case "OU":
			if len(cert.Subject.OrganizationalUnit) > 0 {
				value = cert.Subject.OrganizationalUnit[0]
			}
		}

		pattern, exists := validator.patterns[rule.Pattern]
		if !exists {
			pattern = regexp.MustCompile(rule.Pattern)
			validator.patterns[rule.Pattern] = pattern
		}

		if !pattern.MatchString(value) {
			if rule.Required {
				violation := PolicyViolation{
					Severity:    RuleSeverityError,
					Field:       rule.Field,
					Value:       value,
					Expected:    rule.Pattern,
					Description: fmt.Sprintf("O-RAN naming violation: %s", rule.Description),
				}
				result.Violations = append(result.Violations, violation)
				result.Valid = false
				result.Score -= 10.0
			} else {
				warning := PolicyWarning{
					Field:       rule.Field,
					Value:       value,
					Suggestion:  fmt.Sprintf("should match pattern: %s", rule.Pattern),
					Description: rule.Description,
				}
				result.Warnings = append(result.Warnings, warning)
				result.Score -= 2.0
			}
		}
	}
}

func (pe *PolicyEngine) validateHSMProtection(cert *x509.Certificate, result *PolicyValidationResult) {
	// Check for HSM protection indicators
	// This might be in custom extensions or key usage
	// Simplified implementation

	hasHSMIndicator := false
	for _, ext := range cert.Extensions {
		// Check for HSM protection OID (vendor-specific)
		if ext.Id.String() == "1.3.6.1.4.1.41482.3.1" { // Example HSM OID
			hasHSMIndicator = true
			break
		}
	}

	if !hasHSMIndicator {
		violation := PolicyViolation{
			Severity:    RuleSeverityError,
			Field:       "hsm_protection",
			Value:       "not detected",
			Expected:    "HSM-protected key",
			Description: "O-RAN requires HSM protection for this certificate",
		}
		result.Violations = append(result.Violations, violation)
		result.Valid = false
		result.Score -= 20.0
	}
}

// Service policy application

func (pe *PolicyEngine) applyServicePolicy(cert *x509.Certificate, policy *ServicePolicy, result *PolicyValidationResult) {
	// Check pinned certificates
	if len(policy.PinnedCertificates) > 0 {
		fingerprint := pe.calculateFingerprint(cert)
		found := false
		for _, pinned := range policy.PinnedCertificates {
			if pinned == fingerprint {
				found = true
				break
			}
		}
		if !found {
			violation := PolicyViolation{
				Severity:    RuleSeverityCritical,
				Field:       "service_pin",
				Value:       fingerprint,
				Expected:    "pinned certificate",
				Description: fmt.Sprintf("certificate not pinned for service %s", policy.ServiceName),
			}
			result.Violations = append(result.Violations, violation)
			result.Valid = false
			result.Score -= 30.0
		}
	}

	// Check key usage
	if len(policy.RequiredKeyUsage) > 0 {
		for _, required := range policy.RequiredKeyUsage {
			if cert.KeyUsage&required == 0 {
				violation := PolicyViolation{
					Severity:    RuleSeverityError,
					Field:       "key_usage",
					Value:       fmt.Sprintf("%d", cert.KeyUsage),
					Expected:    fmt.Sprintf("includes %d", required),
					Description: "missing required key usage for service",
				}
				result.Violations = append(result.Violations, violation)
				result.Valid = false
				result.Score -= 10.0
			}
		}
	}

	// Check DNS names
	if len(policy.AllowedDNSNames) > 0 {
		pe.validateAllowedDNSNames(cert, policy, result)
	}

	// Check validity period
	pe.validateValidityPeriod(cert, policy, result)

	// Apply custom rules
	for _, rule := range policy.CustomRules {
		pe.applyPolicyRule(cert, &rule, result)
	}
}

func (pe *PolicyEngine) validateAllowedDNSNames(cert *x509.Certificate, policy *ServicePolicy, result *PolicyValidationResult) {
	for _, dnsName := range cert.DNSNames {
		allowed := false
		for _, allowedPattern := range policy.AllowedDNSNames {
			if matched, _ := regexp.MatchString(allowedPattern, dnsName); matched {
				allowed = true
				break
			}
		}
		if !allowed {
			violation := PolicyViolation{
				Severity:    RuleSeverityError,
				Field:       "dns_name",
				Value:       dnsName,
				Expected:    strings.Join(policy.AllowedDNSNames, ", "),
				Description: "DNS name not in allowed list for service",
			}
			result.Violations = append(result.Violations, violation)
			result.Valid = false
			result.Score -= 10.0
		}
	}
}

func (pe *PolicyEngine) validateValidityPeriod(cert *x509.Certificate, policy *ServicePolicy, result *PolicyValidationResult) {
	validityDays := int(cert.NotAfter.Sub(cert.NotBefore).Hours() / 24)

	if policy.MinimumValidityDays > 0 && validityDays < policy.MinimumValidityDays {
		violation := PolicyViolation{
			Severity:    SeverityWarning,
			Field:       "validity_period",
			Value:       fmt.Sprintf("%d days", validityDays),
			Expected:    fmt.Sprintf(">= %d days", policy.MinimumValidityDays),
			Description: "certificate validity period too short",
		}
		result.Violations = append(result.Violations, violation)
		result.Score -= 5.0
	}

	if policy.MaximumValidityDays > 0 && validityDays > policy.MaximumValidityDays {
		violation := PolicyViolation{
			Severity:    RuleSeverityError,
			Field:       "validity_period",
			Value:       fmt.Sprintf("%d days", validityDays),
			Expected:    fmt.Sprintf("<= %d days", policy.MaximumValidityDays),
			Description: "certificate validity period too long",
		}
		result.Violations = append(result.Violations, violation)
		result.Valid = false
		result.Score -= 10.0
	}
}

// Helper methods

func (pe *PolicyEngine) applyPolicyRule(cert *x509.Certificate, rule *PolicyRule, result *PolicyValidationResult) {
	// Apply rule based on type
	// This is simplified - real implementation would have comprehensive rule evaluation
	result.Details.RulesEvaluated++
}

func (pe *PolicyEngine) getServicePolicy(serviceName, namespace string) *ServicePolicy {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", namespace, serviceName)
	return pe.config.ServicePolicies[key]
}

func (pe *PolicyEngine) loadServicePins() {
	for _, policy := range pe.config.ServicePolicies {
		for _, pinHash := range policy.PinnedCertificates {
			pin := &CertificatePin{
				ServiceName: policy.ServiceName,
				Fingerprint: pinHash,
				CreatedAt:   time.Now(),
			}
			pe.pinningStore.pins[policy.ServiceName] = pin
		}
	}
}

func (pe *PolicyEngine) createNamingValidator() *NamingConventionValidator {
	return &NamingConventionValidator{
		patterns: make(map[string]*regexp.Regexp),
		rules: []NamingRule{
			{
				Field:       "CN",
				Pattern:     `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`,
				Required:    true,
				Description: "Common Name must follow O-RAN naming convention",
			},
			{
				Field:       "O",
				Pattern:     `^O-RAN-.*$`,
				Required:    false,
				Description: "Organization should start with O-RAN-",
			},
		},
	}
}

func (pe *PolicyEngine) createSecurityAnalyzer() *SecurityLevelAnalyzer {
	analyzer := &SecurityLevelAnalyzer{
		criteria: make(map[int][]SecurityCriterion),
		weights:  make(map[string]float64),
	}

	// Define security level criteria
	analyzer.criteria[1] = []SecurityCriterion{
		{
			Name: "key_size",
			Check: func(cert *x509.Certificate) bool {
				if rsaKey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
					return rsaKey.N.BitLen() >= 2048
				}
				return true
			},
			Weight:   1.0,
			Required: true,
		},
		{
			Name: "signature_algorithm",
			Check: func(cert *x509.Certificate) bool {
				return cert.SignatureAlgorithm >= x509.SHA256WithRSA
			},
			Weight:   1.0,
			Required: true,
		},
	}

	return analyzer
}

func (sa *SecurityLevelAnalyzer) AnalyzeSecurityLevel(cert *x509.Certificate) int {
	maxLevel := 0

	for level, criteria := range sa.criteria {
		passed := true
		for _, criterion := range criteria {
			if criterion.Required && !criterion.Check(cert) {
				passed = false
				break
			}
		}
		if passed && level > maxLevel {
			maxLevel = level
		}
	}

	return maxLevel
}

func (pe *PolicyEngine) registerCustomValidator(name string) error {
	// This would load and register custom validators
	// Simplified implementation
	return fmt.Errorf("custom validator registration not implemented: %s", name)
}

func (pe *PolicyEngine) mergeValidationResults(target, source *PolicyValidationResult) {
	target.Violations = append(target.Violations, source.Violations...)
	target.Warnings = append(target.Warnings, source.Warnings...)

	if !source.Valid {
		target.Valid = false
	}

	// Merge scores (weighted average)
	target.Score = (target.Score + source.Score) / 2
}

func (pe *PolicyEngine) calculateScore(result *PolicyValidationResult) float64 {
	score := 100.0

	for _, violation := range result.Violations {
		switch violation.Severity {
		case SeverityCritical:
			score -= 25.0
		case SeverityError:
			score -= 15.0
		case SeverityWarning:
			score -= 5.0
		}
	}

	for range result.Warnings {
		score -= 2.0
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (pe *PolicyEngine) enforcePolicy(cert *x509.Certificate, result *PolicyValidationResult) {
	pe.metrics.EnforcementActions.Add(1)

	switch pe.config.EnforcementMode {
	case "strict":
		// Block certificate usage
		pe.logger.Error("certificate policy violation - blocking",
			"serial_number", cert.SerialNumber.String(),
			"violations", len(result.Violations))
	case "permissive":
		// Allow but warn
		pe.logger.Warn("certificate policy violation - allowing",
			"serial_number", cert.SerialNumber.String(),
			"violations", len(result.Violations))
	case "audit":
		// Log only
		pe.logger.Info("certificate policy violation - audit mode",
			"serial_number", cert.SerialNumber.String(),
			"violations", len(result.Violations))
	}
}

// CertificatePinningStore methods

func (ps *CertificatePinningStore) GetPin(serviceName string) *CertificatePin {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.pins[serviceName]
}

func (ps *CertificatePinningStore) AddPin(pin *CertificatePin) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.pins[pin.ServiceName] = pin

	// Add to backup pins if configured
	if pin.IncludeBackup {
		ps.backupPins[pin.ServiceName] = append(ps.backupPins[pin.ServiceName], pin)

		// Keep only last 3 backup pins
		if len(ps.backupPins[pin.ServiceName]) > 3 {
			ps.backupPins[pin.ServiceName] = ps.backupPins[pin.ServiceName][1:]
		}
	}
}

func (ps *CertificatePinningStore) RemovePin(serviceName string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.pins, serviceName)
}

func (ps *CertificatePinningStore) GetBackupPins(serviceName string) []*CertificatePin {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.backupPins[serviceName]
}

// AddPolicyRule adds a new policy rule
func (pe *PolicyEngine) AddPolicyRule(rule PolicyRule) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.rules[rule.Name] = rule
	pe.logger.Info("policy rule added", "rule", rule.Name)
}

// RemovePolicyRule removes a policy rule
func (pe *PolicyEngine) RemovePolicyRule(name string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	delete(pe.rules, name)
	pe.logger.Info("policy rule removed", "rule", name)
}

// GetMetrics returns policy enforcement metrics
func (pe *PolicyEngine) GetMetrics() map[string]uint64 {
	return map[string]uint64{
		"total_validations":    pe.metrics.TotalValidations.Load(),
		"policy_violations":    pe.metrics.PolicyViolations.Load(),
		"pinning_hits":         pe.metrics.PinningHits.Load(),
		"pinning_misses":       pe.metrics.PinningMisses.Load(),
		"algorithm_violations": pe.metrics.AlgorithmViolations.Load(),
		"oran_violations":      pe.metrics.ORANViolations.Load(),
		"custom_violations":    pe.metrics.CustomViolations.Load(),
		"enforcement_actions":  pe.metrics.EnforcementActions.Load(),
	}
}
