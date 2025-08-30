// Nephoran Intent Operator - Global Traffic Controller.

// Phase 4 Enterprise Architecture - Intelligent Multi-Region Routing.

package global

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TrafficController manages global traffic routing and load balancing.

type TrafficController struct {
	client client.Client

	kubeClient kubernetes.Interface

	prometheusAPI v1.API

	logger logr.Logger

	config *TrafficControllerConfig

	regions map[string]*RegionInfo

	routingDecisions chan *RoutingDecision

	healthChecks map[string]*RegionHealth

	mutex sync.RWMutex
}

// TrafficControllerConfig defines configuration for global traffic management.

type TrafficControllerConfig struct {

	// Global routing strategy.

	RoutingStrategy string `json:"routing_strategy"` // latency, capacity, cost, hybrid

	HealthCheckInterval time.Duration `json:"health_check_interval"` // 30s

	RoutingDecisionInterval time.Duration `json:"routing_decision_interval"` // 60s

	// Load balancing configuration.

	LoadBalancingAlgorithm string `json:"load_balancing_algorithm"` // weighted, least_connections, round_robin

	TrafficShiftPercentage int `json:"traffic_shift_percentage"` // 10% maximum shift per interval

	// Failover configuration.

	FailoverEnabled bool `json:"failover_enabled"`

	FailoverThreshold float64 `json:"failover_threshold"` // 0.95 (95% success rate)

	AutoRecoveryEnabled bool `json:"auto_recovery_enabled"`

	RecoveryThreshold float64 `json:"recovery_threshold"` // 0.98 (98% success rate)

	// Geographic routing.

	GeographicRoutingEnabled bool `json:"geographic_routing_enabled"`

	LatencyThreshold time.Duration `json:"latency_threshold"` // 200ms

	// Cost optimization.

	CostOptimizationEnabled bool `json:"cost_optimization_enabled"`

	CostWeight float64 `json:"cost_weight"` // 0.3 (30% weight)

	PerformanceWeight float64 `json:"performance_weight"` // 0.7 (70% weight)

}

// RegionInfo contains information about a specific region.

type RegionInfo struct {
	Name string `json:"name"`

	Endpoint string `json:"endpoint"`

	Priority int `json:"priority"`

	CurrentCapacity *RegionCapacity `json:"current_capacity"`

	AvailabilityZones []string `json:"availability_zones"`

	Features []string `json:"features"`

	GeographicLocation *GeographicLocation `json:"geographic_location"`

	CostProfile *CostProfile `json:"cost_profile"`
}

// RegionCapacity tracks current resource utilization.

type RegionCapacity struct {
	MaxIntentsPerMinute int `json:"max_intents_per_minute"`

	CurrentIntentsPerMinute int `json:"current_intents_per_minute"`

	MaxConcurrentDeployments int `json:"max_concurrent_deployments"`

	CurrentConcurrentDeployments int `json:"current_concurrent_deployments"`

	CPUUtilization float64 `json:"cpu_utilization"`

	MemoryUtilization float64 `json:"memory_utilization"`

	NetworkUtilization float64 `json:"network_utilization"`
}

// RegionHealth tracks health metrics for each region.

type RegionHealth struct {
	Region string `json:"region"`

	IsHealthy bool `json:"is_healthy"`

	SuccessRate float64 `json:"success_rate"`

	AverageLatency time.Duration `json:"average_latency"`

	P95Latency time.Duration `json:"p95_latency"`

	ErrorRate float64 `json:"error_rate"`

	LastHealthCheck time.Time `json:"last_health_check"`

	ConsecutiveFailures int `json:"consecutive_failures"`

	AvailabilityScore float64 `json:"availability_score"`
}

// GeographicLocation defines geographic coordinates and regions.

type GeographicLocation struct {
	Continent string `json:"continent"`

	Country string `json:"country"`

	Latitude float64 `json:"latitude"`

	Longitude float64 `json:"longitude"`

	TimezoneOffset int `json:"timezone_offset"`
}

// CostProfile tracks cost metrics for each region.

type CostProfile struct {
	ComputeCostPerHour float64 `json:"compute_cost_per_hour"`

	NetworkCostPerGB float64 `json:"network_cost_per_gb"`

	StorageCostPerGB float64 `json:"storage_cost_per_gb"`

	LLMCostPerRequest float64 `json:"llm_cost_per_request"`

	TotalCostLast24h float64 `json:"total_cost_last_24h"`
}

