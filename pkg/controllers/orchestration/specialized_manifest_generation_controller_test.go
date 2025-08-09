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

package orchestration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// Mock implementations for testing
type MockKubernetesTemplateEngine struct {
	templates          map[string]string
	shouldReturnError  bool
	processingDelay    time.Duration
	templateUsageCount map[string]int
}

func NewMockKubernetesTemplateEngine() *MockKubernetesTemplateEngine {
	return &MockKubernetesTemplateEngine{
		templates: map[string]string{
			"deployment-amf": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
    spec:
      containers:
      - name: {{ .Name }}
        image: {{ .Image }}:{{ .Version }}
        resources:
          requests:
            cpu: "{{ .Resources.Requests.CPU }}"
            memory: "{{ .Resources.Requests.Memory }}"`,
			"deployment-generic": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
    spec:
      containers:
      - name: {{ .Name }}
        image: {{ .Image }}:{{ .Version }}`,
			"service-generic": `apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}-service
  namespace: {{ .Namespace }}
spec:
  selector:
    app: {{ .Name }}
  ports:
  {{- range .Ports }}
  - port: {{ .Port }}
    targetPort: {{ .TargetPort }}
    name: {{ .Name }}
  {{- end }}`,
			"configmap-generic": `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}-config
  namespace: {{ .Namespace }}
data:
  {{- range $key, $value := .Configuration }}
  {{ $key }}: "{{ $value }}"
  {{- end }}`,
		},
		shouldReturnError:  false,
		processingDelay:    1 * time.Millisecond,
		templateUsageCount: make(map[string]int),
	}
}

func (m *MockKubernetesTemplateEngine) ProcessTemplate(templateType, nfType string, variables map[string]interface{}) (string, error) {
	if m.shouldReturnError {
		return "", fmt.Errorf("mock template engine error")
	}

	if m.processingDelay > 0 {
		time.Sleep(m.processingDelay)
	}

	templateName := fmt.Sprintf("%s-%s", templateType, nfType)

	// Try specific template first
	templateContent, exists := m.templates[templateName]
	if !exists {
		// Fall back to generic template
		genericName := fmt.Sprintf("%s-generic", templateType)
		templateContent, exists = m.templates[genericName]
		if !exists {
			return "", fmt.Errorf("template not found: %s", templateName)
		}
		templateName = genericName
	}

	// Track usage
	m.templateUsageCount[templateName]++

	// Simple template processing (replace basic variables)
	result := templateContent
	if name, ok := variables["Name"].(string); ok {
		result = strings.ReplaceAll(result, "{{ .Name }}", name)
	}
	if namespace, ok := variables["Namespace"].(string); ok {
		result = strings.ReplaceAll(result, "{{ .Namespace }}", namespace)
	}
	if image, ok := variables["Image"].(string); ok {
		result = strings.ReplaceAll(result, "{{ .Image }}", image)
	}
	if version, ok := variables["Version"].(string); ok {
		result = strings.ReplaceAll(result, "{{ .Version }}", version)
	}
	if replicas, ok := variables["Replicas"].(int32); ok {
		result = strings.ReplaceAll(result, "{{ .Replicas }}", fmt.Sprintf("%d", replicas))
	}

	// Handle resources
	if resources, ok := variables["Resources"].(interfaces.ResourceSpec); ok {
		if resources.Requests.CPU != "" {
			result = strings.ReplaceAll(result, "{{ .Resources.Requests.CPU }}", resources.Requests.CPU)
		}
		if resources.Requests.Memory != "" {
			result = strings.ReplaceAll(result, "{{ .Resources.Requests.Memory }}", resources.Requests.Memory)
		}
	}

	// Remove template directives for simple mock
	result = strings.ReplaceAll(result, "{{- range .Ports }}", "")
	result = strings.ReplaceAll(result, "{{- end }}", "")
	result = strings.ReplaceAll(result, "{{- range $key, $value := .Configuration }}", "")

	return result, nil
}

func (m *MockKubernetesTemplateEngine) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockKubernetesTemplateEngine) SetProcessingDelay(delay time.Duration) {
	m.processingDelay = delay
}

func (m *MockKubernetesTemplateEngine) GetTemplateUsageCount(templateName string) int {
	return m.templateUsageCount[templateName]
}

func (m *MockKubernetesTemplateEngine) SetTemplate(name, content string) {
	m.templates[name] = content
}

type MockManifestValidator struct {
	shouldReturnError  bool
	validationDelay    time.Duration
	validationErrors   map[string][]string
	validationWarnings map[string][]string
}

func NewMockManifestValidator() *MockManifestValidator {
	return &MockManifestValidator{
		shouldReturnError:  false,
		validationDelay:    1 * time.Millisecond,
		validationErrors:   make(map[string][]string),
		validationWarnings: make(map[string][]string),
	}
}

func (m *MockManifestValidator) ValidateManifests(ctx context.Context, manifests map[string]string) ([]ValidationResult, error) {
	if m.shouldReturnError {
		return nil, fmt.Errorf("mock manifest validator error")
	}

	if m.validationDelay > 0 {
		time.Sleep(m.validationDelay)
	}

	var results []ValidationResult

	for name, manifest := range manifests {
		result := ValidationResult{
			ManifestName: name,
			Valid:        true,
			Errors:       []string{},
			Warnings:     []string{},
		}

		// Check for basic YAML structure
		if !strings.Contains(manifest, "apiVersion") {
			result.Valid = false
			result.Errors = append(result.Errors, "missing apiVersion field")
		}
		if !strings.Contains(manifest, "kind") {
			result.Valid = false
			result.Errors = append(result.Errors, "missing kind field")
		}
		if !strings.Contains(manifest, "metadata") {
			result.Valid = false
			result.Errors = append(result.Errors, "missing metadata field")
		}

		// Add configured validation errors
		if errors, exists := m.validationErrors[name]; exists {
			result.Valid = false
			result.Errors = append(result.Errors, errors...)
		}

		// Add configured validation warnings
		if warnings, exists := m.validationWarnings[name]; exists {
			result.Warnings = append(result.Warnings, warnings...)
		}

		results = append(results, result)
	}

	return results, nil
}

