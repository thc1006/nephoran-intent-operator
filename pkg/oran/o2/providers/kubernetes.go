package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// KubernetesProvider implements the CloudProvider interface for Kubernetes clusters.

type KubernetesProvider struct {
	name string

	kubeClient client.Client

	clientset kubernetes.Interface

	config map[string]string

	connected bool

	eventCallback EventCallback

	stopChannel chan struct{}

	mutex sync.RWMutex
}

// NewKubernetesProvider creates a new Kubernetes provider instance.

func NewKubernetesProvider(kubeClient client.Client, clientset kubernetes.Interface, config map[string]string) (CloudProvider, error) {

	if config == nil {

		config = make(map[string]string)

	}

	provider := &KubernetesProvider{

		name: "kubernetes",

		kubeClient: kubeClient,

		clientset: clientset,

		config: config,

		stopChannel: make(chan struct{}),
	}

	return provider, nil

}

// GetProviderInfo returns information about this Kubernetes provider.

func (k *KubernetesProvider) GetProviderInfo() *ProviderInfo {

	k.mutex.RLock()

	defer k.mutex.RUnlock()

	return &ProviderInfo{

		Name: k.name,

		Type: ProviderTypeKubernetes,

		Version: "1.0.0",

		Description: "Kubernetes cloud provider for O2 IMS",

		Vendor: "Nephoran",

		Endpoint: k.config["endpoint"],

		Tags: map[string]string{

			"in_cluster": k.config["in_cluster"],

			"kubeconfig": k.config["kubeconfig"],
		},

		LastUpdated: time.Now(),
	}

}

// GetSupportedResourceTypes returns the resource types supported by Kubernetes.

func (k *KubernetesProvider) GetSupportedResourceTypes() []string {

	return []string{

		"deployment",

		"statefulset",

		"daemonset",

		"service",

		"ingress",

		"configmap",

		"secret",

		"persistentvolume",

		"persistentvolumeclaim",

		"storageclass",

		"networkpolicy",

		"pod",

		"replicaset",

		"job",

		"cronjob",
	}

}

// GetCapabilities returns the capabilities of this Kubernetes provider.

func (k *KubernetesProvider) GetCapabilities() *ProviderCapabilities {

	return &ProviderCapabilities{

		ComputeTypes: []string{"deployment", "statefulset", "daemonset", "pod", "job", "cronjob"},

		StorageTypes: []string{"persistentvolume", "persistentvolumeclaim", "storageclass"},

		NetworkTypes: []string{"service", "ingress", "networkpolicy"},

		AcceleratorTypes: []string{"gpu", "fpga"},

		AutoScaling: true,

		LoadBalancing: true,

		Monitoring: true,

		Logging: true,

		Networking: true,

		StorageClasses: true,

		HorizontalPodAutoscaling: true,

		VerticalPodAutoscaling: true,

		ClusterAutoscaling: true,

		Namespaces: true,

		ResourceQuotas: true,

		NetworkPolicies: true,

		RBAC: true,

		MultiZone: true,

		MultiRegion: false,

		BackupRestore: true,

		DisasterRecovery: false,

		Encryption: true,

		SecretManagement: true,

		ImageScanning: false,

		PolicyEngine: true,

		MaxNodes: 1000,

		MaxPods: 30000,

		MaxServices: 5000,

		MaxVolumes: 5000,
	}

}

// Connect establishes connection to the Kubernetes cluster.

func (k *KubernetesProvider) Connect(ctx context.Context) error {

	logger := log.FromContext(ctx)

	logger.Info("connecting to Kubernetes cluster")

	// Test connection by listing nodes.

	_, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})

	if err != nil {

		return fmt.Errorf("failed to connect to Kubernetes cluster: %w", err)

	}

	k.mutex.Lock()

	k.connected = true

	k.mutex.Unlock()

	logger.Info("successfully connected to Kubernetes cluster")

	return nil

}

// Disconnect closes the connection to the Kubernetes cluster.

func (k *KubernetesProvider) Disconnect(ctx context.Context) error {

	logger := log.FromContext(ctx)

	logger.Info("disconnecting from Kubernetes cluster")

	k.mutex.Lock()

	k.connected = false

	k.mutex.Unlock()

	// Stop event watching if running.

	select {

	case k.stopChannel <- struct{}{}:

	default:

	}

	logger.Info("disconnected from Kubernetes cluster")

	return nil

}

// HealthCheck performs a health check on the Kubernetes cluster.

func (k *KubernetesProvider) HealthCheck(ctx context.Context) error {

	// Check if we can list nodes.

	nodes, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 5})

	if err != nil {

		return fmt.Errorf("health check failed: unable to list nodes: %w", err)

	}

	// Check if at least one node is ready.

	readyNodes := 0

	for _, node := range nodes.Items {

		for _, condition := range node.Status.Conditions {

			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {

				readyNodes++

				break

			}

		}

	}

	if readyNodes == 0 {

		return fmt.Errorf("health check failed: no ready nodes found")

	}

	// Check if we can create a test namespace (and delete it).

	testNamespace := &corev1.Namespace{

		ObjectMeta: metav1.ObjectMeta{

			Name: fmt.Sprintf("health-check-%d", time.Now().Unix()),
		},
	}

	createdNs, err := k.clientset.CoreV1().Namespaces().Create(ctx, testNamespace, metav1.CreateOptions{})

	if err != nil {

		return fmt.Errorf("health check failed: unable to create test namespace: %w", err)

	}

	// Clean up test namespace.

	err = k.clientset.CoreV1().Namespaces().Delete(ctx, createdNs.Name, metav1.DeleteOptions{})

	if err != nil {

		// Log warning but don't fail health check.

		logger := log.FromContext(ctx)

		logger.Error(err, "failed to clean up test namespace", "namespace", createdNs.Name)

	}

	return nil

}

// Close closes any resources held by the provider.

func (k *KubernetesProvider) Close() error {

	k.mutex.Lock()

	defer k.mutex.Unlock()

	// Stop event watching.

	select {

	case k.stopChannel <- struct{}{}:

	default:

	}

	k.connected = false

	return nil

}

// CreateResource creates a new Kubernetes resource.

