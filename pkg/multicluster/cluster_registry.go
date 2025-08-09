package multicluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ClusterRegistry manages registration and lifecycle of workload clusters
type ClusterRegistry struct {
	mu                  sync.RWMutex
	clusters            map[string]*WorkloadCluster
	topology            *NetworkTopology
	logger              *zap.Logger
	healthCheckInterval time.Duration
	healthCheckCancel   context.CancelFunc
}

// NewClusterRegistry creates a new cluster registry
func NewClusterRegistry(logger *zap.Logger) *ClusterRegistry {
	return &ClusterRegistry{
		clusters: make(map[string]*WorkloadCluster),
		topology: &NetworkTopology{
			Clusters:      make(map[string]*WorkloadCluster),
			LatencyMatrix: make(map[string]map[string]float64),
		},
		logger:              logger,
		healthCheckInterval: 5 * time.Minute,
	}
}

// RegisterCluster adds a new workload cluster to the registry
func (cr *ClusterRegistry) RegisterCluster(ctx context.Context, cluster *WorkloadCluster, options *ClusterRegistrationOptions) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Validate cluster connection
	if err := cr.validateClusterConnection(ctx, cluster, options); err != nil {
		return fmt.Errorf("cluster validation failed: %w", err)
	}

	// Generate unique cluster ID if not provided
	if cluster.ID == "" {
		cluster.ID = uuid.New().String()
	}

	// Check for duplicate clusters
	if _, exists := cr.clusters[cluster.ID]; exists {
		return fmt.Errorf("cluster with ID %s already registered", cluster.ID)
	}

	// Store cluster in registry
	cr.clusters[cluster.ID] = cluster
	cr.topology.Clusters[cluster.ID] = cluster

	cr.logger.Info("Cluster registered successfully",
		zap.String("clusterID", cluster.ID),
		zap.String("clusterName", cluster.Name))

	return nil
}

// validateClusterConnection performs connection and capability validation
func (cr *ClusterRegistry) validateClusterConnection(ctx context.Context, cluster *WorkloadCluster, options *ClusterRegistrationOptions) error {
	// Default connection timeout
	if options.ConnectionTimeout == 0 {
		options.ConnectionTimeout = 30 * time.Second
	}

	// Create Kubernetes client
	client, err := kubernetes.NewForConfig(cluster.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	cluster.Client = client

	// Validate cluster connection with retries
	var lastErr error
	for attempt := 0; attempt < options.ConnectionRetries; attempt++ {
		ctx, cancel := context.WithTimeout(ctx, options.ConnectionTimeout)
		defer cancel()

		// Perform health check
		_, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err == nil {
			cluster.Status = ClusterStatusHealthy
			cluster.LastCheckedAt = time.Now()
			return nil
		}

		lastErr = err
		time.Sleep(time.Second * time.Duration(attempt+1))
	}

	return fmt.Errorf("cluster connection failed after %d attempts: %w",
		options.ConnectionRetries, lastErr)
}

// UnregisterCluster removes a cluster from the registry
func (cr *ClusterRegistry) UnregisterCluster(ctx context.Context, clusterID string) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if _, exists := cr.clusters[clusterID]; !exists {
		return fmt.Errorf("cluster with ID %s not found", clusterID)
	}

	delete(cr.clusters, clusterID)
	delete(cr.topology.Clusters, clusterID)

	cr.logger.Info("Cluster unregistered", zap.String("clusterID", clusterID))
	return nil
}

// StartHealthChecks begins periodic health monitoring for registered clusters
func (cr *ClusterRegistry) StartHealthChecks(ctx context.Context) {
	ctx, cr.healthCheckCancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(cr.healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cr.performHealthChecks(ctx)
			}
		}
	}()
}

// performHealthChecks checks the status of all registered clusters
func (cr *ClusterRegistry) performHealthChecks(ctx context.Context) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	var wg sync.WaitGroup
	for _, cluster := range cr.clusters {
		wg.Add(1)
		go func(c *WorkloadCluster) {
			defer wg.Done()
			cr.checkClusterHealth(ctx, c)
		}(cluster)
	}
	wg.Wait()
}

// checkClusterHealth performs a comprehensive health check for a cluster
func (cr *ClusterRegistry) checkClusterHealth(ctx context.Context, cluster *WorkloadCluster) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Check node status
	nodes, err := cluster.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		cluster.Status = ClusterStatusUnreachable
		cr.logger.Warn("Cluster health check failed",
			zap.String("clusterID", cluster.ID),
			zap.Error(err))
		return
	}

	// Count ready nodes
	readyNodes := 0
	for _, node := range nodes.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				readyNodes++
			}
		}
	}

	// Determine cluster status based on node readiness
	if readyNodes == 0 {
		cluster.Status = ClusterStatusDegraded
	} else {
		cluster.Status = ClusterStatusHealthy
	}

	cluster.LastCheckedAt = time.Now()
	cr.logger.Info("Cluster health check completed",
		zap.String("clusterID", cluster.ID),
		zap.String("status", string(cluster.Status)))
}

// StopHealthChecks terminates the periodic health checking
func (cr *ClusterRegistry) StopHealthChecks() {
	if cr.healthCheckCancel != nil {
		cr.healthCheckCancel()
	}
}

// GetCluster retrieves a cluster by its ID
func (cr *ClusterRegistry) GetCluster(clusterID string) (*WorkloadCluster, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	cluster, exists := cr.clusters[clusterID]
	if !exists {
		return nil, fmt.Errorf("cluster with ID %s not found", clusterID)
	}

	return cluster, nil
}

// ListClusters returns all registered clusters
func (cr *ClusterRegistry) ListClusters() []*WorkloadCluster {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	clusters := make([]*WorkloadCluster, 0, len(cr.clusters))
	for _, cluster := range cr.clusters {
		clusters = append(clusters, cluster)
	}

	return clusters
}
