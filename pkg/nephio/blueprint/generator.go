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
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"go.uber.org/zap"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// LLMClientInterface defines the interface that the Generator expects from the LLM client
type LLMClientInterface interface {
	ProcessIntent(ctx context.Context, intent string) (string, error)
	HealthCheck(ctx context.Context) bool
}

// Generator handles blueprint package generation from NetworkIntents.

type Generator struct {
	config *BlueprintConfig

	logger *zap.Logger

	llmClient LLMClientInterface

	httpClient *http.Client

	templateCache sync.Map

	// O-RAN specific templates.

	oranTemplates map[string]*template.Template

	coreTemplates map[string]*template.Template

	edgeTemplates map[string]*template.Template

	sliceTemplates map[string]*template.Template

	serviceMeshTemplates map[string]*template.Template

	// Performance optimization.

	resourcePool sync.Pool

	generationMutex sync.RWMutex
}

// GenerationContext contains context for blueprint generation.

type GenerationContext struct {
	Intent *v1.NetworkIntent

	LLMOutput map[string]interface{}

	TargetCluster string

	TargetNamespace string

	NetworkSlice string

	ComponentType v1.ORANComponent

	DeploymentMode string

	SecurityProfile string

	ResourceProfile string
}

// NewGenerator creates a new blueprint generator.

func NewGenerator(config *BlueprintConfig, logger *zap.Logger) (*Generator, error) {
	if config == nil {
		config = DefaultBlueprintConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	// Initialize LLM client.

	llmClient := llm.NewClient(config.LLMEndpoint)

	generator := &Generator{
		config: config,

		logger: logger,

		llmClient: llmClient,

		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},

		oranTemplates: make(map[string]*template.Template),

		coreTemplates: make(map[string]*template.Template),

		edgeTemplates: make(map[string]*template.Template),

		sliceTemplates: make(map[string]*template.Template),

		serviceMeshTemplates: make(map[string]*template.Template),

		resourcePool: sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{})
			},
		},
	}

	// Initialize templates.

	if err := generator.initializeTemplates(); err != nil {
		return nil, fmt.Errorf("failed to initialize templates: %w", err)
	}

	return generator, nil
}

// NewGeneratorWithLLMClient creates a new blueprint generator with a custom LLM client (for testing)
func NewGeneratorWithLLMClient(config *BlueprintConfig, logger *zap.Logger, llmClient LLMClientInterface) (*Generator, error) {
	if config == nil {
		config = DefaultBlueprintConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	if llmClient == nil {
		return nil, fmt.Errorf("LLM client cannot be nil")
	}

	generator := &Generator{
		config: config,
		logger: logger,
		llmClient: llmClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
		oranTemplates: make(map[string]*template.Template),
		coreTemplates: make(map[string]*template.Template),
		edgeTemplates: make(map[string]*template.Template),
		sliceTemplates: make(map[string]*template.Template),
		serviceMeshTemplates: make(map[string]*template.Template),
		resourcePool: sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{})
			},
		},
	}

	// Initialize templates.
	if err := generator.initializeTemplates(); err != nil {
		return nil, fmt.Errorf("failed to initialize templates: %w", err)
	}

	return generator, nil
}

