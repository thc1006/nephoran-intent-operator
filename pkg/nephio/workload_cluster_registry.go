/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package nephio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WorkloadClusterConfig defines configuration for workload cluster management.

type WorkloadClusterConfig struct {
	HealthCheckInterval time.Duration `json:"healthCheckInterval" yaml:"healthCheckInterval"`

	AutoClusterRegistration bool `json:"autoClusterRegistration" yaml:"autoClusterRegistration"`

	RegistrationTimeout time.Duration `json:"registrationTimeout" yaml:"registrationTimeout"`

	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`

	RetryBackoff time.Duration `json:"retryBackoff" yaml:"retryBackoff"`

	EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics"`

	EnableTracing bool `json:"enableTracing" yaml:"enableTracing"`
}

// ClusterRegistryMetrics provides cluster registry metrics.

type ClusterRegistryMetrics struct {
	ClusterRegistrations *prometheus.CounterVec

	ClusterHealth *prometheus.GaugeVec

	HealthCheckDuration *prometheus.HistogramVec

	ClusterCapabilities *prometheus.GaugeVec

	RegistrationErrors *prometheus.CounterVec

	ClusterEvents *prometheus.CounterVec
}

// ClusterHealth represents the health status of a workload cluster.

type ClusterHealth struct {
	Status string `json:"status"`

	LastChecked time.Time `json:"lastChecked"`

	Components map[string]ComponentHealth `json:"components"`

	Resources *ResourceHealthStatus `json:"resources,omitempty"`

	Connectivity *ConnectivityStatus `json:"connectivity,omitempty"`

	Version string `json:"version,omitempty"`

	Uptime time.Duration `json:"uptime"`

	ErrorCount int `json:"errorCount"`

	WarningCount int `json:"warningCount"`

	LastError string `json:"lastError,omitempty"`
}

// ComponentHealth represents health of a cluster component.

type ComponentHealth struct {
	Name string `json:"name"`

	Status string `json:"status"`

	Message string `json:"message,omitempty"`

	LastSeen time.Time `json:"lastSeen"`
}

// ResourceHealthStatus represents cluster resource health.

type ResourceHealthStatus struct {
	CPUUsage float64 `json:"cpuUsage"`

	MemoryUsage float64 `json:"memoryUsage"`

	StorageUsage float64 `json:"storageUsage"`

	NetworkUsage float64 `json:"networkUsage"`

	PodCount int `json:"podCount"`

	NodeCount int `json:"nodeCount"`
}

// ConnectivityStatus represents cluster connectivity status.

type ConnectivityStatus struct {
	APIServer bool `json:"apiServer"`

	Etcd bool `json:"etcd"`

	DNS bool `json:"dns"`

	NetworkPolicy bool `json:"networkPolicy"`

	LastChecked time.Time `json:"lastChecked"`

	Latency time.Duration `json:"latency"`
}

// ClusterConfigSync represents Config Sync configuration for a cluster.

type ClusterConfigSync struct {
	Enabled bool `json:"enabled"`

	Repository string `json:"repository"`

	Branch string `json:"branch"`

	Directory string `json:"directory"`

	SyncPeriod time.Duration `json:"syncPeriod"`

	Status string `json:"status"`

	LastSync *time.Time `json:"lastSync,omitempty"`

	Errors []string `json:"errors,omitempty"`

	Credentials *GitCredentials `json:"credentials,omitempty"`
}

// GitCredentials represents Git repository credentials.

type GitCredentials struct {
	Type string `json:"type"`

	Username string `json:"username,omitempty"`

	Password string `json:"password,omitempty"`

	Token string `json:"token,omitempty"`

	SSHKey string `json:"sshKey,omitempty"`

	SecretName string `json:"secretName,omitempty"`
}

// ClusterHealthMonitor provides cluster health monitoring.

type ClusterHealthMonitor struct {
	client client.Client

	registry *WorkloadClusterRegistry

	config *WorkloadClusterConfig

	metrics *ClusterRegistryMetrics

	tracer trace.Tracer

	stopCh chan struct{}

	mu sync.RWMutex
}

