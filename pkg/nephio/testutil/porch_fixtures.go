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

package testutil

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

const (
	// Default test values
	DefaultTestNamespace   = "test-namespace"
	DefaultTestRepository  = "test-repo"
	DefaultTestPackageName = "test-package"
	DefaultTestRevision    = "v1.0.0"
	DefaultTestBranch      = "main"
	DefaultTestURL         = "https://github.com/test/test-repo.git"
)

// TestFixture provides a complete test environment for Porch operations
type TestFixture struct {
	Client       client.Client
	PorchClient  *porch.Client
	Config       *porch.Config
	Context      context.Context
	Namespace    string
	Repositories map[string]*porch.Repository
	Packages     map[string]*porch.PackageRevision
	Workflows    map[string]*porch.Workflow
}

// NewTestFixture creates a new test fixture with default configuration
func NewTestFixture(ctx context.Context) *TestFixture {
	if ctx == nil {
		ctx = context.Background()
	}

	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)

	fakeClient := clientfake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	config := NewTestConfig()

	fixture := &TestFixture{
		Client:       fakeClient,
		Config:       config,
		Context:      ctx,
		Namespace:    DefaultTestNamespace,
		Repositories: make(map[string]*porch.Repository),
		Packages:     make(map[string]*porch.PackageRevision),
		Workflows:    make(map[string]*porch.Workflow),
	}

	return fixture
}

// NewTestConfig creates a test configuration for Porch
func NewTestConfig() *porch.Config {
	return &porch.Config{
		PorchConfig: &porch.PorchServiceConfig{
			Endpoint: "http://localhost:9080",
			Auth: &porch.AuthenticationConfig{
				Type: "none",
			},
			Timeout: 30 * time.Second,
			Retry: &porch.RetryConfig{
				MaxRetries:    3,
				InitialDelay:  1 * time.Second,
				BackoffFactor: 2.0,
			},
			CircuitBreaker: &porch.CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				Timeout:          60 * time.Second,
				HalfOpenMaxCalls: 3,
			},
			RateLimit: &porch.RateLimitConfig{
				Enabled:           true,
				RequestsPerSecond: 100,
				Burst:             10,
			},
		},
	}
}

// CreateTestRepository creates a test repository fixture
func (f *TestFixture) CreateTestRepository(name string, opts ...RepositoryOption) *porch.Repository {
	if name == "" {
		name = fmt.Sprintf("test-repo-%s", GenerateRandomString(8))
	}

	repo := &porch.Repository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.porch.kpt.dev/v1alpha1",
			Kind:       "Repository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace,
			UID:       types.UID(uuid.New().String()),
			Labels: map[string]string{
				"test.nephoran.com/fixture": "true",
			},
		},
		Spec: porch.RepositorySpec{
			Type:      "git",
			URL:       DefaultTestURL,
			Branch:    DefaultTestBranch,
			Directory: "",
			Capabilities: []string{
				"upstream",
				"deployment",
			},
		},
		Status: porch.RepositoryStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "RepositoryReady",
				},
			},
			Health: porch.RepositoryHealthHealthy,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(repo)
	}

	f.Repositories[name] = repo
	return repo
}

// RepositoryOption allows customization of test repositories
type RepositoryOption func(*porch.Repository)

// WithRepositoryType sets the repository type
func WithRepositoryType(repoType string) RepositoryOption {
	return func(repo *porch.Repository) {
		repo.Spec.Type = repoType
	}
}

// WithRepositoryURL sets the repository URL
func WithRepositoryURL(url string) RepositoryOption {
	return func(repo *porch.Repository) {
		repo.Spec.URL = url
	}
}

// WithRepositoryBranch sets the repository branch
func WithRepositoryBranch(branch string) RepositoryOption {
	return func(repo *porch.Repository) {
		repo.Spec.Branch = branch
	}
}

// WithRepositoryAuth sets authentication configuration
func WithRepositoryAuth(auth *porch.AuthConfig) RepositoryOption {
	return func(repo *porch.Repository) {
		repo.Spec.Auth = auth
	}
}

// WithRepositoryCondition adds a condition to the repository
func WithRepositoryCondition(condition metav1.Condition) RepositoryOption {
	return func(repo *porch.Repository) {
		repo.Status.Conditions = append(repo.Status.Conditions, condition)
	}
}

// CreateTestPackageRevision creates a test package revision fixture
func (f *TestFixture) CreateTestPackageRevision(name, revision string, opts ...PackageOption) *porch.PackageRevision {
	if name == "" {
		name = fmt.Sprintf("test-package-%s", GenerateRandomString(8))
	}
	if revision == "" {
		revision = DefaultTestRevision
	}

	pkg := &porch.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.kpt.dev/v1alpha1",
			Kind:       "PackageRevision",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", name, revision),
			Namespace: f.Namespace,
			UID:       types.UID(uuid.New().String()),
			Labels: map[string]string{
				"test.nephoran.com/fixture": "true",
				"porch.kpt.dev/package":     name,
				"porch.kpt.dev/revision":    revision,
			},
		},
		Spec: porch.PackageRevisionSpec{
			PackageName: name,
			Repository:  DefaultTestRepository,
			Revision:    revision,
			Lifecycle:   porch.PackageRevisionLifecycleDraft,
			Resources: []interface{}{
				map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":      "test-config",
						"namespace": f.Namespace,
					},
					"data": map[string]interface{}{
						"key": "value",
					},
				},
			},
		},
		Status: porch.PackageRevisionStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "PackageReady",
				},
			},
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(pkg)
	}

	key := fmt.Sprintf("%s@%s", name, revision)
	f.Packages[key] = pkg
	return pkg
}

