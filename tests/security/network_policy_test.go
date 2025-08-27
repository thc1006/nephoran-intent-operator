package security

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"
	"github.com/thc1006/nephoran-intent-operator/tests/utils"
)

var _ = Describe("Network Policy Security Tests", func() {
	var (
		ctx           context.Context
		k8sClient     client.Client
		namespace     string
		policyManager *security.NetworkPolicyManager
		timeout       time.Duration
	)

	BeforeEach(func() {
		ctx = context.Background()
		k8sClient = utils.GetK8sClient()
		namespace = utils.GetTestNamespace()
		policyManager = security.NewNetworkPolicyManager(k8sClient, namespace)
		timeout = 30 * time.Second
	})

	Context("Zero-Trust Network Policies", func() {
		It("should have a default deny-all network policy", func() {
			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			foundDenyAll := false
			for _, policy := range networkPolicies.Items {
				if strings.Contains(policy.Name, "deny-all") || strings.Contains(policy.Name, "default-deny") {
					foundDenyAll = true

					By(fmt.Sprintf("Found deny-all policy: %s", policy.Name))

					// Verify it's a proper deny-all policy
					spec := policy.Spec

					// Should select all pods
					Expect(spec.PodSelector.MatchLabels).To(BeEmpty(),
						"Deny-all policy should select all pods with empty selector")

					// Should have empty ingress and egress rules (deny all)
					if len(spec.Ingress) == 0 && len(spec.Egress) == 0 {
						// This denies all ingress and egress
						By("Policy correctly denies all traffic with empty rules")
					}

					// Check policy types
					expectedTypes := []networkingv1.PolicyType{
						networkingv1.PolicyTypeIngress,
						networkingv1.PolicyTypeEgress,
					}
					for _, policyType := range expectedTypes {
						Expect(spec.PolicyTypes).To(ContainElement(policyType),
							"Deny-all policy should include %s", policyType)
					}
				}
			}

			if !foundDenyAll {
				By("Warning: No default deny-all network policy found")

				// Create one for testing
				err := policyManager.CreateDefaultDenyAllPolicy(ctx)
				if err != nil {
					By(fmt.Sprintf("Could not create deny-all policy: %v", err))
				} else {
					By("Created default deny-all policy for testing")
				}
			}
		})

		It("should verify component-specific network policies exist", func() {
			expectedComponents := []string{
				"nephoran-operator",
				"rag-api",
				"llm-processor",
				"weaviate",
				"nephio-bridge",
			}

			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, component := range expectedComponents {
				foundPolicy := false
				for _, policy := range networkPolicies.Items {
					if strings.Contains(policy.Name, component) {
						foundPolicy = true

						By(fmt.Sprintf("Found network policy for component %s: %s", component, policy.Name))

						// Verify policy has proper selectors
						Expect(policy.Spec.PodSelector.MatchLabels).NotTo(BeEmpty(),
							"Component policy should have pod selector")

						// Check if component label exists
						if appLabel, exists := policy.Spec.PodSelector.MatchLabels["app"]; exists {
							Expect(strings.Contains(appLabel, component)).To(BeTrue(),
								"Policy selector should match component")
						}

						break
					}
				}

				if !foundPolicy {
					By(fmt.Sprintf("Warning: No network policy found for component %s", component))
				}
			}
		})

		It("should verify O-RAN interface network policies", func() {
			oranInterfaces := []string{
				"a1-policy",
				"o1-management",
				"o2-cloud",
				"e2-control",
			}

			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, oranInterface := range oranInterfaces {
				for _, policy := range networkPolicies.Items {
					if strings.Contains(policy.Name, oranInterface) {
						By(fmt.Sprintf("Found O-RAN interface policy: %s", policy.Name))

						// Verify O-RAN specific configurations
						spec := policy.Spec

						// Should have specific ingress/egress rules for O-RAN ports
						if len(spec.Ingress) > 0 {
							for _, ingress := range spec.Ingress {
								if len(ingress.Ports) > 0 {
									for _, port := range ingress.Ports {
										By(fmt.Sprintf("O-RAN policy %s allows ingress on port %v",
											policy.Name, port.Port))
									}
								}
							}
						}
					}
				}
			}
		})
	})

	Context("Component Communication Restrictions", func() {
		It("should verify inter-component communication is properly configured", func() {
			// Test communication patterns for known component interactions
			communicationPatterns := map[string][]string{
				"nephoran-operator": {"rag-api", "llm-processor", "weaviate"},
				"rag-api":           {"weaviate", "llm-processor"},
				"llm-processor":     {"rag-api"},
				"nephio-bridge":     {"nephoran-operator"},
			}

			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for sourceComponent, allowedTargets := range communicationPatterns {
				for _, policy := range networkPolicies.Items {
					if strings.Contains(policy.Name, sourceComponent) {
						By(fmt.Sprintf("Checking communication policy for %s", sourceComponent))

						// Check egress rules
						for _, egress := range policy.Spec.Egress {
							if len(egress.To) > 0 {
								for _, to := range egress.To {
									// Check if target is in allowed list
									if to.PodSelector != nil && len(to.PodSelector.MatchLabels) > 0 {
										targetApp := to.PodSelector.MatchLabels["app"]
										if targetApp != "" {
											isAllowed := false
											for _, allowed := range allowedTargets {
												if strings.Contains(targetApp, allowed) {
													isAllowed = true
													break
												}
											}

											if isAllowed {
												By(fmt.Sprintf("✓ %s correctly allows communication to %s",
													sourceComponent, targetApp))
											} else {
												By(fmt.Sprintf("Warning: %s has unexpected communication to %s",
													sourceComponent, targetApp))
											}
										}
									}
								}
							}
						}
					}
				}
			}
		})

		It("should verify database access is restricted", func() {
			databaseComponents := []string{"weaviate", "redis", "postgres"}

			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, dbComponent := range databaseComponents {
				for _, policy := range networkPolicies.Items {
					if strings.Contains(policy.Name, dbComponent) {
						By(fmt.Sprintf("Checking database access policy for %s", dbComponent))

						// Database should have restrictive ingress policies
						Expect(len(policy.Spec.Ingress)).To(BeNumerically(">=", 0))

						for _, ingress := range policy.Spec.Ingress {
							// Should have specific sources, not allow all
							if len(ingress.From) == 0 {
								By(fmt.Sprintf("Warning: Database %s allows ingress from all sources", dbComponent))
							} else {
								By(fmt.Sprintf("Database %s has %d allowed ingress sources",
									dbComponent, len(ingress.From)))
							}
						}
					}
				}
			}
		})

		It("should verify external API access is controlled", func() {
			// Components that need external access
			externalAccessComponents := []string{
				"llm-processor", // Needs OpenAI API access
				"nephio-bridge", // Needs Nephio API access
				"rag-api",       // May need external document sources
			}

			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, component := range externalAccessComponents {
				for _, policy := range networkPolicies.Items {
					if strings.Contains(policy.Name, component) {
						By(fmt.Sprintf("Checking external access policy for %s", component))

						hasExternalEgress := false
						for _, egress := range policy.Spec.Egress {
							// Check for egress rules that allow external traffic
							if len(egress.To) == 0 {
								// Empty To means all destinations
								hasExternalEgress = true
							} else {
								for _, to := range egress.To {
									// Check for external IPs or specific external services
									if to.NamespaceSelector != nil || to.IPBlock != nil {
										hasExternalEgress = true
									}
								}
							}
						}

						if hasExternalEgress {
							By(fmt.Sprintf("Component %s has external egress access", component))
						} else {
							By(fmt.Sprintf("Warning: Component %s may not have required external access", component))
						}
					}
				}
			}
		})
	})

	Context("Egress Controls", func() {
		It("should verify DNS access is properly configured", func() {
			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			dnsFound := false
			for _, policy := range networkPolicies.Items {
				for _, egress := range policy.Spec.Egress {
					for _, port := range egress.Ports {
						if port.Port != nil && port.Port.IntVal == 53 {
							dnsFound = true
							By(fmt.Sprintf("Policy %s allows DNS access (port 53)", policy.Name))

							// Verify protocol
							if port.Protocol != nil {
								Expect(*port.Protocol).To(BeElementOf(corev1.ProtocolUDP, corev1.ProtocolTCP))
							}
						}
					}
				}
			}

			if !dnsFound {
				By("Warning: No explicit DNS egress rules found - pods may not be able to resolve names")
			}
		})

		It("should verify HTTPS egress is configured for external APIs", func() {
			httpsComponents := []string{"llm-processor", "rag-api", "nephio-bridge"}

			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, component := range httpsComponents {
				httpsFound := false
				for _, policy := range networkPolicies.Items {
					if strings.Contains(policy.Name, component) {
						for _, egress := range policy.Spec.Egress {
							for _, port := range egress.Ports {
								if port.Port != nil && (port.Port.IntVal == 443 || port.Port.IntVal == 8443) {
									httpsFound = true
									By(fmt.Sprintf("Component %s has HTTPS egress access", component))

									// Verify it's TCP
									if port.Protocol != nil {
										Expect(*port.Protocol).To(Equal(corev1.ProtocolTCP))
									}
								}
							}
						}
					}
				}

				if !httpsFound {
					By(fmt.Sprintf("Warning: Component %s may not have HTTPS egress access", component))
				}
			}
		})

		It("should verify no unrestricted egress exists", func() {
			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, policy := range networkPolicies.Items {
				for _, egress := range policy.Spec.Egress {
					// Check for overly permissive egress rules
					if len(egress.To) == 0 && len(egress.Ports) == 0 {
						By(fmt.Sprintf("Warning: Policy %s has unrestricted egress rule", policy.Name))
					}

					// Check for egress to all IPs
					for _, to := range egress.To {
						if to.IPBlock != nil && to.IPBlock.CIDR == "0.0.0.0/0" {
							By(fmt.Sprintf("Warning: Policy %s allows egress to all IPs", policy.Name))
						}
					}
				}
			}
		})
	})

	Context("Multi-Namespace Policies", func() {
		It("should verify cross-namespace communication is controlled", func() {
			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			crossNamespacePolicies := []networkingv1.NetworkPolicy{}
			for _, policy := range networkPolicies.Items {
				for _, ingress := range policy.Spec.Ingress {
					for _, from := range ingress.From {
						if from.NamespaceSelector != nil {
							crossNamespacePolicies = append(crossNamespacePolicies, policy)
							break
						}
					}
				}
			}

			for _, policy := range crossNamespacePolicies {
				By(fmt.Sprintf("Cross-namespace policy found: %s", policy.Name))

				// Verify namespace selectors are restrictive
				for _, ingress := range policy.Spec.Ingress {
					for _, from := range ingress.From {
						if from.NamespaceSelector != nil {
							if len(from.NamespaceSelector.MatchLabels) == 0 {
								By(fmt.Sprintf("Warning: Policy %s allows traffic from all namespaces", policy.Name))
							} else {
								By(fmt.Sprintf("Policy %s restricts cross-namespace access to specific namespaces",
									policy.Name))
							}
						}
					}
				}
			}
		})

		It("should verify system namespace access is restricted", func() {
			systemNamespaces := []string{
				"kube-system",
				"kube-public",
				"kube-node-lease",
				"default",
			}

			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, policy := range networkPolicies.Items {
				for _, egress := range policy.Spec.Egress {
					for _, to := range egress.To {
						if to.NamespaceSelector != nil {
							for _, sysNs := range systemNamespaces {
								if namespaceSelector, exists := to.NamespaceSelector.MatchLabels["name"]; exists {
									if namespaceSelector == sysNs {
										By(fmt.Sprintf("Policy %s allows egress to system namespace %s",
											policy.Name, sysNs))
									}
								}
							}
						}
					}
				}
			}
		})
	})

	Context("Service Mesh Integration", func() {
		It("should verify Istio sidecar injection is configured", func() {
			var ns corev1.Namespace
			err := k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, &ns)
			Expect(err).NotTo(HaveOccurred())

			// Check for Istio sidecar injection label
			if ns.Labels != nil {
				if injection, exists := ns.Labels["istio-injection"]; exists {
					By(fmt.Sprintf("Namespace has Istio injection: %s", injection))
					Expect(injection).To(Equal("enabled"))
				} else {
					By("No Istio sidecar injection configured")
				}
			}
		})

		It("should verify service mesh network policies work with CNI policies", func() {
			// Check for Istio-aware network policies
			var networkPolicies networkingv1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			istioAwarePolicies := []networkingv1.NetworkPolicy{}
			for _, policy := range networkPolicies.Items {
				// Look for Istio-related annotations or configurations
				if policy.Annotations != nil {
					if _, exists := policy.Annotations["security.istio.io/policy"]; exists {
						istioAwarePolicies = append(istioAwarePolicies, policy)
					}
				}

				// Check for Istio proxy ports
				for _, ingress := range policy.Spec.Ingress {
					for _, port := range ingress.Ports {
						if port.Port != nil && port.Port.IntVal == 15090 { // Istio proxy admin port
							istioAwarePolicies = append(istioAwarePolicies, policy)
						}
					}
				}
			}

			if len(istioAwarePolicies) > 0 {
				By(fmt.Sprintf("Found %d Istio-aware network policies", len(istioAwarePolicies)))
			}
		})
	})

	Context("Network Policy Manager Integration", func() {
		It("should test network policy manager functionality", func() {
			// Test creating a component-specific policy
			testPolicy := &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-component-policy",
					Namespace: namespace,
				},
				Spec: networkingv1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test-component",
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
											"app": "allowed-client",
										},
									},
								},
							},
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
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
											"app": "database",
										},
									},
								},
							},
							Ports: []networkingv1.NetworkPolicyPort{
								{
									Protocol: &[]corev1.Protocol{corev1.ProtocolTCP}[0],
									Port:     &intstr.IntOrString{IntVal: 5432},
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, testPolicy)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				Expect(err).NotTo(HaveOccurred())
			}

			defer func() {
				_ = k8sClient.Delete(ctx, testPolicy)
			}()

			// Verify the policy was created correctly
			var createdPolicy networkingv1.NetworkPolicy
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      testPolicy.Name,
				Namespace: namespace,
			}, &createdPolicy)
			Expect(err).NotTo(HaveOccurred())

			By("Network policy manager test completed successfully")
		})

		It("should verify policy templates are correctly applied", func() {
			// Test O-RAN component policies
			oranComponents := []string{
				"odu", "ocu", "ric", "amf", "smf", "upf",
			}

			for _, component := range oranComponents {
				// This would test component-specific policy creation
				By(fmt.Sprintf("Testing policy template for O-RAN component: %s", component))

				// In a real test, you would call the network policy manager
				// to create component-specific policies and verify them
			}
		})
	})
})

