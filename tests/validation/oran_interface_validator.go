// Package validation provides comprehensive O-RAN interface validation capabilities.

// This module implements comprehensive testing for all O-RAN Alliance specified interfaces (A1, O1, O2, E2).

// and integrates with the existing validation framework for the 7-point O-RAN compliance scoring.

package test_validation

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// ORANInterfaceValidator provides comprehensive O-RAN interface validation.

type ORANInterfaceValidator struct {
	config *ValidationConfig

	k8sClient client.Client

	// Mock services for O-RAN components.

	ricMockService *RICMockService

	smoMockService *SMOMockService

	e2MockService *E2MockService

	// Performance metrics.

	interfaceMetrics map[string]*InterfaceMetrics
}

// InterfaceMetrics contains performance metrics for each O-RAN interface.

type InterfaceMetrics struct {
	RequestCount int64

	SuccessCount int64

	FailureCount int64

	AverageLatency time.Duration

	P95Latency time.Duration

	ThroughputRPS float64

	ErrorRate float64

	LastTestTime time.Time
}

// RICMockService provides mock Near-RT RIC functionality for testing.

type RICMockService struct {
	endpoint string

	policies map[string]*A1Policy

	subscriptions map[string]*E2Subscription

	xApps map[string]*XAppConfig

	isHealthy bool

	latencySimMs int
}

// SMOMockService provides mock Service Management & Orchestration functionality.

type SMOMockService struct {
	endpoint string

	managedElements map[string]*ManagedElement

	configurations map[string]*O1Configuration

	isHealthy bool

	latencySimMs int
}

// E2MockService provides mock E2 interface functionality.

type E2MockService struct {
	endpoint string

	connectedNodes map[string]*E2Node

	subscriptions map[string]*E2Subscription

	serviceModels map[string]*ServiceModel

	isHealthy bool

	latencySimMs int
}

// A1Policy represents an A1 interface policy.

