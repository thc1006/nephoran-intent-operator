package performance

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// MetricsAnalyzer provides advanced metrics analysis capabilities.
type MetricsAnalyzer struct {
	samples        map[string][]float64
	latencies      map[string][]time.Duration
	throughput     map[string][]float64
	resourceUsage  map[string][]ResourceSnapshot
	anomalies      []Anomaly
	bottlenecks    []MetricsBottleneck
	mu             sync.RWMutex
	historyWindow  time.Duration
	analysisConfig AnalysisConfig
}

// ResourceSnapshot captures resource state at a point in time.
type ResourceSnapshot struct {
	Timestamp      time.Time
	CPUPercent     float64
	MemoryMB       float64
	GoroutineCount int
	NetworkIO      NetworkMetrics
	DiskIO         DiskMetrics
}

// Anomaly represents a detected performance anomaly.
type Anomaly struct {
	Timestamp   time.Time
	Type        string
	Metric      string
	Value       float64
	Expected    float64
	Deviation   float64
	Severity    string
	Description string
}

// MetricsBottleneck represents a detected performance bottleneck from metrics analysis.
type MetricsBottleneck struct {
	Component   string
	Type        string
	Impact      float64
	Description string
	Metrics     map[string]float64
	Suggestions []string
}

// AnalysisConfig configures the analysis parameters.
type AnalysisConfig struct {
	AnomalyThreshold       float64 // Standard deviations for anomaly detection
	BottleneckThreshold    float64 // Percentage threshold for bottleneck detection
	HistoryWindow          time.Duration
	SamplingInterval       time.Duration
	EnableAutoOptimization bool
}

// NewMetricsAnalyzer creates a new metrics analyzer.
func NewMetricsAnalyzer() *MetricsAnalyzer {
	return &MetricsAnalyzer{
		samples:       make(map[string][]float64),
		latencies:     make(map[string][]time.Duration),
		throughput:    make(map[string][]float64),
		resourceUsage: make(map[string][]ResourceSnapshot),
		anomalies:     make([]Anomaly, 0),
		bottlenecks:   make([]MetricsBottleneck, 0),
		historyWindow: 10 * time.Minute,
		analysisConfig: AnalysisConfig{
			AnomalyThreshold:       3.0,  // 3 standard deviations
			BottleneckThreshold:    80.0, // 80% utilization
			HistoryWindow:          10 * time.Minute,
			SamplingInterval:       100 * time.Millisecond,
			EnableAutoOptimization: true,
		},
	}
}

// CalculateAverage calculates the average of a slice of durations.
func (ma *MetricsAnalyzer) CalculateAverage(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}

	var sum time.Duration
	for _, v := range values {
		sum += v
	}
	return sum / time.Duration(len(values))
}

// CalculateAverageFloat calculates the average of float values.
func (ma *MetricsAnalyzer) CalculateAverageFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// CalculatePercentile calculates the Nth percentile of latencies.
func (ma *MetricsAnalyzer) CalculatePercentile(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}

	// Sort values.
	sorted := make([]time.Duration, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate percentile index.
	index := int(math.Ceil(float64(len(sorted)) * percentile / 100.0))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	if index < 0 {
		index = 0
	}

	return sorted[index]
}

// CalculatePercentileFloat calculates the Nth percentile of float values.
func (ma *MetricsAnalyzer) CalculatePercentileFloat(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values.
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Calculate percentile index.
	index := int(math.Ceil(float64(len(sorted)) * percentile / 100.0))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	if index < 0 {
		index = 0
	}

	return sorted[index]
}

// CalculateMax returns the maximum value from a slice of durations.
func (ma *MetricsAnalyzer) CalculateMax(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}

	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

// CalculateMaxFloat returns the maximum value from a slice of floats.
func (ma *MetricsAnalyzer) CalculateMaxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

// CalculateThroughput calculates requests per second.
func (ma *MetricsAnalyzer) CalculateThroughput(requestCount int64, duration time.Duration) float64 {
	if duration == 0 {
		return 0
	}
	return float64(requestCount) / duration.Seconds()
}

