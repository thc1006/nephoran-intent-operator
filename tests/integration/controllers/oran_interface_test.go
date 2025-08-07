package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("O-RAN Interface Integration Tests", func() {
	var (
		namespace       *corev1.Namespace
		testCtx         context.Context
		a1Server        *httptest.Server
		o1Server        *httptest.Server
		o2Server        *httptest.Server
		e2Server        *httptest.Server
		interfaceTracker *ORANInterfaceTracker
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 6*time.Minute)
		DeferCleanup(cancel)

		// Initialize O-RAN interface tracker
		interfaceTracker = NewORANInterfaceTracker()

		// Setup mock O-RAN interface servers
		a1Server = setupA1InterfaceServer(interfaceTracker)
		o1Server = setupO1InterfaceServer(interfaceTracker)
		o2Server = setupO2InterfaceServer(interfaceTracker)
		e2Server = setupE2InterfaceServer(interfaceTracker)

		DeferCleanup(func() {
			a1Server.Close()
			o1Server.Close()
			o2Server.Close()
			e2Server.Close()
		})
	})

	Describe("A1 Interface Integration", func() {
		Context("when managing RIC policies through A1 interface", func() {
			It("should successfully create and manage policies via A1", func() {
				By("creating NetworkIntent that triggers A1 policy management")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a1-policy-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/a1-endpoint":    a1Server.URL,
							"nephoran.com/policy-enabled": "true",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy Near-RT RIC with traffic steering policies via A1 interface",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNearRTRIC,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for A1 policy operations")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("a1-policy-create")
				}, 90*time.Second, 3*time.Second).Should(BeNumerically(">", 0))

				By("verifying policy type creation")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("a1-policy-type-create")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking created policies")
				policies := interfaceTracker.GetCreatedPolicies()
				Expect(policies).To(HaveLen(BeNumerically(">", 0)))
				
				for _, policy := range policies {
					Expect(policy.PolicyID).NotTo(BeEmpty())
					Expect(policy.PolicyType).To(Or(Equal("QoS-Optimization"), Equal("Traffic-Steering")))
					Expect(policy.Status).To(Equal("active"))
				}
			})

			It("should handle policy updates and lifecycle management", func() {
				By("creating initial policy through NetworkIntent")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a1-policy-update-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/a1-endpoint":      a1Server.URL,
							"nephoran.com/policy-enabled":   "true",
							"nephoran.com/policy-type":      "QoS-Optimization",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Optimize QoS policies for enhanced mobile broadband",
						IntentType: nephoranv1.IntentTypeOptimization,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNearRTRIC,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for initial policy creation")
				Eventually(func() int {
					return len(interfaceTracker.GetCreatedPolicies())
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				initialPolicyCount := len(interfaceTracker.GetCreatedPolicies())

				By("updating the intent to trigger policy modification")
				createdIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(testCtx, types.NamespacedName{
					Name: intent.Name, Namespace: intent.Namespace,
				}, createdIntent)).To(Succeed())

				createdIntent.Spec.Intent = "Update QoS optimization with enhanced throughput targets"
				createdIntent.Annotations["nephoran.com/policy-update"] = "true"
				Expect(k8sClient.Update(testCtx, createdIntent)).To(Succeed())

				By("verifying policy update operations")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("a1-policy-update")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking policy status after update")
				updatedPolicies := interfaceTracker.GetCreatedPolicies()
				Expect(len(updatedPolicies)).To(BeNumerically(">=", initialPolicyCount))
				
				// Verify at least one policy was updated
				foundUpdated := false
				for _, policy := range updatedPolicies {
					if policy.LastModified.After(time.Now().Add(-60*time.Second)) {
						foundUpdated = true
						break
					}
				}
				Expect(foundUpdated).To(BeTrue())
			})

			It("should handle A1 policy enforcement status callbacks", func() {
				By("setting up policy with status callbacks")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a1-callback-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/a1-endpoint":         a1Server.URL,
							"nephoran.com/policy-enabled":      "true",
							"nephoran.com/status-callback":     "true",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy admission control policy with status monitoring",
						IntentType: nephoranv1.IntentTypeDeployment,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNearRTRIC,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for policy creation with callback setup")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("a1-policy-create")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("simulating policy enforcement status update")
				policies := interfaceTracker.GetCreatedPolicies()
				if len(policies) > 0 {
					policy := policies[0]
					interfaceTracker.UpdatePolicyStatus(policy.PolicyID, "enforced", "Policy successfully enforced on xApps")
				}

				By("verifying status callback was received")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("a1-status-callback")
				}, 30*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking policy enforcement status")
				if len(policies) > 0 {
					policy := interfaceTracker.GetPolicy(policies[0].PolicyID)
					Expect(policy).NotTo(BeNil())
					Expect(policy.Status).To(Equal("enforced"))
					Expect(policy.StatusMessage).To(ContainSubstring("successfully enforced"))
				}
			})
		})
	})

	Describe("O1 Interface Integration", func() {
		Context("when managing FCAPS through O1 interface", func() {
			It("should perform configuration management via O1", func() {
				By("creating NetworkIntent requiring O1 configuration")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "o1-config-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/o1-endpoint":      o1Server.URL,
							"nephoran.com/fcaps-enabled":    "true",
							"nephoran.com/config-mgmt":      "true",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Configure O-DU parameters through O1 interface with NETCONF",
						IntentType: nephoranv1.IntentTypeDeployment,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentODU,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for O1 configuration operations")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("o1-config-set")
				}, 90*time.Second, 3*time.Second).Should(BeNumerically(">", 0))

				By("verifying NETCONF operations")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("o1-netconf-edit")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking configuration transactions")
				configs := interfaceTracker.GetConfigurationChanges()
				Expect(configs).To(HaveLen(BeNumerically(">", 0)))
				
				for _, config := range configs {
					Expect(config.Operation).To(Or(Equal("create"), Equal("update")))
					Expect(config.Target).NotTo(BeEmpty())
					Expect(config.Status).To(Equal("committed"))
				}
			})

			It("should collect performance metrics via O1", func() {
				By("creating NetworkIntent with performance monitoring")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "o1-perf-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/o1-endpoint":       o1Server.URL,
							"nephoran.com/perf-monitoring":   "true",
							"nephoran.com/metrics-interval":  "30s",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Monitor O-CU performance metrics via O1 interface",
						IntentType: nephoranv1.IntentTypeOptimization,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentOCUCP,
							nephoranv1.TargetComponentOCUUP,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for performance monitoring setup")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("o1-perf-monitor-start")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("simulating metric collection")
				time.Sleep(5 * time.Second)
				interfaceTracker.SimulateMetricCollection("O-CU-CP", map[string]float64{
					"cpu_utilization":    75.5,
					"memory_usage":       68.2,
					"active_connections": 1250,
					"throughput_mbps":    850.3,
				})

				By("verifying metrics are collected")
				Eventually(func() int {
					return len(interfaceTracker.GetCollectedMetrics())
				}, 30*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				metrics := interfaceTracker.GetCollectedMetrics()
				foundCUMetrics := false
				for component, componentMetrics := range metrics {
					if strings.Contains(component, "O-CU") {
						foundCUMetrics = true
						Expect(componentMetrics).To(HaveKey("cpu_utilization"))
						Expect(componentMetrics).To(HaveKey("throughput_mbps"))
						break
					}
				}
				Expect(foundCUMetrics).To(BeTrue())
			})

			It("should handle fault management and alarms", func() {
				By("creating NetworkIntent with fault management")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "o1-fault-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/o1-endpoint":     o1Server.URL,
							"nephoran.com/fault-mgmt":      "true",
							"nephoran.com/alarm-handling":  "true",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Monitor O-eNB faults and alarms through O1",
						IntentType: nephoranv1.IntentTypeOptimization,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentOENB,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for fault monitoring setup")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("o1-fault-monitor-start")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("simulating fault condition")
				interfaceTracker.SimulateFault("O-eNB-001", "communication-loss", "critical", 
					"Lost communication with UE in cell sector 1")

				By("verifying alarm generation")
				Eventually(func() int {
					return len(interfaceTracker.GetActiveAlarms())
				}, 30*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				alarms := interfaceTracker.GetActiveAlarms()
				foundCriticalAlarm := false
				for _, alarm := range alarms {
					if alarm.Severity == "critical" && alarm.AlarmType == "communication-loss" {
						foundCriticalAlarm = true
						Expect(alarm.Source).To(Equal("O-eNB-001"))
						Expect(alarm.Status).To(Equal("active"))
						break
					}
				}
				Expect(foundCriticalAlarm).To(BeTrue())
			})
		})
	})

	Describe("E2 Interface Integration", func() {
		Context("when managing E2 nodes and subscriptions", func() {
			It("should handle E2NodeSet with E2 interface interactions", func() {
				By("creating E2NodeSet with E2 interface configuration")
				e2NodeSet := &nephoranv1.E2NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "e2-interface-nodeset",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/e2-endpoint": e2Server.URL,
							"nephoran.com/ric-endpoint": e2Server.URL + "/ric",
						},
					},
					Spec: nephoranv1.E2NodeSetSpec{
						Replicas: 5,
						Template: nephoranv1.E2NodeTemplate{
							Spec: nephoranv1.E2NodeSpec{
								NodeID:             "e2-interface-node",
								E2InterfaceVersion: "v3.0",
								SupportedRANFunctions: []nephoranv1.RANFunction{
									{
										FunctionID:  1,
										Revision:    1,
										Description: "KPM Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.2",
									},
									{
										FunctionID:  2,
										Revision:    1,
										Description: "RC Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.3",
									},
								},
							},
						},
						RICConfiguration: &nephoranv1.RICConfiguration{
							RICEndpoint:       e2Server.URL + "/ric",
							ConnectionTimeout: "30s",
							HeartbeatInterval: "10s",
						},
					},
				}

				Expect(k8sClient.Create(testCtx, e2NodeSet)).To(Succeed())

				By("waiting for E2 node connections")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("e2-setup-request")
				}, 90*time.Second, 3*time.Second).Should(BeNumerically(">", 0))

				By("verifying E2 setup responses")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("e2-setup-response")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking connected E2 nodes")
				connectedNodes := interfaceTracker.GetConnectedE2Nodes()
				Expect(connectedNodes).To(HaveLen(BeNumerically(">", 0)))
				
				for _, node := range connectedNodes {
					Expect(node.NodeID).To(ContainSubstring("e2-interface-node"))
					Expect(node.E2InterfaceVersion).To(Equal("v3.0"))
					Expect(node.ConnectionStatus).To(Equal("connected"))
					Expect(node.SupportedFunctions).To(HaveLen(BeNumerically(">=", 2)))
				}
			})

			It("should manage E2 subscriptions and indications", func() {
				By("creating E2NodeSet for subscription testing")
				e2NodeSet := &nephoranv1.E2NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "e2-subscription-nodeset",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/e2-endpoint":        e2Server.URL,
							"nephoran.com/subscription-test":  "true",
						},
					},
					Spec: nephoranv1.E2NodeSetSpec{
						Replicas: 3,
						Template: nephoranv1.E2NodeTemplate{
							Spec: nephoranv1.E2NodeSpec{
								NodeID:             "sub-test-node",
								E2InterfaceVersion: "v3.0",
								SupportedRANFunctions: []nephoranv1.RANFunction{
									{
										FunctionID:  1,
										Revision:    1,
										Description: "KPM Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.2",
									},
								},
							},
						},
					},
				}

				Expect(k8sClient.Create(testCtx, e2NodeSet)).To(Succeed())

				By("waiting for E2 nodes to connect")
				Eventually(func() int {
					return len(interfaceTracker.GetConnectedE2Nodes())
				}, 90*time.Second, 3*time.Second).Should(BeNumerically(">", 0))

				By("creating E2 subscriptions")
				nodes := interfaceTracker.GetConnectedE2Nodes()
				if len(nodes) > 0 {
					interfaceTracker.CreateE2Subscription(nodes[0].NodeID, 1, "KPM", 30*time.Second)
				}

				By("verifying subscription creation")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("e2-subscription-request")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				Eventually(func() int {
					return interfaceTracker.GetOperationCount("e2-subscription-response")
				}, 30*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("simulating E2 indications")
				subscriptions := interfaceTracker.GetActiveSubscriptions()
				if len(subscriptions) > 0 {
					sub := subscriptions[0]
					interfaceTracker.SimulateE2Indication(sub.NodeID, sub.SubscriptionID, map[string]interface{}{
						"timestamp": time.Now(),
						"metrics": map[string]interface{}{
							"dl_prbUsage": 75.5,
							"ul_prbUsage": 68.2,
						},
					})
				}

				By("verifying E2 indications are received")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("e2-indication")
				}, 30*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking indication data")
				indications := interfaceTracker.GetReceivedIndications()
				Expect(indications).To(HaveLen(BeNumerically(">", 0)))
				
				if len(indications) > 0 {
					indication := indications[0]
					Expect(indication.NodeID).NotTo(BeEmpty())
					Expect(indication.Data).To(HaveKey("metrics"))
				}
			})
		})
	})

	Describe("O2 Interface Integration", func() {
		Context("when managing cloud infrastructure through O2", func() {
			It("should handle infrastructure provisioning via O2", func() {
				By("creating NetworkIntent requiring infrastructure provisioning")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "o2-infra-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/o2-endpoint":       o2Server.URL,
							"nephoran.com/infra-provisioning": "true",
							"nephoran.com/cloud-provider":     "kubernetes",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Provision cloud infrastructure for distributed UPF deployment",
						IntentType: nephoranv1.IntentTypeDeployment,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentUPF,
						},
						TargetCluster: "edge-cluster",
						Region:        "us-west-2",
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for infrastructure provisioning")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("o2-infra-provision")
				}, 90*time.Second, 3*time.Second).Should(BeNumerically(">", 0))

				By("verifying resource allocation")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("o2-resource-allocate")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking provisioned infrastructure")
				infraResources := interfaceTracker.GetProvisionedResources()
				Expect(infraResources).To(HaveLen(BeNumerically(">", 0)))
				
				for _, resource := range infraResources {
					Expect(resource.Type).To(Or(Equal("compute"), Equal("network"), Equal("storage")))
					Expect(resource.Status).To(Equal("provisioned"))
					Expect(resource.Region).To(Equal("us-west-2"))
				}
			})

			It("should manage infrastructure lifecycle and scaling", func() {
				By("creating scalable infrastructure intent")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "o2-scaling-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/o2-endpoint":      o2Server.URL,
							"nephoran.com/auto-scaling":     "true",
							"nephoran.com/scale-policy":     "cpu-based",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy auto-scaling AMF cluster with dynamic resource management",
						IntentType: nephoranv1.IntentTypeScaling,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for scaling configuration")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("o2-scaling-config")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("simulating scale-out trigger")
				interfaceTracker.TriggerScalingEvent("scale-out", "cpu-threshold-exceeded", 85.0)

				By("verifying scale-out operation")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("o2-scale-out")
				}, 30*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking scaling history")
				scalingEvents := interfaceTracker.GetScalingEvents()
				Expect(scalingEvents).To(HaveLen(BeNumerically(">", 0)))
				
				foundScaleOut := false
				for _, event := range scalingEvents {
					if event.Action == "scale-out" && event.Reason == "cpu-threshold-exceeded" {
						foundScaleOut = true
						Expect(event.TriggerValue).To(Equal(85.0))
						break
					}
				}
				Expect(foundScaleOut).To(BeTrue())
			})
		})
	})

	Describe("Multi-Interface Coordination", func() {
		Context("when coordinating across multiple O-RAN interfaces", func() {
			It("should coordinate A1 policies with E2 measurements", func() {
				By("creating NetworkIntent requiring A1 and E2 coordination")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-interface-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/a1-endpoint":       a1Server.URL,
							"nephoran.com/e2-endpoint":       e2Server.URL,
							"nephoran.com/interface-coord":   "true",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy intelligent RAN optimization with A1 policies and E2 measurements",
						IntentType: nephoranv1.IntentTypeOptimization,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNearRTRIC,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for A1 policy deployment")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("a1-policy-create")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("waiting for E2 subscription setup")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("e2-subscription-request")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("simulating coordinated measurement and policy update")
				// Simulate E2 indication triggering policy update
				interfaceTracker.SimulateE2Indication("coord-node-1", "sub-001", map[string]interface{}{
					"dl_prbUsage": 90.0, // High usage triggers policy update
					"qci1_delay":  15.0,
				})

				By("verifying policy update based on E2 measurements")
				Eventually(func() int {
					return interfaceTracker.GetOperationCount("a1-policy-update")
				}, 45*time.Second, 3*time.Second).Should(BeNumerically(">", 0))

				By("checking coordination metrics")
				coordMetrics := interfaceTracker.GetCoordinationMetrics()
				Expect(coordMetrics).To(HaveKey("a1_e2_coordination_events"))
				Expect(coordMetrics["a1_e2_coordination_events"]).To(BeNumerically(">", 0))
			})
		})
	})
})

