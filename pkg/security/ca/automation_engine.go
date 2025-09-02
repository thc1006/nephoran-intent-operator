// Package ca provides Certificate Authority automation and management functionality

// for the Nephoran Intent Operator security infrastructure.

package ca

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// AutomationEngine manages automated certificate operations.

type AutomationEngine struct {
	logger *logging.StructuredLogger

	config *AutomationConfig

	manager *CAManager

	caManager *CAManager // Required for Kubernetes integration

	healthChecker *HealthChecker

	kubeClient kubernetes.Interface

	watchers map[string]*ServiceWatcher

	watchersMux sync.RWMutex

	stopCh chan struct{}

	running bool

	runningMux sync.RWMutex

	requestQueue []*AutomationRequest

	requestQueueMux sync.RWMutex

	// Concurrency control fields for Kubernetes integration
	wg  sync.WaitGroup
	ctx context.Context
}

// AutomationConfig holds automation configuration.

type AutomationConfig struct {
	// Discovery settings.

	ServiceDiscoveryEnabled bool `yaml:"service_discovery_enabled"`

	DiscoveryInterval time.Duration `yaml:"discovery_interval"`

	DiscoveryNamespaces []string `yaml:"discovery_namespaces"`

	DiscoverySelectors []string `yaml:"discovery_selectors"`

	// Certificate management.

	AutoRenewalEnabled bool `yaml:"auto_renewal_enabled"`

	RenewalThreshold time.Duration `yaml:"renewal_threshold"`

	RenewalCheckInterval time.Duration `yaml:"renewal_check_interval"`

	CertificateBackup bool `yaml:"certificate_backup"`

	BackupRetention int `yaml:"backup_retention"`

	// Health checking.

	HealthCheckEnabled bool `yaml:"health_check_enabled"`

	HealthCheckTimeout time.Duration `yaml:"health_check_timeout"`

	HealthCheckRetries int `yaml:"health_check_retries"`

	HealthCheckInterval time.Duration `yaml:"health_check_interval"`

	// Notification settings.

	NotificationEnabled bool `yaml:"notification_enabled"`

	NotificationEndpoints []string `yaml:"notification_endpoints"`

	AlertThresholds map[string]int `yaml:"alert_thresholds"`

	// Performance tuning.

	MaxConcurrentOps int `yaml:"max_concurrent_operations"`

	OperationTimeout time.Duration `yaml:"operation_timeout"`

	BatchSize int `yaml:"batch_size"`

	ProcessingInterval time.Duration `yaml:"processing_interval"`

	// Security.

	AllowedServiceTypes []string `yaml:"allowed_service_types"`

	RequiredAnnotations map[string]string `yaml:"required_annotations"`

	TLSValidation bool `yaml:"tls_validation"`

	PolicyValidation bool `yaml:"policy_validation"`

	// Kubernetes Integration.

	KubernetesIntegration *KubernetesIntegrationConfig `yaml:"kubernetes_integration"`
}

// KubernetesIntegrationConfig holds Kubernetes integration configuration.

type KubernetesIntegrationConfig struct {
	// Service monitoring
	ServiceSelector  string   `yaml:"service_selector"`
	PodSelector      string   `yaml:"pod_selector"`
	IngressSelector  string   `yaml:"ingress_selector"`
	Namespaces       []string `yaml:"namespaces"`
	AnnotationPrefix string   `yaml:"annotation_prefix"`
	SecretPrefix     string   `yaml:"secret_prefix"`

	// Admission webhook configuration
	AdmissionWebhook AdmissionWebhookConfig `yaml:"admission_webhook"`
}

// AdmissionWebhookConfig holds admission webhook configuration.

type AdmissionWebhookConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	CertDir string `yaml:"cert_dir"`
}

// ServiceWatcher watches for service changes.

type ServiceWatcher struct {
	namespace string

	selector fields.Selector

	informer cache.SharedIndexInformer

	stopCh chan struct{}
}

// ServiceDiscoveryResult represents a discovered service.

type ServiceDiscoveryResult struct {
	Name string `json:"name"`

	Namespace string `json:"namespace"`

	Type string `json:"type"`

	Endpoint string `json:"endpoint"`

	Labels map[string]string `json:"labels"`

	Annotations map[string]string `json:"annotations"`

	Ports []ServicePort `json:"ports"`

	TLSEnabled bool `json:"tls_enabled"`

	CertInfo *CertificateInfo `json:"cert_info,omitempty"`
}

// ServicePort represents a service port.

type ServicePort struct {
	Name string `json:"name"`

	Port int32 `json:"port"`

	Protocol string `json:"protocol"`
}

// CertificateInfo represents certificate information for a service.

type CertificateInfo struct {
	SerialNumber string `json:"serial_number"`

	Subject string `json:"subject"`

	Issuer string `json:"issuer"`

	NotBefore time.Time `json:"not_before"`

	NotAfter time.Time `json:"not_after"`

	DNSNames []string `json:"dns_names"`

	IPAddresses []string `json:"ip_addresses"`
}

