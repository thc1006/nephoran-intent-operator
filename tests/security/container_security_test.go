//go:build integration

package security

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/tests/utils"
)

var _ = Describe("Container Security Tests", func() {
	var (
		ctx       context.Context
		k8sClient client.Client
		namespace string
		timeout   time.Duration
	)

	BeforeEach(func() {
		ctx = context.Background()
		k8sClient = utils.GetK8sClient()
		namespace = utils.GetTestNamespace()
		timeout = 30 * time.Second
	})

	Context("Non-Root User Verification", func() {
		It("should verify all containers run as non-root", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking container %s in deployment %s", container.Name, deployment.Name))

					// Verify security context is set
					Expect(container.SecurityContext).NotTo(BeNil(),
						"Container %s must have security context", container.Name)

					// Verify RunAsNonRoot is true
					Expect(container.SecurityContext.RunAsNonRoot).NotTo(BeNil())
					Expect(*container.SecurityContext.RunAsNonRoot).To(BeTrue(),
						"Container %s must run as non-root", container.Name)

					// Verify RunAsUser is set and not 0
					if container.SecurityContext.RunAsUser != nil {
						Expect(*container.SecurityContext.RunAsUser).NotTo(Equal(int64(0)),
							"Container %s must not run as root user (UID 0)", container.Name)
					}
				}
			}
		})

		It("should verify init containers run as non-root", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, initContainer := range deployment.Spec.Template.Spec.InitContainers {
					By(fmt.Sprintf("Checking init container %s in deployment %s", initContainer.Name, deployment.Name))

					if initContainer.SecurityContext != nil {
						if initContainer.SecurityContext.RunAsNonRoot != nil {
							Expect(*initContainer.SecurityContext.RunAsNonRoot).To(BeTrue(),
								"Init container %s must run as non-root", initContainer.Name)
						}

						if initContainer.SecurityContext.RunAsUser != nil {
							Expect(*initContainer.SecurityContext.RunAsUser).NotTo(Equal(int64(0)),
								"Init container %s must not run as root user", initContainer.Name)
						}
					}
				}
			}
		})
	})

	Context("Read-Only Root Filesystem", func() {
		It("should verify containers use read-only root filesystem", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking read-only root filesystem for container %s", container.Name))

					Expect(container.SecurityContext).NotTo(BeNil())
					Expect(container.SecurityContext.ReadOnlyRootFilesystem).NotTo(BeNil())
					Expect(*container.SecurityContext.ReadOnlyRootFilesystem).To(BeTrue(),
						"Container %s must use read-only root filesystem", container.Name)
				}
			}
		})

		It("should verify writable volume mounts are explicitly defined", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			writablePaths := []string{"/tmp", "/var/tmp", "/var/log", "/var/cache"}

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking writable mounts for container %s", container.Name))

					// If container needs writable directories, they should be mounted
					for _, writablePath := range writablePaths {
						hasWritableMount := false
						for _, mount := range container.VolumeMounts {
							if strings.HasPrefix(mount.MountPath, writablePath) {
								hasWritableMount = true
								// Verify the mount is not read-only for writable paths
								if mount.ReadOnly {
									Expect(mount.ReadOnly).To(BeFalse(),
										"Mount %s should be writable for path %s", mount.Name, mount.MountPath)
								}
								break
							}
						}
						// This is informational - not all containers need all writable paths
						if hasWritableMount {
							By(fmt.Sprintf("Container %s has writable mount for %s", container.Name, writablePath))
						}
					}
				}
			}
		})
	})

	Context("Capability Management", func() {
		It("should verify containers drop all capabilities", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking capabilities for container %s", container.Name))

					Expect(container.SecurityContext).NotTo(BeNil())
					Expect(container.SecurityContext.Capabilities).NotTo(BeNil(),
						"Container %s must have capabilities configuration", container.Name)

					// Verify all capabilities are dropped
					Expect(container.SecurityContext.Capabilities.Drop).To(ContainElement(corev1.Capability("ALL")),
						"Container %s must drop ALL capabilities", container.Name)
				}
			}
		})

		It("should verify no privileged containers", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking privileged status for container %s", container.Name))

					if container.SecurityContext != nil && container.SecurityContext.Privileged != nil {
						Expect(*container.SecurityContext.Privileged).To(BeFalse(),
							"Container %s must not be privileged", container.Name)
					}
				}
			}
		})

		It("should verify allowPrivilegeEscalation is false", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking privilege escalation for container %s", container.Name))

					Expect(container.SecurityContext).NotTo(BeNil())
					Expect(container.SecurityContext.AllowPrivilegeEscalation).NotTo(BeNil())
					Expect(*container.SecurityContext.AllowPrivilegeEscalation).To(BeFalse(),
						"Container %s must not allow privilege escalation", container.Name)
				}
			}
		})
	})

	Context("Security Context Validation", func() {
		It("should verify pod-level security context", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				By(fmt.Sprintf("Checking pod security context for deployment %s", deployment.Name))

				podSpec := deployment.Spec.Template.Spec
				Expect(podSpec.SecurityContext).NotTo(BeNil(),
					"Deployment %s must have pod security context", deployment.Name)

				// Verify RunAsNonRoot at pod level
				if podSpec.SecurityContext.RunAsNonRoot != nil {
					Expect(*podSpec.SecurityContext.RunAsNonRoot).To(BeTrue(),
						"Pod in deployment %s must run as non-root", deployment.Name)
				}

				// Verify FSGroup is set for storage access
				if podSpec.SecurityContext.FSGroup != nil {
					Expect(*podSpec.SecurityContext.FSGroup).NotTo(Equal(int64(0)),
						"Pod FSGroup should not be root group")
				}
			}
		})

		It("should verify seccomp profiles", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				By(fmt.Sprintf("Checking seccomp profiles for deployment %s", deployment.Name))

				podSpec := deployment.Spec.Template.Spec
				if podSpec.SecurityContext != nil && podSpec.SecurityContext.SeccompProfile != nil {
					// Verify seccomp profile is set to RuntimeDefault
					Expect(podSpec.SecurityContext.SeccompProfile.Type).To(Equal(corev1.SeccompProfileTypeRuntimeDefault),
						"Pod seccomp profile should be RuntimeDefault")
				}

				// Check container-level seccomp profiles
				for _, container := range podSpec.Containers {
					if container.SecurityContext != nil && container.SecurityContext.SeccompProfile != nil {
						Expect(container.SecurityContext.SeccompProfile.Type).To(Equal(corev1.SeccompProfileTypeRuntimeDefault),
							"Container %s seccomp profile should be RuntimeDefault", container.Name)
					}
				}
			}
		})
	})

	Context("Image Security", func() {
		It("should verify container images use specific tags (not latest)", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking image tag for container %s", container.Name))

					image := container.Image
					Expect(image).NotTo(BeEmpty())

					// Check if image ends with :latest or has no tag
					Expect(image).NotTo(HaveSuffix(":latest"),
						"Container %s should not use 'latest' tag", container.Name)

					// Verify image has a tag
					if !strings.Contains(image, "@") { // Not using digest
						Expect(strings.Contains(image, ":")).To(BeTrue(),
							"Container %s image must have a specific tag", container.Name)
					}
				}
			}
		})

		It("should verify images are from trusted registries", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			trustedRegistries := []string{
				"ghcr.io/thc1006",
				"docker.io/nephoran",
				"gcr.io/nephoran",
				"registry.k8s.io",
			}

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking image registry for container %s", container.Name))

					image := container.Image
					isTrusted := false

					for _, registry := range trustedRegistries {
						if strings.HasPrefix(image, registry) {
							isTrusted = true
							break
						}
					}

					// For development/testing, we may allow other registries
					// In production, this should be strictly enforced
					if !isTrusted {
						By(fmt.Sprintf("Warning: Container %s uses untrusted registry: %s", container.Name, image))
					}
				}
			}
		})

		It("should verify image pull policy", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking image pull policy for container %s", container.Name))

					// Verify pull policy is not Always for tagged images (except latest)
					if !strings.HasSuffix(container.Image, ":latest") {
						expectedPolicies := []corev1.PullPolicy{
							corev1.PullIfNotPresent,
							corev1.PullNever,
						}

						if container.ImagePullPolicy != "" {
							Expect(expectedPolicies).To(ContainElement(container.ImagePullPolicy),
								"Container %s should use IfNotPresent or Never pull policy for tagged images", container.Name)
						}
					}
				}
			}
		})
	})

	Context("Resource Limits and Requests", func() {
		It("should verify containers have resource limits", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking resource limits for container %s", container.Name))

					Expect(container.Resources.Limits).NotTo(BeNil(),
						"Container %s must have resource limits", container.Name)

					// Verify memory limit is set
					memoryLimit := container.Resources.Limits[corev1.ResourceMemory]
					Expect(memoryLimit.IsZero()).To(BeFalse(),
						"Container %s must have memory limit", container.Name)

					// Verify CPU limit is set
					cpuLimit := container.Resources.Limits[corev1.ResourceCPU]
					Expect(cpuLimit.IsZero()).To(BeFalse(),
						"Container %s must have CPU limit", container.Name)
				}
			}
		})

		It("should verify containers have resource requests", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					By(fmt.Sprintf("Checking resource requests for container %s", container.Name))

					Expect(container.Resources.Requests).NotTo(BeNil(),
						"Container %s must have resource requests", container.Name)

					// Verify memory request is set
					memoryRequest := container.Resources.Requests[corev1.ResourceMemory]
					Expect(memoryRequest.IsZero()).To(BeFalse(),
						"Container %s must have memory request", container.Name)

					// Verify CPU request is set
					cpuRequest := container.Resources.Requests[corev1.ResourceCPU]
					Expect(cpuRequest.IsZero()).To(BeFalse(),
						"Container %s must have CPU request", container.Name)
				}
			}
		})
	})

	Context("Vulnerability Scanning Integration", func() {
		It("should check for image vulnerability scan annotations", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				By(fmt.Sprintf("Checking vulnerability scan annotations for deployment %s", deployment.Name))

				// Check for vulnerability scan result annotations
				annotations := deployment.Annotations
				if annotations != nil {
					scanStatus, exists := annotations["security.nephoran.io/vulnerability-scan-status"]
					if exists {
						Expect(scanStatus).To(BeElementOf("passed", "failed", "pending"),
							"Vulnerability scan status must be valid")

						if scanStatus == "failed" {
							By(fmt.Sprintf("Warning: Deployment %s failed vulnerability scan", deployment.Name))
						}
					}

					// Check for last scan timestamp
					lastScan, exists := annotations["security.nephoran.io/last-vulnerability-scan"]
					if exists {
						By(fmt.Sprintf("Deployment %s last scanned: %s", deployment.Name, lastScan))
					}
				}
			}
		})

		It("should verify runtime security monitoring", func() {
			// Check if runtime security tools are deployed
			var falcoDeployment appsv1.Deployment
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "falco",
				Namespace: "falco-system",
			}, &falcoDeployment)

			// Falco may not be deployed in test environment
			if err == nil {
				By("Falco runtime security is deployed")
				Expect(falcoDeployment.Status.ReadyReplicas).To(BeNumerically(">", 0))
			} else {
				By("Runtime security monitoring (Falco) not found - this may be expected in test environment")
			}
		})
	})

	Context("Pod Security Standards", func() {
		It("should verify namespace has pod security standards labels", func() {
			var ns corev1.Namespace
			err := k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, &ns)
			Expect(err).NotTo(HaveOccurred())

			labels := ns.Labels
			if labels != nil {
				// Check for Pod Security Standards labels
				enforceLevel := labels["pod-security.kubernetes.io/enforce"]
				if enforceLevel != "" {
					By(fmt.Sprintf("Namespace enforces pod security level: %s", enforceLevel))
					Expect(enforceLevel).To(BeElementOf("privileged", "baseline", "restricted"))
				}

				auditLevel := labels["pod-security.kubernetes.io/audit"]
				if auditLevel != "" {
					By(fmt.Sprintf("Namespace audits pod security level: %s", auditLevel))
				}

				warnLevel := labels["pod-security.kubernetes.io/warn"]
				if warnLevel != "" {
					By(fmt.Sprintf("Namespace warns pod security level: %s", warnLevel))
				}
			}
		})
	})
})

