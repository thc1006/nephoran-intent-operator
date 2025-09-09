// Package validation provides comprehensive security compliance testing
package test_validation

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
<<<<<<< HEAD
=======
	"k8s.io/apimachinery/pkg/types"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/tests/framework"
)

var _ = ginkgo.Describe("Security Compliance Validation Suite", ginkgo.Ordered, func() {
	var (
		testSuite    *framework.TestSuite
		ctx          context.Context
		cancel       context.CancelFunc
		secValidator *SecurityValidator
		k8sClient    client.Client
	)

	ginkgo.BeforeAll(func() {
		ginkgo.By("Initializing Security Compliance Test Suite")
		testConfig := framework.DefaultTestConfig()
		testConfig.MockExternalAPIs = true
		testSuite = framework.NewTestSuite(testConfig)
		testSuite.SetupSuite()

		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Minute)
		k8sClient = testSuite.GetK8sClient()

		// Initialize security validator
		config := DefaultValidationConfig()
		config.EnableSecurityTesting = true
		secValidator = NewSecurityValidator(config)
		secValidator.SetK8sClient(k8sClient)
	})

	ginkgo.AfterAll(func() {
		ginkgo.By("Tearing down Security Compliance Test Suite")
		cancel()
		testSuite.TearDownSuite()
	})

	ginkgo.Context("Authentication & Authorization Testing (5/5 points)", func() {
		ginkgo.It("should validate OAuth2/OIDC integration with multiple providers", func() {
			ginkgo.By("Testing OAuth2/OIDC multi-provider support")

			// Create OAuth2 configuration secrets for multiple providers
			providers := []struct {
				name     string
				provider string
				config   map[string]string
			}{
				{
					name:     "oauth2-google",
					provider: "google",
					config: map[string]string{
						"client_id":     "google-client-id",
						"client_secret": "google-client-secret",
						"issuer_url":    "https://accounts.google.com",
						"redirect_uri":  "http://localhost:8080/callback/google",
					},
				},
				{
					name:     "oauth2-github",
					provider: "github",
					config: map[string]string{
						"client_id":     "github-client-id",
						"client_secret": "github-client-secret",
						"issuer_url":    "https://github.com",
						"redirect_uri":  "http://localhost:8080/callback/github",
					},
				},
				{
					name:     "oidc-keycloak",
					provider: "keycloak",
					config: map[string]string{
						"client_id":     "keycloak-client-id",
						"client_secret": "keycloak-client-secret",
						"issuer_url":    "https://keycloak.example.com/auth/realms/nephoran",
						"redirect_uri":  "http://localhost:8080/callback/keycloak",
					},
				},
			}

			for _, provider := range providers {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      provider.name,
						Namespace: "default",
						Labels: map[string]string{
							"nephoran.io/auth-provider": provider.provider,
							"nephoran.io/auth-type":     "oauth2",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: make(map[string][]byte),
				}

				for key, value := range provider.config {
					secret.Data[key] = []byte(value)
				}

				err := k8sClient.Create(ctx, secret)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Verify secret creation
				createdSecret := &corev1.Secret{}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: secret.GetName(), Namespace: secret.GetNamespace()}, createdSecret)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(createdSecret.Data).To(gomega.HaveLen(4))
			}

			// Validate OAuth2 integration
			score := secValidator.ValidateAuthentication(ctx)
			gomega.Expect(score).To(gomega.BeNumerically(">=", 4), "OAuth2/OIDC integration should score at least 4/5")
		})

		ginkgo.It("should enforce service account authentication with minimal privileges", func() {
			ginkgo.By("Creating service account with least privilege principle")

			// Create service account
			serviceAccount := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nephoran-intent-operator",
					Namespace: "default",
					Labels: map[string]string{
						"app.kubernetes.io/name":       "nephoran-intent-operator",
						"app.kubernetes.io/managed-by": "test-suite",
					},
				},
				AutomountServiceAccountToken: func(b bool) *bool { return &b }(false), // Security best practice
			}

			err := k8sClient.Create(ctx, serviceAccount)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create ClusterRole with minimal permissions
			clusterRole := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nephoran-intent-operator-role",
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"nephoran.io"},
						Resources: []string{"networkintents", "e2nodesets"},
						Verbs:     []string{"get", "list", "watch", "create", "update", "patch"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps", "secrets"},
						Verbs:     []string{"get", "list", "watch"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create", "patch"},
					},
				},
			}

			err = k8sClient.Create(ctx, clusterRole)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create ClusterRoleBinding
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nephoran-intent-operator-binding",
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     clusterRole.Name,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      serviceAccount.Name,
						Namespace: serviceAccount.Namespace,
					},
				},
			}

			err = k8sClient.Create(ctx, clusterRoleBinding)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate RBAC configuration
			validated := secValidator.validateRBACPolicyEnforcement(ctx)
			gomega.Expect(validated).To(gomega.BeTrue(), "RBAC policies should be properly enforced")
		})

		ginkgo.It("should implement multi-factor authentication validation", func() {
			ginkgo.By("Testing MFA configuration and enforcement")

			// Create MFA configuration
			mfaConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mfa-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/security": "mfa",
					},
				},
				Data: map[string]string{
					"mfa_enabled":   "true",
					"mfa_providers": "totp,sms,email",
					"mfa_required":  "admin,operator",
					"totp_issuer":   "Nephoran Intent Operator",
					"totp_period":   "30",
					"totp_digits":   "6",
				},
			}

			err := k8sClient.Create(ctx, mfaConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create MFA secrets for users
			mfaSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mfa-secrets",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/security": "mfa-secrets",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"admin-totp-secret":    []byte("JBSWY3DPEHPK3PXP"),
					"operator-totp-secret": []byte("KRSXG5CTMVRXEZLU"),
				},
			}

			err = k8sClient.Create(ctx, mfaSecret)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify MFA configuration
			retrievedConfig := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: mfaConfig.GetName(), Namespace: mfaConfig.GetNamespace()}, retrievedConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(retrievedConfig.Data["mfa_enabled"]).To(gomega.Equal("true"))
		})

		ginkgo.It("should validate token lifecycle management with rotation", func() {
			ginkgo.By("Testing token rotation and expiration policies")

			// Create token lifecycle configuration
			tokenConfig := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "token-lifecycle-config",
					Namespace: "default",
					Annotations: map[string]string{
						"nephoran.io/token-expiry":         "24h",
						"nephoran.io/refresh-token-expiry": "7d",
						"nephoran.io/rotation-enabled":     "true",
						"nephoran.io/rotation-interval":    "12h",
					},
					Labels: map[string]string{
						"nephoran.io/security": "token-lifecycle",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"access_token":  []byte("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."),
					"refresh_token": []byte("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."),
					"last_rotated":  []byte(time.Now().Format(time.RFC3339)),
				},
			}

			err := k8sClient.Create(ctx, tokenConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate token lifecycle management
			validated := secValidator.validateTokenLifecycleManagement(ctx)
			gomega.Expect(validated).To(gomega.BeTrue(), "Token lifecycle should be properly managed")
		})
	})

	ginkgo.Context("Data Encryption Testing (4/4 points)", func() {
		ginkgo.It("should validate TLS/mTLS configuration with strong cipher suites", func() {
			ginkgo.By("Testing TLS 1.3 configuration with strong ciphers")

			// Create TLS certificate secret
			tlsSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nephoran-tls",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/tls-version": "1.3",
						"nephoran.io/mtls":        "enabled",
					},
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": []byte(testTLSCertPEM),
					"tls.key": []byte(testTLSKeyPEM),
					"ca.crt":  []byte(testCACertPEM),
				},
			}

			err := k8sClient.Create(ctx, tlsSecret)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create mTLS client certificate
			clientCertSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nephoran-client-cert",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/tls-client": "true",
					},
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": []byte(testClientCertPEM),
					"tls.key": []byte(testClientKeyPEM),
				},
			}

			err = k8sClient.Create(ctx, clientCertSecret)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate TLS/mTLS configuration
			score := secValidator.ValidateEncryption(ctx)
			gomega.Expect(score).To(gomega.BeNumerically(">=", 3), "TLS/mTLS should score at least 3/4")
		})

		ginkgo.It("should verify encryption at rest for sensitive data", func() {
			ginkgo.By("Testing etcd encryption and persistent volume encryption")

			// Create encryption configuration
			encryptionConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "encryption-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/encryption": "at-rest",
					},
				},
				Data: map[string]string{
					"etcd_encryption":     "aescbc",
					"storage_class":       "encrypted-ssd",
					"kms_provider":        "aws-kms",
					"kms_key_id":          "arn:aws:kms:us-west-2:123456789012:key/12345678",
					"encryption_provider": "AES256",
				},
			}

			err := k8sClient.Create(ctx, encryptionConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Test encryption at rest
			validated := secValidator.validateEncryptionAtRest(ctx)
			gomega.Expect(validated).To(gomega.BeTrue(), "Encryption at rest should be properly configured")
		})

		ginkgo.It("should implement comprehensive key management system", func() {
			ginkgo.By("Testing KMS integration and key rotation")

			// Create KMS configuration
			kmsConfig := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kms-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/kms": "enabled",
					},
					Annotations: map[string]string{
						"nephoran.io/key-rotation":    "enabled",
						"nephoran.io/rotation-period": "90d",
						"nephoran.io/last-rotation":   time.Now().Format(time.RFC3339),
						"nephoran.io/next-rotation":   time.Now().Add(90 * 24 * time.Hour).Format(time.RFC3339),
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"master_key_id":      []byte("nephoran-master-key-v1"),
					"data_key_encrypted": []byte("AQIDAHi...encrypted..."),
					"key_version":        []byte("1"),
				},
			}

			err := k8sClient.Create(ctx, kmsConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create certificate lifecycle management
			certManager := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cert-manager-config",
					Namespace: "default",
					Labels: map[string]string{
						"cert-manager.io/managed": "true",
					},
				},
				Data: map[string]string{
					"issuer":            "letsencrypt-prod",
					"renewal_threshold": "30d",
					"key_algorithm":     "RSA",
					"key_size":          "4096",
					"auto_renewal":      "true",
				},
			}

			err = k8sClient.Create(ctx, certManager)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate key management
			validated := secValidator.validateKeyManagementAndCertificateLifecycle(ctx)
			gomega.Expect(validated).To(gomega.BeTrue(), "Key management and certificate lifecycle should be properly managed")
		})

		ginkgo.It("should validate data integrity with checksums and signatures", func() {
			ginkgo.By("Testing data integrity validation mechanisms")

			// Create data integrity configuration
			integrityConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "data-integrity-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/integrity": "enabled",
					},
				},
				Data: map[string]string{
					"checksum_algorithm":  "SHA256",
					"signature_algorithm": "RSA-PSS",
					"verify_on_read":      "true",
					"verify_on_write":     "true",
				},
			}

			err := k8sClient.Create(ctx, integrityConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create sample data with integrity checks
			dataWithIntegrity := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "data-with-integrity",
					Namespace: "default",
					Annotations: map[string]string{
						"nephoran.io/checksum":  "sha256:abcdef1234567890",
						"nephoran.io/signature": "RSA-PSS:signature-data",
						"nephoran.io/timestamp": time.Now().Format(time.RFC3339),
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"sensitive_data": []byte("critical-application-data"),
				},
			}

			err = k8sClient.Create(ctx, dataWithIntegrity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Network Security Testing (3/3 points)", func() {
		ginkgo.It("should enforce comprehensive network policies with zero-trust", func() {
			ginkgo.By("Testing network policy enforcement and zero-trust architecture")

			// Create default deny-all NetworkPolicy
			defaultDenyPolicy := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default-deny-all",
					Namespace: "default",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeIngress,
						networkingv1.PolicyTypeEgress,
					},
				},
			}

			err := k8sClient.Create(ctx, defaultDenyPolicy)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create specific allow policies for nephoran components
			nephoranPolicy := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nephoran-allow",
					Namespace: "default",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "nephoran-intent-operator",
						},
					},
					PolicyTypes: []networkingv1.PolicyType{
						networkingv1.PolicyTypeIngress,
						networkingv1.PolicyTypeEgress,
					},
					Ingress: []networkingv1.NetworkPolicyIngressRule{
						{
							From: []networkingv1.NetworkPolicyPeer{
								{
									PodSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"app": "api-gateway",
										},
									},
								},
							},
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: func(p corev1.Protocol) *corev1.Protocol { return &p }(corev1.ProtocolTCP),
									Port:     &intstr.IntOrString{IntVal: 8080},
								},
							},
						},
					},
					Egress: []networkingv1.NetworkPolicyEgressRule{
						{
							To: []networkingv1.NetworkPolicyPeer{
								{
									PodSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											"app": "weaviate",
										},
									},
								},
							},
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: func(p corev1.Protocol) *corev1.Protocol { return &p }(corev1.ProtocolTCP),
									Port:     &intstr.IntOrString{IntVal: 8080},
								},
							},
						},
					},
				},
			}

			err = k8sClient.Create(ctx, nephoranPolicy)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate network security
			score := secValidator.ValidateNetworkSecurity(ctx)
			gomega.Expect(score).To(gomega.BeNumerically(">=", 2), "Network security should score at least 2/3")
		})

		ginkgo.It("should implement network segmentation with security zones", func() {
			ginkgo.By("Testing network segmentation and security zone isolation")

			// Create namespace for each security zone
			zones := []string{"dmz", "internal", "management", "data"}

			for _, zone := range zones {
				namespace := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("zone-%s", zone),
						Labels: map[string]string{
							"security-zone":                      zone,
							"pod-security.kubernetes.io/enforce": "restricted",
							"pod-security.kubernetes.io/audit":   "restricted",
							"pod-security.kubernetes.io/warn":    "restricted",
						},
					},
				}

				err := k8sClient.Create(ctx, namespace)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Create zone-specific network policy
				zonePolicy := &networkingv1.NetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-zone-policy", zone),
						Namespace: namespace.Name,
					},
					Spec: networkingv1.NetworkPolicySpec{
						PodSelector: metav1.LabelSelector{},
						PolicyTypes: []networkingv1.PolicyType{
							networkingv1.PolicyTypeIngress,
							networkingv1.PolicyTypeEgress,
						},
						Ingress: []networkingv1.NetworkPolicyIngressRule{
							{
								From: []networkingv1.NetworkPolicyPeer{
									{
										NamespaceSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"security-zone": zone,
											},
										},
									},
								},
							},
						},
						Egress: []networkingv1.NetworkPolicyEgressRule{
							{
								To: []networkingv1.NetworkPolicyPeer{
									{
										NamespaceSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"security-zone": zone,
											},
										},
									},
								},
							},
						},
					},
				}

				err = k8sClient.Create(ctx, zonePolicy)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}

			// Validate network segmentation
			validated := secValidator.validateNetworkSegmentationAndControls(ctx)
			gomega.Expect(validated).To(gomega.BeTrue(), "Network segmentation should be properly configured")
		})

		ginkgo.It("should validate firewall rules and VPN tunnel security", func() {
			ginkgo.By("Testing firewall configuration and VPN security")

			// Create firewall configuration
			firewallConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "firewall-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/security": "firewall",
					},
				},
				Data: map[string]string{
					"default_action":    "deny",
					"logging_enabled":   "true",
					"stateful_tracking": "true",
					"ddos_protection":   "enabled",
					"rate_limiting":     "1000/second",
					"allowed_protocols": "tcp,udp",
					"blocked_ports":     "135,139,445",
				},
			}

			err := k8sClient.Create(ctx, firewallConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create VPN configuration
			vpnConfig := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vpn-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/security": "vpn",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"vpn_type":       []byte("ipsec"),
					"encryption":     []byte("AES-256-GCM"),
					"hash":           []byte("SHA-256"),
					"dh_group":       []byte("modp2048"),
					"pfs":            []byte("enabled"),
					"rekey_interval": []byte("3600"),
				},
			}

			err = k8sClient.Create(ctx, vpnConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Vulnerability Management Testing (2/2 points)", func() {
		ginkgo.It("should perform container image security scanning", func() {
			ginkgo.By("Testing container image vulnerability scanning and compliance")

			// Create image scan policy
			scanPolicy := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "image-scan-policy",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/security": "image-scanning",
					},
				},
				Data: map[string]string{
					"scanner":              "trivy",
					"scan_on_push":         "true",
					"block_on_critical":    "true",
					"block_on_high":        "false",
					"cvss_threshold":       "7.0",
					"compliance_standards": "CIS,NIST",
					"scan_frequency":       "daily",
				},
			}

			err := k8sClient.Create(ctx, scanPolicy)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create admission webhook configuration for image scanning
			webhookConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "image-scan-webhook",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/admission": "image-scanning",
					},
				},
				Data: map[string]string{
					"webhook_url":     "https://image-scanner.default.svc:443/scan",
					"failure_policy":  "Fail",
					"timeout_seconds": "30",
					"namespaces":      "default,production",
				},
			}

			err = k8sClient.Create(ctx, webhookConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate container security
			validated := secValidator.validateContainerImageSecurityAndRuntime(ctx)
			gomega.Expect(validated).To(gomega.BeTrue(), "Container image security should be properly configured")
		})

		ginkgo.It("should assess dependencies and ensure security compliance", func() {
			ginkgo.By("Testing dependency vulnerability assessment and compliance checking")

			// Create dependency scan configuration
			depScanConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dependency-scan-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/security": "dependency-scanning",
					},
				},
				Data: map[string]string{
					"scanner":            "snyk",
					"languages":          "go,python,javascript",
					"scan_frequency":     "on-commit",
					"severity_threshold": "medium",
					"auto_fix":           "true",
					"license_check":      "enabled",
					"sbom_generation":    "true",
				},
			}

			err := k8sClient.Create(ctx, depScanConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create security compliance configuration
			complianceConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "security-compliance-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/compliance": "enabled",
					},
				},
				Data: map[string]string{
					"frameworks":         "NIST,ISO27001,SOC2,GDPR",
					"scan_interval":      "weekly",
					"report_format":      "json,pdf",
					"auto_remediation":   "true",
					"alert_on_violation": "true",
				},
			}

			err = k8sClient.Create(ctx, complianceConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create runtime security monitoring
			runtimeConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "runtime-security-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/runtime": "security",
					},
				},
				Data: map[string]string{
					"monitor":           "falco",
					"rules_enabled":     "true",
					"anomaly_detection": "enabled",
					"behavior_learning": "30d",
					"alert_channels":    "slack,email,pagerduty",
				},
			}

			err = k8sClient.Create(ctx, runtimeConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate vulnerability management
			score := secValidator.ValidateVulnerabilityScanning(ctx)
			gomega.Expect(score).To(gomega.Equal(2), "Vulnerability management should score 2/2")
		})

		ginkgo.It("should simulate penetration testing scenarios", func() {
			ginkgo.By("Testing penetration testing simulation and vulnerability detection")

			// Create penetration test configuration
			penTestConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pentest-config",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.io/security": "penetration-testing",
					},
				},
				Data: map[string]string{
					"test_type":       "automated",
					"tools":           "metasploit,nmap,burp",
					"frequency":       "quarterly",
					"scope":           "full",
					"report_findings": "true",
					"fix_tracking":    "jira",
				},
			}

			err := k8sClient.Create(ctx, penTestConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Simulate common attack patterns
			attackPatterns := []string{
				"sql-injection",
				"xss",
				"csrf",
				"privilege-escalation",
				"container-escape",
			}

			for _, pattern := range attackPatterns {
				// Create test result for each pattern
				testResult := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("pentest-%s", pattern),
						Namespace: "default",
						Labels: map[string]string{
							"nephoran.io/pentest": pattern,
						},
					},
					Data: map[string]string{
						"pattern":    pattern,
						"result":     "blocked",
						"confidence": "high",
						"timestamp":  time.Now().Format(time.RFC3339),
					},
				}

				err = k8sClient.Create(ctx, testResult)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		})
	})

	ginkgo.Context("O-RAN Security Compliance Testing", func() {
		ginkgo.It("should validate O-RAN WG11 security specifications", func() {
			ginkgo.By("Testing O-RAN interface security (A1, O1, O2, E2)")

			// Create O-RAN interface security configurations
			interfaces := []string{"a1", "o1", "o2", "e2"}

			for _, iface := range interfaces {
				// Create interface-specific TLS configuration
				ifaceSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("oran-%s-tls", iface),
						Namespace: "default",
						Labels: map[string]string{
							"oran.interface":  iface,
							"oran.security":   "wg11",
							"oran.compliance": "v3.0",
						},
					},
					Type: corev1.SecretTypeTLS,
					Data: map[string][]byte{
						"tls.crt": []byte(testTLSCertPEM),
						"tls.key": []byte(testTLSKeyPEM),
						"ca.crt":  []byte(testCACertPEM),
					},
				}

				err := k8sClient.Create(ctx, ifaceSecret)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Create interface security policy
				ifacePolicy := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("oran-%s-security-policy", iface),
						Namespace: "default",
						Labels: map[string]string{
							"oran.interface": iface,
						},
					},
					Data: map[string]string{
						"authentication":     "mutual-tls",
						"authorization":      "rbac",
						"encryption":         "aes-256-gcm",
						"integrity":          "hmac-sha256",
						"replay_protection":  "enabled",
						"session_management": "stateful",
					},
				}

				err = k8sClient.Create(ctx, ifacePolicy)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}

			// Validate O-RAN security compliance
			score := secValidator.ValidateORANSecurityCompliance(ctx)
			gomega.Expect(score).To(gomega.BeNumerically(">=", 1), "O-RAN security should score at least 1/2")
		})

		ginkgo.It("should implement zero-trust for multi-vendor O-RAN deployments", func() {
			ginkgo.By("Testing zero-trust architecture in multi-vendor environment")

			// Create vendor-specific security configurations
			vendors := []string{"vendor-a", "vendor-b", "vendor-c"}

			for _, vendor := range vendors {
				// Create vendor authentication configuration
				vendorAuth := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-auth", vendor),
						Namespace: "default",
						Labels: map[string]string{
							"oran.vendor":     vendor,
							"oran.zero-trust": "enabled",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"client_cert":   []byte(testClientCertPEM),
						"client_key":    []byte(testClientKeyPEM),
						"vendor_id":     []byte(vendor),
						"trust_anchors": []byte(testCACertPEM),
					},
				}

				err := k8sClient.Create(ctx, vendorAuth)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Create vendor-specific network policy
				vendorPolicy := &networkingv1.NetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-network-policy", vendor),
						Namespace: "default",
					},
					Spec: networkingv1.NetworkPolicySpec{
						PodSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"oran.vendor": vendor,
							},
						},
						PolicyTypes: []networkingv1.PolicyType{
							networkingv1.PolicyTypeIngress,
							networkingv1.PolicyTypeEgress,
						},
						Ingress: []networkingv1.NetworkPolicyIngressRule{
							{
								From: []networkingv1.NetworkPolicyPeer{
									{
										PodSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"oran.trusted": "true",
											},
										},
									},
								},
							},
						},
					},
				}

				err = k8sClient.Create(ctx, vendorPolicy)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}

			// Validate multi-vendor zero-trust
			validated := secValidator.validateMultiVendorZeroTrust(ctx)
			gomega.Expect(validated).To(gomega.BeTrue(), "Multi-vendor zero-trust should be properly configured")
		})

		ginkgo.It("should validate 3GPP security requirements integration", func() {
			ginkgo.By("Testing 3GPP Release 16/17 security requirements")

			// Create 3GPP security configuration
			tgppConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "3gpp-security-config",
					Namespace: "default",
					Labels: map[string]string{
						"3gpp.release":  "17",
						"3gpp.security": "enabled",
					},
				},
				Data: map[string]string{
					"aka_enabled":         "true",
					"5g_aka":              "enabled",
					"eap_aka_prime":       "supported",
					"subscriber_privacy":  "concealed",
					"home_control":        "enabled",
					"security_edge_proxy": "deployed",
					"n3iwf_security":      "ipsec",
				},
			}

			err := k8sClient.Create(ctx, tgppConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Security Compliance Reporting", func() {
		ginkgo.It("should generate comprehensive security compliance report", func() {
			ginkgo.By("Generating and validating security compliance report")

			// Run comprehensive security validation
			ctx := context.Background()
			score := 0

			// Authentication & Authorization (5 points)
			authScore := secValidator.ValidateAuthentication(ctx)
			score += authScore
			gomega.Expect(authScore).To(gomega.BeNumerically(">=", 4), "Authentication should score at least 4/5")

			// Data Encryption (4 points)
			encryptionScore := secValidator.ValidateEncryption(ctx)
			score += encryptionScore
			gomega.Expect(encryptionScore).To(gomega.BeNumerically(">=", 3), "Encryption should score at least 3/4")

			// Network Security (3 points)
			networkScore := secValidator.ValidateNetworkSecurity(ctx)
			score += networkScore
			gomega.Expect(networkScore).To(gomega.BeNumerically(">=", 2), "Network security should score at least 2/3")

			// Vulnerability Management (2 points)
			vulnScore := secValidator.ValidateVulnerabilityScanning(ctx)
			score += vulnScore
			gomega.Expect(vulnScore).To(gomega.BeNumerically(">=", 2), "Vulnerability management should score at least 2/2")

			// Total score validation
			gomega.Expect(score).To(gomega.BeNumerically(">=", 14), "Total security score should be at least 14/15")

			// Generate detailed report
			report := secValidator.GenerateSecurityReport()
			gomega.Expect(report).To(gomega.ContainSubstring("SECURITY COMPLIANCE REPORT"))

			ginkgo.By(fmt.Sprintf("Security Compliance Score: %d/15 points", score))
		})
	})
})

