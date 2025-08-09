package ca

import (
	"context"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AutomationEngine provides comprehensive certificate automation capabilities
type AutomationEngine struct {
	config            *AutomationConfig
	logger            *logging.StructuredLogger
	caManager         *CAManager
	validator         *ValidationFramework
	revocationChecker *RevocationChecker
	kubeClient        kubernetes.Interface
	ctrlClient        client.Client

	// Internal state
	provisioningQueue chan *ProvisioningRequest
	renewalQueue      chan *RenewalRequest
	validationCache   *ValidationCache

	// Enhanced features
	rotationCoordinator *RotationCoordinator
	serviceDiscovery    *ServiceDiscovery
	performanceCache    *PerformanceCache
	healthChecker       *HealthChecker

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// AutomationConfig configures the automation engine
type AutomationConfig struct {
	// Provisioning settings
	ProvisioningEnabled     bool          `yaml:"provisioning_enabled"`
	AutoInjectCertificates  bool          `yaml:"auto_inject_certificates"`
	ServiceDiscoveryEnabled bool          `yaml:"service_discovery_enabled"`
	ProvisioningWorkers     int           `yaml:"provisioning_workers"`
	ProvisioningTimeout     time.Duration `yaml:"provisioning_timeout"`

	// Renewal settings
	RenewalEnabled         bool          `yaml:"renewal_enabled"`
	RenewalThreshold       time.Duration `yaml:"renewal_threshold"`
	RenewalWorkers         int           `yaml:"renewal_workers"`
	RenewalWindow          time.Duration `yaml:"renewal_window"`
	GracefulRenewalEnabled bool          `yaml:"graceful_renewal_enabled"`

	// Validation settings
	ValidationEnabled      bool          `yaml:"validation_enabled"`
	RealtimeValidation     bool          `yaml:"realtime_validation"`
	ChainValidationEnabled bool          `yaml:"chain_validation_enabled"`
	CTLogValidationEnabled bool          `yaml:"ct_log_validation_enabled"`
	ValidationCacheSize    int           `yaml:"validation_cache_size"`
	ValidationCacheTTL     time.Duration `yaml:"validation_cache_ttl"`

	// Revocation settings
	RevocationEnabled   bool          `yaml:"revocation_enabled"`
	CRLCheckEnabled     bool          `yaml:"crl_check_enabled"`
	OCSPCheckEnabled    bool          `yaml:"ocsp_check_enabled"`
	RevocationCacheSize int           `yaml:"revocation_cache_size"`
	RevocationCacheTTL  time.Duration `yaml:"revocation_cache_ttl"`

	// Integration settings
	KubernetesIntegration *K8sIntegrationConfig     `yaml:"kubernetes_integration"`
	MonitoringIntegration *MonitoringIntegration    `yaml:"monitoring_integration"`
	AlertingConfig        *AutomationAlertingConfig `yaml:"alerting_config"`

	// Performance settings
	BatchSize               int           `yaml:"batch_size"`
	MaxConcurrentOperations int           `yaml:"max_concurrent_operations"`
	OperationTimeout        time.Duration `yaml:"operation_timeout"`
	RetryAttempts           int           `yaml:"retry_attempts"`
	RetryBackoff            time.Duration `yaml:"retry_backoff"`
}

// K8sIntegrationConfig configures Kubernetes integration
type K8sIntegrationConfig struct {
	Enabled          bool                    `yaml:"enabled"`
	Namespaces       []string                `yaml:"namespaces"`
	ServiceSelector  string                  `yaml:"service_selector"`
	PodSelector      string                  `yaml:"pod_selector"`
	IngressSelector  string                  `yaml:"ingress_selector"`
	AdmissionWebhook *AdmissionWebhookConfig `yaml:"admission_webhook"`
	SecretPrefix     string                  `yaml:"secret_prefix"`
	AnnotationPrefix string                  `yaml:"annotation_prefix"`
}

// AdmissionWebhookConfig configures admission webhook for certificate injection
type AdmissionWebhookConfig struct {
	Enabled          bool   `yaml:"enabled"`
	WebhookName      string `yaml:"webhook_name"`
	ServiceName      string `yaml:"service_name"`
	ServiceNamespace string `yaml:"service_namespace"`
	CertDir          string `yaml:"cert_dir"`
	Port             int    `yaml:"port"`
}

// MonitoringIntegration configures monitoring integration
type MonitoringIntegration struct {
	PrometheusEnabled    bool `yaml:"prometheus_enabled"`
	JaegerEnabled        bool `yaml:"jaeger_enabled"`
	GrafanaEnabled       bool `yaml:"grafana_enabled"`
	CustomMetricsEnabled bool `yaml:"custom_metrics_enabled"`
}

// AutomationAlertingConfig configures automation-specific alerting
type AutomationAlertingConfig struct {
	Enabled                  bool          `yaml:"enabled"`
	ProvisioningFailureAlert bool          `yaml:"provisioning_failure_alert"`
	RenewalFailureAlert      bool          `yaml:"renewal_failure_alert"`
	ValidationFailureAlert   bool          `yaml:"validation_failure_alert"`
	RevocationAlert          bool          `yaml:"revocation_alert"`
	AlertCooldown            time.Duration `yaml:"alert_cooldown"`
}

// ProvisioningRequest represents a certificate provisioning request
type ProvisioningRequest struct {
	ID          string               `json:"id"`
	ServiceName string               `json:"service_name"`
	Namespace   string               `json:"namespace"`
	Template    string               `json:"template"`
	DNSNames    []string             `json:"dns_names"`
	IPAddresses []string             `json:"ip_addresses"`
	Metadata    map[string]string    `json:"metadata"`
	Priority    ProvisioningPriority `json:"priority"`
	CreatedAt   time.Time            `json:"created_at"`
	Deadline    time.Time            `json:"deadline,omitempty"`
}

// ProvisioningPriority represents the priority of provisioning requests
type ProvisioningPriority int

const (
	PriorityLow ProvisioningPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// RenewalRequest represents a certificate renewal request
type RenewalRequest struct {
	SerialNumber        string             `json:"serial_number"`
	CurrentExpiry       time.Time          `json:"current_expiry"`
	ServiceName         string             `json:"service_name"`
	Namespace           string             `json:"namespace"`
	RenewalWindow       time.Duration      `json:"renewal_window"`
	GracefulRenewal     bool               `json:"graceful_renewal"`
	ZeroDowntime        bool               `json:"zero_downtime"`
	CoordinatedRotation bool               `json:"coordinated_rotation"`
	HealthCheckConfig   *HealthCheckConfig `json:"health_check_config,omitempty"`
	Metadata            map[string]string  `json:"metadata"`
	CreatedAt           time.Time          `json:"created_at"`
}

// HealthCheckConfig defines health check configuration for zero-downtime rotation
type HealthCheckConfig struct {
	Enabled            bool          `json:"enabled"`
	HTTPEndpoint       string        `json:"http_endpoint"`
	GRPCService        string        `json:"grpc_service"`
	CheckInterval      time.Duration `json:"check_interval"`
	TimeoutPerCheck    time.Duration `json:"timeout_per_check"`
	HealthyThreshold   int           `json:"healthy_threshold"`
	UnhealthyThreshold int           `json:"unhealthy_threshold"`
}

// ValidationCache caches certificate validation results
type ValidationCache struct {
	cache map[string]*ValidationResult
	ttl   time.Duration
	mu    sync.RWMutex
}

// ValidationResult represents a certificate validation result
type ValidationResult struct {
	SerialNumber     string           `json:"serial_number"`
	Valid            bool             `json:"valid"`
	ChainValid       bool             `json:"chain_valid"`
	CTLogVerified    bool             `json:"ct_log_verified"`
	RevocationStatus RevocationStatus `json:"revocation_status"`
	ValidationTime   time.Time        `json:"validation_time"`
	ExpiresAt        time.Time        `json:"expires_at"`
	Errors           []string         `json:"errors,omitempty"`
	Warnings         []string         `json:"warnings,omitempty"`
}

// RevocationStatus represents certificate revocation status
type RevocationStatus string

const (
	RevocationStatusGood    RevocationStatus = "good"
	RevocationStatusRevoked RevocationStatus = "revoked"
	RevocationStatusUnknown RevocationStatus = "unknown"
)

// NewAutomationEngine creates a new certificate automation engine
func NewAutomationEngine(
	config *AutomationConfig,
	logger *logging.StructuredLogger,
	caManager *CAManager,
	kubeClient kubernetes.Interface,
	ctrlClient client.Client,
) (*AutomationEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("automation config is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &AutomationEngine{
		config:            config,
		logger:            logger,
		caManager:         caManager,
		kubeClient:        kubeClient,
		ctrlClient:        ctrlClient,
		provisioningQueue: make(chan *ProvisioningRequest, config.MaxConcurrentOperations),
		renewalQueue:      make(chan *RenewalRequest, config.MaxConcurrentOperations),
		ctx:               ctx,
		cancel:            cancel,
	}

	// Initialize rotation coordinator
	engine.rotationCoordinator = NewRotationCoordinator(logger, kubeClient)

	// Initialize service discovery
	if config.KubernetesIntegration != nil && config.ServiceDiscoveryEnabled {
		serviceDiscoveryConfig := &ServiceDiscoveryConfig{
			Enabled:                 true,
			WatchNamespaces:         config.KubernetesIntegration.Namespaces,
			ServiceAnnotationPrefix: config.KubernetesIntegration.AnnotationPrefix,
			AutoProvisionEnabled:    config.AutoInjectCertificates,
			TemplateMatching: &TemplateMatchingConfig{
				Enabled:          true,
				DefaultTemplate:  "default",
				FallbackBehavior: "default",
			},
			PreProvisioningEnabled: config.ProvisioningEnabled,
		}
		engine.serviceDiscovery = NewServiceDiscovery(logger, kubeClient, serviceDiscoveryConfig, engine)
	}

	// Initialize performance cache
	performanceCacheConfig := &PerformanceCacheConfig{
		L1CacheSize:            1000,
		L1CacheTTL:             5 * time.Minute,
		L2CacheSize:            5000,
		L2CacheTTL:             30 * time.Minute,
		PreProvisioningEnabled: true,
		PreProvisioningSize:    100,
		PreProvisioningTTL:     24 * time.Hour,
		BatchOperationsEnabled: config.BatchSize > 1,
		BatchSize:              config.BatchSize,
		BatchTimeout:           config.OperationTimeout,
		MetricsEnabled:         config.MonitoringIntegration != nil && config.MonitoringIntegration.PrometheusEnabled,
		CleanupInterval:        1 * time.Hour,
	}
	engine.performanceCache = NewPerformanceCache(logger, performanceCacheConfig)

	// Initialize health checker
	healthCheckerConfig := &HealthCheckerConfig{
		Enabled:                   true,
		DefaultTimeout:            30 * time.Second,
		DefaultInterval:           10 * time.Second,
		DefaultHealthyThreshold:   3,
		DefaultUnhealthyThreshold: 3,
		ConcurrentChecks:          10,
		RetryAttempts:             3,
		RetryDelay:                5 * time.Second,
		TLSVerificationEnabled:    true,
		MetricsEnabled:            config.MonitoringIntegration != nil && config.MonitoringIntegration.PrometheusEnabled,
	}
	engine.healthChecker = NewHealthChecker(logger, healthCheckerConfig)

	// Initialize validation framework
	if config.ValidationEnabled {
		validator, err := NewValidationFramework(&ValidationConfig{
			RealtimeValidation:     config.RealtimeValidation,
			ChainValidationEnabled: config.ChainValidationEnabled,
			CTLogValidationEnabled: config.CTLogValidationEnabled,
		}, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to initialize validation framework: %w", err)
		}
		engine.validator = validator

		// Initialize validation cache
		engine.validationCache = &ValidationCache{
			cache: make(map[string]*ValidationResult),
			ttl:   config.ValidationCacheTTL,
		}
	}

	// Initialize revocation checker
	if config.RevocationEnabled {
		checker, err := NewRevocationChecker(&RevocationConfig{
			CRLCheckEnabled:  config.CRLCheckEnabled,
			OCSPCheckEnabled: config.OCSPCheckEnabled,
			CacheSize:        config.RevocationCacheSize,
			CacheTTL:         config.RevocationCacheTTL,
		}, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to initialize revocation checker: %w", err)
		}
		engine.revocationChecker = checker
	}

	return engine, nil
}

// Start starts the automation engine
func (e *AutomationEngine) Start(ctx context.Context) error {
	e.logger.Info("starting certificate automation engine")

	// Start performance cache
	if e.performanceCache != nil {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			if err := e.performanceCache.Start(ctx); err != nil {
				e.logger.Error("performance cache start failed", "error", err)
			}
		}()
	}

	// Start health checker
	if e.healthChecker != nil {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			if err := e.healthChecker.Start(ctx); err != nil {
				e.logger.Error("health checker start failed", "error", err)
			}
		}()
	}

	// Start service discovery
	if e.serviceDiscovery != nil {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			if err := e.serviceDiscovery.Start(ctx); err != nil {
				e.logger.Error("service discovery start failed", "error", err)
			}
		}()
	}

	// Start provisioning workers
	if e.config.ProvisioningEnabled {
		for i := 0; i < e.config.ProvisioningWorkers; i++ {
			e.wg.Add(1)
			go e.runProvisioningWorker(i)
		}
	}

	// Start renewal workers
	if e.config.RenewalEnabled {
		for i := 0; i < e.config.RenewalWorkers; i++ {
			e.wg.Add(1)
			go e.runRenewalWorker(i)
		}
	}

	// Start renewal scheduler
	if e.config.RenewalEnabled {
		e.wg.Add(1)
		go e.runRenewalScheduler()
	}

	// Start validation service
	if e.validator != nil {
		e.wg.Add(1)
		go e.runValidationService()
	}

	// Start Kubernetes integration
	if e.config.KubernetesIntegration != nil && e.config.KubernetesIntegration.Enabled {
		e.wg.Add(1)
		go e.runKubernetesIntegration()
	}

	// Start admission webhook
	if e.config.KubernetesIntegration != nil &&
		e.config.KubernetesIntegration.AdmissionWebhook != nil &&
		e.config.KubernetesIntegration.AdmissionWebhook.Enabled {
		e.wg.Add(1)
		go e.runAdmissionWebhook()
	}

	// Wait for context cancellation
	<-ctx.Done()

	e.logger.Info("shutting down certificate automation engine")
	e.cancel()
	e.wg.Wait()

	return nil
}

