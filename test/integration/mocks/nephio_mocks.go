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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
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
	PackageName   string                     `json:"packageName,omitempty"`
	Repository    string                     `json:"repository,omitempty"`
	Revision      string                     `json:"revision,omitempty"`
	Lifecycle     PackageRevisionLifecycle   `json:"lifecycle,omitempty"`
	WorkspaceName string                     `json:"workspaceName,omitempty"`
	Tasks         []Task                     `json:"tasks,omitempty"`
}

type PackageRevisionStatus struct {
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	PublishedBy        string             `json:"publishedBy,omitempty"`
	PublishedAt        *metav1.Time       `json:"publishedAt,omitempty"`
	UpstreamLock       *UpstreamLock      `json:"upstreamLock,omitempty"`
	DeploymentStatus   DeploymentStatus   `json:"deploymentStatus,omitempty"`
}

type PackageRevisionLifecycle string

const (
	PackageRevisionLifecycleDraft     PackageRevisionLifecycle = "Draft"
	PackageRevisionLifecycleProposed  PackageRevisionLifecycle = "Proposed"
	PackageRevisionLifecyclePublished PackageRevisionLifecycle = "Published"
)

type Task struct {
	Type   string             `json:"type"`
	Clone  *PackageCloneTask  `json:"clone,omitempty"`
	Patch  *PackagePatchTask  `json:"patch,omitempty"`
	Edit   *PackageEditTask   `json:"edit,omitempty"`
	Eval   *FunctionEvalTask  `json:"eval,omitempty"`
}

type PackageCloneTask struct {
	Upstream UpstreamPackage `json:"upstream"`
}

type PackagePatchTask struct {
	Patches []PatchSpec `json:"patches"`
}

type PackageEditTask struct {
	Source *PackageEditTaskSource `json:"source,omitempty"`
}

type PackageEditTaskSource struct {
	Git *GitPackage `json:"git,omitempty"`
}

type GitPackage struct {
	Repo      string `json:"repo"`
	Ref       string `json:"ref"`
	Directory string `json:"directory"`
}

type FunctionEvalTask struct {
	Image        string                 `json:"image"`
	ConfigMap    map[string]string      `json:"configMap,omitempty"`
	Exec         string                 `json:"exec,omitempty"`
	EnableStderr bool                   `json:"enableStderr,omitempty"`
	Env          []runtime.RawExtension `json:"env,omitempty"`
}

type PatchSpec struct {
	File      string `json:"file"`
	Contents  string `json:"contents"`
	PatchType string `json:"patchType,omitempty"`
}

type UpstreamPackage struct {
	Git  *GitPackage  `json:"git,omitempty"`
	Oci  *OciPackage  `json:"oci,omitempty"`
	Type OriginType   `json:"type,omitempty"`
}

type OciPackage struct {
	Image string `json:"image"`
}

type OriginType string

const (
	OriginTypeGit OriginType = "git"
	OriginTypeOci OriginType = "oci"
)

type UpstreamLock struct {
	Type OriginType   `json:"type"`
	Git  *GitLock     `json:"git,omitempty"`
	Oci  *OciLock     `json:"oci,omitempty"`
}

type GitLock struct {
	Repo      string `json:"repo"`
	Directory string `json:"directory"`
	Ref       string `json:"ref"`
	Commit    string `json:"commit"`
}

type OciLock struct {
	Image  string `json:"image"`
	Digest string `json:"digest"`
}

type DeploymentStatus struct {
	Status     string `json:"status,omitempty"`
	Message    string `json:"message,omitempty"`
	LastUpdate string `json:"lastUpdate,omitempty"`
}

// Repository mocks the Nephio Porch Repository type
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

type RepositorySpec struct {
	Description string         `json:"description,omitempty"`
	Type        RepositoryType `json:"type"`
	Content     RepositoryContent `json:"content,omitempty"`
	Deployment  RepositoryDeployment `json:"deployment,omitempty"`
	Git         *GitRepository       `json:"git,omitempty"`
	Oci         *OciRepository       `json:"oci,omitempty"`
}

type RepositoryType string

const (
	RepositoryTypeGit RepositoryType = "git"
	RepositoryTypeOci RepositoryType = "oci"
)

type RepositoryContent string

const (
	RepositoryContentPackage RepositoryContent = "Package"
	RepositoryContentFunction RepositoryContent = "Function"
)

type RepositoryDeployment bool

type GitRepository struct {
	Repo      string `json:"repo"`
	Branch    string `json:"branch"`
	Directory string `json:"directory"`
	SecretRef SecretRef `json:"secretRef,omitempty"`
}

type OciRepository struct {
	Registry string `json:"registry"`
}

type SecretRef struct {
	Name string `json:"name"`
}

type RepositoryStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Mock Intent v1alpha1 types

// Intent mocks the Nephio Intent type
type Intent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	
	Spec   IntentSpec   `json:"spec,omitempty"`
	Status IntentStatus `json:"status,omitempty"`
}

type IntentSpec struct {
	PackageName string            `json:"packageName"`
	Repository  string            `json:"repository"`
	Revision    string            `json:"revision,omitempty"`
	Lifecycle   string            `json:"lifecycle,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

type IntentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Phase      string             `json:"phase,omitempty"`
	Message    string             `json:"message,omitempty"`
}

// Mock client interface
type Client interface {
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

// Test environment setup helpers
func NewTestConfig() *rest.Config {
	return &rest.Config{
		Host:    "http://localhost:8080",
		QPS:     100,
		Burst:   150,
		Timeout: 30 * time.Second,
	}
}

func SetupTestEnvironment() (*rest.Config, error) {
	return NewTestConfig(), nil
}