// ORANInterfaceTracker tracks O-RAN interface operations for testing
type ORANInterfaceTracker struct {
	mu                   sync.RWMutex
	operationCounts      map[string]int
	createdPolicies      []A1Policy
	configChanges        []O1Configuration
	collectedMetrics     map[string]map[string]float64
	activeAlarms         []O1Alarm
	connectedE2Nodes     []E2Node
	activeSubscriptions  []E2Subscription
	receivedIndications  []E2Indication
	provisionedResources []O2Resource
	scalingEvents        []O2ScalingEvent
	coordinationMetrics  map[string]float64
}

// A1Policy represents an A1 interface policy
type A1Policy struct {
	PolicyID      string
	PolicyType    string
	Status        string
	StatusMessage string
	CreatedAt     time.Time
	LastModified  time.Time
}

// O1Configuration represents an O1 configuration change
type O1Configuration struct {
	ConfigID  string
	Target    string
	Operation string
	Status    string
	Timestamp time.Time
}

// O1Alarm represents an O1 fault management alarm
type O1Alarm struct {
	AlarmID   string
	Source    string
	AlarmType string
	Severity  string
	Status    string
	Message   string
	Timestamp time.Time
}

// E2Node represents a connected E2 node
type E2Node struct {
	NodeID               string
	E2InterfaceVersion   string
	ConnectionStatus     string
	SupportedFunctions   []RANFunctionInfo
	LastHeartbeat        time.Time
	ConnectionTimestamp  time.Time
}

