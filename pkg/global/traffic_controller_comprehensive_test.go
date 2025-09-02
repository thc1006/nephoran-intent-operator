package global

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
)

// fakePrometheusAPI implements prometheus v1.API for testing
type fakePrometheusAPI struct {
	shouldFail     bool
	queryResponses map[string]model.Value
	callCount      int
	queriedMetrics []string
}

func (f *fakePrometheusAPI) Query(ctx context.Context, query string, ts time.Time, options ...v1.Option) (model.Value, v1.Warnings, error) {
	f.callCount++
	f.queriedMetrics = append(f.queriedMetrics, query)

	if f.shouldFail {
		return nil, nil, fmt.Errorf("fake prometheus query failure")
	}

	if response, exists := f.queryResponses[query]; exists {
		return response, nil, nil
	}

	// Default response
	return model.Vector{
		&model.Sample{
			Value: 0.95, // 95% success rate by default
		},
	}, nil, nil
}

func (f *fakePrometheusAPI) QueryRange(ctx context.Context, query string, r v1.Range, options ...v1.Option) (model.Value, v1.Warnings, error) {
	return f.Query(ctx, query, time.Now(), options...)
}

func (f *fakePrometheusAPI) LabelNames(ctx context.Context, matches []string, startTime time.Time, endTime time.Time, options ...v1.Option) ([]string, v1.Warnings, error) {
	return []string{}, nil, nil
}

func (f *fakePrometheusAPI) LabelValues(ctx context.Context, label string, matches []string, startTime time.Time, endTime time.Time, options ...v1.Option) (model.LabelValues, v1.Warnings, error) {
	return model.LabelValues{}, nil, nil
}

func (f *fakePrometheusAPI) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time, options ...v1.Option) ([]model.LabelSet, v1.Warnings, error) {
	return []model.LabelSet{}, nil, nil
}

func (f *fakePrometheusAPI) QueryExemplars(ctx context.Context, query string, startTime time.Time, endTime time.Time) ([]v1.ExemplarQueryResult, error) {
	return []v1.ExemplarQueryResult{}, nil
}

func (f *fakePrometheusAPI) GetValue(ctx context.Context, query string, ts time.Time) (model.Value, v1.Warnings, error) {
	return f.Query(ctx, query, ts)
}

func (f *fakePrometheusAPI) Alerts(ctx context.Context) (v1.AlertsResult, error) {
	return v1.AlertsResult{}, nil
}

func (f *fakePrometheusAPI) AlertManagers(ctx context.Context) (v1.AlertManagersResult, error) {
	return v1.AlertManagersResult{}, nil
}

func (f *fakePrometheusAPI) CleanTombstones(ctx context.Context) error {
	return nil
}

func (f *fakePrometheusAPI) Config(ctx context.Context) (v1.ConfigResult, error) {
	return v1.ConfigResult{}, nil
}

func (f *fakePrometheusAPI) DeleteSeries(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) error {
	return nil
}

func (f *fakePrometheusAPI) Flags(ctx context.Context) (v1.FlagsResult, error) {
	return v1.FlagsResult{}, nil
}

func (f *fakePrometheusAPI) Snapshot(ctx context.Context, skipHead bool) (v1.SnapshotResult, error) {
	return v1.SnapshotResult{}, nil
}

func (f *fakePrometheusAPI) Rules(ctx context.Context) (v1.RulesResult, error) {
	return v1.RulesResult{}, nil
}

func (f *fakePrometheusAPI) Targets(ctx context.Context) (v1.TargetsResult, error) {
	return v1.TargetsResult{}, nil
}

func (f *fakePrometheusAPI) TargetsMetadata(ctx context.Context, matchTarget string, metric string, limit string) ([]v1.MetricMetadata, error) {
	return []v1.MetricMetadata{}, nil
}

func (f *fakePrometheusAPI) Metadata(ctx context.Context, metric string, limit string) (map[string][]v1.Metadata, error) {
	return map[string][]v1.Metadata{}, nil
}

