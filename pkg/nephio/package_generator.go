// Package nephio provides Nephio/Porch integration and package generation for the Nephoran Intent Operator.
package nephio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
	"sigs.k8s.io/yaml"
)

// PackageGenerator generates Nephio KRM packages from NetworkIntent resources
// It supports the following intent types:
// - "deployment": Generates Deployment, Service, and ConfigMap resources
// - "scaling": Generates scaling patches for existing deployments
// - "policy": Generates NetworkPolicy and A1 policy resources
// If no intent type is specified, it defaults to "deployment"
type PackageGenerator struct {
	templates   map[string]*template.Template
	porchClient porch.PorchClient // Optional Porch client for direct API calls
}

// NewPackageGenerator creates a new package generator
func NewPackageGenerator() (*PackageGenerator, error) {
	pg := &PackageGenerator{
		templates: make(map[string]*template.Template),
	}

	// Initialize templates
	if err := pg.initTemplates(); err != nil {
		return nil, fmt.Errorf("failed to initialize templates: %w", err)
	}

	return pg, nil
}

// GeneratePackage generates a complete Nephio package from a NetworkIntent
func (pg *PackageGenerator) GeneratePackage(intent *v1.NetworkIntent) (map[string]string, error) {
	files := make(map[string]string)

	// Generate package structure
	packageName := fmt.Sprintf("%s-package", intent.Name)
	packagePath := filepath.Join("packages", intent.Namespace, packageName)

	// Generate Kptfile
	kptfile, err := pg.generateKptfile(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Kptfile: %w", err)
	}
	files[filepath.Join(packagePath, "Kptfile")] = kptfile

	// Generate package resources based on intent type
	// Use the IntentType from the spec, defaulting to "deployment" if empty
	intentType := intent.Spec.IntentType
	if intentType == "" {
		intentType = "deployment"
	}

	switch intentType {
	case "deployment":
		resources, err := pg.generateDeploymentResources(intent)
		if err != nil {
			return nil, fmt.Errorf("failed to generate deployment resources: %w", err)
		}
		for path, content := range resources {
			files[filepath.Join(packagePath, path)] = content
		}
	case "scaling":
		resources, err := pg.generateScalingResources(intent)
		if err != nil {
			return nil, fmt.Errorf("failed to generate scaling resources: %w", err)
		}
		for path, content := range resources {
			files[filepath.Join(packagePath, path)] = content
		}
	case "policy":
		resources, err := pg.generatePolicyResources(intent)
		if err != nil {
			return nil, fmt.Errorf("failed to generate policy resources: %w", err)
		}
		for path, content := range resources {
			files[filepath.Join(packagePath, path)] = content
		}
	default:
		return nil, fmt.Errorf("unsupported intent type: %s", intentType)
	}

	// Generate README
	readme, err := pg.generateReadme(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate README: %w", err)
	}
	files[filepath.Join(packagePath, "README.md")] = readme

	// Generate function configuration
	fnConfig, err := pg.generateFunctionConfig(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate function config: %w", err)
	}
	files[filepath.Join(packagePath, "fn-config.yaml")] = fnConfig

	return files, nil
}

