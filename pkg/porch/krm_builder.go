package porch

import (
	"fmt"
	"strings"
)

// KRMPackage represents a KRM package structure.
type KRMPackage struct {
	Name      string
	Namespace string
	Content   map[string]string // filename -> content
}

// BuildKRMPackage creates a KRM package from a scaling intent.
func BuildKRMPackage(intent *ScalingIntent, packageName string) (*KRMPackage, error) {
	pkg := &KRMPackage{
		Name:      packageName,
		Namespace: intent.Namespace,
		Content:   make(map[string]string),
	}

	// Create Kptfile.
	kptfile := fmt.Sprintf(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: %s
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: Scaling package for %s
  keywords:
  - scaling
  - %s
`, packageName, intent.Target, intent.Namespace)
	pkg.Content["Kptfile"] = kptfile

	// Create deployment patch for scaling.
	deploymentPatch := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  annotations:
    scaling.nephio.org/intent-type: "%s"
    scaling.nephio.org/replicas: "%d"
`, intent.Target, intent.Namespace, intent.IntentType, intent.Replicas)

	if intent.Reason != "" {
		deploymentPatch += fmt.Sprintf(`    scaling.nephio.org/reason: "%s"
`, intent.Reason)
	}
	if intent.Source != "" {
		deploymentPatch += fmt.Sprintf(`    scaling.nephio.org/source: "%s"
`, intent.Source)
	}
	if intent.CorrelationID != "" {
		deploymentPatch += fmt.Sprintf(`    scaling.nephio.org/correlation-id: "%s"
`, intent.CorrelationID)
	}

	deploymentPatch += fmt.Sprintf(`spec:
  replicas: %d
`, intent.Replicas)

	pkg.Content["deployment-patch.yaml"] = deploymentPatch

	// Create kustomization.yaml for the package.
	kustomization := fmt.Sprintf(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: %s

patchesStrategicMerge:
- deployment-patch.yaml

commonAnnotations:
  managed-by: porch-direct
  intent-processor: nephoran
`, intent.Namespace)

	pkg.Content["kustomization.yaml"] = kustomization

	// Create README for the package.
	readme := fmt.Sprintf(`# Scaling Package: %s

## Intent Details
- Target: %s
- Namespace: %s
- Replicas: %d
- Intent Type: %s

## Package Contents
- Kptfile: Package metadata
- deployment-patch.yaml: Kubernetes deployment patch for scaling
- kustomization.yaml: Kustomize configuration

## Usage
This package is managed by Porch and applies scaling operations to the specified deployment.
`, packageName, intent.Target, intent.Namespace, intent.Replicas, intent.IntentType)

	pkg.Content["README.md"] = readme

	return pkg, nil
}

// GeneratePackagePath creates the package directory path.
func GeneratePackagePath(repoName, packageName, revision string) string {
	if revision == "" {
		revision = "draft"
	}
	// Clean revision to be filesystem-safe.
	revision = strings.ReplaceAll(revision, "/", "-")
	return fmt.Sprintf("examples/packages/%s/%s/%s", repoName, packageName, revision)
}
