package chaos

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SLAViolation represents a detected SLA violation.
type SLAViolation struct {
	Timestamp   time.Time
	Metric      string
	Threshold   float64
	ActualValue float64
	Severity    string // Minor, Major, Critical
	Description string
	Impact      string
}

// ValidationResult represents the result of a validation check.
type ValidationResult struct {
	Success   bool
	Timestamp time.Time
	Checks    []ValidationCheck
	Issues    string
}

// ValidationCheck represents a single validation check.
type ValidationCheck struct {
	Name        string
	Description string
	Passed      bool
	Value       interface{}
	Threshold   interface{}
	Details     string
}

// PreCheckResult represents pre-experiment validation result.
type PreCheckResult struct {
	Ready       bool
	Reason      string
	Warnings    []string
	SystemState SystemState
}

// SystemState captures system state before experiment.
type SystemState struct {
	Availability      float64
	LatencyP50        time.Duration
	LatencyP95        time.Duration
	LatencyP99        time.Duration
	Throughput        float64
	ErrorRate         float64
	ActivePods        int
	HealthyPods       int
	CPUUtilization    float64
	MemoryUtilization float64
}

// ResilienceValidator validates SLA resilience during chaos experiments.
type ResilienceValidator struct {
	client        client.Client
	logger        *zap.Logger
	promClient    v1.API
	config        *ValidatorConfig
	baselineState *SystemState
	mu            sync.RWMutex
}

// ValidatorConfig defines validator configuration.
type ValidatorConfig struct {
	PrometheusURL        string
	MetricsInterval      time.Duration
	BaselineWindow       time.Duration
	SLAConfig            SLAConfig
	CircuitBreakerConfig CircuitBreakerValidationConfig
	AutoScalingConfig    AutoScalingValidationConfig
	DegradationConfig    DegradationValidationConfig
}

// SLAConfig defines SLA thresholds for validation.
type SLAConfig struct {
	TargetAvailability float64       // 99.95%
	MaxLatencyP95      time.Duration // 2 seconds
	MaxLatencyP99      time.Duration // 3 seconds
	MinThroughput      float64       // 45 intents/minute
	MaxErrorRate       float64       // 0.05%
}

// CircuitBreakerValidationConfig defines circuit breaker validation parameters.
type CircuitBreakerValidationConfig struct {
	Enabled               bool
	ExpectedTriggerTime   time.Duration
	ExpectedRecoveryTime  time.Duration
	MaxConsecutiveErrors  int
	ErrorThresholdPercent float64
}

// AutoScalingValidationConfig defines auto-scaling validation parameters.
type AutoScalingValidationConfig struct {
	Enabled               bool
	ExpectedScaleUpTime   time.Duration
	ExpectedScaleDownTime time.Duration
	MinReplicas           int
	MaxReplicas           int
	TargetCPUPercent      float64
}

// DegradationValidationConfig defines graceful degradation validation.
type DegradationValidationConfig struct {
	Enabled                bool
	AcceptableDegradedMode []string
	MaxDegradationDuration time.Duration
	RequiredFunctionality  []string
}

// NewResilienceValidator creates a new resilience validator.
func NewResilienceValidator(client client.Client, logger *zap.Logger) *ResilienceValidator {
	// Initialize Prometheus client.
	promConfig := api.Config{
		Address: "http://prometheus:9090", // Default, can be overridden
	}
	promClient, err := api.NewClient(promConfig)
	if err != nil {
		logger.Warn("Failed to create Prometheus client", zap.Error(err))
	}

	return &ResilienceValidator{
		client:     client,
		logger:     logger,
		promClient: v1.NewAPI(promClient),
		config: &ValidatorConfig{
			MetricsInterval: 10 * time.Second,
			BaselineWindow:  5 * time.Minute,
			SLAConfig: SLAConfig{
				TargetAvailability: 99.95,
				MaxLatencyP95:      2 * time.Second,
				MaxLatencyP99:      3 * time.Second,
				MinThroughput:      45,
				MaxErrorRate:       0.05,
			},
			CircuitBreakerConfig: CircuitBreakerValidationConfig{
				Enabled:               true,
				ExpectedTriggerTime:   30 * time.Second,
				ExpectedRecoveryTime:  60 * time.Second,
				MaxConsecutiveErrors:  5,
				ErrorThresholdPercent: 50,
			},
			AutoScalingConfig: AutoScalingValidationConfig{
				Enabled:               true,
				ExpectedScaleUpTime:   2 * time.Minute,
				ExpectedScaleDownTime: 5 * time.Minute,
				MinReplicas:           2,
				MaxReplicas:           10,
				TargetCPUPercent:      70,
			},
			DegradationConfig: DegradationValidationConfig{
				Enabled:                true,
				AcceptableDegradedMode: []string{"read-only", "cached-responses", "reduced-features"},
				MaxDegradationDuration: 10 * time.Minute,
				RequiredFunctionality:  []string{"intent-processing", "monitoring", "health-check"},
			},
		},
	}
}

