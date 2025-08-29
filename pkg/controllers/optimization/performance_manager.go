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

package optimization

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers/interfaces"
)

// PerformanceManager handles performance optimization and scaling decisions.

type PerformanceManager struct {
	logger logr.Logger

	// Resource monitoring.

	resourceMonitor *ResourceMonitor

	metricsAnalyzer *MetricsAnalyzer

	scalingDecision *ScalingDecisionEngine

	// Caching.

	cacheManager *MultiLevelCacheManager

	// Connection pooling.

	connectionManager *ConnectionPoolManager

	// Load balancing.

	loadBalancer *LoadBalancer

	// Performance profiles.

	profiles map[string]*PerformanceProfile

	// Configuration.

	config *PerformanceConfig

	// State.

	started bool

	stopChan chan bool

	mutex sync.RWMutex
}

// PerformanceConfig holds performance optimization configuration.

type PerformanceConfig struct {

	// Monitoring intervals.

	MetricsCollectionInterval time.Duration `json:"metricsCollectionInterval"`

	AnalysisInterval time.Duration `json:"analysisInterval"`

	// Scaling thresholds.

	CPUScaleUpThreshold float64 `json:"cpuScaleUpThreshold"`

	CPUScaleDownThreshold float64 `json:"cpuScaleDownThreshold"`

	MemoryScaleUpThreshold float64 `json:"memoryScaleUpThreshold"`

	MemoryScaleDownThreshold float64 `json:"memoryScaleDownThreshold"`

	QueueDepthThreshold int `json:"queueDepthThreshold"`

	ResponseTimeThreshold time.Duration `json:"responseTimeThreshold"`

	// Connection pool settings.

	MaxConnectionsPerPool int `json:"maxConnectionsPerPool"`

	ConnectionIdleTimeout time.Duration `json:"connectionIdleTimeout"`

	ConnectionMaxLifetime time.Duration `json:"connectionMaxLifetime"`

	// Cache settings.

	L1CacheSize int64 `json:"l1CacheSize"`

	L2CacheSize int64 `json:"l2CacheSize"`

	DefaultCacheTTL time.Duration `json:"defaultCacheTTL"`

	// Performance profiles.

	Profiles map[string]PerformanceProfile `json:"profiles"`
}

// PerformanceProfile defines performance characteristics for different workloads.

type PerformanceProfile struct {
	Name string `json:"name"`

	Description string `json:"description"`

	MaxConcurrentOperations int `json:"maxConcurrentOperations"`

	TimeoutSettings TimeoutProfile `json:"timeoutSettings"`

	CachingStrategy CachingProfile `json:"cachingStrategy"`

	ResourceLimits ResourceProfile `json:"resourceLimits"`

	ScalingPolicy ScalingProfile `json:"scalingPolicy"`
}

// TimeoutProfile defines timeout settings.

type TimeoutProfile struct {
	LLMProcessing time.Duration `json:"llmProcessing"`

	ResourcePlanning time.Duration `json:"resourcePlanning"`

	ManifestGeneration time.Duration `json:"manifestGeneration"`

	GitOpsCommit time.Duration `json:"gitOpsCommit"`

	DeploymentVerification time.Duration `json:"deploymentVerification"`
}

// CachingProfile defines caching strategy.

type CachingProfile struct {
	EnableL1Cache bool `json:"enableL1Cache"`

	EnableL2Cache bool `json:"enableL2Cache"`

	EnableL3Cache bool `json:"enableL3Cache"`

	TTLMultiplier float64 `json:"ttlMultiplier"`

	PrefetchEnabled bool `json:"prefetchEnabled"`

	CompressionEnabled bool `json:"compressionEnabled"`
}

// ResourceProfile defines resource allocation.

type ResourceProfile struct {
	CPURequest string `json:"cpuRequest"`

	CPULimit string `json:"cpuLimit"`

	MemoryRequest string `json:"memoryRequest"`

	MemoryLimit string `json:"memoryLimit"`

	MaxWorkers int `json:"maxWorkers"`
}

