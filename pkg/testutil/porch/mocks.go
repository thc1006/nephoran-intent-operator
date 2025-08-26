/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package porch

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// MockPorchClient implements the PorchClient interface for testing
type MockPorchClient struct {
	repositories    map[string]*porch.Repository
	packages        map[string]*porch.PackageRevision
	workflows       map[string]*porch.Workflow
	packageContents map[string]map[string][]byte
	mutex           sync.RWMutex

	// Behavior control flags
	simulateErrors     bool
	simulateLatency    time.Duration
	circuitBreakerOpen bool
	rateLimited        bool
	healthCheckFails   bool

	// Call tracking for assertions
	calls map[string]int
}

// NewMockPorchClient creates a new mock Porch client
func NewMockPorchClient() *MockPorchClient {
	return &MockPorchClient{
		repositories:    make(map[string]*porch.Repository),
		packages:        make(map[string]*porch.PackageRevision),
		workflows:       make(map[string]*porch.Workflow),
		packageContents: make(map[string]map[string][]byte),
		calls:           make(map[string]int),
	}
}

// Repository Operations

func (m *MockPorchClient) GetRepository(ctx context.Context, name string) (*porch.Repository, error) {
	m.trackCall("GetRepository")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if repo, exists := m.repositories[name]; exists {
		return m.cloneRepository(repo), nil
	}

	return nil, fmt.Errorf("repository %s not found", name)
}

func (m *MockPorchClient) ListRepositories(ctx context.Context, opts *porch.ListOptions) (*porch.RepositoryList, error) {
	m.trackCall("ListRepositories")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	items := make([]porch.Repository, 0, len(m.repositories))
	for _, repo := range m.repositories {
		if m.matchesListOptions(repo.ObjectMeta, opts) {
			items = append(items, *m.cloneRepository(repo))
		}
	}

	return &porch.RepositoryList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.porch.kpt.dev/v1alpha1",
			Kind:       "RepositoryList",
		},
		Items: items,
	}, nil
}

func (m *MockPorchClient) CreateRepository(ctx context.Context, repo *porch.Repository) (*porch.Repository, error) {
	m.trackCall("CreateRepository")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.repositories[repo.Name]; exists {
		return nil, fmt.Errorf("repository %s already exists", repo.Name)
	}

	created := m.cloneRepository(repo)
	created.Status.Conditions = []metav1.Condition{
		{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "RepositoryCreated",
			Message: "Repository created successfully",
		},
	}
	created.Status.Health = porch.RepositoryHealthHealthy
	now := metav1.Now()
	created.Status.LastSyncTime = &now

	m.repositories[repo.Name] = created
	return m.cloneRepository(created), nil
}

func (m *MockPorchClient) UpdateRepository(ctx context.Context, repo *porch.Repository) (*porch.Repository, error) {
	m.trackCall("UpdateRepository")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.repositories[repo.Name]; !exists {
		return nil, fmt.Errorf("repository %s not found", repo.Name)
	}

	updated := m.cloneRepository(repo)
	m.repositories[repo.Name] = updated
	return m.cloneRepository(updated), nil
}

func (m *MockPorchClient) DeleteRepository(ctx context.Context, name string) error {
	m.trackCall("DeleteRepository")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.repositories[name]; !exists {
		return fmt.Errorf("repository %s not found", name)
	}

	delete(m.repositories, name)
	return nil
}

func (m *MockPorchClient) SyncRepository(ctx context.Context, name string) error {
	m.trackCall("SyncRepository")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	repo, exists := m.repositories[name]
	if !exists {
		return fmt.Errorf("repository %s not found", name)
	}

	now := metav1.Now()
	repo.Status.LastSyncTime = &now
	repo.Status.SyncError = ""

	return nil
}

// PackageRevision Operations

func (m *MockPorchClient) GetPackageRevision(ctx context.Context, name string, revision string) (*porch.PackageRevision, error) {
	m.trackCall("GetPackageRevision")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	key := fmt.Sprintf("%s@%s", name, revision)
	if pkg, exists := m.packages[key]; exists {
		return m.clonePackageRevision(pkg), nil
	}

	return nil, fmt.Errorf("package revision %s@%s not found", name, revision)
}