// RoutingDecision represents a traffic routing decision.

type RoutingDecision struct {
	Timestamp time.Time `json:"timestamp"`

	SourceRegion string `json:"source_region,omitempty"`

	TargetRegions []WeightedRegion `json:"target_regions"`

	DecisionReason string `json:"decision_reason"`

	ExpectedLatency time.Duration `json:"expected_latency"`

	ExpectedCost float64 `json:"expected_cost"`

	ConfidenceScore float64 `json:"confidence_score"`
}

// WeightedRegion represents a region with its traffic weight.

type WeightedRegion struct {
	Region string `json:"region"`

	Weight int `json:"weight"`

	Reason string `json:"reason"`
}

// NewTrafficController creates a new global traffic controller.

func NewTrafficController(

	client client.Client,

	kubeClient kubernetes.Interface,

	prometheusClient api.Client,

	config *TrafficControllerConfig,

	logger logr.Logger,

) *TrafficController {

	return &TrafficController{

		client: client,

		kubeClient: kubeClient,

		prometheusAPI: v1.NewAPI(prometheusClient),

		logger: logger,

		config: config,

		regions: make(map[string]*RegionInfo),

		routingDecisions: make(chan *RoutingDecision, 100),

		healthChecks: make(map[string]*RegionHealth),
	}

}

// Start begins the traffic controller operations.

func (tc *TrafficController) Start(ctx context.Context) error {

	tc.logger.Info("Starting Global Traffic Controller")

	// Initialize regions from configuration.

	if err := tc.loadRegionConfiguration(ctx); err != nil {

		return fmt.Errorf("failed to load region configuration: %w", err)

	}

	// Start health checking goroutine.

	go tc.runHealthChecks(ctx)

	// Start routing decision engine.

	go tc.runRoutingEngine(ctx)

	// Start metrics collection.

	go tc.collectRegionalMetrics(ctx)

	// Start routing decision processor.

	go tc.processRoutingDecisions(ctx)

	tc.logger.Info("Global Traffic Controller started successfully")

	return nil

}

// loadRegionConfiguration loads region information from ConfigMap.

func (tc *TrafficController) loadRegionConfiguration(ctx context.Context) error {

	configMap, err := tc.kubeClient.CoreV1().ConfigMaps("nephoran-global").Get(

		ctx, "nephoran-global-config", metav1.GetOptions{},
	)

	if err != nil {

		return fmt.Errorf("failed to get global config: %w", err)

	}

	var config struct {
		Regions map[string]*RegionInfo `yaml:"regions"`
	}

	if err := json.Unmarshal([]byte(configMap.Data["regions.yaml"]), &config); err != nil {

		return fmt.Errorf("failed to parse regions config: %w", err)

	}

	tc.mutex.Lock()

	tc.regions = config.Regions

	tc.mutex.Unlock()

	tc.logger.Info("Loaded region configuration", "regions", len(config.Regions))

	return nil

}

// runHealthChecks performs periodic health checks for all regions.

func (tc *TrafficController) runHealthChecks(ctx context.Context) {

	ticker := time.NewTicker(tc.config.HealthCheckInterval)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			tc.performHealthChecks(ctx)

		}

	}

}

// performHealthChecks checks health status of all regions.

func (tc *TrafficController) performHealthChecks(ctx context.Context) {

	tc.mutex.RLock()

	regions := make(map[string]*RegionInfo)

	for k, v := range tc.regions {

		regions[k] = v

	}

	tc.mutex.RUnlock()

	var wg sync.WaitGroup

	for regionName, region := range regions {

		wg.Add(1)

		go func(name string, r *RegionInfo) {

			defer wg.Done()

			health := tc.checkRegionHealth(ctx, name, r)

			tc.mutex.Lock()

			tc.healthChecks[name] = health

			tc.mutex.Unlock()

		}(regionName, region)

	}

	wg.Wait()

	tc.logger.V(1).Info("Completed health checks for all regions")

}

// checkRegionHealth performs health check for a specific region.

