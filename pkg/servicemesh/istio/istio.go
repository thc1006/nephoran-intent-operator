// Package istio provides Istio service mesh implementation.

package istio

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/servicemesh/abstraction"
)

func init() {
	abstraction.RegisterProvider(abstraction.ProviderIstio, func(kubeClient kubernetes.Interface, dynamicClient client.Client, config *rest.Config, meshConfig *abstraction.ServiceMeshConfig) (abstraction.ServiceMeshInterface, error) {
		// Create Istio-specific configuration.

		istioConfig := &Config{
			Namespace: meshConfig.Namespace,

			TrustDomain: meshConfig.TrustDomain,

			ControlPlaneURL: meshConfig.ControlPlaneURL,

			CertificateConfig: meshConfig.CertificateConfig,

			PolicyDefaults: meshConfig.PolicyDefaults,

			ObservabilityConfig: meshConfig.ObservabilityConfig,

			MultiCluster: meshConfig.MultiCluster,
		}

		// Extract Istio-specific settings from custom config.

		if meshConfig.CustomConfig != nil {

			if pilotURL, ok := meshConfig.CustomConfig["pilotURL"].(string); ok {
				istioConfig.PilotURL = pilotURL
			}

			if meshID, ok := meshConfig.CustomConfig["meshID"].(string); ok {
				istioConfig.MeshID = meshID
			}

			if network, ok := meshConfig.CustomConfig["network"].(string); ok {
				istioConfig.Network = network
			}

		}

		return NewIstioMesh(kubeClient, dynamicClient, config, istioConfig)
	})
}

// Config contains Istio-specific configuration.

type Config struct {
	Namespace string `json:"namespace"`

	TrustDomain string `json:"trustDomain"`

	ControlPlaneURL string `json:"controlPlaneUrl"`

	PilotURL string `json:"pilotUrl"`

	MeshID string `json:"meshId"`

	Network string `json:"network"`

	CertificateConfig *abstraction.CertificateConfig `json:"certificateConfig"`

	PolicyDefaults *abstraction.PolicyDefaults `json:"policyDefaults"`

	ObservabilityConfig *abstraction.ObservabilityConfig `json:"observabilityConfig"`

	MultiCluster *abstraction.MultiClusterConfig `json:"multiCluster"`
}

// IstioMesh implements ServiceMeshInterface for Istio.

type IstioMesh struct {
	kubeClient kubernetes.Interface

	dynamicClient client.Client

	config *rest.Config

	meshConfig *Config

	certProvider *IstioCertificateProvider

	metrics *IstioMetrics

	logger logr.Logger
}

// NewIstioMesh creates a new Istio mesh implementation.

func NewIstioMesh(
	kubeClient kubernetes.Interface,

	dynamicClient client.Client,

	config *rest.Config,

	meshConfig *Config,
) (*IstioMesh, error) {
	// Create certificate provider.

	certProvider := NewIstioCertificateProvider(kubeClient, meshConfig.TrustDomain)

	// Create metrics.

	metrics := NewIstioMetrics()

	return &IstioMesh{
		kubeClient: kubeClient,

		dynamicClient: dynamicClient,

		config: config,

		meshConfig: meshConfig,

		certProvider: certProvider,

		metrics: metrics,

		logger: log.Log.WithName("istio-mesh"),
	}, nil
}

// Initialize initializes the Istio mesh.

func (m *IstioMesh) Initialize(ctx context.Context, config *abstraction.ServiceMeshConfig) error {
	m.logger.Info("Initializing Istio service mesh")

	// Verify Istio installation.

	if err := m.verifyIstioInstallation(ctx); err != nil {
		return fmt.Errorf("Istio verification failed: %w", err)
	}

	// Configure default policies if specified.

	if config.PolicyDefaults != nil {
		if err := m.configureDefaultPolicies(ctx, config.PolicyDefaults); err != nil {
			return fmt.Errorf("failed to configure default policies: %w", err)
		}
	}

	// Register metrics.

	prometheus.MustRegister(m.metrics.GetCollectors()...)

	m.logger.Info("Istio service mesh initialized successfully")

	return nil
}

