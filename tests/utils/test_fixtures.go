package testutils

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// TestFixtures provides common test fixtures and utilities
type TestFixtures struct {
	Scheme    *runtime.Scheme
	Client    client.Client
	Namespace string
}

// NewTestFixtures creates a new test fixtures instance
func NewTestFixtures() *TestFixtures {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = nephoranv1.AddToScheme(scheme)

	return &TestFixtures{
		Scheme:    scheme,
		Client:    fake.NewClientBuilder().WithScheme(scheme).Build(),
		Namespace: "default",
	}
}

// NetworkIntentFixtures provides NetworkIntent test fixtures
type NetworkIntentFixtures struct{}

// CreateBasicNetworkIntent creates a basic NetworkIntent for testing
func (nif *NetworkIntentFixtures) CreateBasicNetworkIntent(name, namespace string) *nephoranv1.NetworkIntent {
	return &nephoranv1.NetworkIntent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "nephoran.com/v1",
			Kind:       "NetworkIntent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:     "Deploy a high-performance 5G AMF instance with auto-scaling enabled",
			Priority:   nephoranv1.PriorityHigh,
			MaxRetries: 3,
			Timeout:    metav1.Duration{Duration: 5 * time.Minute},
			Config: map[string]string{
				"deployment.replicas": "3",
				"scaling.enabled":     "true",
			},
		},
		Status: nephoranv1.NetworkIntentStatus{
			Phase:   nephoranv1.NetworkIntentPhasePending,
			Message: "Intent created",
			Conditions: []nephoranv1.NetworkIntentCondition{
				{
					Type:   nephoranv1.NetworkIntentConditionReady,
					Status: corev1.ConditionFalse,
					Reason: "Pending",
				},
			},
		},
	}
}

// CreateProcessingNetworkIntent creates a NetworkIntent in Processing phase
func (nif *NetworkIntentFixtures) CreateProcessingNetworkIntent(name, namespace string) *nephoranv1.NetworkIntent {
	intent := nif.CreateBasicNetworkIntent(name, namespace)
	intent.Status.Phase = nephoranv1.NetworkIntentPhaseProcessing
	intent.Status.Message = "Intent processing started"
	intent.Status.ProcessingPhase = "LLMProcessing"
	return intent
}

// CreateDeployedNetworkIntent creates a NetworkIntent in Deployed phase
func (nif *NetworkIntentFixtures) CreateDeployedNetworkIntent(name, namespace string) *nephoranv1.NetworkIntent {
	intent := nif.CreateBasicNetworkIntent(name, namespace)
	intent.Status.Phase = nephoranv1.NetworkIntentPhaseDeployed
	intent.Status.Message = "Intent successfully deployed"
	intent.Status.ProcessingPhase = "DeploymentVerification"
	intent.Status.Conditions = []nephoranv1.NetworkIntentCondition{
		{
			Type:   nephoranv1.NetworkIntentConditionReady,
			Status: corev1.ConditionTrue,
			Reason: "Deployed",
		},
	}
	return intent
}

// E2NodeSetFixtures provides E2NodeSet test fixtures
type E2NodeSetFixtures struct{}

// CreateBasicE2NodeSet creates a basic E2NodeSet for testing
func (enf *E2NodeSetFixtures) CreateBasicE2NodeSet(name, namespace string, replicas int32) *nephoranv1.E2NodeSet {
	return &nephoranv1.E2NodeSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "nephoran.com/v1",
			Kind:       "E2NodeSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas:           replicas,
			E2InterfaceVersion: "v3.0",
			RICEndpoint:        "http://ric-service:8080",
			HeartbeatInterval:  metav1.Duration{Duration: 30 * time.Second},
			SimulationConfig: nephoranv1.SimulationConfig{
				UECount:           100,
				TrafficGeneration: true,
				TrafficProfile:    "EMBB",
				MetricsInterval:   metav1.Duration{Duration: 30 * time.Second},
			},
			RANFunctions: []nephoranv1.RANFunction{
				{
					FunctionID:  1,
					Revision:    1,
					Description: "KPM Service Model",
					OID:         "1.3.6.1.4.1.53148.1.1.2.2",
				},
			},
		},
		Status: nephoranv1.E2NodeSetStatus{
			Phase:        nephoranv1.E2NodeSetPhasePending,
			ReadyNodes:   0,
			CurrentNodes: 0,
		},
	}
}

