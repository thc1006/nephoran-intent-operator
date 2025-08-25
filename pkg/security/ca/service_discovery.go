package ca

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ServiceDiscovery handles automatic discovery and certificate provisioning for new services
type ServiceDiscovery struct {
	logger           *logging.StructuredLogger
	kubeClient       kubernetes.Interface
	config           *ServiceDiscoveryConfig
	automationEngine *AutomationEngine

	// Discovery state
	discoveredServices map[string]*DiscoveredService
	certTemplates      map[string]*CertificateTemplate
	mu                 sync.RWMutex

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ServiceDiscoveryConfig configures the service discovery
type ServiceDiscoveryConfig struct {
	Enabled                 bool                    `yaml:"enabled"`
	WatchNamespaces         []string                `yaml:"watch_namespaces"`
	ServiceAnnotationPrefix string                  `yaml:"service_annotation_prefix"`
	AutoProvisionEnabled    bool                    `yaml:"auto_provision_enabled"`
	TemplateMatching        *TemplateMatchingConfig `yaml:"template_matching"`
	PreProvisioningEnabled  bool                    `yaml:"pre_provisioning_enabled"`
	ServiceMeshIntegration  *BasicServiceMeshIntegration `yaml:"service_mesh_integration"`
}

// TemplateMatchingConfig configures certificate template matching
type TemplateMatchingConfig struct {
	Enabled          bool                    `yaml:"enabled"`
	DefaultTemplate  string                  `yaml:"default_template"`
	MatchingRules    []*TemplateMatchingRule `yaml:"matching_rules"`
	FallbackBehavior string                  `yaml:"fallback_behavior"`
}

// TemplateMatchingRule defines rules for matching services to certificate templates
type TemplateMatchingRule struct {
	Name                string            `yaml:"name"`
	Priority            int               `yaml:"priority"`
	LabelSelectors      map[string]string `yaml:"label_selectors"`
	AnnotationSelectors map[string]string `yaml:"annotation_selectors"`
	NamespacePattern    string            `yaml:"namespace_pattern"`
	ServicePattern      string            `yaml:"service_pattern"`
	Template            string            `yaml:"template"`
}

// ServiceMeshIntegration configures service mesh integration
type BasicServiceMeshIntegration struct {
	Enabled          bool   `yaml:"enabled"`
	MeshType         string `yaml:"mesh_type"` // "istio", "linkerd", "consul"
	MTLSEnabled      bool   `yaml:"mtls_enabled"`
	SidecarInjection bool   `yaml:"sidecar_injection"`
}

// DiscoveredService represents a discovered service
type DiscoveredService struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
	Ports           []ServicePort     `json:"ports"`
	Template        string            `json:"template"`
	CertProvisioned bool              `json:"cert_provisioned"`
	DiscoveredAt    time.Time         `json:"discovered_at"`
	LastProvisioned time.Time         `json:"last_provisioned,omitempty"`
	Status          DiscoveryStatus   `json:"status"`
}

// ServicePort represents a service port
type ServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	Protocol   string `json:"protocol"`
	TLSEnabled bool   `json:"tls_enabled"`
}

// DiscoveryStatus represents the status of service discovery
type DiscoveryStatus string

const (
	StatusDiscovered      DiscoveryStatus = "discovered"
	StatusProvisioning    DiscoveryStatus = "provisioning"
	StatusProvisioned     DiscoveryStatus = "provisioned"
	StatusProvisionFailed DiscoveryStatus = "provision_failed"
	StatusSkipped         DiscoveryStatus = "skipped"
)