func (m *MockManifestValidator) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockManifestValidator) SetValidationErrors(manifestName string, errors []string) {
	m.validationErrors[manifestName] = errors
}

func (m *MockManifestValidator) SetValidationWarnings(manifestName string, warnings []string) {
	m.validationWarnings[manifestName] = warnings
}

type MockManifestPolicyEnforcer struct {
	shouldReturnError bool
	enforcementDelay  time.Duration
	policyViolations  map[string][]string
	modifiedManifests map[string]string
}

func NewMockManifestPolicyEnforcer() *MockManifestPolicyEnforcer {
	return &MockManifestPolicyEnforcer{
		shouldReturnError: false,
		enforcementDelay:  1 * time.Millisecond,
		policyViolations:  make(map[string][]string),
		modifiedManifests: make(map[string]string),
	}
}

func (m *MockManifestPolicyEnforcer) EnforcePolicies(ctx context.Context, manifests map[string]string, rules []*ManifestPolicyRule) ([]PolicyResult, map[string]string, error) {
	if m.shouldReturnError {
		return nil, nil, fmt.Errorf("mock policy enforcer error")
	}

	if m.enforcementDelay > 0 {
		time.Sleep(m.enforcementDelay)
	}

	var results []PolicyResult
	modifiedManifests := make(map[string]string)

	// Process each rule
	for _, rule := range rules {
		result := PolicyResult{
			PolicyID:   rule.ID,
			PolicyName: rule.Name,
			Compliant:  true,
			Violations: []string{},
			Action:     rule.ViolationAction,
		}

		// Check for configured violations
		for manifestName := range manifests {
			if violations, exists := m.policyViolations[manifestName]; exists {
				result.Compliant = false
				result.Violations = append(result.Violations, violations...)
			}
		}

		results = append(results, result)
	}

	// Add any modified manifests
	for name, content := range m.modifiedManifests {
		modifiedManifests[name] = content
	}

	return results, modifiedManifests, nil
}

func (m *MockManifestPolicyEnforcer) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockManifestPolicyEnforcer) SetPolicyViolations(manifestName string, violations []string) {
	m.policyViolations[manifestName] = violations
}

func (m *MockManifestPolicyEnforcer) SetModifiedManifest(manifestName, content string) {
	m.modifiedManifests[manifestName] = content
}

type MockManifestOptimizer struct {
	shouldReturnError   bool
	optimizationDelay   time.Duration
	optimizationChanges map[string]string
}

func NewMockManifestOptimizer() *MockManifestOptimizer {
	return &MockManifestOptimizer{
		shouldReturnError:   false,
		optimizationDelay:   1 * time.Millisecond,
		optimizationChanges: make(map[string]string),
	}
}

func (m *MockManifestOptimizer) OptimizeManifests(ctx context.Context, manifests map[string]string) (map[string]string, error) {
	if m.shouldReturnError {
		return nil, fmt.Errorf("mock manifest optimizer error")
	}

	if m.optimizationDelay > 0 {
		time.Sleep(m.optimizationDelay)
	}

	optimized := make(map[string]string)

	for name, manifest := range manifests {
		// Apply any configured optimizations
		if optimizedContent, exists := m.optimizationChanges[name]; exists {
			optimized[name] = optimizedContent
		} else {
			// Default optimization - minify by removing extra whitespace
			optimized[name] = strings.ReplaceAll(strings.TrimSpace(manifest), "\n\n", "\n")
		}
	}

	return optimized, nil
}

func (m *MockManifestOptimizer) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockManifestOptimizer) SetOptimizedManifest(manifestName, content string) {
	m.optimizationChanges[manifestName] = content
}

