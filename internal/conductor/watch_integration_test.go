package conductor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestWatchReconcilerJSONGeneration(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = nephoranv1.AddToScheme(scheme)

	// Create test NetworkIntent
	networkIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-scaling",
			Namespace: "production",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "scale deployment web-server to 5 replicas",
		},
	}

	// Create fake client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(networkIntent).
		Build()

	// Create temp output directory
	tempDir := t.TempDir()

	// Create reconciler with mock porch (echo command)
	reconciler := &WatchReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		PorchPath: "echo", // Use echo as a mock porch CLI
		PorchMode: "dry-run",
		OutputDir: tempDir,
	}

	// Set up logger for test
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Test reconciliation
	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-scaling",
			Namespace: "production",
		},
	}

	// Call Reconcile
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no immediate requeue
	if result.Requeue {
		t.Error("Expected no immediate requeue")
	}

	// Verify JSON file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No JSON files created")
	}

	// Find the intent JSON file
	var intentFile string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			intentFile = filepath.Join(tempDir, file.Name())
			break
		}
	}

	if intentFile == "" {
		t.Fatal("No JSON file found")
	}

	// Read and verify JSON content
	data, err := os.ReadFile(intentFile)
	if err != nil {
		t.Fatalf("Failed to read intent JSON: %v", err)
	}

	var intent map[string]interface{}
	if err := json.Unmarshal(data, &intent); err != nil {
		t.Fatalf("Failed to parse intent JSON: %v", err)
	}

	// Verify intent fields
	if intent["intent_type"] != "scaling" {
		t.Errorf("Expected intent_type=scaling, got %v", intent["intent_type"])
	}
	if intent["target"] != "web-server" {
		t.Errorf("Expected target=web-server, got %v", intent["target"])
	}
	if intent["namespace"] != "production" {
		t.Errorf("Expected namespace=production, got %v", intent["namespace"])
	}

	replicas, ok := intent["replicas"].(float64)
	if !ok || int(replicas) != 5 {
		t.Errorf("Expected replicas=5, got %v", intent["replicas"])
	}

	t.Logf("Successfully generated intent JSON: %s", intentFile)
	t.Logf("Intent content: %+v", intent)
}

func TestWatchReconcilerInvalidIntent(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = nephoranv1.AddToScheme(scheme)

	// Create test NetworkIntent with invalid intent
	networkIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-invalid",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "do something random",
		},
	}

	// Create fake client
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(networkIntent).
		Build()

	// Create temp output directory
	tempDir := t.TempDir()

	// Create reconciler
	reconciler := &WatchReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		PorchPath: "echo",
		PorchMode: "dry-run",
		OutputDir: tempDir,
	}

	// Set up logger for test
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Test reconciliation
	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-invalid",
			Namespace: "default",
		},
	}

	// Call Reconcile
	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Should requeue after 30 seconds due to parse error
	if !result.Requeue && result.RequeueAfter == 0 {
		t.Error("Expected requeue after parse error")
	}

	// Verify no JSON file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	if len(files) > 0 {
		t.Error("Should not create JSON file for invalid intent")
	}

	t.Log("Successfully handled invalid intent with requeue")
}