func (m *MockPorchClient) ListPackageRevisions(ctx context.Context, opts *porch.ListOptions) (*porch.PackageRevisionList, error) {
	m.trackCall("ListPackageRevisions")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	items := make([]porch.PackageRevision, 0, len(m.packages))
	for _, pkg := range m.packages {
		if m.matchesListOptions(pkg.ObjectMeta, opts) {
			items = append(items, *m.clonePackageRevision(pkg))
		}
	}

	return &porch.PackageRevisionList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.kpt.dev/v1alpha1",
			Kind:       "PackageRevisionList",
		},
		Items: items,
	}, nil
}

func (m *MockPorchClient) CreatePackageRevision(ctx context.Context, pkg *porch.PackageRevision) (*porch.PackageRevision, error) {
	m.trackCall("CreatePackageRevision")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s@%s", pkg.Spec.PackageName, pkg.Spec.Revision)
	if _, exists := m.packages[key]; exists {
		return nil, fmt.Errorf("package revision %s already exists", key)
	}

	created := m.clonePackageRevision(pkg)
	created.Status.Conditions = []metav1.Condition{
		{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "PackageCreated",
			Message: "Package revision created successfully",
		},
	}

	m.packages[key] = created
	return m.clonePackageRevision(created), nil
}

func (m *MockPorchClient) UpdatePackageRevision(ctx context.Context, pkg *porch.PackageRevision) (*porch.PackageRevision, error) {
	m.trackCall("UpdatePackageRevision")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s@%s", pkg.Spec.PackageName, pkg.Spec.Revision)
	if _, exists := m.packages[key]; !exists {
		return nil, fmt.Errorf("package revision %s not found", key)
	}

	updated := m.clonePackageRevision(pkg)
	m.packages[key] = updated
	return m.clonePackageRevision(updated), nil
}

func (m *MockPorchClient) DeletePackageRevision(ctx context.Context, name string, revision string) error {
	m.trackCall("DeletePackageRevision")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s@%s", name, revision)
	if _, exists := m.packages[key]; !exists {
		return fmt.Errorf("package revision %s not found", key)
	}

	delete(m.packages, key)
	return nil
}

func (m *MockPorchClient) ApprovePackageRevision(ctx context.Context, name string, revision string) error {
	m.trackCall("ApprovePackageRevision")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s@%s", name, revision)
	pkg, exists := m.packages[key]
	if !exists {
		return fmt.Errorf("package revision %s not found", key)
	}

	// Validate lifecycle transition
	if !pkg.Spec.Lifecycle.CanTransitionTo(porch.PackageRevisionLifecyclePublished) {
		return fmt.Errorf("cannot transition from %s to Published", pkg.Spec.Lifecycle)
	}

	pkg.Spec.Lifecycle = porch.PackageRevisionLifecyclePublished
	now := metav1.Now()
	pkg.Status.PublishTime = &now

	return nil
}

func (m *MockPorchClient) ProposePackageRevision(ctx context.Context, name string, revision string) error {
	m.trackCall("ProposePackageRevision")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s@%s", name, revision)
	pkg, exists := m.packages[key]
	if !exists {
		return fmt.Errorf("package revision %s not found", key)
	}

	// Validate lifecycle transition
	if !pkg.Spec.Lifecycle.CanTransitionTo(porch.PackageRevisionLifecycleProposed) {
		return fmt.Errorf("cannot transition from %s to Proposed", pkg.Spec.Lifecycle)
	}

	pkg.Spec.Lifecycle = porch.PackageRevisionLifecycleProposed
	return nil
}

func (m *MockPorchClient) RejectPackageRevision(ctx context.Context, name string, revision string, reason string) error {
	m.trackCall("RejectPackageRevision")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s@%s", name, revision)
	pkg, exists := m.packages[key]
	if !exists {
		return fmt.Errorf("package revision %s not found", key)
	}

	pkg.Spec.Lifecycle = porch.PackageRevisionLifecycleDraft
	pkg.Status.Conditions = append(pkg.Status.Conditions, metav1.Condition{
		Type:    "Rejected",
		Status:  metav1.ConditionTrue,
		Reason:  "PackageRejected",
		Message: reason,
	})

	return nil
}

// Package Content Operations

