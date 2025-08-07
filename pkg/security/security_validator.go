package security

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecurityValidator performs comprehensive security validation
type SecurityValidator struct {
	client    client.Client
	namespace string
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(client client.Client, namespace string) *SecurityValidator {
	return &SecurityValidator{
		client:    client,
		namespace: namespace,
	}
}

// SecurityValidationReport contains comprehensive security validation results
type SecurityValidationReport struct {
	Timestamp              metav1.Time
	Namespace              string
	Compliant              bool
	ContainerSecurityIssues []SecurityIssue
	RBACIssues             []SecurityIssue
	NetworkPolicyIssues    []SecurityIssue
	SecretManagementIssues []SecurityIssue
	TLSConfigurationIssues []SecurityIssue
	Score                  int // Security score 0-100
}

// SecurityIssue represents a security issue found during validation
type SecurityIssue struct {
	Severity    string // Critical, High, Medium, Low
	Component   string
	Description string
	Remediation string
}

// ValidateAll performs comprehensive security validation
func (v *SecurityValidator) ValidateAll(ctx context.Context) (*SecurityValidationReport, error) {
	logger := log.FromContext(ctx)
	
	report := &SecurityValidationReport{
		Timestamp:               metav1.Now(),
		Namespace:               v.namespace,
		ContainerSecurityIssues: []SecurityIssue{},
		RBACIssues:             []SecurityIssue{},
		NetworkPolicyIssues:    []SecurityIssue{},
		SecretManagementIssues: []SecurityIssue{},
		TLSConfigurationIssues: []SecurityIssue{},
	}
	
	// Validate container security
	containerIssues, err := v.ValidateContainerSecurity(ctx)
	if err != nil {
		logger.Error(err, "Container security validation failed")
	}
	report.ContainerSecurityIssues = append(report.ContainerSecurityIssues, containerIssues...)
	
	// Validate RBAC permissions
	rbacIssues, err := v.ValidateRBACPermissions(ctx)
	if err != nil {
		logger.Error(err, "RBAC validation failed")
	}
	report.RBACIssues = append(report.RBACIssues, rbacIssues...)
	
	// Validate network policies
	networkIssues, err := v.ValidateNetworkPolicies(ctx)
	if err != nil {
		logger.Error(err, "Network policy validation failed")
	}
	report.NetworkPolicyIssues = append(report.NetworkPolicyIssues, networkIssues...)
	
	// Validate secret management
	secretIssues, err := v.ValidateSecretManagement(ctx)
	if err != nil {
		logger.Error(err, "Secret management validation failed")
	}
	report.SecretManagementIssues = append(report.SecretManagementIssues, secretIssues...)
	
	// Validate TLS configuration
	tlsIssues, err := v.ValidateTLSConfiguration(ctx)
	if err != nil {
		logger.Error(err, "TLS configuration validation failed")
	}
	report.TLSConfigurationIssues = append(report.TLSConfigurationIssues, tlsIssues...)
	
	// Calculate security score
	report.Score = v.calculateSecurityScore(report)
	report.Compliant = report.Score >= 80 // 80% threshold for compliance
	
	logger.Info("Security validation completed", 
		"score", report.Score,
		"compliant", report.Compliant,
		"totalIssues", v.getTotalIssueCount(report))
	
	return report, nil
}

// ValidateContainerSecurity validates container security contexts
func (v *SecurityValidator) ValidateContainerSecurity(ctx context.Context) ([]SecurityIssue, error) {
	issues := []SecurityIssue{}
	
	// Check deployments
	deployments := &appsv1.DeploymentList{}
	if err := v.client.List(ctx, deployments, client.InNamespace(v.namespace)); err != nil {
		return issues, fmt.Errorf("failed to list deployments: %w", err)
	}
	
	for _, deployment := range deployments.Items {
		// Check pod security context
		podSpec := deployment.Spec.Template.Spec
		
		if podSpec.SecurityContext == nil {
			issues = append(issues, SecurityIssue{
				Severity:    "High",
				Component:   fmt.Sprintf("Deployment/%s", deployment.Name),
				Description: "Missing pod security context",
				Remediation: "Add pod security context with runAsNonRoot and fsGroup settings",
			})
		} else {
			// Check runAsNonRoot
			if podSpec.SecurityContext.RunAsNonRoot == nil || !*podSpec.SecurityContext.RunAsNonRoot {
				issues = append(issues, SecurityIssue{
					Severity:    "High",
					Component:   fmt.Sprintf("Deployment/%s", deployment.Name),
					Description: "Pod not configured to run as non-root",
					Remediation: "Set securityContext.runAsNonRoot to true",
				})
			}
			
			// Check fsGroup
			if podSpec.SecurityContext.FSGroup == nil {
				issues = append(issues, SecurityIssue{
					Severity:    "Medium",
					Component:   fmt.Sprintf("Deployment/%s", deployment.Name),
					Description: "Missing fsGroup in pod security context",
					Remediation: "Set securityContext.fsGroup to a non-zero value",
				})
			}
		}
		
		// Check container security contexts
		for _, container := range podSpec.Containers {
			if container.SecurityContext == nil {
				issues = append(issues, SecurityIssue{
					Severity:    "High",
					Component:   fmt.Sprintf("Deployment/%s/Container/%s", deployment.Name, container.Name),
					Description: "Missing container security context",
					Remediation: "Add container security context with proper restrictions",
				})
				continue
			}
			
			// Check readOnlyRootFilesystem
			if container.SecurityContext.ReadOnlyRootFilesystem == nil || !*container.SecurityContext.ReadOnlyRootFilesystem {
				issues = append(issues, SecurityIssue{
					Severity:    "Medium",
					Component:   fmt.Sprintf("Deployment/%s/Container/%s", deployment.Name, container.Name),
					Description: "Container root filesystem is not read-only",
					Remediation: "Set securityContext.readOnlyRootFilesystem to true",
				})
			}
			
			// Check allowPrivilegeEscalation
			if container.SecurityContext.AllowPrivilegeEscalation != nil && *container.SecurityContext.AllowPrivilegeEscalation {
				issues = append(issues, SecurityIssue{
					Severity:    "Critical",
					Component:   fmt.Sprintf("Deployment/%s/Container/%s", deployment.Name, container.Name),
					Description: "Container allows privilege escalation",
					Remediation: "Set securityContext.allowPrivilegeEscalation to false",
				})
			}
			
			// Check privileged
			if container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
				issues = append(issues, SecurityIssue{
					Severity:    "Critical",
					Component:   fmt.Sprintf("Deployment/%s/Container/%s", deployment.Name, container.Name),
					Description: "Container running in privileged mode",
					Remediation: "Remove privileged mode or use specific capabilities",
				})
			}
			
			// Check capabilities
			if container.SecurityContext.Capabilities != nil {
				if len(container.SecurityContext.Capabilities.Add) > 0 {
					for _, cap := range container.SecurityContext.Capabilities.Add {
						if cap == "ALL" || cap == "SYS_ADMIN" {
							issues = append(issues, SecurityIssue{
								Severity:    "High",
								Component:   fmt.Sprintf("Deployment/%s/Container/%s", deployment.Name, container.Name),
								Description: fmt.Sprintf("Container has dangerous capability: %s", cap),
								Remediation: "Remove or restrict dangerous capabilities",
							})
						}
					}
				}
				
				if container.SecurityContext.Capabilities.Drop == nil || len(container.SecurityContext.Capabilities.Drop) == 0 {
					issues = append(issues, SecurityIssue{
						Severity:    "Medium",
						Component:   fmt.Sprintf("Deployment/%s/Container/%s", deployment.Name, container.Name),
						Description: "Container not dropping any capabilities",
						Remediation: "Drop ALL capabilities and add only required ones",
					})
				}
			}
			
			// Check for latest image tag
			if strings.Contains(container.Image, ":latest") || !strings.Contains(container.Image, ":") {
				issues = append(issues, SecurityIssue{
					Severity:    "Medium",
					Component:   fmt.Sprintf("Deployment/%s/Container/%s", deployment.Name, container.Name),
					Description: "Container using latest or untagged image",
					Remediation: "Use specific image tags for reproducibility and security",
				})
			}
		}
		
		// Check service account
		if podSpec.ServiceAccountName == "" || podSpec.ServiceAccountName == "default" {
			issues = append(issues, SecurityIssue{
				Severity:    "Medium",
				Component:   fmt.Sprintf("Deployment/%s", deployment.Name),
				Description: "Using default service account",
				Remediation: "Create and use a dedicated service account with minimal permissions",
			})
		}
		
		// Check automountServiceAccountToken
		if podSpec.AutomountServiceAccountToken == nil || *podSpec.AutomountServiceAccountToken {
			issues = append(issues, SecurityIssue{
				Severity:    "Low",
				Component:   fmt.Sprintf("Deployment/%s", deployment.Name),
				Description: "Service account token automatically mounted",
				Remediation: "Set automountServiceAccountToken to false if not needed",
			})
		}
	}
	
	return issues, nil
}

