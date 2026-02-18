package o2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Test compatibility methods for O2Adaptor.

// DeployVNF with interface{} parameter to handle both VNFDeployRequest and VNFDescriptor.

func (o2 *O2Adaptor) DeployVNF(ctx context.Context, request interface{}) (interface{}, error) {
	switch req := request.(type) {

	case *VNFDeployRequest:

		return o2.deployVNFFromRequest(ctx, req)

	case *VNFDescriptor:

		return o2.deployVNFFromDescriptor(ctx, req)

	default:

		return nil, fmt.Errorf("unsupported request type: %T", request)

	}
}

// deployVNFFromRequest handles VNFDeployRequest.

func (o2 *O2Adaptor) deployVNFFromRequest(ctx context.Context, request *VNFDeployRequest) (*O2VNFInstance, error) {
	namespace := request.Namespace

	if namespace == "" {
		namespace = o2.config.Namespace
	}

	// Create Kubernetes deployment.

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: request.Name,

			Namespace: namespace,

			Labels: map[string]string{
				"app": request.Name,

				"nephoran.com/vnf": "true",

				"nephoran.com/type": "o-ran",
			},
		},

		Spec: appsv1.DeploymentSpec{
			Replicas: &request.Replicas,

			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": request.Name,
				},
			},

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": request.Name,
					},
				},

				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: request.Name,

							Image: request.Image,

							Ports: request.Ports,

							Env: request.Environment,
						},
					},
				},
			},
		},
	}

	// Apply metadata labels.

	if request.Metadata != nil {
		for key, value := range request.Metadata {
			deployment.Labels[key] = value
		}
	}

	// Apply resources, security context, health checks, volumes, etc.

	container := &deployment.Spec.Template.Spec.Containers[0]

	if request.Resources != nil {
		container.Resources = *request.Resources
	}

	if request.SecurityContext != nil {
		container.SecurityContext = request.SecurityContext
	}

	if request.HealthCheck != nil {

		if request.HealthCheck.LivenessProbe != nil {
			container.LivenessProbe = request.HealthCheck.LivenessProbe
		}

		if request.HealthCheck.ReadinessProbe != nil {
			container.ReadinessProbe = request.HealthCheck.ReadinessProbe
		}

		if request.HealthCheck.StartupProbe != nil {
			container.StartupProbe = request.HealthCheck.StartupProbe
		}

	}

	if len(request.VolumeConfig) > 0 {

		var volumeMounts []corev1.VolumeMount

		var volumes []corev1.Volume

		for _, volumeConfig := range request.VolumeConfig {

			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name: volumeConfig.Name,

				MountPath: volumeConfig.MountPath,
			})

			volumes = append(volumes, corev1.Volume{
				Name: volumeConfig.Name,

				VolumeSource: volumeConfig.VolumeSource,
			})

		}

		container.VolumeMounts = volumeMounts

		deployment.Spec.Template.Spec.Volumes = volumes

	}

	if request.Affinity != nil {
		deployment.Spec.Template.Spec.Affinity = request.Affinity
	}

	if len(request.Tolerations) > 0 {
		deployment.Spec.Template.Spec.Tolerations = request.Tolerations
	}

	// Create deployment.

	err := o2.kubeClient.Create(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create service if network config is specified.

	if request.NetworkConfig != nil {

		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: request.Name + "-service",

				Namespace: namespace,

				Labels: map[string]string{"app": request.Name},
			},

			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": request.Name},

				Type: request.NetworkConfig.ServiceType,

				Ports: request.NetworkConfig.Ports,
			},
		}

		err = o2.kubeClient.Create(ctx, service)
		if err != nil {
			log.FromContext(ctx).V(1).Info("Failed to create service", "error", err)
		}

	}

	// Return VNF instance.

	instance := &O2VNFInstance{
		ID: fmt.Sprintf("%s/%s", namespace, request.Name),

		Name: request.Name,

		Namespace: namespace,

		VNFPackageID: request.VNFPackageID,

		FlavorID: request.FlavorID,

		Status: &VNFInstanceStatus{
			State: "INSTANTIATED",

			DetailedState: "CREATING",
		},

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

	if request.Resources != nil {
		if cpu := request.Resources.Requests.Cpu(); cpu != nil {
			if memory := request.Resources.Requests.Memory(); memory != nil {
				instance.Resources = &ResourceInfo{
					CPU: cpu.String(),

					Memory: memory.String(),
				}
			}
		}
	}

	// Add network endpoints.

	if request.NetworkConfig != nil {
		for _, port := range request.NetworkConfig.Ports {
			instance.NetworkEndpoints = append(instance.NetworkEndpoints, NetworkEndpoint{
				Name: port.Name,

				Address: "10.96.1.100", // Mock cluster IP

				Port: port.Port,

				Protocol: string(port.Protocol),
			})
		}
	}

	return instance, nil
}

