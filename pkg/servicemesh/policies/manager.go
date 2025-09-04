// Package policies provides policy management for service mesh integration.

package policies

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/servicemesh/abstraction"
)

// PolicyManager manages service mesh policies.

type PolicyManager struct {
	mesh abstraction.ServiceMeshInterface

	policyStore *PolicyStore

	validator *PolicyValidator

	enforcer *PolicyEnforcer

	logger logr.Logger
}

// NewPolicyManager creates a new policy manager.

func NewPolicyManager(mesh abstraction.ServiceMeshInterface) *PolicyManager {
	return &PolicyManager{
		mesh: mesh,

		policyStore: NewPolicyStore(),

		validator: NewPolicyValidator(mesh),

		enforcer: NewPolicyEnforcer(mesh),

		logger: log.Log.WithName("policy-manager"),
	}
}

// ApplyPolicySet applies a set of policies.

func (m *PolicyManager) ApplyPolicySet(ctx context.Context, policySet *PolicySet) error {
	m.logger.Info("Applying policy set", "name", policySet.Name, "policies", len(policySet.Policies))

	// Validate all policies first.

	for _, policy := range policySet.Policies {
		if err := m.validator.ValidatePolicy(ctx, policy); err != nil {
			return fmt.Errorf("policy validation failed for %s: %w", policy.Name, err)
		}
	}

	// Check for conflicts.

	conflicts := m.validator.CheckConflicts(policySet.Policies)

	if len(conflicts) > 0 {
		return fmt.Errorf("policy conflicts detected: %v", conflicts)
	}

	// Apply policies in order.

	appliedPolicies := []Policy{}

	for _, policy := range policySet.Policies {

		if err := m.applyPolicy(ctx, policy); err != nil {

			// Rollback applied policies on error.

			m.rollbackPolicies(ctx, appliedPolicies)

			return fmt.Errorf("failed to apply policy %s: %w", policy.Name, err)

		}

		appliedPolicies = append(appliedPolicies, policy)

	}

	// Store policy set.

	m.policyStore.StorePolicySet(policySet)

	m.logger.Info("Policy set applied successfully", "name", policySet.Name)

	return nil
}

// applyPolicy applies a single policy.

func (m *PolicyManager) applyPolicy(ctx context.Context, policy Policy) error {
	switch policy.Type {

	case PolicyTypeMTLS:

		return m.applyMTLSPolicy(ctx, policy)

	case PolicyTypeAuthorization:

		return m.applyAuthorizationPolicy(ctx, policy)

	case PolicyTypeTraffic:

		return m.applyTrafficPolicy(ctx, policy)

	case PolicyTypeNetworkSegmentation:

		return m.applyNetworkSegmentationPolicy(ctx, policy)

	default:

		return fmt.Errorf("unknown policy type: %s", policy.Type)

	}
}

// applyMTLSPolicy applies an mTLS policy.

func (m *PolicyManager) applyMTLSPolicy(ctx context.Context, policy Policy) error {
	mtlsPolicy := &abstraction.MTLSPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policy.Name,

			Namespace: policy.Namespace,

			Labels: policy.Labels,
		},

		Spec: abstraction.MTLSPolicySpec{
			Selector: policy.Selector,

			Mode: policy.Spec.MTLSMode,
		},
	}

	// Add port-level mTLS if specified.

	if portMTLS, ok := policy.Spec.PortLevelMTLS.([]interface{}); ok {
		for _, pm := range portMTLS {
			if portMap, ok := pm.(map[string]interface{}); ok {

				portFloat := portMap["port"].(float64)
				if portFloat < 0 || portFloat > 65535 {
					return fmt.Errorf("port number out of range: %f", portFloat)
				}
				port := int(portFloat)

				mode := portMap["mode"].(string)

				mtlsPolicy.Spec.PortLevelMTLS = append(mtlsPolicy.Spec.PortLevelMTLS,

					abstraction.PortMTLS{Port: port, Mode: mode})

			}
		}
	}

	return m.mesh.ApplyMTLSPolicy(ctx, mtlsPolicy)
}

// applyAuthorizationPolicy applies an authorization policy.

func (m *PolicyManager) applyAuthorizationPolicy(ctx context.Context, policy Policy) error {
	authPolicy := &abstraction.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policy.Name,

			Namespace: policy.Namespace,

			Labels: policy.Labels,
		},

		Spec: abstraction.AuthorizationPolicySpec{
			Selector: policy.Selector,

			Action: policy.Spec.Action,
		},
	}

	// Convert rules.

	if rules, ok := policy.Spec.Rules.([]interface{}); ok {
		for _, rule := range rules {
			if ruleMap, ok := rule.(map[string]interface{}); ok {

				authRule := m.convertAuthorizationRule(ruleMap)

				authPolicy.Spec.Rules = append(authPolicy.Spec.Rules, authRule)

			}
		}
	}

	return m.mesh.ApplyAuthorizationPolicy(ctx, authPolicy)
}

