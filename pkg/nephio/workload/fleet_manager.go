package workload

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
)

// FleetManager manages fleets of Nephio workload clusters.
type FleetManager struct {
	registry     *ClusterRegistry
	fleets       map[string]*Fleet
	policies     map[string]*FleetPolicy
	scheduler    *WorkloadScheduler
	loadBalancer *LoadBalancer
	analytics    *FleetAnalytics
	logger       logr.Logger
	metrics      *fleetMetrics
	mu           sync.RWMutex
	stopCh       chan struct{}
}

// Fleet represents a group of clusters managed together.
type Fleet struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ClusterIDs  []string          `json:"cluster_ids"`
	Selector    labels.Selector   `json:"-"`
	Policy      *FleetPolicy      `json:"policy"`
	Metadata    map[string]string `json:"metadata"`
	Status      FleetStatus       `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// FleetStatus represents the status of a fleet.
type FleetStatus struct {
	State           string               `json:"state"`
	ClusterCount    int                  `json:"cluster_count"`
	HealthyClusters int                  `json:"healthy_clusters"`
	TotalResources  ResourceCapacity     `json:"total_resources"`
	UsedResources   ResourceCapacity     `json:"used_resources"`
	Workloads       []WorkloadDeployment `json:"workloads"`
	LastSync        time.Time            `json:"last_sync"`
	Message         string               `json:"message"`
}

// FleetPolicy defines policies for fleet management.
type FleetPolicy struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	ScalingPolicy       ScalingPolicy       `json:"scaling_policy"`
	PlacementPolicy     PlacementPolicy     `json:"placement_policy"`
	LoadBalancingPolicy LoadBalancingPolicy `json:"load_balancing_policy"`
	FailoverPolicy      FailoverPolicy      `json:"failover_policy"`
	SecurityPolicy      SecurityPolicy      `json:"security_policy"`
	CompliancePolicy    CompliancePolicy    `json:"compliance_policy"`
	CostPolicy          CostPolicy          `json:"cost_policy"`
	Constraints         []FleetConstraint   `json:"constraints"`
}

// ScalingPolicy defines scaling rules for the fleet.
type ScalingPolicy struct {
	MinClusters       int            `json:"min_clusters"`
	MaxClusters       int            `json:"max_clusters"`
	AutoScale         bool           `json:"auto_scale"`
	ScaleUpTriggers   []ScaleTrigger `json:"scale_up_triggers"`
	ScaleDownTriggers []ScaleTrigger `json:"scale_down_triggers"`
	CooldownPeriod    time.Duration  `json:"cooldown_period"`
}

// ScaleTrigger defines a trigger for scaling operations.
type ScaleTrigger struct {
	Metric    string        `json:"metric"`
	Threshold float64       `json:"threshold"`
	Duration  time.Duration `json:"duration"`
	Action    string        `json:"action"`
}

// PlacementPolicy defines workload placement rules.
type PlacementPolicy struct {
	Strategy          PlacementStrategy     `json:"strategy"`
	AffinityRules     []AffinityRule        `json:"affinity_rules"`
	AntiAffinityRules []AntiAffinityRule    `json:"anti_affinity_rules"`
	SpreadConstraints []SpreadConstraint    `json:"spread_constraints"`
	Preferences       []PlacementPreference `json:"preferences"`
}

// PlacementStrategy defines the strategy for workload placement.
type PlacementStrategy string

const (
	// PlacementStrategyBinPacking holds placementstrategybinpacking value.
	PlacementStrategyBinPacking PlacementStrategy = "bin-packing"
	// PlacementStrategySpread holds placementstrategyspread value.
	PlacementStrategySpread PlacementStrategy = "spread"
	// PlacementStrategyCostOptimized holds placementstrategycostoptimized value.
	PlacementStrategyCostOptimized PlacementStrategy = "cost-optimized"
	// PlacementStrategyLatencyOptimized holds placementstrategylatencyoptimized value.
	PlacementStrategyLatencyOptimized PlacementStrategy = "latency-optimized"
	// PlacementStrategyBalanced holds placementstrategybalanced value.
	PlacementStrategyBalanced PlacementStrategy = "balanced"
)

// AffinityRule defines affinity requirements.
type AffinityRule struct {
	Type     string          `json:"type"`
	Selector labels.Selector `json:"-"`
	Weight   int             `json:"weight"`
	Topology string          `json:"topology"`
}

// AntiAffinityRule defines anti-affinity requirements.
type AntiAffinityRule struct {
	Type     string          `json:"type"`
	Selector labels.Selector `json:"-"`
	Weight   int             `json:"weight"`
	Topology string          `json:"topology"`
}

// SpreadConstraint defines spread requirements.
type SpreadConstraint struct {
	MaxSkew           int    `json:"max_skew"`
	TopologyKey       string `json:"topology_key"`
	WhenUnsatisfiable string `json:"when_unsatisfiable"`
}

// PlacementPreference defines placement preferences.
type PlacementPreference struct {
	Type   string `json:"type"`
	Weight int    `json:"weight"`
	Value  string `json:"value"`
}

// LoadBalancingPolicy defines load balancing rules.
type LoadBalancingPolicy struct {
	Algorithm           LoadBalancingAlgorithm `json:"algorithm"`
	HealthCheckPath     string                 `json:"health_check_path"`
	HealthCheckInterval time.Duration          `json:"health_check_interval"`
	SessionAffinity     bool                   `json:"session_affinity"`
	WeightedTargets     map[string]int         `json:"weighted_targets"`
}

// LoadBalancingAlgorithm defines the load balancing algorithm.
type LoadBalancingAlgorithm string

const (
	// LoadBalancingRoundRobin holds loadbalancingroundrobin value.
	LoadBalancingRoundRobin LoadBalancingAlgorithm = "round-robin"
	// LoadBalancingLeastConnections holds loadbalancingleastconnections value.
	LoadBalancingLeastConnections LoadBalancingAlgorithm = "least-connections"
	// LoadBalancingWeighted holds loadbalancingweighted value.
	LoadBalancingWeighted LoadBalancingAlgorithm = "weighted"
	// LoadBalancingRandom holds loadbalancingrandom value.
	LoadBalancingRandom LoadBalancingAlgorithm = "random"
	// LoadBalancingIPHash holds loadbalancingiphash value.
	LoadBalancingIPHash LoadBalancingAlgorithm = "ip-hash"
)

// FailoverPolicy defines failover rules.
type FailoverPolicy struct {
	Enabled             bool          `json:"enabled"`
	FailoverThreshold   int           `json:"failover_threshold"`
	FailbackDelay       time.Duration `json:"failback_delay"`
	PreferredClusters   []string      `json:"preferred_clusters"`
	BackupClusters      []string      `json:"backup_clusters"`
	DataReplicationMode string        `json:"data_replication_mode"`
}

// SecurityPolicy defines security requirements for the fleet.
type SecurityPolicy struct {
	NetworkPolicies     []string `json:"network_policies"`
	PodSecurityStandard string   `json:"pod_security_standard"`
	EncryptionRequired  bool     `json:"encryption_required"`
	MutualTLS           bool     `json:"mutual_tls"`
	ComplianceStandards []string `json:"compliance_standards"`
}

// CompliancePolicy defines compliance requirements.
type CompliancePolicy struct {
	DataResidency   []string      `json:"data_residency"`
	Standards       []string      `json:"standards"`
	AuditLogging    bool          `json:"audit_logging"`
	RetentionPeriod time.Duration `json:"retention_period"`
}

// CostPolicy defines cost management rules.
type CostPolicy struct {
	MaxMonthlyCost       float64         `json:"max_monthly_cost"`
	PreferredProviders   []CloudProvider `json:"preferred_providers"`
	SpotInstancesAllowed bool            `json:"spot_instances_allowed"`
	ReservedInstances    bool            `json:"reserved_instances"`
	CostAlerts           []CostAlert     `json:"cost_alerts"`
}

// CostAlert defines a cost alert threshold.
type CostAlert struct {
	Threshold float64  `json:"threshold"`
	Type      string   `json:"type"`
	Actions   []string `json:"actions"`
}

// FleetConstraint defines a constraint for the fleet.
type FleetConstraint struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Enforcement string `json:"enforcement"`
}

// WorkloadDeployment represents a workload deployed across the fleet.
type WorkloadDeployment struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Type         string               `json:"type"`
	Replicas     int                  `json:"replicas"`
	Distribution map[string]int       `json:"distribution"`
	Resources    ResourceRequirements `json:"resources"`
	Status       WorkloadStatus       `json:"status"`
	CreatedAt    time.Time            `json:"created_at"`
}

// WorkloadStatus represents the status of a workload.
type WorkloadStatus struct {
	State         string    `json:"state"`
	ReadyReplicas int       `json:"ready_replicas"`
	TotalReplicas int       `json:"total_replicas"`
	Message       string    `json:"message"`
	LastUpdate    time.Time `json:"last_update"`
}

// WorkloadScheduler handles workload scheduling across clusters.
type WorkloadScheduler struct {
	registry  *ClusterRegistry
	policies  map[string]*PlacementPolicy
	optimizer *PlacementOptimizer
	logger    logr.Logger
}

// LoadBalancer handles load balancing across clusters.
type LoadBalancer struct {
	backends     map[string]*LoadBalancerBackend
	healthChecks map[string]*LoadBalancerHealthCheck
	algorithm    LoadBalancingAlgorithm
	mu           sync.RWMutex
}

// LoadBalancerBackend represents a backend for load balancing.
type LoadBalancerBackend struct {
	ClusterID   string    `json:"cluster_id"`
	Endpoint    string    `json:"endpoint"`
	Weight      int       `json:"weight"`
	Healthy     bool      `json:"healthy"`
	Connections int       `json:"connections"`
	LastCheck   time.Time `json:"last_check"`
}

// LoadBalancerHealthCheck represents a health check configuration for load balancer.
type LoadBalancerHealthCheck struct {
	Path     string        `json:"path"`
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
	Retries  int           `json:"retries"`
}

// FleetAnalytics provides analytics for fleet operations.
type FleetAnalytics struct {
	metrics     map[string]*FleetMetrics
	trends      map[string]*FleetTrends
	predictions map[string]*FleetPredictions
	mu          sync.RWMutex
}

// FleetMetrics contains metrics for a fleet.
type FleetMetrics struct {
	ResourceUtilization float64 `json:"resource_utilization"`
	CostPerWorkload     float64 `json:"cost_per_workload"`
	AvailabilityScore   float64 `json:"availability_score"`
	PerformanceScore    float64 `json:"performance_score"`
	ComplianceScore     float64 `json:"compliance_score"`
	WorkloadCount       int     `json:"workload_count"`
	ErrorRate           float64 `json:"error_rate"`
	Throughput          float64 `json:"throughput"`
}

// FleetTrends contains trend analysis for a fleet.
type FleetTrends struct {
	GrowthRate         float64  `json:"growth_rate"`
	CostTrend          string   `json:"cost_trend"`
	UtilizationTrend   string   `json:"utilization_trend"`
	PredictedDemand    float64  `json:"predicted_demand"`
	RecommendedActions []string `json:"recommended_actions"`
}

// FleetPredictions contains predictions for fleet operations.
type FleetPredictions struct {
	FutureCapacityNeeds ResourceCapacity `json:"future_capacity_needs"`
	ExpectedCost        float64          `json:"expected_cost"`
	ScalingEvents       []ScalingEvent   `json:"scaling_events"`
	MaintenanceWindows  []time.Time      `json:"maintenance_windows"`
}

// ResourceRequirements represents resource requirements for a workload.
type ResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty"`
}

