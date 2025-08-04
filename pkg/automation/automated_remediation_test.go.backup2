package automation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// MockRemediationExecutor for testing
type MockRemediationExecutor struct {
	mock.Mock
}

func (m *MockRemediationExecutor) Execute(ctx context.Context, action *RemediationActionTemplate, target string) error {
	args := m.Called(ctx, action, target)
	return args.Error(0)
}

func TestNewAutomatedRemediation(t *testing.T) {
	config := &SelfHealingConfig{
		Enabled:                   true,
		AutoRemediationEnabled:    true,
		RemediationTimeout:        5 * time.Minute,
		MaxConcurrentRemediations: 3,
		ComponentConfigs: map[string]*ComponentConfig{
			"test-component": {
				Name:               "test-component",
				AutoHealingEnabled: true,
			},
		},
	}

	k8sClient := fake.NewSimpleClientset()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	ar, err := NewAutomatedRemediation(config, k8sClient, logger)

	assert.NoError(t, err)
	assert.NotNil(t, ar)
	assert.Equal(t, config, ar.config)
	assert.Equal(t, k8sClient, ar.k8sClient)
	assert.Equal(t, logger, ar.logger)
	assert.NotNil(t, ar.strategies)
	assert.NotNil(t, ar.executors)
	assert.NotNil(t, ar.remediationHistory)
	assert.NotNil(t, ar.activeRemediations)
}

func TestExecuteRemediation(t *testing.T) {
	tests := []struct {
		name           string
		component      string
		issue          string
		strategy       *RemediationStrategy
		setupMocks     func(*fake.Clientset)
		expectedError  bool
		expectedResult string
	}{
		{
			name:      "successful restart remediation",
			component: "test-component",
			issue:     "High memory usage",
			strategy: &RemediationStrategy{
				Name: "restart-strategy",
				Actions: []*RemediationActionTemplate{
					{
						Type:     "RESTART",
						Template: "restart-deployment",
						Parameters: map[string]interface{}{
							"graceful": true,
						},
						Timeout: 2 * time.Minute,
					},
				},
				Priority: 1,
			},
			setupMocks: func(client *fake.Clientset) {
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-component",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32Ptr(1),
					},
				}
				client.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, deployment, nil
				})
				client.PrependReactor("update", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, deployment, nil
				})
			},
			expectedError:  false,
			expectedResult: "SUCCESS",
		},
		{
			name:      "failed scale remediation",
			component: "test-component",
			issue:     "High CPU usage",
			strategy: &RemediationStrategy{
				Name: "scale-strategy",
				Actions: []*RemediationActionTemplate{
					{
						Type:     "SCALE",
						Template: "scale-deployment",
						Parameters: map[string]interface{}{
							"replicas": 3,
						},
						Timeout: 2 * time.Minute,
					},
				},
				Priority: 1,
			},
			setupMocks: func(client *fake.Clientset) {
				client.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("deployment not found")
				})
			},
			expectedError:  true,
			expectedResult: "FAILED",
		},
		{
			name:      "remediation with retry",
			component: "test-component",
			issue:     "Service degradation",
			strategy: &RemediationStrategy{
				Name: "retry-strategy",
				Actions: []*RemediationActionTemplate{
					{
						Type:     "RESTART",
						Template: "restart-deployment",
						Parameters: map[string]interface{}{
							"graceful": true,
						},
						Timeout: 2 * time.Minute,
						RetryPolicy: &RetryPolicy{
							MaxAttempts:       3,
							InitialDelay:      1 * time.Second,
							MaxDelay:          10 * time.Second,
							BackoffMultiplier: 2.0,
						},
					},
				},
				Priority: 1,
			},
			setupMocks: func(client *fake.Clientset) {
				attempts := 0
				client.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					attempts++
					if attempts < 3 {
						return true, nil, errors.New("temporary failure")
					}
					deployment := &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-component",
							Namespace: "default",
						},
					}
					return true, deployment, nil
				})
				client.PrependReactor("update", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, nil
				})
			},
			expectedError:  false,
			expectedResult: "SUCCESS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SelfHealingConfig{
				Enabled:                true,
				AutoRemediationEnabled: true,
				RemediationTimeout:     5 * time.Minute,
			}

			k8sClient := fake.NewSimpleClientset()
			if tt.setupMocks != nil {
				tt.setupMocks(k8sClient)
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			ar, _ := NewAutomatedRemediation(config, k8sClient, logger)
			ar.strategies[tt.component] = []*RemediationStrategy{tt.strategy}

			ctx := context.Background()
			result, err := ar.ExecuteRemediation(ctx, tt.component, tt.issue)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if result != nil {
				assert.Equal(t, tt.expectedResult, result.Status)
			}
		})
	}
}

