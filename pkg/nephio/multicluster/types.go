package multicluster

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Common types for multicluster package management
// Note: ResourceUtilization is defined in cluster_manager.go to avoid redeclaration

// DependencyHealthReport provides health status for dependencies
type DependencyHealthReport struct {
	ComponentName    string
	Status           HealthStatus
	LastCheck        time.Time
	ErrorMessage     string
	Dependencies     []string
	RecommendedAction string
}

// MaintenanceWindow defines scheduled maintenance periods
type MaintenanceWindow struct {
	Name        string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	Frequency   string // "weekly", "monthly", etc.
	Clusters    []types.NamespacedName
	Operations  []string
}

// AlertThresholds defines monitoring thresholds
type AlertThresholds struct {
	CPUWarning       float64
	CPUCritical      float64
	MemoryWarning    float64
	MemoryCritical   float64
	DiskWarning      float64
	DiskCritical     float64
	ResponseTimeMs   int64
	ErrorRatePercent float64
}

// BusinessImpact describes the potential impact of operations
type BusinessImpact struct {
	Severity          string // "low", "medium", "high", "critical"
	AffectedServices  []string
	EstimatedDowntime time.Duration
	UserImpact        string
	RevenueImpact     string
	MitigationPlan    []string
}

// Recommendation provides automated recommendations for operations
type Recommendation struct {
	ID          string
	Type        string // "scaling", "maintenance", "optimization", "security"
	Priority    int    // 1-5, where 1 is highest
	Title       string
	Description string
	Actions     []RecommendationAction
	Benefits    []string
	Risks       []string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// RecommendationAction defines a specific action to take
type RecommendationAction struct {
	Step        int
	Description string
	Command     string
	Parameters  map[string]interface{}
	Validation  string
}

// HealthStatus represents the overall health of components (already exists in health_monitor.go)
// Keep this here for reference but don't duplicate
// type HealthStatus string

// Alert types and severity (already exists in health_monitor.go)
// Keep these here for reference but don't duplicate
// type Alert struct { ... }
// type AlertSeverity string
// type AlertType string

// Nephio-specific types (since external dependency is not available)

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusPending   DeploymentStatus = "Pending"
	DeploymentStatusRunning   DeploymentStatus = "Running"
	DeploymentStatusSucceeded DeploymentStatus = "Succeeded"
	DeploymentStatusFailed    DeploymentStatus = "Failed"
)

// ClusterDeploymentStatus represents the deployment status for a specific cluster
type ClusterDeploymentStatus struct {
	ClusterName string
	Status      DeploymentStatus
	Timestamp   time.Time
	Message     string
	Errors      []string
}

// MultiClusterDeploymentStatus represents the overall deployment status across multiple clusters
type MultiClusterDeploymentStatus struct {
	Clusters     map[string]ClusterDeploymentStatus
	OverallStatus DeploymentStatus
	StartTime    time.Time
	EndTime      time.Time
	Summary      string
}

// Porch-like API types for package management
// These types follow the Porch/kpt patterns but are defined locally

// PackageRevisionLifecycle represents the lifecycle state of a package revision
type PackageRevisionLifecycle string

const (
	PackageRevisionLifecycleDraft     PackageRevisionLifecycle = "Draft"
	PackageRevisionLifecycleProposed  PackageRevisionLifecycle = "Proposed"
	PackageRevisionLifecyclePublished PackageRevisionLifecycle = "Published"
)

// PackageRevisionSpec defines the desired state of PackageRevision
type PackageRevisionSpec struct {
	PackageName    string                   `json:"packageName,omitempty"`
	Revision       string                   `json:"revision,omitempty"`
	Lifecycle      PackageRevisionLifecycle `json:"lifecycle,omitempty"`
	Repository     string                   `json:"repository,omitempty"`
	WorkspaceName  string                   `json:"workspaceName,omitempty"`
	Tasks          []Task                   `json:"tasks,omitempty"`
}

// PackageRevisionStatus defines the observed state of PackageRevision
type PackageRevisionStatus struct {
	Conditions      []metav1.Condition `json:"conditions,omitempty"`
	UpstreamLock    *UpstreamLock      `json:"upstreamLock,omitempty"`
	PublishedBy     string             `json:"publishedBy,omitempty"`
	PublishedAt     metav1.Time        `json:"publishedAt,omitempty"`
	DeploymentReady bool               `json:"deploymentReady,omitempty"`
}

// PackageRevision represents a revision of a package
type PackageRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageRevisionSpec   `json:"spec,omitempty"`
	Status PackageRevisionStatus `json:"status,omitempty"`
}

// Task represents a configuration transformation task
type Task struct {
	Type   string                 `json:"type,omitempty"`
	Image  string                 `json:"image,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// UpstreamLock contains information about the upstream source
type UpstreamLock struct {
	Type string  `json:"type,omitempty"`
	Git  GitLock `json:"git,omitempty"`
}

// GitLock contains Git-specific upstream information
type GitLock struct {
	Repo      string `json:"repo,omitempty"`
	Directory string `json:"directory,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Commit    string `json:"commit,omitempty"`
}