func (m *MockPorchClient) GetPackageContents(ctx context.Context, name string, revision string) (map[string][]byte, error) {
	m.trackCall("GetPackageContents")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	key := fmt.Sprintf("%s@%s", name, revision)
	if contents, exists := m.packageContents[key]; exists {
		// Deep copy the contents
		result := make(map[string][]byte)
		for k, v := range contents {
			result[k] = make([]byte, len(v))
			copy(result[k], v)
		}
		return result, nil
	}

	// Return default contents if none exist
	return GeneratePackageContents(), nil
}

func (m *MockPorchClient) UpdatePackageContents(ctx context.Context, name string, revision string, contents map[string][]byte) error {
	m.trackCall("UpdatePackageContents")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s@%s", name, revision)

	// Check if package exists
	if _, exists := m.packages[key]; !exists {
		return fmt.Errorf("package revision %s not found", key)
	}

	// Deep copy the contents
	copied := make(map[string][]byte)
	for k, v := range contents {
		copied[k] = make([]byte, len(v))
		copy(copied[k], v)
	}

	m.packageContents[key] = copied
	return nil
}

func (m *MockPorchClient) RenderPackage(ctx context.Context, name string, revision string) (*porch.RenderResult, error) {
	m.trackCall("RenderPackage")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	// Simulate rendering by returning some test resources
	resources := []porch.KRMResource{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Metadata: map[string]interface{}{
				"name":      "rendered-deployment",
				"namespace": "default",
			},
		},
		{
			APIVersion: "v1",
			Kind:       "Service",
			Metadata: map[string]interface{}{
				"name":      "rendered-service",
				"namespace": "default",
			},
		},
	}

	return &porch.RenderResult{
		Resources: resources,
		Results: []*porch.FunctionResult{
			{
				Message:  "Package rendered successfully",
				Severity: "info",
			},
		},
	}, nil
}

// Function Operations

func (m *MockPorchClient) RunFunction(ctx context.Context, req *porch.FunctionRequest) (*porch.FunctionResponse, error) {
	m.trackCall("RunFunction")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	// Simulate function execution by returning modified resources
	response := &porch.FunctionResponse{
		Resources: req.Resources, // Echo input resources
		Results: []*porch.FunctionResult{
			{
				Message:  fmt.Sprintf("Function %s executed successfully", req.FunctionConfig.Image),
				Severity: "info",
			},
		},
		Logs: []string{
			"Starting function execution",
			"Processing resources",
			"Function execution completed",
		},
	}

	return response, nil
}

func (m *MockPorchClient) ValidatePackage(ctx context.Context, name string, revision string) (*porch.ValidationResult, error) {
	m.trackCall("ValidatePackage")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	// Simulate validation - check if package exists
	key := fmt.Sprintf("%s@%s", name, revision)
	m.mutex.RLock()
	pkg, exists := m.packages[key]
	m.mutex.RUnlock()

	if !exists {
		return &porch.ValidationResult{
			Valid: false,
			Errors: []porch.ValidationError{
				{
					Path:     fmt.Sprintf("packages/%s", key),
					Message:  "Package not found",
					Severity: "error",
				},
			},
		}, nil
	}

	// Basic validation - simplified for testing
	var validationErrors []porch.ValidationError
	if pkg.Spec.PackageName == "" {
		validationErrors = append(validationErrors, porch.ValidationError{
			Path:     "spec.packageName",
			Message:  "package name is required",
			Severity: "error",
		})
	}
	if pkg.Spec.Repository == "" {
		validationErrors = append(validationErrors, porch.ValidationError{
			Path:     "spec.repository",
			Message:  "repository is required",
			Severity: "error",
		})
	}

	if len(validationErrors) > 0 {
		return &porch.ValidationResult{
			Valid:  false,
			Errors: validationErrors,
		}, nil
	}

	return &porch.ValidationResult{
		Valid: true,
	}, nil
}

// Workflow Operations

func (m *MockPorchClient) GetWorkflow(ctx context.Context, name string) (*porch.Workflow, error) {
	m.trackCall("GetWorkflow")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if workflow, exists := m.workflows[name]; exists {
		return m.cloneWorkflow(workflow), nil
	}

	return nil, fmt.Errorf("workflow %s not found", name)
}