// PreExperimentValidation performs pre-experiment validation.
func (v *ResilienceValidator) PreExperimentValidation(ctx context.Context, experiment *Experiment) *PreCheckResult {
	v.logger.Info("Performing pre-experiment validation",
		zap.String("experiment", experiment.ID))

	result := &PreCheckResult{
		Ready:    true,
		Warnings: []string{},
	}

	// Capture baseline system state.
	state, err := v.captureSystemState(ctx)
	if err != nil {
		result.Ready = false
		result.Reason = fmt.Sprintf("Failed to capture system state: %v", err)
		return result
	}
	result.SystemState = *state

	// Store baseline for comparison.
	v.mu.Lock()
	v.baselineState = state
	v.mu.Unlock()

	// Check if system is already degraded.
	if state.Availability < v.config.SLAConfig.TargetAvailability {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("System availability %.2f%% is below target %.2f%%",
				state.Availability, v.config.SLAConfig.TargetAvailability))
	}

	if state.ErrorRate > v.config.SLAConfig.MaxErrorRate {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%",
				state.ErrorRate, v.config.SLAConfig.MaxErrorRate))
	}

	// Check resource availability.
	if state.CPUUtilization > 80 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("High CPU utilization: %.2f%%", state.CPUUtilization))
	}

	if state.MemoryUtilization > 80 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("High memory utilization: %.2f%%", state.MemoryUtilization))
	}

	// Check pod health.
	if state.HealthyPods < state.ActivePods {
		unhealthyCount := state.ActivePods - state.HealthyPods
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("%d pods are not healthy", unhealthyCount))
	}

	// Determine if it's safe to proceed.
	if len(result.Warnings) > 3 {
		result.Ready = false
		result.Reason = "Too many warnings detected, system may not be stable for chaos testing"
	}

	return result
}

// CheckSLACompliance checks for SLA violations during experiment.
func (v *ResilienceValidator) CheckSLACompliance(ctx context.Context, thresholds SLAThresholds) []SLAViolation {
	violations := []SLAViolation{}

	// Get current metrics.
	state, err := v.captureSystemState(ctx)
	if err != nil {
		v.logger.Error("Failed to capture system state for SLA check", zap.Error(err))
		return violations
	}

	// Check availability.
	if thresholds.MinAvailability > 0 && state.Availability < thresholds.MinAvailability {
		violations = append(violations, SLAViolation{
			Timestamp:   time.Now(),
			Metric:      "availability",
			Threshold:   thresholds.MinAvailability,
			ActualValue: state.Availability,
			Severity:    v.calculateSeverity("availability", state.Availability, thresholds.MinAvailability),
			Description: fmt.Sprintf("Availability %.2f%% below threshold %.2f%%",
				state.Availability, thresholds.MinAvailability),
			Impact: "Service availability degraded, potential user impact",
		})
	}

	// Check latency.
	maxLatency := time.Duration(thresholds.MaxLatencyMS) * time.Millisecond
	if thresholds.MaxLatencyMS > 0 && state.LatencyP95 > maxLatency {
		violations = append(violations, SLAViolation{
			Timestamp:   time.Now(),
			Metric:      "latency_p95",
			Threshold:   float64(maxLatency.Milliseconds()),
			ActualValue: float64(state.LatencyP95.Milliseconds()),
			Severity:    v.calculateSeverity("latency", float64(state.LatencyP95.Milliseconds()), float64(maxLatency.Milliseconds())),
			Description: fmt.Sprintf("P95 latency %v exceeds threshold %v",
				state.LatencyP95, maxLatency),
			Impact: "User experience degraded due to high latency",
		})
	}

	// Check throughput.
	if thresholds.MinThroughput > 0 && state.Throughput < thresholds.MinThroughput {
		violations = append(violations, SLAViolation{
			Timestamp:   time.Now(),
			Metric:      "throughput",
			Threshold:   thresholds.MinThroughput,
			ActualValue: state.Throughput,
			Severity:    v.calculateSeverity("throughput", state.Throughput, thresholds.MinThroughput),
			Description: fmt.Sprintf("Throughput %.2f below threshold %.2f intents/min",
				state.Throughput, thresholds.MinThroughput),
			Impact: "Reduced processing capacity",
		})
	}

	// Check error rate.
	if thresholds.MaxErrorRate > 0 && state.ErrorRate > thresholds.MaxErrorRate {
		violations = append(violations, SLAViolation{
			Timestamp:   time.Now(),
			Metric:      "error_rate",
			Threshold:   thresholds.MaxErrorRate,
			ActualValue: state.ErrorRate,
			Severity:    v.calculateSeverity("error_rate", state.ErrorRate, thresholds.MaxErrorRate),
			Description: fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%",
				state.ErrorRate, thresholds.MaxErrorRate),
			Impact: "Increased failures affecting users",
		})
	}

	return violations
}

