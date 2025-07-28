package o2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
)

// O2AdaptorInterface defines the O2 interface for cloud infrastructure and VNF lifecycle management
type O2AdaptorInterface interface {
	// VNF Lifecycle Management
	DeployVNF(ctx context.Context, req *VNFDeployRequest) (*VNFInstance, error)
	ScaleVNF(ctx context.Context, instanceID string, req *VNFScaleRequest) error
	UpdateVNF(ctx context.Context, instanceID string, req *VNFUpdateRequest) error
	TerminateVNF(ctx context.Context, instanceID string) error
	GetVNFInstance(ctx context.Context, instanceID string) (*VNFInstance, error)
	ListVNFInstances(ctx context.Context, filter *VNFFilter) ([]*VNFInstance, error)
	
	// Infrastructure Management
	GetInfrastructureInfo(ctx context.Context) (*InfrastructureInfo, error)
	GetNodeResources(ctx context.Context, nodeID string) (*NodeResources, error)
	ListAvailableResources(ctx context.Context) (*ResourceAvailability, error)
	
	// Container and Pod Management
	CreatePod(ctx context.Context, req *PodCreateRequest) (*corev1.Pod, error)
	UpdatePod(ctx context.Context, podName, namespace string, req *PodUpdateRequest) error
	DeletePod(ctx context.Context, podName, namespace string) error
	GetPodStatus(ctx context.Context, podName, namespace string) (*PodStatus, error)
	
	// Service and Network Management
	CreateService(ctx context.Context, req *ServiceCreateRequest) (*corev1.Service, error)
	UpdateService(ctx context.Context, serviceName, namespace string, req *ServiceUpdateRequest) error
	DeleteService(ctx context.Context, serviceName, namespace string) error
	
	// Monitoring and Metrics
	GetVNFMetrics(ctx context.Context, instanceID string) (*VNFMetrics, error)
	GetInfrastructureMetrics(ctx context.Context) (*InfrastructureMetrics, error)
	
	// Fault and Healing
	GetVNFFaults(ctx context.Context, instanceID string) ([]*VNFFault, error)
	TriggerVNFHealing(ctx context.Context, instanceID string, req *HealingRequest) error
}

// O2Adaptor implements the O2 interface for Kubernetes-based cloud infrastructure
type O2Adaptor struct {
	kubeClient    client.Client
	clientset     kubernetes.Interface
	config        *O2Config
}

// O2Config holds O2 interface configuration
type O2Config struct {
	Namespace         string
	DefaultResources  *corev1.ResourceRequirements
	ImagePullSecrets  []string
	NodeSelector      map[string]string
	Tolerations       []corev1.Toleration
	ServiceAccount    string
	TLSConfig         *oran.TLSConfig
}

// VNF Lifecycle types
type VNFDeployRequest struct {
	Name             string                       `json:"name"`
	Namespace        string                       `json:"namespace"`
	VNFPackageID     string                       `json:"vnf_package_id"`
	FlavorID         string                       `json:"flavor_id"`
	Image            string                       `json:"image"`
	Replicas         int32                        `json:"replicas"`
	Resources        *corev1.ResourceRequirements `json:"resources"`
	Environment      []corev1.EnvVar              `json:"environment"`
	Ports            []corev1.ContainerPort       `json:"ports"`
	VolumeConfig     []VolumeConfig               `json:"volume_config"`
	NetworkConfig    *NetworkConfig               `json:"network_config"`
	SecurityContext  *corev1.SecurityContext      `json:"security_context"`
	Metadata         map[string]string            `json:"metadata"`
}

type VNFScaleRequest struct {
	ScaleType       string `json:"scale_type"` // SCALE_OUT, SCALE_IN, SCALE_UP, SCALE_DOWN
	NumberOfSteps   int32  `json:"number_of_steps"`
	AspectID        string `json:"aspect_id"`
	AdditionalParams map[string]interface{} `json:"additional_params"`
}

type VNFUpdateRequest struct {
	VNFPackageID     string                 `json:"vnf_package_id"`
	Image            string                 `json:"image"`
	Environment      []corev1.EnvVar        `json:"environment"`
	AdditionalParams map[string]interface{} `json:"additional_params"`
}