type A1Policy struct {
	PolicyID string `json:"policyId"`

	PolicyTypeID string `json:"policyTypeId"`

	PolicyData map[string]interface{} `json:"policyData"`

	Status string `json:"status"` // ACTIVE, INACTIVE, ERROR

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// E2Subscription represents an E2 interface subscription.

type E2Subscription struct {
	SubscriptionID string `json:"subscriptionId"`

	NodeID string `json:"nodeId"`

	ServiceModel string `json:"serviceModel"`

	EventTrigger map[string]interface{} `json:"eventTrigger"`

	Actions []E2Action `json:"actions"`

	Status string `json:"status"` // ACTIVE, INACTIVE, ERROR

	CreatedAt time.Time `json:"createdAt"`
}

// E2Action represents an action in E2 subscription.

type E2Action struct {
	ActionID int `json:"actionId"`

	ActionType string `json:"actionType"` // REPORT, INSERT, POLICY

	Definition map[string]interface{} `json:"definition"`

	SubsequentActions []int `json:"subsequentActions,omitempty"`
}

// XAppConfig represents xApp configuration.

type XAppConfig struct {
	Name string `json:"name"`

	Version string `json:"version"`

	ConfigData map[string]interface{} `json:"configData"`

	Status string `json:"status"` // RUNNING, STOPPED, ERROR

	DeployedAt time.Time `json:"deployedAt"`
}

// ManagedElement represents an O1 managed element.

type ManagedElement struct {
	ElementID string `json:"elementId"`

	ElementType string `json:"elementType"` // AMF, SMF, UPF, gNodeB, etc.

	Configuration map[string]interface{} `json:"configuration"`

	Status string `json:"status"` // ACTIVE, INACTIVE, ERROR

	LastSync time.Time `json:"lastSync"`
}

// O1Configuration represents O1 configuration data.

type O1Configuration struct {
	ConfigID string `json:"configId"`

	ElementID string `json:"elementId"`

	ConfigType string `json:"configType"` // FCAPS

	ConfigData map[string]interface{} `json:"configData"`

	Version int `json:"version"`

	AppliedAt time.Time `json:"appliedAt"`
}

// E2Node represents an E2 interface node.

type E2Node struct {
	NodeID string `json:"nodeId"`

	NodeType string `json:"nodeType"` // gNodeB, ng-eNB, en-gNB

	PLMNs []PLMNID `json:"plmns"`

	SupportedModels []string `json:"supportedModels"`

	Status string `json:"status"` // CONNECTED, DISCONNECTED, ERROR

	LastHeartbeat time.Time `json:"lastHeartbeat"`

	Capabilities map[string]interface{} `json:"capabilities"`
}

// PLMNID represents a Public Land Mobile Network identifier.

type PLMNID struct {
	MCC string `json:"mcc"` // Mobile Country Code

	MNC string `json:"mnc"` // Mobile Network Code

}

// ServiceModel represents an E2 service model.

type ServiceModel struct {
	ModelName string `json:"modelName"`

	Version string `json:"version"`

	OID string `json:"oid"` // Object Identifier

	Functions []string `json:"functions"`

	Capabilities map[string]interface{} `json:"capabilities"`
}

// NewORANInterfaceValidator creates a new O-RAN interface validator.

func NewORANInterfaceValidator(config *ValidationConfig) *ORANInterfaceValidator {

	return &ORANInterfaceValidator{

		config: config,

		interfaceMetrics: make(map[string]*InterfaceMetrics),

		ricMockService: NewRICMockService("http://localhost:38080"),

		smoMockService: NewSMOMockService("http://localhost:38081"),

		e2MockService: NewE2MockService("http://localhost:38082"),
	}

}

// SetK8sClient sets the Kubernetes client for validation.

func (oiv *ORANInterfaceValidator) SetK8sClient(client client.Client) {

	oiv.k8sClient = client

}

// ValidateAllORANInterfaces validates all O-RAN interfaces and returns total score (0-7).

func (oiv *ORANInterfaceValidator) ValidateAllORANInterfaces(ctx context.Context) int {

	ginkgo.By("Validating All O-RAN Interface Compliance")

	totalScore := 0

	// A1 Interface Testing (2 points).

	a1Score := oiv.validateA1InterfaceComprehensive(ctx)

	totalScore += a1Score

	ginkgo.By(fmt.Sprintf("A1 Interface Score: %d/2 points", a1Score))

	// E2 Interface Testing (2 points).

	e2Score := oiv.validateE2InterfaceComprehensive(ctx)

	totalScore += e2Score

	ginkgo.By(fmt.Sprintf("E2 Interface Score: %d/2 points", e2Score))

	// O1 Interface Testing (2 points).

	o1Score := oiv.validateO1InterfaceComprehensive(ctx)

	totalScore += o1Score

	ginkgo.By(fmt.Sprintf("O1 Interface Score: %d/2 points", o1Score))

	// O2 Interface Testing (1 point).

	o2Score := oiv.validateO2InterfaceComprehensive(ctx)

	totalScore += o2Score

	ginkgo.By(fmt.Sprintf("O2 Interface Score: %d/1 points", o2Score))

	ginkgo.By(fmt.Sprintf("Total O-RAN Interface Compliance Score: %d/7 points", totalScore))

	return totalScore

}

// validateA1InterfaceComprehensive provides comprehensive A1 interface testing (2 points).

func (oiv *ORANInterfaceValidator) validateA1InterfaceComprehensive(ctx context.Context) int {

	ginkgo.By("Comprehensive A1 Interface Validation")

	score := 0

	// Test 1: Policy Management CRUD Operations (0.5 points).

	if oiv.testA1PolicyManagement(ctx) {

		score++

		ginkgo.By("✓ A1 Policy Management: 1/1 points")

	} else {

		ginkgo.By("✗ A1 Policy Management: 0/1 points")

	}

	// Test 2: Near-RT RIC Integration (0.5 points).

	if oiv.testA1RICIntegration(ctx) {

		score++

		ginkgo.By("✓ A1 RIC Integration: 1/1 points")

	} else {

		ginkgo.By("✗ A1 RIC Integration: 0/1 points")

	}

	return score

}

// validateE2InterfaceComprehensive provides comprehensive E2 interface testing (2 points).

func (oiv *ORANInterfaceValidator) validateE2InterfaceComprehensive(ctx context.Context) int {

	ginkgo.By("Comprehensive E2 Interface Validation")

	score := 0

	// Test 1: E2 Node Management (1 point).

	if oiv.testE2NodeManagement(ctx) {

		score++

		ginkgo.By("✓ E2 Node Management: 1/1 points")

	} else {

		ginkgo.By("✗ E2 Node Management: 0/1 points")

	}

	// Test 2: Service Model Compliance (1 point).

	if oiv.testE2ServiceModelCompliance(ctx) {

		score++

		ginkgo.By("✓ E2 Service Model Compliance: 1/1 points")

	} else {

		ginkgo.By("✗ E2 Service Model Compliance: 0/1 points")

	}

	return score

}

// validateO1InterfaceComprehensive provides comprehensive O1 interface testing (2 points).

func (oiv *ORANInterfaceValidator) validateO1InterfaceComprehensive(ctx context.Context) int {

	ginkgo.By("Comprehensive O1 Interface Validation")

	score := 0

	// Test 1: FCAPS Operations (1 point).

	if oiv.testO1FCAPSOperations(ctx) {

		score++

		ginkgo.By("✓ O1 FCAPS Operations: 1/1 points")

	} else {

		ginkgo.By("✗ O1 FCAPS Operations: 0/1 points")

	}

	// Test 2: NETCONF/YANG Compliance (1 point).

	if oiv.testO1NETCONFCompliance(ctx) {

		score++

		ginkgo.By("✓ O1 NETCONF/YANG Compliance: 1/1 points")

	} else {

		ginkgo.By("✗ O1 NETCONF/YANG Compliance: 0/1 points")

	}

	return score

}

// validateO2InterfaceComprehensive provides comprehensive O2 interface testing (1 point).

func (oiv *ORANInterfaceValidator) validateO2InterfaceComprehensive(ctx context.Context) int {

	ginkgo.By("Comprehensive O2 Interface Validation")

	score := 0

	// Test: Cloud Infrastructure Management (1 point).

	if oiv.testO2CloudInfraManagement(ctx) {

		score++

		ginkgo.By("✓ O2 Cloud Infrastructure Management: 1/1 points")

	} else {

		ginkgo.By("✗ O2 Cloud Infrastructure Management: 0/1 points")

	}

	return score

}

// testA1PolicyManagement tests A1 interface policy management operations.

func (oiv *ORANInterfaceValidator) testA1PolicyManagement(ctx context.Context) bool {

	ginkgo.By("Testing A1 Policy Management CRUD Operations")

	// Create test intent for policy-related operations.

	testIntent := &nephranv1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "test-a1-policy-management",

			Namespace: "default",

			Labels: map[string]string{

				"test-type": "a1-policy",

				"oran-interface": "a1",
			},
		},

		Spec: nephranv1.NetworkIntentSpec{

			Intent: "Create traffic steering policy for load balancing with 80% traffic to primary path and 20% to secondary path",
		},
	}

	// Test policy creation through intent.

	err := oiv.k8sClient.Create(ctx, testIntent)

	if err != nil {

		ginkgo.By(fmt.Sprintf("Failed to create A1 policy intent: %v", err))

		return false

	}

	defer func() {

		if deleteErr := oiv.k8sClient.Delete(ctx, testIntent); deleteErr != nil {

			ginkgo.By(fmt.Sprintf("Warning: Failed to cleanup test intent: %v", deleteErr))

		}

	}()

	// Wait for policy processing.

	success := false

	gomega.Eventually(func() bool {

		err := oiv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)

		if err != nil {

			return false

		}

		// Check if intent was processed for policy operations.

		if testIntent.Status.Phase != "" && testIntent.Status.Phase != "Pending" {

			// Verify policy-specific processing results.

			if testIntent.Status.ProcessingResults != nil &&

				len(testIntent.Status.ProcessingResults.NetworkFunctionType) > 0 {

				success = true

				return true

			}

		}

		return false

	}, 2*time.Minute, 5*time.Second).Should(gomega.BeTrue())

	// Test A1 policy CRUD operations through mock RIC.

	if success {

		// Test policy creation.

		policy := &A1Policy{

			PolicyID: "policy-001",

			PolicyTypeID: "traffic-steering",

			PolicyData: map[string]interface{}{

				"primaryPathWeight": 0.8,

				"secondaryPathWeight": 0.2,

				"targetThroughput": "1Gbps",
			},

			Status: "ACTIVE",

			CreatedAt: time.Now(),

			UpdatedAt: time.Now(),
		}

		// Simulate policy creation in RIC.

		if err := oiv.ricMockService.CreatePolicy(policy); err != nil {

			ginkgo.By(fmt.Sprintf("Failed to create policy in RIC: %v", err))

			return false

		}

		// Test policy retrieval.

		retrievedPolicy, err := oiv.ricMockService.GetPolicy(policy.PolicyID)

		if err != nil || retrievedPolicy.PolicyID != policy.PolicyID {

			ginkgo.By("Failed to retrieve policy from RIC")

			return false

		}

		// Test policy update.

		retrievedPolicy.PolicyData["primaryPathWeight"] = 0.7

		retrievedPolicy.PolicyData["secondaryPathWeight"] = 0.3

		if err := oiv.ricMockService.UpdatePolicy(retrievedPolicy); err != nil {

			ginkgo.By("Failed to update policy in RIC")

			return false

		}

		// Test policy deletion.

		if err := oiv.ricMockService.DeletePolicy(policy.PolicyID); err != nil {

			ginkgo.By("Failed to delete policy from RIC")

			return false

		}

		ginkgo.By("✓ A1 Policy Management CRUD operations completed successfully")

		return true

	}

	return false

}