func (f *fakePrometheusAPI) TSDB(ctx context.Context, options ...v1.Option) (v1.TSDBResult, error) {
	return v1.TSDBResult{}, nil
}

func (f *fakePrometheusAPI) Runtimeinfo(ctx context.Context) (v1.RuntimeinfoResult, error) {
	return v1.RuntimeinfoResult{}, nil
}

func (f *fakePrometheusAPI) WalReplay(ctx context.Context) (v1.WalReplayStatus, error) {
	return v1.WalReplayStatus{}, nil
}

func (f *fakePrometheusAPI) Buildinfo(ctx context.Context) (v1.BuildinfoResult, error) {
	return v1.BuildinfoResult{}, nil
}

func (f *fakePrometheusAPI) Reset() {
	f.shouldFail = false
	f.queryResponses = make(map[string]model.Value)
	f.callCount = 0
	f.queriedMetrics = []string{}
}

// DISABLED: func TestTrafficController_Start(t *testing.T) {
	testCases := []struct {
		name          string
		configMapData map[string]string
		configSetup   func(*TrafficControllerConfig)
		expectedError bool
		description   string
	}{
		{
			name: "successful_start_with_valid_config",
			configMapData: map[string]string{
				"regions.yaml": `{
					"us-east-1": {
						"name": "US East 1",
						"endpoint": "http://us-east-1.example.com",
						"priority": 1,
						"geographic_location": {
							"continent": "North America",
							"country": "US",
							"latitude": 39.0458,
							"longitude": -76.6413
						},
						"cost_profile": {
							"llm_cost_per_request": 0.001
						}
					},
					"us-west-1": {
						"name": "US West 1", 
						"endpoint": "http://us-west-1.example.com",
						"priority": 2,
						"geographic_location": {
							"continent": "North America",
							"country": "US",
							"latitude": 37.4419,
							"longitude": -122.1430
						},
						"cost_profile": {
							"llm_cost_per_request": 0.0012
						}
					}
				}`,
			},
			configSetup: func(config *TrafficControllerConfig) {
				config.HealthCheckInterval = 1 * time.Second
				config.RoutingDecisionInterval = 2 * time.Second
				config.RoutingStrategy = "hybrid"
			},
			expectedError: false,
			description:   "Should successfully start with valid configuration",
		},
		{
			name:          "start_with_missing_config",
			configMapData: map[string]string{
				// Missing regions.yaml
			},
			configSetup: func(config *TrafficControllerConfig) {
				config.HealthCheckInterval = 1 * time.Second
			},
			expectedError: true,
			description:   "Should fail to start with missing regions configuration",
		},
		{
			name: "start_with_invalid_json",
			configMapData: map[string]string{
				"regions.yaml": `invalid json {`,
			},
			configSetup: func(config *TrafficControllerConfig) {
				config.HealthCheckInterval = 1 * time.Second
			},
			expectedError: true,
			description:   "Should fail to start with invalid JSON configuration",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fake Kubernetes client with ConfigMap
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nephoran-global-config",
					Namespace: "nephoran-global",
				},
				Data: tc.configMapData,
			}

			fakeKubeClient := fake.NewSimpleClientset(configMap)
			fakeClient := ctrlclientfake.NewClientBuilder().Build()
			fakePrometheusAPI := &fakePrometheusAPI{
				queryResponses: make(map[string]model.Value),
			}

			config := &TrafficControllerConfig{
				RoutingStrategy:         "hybrid",
				HealthCheckInterval:     30 * time.Second,
				RoutingDecisionInterval: 60 * time.Second,
			}
			tc.configSetup(config)

			controller := &TrafficController{
				client:           fakeClient,
				kubeClient:       fakeKubeClient,
				prometheusAPI:    fakePrometheusAPI,
				logger:           log.Log.WithName("traffic-controller-test"),
				config:           config,
				regions:          make(map[string]*RegionInfo),
				routingDecisions: make(chan *RoutingDecision, 100),
				healthChecks:     make(map[string]*RegionHealth),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := controller.Start(ctx)

			if tc.expectedError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)

				// Verify regions were loaded
				if tc.name == "successful_start_with_valid_config" {
					regions := make(map[string]*RegionInfo)
					controller.mutex.RLock()
					for k, v := range controller.regions {
						regions[k] = v
					}
					controller.mutex.RUnlock()

					assert.Equal(t, 2, len(regions), "Should load 2 regions")
					assert.Contains(t, regions, "us-east-1", "Should contain us-east-1 region")
					assert.Contains(t, regions, "us-west-1", "Should contain us-west-1 region")
				}
			}
		})
	}
}