// AutomationRequest represents an automation request.

type AutomationRequest struct {
	Type RequestType `json:"type"`

	ServiceName string `json:"service_name"`

	ServiceNamespace string `json:"service_namespace"`

	CertificateTemplate *x509.Certificate `json:"certificate_template"`

	RenewalConfig *RenewalConfig `json:"renewal_config,omitempty"`

	HealthCheckConfig *HealthCheckConfig `json:"health_check_config,omitempty"`

	NotificationConfig *AutomationNotificationConfig `json:"notification_config,omitempty"`

	Priority RequestPriority `json:"priority"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RequestType defines types of automation requests.

type RequestType string

const (

	// RequestTypeDiscovery holds requesttypediscovery value.

	RequestTypeDiscovery RequestType = "discovery"

	// RequestTypeRenewal holds requesttyperenewal value.

	RequestTypeRenewal RequestType = "renewal"

	// RequestTypeProvisioning holds requesttypeprovisioning value.

	RequestTypeProvisioning RequestType = "provisioning"

	// RequestTypeRevocation holds requesttyperevocation value.

	RequestTypeRevocation RequestType = "revocation"

	// RequestTypeHealthCheck holds requesttypehealthcheck value.

	RequestTypeHealthCheck RequestType = "health_check"
)

// RequestPriority defines request priorities.

type RequestPriority int

const (

	// PriorityLow holds prioritylow value.

	PriorityLow RequestPriority = iota

	// PriorityNormal holds prioritynormal value.

	PriorityNormal

	// PriorityHigh holds priorityhigh value.

	PriorityHigh

	// PriorityCritical holds prioritycritical value.

	PriorityCritical
)

// RenewalConfig holds renewal configuration.

type RenewalConfig struct {
	Threshold time.Duration `json:"threshold"`

	MaxRetries int `json:"max_retries"`

	BackoffDelay time.Duration `json:"backoff_delay"`
}

// HealthCheckConfig holds health check configuration.

type HealthCheckConfig struct {
	Enabled bool `json:"enabled"`

	Timeout time.Duration `json:"timeout"`

	Retries int `json:"retries"`

	Interval time.Duration `json:"interval"`

	CheckInterval time.Duration `json:"check_interval"`

	TimeoutPerCheck time.Duration `json:"timeout_per_check"`

	ExpectedStatus int `json:"expected_status"`

	HTTPEndpoint string `json:"http_endpoint"`

	GRPCService string `json:"grpc_service"`

	HealthyThreshold int `json:"healthy_threshold"`

	ValidationMode string `json:"validation_mode"`
}

// NotificationConfig holds notification configuration.

type AutomationNotificationConfig struct {
	Enabled bool `json:"enabled"`

	Channels []string `json:"channels"`

	Events []string `json:"events"`

	Templates map[string]string `json:"templates"`
}

// AutomationResponse represents an automation response.

type AutomationResponse struct {
	RequestID string `json:"request_id"`

	Status ResponseStatus `json:"status"`

	Message string `json:"message"`

	Certificate *x509.Certificate `json:"certificate,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Duration time.Duration `json:"duration"`

	Error string `json:"error,omitempty"`
}

// ResponseStatus defines response statuses.

type ResponseStatus string

const (

	// StatusPending holds statuspending value.

	StatusPending ResponseStatus = "pending"

	// StatusProcessing holds statusprocessing value.

	StatusProcessing ResponseStatus = "processing"

	// StatusCompleted holds statuscompleted value.

	StatusCompleted ResponseStatus = "completed"

	// StatusFailed holds statusfailed value.

	StatusFailed ResponseStatus = "failed"

	// StatusCanceled holds statuscanceled value.

	StatusCanceled ResponseStatus = "canceled"
)

// ProvisioningRequest represents a certificate provisioning request for Kubernetes integration.

type ProvisioningRequest struct {
	ID          string            `json:"id"`
	ServiceName string            `json:"service_name"`
	Namespace   string            `json:"namespace"`
	Template    string            `json:"template"`
	DNSNames    []string          `json:"dns_names"`
	Priority    RequestPriority   `json:"priority"`
	Metadata    map[string]string `json:"metadata"`
}

// NewAutomationEngine creates a new automation engine.

func NewAutomationEngine(config *AutomationConfig, logger *logging.StructuredLogger, manager *CAManager, kubeClient kubernetes.Interface, k8sClient interface{}) (*AutomationEngine, error) {
	// Create a health checker if not provided.

	healthChecker := &HealthChecker{
		logger: logger,
	}

	engine := &AutomationEngine{
		logger: logger,

		config: config,

		manager: manager,

		caManager: manager, // Initialize caManager with the same reference as manager

		healthChecker: healthChecker,

		kubeClient: kubeClient,

		watchers: make(map[string]*ServiceWatcher),

		stopCh: make(chan struct{}),

		requestQueue: make([]*AutomationRequest, 0),

		wg: sync.WaitGroup{}, // Initialize WaitGroup

		ctx: context.Background(), // Initialize context
	}

	return engine, nil
}