// Stop stops the automation engine
func (e *AutomationEngine) Stop() {
	e.logger.Info("stopping certificate automation engine")
	e.cancel()
	e.wg.Wait()
}

// RequestProvisioning requests certificate provisioning
func (e *AutomationEngine) RequestProvisioning(req *ProvisioningRequest) error {
	if !e.config.ProvisioningEnabled {
		return fmt.Errorf("certificate provisioning is disabled")
	}

	req.CreatedAt = time.Now()
	if req.ID == "" {
		req.ID = generateRequestID()
	}

	select {
	case e.provisioningQueue <- req:
		e.logger.Info("provisioning request queued",
			"request_id", req.ID,
			"service", req.ServiceName,
			"namespace", req.Namespace,
			"priority", req.Priority)
		return nil
	default:
		return fmt.Errorf("provisioning queue is full")
	}
}

// RequestRenewal requests certificate renewal
func (e *AutomationEngine) RequestRenewal(req *RenewalRequest) error {
	if !e.config.RenewalEnabled {
		return fmt.Errorf("certificate renewal is disabled")
	}

	req.CreatedAt = time.Now()

	select {
	case e.renewalQueue <- req:
		e.logger.Info("renewal request queued",
			"serial_number", req.SerialNumber,
			"service", req.ServiceName,
			"namespace", req.Namespace,
			"current_expiry", req.CurrentExpiry)
		return nil
	default:
		return fmt.Errorf("renewal queue is full")
	}
}