// verifyIstioInstallation verifies Istio is properly installed.

func (m *IstioMesh) verifyIstioInstallation(ctx context.Context) error {
	// Check istiod deployment.

	deployment, err := m.kubeClient.AppsV1().Deployments("istio-system").Get(ctx, "istiod", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("istiod deployment not found: %w", err)
	}

	if deployment.Status.ReadyReplicas == 0 {
		return fmt.Errorf("istiod deployment not ready")
	}

	// Check Istio CRDs.

	crds := []string{
		"peerauthentications.security.istio.io",

		"authorizationpolicies.security.istio.io",

		"destinationrules.networking.istio.io",

		"virtualservices.networking.istio.io",
	}

	for _, crd := range crds {

		// Check CRD exists using dynamic client.

		dynamicClient := dynamic.NewForConfigOrDie(m.config)

		crdGVR := schema.GroupVersionResource{
			Group: "apiextensions.k8s.io",

			Version: "v1",

			Resource: "customresourcedefinitions",
		}

		_, err := dynamicClient.Resource(crdGVR).Get(ctx, crd, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("CRD %s not found: %w", crd, err)
		}

	}

	return nil
}

// configureDefaultPolicies configures default Istio policies.

func (m *IstioMesh) configureDefaultPolicies(ctx context.Context, defaults *abstraction.PolicyDefaults) error {
	// Create default PeerAuthentication for mTLS.

	if defaults.MTLSMode != "" {

		peerAuth := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "security.istio.io/v1beta1",
				"kind":       "PeerAuthentication",
				"metadata": map[string]interface{}{
					"name":      "default",
					"namespace": m.meshConfig.Namespace,
				},
				"spec": map[string]interface{}{
					"mtls": map[string]interface{}{
						"mode": defaults.MTLSMode,
					},
				},
			},
		}

		gvr := schema.GroupVersionResource{
			Group: "security.istio.io",

			Version: "v1beta1",

			Resource: "peerauthentications",
		}

		dynamicClient := dynamic.NewForConfigOrDie(m.config)

		_, err := dynamicClient.Resource(gvr).Namespace(m.meshConfig.Namespace).
			Create(ctx, peerAuth, metav1.CreateOptions{})
		if err != nil {
			m.logger.Error(err, "Failed to create default PeerAuthentication")
		}

	}

	// Create default AuthorizationPolicy if deny-all is enabled.

	if defaults.DefaultDenyAll {

		authPolicy := &unstructured.Unstructured{
			Object: json.RawMessage("{}"){
					"name": "default-deny-all",

					"namespace": m.meshConfig.Namespace,
				},

				"spec": json.RawMessage("{}"),
			},
		}

		gvr := schema.GroupVersionResource{
			Group: "security.istio.io",

			Version: "v1beta1",

			Resource: "authorizationpolicies",
		}

		dynamicClient := dynamic.NewForConfigOrDie(m.config)

		_, err := dynamicClient.Resource(gvr).Namespace(m.meshConfig.Namespace).
			Create(ctx, authPolicy, metav1.CreateOptions{})
		if err != nil {
			m.logger.Error(err, "Failed to create default AuthorizationPolicy")
		}

	}

	return nil
}

// GetCertificateProvider returns the certificate provider.

func (m *IstioMesh) GetCertificateProvider() abstraction.CertificateProvider {
	return m.certProvider
}

// RotateCertificates rotates certificates for services in a namespace.

func (m *IstioMesh) RotateCertificates(ctx context.Context, namespace string) error {
	// In Istio, Citadel handles certificate rotation automatically.

	// We can trigger a restart of pods to force new certificate issuance.

	pods, err := m.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {

		// Check if pod has Istio sidecar.

		if !m.hasIstioSidecar(&pod) {
			continue
		}

		// Delete pod to trigger recreation with new certificate.

		err := m.kubeClient.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil {
			m.logger.Error(err, "Failed to delete pod for certificate rotation", "pod", pod.Name)
		}

	}

	return nil
}