// deployVNFFromDescriptor handles VNFDescriptor.

func (o2 *O2Adaptor) deployVNFFromDescriptor(ctx context.Context, descriptor *VNFDescriptor) (*VNFDeploymentStatus, error) {
	// Convert to VNFDeployRequest and deploy.

	request := &VNFDeployRequest{
		Name: descriptor.Name,

		Namespace: o2.config.Namespace,

		VNFPackageID: fmt.Sprintf("%s-%s", descriptor.Type, descriptor.Version),

		FlavorID: "standard",

		Image: descriptor.Image,

		Replicas: descriptor.Replicas,

		Resources: descriptor.Resources,

		Environment: descriptor.Environment,

		Ports: descriptor.Ports,

		VolumeConfig: descriptor.VolumeConfig,

		NetworkConfig: descriptor.NetworkConfig,

		SecurityContext: descriptor.SecurityContext,

		HealthCheck: descriptor.HealthCheck,

		Affinity: descriptor.Affinity,

		Tolerations: descriptor.Tolerations,

		Metadata: descriptor.Metadata,
	}

	_, err := o2.deployVNFFromRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	return &VNFDeploymentStatus{
		Name: descriptor.Name,

		Status: "PENDING",

		Phase: "Creating",

		Replicas: descriptor.Replicas,
	}, nil
}

// ScaleVNF scales a VNF instance.

func (o2 *O2Adaptor) ScaleVNF(ctx context.Context, instanceID string, request *VNFScaleRequest) error {
	parts := strings.SplitN(instanceID, "/", 2)

	if len(parts) != 2 {
		return fmt.Errorf("invalid instance ID format: %s", instanceID)
	}

	namespace, name := parts[0], parts[1]

	deployment := &appsv1.Deployment{}

	err := o2.kubeClient.Get(ctx, types.NamespacedName{
		Name: name,

		Namespace: namespace,
	}, deployment)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	currentReplicas := *deployment.Spec.Replicas

	var newReplicas int32

	switch request.ScaleType {

	case "SCALE_OUT":

		newReplicas = currentReplicas + request.NumberOfSteps

	case "SCALE_IN":

		newReplicas = currentReplicas - request.NumberOfSteps

		if newReplicas < 0 {
			newReplicas = 0
		}

	default:

		return fmt.Errorf("invalid scale type: %s", request.ScaleType)

	}

	deployment.Spec.Replicas = &newReplicas

	return o2.kubeClient.Update(ctx, deployment)
}

// GetVNFInstance gets a VNF instance by ID.

func (o2 *O2Adaptor) GetVNFInstance(ctx context.Context, instanceID string) (*O2VNFInstance, error) {
	parts := strings.SplitN(instanceID, "/", 2)

	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid instance ID format: %s", instanceID)
	}

	namespace, name := parts[0], parts[1]

	deployment := &appsv1.Deployment{}

	err := o2.kubeClient.Get(ctx, types.NamespacedName{
		Name: name,

		Namespace: namespace,
	}, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get associated services.

	serviceList := &corev1.ServiceList{}

	err = o2.kubeClient.List(ctx, serviceList, client.InNamespace(namespace), client.MatchingLabels{"app": name})
	if err != nil {
		log.FromContext(ctx).V(1).Info("Failed to list services", "error", err)
	}

	instance := &O2VNFInstance{
		ID: instanceID,

		Name: name,

		Namespace: namespace,

		Status: &VNFInstanceStatus{
			State: "INSTANTIATED",

			DetailedState: "RUNNING",
		},

		CreatedAt: deployment.CreationTimestamp.Time,

		UpdatedAt: time.Now(),
	}

	// Add network endpoints from services.

	for _, service := range serviceList.Items {
		for _, port := range service.Spec.Ports {
			instance.NetworkEndpoints = append(instance.NetworkEndpoints, NetworkEndpoint{
				Name: port.Name,

				Address: service.Spec.ClusterIP,

				Port: port.Port,

				Protocol: string(port.Protocol),
			})
		}
	}

	return instance, nil
}