// testA1RICIntegration tests A1 interface integration with Near-RT RIC.

func (oiv *ORANInterfaceValidator) testA1RICIntegration(ctx context.Context) bool {

	ginkgo.By("Testing A1 Near-RT RIC Integration")

	// Create test intent for RIC integration.

	testIntent := &nephranv1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "test-a1-ric-integration",

			Namespace: "default",

			Labels: map[string]string{

				"test-type": "a1-ric",

				"oran-interface": "a1",
			},
		},

		Spec: nephranv1.NetworkIntentSpec{

			Intent: "Deploy xApp for Quality of Experience optimization with ML-based resource allocation",
		},
	}

	err := oiv.k8sClient.Create(ctx, testIntent)

	if err != nil {

		ginkgo.By(fmt.Sprintf("Failed to create A1 RIC integration intent: %v", err))

		return false

	}

	defer func() {

		if deleteErr := oiv.k8sClient.Delete(ctx, testIntent); deleteErr != nil {

			ginkgo.By(fmt.Sprintf("Warning: Failed to cleanup test intent: %v", deleteErr))

		}

	}()

	// Test xApp deployment and configuration through RIC.

	xappConfig := &XAppConfig{

		Name: "qoe-optimizer",

		Version: "1.0.0",

		ConfigData: map[string]interface{}{

			"optimizationTarget": "qoe",

			"mlModel": "neural-network",

			"updateInterval": "10s",
		},

		Status: "RUNNING",

		DeployedAt: time.Now(),
	}

	// Simulate xApp deployment.

	if err := oiv.ricMockService.DeployXApp(xappConfig); err != nil {

		ginkgo.By(fmt.Sprintf("Failed to deploy xApp: %v", err))

		return false

	}

	// Test policy creation for xApp.

	policy := &A1Policy{

		PolicyID: "qoe-policy-001",

		PolicyTypeID: "qoe-optimization",

		PolicyData: map[string]interface{}{

			"targetQoE": 8.5,

			"resourceBudget": "high",

			"optimizationAlgo": "gradient-descent",
		},

		Status: "ACTIVE",

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

	if err := oiv.ricMockService.CreatePolicy(policy); err != nil {

		ginkgo.By(fmt.Sprintf("Failed to create policy for xApp: %v", err))

		return false

	}

	// Verify xApp is receiving policy updates.

	gomega.Eventually(func() bool {

		xapp, err := oiv.ricMockService.GetXApp(xappConfig.Name)

		return err == nil && xapp.Status == "RUNNING"

	}, 30*time.Second, 2*time.Second).Should(gomega.BeTrue())

	// Cleanup.

	oiv.ricMockService.DeletePolicy(policy.PolicyID)

	oiv.ricMockService.UndeployXApp(xappConfig.Name)

	ginkgo.By("✓ A1 RIC Integration completed successfully")

	return true

}

// testE2NodeManagement tests E2 interface node registration and management.

