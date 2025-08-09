package automation

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	// Health monitoring metrics
	healthCheckDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nephoran_health_check_duration_seconds",
		Help:    "Duration of health check operations",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
	}, []string{"component", "check_type"})

	healthCheckResults = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nephoran_health_check_results_total",
		Help: "Total number of health check results",
	}, []string{"component", "status"})

	componentHealthScore = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nephoran_component_health_score",
		Help: "Health score for each component (0-100)",
	}, []string{"component"})
)

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(config *SelfHealingConfig, k8sClient kubernetes.Interface, logger *slog.Logger) (*HealthMonitor, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	monitor := &HealthMonitor{
		config:         config,
		logger:         logger,
		k8sClient:      k8sClient,
		healthCheckers: make(map[string]*ComponentHealthChecker),
		systemMetrics: &SystemHealthMetrics{
			ComponentHealth:     make(map[string]HealthStatus),
			PredictedFailures:   make(map[string]float64),
			ResourceUtilization: make(map[string]float64),
			PerformanceMetrics:  make(map[string]float64),
		},
	}

	// Initialize component health checkers
	for componentName, componentConfig := range config.ComponentConfigs {
		checker := &ComponentHealthChecker{
			component:           componentConfig,
			currentStatus:       HealthStatusHealthy,
			consecutiveFailures: 0,
			metrics: &ComponentMetrics{
				RestartCount: 0,
			},
			restartHistory: make([]*RestartEvent, 0),
		}
		monitor.healthCheckers[componentName] = checker
	}

	return monitor, nil
}

// Start starts the health monitor
func (hm *HealthMonitor) Start(ctx context.Context) {
	hm.logger.Info("Starting health monitor")

	// Start individual component health checkers
	for name, checker := range hm.healthCheckers {
		go hm.startComponentHealthChecker(ctx, name, checker)
	}

	// Start system health aggregation
	go hm.runSystemHealthAggregation(ctx)

	hm.logger.Info("Health monitor started successfully")
}

// startComponentHealthChecker starts health checking for a specific component
func (hm *HealthMonitor) startComponentHealthChecker(ctx context.Context, componentName string, checker *ComponentHealthChecker) {
	ticker := time.NewTicker(hm.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.checkComponentHealth(ctx, componentName, checker)
		}
	}
}

// checkComponentHealth performs health check for a single component
func (hm *HealthMonitor) checkComponentHealth(ctx context.Context, componentName string, checker *ComponentHealthChecker) {
	start := time.Now()
	defer func() {
		healthCheckDuration.WithLabelValues(componentName, "comprehensive").Observe(time.Since(start).Seconds())
	}()

	hm.logger.Debug("Checking component health", "component", componentName)

	// Perform multiple health checks
	healthStatus := hm.performComprehensiveHealthCheck(ctx, componentName, checker)

	// Update checker state
	checker.lastCheck = time.Now()
	previousStatus := checker.currentStatus
	checker.currentStatus = healthStatus

	// Track consecutive failures
	if healthStatus == HealthStatusHealthy {
		checker.consecutiveFailures = 0
	} else {
		checker.consecutiveFailures++
	}

	// Update metrics
	hm.updateComponentMetrics(componentName, checker)

	// Log status changes
	if previousStatus != healthStatus {
		hm.logger.Info("Component health status changed",
			"component", componentName,
			"previous", previousStatus,
			"current", healthStatus,
			"consecutive_failures", checker.consecutiveFailures)

		// Send alert if needed
		if hm.alertManager != nil {
			hm.sendHealthAlert(componentName, healthStatus, checker.consecutiveFailures)
		}
	}

	// Record health check result
	healthCheckResults.WithLabelValues(componentName, string(healthStatus)).Inc()
}

