package throughput

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// LoadTester provides comprehensive load testing capabilities
type LoadTester struct {
	mu sync.Mutex

	// Test configuration
	config LoadTestConfig

	// Performance tracking
	results     []LoadTestResult
	currentTest *LoadTestRun

	// Metrics
	throughputMetric  *prometheus.HistogramVec
	latencyMetric     *prometheus.HistogramVec
	errorRateMetric   *prometheus.CounterVec
	concurrencyMetric prometheus.Gauge
}

// LoadTestConfig defines parameters for load testing
type LoadTestConfig struct {
	// Throughput targets
	MaxConcurrentIntents int
	IntentsPerMinute     int

	// Duration configuration
	TestDuration time.Duration

	// Failure simulation
	ErrorInjectionRate float64

	// Load profile
	LoadProfile LoadProfile
}

// LoadProfile defines varying load patterns
type LoadProfile string

const (
	ProfileSteady      LoadProfile = "steady"
	ProfileRampUp      LoadProfile = "ramp_up"
	ProfilePeakBurst   LoadProfile = "peak_burst"
	ProfileRandomSpike LoadProfile = "random_spike"
)

// LoadTestResult captures the outcome of a load test
type LoadTestResult struct {
	Timestamp         time.Time
	Throughput        float64
	Latency           time.Duration
	ErrorRate         float64
	ConcurrentIntents int
	Success           bool
}

// LoadTestRun tracks an ongoing load test
type LoadTestRun struct {
	StartTime      time.Time
	CompletedTests int
	Successful     int
	Failed         int
}

// NewLoadTester creates a new load tester
func NewLoadTester(config LoadTestConfig) *LoadTester {
	return &LoadTester{
		config:  config,
		results: make([]LoadTestResult, 0, 1000),
		throughputMetric: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_load_test_throughput",
			Help:    "Throughput during load testing",
			Buckets: prometheus.LinearBuckets(0, 10, 20),
		}, []string{"profile", "status"}),
		latencyMetric: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_load_test_latency",
			Help:    "Latency during load testing",
			Buckets: prometheus.DefBuckets,
		}, []string{"profile", "status"}),
		errorRateMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_load_test_errors",
			Help: "Error counts during load testing",
		}, []string{"profile", "type"}),
		concurrencyMetric: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_load_test_concurrency",
			Help: "Current concurrent intents during load test",
		}),
	}
}

// RunLoadTest executes a comprehensive load test
func (lt *LoadTester) RunLoadTest(ctx context.Context) ([]LoadTestResult, error) {
	lt.mu.Lock()
	lt.currentTest = &LoadTestRun{
		StartTime: time.Now(),
	}
	lt.mu.Unlock()

	// Apply context deadline or use default
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, lt.config.TestDuration)
		defer cancel()
	}

	// Select appropriate load generation strategy
	var generateLoad func(context.Context) error
	switch lt.config.LoadProfile {
	case ProfileSteady:
		generateLoad = lt.generateSteadyLoad
	case ProfileRampUp:
		generateLoad = lt.generateRampUpLoad
	case ProfilePeakBurst:
		generateLoad = lt.generatePeakBurstLoad
	case ProfileRandomSpike:
		generateLoad = lt.generateRandomSpikeLoad
	default:
		return nil, fmt.Errorf("unsupported load profile: %s", lt.config.LoadProfile)
	}

	// Run load generation
	err := generateLoad(ctx)

	// Compile and return results
	return lt.results, err
}

// generateSteadyLoad maintains consistent intent processing rate
func (lt *LoadTester) generateSteadyLoad(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute / time.Duration(lt.config.IntentsPerMinute))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			go lt.simulateIntentProcessing(lt.config.LoadProfile)
		}
	}
}

// generateRampUpLoad incrementally increases load
func (lt *LoadTester) generateRampUpLoad(ctx context.Context) error {
	rampDuration := lt.config.TestDuration / 4
	for phase := 1; phase <= 4; phase++ {
		// Calculate current phase's intents per minute
		currentRate := lt.config.IntentsPerMinute * phase / 4
		ticker := time.NewTicker(time.Minute / time.Duration(currentRate))

		phaseCtx, cancel := context.WithTimeout(ctx, rampDuration)
		defer cancel()

		for {
			select {
			case <-phaseCtx.Done():
				return nil
			case <-ticker.C:
				go lt.simulateIntentProcessing(lt.config.LoadProfile)
			}
		}
	}
	return nil
}

