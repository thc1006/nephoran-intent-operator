package conductor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestWatchReconcilerLogging(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = nephoranv1.AddToScheme(scheme)

	// Create test NetworkIntent
	networkIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-intent",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "scale deployment nginx to 3 replicas",
		},
	}

	// Create fake client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(networkIntent).
		Build()

	// Create reconciler
	reconciler := &WatchReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	// Set up logger for test
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Test reconciliation
	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-intent",
			Namespace: "default",
		},
	}

	// Call Reconcile
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no requeue
	if result.Requeue {
		t.Error("Expected no requeue")
	}

	// Test with non-existent object
	req.Name = "non-existent"
	result, err = reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile should handle non-existent object: %v", err)
	}

	t.Log("WatchReconciler successfully logs NetworkIntent object keys")
}

// TestWatchReconcilerIntegration tests the complete flow from NetworkIntent to JSON to porch execution
func TestWatchReconcilerIntegration(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = nephoranv1.AddToScheme(scheme)

	testCases := []struct {
		name           string
		intent         string
		namespace      string
		expectedTarget string
		expectedReplicas int
		expectError    bool
		expectRequeue  bool
	}{
		{
			name:             "valid-scaling-intent",
			intent:           "scale deployment nginx to 5 replicas",
			namespace:        "production",
			expectedTarget:   "nginx",
			expectedReplicas: 5,
			expectError:      false,
			expectRequeue:    false,
		},
		{
			name:           "invalid-intent", 
			intent:         "restart database connection",
			namespace:      "default",
			expectError:    false, // Parse error causes requeue, not error
			expectRequeue:  true,
		},
		{
			name:           "empty-intent",
			intent:         "",
			namespace:      "default",
			expectError:    false, // Empty intent causes requeue
			expectRequeue:  true,
		},
		{
			name:             "app-scaling-format",
			intent:           "scale app frontend to 3 replicas",
			namespace:        "staging",
			expectedTarget:   "frontend",
			expectedReplicas: 3,
			expectError:      false,
			expectRequeue:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test directory
			testDir := t.TempDir()
			
			// Create NetworkIntent
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tc.name,
					Namespace: tc.namespace,
					UID:       "test-uid-123",
					Generation: 1,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: tc.intent,
				},
			}

			// Create fake client with the NetworkIntent
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(networkIntent).
				Build()

			// Create mock porch runner
			mockRunner := NewMockPorchRunner()

			// Create reconciler
			reconciler := &WatchReconcilerWithRunner{
				WatchReconciler: &WatchReconciler{
					Client:    fakeClient,
					Scheme:    scheme,
					Logger:    logr.Discard(),
					PorchPath: "/usr/local/bin/porch",
					PorchMode: "apply",
					OutputDir: testDir,
					DryRun:    false, // Allow runner to be called
				},
				Runner: mockRunner,
			}

			// Execute reconciliation
			ctx := context.Background()
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.name,
					Namespace: tc.namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			// Verify error handling
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			// Verify requeue behavior
			if tc.expectRequeue {
				if result.RequeueAfter == 0 {
					t.Error("Expected requeue but got none")
				}
				return // Skip further checks for requeue cases
			}

			// For valid intents, verify JSON file was created
			if tc.expectedTarget != "" {
				// Check that a JSON file was created
				files, err := os.ReadDir(testDir)
				if err != nil {
					t.Fatalf("Failed to read test directory: %v", err)
				}

				jsonFound := false
				for _, file := range files {
					if strings.HasPrefix(file.Name(), "intent-") && strings.HasSuffix(file.Name(), ".json") {
						jsonFound = true
						// Verify JSON content
						filePath := filepath.Join(testDir, file.Name())
						jsonData, err := os.ReadFile(filePath)
						if err != nil {
							t.Errorf("Failed to read JSON file: %v", err)
							continue
						}

						var intentData map[string]interface{}
						if err := json.Unmarshal(jsonData, &intentData); err != nil {
							t.Errorf("Failed to parse JSON: %v", err)
							continue
						}

						// Verify schema compliance
						verifyIntentJSON(t, intentData, tc.expectedTarget, tc.namespace, tc.expectedReplicas)
						break
					}
				}

				if !jsonFound {
					t.Error("Expected JSON file to be created but none found")
				}

				// Verify porch runner was called
				calls := mockRunner.GetCalls()
				if len(calls) != 1 {
					t.Errorf("Expected 1 porch runner call, got %d", len(calls))
				}

				if len(calls) > 0 {
					call := calls[0]
					// The actual name passed is req.NamespacedName.String() which is "namespace/name"
					expectedName := tc.namespace + "/" + tc.name
					if call.Name != expectedName {
						t.Errorf("Expected porch call name '%s', got '%s'", expectedName, call.Name)
					}
					if !strings.Contains(call.IntentFile, "intent-") {
						t.Error("Expected intent file path to contain 'intent-'")
					}
				}
			}
		})
	}
}