// ScalingProfile defines scaling behavior.

type ScalingProfile struct {
	MinReplicas int `json:"minReplicas"`

	MaxReplicas int `json:"maxReplicas"`

	ScaleUpPolicy ScalingPolicy `json:"scaleUpPolicy"`

	ScaleDownPolicy ScalingPolicy `json:"scaleDownPolicy"`
}

// ScalingPolicy defines how scaling should occur.

type ScalingPolicy struct {
	MetricType string `json:"metricType"`

	TargetValue float64 `json:"targetValue"`

	StabilizationWindow time.Duration `json:"stabilizationWindow"`

	ScalingFactor float64 `json:"scalingFactor"`
}

// NewPerformanceManager creates a new performance manager.

func NewPerformanceManager(config *PerformanceConfig, logger logr.Logger) *PerformanceManager {

	pm := &PerformanceManager{

		logger: logger.WithName("performance-manager"),

		config: config,

		profiles: make(map[string]*PerformanceProfile),

		stopChan: make(chan bool),
	}

	// Initialize components.

	pm.resourceMonitor = NewResourceMonitor(logger)

	pm.metricsAnalyzer = NewMetricsAnalyzer(logger)

	pm.scalingDecision = NewScalingDecisionEngine(config, logger)

	pm.cacheManager = NewMultiLevelCacheManager(config, logger)

	pm.connectionManager = NewConnectionPoolManager(config, logger)

	pm.loadBalancer = NewLoadBalancer(logger)

	// Load performance profiles.

	pm.loadPerformanceProfiles(config.Profiles)

	return pm

}

// Start starts the performance manager.

func (pm *PerformanceManager) Start(ctx context.Context) error {

	pm.mutex.Lock()

	defer pm.mutex.Unlock()

	if pm.started {

		return fmt.Errorf("performance manager already started")

	}

	// Start all components.

	if err := pm.resourceMonitor.Start(ctx); err != nil {

		return fmt.Errorf("failed to start resource monitor: %w", err)

	}

	if err := pm.cacheManager.Start(ctx); err != nil {

		return fmt.Errorf("failed to start cache manager: %w", err)

	}

	if err := pm.connectionManager.Start(ctx); err != nil {

		return fmt.Errorf("failed to start connection manager: %w", err)

	}

	// Start performance monitoring and optimization loops.

	go pm.metricsCollectionLoop(ctx)

	go pm.performanceAnalysisLoop(ctx)

	go pm.scalingDecisionLoop(ctx)

	pm.started = true

	pm.logger.Info("Performance manager started")

	return nil

}

// Stop stops the performance manager.

func (pm *PerformanceManager) Stop(ctx context.Context) error {

	pm.mutex.Lock()

	defer pm.mutex.Unlock()

	if !pm.started {

		return nil

	}

	pm.logger.Info("Stopping performance manager")

	// Stop all components.

	if err := pm.connectionManager.Stop(ctx); err != nil {

		pm.logger.Error(err, "Error stopping connection manager")

	}

	if err := pm.cacheManager.Stop(ctx); err != nil {

		pm.logger.Error(err, "Error stopping cache manager")

	}

	if err := pm.resourceMonitor.Stop(ctx); err != nil {

		pm.logger.Error(err, "Error stopping resource monitor")

	}

	// Signal stop to all loops.

	close(pm.stopChan)

	pm.started = false

	pm.logger.Info("Performance manager stopped")

	return nil

}

// OptimizeForProfile applies a performance profile to the system.

func (pm *PerformanceManager) OptimizeForProfile(ctx context.Context, profileName string, phase interfaces.ProcessingPhase) error {

	profile, exists := pm.profiles[profileName]

	if !exists {

		return fmt.Errorf("performance profile %s not found", profileName)

	}

	pm.logger.Info("Applying performance profile", "profile", profileName, "phase", phase)

	// Apply caching strategy.

	if err := pm.cacheManager.ApplyCachingProfile(profile.CachingStrategy); err != nil {

		pm.logger.Error(err, "Failed to apply caching profile")

	}

	// Apply connection pool settings.

	if err := pm.connectionManager.ApplyResourceProfile(profile.ResourceLimits); err != nil {

		pm.logger.Error(err, "Failed to apply resource profile")

	}

	// Apply scaling policies.

	pm.scalingDecision.ApplyScalingProfile(profile.ScalingPolicy)

	return nil

}