// CalculateResourceEfficiency calculates resource efficiency metrics.
func (ma *MetricsAnalyzer) CalculateResourceEfficiency(cpu float64, memory float64, throughput float64) map[string]float64 {
	efficiency := make(map[string]float64)

	// CPU efficiency: throughput per CPU percent.
	if cpu > 0 {
		efficiency["cpu_per_request"] = cpu / throughput
		efficiency["requests_per_cpu"] = throughput / cpu
	}

	// Memory efficiency: throughput per MB.
	if memory > 0 {
		efficiency["memory_per_request"] = memory / throughput
		efficiency["requests_per_mb"] = throughput / memory
	}

	// Overall efficiency score (0-100).
	cpuScore := math.Min(100, (throughput/cpu)*10)
	memScore := math.Min(100, (throughput/memory)*10)
	efficiency["overall_score"] = (cpuScore + memScore) / 2

	return efficiency
}

// DetectBottlenecks analyzes metrics to identify performance bottlenecks.
func (ma *MetricsAnalyzer) DetectBottlenecks(metrics map[string]interface{}) []MetricsBottleneck {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	bottlenecks := []MetricsBottleneck{}

	// CPU bottleneck detection.
	if cpu, ok := metrics["cpu_percent"].(float64); ok && cpu > ma.analysisConfig.BottleneckThreshold {
		bottlenecks = append(bottlenecks, MetricsBottleneck{
			Component:   "CPU",
			Type:        "Resource Saturation",
			Impact:      cpu,
			Description: fmt.Sprintf("CPU utilization at %.2f%% exceeds threshold", cpu),
			Metrics: map[string]float64{
				"cpu_percent": cpu,
				"threshold":   ma.analysisConfig.BottleneckThreshold,
			},
			Suggestions: []string{
				"Consider horizontal scaling",
				"Optimize CPU-intensive operations",
				"Review goroutine usage patterns",
				"Enable CPU profiling to identify hot spots",
			},
		})
	}

	// Memory bottleneck detection.
	if memory, ok := metrics["memory_mb"].(float64); ok {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		memoryPercent := (memory / float64(memStats.Sys/1024/1024)) * 100

		if memoryPercent > ma.analysisConfig.BottleneckThreshold {
			bottlenecks = append(bottlenecks, MetricsBottleneck{
				Component:   "Memory",
				Type:        "Memory Pressure",
				Impact:      memoryPercent,
				Description: fmt.Sprintf("Memory usage at %.2f%% of system memory", memoryPercent),
				Metrics: map[string]float64{
					"memory_mb":      memory,
					"memory_percent": memoryPercent,
					"gc_pause_ms":    float64(memStats.PauseNs[(memStats.NumGC+255)%256]) / 1e6,
				},
				Suggestions: []string{
					"Review memory allocations",
					"Implement object pooling",
					"Optimize cache sizes",
					"Consider memory profiling",
				},
			})
		}
	}

	// Goroutine bottleneck detection.
	if goroutines, ok := metrics["goroutines"].(int); ok && goroutines > 1000 {
		bottlenecks = append(bottlenecks, MetricsBottleneck{
			Component:   "Goroutines",
			Type:        "Concurrency Issue",
			Impact:      float64(goroutines),
			Description: fmt.Sprintf("High goroutine count: %d", goroutines),
			Metrics: map[string]float64{
				"goroutine_count": float64(goroutines),
			},
			Suggestions: []string{
				"Review goroutine lifecycle management",
				"Implement goroutine pools",
				"Check for goroutine leaks",
				"Use context for cancellation",
			},
		})
	}

	// Latency bottleneck detection.
	if latencyP95, ok := metrics["latency_p95_ms"].(float64); ok && latencyP95 > 5000 {
		bottlenecks = append(bottlenecks, MetricsBottleneck{
			Component:   "Latency",
			Type:        "High Response Time",
			Impact:      latencyP95 / 1000, // Convert to seconds
			Description: fmt.Sprintf("P95 latency at %.2fs exceeds target", latencyP95/1000),
			Metrics: map[string]float64{
				"latency_p95_ms": latencyP95,
				"target_ms":      5000,
			},
			Suggestions: []string{
				"Enable request caching",
				"Optimize database queries",
				"Review network round trips",
				"Consider asynchronous processing",
			},
		})
	}

	// Queue bottleneck detection.
	if queueSize, ok := metrics["queue_size"].(int); ok && queueSize > 100 {
		bottlenecks = append(bottlenecks, MetricsBottleneck{
			Component:   "Queue",
			Type:        "Backpressure",
			Impact:      float64(queueSize),
			Description: fmt.Sprintf("Queue backlog of %d items", queueSize),
			Metrics: map[string]float64{
				"queue_size": float64(queueSize),
			},
			Suggestions: []string{
				"Increase worker pool size",
				"Optimize processing time",
				"Implement batch processing",
				"Consider rate limiting",
			},
		})
	}

	ma.bottlenecks = append(ma.bottlenecks, bottlenecks...)
	return bottlenecks
}

