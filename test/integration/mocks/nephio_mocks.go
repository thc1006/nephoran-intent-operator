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



// Mock Porch v1alpha1 types.



// PackageRevision mocks the Nephio Porch PackageRevision type.

type PackageRevision struct {

	metav1.TypeMeta   `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`



	Spec   PackageRevisionSpec   `json:"spec,omitempty"`

	Status PackageRevisionStatus `json:"status,omitempty"`

}



// PackageRevisionSpec represents a packagerevisionspec.

type PackageRevisionSpec struct {

	PackageName   string                   `json:"packageName,omitempty"`

	Repository    string                   `json:"repository,omitempty"`

	Revision      string                   `json:"revision,omitempty"`

	Lifecycle     PackageRevisionLifecycle `json:"lifecycle,omitempty"`

	WorkspaceName string                   `json:"workspaceName,omitempty"`

	Tasks         []Task                   `json:"tasks,omitempty"`

}



// PackageRevisionStatus represents a packagerevisionstatus.

type PackageRevisionStatus struct {

	Conditions       []metav1.Condition `json:"conditions,omitempty"`

	PublishedBy      string             `json:"publishedBy,omitempty"`

	PublishedAt      *metav1.Time       `json:"publishedAt,omitempty"`

	UpstreamLock     *UpstreamLock      `json:"upstreamLock,omitempty"`

	DeploymentStatus DeploymentStatus   `json:"deploymentStatus,omitempty"`

}



// PackageRevisionLifecycle represents a packagerevisionlifecycle.

type PackageRevisionLifecycle string



const (

	// PackageRevisionLifecycleDraft holds packagerevisionlifecycledraft value.

	PackageRevisionLifecycleDraft PackageRevisionLifecycle = "Draft"

	// PackageRevisionLifecycleProposed holds packagerevisionlifecycleproposed value.

	PackageRevisionLifecycleProposed PackageRevisionLifecycle = "Proposed"

	// PackageRevisionLifecyclePublished holds packagerevisionlifecyclepublished value.

	PackageRevisionLifecyclePublished PackageRevisionLifecycle = "Published"

)



// Task represents a task.

type Task struct {

	Type string `json:"type,omitempty"`

}



// UpstreamLock represents a upstreamlock.

type UpstreamLock struct {

	Type string  `json:"type,omitempty"`

	Git  GitLock `json:"git,omitempty"`

}



// GitLock represents a gitlock.

type GitLock struct {

	Repo      string `json:"repo,omitempty"`

	Directory string `json:"directory,omitempty"`

	Ref       string `json:"ref,omitempty"`

	Commit    string `json:"commit,omitempty"`

}



// DeploymentStatus represents a deploymentstatus.

type DeploymentStatus struct {

	Status string `json:"status,omitempty"`

}



// PackageRevisionResources mocks the resources associated with a PackageRevision.

type PackageRevisionResources struct {

	metav1.TypeMeta   `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`



	Spec   PackageRevisionResourcesSpec   `json:"spec,omitempty"`

	Status PackageRevisionResourcesStatus `json:"status,omitempty"`

}



// PackageRevisionResourcesSpec represents a packagerevisionresourcesspec.

type PackageRevisionResourcesSpec struct {

	PackageName string            `json:"packageName,omitempty"`

	Revision    string            `json:"revision,omitempty"`

	Resources   map[string]string `json:"resources,omitempty"`

}



// PackageRevisionResourcesStatus represents a packagerevisionresourcesstatus.

type PackageRevisionResourcesStatus struct {

	RenderStatus RenderStatus `json:"renderStatus,omitempty"`

}



// RenderStatus represents a renderstatus.

type RenderStatus struct {

	Result string        `json:"result,omitempty"`

	Errors []RenderError `json:"errors,omitempty"`

}



// RenderError represents a rendererror.

type RenderError struct {

	File    string `json:"file,omitempty"`

	Index   int    `json:"index,omitempty"`

	Path    string `json:"path,omitempty"`

	Message string `json:"message,omitempty"`

}



// Repository mocks the Porch Repository type.

type Repository struct {

	metav1.TypeMeta   `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`



	Spec   RepositorySpec   `json:"spec,omitempty"`

	Status RepositoryStatus `json:"status,omitempty"`

}



// RepositorySpec represents a repositoryspec.

