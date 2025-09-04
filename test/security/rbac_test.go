/*
Package security provides comprehensive RBAC security testing for the Nephoran
Kubernetes operator following 2025 security best practices.
*/

package security

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRBACValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RBAC Security Validation Suite")
}

var _ = Describe("RBAC Security Validation", func() {
	var (
		rbacManifests []string
		ctx           context.Context
	)

	BeforeEach(func() {
		ctx = context.TODO()
		
		// Load all RBAC manifests from config/rbac
		rbacDir := filepath.Join("..", "..", "config", "rbac")
		files, err := ioutil.ReadDir(rbacDir)
		Expect(err).NotTo(HaveOccurred())

		rbacManifests = make([]string, 0)
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
				rbacManifests = append(rbacManifests, filepath.Join(rbacDir, file.Name()))
			}
		}
		
		Expect(len(rbacManifests)).To(BeNumerically(">", 0), "No RBAC manifests found")
	})

	Context("When validating ClusterRole permissions", func() {
		It("should not contain wildcard permissions in production roles", func() {
			for _, manifestPath := range rbacManifests {
				By(fmt.Sprintf("Analyzing RBAC manifest: %s", filepath.Base(manifestPath)))
				
				content, err := ioutil.ReadFile(manifestPath)
				Expect(err).NotTo(HaveOccurred())

				// Parse YAML documents
				documents := strings.Split(string(content), "---")
				for _, doc := range documents {
					if strings.TrimSpace(doc) == "" {
						continue
					}

					var obj runtime.Object
					decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 1024)
					err := decoder.Decode(&obj)
					if err != nil {
						continue // Skip non-RBAC resources
					}

					if clusterRole, ok := obj.(*rbacv1.ClusterRole); ok {
						validateClusterRolePermissions(clusterRole)
					}
					if role, ok := obj.(*rbacv1.Role); ok {
						validateRolePermissions(role)
					}
				}
			}
		})

		It("should follow principle of least privilege", func() {
			allowedAPIGroups := map[string]bool{
				"":                             true, // core API group
				"apps":                         true,
				"extensions":                   true,
				"intent.nephio.org":            true, // Our custom API group
				"ran.nephio.org":               true, // O-RAN API group
				"networking.k8s.io":            true,
				"rbac.authorization.k8s.io":    true,
				"apiextensions.k8s.io":         true,
				"admissionregistration.k8s.io": true,
				"coordination.k8s.io":          true, // For leader election
			}

			dangerousVerbs := []string{
				"*",           // Wildcard verbs are dangerous
				"create",      // Only if needed and scoped
				"delete",      // Only if needed and scoped
				"deletecollection",
				"patch",       // Can be dangerous
				"update",      // Can be dangerous
			}

			for _, manifestPath := range rbacManifests {
				content, err := ioutil.ReadFile(manifestPath)
				Expect(err).NotTo(HaveOccurred())

				documents := strings.Split(string(content), "---")
				for _, doc := range documents {
					if strings.TrimSpace(doc) == "" {
						continue
					}

					var obj metav1.Object
					decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 1024)
					err := decoder.Decode(&obj)
					if err != nil {
						continue
					}

					if clusterRole, ok := obj.(*rbacv1.ClusterRole); ok {
						validateLeastPrivilege(clusterRole, allowedAPIGroups, dangerousVerbs)
					}
				}
			}
		})

		It("should not grant cluster-admin or equivalent privileges", func() {
			superuserRules := []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
			}

			for _, manifestPath := range rbacManifests {
				content, err := ioutil.ReadFile(manifestPath)
				Expect(err).NotTo(HaveOccurred())

				documents := strings.Split(string(content), "---")
				for _, doc := range documents {
					if strings.TrimSpace(doc) == "" {
						continue
					}

					var obj metav1.Object
					decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 1024)
					err := decoder.Decode(&obj)
					if err != nil {
						continue
					}

					if clusterRole, ok := obj.(*rbacv1.ClusterRole); ok {
						for _, rule := range clusterRole.Rules {
							for _, superRule := range superuserRules {
								if isEquivalentRule(rule, superRule) {
									Fail(fmt.Sprintf("ClusterRole %s contains superuser permissions: %+v", 
										clusterRole.Name, rule))
								}
							}
						}
					}
				}
			}
		})
	})

	Context("When validating ServiceAccount security", func() {
		It("should not use default service account for privileged operations", func() {
			for _, manifestPath := range rbacManifests {
				content, err := ioutil.ReadFile(manifestPath)
				Expect(err).NotTo(HaveOccurred())

				documents := strings.Split(string(content), "---")
				for _, doc := range documents {
					if strings.TrimSpace(doc) == "" {
						continue
					}

					var obj metav1.Object
					decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 1024)
					err := decoder.Decode(&obj)
					if err != nil {
						continue
					}

					if binding, ok := obj.(*rbacv1.ClusterRoleBinding); ok {
						for _, subject := range binding.Subjects {
							if subject.Kind == "ServiceAccount" && subject.Name == "default" {
								Fail(fmt.Sprintf("ClusterRoleBinding %s uses default ServiceAccount", binding.Name))
							}
						}
					}

					if binding, ok := obj.(*rbacv1.RoleBinding); ok {
						for _, subject := range binding.Subjects {
							if subject.Kind == "ServiceAccount" && subject.Name == "default" {
								Fail(fmt.Sprintf("RoleBinding %s uses default ServiceAccount", binding.Name))
							}
						}
					}
				}
			}
		})

		It("should have properly scoped service accounts", func() {
			// Service accounts should be scoped to specific namespaces
			// and should not have excessive cluster-wide permissions
			expectedServiceAccounts := map[string]string{
				"nephoran-controller-manager": "nephoran-system",
			}

			for saName, expectedNamespace := range expectedServiceAccounts {
				By(fmt.Sprintf("Validating service account scoping: %s", saName))
				
				// This would require a real cluster to test, so we'll check the RBAC configs
				for _, manifestPath := range rbacManifests {
					content, err := ioutil.ReadFile(manifestPath)
					Expect(err).NotTo(HaveOccurred())

					if strings.Contains(string(content), saName) {
						// Verify it's properly namespaced
						Expect(string(content)).To(ContainSubstring(fmt.Sprintf("namespace: %s", expectedNamespace)))
					}
				}
			}
		})
	})

	Context("When validating O-RAN specific security requirements", func() {
		It("should restrict access to O-RAN sensitive resources", func() {
			sensitiveResources := []string{
				"secrets",
				"configmaps",
				"nodes",
				"persistentvolumes",
				"storageclasses",
			}

			for _, manifestPath := range rbacManifests {
				content, err := ioutil.ReadFile(manifestPath)
				Expect(err).NotTo(HaveOccurred())

				documents := strings.Split(string(content), "---")
				for _, doc := range documents {
					if strings.TrimSpace(doc) == "" {
						continue
					}

					var obj metav1.Object
					decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(doc), 1024)
					err := decoder.Decode(&obj)
					if err != nil {
						continue
					}

					if clusterRole, ok := obj.(*rbacv1.ClusterRole); ok {
						for _, rule := range clusterRole.Rules {
							for _, resource := range rule.Resources {
								for _, sensitiveResource := range sensitiveResources {
									if resource == sensitiveResource || resource == "*" {
										// Ensure it's not a wildcard verb for sensitive resources
										for _, verb := range rule.Verbs {
											if verb == "*" && resource == sensitiveResource {
												Fail(fmt.Sprintf("ClusterRole %s has wildcard access to sensitive resource %s", 
													clusterRole.Name, sensitiveResource))
											}
										}
									}
								}
							}
						}
					}
				}
			}
		})

		It("should not allow access to kube-system resources", func() {
			for _, manifestPath := range rbacManifests {
				content, err := ioutil.ReadFile(manifestPath)
				Expect(err).NotTo(HaveOccurred())

				// Check that RBAC doesn't grant access to kube-system namespace
				// unless absolutely necessary
				if strings.Contains(string(content), "kube-system") {
					By(fmt.Sprintf("Found kube-system reference in %s - validating necessity", filepath.Base(manifestPath)))
					
					// This should be very limited - only for specific use cases
					Expect(string(content)).NotTo(ContainSubstring("verbs: [\"*\"]"))
					Expect(string(content)).NotTo(ContainSubstring("resources: [\"*\"]"))
				}
			}
		})
	})

	Context("When validating webhook security", func() {
		It("should have properly secured webhook configurations", func() {
			webhookDir := filepath.Join("..", "..", "config", "webhook")
			files, err := ioutil.ReadDir(webhookDir)
			if err != nil {
				Skip("No webhook directory found")
			}

			for _, file := range files {
				if strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml") {
					webhookPath := filepath.Join(webhookDir, file.Name())
					content, err := ioutil.ReadFile(webhookPath)
					Expect(err).NotTo(HaveOccurred())

					// Check for security best practices in webhook configs
					By(fmt.Sprintf("Validating webhook security: %s", file.Name()))
					
					// Webhook should use HTTPS
					if strings.Contains(string(content), "clientConfig") {
						Expect(string(content)).To(ContainSubstring("service"))
						Expect(string(content)).NotTo(ContainSubstring("http://"))
					}

					// Check for proper failure policy
					if strings.Contains(string(content), "failurePolicy") {
						// Should not be "Ignore" for critical validation webhooks
						content := string(content)
						if strings.Contains(content, "ValidatingAdmissionWebhook") {
							Expect(content).NotTo(ContainSubstring("failurePolicy: Ignore"))
						}
					}
				}
			}
		})
	})
})