// ValidateCircuitBreaker validates circuit breaker behavior during chaos.
func (v *ResilienceValidator) ValidateCircuitBreaker(ctx context.Context, startTime time.Time) *ValidationResult {
	result := &ValidationResult{
		Success:   true,
		Timestamp: time.Now(),
		Checks:    []ValidationCheck{},
	}

	if !v.config.CircuitBreakerConfig.Enabled {
		result.Checks = append(result.Checks, ValidationCheck{
			Name:        "Circuit Breaker",
			Description: "Circuit breaker validation disabled",
			Passed:      true,
		})
		return result
	}

	// Query circuit breaker metrics.
	cbState, err := v.queryCircuitBreakerState(ctx)
	if err != nil {
		result.Success = false
		result.Issues = fmt.Sprintf("Failed to query circuit breaker state: %v", err)
		return result
	}

	// Check if circuit breaker triggered.
	triggerCheck := ValidationCheck{
		Name:        "Circuit Breaker Trigger",
		Description: "Verify circuit breaker triggers on failures",
		Value:       cbState.TriggeredAt,
		Threshold:   v.config.CircuitBreakerConfig.ExpectedTriggerTime,
	}

	if cbState.State == "open" {
		triggerTime := cbState.TriggeredAt.Sub(startTime)
		if triggerTime <= v.config.CircuitBreakerConfig.ExpectedTriggerTime {
			triggerCheck.Passed = true
			triggerCheck.Details = fmt.Sprintf("Circuit breaker triggered in %v", triggerTime)
		} else {
			triggerCheck.Passed = false
			triggerCheck.Details = fmt.Sprintf("Circuit breaker trigger delayed: %v", triggerTime)
			result.Success = false
		}
	} else {
		triggerCheck.Passed = false
		triggerCheck.Details = "Circuit breaker did not trigger"
		result.Success = false
	}
	result.Checks = append(result.Checks, triggerCheck)

	// Check recovery behavior.
	if cbState.State == "half-open" || cbState.RecoveredAt != nil {
		recoveryCheck := ValidationCheck{
			Name:        "Circuit Breaker Recovery",
			Description: "Verify circuit breaker recovery",
			Passed:      true,
		}

		if cbState.RecoveredAt != nil {
			recoveryTime := cbState.RecoveredAt.Sub(*cbState.TriggeredAt)
			recoveryCheck.Value = recoveryTime
			recoveryCheck.Threshold = v.config.CircuitBreakerConfig.ExpectedRecoveryTime

			if recoveryTime <= v.config.CircuitBreakerConfig.ExpectedRecoveryTime {
				recoveryCheck.Details = fmt.Sprintf("Recovered in %v", recoveryTime)
			} else {
				recoveryCheck.Passed = false
				recoveryCheck.Details = fmt.Sprintf("Recovery delayed: %v", recoveryTime)
				result.Success = false
			}
		}
		result.Checks = append(result.Checks, recoveryCheck)
	}

	// Check error threshold behavior.
	errorCheck := ValidationCheck{
		Name:        "Error Threshold",
		Description: "Verify circuit breaker error threshold",
		Value:       cbState.ErrorRate,
		Threshold:   v.config.CircuitBreakerConfig.ErrorThresholdPercent,
		Passed:      cbState.ErrorRate >= v.config.CircuitBreakerConfig.ErrorThresholdPercent,
	}

	if errorCheck.Passed {
		errorCheck.Details = fmt.Sprintf("Error rate %.2f%% triggered circuit breaker", cbState.ErrorRate)
	} else {
		errorCheck.Details = fmt.Sprintf("Circuit breaker triggered with low error rate: %.2f%%", cbState.ErrorRate)
	}
	result.Checks = append(result.Checks, errorCheck)

	return result
}

