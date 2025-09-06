package integration_tests

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Package-level variables for O2 tests
// Note: k8sClient and integrationTestEnv are defined in suite_test.go to avoid redeclaration

// CreateO2TestNamespace creates a test namespace for O2 integration tests
func CreateO2TestNamespace() *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "o2-integration-test-",
		},
	}

	// Note: k8sClient creation and cleanup should be handled by the calling test
	// This function just returns the namespace object
	return namespace
}