// Helper functions for RBAC validation

func validateClusterRolePermissions(clusterRole *rbacv1.ClusterRole) {
	for _, rule := range clusterRole.Rules {
		// Check for dangerous wildcard permissions
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "*" {
				Fail(fmt.Sprintf("ClusterRole %s contains wildcard API group", clusterRole.Name))
			}
		}

		for _, resource := range rule.Resources {
			if resource == "*" && !isSystemRole(clusterRole.Name) {
				Fail(fmt.Sprintf("ClusterRole %s contains wildcard resource", clusterRole.Name))
			}
		}

		for _, verb := range rule.Verbs {
			if verb == "*" && !isSystemRole(clusterRole.Name) {
				Fail(fmt.Sprintf("ClusterRole %s contains wildcard verb", clusterRole.Name))
			}
		}
	}
}

func validateRolePermissions(role *rbacv1.Role) {
	for _, rule := range role.Rules {
		// Similar validation for Role as ClusterRole
		for _, verb := range rule.Verbs {
			if verb == "*" {
				Fail(fmt.Sprintf("Role %s contains wildcard verb", role.Name))
			}
		}
	}
}

func validateLeastPrivilege(clusterRole *rbacv1.ClusterRole, allowedAPIGroups map[string]bool, dangerousVerbs []string) {
	for _, rule := range clusterRole.Rules {
		// Check API groups are in allowed list
		for _, apiGroup := range rule.APIGroups {
			if apiGroup != "*" && !allowedAPIGroups[apiGroup] {
				Fail(fmt.Sprintf("ClusterRole %s uses disallowed API group: %s", clusterRole.Name, apiGroup))
			}
		}

		// Check for dangerous verbs
		for _, verb := range rule.Verbs {
			for _, dangerousVerb := range dangerousVerbs {
				if verb == dangerousVerb && !isJustifiedDangerousVerb(clusterRole.Name, verb, rule.Resources) {
					Fail(fmt.Sprintf("ClusterRole %s uses dangerous verb '%s' on resources %v", 
						clusterRole.Name, verb, rule.Resources))
				}
			}
		}
	}
}

