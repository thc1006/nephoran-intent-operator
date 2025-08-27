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

package mocks

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Mock Porch v1alpha1 types

// PackageRevision mocks the Nephio Porch PackageRevision type
type PackageRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageRevisionSpec   `json:"spec,omitempty"`
	Status PackageRevisionStatus `json:"status,omitempty"`
}

type PackageRevisionSpec struct {
	PackageName   string                   `json:"packageName,omitempty"`
	Repository    string                   `json:"repository,omitempty"`
	Revision      string                   `json:"revision,omitempty"`
	Lifecycle     PackageRevisionLifecycle `json:"lifecycle,omitempty"`
	WorkspaceName string                   `json:"workspaceName,omitempty"`
	Tasks         []Task                   `json:"tasks,omitempty"`
}

type PackageRevisionStatus struct {
	Conditions       []metav1.Condition `json:"conditions,omitempty"`
	PublishedBy      string             `json:"publishedBy,omitempty"`
	PublishedAt      *metav1.Time       `json:"publishedAt,omitempty"`
	UpstreamLock     *UpstreamLock      `json:"upstreamLock,omitempty"`
	DeploymentStatus DeploymentStatus   `json:"deploymentStatus,omitempty"`
}

type PackageRevisionLifecycle string

const (
	PackageRevisionLifecycleDraft     PackageRevisionLifecycle = "Draft"
	PackageRevisionLifecycleProposed  PackageRevisionLifecycle = "Proposed"
	PackageRevisionLifecyclePublished PackageRevisionLifecycle = "Published"
)

type Task struct {
	Type string `json:"type,omitempty"`
}

type UpstreamLock struct {
	Type string  `json:"type,omitempty"`
	Git  GitLock `json:"git,omitempty"`
}

type GitLock struct {
	Repo      string `json:"repo,omitempty"`
	Directory string `json:"directory,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Commit    string `json:"commit,omitempty"`
}

type DeploymentStatus struct {
	Status string `json:"status,omitempty"`
}

// PackageRevisionResources mocks the resources associated with a PackageRevision
type PackageRevisionResources struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageRevisionResourcesSpec   `json:"spec,omitempty"`
	Status PackageRevisionResourcesStatus `json:"status,omitempty"`
}

type PackageRevisionResourcesSpec struct {
	PackageName string            `json:"packageName,omitempty"`
	Revision    string            `json:"revision,omitempty"`
	Resources   map[string]string `json:"resources,omitempty"`
}

type PackageRevisionResourcesStatus struct {
	RenderStatus RenderStatus `json:"renderStatus,omitempty"`
}

type RenderStatus struct {
	Result string        `json:"result,omitempty"`
	Errors []RenderError `json:"errors,omitempty"`
}

type RenderError struct {
	File    string `json:"file,omitempty"`
	Index   int    `json:"index,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
}

// Repository mocks the Porch Repository type
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

type RepositorySpec struct {
	Type        string   `json:"type,omitempty"`
	Content     string   `json:"content,omitempty"`
	Deployment  bool     `json:"deployment,omitempty"`
	Git         *GitSpec `json:"git,omitempty"`
	Oci         *OciSpec `json:"oci,omitempty"`
	Description string   `json:"description,omitempty"`
}

type GitSpec struct {
	Repo      string           `json:"repo,omitempty"`
	Branch    string           `json:"branch,omitempty"`
	Directory string           `json:"directory,omitempty"`
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

type OciSpec struct {
	Registry  string           `json:"registry,omitempty"`
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

type SecretReference struct {
	Name string `json:"name,omitempty"`
}

type RepositoryStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Function represents a Porch Function
type Function struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionSpec   `json:"spec,omitempty"`
	Status FunctionStatus `json:"status,omitempty"`
}

type FunctionSpec struct {
	Image         string        `json:"image,omitempty"`
	Description   string        `json:"description,omitempty"`
	RepositoryRef RepositoryRef `json:"repositoryRef,omitempty"`
	Keywords      []string      `json:"keywords,omitempty"`
}

type RepositoryRef struct {
	Name string `json:"name,omitempty"`
}

type FunctionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Mock REST Config
type MockRESTConfig struct {
	HostURL    string
	APIPathURL string
	Username   string
	Password   string
}

func (m *MockRESTConfig) Host() string {
	return m.HostURL
}

func (m *MockRESTConfig) APIPath() string {
	return m.APIPathURL
}

// Client interface to match controller-runtime client
type MockClient interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object) error
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
	Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error
	DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error
	Status() client.StatusWriter
	Scheme() *runtime.Scheme
	RESTMapper() meta.RESTMapper
}

// Mock Nephio client that implements the Client interface
type MockNephioClient struct {
	client.Client
	Objects map[string]client.Object
}

func NewMockNephioClient() *MockNephioClient {
	return &MockNephioClient{
		Objects: make(map[string]client.Object),
	}
}

func (m *MockNephioClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	// Mock implementation
	return nil
}

func (m *MockNephioClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	// Mock implementation
	return nil
}

func (m *MockNephioClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	// Mock implementation
	return nil
}

func (m *MockNephioClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	// Mock implementation
	return nil
}

func (m *MockNephioClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	// Mock implementation
	return nil
}

func (m *MockNephioClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	// Mock implementation
	return nil
}

func (m *MockNephioClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	// Mock implementation
	return nil
}

func (m *MockNephioClient) Status() client.StatusWriter {
	return &MockStatusWriter{}
}

func (m *MockNephioClient) Scheme() *runtime.Scheme {
	return runtime.NewScheme()
}

func (m *MockNephioClient) RESTMapper() meta.RESTMapper {
	return &MockRESTMapper{}
}