type RepositorySpec struct {

	Type        string   `json:"type,omitempty"`

	Content     string   `json:"content,omitempty"`

	Deployment  bool     `json:"deployment,omitempty"`

	Git         *GitSpec `json:"git,omitempty"`

	Oci         *OciSpec `json:"oci,omitempty"`

	Description string   `json:"description,omitempty"`

}



// GitSpec represents a gitspec.

type GitSpec struct {

	Repo      string           `json:"repo,omitempty"`

	Branch    string           `json:"branch,omitempty"`

	Directory string           `json:"directory,omitempty"`

	SecretRef *SecretReference `json:"secretRef,omitempty"`

}



// OciSpec represents a ocispec.

type OciSpec struct {

	Registry  string           `json:"registry,omitempty"`

	SecretRef *SecretReference `json:"secretRef,omitempty"`

}



// SecretReference represents a secretreference.

type SecretReference struct {

	Name string `json:"name,omitempty"`

}



// RepositoryStatus represents a repositorystatus.

type RepositoryStatus struct {

	Conditions []metav1.Condition `json:"conditions,omitempty"`

}



// Function represents a Porch Function.

type Function struct {

	metav1.TypeMeta   `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`



	Spec   FunctionSpec   `json:"spec,omitempty"`

	Status FunctionStatus `json:"status,omitempty"`

}



// FunctionSpec represents a functionspec.

type FunctionSpec struct {

	Image         string        `json:"image,omitempty"`

	Description   string        `json:"description,omitempty"`

	RepositoryRef RepositoryRef `json:"repositoryRef,omitempty"`

	Keywords      []string      `json:"keywords,omitempty"`

}



// RepositoryRef represents a repositoryref.

type RepositoryRef struct {

	Name string `json:"name,omitempty"`

}



// FunctionStatus represents a functionstatus.

type FunctionStatus struct {

	Conditions []metav1.Condition `json:"conditions,omitempty"`

}



// Mock REST Config.

type MockRESTConfig struct {

	HostURL    string

	APIPathURL string

	Username   string

	Password   string

}



// Host performs host operation.

func (m *MockRESTConfig) Host() string {

	return m.HostURL

}



// APIPath performs apipath operation.

func (m *MockRESTConfig) APIPath() string {

	return m.APIPathURL

}



// Client interface to match controller-runtime client.

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



// Mock Nephio client that implements the Client interface.

type MockNephioClient struct {

	client.Client

	Objects map[string]client.Object

}



// NewMockNephioClient performs newmocknephioclient operation.

func NewMockNephioClient() *MockNephioClient {

	return &MockNephioClient{

		Objects: make(map[string]client.Object),

	}

}



// Get performs get operation.

