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

// O2Manager implements comprehensive cloud management capabilities
type O2Manager struct {
	adaptor *O2Adaptor
}

// NewO2Manager creates a new O2Manager with cloud management capabilities
func NewO2Manager(adaptor *O2Adaptor) *O2Manager {
	return &O2Manager{
		adaptor: adaptor,
	}
}

// ResourceMap represents discovered cloud resources
type ResourceMap struct {
	Nodes       map[string]*NodeInfo      `json:"nodes"`
	Namespaces  map[string]*NamespaceInfo `json:"namespaces"`
	StorageClasses []string               `json:"storage_classes"`
	Networks    map[string]*NetworkInfo   `json:"networks"`
	Metrics     *ClusterResourceMetrics   `json:"metrics"`
	Timestamp   time.Time                 `json:"timestamp"`
}

type NodeInfo struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Roles        []string          `json:"roles"`
	Capacity     *NodeResources    `json:"capacity"`
	Allocatable  *NodeResources    `json:"allocatable"`
	Used         *NodeResources    `json:"used"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Conditions   []string          `json:"conditions"`
}

type NamespaceInfo struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	ResourceQuota *ResourceQuota   `json:"resource_quota,omitempty"`
	PodCount     int32             `json:"pod_count"`
}

type NetworkInfo struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"` // CNI, overlay, underlay
	CIDR         string            `json:"cidr"`
	Gateway      string            `json:"gateway"`
	Endpoints    int32             `json:"endpoints"`
}

type ResourceQuota struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Pods   int32  `json:"pods"`
	PVCs   int32  `json:"pvcs"`
}

type ClusterResourceMetrics struct {
	TotalNodes      int32   `json:"total_nodes"`
	ReadyNodes      int32   `json:"ready_nodes"`
	TotalPods       int32   `json:"total_pods"`
	RunningPods     int32   `json:"running_pods"`
	TotalCPU        string  `json:"total_cpu"`
	TotalMemory     string  `json:"total_memory"`
	UsedCPU         string  `json:"used_cpu"`
	UsedMemory      string  `json:"used_memory"`
	CPUUtilization  float64 `json:"cpu_utilization"`
	MemoryUtilization float64 `json:"memory_utilization"`
}

// VNFDescriptor represents a VNF deployment specification
type VNFDescriptor struct {
	Name             string                       `json:"name"`
	Type             string                       `json:"type"` // amf, smf, upf, etc.
	Version          string                       `json:"version"`
	Vendor           string                       `json:"vendor"`
	Description      string                       `json:"description"`
	Image            string                       `json:"image"`
	Replicas         int32                        `json:"replicas"`
	Resources        *corev1.ResourceRequirements `json:"resources"`
	Environment      []corev1.EnvVar              `json:"environment"`
	Ports            []corev1.ContainerPort       `json:"ports"`
	VolumeConfig     []VolumeConfig               `json:"volume_config"`
	NetworkConfig    *NetworkConfig               `json:"network_config"`
	SecurityContext  *corev1.SecurityContext      `json:"security_context"`
	HealthCheck      *HealthCheckConfig           `json:"health_check"`
	Affinity         *corev1.Affinity             `json:"affinity"`
	Tolerations      []corev1.Toleration          `json:"tolerations"`
	Metadata         map[string]string            `json:"metadata"`
}

type HealthCheckConfig struct {
	LivenessProbe  *corev1.Probe `json:"liveness_probe"`
	ReadinessProbe *corev1.Probe `json:"readiness_probe"`
	StartupProbe   *corev1.Probe `json:"startup_probe"`
}

type DeploymentStatus struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Status           string            `json:"status"` // PENDING, RUNNING, FAILED, SUCCEEDED
	Phase            string            `json:"phase"`  // Creating, Ready, Updating, Terminating
	Replicas         int32             `json:"replicas"`
	ReadyReplicas    int32             `json:"ready_replicas"`
	UpdatedReplicas  int32             `json:"updated_replicas"`
	AvailableReplicas int32            `json:"available_replicas"`
	Conditions       []string          `json:"conditions"`
	LastUpdated      time.Time         `json:"last_updated"`
	Events           []string          `json:"events"`
	Resources        *VNFResources     `json:"resources"`
	Endpoints        []NetworkEndpoint `json:"endpoints"`
}

