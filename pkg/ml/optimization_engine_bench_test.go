//go:build ml && !test

package ml

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// mockPrometheusAPI implements v1.API for benchmarking
type mockPrometheusAPI struct {
	queryDelay time.Duration
	dataPoints int
}

func (m *mockPrometheusAPI) QueryRange(ctx context.Context, query string, r v1.Range, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	// Simulate network delay
	if m.queryDelay > 0 {
		time.Sleep(m.queryDelay)
	}

	// Generate mock data
	matrix := make(model.Matrix, 1)
	values := make([]model.SamplePair, 0, m.dataPoints)

	step := r.Step.Seconds()
	current := float64(r.Start.Unix())
	end := float64(r.End.Unix())

	for current <= end {
		values = append(values, model.SamplePair{
			Timestamp: model.Time(current * 1000),                // Convert to milliseconds
			Value:     model.SampleValue(50 + (current/3600)%50), // Varying values
		})
		current += step
	}

	matrix[0] = &model.SampleStream{
		Metric: model.Metric{"__name__": "test_metric"},
		Values: values,
	}

	return matrix, nil, nil
}

// Implement other required methods with stubs
func (m *mockPrometheusAPI) Query(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Alerts(ctx context.Context) (v1.AlertsResult, error) {
	return v1.AlertsResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) AlertManagers(ctx context.Context) (v1.AlertManagersResult, error) {
	return v1.AlertManagersResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) CleanTombstones(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Config(ctx context.Context) (v1.ConfigResult, error) {
	return v1.ConfigResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) DeleteSeries(ctx context.Context, matches []string, startTime, endTime time.Time) error {
	return fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Flags(ctx context.Context) (v1.FlagsResult, error) {
	return v1.FlagsResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) LabelNames(ctx context.Context, matches []string, startTime, endTime time.Time) ([]string, v1.Warnings, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time) (model.LabelValues, v1.Warnings, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Series(ctx context.Context, matches []string, startTime, endTime time.Time) ([]model.LabelSet, v1.Warnings, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Snapshot(ctx context.Context, skipHead bool) (v1.SnapshotResult, error) {
	return v1.SnapshotResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Rules(ctx context.Context) (v1.RulesResult, error) {
	return v1.RulesResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Targets(ctx context.Context) (v1.TargetsResult, error) {
	return v1.TargetsResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) TargetsMetadata(ctx context.Context, matchTarget, metric, limit string) ([]v1.MetricMetadata, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Metadata(ctx context.Context, metric, limit string) (map[string][]v1.Metadata, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) TSDB(ctx context.Context) (v1.TSDBResult, error) {
	return v1.TSDBResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) WalReplay(ctx context.Context) (v1.WalReplayStatus, error) {
	return v1.WalReplayStatus{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) Runtimeinfo(ctx context.Context) (v1.RuntimeinfoResult, error) {
	return v1.RuntimeinfoResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) BuildInfo(ctx context.Context) (v1.BuildInfoResult, error) {
	return v1.BuildInfoResult{}, fmt.Errorf("not implemented")
}

func (m *mockPrometheusAPI) QueryExemplars(ctx context.Context, query string, startTime, endTime time.Time) ([]v1.ExemplarQueryResult, error) {
	return nil, fmt.Errorf("not implemented")
}

// Benchmark helper to create test engine
func createBenchmarkEngine(b *testing.B, dataPoints int, queryDelay time.Duration) *OptimizationEngine {
	config := &OptimizationConfig{
		PrometheusURL:           "http://mock-prometheus:9090",
		ModelUpdateInterval:     time.Hour,
		PredictionHorizon:       24 * time.Hour,
		OptimizationThreshold:   0.7,
		EnablePredictiveScaling: true,
		EnableAnomalyDetection:  true,
		EnableResourceOptim:     true,
		MLModelConfig: &MLModelConfig{
			TrafficPrediction: &ModelConfig{
				Algorithm: "linear",
				Enabled:   true,
			},
			ResourceOptimization: &ModelConfig{
				Enabled: true,
			},
			AnomalyDetection: &ModelConfig{
				Enabled: true,
			},
			IntentClassification: &ModelConfig{
				Enabled: true,
			},
		},
	}

	engine, err := NewOptimizationEngine(config)
	if err != nil {
		b.Fatal(err)
	}

	// Replace Prometheus client with mock
	engine.prometheusClient = &mockPrometheusAPI{
		queryDelay: queryDelay,
		dataPoints: dataPoints,
	}

	return engine
}

// BenchmarkOptimizeNetworkDeployment tests the main optimization function
func BenchmarkOptimizeNetworkDeployment(b *testing.B) {
	testCases := []struct {
		name       string
		dataPoints int
		queryDelay time.Duration
	}{
		{"Small-NoDelay", 24, 0},   // 24 hours of data
		{"Medium-NoDelay", 168, 0}, // 7 days of data
		{"Large-NoDelay", 720, 0},  // 30 days of data
		{"Small-NetworkDelay", 24, 10 * time.Millisecond},
		{"Medium-NetworkDelay", 168, 10 * time.Millisecond},
		{"Large-NetworkDelay", 720, 10 * time.Millisecond},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			engine := createBenchmarkEngine(b, tc.dataPoints, tc.queryDelay)
			intent := &NetworkIntent{
				ID:          "test-intent",
				Description: "Benchmark test intent",
				Priority:    "high",
				Parameters: json.RawMessage(`{}`),
			}

			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := engine.OptimizeNetworkDeployment(ctx, intent)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkGatherHistoricalData tests data gathering performance
func BenchmarkGatherHistoricalData(b *testing.B) {
	testCases := []struct {
		name       string
		dataPoints int
		parallel   bool
	}{
		{"Sequential-Small", 24, false},
		{"Sequential-Medium", 168, false},
		{"Sequential-Large", 720, false},
		{"Parallel-Small", 24, true},
		{"Parallel-Medium", 168, true},
		{"Parallel-Large", 720, true},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			engine := createBenchmarkEngine(b, tc.dataPoints, 5*time.Millisecond)
			intent := &NetworkIntent{ID: "test-intent"}
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if tc.parallel {
					// Simulate parallel data gathering
					var wg sync.WaitGroup
					wg.Add(3)

					go func() {
						defer wg.Done()
						engine.gatherHistoricalData(ctx, intent)
					}()
					go func() {
						defer wg.Done()
						engine.gatherHistoricalData(ctx, intent)
					}()
					go func() {
						defer wg.Done()
						engine.gatherHistoricalData(ctx, intent)
					}()

					wg.Wait()
				} else {
					engine.gatherHistoricalData(ctx, intent)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage tests memory consumption
func BenchmarkMemoryUsage(b *testing.B) {
	testCases := []struct {
		name       string
		dataPoints int
		intents    int
	}{
		{"Small-Single", 24, 1},
		{"Large-Single", 720, 1},
		{"Small-Multiple", 24, 10},
		{"Large-Multiple", 720, 10},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			engine := createBenchmarkEngine(b, tc.dataPoints, 0)
			intents := make([]*NetworkIntent, tc.intents)
			for i := 0; i < tc.intents; i++ {
				intents[i] = &NetworkIntent{
					ID: fmt.Sprintf("intent-%d", i),
				}
			}

			ctx := context.Background()

			// Force GC before measurement
			runtime.GC()
			var m1 runtime.MemStats
			runtime.ReadMemStats(&m1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, intent := range intents {
					engine.OptimizeNetworkDeployment(ctx, intent)
				}
			}

			// Measure memory after
			runtime.GC()
			var m2 runtime.MemStats
			runtime.ReadMemStats(&m2)

			b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
			b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "allocs/op")
		})
	}
}

// BenchmarkModelPrediction tests ML model prediction performance
func BenchmarkModelPrediction(b *testing.B) {
	models := []struct {
		name      string
		modelType string
	}{
		{"TrafficPrediction", "traffic_prediction"},
		{"ResourceOptimization", "resource_optimization"},
		{"AnomalyDetection", "anomaly_detection"},
		{"IntentClassification", "intent_classification"},
	}

	for _, m := range models {
		b.Run(m.name, func(b *testing.B) {
			engine := createBenchmarkEngine(b, 100, 0)

			// Prepare test data
			testData := make([]DataPoint, 100)
			for i := 0; i < 100; i++ {
				testData[i] = DataPoint{
					Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
					Features: map[string]float64{
						"cpu_utilization":    float64(50 + i%30),
						"memory_utilization": float64(60 + i%20),
						"request_rate":       float64(1000 + i*10),
					},
				}
			}

			// Train the model
			ctx := context.Background()
			if model, exists := engine.models[m.modelType]; exists {
				model.Train(ctx, testData)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if model, exists := engine.models[m.modelType]; exists {
					_, err := model.Predict(ctx, testData)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

// BenchmarkConcurrentOptimization tests concurrent request handling
func BenchmarkConcurrentOptimization(b *testing.B) {
	concurrencyLevels := []int{1, 10, 50, 100}

	for _, level := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrent-%d", level), func(b *testing.B) {
			engine := createBenchmarkEngine(b, 100, 5*time.Millisecond)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				intent := &NetworkIntent{
					ID:          fmt.Sprintf("intent-%d", runtime.NumGoroutine()),
					Description: "Concurrent test",
				}

				for pb.Next() {
					_, err := engine.OptimizeNetworkDeployment(ctx, intent)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// BenchmarkDataPointProcessing tests the efficiency of data point processing
func BenchmarkDataPointProcessing(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%d", size), func(b *testing.B) {
			// Create test data
			dataPoints := make([]DataPoint, size)
			for i := 0; i < size; i++ {
				dataPoints[i] = DataPoint{
					Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
					Features: map[string]float64{
						"cpu":    float64(i % 100),
						"memory": float64(i % 100),
						"disk":   float64(i % 100),
					},
					Metadata: map[string]string{
						"host": fmt.Sprintf("host-%d", i%10),
						"app":  fmt.Sprintf("app-%d", i%5),
					},
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate data processing
				var totalCPU, totalMemory float64
				for _, dp := range dataPoints {
					if cpu, ok := dp.Features["cpu"]; ok {
						totalCPU += cpu
					}
					if mem, ok := dp.Features["memory"]; ok {
						totalMemory += mem
					}
				}
				_ = totalCPU / float64(len(dataPoints))
				_ = totalMemory / float64(len(dataPoints))
			}
		})
	}
}

// BenchmarkConfidenceScoreCalculation tests score calculation performance
func BenchmarkConfidenceScoreCalculation(b *testing.B) {
	engine := createBenchmarkEngine(b, 100, 0)

	recommendations := &OptimizationRecommendations{
		IntentID: "test",
		ResourceAllocation: &ResourceRecommendation{
			CPU:    "2000m",
			Memory: "4Gi",
		},
		ScalingParameters: &ScalingRecommendation{
			MinReplicas:          2,
			MaxReplicas:          10,
			TargetCPUUtilization: 70,
		},
		PerformanceTuning: &PerformanceRecommendation{
			OptimizationProfile: "high-throughput",
		},
		RiskAssessment: &RiskAssessment{
			OverallRisk: "LOW",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.calculateConfidenceScore(recommendations)
	}
}