// CertificateTemplate defines a certificate template for services
type CertificateTemplate struct {
	Name              string            `yaml:"name"`
	Description       string            `yaml:"description"`
	ValidityDuration  time.Duration     `yaml:"validity_duration"`
	KeyType           string            `yaml:"key_type"`
	KeySize           int               `yaml:"key_size"`
	DNSNamePatterns   []string          `yaml:"dns_name_patterns"`
	IPAddressPatterns []string          `yaml:"ip_address_patterns"`
	ExtendedKeyUsages []string          `yaml:"extended_key_usages"`
	Metadata          map[string]string `yaml:"metadata"`
	AutoRenew         bool              `yaml:"auto_renew"`
	RenewalThreshold  time.Duration     `yaml:"renewal_threshold"`
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(
	logger *logging.StructuredLogger,
	kubeClient kubernetes.Interface,
	config *ServiceDiscoveryConfig,
	automationEngine *AutomationEngine,
) *ServiceDiscovery {
	ctx, cancel := context.WithCancel(context.Background())

	return &ServiceDiscovery{
		logger:             logger,
		kubeClient:         kubeClient,
		config:             config,
		automationEngine:   automationEngine,
		discoveredServices: make(map[string]*DiscoveredService),
		certTemplates:      make(map[string]*CertificateTemplate),
		ctx:                ctx,
		cancel:             cancel,
	}
}

// Start starts the service discovery
func (sd *ServiceDiscovery) Start(ctx context.Context) error {
	if !sd.config.Enabled {
		sd.logger.Info("service discovery is disabled")
		return nil
	}

	sd.logger.Info("starting service discovery",
		"watch_namespaces", sd.config.WatchNamespaces,
		"auto_provision", sd.config.AutoProvisionEnabled)

	// Load certificate templates
	if err := sd.loadCertificateTemplates(); err != nil {
		return fmt.Errorf("failed to load certificate templates: %w", err)
	}

	// Start service watchers
	for _, namespace := range sd.config.WatchNamespaces {
		sd.wg.Add(1)
		go sd.watchServices(namespace)
	}

	// Start pre-provisioning if enabled
	if sd.config.PreProvisioningEnabled {
		sd.wg.Add(1)
		go sd.runPreProvisioning()
	}

	// Start discovery cleanup
	sd.wg.Add(1)
	go sd.runDiscoveryCleanup()

	// Wait for context cancellation
	<-ctx.Done()
	sd.cancel()
	sd.wg.Wait()

	return nil
}

// Stop stops the service discovery
func (sd *ServiceDiscovery) Stop() {
	sd.logger.Info("stopping service discovery")
	sd.cancel()
	sd.wg.Wait()
}

// watchServices watches services in a specific namespace
func (sd *ServiceDiscovery) watchServices(namespace string) {
	defer sd.wg.Done()

	sd.logger.Info("starting service watcher", "namespace", namespace)

	// Create list watcher for services
	watchlist := cache.NewListWatchFromClient(
		sd.kubeClient.CoreV1().RESTClient(),
		"services",
		namespace,
		fields.Everything(),
	)

	// Create informer
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Service{},
		time.Minute*5,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if service, ok := obj.(*v1.Service); ok {
					sd.handleServiceAdded(service)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if service, ok := newObj.(*v1.Service); ok {
					sd.handleServiceUpdated(service)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if service, ok := obj.(*v1.Service); ok {
					sd.handleServiceDeleted(service)
				}
			},
		},
	)

	// Run controller
	go controller.Run(sd.ctx.Done())
}

// handleServiceAdded handles new service discovery
func (sd *ServiceDiscovery) handleServiceAdded(service *v1.Service) {
	sd.logger.Debug("service added",
		"service", service.Name,
		"namespace", service.Namespace)

	// Check if service should be managed
	if !sd.shouldManageService(service) {
		return
	}

	// Discover service
	discovered := sd.discoverService(service)
	if discovered == nil {
		return
	}

	// Store discovered service
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	sd.mu.Lock()
	sd.discoveredServices[serviceKey] = discovered
	sd.mu.Unlock()

	sd.logger.Info("service discovered",
		"service", service.Name,
		"namespace", service.Namespace,
		"template", discovered.Template,
		"tls_ports", sd.countTLSPorts(discovered.Ports))

	// Auto-provision if enabled
	if sd.config.AutoProvisionEnabled {
		go sd.provisionCertificateForService(discovered)
	}
}

