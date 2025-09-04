package integration_tests_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// Package-level variables for O2 tests
var (
	k8sClient client.Client
	testEnv   *envtest.Environment
)

// CreateO2TestNamespace creates a test namespace for O2 integration tests
func CreateO2TestNamespace() *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "o2-integration-test-",
		},
	}
	Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed())

	DeferCleanup(func() {
		k8sClient.Delete(context.Background(), namespace)
	})

	return namespace
}