// DiscoverResources discovers and maps all available cloud resources
func (m *O2Manager) DiscoverResources(ctx context.Context) (*ResourceMap, error) {
	logger := log.FromContext(ctx)
	logger.Info("starting resource discovery")

	resourceMap := &ResourceMap{
		Nodes:      make(map[string]*NodeInfo),
		Namespaces: make(map[string]*NamespaceInfo),
		Networks:   make(map[string]*NetworkInfo),
		Timestamp:  time.Now(),
	}

	// Discover nodes
	nodes, err := m.adaptor.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	totalCPU := int64(0)
	totalMemory := int64(0)
	usedCPU := int64(0)
	usedMemory := int64(0)
	readyNodes := int32(0)

	for _, node := range nodes.Items {
		nodeInfo := &NodeInfo{
			Name:        node.Name,
			Status:      string(node.Status.Phase),
			Labels:      node.Labels,
			Annotations: node.Annotations,
		}

		// Extract node roles
		for label := range node.Labels {
			if strings.Contains(label, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
				nodeInfo.Roles = append(nodeInfo.Roles, role)
			}
		}

		// Calculate resources
		if cpu := node.Status.Capacity[corev1.ResourceCPU]; !cpu.IsZero() {
			totalCPU += cpu.MilliValue()
		}
		if memory := node.Status.Capacity[corev1.ResourceMemory]; !memory.IsZero() {
			totalMemory += memory.Value()
		}

		nodeInfo.Capacity = &NodeResources{
			CPU:    node.Status.Capacity.Cpu().String(),
			Memory: node.Status.Capacity.Memory().String(),
			Pods:   node.Status.Capacity.Pods().String(),
		}

		nodeInfo.Allocatable = &NodeResources{
			CPU:    node.Status.Allocatable.Cpu().String(),
			Memory: node.Status.Allocatable.Memory().String(),
			Pods:   node.Status.Allocatable.Pods().String(),
		}

		// Check node conditions
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				readyNodes++
			}
			nodeInfo.Conditions = append(nodeInfo.Conditions, 
				fmt.Sprintf("%s=%s", condition.Type, condition.Status))
		}

		resourceMap.Nodes[node.Name] = nodeInfo
	}

	// Discover namespaces
	namespaces, err := m.adaptor.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to list namespaces")
	} else {
		for _, ns := range namespaces.Items {
			nsInfo := &NamespaceInfo{
				Name:   ns.Name,
				Status: string(ns.Status.Phase),
				Labels: ns.Labels,
			}

			// Count pods in namespace
			pods, err := m.adaptor.clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
			if err == nil {
				nsInfo.PodCount = int32(len(pods.Items))
				
				// Calculate used resources in namespace
				for _, pod := range pods.Items {
					for _, container := range pod.Spec.Containers {
						if cpu := container.Resources.Requests.Cpu(); !cpu.IsZero() {
							usedCPU += cpu.MilliValue()
						}
						if memory := container.Resources.Requests.Memory(); !memory.IsZero() {
							usedMemory += memory.Value()
						}
					}
				}
			}

			resourceMap.Namespaces[ns.Name] = nsInfo
		}
	}

	// Discover storage classes
	storageClasses, err := m.adaptor.clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to list storage classes")
	} else {
		for _, sc := range storageClasses.Items {
			resourceMap.StorageClasses = append(resourceMap.StorageClasses, sc.Name)
		}
	}

	// Calculate cluster metrics
	totalPods := int32(0)
	runningPods := int32(0)
	for _, nsInfo := range resourceMap.Namespaces {
		totalPods += nsInfo.PodCount
		runningPods += nsInfo.PodCount // Simplified - would check actual pod status
	}

	resourceMap.Metrics = &ClusterResourceMetrics{
		TotalNodes:        int32(len(nodes.Items)),
		ReadyNodes:        readyNodes,
		TotalPods:         totalPods,
		RunningPods:       runningPods,
		TotalCPU:          fmt.Sprintf("%dm", totalCPU),
		TotalMemory:       fmt.Sprintf("%d", totalMemory),
		UsedCPU:           fmt.Sprintf("%dm", usedCPU),
		UsedMemory:        fmt.Sprintf("%d", usedMemory),
		CPUUtilization:    float64(usedCPU) / float64(totalCPU) * 100,
		MemoryUtilization: float64(usedMemory) / float64(totalMemory) * 100,
	}

	logger.Info("resource discovery completed", 
		"nodes", len(resourceMap.Nodes),
		"namespaces", len(resourceMap.Namespaces),
		"storageClasses", len(resourceMap.StorageClasses))

	return resourceMap, nil
}

