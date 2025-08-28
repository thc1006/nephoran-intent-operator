// Package workload provides comprehensive Nephio workload cluster management.
package workload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterType represents the type of workload cluster.
type ClusterType string

const (
	// ClusterTypeManagement holds clustertypemanagement value.
	ClusterTypeManagement ClusterType = "management"
	// ClusterTypeWorkload holds clustertypeworkload value.
	ClusterTypeWorkload ClusterType = "workload"
	// ClusterTypeEdge holds clustertypeedge value.
	ClusterTypeEdge ClusterType = "edge"
	// ClusterTypeDevelopment holds clustertypedevelopment value.
	ClusterTypeDevelopment ClusterType = "development"
	// ClusterTypeProduction holds clustertypeproduction value.
	ClusterTypeProduction ClusterType = "production"
	// ClusterTypeSpecialized holds clustertypespecialized value.
	ClusterTypeSpecialized ClusterType = "specialized"
)

// CloudProvider represents the cloud provider for the cluster.
type CloudProvider string

const (
	// CloudProviderAWS holds cloudprovideraws value.
	CloudProviderAWS CloudProvider = "aws"
	// CloudProviderAzure holds cloudproviderazure value.
	CloudProviderAzure CloudProvider = "azure"
	// CloudProviderGCP holds cloudprovidergcp value.
	CloudProviderGCP CloudProvider = "gcp"
	// CloudProviderOnPremise holds cloudprovideronpremise value.
	CloudProviderOnPremise CloudProvider = "on-premise"
	// CloudProviderEdge holds cloudprovideredge value.
	CloudProviderEdge CloudProvider = "edge"
	// CloudProviderHybrid holds cloudproviderhybrid value.
	CloudProviderHybrid CloudProvider = "hybrid"
)

// ClusterCapability represents a capability of a cluster.
type ClusterCapability string

const (
	// CapabilityGPU holds capabilitygpu value.
	CapabilityGPU ClusterCapability = "gpu"
	// CapabilityFPGA holds capabilityfpga value.
	CapabilityFPGA ClusterCapability = "fpga"
	// CapabilityHighMemory holds capabilityhighmemory value.
	CapabilityHighMemory ClusterCapability = "high-memory"
	// CapabilityLowLatency holds capabilitylowlatency value.
	CapabilityLowLatency ClusterCapability = "low-latency"
	// CapabilityServiceMesh holds capabilityservicemesh value.
	CapabilityServiceMesh ClusterCapability = "service-mesh"
	// Capability5GCore holds capability5gcore value.
	Capability5GCore ClusterCapability = "5g-core"
	// CapabilityORANRIC holds capabilityoranric value.
	CapabilityORANRIC ClusterCapability = "oran-ric"
	// CapabilityNetworkSlice holds capabilitynetworkslice value.
	CapabilityNetworkSlice ClusterCapability = "network-slice"
)

// ClusterStatus represents the status of a cluster.
type ClusterStatus string

const (
	// ClusterStatusHealthy holds clusterstatushealthy value.
	ClusterStatusHealthy ClusterStatus = "healthy"
	// ClusterStatusDegraded holds clusterstatusdegraded value.
	ClusterStatusDegraded ClusterStatus = "degraded"
	// ClusterStatusUnhealthy holds clusterstatusunhealthy value.
	ClusterStatusUnhealthy ClusterStatus = "unhealthy"
	// ClusterStatusProvisioning holds clusterstatusprovisioning value.
	ClusterStatusProvisioning ClusterStatus = "provisioning"
	// ClusterStatusUpgrading holds clusterstatusupgrading value.
	ClusterStatusUpgrading ClusterStatus = "upgrading"
	// ClusterStatusMaintenance holds clusterstatusmaintenance value.
	ClusterStatusMaintenance ClusterStatus = "maintenance"
	// ClusterStatusUnknown holds clusterstatusunknown value.
	ClusterStatusUnknown ClusterStatus = "unknown"
)

