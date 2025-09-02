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

package multicluster

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	// 	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porchapi/v1alpha1" // DISABLED: external dependency not available
	// 	nephiov1alpha1 "github.com/nephio-project/nephio/api/v1alpha1" // DISABLED: external dependency not available
)

// ChaosScenario defines different types of chaos experiments
type ChaosScenario struct {
	Name         string
	Description  string
	ExecuteFunc  func(t *testing.T, components *MultiClusterComponents)
	ValidateFunc func(t *testing.T, components *MultiClusterComponents)
}

// MultiClusterComponents holds all multi-cluster components for chaos testing
type MultiClusterComponents struct {
	ClusterMgr    *ClusterManager
	Propagator    *PackagePropagator
	SyncEngine    *SyncEngine
	HealthMonitor *HealthMonitor
	Customizer    *Customizer
	TestClusters  map[types.NamespacedName]*ClusterInfo
	AlertHandlers []*MockAlertHandler
}

// MockFailingSyncEngine simulates sync failures
type MockFailingSyncEngine struct {
	*SyncEngine
	failureRate  float64
	latencyRange time.Duration
	mutex        sync.RWMutex
	callCount    int
	failureCount int
}

func NewMockFailingSyncEngine(baseEngine *SyncEngine, failureRate float64, latencyRange time.Duration) *MockFailingSyncEngine {
	return &MockFailingSyncEngine{
		SyncEngine:   baseEngine,
		failureRate:  failureRate,
		latencyRange: latencyRange,
	}
}

func (m *MockFailingSyncEngine) SyncPackageToCluster(
	ctx context.Context,
	packageRevision *porchv1alpha1.PackageRevision,
	targetCluster types.NamespacedName,
) (*nephiov1alpha1.ClusterDeploymentStatus, error) {
	m.mutex.Lock()
	m.callCount++
	callCount := m.callCount
	m.mutex.Unlock()

	// Simulate random latency
	if m.latencyRange > 0 {
		latency := time.Duration(rand.Int63n(int64(m.latencyRange)))
		time.Sleep(latency)
	}

	// Simulate failures based on failure rate
	if rand.Float64() < m.failureRate {
		m.mutex.Lock()
		m.failureCount++
		m.mutex.Unlock()

		return nil, fmt.Errorf("simulated sync failure for cluster %s (call %d)", targetCluster, callCount)
	}

	// Call original method for successful cases
	return m.SyncEngine.SyncPackageToCluster(ctx, packageRevision, targetCluster)
}

func (m *MockFailingSyncEngine) GetStats() (calls int, failures int) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.callCount, m.failureCount
}

// MockUnstableClusterManager simulates cluster instability
type MockUnstableClusterManager struct {
	*ClusterManager
	unstableClusters map[types.NamespacedName]bool
	mutex            sync.RWMutex
}

func NewMockUnstableClusterManager(baseMgr *ClusterManager) *MockUnstableClusterManager {
	return &MockUnstableClusterManager{
		ClusterManager:   baseMgr,
		unstableClusters: make(map[types.NamespacedName]bool),
	}
}

func (m *MockUnstableClusterManager) SetClusterUnstable(clusterName types.NamespacedName, unstable bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.unstableClusters[clusterName] = unstable
}

func (m *MockUnstableClusterManager) SelectTargetClusters(
	ctx context.Context,
	candidates []types.NamespacedName,
	packageRevision *porchv1alpha1.PackageRevision,
) ([]types.NamespacedName, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Filter out unstable clusters
	stableCandidates := make([]types.NamespacedName, 0)
	for _, candidate := range candidates {
		if !m.unstableClusters[candidate] {
			stableCandidates = append(stableCandidates, candidate)
		}
	}

	return m.ClusterManager.SelectTargetClusters(ctx, stableCandidates, packageRevision)
}

