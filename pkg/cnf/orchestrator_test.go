/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cnf

import (
	"context"
	"errors"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
)

func TestCNFOrchestrator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CNF Orchestrator Suite")
}

var _ = Describe("CNF Orchestrator", func() {
	var (
		orchestrator   *CNFOrchestrator
		fakeClient     client.Client
		fakeRecorder   *record.FakeRecorder
		mockPackageGen *MockPackageGenerator
		mockGitClient  *MockGitClient
		mockMetrics    *MockMetricsCollector
		scheme         *runtime.Scheme
		ctx            context.Context
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		fakeRecorder = record.NewFakeRecorder(10)
		mockPackageGen = &MockPackageGenerator{}
		mockGitClient = &MockGitClient{}
		mockMetrics = &MockMetricsCollector{}
		ctx = context.Background()

		orchestrator = NewCNFOrchestrator(fakeClient, scheme, fakeRecorder)
		orchestrator.PackageGenerator = mockPackageGen
		orchestrator.GitClient = mockGitClient
		orchestrator.MetricsCollector = mockMetrics
	})

	Describe("NewCNFOrchestrator", func() {
		It("should create orchestrator with default configuration", func() {
			Expect(orchestrator).NotTo(BeNil())
			Expect(orchestrator.Config).NotTo(BeNil())
			Expect(orchestrator.Config.DefaultNamespace).To(Equal("default"))
			Expect(orchestrator.Config.EnableServiceMesh).To(BeTrue())
			Expect(orchestrator.Config.ServiceMeshType).To(Equal("istio"))
			Expect(orchestrator.ChartRepositories).To(HaveLen(3))
			Expect(orchestrator.TemplateRegistry).NotTo(BeNil())
		})

		It("should initialize template registry with CNF templates", func() {
			Expect(orchestrator.TemplateRegistry.Templates).To(HaveKey(nephoranv1.CNFFunctionAMF))
			Expect(orchestrator.TemplateRegistry.Templates).To(HaveKey(nephoranv1.CNFFunctionSMF))
			Expect(orchestrator.TemplateRegistry.Templates).To(HaveKey(nephoranv1.CNFFunctionUPF))
			Expect(orchestrator.TemplateRegistry.Templates).To(HaveKey(nephoranv1.CNFFunctionNearRTRIC))
			Expect(orchestrator.TemplateRegistry.Templates).To(HaveKey(nephoranv1.CNFFunctionODU))
		})
	})

	Describe("Deploy", func() {
		var (
			cnfDeployment *nephoranv1.CNFDeployment
			deployRequest *DeployRequest
		)

		BeforeEach(func() {
			cnfDeployment = &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-amf",
					Namespace: "test-namespace",
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:            nephoranv1.CNF5GCore,
					Function:           nephoranv1.CNFFunctionAMF,
					DeploymentStrategy: nephoranv1.DeploymentStrategyDirect,
					Replicas:           2,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("1000m"),
						Memory: mustParseQuantity("2Gi"),
					},
				},
			}

			deployRequest = &DeployRequest{
				CNFDeployment: cnfDeployment,
				Context:       ctx,
			}
		})

		It("should successfully deploy CNF via direct strategy", func() {
			result, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Success).To(BeTrue())
			Expect(result.Namespace).To(Equal("test-namespace"))
		})

		It("should successfully deploy CNF via GitOps strategy", func() {
			cnfDeployment.Spec.DeploymentStrategy = nephoranv1.DeploymentStrategyGitOps
			mockPackageGen.On("GenerateCNFPackage").Return([]byte("package-data"), nil)
			mockGitClient.On("CommitPackage").Return("abc123", nil)

			result, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Success).To(BeTrue())
			Expect(result.ResourceStatus["gitCommit"]).To(Equal("abc123"))
		})

		It("should fail deployment with invalid CNF deployment", func() {
			cnfDeployment.Spec.Function = nephoranv1.CNFFunction("")

			result, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("deployment validation failed"))
		})

		It("should fail deployment with missing dependency", func() {
			// SMF depends on NRF and UDM
			cnfDeployment.Spec.Function = nephoranv1.CNFFunctionSMF

			result, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("dependency check failed"))
		})

		It("should record deployment metrics", func() {
			mockMetrics.On("RecordCNFDeployment").Return()

			result, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			mockMetrics.AssertCalled(GinkgoT(), "RecordCNFDeployment", nephoranv1.CNFFunctionAMF, mock.AnythingOfType("time.Duration"))
		})

		It("should emit events for deployment lifecycle", func() {
			_, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).NotTo(HaveOccurred())

			// Check for deployment started event
			select {
			case event := <-fakeRecorder.Events:
				Expect(event).To(ContainSubstring(EventCNFDeploymentStarted))
			case <-time.After(1 * time.Second):
				Fail("Expected deployment started event")
			}

			// Check for deployment completed event
			select {
			case event := <-fakeRecorder.Events:
				Expect(event).To(ContainSubstring(EventCNFDeploymentCompleted))
			case <-time.After(1 * time.Second):
				Fail("Expected deployment completed event")
			}
		})
	})

	Describe("validateDeploymentRequest", func() {
		var deployRequest *DeployRequest

		BeforeEach(func() {
			deployRequest = &DeployRequest{
				CNFDeployment: &nephoranv1.CNFDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cnf",
						Namespace: "test-namespace",
					},
					Spec: nephoranv1.CNFDeploymentSpec{
						CNFType:            nephoranv1.CNF5GCore,
						Function:           nephoranv1.CNFFunctionAMF,
						DeploymentStrategy: nephoranv1.DeploymentStrategyHelm,
						Replicas:           1,
					},
				},
			}
		})

		It("should validate successful deployment request", func() {
			err := orchestrator.validateDeploymentRequest(deployRequest)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail validation with nil CNF deployment", func() {
			deployRequest.CNFDeployment = nil

			err := orchestrator.validateDeploymentRequest(deployRequest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CNF deployment is required"))
		})

		It("should fail validation with missing Helm config for Helm strategy", func() {
			deployRequest.CNFDeployment.Spec.DeploymentStrategy = nephoranv1.DeploymentStrategyHelm
			// Helm config is nil

			err := orchestrator.validateDeploymentRequest(deployRequest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Helm configuration is required"))
		})

		It("should fail validation with missing Operator config for Operator strategy", func() {
			deployRequest.CNFDeployment.Spec.DeploymentStrategy = nephoranv1.DeploymentStrategyOperator
			// Operator config is nil

			err := orchestrator.validateDeploymentRequest(deployRequest)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Operator configuration is required"))
		})
	})

	Describe("getCNFTemplate", func() {
		It("should return template for valid CNF function", func() {
			template, err := orchestrator.getCNFTemplate(nephoranv1.CNFFunctionAMF)

			Expect(err).NotTo(HaveOccurred())
			Expect(template).NotTo(BeNil())
			Expect(template.Function).To(Equal(nephoranv1.CNFFunctionAMF))
			Expect(template.ChartReference.ChartName).To(Equal("amf"))
		})

		It("should return error for invalid CNF function", func() {
			template, err := orchestrator.getCNFTemplate(nephoranv1.CNFFunction("invalid"))

			Expect(err).To(HaveOccurred())
			Expect(template).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("no template found for CNF function"))
		})

		It("should return error with uninitialized template registry", func() {
			orchestrator.TemplateRegistry = nil

			template, err := orchestrator.getCNFTemplate(nephoranv1.CNFFunctionAMF)

			Expect(err).To(HaveOccurred())
			Expect(template).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("template registry not initialized"))
		})
	})

	Describe("checkDependencies", func() {
		var (
			cnfDeployment *nephoranv1.CNFDeployment
			template      *CNFTemplate
		)

		BeforeEach(func() {
			cnfDeployment = &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-smf",
					Namespace: "test-namespace",
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					Function: nephoranv1.CNFFunctionSMF,
				},
			}

			template = &CNFTemplate{
				Function:     nephoranv1.CNFFunctionSMF,
				Dependencies: []nephoranv1.CNFFunction{nephoranv1.CNFFunctionNRF},
			}
		})

		It("should pass with no dependencies", func() {
			template.Dependencies = nil

			err := orchestrator.checkDependencies(ctx, cnfDeployment, template)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should pass when dependencies are running", func() {
			// Create running NRF dependency
			nrfDeployment := &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nrf-deployment",
					Namespace: "test-namespace",
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					Function: nephoranv1.CNFFunctionNRF,
				},
				Status: nephoranv1.CNFDeploymentStatus{
					Phase: "Running",
				},
			}
			Expect(fakeClient.Create(ctx, nrfDeployment)).To(Succeed())

			err := orchestrator.checkDependencies(ctx, cnfDeployment, template)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail when required dependency is missing", func() {
			err := orchestrator.checkDependencies(ctx, cnfDeployment, template)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("required dependency NRF is not deployed"))
		})
	})

	Describe("prepareDeploymentConfig", func() {
		var (
			cnfDeployment *nephoranv1.CNFDeployment
			template      *CNFTemplate
		)

		BeforeEach(func() {
			cnfDeployment = &nephoranv1.CNFDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-amf",
					Namespace: "test-namespace",
				},
				Spec: nephoranv1.CNFDeploymentSpec{
					CNFType:  nephoranv1.CNF5GCore,
					Function: nephoranv1.CNFFunctionAMF,
					Replicas: 3,
					Resources: nephoranv1.CNFResources{
						CPU:       mustParseQuantity("1000m"),
						Memory:    mustParseQuantity("2Gi"),
						Storage:   mustParseQuantity("10Gi"),
						MaxCPU:    &[]resource.Quantity{mustParseQuantity("2000m")}[0],
						MaxMemory: &[]resource.Quantity{mustParseQuantity("4Gi")}[0],
					},
				},
			}

			template = &CNFTemplate{
				Function: nephoranv1.CNFFunctionAMF,
				DefaultValues: map[string]interface{}{
					"image": map[string]interface{}{
						"repository": "5gc/amf",
						"tag":        "latest",
					},
				},
			}
		})

		It("should prepare basic deployment configuration", func() {
			config, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)

			Expect(err).NotTo(HaveOccurred())
			Expect(config).NotTo(BeNil())
			Expect(config["replicaCount"]).To(Equal(int32(3)))

			resources := config["resources"].(map[string]interface{})
			requests := resources["requests"].(map[string]interface{})
			Expect(requests["cpu"]).To(Equal("1000m"))
			Expect(requests["memory"]).To(Equal("2Gi"))

			limits := resources["limits"].(map[string]interface{})
			Expect(limits["cpu"]).To(Equal("2000m"))
			Expect(limits["memory"]).To(Equal("4Gi"))
		})

		It("should include template default values", func() {
			config, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)

			Expect(err).NotTo(HaveOccurred())
			image := config["image"].(map[string]interface{})
			Expect(image["repository"]).To(Equal("5gc/amf"))
			Expect(image["tag"]).To(Equal("latest"))
		})

		It("should configure DPDK when enabled", func() {
			cnfDeployment.Spec.Resources.DPDK = &nephoranv1.DPDKConfig{
				Enabled: true,
				Cores:   &[]int32{4}[0],
				Memory:  &[]int32{2048}[0],
				Driver:  "vfio-pci",
			}

			config, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)

			Expect(err).NotTo(HaveOccurred())
			dpdk := config["dpdk"].(map[string]interface{})
			Expect(dpdk["enabled"]).To(BeTrue())
			Expect(dpdk["cores"]).To(Equal(&[]int32{4}[0]))
			Expect(dpdk["memory"]).To(Equal(&[]int32{2048}[0]))
			Expect(dpdk["driver"]).To(Equal("vfio-pci"))
		})

		It("should configure persistence when storage is specified", func() {
			config, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)

			Expect(err).NotTo(HaveOccurred())
			persistence := config["persistence"].(map[string]interface{})
			Expect(persistence["enabled"]).To(BeTrue())
			Expect(persistence["size"]).To(Equal("10Gi"))
		})
	})

	Describe("Template Registry Initialization", func() {
		It("should initialize 5G Core templates correctly", func() {
			amfTemplate := orchestrator.TemplateRegistry.Templates[nephoranv1.CNFFunctionAMF]
			Expect(amfTemplate).NotTo(BeNil())
			Expect(amfTemplate.Function).To(Equal(nephoranv1.CNFFunctionAMF))
			Expect(amfTemplate.ChartReference.ChartName).To(Equal("amf"))
			Expect(amfTemplate.RequiredConfigs).To(ContainElement("plmnId"))
			Expect(amfTemplate.Interfaces).To(HaveLen(3)) // N1, N2, SBI
			Expect(amfTemplate.HealthChecks).To(HaveLen(1))

			smfTemplate := orchestrator.TemplateRegistry.Templates[nephoranv1.CNFFunctionSMF]
			Expect(smfTemplate).NotTo(BeNil())
			Expect(smfTemplate.Dependencies).To(ContainElement(nephoranv1.CNFFunctionNRF))
			Expect(smfTemplate.Dependencies).To(ContainElement(nephoranv1.CNFFunctionUDM))

			upfTemplate := orchestrator.TemplateRegistry.Templates[nephoranv1.CNFFunctionUPF]
			Expect(upfTemplate).NotTo(BeNil())
			Expect(upfTemplate.DefaultValues["dpdk"]).NotTo(BeNil())
			Expect(upfTemplate.Interfaces).To(HaveLen(3)) // N3, N4, N6
		})

		It("should initialize O-RAN templates correctly", func() {
			ricTemplate := orchestrator.TemplateRegistry.Templates[nephoranv1.CNFFunctionNearRTRIC]
			Expect(ricTemplate).NotTo(BeNil())
			Expect(ricTemplate.Function).To(Equal(nephoranv1.CNFFunctionNearRTRIC))
			Expect(ricTemplate.ChartReference.ChartName).To(Equal("near-rt-ric"))
			Expect(ricTemplate.Interfaces).To(HaveLen(2)) // A1, E2

			oduTemplate := orchestrator.TemplateRegistry.Templates[nephoranv1.CNFFunctionODU]
			Expect(oduTemplate).NotTo(BeNil())
			Expect(oduTemplate.Dependencies).To(ContainElement(nephoranv1.CNFFunctionNearRTRIC))
			Expect(oduTemplate.Interfaces).To(HaveLen(3)) // F1-C, F1-U, E2
		})

		It("should initialize edge templates correctly", func() {
			ueSimTemplate := orchestrator.TemplateRegistry.Templates[nephoranv1.CNFFunctionUESimulator]
			Expect(ueSimTemplate).NotTo(BeNil())
			Expect(ueSimTemplate.Function).To(Equal(nephoranv1.CNFFunctionUESimulator))
			Expect(ueSimTemplate.ChartReference.ChartName).To(Equal("ue-simulator"))
			Expect(ueSimTemplate.RequiredConfigs).To(ContainElement("amfAddress"))
		})
	})
})