// ValidateCertificate validates a certificate with caching
func (e *AutomationEngine) ValidateCertificate(cert *x509.Certificate) (*ValidationResult, error) {
	if e.validator == nil {
		return nil, fmt.Errorf("certificate validation is disabled")
	}

	serialNumber := cert.SerialNumber.String()

	// Check cache first
	if cached := e.getCachedValidation(serialNumber); cached != nil {
		if time.Since(cached.ValidationTime) < e.config.ValidationCacheTTL {
			return cached, nil
		}
	}

	// Perform validation
	result, err := e.validator.ValidateCertificate(e.ctx, cert)
	if err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	// Cache result
	e.cacheValidation(serialNumber, result)

	return result, nil
}

// CheckRevocationStatus checks certificate revocation status
func (e *AutomationEngine) CheckRevocationStatus(cert *x509.Certificate) (RevocationStatus, error) {
	if e.revocationChecker == nil {
		return RevocationStatusUnknown, fmt.Errorf("revocation checking is disabled")
	}

	return e.revocationChecker.CheckRevocation(e.ctx, cert)
}

// GetProvisioningQueueSize returns the current provisioning queue size
func (e *AutomationEngine) GetProvisioningQueueSize() int {
	return len(e.provisioningQueue)
}

// GetRenewalQueueSize returns the current renewal queue size
func (e *AutomationEngine) GetRenewalQueueSize() int {
	return len(e.renewalQueue)
}