// ValidateAutoScaling validates auto-scaling behavior during chaos.
func (v *ResilienceValidator) ValidateAutoScaling(ctx context.Context, experiment *Experiment) *ValidationResult {
	result := &ValidationResult{
		Success:   true,
		Timestamp: time.Now(),
		Checks:    []ValidationCheck{},
	}

	if !v.config.AutoScalingConfig.Enabled {
		result.Checks = append(result.Checks, ValidationCheck{
			Name:        "Auto-scaling",
			Description: "Auto-scaling validation disabled",
			Passed:      true,
		})
		return result
	}

	// Get HPA status.
	hpaList, err := v.getHPAStatus(ctx, experiment.Target.Namespace)
	if err != nil {
		result.Success = false
		result.Issues = fmt.Sprintf("Failed to get HPA status: %v", err)
		return result
	}

	for _, hpa := range hpaList {
		// Check scale-up behavior.
		scaleUpCheck := ValidationCheck{
			Name:        fmt.Sprintf("Scale-up: %s", hpa.Name),
			Description: "Verify auto-scaling triggers on high load",
			Value:       hpa.CurrentReplicas,
			Threshold:   hpa.TargetReplicas,
		}

		if hpa.CurrentCPU > v.config.AutoScalingConfig.TargetCPUPercent {
			if hpa.CurrentReplicas >= hpa.TargetReplicas {
				scaleUpCheck.Passed = true
				scaleUpCheck.Details = fmt.Sprintf("Scaled to %d replicas at %.2f%% CPU",
					hpa.CurrentReplicas, hpa.CurrentCPU)
			} else {
				scaleUpCheck.Passed = false
				scaleUpCheck.Details = fmt.Sprintf("Failed to scale: %d/%d replicas at %.2f%% CPU",
					hpa.CurrentReplicas, hpa.TargetReplicas, hpa.CurrentCPU)
				result.Success = false
			}
		} else {
			scaleUpCheck.Passed = true
			scaleUpCheck.Details = fmt.Sprintf("CPU %.2f%% below threshold", hpa.CurrentCPU)
		}
		result.Checks = append(result.Checks, scaleUpCheck)

		// Check replica bounds.
		boundsCheck := ValidationCheck{
			Name:        fmt.Sprintf("Replica bounds: %s", hpa.Name),
			Description: "Verify replicas stay within configured bounds",
			Value:       hpa.CurrentReplicas,
			Threshold:   fmt.Sprintf("%d-%d", v.config.AutoScalingConfig.MinReplicas, v.config.AutoScalingConfig.MaxReplicas),
			Passed: hpa.CurrentReplicas >= v.config.AutoScalingConfig.MinReplicas &&
				hpa.CurrentReplicas <= v.config.AutoScalingConfig.MaxReplicas,
		}

		if !boundsCheck.Passed {
			boundsCheck.Details = fmt.Sprintf("Replicas %d outside bounds", hpa.CurrentReplicas)
			result.Success = false
		} else {
			boundsCheck.Details = fmt.Sprintf("Replicas within bounds: %d", hpa.CurrentReplicas)
		}
		result.Checks = append(result.Checks, boundsCheck)
	}

	return result
}