// ValidateRBACPermissions validates RBAC configurations
func (v *SecurityValidator) ValidateRBACPermissions(ctx context.Context) ([]SecurityIssue, error) {
	issues := []SecurityIssue{}
	
	// Check ClusterRoles
	clusterRoles := &rbacv1.ClusterRoleList{}
	if err := v.client.List(ctx, clusterRoles); err != nil {
		return issues, fmt.Errorf("failed to list ClusterRoles: %w", err)
	}
	
	for _, cr := range clusterRoles.Items {
		// Skip system roles
		if strings.HasPrefix(cr.Name, "system:") {
			continue
		}
		
		for _, rule := range cr.Rules {
			// Check for wildcard permissions
			for _, apiGroup := range rule.APIGroups {
				if apiGroup == "*" {
					issues = append(issues, SecurityIssue{
						Severity:    "High",
						Component:   fmt.Sprintf("ClusterRole/%s", cr.Name),
						Description: "Wildcard API group permission",
						Remediation: "Use specific API groups instead of wildcards",
					})
				}
			}
			
			for _, resource := range rule.Resources {
				if resource == "*" {
					issues = append(issues, SecurityIssue{
						Severity:    "High",
						Component:   fmt.Sprintf("ClusterRole/%s", cr.Name),
						Description: "Wildcard resource permission",
						Remediation: "Use specific resources instead of wildcards",
					})
				}
				
				// Check for sensitive resources
				if resource == "secrets" || resource == "configmaps" {
					if contains(rule.Verbs, "list") || contains(rule.Verbs, "get") {
						if contains(rule.Verbs, "*") {
							issues = append(issues, SecurityIssue{
								Severity:    "High",
								Component:   fmt.Sprintf("ClusterRole/%s", cr.Name),
								Description: fmt.Sprintf("Broad access to sensitive resource: %s", resource),
								Remediation: "Restrict access to specific secrets/configmaps using resourceNames",
							})
						}
					}
				}
			}
			
			for _, verb := range rule.Verbs {
				if verb == "*" {
					issues = append(issues, SecurityIssue{
						Severity:    "High",
						Component:   fmt.Sprintf("ClusterRole/%s", cr.Name),
						Description: "Wildcard verb permission",
						Remediation: "Use specific verbs instead of wildcards",
					})
				}
			}
			
			// Check for escalation permissions
			if (contains(rule.Resources, "clusterroles") || contains(rule.Resources, "roles")) &&
				(contains(rule.Verbs, "bind") || contains(rule.Verbs, "escalate")) {
				issues = append(issues, SecurityIssue{
					Severity:    "Critical",
					Component:   fmt.Sprintf("ClusterRole/%s", cr.Name),
					Description: "Potential privilege escalation permission",
					Remediation: "Review and restrict role binding permissions",
				})
			}
		}
	}
	
	// Check ClusterRoleBindings
	clusterRoleBindings := &rbacv1.ClusterRoleBindingList{}
	if err := v.client.List(ctx, clusterRoleBindings); err != nil {
		return issues, fmt.Errorf("failed to list ClusterRoleBindings: %w", err)
	}
	
	for _, crb := range clusterRoleBindings.Items {
		// Check for cluster-admin bindings
		if crb.RoleRef.Name == "cluster-admin" {
			for _, subject := range crb.Subjects {
				if subject.Kind == "User" || subject.Kind == "ServiceAccount" {
					issues = append(issues, SecurityIssue{
						Severity:    "Critical",
						Component:   fmt.Sprintf("ClusterRoleBinding/%s", crb.Name),
						Description: fmt.Sprintf("cluster-admin bound to %s/%s", subject.Kind, subject.Name),
						Remediation: "Avoid using cluster-admin; create specific roles with minimal permissions",
					})
				}
			}
		}
		
		// Check for system:masters group binding
		for _, subject := range crb.Subjects {
			if subject.Kind == "Group" && subject.Name == "system:masters" {
				issues = append(issues, SecurityIssue{
					Severity:    "Critical",
					Component:   fmt.Sprintf("ClusterRoleBinding/%s", crb.Name),
					Description: "Binding to system:masters group",
					Remediation: "Remove system:masters group binding; use specific roles",
				})
			}
		}
	}
	
	return issues, nil
}