// initTemplates initializes all package templates
func (pg *PackageGenerator) initTemplates() error {
	// Kptfile template
	kptfileTmpl := `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: {{ .Name }}-package
  annotations:
    config.kubernetes.io/local-config: "true"
    nephoran.com/intent-id: {{ .Name }}
info:
  description: |
    {{ .Description }}
    Generated from NetworkIntent: {{ .Name }}
    Original Intent: {{ .Intent }}
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/apply-setters:v0.2.0
    configPath: setters.yaml
  - image: gcr.io/kpt-fn/set-namespace:v0.4.1
    configMap:
      namespace: {{ .Namespace }}
  validators:
  - image: gcr.io/kpt-fn/kubeval:v0.3.0
`

	tmpl, err := template.New("kptfile").Parse(kptfileTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse kptfile template: %w", err)
	}
	pg.templates["kptfile"] = tmpl

	// Deployment template
	deploymentTmpl := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    app: {{ .Name }}
    nephoran.com/component: {{ .Component }}
    nephoran.com/intent-id: {{ .IntentID }}
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
        nephoran.com/component: {{ .Component }}
    spec:
      containers:
      - name: {{ .Name }}
        image: {{ .Image }}
        ports:
        {{- range .Ports }}
        - name: {{ .Name }}
          containerPort: {{ .Port }}
          protocol: {{ .Protocol }}
        {{- end }}
        env:
        {{- range .Env }}
        - name: {{ .Name }}
          value: "{{ .Value }}"
        {{- end }}
        resources:
          requests:
            cpu: {{ .Resources.Requests.CPU }}
            memory: {{ .Resources.Requests.Memory }}
          limits:
            cpu: {{ .Resources.Limits.CPU }}
            memory: {{ .Resources.Limits.Memory }}
`

	tmpl, err = template.New("deployment").Parse(deploymentTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse deployment template: %w", err)
	}
	pg.templates["deployment"] = tmpl

	// Service template
	serviceTmpl := `
apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    app: {{ .Name }}
    nephoran.com/component: {{ .Component }}
spec:
  selector:
    app: {{ .Name }}
  type: {{ .Type }}
  ports:
  {{- range .Ports }}
  - name: {{ .Name }}
    port: {{ .Port }}
    targetPort: {{ .Port }}
    protocol: {{ .Protocol }}
  {{- end }}
`

	tmpl, err = template.New("service").Parse(serviceTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse service template: %w", err)
	}
	pg.templates["service"] = tmpl

	// ConfigMap template for O-RAN configuration
	configMapTmpl := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}-config
  namespace: {{ .Namespace }}
  labels:
    app: {{ .Name }}
    nephoran.com/component: {{ .Component }}
data:
  {{- range $key, $value := .Data }}
  {{ $key }}: |
    {{ $value | indent 4 }}
  {{- end }}
`

	tmpl, err = template.New("configmap").Parse(configMapTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse configmap template: %w", err)
	}
	pg.templates["configmap"] = tmpl

	// README template
	readmeTmpl := `
# {{ .Name }} Package

## Description
{{ .Description }}

## Generated from NetworkIntent
- **Intent ID**: {{ .Name }}
- **Original Intent**: {{ .Intent }}
- **Generated At**: {{ .GeneratedAt }}

## Package Contents
{{ .Contents }}

## Deployment Instructions
1. Review and customize the configuration in setters.yaml
2. Apply the package using:
   ` + "```bash" + `
   kpt fn render
   kpt live apply
   ` + "```" + `

## O-RAN Integration
{{ .ORANDetails }}

## Network Slice Configuration
{{ .NetworkSliceDetails }}
`

	tmpl, err = template.New("readme").Funcs(template.FuncMap{
		"indent": func(n int, s string) string {
			pad := strings.Repeat(" ", n)
			return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
		},
	}).Parse(readmeTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse readme template: %w", err)
	}
	pg.templates["readme"] = tmpl

	return nil
}