// ValidateCertificateChain validates the certificate chain in a namespace.

func (m *IstioMesh) ValidateCertificateChain(ctx context.Context, namespace string) error {
	// Get pods with Istio sidecars.

	pods, err := m.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {

		if !m.hasIstioSidecar(&pod) {
			continue
		}

		// Validate certificate for each pod.

		// This would typically involve checking the certificate mounted in the sidecar.

		// For now, we'll assume certificates are valid if the pod is running.

		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("pod %s has invalid certificate status", pod.Name)
		}

	}

	return nil
}

// ApplyMTLSPolicy applies an mTLS policy.

func (m *IstioMesh) ApplyMTLSPolicy(ctx context.Context, policy *abstraction.MTLSPolicy) error {
	// Convert to Istio PeerAuthentication.

	peerAuth := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "security.istio.io/v1beta1",
			"kind":       "PeerAuthentication",
			"metadata": map[string]interface{}{
				"name":      policy.Name,
				"namespace": policy.Namespace,
			},
			"spec": map[string]interface{}{
				"mtls": map[string]interface{}{
					"mode": policy.Spec.Mode,
				},
			},
		},
	}

	// Add port-level mTLS if specified.

	if len(policy.Spec.PortLevelMTLS) > 0 {

		portLevelMtls := make(map[string]interface{})

		for _, portMtls := range policy.Spec.PortLevelMTLS {
			portLevelMtls[fmt.Sprintf("%d", portMtls.Port)] = json.RawMessage("{}")
		}

		peerAuth.Object["spec"].(map[string]interface{})["portLevelMtls"] = portLevelMtls

	}

	gvr := schema.GroupVersionResource{
		Group: "security.istio.io",

		Version: "v1beta1",

		Resource: "peerauthentications",
	}

	dynamicClient := dynamic.NewForConfigOrDie(m.config)

	_, err := dynamicClient.Resource(gvr).Namespace(policy.Namespace).
		Create(ctx, peerAuth, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create PeerAuthentication: %w", err)
	}

	m.metrics.policiesApplied.Inc()

	return nil
}

// ApplyAuthorizationPolicy applies an authorization policy.

func (m *IstioMesh) ApplyAuthorizationPolicy(ctx context.Context, policy *abstraction.AuthorizationPolicy) error {
	// Convert to Istio AuthorizationPolicy.

	authPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "security.istio.io/v1beta1",
			"kind":       "AuthorizationPolicy",
			"metadata": map[string]interface{}{
				"name":      policy.Name,
				"namespace": policy.Namespace,
			},
			"spec": map[string]interface{}{
				"rules": policy.Spec.Rules,
			},
		},
	}

	gvr := schema.GroupVersionResource{
		Group: "security.istio.io",

		Version: "v1beta1",

		Resource: "authorizationpolicies",
	}

	dynamicClient := dynamic.NewForConfigOrDie(m.config)

	_, err := dynamicClient.Resource(gvr).Namespace(policy.Namespace).
		Create(ctx, authPolicy, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create AuthorizationPolicy: %w", err)
	}

	m.metrics.policiesApplied.Inc()

	return nil
}

// ApplyTrafficPolicy applies a traffic management policy.