// ResourceList represents a list of resources.
type ResourceList map[string]resource.Quantity

// ScalingEvent represents a predicted scaling event.
type ScalingEvent struct {
	Time      time.Time `json:"time"`
	Type      string    `json:"type"`
	Magnitude int       `json:"magnitude"`
	Reason    string    `json:"reason"`
}

// PlacementOptimizer optimizes workload placement.
type PlacementOptimizer struct {
	strategies map[PlacementStrategy]PlacementFunction
	logger     logr.Logger
}

// PlacementFunction is a function that implements a placement strategy.
type PlacementFunction func(workload *WorkloadDeployment, clusters []*ClusterEntry) (*PlacementDecision, error)

// PlacementDecision represents a placement decision.
type PlacementDecision struct {
	ClusterDistribution map[string]int `json:"cluster_distribution"`
	Score               float64        `json:"score"`
	Constraints         []string       `json:"constraints"`
	Reasons             []string       `json:"reasons"`
}

// fleetMetrics contains Prometheus metrics for fleet operations.
type fleetMetrics struct {
	fleetsTotal            *prometheus.GaugeVec
	workloadsTotal         *prometheus.GaugeVec
	placementDuration      *prometheus.HistogramVec
	scalingOperations      *prometheus.CounterVec
	loadBalancingDecisions *prometheus.CounterVec
	failoverEvents         *prometheus.CounterVec
}