func (m *MockPorchClient) ListWorkflows(ctx context.Context, opts *porch.ListOptions) (*porch.WorkflowList, error) {
	m.trackCall("ListWorkflows")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	items := make([]porch.Workflow, 0, len(m.workflows))
	for _, workflow := range m.workflows {
		if m.matchesListOptions(workflow.ObjectMeta, opts) {
			items = append(items, *m.cloneWorkflow(workflow))
		}
	}

	return &porch.WorkflowList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.kpt.dev/v1alpha1",
			Kind:       "WorkflowList",
		},
		Items: items,
	}, nil
}

func (m *MockPorchClient) CreateWorkflow(ctx context.Context, workflow *porch.Workflow) (*porch.Workflow, error) {
	m.trackCall("CreateWorkflow")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.workflows[workflow.Name]; exists {
		return nil, fmt.Errorf("workflow %s already exists", workflow.Name)
	}

	created := m.cloneWorkflow(workflow)
	created.Status.Phase = string(porch.WorkflowPhasePending)
	created.Status.Conditions = []metav1.Condition{
		{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "WorkflowCreated",
			Message: "Workflow created successfully",
		},
	}

	m.workflows[workflow.Name] = created
	return m.cloneWorkflow(created), nil
}

func (m *MockPorchClient) UpdateWorkflow(ctx context.Context, workflow *porch.Workflow) (*porch.Workflow, error) {
	m.trackCall("UpdateWorkflow")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.workflows[workflow.Name]; !exists {
		return nil, fmt.Errorf("workflow %s not found", workflow.Name)
	}

	updated := m.cloneWorkflow(workflow)
	m.workflows[workflow.Name] = updated
	return m.cloneWorkflow(updated), nil
}

func (m *MockPorchClient) DeleteWorkflow(ctx context.Context, name string) error {
	m.trackCall("DeleteWorkflow")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.workflows[name]; !exists {
		return fmt.Errorf("workflow %s not found", name)
	}

	delete(m.workflows, name)
	return nil
}

// Health and Status Operations

func (m *MockPorchClient) Health(ctx context.Context) (*porch.HealthStatus, error) {
	m.trackCall("Health")

	if m.healthCheckFails {
		return &porch.HealthStatus{
			Status:    "unhealthy",
			Timestamp: &metav1.Time{Time: time.Now()},
			Components: []porch.ComponentHealth{
				{
					Name:   "porch-server",
					Status: "unhealthy",
					Error:  "simulated health check failure",
				},
			},
		}, nil
	}

	return &porch.HealthStatus{
		Status:    "healthy",
		Timestamp: &metav1.Time{Time: time.Now()},
		Components: []porch.ComponentHealth{
			{
				Name:   "porch-server",
				Status: "healthy",
			},
		},
	}, nil
}

func (m *MockPorchClient) Version(ctx context.Context) (*porch.VersionInfo, error) {
	m.trackCall("Version")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	buildTime, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	return &porch.VersionInfo{
		Version:   "v1.0.0-test",
		GitCommit: "abcd1234",
		BuildTime: buildTime,
	}, nil
}

// Test utility methods

// SetSimulateErrors enables or disables error simulation
func (m *MockPorchClient) SetSimulateErrors(enable bool) {
	m.simulateErrors = enable
}

// SetSimulateLatency sets artificial latency for operations
func (m *MockPorchClient) SetSimulateLatency(duration time.Duration) {
	m.simulateLatency = duration
}

// SetCircuitBreakerOpen simulates circuit breaker being open
func (m *MockPorchClient) SetCircuitBreakerOpen(open bool) {
	m.circuitBreakerOpen = open
}

// SetRateLimited simulates rate limiting
func (m *MockPorchClient) SetRateLimited(limited bool) {
	m.rateLimited = limited
}

// SetHealthCheckFails simulates health check failures
func (m *MockPorchClient) SetHealthCheckFails(fails bool) {
	m.healthCheckFails = fails
}

// GetCallCount returns the number of times a method was called
func (m *MockPorchClient) GetCallCount(method string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.calls[method]
}

// ResetCallCounts resets all call counters
func (m *MockPorchClient) ResetCallCounts() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.calls = make(map[string]int)
}

// AddRepository adds a repository to the mock data
func (m *MockPorchClient) AddRepository(repo *porch.Repository) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.repositories[repo.Name] = m.cloneRepository(repo)
}

