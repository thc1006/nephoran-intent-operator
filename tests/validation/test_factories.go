// Package validation provides comprehensive test data factories and fixtures
package validation

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// TestDataFactory provides factory methods for creating test data
type TestDataFactory struct {
	namespace string
	counter   int
}

// NewTestDataFactory creates a new test data factory
func NewTestDataFactory(namespace string) *TestDataFactory {
	return &TestDataFactory{
		namespace: namespace,
		counter:   0,
	}
}

// NetworkIntentFactory provides factory methods for NetworkIntent test objects
type NetworkIntentFactory struct {
	factory *TestDataFactory
}

// E2NodeSetFactory provides factory methods for E2NodeSet test objects
type E2NodeSetFactory struct {
	factory *TestDataFactory
}

// GetNetworkIntentFactory returns a NetworkIntent factory
func (tdf *TestDataFactory) GetNetworkIntentFactory() *NetworkIntentFactory {
	return &NetworkIntentFactory{factory: tdf}
}

// GetE2NodeSetFactory returns an E2NodeSet factory
func (tdf *TestDataFactory) GetE2NodeSetFactory() *E2NodeSetFactory {
	return &E2NodeSetFactory{factory: tdf}
}

// getUniqueName generates a unique test name
func (tdf *TestDataFactory) getUniqueName(prefix string) string {
	tdf.counter++
	return fmt.Sprintf("%s-test-%d-%d", prefix, tdf.counter, time.Now().Unix())
}

// NetworkIntent Factory Methods

// CreateBasicNetworkIntent creates a basic NetworkIntent for testing
func (nif *NetworkIntentFactory) CreateBasicNetworkIntent(name, intent string) *nephranv1.NetworkIntent {
	if name == "" {
		name = nif.factory.getUniqueName("basic-intent")
	}

	return &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nif.factory.namespace,
			Labels: map[string]string{
				"test-type":    "basic",
				"created-by":   "test-factory",
				"test-session": fmt.Sprintf("session-%d", time.Now().Unix()),
			},
			Annotations: map[string]string{
				"test.nephoran.io/purpose": "validation",
				"test.nephoran.io/created": time.Now().Format(time.RFC3339),
			},
		},
		Spec: nephranv1.NetworkIntentSpec{
			Intent: intent,
		},
		Status: nephranv1.NetworkIntentStatus{
			Phase: nephranv1.NetworkIntentPhasePending,
		},
	}
}

// CreateProcessingNetworkIntent creates a NetworkIntent in processing state
func (nif *NetworkIntentFactory) CreateProcessingNetworkIntent(name, intent string) *nephranv1.NetworkIntent {
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["test-type"] = "processing"
	ni.Status.Phase = nephranv1.NetworkIntentPhaseProcessing
	ni.Status.ProcessingStartTime = &metav1.Time{Time: time.Now().Add(-30 * time.Second)}

	return ni
}

// CreateDeployedNetworkIntent creates a NetworkIntent in deployed state
func (nif *NetworkIntentFactory) CreateDeployedNetworkIntent(name, intent string) *nephranv1.NetworkIntent {
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["test-type"] = "deployed"
	ni.Status.Phase = nephranv1.NetworkIntentPhaseCompleted
	now := metav1.Now()
	ni.Status.ProcessingStartTime = &metav1.Time{Time: now.Add(-2 * time.Minute)}
	ni.Status.CompletionTime = &now

	// Add processing results
	ni.Status.ProcessingResults = &nephranv1.ProcessingResult{
		LLMResponse:         "Successfully processed intent for AMF deployment",
		NetworkFunctionType: "AMF",
		DeploymentParameters: map[string]string{
			"replicas":   "3",
			"region":     "us-west-2",
			"ha-enabled": "true",
		},
	}

	return ni
}

// CreateFailedNetworkIntent creates a NetworkIntent in failed state
func (nif *NetworkIntentFactory) CreateFailedNetworkIntent(name, intent string, errorMsg string) *nephranv1.NetworkIntent {
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["test-type"] = "failed"
	ni.Status.Phase = nephranv1.NetworkIntentPhaseFailed
	now := metav1.Now()
	ni.Status.ProcessingStartTime = &metav1.Time{Time: now.Add(-1 * time.Minute)}
	ni.Status.CompletionTime = &now
	ni.Status.ErrorMessage = errorMsg

	return ni
}