func (k *KubernetesProvider) CreateResource(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {

	logger := log.FromContext(ctx)

	logger.Info("creating Kubernetes resource", "type", req.Type, "name", req.Name, "namespace", req.Namespace)

	switch strings.ToLower(req.Type) {

	case "deployment":

		return k.CreateDeployment(ctx, req)

	case "service":

		return k.CreateService(ctx, req)

	case "configmap":

		return k.CreateConfigMap(ctx, req)

	case "secret":

		return k.CreateSecret(ctx, req)

	case "persistentvolumeclaim":

		return k.CreatePersistentVolumeClaim(ctx, req)

	case "ingress":

		return k.CreateIngress(ctx, req)

	default:

		return nil, fmt.Errorf("unsupported resource type: %s", req.Type)

	}

}

// GetResource retrieves a Kubernetes resource.

func (k *KubernetesProvider) GetResource(ctx context.Context, resourceID string) (*ResourceResponse, error) {

	logger := log.FromContext(ctx)

	logger.V(1).Info("getting Kubernetes resource", "resourceID", resourceID)

	// Parse resourceID format: namespace/type/name.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return nil, fmt.Errorf("invalid resourceID format, expected namespace/type/name or type/name: %s", resourceID)

	}

	var namespace, resourceType, name string

	if len(parts) == 2 {

		resourceType, name = parts[0], parts[1]

		namespace = "default"

	} else {

		namespace, resourceType, name = parts[0], parts[1], parts[2]

	}

	switch strings.ToLower(resourceType) {

	case "deployment":

		return k.GetDeploymentResource(ctx, namespace, name)

	case "service":

		return k.GetService(ctx, namespace, name)

	case "configmap":

		return k.GetConfigMap(ctx, namespace, name)

	case "secret":

		return k.GetSecret(ctx, namespace, name)

	case "persistentvolumeclaim":

		return k.GetPersistentVolumeClaim(ctx, namespace, name)

	default:

		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)

	}

}

// UpdateResource updates a Kubernetes resource.

func (k *KubernetesProvider) UpdateResource(ctx context.Context, resourceID string, req *UpdateResourceRequest) (*ResourceResponse, error) {

	logger := log.FromContext(ctx)

	logger.Info("updating Kubernetes resource", "resourceID", resourceID)

	// Parse resourceID.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)

	}

	var namespace, resourceType, name string

	if len(parts) == 2 {

		resourceType, name = parts[0], parts[1]

		namespace = "default"

	} else {

		namespace, resourceType, name = parts[0], parts[1], parts[2]

	}

	switch strings.ToLower(resourceType) {

	case "deployment":

		return k.UpdateDeploymentResource(ctx, namespace, name, req)

	case "service":

		return k.UpdateService(ctx, namespace, name, req)

	case "configmap":

		return k.UpdateConfigMap(ctx, namespace, name, req)

	case "secret":

		return k.UpdateSecret(ctx, namespace, name, req)

	default:

		return nil, fmt.Errorf("unsupported resource type for update: %s", resourceType)

	}

}

// DeleteResource deletes a Kubernetes resource.

func (k *KubernetesProvider) DeleteResource(ctx context.Context, resourceID string) error {

	logger := log.FromContext(ctx)

	logger.Info("deleting Kubernetes resource", "resourceID", resourceID)

	// Parse resourceID.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return fmt.Errorf("invalid resourceID format: %s", resourceID)

	}

	var namespace, resourceType, name string

	if len(parts) == 2 {

		resourceType, name = parts[0], parts[1]

		namespace = "default"

	} else {

		namespace, resourceType, name = parts[0], parts[1], parts[2]

	}

	switch strings.ToLower(resourceType) {

	case "deployment":

		return k.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	case "service":

		return k.clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	case "configmap":

		return k.clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	case "secret":

		return k.clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	case "persistentvolumeclaim":

		return k.clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	default:

		return fmt.Errorf("unsupported resource type for deletion: %s", resourceType)

	}

}

// ListResources lists Kubernetes resources with optional filtering.

func (k *KubernetesProvider) ListResources(ctx context.Context, filter *ResourceFilter) ([]*ResourceResponse, error) {

	logger := log.FromContext(ctx)

	logger.V(1).Info("listing Kubernetes resources", "filter", filter)

	var resources []*ResourceResponse

	// If specific types are requested, only query those.

	resourceTypes := filter.Types

	if len(resourceTypes) == 0 {

		// Default to common resource types.

		resourceTypes = []string{"deployment", "service", "configmap", "secret"}

	}

	for _, resourceType := range resourceTypes {

		typeResources, err := k.listResourcesByType(ctx, resourceType, filter)

		if err != nil {

			logger.Error(err, "failed to list resources", "type", resourceType)

			continue

		}

		resources = append(resources, typeResources...)

	}

	// Apply additional filtering and limits.

	resources = k.applyResourceFilters(resources, filter)

	return resources, nil

}

// Deploy creates a deployment using various template types.

func (k *KubernetesProvider) Deploy(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {

	logger := log.FromContext(ctx)

	logger.Info("deploying template", "name", req.Name, "type", req.TemplateType)

	switch strings.ToLower(req.TemplateType) {

	case "kubernetes", "k8s", "yaml":

		return k.deployKubernetesTemplate(ctx, req)

	case "helm":

		return k.deployHelmTemplate(ctx, req)

	default:

		return nil, fmt.Errorf("unsupported template type: %s", req.TemplateType)

	}

}

// GetDeployment retrieves a deployment.

func (k *KubernetesProvider) GetDeployment(ctx context.Context, deploymentID string) (*DeploymentResponse, error) {

	// Parse deploymentID (could be namespace/name or name).

	parts := strings.Split(deploymentID, "/")

	var namespace, name string

	if len(parts) == 1 {

		namespace = "default"

		name = parts[0]

	} else {

		namespace, name = parts[0], parts[1]

	}

	deployment, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get deployment: %w", err)

	}

	return k.convertDeploymentToResponse(deployment), nil

}

// UpdateDeployment updates a deployment.

func (k *KubernetesProvider) UpdateDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*DeploymentResponse, error) {

	logger := log.FromContext(ctx)

	logger.Info("updating deployment", "deploymentID", deploymentID)

	// Get current deployment.

	current, err := k.GetDeployment(ctx, deploymentID)

	if err != nil {

		return nil, fmt.Errorf("failed to get current deployment: %w", err)

	}

	// Apply updates based on template type.

	switch strings.ToLower(current.TemplateType) {

	case "kubernetes", "k8s", "yaml":

		return k.updateKubernetesDeployment(ctx, deploymentID, req)

	case "helm":

		return k.updateHelmDeployment(ctx, deploymentID, req)

	default:

		return nil, fmt.Errorf("unsupported template type for update: %s", current.TemplateType)

	}

}

// DeleteDeployment deletes a deployment.