// generateKptfile generates the Kptfile for the package
func (pg *PackageGenerator) generateKptfile(intent *v1.NetworkIntent) (string, error) {
	data := map[string]interface{}{
		"Name":        intent.Name,
		"Namespace":   intent.Namespace,
		"Description": fmt.Sprintf("Network function package for %s", intent.Name),
		"Intent":      intent.Spec.Intent,
	}

	var buf bytes.Buffer
	if err := pg.templates["kptfile"].Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute kptfile template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// generateDeploymentResources generates resources for deployment intents
func (pg *PackageGenerator) generateDeploymentResources(intent *v1.NetworkIntent) (map[string]string, error) {
	resources := make(map[string]string)

	// Parse structured parameters from ProcessedParameters
	var params map[string]interface{}
	if intent.Spec.ProcessedParameters != nil {
		// Convert ProcessedParameters to map for template processing
		paramsJSON, err := json.Marshal(intent.Spec.ProcessedParameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal processed parameters: %w", err)
		}
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}
	if params == nil {
		return nil, fmt.Errorf("no parameters found in intent")
	}

	// Extract deployment details from parameters
	deploymentData := extractDeploymentData(params)

	// Generate deployment YAML
	var buf bytes.Buffer
	if err := pg.templates["deployment"].Execute(&buf, deploymentData); err != nil {
		return nil, fmt.Errorf("failed to execute deployment template: %w", err)
	}
	resources["deployment.yaml"] = strings.TrimSpace(buf.String())

	// Generate service YAML
	buf.Reset()
	serviceData := extractServiceData(params)
	if err := pg.templates["service"].Execute(&buf, serviceData); err != nil {
		return nil, fmt.Errorf("failed to execute service template: %w", err)
	}
	resources["service.yaml"] = strings.TrimSpace(buf.String())

	// Generate ConfigMap for O-RAN configuration if present
	if oranConfig := extractORANConfig(params); oranConfig != nil {
		buf.Reset()
		if err := pg.templates["configmap"].Execute(&buf, oranConfig); err != nil {
			return nil, fmt.Errorf("failed to execute configmap template: %w", err)
		}
		resources["oran-config.yaml"] = strings.TrimSpace(buf.String())
	}

	// Generate setters configuration
	setters := generateSetters(params)
	resources["setters.yaml"] = setters

	return resources, nil
}

// generateScalingResources generates resources for scaling intents
func (pg *PackageGenerator) generateScalingResources(intent *v1.NetworkIntent) (map[string]string, error) {
	resources := make(map[string]string)

	// Parse structured parameters from ProcessedParameters
	var params map[string]interface{}
	if intent.Spec.ProcessedParameters != nil {
		// Convert ProcessedParameters to map for template processing
		paramsJSON, err := json.Marshal(intent.Spec.ProcessedParameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal processed parameters: %w", err)
		}
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}
	if params == nil {
		return nil, fmt.Errorf("no parameters found in intent")
	}

	// Generate scaling patch
	patch := generateScalingPatch(params)
	resources["scaling-patch.yaml"] = patch

	// Generate setters for scaling parameters
	setters := generateScalingSetters(params)
	resources["setters.yaml"] = setters

	return resources, nil
}

// generatePolicyResources generates resources for policy intents
func (pg *PackageGenerator) generatePolicyResources(intent *v1.NetworkIntent) (map[string]string, error) {
	resources := make(map[string]string)

	// Parse structured parameters from ProcessedParameters
	var params map[string]interface{}
	if intent.Spec.ProcessedParameters != nil {
		// Convert ProcessedParameters to map for template processing
		paramsJSON, err := json.Marshal(intent.Spec.ProcessedParameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal processed parameters: %w", err)
		}
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}
	if params == nil {
		return nil, fmt.Errorf("no parameters found in intent")
	}

	// Generate NetworkPolicy or other policy resources
	policy := generatePolicyResource(params)
	resources["policy.yaml"] = policy

	// Generate A1 policy configuration if applicable
	if a1Policy := extractA1Policy(params); a1Policy != nil {
		a1Yaml, err := yaml.Marshal(a1Policy)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal A1 policy: %w", err)
		}
		resources["a1-policy.yaml"] = string(a1Yaml)
	}

	return resources, nil
}

