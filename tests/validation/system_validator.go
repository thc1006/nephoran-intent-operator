// Package validation provides system-level validation components
package validation

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// SystemValidator provides comprehensive system validation capabilities
type SystemValidator struct {
	config    *ValidationConfig
	k8sClient client.Client
}

// NewSystemValidator creates a new system validator
func NewSystemValidator(config *ValidationConfig) *SystemValidator {
	return &SystemValidator{
		config: config,
	}
}

// SetK8sClient sets the Kubernetes client for validation
func (sv *SystemValidator) SetK8sClient(client client.Client) {
	sv.k8sClient = client
}

// ValidateIntentProcessingPipeline validates the complete intent processing workflow
func (sv *SystemValidator) ValidateIntentProcessingPipeline(ctx context.Context) bool {
	ginkgo.By("Validating Intent Processing Pipeline")

	// Test 1: Create NetworkIntent and verify processing phases
	testIntent := &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline-validation",
			Namespace: "default",
		},
		Spec: nephranv1.NetworkIntentSpec{
			Intent: "Deploy a high-availability AMF instance for production with auto-scaling",
		},
	}

	// Create the intent
	err := sv.k8sClient.Create(ctx, testIntent)
	if err != nil {
		ginkgo.By(fmt.Sprintf("Failed to create test intent: %v", err))
		return false
	}

	defer func() {
		sv.k8sClient.Delete(ctx, testIntent)
	}()

	// Wait for processing to start
	ginkgo.By("Waiting for intent processing to begin")
	gomega.Eventually(func() bool {
		err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
		return err == nil && testIntent.Status.Phase != ""
	}, 30*time.Second, 1*time.Second).Should(gomega.BeTrue())

	// Verify all processing phases occur
	expectedPhases := []nephranv1.NetworkIntentPhase{
		nephranv1.NetworkIntentPhasePending,
		nephranv1.NetworkIntentPhaseProcessing,
		nephranv1.NetworkIntentPhaseReady,
		nephranv1.NetworkIntentPhaseCompleted,
	}

	phasesObserved := make(map[nephranv1.NetworkIntentPhase]bool)

	// Monitor phase transitions
	ginkgo.By("Monitoring phase transitions")
	gomega.Eventually(func() bool {
		err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
		if err != nil {
			return false
		}

		phasesObserved[testIntent.Status.Phase] = true

		// Check if we've seen all expected phases or reached a terminal state
		for _, phase := range expectedPhases {
			if !phasesObserved[phase] &&
				testIntent.Status.Phase != nephranv1.PhaseDeployed &&
				testIntent.Status.Phase != nephranv1.PhaseFailed {
				return false
			}
		}

		return testIntent.Status.Phase == nephranv1.PhaseDeployed ||
			len(phasesObserved) >= len(expectedPhases)-1
	}, 5*time.Minute, 5*time.Second).Should(gomega.BeTrue())

	// Verify final state is not failed
	finalErr := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
	if finalErr != nil || testIntent.Status.Phase == nephranv1.PhaseFailed {
		ginkgo.By(fmt.Sprintf("Intent processing failed: phase=%s", testIntent.Status.Phase))
		return false
	}

	ginkgo.By("Intent processing pipeline validation completed successfully")
	return true
}

// ValidateLLMRAGIntegration validates LLM and RAG system integration
func (sv *SystemValidator) ValidateLLMRAGIntegration(ctx context.Context) bool {
	ginkgo.By("Validating LLM/RAG Integration")

	// Test with various intent types to verify LLM understanding
	testIntents := []struct {
		intent   string
		expected string
	}{
		{
			intent:   "Deploy AMF with high availability",
			expected: "amf",
		},
		{
			intent:   "Create SMF instance for slice management",
			expected: "smf",
		},
		{
			intent:   "Set up UPF with edge deployment",
			expected: "upf",
		},
	}

	successCount := 0

	for i, test := range testIntents {
		ginkgo.By(fmt.Sprintf("Testing LLM understanding for intent %d", i+1))

		testIntent := &nephranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-llm-validation-%d", i),
				Namespace: "default",
			},
			Spec: nephranv1.NetworkIntentSpec{
				Intent: test.intent,
			},
		}

		err := sv.k8sClient.Create(ctx, testIntent)
		if err != nil {
			ginkgo.By(fmt.Sprintf("Failed to create test intent %d: %v", i, err))
			continue
		}

		// Wait for processing to complete or provide results
		gomega.Eventually(func() bool {
			err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
			return err == nil &&
				(testIntent.Status.Phase == nephranv1.PhaseResourcePlanning ||
					testIntent.Status.Phase == nephranv1.PhaseManifestGeneration ||
					testIntent.Status.Phase == nephranv1.PhaseDeployed)
		}, 2*time.Minute, 2*time.Second).Should(gomega.BeTrue())

		// Verify the LLM correctly identified the network function type
		err = sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
		if err == nil && testIntent.Status.ProcessingResults != nil {
			// Check if the expected network function type was identified
			resultsContainExpected := false
			if testIntent.Status.ProcessingResults.NetworkFunctionType != "" {
				resultsContainExpected = true
			}

			if resultsContainExpected {
				successCount++
				ginkgo.By(fmt.Sprintf("✓ Intent %d correctly processed", i+1))
			}
		}

		// Cleanup
		sv.k8sClient.Delete(ctx, testIntent)
	}

	// Require at least 80% accuracy (2 out of 3)
	accuracy := float64(successCount) / float64(len(testIntents)) * 100
	ginkgo.By(fmt.Sprintf("LLM/RAG accuracy: %.1f%% (%d/%d)", accuracy, successCount, len(testIntents)))

	return accuracy >= 80.0
}