// Test setup helper
func setupChaosTestComponents(t *testing.T) *MultiClusterComponents {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	logger := testr.New(t)

	clusterMgr := NewClusterManager(client, logger)
	syncEngine := NewSyncEngine(client, logger)
	customizer := NewCustomizer(client, logger)
	healthMonitor := NewHealthMonitor(client, logger)
	propagator := NewPackagePropagator(client, logger, clusterMgr, syncEngine, customizer)

	// Create test clusters
	testClusters := make(map[types.NamespacedName]*ClusterInfo)
	config := &rest.Config{Host: "https://test-cluster.example.com"}

	for i := 1; i <= 5; i++ {
		clusterName := types.NamespacedName{
			Name:      fmt.Sprintf("chaos-cluster-%d", i),
			Namespace: "default",
		}

		clusterInfo := &ClusterInfo{
			Name:       clusterName,
			Kubeconfig: config,
			ResourceUtilization: ResourceUtilization{
				CPUTotal:    float64(4 + rand.Intn(8)),
				MemoryTotal: int64((8 + rand.Intn(16)) * 1024 * 1024 * 1024),
			},
			HealthStatus: ClusterHealthStatus{Available: true},
		}

		testClusters[clusterName] = clusterInfo
		clusterMgr.clusters[clusterName] = clusterInfo

		healthMonitor.clusters[clusterName] = &ClusterHealthState{
			Name:          clusterName,
			OverallStatus: HealthStatusHealthy,
		}
	}

	// Setup alert handlers
	alertHandlers := make([]*MockAlertHandler, 3)
	for i := range alertHandlers {
		alertHandlers[i] = &MockAlertHandler{}
		healthMonitor.RegisterAlertHandler(alertHandlers[i])
	}

	return &MultiClusterComponents{
		ClusterMgr:    clusterMgr,
		Propagator:    propagator,
		SyncEngine:    syncEngine,
		HealthMonitor: healthMonitor,
		Customizer:    customizer,
		TestClusters:  testClusters,
		AlertHandlers: alertHandlers,
	}
}

// Chaos Scenarios
// DISABLED: func TestChaos_NetworkPartition(t *testing.T) {
	scenario := ChaosScenario{
		Name:        "Network Partition",
		Description: "Simulates network partition affecting subset of clusters",
		ExecuteFunc: func(t *testing.T, components *MultiClusterComponents) {
			// Simulate network partition by making clusters unreachable
			partitionedClusters := []types.NamespacedName{
				{Name: "chaos-cluster-1", Namespace: "default"},
				{Name: "chaos-cluster-2", Namespace: "default"},
			}

			for _, clusterName := range partitionedClusters {
				if cluster, exists := components.TestClusters[clusterName]; exists {
					cluster.HealthStatus = ClusterHealthStatus{
						Available:        false,
						LastFailureCheck: time.Now(),
						FailureReason:    "Network partition detected",
					}
					components.ClusterMgr.clusters[clusterName] = cluster
				}
			}
		},
		ValidateFunc: func(t *testing.T, components *MultiClusterComponents) {
			// Verify that deployment continues to available clusters
			ctx := context.Background()
			packageRevision := createTestPackageRevision("partition-test", "v1.0.0")

			allClusters := make([]types.NamespacedName, 0)
			for clusterName := range components.TestClusters {
				allClusters = append(allClusters, clusterName)
			}

			selectedClusters, err := components.ClusterMgr.SelectTargetClusters(
				ctx, allClusters, packageRevision,
			)

			assert.NoError(t, err)
			assert.True(t, len(selectedClusters) >= 3) // Should still have 3+ available clusters
		},
	}

	t.Run(scenario.Name, func(t *testing.T) {
		components := setupChaosTestComponents(t)
		scenario.ExecuteFunc(t, components)
		scenario.ValidateFunc(t, components)
	})
}

