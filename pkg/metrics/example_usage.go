package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"
	"gonum.org/v1/gonum/floats"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// This is an example of how to use the dependencies in a metrics context

var (
	// Prometheus metric for rate limiting
	rateLimitedRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_rate_limited_requests_total",
			Help: "Total number of rate-limited requests",
		},
		[]string{"service"},
	)

	// Histogram for processing times
	processingTimeHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephoran_processing_time_seconds",
			Help:    "Processing time for network functions",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service"},
	)
)

// RateLimitedOperation demonstrates using rate limiting with Prometheus metrics
func RateLimitedOperation(serviceName string, limiter *rate.Limiter) error {
	if !limiter.Allow() {
		rateLimitedRequests.WithLabelValues(serviceName).Inc()
		return fmt.Errorf("rate limit exceeded for %s", serviceName)
	}

	start := time.Now()
	// Simulated operation
	defer func() {
		processingTimeHistogram.WithLabelValues(serviceName).Observe(time.Since(start).Seconds())
	}()

	return nil
}

// StatisticalAnalysis demonstrates using Gonum for statistical computations
func StatisticalAnalysis(data []float64) {
	// Compute statistical properties
	mean := stat.Mean(data, nil)
	stdDev := stat.StdDev(data, nil)

	// Find max and min
	max := floats.Max(data)
	min := floats.Min(data)

	// Optional: Matrix operations
	_ = mat.NewDense(len(data), 1, data)

	prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "nephoran_data_stats",
			Help: "Statistical properties of network data",
			ConstLabels: prometheus.Labels{
				"mean":   fmt.Sprintf("%f", mean),
				"stddev": fmt.Sprintf("%f", stdDev),
				"max":    fmt.Sprintf("%f", max),
				"min":    fmt.Sprintf("%f", min),
			},
		},
		func() float64 { return mean },
	)
}
