// Package servicemesh provides the service mesh controller for Kubernetes integration
package servicemesh

import (
	"context"
	"fmt"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/servicemesh/abstraction"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/servicemesh/consul"  // Register consul provider
	_ "github.com/thc1006/nephoran-intent-operator/pkg/servicemesh/istio"   // Register istio provider
	_ "github.com/thc1006/nephoran-intent-operator/pkg/servicemesh/linkerd" // Register linkerd provider
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ServiceMeshController manages service mesh integration
type ServiceMeshController struct {
	client.Client
	Scheme      *runtime.Scheme
	kubeClient  kubernetes.Interface
	restConfig  *rest.Config
	meshFactory *abstraction.ServiceMeshFactory
	mesh        abstraction.ServiceMeshInterface
	meshConfig  *abstraction.ServiceMeshConfig
	logger      log.Logger
}

// NewServiceMeshController creates a new service mesh controller
func NewServiceMeshController(
	mgr manager.Manager,
	kubeClient kubernetes.Interface,
	meshConfig *abstraction.ServiceMeshConfig,
) (*ServiceMeshController, error) {
	return &ServiceMeshController{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		kubeClient:  kubeClient,
		restConfig:  mgr.GetConfig(),
		meshFactory: abstraction.NewServiceMeshFactory(kubeClient, mgr.GetClient(), mgr.GetConfig()),
		meshConfig:  meshConfig,
		logger:      log.Log.WithName("service-mesh-controller"),
	}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ServiceMeshController) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize the service mesh
	ctx := context.Background()
	mesh, err := r.meshFactory.CreateServiceMesh(ctx, r.meshConfig)
	if err != nil {
		return fmt.Errorf("failed to create service mesh: %w", err)
	}
	r.mesh = mesh

	// Set up the controller
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3,
		}).
		Complete(r)
}