// ValidateGracefulDegradation validates graceful degradation during failures.
func (v *ResilienceValidator) ValidateGracefulDegradation(ctx context.Context) *ValidationResult {
	result := &ValidationResult{
		Success:   true,
		Timestamp: time.Now(),
		Checks:    []ValidationCheck{},
	}

	if !v.config.DegradationConfig.Enabled {
		result.Checks = append(result.Checks, ValidationCheck{
			Name:        "Graceful Degradation",
			Description: "Graceful degradation validation disabled",
			Passed:      true,
		})
		return result
	}

	// Check degradation mode.
	degradationState, err := v.getDegradationState(ctx)
	if err != nil {
		result.Success = false
		result.Issues = fmt.Sprintf("Failed to get degradation state: %v", err)
		return result
	}

	// Verify acceptable degradation mode.
	modeCheck := ValidationCheck{
		Name:        "Degradation Mode",
		Description: "Verify system degrades to acceptable mode",
		Value:       degradationState.Mode,
		Threshold:   v.config.DegradationConfig.AcceptableDegradedMode,
	}

	isAcceptableMode := false
	for _, mode := range v.config.DegradationConfig.AcceptableDegradedMode {
		if mode == degradationState.Mode {
			isAcceptableMode = true
			break
		}
	}

	modeCheck.Passed = isAcceptableMode
	if isAcceptableMode {
		modeCheck.Details = fmt.Sprintf("System in acceptable degraded mode: %s", degradationState.Mode)
	} else {
		modeCheck.Details = fmt.Sprintf("System in unacceptable mode: %s", degradationState.Mode)
		result.Success = false
	}
	result.Checks = append(result.Checks, modeCheck)

	// Check required functionality.
	for _, functionality := range v.config.DegradationConfig.RequiredFunctionality {
		funcCheck := ValidationCheck{
			Name:        fmt.Sprintf("Required Function: %s", functionality),
			Description: "Verify critical functionality remains available",
			Value:       degradationState.AvailableFunctions[functionality],
			Threshold:   true,
		}

		if available, exists := degradationState.AvailableFunctions[functionality]; exists && available {
			funcCheck.Passed = true
			funcCheck.Details = fmt.Sprintf("%s is available", functionality)
		} else {
			funcCheck.Passed = false
			funcCheck.Details = fmt.Sprintf("%s is not available", functionality)
			result.Success = false
		}
		result.Checks = append(result.Checks, funcCheck)
	}

	// Check degradation duration.
	if degradationState.StartTime != nil {
		duration := time.Since(*degradationState.StartTime)
		durationCheck := ValidationCheck{
			Name:        "Degradation Duration",
			Description: "Verify degradation doesn't exceed maximum duration",
			Value:       duration,
			Threshold:   v.config.DegradationConfig.MaxDegradationDuration,
			Passed:      duration <= v.config.DegradationConfig.MaxDegradationDuration,
		}

		if durationCheck.Passed {
			durationCheck.Details = fmt.Sprintf("Degraded for %v", duration)
		} else {
			durationCheck.Details = fmt.Sprintf("Excessive degradation: %v", duration)
			result.Success = false
		}
		result.Checks = append(result.Checks, durationCheck)
	}

	return result
}

// PostExperimentValidation performs post-experiment validation.
func (v *ResilienceValidator) PostExperimentValidation(ctx context.Context, experiment *Experiment, result *ExperimentResult) *ValidationResult {
	validation := &ValidationResult{
		Success:   true,
		Timestamp: time.Now(),
		Checks:    []ValidationCheck{},
	}

	// Get current system state.
	currentState, err := v.captureSystemState(ctx)
	if err != nil {
		validation.Success = false
		validation.Issues = fmt.Sprintf("Failed to capture post-experiment state: %v", err)
		return validation
	}

	v.mu.RLock()
	baseline := v.baselineState
	v.mu.RUnlock()

	if baseline == nil {
		validation.Issues = "No baseline state available for comparison"
		return validation
	}

	// Check system recovery to baseline.
	recoveryChecks := []struct {
		name      string
		current   float64
		baseline  float64
		tolerance float64
	}{
		{"Availability", currentState.Availability, baseline.Availability, 0.5},
		{"Throughput", currentState.Throughput, baseline.Throughput, 5.0},
		{"Error Rate", currentState.ErrorRate, baseline.ErrorRate, 0.1},
		{"CPU Utilization", currentState.CPUUtilization, baseline.CPUUtilization, 10.0},
		{"Memory Utilization", currentState.MemoryUtilization, baseline.MemoryUtilization, 10.0},
	}

	for _, check := range recoveryChecks {
		diff := math.Abs(check.current - check.baseline)
		valCheck := ValidationCheck{
			Name:        fmt.Sprintf("Recovery: %s", check.name),
			Description: fmt.Sprintf("Verify %s returns to baseline", check.name),
			Value:       check.current,
			Threshold:   check.baseline,
			Passed:      diff <= check.tolerance,
		}

		if valCheck.Passed {
			valCheck.Details = fmt.Sprintf("%s recovered to %.2f (baseline: %.2f)",
				check.name, check.current, check.baseline)
		} else {
			valCheck.Details = fmt.Sprintf("%s not recovered: %.2f (baseline: %.2f, diff: %.2f)",
				check.name, check.current, check.baseline, diff)
			validation.Success = false
		}
		validation.Checks = append(validation.Checks, valCheck)
	}

	// Check data consistency.
	consistencyCheck := ValidationCheck{
		Name:        "Data Consistency",
		Description: "Verify data consistency after experiment",
		Passed:      !result.RecoveryMetrics.DataLoss,
	}

	if consistencyCheck.Passed {
		consistencyCheck.Details = "No data loss detected"
	} else {
		consistencyCheck.Details = "Data loss detected during experiment"
		validation.Success = false
	}
	validation.Checks = append(validation.Checks, consistencyCheck)

	// Check pod health.
	podHealthCheck := ValidationCheck{
		Name:        "Pod Health",
		Description: "Verify all pods are healthy",
		Value:       currentState.HealthyPods,
		Threshold:   currentState.ActivePods,
		Passed:      currentState.HealthyPods == currentState.ActivePods,
	}

	if podHealthCheck.Passed {
		podHealthCheck.Details = fmt.Sprintf("All %d pods healthy", currentState.ActivePods)
	} else {
		unhealthy := currentState.ActivePods - currentState.HealthyPods
		podHealthCheck.Details = fmt.Sprintf("%d pods unhealthy", unhealthy)
		validation.Success = false
	}
	validation.Checks = append(validation.Checks, podHealthCheck)

	return validation
}