// CreateAMFIntent creates a NetworkIntent specifically for AMF deployment
func (nif *NetworkIntentFactory) CreateAMFIntent(name string) *nephranv1.NetworkIntent {
	intent := "Deploy a high-availability AMF instance for production with auto-scaling"
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["network-function"] = "amf"
	ni.Labels["deployment-type"] = "production"

	return ni
}

// CreateSMFIntent creates a NetworkIntent specifically for SMF deployment
func (nif *NetworkIntentFactory) CreateSMFIntent(name string) *nephranv1.NetworkIntent {
	intent := "Deploy SMF instance for session management with UPF integration"
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["network-function"] = "smf"
	ni.Labels["integration"] = "upf"

	return ni
}

// CreateUPFIntent creates a NetworkIntent specifically for UPF deployment
func (nif *NetworkIntentFactory) CreateUPFIntent(name string) *nephranv1.NetworkIntent {
	intent := "Deploy UPF at edge location for low-latency data processing"
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["network-function"] = "upf"
	ni.Labels["deployment"] = "edge"

	return ni
}

// CreateNSSFIntent creates a NetworkIntent specifically for NSSF deployment
func (nif *NetworkIntentFactory) CreateNSSFIntent(name string) *nephranv1.NetworkIntent {
	intent := "Deploy NSSF for network slice selection with policy management"
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["network-function"] = "nssf"
	ni.Labels["capability"] = "slice-selection"

	return ni
}

// CreateSliceIntent creates a NetworkIntent for network slicing
func (nif *NetworkIntentFactory) CreateSliceIntent(name string, sliceType string) *nephranv1.NetworkIntent {
	intent := fmt.Sprintf("Create %s network slice with QoS requirements", sliceType)
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["intent-type"] = "slice"
	ni.Labels["slice-type"] = sliceType

	return ni
}

// CreateComplexIntent creates a NetworkIntent with complex requirements
func (nif *NetworkIntentFactory) CreateComplexIntent(name string) *nephranv1.NetworkIntent {
	intent := "Deploy 5G core network with AMF, SMF, UPF, and NSSF components across multiple regions with high availability, auto-scaling, and edge deployment"
	ni := nif.CreateBasicNetworkIntent(name, intent)
	ni.Labels["intent-type"] = "complex"
	ni.Labels["components"] = "multi"
	ni.Labels["deployment"] = "multi-region"

	return ni
}

// CreateBatchIntents creates multiple NetworkIntents for batch testing
func (nif *NetworkIntentFactory) CreateBatchIntents(count int, intentType string) []*nephranv1.NetworkIntent {
	intents := make([]*nephranv1.NetworkIntent, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("batch-%s-%d", intentType, i)

		switch intentType {
		case "amf":
			intents[i] = nif.CreateAMFIntent(name)
		case "smf":
			intents[i] = nif.CreateSMFIntent(name)
		case "upf":
			intents[i] = nif.CreateUPFIntent(name)
		case "nssf":
			intents[i] = nif.CreateNSSFIntent(name)
		default:
			intents[i] = nif.CreateBasicNetworkIntent(name, fmt.Sprintf("Batch test intent %d", i))
		}

		intents[i].Labels["batch-test"] = "true"
		intents[i].Labels["batch-id"] = fmt.Sprintf("batch-%d", time.Now().Unix())
	}

	return intents
}

// E2NodeSet Factory Methods

// CreateBasicE2NodeSet creates a basic E2NodeSet for testing
func (enf *E2NodeSetFactory) CreateBasicE2NodeSet(name string, replicas int32) *nephranv1.E2NodeSet {
	if name == "" {
		name = enf.factory.getUniqueName("basic-e2nodeset")
	}

	return &nephranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: enf.factory.namespace,
			Labels: map[string]string{
				"test-type":    "basic",
				"created-by":   "test-factory",
				"test-session": fmt.Sprintf("session-%d", time.Now().Unix()),
			},
			Annotations: map[string]string{
				"test.nephoran.io/purpose": "validation",
				"test.nephoran.io/created": time.Now().Format(time.RFC3339),
			},
		},
		Spec: nephranv1.E2NodeSetSpec{
			Replicas: replicas,
		},
		Status: nephranv1.E2NodeSetStatus{
			Replicas:      replicas,
			ReadyReplicas: 0,
		},
	}
}

