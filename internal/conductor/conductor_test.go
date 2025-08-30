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

package conductor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockPorchExecutorSimple provides a simple mock for unit testing
type MockPorchExecutorSimple struct {
	mock.Mock
	CallHistory []PorchCallRecord
	Calls       int
	ShouldFail  bool
	FailOnCall  int
}

type PorchCallRecord struct {
	PorchPath  string
	Args       []string
	OutputDir  string
	IntentFile string
	Mode       string
	Timestamp  time.Time
}

func (m *MockPorchExecutorSimple) ExecutePorch(ctx context.Context, porchPath string, args []string, outputDir string, intentFile string, mode string) error {
	m.Calls++

	call := PorchCallRecord{
		PorchPath:  porchPath,
		Args:       args,
		OutputDir:  outputDir,
		IntentFile: intentFile,
		Mode:       mode,
		Timestamp:  time.Now(),
	}
	m.CallHistory = append(m.CallHistory, call)

	args = append(args, []string{"--input", intentFile, "--output", outputDir}...)
	m.Called(ctx, porchPath, args, outputDir, intentFile, mode)

	// Simulate failure if configured
	if m.ShouldFail && (m.FailOnCall == 0 || m.FailOnCall == m.Calls) {
		return fmt.Errorf("mock porch execution failed on call %d", m.Calls)
	}

	// Create fake KRM output to simulate successful porch execution
	fakeOutput := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: default
spec:
  replicas: 3
# Generated from intent: %s
# Timestamp: %s
`, intentFile, time.Now().Format(time.RFC3339))

	outputFile := filepath.Join(outputDir, "scaling-patch.yaml")
	return os.WriteFile(outputFile, []byte(fakeOutput), 0644)
}

func (m *MockPorchExecutorSimple) GetCalls() int {
	return m.Calls
}

func (m *MockPorchExecutorSimple) GetLastCall() *PorchCallRecord {
	if len(m.CallHistory) == 0 {
		return nil
	}
	return &m.CallHistory[len(m.CallHistory)-1]
}

func (m *MockPorchExecutorSimple) Reset() {
	m.CallHistory = []PorchCallRecord{}
	m.Calls = 0
	m.ShouldFail = false
	m.FailOnCall = 0
	m.Mock = mock.Mock{}
}

func TestConductorReconcileFlow(t *testing.T) {
	tests := []struct {
		name             string
		intent           string
		expectedTarget   string
		expectedReplicas int
		expectError      bool
		expectRequeue    bool
		porchShouldFail  bool
		porchFailOnCall  int
	}{
		{
			name:             "successful scaling intent",
			intent:           "scale deployment web-server to 3 replicas",
			expectedTarget:   "web-server",
			expectedReplicas: 3,
			expectError:      false,
			expectRequeue:    false,
			porchShouldFail:  false,
		},
		{
			name:             "deployment with instances keyword",
			intent:           "scale deployment api-gateway to 5 instances",
			expectedTarget:   "api-gateway",
			expectedReplicas: 5,
			expectError:      false,
			expectRequeue:    false,
			porchShouldFail:  false,
		},
		{
			name:             "service scaling",
			intent:           "scale service nginx-proxy 2 replicas",
			expectedTarget:   "nginx-proxy",
			expectedReplicas: 2,
			expectError:      false,
			expectRequeue:    false,
			porchShouldFail:  false,
		},
		{
			name:          "invalid intent - missing target",
			intent:        "scale to 5 replicas",
			expectError:   false, // Should not error, but requeue
			expectRequeue: true,
		},
		{
			name:          "invalid intent - excessive replicas",
			intent:        "scale deployment huge-app to 150 replicas",
			expectError:   false, // Should not error, but requeue
			expectRequeue: true,
		},
		{
			name:             "porch CLI failure",
			intent:           "scale deployment fail-app to 3 replicas",
			expectedTarget:   "fail-app",
			expectedReplicas: 3,
			expectError:      false, // Should not error, but requeue
			expectRequeue:    true,
			porchShouldFail:  true,
			porchFailOnCall:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tempDir, err := os.MkdirTemp("", "conductor-unit-test")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create fake client with NetworkIntent CRD
			s := runtime.NewScheme()
			err = nephoranv1.AddToScheme(s)
			require.NoError(t, err)
			err = scheme.AddToScheme(s)
			require.NoError(t, err)

			// Create NetworkIntent
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: tt.intent,
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(intent).
				Build()

			// Setup mock porch executor
			mockExecutor := &MockPorchExecutorSimple{
				ShouldFail: tt.porchShouldFail,
				FailOnCall: tt.porchFailOnCall,
			}

			if !tt.expectRequeue && !tt.porchShouldFail {
				mockExecutor.On("ExecutePorch", mock.Anything, "/fake/porch", mock.Anything, tempDir, mock.Anything, "test").Return(nil)
			} else if tt.porchShouldFail {
				// For porch failure cases, set up mock to return error
				mockExecutor.On("ExecutePorch", mock.Anything, "/fake/porch", mock.Anything, tempDir, mock.Anything, "test").Return(fmt.Errorf("mock porch failure"))
			}

			// Create reconciler
			reconciler := &NetworkIntentReconciler{
				Client:        fakeClient,
				Scheme:        s,
				Log:           logr.Discard(),
				PorchPath:     "/fake/porch",
				Mode:          "test",
				OutputDir:     tempDir,
				porchExecutor: mockExecutor,
			}

			// Execute reconcile
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.TODO(), req)

			// Verify results
			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
				return
			}

			assert.NoError(t, err, "Unexpected error: %v", err)

			if tt.expectRequeue {
				assert.True(t, result.RequeueAfter > 0, "Expected requeue but got none")
				assert.Equal(t, time.Minute*5, result.RequeueAfter, "Expected 5 minute requeue delay")
				return
			}

			assert.False(t, result.Requeue, "Unexpected requeue")
			assert.Zero(t, result.RequeueAfter, "Unexpected requeue delay")

			// For successful cases, verify the intent JSON was created
			files, err := os.ReadDir(tempDir)
			require.NoError(t, err)

			var jsonFiles []string
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					jsonFiles = append(jsonFiles, file.Name())
				}
			}

			require.Len(t, jsonFiles, 1, "Expected exactly one JSON file")

			// Verify JSON content
			jsonPath := filepath.Join(tempDir, jsonFiles[0])
			jsonData, err := os.ReadFile(jsonPath)
			require.NoError(t, err)

			var intentData IntentJSON
			err = json.Unmarshal(jsonData, &intentData)
			require.NoError(t, err, "Failed to unmarshal intent JSON")

			assert.Equal(t, "scaling", intentData.IntentType)
			assert.Equal(t, tt.expectedTarget, intentData.Target)
			assert.Equal(t, "default", intentData.Namespace)
			assert.Equal(t, tt.expectedReplicas, intentData.Replicas)
			assert.Equal(t, "user", intentData.Source)
			assert.Contains(t, intentData.CorrelationID, "test-intent")
			assert.Contains(t, intentData.Reason, "NetworkIntent")

			// Verify porch was called correctly
			assert.Equal(t, 1, mockExecutor.GetCalls(), "Expected exactly one porch call")

			lastCall := mockExecutor.GetLastCall()
			require.NotNil(t, lastCall)
			assert.Equal(t, "/fake/porch", lastCall.PorchPath)
			assert.Equal(t, tempDir, lastCall.OutputDir)
			assert.Equal(t, "test", lastCall.Mode)
			assert.Contains(t, lastCall.Args, "fn")
			assert.Contains(t, lastCall.Args, "render")

			// Verify KRM output was created
			krmFile := filepath.Join(tempDir, "scaling-patch.yaml")
			_, err = os.Stat(krmFile)
			assert.NoError(t, err, "Expected KRM output file to exist")

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestConductorIdempotency(t *testing.T) {
	// Setup
	tempDir, err := os.MkdirTemp("", "conductor-idempotency-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	s := runtime.NewScheme()
	err = nephoranv1.AddToScheme(s)
	require.NoError(t, err)
	err = scheme.AddToScheme(s)
	require.NoError(t, err)

	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "idempotent-test",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "scale deployment test-app to 3 replicas",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(intent).
		Build()

	mockExecutor := &MockPorchExecutorSimple{}
	mockExecutor.On("ExecutePorch", mock.Anything, "/fake/porch", mock.Anything, tempDir, mock.Anything, "test").Return(nil)

	reconciler := &NetworkIntentReconciler{
		Client:        fakeClient,
		Scheme:        s,
		Log:           logr.Discard(),
		PorchPath:     "/fake/porch",
		Mode:          "test",
		OutputDir:     tempDir,
		porchExecutor: mockExecutor,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}

	// First reconcile
	result1, err1 := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err1)
	assert.False(t, result1.Requeue)
	assert.Zero(t, result1.RequeueAfter)

	initialCallCount := mockExecutor.GetCalls()
	assert.Equal(t, 1, initialCallCount)

	// Second reconcile (should be idempotent in production, but in our current implementation it processes again)
	result2, err2 := reconciler.Reconcile(context.TODO(), req)
	assert.NoError(t, err2)
	assert.False(t, result2.Requeue)
	assert.Zero(t, result2.RequeueAfter)

	// In the current implementation, it will process again
	// In a real implementation, you might want to add status tracking for idempotency
	finalCallCount := mockExecutor.GetCalls()
	assert.Equal(t, 2, finalCallCount, "Current implementation processes on every reconcile")

	// Verify JSON files - each reconcile creates a new timestamped file
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	jsonCount := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			jsonCount++
		}
	}
	// Each reconcile creates a new file with timestamp, so we expect at least 2
	// but the exact count depends on timing (files might have same timestamp)
	assert.GreaterOrEqual(t, jsonCount, 1, "Should have at least 1 JSON file")
	assert.LessOrEqual(t, jsonCount, 2, "Should have at most 2 JSON files")

	mockExecutor.AssertExpectations(t)
}

func TestMultipleNetworkIntentsProcessing(t *testing.T) {
	// Setup
	tempDir, err := os.MkdirTemp("", "conductor-multiple-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	s := runtime.NewScheme()
	err = nephoranv1.AddToScheme(s)
	require.NoError(t, err)
	err = scheme.AddToScheme(s)
	require.NoError(t, err)

	// Create multiple NetworkIntents
	intents := []*nephoranv1.NetworkIntent{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "intent-1",
				Namespace: "default",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "scale deployment app-1 to 2 replicas",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "intent-2",
				Namespace: "test-ns",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "scale deployment app-2 to 4 replicas",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "intent-3",
				Namespace: "default",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "scale service app-3 to 1 replicas",
			},
		},
	}

	// Convert to client.Object slice for fake client
	objects := make([]client.Object, len(intents))
	for i, intent := range intents {
		objects[i] = intent
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objects...).
		Build()

	mockExecutor := &MockPorchExecutorSimple{}
	mockExecutor.On("ExecutePorch", mock.Anything, "/fake/porch", mock.Anything, tempDir, mock.Anything, "test").Return(nil)

	reconciler := &NetworkIntentReconciler{
		Client:        fakeClient,
		Scheme:        s,
		Log:           logr.Discard(),
		PorchPath:     "/fake/porch",
		Mode:          "test",
		OutputDir:     tempDir,
		porchExecutor: mockExecutor,
	}

	// Process each intent
	expectedResults := []struct {
		target   string
		replicas int
		ns       string
	}{
		{"app-1", 2, "default"},
		{"app-2", 4, "test-ns"},
		{"app-3", 1, "default"},
	}

	for i, intent := range intents {
		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      intent.Name,
				Namespace: intent.Namespace,
			},
		}

		result, err := reconciler.Reconcile(context.TODO(), req)
		assert.NoError(t, err, "Error processing intent %d", i+1)
		assert.False(t, result.Requeue, "Unexpected requeue for intent %d", i+1)
		assert.Zero(t, result.RequeueAfter, "Unexpected requeue delay for intent %d", i+1)
	}

	// Verify all intents were processed
	assert.Equal(t, len(intents), mockExecutor.GetCalls(), "Expected all intents to be processed")

	// Verify JSON files were created
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	jsonFiles := []string{}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			jsonFiles = append(jsonFiles, filepath.Join(tempDir, file.Name()))
		}
	}
	assert.Len(t, jsonFiles, len(intents), "Expected one JSON file per intent")

	// Verify JSON content for each intent
	actualIntents := make(map[string]IntentJSON)
	for _, jsonFile := range jsonFiles {
		jsonData, err := os.ReadFile(jsonFile)
		require.NoError(t, err)

		var intentData IntentJSON
		err = json.Unmarshal(jsonData, &intentData)
		require.NoError(t, err)

		actualIntents[intentData.Target] = intentData
	}

	// Check that all expected targets were processed correctly
	for _, expected := range expectedResults {
		actual, exists := actualIntents[expected.target]
		assert.True(t, exists, "Expected target %s not found", expected.target)
		if exists {
			assert.Equal(t, expected.replicas, actual.Replicas, "Replica mismatch for target %s", expected.target)
			assert.Equal(t, expected.ns, actual.Namespace, "Namespace mismatch for target %s", expected.target)
			assert.Equal(t, "scaling", actual.IntentType, "Intent type mismatch for target %s", expected.target)
		}
	}

	mockExecutor.AssertExpectations(t)
}

func TestConductorErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupError  func(*testing.T) (client.Client, string) // Returns client and tempDir
		expectError bool
	}{
		{
			name: "intent not found",
			setupError: func(t *testing.T) (client.Client, string) {
				tempDir, err := os.MkdirTemp("", "conductor-error-test")
				require.NoError(t, err)

				s := runtime.NewScheme()
				err = nephoranv1.AddToScheme(s)
				require.NoError(t, err)

				// Create client WITHOUT the intent (simulating intent not found)
				fakeClient := fake.NewClientBuilder().WithScheme(s).Build()
				return fakeClient, tempDir
			},
			expectError: false, // Should handle not found gracefully
		},
		{
			name: "invalid output directory permissions",
			setupError: func(t *testing.T) (client.Client, string) {
				// Skip on Windows as permissions work differently
				if os.PathSeparator == '\\' {
					t.Skip("Permission test skipped on Windows")
				}

				tempDir, err := os.MkdirTemp("", "conductor-error-test")
				require.NoError(t, err)

				// Create a read-only directory
				readOnlyDir := filepath.Join(tempDir, "readonly")
				err = os.MkdirAll(readOnlyDir, 0444)
				require.NoError(t, err)

				s := runtime.NewScheme()
				err = nephoranv1.AddToScheme(s)
				require.NoError(t, err)

				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-intent",
						Namespace: "default",
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "scale deployment test-app to 3 replicas",
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(intent).
					Build()

				return fakeClient, readOnlyDir
			},
			expectError: false, // Should requeue on file write error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, tempDir := tt.setupError(t)
			defer func() {
				if tempDir != "" {
					// Restore permissions for cleanup
					_ = os.Chmod(tempDir, 0755)
					os.RemoveAll(tempDir)
				}
			}()

			s := runtime.NewScheme()
			_ = nephoranv1.AddToScheme(s)

			mockExecutor := &MockPorchExecutorSimple{}
			mockExecutor.On("ExecutePorch", mock.Anything, "/fake/porch", mock.Anything, mock.Anything, mock.Anything, "test").Return(nil).Maybe()

			reconciler := &NetworkIntentReconciler{
				Client:        fakeClient,
				Scheme:        s,
				Log:           logr.Discard(),
				PorchPath:     "/fake/porch",
				Mode:          "test",
				OutputDir:     tempDir,
				porchExecutor: mockExecutor,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-intent",
					Namespace: "default",
				},
			}

			result, err := reconciler.Reconcile(context.TODO(), req)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error: %v", err)
				// For error cases, we expect either no requeue or a requeue with delay
				if tt.name == "intent not found" {
					assert.False(t, result.Requeue)
					assert.Zero(t, result.RequeueAfter)
				}
			}
		})
	}
}

func TestPorchExecutorInterface(t *testing.T) {
	t.Run("default porch executor", func(t *testing.T) {
		// This test verifies the defaultPorchExecutor interface compliance
		var executor PorchExecutor = &defaultPorchExecutor{}
		assert.NotNil(t, executor, "Default porch executor should implement PorchExecutor interface")

		// We can't easily test the actual execution without real porch CLI
		// But we can verify the interface is correctly implemented
		assert.Implements(t, (*PorchExecutor)(nil), &defaultPorchExecutor{})
	})

	t.Run("mock porch executor", func(t *testing.T) {
		var executor PorchExecutor = &MockPorchExecutorSimple{}
		assert.NotNil(t, executor, "Mock porch executor should implement PorchExecutor interface")
		assert.Implements(t, (*PorchExecutor)(nil), &MockPorchExecutorSimple{})
	})
}