// ValidateNetworkPolicies validates network policy effectiveness
func (v *SecurityValidator) ValidateNetworkPolicies(ctx context.Context) ([]SecurityIssue, error) {
	issues := []SecurityIssue{}
	
	// List network policies
	policies := &networkingv1.NetworkPolicyList{}
	if err := v.client.List(ctx, policies, client.InNamespace(v.namespace)); err != nil {
		return issues, fmt.Errorf("failed to list network policies: %w", err)
	}
	
	// Check for default deny-all policy
	hasDenyAll := false
	for _, policy := range policies.Items {
		if policy.Name == "default-deny-all" || 
			(len(policy.Spec.Ingress) == 0 && len(policy.Spec.Egress) == 0 && 
			 len(policy.Spec.PodSelector.MatchLabels) == 0) {
			hasDenyAll = true
			break
		}
	}
	
	if !hasDenyAll {
		issues = append(issues, SecurityIssue{
			Severity:    "Critical",
			Component:   "NetworkPolicy",
			Description: "No default deny-all network policy found",
			Remediation: "Create a default deny-all policy as zero-trust baseline",
		})
	}
	
	// Check for overly permissive policies
	for _, policy := range policies.Items {
		// Check ingress rules
		for i, ingress := range policy.Spec.Ingress {
			if len(ingress.From) == 0 {
				issues = append(issues, SecurityIssue{
					Severity:    "High",
					Component:   fmt.Sprintf("NetworkPolicy/%s/Ingress[%d]", policy.Name, i),
					Description: "Ingress rule allows traffic from all sources",
					Remediation: "Specify allowed sources using podSelector or namespaceSelector",
				})
			}
			
			if len(ingress.Ports) == 0 {
				issues = append(issues, SecurityIssue{
					Severity:    "Medium",
					Component:   fmt.Sprintf("NetworkPolicy/%s/Ingress[%d]", policy.Name, i),
					Description: "Ingress rule allows all ports",
					Remediation: "Specify allowed ports explicitly",
				})
			}
		}
		
		// Check egress rules
		for i, egress := range policy.Spec.Egress {
			hasUnrestrictedInternet := false
			for _, to := range egress.To {
				if to.IPBlock != nil && to.IPBlock.CIDR == "0.0.0.0/0" {
					if len(to.IPBlock.Except) == 0 {
						hasUnrestrictedInternet = true
					}
				}
			}
			
			if hasUnrestrictedInternet {
				issues = append(issues, SecurityIssue{
					Severity:    "High",
					Component:   fmt.Sprintf("NetworkPolicy/%s/Egress[%d]", policy.Name, i),
					Description: "Egress rule allows unrestricted internet access",
					Remediation: "Restrict egress to specific IP ranges or use except blocks",
				})
			}
			
			if len(egress.Ports) == 0 {
				issues = append(issues, SecurityIssue{
					Severity:    "Medium",
					Component:   fmt.Sprintf("NetworkPolicy/%s/Egress[%d]", policy.Name, i),
					Description: "Egress rule allows all ports",
					Remediation: "Specify allowed ports explicitly",
				})
			}
		}
	}
	
	// Check pod coverage
	pods := &corev1.PodList{}
	if err := v.client.List(ctx, pods, client.InNamespace(v.namespace)); err != nil {
		return issues, fmt.Errorf("failed to list pods: %w", err)
	}
	
	uncoveredPods := []string{}
	for _, pod := range pods.Items {
		covered := false
		for _, policy := range policies.Items {
			if v.isPodCoveredByPolicy(pod, policy) {
				covered = true
				break
			}
		}
		if !covered {
			uncoveredPods = append(uncoveredPods, pod.Name)
		}
	}
	
	if len(uncoveredPods) > 0 {
		issues = append(issues, SecurityIssue{
			Severity:    "High",
			Component:   "NetworkPolicy",
			Description: fmt.Sprintf("%d pods not covered by network policies", len(uncoveredPods)),
			Remediation: "Create network policies for all pods",
		})
	}
	
	return issues, nil
}