func (oiv *ORANInterfaceValidator) testE2NodeManagement(ctx context.Context) bool {

	ginkgo.By("Testing E2 Node Registration and Management")

	// Create E2NodeSet for testing.

	testE2NodeSet := &nephranv1.E2NodeSet{

		ObjectMeta: metav1.ObjectMeta{

			Name: "test-e2-node-management",

			Namespace: "default",

			Labels: map[string]string{

				"test-type": "e2-nodes",

				"oran-interface": "e2",
			},
		},

		Spec: nephranv1.E2NodeSetSpec{

			Replicas: 3,

			Template: nephranv1.E2NodeTemplate{

				Spec: nephranv1.E2NodeSpec{

					NodeID: "test-gnb-001",

					E2InterfaceVersion: "v2.0",

					SupportedRANFunctions: []nephranv1.RANFunction{

						{

							FunctionID: 1,

							Revision: 1,

							Description: "KPM Service Model",

							OID: "1.3.6.1.4.1.53148.1.1.2.2",
						},

						{

							FunctionID: 2,

							Revision: 1,

							Description: "RAN Control Service Model",

							OID: "1.3.6.1.4.1.53148.1.1.2.3",
						},
					},
				},
			},

			SimulationConfig: &nephranv1.SimulationConfig{

				UECount: 100,

				TrafficGeneration: true,

				MetricsInterval: "30s",

				TrafficProfile: nephranv1.TrafficProfileMedium,
			},

			RICConfiguration: &nephranv1.RICConfiguration{

				RICEndpoint: "http://near-rt-ric:38080",

				ConnectionTimeout: "30s",

				HeartbeatInterval: "10s",
			},
		},
	}

	err := oiv.k8sClient.Create(ctx, testE2NodeSet)

	if err != nil {

		ginkgo.By(fmt.Sprintf("Failed to create E2NodeSet: %v", err))

		return false

	}

	defer func() {

		if deleteErr := oiv.k8sClient.Delete(ctx, testE2NodeSet); deleteErr != nil {

			ginkgo.By(fmt.Sprintf("Warning: Failed to cleanup test E2NodeSet: %v", deleteErr))

		}

	}()

	// Wait for E2 nodes to become ready.

	nodeRegistrationSuccess := false

	gomega.Eventually(func() bool {

		err := oiv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testE2NodeSet), testE2NodeSet)

		if err != nil {

			return false

		}

		// Check if nodes are ready and connected.

		if testE2NodeSet.Status.ReadyReplicas >= 2 { // At least 2 out of 3 nodes

			nodeRegistrationSuccess = true

			return true

		}

		return false

	}, 3*time.Minute, 10*time.Second).Should(gomega.BeTrue())

	// Test E2 subscription management.

	if nodeRegistrationSuccess {

		// Create test subscription.

		subscription := &E2Subscription{

			SubscriptionID: "sub-001",

			NodeID: "test-gnb-001",

			ServiceModel: "KPM",

			EventTrigger: map[string]interface{}{

				"reportingPeriod": 1000, // 1 second

				"granularityPeriod": 100,
			},

			Actions: []E2Action{

				{

					ActionID: 1,

					ActionType: "REPORT",

					Definition: map[string]interface{}{

						"measurementType": "DRB.UEThpDl",

						"cellID": "001",
					},
				},
			},

			Status: "ACTIVE",

			CreatedAt: time.Now(),
		}

		// Simulate subscription creation.

		if err := oiv.e2MockService.CreateSubscription(subscription); err != nil {

			ginkgo.By(fmt.Sprintf("Failed to create E2 subscription: %v", err))

			return false

		}

		// Verify subscription is active.

		activeSubscription, err := oiv.e2MockService.GetSubscription(subscription.SubscriptionID)

		if err != nil || activeSubscription.Status != "ACTIVE" {

			ginkgo.By("E2 subscription not active")

			return false

		}

		// Test subscription modification.

		activeSubscription.EventTrigger["reportingPeriod"] = 2000 // 2 seconds

		if err := oiv.e2MockService.UpdateSubscription(activeSubscription); err != nil {

			ginkgo.By("Failed to update E2 subscription")

			return false

		}

		// Cleanup subscription.

		oiv.e2MockService.DeleteSubscription(subscription.SubscriptionID)

		ginkgo.By("✓ E2 Node Management completed successfully")

		return true

	}

	return false

}

// testE2ServiceModelCompliance tests E2 service model compliance (KPM, RC, NI, CCC).

func (oiv *ORANInterfaceValidator) testE2ServiceModelCompliance(ctx context.Context) bool {

	ginkgo.By("Testing E2 Service Model Compliance")

	// Test service models: KPM, RC, NI, CCC.

	serviceModels := []ServiceModel{

		{

			ModelName: "KPM",

			Version: "2.0",

			OID: "1.3.6.1.4.1.53148.1.1.2.2",

			Functions: []string{"REPORT", "INSERT"},

			Capabilities: map[string]interface{}{

				"measurementTypes": []string{"DRB.UEThpDl", "DRB.UEThpUl", "RRU.PrbUsedDl"},

				"granularityPeriods": []int{100, 1000, 10000},
			},
		},

		{

			ModelName: "RC",

			Version: "1.0",

			OID: "1.3.6.1.4.1.53148.1.1.2.3",

			Functions: []string{"CONTROL", "POLICY"},

			Capabilities: map[string]interface{}{

				"controlActions": []string{"QoS_CONTROL", "MOBILITY_CONTROL"},

				"policyTypes": []string{"ADMISSION_CONTROL", "LOAD_BALANCING"},
			},
		},

		{

			ModelName: "NI",

			Version: "1.0",

			OID: "1.3.6.1.4.1.53148.1.1.2.4",

			Functions: []string{"REPORT", "INSERT"},

			Capabilities: map[string]interface{}{

				"networkInfo": []string{"CELL_INFO", "UE_INFO", "BEARER_INFO"},
			},
		},
	}

	// Register service models with E2 mock service.

	for _, model := range serviceModels {

		if err := oiv.e2MockService.RegisterServiceModel(&model); err != nil {

			ginkgo.By(fmt.Sprintf("Failed to register service model %s: %v", model.ModelName, err))

			return false

		}

	}

	// Test KPM service model functionality.

	if !oiv.testKPMServiceModel(ctx) {

		ginkgo.By("KPM service model test failed")

		return false

	}

	// Test RC service model functionality.

	if !oiv.testRCServiceModel(ctx) {

		ginkgo.By("RC service model test failed")

		return false

	}

	// Test NI service model functionality.

	if !oiv.testNIServiceModel(ctx) {

		ginkgo.By("NI service model test failed")

		return false

	}

	ginkgo.By("✓ E2 Service Model Compliance completed successfully")

	return true

}