// generateReadme generates the README for the package
func (pg *PackageGenerator) generateReadme(intent *v1.NetworkIntent) (string, error) {
	data := map[string]interface{}{
		"Name":                intent.Name,
		"Intent":              intent.Spec.Intent,
		"GeneratedAt":         time.Now().Format("2006-01-02 15:04:05 UTC"),
		"Description":         fmt.Sprintf("This package was automatically generated from the NetworkIntent '%s'", intent.Name),
		"Contents":            "- Kubernetes manifests\n- O-RAN configuration\n- Network slice parameters\n- Setters for customization",
		"ORANDetails":         pg.extractORANDetailsFromProcessed(intent.Spec.ProcessedParameters),
		"NetworkSliceDetails": pg.extractNetworkSliceDetailsFromProcessed(intent.Spec.ProcessedParameters),
	}

	var buf bytes.Buffer
	if err := pg.templates["readme"].Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute readme template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// extractORANDetailsFromProcessed extracts O-RAN details from ProcessedParameters
func (pg *PackageGenerator) extractORANDetailsFromProcessed(processedParams *v1.ProcessedParameters) string {
	if processedParams == nil {
		return "No processed parameters available"
	}
	var params map[string]interface{}
	if paramsJSON, err := json.Marshal(processedParams); err == nil {
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return "Unable to parse parameters"
		}
	}
	return extractORANDetails(params)
}

// extractNetworkSliceDetailsFromProcessed extracts network slice details from ProcessedParameters
func (pg *PackageGenerator) extractNetworkSliceDetailsFromProcessed(processedParams *v1.ProcessedParameters) string {
	if processedParams == nil {
		return "No processed parameters available"
	}
	var params map[string]interface{}
	if paramsJSON, err := json.Marshal(processedParams); err == nil {
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return "Unable to parse parameters"
		}
	}
	return extractNetworkSliceDetails(params)
}

// generateFunctionConfig generates the function configuration
func (pg *PackageGenerator) generateFunctionConfig(intent *v1.NetworkIntent) (string, error) {
	fnConfig := map[string]interface{}{
		"apiVersion": "fn.kpt.dev/v1alpha1",
		"kind":       "SetNamespace",
		"metadata": map[string]interface{}{
			"name": "set-namespace",
			"annotations": map[string]string{
				"config.kubernetes.io/local-config": "true",
			},
		},
		"spec": map[string]interface{}{
			"namespace": intent.Namespace,
		},
	}

	yamlData, err := yaml.Marshal(fnConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal function config: %w", err)
	}

	return string(yamlData), nil
}

// Helper functions to extract data from parameters

func extractDeploymentData(params map[string]interface{}) map[string]interface{} {
	// Extract deployment-specific data from parameters
	// This would parse the structured output from the LLM
	data := map[string]interface{}{
		"Name":      params["name"],
		"Namespace": params["namespace"],
		"Component": params["component"],
		"IntentID":  params["intent_id"],
		"Replicas":  params["replicas"],
		"Image":     params["image"],
		"Ports":     params["ports"],
		"Env":       params["env"],
		"Resources": params["resources"],
	}
	return data
}

func extractServiceData(params map[string]interface{}) map[string]interface{} {
	// Extract service configuration from parameters
	return map[string]interface{}{
		"Name":      params["name"],
		"Namespace": params["namespace"],
		"Component": params["component"],
		"Type":      "ClusterIP",
		"Ports":     params["ports"],
	}
}

func extractORANConfig(params map[string]interface{}) map[string]interface{} {
	// Extract O-RAN specific configuration
	if o1Config, ok := params["o1_config"]; ok {
		return map[string]interface{}{
			"Name":      params["name"],
			"Namespace": params["namespace"],
			"Component": "o-ran",
			"Data": map[string]string{
				"o1-config.yaml": fmt.Sprintf("%v", o1Config),
			},
		}
	}
	return nil
}

func generateSetters(params map[string]interface{}) string {
	// Generate setters.yaml for Kpt functions
	setters := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "setters",
			"annotations": map[string]string{
				"config.kubernetes.io/local-config": "true",
			},
		},
		"data": map[string]interface{}{
			"namespace": params["namespace"],
			"replicas":  fmt.Sprintf("%v", params["replicas"]),
			"image":     params["image"],
		},
	}

	yamlData, _ := yaml.Marshal(setters)
	return string(yamlData)
}

