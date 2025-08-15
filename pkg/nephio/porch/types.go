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
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// PorchClient defines the interface for interacting with Porch API
// Provides comprehensive CRUD operations for Porch resources with O-RAN compliance
type PorchClient interface {
	// Repository Operations
	GetRepository(ctx context.Context, name string) (*Repository, error)
	ListRepositories(ctx context.Context, opts *ListOptions) (*RepositoryList, error)
	CreateRepository(ctx context.Context, repo *Repository) (*Repository, error)
	UpdateRepository(ctx context.Context, repo *Repository) (*Repository, error)
	DeleteRepository(ctx context.Context, name string) error
	SyncRepository(ctx context.Context, name string) error

	// PackageRevision Operations
	GetPackageRevision(ctx context.Context, name string, revision string) (*PackageRevision, error)
	ListPackageRevisions(ctx context.Context, opts *ListOptions) (*PackageRevisionList, error)
	CreatePackageRevision(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error)
	UpdatePackageRevision(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error)
	DeletePackageRevision(ctx context.Context, name string, revision string) error
	ApprovePackageRevision(ctx context.Context, name string, revision string) error
	ProposePackageRevision(ctx context.Context, name string, revision string) error
	RejectPackageRevision(ctx context.Context, name string, revision string, reason string) error

	// Package Content Operations
	GetPackageContents(ctx context.Context, name string, revision string) (map[string][]byte, error)
	UpdatePackageContents(ctx context.Context, name string, revision string, contents map[string][]byte) error
	RenderPackage(ctx context.Context, name string, revision string) (*RenderResult, error)

	// Function Operations
	RunFunction(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error)
	ValidatePackage(ctx context.Context, name string, revision string) (*ValidationResult, error)

	// Workflow Operations
	GetWorkflow(ctx context.Context, name string) (*Workflow, error)
	ListWorkflows(ctx context.Context, opts *ListOptions) (*WorkflowList, error)
	CreateWorkflow(ctx context.Context, workflow *Workflow) (*Workflow, error)
	UpdateWorkflow(ctx context.Context, workflow *Workflow) (*Workflow, error)
	DeleteWorkflow(ctx context.Context, name string) error

	// Health and Status
	Health(ctx context.Context) (*HealthStatus, error)
	Version(ctx context.Context) (*VersionInfo, error)
}

// RepositoryManager provides high-level repository lifecycle management
type RepositoryManager interface {
	// Repository lifecycle
	RegisterRepository(ctx context.Context, config *RepositoryConfig) (*Repository, error)
	UnregisterRepository(ctx context.Context, name string) error
	SynchronizeRepository(ctx context.Context, name string) (*SyncResult, error)
	GetRepositoryHealth(ctx context.Context, name string) (*RepositoryHealth, error)

	// Branch management
	CreateBranch(ctx context.Context, repoName string, branchName string, baseBranch string) error
	DeleteBranch(ctx context.Context, repoName string, branchName string) error
	ListBranches(ctx context.Context, repoName string) ([]string, error)

	// Authentication and credentials
	UpdateCredentials(ctx context.Context, repoName string, creds *Credentials) error
	ValidateAccess(ctx context.Context, repoName string) error
}

// PackageRevisionManager provides package revision lifecycle management
type PackageRevisionManager interface {
	// Package lifecycle
	CreatePackage(ctx context.Context, spec *PackageSpec) (*PackageRevision, error)
	ClonePackage(ctx context.Context, source *PackageReference, target *PackageSpec) (*PackageRevision, error)
	DeletePackage(ctx context.Context, ref *PackageReference) error

	// Revision management
	CreateRevision(ctx context.Context, ref *PackageReference) (*PackageRevision, error)
	GetRevision(ctx context.Context, ref *PackageReference) (*PackageRevision, error)
	ListRevisions(ctx context.Context, packageName string) ([]*PackageRevision, error)
	CompareRevisions(ctx context.Context, ref1, ref2 *PackageReference) (*ComparisonResult, error)

	// State transitions
	PromoteToProposed(ctx context.Context, ref *PackageReference) error
	PromoteToPublished(ctx context.Context, ref *PackageReference) error
	RevertToRevision(ctx context.Context, ref *PackageReference, targetRevision string) error

	// Package content management
	UpdateContent(ctx context.Context, ref *PackageReference, updates map[string][]byte) error
	GetContent(ctx context.Context, ref *PackageReference) (*PackageContent, error)
	ValidateContent(ctx context.Context, ref *PackageReference) (*ValidationResult, error)
}