// GetPerformanceRecommendations analyzes current performance and provides recommendations.

func (pm *PerformanceManager) GetPerformanceRecommendations(ctx context.Context) (*PerformanceRecommendations, error) {

	// Collect current metrics.

	currentMetrics := pm.resourceMonitor.GetCurrentMetrics()

	// Analyze performance.

	analysis := pm.metricsAnalyzer.AnalyzePerformance(currentMetrics)

	// Generate recommendations.

	recommendations := &PerformanceRecommendations{

		Timestamp: time.Now(),

		CurrentMetrics: currentMetrics,

		Analysis: analysis,

		Recommendations: make([]Recommendation, 0),
	}

	// CPU optimization recommendations.

	if analysis.CPUUtilization > pm.config.CPUScaleUpThreshold {

		recommendations.Recommendations = append(recommendations.Recommendations, Recommendation{

			Type: "scaling",

			Priority: "high",

			Description: "CPU utilization is high, consider scaling up workers",

			Action: "scale_up",

			Parameters: map[string]interface{}{

				"metric": "cpu",

				"current": analysis.CPUUtilization,

				"threshold": pm.config.CPUScaleUpThreshold,

				"suggestion": "increase worker count by 25%",
			},
		})

	}

	// Memory optimization recommendations.

	if analysis.MemoryUtilization > pm.config.MemoryScaleUpThreshold {

		recommendations.Recommendations = append(recommendations.Recommendations, Recommendation{

			Type: "resource",

			Priority: "medium",

			Description: "Memory utilization is high, consider optimizing caching",

			Action: "optimize_cache",

			Parameters: map[string]interface{}{

				"metric": "memory",

				"current": analysis.MemoryUtilization,

				"suggestion": "reduce cache size or implement compression",
			},
		})

	}

	// Queue depth recommendations.

	if analysis.MaxQueueDepth > pm.config.QueueDepthThreshold {

		recommendations.Recommendations = append(recommendations.Recommendations, Recommendation{

			Type: "scaling",

			Priority: "high",

			Description: "Queue depth is high, consider increasing processing capacity",

			Action: "scale_up",

			Parameters: map[string]interface{}{

				"metric": "queue_depth",

				"current": analysis.MaxQueueDepth,

				"threshold": pm.config.QueueDepthThreshold,

				"suggestion": "increase worker pools for bottleneck phases",
			},
		})

	}

	// Response time recommendations.

	if analysis.AverageResponseTime > pm.config.ResponseTimeThreshold {

		recommendations.Recommendations = append(recommendations.Recommendations, Recommendation{

			Type: "performance",

			Priority: "medium",

			Description: "Response time is degraded, consider optimization",

			Action: "optimize_performance",

			Parameters: map[string]interface{}{

				"metric": "response_time",

				"current": analysis.AverageResponseTime.String(),

				"threshold": pm.config.ResponseTimeThreshold.String(),

				"suggestion": "enable advanced caching and connection pooling",
			},
		})

	}

	// Cache optimization recommendations.

	if analysis.CacheHitRate < 0.7 { // Less than 70% hit rate

		recommendations.Recommendations = append(recommendations.Recommendations, Recommendation{

			Type: "caching",

			Priority: "low",

			Description: "Cache hit rate is low, consider tuning cache strategy",

			Action: "optimize_cache",

			Parameters: map[string]interface{}{

				"metric": "cache_hit_rate",

				"current": analysis.CacheHitRate,

				"suggestion": "increase cache size or adjust TTL settings",
			},
		})

	}

	return recommendations, nil

}

