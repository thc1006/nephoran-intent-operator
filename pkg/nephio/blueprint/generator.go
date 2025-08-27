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

package blueprint

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"text/template"

	"github.com/Masterminds/sprig/v3"
	"go.uber.org/zap"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// Generator handles blueprint package generation from NetworkIntents
type Generator struct {
	config        *BlueprintConfig
	logger        *zap.Logger
	llmClient     *llm.Client
	httpClient    *http.Client
	templateCache sync.Map

	// O-RAN specific templates
	oranTemplates        map[string]*template.Template
	coreTemplates        map[string]*template.Template
	edgeTemplates        map[string]*template.Template
	sliceTemplates       map[string]*template.Template
	serviceMeshTemplates map[string]*template.Template

	// Performance optimization
	resourcePool    sync.Pool
	generationMutex sync.RWMutex
}

// GenerationContext contains context for blueprint generation
type GenerationContext struct {
	Intent          *v1.NetworkIntent
	LLMOutput       map[string]interface{}
	TargetCluster   string
	TargetNamespace string
	NetworkSlice    string
	ComponentType   v1.NetworkTargetComponent
	DeploymentMode  string
	SecurityProfile string
	ResourceProfile string
}

// NewGenerator creates a new blueprint generator
func NewGenerator(config *BlueprintConfig, logger *zap.Logger) (*Generator, error) {
	if config == nil {
		config = DefaultBlueprintConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	// Initialize LLM client
	llmClient := llm.NewClient(config.LLMEndpoint)

	generator := &Generator{
		config:               config,
		logger:               logger,
		llmClient:            llmClient,
		httpClient:           &http.Client{Timeout: 30 * time.Second},
		oranTemplates:        make(map[string]*template.Template),
		coreTemplates:        make(map[string]*template.Template),
		edgeTemplates:        make(map[string]*template.Template),
		sliceTemplates:       make(map[string]*template.Template),
		serviceMeshTemplates: make(map[string]*template.Template),
		resourcePool: sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{})
			},
		},
	}

	// Initialize templates
	if err := generator.initializeTemplates(); err != nil {
		return nil, fmt.Errorf("failed to initialize templates: %w", err)
	}

	return generator, nil
}

// GenerateFromNetworkIntent generates blueprint files from NetworkIntent
func (g *Generator) GenerateFromNetworkIntent(ctx context.Context, intent *v1.NetworkIntent) (map[string]string, error) {
	startTime := time.Now()

	g.logger.Info("Generating blueprint from NetworkIntent",
		zap.String("intent_name", intent.Name),
		zap.String("intent_type", string(intent.Spec.IntentType)),
		zap.Strings("target_components", g.targetComponentsToStrings(intent.Spec.TargetComponents)))

	// Step 1: Process intent with LLM to extract parameters
	llmOutput, err := g.processIntentWithLLM(ctx, intent)
	if err != nil {
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}

	// Step 2: Create generation context
	genCtx := &GenerationContext{
		Intent:          intent,
		LLMOutput:       llmOutput,
		TargetCluster:   intent.Spec.TargetCluster,
		TargetNamespace: intent.Spec.TargetNamespace,
		NetworkSlice:    intent.Spec.NetworkSlice,
		DeploymentMode:  g.extractDeploymentMode(llmOutput),
		SecurityProfile: g.extractSecurityProfile(llmOutput),
		ResourceProfile: g.extractResourceProfile(llmOutput),
	}

	if genCtx.TargetNamespace == "" {
		genCtx.TargetNamespace = g.config.DefaultNamespace
	}

	// Step 3: Generate blueprint files based on intent type and target components
	files := make(map[string]string)

	// Generate core blueprint structure
	if err := g.generateCoreStructure(ctx, genCtx, files); err != nil {
		return nil, fmt.Errorf("core structure generation failed: %w", err)
	}

	// Generate component-specific blueprints
	for _, component := range intent.Spec.TargetComponents {
		genCtx.ComponentType = component
		if err := g.generateComponentBlueprint(ctx, genCtx, component, files); err != nil {
			return nil, fmt.Errorf("component blueprint generation failed for %s: %w", component, err)
		}
	}

	// Generate network slice configuration if applicable
	if genCtx.NetworkSlice != "" {
		if err := g.generateNetworkSliceConfiguration(ctx, genCtx, files); err != nil {
			return nil, fmt.Errorf("network slice configuration generation failed: %w", err)
		}
	}

	// Generate service mesh configuration
	if g.shouldGenerateServiceMesh(genCtx) {
		if err := g.generateServiceMeshConfiguration(ctx, genCtx, files); err != nil {
			return nil, fmt.Errorf("service mesh configuration generation failed: %w", err)
		}
	}

	// Generate monitoring and observability configuration
	if err := g.generateObservabilityConfiguration(ctx, genCtx, files); err != nil {
		return nil, fmt.Errorf("observability configuration generation failed: %w", err)
	}

	duration := time.Since(startTime)
	g.logger.Info("Successfully generated blueprint",
		zap.String("intent_name", intent.Name),
		zap.Duration("duration", duration),
		zap.Int("files_generated", len(files)))

	return files, nil
}