// performComprehensiveHealthCheck performs comprehensive health assessment
func (hm *HealthMonitor) performComprehensiveHealthCheck(ctx context.Context, componentName string, checker *ComponentHealthChecker) HealthStatus {
	component := checker.component
	overallScore := 100.0
	checkResults := make(map[string]float64)

	// 1. Kubernetes deployment health check
	if deploymentScore := hm.checkKubernetesDeployment(ctx, componentName); deploymentScore >= 0 {
		checkResults["kubernetes"] = deploymentScore
		overallScore *= (deploymentScore / 100.0)
	}

	// 2. HTTP endpoint health check
	if component.HealthCheckEndpoint != "" {
		if endpointScore := hm.checkHTTPEndpoint(ctx, component.HealthCheckEndpoint); endpointScore >= 0 {
			checkResults["endpoint"] = endpointScore
			overallScore *= (endpointScore / 100.0)
		}
	}

	// 3. Resource utilization check
	if resourceScore := hm.checkResourceUtilization(ctx, componentName, component); resourceScore >= 0 {
		checkResults["resources"] = resourceScore
		overallScore *= (resourceScore / 100.0)
	}

	// 4. Performance metrics check
	if performanceScore := hm.checkPerformanceMetrics(ctx, componentName, component); performanceScore >= 0 {
		checkResults["performance"] = performanceScore
		overallScore *= (performanceScore / 100.0)
	}

	// 5. Dependency health check
	if dependencyScore := hm.checkDependencies(ctx, component.DependsOn); dependencyScore >= 0 {
		checkResults["dependencies"] = dependencyScore
		overallScore *= (dependencyScore / 100.0)
	}

	// Update component health score metric
	componentHealthScore.WithLabelValues(componentName).Set(overallScore)

	// Determine overall health status
	return hm.calculateHealthStatus(overallScore, checker.consecutiveFailures)
}

// checkKubernetesDeployment checks Kubernetes deployment health
func (hm *HealthMonitor) checkKubernetesDeployment(ctx context.Context, componentName string) float64 {
	// Get deployment
	deployment, err := hm.k8sClient.AppsV1().Deployments(hm.config.ComponentConfigs[componentName].Name).
		Get(ctx, componentName, metav1.GetOptions{})
	if err != nil {
		hm.logger.Debug("Failed to get deployment", "component", componentName, "error", err)
		return 0.0
	}

	// Check if deployment is available
	if deployment.Status.AvailableReplicas == 0 {
		return 0.0
	}

	// Calculate health score based on replica ratio
	desiredReplicas := float64(*deployment.Spec.Replicas)
	availableReplicas := float64(deployment.Status.AvailableReplicas)
	readyReplicas := float64(deployment.Status.ReadyReplicas)

	if desiredReplicas == 0 {
		return 100.0
	}

	// Weight available and ready replicas
	availabilityScore := (availableReplicas / desiredReplicas) * 60.0
	readinessScore := (readyReplicas / desiredReplicas) * 40.0

	return availabilityScore + readinessScore
}

// checkHTTPEndpoint checks HTTP endpoint health
func (hm *HealthMonitor) checkHTTPEndpoint(ctx context.Context, endpoint string) float64 {
	client := &http.Client{
		Timeout: hm.config.HealthCheckTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0.0
	}

	start := time.Now()
	resp, err := client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return 0.0
	}
	defer resp.Body.Close()

	// Base score based on response code
	var baseScore float64
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		baseScore = 100.0
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		baseScore = 80.0
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		baseScore = 40.0
	default:
		baseScore = 0.0
	}

	// Adjust score based on response time
	if responseTime > 5*time.Second {
		baseScore *= 0.5
	} else if responseTime > 2*time.Second {
		baseScore *= 0.8
	} else if responseTime > 1*time.Second {
		baseScore *= 0.9
	}

	return baseScore
}

