package ca

import (
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// PolicyEngine enforces certificate policies and validation rules
type PolicyEngine struct {
	config     *PolicyConfig
	logger     *logging.StructuredLogger
	templates  map[string]*PolicyTemplate
	rules      map[string]*ValidationRule
	workflows  map[string]*ApprovalWorkflow
	mu         sync.RWMutex
}

// PolicyValidationResult represents the result of policy validation
type PolicyValidationResult struct {
	Valid      bool                    `json:"valid"`
	Errors     []PolicyError          `json:"errors,omitempty"`
	Warnings   []PolicyWarning        `json:"warnings,omitempty"`
	Template   *PolicyTemplate        `json:"template,omitempty"`
	Workflow   *ApprovalWorkflow      `json:"workflow,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyError represents a policy validation error
type PolicyError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Field       string `json:"field,omitempty"`
	Rule        string `json:"rule,omitempty"`
	Severity    string `json:"severity"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// PolicyWarning represents a policy validation warning
type PolicyWarning struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Field       string `json:"field,omitempty"`
	Rule        string `json:"rule,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// ApprovalRequest represents a certificate approval request
type ApprovalRequest struct {
	ID                string                     `json:"id"`
	CertificateRequest *CertificateRequest       `json:"certificate_request"`
	PolicyTemplate    string                     `json:"policy_template"`
	Workflow          *ApprovalWorkflow         `json:"workflow"`
	CurrentStage      int                       `json:"current_stage"`
	Status            ApprovalStatus            `json:"status"`
	Approvals         []Approval                `json:"approvals"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
	ExpiresAt         time.Time                 `json:"expires_at"`
	RequestedBy       string                    `json:"requested_by"`
	Justification     string                    `json:"justification,omitempty"`
}

// ApprovalStatus represents approval request status
type ApprovalStatus string

const (
	ApprovalStatusPending   ApprovalStatus = "pending"
	ApprovalStatusApproved  ApprovalStatus = "approved"
	ApprovalStatusRejected  ApprovalStatus = "rejected"
	ApprovalStatusExpired   ApprovalStatus = "expired"
	ApprovalStatusEscalated ApprovalStatus = "escalated"
)

// Approval represents an individual approval
type Approval struct {
	StageIndex  int       `json:"stage_index"`
	StageName   string    `json:"stage_name"`
	Approver    string    `json:"approver"`
	Status      string    `json:"status"` // approved, rejected
	Comments    string    `json:"comments,omitempty"`
	ApprovedAt  time.Time `json:"approved_at"`
	IPAddress   string    `json:"ip_address,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Field       string    `json:"field"`
	Rule        string    `json:"rule"`
	Value       string    `json:"value"`
	Expected    string    `json:"expected"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ValidatorFunc represents a custom validation function
type ValidatorFunc func(req *CertificateRequest, template *PolicyTemplate) []PolicyViolation

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(config *PolicyConfig, logger *logging.StructuredLogger) (*PolicyEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("policy config is required")
	}

	engine := &PolicyEngine{
		config:    config,
		logger:    logger,
		templates: make(map[string]*PolicyTemplate),
		rules:     make(map[string]*ValidationRule),
		workflows: make(map[string]*ApprovalWorkflow),
	}

	// Load policy templates
	for name, template := range config.PolicyTemplates {
		engine.templates[name] = template
	}

	// Load validation rules
	for _, rule := range config.ValidationRules {
		engine.rules[rule.Name] = &rule
	}

	// Load approval workflow
	if config.ApprovalWorkflow != nil {
		engine.workflows["default"] = config.ApprovalWorkflow
	}

	logger.Info("policy engine initialized",
		"templates", len(engine.templates),
		"rules", len(engine.rules),
		"workflows", len(engine.workflows))

	return engine, nil
}

// ValidateRequest validates a certificate request against policies
func (e *PolicyEngine) ValidateRequest(req *CertificateRequest) error {
	e.logger.Debug("validating certificate request",
		"request_id", req.ID,
		"common_name", req.CommonName,
		"policy_template", req.PolicyTemplate)

	result := e.ValidateRequestDetailed(req)
	if !result.Valid {
		return fmt.Errorf("policy validation failed: %v", result.Errors)
	}

	return nil
}

// ValidateRequestDetailed performs detailed policy validation
func (e *PolicyEngine) ValidateRequestDetailed(req *CertificateRequest) *PolicyValidationResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := &PolicyValidationResult{
		Valid:    true,
		Errors:   []PolicyError{},
		Warnings: []PolicyWarning{},
		Metadata: make(map[string]interface{}),
	}

	// Select policy template
	template := e.selectPolicyTemplate(req)
	if template != nil {
		result.Template = template
		result.Metadata["policy_template"] = template.Name
	}

	// Validate against template
	if template != nil {
		violations := e.validateAgainstTemplate(req, template)
		for _, violation := range violations {
			if violation.Severity == "error" {
				result.Errors = append(result.Errors, PolicyError{
					Code:       violation.Code,
					Message:    violation.Message,
					Field:      violation.Field,
					Rule:       violation.Rule,
					Severity:   violation.Severity,
					Suggestion: e.getSuggestion(violation),
				})
				result.Valid = false
			} else if violation.Severity == "warning" {
				result.Warnings = append(result.Warnings, PolicyWarning{
					Code:       violation.Code,
					Message:    violation.Message,
					Field:      violation.Field,
					Rule:       violation.Rule,
					Suggestion: e.getSuggestion(violation),
				})
			}
		}
	}

	// Validate against global rules
	for _, rule := range e.rules {
		violations := e.validateAgainstRule(req, rule)
		for _, violation := range violations {
			if rule.Required || violation.Severity == "error" {
				result.Errors = append(result.Errors, PolicyError{
					Code:       violation.Code,
					Message:    violation.Message,
					Field:      violation.Field,
					Rule:       violation.Rule,
					Severity:   "error",
					Suggestion: e.getSuggestion(violation),
				})
				result.Valid = false
			} else {
				result.Warnings = append(result.Warnings, PolicyWarning{
					Code:       violation.Code,
					Message:    violation.Message,
					Field:      violation.Field,
					Rule:       violation.Rule,
					Suggestion: e.getSuggestion(violation),
				})
			}
		}
	}

	// Check if approval is required
	if e.requiresApproval(req, template) {
		workflow := e.selectApprovalWorkflow(req, template)
		if workflow != nil {
			result.Workflow = workflow
			result.Metadata["requires_approval"] = true
			result.Metadata["approval_workflow"] = workflow.Stages
		}
	}

	e.logger.Debug("policy validation completed",
		"request_id", req.ID,
		"valid", result.Valid,
		"errors", len(result.Errors),
		"warnings", len(result.Warnings))

	return result
}