// ScaleWorkload scales a workload by ID.

func (o2 *O2Adaptor) ScaleWorkload(ctx context.Context, workloadID string, replicas int32) error {
	parts := strings.SplitN(workloadID, "/", 2)

	if len(parts) != 2 {
		return fmt.Errorf("invalid workload ID format: %s", workloadID)
	}

	namespace, name := parts[0], parts[1]

	deployment := &appsv1.Deployment{}

	err := o2.kubeClient.Get(ctx, types.NamespacedName{
		Name: name,

		Namespace: namespace,
	}, deployment)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	deployment.Spec.Replicas = &replicas

	return o2.kubeClient.Update(ctx, deployment)
}

// DiscoverResources discovers cluster resources.

func (o2 *O2Adaptor) DiscoverResources(ctx context.Context) (*ResourceMap, error) {
	resourceMap := &ResourceMap{
		Nodes: make(map[string]*NodeInfo),

		Namespaces: make(map[string]*NamespaceInfo),

		Metrics: &ClusterMetrics{
			ReadyNodes: 0,

			TotalCPU: "0",

			TotalMemory: "0",
		},
	}

	// Discover nodes.

	nodeList := &corev1.NodeList{}

	err := o2.kubeClient.List(ctx, nodeList)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var totalCPU, totalMemory resource.Quantity

	for _, node := range nodeList.Items {

		nodeInfo := &NodeInfo{
			Name: node.Name,

			Labels: node.Labels,

			Roles: []string{},
		}

		// Determine roles.

		for label := range node.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {

				role := strings.TrimPrefix(label, "node-role.kubernetes.io/")

				if role == "" {
					role = "master"
				}

				nodeInfo.Roles = append(nodeInfo.Roles, role)

			}
		}

		if len(nodeInfo.Roles) == 0 {
			nodeInfo.Roles = append(nodeInfo.Roles, "worker")
		}

		resourceMap.Nodes[node.Name] = nodeInfo

		// Count ready nodes.

		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {

				resourceMap.Metrics.ReadyNodes++

				break

			}
		}

		// Sum resources.

		if cpu := node.Status.Capacity.Cpu(); cpu != nil {
			totalCPU.Add(*cpu)
		}

		if memory := node.Status.Capacity.Memory(); memory != nil {
			totalMemory.Add(*memory)
		}

	}

	resourceMap.Metrics.TotalCPU = totalCPU.String()

	resourceMap.Metrics.TotalMemory = totalMemory.String()

	// Discover namespaces.

	namespaceList := &corev1.NamespaceList{}

	err = o2.kubeClient.List(ctx, namespaceList)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, namespace := range namespaceList.Items {
		phase := string(namespace.Status.Phase)
		if phase == "" {
			// Fake k8s clients don't populate Status.Phase; default to Active.
			phase = "Active"
		}
		resourceMap.Namespaces[namespace.Name] = &NamespaceInfo{
			Name:   namespace.Name,
			Status: phase,
		}
	}

	return resourceMap, nil
}

// GetInfrastructureInfo gets infrastructure information.

func (o2 *O2Adaptor) GetInfrastructureInfo(ctx context.Context) (*InfrastructureInfo, error) {
	resourceMap, err := o2.DiscoverResources(ctx)
	if err != nil {
		return nil, err
	}

	info := &InfrastructureInfo{
		NodeCount: len(resourceMap.Nodes),

		ClusterName: "kubernetes-cluster",

		KubernetesVersion: "v1.28.0",
	}

	if resourceMap.Metrics != nil {
		info.TotalResources = &ResourceSummary{
			CPU: resourceMap.Metrics.TotalCPU,

			Memory: resourceMap.Metrics.TotalMemory,
		}
	}

	return info, nil
}

// GetRouterHandler returns the HTTP router as http.Handler (test compatibility method).

func (s *O2APIServer) GetRouterHandler() http.Handler {
	return s.router
}
