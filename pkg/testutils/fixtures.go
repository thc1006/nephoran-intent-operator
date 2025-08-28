package testutils

import (
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkIntentFixture represents a test fixture for NetworkIntent resources.
type NetworkIntentFixture struct {
	Name      string
	Namespace string
	Intent    string
	Expected  nephoranv1.NetworkIntentSpec
}

// E2NodeSetFixture represents a test fixture for E2NodeSet resources.
type E2NodeSetFixture struct {
	Name      string
	Namespace string
	Replicas  int32
	Expected  nephoranv1.E2NodeSetStatus
}

// NetworkIntentFixtures provides common NetworkIntent test fixtures.
var NetworkIntentFixtures = []NetworkIntentFixture{
	{
		Name:      "upf-deployment",
		Namespace: "5g-core",
		Intent:    "Deploy UPF with 3 replicas for high availability",
		Expected: nephoranv1.NetworkIntentSpec{
			Intent: "Deploy UPF with 3 replicas for high availability",
		},
	},
	{
		Name:      "amf-scaling",
		Namespace: "5g-core",
		Intent:    "Scale AMF instances to 5 replicas to handle increased signaling load",
		Expected: nephoranv1.NetworkIntentSpec{
			Intent: "Scale AMF instances to 5 replicas to handle increased signaling load",
		},
	},
	{
		Name:      "near-rt-ric-deployment",
		Namespace: "o-ran",
		Intent:    "Set up Near-RT RIC with xApp support for intelligent traffic management",
		Expected: nephoranv1.NetworkIntentSpec{
			Intent: "Set up Near-RT RIC with xApp support for intelligent traffic management",
		},
	},
	{
		Name:      "edge-computing-node",
		Namespace: "edge-apps",
		Intent:    "Configure edge computing node with GPU acceleration for video processing",
		Expected: nephoranv1.NetworkIntentSpec{
			Intent: "Configure edge computing node with GPU acceleration for video processing",
		},
	},
	{
		Name:      "upf-vertical-scaling",
		Namespace: "5g-core",
		Intent:    "Increase UPF resources to 4 CPU cores and 8GB memory for high throughput",
		Expected: nephoranv1.NetworkIntentSpec{
			Intent: "Increase UPF resources to 4 CPU cores and 8GB memory for high throughput",
		},
	},
}

// E2NodeSetFixtures provides common E2NodeSet test fixtures.
var E2NodeSetFixtures = []E2NodeSetFixture{
	{
		Name:      "test-e2nodeset-small",
		Namespace: "default",
		Replicas:  1,
		Expected: nephoranv1.E2NodeSetStatus{
			ReadyReplicas: 1,
		},
	},
	{
		Name:      "test-e2nodeset-medium",
		Namespace: "o-ran",
		Replicas:  3,
		Expected: nephoranv1.E2NodeSetStatus{
			ReadyReplicas: 3,
		},
	},
	{
		Name:      "test-e2nodeset-large",
		Namespace: "o-ran",
		Replicas:  5,
		Expected: nephoranv1.E2NodeSetStatus{
			ReadyReplicas: 5,
		},
	},
	{
		Name:      "test-e2nodeset-ha",
		Namespace: "o-ran-ha",
		Replicas:  7,
		Expected: nephoranv1.E2NodeSetStatus{
			ReadyReplicas: 7,
		},
	},
}

// CreateNetworkIntent creates a NetworkIntent resource from a fixture.
func CreateNetworkIntent(fixture NetworkIntentFixture) *nephoranv1.NetworkIntent {
	return &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fixture.Name,
			Namespace: fixture.Namespace,
		},
		Spec: fixture.Expected,
		Status: nephoranv1.NetworkIntentStatus{
			Phase:       "Pending",
			LastMessage: "NetworkIntent has been created and is pending processing",
		},
	}
}

// CreateE2NodeSet creates an E2NodeSet resource from a fixture.
func CreateE2NodeSet(fixture E2NodeSetFixture) *nephoranv1.E2NodeSet {
	return &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fixture.Name,
			Namespace: fixture.Namespace,
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: fixture.Replicas,
		},
		Status: nephoranv1.E2NodeSetStatus{
			ReadyReplicas: 0, // Start with 0, will be updated by controller
		},
	}
}

// CreateProcessedNetworkIntent creates a NetworkIntent with processed status.
func CreateProcessedNetworkIntent(fixture NetworkIntentFixture) *nephoranv1.NetworkIntent {
	ni := CreateNetworkIntent(fixture)
	ni.Status.Phase = "Processed"
	ni.Status.LastMessage = "Intent has been successfully processed by LLM"
	return ni
}

// CreateDeployedNetworkIntent creates a NetworkIntent with deployed status.
func CreateDeployedNetworkIntent(fixture NetworkIntentFixture) *nephoranv1.NetworkIntent {
	ni := CreateProcessedNetworkIntent(fixture)
	ni.Status.Phase = "Deployed"
	ni.Status.LastMessage = "Resources have been successfully deployed via GitOps"
	return ni
}

// CreateReadyE2NodeSet creates an E2NodeSet with ready status.
func CreateReadyE2NodeSet(fixture E2NodeSetFixture) *nephoranv1.E2NodeSet {
	e2ns := CreateE2NodeSet(fixture)
	e2ns.Status.ReadyReplicas = fixture.Replicas
	return e2ns
}

// GetNetworkIntentByName returns a NetworkIntent fixture by name.
func GetNetworkIntentByName(name string) *NetworkIntentFixture {
	for _, fixture := range NetworkIntentFixtures {
		if fixture.Name == name {
			return &fixture
		}
	}
	return nil
}

// GetE2NodeSetByName returns an E2NodeSet fixture by name.
func GetE2NodeSetByName(name string) *E2NodeSetFixture {
	for _, fixture := range E2NodeSetFixtures {
		if fixture.Name == name {
			return &fixture
		}
	}
	return nil
}