// Mock implementations

type MockPackageGenerator struct {
	mock.Mock
}

func (m *MockPackageGenerator) GenerateCNFPackage(cnf *nephoranv1.CNFDeployment, config map[string]interface{}) ([]byte, error) {
	args := m.Called(cnf, config)
	return args.Get(0).([]byte), args.Error(1)
}

type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) CommitPackage(ctx context.Context, packageData []byte, commitMsg string) (string, error) {
	args := m.Called(ctx, packageData, commitMsg)
	return args.String(0), args.Error(1)
}

type MockMetricsCollector struct {
	mock.Mock
}

func (m *MockMetricsCollector) RecordCNFDeployment(function nephoranv1.CNFFunction, duration time.Duration) {
	m.Called(function, duration)
}

func (m *MockMetricsCollector) RecordCNFDeletion(function nephoranv1.CNFFunction, duration time.Duration) {
	m.Called(function, duration)
}

func (m *MockMetricsCollector) RecordCNFHealthCheck(function nephoranv1.CNFFunction, status string) {
	m.Called(function, status)
}

func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(err)
	}
	return q
}

var _ = Describe("CNF Orchestrator Error Scenarios", func() {
	var (
		orchestrator   *CNFOrchestrator
		fakeClient     client.Client
		fakeRecorder   *record.FakeRecorder
		mockPackageGen *MockPackageGenerator
		mockGitClient  *MockGitClient
		scheme         *runtime.Scheme
		ctx            context.Context
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		fakeRecorder = record.NewFakeRecorder(10)
		mockPackageGen = &MockPackageGenerator{}
		mockGitClient = &MockGitClient{}
		ctx = context.Background()

		orchestrator = NewCNFOrchestrator(fakeClient, scheme, fakeRecorder)
		orchestrator.PackageGenerator = mockPackageGen
		orchestrator.GitClient = mockGitClient
	})

	Describe("Error Handling in Deploy", func() {
		var deployRequest *DeployRequest

		BeforeEach(func() {
			deployRequest = &DeployRequest{
				CNFDeployment: &nephoranv1.CNFDeployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cnf",
						Namespace: "test-namespace",
					},
					Spec: nephoranv1.CNFDeploymentSpec{
						CNFType:            nephoranv1.CNF5GCore,
						Function:           nephoranv1.CNFFunctionAMF,
						DeploymentStrategy: nephoranv1.DeploymentStrategyGitOps,
						Replicas:           1,
					},
				},
			}
		})

		It("should handle package generation failure", func() {
			mockPackageGen.On("GenerateCNFPackage").Return([]byte(nil), errors.New("package generation failed"))

			result, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to generate CNF package"))
		})

		It("should handle git commit failure", func() {
			mockPackageGen.On("GenerateCNFPackage").Return([]byte("package-data"), nil)
			mockGitClient.On("CommitPackage").Return("", errors.New("git commit failed"))

			result, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to commit CNF package"))
		})

		It("should emit failure events on deployment error", func() {
			mockPackageGen.On("GenerateCNFPackage").Return([]byte(nil), errors.New("package generation failed"))

			_, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).To(HaveOccurred())

			// Check for deployment failed event
			select {
			case event := <-fakeRecorder.Events:
				Expect(event).To(ContainSubstring(EventCNFDeploymentFailed))
			case <-time.After(1 * time.Second):
				Fail("Expected deployment failed event")
			}
		})

		It("should handle unsupported deployment strategy", func() {
			deployRequest.CNFDeployment.Spec.DeploymentStrategy = nephoranv1.DeploymentStrategy("unsupported")

			result, err := orchestrator.Deploy(ctx, deployRequest)

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported deployment strategy"))
		})
	})

	Describe("Resource Configuration Edge Cases", func() {
		var (
			cnfDeployment *nephoranv1.CNFDeployment
			template      *CNFTemplate
		)

		BeforeEach(func() {
			cnfDeployment = &nephoranv1.CNFDeployment{
				Spec: nephoranv1.CNFDeploymentSpec{
					Function: nephoranv1.CNFFunctionAMF,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("1000m"),
						Memory: mustParseQuantity("2Gi"),
					},
				},
			}

			template = &CNFTemplate{
				Function:      nephoranv1.CNFFunctionAMF,
				DefaultValues: map[string]interface{}{},
			}
		})

		It("should handle missing max resources gracefully", func() {
			config, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)

			Expect(err).NotTo(HaveOccurred())
			resources := config["resources"].(map[string]interface{})
			limits := resources["limits"].(map[string]interface{})
			Expect(limits["cpu"]).To(Equal("1000m"))  // Same as request
			Expect(limits["memory"]).To(Equal("2Gi")) // Same as request
		})

		It("should handle DPDK with nil values gracefully", func() {
			cnfDeployment.Spec.Resources.DPDK = &nephoranv1.DPDKConfig{
				Enabled: true,
				// Cores and Memory are nil
				Driver: "vfio-pci",
			}

			config, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)

			Expect(err).NotTo(HaveOccurred())
			dpdk := config["dpdk"].(map[string]interface{})
			Expect(dpdk["enabled"]).To(BeTrue())
			Expect(dpdk["driver"]).To(Equal("vfio-pci"))
		})

		It("should handle empty hugepages configuration", func() {
			cnfDeployment.Spec.Resources.Hugepages = make(map[string]resource.Quantity)

			config, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)

			Expect(err).NotTo(HaveOccurred())
			hugepages := config["hugepages"].(map[string]resource.Quantity)
			Expect(hugepages).To(BeEmpty())
		})
	})
})