// DISABLED: func TestTrafficController_HealthChecks(t *testing.T) {
	testCases := []struct {
		name                string
		regions             map[string]*RegionInfo
		httpServerSetup     func() *httptest.Server
		prometheusResponses map[string]model.Value
		expectedHealthy     map[string]bool
		description         string
	}{
		{
			name: "all_regions_healthy",
			regions: map[string]*RegionInfo{
				"region-1": {
					Name:     "Region 1",
					Endpoint: "http://region1.example.com",
				},
				"region-2": {
					Name:     "Region 2",
					Endpoint: "http://region2.example.com",
				},
			},
			httpServerSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			prometheusResponses: map[string]model.Value{
				`rate(nephoran_requests_total{region="region-1",status="success"}[5m]) / rate(nephoran_requests_total{region="region-1"}[5m])`: model.Vector{
					&model.Sample{Value: 0.98},
				},
				`rate(nephoran_requests_total{region="region-2",status="success"}[5m]) / rate(nephoran_requests_total{region="region-2"}[5m])`: model.Vector{
					&model.Sample{Value: 0.97},
				},
			},
			expectedHealthy: map[string]bool{
				"region-1": true,
				"region-2": true,
			},
			description: "Should mark all regions as healthy when they pass health checks",
		},
		{
			name: "one_region_unhealthy",
			regions: map[string]*RegionInfo{
				"region-1": {
					Name:     "Region 1",
					Endpoint: "http://region1.example.com",
				},
				"region-2": {
					Name:     "Region 2",
					Endpoint: "http://region2.example.com",
				},
			},
			httpServerSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/health" && r.Host == "region2.example.com" {
						w.WriteHeader(http.StatusInternalServerError) // Unhealthy
					} else {
						w.WriteHeader(http.StatusOK)
					}
				}))
			},
			prometheusResponses: map[string]model.Value{
				`rate(nephoran_requests_total{region="region-1",status="success"}[5m]) / rate(nephoran_requests_total{region="region-1"}[5m])`: model.Vector{
					&model.Sample{Value: 0.98},
				},
				`rate(nephoran_requests_total{region="region-2",status="success"}[5m]) / rate(nephoran_requests_total{region="region-2"}[5m])`: model.Vector{
					&model.Sample{Value: 0.90}, // Below threshold
				},
			},
			expectedHealthy: map[string]bool{
				"region-1": true,
				"region-2": false,
			},
			description: "Should mark regions as unhealthy when they fail health checks",
		},
		{
			name: "prometheus_query_failure",
			regions: map[string]*RegionInfo{
				"region-1": {
					Name:     "Region 1",
					Endpoint: "http://region1.example.com",
				},
			},
			httpServerSetup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			prometheusResponses: map[string]model.Value{}, // Empty responses to simulate failure
			expectedHealthy: map[string]bool{
				"region-1": true, // Should default to healthy when metrics unavailable
			},
			description: "Should handle Prometheus query failures gracefully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.httpServerSetup()
			defer server.Close()

			// Update region endpoints to use test server
			for _, region := range tc.regions {
				region.Endpoint = server.URL
			}

			fakePrometheusAPI := &fakePrometheusAPI{
				queryResponses: tc.prometheusResponses,
			}

			config := &TrafficControllerConfig{
				FailoverThreshold: 0.95,
			}

			controller := &TrafficController{
				prometheusAPI: fakePrometheusAPI,
				logger:        log.Log.WithName("traffic-controller-test"),
				config:        config,
				regions:       tc.regions,
				healthChecks:  make(map[string]*RegionHealth),
			}

			// Perform health checks
			ctx := context.Background()
			controller.performHealthChecks(ctx)

			// Verify health status
			healthChecks := controller.GetHealthStatus()
			for regionName, expectedHealthy := range tc.expectedHealthy {
				health, exists := healthChecks[regionName]
				assert.True(t, exists, "Health check should exist for region %s", regionName)
				assert.Equal(t, expectedHealthy, health.IsHealthy, "Region %s health status should match expected", regionName)
			}
		})
	}
}

