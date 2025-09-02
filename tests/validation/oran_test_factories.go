// Package validation provides comprehensive test data factories for O-RAN interface testing.

// This module creates realistic test data for A1, E2, O1, and O2 interface testing scenarios.

package test_validation

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// ORANTestFactory provides factory methods for creating O-RAN test data.

type ORANTestFactory struct {
	nameCounter int

	timeBase time.Time
}

// NewORANTestFactory creates a new O-RAN test factory.

func NewORANTestFactory() *ORANTestFactory {
	return &ORANTestFactory{
		nameCounter: 1,

		timeBase: time.Now(),
	}
}

// GetNextName generates a unique test name.

func (otf *ORANTestFactory) GetNextName(prefix string) string {
	name := fmt.Sprintf("%s-%03d", prefix, otf.nameCounter)

	otf.nameCounter++

	return name
}

// A1 Interface Test Factories.

// CreateA1PolicyManagementIntent creates a NetworkIntent for A1 policy management testing.

func (otf *ORANTestFactory) CreateA1PolicyManagementIntent(scenario string) *nephranv1.NetworkIntent {
	var intent string

	var labels map[string]string

	switch scenario {

	case "traffic-steering":

		intent = "Create traffic steering policy for load balancing with 70% traffic to primary RAN node and 30% to secondary node"

		labels = map[string]string{
			"test-type": "a1-traffic-steering",

			"oran-interface": "a1",

			"policy-type": "traffic-steering",
		}

	case "qos-optimization":

		intent = "Deploy QoS optimization policy for enhanced mobile broadband with latency target 10ms and throughput 1Gbps"

		labels = map[string]string{
			"test-type": "a1-qos-optimization",

			"oran-interface": "a1",

			"policy-type": "qos-optimization",
		}

	case "admission-control":

		intent = "Configure admission control policy for URLLC services with priority-based resource allocation"

		labels = map[string]string{
			"test-type": "a1-admission-control",

			"oran-interface": "a1",

			"policy-type": "admission-control",
		}

	case "energy-saving":

		intent = "Create energy saving policy for RAN optimization during low traffic periods with 20% power reduction"

		labels = map[string]string{
			"test-type": "a1-energy-saving",

			"oran-interface": "a1",

			"policy-type": "energy-saving",
		}

	default:

		intent = "Create basic traffic management policy for RAN optimization"

		labels = map[string]string{
			"test-type": "a1-basic-policy",

			"oran-interface": "a1",

			"policy-type": "basic",
		}

	}

	return &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name: otf.GetNextName("a1-policy-intent"),

			Namespace: "default",

			Labels: labels,

			Annotations: map[string]string{
				"test-scenario": scenario,

				"test-created-at": otf.timeBase.Format(time.RFC3339),
			},
		},

		Spec: nephranv1.NetworkIntentSpec{
			Intent: intent,

			IntentType: nephranv1.IntentTypeOptimization,

			Priority: nephranv1.NetworkPriorityNormal,

			TargetComponents: []nephranv1.NetworkTargetComponent{
				nephranv1.NetworkTargetComponentNRF,
				nephranv1.NetworkTargetComponentXApp,
			},
		},
	}
}

// CreateA1Policy creates a test A1 policy.

func (otf *ORANTestFactory) CreateA1Policy(policyType string) *A1Policy {
	var policyData map[string]interface{}

	switch policyType {

	case "traffic-steering":

		policyData = json.RawMessage(`{}`),
		}

	case "qos-optimization":

		policyData = json.RawMessage(`{}`)

	case "admission-control":

		policyData = json.RawMessage(`{}`)

	case "energy-saving":

		policyData = json.RawMessage(`{}`), // 10 PM to 5 AM

		}

	default:

		policyData = json.RawMessage(`{}`)

	}

	return &A1Policy{
		PolicyID: otf.GetNextName("policy"),

		PolicyTypeID: policyType,

		PolicyData: policyData,

		Status: "ACTIVE",

		CreatedAt: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Second),

		UpdatedAt: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Second),
	}
}

// CreateXAppConfig creates a test xApp configuration.