func (k *KubernetesProvider) DeleteDeployment(ctx context.Context, deploymentID string) error {

	logger := log.FromContext(ctx)

	logger.Info("deleting deployment", "deploymentID", deploymentID)

	// Parse deploymentID.

	parts := strings.Split(deploymentID, "/")

	var namespace, name string

	if len(parts) == 1 {

		namespace = "default"

		name = parts[0]

	} else {

		namespace, name = parts[0], parts[1]

	}

	// Delete the deployment.

	err := k.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	if err != nil && !errors.IsNotFound(err) {

		return fmt.Errorf("failed to delete deployment: %w", err)

	}

	// Also delete associated services.

	services, err := k.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{

		LabelSelector: fmt.Sprintf("app=%s", name),
	})

	if err == nil {

		for _, service := range services.Items {

			k.clientset.CoreV1().Services(namespace).Delete(ctx, service.Name, metav1.DeleteOptions{})

		}

	}

	return nil

}

// ListDeployments lists deployments with optional filtering.

func (k *KubernetesProvider) ListDeployments(ctx context.Context, filter *DeploymentFilter) ([]*DeploymentResponse, error) {

	logger := log.FromContext(ctx)

	logger.V(1).Info("listing deployments", "filter", filter)

	var listOptions metav1.ListOptions

	if len(filter.Names) > 0 {

		// Convert names to field selector.

		var fieldSelectors []string

		for _, name := range filter.Names {

			fieldSelectors = append(fieldSelectors, fmt.Sprintf("metadata.name=%s", name))

		}

		listOptions.FieldSelector = strings.Join(fieldSelectors, ",")

	}

	if filter.Labels != nil {

		listOptions.LabelSelector = labels.FormatLabels(filter.Labels)

	}

	if filter.Limit > 0 {

		listOptions.Limit = int64(filter.Limit)

	}

	var deployments []*DeploymentResponse

	// If specific namespaces are requested, query them.

	namespaces := filter.Namespaces

	if len(namespaces) == 0 {

		namespaces = []string{"default"}

	}

	for _, namespace := range namespaces {

		deploymentList, err := k.clientset.AppsV1().Deployments(namespace).List(ctx, listOptions)

		if err != nil {

			logger.Error(err, "failed to list deployments", "namespace", namespace)

			continue

		}

		for _, deployment := range deploymentList.Items {

			resp := k.convertDeploymentToResponse(&deployment)

			// Apply status filter if specified.

			if len(filter.Statuses) > 0 {

				found := false

				for _, status := range filter.Statuses {

					if resp.Status == status {

						found = true

						break

					}

				}

				if !found {

					continue

				}

			}

			deployments = append(deployments, resp)

		}

	}

	return deployments, nil

}

// ScaleResource scales a Kubernetes resource.

func (k *KubernetesProvider) ScaleResource(ctx context.Context, resourceID string, req *ScaleRequest) error {

	logger := log.FromContext(ctx)

	logger.Info("scaling resource", "resourceID", resourceID, "type", req.Type, "direction", req.Direction)

	// Parse resourceID.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return fmt.Errorf("invalid resourceID format: %s", resourceID)

	}

	var namespace, resourceType, name string

	if len(parts) == 2 {

		resourceType, name = parts[0], parts[1]

		namespace = "default"

	} else {

		namespace, resourceType, name = parts[0], parts[1], parts[2]

	}

	switch strings.ToLower(resourceType) {

	case "deployment":

		return k.scaleDeployment(ctx, namespace, name, req)

	case "statefulset":

		return k.scaleStatefulSet(ctx, namespace, name, req)

	default:

		return fmt.Errorf("resource type %s does not support scaling", resourceType)

	}

}

// GetScalingCapabilities returns the scaling capabilities of a resource.

func (k *KubernetesProvider) GetScalingCapabilities(ctx context.Context, resourceID string) (*ScalingCapabilities, error) {

	// Parse resourceID to determine type.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)

	}

	resourceType := parts[len(parts)-2] // Second to last part is the type

	switch strings.ToLower(resourceType) {

	case "deployment", "statefulset":

		return &ScalingCapabilities{

			HorizontalScaling: true,

			VerticalScaling: false, // Requires VPA operator

			MinReplicas: 0,

			MaxReplicas: 1000,

			SupportedMetrics: []string{"cpu", "memory"},

			ScaleUpCooldown: 30 * time.Second,

			ScaleDownCooldown: 30 * time.Second,
		}, nil

	default:

		return &ScalingCapabilities{

			HorizontalScaling: false,

			VerticalScaling: false,

			MinReplicas: 1,

			MaxReplicas: 1,
		}, nil

	}

}

// GetMetrics returns cluster-level metrics.

func (k *KubernetesProvider) GetMetrics(ctx context.Context) (map[string]interface{}, error) {

	metrics := make(map[string]interface{})

	// Get node count and status.

	nodes, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})

	if err == nil {

		totalNodes := len(nodes.Items)

		readyNodes := 0

		for _, node := range nodes.Items {

			for _, condition := range node.Status.Conditions {

				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {

					readyNodes++

					break

				}

			}

		}

		metrics["nodes_total"] = totalNodes

		metrics["nodes_ready"] = readyNodes

	}

	// Get pod count across all namespaces.

	pods, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})

	if err == nil {

		totalPods := len(pods.Items)

		runningPods := 0

		for _, pod := range pods.Items {

			if pod.Status.Phase == corev1.PodRunning {

				runningPods++

			}

		}

		metrics["pods_total"] = totalPods

		metrics["pods_running"] = runningPods

	}

	// Get namespace count.

	namespaces, err := k.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})

	if err == nil {

		metrics["namespaces_total"] = len(namespaces.Items)

	}

	// Get service count.

	services, err := k.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})

	if err == nil {

		metrics["services_total"] = len(services.Items)

	}

	metrics["timestamp"] = time.Now().Unix()

	return metrics, nil

}

// GetResourceMetrics returns metrics for a specific resource.

func (k *KubernetesProvider) GetResourceMetrics(ctx context.Context, resourceID string) (map[string]interface{}, error) {

	// Parse resourceID.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)

	}

	var namespace, resourceType, name string

	if len(parts) == 2 {

		resourceType, name = parts[0], parts[1]

		namespace = "default"

	} else {

		namespace, resourceType, name = parts[0], parts[1], parts[2]

	}

	metrics := make(map[string]interface{})

	switch strings.ToLower(resourceType) {

	case "deployment":

		deployment, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})

		if err != nil {

			return nil, fmt.Errorf("failed to get deployment: %w", err)

		}

		metrics["desired_replicas"] = *deployment.Spec.Replicas

		metrics["ready_replicas"] = deployment.Status.ReadyReplicas

		metrics["available_replicas"] = deployment.Status.AvailableReplicas

		metrics["updated_replicas"] = deployment.Status.UpdatedReplicas

	case "service":

		service, err := k.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})

		if err != nil {

			return nil, fmt.Errorf("failed to get service: %w", err)

		}

		metrics["type"] = string(service.Spec.Type)

		metrics["port_count"] = len(service.Spec.Ports)

		metrics["cluster_ip"] = service.Spec.ClusterIP

		// Get endpoint count.

		endpoints, err := k.clientset.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})

		if err == nil {

			endpointCount := 0

			for _, subset := range endpoints.Subsets {

				endpointCount += len(subset.Addresses)

			}

			metrics["endpoint_count"] = endpointCount

		}

	}

	metrics["timestamp"] = time.Now().Unix()

	return metrics, nil

}