// WorkloadDeployment represents a deployment to a workload cluster.

type WorkloadDeployment struct {
	ID string `json:"id"`

	PackageVariant *PackageVariant `json:"packageVariant"`

	TargetCluster *WorkloadCluster `json:"targetCluster"`

	Status string `json:"status"`

	StartedAt time.Time `json:"startedAt"`

	CompletedAt *time.Time `json:"completedAt,omitempty"`

	SyncResult *SyncResult `json:"syncResult,omitempty"`

	Error string `json:"error,omitempty"`

	Retries int `json:"retries"`

	LastRetry *time.Time `json:"lastRetry,omitempty"`
}

// DefaultWorkloadClusterConfig provides default configuration values.

var DefaultWorkloadClusterConfig = &WorkloadClusterConfig{

	HealthCheckInterval: 1 * time.Minute,

	AutoClusterRegistration: true,

	RegistrationTimeout: 5 * time.Minute,

	MaxRetries: 3,

	RetryBackoff: 30 * time.Second,

	EnableMetrics: true,

	EnableTracing: true,
}

// NewWorkloadClusterRegistry creates a new workload cluster registry.

func NewWorkloadClusterRegistry(

	client client.Client,

	config *WorkloadClusterConfig,

) (*WorkloadClusterRegistry, error) {

	if config == nil {

		config = DefaultWorkloadClusterConfig

	}

	// Initialize metrics.

	metrics := &ClusterRegistryMetrics{

		ClusterRegistrations: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephio_cluster_registrations_total",

				Help: "Total number of workload cluster registrations",
			},

			[]string{"cluster", "region", "status"},
		),

		ClusterHealth: promauto.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "nephio_cluster_health",

				Help: "Health status of workload clusters",
			},

			[]string{"cluster", "region", "component"},
		),

		HealthCheckDuration: promauto.NewHistogramVec(

			prometheus.HistogramOpts{

				Name: "nephio_cluster_health_check_duration_seconds",

				Help: "Duration of cluster health checks",

				Buckets: prometheus.ExponentialBuckets(0.1, 2, 8),
			},

			[]string{"cluster", "region"},
		),

		ClusterCapabilities: promauto.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "nephio_cluster_capabilities",

				Help: "Number of capabilities per cluster",
			},

			[]string{"cluster", "region", "capability"},
		),

		RegistrationErrors: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephio_cluster_registration_errors_total",

				Help: "Total number of cluster registration errors",
			},

			[]string{"cluster", "error_type"},
		),

		ClusterEvents: promauto.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephio_cluster_events_total",

				Help: "Total number of cluster events",
			},

			[]string{"cluster", "event_type"},
		),
	}

	registry := &WorkloadClusterRegistry{

		client: client,

		config: config,

		metrics: metrics,

		tracer: otel.Tracer("nephio-workload-registry"),
	}

	// Initialize health monitor.

	registry.healthMonitor = &ClusterHealthMonitor{

		client: client,

		registry: registry,

		config: config,

		metrics: metrics,

		tracer: otel.Tracer("nephio-cluster-health"),

		stopCh: make(chan struct{}),
	}

	// Start health monitoring if enabled.

	if config.HealthCheckInterval > 0 {

		go registry.healthMonitor.Start()

	}

	// Initialize with standard clusters if auto-registration is enabled.

	if config.AutoClusterRegistration {

		go registry.discoverAndRegisterClusters(context.Background())

	}

	return registry, nil

}

// RegisterWorkloadCluster registers a new workload cluster.