func (otf *ORANTestFactory) CreateXAppConfig(xappType string) *XAppConfig {
	var configData map[string]interface{}

	switch xappType {

	case "qoe-optimizer":

		configData = json.RawMessage(`{}`),

			"adaptationThreshold": 0.1,
		}

	case "load-balancer":

		configData = json.RawMessage(`{}`),
		}

	case "anomaly-detector":

		configData = json.RawMessage(`{}`),

			"alertThreshold": 0.95,
		}

	case "slice-optimizer":

		configData = json.RawMessage(`{}`),

			"optimizationGoal": "resource-efficiency",

			"rebalanceInterval": "30s",

			"minSliceResources": map[string]string{
				"cpu": "100m",

				"memory": "256Mi",
			},
		}

	default:

		configData = json.RawMessage(`{}`)

	}

	return &XAppConfig{
		Name: otf.GetNextName("xapp-" + xappType),

		Version: "1.0.0",

		ConfigData: configData,

		Status: "RUNNING",

		DeployedAt: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Minute),
	}
}

// E2 Interface Test Factories.

// CreateE2NodeManagementIntent creates a NetworkIntent for E2 node management testing.

func (otf *ORANTestFactory) CreateE2NodeManagementIntent(scenario string) *nephranv1.NetworkIntent {
	var intent string

	var labels map[string]string

	switch scenario {

	case "gnodeb-registration":

		intent = "Register gNodeB with E2 interface supporting KPM and RC service models for 5G standalone deployment"

		labels = map[string]string{
			"test-type": "e2-gnodeb-registration",

			"oran-interface": "e2",

			"node-type": "gnodeb",
		}

	case "enb-registration":

		intent = "Register eNodeB with E2 interface for LTE to 5G NSA deployment with measurement reporting"

		labels = map[string]string{
			"test-type": "e2-enb-registration",

			"oran-interface": "e2",

			"node-type": "enb",
		}

	case "multi-node-deployment":

		intent = "Deploy multiple E2 nodes with different configurations for distributed RAN testing"

		labels = map[string]string{
			"test-type": "e2-multi-node",

			"oran-interface": "e2",

			"deployment-type": "distributed",
		}

	default:

		intent = "Create basic E2 node registration for RAN testing"

		labels = map[string]string{
			"test-type": "e2-basic-node",

			"oran-interface": "e2",

			"node-type": "generic",
		}

	}

	return &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name: otf.GetNextName("e2-node-intent"),

			Namespace: "default",

			Labels: labels,

			Annotations: map[string]string{
				"test-scenario": scenario,

				"test-created-at": otf.timeBase.Format(time.RFC3339),
			},
		},

		Spec: nephranv1.NetworkIntentSpec{
			Intent: intent,

			IntentType: nephranv1.IntentTypeDeployment,

			Priority: nephranv1.NetworkPriorityNormal,

			TargetComponents: []nephranv1.NetworkTargetComponent{
				nephranv1.NetworkTargetComponentAMF,
				nephranv1.NetworkTargetComponentSMF,
			},
		},
	}
}

// CreateE2NodeSet creates a test E2NodeSet with comprehensive configuration.

