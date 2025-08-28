package multicluster

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ClusterStatus represents the current state of a workload cluster.
type ClusterStatus string

const (
	// ClusterStatusHealthy holds clusterstatushealthy value.
	ClusterStatusHealthy ClusterStatus = "Healthy"
	// ClusterStatusDegraded holds clusterstatusdegraded value.
	ClusterStatusDegraded ClusterStatus = "Degraded"
	// ClusterStatusUnreachable holds clusterstatusunreachable value.
	ClusterStatusUnreachable ClusterStatus = "Unreachable"
)

// ResourceCapacity represents the computational resources of a cluster.
type ResourceCapacity struct {
	CPU              int64 // millicores
	Memory           int64 // bytes
	StorageGB        int64
	EphemeralStorage int64 // bytes
}

// EdgeLocation provides geographic and network context for edge clusters.
type EdgeLocation struct {
	Region           string
	Zone             string
	Latitude         float64
	Longitude        float64
	NetworkLatencyMS float64
}

// WorkloadCluster represents a managed Kubernetes cluster.
type WorkloadCluster struct {
	// Unique identifier for the cluster.
	ID string

	// Cluster metadata.
	Name       string
	KubeConfig *rest.Config
	Client     *kubernetes.Clientset
	Region     string
	Zone       string

	// Capabilities and characteristics.
	EdgeLocation *EdgeLocation
	Capabilities []string
	Resources    *ResourceCapacity

	// Current cluster status.
	Status        ClusterStatus
	LastCheckedAt time.Time

	// Additional metadata.
	Labels      map[string]string
	Annotations map[string]string
}

// ClusterRegistrationOptions provides configuration for cluster registration.
type ClusterRegistrationOptions struct {
	// Optional timeout for cluster connection and health check.
	ConnectionTimeout time.Duration

	// Number of retries for cluster connection.
	ConnectionRetries int

	// Validation and security options.
	RequiredCapabilities     []string
	MinimumResourceThreshold *ResourceCapacity
}

// NetworkTopology represents the network relationships between clusters.
type NetworkTopology struct {
	Clusters        map[string]*WorkloadCluster
	LatencyMatrix   map[string]map[string]float64
	NetworkPolicies []NetworkPolicy
}

// NetworkPolicy defines network constraints and routing rules.
type NetworkPolicy struct {
	Source           string
	Destination      string
	AllowedProtocols []string
	MaxLatencyMS     float64
	Bandwidth        int64 // Mbps
}

// DeploymentTarget represents a selected cluster for package deployment.
type DeploymentTarget struct {
	Cluster     *WorkloadCluster
	Constraints []PlacementConstraint
	Priority    int
	Fitness     float64 // Deployment suitability score
}

// PlacementConstraint defines rules for package deployment.
type PlacementConstraint struct {
	Type        string
	Value       string
	Requirement ConstraintType
}

// ConstraintType defines how a constraint should be evaluated.
type ConstraintType string

const (
	// ConstraintMust holds constraintmust value.
	ConstraintMust ConstraintType = "Must"
	// ConstraintPreferred holds constraintpreferred value.
	ConstraintPreferred ConstraintType = "Preferred"
	// ConstraintAvoid holds constraintavoid value.
	ConstraintAvoid ConstraintType = "Avoid"
)

// PropagationResult captures the outcome of package deployment.
type PropagationResult struct {
	PackageName           string
	TargetClusters        []string
	SuccessfulDeployments []string
	FailedDeployments     []string
	Timestamp             metav1.Time
	TotalLatencyMS        float64
}

// DeploymentStrategy defines how packages are propagated.
type DeploymentStrategy struct {
	Type                  string
	RolloutPercentage     int
	MaxConcurrentClusters int
	RollbackOnFailure     bool
}

// MultiClusterStatus aggregates deployment status across clusters.
type MultiClusterStatus struct {
	PackageName     string
	OverallStatus   string
	ClusterStatuses map[string]ClusterDeploymentStatus
	LastUpdated     metav1.Time
}

// ClusterDeploymentStatus represents the deployment status for a specific cluster.
type ClusterDeploymentStatus struct {
	ClusterName  string
	Status       string
	LastUpdated  metav1.Time
	ErrorMessage string
}

// PropagationOptions provides configuration for package deployment.
type PropagationOptions struct {
	Strategy     DeploymentStrategy
	Constraints  []PlacementConstraint
	ValidateOnly bool
	DryRun       bool
}

// ClusterPropagationManager interface defines multi-cluster package management.
type ClusterPropagationManager interface {
	PropagatePackage(ctx context.Context, packageName string, options *PropagationOptions) (*PropagationResult, error)
	GetMultiClusterStatus(ctx context.Context, packageName string) (*MultiClusterStatus, error)
	RegisterCluster(ctx context.Context, cluster *WorkloadCluster, options *ClusterRegistrationOptions) error
	UnregisterCluster(ctx context.Context, clusterID string) error
}
