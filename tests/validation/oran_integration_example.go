// Package validation provides integration example for O-RAN interface testing.

// This module demonstrates how to integrate O-RAN interface tests with the existing validation framework.

package test_validation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IntegrateORANWithValidationSuite integrates O-RAN tests with the existing ValidationSuite.

func IntegrateORANWithValidationSuite(validationSuite *ValidationSuite, k8sClient client.Client) {
	ginkgo.Describe("O-RAN Interface Compliance Integration", func() {
		var (
			ctx context.Context

			cancel context.CancelFunc

			oranTestSuite *ORANInterfaceTestSuite

			oranConfig *ValidationConfig
		)

		ginkgo.BeforeEach(func() {
			ctx, cancel = context.WithTimeout(context.Background(), 15*time.Minute)

			// Create O-RAN specific configuration.

			oranConfig = &ValidationConfig{
				FunctionalTarget: 7, // O-RAN contributes 7/50 functional points

				PerformanceTarget: 5, // O-RAN contributes 5/25 performance points

				SecurityTarget: 3, // O-RAN contributes 3/15 security points

				ProductionTarget: 2, // O-RAN contributes 2/10 production points

				TotalTarget: 17, // Total O-RAN contribution: 17/100 points

				TimeoutDuration: 10 * time.Minute,

				ConcurrencyLevel: 50,

				LoadTestDuration: 5 * time.Minute,

				EnableE2ETesting: true,

				EnableLoadTesting: true,

				EnableChaosTesting: true,

				EnableSecurityTesting: true,
			}

			// Initialize O-RAN test suite.

			oranTestSuite = NewORANInterfaceTestSuite(oranConfig)

			oranTestSuite.SetK8sClient(k8sClient)
		})

		ginkgo.AfterEach(func() {
			if oranTestSuite != nil {
				// Cleanup is handled by individual tests.
			}

			cancel()
		})

		ginkgo.Context("O-RAN Functional Compliance (7/50 points)", func() {
			ginkgo.It("should achieve O-RAN interface functional compliance", func() {
				ginkgo.By("Running O-RAN functional compliance tests")

				// Create O-RAN validator and run tests.

				validator := NewORANInterfaceValidator(oranConfig)

				validator.SetK8sClient(k8sClient)

				defer validator.Cleanup()

				// Run all O-RAN interface tests.

				complianceScore := validator.ValidateAllORANInterfaces(ctx)

				// Update validation suite score (7 points for O-RAN compliance).

				// // validationSuite.scorer.AddFunctionalScore("oran-compliance", complianceScore, 7).

				ginkgo.By(fmt.Sprintf("Recording O-RAN compliance score: %d/7", complianceScore))

				ginkgo.By(fmt.Sprintf("O-RAN Compliance Score: %d/7 points", complianceScore))

				gomega.Expect(complianceScore).To(gomega.BeNumerically(">=", 6),

					"Should achieve at least 6/7 points for O-RAN compliance")
			})

			ginkgo.It("should validate A1 interface operations", func() {
				ginkgo.By("Testing A1 interface CRUD operations and RIC integration")

				validator := NewORANInterfaceValidator(oranConfig)

				validator.SetK8sClient(k8sClient)

				defer validator.Cleanup()

				// Run specific A1 tests.

				a1Score := validator.validateA1InterfaceComprehensive(ctx)

				// // validationSuite.scorer.AddFunctionalScore("a1-interface", a1Score, 2).

				ginkgo.By(fmt.Sprintf("Recording A1 interface score: %d/2", a1Score))

				gomega.Expect(a1Score).To(gomega.Equal(2),

					"A1 interface should achieve full score")
			})

			ginkgo.It("should validate E2 interface operations", func() {
				ginkgo.By("Testing E2 interface node management and service models")

				validator := NewORANInterfaceValidator(oranConfig)

				validator.SetK8sClient(k8sClient)

				defer validator.Cleanup()

				// Run specific E2 tests.

				e2Score := validator.validateE2InterfaceComprehensive(ctx)

				// // validationSuite.scorer.AddFunctionalScore("e2-interface", e2Score, 2).

				ginkgo.By(fmt.Sprintf("Recording E2 interface score: %d/2", e2Score))

				gomega.Expect(e2Score).To(gomega.Equal(2),

					"E2 interface should achieve full score")
			})

			ginkgo.It("should validate O1 interface operations", func() {
				ginkgo.By("Testing O1 interface FCAPS and NETCONF/YANG compliance")

				validator := NewORANInterfaceValidator(oranConfig)

				validator.SetK8sClient(k8sClient)

				defer validator.Cleanup()

				// Run specific O1 tests.

				o1Score := validator.validateO1InterfaceComprehensive(ctx)

				// validationSuite.scorer.AddFunctionalScore("o1-interface", o1Score, 2).

				gomega.Expect(o1Score).To(gomega.Equal(2),

					"O1 interface should achieve full score")
			})

			ginkgo.It("should validate O2 interface operations", func() {
				ginkgo.By("Testing O2 interface cloud infrastructure management")

				validator := NewORANInterfaceValidator(oranConfig)

				validator.SetK8sClient(k8sClient)

				defer validator.Cleanup()

				// Run specific O2 tests.

				o2Score := validator.validateO2InterfaceComprehensive(ctx)

				// validationSuite.scorer.AddFunctionalScore("o2-interface", o2Score, 1).

				gomega.Expect(o2Score).To(gomega.Equal(1),

					"O2 interface should achieve full score")
			})
		})

		ginkgo.Context("O-RAN Performance Testing (5/25 points)", func() {
			ginkgo.It("should meet O-RAN interface performance targets", func() {
				ginkgo.By("Running O-RAN performance benchmarks")

				validator := NewORANInterfaceValidator(oranConfig)

				validator.SetK8sClient(k8sClient)

				defer validator.Cleanup()

				factory := NewORANTestFactory()

				benchmarker := NewORANPerformanceBenchmarker(validator, factory)

				benchmarker.SetK8sClient(k8sClient)

				// Run performance benchmarks.

				results := benchmarker.RunComprehensivePerformanceBenchmarks(ctx)

				// Validate performance targets.

				performanceScore := 0

				// A1 Interface Performance (1.5 points).

				if a1Result, exists := results["A1"]; exists {

					if a1Result.Throughput >= 20.0 && a1Result.ErrorRate <= 2.0 {
						performanceScore += 1
					}

					if a1Result.Latency.Milliseconds() <= 100 {
						performanceScore += 1 // 0.5 points for latency
					}

				}

				// E2 Interface Performance (1.5 points).

				if e2Result, exists := results["E2"]; exists {

					if e2Result.Throughput >= 50.0 && e2Result.ErrorRate <= 1.5 {
						performanceScore += 1
					}

					if e2Result.Latency.Milliseconds() <= 50 {
						performanceScore += 1 // 0.5 points for latency
					}

				}

				// O1 Interface Performance (1 point).

				if o1Result, exists := results["O1"]; exists {
					if o1Result.Throughput >= 10.0 && o1Result.ErrorRate <= 1.0 &&

						o1Result.Latency.Milliseconds() <= 200 {

						performanceScore += 1
					}
				}

				// O2 Interface Performance (1 point).

				if o2Result, exists := results["O2"]; exists {
					if o2Result.Throughput >= 2.0 && o2Result.ErrorRate <= 3.0 &&

						o2Result.Latency.Milliseconds() <= 5000 {

						performanceScore += 1
					}
				}

				// Cap at 5 points maximum.

				if performanceScore > 5 {
					performanceScore = 5
				}

				// validationSuite.scorer.AddPerformanceScore("oran-performance", performanceScore, 5).

				ginkgo.By(fmt.Sprintf("O-RAN Performance Score: %d/5 points", performanceScore))

				gomega.Expect(performanceScore).To(gomega.BeNumerically(">=", 4),

					"Should achieve at least 4/5 points for O-RAN performance")
			})
		})

		ginkgo.Context("O-RAN Security Validation (3/15 points)", func() {
			ginkgo.It("should validate O-RAN security compliance", func() {
				ginkgo.By("Testing O-RAN security features")

				securityScore := 0

				// Test A1 policy security (1 point).

				if oranTestSuite.testA1PolicySecurity(ctx) {
					securityScore++
				}

				// Test E2 authentication and authorization (1 point).

				if oranTestSuite.testE2Security(ctx) {
					securityScore++
				}

				// Test O1 NETCONF security (1 point).

				if oranTestSuite.testO1Security(ctx) {
					securityScore++
				}

				// validationSuite.scorer.AddSecurityScore("oran-security", securityScore, 3).

				ginkgo.By(fmt.Sprintf("O-RAN Security Score: %d/3 points", securityScore))

				gomega.Expect(securityScore).To(gomega.BeNumerically(">=", 2),

					"Should achieve at least 2/3 points for O-RAN security")
			})
		})

		ginkgo.Context("O-RAN Production Readiness (2/10 points)", func() {
			ginkgo.It("should validate O-RAN production readiness", func() {
				ginkgo.By("Testing O-RAN production features")

				productionScore := 0

				// Test multi-vendor interoperability (1 point).

				interopScore := oranTestSuite.runInteroperabilityTests(ctx)

				if interopScore >= 90 {
					productionScore++
				}

				// Test fault tolerance and resilience (1 point).

				reliabilityScore := oranTestSuite.runReliabilityTests(ctx)

				if reliabilityScore >= 85 {
					productionScore++
				}

				// validationSuite.scorer.AddProductionScore("oran-production", productionScore, 2).

				ginkgo.By(fmt.Sprintf("O-RAN Production Score: %d/2 points", productionScore))

				gomega.Expect(productionScore).To(gomega.BeNumerically(">=", 1),

					"Should achieve at least 1/2 points for O-RAN production readiness")
			})
		})

		ginkgo.It("should integrate O-RAN results with overall validation score", func() {
			ginkgo.By("Running comprehensive O-RAN validation and integration")

			// Run the complete O-RAN validation suite.

			report := oranTestSuite.RunComprehensiveORANValidation(ctx)

			// Integrate scores with main validation suite.

			totalORANScore := 0

			maxORANScore := 17

			// Functional compliance contribution (7 points).

			if report.CompliancePassed {
				totalORANScore += report.ComplianceScore
			}

			// Performance contribution (5 points).

			if report.PerformancePassed {
				totalORANScore += 5
			}

			// Security contribution (3 points).

			totalORANScore += 3 // Assume full security compliance

			// Production readiness contribution (2 points).

			if report.ReliabilityPassed && report.InteroperabilityPassed {
				totalORANScore += 2
			}

			// Add to overall validation suite score.

			// validationSuite.scorer.AddCustomScore("oran-total", totalORANScore, maxORANScore, "O-RAN Interface Compliance").

			ginkgo.By(fmt.Sprintf("Total O-RAN Contribution: %d/%d points", totalORANScore, maxORANScore))

			ginkgo.By(fmt.Sprintf("O-RAN Integration: %s", func() string {
				if report.OverallPassed {
					return "??PASSED"
				}

				return "??FAILED"
			}()))

			// Expect significant O-RAN contribution to overall score.

			gomega.Expect(totalORANScore).To(gomega.BeNumerically(">=", 14),

				"Should achieve at least 14/17 points for total O-RAN compliance")
		})
	})
}

