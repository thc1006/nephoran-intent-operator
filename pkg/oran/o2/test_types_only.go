package o2

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Test-specific types that are not defined elsewhere.

// VNFDeployRequest represents a request to deploy a VNF.

type VNFDeployRequest struct {
	Name string `json:"name"`

	Namespace string `json:"namespace"`

	VNFPackageID string `json:"vnfPackageId"`

	FlavorID string `json:"flavorId"`

	Image string `json:"image"`

	Replicas int32 `json:"replicas"`

	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	Environment []corev1.EnvVar `json:"environment,omitempty"`

	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	VolumeConfig []VolumeConfig `json:"volumeConfig,omitempty"`

	NetworkConfig *NetworkConfig `json:"networkConfig,omitempty"`

	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	HealthCheck *TestHealthCheckConfig `json:"healthCheck,omitempty"`

	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// O2VNFDeployRequest is an alias for VNFDeployRequest.

type O2VNFDeployRequest = VNFDeployRequest

// VNFInstance is an alias for O2VNFInstance for test compatibility.

type VNFInstance = O2VNFInstance

// VNFDescriptor describes a VNF for deployment.

type VNFDescriptor struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Version string `json:"version"`

	Vendor string `json:"vendor"`

	Description string `json:"description,omitempty"`

	Image string `json:"image"`

	Replicas int32 `json:"replicas"`

	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	Environment []corev1.EnvVar `json:"environment,omitempty"`

	Ports []corev1.ContainerPort `json:"ports,omitempty"`

	VolumeConfig []VolumeConfig `json:"volumeConfig,omitempty"`

	NetworkConfig *NetworkConfig `json:"networkConfig,omitempty"`

	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	HealthCheck *TestHealthCheckConfig `json:"healthCheck,omitempty"`

	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// VNFScaleRequest represents a request to scale a VNF.

type VNFScaleRequest struct {
	ScaleType string `json:"scaleType"` // SCALE_OUT, SCALE_IN

	NumberOfSteps int32 `json:"numberOfSteps"`

	AspectID string `json:"aspectId,omitempty"`
}

// O2VNFInstance represents a deployed VNF instance.

type O2VNFInstance struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Namespace string `json:"namespace"`

	VNFPackageID string `json:"vnfPackageId"`

	FlavorID string `json:"flavorId"`

	Status *VNFInstanceStatus `json:"status"`

	Resources *ResourceInfo `json:"resources,omitempty"`

	NetworkEndpoints []NetworkEndpoint `json:"networkEndpoints,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// VNFInstanceStatus represents the status of a VNF instance.

type VNFInstanceStatus struct {
	State string `json:"state"` // INSTANTIATED, NOT_INSTANTIATED, TERMINATED

	DetailedState string `json:"detailedState"` // RUNNING, PENDING, ERROR
}

// ResourceInfo represents resource information for a VNF instance.

type ResourceInfo struct {
	CPU string `json:"cpu"`

	Memory string `json:"memory"`
}

// NetworkEndpoint represents a network endpoint for a VNF instance.

type NetworkEndpoint struct {
	Name string `json:"name"`

	Address string `json:"address"`

	Port int32 `json:"port"`

	Protocol string `json:"protocol"`
}

// VolumeConfig represents volume configuration for VNFs.

type VolumeConfig struct {
	Name string `json:"name"`

	MountPath string `json:"mountPath"`

	VolumeSource corev1.VolumeSource `json:"volumeSource"`
}

// NetworkConfig represents network configuration for VNFs.

type NetworkConfig struct {
	ServiceType corev1.ServiceType `json:"serviceType"`

	Ports []corev1.ServicePort `json:"ports"`
}

// TestHealthCheckConfig represents health check configuration for VNFs (test-specific).

type TestHealthCheckConfig struct {
	LivenessProbe *corev1.Probe `json:"livenessProbe,omitempty"`

	ReadinessProbe *corev1.Probe `json:"readinessProbe,omitempty"`

	StartupProbe *corev1.Probe `json:"startupProbe,omitempty"`
}

// InfrastructureInfo represents infrastructure information.

type InfrastructureInfo struct {
	NodeCount int `json:"nodeCount"`

	ClusterName string `json:"clusterName"`

	KubernetesVersion string `json:"kubernetesVersion"`

	TotalResources *ResourceSummary `json:"totalResources,omitempty"`
}

// ResourceSummary represents a summary of resources.

type ResourceSummary struct {
	CPU string `json:"cpu"`

	Memory string `json:"memory"`
}

// ResourceMap represents discovered resources.

type ResourceMap struct {
	Nodes map[string]*NodeInfo `json:"nodes"`

	Namespaces map[string]*NamespaceInfo `json:"namespaces"`

	Metrics *ClusterMetrics `json:"metrics"`
}

// NodeInfo represents information about a node.

type NodeInfo struct {
	Name string `json:"name"`

	Labels map[string]string `json:"labels"`

	Roles []string `json:"roles"`
}

// NamespaceInfo represents information about a namespace.

type NamespaceInfo struct {
	Name string `json:"name"`

	Status string `json:"status"`
	
	PodCount int32 `json:"pod_count"`
}

// O2Manager provides high-level management operations for O2 interface.
type O2Manager struct {
	adaptor *O2Adaptor
}

// NewO2Manager creates a new O2Manager instance.
func NewO2Manager(adaptor *O2Adaptor) *O2Manager {
	return &O2Manager{
		adaptor: adaptor,
	}
}

// DiscoverResources discovers resources in the cluster.
func (m *O2Manager) DiscoverResources(ctx context.Context) (*ResourceMap, error) {
	resourceMap, err := m.adaptor.DiscoverResources(ctx)
	if err != nil {
		return nil, err
	}

	// Set TotalNodes from the discovered nodes.
	resourceMap.Metrics.TotalNodes = int32(len(resourceMap.Nodes))

	// List pods and count per namespace.
	podList := &corev1.PodList{}
	if listErr := m.adaptor.kubeClient.List(ctx, podList); listErr == nil {
		for i := range podList.Items {
			pod := &podList.Items[i]
			ns := pod.Namespace
			if nsInfo, ok := resourceMap.Namespaces[ns]; ok {
				nsInfo.PodCount++
			}
			resourceMap.Metrics.TotalPods++
		}
	}

	return resourceMap, nil
}

// ScaleWorkload scales a workload to the specified number of replicas.
// workloadID format: "namespace/name"
func (m *O2Manager) ScaleWorkload(ctx context.Context, workloadID string, replicas int32) error {
	parts := strings.SplitN(workloadID, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid workload ID format, expected namespace/name: %s", workloadID)
	}
	namespace, name := parts[0], parts[1]

	deployment := &appsv1.Deployment{}
	if err := m.adaptor.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment); err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	deployment.Spec.Replicas = &replicas
	if err := m.adaptor.kubeClient.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to scale workload: %w", err)
	}

	return nil
}

// DeployVNF deploys a VNF with the given specification.
// It delegates to the adaptor's full implementation which uses the configured
// namespace, applies all labels from Metadata, and wires up volumes, health
// checks, affinity, tolerations, and creates the associated Service.
func (m *O2Manager) DeployVNF(ctx context.Context, vnfSpec *VNFDescriptor) (*DeploymentStatus, error) {
	status, err := m.adaptor.deployVNFFromDescriptor(ctx, vnfSpec)
	if err != nil {
		return nil, err
	}

	namespace := m.adaptor.config.Namespace
	return &DeploymentStatus{
		ID:              fmt.Sprintf("%s/%s", namespace, vnfSpec.Name),
		Name:            status.Name,
		Status:          status.Status,
		State:           status.Status,
		Phase:           status.Phase,
		Replicas:        status.Replicas,
		Health:          "UNKNOWN",
		LastStateChange: time.Now(),
	}, nil
}

// ClusterMetrics represents cluster metrics.

type ClusterMetrics struct {
	TotalNodes int32 `json:"totalNodes"`

	ReadyNodes int32 `json:"readyNodes"`

	TotalPods int32 `json:"totalPods"`

	TotalCPU string `json:"totalCPU"`

	TotalMemory string `json:"totalMemory"`
}

// VNFDeploymentStatus represents the status of a VNF deployment.

type VNFDeploymentStatus struct {
	Name string `json:"name"`

	Status string `json:"status"`

	Phase string `json:"phase"`

	Replicas int32 `json:"replicas"`
}