// Start starts the automation engine.

func (e *AutomationEngine) Start(ctx context.Context) error {
	e.runningMux.Lock()

	defer e.runningMux.Unlock()

	if e.running {
		return fmt.Errorf("automation engine already running")
	}

	e.logger.Info("Starting automation engine",

		"service_discovery_enabled", e.config.ServiceDiscoveryEnabled,

		"auto_renewal_enabled", e.config.AutoRenewalEnabled,

		"health_check_enabled", e.config.HealthCheckEnabled)

	// Start service discovery if enabled.

	if e.config.ServiceDiscoveryEnabled {
		if err := e.startServiceDiscovery(ctx); err != nil {
			return fmt.Errorf("failed to start service discovery: %w", err)
		}
	}

	// Start renewal checker if enabled.

	if e.config.AutoRenewalEnabled {
		go e.runRenewalChecker(ctx)
	}

	// Start health checker if enabled.

	if e.config.HealthCheckEnabled {
		go e.runHealthChecker(ctx)
	}

	// Start request processor.

	go e.runRequestProcessor(ctx)

	e.running = true

	e.logger.Info("Automation engine started successfully")

	return nil
}

// Stop stops the automation engine.

func (e *AutomationEngine) Stop() error {
	e.runningMux.Lock()

	defer e.runningMux.Unlock()

	if !e.running {
		return nil
	}

	e.logger.Info("Stopping automation engine")

	// Stop all watchers.

	e.watchersMux.Lock()

	for namespace, watcher := range e.watchers {

		close(watcher.stopCh)

		delete(e.watchers, namespace)

	}

	e.watchersMux.Unlock()

	// Send stop signal.

	close(e.stopCh)

	e.stopCh = make(chan struct{})

	e.running = false

	e.logger.Info("Automation engine stopped")

	return nil
}

// startServiceDiscovery starts service discovery.

func (e *AutomationEngine) startServiceDiscovery(ctx context.Context) error {
	e.logger.Info("Starting service discovery",

		"namespaces", e.config.DiscoveryNamespaces,

		"selectors", e.config.DiscoverySelectors)

	// Start watchers for each namespace.

	for _, namespace := range e.config.DiscoveryNamespaces {
		if err := e.startNamespaceWatcher(namespace); err != nil {
			return fmt.Errorf("failed to start watcher for namespace %s: %w", namespace, err)
		}
	}

	return nil
}

// startNamespaceWatcher starts a watcher for a specific namespace.

func (e *AutomationEngine) startNamespaceWatcher(namespace string) error {
	e.watchersMux.Lock()

	defer e.watchersMux.Unlock()

	// Check if watcher already exists.

	if _, exists := e.watchers[namespace]; exists {
		return nil
	}

	// Create list watcher.

	listWatcher := cache.NewListWatchFromClient(

		e.kubeClient.CoreV1().RESTClient(),

		"services",

		namespace,

		fields.Everything(),
	)

	// Create informer.

	informer := cache.NewSharedIndexInformer(

		listWatcher,

		&v1.Service{},

		e.config.DiscoveryInterval,

		cache.Indexers{},
	)

	// Add event handlers.

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: e.onServiceAdded,

		UpdateFunc: e.onServiceUpdated,

		DeleteFunc: e.onServiceDeleted,
	})

	// Create watcher.

	watcher := &ServiceWatcher{
		namespace: namespace,

		selector: fields.Everything(),

		informer: informer,

		stopCh: make(chan struct{}),
	}

	e.watchers[namespace] = watcher

	// Start informer.

	go informer.Run(watcher.stopCh)

	// Wait for cache sync.

	if !cache.WaitForCacheSync(watcher.stopCh, informer.HasSynced) {
		return fmt.Errorf("failed to sync cache for namespace %s", namespace)
	}

	e.logger.Info("Started namespace watcher",

		"namespace", namespace)

	return nil
}

// onServiceAdded handles service addition events.

func (e *AutomationEngine) onServiceAdded(obj interface{}) {
	service, ok := obj.(*v1.Service)

	if !ok {

		e.logger.Warn("Received non-service object in service watcher")

		return

	}

	e.logger.Debug("Service added",

		"name", service.Name,

		"namespace", service.Namespace)

	// Process service for certificate provisioning.

	go e.processServiceForProvisioning(service)
}

// onServiceUpdated handles service update events.

func (e *AutomationEngine) onServiceUpdated(oldObj, newObj interface{}) {
	oldService, ok := oldObj.(*v1.Service)

	if !ok {
		return
	}

	newService, ok := newObj.(*v1.Service)

	if !ok {
		return
	}

	e.logger.Debug("Service updated",

		"name", newService.Name,

		"namespace", newService.Namespace)

	// Check if service configuration changed.

	if e.serviceConfigChanged(oldService, newService) {
		go e.processServiceForProvisioning(newService)
	}
}