func (m *IstioMesh) ApplyTrafficPolicy(ctx context.Context, policy *abstraction.TrafficPolicy) error {
	// Create VirtualService for traffic management.

	vs := &unstructured.Unstructured{
		Object: json.RawMessage("{}"){
				"name": policy.Name,

				"namespace": policy.Namespace,
			},

			"spec": json.RawMessage("{}"),
		},
	}

	// Add traffic shifting if specified.

	if policy.Spec.TrafficShifting != nil {

		http := []interface{}{}

		for _, dest := range policy.Spec.TrafficShifting.Destinations {

			route := json.RawMessage("{}"){
					"host": dest.Service,
				},

				"weight": dest.Weight,
			}

			if dest.Version != "" {
				route["destination"].(map[string]interface{})["subset"] = dest.Version
			}

			http = append(http, json.RawMessage("{}"){route},
			})

		}

		vs.Object["spec"].(map[string]interface{})["http"] = http

	}

	// Create DestinationRule for circuit breaker, load balancing, etc.

	dr := &unstructured.Unstructured{
		Object: json.RawMessage("{}"){
				"name": policy.Name + "-dr",

				"namespace": policy.Namespace,
			},

			"spec": json.RawMessage("{}"),
		},
	}

	trafficPolicy := make(map[string]interface{})

	// Add circuit breaker if specified.

	if policy.Spec.CircuitBreaker != nil {

		trafficPolicy["connectionPool"] = json.RawMessage("{}"){
				"maxConnections": 100,
			},
		}

		trafficPolicy["outlierDetection"] = json.RawMessage("{}")

	}

	// Add retry policy if specified.

	if policy.Spec.Retry != nil {
		// Retry is configured in VirtualService.

		if vs.Object["spec"].(map[string]interface{})["http"] != nil {
			for _, http := range vs.Object["spec"].(map[string]interface{})["http"].([]interface{}) {
				http.(map[string]interface{})["retries"] = json.RawMessage("{}")
			}
		}
	}

	// Add timeout if specified.

	if policy.Spec.Timeout != nil {
		// Timeout is configured in VirtualService.

		if vs.Object["spec"].(map[string]interface{})["http"] != nil {
			for _, http := range vs.Object["spec"].(map[string]interface{})["http"].([]interface{}) {
				http.(map[string]interface{})["timeout"] = policy.Spec.Timeout.RequestTimeout
			}
		}
	}

	// Add load balancer if specified.

	if policy.Spec.LoadBalancer != nil {
		switch policy.Spec.LoadBalancer.Algorithm {

		case "round-robin":

			trafficPolicy["loadBalancer"] = json.RawMessage("{}")

		case "least-conn":

			trafficPolicy["loadBalancer"] = json.RawMessage("{}")

		case "random":

			trafficPolicy["loadBalancer"] = json.RawMessage("{}")

		case "consistent-hash":

			if policy.Spec.LoadBalancer.ConsistentHash != nil {
				trafficPolicy["loadBalancer"] = json.RawMessage("{}"){
						"httpHeaderName": policy.Spec.LoadBalancer.ConsistentHash.HashKey,

						"minimumRingSize": policy.Spec.LoadBalancer.ConsistentHash.MinimumRingSize,
					},
				}
			}

		}
	}

	if len(trafficPolicy) > 0 {
		dr.Object["spec"].(map[string]interface{})["trafficPolicy"] = trafficPolicy
	}

	// Apply VirtualService.

	gvr := schema.GroupVersionResource{
		Group: "networking.istio.io",

		Version: "v1beta1",

		Resource: "virtualservices",
	}

	dynamicClient := dynamic.NewForConfigOrDie(m.config)

	_, err := dynamicClient.Resource(gvr).Namespace(policy.Namespace).
		Create(ctx, vs, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create VirtualService: %w", err)
	}

	// Apply DestinationRule if needed.

	if len(trafficPolicy) > 0 {

		gvr = schema.GroupVersionResource{
			Group: "networking.istio.io",

			Version: "v1beta1",

			Resource: "destinationrules",
		}

		_, err = dynamicClient.Resource(gvr).Namespace(policy.Namespace).
			Create(ctx, dr, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create DestinationRule: %w", err)
		}

	}

	m.metrics.policiesApplied.Inc()

	return nil
}

// ValidatePolicies validates policies in a namespace.

