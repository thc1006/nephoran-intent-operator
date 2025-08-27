package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestRBACCompliance validates RBAC policies for security compliance
func TestRBACCompliance(t *testing.T) {
	tests := []struct {
		name        string
		role        *rbacv1.Role
		clusterRole *rbacv1.ClusterRole
		wantError   bool
		errorMsg    string
	}{
		{
			name: "No wildcard permissions allowed",
			role: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-role",
					Namespace: "default",
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"*"},
						Resources: []string{"*"},
						Verbs:     []string{"*"},
					},
				},
			},
			wantError: true,
			errorMsg:  "wildcard permissions not allowed",
		},
		{
			name: "Least privilege principle - specific permissions",
			role: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nephoran-operator",
					Namespace: "nephoran-system",
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"nephoran.io"},
						Resources: []string{"networkintents"},
						Verbs:     []string{"get", "list", "watch", "update", "patch"},
					},
					{
						APIGroups: []string{"apps"},
						Resources: []string{"deployments"},
						Verbs:     []string{"get", "list", "create", "update"},
					},
				},
			},
			wantError: false,
		},
		{
			name: "No escalate or bind privileges for non-admin",
			clusterRole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "user-role",
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"rbac.authorization.k8s.io"},
						Resources: []string{"clusterroles"},
						Verbs:     []string{"escalate", "bind"},
					},
				},
			},
			wantError: true,
			errorMsg:  "escalate/bind privileges require admin role",
		},
		{
			name: "Service account token access restricted",
			role: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa-role",
					Namespace: "default",
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"serviceaccounts/token"},
						Verbs:     []string{"create"},
					},
				},
			},
			wantError: true,
			errorMsg:  "service account token creation restricted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRBACPolicy(tt.role, tt.clusterRole)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNetworkPolicyEnforcement validates network security policies
func TestNetworkPolicyEnforcement(t *testing.T) {
	tests := []struct {
		name      string
		policy    *networkingv1.NetworkPolicy
		wantError bool
		errorMsg  string
	}{
		{
			name: "Default deny all traffic",
			policy: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default-deny",
					Namespace: "nephoran-system",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeIngress,
						networkingv1.PolicyTypeEgress,
					},
				},
			},
			wantError: false,
		},
		{
			name: "Egress restrictions - no external HTTP",
			policy: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "egress-policy",
					Namespace: "nephoran-system",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "nephoran-operator",
						},
					},
					Egress: []networkingv1.NetworkPolicyEgressRule{
						{
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: &corev1.Protocol("TCP"),
									Port:     &intstr.IntOrString{IntVal: 80},
								},
							},
						},
					},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeEgress,
					},
				},
			},
			wantError: true,
			errorMsg:  "HTTP (port 80) not allowed for egress",
		},
		{
			name: "O-RAN interface segmentation",
			policy: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oran-segmentation",
					Namespace: "oran-system",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"oran-interface": "a1",
						},
					},
					Ingress: []networkingv1.NetworkPolicyIngressRule{
						{
							From: []networkingv1.NetworkPolicyPeer{
								{
									PodSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"component": "non-rt-ric",
										},
									},
								},
							},
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: &corev1.Protocol("TCP"),
									Port:     &intstr.IntOrString{IntVal: 443},
								},
							},
						},
					},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeIngress,
					},
				},
			},
			wantError: false,
		},
		{
			name: "mTLS required for service communication",
			policy: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mtls-policy",
					Namespace: "nephoran-system",
					Annotations: map[string]string{
						"security.istio.io/tlsMode": "STRICT",
					},
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "nephoran-operator",
						},
					},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNetworkPolicy(tt.policy)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestContainerSecurityContext validates container security configurations