// onServiceDeleted handles service deletion events.

func (e *AutomationEngine) onServiceDeleted(obj interface{}) {
	service, ok := obj.(*v1.Service)

	if !ok {
		return
	}

	e.logger.Debug("Service deleted",

		"name", service.Name,

		"namespace", service.Namespace)

	// Process service for certificate revocation.

	go e.processServiceForRevocation(service)
}

// serviceConfigChanged checks if service configuration changed.

func (e *AutomationEngine) serviceConfigChanged(oldService, newService *v1.Service) bool {
	// Check if TLS annotations changed.

	if oldService.Annotations["tls.enabled"] != newService.Annotations["tls.enabled"] {
		return true
	}

	// Check if certificate annotations changed.

	if oldService.Annotations["cert.auto-provision"] != newService.Annotations["cert.auto-provision"] {
		return true
	}

	// Check if service ports changed.

	if len(oldService.Spec.Ports) != len(newService.Spec.Ports) {
		return true
	}

	for i, oldPort := range oldService.Spec.Ports {

		newPort := newService.Spec.Ports[i]

		if oldPort.Port != newPort.Port || oldPort.Protocol != newPort.Protocol {
			return true
		}

	}

	return false
}

// processServiceForProvisioning processes a service for certificate provisioning.

func (e *AutomationEngine) processServiceForProvisioning(service *v1.Service) {
	// Check if service requires certificate provisioning.

	if !e.serviceRequiresCertificate(service) {
		return
	}

	e.logger.Info("Processing service for certificate provisioning",

		"name", service.Name,

		"namespace", service.Namespace)

	// Create automation request.

	req := &AutomationRequest{
		Type: RequestTypeProvisioning,

		ServiceName: service.Name,

		ServiceNamespace: service.Namespace,

		Priority: PriorityNormal,

		Metadata: map[string]interface{}{
			"service_uid": string(service.UID),

			"labels": service.Labels,

			"annotations": service.Annotations,
		},
	}

	// Add health check configuration if enabled.

	if e.config.HealthCheckEnabled {
		req.HealthCheckConfig = &HealthCheckConfig{
			Enabled: true,

			Timeout: e.config.HealthCheckTimeout,

			Retries: e.config.HealthCheckRetries,

			Interval: e.config.HealthCheckInterval,

			ExpectedStatus: 200,
		}
	}

	// Process request.

	go e.processAutomationRequest(req)
}

// processServiceForRevocation processes a service for certificate revocation.

func (e *AutomationEngine) processServiceForRevocation(service *v1.Service) {
	// Check if service had certificates.

	if !e.serviceRequiresCertificate(service) {
		return
	}

	e.logger.Info("Processing service for certificate revocation",

		"name", service.Name,

		"namespace", service.Namespace)

	// Create automation request.

	req := &AutomationRequest{
		Type: RequestTypeRevocation,

		ServiceName: service.Name,

		ServiceNamespace: service.Namespace,

		Priority: PriorityHigh,

		Metadata: map[string]interface{}{
			"service_uid": string(service.UID),

			"labels": service.Labels,

			"annotations": service.Annotations,
		},
	}

	// Process request.

	go e.processAutomationRequest(req)
}

// serviceRequiresCertificate checks if a service requires certificate management.

func (e *AutomationEngine) serviceRequiresCertificate(service *v1.Service) bool {
	// Check if service has TLS enabled annotation.

	if tls, exists := service.Annotations["tls.enabled"]; exists && tls == "true" {
		return true
	}

	// Check if service has auto-provision annotation.

	if provision, exists := service.Annotations["cert.auto-provision"]; exists && provision == "true" {
		return true
	}

	// Check if service type is in allowed list.

	serviceType := string(service.Spec.Type)

	for _, allowedType := range e.config.AllowedServiceTypes {
		if serviceType == allowedType {
			return true
		}
	}

	// Check for HTTPS ports.

	for _, port := range service.Spec.Ports {
		if port.Port == 443 || port.Port == 8443 {
			return true
		}
	}

	return false
}

// runRenewalChecker runs the certificate renewal checker.

func (e *AutomationEngine) runRenewalChecker(ctx context.Context) {
	ticker := time.NewTicker(e.config.RenewalCheckInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-e.stopCh:

			return

		case <-ticker.C:

			e.checkForRenewals(ctx)

		}
	}
}

// checkForRenewals checks for certificates that need renewal.

