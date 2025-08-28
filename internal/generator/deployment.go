package generator

import (
	"fmt"

	"github.com/thc1006/nephoran-intent-operator/internal/intent"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

// DeploymentGenerator generates Kubernetes Deployment resources
type DeploymentGenerator struct{}

// NewDeploymentGenerator creates a new deployment generator
func NewDeploymentGenerator() *DeploymentGenerator {
	return &DeploymentGenerator{}
}

// Generate creates a Kubernetes Deployment from a scaling intent
func (g *DeploymentGenerator) Generate(intent *intent.ScalingIntent) ([]byte, error) {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      intent.Target,
			Namespace: intent.Namespace,
			Labels: map[string]string{
				"app":                          intent.Target,
				"app.kubernetes.io/name":       intent.Target,
				"app.kubernetes.io/component":  "nf-simulator",
				"app.kubernetes.io/part-of":    "nephoran-intent-operator",
				"app.kubernetes.io/managed-by": "porch-direct",
			},
			Annotations: map[string]string{
				"nephoran.com/intent-type": intent.IntentType,
				"nephoran.com/source":      intent.Source,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(safeIntToInt32(intent.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": intent.Target,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                          intent.Target,
						"app.kubernetes.io/name":       intent.Target,
						"app.kubernetes.io/component":  "nf-simulator",
						"app.kubernetes.io/part-of":    "nephoran-intent-operator",
						"app.kubernetes.io/managed-by": "porch-direct",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  intent.Target,
							Image: "nephoran/nf-sim:latest",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "metrics",
									ContainerPort: 9090,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "NF_TYPE",
									Value: "simulation",
								},
								{
									Name:  "NF_NAME",
									Value: intent.Target,
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    mustParseQuantity("100m"),
									corev1.ResourceMemory: mustParseQuantity("128Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    mustParseQuantity("50m"),
									corev1.ResourceMemory: mustParseQuantity("64Mi"),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ready",
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			},
		},
	}

	// Add correlation ID if provided
	if intent.CorrelationID != "" {
		deployment.ObjectMeta.Annotations["nephoran.com/correlation-id"] = intent.CorrelationID
	}

	// Add reason if provided
	if intent.Reason != "" {
		deployment.ObjectMeta.Annotations["nephoran.com/reason"] = intent.Reason
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment to YAML: %w", err)
	}

	return yamlData, nil
}

// GenerateService creates a companion Service for the deployment
func (g *DeploymentGenerator) GenerateService(intent *intent.ScalingIntent) ([]byte, error) {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      intent.Target,
			Namespace: intent.Namespace,
			Labels: map[string]string{
				"app":                          intent.Target,
				"app.kubernetes.io/name":       intent.Target,
				"app.kubernetes.io/component":  "nf-simulator",
				"app.kubernetes.io/part-of":    "nephoran-intent-operator",
				"app.kubernetes.io/managed-by": "porch-direct",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": intent.Target,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "metrics",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	yamlData, err := yaml.Marshal(service)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service to YAML: %w", err)
	}

	return yamlData, nil
}

// Helper functions

func int32Ptr(i int32) *int32 {
	return &i
}

// safeIntToInt32 safely converts an int to int32 with bounds checking
func safeIntToInt32(i int) int32 {
	// Check for overflow - int32 max value is 2147483647
	const maxInt32 = int(^uint32(0) >> 1)
	if i > maxInt32 {
		return int32(maxInt32)
	}
	if i < 0 {
		return 0
	}
	return int32(i)
}

// mustParseQuantity parses a resource quantity and panics if it fails
// This is safe for compile-time constants
func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(fmt.Sprintf("invalid quantity %q: %v", s, err))
	}
	return q
}