// RANFunctionInfo represents RAN function information
type RANFunctionInfo struct {
	FunctionID  int
	Revision    int
	Description string
	OID         string
}

// E2Subscription represents an E2 subscription
type E2Subscription struct {
	SubscriptionID string
	NodeID         string
	FunctionID     int
	ServiceModel   string
	ReportPeriod   time.Duration
	Status         string
	CreatedAt      time.Time
}

// E2Indication represents received E2 indication
type E2Indication struct {
	NodeID         string
	SubscriptionID string
	Data           map[string]interface{}
	Timestamp      time.Time
}

// O2Resource represents provisioned infrastructure resource
type O2Resource struct {
	ResourceID string
	Type       string
	Status     string
	Region     string
	Specs      map[string]interface{}
	CreatedAt  time.Time
}

// O2ScalingEvent represents a scaling event
type O2ScalingEvent struct {
	EventID      string
	Action       string
	Reason       string
	TriggerValue float64
	Timestamp    time.Time
}

func NewORANInterfaceTracker() *ORANInterfaceTracker {
	return &ORANInterfaceTracker{
		operationCounts:      make(map[string]int),
		createdPolicies:      make([]A1Policy, 0),
		configChanges:        make([]O1Configuration, 0),
		collectedMetrics:     make(map[string]map[string]float64),
		activeAlarms:         make([]O1Alarm, 0),
		connectedE2Nodes:     make([]E2Node, 0),
		activeSubscriptions:  make([]E2Subscription, 0),
		receivedIndications:  make([]E2Indication, 0),
		provisionedResources: make([]O2Resource, 0),
		scalingEvents:        make([]O2ScalingEvent, 0),
		coordinationMetrics:  make(map[string]float64),
	}
}