// AutoOptimize automatically applies optimizations based on current performance.

func (pm *PerformanceManager) AutoOptimize(ctx context.Context) error {

	recommendations, err := pm.GetPerformanceRecommendations(ctx)

	if err != nil {

		return fmt.Errorf("failed to get recommendations: %w", err)

	}

	pm.logger.Info("Applying auto-optimizations", "recommendationCount", len(recommendations.Recommendations))

	for _, recommendation := range recommendations.Recommendations {

		if recommendation.Priority == "high" || recommendation.Priority == "medium" {

			if err := pm.applyRecommendation(ctx, recommendation); err != nil {

				pm.logger.Error(err, "Failed to apply recommendation", "type", recommendation.Type, "action", recommendation.Action)

			}

		}

	}

	return nil

}

// metricsCollectionLoop continuously collects performance metrics.

func (pm *PerformanceManager) metricsCollectionLoop(ctx context.Context) {

	ticker := time.NewTicker(pm.config.MetricsCollectionInterval)

	defer ticker.Stop()

	pm.logger.Info("Started metrics collection loop")

	for {

		select {

		case <-ticker.C:

			pm.collectMetrics(ctx)

		case <-pm.stopChan:

			pm.logger.Info("Metrics collection loop stopped")

			return

		case <-ctx.Done():

			pm.logger.Info("Metrics collection loop cancelled")

			return

		}

	}

}

// performanceAnalysisLoop continuously analyzes performance.

func (pm *PerformanceManager) performanceAnalysisLoop(ctx context.Context) {

	ticker := time.NewTicker(pm.config.AnalysisInterval)

	defer ticker.Stop()

	pm.logger.Info("Started performance analysis loop")

	for {

		select {

		case <-ticker.C:

			pm.analyzePerformance(ctx)

		case <-pm.stopChan:

			pm.logger.Info("Performance analysis loop stopped")

			return

		case <-ctx.Done():

			pm.logger.Info("Performance analysis loop cancelled")

			return

		}

	}

}

// scalingDecisionLoop makes scaling decisions based on performance metrics.

func (pm *PerformanceManager) scalingDecisionLoop(ctx context.Context) {

	ticker := time.NewTicker(1 * time.Minute) // Check every minute

	defer ticker.Stop()

	pm.logger.Info("Started scaling decision loop")

	for {

		select {

		case <-ticker.C:

			pm.makeScalingDecisions(ctx)

		case <-pm.stopChan:

			pm.logger.Info("Scaling decision loop stopped")

			return

		case <-ctx.Done():

			pm.logger.Info("Scaling decision loop cancelled")

			return

		}

	}

}

// Helper methods.

func (pm *PerformanceManager) loadPerformanceProfiles(profiles map[string]PerformanceProfile) {

	for name, profile := range profiles {

		profileCopy := profile

		pm.profiles[name] = &profileCopy

	}

	// Add default profiles if none exist.

	if len(pm.profiles) == 0 {

		pm.addDefaultProfiles()

	}

	pm.logger.Info("Loaded performance profiles", "count", len(pm.profiles))

}

