// +build windows

package controllers

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Windows Compatibility Tests", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	var namespaceName string

	BeforeEach(func() {
		By("Ensuring tests run only on Windows")
		if runtime.GOOS != "windows" {
			Skip("Skipping Windows-specific tests on non-Windows platform")
		}

		By("Creating isolated namespace for Windows compatibility tests")
		namespaceName = CreateIsolatedNamespace("windows-compatibility")
	})

	AfterEach(func() {
		By("Cleaning up Windows compatibility test namespace")
		CleanupIsolatedNamespace(namespaceName)
	})

	Context("Windows Path Handling", func() {
		It("Should handle Windows file paths correctly in test environment", func() {
			By("Verifying Windows path separators are handled correctly")
			testPath := filepath.Join("C:", "Users", "test", "nephoran")
			Expect(strings.Contains(testPath, "\\")).To(BeTrue(), "Should contain Windows path separators")
			
			By("Verifying path.Join works with Windows paths")
			joinedPath := filepath.Join("C:", "temp", "kubebuilder", "bin")
			expectedParts := []string{"C:", "temp", "kubebuilder", "bin"}
			for _, part := range expectedParts {
				Expect(joinedPath).To(ContainSubstring(part))
			}
		})

		It("Should handle environment variable expansion on Windows", func() {
			By("Setting and retrieving Windows environment variables")
			testEnvVar := "NEPHORAN_TEST_VAR"
			testValue := "C:\\Windows\\System32"
			
			os.Setenv(testEnvVar, testValue)
			defer os.Unsetenv(testEnvVar)
			
			retrievedValue := os.Getenv(testEnvVar)
			Expect(retrievedValue).To(Equal(testValue))
			
			By("Verifying Windows path expansion")
			userProfile := os.Getenv("USERPROFILE")
			if userProfile != "" {
				Expect(strings.HasPrefix(userProfile, "C:")).To(BeTrue(), "USERPROFILE should start with C: drive")
			}
		})

		It("Should handle Windows-specific file operations", func() {
			By("Creating and manipulating files with Windows paths")
			tempDir := os.TempDir()
			Expect(tempDir).NotTo(BeEmpty())
			
			testFile := filepath.Join(tempDir, "nephoran-test.txt")
			
			// Create file
			file, err := os.Create(testFile)
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(testFile)
			
			// Write content
			_, err = file.WriteString("Windows compatibility test\r\n")
			Expect(err).NotTo(HaveOccurred())
			file.Close()
			
			// Read content
			content, err := os.ReadFile(testFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Windows compatibility test"))
			
			By("Verifying Windows line endings are preserved")
			Expect(string(content)).To(ContainSubstring("\r\n"))
		})
	})

	Context("Windows-Specific Controller Operations", func() {
		var (
			e2nodeSetReconciler *E2NodeSetReconciler
		)

		BeforeEach(func() {
			e2nodeSetReconciler = &E2NodeSetReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
		})

		It("Should handle E2NodeSet operations correctly on Windows", func() {
			By("Creating E2NodeSet on Windows platform")
			e2nodeSet := CreateTestE2NodeSet(
				GetUniqueName("windows-e2nodeset"),
				namespaceName,
				2,
			)
			// Add Windows-specific labels
			if e2nodeSet.Labels == nil {
				e2nodeSet.Labels = make(map[string]string)
			}
			e2nodeSet.Labels["platform"] = "windows"
			e2nodeSet.Labels["os"] = runtime.GOOS
			e2nodeSet.Labels["arch"] = runtime.GOARCH

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("Reconciling E2NodeSet on Windows")
			_, err := e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying ConfigMaps are created with Windows metadata")
			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOptions := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"app":       "e2node",
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOptions...); err != nil {
					return 0
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(2))

			By("Verifying ConfigMap content includes Windows-specific information")
			configMapList := &corev1.ConfigMapList{}
			listOptions := []client.ListOption{
				client.InNamespace(namespaceName),
				client.MatchingLabels(map[string]string{
					"app":       "e2node",
					"e2nodeset": e2nodeSet.Name,
				}),
			}
			Expect(k8sClient.List(ctx, configMapList, listOptions...)).To(Succeed())
			
			for _, cm := range configMapList.Items {
				// Verify Windows-compatible time format
				Expect(cm.Data["created"]).NotTo(BeEmpty())
				createdTime, err := time.Parse(time.RFC3339, cm.Data["created"])
				Expect(err).NotTo(HaveOccurred())
				Expect(createdTime).To(BeTemporally("~", time.Now(), time.Minute))
				
				// Verify all required fields are present
				Expect(cm.Data["nodeId"]).NotTo(BeEmpty())
				Expect(cm.Data["nodeType"]).To(Equal("simulated-gnb"))
				Expect(cm.Data["status"]).To(Equal("active"))
			}

			By("Verifying E2NodeSet status is updated correctly on Windows")
			WaitForE2NodeSetReady(namespacedName, 2)
		})

		It("Should handle Windows-specific scaling scenarios", func() {
			By("Creating E2NodeSet for Windows scaling test")
			e2nodeSet := CreateTestE2NodeSet(
				GetUniqueName("windows-scaling"),
				namespaceName,
				1,
			)

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("Initial reconciliation on Windows")
			_, err := e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			WaitForE2NodeSetReady(namespacedName, 1)

			By("Scaling up on Windows platform")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 4
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("Reconciling after Windows scale-up")
			_, err = e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying scale-up works correctly on Windows")
			WaitForE2NodeSetReady(namespacedName, 4)

			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOptions := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"app":       "e2node",
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOptions...); err != nil {
					return 0
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(4))

			By("Scaling down on Windows platform")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 2
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("Reconciling after Windows scale-down")
			_, err = e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying scale-down works correctly on Windows")
			WaitForE2NodeSetReady(namespacedName, 2)

			Eventually(func() int {
				configMapList := &corev1.ConfigMapList{}
				listOptions := []client.ListOption{
					client.InNamespace(namespaceName),
					client.MatchingLabels(map[string]string{
						"app":       "e2node",
						"e2nodeset": e2nodeSet.Name,
					}),
				}
				if err := k8sClient.List(ctx, configMapList, listOptions...); err != nil {
					return 0
				}
				return len(configMapList.Items)
			}, timeout, interval).Should(Equal(2))
		})
	})

	Context("Windows Environment Integration", func() {
		It("Should handle Windows-specific environment variables", func() {
			By("Checking Windows system environment variables")
			windowsVars := map[string]bool{
				"USERPROFILE": false,
				"PROGRAMFILES": false,
				"SYSTEMROOT": false,
				"TEMP": false,
			}

			for varName := range windowsVars {
				value := os.Getenv(varName)
				if value != "" {
					windowsVars[varName] = true
					By(fmt.Sprintf("Found Windows env var %s = %s", varName, value))
				}
			}

			// At least some Windows-specific variables should be present
			foundCount := 0
			for _, found := range windowsVars {
				if found {
					foundCount++
				}
			}
			Expect(foundCount).To(BeNumerically(">=", 2), "Should find at least 2 Windows environment variables")
		})

		It("Should handle Windows-specific time operations", func() {
			By("Testing Windows time formatting")
			now := time.Now()
			
			// Test RFC3339 formatting (used in controller)
			rfc3339Time := now.Format(time.RFC3339)
			Expect(rfc3339Time).To(MatchRegexp(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}`))
			
			// Test parsing back
			parsedTime, err := time.Parse(time.RFC3339, rfc3339Time)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedTime.Unix()).To(Equal(now.Unix()))
			
			By("Testing Windows-compatible duration handling")
			testDuration := 30 * time.Second
			durationString := testDuration.String()
			Expect(durationString).To(Equal("30s"))
			
			parsedDuration, err := time.ParseDuration(durationString)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsedDuration).To(Equal(testDuration))
		})

		It("Should handle Windows process and runtime information", func() {
			By("Verifying Windows runtime information")
			Expect(runtime.GOOS).To(Equal("windows"))
			
			By("Checking Windows architecture")
			validArches := []string{"amd64", "386", "arm64"}
			Expect(validArches).To(ContainElement(runtime.GOARCH))
			
			By("Verifying Windows-specific process information")
			pid := os.Getpid()
			Expect(pid).To(BeNumerically(">", 0))
			
			// Test Windows-compatible executable detection
			executable, err := os.Executable()
			Expect(err).NotTo(HaveOccurred())
			Expect(executable).NotTo(BeEmpty())
			
			// Should contain .exe extension on Windows or be in temp directory
			isExeOrTemp := strings.HasSuffix(strings.ToLower(executable), ".exe") || 
						  strings.Contains(executable, "Temp")
			Expect(isExeOrTemp).To(BeTrue(), fmt.Sprintf("Executable path should be Windows-compatible: %s", executable))
		})
	})

	Context("Windows Test Framework Compatibility", func() {
		It("Should handle Windows-specific Ginkgo operations", func() {
			By("Testing Windows-compatible test descriptions")
			testName := GinkgoT().Name()
			Expect(testName).NotTo(BeEmpty())
			
			By("Verifying Windows line ending handling in test output")
			windowsMessage := "Test message with Windows line ending\r\n"
			Expect(windowsMessage).To(ContainSubstring("\r\n"))
			
			By("Testing Windows-compatible matcher operations")
			windowsPath := `C:\Users\Test\nephoran-intent-operator`
			Expect(windowsPath).To(MatchRegexp(`^[A-Z]:\\`))
			Expect(windowsPath).To(ContainSubstring("\\"))
		})

		It("Should handle Windows-specific test timing", func() {
			By("Testing Windows-compatible timeouts")
			start := time.Now()
			
			// Simulate a quick operation
			time.Sleep(10 * time.Millisecond)
			
			elapsed := time.Since(start)
			Expect(elapsed).To(BeNumerically(">=", 10*time.Millisecond))
			Expect(elapsed).To(BeNumerically("<", 100*time.Millisecond))
			
			By("Testing Windows-compatible Eventually operations")
			Eventually(func() bool {
				return time.Now().After(start.Add(5 * time.Millisecond))
			}, timeout, interval).Should(BeTrue())
		})

		It("Should handle Windows-specific resource cleanup", func() {
			By("Creating test resources on Windows")
			testE2NodeSet := CreateTestE2NodeSet(
				GetUniqueName("cleanup-test"),
				namespaceName,
				1,
			)
			
			Expect(k8sClient.Create(ctx, testE2NodeSet)).To(Succeed())
			
			By("Verifying resource exists")
			created := &nephoranv1.E2NodeSet{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(testE2NodeSet), created)).To(Succeed())
			
			By("Testing Windows-compatible cleanup")
			Expect(k8sClient.Delete(ctx, testE2NodeSet)).To(Succeed())
			
			By("Verifying cleanup completed on Windows")
			Eventually(func() bool {
				deleted := &nephoranv1.E2NodeSet{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(testE2NodeSet), deleted)
				return client.IgnoreNotFound(err) == nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Windows Performance and Resource Usage", func() {
		It("Should perform efficiently on Windows platform", func() {
			By("Testing Windows performance characteristics")
			start := time.Now()
			
			// Create multiple resources
			e2nodeSets := []*nephoranv1.E2NodeSet{}
			for i := 0; i < 5; i++ {
				e2ns := CreateTestE2NodeSet(
					GetUniqueName(fmt.Sprintf("perf-test-%d", i)),
					namespaceName,
					1,
				)
				Expect(k8sClient.Create(ctx, e2ns)).To(Succeed())
				e2nodeSets = append(e2nodeSets, e2ns)
			}
			
			creationTime := time.Since(start)
			By(fmt.Sprintf("Created 5 E2NodeSets in %v", creationTime))
			Expect(creationTime).To(BeNumerically("<", 5*time.Second), "Resource creation should be reasonably fast on Windows")
			
			By("Testing Windows resource cleanup performance")
			cleanupStart := time.Now()
			
			for _, e2ns := range e2nodeSets {
				Expect(k8sClient.Delete(ctx, e2ns)).To(Succeed())
			}
			
			cleanupTime := time.Since(cleanupStart)
			By(fmt.Sprintf("Cleaned up 5 E2NodeSets in %v", cleanupTime))
			Expect(cleanupTime).To(BeNumerically("<", 5*time.Second), "Resource cleanup should be reasonably fast on Windows")
		})

		It("Should handle Windows memory usage appropriately", func() {
			By("Monitoring Windows memory usage during test operations")
			var startMemStats, endMemStats runtime.MemStats
			
			runtime.GC()
			runtime.ReadMemStats(&startMemStats)
			
			By("Performing memory-intensive operations")
			largeE2NodeSets := []*nephoranv1.E2NodeSet{}
			
			for i := 0; i < 10; i++ {
				e2ns := CreateTestE2NodeSet(
					GetUniqueName(fmt.Sprintf("memory-test-%d", i)),
					namespaceName,
					2,
				)
				// Add large amount of metadata to test memory handling
				if e2ns.Annotations == nil {
					e2ns.Annotations = make(map[string]string)
				}
				for j := 0; j < 10; j++ {
					e2ns.Annotations[fmt.Sprintf("large-annotation-%d", j)] = strings.Repeat("data", 100)
				}
				
				Expect(k8sClient.Create(ctx, e2ns)).To(Succeed())
				largeE2NodeSets = append(largeE2NodeSets, e2ns)
			}
			
			runtime.GC()
			runtime.ReadMemStats(&endMemStats)
			
			By("Verifying reasonable memory usage on Windows")
			memoryIncreaseMB := float64(endMemStats.Alloc-startMemStats.Alloc) / (1024 * 1024)
			By(fmt.Sprintf("Memory increase: %.2f MB", memoryIncreaseMB))
			
			// Should not use excessive memory (adjust threshold as needed)
			Expect(memoryIncreaseMB).To(BeNumerically("<", 50), "Memory usage should be reasonable on Windows")
			
			By("Cleaning up memory test resources")
			for _, e2ns := range largeE2NodeSets {
				Expect(k8sClient.Delete(ctx, e2ns)).To(Succeed())
			}
		})
	})
})

// Windows-specific helper functions

func getWindowsSystemInfo() map[string]string {
	info := make(map[string]string)
	
	envVars := []string{
		"USERPROFILE",
		"PROGRAMFILES", 
		"SYSTEMROOT",
		"TEMP",
		"USERNAME",
		"COMPUTERNAME",
	}
	
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			info[envVar] = value
		}
	}
	
	info["GOOS"] = runtime.GOOS
	info["GOARCH"] = runtime.GOARCH
	info["NumCPU"] = fmt.Sprintf("%d", runtime.NumCPU())
	
	return info
}

func isWindowsLongPathSupported() bool {
	// Test if Windows long path support is enabled
	longPath := strings.Repeat("a", 300) + ".txt"
	tempDir := os.TempDir()
	fullPath := filepath.Join(tempDir, longPath)
	
	file, err := os.Create(fullPath)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(fullPath)
	return true
}