// FunctionRunner provides KRM function execution capabilities
type FunctionRunner interface {
	// Function execution
	ExecuteFunction(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error)
	ExecutePipeline(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error)
	ValidateFunction(ctx context.Context, functionName string) (*FunctionValidation, error)

	// Function management
	ListFunctions(ctx context.Context) ([]*FunctionInfo, error)
	GetFunctionSchema(ctx context.Context, functionName string) (*FunctionSchema, error)
	RegisterFunction(ctx context.Context, info *FunctionInfo) error
}

// NetworkIntentExtensions provides Nephio-specific extensions to NetworkIntent CRD
type NetworkIntentExtensions struct {
	// Porch integration metadata
	PorchMetadata *PorchMetadata `json:"porchMetadata,omitempty"`

	// Package specifications
	PackageSpec *PackageSpec `json:"packageSpec,omitempty"`

	// Target cluster information
	ClusterTargets []*ClusterTarget `json:"clusterTargets,omitempty"`

	// Workflow definition
	Workflow *WorkflowSpec `json:"workflow,omitempty"`

	// O-RAN compliance settings
	ORANCompliance *ORANComplianceSpec `json:"oranCompliance,omitempty"`

	// Network slice configuration
	NetworkSlice *NetworkSliceSpec `json:"networkSlice,omitempty"`
}

// Core Porch Resource Types

// Repository represents a Git repository in Porch
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// RepositoryList contains a list of repositories
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Repository `json:"items"`
}

// RepositorySpec defines the desired state of a repository
type RepositorySpec struct {
	// Repository type (git, oci)
	Type string `json:"type"`

	// Repository URL
	URL string `json:"url"`

	// Branch to track
	Branch string `json:"branch,omitempty"`

	// Directory within repository
	Directory string `json:"directory,omitempty"`

	// Authentication configuration
	Auth *AuthConfig `json:"auth,omitempty"`

	// Synchronization settings
	Sync *SyncConfig `json:"sync,omitempty"`

	// Repository capabilities
	Capabilities []string `json:"capabilities,omitempty"`
}

// RepositoryStatus defines the observed state of a repository
type RepositoryStatus struct {
	// Conditions represent the current state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Last sync time
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Current commit hash
	CommitHash string `json:"commitHash,omitempty"`

	// Sync error if any
	SyncError string `json:"syncError,omitempty"`

	// Package count
	PackageCount int32 `json:"packageCount,omitempty"`

	// Repository health
	Health RepositoryHealth `json:"health,omitempty"`
}

// PackageRevision represents a package revision in Porch
type PackageRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageRevisionSpec   `json:"spec,omitempty"`
	Status PackageRevisionStatus `json:"status,omitempty"`
}

// PackageRevisionList contains a list of package revisions
type PackageRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []PackageRevision `json:"items"`
}

// PackageRevisionSpec defines the desired state of a package revision
type PackageRevisionSpec struct {
	// Package name
	PackageName string `json:"packageName"`

	// Repository name
	Repository string `json:"repository"`

	// Revision identifier
	Revision string `json:"revision"`

	// Lifecycle stage
	Lifecycle PackageRevisionLifecycle `json:"lifecycle"`

	// Package content
	Resources []KRMResource `json:"resources,omitempty"`

	// Function pipeline
	Functions []FunctionConfig `json:"functions,omitempty"`

	// Package metadata
	PackageMetadata *PackageMetadata `json:"packageMetadata,omitempty"`

	// Approval workflow
	WorkflowLock *WorkflowLock `json:"workflowLock,omitempty"`
}

// PackageRevisionStatus defines the observed state of a package revision
type PackageRevisionStatus struct {
	// Conditions represent the current state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Publishing timestamp
	PublishTime *metav1.Time `json:"publishTime,omitempty"`

	// Validation results
	ValidationResults []*ValidationResult `json:"validationResults,omitempty"`

	// Rendering results
	RenderingResults *RenderResult `json:"renderingResults,omitempty"`

	// Deployment status
	DeploymentStatus *DeploymentStatus `json:"deploymentStatus,omitempty"`

	// Package size
	PackageSize int64 `json:"packageSize,omitempty"`

	// Downstream packages using this revision
	Downstream []PackageReference `json:"downstream,omitempty"`
}

// Workflow represents a package approval workflow
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// WorkflowList contains a list of workflows
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Workflow `json:"items"`
}

// Supporting Types and Enums

// PackageRevisionLifecycle defines the lifecycle stages of a package revision
type PackageRevisionLifecycle string

const (
	PackageRevisionLifecycleDraft     PackageRevisionLifecycle = "Draft"
	PackageRevisionLifecycleProposed  PackageRevisionLifecycle = "Proposed"
	PackageRevisionLifecyclePublished PackageRevisionLifecycle = "Published"
	PackageRevisionLifecycleDeletable PackageRevisionLifecycle = "Deletable"
)