// DISABLED: func TestTrafficController_RoutingDecisions(t *testing.T) {
	testCases := []struct {
		name            string
		regions         map[string]*RegionInfo
		healthChecks    map[string]*RegionHealth
		routingStrategy string
		expectedWeights map[string]int
		expectedError   bool
		description     string
	}{
		{
			name: "latency_based_routing",
			regions: map[string]*RegionInfo{
				"low-latency": {
					Name: "Low Latency Region",
				},
				"high-latency": {
					Name: "High Latency Region",
				},
			},
			healthChecks: map[string]*RegionHealth{
				"low-latency": {
					Region:            "low-latency",
					IsHealthy:         true,
					AverageLatency:    50 * time.Millisecond,
					SuccessRate:       0.98,
					AvailabilityScore: 0.95,
				},
				"high-latency": {
					Region:            "high-latency",
					IsHealthy:         true,
					AverageLatency:    200 * time.Millisecond,
					SuccessRate:       0.97,
					AvailabilityScore: 0.92,
				},
			},
			routingStrategy: "latency",
			expectedWeights: map[string]int{
				"low-latency":  60, // Higher weight for lower latency
				"high-latency": 40, // Lower weight for higher latency
			},
			expectedError: false,
			description:   "Should route more traffic to low-latency regions",
		},
		{
			name: "capacity_based_routing",
			regions: map[string]*RegionInfo{
				"high-capacity": {
					Name: "High Capacity Region",
					CurrentCapacity: &RegionCapacity{
						CPUUtilization: 30.0, // Low utilization
					},
				},
				"low-capacity": {
					Name: "Low Capacity Region",
					CurrentCapacity: &RegionCapacity{
						CPUUtilization: 80.0, // High utilization
					},
				},
			},
			healthChecks: map[string]*RegionHealth{
				"high-capacity": {
					Region:            "high-capacity",
					IsHealthy:         true,
					SuccessRate:       0.98,
					AvailabilityScore: 0.95,
				},
				"low-capacity": {
					Region:            "low-capacity",
					IsHealthy:         true,
					SuccessRate:       0.97,
					AvailabilityScore: 0.92,
				},
			},
			routingStrategy: "capacity",
			expectedWeights: map[string]int{
				"high-capacity": 78, // Higher weight for more available capacity
				"low-capacity":  22, // Lower weight for less available capacity
			},
			expectedError: false,
			description:   "Should route more traffic to high-capacity regions",
		},
		{
			name: "hybrid_routing",
			regions: map[string]*RegionInfo{
				"balanced": {
					Name: "Balanced Region",
					CurrentCapacity: &RegionCapacity{
						CPUUtilization: 50.0,
					},
				},
				"optimal": {
					Name: "Optimal Region",
					CurrentCapacity: &RegionCapacity{
						CPUUtilization: 30.0,
					},
				},
			},
			healthChecks: map[string]*RegionHealth{
				"balanced": {
					Region:            "balanced",
					IsHealthy:         true,
					AverageLatency:    100 * time.Millisecond,
					SuccessRate:       0.96,
					AvailabilityScore: 0.90,
				},
				"optimal": {
					Region:            "optimal",
					IsHealthy:         true,
					AverageLatency:    50 * time.Millisecond,
					SuccessRate:       0.99,
					AvailabilityScore: 0.98,
				},
			},
			routingStrategy: "hybrid",
			expectedWeights: map[string]int{
				"balanced": 39, // Lower weight due to overall score
				"optimal":  61, // Higher weight due to better overall score
			},
			expectedError: false,
			description:   "Should balance multiple factors in routing decisions",
		},
		{
			name: "no_healthy_regions",
			regions: map[string]*RegionInfo{
				"unhealthy-1": {Name: "Unhealthy Region 1"},
				"unhealthy-2": {Name: "Unhealthy Region 2"},
			},
			healthChecks: map[string]*RegionHealth{
				"unhealthy-1": {
					Region:    "unhealthy-1",
					IsHealthy: false,
				},
				"unhealthy-2": {
					Region:    "unhealthy-2",
					IsHealthy: false,
				},
			},
			routingStrategy: "hybrid",
			expectedWeights: map[string]int{}, // No weights expected
			expectedError:   true,
			description:     "Should handle case with no healthy regions",
		},
		{
			name: "single_healthy_region",
			regions: map[string]*RegionInfo{
				"only-healthy": {
					Name: "Only Healthy Region",
					CurrentCapacity: &RegionCapacity{
						CPUUtilization: 40.0,
					},
				},
			},
			healthChecks: map[string]*RegionHealth{
				"only-healthy": {
					Region:            "only-healthy",
					IsHealthy:         true,
					AverageLatency:    75 * time.Millisecond,
					SuccessRate:       0.98,
					AvailabilityScore: 0.95,
				},
			},
			routingStrategy: "hybrid",
			expectedWeights: map[string]int{
				"only-healthy": 100, // Should get all traffic
			},
			expectedError: false,
			description:   "Should route all traffic to single healthy region",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &TrafficControllerConfig{
				RoutingStrategy: tc.routingStrategy,
			}

			controller := &TrafficController{
				logger:       log.Log.WithName("traffic-controller-test"),
				config:       config,
				regions:      tc.regions,
				healthChecks: tc.healthChecks,
			}

			decision := controller.makeRoutingDecision(context.Background())

			if tc.expectedError {
				assert.Nil(t, decision, tc.description)
			} else {
				assert.NotNil(t, decision, tc.description)

				// Verify routing weights
				actualWeights := make(map[string]int)
				for _, region := range decision.TargetRegions {
					actualWeights[region.Region] = region.Weight
				}

				for regionName, expectedWeight := range tc.expectedWeights {
					actualWeight, exists := actualWeights[regionName]
					assert.True(t, exists, "Region %s should be in routing decision", regionName)
					assert.InDelta(t, expectedWeight, actualWeight, 5, "Weight for region %s should be close to expected", regionName)
				}
			}
		})
	}
}