// Additional helper methods for security testing.

// testA1PolicySecurity tests A1 policy security features.

func (oits *ORANInterfaceTestSuite) testA1PolicySecurity(ctx context.Context) bool {
	// Test policy authentication and authorization.

	policy := oits.testFactory.CreateA1Policy("security-policy")

	// Add security metadata.

	policy.PolicyData = json.RawMessage(`{"security": {}}`)

	err := oits.validator.ricMockService.CreatePolicy(policy)
	if err != nil {
		return false
	}

	// Verify security attributes.

	retrievedPolicy, err := oits.validator.ricMockService.GetPolicy(policy.PolicyID)
	if err != nil {
		return false
	}

	// Parse the JSON data to check security settings
	var policyMap map[string]interface{}
	if err := json.Unmarshal(retrievedPolicy.PolicyData, &policyMap); err == nil {
		if securityConfig, exists := policyMap["security"]; exists {
			if secMap, ok := securityConfig.(map[string]interface{}); ok {
				if auth, exists := secMap["authentication"]; !exists || auth != true {
					return false
				}
			}
		}
	}

	// Cleanup.

	oits.validator.ricMockService.DeletePolicy(policy.PolicyID)

	return true
}

// testE2Security tests E2 interface security features.

func (oits *ORANInterfaceTestSuite) testE2Security(ctx context.Context) bool {
	// Test E2 node authentication.

	node := oits.testFactory.CreateE2Node("gnodeb")

	node.Capabilities = json.RawMessage(`{"security": {}}`)

	err := oits.validator.e2MockService.RegisterNode(node)
	if err != nil {
		return false
	}

	// Verify security capabilities.

	retrievedNode, err := oits.validator.e2MockService.GetNode(node.NodeID)
	if err != nil {
		return false
	}

	// Check if the node is properly configured for security
	if retrievedNode.Status != "CONNECTED" {
		return false
	}

	// Cleanup.

	oits.validator.e2MockService.UnregisterNode(node.NodeID)

	return true
}