// CreateReadyE2NodeSet creates an E2NodeSet in ready state
func (enf *E2NodeSetFactory) CreateReadyE2NodeSet(name string, replicas int32) *nephranv1.E2NodeSet {
	e2ns := enf.CreateBasicE2NodeSet(name, replicas)
	e2ns.Labels["test-type"] = "ready"
	e2ns.Status.ReadyReplicas = replicas

	// Add conditions
	e2ns.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "AllReplicasReady",
			Message:            fmt.Sprintf("All %d replicas are ready", replicas),
		},
	}

	return e2ns
}

// CreateScalingE2NodeSet creates an E2NodeSet in scaling state
func (enf *E2NodeSetFactory) CreateScalingE2NodeSet(name string, currentReplicas, targetReplicas int32) *nephranv1.E2NodeSet {
	e2ns := enf.CreateBasicE2NodeSet(name, targetReplicas)
	e2ns.Labels["test-type"] = "scaling"
	e2ns.Status.ReadyReplicas = currentReplicas

	// Add scaling condition
	e2ns.Status.Conditions = []metav1.Condition{
		{
			Type:               "Scaling",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "ScalingInProgress",
			Message:            fmt.Sprintf("Scaling from %d to %d replicas", currentReplicas, targetReplicas),
		},
	}

	return e2ns
}

// CreateLargeE2NodeSet creates an E2NodeSet for scalability testing
func (enf *E2NodeSetFactory) CreateLargeE2NodeSet(name string, replicas int32) *nephranv1.E2NodeSet {
	e2ns := enf.CreateBasicE2NodeSet(name, replicas)
	e2ns.Labels["test-type"] = "scalability"
	e2ns.Labels["scale-test"] = "large"

	return e2ns
}

// CreateBatchE2NodeSets creates multiple E2NodeSets for batch testing
func (enf *E2NodeSetFactory) CreateBatchE2NodeSets(count int, replicasPerSet int32) []*nephranv1.E2NodeSet {
	nodeSets := make([]*nephranv1.E2NodeSet, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("batch-e2nodeset-%d", i)
		nodeSets[i] = enf.CreateBasicE2NodeSet(name, replicasPerSet)
		nodeSets[i].Labels["batch-test"] = "true"
		nodeSets[i].Labels["batch-id"] = fmt.Sprintf("batch-%d", time.Now().Unix())
	}

	return nodeSets
}

// Test Scenario Factories

// TestScenarioFactory provides factory methods for complete test scenarios
type TestScenarioFactory struct {
	factory *TestDataFactory
}

// GetTestScenarioFactory returns a test scenario factory
func (tdf *TestDataFactory) GetTestScenarioFactory() *TestScenarioFactory {
	return &TestScenarioFactory{factory: tdf}
}

// CreateLatencyTestScenario creates objects for latency testing
func (tsf *TestScenarioFactory) CreateLatencyTestScenario(numIntents int) []*nephranv1.NetworkIntent {
	nif := tsf.factory.GetNetworkIntentFactory()
	intents := make([]*nephranv1.NetworkIntent, numIntents)

	intentTypes := []string{"amf", "smf", "upf", "nssf"}

	for i := 0; i < numIntents; i++ {
		intentType := intentTypes[i%len(intentTypes)]
		name := fmt.Sprintf("latency-test-%d", i)

		switch intentType {
		case "amf":
			intents[i] = nif.CreateAMFIntent(name)
		case "smf":
			intents[i] = nif.CreateSMFIntent(name)
		case "upf":
			intents[i] = nif.CreateUPFIntent(name)
		case "nssf":
			intents[i] = nif.CreateNSSFIntent(name)
		}

		intents[i].Labels["scenario"] = "latency-test"
		intents[i].Labels["test-order"] = fmt.Sprintf("%d", i)
	}

	return intents
}

// CreateThroughputTestScenario creates objects for throughput testing
func (tsf *TestScenarioFactory) CreateThroughputTestScenario(numIntents int) []*nephranv1.NetworkIntent {
	nif := tsf.factory.GetNetworkIntentFactory()
	intents := make([]*nephranv1.NetworkIntent, numIntents)

	for i := 0; i < numIntents; i++ {
		name := fmt.Sprintf("throughput-test-%d", i)
		intent := fmt.Sprintf("Deploy network function %d for throughput testing", i)
		intents[i] = nif.CreateBasicNetworkIntent(name, intent)
		intents[i].Labels["scenario"] = "throughput-test"
		intents[i].Labels["batch-size"] = fmt.Sprintf("%d", numIntents)
	}

	return intents
}