func (pm *PerformanceManager) addDefaultProfiles() {

	// High-performance profile.

	pm.profiles["high-performance"] = &PerformanceProfile{

		Name: "high-performance",

		Description: "Maximum performance for production workloads",

		MaxConcurrentOperations: 100,

		TimeoutSettings: TimeoutProfile{

			LLMProcessing: 30 * time.Second,

			ResourcePlanning: 15 * time.Second,

			ManifestGeneration: 10 * time.Second,

			GitOpsCommit: 30 * time.Second,

			DeploymentVerification: 120 * time.Second,
		},

		CachingStrategy: CachingProfile{

			EnableL1Cache: true,

			EnableL2Cache: true,

			EnableL3Cache: true,

			TTLMultiplier: 1.5,

			PrefetchEnabled: true,

			CompressionEnabled: false, // Trade CPU for speed

		},

		ResourceLimits: ResourceProfile{

			CPURequest: "2",

			CPULimit: "4",

			MemoryRequest: "4Gi",

			MemoryLimit: "8Gi",

			MaxWorkers: 20,
		},

		ScalingPolicy: ScalingProfile{

			MinReplicas: 3,

			MaxReplicas: 50,

			ScaleUpPolicy: ScalingPolicy{

				MetricType: "cpu",

				TargetValue: 70.0,

				StabilizationWindow: 1 * time.Minute,

				ScalingFactor: 1.5,
			},
		},
	}

	// Balanced profile.

	pm.profiles["balanced"] = &PerformanceProfile{

		Name: "balanced",

		Description: "Balanced performance and resource usage",

		MaxConcurrentOperations: 50,

		TimeoutSettings: TimeoutProfile{

			LLMProcessing: 60 * time.Second,

			ResourcePlanning: 30 * time.Second,

			ManifestGeneration: 20 * time.Second,

			GitOpsCommit: 60 * time.Second,

			DeploymentVerification: 180 * time.Second,
		},

		CachingStrategy: CachingProfile{

			EnableL1Cache: true,

			EnableL2Cache: true,

			EnableL3Cache: false,

			TTLMultiplier: 1.0,

			PrefetchEnabled: false,

			CompressionEnabled: true,
		},

		ResourceLimits: ResourceProfile{

			CPURequest: "1",

			CPULimit: "2",

			MemoryRequest: "2Gi",

			MemoryLimit: "4Gi",

			MaxWorkers: 10,
		},

		ScalingPolicy: ScalingProfile{

			MinReplicas: 2,

			MaxReplicas: 20,

			ScaleUpPolicy: ScalingPolicy{

				MetricType: "cpu",

				TargetValue: 80.0,

				StabilizationWindow: 3 * time.Minute,

				ScalingFactor: 1.25,
			},
		},
	}

	// Resource-efficient profile.

	pm.profiles["efficient"] = &PerformanceProfile{

		Name: "efficient",

		Description: "Optimized for resource efficiency",

		MaxConcurrentOperations: 20,

		TimeoutSettings: TimeoutProfile{

			LLMProcessing: 120 * time.Second,

			ResourcePlanning: 60 * time.Second,

			ManifestGeneration: 30 * time.Second,

			GitOpsCommit: 90 * time.Second,

			DeploymentVerification: 300 * time.Second,
		},

		CachingStrategy: CachingProfile{

			EnableL1Cache: true,

			EnableL2Cache: false,

			EnableL3Cache: false,

			TTLMultiplier: 0.5,

			PrefetchEnabled: false,

			CompressionEnabled: true,
		},

		ResourceLimits: ResourceProfile{

			CPURequest: "500m",

			CPULimit: "1",

			MemoryRequest: "1Gi",

			MemoryLimit: "2Gi",

			MaxWorkers: 5,
		},

		ScalingPolicy: ScalingProfile{

			MinReplicas: 1,

			MaxReplicas: 10,

			ScaleUpPolicy: ScalingPolicy{

				MetricType: "memory",

				TargetValue: 90.0,

				StabilizationWindow: 5 * time.Minute,

				ScalingFactor: 1.1,
			},
		},
	}

}

func (pm *PerformanceManager) collectMetrics(ctx context.Context) {

	pm.logger.V(1).Info("Collecting performance metrics")

	pm.resourceMonitor.CollectMetrics(ctx)

}

func (pm *PerformanceManager) analyzePerformance(ctx context.Context) {

	pm.logger.V(1).Info("Analyzing performance")

	metrics := pm.resourceMonitor.GetCurrentMetrics()

	analysis := pm.metricsAnalyzer.AnalyzePerformance(metrics)

	// Log significant performance issues.

	if analysis.CPUUtilization > pm.config.CPUScaleUpThreshold {

		pm.logger.Info("High CPU utilization detected", "utilization", analysis.CPUUtilization)

	}

	if analysis.MemoryUtilization > pm.config.MemoryScaleUpThreshold {

		pm.logger.Info("High memory utilization detected", "utilization", analysis.MemoryUtilization)

	}

	if analysis.MaxQueueDepth > pm.config.QueueDepthThreshold {

		pm.logger.Info("High queue depth detected", "depth", analysis.MaxQueueDepth)

	}

}