// AddPackageRevision adds a package revision to the mock data
func (m *MockPorchClient) AddPackageRevision(pkg *porch.PackageRevision) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	key := fmt.Sprintf("%s@%s", pkg.Spec.PackageName, pkg.Spec.Revision)
	m.packages[key] = m.clonePackageRevision(pkg)
}

// AddWorkflow adds a workflow to the mock data
func (m *MockPorchClient) AddWorkflow(workflow *porch.Workflow) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.workflows[workflow.Name] = m.cloneWorkflow(workflow)
}

// Clear removes all mock data
func (m *MockPorchClient) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.repositories = make(map[string]*porch.Repository)
	m.packages = make(map[string]*porch.PackageRevision)
	m.workflows = make(map[string]*porch.Workflow)
	m.packageContents = make(map[string]map[string][]byte)
	m.calls = make(map[string]int)
}

// Helper methods

// Helper functions are defined in fixtures.go to avoid duplication

func (m *MockPorchClient) trackCall(method string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.calls[method]++
}

func (m *MockPorchClient) simulateConditions() error {
	if m.simulateLatency > 0 {
		time.Sleep(m.simulateLatency)
	}

	if m.circuitBreakerOpen {
		return fmt.Errorf("circuit breaker is open")
	}

	if m.rateLimited {
		return fmt.Errorf("rate limit exceeded")
	}

	if m.simulateErrors {
		return fmt.Errorf("simulated error")
	}

	return nil
}

func (m *MockPorchClient) matchesListOptions(meta metav1.ObjectMeta, opts *porch.ListOptions) bool {
	if opts == nil {
		return true
	}

	// Simple label selector matching
	if opts.LabelSelector != "" {
		// This is a simplified implementation
		// In real scenarios, you'd parse the selector properly
		return true
	}

	// Simple namespace filtering
	if opts.Namespace != "" && meta.Namespace != opts.Namespace {
		return false
	}

	return true
}

// Deep copy methods

func (m *MockPorchClient) cloneRepository(repo *porch.Repository) *porch.Repository {
	if repo == nil {
		return nil
	}
	return repo.DeepCopy()
}

func (m *MockPorchClient) clonePackageRevision(pkg *porch.PackageRevision) *porch.PackageRevision {
	if pkg == nil {
		return nil
	}
	// Since PackageRevision doesn't have DeepCopy in the actual types file,
	// we'll do a manual deep copy for the mock
	clone := *pkg

	// Deep copy resources
	if pkg.Spec.Resources != nil {
		clone.Spec.Resources = make([]porch.KRMResource, len(pkg.Spec.Resources))
		copy(clone.Spec.Resources, pkg.Spec.Resources)
	}

	// Deep copy functions
	if pkg.Spec.Functions != nil {
		clone.Spec.Functions = make([]porch.FunctionConfig, len(pkg.Spec.Functions))
		copy(clone.Spec.Functions, pkg.Spec.Functions)
	}

	// Deep copy conditions
	if pkg.Status.Conditions != nil {
		clone.Status.Conditions = make([]metav1.Condition, len(pkg.Status.Conditions))
		copy(clone.Status.Conditions, pkg.Status.Conditions)
	}

	return &clone
}

func (m *MockPorchClient) cloneWorkflow(workflow *porch.Workflow) *porch.Workflow {
	if workflow == nil {
		return nil
	}
	// Manual deep copy for workflow
	clone := *workflow

	// Deep copy stages
	if workflow.Spec.Stages != nil {
		clone.Spec.Stages = make([]porch.WorkflowStage, len(workflow.Spec.Stages))
		copy(clone.Spec.Stages, workflow.Spec.Stages)
	}

	// Deep copy approvers
	if workflow.Spec.Approvers != nil {
		clone.Spec.Approvers = make([]porch.Approver, len(workflow.Spec.Approvers))
		copy(clone.Spec.Approvers, workflow.Spec.Approvers)
	}

	// Deep copy conditions
	if workflow.Status.Conditions != nil {
		clone.Status.Conditions = make([]metav1.Condition, len(workflow.Status.Conditions))
		copy(clone.Status.Conditions, workflow.Status.Conditions)
	}

	return &clone
}