// GetResourceHealth returns the health status of a resource.

func (k *KubernetesProvider) GetResourceHealth(ctx context.Context, resourceID string) (*HealthStatus, error) {

	// Parse resourceID.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)

	}

	var namespace, resourceType, name string

	if len(parts) == 2 {

		resourceType, name = parts[0], parts[1]

		namespace = "default"

	} else {

		namespace, resourceType, name = parts[0], parts[1], parts[2]

	}

	switch strings.ToLower(resourceType) {

	case "deployment":

		return k.getDeploymentHealth(ctx, namespace, name)

	case "service":

		return k.getServiceHealth(ctx, namespace, name)

	case "pod":

		return k.getPodHealth(ctx, namespace, name)

	default:

		return &HealthStatus{

			Status: HealthStatusUnknown,

			Message: fmt.Sprintf("Health check not implemented for resource type: %s", resourceType),

			LastUpdated: time.Now(),
		}, nil

	}

}

// CreateNetworkService creates a network service (Service, Ingress, or NetworkPolicy).

func (k *KubernetesProvider) CreateNetworkService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {

	logger := log.FromContext(ctx)

	logger.Info("creating network service", "type", req.Type, "name", req.Name)

	switch strings.ToLower(req.Type) {

	case "service":

		return k.createKubernetesService(ctx, req)

	case "ingress":

		return k.createKubernetesIngress(ctx, req)

	case "networkpolicy":

		return k.createNetworkPolicy(ctx, req)

	default:

		return nil, fmt.Errorf("unsupported network service type: %s", req.Type)

	}

}

// GetNetworkService retrieves a network service.

func (k *KubernetesProvider) GetNetworkService(ctx context.Context, serviceID string) (*NetworkServiceResponse, error) {

	// Parse serviceID format: namespace/type/name.

	parts := strings.Split(serviceID, "/")

	if len(parts) != 3 {

		return nil, fmt.Errorf("invalid serviceID format, expected namespace/type/name: %s", serviceID)

	}

	namespace, serviceType, name := parts[0], parts[1], parts[2]

	switch strings.ToLower(serviceType) {

	case "service":

		return k.getKubernetesService(ctx, namespace, name)

	case "ingress":

		return k.getKubernetesIngress(ctx, namespace, name)

	case "networkpolicy":

		return k.getKubernetesNetworkPolicy(ctx, namespace, name)

	default:

		return nil, fmt.Errorf("unsupported network service type: %s", serviceType)

	}

}

// DeleteNetworkService deletes a network service.

func (k *KubernetesProvider) DeleteNetworkService(ctx context.Context, serviceID string) error {

	// Parse serviceID.

	parts := strings.Split(serviceID, "/")

	if len(parts) != 3 {

		return fmt.Errorf("invalid serviceID format: %s", serviceID)

	}

	namespace, serviceType, name := parts[0], parts[1], parts[2]

	switch strings.ToLower(serviceType) {

	case "service":

		return k.clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	case "ingress":

		return k.clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	case "networkpolicy":

		return k.clientset.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	default:

		return fmt.Errorf("unsupported network service type: %s", serviceType)

	}

}

// ListNetworkServices lists network services with filtering.

func (k *KubernetesProvider) ListNetworkServices(ctx context.Context, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error) {

	var services []*NetworkServiceResponse

	serviceTypes := filter.Types

	if len(serviceTypes) == 0 {

		serviceTypes = []string{"service", "ingress", "networkpolicy"}

	}

	namespaces := filter.Namespaces

	if len(namespaces) == 0 {

		namespaces = []string{"default"}

	}

	for _, serviceType := range serviceTypes {

		for _, namespace := range namespaces {

			typeServices, err := k.listNetworkServicesByType(ctx, serviceType, namespace, filter)

			if err != nil {

				continue // Log error but continue with other types

			}

			services = append(services, typeServices...)

		}

	}

	return services, nil

}

// CreateStorageResource creates a storage resource.

func (k *KubernetesProvider) CreateStorageResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {

	logger := log.FromContext(ctx)

	logger.Info("creating storage resource", "type", req.Type, "name", req.Name)

	switch strings.ToLower(req.Type) {

	case "persistentvolumeclaim":

		return k.createPVC(ctx, req)

	case "storageclass":

		return k.createStorageClass(ctx, req)

	default:

		return nil, fmt.Errorf("unsupported storage resource type: %s", req.Type)

	}

}

// GetStorageResource retrieves a storage resource.

func (k *KubernetesProvider) GetStorageResource(ctx context.Context, resourceID string) (*StorageResourceResponse, error) {

	// Parse resourceID.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return nil, fmt.Errorf("invalid resourceID format: %s", resourceID)

	}

	var namespace, resourceType, name string

	if len(parts) == 2 {

		resourceType, name = parts[0], parts[1]

		namespace = "default"

	} else {

		namespace, resourceType, name = parts[0], parts[1], parts[2]

	}

	switch strings.ToLower(resourceType) {

	case "persistentvolumeclaim":

		return k.getPVC(ctx, namespace, name)

	case "storageclass":

		return k.getStorageClass(ctx, name)

	default:

		return nil, fmt.Errorf("unsupported storage resource type: %s", resourceType)

	}

}

// DeleteStorageResource deletes a storage resource.

func (k *KubernetesProvider) DeleteStorageResource(ctx context.Context, resourceID string) error {

	// Parse resourceID.

	parts := strings.Split(resourceID, "/")

	if len(parts) < 2 {

		return fmt.Errorf("invalid resourceID format: %s", resourceID)

	}

	var namespace, resourceType, name string

	if len(parts) == 2 {

		resourceType, name = parts[0], parts[1]

		namespace = "default"

	} else {

		namespace, resourceType, name = parts[0], parts[1], parts[2]

	}

	switch strings.ToLower(resourceType) {

	case "persistentvolumeclaim":

		return k.clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})

	case "storageclass":

		return k.clientset.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})

	default:

		return fmt.Errorf("unsupported storage resource type: %s", resourceType)

	}

}

// ListStorageResources lists storage resources.

func (k *KubernetesProvider) ListStorageResources(ctx context.Context, filter *StorageResourceFilter) ([]*StorageResourceResponse, error) {

	var resources []*StorageResourceResponse

	resourceTypes := filter.Types

	if len(resourceTypes) == 0 {

		resourceTypes = []string{"persistentvolumeclaim", "storageclass"}

	}

	for _, resourceType := range resourceTypes {

		typeResources, err := k.listStorageResourcesByType(ctx, resourceType, filter)

		if err != nil {

			continue

		}

		resources = append(resources, typeResources...)

	}

	return resources, nil

}

