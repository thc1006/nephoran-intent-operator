package e2e

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	// k8sClient is initialized in main_test.go BeforeSuite and shared across all test files
	k8sClient client.Client
)

// BeforeSuite setup is handled in main_test.go for E2E testing

// AfterSuite cleanup is handled in main_test.go for E2E testing