// processIntentWithLLM processes the NetworkIntent with LLM to extract structured parameters
func (g *Generator) processIntentWithLLM(ctx context.Context, intent *v1.NetworkIntent) (map[string]interface{}, error) {
	prompt := g.buildLLMPrompt(intent)

	response, err := g.llmClient.ProcessIntent(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}

	// Parse LLM response as JSON
	var llmOutput map[string]interface{}
	if err := json.Unmarshal([]byte(response), &llmOutput); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return llmOutput, nil
}

// buildLLMPrompt creates a specialized prompt for blueprint generation
func (g *Generator) buildLLMPrompt(intent *v1.NetworkIntent) string {
	return fmt.Sprintf(`
You are an expert in O-RAN and 5G Core network function deployment. 
Analyze the following NetworkIntent and extract structured parameters for blueprint generation.

Intent: %s
Intent Type: %s
Target Components: %s
Priority: %s

Provide a JSON response with the following structure:
{
  "deployment_config": {
    "replicas": <number>,
    "image": "<container_image>",
    "resources": {
      "requests": {"cpu": "<value>", "memory": "<value>"},
      "limits": {"cpu": "<value>", "memory": "<value>"}
    },
    "env_vars": [{"name": "<name>", "value": "<value>"}]
  },
  "network_function": {
    "type": "<nf_type>",
    "interfaces": ["<interface1>", "<interface2>"],
    "configuration": {}
  },
  "oran_interfaces": {
    "a1_enabled": <boolean>,
    "o1_enabled": <boolean>,
    "o2_enabled": <boolean>,
    "e2_enabled": <boolean>
  },
  "performance_requirements": {
    "throughput": "<value>",
    "latency": "<value>",
    "availability": "<value>"
  },
  "security_policies": {
    "encryption": <boolean>,
    "authentication": "<type>",
    "network_policies": []
  },
  "service_mesh": {
    "enabled": <boolean>,
    "mtls_enabled": <boolean>,
    "ingress_enabled": <boolean>
  }
}

Ensure all values are appropriate for the specified intent and components.
`, intent.Spec.Intent, intent.Spec.IntentType,
		strings.Join(g.targetComponentsToStrings(intent.Spec.TargetComponents), ", "),
		intent.Spec.Priority)
}

// buildLLMContext creates context for LLM processing
func (g *Generator) buildLLMContext(intent *v1.NetworkIntent) map[string]interface{} {
	return map[string]interface{}{
		"intent_type":          intent.Spec.IntentType,
		"target_components":    intent.Spec.TargetComponents,
		"resource_constraints": intent.Spec.ResourceConstraints,
		"network_slice":        intent.Spec.NetworkSlice,
		"target_cluster":       intent.Spec.TargetCluster,
		"target_namespace":     intent.Spec.TargetNamespace,
		"priority":             intent.Spec.Priority,
	}
}

// generateCoreStructure generates the core blueprint structure files
func (g *Generator) generateCoreStructure(ctx context.Context, genCtx *GenerationContext, files map[string]string) error {
	// Generate Kptfile
	kptfile, err := g.generateKptfile(genCtx)
	if err != nil {
		return fmt.Errorf("failed to generate Kptfile: %w", err)
	}
	files["Kptfile"] = kptfile

	// Generate README
	readme, err := g.generateREADME(genCtx)
	if err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}
	files["README.md"] = readme

	// Generate package metadata
	metadata, err := g.generatePackageMetadata(genCtx)
	if err != nil {
		return fmt.Errorf("failed to generate package metadata: %w", err)
	}
	files["package-meta.yaml"] = metadata

	// Generate function configuration
	fnConfig, err := g.generateFunctionConfig(genCtx)
	if err != nil {
		return fmt.Errorf("failed to generate function config: %w", err)
	}
	files["fn-config.yaml"] = fnConfig

	return nil
}