func (otf *ORANTestFactory) CreateE2NodeSet(scenario string, replicas int32) *nephranv1.E2NodeSet {
	var ranFunctions []nephranv1.RANFunction

	var simulationConfig *nephranv1.SimulationConfig

	var ricConfig *nephranv1.RICConfiguration

	switch scenario {

	case "kpm-testing":

		ranFunctions = []nephranv1.RANFunction{
			{
				FunctionID: 1,

				Revision: 2,

				Description: "KPM Service Model v2.0",

				OID: "1.3.6.1.4.1.53148.1.1.2.2",
			},
		}

		simulationConfig = &nephranv1.SimulationConfig{
			UECount: 500,

			TrafficGeneration: true,

			MetricsInterval: "15s",

			TrafficProfile: nephranv1.TrafficProfileHigh,
		}

	case "rc-testing":

		ranFunctions = []nephranv1.RANFunction{
			{
				FunctionID: 2,

				Revision: 1,

				Description: "RAN Control Service Model v1.0",

				OID: "1.3.6.1.4.1.53148.1.1.2.3",
			},
		}

		simulationConfig = &nephranv1.SimulationConfig{
			UECount: 200,

			TrafficGeneration: true,

			MetricsInterval: "10s",

			TrafficProfile: nephranv1.TrafficProfileMedium,
		}

	case "multi-service-model":

		ranFunctions = []nephranv1.RANFunction{
			{
				FunctionID: 1,

				Revision: 2,

				Description: "KPM Service Model v2.0",

				OID: "1.3.6.1.4.1.53148.1.1.2.2",
			},

			{
				FunctionID: 2,

				Revision: 1,

				Description: "RAN Control Service Model v1.0",

				OID: "1.3.6.1.4.1.53148.1.1.2.3",
			},

			{
				FunctionID: 3,

				Revision: 1,

				Description: "Network Information Service Model v1.0",

				OID: "1.3.6.1.4.1.53148.1.1.2.4",
			},
		}

		simulationConfig = &nephranv1.SimulationConfig{
			UECount: 1000,

			TrafficGeneration: true,

			MetricsInterval: "30s",

			TrafficProfile: nephranv1.TrafficProfileBurst,
		}

	default:

		ranFunctions = []nephranv1.RANFunction{
			{
				FunctionID: 1,

				Revision: 1,

				Description: "Basic KPM Service Model",

				OID: "1.3.6.1.4.1.53148.1.1.2.2",
			},
		}

		simulationConfig = &nephranv1.SimulationConfig{
			UECount: 100,

			TrafficGeneration: false,

			MetricsInterval: "60s",

			TrafficProfile: nephranv1.TrafficProfileLow,
		}

	}

	ricConfig = &nephranv1.RICConfiguration{
		RICEndpoint: "http://near-rt-ric:38080",

		ConnectionTimeout: "30s",

		HeartbeatInterval: "10s",

		RetryConfig: &nephranv1.RetryConfig{
			MaxAttempts: 3,

			BackoffInterval: "5s",
		},
	}

	return &nephranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: otf.GetNextName("e2nodeset-" + scenario),

			Namespace: "default",

			Labels: map[string]string{
				"test-type": "e2-nodeset",

				"oran-interface": "e2",

				"test-scenario": scenario,
			},

			Annotations: map[string]string{
				"test-created-at": otf.timeBase.Format(time.RFC3339),
			},
		},

		Spec: nephranv1.E2NodeSetSpec{
			Replicas: replicas,

			Template: nephranv1.E2NodeTemplate{
				Spec: nephranv1.E2NodeSpec{
					NodeID: otf.GetNextName("test-gnb"),

					E2InterfaceVersion: "v2.0",

					SupportedRANFunctions: ranFunctions,
				},
			},

			SimulationConfig: simulationConfig,

			RICConfiguration: ricConfig,
		},
	}
}

// CreateE2Subscription creates a test E2 subscription.

func (otf *ORANTestFactory) CreateE2Subscription(serviceModel, nodeID string) *E2Subscription {
	var eventTrigger map[string]interface{}

	var actions []E2Action

	switch serviceModel {

	case "KPM":

		eventTrigger = json.RawMessage(`{}`)

		actions = []E2Action{
			{
				ActionID: 1,

				ActionType: "REPORT",

				Definition: json.RawMessage(`{}`),
				},
			},

			{
				ActionID: 2,

				ActionType: "REPORT",

				Definition: json.RawMessage(`{}`),
			},
		}

	case "RC":

		eventTrigger = json.RawMessage(`{}`)

		actions = []E2Action{
			{
				ActionID: 1,

				ActionType: "CONTROL",

				Definition: json.RawMessage(`{}`){
						"5qi": 1,

						"arp": 1,
					},
				},
			},
		}

	case "NI":

		eventTrigger = json.RawMessage(`{}`)

		actions = []E2Action{
			{
				ActionID: 1,

				ActionType: "REPORT",

				Definition: json.RawMessage(`{}`),
			},
		}

	default:

		eventTrigger = json.RawMessage(`{}`)

		actions = []E2Action{
			{
				ActionID: 1,

				ActionType: "REPORT",

				Definition: json.RawMessage(`{}`),
			},
		}

	}

	return &E2Subscription{
		SubscriptionID: otf.GetNextName("e2-sub"),

		NodeID: nodeID,

		ServiceModel: serviceModel,

		EventTrigger: eventTrigger,

		Actions: actions,

		Status: "ACTIVE",

		CreatedAt: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Second),
	}
}

