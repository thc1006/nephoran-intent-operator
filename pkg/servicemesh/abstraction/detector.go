// Package abstraction provides service mesh detection and factory capabilities
package abstraction

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ServiceMeshDetector detects installed service mesh providers
type ServiceMeshDetector struct {
	kubeClient kubernetes.Interface
	config     *rest.Config
	logger     logr.Logger
}

// NewServiceMeshDetector creates a new service mesh detector
func NewServiceMeshDetector(kubeClient kubernetes.Interface, config *rest.Config) *ServiceMeshDetector {
	return &ServiceMeshDetector{
		kubeClient: kubeClient,
		config:     config,
		logger:     log.Log.WithName("service-mesh-detector"),
	}
}

// DetectServiceMesh detects the installed service mesh provider
func (d *ServiceMeshDetector) DetectServiceMesh(ctx context.Context) (ServiceMeshProvider, error) {
	// Check for Istio
	if provider, err := d.detectIstio(ctx); err == nil && provider != ProviderNone {
		d.logger.Info("Detected service mesh provider", "provider", provider)
		return provider, nil
	}

	// Check for Linkerd
	if provider, err := d.detectLinkerd(ctx); err == nil && provider != ProviderNone {
		d.logger.Info("Detected service mesh provider", "provider", provider)
		return provider, nil
	}

	// Check for Consul
	if provider, err := d.detectConsul(ctx); err == nil && provider != ProviderNone {
		d.logger.Info("Detected service mesh provider", "provider", provider)
		return provider, nil
	}

	d.logger.Info("No service mesh provider detected")
	return ProviderNone, nil
}

// detectIstio checks for Istio installation
func (d *ServiceMeshDetector) detectIstio(ctx context.Context) (ServiceMeshProvider, error) {
	// Check for istio-system namespace
	namespaces, err := d.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return ProviderNone, fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		if ns.Name == "istio-system" {
			// Check for istiod deployment
			deployments, err := d.kubeClient.AppsV1().Deployments("istio-system").List(ctx, metav1.ListOptions{})
			if err != nil {
				return ProviderNone, fmt.Errorf("failed to list deployments: %w", err)
			}

			for _, deploy := range deployments.Items {
				if strings.HasPrefix(deploy.Name, "istiod") {
					// Check if deployment is ready
					if deploy.Status.ReadyReplicas > 0 {
						return ProviderIstio, nil
					}
				}
			}
		}
	}

	return ProviderNone, nil
}

// detectLinkerd checks for Linkerd installation
func (d *ServiceMeshDetector) detectLinkerd(ctx context.Context) (ServiceMeshProvider, error) {
	// Check for linkerd namespace
	namespaces, err := d.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return ProviderNone, fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		if ns.Name == "linkerd" {
			// Check for linkerd-controller deployment
			deployments, err := d.kubeClient.AppsV1().Deployments("linkerd").List(ctx, metav1.ListOptions{})
			if err != nil {
				return ProviderNone, fmt.Errorf("failed to list deployments: %w", err)
			}

			for _, deploy := range deployments.Items {
				if deploy.Name == "linkerd-controller" || deploy.Name == "linkerd-destination" {
					if deploy.Status.ReadyReplicas > 0 {
						return ProviderLinkerd, nil
					}
				}
			}
		}
	}

	return ProviderNone, nil
}

// detectConsul checks for Consul installation
func (d *ServiceMeshDetector) detectConsul(ctx context.Context) (ServiceMeshProvider, error) {
	// Check for consul namespace
	namespaces, err := d.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return ProviderNone, fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		if ns.Name == "consul" {
			// Check for consul-server statefulset
			statefulsets, err := d.kubeClient.AppsV1().StatefulSets("consul").List(ctx, metav1.ListOptions{})
			if err != nil {
				return ProviderNone, fmt.Errorf("failed to list statefulsets: %w", err)
			}

			for _, sts := range statefulsets.Items {
				if strings.Contains(sts.Name, "consul-server") {
					if sts.Status.ReadyReplicas > 0 {
						return ProviderConsul, nil
					}
				}
			}
		}
	}

	return ProviderNone, nil
}

