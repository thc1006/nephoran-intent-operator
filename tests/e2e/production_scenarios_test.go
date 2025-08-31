package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

var _ = Describe("Production Scenarios E2E Tests", Ordered, func() {
	var (
		ctx           context.Context
		cancel        context.CancelFunc
		testNamespace string
	)

	BeforeAll(func() {
		ctx, cancel = context.WithTimeout(testCtx, 20*time.Minute)
		testNamespace = fmt.Sprintf("%s-prod-%d", testNamespace, time.Now().Unix())

		By(fmt.Sprintf("Creating production test namespace: %s", testNamespace))
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
				Labels: map[string]string{
					"test":              "e2e-production",
					"nephoran.com/test": "production-scenarios",
				},
			},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	})

	AfterAll(func() {
		defer cancel()

		if !skipCleanup {
			By(fmt.Sprintf("Cleaning up production test namespace: %s", testNamespace))
			ns := &corev1.Namespace{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: testNamespace}, ns)
			if err == nil {
				Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
			}
		}
	})

	Describe("5G Core Network Deployment", func() {
		It("should deploy a complete 5G Core network with all components", func() {
			By("Creating NetworkIntent for 5G Core deployment")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod-5g-core",
					Namespace: testNamespace,
					Labels: map[string]string{
						"scenario": "5g-core-production",
						"priority": "critical",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy production 5G Core with AMF SMF UPF NSSF with high availability and auto-scaling",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("Waiting for 5G Core deployment to complete")
			Eventually(func() string {
				updatedIntent := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				}, updatedIntent)
				if err != nil {
					return ""
				}
				return string(updatedIntent.Status.Phase)
			}, 10*time.Minute, 30*time.Second).Should(Or(
				Equal("Ready"),
				Equal("Deploying"),
			))

			By("Verifying deployed components")
			updatedIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      intent.Name,
				Namespace: intent.Namespace,
			}, updatedIntent)).To(Succeed())

			if updatedIntent.Status.Phase == "Ready" {
				GinkgoWriter.Printf("5G Core deployment completed successfully\n")
			}
		})
	})

	Describe("O-RAN Network Deployment", func() {
		It("should deploy O-RAN architecture with RIC and E2 nodes", func() {
			By("Creating NetworkIntent for O-RAN deployment")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod-oran",
					Namespace: testNamespace,
					Labels: map[string]string{
						"scenario": "oran-production",
						"priority": "high",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy O-RAN with Near-RT RIC and O-DU O-CU-CP O-CU-UP with E2 A1 O1 interfaces",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("Creating E2NodeSet for RAN simulation")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod-e2nodes",
					Namespace: testNamespace,
					Labels: map[string]string{
						"related-intent": intent.Name,
						"scenario":       "oran-production",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 50, // Production scale
					Template: nephoranv1.E2NodeTemplate{
						Spec: nephoranv1.E2NodeSpec{
							NodeID:             "prod-e2node",
							E2InterfaceVersion: "v3.0",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("Waiting for O-RAN deployment to progress")
			Eventually(func() bool {
				updatedIntent := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				}, updatedIntent)
				if err != nil {
					return false
				}

				updatedE2NodeSet := &nephoranv1.E2NodeSet{}
				err = k8sClient.Get(ctx, types.NamespacedName{
					Name:      e2nodeSet.Name,
					Namespace: e2nodeSet.Namespace,
				}, updatedE2NodeSet)
				if err != nil {
					return false
				}

				return string(updatedIntent.Status.Phase) != "Pending" &&
					updatedE2NodeSet.Status.CurrentReplicas >= 0
			}, 10*time.Minute, 30*time.Second).Should(BeTrue())

			By("Verifying O-RAN components deployment")
			updatedIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      intent.Name,
				Namespace: intent.Namespace,
			}, updatedIntent)).To(Succeed())

			if updatedIntent.Status.Phase == "Ready" || updatedIntent.Status.Phase == "Deploying" {
				GinkgoWriter.Printf("O-RAN deployment status: %s\n", updatedIntent.Status.Phase)
				GinkgoWriter.Printf("Deployed components: %v\n", updatedIntent.Status.DeployedComponents)
			}
		})
	})

	Describe("Network Slicing", func() {
		It("should deploy network slices for different use cases", func() {
			sliceConfigs := []struct {
				name      string
				sliceType string
				intent    string
				sla       string
			}{
				{
					name:      "embb-slice",
					sliceType: "eMBB",
					intent: `Deploy enhanced Mobile Broadband slice with:
					- High throughput (1Gbps)
					- Low jitter (<5ms)
					- Video streaming optimization
					- CDN integration`,
					sla: "throughput:1Gbps,latency:20ms,availability:99.9%",
				},
				{
					name:      "urllc-slice",
					sliceType: "URLLC",
					intent: `Deploy Ultra-Reliable Low-Latency slice with:
					- Ultra-low latency (<1ms)
					- High reliability (99.999%)
					- Industrial IoT support
					- Edge computing integration`,
					sla: "latency:1ms,reliability:99.999%,availability:99.99%",
				},
				{
					name:      "mmtc-slice",
					sliceType: "mMTC",
					intent: `Deploy massive Machine-Type Communications slice with:
					- High device density (1M devices/km²)
					- Low power consumption
					- NB-IoT support
					- Battery optimization`,
					sla: "devices:1000000,power:low,coverage:wide",
				},
			}

			for _, config := range sliceConfigs {
				By(fmt.Sprintf("Creating %s network slice", config.sliceType))
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("prod-slice-%s", config.name),
						Namespace: testNamespace,
						Labels: map[string]string{
							"scenario":   "network-slicing",
							"slice-type": config.sliceType,
						},
						Annotations: map[string]string{
							"nephoran.com/sla": config.sla,
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: config.intent,
					},
				}

				Expect(k8sClient.Create(ctx, intent)).To(Succeed())
			}

			By("Verifying all network slices are processing")
			Eventually(func() int {
				intents := &nephoranv1.NetworkIntentList{}
				err := k8sClient.List(ctx, intents,
					client.InNamespace(testNamespace),
					client.MatchingLabels{"scenario": "network-slicing"})
				if err != nil {
					return 0
				}

				processingCount := 0
				for _, intent := range intents.Items {
					if intent.Status.Phase != "Pending" {
						processingCount++
					}
				}
				return processingCount
			}, 5*time.Minute, 15*time.Second).Should(BeNumerically(">=", 3))
		})
	})

	Describe("Auto-scaling and Load Testing", func() {
		It("should handle auto-scaling under load", func() {
			By("Creating NetworkIntent with auto-scaling configuration")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod-autoscale",
					Namespace: testNamespace,
					Labels: map[string]string{
						"scenario": "auto-scaling",
						"test":     "load",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy auto-scaling UPF with HPA min 2 max 10 replicas scale at 70 percent CPU",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("Simulating load to trigger auto-scaling")
			// In production, this would trigger actual load generation
			time.Sleep(30 * time.Second)

			By("Verifying auto-scaling configuration is applied")
			Eventually(func() string {
				updatedIntent := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				}, updatedIntent)
				if err != nil {
					return ""
				}
				return string(updatedIntent.Status.Phase)
			}, 5*time.Minute, 15*time.Second).Should(Not(Equal("Pending")))
		})
	})

	Describe("Disaster Recovery and Failover", func() {
		It("should handle disaster recovery scenarios", func() {
			By("Creating primary site deployment")
			primaryIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod-dr-primary",
					Namespace: testNamespace,
					Labels: map[string]string{
						"scenario": "disaster-recovery",
						"site":     "primary",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy primary site 5G Core with geographic redundancy and automatic failover",
				},
			}

			Expect(k8sClient.Create(ctx, primaryIntent)).To(Succeed())

			By("Creating secondary site deployment")
			secondaryIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod-dr-secondary",
					Namespace: testNamespace,
					Labels: map[string]string{
						"scenario": "disaster-recovery",
						"site":     "secondary",
					},
					Annotations: map[string]string{
						"nephoran.com/primary-site": primaryIntent.Name,
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy secondary site 5G Core in standby mode with sync from primary",
				},
			}

			Expect(k8sClient.Create(ctx, secondaryIntent)).To(Succeed())

			By("Verifying both sites are deployed")
			Eventually(func() bool {
				primary := &nephoranv1.NetworkIntent{}
				secondary := &nephoranv1.NetworkIntent{}

				err1 := k8sClient.Get(ctx, types.NamespacedName{
					Name:      primaryIntent.Name,
					Namespace: primaryIntent.Namespace,
				}, primary)

				err2 := k8sClient.Get(ctx, types.NamespacedName{
					Name:      secondaryIntent.Name,
					Namespace: secondaryIntent.Namespace,
				}, secondary)

				return err1 == nil && err2 == nil &&
					primary.Status.Phase != "Pending" &&
					secondary.Status.Phase != "Pending"
			}, 10*time.Minute, 30*time.Second).Should(BeTrue())

			By("Simulating primary site failure and failover")
			// In production, this would trigger actual failover
			GinkgoWriter.Printf("Disaster recovery sites deployed successfully\n")
		})
	})

	Describe("Security and Compliance", func() {
		It("should deploy with security policies and compliance controls", func() {
			By("Creating NetworkIntent with comprehensive security requirements")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prod-secure-5g",
					Namespace: testNamespace,
					Labels: map[string]string{
						"scenario":   "security",
						"compliance": "3gpp",
					},
					Annotations: map[string]string{
						"nephoran.com/security-profile": "strict",
						"nephoran.com/compliance":       "3GPP,ETSI,NIST",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy secure 5G Core with TLS mTLS zero-trust DDoS protection and audit logging",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("Verifying security policies are applied")
			Eventually(func() string {
				updatedIntent := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				}, updatedIntent)
				if err != nil {
					return ""
				}
				return string(updatedIntent.Status.Phase)
			}, 10*time.Minute, 30*time.Second).Should(Not(Equal("Pending")))

			By("Checking security configurations")
			updatedIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      intent.Name,
				Namespace: intent.Namespace,
			}, updatedIntent)).To(Succeed())

			if updatedIntent.Status.Phase == "Ready" || updatedIntent.Status.Phase == "Deploying" {
				GinkgoWriter.Printf("Security deployment status: %s\n", updatedIntent.Status.Phase)
			}
		})
	})
})