func (oit *ORANInterfaceTracker) IncrementOperation(operation string) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	oit.operationCounts[operation]++
}

func (oit *ORANInterfaceTracker) GetOperationCount(operation string) int {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return oit.operationCounts[operation]
}

// A1 Interface methods
func (oit *ORANInterfaceTracker) CreatePolicy(policyType string) A1Policy {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	policy := A1Policy{
		PolicyID:     fmt.Sprintf("policy-%d", len(oit.createdPolicies)+1),
		PolicyType:   policyType,
		Status:       "active",
		CreatedAt:    time.Now(),
		LastModified: time.Now(),
	}
	
	oit.createdPolicies = append(oit.createdPolicies, policy)
	return policy
}

func (oit *ORANInterfaceTracker) GetCreatedPolicies() []A1Policy {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return append([]A1Policy(nil), oit.createdPolicies...)
}

func (oit *ORANInterfaceTracker) GetPolicy(policyID string) *A1Policy {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	
	for i, policy := range oit.createdPolicies {
		if policy.PolicyID == policyID {
			return &oit.createdPolicies[i]
		}
	}
	return nil
}

func (oit *ORANInterfaceTracker) UpdatePolicyStatus(policyID, status, message string) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	for i, policy := range oit.createdPolicies {
		if policy.PolicyID == policyID {
			oit.createdPolicies[i].Status = status
			oit.createdPolicies[i].StatusMessage = message
			oit.createdPolicies[i].LastModified = time.Now()
			break
		}
	}
}