// generateComponentBlueprint generates blueprint for specific O-RAN/5GC component
func (g *Generator) generateComponentBlueprint(ctx context.Context, genCtx *GenerationContext, component v1.NetworkTargetComponent, files map[string]string) error {
	switch component {
	// 5G Core Components
	case v1.NetworkTargetComponentAMF:
		return g.generateAMFBlueprint(genCtx, files)
	case v1.NetworkTargetComponentSMF:
		return g.generateSMFBlueprint(genCtx, files)
	case v1.NetworkTargetComponentUPF:
		return g.generateUPFBlueprint(genCtx, files)
	case v1.NetworkTargetComponentNSSF:
		return g.generateNSSFBlueprint(genCtx, files)

	// O-RAN Components (using string constants)
	case "O-DU":
		return g.generateODUBlueprint(genCtx, files)
	case "O-CU-CP":
		return g.generateOCUCPBlueprint(genCtx, files)
	case "O-CU-UP":
		return g.generateOCUUPBlueprint(genCtx, files)
	case "Near-RT-RIC":
		return g.generateNearRTRICBlueprint(genCtx, files)
	case "Non-RT-RIC":
		return g.generateNonRTRICBlueprint(genCtx, files)

	// RIC Applications
	case "xApp":
		return g.generateXAppBlueprint(genCtx, files)
	case "rApp":
		return g.generateRAppBlueprint(genCtx, files)

	default:
		return g.generateGenericBlueprint(genCtx, component, files)
	}
}

// generateAMFBlueprint generates AMF-specific blueprint
func (g *Generator) generateAMFBlueprint(genCtx *GenerationContext, files map[string]string) error {
	template := g.coreTemplates["amf"]
	if template == nil {
		return fmt.Errorf("AMF template not found")
	}

	data := g.buildAMFTemplateData(genCtx)
	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("AMF template execution failed: %w", err)
	}

	files["amf-deployment.yaml"] = content

	// Generate AMF service
	if serviceContent, err := g.generateComponentService(genCtx, "amf"); err == nil {
		files["amf-service.yaml"] = serviceContent
	}

	// Generate AMF configuration
	if configContent, err := g.generateAMFConfig(genCtx); err == nil {
		files["amf-config.yaml"] = configContent
	}

	return nil
}

// generateSMFBlueprint generates SMF-specific blueprint
func (g *Generator) generateSMFBlueprint(genCtx *GenerationContext, files map[string]string) error {
	template := g.coreTemplates["smf"]
	if template == nil {
		return fmt.Errorf("SMF template not found")
	}

	data := g.buildSMFTemplateData(genCtx)
	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("SMF template execution failed: %w", err)
	}

	files["smf-deployment.yaml"] = content

	// Generate SMF service and configuration
	if serviceContent, err := g.generateComponentService(genCtx, "smf"); err == nil {
		files["smf-service.yaml"] = serviceContent
	}

	if configContent, err := g.generateSMFConfig(genCtx); err == nil {
		files["smf-config.yaml"] = configContent
	}

	return nil
}

// generateUPFBlueprint generates UPF-specific blueprint
func (g *Generator) generateUPFBlueprint(genCtx *GenerationContext, files map[string]string) error {
	template := g.coreTemplates["upf"]
	if template == nil {
		return fmt.Errorf("UPF template not found")
	}

	data := g.buildUPFTemplateData(genCtx)
	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("UPF template execution failed: %w", err)
	}

	files["upf-deployment.yaml"] = content

	// UPF typically needs special networking configuration
	if networkContent, err := g.generateUPFNetworking(genCtx); err == nil {
		files["upf-networking.yaml"] = networkContent
	}

	return nil
}

// generateNearRTRICBlueprint generates Near-RT RIC blueprint
func (g *Generator) generateNearRTRICBlueprint(genCtx *GenerationContext, files map[string]string) error {
	template := g.oranTemplates["near-rt-ric"]
	if template == nil {
		return fmt.Errorf("Near-RT RIC template not found")
	}

	data := g.buildRICTemplateData(genCtx, "near-rt")
	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("Near-RT RIC template execution failed: %w", err)
	}

	files["near-rt-ric-deployment.yaml"] = content

	// Generate E2 interface configuration
	if e2Config, err := g.generateE2Configuration(genCtx); err == nil {
		files["e2-config.yaml"] = e2Config
	}

	// Generate xApp management configuration
	if xappConfig, err := g.generateXAppManagementConfig(genCtx); err == nil {
		files["xapp-mgmt-config.yaml"] = xappConfig
	}

	return nil
}