// DISABLED: func TestChaos_HighSyncFailureRate(t *testing.T) {
	scenario := ChaosScenario{
		Name:        "High Sync Failure Rate",
		Description: "Simulates 50% sync failure rate with varying latencies",
		ExecuteFunc: func(t *testing.T, components *MultiClusterComponents) {
			// Replace sync engine with failing one
			failingSyncEngine := NewMockFailingSyncEngine(
				components.SyncEngine,
				0.5,           // 50% failure rate
				2*time.Second, // Up to 2s latency
			)

			// Update propagator with failing sync engine
			components.Propagator.syncEngine = failingSyncEngine
		},
		ValidateFunc: func(t *testing.T, components *MultiClusterComponents) {
			ctx := context.Background()
			packageRevision := createTestPackageRevision("failure-test", "v1.0.0")

			allClusters := make([]types.NamespacedName, 0)
			for clusterName := range components.TestClusters {
				allClusters = append(allClusters, clusterName)
			}

			// Use sequential deployment to observe failures
			deploymentOptions := DeploymentOptions{
				Strategy:          StrategySequential,
				MaxConcurrentDepl: 1,
				Timeout:           30 * time.Second,
				RollbackOnFailure: false,
			}

			_, err := components.Propagator.DeployPackage(
				ctx, packageRevision, allClusters, deploymentOptions,
			)

			// Should fail due to sync failures
			assert.Error(t, err)

			// Verify sync engine recorded failures
			if failingEngine, ok := components.Propagator.syncEngine.(*MockFailingSyncEngine); ok {
				calls, failures := failingEngine.GetStats()
				assert.Greater(t, calls, 0)
				assert.Greater(t, failures, 0)

				// Should have roughly 50% failure rate (with some variance)
				failureRate := float64(failures) / float64(calls)
				assert.True(t, failureRate > 0.3 && failureRate < 0.7,
					"Expected ~50%% failure rate, got %.2f%%", failureRate*100)
			}
		},
	}

	t.Run(scenario.Name, func(t *testing.T) {
		components := setupChaosTestComponents(t)
		scenario.ExecuteFunc(t, components)
		scenario.ValidateFunc(t, components)
	})
}

// DISABLED: func TestChaos_ClusterResourceExhaustion(t *testing.T) {
	scenario := ChaosScenario{
		Name:        "Cluster Resource Exhaustion",
		Description: "Simulates clusters running out of resources during deployment",
		ExecuteFunc: func(t *testing.T, components *MultiClusterComponents) {
			// Exhaust resources on some clusters
			exhaustedClusters := []types.NamespacedName{
				{Name: "chaos-cluster-3", Namespace: "default"},
				{Name: "chaos-cluster-4", Namespace: "default"},
			}

			for _, clusterName := range exhaustedClusters {
				if cluster, exists := components.TestClusters[clusterName]; exists {
					// Set very high resource utilization
					cluster.ResourceUtilization = ResourceUtilization{
						CPUTotal:    4.0,
						CPUUsed:     3.9, // 97.5% utilization
						MemoryTotal: 8 * 1024 * 1024 * 1024,
						MemoryUsed:  7.8 * 1024 * 1024 * 1024, // 97.5% utilization
					}
					components.ClusterMgr.clusters[clusterName] = cluster

					// Update health status
					components.HealthMonitor.clusters[clusterName] = &ClusterHealthState{
						Name:          clusterName,
						OverallStatus: HealthStatusDegraded,
						Alerts: []Alert{
							{
								Severity:  SeverityCritical,
								Type:      AlertTypeResourcePressure,
								Message:   "Critical resource exhaustion",
								Timestamp: time.Now(),
							},
						},
					}
				}
			}
		},
		ValidateFunc: func(t *testing.T, components *MultiClusterComponents) {
			// Start health monitoring to process alerts
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			components.HealthMonitor.StartHealthMonitoring(ctx, 100*time.Millisecond)
			time.Sleep(300 * time.Millisecond)

			// Verify alerts were generated for exhausted clusters
			totalAlerts := 0
			for _, handler := range components.AlertHandlers {
				alerts := handler.GetAlerts()
				for _, alert := range alerts {
					if alert.Type == AlertTypeResourcePressure {
						totalAlerts++
					}
				}
			}

			// Should have alerts about resource pressure
			assert.Greater(t, totalAlerts, 0, "Expected resource pressure alerts")
		},
	}

	t.Run(scenario.Name, func(t *testing.T) {
		components := setupChaosTestComponents(t)
		scenario.ExecuteFunc(t, components)
		scenario.ValidateFunc(t, components)
	})
}