// SubscribeToEvents subscribes to Kubernetes events.

func (k *KubernetesProvider) SubscribeToEvents(ctx context.Context, callback EventCallback) error {

	logger := log.FromContext(ctx)

	logger.Info("subscribing to Kubernetes events")

	k.mutex.Lock()

	k.eventCallback = callback

	k.mutex.Unlock()

	// Start watching events in a separate goroutine.

	go k.watchEvents(ctx)

	return nil

}

// UnsubscribeFromEvents unsubscribes from Kubernetes events.

func (k *KubernetesProvider) UnsubscribeFromEvents(ctx context.Context) error {

	logger := log.FromContext(ctx)

	logger.Info("unsubscribing from Kubernetes events")

	k.mutex.Lock()

	k.eventCallback = nil

	k.mutex.Unlock()

	// Stop event watching.

	select {

	case k.stopChannel <- struct{}{}:

	default:

	}

	return nil

}

// ApplyConfiguration applies provider configuration.

func (k *KubernetesProvider) ApplyConfiguration(ctx context.Context, config *ProviderConfiguration) error {

	logger := log.FromContext(ctx)

	logger.Info("applying provider configuration", "name", config.Name)

	k.mutex.Lock()

	defer k.mutex.Unlock()

	// Update internal configuration.

	for key, value := range config.Parameters {

		if strValue, ok := value.(string); ok {

			k.config[key] = strValue

		}

	}

	return nil

}

// GetConfiguration retrieves current provider configuration.

func (k *KubernetesProvider) GetConfiguration(ctx context.Context) (*ProviderConfiguration, error) {

	k.mutex.RLock()

	defer k.mutex.RUnlock()

	config := &ProviderConfiguration{

		Name: k.name,

		Type: ProviderTypeKubernetes,

		Version: "1.0.0",

		Enabled: k.connected,

		Parameters: make(map[string]interface{}),
	}

	for key, value := range k.config {

		config.Parameters[key] = value

	}

	return config, nil

}

// ValidateConfiguration validates provider configuration.

func (k *KubernetesProvider) ValidateConfiguration(ctx context.Context, config *ProviderConfiguration) error {

	// Validate required parameters.

	if config.Type != ProviderTypeKubernetes {

		return fmt.Errorf("invalid provider type: expected %s, got %s", ProviderTypeKubernetes, config.Type)

	}

	// Validate parameters if any specific validation is needed.

	// For now, basic validation is sufficient.

	return nil

}

// Resource creation methods (thin wrappers for client-go).

// CreateDeployment creates a Kubernetes deployment.