func (e *AutomationEngine) checkForRenewals(ctx context.Context) {
	e.logger.Debug("Checking for certificate renewals")

	// Get all managed certificates.

	certificates, err := e.manager.ListCertificates(nil)
	if err != nil {

		e.logger.Error("Failed to list certificates for renewal check",

			"error", err.Error())

		return

	}

	renewalCount := 0

	for _, cert := range certificates {

		// Check if certificate is close to expiration.

		timeUntilExpiry := time.Until(cert.ExpiresAt)

		if timeUntilExpiry <= e.config.RenewalThreshold {

			e.logger.Info("Certificate needs renewal",

				"serial", cert.SerialNumber,

				"subject", cert.Certificate.Subject.String(),

				"expires", cert.ExpiresAt,

				"time_until_expiry", timeUntilExpiry)

			// Create renewal request.

			req := &AutomationRequest{
				Type: RequestTypeRenewal,

				Priority: PriorityHigh,

				CertificateTemplate: cert.Certificate,

				Metadata: map[string]interface{}{
					"serial_number": cert.SerialNumber,

					"expires": cert.ExpiresAt,
				},
			}

			// Process renewal request.

			go e.processAutomationRequest(req)

			renewalCount++

		}

	}

	if renewalCount > 0 {
		e.logger.Info("Initiated certificate renewals",

			"renewal_count", renewalCount)
	}
}

// runHealthChecker runs the health checker.

func (e *AutomationEngine) runHealthChecker(ctx context.Context) {
	ticker := time.NewTicker(e.config.HealthCheckInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-e.stopCh:

			return

		case <-ticker.C:

			e.performHealthChecks(ctx)

		}
	}
}

// performHealthChecks performs health checks on managed services.

func (e *AutomationEngine) performHealthChecks(ctx context.Context) {
	e.logger.Debug("Performing health checks")

	// Get all discovered services.

	services, err := e.discoverServices(ctx)
	if err != nil {

		e.logger.Error("Failed to discover services for health checks",

			"error", err.Error())

		return

	}

	healthyCount := 0

	unhealthyCount := 0

	for _, service := range services {
		if service.TLSEnabled {

			// Perform health check.

			target := &HealthCheckTarget{
				Name: service.Name,

				Address: strings.Split(service.Endpoint, ":")[0],

				Port: e.parsePort(service.Endpoint),
			}

			config := &HealthCheckConfig{
				Enabled: true,

				Timeout: e.config.HealthCheckTimeout,

				Retries: e.config.HealthCheckRetries,

				ExpectedStatus: 200,
			}

			session, err := e.healthChecker.StartHealthCheck(target, config)
			if err != nil {

				e.logger.Warn("Failed to start health check",

					"service", service.Name,

					"error", err.Error())

				unhealthyCount++

				continue

			}

			// Perform health check.

			result, err := e.healthChecker.PerformHealthCheck(session)

			if err != nil || !result.Success {

				unhealthyCount++

				e.logger.Warn("Service health check failed",

					"service", service.Name,

					"error", func() string {
						if err != nil {
							return err.Error()
						}

						if result != nil {
							return result.ErrorMessage
						}

						return "unknown error"
					}())

			} else {
				healthyCount++
			}

		}
	}

	e.logger.Debug("Health checks completed",

		"healthy_services", healthyCount,

		"unhealthy_services", unhealthyCount)
}

// runRequestProcessor runs the automation request processor.

func (e *AutomationEngine) runRequestProcessor(ctx context.Context) {
	ticker := time.NewTicker(e.config.ProcessingInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-e.stopCh:

			return

		case <-ticker.C:

			// Process pending requests.

			// This would typically read from a queue or database.

			// For now, it's a placeholder.

		}
	}
}

// processAutomationRequest processes an automation request.

func (e *AutomationEngine) processAutomationRequest(req *AutomationRequest) {
	startTime := time.Now()

	e.logger.Info("Processing automation request",

		"type", req.Type,

		"service_name", req.ServiceName,

		"service_namespace", req.ServiceNamespace,

		"priority", req.Priority)

	var response *AutomationResponse

	switch req.Type {

	case RequestTypeProvisioning:

		response = e.processProvisioningRequest(req)

	case RequestTypeRenewal:

		response = e.processRenewalRequest(req)

	case RequestTypeRevocation:

		response = e.processRevocationRequest(req)

	case RequestTypeDiscovery:

		response = e.processDiscoveryRequest(req)

	case RequestTypeHealthCheck:

		response = e.processHealthCheckRequest(req)

	default:

		response = &AutomationResponse{
			Status: StatusFailed,

			Message: fmt.Sprintf("unknown request type: %s", req.Type),

			Timestamp: time.Now(),

			Duration: time.Since(startTime),
		}

	}

	e.logger.Info("Automation request processed",

		"type", req.Type,

		"status", response.Status,

		"duration", response.Duration,

		"message", response.Message)

	// Send notifications if configured.

	if e.config.NotificationEnabled && req.NotificationConfig != nil && req.NotificationConfig.Enabled {
		go e.sendNotification(req, response)
	}
}

// processProvisioningRequest processes a certificate provisioning request.