// DetectAnomalies uses statistical analysis to detect performance anomalies.
func (ma *MetricsAnalyzer) DetectAnomalies(metricName string, currentValue float64) *Anomaly {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	samples, exists := ma.samples[metricName]
	if !exists || len(samples) < 10 {
		// Not enough data for anomaly detection.
		ma.samples[metricName] = append(ma.samples[metricName], currentValue)
		return nil
	}

	// Calculate statistics.
	mean := ma.CalculateAverageFloat(samples)
	stdDev := ma.calculateStandardDeviation(samples)

	// Check if current value is anomalous.
	zScore := math.Abs(currentValue-mean) / stdDev
	if zScore > ma.analysisConfig.AnomalyThreshold {
		anomaly := &Anomaly{
			Timestamp:   time.Now(),
			Type:        "Statistical",
			Metric:      metricName,
			Value:       currentValue,
			Expected:    mean,
			Deviation:   zScore,
			Severity:    ma.getSeverity(zScore),
			Description: fmt.Sprintf("%s value %.2f deviates %.2f standard deviations from mean %.2f", metricName, currentValue, zScore, mean),
		}

		ma.anomalies = append(ma.anomalies, *anomaly)
		return anomaly
	}

	// Add to samples (sliding window).
	if len(samples) > 1000 {
		ma.samples[metricName] = samples[1:]
	}
	ma.samples[metricName] = append(ma.samples[metricName], currentValue)

	return nil
}

// calculateStandardDeviation calculates the standard deviation of values.
func (ma *MetricsAnalyzer) calculateStandardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := ma.CalculateAverageFloat(values)
	var sumSquares float64

	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(values)-1)
	return math.Sqrt(variance)
}