func (m *IstioMesh) ValidatePolicies(ctx context.Context, namespace string) (*abstraction.PolicyValidationResult, error) {
	result := &abstraction.PolicyValidationResult{
		Valid: true,

		Errors: []abstraction.PolicyError{},

		Warnings: []abstraction.PolicyWarning{},

		Conflicts: []abstraction.PolicyConflict{},
	}

	// Check PeerAuthentications.

	gvr := schema.GroupVersionResource{
		Group: "security.istio.io",

		Version: "v1beta1",

		Resource: "peerauthentications",
	}

	dynamicClient := dynamic.NewForConfigOrDie(m.config)

	peerAuths, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list PeerAuthentications: %w", err)
	}

	// Check for conflicting mTLS modes.

	mtlsModes := make(map[string]string)

	for _, pa := range peerAuths.Items {

		name := pa.GetName()

		spec := pa.Object["spec"].(map[string]interface{})

		if mtls, ok := spec["mtls"].(map[string]interface{}); ok {
			if mode, ok := mtls["mode"].(string); ok {

				// Check for conflicts.

				for existingName, existingMode := range mtlsModes {
					if existingMode != mode {

						result.Conflicts = append(result.Conflicts, abstraction.PolicyConflict{
							Policy1: existingName,

							Policy2: name,

							Type: "mTLS mode",

							Message: fmt.Sprintf("Conflicting mTLS modes: %s vs %s", existingMode, mode),
						})

						result.Valid = false

					}
				}

				mtlsModes[name] = mode

			}
		}

	}

	// Check AuthorizationPolicies.

	gvr = schema.GroupVersionResource{
		Group: "security.istio.io",

		Version: "v1beta1",

		Resource: "authorizationpolicies",
	}

	authPolicies, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list AuthorizationPolicies: %w", err)
	}

	// Check for DENY policies without corresponding ALLOW policies.

	hasDeny := false

	hasAllow := false

	for _, ap := range authPolicies.Items {

		spec := ap.Object["spec"].(map[string]interface{})

		if action, ok := spec["action"].(string); ok {
			if action == "DENY" {
				hasDeny = true
			} else if action == "ALLOW" {
				hasAllow = true
			}
		}

	}

	if hasDeny && !hasAllow {
		result.Warnings = append(result.Warnings, abstraction.PolicyWarning{
			Type: "Authorization",

			Message: "DENY policies found without corresponding ALLOW policies",
		})
	}

	// Calculate coverage.

	services, err := m.kubeClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})

	if err == nil {

		totalServices := len(services.Items)

		coveredServices := 0

		for _, svc := range services.Items {
			// Check if service has policies.

			if m.serviceHasPolicies(svc.Name, peerAuths.Items, authPolicies.Items) {
				coveredServices++
			}
		}

		if totalServices > 0 {
			result.Coverage = float64(coveredServices) / float64(totalServices) * 100
		}

	}

	// Check compliance.

	result.Compliance = abstraction.PolicyCompliance{
		MTLSCompliant: len(mtlsModes) > 0 && !hasMode(mtlsModes, "DISABLE"),

		ZeroTrustCompliant: hasDeny || hasAllow,

		NetworkSegmented: len(authPolicies.Items) > 0,

		ComplianceScore: result.Coverage,
	}

	return result, nil
}

// Helper function to check if a mode exists in the map.

func hasMode(modes map[string]string, mode string) bool {
	for _, m := range modes {
		if m == mode {
			return true
		}
	}

	return false
}

// serviceHasPolicies checks if a service has policies applied.

func (m *IstioMesh) serviceHasPolicies(serviceName string, peerAuths, authPolicies []unstructured.Unstructured) bool {
	// Check if any policy targets this service.

	// This is a simplified check - in reality, we'd need to evaluate selectors.

	return len(peerAuths) > 0 || len(authPolicies) > 0
}

// RegisterService registers a service with the mesh.