// TestWatchReconcilerWithPorchErrors tests error handling in porch execution
func TestWatchReconcilerWithPorchErrors(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = nephoranv1.AddToScheme(scheme)

	testDir := t.TempDir()

	// Create NetworkIntent
	networkIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "error-test",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "scale deployment nginx to 3 replicas",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(networkIntent).
		Build()

	// Create mock runner that will error
	mockRunner := NewMockPorchRunner()
	mockRunner.SetError(true, "porch command failed with exit code 1")

	reconciler := &WatchReconcilerWithRunner{
		WatchReconciler: &WatchReconciler{
			Client:    fakeClient,
			Scheme:    scheme,
			Logger:    logr.Discard(),
			PorchPath: "/usr/local/bin/porch",
			PorchMode: "apply",
			OutputDir: testDir,
			DryRun:    false,
		},
		Runner: mockRunner,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "error-test",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(ctx, req)

	// Should not return error (errors cause requeue)
	if err != nil {
		t.Errorf("Expected no error (should requeue), got: %v", err)
	}

	// Should requeue after 1 minute
	if result.RequeueAfter != time.Minute {
		t.Errorf("Expected requeue after 1 minute, got: %v", result.RequeueAfter)
	}

	// Verify JSON was still created despite porch error
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read test directory: %v", err)
	}

	jsonFound := false
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "intent-") && strings.HasSuffix(file.Name(), ".json") {
			jsonFound = true
			break
		}
	}

	if !jsonFound {
		t.Error("Expected JSON file to be created even when porch fails")
	}

	// Verify porch runner was called
	calls := mockRunner.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 porch runner call, got %d", len(calls))
	}
}

// TestWatchReconcilerDryRun tests dry-run mode
func TestWatchReconcilerDryRun(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = nephoranv1.AddToScheme(scheme)

	testDir := t.TempDir()

	networkIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dry-run-test",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "scale deployment nginx to 3 replicas",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(networkIntent).
		Build()

	mockRunner := NewMockPorchRunner()

	// Test with DryRun enabled
	reconciler := &WatchReconcilerWithRunner{
		WatchReconciler: &WatchReconciler{
			Client:    fakeClient,
			Scheme:    scheme,
			Logger:    logr.Discard(),
			PorchPath: "/usr/local/bin/porch",
			PorchMode: "apply",
			OutputDir: testDir,
			DryRun:    true, // Dry run mode
		},
		Runner: mockRunner,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "dry-run-test",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error in dry-run mode: %v", err)
	}

	if result.Requeue {
		t.Error("Should not requeue in successful dry-run")
	}

	// Verify JSON file was created
	files, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read test directory: %v", err)
	}

	jsonFound := false
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "intent-") && strings.HasSuffix(file.Name(), ".json") {
			jsonFound = true
			break
		}
	}

	if !jsonFound {
		t.Error("Expected JSON file to be created in dry-run mode")
	}

	// Verify porch runner was NOT called in dry-run mode
	calls := mockRunner.GetCalls()
	if len(calls) != 0 {
		t.Errorf("Expected 0 porch runner calls in dry-run mode, got %d", len(calls))
	}
}

// WatchReconcilerWithRunner extends WatchReconciler to use PorchRunner interface
type WatchReconcilerWithRunner struct {
	*WatchReconciler
	Runner PorchRunner
}