// NewFleetManager creates a new fleet manager.
func NewFleetManager(registry *ClusterRegistry, logger logr.Logger) *FleetManager {
	metrics := &fleetMetrics{
		fleetsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_fleet_total",
				Help: "Total number of fleets",
			},
			[]string{"status"},
		),
		workloadsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_fleet_workloads_total",
				Help: "Total number of workloads in fleets",
			},
			[]string{"fleet", "type"},
		),
		placementDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephio_fleet_placement_duration_seconds",
				Help:    "Duration of workload placement operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"strategy"},
		),
		scalingOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_fleet_scaling_operations_total",
				Help: "Total number of scaling operations",
			},
			[]string{"fleet", "direction", "result"},
		),
		loadBalancingDecisions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_fleet_load_balancing_decisions_total",
				Help: "Total number of load balancing decisions",
			},
			[]string{"algorithm", "result"},
		),
		failoverEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_fleet_failover_events_total",
				Help: "Total number of failover events",
			},
			[]string{"fleet", "reason"},
		),
	}

	// Register metrics.
	prometheus.MustRegister(
		metrics.fleetsTotal,
		metrics.workloadsTotal,
		metrics.placementDuration,
		metrics.scalingOperations,
		metrics.loadBalancingDecisions,
		metrics.failoverEvents,
	)

	scheduler := &WorkloadScheduler{
		registry:  registry,
		policies:  make(map[string]*PlacementPolicy),
		optimizer: NewPlacementOptimizer(logger),
		logger:    logger.WithName("scheduler"),
	}

	loadBalancer := &LoadBalancer{
		backends:     make(map[string]*LoadBalancerBackend),
		healthChecks: make(map[string]*LoadBalancerHealthCheck),
		algorithm:    LoadBalancingRoundRobin,
	}

	analytics := &FleetAnalytics{
		metrics:     make(map[string]*FleetMetrics),
		trends:      make(map[string]*FleetTrends),
		predictions: make(map[string]*FleetPredictions),
	}

	return &FleetManager{
		registry:     registry,
		fleets:       make(map[string]*Fleet),
		policies:     make(map[string]*FleetPolicy),
		scheduler:    scheduler,
		loadBalancer: loadBalancer,
		analytics:    analytics,
		logger:       logger.WithName("fleet-manager"),
		metrics:      metrics,
		stopCh:       make(chan struct{}),
	}
}

// Start starts the fleet manager.
func (fm *FleetManager) Start(ctx context.Context) error {
	fm.logger.Info("Starting fleet manager")

	// Start fleet synchronization.
	go fm.runFleetSync(ctx)

	// Start auto-scaling.
	go fm.runAutoScaling(ctx)

	// Start health monitoring.
	go fm.runHealthMonitoring(ctx)

	// Start analytics.
	go fm.runAnalytics(ctx)

	return nil
}

// Stop stops the fleet manager.
func (fm *FleetManager) Stop() {
	fm.logger.Info("Stopping fleet manager")
	close(fm.stopCh)
}

// CreateFleet creates a new fleet.
func (fm *FleetManager) CreateFleet(ctx context.Context, fleet *Fleet) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if _, exists := fm.fleets[fleet.ID]; exists {
		return fmt.Errorf("fleet %s already exists", fleet.ID)
	}

	fm.logger.Info("Creating fleet", "id", fleet.ID, "name", fleet.Name)

	// Initialize fleet status.
	fleet.Status = FleetStatus{
		State:        "active",
		ClusterCount: len(fleet.ClusterIDs),
		LastSync:     time.Now(),
	}

	// Calculate initial resources.
	fleet.Status.TotalResources = fm.calculateFleetResources(fleet.ClusterIDs)

	// Add fleet to manager.
	fm.fleets[fleet.ID] = fleet

	// Update metrics.
	fm.metrics.fleetsTotal.WithLabelValues("active").Inc()

	fm.logger.Info("Successfully created fleet", "id", fleet.ID)
	return nil
}

// UpdateFleet updates an existing fleet.
func (fm *FleetManager) UpdateFleet(ctx context.Context, fleetID string, updates *Fleet) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fleet, exists := fm.fleets[fleetID]
	if !exists {
		return fmt.Errorf("fleet %s not found", fleetID)
	}

	fm.logger.Info("Updating fleet", "id", fleetID)

	// Update fleet properties.
	if updates.Name != "" {
		fleet.Name = updates.Name
	}
	if updates.Description != "" {
		fleet.Description = updates.Description
	}
	if len(updates.ClusterIDs) > 0 {
		fleet.ClusterIDs = updates.ClusterIDs
		fleet.Status.ClusterCount = len(updates.ClusterIDs)
		fleet.Status.TotalResources = fm.calculateFleetResources(updates.ClusterIDs)
	}
	if updates.Policy != nil {
		fleet.Policy = updates.Policy
	}

	fleet.UpdatedAt = time.Now()

	fm.logger.Info("Successfully updated fleet", "id", fleetID)
	return nil
}

// DeleteFleet deletes a fleet.
func (fm *FleetManager) DeleteFleet(ctx context.Context, fleetID string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fleet, exists := fm.fleets[fleetID]
	if !exists {
		return fmt.Errorf("fleet %s not found", fleetID)
	}

	fm.logger.Info("Deleting fleet", "id", fleetID)

	// Check for active workloads.
	if len(fleet.Status.Workloads) > 0 {
		return fmt.Errorf("cannot delete fleet %s with active workloads", fleetID)
	}

	// Remove fleet.
	delete(fm.fleets, fleetID)

	// Update metrics.
	fm.metrics.fleetsTotal.WithLabelValues("active").Dec()

	fm.logger.Info("Successfully deleted fleet", "id", fleetID)
	return nil
}

// GetFleet retrieves a fleet by ID.
func (fm *FleetManager) GetFleet(fleetID string) (*Fleet, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	fleet, exists := fm.fleets[fleetID]
	if !exists {
		return nil, fmt.Errorf("fleet %s not found", fleetID)
	}

	return fleet, nil
}