// getSeverity determines anomaly severity based on deviation.
func (ma *MetricsAnalyzer) getSeverity(zScore float64) string {
	switch {
	case zScore > 5:
		return "CRITICAL"
	case zScore > 4:
		return "HIGH"
	case zScore > 3:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// AnalyzeLatencyPattern analyzes latency patterns for optimization opportunities.
func (ma *MetricsAnalyzer) AnalyzeLatencyPattern(latencies []time.Duration) map[string]interface{} {
	if len(latencies) == 0 {
		return nil
	}

	analysis := make(map[string]interface{})

	// Basic statistics.
	analysis["count"] = len(latencies)
	analysis["average"] = ma.CalculateAverage(latencies)
	analysis["p50"] = ma.CalculatePercentile(latencies, 50)
	analysis["p95"] = ma.CalculatePercentile(latencies, 95)
	analysis["p99"] = ma.CalculatePercentile(latencies, 99)
	analysis["max"] = ma.CalculateMax(latencies)

	// Convert to float64 for advanced analysis.
	values := make([]float64, len(latencies))
	for i, l := range latencies {
		values[i] = l.Seconds()
	}

	// Distribution analysis.
	analysis["std_dev"] = ma.calculateStandardDeviation(values)
	analysis["coefficient_of_variation"] = ma.calculateCoefficientOfVariation(values)

	// Trend analysis.
	if len(values) > 10 {
		analysis["trend"] = ma.calculateTrend(values)
		analysis["is_degrading"] = ma.isPerformanceDegrading(values)
	}

	// Outlier detection.
	outliers := ma.detectOutliers(values)
	analysis["outlier_count"] = len(outliers)
	analysis["outlier_percentage"] = float64(len(outliers)) / float64(len(values)) * 100

	return analysis
}

// calculateCoefficientOfVariation calculates CV for stability analysis.
func (ma *MetricsAnalyzer) calculateCoefficientOfVariation(values []float64) float64 {
	mean := ma.CalculateAverageFloat(values)
	if mean == 0 {
		return 0
	}
	stdDev := ma.calculateStandardDeviation(values)
	return (stdDev / mean) * 100
}

// calculateTrend calculates the trend direction of values.
func (ma *MetricsAnalyzer) calculateTrend(values []float64) string {
	if len(values) < 2 {
		return "stable"
	}

	// Simple linear regression.
	n := float64(len(values))
	var sumX, sumY, sumXY, sumX2 float64

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine trend based on slope.
	mean := sumY / n
	slopePercent := (slope / mean) * 100

	switch {
	case slopePercent > 5:
		return "increasing"
	case slopePercent < -5:
		return "decreasing"
	default:
		return "stable"
	}
}

// isPerformanceDegrading checks if performance is degrading over time.
func (ma *MetricsAnalyzer) isPerformanceDegrading(values []float64) bool {
	if len(values) < 20 {
		return false
	}

	// Compare recent performance with historical.
	midPoint := len(values) / 2
	historical := values[:midPoint]
	recent := values[midPoint:]

	historicalAvg := ma.CalculateAverageFloat(historical)
	recentAvg := ma.CalculateAverageFloat(recent)

	// Performance is degrading if recent average is 20% worse.
	return recentAvg > historicalAvg*1.2
}

// detectOutliers identifies outlier values using IQR method.
func (ma *MetricsAnalyzer) detectOutliers(values []float64) []int {
	if len(values) < 4 {
		return []int{}
	}

	q1 := ma.CalculatePercentileFloat(values, 25)
	q3 := ma.CalculatePercentileFloat(values, 75)
	iqr := q3 - q1

	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	outliers := []int{}
	for i, v := range values {
		if v < lowerBound || v > upperBound {
			outliers = append(outliers, i)
		}
	}

	return outliers
}

// RecordLatency records a latency measurement.
func (ma *MetricsAnalyzer) RecordLatency(operation string, latency time.Duration) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if ma.latencies[operation] == nil {
		ma.latencies[operation] = make([]time.Duration, 0)
	}

	ma.latencies[operation] = append(ma.latencies[operation], latency)

	// Maintain sliding window.
	if len(ma.latencies[operation]) > 10000 {
		ma.latencies[operation] = ma.latencies[operation][1:]
	}

	// Check for anomaly.
	if anomaly := ma.DetectAnomalies(operation+"_latency", latency.Seconds()); anomaly != nil {
		klog.Warningf("Latency anomaly detected: %v", anomaly)
	}
}

// RecordThroughput records throughput measurement.
func (ma *MetricsAnalyzer) RecordThroughput(operation string, throughput float64) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if ma.throughput[operation] == nil {
		ma.throughput[operation] = make([]float64, 0)
	}

	ma.throughput[operation] = append(ma.throughput[operation], throughput)

	// Maintain sliding window.
	if len(ma.throughput[operation]) > 1000 {
		ma.throughput[operation] = ma.throughput[operation][1:]
	}
}

// RecordResourceSnapshot records current resource usage.
func (ma *MetricsAnalyzer) RecordResourceSnapshot(snapshot ResourceSnapshot) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	key := "system"
	if ma.resourceUsage[key] == nil {
		ma.resourceUsage[key] = make([]ResourceSnapshot, 0)
	}

	ma.resourceUsage[key] = append(ma.resourceUsage[key], snapshot)

	// Maintain sliding window based on time.
	cutoff := time.Now().Add(-ma.historyWindow)
	filtered := make([]ResourceSnapshot, 0)
	for _, s := range ma.resourceUsage[key] {
		if s.Timestamp.After(cutoff) {
			filtered = append(filtered, s)
		}
	}
	ma.resourceUsage[key] = filtered

	// Detect resource anomalies.
	ma.DetectAnomalies("cpu_percent", snapshot.CPUPercent)
	ma.DetectAnomalies("memory_mb", snapshot.MemoryMB)
	ma.DetectAnomalies("goroutines", float64(snapshot.GoroutineCount))
}