// generateXAppBlueprint generates xApp blueprint
func (g *Generator) generateXAppBlueprint(genCtx *GenerationContext, files map[string]string) error {
	template := g.oranTemplates["xapp"]
	if template == nil {
		return fmt.Errorf("xApp template not found")
	}

	data := g.buildXAppTemplateData(genCtx)
	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("xApp template execution failed: %w", err)
	}

	files["xapp-deployment.yaml"] = content

	// Generate xApp descriptor
	if descriptor, err := g.generateXAppDescriptor(genCtx); err == nil {
		files["xapp-descriptor.json"] = descriptor
	}

	return nil
}

// generateNetworkSliceConfiguration generates network slice configuration
func (g *Generator) generateNetworkSliceConfiguration(ctx context.Context, genCtx *GenerationContext, files map[string]string) error {
	template := g.sliceTemplates["network-slice"]
	if template == nil {
		return fmt.Errorf("network slice template not found")
	}

	data := g.buildNetworkSliceTemplateData(genCtx)
	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("network slice template execution failed: %w", err)
	}

	files["network-slice.yaml"] = content

	// Generate slice-specific QoS configuration
	if qosConfig, err := g.generateSliceQoSConfig(genCtx); err == nil {
		files["slice-qos-config.yaml"] = qosConfig
	}

	return nil
}

// generateServiceMeshConfiguration generates Istio service mesh configuration
func (g *Generator) generateServiceMeshConfiguration(ctx context.Context, genCtx *GenerationContext, files map[string]string) error {
	// Generate VirtualService
	if vsConfig, err := g.generateVirtualService(genCtx); err == nil {
		files["virtual-service.yaml"] = vsConfig
	}

	// Generate DestinationRule
	if drConfig, err := g.generateDestinationRule(genCtx); err == nil {
		files["destination-rule.yaml"] = drConfig
	}

	// Generate Gateway if ingress is needed
	if g.shouldGenerateGateway(genCtx) {
		if gwConfig, err := g.generateGateway(genCtx); err == nil {
			files["gateway.yaml"] = gwConfig
		}
	}

	// Generate PeerAuthentication for mTLS
	if paConfig, err := g.generatePeerAuthentication(genCtx); err == nil {
		files["peer-authentication.yaml"] = paConfig
	}

	return nil
}

// generateObservabilityConfiguration generates monitoring and observability configuration
func (g *Generator) generateObservabilityConfiguration(ctx context.Context, genCtx *GenerationContext, files map[string]string) error {
	// Generate ServiceMonitor for Prometheus
	if smConfig, err := g.generateServiceMonitor(genCtx); err == nil {
		files["service-monitor.yaml"] = smConfig
	}

	// Generate Grafana dashboard configuration
	if dashConfig, err := g.generateGrafanaDashboard(genCtx); err == nil {
		files["grafana-dashboard.json"] = dashConfig
	}

	// Generate alert rules
	if alertConfig, err := g.generateAlertRules(genCtx); err == nil {
		files["alert-rules.yaml"] = alertConfig
	}

	return nil
}

// Helper methods for template data building

func (g *Generator) buildAMFTemplateData(genCtx *GenerationContext) map[string]interface{} {
	deployConfig := g.extractDeploymentConfig(genCtx.LLMOutput)

	return map[string]interface{}{
		"Name":         fmt.Sprintf("%s-amf", genCtx.Intent.Name),
		"Namespace":    genCtx.TargetNamespace,
		"Image":        g.getOrDefault(deployConfig, "image", "free5gc/amf:v3.2.1").(string),
		"Replicas":     g.getOrDefault(deployConfig, "replicas", 1).(int),
		"Resources":    g.extractResources(deployConfig),
		"Environment":  g.extractEnvironment(deployConfig),
		"NetworkSlice": genCtx.NetworkSlice,
		"Config":       g.buildAMFConfig(genCtx),
	}
}

func (g *Generator) buildSMFTemplateData(genCtx *GenerationContext) map[string]interface{} {
	deployConfig := g.extractDeploymentConfig(genCtx.LLMOutput)

	return map[string]interface{}{
		"Name":        fmt.Sprintf("%s-smf", genCtx.Intent.Name),
		"Namespace":   genCtx.TargetNamespace,
		"Image":       g.getOrDefault(deployConfig, "image", "free5gc/smf:v3.2.1").(string),
		"Replicas":    g.getOrDefault(deployConfig, "replicas", 1).(int),
		"Resources":   g.extractResources(deployConfig),
		"Environment": g.extractEnvironment(deployConfig),
		"Config":      g.buildSMFConfig(genCtx),
	}
}