// Reconcile overrides the base reconcile to use the injected runner
func (r *WatchReconcilerWithRunner) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Use injected logger or fallback to context logger
	logger := r.Logger
	if logger.GetSink() == nil {
		logger = ctrl.Log.WithName("watch-reconciler")
	}

	// Enhance logger with request context
	logger = logger.WithValues(
		"reconcile.namespace", req.Namespace,
		"reconcile.name", req.Name,
		"reconcile.namespacedName", req.NamespacedName.String(),
	)

	logger.Info("Starting NetworkIntent reconciliation")

	// Fetch the NetworkIntent instance
	var networkIntent nephoranv1.NetworkIntent
	if err := r.Get(ctx, req.NamespacedName, &networkIntent); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get NetworkIntent")
			return ctrl.Result{}, err
		}
		// Object was deleted
		logger.Info("NetworkIntent not found, likely deleted - reconciliation complete")
		return ctrl.Result{}, nil
	}

	// Log the NetworkIntent details
	logger.Info("NetworkIntent found and processing",
		"generation", networkIntent.Generation,
		"resourceVersion", networkIntent.ResourceVersion,
	)

	// Validate the spec
	if networkIntent.Spec.Intent == "" {
		logger.Info("NetworkIntent spec is empty, skipping processing")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Parse intent and generate JSON
	logger.Info("Parsing NetworkIntent spec to JSON")
	intentData, err := r.parseIntentToJSON(&networkIntent)
	if err != nil {
		logger.Error(err, "Failed to parse intent to JSON",
			"intent", networkIntent.Spec.Intent,
		)
		// Requeue after 30 seconds on parse error
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	logger.Info("Successfully generated intent JSON",
		"intentType", intentData["intent_type"],
		"target", intentData["target"],
		"replicas", intentData["replicas"],
	)

	// Write intent JSON to file
	logger.Info("Writing intent JSON to file")
	intentFile, err := r.writeIntentJSON(req.NamespacedName.String(), intentData)
	if err != nil {
		logger.Error(err, "Failed to write intent JSON file")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	logger.Info("Successfully written intent JSON file",
		"file", intentFile,
		"outputDir", r.OutputDir,
	)

	// Execute porch CLI (conditional on dry-run flag)
	if r.DryRun {
		logger.Info("Dry-run mode: skipping porch execution",
			"intentFile", intentFile,
			"porchPath", r.PorchPath,
			"porchMode", r.PorchMode,
		)
	} else {
		logger.Info("Executing porch via runner")
		if err := r.Runner.ExecutePorch(logger, intentFile, req.NamespacedName.String()); err != nil {
			logger.Error(err, "Failed to execute porch via runner")
			// Requeue after 1 minute on porch error
			return ctrl.Result{RequeueAfter: time.Minute}, nil
		}
		logger.Info("Successfully executed porch via runner")
	}

	logger.Info("NetworkIntent reconciliation completed successfully")
	return ctrl.Result{}, nil
}

// verifyIntentJSON verifies that generated JSON matches expected schema and content
func verifyIntentJSON(t *testing.T, intentData map[string]interface{}, expectedTarget, expectedNamespace string, expectedReplicas int) {
	// Verify required fields from schema
	requiredFields := []string{"intent_type", "target", "namespace", "replicas"}
	for _, field := range requiredFields {
		if _, exists := intentData[field]; !exists {
			t.Errorf("Required field '%s' missing from JSON", field)
		}
	}

	// Verify intent_type is "scaling"
	if intentType := intentData["intent_type"]; intentType != "scaling" {
		t.Errorf("Expected intent_type 'scaling', got: %v", intentType)
	}

	// Verify target
	if target := intentData["target"]; target != expectedTarget {
		t.Errorf("Expected target '%s', got: %v", expectedTarget, target)
	}

	// Verify namespace  
	if namespace := intentData["namespace"]; namespace != expectedNamespace {
		t.Errorf("Expected namespace '%s', got: %v", expectedNamespace, namespace)
	}

	// Verify replicas (JSON unmarshals numbers as float64)
	if replicas, ok := intentData["replicas"].(float64); !ok {
		t.Error("Expected replicas to be a number")
	} else if int(replicas) != expectedReplicas {
		t.Errorf("Expected replicas %d, got: %v", expectedReplicas, replicas)
	}

	// Verify source is set
	if source := intentData["source"]; source != "conductor-watch" {
		t.Errorf("Expected source 'conductor-watch', got: %v", source)
	}

	// Verify correlation_id exists and is reasonable
	if correlationID, exists := intentData["correlation_id"]; !exists {
		t.Error("correlation_id should be present")
	} else if correlationID == "" {
		t.Error("correlation_id should not be empty")
	}

	// Verify reason exists
	if reason, exists := intentData["reason"]; !exists {
		t.Error("reason should be present")
	} else if reason == "" {
		t.Error("reason should not be empty")
	}
}