// ApplyTemplate applies a policy template to a certificate request
func (e *PolicyEngine) ApplyTemplate(req *CertificateRequest, templateName string) error {
	e.mu.RLock()
	template, exists := e.templates[templateName]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("policy template '%s' not found", templateName)
	}

	// Apply template settings
	if template.ValidityDuration > 0 {
		req.ValidityDuration = template.ValidityDuration
	}

	if template.KeySize > 0 {
		req.KeySize = template.KeySize
	}

	if len(template.KeyUsage) > 0 {
		req.KeyUsage = template.KeyUsage
	}

	if len(template.ExtKeyUsage) > 0 {
		req.ExtKeyUsage = template.ExtKeyUsage
	}

	// Apply custom extensions
	if req.CustomExtensions == nil {
		req.CustomExtensions = make(map[string]string)
	}
	for key, value := range template.CustomExtensions {
		req.CustomExtensions[key] = value
	}

	req.PolicyTemplate = templateName

	e.logger.Info("policy template applied",
		"request_id", req.ID,
		"template", templateName)

	return nil
}

// CreateApprovalRequest creates an approval request
func (e *PolicyEngine) CreateApprovalRequest(req *CertificateRequest, requestedBy, justification string) (*ApprovalRequest, error) {
	template := e.selectPolicyTemplate(req)
	workflow := e.selectApprovalWorkflow(req, template)

	if workflow == nil {
		return nil, fmt.Errorf("no approval workflow available")
	}

	approvalReq := &ApprovalRequest{
		ID:                 generateApprovalRequestID(),
		CertificateRequest: req,
		PolicyTemplate:     req.PolicyTemplate,
		Workflow:          workflow,
		CurrentStage:      0,
		Status:            ApprovalStatusPending,
		Approvals:         []Approval{},
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(workflow.Timeout),
		RequestedBy:       requestedBy,
		Justification:     justification,
	}

	e.logger.Info("approval request created",
		"approval_id", approvalReq.ID,
		"request_id", req.ID,
		"workflow_stages", len(workflow.Stages))

	return approvalReq, nil
}