// GetPerformanceSummary returns a comprehensive performance summary.
func (ma *MetricsAnalyzer) GetPerformanceSummary() map[string]interface{} {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	summary := make(map[string]interface{})

	// Latency summaries.
	latencySummary := make(map[string]interface{})
	for operation, latencies := range ma.latencies {
		if len(latencies) > 0 {
			latencySummary[operation] = ma.AnalyzeLatencyPattern(latencies)
		}
	}
	summary["latencies"] = latencySummary

	// Throughput summaries.
	throughputSummary := make(map[string]interface{})
	for operation, values := range ma.throughput {
		if len(values) > 0 {
			throughputSummary[operation] = map[string]float64{
				"average": ma.CalculateAverageFloat(values),
				"p50":     ma.CalculatePercentileFloat(values, 50),
				"p95":     ma.CalculatePercentileFloat(values, 95),
				"max":     ma.CalculateMaxFloat(values),
			}
		}
	}
	summary["throughput"] = throughputSummary

	// Resource usage summaries.
	if snapshots, exists := ma.resourceUsage["system"]; exists && len(snapshots) > 0 {
		cpuValues := make([]float64, len(snapshots))
		memValues := make([]float64, len(snapshots))
		goroutineValues := make([]float64, len(snapshots))

		for i, s := range snapshots {
			cpuValues[i] = s.CPUPercent
			memValues[i] = s.MemoryMB
			goroutineValues[i] = float64(s.GoroutineCount)
		}

		summary["resources"] = map[string]interface{}{
			"cpu": map[string]float64{
				"average": ma.CalculateAverageFloat(cpuValues),
				"p95":     ma.CalculatePercentileFloat(cpuValues, 95),
				"max":     ma.CalculateMaxFloat(cpuValues),
			},
			"memory": map[string]float64{
				"average": ma.CalculateAverageFloat(memValues),
				"p95":     ma.CalculatePercentileFloat(memValues, 95),
				"max":     ma.CalculateMaxFloat(memValues),
			},
			"goroutines": map[string]float64{
				"average": ma.CalculateAverageFloat(goroutineValues),
				"p95":     ma.CalculatePercentileFloat(goroutineValues, 95),
				"max":     ma.CalculateMaxFloat(goroutineValues),
			},
		}
	}

	// Recent anomalies.
	recentAnomalies := []Anomaly{}
	cutoff := time.Now().Add(-5 * time.Minute)
	for _, anomaly := range ma.anomalies {
		if anomaly.Timestamp.After(cutoff) {
			recentAnomalies = append(recentAnomalies, anomaly)
		}
	}
	summary["recent_anomalies"] = recentAnomalies

	// Current bottlenecks.
	summary["bottlenecks"] = ma.bottlenecks

	return summary
}

// ExportToPrometheus exports metrics to Prometheus.
func (ma *MetricsAnalyzer) ExportToPrometheus(registry *prometheus.Registry) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	// Export latency metrics.
	for operation, latencies := range ma.latencies {
		if len(latencies) > 0 {
			// Create histogram.
			histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name:    fmt.Sprintf("analyzer_%s_latency_seconds", operation),
				Help:    fmt.Sprintf("Latency distribution for %s operation", operation),
				Buckets: prometheus.DefBuckets,
			}, []string{"operation"})

			for _, latency := range latencies {
				histogram.WithLabelValues(operation).Observe(latency.Seconds())
			}

			registry.MustRegister(histogram)
		}
	}

	// Export throughput metrics.
	for operation, values := range ma.throughput {
		if len(values) > 0 {
			gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: fmt.Sprintf("analyzer_%s_throughput_rps", operation),
				Help: fmt.Sprintf("Throughput for %s operation", operation),
			}, []string{"operation", "metric"})

			gauge.WithLabelValues(operation, "average").Set(ma.CalculateAverageFloat(values))
			gauge.WithLabelValues(operation, "p95").Set(ma.CalculatePercentileFloat(values, 95))

			registry.MustRegister(gauge)
		}
	}

	// Export anomaly metrics.
	anomalyGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analyzer_anomalies_total",
		Help: "Total number of detected anomalies",
	}, []string{"severity"})

	anomalyCounts := make(map[string]int)
	for _, anomaly := range ma.anomalies {
		anomalyCounts[anomaly.Severity]++
	}

	for severity, count := range anomalyCounts {
		anomalyGauge.WithLabelValues(severity).Set(float64(count))
	}

	registry.MustRegister(anomalyGauge)

	// Export bottleneck metrics.
	bottleneckGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "analyzer_bottlenecks_total",
		Help: "Total number of detected bottlenecks",
	}, []string{"component", "type"})

	for _, bottleneck := range ma.bottlenecks {
		bottleneckGauge.WithLabelValues(bottleneck.Component, bottleneck.Type).Set(bottleneck.Impact)
	}

	registry.MustRegister(bottleneckGauge)
}