func (tc *TrafficController) checkRegionHealth(ctx context.Context, regionName string, region *RegionInfo) *RegionHealth {

	health := &RegionHealth{

		Region: regionName,

		LastHealthCheck: time.Now(),
	}

	// HTTP health check.

	client := &http.Client{Timeout: 10 * time.Second}

	// Security fix (bodyclose): Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, region.Endpoint+"/health", http.NoBody)
	if err != nil {
		health.IsHealthy = false
		if existing, ok := tc.healthChecks[regionName]; ok {
			health.ConsecutiveFailures = existing.ConsecutiveFailures + 1
		} else {
			health.ConsecutiveFailures = 1
		}
		tc.logger.V(1).Info("Failed to create health check request",
			"region", regionName, "error", err)
		return health
	}

	resp, err := client.Do(req)
	if err != nil {
		health.IsHealthy = false
		if existing, ok := tc.healthChecks[regionName]; ok {
			health.ConsecutiveFailures = existing.ConsecutiveFailures + 1
		} else {
			health.ConsecutiveFailures = 1
		}
		tc.logger.V(1).Info("Region health check failed",
			"region", regionName, "error", err)
		return health
	}
	// Security fix (bodyclose): Always close response body
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		health.IsHealthy = false
		if existing, ok := tc.healthChecks[regionName]; ok {
			health.ConsecutiveFailures = existing.ConsecutiveFailures + 1
		} else {
			health.ConsecutiveFailures = 1
		}
		tc.logger.V(1).Info("Region health check returned non-200 status",
			"region", regionName, "status", resp.StatusCode)
		return health
	}

	// Query metrics from Prometheus.

	successRate, err := tc.querySuccessRate(ctx, regionName)

	if err != nil {

		tc.logger.Error(err, "Failed to query success rate", "region", regionName)

		successRate = 0.0

	}

	avgLatency, err := tc.queryAverageLatency(ctx, regionName)

	if err != nil {

		tc.logger.Error(err, "Failed to query average latency", "region", regionName)

		avgLatency = time.Second

	}

	p95Latency, err := tc.queryP95Latency(ctx, regionName)

	if err != nil {

		tc.logger.Error(err, "Failed to query P95 latency", "region", regionName)

		p95Latency = 2 * time.Second

	}

	// Update health metrics.

	health.IsHealthy = successRate >= tc.config.FailoverThreshold

	health.SuccessRate = successRate

	health.AverageLatency = avgLatency

	health.P95Latency = p95Latency

	health.ErrorRate = 1.0 - successRate

	health.ConsecutiveFailures = 0

	health.AvailabilityScore = tc.calculateAvailabilityScore(health)

	return health

}

// querySuccessRate queries success rate from Prometheus.

func (tc *TrafficController) querySuccessRate(ctx context.Context, region string) (float64, error) {

	query := fmt.Sprintf(

		`rate(nephoran_requests_total{region="%s",status="success"}[5m]) / rate(nephoran_requests_total{region="%s"}[5m])`,

		region, region,
	)

	result, _, err := tc.prometheusAPI.Query(ctx, query, time.Now())

	if err != nil {

		return 0, err

	}

	if vector, ok := result.(model.Vector); ok && len(vector) > 0 {

		return float64(vector[0].Value), nil

	}

	return 1.0, nil // Default to 100% if no data

}

// queryAverageLatency queries average latency from Prometheus.

func (tc *TrafficController) queryAverageLatency(ctx context.Context, region string) (time.Duration, error) {

	query := fmt.Sprintf(

		`rate(nephoran_request_duration_seconds_sum{region="%s"}[5m]) / rate(nephoran_request_duration_seconds_count{region="%s"}[5m])`,

		region, region,
	)

	result, _, err := tc.prometheusAPI.Query(ctx, query, time.Now())

	if err != nil {

		return 0, err

	}

	if vector, ok := result.(model.Vector); ok && len(vector) > 0 {

		return time.Duration(float64(vector[0].Value) * float64(time.Second)), nil

	}

	return 100 * time.Millisecond, nil

}

// queryP95Latency queries P95 latency from Prometheus.

func (tc *TrafficController) queryP95Latency(ctx context.Context, region string) (time.Duration, error) {

	query := fmt.Sprintf(

		`histogram_quantile(0.95, rate(nephoran_request_duration_seconds_bucket{region="%s"}[5m]))`,

		region,
	)

	result, _, err := tc.prometheusAPI.Query(ctx, query, time.Now())

	if err != nil {

		return 0, err

	}

	if vector, ok := result.(model.Vector); ok && len(vector) > 0 {

		return time.Duration(float64(vector[0].Value) * float64(time.Second)), nil

	}

	return 500 * time.Millisecond, nil

}