func (wcr *WorkloadClusterRegistry) RegisterWorkloadCluster(ctx context.Context, cluster *WorkloadCluster) error {

	ctx, span := wcr.tracer.Start(ctx, "register-workload-cluster")

	defer span.End()

	logger := log.FromContext(ctx).WithName("cluster-registry").WithValues(

		"cluster", cluster.Name,

		"region", cluster.Region,

		"zone", cluster.Zone,
	)

	span.SetAttributes(

		attribute.String("cluster.name", cluster.Name),

		attribute.String("cluster.region", cluster.Region),

		attribute.String("cluster.zone", cluster.Zone),
	)

	startTime := time.Now()

	// Validate cluster.

	if err := wcr.validateCluster(ctx, cluster); err != nil {

		span.RecordError(err)

		wcr.metrics.RegistrationErrors.WithLabelValues(

			cluster.Name, "validation_failed",
		).Inc()

		return fmt.Errorf("cluster validation failed: %w", err)

	}

	// Check if cluster already exists.

	if _, exists := wcr.clusters.Load(cluster.Name); exists {

		wcr.metrics.RegistrationErrors.WithLabelValues(

			cluster.Name, "already_exists",
		).Inc()

		return fmt.Errorf("cluster %s already registered", cluster.Name)

	}

	// Set initial status.

	cluster.Status = WorkloadClusterStatusRegistering

	cluster.CreatedAt = time.Now()

	// Perform cluster connectivity check.

	if err := wcr.performConnectivityCheck(ctx, cluster); err != nil {

		logger.Error(err, "Cluster connectivity check failed")

		cluster.Status = WorkloadClusterStatusUnreachable

		wcr.metrics.RegistrationErrors.WithLabelValues(

			cluster.Name, "connectivity_failed",
		).Inc()

		// Continue with registration but mark as unreachable.

	} else {

		cluster.Status = WorkloadClusterStatusActive

	}

	// Initialize health monitoring.

	cluster.Health = &ClusterHealth{

		Status: "Unknown",

		LastChecked: time.Now(),

		Components: make(map[string]ComponentHealth),

		ErrorCount: 0,

		WarningCount: 0,
	}

	// Setup Config Sync if enabled.

	if err := wcr.setupConfigSync(ctx, cluster); err != nil {

		logger.Error(err, "Failed to setup Config Sync")

		// Continue with registration - Config Sync is not critical for registration.

	}

	// Store cluster in registry.

	wcr.clusters.Store(cluster.Name, cluster)

	wcr.registrations.Store(cluster.Name, &ClusterRegistration{

		Cluster: cluster,

		RegisteredAt: time.Now(),

		Status: "active",
	})

	// Update metrics.

	wcr.metrics.ClusterRegistrations.WithLabelValues(

		cluster.Name, cluster.Region, "success",
	).Inc()

	// Update capability metrics.

	for _, capability := range cluster.Capabilities {

		wcr.metrics.ClusterCapabilities.WithLabelValues(

			cluster.Name, cluster.Region, capability.Name,
		).Inc()

	}

	duration := time.Since(startTime)

	logger.Info("Cluster registered successfully",

		"duration", duration,

		"status", cluster.Status,

		"capabilities", len(cluster.Capabilities),
	)

	// Emit cluster event.

	wcr.metrics.ClusterEvents.WithLabelValues(

		cluster.Name, "registered",
	).Inc()

	span.SetAttributes(

		attribute.String("cluster.status", string(cluster.Status)),

		attribute.Int("cluster.capabilities", len(cluster.Capabilities)),

		attribute.Float64("registration.duration", duration.Seconds()),
	)

	return nil

}

// UnregisterWorkloadCluster removes a workload cluster from the registry.

func (wcr *WorkloadClusterRegistry) UnregisterWorkloadCluster(ctx context.Context, clusterName string) error {

	ctx, span := wcr.tracer.Start(ctx, "unregister-workload-cluster")

	defer span.End()

	logger := log.FromContext(ctx).WithName("cluster-registry").WithValues("cluster", clusterName)

	span.SetAttributes(attribute.String("cluster.name", clusterName))

	// Check if cluster exists.

	value, exists := wcr.clusters.Load(clusterName)

	if !exists {

		return fmt.Errorf("cluster %s not found", clusterName)

	}

	cluster, ok := value.(*WorkloadCluster)

	if !ok {

		return fmt.Errorf("invalid cluster object for %s", clusterName)

	}

	// Update cluster status.

	cluster.Status = WorkloadClusterStatusTerminating

	// Cleanup Config Sync.

	if err := wcr.cleanupConfigSync(ctx, cluster); err != nil {

		logger.Error(err, "Failed to cleanup Config Sync")

		// Continue with unregistration.

	}

	// Remove from registry.

	wcr.clusters.Delete(clusterName)

	wcr.registrations.Delete(clusterName)

	logger.Info("Cluster unregistered successfully")

	// Emit cluster event.

	wcr.metrics.ClusterEvents.WithLabelValues(clusterName, "unregistered").Inc()

	return nil

}