// handleServiceUpdated handles service updates
func (sd *ServiceDiscovery) handleServiceUpdated(service *v1.Service) {
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)

	sd.mu.Lock()
	existing, exists := sd.discoveredServices[serviceKey]
	sd.mu.Unlock()

	if !exists {
		// Treat as new service
		sd.handleServiceAdded(service)
		return
	}

	// Check if service configuration changed
	discovered := sd.discoverService(service)
	if discovered == nil {
		return
	}

	// Update discovered service
	sd.mu.Lock()
	sd.discoveredServices[serviceKey] = discovered
	sd.mu.Unlock()

	// Check if certificate needs to be reprovisioned
	if sd.shouldReprovisionCertificate(existing, discovered) {
		sd.logger.Info("service configuration changed, reprovisioning certificate",
			"service", service.Name,
			"namespace", service.Namespace)

		go sd.provisionCertificateForService(discovered)
	}
}

// handleServiceDeleted handles service deletion
func (sd *ServiceDiscovery) handleServiceDeleted(service *v1.Service) {
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)

	sd.mu.Lock()
	discovered, exists := sd.discoveredServices[serviceKey]
	delete(sd.discoveredServices, serviceKey)
	sd.mu.Unlock()

	if exists {
		sd.logger.Info("service deleted",
			"service", service.Name,
			"namespace", service.Namespace,
			"was_provisioned", discovered.CertProvisioned)

		// Optionally revoke certificate
		if discovered.CertProvisioned {
			go sd.revokeCertificateForService(discovered)
		}
	}
}

// shouldManageService determines if a service should be managed
func (sd *ServiceDiscovery) shouldManageService(service *v1.Service) bool {
	// Skip system services
	if service.Namespace == "kube-system" ||
		service.Namespace == "kube-public" ||
		strings.HasPrefix(service.Name, "kubernetes") {
		return false
	}

	// Check for explicit opt-out annotation
	optOutKey := fmt.Sprintf("%s/manage", sd.config.ServiceAnnotationPrefix)
	if value, exists := service.Annotations[optOutKey]; exists && value == "false" {
		return false
	}

	// Check for TLS ports or explicit opt-in
	hasTLSPorts := sd.servicePorts(service).HasTLSPorts()
	optInKey := fmt.Sprintf("%s/auto-certificate", sd.config.ServiceAnnotationPrefix)
	hasOptIn := false
	if value, exists := service.Annotations[optInKey]; exists && value == "true" {
		hasOptIn = true
	}

	return hasTLSPorts || hasOptIn
}

// discoverService creates a discovered service from a Kubernetes service
func (sd *ServiceDiscovery) discoverService(service *v1.Service) *DiscoveredService {
	ports := sd.servicePorts(service)
	template := sd.selectCertificateTemplate(service)

	if template == "" {
		sd.logger.Debug("no suitable certificate template found",
			"service", service.Name,
			"namespace", service.Namespace)
		return nil
	}

	return &DiscoveredService{
		Name:         service.Name,
		Namespace:    service.Namespace,
		Labels:       service.Labels,
		Annotations:  service.Annotations,
		Ports:        ports.ToServicePorts(),
		Template:     template,
		DiscoveredAt: time.Now(),
		Status:       StatusDiscovered,
	}
}