// calculateAvailabilityScore calculates an overall availability score.

func (tc *TrafficController) calculateAvailabilityScore(health *RegionHealth) float64 {

	// Weighted score based on multiple factors.

	successWeight := 0.5

	latencyWeight := 0.3

	errorWeight := 0.2

	// Normalize latency score (lower is better).

	latencyScore := 1.0 - (float64(health.AverageLatency.Milliseconds()) / 1000.0)

	if latencyScore < 0 {

		latencyScore = 0

	}

	errorScore := 1.0 - health.ErrorRate

	return (health.SuccessRate * successWeight) +

		(latencyScore * latencyWeight) +

		(errorScore * errorWeight)

}

// runRoutingEngine runs the intelligent routing decision engine.

func (tc *TrafficController) runRoutingEngine(ctx context.Context) {

	ticker := time.NewTicker(tc.config.RoutingDecisionInterval)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			decision := tc.makeRoutingDecision(ctx)

			if decision != nil {

				select {

				case tc.routingDecisions <- decision:

				default:

					tc.logger.V(1).Info("Routing decision channel full, dropping decision")

				}

			}

		}

	}

}

// makeRoutingDecision creates an intelligent routing decision.

func (tc *TrafficController) makeRoutingDecision(ctx context.Context) *RoutingDecision {

	tc.mutex.RLock()

	healthChecks := make(map[string]*RegionHealth)

	for k, v := range tc.healthChecks {

		healthChecks[k] = v

	}

	regions := make(map[string]*RegionInfo)

	for k, v := range tc.regions {

		regions[k] = v

	}

	tc.mutex.RUnlock()

	if len(healthChecks) == 0 {

		return nil

	}

	// Filter healthy regions.

	healthyRegions := make(map[string]*RegionHealth)

	for name, health := range healthChecks {

		if health.IsHealthy {

			healthyRegions[name] = health

		}

	}

	if len(healthyRegions) == 0 {

		tc.logger.Error(nil, "No healthy regions available for routing")

		return nil

	}

	// Calculate routing weights based on strategy.

	weightedRegions := tc.calculateRoutingWeights(healthyRegions, regions)

	decision := &RoutingDecision{

		Timestamp: time.Now(),

		TargetRegions: weightedRegions,

		DecisionReason: fmt.Sprintf("Strategy: %s", tc.config.RoutingStrategy),

		ConfidenceScore: tc.calculateConfidenceScore(healthyRegions),
	}

	// Calculate expected metrics.

	decision.ExpectedLatency = tc.calculateExpectedLatency(healthyRegions, weightedRegions)

	decision.ExpectedCost = tc.calculateExpectedCost(regions, weightedRegions)

	return decision

}

// calculateRoutingWeights calculates traffic weights based on configured strategy.