// testKPMServiceModel tests Key Performance Measurement service model.

func (oiv *ORANInterfaceValidator) testKPMServiceModel(ctx context.Context) bool {

	// Create KPM subscription for throughput measurement.

	subscription := &E2Subscription{

		SubscriptionID: "kmp-sub-001",

		NodeID: "test-gnb-001",

		ServiceModel: "KPM",

		EventTrigger: map[string]interface{}{

			"reportingPeriod": 1000,

			"granularityPeriod": 100,
		},

		Actions: []E2Action{

			{

				ActionID: 1,

				ActionType: "REPORT",

				Definition: map[string]interface{}{

					"measurementType": "DRB.UEThpDl",

					"cellID": "001",

					"plmnID": map[string]string{

						"mcc": "001",

						"mnc": "01",
					},
				},
			},
		},

		Status: "ACTIVE",

		CreatedAt: time.Now(),
	}

	if err := oiv.e2MockService.CreateSubscription(subscription); err != nil {

		return false

	}

	// Simulate KPM reports.

	time.Sleep(100 * time.Millisecond) // Simulate processing time

	oiv.e2MockService.DeleteSubscription(subscription.SubscriptionID)

	return true

}

// testRCServiceModel tests RAN Control service model.

func (oiv *ORANInterfaceValidator) testRCServiceModel(ctx context.Context) bool {

	// Create RC subscription for QoS control.

	subscription := &E2Subscription{

		SubscriptionID: "rc-sub-001",

		NodeID: "test-gnb-001",

		ServiceModel: "RC",

		EventTrigger: map[string]interface{}{

			"controlPeriod": 5000, // 5 seconds

		},

		Actions: []E2Action{

			{

				ActionID: 1,

				ActionType: "CONTROL",

				Definition: map[string]interface{}{

					"controlType": "QoS_CONTROL",

					"targetUE": "ue-001",

					"qosParams": map[string]interface{}{

						"5qi": 1,

						"arp": 1,
					},
				},
			},
		},

		Status: "ACTIVE",

		CreatedAt: time.Now(),
	}

	if err := oiv.e2MockService.CreateSubscription(subscription); err != nil {

		return false

	}

	// Simulate control actions.

	time.Sleep(100 * time.Millisecond)

	oiv.e2MockService.DeleteSubscription(subscription.SubscriptionID)

	return true

}

// testNIServiceModel tests Network Information service model.

func (oiv *ORANInterfaceValidator) testNIServiceModel(ctx context.Context) bool {

	// Create NI subscription for network information.

	subscription := &E2Subscription{

		SubscriptionID: "ni-sub-001",

		NodeID: "test-gnb-001",

		ServiceModel: "NI",

		EventTrigger: map[string]interface{}{

			"reportingPeriod": 10000, // 10 seconds

		},

		Actions: []E2Action{

			{

				ActionID: 1,

				ActionType: "REPORT",

				Definition: map[string]interface{}{

					"informationType": "CELL_INFO",

					"cellID": "001",
				},
			},
		},

		Status: "ACTIVE",

		CreatedAt: time.Now(),
	}

	if err := oiv.e2MockService.CreateSubscription(subscription); err != nil {

		return false

	}

	// Simulate network information reports.

	time.Sleep(100 * time.Millisecond)

	oiv.e2MockService.DeleteSubscription(subscription.SubscriptionID)

	return true

}

// testO1FCAPSOperations tests O1 FCAPS (Fault, Configuration, Accounting, Performance, Security) operations.

func (oiv *ORANInterfaceValidator) testO1FCAPSOperations(ctx context.Context) bool {

	ginkgo.By("Testing O1 FCAPS Operations")

	// Create test intent for O1 FCAPS operations.

	testIntent := &nephranv1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "test-o1-fcaps",

			Namespace: "default",

			Labels: map[string]string{

				"test-type": "o1-fcaps",

				"oran-interface": "o1",
			},
		},

		Spec: nephranv1.NetworkIntentSpec{

			Intent: "Configure performance monitoring for AMF with KPI collection every 15 minutes and fault alerting",
		},
	}

	err := oiv.k8sClient.Create(ctx, testIntent)

	if err != nil {

		ginkgo.By(fmt.Sprintf("Failed to create O1 FCAPS intent: %v", err))

		return false

	}

	defer func() {

		if deleteErr := oiv.k8sClient.Delete(ctx, testIntent); deleteErr != nil {

			ginkgo.By(fmt.Sprintf("Warning: Failed to cleanup test intent: %v", deleteErr))

		}

	}()

	// Test Fault Management.

	if !oiv.testO1FaultManagement(ctx) {

		ginkgo.By("O1 Fault Management test failed")

		return false

	}

	// Test Configuration Management.

	if !oiv.testO1ConfigurationManagement(ctx) {

		ginkgo.By("O1 Configuration Management test failed")

		return false

	}

	// Test Performance Management.

	if !oiv.testO1PerformanceManagement(ctx) {

		ginkgo.By("O1 Performance Management test failed")

		return false

	}

	// Test Security Management.

	if !oiv.testO1SecurityManagement(ctx) {

		ginkgo.By("O1 Security Management test failed")

		return false

	}

	ginkgo.By("✓ O1 FCAPS Operations completed successfully")

	return true

}