// ListFleets lists all fleets.
func (fm *FleetManager) ListFleets() []*Fleet {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	fleets := make([]*Fleet, 0, len(fm.fleets))
	for _, fleet := range fm.fleets {
		fleets = append(fleets, fleet)
	}

	return fleets
}

// DeployWorkload deploys a workload across a fleet.
func (fm *FleetManager) DeployWorkload(ctx context.Context, fleetID string, workload *WorkloadDeployment) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fleet, exists := fm.fleets[fleetID]
	if !exists {
		return fmt.Errorf("fleet %s not found", fleetID)
	}

	fm.logger.Info("Deploying workload", "fleet", fleetID, "workload", workload.Name)

	// Get available clusters.
	clusters := fm.getFleetClusters(fleet)
	if len(clusters) == 0 {
		return fmt.Errorf("no available clusters in fleet %s", fleetID)
	}

	// Calculate placement.
	timer := prometheus.NewTimer(fm.metrics.placementDuration.WithLabelValues(string(fleet.Policy.PlacementPolicy.Strategy)))
	decision, err := fm.scheduler.ScheduleWorkload(ctx, workload, clusters, &fleet.Policy.PlacementPolicy)
	timer.ObserveDuration()

	if err != nil {
		return fmt.Errorf("failed to schedule workload: %w", err)
	}

	// Apply placement decision.
	workload.Distribution = decision.ClusterDistribution
	workload.Status = WorkloadStatus{
		State:         "deploying",
		TotalReplicas: workload.Replicas,
		LastUpdate:    time.Now(),
	}

	// Add workload to fleet.
	fleet.Status.Workloads = append(fleet.Status.Workloads, *workload)

	// Update metrics.
	fm.metrics.workloadsTotal.WithLabelValues(fleetID, workload.Type).Inc()

	fm.logger.Info("Successfully scheduled workload", "fleet", fleetID, "workload", workload.Name)
	return nil
}

// ScaleWorkload scales a workload in a fleet.
func (fm *FleetManager) ScaleWorkload(ctx context.Context, fleetID string, workloadID string, replicas int) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fleet, exists := fm.fleets[fleetID]
	if !exists {
		return fmt.Errorf("fleet %s not found", fleetID)
	}

	// Find workload.
	var workload *WorkloadDeployment
	for i := range fleet.Status.Workloads {
		if fleet.Status.Workloads[i].ID == workloadID {
			workload = &fleet.Status.Workloads[i]
			break
		}
	}

	if workload == nil {
		return fmt.Errorf("workload %s not found in fleet %s", workloadID, fleetID)
	}

	fm.logger.Info("Scaling workload", "fleet", fleetID, "workload", workloadID, "replicas", replicas)

	oldReplicas := workload.Replicas
	workload.Replicas = replicas

	// Recalculate distribution.
	clusters := fm.getFleetClusters(fleet)
	decision, err := fm.scheduler.ScheduleWorkload(ctx, workload, clusters, &fleet.Policy.PlacementPolicy)
	if err != nil {
		workload.Replicas = oldReplicas // Rollback
		return fmt.Errorf("failed to reschedule workload: %w", err)
	}

	workload.Distribution = decision.ClusterDistribution
	workload.Status.TotalReplicas = replicas
	workload.Status.LastUpdate = time.Now()

	// Update metrics.
	direction := "up"
	if replicas < oldReplicas {
		direction = "down"
	}
	fm.metrics.scalingOperations.WithLabelValues(fleetID, direction, "success").Inc()

	fm.logger.Info("Successfully scaled workload", "fleet", fleetID, "workload", workloadID)
	return nil
}

// MigrateWorkload migrates a workload between clusters.
func (fm *FleetManager) MigrateWorkload(ctx context.Context, fleetID string, workloadID string, fromCluster string, toCluster string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fleet, exists := fm.fleets[fleetID]
	if !exists {
		return fmt.Errorf("fleet %s not found", fleetID)
	}

	// Find workload.
	var workload *WorkloadDeployment
	for i := range fleet.Status.Workloads {
		if fleet.Status.Workloads[i].ID == workloadID {
			workload = &fleet.Status.Workloads[i]
			break
		}
	}

	if workload == nil {
		return fmt.Errorf("workload %s not found in fleet %s", workloadID, fleetID)
	}

	fm.logger.Info("Migrating workload", "fleet", fleetID, "workload", workloadID, "from", fromCluster, "to", toCluster)

	// Check if migration is valid.
	fromReplicas, fromExists := workload.Distribution[fromCluster]
	if !fromExists || fromReplicas == 0 {
		return fmt.Errorf("workload %s has no replicas on cluster %s", workloadID, fromCluster)
	}

	// Perform migration.
	workload.Distribution[fromCluster]--
	if workload.Distribution[fromCluster] == 0 {
		delete(workload.Distribution, fromCluster)
	}
	workload.Distribution[toCluster]++

	workload.Status.LastUpdate = time.Now()
	workload.Status.Message = fmt.Sprintf("Migrated from %s to %s", fromCluster, toCluster)

	fm.logger.Info("Successfully migrated workload", "fleet", fleetID, "workload", workloadID)
	return nil
}

// BalanceFleet rebalances workloads across a fleet.
func (fm *FleetManager) BalanceFleet(ctx context.Context, fleetID string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fleet, exists := fm.fleets[fleetID]
	if !exists {
		return fmt.Errorf("fleet %s not found", fleetID)
	}

	fm.logger.Info("Balancing fleet", "id", fleetID)

	clusters := fm.getFleetClusters(fleet)
	if len(clusters) == 0 {
		return fmt.Errorf("no available clusters in fleet %s", fleetID)
	}

	// Rebalance each workload.
	for i := range fleet.Status.Workloads {
		workload := &fleet.Status.Workloads[i]

		decision, err := fm.scheduler.ScheduleWorkload(ctx, workload, clusters, &fleet.Policy.PlacementPolicy)
		if err != nil {
			fm.logger.Error(err, "Failed to rebalance workload", "workload", workload.ID)
			continue
		}

		workload.Distribution = decision.ClusterDistribution
		workload.Status.LastUpdate = time.Now()
		workload.Status.Message = "Rebalanced"
	}

	fleet.Status.LastSync = time.Now()
	fleet.Status.Message = "Fleet rebalanced"

	fm.logger.Info("Successfully balanced fleet", "id", fleetID)
	return nil
}