// RepositoryHealth represents the health status of a repository
type RepositoryHealth string

const (
	RepositoryHealthHealthy   RepositoryHealth = "Healthy"
	RepositoryHealthUnhealthy RepositoryHealth = "Unhealthy"
	RepositoryHealthUnknown   RepositoryHealth = "Unknown"
)

// Configuration Types

// RepositoryConfig defines repository configuration
type RepositoryConfig struct {
	Name         string      `json:"name"`
	URL          string      `json:"url"`
	Type         string      `json:"type"`
	Branch       string      `json:"branch,omitempty"`
	Directory    string      `json:"directory,omitempty"`
	Auth         *AuthConfig `json:"auth,omitempty"`
	Sync         *SyncConfig `json:"sync,omitempty"`
	Capabilities []string    `json:"capabilities,omitempty"`
}

// GitAuthConfig defines Git authentication configuration
type GitAuthConfig struct {
	Type       string            `json:"type"` // basic, token, ssh
	Username   string            `json:"username,omitempty"`
	Password   string            `json:"password,omitempty"`
	Token      string            `json:"token,omitempty"`
	PrivateKey string            `json:"privateKey,omitempty"`
	SecretRef  *SecretReference  `json:"secretRef,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// SyncConfig defines synchronization configuration
type SyncConfig struct {
	Interval       *metav1.Duration `json:"interval,omitempty"`
	AutoSync       bool             `json:"autoSync,omitempty"`
	MaxRetries     int32            `json:"maxRetries,omitempty"`
	BackoffLimit   *metav1.Duration `json:"backoffLimit,omitempty"`
	WebhookEnabled bool             `json:"webhookEnabled,omitempty"`
}

// Package Types

// PackageSpec defines package creation specification
type PackageSpec struct {
	Repository  string                   `json:"repository"`
	PackageName string                   `json:"packageName"`
	Revision    string                   `json:"revision,omitempty"`
	Lifecycle   PackageRevisionLifecycle `json:"lifecycle,omitempty"`
	Labels      map[string]string        `json:"labels,omitempty"`
	Annotations map[string]string        `json:"annotations,omitempty"`
}

// PackageReference uniquely identifies a package revision
type PackageReference struct {
	Repository  string `json:"repository"`
	PackageName string `json:"packageName"`
	Revision    string `json:"revision"`
}

// PackageContent contains the full content of a package
type PackageContent struct {
	Files   map[string][]byte `json:"files"`
	Kptfile *KptfileContent   `json:"kptfile,omitempty"`
}

// KptfileContent represents the Kptfile structure
type KptfileContent struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Info       *PackageMetadata       `json:"info,omitempty"`
	Pipeline   *Pipeline              `json:"pipeline,omitempty"`
}

// PackageMetadata contains package metadata information
type PackageMetadata struct {
	Description string            `json:"description,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Site        string            `json:"site,omitempty"`
	Emails      []string          `json:"emails,omitempty"`
	License     string            `json:"license,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// Pipeline defines the KRM function pipeline
type Pipeline struct {
	Mutators   []FunctionConfig `json:"mutators,omitempty"`
	Validators []FunctionConfig `json:"validators,omitempty"`
}

// Function Types

// FunctionConfig defines a KRM function configuration
type FunctionConfig struct {
	Image      string                 `json:"image"`
	ConfigPath string                 `json:"configPath,omitempty"`
	ConfigMap  map[string]interface{} `json:"configMap,omitempty"`
	Selectors  []ResourceSelector     `json:"selectors,omitempty"`
	Exec       *ExecConfig            `json:"exec,omitempty"`
}

// FunctionRequest represents a function execution request
type FunctionRequest struct {
	FunctionConfig FunctionConfig   `json:"functionConfig"`
	Resources      []KRMResource    `json:"resources"`
	Context        *FunctionContext `json:"context,omitempty"`
}

// FunctionResponse represents a function execution response
type FunctionResponse struct {
	Resources []KRMResource     `json:"resources"`
	Results   []*FunctionResult `json:"results,omitempty"`
	Logs      []string          `json:"logs,omitempty"`
	Error     *FunctionError    `json:"error,omitempty"`
}

// PipelineRequest represents a pipeline execution request
type PipelineRequest struct {
	Pipeline  Pipeline         `json:"pipeline"`
	Resources []KRMResource    `json:"resources"`
	Context   *FunctionContext `json:"context,omitempty"`
}

// PipelineResponse represents a pipeline execution response
type PipelineResponse struct {
	Resources []KRMResource     `json:"resources"`
	Results   []*FunctionResult `json:"results,omitempty"`
	Error     *FunctionError    `json:"error,omitempty"`
}

// KRM Resource Types

// KRMResource represents a Kubernetes resource manifest
type KRMResource struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       map[string]interface{} `json:"spec,omitempty"`
	Status     map[string]interface{} `json:"status,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// ResourceSelector defines resource selection criteria
type ResourceSelector struct {
	APIVersion string            `json:"apiVersion,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Name       string            `json:"name,omitempty"`
	Namespace  string            `json:"namespace,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// Validation and Rendering Types

// ValidationResult contains validation results
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Path        string `json:"path"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	Code        string `json:"code,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// RenderResult contains rendering results
type RenderResult struct {
	Resources []KRMResource     `json:"resources"`
	Results   []*FunctionResult `json:"results,omitempty"`
	Error     *RenderError      `json:"error,omitempty"`
}

// RenderError represents a rendering error
type RenderError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Details string `json:"details,omitempty"`
}

// Workflow Types

// WorkflowSpec defines the desired state of a workflow
type WorkflowSpec struct {
	Stages      []WorkflowStage   `json:"stages"`
	Triggers    []WorkflowTrigger `json:"triggers,omitempty"`
	Approvers   []Approver        `json:"approvers,omitempty"`
	Timeout     *metav1.Duration  `json:"timeout,omitempty"`
	RetryPolicy *RetryPolicy      `json:"retryPolicy,omitempty"`
}

// GeneralWorkflowStatus defines the general observed state of a workflow
type GeneralWorkflowStatus struct {
	Phase      WorkflowPhase      `json:"phase"`
	Stage      string             `json:"stage,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	StartTime  *metav1.Time       `json:"startTime,omitempty"`
	EndTime    *metav1.Time       `json:"endTime,omitempty"`
	Results    []WorkflowResult   `json:"results,omitempty"`
}

// WorkflowStage represents a workflow stage
type WorkflowStage struct {
	Name       string              `json:"name"`
	Type       WorkflowStageType   `json:"type"`
	Conditions []WorkflowCondition `json:"conditions,omitempty"`
	Actions    []WorkflowAction    `json:"actions"`
	Approvers  []Approver          `json:"approvers,omitempty"`
	Timeout    *metav1.Duration    `json:"timeout,omitempty"`
	OnFailure  *FailureAction      `json:"onFailure,omitempty"`
}

// Nephio-Specific Types

// PorchMetadata contains Porch integration metadata
type PorchMetadata struct {
	Repository  string            `json:"repository"`
	PackageName string            `json:"packageName"`
	Revision    string            `json:"revision"`
	GeneratedAt *metav1.Time      `json:"generatedAt"`
	GeneratedBy string            `json:"generatedBy,omitempty"`
	IntentID    string            `json:"intentId"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ClusterTarget defines a target cluster for deployment
type ClusterTarget struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Context     string            `json:"context,omitempty"`
	Kubeconfig  string            `json:"kubeconfig,omitempty"`
}

// ORANComplianceSpec defines O-RAN compliance requirements
type ORANComplianceSpec struct {
	Interfaces     []ORANInterface  `json:"interfaces,omitempty"`
	Validations    []ComplianceRule `json:"validations,omitempty"`
	Certifications []string         `json:"certifications,omitempty"`
	Standards      []StandardRef    `json:"standards,omitempty"`
}

// ORANInterface defines an O-RAN interface configuration
type ORANInterface struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // A1, O1, O2, E2
	Version  string                 `json:"version"`
	Endpoint string                 `json:"endpoint,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Enabled  bool                   `json:"enabled"`
}

// ComplianceRule defines a compliance validation rule
type ComplianceRule struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Rule        string `json:"rule"`
	Remediation string `json:"remediation,omitempty"`
}

// StandardRef references an industry standard
type StandardRef struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Section  string `json:"section,omitempty"`
	Required bool   `json:"required"`
}

// NetworkSliceSpec defines network slice configuration
type NetworkSliceSpec struct {
	SliceID   string               `json:"sliceId"`
	SliceType string               `json:"sliceType"` // eMBB, URLLC, mMTC
	SLA       *SLAParameters       `json:"sla"`
	QoS       *QoSParameters       `json:"qos,omitempty"`
	Resources *SliceResources      `json:"resources,omitempty"`
	Isolation *IsolationParameters `json:"isolation,omitempty"`
	Geography []GeographicArea     `json:"geography,omitempty"`
}

// SLAParameters defines service level agreement parameters
type SLAParameters struct {
	Latency      *LatencyRequirement      `json:"latency,omitempty"`
	Throughput   *ThroughputRequirement   `json:"throughput,omitempty"`
	Availability *AvailabilityRequirement `json:"availability,omitempty"`
	Reliability  *ReliabilityRequirement  `json:"reliability,omitempty"`
}

// Supporting utility types and interfaces

// ListOptions defines options for list operations
type ListOptions struct {
	LabelSelector string
	FieldSelector string
	Namespace     string
	Limit         int64
	Continue      string
}

// SecretReference references a Kubernetes secret
type SecretReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

// Credentials contains authentication credentials
type Credentials struct {
	Type       string            `json:"type"`
	Username   string            `json:"username,omitempty"`
	Password   string            `json:"password,omitempty"`
	Token      string            `json:"token,omitempty"`
	PrivateKey []byte            `json:"privateKey,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// SyncResult contains repository synchronization results
type SyncResult struct {
	Success      bool      `json:"success"`
	CommitHash   string    `json:"commitHash,omitempty"`
	PackageCount int32     `json:"packageCount"`
	SyncTime     time.Time `json:"syncTime"`
	Error        string    `json:"error,omitempty"`
}

// ComparisonResult contains package revision comparison results
type ComparisonResult struct {
	Added    []string `json:"added,omitempty"`
	Modified []string `json:"modified,omitempty"`
	Deleted  []string `json:"deleted,omitempty"`
	Diff     string   `json:"diff,omitempty"`
}

// WorkflowLock represents a workflow lock on a package
type WorkflowLock struct {
	LockedBy   string       `json:"lockedBy"`
	LockedAt   *metav1.Time `json:"lockedAt"`
	Reason     string       `json:"reason,omitempty"`
	WorkflowID string       `json:"workflowId"`
}

// DeploymentStatus represents package deployment status
type DeploymentStatus struct {
	Phase      string             `json:"phase"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Targets    []DeploymentTarget `json:"targets,omitempty"`
	StartTime  *metav1.Time       `json:"startTime,omitempty"`
	EndTime    *metav1.Time       `json:"endTime,omitempty"`
}

// DeploymentTarget represents a deployment target
type DeploymentTarget struct {
	Cluster   string             `json:"cluster"`
	Namespace string             `json:"namespace"`
	Status    string             `json:"status"`
	Error     string             `json:"error,omitempty"`
	Resources []DeployedResource `json:"resources,omitempty"`
}

// DeployedResource represents a deployed resource
type DeployedResource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	Status     string `json:"status"`
}

// Function execution and validation types

// FunctionContext provides context for function execution
type FunctionContext struct {
	Package     *PackageReference   `json:"package,omitempty"`
	Invocation  *FunctionInvocation `json:"invocation,omitempty"`
	Environment map[string]string   `json:"environment,omitempty"`
}

// FunctionInvocation contains function invocation details
type FunctionInvocation struct {
	ID        string       `json:"id"`
	Timestamp *metav1.Time `json:"timestamp"`
	User      string       `json:"user,omitempty"`
}

// FunctionResult represents a function execution result
type FunctionResult struct {
	Message  string            `json:"message"`
	Severity string            `json:"severity"`
	Field    string            `json:"field,omitempty"`
	File     string            `json:"file,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// FunctionError represents a function execution error
type FunctionError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// FunctionValidation contains function validation results
type FunctionValidation struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
	Schema   *FunctionSchema   `json:"schema,omitempty"`
}

// FunctionInfo contains function metadata
type FunctionInfo struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Description string            `json:"description,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Types       []string          `json:"types,omitempty"` // mutator, validator
	Schema      *FunctionSchema   `json:"schema,omitempty"`
	Examples    []FunctionExample `json:"examples,omitempty"`
}

// FunctionSchema defines function configuration schema
type FunctionSchema struct {
	OpenAPIV3Schema *runtime.RawExtension     `json:"openAPIV3Schema,omitempty"`
	Properties      map[string]SchemaProperty `json:"properties,omitempty"`
	Required        []string                  `json:"required,omitempty"`
}

// SchemaProperty defines a schema property
type SchemaProperty struct {
	Type        string        `json:"type"`
	Description string        `json:"description,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Examples    []interface{} `json:"examples,omitempty"`
}