func (e *AutomationEngine) processProvisioningRequest(req *AutomationRequest) *AutomationResponse {
	startTime := time.Now()

	// Generate certificate for service.

	cert, err := e.generateServiceCertificate(req.ServiceName, req.ServiceNamespace)
	if err != nil {
		return &AutomationResponse{
			Status: StatusFailed,

			Message: "failed to generate certificate",

			Error: err.Error(),

			Timestamp: time.Now(),

			Duration: time.Since(startTime),
		}
	}

	// Store certificate.

	if err := e.storeServiceCertificate(req.ServiceName, req.ServiceNamespace, cert); err != nil {
		return &AutomationResponse{
			Status: StatusFailed,

			Message: "failed to store certificate",

			Error: err.Error(),

			Timestamp: time.Now(),

			Duration: time.Since(startTime),
		}
	}

	// Perform health check if configured.

	if req.HealthCheckConfig != nil && req.HealthCheckConfig.Enabled {

		// Discover service to get endpoint.

		services, err := e.discoverServices(context.Background())
		if err != nil {
			return &AutomationResponse{
				Status: StatusFailed,

				Message: "failed to discover service for health check",

				Error: err.Error(),

				Timestamp: time.Now(),

				Duration: time.Since(startTime),
			}
		}

		// Find the service.

		var serviceInstance *ServiceDiscoveryResult

		for _, service := range services {
			if service.Name == req.ServiceName && service.Namespace == req.ServiceNamespace {

				serviceInstance = service

				break

			}
		}

		if serviceInstance == nil {
			return &AutomationResponse{
				Status: StatusFailed,

				Message: "service not found for health check",

				Timestamp: time.Now(),

				Duration: time.Since(startTime),
			}
		}

		// Perform health check.

		target := &HealthCheckTarget{
			Name: serviceInstance.Name,

			Address: strings.Split(serviceInstance.Endpoint, ":")[0],

			Port: e.parsePort(serviceInstance.Endpoint),
		}

		healthSession, err := e.healthChecker.StartHealthCheck(target, req.HealthCheckConfig)
		if err != nil {
			return &AutomationResponse{
				Status: StatusFailed,

				Message: "failed to start health check",

				Error: err.Error(),

				Timestamp: time.Now(),

				Duration: time.Since(startTime),
			}
		}

		// Get health check result.

		result, err := e.healthChecker.PerformHealthCheck(healthSession)

		if err != nil || (result != nil && !result.Success) {

			errorMsg := "unknown error"

			if err != nil {
				errorMsg = err.Error()
			} else if result != nil {
				errorMsg = result.ErrorMessage
			}

			return &AutomationResponse{
				Status: StatusFailed,

				Message: "service health check failed after certificate provisioning",

				Error: errorMsg,

				Timestamp: time.Now(),

				Duration: time.Since(startTime),
			}

		}

	}

	return &AutomationResponse{
		Status: StatusCompleted,

		Message: "certificate provisioned successfully",

		Certificate: cert,

		Timestamp: time.Now(),

		Duration: time.Since(startTime),
	}
}

// processRenewalRequest processes a certificate renewal request.

func (e *AutomationEngine) processRenewalRequest(req *AutomationRequest) *AutomationResponse {
	startTime := time.Now()

	if req.CertificateTemplate == nil {
		return &AutomationResponse{
			Status: StatusFailed,

			Message: "certificate template required for renewal",

			Timestamp: time.Now(),

			Duration: time.Since(startTime),
		}
	}

	// Renew certificate.

	newCert, err := e.manager.RenewCertificate(context.Background(), req.CertificateTemplate.SerialNumber.String())
	if err != nil {
		return &AutomationResponse{
			Status: StatusFailed,

			Message: "failed to renew certificate",

			Error: err.Error(),

			Timestamp: time.Now(),

			Duration: time.Since(startTime),
		}
	}

	return &AutomationResponse{
		Status: StatusCompleted,

		Message: "certificate renewed successfully",

		Certificate: newCert.Certificate,

		Timestamp: time.Now(),

		Duration: time.Since(startTime),
	}
}

// processRevocationRequest processes a certificate revocation request.

func (e *AutomationEngine) processRevocationRequest(req *AutomationRequest) *AutomationResponse {
	startTime := time.Now()

	// Get certificates for service.

	certs, err := e.getServiceCertificates(req.ServiceName, req.ServiceNamespace)
	if err != nil {
		return &AutomationResponse{
			Status: StatusFailed,

			Message: "failed to get service certificates",

			Error: err.Error(),

			Timestamp: time.Now(),

			Duration: time.Since(startTime),
		}
	}

	revokedCount := 0

	for _, cert := range certs {
		if err := e.manager.RevokeCertificate(context.Background(), cert.SerialNumber.String(), 1, "service_deleted"); err != nil {
			e.logger.Warn("Failed to revoke certificate",

				"serial", cert.SerialNumber.String(),

				"error", err.Error())
		} else {
			revokedCount++
		}
	}

	return &AutomationResponse{
		Status: StatusCompleted,

		Message: fmt.Sprintf("revoked %d certificates", revokedCount),

		Timestamp: time.Now(),

		Duration: time.Since(startTime),

		Metadata: map[string]interface{}{
			"revoked_count": revokedCount,
		},
	}
}

