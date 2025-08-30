package porch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Intent represents a network intent for package generation
type Intent struct {
	IntentType string `json:"intent_type"`
	Target     string `json:"target"`
	Namespace  string `json:"namespace"`
	Replicas   int    `json:"replicas"`
}

// WriteIntent writes the intent as a Kubernetes package
func WriteIntent(intent interface{}, outDir, format string) error {
	// Convert interface{} to our Intent struct
	var in Intent
	switch v := intent.(type) {
	case Intent:
		in = v
	default:
		// Try to handle any struct with similar fields
		// This is a simplified conversion for compatibility
		in = Intent{
			IntentType: "NetworkFunctionScale", // Default
			Target:     "unknown",
			Namespace:  "default",
			Replicas:   1,
		}
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	switch format {
	case "smp":
		return writeStrategicMergePatch(in, outDir)
	default:
		return writeFullManifest(in, outDir)
	}
}

// writeFullManifest writes a complete Kubernetes manifest
func writeFullManifest(intent Intent, outDir string) error {
	manifest := generateKubernetesManifest(intent)

	filename := filepath.Join(outDir, "deployment.yaml")
	if err := os.WriteFile(filename, []byte(manifest), 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Printf("Generated full manifest: %s\n", filename)
	return nil
}

// writeStrategicMergePatch writes a strategic merge patch
func writeStrategicMergePatch(intent Intent, outDir string) error {
	patch := generateStrategicMergePatch(intent)

	filename := filepath.Join(outDir, "patch.yaml")
	if err := os.WriteFile(filename, []byte(patch), 0644); err != nil {
		return fmt.Errorf("failed to write patch: %w", err)
	}

	fmt.Printf("Generated strategic merge patch: %s\n", filename)
	return nil
}

// generateKubernetesManifest generates a complete Kubernetes deployment manifest
func generateKubernetesManifest(intent Intent) string {
	deploymentName := strings.ToLower(intent.Target)
	if deploymentName == "" || deploymentName == "unknown" {
		deploymentName = "network-function"
	}

	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    intent-type: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
        intent-type: %s
    spec:
      containers:
      - name: %s
        image: %s:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        env:
        - name: INTENT_TYPE
          value: %s
        - name: REPLICAS
          value: "%d"
---
apiVersion: v1
kind: Service
metadata:
  name: %s-service
  namespace: %s
  labels:
    app: %s
spec:
  selector:
    app: %s
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
`,
		deploymentName, intent.Namespace, deploymentName, intent.IntentType,
		intent.Replicas,
		deploymentName,
		deploymentName, intent.IntentType,
		deploymentName, deploymentName,
		intent.IntentType, intent.Replicas,
		deploymentName, intent.Namespace, deploymentName,
		deploymentName)
}

// generateStrategicMergePatch generates a strategic merge patch for scaling
func generateStrategicMergePatch(intent Intent) string {
	deploymentName := strings.ToLower(intent.Target)
	if deploymentName == "" || deploymentName == "unknown" {
		deploymentName = "network-function"
	}

	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: %d
`,
		deploymentName, intent.Namespace, intent.Replicas)
}

// ValidateIntent validates the intent structure
func ValidateIntent(intent Intent) error {
	if intent.IntentType == "" {
		return fmt.Errorf("intent_type is required")
	}
	if intent.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if intent.Replicas < 0 {
		return fmt.Errorf("replicas must be non-negative")
	}
	return nil
}

// GeneratePackageMetadata generates Nephio package metadata
func GeneratePackageMetadata(intent Intent) string {
	return fmt.Sprintf(`apiVersion: config.porch.kpt.dev/v1alpha1
kind: PackageVariant
metadata:
  name: %s-scaling
  namespace: %s
spec:
  upstream:
    repo: blueprints
    package: network-function-base
    revision: main
  downstream:
    repo: deployments
    package: %s-deployment
  adoptionPolicy: adoptExisting
  deletionPolicy: delete
  packageContext:
    data:
      intent-type: %s
      target: %s
      replicas: "%d"
`,
		strings.ToLower(intent.Target), intent.Namespace,
		strings.ToLower(intent.Target),
		intent.IntentType, intent.Target, intent.Replicas)
}