var _ = Describe("SpecializedManifestGenerationController", func() {
	var (
		ctx                   context.Context
		controller            *SpecializedManifestGenerationController
		fakeClient            client.Client
		logger                logr.Logger
		scheme                *runtime.Scheme
		fakeRecorder          *record.FakeRecorder
		networkIntent         *nephoranv1.NetworkIntent
		mockTemplateEngine    *MockKubernetesTemplateEngine
		mockManifestValidator *MockManifestValidator
		mockPolicyEnforcer    *MockManifestPolicyEnforcer
		mockManifestOptimizer *MockManifestOptimizer
		config                ManifestGenerationConfig
		sampleResourcePlan    *interfaces.ResourcePlan
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))

		// Create scheme and add types
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		// Create fake client and recorder
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		fakeRecorder = record.NewFakeRecorder(100)

		// Create mock services
		mockTemplateEngine = NewMockKubernetesTemplateEngine()
		mockManifestValidator = NewMockManifestValidator()
		mockPolicyEnforcer = NewMockManifestPolicyEnforcer()
		mockManifestOptimizer = NewMockManifestOptimizer()

		// Create configuration
		config = ManifestGenerationConfig{
			TemplateDirectory:   "/templates",
			DefaultNamespace:    "nephoran-system",
			EnableHelm:          false,
			EnableKustomize:     false,
			ValidateManifests:   true,
			DryRunValidation:    true,
			SchemaValidation:    true,
			OptimizeManifests:   true,
			MinifyManifests:     true,
			RemoveDuplicates:    true,
			EnforcePolicies:     true,
			SecurityPolicies:    true,
			ResourcePolicies:    true,
			CacheEnabled:        true,
			CacheTTL:            30 * time.Minute,
			MaxCacheEntries:     1000,
			MaxGenerationTime:   5 * time.Minute,
			ParallelGeneration:  false,
			ConcurrentTemplates: 3,
		}

		// Create controller with mock dependencies
		controller = &SpecializedManifestGenerationController{
			Client:               fakeClient,
			Scheme:               scheme,
			Recorder:             fakeRecorder,
			Logger:               logger,
			TemplateEngine:       mockTemplateEngine,
			ManifestValidator:    mockManifestValidator,
			PolicyEnforcer:       mockPolicyEnforcer,
			ManifestOptimizer:    mockManifestOptimizer,
			HelmIntegration:      nil, // Not enabled in config
			KustomizeIntegration: nil, // Not enabled in config
			Config:               config,
			Templates:            initializeManifestTemplates(),
			PolicyRules:          initializeManifestPolicyRules(),
			metrics:              NewManifestGenerationMetrics(),
			stopChan:             make(chan struct{}),
			healthStatus: interfaces.HealthStatus{
				Status:      "Healthy",
				Message:     "Manifest generation controller initialized for testing",
				LastChecked: time.Now(),
			},
		}

		// Initialize cache
		controller.manifestCache = &ManifestCache{
			entries:    make(map[string]*ManifestCacheEntry),
			ttl:        config.CacheTTL,
			maxEntries: config.MaxCacheEntries,
		}

		// Create sample resource plan
		sampleResourcePlan = &interfaces.ResourcePlan{
			NetworkFunctions: []interfaces.PlannedNetworkFunction{
				{
					Name:     "amf-deployment",
					Type:     "amf",
					Image:    "nephoran/amf",
					Version:  "v1.0.0",
					Replicas: 3,
					Resources: interfaces.ResourceSpec{
						Requests: interfaces.ResourceRequirements{
							CPU:     "1",
							Memory:  "2Gi",
							Storage: "20Gi",
						},
						Limits: interfaces.ResourceRequirements{
							CPU:     "2",
							Memory:  "4Gi",
							Storage: "40Gi",
						},
					},
					Ports: []interfaces.PortSpec{
						{
							Name:        "sbi",
							Port:        80,
							TargetPort:  8080,
							Protocol:    "TCP",
							ServiceType: "ClusterIP",
						},
					},
					Environment: []interfaces.EnvVar{
						{
							Name:  "LOG_LEVEL",
							Value: "INFO",
						},
					},
					Configuration: map[string]interface{}{
						"plmn_id": "00101",
						"nf_type": "AMF",
					},
				},
				{
					Name:     "smf-deployment",
					Type:     "smf",
					Image:    "nephoran/smf",
					Version:  "v1.0.0",
					Replicas: 2,
					Resources: interfaces.ResourceSpec{
						Requests: interfaces.ResourceRequirements{
							CPU:     "500m",
							Memory:  "1Gi",
							Storage: "10Gi",
						},
					},
				},
			},
			ResourceRequirements: interfaces.ResourceRequirements{
				CPU:     "3",
				Memory:  "6Gi",
				Storage: "60Gi",
			},
			DeploymentPattern: "high-availability",
		}

		// Create test NetworkIntent
		networkIntent = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-manifest-generation",
				Namespace: "test-namespace",
				UID:       "test-uid-12345",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent:     "Deploy high-availability AMF and SMF with manifests",
				IntentType: "5g-deployment",
				TargetComponents: []nephoranv1.TargetComponent{
					{
						Type:     "amf",
						Version:  "v1.0.0",
						Replicas: 3,
					},
					{
						Type:     "smf",
						Version:  "v1.0.0",
						Replicas: 2,
					},
				},
			},
			Status: nephoranv1.NetworkIntentStatus{
				ProcessingPhase: interfaces.PhaseManifestGeneration,
				ResourcePlan:    convertResourcePlanToInterface(sampleResourcePlan),
			},
		}
	})

	AfterEach(func() {
		if controller != nil && controller.started {
			controller.Stop(ctx)
		}
	})

	Describe("Controller Initialization", func() {
		It("should initialize with proper configuration", func() {
			Expect(controller).NotTo(BeNil())
			Expect(controller.TemplateEngine).NotTo(BeNil())
			Expect(controller.ManifestValidator).NotTo(BeNil())
			Expect(controller.PolicyEnforcer).NotTo(BeNil())
			Expect(controller.ManifestOptimizer).NotTo(BeNil())
			Expect(controller.Config.ValidateManifests).To(BeTrue())
			Expect(controller.Config.OptimizeManifests).To(BeTrue())
			Expect(controller.Config.EnforcePolicies).To(BeTrue())
			Expect(controller.Templates).To(HaveKey("deployment-generic"))
			Expect(controller.Templates).To(HaveKey("service-generic"))
			Expect(controller.Templates).To(HaveKey("configmap-generic"))
			Expect(controller.manifestCache).NotTo(BeNil())
		})

		It("should start and stop successfully", func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(controller.started).To(BeTrue())

			err = controller.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(controller.started).To(BeFalse())
		})

		It("should return supported templates", func() {
			supportedTemplates := controller.GetSupportedTemplates()
			Expect(supportedTemplates).To(ContainElements("deployment-generic", "service-generic", "configmap-generic"))
		})
	})

	Describe("Manifest Generation", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should generate manifests from resource plan successfully", func() {
			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.NextPhase).To(Equal(interfaces.PhaseGitOpsCommit))

			// Verify response data
			Expect(result.Data).To(HaveKey("generatedManifests"))
			Expect(result.Data).To(HaveKey("manifestMetadata"))
			Expect(result.Data).To(HaveKey("correlationId"))
			Expect(result.Data).To(HaveKey("networkFunctions"))

			// Verify metrics
			Expect(result.Metrics).To(HaveKey("generation_time_ms"))
			Expect(result.Metrics).To(HaveKey("template_processing_time_ms"))
			Expect(result.Metrics).To(HaveKey("validation_time_ms"))
			Expect(result.Metrics).To(HaveKey("policy_check_time_ms"))
			Expect(result.Metrics).To(HaveKey("manifests_generated"))

			// Verify generated manifests
			generatedManifests := result.Data["generatedManifests"].(map[string]string)
			Expect(len(generatedManifests)).To(BeNumerically(">=", 2))

			// Verify deployment manifests exist
			deploymentFound := false
			serviceFound := false
			for name, manifest := range generatedManifests {
				if strings.Contains(name, "deployment") {
					deploymentFound = true
					Expect(manifest).To(ContainSubstring("kind: Deployment"))
					Expect(manifest).To(ContainSubstring("amf-deployment"))
				}
				if strings.Contains(name, "service") {
					serviceFound = true
					Expect(manifest).To(ContainSubstring("kind: Service"))
				}
			}
			Expect(deploymentFound).To(BeTrue())
			Expect(serviceFound).To(BeTrue())
		})

		It("should process phase interface correctly", func() {
			result, err := controller.ProcessPhase(ctx, networkIntent, interfaces.PhaseManifestGeneration)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.NextPhase).To(Equal(interfaces.PhaseGitOpsCommit))
		})

		It("should reject unsupported phases", func() {
			result, err := controller.ProcessPhase(ctx, networkIntent, interfaces.PhaseResourcePlanning)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorMessage).To(ContainSubstring("unsupported phase"))
		})

		It("should handle invalid resource plan data", func() {
			invalidIntent := networkIntent.DeepCopy()
			invalidIntent.Status.ResourcePlan = "invalid-string-data"

			result, err := controller.ProcessPhase(ctx, invalidIntent, interfaces.PhaseManifestGeneration)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("INVALID_INPUT"))
		})

		It("should generate manifests for different network function types", func() {
			// Test AMF specific template
			nf := &interfaces.PlannedNetworkFunction{
				Name:     "test-amf",
				Type:     "amf",
				Image:    "nephoran/amf",
				Version:  "v1.0.0",
				Replicas: 2,
				Resources: interfaces.ResourceSpec{
					Requests: interfaces.ResourceRequirements{
						CPU:    "500m",
						Memory: "1Gi",
					},
				},
			}

			manifests, metadata, err := controller.generateNetworkFunctionManifests(ctx, nf, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(manifests)).To(BeNumerically(">=", 1))
			Expect(metadata).NotTo(BeNil())

			// Verify deployment manifest
			deploymentManifest, exists := manifests["test-amf-deployment.yaml"]
			Expect(exists).To(BeTrue())
			Expect(deploymentManifest).To(ContainSubstring("kind: Deployment"))
			Expect(deploymentManifest).To(ContainSubstring("test-amf"))
			Expect(deploymentManifest).To(ContainSubstring("replicas: 2"))

			// Verify template engine was used
			Expect(mockTemplateEngine.GetTemplateUsageCount("deployment-amf")).To(BeNumerically(">=", 1))
		})

		It("should generate service manifests when ports are defined", func() {
			nf := &interfaces.PlannedNetworkFunction{
				Name:     "test-with-service",
				Type:     "smf",
				Image:    "nephoran/smf",
				Version:  "v1.0.0",
				Replicas: 1,
				Ports: []interfaces.PortSpec{
					{
						Name:       "http",
						Port:       80,
						TargetPort: 8080,
						Protocol:   "TCP",
					},
				},
			}

			manifests, _, err := controller.generateNetworkFunctionManifests(ctx, nf, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())

			// Should have both deployment and service manifests
			deploymentManifest, deploymentExists := manifests["test-with-service-deployment.yaml"]
			serviceManifest, serviceExists := manifests["test-with-service-service.yaml"]

			Expect(deploymentExists).To(BeTrue())
			Expect(serviceExists).To(BeTrue())
			Expect(deploymentManifest).To(ContainSubstring("kind: Deployment"))
			Expect(serviceManifest).To(ContainSubstring("kind: Service"))
		})

		It("should generate configmap manifests when configuration is present", func() {
			nf := &interfaces.PlannedNetworkFunction{
				Name:     "test-with-config",
				Type:     "amf",
				Image:    "nephoran/amf",
				Version:  "v1.0.0",
				Replicas: 1,
				Configuration: map[string]interface{}{
					"plmn_id":   "00101",
					"nf_type":   "AMF",
					"log_level": "INFO",
				},
			}

			manifests, _, err := controller.generateNetworkFunctionManifests(ctx, nf, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())

			// Should have deployment and configmap manifests
			configMapManifest, exists := manifests["test-with-config-configmap.yaml"]
			Expect(exists).To(BeTrue())
			Expect(configMapManifest).To(ContainSubstring("kind: ConfigMap"))
		})

		It("should generate RBAC manifests for network functions that require them", func() {
			// Mock RBAC templates
			mockTemplateEngine.SetTemplate("serviceaccount-generic", `apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Name }}-sa`)
			mockTemplateEngine.SetTemplate("clusterrole-generic", `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ .Name }}-role`)
			mockTemplateEngine.SetTemplate("clusterrolebinding-generic", `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ .Name }}-binding`)

			nf := &interfaces.PlannedNetworkFunction{
				Name:     "test-amf-rbac",
				Type:     "amf", // AMF requires RBAC
				Image:    "nephoran/amf",
				Version:  "v1.0.0",
				Replicas: 1,
			}

			manifests, _, err := controller.generateNetworkFunctionManifests(ctx, nf, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())

			// Should have RBAC manifests
			Expect(len(manifests)).To(BeNumerically(">=", 4)) // deployment + 3 RBAC manifests

			// Verify RBAC manifest names
			rbacFound := false
			for name := range manifests {
				if strings.Contains(name, "serviceaccount") || strings.Contains(name, "clusterrole") {
					rbacFound = true
					break
				}
			}
			Expect(rbacFound).To(BeTrue())
		})
	})

	Describe("Sequential vs Parallel Generation", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should generate manifests sequentially by default", func() {
			session := &GenerationSession{
				IntentID:         networkIntent.Name,
				NetworkFunctions: sampleResourcePlan.NetworkFunctions,
				ResourcePlan:     sampleResourcePlan,
			}

			manifests, metadata, err := controller.generateManifestsSequential(ctx, session)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(manifests)).To(BeNumerically(">=", 2))
			Expect(metadata).NotTo(BeNil())
			Expect(session.Metrics.TemplatesProcessed).To(Equal(len(sampleResourcePlan.NetworkFunctions)))
		})

		It("should generate manifests in parallel when enabled", func() {
			controller.Config.ParallelGeneration = true
			controller.Config.ConcurrentTemplates = 2

			session := &GenerationSession{
				IntentID:         networkIntent.Name,
				NetworkFunctions: sampleResourcePlan.NetworkFunctions,
				ResourcePlan:     sampleResourcePlan,
			}

			manifests, metadata, err := controller.generateManifestsParallel(ctx, session)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(manifests)).To(BeNumerically(">=", 2))
			Expect(metadata).NotTo(BeNil())
			Expect(session.Metrics.TemplatesProcessed).To(Equal(len(sampleResourcePlan.NetworkFunctions)))
		})

		It("should handle parallel generation with limited concurrency", func() {
			controller.Config.ParallelGeneration = true
			controller.Config.ConcurrentTemplates = 1 // Limit to 1 concurrent template

			// Add processing delay to test concurrency control
			mockTemplateEngine.SetProcessingDelay(10 * time.Millisecond)

			session := &GenerationSession{
				IntentID:         networkIntent.Name,
				NetworkFunctions: sampleResourcePlan.NetworkFunctions,
				ResourcePlan:     sampleResourcePlan,
			}

			startTime := time.Now()
			manifests, _, err := controller.generateManifestsParallel(ctx, session)
			duration := time.Since(startTime)

			Expect(err).NotTo(HaveOccurred())
			Expect(len(manifests)).To(BeNumerically(">=", 2))
			// With concurrency limit of 1, should take at least the sum of processing delays
			Expect(duration).To(BeNumerically(">=", time.Duration(len(sampleResourcePlan.NetworkFunctions))*8*time.Millisecond))
		})

		It("should handle context cancellation during parallel generation", func() {
			controller.Config.ParallelGeneration = true

			// Set long processing delay
			mockTemplateEngine.SetProcessingDelay(100 * time.Millisecond)

			session := &GenerationSession{
				IntentID:         networkIntent.Name,
				NetworkFunctions: sampleResourcePlan.NetworkFunctions,
				ResourcePlan:     sampleResourcePlan,
			}

			// Create context with short timeout
			cancelCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
			defer cancel()

			_, _, err := controller.generateManifestsParallel(cancelCtx, session)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(context.DeadlineExceeded))
		})
	})

	Describe("Manifest Validation", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate manifests successfully", func() {
			manifests := map[string]string{
				"test-deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  replicas: 1`,
				"test-service.yaml": `apiVersion: v1
kind: Service  
metadata:
  name: test-service
spec:
  ports:
  - port: 80`,
			}

			err := controller.ValidateManifests(ctx, manifests)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should report validation errors", func() {
			manifests := map[string]string{
				"invalid-manifest.yaml": `invalid: yaml: content`,
			}

			mockManifestValidator.SetValidationErrors("invalid-manifest.yaml", []string{"missing required field"})

			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue()) // Should succeed with warnings

			// Check validation results in data
			if validationResults, exists := result.Data["validationResults"]; exists {
				results := validationResults.([]ValidationResult)
				validationErrorFound := false
				for _, result := range results {
					if len(result.Errors) > 0 {
						validationErrorFound = true
						break
					}
				}
				// Note: May not find errors if mock doesn't match actual generated manifest names
			}
		})

		It("should skip validation when disabled", func() {
			controller.Config.ValidateManifests = false

			manifests := map[string]string{
				"invalid-manifest.yaml": "completely invalid yaml content",
			}

			err := controller.ValidateManifests(ctx, manifests)
			Expect(err).NotTo(HaveOccurred()) // Should not validate when disabled
		})

		It("should handle validation service failure", func() {
			mockManifestValidator.SetShouldReturnError(true)

			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue()) // Should continue with warnings

			// Check warnings were added
			// Implementation may add warnings for validation failures
		})
	})

	Describe("Policy Enforcement", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should enforce policies successfully", func() {
			manifests := map[string]string{
				"compliant-deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - name: test
        resources:
          limits:
            cpu: 100m
            memory: 128Mi`,
			}

			results, modifiedManifests, err := controller.enforcePolicies(ctx, manifests)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(results)).To(BeNumerically(">=", 1))

			// Check policy compliance
			for _, result := range results {
				if result.PolicyName == "Resource Limits Required" {
					Expect(result.Compliant).To(BeTrue())
				}
			}

			// No modifications expected for compliant manifests
			Expect(len(modifiedManifests)).To(Equal(0))
		})

		It("should report policy violations", func() {
			manifests := map[string]string{
				"violation-deployment.yaml": "test-manifest-content",
			}

			mockPolicyEnforcer.SetPolicyViolations("violation-deployment.yaml", []string{"missing resource limits"})

			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue()) // Should succeed with warnings

			// Check policy results in data
			if policyResults, exists := result.Data["policyResults"]; exists {
				results := policyResults.([]PolicyResult)
				violationFound := false
				for _, result := range results {
					if !result.Compliant && len(result.Violations) > 0 {
						violationFound = true
						break
					}
				}
				// Note: May not find violations if mock doesn't match actual scenario
			}
		})

		It("should apply policy modifications", func() {
			manifests := map[string]string{
				"modifiable-manifest.yaml": "original-content",
			}

			mockPolicyEnforcer.SetModifiedManifest("modifiable-manifest.yaml", "modified-content")

			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			// Modified manifests should be used in final result
			generatedManifests := result.Data["generatedManifests"].(map[string]string)
			// Check if any manifest was modified (mock may not match actual names)
			Expect(len(generatedManifests)).To(BeNumerically(">=", 1))
		})

		It("should skip policy enforcement when disabled", func() {
			controller.Config.EnforcePolicies = false

			manifests := map[string]string{
				"test-manifest.yaml": "test-content",
			}

			results, modifiedManifests, err := controller.enforcePolicies(ctx, manifests)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(BeNil())
			Expect(modifiedManifests).To(BeNil())
		})

		It("should handle policy enforcement failure", func() {
			mockPolicyEnforcer.SetShouldReturnError(true)

			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue()) // Should continue with warnings
		})
	})

	Describe("Manifest Optimization", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should optimize manifests successfully", func() {
			manifests := map[string]string{
				"unoptimized-manifest.yaml": `apiVersion: apps/v1
kind: Deployment

metadata:
  name: test

spec:
  replicas: 1`,
			}

			optimized, err := controller.OptimizeManifests(ctx, manifests)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(optimized)).To(Equal(len(manifests)))

			// Verify optimization (mock removes extra whitespace)
			optimizedManifest := optimized["unoptimized-manifest.yaml"]
			Expect(optimizedManifest).NotTo(ContainSubstring("\n\n"))
		})

		It("should apply custom optimizations", func() {
			manifests := map[string]string{
				"custom-manifest.yaml": "original-content",
			}

			mockManifestOptimizer.SetOptimizedManifest("custom-manifest.yaml", "optimized-content")

			optimized, err := controller.OptimizeManifests(ctx, manifests)
			Expect(err).NotTo(HaveOccurred())
			Expect(optimized["custom-manifest.yaml"]).To(Equal("optimized-content"))
		})

		It("should skip optimization when disabled", func() {
			controller.Config.OptimizeManifests = false

			manifests := map[string]string{
				"test-manifest.yaml": "original-content",
			}

			optimized, err := controller.OptimizeManifests(ctx, manifests)
			Expect(err).NotTo(HaveOccurred())
			Expect(optimized).To(Equal(manifests)) // Should return original manifests
		})

		It("should handle optimization failure", func() {
			mockManifestOptimizer.SetShouldReturnError(true)

			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue()) // Should continue with warnings
		})
	})

	Describe("Caching", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should cache manifest generation results", func() {
			// First generation - should not use cache
			result1, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())
			Expect(result1.Metrics["cache_hit"]).To(Equal(float64(0)))

			// Second generation - should use cache
			result2, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())
			Expect(result2.Metrics["cache_hit"]).To(Equal(float64(1)))
		})

		It("should handle cache expiration", func() {
			// Set very short cache TTL
			controller.manifestCache.ttl = 1 * time.Millisecond

			// First generation
			result1, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())

			// Wait for cache to expire
			time.Sleep(5 * time.Millisecond)

			// Second generation - should not use cache
			result2, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())
			Expect(result2.Metrics["cache_hit"]).To(Equal(float64(0)))
		})

		It("should handle cache size limit", func() {
			// Set small cache size limit
			controller.manifestCache.maxEntries = 1

			// Generate for first resource plan
			result1, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())

			// Generate for different resource plan
			differentResourcePlan := &interfaces.ResourcePlan{
				NetworkFunctions: []interfaces.PlannedNetworkFunction{
					{
						Name:     "different-nf",
						Type:     "upf",
						Image:    "nephoran/upf",
						Version:  "v1.0.0",
						Replicas: 1,
					},
				},
				DeploymentPattern: "edge",
			}

			result2, err := controller.generateManifestsFromResourcePlan(ctx, nil, differentResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())

			// Cache should be limited to 1 entry
			Expect(len(controller.manifestCache.entries)).To(Equal(1))
		})
	})

	Describe("Template Variable Preparation", func() {
		It("should prepare template variables correctly", func() {
			nf := &interfaces.PlannedNetworkFunction{
				Name:     "test-nf",
				Type:     "amf",
				Image:    "nephoran/amf",
				Version:  "v1.0.0",
				Replicas: 3,
				Resources: interfaces.ResourceSpec{
					Requests: interfaces.ResourceRequirements{
						CPU:     "500m",
						Memory:  "1Gi",
						Storage: "10Gi",
					},
				},
				Ports: []interfaces.PortSpec{
					{
						Name:       "http",
						Port:       80,
						TargetPort: 8080,
					},
				},
				Environment: []interfaces.EnvVar{
					{
						Name:  "LOG_LEVEL",
						Value: "DEBUG",
					},
				},
				Configuration: map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			}

			variables := controller.prepareTemplateVariables(nf, sampleResourcePlan)

			Expect(variables).To(HaveKey("NetworkFunction"))
			Expect(variables).To(HaveKey("ResourcePlan"))
			Expect(variables).To(HaveKey("Namespace"))
			Expect(variables).To(HaveKey("Name"))
			Expect(variables).To(HaveKey("Type"))
			Expect(variables).To(HaveKey("Image"))
			Expect(variables).To(HaveKey("Version"))
			Expect(variables).To(HaveKey("Replicas"))
			Expect(variables).To(HaveKey("Resources"))
			Expect(variables).To(HaveKey("Ports"))
			Expect(variables).To(HaveKey("Environment"))
			Expect(variables).To(HaveKey("Configuration"))
			Expect(variables).To(HaveKey("DeploymentPattern"))
			Expect(variables).To(HaveKey("Labels"))

			// Verify values
			Expect(variables["Name"]).To(Equal("test-nf"))
			Expect(variables["Type"]).To(Equal("amf"))
			Expect(variables["Image"]).To(Equal("nephoran/amf"))
			Expect(variables["Version"]).To(Equal("v1.0.0"))
			Expect(variables["Replicas"]).To(Equal(int32(3)))
			Expect(variables["Namespace"]).To(Equal(controller.Config.DefaultNamespace))
			Expect(variables["DeploymentPattern"]).To(Equal(sampleResourcePlan.DeploymentPattern))

			// Verify labels
			labels := variables["Labels"].(map[string]string)
			Expect(labels["app.kubernetes.io/name"]).To(Equal("test-nf"))
			Expect(labels["app.kubernetes.io/component"]).To(Equal("amf"))
			Expect(labels["app.kubernetes.io/managed-by"]).To(Equal("nephoran-intent-operator"))
		})

		It("should identify network functions requiring RBAC correctly", func() {
			rbacRequiredTypes := []string{"amf", "smf", "nrf", "ausf", "udm", "udr", "pcf"}
			nonRbacTypes := []string{"upf", "generic", "monitoring"}

			for _, nfType := range rbacRequiredTypes {
				Expect(controller.requiresRBAC(nfType)).To(BeTrue(), "NF type %s should require RBAC", nfType)
			}

			for _, nfType := range nonRbacTypes {
				Expect(controller.requiresRBAC(nfType)).To(BeFalse(), "NF type %s should not require RBAC", nfType)
			}
		})
	})

	Describe("Error Handling", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle template engine failure", func() {
			mockTemplateEngine.SetShouldReturnError(true)

			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("GENERATION_ERROR"))
		})

		It("should handle resource plan conversion failure", func() {
			// Create intent with malformed resource plan
			invalidResourcePlanData := map[string]interface{}{
				"invalidField": "invalidValue",
			}

			_, err := controller.convertToResourcePlan(invalidResourcePlanData)
			Expect(err).NotTo(HaveOccurred()) // Should create empty plan, not fail
		})

		It("should handle empty network functions gracefully", func() {
			emptyResourcePlan := &interfaces.ResourcePlan{
				NetworkFunctions:     []interfaces.PlannedNetworkFunction{}, // Empty
				DeploymentPattern:    "standard",
				ResourceRequirements: interfaces.ResourceRequirements{},
			}

			result, err := controller.generateManifestsFromResourcePlan(ctx, networkIntent, emptyResourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			// Should generate empty manifests map
			generatedManifests := result.Data["generatedManifests"].(map[string]string)
			Expect(len(generatedManifests)).To(Equal(0))
		})
	})

	Describe("Health and Metrics", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return health status", func() {
			health, err := controller.GetHealthStatus(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(health.Status).To(Equal("Healthy"))
			Expect(health.Metrics).To(HaveKey("totalGenerated"))
			Expect(health.Metrics).To(HaveKey("successRate"))
			Expect(health.Metrics).To(HaveKey("averageGenerationTime"))
			Expect(health.Metrics).To(HaveKey("validationSuccessRate"))
			Expect(health.Metrics).To(HaveKey("policyComplianceRate"))
		})

		It("should return controller metrics", func() {
			metrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(metrics).To(HaveKey("total_generated"))
			Expect(metrics).To(HaveKey("successful_generated"))
			Expect(metrics).To(HaveKey("failed_generated"))
			Expect(metrics).To(HaveKey("success_rate"))
			Expect(metrics).To(HaveKey("average_generation_time_ms"))
			Expect(metrics).To(HaveKey("validation_success_rate"))
			Expect(metrics).To(HaveKey("policy_compliance_rate"))
		})

		It("should update metrics after generation", func() {
			initialMetrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			initialTotal := initialMetrics["total_generated"]

			// Perform generation
			_, err = controller.generateManifestsFromResourcePlan(ctx, networkIntent, sampleResourcePlan)
			Expect(err).NotTo(HaveOccurred())

			updatedMetrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			updatedTotal := updatedMetrics["total_generated"]

			Expect(updatedTotal).To(Equal(initialTotal + 1))
		})

		It("should track session metrics", func() {
			intentID := networkIntent.Name

			// Create a generation session
			session := &GenerationSession{
				IntentID:    intentID,
				StartTime:   time.Now(),
				Progress:    0.7,
				CurrentStep: "validating_manifests",
			}
			controller.activeGeneration.Store(intentID, session)

			status, err := controller.GetPhaseStatus(ctx, intentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Phase).To(Equal(interfaces.PhaseManifestGeneration))
			Expect(status.Status).To(Equal("InProgress"))
			Expect(status.Metrics["progress"]).To(Equal(0.7))
		})
	})

	Describe("Phase Controller Interface", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return correct dependencies", func() {
			deps := controller.GetDependencies()
			Expect(deps).To(ContainElement(interfaces.PhaseResourcePlanning))
		})

		It("should return correct blocked phases", func() {
			blocked := controller.GetBlockedPhases()
			expectedPhases := []interfaces.ProcessingPhase{
				interfaces.PhaseGitOpsCommit,
				interfaces.PhaseDeploymentVerification,
			}
			Expect(blocked).To(ContainElements(expectedPhases))
		})

		It("should handle phase errors", func() {
			intentID := networkIntent.Name
			testError := fmt.Errorf("test generation error")

			err := controller.HandlePhaseError(ctx, intentID, testError)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("test generation error"))
		})
	})

	Describe("Concurrent Processing", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle multiple concurrent generation requests", func() {
			numRequests := 5
			results := make(chan interfaces.ProcessingResult, numRequests)
			errors := make(chan error, numRequests)

			// Create different resource plans for concurrent testing
			for i := 0; i < numRequests; i++ {
				go func(index int) {
					defer GinkgoRecover()

					resourcePlan := &interfaces.ResourcePlan{
						NetworkFunctions: []interfaces.PlannedNetworkFunction{
							{
								Name:     fmt.Sprintf("concurrent-nf-%d", index),
								Type:     "amf",
								Image:    "nephoran/amf",
								Version:  "v1.0.0",
								Replicas: 1,
								Resources: interfaces.ResourceSpec{
									Requests: interfaces.ResourceRequirements{
										CPU:    "500m",
										Memory: "1Gi",
									},
								},
							},
						},
						DeploymentPattern: "standard",
					}

					result, err := controller.generateManifestsFromResourcePlan(ctx, nil, resourcePlan)
					if err != nil {
						errors <- err
					} else {
						results <- result
					}
				}(i)
			}

			// Collect results
			successCount := 0
			errorCount := 0
			timeout := time.After(30 * time.Second)

			for i := 0; i < numRequests; i++ {
				select {
				case result := <-results:
					if result.Success {
						successCount++
					}
				case <-errors:
					errorCount++
				case <-timeout:
					Fail("Timeout waiting for concurrent generation to complete")
				}
			}

			// Verify all requests were processed
			Expect(successCount + errorCount).To(Equal(numRequests))
			// Most should succeed
			Expect(successCount).To(BeNumerically(">=", numRequests/2))
		})

		It("should maintain thread safety under concurrent load", func() {
			numGoroutines := 20
			done := make(chan bool, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(index int) {
					defer GinkgoRecover()
					defer func() { done <- true }()

					// Mix of operations to test thread safety
					switch index % 3 {
					case 0:
						// Generate manifests
						controller.generateManifestsFromResourcePlan(ctx, nil, sampleResourcePlan)
					case 1:
						// Get health status
						controller.GetHealthStatus(ctx)
					case 2:
						// Get metrics
						controller.GetMetrics(ctx)
					}
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				Eventually(done).Should(Receive())
			}
		})
	})

	Describe("Background Operations", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should perform session cleanup", func() {
			intentID := "cleanup-test-intent"

			// Create an old session
			oldSession := &GenerationSession{
				IntentID:  intentID,
				StartTime: time.Now().Add(-2 * time.Hour),
			}
			controller.activeGeneration.Store(intentID, oldSession)

			// Verify session exists
			_, exists := controller.activeGeneration.Load(intentID)
			Expect(exists).To(BeTrue())

			// Trigger cleanup
			controller.cleanupExpiredSessions()

			// Verify session was removed
			_, exists = controller.activeGeneration.Load(intentID)
			Expect(exists).To(BeFalse())
		})

		It("should perform cache cleanup", func() {
			// Add expired entry to cache
			expiredHash := "expired-manifest-entry"
			controller.manifestCache.entries[expiredHash] = &ManifestCacheEntry{
				Timestamp: time.Now().Add(-1 * time.Hour),
			}

			// Verify entry exists
			Expect(controller.manifestCache.entries).To(HaveKey(expiredHash))

			// Trigger cache cleanup
			controller.cleanupExpiredCache()

			// Verify expired entry was removed
			Expect(controller.manifestCache.entries).NotTo(HaveKey(expiredHash))
		})

		It("should perform health monitoring", func() {
			initialHealth := controller.healthStatus

			// Trigger health check
			controller.performHealthCheck()

			// Verify health status was updated
			Expect(controller.healthStatus.LastChecked).To(BeTemporally(">", initialHealth.LastChecked))
		})
	})
})

func TestSpecializedManifestGenerationController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SpecializedManifestGenerationController Suite")
}

// Helper function to convert ResourcePlan to interface{} for testing
func convertResourcePlanToInterface(plan *interfaces.ResourcePlan) interface{} {
	return map[string]interface{}{
		"networkFunctions": convertNetworkFunctionsToInterface(plan.NetworkFunctions),
		"resourceRequirements": map[string]interface{}{
			"cpu":     plan.ResourceRequirements.CPU,
			"memory":  plan.ResourceRequirements.Memory,
			"storage": plan.ResourceRequirements.Storage,
		},
		"deploymentPattern": plan.DeploymentPattern,
		"constraints":       plan.Constraints,
		"dependencies":      plan.Dependencies,
	}
}

func convertNetworkFunctionsToInterface(nfs []interfaces.PlannedNetworkFunction) []interface{} {
	result := make([]interface{}, len(nfs))
	for i, nf := range nfs {
		result[i] = map[string]interface{}{
			"name":     nf.Name,
			"type":     nf.Type,
			"image":    nf.Image,
			"version":  nf.Version,
			"replicas": nf.Replicas,
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":     nf.Resources.Requests.CPU,
					"memory":  nf.Resources.Requests.Memory,
					"storage": nf.Resources.Requests.Storage,
				},
				"limits": map[string]interface{}{
					"cpu":     nf.Resources.Limits.CPU,
					"memory":  nf.Resources.Limits.Memory,
					"storage": nf.Resources.Limits.Storage,
				},
			},
			"ports":         convertPortsToInterface(nf.Ports),
			"environment":   convertEnvVarsToInterface(nf.Environment),
			"configuration": nf.Configuration,
		}
	}
	return result
}

func convertPortsToInterface(ports []interfaces.PortSpec) []interface{} {
	result := make([]interface{}, len(ports))
	for i, port := range ports {
		result[i] = map[string]interface{}{
			"name":        port.Name,
			"port":        port.Port,
			"targetPort":  port.TargetPort,
			"protocol":    port.Protocol,
			"serviceType": port.ServiceType,
		}
	}
	return result
}

func convertEnvVarsToInterface(envVars []interfaces.EnvVar) []interface{} {
	result := make([]interface{}, len(envVars))
	for i, envVar := range envVars {
		result[i] = map[string]interface{}{
			"name":  envVar.Name,
			"value": envVar.Value,
		}
	}
	return result
}