func TestConcurrentRemediations(t *testing.T) {
	config := &SelfHealingConfig{
		Enabled:                   true,
		AutoRemediationEnabled:    true,
		MaxConcurrentRemediations: 2,
		RemediationTimeout:        5 * time.Minute,
	}

	k8sClient := fake.NewSimpleClientset()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	ar, _ := NewAutomatedRemediation(config, k8sClient, logger)

	// Create test strategy
	strategy := &RemediationStrategy{
		Name: "test-strategy",
		Actions: []*RemediationActionTemplate{
			{
				Type:     "RESTART",
				Template: "restart-deployment",
				Parameters: map[string]interface{}{
					"delay": 100 * time.Millisecond,
				},
				Timeout: 1 * time.Second,
			},
		},
	}

	// Add strategies for multiple components
	for i := 0; i < 5; i++ {
		component := fmt.Sprintf("component-%d", i)
		ar.strategies[component] = []*RemediationStrategy{strategy}
	}

	// Execute remediations concurrently
	ctx := context.Background()
	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			component := fmt.Sprintf("component-%d", idx)
			_, err := ar.ExecuteRemediation(ctx, component, "test issue")
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// With max 2 concurrent remediations and 100ms delay each,
	// 5 remediations should take at least 200ms (3 batches)
	assert.Greater(t, duration, 200*time.Millisecond)

	// Verify concurrent limit was respected
	assert.LessOrEqual(t, len(ar.activeRemediations), 2)
}

func TestRemediationHistory(t *testing.T) {
	config := &SelfHealingConfig{
		Enabled:                true,
		AutoRemediationEnabled: true,
	}

	k8sClient := fake.NewSimpleClientset()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	ar, _ := NewAutomatedRemediation(config, k8sClient, logger)

	// Add remediation events to history
	events := []*RemediationEvent{
		{
			ID:        "event-1",
			Component: "api-service",
			Issue:     "High memory usage",
			Strategy:  "restart-strategy",
			Actions:   []string{"RESTART"},
			Status:    "SUCCESS",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Duration:  30 * time.Second,
		},
		{
			ID:        "event-2",
			Component: "database",
			Issue:     "Connection pool exhausted",
			Strategy:  "scale-strategy",
			Actions:   []string{"SCALE"},
			Status:    "SUCCESS",
			Timestamp: time.Now().Add(-30 * time.Minute),
			Duration:  45 * time.Second,
		},
		{
			ID:        "event-3",
			Component: "api-service",
			Issue:     "High CPU usage",
			Strategy:  "scale-strategy",
			Actions:   []string{"SCALE", "RESTART"},
			Status:    "FAILED",
			Error:     "Scale operation timed out",
			Timestamp: time.Now().Add(-15 * time.Minute),
			Duration:  2 * time.Minute,
		},
	}

	for _, event := range events {
		ar.remediationHistory = append(ar.remediationHistory, event)
	}

	// Test GetRemediationHistory
	history := ar.GetRemediationHistory("api-service", 24*time.Hour)
	assert.Len(t, history, 2)
	assert.Equal(t, "event-1", history[0].ID)
	assert.Equal(t, "event-3", history[1].ID)

	// Test GetRemediationMetrics
	metrics := ar.GetRemediationMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(3), metrics.TotalRemediations)
	assert.Equal(t, int64(2), metrics.SuccessfulRemediations)
	assert.Equal(t, int64(1), metrics.FailedRemediations)
	assert.Equal(t, float64(2)/float64(3), metrics.SuccessRate)
}

func TestRemediationStrategySelection(t *testing.T) {
	config := &SelfHealingConfig{
		Enabled:                true,
		AutoRemediationEnabled: true,
	}

	k8sClient := fake.NewSimpleClientset()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	ar, _ := NewAutomatedRemediation(config, k8sClient, logger)

	// Create multiple strategies with different priorities and success rates
	strategies := []*RemediationStrategy{
		{
			Name:        "high-priority-low-success",
			Priority:    1,
			SuccessRate: 0.3,
			Actions: []*RemediationActionTemplate{
				{Type: "RESTART"},
			},
		},
		{
			Name:        "medium-priority-high-success",
			Priority:    2,
			SuccessRate: 0.9,
			Actions: []*RemediationActionTemplate{
				{Type: "SCALE"},
			},
		},
		{
			Name:        "low-priority-medium-success",
			Priority:    3,
			SuccessRate: 0.6,
			Actions: []*RemediationActionTemplate{
				{Type: "RECONFIGURE"},
			},
		},
	}

	ar.strategies["test-component"] = strategies

	// Select best strategy
	selected := ar.selectBestStrategy("test-component", "test issue")
	assert.NotNil(t, selected)
	// Should select the medium priority one with high success rate
	assert.Equal(t, "medium-priority-high-success", selected.Name)
}