// Worker methods

func (e *AutomationEngine) runProvisioningWorker(workerID int) {
	defer e.wg.Done()

	e.logger.Info("starting provisioning worker", "worker_id", workerID)

	for {
		select {
		case <-e.ctx.Done():
			return
		case req := <-e.provisioningQueue:
			e.processProvisioningRequest(workerID, req)
		}
	}
}

func (e *AutomationEngine) processProvisioningRequest(workerID int, req *ProvisioningRequest) {
	start := time.Now()

	e.logger.Info("processing provisioning request",
		"worker_id", workerID,
		"request_id", req.ID,
		"service", req.ServiceName,
		"namespace", req.Namespace)

	// Create certificate request
	certReq := &CertificateRequest{
		ID:               req.ID,
		CommonName:       req.ServiceName,
		DNSNames:         req.DNSNames,
		IPAddresses:      req.IPAddresses,
		ValidityDuration: e.caManager.config.DefaultValidityDuration,
		AutoRenew:        true,
		PolicyTemplate:   req.Template,
		Metadata:         req.Metadata,
	}

	// Add service-specific metadata
	if certReq.Metadata == nil {
		certReq.Metadata = make(map[string]string)
	}
	certReq.Metadata["service_name"] = req.ServiceName
	certReq.Metadata["namespace"] = req.Namespace
	certReq.Metadata["provisioned_by"] = "automation-engine"
	certReq.Metadata["provisioned_at"] = time.Now().Format(time.RFC3339)

	// Issue certificate
	response, err := e.caManager.IssueCertificate(e.ctx, certReq)
	if err != nil {
		duration := time.Since(start)
		e.logger.Error("certificate provisioning failed",
			"worker_id", workerID,
			"request_id", req.ID,
			"error", err,
			"duration", duration)

		// Send alert if configured
		e.sendAlert("provisioning_failure", map[string]string{
			"request_id": req.ID,
			"service":    req.ServiceName,
			"namespace":  req.Namespace,
			"error":      err.Error(),
		})
		return
	}

	// Store certificate in Kubernetes if integration is enabled
	if e.config.KubernetesIntegration != nil && e.config.KubernetesIntegration.Enabled {
		if err := e.storeCertificateInKubernetes(req, response); err != nil {
			e.logger.Warn("failed to store certificate in Kubernetes",
				"request_id", req.ID,
				"serial_number", response.SerialNumber,
				"error", err)
		}
	}

	duration := time.Since(start)
	e.logger.Info("certificate provisioned successfully",
		"worker_id", workerID,
		"request_id", req.ID,
		"serial_number", response.SerialNumber,
		"expires_at", response.ExpiresAt,
		"duration", duration)
}