// GenerateFromNetworkIntent generates blueprint files from NetworkIntent.
func (g *Generator) GenerateFromNetworkIntent(ctx context.Context, intent *v1.NetworkIntent) (map[string]string, error) {
	// Defensive programming: Validate inputs
	if g == nil {
		return nil, fmt.Errorf("generator is nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if intent == nil {
		return nil, fmt.Errorf("intent is nil")
	}
	if g.logger == nil {
		g.logger = zap.NewNop()
	}
	
	// Add panic recovery for critical path
	defer func() {
		if r := recover(); r != nil {
			g.logger.Error("Panic recovered in GenerateFromNetworkIntent", 
				zap.Any("panic", r),
				zap.Stack("stack"))
		}
	}()

	startTime := time.Now()

	g.logger.Info("Generating blueprint from NetworkIntent",
		zap.String("intent_name", intent.Name),
		zap.String("intent_type", string(intent.Spec.IntentType)),
		zap.Strings("target_components", networkTargetComponentsToStrings(intent.Spec.TargetComponents)))

	// Step 1: Process intent with LLM to extract parameters.
	llmOutput, err := g.processIntentWithLLM(ctx, intent)
	if err != nil {
		// Try fallback processing if LLM fails
		if fallbackOutput := g.getFallbackLLMOutput(intent); fallbackOutput != nil {
			g.logger.Warn("LLM processing failed, using fallback", zap.Error(err))
			llmOutput = fallbackOutput
		} else {
			return nil, fmt.Errorf("LLM processing failed: %w", err)
		}
	}

	// Step 2: Create generation context.

	genCtx := &GenerationContext{
		Intent: intent,

		LLMOutput: llmOutput,

		TargetCluster: intent.Spec.TargetCluster,

		TargetNamespace: intent.Spec.TargetNamespace,

		NetworkSlice: intent.Spec.NetworkSlice,

		DeploymentMode: g.extractDeploymentMode(llmOutput),

		SecurityProfile: g.extractSecurityProfile(llmOutput),

		ResourceProfile: g.extractResourceProfile(llmOutput),
	}

	if genCtx.TargetNamespace == "" {
		genCtx.TargetNamespace = g.config.DefaultNamespace
	}

	// Step 3: Generate blueprint files based on intent type and target components.

	files := make(map[string]string)

	// Generate core blueprint structure.

	if err := g.generateCoreStructure(ctx, genCtx, files); err != nil {
		return nil, fmt.Errorf("core structure generation failed: %w", err)
	}

	// Generate component-specific blueprints.

	for _, component := range intent.Spec.TargetComponents {

		oranComponent := convertNetworkTargetComponentToORANComponent(component)
		genCtx.ComponentType = oranComponent

		if err := g.generateComponentBlueprint(ctx, genCtx, oranComponent, files); err != nil {
			return nil, fmt.Errorf("component blueprint generation failed for %s: %w", component, err)
		}

	}

	// Generate network slice configuration if applicable.

	if genCtx.NetworkSlice != "" {
		if err := g.generateNetworkSliceConfiguration(ctx, genCtx, files); err != nil {
			return nil, fmt.Errorf("network slice configuration generation failed: %w", err)
		}
	}

	// Generate service mesh configuration.

	if g.shouldGenerateServiceMesh(genCtx) {
		if err := g.generateServiceMeshConfiguration(ctx, genCtx, files); err != nil {
			return nil, fmt.Errorf("service mesh configuration generation failed: %w", err)
		}
	}

	// Generate monitoring and observability configuration.

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

// processIntentWithLLM processes the NetworkIntent with LLM to extract structured parameters.
func (g *Generator) processIntentWithLLM(ctx context.Context, intent *v1.NetworkIntent) (map[string]interface{}, error) {
	// Defensive programming: Validate inputs
	if g == nil {
		return nil, fmt.Errorf("generator is nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if intent == nil {
		return nil, fmt.Errorf("intent is nil")
	}
	if g.llmClient == nil {
		return nil, fmt.Errorf("llmClient is nil")
	}

	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			if g.logger != nil {
				g.logger.Error("Panic recovered in processIntentWithLLM", 
					zap.Any("panic", r),
					zap.Stack("stack"))
			}
		}
	}()

	prompt := g.buildLLMPrompt(intent)
	if prompt == "" {
		return nil, fmt.Errorf("failed to build LLM prompt")
	}

	// Add timeout to context if not already present
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := g.llmClient.ProcessIntent(ctxWithTimeout, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}

	// Defensive programming: Validate response
	if response == "" {
		return nil, fmt.Errorf("LLM returned empty response")
	}
	if len(response) > 1024*1024 { // 1MB max
		return nil, fmt.Errorf("LLM response too large: %d bytes", len(response))
	}

	// Parse LLM response as JSON with defensive handling
	var llmOutput map[string]interface{}
	if err := json.Unmarshal([]byte(response), &llmOutput); err != nil {
		// Try to clean malformed JSON
		cleanedResponse := g.cleanMalformedJSON(response)
		if err := json.Unmarshal([]byte(cleanedResponse), &llmOutput); err != nil {
			return nil, fmt.Errorf("failed to parse LLM response as JSON: %w (original response: %s)", err, response)
		}
		if g.logger != nil {
			g.logger.Warn("Had to clean malformed JSON from LLM response")
		}
	}

	// Defensive programming: Validate llmOutput structure
	if llmOutput == nil {
		return make(map[string]interface{}), nil
	}

	return llmOutput, nil
}

// buildLLMPrompt creates a specialized prompt for blueprint generation.

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

		strings.Join(networkTargetComponentsToStrings(intent.Spec.TargetComponents), ", "),

		intent.Spec.Priority)
}

// buildLLMContext creates context for LLM processing.