func (k *KubernetesProvider) CreateDeployment(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {

	// Convert request specification to Deployment.

	spec, ok := req.Specification["deployment"].(map[string]interface{})

	if !ok {

		return nil, fmt.Errorf("invalid deployment specification")

	}

	// Extract replica count from specification.

	replicas := int32(1)

	if replicaCount, ok := spec["replicas"].(float64); ok {

		replicas = int32(replicaCount)

	}

	// Create Kubernetes Deployment object.

	deployment := &appsv1.Deployment{

		ObjectMeta: metav1.ObjectMeta{

			Name: req.Name,

			Namespace: req.Namespace,

			Labels: req.Labels,

			Annotations: req.Annotations,
		},

		Spec: appsv1.DeploymentSpec{

			Replicas: &replicas,

			Selector: &metav1.LabelSelector{

				MatchLabels: map[string]string{

					"app": req.Name,
				},
			},

			Template: corev1.PodTemplateSpec{

				ObjectMeta: metav1.ObjectMeta{

					Labels: map[string]string{

						"app": req.Name,
					},
				},

				Spec: corev1.PodSpec{

					Containers: []corev1.Container{

						{

							Name: req.Name,

							Image: "nginx:latest", // Default image, should come from spec

						},
					},
				},
			},
		},
	}

	// Create the deployment.

	createdDeployment, err := k.clientset.AppsV1().Deployments(req.Namespace).Create(ctx, deployment, metav1.CreateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to create deployment: %w", err)

	}

	// Convert to response format.

	return k.convertDeploymentToResourceResponse(createdDeployment), nil

}

// CreateService creates a Kubernetes service.

func (k *KubernetesProvider) CreateService(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {

	spec, ok := req.Specification["service"].(map[string]interface{})

	if !ok {

		return nil, fmt.Errorf("invalid service specification")

	}

	service := &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{

			Name: req.Name,

			Namespace: req.Namespace,

			Labels: req.Labels,

			Annotations: req.Annotations,
		},

		Spec: corev1.ServiceSpec{

			Selector: map[string]string{

				"app": req.Name,
			},

			Type: corev1.ServiceTypeClusterIP,

			Ports: []corev1.ServicePort{

				{

					Port: 80,

					TargetPort: intstr.FromInt(80),

					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// Extract service type if specified.

	if serviceType, ok := spec["type"].(string); ok {

		service.Spec.Type = corev1.ServiceType(serviceType)

	}

	createdService, err := k.clientset.CoreV1().Services(req.Namespace).Create(ctx, service, metav1.CreateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to create service: %w", err)

	}

	return k.convertServiceToResourceResponse(createdService), nil

}

// CreateConfigMap creates a Kubernetes ConfigMap.

func (k *KubernetesProvider) CreateConfigMap(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {

	spec, ok := req.Specification["configmap"].(map[string]interface{})

	if !ok {

		return nil, fmt.Errorf("invalid configmap specification")

	}

	data := make(map[string]string)

	if specData, ok := spec["data"].(map[string]interface{}); ok {

		for key, value := range specData {

			if strValue, ok := value.(string); ok {

				data[key] = strValue

			}

		}

	}

	configMap := &corev1.ConfigMap{

		ObjectMeta: metav1.ObjectMeta{

			Name: req.Name,

			Namespace: req.Namespace,

			Labels: req.Labels,

			Annotations: req.Annotations,
		},

		Data: data,
	}

	createdConfigMap, err := k.clientset.CoreV1().ConfigMaps(req.Namespace).Create(ctx, configMap, metav1.CreateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to create configmap: %w", err)

	}

	return k.convertConfigMapToResourceResponse(createdConfigMap), nil

}

// CreateSecret creates a Kubernetes Secret.

func (k *KubernetesProvider) CreateSecret(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {

	spec, ok := req.Specification["secret"].(map[string]interface{})

	if !ok {

		return nil, fmt.Errorf("invalid secret specification")

	}

	data := make(map[string][]byte)

	if specData, ok := spec["data"].(map[string]interface{}); ok {

		for key, value := range specData {

			if strValue, ok := value.(string); ok {

				data[key] = []byte(strValue)

			}

		}

	}

	secret := &corev1.Secret{

		ObjectMeta: metav1.ObjectMeta{

			Name: req.Name,

			Namespace: req.Namespace,

			Labels: req.Labels,

			Annotations: req.Annotations,
		},

		Type: corev1.SecretTypeOpaque,

		Data: data,
	}

	createdSecret, err := k.clientset.CoreV1().Secrets(req.Namespace).Create(ctx, secret, metav1.CreateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to create secret: %w", err)

	}

	return k.convertSecretToResourceResponse(createdSecret), nil

}

// CreatePersistentVolumeClaim creates a Kubernetes PVC.

func (k *KubernetesProvider) CreatePersistentVolumeClaim(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {

	spec, ok := req.Specification["persistentvolumeclaim"].(map[string]interface{})

	if !ok {

		return nil, fmt.Errorf("invalid persistentvolumeclaim specification")

	}

	// Extract size.

	size := "1Gi"

	if specSize, ok := spec["size"].(string); ok {

		size = specSize

	}

	pvc := &corev1.PersistentVolumeClaim{

		ObjectMeta: metav1.ObjectMeta{

			Name: req.Name,

			Namespace: req.Namespace,

			Labels: req.Labels,

			Annotations: req.Annotations,
		},

		Spec: corev1.PersistentVolumeClaimSpec{

			AccessModes: []corev1.PersistentVolumeAccessMode{

				corev1.ReadWriteOnce,
			},

			Resources: corev1.VolumeResourceRequirements{

				Requests: corev1.ResourceList{

					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
	}

	createdPVC, err := k.clientset.CoreV1().PersistentVolumeClaims(req.Namespace).Create(ctx, pvc, metav1.CreateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to create persistentvolumeclaim: %w", err)

	}

	return k.convertPVCToResourceResponse(createdPVC), nil

}

// CreateIngress creates a Kubernetes Ingress.

func (k *KubernetesProvider) CreateIngress(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error) {

	spec, ok := req.Specification["ingress"].(map[string]interface{})

	if !ok {

		return nil, fmt.Errorf("invalid ingress specification")

	}

	ingress := &networkingv1.Ingress{

		ObjectMeta: metav1.ObjectMeta{

			Name: req.Name,

			Namespace: req.Namespace,

			Labels: req.Labels,

			Annotations: req.Annotations,
		},

		Spec: networkingv1.IngressSpec{},
	}

	// Extract host and path rules if specified.

	if rules, ok := spec["rules"].([]interface{}); ok {

		for _, rule := range rules {

			if ruleMap, ok := rule.(map[string]interface{}); ok {

				if host, ok := ruleMap["host"].(string); ok {

					ingressRule := networkingv1.IngressRule{

						Host: host,
					}

					ingress.Spec.Rules = append(ingress.Spec.Rules, ingressRule)

				}

			}

		}

	}

	createdIngress, err := k.clientset.NetworkingV1().Ingresses(req.Namespace).Create(ctx, ingress, metav1.CreateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to create ingress: %w", err)

	}

	return k.convertIngressToResourceResponse(createdIngress), nil

}

// Get resource methods.

// GetDeploymentResource gets a deployment as a ResourceResponse.

func (k *KubernetesProvider) GetDeploymentResource(ctx context.Context, namespace, name string) (*ResourceResponse, error) {

	deployment, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get deployment: %w", err)

	}

	return k.convertDeploymentToResourceResponse(deployment), nil

}

// GetService gets a service as a ResourceResponse.

func (k *KubernetesProvider) GetService(ctx context.Context, namespace, name string) (*ResourceResponse, error) {

	service, err := k.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get service: %w", err)

	}

	return k.convertServiceToResourceResponse(service), nil

}

// GetConfigMap gets a configmap as a ResourceResponse.

func (k *KubernetesProvider) GetConfigMap(ctx context.Context, namespace, name string) (*ResourceResponse, error) {

	configMap, err := k.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get configmap: %w", err)

	}

	return k.convertConfigMapToResourceResponse(configMap), nil

}

// GetSecret gets a secret as a ResourceResponse.

func (k *KubernetesProvider) GetSecret(ctx context.Context, namespace, name string) (*ResourceResponse, error) {

	secret, err := k.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get secret: %w", err)

	}

	return k.convertSecretToResourceResponse(secret), nil

}

// GetPersistentVolumeClaim gets a PVC as a ResourceResponse.

func (k *KubernetesProvider) GetPersistentVolumeClaim(ctx context.Context, namespace, name string) (*ResourceResponse, error) {

	pvc, err := k.clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get persistentvolumeclaim: %w", err)

	}

	return k.convertPVCToResourceResponse(pvc), nil

}

// Update resource methods.

// UpdateDeploymentResource updates a deployment resource.

func (k *KubernetesProvider) UpdateDeploymentResource(ctx context.Context, namespace, name string, req *UpdateResourceRequest) (*ResourceResponse, error) {

	deployment, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get deployment for update: %w", err)

	}

	// Update labels and annotations.

	if req.Labels != nil {

		if deployment.Labels == nil {

			deployment.Labels = make(map[string]string)

		}

		for key, value := range req.Labels {

			deployment.Labels[key] = value

		}

	}

	if req.Annotations != nil {

		if deployment.Annotations == nil {

			deployment.Annotations = make(map[string]string)

		}

		for key, value := range req.Annotations {

			deployment.Annotations[key] = value

		}

	}

	updatedDeployment, err := k.clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to update deployment: %w", err)

	}

	return k.convertDeploymentToResourceResponse(updatedDeployment), nil

}

// UpdateService updates a service resource.

func (k *KubernetesProvider) UpdateService(ctx context.Context, namespace, name string, req *UpdateResourceRequest) (*ResourceResponse, error) {

	service, err := k.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get service for update: %w", err)

	}

	// Update labels and annotations.

	if req.Labels != nil {

		if service.Labels == nil {

			service.Labels = make(map[string]string)

		}

		for key, value := range req.Labels {

			service.Labels[key] = value

		}

	}

	if req.Annotations != nil {

		if service.Annotations == nil {

			service.Annotations = make(map[string]string)

		}

		for key, value := range req.Annotations {

			service.Annotations[key] = value

		}

	}

	updatedService, err := k.clientset.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to update service: %w", err)

	}

	return k.convertServiceToResourceResponse(updatedService), nil

}

// UpdateConfigMap updates a configmap resource.

func (k *KubernetesProvider) UpdateConfigMap(ctx context.Context, namespace, name string, req *UpdateResourceRequest) (*ResourceResponse, error) {

	configMap, err := k.clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get configmap for update: %w", err)

	}

	// Update labels and annotations.

	if req.Labels != nil {

		if configMap.Labels == nil {

			configMap.Labels = make(map[string]string)

		}

		for key, value := range req.Labels {

			configMap.Labels[key] = value

		}

	}

	if req.Annotations != nil {

		if configMap.Annotations == nil {

			configMap.Annotations = make(map[string]string)

		}

		for key, value := range req.Annotations {

			configMap.Annotations[key] = value

		}

	}

	updatedConfigMap, err := k.clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to update configmap: %w", err)

	}

	return k.convertConfigMapToResourceResponse(updatedConfigMap), nil

}

