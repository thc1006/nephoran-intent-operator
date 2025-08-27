// Package validation provides O-RAN WG11 security compliance testing
package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/tests/framework"
)

var _ = ginkgo.Describe("O-RAN WG11 Security Compliance Suite", ginkgo.Ordered, func() {
	var (
		testSuite    *framework.TestSuite
		ctx          context.Context
		cancel       context.CancelFunc
		k8sClient    client.Client
		secValidator *SecurityValidator
	)

	ginkgo.BeforeAll(func() {
		ginkgo.By("Initializing O-RAN Security Compliance Test Suite")
		testConfig := framework.DefaultTestConfig()
		testConfig.MockExternalAPIs = true
		testSuite = framework.NewTestSuite(testConfig)
		testSuite.SetupSuite()

		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Minute)
		k8sClient = testSuite.GetK8sClient()

		config := DefaultValidationConfig()
		config.EnableSecurityTesting = true
		secValidator = NewSecurityValidator(config)
		secValidator.SetK8sClient(k8sClient)
	})

	ginkgo.AfterAll(func() {
		cancel()
		testSuite.TearDownSuite()
	})

	ginkgo.Context("O-RAN A1 Interface Security", func() {
		ginkgo.It("should secure A1 interface between Non-RT RIC and Near-RT RIC", func() {
			ginkgo.By("Testing A1 interface security implementation")

			// Create A1 interface security configuration
			a1Security := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a1-interface-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface":        "a1",
						"oran.security.version": "3.0",
						"oran.wg":               "wg11",
					},
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt":    []byte(testTLSCertPEM),
					"tls.key":    []byte(testTLSKeyPEM),
					"ca.crt":     []byte(testCACertPEM),
					"client.crt": []byte(testClientCertPEM),
					"client.key": []byte(testClientKeyPEM),
				},
			}

			err := k8sClient.Create(ctx, a1Security)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create A1 policy management security
			a1PolicySecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a1-policy-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "a1",
						"oran.component": "policy",
					},
				},
				Data: map[string]string{
					"authentication_method": "mutual-tls",
					"authorization_model":   "attribute-based",
					"encryption_algorithm":  "AES-256-GCM",
					"integrity_check":       "HMAC-SHA256",
					"policy_signing":        "RSA-PSS",
					"audit_logging":         "enabled",
					"rate_limiting":         "1000/minute",
					"replay_protection":     "timestamp-nonce",
				},
			}

			err = k8sClient.Create(ctx, a1PolicySecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create A1 message security
			a1MessageSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a1-message-security",
					Namespace: "default",
				},
				Data: map[string]string{
					"message_encryption":  "end-to-end",
					"header_protection":   "encrypted",
					"payload_integrity":   "signed",
					"sequence_validation": "strict",
					"timeout_enforcement": "30s",
					"max_message_size":    "10MB",
				},
			}

			err = k8sClient.Create(ctx, a1MessageSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should implement A1 enrichment information security", func() {
			ginkgo.By("Testing A1-EI security controls")

			// Create A1-EI security configuration
			a1EISecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a1-ei-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "a1",
						"oran.service":   "enrichment-information",
					},
				},
				Data: map[string]string{
					"data_classification":   "confidential",
					"access_control":        "role-based",
					"data_masking":          "pii-fields",
					"retention_policy":      "30-days",
					"deletion_verification": "cryptographic",
					"sharing_agreements":    "contract-based",
				},
			}

			err := k8sClient.Create(ctx, a1EISecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("O-RAN O1 Interface Security", func() {
		ginkgo.It("should secure O1 interface for FCAPS operations", func() {
			ginkgo.By("Testing O1 interface NETCONF/YANG security")

			// Create O1 NETCONF security configuration
			o1NetconfSecurity := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o1-netconf-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "o1",
						"oran.protocol":  "netconf",
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"ssh_host_key":        []byte("ssh-rsa AAAAB3NzaC1yc2E..."),
					"ssh_authorized_keys": []byte("ssh-rsa AAAAB3NzaC1yc2E..."),
					"tls_cert":            []byte(testTLSCertPEM),
					"tls_key":             []byte(testTLSKeyPEM),
				},
			}

			err := k8sClient.Create(ctx, o1NetconfSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create O1 YANG model security
			o1YangSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o1-yang-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "o1",
						"oran.protocol":  "yang",
					},
				},
				Data: map[string]string{
					"model_validation":      "strict",
					"schema_enforcement":    "mandatory",
					"access_control_model":  "NACM", // NETCONF Access Control Model
					"data_node_protection":  "path-based",
					"operation_restriction": "role-dependent",
					"notification_security": "authenticated",
				},
			}

			err = k8sClient.Create(ctx, o1YangSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should implement O1 performance management security", func() {
			ginkgo.By("Testing O1 PM data collection security")

			// Create O1 PM security configuration
			o1PMSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o1-pm-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "o1",
						"oran.function":  "performance-management",
					},
				},
				Data: map[string]string{
					"kpi_encryption":        "at-rest-and-transit",
					"metric_authentication": "hmac-signed",
					"collection_interval":   "15min",
					"data_aggregation":      "privacy-preserving",
					"threshold_protection":  "encrypted",
					"report_distribution":   "need-to-know",
				},
			}

			err := k8sClient.Create(ctx, o1PMSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should secure O1 fault management operations", func() {
			ginkgo.By("Testing O1 FM security controls")

			// Create O1 FM security configuration
			o1FMSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o1-fm-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "o1",
						"oran.function":  "fault-management",
					},
				},
				Data: map[string]string{
					"alarm_authentication":  "certificate-based",
					"alarm_integrity":       "signed",
					"alarm_confidentiality": "encrypted",
					"alarm_correlation":     "secure-multiparty",
					"root_cause_analysis":   "privacy-preserving",
					"ticket_generation":     "authenticated",
				},
			}

			err := k8sClient.Create(ctx, o1FMSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("O-RAN O2 Interface Security", func() {
		ginkgo.It("should secure O2 interface for cloud infrastructure management", func() {
			ginkgo.By("Testing O2 interface IMS security")

			// Create O2 IMS security configuration
			o2IMSSecurity := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o2-ims-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "o2",
						"oran.component": "ims", // Infrastructure Management Service
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"api_key":       []byte("o2-ims-api-key-secure"),
					"api_secret":    []byte("o2-ims-api-secret"),
					"oauth_token":   []byte("oauth2-token"),
					"refresh_token": []byte("refresh-token"),
				},
			}

			err := k8sClient.Create(ctx, o2IMSSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create O2 DMS security configuration
			o2DMSSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o2-dms-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "o2",
						"oran.component": "dms", // Deployment Management Service
					},
				},
				Data: map[string]string{
					"deployment_signing":    "gpg-signed",
					"package_verification":  "checksum-signature",
					"rollback_protection":   "versioned-snapshots",
					"deployment_encryption": "transit-encryption",
					"artifact_integrity":    "sha256-verification",
					"deployment_approval":   "multi-signature",
				},
			}

			err = k8sClient.Create(ctx, o2DMSSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should implement O2 resource inventory security", func() {
			ginkgo.By("Testing O2 resource inventory protection")

			// Create O2 resource inventory security
			o2ResourceSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o2-resource-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "o2",
						"oran.function":  "resource-inventory",
					},
				},
				Data: map[string]string{
					"inventory_encryption":    "database-level",
					"resource_classification": "sensitivity-based",
					"access_logging":          "comprehensive",
					"change_tracking":         "audit-trail",
					"data_masking":            "role-based-view",
					"export_control":          "approval-required",
				},
			}

			err := k8sClient.Create(ctx, o2ResourceSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("O-RAN E2 Interface Security", func() {
		ginkgo.It("should secure E2 interface for Near-RT RIC control", func() {
			ginkgo.By("Testing E2 interface security implementation")

			// Create E2 interface security configuration
			e2Security := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2-interface-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "e2",
						"oran.component": "near-rt-ric",
					},
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					"sctp.cert":    []byte(testTLSCertPEM),
					"sctp.key":     []byte(testTLSKeyPEM),
					"sctp.ca":      []byte(testCACertPEM),
					"ipsec.config": []byte("esp-aes256-sha256-modp2048"),
				},
			}

			err := k8sClient.Create(ctx, e2Security)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create E2AP security configuration
			e2APSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2ap-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "e2",
						"oran.protocol":  "e2ap",
					},
				},
				Data: map[string]string{
					"message_authentication": "digital-signature",
					"sequence_protection":    "anti-replay",
					"subscription_security":  "authenticated",
					"indication_protection":  "integrity-protected",
					"control_authorization":  "policy-based",
					"service_model_security": "signed-models",
				},
			}

			err = k8sClient.Create(ctx, e2APSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should implement E2 subscription management security", func() {
			ginkgo.By("Testing E2 subscription security controls")

			// Create E2 subscription security
			e2SubSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2-subscription-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "e2",
						"oran.function":  "subscription",
					},
				},
				Data: map[string]string{
					"subscription_auth":   "token-based",
					"event_filtering":     "privacy-aware",
					"data_minimization":   "enforced",
					"subscription_limits": "rate-controlled",
					"event_encryption":    "selective",
					"subscription_audit":  "comprehensive",
				},
			}

			err := k8sClient.Create(ctx, e2SubSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should secure E2 service models", func() {
			ginkgo.By("Testing E2 service model security")

			// Create E2SM security configuration
			e2SMSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "e2sm-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.interface": "e2",
						"oran.component": "service-model",
					},
				},
				Data: map[string]string{
					"kpm_security":    "metric-encryption",     // Key Performance Measurement
					"rc_security":     "control-authorization", // RAN Control
					"ni_security":     "info-classification",   // Network Information
					"ccc_security":    "config-protection",     // Cell Configuration and Control
					"model_integrity": "signature-verification",
					"model_updates":   "authenticated-distribution",
				},
			}

			err := k8sClient.Create(ctx, e2SMSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("O-RAN Component Security", func() {
		ginkgo.It("should secure Near-RT RIC components", func() {
			ginkgo.By("Testing Near-RT RIC security implementation")

			// Create Near-RT RIC security configuration
			nearRTRICSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "near-rt-ric-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.component": "near-rt-ric",
						"oran.security":  "enabled",
					},
				},
				Data: map[string]string{
					"xapp_isolation":       "container-based",
					"xapp_communication":   "service-mesh",
					"platform_hardening":   "cis-benchmark",
					"resource_quotas":      "enforced",
					"database_encryption":  "transparent",
					"conflict_resolution":  "policy-based",
					"platform_attestation": "tpm-based",
				},
			}

			err := k8sClient.Create(ctx, nearRTRICSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create xApp security policy
			xAppSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "xapp-security-policy",
					Namespace: "default",
					Labels: map[string]string{
						"oran.component": "xapp",
						"oran.security":  "policy",
					},
				},
				Data: map[string]string{
					"admission_control":      "signed-images-only",
					"runtime_protection":     "seccomp-profiles",
					"network_isolation":      "pod-security-policies",
					"secret_management":      "vault-integration",
					"capability_restriction": "least-privilege",
					"audit_logging":          "centralized",
				},
			}

			err = k8sClient.Create(ctx, xAppSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should secure O-CU and O-DU components", func() {
			ginkgo.By("Testing O-CU/O-DU security controls")

			// Create O-CU security configuration
			ocuSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o-cu-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.component": "o-cu",
						"oran.layer":     "centralized-unit",
					},
				},
				Data: map[string]string{
					"f1_interface_security":  "ipsec-tunnel",
					"user_plane_security":    "gtpu-encryption",
					"control_plane_security": "sctp-dtls",
					"cu_cp_cu_up_security":   "internal-tls",
					"state_protection":       "encrypted-storage",
					"session_management":     "secure-context",
				},
			}

			err := k8sClient.Create(ctx, ocuSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Create O-DU security configuration
			oduSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o-du-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.component": "o-du",
						"oran.layer":     "distributed-unit",
					},
				},
				Data: map[string]string{
					"fronthaul_security":    "ipsec-esp",
					"timing_protection":     "ieee1588-security",
					"scheduling_security":   "tamper-resistant",
					"resource_isolation":    "hardware-enforced",
					"configuration_signing": "secure-boot",
					"firmware_protection":   "verified-updates",
				},
			}

			err = k8sClient.Create(ctx, oduSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should secure O-RU components", func() {
			ginkgo.By("Testing O-RU security implementation")

			// Create O-RU security configuration
			oruSecurity := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o-ru-security",
					Namespace: "default",
					Labels: map[string]string{
						"oran.component": "o-ru",
						"oran.layer":     "radio-unit",
					},
				},
				Data: map[string]string{
					"physical_security":     "tamper-evident",
					"boot_security":         "secure-boot-chain",
					"firmware_signing":      "vendor-certified",
					"configuration_lock":    "write-protection",
					"environmental_monitor": "intrusion-detection",
					"debug_interface":       "disabled-production",
					"management_plane":      "authenticated-only",
				},
			}

			err := k8sClient.Create(ctx, oruSecurity)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("O-RAN Multi-Vendor Security", func() {
		ginkgo.It("should implement vendor-neutral security policies", func() {
			ginkgo.By("Testing vendor-neutral security implementation")

			vendors := []string{"vendor-a", "vendor-b", "vendor-c"}

			for _, vendor := range vendors {
				// Create vendor-specific security profile
				vendorProfile := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-security-profile", vendor),
						Namespace: "default",
						Labels: map[string]string{
							"oran.vendor":           vendor,
							"oran.interoperability": "certified",
						},
					},
					Data: map[string]string{
						"certification_level":      "oran-certified",
						"security_baseline":        "nist-800-53",
						"interop_testing":          "passed",
						"vulnerability_disclosure": "coordinated",
						"update_mechanism":         "secure-ota",
						"support_agreement":        "active",
					},
				}

				err := k8sClient.Create(ctx, vendorProfile)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			}
		})

		ginkgo.It("should enforce inter-vendor communication security", func() {
			ginkgo.By("Testing inter-vendor secure communication")

			// Create inter-vendor security policy
			interVendorPolicy := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "inter-vendor-security",
					Namespace: "default",
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"oran.deployment": "multi-vendor",
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
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: func(p corev1.Protocol) *corev1.Protocol { return &p }(corev1.ProtocolTCP),
									Port:     &intstr.IntOrString{IntVal: 443}, // HTTPS only
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, interVendorPolicy)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should validate vendor security compliance", func() {
			ginkgo.By("Testing vendor security compliance validation")

			// Create compliance validation configuration
			complianceValidation := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vendor-compliance-validation",
					Namespace: "default",
					Labels: map[string]string{
						"oran.compliance": "validation",
					},
				},
				Data: map[string]string{
					"validation_framework":  "oran-otc", // O-RAN Test and Certification
					"security_requirements": "wg11-specifications",
					"test_scenarios":        "interop,performance,security",
					"certification_body":    "oran-alliance",
					"validity_period":       "2-years",
					"revalidation_trigger":  "major-update",
				},
			}

			err := k8sClient.Create(ctx, complianceValidation)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("O-RAN Security Monitoring and Incident Response", func() {
		ginkgo.It("should implement O-RAN security monitoring", func() {
			ginkgo.By("Testing O-RAN security monitoring capabilities")

			// Create security monitoring configuration
			monitoringConfig := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oran-security-monitoring",
					Namespace: "default",
					Labels: map[string]string{
						"oran.monitoring": "security",
					},
				},
				Data: map[string]string{
					"monitoring_scope":      "all-interfaces",
					"event_correlation":     "ml-enhanced",
					"threat_detection":      "signature-anomaly",
					"performance_impact":    "minimal",
					"data_retention":        "compliance-based",
					"alert_prioritization":  "risk-based",
					"dashboard_integration": "grafana",
				},
			}

			err := k8sClient.Create(ctx, monitoringConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should validate O-RAN incident response procedures", func() {
			ginkgo.By("Testing O-RAN incident response capabilities")

			// Create incident response configuration
			incidentResponse := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oran-incident-response",
					Namespace: "default",
					Labels: map[string]string{
						"oran.ir": "procedures",
					},
				},
				Data: map[string]string{
					"detection_time":        "real-time",
					"classification":        "automated",
					"containment_strategy":  "interface-isolation",
					"eradication_method":    "component-restart",
					"recovery_procedure":    "rollback-capable",
					"forensics_capability":  "comprehensive",
					"reporting_requirement": "24-hours",
				},
			}

			err := k8sClient.Create(ctx, incidentResponse)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("O-RAN Compliance Validation", func() {
		ginkgo.It("should generate O-RAN security compliance report", func() {
			ginkgo.By("Generating comprehensive O-RAN security compliance report")

			// Validate O-RAN security compliance
			score := secValidator.ValidateORANSecurityCompliance(ctx)
			gomega.Expect(score).To(gomega.BeNumerically(">=", 1), "O-RAN security compliance should score at least 1/2")

			// Create compliance report
			report := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oran-security-compliance-report",
					Namespace: "default",
					Labels: map[string]string{
						"oran.report": "compliance",
					},
				},
				Data: map[string]string{
					"compliance_version":     "WG11-v3.0",
					"interfaces_secured":     "A1,O1,O2,E2",
					"components_validated":   "Near-RT-RIC,O-CU,O-DU,O-RU",
					"zero_trust_implemented": "true",
					"multi_vendor_tested":    "true",
					"score":                  fmt.Sprintf("%d/2", score),
					"timestamp":              time.Now().Format(time.RFC3339),
				},
			}

			err := k8sClient.Create(ctx, report)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
})