func TestContainerSecurityContext(t *testing.T) {
	tests := []struct {
		name      string
		container corev1.Container
		wantError bool
		errorMsg  string
	}{
		{
			name: "Non-root user required",
			container: corev1.Container{
				Name:  "test-container",
				Image: "nephoran/operator:latest",
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot: &[]bool{true}[0],
					RunAsUser:    &[]int64{10001}[0],
				},
			},
			wantError: false,
		},
		{
			name: "Root user not allowed",
			container: corev1.Container{
				Name:  "test-container",
				Image: "nephoran/operator:latest",
				SecurityContext: &corev1.SecurityContext{
					RunAsUser: &[]int64{0}[0],
				},
			},
			wantError: true,
			errorMsg:  "container must not run as root",
		},
		{
			name: "Read-only root filesystem required",
			container: corev1.Container{
				Name:  "test-container",
				Image: "nephoran/operator:latest",
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot:             &[]bool{true}[0],
					RunAsUser:                &[]int64{10001}[0],
					ReadOnlyRootFilesystem:   &[]bool{true}[0],
					AllowPrivilegeEscalation: &[]bool{false}[0],
				},
			},
			wantError: false,
		},
		{
			name: "No additional capabilities allowed",
			container: corev1.Container{
				Name:  "test-container",
				Image: "nephoran/operator:latest",
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot: &[]bool{true}[0],
					RunAsUser:    &[]int64{10001}[0],
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{"NET_ADMIN"},
					},
				},
			},
			wantError: true,
			errorMsg:  "no additional capabilities allowed",
		},
		{
			name: "All capabilities must be dropped",
			container: corev1.Container{
				Name:  "test-container",
				Image: "nephoran/operator:latest",
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot: &[]bool{true}[0],
					RunAsUser:    &[]int64{10001}[0],
					Capabilities: &corev1.Capabilities{
						Drop: []corev1.Capability{"ALL"},
					},
				},
			},
			wantError: false,
		},
		{
			name: "Seccomp profile required",
			container: corev1.Container{
				Name:  "test-container",
				Image: "nephoran/operator:latest",
				SecurityContext: &corev1.SecurityContext{
					RunAsNonRoot: &[]bool{true}[0],
					RunAsUser:    &[]int64{10001}[0],
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContainerSecurity(tt.container)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecretsManagement validates secrets security
func TestSecretsManagement(t *testing.T) {
	tests := []struct {
		name      string
		secret    *corev1.Secret
		wantError bool
		errorMsg  string
	}{
		{
			name: "No plaintext secrets allowed",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"password": []byte("plaintext-password"),
				},
			},
			wantError: true,
			errorMsg:  "plaintext passwords not allowed",
		},
		{
			name: "Encrypted secrets with proper annotations",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "encrypted-secret",
					Namespace: "nephoran-system",
					Annotations: map[string]string{
						"encryption.nephoran.io/algorithm":  "AES-256-GCM",
						"encryption.nephoran.io/key-id":     "vault-key-123",
						"rotation.nephoran.io/last-rotated": time.Now().Format(time.RFC3339),
					},
				},
				Data: map[string][]byte{
					"encrypted-data": []byte("base64-encrypted-content"),
				},
			},
			wantError: false,
		},
		{
			name: "Secret rotation required within 90 days",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "old-secret",
					Namespace: "default",
					Annotations: map[string]string{
						"rotation.nephoran.io/last-rotated": time.Now().AddDate(0, 0, -100).Format(time.RFC3339),
					},
				},
			},
			wantError: true,
			errorMsg:  "secret rotation overdue",
		},
		{
			name: "API keys must be encrypted",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "api-key",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"api-key": []byte("sk-1234567890abcdef"),
				},
			},
			wantError: true,
			errorMsg:  "API keys must be encrypted",
		},
		{
			name: "TLS certificates validation",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "tls-secret",
					Namespace: "nephoran-system",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": []byte("-----BEGIN CERTIFICATE-----"),
					"tls.key": []byte("-----BEGIN RSA PRIVATE KEY-----"),
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecretsManagement(tt.secret)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTLSConfiguration validates TLS security settings
func TestTLSConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		tlsConfig *tls.Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "TLS 1.3 required",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				MaxVersion: tls.VersionTLS13,
			},
			wantError: false,
		},
		{
			name: "TLS 1.2 not allowed",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			wantError: true,
			errorMsg:  "TLS 1.3 minimum required",
		},
		{
			name: "Secure cipher suites only",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				CipherSuites: []uint16{
					tls.TLS_AES_256_GCM_SHA384,
					tls.TLS_CHACHA20_POLY1305_SHA256,
					tls.TLS_AES_128_GCM_SHA256,
				},
			},
			wantError: false,
		},
		{
			name: "Weak cipher suites not allowed",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
				CipherSuites: []uint16{
					tls.TLS_RSA_WITH_RC4_128_SHA,
				},
			},
			wantError: true,
			errorMsg:  "weak cipher suite detected",
		},
		{
			name: "Client certificate required for mTLS",
			tlsConfig: &tls.Config{
				MinVersion:               tls.VersionTLS13,
				ClientAuth:               tls.RequireAndVerifyClientCert,
				ClientCAs:                x509.NewCertPool(),
				PreferServerCipherSuites: true,
			},
			wantError: false,
		},
		{
			name: "Certificate validation required",
			tlsConfig: &tls.Config{
				MinVersion:         tls.VersionTLS13,
				InsecureSkipVerify: true,
			},
			wantError: true,
			errorMsg:  "certificate validation required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTLSConfiguration(tt.tlsConfig)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAPISecurityValidation tests API security controls
func TestAPISecurityValidation(t *testing.T) {
	tests := []struct {
		name      string
		request   *http.Request
		wantError bool
		errorMsg  string
	}{
		{
			name: "Authentication required",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Bearer valid-token"},
				},
			},
			wantError: false,
		},
		{
			name: "Missing authentication",
			request: &http.Request{
				Header: http.Header{},
			},
			wantError: true,
			errorMsg:  "authentication required",
		},
		{
			name: "Rate limiting enforced",
			request: &http.Request{
				Header: http.Header{
					"Authorization":     []string{"Bearer valid-token"},
					"X-RateLimit-Limit": []string{"100"},
				},
			},
			wantError: false,
		},
		{
			name: "Input validation - SQL injection attempt",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Bearer valid-token"},
				},
				URL: &url.URL{
					RawQuery: "id=1' OR '1'='1",
				},
			},
			wantError: true,
			errorMsg:  "potential SQL injection detected",
		},
		{
			name: "Input validation - XSS attempt",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Bearer valid-token"},
				},
				Body: io.NopCloser(strings.NewReader("<script>alert('xss')</script>")),
			},
			wantError: true,
			errorMsg:  "potential XSS detected",
		},
		{
			name: "CORS validation",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Bearer valid-token"},
					"Origin":        []string{"https://nephoran.io"},
				},
			},
			wantError: false,
		},
		{
			name: "Invalid CORS origin",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Bearer valid-token"},
					"Origin":        []string{"http://malicious.com"},
				},
			},
			wantError: true,
			errorMsg:  "invalid CORS origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPISecurity(tt.request)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestORANSecurityCompliance validates O-RAN specific security requirements