// selectCertificateTemplate selects the appropriate certificate template
func (sd *ServiceDiscovery) selectCertificateTemplate(service *v1.Service) string {
	if !sd.config.TemplateMatching.Enabled {
		return sd.config.TemplateMatching.DefaultTemplate
	}

	// Check for explicit template annotation
	templateKey := fmt.Sprintf("%s/certificate-template", sd.config.ServiceAnnotationPrefix)
	if template, exists := service.Annotations[templateKey]; exists {
		if _, templateExists := sd.certTemplates[template]; templateExists {
			return template
		}
	}

	// Apply matching rules
	var bestMatch *TemplateMatchingRule
	var bestPriority int = -1

	for _, rule := range sd.config.TemplateMatching.MatchingRules {
		if sd.matchesRule(service, rule) && rule.Priority > bestPriority {
			bestMatch = rule
			bestPriority = rule.Priority
		}
	}

	if bestMatch != nil {
		return bestMatch.Template
	}

	// Fallback behavior
	switch sd.config.TemplateMatching.FallbackBehavior {
	case "default":
		return sd.config.TemplateMatching.DefaultTemplate
	case "skip":
		return ""
	default:
		return sd.config.TemplateMatching.DefaultTemplate
	}
}

// matchesRule checks if a service matches a template matching rule
func (sd *ServiceDiscovery) matchesRule(service *v1.Service, rule *TemplateMatchingRule) bool {
	// Check namespace pattern
	if rule.NamespacePattern != "" && !sd.matchesPattern(service.Namespace, rule.NamespacePattern) {
		return false
	}

	// Check service pattern
	if rule.ServicePattern != "" && !sd.matchesPattern(service.Name, rule.ServicePattern) {
		return false
	}

	// Check label selectors
	for labelKey, labelValue := range rule.LabelSelectors {
		if serviceValue, exists := service.Labels[labelKey]; !exists || serviceValue != labelValue {
			return false
		}
	}

	// Check annotation selectors
	for annotationKey, annotationValue := range rule.AnnotationSelectors {
		if serviceValue, exists := service.Annotations[annotationKey]; !exists || serviceValue != annotationValue {
			return false
		}
	}

	return true
}

// matchesPattern checks if a string matches a pattern (basic glob support)
func (sd *ServiceDiscovery) matchesPattern(value, pattern string) bool {
	// Simple pattern matching - could be enhanced with regex or glob library
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(value, suffix)
	}
	return value == pattern
}

// provisionCertificateForService provisions certificate for a discovered service
func (sd *ServiceDiscovery) provisionCertificateForService(discovered *DiscoveredService) {
	sd.logger.Info("provisioning certificate for service",
		"service", discovered.Name,
		"namespace", discovered.Namespace,
		"template", discovered.Template)

	// Update status
	discovered.Status = StatusProvisioning

	// Get certificate template
	template, exists := sd.certTemplates[discovered.Template]
	if !exists {
		sd.logger.Error("certificate template not found",
			"service", discovered.Name,
			"template", discovered.Template)
		discovered.Status = StatusProvisionFailed
		return
	}

	// Generate DNS names
	dnsNames := sd.generateDNSNames(discovered, template)

	// Create provisioning request
	req := &ProvisioningRequest{
		ID:          fmt.Sprintf("discovery-%s-%s-%d", discovered.Namespace, discovered.Name, time.Now().Unix()),
		ServiceName: discovered.Name,
		Namespace:   discovered.Namespace,
		Template:    discovered.Template,
		DNSNames:    dnsNames,
		Priority:    PriorityNormal,
		Metadata: map[string]string{
			"discovered_by":       "service-discovery",
			"discovery_timestamp": discovered.DiscoveredAt.Format(time.RFC3339),
			"template_used":       discovered.Template,
		},
	}

	// Submit provisioning request
	if err := sd.automationEngine.RequestProvisioning(req); err != nil {
		sd.logger.Error("failed to request certificate provisioning",
			"service", discovered.Name,
			"namespace", discovered.Namespace,
			"error", err)
		discovered.Status = StatusProvisionFailed
		return
	}

	// Update service state
	discovered.CertProvisioned = true
	discovered.LastProvisioned = time.Now()
	discovered.Status = StatusProvisioned

	sd.logger.Info("certificate provisioning requested",
		"service", discovered.Name,
		"namespace", discovered.Namespace,
		"request_id", req.ID)
}