// GetWorkloadCluster retrieves a workload cluster by name.

func (wcr *WorkloadClusterRegistry) GetWorkloadCluster(ctx context.Context, clusterName string) (*WorkloadCluster, error) {

	if value, exists := wcr.clusters.Load(clusterName); exists {

		if cluster, ok := value.(*WorkloadCluster); ok {

			return cluster, nil

		}

	}

	return nil, fmt.Errorf("cluster %s not found", clusterName)

}

// ListWorkloadClusters lists all registered workload clusters.

func (wcr *WorkloadClusterRegistry) ListWorkloadClusters(ctx context.Context) ([]*WorkloadCluster, error) {

	clusters := make([]*WorkloadCluster, 0)

	wcr.clusters.Range(func(key, value interface{}) bool {

		if cluster, ok := value.(*WorkloadCluster); ok {

			clusters = append(clusters, cluster)

		}

		return true

	})

	return clusters, nil

}

// CheckClusterHealth performs a health check on a specific cluster.

func (wcr *WorkloadClusterRegistry) CheckClusterHealth(ctx context.Context, clusterName string) (*ClusterHealth, error) {

	ctx, span := wcr.tracer.Start(ctx, "check-cluster-health")

	defer span.End()

	span.SetAttributes(attribute.String("cluster.name", clusterName))

	startTime := time.Now()

	defer func() {

		wcr.metrics.HealthCheckDuration.WithLabelValues(

			clusterName, "unknown", // region would need to be extracted from cluster

		).Observe(time.Since(startTime).Seconds())

	}()

	cluster, err := wcr.GetWorkloadCluster(ctx, clusterName)

	if err != nil {

		span.RecordError(err)

		return nil, err

	}

	// Perform comprehensive health check.

	health := &ClusterHealth{

		LastChecked: time.Now(),

		Components: make(map[string]ComponentHealth),
	}

	// Check API server connectivity.

	apiHealth := wcr.checkAPIServerHealth(ctx, cluster)

	health.Components["api-server"] = apiHealth

	// Check node health.

	nodeHealth := wcr.checkNodeHealth(ctx, cluster)

	health.Components["nodes"] = nodeHealth

	// Check system pods.

	podHealth := wcr.checkSystemPodsHealth(ctx, cluster)

	health.Components["system-pods"] = podHealth

	// Check network connectivity.

	networkHealth := wcr.checkNetworkHealth(ctx, cluster)

	health.Components["network"] = networkHealth

	// Check resource utilization.

	resourceHealth := wcr.checkResourceHealth(ctx, cluster)

	health.Resources = resourceHealth

	// Determine overall status.

	health.Status = wcr.calculateOverallHealth(health.Components)

	// Count errors and warnings.

	for _, component := range health.Components {

		if component.Status == "Error" {

			health.ErrorCount++

		} else if component.Status == "Warning" {

			health.WarningCount++

		}

	}

	// Update cluster health.

	cluster.Health = health

	cluster.LastHealthCheck = &health.LastChecked

	// Update metrics.

	healthValue := 1.0

	if health.Status != "Healthy" {

		healthValue = 0.0

	}

	wcr.metrics.ClusterHealth.WithLabelValues(

		clusterName, cluster.Region, "overall",
	).Set(healthValue)

	span.SetAttributes(

		attribute.String("health.status", health.Status),

		attribute.Int("health.errors", health.ErrorCount),

		attribute.Int("health.warnings", health.WarningCount),
	)

	return health, nil

}