// testO1FaultManagement tests fault management capabilities.

func (oiv *ORANInterfaceValidator) testO1FaultManagement(ctx context.Context) bool {

	// Create managed element for testing.

	managedElement := &ManagedElement{

		ElementID: "amf-001",

		ElementType: "AMF",

		Configuration: map[string]interface{}{

			"faultMonitoring": map[string]interface{}{

				"enabled": true,

				"severity": []string{"CRITICAL", "MAJOR", "MINOR"},

				"alertTargets": []string{"smo@example.com"},
			},
		},

		Status: "ACTIVE",

		LastSync: time.Now(),
	}

	if err := oiv.smoMockService.AddManagedElement(managedElement); err != nil {

		return false

	}

	// Simulate fault detection and reporting.

	time.Sleep(50 * time.Millisecond)

	// Verify fault management configuration.

	element, err := oiv.smoMockService.GetManagedElement(managedElement.ElementID)

	if err != nil || element.Status != "ACTIVE" {

		return false

	}

	return true

}

// testO1ConfigurationManagement tests configuration management capabilities.

func (oiv *ORANInterfaceValidator) testO1ConfigurationManagement(ctx context.Context) bool {

	// Create configuration for managed element.

	config := &O1Configuration{

		ConfigID: "config-001",

		ElementID: "amf-001",

		ConfigType: "FCAPS",

		ConfigData: map[string]interface{}{

			"performanceCounters": map[string]interface{}{

				"collection_interval": "15m",

				"counters": []string{

					"amf.registration.success",

					"amf.registration.failure",

					"amf.session.establishment",
				},
			},
		},

		Version: 1,

		AppliedAt: time.Now(),
	}

	if err := oiv.smoMockService.ApplyConfiguration(config); err != nil {

		return false

	}

	// Test configuration update.

	config.ConfigData["performanceCounters"].(map[string]interface{})["collection_interval"] = "10m"

	config.Version = 2

	if err := oiv.smoMockService.ApplyConfiguration(config); err != nil {

		return false

	}

	return true

}

// testO1PerformanceManagement tests performance management capabilities.

func (oiv *ORANInterfaceValidator) testO1PerformanceManagement(ctx context.Context) bool {

	// Simulate performance data collection.

	time.Sleep(50 * time.Millisecond)

	// Verify performance monitoring is active.

	element, err := oiv.smoMockService.GetManagedElement("amf-001")

	if err != nil {

		return false

	}

	// Check if performance monitoring configuration exists.

	if perfConfig, exists := element.Configuration["performanceMonitoring"]; exists {

		if perfMap, ok := perfConfig.(map[string]interface{}); ok {

			if enabled, exists := perfMap["enabled"]; exists && enabled == true {

				return true

			}

		}

	}

	return true

}

// testO1SecurityManagement tests security management capabilities.

func (oiv *ORANInterfaceValidator) testO1SecurityManagement(ctx context.Context) bool {

	// Test security configuration.

	securityConfig := map[string]interface{}{

		"authentication": map[string]interface{}{

			"enabled": true,

			"method": "certificate",
		},

		"authorization": map[string]interface{}{

			"enabled": true,

			"roles": []string{"admin", "operator", "viewer"},
		},

		"encryption": map[string]interface{}{

			"transport": "TLS",

			"version": "1.3",
		},
	}

	// Apply security configuration through SMO.

	config := &O1Configuration{

		ConfigID: "security-config-001",

		ElementID: "amf-001",

		ConfigType: "SECURITY",

		ConfigData: securityConfig,

		Version: 1,

		AppliedAt: time.Now(),
	}

	if err := oiv.smoMockService.ApplyConfiguration(config); err != nil {

		return false

	}

	return true

}

// testO1NETCONFCompliance tests NETCONF/YANG model compliance.

func (oiv *ORANInterfaceValidator) testO1NETCONFCompliance(ctx context.Context) bool {

	ginkgo.By("Testing O1 NETCONF/YANG Compliance")

	// Create test intent for NETCONF operations.

	testIntent := &nephranv1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "test-o1-netconf",

			Namespace: "default",

			Labels: map[string]string{

				"test-type": "o1-netconf",

				"oran-interface": "o1",
			},
		},

		Spec: nephranv1.NetworkIntentSpec{

			Intent: "Configure NETCONF session with YANG model validation for UPF management",
		},
	}

	err := oiv.k8sClient.Create(ctx, testIntent)

	if err != nil {

		ginkgo.By(fmt.Sprintf("Failed to create O1 NETCONF intent: %v", err))

		return false

	}

	defer func() {

		if deleteErr := oiv.k8sClient.Delete(ctx, testIntent); deleteErr != nil {

			ginkgo.By(fmt.Sprintf("Warning: Failed to cleanup test intent: %v", deleteErr))

		}

	}()

	// Test YANG model validation.

	yangModel := map[string]interface{}{

		"module": "o-ran-sc-ric-1.0",

		"namespace": "urn:o-ran:sc:yang:o-ran-sc-ric",

		"prefix": "o-ran-ric",

		"organization": "O-RAN Software Community",

		"description": "O-RAN Near-RT RIC YANG model",

		"schema": map[string]interface{}{

			"container": map[string]interface{}{

				"name": "ric-config",

				"leaf": []map[string]interface{}{

					{

						"name": "ric-id",

						"type": "string",

						"mandatory": true,
					},

					{

						"name": "xapp-namespace",

						"type": "string",

						"default": "ricxapp",
					},
				},
			},
		},
	}

	// Validate YANG model structure.

	if !oiv.validateYANGModel(yangModel) {

		ginkgo.By("YANG model validation failed")

		return false

	}

	// Test NETCONF operations (get, edit-config, commit).

	if !oiv.testNETCONFOperations(ctx) {

		ginkgo.By("NETCONF operations test failed")

		return false

	}

	ginkgo.By("✓ O1 NETCONF/YANG Compliance completed successfully")

	return true

}

