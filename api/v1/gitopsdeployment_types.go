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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// GitOpsDeploymentPhase represents the phase of GitOps deployment
type GitOpsDeploymentPhase string

const (
	// GitOpsDeploymentPhasePending indicates deployment is pending
	GitOpsDeploymentPhasePending GitOpsDeploymentPhase = "Pending"
	// GitOpsDeploymentPhaseCommitting indicates Git commit is in progress
	GitOpsDeploymentPhaseCommitting GitOpsDeploymentPhase = "Committing"
	// GitOpsDeploymentPhaseSyncing indicates sync with Git is in progress
	GitOpsDeploymentPhaseSyncing GitOpsDeploymentPhase = "Syncing"
	// GitOpsDeploymentPhaseDeploying indicates deployment is in progress
	GitOpsDeploymentPhaseDeploying GitOpsDeploymentPhase = "Deploying"
	// GitOpsDeploymentPhaseVerifying indicates verification is in progress
	GitOpsDeploymentPhaseVerifying GitOpsDeploymentPhase = "Verifying"
	// GitOpsDeploymentPhaseCompleted indicates deployment is completed
	GitOpsDeploymentPhaseCompleted GitOpsDeploymentPhase = "Completed"
	// GitOpsDeploymentPhaseFailed indicates deployment has failed
	GitOpsDeploymentPhaseFailed GitOpsDeploymentPhase = "Failed"
	// GitOpsDeploymentPhaseRollingBack indicates rollback is in progress
	GitOpsDeploymentPhaseRollingBack GitOpsDeploymentPhase = "RollingBack"
)

// GitProvider represents the Git provider
type GitProvider string

const (
	// GitProviderGitHub represents GitHub
	GitProviderGitHub GitProvider = "github"
	// GitProviderGitLab represents GitLab
	GitProviderGitLab GitProvider = "gitlab"
	// GitProviderBitbucket represents Bitbucket
	GitProviderBitbucket GitProvider = "bitbucket"
	// GitProviderAzureDevOps represents Azure DevOps
	GitProviderAzureDevOps GitProvider = "azuredevops"
	// GitProviderGeneric represents generic Git
	GitProviderGeneric GitProvider = "generic"
)

// DeploymentStrategy represents the deployment strategy
type DeploymentStrategy string

const (
	// DeploymentStrategyRecreate recreates all resources
	DeploymentStrategyRecreate DeploymentStrategy = "Recreate"
	// DeploymentStrategyRollingUpdate performs rolling updates
	DeploymentStrategyRollingUpdate DeploymentStrategy = "RollingUpdate"
	// DeploymentStrategyCanary performs canary deployments
	DeploymentStrategyCanary DeploymentStrategy = "Canary"
	// DeploymentStrategyBlueGreen performs blue-green deployments
	DeploymentStrategyBlueGreen DeploymentStrategy = "BlueGreen"
	// DeploymentStrategyHelm uses Helm for deployment
	DeploymentStrategyHelm DeploymentStrategy = "Helm"
	// DeploymentStrategyOperator uses Kubernetes operators for deployment
	DeploymentStrategyOperator DeploymentStrategy = "Operator"
	// DeploymentStrategyDirect uses direct Kubernetes manifests
	DeploymentStrategyDirect DeploymentStrategy = "Direct"
	// DeploymentStrategyGitOps uses GitOps workflow for deployment
	DeploymentStrategyGitOps DeploymentStrategy = "GitOps"
)