// generatePeakBurstLoad simulates high-intensity short bursts
func (lt *LoadTester) generatePeakBurstLoad(ctx context.Context) error {
	// Burst configuration
	burstDuration := 30 * time.Second
	cooldownDuration := 2 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// High-intensity burst
			burstCtx, cancel := context.WithTimeout(ctx, burstDuration)
			lt.runIntenseBurst(burstCtx)
			cancel()

			// Cooldown period
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cooldownDuration):
				continue
			}
		}
	}
}

// generateRandomSpikeLoad creates unpredictable load variations
func (lt *LoadTester) generateRandomSpikeLoad(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Random spike between 50-200% of base rate
			spikeMultiplier := 0.5 + rand.Float64()*1.5
			currentRate := int(float64(lt.config.IntentsPerMinute) * spikeMultiplier)

			ticker := time.NewTicker(time.Minute / time.Duration(currentRate))
			spikeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)

			for {
				select {
				case <-spikeCtx.Done():
					cancel()
					goto NextSpike
				case <-ticker.C:
					go lt.simulateIntentProcessing(lt.config.LoadProfile)
				}
			}
		NextSpike:
		}
	}
}

// runIntenseBurst simulates a high-concurrency burst
func (lt *LoadTester) runIntenseBurst(ctx context.Context) {
	// Create semaphore for max concurrency
	sem := make(chan struct{}, lt.config.MaxConcurrentIntents)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			sem <- struct{}{}
			go func() {
				defer func() { <-sem }()
				lt.simulateIntentProcessing(lt.config.LoadProfile)
			}()
		}
	}
}

// simulateIntentProcessing models an intent processing event
func (lt *LoadTester) simulateIntentProcessing(profile LoadProfile) {
	startTime := time.Now()
	lt.mu.Lock()
	lt.currentTest.CompletedTests++
	lt.mu.Unlock()

	// Update concurrency metric
	lt.concurrencyMetric.Inc()
	defer lt.concurrencyMetric.Dec()

	// Simulate processing with potential errors
	success := true
	var processingError error
	if rand.Float64() < lt.config.ErrorInjectionRate {
		success = false
		processingError = fmt.Errorf("simulated error")
	}

	// Calculate latency with some randomness
	latency := time.Duration(50+rand.Intn(200)) * time.Millisecond

	// Record result
	result := LoadTestResult{
		Timestamp:         startTime,
		Throughput:        float64(lt.config.IntentsPerMinute),
		Latency:           latency,
		ErrorRate:         lt.config.ErrorInjectionRate,
		ConcurrentIntents: lt.config.MaxConcurrentIntents,
		Success:           success,
	}

	lt.mu.Lock()
	lt.results = append(lt.results, result)

	// Track test outcomes
	if success {
		lt.currentTest.Successful++
	} else {
		lt.currentTest.Failed++
	}
	lt.mu.Unlock()

	// Update Prometheus metrics
	lt.throughputMetric.WithLabelValues(string(profile), fmt.Sprintf("%v", success)).Observe(result.Throughput)
	lt.latencyMetric.WithLabelValues(string(profile), fmt.Sprintf("%v", success)).Observe(result.Latency.Seconds())

	if processingError != nil {
		lt.errorRateMetric.WithLabelValues(string(profile), "simulated").Inc()
	}
}

// AnalyzeTestResults provides comprehensive load test analysis
func (lt *LoadTester) AnalyzeTestResults() map[string]interface{} {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	analysis := make(map[string]interface{})

	// Calculate overall metrics
	successRate := float64(lt.currentTest.Successful) / float64(lt.currentTest.CompletedTests)
	totalThroughput := 0.0
	latencyTotal := time.Duration(0)

	for _, result := range lt.results {
		totalThroughput += result.Throughput
		latencyTotal += result.Latency
	}

	analysis["total_tests"] = lt.currentTest.CompletedTests
	analysis["successful_tests"] = lt.currentTest.Successful
	analysis["failed_tests"] = lt.currentTest.Failed
	analysis["success_rate"] = successRate
	analysis["avg_throughput"] = totalThroughput / float64(len(lt.results))
	analysis["avg_latency"] = latencyTotal / time.Duration(len(lt.results))
	analysis["peak_concurrency"] = lt.config.MaxConcurrentIntents

	return analysis
}