// ClusterMetadata contains metadata about a cluster.
type ClusterMetadata struct {
	Name         string                `json:"name"`
	ID           string                `json:"id"`
	Type         ClusterType           `json:"type"`
	Provider     CloudProvider         `json:"provider"`
	Region       string                `json:"region"`
	Zone         string                `json:"zone"`
	Environment  string                `json:"environment"`
	Version      string                `json:"version"`
	Capabilities []ClusterCapability   `json:"capabilities"`
	Tags         map[string]string     `json:"tags"`
	Labels       map[string]string     `json:"labels"`
	Annotations  map[string]string     `json:"annotations"`
	Resources    ResourceCapacity      `json:"resources"`
	Network      NetworkConfiguration  `json:"network"`
	Security     SecurityConfiguration `json:"security"`
	Cost         CostMetadata          `json:"cost"`
	Compliance   ComplianceMetadata    `json:"compliance"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// ResourceCapacity represents the resource capacity of a cluster.
type ResourceCapacity struct {
	TotalCPU         int64   `json:"total_cpu"`
	AvailableCPU     int64   `json:"available_cpu"`
	TotalMemory      int64   `json:"total_memory"`
	AvailableMemory  int64   `json:"available_memory"`
	TotalStorage     int64   `json:"total_storage"`
	AvailableStorage int64   `json:"available_storage"`
	TotalGPU         int     `json:"total_gpu,omitempty"`
	AvailableGPU     int     `json:"available_gpu,omitempty"`
	NodeCount        int     `json:"node_count"`
	PodCapacity      int     `json:"pod_capacity"`
	AvailablePods    int     `json:"available_pods"`
	Utilization      float64 `json:"utilization"`
}

// NetworkConfiguration contains network configuration for a cluster.
type NetworkConfiguration struct {
	ServiceCIDR   string   `json:"service_cidr"`
	PodCIDR       string   `json:"pod_cidr"`
	DNSServers    []string `json:"dns_servers"`
	LoadBalancers []string `json:"load_balancers"`
	IngressClass  string   `json:"ingress_class"`
	ServiceMesh   string   `json:"service_mesh,omitempty"`
	VPNEndpoint   string   `json:"vpn_endpoint,omitempty"`
}

// SecurityConfiguration contains security configuration for a cluster.
type SecurityConfiguration struct {
	AuthProvider     string           `json:"auth_provider"`
	TLSEnabled       bool             `json:"tls_enabled"`
	NetworkPolicies  bool             `json:"network_policies"`
	PodSecurityLevel string           `json:"pod_security_level"`
	Encryption       EncryptionConfig `json:"encryption"`
	Compliance       []string         `json:"compliance_standards"`
	AuditLogging     bool             `json:"audit_logging"`
}

// EncryptionConfig contains encryption configuration.
type EncryptionConfig struct {
	AtRest    bool   `json:"at_rest"`
	InTransit bool   `json:"in_transit"`
	KMSKeyID  string `json:"kms_key_id,omitempty"`
}

// CostMetadata contains cost information for a cluster.
type CostMetadata struct {
	HourlyCost  float64           `json:"hourly_cost"`
	MonthlyCost float64           `json:"monthly_cost"`
	CostCenter  string            `json:"cost_center"`
	Budget      float64           `json:"budget"`
	BillingTags map[string]string `json:"billing_tags"`
}

// ComplianceMetadata contains compliance information.
type ComplianceMetadata struct {
	Standards       []string  `json:"standards"`
	Certifications  []string  `json:"certifications"`
	LastAudit       time.Time `json:"last_audit"`
	NextAudit       time.Time `json:"next_audit"`
	ComplianceScore float64   `json:"compliance_score"`
}

// ClusterEntry represents a registered cluster in the registry.
type ClusterEntry struct {
	Metadata     ClusterMetadata      `json:"metadata"`
	Status       ClusterStatus        `json:"status"`
	Health       ClusterHealth        `json:"health"`
	Credentials  ClusterCredentials   `json:"-"` // Never serialize credentials
	Config       *rest.Config         `json:"-"`
	Client       client.Client        `json:"-"`
	Clientset    kubernetes.Interface `json:"-"`
	LastSeen     time.Time            `json:"last_seen"`
	RegisteredAt time.Time            `json:"registered_at"`
}

// ClusterHealth represents the health status of a cluster.
type ClusterHealth struct {
	Status     ClusterStatus     `json:"status"`
	Message    string            `json:"message"`
	LastCheck  time.Time         `json:"last_check"`
	Components []ComponentHealth `json:"components"`
	Metrics    HealthMetrics     `json:"metrics"`
	Issues     []HealthIssue     `json:"issues"`
}

// ComponentHealth represents health of a cluster component.
type ComponentHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HealthMetrics contains health-related metrics.
type HealthMetrics struct {
	CPUUtilization    float64 `json:"cpu_utilization"`
	MemoryUtilization float64 `json:"memory_utilization"`
	DiskUtilization   float64 `json:"disk_utilization"`
	NetworkLatency    float64 `json:"network_latency"`
	ErrorRate         float64 `json:"error_rate"`
	RequestRate       float64 `json:"request_rate"`
}

// HealthIssue represents a health issue in a cluster.
type HealthIssue struct {
	Severity    string    `json:"severity"`
	Component   string    `json:"component"`
	Description string    `json:"description"`
	DetectedAt  time.Time `json:"detected_at"`
}

// ClusterCredentials contains credentials for accessing a cluster.
type ClusterCredentials struct {
	Kubeconfig   []byte    `json:"-"`
	Token        string    `json:"-"`
	Certificate  []byte    `json:"-"`
	Key          []byte    `json:"-"`
	CACert       []byte    `json:"-"`
	Endpoint     string    `json:"endpoint"`
	AuthType     string    `json:"auth_type"`
	SecretRef    string    `json:"secret_ref,omitempty"`
	RotationTime time.Time `json:"rotation_time"`
}

// ClusterQuery represents a query for filtering clusters.
type ClusterQuery struct {
	Types        []ClusterType       `json:"types,omitempty"`
	Providers    []CloudProvider     `json:"providers,omitempty"`
	Regions      []string            `json:"regions,omitempty"`
	Environments []string            `json:"environments,omitempty"`
	Capabilities []ClusterCapability `json:"capabilities,omitempty"`
	Tags         map[string]string   `json:"tags,omitempty"`
	Labels       labels.Selector     `json:"-"`
	MinResources *ResourceCapacity   `json:"min_resources,omitempty"`
	MaxCost      float64             `json:"max_cost,omitempty"`
	Status       []ClusterStatus     `json:"status,omitempty"`
}

// ClusterRegistry manages cluster registration and discovery.
type ClusterRegistry struct {
	mu               sync.RWMutex
	clusters         map[string]*ClusterEntry
	clustersByType   map[ClusterType][]*ClusterEntry
	clustersByRegion map[string][]*ClusterEntry
	discoveryConfig  DiscoveryConfig
	credentialStore  CredentialStore
	logger           logr.Logger
	metrics          *registryMetrics
	stopCh           chan struct{}
	client           client.Client
}

// DiscoveryConfig contains configuration for cluster discovery.
type DiscoveryConfig struct {
	AutoDiscovery     bool            `json:"auto_discovery"`
	DiscoveryInterval time.Duration   `json:"discovery_interval"`
	Namespaces        []string        `json:"namespaces"`
	LabelSelector     labels.Selector `json:"label_selector"`
	CloudProviders    []CloudProvider `json:"cloud_providers"`
	Regions           []string        `json:"regions"`
}

// CredentialStore interface for managing cluster credentials.
type CredentialStore interface {
	Store(ctx context.Context, clusterID string, creds ClusterCredentials) error
	Retrieve(ctx context.Context, clusterID string) (*ClusterCredentials, error)
	Rotate(ctx context.Context, clusterID string) (*ClusterCredentials, error)
	Delete(ctx context.Context, clusterID string) error
}

// registryMetrics contains Prometheus metrics for the registry.
type registryMetrics struct {
	clustersTotal       *prometheus.GaugeVec
	clustersByStatus    *prometheus.GaugeVec
	registrations       *prometheus.CounterVec
	discoveryDuration   *prometheus.HistogramVec
	credentialRotations *prometheus.CounterVec
}

// NewClusterRegistry creates a new cluster registry.
func NewClusterRegistry(
	client client.Client,
	logger logr.Logger,
	config DiscoveryConfig,
	credStore CredentialStore,
) *ClusterRegistry {
	metrics := &registryMetrics{
		clustersTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_cluster_registry_total",
				Help: "Total number of registered clusters",
			},
			[]string{"type", "provider", "region"},
		),
		clustersByStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_cluster_registry_status",
				Help: "Clusters by status",
			},
			[]string{"status"},
		),
		registrations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_cluster_registrations_total",
				Help: "Total number of cluster registrations",
			},
			[]string{"type", "result"},
		),
		discoveryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephio_cluster_discovery_duration_seconds",
				Help:    "Duration of cluster discovery operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"provider"},
		),
		credentialRotations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_cluster_credential_rotations_total",
				Help: "Total number of credential rotations",
			},
			[]string{"cluster", "result"},
		),
	}

	// Register metrics.
	prometheus.MustRegister(
		metrics.clustersTotal,
		metrics.clustersByStatus,
		metrics.registrations,
		metrics.discoveryDuration,
		metrics.credentialRotations,
	)

	return &ClusterRegistry{
		clusters:         make(map[string]*ClusterEntry),
		clustersByType:   make(map[ClusterType][]*ClusterEntry),
		clustersByRegion: make(map[string][]*ClusterEntry),
		discoveryConfig:  config,
		credentialStore:  credStore,
		logger:           logger.WithName("cluster-registry"),
		metrics:          metrics,
		stopCh:           make(chan struct{}),
		client:           client,
	}
}

// Start starts the cluster registry with automatic discovery.
func (r *ClusterRegistry) Start(ctx context.Context) error {
	r.logger.Info("Starting cluster registry")

	// Start automatic discovery if enabled.
	if r.discoveryConfig.AutoDiscovery {
		go r.runDiscovery(ctx)
	}

	// Start health monitoring.
	go r.runHealthMonitoring(ctx)

	// Start credential rotation.
	go r.runCredentialRotation(ctx)

	return nil
}

// Stop stops the cluster registry.
func (r *ClusterRegistry) Stop() {
	r.logger.Info("Stopping cluster registry")
	close(r.stopCh)
}

// RegisterCluster registers a new cluster.
func (r *ClusterRegistry) RegisterCluster(ctx context.Context, metadata ClusterMetadata, creds ClusterCredentials) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate cluster ID if not provided.
	if metadata.ID == "" {
		metadata.ID = r.generateClusterID(metadata)
	}

	r.logger.Info("Registering cluster", "id", metadata.ID, "name", metadata.Name)

	// Create REST config from credentials.
	config, err := r.createRestConfig(creds)
	if err != nil {
		r.metrics.registrations.WithLabelValues(string(metadata.Type), "failed").Inc()
		return fmt.Errorf("failed to create REST config: %w", err)
	}

	// Create Kubernetes clients.
	k8sClient, err := client.New(config, client.Options{})
	if err != nil {
		r.metrics.registrations.WithLabelValues(string(metadata.Type), "failed").Inc()
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		r.metrics.registrations.WithLabelValues(string(metadata.Type), "failed").Inc()
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Store credentials securely.
	if err := r.credentialStore.Store(ctx, metadata.ID, creds); err != nil {
		r.metrics.registrations.WithLabelValues(string(metadata.Type), "failed").Inc()
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Discover cluster capabilities.
	capabilities, err := r.discoverCapabilities(ctx, clientset)
	if err != nil {
		r.logger.Error(err, "Failed to discover capabilities", "cluster", metadata.ID)
	}
	metadata.Capabilities = append(metadata.Capabilities, capabilities...)

	// Get cluster resources.
	resources, err := r.getClusterResources(ctx, clientset)
	if err != nil {
		r.logger.Error(err, "Failed to get cluster resources", "cluster", metadata.ID)
	}
	metadata.Resources = resources

	// Create cluster entry.
	entry := &ClusterEntry{
		Metadata:     metadata,
		Status:       ClusterStatusHealthy,
		Health:       ClusterHealth{Status: ClusterStatusHealthy},
		Credentials:  creds,
		Config:       config,
		Client:       k8sClient,
		Clientset:    clientset,
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}

	// Add to registry.
	r.clusters[metadata.ID] = entry
	r.clustersByType[metadata.Type] = append(r.clustersByType[metadata.Type], entry)
	r.clustersByRegion[metadata.Region] = append(r.clustersByRegion[metadata.Region], entry)

	// Update metrics.
	r.metrics.clustersTotal.WithLabelValues(
		string(metadata.Type),
		string(metadata.Provider),
		metadata.Region,
	).Inc()
	r.metrics.registrations.WithLabelValues(string(metadata.Type), "success").Inc()

	r.logger.Info("Successfully registered cluster", "id", metadata.ID, "name", metadata.Name)
	return nil
}

// DeregisterCluster removes a cluster from the registry.
func (r *ClusterRegistry) DeregisterCluster(ctx context.Context, clusterID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	r.logger.Info("Deregistering cluster", "id", clusterID, "name", entry.Metadata.Name)

	// Remove from type index.
	r.removeFromTypeIndex(entry)

	// Remove from region index.
	r.removeFromRegionIndex(entry)

	// Delete credentials.
	if err := r.credentialStore.Delete(ctx, clusterID); err != nil {
		r.logger.Error(err, "Failed to delete credentials", "cluster", clusterID)
	}

	// Remove from main registry.
	delete(r.clusters, clusterID)

	// Update metrics.
	r.metrics.clustersTotal.WithLabelValues(
		string(entry.Metadata.Type),
		string(entry.Metadata.Provider),
		entry.Metadata.Region,
	).Dec()

	r.logger.Info("Successfully deregistered cluster", "id", clusterID)
	return nil
}

// GetCluster retrieves a cluster by ID.
func (r *ClusterRegistry) GetCluster(clusterID string) (*ClusterEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	return entry, nil
}

// ListClusters lists all registered clusters.
func (r *ClusterRegistry) ListClusters() []*ClusterEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	clusters := make([]*ClusterEntry, 0, len(r.clusters))
	for _, entry := range r.clusters {
		clusters = append(clusters, entry)
	}

	return clusters
}

// QueryClusters queries clusters based on criteria.
func (r *ClusterRegistry) QueryClusters(query ClusterQuery) []*ClusterEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*ClusterEntry

	for _, entry := range r.clusters {
		if r.matchesQuery(entry, query) {
			results = append(results, entry)
		}
	}

	return results
}

// UpdateClusterMetadata updates cluster metadata.
func (r *ClusterRegistry) UpdateClusterMetadata(ctx context.Context, clusterID string, metadata ClusterMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	// Update metadata.
	entry.Metadata = metadata
	entry.Metadata.UpdatedAt = time.Now()

	r.logger.Info("Updated cluster metadata", "id", clusterID)
	return nil
}

// UpdateClusterStatus updates cluster status.
func (r *ClusterRegistry) UpdateClusterStatus(clusterID string, status ClusterStatus, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.clusters[clusterID]
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	oldStatus := entry.Status
	entry.Status = status
	entry.Health.Status = status
	entry.Health.Message = message
	entry.Health.LastCheck = time.Now()
	entry.LastSeen = time.Now()

	// Update metrics.
	if oldStatus != status {
		r.metrics.clustersByStatus.WithLabelValues(string(oldStatus)).Dec()
		r.metrics.clustersByStatus.WithLabelValues(string(status)).Inc()
	}

	r.logger.Info("Updated cluster status", "id", clusterID, "status", status)
	return nil
}

// GetClustersByType retrieves clusters by type.
func (r *ClusterRegistry) GetClustersByType(clusterType ClusterType) []*ClusterEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.clustersByType[clusterType]
}

// GetClustersByRegion retrieves clusters by region.
func (r *ClusterRegistry) GetClustersByRegion(region string) []*ClusterEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.clustersByRegion[region]
}

// GetClustersByCapability retrieves clusters with specific capability.
func (r *ClusterRegistry) GetClustersByCapability(capability ClusterCapability) []*ClusterEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*ClusterEntry
	for _, entry := range r.clusters {
		for _, cap := range entry.Metadata.Capabilities {
			if cap == capability {
				results = append(results, entry)
				break
			}
		}
	}

	return results
}

// Private methods.

func (r *ClusterRegistry) runDiscovery(ctx context.Context) {
	ticker := time.NewTicker(r.discoveryConfig.DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.discoverClusters(ctx)
		}
	}
}

func (r *ClusterRegistry) discoverClusters(ctx context.Context) {
	r.logger.Info("Running cluster discovery")
	startTime := time.Now()

	// Discover clusters from cloud providers.
	for _, provider := range r.discoveryConfig.CloudProviders {
		timer := prometheus.NewTimer(r.metrics.discoveryDuration.WithLabelValues(string(provider)))
		r.discoverCloudClusters(ctx, provider)
		timer.ObserveDuration()
	}

	// Discover clusters from Kubernetes resources.
	r.discoverKubernetesClusters(ctx)

	r.logger.Info("Cluster discovery completed", "duration", time.Since(startTime))
}

func (r *ClusterRegistry) discoverCloudClusters(ctx context.Context, provider CloudProvider) {
	// Implementation would integrate with cloud provider APIs.
	// This is a placeholder for the actual implementation.
	r.logger.Info("Discovering clusters from cloud provider", "provider", provider)
}

func (r *ClusterRegistry) discoverKubernetesClusters(ctx context.Context) {
	// Discover clusters from Kubernetes secrets or ConfigMaps.
	secretList := &corev1.SecretList{}
	listOpts := []client.ListOption{
		client.MatchingLabels{"nephio.io/cluster": "true"},
	}

	if err := r.client.List(ctx, secretList, listOpts...); err != nil {
		r.logger.Error(err, "Failed to list cluster secrets")
		return
	}

	for _, secret := range secretList.Items {
		r.processClusterSecret(ctx, &secret)
	}
}

func (r *ClusterRegistry) processClusterSecret(ctx context.Context, secret *corev1.Secret) {
	// Extract cluster metadata from secret.
	metadataJSON, exists := secret.Data["metadata"]
	if !exists {
		r.logger.Error(nil, "Cluster secret missing metadata", "secret", secret.Name)
		return
	}

	var metadata ClusterMetadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		r.logger.Error(err, "Failed to unmarshal cluster metadata", "secret", secret.Name)
		return
	}

	// Extract credentials.
	kubeconfig, exists := secret.Data["kubeconfig"]
	if !exists {
		r.logger.Error(nil, "Cluster secret missing kubeconfig", "secret", secret.Name)
		return
	}

	creds := ClusterCredentials{
		Kubeconfig: kubeconfig,
		AuthType:   "kubeconfig",
		SecretRef:  fmt.Sprintf("%s/%s", secret.Namespace, secret.Name),
	}

	// Register the discovered cluster.
	if err := r.RegisterCluster(ctx, metadata, creds); err != nil {
		r.logger.Error(err, "Failed to register discovered cluster", "cluster", metadata.Name)
	}
}

func (r *ClusterRegistry) runHealthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.checkClusterHealth(ctx)
		}
	}
}

func (r *ClusterRegistry) checkClusterHealth(ctx context.Context) {
	r.mu.RLock()
	clusters := make([]*ClusterEntry, 0, len(r.clusters))
	for _, entry := range r.clusters {
		clusters = append(clusters, entry)
	}
	r.mu.RUnlock()

	for _, entry := range clusters {
		go r.checkSingleClusterHealth(ctx, entry)
	}
}

func (r *ClusterRegistry) checkSingleClusterHealth(ctx context.Context, entry *ClusterEntry) {
	health := ClusterHealth{
		LastCheck:  time.Now(),
		Components: []ComponentHealth{},
		Issues:     []HealthIssue{},
	}

	// Check API server health.
	_, err := entry.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		health.Status = ClusterStatusUnhealthy
		health.Message = fmt.Sprintf("API server unreachable: %v", err)
		health.Issues = append(health.Issues, HealthIssue{
			Severity:    "critical",
			Component:   "api-server",
			Description: err.Error(),
			DetectedAt:  time.Now(),
		})
	} else {
		health.Status = ClusterStatusHealthy
		health.Message = "Cluster is healthy"
	}

	// Check component health.
	health.Components = r.checkComponentHealth(ctx, entry)

	// Get health metrics.
	health.Metrics = r.getHealthMetrics(ctx, entry)

	// Update cluster health.
	r.mu.Lock()
	entry.Health = health
	entry.Status = health.Status
	entry.LastSeen = time.Now()
	r.mu.Unlock()

	// Update metrics.
	r.metrics.clustersByStatus.WithLabelValues(string(health.Status)).Inc()
}

func (r *ClusterRegistry) checkComponentHealth(ctx context.Context, entry *ClusterEntry) []ComponentHealth {
	components := []ComponentHealth{}

	// Check core components.
	coreComponents := []string{"kube-apiserver", "kube-controller-manager", "kube-scheduler", "etcd"}
	for _, comp := range coreComponents {
		podList, err := entry.Clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("component=%s", comp),
		})

		status := "healthy"
		message := "Component is running"
		if err != nil || len(podList.Items) == 0 {
			status = "unhealthy"
			message = "Component not found or not running"
		}

		components = append(components, ComponentHealth{
			Name:    comp,
			Status:  status,
			Message: message,
		})
	}

	return components
}

func (r *ClusterRegistry) getHealthMetrics(ctx context.Context, entry *ClusterEntry) HealthMetrics {
	// This would integrate with cluster monitoring systems.
	// Placeholder implementation.
	return HealthMetrics{
		CPUUtilization:    0.65,
		MemoryUtilization: 0.72,
		DiskUtilization:   0.55,
		NetworkLatency:    2.5,
		ErrorRate:         0.001,
		RequestRate:       1000,
	}
}

func (r *ClusterRegistry) runCredentialRotation(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.rotateCredentials(ctx)
		}
	}
}

func (r *ClusterRegistry) rotateCredentials(ctx context.Context) {
	r.mu.RLock()
	clusters := make([]*ClusterEntry, 0, len(r.clusters))
	for _, entry := range r.clusters {
		clusters = append(clusters, entry)
	}
	r.mu.RUnlock()

	for _, entry := range clusters {
		// Check if credentials need rotation (older than 30 days).
		if time.Since(entry.Credentials.RotationTime) > 30*24*time.Hour {
			r.rotateSingleClusterCredentials(ctx, entry)
		}
	}
}

func (r *ClusterRegistry) rotateSingleClusterCredentials(ctx context.Context, entry *ClusterEntry) {
	r.logger.Info("Rotating credentials", "cluster", entry.Metadata.ID)

	newCreds, err := r.credentialStore.Rotate(ctx, entry.Metadata.ID)
	if err != nil {
		r.logger.Error(err, "Failed to rotate credentials", "cluster", entry.Metadata.ID)
		r.metrics.credentialRotations.WithLabelValues(entry.Metadata.ID, "failed").Inc()
		return
	}

	// Create new REST config.
	config, err := r.createRestConfig(*newCreds)
	if err != nil {
		r.logger.Error(err, "Failed to create REST config with new credentials", "cluster", entry.Metadata.ID)
		r.metrics.credentialRotations.WithLabelValues(entry.Metadata.ID, "failed").Inc()
		return
	}

	// Update cluster entry.
	r.mu.Lock()
	entry.Credentials = *newCreds
	entry.Config = config
	r.mu.Unlock()

	r.metrics.credentialRotations.WithLabelValues(entry.Metadata.ID, "success").Inc()
	r.logger.Info("Successfully rotated credentials", "cluster", entry.Metadata.ID)
}

func (r *ClusterRegistry) createRestConfig(creds ClusterCredentials) (*rest.Config, error) {
	if len(creds.Kubeconfig) > 0 {
		return clientcmd.RESTConfigFromKubeConfig(creds.Kubeconfig)
	}

	return &rest.Config{
		Host:        creds.Endpoint,
		BearerToken: creds.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CertData: creds.Certificate,
			KeyData:  creds.Key,
			CAData:   creds.CACert,
		},
	}, nil
}

func (r *ClusterRegistry) generateClusterID(metadata ClusterMetadata) string {
	data := fmt.Sprintf("%s-%s-%s-%s", metadata.Name, metadata.Provider, metadata.Region, time.Now().String())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

func (r *ClusterRegistry) discoverCapabilities(ctx context.Context, clientset kubernetes.Interface) ([]ClusterCapability, error) {
	capabilities := []ClusterCapability{}

	// Check for GPU nodes.
	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return capabilities, err
	}

	for _, node := range nodeList.Items {
		if _, hasGPU := node.Status.Capacity["nvidia.com/gpu"]; hasGPU {
			capabilities = append(capabilities, CapabilityGPU)
			break
		}
	}

	// Check for service mesh.
	if _, err := clientset.AppsV1().Deployments("istio-system").Get(ctx, "istiod", metav1.GetOptions{}); err == nil {
		capabilities = append(capabilities, CapabilityServiceMesh)
	}

	// Check for 5G core components.
	if _, err := clientset.CoreV1().Namespaces().Get(ctx, "open5gs", metav1.GetOptions{}); err == nil {
		capabilities = append(capabilities, Capability5GCore)
	}

	return capabilities, nil
}

func (r *ClusterRegistry) getClusterResources(ctx context.Context, clientset kubernetes.Interface) (ResourceCapacity, error) {
	resources := ResourceCapacity{}

	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return resources, err
	}

	resources.NodeCount = len(nodeList.Items)

	for _, node := range nodeList.Items {
		// Add CPU and memory.
		if cpu, ok := node.Status.Capacity["cpu"]; ok {
			resources.TotalCPU += cpu.Value()
		}
		if memory, ok := node.Status.Capacity["memory"]; ok {
			resources.TotalMemory += memory.Value()
		}
		if storage, ok := node.Status.Capacity["ephemeral-storage"]; ok {
			resources.TotalStorage += storage.Value()
		}
		if pods, ok := node.Status.Capacity["pods"]; ok {
			resources.PodCapacity += int(pods.Value())
		}

		// Add allocatable resources.
		if cpu, ok := node.Status.Allocatable["cpu"]; ok {
			resources.AvailableCPU += cpu.Value()
		}
		if memory, ok := node.Status.Allocatable["memory"]; ok {
			resources.AvailableMemory += memory.Value()
		}
		if storage, ok := node.Status.Allocatable["ephemeral-storage"]; ok {
			resources.AvailableStorage += storage.Value()
		}
		if pods, ok := node.Status.Allocatable["pods"]; ok {
			resources.AvailablePods += int(pods.Value())
		}
	}

	// Calculate utilization.
	if resources.TotalCPU > 0 {
		resources.Utilization = float64(resources.TotalCPU-resources.AvailableCPU) / float64(resources.TotalCPU)
	}

	return resources, nil
}

func (r *ClusterRegistry) matchesQuery(entry *ClusterEntry, query ClusterQuery) bool {
	// Check type.
	if len(query.Types) > 0 {
		found := false
		for _, t := range query.Types {
			if entry.Metadata.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check provider.
	if len(query.Providers) > 0 {
		found := false
		for _, p := range query.Providers {
			if entry.Metadata.Provider == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check region.
	if len(query.Regions) > 0 {
		found := false
		for _, r := range query.Regions {
			if entry.Metadata.Region == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check environment.
	if len(query.Environments) > 0 {
		found := false
		for _, e := range query.Environments {
			if entry.Metadata.Environment == e {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check capabilities.
	if len(query.Capabilities) > 0 {
		for _, reqCap := range query.Capabilities {
			found := false
			for _, cap := range entry.Metadata.Capabilities {
				if cap == reqCap {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check tags.
	if len(query.Tags) > 0 {
		for k, v := range query.Tags {
			if tagVal, exists := entry.Metadata.Tags[k]; !exists || tagVal != v {
				return false
			}
		}
	}

	// Check labels.
	if query.Labels != nil {
		labelSet := labels.Set(entry.Metadata.Labels)
		if !query.Labels.Matches(labelSet) {
			return false
		}
	}

	// Check resources.
	if query.MinResources != nil {
		if entry.Metadata.Resources.AvailableCPU < query.MinResources.AvailableCPU ||
			entry.Metadata.Resources.AvailableMemory < query.MinResources.AvailableMemory ||
			entry.Metadata.Resources.AvailableStorage < query.MinResources.AvailableStorage {
			return false
		}
	}

	// Check cost.
	if query.MaxCost > 0 {
		if entry.Metadata.Cost.MonthlyCost > query.MaxCost {
			return false
		}
	}

	// Check status.
	if len(query.Status) > 0 {
		found := false
		for _, s := range query.Status {
			if entry.Status == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (r *ClusterRegistry) removeFromTypeIndex(entry *ClusterEntry) {
	clusters := r.clustersByType[entry.Metadata.Type]
	for i, c := range clusters {
		if c.Metadata.ID == entry.Metadata.ID {
			r.clustersByType[entry.Metadata.Type] = append(clusters[:i], clusters[i+1:]...)
			break
		}
	}
}

func (r *ClusterRegistry) removeFromRegionIndex(entry *ClusterEntry) {
	clusters := r.clustersByRegion[entry.Metadata.Region]
	for i, c := range clusters {
		if c.Metadata.ID == entry.Metadata.ID {
			r.clustersByRegion[entry.Metadata.Region] = append(clusters[:i], clusters[i+1:]...)
			break
		}
	}
}