// validateCluster validates cluster configuration.

func (wcr *WorkloadClusterRegistry) validateCluster(ctx context.Context, cluster *WorkloadCluster) error {

	if cluster.Name == "" {

		return fmt.Errorf("cluster name is required")

	}

	if cluster.Endpoint == "" {

		return fmt.Errorf("cluster endpoint is required")

	}

	if cluster.Region == "" {

		return fmt.Errorf("cluster region is required")

	}

	return nil

}

// performConnectivityCheck checks if cluster is reachable.

func (wcr *WorkloadClusterRegistry) performConnectivityCheck(ctx context.Context, cluster *WorkloadCluster) error {

	// In a real implementation, this would:.

	// 1. Test connection to cluster API server.

	// 2. Verify authentication credentials.

	// 3. Test basic API operations.

	// 4. Check cluster version compatibility.

	// For now, simulate connectivity check.

	// return nil for successful connection.

	return nil

}

// setupConfigSync configures Config Sync for the cluster.

func (wcr *WorkloadClusterRegistry) setupConfigSync(ctx context.Context, cluster *WorkloadCluster) error {

	if cluster.ConfigSync == nil {

		cluster.ConfigSync = &ClusterConfigSync{

			Enabled: true,

			Repository: "https://github.com/nephoran/cluster-configs",

			Branch: "main",

			Directory: fmt.Sprintf("clusters/%s", cluster.Name),

			SyncPeriod: 30 * time.Second,

			Status: "Initializing",
		}

	}

	// In a real implementation, this would:.

	// 1. Install Config Sync in the cluster.

	// 2. Configure repository access.

	// 3. Set up sync policies.

	// 4. Verify sync status.

	cluster.ConfigSync.Status = "Active"

	now := time.Now()

	cluster.ConfigSync.LastSync = &now

	return nil

}

// cleanupConfigSync removes Config Sync configuration.

func (wcr *WorkloadClusterRegistry) cleanupConfigSync(ctx context.Context, cluster *WorkloadCluster) error {

	if cluster.ConfigSync != nil {

		cluster.ConfigSync.Status = "Terminating"

	}

	// In a real implementation, this would:.

	// 1. Remove Config Sync components.

	// 2. Clean up repository configurations.

	// 3. Remove sync policies.

	return nil

}

// Health check methods.

func (wcr *WorkloadClusterRegistry) checkAPIServerHealth(ctx context.Context, cluster *WorkloadCluster) ComponentHealth {

	// In a real implementation, this would test API server connectivity.

	return ComponentHealth{

		Name: "api-server",

		Status: "Healthy",

		Message: "API server is responsive",

		LastSeen: time.Now(),
	}

}

func (wcr *WorkloadClusterRegistry) checkNodeHealth(ctx context.Context, cluster *WorkloadCluster) ComponentHealth {

	// In a real implementation, this would check node readiness.

	return ComponentHealth{

		Name: "nodes",

		Status: "Healthy",

		Message: "All nodes are ready",

		LastSeen: time.Now(),
	}

}

func (wcr *WorkloadClusterRegistry) checkSystemPodsHealth(ctx context.Context, cluster *WorkloadCluster) ComponentHealth {

	// In a real implementation, this would check system pod status.

	return ComponentHealth{

		Name: "system-pods",

		Status: "Healthy",

		Message: "All system pods are running",

		LastSeen: time.Now(),
	}

}

func (wcr *WorkloadClusterRegistry) checkNetworkHealth(ctx context.Context, cluster *WorkloadCluster) ComponentHealth {

	// In a real implementation, this would test network connectivity.

	return ComponentHealth{

		Name: "network",

		Status: "Healthy",

		Message: "Network connectivity is good",

		LastSeen: time.Now(),
	}

}

func (wcr *WorkloadClusterRegistry) checkResourceHealth(ctx context.Context, cluster *WorkloadCluster) *ResourceHealthStatus {

	// In a real implementation, this would gather resource metrics.

	return &ResourceHealthStatus{

		CPUUsage: 45.5,

		MemoryUsage: 60.2,

		StorageUsage: 30.1,

		NetworkUsage: 25.7,

		PodCount: 45,

		NodeCount: 3,
	}

}