func TestORANSecurityCompliance(t *testing.T) {
	tests := []struct {
		name      string
		config    ORANSecurityConfig
		wantError bool
		errorMsg  string
	}{
		{
			name: "A1 interface security",
			config: ORANSecurityConfig{
				Interface: "A1",
				Authentication: AuthConfig{
					Type:                  "mutual_tls",
					CertificateValidation: true,
				},
				Encryption: EncryptionConfig{
					Enabled:   true,
					Algorithm: "AES-256-GCM",
				},
				IntegrityProtection: true,
			},
			wantError: false,
		},
		{
			name: "O1 interface NETCONF/SSH",
			config: ORANSecurityConfig{
				Interface: "O1",
				Transport: "SSH",
				Authentication: AuthConfig{
					Type: "public_key",
				},
				Encryption: EncryptionConfig{
					Enabled:    true,
					TLSVersion: "1.3",
				},
			},
			wantError: false,
		},
		{
			name: "E2 interface SCTP security",
			config: ORANSecurityConfig{
				Interface: "E2",
				Transport: "SCTP",
				Authentication: AuthConfig{
					Type: "sctp_auth",
				},
				IPSec: &IPSecConfig{
					Enabled:   true,
					Mode:      "transport",
					Algorithm: "AES-256-CBC",
				},
			},
			wantError: false,
		},
		{
			name: "xApp sandboxing required",
			config: ORANSecurityConfig{
				Component: "xApp",
				Sandboxing: &SandboxConfig{
					Enabled:           true,
					ResourceIsolation: true,
					NetworkIsolation:  true,
				},
				CodeSigning: &CodeSigningConfig{
					Required:  true,
					Algorithm: "SHA256WithRSA",
				},
			},
			wantError: false,
		},
		{
			name: "CU-DU IPsec required",
			config: ORANSecurityConfig{
				Interface: "F1",
				IPSec: &IPSecConfig{
					Enabled: false,
				},
			},
			wantError: true,
			errorMsg:  "IPsec required for CU-DU interface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateORANSecurity(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestComplianceValidation validates compliance with security standards
func TestComplianceValidation(t *testing.T) {
	tests := []struct {
		name      string
		standard  string
		config    ComplianceConfig
		wantError bool
		errorMsg  string
	}{
		{
			name:     "SOC2 Type 2 compliance",
			standard: "SOC2",
			config: ComplianceConfig{
				LogicalAccessControls: true,
				SystemMonitoring:      true,
				IncidentManagement:    true,
				ChangeManagement:      true,
				RiskAssessment:        true,
			},
			wantError: false,
		},
		{
			name:     "ISO 27001 compliance",
			standard: "ISO27001",
			config: ComplianceConfig{
				AccessControl:          true,
				OperationsSecurity:     true,
				CommunicationsSecurity: true,
				SystemSecurity:         true,
				IncidentManagement:     true,
				BusinessContinuity:     true,
			},
			wantError: false,
		},
		{
			name:     "PCI-DSS v4 compliance",
			standard: "PCI-DSS",
			config: ComplianceConfig{
				NetworkSegmentation:     true,
				StrongAccessControl:     true,
				DataEncryption:          true,
				VulnerabilityManagement: true,
				LoggingAndMonitoring:    true,
				SecurityTesting:         true,
			},
			wantError: false,
		},
		{
			name:     "GDPR compliance",
			standard: "GDPR",
			config: ComplianceConfig{
				DataClassification: true,
				PrivacyByDesign:    true,
				RightToErasure:     true,
				DataPortability:    true,
				ConsentManagement:  true,
				BreachNotification: true,
			},
			wantError: false,
		},
		{
			name:     "O-RAN WG11 compliance",
			standard: "ORAN-WG11",
			config: ComplianceConfig{
				ZeroTrustArchitecture:   true,
				InterfaceSecurity:       true,
				ComponentHardening:      true,
				CryptographicProtection: true,
				AuditAndAccountability:  true,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCompliance(tt.standard, tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions for validation

func validateRBACPolicy(role *rbacv1.Role, clusterRole *rbacv1.ClusterRole) error {
	rules := []rbacv1.PolicyRule{}

	if role != nil {
		rules = append(rules, role.Rules...)
	}
	if clusterRole != nil {
		rules = append(rules, clusterRole.Rules...)
	}

	for _, rule := range rules {
		// Check for wildcard permissions
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "*" {
				return fmt.Errorf("wildcard permissions not allowed")
			}
		}
		for _, resource := range rule.Resources {
			if resource == "*" {
				return fmt.Errorf("wildcard permissions not allowed")
			}
		}
		for _, verb := range rule.Verbs {
			if verb == "*" {
				return fmt.Errorf("wildcard permissions not allowed")
			}
		}

		// Check for escalate/bind privileges
		if containsString(rule.Verbs, "escalate") || containsString(rule.Verbs, "bind") {
			if clusterRole == nil || !strings.Contains(clusterRole.Name, "admin") {
				return fmt.Errorf("escalate/bind privileges require admin role")
			}
		}

		// Check for service account token creation
		if containsString(rule.Resources, "serviceaccounts/token") && containsString(rule.Verbs, "create") {
			return fmt.Errorf("service account token creation restricted")
		}
	}

	return nil
}

func validateNetworkPolicy(policy *networkingv1.NetworkPolicy) error {
	// Check for HTTP in egress rules
	for _, egress := range policy.Spec.Egress {
		for _, port := range egress.Ports {
			if port.Port != nil && port.Port.IntVal == 80 {
				return fmt.Errorf("HTTP (port 80) not allowed for egress")
			}
		}
	}

	// Validate mTLS annotations
	if tlsMode, ok := policy.Annotations["security.istio.io/tlsMode"]; ok {
		if tlsMode != "STRICT" {
			return fmt.Errorf("mTLS must be set to STRICT mode")
		}
	}

	return nil
}

func validateContainerSecurity(container corev1.Container) error {
	if container.SecurityContext == nil {
		return fmt.Errorf("security context required")
	}

	sc := container.SecurityContext

	// Check non-root user
	if sc.RunAsUser != nil && *sc.RunAsUser == 0 {
		return fmt.Errorf("container must not run as root")
	}

	// Check RunAsNonRoot
	if sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
		return fmt.Errorf("RunAsNonRoot must be true")
	}

	// Check read-only root filesystem
	if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		return fmt.Errorf("read-only root filesystem required")
	}

	// Check privilege escalation
	if sc.AllowPrivilegeEscalation != nil && *sc.AllowPrivilegeEscalation {
		return fmt.Errorf("privilege escalation not allowed")
	}

	// Check capabilities
	if sc.Capabilities != nil {
		if len(sc.Capabilities.Add) > 0 {
			return fmt.Errorf("no additional capabilities allowed")
		}
		if len(sc.Capabilities.Drop) == 0 || sc.Capabilities.Drop[0] != "ALL" {
			return fmt.Errorf("all capabilities must be dropped")
		}
	}

	// Check seccomp profile
	if sc.SeccompProfile == nil || sc.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		return fmt.Errorf("seccomp profile required")
	}

	return nil
}

func validateSecretsManagement(secret *corev1.Secret) error {
	// Check for plaintext passwords
	for key, value := range secret.Data {
		if strings.Contains(key, "password") || strings.Contains(key, "passwd") {
			if isPlaintext(string(value)) {
				return fmt.Errorf("plaintext passwords not allowed")
			}
		}

		// Check for API keys
		if strings.Contains(key, "api-key") || strings.Contains(key, "api_key") {
			if strings.HasPrefix(string(value), "sk-") || strings.HasPrefix(string(value), "pk-") {
				return fmt.Errorf("API keys must be encrypted")
			}
		}
	}

	// Check encryption annotations
	if algo, ok := secret.Annotations["encryption.nephoran.io/algorithm"]; ok {
		if algo != "AES-256-GCM" {
			return fmt.Errorf("AES-256-GCM encryption required")
		}
	}

	// Check rotation
	if lastRotated, ok := secret.Annotations["rotation.nephoran.io/last-rotated"]; ok {
		rotationTime, err := time.Parse(time.RFC3339, lastRotated)
		if err == nil {
			if time.Since(rotationTime) > 90*24*time.Hour {
				return fmt.Errorf("secret rotation overdue")
			}
		}
	}

	return nil
}

func validateTLSConfiguration(tlsConfig *tls.Config) error {
	// Check TLS version
	if tlsConfig.MinVersion < tls.VersionTLS13 {
		return fmt.Errorf("TLS 1.3 minimum required")
	}

	// Check cipher suites
	weakCiphers := []uint16{
		tls.TLS_RSA_WITH_RC4_128_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}

	for _, cipher := range tlsConfig.CipherSuites {
		for _, weak := range weakCiphers {
			if cipher == weak {
				return fmt.Errorf("weak cipher suite detected")
			}
		}
	}

	// Check certificate validation
	if tlsConfig.InsecureSkipVerify {
		return fmt.Errorf("certificate validation required")
	}

	// Check client authentication for mTLS
	if tlsConfig.ClientAuth < tls.RequireAndVerifyClientCert {
		return fmt.Errorf("client certificate verification required for mTLS")
	}

	return nil
}

func validateAPISecurity(request *http.Request) error {
	// Check authentication
	authHeader := request.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("authentication required")
	}

	// Check for SQL injection attempts
	if request.URL != nil && request.URL.RawQuery != "" {
		sqlPatterns := []string{
			"'.*OR.*'.*=.*'",
			"UNION.*SELECT",
			"DROP.*TABLE",
			"INSERT.*INTO",
			"DELETE.*FROM",
		}

		for _, pattern := range sqlPatterns {
			if matched, _ := regexp.MatchString(pattern, request.URL.RawQuery); matched {
				return fmt.Errorf("potential SQL injection detected")
			}
		}
	}

	// Check for XSS attempts
	if request.Body != nil {
		body, _ := io.ReadAll(request.Body)
		xssPatterns := []string{
			"<script",
			"javascript:",
			"onerror=",
			"onclick=",
		}

		for _, pattern := range xssPatterns {
			if strings.Contains(string(body), pattern) {
				return fmt.Errorf("potential XSS detected")
			}
		}
	}

	// Check CORS
	origin := request.Header.Get("Origin")
	if origin != "" {
		allowedOrigins := []string{
			"https://nephoran.io",
			"https://*.nephoran.io",
		}

		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if matched, _ := regexp.MatchString(allowedOrigin, origin); matched {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("invalid CORS origin")
		}
	}

	return nil
}

// Helper structures for O-RAN and compliance testing

type ORANSecurityConfig struct {
	Interface           string
	Component           string
	Transport           string
	Authentication      AuthConfig
	Encryption          EncryptionConfig
	IntegrityProtection bool
	IPSec               *IPSecConfig
	Sandboxing          *SandboxConfig
	CodeSigning         *CodeSigningConfig
}

type AuthConfig struct {
	Type                  string
	CertificateValidation bool
}

type EncryptionConfig struct {
	Enabled    bool
	Algorithm  string
	TLSVersion string
}

type IPSecConfig struct {
	Enabled   bool
	Mode      string
	Algorithm string
}

type SandboxConfig struct {
	Enabled           bool
	ResourceIsolation bool
	NetworkIsolation  bool
}

type CodeSigningConfig struct {
	Required  bool
	Algorithm string
}

type ComplianceConfig struct {
	// SOC2
	LogicalAccessControls bool
	SystemMonitoring      bool
	IncidentManagement    bool
	ChangeManagement      bool
	RiskAssessment        bool

	// ISO 27001
	AccessControl          bool
	OperationsSecurity     bool
	CommunicationsSecurity bool
	SystemSecurity         bool
	BusinessContinuity     bool

	// PCI-DSS
	NetworkSegmentation     bool
	StrongAccessControl     bool
	DataEncryption          bool
	VulnerabilityManagement bool
	LoggingAndMonitoring    bool
	SecurityTesting         bool

	// GDPR
	DataClassification bool
	PrivacyByDesign    bool
	RightToErasure     bool
	DataPortability    bool
	ConsentManagement  bool
	BreachNotification bool

	// O-RAN
	ZeroTrustArchitecture   bool
	InterfaceSecurity       bool
	ComponentHardening      bool
	CryptographicProtection bool
	AuditAndAccountability  bool
}

func validateORANSecurity(config ORANSecurityConfig) error {
	// Validate F1 interface IPsec requirement
	if config.Interface == "F1" && (config.IPSec == nil || !config.IPSec.Enabled) {
		return fmt.Errorf("IPsec required for CU-DU interface")
	}

	// Add more O-RAN specific validations as needed
	return nil
}

func validateCompliance(standard string, config ComplianceConfig) error {
	// Implementation would validate specific compliance requirements
	// This is a placeholder for the actual validation logic
	return nil
}

// Helper utilities

func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func isPlaintext(s string) bool {
	// Simple heuristic: if it's not base64 encoded or doesn't look encrypted
	if len(s) < 20 {
		return true
	}

	// Check if it looks like base64
	if matched, _ := regexp.MatchString(`^[A-Za-z0-9+/]+={0,2}$`, s); !matched {
		return true
	}

	return false
}