// ValidateSecretManagement validates secret management practices
func (v *SecurityValidator) ValidateSecretManagement(ctx context.Context) ([]SecurityIssue, error) {
	issues := []SecurityIssue{}
	
	// List secrets
	secrets := &corev1.SecretList{}
	if err := v.client.List(ctx, secrets, client.InNamespace(v.namespace)); err != nil {
		return issues, fmt.Errorf("failed to list secrets: %w", err)
	}
	
	for _, secret := range secrets.Items {
		// Skip service account tokens
		if secret.Type == corev1.SecretTypeServiceAccountToken {
			continue
		}
		
		// Check for plain text passwords in data
		for key, value := range secret.Data {
			keyLower := strings.ToLower(key)
			if strings.Contains(keyLower, "password") || 
			   strings.Contains(keyLower, "passwd") || 
			   strings.Contains(keyLower, "pwd") {
				// Check if value looks like plain text
				if len(value) > 0 && v.isLikelyPlainText(value) {
					issues = append(issues, SecurityIssue{
						Severity:    "Critical",
						Component:   fmt.Sprintf("Secret/%s", secret.Name),
						Description: fmt.Sprintf("Potential plain text password in key: %s", key),
						Remediation: "Use proper encryption or hashing for passwords",
					})
				}
			}
			
			// Check for API keys and tokens
			if strings.Contains(keyLower, "apikey") || 
			   strings.Contains(keyLower, "token") || 
			   strings.Contains(keyLower, "key") {
				if len(value) < 16 {
					issues = append(issues, SecurityIssue{
						Severity:    "High",
						Component:   fmt.Sprintf("Secret/%s", secret.Name),
						Description: fmt.Sprintf("Weak API key/token in key: %s", key),
						Remediation: "Use strong, randomly generated API keys (min 16 characters)",
					})
				}
			}
		}
		
		// Check secret annotations
		if secret.Annotations == nil || secret.Annotations["security.nephoran.io/encrypted"] != "true" {
			issues = append(issues, SecurityIssue{
				Severity:    "Medium",
				Component:   fmt.Sprintf("Secret/%s", secret.Name),
				Description: "Secret not marked as encrypted",
				Remediation: "Ensure secrets are encrypted at rest",
			})
		}
		
		// Check for rotation policy
		if secret.Annotations == nil || secret.Annotations["security.nephoran.io/rotation-policy"] == "" {
			issues = append(issues, SecurityIssue{
				Severity:    "Low",
				Component:   fmt.Sprintf("Secret/%s", secret.Name),
				Description: "No rotation policy defined for secret",
				Remediation: "Define and implement secret rotation policy",
			})
		}
	}
	
	// Check ConfigMaps for sensitive data
	configMaps := &corev1.ConfigMapList{}
	if err := v.client.List(ctx, configMaps, client.InNamespace(v.namespace)); err != nil {
		return issues, fmt.Errorf("failed to list configmaps: %w", err)
	}
	
	for _, cm := range configMaps.Items {
		for key, value := range cm.Data {
			// Check for sensitive patterns in ConfigMaps
			valueLower := strings.ToLower(value)
			if strings.Contains(valueLower, "password") ||
			   strings.Contains(valueLower, "apikey") ||
			   strings.Contains(valueLower, "secret") ||
			   strings.Contains(valueLower, "token") {
				issues = append(issues, SecurityIssue{
					Severity:    "High",
					Component:   fmt.Sprintf("ConfigMap/%s", cm.Name),
					Description: fmt.Sprintf("Potential sensitive data in ConfigMap key: %s", key),
					Remediation: "Move sensitive data to Secrets, not ConfigMaps",
				})
			}
		}
	}
	
	return issues, nil
}