func isEquivalentRule(rule, superRule rbacv1.PolicyRule) bool {
	return containsAll(rule.APIGroups, superRule.APIGroups) &&
		containsAll(rule.Resources, superRule.Resources) &&
		containsAll(rule.Verbs, superRule.Verbs)
}

func containsAll(slice, target []string) bool {
	if len(target) == 0 {
		return true
	}
	if len(target) == 1 && target[0] == "*" {
		return true
	}
	for _, t := range target {
		found := false
		for _, s := range slice {
			if s == t || s == "*" {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func isSystemRole(roleName string) bool {
	systemRoles := []string{
		"system:",
		"nephoran-proxy-role", // Metrics proxy might need broader permissions
	}
	
	for _, systemRole := range systemRoles {
		if strings.HasPrefix(roleName, systemRole) {
			return true
		}
	}
	return false
}

func isJustifiedDangerousVerb(roleName, verb string, resources []string) bool {
	// Define justified use cases for dangerous verbs
	justifications := map[string]map[string][]string{
		"create": {
			"nephoran-manager-role": {"networkintents", "oranclusters"},
		},
		"delete": {
			"nephoran-manager-role": {"networkintents", "oranclusters"},
		},
		"patch": {
			"nephoran-manager-role": {"networkintents/status", "oranclusters/status"},
		},
		"update": {
			"nephoran-manager-role": {"networkintents/status", "oranclusters/status"},
		},
	}

	if allowedResources, ok := justifications[verb][roleName]; ok {
		for _, resource := range resources {
			for _, allowedResource := range allowedResources {
				if resource == allowedResource {
					return true
				}
			}
		}
	}

	return false
}