func (g *Generator) buildLLMContext(intent *v1.NetworkIntent) map[string]interface{} {
	return map[string]interface{}{
		"intent_type": intent.Spec.IntentType,
		"target":      intent.Spec.TargetComponents,
		"namespace":   intent.Spec.TargetNamespace,
	}
}

// generateCoreStructure generates the core blueprint structure files.

func (g *Generator) generateCoreStructure(ctx context.Context, genCtx *GenerationContext, files map[string]string) error {
	// Generate Kptfile.

	kptfile, err := g.generateKptFile(genCtx)
	if err != nil {
		return fmt.Errorf("failed to generate Kptfile: %w", err)
	}

	files["Kptfile"] = kptfile

	// Generate README.

	readme, err := g.generateReadme(genCtx)
	if err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}

	files["README.md"] = readme

	// Generate package metadata.

	metadata, err := g.generateMetadata(genCtx)
	if err != nil {
		return fmt.Errorf("failed to generate package metadata: %w", err)
	}

	files["package-meta.yaml"] = metadata

	// Generate function configuration.

	fnConfig, err := g.generateFunctionConfig(genCtx)
	if err != nil {
		return fmt.Errorf("failed to generate function config: %w", err)
	}

	files["fn-config.yaml"] = fnConfig

	return nil
}

// generateComponentBlueprint generates blueprint for specific O-RAN/5GC component.

func (g *Generator) generateComponentBlueprint(ctx context.Context, genCtx *GenerationContext, component v1.ORANComponent, files map[string]string) error {
	switch component {

	// 5G Core Components.

	case v1.ORANComponentAMF:

		return g.generateAMFBlueprint(genCtx, files)

	case v1.ORANComponentSMF:

		return g.generateSMFBlueprint(genCtx, files)

	case v1.ORANComponentUPF:

		return g.generateUPFBlueprint(genCtx, files)

	// O-RAN Components.

	case v1.ORANComponentNearRTRIC:

		return g.generateNearRTRICBlueprint(genCtx, files)

	case v1.ORANComponentXApp:

		return g.generateXAppBlueprint(genCtx, files)

	case v1.ORANComponentGNodeB:

		return g.generateGenericBlueprint(genCtx, component, files)

	default:

		return g.generateGenericBlueprint(genCtx, component, files)

	}
}

// generateAMFBlueprint generates AMF-specific blueprint.

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

	// Generate AMF service.

	if serviceContent, err := g.generateComponentService(genCtx, "amf"); err == nil {
		files["amf-service.yaml"] = serviceContent
	}

	// Generate AMF configuration.

	if configContent, err := g.generateAMFConfig(genCtx); err == nil {
		files["amf-config.yaml"] = configContent
	}

	return nil
}

// generateSMFBlueprint generates SMF-specific blueprint.

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

	// Generate SMF service and configuration.

	if serviceContent, err := g.generateComponentService(genCtx, "smf"); err == nil {
		files["smf-service.yaml"] = serviceContent
	}

	if configContent, err := g.generateSMFConfig(genCtx); err == nil {
		files["smf-config.yaml"] = configContent
	}

	return nil
}

// generateUPFBlueprint generates UPF-specific blueprint.

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

	// TODO: UPF typically needs special networking configuration.

	// if networkContent, err := g.generateUPFNetworking(genCtx); err == nil {.

	// 	files["upf-networking.yaml"] = networkContent.

	// }.

	return nil
}

// generateNearRTRICBlueprint generates Near-RT RIC blueprint.

func (g *Generator) generateNearRTRICBlueprint(genCtx *GenerationContext, files map[string]string) error {
	template := g.oranTemplates["near-rt-ric"]

	if template == nil {
		return fmt.Errorf("Near-RT RIC template not found")
	}

	// TODO: data := g.buildRICTemplateData(genCtx, "near-rt").

	data := json.RawMessage(`{}`)

	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("Near-RT RIC template execution failed: %w", err)
	}

	files["near-rt-ric-deployment.yaml"] = content

	// TODO: Generate E2 interface configuration.

	// if e2Config, err := g.generateE2Configuration(genCtx); err == nil {.

	// 	files["e2-config.yaml"] = e2Config.

	// }.

	// TODO: Generate xApp management configuration.

	// if xappConfig, err := g.generateXAppManagementConfig(genCtx); err == nil {.

	// 	files["xapp-mgmt-config.yaml"] = xappConfig.

	// }.

	return nil
}

// generateXAppBlueprint generates xApp blueprint.