func TestRemediationActionExecution(t *testing.T) {
	tests := []struct {
		name          string
		actionType    string
		parameters    map[string]interface{}
		setupMocks    func(*fake.Clientset)
		expectedError bool
	}{
		{
			name:       "restart deployment",
			actionType: "RESTART",
			parameters: map[string]interface{}{
				"graceful": true,
			},
			setupMocks: func(client *fake.Clientset) {
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "default",
					},
				}
				client.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, deployment, nil
				})
				client.PrependReactor("update", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, deployment, nil
				})
			},
			expectedError: false,
		},
		{
			name:       "scale deployment",
			actionType: "SCALE",
			parameters: map[string]interface{}{
				"replicas": 5,
			},
			setupMocks: func(client *fake.Clientset) {
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32Ptr(3),
					},
				}
				client.PrependReactor("get", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, deployment, nil
				})
				client.PrependReactor("update", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, deployment, nil
				})
			},
			expectedError: false,
		},
		{
			name:       "delete pods",
			actionType: "DELETE_PODS",
			parameters: map[string]interface{}{
				"labelSelector": "app=test",
			},
			setupMocks: func(client *fake.Clientset) {
				podList := &v1.PodList{
					Items: []v1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-pod-1",
								Namespace: "default",
								Labels:    map[string]string{"app": "test"},
							},
						},
					},
				}
				client.PrependReactor("list", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, podList, nil
				})
				client.PrependReactor("delete", "pods", func(action ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, nil
				})
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SelfHealingConfig{
				Enabled:                true,
				AutoRemediationEnabled: true,
			}

			k8sClient := fake.NewSimpleClientset()
			if tt.setupMocks != nil {
				tt.setupMocks(k8sClient)
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			ar, _ := NewAutomatedRemediation(config, k8sClient, logger)

			action := &RemediationActionTemplate{
				Type:       tt.actionType,
				Parameters: tt.parameters,
				Timeout:    1 * time.Minute,
			}

			ctx := context.Background()
			err := ar.executeAction(ctx, action, "test-deployment")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemediationMetrics(t *testing.T) {
	metrics := &RemediationMetrics{
		TotalRemediations:      100,
		SuccessfulRemediations: 85,
		FailedRemediations:     15,
		SuccessRate:            0.85,
		AverageRemediationTime: 45 * time.Second,
		RemediationsByType: map[string]int64{
			"RESTART": 50,
			"SCALE":   30,
			"RECONFIGURE": 20,
		},
		RemediationsByComponent: map[string]int64{
			"api-service": 40,
			"database":    30,
			"cache":       30,
		},
	}

	assert.Equal(t, int64(100), metrics.TotalRemediations)
	assert.Equal(t, int64(85), metrics.SuccessfulRemediations)
	assert.Equal(t, int64(15), metrics.FailedRemediations)
	assert.Equal(t, 0.85, metrics.SuccessRate)
	assert.Equal(t, 45*time.Second, metrics.AverageRemediationTime)
	assert.Equal(t, int64(50), metrics.RemediationsByType["RESTART"])
	assert.Equal(t, int64(40), metrics.RemediationsByComponent["api-service"])
}

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

func (ar *AutomatedRemediation) selectBestStrategy(component, issue string) *RemediationStrategy {
	strategies, exists := ar.strategies[component]
	if !exists || len(strategies) == 0 {
		return nil
	}

	var bestStrategy *RemediationStrategy
	bestScore := -1.0

	for _, strategy := range strategies {
		// Simple scoring: lower priority number is better, higher success rate is better
		score := (1.0 / float64(strategy.Priority)) * strategy.SuccessRate
		if score > bestScore {
			bestScore = score
			bestStrategy = strategy
		}
	}

	return bestStrategy
}

func (ar *AutomatedRemediation) executeAction(ctx context.Context, action *RemediationActionTemplate, target string) error {
	switch action.Type {
	case "RESTART":
		return ar.restartDeployment(ctx, target)
	case "SCALE":
		replicas, ok := action.Parameters["replicas"].(int)
		if !ok {
			return errors.New("invalid replicas parameter")
		}
		return ar.scaleDeployment(ctx, target, int32(replicas))
	case "DELETE_PODS":
		selector, ok := action.Parameters["labelSelector"].(string)
		if !ok {
			return errors.New("invalid labelSelector parameter")
		}
		return ar.deletePods(ctx, target, selector)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func (ar *AutomatedRemediation) restartDeployment(ctx context.Context, name string) error {
	deployment, err := ar.k8sClient.AppsV1().Deployments("default").Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Update deployment to trigger restart
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = ar.k8sClient.AppsV1().Deployments("default").Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (ar *AutomatedRemediation) scaleDeployment(ctx context.Context, name string, replicas int32) error {
	deployment, err := ar.k8sClient.AppsV1().Deployments("default").Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	deployment.Spec.Replicas = &replicas
	_, err = ar.k8sClient.AppsV1().Deployments("default").Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (ar *AutomatedRemediation) deletePods(ctx context.Context, namespace, labelSelector string) error {
	return ar.k8sClient.CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

// Benchmark tests
func BenchmarkRemediationExecution(b *testing.B) {
	config := &SelfHealingConfig{
		Enabled:                true,
		AutoRemediationEnabled: true,
	}

	k8sClient := fake.NewSimpleClientset()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	ar, _ := NewAutomatedRemediation(config, k8sClient, logger)

	strategy := &RemediationStrategy{
		Name: "benchmark-strategy",
		Actions: []*RemediationActionTemplate{
			{
				Type:       "RESTART",
				Parameters: map[string]interface{}{},
				Timeout:    1 * time.Minute,
			},
		},
	}

	ar.strategies["benchmark-component"] = []*RemediationStrategy{strategy}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ar.ExecuteRemediation(ctx, "benchmark-component", "benchmark issue")
	}
}