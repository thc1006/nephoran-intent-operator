package porch_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Porch Resilience", func() {
	Context("When creating packages under stress", func() {
		It("should handle rapid package creation", func() {
			ctx := context.Background()
			packageCount := 50

			for i := 0; i < packageCount; i++ {
				pkg := &Package{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("resilience-test-pkg-%d", i),
						Namespace: "default",
					},
					Spec: PackageSpec{
						Repository: "resilience-test",
					},
				}

				err := k8sClient.Create(ctx, pkg)
				Expect(err).NotTo(HaveOccurred(), "Should successfully create package %d", i)
			}
		})

		It("should recover from temporary failures", func() {
			ctx := context.Background()
			
			// Create a package that might fail initially
			pkg := &Package{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "recovery-test-pkg",
					Namespace: "default",
				},
				Spec: PackageSpec{
					Repository: "recovery-test",
				},
			}

			// Attempt creation with retry logic
			var err error
			for attempt := 0; attempt < 3; attempt++ {
				err = k8sClient.Create(ctx, pkg)
				if err == nil {
					break
				}
				
				// Wait before retry
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			}
			
			Expect(err).NotTo(HaveOccurred(), "Should eventually create package after retries")
		})
	})

	Context("When handling concurrent operations", func() {
		It("should maintain consistency under concurrent load", func() {
			ctx := context.Background()
			concurrentOperations := 10
			
			// Channel to collect results
			results := make(chan error, concurrentOperations)
			
			// Launch concurrent package creations
			for i := 0; i < concurrentOperations; i++ {
				go func(index int) {
					pkg := &Package{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("concurrent-test-pkg-%d", index),
							Namespace: "default",
						},
						Spec: PackageSpec{
							Repository: "concurrent-test",
						},
					}
					
					err := k8sClient.Create(ctx, pkg)
					results <- err
				}(i)
			}
			
			// Collect all results
			var errors []error
			for i := 0; i < concurrentOperations; i++ {
				if err := <-results; err != nil {
					errors = append(errors, err)
				}
			}
			
			// Should have minimal or no errors
			Expect(len(errors)).To(BeNumerically("<=", 2), "Should have few or no errors in concurrent operations")
		})
	})
})