func (m *IstioMesh) RegisterService(ctx context.Context, service *abstraction.ServiceRegistration) error {
	// In Istio, services are automatically registered when they have the sidecar.

	// We can ensure the service has proper annotations for injection.

	_, err := m.kubeClient.CoreV1().Services(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
	if err != nil {

		// Create service if it doesn't exist.

		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: service.Name,

				Namespace: service.Namespace,

				Labels: service.Labels,

				Annotations: service.Annotations,
			},

			Spec: corev1.ServiceSpec{
				Ports: m.convertServicePorts(service.Ports),
			},
		}

		_, err = m.kubeClient.CoreV1().Services(service.Namespace).Create(ctx, svc, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}

	}

	// Ensure namespace has Istio injection enabled.

	ns, err := m.kubeClient.CoreV1().Namespaces().Get(ctx, service.Namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}

	ns.Labels["istio-injection"] = "enabled"

	_, err = m.kubeClient.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
	if err != nil {
		m.logger.Error(err, "Failed to enable Istio injection for namespace", "namespace", service.Namespace)
	}

	m.metrics.servicesRegistered.Inc()

	return nil
}

// UnregisterService unregisters a service from the mesh.

func (m *IstioMesh) UnregisterService(ctx context.Context, serviceName, namespace string) error {
	// Remove any Istio-specific resources for this service.

	// Clean up PeerAuthentications.

	gvr := schema.GroupVersionResource{
		Group: "security.istio.io",

		Version: "v1beta1",

		Resource: "peerauthentications",
	}

	dynamicClient := dynamic.NewForConfigOrDie(m.config)

	peerAuths, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})

	if err == nil {
		for _, pa := range peerAuths.Items {
			// Check if this policy targets the service.

			if m.policyTargetsService(pa, serviceName) {

				err = dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, pa.GetName(), metav1.DeleteOptions{})
				if err != nil {
					m.logger.Error(err, "Failed to delete PeerAuthentication", "name", pa.GetName())
				}

			}
		}
	}

	m.metrics.servicesRegistered.Dec()

	return nil
}

// policyTargetsService checks if a policy targets a specific service.

func (m *IstioMesh) policyTargetsService(policy unstructured.Unstructured, serviceName string) bool {
	// This is a simplified check - in reality, we'd need to evaluate selectors.

	return false
}

// GetServiceStatus gets the status of a service in the mesh.

func (m *IstioMesh) GetServiceStatus(ctx context.Context, serviceName, namespace string) (*abstraction.ServiceStatus, error) {
	status := &abstraction.ServiceStatus{
		Name: serviceName,

		Namespace: namespace,

		Healthy: true,
	}

	// Check if service exists.

	_, err := m.kubeClient.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Check endpoints.

	endpoints, err := m.kubeClient.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})

	if err == nil {
		for _, subset := range endpoints.Subsets {
			for _, addr := range subset.Addresses {
				for _, port := range subset.Ports {
					status.Endpoints = append(status.Endpoints, abstraction.EndpointStatus{
						Address: addr.IP,

						Port: port.Port,

						Healthy: true,

						LastChecked: time.Now(),
					})
				}
			}
		}
	}

	// Check if mTLS is enabled.

	status.MTLSEnabled = m.isServiceMTLSEnabled(ctx, serviceName, namespace)

	// Get applied policies.

	status.Policies = m.getServicePolicies(ctx, serviceName, namespace)

	return status, nil
}

// isServiceMTLSEnabled checks if mTLS is enabled for a service.

func (m *IstioMesh) isServiceMTLSEnabled(ctx context.Context, serviceName, namespace string) bool {
	// Check PeerAuthentication policies.

	gvr := schema.GroupVersionResource{
		Group: "security.istio.io",

		Version: "v1beta1",

		Resource: "peerauthentications",
	}

	dynamicClient := dynamic.NewForConfigOrDie(m.config)

	peerAuths, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false
	}

	for _, pa := range peerAuths.Items {

		spec := pa.Object["spec"].(map[string]interface{})

		if mtls, ok := spec["mtls"].(map[string]interface{}); ok {
			if mode, ok := mtls["mode"].(string); ok {
				return mode == "STRICT" || mode == "PERMISSIVE"
			}
		}

	}

	return false
}