// ProcessApproval processes an approval decision
func (e *PolicyEngine) ProcessApproval(approvalID, approver, decision, comments string) error {
	// This would typically load from persistent storage
	// For now, this is a placeholder implementation
	
	e.logger.Info("processing approval",
		"approval_id", approvalID,
		"approver", approver,
		"decision", decision)

	// Implementation would:
	// 1. Load approval request from storage
	// 2. Validate approver permissions
	// 3. Record approval/rejection
	// 4. Advance to next stage or complete
	// 5. Send notifications
	// 6. Update persistent storage

	return nil
}

// GetApprovalStatus returns the status of an approval request
func (e *PolicyEngine) GetApprovalStatus(approvalID string) (*ApprovalRequest, error) {
	// This would load from persistent storage
	return nil, fmt.Errorf("approval request %s not found", approvalID)
}

// AddPolicyTemplate adds a new policy template
func (e *PolicyEngine) AddPolicyTemplate(name string, template *PolicyTemplate) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.templates[name]; exists {
		return fmt.Errorf("policy template '%s' already exists", name)
	}

	e.templates[name] = template

	e.logger.Info("policy template added",
		"name", name,
		"validity_duration", template.ValidityDuration)

	return nil
}

// UpdatePolicyTemplate updates an existing policy template
func (e *PolicyEngine) UpdatePolicyTemplate(name string, template *PolicyTemplate) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.templates[name]; !exists {
		return fmt.Errorf("policy template '%s' not found", name)
	}

	e.templates[name] = template

	e.logger.Info("policy template updated",
		"name", name)

	return nil
}

// DeletePolicyTemplate removes a policy template
func (e *PolicyEngine) DeletePolicyTemplate(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.templates[name]; !exists {
		return fmt.Errorf("policy template '%s' not found", name)
	}

	delete(e.templates, name)

	e.logger.Info("policy template deleted",
		"name", name)

	return nil
}

// ListPolicyTemplates returns all policy templates
func (e *PolicyEngine) ListPolicyTemplates() map[string]*PolicyTemplate {
	e.mu.RLock()
	defer e.mu.RUnlock()

	templates := make(map[string]*PolicyTemplate)
	for name, template := range e.templates {
		templates[name] = template
	}

	return templates
}

// Helper methods

func (e *PolicyEngine) selectPolicyTemplate(req *CertificateRequest) *PolicyTemplate {
	if req.PolicyTemplate != "" {
		if template, exists := e.templates[req.PolicyTemplate]; exists {
			return template
		}
	}

	// Default template selection logic based on certificate properties
	if len(req.DNSNames) > 0 {
		if template, exists := e.templates["server"]; exists {
			return template
		}
	}

	if len(req.EmailAddresses) > 0 {
		if template, exists := e.templates["email"]; exists {
			return template
		}
	}

	// Return default template if available
	if template, exists := e.templates["default"]; exists {
		return template
	}

	return nil
}

