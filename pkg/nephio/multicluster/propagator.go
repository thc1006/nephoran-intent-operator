package multicluster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// Porch types are now defined locally in types.go.
)

// PackagePropagator manages multi-cluster package deployment.
type PackagePropagator struct {
	client     client.Client
	logger     logr.Logger
	clusterMgr ClusterManagerInterface
	syncEngine SyncEngineInterface
	customizer *Customizer
}

// PropagationStrategy defines different deployment strategies.
type PropagationStrategy string

const (
	// StrategySequential holds strategysequential value.
	StrategySequential PropagationStrategy = "sequential"
	// StrategyParallel holds strategyparallel value.
	StrategyParallel PropagationStrategy = "parallel"
	// StrategyCanaried holds strategycanaried value.
	StrategyCanaried PropagationStrategy = "canary"
)

// DeploymentOptions configures package deployment across clusters.
type DeploymentOptions struct {
	Strategy          PropagationStrategy
	MaxConcurrentDepl int
	Timeout           time.Duration
	RollbackOnFailure bool
}

// DeployPackage propagates a package across multiple clusters.
func (p *PackagePropagator) DeployPackage(
	ctx context.Context,
	packageRevision *PackageRevision,
	targetClusters []types.NamespacedName,
	opts DeploymentOptions,
) (*MultiClusterDeploymentStatus, error) {
	// 1. Validate input and set defaults.
	if err := p.validateDeploymentOptions(&opts); err != nil {
		return nil, fmt.Errorf("invalid deployment options: %w", err)
	}

	// 2. Select target clusters based on requirements.
	selectedClusters, err := p.clusterMgr.SelectTargetClusters(ctx, targetClusters, packageRevision)
	if err != nil {
		return nil, fmt.Errorf("cluster selection failed: %w", err)
	}

	// 3. Apply different propagation strategies.
	switch opts.Strategy {
	case StrategySequential:
		return p.deploySequential(ctx, packageRevision, selectedClusters, opts)
	case StrategyParallel:
		return p.deployParallel(ctx, packageRevision, selectedClusters, opts)
	case StrategyCanaried:
		return p.deployCanary(ctx, packageRevision, selectedClusters, opts)
	default:
		return nil, fmt.Errorf("unsupported propagation strategy: %s", opts.Strategy)
	}
}

// deploySequential deploys packages to clusters sequentially.
func (p *PackagePropagator) deploySequential(
	ctx context.Context,
	packageRevision *PackageRevision,
	clusters []types.NamespacedName,
	opts DeploymentOptions,
) (*MultiClusterDeploymentStatus, error) {
	deploymentStatus := &MultiClusterDeploymentStatus{
		Clusters: make(map[string]ClusterDeploymentStatus),
	}

	for _, cluster := range clusters {
		// Create cluster-specific package variant.
		customizedPkg, err := p.customizer.CustomizePackage(ctx, packageRevision, cluster)
		if err != nil {
			return nil, fmt.Errorf("package customization failed for cluster %v: %w", cluster, err)
		}

		// Deploy to single cluster.
		clusterStatus, err := p.syncEngine.SyncPackageToCluster(ctx, customizedPkg, cluster)
		if err != nil {
			// Handle rollback if configured.
			if opts.RollbackOnFailure {
				p.rollbackDeployment(ctx, deploymentStatus)
			}
			return nil, fmt.Errorf("deployment to cluster %v failed: %w", cluster, err)
		}

		deploymentStatus.Clusters[cluster.String()] = *clusterStatus
	}

	return deploymentStatus, nil
}

// deployParallel deploys packages to multiple clusters concurrently.
func (p *PackagePropagator) deployParallel(
	ctx context.Context,
	packageRevision *PackageRevision,
	clusters []types.NamespacedName,
	opts DeploymentOptions,
) (*MultiClusterDeploymentStatus, error) {
	deploymentStatus := &MultiClusterDeploymentStatus{
		Clusters: make(map[string]ClusterDeploymentStatus),
	}
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use semaphore to limit concurrent deployments.
	sem := make(chan struct{}, opts.MaxConcurrentDepl)

	for _, cluster := range clusters {
		wg.Add(1)
		sem <- struct{}{}

		go func(cluster types.NamespacedName) {
			defer wg.Done()
			defer func() { <-sem }()

			// Create cluster-specific package variant.
			customizedPkg, err := p.customizer.CustomizePackage(ctx, packageRevision, cluster)
			if err != nil {
				p.logger.Error(err, "Package customization failed", "cluster", cluster)
				return
			}

			// Deploy to cluster.
			clusterStatus, err := p.syncEngine.SyncPackageToCluster(ctx, customizedPkg, cluster)
			if err != nil {
				p.logger.Error(err, "Deployment to cluster failed", "cluster", cluster)
				return
			}

			// Thread-safe status update.
			mu.Lock()
			deploymentStatus.Clusters[cluster.String()] = *clusterStatus
			mu.Unlock()
		}(cluster)
	}

	// Wait for all deployments to complete.
	wg.Wait()
	close(sem)

	return deploymentStatus, nil
}

// deployCanary implements a canary deployment strategy.
func (p *PackagePropagator) deployCanary(
	ctx context.Context,
	packageRevision *PackageRevision,
	clusters []types.NamespacedName,
	opts DeploymentOptions,
) (*MultiClusterDeploymentStatus, error) {
	// Implement canary deployment logic.
	// 1. Select a subset of clusters for initial deployment.
	// 2. Monitor health and performance.
	// 3. Gradually roll out to remaining clusters.
	return nil, fmt.Errorf("canary deployment not yet implemented")
}

// rollbackDeployment handles rollback of a multi-cluster deployment.
func (p *PackagePropagator) rollbackDeployment(
	ctx context.Context,
	status *MultiClusterDeploymentStatus,
) error {
	// Implement rollback logic for deployed clusters.
	return nil
}

// validateDeploymentOptions ensures deployment options are valid.
func (p *PackagePropagator) validateDeploymentOptions(opts *DeploymentOptions) error {
	// Set defaults.
	if opts.Strategy == "" {
		opts.Strategy = StrategyParallel
	}
	if opts.MaxConcurrentDepl == 0 {
		opts.MaxConcurrentDepl = 10
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Minute
	}

	return nil
}

// NewPackagePropagator creates a new package propagator.
func NewPackagePropagator(
	client client.Client,
	logger logr.Logger,
	clusterMgr ClusterManagerInterface,
	syncEngine SyncEngineInterface,
	customizer *Customizer,
) *PackagePropagator {
	return &PackagePropagator{
		client:     client,
		logger:     logger,
		clusterMgr: clusterMgr,
		syncEngine: syncEngine,
		customizer: customizer,
	}
}

// SetSyncEngine sets the sync engine for testing purposes.
func (p *PackagePropagator) SetSyncEngine(syncEngine SyncEngineInterface) {
	p.syncEngine = syncEngine
}

// GetSyncEngine returns the sync engine for testing purposes.
func (p *PackagePropagator) GetSyncEngine() SyncEngineInterface {
	return p.syncEngine
}

// SetClusterManager sets the cluster manager for testing purposes.
func (p *PackagePropagator) SetClusterManager(clusterMgr ClusterManagerInterface) {
	p.clusterMgr = clusterMgr
}

// GetClusterManager returns the cluster manager for testing purposes.
func (p *PackagePropagator) GetClusterManager() ClusterManagerInterface {
	return p.clusterMgr
}
