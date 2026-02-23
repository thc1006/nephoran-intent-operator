package loop

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// K8sSubmitFunc creates NetworkIntent CRs directly using Kubernetes API
func K8sSubmitFunc(ctx context.Context, intent *ingest.Intent, mode string) error {
	// Get Kubernetes config (try in-cluster first, then kubeconfig)
	cfg, err := getK8sConfig()
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Define NetworkIntent GVR (GroupVersionResource)
	// Using intent.nephoran.com group as that's the one in config/crd/intent/
	gvr := schema.GroupVersionResource{
		Group:    "intent.nephoran.com",
		Version:  "v1alpha1",
		Resource: "networkintents",
	}

	// Convert intent to NetworkIntent CR
	networkIntent, err := intentToNetworkIntentCR(intent)
	if err != nil {
		return fmt.Errorf("failed to convert intent to NetworkIntent CR: %w", err)
	}

	// Determine target namespace (default to "default" if not specified)
	targetNamespace := intent.Namespace
	if targetNamespace == "" {
		targetNamespace = "default"
	}

	// Create the NetworkIntent CR
	log.Printf("Creating NetworkIntent CR: %s in namespace %s", networkIntent.GetName(), targetNamespace)
	_, err = dynamicClient.Resource(gvr).Namespace(targetNamespace).Create(ctx, networkIntent, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create NetworkIntent CR: %w", err)
	}

	log.Printf("Successfully created NetworkIntent CR: %s/%s", targetNamespace, networkIntent.GetName())
	return nil
}

// getK8sConfig returns Kubernetes config, trying in-cluster first, then kubeconfig
func getK8sConfig() (*rest.Config, error) {
	// Try controller-runtime config helper (handles both in-cluster and kubeconfig)
	cfg, err := config.GetConfig()
	if err != nil {
		// Fallback to explicit kubeconfig loading
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		cfg, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}
	return cfg, nil
}

// intentToNetworkIntentCR converts an ingest.Intent to an unstructured NetworkIntent CR
func intentToNetworkIntentCR(intent *ingest.Intent) (*unstructured.Unstructured, error) {
	// Generate a name for the NetworkIntent
	name := fmt.Sprintf("intent-%s-%s", intent.Target, generateShortID())

	// Build the NetworkIntent CR structure
	obj := map[string]interface{}{
		"apiVersion": "intent.nephoran.com/v1alpha1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name": name,
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by": "nephoran-conductor",
				"nephoran.com/intent-type":     intent.IntentType,
				"nephoran.com/target":          intent.Target,
			},
		},
		"spec": map[string]interface{}{
			"source":     getSource(intent.Source),
			"intentType": intent.IntentType,
			"target":     intent.Target,
			"namespace":  intent.Namespace,
			"replicas":   int32(intent.Replicas),
		},
	}

	// Add optional fields if present
	if intent.Reason != "" {
		metadata := obj["metadata"].(map[string]interface{})
		annotations := make(map[string]interface{})
		annotations["nephoran.com/reason"] = intent.Reason
		metadata["annotations"] = annotations
	}

	if intent.CorrelationID != "" {
		spec := obj["spec"].(map[string]interface{})
		spec["correlationId"] = intent.CorrelationID
	}

	// Convert to Unstructured
	unstructuredObj := &unstructured.Unstructured{}
	unstructuredObj.Object = obj

	return unstructuredObj, nil
}

// getSource returns the source value, defaulting to "conductor-loop" if empty
func getSource(source string) string {
	if source == "" {
		return "conductor-loop"
	}
	return source
}

// generateShortID generates a short unique ID for naming
func generateShortID() string {
	// Simple timestamp-based ID (can be enhanced with UUID if needed)
	return fmt.Sprintf("%d", metav1.Now().Unix()%100000)
}