// checkResourceUtilization checks resource utilization health
func (hm *HealthMonitor) checkResourceUtilization(ctx context.Context, componentName string, component *ComponentConfig) float64 {
	if component.ResourceLimits == nil {
		return 100.0 // No limits defined, assume healthy
	}

	// Get pod metrics (simplified - in real implementation would use metrics API)
	// For now, simulate resource check
	cpuUsage := hm.getCurrentCPUUsage(ctx, componentName)
	memoryUsage := hm.getCurrentMemoryUsage(ctx, componentName)

	// Calculate resource health score
	cpuScore := 100.0
	if cpuUsage > 90.0 {
		cpuScore = 0.0
	} else if cpuUsage > 80.0 {
		cpuScore = 50.0
	} else if cpuUsage > 70.0 {
		cpuScore = 80.0
	}

	memoryScore := 100.0
	if memoryUsage > 90.0 {
		memoryScore = 0.0
	} else if memoryUsage > 80.0 {
		memoryScore = 50.0
	} else if memoryUsage > 70.0 {
		memoryScore = 80.0
	}

	return (cpuScore + memoryScore) / 2.0
}

// checkPerformanceMetrics checks performance-related health
func (hm *HealthMonitor) checkPerformanceMetrics(ctx context.Context, componentName string, component *ComponentConfig) float64 {
	if component.PerformanceThresholds == nil {
		return 100.0
	}

	score := 100.0

	// Check latency (simulated)
	currentLatency := hm.getCurrentLatency(ctx, componentName)
	if component.PerformanceThresholds.MaxLatency > 0 && currentLatency > component.PerformanceThresholds.MaxLatency {
		score *= 0.7
	}

	// Check error rate (simulated)
	currentErrorRate := hm.getCurrentErrorRate(ctx, componentName)
	if component.PerformanceThresholds.MaxErrorRate > 0 && currentErrorRate > component.PerformanceThresholds.MaxErrorRate {
		score *= 0.5
	}

	// Check throughput (simulated)
	currentThroughput := hm.getCurrentThroughput(ctx, componentName)
	if component.PerformanceThresholds.MinThroughput > 0 && currentThroughput < component.PerformanceThresholds.MinThroughput {
		score *= 0.8
	}

	return score
}

// checkDependencies checks health of component dependencies
func (hm *HealthMonitor) checkDependencies(ctx context.Context, dependencies []string) float64 {
	if len(dependencies) == 0 {
		return 100.0
	}

	totalScore := 0.0
	for _, dep := range dependencies {
		if checker, exists := hm.healthCheckers[dep]; exists {
			switch checker.currentStatus {
			case HealthStatusHealthy:
				totalScore += 100.0
			case HealthStatusDegraded:
				totalScore += 70.0
			case HealthStatusUnhealthy:
				totalScore += 30.0
			case HealthStatusCritical:
				totalScore += 0.0
			}
		} else {
			// Dependency not monitored, assume healthy
			totalScore += 100.0
		}
	}

	return totalScore / float64(len(dependencies))
}

// calculateHealthStatus determines health status from score and failures
func (hm *HealthMonitor) calculateHealthStatus(score float64, consecutiveFailures int) HealthStatus {
	// Consider consecutive failures
	if consecutiveFailures >= 5 {
		return HealthStatusCritical
	} else if consecutiveFailures >= 3 {
		return HealthStatusUnhealthy
	}

	// Consider overall score
	switch {
	case score >= 90.0:
		return HealthStatusHealthy
	case score >= 70.0:
		return HealthStatusDegraded
	case score >= 30.0:
		return HealthStatusUnhealthy
	default:
		return HealthStatusCritical
	}
}

// runSystemHealthAggregation aggregates component health into system health
func (hm *HealthMonitor) runSystemHealthAggregation(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Aggregate every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.aggregateSystemHealth()
		}
	}
}

// aggregateSystemHealth aggregates individual component health into system health
func (hm *HealthMonitor) aggregateSystemHealth() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	totalComponents := len(hm.healthCheckers)
	if totalComponents == 0 {
		return
	}

	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0
	criticalCount := 0

	// Count components by health status
	for componentName, checker := range hm.healthCheckers {
		hm.systemMetrics.ComponentHealth[componentName] = checker.currentStatus

		switch checker.currentStatus {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusDegraded:
			degradedCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		case HealthStatusCritical:
			criticalCount++
		}
	}

	// Determine overall system health
	if criticalCount > 0 || unhealthyCount > totalComponents/2 {
		hm.systemMetrics.OverallHealth = HealthStatusCritical
	} else if unhealthyCount > 0 || degradedCount > totalComponents/2 {
		hm.systemMetrics.OverallHealth = HealthStatusUnhealthy
	} else if degradedCount > 0 {
		hm.systemMetrics.OverallHealth = HealthStatusDegraded
	} else {
		hm.systemMetrics.OverallHealth = HealthStatusHealthy
	}

	hm.logger.Debug("System health aggregated",
		"overall", hm.systemMetrics.OverallHealth,
		"healthy", healthyCount,
		"degraded", degradedCount,
		"unhealthy", unhealthyCount,
		"critical", criticalCount)
}