// generateDNSNames generates DNS names for the certificate
func (sd *ServiceDiscovery) generateDNSNames(discovered *DiscoveredService, template *CertificateTemplate) []string {
	var dnsNames []string

	// Add standard Kubernetes DNS names
	dnsNames = append(dnsNames,
		fmt.Sprintf("%s.%s.svc.cluster.local", discovered.Name, discovered.Namespace),
		fmt.Sprintf("%s.%s.svc", discovered.Name, discovered.Namespace),
		fmt.Sprintf("%s.%s", discovered.Name, discovered.Namespace),
		discovered.Name,
	)

	// Add template-specific DNS patterns
	for _, pattern := range template.DNSNamePatterns {
		dnsName := strings.ReplaceAll(pattern, "{service}", discovered.Name)
		dnsName = strings.ReplaceAll(dnsName, "{namespace}", discovered.Namespace)
		dnsNames = append(dnsNames, dnsName)
	}

	// Add DNS names from service annotations
	dnsAnnotationKey := fmt.Sprintf("%s/additional-dns", sd.config.ServiceAnnotationPrefix)
	if additionalDNS, exists := discovered.Annotations[dnsAnnotationKey]; exists {
		for _, dns := range strings.Split(additionalDNS, ",") {
			dnsNames = append(dnsNames, strings.TrimSpace(dns))
		}
	}

	return dnsNames
}

// shouldReprovisionCertificate determines if certificate should be reprovisioned
func (sd *ServiceDiscovery) shouldReprovisionCertificate(existing, updated *DiscoveredService) bool {
	// Check if template changed
	if existing.Template != updated.Template {
		return true
	}

	// Check if ports changed
	if len(existing.Ports) != len(updated.Ports) {
		return true
	}

	// Check if TLS configuration changed
	existingTLSPorts := sd.countTLSPorts(existing.Ports)
	updatedTLSPorts := sd.countTLSPorts(updated.Ports)
	if existingTLSPorts != updatedTLSPorts {
		return true
	}

	// Check if DNS annotations changed
	dnsKey := fmt.Sprintf("%s/additional-dns", sd.config.ServiceAnnotationPrefix)
	existingDNS := existing.Annotations[dnsKey]
	updatedDNS := updated.Annotations[dnsKey]
	if existingDNS != updatedDNS {
		return true
	}

	return false
}

// revokeCertificateForService revokes certificate for a deleted service
func (sd *ServiceDiscovery) revokeCertificateForService(discovered *DiscoveredService) {
	sd.logger.Info("revoking certificate for deleted service",
		"service", discovered.Name,
		"namespace", discovered.Namespace)

	// This would integrate with the CA manager to revoke certificates
	// Implementation depends on the specific CA backend
}

// runPreProvisioning handles pre-provisioning of certificates
func (sd *ServiceDiscovery) runPreProvisioning() {
	defer sd.wg.Done()

	sd.logger.Info("starting pre-provisioning service")

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sd.ctx.Done():
			return
		case <-ticker.C:
			sd.performPreProvisioning()
		}
	}
}

// performPreProvisioning performs pre-provisioning check
func (sd *ServiceDiscovery) performPreProvisioning() {
	sd.logger.Debug("performing pre-provisioning check")

	sd.mu.RLock()
	services := make([]*DiscoveredService, 0, len(sd.discoveredServices))
	for _, service := range sd.discoveredServices {
		services = append(services, service)
	}
	sd.mu.RUnlock()

	for _, service := range services {
		if !service.CertProvisioned && service.Status == StatusDiscovered {
			// Check if service is stable (discovered more than 5 minutes ago)
			if time.Since(service.DiscoveredAt) > 5*time.Minute {
				go sd.provisionCertificateForService(service)
			}
		}
	}
}

// runDiscoveryCleanup handles cleanup of old discovery data
func (sd *ServiceDiscovery) runDiscoveryCleanup() {
	defer sd.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-sd.ctx.Done():
			return
		case <-ticker.C:
			sd.performDiscoveryCleanup()
		}
	}
}

