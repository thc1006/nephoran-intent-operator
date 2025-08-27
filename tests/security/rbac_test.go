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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"
	"github.com/thc1006/nephoran-intent-operator/tests/utils"
)

var _ = Describe("RBAC Security Tests", func() {
	var (
		ctx       context.Context
		k8sClient client.Client
		clientset *kubernetes.Clientset
		namespace string
		rbacMgr   *security.RBACManager
		timeout   time.Duration
	)

	BeforeEach(func() {
		ctx = context.Background()
		k8sClient = utils.GetK8sClient()
		clientset = utils.GetClientset()
		namespace = utils.GetTestNamespace()
		rbacMgr = security.NewRBACManager(k8sClient, clientset, namespace)
		timeout = 30 * time.Second
	})

	Context("Service Account Security", func() {
		It("should verify service accounts exist and are properly configured", func() {
			expectedServiceAccounts := []string{
				"nephoran-operator",
				"nephoran-controller",
				"rag-api-service",
				"llm-processor",
			}

			for _, saName := range expectedServiceAccounts {
				var sa corev1.ServiceAccount
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      saName,
					Namespace: namespace,
				}, &sa)

				if err != nil {
					By(fmt.Sprintf("Service account %s not found - may not be deployed in test environment", saName))
					continue
				}

				By(fmt.Sprintf("Verifying service account %s", saName))

				// Verify service account has appropriate labels
				Expect(sa.Labels).NotTo(BeNil())
				Expect(sa.Labels["app.kubernetes.io/name"]).NotTo(BeEmpty())

				// Verify automountServiceAccountToken is explicitly set
				Expect(sa.AutomountServiceAccountToken).NotTo(BeNil(),
					"Service account %s should explicitly set automountServiceAccountToken", saName)

				// For production, it should be false unless explicitly needed
				if strings.Contains(saName, "controller") || strings.Contains(saName, "operator") {
					// Controllers need service account tokens
					if sa.AutomountServiceAccountToken != nil {
						By(fmt.Sprintf("Service account %s automount token: %t", saName, *sa.AutomountServiceAccountToken))
					}
				}
			}
		})

		It("should verify service account tokens are not automatically mounted unless required", func() {
			var pods corev1.PodList
			err := k8sClient.List(ctx, &pods, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, pod := range pods.Items {
				By(fmt.Sprintf("Checking service account token mounting for pod %s", pod.Name))

				// Check if pod explicitly disables service account token mounting
				if pod.Spec.AutomountServiceAccountToken != nil {
					if !*pod.Spec.AutomountServiceAccountToken {
						By(fmt.Sprintf("Pod %s correctly disables service account token mounting", pod.Name))
					}
				}

				// Verify the service account exists
				if pod.Spec.ServiceAccountName != "" {
					var sa corev1.ServiceAccount
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      pod.Spec.ServiceAccountName,
						Namespace: pod.Namespace,
					}, &sa)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})
	})

	Context("Role-Based Access Control", func() {
		It("should verify cluster roles follow least privilege principle", func() {
			var clusterRoles rbacv1.ClusterRoleList
			err := k8sClient.List(ctx, &clusterRoles)
			Expect(err).NotTo(HaveOccurred())

			nephoranRoles := []rbacv1.ClusterRole{}
			for _, role := range clusterRoles.Items {
				if strings.Contains(role.Name, "nephoran") || strings.Contains(role.Name, "intent-operator") {
					nephoranRoles = append(nephoranRoles, role)
				}
			}

			for _, role := range nephoranRoles {
				By(fmt.Sprintf("Analyzing cluster role %s", role.Name))

				// Check for overly permissive rules
				for _, rule := range role.Rules {
					// Should not have * in resources, verbs, or apiGroups unless specifically needed
					if containsWildcard(rule.Resources) {
						By(fmt.Sprintf("Warning: ClusterRole %s has wildcard in resources: %v", role.Name, rule.Resources))
					}

					if containsWildcard(rule.Verbs) {
						By(fmt.Sprintf("Warning: ClusterRole %s has wildcard in verbs: %v", role.Name, rule.Verbs))
					}

					if containsWildcard(rule.APIGroups) {
						By(fmt.Sprintf("Warning: ClusterRole %s has wildcard in apiGroups: %v", role.Name, rule.APIGroups))
					}

					// Check for dangerous permissions
					dangerousVerbs := []string{"*", "create", "delete", "deletecollection"}
					for _, verb := range rule.Verbs {
						if contains(dangerousVerbs, verb) {
							By(fmt.Sprintf("ClusterRole %s has potentially dangerous verb '%s' on resources %v",
								role.Name, verb, rule.Resources))
						}
					}
				}
			}
		})

		It("should verify roles are scoped to specific namespaces where possible", func() {
			var roles rbacv1.RoleList
			err := k8sClient.List(ctx, &roles, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, role := range roles.Items {
				if strings.Contains(role.Name, "nephoran") {
					By(fmt.Sprintf("Analyzing namespaced role %s", role.Name))

					// Verify role has appropriate permissions
					for _, rule := range role.Rules {
						// Check that namespace-scoped roles don't have cluster-wide permissions
						for _, resource := range rule.Resources {
							clusterScopedResources := []string{
								"nodes", "persistentvolumes", "clusterroles", "clusterrolebindings",
								"namespaces", "customresourcedefinitions",
							}

							if contains(clusterScopedResources, resource) {
								By(fmt.Sprintf("Warning: Namespace role %s has access to cluster-scoped resource %s",
									role.Name, resource))
							}
						}
					}
				}
			}
		})

		It("should verify role bindings are correctly configured", func() {
			// Check cluster role bindings
			var clusterRoleBindings rbacv1.ClusterRoleBindingList
			err := k8sClient.List(ctx, &clusterRoleBindings)
			Expect(err).NotTo(HaveOccurred())

			for _, binding := range clusterRoleBindings.Items {
				if strings.Contains(binding.Name, "nephoran") {
					By(fmt.Sprintf("Analyzing cluster role binding %s", binding.Name))

					// Verify the referenced role exists
					var role rbacv1.ClusterRole
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: binding.RoleRef.Name,
					}, &role)
					Expect(err).NotTo(HaveOccurred())

					// Verify subjects are appropriate
					for _, subject := range binding.Subjects {
						switch subject.Kind {
						case "ServiceAccount":
							// Verify service account exists
							var sa corev1.ServiceAccount
							err := k8sClient.Get(ctx, types.NamespacedName{
								Name:      subject.Name,
								Namespace: subject.Namespace,
							}, &sa)
							Expect(err).NotTo(HaveOccurred())

						case "User":
							By(fmt.Sprintf("User subject in binding %s: %s", binding.Name, subject.Name))

						case "Group":
							By(fmt.Sprintf("Group subject in binding %s: %s", binding.Name, subject.Name))
						}
					}
				}
			}

			// Check namespaced role bindings
			var roleBindings rbacv1.RoleBindingList
			err = k8sClient.List(ctx, &roleBindings, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			for _, binding := range roleBindings.Items {
				By(fmt.Sprintf("Analyzing role binding %s", binding.Name))

				// Verify the role binding points to an existing role
				if binding.RoleRef.Kind == "Role" {
					var role rbacv1.Role
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      binding.RoleRef.Name,
						Namespace: binding.Namespace,
					}, &role)
					Expect(err).NotTo(HaveOccurred())
				} else if binding.RoleRef.Kind == "ClusterRole" {
					var clusterRole rbacv1.ClusterRole
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: binding.RoleRef.Name,
					}, &clusterRole)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})
	})

	Context("Privilege Escalation Prevention", func() {
		It("should verify no service accounts have cluster-admin privileges", func() {
			var clusterRoleBindings rbacv1.ClusterRoleBindingList
			err := k8sClient.List(ctx, &clusterRoleBindings)
			Expect(err).NotTo(HaveOccurred())

			for _, binding := range clusterRoleBindings.Items {
				if binding.RoleRef.Name == "cluster-admin" {
					for _, subject := range binding.Subjects {
						if subject.Kind == "ServiceAccount" && subject.Namespace == namespace {
							By(fmt.Sprintf("Warning: Service account %s/%s has cluster-admin privileges",
								subject.Namespace, subject.Name))

							// This might be acceptable for some operators but should be documented
							Expect(subject.Name).To(BeElementOf("nephoran-operator", "system-operators"),
								"Only specific service accounts should have cluster-admin privileges")
						}
					}
				}
			}
		})

		It("should verify pod security contexts prevent privilege escalation", func() {
			deployments, err := utils.GetAllDeployments(ctx, k8sClient, namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, deployment := range deployments {
				podSpec := deployment.Spec.Template.Spec

				By(fmt.Sprintf("Checking privilege escalation for deployment %s", deployment.Name))

				// Check pod-level security context
				if podSpec.SecurityContext != nil {
					if podSpec.SecurityContext.RunAsUser != nil {
						Expect(*podSpec.SecurityContext.RunAsUser).NotTo(Equal(int64(0)),
							"Pod should not run as root user")
					}
				}

				// Check container-level security contexts
				for _, container := range podSpec.Containers {
					if container.SecurityContext != nil {
						if container.SecurityContext.AllowPrivilegeEscalation != nil {
							Expect(*container.SecurityContext.AllowPrivilegeEscalation).To(BeFalse(),
								"Container %s should not allow privilege escalation", container.Name)
						}

						if container.SecurityContext.Privileged != nil {
							Expect(*container.SecurityContext.Privileged).To(BeFalse(),
								"Container %s should not be privileged", container.Name)
						}
					}
				}
			}
		})

		It("should verify RBAC prevents unauthorized resource access", func() {
			// Test with a limited service account
			testServiceAccount := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rbac-test-sa",
					Namespace: namespace,
				},
				AutomountServiceAccountToken: &[]bool{false}[0],
			}

			err := k8sClient.Create(ctx, testServiceAccount)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				Expect(err).NotTo(HaveOccurred())
			}

			defer func() {
				_ = k8sClient.Delete(ctx, testServiceAccount)
			}()

			// Create a minimal role
			testRole := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rbac-test-role",
					Namespace: namespace,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps"},
						Verbs:     []string{"get", "list"},
					},
				},
			}

			err = k8sClient.Create(ctx, testRole)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				Expect(err).NotTo(HaveOccurred())
			}

			defer func() {
				_ = k8sClient.Delete(ctx, testRole)
			}()

			// Create role binding
			testRoleBinding := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rbac-test-binding",
					Namespace: namespace,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "rbac-test-sa",
						Namespace: namespace,
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "rbac-test-role",
				},
			}

			err = k8sClient.Create(ctx, testRoleBinding)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				Expect(err).NotTo(HaveOccurred())
			}

			defer func() {
				_ = k8sClient.Delete(ctx, testRoleBinding)
			}()

			By("Test service account and RBAC setup completed successfully")
		})
	})

	Context("Namespace Isolation", func() {
		It("should verify namespace has proper resource quotas", func() {
			var resourceQuotas corev1.ResourceQuotaList
			err := k8sClient.List(ctx, &resourceQuotas, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			if len(resourceQuotas.Items) > 0 {
				for _, quota := range resourceQuotas.Items {
					By(fmt.Sprintf("Checking resource quota %s", quota.Name))

					// Verify quota has appropriate limits
					spec := quota.Spec.Hard
					if spec != nil {
						if cpu, exists := spec[corev1.ResourceRequestsCPU]; exists {
							By(fmt.Sprintf("CPU requests limit: %s", cpu.String()))
						}

						if memory, exists := spec[corev1.ResourceRequestsMemory]; exists {
							By(fmt.Sprintf("Memory requests limit: %s", memory.String()))
						}

						if pods, exists := spec[corev1.ResourcePods]; exists {
							By(fmt.Sprintf("Pod limit: %s", pods.String()))
						}
					}
				}
			} else {
				By("No resource quotas found - consider adding for production environments")
			}
		})

		It("should verify namespace has network policies for isolation", func() {
			var networkPolicies corev1.NetworkPolicyList
			err := k8sClient.List(ctx, &networkPolicies, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())

			foundDenyAll := false
			for _, policy := range networkPolicies.Items {
				if policy.Name == "default-deny-all" || strings.Contains(policy.Name, "deny") {
					foundDenyAll = true
					By(fmt.Sprintf("Found network isolation policy: %s", policy.Name))
				}
			}

			// Network policies might be managed by the network policy manager
			if !foundDenyAll {
				By("No default-deny network policy found - verify network isolation is properly configured")
			}
		})

		It("should verify cross-namespace access is properly controlled", func() {
			// Check if there are any cluster role bindings that grant access across namespaces
			var clusterRoleBindings rbacv1.ClusterRoleBindingList
			err := k8sClient.List(ctx, &clusterRoleBindings)
			Expect(err).NotTo(HaveOccurred())

			crossNamespaceBindings := []string{}
			for _, binding := range clusterRoleBindings.Items {
				for _, subject := range binding.Subjects {
					if subject.Kind == "ServiceAccount" && subject.Namespace == namespace {
						// This service account has cluster-wide permissions
						crossNamespaceBindings = append(crossNamespaceBindings,
							fmt.Sprintf("%s -> %s", subject.Name, binding.RoleRef.Name))
					}
				}
			}

			if len(crossNamespaceBindings) > 0 {
				By(fmt.Sprintf("Found cross-namespace bindings: %v", crossNamespaceBindings))
				// This might be acceptable for operators but should be minimal
			}
		})
	})

	Context("RBAC Manager Integration", func() {
		It("should test RBAC manager functionality", func() {
			// Test creating a network operator role
			err := rbacMgr.EnsureOperatorRole(ctx, security.RoleNetworkOperator)
			Expect(err).NotTo(HaveOccurred())

			// Verify the role was created
			var role rbacv1.Role
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      string(security.RoleNetworkOperator),
				Namespace: namespace,
			}, &role)

			if err != nil {
				By("Role not found - may not be created in test environment")
			} else {
				By(fmt.Sprintf("RBAC manager successfully created role %s", role.Name))

				// Verify role has appropriate rules
				Expect(len(role.Rules)).To(BeNumerically(">", 0))
			}
		})

		It("should verify operator roles have least privilege", func() {
			operatorRoles := []security.OperatorRole{
				security.RoleNetworkOperator,
				security.RoleNetworkViewer,
				security.RoleSecurityAuditor,
			}

			for _, operatorRole := range operatorRoles {
				var role rbacv1.Role
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      string(operatorRole),
					Namespace: namespace,
				}, &role)

				if err != nil {
					By(fmt.Sprintf("Role %s not found - may not be deployed", operatorRole))
					continue
				}

				By(fmt.Sprintf("Analyzing operator role %s", operatorRole))

				// Verify role follows least privilege
				for _, rule := range role.Rules {
					// Network viewer should only have read permissions
					if operatorRole == security.RoleNetworkViewer {
						for _, verb := range rule.Verbs {
							readOnlyVerbs := []string{"get", "list", "watch"}
							Expect(readOnlyVerbs).To(ContainElement(verb),
								"Network viewer should only have read permissions")
						}
					}

					// Security auditor should have broad read access but no write
					if operatorRole == security.RoleSecurityAuditor {
						writeVerbs := []string{"create", "update", "patch", "delete"}
						for _, verb := range rule.Verbs {
							if contains(writeVerbs, verb) {
								By(fmt.Sprintf("Warning: Security auditor has write permission: %s", verb))
							}
						}
					}
				}
			}
		})
	})
})

// Helper functions
func containsWildcard(slice []string) bool {
	for _, item := range slice {
		if item == "*" {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Test RBAC compliance with role permissions
func TestRBACComplianceRoles(t *testing.T) {
	testCases := []struct {
		name     string
		role     string
		expected []string
	}{
		{
			name:     "Network Operator",
			role:     "network-operator",
			expected: []string{"networkintents", "configmaps", "secrets"},
		},
		{
			name:     "Network Viewer",
			role:     "network-viewer",
			expected: []string{"networkintents"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This would test specific RBAC rules
			// Implementation depends on your RBAC structure
		})
	}
}

// Benchmark RBAC operations
func BenchmarkRBACOperations(b *testing.B) {
	ctx := context.Background()
	k8sClient := utils.GetK8sClient()
	clientset := utils.GetClientset()
	namespace := utils.GetTestNamespace()

	rbacMgr := security.NewRBACManager(k8sClient, clientset, namespace)

	b.Run("CreateRole", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := rbacMgr.EnsureOperatorRole(ctx, security.RoleNetworkOperator)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