// processDiscoveryRequest processes a service discovery request.

func (e *AutomationEngine) processDiscoveryRequest(req *AutomationRequest) *AutomationResponse {
	startTime := time.Now()

	services, err := e.discoverServices(context.Background())
	if err != nil {
		return &AutomationResponse{
			Status: StatusFailed,

			Message: "service discovery failed",

			Error: err.Error(),

			Timestamp: time.Now(),

			Duration: time.Since(startTime),
		}
	}

	return &AutomationResponse{
		Status: StatusCompleted,

		Message: fmt.Sprintf("discovered %d services", len(services)),

		Timestamp: time.Now(),

		Duration: time.Since(startTime),

		Metadata: map[string]interface{}{
			"services": services,

			"service_count": len(services),
		},
	}
}

// processHealthCheckRequest processes a health check request.

func (e *AutomationEngine) processHealthCheckRequest(req *AutomationRequest) *AutomationResponse {
	startTime := time.Now()

	// This would implement health checking logic.

	// For now, it's a placeholder.

	return &AutomationResponse{
		Status: StatusCompleted,

		Message: "health check completed",

		Timestamp: time.Now(),

		Duration: time.Since(startTime),
	}
}

// discoverServices discovers services in the cluster.

func (e *AutomationEngine) discoverServices(ctx context.Context) ([]*ServiceDiscoveryResult, error) {
	var allServices []*ServiceDiscoveryResult

	for _, namespace := range e.config.DiscoveryNamespaces {

		services, err := e.kubeClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list services in namespace %s: %w", namespace, err)
		}

		for _, service := range services.Items {
			if e.serviceRequiresCertificate(&service) {

				result := e.convertServiceToDiscoveryResult(&service)

				allServices = append(allServices, result)

			}
		}

	}

	return allServices, nil
}

// convertServiceToDiscoveryResult converts a Kubernetes service to discovery result.

func (e *AutomationEngine) convertServiceToDiscoveryResult(service *v1.Service) *ServiceDiscoveryResult {
	result := &ServiceDiscoveryResult{
		Name: service.Name,

		Namespace: service.Namespace,

		Type: string(service.Spec.Type),

		Labels: service.Labels,

		Annotations: service.Annotations,

		TLSEnabled: e.serviceRequiresCertificate(service),
	}

	// Extract service ports.

	for _, port := range service.Spec.Ports {
		result.Ports = append(result.Ports, ServicePort{
			Name: port.Name,

			Port: port.Port,

			Protocol: string(port.Protocol),
		})
	}

	// Determine service endpoint.

	if service.Spec.Type == v1.ServiceTypeLoadBalancer && len(service.Status.LoadBalancer.Ingress) > 0 {

		ingress := service.Status.LoadBalancer.Ingress[0]

		if ingress.IP != "" {
			result.Endpoint = fmt.Sprintf("%s:%d", ingress.IP, service.Spec.Ports[0].Port)
		} else if ingress.Hostname != "" {
			result.Endpoint = fmt.Sprintf("%s:%d", ingress.Hostname, service.Spec.Ports[0].Port)
		}

	} else if service.Spec.ClusterIP != "" && service.Spec.ClusterIP != "None" {
		result.Endpoint = fmt.Sprintf("%s:%d", service.Spec.ClusterIP, service.Spec.Ports[0].Port)
	}

	return result
}

// generateServiceCertificate generates a certificate for a service.

func (e *AutomationEngine) generateServiceCertificate(serviceName, namespace string) (*x509.Certificate, error) {
	// Create certificate request for service.

	subject := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)

	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: subject,

			OrganizationalUnit: []string{"Automated Certificate Management"},

			Organization: []string{"Nephoran Intent Operator"},
		},

		DNSNames: []string{
			subject,

			fmt.Sprintf("%s.%s", serviceName, namespace),

			serviceName,
		},
	}

	// Create certificate request from template.

	req := &CertificateRequest{
		ID: fmt.Sprintf("auto-%s-%s", serviceName, namespace),

		TenantID: "default",

		CommonName: template.Subject.CommonName,

		DNSNames: template.DNSNames,

		KeySize: 2048,

		ValidityDuration: 24 * time.Hour * 365, // 1 year

		KeyUsage: []string{"digital_signature", "key_encipherment"},

		ExtKeyUsage: []string{"server_auth", "client_auth"},

		AutoRenew: true,
	}

	// Generate certificate.

	cert, err := e.manager.IssueCertificate(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}

	return cert.Certificate, nil
}