// FunctionExample contains function usage examples
type FunctionExample struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Config      *FunctionConfig `json:"config"`
	Input       []KRMResource   `json:"input"`
	Output      []KRMResource   `json:"output"`
}

// ExecConfig defines execution configuration for container functions
type ExecConfig struct {
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	WorkDir string            `json:"workDir,omitempty"`
}

// Health and Status Types

// HealthStatus represents overall system health
type HealthStatus struct {
	Status     string            `json:"status"`
	Timestamp  *metav1.Time      `json:"timestamp"`
	Components []ComponentHealth `json:"components,omitempty"`
}

// ComponentHealth represents component health status
type ComponentHealth struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// BuildVersionInfo contains build version information
type BuildVersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit,omitempty"`
	BuildTime string `json:"buildTime,omitempty"`
	GoVersion string `json:"goVersion,omitempty"`
}

// Workflow execution types

// WorkflowPhase represents workflow execution phase
type WorkflowPhase string

const (
	WorkflowPhasePending   WorkflowPhase = "Pending"
	WorkflowPhaseRunning   WorkflowPhase = "Running"
	WorkflowPhaseSucceeded WorkflowPhase = "Succeeded"
	WorkflowPhaseFailed    WorkflowPhase = "Failed"
	WorkflowPhaseTimeout   WorkflowPhase = "Timeout"
)