// O1 Interface methods
func (oit *ORANInterfaceTracker) AddConfigurationChange(target, operation string) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	config := O1Configuration{
		ConfigID:  fmt.Sprintf("config-%d", len(oit.configChanges)+1),
		Target:    target,
		Operation: operation,
		Status:    "committed",
		Timestamp: time.Now(),
	}
	
	oit.configChanges = append(oit.configChanges, config)
}

func (oit *ORANInterfaceTracker) GetConfigurationChanges() []O1Configuration {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return append([]O1Configuration(nil), oit.configChanges...)
}

func (oit *ORANInterfaceTracker) SimulateMetricCollection(component string, metrics map[string]float64) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	oit.collectedMetrics[component] = metrics
}

func (oit *ORANInterfaceTracker) GetCollectedMetrics() map[string]map[string]float64 {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	
	result := make(map[string]map[string]float64)
	for k, v := range oit.collectedMetrics {
		componentMetrics := make(map[string]float64)
		for mk, mv := range v {
			componentMetrics[mk] = mv
		}
		result[k] = componentMetrics
	}
	return result
}

func (oit *ORANInterfaceTracker) SimulateFault(source, alarmType, severity, message string) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	alarm := O1Alarm{
		AlarmID:   fmt.Sprintf("alarm-%d", len(oit.activeAlarms)+1),
		Source:    source,
		AlarmType: alarmType,
		Severity:  severity,
		Status:    "active",
		Message:   message,
		Timestamp: time.Now(),
	}
	
	oit.activeAlarms = append(oit.activeAlarms, alarm)
}