// PackageOption allows customization of test packages
type PackageOption func(*porch.PackageRevision)

// WithPackageRepository sets the package repository
func WithPackageRepository(repository string) PackageOption {
	return func(pkg *porch.PackageRevision) {
		pkg.Spec.Repository = repository
	}
}

// WithPackageLifecycle sets the package lifecycle
func WithPackageLifecycle(lifecycle porch.PackageRevisionLifecycle) PackageOption {
	return func(pkg *porch.PackageRevision) {
		pkg.Spec.Lifecycle = lifecycle
	}
}

// WithPackageResource adds a resource to the package
func WithPackageResource(resource interface{}) PackageOption {
	return func(pkg *porch.PackageRevision) {
		pkg.Spec.Resources = append(pkg.Spec.Resources, resource)
	}
}

// WithPackageFunction adds a function to the package
func WithPackageFunction(function interface{}) PackageOption {
	return func(pkg *porch.PackageRevision) {
		pkg.Spec.Functions = append(pkg.Spec.Functions, function)
	}
}

// WithPackageCondition adds a condition to the package
func WithPackageCondition(condition metav1.Condition) PackageOption {
	return func(pkg *porch.PackageRevision) {
		pkg.Status.Conditions = append(pkg.Status.Conditions, condition)
	}
}

// CreateTestWorkflow creates a test workflow fixture
func (f *TestFixture) CreateTestWorkflow(name string, opts ...WorkflowOption) *porch.Workflow {
	if name == "" {
		name = fmt.Sprintf("test-workflow-%s", GenerateRandomString(8))
	}

	workflow := &porch.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.kpt.dev/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace,
			UID:       types.UID(uuid.New().String()),
			Labels: map[string]string{
				"test.nephoran.com/fixture": "true",
			},
		},
		Spec: porch.WorkflowSpec{
			Stages: []porch.WorkflowStage{
				{
					Name: "validation",
					Type: porch.WorkflowStageTypeValidation,
					Actions: []porch.WorkflowAction{
						{
							Type:   "validate",
							Config: map[string]interface{}{},
						},
					},
				},
				{
					Name: "approval",
					Type: porch.WorkflowStageTypeApproval,
					Actions: []porch.WorkflowAction{
						{
							Type:   "approve",
							Config: map[string]interface{}{},
						},
					},
					Approvers: []porch.Approver{
						{
							Type: "user",
							Name: "admin",
						},
					},
				},
			},
		},
		Status: porch.WorkflowStatus{
			Phase: porch.WorkflowPhasePending,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(workflow)
	}

	f.Workflows[name] = workflow
	return workflow
}

// WorkflowOption allows customization of test workflows
type WorkflowOption func(*porch.Workflow)

// WithWorkflowStage adds a stage to the workflow
func WithWorkflowStage(stage porch.WorkflowStage) WorkflowOption {
	return func(workflow *porch.Workflow) {
		workflow.Spec.Stages = append(workflow.Spec.Stages, stage)
	}
}

// WithWorkflowApprover adds an approver to the workflow
func WithWorkflowApprover(approver porch.Approver) WorkflowOption {
	return func(workflow *porch.Workflow) {
		workflow.Spec.Approvers = append(workflow.Spec.Approvers, approver)
	}
}

// WithWorkflowPhase sets the workflow phase
func WithWorkflowPhase(phase porch.WorkflowPhase) WorkflowOption {
	return func(workflow *porch.Workflow) {
		workflow.Status.Phase = phase
	}
}

// CreateTestNetworkIntent creates a test NetworkIntent for testing
func (f *TestFixture) CreateTestNetworkIntent(name string, opts ...NetworkIntentOption) *v1.NetworkIntent {
	if name == "" {
		name = fmt.Sprintf("test-intent-%s", GenerateRandomString(8))
	}

	intent := &v1.NetworkIntent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "nephoran.com/v1",
			Kind:       "NetworkIntent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.Namespace,
			UID:       types.UID(uuid.New().String()),
			Labels: map[string]string{
				"test.nephoran.com/fixture": "true",
			},
		},
		Spec: v1.NetworkIntentSpec{
			Intent:     "Deploy 3 AMF instances in us-east-1 region",
			IntentType: v1.IntentTypeDeployment,
			Priority:   v1.PriorityMedium,
			TargetComponents: []v1.ORANComponent{
				v1.ORANComponentAMF,
			},
		},
		Status: v1.NetworkIntentStatus{
			Phase:       "processing",
			LastMessage: "Processing deployment intent",
			LastUpdateTime: metav1.Now(),
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(intent)
	}

	return intent
}