// applyTrafficPolicy applies a traffic management policy.

func (m *PolicyManager) applyTrafficPolicy(ctx context.Context, policy Policy) error {
	trafficPolicy := &abstraction.TrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policy.Name,

			Namespace: policy.Namespace,

			Labels: policy.Labels,
		},

		Spec: abstraction.TrafficPolicySpec{
			Selector: policy.Selector,
		},
	}

	// Add traffic management features.

	if cb, ok := policy.Spec.CircuitBreaker.(map[string]interface{}); ok {
		trafficPolicy.Spec.CircuitBreaker = &abstraction.CircuitBreaker{
			ConsecutiveErrors: func() int {
				f := cb["consecutiveErrors"].(float64)
				if f < 0 || f > 2147483647 {
					panic("consecutiveErrors out of int range")
				}
				return int(f)
			}(),

			Interval: cb["interval"].(string),

			BaseEjectionTime: cb["baseEjectionTime"].(string),

			MaxEjectionPercent: func() int {
				f := cb["maxEjectionPercent"].(float64)
				if f < 0 || f > 100 {
					panic("maxEjectionPercent must be 0-100")
				}
				return int(f)
			}(),
		}
	}

	if retry, ok := policy.Spec.Retry.(map[string]interface{}); ok {
		trafficPolicy.Spec.Retry = &abstraction.RetryPolicy{
			Attempts: func() int {
				f := retry["attempts"].(float64)
				if f < 0 || f > 2147483647 {
					panic("attempts out of int range")
				}
				return int(f)
			}(),

			PerTryTimeout: retry["perTryTimeout"].(string),
		}
	}

	return m.mesh.ApplyTrafficPolicy(ctx, trafficPolicy)
}

// applyNetworkSegmentationPolicy applies network segmentation.

func (m *PolicyManager) applyNetworkSegmentationPolicy(ctx context.Context, policy Policy) error {
	// Network segmentation is implemented through authorization policies.

	// Create deny-all by default and then allow specific traffic.

	// Create deny-all policy.

	denyAll := &abstraction.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-deny-all", policy.Name),

			Namespace: policy.Namespace,
		},

		Spec: abstraction.AuthorizationPolicySpec{
			Selector: policy.Selector,

			Action: "DENY",
		},
	}

	if err := m.mesh.ApplyAuthorizationPolicy(ctx, denyAll); err != nil {
		return fmt.Errorf("failed to apply deny-all policy: %w", err)
	}

	// Create allow policies for specific segments.

	if segments, ok := policy.Spec.AllowedSegments.([]interface{}); ok {
		for _, segment := range segments {
			if segMap, ok := segment.(map[string]interface{}); ok {

				allowPolicy := &abstraction.AuthorizationPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-allow-%s", policy.Name, segMap["name"].(string)),

						Namespace: policy.Namespace,
					},

					Spec: abstraction.AuthorizationPolicySpec{
						Selector: policy.Selector,

						Action: "ALLOW",

						Rules: m.createSegmentRules(segMap),
					},
				}

				if err := m.mesh.ApplyAuthorizationPolicy(ctx, allowPolicy); err != nil {
					return fmt.Errorf("failed to apply allow policy for segment %s: %w", segMap["name"], err)
				}

			}
		}
	}

	return nil
}

// rollbackPolicies rolls back applied policies.

func (m *PolicyManager) rollbackPolicies(ctx context.Context, policies []Policy) {
	m.logger.Info("Rolling back policies", "count", len(policies))

	for i := len(policies) - 1; i >= 0; i-- {
		// TODO: Implement policy deletion.

		m.logger.Info("Rolling back policy", "name", policies[i].Name)
	}
}

// convertAuthorizationRule converts a map to an authorization rule.