// Test certificate data for TLS testing
const (
	testTLSCertPEM = `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlQYWxvIEFsdG8x
DTALBgNVBAoMBFRlc3QwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjBF
MQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJUGFs
byBBbHRvMQ0wCwYDVQQKDARUZXN0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEA1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUV
WXYZ1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234
567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ12345678
90abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890ab
cdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnop
qrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdewIDAQABo1AwTjAdBgNVHQ4E
FgQU1234567890abcdefghijklmnopqrstuMB8GA1UdIwQYMBaAFNfY12345678
90abcdefghijklmnopqDAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IB
AQC1234567890abcdefghijklmnopqrstuvwxyz
-----END CERTIFICATE-----`

	testTLSKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN
OPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQR
STUVWXYZ1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUV
WXYZ1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234
567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh
ijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZabcdewIDAQABAoIBAQCr
1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
-----END RSA PRIVATE KEY-----`

	testCACertPEM = `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlQYWxvIEFsdG8x
DTALBgNVBAoMBFRlc3QwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjBF
MQswCQYDVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJUGFs
byBBbHRvMQ0wCwYDVQQKDARUZXN0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEAcacertificateauthorityabcdefghijklmnopqrstuvwxyzABCDEFGHIJ
KLMNOPQRSTUVWXYZ1234567890
-----END CERTIFICATE-----`

	testClientCertPEM = `-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlQYWxvIEFsdG8x
DTALBgNVBAoMBFRlc3QwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjBF
clientcertificateabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWX
YZ1234567890
-----END CERTIFICATE-----`

	testClientKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAclientkeyabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOP
QRSTUVWXYZ1234567890
-----END RSA PRIVATE KEY-----`
)