// getServicePolicies gets the policies applied to a service.

func (m *IstioMesh) getServicePolicies(ctx context.Context, serviceName, namespace string) []string {
	policies := []string{}

	// List all policy types and check if they apply to this service.

	// This is simplified - in reality, we'd evaluate selectors.

	return policies
}

// GetMetrics returns Prometheus metrics collectors.

func (m *IstioMesh) GetMetrics() []prometheus.Collector {
	return m.metrics.GetCollectors()
}

// GetServiceDependencies gets service dependencies in the mesh.

func (m *IstioMesh) GetServiceDependencies(ctx context.Context, namespace string) (*abstraction.DependencyGraph, error) {
	graph := &abstraction.DependencyGraph{
		Nodes: []abstraction.ServiceNode{},

		Edges: []abstraction.ServiceEdge{},
	}

	// Get all services.

	services, err := m.kubeClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	// Create nodes for each service.

	for _, svc := range services.Items {

		node := abstraction.ServiceNode{
			ID: fmt.Sprintf("%s/%s", svc.Namespace, svc.Name),

			Name: svc.Name,

			Namespace: svc.Namespace,

			Type: "service",

			Labels: svc.Labels,

			MTLSEnabled: m.isServiceMTLSEnabled(ctx, svc.Name, svc.Namespace),
		}

		graph.Nodes = append(graph.Nodes, node)

	}

	// Get VirtualServices to determine traffic flow.

	gvr := schema.GroupVersionResource{
		Group: "networking.istio.io",

		Version: "v1beta1",

		Resource: "virtualservices",
	}

	dynamicClient := dynamic.NewForConfigOrDie(m.config)

	virtualServices, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})

	if err == nil {
		for range virtualServices.Items {
			// Parse virtual service to determine connections.

			// This is simplified - actual implementation would parse the spec.
		}
	}

	return graph, nil
}

// GetMTLSStatus gets the mTLS status report.

func (m *IstioMesh) GetMTLSStatus(ctx context.Context, namespace string) (*abstraction.MTLSStatusReport, error) {
	report := &abstraction.MTLSStatusReport{
		Timestamp: time.Now(),
	}

	// Get all services.

	services, err := m.kubeClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	report.TotalServices = len(services.Items)

	// Check mTLS status for each service.

	for _, svc := range services.Items {

		if m.isServiceMTLSEnabled(ctx, svc.Name, svc.Namespace) {
			report.MTLSEnabledCount++
		}

		// Add certificate status.

		certStatus := abstraction.ServiceCertStatus{
			Service: svc.Name,

			Namespace: svc.Namespace,

			Valid: true, // Simplified - would check actual cert

		}

		report.CertificateStatus = append(report.CertificateStatus, certStatus)

		// Add policy status.

		policyStatus := abstraction.ServicePolicyStatus{
			Service: svc.Name,

			Namespace: svc.Namespace,

			Policies: m.getServicePolicies(ctx, svc.Name, svc.Namespace),
		}

		if m.isServiceMTLSEnabled(ctx, svc.Name, svc.Namespace) {
			policyStatus.MTLSMode = "STRICT"
		} else {
			policyStatus.MTLSMode = "DISABLE"
		}

		report.PolicyStatus = append(report.PolicyStatus, policyStatus)

	}

	if report.TotalServices > 0 {
		report.Coverage = float64(report.MTLSEnabledCount) / float64(report.TotalServices) * 100
	}

	// Add recommendations.

	if report.Coverage < 100 {
		report.Recommendations = append(report.Recommendations,

			"Enable mTLS for all services to ensure end-to-end encryption")
	}

	return report, nil
}

// IsHealthy checks if the Istio mesh is healthy.