// CreateE2Node creates a test E2 node.

func (otf *ORANTestFactory) CreateE2Node(nodeType string) *E2Node {
	var supportedModels []string

	var capabilities map[string]interface{}

	switch nodeType {

	case "gnodeb":

		supportedModels = []string{"KPM", "RC", "NI"}

		capabilities = json.RawMessage(`{}`),

			"mimo": "8x8",

			"carrierAggregation": true,
		}

	case "enb":

		supportedModels = []string{"KPM"}

		capabilities = json.RawMessage(`{}`),

			"mimo": "4x4",
		}

	case "ng-enb":

		supportedModels = []string{"KPM", "RC"}

		capabilities = json.RawMessage(`{}`),

			"mimo": "4x4",

			"nsa": true,
		}

	default:

		supportedModels = []string{"KPM"}

		capabilities = json.RawMessage(`{}`)

	}

	plmns := []PLMNID{
		{MCC: "001", MNC: "01"},

		{MCC: "001", MNC: "02"},
	}

	return &E2Node{
		NodeID: otf.GetNextName("node-" + nodeType),

		NodeType: nodeType,

		PLMNs: plmns,

		SupportedModels: supportedModels,

		Status: "CONNECTED",

		LastHeartbeat: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Second),

		Capabilities: capabilities,
	}
}

// O1 Interface Test Factories.

// CreateO1FCAPSIntent creates a NetworkIntent for O1 FCAPS testing.

func (otf *ORANTestFactory) CreateO1FCAPSIntent(scenario string) *nephranv1.NetworkIntent {
	var intent string

	var labels map[string]string

	switch scenario {

	case "fault-management":

		intent = "Configure fault management for AMF with critical alarm monitoring and automatic incident creation"

		labels = map[string]string{
			"test-type": "o1-fault-mgmt",

			"oran-interface": "o1",

			"fcaps-category": "fault",
		}

	case "configuration-management":

		intent = "Setup configuration management for UPF with NETCONF interface and YANG model validation"

		labels = map[string]string{
			"test-type": "o1-config-mgmt",

			"oran-interface": "o1",

			"fcaps-category": "configuration",
		}

	case "performance-management":

		intent = "Enable performance monitoring for SMF with KPI collection every 15 minutes and SLA tracking"

		labels = map[string]string{
			"test-type": "o1-perf-mgmt",

			"oran-interface": "o1",

			"fcaps-category": "performance",
		}

	case "security-management":

		intent = "Configure security management for 5G Core with certificate automation and access control"

		labels = map[string]string{
			"test-type": "o1-security-mgmt",

			"oran-interface": "o1",

			"fcaps-category": "security",
		}

	default:

		intent = "Setup basic O1 management interface for network function monitoring"

		labels = map[string]string{
			"test-type": "o1-basic-mgmt",

			"oran-interface": "o1",

			"fcaps-category": "basic",
		}

	}

	return &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name: otf.GetNextName("o1-fcaps-intent"),

			Namespace: "default",

			Labels: labels,

			Annotations: map[string]string{
				"test-scenario": scenario,

				"test-created-at": otf.timeBase.Format(time.RFC3339),
			},
		},

		Spec: nephranv1.NetworkIntentSpec{
			Intent: intent,

			IntentType: nephranv1.IntentTypeOptimization,

			Priority: nephranv1.NetworkPriorityNormal,

			TargetComponents: []nephranv1.NetworkTargetComponent{
				nephranv1.NetworkTargetComponentSMF,
				nephranv1.NetworkTargetComponentAMF,
				nephranv1.NetworkTargetComponentSMF,
				nephranv1.NetworkTargetComponentUPF,
			},
		},
	}
}

// CreateManagedElement creates a test managed element for O1 testing.