func (e *PolicyEngine) validateAgainstTemplate(req *CertificateRequest, template *PolicyTemplate) []PolicyViolation {
	var violations []PolicyViolation

	// Validate validity duration
	if template.ValidityDuration > 0 && req.ValidityDuration > template.ValidityDuration {
		violations = append(violations, PolicyViolation{
			Code:     "VALIDITY_TOO_LONG",
			Message:  fmt.Sprintf("Validity duration %v exceeds template maximum %v", req.ValidityDuration, template.ValidityDuration),
			Field:    "validity_duration",
			Rule:     template.Name,
			Value:    req.ValidityDuration.String(),
			Expected: template.ValidityDuration.String(),
			Severity: "error",
			Category: "validity",
		})
	}

	// Validate key size
	if template.KeySize > 0 && req.KeySize < template.KeySize {
		violations = append(violations, PolicyViolation{
			Code:     "KEY_SIZE_TOO_SMALL",
			Message:  fmt.Sprintf("Key size %d is below template minimum %d", req.KeySize, template.KeySize),
			Field:    "key_size",
			Rule:     template.Name,
			Value:    fmt.Sprintf("%d", req.KeySize),
			Expected: fmt.Sprintf("%d", template.KeySize),
			Severity: "error",
			Category: "key_properties",
		})
	}

	// Validate key usage
	if len(template.KeyUsage) > 0 {
		for _, required := range template.KeyUsage {
			found := false
			for _, actual := range req.KeyUsage {
				if actual == required {
					found = true
					break
				}
			}
			if !found {
				violations = append(violations, PolicyViolation{
					Code:     "MISSING_KEY_USAGE",
					Message:  fmt.Sprintf("Required key usage '%s' is missing", required),
					Field:    "key_usage",
					Rule:     template.Name,
					Value:    strings.Join(req.KeyUsage, ","),
					Expected: required,
					Severity: "error",
					Category: "key_usage",
				})
			}
		}
	}

	// Validate extended key usage
	if len(template.ExtKeyUsage) > 0 {
		for _, required := range template.ExtKeyUsage {
			found := false
			for _, actual := range req.ExtKeyUsage {
				if actual == required {
					found = true
					break
				}
			}
			if !found {
				violations = append(violations, PolicyViolation{
					Code:     "MISSING_EXT_KEY_USAGE",
					Message:  fmt.Sprintf("Required extended key usage '%s' is missing", required),
					Field:    "ext_key_usage",
					Rule:     template.Name,
					Value:    strings.Join(req.ExtKeyUsage, ","),
					Expected: required,
					Severity: "error",
					Category: "ext_key_usage",
				})
			}
		}
	}

	// Validate SAN types
	if len(template.AllowedSANTypes) > 0 {
		violations = append(violations, e.validateSANTypes(req, template.AllowedSANTypes, template.Name)...)
	}

	// Validate required fields
	violations = append(violations, e.validateRequiredFields(req, template.RequiredFields, template.Name)...)

	return violations
}

func (e *PolicyEngine) validateAgainstRule(req *CertificateRequest, rule *ValidationRule) []PolicyViolation {
	var violations []PolicyViolation

	switch rule.Type {
	case "subject":
		if rule.Pattern != "" {
			regex, err := regexp.Compile(rule.Pattern)
			if err == nil {
				if !regex.MatchString(req.CommonName) {
					violations = append(violations, PolicyViolation{
						Code:     "SUBJECT_PATTERN_MISMATCH",
						Message:  rule.ErrorMessage,
						Field:    "common_name",
						Rule:     rule.Name,
						Value:    req.CommonName,
						Expected: rule.Pattern,
						Severity: e.getSeverity(rule.Required),
						Category: "subject",
					})
				}
			}
		}

	case "san":
		violations = append(violations, e.validateSANPattern(req, rule)...)

	case "key_usage":
		violations = append(violations, e.validateKeyUsageRule(req, rule)...)

	case "validity":
		violations = append(violations, e.validateValidityRule(req, rule)...)

	case "custom":
		// Custom validation logic would go here
		violations = append(violations, e.validateCustomRule(req, rule)...)
	}

	return violations
}