// ValidateTLSConfiguration validates TLS/mTLS configuration
func (v *SecurityValidator) ValidateTLSConfiguration(ctx context.Context) ([]SecurityIssue, error) {
	issues := []SecurityIssue{}
	
	// Check TLS secrets
	secrets := &corev1.SecretList{}
	if err := v.client.List(ctx, secrets, client.InNamespace(v.namespace)); err != nil {
		return issues, fmt.Errorf("failed to list secrets: %w", err)
	}
	
	for _, secret := range secrets.Items {
		if secret.Type != corev1.SecretTypeTLS {
			continue
		}
		
		// Validate certificate
		if certData, exists := secret.Data["tls.crt"]; exists {
			certIssues := v.validateCertificate(certData, secret.Name)
			issues = append(issues, certIssues...)
		}
		
		// Validate private key
		if keyData, exists := secret.Data["tls.key"]; exists {
			keyIssues := v.validatePrivateKey(keyData, secret.Name)
			issues = append(issues, keyIssues...)
		}
	}
	
	// Check Ingress TLS configuration
	ingresses := &networkingv1.IngressList{}
	if err := v.client.List(ctx, ingresses, client.InNamespace(v.namespace)); err != nil {
		return issues, fmt.Errorf("failed to list ingresses: %w", err)
	}
	
	for _, ingress := range ingresses.Items {
		if len(ingress.Spec.TLS) == 0 {
			issues = append(issues, SecurityIssue{
				Severity:    "High",
				Component:   fmt.Sprintf("Ingress/%s", ingress.Name),
				Description: "Ingress does not have TLS configured",
				Remediation: "Configure TLS for all ingress resources",
			})
			continue
		}
		
		for _, tls := range ingress.Spec.TLS {
			if tls.SecretName == "" {
				issues = append(issues, SecurityIssue{
					Severity:    "High",
					Component:   fmt.Sprintf("Ingress/%s", ingress.Name),
					Description: "TLS configuration missing secret reference",
					Remediation: "Specify TLS secret for ingress",
				})
			}
		}
		
		// Check for HTTP to HTTPS redirect
		if ingress.Annotations == nil || 
		   ingress.Annotations["nginx.ingress.kubernetes.io/ssl-redirect"] != "true" {
			issues = append(issues, SecurityIssue{
				Severity:    "Medium",
				Component:   fmt.Sprintf("Ingress/%s", ingress.Name),
				Description: "HTTP to HTTPS redirect not configured",
				Remediation: "Enable SSL redirect annotation",
			})
		}
		
		// Check TLS version
		minTLSVersion := ingress.Annotations["nginx.ingress.kubernetes.io/ssl-protocols"]
		if minTLSVersion == "" || strings.Contains(minTLSVersion, "TLSv1.0") || strings.Contains(minTLSVersion, "TLSv1.1") {
			issues = append(issues, SecurityIssue{
				Severity:    "High",
				Component:   fmt.Sprintf("Ingress/%s", ingress.Name),
				Description: "Weak TLS version allowed (< TLS 1.2)",
				Remediation: "Configure minimum TLS version to 1.2 or higher",
			})
		}
		
		// Check cipher suites
		ciphers := ingress.Annotations["nginx.ingress.kubernetes.io/ssl-ciphers"]
		if ciphers != "" && v.hasWeakCiphers(ciphers) {
			issues = append(issues, SecurityIssue{
				Severity:    "Medium",
				Component:   fmt.Sprintf("Ingress/%s", ingress.Name),
				Description: "Weak cipher suites configured",
				Remediation: "Use only strong cipher suites (AES-GCM, ChaCha20-Poly1305)",
			})
		}
	}
	
	return issues, nil
}