func (e *AutomationEngine) runRenewalWorker(workerID int) {
	defer e.wg.Done()

	e.logger.Info("starting renewal worker", "worker_id", workerID)

	for {
		select {
		case <-e.ctx.Done():
			return
		case req := <-e.renewalQueue:
			e.processRenewalRequest(workerID, req)
		}
	}
}

func (e *AutomationEngine) processRenewalRequest(workerID int, req *RenewalRequest) {
	start := time.Now()

	e.logger.Info("processing renewal request",
		"worker_id", workerID,
		"serial_number", req.SerialNumber,
		"service", req.ServiceName,
		"namespace", req.Namespace,
		"current_expiry", req.CurrentExpiry,
		"zero_downtime", req.ZeroDowntime,
		"coordinated_rotation", req.CoordinatedRotation)

	// Check if coordinated rotation is requested
	if req.CoordinatedRotation && e.rotationCoordinator != nil {
		e.performCoordinatedRenewal(workerID, req)
		return
	}

	// Standard renewal process
	newCert, err := e.caManager.RenewCertificate(e.ctx, req.SerialNumber)
	if err != nil {
		duration := time.Since(start)
		e.logger.Error("certificate renewal failed",
			"worker_id", workerID,
			"serial_number", req.SerialNumber,
			"error", err,
			"duration", duration)

		// Send alert if configured
		e.sendAlert("renewal_failure", map[string]string{
			"serial_number": req.SerialNumber,
			"service":       req.ServiceName,
			"namespace":     req.Namespace,
			"error":         err.Error(),
		})
		return
	}

	// Update certificate in Kubernetes if integration is enabled
	if e.config.KubernetesIntegration != nil && e.config.KubernetesIntegration.Enabled {
		if err := e.updateCertificateInKubernetes(req, newCert); err != nil {
			e.logger.Warn("failed to update certificate in Kubernetes",
				"old_serial_number", req.SerialNumber,
				"new_serial_number", newCert.SerialNumber,
				"error", err)
		}
	}

	duration := time.Since(start)
	e.logger.Info("certificate renewed successfully",
		"worker_id", workerID,
		"old_serial_number", req.SerialNumber,
		"new_serial_number", newCert.SerialNumber,
		"expires_at", newCert.ExpiresAt,
		"duration", duration)
}