func (wcr *WorkloadClusterRegistry) calculateOverallHealth(components map[string]ComponentHealth) string {

	errorCount := 0

	warningCount := 0

	for _, component := range components {

		switch component.Status {

		case "Error":

			errorCount++

		case "Warning":

			warningCount++

		}

	}

	if errorCount > 0 {

		return "Unhealthy"

	}

	if warningCount > 0 {

		return "Warning"

	}

	return "Healthy"

}

// discoverAndRegisterClusters automatically discovers and registers clusters.

func (wcr *WorkloadClusterRegistry) discoverAndRegisterClusters(ctx context.Context) error {

	logger := log.FromContext(ctx).WithName("cluster-discovery")

	// In a real implementation, this would:.

	// 1. Query cloud providers for managed clusters.

	// 2. Scan network for cluster API servers.

	// 3. Check configuration files for cluster definitions.

	// 4. Register discovered clusters.

	// For now, register some example clusters.

	exampleClusters := []*WorkloadCluster{

		{

			Name: "edge-cluster-1",

			Endpoint: "https://edge-cluster-1.example.com:6443",

			Region: "us-east-1",

			Zone: "us-east-1a",

			Capabilities: []ClusterCapability{

				{

					Name: "5g-core",

					Type: "network-function",

					Version: "1.0",

					Status: "ready",
				},

				{

					Name: "edge-computing",

					Type: "compute",

					Version: "1.0",

					Status: "ready",
				},
			},

			Labels: map[string]string{

				"cluster-type": "edge",

				"provider": "aws",
			},
		},

		{

			Name: "central-cluster-1",

			Endpoint: "https://central-cluster-1.example.com:6443",

			Region: "us-west-2",

			Zone: "us-west-2b",

			Capabilities: []ClusterCapability{

				{

					Name: "5g-core",

					Type: "network-function",

					Version: "1.0",

					Status: "ready",
				},

				{

					Name: "oran-ric",

					Type: "network-function",

					Version: "1.0",

					Status: "ready",
				},
			},

			Labels: map[string]string{

				"cluster-type": "central",

				"provider": "aws",
			},
		},
	}

	for _, cluster := range exampleClusters {

		if err := wcr.RegisterWorkloadCluster(ctx, cluster); err != nil {

			logger.Error(err, "Failed to register discovered cluster", "cluster", cluster.Name)

			continue

		}

		logger.Info("Registered discovered cluster", "cluster", cluster.Name)

	}

	return nil

}

// ClusterRegistration represents a cluster registration record.

type ClusterRegistration struct {
	Cluster *WorkloadCluster `json:"cluster"`

	RegisteredAt time.Time `json:"registeredAt"`

	Status string `json:"status"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// Start health monitoring.

func (chm *ClusterHealthMonitor) Start() {

	logger := log.Log.WithName("cluster-health-monitor")

	logger.Info("Starting cluster health monitoring")

	ticker := time.NewTicker(chm.config.HealthCheckInterval)

	defer ticker.Stop()

	for {

		select {

		case <-ticker.C:

			chm.performHealthChecks(context.Background())

		case <-chm.stopCh:

			logger.Info("Stopping cluster health monitoring")

			return

		}

	}

}

// Stop health monitoring.

func (chm *ClusterHealthMonitor) Stop() {

	close(chm.stopCh)

}

// performHealthChecks performs health checks on all registered clusters.

func (chm *ClusterHealthMonitor) performHealthChecks(ctx context.Context) {

	logger := log.FromContext(ctx).WithName("health-checks")

	chm.registry.clusters.Range(func(key, value interface{}) bool {

		clusterName := key.(string)

		go func(name string) {

			if _, err := chm.registry.CheckClusterHealth(ctx, name); err != nil {

				logger.Error(err, "Health check failed", "cluster", name)

			}

		}(clusterName)

		return true

	})

}
