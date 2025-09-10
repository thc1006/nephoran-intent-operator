package integration_tests

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thc1006/nephoran-intent-operator/hack/testtools"
)

// TestEnvironment provides access to envtest environment
var TestEnv *testtools.TestEnvironment

// Note: k8sClient and ctx are defined in suite_test.go as package-level globals

// CreateIntegrationTestNamespace creates a test namespace using envtest patterns for 2025 Go testing best practices
func CreateIntegrationTestNamespace() *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("test-integration-%d", time.Now().UnixNano()),
			Labels: map[string]string{
				"test-namespace":       "true",
				"nephoran.com/test":    "integration",
				"nephoran.com/envtest": "true",
			},
		},
	}

	// If we have a test environment, create the namespace in the cluster
	if TestEnv != nil && k8sClient != nil {
		if err := TestEnv.CreateTestObject(namespace); err != nil {
			// Fallback to returning the namespace object without creation
			return namespace
		}
	}

	return namespace
}

// SetupTestEnvironment initializes the test environment using envtest patterns
func SetupTestEnvironment() error {
	var err error

	// Use CI-optimized options if in CI, otherwise development options
	opts := testtools.GetRecommendedOptions()

	// Override some settings for integration tests
	opts.CRDDirectoryPaths = []string{
		"../../../deployments/crds",
		"../../deployments/crds",
		"deployments/crds",
		"crds",
	}

	TestEnv, err = testtools.SetupTestEnvironmentWithOptions(opts)
	if err != nil {
		return fmt.Errorf("failed to setup test environment: %w", err)
	}

	// Note: Global k8sClient and ctx should be set in suite_test.go
	
	return nil
}

// TeardownTestEnvironment cleans up the test environment
func TeardownTestEnvironment() {
	if TestEnv != nil {
		TestEnv.TeardownTestEnvironment()
	}
}

// E2EWorkflowTracker tracks the progress of E2E workflow tests
type E2EWorkflowTracker struct {
	workflows map[string]*WorkflowStatus
}

type WorkflowStatus struct {
	Name      string
	Type      string
	StartTime time.Time
	Phases    map[string]string
}

func NewE2EWorkflowTracker() *E2EWorkflowTracker {
	return &E2EWorkflowTracker{
		workflows: make(map[string]*WorkflowStatus),
	}
}

func (t *E2EWorkflowTracker) StartWorkflow(name, workflowType string) {
	t.workflows[name] = &WorkflowStatus{
		Name:      name,
		Type:      workflowType,
		StartTime: time.Now(),
		Phases:    make(map[string]string),
	}
}

func (t *E2EWorkflowTracker) RecordPhase(name, phase, status string) {
	if workflow, exists := t.workflows[name]; exists {
		workflow.Phases[phase] = status
	}
}