// GetSystemHealth returns current system health
func (hm *HealthMonitor) GetSystemHealth() *SystemHealthMetrics {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &SystemHealthMetrics{
		OverallHealth:       hm.systemMetrics.OverallHealth,
		ComponentHealth:     make(map[string]HealthStatus),
		ActiveIncidents:     hm.systemMetrics.ActiveIncidents,
		ResolvedIncidents:   hm.systemMetrics.ResolvedIncidents,
		PredictedFailures:   make(map[string]float64),
		SystemLoad:          hm.systemMetrics.SystemLoad,
		ResourceUtilization: make(map[string]float64),
		PerformanceMetrics:  make(map[string]float64),
	}

	// Copy maps
	for k, v := range hm.systemMetrics.ComponentHealth {
		metrics.ComponentHealth[k] = v
	}
	for k, v := range hm.systemMetrics.PredictedFailures {
		metrics.PredictedFailures[k] = v
	}
	for k, v := range hm.systemMetrics.ResourceUtilization {
		metrics.ResourceUtilization[k] = v
	}
	for k, v := range hm.systemMetrics.PerformanceMetrics {
		metrics.PerformanceMetrics[k] = v
	}

	return metrics
}

// updateComponentMetrics updates component-specific metrics
func (hm *HealthMonitor) updateComponentMetrics(componentName string, checker *ComponentHealthChecker) {
	// Update component metrics (simulated data for now)
	checker.metrics.ResponseTime = hm.getCurrentLatency(context.Background(), componentName)
	checker.metrics.ErrorRate = hm.getCurrentErrorRate(context.Background(), componentName)
	checker.metrics.Throughput = hm.getCurrentThroughput(context.Background(), componentName)
	checker.metrics.CPUUsage = hm.getCurrentCPUUsage(context.Background(), componentName)
	checker.metrics.MemoryUsage = hm.getCurrentMemoryUsage(context.Background(), componentName)
}

// sendHealthAlert sends health-related alerts
func (hm *HealthMonitor) sendHealthAlert(componentName string, status HealthStatus, consecutiveFailures int) {
	// Implementation would send alerts through the alert manager
	hm.logger.Info("Health alert triggered",
		"component", componentName,
		"status", status,
		"consecutive_failures", consecutiveFailures)
}

// Helper methods for getting current metrics (simplified implementations)
func (hm *HealthMonitor) getCurrentCPUUsage(ctx context.Context, componentName string) float64 {
	// In real implementation, would query metrics API
	return 45.0 + float64(len(componentName)%20) // Simulated
}

func (hm *HealthMonitor) getCurrentMemoryUsage(ctx context.Context, componentName string) float64 {
	// In real implementation, would query metrics API
	return 60.0 + float64(len(componentName)%30) // Simulated
}

func (hm *HealthMonitor) getCurrentLatency(ctx context.Context, componentName string) time.Duration {
	// In real implementation, would query metrics
	return time.Duration(100+len(componentName)%200) * time.Millisecond // Simulated
}

func (hm *HealthMonitor) getCurrentErrorRate(ctx context.Context, componentName string) float64 {
	// In real implementation, would query metrics
	return float64(len(componentName)%5) / 100.0 // Simulated
}

func (hm *HealthMonitor) getCurrentThroughput(ctx context.Context, componentName string) float64 {
	// In real implementation, would query metrics
	return 100.0 + float64(len(componentName)%50) // Simulated
}