// GitOpsDeploymentSpec defines the desired state of GitOpsDeployment
type GitOpsDeploymentSpec struct {
	// ParentIntentRef references the parent NetworkIntent
	// +kubebuilder:validation:Required
	ParentIntentRef ObjectReference `json:"parentIntentRef"`

	// ManifestGenerationRef references the ManifestGeneration resource
	// +optional
	ManifestGenerationRef *ObjectReference `json:"manifestGenerationRef,omitempty"`

	// Manifests contains the manifests to deploy
	// +kubebuilder:validation:Required
	Manifests map[string]string `json:"manifests"`

	// GitRepository defines the Git repository configuration
	// +kubebuilder:validation:Required
	GitRepository *GitRepositoryConfig `json:"gitRepository"`

	// TargetClusters specifies the target clusters for deployment
	// +optional
	TargetClusters []TargetCluster `json:"targetClusters,omitempty"`

	// DeploymentStrategy specifies the deployment strategy
	// +optional
	// +kubebuilder:default="RollingUpdate"
	DeploymentStrategy DeploymentStrategy `json:"deploymentStrategy,omitempty"`

	// DeploymentConfig contains deployment-specific configuration
	// +optional
	DeploymentConfig *DeploymentConfig `json:"deploymentConfig,omitempty"`

	// Priority defines deployment priority
	// +optional
	// +kubebuilder:default="medium"
	Priority Priority `json:"priority,omitempty"`

	// AutoSync enables automatic synchronization
	// +optional
	// +kubebuilder:default=true
	AutoSync *bool `json:"autoSync,omitempty"`

	// SyncPolicy defines synchronization policy
	// +optional
	SyncPolicy *SyncPolicy `json:"syncPolicy,omitempty"`

	// HealthChecks defines health check configuration
	// +optional
	HealthChecks *HealthCheckConfig `json:"healthChecks,omitempty"`

	// RollbackConfig defines rollback configuration
	// +optional
	RollbackConfig *RollbackConfig `json:"rollbackConfig,omitempty"`
}

// GitRepositoryConfig defines Git repository configuration
type GitRepositoryConfig struct {
	// URL of the Git repository
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Provider specifies the Git provider
	// +optional
	// +kubebuilder:default="generic"
	Provider GitProvider `json:"provider,omitempty"`

	// Branch to commit to
	// +optional
	// +kubebuilder:default="main"
	Branch string `json:"branch,omitempty"`

	// Path within the repository
	// +optional
	// +kubebuilder:default="deployments"
	Path string `json:"path,omitempty"`

	// SecretRef for Git authentication
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// CommitMessage template
	// +optional
	CommitMessage string `json:"commitMessage,omitempty"`

	// AuthorName for Git commits
	// +optional
	// +kubebuilder:default="nephoran-intent-operator"
	AuthorName string `json:"authorName,omitempty"`

	// AuthorEmail for Git commits
	// +optional
	// +kubebuilder:default="noreply@nephoran.com"
	AuthorEmail string `json:"authorEmail,omitempty"`

	// SignCommits enables commit signing
	// +optional
	// +kubebuilder:default=false
	SignCommits *bool `json:"signCommits,omitempty"`
}

// TargetCluster defines a target cluster for deployment
type TargetCluster struct {
	// Name of the cluster
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace for deployment in the cluster
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Server URL of the cluster
	// +optional
	Server string `json:"server,omitempty"`

	// Config contains cluster-specific configuration
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config runtime.RawExtension `json:"config,omitempty"`

	// SecretRef for cluster authentication
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`

	// Labels for cluster identification
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Priority for deployment order
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	Priority *int `json:"priority,omitempty"`
}

// DeploymentConfig contains deployment-specific configuration
type DeploymentConfig struct {
	// Timeout for deployment operations
	// +optional
	// +kubebuilder:default=600
	// +kubebuilder:validation:Minimum=60
	// +kubebuilder:validation:Maximum=3600
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty"`

	// MaxRetries for failed operations
	// +optional
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	MaxRetries *int32 `json:"maxRetries,omitempty"`

	// RetryBackoff defines retry backoff strategy
	// +optional
	RetryBackoff *RetryBackoffConfig `json:"retryBackoff,omitempty"`

	// ResourceOrder defines the order of resource deployment
	// +optional
	ResourceOrder []string `json:"resourceOrder,omitempty"`

	// DependencyChecks defines dependency validation
	// +optional
	DependencyChecks *DependencyCheckConfig `json:"dependencyChecks,omitempty"`

	// PreDeploymentHooks defines pre-deployment hooks
	// +optional
	PreDeploymentHooks []DeploymentHook `json:"preDeploymentHooks,omitempty"`

	// PostDeploymentHooks defines post-deployment hooks
	// +optional
	PostDeploymentHooks []DeploymentHook `json:"postDeploymentHooks,omitempty"`
}

// RetryBackoffConfig defines retry backoff configuration
type RetryBackoffConfig struct {
	// InitialInterval for first retry
	// +optional
	// +kubebuilder:default="30s"
	InitialInterval string `json:"initialInterval,omitempty"`

	// MaxInterval caps the backoff interval
	// +optional
	// +kubebuilder:default="300s"
	MaxInterval string `json:"maxInterval,omitempty"`

	// Multiplier for exponential backoff
	// +optional
	// +kubebuilder:default=2.0
	Multiplier *float64 `json:"multiplier,omitempty"`
}