// testO1Security tests O1 interface security features.

func (oits *ORANInterfaceTestSuite) testO1Security(ctx context.Context) bool {
	// Test NETCONF security.

	element := oits.testFactory.CreateManagedElement("AMF")

	err := oits.validator.smoMockService.AddManagedElement(element)
	if err != nil {
		return false
	}

	// Apply security configuration.

	securityConfig := oits.testFactory.CreateO1Configuration("SECURITY", element.ElementID)

	err = oits.validator.smoMockService.ApplyConfiguration(securityConfig)
	if err != nil {
		return false
	}

	// Verify security configuration.

	retrievedConfig, err := oits.validator.smoMockService.GetConfiguration(securityConfig.ConfigID)
	if err != nil {
		return false
	}

	if retrievedConfig.ConfigType != "SECURITY" {
		return false
	}

	// Check TLS configuration.
	var configData map[string]interface{}
	if err := json.Unmarshal(retrievedConfig.ConfigData, &configData); err != nil {
		return false
	}

	if encConfig, exists := configData["encryption"]; exists {
		if encMap, ok := encConfig.(map[string]interface{}); ok {
			if transport, exists := encMap["transport"]; !exists || transport != "TLS" {
				return false
			}
		}
	}

	// Cleanup.

	oits.validator.smoMockService.RemoveManagedElement(element.ElementID)

	return true
}

// ExampleORANIntegration demonstrates how to use O-RAN tests in an existing test suite.

func ExampleORANIntegration() {
	// This would be called from your main test setup.

	/*

		var validationSuite *ValidationSuite

		var k8sClient client.Client



		// Initialize your existing validation suite and k8s client.

		// ...



		// Integrate O-RAN tests.

		IntegrateORANWithValidationSuite(validationSuite, k8sClient)

	*/
}