func (otf *ORANTestFactory) CreateManagedElement(elementType string) *ManagedElement {
	var configuration map[string]interface{}

	switch elementType {

	case "AMF":

		configuration = json.RawMessage(`{}`){
				"enabled": true,

				"severity": []string{"CRITICAL", "MAJOR", "MINOR"},

				"alertTargets": []string{"smo@example.com"},
			},

			"performanceMonitoring": json.RawMessage(`{}`),
			},
		}

	case "SMF":

		configuration = json.RawMessage(`{}`){
				"maxSessions": 100000,

				"sessionTimeout": "300s",

				"retryAttempts": 3,
			},

			"performanceMonitoring": json.RawMessage(`{}`),
			},
		}

	case "UPF":

		configuration = json.RawMessage(`{}`){
				"maxThroughput": "100Gbps",

				"bufferSize": "1GB",

				"qosSupport": true,
			},

			"performanceMonitoring": json.RawMessage(`{}`),
			},
		}

	default:

		configuration = json.RawMessage(`{}`){
				"enabled": true,

				"interval": "60s",
			},
		}

	}

	return &ManagedElement{
		ElementID: otf.GetNextName("element-" + elementType),

		ElementType: elementType,

		Configuration: configuration,

		Status: "ACTIVE",

		LastSync: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Second),
	}
}

// CreateO1Configuration creates a test O1 configuration.

func (otf *ORANTestFactory) CreateO1Configuration(configType, elementID string) *O1Configuration {
	var configData map[string]interface{}

	switch configType {

	case "FCAPS":

		configData = json.RawMessage(`{}`){
				"alarmSeverityFilter": []string{"CRITICAL", "MAJOR"},

				"autoAcknowledge": false,

				"notificationTargets": []string{"http://smo.example.com/alarms"},
			},

			"configurationManagement": json.RawMessage(`{}`),

			"performanceManagement": json.RawMessage(`{}`),
		}

	case "SECURITY":

		configData = json.RawMessage(`{}`){
				"enabled": true,

				"method": "certificate",

				"keySize": 2048,
			},

			"authorization": json.RawMessage(`{}`),

				"rbac": true,
			},

			"encryption": json.RawMessage(`{}`),
			},
		}

	case "PERFORMANCE":

		configData = json.RawMessage(`{}`),

			"thresholds": json.RawMessage(`{}`),

			"reporting": json.RawMessage(`{}`),
		}

	default:

		configData = json.RawMessage(`{}`){
				"enabled": true,

				"interval": "60s",
			},
		}

	}

	return &O1Configuration{
		ConfigID: otf.GetNextName("config-" + configType),

		ElementID: elementID,

		ConfigType: configType,

		ConfigData: configData,

		Version: 1,

		AppliedAt: otf.timeBase.Add(time.Duration(otf.nameCounter) * time.Minute),
	}
}

// O2 Interface Test Factories.

// CreateO2CloudInfraIntent creates a NetworkIntent for O2 cloud infrastructure testing.

func (otf *ORANTestFactory) CreateO2CloudInfraIntent(scenario string) *nephranv1.NetworkIntent {
	var intent string

	var labels map[string]string

	switch scenario {

	case "multi-cloud-deployment":

		intent = "Provision multi-cloud infrastructure across AWS, Azure, and GCP for 5G Core redundancy"

		labels = map[string]string{
			"test-type": "o2-multi-cloud",

			"oran-interface": "o2",

			"deployment-scope": "multi-cloud",
		}

	case "edge-cloud-deployment":

		intent = "Deploy edge cloud infrastructure for UPF with low-latency requirements and local breakout"

		labels = map[string]string{
			"test-type": "o2-edge-cloud",

			"oran-interface": "o2",

			"deployment-scope": "edge",
		}

	case "hybrid-cloud-deployment":

		intent = "Setup hybrid cloud deployment with private cloud for core functions and public cloud for scaling"

		labels = map[string]string{
			"test-type": "o2-hybrid-cloud",

			"oran-interface": "o2",

			"deployment-scope": "hybrid",
		}

	case "container-orchestration":

		intent = "Deploy Kubernetes clusters for containerized network functions with auto-scaling and service mesh"

		labels = map[string]string{
			"test-type": "o2-container-orch",

			"oran-interface": "o2",

			"deployment-scope": "containers",
		}

	default:

		intent = "Provision basic cloud infrastructure for network function deployment"

		labels = map[string]string{
			"test-type": "o2-basic-cloud",

			"oran-interface": "o2",

			"deployment-scope": "basic",
		}

	}

	return &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name: otf.GetNextName("o2-cloud-intent"),

			Namespace: "default",

			Labels: labels,

			Annotations: map[string]string{
				"test-scenario": scenario,

				"test-created-at": otf.timeBase.Format(time.RFC3339),
			},
		},

		Spec: nephranv1.NetworkIntentSpec{
			Intent: intent,

			IntentType: nephranv1.IntentTypeDeployment,

			Priority: nephranv1.NetworkPriorityHigh,

			TargetComponents: []nephranv1.NetworkTargetComponent{
				nephranv1.NetworkTargetComponentAMF,
				nephranv1.NetworkTargetComponentSMF,
				nephranv1.NetworkTargetComponentUPF,
			},
		},
	}
}

