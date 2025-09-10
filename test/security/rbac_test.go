/*
Package security provides comprehensive RBAC security testing for the Nephoran
Kubernetes operator following 2025 security best practices.
*/

package security

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
<<<<<<< HEAD
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("RBAC Security Tests", func() {
	Context("When validating operator permissions", func() {
		It("Should have minimal required permissions", func() {
			ctx := context.Background()
			
			// Test that ClusterRole exists and has appropriate permissions
			clusterRole := &rbacv1.ClusterRole{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name: "manager-role",
			}, clusterRole)
			
			if err == nil {
				// Verify permissions are minimal
				Expect(clusterRole.Rules).ToNot(BeEmpty())
=======
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestRBACValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RBAC Security Validation Suite")
}

var _ = Describe("RBAC Security Validation", func() {
	var (
		rbacManifests []string
	)

	BeforeEach(func() {
		
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
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				
				for _, rule := range clusterRole.Rules {
					// Ensure no wildcard permissions
					Expect(rule.Resources).ToNot(ContainElement("*"))
					Expect(rule.Verbs).ToNot(ContainElement("*"))
				}
			}
		})
	})
})