// DISABLED: func TestChaos_RapidClusterFlapping(t *testing.T) {
	scenario := ChaosScenario{
		Name:        "Rapid Cluster Flapping",
		Description: "Simulates clusters rapidly going online/offline",
		ExecuteFunc: func(t *testing.T, components *MultiClusterComponents) {
			unstableMgr := NewMockUnstableClusterManager(components.ClusterMgr)
			components.ClusterMgr = unstableMgr

			// Make clusters unstable in a flapping pattern
			flappingClusters := []types.NamespacedName{
				{Name: "chaos-cluster-1", Namespace: "default"},
				{Name: "chaos-cluster-3", Namespace: "default"},
				{Name: "chaos-cluster-5", Namespace: "default"},
			}

			// Start flapping simulation
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			go func() {
				ticker := time.NewTicker(200 * time.Millisecond)
				defer ticker.Stop()

				flip := false
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						for _, clusterName := range flappingClusters {
							unstableMgr.SetClusterUnstable(clusterName, flip)
						}
						flip = !flip
					}
				}
			}()

			// Let flapping continue during validation
			time.Sleep(1 * time.Second)
		},
		ValidateFunc: func(t *testing.T, components *MultiClusterComponents) {
			ctx := context.Background()
			packageRevision := createTestPackageRevision("flapping-test", "v1.0.0")

			// Attempt multiple deployments during flapping
			var lastSelectedCount int
			consistentSelections := 0

			for i := 0; i < 10; i++ {
				allClusters := make([]types.NamespacedName, 0)
				for clusterName := range components.TestClusters {
					allClusters = append(allClusters, clusterName)
				}

				selectedClusters, err := components.ClusterMgr.SelectTargetClusters(
					ctx, allClusters, packageRevision,
				)

				if err == nil {
					if len(selectedClusters) == lastSelectedCount {
						consistentSelections++
					}
					lastSelectedCount = len(selectedClusters)
				}

				time.Sleep(100 * time.Millisecond)
			}

			// With flapping clusters, we should see some variation in selection
			assert.Less(t, consistentSelections, 8, "Expected some variation due to flapping")
		},
	}

	t.Run(scenario.Name, func(t *testing.T) {
		components := setupChaosTestComponents(t)
		scenario.ExecuteFunc(t, components)
		scenario.ValidateFunc(t, components)
	})
}

// DISABLED: func TestChaos_ConcurrentDeploymentStorm(t *testing.T) {
	scenario := ChaosScenario{
		Name:        "Concurrent Deployment Storm",
		Description: "Simulates many concurrent deployments overwhelming the system",
		ExecuteFunc: func(t *testing.T, components *MultiClusterComponents) {
			// No specific setup needed for this chaos test
		},
		ValidateFunc: func(t *testing.T, components *MultiClusterComponents) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			numConcurrentDeployments := 20
			var wg sync.WaitGroup
			var successCount, errorCount int64
			var mutex sync.Mutex

			for i := 0; i < numConcurrentDeployments; i++ {
				wg.Add(1)
				go func(deploymentIndex int) {
					defer wg.Done()

					packageRevision := createTestPackageRevision(
						fmt.Sprintf("storm-test-%d", deploymentIndex),
						"v1.0.0",
					)

					allClusters := make([]types.NamespacedName, 0)
					for clusterName := range components.TestClusters {
						allClusters = append(allClusters, clusterName)
					}

					deploymentOptions := DeploymentOptions{
						Strategy:          StrategyParallel,
						MaxConcurrentDepl: 2,
						Timeout:           5 * time.Second,
						RollbackOnFailure: false,
					}

					_, err := components.Propagator.DeployPackage(
						ctx, packageRevision, allClusters, deploymentOptions,
					)

					mutex.Lock()
					if err != nil {
						errorCount++
					} else {
						successCount++
					}
					mutex.Unlock()
				}(i)
			}

			wg.Wait()

			mutex.Lock()
			totalAttempts := successCount + errorCount
			mutex.Unlock()

			assert.Equal(t, int64(numConcurrentDeployments), totalAttempts)

			// System should handle concurrent load gracefully
			// Even if some deployments fail, system should not crash
			t.Logf("Deployment storm results: %d successes, %d errors out of %d attempts",
				successCount, errorCount, totalAttempts)
		},
	}

	t.Run(scenario.Name, func(t *testing.T) {
		components := setupChaosTestComponents(t)
		scenario.ExecuteFunc(t, components)
		scenario.ValidateFunc(t, components)
	})
}

