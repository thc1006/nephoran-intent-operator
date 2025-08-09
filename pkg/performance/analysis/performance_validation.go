package analysis

import (
	"fmt"
	"math"
	"time"
)

// PerformanceValidationConfig defines the performance expectations
type PerformanceValidationConfig struct {
	MaxLatency           time.Duration // Sub-2-second P95 latency
	MaxConcurrentIntents int           // 200+ concurrent intents
	MinThroughputPerMin  int           // 45 intents per minute
	MinAvailabilityRate  float64       // 99.95% availability
	MaxRetrievalLatency  time.Duration // Sub-200ms retrieval
	MinCacheHitRate      float64       // 87% cache hit rate
}

// PerformanceValidator provides methods to validate performance claims
type PerformanceValidator struct {
	config           PerformanceValidationConfig
	latencyData      []time.Duration
	availabilityData []float64
	cacheData        []float64
}

// NewPerformanceValidator creates a new validator instance
func NewPerformanceValidator(config PerformanceValidationConfig) *PerformanceValidator {
	return &PerformanceValidator{
		config: config,
	}
}

// AddLatencyMeasurement records a new latency measurement
func (pv *PerformanceValidator) AddLatencyMeasurement(duration time.Duration) {
	pv.latencyData = append(pv.latencyData, duration)
}

// AddAvailabilityMeasurement records availability percentage
func (pv *PerformanceValidator) AddAvailabilityMeasurement(availability float64) {
	pv.availabilityData = append(pv.availabilityData, availability)
}

// AddCacheMeasurement records cache hit rate
func (pv *PerformanceValidator) AddCacheMeasurement(hitRate float64) {
	pv.cacheData = append(pv.cacheData, hitRate)
}

// ValidateLatency checks if latency meets performance requirements
func (pv *PerformanceValidator) ValidateLatency() (bool, string) {
	if len(pv.latencyData) == 0 {
		return false, "No latency data available"
	}

	// Sort latencies to find P95
	sorted := make([]time.Duration, len(pv.latencyData))
	copy(sorted, pv.latencyData)
	// TODO: Implement sorting of durations

	p95Index := int(float64(len(sorted)) * 0.95)
	p95Latency := sorted[p95Index]

	if p95Latency > pv.config.MaxLatency {
		return false, fmt.Sprintf("P95 Latency exceeds threshold: %v > %v", p95Latency, pv.config.MaxLatency)
	}
	return true, "Latency requirements met"
}

// ValidateThroughput checks concurrent intent and throughput capabilities
func (pv *PerformanceValidator) ValidateThroughput(concurrentIntents int, intentsPerMinute int) (bool, string) {
	if concurrentIntents > pv.config.MaxConcurrentIntents {
		return false, fmt.Sprintf("Concurrent intents exceed limit: %d > %d",
			concurrentIntents, pv.config.MaxConcurrentIntents)
	}

	if intentsPerMinute < pv.config.MinThroughputPerMin {
		return false, fmt.Sprintf("Throughput below minimum: %d < %d",
			intentsPerMinute, pv.config.MinThroughputPerMin)
	}
	return true, "Throughput requirements met"
}

// ValidateAvailability checks system availability
func (pv *PerformanceValidator) ValidateAvailability() (bool, string) {
	if len(pv.availabilityData) == 0 {
		return false, "No availability data available"
	}

	avgAvailability := 0.0
	for _, availability := range pv.availabilityData {
		avgAvailability += availability
	}
	avgAvailability /= float64(len(pv.availabilityData))

	if avgAvailability < pv.config.MinAvailabilityRate {
		return false, fmt.Sprintf("Availability below threshold: %.4f < %.4f",
			avgAvailability, pv.config.MinAvailabilityRate)
	}
	return true, "Availability requirements met"
}

// ValidateCachePerformance checks cache hit rate
func (pv *PerformanceValidator) ValidateCachePerformance() (bool, string) {
	if len(pv.cacheData) == 0 {
		return false, "No cache performance data available"
	}

	avgCacheHitRate := 0.0
	for _, hitRate := range pv.cacheData {
		avgCacheHitRate += hitRate
	}
	avgCacheHitRate /= float64(len(pv.cacheData))

	if avgCacheHitRate < pv.config.MinCacheHitRate {
		return false, fmt.Sprintf("Cache hit rate below threshold: %.2f%% < %.2f%%",
			avgCacheHitRate*100, pv.config.MinCacheHitRate*100)
	}
	return true, "Cache performance requirements met"
}

// GeneratePerformanceReport creates a comprehensive validation report
func (pv *PerformanceValidator) GeneratePerformanceReport() PerformanceReport {
	report := PerformanceReport{
		ValidationTimestamp: time.Now(),
	}

	// Validate each performance metric
	var overallPass = true
	var failureReasons []string

	// Latency Validation
	latencyPass, latencyMsg := pv.ValidateLatency()
	report.LatencyValidation = ValidationResult{
		Passed:  latencyPass,
		Message: latencyMsg,
	}
	if !latencyPass {
		overallPass = false
		failureReasons = append(failureReasons, latencyMsg)
	}

	// Availability Validation
	availPass, availMsg := pv.ValidateAvailability()
	report.AvailabilityValidation = ValidationResult{
		Passed:  availPass,
		Message: availMsg,
	}
	if !availPass {
		overallPass = false
		failureReasons = append(failureReasons, availMsg)
	}

	// Cache Performance Validation
	cachePass, cacheMsg := pv.ValidateCachePerformance()
	report.CacheValidation = ValidationResult{
		Passed:  cachePass,
		Message: cacheMsg,
	}
	if !cachePass {
		overallPass = false
		failureReasons = append(failureReasons, cacheMsg)
	}

	report.OverallValidation = ValidationResult{
		Passed:  overallPass,
		Message: fmt.Sprintf("Performance validation %v", map[bool]string{true: "PASSED", false: "FAILED"}[overallPass]),
	}
	report.FailureReasons = failureReasons

	return report
}

// Structs for reporting
type ValidationResult struct {
	Passed  bool
	Message string
}

type PerformanceReport struct {
	ValidationTimestamp    time.Time
	LatencyValidation      ValidationResult
	AvailabilityValidation ValidationResult
	CacheValidation        ValidationResult
	OverallValidation      ValidationResult
	FailureReasons         []string
}