func (e *AutomationEngine) runRenewalScheduler() {
	defer e.wg.Done()

	e.logger.Info("starting renewal scheduler")

	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.scheduleRenewals()
		}
	}
}

func (e *AutomationEngine) scheduleRenewals() {
	threshold := time.Now().Add(e.config.RenewalThreshold)

	// Get expiring certificates from CA manager
	filters := map[string]string{
		"auto_renew": "true",
		"status":     string(StatusIssued),
	}

	certificates, err := e.caManager.ListCertificates(filters)
	if err != nil {
		e.logger.Error("failed to list certificates for renewal", "error", err)
		return
	}

	renewalCount := 0
	for _, cert := range certificates {
		if cert.ExpiresAt.Before(threshold) {
			// Create renewal request
			req := &RenewalRequest{
				SerialNumber:    cert.SerialNumber,
				CurrentExpiry:   cert.ExpiresAt,
				ServiceName:     cert.Metadata["service_name"],
				Namespace:       cert.Metadata["namespace"],
				RenewalWindow:   e.config.RenewalWindow,
				GracefulRenewal: e.config.GracefulRenewalEnabled,
				Metadata:        cert.Metadata,
			}

			if err := e.RequestRenewal(req); err != nil {
				e.logger.Warn("failed to queue renewal request",
					"serial_number", cert.SerialNumber,
					"error", err)
			} else {
				renewalCount++
			}
		}
	}

	if renewalCount > 0 {
		e.logger.Info("scheduled certificate renewals",
			"count", renewalCount,
			"threshold", threshold)
	}
}

func (e *AutomationEngine) runValidationService() {
	defer e.wg.Done()

	e.logger.Info("starting validation service")

	ticker := time.NewTicker(5 * time.Minute) // Validate every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.performBatchValidation()
		}
	}
}

func (e *AutomationEngine) performBatchValidation() {
	// Get active certificates
	filters := map[string]string{
		"status": string(StatusIssued),
	}

	certificates, err := e.caManager.ListCertificates(filters)
	if err != nil {
		e.logger.Error("failed to list certificates for validation", "error", err)
		return
	}

	validationCount := 0
	for _, cert := range certificates {
		// Skip recently validated certificates
		if cached := e.getCachedValidation(cert.SerialNumber); cached != nil {
			if time.Since(cached.ValidationTime) < e.config.ValidationCacheTTL {
				continue
			}
		}

		// Validate certificate
		if _, err := e.ValidateCertificate(cert.Certificate); err != nil {
			e.logger.Warn("certificate validation failed",
				"serial_number", cert.SerialNumber,
				"error", err)

			// Send alert if configured
			e.sendAlert("validation_failure", map[string]string{
				"serial_number": cert.SerialNumber,
				"error":         err.Error(),
			})
		} else {
			validationCount++
		}
	}

	if validationCount > 0 {
		e.logger.Debug("performed batch certificate validation", "validated_count", validationCount)
	}
}

// Cache methods

func (e *AutomationEngine) getCachedValidation(serialNumber string) *ValidationResult {
	if e.validationCache == nil {
		return nil
	}

	e.validationCache.mu.RLock()
	defer e.validationCache.mu.RUnlock()

	return e.validationCache.cache[serialNumber]
}

func (e *AutomationEngine) cacheValidation(serialNumber string, result *ValidationResult) {
	if e.validationCache == nil {
		return
	}

	e.validationCache.mu.Lock()
	defer e.validationCache.mu.Unlock()

	e.validationCache.cache[serialNumber] = result

	// Clean up expired entries if cache is getting large
	if len(e.validationCache.cache) > e.config.ValidationCacheSize {
		e.cleanupValidationCache()
	}
}