// validateCertificate validates a TLS certificate
func (v *SecurityValidator) validateCertificate(certData []byte, secretName string) []SecurityIssue {
	issues := []SecurityIssue{}
	
	block, _ := pem.Decode(certData)
	if block == nil {
		issues = append(issues, SecurityIssue{
			Severity:    "Critical",
			Component:   fmt.Sprintf("Secret/%s", secretName),
			Description: "Invalid certificate format",
			Remediation: "Provide valid PEM-encoded certificate",
		})
		return issues
	}
	
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		issues = append(issues, SecurityIssue{
			Severity:    "Critical",
			Component:   fmt.Sprintf("Secret/%s", secretName),
			Description: "Failed to parse certificate",
			Remediation: "Provide valid X.509 certificate",
		})
		return issues
	}
	
	// Check expiration
	now := time.Now()
	if cert.NotAfter.Before(now) {
		issues = append(issues, SecurityIssue{
			Severity:    "Critical",
			Component:   fmt.Sprintf("Secret/%s", secretName),
			Description: "Certificate has expired",
			Remediation: "Renew certificate immediately",
		})
	} else if cert.NotAfter.Sub(now) < 30*24*time.Hour {
		issues = append(issues, SecurityIssue{
			Severity:    "High",
			Component:   fmt.Sprintf("Secret/%s", secretName),
			Description: fmt.Sprintf("Certificate expires in %d days", int(cert.NotAfter.Sub(now).Hours()/24)),
			Remediation: "Renew certificate before expiration",
		})
	}
	
	// Check key strength
	if cert.PublicKeyAlgorithm == x509.RSA {
		if rsaKey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			if rsaKey.N.BitLen() < 2048 {
				issues = append(issues, SecurityIssue{
					Severity:    "High",
					Component:   fmt.Sprintf("Secret/%s", secretName),
					Description: fmt.Sprintf("Weak RSA key size: %d bits", rsaKey.N.BitLen()),
					Remediation: "Use RSA keys of at least 2048 bits",
				})
			}
		}
	}
	
	// Check signature algorithm
	weakAlgos := []x509.SignatureAlgorithm{
		x509.SHA1WithRSA,
		x509.MD5WithRSA,
		x509.DSAWithSHA1,
		x509.ECDSAWithSHA1,
	}
	
	for _, weakAlgo := range weakAlgos {
		if cert.SignatureAlgorithm == weakAlgo {
			issues = append(issues, SecurityIssue{
				Severity:    "High",
				Component:   fmt.Sprintf("Secret/%s", secretName),
				Description: fmt.Sprintf("Weak signature algorithm: %s", cert.SignatureAlgorithm),
				Remediation: "Use SHA256 or stronger signature algorithm",
			})
			break
		}
	}
	
	return issues
}