func (g *Generator) buildUPFTemplateData(genCtx *GenerationContext) map[string]interface{} {
	deployConfig := g.extractDeploymentConfig(genCtx.LLMOutput)

	return map[string]interface{}{
		"Name":          fmt.Sprintf("%s-upf", genCtx.Intent.Name),
		"Namespace":     genCtx.TargetNamespace,
		"Image":         g.getOrDefault(deployConfig, "image", "free5gc/upf:v3.2.1").(string),
		"Replicas":      g.getOrDefault(deployConfig, "replicas", 1).(int),
		"Resources":     g.extractResources(deployConfig),
		"Environment":   g.extractEnvironment(deployConfig),
		"Privileged":    true, // UPF typically needs privileged access
		"NetworkConfig": g.buildUPFNetworkConfig(genCtx),
	}
}

// Template execution helper
func (g *Generator) executeTemplate(tmpl *template.Template, data interface{}) (string, error) {
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Configuration extraction helpers
func (g *Generator) extractDeploymentConfig(llmOutput map[string]interface{}) map[string]interface{} {
	if config, ok := llmOutput["deployment_config"].(map[string]interface{}); ok {
		return config
	}
	return make(map[string]interface{})
}

func (g *Generator) extractSecurityProfile(llmOutput map[string]interface{}) string {
	if security, ok := llmOutput["security_policies"].(map[string]interface{}); ok {
		if profile, ok := security["profile"].(string); ok {
			return profile
		}
	}
	return "default"
}

func (g *Generator) extractResourceProfile(llmOutput map[string]interface{}) string {
	if perf, ok := llmOutput["performance_requirements"].(map[string]interface{}); ok {
		if profile, ok := perf["profile"].(string); ok {
			return profile
		}
	}
	return "medium"
}

func (g *Generator) extractDeploymentMode(llmOutput map[string]interface{}) string {
	if config, ok := llmOutput["deployment_config"].(map[string]interface{}); ok {
		if mode, ok := config["mode"].(string); ok {
			return mode
		}
	}
	return "kubernetes"
}

// Utility functions
func (g *Generator) targetComponentsToStrings(components []v1.NetworkTargetComponent) []string {
	result := make([]string, len(components))
	for i, component := range components {
		result[i] = string(component)
	}
	return result
}

func (g *Generator) getOrDefault(m map[string]interface{}, key string, defaultValue interface{}) interface{} {
	if val, ok := m[key]; ok {
		return val
	}
	return defaultValue
}

func (g *Generator) shouldGenerateServiceMesh(genCtx *GenerationContext) bool {
	if serviceMesh, ok := genCtx.LLMOutput["service_mesh"].(map[string]interface{}); ok {
		if enabled, ok := serviceMesh["enabled"].(bool); ok {
			return enabled
		}
	}
	return len(genCtx.Intent.Spec.TargetComponents) > 1 // Enable for multi-component deployments
}

func (g *Generator) shouldGenerateGateway(genCtx *GenerationContext) bool {
	if serviceMesh, ok := genCtx.LLMOutput["service_mesh"].(map[string]interface{}); ok {
		if enabled, ok := serviceMesh["ingress_enabled"].(bool); ok {
			return enabled
		}
	}
	return false
}

// HealthCheck performs health check on the generator
func (g *Generator) HealthCheck(ctx context.Context) bool {
	// Check LLM connectivity
	// Note: Current LLM client doesn't have HealthCheck method, so we skip this check
	// TODO: Add health check when LLM client supports it

	// Check template availability
	if len(g.coreTemplates) == 0 && len(g.oranTemplates) == 0 {
		g.logger.Warn("No templates loaded")
		return false
	}

	return true
}

// initializeTemplates loads and compiles all blueprint templates
func (g *Generator) initializeTemplates() error {
	// This would typically load templates from files or embedded resources
	// For brevity, showing the structure without full template content

	funcMap := sprig.TxtFuncMap()

	// Load core templates (AMF, SMF, UPF, etc.)
	coreTemplateNames := []string{"amf", "smf", "upf", "nssf", "ausf", "udm"}
	for _, name := range coreTemplateNames {
		tmpl, err := template.New(name).Funcs(funcMap).Parse(g.getCoreTemplate(name))
		if err != nil {
			return fmt.Errorf("failed to parse %s template: %w", name, err)
		}
		g.coreTemplates[name] = tmpl
	}

	// Load O-RAN templates
	oranTemplateNames := []string{"near-rt-ric", "non-rt-ric", "xapp", "rapp", "o-du", "o-cu-cp", "o-cu-up"}
	for _, name := range oranTemplateNames {
		tmpl, err := template.New(name).Funcs(funcMap).Parse(g.getORANTemplate(name))
		if err != nil {
			return fmt.Errorf("failed to parse %s template: %w", name, err)
		}
		g.oranTemplates[name] = tmpl
	}

	g.logger.Info("Templates initialized successfully",
		zap.Int("core_templates", len(g.coreTemplates)),
		zap.Int("oran_templates", len(g.oranTemplates)))

	return nil
}

// Placeholder template getters (in a real implementation, these would load from files)
func (g *Generator) getCoreTemplate(name string) string {
	// Return appropriate template content based on name
	return fmt.Sprintf("# %s template placeholder\n", name)
}

func (g *Generator) getORANTemplate(name string) string {
	// Return appropriate O-RAN template content based on name
	return fmt.Sprintf("# O-RAN %s template placeholder\n", name)
}

// Additional helper methods would be implemented for:
// - buildRICTemplateData
// - buildXAppTemplateData
// - buildNetworkSliceTemplateData
// - generateComponentService
// - generateAMFConfig, generateSMFConfig, etc.
// - generateE2Configuration
// - generateXAppDescriptor
// - generateVirtualService, generateDestinationRule, etc.
// - generateServiceMonitor, generateGrafanaDashboard, etc.

// generateKptfile generates the Kptfile for the package
func (g *Generator) generateKptfile(genCtx *GenerationContext) (string, error) {
	kptfile := fmt.Sprintf(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: %s
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: Network function blueprint generated from intent
  keywords:
    - o-ran
    - 5g-core
    - network-function
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.2.0
      configMap:
        intent.nephio.io/name: %s
        intent.nephio.io/type: %s
    - image: gcr.io/kpt-fn/set-namespace:v0.4.1
      configMap:
        namespace: %s
`, genCtx.Intent.Name, genCtx.Intent.Name, genCtx.Intent.Spec.IntentType, genCtx.TargetNamespace)

	return kptfile, nil
}

// generateREADME generates the README.md file for the package
func (g *Generator) generateREADME(genCtx *GenerationContext) (string, error) {
	readme := fmt.Sprintf(`# %s Blueprint

This blueprint package was generated from a NetworkIntent for deploying network functions.

## Intent Details

- **Intent Type**: %s
- **Target Components**: %s
- **Target Cluster**: %s
- **Target Namespace**: %s
- **Network Slice**: %s

## Generated Resources

This package contains Kubernetes manifests for:
- Deployments and Services
- ConfigMaps and Secrets
- Network Policies (if service mesh is enabled)
- Monitoring configuration

## Usage

1. Review the generated resources
2. Apply using kpt or kubectl:
   ` + "```" + `bash
   kpt live apply
   # or
   kubectl apply -f .
   ` + "```" + `

## Customization

The resources can be customized by modifying the YAML files or by using kpt functions.
`, genCtx.Intent.Name, genCtx.Intent.Spec.IntentType, 
		strings.Join(g.targetComponentsToStrings(genCtx.Intent.Spec.TargetComponents), ", "),
		genCtx.TargetCluster, genCtx.TargetNamespace, genCtx.NetworkSlice)

	return readme, nil
}

// generatePackageMetadata generates package metadata
func (g *Generator) generatePackageMetadata(genCtx *GenerationContext) (string, error) {
	metadata := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-metadata
  namespace: %s
  annotations:
    config.kubernetes.io/local-config: "true"
data:
  intent-name: %s
  intent-type: %s
  generated-at: %s
  target-cluster: %s
  target-namespace: %s
  network-slice: %s
`, genCtx.Intent.Name, genCtx.TargetNamespace, 
   genCtx.Intent.Name, genCtx.Intent.Spec.IntentType, 
   time.Now().Format(time.RFC3339),
   genCtx.TargetCluster, genCtx.TargetNamespace, genCtx.NetworkSlice)

	return metadata, nil
}

// generateFunctionConfig generates kpt function configuration
func (g *Generator) generateFunctionConfig(genCtx *GenerationContext) (string, error) {
	fnConfig := fmt.Sprintf(`apiVersion: fn.kpt.dev/v1alpha1
kind: ApplyReplacements
metadata:
  name: %s-apply-replacements
  annotations:
    config.kubernetes.io/local-config: "true"
replacements:
- source:
    kind: ConfigMap
    name: %s-metadata
    fieldPath: data.target-namespace
  targets:
  - select:
      kind: Deployment
    fieldPaths:
    - metadata.namespace
  - select:
      kind: Service
    fieldPaths:
    - metadata.namespace
`, genCtx.Intent.Name, genCtx.Intent.Name)

	return fnConfig, nil
}

// generateGenericBlueprint generates a generic blueprint for unknown components
func (g *Generator) generateGenericBlueprint(genCtx *GenerationContext, component v1.NetworkTargetComponent, files map[string]string) error {
	deployConfig := g.extractDeploymentConfig(genCtx.LLMOutput)
	
	deployment := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s-%s
  namespace: %s
  labels:
    app: %s
    component: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
      component: %s
  template:
    metadata:
      labels:
        app: %s
        component: %s
    spec:
      containers:
      - name: %s
        image: %s
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
`, strings.ToLower(genCtx.Intent.Name), strings.ToLower(string(component)), 
   genCtx.TargetNamespace, strings.ToLower(genCtx.Intent.Name), strings.ToLower(string(component)),
   g.getOrDefault(deployConfig, "replicas", 1).(int),
   strings.ToLower(genCtx.Intent.Name), strings.ToLower(string(component)),
   strings.ToLower(genCtx.Intent.Name), strings.ToLower(string(component)),
   strings.ToLower(string(component)),
   g.getOrDefault(deployConfig, "image", "nginx:latest").(string))

	files[fmt.Sprintf("%s-deployment.yaml", strings.ToLower(string(component)))] = deployment

	// Generate service
	service := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s-%s
  namespace: %s
  labels:
    app: %s
    component: %s
spec:
  selector:
    app: %s
    component: %s
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP
`, strings.ToLower(genCtx.Intent.Name), strings.ToLower(string(component)),
   genCtx.TargetNamespace, strings.ToLower(genCtx.Intent.Name), strings.ToLower(string(component)),
   strings.ToLower(genCtx.Intent.Name), strings.ToLower(string(component)))

	files[fmt.Sprintf("%s-service.yaml", strings.ToLower(string(component)))] = service

	return nil
}