// performDiscoveryCleanup cleans up old discovery data
func (sd *ServiceDiscovery) performDiscoveryCleanup() {
	sd.logger.Debug("performing discovery cleanup")

	cutoff := time.Now().Add(-24 * time.Hour) // Keep data for 24 hours

	sd.mu.Lock()
	defer sd.mu.Unlock()

	for serviceKey, service := range sd.discoveredServices {
		if service.Status == StatusProvisionFailed && service.DiscoveredAt.Before(cutoff) {
			delete(sd.discoveredServices, serviceKey)
			sd.logger.Debug("cleaned up failed discovery entry",
				"service", service.Name,
				"namespace", service.Namespace)
		}
	}
}

// loadCertificateTemplates loads certificate templates from configuration
func (sd *ServiceDiscovery) loadCertificateTemplates() error {
	// Load default templates
	sd.certTemplates["default"] = &CertificateTemplate{
		Name:             "default",
		Description:      "Default certificate template for services",
		ValidityDuration: 90 * 24 * time.Hour, // 90 days
		KeyType:          "RSA",
		KeySize:          2048,
		DNSNamePatterns:  []string{},
		AutoRenew:        true,
		RenewalThreshold: 30 * 24 * time.Hour, // 30 days
	}

	sd.certTemplates["microservice"] = &CertificateTemplate{
		Name:              "microservice",
		Description:       "Certificate template for microservices",
		ValidityDuration:  30 * 24 * time.Hour, // 30 days
		KeyType:           "ECDSA",
		KeySize:           256,
		DNSNamePatterns:   []string{},
		ExtendedKeyUsages: []string{"ServerAuth", "ClientAuth"},
		AutoRenew:         true,
		RenewalThreshold:  7 * 24 * time.Hour, // 7 days
	}

	sd.certTemplates["public-facing"] = &CertificateTemplate{
		Name:              "public-facing",
		Description:       "Certificate template for public-facing services",
		ValidityDuration:  365 * 24 * time.Hour, // 1 year
		KeyType:           "RSA",
		KeySize:           4096,
		DNSNamePatterns:   []string{},
		ExtendedKeyUsages: []string{"ServerAuth"},
		AutoRenew:         true,
		RenewalThreshold:  60 * 24 * time.Hour, // 60 days
	}

	sd.logger.Info("loaded certificate templates",
		"template_count", len(sd.certTemplates))

	return nil
}

// GetDiscoveredServices returns all discovered services
func (sd *ServiceDiscovery) GetDiscoveredServices() map[string]*DiscoveredService {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]*DiscoveredService)
	for k, v := range sd.discoveredServices {
		result[k] = v
	}

	return result
}

// Helper types and methods

type servicePorts []v1.ServicePort

func (sd *ServiceDiscovery) servicePorts(service *v1.Service) servicePorts {
	return servicePorts(service.Spec.Ports)
}

func (sp servicePorts) HasTLSPorts() bool {
	for _, port := range sp {
		if sp.isTLSPort(port) {
			return true
		}
	}
	return false
}

func (sp servicePorts) isTLSPort(port v1.ServicePort) bool {
	return port.Name == "https" ||
		port.Name == "tls" ||
		port.Port == 443 ||
		port.Port == 8443 ||
		strings.Contains(port.Name, "tls") ||
		strings.Contains(port.Name, "ssl")
}

func (sp servicePorts) ToServicePorts() []ServicePort {
	var result []ServicePort
	for _, port := range sp {
		result = append(result, ServicePort{
			Name:       port.Name,
			Port:       port.Port,
			Protocol:   string(port.Protocol),
			TLSEnabled: sp.isTLSPort(port),
		})
	}
	return result
}

func (sd *ServiceDiscovery) countTLSPorts(ports []ServicePort) int {
	count := 0
	for _, port := range ports {
		if port.TLSEnabled {
			count++
		}
	}
	return count
}