func (oit *ORANInterfaceTracker) GetActiveAlarms() []O1Alarm {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return append([]O1Alarm(nil), oit.activeAlarms...)
}

// E2 Interface methods
func (oit *ORANInterfaceTracker) ConnectE2Node(nodeID, version string, functions []RANFunctionInfo) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	node := E2Node{
		NodeID:               nodeID,
		E2InterfaceVersion:   version,
		ConnectionStatus:     "connected",
		SupportedFunctions:   functions,
		LastHeartbeat:        time.Now(),
		ConnectionTimestamp:  time.Now(),
	}
	
	oit.connectedE2Nodes = append(oit.connectedE2Nodes, node)
}

func (oit *ORANInterfaceTracker) GetConnectedE2Nodes() []E2Node {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return append([]E2Node(nil), oit.connectedE2Nodes...)
}

func (oit *ORANInterfaceTracker) CreateE2Subscription(nodeID string, functionID int, serviceModel string, reportPeriod time.Duration) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	subscription := E2Subscription{
		SubscriptionID: fmt.Sprintf("sub-%d", len(oit.activeSubscriptions)+1),
		NodeID:         nodeID,
		FunctionID:     functionID,
		ServiceModel:   serviceModel,
		ReportPeriod:   reportPeriod,
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	
	oit.activeSubscriptions = append(oit.activeSubscriptions, subscription)
}

