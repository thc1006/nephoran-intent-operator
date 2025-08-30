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
	"math/rand"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
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

// Local type definitions to avoid import cycles

// Client represents a Porch client
type Client struct {
	Config *Config
}

// Config represents Porch configuration
type Config struct {
	PorchConfig *PorchConfiguration
}

// PorchConfiguration represents detailed Porch configuration
type PorchConfiguration struct {
	Endpoint       string                       `json:"endpoint"`
	Auth           *AuthConfiguration           `json:"auth,omitempty"`
	Timeout        *metav1.Duration             `json:"timeout,omitempty"`
	Retry          *RetryConfiguration          `json:"retry,omitempty"`
	CircuitBreaker *CircuitBreakerConfiguration `json:"circuit_breaker,omitempty"`
	RateLimit      *RateLimitConfiguration      `json:"rate_limit,omitempty"`
}

// AuthConfiguration represents authentication configuration
type AuthConfiguration struct {
	Type string `json:"type"`
}

// RetryConfiguration represents retry configuration
type RetryConfiguration struct {
	MaxRetries   int              `json:"max_retries"`
	BackoffDelay *metav1.Duration `json:"backoff_delay,omitempty"`
}

// CircuitBreakerConfiguration represents circuit breaker configuration
type CircuitBreakerConfiguration struct {
	Enabled          bool             `json:"enabled"`
	FailureThreshold int              `json:"failure_threshold"`
	Timeout          *metav1.Duration `json:"timeout,omitempty"`
	HalfOpenMaxCalls int              `json:"half_open_max_calls"`
}

// RateLimitConfiguration represents rate limit configuration
type RateLimitConfiguration struct {
	Enabled           bool `json:"enabled"`
	RequestsPerSecond int  `json:"requests_per_second"`
	Burst             int  `json:"burst"`
}

// Repository represents a Porch repository
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RepositorySpec   `json:"spec,omitempty"`
	Status            RepositoryStatus `json:"status,omitempty"`
}

// RepositorySpec defines the desired state of Repository
type RepositorySpec struct {
	Type         string      `json:"type"`
	URL          string      `json:"url"`
	Branch       string      `json:"branch,omitempty"`
	Directory    string      `json:"directory,omitempty"`
	Capabilities []string    `json:"capabilities,omitempty"`
	Auth         *AuthConfig `json:"auth,omitempty"`
}

// AuthConfig represents authentication configuration for repositories
type AuthConfig struct {
	Type   string `json:"type"`
	Secret string `json:"secret,omitempty"`
}

// RepositoryHealth represents repository health status
type RepositoryHealth string

const (
	RepositoryHealthHealthy   RepositoryHealth = "Healthy"
	RepositoryHealthUnhealthy RepositoryHealth = "Unhealthy"
)

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Health     RepositoryHealth   `json:"health,omitempty"`
}

// PackageRevision represents a package revision
type PackageRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PackageRevisionSpec   `json:"spec,omitempty"`
	Status            PackageRevisionStatus `json:"status,omitempty"`
}

// PackageRevisionLifecycle represents package lifecycle states
type PackageRevisionLifecycle string

const (
	PackageRevisionLifecycleDraft     PackageRevisionLifecycle = "Draft"
	PackageRevisionLifecycleProposed  PackageRevisionLifecycle = "Proposed"
	PackageRevisionLifecyclePublished PackageRevisionLifecycle = "Published"
)

// PackageRevisionSpec defines the desired state of PackageRevision
type PackageRevisionSpec struct {
	PackageName string                   `json:"packageName"`
	Repository  string                   `json:"repository"`
	Revision    string                   `json:"revision"`
	Lifecycle   PackageRevisionLifecycle `json:"lifecycle,omitempty"`
	Resources   []KRMResource            `json:"resources,omitempty"`
	Functions   []FunctionConfig         `json:"functions,omitempty"`
}