// GetFleetAnalytics retrieves analytics for a fleet.
func (fm *FleetManager) GetFleetAnalytics(fleetID string) (*FleetMetrics, *FleetTrends, *FleetPredictions, error) {
	fm.analytics.mu.RLock()
	defer fm.analytics.mu.RUnlock()

	metrics, metricsExists := fm.analytics.metrics[fleetID]
	trends, trendsExists := fm.analytics.trends[fleetID]
	predictions, predictionsExists := fm.analytics.predictions[fleetID]

	if !metricsExists || !trendsExists || !predictionsExists {
		return nil, nil, nil, fmt.Errorf("analytics not available for fleet %s", fleetID)
	}

	return metrics, trends, predictions, nil
}

// Private methods.

func (fm *FleetManager) runFleetSync(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fm.stopCh:
			return
		case <-ticker.C:
			fm.syncFleets(ctx)
		}
	}
}

func (fm *FleetManager) syncFleets(ctx context.Context) {
	fm.mu.RLock()
	fleets := make([]*Fleet, 0, len(fm.fleets))
	for _, fleet := range fm.fleets {
		fleets = append(fleets, fleet)
	}
	fm.mu.RUnlock()

	for _, fleet := range fleets {
		fm.syncSingleFleet(ctx, fleet)
	}
}

func (fm *FleetManager) syncSingleFleet(ctx context.Context, fleet *Fleet) {
	// Update cluster membership.
	clusters := fm.getFleetClusters(fleet)

	fm.mu.Lock()
	fleet.Status.ClusterCount = len(clusters)
	fleet.Status.HealthyClusters = 0

	for _, cluster := range clusters {
		if cluster.Status == ClusterStatusHealthy {
			fleet.Status.HealthyClusters++
		}
	}

	// Update resource totals.
	fleet.Status.TotalResources = fm.calculateFleetResources(fleet.ClusterIDs)

	// Update workload status.
	for i := range fleet.Status.Workloads {
		workload := &fleet.Status.Workloads[i]
		fm.updateWorkloadStatus(ctx, workload, clusters)
	}

	fleet.Status.LastSync = time.Now()
	fm.mu.Unlock()
}

func (fm *FleetManager) runAutoScaling(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fm.stopCh:
			return
		case <-ticker.C:
			fm.checkAutoScaling(ctx)
		}
	}
}

func (fm *FleetManager) checkAutoScaling(ctx context.Context) {
	fm.mu.RLock()
	fleets := make([]*Fleet, 0, len(fm.fleets))
	for _, fleet := range fm.fleets {
		if fleet.Policy != nil && fleet.Policy.ScalingPolicy.AutoScale {
			fleets = append(fleets, fleet)
		}
	}
	fm.mu.RUnlock()

	for _, fleet := range fleets {
		fm.checkFleetScaling(ctx, fleet)
	}
}

func (fm *FleetManager) checkFleetScaling(ctx context.Context, fleet *Fleet) {
	// Check scale up triggers.
	for _, trigger := range fleet.Policy.ScalingPolicy.ScaleUpTriggers {
		if fm.evaluateScaleTrigger(fleet, trigger) {
			fm.scaleFleet(ctx, fleet, 1)
			fm.metrics.scalingOperations.WithLabelValues(fleet.ID, "up", "auto").Inc()
			return
		}
	}

	// Check scale down triggers.
	for _, trigger := range fleet.Policy.ScalingPolicy.ScaleDownTriggers {
		if fm.evaluateScaleTrigger(fleet, trigger) {
			fm.scaleFleet(ctx, fleet, -1)
			fm.metrics.scalingOperations.WithLabelValues(fleet.ID, "down", "auto").Inc()
			return
		}
	}
}

func (fm *FleetManager) evaluateScaleTrigger(fleet *Fleet, trigger ScaleTrigger) bool {
	// Evaluate trigger based on fleet metrics.
	// This is a simplified implementation.
	metrics, exists := fm.analytics.metrics[fleet.ID]
	if !exists {
		return false
	}

	switch trigger.Metric {
	case "cpu_utilization":
		return metrics.ResourceUtilization > trigger.Threshold
	case "error_rate":
		return metrics.ErrorRate > trigger.Threshold
	case "throughput":
		return metrics.Throughput > trigger.Threshold
	default:
		return false
	}
}

func (fm *FleetManager) scaleFleet(ctx context.Context, fleet *Fleet, delta int) error {
	// Implementation would provision or deprovision clusters.
	fm.logger.Info("Scaling fleet", "fleet", fleet.ID, "delta", delta)
	return nil
}

func (fm *FleetManager) runHealthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fm.stopCh:
			return
		case <-ticker.C:
			fm.monitorFleetHealth(ctx)
		}
	}
}

func (fm *FleetManager) monitorFleetHealth(ctx context.Context) {
	fm.mu.RLock()
	fleets := make([]*Fleet, 0, len(fm.fleets))
	for _, fleet := range fm.fleets {
		fleets = append(fleets, fleet)
	}
	fm.mu.RUnlock()

	for _, fleet := range fleets {
		fm.checkFleetHealth(ctx, fleet)
	}
}

func (fm *FleetManager) checkFleetHealth(ctx context.Context, fleet *Fleet) {
	clusters := fm.getFleetClusters(fleet)

	// Check for failover conditions.
	if fleet.Policy != nil && fleet.Policy.FailoverPolicy.Enabled {
		unhealthyCount := 0
		for _, cluster := range clusters {
			if cluster.Status != ClusterStatusHealthy {
				unhealthyCount++
			}
		}

		if unhealthyCount >= fleet.Policy.FailoverPolicy.FailoverThreshold {
			fm.triggerFailover(ctx, fleet)
			fm.metrics.failoverEvents.WithLabelValues(fleet.ID, "health").Inc()
		}
	}
}

func (fm *FleetManager) triggerFailover(ctx context.Context, fleet *Fleet) {
	fm.logger.Info("Triggering failover", "fleet", fleet.ID)

	// Implementation would:.
	// 1. Identify healthy backup clusters.
	// 2. Migrate workloads.
	// 3. Update DNS/load balancer.
	// 4. Notify operators.
}

func (fm *FleetManager) runAnalytics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fm.stopCh:
			return
		case <-ticker.C:
			fm.updateAnalytics(ctx)
		}
	}
}

