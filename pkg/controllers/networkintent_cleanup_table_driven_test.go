package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configPkg "github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
	"github.com/thc1006/nephoran-intent-operator/hack/testtools"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

// Import constants from config package
// constants.NetworkIntentFinalizer is accessed via configPkg.LoadConstants().constants.NetworkIntentFinalizer

var _ = Describe("NetworkIntent Controller Cleanup Table-Driven Tests", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)
	
	var (
		testEnv *testtools.TestEnvironment
	)

	var (
		ctx           context.Context
		namespaceName string
		reconciler    *NetworkIntentReconciler
		mockDeps      *testutils.MockDependencies
		constants     *configPkg.Constants
	)

	BeforeEach(func() {
		ctx = context.Background()
		constants = configPkg.LoadConstants()

		By("Creating a new isolated namespace for table-driven tests")
		namespaceName = testutils.CreateIsolatedNamespace("cleanup-table-driven")

		By("Setting up the reconciler with mock dependencies")
		mockDeps = testutils.NewMockDependenciesBuilder().
			WithLLMClient(testutils.NewMockLLMClient()).
			WithGitClient(testutils.NewMockGitClient()).
			Build()

		config := &Config{
			MaxRetries:      3,
			RetryDelay:      time.Second,
			Timeout:         30 * time.Second,
			GitRepoURL:      "https://github.com/test/deployments.git",
			GitBranch:       "main",
			GitDeployPath:   "networkintents",
			LLMProcessorURL: "http://localhost:8080",
			UseNephioPorch:  false,
		}

		var err error
		reconciler, err = NewNetworkIntentReconciler(k8sClient, testEnv.Scheme, mockDeps, config)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("Cleaning up the test namespace")
		testutils.CleanupIsolatedNamespace(namespaceName)
	})

	Context("Table-driven tests for cleanupGitOpsPackages", func() {
		type gitOpsTestCase struct {
			name                   string
			networkIntentName      string
			networkIntentNamespace string
			gitRepoURL             string
			gitDeployPath          string
			removeDirectoryError   error
			commitError            error
			expectedError          bool
			expectedErrorSubstring string
		}

		DescribeTable("cleanupGitOpsPackages scenarios",
			func(tc gitOpsTestCase) {
				By(fmt.Sprintf("Running test case: %s", tc.name))

				// Create NetworkIntent with specified properties
				networkIntent := testutils.CreateTestNetworkIntent(
					tc.networkIntentName,
					tc.networkIntentNamespace,
					"Table-driven test for GitOps cleanup",
				)

				// Update reconciler configuration if needed
				if tc.gitRepoURL != "" {
					reconciler.config.GitRepoURL = tc.gitRepoURL
				}
				if tc.gitDeployPath != "" {
					reconciler.config.GitDeployPath = tc.gitDeployPath
				}

				// Set up Git client mock expectations
				mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
				expectedPath := fmt.Sprintf("%s/%s-%s", reconciler.config.GitDeployPath, networkIntent.Namespace, networkIntent.Name)
				expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

				if tc.removeDirectoryError != nil {
					mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(tc.removeDirectoryError)
				} else {
					mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
					if tc.commitError != nil {
						mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(tc.commitError)
					} else {
						mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil)
					}
				}

				// Call the function under test
				err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

				// Verify results
				if tc.expectedError {
					Expect(err).To(HaveOccurred())
					if tc.expectedErrorSubstring != "" {
						Expect(err.Error()).To(ContainSubstring(tc.expectedErrorSubstring))
					}
				} else {
					Expect(err).NotTo(HaveOccurred())
				}

				mockGitClient.AssertExpectations(GinkgoT())
			},
			Entry("successful cleanup", gitOpsTestCase{
				name:                   "successful cleanup",
				networkIntentName:      "success-test",
				networkIntentNamespace: namespaceName,
				expectedError:          false,
			}),
			Entry("git remove directory failure", gitOpsTestCase{
				name:                   "git remove directory failure",
				networkIntentName:      "remove-fail-test",
				networkIntentNamespace: namespaceName,
				removeDirectoryError:   errors.New("failed to remove directory"),
				expectedError:          true,
				expectedErrorSubstring: "failed to remove GitOps package directory",
			}),
			Entry("git commit failure", gitOpsTestCase{
				name:                   "git commit failure",
				networkIntentName:      "commit-fail-test",
				networkIntentNamespace: namespaceName,
				commitError:            errors.New("failed to commit changes"),
				expectedError:          true,
				expectedErrorSubstring: "failed to commit package removal",
			}),
			Entry("authentication failure", gitOpsTestCase{
				name:                   "authentication failure",
				networkIntentName:      "auth-fail-test",
				networkIntentNamespace: namespaceName,
				removeDirectoryError:   ErrGitAuthenticationFailed,
				expectedError:          true,
				expectedErrorSubstring: "SSH key authentication failed",
			}),
			Entry("network timeout", gitOpsTestCase{
				name:                   "network timeout",
				networkIntentName:      "timeout-test",
				networkIntentNamespace: namespaceName,
				commitError:            ErrGitNetworkTimeout,
				expectedError:          true,
				expectedErrorSubstring: "network timeout",
			}),
			Entry("repository corruption", gitOpsTestCase{
				name:                   "repository corruption",
				networkIntentName:      "corruption-test",
				networkIntentNamespace: namespaceName,
				removeDirectoryError:   ErrGitRepositoryCorrupted,
				expectedError:          true,
				expectedErrorSubstring: "repository is corrupted",
			}),
			Entry("directory not found", gitOpsTestCase{
				name:                   "directory not found",
				networkIntentName:      "not-found-test",
				networkIntentNamespace: namespaceName,
				removeDirectoryError:   ErrGitDirectoryNotFound,
				expectedError:          true,
				expectedErrorSubstring: "directory not found",
			}),
			Entry("custom deploy path", gitOpsTestCase{
				name:                   "custom deploy path",
				networkIntentName:      "custom-path-test",
				networkIntentNamespace: namespaceName,
				gitDeployPath:          "custom/deploy/path",
				expectedError:          false,
			}),
			Entry("long resource names", gitOpsTestCase{
				name:                   "long resource names",
				networkIntentName:      "very-long-network-intent-name-that-tests-boundary-conditions",
				networkIntentNamespace: namespaceName,
				expectedError:          false,
			}),
		)
	})

	Context("Table-driven tests for cleanupGeneratedResources", func() {
		type resourceCleanupTestCase struct {
			name           string
			setupResources []client.Object
			expectedError  bool
			errorSubstring string
		}

		DescribeTable("cleanupGeneratedResources scenarios",
			func(tc resourceCleanupTestCase) {
				By(fmt.Sprintf("Running test case: %s", tc.name))

				networkIntent := testutils.CreateTestNetworkIntent(
					testutils.GetUniqueName("resource-test"),
					namespaceName,
					"Table-driven test for resource cleanup",
				)

				// Create test resources if specified
				for _, resource := range tc.setupResources {
					// Ensure resource is in the correct namespace
					resource.SetNamespace(networkIntent.Namespace)
					Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				}

				// Call the function under test
				err := reconciler.cleanupGeneratedResources(ctx, networkIntent)

				// Verify results
				if tc.expectedError {
					Expect(err).To(HaveOccurred())
					if tc.errorSubstring != "" {
						Expect(err.Error()).To(ContainSubstring(tc.errorSubstring))
					}
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			},
			Entry("no resources to clean", resourceCleanupTestCase{
				name:          "no resources to clean",
				expectedError: false,
			}),
			Entry("cleanup ConfigMaps", resourceCleanupTestCase{
				name: "cleanup ConfigMaps",
				setupResources: []client.Object{
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-configmap-1",
							Labels: map[string]string{
								"app.kubernetes.io/name":       "networkintent",
								"app.kubernetes.io/managed-by": "nephoran-intent-operator",
							},
						},
						Data: map[string]string{"key": "value"},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-configmap-2",
							Labels: map[string]string{
								"app.kubernetes.io/name":       "networkintent",
								"app.kubernetes.io/managed-by": "nephoran-intent-operator",
							},
						},
						Data: map[string]string{"key": "value"},
					},
				},
				expectedError: false,
			}),
			Entry("cleanup Secrets", resourceCleanupTestCase{
				name: "cleanup Secrets",
				setupResources: []client.Object{
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-secret-1",
							Labels: map[string]string{
								"app.kubernetes.io/name":       "networkintent",
								"app.kubernetes.io/managed-by": "nephoran-intent-operator",
							},
						},
						StringData: map[string]string{"secret": "value"},
					},
				},
				expectedError: false,
			}),
			Entry("mixed resources", resourceCleanupTestCase{
				name: "mixed resources",
				setupResources: []client.Object{
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mixed-configmap",
							Labels: map[string]string{
								"app.kubernetes.io/name":       "networkintent",
								"app.kubernetes.io/managed-by": "nephoran-intent-operator",
							},
						},
						Data: map[string]string{"key": "value"},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: "mixed-secret",
							Labels: map[string]string{
								"app.kubernetes.io/name":       "networkintent",
								"app.kubernetes.io/managed-by": "nephoran-intent-operator",
							},
						},
						StringData: map[string]string{"secret": "value"},
					},
				},
				expectedError: false,
			}),
			Entry("resources with partial labels", resourceCleanupTestCase{
				name: "resources with partial labels",
				setupResources: []client.Object{
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "partial-labels-configmap",
							Labels: map[string]string{
								"app.kubernetes.io/name": "networkintent",
								// Missing managed-by label
							},
						},
						Data: map[string]string{"key": "value"},
					},
				},
				expectedError: false,
			}),
			Entry("resources without matching labels", resourceCleanupTestCase{
				name: "resources without matching labels",
				setupResources: []client.Object{
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name: "no-matching-labels",
							Labels: map[string]string{
								"different": "label",
							},
						},
						Data: map[string]string{"key": "value"},
					},
				},
				expectedError: false,
			}),
		)
	})

	Context("Table-driven tests for handleDeletion", func() {
		type deletionTestCase struct {
			name                   string
			finalizers             []string
			gitCleanupError        error
			resourceCleanupError   error
			expectedRequeue        bool
			expectedError          bool
			expectedErrorSubstring string
		}

		DescribeTable("handleDeletion scenarios",
			func(tc deletionTestCase) {
				By(fmt.Sprintf("Running test case: %s", tc.name))

				networkIntent := testutils.CreateTestNetworkIntent(
					testutils.GetUniqueName("deletion-test"),
					namespaceName,
					"Table-driven test for deletion handling",
				)
				networkIntent.Finalizers = tc.finalizers
				networkIntent.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

				// Set up Git client mock based on test case
				mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
				if tc.gitCleanupError != nil {
					expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
					expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
					mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(tc.gitCleanupError)
				} else {
					// Only set up successful expectations if we have the NetworkIntent finalizer
					if containsFinalizer(tc.finalizers, constants.NetworkIntentFinalizer) {
						expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
						expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)
						mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
						mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(nil)
					}
				}

				// Call the function under test
				result, err := reconciler.handleDeletion(ctx, networkIntent)

				// Verify results
				if tc.expectedError {
					Expect(err).To(HaveOccurred())
					if tc.expectedErrorSubstring != "" {
						Expect(err.Error()).To(ContainSubstring(tc.expectedErrorSubstring))
					}
				} else {
					Expect(err).NotTo(HaveOccurred())
				}

				if tc.expectedRequeue {
					Expect(result.RequeueAfter).To(BeNumerically(">", 0))
				} else {
					Expect(result.Requeue).To(BeFalse())
					Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
				}

				mockGitClient.AssertExpectations(GinkgoT())
			},
			Entry("successful deletion with finalizer", deletionTestCase{
				name:            "successful deletion with finalizer",
				finalizers:      []string{constants.NetworkIntentFinalizer},
				expectedRequeue: false,
				expectedError:   false,
			}),
			Entry("deletion without finalizers", deletionTestCase{
				name:            "deletion without finalizers",
				finalizers:      []string{},
				expectedRequeue: false,
				expectedError:   false,
			}),
			Entry("deletion with multiple finalizers", deletionTestCase{
				name: "deletion with multiple finalizers",
				finalizers: []string{
					constants.NetworkIntentFinalizer,
					"other.controller/finalizer",
					"third.controller/finalizer",
				},
				expectedRequeue: false,
				expectedError:   false,
			}),
			Entry("git cleanup failure", deletionTestCase{
				name:                   "git cleanup failure",
				finalizers:             []string{constants.NetworkIntentFinalizer},
				gitCleanupError:        errors.New("git cleanup failed"),
				expectedRequeue:        true,
				expectedError:          true,
				expectedErrorSubstring: "git cleanup failed",
			}),
			Entry("git authentication failure", deletionTestCase{
				name:                   "git authentication failure",
				finalizers:             []string{constants.NetworkIntentFinalizer},
				gitCleanupError:        ErrGitAuthenticationFailed,
				expectedRequeue:        true,
				expectedError:          true,
				expectedErrorSubstring: "SSH key authentication failed",
			}),
			Entry("git network timeout", deletionTestCase{
				name:                   "git network timeout",
				finalizers:             []string{constants.NetworkIntentFinalizer},
				gitCleanupError:        ErrGitNetworkTimeout,
				expectedRequeue:        true,
				expectedError:          true,
				expectedErrorSubstring: "network timeout",
			}),
			Entry("git repository corruption", deletionTestCase{
				name:                   "git repository corruption",
				finalizers:             []string{constants.NetworkIntentFinalizer},
				gitCleanupError:        ErrGitRepositoryCorrupted,
				expectedRequeue:        true,
				expectedError:          true,
				expectedErrorSubstring: "repository is corrupted",
			}),
		)
	})

	Context("Table-driven tests for Git client error scenarios", func() {
		type gitErrorTestCase struct {
			name            string
			error           error
			operation       string
			shouldPropagate bool
		}

		DescribeTable("Git client error handling",
			func(tc gitErrorTestCase) {
				By(fmt.Sprintf("Running git error test: %s", tc.name))

				networkIntent := testutils.CreateTestNetworkIntent(
					testutils.GetUniqueName("git-error-test"),
					namespaceName,
					"Table-driven test for Git errors",
				)

				mockGitClient := mockDeps.GetGitClient().(*testutils.MockGitClient)
				expectedPath := fmt.Sprintf("networkintents/%s-%s", networkIntent.Namespace, networkIntent.Name)
				expectedMessage := fmt.Sprintf("Remove NetworkIntent package: %s-%s", networkIntent.Namespace, networkIntent.Name)

				switch tc.operation {
				case "RemoveDirectory":
					mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(tc.error)
				case "CommitAndPushChanges":
					mockGitClient.On("RemoveDirectory", expectedPath, expectedMessage).Return(nil)
					mockGitClient.On("CommitAndPushChanges", expectedMessage).Return(tc.error)
				}

				err := reconciler.cleanupGitOpsPackages(ctx, networkIntent, mockGitClient)

				if tc.shouldPropagate {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(tc.error.Error()))
				} else {
					Expect(err).NotTo(HaveOccurred())
				}

				mockGitClient.AssertExpectations(GinkgoT())
			},
			Entry("authentication failure on remove", gitErrorTestCase{
				name:            "authentication failure on remove",
				error:           ErrGitAuthenticationFailed,
				operation:       "RemoveDirectory",
				shouldPropagate: true,
			}),
			Entry("network timeout on commit", gitErrorTestCase{
				name:            "network timeout on commit",
				error:           ErrGitNetworkTimeout,
				operation:       "CommitAndPushChanges",
				shouldPropagate: true,
			}),
			Entry("repository corruption", gitErrorTestCase{
				name:            "repository corruption",
				error:           ErrGitRepositoryCorrupted,
				operation:       "RemoveDirectory",
				shouldPropagate: true,
			}),
			Entry("directory not found", gitErrorTestCase{
				name:            "directory not found",
				error:           ErrGitDirectoryNotFound,
				operation:       "RemoveDirectory",
				shouldPropagate: true,
			}),
			Entry("push rejected", gitErrorTestCase{
				name:            "push rejected",
				error:           ErrGitPushRejected,
				operation:       "CommitAndPushChanges",
				shouldPropagate: true,
			}),
			Entry("no changes to commit", gitErrorTestCase{
				name:            "no changes to commit",
				error:           ErrGitNoChangesToCommit,
				operation:       "CommitAndPushChanges",
				shouldPropagate: true,
			}),
		)
	})

	Context("Table-driven tests for label selector edge cases", func() {
		type labelSelectorTestCase struct {
			name     string
			labels   map[string]string
			expected []string // Expected substrings in the selector
		}

		DescribeTable("createLabelSelector scenarios",
			func(tc labelSelectorTestCase) {
				By(fmt.Sprintf("Running label selector test: %s", tc.name))

				selector := createLabelSelector(tc.labels)

				Expect(selector).NotTo(BeEmpty())
				for _, expectedSubstring := range tc.expected {
					Expect(selector).To(ContainSubstring(expectedSubstring),
						"Selector should contain '%s': %s", expectedSubstring, selector)
				}
			},
			Entry("standard labels", labelSelectorTestCase{
				name: "standard labels",
				labels: map[string]string{
					"app.kubernetes.io/name":       "networkintent",
					"app.kubernetes.io/managed-by": "nephoran-intent-operator",
				},
				expected: []string{"app.kubernetes.io/name=networkintent", "nephoran-intent-operator"},
			}),
			Entry("single label", labelSelectorTestCase{
				name: "single label",
				labels: map[string]string{
					"test": "value",
				},
				expected: []string{"test=value"},
			}),
			Entry("empty labels", labelSelectorTestCase{
				name:     "empty labels",
				labels:   map[string]string{},
				expected: []string{}, // No specific expectations for empty
			}),
			Entry("special characters", labelSelectorTestCase{
				name: "special characters",
				labels: map[string]string{
					"nephoran.com/intent-name": "test-intent-123",
					"example.com/type":         "network-config",
				},
				expected: []string{"nephoran.com/intent-name=test-intent-123", "example.com/type=network-config"},
			}),
			Entry("numeric values", labelSelectorTestCase{
				name: "numeric values",
				labels: map[string]string{
					"version": "1",
					"replica": "3",
				},
				expected: []string{"version=1", "replica=3"},
			}),
		)
	})

	Context("Table-driven tests for finalizer management", func() {
		type finalizerTestCase struct {
			name              string
			initialFinalizers []string
			finalizerToCheck  string
			finalizerToRemove string
			expectedContains  bool
			expectedRemaining []string
		}

		DescribeTable("finalizer management scenarios",
			func(tc finalizerTestCase) {
				By(fmt.Sprintf("Running finalizer test: %s", tc.name))

				if tc.finalizerToCheck != "" {
					result := containsFinalizer(tc.initialFinalizers, tc.finalizerToCheck)
					Expect(result).To(Equal(tc.expectedContains))
				}

				if tc.finalizerToRemove != "" {
					result := removeFinalizer(tc.initialFinalizers, tc.finalizerToRemove)
					Expect(result).To(Equal(tc.expectedRemaining))
				}
			},
			Entry("contains existing finalizer", finalizerTestCase{
				name:              "contains existing finalizer",
				initialFinalizers: []string{constants.NetworkIntentFinalizer, "other.finalizer"},
				finalizerToCheck:  constants.NetworkIntentFinalizer,
				expectedContains:  true,
			}),
			Entry("does not contain non-existent finalizer", finalizerTestCase{
				name:              "does not contain non-existent finalizer",
				initialFinalizers: []string{"other.finalizer"},
				finalizerToCheck:  constants.NetworkIntentFinalizer,
				expectedContains:  false,
			}),
			Entry("empty finalizers list", finalizerTestCase{
				name:              "empty finalizers list",
				initialFinalizers: []string{},
				finalizerToCheck:  constants.NetworkIntentFinalizer,
				expectedContains:  false,
			}),
			Entry("remove existing finalizer", finalizerTestCase{
				name:              "remove existing finalizer",
				initialFinalizers: []string{constants.NetworkIntentFinalizer, "other.finalizer"},
				finalizerToRemove: constants.NetworkIntentFinalizer,
				expectedRemaining: []string{"other.finalizer"},
			}),
			Entry("remove non-existent finalizer", finalizerTestCase{
				name:              "remove non-existent finalizer",
				initialFinalizers: []string{"other.finalizer"},
				finalizerToRemove: constants.NetworkIntentFinalizer,
				expectedRemaining: []string{"other.finalizer"},
			}),
			Entry("remove all finalizers", finalizerTestCase{
				name:              "remove all finalizers",
				initialFinalizers: []string{constants.NetworkIntentFinalizer},
				finalizerToRemove: constants.NetworkIntentFinalizer,
				expectedRemaining: []string{},
			}),
			Entry("remove with duplicates", finalizerTestCase{
				name:              "remove with duplicates",
				initialFinalizers: []string{constants.NetworkIntentFinalizer, "other", constants.NetworkIntentFinalizer},
				finalizerToRemove: constants.NetworkIntentFinalizer,
				expectedRemaining: []string{"other"},
			}),
		)
	})
})