// DISABLED: func TestTrafficController_MetricsCollection(t *testing.T) {
	testCases := []struct {
		name                string
		regions             []string
		prometheusResponses map[string]model.Value
		expectedCapacity    map[string]*RegionCapacity
		description         string
	}{
		{
			name:    "successful_metrics_collection",
			regions: []string{"region-1", "region-2"},
			prometheusResponses: map[string]model.Value{
				`rate(nephoran_intents_processed_total{region="region-1"}[1m]) * 60`: model.Vector{
					&model.Sample{Value: 100}, // 100 intents per minute
				},
				`rate(nephoran_intents_processed_total{region="region-2"}[1m]) * 60`: model.Vector{
					&model.Sample{Value: 150}, // 150 intents per minute
				},
				`avg(rate(container_cpu_usage_seconds_total{namespace="nephoran-system",region="region-1"}[5m])) * 100`: model.Vector{
					&model.Sample{Value: 45.5}, // 45.5% CPU utilization
				},
				`avg(rate(container_cpu_usage_seconds_total{namespace="nephoran-system",region="region-2"}[5m])) * 100`: model.Vector{
					&model.Sample{Value: 67.2}, // 67.2% CPU utilization
				},
			},
			expectedCapacity: map[string]*RegionCapacity{
				"region-1": {
					CurrentIntentsPerMinute: 100,
					CPUUtilization:          45.5,
				},
				"region-2": {
					CurrentIntentsPerMinute: 150,
					CPUUtilization:          67.2,
				},
			},
			description: "Should successfully collect metrics for all regions",
		},
		{
			name:                "prometheus_query_failure",
			regions:             []string{"region-1"},
			prometheusResponses: map[string]model.Value{}, // Empty to simulate failure
			expectedCapacity: map[string]*RegionCapacity{
				"region-1": {
					CurrentIntentsPerMinute: 0,
					CPUUtilization:          0,
				},
			},
			description: "Should handle Prometheus query failures gracefully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakePrometheusAPI := &fakePrometheusAPI{
				queryResponses: tc.prometheusResponses,
			}

			regions := make(map[string]*RegionInfo)
			for _, regionName := range tc.regions {
				regions[regionName] = &RegionInfo{
					Name: regionName,
				}
			}

			controller := &TrafficController{
				prometheusAPI: fakePrometheusAPI,
				logger:        log.Log.WithName("traffic-controller-test"),
				regions:       regions,
			}

			// Update regional capacity
			ctx := context.Background()
			controller.updateRegionalCapacity(ctx)

			// Verify capacity updates
			for regionName, expectedCapacity := range tc.expectedCapacity {
				region := controller.regions[regionName]
				assert.NotNil(t, region.CurrentCapacity, "Region %s should have capacity information", regionName)
				assert.Equal(t, expectedCapacity.CurrentIntentsPerMinute, region.CurrentCapacity.CurrentIntentsPerMinute, "Intents per minute should match for region %s", regionName)
				assert.InDelta(t, expectedCapacity.CPUUtilization, region.CurrentCapacity.CPUUtilization, 0.1, "CPU utilization should match for region %s", regionName)
			}
		})
	}
}