// KRMResource represents a Kubernetes Resource Model resource
type KRMResource struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Spec       map[string]interface{} `json:"spec,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// FunctionConfig represents a function configuration
type FunctionConfig struct {
	Image     string                 `json:"image"`
	ConfigMap map[string]interface{} `json:"configMap,omitempty"`
}

// PackageRevisionStatus defines the observed state of PackageRevision
type PackageRevisionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Workflow represents a workflow
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkflowSpec   `json:"spec,omitempty"`
	Status            WorkflowStatus `json:"status,omitempty"`
}

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	Stages    []WorkflowStage `json:"stages,omitempty"`
	Approvers []Approver      `json:"approvers,omitempty"`
}

// WorkflowStageType represents workflow stage types
type WorkflowStageType string

const (
	WorkflowStageTypeValidation WorkflowStageType = "validation"
	WorkflowStageTypeApproval   WorkflowStageType = "approval"
)

// WorkflowStage represents a workflow stage
type WorkflowStage struct {
	Name      string            `json:"name"`
	Type      WorkflowStageType `json:"type"`
	Actions   []WorkflowAction  `json:"actions,omitempty"`
	Approvers []Approver        `json:"approvers,omitempty"`
}

// WorkflowAction represents a workflow action
type WorkflowAction struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// Approver represents a workflow approver
type Approver struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// WorkflowPhase represents workflow phases
type WorkflowPhase string

const (
	WorkflowPhasePending   WorkflowPhase = "Pending"
	WorkflowPhaseRunning   WorkflowPhase = "Running"
	WorkflowPhaseCompleted WorkflowPhase = "Completed"
	WorkflowPhaseFailed    WorkflowPhase = "Failed"
)

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	Phase WorkflowPhase `json:"phase,omitempty"`
}

// IsValidLifecycle checks if a lifecycle value is valid
func IsValidLifecycle(lifecycle PackageRevisionLifecycle) bool {
	switch lifecycle {
	case PackageRevisionLifecycleDraft, PackageRevisionLifecycleProposed, PackageRevisionLifecyclePublished:
		return true
	default:
		return false
	}
}

// TestFixture provides a complete test environment for Porch operations
type TestFixture struct {
	Client       client.Client
	PorchClient  *Client
	Config       *Config
	Context      context.Context
	Namespace    string
	Repositories map[string]*Repository
	Packages     map[string]*PackageRevision
	Workflows    map[string]*Workflow
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
		Repositories: make(map[string]*Repository),
		Packages:     make(map[string]*PackageRevision),
		Workflows:    make(map[string]*Workflow),
	}

	return fixture
}

// NewTestConfig creates a test configuration for Porch
func NewTestConfig() *Config {
	return &Config{
		PorchConfig: &PorchConfiguration{
			Endpoint: "http://localhost:9080",
			Auth: &AuthConfiguration{
				Type: "none",
			},
			Timeout: &metav1.Duration{Duration: 30 * time.Second},
			Retry: &RetryConfiguration{
				MaxRetries:   3,
				BackoffDelay: &metav1.Duration{Duration: 1 * time.Second},
			},
			CircuitBreaker: &CircuitBreakerConfiguration{
				Enabled:          true,
				FailureThreshold: 5,
				Timeout:          &metav1.Duration{Duration: 60 * time.Second},
				HalfOpenMaxCalls: 3,
			},
			RateLimit: &RateLimitConfiguration{
				Enabled:           true,
				RequestsPerSecond: 100,
				Burst:             10,
			},
		},
	}
}

// CreateTestRepository creates a test repository fixture
func (f *TestFixture) CreateTestRepository(name string, opts ...RepositoryOption) *Repository {
	if name == "" {
		name = fmt.Sprintf("test-repo-%s", GenerateRandomString(8))
	}

	repo := &Repository{
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
		Spec: RepositorySpec{
			Type:      "git",
			URL:       DefaultTestURL,
			Branch:    DefaultTestBranch,
			Directory: "",
			Capabilities: []string{
				"upstream",
				"deployment",
			},
		},
		Status: RepositoryStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "RepositoryReady",
				},
			},
			Health: RepositoryHealthHealthy,
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
type RepositoryOption func(*Repository)

// WithRepositoryType sets the repository type
func WithRepositoryType(repoType string) RepositoryOption {
	return func(repo *Repository) {
		repo.Spec.Type = repoType
	}
}

// WithRepositoryURL sets the repository URL
func WithRepositoryURL(url string) RepositoryOption {
	return func(repo *Repository) {
		repo.Spec.URL = url
	}
}

// WithRepositoryBranch sets the repository branch
func WithRepositoryBranch(branch string) RepositoryOption {
	return func(repo *Repository) {
		repo.Spec.Branch = branch
	}
}

// WithRepositoryAuth sets authentication configuration
func WithRepositoryAuth(auth *AuthConfig) RepositoryOption {
	return func(repo *Repository) {
		repo.Spec.Auth = auth
	}
}

// WithRepositoryCondition adds a condition to the repository
func WithRepositoryCondition(condition metav1.Condition) RepositoryOption {
	return func(repo *Repository) {
		repo.Status.Conditions = append(repo.Status.Conditions, condition)
	}
}

// CreateTestPackageRevision creates a test package revision fixture
func (f *TestFixture) CreateTestPackageRevision(name, revision string, opts ...PackageOption) *PackageRevision {
	if name == "" {
		name = fmt.Sprintf("test-package-%s", GenerateRandomString(8))
	}
	if revision == "" {
		revision = DefaultTestRevision
	}

	pkg := &PackageRevision{
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
		Spec: PackageRevisionSpec{
			PackageName: name,
			Repository:  DefaultTestRepository,
			Revision:    revision,
			Lifecycle:   PackageRevisionLifecycleDraft,
			Resources: []KRMResource{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Metadata: map[string]interface{}{
						"name":      "test-config",
						"namespace": f.Namespace,
					},
					Data: map[string]interface{}{
						"key": "value",
					},
				},
			},
		},
		Status: PackageRevisionStatus{
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
type PackageOption func(*PackageRevision)

// WithPackageRepository sets the package repository
func WithPackageRepository(repository string) PackageOption {
	return func(pkg *PackageRevision) {
		pkg.Spec.Repository = repository
	}
}

// WithPackageLifecycle sets the package lifecycle
func WithPackageLifecycle(lifecycle PackageRevisionLifecycle) PackageOption {
	return func(pkg *PackageRevision) {
		pkg.Spec.Lifecycle = lifecycle
	}
}

// WithPackageResource adds a resource to the package
func WithPackageResource(resource KRMResource) PackageOption {
	return func(pkg *PackageRevision) {
		pkg.Spec.Resources = append(pkg.Spec.Resources, resource)
	}
}

// WithPackageFunction adds a function to the package
func WithPackageFunction(function FunctionConfig) PackageOption {
	return func(pkg *PackageRevision) {
		pkg.Spec.Functions = append(pkg.Spec.Functions, function)
	}
}

// WithPackageCondition adds a condition to the package
func WithPackageCondition(condition metav1.Condition) PackageOption {
	return func(pkg *PackageRevision) {
		pkg.Status.Conditions = append(pkg.Status.Conditions, condition)
	}
}

// CreateTestWorkflow creates a test workflow fixture
func (f *TestFixture) CreateTestWorkflow(name string, opts ...WorkflowOption) *Workflow {
	if name == "" {
		name = fmt.Sprintf("test-workflow-%s", GenerateRandomString(8))
	}

	workflow := &Workflow{
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
		Spec: WorkflowSpec{
			Stages: []WorkflowStage{
				{
					Name: "validation",
					Type: WorkflowStageTypeValidation,
					Actions: []WorkflowAction{
						{
							Type:   "validate",
							Config: map[string]interface{}{},
						},
					},
				},
				{
					Name: "approval",
					Type: WorkflowStageTypeApproval,
					Actions: []WorkflowAction{
						{
							Type:   "approve",
							Config: map[string]interface{}{},
						},
					},
					Approvers: []Approver{
						{
							Type: "user",
							Name: "admin",
						},
					},
				},
			},
		},
		Status: WorkflowStatus{
			Phase: WorkflowPhasePending,
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
type WorkflowOption func(*Workflow)

// WithWorkflowStage adds a stage to the workflow
func WithWorkflowStage(stage WorkflowStage) WorkflowOption {
	return func(workflow *Workflow) {
		workflow.Spec.Stages = append(workflow.Spec.Stages, stage)
	}
}

// WithWorkflowApprover adds an approver to the workflow
func WithWorkflowApprover(approver Approver) WorkflowOption {
	return func(workflow *Workflow) {
		workflow.Spec.Approvers = append(workflow.Spec.Approvers, approver)
	}
}

// WithWorkflowPhase sets the workflow phase
func WithWorkflowPhase(phase WorkflowPhase) WorkflowOption {
	return func(workflow *Workflow) {
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
			IntentType: v1.IntentTypeDeployment,
			Priority:   v1.NetworkPriorityNormal,
			TargetComponents: []v1.NetworkTargetComponent{
				v1.NetworkTargetComponentAMF,
			},
			Parameters: &runtime.RawExtension{
				Raw: []byte(`{"replicas":"3","region":"us-east-1"}`),
			},
		},
		Status: v1.NetworkIntentStatus{
			Phase: v1.NetworkIntentPhaseProcessing,
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionFalse,
					Reason: "Processing",
				},
			},
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
func WithIntentPriority(priority v1.NetworkPriority) NetworkIntentOption {
	return func(intent *v1.NetworkIntent) {
		intent.Spec.Priority = priority
	}
}

// WithTargetComponent adds a target component
func WithTargetComponent(component v1.NetworkTargetComponent) NetworkIntentOption {
	return func(intent *v1.NetworkIntent) {
		intent.Spec.TargetComponents = append(intent.Spec.TargetComponents, component)
	}
}

// WithIntentParameter adds a parameter to the intent
func WithIntentParameter(key, value string) NetworkIntentOption {
	return func(intent *v1.NetworkIntent) {
		params := make(map[string]interface{})
		if intent.Spec.Parameters != nil && intent.Spec.Parameters.Raw != nil {
			// Try to unmarshal existing parameters
			// For simplicity in tests, we'll just create a new map
		}
		params[key] = value
		rawParams := fmt.Sprintf(`{"%s":"%s"}`, key, value)
		intent.Spec.Parameters = &runtime.RawExtension{
			Raw: []byte(rawParams),
		}
	}
}

// WithIntentPhase sets the intent phase
func WithIntentPhase(phase v1.NetworkIntentPhase) NetworkIntentOption {
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
func GenerateTestResource(apiVersion, kind, name, namespace string) KRMResource {
	return KRMResource{
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
func GenerateTestFunction(image string) FunctionConfig {
	return FunctionConfig{
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
	f.Repositories = make(map[string]*Repository)
	f.Packages = make(map[string]*PackageRevision)
	f.Workflows = make(map[string]*Workflow)
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
func (f *TestFixture) CreateMultipleTestPackages(count int, namePrefix string) []*PackageRevision {
	packages := make([]*PackageRevision, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i)
		revision := fmt.Sprintf("v1.0.%d", i)
		packages[i] = f.CreateTestPackageRevision(name, revision)
	}

	return packages
}

// CreateMultipleTestRepositories creates multiple test repositories for bulk testing
func (f *TestFixture) CreateMultipleTestRepositories(count int, namePrefix string) []*Repository {
	repositories := make([]*Repository, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i)
		repositories[i] = f.CreateTestRepository(name)
	}

	return repositories
}

// ValidatePackageRevision performs basic validation on a package revision
func ValidatePackageRevision(pkg *PackageRevision) []string {
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

	if !IsValidLifecycle(pkg.Spec.Lifecycle) {
		errors = append(errors, "invalid lifecycle")
	}

	return errors
}

// ValidateRepository performs basic validation on a repository
func ValidateRepository(repo *Repository) []string {
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