func (e *AutomationEngine) cleanupValidationCache() {
	now := time.Now()
	for serialNumber, result := range e.validationCache.cache {
		if now.Sub(result.ValidationTime) > e.validationCache.ttl {
			delete(e.validationCache.cache, serialNumber)
		}
	}
}

// Alert helper

func (e *AutomationEngine) sendAlert(alertType string, labels map[string]string) {
	if e.config.AlertingConfig == nil || !e.config.AlertingConfig.Enabled {
		return
	}

	// This would integrate with the existing alert manager
	e.logger.Warn("automation alert triggered",
		"type", alertType,
		"labels", labels)
}

// Helper methods

func (e *AutomationEngine) storeCertificateInKubernetes(req *ProvisioningRequest, cert *CertificateResponse) error {
	secretName := fmt.Sprintf("%s-%s-tls", e.config.KubernetesIntegration.SecretPrefix, req.ServiceName)

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: req.Namespace,
			Annotations: map[string]string{
				fmt.Sprintf("%s/serial-number", e.config.KubernetesIntegration.AnnotationPrefix):  cert.SerialNumber,
				fmt.Sprintf("%s/expires-at", e.config.KubernetesIntegration.AnnotationPrefix):     cert.ExpiresAt.Format(time.RFC3339),
				fmt.Sprintf("%s/issued-by", e.config.KubernetesIntegration.AnnotationPrefix):      cert.IssuedBy,
				fmt.Sprintf("%s/provisioned-by", e.config.KubernetesIntegration.AnnotationPrefix): "automation-engine",
			},
		},
		Type: v1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": []byte(cert.CertificatePEM),
			"tls.key": []byte(cert.PrivateKeyPEM),
			"ca.crt":  []byte(cert.CACertificatePEM),
		},
	}

	_, err := e.kubeClient.CoreV1().Secrets(req.Namespace).Create(e.ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create certificate secret: %w", err)
	}

	return nil
}

func (e *AutomationEngine) updateCertificateInKubernetes(req *RenewalRequest, cert *CertificateResponse) error {
	secretName := fmt.Sprintf("%s-%s-tls", e.config.KubernetesIntegration.SecretPrefix, req.ServiceName)

	secret, err := e.kubeClient.CoreV1().Secrets(req.Namespace).Get(e.ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get certificate secret: %w", err)
	}

	// Update secret data
	secret.Data["tls.crt"] = []byte(cert.CertificatePEM)
	secret.Data["tls.key"] = []byte(cert.PrivateKeyPEM)
	secret.Data["ca.crt"] = []byte(cert.CACertificatePEM)

	// Update annotations
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}
	secret.Annotations[fmt.Sprintf("%s/serial-number", e.config.KubernetesIntegration.AnnotationPrefix)] = cert.SerialNumber
	secret.Annotations[fmt.Sprintf("%s/expires-at", e.config.KubernetesIntegration.AnnotationPrefix)] = cert.ExpiresAt.Format(time.RFC3339)
	secret.Annotations[fmt.Sprintf("%s/renewed-at", e.config.KubernetesIntegration.AnnotationPrefix)] = time.Now().Format(time.RFC3339)

	_, err = e.kubeClient.CoreV1().Secrets(req.Namespace).Update(e.ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update certificate secret: %w", err)
	}

	return nil
}