func (pm *PerformanceManager) makeScalingDecisions(ctx context.Context) {

	pm.logger.V(1).Info("Making scaling decisions")

	metrics := pm.resourceMonitor.GetCurrentMetrics()

	decisions := pm.scalingDecision.EvaluateScaling(ctx, metrics)

	for _, decision := range decisions {

		pm.logger.Info("Scaling decision made",

			"action", decision.Action,

			"target", decision.TargetReplicas,

			"reason", decision.Reason,

			"confidence", decision.Confidence)

		// Apply scaling decisions (this would integrate with Kubernetes HPA or custom scaling logic).

		if err := pm.applyScalingDecision(ctx, &decision); err != nil {

			pm.logger.Error(err, "Failed to apply scaling decision", "action", decision.Action)

		}

	}

}

func (pm *PerformanceManager) applyRecommendation(ctx context.Context, recommendation Recommendation) error {

	pm.logger.Info("Applying performance recommendation",

		"type", recommendation.Type,

		"action", recommendation.Action,

		"priority", recommendation.Priority)

	switch recommendation.Action {

	case "scale_up":

		return pm.handleScaleUpRecommendation(ctx, recommendation)

	case "optimize_cache":

		return pm.handleCacheOptimizationRecommendation(ctx, recommendation)

	case "optimize_performance":

		return pm.handlePerformanceOptimizationRecommendation(ctx, recommendation)

	default:

		return fmt.Errorf("unknown recommendation action: %s", recommendation.Action)

	}

}

func (pm *PerformanceManager) handleScaleUpRecommendation(ctx context.Context, recommendation Recommendation) error {

	// This would implement actual scaling logic.

	pm.logger.Info("Handling scale up recommendation", "parameters", recommendation.Parameters)

	return nil

}

func (pm *PerformanceManager) handleCacheOptimizationRecommendation(ctx context.Context, recommendation Recommendation) error {

	// This would implement cache optimization logic.

	pm.logger.Info("Handling cache optimization recommendation", "parameters", recommendation.Parameters)

	return pm.cacheManager.OptimizeCache(ctx, recommendation.Parameters)

}

func (pm *PerformanceManager) handlePerformanceOptimizationRecommendation(ctx context.Context, recommendation Recommendation) error {

	// This would implement general performance optimization.

	pm.logger.Info("Handling performance optimization recommendation", "parameters", recommendation.Parameters)

	return nil

}

func (pm *PerformanceManager) applyScalingDecision(ctx context.Context, decision *ScalingDecision) error {

	// This would implement actual scaling through Kubernetes APIs.

	pm.logger.Info("Applying scaling decision", "decision", decision)

	return nil

}

// Data structures.

// PerformanceRecommendations contains performance analysis and recommendations.

type PerformanceRecommendations struct {
	Timestamp time.Time `json:"timestamp"`

	CurrentMetrics *SystemMetrics `json:"currentMetrics"`

	Analysis *PerformanceAnalysis `json:"analysis"`

	Recommendations []Recommendation `json:"recommendations"`
}

// Recommendation represents a performance optimization recommendation.

type Recommendation struct {
	Type string `json:"type"` // scaling, caching, resource, performance

	Priority string `json:"priority"` // low, medium, high, critical

	Description string `json:"description"`

	Action string `json:"action"` // scale_up, scale_down, optimize_cache, etc.

	Parameters map[string]interface{} `json:"parameters"`

	Confidence float64 `json:"confidence"` // 0.0 to 1.0

}

// SystemMetrics represents current system performance metrics.