// ScaleWorkload scales a workload (deployment) to specified replicas
func (m *O2Manager) ScaleWorkload(ctx context.Context, workloadID string, replicas int32) error {
	logger := log.FromContext(ctx)
	logger.Info("scaling workload", "workloadID", workloadID, "replicas", replicas)

	parts := strings.SplitN(workloadID, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid workload ID format, expected namespace/name: %s", workloadID)
	}
	namespace, name := parts[0], parts[1]

	// Get current deployment
	deployment := &appsv1.Deployment{}
	if err := m.adaptor.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment); err != nil {
		return fmt.Errorf("failed to get deployment %s: %w", workloadID, err)
	}

	// Store original replica count for rollback
	originalReplicas := *deployment.Spec.Replicas

	// Update replica count
	deployment.Spec.Replicas = &replicas
	if err := m.adaptor.kubeClient.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to scale deployment %s: %w", workloadID, err)
	}

	// Wait for scaling to complete with timeout
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Rollback on timeout
			deployment.Spec.Replicas = &originalReplicas
			m.adaptor.kubeClient.Update(ctx, deployment)
			return fmt.Errorf("timeout scaling workload %s to %d replicas", workloadID, replicas)
		case <-ticker.C:
			// Check scaling progress
			if err := m.adaptor.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment); err != nil {
				continue
			}
			
			if deployment.Status.ReadyReplicas == replicas {
				logger.Info("workload scaled successfully", 
					"workloadID", workloadID, 
					"oldReplicas", originalReplicas,
					"newReplicas", replicas)
				return nil
			}
		}
	}
}

// DeployVNF deploys a VNF using the provided descriptor
func (m *O2Manager) DeployVNF(ctx context.Context, spec *VNFDescriptor) (*DeploymentStatus, error) {
	logger := log.FromContext(ctx)
	logger.Info("deploying VNF", "name", spec.Name, "type", spec.Type)

	// Convert VNFDescriptor to VNFDeployRequest
	deployReq := &VNFDeployRequest{
		Name:            spec.Name,
		Namespace:       "o-ran-vnfs", // Default namespace for VNFs
		VNFPackageID:    fmt.Sprintf("%s-%s", spec.Type, spec.Version),
		FlavorID:        "standard",
		Image:           spec.Image,
		Replicas:        spec.Replicas,
		Resources:       spec.Resources,
		Environment:     spec.Environment,
		Ports:           spec.Ports,
		VolumeConfig:    spec.VolumeConfig,
		NetworkConfig:   spec.NetworkConfig,
		SecurityContext: spec.SecurityContext,
		Metadata: map[string]string{
			"vnf-type":    spec.Type,
			"vnf-version": spec.Version,
			"vnf-vendor":  spec.Vendor,
		},
	}

	// Add metadata from descriptor
	for k, v := range spec.Metadata {
		deployReq.Metadata[k] = v
	}

	// Deploy the VNF
	instance, err := m.adaptor.DeployVNF(ctx, deployReq)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy VNF %s: %w", spec.Name, err)
	}

	// Create enhanced deployment status
	status := &DeploymentStatus{
		ID:                instance.ID,
		Name:              instance.Name,
		Status:            "PENDING",
		Phase:             "Creating",
		Replicas:          spec.Replicas,
		ReadyReplicas:     0,
		UpdatedReplicas:   0,
		AvailableReplicas: 0,
		LastUpdated:       time.Now(),
		Resources:         instance.Resources,
		Endpoints:         instance.NetworkEndpoints,
	}

	// Monitor deployment progress
	go m.monitorDeployment(context.Background(), instance.ID, status)

	logger.Info("VNF deployment initiated", "vnfID", instance.ID, "type", spec.Type)
	return status, nil
}