// performCoordinatedRenewal performs a coordinated renewal with zero-downtime rotation
func (e *AutomationEngine) performCoordinatedRenewal(workerID int, req *RenewalRequest) {
	e.logger.Info("starting coordinated renewal",
		"worker_id", workerID,
		"serial_number", req.SerialNumber,
		"service", req.ServiceName,
		"namespace", req.Namespace)

	// Start coordinated rotation session
	session, err := e.rotationCoordinator.StartCoordinatedRotation(e.ctx, req)
	if err != nil {
		e.logger.Error("failed to start coordinated rotation",
			"worker_id", workerID,
			"serial_number", req.SerialNumber,
			"error", err)

		e.sendAlert("coordinated_renewal_failure", map[string]string{
			"serial_number": req.SerialNumber,
			"service":       req.ServiceName,
			"namespace":     req.Namespace,
			"error":         err.Error(),
		})
		return
	}

	// Define rotation function
	rotationFunc := func(instance *ServiceInstance) error {
		// Renew certificate for this instance
		newCert, err := e.caManager.RenewCertificate(e.ctx, req.SerialNumber)
		if err != nil {
			return fmt.Errorf("certificate renewal failed: %w", err)
		}

		// Update certificate in Kubernetes
		if e.config.KubernetesIntegration != nil && e.config.KubernetesIntegration.Enabled {
			if err := e.updateCertificateInKubernetes(req, newCert); err != nil {
				return fmt.Errorf("failed to update certificate in Kubernetes: %w", err)
			}
		}

		// Wait for service to pick up new certificate
		if req.HealthCheckConfig != nil && req.HealthCheckConfig.Enabled {
			// Perform health check to ensure service is healthy with new certificate
			target := &HealthCheckTarget{
				Name:    instance.Name,
				Address: strings.Split(instance.Endpoint, ":")[0],
				Port:    e.parsePort(instance.Endpoint),
			}

			healthSession, err := e.healthChecker.StartHealthCheck(target, req.HealthCheckConfig)
			if err != nil {
				return fmt.Errorf("failed to start health check: %w", err)
			}

			// Perform a few health checks to ensure stability
			checkDuration := 30 * time.Second
			if err := e.healthChecker.PerformContinuousHealthCheck(healthSession, checkDuration); err != nil {
				return fmt.Errorf("health check failed after certificate update: %w", err)
			}
		}

		return nil
	}

	// Execute coordinated rotation
	if err := e.rotationCoordinator.ExecuteCoordinatedRotation(e.ctx, session, rotationFunc); err != nil {
		e.logger.Error("coordinated rotation failed",
			"worker_id", workerID,
			"session_id", session.ID,
			"error", err)

		e.sendAlert("coordinated_renewal_failure", map[string]string{
			"session_id":    session.ID,
			"serial_number": req.SerialNumber,
			"service":       req.ServiceName,
			"namespace":     req.Namespace,
			"error":         err.Error(),
		})
		return
	}

	e.logger.Info("coordinated renewal completed successfully",
		"worker_id", workerID,
		"session_id", session.ID,
		"serial_number", req.SerialNumber,
		"service", req.ServiceName,
		"namespace", req.Namespace)
}

// parsePort parses port from endpoint string
func (e *AutomationEngine) parsePort(endpoint string) int {
	parts := strings.Split(endpoint, ":")
	if len(parts) != 2 {
		return 443 // Default HTTPS port
	}

	// Simple port parsing - could be enhanced
	switch parts[1] {
	case "80":
		return 80
	case "443":
		return 443
	case "8080":
		return 8080
	case "8443":
		return 8443
	default:
		return 443
	}
}

// RequestCoordinatedRenewal requests a coordinated certificate renewal
func (e *AutomationEngine) RequestCoordinatedRenewal(req *RenewalRequest) error {
	if !e.config.RenewalEnabled {
		return fmt.Errorf("certificate renewal is disabled")
	}

	req.CoordinatedRotation = true
	req.ZeroDowntime = true
	req.CreatedAt = time.Now()

	select {
	case e.renewalQueue <- req:
		e.logger.Info("coordinated renewal request queued",
			"serial_number", req.SerialNumber,
			"service", req.ServiceName,
			"namespace", req.Namespace)
		return nil
	default:
		return fmt.Errorf("renewal queue is full")
	}
}

// GetPerformanceStatistics returns performance cache statistics
func (e *AutomationEngine) GetPerformanceStatistics() *CacheStatistics {
	if e.performanceCache != nil {
		return e.performanceCache.GetStatistics()
	}
	return nil
}

// GetDiscoveredServices returns all discovered services
func (e *AutomationEngine) GetDiscoveredServices() map[string]*DiscoveredService {
	if e.serviceDiscovery != nil {
		return e.serviceDiscovery.GetDiscoveredServices()
	}
	return nil
}

// GetActiveRotationSessions returns all active rotation sessions
func (e *AutomationEngine) GetActiveRotationSessions() []*RotationSession {
	if e.rotationCoordinator == nil {
		return nil
	}

	var sessions []*RotationSession
	// This would get all active sessions from the coordinator
	return sessions
}

// GetActiveHealthCheckSessions returns all active health check sessions
func (e *AutomationEngine) GetActiveHealthCheckSessions() []*HealthCheckSession {
	if e.healthChecker != nil {
		return e.healthChecker.GetActiveHealthCheckSessions()
	}
	return nil
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
