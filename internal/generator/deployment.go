<<<<<<< HEAD
// Package generator provides utilities for generating Kubernetes deployment resources.
=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
package generator

import (
	"fmt"

<<<<<<< HEAD
=======
	"github.com/thc1006/nephoran-intent-operator/internal/intent"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
<<<<<<< HEAD

	"github.com/thc1006/nephoran-intent-operator/internal/intent"
)

// DeploymentGenerator generates Kubernetes Deployment resources.

type DeploymentGenerator struct{}

// NewDeploymentGenerator creates a new deployment generator.

=======
)

// DeploymentGenerator generates Kubernetes Deployment resources
type DeploymentGenerator struct{}

// NewDeploymentGenerator creates a new deployment generator
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func NewDeploymentGenerator() *DeploymentGenerator {
	return &DeploymentGenerator{}
}

<<<<<<< HEAD
// Generate creates a Kubernetes Deployment from a scaling intent.

=======
// Generate creates a Kubernetes Deployment from a scaling intent
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func (g *DeploymentGenerator) Generate(intent *intent.ScalingIntent) ([]byte, error) {
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
<<<<<<< HEAD

			Kind: "Deployment",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: intent.Target,

			Namespace: intent.Namespace,

			Labels: map[string]string{
				"app": intent.Target,

				"app.kubernetes.io/name": intent.Target,

				"app.kubernetes.io/component": "nf-simulator",

				"app.kubernetes.io/part-of": "nephoran-intent-operator",

				"app.kubernetes.io/managed-by": "porch-direct",
			},

			Annotations: map[string]string{
				"nephoran.com/intent-type": intent.IntentType,

				"nephoran.com/source": intent.Source,
			},
		},

		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(safeIntToInt32(intent.Replicas)),

=======
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
			Replicas: int32Ptr(int32(intent.Replicas)),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": intent.Target,
				},
			},
<<<<<<< HEAD

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": intent.Target,

						"app.kubernetes.io/name": intent.Target,

						"app.kubernetes.io/component": "nf-simulator",

						"app.kubernetes.io/part-of": "nephoran-intent-operator",

						"app.kubernetes.io/managed-by": "porch-direct",
					},
				},

				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: intent.Target,

							Image: "nephoran/nf-sim:latest",

							Ports: []corev1.ContainerPort{
								{
									Name: "http",

									ContainerPort: 8080,

									Protocol: corev1.ProtocolTCP,
								},

								{
									Name: "metrics",

									ContainerPort: 9090,

									Protocol: corev1.ProtocolTCP,
								},
							},

							Env: []corev1.EnvVar{
								{
									Name: "NF_TYPE",

									Value: "simulation",
								},

								{
									Name: "NF_NAME",

									Value: intent.Target,
								},

								{
									Name: "POD_NAME",

=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
<<<<<<< HEAD

								{
									Name: "POD_NAMESPACE",

=======
								{
									Name: "POD_NAMESPACE",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
<<<<<<< HEAD

							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: mustParseQuantity("100m"),

									corev1.ResourceMemory: mustParseQuantity("128Mi"),
								},

								Requests: corev1.ResourceList{
									corev1.ResourceCPU: mustParseQuantity("50m"),

									corev1.ResourceMemory: mustParseQuantity("64Mi"),
								},
							},

=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
<<<<<<< HEAD

										Port: intstr.FromInt(8080),
									},
								},

								InitialDelaySeconds: 30,

								PeriodSeconds: 10,
							},

=======
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ready",
<<<<<<< HEAD

										Port: intstr.FromInt(8080),
									},
								},

								InitialDelaySeconds: 5,

								PeriodSeconds: 5,
							},
						},
					},

=======
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
					RestartPolicy: corev1.RestartPolicyAlways,
				},
			},
		},
	}

<<<<<<< HEAD
	// Add correlation ID if provided.

=======
	// Add correlation ID if provided
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if intent.CorrelationID != "" {
		deployment.ObjectMeta.Annotations["nephoran.com/correlation-id"] = intent.CorrelationID
	}

<<<<<<< HEAD
	// Add reason if provided.

=======
	// Add reason if provided
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	if intent.Reason != "" {
		deployment.ObjectMeta.Annotations["nephoran.com/reason"] = intent.Reason
	}

<<<<<<< HEAD
	// Convert to YAML.

=======
	// Convert to YAML
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	yamlData, err := yaml.Marshal(deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment to YAML: %w", err)
	}

	return yamlData, nil
}

<<<<<<< HEAD
// GenerateService creates a companion Service for the deployment.

=======
// GenerateService creates a companion Service for the deployment
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func (g *DeploymentGenerator) GenerateService(intent *intent.ScalingIntent) ([]byte, error) {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
<<<<<<< HEAD

			Kind: "Service",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name: intent.Target,

			Namespace: intent.Namespace,

			Labels: map[string]string{
				"app": intent.Target,

				"app.kubernetes.io/name": intent.Target,

				"app.kubernetes.io/component": "nf-simulator",

				"app.kubernetes.io/part-of": "nephoran-intent-operator",

				"app.kubernetes.io/managed-by": "porch-direct",
			},
		},

=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": intent.Target,
			},
<<<<<<< HEAD

			Ports: []corev1.ServicePort{
				{
					Name: "http",

					Port: 80,

					TargetPort: intstr.FromInt(8080),

					Protocol: corev1.ProtocolTCP,
				},

				{
					Name: "metrics",

					Port: 9090,

					TargetPort: intstr.FromInt(9090),

					Protocol: corev1.ProtocolTCP,
				},
			},

=======
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	yamlData, err := yaml.Marshal(service)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service to YAML: %w", err)
	}

	return yamlData, nil
}

<<<<<<< HEAD
// Helper functions.
=======
// Helper functions
>>>>>>> 6835433495e87288b95961af7173d866977175ff

func int32Ptr(i int32) *int32 {
	return &i
}

<<<<<<< HEAD
// safeIntToInt32 safely converts an int to int32 with bounds checking.

func safeIntToInt32(i int) int32 {
	// Check for overflow - int32 max value is 2147483647.

	const maxInt32 = int(^uint32(0) >> 1)

	if i > maxInt32 {
		return int32(maxInt32)
	}

	if i < 0 {
		return 0
	}

	return int32(i)
}

// mustParseQuantity parses a resource quantity and panics if it fails.

// This is safe for compile-time constants.

=======
// mustParseQuantity parses a resource quantity and panics if it fails
// This is safe for compile-time constants
>>>>>>> 6835433495e87288b95961af7173d866977175ff
func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(fmt.Sprintf("invalid quantity %q: %v", s, err))
	}
<<<<<<< HEAD

	return q
}
=======
	return q
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