func (e *PolicyEngine) validateSANTypes(req *CertificateRequest, allowedTypes []string, ruleName string) []PolicyViolation {
	var violations []PolicyViolation

	typeMap := make(map[string]bool)
	for _, t := range allowedTypes {
		typeMap[t] = true
	}

	if len(req.DNSNames) > 0 && !typeMap["dns"] {
		violations = append(violations, PolicyViolation{
			Code:     "DISALLOWED_SAN_TYPE",
			Message:  "DNS SANs are not allowed by policy",
			Field:    "dns_names",
			Rule:     ruleName,
			Severity: "error",
			Category: "san_types",
		})
	}

	if len(req.IPAddresses) > 0 && !typeMap["ip"] {
		violations = append(violations, PolicyViolation{
			Code:     "DISALLOWED_SAN_TYPE",
			Message:  "IP SANs are not allowed by policy",
			Field:    "ip_addresses",
			Rule:     ruleName,
			Severity: "error",
			Category: "san_types",
		})
	}

	if len(req.EmailAddresses) > 0 && !typeMap["email"] {
		violations = append(violations, PolicyViolation{
			Code:     "DISALLOWED_SAN_TYPE",
			Message:  "Email SANs are not allowed by policy",
			Field:    "email_addresses",
			Rule:     ruleName,
			Severity: "error",
			Category: "san_types",
		})
	}

	if len(req.URIs) > 0 && !typeMap["uri"] {
		violations = append(violations, PolicyViolation{
			Code:     "DISALLOWED_SAN_TYPE",
			Message:  "URI SANs are not allowed by policy",
			Field:    "uris",
			Rule:     ruleName,
			Severity: "error",
			Category: "san_types",
		})
	}

	return violations
}

func (e *PolicyEngine) validateRequiredFields(req *CertificateRequest, requiredFields []string, ruleName string) []PolicyViolation {
	var violations []PolicyViolation

	for _, field := range requiredFields {
		switch field {
		case "common_name":
			if req.CommonName == "" {
				violations = append(violations, PolicyViolation{
					Code:     "REQUIRED_FIELD_MISSING",
					Message:  "Common name is required",
					Field:    "common_name",
					Rule:     ruleName,
					Severity: "error",
					Category: "required_fields",
				})
			}
		case "dns_names":
			if len(req.DNSNames) == 0 {
				violations = append(violations, PolicyViolation{
					Code:     "REQUIRED_FIELD_MISSING",
					Message:  "DNS names are required",
					Field:    "dns_names",
					Rule:     ruleName,
					Severity: "error",
					Category: "required_fields",
				})
			}
		case "email_addresses":
			if len(req.EmailAddresses) == 0 {
				violations = append(violations, PolicyViolation{
					Code:     "REQUIRED_FIELD_MISSING",
					Message:  "Email addresses are required",
					Field:    "email_addresses",
					Rule:     ruleName,
					Severity: "error",
					Category: "required_fields",
				})
			}
		}
	}

	return violations
}

func (e *PolicyEngine) validateSANPattern(req *CertificateRequest, rule *ValidationRule) []PolicyViolation {
	var violations []PolicyViolation

	if rule.Pattern == "" {
		return violations
	}

	regex, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return violations
	}

	// Validate DNS SANs
	for _, dns := range req.DNSNames {
		if !regex.MatchString(dns) {
			violations = append(violations, PolicyViolation{
				Code:     "SAN_PATTERN_MISMATCH",
				Message:  fmt.Sprintf("DNS SAN '%s' does not match required pattern", dns),
				Field:    "dns_names",
				Rule:     rule.Name,
				Value:    dns,
				Expected: rule.Pattern,
				Severity: e.getSeverity(rule.Required),
				Category: "san_pattern",
			})
		}
	}

	return violations
}