type VNFInstance struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	VNFPackageID      string            `json:"vnf_package_id"`
	FlavorID          string            `json:"flavor_id"`
	Status            VNFInstanceStatus `json:"status"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	Metadata          map[string]string `json:"metadata"`
	Resources         *VNFResources     `json:"resources"`
	NetworkEndpoints  []NetworkEndpoint `json:"network_endpoints"`
}

type VNFInstanceStatus struct {
	State           string `json:"state"` // INSTANTIATED, NOT_INSTANTIATED, FAILED
	DetailedState   string `json:"detailed_state"`
	LastStateChange time.Time `json:"last_state_change"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

type VNFResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Storage string `json:"storage"`
}

type VNFFilter struct {
	Names      []string `json:"names,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	States     []string `json:"states,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// Infrastructure types
type InfrastructureInfo struct {
	ClusterName     string         `json:"cluster_name"`
	KubernetesVersion string       `json:"kubernetes_version"`
	NodeCount       int            `json:"node_count"`
	TotalResources  *NodeResources `json:"total_resources"`
	AvailableResources *NodeResources `json:"available_resources"`
	StorageClasses  []string       `json:"storage_classes"`
	NetworkPolicies []string       `json:"network_policies"`
}

type NodeResources struct {
	CPU     string            `json:"cpu"`
	Memory  string            `json:"memory"`
	Storage string            `json:"storage"`
	Pods    string            `json:"pods"`
	Custom  map[string]string `json:"custom,omitempty"`
}

type ResourceAvailability struct {
	Nodes map[string]*NodeResources `json:"nodes"`
	Total *NodeResources             `json:"total"`
}

// Container and Pod types
type PodCreateRequest struct {
	Name            string                  `json:"name"`
	Namespace       string                  `json:"namespace"`
	Image           string                  `json:"image"`
	Command         []string                `json:"command,omitempty"`
	Args            []string                `json:"args,omitempty"`
	Environment     []corev1.EnvVar         `json:"environment,omitempty"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
	Ports           []corev1.ContainerPort  `json:"ports,omitempty"`
	VolumeConfig    []VolumeConfig          `json:"volume_config,omitempty"`
	SecurityContext *corev1.SecurityContext `json:"security_context,omitempty"`
	Labels          map[string]string       `json:"labels,omitempty"`
	Annotations     map[string]string       `json:"annotations,omitempty"`
}

type PodUpdateRequest struct {
	Image       string          `json:"image,omitempty"`
	Environment []corev1.EnvVar `json:"environment,omitempty"`
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type PodStatus struct {
	Phase      string                `json:"phase"`
	Conditions []corev1.PodCondition `json:"conditions"`
	StartTime  *metav1.Time          `json:"start_time"`
	Ready      bool                  `json:"ready"`
	RestartCount int32               `json:"restart_count"`
	Message    string                `json:"message,omitempty"`
}

type VolumeConfig struct {
	Name      string           `json:"name"`
	MountPath string           `json:"mount_path"`
	VolumeSource corev1.VolumeSource `json:"volume_source"`
}

type NetworkConfig struct {
	ServiceType    corev1.ServiceType `json:"service_type"`
	Ports          []corev1.ServicePort `json:"ports"`
	NetworkPolicies []string           `json:"network_policies,omitempty"`
}

type NetworkEndpoint struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int32  `json:"port"`
}

// Service types
type ServiceCreateRequest struct {
	Name        string                `json:"name"`
	Namespace   string                `json:"namespace"`
	Selector    map[string]string     `json:"selector"`
	Ports       []corev1.ServicePort  `json:"ports"`
	Type        corev1.ServiceType    `json:"type"`
	Labels      map[string]string     `json:"labels,omitempty"`
	Annotations map[string]string     `json:"annotations,omitempty"`
}

type ServiceUpdateRequest struct {
	Ports       []corev1.ServicePort  `json:"ports,omitempty"`
	Type        corev1.ServiceType    `json:"type,omitempty"`
	Labels      map[string]string     `json:"labels,omitempty"`
	Annotations map[string]string     `json:"annotations,omitempty"`
}

// Monitoring types
type VNFMetrics struct {
	InstanceID   string                 `json:"instance_id"`
	Timestamp    time.Time              `json:"timestamp"`
	CPUUsage     float64                `json:"cpu_usage"`
	MemoryUsage  float64                `json:"memory_usage"`
	NetworkIO    NetworkIOMetrics       `json:"network_io"`
	StorageIO    StorageIOMetrics       `json:"storage_io"`
	CustomMetrics map[string]interface{} `json:"custom_metrics"`
}

type InfrastructureMetrics struct {
	Timestamp      time.Time                    `json:"timestamp"`
	ClusterMetrics *ClusterMetrics              `json:"cluster_metrics"`
	NodeMetrics    map[string]*NodeMetrics      `json:"node_metrics"`
}

type ClusterMetrics struct {
	TotalPods       int32   `json:"total_pods"`
	RunningPods     int32   `json:"running_pods"`
	FailedPods      int32   `json:"failed_pods"`
	CPUUtilization  float64 `json:"cpu_utilization"`
	MemoryUtilization float64 `json:"memory_utilization"`
}

type NodeMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	PodCount    int32   `json:"pod_count"`
}

type NetworkIOMetrics struct {
	BytesReceived    int64 `json:"bytes_received"`
	BytesTransmitted int64 `json:"bytes_transmitted"`
	PacketsReceived  int64 `json:"packets_received"`
	PacketsTransmitted int64 `json:"packets_transmitted"`
}

type StorageIOMetrics struct {
	BytesRead    int64 `json:"bytes_read"`
	BytesWritten int64 `json:"bytes_written"`
	ReadOps      int64 `json:"read_ops"`
	WriteOps     int64 `json:"write_ops"`
}

// Fault and Healing types
type VNFFault struct {
	ID           string    `json:"id"`
	InstanceID   string    `json:"instance_id"`
	FaultType    string    `json:"fault_type"`
	Severity     string    `json:"severity"`
	Description  string    `json:"description"`
	DetectedAt   time.Time `json:"detected_at"`
	Status       string    `json:"status"` // ACTIVE, RESOLVED, HEALING
	Recommendations []string `json:"recommendations"`
}

type HealingRequest struct {
	FaultID     string                 `json:"fault_id"`
	HealingType string                 `json:"healing_type"` // RESTART, RECREATE, SCALE
	Parameters  map[string]interface{} `json:"parameters"`
}

// NewO2Adaptor creates a new O2 adaptor
func NewO2Adaptor(kubeClient client.Client, clientset kubernetes.Interface, config *O2Config) *O2Adaptor {
	if config == nil {
		config = &O2Config{
			Namespace: "default",
			DefaultResources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    "100m",
					corev1.ResourceMemory: "128Mi",
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    "500m",
					corev1.ResourceMemory: "512Mi",
				},
			},
		}
	}
	
	return &O2Adaptor{
		kubeClient: kubeClient,
		clientset:  clientset,
		config:     config,
	}
}

// DeployVNF deploys a VNF instance using Kubernetes deployment
func (a *O2Adaptor) DeployVNF(ctx context.Context, req *VNFDeployRequest) (*VNFInstance, error) {
	logger := log.FromContext(ctx)
	logger.Info("deploying VNF", "name", req.Name, "namespace", req.Namespace)
	
	// Create namespace if it doesn't exist
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Namespace,
		},
	}
	if err := a.kubeClient.Create(ctx, namespace); err != nil && !strings.Contains(err.Error(), "already exists") {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}
	
	// Create deployment
	deployment := a.buildDeployment(req)
	if err := a.kubeClient.Create(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}
	
	// Create service if network config is provided
	var service *corev1.Service
	if req.NetworkConfig != nil {
		serviceReq := &ServiceCreateRequest{
			Name:      req.Name + "-service",
			Namespace: req.Namespace,
			Selector:  map[string]string{"app": req.Name},
			Ports:     req.NetworkConfig.Ports,
			Type:      req.NetworkConfig.ServiceType,
		}
		
		var err error
		service, err = a.CreateService(ctx, serviceReq)
		if err != nil {
			logger.Error(err, "failed to create service, continuing without service")
		}
	}
	
	// Build VNF instance response
	instance := &VNFInstance{
		ID:           fmt.Sprintf("%s-%s", req.Namespace, req.Name),
		Name:         req.Name,
		Namespace:    req.Namespace,
		VNFPackageID: req.VNFPackageID,
		FlavorID:     req.FlavorID,
		Status: VNFInstanceStatus{
			State:           "INSTANTIATED",
			DetailedState:   "CREATING",
			LastStateChange: time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  req.Metadata,
		Resources: &VNFResources{
			CPU:     req.Resources.Requests.Cpu().String(),
			Memory:  req.Resources.Requests.Memory().String(),
			Storage: "10Gi", // Default storage
		},
	}
	
	// Add network endpoints if service was created
	if service != nil {
		for _, port := range service.Spec.Ports {
			endpoint := NetworkEndpoint{
				Name:     port.Name,
				Protocol: string(port.Protocol),
				Address:  service.Spec.ClusterIP,
				Port:     port.Port,
			}
			instance.NetworkEndpoints = append(instance.NetworkEndpoints, endpoint)
		}
	}
	
	logger.Info("VNF deployed successfully", "instanceID", instance.ID)
	return instance, nil
}

// ScaleVNF scales a VNF instance
func (a *O2Adaptor) ScaleVNF(ctx context.Context, instanceID string, req *VNFScaleRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("scaling VNF", "instanceID", instanceID, "scaleType", req.ScaleType)
	
	// Parse instance ID to get namespace and name
	parts := strings.SplitN(instanceID, "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid instance ID format: %s", instanceID)
	}
	namespace, name := parts[0], parts[1]
	
	// Get current deployment
	deployment := &appsv1.Deployment{}
	if err := a.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment); err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}
	
	// Calculate new replica count based on scale type
	currentReplicas := *deployment.Spec.Replicas
	var newReplicas int32
	
	switch req.ScaleType {
	case "SCALE_OUT":
		newReplicas = currentReplicas + req.NumberOfSteps
	case "SCALE_IN":
		newReplicas = currentReplicas - req.NumberOfSteps
		if newReplicas < 0 {
			newReplicas = 0
		}
	case "SCALE_UP", "SCALE_DOWN":
		// For vertical scaling, we would modify resource requests/limits
		return fmt.Errorf("vertical scaling not implemented yet")
	default:
		return fmt.Errorf("unsupported scale type: %s", req.ScaleType)
	}
	
	// Update deployment
	deployment.Spec.Replicas = &newReplicas
	if err := a.kubeClient.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	
	logger.Info("VNF scaled successfully", "instanceID", instanceID, "oldReplicas", currentReplicas, "newReplicas", newReplicas)
	return nil
}

// GetVNFInstance retrieves VNF instance information
func (a *O2Adaptor) GetVNFInstance(ctx context.Context, instanceID string) (*VNFInstance, error) {
	// Parse instance ID
	parts := strings.SplitN(instanceID, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid instance ID format: %s", instanceID)
	}
	namespace, name := parts[0], parts[1]
	
	// Get deployment
	deployment := &appsv1.Deployment{}
	if err := a.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment); err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	
	// Build instance from deployment
	instance := &VNFInstance{
		ID:        instanceID,
		Name:      name,
		Namespace: namespace,
		Status: VNFInstanceStatus{
			State:           "INSTANTIATED",
			DetailedState:   a.getDeploymentState(deployment),
			LastStateChange: deployment.CreationTimestamp.Time,
		},
		CreatedAt: deployment.CreationTimestamp.Time,
		UpdatedAt: time.Now(),
		Metadata:  deployment.Labels,
	}
	
	// Get associated service
	serviceList := &corev1.ServiceList{}
	listOpts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{"app": name}),
	}
	
	if err := a.kubeClient.List(ctx, serviceList, listOpts); err == nil {
		for _, service := range serviceList.Items {
			for _, port := range service.Spec.Ports {
				endpoint := NetworkEndpoint{
					Name:     port.Name,
					Protocol: string(port.Protocol),
					Address:  service.Spec.ClusterIP,
					Port:     port.Port,
				}
				instance.NetworkEndpoints = append(instance.NetworkEndpoints, endpoint)
			}
		}
	}
	
	return instance, nil
}

// GetInfrastructureInfo retrieves cluster infrastructure information
func (a *O2Adaptor) GetInfrastructureInfo(ctx context.Context) (*InfrastructureInfo, error) {
	logger := log.FromContext(ctx)
	
	// Get cluster version
	version, err := a.clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}
	
	// Get nodes
	nodes, err := a.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	
	// Calculate total resources
	totalResources := &NodeResources{
		CPU:     "0",
		Memory:  "0",
		Storage: "0",
		Pods:    "0",
	}
	
	for _, node := range nodes.Items {
		// Add node resources to total (simplified calculation)
		if cpu := node.Status.Capacity[corev1.ResourceCPU]; !cpu.IsZero() {
			// In a real implementation, we would properly add these values
		}
	}
	
	// Get storage classes
	storageClasses, err := a.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	var scNames []string
	if err == nil {
		for _, sc := range storageClasses.Items {
			scNames = append(scNames, sc.Name)
		}
	}
	
	info := &InfrastructureInfo{
		ClusterName:       "kubernetes-cluster", // Could be read from cluster-info
		KubernetesVersion: version.GitVersion,
		NodeCount:         len(nodes.Items),
		TotalResources:    totalResources,
		AvailableResources: totalResources, // Simplified
		StorageClasses:    scNames,
	}
	
	logger.Info("retrieved infrastructure info", "nodeCount", info.NodeCount, "k8sVersion", info.KubernetesVersion)
	return info, nil
}

// CreatePod creates a pod
func (a *O2Adaptor) CreatePod(ctx context.Context, req *PodCreateRequest) (*corev1.Pod, error) {
	logger := log.FromContext(ctx)
	
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            req.Name,
					Image:           req.Image,
					Command:         req.Command,
					Args:            req.Args,
					Env:             req.Environment,
					Resources:       *req.Resources,
					Ports:           req.Ports,
					SecurityContext: req.SecurityContext,
				},
			},
		},
	}
	
	// Add volumes if configured
	for _, volConfig := range req.VolumeConfig {
		volume := corev1.Volume{
			Name:         volConfig.Name,
			VolumeSource: volConfig.VolumeSource,
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		
		volumeMount := corev1.VolumeMount{
			Name:      volConfig.Name,
			MountPath: volConfig.MountPath,
		}
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, volumeMount)
	}
	
	if err := a.kubeClient.Create(ctx, pod); err != nil {
		return nil, fmt.Errorf("failed to create pod: %w", err)
	}
	
	logger.Info("pod created successfully", "name", req.Name, "namespace", req.Namespace)
	return pod, nil
}

// CreateService creates a service
func (a *O2Adaptor) CreateService(ctx context.Context, req *ServiceCreateRequest) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: req.Selector,
			Ports:    req.Ports,
			Type:     req.Type,
		},
	}
	
	if err := a.kubeClient.Create(ctx, service); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}
	
	return service, nil
}

// GetVNFMetrics retrieves VNF performance metrics
func (a *O2Adaptor) GetVNFMetrics(ctx context.Context, instanceID string) (*VNFMetrics, error) {
	// In a real implementation, we would integrate with Prometheus or other metrics systems
	metrics := &VNFMetrics{
		InstanceID:  instanceID,
		Timestamp:   time.Now(),
		CPUUsage:    45.2,
		MemoryUsage: 62.8,
		NetworkIO: NetworkIOMetrics{
			BytesReceived:    1024 * 1024 * 100, // 100MB
			BytesTransmitted: 1024 * 1024 * 80,  // 80MB
			PacketsReceived:  50000,
			PacketsTransmitted: 40000,
		},
		StorageIO: StorageIOMetrics{
			BytesRead:    1024 * 1024 * 200, // 200MB
			BytesWritten: 1024 * 1024 * 150, // 150MB
			ReadOps:      10000,
			WriteOps:     8000,
		},
		CustomMetrics: map[string]interface{}{
			"active_connections": 120,
			"request_rate":       150.5,
			"error_rate":         0.02,
		},
	}
	
	return metrics, nil
}

// Helper functions

func (a *O2Adaptor) buildDeployment(req *VNFDeployRequest) *appsv1.Deployment {
	labels := map[string]string{
		"app":                req.Name,
		"nephoran.com/vnf":   "true",
		"nephoran.com/type":  "o-ran",
	}
	
	// Add custom labels
	for k, v := range req.Metadata {
		labels[k] = v
	}
	
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &req.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": req.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            req.Name,
							Image:           req.Image,
							Env:             req.Environment,
							Resources:       *req.Resources,
							Ports:           req.Ports,
							SecurityContext: req.SecurityContext,
						},
					},
				},
			},
		},
	}
	
	// Add volumes
	for _, volConfig := range req.VolumeConfig {
		volume := corev1.Volume{
			Name:         volConfig.Name,
			VolumeSource: volConfig.VolumeSource,
		}
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)
		
		volumeMount := corev1.VolumeMount{
			Name:      volConfig.Name,
			MountPath: volConfig.MountPath,
		}
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts, volumeMount)
	}
	
	return deployment
}

func (a *O2Adaptor) getDeploymentState(deployment *appsv1.Deployment) string {
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return "RUNNING"
	} else if deployment.Status.ReadyReplicas > 0 {
		return "PARTIALLY_RUNNING"
	} else {
		return "STARTING"
	}
}

// Additional interface methods (simplified implementations)

func (a *O2Adaptor) UpdateVNF(ctx context.Context, instanceID string, req *VNFUpdateRequest) error {
	// Implementation would update deployment with new image/config
	return nil
}

func (a *O2Adaptor) TerminateVNF(ctx context.Context, instanceID string) error {
	parts := strings.SplitN(instanceID, "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid instance ID format: %s", instanceID)
	}
	namespace, name := parts[0], parts[1]
	
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	
	return a.kubeClient.Delete(ctx, deployment)
}

func (a *O2Adaptor) ListVNFInstances(ctx context.Context, filter *VNFFilter) ([]*VNFInstance, error) {
	// Implementation would list deployments and convert to VNF instances
	return []*VNFInstance{}, nil
}

func (a *O2Adaptor) GetNodeResources(ctx context.Context, nodeID string) (*NodeResources, error) {
	// Implementation would get specific node resource info
	return &NodeResources{}, nil
}

func (a *O2Adaptor) ListAvailableResources(ctx context.Context) (*ResourceAvailability, error) {
	// Implementation would calculate available resources across cluster
	return &ResourceAvailability{}, nil
}

func (a *O2Adaptor) UpdatePod(ctx context.Context, podName, namespace string, req *PodUpdateRequest) error {
	// Implementation would update pod configuration
	return nil
}

func (a *O2Adaptor) DeletePod(ctx context.Context, podName, namespace string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
	}
	return a.kubeClient.Delete(ctx, pod)
}

func (a *O2Adaptor) GetPodStatus(ctx context.Context, podName, namespace string) (*PodStatus, error) {
	// Implementation would get pod status
	return &PodStatus{}, nil
}

func (a *O2Adaptor) UpdateService(ctx context.Context, serviceName, namespace string, req *ServiceUpdateRequest) error {
	// Implementation would update service configuration
	return nil
}

func (a *O2Adaptor) DeleteService(ctx context.Context, serviceName, namespace string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
	}
	return a.kubeClient.Delete(ctx, service)
}

func (a *O2Adaptor) GetInfrastructureMetrics(ctx context.Context) (*InfrastructureMetrics, error) {
	// Implementation would gather cluster-wide metrics
	return &InfrastructureMetrics{}, nil
}

func (a *O2Adaptor) GetVNFFaults(ctx context.Context, instanceID string) ([]*VNFFault, error) {
	// Implementation would check for pod/deployment issues
	return []*VNFFault{}, nil
}

func (a *O2Adaptor) TriggerVNFHealing(ctx context.Context, instanceID string, req *HealingRequest) error {
	// Implementation would perform healing actions (restart, recreate, etc.)
	return nil
}