// Performance Benchmark Factories.

// CreatePerformanceBenchmarkData creates test data for performance benchmarking.

func (otf *ORANTestFactory) CreatePerformanceBenchmarkData() map[string]*InterfaceMetrics {
	return map[string]*InterfaceMetrics{
		"A1": {
			RequestCount: 1000,

			SuccessCount: 985,

			FailureCount: 15,

			AverageLatency: 45 * time.Millisecond,

			P95Latency: 89 * time.Millisecond,

			ThroughputRPS: 50.2,

			ErrorRate: 1.5,

			LastTestTime: time.Now(),
		},

		"E2": {
			RequestCount: 2500,

			SuccessCount: 2465,

			FailureCount: 35,

			AverageLatency: 25 * time.Millisecond,

			P95Latency: 48 * time.Millisecond,

			ThroughputRPS: 125.8,

			ErrorRate: 1.4,

			LastTestTime: time.Now(),
		},

		"O1": {
			RequestCount: 800,

			SuccessCount: 792,

			FailureCount: 8,

			AverageLatency: 120 * time.Millisecond,

			P95Latency: 245 * time.Millisecond,

			ThroughputRPS: 12.5,

			ErrorRate: 1.0,

			LastTestTime: time.Now(),
		},

		"O2": {
			RequestCount: 150,

			SuccessCount: 148,

			FailureCount: 2,

			AverageLatency: 2500 * time.Millisecond,

			P95Latency: 4800 * time.Millisecond,

			ThroughputRPS: 2.1,

			ErrorRate: 1.3,

			LastTestTime: time.Now(),
		},
	}
}

// CreateServiceModels creates test service models for E2 interface.

func (otf *ORANTestFactory) CreateServiceModels() []*ServiceModel {
	return []*ServiceModel{
		{
			ModelName: "KPM",

			Version: "2.0",

			OID: "1.3.6.1.4.1.53148.1.1.2.2",

			Functions: []string{"REPORT", "INSERT"},

			Capabilities: json.RawMessage(`{}`),

				"granularityPeriods": []int{100, 1000, 10000},

				"maxReports": 1000,
			},
		},

		{
			ModelName: "RC",

			Version: "1.0",

			OID: "1.3.6.1.4.1.53148.1.1.2.3",

			Functions: []string{"CONTROL", "POLICY"},

			Capabilities: json.RawMessage(`{}`),

				"policyTypes": []string{
					"ADMISSION_CONTROL", "LOAD_BALANCING", "ENERGY_SAVING",
				},

				"maxControlActions": 100,
			},
		},

		{
			ModelName: "NI",

			Version: "1.0",

			OID: "1.3.6.1.4.1.53148.1.1.2.4",

			Functions: []string{"REPORT", "INSERT"},

			Capabilities: json.RawMessage(`{}`),

				"maxInformationReports": 500,
			},
		},

		{
			ModelName: "CCC",

			Version: "1.0",

			OID: "1.3.6.1.4.1.53148.1.1.2.5",

			Functions: []string{"CONTROL", "REPORT"},

			Capabilities: json.RawMessage(`{}`),

				"maxConcurrentConfigs": 50,
			},
		},
	}
}

// Reset resets the factory counters.

func (otf *ORANTestFactory) Reset() {
	otf.nameCounter = 1

	otf.timeBase = time.Now()
}