// Reconcile handles service reconciliation for service mesh integration
func (r *ServiceMeshController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.logger.WithValues("service", req.NamespacedName)

	// Get the service
	var service corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		if errors.IsNotFound(err) {
			// Service was deleted, unregister from mesh
			if err := r.mesh.UnregisterService(ctx, req.Name, req.Namespace); err != nil {
				log.Error(err, "Failed to unregister service from mesh")
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if service should be managed by service mesh
	if !r.shouldManageService(&service) {
		return ctrl.Result{}, nil
	}

	// Register service with mesh
	registration := &abstraction.ServiceRegistration{
		Name:        service.Name,
		Namespace:   service.Namespace,
		Labels:      service.Labels,
		Annotations: service.Annotations,
		Ports:       r.convertServicePorts(service.Spec.Ports),
		MTLSMode:    r.getMTLSMode(&service),
	}

	if err := r.mesh.RegisterService(ctx, registration); err != nil {
		log.Error(err, "Failed to register service with mesh")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Apply default policies if configured
	if err := r.applyDefaultPolicies(ctx, &service); err != nil {
		log.Error(err, "Failed to apply default policies")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Update service annotations with mesh status
	if err := r.updateServiceAnnotations(ctx, &service); err != nil {
		log.Error(err, "Failed to update service annotations")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Check certificate status
	if err := r.checkCertificateStatus(ctx, &service); err != nil {
		log.Error(err, "Certificate check failed")
		// Don't fail reconciliation for certificate issues
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// shouldManageService determines if a service should be managed by the service mesh
func (r *ServiceMeshController) shouldManageService(service *corev1.Service) bool {
	// Skip system services
	if service.Namespace == "kube-system" || service.Namespace == "kube-public" {
		return false
	}

	// Check for opt-out annotation
	if val, ok := service.Annotations["mesh.nephoran.io/enabled"]; ok && val == "false" {
		return false
	}

	// Check for service mesh provider annotation
	if val, ok := service.Annotations["mesh.nephoran.io/provider"]; ok {
		return val == string(r.mesh.GetProvider())
	}

	// Default to managing all services in mesh-enabled namespaces
	return r.isNamespaceMeshEnabled(service.Namespace)
}

// isNamespaceMeshEnabled checks if a namespace is mesh-enabled
func (r *ServiceMeshController) isNamespaceMeshEnabled(namespace string) bool {
	ctx := context.Background()
	ns, err := r.kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return false
	}

	// Check for mesh injection label based on provider
	switch r.mesh.GetProvider() {
	case abstraction.ProviderIstio:
		return ns.Labels["istio-injection"] == "enabled"
	case abstraction.ProviderLinkerd:
		return ns.Labels["linkerd.io/inject"] == "enabled"
	case abstraction.ProviderConsul:
		return ns.Annotations["consul.hashicorp.com/connect-inject"] == "true"
	default:
		return false
	}
}

// getMTLSMode gets the mTLS mode for a service
func (r *ServiceMeshController) getMTLSMode(service *corev1.Service) string {
	// Check service annotation
	if mode, ok := service.Annotations["mesh.nephoran.io/mtls-mode"]; ok {
		return mode
	}

	// Use default from mesh config
	if r.meshConfig.PolicyDefaults != nil {
		return r.meshConfig.PolicyDefaults.MTLSMode
	}

	return "STRICT"
}

// convertServicePorts converts Kubernetes service ports to abstraction format
func (r *ServiceMeshController) convertServicePorts(ports []corev1.ServicePort) []abstraction.ServicePort {
	result := []abstraction.ServicePort{}
	for _, port := range ports {
		result = append(result, abstraction.ServicePort{
			Name:       port.Name,
			Port:       port.Port,
			TargetPort: port.TargetPort.IntVal,
			Protocol:   string(port.Protocol),
		})
	}
	return result
}

// applyDefaultPolicies applies default policies to a service
func (r *ServiceMeshController) applyDefaultPolicies(ctx context.Context, service *corev1.Service) error {
	// Apply mTLS policy
	mtlsPolicy := &abstraction.MTLSPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-mtls", service.Name),
			Namespace: service.Namespace,
		},
		Spec: abstraction.MTLSPolicySpec{
			Selector: &abstraction.LabelSelector{
				MatchLabels: map[string]string{
					"app": service.Name,
				},
			},
			Mode: r.getMTLSMode(service),
		},
	}

	if err := r.mesh.ApplyMTLSPolicy(ctx, mtlsPolicy); err != nil {
		return fmt.Errorf("failed to apply mTLS policy: %w", err)
	}

	// Apply authorization policy if deny-all is configured
	if r.meshConfig.PolicyDefaults != nil && r.meshConfig.PolicyDefaults.DefaultDenyAll {
		authPolicy := &abstraction.AuthorizationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-authz", service.Name),
				Namespace: service.Namespace,
			},
			Spec: abstraction.AuthorizationPolicySpec{
				Selector: &abstraction.LabelSelector{
					MatchLabels: map[string]string{
						"app": service.Name,
					},
				},
				Action: "DENY",
			},
		}

		if err := r.mesh.ApplyAuthorizationPolicy(ctx, authPolicy); err != nil {
			return fmt.Errorf("failed to apply authorization policy: %w", err)
		}
	}

	return nil
}

// updateServiceAnnotations updates service annotations with mesh status
func (r *ServiceMeshController) updateServiceAnnotations(ctx context.Context, service *corev1.Service) error {
	// Get service status from mesh
	status, err := r.mesh.GetServiceStatus(ctx, service.Name, service.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	// Update annotations
	if service.Annotations == nil {
		service.Annotations = make(map[string]string)
	}

	service.Annotations["mesh.nephoran.io/status"] = "registered"
	service.Annotations["mesh.nephoran.io/provider"] = string(r.mesh.GetProvider())
	service.Annotations["mesh.nephoran.io/mtls-enabled"] = fmt.Sprintf("%v", status.MTLSEnabled)
	service.Annotations["mesh.nephoran.io/last-sync"] = time.Now().Format(time.RFC3339)

	// Update the service
	if err := r.Update(ctx, service); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	return nil
}

// checkCertificateStatus checks certificate status for a service
func (r *ServiceMeshController) checkCertificateStatus(ctx context.Context, service *corev1.Service) error {
	certProvider := r.mesh.GetCertificateProvider()

	// Check if certificate is expiring soon
	cert, err := certProvider.IssueCertificate(ctx, service.Name, service.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}

	// Check expiry
	daysUntilExpiry := time.Until(cert.NotAfter).Hours() / 24
	if daysUntilExpiry < 30 {
		r.logger.Info("Certificate expiring soon", "service", service.Name, "daysUntilExpiry", daysUntilExpiry)

		// Trigger rotation if less than 7 days
		if daysUntilExpiry < 7 {
			if err := r.mesh.RotateCertificates(ctx, service.Namespace); err != nil {
				return fmt.Errorf("failed to rotate certificates: %w", err)
			}
		}
	}

	return nil
}

// EnforceMTLSEverywhere ensures mTLS is enabled for all services
func (r *ServiceMeshController) EnforceMTLSEverywhere(ctx context.Context) error {
	r.logger.Info("Enforcing mTLS everywhere")

	// List all namespaces
	namespaces, err := r.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		// Skip system namespaces
		if ns.Name == "kube-system" || ns.Name == "kube-public" {
			continue
		}

		// Apply strict mTLS policy for the namespace
		policy := &abstraction.MTLSPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-mtls",
				Namespace: ns.Name,
			},
			Spec: abstraction.MTLSPolicySpec{
				Mode: "STRICT",
			},
		}

		if err := r.mesh.ApplyMTLSPolicy(ctx, policy); err != nil {
			r.logger.Error(err, "Failed to apply mTLS policy", "namespace", ns.Name)
			continue
		}

		r.logger.Info("Applied strict mTLS policy", "namespace", ns.Name)
	}

	return nil
}