// WorkflowStageType represents workflow stage type
type WorkflowStageType string

const (
	WorkflowStageTypeValidation WorkflowStageType = "Validation"
	WorkflowStageTypeApproval   WorkflowStageType = "Approval"
	WorkflowStageTypeDeployment WorkflowStageType = "Deployment"
	WorkflowStageTypeTesting    WorkflowStageType = "Testing"
)

// WorkflowTrigger defines when workflow should be triggered
type WorkflowTrigger struct {
	Type      string                 `json:"type"`
	Condition map[string]interface{} `json:"condition"`
}

// Approver defines who can approve workflow stages
type Approver struct {
	Type   string   `json:"type"` // user, group, service
	Name   string   `json:"name"`
	Roles  []string `json:"roles,omitempty"`
	Stages []string `json:"stages,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries   int32            `json:"maxRetries"`
	BackoffDelay *metav1.Duration `json:"backoffDelay,omitempty"`
	BackoffType  string           `json:"backoffType,omitempty"` // fixed, exponential
}

// WorkflowCondition defines conditions for stage execution
type WorkflowCondition struct {
	Type      string                 `json:"type"`
	Condition map[string]interface{} `json:"condition"`
}

// WorkflowAction defines actions to execute in a stage
type WorkflowAction struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// FailureAction defines what to do on stage failure
type FailureAction struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// WorkflowResult contains workflow execution results
type WorkflowResult struct {
	Stage     string                 `json:"stage"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp *metav1.Time           `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Network slice supporting types

// QoSParameters defines quality of service parameters
type QoSParameters struct {
	FiveQI             int32  `json:"5qi,omitempty"`
	PriorityLevel      int32  `json:"priorityLevel,omitempty"`
	PacketDelayBudget  int32  `json:"packetDelayBudget,omitempty"`
	PacketErrorRate    string `json:"packetErrorRate,omitempty"`
	MaxDataBurstVolume int64  `json:"maxDataBurstVolume,omitempty"`
}

// SliceResources defines resource requirements for a slice
type SliceResources struct {
	CPU         *resource.Quantity `json:"cpu,omitempty"`
	Memory      *resource.Quantity `json:"memory,omitempty"`
	Storage     *resource.Quantity `json:"storage,omitempty"`
	Bandwidth   *resource.Quantity `json:"bandwidth,omitempty"`
	Connections int32              `json:"connections,omitempty"`
}

// IsolationParameters defines isolation requirements
type IsolationParameters struct {
	Level      string   `json:"level"` // physical, logical, none
	Mechanisms []string `json:"mechanisms,omitempty"`
	Encryption bool     `json:"encryption,omitempty"`
}

// GeographicArea defines a geographic area for slice coverage
type GeographicArea struct {
	Type        string              `json:"type"` // country, region, city, custom
	Name        string              `json:"name"`
	Coordinates []Coordinate        `json:"coordinates,omitempty"`
	Coverage    *CoverageParameters `json:"coverage,omitempty"`
}

// Coordinate represents a geographic coordinate
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// CoverageParameters defines coverage requirements
type CoverageParameters struct {
	Indoor   bool   `json:"indoor,omitempty"`
	Outdoor  bool   `json:"outdoor,omitempty"`
	Mobility string `json:"mobility,omitempty"` // stationary, pedestrian, vehicular
	Density  string `json:"density,omitempty"`  // sparse, dense, ultra_dense
}

// SLA requirement types

// LatencyRequirement defines latency requirements
type LatencyRequirement struct {
	MaxLatency     *metav1.Duration `json:"maxLatency"`
	TypicalLatency *metav1.Duration `json:"typicalLatency,omitempty"`
	Percentile     float64          `json:"percentile,omitempty"`
}

// ThroughputRequirement defines throughput requirements
type ThroughputRequirement struct {
	MinDownlink *resource.Quantity `json:"minDownlink,omitempty"`
	MinUplink   *resource.Quantity `json:"minUplink,omitempty"`
	MaxDownlink *resource.Quantity `json:"maxDownlink,omitempty"`
	MaxUplink   *resource.Quantity `json:"maxUplink,omitempty"`
	UserDensity int32              `json:"userDensity,omitempty"`
}

// AvailabilityRequirement defines availability requirements
type AvailabilityRequirement struct {
	Target       float64          `json:"target"` // 0.999, 0.9999, etc.
	ServiceLevel string           `json:"serviceLevel,omitempty"`
	Downtime     *metav1.Duration `json:"downtime,omitempty"`
}

// ReliabilityRequirement defines reliability requirements
type ReliabilityRequirement struct {
	SuccessRate             float64          `json:"successRate"`
	ErrorRate               float64          `json:"errorRate,omitempty"`
	MeanTimeBetweenFailures *metav1.Duration `json:"mtbf,omitempty"`
	MeanTimeToRepair        *metav1.Duration `json:"mttr,omitempty"`
}

// Utility functions and helpers

// GetPackageReference creates a PackageReference from components
func GetPackageReference(repository, packageName, revision string) *PackageReference {
	return &PackageReference{
		Repository:  repository,
		PackageName: packageName,
		Revision:    revision,
	}
}

// GetPackageKey returns a unique key for a package revision
func (pr *PackageReference) GetPackageKey() string {
	return fmt.Sprintf("%s/%s@%s", pr.Repository, pr.PackageName, pr.Revision)
}

// IsValidLifecycle checks if a lifecycle value is valid
func IsValidLifecycle(lifecycle PackageRevisionLifecycle) bool {
	switch lifecycle {
	case PackageRevisionLifecycleDraft,
		PackageRevisionLifecycleProposed,
		PackageRevisionLifecyclePublished,
		PackageRevisionLifecycleDeletable:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if lifecycle transition is valid
func (current PackageRevisionLifecycle) CanTransitionTo(target PackageRevisionLifecycle) bool {
	transitions := map[PackageRevisionLifecycle][]PackageRevisionLifecycle{
		PackageRevisionLifecycleDraft: {
			PackageRevisionLifecycleProposed,
			PackageRevisionLifecycleDeletable,
		},
		PackageRevisionLifecycleProposed: {
			PackageRevisionLifecyclePublished,
			PackageRevisionLifecycleDraft,
			PackageRevisionLifecycleDeletable,
		},
		PackageRevisionLifecyclePublished: {
			PackageRevisionLifecycleDeletable,
		},
		PackageRevisionLifecycleDeletable: {},
	}

	validTargets, exists := transitions[current]
	if !exists {
		return false
	}

	for _, valid := range validTargets {
		if valid == target {
			return true
		}
	}
	return false
}

// Helper methods for working with conditions
func (r *Repository) GetCondition(conditionType string) *metav1.Condition {
	for _, condition := range r.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func (r *Repository) SetCondition(condition metav1.Condition) {
	for i, existing := range r.Status.Conditions {
		if existing.Type == condition.Type {
			r.Status.Conditions[i] = condition
			return
		}
	}
	r.Status.Conditions = append(r.Status.Conditions, condition)
}

func (pr *PackageRevision) GetCondition(conditionType string) *metav1.Condition {
	for _, condition := range pr.Status.Conditions {
		if condition.Type == condition.Type {
			return &condition
		}
	}
	return nil
}

func (pr *PackageRevision) SetCondition(condition metav1.Condition) {
	for i, existing := range pr.Status.Conditions {
		if existing.Type == condition.Type {
			pr.Status.Conditions[i] = condition
			return
		}
	}
	pr.Status.Conditions = append(pr.Status.Conditions, condition)
}

// Implement runtime.Object interface for Kubernetes integration
func (r *Repository) DeepCopyObject() runtime.Object {
	return r.DeepCopy()
}

func (rl *RepositoryList) DeepCopyObject() runtime.Object {
	return rl.DeepCopy()
}

func (pr *PackageRevision) DeepCopyObject() runtime.Object {
	return pr.DeepCopy()
}

func (prl *PackageRevisionList) DeepCopyObject() runtime.Object {
	return prl.DeepCopy()
}

func (w *Workflow) DeepCopyObject() runtime.Object {
	return w.DeepCopy()
}

func (wl *WorkflowList) DeepCopyObject() runtime.Object {
	return wl.DeepCopy()
}

// DeepCopy methods would be generated by code-gen or implemented manually
// For brevity, showing the pattern for Repository
func (r *Repository) DeepCopy() *Repository {
	if r == nil {
		return nil
	}
	out := new(Repository)
	r.DeepCopyInto(out)
	return out
}

func (r *Repository) DeepCopyInto(out *Repository) {
	*out = *r
	out.TypeMeta = r.TypeMeta
	r.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	r.Spec.DeepCopyInto(&out.Spec)
	r.Status.DeepCopyInto(&out.Status)
}

func (rs *RepositorySpec) DeepCopyInto(out *RepositorySpec) {
	*out = *rs
	if rs.Auth != nil {
		out.Auth = rs.Auth.DeepCopy()
	}
	if rs.Sync != nil {
		out.Sync = rs.Sync.DeepCopy()
	}
	if rs.Capabilities != nil {
		out.Capabilities = make([]string, len(rs.Capabilities))
		copy(out.Capabilities, rs.Capabilities)
	}
}

func (rs *RepositoryStatus) DeepCopyInto(out *RepositoryStatus) {
	*out = *rs
	if rs.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(rs.Conditions))
		for i := range rs.Conditions {
			rs.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
	if rs.LastSyncTime != nil {
		out.LastSyncTime = rs.LastSyncTime.DeepCopy()
	}
}

// Additional DeepCopy methods would follow the same pattern...
// This provides type-safe deep copying required for Kubernetes client-go

// ContentMetrics represents metrics for package content
type ContentMetrics struct {
	FileCount      int            `json:"fileCount"`
	TotalSize      int64          `json:"totalSize"`
	ResourceCounts map[string]int `json:"resourceCounts"`
	LastModified   time.Time      `json:"lastModified"`
}

// ContentManagerHealth represents health status of the content manager
type ContentManagerHealth struct {
	Status         string    `json:"status"`
	LastCheck      time.Time `json:"lastCheck"`
	CacheSize      int64     `json:"cacheSize"`
	ActiveRequests int       `json:"activeRequests"`
	ErrorCount     int       `json:"errorCount"`
}

// ContentManagerMetrics tracks metrics for content manager operations
type ContentManagerMetrics struct {
	TotalRequests   int64 `json:"totalRequests"`
	SuccessCount    int64 `json:"successCount"`
	ErrorCount      int64 `json:"errorCount"`
	CacheHits       int64 `json:"cacheHits"`
	CacheMisses     int64 `json:"cacheMisses"`
	AverageLatency  int64 `json:"averageLatency"`
}

// Error types for better error handling
type PorchError struct {
	Type    string
	Message string
	Cause   error
}

func (e *PorchError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *PorchError) Unwrap() error {
	return e.Cause
}

// Common error types
var (
	ErrRepositoryNotFound      = &PorchError{Type: "RepositoryNotFound", Message: "repository not found"}
	ErrPackageNotFound         = &PorchError{Type: "PackageNotFound", Message: "package not found"}
	ErrInvalidLifecycle        = &PorchError{Type: "InvalidLifecycle", Message: "invalid lifecycle transition"}
	ErrFunctionExecutionFailed = &PorchError{Type: "FunctionExecutionFailed", Message: "function execution failed"}
	ErrValidationFailed        = &PorchError{Type: "ValidationFailed", Message: "package validation failed"}
	ErrWorkflowLocked          = &PorchError{Type: "WorkflowLocked", Message: "package is locked by workflow"}
)

// Constants for common values
const (
	// Repository types
	RepositoryTypeGit = "git"
	RepositoryTypeOCI = "oci"

	// Auth types
	AuthTypeBasic = "basic"
	AuthTypeToken = "token"
	AuthTypeSSH   = "ssh"

	// Function types
	FunctionTypeMutator   = "mutator"
	FunctionTypeValidator = "validator"

	// Workflow trigger types
	WorkflowTriggerManual    = "manual"
	WorkflowTriggerAutomatic = "automatic"
	WorkflowTriggerSchedule  = "schedule"

	// Standard annotations
	AnnotationManagedBy      = "porch.nephoran.com/managed-by"
	AnnotationGeneratedBy    = "porch.nephoran.com/generated-by"
	AnnotationIntentID       = "porch.nephoran.com/intent-id"
	AnnotationRepository     = "porch.nephoran.com/repository"
	AnnotationPackageName    = "porch.nephoran.com/package-name"
	AnnotationRevision       = "porch.nephoran.com/revision"
	AnnotationWorkflowID     = "porch.nephoran.com/workflow-id"
	AnnotationORANCompliance = "porch.nephoran.com/oran-compliance"

	// Standard labels
	LabelComponent       = "porch.nephoran.com/component"
	LabelRepository      = "porch.nephoran.com/repository"
	LabelPackageName     = "porch.nephoran.com/package-name"
	LabelRevision        = "porch.nephoran.com/revision"
	LabelLifecycle       = "porch.nephoran.com/lifecycle"
	LabelIntentType      = "porch.nephoran.com/intent-type"
	LabelTargetComponent = "porch.nephoran.com/target-component"
	LabelNetworkSlice    = "porch.nephoran.com/network-slice"
	LabelORANInterface   = "porch.nephoran.com/oran-interface"
)