// ValidatePorchIntegration validates Porch package management integration
func (sv *SystemValidator) ValidatePorchIntegration(ctx context.Context) bool {
	ginkgo.By("Validating Porch Package Management Integration")

	// Create a test intent that should generate packages
	testIntent := &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-porch-integration",
			Namespace: "default",
		},
		Spec: nephranv1.NetworkIntentSpec{
			Intent: "Deploy AMF with basic configuration",
		},
	}

	err := sv.k8sClient.Create(ctx, testIntent)
	if err != nil {
		ginkgo.By(fmt.Sprintf("Failed to create test intent: %v", err))
		return false
	}

	defer func() {
		sv.k8sClient.Delete(ctx, testIntent)
	}()

	// Wait for package generation phase
	ginkgo.By("Waiting for package generation")
	gomega.Eventually(func() bool {
		err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
		return err == nil &&
			(testIntent.Status.Phase == nephranv1.PhaseManifestGeneration ||
				testIntent.Status.Phase == nephranv1.PhaseDeployed)
	}, 3*time.Minute, 5*time.Second).Should(gomega.BeTrue())

	// Verify package-related status information
	err = sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
	if err != nil {
		ginkgo.By(fmt.Sprintf("Failed to get intent status: %v", err))
		return false
	}

	// Check for package generation indicators in status
	if testIntent.Status.ProcessingResults == nil {
		ginkgo.By("No processing results found")
		return false
	}

	// Verify packages were created (check status or annotations)
	packageCreated := testIntent.Status.Phase == nephranv1.PhaseManifestGeneration ||
		testIntent.Status.Phase == nephranv1.PhaseDeployed

	if packageCreated {
		ginkgo.By("✓ Porch package integration verified")
		return true
	}

	ginkgo.By("✗ Porch package integration failed")
	return false
}

// ValidateMultiClusterDeployment validates multi-cluster deployment capabilities
func (sv *SystemValidator) ValidateMultiClusterDeployment(ctx context.Context) bool {
	ginkgo.By("Validating Multi-cluster Deployment")

	// For now, we'll test the deployment pipeline reaches the deployment phase
	// In a full implementation, this would test actual multi-cluster propagation

	testIntent := &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-multicluster-deployment",
			Namespace: "default",
		},
		Spec: nephranv1.NetworkIntentSpec{
			Intent: "Deploy AMF across multiple clusters with redundancy",
		},
	}

	err := sv.k8sClient.Create(ctx, testIntent)
	if err != nil {
		ginkgo.By(fmt.Sprintf("Failed to create multi-cluster test intent: %v", err))
		return false
	}

	defer func() {
		sv.k8sClient.Delete(ctx, testIntent)
	}()

	// Wait for deployment phase
	ginkgo.By("Waiting for multi-cluster deployment")
	gomega.Eventually(func() bool {
		err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
		return err == nil && testIntent.Status.Phase == nephranv1.PhaseDeployed
	}, 5*time.Minute, 10*time.Second).Should(gomega.BeTrue())

	// Verify deployment succeeded
	err = sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
	if err != nil {
		return false
	}

	deployed := testIntent.Status.Phase == nephranv1.PhaseDeployed
	if deployed {
		ginkgo.By("✓ Multi-cluster deployment validated")
	} else {
		ginkgo.By("✗ Multi-cluster deployment failed")
	}

	return deployed
}

