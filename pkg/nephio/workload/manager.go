package workload

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkloadManager manages Nephio workloads
type WorkloadManager struct {
	client    client.Client
	clientset kubernetes.Interface
	scheme    *runtime.Scheme
	logger    logr.Logger
}

// NewWorkloadManager creates a new workload manager
func NewWorkloadManager(client client.Client, clientset kubernetes.Interface, scheme *runtime.Scheme, logger logr.Logger) *WorkloadManager {
	return &WorkloadManager{
		client:    client,
		clientset: clientset,
		scheme:    scheme,
		logger:    logger,
	}
}

// WorkloadSpec defines the specification for a workload
type WorkloadSpec struct {
	Name      string
	Namespace string
	Image     string
	Replicas  int32
	Labels    map[string]string
	Env       []corev1.EnvVar
}

// CreateWorkload creates a new workload deployment
func (wm *WorkloadManager) CreateWorkload(ctx context.Context, spec *WorkloadSpec) error {
	wm.logger.Info("Creating workload", "name", spec.Name, "namespace", spec.Namespace)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: spec.Namespace,
			Labels:    spec.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: spec.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: spec.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  spec.Name,
							Image: spec.Image,
							Env:   spec.Env,
						},
					},
				},
			},
		},
	}

	if err := wm.client.Create(ctx, deployment); err != nil {
		wm.logger.Error(err, "Failed to create workload deployment", "name", spec.Name)
		return fmt.Errorf("failed to create workload deployment: %w", err)
	}

	wm.logger.Info("Workload created successfully", "name", spec.Name)
	return nil
}

// UpdateWorkload updates an existing workload
func (wm *WorkloadManager) UpdateWorkload(ctx context.Context, spec *WorkloadSpec) error {
	wm.logger.Info("Updating workload", "name", spec.Name, "namespace", spec.Namespace)

	deployment := &appsv1.Deployment{}
	key := client.ObjectKey{
		Name:      spec.Name,
		Namespace: spec.Namespace,
	}

	if err := wm.client.Get(ctx, key, deployment); err != nil {
		wm.logger.Error(err, "Failed to get workload deployment", "name", spec.Name)
		return fmt.Errorf("failed to get workload deployment: %w", err)
	}

	deployment.Spec.Replicas = &spec.Replicas
	deployment.Spec.Template.Spec.Containers[0].Image = spec.Image
	deployment.Spec.Template.Spec.Containers[0].Env = spec.Env

	if err := wm.client.Update(ctx, deployment); err != nil {
		wm.logger.Error(err, "Failed to update workload deployment", "name", spec.Name)
		return fmt.Errorf("failed to update workload deployment: %w", err)
	}

	wm.logger.Info("Workload updated successfully", "name", spec.Name)
	return nil
}

// DeleteWorkload deletes a workload
func (wm *WorkloadManager) DeleteWorkload(ctx context.Context, name, namespace string) error {
	wm.logger.Info("Deleting workload", "name", name, "namespace", namespace)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := wm.client.Delete(ctx, deployment); err != nil {
		wm.logger.Error(err, "Failed to delete workload deployment", "name", name)
		return fmt.Errorf("failed to delete workload deployment: %w", err)
	}

	wm.logger.Info("Workload deleted successfully", "name", name)
	return nil
}

// GetWorkloadStatus gets the status of a workload
func (wm *WorkloadManager) GetWorkloadStatus(ctx context.Context, name, namespace string) (*appsv1.DeploymentStatus, error) {
	deployment := &appsv1.Deployment{}
	key := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}

	if err := wm.client.Get(ctx, key, deployment); err != nil {
		return nil, fmt.Errorf("failed to get workload deployment: %w", err)
	}

	return &deployment.Status, nil
}

// WaitForWorkloadReady waits for a workload to become ready
func (wm *WorkloadManager) WaitForWorkloadReady(ctx context.Context, name, namespace string, timeout time.Duration) error {
	wm.logger.Info("Waiting for workload to be ready", "name", name, "timeout", timeout)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctxWithTimeout.Done():
			return fmt.Errorf("timeout waiting for workload %s to be ready", name)
		default:
			status, err := wm.GetWorkloadStatus(ctxWithTimeout, name, namespace)
			if err != nil {
				return err
			}

			if status.ReadyReplicas == status.Replicas && status.Replicas > 0 {
				wm.logger.Info("Workload is ready", "name", name)
				return nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}