// monitorDeployment monitors a deployment's progress and updates status
func (m *O2Manager) monitorDeployment(ctx context.Context, instanceID string, status *DeploymentStatus) {
	logger := log.FromContext(ctx)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			instance, err := m.adaptor.GetVNFInstance(ctx, instanceID)
			if err != nil {
				logger.Error(err, "failed to get VNF instance", "instanceID", instanceID)
				continue
			}

			// Update status based on VNF instance
			status.LastUpdated = time.Now()
			status.Phase = instance.Status.DetailedState

			// In a real implementation, we would update persistent storage
			// and emit events for status changes
		}
	}
}

// Additional interface methods with full implementations

func (a *O2Adaptor) UpdateVNF(ctx context.Context, instanceID string, req *VNFUpdateRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("updating VNF", "instanceID", instanceID)

	// Parse instance ID
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

	// Update deployment spec
	if req.Image != "" {
		deployment.Spec.Template.Spec.Containers[0].Image = req.Image
	}

	if req.Environment != nil {
		deployment.Spec.Template.Spec.Containers[0].Env = req.Environment
	}

	// Update deployment
	if err := a.kubeClient.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	logger.Info("VNF updated successfully", "instanceID", instanceID)
	return nil
}

func (a *O2Adaptor) TerminateVNF(ctx context.Context, instanceID string) error {
	logger := log.FromContext(ctx)
	logger.Info("terminating VNF", "instanceID", instanceID)

	parts := strings.SplitN(instanceID, "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid instance ID format: %s", instanceID)
	}
	namespace, name := parts[0], parts[1]

	// Delete associated service first
	serviceList := &corev1.ServiceList{}
	listOpts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{"app": name}),
	}

	if err := a.kubeClient.List(ctx, serviceList, listOpts); err == nil {
		for _, service := range serviceList.Items {
			if err := a.kubeClient.Delete(ctx, &service); err != nil {
				logger.Error(err, "failed to delete service", "service", service.Name)
			}
		}
	}

	// Delete deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := a.kubeClient.Delete(ctx, deployment); err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	logger.Info("VNF terminated successfully", "instanceID", instanceID)
	return nil
}