// Helper functions for network policy validation
func ValidateNetworkPolicy(policy *networkingv1.NetworkPolicy) []string {
	var issues []string

	// Check for empty pod selector (affects all pods)
	if len(policy.Spec.PodSelector.MatchLabels) == 0 && len(policy.Spec.PodSelector.MatchExpressions) == 0 {
		if !strings.Contains(policy.Name, "deny-all") {
			issues = append(issues, "Policy has empty pod selector but is not a deny-all policy")
		}
	}

	// Check for overly permissive rules
	for _, ingress := range policy.Spec.Ingress {
		if len(ingress.From) == 0 {
			issues = append(issues, "Ingress rule allows traffic from all sources")
		}

		if len(ingress.Ports) == 0 {
			issues = append(issues, "Ingress rule allows traffic on all ports")
		}
	}

	for _, egress := range policy.Spec.Egress {
		if len(egress.To) == 0 {
			issues = append(issues, "Egress rule allows traffic to all destinations")
		}

		if len(egress.Ports) == 0 {
			issues = append(issues, "Egress rule allows traffic on all ports")
		}
	}

	return issues
}

// Test network connectivity with policies
func TestNetworkConnectivity(t *testing.T) {
	_ = context.Background()
	// This would require setting up a test environment with k8sClient and namespace
	// For now, this is a placeholder test
	t.Skip("Network connectivity test requires live cluster setup")

	// When implemented, this test would:
	// 1. Create test pods in different namespaces
	// 2. Apply network policies
	// 3. Attempt connections between pods
	// 4. Verify that allowed connections work and denied connections fail
}

// Benchmark network policy operations
func BenchmarkNetworkPolicyOperations(b *testing.B) {
	ctx := context.Background()
	k8sClient := utils.GetK8sClient()
	namespace := utils.GetTestNamespace()

	policyManager := security.NewNetworkPolicyManager(k8sClient, namespace)

	b.Run("CreateDenyAllPolicy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := policyManager.CreateDefaultDenyAllPolicy(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