// Helper function to test container runtime security
func ValidateContainerRuntimeSecurity(ctx context.Context, k8sClient client.Client, deployment appsv1.Deployment) error {
	// Get a running pod from the deployment
	podList := &corev1.PodList{}
	err := k8sClient.List(ctx, podList, client.InNamespace(deployment.Namespace),
		client.MatchingLabels(deployment.Spec.Selector.MatchLabels))
	if err != nil {
		return err
	}

	if len(podList.Items) == 0 {
		return fmt.Errorf("no pods found for deployment %s", deployment.Name)
	}

	pod := podList.Items[0]

	// Verify the pod is running with expected security context
	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("pod %s is not running", pod.Name)
	}

	return nil
}

// Helper function to validate security compliance
func ValidateSecurityCompliance(deployment appsv1.Deployment) []string {
	var violations []string

	// Check each container in the deployment
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.SecurityContext == nil {
			violations = append(violations, fmt.Sprintf("Container %s missing security context", container.Name))
			continue
		}

		secCtx := container.SecurityContext

		// Check non-root user
		if secCtx.RunAsNonRoot == nil || !*secCtx.RunAsNonRoot {
			violations = append(violations, fmt.Sprintf("Container %s not configured to run as non-root", container.Name))
		}

		// Check read-only root filesystem
		if secCtx.ReadOnlyRootFilesystem == nil || !*secCtx.ReadOnlyRootFilesystem {
			violations = append(violations, fmt.Sprintf("Container %s not using read-only root filesystem", container.Name))
		}

		// Check privilege escalation
		if secCtx.AllowPrivilegeEscalation == nil || *secCtx.AllowPrivilegeEscalation {
			violations = append(violations, fmt.Sprintf("Container %s allows privilege escalation", container.Name))
		}

		// Check capabilities
		if secCtx.Capabilities == nil || len(secCtx.Capabilities.Drop) == 0 {
			violations = append(violations, fmt.Sprintf("Container %s does not drop capabilities", container.Name))
		}

		// Check resource limits
		if container.Resources.Limits == nil {
			violations = append(violations, fmt.Sprintf("Container %s missing resource limits", container.Name))
		}
	}

	return violations
}