type SystemMetrics struct {
	Timestamp time.Time `json:"timestamp"`

	CPUUtilization float64 `json:"cpuUtilization"`

	MemoryUtilization float64 `json:"memoryUtilization"`

	NetworkIO NetworkIOMetrics `json:"networkIO"`

	DiskIO DiskIOMetrics `json:"diskIO"`

	QueueDepths map[interfaces.ProcessingPhase]int `json:"queueDepths"`

	ResponseTimes map[interfaces.ProcessingPhase]time.Duration `json:"responseTimes"`

	ThroughputRates map[interfaces.ProcessingPhase]float64 `json:"throughputRates"`

	ErrorRates map[interfaces.ProcessingPhase]float64 `json:"errorRates"`

	CacheMetrics *CacheMetrics `json:"cacheMetrics"`

	ConnectionMetrics *ConnectionMetrics `json:"connectionMetrics"`
}

// NetworkIOMetrics represents network I/O metrics.

type NetworkIOMetrics struct {
	BytesIn int64 `json:"bytesIn"`

	BytesOut int64 `json:"bytesOut"`

	PacketsIn int64 `json:"packetsIn"`

	PacketsOut int64 `json:"packetsOut"`
}

// DiskIOMetrics represents disk I/O metrics.

type DiskIOMetrics struct {
	BytesRead int64 `json:"bytesRead"`

	BytesWritten int64 `json:"bytesWritten"`

	IOPSRead int64 `json:"iopsRead"`

	IOPSWrite int64 `json:"iopsWrite"`
}

// CacheMetrics represents cache performance metrics.

type CacheMetrics struct {
	HitRate float64 `json:"hitRate"`

	MissRate float64 `json:"missRate"`

	EvictionRate float64 `json:"evictionRate"`

	SizeBytes int64 `json:"sizeBytes"`

	ItemCount int64 `json:"itemCount"`
}

// ConnectionMetrics represents connection pool metrics.

type ConnectionMetrics struct {
	TotalConnections int `json:"totalConnections"`

	ActiveConnections int `json:"activeConnections"`

	IdleConnections int `json:"idleConnections"`

	FailedConnections int `json:"failedConnections"`
}

// PerformanceAnalysis represents the results of performance analysis.

type PerformanceAnalysis struct {
	OverallHealth string `json:"overallHealth"` // healthy, degraded, unhealthy

	CPUUtilization float64 `json:"cpuUtilization"`

	MemoryUtilization float64 `json:"memoryUtilization"`

	MaxQueueDepth int `json:"maxQueueDepth"`

	AverageResponseTime time.Duration `json:"averageResponseTime"`

	TotalThroughput float64 `json:"totalThroughput"`

	OverallErrorRate float64 `json:"overallErrorRate"`

	CacheHitRate float64 `json:"cacheHitRate"`

	Bottlenecks []PerformanceBottleneck `json:"bottlenecks"`

	Trends map[string]PerformanceTrend `json:"trends"`
}

// PerformanceBottleneck identifies a performance bottleneck.

type PerformanceBottleneck struct {
	Phase interfaces.ProcessingPhase `json:"phase"`

	Type string `json:"type"` // cpu, memory, io, queue, external

	Severity string `json:"severity"` // low, medium, high, critical

	Description string `json:"description"`

	Impact float64 `json:"impact"` // 0.0 to 1.0

}

// PerformanceTrend represents a performance trend over time.

type PerformanceTrend struct {
	Metric string `json:"metric"`

	Direction string `json:"direction"` // improving, degrading, stable

	Rate float64 `json:"rate"` // rate of change

	Confidence float64 `json:"confidence"` // 0.0 to 1.0

}

// ScalingDecision represents a scaling decision.

type ScalingDecision struct {
	Action string `json:"action"` // scale_up, scale_down, no_action

	Phase interfaces.ProcessingPhase `json:"phase"` // which phase to scale

	CurrentReplicas int `json:"currentReplicas"`

	TargetReplicas int `json:"targetReplicas"`

	Reason string `json:"reason"`

	Confidence float64 `json:"confidence"`

	Timestamp time.Time `json:"timestamp"`
}