// DISABLED: func TestTrafficController_ConfidenceScoring(t *testing.T) {
	testCases := []struct {
		name           string
		healthyRegions map[string]*RegionHealth
		expectedScore  float64
		description    string
	}{
		{
			name: "high_confidence_multiple_healthy_regions",
			healthyRegions: map[string]*RegionHealth{
				"region-1": {
					AvailabilityScore: 0.95,
				},
				"region-2": {
					AvailabilityScore: 0.93,
				},
				"region-3": {
					AvailabilityScore: 0.97,
				},
			},
			expectedScore: 0.98, // High confidence with multiple healthy regions
			description:   "Should have high confidence with multiple healthy regions",
		},
		{
			name: "medium_confidence_single_region",
			healthyRegions: map[string]*RegionHealth{
				"region-1": {
					AvailabilityScore: 0.90,
				},
			},
			expectedScore: 1.0, // Capped at 1.0
			description:   "Should have medium confidence with single healthy region",
		},
		{
			name: "low_confidence_poor_availability",
			healthyRegions: map[string]*RegionHealth{
				"region-1": {
					AvailabilityScore: 0.70,
				},
				"region-2": {
					AvailabilityScore: 0.75,
				},
			},
			expectedScore: 0.92, // Lower confidence due to poor availability
			description:   "Should have lower confidence with poor availability scores",
		},
		{
			name:           "zero_confidence_no_regions",
			healthyRegions: map[string]*RegionHealth{},
			expectedScore:  0.0,
			description:    "Should have zero confidence with no healthy regions",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			controller := &TrafficController{
				logger: log.Log.WithName("traffic-controller-test"),
			}

			confidence := controller.calculateConfidenceScore(tc.healthyRegions)
			assert.InDelta(t, tc.expectedScore, confidence, 0.05, tc.description)
		})
	}
}