// Mock StatusWriter
type MockStatusWriter struct{}

func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return nil
}

func (m *MockStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return nil
}

func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return nil
}

// Mock RESTMapper
type MockRESTMapper struct{}

func (m *MockRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *MockRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	return nil, nil
}

func (m *MockRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, nil
}

func (m *MockRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	return nil, nil
}

func (m *MockRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	return nil, nil
}

func (m *MockRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	return nil, nil
}

func (m *MockRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	return "", nil
}

// Porch Service mocks

type MockPorchService struct {
	Packages     []*PackageRevision
	Repositories []*Repository
	Functions    []*Function
}

func NewMockPorchService() *MockPorchService {
	return &MockPorchService{
		Packages:     []*PackageRevision{},
		Repositories: []*Repository{},
		Functions:    []*Function{},
	}
}

func (m *MockPorchService) CreatePackage(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	// Mock package creation
	pkg.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "Created",
			Message:            "Package created successfully",
			LastTransitionTime: metav1.NewTime(time.Now()),
		},
	}
	m.Packages = append(m.Packages, pkg)
	return pkg, nil
}

func (m *MockPorchService) GetPackage(ctx context.Context, name, namespace string) (*PackageRevision, error) {
	for _, pkg := range m.Packages {
		if pkg.Name == name && pkg.Namespace == namespace {
			return pkg, nil
		}
	}
	return nil, nil
}

func (m *MockPorchService) UpdatePackage(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	for i, existingPkg := range m.Packages {
		if existingPkg.Name == pkg.Name && existingPkg.Namespace == pkg.Namespace {
			m.Packages[i] = pkg
			return pkg, nil
		}
	}
	return nil, nil
}

func (m *MockPorchService) DeletePackage(ctx context.Context, name, namespace string) error {
	for i, pkg := range m.Packages {
		if pkg.Name == name && pkg.Namespace == namespace {
			m.Packages = append(m.Packages[:i], m.Packages[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MockPorchService) ListPackages(ctx context.Context, namespace string) ([]*PackageRevision, error) {
	var result []*PackageRevision
	for _, pkg := range m.Packages {
		if pkg.Namespace == namespace || namespace == "" {
			result = append(result, pkg)
		}
	}
	return result, nil
}

func (m *MockPorchService) PublishPackage(ctx context.Context, name, namespace string) (*PackageRevision, error) {
	for _, pkg := range m.Packages {
		if pkg.Name == name && pkg.Namespace == namespace {
			pkg.Spec.Lifecycle = PackageRevisionLifecyclePublished
			pkg.Status.PublishedBy = "mock-user"
			now := metav1.NewTime(time.Now())
			pkg.Status.PublishedAt = &now
			return pkg, nil
		}
	}
	return nil, nil
}

// Additional helper functions for tests

func CreateMockPackageRevision(name, namespace, packageName string) *PackageRevision {
	return &PackageRevision{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.kpt.dev/v1alpha1",
			Kind:       "PackageRevision",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: PackageRevisionSpec{
			PackageName: packageName,
			Repository:  "mock-repo",
			Revision:    "v1.0.0",
			Lifecycle:   PackageRevisionLifecycleDraft,
		},
		Status: PackageRevisionStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					Reason:             "Initialized",
					Message:            "Package revision initialized",
					LastTransitionTime: metav1.NewTime(time.Now()),
				},
			},
		},
	}
}

func CreateMockRepository(name, namespace, repoType string) *Repository {
	return &Repository{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.porch.kpt.dev/v1alpha1",
			Kind:       "Repository",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: RepositorySpec{
			Type:       repoType,
			Content:    "Package",
			Deployment: true,
			Git: &GitSpec{
				Repo:   "https://github.com/example/repo.git",
				Branch: "main",
			},
		},
		Status: RepositoryStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					Reason:             "Registered",
					Message:            "Repository registered successfully",
					LastTransitionTime: metav1.NewTime(time.Now()),
				},
			},
		},
	}
}

func CreateMockFunction(name, namespace, image string) *Function {
	return &Function{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.porch.kpt.dev/v1alpha1",
			Kind:       "Function",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: FunctionSpec{
			Image:       image,
			Description: "Mock function for testing",
			Keywords:    []string{"mock", "test"},
		},
		Status: FunctionStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					Reason:             "Available",
					Message:            "Function is available",
					LastTransitionTime: metav1.NewTime(time.Now()),
				},
			},
		},
	}
}

// Mock REST Config factory
func NewMockRESTConfig() *rest.Config {
	return &rest.Config{
		Host:     "http://mock-api-server:8080",
		APIPath:  "/api",
		Username: "mock-user",
		Password: "mock-password",
	}
}

// Mock client factory
func NewMockRESTClient() rest.Interface {
	return &MockRESTClient{}
}

// Mock REST Client implementation
type MockRESTClient struct{}

func (m *MockRESTClient) GetRateLimiter() flowcontrol.RateLimiter {
	return nil
}

func (m *MockRESTClient) Verb(verb string) *rest.Request {
	return &rest.Request{}
}

func (m *MockRESTClient) Post() *rest.Request {
	return &rest.Request{}
}

func (m *MockRESTClient) Put() *rest.Request {
	return &rest.Request{}
}

func (m *MockRESTClient) Patch(pt types.PatchType) *rest.Request {
	return &rest.Request{}
}

func (m *MockRESTClient) Get() *rest.Request {
	return &rest.Request{}
}

func (m *MockRESTClient) Delete() *rest.Request {
	return &rest.Request{}
}

func (m *MockRESTClient) APIVersion() schema.GroupVersion {
	return schema.GroupVersion{Group: "", Version: "v1"}
}