func generateScalingPatch(params map[string]interface{}) string {
	// Generate a structured patch for scaling operations with enhanced metadata
	patch := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name": params["target"],
			"annotations": map[string]interface{}{
				"porch.kpt.dev/managed":  "true",
				"nephoran.com/intent":    "scaling",
				"nephoran.com/timestamp": time.Now().Format(time.RFC3339),
			},
		},
		"spec": map[string]interface{}{
			"replicas": params["replicas"],
		},
	}

	// Add resource requests/limits if provided
	if resources, ok := params["resources"].(map[string]interface{}); ok {
		if patch["spec"].(map[string]interface{})["template"] == nil {
			patch["spec"].(map[string]interface{})["template"] = map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":      "main",
							"resources": resources,
						},
					},
				},
			}
		}
	}

	// Add HPA configuration if autoscaling is enabled
	if autoscaling, ok := params["autoscaling"].(map[string]interface{}); ok {
		if enabled, ok := autoscaling["enabled"].(bool); ok && enabled {
			patch["spec"].(map[string]interface{})["autoscaling"] = autoscaling
		}
	}

	yamlData, _ := yaml.Marshal(patch)
	return string(yamlData)
}

func generateScalingSetters(params map[string]interface{}) string {
	setters := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name": "scaling-setters",
			"annotations": map[string]string{
				"config.kubernetes.io/local-config": "true",
			},
		},
		"data": map[string]interface{}{
			"target":   params["target"],
			"replicas": fmt.Sprintf("%v", params["replicas"]),
		},
	}

	yamlData, _ := yaml.Marshal(setters)
	return string(yamlData)
}

func generatePolicyResource(params map[string]interface{}) string {
	// Generate network policy or other policy resources
	policy := map[string]interface{}{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "NetworkPolicy",
		"metadata": map[string]interface{}{
			"name":      params["name"],
			"namespace": params["namespace"],
		},
		"spec": params["policy_spec"],
	}

	yamlData, _ := yaml.Marshal(policy)
	return string(yamlData)
}

func extractA1Policy(params map[string]interface{}) map[string]interface{} {
	if a1Policy, ok := params["a1_policy"]; ok {
		return a1Policy.(map[string]interface{})
	}
	return nil
}

func extractORANDetails(params map[string]interface{}) string {
	details := []string{}

	if _, ok := params["o1_config"]; ok {
		details = append(details, "- O1 Interface: Configured for FCAPS management")
	}
	if _, ok := params["a1_policy"]; ok {
		details = append(details, "- A1 Interface: Policy management enabled")
	}
	if _, ok := params["e2_config"]; ok {
		details = append(details, "- E2 Interface: RAN control configured")
	}

	if len(details) == 0 {
		return "No O-RAN specific configuration in this package"
	}

	return strings.Join(details, "\n")
}

func extractNetworkSliceDetails(params map[string]interface{}) string {
	if slice, ok := params["network_slice"]; ok {
		sliceMap := slice.(map[string]interface{})
		return fmt.Sprintf(`- Slice ID: %s
- Slice Type: %s
- SLA Parameters: %v`,
			sliceMap["slice_id"],
			sliceMap["slice_type"],
			sliceMap["sla_parameters"])
	}
	return "No network slice configuration in this package"
}