// CreateScalabilityTestScenario creates objects for scalability testing
func (tsf *TestScenarioFactory) CreateScalabilityTestScenario(maxConcurrency int) []*nephranv1.NetworkIntent {
	nif := tsf.factory.GetNetworkIntentFactory()
	intents := make([]*nephranv1.NetworkIntent, maxConcurrency)

	for i := 0; i < maxConcurrency; i++ {
		name := fmt.Sprintf("scale-test-%d", i)
		intent := fmt.Sprintf("Deploy scalability test function %d", i)
		intents[i] = nif.CreateBasicNetworkIntent(name, intent)
		intents[i].Labels["scenario"] = "scalability-test"
		intents[i].Labels["concurrency-level"] = fmt.Sprintf("%d", maxConcurrency)
	}

	return intents
}

// CreateReliabilityTestScenario creates objects for reliability testing
func (tsf *TestScenarioFactory) CreateReliabilityTestScenario() *ReliabilityTestScenario {
	nif := tsf.factory.GetNetworkIntentFactory()
	enf := tsf.factory.GetE2NodeSetFactory()

	return &ReliabilityTestScenario{
		NormalIntent:   nif.CreateAMFIntent("reliability-normal"),
		RestartIntent:  nif.CreateSMFIntent("reliability-restart"),
		FailoverIntent: nif.CreateUPFIntent("reliability-failover"),
		E2NodeSet:      enf.CreateReadyE2NodeSet("reliability-e2nodes", 5),
	}
}

// ReliabilityTestScenario contains objects for reliability testing
type ReliabilityTestScenario struct {
	NormalIntent   *nephranv1.NetworkIntent
	RestartIntent  *nephranv1.NetworkIntent
	FailoverIntent *nephranv1.NetworkIntent
	E2NodeSet      *nephranv1.E2NodeSet
}

// CreateSecurityTestScenario creates objects for security testing
func (tsf *TestScenarioFactory) CreateSecurityTestScenario() *SecurityTestScenario {
	nif := tsf.factory.GetNetworkIntentFactory()

	// Create intents with different security implications
	authIntent := nif.CreateBasicNetworkIntent("security-auth-test",
		"Deploy network function with OAuth2 authentication")
	authIntent.Labels["security-test"] = "authentication"

	encryptionIntent := nif.CreateBasicNetworkIntent("security-encryption-test",
		"Deploy network function with TLS encryption")
	encryptionIntent.Labels["security-test"] = "encryption"

	rbacIntent := nif.CreateBasicNetworkIntent("security-rbac-test",
		"Deploy network function with RBAC authorization")
	rbacIntent.Labels["security-test"] = "rbac"

	return &SecurityTestScenario{
		AuthenticationIntent: authIntent,
		EncryptionIntent:     encryptionIntent,
		RBACIntent:           rbacIntent,
	}
}

// SecurityTestScenario contains objects for security testing
type SecurityTestScenario struct {
	AuthenticationIntent *nephranv1.NetworkIntent
	EncryptionIntent     *nephranv1.NetworkIntent
	RBACIntent           *nephranv1.NetworkIntent
}

// Utility methods for test data manipulation

// ApplyTestLabels applies common test labels to any Kubernetes object
func (tdf *TestDataFactory) ApplyTestLabels(obj metav1.Object, testType string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels["test.nephoran.io/managed"] = "true"
	labels["test.nephoran.io/type"] = testType
	labels["test.nephoran.io/session"] = fmt.Sprintf("session-%d", time.Now().Unix())

	obj.SetLabels(labels)
}

// ApplyTestAnnotations applies common test annotations to any Kubernetes object
func (tdf *TestDataFactory) ApplyTestAnnotations(obj metav1.Object) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations["test.nephoran.io/created"] = time.Now().Format(time.RFC3339)
	annotations["test.nephoran.io/purpose"] = "validation"

	obj.SetAnnotations(annotations)
}

// CleanupSelector returns a label selector for cleaning up test objects
func (tdf *TestDataFactory) CleanupSelector() string {
	return "test.nephoran.io/managed=true"
}

// GetTestNamespace returns the namespace used for testing
func (tdf *TestDataFactory) GetTestNamespace() string {
	return tdf.namespace
}

// GenerateUID generates a unique identifier for test objects
func (tdf *TestDataFactory) GenerateUID() types.UID {
	return types.UID(fmt.Sprintf("test-uid-%d-%d", tdf.counter, time.Now().UnixNano()))
}
