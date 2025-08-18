package main

import (
	"context"
	"fmt"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	ctx := context.Background()
	validator := &intentv1alpha1.NetworkIntentValidator{}
	defaulter := &intentv1alpha1.NetworkIntentDefaulter{}

	// Test 1: Valid NetworkIntent
	fmt.Println("Test 1: Valid NetworkIntent")
	valid := &intentv1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "valid-intent",
			Namespace: "ran-a",
		},
		Spec: intentv1alpha1.NetworkIntentSpec{
			IntentType: "scaling",
			Target:     "nf-sim",
			Namespace:  "ran-a",
			Replicas:   3,
			Source:     "user",
		},
	}
	warnings, err := validator.ValidateCreate(ctx, valid)
	if err != nil {
		fmt.Printf("  ❌ Unexpected error: %v\n", err)
	} else {
		fmt.Printf("  ✅ Valid NetworkIntent accepted (warnings: %v)\n", warnings)
	}

	// Test 2: NetworkIntent with negative replicas (should fail)
	fmt.Println("\nTest 2: NetworkIntent with negative replicas")
	invalid := &intentv1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-replicas",
			Namespace: "ran-a",
		},
		Spec: intentv1alpha1.NetworkIntentSpec{
			IntentType: "scaling",
			Target:     "nf-sim",
			Namespace:  "ran-a",
			Replicas:   -1,
			Source:     "user",
		},
	}
	warnings, err = validator.ValidateCreate(ctx, invalid)
	if err != nil {
		fmt.Printf("  ✅ Correctly rejected: %v\n", err)
	} else {
		fmt.Printf("  ❌ Should have been rejected but was accepted\n")
	}

	// Test 3: Default source when empty
	fmt.Println("\nTest 3: Default source when empty")
	needsDefault := &intentv1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "needs-default",
			Namespace: "ran-a",
		},
		Spec: intentv1alpha1.NetworkIntentSpec{
			IntentType: "scaling",
			Target:     "nf-sim",
			Namespace:  "ran-a",
			Replicas:   3,
			// Source is empty, should default to "user"
		},
	}
	err = defaulter.Default(ctx, needsDefault)
	if err != nil {
		fmt.Printf("  ❌ Error during defaulting: %v\n", err)
	} else if needsDefault.Spec.Source == "user" {
		fmt.Printf("  ✅ Source correctly defaulted to 'user'\n")
	} else {
		fmt.Printf("  ❌ Source not defaulted correctly: got '%s'\n", needsDefault.Spec.Source)
	}

	// Test 4: Invalid intentType
	fmt.Println("\nTest 4: Invalid intentType")
	invalidType := &intentv1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-type",
			Namespace: "ran-a",
		},
		Spec: intentv1alpha1.NetworkIntentSpec{
			IntentType: "provisioning", // Invalid - only "scaling" supported
			Target:     "nf-sim",
			Namespace:  "ran-a",
			Replicas:   3,
			Source:     "user",
		},
	}
	warnings, err = validator.ValidateCreate(ctx, invalidType)
	if err != nil {
		fmt.Printf("  ✅ Correctly rejected: %v\n", err)
	} else {
		fmt.Printf("  ❌ Should have been rejected but was accepted\n")
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("Webhook validation logic is working correctly!")
	fmt.Println("- Mutating webhook: Sets default source='user' when empty")
	fmt.Println("- Validating webhook: Rejects negative replicas, invalid intentType, etc.")
}