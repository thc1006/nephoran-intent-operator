package compliance

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/storage"
	"github.com/open-policy-agent/opa/storage/inmem"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// OPACompliancePolicyEngine implements comprehensive policy enforcement for all compliance frameworks
type OPACompliancePolicyEngine struct {
	opaStore         storage.Store
	regoPolicies     map[string]*rego.PreparedEvalQuery
	admissionWebhook *AdmissionWebhookController
	networkPolicies  *NetworkPolicyEnforcer
	rbacPolicies     *RBACPolicyEnforcer
	compliancePolicies *CompliancePolicyEnforcer
	runtimePolicies  *RuntimePolicyEnforcer
	auditPolicies    *AuditPolicyEnforcer
	logger           *slog.Logger
	policyCache      *PolicyCache
	violationStore   *ViolationStore
}

// PolicyDefinition represents a comprehensive compliance policy
type PolicyDefinition struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Framework       string                 `json:"framework"` // CIS, NIST, OWASP, GDPR, ORAN, Custom
	Category        string                 `json:"category"`
	Severity        string                 `json:"severity"`
	Description     string                 `json:"description"`
	RegoPolicy      string                 `json:"rego_policy"`
	Enabled         bool                   `json:"enabled"`
	Enforcement     string                 `json:"enforcement"` // deny, warn, dryrun
	Scope           PolicyScope            `json:"scope"`
	Metadata        PolicyMetadata         `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Version         string                 `json:"version"`
	Dependencies    []string               `json:"dependencies"`
	Exceptions      []PolicyException      `json:"exceptions"`
	Parameters      map[string]interface{} `json:"parameters"`
}

type PolicyScope struct {
	Resources   []string `json:"resources"`
	Namespaces  []string `json:"namespaces"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type PolicyMetadata struct {
	Source      string            `json:"source"`
	Reference   string            `json:"reference"`
	Tags        []string          `json:"tags"`
	Attributes  map[string]string `json:"attributes"`
	Owner       string            `json:"owner"`
	Maintainer  string            `json:"maintainer"`
}