func (tc *TrafficController) calculateRoutingWeights(

	healthyRegions map[string]*RegionHealth,

	regions map[string]*RegionInfo,

) []WeightedRegion {

	var weightedRegions []WeightedRegion

	totalWeight := 0

	switch tc.config.RoutingStrategy {

	case "latency":

		// Route based on lowest latency.

		type regionLatency struct {
			region string

			latency time.Duration
		}

		var regionLatencies []regionLatency

		for name, health := range healthyRegions {

			regionLatencies = append(regionLatencies, regionLatency{

				region: name,

				latency: health.AverageLatency,
			})

		}

		sort.Slice(regionLatencies, func(i, j int) bool {

			return regionLatencies[i].latency < regionLatencies[j].latency

		})

		// Assign weights inversely proportional to latency.

		for i, rl := range regionLatencies {

			weight := len(regionLatencies) - i

			weightedRegions = append(weightedRegions, WeightedRegion{

				Region: rl.region,

				Weight: weight * 20, // Scale up

				Reason: fmt.Sprintf("Low latency: %v", rl.latency),
			})

			totalWeight += weight * 20

		}

	case "capacity":

		// Route based on available capacity.

		for name, region := range regions {

			if _, ok := healthyRegions[name]; ok {

				if region.CurrentCapacity != nil {

					utilizationScore := 1.0 - (float64(region.CurrentCapacity.CPUUtilization) / 100.0)

					weight := int(utilizationScore * 100)

					if weight > 0 {

						weightedRegions = append(weightedRegions, WeightedRegion{

							Region: name,

							Weight: weight,

							Reason: fmt.Sprintf("Available capacity: %.1f%%", utilizationScore*100),
						})

						totalWeight += weight

					}

				}

			}

		}

	default: // hybrid and any other strategy

		// Hybrid approach considering latency, capacity, and success rate.

		for name, health := range healthyRegions {

			region := regions[name]

			// Latency score (normalized).

			latencyScore := 1.0 - (float64(health.AverageLatency.Milliseconds()) / 1000.0)

			if latencyScore < 0 {

				latencyScore = 0

			}

			// Capacity score.

			capacityScore := 1.0

			if region.CurrentCapacity != nil {

				capacityScore = 1.0 - (region.CurrentCapacity.CPUUtilization / 100.0)

			}

			// Combined score.

			combinedScore := (health.SuccessRate * 0.4) +

				(latencyScore * 0.3) +

				(capacityScore * 0.3)

			weight := int(combinedScore * 100)

			if weight > 0 {

				weightedRegions = append(weightedRegions, WeightedRegion{

					Region: name,

					Weight: weight,

					Reason: fmt.Sprintf("Hybrid score: %.2f", combinedScore),
				})

				totalWeight += weight

			}

		}

	}

	// Normalize weights to sum to 100.

	if totalWeight > 0 {

		for i := range weightedRegions {

			weightedRegions[i].Weight = (weightedRegions[i].Weight * 100) / totalWeight

		}

	}

	return weightedRegions

}

// calculateConfidenceScore calculates confidence in the routing decision.

func (tc *TrafficController) calculateConfidenceScore(healthyRegions map[string]*RegionHealth) float64 {

	if len(healthyRegions) == 0 {

		return 0.0

	}

	totalScore := 0.0

	for _, health := range healthyRegions {

		totalScore += health.AvailabilityScore

	}

	avgScore := totalScore / float64(len(healthyRegions))

	diversityBonus := float64(len(healthyRegions)) / 10.0 // Bonus for having multiple healthy regions

	confidence := avgScore + diversityBonus

	if confidence > 1.0 {

		confidence = 1.0

	}

	return confidence

}

// calculateExpectedLatency calculates expected latency for the routing decision.

func (tc *TrafficController) calculateExpectedLatency(

	healthyRegions map[string]*RegionHealth,

	weightedRegions []WeightedRegion,

) time.Duration {

	totalWeightedLatency := float64(0)

	totalWeight := 0

	for _, wr := range weightedRegions {

		if health, ok := healthyRegions[wr.Region]; ok {

			totalWeightedLatency += float64(health.AverageLatency.Nanoseconds()) * float64(wr.Weight)

			totalWeight += wr.Weight

		}

	}

	if totalWeight == 0 {

		return time.Second

	}

	avgLatency := totalWeightedLatency / float64(totalWeight)

	return time.Duration(avgLatency)

}

// calculateExpectedCost calculates expected cost for the routing decision.

func (tc *TrafficController) calculateExpectedCost(

	regions map[string]*RegionInfo,

	weightedRegions []WeightedRegion,

) float64 {

	totalWeightedCost := 0.0

	totalWeight := 0

	for _, wr := range weightedRegions {

		if region, ok := regions[wr.Region]; ok && region.CostProfile != nil {

			totalWeightedCost += region.CostProfile.LLMCostPerRequest * float64(wr.Weight)

			totalWeight += wr.Weight

		}

	}

	if totalWeight == 0 {

		return 0.01 // Default cost

	}

	return totalWeightedCost / float64(totalWeight)

}

// collectRegionalMetrics collects metrics from all regions.

func (tc *TrafficController) collectRegionalMetrics(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			tc.updateRegionalCapacity(ctx)

		}

	}

}

// updateRegionalCapacity updates capacity information for all regions.

func (tc *TrafficController) updateRegionalCapacity(ctx context.Context) {

	tc.mutex.RLock()

	regions := make(map[string]*RegionInfo)

	for k, v := range tc.regions {

		regions[k] = v

	}

	tc.mutex.RUnlock()

	for regionName := range regions {

		capacity, err := tc.queryRegionalCapacity(ctx, regionName)

		if err != nil {

			tc.logger.Error(err, "Failed to query regional capacity", "region", regionName)

			continue

		}

		tc.mutex.Lock()

		if tc.regions[regionName] != nil {

			tc.regions[regionName].CurrentCapacity = capacity

		}

		tc.mutex.Unlock()

	}

}