// CollectMetrics collects current system metrics.
func (v *ResilienceValidator) CollectMetrics(ctx context.Context) MetricsSample {
	state, err := v.captureSystemState(ctx)
	if err != nil {
		v.logger.Error("Failed to collect metrics", zap.Error(err))
		return MetricsSample{}
	}

	return MetricsSample{
		Timestamp:    time.Now(),
		Availability: state.Availability,
		LatencyP50:   state.LatencyP50,
		LatencyP95:   state.LatencyP95,
		LatencyP99:   state.LatencyP99,
		Throughput:   state.Throughput,
		ErrorRate:    state.ErrorRate,
	}
}

// captureSystemState captures current system state.
func (v *ResilienceValidator) captureSystemState(ctx context.Context) (*SystemState, error) {
	state := &SystemState{}

	// Query availability metric.
	availability, err := v.queryMetric(ctx, `avg(up{job="intent-controller"}) * 100`)
	if err == nil && len(availability) > 0 {
		state.Availability = float64(availability[0].Value)
	} else {
		state.Availability = 100.0 // Default if metric not available
	}

	// Query latency metrics.
	latencyP50, err := v.queryMetric(ctx, `histogram_quantile(0.5, rate(http_request_duration_seconds_bucket[5m]))`)
	if err == nil && len(latencyP50) > 0 {
		state.LatencyP50 = time.Duration(float64(latencyP50[0].Value) * float64(time.Second))
	}

	latencyP95, err := v.queryMetric(ctx, `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`)
	if err == nil && len(latencyP95) > 0 {
		state.LatencyP95 = time.Duration(float64(latencyP95[0].Value) * float64(time.Second))
	}

	latencyP99, err := v.queryMetric(ctx, `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))`)
	if err == nil && len(latencyP99) > 0 {
		state.LatencyP99 = time.Duration(float64(latencyP99[0].Value) * float64(time.Second))
	}

	// Query throughput.
	throughput, err := v.queryMetric(ctx, `rate(intent_processed_total[1m]) * 60`)
	if err == nil && len(throughput) > 0 {
		state.Throughput = float64(throughput[0].Value)
	}

	// Query error rate.
	errorRate, err := v.queryMetric(ctx, `rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100`)
	if err == nil && len(errorRate) > 0 {
		state.ErrorRate = float64(errorRate[0].Value)
	}

	// Query resource utilization.
	cpuUtil, err := v.queryMetric(ctx, `avg(rate(container_cpu_usage_seconds_total[5m])) * 100`)
	if err == nil && len(cpuUtil) > 0 {
		state.CPUUtilization = float64(cpuUtil[0].Value)
	}

	memUtil, err := v.queryMetric(ctx, `avg(container_memory_usage_bytes / container_spec_memory_limit_bytes) * 100`)
	if err == nil && len(memUtil) > 0 {
		state.MemoryUtilization = float64(memUtil[0].Value)
	}

	// Get pod counts.
	podList := &corev1.PodList{}
	if err := v.client.List(ctx, podList); err == nil {
		state.ActivePods = len(podList.Items)
		for _, pod := range podList.Items {
			if pod.Status.Phase == corev1.PodPhase("Running") {
				allContainersReady := true
				for _, condition := range pod.Status.Conditions {
					if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
						allContainersReady = false
						break
					}
				}
				if allContainersReady {
					state.HealthyPods++
				}
			}
		}
	}

	return state, nil
}

// queryMetric queries Prometheus for a specific metric.
func (v *ResilienceValidator) queryMetric(ctx context.Context, query string) (model.Vector, error) {
	if v.promClient == nil {
		return nil, fmt.Errorf("Prometheus client not initialized")
	}

	result, warnings, err := v.promClient.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		v.logger.Warn("Prometheus query warnings",
			zap.String("query", query),
			zap.Strings("warnings", warnings))
	}

	vector, ok := result.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}

	return vector, nil
}

// CircuitBreakerState represents circuit breaker state.
type CircuitBreakerState struct {
	State       string // open, half-open, closed
	ErrorRate   float64
	TriggeredAt *time.Time
	RecoveredAt *time.Time
}