func (a *O2Adaptor) ListVNFInstances(ctx context.Context, filter *VNFFilter) ([]*VNFInstance, error) {
	logger := log.FromContext(ctx)

	// List deployments with VNF label
	deploymentList := &appsv1.DeploymentList{}
	listOpts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{"nephoran.com/vnf": "true"}),
	}

	// Apply namespace filter if specified
	if filter != nil && len(filter.Namespaces) > 0 {
		// For simplicity, check first namespace in filter
		listOpts.Namespace = filter.Namespaces[0]
	}

	if err := a.kubeClient.List(ctx, deploymentList, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var instances []*VNFInstance
	for _, deployment := range deploymentList.Items {
		// Convert deployment to VNF instance
		instanceID := fmt.Sprintf("%s-%s", deployment.Namespace, deployment.Name)
		
		// Apply name filter if specified
		if filter != nil && len(filter.Names) > 0 {
			found := false
			for _, name := range filter.Names {
				if deployment.Name == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		instance := &VNFInstance{
			ID:        instanceID,
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Status: VNFInstanceStatus{
				State:           "INSTANTIATED",
				DetailedState:   a.getDeploymentState(&deployment),
				LastStateChange: deployment.CreationTimestamp.Time,
			},
			CreatedAt: deployment.CreationTimestamp.Time,
			UpdatedAt: time.Now(),
			Metadata:  deployment.Labels,
		}

		instances = append(instances, instance)
	}

	logger.Info("listed VNF instances", "count", len(instances))
	return instances, nil
}

func (a *O2Adaptor) GetNodeResources(ctx context.Context, nodeID string) (*NodeResources, error) {
	node, err := a.clientset.CoreV1().Nodes().Get(ctx, nodeID, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", nodeID, err)
	}

	resources := &NodeResources{
		CPU:     node.Status.Capacity.Cpu().String(),
		Memory:  node.Status.Capacity.Memory().String(),
		Storage: node.Status.Capacity.StorageEphemeral().String(),
		Pods:    node.Status.Capacity.Pods().String(),
		Custom:  make(map[string]string),
	}

	// Add custom resource information
	for resourceName, quantity := range node.Status.Capacity {
		if !strings.HasPrefix(string(resourceName), "kubernetes.io/") &&
		   !strings.HasPrefix(string(resourceName), "cpu") &&
		   !strings.HasPrefix(string(resourceName), "memory") &&
		   !strings.HasPrefix(string(resourceName), "storage") &&
		   !strings.HasPrefix(string(resourceName), "pods") {
			resources.Custom[string(resourceName)] = quantity.String()
		}
	}

	return resources, nil
}

func (a *O2Adaptor) ListAvailableResources(ctx context.Context) (*ResourceAvailability, error) {
	nodes, err := a.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	availability := &ResourceAvailability{
		Nodes: make(map[string]*NodeResources),
		Total: &NodeResources{Custom: make(map[string]string)},
	}

	totalCPU := int64(0)
	totalMemory := int64(0)

	for _, node := range nodes.Items {
		nodeResources := &NodeResources{
			CPU:     node.Status.Allocatable.Cpu().String(),
			Memory:  node.Status.Allocatable.Memory().String(),
			Storage: node.Status.Allocatable.StorageEphemeral().String(),
			Pods:    node.Status.Allocatable.Pods().String(),
			Custom:  make(map[string]string),
		}

		// Add to totals
		if cpu := node.Status.Allocatable.Cpu(); !cpu.IsZero() {
			totalCPU += cpu.MilliValue()
		}
		if memory := node.Status.Allocatable.Memory(); !memory.IsZero() {
			totalMemory += memory.Value()
		}

		availability.Nodes[node.Name] = nodeResources
	}

	availability.Total.CPU = fmt.Sprintf("%dm", totalCPU)
	availability.Total.Memory = fmt.Sprintf("%d", totalMemory)

	return availability, nil
}

func (a *O2Adaptor) UpdatePod(ctx context.Context, podName, namespace string, req *PodUpdateRequest) error {
	// Get current pod
	pod := &corev1.Pod{}
	if err := a.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, pod); err != nil {
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Update pod spec (limited fields can be updated)
	if req.Labels != nil {
		pod.Labels = req.Labels
	}
	if req.Annotations != nil {
		pod.Annotations = req.Annotations
	}

	// Update pod
	if err := a.kubeClient.Update(ctx, pod); err != nil {
		return fmt.Errorf("failed to update pod: %w", err)
	}

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
	pod := &corev1.Pod{}
	if err := a.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, pod); err != nil {
		return nil, fmt.Errorf("failed to get pod: %w", err)
	}

	ready := false
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			ready = true
			break
		}
	}

	restartCount := int32(0)
	if len(pod.Status.ContainerStatuses) > 0 {
		restartCount = pod.Status.ContainerStatuses[0].RestartCount
	}

	return &PodStatus{
		Phase:        string(pod.Status.Phase),
		Conditions:   pod.Status.Conditions,
		StartTime:    pod.Status.StartTime,
		Ready:        ready,
		RestartCount: restartCount,
		Message:      pod.Status.Message,
	}, nil
}

func (a *O2Adaptor) UpdateService(ctx context.Context, serviceName, namespace string, req *ServiceUpdateRequest) error {
	// Get current service
	service := &corev1.Service{}
	if err := a.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: serviceName}, service); err != nil {
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Update service spec
	if req.Ports != nil {
		service.Spec.Ports = req.Ports
	}
	if req.Type != "" {
		service.Spec.Type = req.Type
	}
	if req.Labels != nil {
		service.Labels = req.Labels
	}
	if req.Annotations != nil {
		service.Annotations = req.Annotations
	}

	// Update service
	if err := a.kubeClient.Update(ctx, service); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

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
	metrics := &InfrastructureMetrics{
		Timestamp:      time.Now(),
		ClusterMetrics: &ClusterMetrics{},
		NodeMetrics:    make(map[string]*NodeMetrics),
	}

	// Get cluster-wide pod metrics
	pods, err := a.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		totalPods := int32(len(pods.Items))
		runningPods := int32(0)
		failedPods := int32(0)

		for _, pod := range pods.Items {
			switch pod.Status.Phase {
			case corev1.PodRunning:
				runningPods++
			case corev1.PodFailed:
				failedPods++
			}
		}

		metrics.ClusterMetrics.TotalPods = totalPods
		metrics.ClusterMetrics.RunningPods = runningPods
		metrics.ClusterMetrics.FailedPods = failedPods
	}

	// Get node metrics
	nodes, err := a.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, node := range nodes.Items {
			// In a real implementation, we would integrate with metrics server
			nodeMetrics := &NodeMetrics{
				CPUUsage:    50.0, // Mock data
				MemoryUsage: 60.0,
				DiskUsage:   30.0,
				PodCount:    10,
			}
			metrics.NodeMetrics[node.Name] = nodeMetrics
		}
	}

	return metrics, nil
}

