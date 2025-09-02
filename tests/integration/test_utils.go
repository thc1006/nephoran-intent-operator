package integration_tests

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateTestNamespace creates a test namespace for integration tests
func CreateTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("test-integration-%d", time.Now().UnixNano()),
			Labels: map[string]string{
				"test-namespace": "true",
			},
		},
	}
}

// Global variable ctx is usually provided by the test suite
var ctx = context.Background()

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