func (oit *ORANInterfaceTracker) GetActiveSubscriptions() []E2Subscription {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return append([]E2Subscription(nil), oit.activeSubscriptions...)
}

func (oit *ORANInterfaceTracker) SimulateE2Indication(nodeID, subscriptionID string, data map[string]interface{}) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	indication := E2Indication{
		NodeID:         nodeID,
		SubscriptionID: subscriptionID,
		Data:           data,
		Timestamp:      time.Now(),
	}
	
	oit.receivedIndications = append(oit.receivedIndications, indication)
	
	// Update coordination metrics if this triggers cross-interface coordination
	oit.coordinationMetrics["a1_e2_coordination_events"]++
}

func (oit *ORANInterfaceTracker) GetReceivedIndications() []E2Indication {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return append([]E2Indication(nil), oit.receivedIndications...)
}

// O2 Interface methods
func (oit *ORANInterfaceTracker) ProvisionResource(resourceType, region string, specs map[string]interface{}) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	resource := O2Resource{
		ResourceID: fmt.Sprintf("resource-%d", len(oit.provisionedResources)+1),
		Type:       resourceType,
		Status:     "provisioned",
		Region:     region,
		Specs:      specs,
		CreatedAt:  time.Now(),
	}
	
	oit.provisionedResources = append(oit.provisionedResources, resource)
}

func (oit *ORANInterfaceTracker) GetProvisionedResources() []O2Resource {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return append([]O2Resource(nil), oit.provisionedResources...)
}

func (oit *ORANInterfaceTracker) TriggerScalingEvent(action, reason string, triggerValue float64) {
	oit.mu.Lock()
	defer oit.mu.Unlock()
	
	event := O2ScalingEvent{
		EventID:      fmt.Sprintf("scale-%d", len(oit.scalingEvents)+1),
		Action:       action,
		Reason:       reason,
		TriggerValue: triggerValue,
		Timestamp:    time.Now(),
	}
	
	oit.scalingEvents = append(oit.scalingEvents, event)
}

func (oit *ORANInterfaceTracker) GetScalingEvents() []O2ScalingEvent {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	return append([]O2ScalingEvent(nil), oit.scalingEvents...)
}

func (oit *ORANInterfaceTracker) GetCoordinationMetrics() map[string]float64 {
	oit.mu.RLock()
	defer oit.mu.RUnlock()
	
	result := make(map[string]float64)
	for k, v := range oit.coordinationMetrics {
		result[k] = v
	}
	return result
}