// DISABLED: func TestChaos_AlertStorm(t *testing.T) {
	scenario := ChaosScenario{
		Name:        "Alert Storm",
		Description: "Simulates overwhelming number of alerts being generated",
		ExecuteFunc: func(t *testing.T, components *MultiClusterComponents) {
			// Generate alert storm
			alertTypes := []AlertType{
				AlertTypeResourcePressure,
				AlertTypeComponentFailure,
				AlertTypeNetworkIssue,
				AlertTypeSecurityViolation,
				AlertTypePerformanceDegred,
			}

			severities := []AlertSeverity{
				SeverityInfo,
				SeverityCaution,
				SeverityWarning,
				SeverityCritical,
			}

			// Generate large number of alerts
			numAlerts := 1000
			alerts := make([]Alert, numAlerts)

			for i := 0; i < numAlerts; i++ {
				alerts[i] = Alert{
					Severity:     severities[rand.Intn(len(severities))],
					Type:         alertTypes[rand.Intn(len(alertTypes))],
					Message:      fmt.Sprintf("Storm alert #%d", i),
					ResourceName: fmt.Sprintf("resource-%d", rand.Intn(100)),
					Timestamp:    time.Now().Add(time.Duration(rand.Intn(1000)) * time.Millisecond),
				}
			}

			// Process alert storm
			start := time.Now()
			components.HealthMonitor.processAlerts(alerts)
			duration := time.Since(start)

			t.Logf("Processed %d alerts in %v", numAlerts, duration)
		},
		ValidateFunc: func(t *testing.T, components *MultiClusterComponents) {
			// Verify all alert handlers received the alerts
			for i, handler := range components.AlertHandlers {
				alertCount := handler.GetAlertsCount()
				assert.Equal(t, 1000, alertCount,
					"Alert handler %d should have received all 1000 alerts", i)
			}

			// Verify system remains responsive after alert storm
			ctx := context.Background()
			packageRevision := createTestPackageRevision("post-storm-test", "v1.0.0")

			allClusters := make([]types.NamespacedName, 0)
			for clusterName := range components.TestClusters {
				allClusters = append(allClusters, clusterName)
			}

			selectedClusters, err := components.ClusterMgr.SelectTargetClusters(
				ctx, allClusters, packageRevision,
			)

			assert.NoError(t, err)
			assert.Greater(t, len(selectedClusters), 0,
				"System should remain functional after alert storm")
		},
	}

	t.Run(scenario.Name, func(t *testing.T) {
		components := setupChaosTestComponents(t)
		scenario.ExecuteFunc(t, components)
		scenario.ValidateFunc(t, components)
	})
}