// Additional stub methods for missing component-specific generators
func (g *Generator) generateNSSFBlueprint(genCtx *GenerationContext, files map[string]string) error {
	return g.generateGenericBlueprint(genCtx, v1.NetworkTargetComponentNSSF, files)
}

func (g *Generator) generateODUBlueprint(genCtx *GenerationContext, files map[string]string) error {
	return g.generateGenericBlueprint(genCtx, "O-DU", files)
}

func (g *Generator) generateOCUCPBlueprint(genCtx *GenerationContext, files map[string]string) error {
	return g.generateGenericBlueprint(genCtx, "O-CU-CP", files)
}

func (g *Generator) generateOCUUPBlueprint(genCtx *GenerationContext, files map[string]string) error {
	return g.generateGenericBlueprint(genCtx, "O-CU-UP", files)
}

func (g *Generator) generateNonRTRICBlueprint(genCtx *GenerationContext, files map[string]string) error {
	return g.generateGenericBlueprint(genCtx, "Non-RT-RIC", files)
}

func (g *Generator) generateRAppBlueprint(genCtx *GenerationContext, files map[string]string) error {
	return g.generateGenericBlueprint(genCtx, "rApp", files)
}

// Additional helper method stubs that may be referenced
func (g *Generator) generateComponentService(genCtx *GenerationContext, componentName string) (string, error) {
	service := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s-%s
  namespace: %s
  labels:
    app: %s
    component: %s
spec:
  selector:
    app: %s
    component: %s
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP
`, genCtx.Intent.Name, componentName, genCtx.TargetNamespace,
   genCtx.Intent.Name, componentName, genCtx.Intent.Name, componentName)

	return service, nil
}

func (g *Generator) generateAMFConfig(genCtx *GenerationContext) (string, error) {
	config := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-amf-config
  namespace: %s
data:
  amf.conf: |
    # AMF Configuration
    networkSlice: %s
    targetCluster: %s
`, genCtx.Intent.Name, genCtx.TargetNamespace, genCtx.NetworkSlice, genCtx.TargetCluster)

	return config, nil
}

