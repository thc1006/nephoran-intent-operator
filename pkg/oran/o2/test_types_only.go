package o2

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

// Test-specific types that are not defined elsewhere.

// VNFDeployRequest represents a request to deploy a VNF.
type VNFDeployRequest struct {
	Name            string                       `json:"name"`
	Namespace       string                       `json:"namespace"`
	VNFPackageID    string                       `json:"vnfPackageId"`
	FlavorID        string                       `json:"flavorId"`
	Image           string                       `json:"image"`
	Replicas        int32                        `json:"replicas"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
	Environment     []corev1.EnvVar              `json:"environment,omitempty"`
	Ports           []corev1.ContainerPort       `json:"ports,omitempty"`
	VolumeConfig    []VolumeConfig               `json:"volumeConfig,omitempty"`
	NetworkConfig   *NetworkConfig               `json:"networkConfig,omitempty"`
	SecurityContext *corev1.SecurityContext      `json:"securityContext,omitempty"`
	HealthCheck     *TestHealthCheckConfig       `json:"healthCheck,omitempty"`
	Affinity        *corev1.Affinity             `json:"affinity,omitempty"`
	Tolerations     []corev1.Toleration          `json:"tolerations,omitempty"`
	Metadata        map[string]string            `json:"metadata,omitempty"`
}

// O2VNFDeployRequest is an alias for VNFDeployRequest.
type O2VNFDeployRequest = VNFDeployRequest

// VNFDescriptor describes a VNF for deployment.
type VNFDescriptor struct {
	Name            string                       `json:"name"`
	Type            string                       `json:"type"`
	Version         string                       `json:"version"`
	Vendor          string                       `json:"vendor"`
	Description     string                       `json:"description,omitempty"`
	Image           string                       `json:"image"`
	Replicas        int32                        `json:"replicas"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
	Environment     []corev1.EnvVar              `json:"environment,omitempty"`
	Ports           []corev1.ContainerPort       `json:"ports,omitempty"`
	VolumeConfig    []VolumeConfig               `json:"volumeConfig,omitempty"`
	NetworkConfig   *NetworkConfig               `json:"networkConfig,omitempty"`
	SecurityContext *corev1.SecurityContext      `json:"securityContext,omitempty"`
	HealthCheck     *TestHealthCheckConfig       `json:"healthCheck,omitempty"`
	Affinity        *corev1.Affinity             `json:"affinity,omitempty"`
	Tolerations     []corev1.Toleration          `json:"tolerations,omitempty"`
	Metadata        map[string]string            `json:"metadata,omitempty"`
}

// VNFScaleRequest represents a request to scale a VNF.
type VNFScaleRequest struct {
	ScaleType     string `json:"scaleType"` // SCALE_OUT, SCALE_IN
	NumberOfSteps int32  `json:"numberOfSteps"`
	AspectID      string `json:"aspectId,omitempty"`
}

// O2VNFInstance represents a deployed VNF instance.
type O2VNFInstance struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Namespace        string             `json:"namespace"`
	VNFPackageID     string             `json:"vnfPackageId"`
	FlavorID         string             `json:"flavorId"`
	Status           *VNFInstanceStatus `json:"status"`
	Resources        *ResourceInfo      `json:"resources,omitempty"`
	NetworkEndpoints []NetworkEndpoint  `json:"networkEndpoints,omitempty"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
}

// VNFInstanceStatus represents the status of a VNF instance.
type VNFInstanceStatus struct {
	State         string `json:"state"`         // INSTANTIATED, NOT_INSTANTIATED, TERMINATED
	DetailedState string `json:"detailedState"` // RUNNING, PENDING, ERROR
}

// ResourceInfo represents resource information for a VNF instance.
type ResourceInfo struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// NetworkEndpoint represents a network endpoint for a VNF instance.
type NetworkEndpoint struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

// VolumeConfig represents volume configuration for VNFs.
type VolumeConfig struct {
	Name         string              `json:"name"`
	MountPath    string              `json:"mountPath"`
	VolumeSource corev1.VolumeSource `json:"volumeSource"`
}

// NetworkConfig represents network configuration for VNFs.
type NetworkConfig struct {
	ServiceType corev1.ServiceType   `json:"serviceType"`
	Ports       []corev1.ServicePort `json:"ports"`
}

// TestHealthCheckConfig represents health check configuration for VNFs (test-specific).
type TestHealthCheckConfig struct {
	LivenessProbe  *corev1.Probe `json:"livenessProbe,omitempty"`
	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`
	StartupProbe   *corev1.Probe `json:"startupProbe,omitempty"`
}

// InfrastructureInfo represents infrastructure information.
type InfrastructureInfo struct {
	NodeCount         int              `json:"nodeCount"`
	ClusterName       string           `json:"clusterName"`
	KubernetesVersion string           `json:"kubernetesVersion"`
	TotalResources    *ResourceSummary `json:"totalResources,omitempty"`
}

// ResourceSummary represents a summary of resources.
type ResourceSummary struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// ResourceMap represents discovered resources.
type ResourceMap struct {
	Nodes      map[string]*NodeInfo      `json:"nodes"`
	Namespaces map[string]*NamespaceInfo `json:"namespaces"`
	Metrics    *ClusterMetrics           `json:"metrics"`
}

// NodeInfo represents information about a node.
type NodeInfo struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Roles  []string          `json:"roles"`
}

// NamespaceInfo represents information about a namespace.
type NamespaceInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ClusterMetrics represents cluster metrics.
type ClusterMetrics struct {
	ReadyNodes  int32  `json:"readyNodes"`
	TotalCPU    string `json:"totalCPU"`
	TotalMemory string `json:"totalMemory"`
}

// VNFDeploymentStatus represents the status of a VNF deployment.
type VNFDeploymentStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Phase    string `json:"phase"`
	Replicas int32  `json:"replicas"`
}