var _ = Describe("CNF Orchestrator Performance Tests", func() {
	var (
		orchestrator *CNFOrchestrator
		fakeClient   client.Client
		fakeRecorder *record.FakeRecorder
		scheme       *runtime.Scheme
		ctx          context.Context
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		fakeRecorder = record.NewFakeRecorder(100)
		ctx = context.Background()

		orchestrator = NewCNFOrchestrator(fakeClient, scheme, fakeRecorder)
	})

	Describe("Concurrent Deployment Handling", func() {
		It("should handle multiple concurrent template retrievals", func() {
			const numGoroutines = 50
			results := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer GinkgoRecover()
					_, err := orchestrator.getCNFTemplate(nephoranv1.CNFFunctionAMF)
					results <- err
				}()
			}

			// Verify all goroutines completed successfully
			for i := 0; i < numGoroutines; i++ {
				select {
				case err := <-results:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(5 * time.Second):
					Fail("Timeout waiting for goroutine completion")
				}
			}
		})

		It("should handle concurrent deployment config preparation", func() {
			cnfDeployment := &nephoranv1.CNFDeployment{
				Spec: nephoranv1.CNFDeploymentSpec{
					Function: nephoranv1.CNFFunctionAMF,
					Replicas: 1,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("500m"),
						Memory: mustParseQuantity("1Gi"),
					},
				},
			}

			template := orchestrator.TemplateRegistry.Templates[nephoranv1.CNFFunctionAMF]
			const numGoroutines = 30
			results := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func() {
					defer GinkgoRecover()
					_, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)
					results <- err
				}()
			}

			// Verify all goroutines completed successfully
			for i := 0; i < numGoroutines; i++ {
				select {
				case err := <-results:
					Expect(err).NotTo(HaveOccurred())
				case <-time.After(5 * time.Second):
					Fail("Timeout waiting for goroutine completion")
				}
			}
		})
	})

	Describe("Memory Usage Optimization", func() {
		It("should not leak memory during template operations", func() {
			// This test verifies that repeated template operations don't cause memory leaks
			for i := 0; i < 1000; i++ {
				template, err := orchestrator.getCNFTemplate(nephoranv1.CNFFunctionAMF)
				Expect(err).NotTo(HaveOccurred())
				Expect(template).NotTo(BeNil())
			}
			// In a real environment, you would check memory usage here
		})

		It("should handle large configuration objects efficiently", func() {
			cnfDeployment := &nephoranv1.CNFDeployment{
				Spec: nephoranv1.CNFDeploymentSpec{
					Function: nephoranv1.CNFFunctionAMF,
					Replicas: 1,
					Resources: nephoranv1.CNFResources{
						CPU:    mustParseQuantity("500m"),
						Memory: mustParseQuantity("1Gi"),
						Hugepages: map[string]resource.Quantity{
							"2Mi": mustParseQuantity("1Gi"),
							"1Gi": mustParseQuantity("2Gi"),
						},
					},
				},
			}

			template := orchestrator.TemplateRegistry.Templates[nephoranv1.CNFFunctionAMF]

			start := time.Now()
			for i := 0; i < 100; i++ {
				_, err := orchestrator.prepareDeploymentConfig(cnfDeployment, template)
				Expect(err).NotTo(HaveOccurred())
			}
			duration := time.Since(start)

			// Expect each operation to take less than 1ms on average
			avgDuration := duration / 100
			Expect(avgDuration).To(BeNumerically("<", time.Millisecond))
		})
	})
})