func (a *O2Adaptor) GetVNFFaults(ctx context.Context, instanceID string) ([]*VNFFault, error) {
	parts := strings.SplitN(instanceID, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid instance ID format: %s", instanceID)
	}
	namespace, name := parts[0], parts[1]

	var faults []*VNFFault

	// Check deployment events for faults
	events, err := a.clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", name),
	})
	
	if err == nil {
		for _, event := range events.Items {
			if event.Type == "Warning" {
				fault := &VNFFault{
					ID:          fmt.Sprintf("%s-%s", instanceID, event.Name),
					InstanceID:  instanceID,
					FaultType:   event.Reason,
					Severity:    "WARNING",
					Description: event.Message,
					DetectedAt:  event.CreationTimestamp.Time,
					Status:      "ACTIVE",
					Recommendations: []string{
						"Check pod logs for detailed error information",
						"Verify resource availability",
						"Check network connectivity",
					},
				}
				faults = append(faults, fault)
			}
		}
	}

	return faults, nil
}

func (a *O2Adaptor) TriggerVNFHealing(ctx context.Context, instanceID string, req *HealingRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("triggering VNF healing", "instanceID", instanceID, "healingType", req.HealingType)

	parts := strings.SplitN(instanceID, "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid instance ID format: %s", instanceID)
	}
	namespace, name := parts[0], parts[1]

	switch req.HealingType {
	case "RESTART":
		// Restart deployment by updating a label to trigger pod recreation
		deployment := &appsv1.Deployment{}
		if err := a.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment); err != nil {
			return fmt.Errorf("failed to get deployment: %w", err)
		}

		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = make(map[string]string)
		}
		deployment.Spec.Template.Annotations["nephoran.com/restart"] = time.Now().Format(time.RFC3339)

		if err := a.kubeClient.Update(ctx, deployment); err != nil {
			return fmt.Errorf("failed to restart deployment: %w", err)
		}

	case "RECREATE":
		// Delete and recreate deployment
		deployment := &appsv1.Deployment{}
		if err := a.kubeClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment); err != nil {
			return fmt.Errorf("failed to get deployment: %w", err)
		}

		// Store deployment spec for recreation
		deploymentSpec := deployment.Spec.DeepCopy()
		
		// Delete deployment
		if err := a.kubeClient.Delete(ctx, deployment); err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}

		// Wait a moment for cleanup
		time.Sleep(5 * time.Second)

		// Recreate deployment
		newDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels:    deployment.Labels,
			},
			Spec: *deploymentSpec,
		}

		if err := a.kubeClient.Create(ctx, newDeployment); err != nil {
			return fmt.Errorf("failed to recreate deployment: %w", err)
		}

	case "SCALE":
		// Scale deployment based on parameters
		replicas, ok := req.Parameters["replicas"].(float64)
		if !ok {
			return fmt.Errorf("replicas parameter required for SCALE healing")
		}

		scaleReq := &VNFScaleRequest{
			ScaleType:     "SCALE_OUT",
			NumberOfSteps: int32(replicas),
		}

		if err := a.ScaleVNF(ctx, instanceID, scaleReq); err != nil {
			return fmt.Errorf("failed to scale deployment: %w", err)
		}

	default:
		return fmt.Errorf("unsupported healing type: %s", req.HealingType)
	}

	logger.Info("VNF healing completed", "instanceID", instanceID, "healingType", req.HealingType)
	return nil
}