package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
	"nephoran-intent-operator/pkg/git"
)

// ... (constants and Reconciler struct remain the same, but add GitClient)
type NetworkIntentReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	GitClient *git.Client
}

// ... (Reconcile function remains the same, dispatching to handlers)

func (r *NetworkIntentReconciler) handleDeploymentIntent(ctx context.Context, intent *NetworkFunctionDeploymentIntent) (string, error) {
	deployment := r.deploymentForIntent(intent)
	managedElement := r.managedElementForIntent(intent)

	commitMessage := fmt.Sprintf("feat(%s): create network function %s", intent.Namespace, intent.Name)

	modifyFunc := func(repoPath string) error {
		pkgPath := filepath.Join(repoPath, intent.Namespace, intent.Name)
		if err := os.MkdirAll(pkgPath, 0755); err != nil {
			return fmt.Errorf("failed to create package directory: %w", err)
		}

		if err := writeYAML(filepath.Join(pkgPath, "deployment.yaml"), deployment); err != nil {
			return err
		}
		return writeYAML(filepath.Join(pkgPath, "managedelement.yaml"), managedElement)
	}

	return r.GitClient.CommitAndPush(ctx, commitMessage, modifyFunc)
}

func (r *NetworkIntentReconciler) handleScaleIntent(ctx context.Context, intent *NetworkFunctionScaleIntent) (string, error) {
	commitMessage := fmt.Sprintf("feat(%s): scale network function %s to %d replicas", intent.Namespace, intent.Name, intent.Replicas)

	modifyFunc := func(repoPath string) error {
		deploymentPath := filepath.Join(repoPath, intent.Namespace, intent.Name, "deployment.yaml")

		// Read the existing deployment
		data, err := os.ReadFile(deploymentPath)
		if err != nil {
			return fmt.Errorf("failed to read existing deployment.yaml: %w", err)
		}
		var deployment appsv1.Deployment
		if err := yaml.Unmarshal(data, &deployment); err != nil {
			return fmt.Errorf("failed to unmarshal existing deployment.yaml: %w", err)
		}

		// Modify the replicas
		replicas := int32(intent.Replicas)
		deployment.Spec.Replicas = &replicas

		// Write the updated deployment back
		return writeYAML(deploymentPath, &deployment)
	}

	return r.GitClient.CommitAndPush(ctx, commitMessage, modifyFunc)
}

// ... (helper functions like deploymentForIntent, managedElementForIntent)

func writeYAML(path string, obj interface{}) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML for %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0644)
}

// ... (SetupWithManager remains the same)