type PolicyException struct {
	ID          string            `json:"id"`
	Reason      string            `json:"reason"`
	Conditions  map[string]string `json:"conditions"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	ApprovedBy  string            `json:"approved_by"`
	ApprovedAt  time.Time         `json:"approved_at"`
}

type PolicyEvaluationResult struct {
	PolicyID      string                 `json:"policy_id"`
	Allowed       bool                   `json:"allowed"`
	Violations    []PolicyViolationDetail `json:"violations"`
	Warnings      []string               `json:"warnings"`
	Metadata      map[string]interface{} `json:"metadata"`
	EvaluatedAt   time.Time              `json:"evaluated_at"`
	ExecutionTime time.Duration          `json:"execution_time"`
}

type PolicyViolationDetail struct {
	Message     string                 `json:"message"`
	Path        string                 `json:"path"`
	Value       interface{}            `json:"value"`
	Expected    interface{}            `json:"expected"`
	Severity    string                 `json:"severity"`
	Remediation string                 `json:"remediation"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// =============================================================================
// CIS Kubernetes Benchmark Policies
// =============================================================================

const CISPodSecurityPolicy = `
package cis.pod.security

import rego.v1

# CIS 5.2.1 - Minimize privileged containers
deny_privileged_containers contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    container.securityContext.privileged == true
    msg := sprintf("Container '%s' runs in privileged mode (CIS 5.2.1)", [container.name])
}

# CIS 5.2.2 - Minimize containers with allowPrivilegeEscalation
deny_privilege_escalation contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    container.securityContext.allowPrivilegeEscalation == true
    msg := sprintf("Container '%s' allows privilege escalation (CIS 5.2.2)", [container.name])
}

# CIS 5.2.3 - Minimize containers running with root user
deny_root_user contains msg if {
    input.request.kind.kind == "Pod"
    pod := input.request.object
    not pod.spec.securityContext.runAsNonRoot
    msg := "Pod must run as non-root user (CIS 5.2.3)"
}

# CIS 5.2.4 - Minimize containers with NET_RAW capability
deny_net_raw_capability contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    "NET_RAW" in container.securityContext.capabilities.add
    msg := sprintf("Container '%s' has NET_RAW capability (CIS 5.2.4)", [container.name])
}

# CIS 5.2.5 - Minimize containers mounting Docker socket
deny_docker_socket_mount contains msg if {
    input.request.kind.kind == "Pod"
    volume := input.request.object.spec.volumes[_]
    volume.hostPath.path in ["/var/run/docker.sock", "/run/docker.sock"]
    msg := sprintf("Volume '%s' mounts Docker socket (CIS 5.2.5)", [volume.name])
}

# Main evaluation rule
violation contains result if {
    msgs := deny_privileged_containers
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "CIS-5.2.1-privileged-containers",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := deny_privilege_escalation  
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "CIS-5.2.2-privilege-escalation",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := deny_root_user
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "CIS-5.2.3-root-user",
        "severity": "MEDIUM"
    }
}

violation contains result if {
    msgs := deny_net_raw_capability
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "CIS-5.2.4-net-raw",
        "severity": "MEDIUM"
    }
}

violation contains result if {
    msgs := deny_docker_socket_mount
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "CIS-5.2.5-docker-socket",
        "severity": "HIGH"
    }
}
`

const CISNetworkPolicy = `
package cis.network.security

import rego.v1

# CIS 5.3.1 - Ensure Network Policies are enabled
require_network_policies contains msg if {
    input.request.kind.kind == "Namespace"
    namespace := input.request.object.metadata.name
    not namespace in ["kube-system", "kube-public", "kube-node-lease"]
    not has_network_policy(namespace)
    msg := sprintf("Namespace '%s' missing required Network Policy (CIS 5.3.1)", [namespace])
}

# CIS 5.3.2 - Ensure default deny Network Policy exists
require_default_deny contains msg if {
    input.request.kind.kind == "NetworkPolicy"
    policy := input.request.object
    not is_default_deny_policy(policy)
    msg := sprintf("NetworkPolicy '%s' should implement default deny (CIS 5.3.2)", [policy.metadata.name])
}

has_network_policy(namespace) if {
    # This would check if namespace has associated NetworkPolicies
    # Implementation would query existing NetworkPolicies
    true
}

is_default_deny_policy(policy) if {
    count(policy.spec.podSelector) == 0
    "Ingress" in policy.spec.policyTypes
    "Egress" in policy.spec.policyTypes
    count(policy.spec.ingress) == 0
    count(policy.spec.egress) == 0
}

violation contains result if {
    msgs := require_network_policies
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "CIS-5.3.1-network-policies",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := require_default_deny
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "CIS-5.3.2-default-deny",
        "severity": "MEDIUM"
    }
}
`

// =============================================================================
// NIST Cybersecurity Framework Policies
// =============================================================================

const NISTAccessControlPolicy = `
package nist.access.control

import rego.v1

# NIST PR.AC-1 - Identity and credential management
enforce_identity_management contains msg if {
    input.request.kind.kind == "ServiceAccount"
    sa := input.request.object
    not has_proper_annotations(sa)
    msg := sprintf("ServiceAccount '%s' lacks proper identity management annotations (NIST PR.AC-1)", [sa.metadata.name])
}

# NIST PR.AC-3 - Remote access is managed
control_remote_access contains msg if {
    input.request.kind.kind == "Service"
    service := input.request.object
    service.spec.type == "LoadBalancer"
    not has_access_restrictions(service)
    msg := sprintf("Service '%s' allows unrestricted remote access (NIST PR.AC-3)", [service.metadata.name])
}

# NIST PR.AC-4 - Access permissions are managed
manage_access_permissions contains msg if {
    input.request.kind.kind == "RoleBinding"
    binding := input.request.object
    has_excessive_permissions(binding)
    msg := sprintf("RoleBinding '%s' grants excessive permissions (NIST PR.AC-4)", [binding.metadata.name])
}

# NIST PR.AC-6 - Identities are protected
protect_identities contains msg if {
    input.request.kind.kind == "Secret"
    secret := input.request.object
    secret.type == "kubernetes.io/service-account-token"
    not has_encryption_config(secret)
    msg := sprintf("Secret '%s' lacks proper encryption configuration (NIST PR.AC-6)", [secret.metadata.name])
}

has_proper_annotations(sa) if {
    sa.metadata.annotations["iam.amazonaws.com/role"]
}

has_proper_annotations(sa) if {
    sa.metadata.annotations["azure.workload.identity/client-id"]
}

has_access_restrictions(service) if {
    service.metadata.annotations["service.beta.kubernetes.io/aws-load-balancer-ssl-cert"]
}

has_access_restrictions(service) if {
    service.spec.loadBalancerSourceRanges
    count(service.spec.loadBalancerSourceRanges) > 0
}

has_excessive_permissions(binding) if {
    binding.roleRef.kind == "ClusterRole"
    binding.roleRef.name == "cluster-admin"
    count(binding.subjects) > 1
}

has_encryption_config(secret) if {
    secret.metadata.annotations["encryption.kubernetes.io/provider"]
}

violation contains result if {
    msgs := enforce_identity_management
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "NIST-PR.AC-1-identity-management",
        "severity": "MEDIUM"
    }
}

violation contains result if {
    msgs := control_remote_access
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "NIST-PR.AC-3-remote-access",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := manage_access_permissions
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "NIST-PR.AC-4-access-permissions",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := protect_identities
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "NIST-PR.AC-6-protect-identities",
        "severity": "MEDIUM"
    }
}
`

// =============================================================================
// OWASP Top 10 Security Policies
// =============================================================================

const OWASPSecurityPolicy = `
package owasp.security

import rego.v1

# OWASP A01:2021 - Broken Access Control
prevent_broken_access_control contains msg if {
    input.request.kind.kind == "Ingress"
    ingress := input.request.object
    not has_authentication_annotation(ingress)
    msg := sprintf("Ingress '%s' lacks authentication controls (OWASP A01:2021)", [ingress.metadata.name])
}

# OWASP A02:2021 - Cryptographic Failures
enforce_encryption contains msg if {
    input.request.kind.kind == "Ingress"
    ingress := input.request.object
    not enforces_tls(ingress)
    msg := sprintf("Ingress '%s' does not enforce TLS encryption (OWASP A02:2021)", [ingress.metadata.name])
}

# OWASP A03:2021 - Injection
prevent_injection contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    has_injection_risk(container)
    msg := sprintf("Container '%s' has potential injection vulnerabilities (OWASP A03:2021)", [container.name])
}

# OWASP A05:2021 - Security Misconfiguration
prevent_misconfiguration contains msg if {
    input.request.kind.kind == "Pod"
    pod := input.request.object
    has_security_misconfiguration(pod)
    msg := sprintf("Pod '%s' has security misconfigurations (OWASP A05:2021)", [pod.metadata.name])
}

# OWASP A06:2021 - Vulnerable and Outdated Components
check_vulnerable_components contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    has_vulnerable_image(container.image)
    msg := sprintf("Container '%s' uses vulnerable image '%s' (OWASP A06:2021)", [container.name, container.image])
}

# OWASP A09:2021 - Security Logging and Monitoring Failures
require_security_logging contains msg if {
    input.request.kind.kind == "Pod"
    pod := input.request.object
    not has_logging_configuration(pod)
    msg := sprintf("Pod '%s' lacks proper security logging configuration (OWASP A09:2021)", [pod.metadata.name])
}

has_authentication_annotation(ingress) if {
    ingress.metadata.annotations["nginx.ingress.kubernetes.io/auth-type"]
}

has_authentication_annotation(ingress) if {
    ingress.metadata.annotations["traefik.ingress.kubernetes.io/auth-type"]
}

enforces_tls(ingress) if {
    count(ingress.spec.tls) > 0
}

has_injection_risk(container) if {
    # Check for command injection patterns
    command := container.command[_]
    contains(command, "$(")
}

has_injection_risk(container) if {
    # Check for SQL injection patterns in environment variables
    env := container.env[_]
    contains(env.value, "SELECT")
    contains(env.value, "FROM")
}

has_security_misconfiguration(pod) if {
    # Running as root
    not pod.spec.securityContext.runAsNonRoot
}

has_security_misconfiguration(pod) if {
    # Default service account usage
    pod.spec.serviceAccountName == "default"
    not pod.spec.automountServiceAccountToken == false
}

has_vulnerable_image(image) if {
    # Check for known vulnerable base images
    contains(image, ":latest")
}

has_vulnerable_image(image) if {
    # Check for outdated versions (simplified)
    contains(image, "ubuntu:16.04")
}

has_logging_configuration(pod) if {
    pod.metadata.annotations["fluentd.kubernetes.io/exclude"] == "false"
}

has_logging_configuration(pod) if {
    volume := pod.spec.volumes[_]
    volume.name == "log-volume"
}

violation contains result if {
    msgs := prevent_broken_access_control
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "OWASP-A01-broken-access-control",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := enforce_encryption
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "OWASP-A02-cryptographic-failures",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := prevent_injection
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "OWASP-A03-injection",
        "severity": "CRITICAL"
    }
}

violation contains result if {
    msgs := prevent_misconfiguration
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "OWASP-A05-security-misconfiguration",
        "severity": "MEDIUM"
    }
}

violation contains result if {
    msgs := check_vulnerable_components
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "OWASP-A06-vulnerable-components",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := require_security_logging
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "OWASP-A09-security-logging",
        "severity": "MEDIUM"
    }
}
`

// =============================================================================
// GDPR Data Protection Policies  
// =============================================================================

const GDPRDataProtectionPolicy = `
package gdpr.data.protection

import rego.v1

# GDPR Article 25 - Data Protection by Design and by Default
enforce_privacy_by_design contains msg if {
    input.request.kind.kind == "Deployment"
    deployment := input.request.object
    processes_personal_data(deployment)
    not has_privacy_controls(deployment)
    msg := sprintf("Deployment '%s' processes personal data without privacy controls (GDPR Art. 25)", [deployment.metadata.name])
}

# GDPR Article 32 - Security of Processing
require_security_measures contains msg if {
    input.request.kind.kind == "Pod"
    pod := input.request.object
    handles_personal_data(pod)
    not has_encryption_at_rest(pod)
    msg := sprintf("Pod '%s' handles personal data without encryption (GDPR Art. 32)", [pod.metadata.name])
}

# GDPR Article 35 - Data Protection Impact Assessment
require_dpia_annotation contains msg if {
    input.request.kind.kind == "Deployment"
    deployment := input.request.object
    has_high_privacy_risk(deployment)
    not has_dpia_annotation(deployment)
    msg := sprintf("Deployment '%s' requires DPIA annotation (GDPR Art. 35)", [deployment.metadata.name])
}

# GDPR Article 5 - Data Minimization Principle
enforce_data_minimization contains msg if {
    input.request.kind.kind == "ConfigMap"
    configmap := input.request.object
    contains_excessive_personal_data(configmap)
    msg := sprintf("ConfigMap '%s' violates data minimization principle (GDPR Art. 5)", [configmap.metadata.name])
}

# GDPR Article 17 - Right to Erasure
require_deletion_capability contains msg if {
    input.request.kind.kind == "StatefulSet"
    statefulset := input.request.object
    stores_personal_data(statefulset)
    not has_deletion_capability(statefulset)
    msg := sprintf("StatefulSet '%s' lacks data deletion capability (GDPR Art. 17)", [statefulset.metadata.name])
}

processes_personal_data(deployment) if {
    deployment.metadata.labels["data.classification"] == "personal"
}

processes_personal_data(deployment) if {
    deployment.metadata.annotations["gdpr.data-processing"] == "true"
}

has_privacy_controls(deployment) if {
    deployment.metadata.annotations["privacy.controls"] == "enabled"
}

has_privacy_controls(deployment) if {
    deployment.spec.template.spec.securityContext.runAsNonRoot == true
}

handles_personal_data(pod) if {
    pod.metadata.labels["data.classification"] in ["personal", "sensitive"]
}

has_encryption_at_rest(pod) if {
    volume := pod.spec.volumes[_]
    volume.csi.driver == "secrets-store.csi.k8s.io"
}

has_encryption_at_rest(pod) if {
    pod.metadata.annotations["encryption.kubernetes.io/enabled"] == "true"
}

has_high_privacy_risk(deployment) if {
    deployment.metadata.labels["privacy.risk"] == "high"
}

has_high_privacy_risk(deployment) if {
    deployment.metadata.annotations["gdpr.large-scale-processing"] == "true"
}

has_dpia_annotation(deployment) if {
    deployment.metadata.annotations["gdpr.dpia.completed"]
}

contains_excessive_personal_data(configmap) if {
    # Check for PII patterns in ConfigMap data
    value := configmap.data[_]
    regex.match("\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Z|a-z]{2,}\\b", value)
    not configmap.metadata.annotations["gdpr.purpose-limitation"] == "documented"
}

stores_personal_data(statefulset) if {
    statefulset.metadata.labels["data.type"] == "user-data"
}

has_deletion_capability(statefulset) if {
    statefulset.metadata.annotations["gdpr.deletion.enabled"] == "true"
}

violation contains result if {
    msgs := enforce_privacy_by_design
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "GDPR-Art25-privacy-by-design",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := require_security_measures
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "GDPR-Art32-security-processing",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := require_dpia_annotation
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "GDPR-Art35-dpia-required",
        "severity": "MEDIUM"
    }
}

violation contains result if {
    msgs := enforce_data_minimization
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "GDPR-Art5-data-minimization",
        "severity": "MEDIUM"
    }
}

violation contains result if {
    msgs := require_deletion_capability
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "GDPR-Art17-right-erasure",
        "severity": "MEDIUM"
    }
}
`

// =============================================================================
// O-RAN WG11 Security Policies
// =============================================================================

const ORANSecurityPolicy = `
package oran.wg11.security

import rego.v1

# O-RAN WG11 - E2 Interface Security
enforce_e2_security contains msg if {
    input.request.kind.kind == "Service"
    service := input.request.object
    is_e2_interface_service(service)
    not has_mutual_tls(service)
    msg := sprintf("Service '%s' E2 interface lacks mTLS (O-RAN WG11)", [service.metadata.name])
}

# O-RAN WG11 - A1 Interface Security
enforce_a1_security contains msg if {
    input.request.kind.kind == "Ingress"
    ingress := input.request.object
    is_a1_interface_ingress(ingress)
    not has_oauth2_authentication(ingress)
    msg := sprintf("Ingress '%s' A1 interface lacks OAuth2 (O-RAN WG11)", [ingress.metadata.name])
}

# O-RAN WG11 - O1 Interface Security
enforce_o1_security contains msg if {
    input.request.kind.kind == "Pod"
    pod := input.request.object
    is_o1_interface_pod(pod)
    not has_netconf_security(pod)
    msg := sprintf("Pod '%s' O1 interface lacks NETCONF security (O-RAN WG11)", [pod.metadata.name])
}

# O-RAN WG11 - Zero Trust Architecture
enforce_zero_trust contains msg if {
    input.request.kind.kind == "NetworkPolicy"
    policy := input.request.object
    is_oran_component(policy)
    not implements_zero_trust(policy)
    msg := sprintf("NetworkPolicy '%s' doesn't implement zero trust (O-RAN WG11)", [policy.metadata.name])
}

# O-RAN WG11 - Runtime Security Monitoring
require_runtime_monitoring contains msg if {
    input.request.kind.kind == "Deployment"
    deployment := input.request.object
    is_oran_nf_deployment(deployment)
    not has_runtime_monitoring(deployment)
    msg := sprintf("Deployment '%s' lacks runtime security monitoring (O-RAN WG11)", [deployment.metadata.name])
}

is_e2_interface_service(service) if {
    service.metadata.labels["oran.interface"] == "E2"
}

is_e2_interface_service(service) if {
    service.spec.ports[_].port == 36421
}

has_mutual_tls(service) if {
    service.metadata.annotations["oran.security/mtls"] == "required"
}

has_mutual_tls(service) if {
    service.metadata.annotations["istio.io/tls-mode"] == "MUTUAL"
}

is_a1_interface_ingress(ingress) if {
    ingress.metadata.labels["oran.interface"] == "A1"
}

has_oauth2_authentication(ingress) if {
    ingress.metadata.annotations["nginx.ingress.kubernetes.io/auth-url"]
}

has_oauth2_authentication(ingress) if {
    ingress.metadata.annotations["oran.security/oauth2"] == "enabled"
}

is_o1_interface_pod(pod) if {
    pod.metadata.labels["oran.interface"] == "O1"
}

has_netconf_security(pod) if {
    pod.metadata.annotations["oran.o1/netconf-acm"] == "enabled"
}

has_netconf_security(pod) if {
    container := pod.spec.containers[_]
    container.ports[_].containerPort == 830
    container.env[_].name == "NETCONF_TLS_ENABLED"
}

is_oran_component(policy) if {
    policy.metadata.labels["oran.component"] != ""
}

implements_zero_trust(policy) if {
    # Default deny all
    count(policy.spec.podSelector) == 0
    "Ingress" in policy.spec.policyTypes
    "Egress" in policy.spec.policyTypes
}

is_oran_nf_deployment(deployment) if {
    deployment.metadata.labels["oran.nf-type"] != ""
}

has_runtime_monitoring(deployment) if {
    deployment.spec.template.metadata.annotations["falco.security/monitor"] == "enabled"
}

has_runtime_monitoring(deployment) if {
    deployment.spec.template.spec.containers[_].name == "security-monitor"
}

violation contains result if {
    msgs := enforce_e2_security
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "ORAN-WG11-E2-interface-security",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := enforce_a1_security
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "ORAN-WG11-A1-interface-security",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := enforce_o1_security
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "ORAN-WG11-O1-interface-security",
        "severity": "MEDIUM"
    }
}

violation contains result if {
    msgs := enforce_zero_trust
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "ORAN-WG11-zero-trust",
        "severity": "HIGH"
    }
}

violation contains result if {
    msgs := require_runtime_monitoring
    count(msgs) > 0
    result := {
        "msg": msgs[_],
        "policy": "ORAN-WG11-runtime-monitoring",
        "severity": "MEDIUM"
    }
}
`

// =============================================================================
// OPA Policy Engine Implementation
// =============================================================================

func NewOPACompliancePolicyEngine(logger *slog.Logger) (*OPACompliancePolicyEngine, error) {
	// Initialize OPA store
	store := inmem.New()
	
	engine := &OPACompliancePolicyEngine{
		opaStore:        store,
		regoPolicies:    make(map[string]*rego.PreparedQuery),
		logger:         logger,
		policyCache:    NewPolicyCache(),
		violationStore: NewViolationStore(),
	}

	// Initialize sub-engines
	engine.admissionWebhook = NewAdmissionWebhookController(engine)
	engine.networkPolicies = NewNetworkPolicyEnforcer(engine)
	engine.rbacPolicies = NewRBACPolicyEnforcer(engine)
	engine.compliancePolicies = NewCompliancePolicyEnforcer(engine)
	engine.runtimePolicies = NewRuntimePolicyEnforcer(engine)
	engine.auditPolicies = NewAuditPolicyEnforcer(engine)

	// Load default compliance policies
	if err := engine.loadDefaultPolicies(); err != nil {
		return nil, fmt.Errorf("failed to load default policies: %w", err)
	}

	return engine, nil
}

func (ope *OPACompliancePolicyEngine) loadDefaultPolicies() error {
	defaultPolicies := []PolicyDefinition{
		{
			ID:          "cis-pod-security",
			Name:        "CIS Kubernetes Pod Security Standards",
			Framework:   "CIS",
			Category:    "Pod Security",
			Severity:    "HIGH",
			Description: "Enforces CIS Kubernetes Benchmark pod security controls",
			RegoPolicy:  CISPodSecurityPolicy,
			Enabled:     true,
			Enforcement: "deny",
			Scope: PolicyScope{
				Resources:  []string{"Pod", "Deployment", "StatefulSet", "DaemonSet"},
				Namespaces: []string{"*"},
			},
			CreatedAt: time.Now(),
			Version:   "1.0.0",
		},
		{
			ID:          "cis-network-security",
			Name:        "CIS Kubernetes Network Security",
			Framework:   "CIS",
			Category:    "Network Security",
			Severity:    "HIGH",
			Description: "Enforces CIS Kubernetes network security controls",
			RegoPolicy:  CISNetworkPolicy,
			Enabled:     true,
			Enforcement: "deny",
			Scope: PolicyScope{
				Resources: []string{"NetworkPolicy", "Namespace"},
				Namespaces: []string{"*"},
			},
			CreatedAt: time.Now(),
			Version:   "1.0.0",
		},
		{
			ID:          "nist-access-control",
			Name:        "NIST Access Control Framework",
			Framework:   "NIST",
			Category:    "Access Control",
			Severity:    "MEDIUM",
			Description: "Implements NIST Cybersecurity Framework access controls",
			RegoPolicy:  NISTAccessControlPolicy,
			Enabled:     true,
			Enforcement: "warn",
			Scope: PolicyScope{
				Resources: []string{"ServiceAccount", "Service", "RoleBinding", "Secret"},
				Namespaces: []string{"*"},
			},
			CreatedAt: time.Now(),
			Version:   "1.0.0",
		},
		{
			ID:          "owasp-security-controls",
			Name:        "OWASP Top 10 Security Controls",
			Framework:   "OWASP",
			Category:    "Application Security",
			Severity:    "HIGH",
			Description: "Implements OWASP Top 10 security controls",
			RegoPolicy:  OWASPSecurityPolicy,
			Enabled:     true,
			Enforcement: "deny",
			Scope: PolicyScope{
				Resources: []string{"Pod", "Ingress", "Service"},
				Namespaces: []string{"*"},
			},
			CreatedAt: time.Now(),
			Version:   "1.0.0",
		},
		{
			ID:          "gdpr-data-protection",
			Name:        "GDPR Data Protection Controls",
			Framework:   "GDPR",
			Category:    "Data Protection",
			Severity:    "HIGH",
			Description: "Enforces GDPR data protection requirements",
			RegoPolicy:  GDPRDataProtectionPolicy,
			Enabled:     true,
			Enforcement: "deny",
			Scope: PolicyScope{
				Resources: []string{"Pod", "Deployment", "ConfigMap", "StatefulSet"},
				Namespaces: []string{"*"},
			},
			CreatedAt: time.Now(),
			Version:   "1.0.0",
		},
		{
			ID:          "oran-wg11-security",
			Name:        "O-RAN WG11 Security Requirements",
			Framework:   "ORAN",
			Category:    "Interface Security",
			Severity:    "HIGH",
			Description: "Implements O-RAN WG11 security specifications",
			RegoPolicy:  ORANSecurityPolicy,
			Enabled:     true,
			Enforcement: "deny",
			Scope: PolicyScope{
				Resources: []string{"Pod", "Service", "Ingress", "NetworkPolicy", "Deployment"},
				Namespaces: []string{"*"},
			},
			CreatedAt: time.Now(),
			Version:   "1.0.0",
		},
	}

	for _, policy := range defaultPolicies {
		if err := ope.LoadPolicy(policy); err != nil {
			ope.logger.Error("Failed to load default policy", "policy_id", policy.ID, "error", err)
			continue
		}
		ope.logger.Info("Loaded default policy", "policy_id", policy.ID, "framework", policy.Framework)
	}

	return nil
}

func (ope *OPACompliancePolicyEngine) LoadPolicy(policy PolicyDefinition) error {
	// Parse and prepare Rego policy
	query := rego.New(
		rego.Query("data."+strings.Replace(policy.ID, "-", "_", -1)+".violation"),
		rego.Module(policy.ID+".rego", policy.RegoPolicy),
		rego.Store(ope.opaStore),
	)

	preparedQuery, err := query.PrepareForEval(context.Background())
	if err != nil {
		return fmt.Errorf("failed to prepare policy %s: %w", policy.ID, err)
	}

	ope.regoPolicies[policy.ID] = &preparedQuery
	ope.policyCache.Set(policy.ID, policy)

	ope.logger.Info("Policy loaded successfully",
		"policy_id", policy.ID,
		"framework", policy.Framework,
		"enforcement", policy.Enforcement)

	return nil
}

func (ope *OPACompliancePolicyEngine) EvaluateAdmission(ctx context.Context, req admission.Request) (*PolicyEvaluationResult, error) {
	startTime := time.Now()
	
	input := map[string]interface{}{
		"request": map[string]interface{}{
			"uid":       req.UID,
			"kind":      req.Kind,
			"resource":  req.Resource,
			"namespace": req.Namespace,
			"object":    req.Object,
			"operation": req.Operation,
		},
	}

	allViolations := []PolicyViolationDetail{}
	allWarnings := []string{}
	allowed := true

	// Evaluate all applicable policies
	for policyID, preparedQuery := range ope.regoPolicies {
		policy, exists := ope.policyCache.Get(policyID)
		if !exists || !policy.Enabled {
			continue
		}

		// Check if policy applies to this resource
		if !ope.isPolicyApplicable(policy, req) {
			continue
		}

		// Evaluate policy
		results, err := preparedQuery.Eval(ctx, rego.EvalInput(input))
		if err != nil {
			ope.logger.Error("Policy evaluation failed", "policy_id", policyID, "error", err)
			continue
		}

		// Process results
		for _, result := range results {
			if violations, ok := result.Expressions[0].Value.([]interface{}); ok {
				for _, violation := range violations {
					if violationMap, ok := violation.(map[string]interface{}); ok {
						violationDetail := ope.parseViolationDetail(violationMap, policy)
						allViolations = append(allViolations, violationDetail)

						// Determine if request should be denied
						if policy.Enforcement == "deny" && violationDetail.Severity != "LOW" {
							allowed = false
						} else if policy.Enforcement == "warn" {
							allWarnings = append(allWarnings, violationDetail.Message)
						}
					}
				}
			}
		}
	}

	result := &PolicyEvaluationResult{
		PolicyID:      "multi-policy",
		Allowed:       allowed,
		Violations:    allViolations,
		Warnings:      allWarnings,
		EvaluatedAt:   time.Now(),
		ExecutionTime: time.Since(startTime),
		Metadata: map[string]interface{}{
			"policies_evaluated": len(ope.regoPolicies),
			"violations_count":   len(allViolations),
			"warnings_count":     len(allWarnings),
		},
	}

	// Store violations for audit
	for _, violation := range allViolations {
		ope.violationStore.Store(PolicyViolation{
			ID:          fmt.Sprintf("violation-%d", time.Now().UnixNano()),
			PolicyID:    violation.Message, // Simplified
			Resource:    fmt.Sprintf("%s/%s", req.Kind.Kind, req.Name),
			Namespace:   req.Namespace,
			Violation:   violation.Message,
			Severity:    violation.Severity,
			Timestamp:   time.Now(),
			Enforcement: "admission",
		})
	}

	ope.logger.Info("Policy evaluation completed",
		"allowed", allowed,
		"violations", len(allViolations),
		"warnings", len(allWarnings),
		"execution_time", result.ExecutionTime)

	return result, nil
}

func (ope *OPACompliancePolicyEngine) isPolicyApplicable(policy PolicyDefinition, req admission.Request) bool {
	// Check resource type
	resourceMatches := false
	for _, resource := range policy.Scope.Resources {
		if resource == "*" || resource == req.Kind.Kind {
			resourceMatches = true
			break
		}
	}
	if !resourceMatches {
		return false
	}

	// Check namespace
	namespaceMatches := false
	for _, namespace := range policy.Scope.Namespaces {
		if namespace == "*" || namespace == req.Namespace {
			namespaceMatches = true
			break
		}
	}
	if !namespaceMatches {
		return false
	}

	return true
}

func (ope *OPACompliancePolicyEngine) parseViolationDetail(violationMap map[string]interface{}, policy PolicyDefinition) PolicyViolationDetail {
	detail := PolicyViolationDetail{
		Metadata: map[string]interface{}{
			"policy_id":    policy.ID,
			"framework":    policy.Framework,
			"enforcement":  policy.Enforcement,
		},
	}

	if msg, ok := violationMap["msg"].(string); ok {
		detail.Message = msg
	}

	if severity, ok := violationMap["severity"].(string); ok {
		detail.Severity = severity
	} else {
		detail.Severity = policy.Severity
	}

	if policyName, ok := violationMap["policy"].(string); ok {
		detail.Metadata["policy_name"] = policyName
	}

	return detail
}

// =============================================================================
// Supporting Types and Implementations
// =============================================================================

type PolicyCache struct {
	policies map[string]PolicyDefinition
}

func NewPolicyCache() *PolicyCache {
	return &PolicyCache{
		policies: make(map[string]PolicyDefinition),
	}
}

func (pc *PolicyCache) Set(id string, policy PolicyDefinition) {
	pc.policies[id] = policy
}

func (pc *PolicyCache) Get(id string) (PolicyDefinition, bool) {
	policy, exists := pc.policies[id]
	return policy, exists
}

type ViolationStore struct {
	violations []PolicyViolation
}

type PolicyViolation struct {
	ID          string    `json:"id"`
	PolicyID    string    `json:"policy_id"`
	Resource    string    `json:"resource"`
	Namespace   string    `json:"namespace"`
	Violation   string    `json:"violation"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Enforcement string    `json:"enforcement"`
}

func NewViolationStore() *ViolationStore {
	return &ViolationStore{
		violations: make([]PolicyViolation, 0),
	}
}

func (vs *ViolationStore) Store(violation PolicyViolation) {
	vs.violations = append(vs.violations, violation)
	// Keep only last 10000 violations
	if len(vs.violations) > 10000 {
		vs.violations = vs.violations[1:]
	}
}

func (vs *ViolationStore) GetViolations(limit int) []PolicyViolation {
	if limit > len(vs.violations) {
		limit = len(vs.violations)
	}
	return vs.violations[len(vs.violations)-limit:]
}

// Supporting sub-engines (simplified implementations)
type AdmissionWebhookController struct {
	engine *OPACompliancePolicyEngine
}

func NewAdmissionWebhookController(engine *OPACompliancePolicyEngine) *AdmissionWebhookController {
	return &AdmissionWebhookController{engine: engine}
}

type NetworkPolicyEnforcer struct {
	engine *OPACompliancePolicyEngine
}

func NewNetworkPolicyEnforcer(engine *OPACompliancePolicyEngine) *NetworkPolicyEnforcer {
	return &NetworkPolicyEnforcer{engine: engine}
}

type RBACPolicyEnforcer struct {
	engine *OPACompliancePolicyEngine
}

func NewRBACPolicyEnforcer(engine *OPACompliancePolicyEngine) *RBACPolicyEnforcer {
	return &RBACPolicyEnforcer{engine: engine}
}

type CompliancePolicyEnforcer struct {
	engine *OPACompliancePolicyEngine
}

func NewCompliancePolicyEnforcer(engine *OPACompliancePolicyEngine) *CompliancePolicyEnforcer {
	return &CompliancePolicyEnforcer{engine: engine}
}

type RuntimePolicyEnforcer struct {
	engine *OPACompliancePolicyEngine
}

func NewRuntimePolicyEnforcer(engine *OPACompliancePolicyEngine) *RuntimePolicyEnforcer {
	return &RuntimePolicyEnforcer{engine: engine}
}

type AuditPolicyEnforcer struct {
	engine *OPACompliancePolicyEngine
}

func NewAuditPolicyEnforcer(engine *OPACompliancePolicyEngine) *AuditPolicyEnforcer {
	return &AuditPolicyEnforcer{engine: engine}
}