// queryRegionalCapacity queries capacity metrics for a specific region.

func (tc *TrafficController) queryRegionalCapacity(ctx context.Context, region string) (*RegionCapacity, error) {

	capacity := &RegionCapacity{}

	// Query current intent processing rate.

	query := fmt.Sprintf(`rate(nephoran_intents_processed_total{region="%s"}[1m]) * 60`, region)

	result, _, err := tc.prometheusAPI.Query(ctx, query, time.Now())

	if err == nil {

		if vector, ok := result.(model.Vector); ok && len(vector) > 0 {

			capacity.CurrentIntentsPerMinute = int(vector[0].Value)

		}

	}

	// Query CPU utilization.

	query = fmt.Sprintf(`avg(rate(container_cpu_usage_seconds_total{namespace="nephoran-system",region="%s"}[5m])) * 100`, region)

	result, _, err = tc.prometheusAPI.Query(ctx, query, time.Now())

	if err == nil {

		if vector, ok := result.(model.Vector); ok && len(vector) > 0 {

			capacity.CPUUtilization = float64(vector[0].Value)

		}

	}

	// Query memory utilization.

	query = fmt.Sprintf(`avg(container_memory_usage_bytes{namespace="nephoran-system",region="%s"}) / avg(container_spec_memory_limit_bytes{namespace="nephoran-system",region="%s"}) * 100`, region, region)

	result, _, err = tc.prometheusAPI.Query(ctx, query, time.Now())

	if err == nil {

		if vector, ok := result.(model.Vector); ok && len(vector) > 0 {

			capacity.MemoryUtilization = float64(vector[0].Value)

		}

	}

	return capacity, nil

}

// processRoutingDecisions processes routing decisions and applies them.

func (tc *TrafficController) processRoutingDecisions(ctx context.Context) {

	for {

		select {

		case <-ctx.Done():

			return

		case decision := <-tc.routingDecisions:

			if err := tc.applyRoutingDecision(ctx, decision); err != nil {

				tc.logger.Error(err, "Failed to apply routing decision")

			}

		}

	}

}

// applyRoutingDecision applies a routing decision by updating Istio configuration.

func (tc *TrafficController) applyRoutingDecision(ctx context.Context, decision *RoutingDecision) error {

	tc.logger.Info("Applying routing decision",

		"regions", len(decision.TargetRegions),

		"reason", decision.DecisionReason,

		"confidence", decision.ConfidenceScore)

	// Create VirtualService update.

	virtualServiceManifest := tc.generateVirtualServiceManifest(decision)

	// Apply to cluster (this would be implemented based on your controller-runtime setup).

	// For now, we'll log the decision.

	tc.logger.Info("Generated VirtualService manifest",

		"manifest_size", len(virtualServiceManifest),

		"expected_latency", decision.ExpectedLatency,

		"expected_cost", decision.ExpectedCost)

	return nil

}

// generateVirtualServiceManifest generates Istio VirtualService YAML for routing decision.

func (tc *TrafficController) generateVirtualServiceManifest(decision *RoutingDecision) string {

	// This would generate the actual Istio VirtualService YAML.

	// based on the routing decision.

	manifest := "apiVersion: networking.istio.io/v1beta1\nkind: VirtualService\n"

	manifest += "metadata:\n  name: global-traffic-routing\n"

	manifest += "spec:\n  http:\n  - route:\n"

	for _, region := range decision.TargetRegions {

		manifest += fmt.Sprintf("    - destination:\n        host: %s.nephoran.local\n      weight: %d\n",

			region.Region, region.Weight)

	}

	return manifest

}

// GetHealthStatus returns current health status of all regions.

func (tc *TrafficController) GetHealthStatus() map[string]*RegionHealth {

	tc.mutex.RLock()

	defer tc.mutex.RUnlock()

	result := make(map[string]*RegionHealth)

	for k, v := range tc.healthChecks {

		result[k] = v

	}

	return result

}

// GetLatestRoutingDecision returns the most recent routing decision.

func (tc *TrafficController) GetLatestRoutingDecision() *RoutingDecision {

	// In a real implementation, you'd store the latest decision.

	// For now, return nil.

	return nil

}