func (fm *FleetManager) updateAnalytics(ctx context.Context) {
	fm.mu.RLock()
	fleets := make([]*Fleet, 0, len(fm.fleets))
	for _, fleet := range fm.fleets {
		fleets = append(fleets, fleet)
	}
	fm.mu.RUnlock()

	for _, fleet := range fleets {
		fm.updateFleetAnalytics(ctx, fleet)
	}
}

func (fm *FleetManager) updateFleetAnalytics(ctx context.Context, fleet *Fleet) {
	fm.analytics.mu.Lock()
	defer fm.analytics.mu.Unlock()

	// Update metrics.
	metrics := fm.calculateFleetMetrics(fleet)
	fm.analytics.metrics[fleet.ID] = metrics

	// Update trends.
	trends := fm.analyzeFleetTrends(fleet, metrics)
	fm.analytics.trends[fleet.ID] = trends

	// Update predictions.
	predictions := fm.predictFleetNeeds(fleet, metrics, trends)
	fm.analytics.predictions[fleet.ID] = predictions
}

func (fm *FleetManager) calculateFleetMetrics(fleet *Fleet) *FleetMetrics {
	clusters := fm.getFleetClusters(fleet)

	metrics := &FleetMetrics{
		WorkloadCount: len(fleet.Status.Workloads),
	}

	// Calculate resource utilization.
	if fleet.Status.TotalResources.TotalCPU > 0 {
		metrics.ResourceUtilization = float64(fleet.Status.TotalResources.TotalCPU-fleet.Status.TotalResources.AvailableCPU) /
			float64(fleet.Status.TotalResources.TotalCPU)
	}

	// Calculate availability score.
	if len(clusters) > 0 {
		healthyCount := 0
		for _, cluster := range clusters {
			if cluster.Status == ClusterStatusHealthy {
				healthyCount++
			}
		}
		metrics.AvailabilityScore = float64(healthyCount) / float64(len(clusters))
	}

	// Calculate cost per workload.
	totalCost := 0.0
	for _, cluster := range clusters {
		totalCost += cluster.Metadata.Cost.MonthlyCost
	}
	if metrics.WorkloadCount > 0 {
		metrics.CostPerWorkload = totalCost / float64(metrics.WorkloadCount)
	}

	return metrics
}

func (fm *FleetManager) analyzeFleetTrends(fleet *Fleet, metrics *FleetMetrics) *FleetTrends {
	trends := &FleetTrends{}

	// Analyze resource utilization trend.
	if metrics.ResourceUtilization > 0.8 {
		trends.UtilizationTrend = "increasing"
		trends.RecommendedActions = append(trends.RecommendedActions, "Consider adding more clusters")
	} else if metrics.ResourceUtilization < 0.3 {
		trends.UtilizationTrend = "decreasing"
		trends.RecommendedActions = append(trends.RecommendedActions, "Consider removing underutilized clusters")
	} else {
		trends.UtilizationTrend = "stable"
	}

	// Analyze cost trend.
	if metrics.CostPerWorkload > 1000 {
		trends.CostTrend = "high"
		trends.RecommendedActions = append(trends.RecommendedActions, "Review cost optimization opportunities")
	} else {
		trends.CostTrend = "normal"
	}

	return trends
}

func (fm *FleetManager) predictFleetNeeds(fleet *Fleet, metrics *FleetMetrics, trends *FleetTrends) *FleetPredictions {
	predictions := &FleetPredictions{}

	// Predict future capacity needs based on trends.
	if trends.UtilizationTrend == "increasing" {
		predictions.FutureCapacityNeeds = ResourceCapacity{
			TotalCPU:    int64(float64(fleet.Status.TotalResources.TotalCPU) * 1.3),
			TotalMemory: int64(float64(fleet.Status.TotalResources.TotalMemory) * 1.3),
		}

		predictions.ScalingEvents = append(predictions.ScalingEvents, ScalingEvent{
			Time:      time.Now().Add(7 * 24 * time.Hour),
			Type:      "scale-up",
			Magnitude: 1,
			Reason:    "Predicted capacity shortage",
		})
	}

	// Predict costs.
	predictions.ExpectedCost = metrics.CostPerWorkload * float64(metrics.WorkloadCount) * 1.1

	return predictions
}

func (fm *FleetManager) getFleetClusters(fleet *Fleet) []*ClusterEntry {
	clusters := make([]*ClusterEntry, 0, len(fleet.ClusterIDs))

	for _, clusterID := range fleet.ClusterIDs {
		if cluster, err := fm.registry.GetCluster(clusterID); err == nil {
			clusters = append(clusters, cluster)
		}
	}

	// Also check selector-based membership.
	if fleet.Selector != nil {
		allClusters := fm.registry.ListClusters()
		for _, cluster := range allClusters {
			labelSet := labels.Set(cluster.Metadata.Labels)
			if fleet.Selector.Matches(labelSet) {
				clusters = append(clusters, cluster)
			}
		}
	}

	return clusters
}

func (fm *FleetManager) calculateFleetResources(clusterIDs []string) ResourceCapacity {
	resources := ResourceCapacity{}

	for _, clusterID := range clusterIDs {
		if cluster, err := fm.registry.GetCluster(clusterID); err == nil {
			resources.TotalCPU += cluster.Metadata.Resources.TotalCPU
			resources.AvailableCPU += cluster.Metadata.Resources.AvailableCPU
			resources.TotalMemory += cluster.Metadata.Resources.TotalMemory
			resources.AvailableMemory += cluster.Metadata.Resources.AvailableMemory
			resources.TotalStorage += cluster.Metadata.Resources.TotalStorage
			resources.AvailableStorage += cluster.Metadata.Resources.AvailableStorage
			resources.NodeCount += cluster.Metadata.Resources.NodeCount
			resources.PodCapacity += cluster.Metadata.Resources.PodCapacity
			resources.AvailablePods += cluster.Metadata.Resources.AvailablePods
		}
	}

	if resources.TotalCPU > 0 {
		resources.Utilization = float64(resources.TotalCPU-resources.AvailableCPU) / float64(resources.TotalCPU)
	}

	return resources
}

func (fm *FleetManager) updateWorkloadStatus(ctx context.Context, workload *WorkloadDeployment, clusters []*ClusterEntry) {
	readyReplicas := 0

	for clusterID, replicas := range workload.Distribution {
		// Check cluster health.
		for _, cluster := range clusters {
			if cluster.Metadata.ID == clusterID && cluster.Status == ClusterStatusHealthy {
				readyReplicas += replicas
				break
			}
		}
	}

	workload.Status.ReadyReplicas = readyReplicas
	if readyReplicas == workload.Replicas {
		workload.Status.State = "ready"
	} else if readyReplicas > 0 {
		workload.Status.State = "partial"
	} else {
		workload.Status.State = "unavailable"
	}
}