func (e *PolicyEngine) validateKeyUsageRule(req *CertificateRequest, rule *ValidationRule) []PolicyViolation {
	var violations []PolicyViolation

	if rule.Pattern == "" {
		return violations
	}

	requiredUsages := strings.Split(rule.Pattern, ",")
	for _, required := range requiredUsages {
		required = strings.TrimSpace(required)
		found := false
		for _, actual := range req.KeyUsage {
			if actual == required {
				found = true
				break
			}
		}
		if !found && rule.Required {
			violations = append(violations, PolicyViolation{
				Code:     "KEY_USAGE_MISSING",
				Message:  fmt.Sprintf("Required key usage '%s' is missing", required),
				Field:    "key_usage",
				Rule:     rule.Name,
				Value:    strings.Join(req.KeyUsage, ","),
				Expected: required,
				Severity: "error",
				Category: "key_usage",
			})
		}
	}

	return violations
}

func (e *PolicyEngine) validateValidityRule(req *CertificateRequest, rule *ValidationRule) []PolicyViolation {
	var violations []PolicyViolation

	if rule.Pattern == "" {
		return violations
	}

	// Parse pattern as duration
	maxDuration, err := time.ParseDuration(rule.Pattern)
	if err != nil {
		return violations
	}

	if req.ValidityDuration > maxDuration {
		violations = append(violations, PolicyViolation{
			Code:     "VALIDITY_EXCEEDS_MAXIMUM",
			Message:  fmt.Sprintf("Validity duration %v exceeds maximum allowed %v", req.ValidityDuration, maxDuration),
			Field:    "validity_duration",
			Rule:     rule.Name,
			Value:    req.ValidityDuration.String(),
			Expected: maxDuration.String(),
			Severity: e.getSeverity(rule.Required),
			Category: "validity",
		})
	}

	return violations
}

func (e *PolicyEngine) validateCustomRule(req *CertificateRequest, rule *ValidationRule) []PolicyViolation {
	// Placeholder for custom validation logic
	// This could be extended to support plugins or custom validators
	return []PolicyViolation{}
}

func (e *PolicyEngine) requiresApproval(req *CertificateRequest, template *PolicyTemplate) bool {
	if e.config.ApprovalRequired {
		return true
	}

	if template != nil && template.RequireApproval {
		return true
	}

	// Check for high-risk certificates
	if req.ValidityDuration > 365*24*time.Hour {
		return true
	}

	// Check for wildcard certificates
	for _, dns := range req.DNSNames {
		if strings.HasPrefix(dns, "*.") {
			return true
		}
	}

	return false
}

func (e *PolicyEngine) selectApprovalWorkflow(req *CertificateRequest, template *PolicyTemplate) *ApprovalWorkflow {
	// For now, return the default workflow
	if workflow, exists := e.workflows["default"]; exists {
		return workflow
	}

	return nil
}

func (e *PolicyEngine) getSeverity(required bool) string {
	if required {
		return "error"
	}
	return "warning"
}

func (e *PolicyEngine) getSuggestion(violation PolicyViolation) string {
	suggestions := map[string]string{
		"VALIDITY_TOO_LONG":      "Reduce the validity duration to comply with policy limits",
		"KEY_SIZE_TOO_SMALL":     "Increase key size to meet minimum security requirements",
		"MISSING_KEY_USAGE":      "Add the required key usage to the certificate request",
		"MISSING_EXT_KEY_USAGE":  "Add the required extended key usage to the certificate request",
		"DISALLOWED_SAN_TYPE":    "Remove the disallowed SAN types or request policy exemption",
		"SUBJECT_PATTERN_MISMATCH": "Adjust the subject to match the required pattern",
		"SAN_PATTERN_MISMATCH":   "Modify SAN values to match the required pattern",
	}

	if suggestion, ok := suggestions[violation.Code]; ok {
		return suggestion
	}

	return "Please review the policy requirements and adjust the request accordingly"
}

// generateApprovalRequestID generates a unique approval request ID
func generateApprovalRequestID() string {
	return fmt.Sprintf("approval-%d", time.Now().UnixNano())
}