func (g *Generator) generateSMFConfig(genCtx *GenerationContext) (string, error) {
	config := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-smf-config
  namespace: %s
data:
  smf.conf: |
    # SMF Configuration
    networkSlice: %s
    targetCluster: %s
`, genCtx.Intent.Name, genCtx.TargetNamespace, genCtx.NetworkSlice, genCtx.TargetCluster)

	return config, nil
}

func (g *Generator) generateUPFNetworking(genCtx *GenerationContext) (string, error) {
	networking := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-upf-networking
  namespace: %s
data:
  networking.conf: |
    # UPF Networking Configuration
    networkSlice: %s
`, genCtx.Intent.Name, genCtx.TargetNamespace, genCtx.NetworkSlice)

	return networking, nil
}

func (g *Generator) buildAMFConfig(genCtx *GenerationContext) map[string]interface{} {
	return map[string]interface{}{
		"networkSlice": genCtx.NetworkSlice,
		"cluster":      genCtx.TargetCluster,
	}
}

func (g *Generator) buildSMFConfig(genCtx *GenerationContext) map[string]interface{} {
	return map[string]interface{}{
		"networkSlice": genCtx.NetworkSlice,
		"cluster":      genCtx.TargetCluster,
	}
}

func (g *Generator) buildUPFNetworkConfig(genCtx *GenerationContext) map[string]interface{} {
	return map[string]interface{}{
		"networkSlice": genCtx.NetworkSlice,
		"privileged":   true,
	}
}