// WorkloadScheduler methods.

// ScheduleWorkload schedules a workload across clusters based on placement policy.
func (ws *WorkloadScheduler) ScheduleWorkload(ctx context.Context, workload *WorkloadDeployment, clusters []*ClusterEntry, policy *PlacementPolicy) (*PlacementDecision, error) {
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters available for scheduling")
	}

	// Filter clusters based on constraints.
	eligible := ws.filterEligibleClusters(clusters, workload, policy)
	if len(eligible) == 0 {
		return nil, fmt.Errorf("no eligible clusters after applying constraints")
	}

	// Apply placement strategy.
	decision, err := ws.optimizer.Optimize(workload, eligible, policy.Strategy)
	if err != nil {
		return nil, fmt.Errorf("placement optimization failed: %w", err)
	}

	// Apply affinity rules.
	decision = ws.applyAffinityRules(decision, eligible, policy.AffinityRules)

	// Apply anti-affinity rules.
	decision = ws.applyAntiAffinityRules(decision, eligible, policy.AntiAffinityRules)

	// Apply spread constraints.
	decision = ws.applySpreadConstraints(decision, eligible, policy.SpreadConstraints)

	return decision, nil
}

func (ws *WorkloadScheduler) filterEligibleClusters(clusters []*ClusterEntry, workload *WorkloadDeployment, policy *PlacementPolicy) []*ClusterEntry {
	eligible := []*ClusterEntry{}

	for _, cluster := range clusters {
		// Check cluster health.
		if cluster.Status != ClusterStatusHealthy {
			continue
		}

		// Check resource availability.
		if !ws.hasRequiredResources(cluster, workload) {
			continue
		}

		// Check placement preferences.
		if !ws.matchesPreferences(cluster, policy.Preferences) {
			continue
		}

		eligible = append(eligible, cluster)
	}

	return eligible
}

func (ws *WorkloadScheduler) hasRequiredResources(cluster *ClusterEntry, workload *WorkloadDeployment) bool {
	// Check CPU.
	if workload.Resources.Requests != nil {
		if cpu, ok := workload.Resources.Requests["cpu"]; ok {
			if cluster.Metadata.Resources.AvailableCPU < cpu.Value() {
				return false
			}
		}
		// Check memory.
		if memory, ok := workload.Resources.Requests["memory"]; ok {
			if cluster.Metadata.Resources.AvailableMemory < memory.Value() {
				return false
			}
		}
	}

	return true
}

func (ws *WorkloadScheduler) matchesPreferences(cluster *ClusterEntry, preferences []PlacementPreference) bool {
	for _, pref := range preferences {
		switch pref.Type {
		case "region":
			if cluster.Metadata.Region != pref.Value {
				return false
			}
		case "provider":
			if string(cluster.Metadata.Provider) != pref.Value {
				return false
			}
		case "capability":
			hasCapability := false
			for _, cap := range cluster.Metadata.Capabilities {
				if string(cap) == pref.Value {
					hasCapability = true
					break
				}
			}
			if !hasCapability {
				return false
			}
		}
	}

	return true
}

func (ws *WorkloadScheduler) applyAffinityRules(decision *PlacementDecision, clusters []*ClusterEntry, rules []AffinityRule) *PlacementDecision {
	// Implementation would apply affinity rules to favor certain clusters.
	return decision
}

func (ws *WorkloadScheduler) applyAntiAffinityRules(decision *PlacementDecision, clusters []*ClusterEntry, rules []AntiAffinityRule) *PlacementDecision {
	// Implementation would apply anti-affinity rules to avoid certain clusters.
	return decision
}

func (ws *WorkloadScheduler) applySpreadConstraints(decision *PlacementDecision, clusters []*ClusterEntry, constraints []SpreadConstraint) *PlacementDecision {
	// Implementation would spread workloads according to constraints.
	return decision
}

// PlacementOptimizer methods.

// NewPlacementOptimizer creates a new placement optimizer.
func NewPlacementOptimizer(logger logr.Logger) *PlacementOptimizer {
	po := &PlacementOptimizer{
		strategies: make(map[PlacementStrategy]PlacementFunction),
		logger:     logger,
	}

	// Register placement strategies.
	po.strategies[PlacementStrategyBinPacking] = po.binPackingStrategy
	po.strategies[PlacementStrategySpread] = po.spreadStrategy
	po.strategies[PlacementStrategyCostOptimized] = po.costOptimizedStrategy
	po.strategies[PlacementStrategyLatencyOptimized] = po.latencyOptimizedStrategy
	po.strategies[PlacementStrategyBalanced] = po.balancedStrategy

	return po
}

// Optimize optimizes workload placement using the specified strategy.
func (po *PlacementOptimizer) Optimize(workload *WorkloadDeployment, clusters []*ClusterEntry, strategy PlacementStrategy) (*PlacementDecision, error) {
	fn, exists := po.strategies[strategy]
	if !exists {
		return nil, fmt.Errorf("unknown placement strategy: %s", strategy)
	}

	return fn(workload, clusters)
}

func (po *PlacementOptimizer) binPackingStrategy(workload *WorkloadDeployment, clusters []*ClusterEntry) (*PlacementDecision, error) {
	// Sort clusters by available resources (ascending).
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Metadata.Resources.AvailableCPU < clusters[j].Metadata.Resources.AvailableCPU
	})

	distribution := make(map[string]int)
	remainingReplicas := workload.Replicas

	// Pack replicas into clusters with least available resources first.
	for _, cluster := range clusters {
		if remainingReplicas == 0 {
			break
		}

		// Calculate how many replicas this cluster can handle.
		capacity := po.calculateClusterCapacity(cluster, workload)
		replicas := int(math.Min(float64(capacity), float64(remainingReplicas)))

		if replicas > 0 {
			distribution[cluster.Metadata.ID] = replicas
			remainingReplicas -= replicas
		}
	}

	if remainingReplicas > 0 {
		return nil, fmt.Errorf("insufficient cluster capacity for %d replicas", remainingReplicas)
	}

	return &PlacementDecision{
		ClusterDistribution: distribution,
		Score:               0.8,
		Reasons:             []string{"Bin packing strategy applied"},
	}, nil
}