// NetworkIntentOption allows customization of test network intents
type NetworkIntentOption func(*v1.NetworkIntent)

// WithIntentType sets the intent type
func WithIntentType(intentType v1.IntentType) NetworkIntentOption {
	return func(intent *v1.NetworkIntent) {
		intent.Spec.IntentType = intentType
	}
}

// WithIntentPriority sets the intent priority
func WithIntentPriority(priority v1.Priority) NetworkIntentOption {
	return func(intent *v1.NetworkIntent) {
		intent.Spec.Priority = priority
	}
}

// WithTargetComponent adds a target component
func WithTargetComponent(component v1.ORANComponent) NetworkIntentOption {
	return func(intent *v1.NetworkIntent) {
		intent.Spec.TargetComponents = append(intent.Spec.TargetComponents, component)
	}
}

// WithIntent sets the intent text
func WithIntent(intentText string) NetworkIntentOption {
	return func(intent *v1.NetworkIntent) {
		intent.Spec.Intent = intentText
	}
}

// WithIntentPhase sets the intent phase
func WithIntentPhase(phase string) NetworkIntentOption {
	return func(intent *v1.NetworkIntent) {
		intent.Status.Phase = phase
	}
}

// Helper functions for test data generation

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateTestResource creates a test KRM resource
func GenerateTestResource(apiVersion, kind, name, namespace string) porch.KRMResource {
	return porch.KRMResource{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata: map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]string{
				"test.nephoran.com/generated": "true",
			},
		},
		Spec: map[string]interface{}{
			"testField": "testValue",
		},
	}
}

// GenerateTestFunction creates a test function configuration
func GenerateTestFunction(image string) porch.FunctionConfig {
	return porch.FunctionConfig{
		Image: image,
		ConfigMap: map[string]interface{}{
			"param1": "value1",
			"param2": "value2",
		},
	}
}

// GeneratePackageContents creates test package contents
func GeneratePackageContents() map[string][]byte {
	return map[string][]byte{
		"Kptfile": []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-package
info:
  description: Test package for Nephoran
`),
		"deployment.yaml": []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: test-container
        image: nginx:1.21
        ports:
        - containerPort: 80
`),
		"service.yaml": []byte(`
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  selector:
    app: test-app
  ports:
  - port: 80
    targetPort: 80
  type: ClusterIP
`),
	}
}

// Cleanup removes test resources and performs cleanup
func (f *TestFixture) Cleanup() {
	// Clear maps
	f.Repositories = make(map[string]*porch.Repository)
	f.Packages = make(map[string]*porch.PackageRevision)
	f.Workflows = make(map[string]*porch.Workflow)
}

// AssertCondition checks if a condition exists and has expected values
func AssertCondition(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus, reason string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType &&
			condition.Status == status &&
			condition.Reason == reason {
			return true
		}
	}
	return false
}

// WaitForCondition waits for a condition to be met within a timeout
func WaitForCondition(ctx context.Context, check func() bool, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if check() {
				return true
			}
		}
	}
}

// GetTestKubeConfig returns a test Kubernetes configuration
func GetTestKubeConfig() *rest.Config {
	return &rest.Config{
		Host: "https://localhost:6443",
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}
}

// CreateMultipleTestPackages creates multiple test packages for bulk testing
func (f *TestFixture) CreateMultipleTestPackages(count int, namePrefix string) []*porch.PackageRevision {
	packages := make([]*porch.PackageRevision, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i)
		revision := fmt.Sprintf("v1.0.%d", i)
		packages[i] = f.CreateTestPackageRevision(name, revision)
	}

	return packages
}

// CreateMultipleTestRepositories creates multiple test repositories for bulk testing
func (f *TestFixture) CreateMultipleTestRepositories(count int, namePrefix string) []*porch.Repository {
	repositories := make([]*porch.Repository, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i)
		repositories[i] = f.CreateTestRepository(name)
	}

	return repositories
}

// ValidatePackageRevision performs basic validation on a package revision
func ValidatePackageRevision(pkg *porch.PackageRevision) []string {
	var errors []string

	if pkg.Spec.PackageName == "" {
		errors = append(errors, "package name is required")
	}

	if pkg.Spec.Repository == "" {
		errors = append(errors, "repository is required")
	}

	if pkg.Spec.Revision == "" {
		errors = append(errors, "revision is required")
	}

	if !porch.IsValidLifecycle(pkg.Spec.Lifecycle) {
		errors = append(errors, "invalid lifecycle")
	}

	return errors
}

// ValidateRepository performs basic validation on a repository
func ValidateRepository(repo *porch.Repository) []string {
	var errors []string

	if repo.Spec.URL == "" {
		errors = append(errors, "repository URL is required")
	}

	if repo.Spec.Type == "" {
		errors = append(errors, "repository type is required")
	}

	validTypes := []string{"git", "oci"}
	typeValid := false
	for _, validType := range validTypes {
		if repo.Spec.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		errors = append(errors, "invalid repository type")
	}

	return errors
}