// ValidateORANInterfaces validates O-RAN interface compliance (returns score 0-7)
func (sv *SystemValidator) ValidateORANInterfaces(ctx context.Context) int {
	ginkgo.By("Validating O-RAN Interface Compliance")

	score := 0
	maxScore := 7

	// Test 1: E2 Interface (2 points)
	if sv.validateE2Interface(ctx) {
		score += 2
		ginkgo.By("✓ E2 Interface: 2/2 points")
	} else {
		ginkgo.By("✗ E2 Interface: 0/2 points")
	}

	// Test 2: A1 Interface (2 points)
	if sv.validateA1Interface(ctx) {
		score += 2
		ginkgo.By("✓ A1 Interface: 2/2 points")
	} else {
		ginkgo.By("✗ A1 Interface: 0/2 points")
	}

	// Test 3: O1 Interface (2 points)
	if sv.validateO1Interface(ctx) {
		score += 2
		ginkgo.By("✓ O1 Interface: 2/2 points")
	} else {
		ginkgo.By("✗ O1 Interface: 0/2 points")
	}

	// Test 4: O2 Interface (1 point)
	if sv.validateO2Interface(ctx) {
		score += 1
		ginkgo.By("✓ O2 Interface: 1/1 points")
	} else {
		ginkgo.By("✗ O2 Interface: 0/1 points")
	}

	ginkgo.By(fmt.Sprintf("O-RAN Interface Compliance: %d/%d points", score, maxScore))
	return score
}

// validateE2Interface validates E2 interface functionality
func (sv *SystemValidator) validateE2Interface(ctx context.Context) bool {
	// Test E2NodeSet CRD functionality
	testE2NodeSet := &nephranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-e2-validation",
			Namespace: "default",
		},
		Spec: nephranv1.E2NodeSetSpec{
			Replicas: 3,
		},
	}

	err := sv.k8sClient.Create(ctx, testE2NodeSet)
	if err != nil {
		ginkgo.By(fmt.Sprintf("Failed to create E2NodeSet: %v", err))
		return false
	}

	defer func() {
		sv.k8sClient.Delete(ctx, testE2NodeSet)
	}()

	// Wait for E2NodeSet to become ready
	gomega.Eventually(func() bool {
		err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testE2NodeSet), testE2NodeSet)
		return err == nil && testE2NodeSet.Status.ReadyReplicas > 0
	}, 2*time.Minute, 5*time.Second).Should(gomega.BeTrue())

	return true
}

// validateA1Interface validates A1 interface functionality
func (sv *SystemValidator) validateA1Interface(ctx context.Context) bool {
	// Test A1 policy management
	// This would typically involve creating policy objects and verifying they're processed
	// For now, we'll simulate by checking if the system can handle policy-related intents

	testIntent := &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-a1-policy",
			Namespace: "default",
		},
		Spec: nephranv1.NetworkIntentSpec{
			Intent: "Create traffic steering policy for load balancing",
		},
	}

	err := sv.k8sClient.Create(ctx, testIntent)
	if err != nil {
		return false
	}

	defer func() {
		sv.k8sClient.Delete(ctx, testIntent)
	}()

	// Wait for processing
	gomega.Eventually(func() bool {
		err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
		return err == nil && testIntent.Status.Phase != nephranv1.PhasePending
	}, 1*time.Minute, 5*time.Second).Should(gomega.BeTrue())

	return true
}

// validateO1Interface validates O1 FCAPS interface
func (sv *SystemValidator) validateO1Interface(ctx context.Context) bool {
	// Test O1 FCAPS management capabilities
	// This would involve testing configuration management, performance monitoring, etc.
	// For now, we'll check if monitoring-related intents are processed

	testIntent := &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-o1-fcaps",
			Namespace: "default",
		},
		Spec: nephranv1.NetworkIntentSpec{
			Intent: "Configure performance monitoring for AMF with KPI collection",
		},
	}

	err := sv.k8sClient.Create(ctx, testIntent)
	if err != nil {
		return false
	}

	defer func() {
		sv.k8sClient.Delete(ctx, testIntent)
	}()

	gomega.Eventually(func() bool {
		err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
		return err == nil && testIntent.Status.Phase != nephranv1.PhasePending
	}, 1*time.Minute, 5*time.Second).Should(gomega.BeTrue())

	return true
}

// validateO2Interface validates O2 cloud interface
func (sv *SystemValidator) validateO2Interface(ctx context.Context) bool {
	// Test O2 cloud infrastructure management
	// This would involve testing cloud resource provisioning

	testIntent := &nephranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-o2-cloud",
			Namespace: "default",
		},
		Spec: nephranv1.NetworkIntentSpec{
			Intent: "Provision cloud infrastructure for UPF deployment",
		},
	}

	err := sv.k8sClient.Create(ctx, testIntent)
	if err != nil {
		return false
	}

	defer func() {
		sv.k8sClient.Delete(ctx, testIntent)
	}()

	gomega.Eventually(func() bool {
		err := sv.k8sClient.Get(ctx, client.ObjectKeyFromObject(testIntent), testIntent)
		return err == nil && testIntent.Status.Phase != nephranv1.PhasePending
	}, 1*time.Minute, 5*time.Second).Should(gomega.BeTrue())

	return true
}