// DependencyCheckConfig defines dependency validation configuration
type DependencyCheckConfig struct {
	// Enabled determines if dependency checks are enabled
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// Timeout for dependency checks
	// +optional
	// +kubebuilder:default="120s"
	Timeout string `json:"timeout,omitempty"`

	// Dependencies lists the dependencies to check
	// +optional
	Dependencies []ResourceDependency `json:"dependencies,omitempty"`
}

// ResourceDependency defines a resource dependency
type ResourceDependency struct {
	// Name of the dependency
	Name string `json:"name"`

	// APIVersion of the dependency
	APIVersion string `json:"apiVersion"`

	// Kind of the dependency
	Kind string `json:"kind"`

	// Namespace of the dependency
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Condition to check for the dependency
	// +optional
	Condition string `json:"condition,omitempty"`
}

// DeploymentHook defines a deployment hook
type DeploymentHook struct {
	// Name of the hook
	Name string `json:"name"`

	// Type of hook (job, script, webhook, etc.)
	Type string `json:"type"`

	// Configuration for the hook
	// +kubebuilder:pruning:PreserveUnknownFields
	Configuration runtime.RawExtension `json:"configuration"`

	// Timeout for hook execution
	// +optional
	// +kubebuilder:default="300s"
	Timeout string `json:"timeout,omitempty"`

	// OnFailure defines behavior on hook failure
	// +optional
	// +kubebuilder:default="abort"
	// +kubebuilder:validation:Enum=abort;continue;retry
	OnFailure string `json:"onFailure,omitempty"`
}

// SyncPolicy defines synchronization policy
type SyncPolicy struct {
	// Automated enables automated synchronization
	// +optional
	Automated *AutomatedSyncPolicy `json:"automated,omitempty"`

	// SyncOptions defines sync options
	// +optional
	SyncOptions []string `json:"syncOptions,omitempty"`

	// Retry defines retry policy
	// +optional
	Retry *RetryPolicy `json:"retry,omitempty"`
}

// AutomatedSyncPolicy defines automated synchronization policy
type AutomatedSyncPolicy struct {
	// Prune enables pruning of resources
	// +optional
	// +kubebuilder:default=false
	Prune *bool `json:"prune,omitempty"`

	// SelfHeal enables self-healing
	// +optional
	// +kubebuilder:default=true
	SelfHeal *bool `json:"selfHeal,omitempty"`

	// AllowEmpty allows syncing with empty repository
	// +optional
	// +kubebuilder:default=false
	AllowEmpty *bool `json:"allowEmpty,omitempty"`
}

// RetryPolicy defines retry policy for sync operations
type RetryPolicy struct {
	// Limit is the maximum number of retries
	// +optional
	// +kubebuilder:default=5
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=20
	Limit *int64 `json:"limit,omitempty"`

	// Backoff defines the retry backoff strategy
	// +optional
	Backoff *RetryBackoff `json:"backoff,omitempty"`
}

// RetryBackoff defines retry backoff strategy
type RetryBackoff struct {
	// Duration is the initial backoff duration
	// +optional
	// +kubebuilder:default="5s"
	Duration string `json:"duration,omitempty"`

	// Factor is the backoff multiplier
	// +optional
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=1
	Factor *int64 `json:"factor,omitempty"`

	// MaxDuration caps the backoff duration
	// +optional
	// +kubebuilder:default="3m"
	MaxDuration string `json:"maxDuration,omitempty"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	// Enabled determines if health checks are enabled
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// Timeout for health checks
	// +optional
	// +kubebuilder:default="300s"
	Timeout string `json:"timeout,omitempty"`

	// CheckInterval defines how often to check health
	// +optional
	// +kubebuilder:default="30s"
	CheckInterval string `json:"checkInterval,omitempty"`

	// Checks defines specific health checks
	// +optional
	Checks []HealthCheck `json:"checks,omitempty"`
}

// HealthCheck defines a specific health check
type HealthCheck struct {
	// Name of the health check
	Name string `json:"name"`

	// Type of health check (readiness, liveness, custom)
	Type string `json:"type"`

	// Resource to check
	Resource ResourceReference `json:"resource"`

	// Configuration for the health check
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Configuration runtime.RawExtension `json:"configuration,omitempty"`
}

// RollbackConfig defines rollback configuration
type RollbackConfig struct {
	// Enabled determines if rollback is enabled
	// +optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`

	// AutoRollback enables automatic rollback on failure
	// +optional
	// +kubebuilder:default=false
	AutoRollback *bool `json:"autoRollback,omitempty"`

	// Trigger defines rollback triggers
	// +optional
	Trigger *RollbackTrigger `json:"trigger,omitempty"`

	// Strategy defines rollback strategy
	// +optional
	// +kubebuilder:default="immediate"
	// +kubebuilder:validation:Enum=immediate;gradual;manual
	Strategy string `json:"strategy,omitempty"`

	// Timeout for rollback operations
	// +optional
	// +kubebuilder:default="300s"
	Timeout string `json:"timeout,omitempty"`
}