func (po *PlacementOptimizer) spreadStrategy(workload *WorkloadDeployment, clusters []*ClusterEntry) (*PlacementDecision, error) {
	distribution := make(map[string]int)

	// Spread replicas evenly across all clusters.
	replicasPerCluster := workload.Replicas / len(clusters)
	remainder := workload.Replicas % len(clusters)

	for i, cluster := range clusters {
		replicas := replicasPerCluster
		if i < remainder {
			replicas++
		}
		if replicas > 0 {
			distribution[cluster.Metadata.ID] = replicas
		}
	}

	return &PlacementDecision{
		ClusterDistribution: distribution,
		Score:               0.9,
		Reasons:             []string{"Spread strategy applied for high availability"},
	}, nil
}

func (po *PlacementOptimizer) costOptimizedStrategy(workload *WorkloadDeployment, clusters []*ClusterEntry) (*PlacementDecision, error) {
	// Sort clusters by cost (ascending).
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Metadata.Cost.HourlyCost < clusters[j].Metadata.Cost.HourlyCost
	})

	distribution := make(map[string]int)
	remainingReplicas := workload.Replicas

	// Place replicas on cheapest clusters first.
	for _, cluster := range clusters {
		if remainingReplicas == 0 {
			break
		}

		capacity := po.calculateClusterCapacity(cluster, workload)
		replicas := int(math.Min(float64(capacity), float64(remainingReplicas)))

		if replicas > 0 {
			distribution[cluster.Metadata.ID] = replicas
			remainingReplicas -= replicas
		}
	}

	if remainingReplicas > 0 {
		return nil, fmt.Errorf("insufficient cluster capacity for %d replicas", remainingReplicas)
	}

	return &PlacementDecision{
		ClusterDistribution: distribution,
		Score:               0.85,
		Reasons:             []string{"Cost-optimized placement strategy applied"},
	}, nil
}

func (po *PlacementOptimizer) latencyOptimizedStrategy(workload *WorkloadDeployment, clusters []*ClusterEntry) (*PlacementDecision, error) {
	// Group clusters by region.
	clustersByRegion := make(map[string][]*ClusterEntry)
	for _, cluster := range clusters {
		clustersByRegion[cluster.Metadata.Region] = append(clustersByRegion[cluster.Metadata.Region], cluster)
	}

	distribution := make(map[string]int)
	replicasPerRegion := workload.Replicas / len(clustersByRegion)
	remainder := workload.Replicas % len(clustersByRegion)

	regionIndex := 0
	for _, regionClusters := range clustersByRegion {
		regionReplicas := replicasPerRegion
		if regionIndex < remainder {
			regionReplicas++
		}

		// Distribute within region.
		for _, cluster := range regionClusters {
			if regionReplicas == 0 {
				break
			}
			distribution[cluster.Metadata.ID] = 1
			regionReplicas--
		}

		regionIndex++
	}

	return &PlacementDecision{
		ClusterDistribution: distribution,
		Score:               0.95,
		Reasons:             []string{"Latency-optimized placement for geographic distribution"},
	}, nil
}

func (po *PlacementOptimizer) balancedStrategy(workload *WorkloadDeployment, clusters []*ClusterEntry) (*PlacementDecision, error) {
	// Score each cluster based on multiple factors.
	scores := make(map[string]float64)
	for _, cluster := range clusters {
		// Resource availability (40%).
		resourceScore := (1 - cluster.Metadata.Resources.Utilization) * 0.4

		// Cost efficiency (30%).
		maxCost := 100.0 // Assume max hourly cost
		costScore := (1 - cluster.Metadata.Cost.HourlyCost/maxCost) * 0.3

		// Health and reliability (30%).
		healthScore := 0.3
		if cluster.Status == ClusterStatusHealthy {
			healthScore = 0.3
		} else if cluster.Status == ClusterStatusDegraded {
			healthScore = 0.15
		} else {
			healthScore = 0.0
		}

		scores[cluster.Metadata.ID] = resourceScore + costScore + healthScore
	}

	// Sort clusters by score (descending).
	sort.Slice(clusters, func(i, j int) bool {
		return scores[clusters[i].Metadata.ID] > scores[clusters[j].Metadata.ID]
	})

	distribution := make(map[string]int)
	remainingReplicas := workload.Replicas

	// Distribute based on scores.
	for _, cluster := range clusters {
		if remainingReplicas == 0 {
			break
		}

		// Allocate replicas proportional to score.
		capacity := po.calculateClusterCapacity(cluster, workload)
		idealReplicas := int(float64(workload.Replicas) * scores[cluster.Metadata.ID])
		replicas := int(math.Min(float64(capacity), math.Min(float64(idealReplicas), float64(remainingReplicas))))

		if replicas > 0 {
			distribution[cluster.Metadata.ID] = replicas
			remainingReplicas -= replicas
		}
	}

	// Distribute any remaining replicas.
	for _, cluster := range clusters {
		if remainingReplicas == 0 {
			break
		}
		if current, exists := distribution[cluster.Metadata.ID]; exists {
			distribution[cluster.Metadata.ID] = current + 1
			remainingReplicas--
		}
	}

	return &PlacementDecision{
		ClusterDistribution: distribution,
		Score:               0.88,
		Reasons:             []string{"Balanced placement considering resources, cost, and health"},
	}, nil
}

func (po *PlacementOptimizer) calculateClusterCapacity(cluster *ClusterEntry, workload *WorkloadDeployment) int {
	// Calculate how many replicas the cluster can handle.
	capacity := cluster.Metadata.Resources.AvailablePods

	// Check CPU capacity.
	if workload.Resources.Requests != nil {
		if cpu, ok := workload.Resources.Requests["cpu"]; ok && cpu.Value() > 0 {
			cpuCapacity := int(cluster.Metadata.Resources.AvailableCPU / cpu.Value())
			capacity = int(math.Min(float64(capacity), float64(cpuCapacity)))
		}

		// Check memory capacity.
		if memory, ok := workload.Resources.Requests["memory"]; ok && memory.Value() > 0 {
			memCapacity := int(cluster.Metadata.Resources.AvailableMemory / memory.Value())
			capacity = int(math.Min(float64(capacity), float64(memCapacity)))
		}
	}

	return capacity
}