func (m *PolicyManager) convertAuthorizationRule(ruleMap map[string]interface{}) abstraction.AuthorizationRule {
	rule := abstraction.AuthorizationRule{}

	// Convert From.

	if from, ok := ruleMap["from"].([]interface{}); ok {
		for _, f := range from {
			if fromMap, ok := f.(map[string]interface{}); ok {

				source := abstraction.SecuritySource{}

				if principals, ok := fromMap["principals"].([]interface{}); ok {
					for _, p := range principals {
						source.Principals = append(source.Principals, p.(string))
					}
				}

				if namespaces, ok := fromMap["namespaces"].([]interface{}); ok {
					for _, n := range namespaces {
						source.Namespaces = append(source.Namespaces, n.(string))
					}
				}

				rule.From = append(rule.From, source)

			}
		}
	}

	// Convert To.

	if to, ok := ruleMap["to"].([]interface{}); ok {
		for _, t := range to {
			if toMap, ok := t.(map[string]interface{}); ok {

				operation := abstraction.SecurityOperation{}

				if methods, ok := toMap["methods"].([]interface{}); ok {
					for _, m := range methods {
						operation.Methods = append(operation.Methods, m.(string))
					}
				}

				if paths, ok := toMap["paths"].([]interface{}); ok {
					for _, p := range paths {
						operation.Paths = append(operation.Paths, p.(string))
					}
				}

				rule.To = append(rule.To, operation)

			}
		}
	}

	return rule
}

// createSegmentRules creates rules for network segmentation.

func (m *PolicyManager) createSegmentRules(segment map[string]interface{}) []abstraction.AuthorizationRule {
	rules := []abstraction.AuthorizationRule{}

	if sources, ok := segment["sources"].([]interface{}); ok {

		rule := abstraction.AuthorizationRule{}

		for _, source := range sources {
			if sourceMap, ok := source.(map[string]interface{}); ok {

				secSource := abstraction.SecuritySource{}

				if ns, ok := sourceMap["namespace"].(string); ok {
					secSource.Namespaces = []string{ns}
				}

				if labels, ok := sourceMap["labels"].(map[string]interface{}); ok {
					// Convert labels to principals (simplified).

					for k, v := range labels {
						secSource.Principals = append(secSource.Principals,

							fmt.Sprintf("cluster.local/ns/*/sa/%s-%s", k, v))
					}
				}

				rule.From = append(rule.From, secSource)

			}
		}

		rules = append(rules, rule)

	}

	return rules
}

// EnforceZeroTrust enforces zero-trust principles across the mesh.

func (m *PolicyManager) EnforceZeroTrust(ctx context.Context, config *ZeroTrustConfig) error {
	m.logger.Info("Enforcing zero-trust configuration")

	// Create policy set for zero-trust.

	policySet := &PolicySet{
		Name: "zero-trust",

		Description: "Zero-trust security policies",

		Policies: []Policy{},
	}

	// 1. Enforce strict mTLS everywhere.

	mtlsPolicy := Policy{
		Name: "zero-trust-mtls",

		Type: PolicyTypeMTLS,

		Namespace: config.Namespace,

		Spec: PolicySpec{
			MTLSMode: "STRICT",
		},
	}

	policySet.Policies = append(policySet.Policies, mtlsPolicy)

	// 2. Default deny-all authorization.

	denyPolicy := Policy{
		Name: "zero-trust-deny-all",

		Type: PolicyTypeAuthorization,

		Namespace: config.Namespace,

		Spec: PolicySpec{
			Action: "DENY",
		},
	}

	policySet.Policies = append(policySet.Policies, denyPolicy)

	// 3. Allow only explicitly defined communications.

	for _, allowedComm := range config.AllowedCommunications {

		allowPolicy := Policy{
			Name: fmt.Sprintf("zero-trust-allow-%s-to-%s", allowedComm.Source, allowedComm.Destination),

			Type: PolicyTypeAuthorization,

			Namespace: config.Namespace,

			Selector: &abstraction.LabelSelector{
				MatchLabels: map[string]string{
					"app": allowedComm.Destination,
				},
			},

			Spec: PolicySpec{
				Action: "ALLOW",

				Rules: []interface{}{
					map[string]interface{}{
						"from": []interface{}{
							map[string]interface{}{},
						},
					},
				},
			},
		}

		policySet.Policies = append(policySet.Policies, allowPolicy)

	}

	// Apply the policy set.

	return m.ApplyPolicySet(ctx, policySet)
}

// ValidateCompliance validates compliance with security policies.