// ValidateServiceMeshHealth validates the health of the service mesh
func (r *ServiceMeshController) ValidateServiceMeshHealth(ctx context.Context) error {
	// Check mesh health
	if err := r.mesh.IsHealthy(ctx); err != nil {
		return fmt.Errorf("mesh is unhealthy: %w", err)
	}

	// Check mesh readiness
	if err := r.mesh.IsReady(ctx); err != nil {
		return fmt.Errorf("mesh is not ready: %w", err)
	}

	// Validate policies across all namespaces
	namespaces, err := r.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	totalIssues := 0
	for _, ns := range namespaces.Items {
		if ns.Name == "kube-system" || ns.Name == "kube-public" {
			continue
		}

		result, err := r.mesh.ValidatePolicies(ctx, ns.Name)
		if err != nil {
			r.logger.Error(err, "Failed to validate policies", "namespace", ns.Name)
			continue
		}

		if !result.Valid {
			totalIssues++
			r.logger.Warn("Policy validation failed",
				"namespace", ns.Name,
				"errors", len(result.Errors),
				"warnings", len(result.Warnings),
				"conflicts", len(result.Conflicts))
		}
	}

	if totalIssues > 0 {
		return fmt.Errorf("found %d namespaces with policy issues", totalIssues)
	}

	r.logger.Info("Service mesh health check completed successfully")
	return nil
}

// GetServiceMeshStatus returns the overall status of the service mesh
func (r *ServiceMeshController) GetServiceMeshStatus(ctx context.Context) (*ServiceMeshStatus, error) {
	status := &ServiceMeshStatus{
		Provider:     r.mesh.GetProvider(),
		Version:      r.mesh.GetVersion(),
		Capabilities: r.mesh.GetCapabilities(),
		Healthy:      true,
	}

	// Check health
	if err := r.mesh.IsHealthy(ctx); err != nil {
		status.Healthy = false
		status.Issues = append(status.Issues, fmt.Sprintf("Mesh unhealthy: %v", err))
	}

	// Get mTLS coverage across all namespaces
	totalServices := 0
	mtlsEnabledServices := 0

	namespaces, err := r.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		if ns.Name == "kube-system" || ns.Name == "kube-public" {
			continue
		}

		report, err := r.mesh.GetMTLSStatus(ctx, ns.Name)
		if err != nil {
			continue
		}

		totalServices += report.TotalServices
		mtlsEnabledServices += report.MTLSEnabledCount
	}

	if totalServices > 0 {
		status.MTLSCoverage = float64(mtlsEnabledServices) / float64(totalServices) * 100
	}

	status.TotalServices = totalServices
	status.MTLSEnabledServices = mtlsEnabledServices

	return status, nil
}

// ServiceMeshStatus represents the overall status of the service mesh
type ServiceMeshStatus struct {
	Provider            abstraction.ServiceMeshProvider `json:"provider"`
	Version             string                          `json:"version"`
	Capabilities        []abstraction.Capability        `json:"capabilities"`
	Healthy             bool                            `json:"healthy"`
	TotalServices       int                             `json:"totalServices"`
	MTLSEnabledServices int                             `json:"mtlsEnabledServices"`
	MTLSCoverage        float64                         `json:"mtlsCoverage"`
	Issues              []string                        `json:"issues,omitempty"`
}