func (g *Generator) extractResources(deployConfig map[string]interface{}) map[string]interface{} {
	if resources, ok := deployConfig["resources"].(map[string]interface{}); ok {
		return resources
	}
	return map[string]interface{}{
		"requests": map[string]string{"cpu": "100m", "memory": "128Mi"},
		"limits":   map[string]string{"cpu": "500m", "memory": "512Mi"},
	}
}

func (g *Generator) extractEnvironment(deployConfig map[string]interface{}) []map[string]string {
	if envVars, ok := deployConfig["env_vars"].([]interface{}); ok {
		result := make([]map[string]string, len(envVars))
		for i, env := range envVars {
			if envMap, ok := env.(map[string]interface{}); ok {
				result[i] = map[string]string{
					"name":  fmt.Sprintf("%v", envMap["name"]),
					"value": fmt.Sprintf("%v", envMap["value"]),
				}
			}
		}
		return result
	}
	return []map[string]string{}
}

// Observability and service mesh generation stubs
func (g *Generator) generateE2Configuration(genCtx *GenerationContext) (string, error) {
	return "# E2 Interface Configuration placeholder", nil
}

func (g *Generator) generateXAppManagementConfig(genCtx *GenerationContext) (string, error) {
	return "# xApp Management Configuration placeholder", nil
}

func (g *Generator) generateXAppDescriptor(genCtx *GenerationContext) (string, error) {
	return "{\"name\": \"xapp-descriptor\", \"version\": \"1.0\"}", nil
}

func (g *Generator) generateSliceQoSConfig(genCtx *GenerationContext) (string, error) {
	return "# Network Slice QoS Configuration placeholder", nil
}

func (g *Generator) generateVirtualService(genCtx *GenerationContext) (string, error) {
	return "# Istio VirtualService Configuration placeholder", nil
}

func (g *Generator) generateDestinationRule(genCtx *GenerationContext) (string, error) {
	return "# Istio DestinationRule Configuration placeholder", nil
}

func (g *Generator) generateGateway(genCtx *GenerationContext) (string, error) {
	return "# Istio Gateway Configuration placeholder", nil
}

func (g *Generator) generatePeerAuthentication(genCtx *GenerationContext) (string, error) {
	return "# Istio PeerAuthentication Configuration placeholder", nil
}

func (g *Generator) generateServiceMonitor(genCtx *GenerationContext) (string, error) {
	return "# Prometheus ServiceMonitor Configuration placeholder", nil
}

func (g *Generator) generateGrafanaDashboard(genCtx *GenerationContext) (string, error) {
	return "# Grafana Dashboard Configuration placeholder", nil
}

func (g *Generator) generateAlertRules(genCtx *GenerationContext) (string, error) {
	return "# Prometheus Alert Rules Configuration placeholder", nil
}

// Additional template data builders
func (g *Generator) buildRICTemplateData(genCtx *GenerationContext, ricType string) map[string]interface{} {
	return map[string]interface{}{
		"Name":      fmt.Sprintf("%s-%s", genCtx.Intent.Name, ricType),
		"Namespace": genCtx.TargetNamespace,
		"RICType":   ricType,
	}
}

func (g *Generator) buildXAppTemplateData(genCtx *GenerationContext) map[string]interface{} {
	return map[string]interface{}{
		"Name":      fmt.Sprintf("%s-xapp", genCtx.Intent.Name),
		"Namespace": genCtx.TargetNamespace,
		"AppType":   "xApp",
	}
}

func (g *Generator) buildNetworkSliceTemplateData(genCtx *GenerationContext) map[string]interface{} {
	return map[string]interface{}{
		"Name":         genCtx.Intent.Name,
		"Namespace":    genCtx.TargetNamespace,
		"NetworkSlice": genCtx.NetworkSlice,
	}
}

// These would follow similar patterns to the methods shown above
