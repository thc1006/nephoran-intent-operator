package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type VerifyConfig struct {
	Namespace      string
	DeploymentName string
	TargetReplicas int32
	Timeout        time.Duration
	KubeConfig     string
}

func main() {
	config := &VerifyConfig{}
	
	flag.StringVar(&config.Namespace, "namespace", "", "Target namespace (required)")
	flag.StringVar(&config.DeploymentName, "name", "", "Deployment name (required)")
	flag.StringVar(&config.KubeConfig, "kubeconfig", "", "Path to kubeconfig file")
	
	var targetReplicasStr string
	var timeoutStr string
	
	flag.StringVar(&targetReplicasStr, "target-replicas", "3", "Target replica count")
	flag.StringVar(&timeoutStr, "timeout", "120s", "Verification timeout (e.g., 120s, 5m)")
	
	flag.Parse()

	// Validate required parameters
	if config.Namespace == "" || config.DeploymentName == "" {
		fmt.Fprintf(os.Stderr, "Error: --namespace and --name are required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s --namespace=<ns> --name=<deployment> [--target-replicas=<count>] [--timeout=<duration>]\n", os.Args[0])
		os.Exit(1)
	}

	// Parse target replicas
	targetReplicas, err := strconv.ParseInt(targetReplicasStr, 10, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid target-replicas: %v\n", err)
		os.Exit(1)
	}
	config.TargetReplicas = int32(targetReplicas)

	// Parse timeout
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid timeout duration: %v\n", err)
		os.Exit(1)
	}
	config.Timeout = timeout

	fmt.Printf("🔍 Verifying scaling for deployment %s/%s to %d replicas (timeout: %v)\n", 
		config.Namespace, config.DeploymentName, config.TargetReplicas, config.Timeout)

	// Create Kubernetes client
	clientset, err := createKubernetesClient(config.KubeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Verify scaling
	if err := verifyScaling(clientset, config); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Scaling verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Scaling verification successful: %s/%s scaled to %d replicas\n", 
		config.Namespace, config.DeploymentName, config.TargetReplicas)
}

func createKubernetesClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	// Use provided kubeconfig path, otherwise use default loading rules
	if kubeconfigPath == "" {
		kubeconfigPath = clientcmd.RecommendedHomeFile
	}

	// Build config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, nil
}

func verifyScaling(clientset *kubernetes.Clientset, config *VerifyConfig) error {
	ctx := context.Background()
	
	fmt.Printf("⏳ Polling deployment status every 5s...\n")

	// Use wait.PollImmediate for retry logic
	err := wait.PollImmediate(5*time.Second, config.Timeout, func() (bool, error) {
		deployment, err := clientset.AppsV1().Deployments(config.Namespace).Get(
			ctx, config.DeploymentName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("⚠️  Error getting deployment: %v (retrying...)\n", err)
			return false, nil // Continue polling
		}

		specReplicas := int32(0)
		if deployment.Spec.Replicas != nil {
			specReplicas = *deployment.Spec.Replicas
		}

		readyReplicas := deployment.Status.ReadyReplicas
		availableReplicas := deployment.Status.AvailableReplicas

		fmt.Printf("📊 Current status: spec=%d, ready=%d, available=%d (target=%d)\n",
			specReplicas, readyReplicas, availableReplicas, config.TargetReplicas)

		// Check if scaling is complete
		if specReplicas == config.TargetReplicas && 
		   readyReplicas == config.TargetReplicas && 
		   availableReplicas == config.TargetReplicas {
			fmt.Printf("✨ Target replicas achieved!\n")
			return true, nil
		}

		// Check for error conditions
		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentProgressing && 
			   condition.Status == "False" && 
			   condition.Reason == "ProgressDeadlineExceeded" {
				return false, fmt.Errorf("deployment progress deadline exceeded: %s", condition.Message)
			}
		}

		return false, nil // Continue polling
	})

	if err != nil {
		if err == wait.ErrWaitTimeout {
			// Get final status for error message
			deployment, getErr := clientset.AppsV1().Deployments(config.Namespace).Get(
				ctx, config.DeploymentName, metav1.GetOptions{})
			if getErr == nil {
				specReplicas := int32(0)
				if deployment.Spec.Replicas != nil {
					specReplicas = *deployment.Spec.Replicas
				}
				return fmt.Errorf("timeout waiting for scaling (current: spec=%d, ready=%d, target=%d)",
					specReplicas, deployment.Status.ReadyReplicas, config.TargetReplicas)
			}
			return fmt.Errorf("timeout waiting for scaling after %v", config.Timeout)
		}
		return err
	}

	return nil
}