// queryCircuitBreakerState queries circuit breaker state.
func (v *ResilienceValidator) queryCircuitBreakerState(ctx context.Context) (*CircuitBreakerState, error) {
	state := &CircuitBreakerState{}

	// Query circuit breaker state metric.
	cbState, err := v.queryMetric(ctx, `circuit_breaker_state`)
	if err != nil {
		return nil, err
	}

	if len(cbState) > 0 {
		// Map numeric state to string.
		switch cbState[0].Value {
		case 0:
			state.State = "closed"
		case 1:
			state.State = "open"
		case 2:
			state.State = "half-open"
		}
	}

	// Query error rate.
	errorRate, err := v.queryMetric(ctx, `rate(circuit_breaker_errors_total[1m]) / rate(circuit_breaker_requests_total[1m]) * 100`)
	if err == nil && len(errorRate) > 0 {
		state.ErrorRate = float64(errorRate[0].Value)
	}

	// Query trigger time.
	triggerTime, err := v.queryMetric(ctx, `circuit_breaker_opened_timestamp`)
	if err == nil && len(triggerTime) > 0 {
		if float64(triggerTime[0].Value) >= 0 && float64(triggerTime[0].Value) <= float64(1<<62) {
			ts := time.Unix(int64(float64(triggerTime[0].Value)), 0)
			state.TriggeredAt = &ts
		}
	}

	// Query recovery time.
	recoveryTime, err := v.queryMetric(ctx, `circuit_breaker_closed_timestamp`)
	if err == nil && len(recoveryTime) > 0 {
		if float64(recoveryTime[0].Value) >= 0 && float64(recoveryTime[0].Value) <= float64(1<<62) {
			ts := time.Unix(int64(float64(recoveryTime[0].Value)), 0)
			state.RecoveredAt = &ts
		}
	}

	return state, nil
}

// HPAStatus represents HPA status.
type HPAStatus struct {
	Name            string
	CurrentReplicas int
	TargetReplicas  int
	CurrentCPU      float64
	TargetCPU       float64
}