func (m *MockNephioClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {

	// Mock implementation.

	return nil

}



// List performs list operation.

func (m *MockNephioClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {

	// Mock implementation.

	return nil

}



// Create performs create operation.

func (m *MockNephioClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {

	// Mock implementation.

	return nil

}



// Delete performs delete operation.

func (m *MockNephioClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {

	// Mock implementation.

	return nil

}



// Update performs update operation.

func (m *MockNephioClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {

	// Mock implementation.

	return nil

}



// Patch performs patch operation.

func (m *MockNephioClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {

	// Mock implementation.

	return nil

}



// DeleteAllOf performs deleteallof operation.

func (m *MockNephioClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {

	// Mock implementation.

	return nil

}



// Status performs status operation.

func (m *MockNephioClient) Status() client.StatusWriter {

	return &MockStatusWriter{}

}



// Scheme performs scheme operation.

func (m *MockNephioClient) Scheme() *runtime.Scheme {

	return runtime.NewScheme()

}



// RESTMapper performs restmapper operation.

func (m *MockNephioClient) RESTMapper() meta.RESTMapper {

	return &MockRESTMapper{}

}



// Mock StatusWriter.

type MockStatusWriter struct{}



// Update performs update operation.

func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {

	return nil

}



// Create performs create operation.

func (m *MockStatusWriter) Create(ctx context.Context, obj, subResource client.Object, opts ...client.SubResourceCreateOption) error {

	return nil

}



// Patch performs patch operation.

func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {

	return nil

}



// Mock RESTMapper.

type MockRESTMapper struct{}



// KindFor performs kindfor operation.

func (m *MockRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {

	return schema.GroupVersionKind{}, nil

}



// KindsFor performs kindsfor operation.

func (m *MockRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {

	return nil, nil

}



// ResourceFor performs resourcefor operation.

func (m *MockRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {

	return schema.GroupVersionResource{}, nil

}



// ResourcesFor performs resourcesfor operation.

func (m *MockRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {

	return nil, nil

}



// RESTMapping performs restmapping operation.

func (m *MockRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {

	return nil, nil

}



// RESTMappings performs restmappings operation.

func (m *MockRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {

	return nil, nil

}



// ResourceSingularizer performs resourcesingularizer operation.

func (m *MockRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {

	return "", nil

}



// Porch Service mocks.



// MockPorchService represents a mockporchservice.

type MockPorchService struct {

	Packages     []*PackageRevision

	Repositories []*Repository

	Functions    []*Function

}



// NewMockPorchService performs newmockporchservice operation.

func NewMockPorchService() *MockPorchService {

	return &MockPorchService{

		Packages:     []*PackageRevision{},

		Repositories: []*Repository{},

		Functions:    []*Function{},

	}

}



// CreatePackage performs createpackage operation.

func (m *MockPorchService) CreatePackage(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {

	// Mock package creation.

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



// GetPackage performs getpackage operation.

func (m *MockPorchService) GetPackage(ctx context.Context, name, namespace string) (*PackageRevision, error) {

	for _, pkg := range m.Packages {

		if pkg.Name == name && pkg.Namespace == namespace {

			return pkg, nil

		}

	}

	return nil, nil

}



// UpdatePackage performs updatepackage operation.

func (m *MockPorchService) UpdatePackage(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {

	for i, existingPkg := range m.Packages {

		if existingPkg.Name == pkg.Name && existingPkg.Namespace == pkg.Namespace {

			m.Packages[i] = pkg

			return pkg, nil

		}

	}

	return nil, nil

}



// DeletePackage performs deletepackage operation.

func (m *MockPorchService) DeletePackage(ctx context.Context, name, namespace string) error {

	for i, pkg := range m.Packages {

		if pkg.Name == name && pkg.Namespace == namespace {

			m.Packages = append(m.Packages[:i], m.Packages[i+1:]...)

			return nil

		}

	}

	return nil

}



// ListPackages performs listpackages operation.

func (m *MockPorchService) ListPackages(ctx context.Context, namespace string) ([]*PackageRevision, error) {

	var result []*PackageRevision

	for _, pkg := range m.Packages {

		if pkg.Namespace == namespace || namespace == "" {

			result = append(result, pkg)

		}

	}

	return result, nil

}



// PublishPackage performs publishpackage operation.

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



// Additional helper functions for tests.



// CreateMockPackageRevision performs createmockpackagerevision operation.

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



// CreateMockRepository performs createmockrepository operation.

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



// CreateMockFunction performs createmockfunction operation.

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



// Mock REST Config factory.

func NewMockRESTConfig() *rest.Config {

	return &rest.Config{

		Host:     "http://mock-api-server:8080",

		APIPath:  "/api",

		Username: "mock-user",

		Password: "mock-password",

	}

}



// Mock client factory.

func NewMockRESTClient() rest.Interface {

	return &MockRESTClient{}

}



// Mock REST Client implementation.

type MockRESTClient struct{}



// GetRateLimiter performs getratelimiter operation.

func (m *MockRESTClient) GetRateLimiter() flowcontrol.RateLimiter {

	return nil

}



// Verb performs verb operation.

func (m *MockRESTClient) Verb(verb string) *rest.Request {

	return &rest.Request{}

}



// Post performs post operation.

func (m *MockRESTClient) Post() *rest.Request {

	return &rest.Request{}

}



// Put performs put operation.

func (m *MockRESTClient) Put() *rest.Request {

	return &rest.Request{}

}



// Patch performs patch operation.

func (m *MockRESTClient) Patch(pt types.PatchType) *rest.Request {

	return &rest.Request{}

}



// Get performs get operation.

func (m *MockRESTClient) Get() *rest.Request {

	return &rest.Request{}

}



// Delete performs delete operation.

func (m *MockRESTClient) Delete() *rest.Request {

	return &rest.Request{}

}



// APIVersion performs apiversion operation.

func (m *MockRESTClient) APIVersion() schema.GroupVersion {

	return schema.GroupVersion{Group: "", Version: "v1"}

}