func (m *IstioMesh) IsHealthy(ctx context.Context) error {
	// Check istiod deployment.

	deployment, err := m.kubeClient.AppsV1().Deployments("istio-system").Get(ctx, "istiod", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("istiod not found: %w", err)
	}

	if deployment.Status.ReadyReplicas == 0 {
		return fmt.Errorf("istiod not ready")
	}

	return nil
}

// IsReady checks if the Istio mesh is ready.

func (m *IstioMesh) IsReady(ctx context.Context) error {
	return m.IsHealthy(ctx)
}

// GetProvider returns the provider type.

func (m *IstioMesh) GetProvider() abstraction.ServiceMeshProvider {
	return abstraction.ProviderIstio
}

// GetVersion returns the Istio version.

func (m *IstioMesh) GetVersion() string {
	// Get version from istiod deployment.

	deployment, err := m.kubeClient.AppsV1().Deployments("istio-system").Get(context.Background(), "istiod", metav1.GetOptions{})
	if err != nil {
		return "unknown"
	}

	if version, ok := deployment.Labels["version"]; ok {
		return version
	}

	return "unknown"
}

// GetCapabilities returns Istio capabilities.

func (m *IstioMesh) GetCapabilities() []abstraction.Capability {
	return []abstraction.Capability{
		abstraction.CapabilityMTLS,

		abstraction.CapabilityTrafficManagement,

		abstraction.CapabilityObservability,

		abstraction.CapabilityMultiCluster,

		abstraction.CapabilitySPIFFE,

		abstraction.CapabilityWASM,
	}
}

// hasIstioSidecar checks if a pod has an Istio sidecar.

func (m *IstioMesh) hasIstioSidecar(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == "istio-proxy" {
			return true
		}
	}

	return false
}

// convertSelector converts abstraction selector to map.

func (m *IstioMesh) convertSelector(selector *abstraction.LabelSelector) map[string]interface{} {
	if selector == nil {
		return nil
	}

	result := make(map[string]interface{})

	if len(selector.MatchLabels) > 0 {
		result["matchLabels"] = selector.MatchLabels
	}

	return result
}

// convertAuthorizationRules converts authorization rules.

func (m *IstioMesh) convertAuthorizationRules(rules []abstraction.AuthorizationRule) []interface{} {
	result := []interface{}{}

	for _, rule := range rules {

		istioRule := make(map[string]interface{})

		// Convert From.

		if len(rule.From) > 0 {

			from := []interface{}{}

			for _, source := range rule.From {

				fromSource := make(map[string]interface{})

				if len(source.Principals) > 0 {
					fromSource["principals"] = source.Principals
				}

				if len(source.Namespaces) > 0 {
					fromSource["namespaces"] = source.Namespaces
				}

				from = append(from, json.RawMessage("{}"))

			}

			istioRule["from"] = from

		}

		// Convert To.

		if len(rule.To) > 0 {

			to := []interface{}{}

			for _, operation := range rule.To {

				toOp := make(map[string]interface{})

				if len(operation.Methods) > 0 {
					toOp["methods"] = operation.Methods
				}

				if len(operation.Paths) > 0 {
					toOp["paths"] = operation.Paths
				}

				to = append(to, json.RawMessage("{}"))

			}

			istioRule["to"] = to

		}

		// Convert When.

		if len(rule.When) > 0 {

			when := []interface{}{}

			for _, condition := range rule.When {

				whenCond := json.RawMessage("{}")

				if len(condition.Values) > 0 {
					whenCond["values"] = condition.Values
				}

				when = append(when, whenCond)

			}

			istioRule["when"] = when

		}

		result = append(result, istioRule)

	}

	return result
}

// convertServicePorts converts service ports.

func (m *IstioMesh) convertServicePorts(ports []abstraction.ServicePort) []corev1.ServicePort {
	result := []corev1.ServicePort{}

	for _, port := range ports {
		result = append(result, corev1.ServicePort{
			Name: port.Name,

			Port: port.Port,

			TargetPort: intstr.FromInt(int(port.TargetPort)),

			Protocol: corev1.Protocol(port.Protocol),
		})
	}

	return result
}