// RollbackTrigger defines rollback triggers
type RollbackTrigger struct {
	// OnFailure triggers rollback on deployment failure
	// +optional
	// +kubebuilder:default=true
	OnFailure *bool `json:"onFailure,omitempty"`

	// OnHealthCheckFailure triggers rollback on health check failure
	// +optional
	// +kubebuilder:default=false
	OnHealthCheckFailure *bool `json:"onHealthCheckFailure,omitempty"`

	// ErrorThreshold defines error threshold for rollback
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	ErrorThreshold *int `json:"errorThreshold,omitempty"`

	// TimeWindow defines time window for error counting
	// +optional
	// +kubebuilder:default="5m"
	TimeWindow string `json:"timeWindow,omitempty"`
}

// GitOpsDeploymentStatus defines the observed state of GitOpsDeployment
type GitOpsDeploymentStatus struct {
	// Phase represents the current deployment phase
	// +optional
	Phase GitOpsDeploymentPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// DeploymentStartTime indicates when deployment started
	// +optional
	DeploymentStartTime *metav1.Time `json:"deploymentStartTime,omitempty"`

	// DeploymentCompletionTime indicates when deployment completed
	// +optional
	DeploymentCompletionTime *metav1.Time `json:"deploymentCompletionTime,omitempty"`

	// GitCommit contains Git commit information
	// +optional
	GitCommit *GitCommitInfo `json:"gitCommit,omitempty"`

	// SyncStatus contains synchronization status
	// +optional
	SyncStatus *SyncStatus `json:"syncStatus,omitempty"`

	// DeploymentStatus contains deployment status per cluster
	// +optional
	DeploymentStatus []ClusterDeploymentStatus `json:"deploymentStatus,omitempty"`

	// HealthStatus contains health check results
	// +optional
	HealthStatus *HealthStatus `json:"healthStatus,omitempty"`

	// ResourceStatus contains status of deployed resources
	// +optional
	ResourceStatus []DeployedResourceStatus `json:"resourceStatus,omitempty"`

	// RollbackInfo contains rollback information
	// +optional
	RollbackInfo *RollbackInfo `json:"rollbackInfo,omitempty"`

	// DeploymentDuration represents total deployment time
	// +optional
	DeploymentDuration *metav1.Duration `json:"deploymentDuration,omitempty"`

	// RetryCount tracks retry attempts
	// +optional
	RetryCount int32 `json:"retryCount,omitempty"`

	// LastSyncTime indicates last synchronization time
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// ObservedGeneration reflects the generation observed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// GitCommitInfo contains Git commit information
type GitCommitInfo struct {
	// Hash is the Git commit hash
	Hash string `json:"hash"`

	// Message is the commit message
	Message string `json:"message"`

	// Author is the commit author
	Author string `json:"author"`

	// Timestamp is the commit timestamp
	Timestamp metav1.Time `json:"timestamp"`

	// Branch is the target branch
	Branch string `json:"branch"`

	// URL is the commit URL
	// +optional
	URL string `json:"url,omitempty"`

	// Files lists the committed files
	// +optional
	Files []string `json:"files,omitempty"`
}

// SyncStatus contains synchronization status
type SyncStatus struct {
	// Status is the overall sync status
	Status string `json:"status"`

	// Health is the overall health status
	// +optional
	Health string `json:"health,omitempty"`

	// Revision is the Git revision being synced
	// +optional
	Revision string `json:"revision,omitempty"`

	// ComparedTo contains comparison information
	// +optional
	ComparedTo *ComparedToInfo `json:"comparedTo,omitempty"`

	// Resources lists the sync status of resources
	// +optional
	Resources []ResourceSyncStatus `json:"resources,omitempty"`
}

// ComparedToInfo contains comparison information
type ComparedToInfo struct {
	// Source contains source information
	Source SourceInfo `json:"source"`

	// Destination contains destination information
	Destination DestinationInfo `json:"destination"`
}

// SourceInfo contains source information
type SourceInfo struct {
	// RepoURL is the repository URL
	RepoURL string `json:"repoURL"`

	// Path is the path within the repository
	Path string `json:"path"`

	// TargetRevision is the target revision
	TargetRevision string `json:"targetRevision"`
}

// DestinationInfo contains destination information
type DestinationInfo struct {
	// Server is the cluster server
	Server string `json:"server"`

	// Namespace is the target namespace
	Namespace string `json:"namespace"`
}

// ResourceSyncStatus contains resource sync status
type ResourceSyncStatus struct {
	// Name of the resource
	Name string `json:"name"`

	// Kind of the resource
	Kind string `json:"kind"`

	// Namespace of the resource
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Status is the sync status
	Status string `json:"status"`

	// Health is the health status
	// +optional
	Health string `json:"health,omitempty"`

	// Message contains status message
	// +optional
	Message string `json:"message,omitempty"`
}

// ClusterDeploymentStatus contains deployment status for a cluster
type ClusterDeploymentStatus struct {
	// ClusterName is the name of the cluster
	ClusterName string `json:"clusterName"`

	// Status is the deployment status
	Status string `json:"status"`

	// Message contains status message
	// +optional
	Message string `json:"message,omitempty"`

	// LastUpdateTime is the last update time
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// DeployedResources lists deployed resources
	// +optional
	DeployedResources []ResourceReference `json:"deployedResources,omitempty"`

	// FailedResources lists failed resources
	// +optional
	FailedResources []ResourceReference `json:"failedResources,omitempty"`
}

// HealthStatus contains health check results
type HealthStatus struct {
	// OverallStatus is the overall health status
	OverallStatus string `json:"overallStatus"`

	// CheckResults contains individual check results
	// +optional
	CheckResults []HealthCheckResult `json:"checkResults,omitempty"`

	// LastCheckTime is the last health check time
	// +optional
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// HealthScore represents overall health score (0.0-1.0)
	// +optional
	HealthScore *float64 `json:"healthScore,omitempty"`
}

// HealthCheckResult contains result of a health check
type HealthCheckResult struct {
	// Name of the health check
	Name string `json:"name"`

	// Status is the health check status
	Status string `json:"status"`

	// Message contains check result message
	// +optional
	Message string `json:"message,omitempty"`

	// LastCheckTime is the last check time
	// +optional
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// Duration is the check duration
	// +optional
	Duration *metav1.Duration `json:"duration,omitempty"`
}

// DeployedResourceStatus contains status of a deployed resource
type DeployedResourceStatus struct {
	// ResourceReference identifies the resource
	ResourceReference `json:",inline"`

	// Status is the resource status
	Status string `json:"status"`

	// Health is the resource health
	// +optional
	Health string `json:"health,omitempty"`

	// Message contains status message
	// +optional
	Message string `json:"message,omitempty"`

	// SyncWave is the sync wave of the resource
	// +optional
	SyncWave *int `json:"syncWave,omitempty"`

	// CreatedAt is the creation time
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
}

// RollbackInfo contains rollback information
type RollbackInfo struct {
	// Initiated indicates if rollback was initiated
	Initiated bool `json:"initiated"`

	// Reason for rollback
	// +optional
	Reason string `json:"reason,omitempty"`

	// InitiatedAt is the rollback initiation time
	// +optional
	InitiatedAt *metav1.Time `json:"initiatedAt,omitempty"`

	// CompletedAt is the rollback completion time
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// Status is the rollback status
	// +optional
	Status string `json:"status,omitempty"`

	// TargetRevision is the revision being rolled back to
	// +optional
	TargetRevision string `json:"targetRevision,omitempty"`

	// OriginalRevision is the original revision
	// +optional
	OriginalRevision string `json:"originalRevision,omitempty"`
}

// ResourceReference represents a reference to a Kubernetes resource
type ResourceReference struct {
	// APIVersion of the resource
	APIVersion string `json:"apiVersion"`

	// Kind of the resource
	Kind string `json:"kind"`

	// Name of the resource
	Name string `json:"name"`

	// Namespace of the resource
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// UID of the resource
	// +optional
	UID string `json:"uid,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Parent Intent",type=string,JSONPath=`.spec.parentIntentRef.name`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Strategy",type=string,JSONPath=`.spec.deploymentStrategy`
//+kubebuilder:printcolumn:name="Commit",type=string,JSONPath=`.status.gitCommit.hash`
//+kubebuilder:printcolumn:name="Sync Status",type=string,JSONPath=`.status.syncStatus.status`
//+kubebuilder:printcolumn:name="Health",type=string,JSONPath=`.status.healthStatus.overallStatus`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=gd;gitopsdep
//+kubebuilder:storageversion

// GitOpsDeployment is the Schema for the gitopsdeployments API
type GitOpsDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitOpsDeploymentSpec   `json:"spec,omitempty"`
	Status GitOpsDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GitOpsDeploymentList contains a list of GitOpsDeployment
type GitOpsDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitOpsDeployment `json:"items"`
}

// GetParentIntentName returns the name of the parent NetworkIntent
func (gd *GitOpsDeployment) GetParentIntentName() string {
	return gd.Spec.ParentIntentRef.Name
}

// GetParentIntentNamespace returns the namespace of the parent NetworkIntent
func (gd *GitOpsDeployment) GetParentIntentNamespace() string {
	if gd.Spec.ParentIntentRef.Namespace != "" {
		return gd.Spec.ParentIntentRef.Namespace
	}
	return gd.GetNamespace()
}

// IsDeploymentComplete returns true if deployment is complete
func (gd *GitOpsDeployment) IsDeploymentComplete() bool {
	return gd.Status.Phase == GitOpsDeploymentPhaseCompleted
}

// IsDeploymentFailed returns true if deployment has failed
func (gd *GitOpsDeployment) IsDeploymentFailed() bool {
	return gd.Status.Phase == GitOpsDeploymentPhaseFailed
}

// GetCommitHash returns the Git commit hash
func (gd *GitOpsDeployment) GetCommitHash() string {
	if gd.Status.GitCommit != nil {
		return gd.Status.GitCommit.Hash
	}
	return ""
}

// GetSyncStatus returns the synchronization status
func (gd *GitOpsDeployment) GetSyncStatus() string {
	if gd.Status.SyncStatus != nil {
		return gd.Status.SyncStatus.Status
	}
	return "Unknown"
}

// GetHealthStatus returns the overall health status
func (gd *GitOpsDeployment) GetHealthStatus() string {
	if gd.Status.HealthStatus != nil {
		return gd.Status.HealthStatus.OverallStatus
	}
	return "Unknown"
}

// IsAutoSyncEnabled returns true if auto-sync is enabled
func (gd *GitOpsDeployment) IsAutoSyncEnabled() bool {
	if gd.Spec.AutoSync == nil {
		return true
	}
	return *gd.Spec.AutoSync
}

// IsHealthCheckEnabled returns true if health checks are enabled
func (gd *GitOpsDeployment) IsHealthCheckEnabled() bool {
	if gd.Spec.HealthChecks == nil || gd.Spec.HealthChecks.Enabled == nil {
		return true
	}
	return *gd.Spec.HealthChecks.Enabled
}

// IsRollbackEnabled returns true if rollback is enabled
func (gd *GitOpsDeployment) IsRollbackEnabled() bool {
	if gd.Spec.RollbackConfig == nil || gd.Spec.RollbackConfig.Enabled == nil {
		return true
	}
	return *gd.Spec.RollbackConfig.Enabled
}

// GetTargetClusterCount returns the number of target clusters
func (gd *GitOpsDeployment) GetTargetClusterCount() int {
	return len(gd.Spec.TargetClusters)
}

// GetDeployedResourceCount returns the number of deployed resources
func (gd *GitOpsDeployment) GetDeployedResourceCount() int {
	return len(gd.Status.ResourceStatus)
}

func init() {
	SchemeBuilder.Register(&GitOpsDeployment{}, &GitOpsDeploymentList{})
}
