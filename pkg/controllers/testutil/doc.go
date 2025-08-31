// Package testutil provides shared testing utilities for the controllers package.

//

// This package consolidates common test helpers, fake implementations, and utility.

// functions to avoid duplication across test files and enable clean, maintainable.

// test code.

//

// Key components:.

//   - FakeE2Manager: Mock implementation of E2ManagerInterface for testing.

//   - Condition helpers: Functions for working with Kubernetes conditions.

//   - Test helpers: Utilities for creating test resources and managing namespaces.

//   - Constants: Common test values and timeouts.

//

// Usage:.

//

//	import "github.com/thc1006/nephoran-intent-operator/pkg/controllers/testutil"

//

//	// Create a fake E2Manager

//	e2Manager := testutil.NewFakeE2Manager()

//

//	// Check conditions

//	condition := testutil.GetCondition(resource.Status.Conditions, "Ready")

//	if testutil.IsConditionTrue(resource.Status.Conditions, "Ready") {

//		// Handle ready condition

//	}

//

//	// Create test resources

//	e2nodeSet := testutil.CreateTestE2NodeSet("test", "default", 3)

package testutil