// CreateReadyE2NodeSet creates an E2NodeSet in Ready phase
func (enf *E2NodeSetFixtures) CreateReadyE2NodeSet(name, namespace string, replicas int32) *nephoranv1.E2NodeSet {
	nodeSet := enf.CreateBasicE2NodeSet(name, namespace, replicas)
	nodeSet.Status.Phase = nephoranv1.E2NodeSetPhaseReady
	nodeSet.Status.ReadyNodes = replicas
	nodeSet.Status.CurrentNodes = replicas
	nodeSet.Status.Conditions = []nephoranv1.E2NodeSetCondition{
		{
			Type:   nephoranv1.E2NodeSetConditionReady,
			Status: corev1.ConditionTrue,
			Reason: "AllNodesReady",
		},
	}
	return nodeSet
}

// CreateScalingE2NodeSet creates an E2NodeSet in Scaling phase
func (enf *E2NodeSetFixtures) CreateScalingE2NodeSet(name, namespace string, replicas int32, currentNodes int32) *nephoranv1.E2NodeSet {
	nodeSet := enf.CreateBasicE2NodeSet(name, namespace, replicas)
	nodeSet.Status.Phase = nephoranv1.E2NodeSetPhaseScaling
	nodeSet.Status.ReadyNodes = currentNodes
	nodeSet.Status.CurrentNodes = currentNodes
	nodeSet.Status.Conditions = []nephoranv1.E2NodeSetCondition{
		{
			Type:   nephoranv1.E2NodeSetConditionReady,
			Status: corev1.ConditionFalse,
			Reason: "Scaling",
		},
	}
	return nodeSet
}

// ConfigMapFixtures provides ConfigMap test fixtures
type ConfigMapFixtures struct{}

// CreateE2NodeConfigMap creates a ConfigMap for E2 node configuration
func (cmf *ConfigMapFixtures) CreateE2NodeConfigMap(name, namespace string, nodeIndex int) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"nephoran.com/e2-nodeset": "test-nodeset",
				"app":                     "e2-node-simulator",
				"nephoran.com/node-index": string(rune('0' + nodeIndex)),
			},
		},
		Data: map[string]string{
			"e2node-config.json": `{
				"nodeId": "test-node-` + string(rune('0'+nodeIndex)) + `",
				"e2InterfaceVersion": "v3.0",
				"ricEndpoint": "http://ric-service:8080",
				"ranFunctions": [
					{
						"functionId": 1,
						"revision": 1,
						"description": "KPM Service Model",
						"oid": "1.3.6.1.4.1.53148.1.1.2.2"
					}
				],
				"simulationConfig": {
					"ueCount": 100,
					"trafficGeneration": true,
					"metricsInterval": "30s",
					"trafficProfile": "EMBB"
				}
			}`,
			"e2node-status.json": `{
				"nodeId": "test-node-` + string(rune('0'+nodeIndex)) + `",
				"state": "connected",
				"lastHeartbeat": "` + time.Now().Format(time.RFC3339) + `",
				"activeSubscriptions": 0,
				"heartbeatCount": 1
			}`,
		},
	}
}

// Global fixture instances
var (
	NetworkIntentFixture = &NetworkIntentFixtures{}
	E2NodeSetFixture     = &E2NodeSetFixtures{}
	ConfigMapFixture     = &ConfigMapFixtures{}
)

// WaitForCondition is a helper function to wait for a condition
func WaitForCondition(ctx context.Context, condition func() bool, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return false
		case <-ticker.C:
			if condition() {
				return true
			}
		case <-ctx.Done():
			return false
		}
	}
}

// CreateTestContext creates a context with timeout for tests
func CreateTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