// GetServiceMeshVersion gets the version of the detected service mesh
func (d *ServiceMeshDetector) GetServiceMeshVersion(ctx context.Context, provider ServiceMeshProvider) (string, error) {
	switch provider {
	case ProviderIstio:
		return d.getIstioVersion(ctx)
	case ProviderLinkerd:
		return d.getLinkerdVersion(ctx)
	case ProviderConsul:
		return d.getConsulVersion(ctx)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// getIstioVersion gets the Istio version
func (d *ServiceMeshDetector) getIstioVersion(ctx context.Context) (string, error) {
	// Check istiod deployment for version label
	deployments, err := d.kubeClient.AppsV1().Deployments("istio-system").List(ctx, metav1.ListOptions{
		LabelSelector: "app=istiod",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get istiod deployment: %w", err)
	}

	if len(deployments.Items) > 0 {
		deploy := deployments.Items[0]
		if version, ok := deploy.Labels["version"]; ok {
			return version, nil
		}
		// Check container image tag
		for _, container := range deploy.Spec.Template.Spec.Containers {
			if container.Name == "discovery" {
				parts := strings.Split(container.Image, ":")
				if len(parts) > 1 {
					return parts[1], nil
				}
			}
		}
	}

	return "unknown", nil
}

// getLinkerdVersion gets the Linkerd version
func (d *ServiceMeshDetector) getLinkerdVersion(ctx context.Context) (string, error) {
	// Check linkerd-controller deployment for version annotation
	deployments, err := d.kubeClient.AppsV1().Deployments("linkerd").List(ctx, metav1.ListOptions{
		LabelSelector: "linkerd.io/control-plane-component=controller",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get linkerd deployment: %w", err)
	}

	if len(deployments.Items) > 0 {
		deploy := deployments.Items[0]
		if version, ok := deploy.Annotations["linkerd.io/control-plane-version"]; ok {
			return version, nil
		}
		// Check container image tag
		for _, container := range deploy.Spec.Template.Spec.Containers {
			parts := strings.Split(container.Image, ":")
			if len(parts) > 1 {
				return parts[1], nil
			}
		}
	}

	return "unknown", nil
}

// getConsulVersion gets the Consul version
func (d *ServiceMeshDetector) getConsulVersion(ctx context.Context) (string, error) {
	// Check consul-server statefulset for version
	statefulsets, err := d.kubeClient.AppsV1().StatefulSets("consul").List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get consul statefulset: %w", err)
	}

	for _, sts := range statefulsets.Items {
		if strings.Contains(sts.Name, "consul-server") {
			// Check container image tag
			for _, container := range sts.Spec.Template.Spec.Containers {
				if container.Name == "consul" {
					parts := strings.Split(container.Image, ":")
					if len(parts) > 1 {
						return parts[1], nil
					}
				}
			}
		}
	}

	return "unknown", nil
}

// DetectorResult contains service mesh detection results
type DetectorResult struct {
	Provider     ServiceMeshProvider `json:"provider"`
	Version      string              `json:"version"`
	Namespace    string              `json:"namespace"`
	ControlPlane ControlPlaneInfo    `json:"controlPlane"`
	DataPlane    DataPlaneInfo       `json:"dataPlane"`
	Features     []string            `json:"features"`
}

// ControlPlaneInfo contains control plane information
type ControlPlaneInfo struct {
	Ready       bool              `json:"ready"`
	Components  []ComponentStatus `json:"components"`
	Endpoints   []string          `json:"endpoints"`
	Annotations map[string]string `json:"annotations"`
}

// DataPlaneInfo contains data plane information
type DataPlaneInfo struct {
	ProxyCount     int    `json:"proxyCount"`
	ProxyVersion   string `json:"proxyVersion"`
	InjectionMode  string `json:"injectionMode"`
	NamespaceCount int    `json:"namespaceCount"`
}

// ComponentStatus represents a control plane component status
type ComponentStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Ready     bool   `json:"ready"`
	Version   string `json:"version,omitempty"`
}

// GetDetectionResult gets comprehensive detection results
func (d *ServiceMeshDetector) GetDetectionResult(ctx context.Context) (*DetectorResult, error) {
	provider, err := d.DetectServiceMesh(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect service mesh: %w", err)
	}

	if provider == ProviderNone {
		return &DetectorResult{
			Provider: ProviderNone,
		}, nil
	}

	version, err := d.GetServiceMeshVersion(ctx, provider)
	if err != nil {
		d.logger.Error(err, "Failed to get service mesh version")
		version = "unknown"
	}

	result := &DetectorResult{
		Provider: provider,
		Version:  version,
	}

	// Get detailed information based on provider
	switch provider {
	case ProviderIstio:
		result.Namespace = "istio-system"
		result.Features = []string{"mtls", "traffic-management", "observability", "multi-cluster"}
		// Additional Istio-specific detection
		d.enrichIstioInfo(ctx, result)
	case ProviderLinkerd:
		result.Namespace = "linkerd"
		result.Features = []string{"mtls", "traffic-management", "observability"}
		// Additional Linkerd-specific detection
		d.enrichLinkerdInfo(ctx, result)
	case ProviderConsul:
		result.Namespace = "consul"
		result.Features = []string{"mtls", "service-discovery", "kv-store"}
		// Additional Consul-specific detection
		d.enrichConsulInfo(ctx, result)
	}

	return result, nil
}

// enrichIstioInfo adds Istio-specific information
func (d *ServiceMeshDetector) enrichIstioInfo(ctx context.Context, result *DetectorResult) {
	// Count proxies
	pods, err := d.kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "security.istio.io/tlsMode=istio",
	})
	if err == nil {
		result.DataPlane.ProxyCount = len(pods.Items)
	}

	// Get injection namespaces
	namespaces, err := d.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "istio-injection=enabled",
	})
	if err == nil {
		result.DataPlane.NamespaceCount = len(namespaces.Items)
		result.DataPlane.InjectionMode = "automatic"
	}

	// Check control plane components
	result.ControlPlane.Components = []ComponentStatus{
		{Name: "istiod", Namespace: "istio-system", Type: "deployment"},
		{Name: "istio-ingressgateway", Namespace: "istio-system", Type: "deployment"},
		{Name: "istio-egressgateway", Namespace: "istio-system", Type: "deployment"},
	}

	for i := range result.ControlPlane.Components {
		component := &result.ControlPlane.Components[i]
		deploy, err := d.kubeClient.AppsV1().Deployments(component.Namespace).Get(ctx, component.Name, metav1.GetOptions{})
		if err == nil && deploy.Status.ReadyReplicas > 0 {
			component.Ready = true
		}
	}
}