func (g *Generator) generateXAppBlueprint(genCtx *GenerationContext, files map[string]string) error {
	template := g.oranTemplates["xapp"]

	if template == nil {
		return fmt.Errorf("xApp template not found")
	}

	// TODO: data := g.buildXAppTemplateData(genCtx).

	data := json.RawMessage(`{}`)

	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("xApp template execution failed: %w", err)
	}

	files["xapp-deployment.yaml"] = content

	// TODO: Generate xApp descriptor.

	// if descriptor, err := g.generateXAppDescriptor(genCtx); err == nil {.

	// 	files["xapp-descriptor.json"] = descriptor.

	// }.

	return nil
}

// generateNetworkSliceConfiguration generates network slice configuration.

func (g *Generator) generateNetworkSliceConfiguration(ctx context.Context, genCtx *GenerationContext, files map[string]string) error {
	template := g.sliceTemplates["network-slice"]

	if template == nil {
		return fmt.Errorf("network slice template not found")
	}

	// TODO: data := g.buildNetworkSliceTemplateData(genCtx).

	data := json.RawMessage(`{}`)

	content, err := g.executeTemplate(template, data)
	if err != nil {
		return fmt.Errorf("network slice template execution failed: %w", err)
	}

	files["network-slice.yaml"] = content

	// TODO: Generate slice-specific QoS configuration.

	// if qosConfig, err := g.generateSliceQoSConfig(genCtx); err == nil {.

	// 	files["slice-qos-config.yaml"] = qosConfig.

	// }.

	return nil
}

// generateServiceMeshConfiguration generates Istio service mesh configuration.

func (g *Generator) generateServiceMeshConfiguration(ctx context.Context, genCtx *GenerationContext, files map[string]string) error {
	// TODO: Generate VirtualService.

	// if vsConfig, err := g.generateVirtualService(genCtx); err == nil {.

	// 	files["virtual-service.yaml"] = vsConfig.

	// }.

	// TODO: Generate DestinationRule.

	// if drConfig, err := g.generateDestinationRule(genCtx); err == nil {.

	// 	files["destination-rule.yaml"] = drConfig.

	// }.

	// TODO: Generate Gateway if ingress is needed.

	// if g.shouldGenerateGateway(genCtx) {.

	// 	if gwConfig, err := g.generateGateway(genCtx); err == nil {.

	// 		files["gateway.yaml"] = gwConfig.

	// 	}.

	// }.

	// TODO: Generate PeerAuthentication for mTLS.

	// if paConfig, err := g.generatePeerAuthentication(genCtx); err == nil {.

	// 	files["peer-authentication.yaml"] = paConfig.

	// }.

	return nil
}

// generateObservabilityConfiguration generates monitoring and observability configuration.

func (g *Generator) generateObservabilityConfiguration(ctx context.Context, genCtx *GenerationContext, files map[string]string) error {
	// Generate ServiceMonitor for Prometheus.

	if smConfig, err := g.generateServiceMonitor(genCtx); err == nil {
		files["service-monitor.yaml"] = smConfig
	}

	// Generate Grafana dashboard configuration.

	if dashConfig, err := g.generateGrafanaDashboard(genCtx); err == nil {
		files["grafana-dashboard.json"] = dashConfig
	}

	// Generate alert rules.

	if alertConfig, err := g.generateAlertRules(genCtx); err == nil {
		files["alert-rules.yaml"] = alertConfig
	}

	return nil
}

// Core structure generation methods.

// generateKptFile generates a Kptfile for the package.