func (m *PolicyManager) ValidateCompliance(ctx context.Context, standard ComplianceStandard) (*ComplianceReport, error) {
	m.logger.Info("Validating compliance", "standard", standard)

	report := &ComplianceReport{
		Standard: standard,

		Timestamp: time.Now(),

		Compliant: true,

		Checks: []ComplianceCheck{},
	}

	// Check mTLS compliance.

	mtlsCheck := m.checkMTLSCompliance(ctx)

	report.Checks = append(report.Checks, mtlsCheck)

	if !mtlsCheck.Passed {
		report.Compliant = false
	}

	// Check authorization compliance.

	authCheck := m.checkAuthorizationCompliance(ctx)

	report.Checks = append(report.Checks, authCheck)

	if !authCheck.Passed {
		report.Compliant = false
	}

	// Check network segmentation.

	segmentCheck := m.checkSegmentationCompliance(ctx)

	report.Checks = append(report.Checks, segmentCheck)

	if !segmentCheck.Passed {
		report.Compliant = false
	}

	// Check certificate compliance.

	certCheck := m.checkCertificateCompliance(ctx)

	report.Checks = append(report.Checks, certCheck)

	if !certCheck.Passed {
		report.Compliant = false
	}

	// Calculate compliance score.

	passedChecks := 0

	for _, check := range report.Checks {
		if check.Passed {
			passedChecks++
		}
	}

	report.Score = float64(passedChecks) / float64(len(report.Checks)) * 100

	return report, nil
}

// checkMTLSCompliance checks mTLS compliance.

func (m *PolicyManager) checkMTLSCompliance(ctx context.Context) ComplianceCheck {
	check := ComplianceCheck{
		Name: "mTLS Enforcement",

		Description: "All services must use strict mTLS",

		Passed: true,
	}

	// Get mTLS status.

	report, err := m.mesh.GetMTLSStatus(ctx, "")
	if err != nil {

		check.Passed = false

		check.Issues = append(check.Issues, fmt.Sprintf("Failed to get mTLS status: %v", err))

		return check

	}

	if report.Coverage < 100 {

		check.Passed = false

		check.Issues = append(check.Issues,

			fmt.Sprintf("mTLS coverage is %.2f%%, should be 100%%", report.Coverage))

	}

	return check
}

// checkAuthorizationCompliance checks authorization compliance.

func (m *PolicyManager) checkAuthorizationCompliance(ctx context.Context) ComplianceCheck {
	check := ComplianceCheck{
		Name: "Authorization Policies",

		Description: "All services must have explicit authorization policies",

		Passed: true,
	}

	// Validate policies.

	result, err := m.mesh.ValidatePolicies(ctx, "")
	if err != nil {

		check.Passed = false

		check.Issues = append(check.Issues, fmt.Sprintf("Failed to validate policies: %v", err))

		return check

	}

	if !result.Valid {

		check.Passed = false

		for _, err := range result.Errors {
			check.Issues = append(check.Issues, err.Message)
		}

	}

	if !result.Compliance.ZeroTrustCompliant {

		check.Passed = false

		check.Issues = append(check.Issues, "Not compliant with zero-trust principles")

	}

	return check
}

// checkSegmentationCompliance checks network segmentation compliance.

func (m *PolicyManager) checkSegmentationCompliance(ctx context.Context) ComplianceCheck {
	check := ComplianceCheck{
		Name: "Network Segmentation",

		Description: "Network must be properly segmented",

		Passed: true,
	}

	// Check if network segmentation is enforced.

	result, err := m.mesh.ValidatePolicies(ctx, "")
	if err != nil {

		check.Passed = false

		check.Issues = append(check.Issues, fmt.Sprintf("Failed to validate segmentation: %v", err))

		return check

	}

	if !result.Compliance.NetworkSegmented {

		check.Passed = false

		check.Issues = append(check.Issues, "Network segmentation not properly enforced")

	}

	return check
}

// checkCertificateCompliance checks certificate compliance.

func (m *PolicyManager) checkCertificateCompliance(ctx context.Context) ComplianceCheck {
	check := ComplianceCheck{
		Name: "Certificate Management",

		Description: "All certificates must be valid and not expiring",

		Passed: true,
	}

	// Validate certificate chain.

	if err := m.mesh.ValidateCertificateChain(ctx, ""); err != nil {

		check.Passed = false

		check.Issues = append(check.Issues, fmt.Sprintf("Certificate chain validation failed: %v", err))

	}

	return check
}

// ExportPolicies exports policies to YAML format.

func (m *PolicyManager) ExportPolicies(ctx context.Context) (string, error) {
	policies := m.policyStore.GetAllPolicies()

	data, err := yaml.Marshal(policies)
	if err != nil {
		return "", fmt.Errorf("failed to marshal policies: %w", err)
	}

	return string(data), nil
}

// ImportPolicies imports policies from YAML format.

func (m *PolicyManager) ImportPolicies(ctx context.Context, yamlData string) error {
	var policySet PolicySet

	if err := yaml.Unmarshal([]byte(yamlData), &policySet); err != nil {
		return fmt.Errorf("failed to unmarshal policies: %w", err)
	}

	return m.ApplyPolicySet(ctx, &policySet)
}