// enrichLinkerdInfo adds Linkerd-specific information
func (d *ServiceMeshDetector) enrichLinkerdInfo(ctx context.Context, result *DetectorResult) {
	// Count proxies
	pods, err := d.kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		proxyCount := 0
		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				if container.Name == "linkerd-proxy" {
					proxyCount++
					break
				}
			}
		}
		result.DataPlane.ProxyCount = proxyCount
	}

	// Get injection namespaces
	namespaces, err := d.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "linkerd.io/inject=enabled",
	})
	if err == nil {
		result.DataPlane.NamespaceCount = len(namespaces.Items)
		result.DataPlane.InjectionMode = "automatic"
	}

	// Check control plane components
	result.ControlPlane.Components = []ComponentStatus{
		{Name: "linkerd-controller", Namespace: "linkerd", Type: "deployment"},
		{Name: "linkerd-destination", Namespace: "linkerd", Type: "deployment"},
		{Name: "linkerd-identity", Namespace: "linkerd", Type: "deployment"},
		{Name: "linkerd-proxy-injector", Namespace: "linkerd", Type: "deployment"},
	}

	for i := range result.ControlPlane.Components {
		component := &result.ControlPlane.Components[i]
		deploy, err := d.kubeClient.AppsV1().Deployments(component.Namespace).Get(ctx, component.Name, metav1.GetOptions{})
		if err == nil && deploy.Status.ReadyReplicas > 0 {
			component.Ready = true
		}
	}
}

// enrichConsulInfo adds Consul-specific information
func (d *ServiceMeshDetector) enrichConsulInfo(ctx context.Context, result *DetectorResult) {
	// Count connect proxies
	pods, err := d.kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		proxyCount := 0
		for _, pod := range pods.Items {
			if _, ok := pod.Annotations["consul.hashicorp.com/connect-inject"]; ok {
				proxyCount++
			}
		}
		result.DataPlane.ProxyCount = proxyCount
	}

	// Check control plane components
	result.ControlPlane.Components = []ComponentStatus{
		{Name: "consul-server", Namespace: "consul", Type: "statefulset"},
		{Name: "consul-connect-injector", Namespace: "consul", Type: "deployment"},
	}

	// Check statefulset
	sts, err := d.kubeClient.AppsV1().StatefulSets("consul").Get(ctx, "consul-server", metav1.GetOptions{})
	if err == nil && sts.Status.ReadyReplicas > 0 {
		result.ControlPlane.Components[0].Ready = true
	}

	// Check deployment
	deploy, err := d.kubeClient.AppsV1().Deployments("consul").Get(ctx, "consul-connect-injector", metav1.GetOptions{})
	if err == nil && deploy.Status.ReadyReplicas > 0 {
		result.ControlPlane.Components[1].Ready = true
	}
}