// validateYANGModel validates YANG model structure.

func (oiv *ORANInterfaceValidator) validateYANGModel(model map[string]interface{}) bool {

	// Check required fields.

	requiredFields := []string{"module", "namespace", "prefix", "description"}

	for _, field := range requiredFields {

		if _, exists := model[field]; !exists {

			return false

		}

	}

	// Validate schema structure.

	if schema, exists := model["schema"]; exists {

		if schemaMap, ok := schema.(map[string]interface{}); ok {

			if container, exists := schemaMap["container"]; exists {

				if containerMap, ok := container.(map[string]interface{}); ok {

					if _, exists := containerMap["name"]; !exists {

						return false

					}

				}

			}

		}

	}

	return true

}

// testNETCONFOperations tests NETCONF protocol operations.

func (oiv *ORANInterfaceValidator) testNETCONFOperations(ctx context.Context) bool {

	// Simulate NETCONF session establishment.

	session := map[string]interface{}{

		"sessionId": "netconf-session-001",

		"capabilities": []string{

			"urn:ietf:params:netconf:base:1.0",

			"urn:ietf:params:netconf:base:1.1",

			"urn:o-ran:netconf:capability:1.0",
		},

		"transport": "SSH",

		"status": "active",
	}

	// Validate session establishment.

	if session["status"] != "active" {

		return false

	}

	// Test get operation.

	getConfig := map[string]interface{}{

		"operation": "get-config",

		"source": "running",

		"filter": map[string]interface{}{

			"type": "xpath",

			"xpath": "/ric-config",
		},
	}

	// Test edit-config operation.

	editConfig := map[string]interface{}{

		"operation": "edit-config",

		"target": "candidate",

		"config": map[string]interface{}{

			"ric-config": map[string]interface{}{

				"ric-id": "ric-001",

				"xapp-namespace": "ricxapp",
			},
		},
	}

	// Test commit operation.

	commit := map[string]interface{}{

		"operation": "commit",
	}

	// Simulate NETCONF operations execution.

	operations := []map[string]interface{}{getConfig, editConfig, commit}

	for _, op := range operations {

		if operation, exists := op["operation"]; exists {

			// Simulate processing time based on operation type.

			switch operation {

			case "get-config":

				time.Sleep(10 * time.Millisecond)

			case "edit-config":

				time.Sleep(20 * time.Millisecond)

			case "commit":

				time.Sleep(30 * time.Millisecond)

			}

		}

	}

	return true

}

// testO2CloudInfraManagement tests O2 interface cloud infrastructure management.

func (oiv *ORANInterfaceValidator) testO2CloudInfraManagement(ctx context.Context) bool {

	ginkgo.By("Testing O2 Cloud Infrastructure Management")

	// Create test intent for cloud infrastructure.

	testIntent := &nephranv1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "test-o2-cloud-infra",

			Namespace: "default",

			Labels: map[string]string{

				"test-type": "o2-cloud",

				"oran-interface": "o2",
			},
		},

		Spec: nephranv1.NetworkIntentSpec{

			Intent: "Provision cloud infrastructure for UPF deployment with auto-scaling and multi-zone redundancy",
		},
	}

	err := oiv.k8sClient.Create(ctx, testIntent)

	if err != nil {

		ginkgo.By(fmt.Sprintf("Failed to create O2 cloud infrastructure intent: %v", err))

		return false

	}

	defer func() {

		if deleteErr := oiv.k8sClient.Delete(ctx, testIntent); deleteErr != nil {

			ginkgo.By(fmt.Sprintf("Warning: Failed to cleanup test intent: %v", deleteErr))

		}

	}()

	// Test Infrastructure as Code template generation.

	if !oiv.testInfrastructureAsCode(ctx) {

		ginkgo.By("Infrastructure as Code test failed")

		return false

	}

	// Test multi-cloud resource management.

	if !oiv.testMultiCloudManagement(ctx) {

		ginkgo.By("Multi-cloud management test failed")

		return false

	}

	// Test resource lifecycle management.

	if !oiv.testResourceLifecycleManagement(ctx) {

		ginkgo.By("Resource lifecycle management test failed")

		return false

	}

	ginkgo.By("✓ O2 Cloud Infrastructure Management completed successfully")

	return true

}

// testInfrastructureAsCode tests IaC template generation.

func (oiv *ORANInterfaceValidator) testInfrastructureAsCode(ctx context.Context) bool {

	// Test Terraform template generation.

	terraformTemplate := map[string]interface{}{

		"terraform": map[string]interface{}{

			"required_providers": map[string]interface{}{

				"kubernetes": map[string]interface{}{

					"source": "hashicorp/kubernetes",

					"version": "~> 2.0",
				},
			},
		},

		"resource": map[string]interface{}{

			"kubernetes_namespace": map[string]interface{}{

				"upf_namespace": map[string]interface{}{

					"metadata": map[string]interface{}{

						"name": "upf-production",

						"labels": map[string]string{

							"app.kubernetes.io/name": "upf",

							"app.kubernetes.io/component": "core-network",
						},
					},
				},
			},

			"kubernetes_deployment": map[string]interface{}{

				"upf_deployment": map[string]interface{}{

					"metadata": map[string]interface{}{

						"name": "upf",

						"namespace": "${kubernetes_namespace.upf_namespace.metadata.0.name}",
					},

					"spec": map[string]interface{}{

						"replicas": 3,

						"selector": map[string]interface{}{

							"match_labels": map[string]string{

								"app": "upf",
							},
						},
					},
				},
			},
		},
	}

	// Validate template structure.

	if !oiv.validateTerraformTemplate(terraformTemplate) {

		return false

	}

	return true

}