// validatePrivateKey validates a private key
func (v *SecurityValidator) validatePrivateKey(keyData []byte, secretName string) []SecurityIssue {
	issues := []SecurityIssue{}
	
	block, _ := pem.Decode(keyData)
	if block == nil {
		issues = append(issues, SecurityIssue{
			Severity:    "Critical",
			Component:   fmt.Sprintf("Secret/%s", secretName),
			Description: "Invalid private key format",
			Remediation: "Provide valid PEM-encoded private key",
		})
		return issues
	}
	
	// Check if key is encrypted
	if !x509.IsEncryptedPEMBlock(block) {
		issues = append(issues, SecurityIssue{
			Severity:    "High",
			Component:   fmt.Sprintf("Secret/%s", secretName),
			Description: "Private key is not encrypted",
			Remediation: "Encrypt private keys when storing",
		})
	}
	
	return issues
}

// Helper functions

func (v *SecurityValidator) isPodCoveredByPolicy(pod corev1.Pod, policy networkingv1.NetworkPolicy) bool {
	// Check if pod matches the policy's pod selector
	for key, value := range policy.Spec.PodSelector.MatchLabels {
		if podValue, exists := pod.Labels[key]; !exists || podValue != value {
			return false
		}
	}
	return true
}

func (v *SecurityValidator) isLikelyPlainText(data []byte) bool {
	// Simple heuristic: if it's all printable ASCII, it might be plain text
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

func (v *SecurityValidator) hasWeakCiphers(ciphers string) bool {
	weakPatterns := []string{
		"RC4", "DES", "3DES", "MD5", "EXPORT", "NULL", "anon",
		"CBC", // Prefer GCM mode
	}
	
	ciphersLower := strings.ToLower(ciphers)
	for _, pattern := range weakPatterns {
		if strings.Contains(ciphersLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func (v *SecurityValidator) calculateSecurityScore(report *SecurityValidationReport) int {
	totalPossibleScore := 100
	deductions := 0
	
	// Deduct points based on issue severity
	for _, issue := range v.getAllIssues(report) {
		switch issue.Severity {
		case "Critical":
			deductions += 10
		case "High":
			deductions += 5
		case "Medium":
			deductions += 2
		case "Low":
			deductions += 1
		}
	}
	
	score := totalPossibleScore - deductions
	if score < 0 {
		score = 0
	}
	
	return score
}

func (v *SecurityValidator) getAllIssues(report *SecurityValidationReport) []SecurityIssue {
	allIssues := []SecurityIssue{}
	allIssues = append(allIssues, report.ContainerSecurityIssues...)
	allIssues = append(allIssues, report.RBACIssues...)
	allIssues = append(allIssues, report.NetworkPolicyIssues...)
	allIssues = append(allIssues, report.SecretManagementIssues...)
	allIssues = append(allIssues, report.TLSConfigurationIssues...)
	return allIssues
}

func (v *SecurityValidator) getTotalIssueCount(report *SecurityValidationReport) int {
	return len(v.getAllIssues(report))
}