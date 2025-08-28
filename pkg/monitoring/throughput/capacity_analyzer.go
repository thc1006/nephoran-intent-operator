// Package throughput provides throughput monitoring and capacity analysis for network operations.
package throughput

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// CapacityAnalyzer provides advanced capacity planning and analysis.
type CapacityAnalyzer struct {
	mu sync.RWMutex

	// Capacity tracking.
	historicalThroughput []ThroughputDataPoint
	resourceUtilization  ResourceUtilization

	// Prediction models.
	linearRegression *LinearRegression

	// Prometheus metrics.
	capacityPredictionMetric *prometheus.GaugeVec
	resourceEfficiencyMetric *prometheus.GaugeVec
}

// ThroughputDataPoint represents a single throughput measurement.
type ThroughputDataPoint struct {
	Timestamp   time.Time
	Throughput  float64
	CPUUsage    float64
	MemoryUsage float64
}

// ResourceUtilization tracks system resource usage.
type ResourceUtilization struct {
	CPUCores      float64
	MemoryGB      float64
	IntentsPerCPU float64
	IntentsPerGB  float64
}

// NewCapacityAnalyzer creates a new capacity analyzer.
func NewCapacityAnalyzer() *CapacityAnalyzer {
	return &CapacityAnalyzer{
		historicalThroughput: make([]ThroughputDataPoint, 0, 1440), // 24 hours of minute-level data
		linearRegression:     NewLinearRegression(),
		capacityPredictionMetric: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_capacity_prediction",
			Help: "Predicted maximum sustainable throughput",
		}, []string{"metric"}),
		resourceEfficiencyMetric: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_resource_efficiency",
			Help: "Resource efficiency metrics",
		}, []string{"resource_type"}),
	}
}

// RecordThroughputDataPoint adds a new throughput measurement.
func (a *CapacityAnalyzer) RecordThroughputDataPoint(
	throughput float64,
	cpuUsage float64,
	memoryUsage float64,
) {
	a.mu.Lock()
	defer a.mu.Unlock()

	dataPoint := ThroughputDataPoint{
		Timestamp:   time.Now(),
		Throughput:  throughput,
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
	}

	// Add to historical data.
	a.historicalThroughput = append(a.historicalThroughput, dataPoint)

	// Limit historical data to 24 hours.
	if len(a.historicalThroughput) > 1440 {
		a.historicalThroughput = a.historicalThroughput[1:]
	}

	// Update linear regression.
	a.linearRegression.AddDataPoint(dataPoint.Timestamp, dataPoint.Throughput)

	// Update resource utilization.
	a.updateResourceUtilization(throughput, cpuUsage, memoryUsage)
}

// updateResourceUtilization calculates resource efficiency metrics.
func (a *CapacityAnalyzer) updateResourceUtilization(
	throughput, cpuUsage, memoryUsage float64,
) {
	// Calculate intents per resource unit.
	intentsPerCPU := throughput / math.Max(cpuUsage, 0.1)
	intentsPerGB := throughput / math.Max(memoryUsage, 0.1)

	a.resourceUtilization = ResourceUtilization{
		CPUCores:      cpuUsage,
		MemoryGB:      memoryUsage,
		IntentsPerCPU: intentsPerCPU,
		IntentsPerGB:  intentsPerGB,
	}

	// Update Prometheus metrics.
	a.resourceEfficiencyMetric.WithLabelValues("intents_per_cpu").Set(intentsPerCPU)
	a.resourceEfficiencyMetric.WithLabelValues("intents_per_gb").Set(intentsPerGB)
}

// PredictMaxThroughput provides capacity prediction.
func (a *CapacityAnalyzer) PredictMaxThroughput() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Use linear regression to predict maximum sustainable throughput.
	maxThroughput := a.linearRegression.Predict()

	// Update Prometheus metric.
	a.capacityPredictionMetric.WithLabelValues("max_sustainable").Set(maxThroughput)

	return maxThroughput
}

// GetBottlenecks identifies potential bottlenecks.
func (a *CapacityAnalyzer) GetBottlenecks() map[string]string {
	bottlenecks := make(map[string]string)

	// Check resource utilization.
	if a.resourceUtilization.CPUCores > 0.8 {
		bottlenecks["cpu"] = fmt.Sprintf(
			"High CPU usage: %.2f cores (>80%% utilization)",
			a.resourceUtilization.CPUCores,
		)
	}

	if a.resourceUtilization.MemoryGB > 0.9 {
		bottlenecks["memory"] = fmt.Sprintf(
			"High memory usage: %.2f GB (>90%% utilization)",
			a.resourceUtilization.MemoryGB,
		)
	}

	return bottlenecks
}

// GetScalingRecommendations provides scaling advice.
func (a *CapacityAnalyzer) GetScalingRecommendations() map[string]string {
	recommendations := make(map[string]string)

	// Predict future capacity needs.
	predictedThroughput := a.PredictMaxThroughput()
	currentThroughput := a.historicalThroughput[len(a.historicalThroughput)-1].Throughput

	// Calculate scaling ratio.
	scalingRatio := predictedThroughput / math.Max(currentThroughput, 1)

	if scalingRatio > 1.5 {
		recommendations["scale_up"] = fmt.Sprintf(
			"Recommended: Scale up by %.1fx to meet predicted demand",
			scalingRatio,
		)
	} else if scalingRatio < 0.5 {
		recommendations["scale_down"] = "Consider scaling down resources to optimize costs"
	}

	return recommendations
}

// LinearRegression provides simple linear regression for prediction.
type LinearRegression struct {
	mu         sync.Mutex
	timestamps []time.Time
	values     []float64
	slope      float64
	intercept  float64
}

// NewLinearRegression creates a new linear regression instance.
func NewLinearRegression() *LinearRegression {
	return &LinearRegression{
		timestamps: make([]time.Time, 0),
		values:     make([]float64, 0),
	}
}

// AddDataPoint adds a new data point to the regression.
func (lr *LinearRegression) AddDataPoint(timestamp time.Time, value float64) {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	lr.timestamps = append(lr.timestamps, timestamp)
	lr.values = append(lr.values, value)

	// Limit historical data.
	if len(lr.timestamps) > 1440 {
		lr.timestamps = lr.timestamps[1:]
		lr.values = lr.values[1:]
	}

	// Recalculate regression.
	lr.calculateRegression()
}

// calculateRegression performs linear regression calculation.
func (lr *LinearRegression) calculateRegression() {
	if len(lr.timestamps) < 2 {
		return
	}

	// Convert timestamps to numeric values (seconds since first timestamp).
	x := make([]float64, len(lr.timestamps))
	for i := range lr.timestamps {
		x[i] = lr.timestamps[i].Sub(lr.timestamps[0]).Seconds()
	}

	// Simple linear regression (least squares method).
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(x))

	for i := range x {
		sumX += x[i]
		sumY += lr.values[i]
		sumXY += x[i] * lr.values[i]
		sumX2 += x[i] * x[i]
	}

	// Calculate slope and intercept.
	lr.slope = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	lr.intercept = (sumY - lr.slope*sumX) / n
}

// Predict estimates future throughput.
func (lr *LinearRegression) Predict() float64 {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	if len(lr.timestamps) < 2 {
		return 45 // Default to claimed capacity
	}

	// Predict 1 hour into the future.
	futureTime := 3600.0 // 1 hour in seconds

	return lr.slope*futureTime + lr.intercept
}