// getHPAStatus gets HPA status for validation.
func (v *ResilienceValidator) getHPAStatus(ctx context.Context, namespace string) ([]HPAStatus, error) {
	// Query HPA metrics.
	hpaMetrics, err := v.queryMetric(ctx, fmt.Sprintf(`kube_horizontalpodautoscaler_status_current_replicas{namespace="%s"}`, namespace))
	if err != nil {
		return nil, err
	}

	statuses := []HPAStatus{}
	for _, metric := range hpaMetrics {
		hpaName := string(metric.Metric["horizontalpodautoscaler"])
		status := HPAStatus{
			Name: hpaName,
			// Safe integer conversion with bounds checking
			CurrentReplicas: safeFloatToInt(float64(metric.Value)),
		}

		// Get target replicas.
		targetReplicas, err := v.queryMetric(ctx, fmt.Sprintf(`kube_horizontalpodautoscaler_spec_target_replicas{horizontalpodautoscaler="%s"}`, hpaName))
		if err == nil && len(targetReplicas) > 0 {
			status.TargetReplicas = safeFloatToInt(float64(targetReplicas[0].Value))
		}

		// Get current CPU.
		currentCPU, err := v.queryMetric(ctx, fmt.Sprintf(`kube_horizontalpodautoscaler_status_current_metrics_value{horizontalpodautoscaler="%s",metric_name="cpu"}`, hpaName))
		if err == nil && len(currentCPU) > 0 {
			status.CurrentCPU = float64(currentCPU[0].Value)
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// DegradationState represents system degradation state.
type DegradationState struct {
	Mode               string
	StartTime          *time.Time
	AvailableFunctions map[string]bool
}

// getDegradationState gets current degradation state.
func (v *ResilienceValidator) getDegradationState(ctx context.Context) (*DegradationState, error) {
	state := &DegradationState{
		Mode:               "normal",
		AvailableFunctions: make(map[string]bool),
	}

	// Query degradation mode metric.
	degradationMode, err := v.queryMetric(ctx, `system_degradation_mode`)
	if err == nil && len(degradationMode) > 0 {
		// Map numeric mode to string.
		switch degradationMode[0].Value {
		case 1:
			state.Mode = "read-only"
		case 2:
			state.Mode = "cached-responses"
		case 3:
			state.Mode = "reduced-features"
		default:
			state.Mode = "normal"
		}
	}

	// Query degradation start time.
	startTime, err := v.queryMetric(ctx, `system_degradation_start_timestamp`)
	if err == nil && len(startTime) > 0 && startTime[0].Value > 0 {
		if float64(startTime[0].Value) >= 0 && float64(startTime[0].Value) <= float64(1<<62) {
			ts := time.Unix(int64(float64(startTime[0].Value)), 0)
			state.StartTime = &ts
		}
	}

	// Check function availability.
	functions := []string{"intent-processing", "monitoring", "health-check"}
	for _, function := range functions {
		query := fmt.Sprintf(`function_available{function="%s"}`, function)
		available, err := v.queryMetric(ctx, query)
		if err == nil && len(available) > 0 {
			state.AvailableFunctions[function] = available[0].Value == 1
		} else {
			// Assume available if metric not found.
			state.AvailableFunctions[function] = true
		}
	}

	return state, nil
}

// calculateSeverity calculates violation severity.
func (v *ResilienceValidator) calculateSeverity(metric string, actual, threshold float64) string {
	var deviation float64

	switch metric {
	case "availability":
		deviation = (threshold - actual) / threshold * 100
	case "latency", "error_rate":
		deviation = (actual - threshold) / threshold * 100
	case "throughput":
		deviation = (threshold - actual) / threshold * 100
	default:
		deviation = math.Abs(actual-threshold) / threshold * 100
	}

	switch {
	case deviation < 10:
		return "Minor"
	case deviation < 25:
		return "Major"
	default:
		return "Critical"
	}
}

// ValidateRecoveryTime validates system recovery time.
func (v *ResilienceValidator) ValidateRecoveryTime(ctx context.Context, startTime, endTime time.Time) *ValidationResult {
	result := &ValidationResult{
		Success:   true,
		Timestamp: time.Now(),
		Checks:    []ValidationCheck{},
	}

	recoveryDuration := endTime.Sub(startTime)

	// Check MTTR against target.
	mttrCheck := ValidationCheck{
		Name:        "Mean Time To Recovery",
		Description: "Verify system recovers within target MTTR",
		Value:       recoveryDuration,
		Threshold:   5 * time.Minute,
		Passed:      recoveryDuration <= 5*time.Minute,
	}

	if mttrCheck.Passed {
		mttrCheck.Details = fmt.Sprintf("System recovered in %v", recoveryDuration)
	} else {
		mttrCheck.Details = fmt.Sprintf("Recovery took %v, exceeding target", recoveryDuration)
		result.Success = false
	}
	result.Checks = append(result.Checks, mttrCheck)

	return result
}

// GetBaselineMetrics captures baseline metrics for comparison.
func (v *ResilienceValidator) GetBaselineMetrics(ctx context.Context) (*SystemState, error) {
	v.logger.Info("Capturing baseline metrics")

	// Collect multiple samples over baseline window.
	samples := []SystemState{}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	timeout := time.After(v.config.BaselineWindow)

	for {
		select {
		case <-ticker.C:
			state, err := v.captureSystemState(ctx)
			if err != nil {
				v.logger.Warn("Failed to capture baseline sample", zap.Error(err))
				continue
			}
			samples = append(samples, *state)
		case <-timeout:
			// Calculate average baseline.
			if len(samples) == 0 {
				return nil, fmt.Errorf("no baseline samples collected")
			}

			baseline := &SystemState{}
			for _, sample := range samples {
				baseline.Availability += sample.Availability
				baseline.LatencyP50 += sample.LatencyP50
				baseline.LatencyP95 += sample.LatencyP95
				baseline.LatencyP99 += sample.LatencyP99
				baseline.Throughput += sample.Throughput
				baseline.ErrorRate += sample.ErrorRate
				baseline.CPUUtilization += sample.CPUUtilization
				baseline.MemoryUtilization += sample.MemoryUtilization
			}

			count := float64(len(samples))
			baseline.Availability /= count
			baseline.LatencyP50 /= time.Duration(count)
			baseline.LatencyP95 /= time.Duration(count)
			baseline.LatencyP99 /= time.Duration(count)
			baseline.Throughput /= count
			baseline.ErrorRate /= count
			baseline.CPUUtilization /= count
			baseline.MemoryUtilization /= count

			return baseline, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// safeFloatToInt safely converts float64 to int with bounds checking to prevent G115 violations.
func safeFloatToInt(f float64) int {
	// Check for NaN, Inf, and range bounds
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}

	// Check bounds to prevent integer overflow
	const maxInt = int(^uint(0) >> 1)
	const minInt = -maxInt - 1

	if f > float64(maxInt) {
		return maxInt
	}
	if f < float64(minInt) {
		return minInt
	}

	return int(f)
}