// UpdateSecret updates a secret resource.

func (k *KubernetesProvider) UpdateSecret(ctx context.Context, namespace, name string, req *UpdateResourceRequest) (*ResourceResponse, error) {

	secret, err := k.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to get secret for update: %w", err)

	}

	// Update labels and annotations.

	if req.Labels != nil {

		if secret.Labels == nil {

			secret.Labels = make(map[string]string)

		}

		for key, value := range req.Labels {

			secret.Labels[key] = value

		}

	}

	if req.Annotations != nil {

		if secret.Annotations == nil {

			secret.Annotations = make(map[string]string)

		}

		for key, value := range req.Annotations {

			secret.Annotations[key] = value

		}

	}

	updatedSecret, err := k.clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})

	if err != nil {

		return nil, fmt.Errorf("failed to update secret: %w", err)

	}

	return k.convertSecretToResourceResponse(updatedSecret), nil

}

// Conversion helper methods.

func (k *KubernetesProvider) convertDeploymentToResourceResponse(deployment *appsv1.Deployment) *ResourceResponse {

	var status string

	var health HealthStatus

	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {

		status = "ready"

		health = HealthStatus{Status: HealthStatusHealthy}

	} else if deployment.Status.ReadyReplicas > 0 {

		status = "partially_ready"

		health = HealthStatus{Status: HealthStatusHealthy}

	} else {

		status = "not_ready"

		health = HealthStatus{Status: HealthStatusUnhealthy}

	}

	return &ResourceResponse{

		ID: fmt.Sprintf("%s/deployment/%s", deployment.Namespace, deployment.Name),

		Name: deployment.Name,

		Type: "deployment",

		Namespace: deployment.Namespace,

		Status: status,

		Health: health.Status,

		Labels: deployment.Labels,

		Annotations: deployment.Annotations,

		CreatedAt: deployment.CreationTimestamp.Time,

		UpdatedAt: time.Now(),
	}

}

func (k *KubernetesProvider) convertServiceToResourceResponse(service *corev1.Service) *ResourceResponse {

	status := "active"

	health := HealthStatus{Status: HealthStatusHealthy}

	if service.Spec.ClusterIP == "" || service.Spec.ClusterIP == "None" {

		health = HealthStatus{Status: HealthStatusUnknown}

	}

	return &ResourceResponse{

		ID: fmt.Sprintf("%s/service/%s", service.Namespace, service.Name),

		Name: service.Name,

		Type: "service",

		Namespace: service.Namespace,

		Status: status,

		Health: health.Status,

		Labels: service.Labels,

		Annotations: service.Annotations,

		CreatedAt: service.CreationTimestamp.Time,

		UpdatedAt: time.Now(),
	}

}

func (k *KubernetesProvider) convertConfigMapToResourceResponse(configMap *corev1.ConfigMap) *ResourceResponse {

	return &ResourceResponse{

		ID: fmt.Sprintf("%s/configmap/%s", configMap.Namespace, configMap.Name),

		Name: configMap.Name,

		Type: "configmap",

		Namespace: configMap.Namespace,

		Status: "active",

		Health: HealthStatusHealthy,

		Labels: configMap.Labels,

		Annotations: configMap.Annotations,

		CreatedAt: configMap.CreationTimestamp.Time,

		UpdatedAt: time.Now(),
	}

}

func (k *KubernetesProvider) convertSecretToResourceResponse(secret *corev1.Secret) *ResourceResponse {

	return &ResourceResponse{

		ID: fmt.Sprintf("%s/secret/%s", secret.Namespace, secret.Name),

		Name: secret.Name,

		Type: "secret",

		Namespace: secret.Namespace,

		Status: "active",

		Health: HealthStatusHealthy,

		Labels: secret.Labels,

		Annotations: secret.Annotations,

		CreatedAt: secret.CreationTimestamp.Time,

		UpdatedAt: time.Now(),
	}

}

func (k *KubernetesProvider) convertPVCToResourceResponse(pvc *corev1.PersistentVolumeClaim) *ResourceResponse {

	status := strings.ToLower(string(pvc.Status.Phase))

	var health HealthStatus

	switch pvc.Status.Phase {

	case corev1.ClaimBound:

		health = HealthStatus{Status: HealthStatusHealthy}

	case corev1.ClaimPending:

		health = HealthStatus{Status: HealthStatusUnknown}

	default:

		health = HealthStatus{Status: HealthStatusUnhealthy}

	}

	return &ResourceResponse{

		ID: fmt.Sprintf("%s/persistentvolumeclaim/%s", pvc.Namespace, pvc.Name),

		Name: pvc.Name,

		Type: "persistentvolumeclaim",

		Namespace: pvc.Namespace,

		Status: status,

		Health: health.Status,

		Labels: pvc.Labels,

		Annotations: pvc.Annotations,

		CreatedAt: pvc.CreationTimestamp.Time,

		UpdatedAt: time.Now(),
	}

}

func (k *KubernetesProvider) convertIngressToResourceResponse(ingress *networkingv1.Ingress) *ResourceResponse {

	status := "active"

	health := HealthStatus{Status: HealthStatusHealthy}

	// Check if ingress has load balancer status.

	if len(ingress.Status.LoadBalancer.Ingress) == 0 {

		health = HealthStatus{Status: HealthStatusUnknown}

	}

	return &ResourceResponse{

		ID: fmt.Sprintf("%s/ingress/%s", ingress.Namespace, ingress.Name),

		Name: ingress.Name,

		Type: "ingress",

		Namespace: ingress.Namespace,

		Status: status,

		Health: health.Status,

		Labels: ingress.Labels,

		Annotations: ingress.Annotations,

		CreatedAt: ingress.CreationTimestamp.Time,

		UpdatedAt: time.Now(),
	}

}