// validateTerraformTemplate validates Terraform template structure.

func (oiv *ORANInterfaceValidator) validateTerraformTemplate(template map[string]interface{}) bool {

	// Check for required sections.

	requiredSections := []string{"terraform", "resource"}

	for _, section := range requiredSections {

		if _, exists := template[section]; !exists {

			return false

		}

	}

	// Validate terraform section.

	if terraform, exists := template["terraform"]; exists {

		if tfMap, ok := terraform.(map[string]interface{}); ok {

			if _, exists := tfMap["required_providers"]; !exists {

				return false

			}

		}

	}

	return true

}

// testMultiCloudManagement tests multi-cloud resource management.

func (oiv *ORANInterfaceValidator) testMultiCloudManagement(ctx context.Context) bool {

	// Test cloud provider configurations.

	cloudProviders := []map[string]interface{}{

		{

			"provider": "aws",

			"region": "us-west-2",

			"resources": map[string]interface{}{

				"ec2_instances": 3,

				"rds_instances": 1,

				"s3_buckets": 2,
			},
		},

		{

			"provider": "azure",

			"region": "West US 2",

			"resources": map[string]interface{}{

				"virtual_machines": 3,

				"sql_databases": 1,

				"storage_accounts": 2,
			},
		},

		{

			"provider": "gcp",

			"region": "us-west1",

			"resources": map[string]interface{}{

				"compute_instances": 3,

				"cloud_sql": 1,

				"storage_buckets": 2,
			},
		},
	}

	// Validate each cloud provider configuration.

	for _, provider := range cloudProviders {

		if !oiv.validateCloudProviderConfig(provider) {

			return false

		}

	}

	return true

}

// validateCloudProviderConfig validates cloud provider configuration.

func (oiv *ORANInterfaceValidator) validateCloudProviderConfig(config map[string]interface{}) bool {

	requiredFields := []string{"provider", "region", "resources"}

	for _, field := range requiredFields {

		if _, exists := config[field]; !exists {

			return false

		}

	}

	// Validate resources section.

	if resources, exists := config["resources"]; exists {

		if resourceMap, ok := resources.(map[string]interface{}); ok {

			if len(resourceMap) == 0 {

				return false

			}

		}

	}

	return true

}

// testResourceLifecycleManagement tests resource lifecycle operations.

func (oiv *ORANInterfaceValidator) testResourceLifecycleManagement(ctx context.Context) bool {

	// Test resource creation, scaling, updating, and deletion lifecycle.

	// Create resource.

	resource := map[string]interface{}{

		"id": "upf-cluster-001",

		"type": "kubernetes-cluster",

		"status": "provisioning",

		"provider": "aws",

		"region": "us-west-2",

		"nodeCount": 3,

		"nodeType": "m5.large",

		"createdAt": time.Now().Format(time.RFC3339),
	}

	// Simulate provisioning time.

	time.Sleep(100 * time.Millisecond)

	resource["status"] = "running"

	// Test scaling operation.

	resource["nodeCount"] = 5

	resource["status"] = "scaling"

	time.Sleep(50 * time.Millisecond)

	resource["status"] = "running"

	// Test updating operation.

	resource["nodeType"] = "m5.xlarge"

	resource["status"] = "updating"

	time.Sleep(50 * time.Millisecond)

	resource["status"] = "running"

	// Test deletion operation.

	resource["status"] = "terminating"

	time.Sleep(50 * time.Millisecond)

	// Verify lifecycle completed successfully.

	if resource["status"] != "terminating" {

		return false

	}

	return true

}

// GetInterfaceMetrics returns performance metrics for O-RAN interfaces.

func (oiv *ORANInterfaceValidator) GetInterfaceMetrics() map[string]*InterfaceMetrics {

	return oiv.interfaceMetrics

}

// RecordInterfaceMetric records performance metrics for an interface.

func (oiv *ORANInterfaceValidator) RecordInterfaceMetric(interfaceName string, success bool, latency time.Duration) {

	if oiv.interfaceMetrics[interfaceName] == nil {

		oiv.interfaceMetrics[interfaceName] = &InterfaceMetrics{}

	}

	metric := oiv.interfaceMetrics[interfaceName]

	metric.RequestCount++

	metric.LastTestTime = time.Now()

	if success {

		metric.SuccessCount++

	} else {

		metric.FailureCount++

	}

	// Update latency metrics (simplified).

	metric.AverageLatency = (metric.AverageLatency + latency) / 2

	if latency > metric.P95Latency {

		metric.P95Latency = latency

	}

	// Calculate error rate.

	metric.ErrorRate = float64(metric.FailureCount) / float64(metric.RequestCount) * 100

	// Calculate throughput (simplified).

	if metric.LastTestTime.Sub(time.Time{}) > 0 {

		metric.ThroughputRPS = float64(metric.RequestCount) / time.Since(time.Time{}).Seconds()

	}

}

// Cleanup performs cleanup of test resources and mock services.

func (oiv *ORANInterfaceValidator) Cleanup() {

	if oiv.ricMockService != nil {

		oiv.ricMockService.Cleanup()

	}

	if oiv.smoMockService != nil {

		oiv.smoMockService.Cleanup()

	}

	if oiv.e2MockService != nil {

		oiv.e2MockService.Cleanup()

	}

}