// Mock server setup functions
func setupA1InterfaceServer(tracker *ORANInterfaceTracker) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/a1-p/policytypes"):
			handleA1PolicyType(w, r, tracker)
		case strings.Contains(r.URL.Path, "/a1-p/policies"):
			handleA1Policy(w, r, tracker)
		case strings.Contains(r.URL.Path, "/a1-p/status"):
			handleA1Status(w, r, tracker)
		default:
			http.NotFound(w, r)
		}
	}))
}

func setupO1InterfaceServer(tracker *ORANInterfaceTracker) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/netconf"):
			handleO1NETCONF(w, r, tracker)
		case strings.Contains(r.URL.Path, "/performance"):
			handleO1Performance(w, r, tracker)
		case strings.Contains(r.URL.Path, "/fault"):
			handleO1Fault(w, r, tracker)
		default:
			http.NotFound(w, r)
		}
	}))
}

func setupO2InterfaceServer(tracker *ORANInterfaceTracker) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/infrastructure"):
			handleO2Infrastructure(w, r, tracker)
		case strings.Contains(r.URL.Path, "/scaling"):
			handleO2Scaling(w, r, tracker)
		default:
			http.NotFound(w, r)
		}
	}))
}

func setupE2InterfaceServer(tracker *ORANInterfaceTracker) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/e2setup"):
			handleE2Setup(w, r, tracker)
		case strings.Contains(r.URL.Path, "/subscription"):
			handleE2Subscription(w, r, tracker)
		case strings.Contains(r.URL.Path, "/indication"):
			handleE2Indication(w, r, tracker)
		default:
			http.NotFound(w, r)
		}
	}))
}

// Handler implementations (simplified for testing)
func handleA1PolicyType(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("a1-policy-type-create")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func handleA1Policy(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	switch r.Method {
	case http.MethodPost:
		tracker.IncrementOperation("a1-policy-create")
		policy := tracker.CreatePolicy("QoS-Optimization")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(policy)
	case http.MethodPut:
		tracker.IncrementOperation("a1-policy-update")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

func handleA1Status(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("a1-status-callback")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

func handleO1NETCONF(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("o1-netconf-edit")
	tracker.IncrementOperation("o1-config-set")
	tracker.AddConfigurationChange("O-DU", "create")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "configured"})
}

func handleO1Performance(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("o1-perf-monitor-start")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "monitoring started"})
}

func handleO1Fault(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("o1-fault-monitor-start")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "fault monitoring started"})
}

func handleO2Infrastructure(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("o2-infra-provision")
	tracker.IncrementOperation("o2-resource-allocate")
	tracker.ProvisionResource("compute", "us-west-2", map[string]interface{}{"cpu": 4, "memory": "8Gi"})
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "provisioned"})
}

func handleO2Scaling(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	if strings.Contains(r.URL.Path, "config") {
		tracker.IncrementOperation("o2-scaling-config")
	} else if strings.Contains(r.URL.Path, "out") {
		tracker.IncrementOperation("o2-scale-out")
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "scaling configured"})
}

func handleE2Setup(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("e2-setup-request")
	tracker.IncrementOperation("e2-setup-response")
	
	functions := []RANFunctionInfo{
		{FunctionID: 1, Revision: 1, Description: "KPM Service Model", OID: "1.3.6.1.4.1.53148.1.1.2.2"},
		{FunctionID: 2, Revision: 1, Description: "RC Service Model", OID: "1.3.6.1.4.1.53148.1.1.2.3"},
	}
	tracker.ConnectE2Node("test-node-1", "v3.0", functions)
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "connected"})
}

func handleE2Subscription(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("e2-subscription-request")
	tracker.IncrementOperation("e2-subscription-response")
	tracker.CreateE2Subscription("test-node-1", 1, "KPM", 30*time.Second)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "subscribed"})
}

func handleE2Indication(w http.ResponseWriter, r *http.Request, tracker *ORANInterfaceTracker) {
	tracker.IncrementOperation("e2-indication")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "indication received"})
}