// GeneratePatchAndPublishToPorch generates a scaling patch from intent and publishes it to Porch
func (pg *PackageGenerator) GeneratePatchAndPublishToPorch(ctx context.Context, intent *v1.NetworkIntent) error {
	if pg.porchClient == nil {
		return fmt.Errorf("porch client not configured")
	}

	// Parse structured parameters from ProcessedParameters
	var params map[string]interface{}
	if intent.Spec.ProcessedParameters != nil {
		// Convert ProcessedParameters to map for template processing
		paramsJSON, err := json.Marshal(intent.Spec.ProcessedParameters)
		if err != nil {
			return fmt.Errorf("failed to marshal processed parameters: %w", err)
		}
		if err := json.Unmarshal(paramsJSON, &params); err != nil {
			return fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}
	if params == nil {
		return fmt.Errorf("no parameters found in intent")
	}

	// Generate the scaling patch
	patchContent := generateScalingPatch(params)

	// Generate setters for the patch
	settersContent := generateScalingSetters(params)

	// Prepare package contents for Porch
	packageContents := map[string][]byte{
		"scaling-patch.yaml": []byte(patchContent),
		"setters.yaml":       []byte(settersContent),
		"README.md":          []byte(fmt.Sprintf("# Scaling Patch for %s\n\nGenerated from intent: %s\n", params["target"], intent.Name)),
	}

	// Create Kptfile for the package
	kptfile := map[string]interface{}{
		"apiVersion": "kpt.dev/v1",
		"kind":       "Kptfile",
		"metadata": map[string]interface{}{
			"name": fmt.Sprintf("%s-scaling-patch", intent.Name),
			"annotations": map[string]interface{}{
				"nephoran.com/intent-id":   string(intent.UID),
				"nephoran.com/intent-type": "scaling",
				"nephoran.com/generated":   time.Now().Format(time.RFC3339),
			},
		},
		"info": map[string]interface{}{
			"description": fmt.Sprintf("Scaling patch for %s", params["target"]),
		},
	}
	kptfileYAML, _ := yaml.Marshal(kptfile)
	packageContents["Kptfile"] = kptfileYAML

	// Determine repository and package name
	repository := "default"
	if repo, ok := params["repository"].(string); ok {
		repository = repo
	}
	packageName := fmt.Sprintf("%s-scaling-patch", intent.Name)
	revision := "v1"

	// Create package revision in Porch
	packageRevision := &porch.PackageRevision{
		Spec: porch.PackageRevisionSpec{
			Repository:  repository,
			PackageName: packageName,
			Revision:    revision,
			Lifecycle:   porch.PackageRevisionLifecycleDraft,
		},
	}

	// Create the package revision
	createdRevision, err := pg.porchClient.CreatePackageRevision(ctx, packageRevision)
	if err != nil {
		return fmt.Errorf("failed to create package revision in Porch: %w", err)
	}

	// Update package contents
	if err := pg.porchClient.UpdatePackageContents(ctx, packageName, revision, packageContents); err != nil {
		return fmt.Errorf("failed to update package contents in Porch: %w", err)
	}

	// Propose the package revision for review
	if err := pg.porchClient.ProposePackageRevision(ctx, packageName, revision); err != nil {
		return fmt.Errorf("failed to propose package revision: %w", err)
	}

	// If auto-approve is enabled in parameters, approve the package
	if autoApprove, ok := params["auto_approve"].(bool); ok && autoApprove {
		if err := pg.porchClient.ApprovePackageRevision(ctx, packageName, revision); err != nil {
			return fmt.Errorf("failed to approve package revision: %w", err)
		}
	}

	fmt.Printf("Successfully created and published scaling patch to Porch: %s/%s:%s\n",
		repository, packageName, createdRevision.Spec.Revision)

	return nil
}

// SetPorchClient sets the Porch client for API operations
func (pg *PackageGenerator) SetPorchClient(client porch.PorchClient) {
	pg.porchClient = client
}

// GenerateCNFPackage generates a CNF package from a CNFDeployment and configuration
func (pg *PackageGenerator) GenerateCNFPackage(cnf *v1.CNFDeployment, config map[string]interface{}) ([]byte, error) {
	// For now, return a simple stub package
	// This would be implemented to convert the CNF deployment to a proper Nephio package
	packageContent := fmt.Sprintf("# CNF Package Generated\n# CNF: %s\n# Function: %s\n# Config: %v\n",
		cnf.Name, cnf.Spec.Function, config)
	return []byte(packageContent), nil
}