// Resilience and Recovery Tests
// DISABLED: func TestResilience_GracefulDegradation(t *testing.T) {
	components := setupChaosTestComponents(t)

	// Progressively make clusters unavailable
	ctx := context.Background()
	packageRevision := createTestPackageRevision("degradation-test", "v1.0.0")

	allClusters := make([]types.NamespacedName, 0)
	for clusterName := range components.TestClusters {
		allClusters = append(allClusters, clusterName)
	}

	// Initial baseline
	selectedClusters, err := components.ClusterMgr.SelectTargetClusters(
		ctx, allClusters, packageRevision,
	)
	assert.NoError(t, err)
	initialCount := len(selectedClusters)

	// Make clusters unavailable one by one
	clusterNames := make([]types.NamespacedName, 0)
	for name := range components.TestClusters {
		clusterNames = append(clusterNames, name)
	}

	for i, clusterName := range clusterNames {
		// Make cluster unavailable
		if cluster, exists := components.TestClusters[clusterName]; exists {
			cluster.HealthStatus.Available = false
			components.ClusterMgr.clusters[clusterName] = cluster
		}

		// Check remaining available clusters
		selectedClusters, err := components.ClusterMgr.SelectTargetClusters(
			ctx, allClusters, packageRevision,
		)

		if i < len(clusterNames)-1 {
			// Should still have some clusters available
			assert.NoError(t, err)
			assert.Equal(t, initialCount-i-1, len(selectedClusters))
		} else {
			// All clusters unavailable - should fail gracefully
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no clusters match")
		}
	}
}

// DISABLED: func TestResilience_RecoveryAfterFailure(t *testing.T) {
	components := setupChaosTestComponents(t)

	// Simulate total system failure
	for clusterName := range components.TestClusters {
		if cluster, exists := components.TestClusters[clusterName]; exists {
			cluster.HealthStatus.Available = false
			cluster.HealthStatus.FailureReason = "Simulated total failure"
			components.ClusterMgr.clusters[clusterName] = cluster
		}
	}

	ctx := context.Background()
	packageRevision := createTestPackageRevision("recovery-test", "v1.0.0")

	allClusters := make([]types.NamespacedName, 0)
	for clusterName := range components.TestClusters {
		allClusters = append(allClusters, clusterName)
	}

	// Verify system is in failed state
	_, err := components.ClusterMgr.SelectTargetClusters(
		ctx, allClusters, packageRevision,
	)
	assert.Error(t, err)

	// Simulate gradual recovery
	recoveryOrder := make([]types.NamespacedName, 0)
	for name := range components.TestClusters {
		recoveryOrder = append(recoveryOrder, name)
	}

	for i, clusterName := range recoveryOrder {
		// Restore cluster
		if cluster, exists := components.TestClusters[clusterName]; exists {
			cluster.HealthStatus.Available = true
			cluster.HealthStatus.FailureReason = ""
			cluster.HealthStatus.LastSuccessCheck = time.Now()
			components.ClusterMgr.clusters[clusterName] = cluster
		}

		// Test functionality recovery
		selectedClusters, err := components.ClusterMgr.SelectTargetClusters(
			ctx, allClusters, packageRevision,
		)

		assert.NoError(t, err)
		assert.Equal(t, i+1, len(selectedClusters))
	}

	// Verify full recovery
	selectedClusters, err := components.ClusterMgr.SelectTargetClusters(
		ctx, allClusters, packageRevision,
	)
	assert.NoError(t, err)
	assert.Equal(t, len(recoveryOrder), len(selectedClusters))
}

// Performance under stress
// DISABLED: func TestStress_HighFrequencyOperations(t *testing.T) {
	components := setupChaosTestComponents(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var operationCount int64
	var wg sync.WaitGroup

	// High frequency cluster selection operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			packageRevision := createTestPackageRevision("stress-test", "v1.0.0")
			allClusters := make([]types.NamespacedName, 0)
			for clusterName := range components.TestClusters {
				allClusters = append(allClusters, clusterName)
			}

			for {
				select {
				case <-ctx.Done():
					return
				default:
					_, err := components.ClusterMgr.SelectTargetClusters(
						ctx, allClusters, packageRevision,
					)
					if err == nil {
						atomic.AddInt64(&operationCount, 1)
					}

					// Small delay to prevent tight loop
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()

	assert.Greater(t, operationCount, int64(100),
		"Should complete significant number of operations under stress")

	t.Logf("Completed %d cluster selection operations under stress", operationCount)
}