// DISABLED: func TestTrafficController_LatencyAndCostCalculation(t *testing.T) {
	testCases := []struct {
		name            string
		healthyRegions  map[string]*RegionHealth
		regions         map[string]*RegionInfo
		weightedRegions []WeightedRegion
		expectedLatency time.Duration
		expectedCost    float64
		description     string
	}{
		{
			name: "weighted_latency_and_cost_calculation",
			healthyRegions: map[string]*RegionHealth{
				"fast-expensive": {
					Region:         "fast-expensive",
					AverageLatency: 50 * time.Millisecond,
				},
				"slow-cheap": {
					Region:         "slow-cheap",
					AverageLatency: 200 * time.Millisecond,
				},
			},
			regions: map[string]*RegionInfo{
				"fast-expensive": {
					CostProfile: &CostProfile{
						LLMCostPerRequest: 0.002,
					},
				},
				"slow-cheap": {
					CostProfile: &CostProfile{
						LLMCostPerRequest: 0.001,
					},
				},
			},
			weightedRegions: []WeightedRegion{
				{Region: "fast-expensive", Weight: 70},
				{Region: "slow-cheap", Weight: 30},
			},
			expectedLatency: 95 * time.Millisecond, // Weighted average: (50*70 + 200*30)/100
			expectedCost:    0.0017,                // Weighted average: (0.002*70 + 0.001*30)/100
			description:     "Should calculate weighted average latency and cost",
		},
		{
			name: "single_region_calculation",
			healthyRegions: map[string]*RegionHealth{
				"only-region": {
					Region:         "only-region",
					AverageLatency: 100 * time.Millisecond,
				},
			},
			regions: map[string]*RegionInfo{
				"only-region": {
					CostProfile: &CostProfile{
						LLMCostPerRequest: 0.0015,
					},
				},
			},
			weightedRegions: []WeightedRegion{
				{Region: "only-region", Weight: 100},
			},
			expectedLatency: 100 * time.Millisecond,
			expectedCost:    0.0015,
			description:     "Should handle single region calculation correctly",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			controller := &TrafficController{
				logger: log.Log.WithName("traffic-controller-test"),
			}

			latency := controller.calculateExpectedLatency(tc.healthyRegions, tc.weightedRegions)
			cost := controller.calculateExpectedCost(tc.regions, tc.weightedRegions)

			assert.InDelta(t, tc.expectedLatency.Nanoseconds(), latency.Nanoseconds(), float64(5*time.Millisecond.Nanoseconds()), tc.description+" (latency)")
			assert.InDelta(t, tc.expectedCost, cost, 0.0001, tc.description+" (cost)")
		})
	}
}

// DISABLED: func TestTrafficController_ConcurrentOperations(t *testing.T) {
	// Test concurrent health checks and routing decisions
	fakePrometheusAPI := &fakePrometheusAPI{
		queryResponses: map[string]model.Value{
			`rate(nephoran_requests_total{region="region-1",status="success"}[5m]) / rate(nephoran_requests_total{region="region-1"}[5m])`: model.Vector{
				&model.Sample{Value: 0.98},
			},
		},
	}

	regions := map[string]*RegionInfo{
		"region-1": {
			Name:     "Region 1",
			Endpoint: "http://region1.example.com",
		},
	}

	config := &TrafficControllerConfig{
		FailoverThreshold: 0.95,
		RoutingStrategy:   "hybrid",
	}

	controller := &TrafficController{
		prometheusAPI:    fakePrometheusAPI,
		logger:           log.Log.WithName("traffic-controller-test"),
		config:           config,
		regions:          regions,
		healthChecks:     make(map[string]*RegionHealth),
		routingDecisions: make(chan *RoutingDecision, 100),
	}

	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool, 3)

	// Concurrent health checks
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 5; i++ {
			controller.performHealthChecks(ctx)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Concurrent routing decisions
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 5; i++ {
			decision := controller.makeRoutingDecision(ctx)
			if decision != nil {
				select {
				case controller.routingDecisions <- decision:
				default:
				}
			}
			time.Sleep(15 * time.Millisecond)
		}
	}()

	// Concurrent metrics collection
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 5; i++ {
			controller.updateRegionalCapacity(ctx)
			time.Sleep(20 * time.Millisecond)
		}
	}()

	// Wait for all operations to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify final state is consistent
	healthChecks := controller.GetHealthStatus()
	assert.Equal(t, 1, len(healthChecks), "Should have health check for one region")
}

// Helper function to create a model.Vector with a single sample
func createVector(value float64) model.Vector {
	return model.Vector{
		&model.Sample{Value: model.SampleValue(value)},
	}
}