func (k *KubernetesProvider) convertDeploymentToResponse(deployment *appsv1.Deployment) *DeploymentResponse {

	var status string

	var phase string

	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {

		status = "ready"

		phase = "stable"

	} else if deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {

		status = "updating"

		phase = "rolling_update"

	} else {

		status = "pending"

		phase = "creating"

	}

	return &DeploymentResponse{

		ID: fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name),

		Name: deployment.Name,

		Status: status,

		Phase: phase,

		TemplateType: "kubernetes",

		Namespace: deployment.Namespace,

		Labels: deployment.Labels,

		Annotations: deployment.Annotations,

		CreatedAt: deployment.CreationTimestamp.Time,

		UpdatedAt: time.Now(),
	}

}

// Placeholder implementations for remaining methods that are called but not yet implemented.

func (k *KubernetesProvider) listResourcesByType(ctx context.Context, resourceType string, filter *ResourceFilter) ([]*ResourceResponse, error) {

	// Implementation would list resources of specific type.

	return []*ResourceResponse{}, nil

}

func (k *KubernetesProvider) applyResourceFilters(resources []*ResourceResponse, filter *ResourceFilter) []*ResourceResponse {

	// Implementation would apply filters.

	return resources

}

func (k *KubernetesProvider) deployKubernetesTemplate(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {

	// Parse YAML template and create resources.

	return nil, fmt.Errorf("kubernetes template deployment not yet implemented")

}

func (k *KubernetesProvider) deployHelmTemplate(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error) {

	// Deploy using Helm.

	return nil, fmt.Errorf("helm deployment not yet implemented")

}

func (k *KubernetesProvider) updateKubernetesDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*DeploymentResponse, error) {

	return nil, fmt.Errorf("kubernetes deployment update not yet implemented")

}

func (k *KubernetesProvider) updateHelmDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*DeploymentResponse, error) {

	return nil, fmt.Errorf("helm deployment update not yet implemented")

}

func (k *KubernetesProvider) scaleDeployment(ctx context.Context, namespace, name string, req *ScaleRequest) error {

	// Implementation would scale the deployment.

	return fmt.Errorf("deployment scaling not yet implemented")

}

func (k *KubernetesProvider) scaleStatefulSet(ctx context.Context, namespace, name string, req *ScaleRequest) error {

	// Implementation would scale the statefulset.

	return fmt.Errorf("statefulset scaling not yet implemented")

}

func (k *KubernetesProvider) getDeploymentHealth(ctx context.Context, namespace, name string) (*HealthStatus, error) {

	deployment, err := k.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return &HealthStatus{

			Status: HealthStatusUnhealthy,

			Message: fmt.Sprintf("Failed to get deployment: %v", err),

			LastUpdated: time.Now(),
		}, nil

	}

	status := HealthStatusHealthy

	message := "All replicas are ready"

	if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {

		status = HealthStatusUnhealthy

		message = fmt.Sprintf("Only %d of %d replicas are ready", deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)

	}

	return &HealthStatus{

		Status: status,

		Message: message,

		LastUpdated: time.Now(),
	}, nil

}

func (k *KubernetesProvider) getServiceHealth(ctx context.Context, namespace, name string) (*HealthStatus, error) {

	service, err := k.clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return &HealthStatus{

			Status: HealthStatusUnhealthy,

			Message: fmt.Sprintf("Failed to get service: %v", err),

			LastUpdated: time.Now(),
		}, nil

	}

	status := HealthStatusHealthy

	message := "Service is active"

	if service.Spec.ClusterIP == "" || service.Spec.ClusterIP == "None" {

		status = HealthStatusUnknown

		message = "Service has no cluster IP"

	}

	return &HealthStatus{

		Status: status,

		Message: message,

		LastUpdated: time.Now(),
	}, nil

}

func (k *KubernetesProvider) getPodHealth(ctx context.Context, namespace, name string) (*HealthStatus, error) {

	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return &HealthStatus{

			Status: HealthStatusUnhealthy,

			Message: fmt.Sprintf("Failed to get pod: %v", err),

			LastUpdated: time.Now(),
		}, nil

	}

	status := HealthStatusUnknown

	message := string(pod.Status.Phase)

	switch pod.Status.Phase {

	case corev1.PodRunning:

		status = HealthStatusHealthy

		message = "Pod is running"

	case corev1.PodSucceeded:

		status = HealthStatusHealthy

		message = "Pod completed successfully"

	case corev1.PodFailed:

		status = HealthStatusUnhealthy

		message = "Pod failed"

	case corev1.PodPending:

		status = HealthStatusUnknown

		message = "Pod is pending"

	}

	return &HealthStatus{

		Status: status,

		Message: message,

		LastUpdated: time.Now(),
	}, nil

}

func (k *KubernetesProvider) watchEvents(ctx context.Context) {

	// Implementation would watch for Kubernetes events.

	// This is a placeholder.

}

// Network service implementations.

func (k *KubernetesProvider) createKubernetesService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {

	return nil, fmt.Errorf("kubernetes service creation not yet implemented")

}

func (k *KubernetesProvider) createKubernetesIngress(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {

	return nil, fmt.Errorf("kubernetes ingress creation not yet implemented")

}

func (k *KubernetesProvider) createNetworkPolicy(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error) {

	return nil, fmt.Errorf("network policy creation not yet implemented")

}

func (k *KubernetesProvider) getKubernetesService(ctx context.Context, namespace, name string) (*NetworkServiceResponse, error) {

	return nil, fmt.Errorf("kubernetes service get not yet implemented")

}

func (k *KubernetesProvider) getKubernetesIngress(ctx context.Context, namespace, name string) (*NetworkServiceResponse, error) {

	return nil, fmt.Errorf("kubernetes ingress get not yet implemented")

}

func (k *KubernetesProvider) getKubernetesNetworkPolicy(ctx context.Context, namespace, name string) (*NetworkServiceResponse, error) {

	return nil, fmt.Errorf("network policy get not yet implemented")

}

func (k *KubernetesProvider) listNetworkServicesByType(ctx context.Context, serviceType, namespace string, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error) {

	return []*NetworkServiceResponse{}, nil

}

// Storage resource implementations.

func (k *KubernetesProvider) createPVC(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {

	return nil, fmt.Errorf("PVC creation not yet implemented")

}

func (k *KubernetesProvider) createStorageClass(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error) {

	return nil, fmt.Errorf("storage class creation not yet implemented")

}

func (k *KubernetesProvider) getPVC(ctx context.Context, namespace, name string) (*StorageResourceResponse, error) {

	return nil, fmt.Errorf("PVC get not yet implemented")

}

func (k *KubernetesProvider) getStorageClass(ctx context.Context, name string) (*StorageResourceResponse, error) {

	return nil, fmt.Errorf("storage class get not yet implemented")

}

func (k *KubernetesProvider) listStorageResourcesByType(ctx context.Context, resourceType string, filter *StorageResourceFilter) ([]*StorageResourceResponse, error) {

	return []*StorageResourceResponse{}, nil

}