func (g *Generator) generateKptFile(genCtx *GenerationContext) (string, error) {
	kptfileTemplate := `apiVersion: kpt.dev/v1

kind: Kptfile

metadata:

  name: {{ .PackageName }}

info:

  description: {{ .Description }}

  keywords:

    - "o-ran"

    - "5g-core"

    - "network-function"

pipeline:

  mutators:

    - image: gcr.io/kpt-fn/apply-replacements:v0.1.1

      configPath: apply-replacements.yaml

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("kptfile").Funcs(funcMap).Parse(kptfileTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Kptfile template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// generateReadme generates a README.md for the package.

func (g *Generator) generateReadme(genCtx *GenerationContext) (string, error) {
	readmeTemplate := `# {{ .PackageName }}



This package contains blueprint configurations for deploying {{ .Description }}.



## Components



{{ range .Components }}

- {{ . }}

{{ end }}



## Usage



This package is designed to be used with Nephio for automated network function deployment.



## Configuration



The package includes configurations for:

- Network function deployment

- Service mesh setup

- Monitoring and observability

- Security policies



Generated by Nephoran Intent Operator.

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("readme").Funcs(funcMap).Parse(readmeTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse README template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// generateMetadata generates package metadata.

func (g *Generator) generateMetadata(genCtx *GenerationContext) (string, error) {
	metadataTemplate := `apiVersion: config.kubernetes.io/v1alpha1

kind: PackageMetadata

metadata:

  name: {{ .PackageName }}

spec:

  packageName: {{ .PackageName }}

  version: {{ .Version }}

  description: {{ .Description }}

  keywords:

    - o-ran

    - 5g-core

    - network-function

  maintainers:

    - name: Nephoran Intent Operator

      email: nephoran@example.com

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("metadata").Funcs(funcMap).Parse(metadataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse metadata template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// generateFunctionConfig generates function configuration.

func (g *Generator) generateFunctionConfig(genCtx *GenerationContext) (string, error) {
	fnConfigTemplate := `apiVersion: fn.kpt.dev/v1alpha1

kind: ApplyReplacements

metadata:

  name: {{ .PackageName }}-replacements

spec:

  replacements:

    - source:

        kind: ConfigMap

        name: package-context

        fieldPath: data.name

      targets:

        - select:

            name: "*"

          fieldPaths:

            - metadata.name

    - source:

        kind: ConfigMap

        name: package-context

        fieldPath: data.namespace

      targets:

        - select:

            kind: "*"

          fieldPaths:

            - metadata.namespace

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("fnconfig").Funcs(funcMap).Parse(fnConfigTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse function config template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// generateGenericBlueprint generates a generic blueprint for components.

func (g *Generator) generateGenericBlueprint(genCtx *GenerationContext, component v1.ORANComponent, files map[string]string) error {
	// Generate deployment manifest.

	deploymentTemplate := `apiVersion: apps/v1

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

      - name: {{ .ComponentType }}

        image: {{ .Image }}

        ports:

        - containerPort: {{ .Port }}

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("generic-deployment").Funcs(funcMap).Parse(deploymentTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse generic deployment template: %w", err)
	}

	data := json.RawMessage(`{}`)

	content, err := g.executeTemplate(tmpl, data)
	if err != nil {
		return fmt.Errorf("failed to execute generic blueprint template: %w", err)
	}

	files[fmt.Sprintf("%s-deployment.yaml", strings.ToLower(string(component)))] = content

	return nil
}

// generateComponentService generates a Kubernetes service for a component.

func (g *Generator) generateComponentService(genCtx *GenerationContext, componentType string) (string, error) {
	serviceTemplate := `apiVersion: v1

kind: Service

metadata:

  name: {{ .Name }}-service

  namespace: {{ .Namespace }}

spec:

  selector:

    app: {{ .Name }}

  ports:

  - port: {{ .Port }}

    targetPort: {{ .TargetPort }}

  type: ClusterIP

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("component-service").Funcs(funcMap).Parse(serviceTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse component service template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// generateAMFConfig generates AMF-specific configuration.

func (g *Generator) generateAMFConfig(genCtx *GenerationContext) (string, error) {
	configTemplate := `apiVersion: v1

kind: ConfigMap

metadata:

  name: {{ .Name }}-config

  namespace: {{ .Namespace }}

data:

  config.yaml: |

    info:

      version: 1.0.0

      description: AMF Configuration

    configuration:

      amfName: {{ .AMFName }}

      ngapIpList:

        - 127.0.0.1

      sbi:

        scheme: http

        registerIPv4: 127.0.0.1

        bindingIPv4: 0.0.0.0

        port: 8000

      serviceNameList:

        - namf-comm

        - namf-evts

        - namf-mt

        - namf-loc

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("amf-config").Funcs(funcMap).Parse(configTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse AMF config template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// generateSMFConfig generates SMF-specific configuration  .

func (g *Generator) generateSMFConfig(genCtx *GenerationContext) (string, error) {
	configTemplate := `apiVersion: v1

kind: ConfigMap

metadata:

  name: {{ .Name }}-config

  namespace: {{ .Namespace }}

data:

  config.yaml: |

    info:

      version: 1.0.0

      description: SMF Configuration

    configuration:

      smfName: {{ .SMFName }}

      sbi:

        scheme: http

        registerIPv4: 127.0.0.1

        bindingIPv4: 0.0.0.0

        port: 8000

      serviceNameList:

        - nsmf-pdusession

        - nsmf-event-exposure

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("smf-config").Funcs(funcMap).Parse(configTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse SMF config template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// Helper methods for template data building.

func (g *Generator) buildAMFTemplateData(genCtx *GenerationContext) map[string]interface{} {
	deployConfig := g.extractDeploymentConfig(genCtx.LLMOutput)
	_ = deployConfig // TODO: Use deployConfig for building template data

	return map[string]interface{}{
		"name":      "amf",
		"namespace": genCtx.TargetNamespace,
		"config":    g.buildAMFConfig(genCtx),
	}
}

func (g *Generator) buildSMFTemplateData(genCtx *GenerationContext) map[string]interface{} {
	deployConfig := g.extractDeploymentConfig(genCtx.LLMOutput)
	_ = deployConfig // TODO: Use deployConfig for building template data

	return map[string]interface{}{
		"name":      "smf",
		"namespace": genCtx.TargetNamespace,
		"config":    g.buildSMFConfig(genCtx),
	}
}

func (g *Generator) buildUPFTemplateData(genCtx *GenerationContext) map[string]interface{} {
	deployConfig := g.extractDeploymentConfig(genCtx.LLMOutput)
	_ = deployConfig // TODO: Use deployConfig for building template data

	return map[string]interface{}{
		"name":      "upf",
		"namespace": genCtx.TargetNamespace,
		"config":    g.buildUPFNetworkConfig(genCtx),
	}
}

// Template execution helper.

func (g *Generator) executeTemplate(tmpl *template.Template, data interface{}) (string, error) {
	var buf strings.Builder

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Configuration extraction helpers.

func (g *Generator) extractDeploymentConfig(llmOutput map[string]interface{}) map[string]interface{} {
	// Defensive programming: Validate input
	if llmOutput == nil {
		return make(map[string]interface{})
	}
	
	if config, ok := llmOutput["deployment_config"].(map[string]interface{}); ok && config != nil {
		return config
	}

	return make(map[string]interface{})
}

func (g *Generator) extractSecurityProfile(llmOutput map[string]interface{}) string {
	// Defensive programming: Validate input
	if llmOutput == nil {
		return "default"
	}
	
	if security, ok := llmOutput["security_policies"].(map[string]interface{}); ok && security != nil {
		if profile, ok := security["profile"].(string); ok && profile != "" {
			return profile
		}
	}

	return "default"
}

func (g *Generator) extractResourceProfile(llmOutput map[string]interface{}) string {
	// Defensive programming: Validate input
	if llmOutput == nil {
		return "medium"
	}
	
	if perf, ok := llmOutput["performance_requirements"].(map[string]interface{}); ok && perf != nil {
		if profile, ok := perf["profile"].(string); ok && profile != "" {
			return profile
		}
	}

	return "medium"
}

func (g *Generator) extractDeploymentMode(llmOutput map[string]interface{}) string {
	// Defensive programming: Validate input
	if llmOutput == nil {
		return "kubernetes"
	}
	
	if config, ok := llmOutput["deployment_config"].(map[string]interface{}); ok && config != nil {
		if mode, ok := config["mode"].(string); ok && mode != "" {
			return mode
		}
	}

	return "kubernetes"
}

// Utility functions.

func (g *Generator) targetComponentsToStrings(components []v1.ORANComponent) []string {
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

// HealthCheck performs health check on the generator.

func (g *Generator) HealthCheck(ctx context.Context) bool {
	// Check LLM connectivity.

	if !g.llmClient.HealthCheck(ctx) {

		g.logger.Warn("LLM client health check failed")

		return false

	}

	// Check template availability.

	if len(g.coreTemplates) == 0 && len(g.oranTemplates) == 0 {

		g.logger.Warn("No templates loaded")

		return false

	}

	return true
}

// initializeTemplates loads and compiles all blueprint templates.

func (g *Generator) initializeTemplates() error {
	// This would typically load templates from files or embedded resources.

	// For brevity, showing the structure without full template content.

	funcMap := sprig.TxtFuncMap()

	// Load core templates (AMF, SMF, UPF, etc.).

	coreTemplateNames := []string{"amf", "smf", "upf", "nssf", "ausf", "udm"}

	for _, name := range coreTemplateNames {

		tmpl, err := template.New(name).Funcs(funcMap).Parse(g.getCoreTemplate(name))
		if err != nil {
			return fmt.Errorf("failed to parse %s template: %w", name, err)
		}

		g.coreTemplates[name] = tmpl

	}

	// Load O-RAN templates.

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

// Placeholder template getters (in a real implementation, these would load from files).

func (g *Generator) getCoreTemplate(name string) string {
	// Return appropriate template content based on name.

	return fmt.Sprintf("# %s template placeholder\n", name)
}

func (g *Generator) getORANTemplate(name string) string {
	// Return appropriate O-RAN template content based on name.

	return fmt.Sprintf("# O-RAN %s template placeholder\n", name)
}

// Additional helper methods would be implemented for:.

// - buildRICTemplateData.

// - buildXAppTemplateData.

// - buildNetworkSliceTemplateData.

// - generateComponentService.

// - generateAMFConfig, generateSMFConfig, etc.

// - generateE2Configuration.

// - generateXAppDescriptor.

// - generateVirtualService, generateDestinationRule, etc.

// - generateServiceMonitor, generateGrafanaDashboard, etc.

// These would follow similar patterns to the methods shown above.

// generateServiceMonitor generates ServiceMonitor for Prometheus.

func (g *Generator) generateServiceMonitor(genCtx *GenerationContext) (string, error) {
	serviceMonitorTemplate := `apiVersion: monitoring.coreos.com/v1

kind: ServiceMonitor

metadata:

  name: {{ .Name }}-monitor

  namespace: {{ .Namespace }}

spec:

  selector:

    matchLabels:

      app: {{ .Name }}

  endpoints:

  - port: metrics

    interval: 30s

    path: /metrics

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("service-monitor").Funcs(funcMap).Parse(serviceMonitorTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse ServiceMonitor template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// generateGrafanaDashboard generates Grafana dashboard configuration.

func (g *Generator) generateGrafanaDashboard(genCtx *GenerationContext) (string, error) {
	dashboardTemplate := `{

  "dashboard": {

    "id": null,

    "title": "{{ .Title }}",

    "tags": ["o-ran", "5g-core", "nephoran"],

    "timezone": "browser",

    "panels": [

      {

        "id": 1,

        "title": "Pod Status",

        "type": "stat",

        "targets": [

          {

            "expr": "up{job=\"{{ .Name }}-service\"}",

            "legendFormat": "{{ .Name }}"

          }

        ]

      },

      {

        "id": 2,

        "title": "CPU Usage",

        "type": "graph",

        "targets": [

          {

            "expr": "rate(container_cpu_usage_seconds_total{pod=~\"{{ .Name }}.*\"}[5m])",

            "legendFormat": "CPU Usage"

          }

        ]

      }

    ],

    "time": {

      "from": "now-1h",

      "to": "now"

    },

    "refresh": "5s"

  }

}`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("grafana-dashboard").Funcs(funcMap).Parse(dashboardTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Grafana dashboard template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// generateAlertRules generates Prometheus alert rules.

func (g *Generator) generateAlertRules(genCtx *GenerationContext) (string, error) {
	alertRulesTemplate := `apiVersion: monitoring.coreos.com/v1

kind: PrometheusRule

metadata:

  name: {{ .Name }}-alerts

  namespace: {{ .Namespace }}

spec:

  groups:

  - name: {{ .Name }}.rules

    rules:

    - alert: PodDown

      expr: up{job="{{ .Name }}-service"} == 0

      for: 1m

      labels:

        severity: critical

        service: {{ .Name }}

      annotations:

        summary: "Pod {{ .Name }} is down"

        description: "Pod {{ .Name }} in namespace {{ .Namespace }} has been down for more than 1 minute."

    - alert: HighCPUUsage

      expr: rate(container_cpu_usage_seconds_total{pod=~"{{ .Name }}.*"}[5m]) > 0.8

      for: 5m

      labels:

        severity: warning

        service: {{ .Name }}

      annotations:

        summary: "High CPU usage for {{ .Name }}"

        description: "CPU usage for {{ .Name }} is above 80% for more than 5 minutes."

`

	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("alert-rules").Funcs(funcMap).Parse(alertRulesTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse alert rules template: %w", err)
	}

	data := json.RawMessage(`{}`)

	return g.executeTemplate(tmpl, data)
}

// extractResources extracts resource requirements from deployment config.

func (g *Generator) extractResources(deployConfig map[string]interface{}) map[string]interface{} {
	if resources, ok := deployConfig["resources"].(map[string]interface{}); ok {
		return resources
	}

	// Return default resource requirements

	return map[string]interface{}{
		"limits": map[string]string{
			"cpu":    "500m",
			"memory": "512Mi",
		},
	}
}

// extractEnvironment extracts environment variables from deployment config.

func (g *Generator) extractEnvironment(deployConfig map[string]interface{}) []map[string]interface{} {
	if envVars, ok := deployConfig["env_vars"].([]interface{}); ok {

		result := make([]map[string]interface{}, len(envVars))

		for i, envVar := range envVars {
			if env, ok := envVar.(map[string]interface{}); ok {
				result[i] = env
			}
		}

		return result

	}

	// Return default environment variables

	return []map[string]interface{}{
		{"name": "DEBUG", "value": "false"},
	}
}

// buildAMFConfig builds AMF-specific configuration.

func (g *Generator) buildAMFConfig(genCtx *GenerationContext) map[string]interface{} {
	config := map[string]interface{}{
		"sbi": map[string]interface{}{
			"scheme":       "http",
			"registerIPv4": "127.0.0.1",
			"bindingIPv4":  "0.0.0.0",
			"port":         8080,
		},
		"serviceNameList": []string{
			"namf-comm",
			"namf-evts",
			"namf-mt",
			"namf-loc",
		},
	}

	if genCtx.NetworkSlice != "" {
		config["networkSlice"] = genCtx.NetworkSlice
	}

	return config
}

// buildSMFConfig builds SMF-specific configuration.

func (g *Generator) buildSMFConfig(genCtx *GenerationContext) map[string]interface{} {
	return map[string]interface{}{
		"sbi": map[string]interface{}{
			"scheme":       "http",
			"registerIPv4": "127.0.0.1",
			"bindingIPv4":  "0.0.0.0",
			"port":         8000,
		},
		"serviceNameList": []string{
			"nsmf-pdusession",
			"nsmf-event-exposure",
		},
	}
}

// buildUPFNetworkConfig builds UPF-specific network configuration.

func (g *Generator) buildUPFNetworkConfig(genCtx *GenerationContext) map[string]interface{} {
	return map[string]interface{}{
		"gtpu": map[string]interface{}{
			"ifList": []map[string]interface{}{
				{
					"name": "N3",
					"addr": "192.168.1.1",
					"type": "N3",
				},
			},
		},
		"pfcp": map[string]interface{}{
			"addr": "127.0.0.1",
			"port": 8805,
		},
	}
}

// getFallbackLLMOutput provides fallback LLM output when LLM service fails
func (g *Generator) getFallbackLLMOutput(intent *v1.NetworkIntent) map[string]interface{} {
	if intent == nil {
		return nil
	}
	
	// Return basic fallback configuration
	fallback := map[string]interface{}{
		"deployment_config": map[string]interface{}{
			"replicas": 1,
			"image": "default:latest",
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":    "100m",
					"memory": "128Mi",
				},
				"limits": map[string]interface{}{
					"cpu":    "500m",
					"memory": "512Mi",
				},
			},
		},
		"network_function": map[string]interface{}{
			"type":       "generic",
			"interfaces": []string{},
		},
		"oran_interfaces": map[string]interface{}{
			"a1_enabled": false,
			"o1_enabled": false,
			"o2_enabled": false,
			"e2_enabled": false,
		},
		"performance_requirements": map[string]interface{}{
			"throughput": "default",
			"latency":    "default",
			"profile":    "medium",
		},
		"security_policies": map[string]interface{}{
			"encryption":    false,
			"authentication": "none",
			"profile":       "default",
		},
		"service_mesh": map[string]interface{}{
			"enabled":        false,
			"mtls_enabled":   false,
			"ingress_enabled": false,
		},
	}
	
	if g.logger != nil {
		g.logger.Info("Using fallback LLM output", zap.String("intent", intent.Name))
	}
	
	return fallback
}

// cleanMalformedJSON attempts to clean malformed JSON responses from LLM
func (g *Generator) cleanMalformedJSON(response string) string {
	if response == "" {
		return "{}"
	}
	
	// Remove common JSON malformations
	cleaned := strings.TrimSpace(response)
	
	// Ensure it starts and ends with braces
	if !strings.HasPrefix(cleaned, "{") {
		if idx := strings.Index(cleaned, "{"); idx != -1 {
			cleaned = cleaned[idx:]
		} else {
			return "{}"
		}
	}
	
	if !strings.HasSuffix(cleaned, "}") {
		if idx := strings.LastIndex(cleaned, "}"); idx != -1 {
			cleaned = cleaned[:idx+1]
		} else {
			return "{}"
		}
	}
	
	// Remove trailing commas before closing braces/brackets
	cleaned = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(cleaned, "$1")
	
	return cleaned
}