// storeServiceCertificate stores a certificate for a service.

func (e *AutomationEngine) storeServiceCertificate(serviceName, namespace string, cert *x509.Certificate) error {
	// This would implement certificate storage logic.

	// For now, it's a placeholder.

	e.logger.Debug("Storing service certificate",

		"service", serviceName,

		"namespace", namespace,

		"serial", cert.SerialNumber.String())

	return nil
}

// getServiceCertificates gets certificates for a service.

func (e *AutomationEngine) getServiceCertificates(serviceName, namespace string) ([]*x509.Certificate, error) {
	// This would implement certificate retrieval logic.

	// For now, it's a placeholder.

	return []*x509.Certificate{}, nil
}

// parsePort parses port from endpoint string.

func (e *AutomationEngine) parsePort(endpoint string) int {
	parts := strings.Split(endpoint, ":")

	if len(parts) < 2 {
		return 443 // Default HTTPS port
	}

	port := 443

	if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
		return 443
	}

	return port
}

// sendNotification sends a notification for an automation request.

func (e *AutomationEngine) sendNotification(req *AutomationRequest, resp *AutomationResponse) {
	e.logger.Debug("Sending notification",

		"request_type", req.Type,

		"status", resp.Status,

		"channels", req.NotificationConfig.Channels)

	// This would implement notification sending logic.

	// For now, it's a placeholder.
}

// GetMetrics returns automation engine metrics.

func (e *AutomationEngine) GetMetrics() map[string]interface{} {
	e.runningMux.RLock()

	running := e.running

	e.runningMux.RUnlock()

	e.watchersMux.RLock()

	watcherCount := len(e.watchers)

	e.watchersMux.RUnlock()

	return map[string]interface{}{
		"running": running,

		"watcher_count": watcherCount,

		"config": map[string]interface{}{
			"service_discovery_enabled": e.config.ServiceDiscoveryEnabled,

			"auto_renewal_enabled": e.config.AutoRenewalEnabled,

			"health_check_enabled": e.config.HealthCheckEnabled,

			"notification_enabled": e.config.NotificationEnabled,
		},
	}
}

// IsRunning returns whether the automation engine is running.

func (e *AutomationEngine) IsRunning() bool {
	e.runningMux.RLock()

	defer e.runningMux.RUnlock()

	return e.running
}

// GetDiscoveredServices returns currently discovered services.

func (e *AutomationEngine) GetDiscoveredServices(ctx context.Context) ([]*ServiceDiscoveryResult, error) {
	if !e.config.ServiceDiscoveryEnabled {
		return nil, fmt.Errorf("service discovery not enabled")
	}

	return e.discoverServices(ctx)
}

// ProcessManualRequest processes a manual automation request synchronously.

func (e *AutomationEngine) ProcessManualRequest(ctx context.Context, req *AutomationRequest) *AutomationResponse {
	startTime := time.Now()

	switch req.Type {

	case RequestTypeProvisioning:

		return e.processProvisioningRequest(req)

	case RequestTypeRenewal:

		return e.processRenewalRequest(req)

	case RequestTypeRevocation:

		return e.processRevocationRequest(req)

	default:

		return &AutomationResponse{
			Status: StatusFailed,

			Message: fmt.Sprintf("unknown request type: %s", req.Type),

			Timestamp: time.Now(),

			Duration: time.Since(startTime),
		}

	}
}

// GetProvisioningQueueSize returns the current size of the provisioning queue.

func (e *AutomationEngine) GetProvisioningQueueSize() int {
	return len(e.requestQueue)
}

// GetRenewalQueueSize returns the current size of the renewal queue.

func (e *AutomationEngine) GetRenewalQueueSize() int {
	// For now, return 0 as we don't have a separate renewal queue.

	// In a full implementation, this would track renewal-specific requests.

	return 0
}

// RequestProvisioning submits a provisioning request for Kubernetes integration.

func (e *AutomationEngine) RequestProvisioning(req *ProvisioningRequest) error {
	if req == nil {
		return fmt.Errorf("provisioning request cannot be nil")
	}

	// Convert ProvisioningRequest to AutomationRequest
	automationReq := &AutomationRequest{
		Type:             RequestTypeProvisioning,
		ServiceName:      req.ServiceName,
		ServiceNamespace: req.Namespace,
		Priority:         req.Priority,
		Metadata: map[string]interface{}{
			"template":   req.Template,
			"dns_names":  req.DNSNames,
			"request_id": req.ID,
		},
	}

	// Add metadata from provisioning request
	for k, v := range req.Metadata {
		automationReq.Metadata[k] = v
	}

	// Process the request asynchronously
	go e.processAutomationRequest(automationReq)

	e.logger.Info("provisioning request submitted",
		"id", req.ID,
		"service", req.ServiceName,
		"namespace", req.Namespace,
		"dns_names", req.DNSNames)

	return nil
}
