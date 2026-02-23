package loop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// PorchSubmitFunc is the function signature for submitting intents
type PorchSubmitFunc func(ctx context.Context, intent *ingest.Intent, mode string) error

// K8sSubmitFactory creates a NetworkIntent submission function with a reusable K8s client
// This factory pattern ensures the expensive client creation happens once.
func K8sSubmitFactory() (PorchSubmitFunc, error) {
	// Get Kubernetes config (try in-cluster first, then kubeconfig)
	cfg, err := getK8sConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	// Create dynamic client once for reuse
	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Define NetworkIntent GVR (GroupVersionResource)
	// Using intent.nephoran.com group as that's the one in config/crd/intent/
	gvr := schema.GroupVersionResource{
		Group:    "intent.nephoran.com",
		Version:  "v1alpha1",
		Resource: "networkintents",
	}

	// Return the submission function that reuses the client
	return func(ctx context.Context, intent *ingest.Intent, mode string) error {
		// Early context check - fail fast if context already cancelled
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled before submission: %w", err)
		}

		// Nil check for intent parameter
		if intent == nil {
			return fmt.Errorf("intent cannot be nil")
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
	}, nil
}

// K8sSubmitFunc is the legacy function that creates a new client on each call
// Deprecated: Use K8sSubmitFactory() instead for better performance
func K8sSubmitFunc(ctx context.Context, intent *ingest.Intent, mode string) error {
	// Early context check - fail fast if context already cancelled
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before submission: %w", err)
	}

	// Nil check for intent parameter
	if intent == nil {
		return fmt.Errorf("intent cannot be nil")
	}

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
	// Sanitize target name for DNS-1123 compliance
	sanitizedTarget := sanitizeDNS1123Name(intent.Target)

	// Generate a secure random ID
	randomID, err := generateSecureRandomID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate random ID: %w", err)
	}

	// Generate a name for the NetworkIntent
	name := fmt.Sprintf("intent-%s-%s", sanitizedTarget, randomID)

	// Sanitize label values (must be DNS-1123 compliant)
	sanitizedIntentType := sanitizeLabelValue(intent.IntentType)
	sanitizedTargetLabel := sanitizeLabelValue(intent.Target)

	// Build the NetworkIntent CR structure
	obj := map[string]interface{}{
		"apiVersion": "intent.nephoran.com/v1alpha1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name": name,
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by": "nephoran-conductor",
				"nephoran.com/intent-type":     sanitizedIntentType,
				"nephoran.com/target":          sanitizedTargetLabel,
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

	// Add optional fields as annotations (not spec fields)
	annotations := make(map[string]interface{})
	if intent.Reason != "" {
		annotations["nephoran.com/reason"] = intent.Reason
	}

	// CRITICAL FIX: correlationId should be an annotation, not a spec field
	if intent.CorrelationID != "" {
		annotations["nephoran.com/correlation-id"] = intent.CorrelationID
	}

	// Only add annotations if we have any
	if len(annotations) > 0 {
		metadata := obj["metadata"].(map[string]interface{})
		metadata["annotations"] = annotations
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

// generateSecureRandomID generates a cryptographically secure random ID
// Returns 8 hexadecimal characters (4 bytes of entropy)
func generateSecureRandomID() (string, error) {
	bytes := make([]byte, 4) // 4 bytes = 8 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// sanitizeDNS1123Name sanitizes a string to be DNS-1123 compliant
// DNS-1123 names must:
// - contain only lowercase alphanumeric characters or '-'
// - start with an alphanumeric character
// - end with an alphanumeric character
// - be at most 63 characters
func sanitizeDNS1123Name(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace invalid characters with '-'
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	s = reg.ReplaceAllString(s, "-")

	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")

	// Ensure starts with alphanumeric
	if len(s) > 0 && !isAlphanumeric(s[0]) {
		s = "x" + s
	}

	// Ensure ends with alphanumeric
	if len(s) > 0 && !isAlphanumeric(s[len(s)-1]) {
		s = s + "x"
	}

	// Truncate to 63 characters (leaving room for prefix and suffix)
	if len(s) > 40 { // Leave room for "intent-" prefix and "-XXXXXXXX" suffix
		s = s[:40]
	}

	// Handle empty string case
	if s == "" {
		s = "default"
	}

	return s
}

// sanitizeLabelValue sanitizes a string to be a valid Kubernetes label value
// Label values must:
// - be at most 63 characters
// - contain only alphanumeric, '-', '_', or '.'
// - start and end with alphanumeric character
func sanitizeLabelValue(s string) string {
	if s == "" {
		return ""
	}

	// Replace invalid characters with '-'
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	s = reg.ReplaceAllString(s, "-")

	// Remove leading/trailing non-alphanumeric characters
	s = strings.TrimFunc(s, func(r rune) bool {
		return !isAlphanumeric(byte(r))
	})

	// Truncate to 63 characters
	if len(s) > 63 {
		s = s[:63]
	}

	// Handle empty string after sanitization
	if s == "" {
		s = "unknown"
	}

	return s
}

// isAlphanumeric checks if a byte is alphanumeric
func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
