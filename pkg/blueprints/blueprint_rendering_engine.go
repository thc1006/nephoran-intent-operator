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

package blueprints

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/sprig/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"text/template"
	"bytes"

	"github.com/thc1006/nephoran-intent-operator/api/v1"
)

// BlueprintRenderingEngine renders O-RAN compliant KRM resources from blueprint templates
type BlueprintRenderingEngine struct {
	config      *BlueprintConfig
	logger      *zap.Logger
	templateMap sync.Map // Cache for compiled templates
	funcMap     template.FuncMap
	mutex       sync.RWMutex
}

// BlueprintRequest represents a blueprint rendering request
type BlueprintRequest struct {
	Intent     *v1.NetworkIntent
	Templates  []*BlueprintTemplate
	Metadata   *BlueprintMetadata
	Parameters map[string]interface{}
}

// RenderedBlueprint represents the result of blueprint rendering
type RenderedBlueprint struct {
	Name           string
	Version        string
	Description    string
	ORANCompliant  bool
	KRMResources   []KRMResource
	HelmCharts     []HelmChart
	ConfigMaps     []ConfigMap
	Secrets        []Secret
	Metadata       *BlueprintMetadata
	Dependencies   []BlueprintDependency
	GeneratedFiles map[string]string
}

// KRMResource represents a Kubernetes Resource Model resource
type KRMResource struct {
	APIVersion string                 `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                 `json:"kind" yaml:"kind"`
	Metadata   map[string]interface{} `json:"metadata" yaml:"metadata"`
	Spec       map[string]interface{} `json:"spec,omitempty" yaml:"spec,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`
	StringData map[string]string      `json:"stringData,omitempty" yaml:"stringData,omitempty"`
}

// HelmChart represents a Helm chart configuration
type HelmChart struct {
	Name         string                 `json:"name" yaml:"name"`
	Version      string                 `json:"version" yaml:"version"`
	Repository   string                 `json:"repository" yaml:"repository"`
	Values       map[string]interface{} `json:"values" yaml:"values"`
	Dependencies []HelmDependency       `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// ConfigMap represents a Kubernetes ConfigMap
type ConfigMap struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Data      map[string]string `json:"data" yaml:"data"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// Secret represents a Kubernetes Secret
type Secret struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Type      string            `json:"type" yaml:"type"`
	Data      map[string]string `json:"data" yaml:"data"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// BlueprintMetadata contains metadata about the blueprint
type BlueprintMetadata struct {
	Name              string            `json:"name" yaml:"name"`
	Version           string            `json:"version" yaml:"version"`
	Description       string            `json:"description" yaml:"description"`
	Labels            map[string]string `json:"labels" yaml:"labels"`
	Annotations       map[string]string `json:"annotations" yaml:"annotations"`
	ComponentType     v1.TargetComponent `json:"componentType" yaml:"componentType"`
	IntentType        v1.IntentType     `json:"intentType" yaml:"intentType"`
	ORANCompliant     bool              `json:"oranCompliant" yaml:"oranCompliant"`
	InterfaceTypes    []string          `json:"interfaceTypes" yaml:"interfaceTypes"`
	NetworkSlice      string            `json:"networkSlice,omitempty" yaml:"networkSlice,omitempty"`
	Region            string            `json:"region,omitempty" yaml:"region,omitempty"`
	CreatedAt         time.Time         `json:"createdAt" yaml:"createdAt"`
	GeneratedBy       string            `json:"generatedBy" yaml:"generatedBy"`
}

// NewBlueprintRenderingEngine creates a new blueprint rendering engine
func NewBlueprintRenderingEngine(config *BlueprintConfig, logger *zap.Logger) (*BlueprintRenderingEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	engine := &BlueprintRenderingEngine{
		config:  config,
		logger:  logger,
		funcMap: createTemplateFunctionMap(),
	}

	logger.Info("Blueprint rendering engine initialized")
	return engine, nil
}

// RenderORANBlueprint renders O-RAN compliant blueprint from request
func (bre *BlueprintRenderingEngine) RenderORANBlueprint(ctx context.Context, req *BlueprintRequest) (*RenderedBlueprint, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		bre.logger.Debug("Blueprint rendering completed",
			zap.String("intent_name", req.Intent.Name),
			zap.Duration("duration", duration))
	}()

	bre.logger.Info("Rendering O-RAN blueprint",
		zap.String("intent_name", req.Intent.Name),
		zap.Int("template_count", len(req.Templates)))

	// Prepare rendering context
	renderContext := bre.buildRenderingContext(req)

	// Initialize rendered blueprint
	rendered := &RenderedBlueprint{
		Name:           req.Metadata.Name,
		Version:        req.Metadata.Version,
		Description:    req.Metadata.Description,
		ORANCompliant:  req.Metadata.ORANCompliant,
		Metadata:       req.Metadata,
		GeneratedFiles: make(map[string]string),
	}

	// Process each template
	for _, tmpl := range req.Templates {
		if err := bre.renderTemplate(ctx, tmpl, renderContext, rendered); err != nil {
			return nil, fmt.Errorf("failed to render template %s: %w", tmpl.Name, err)
		}
	}

	// Post-process rendered blueprint
	if err := bre.postProcessBlueprint(ctx, rendered, req); err != nil {
		return nil, fmt.Errorf("failed to post-process blueprint: %w", err)
	}

	// Generate final files
	if err := bre.generateFiles(ctx, rendered); err != nil {
		return nil, fmt.Errorf("failed to generate files: %w", err)
	}

	bre.logger.Info("Blueprint rendering completed successfully",
		zap.String("blueprint_name", rendered.Name),
		zap.Int("krm_resources", len(rendered.KRMResources)),
		zap.Int("helm_charts", len(rendered.HelmCharts)),
		zap.Int("config_maps", len(rendered.ConfigMaps)),
		zap.Int("secrets", len(rendered.Secrets)),
		zap.Int("generated_files", len(rendered.GeneratedFiles)))

	return rendered, nil
}

// buildRenderingContext creates the context for template rendering
func (bre *BlueprintRenderingEngine) buildRenderingContext(req *BlueprintRequest) map[string]interface{} {
	context := map[string]interface{}{
		"Intent":    req.Intent,
		"Metadata":  req.Metadata,
		"Values":    req.Parameters,
		"Timestamp": time.Now(),
		"Config":    bre.config,
	}

	// Add intent-specific values
	if req.Intent != nil {
		context["IntentName"] = req.Intent.Name
		context["IntentType"] = req.Intent.Spec.IntentType
		context["Priority"] = req.Intent.Spec.Priority
		context["TargetComponents"] = req.Intent.Spec.TargetComponents
		context["TargetNamespace"] = getTargetNamespace(req.Intent)
		context["TargetCluster"] = req.Intent.Spec.TargetCluster
		context["NetworkSlice"] = req.Intent.Spec.NetworkSlice
		context["Region"] = req.Intent.Spec.Region

		// Add resource constraints if specified
		if req.Intent.Spec.ResourceConstraints != nil {
			context["ResourceConstraints"] = req.Intent.Spec.ResourceConstraints
		}

		// Add processed parameters
		if req.Intent.Spec.ProcessedParameters != nil {
			context["ProcessedParameters"] = req.Intent.Spec.ProcessedParameters
		}
	}

	// Add O-RAN specific context
	context["ORANInterfaces"] = map[string]interface{}{
		"A1": map[string]interface{}{
			"Version":  "v1.0.0",
			"Endpoint": "/a1-p",
			"Port":     8080,
		},
		"O1": map[string]interface{}{
			"Version":  "v1.0.0",
			"Port":     830,
			"Protocol": "NETCONF",
		},
		"O2": map[string]interface{}{
			"Version":  "v1.0.0",
			"Port":     8081,
			"Protocol": "HTTP",
		},
		"E2": map[string]interface{}{
			"Version":  "v1.0.0",
			"Port":     36422,
			"Protocol": "SCTP",
		},
	}

	// Add 5G Core specific context
	context["FiveGCore"] = map[string]interface{}{
		"PLMN": map[string]interface{}{
			"MCC": "001",
			"MNC": "01",
		},
		"NetworkSlicing": map[string]interface{}{
			"Enabled": true,
			"DefaultSlice": map[string]interface{}{
				"SST": 1,
				"SD":  "000001",
			},
		},
	}

	return context
}

// renderTemplate renders a single blueprint template
func (bre *BlueprintRenderingEngine) renderTemplate(
	ctx context.Context,
	tmpl *BlueprintTemplate,
	renderContext map[string]interface{},
	rendered *RenderedBlueprint,
) error {
	bre.logger.Debug("Rendering template",
		zap.String("template_name", tmpl.Name),
		zap.String("component_type", string(tmpl.ComponentType)))

	// Render KRM resources
	for _, krmTemplate := range tmpl.KRMResources {
		resource, err := bre.renderKRMResource(krmTemplate, renderContext)
		if err != nil {
			return fmt.Errorf("failed to render KRM resource: %w", err)
		}
		rendered.KRMResources = append(rendered.KRMResources, *resource)
	}

	// Render Helm chart if present
	if tmpl.HelmChart != nil {
		chart, err := bre.renderHelmChart(tmpl.HelmChart, renderContext)
		if err != nil {
			return fmt.Errorf("failed to render Helm chart: %w", err)
		}
		rendered.HelmCharts = append(rendered.HelmCharts, *chart)
	}

	// Render ConfigMaps
	for _, cmTemplate := range tmpl.ConfigMaps {
		cm, err := bre.renderConfigMap(cmTemplate, renderContext)
		if err != nil {
			return fmt.Errorf("failed to render ConfigMap: %w", err)
		}
		rendered.ConfigMaps = append(rendered.ConfigMaps, *cm)
	}

	// Render Secrets
	for _, secretTemplate := range tmpl.Secrets {
		secret, err := bre.renderSecret(secretTemplate, renderContext)
		if err != nil {
			return fmt.Errorf("failed to render Secret: %w", err)
		}
		rendered.Secrets = append(rendered.Secrets, *secret)
	}

	// Collect dependencies
	rendered.Dependencies = append(rendered.Dependencies, tmpl.Dependencies...)

	return nil
}

// renderKRMResource renders a KRM resource template
func (bre *BlueprintRenderingEngine) renderKRMResource(
	tmpl KRMResourceTemplate,
	context map[string]interface{},
) (*KRMResource, error) {
	// Render metadata
	metadata, err := bre.renderMap(tmpl.Metadata, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render metadata: %w", err)
	}

	// Render spec
	spec, err := bre.renderMap(tmpl.Spec, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render spec: %w", err)
	}

	resource := &KRMResource{
		APIVersion: tmpl.APIVersion,
		Kind:       tmpl.Kind,
		Metadata:   metadata,
		Spec:       spec,
	}

	return resource, nil
}

// renderHelmChart renders a Helm chart template
func (bre *BlueprintRenderingEngine) renderHelmChart(
	tmpl *HelmChartTemplate,
	context map[string]interface{},
) (*HelmChart, error) {
	// Render values
	values, err := bre.renderMap(tmpl.Values, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render Helm values: %w", err)
	}

	chart := &HelmChart{
		Name:         tmpl.Name,
		Version:      tmpl.Version,
		Repository:   tmpl.Repository,
		Values:       values,
		Dependencies: tmpl.Dependencies,
	}

	return chart, nil
}

// renderConfigMap renders a ConfigMap template
func (bre *BlueprintRenderingEngine) renderConfigMap(
	tmpl ConfigMapTemplate,
	context map[string]interface{},
) (*ConfigMap, error) {
	// Render data
	data := make(map[string]string)
	for key, valueTemplate := range tmpl.Data {
		rendered, err := bre.renderString(valueTemplate, context)
		if err != nil {
			return nil, fmt.Errorf("failed to render ConfigMap data key %s: %w", key, err)
		}
		data[key] = rendered
	}

	// Render name and namespace
	name, err := bre.renderString(tmpl.Name, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render ConfigMap name: %w", err)
	}

	namespace, err := bre.renderString(tmpl.Namespace, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render ConfigMap namespace: %w", err)
	}

	cm := &ConfigMap{
		Name:      name,
		Namespace: namespace,
		Data:      data,
		Labels:    bre.buildResourceLabels(context),
	}

	return cm, nil
}

// renderSecret renders a Secret template
func (bre *BlueprintRenderingEngine) renderSecret(
	tmpl SecretTemplate,
	context map[string]interface{},
) (*Secret, error) {
	// Render data
	data := make(map[string]string)
	for key, valueTemplate := range tmpl.Data {
		rendered, err := bre.renderString(valueTemplate, context)
		if err != nil {
			return nil, fmt.Errorf("failed to render Secret data key %s: %w", key, err)
		}
		data[key] = rendered
	}

	// Render name and namespace
	name, err := bre.renderString(tmpl.Name, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render Secret name: %w", err)
	}

	namespace, err := bre.renderString(tmpl.Namespace, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render Secret namespace: %w", err)
	}

	secret := &Secret{
		Name:      name,
		Namespace: namespace,
		Type:      tmpl.Type,
		Data:      data,
		Labels:    bre.buildResourceLabels(context),
	}

	return secret, nil
}

// renderString renders a string template
func (bre *BlueprintRenderingEngine) renderString(templateStr string, context map[string]interface{}) (string, error) {
	if templateStr == "" {
		return "", nil
	}

	tmpl, err := template.New("string").Funcs(bre.funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// renderMap renders a map template
func (bre *BlueprintRenderingEngine) renderMap(templateMap map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	if templateMap == nil {
		return nil, nil
	}

	result := make(map[string]interface{})
	
	for key, value := range templateMap {
		// Render the key
		renderedKey, err := bre.renderString(key, context)
		if err != nil {
			return nil, fmt.Errorf("failed to render map key %s: %w", key, err)
		}

		// Render the value based on its type
		renderedValue, err := bre.renderValue(value, context)
		if err != nil {
			return nil, fmt.Errorf("failed to render map value for key %s: %w", key, err)
		}

		result[renderedKey] = renderedValue
	}

	return result, nil
}

// renderValue renders a value of any type
func (bre *BlueprintRenderingEngine) renderValue(value interface{}, context map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return bre.renderString(v, context)
	case map[string]interface{}:
		return bre.renderMap(v, context)
	case []interface{}:
		return bre.renderSlice(v, context)
	default:
		return value, nil
	}
}

// renderSlice renders a slice template
func (bre *BlueprintRenderingEngine) renderSlice(templateSlice []interface{}, context map[string]interface{}) ([]interface{}, error) {
	if templateSlice == nil {
		return nil, nil
	}

	result := make([]interface{}, len(templateSlice))
	
	for i, value := range templateSlice {
		renderedValue, err := bre.renderValue(value, context)
		if err != nil {
			return nil, fmt.Errorf("failed to render slice item at index %d: %w", i, err)
		}
		result[i] = renderedValue
	}

	return result, nil
}

// postProcessBlueprint performs post-processing on the rendered blueprint
func (bre *BlueprintRenderingEngine) postProcessBlueprint(
	ctx context.Context,
	rendered *RenderedBlueprint,
	req *BlueprintRequest,
) error {
	// Add common labels and annotations to all resources
	bre.addCommonLabelsAndAnnotations(rendered, req)

	// Validate resource relationships
	if err := bre.validateResourceRelationships(rendered); err != nil {
		return fmt.Errorf("resource relationship validation failed: %w", err)
	}

	// Apply naming conventions
	bre.applyNamingConventions(rendered, req)

	// Optimize resource configurations
	bre.optimizeResourceConfigurations(rendered, req)

	return nil
}

// addCommonLabelsAndAnnotations adds common labels and annotations
func (bre *BlueprintRenderingEngine) addCommonLabelsAndAnnotations(rendered *RenderedBlueprint, req *BlueprintRequest) {
	commonLabels := map[string]string{
		"nephoran.com/blueprint":   "true",
		"nephoran.com/intent":      req.Intent.Name,
		"nephoran.com/component":   string(req.Metadata.ComponentType),
		"nephoran.com/version":     rendered.Version,
		"nephoran.com/managed-by":  "nephoran-intent-operator",
	}

	commonAnnotations := map[string]string{
		"nephoran.com/generated-at":    time.Now().Format(time.RFC3339),
		"nephoran.com/blueprint-name":  rendered.Name,
		"nephoran.com/intent-id":       req.Intent.Name,
		"nephoran.com/oran-compliant":  fmt.Sprintf("%t", rendered.ORANCompliant),
	}

	// Add to KRM resources
	for i := range rendered.KRMResources {
		if rendered.KRMResources[i].Metadata == nil {
			rendered.KRMResources[i].Metadata = make(map[string]interface{})
		}
		bre.addLabelsAndAnnotationsToResource(&rendered.KRMResources[i].Metadata, commonLabels, commonAnnotations)
	}

	// Add to ConfigMaps
	for i := range rendered.ConfigMaps {
		if rendered.ConfigMaps[i].Labels == nil {
			rendered.ConfigMaps[i].Labels = make(map[string]string)
		}
		for k, v := range commonLabels {
			rendered.ConfigMaps[i].Labels[k] = v
		}
	}

	// Add to Secrets
	for i := range rendered.Secrets {
		if rendered.Secrets[i].Labels == nil {
			rendered.Secrets[i].Labels = make(map[string]string)
		}
		for k, v := range commonLabels {
			rendered.Secrets[i].Labels[k] = v
		}
	}
}

// addLabelsAndAnnotationsToResource adds labels and annotations to a resource metadata
func (bre *BlueprintRenderingEngine) addLabelsAndAnnotationsToResource(
	metadata *map[string]interface{},
	labels map[string]string,
	annotations map[string]string,
) {
	// Add labels
	if labelsInterface, exists := (*metadata)["labels"]; exists {
		if labelsMap, ok := labelsInterface.(map[string]interface{}); ok {
			for k, v := range labels {
				labelsMap[k] = v
			}
		}
	} else {
		labelsMap := make(map[string]interface{})
		for k, v := range labels {
			labelsMap[k] = v
		}
		(*metadata)["labels"] = labelsMap
	}

	// Add annotations
	if annotationsInterface, exists := (*metadata)["annotations"]; exists {
		if annotationsMap, ok := annotationsInterface.(map[string]interface{}); ok {
			for k, v := range annotations {
				annotationsMap[k] = v
			}
		}
	} else {
		annotationsMap := make(map[string]interface{})
		for k, v := range annotations {
			annotationsMap[k] = v
		}
		(*metadata)["annotations"] = annotationsMap
	}
}

// validateResourceRelationships validates relationships between resources
func (bre *BlueprintRenderingEngine) validateResourceRelationships(rendered *RenderedBlueprint) error {
	// Basic validation - can be extended with specific relationship checks
	if len(rendered.KRMResources) == 0 && len(rendered.HelmCharts) == 0 {
		return fmt.Errorf("blueprint must contain at least one KRM resource or Helm chart")
	}

	return nil
}

// applyNamingConventions applies consistent naming conventions
func (bre *BlueprintRenderingEngine) applyNamingConventions(rendered *RenderedBlueprint, req *BlueprintRequest) {
	prefix := fmt.Sprintf("%s-%s", req.Intent.Name, strings.ToLower(string(req.Metadata.ComponentType)))
	
	// Apply to KRM resources
	for i := range rendered.KRMResources {
		if metadata, ok := rendered.KRMResources[i].Metadata["name"].(string); ok {
			if !strings.HasPrefix(metadata, prefix) {
				rendered.KRMResources[i].Metadata["name"] = fmt.Sprintf("%s-%s", prefix, metadata)
			}
		}
	}

	// Apply to ConfigMaps
	for i := range rendered.ConfigMaps {
		if !strings.HasPrefix(rendered.ConfigMaps[i].Name, prefix) {
			rendered.ConfigMaps[i].Name = fmt.Sprintf("%s-%s", prefix, rendered.ConfigMaps[i].Name)
		}
	}

	// Apply to Secrets
	for i := range rendered.Secrets {
		if !strings.HasPrefix(rendered.Secrets[i].Name, prefix) {
			rendered.Secrets[i].Name = fmt.Sprintf("%s-%s", prefix, rendered.Secrets[i].Name)
		}
	}
}

// optimizeResourceConfigurations optimizes resource configurations
func (bre *BlueprintRenderingEngine) optimizeResourceConfigurations(rendered *RenderedBlueprint, req *BlueprintRequest) {
	// Apply resource constraints from intent
	if req.Intent.Spec.ResourceConstraints != nil {
		constraints := req.Intent.Spec.ResourceConstraints
		
		for i := range rendered.KRMResources {
			resource := &rendered.KRMResources[i]
			
			// Apply to Deployment and StatefulSet resources
			if resource.Kind == "Deployment" || resource.Kind == "StatefulSet" {
				bre.applyResourceConstraints(resource, constraints)
			}
		}
	}

	// Apply priority-based configurations
	if req.Intent.Spec.Priority == v1.PriorityCritical {
		bre.applyCriticalPriorityOptimizations(rendered)
	}
}

// applyResourceConstraints applies resource constraints to a resource
func (bre *BlueprintRenderingEngine) applyResourceConstraints(resource *KRMResource, constraints *v1.ResourceConstraints) {
	// Navigate to container spec and apply constraints
	if spec, ok := resource.Spec["template"].(map[string]interface{}); ok {
		if podSpec, ok := spec["spec"].(map[string]interface{}); ok {
			if containers, ok := podSpec["containers"].([]interface{}); ok {
				for _, containerInterface := range containers {
					if container, ok := containerInterface.(map[string]interface{}); ok {
						bre.setContainerResources(container, constraints)
					}
				}
			}
		}
	}
}

// setContainerResources sets container resource constraints
func (bre *BlueprintRenderingEngine) setContainerResources(container map[string]interface{}, constraints *v1.ResourceConstraints) {
	resourcesMap := make(map[string]interface{})
	
	// Set requests
	if requests, exists := resourcesMap["requests"]; exists {
		if requestsMap, ok := requests.(map[string]interface{}); ok {
			if constraints.CPU != nil {
				requestsMap["cpu"] = constraints.CPU.String()
			}
			if constraints.Memory != nil {
				requestsMap["memory"] = constraints.Memory.String()
			}
		}
	} else {
		requestsMap := make(map[string]interface{})
		if constraints.CPU != nil {
			requestsMap["cpu"] = constraints.CPU.String()
		}
		if constraints.Memory != nil {
			requestsMap["memory"] = constraints.Memory.String()
		}
		resourcesMap["requests"] = requestsMap
	}
	
	// Set limits
	if limits, exists := resourcesMap["limits"]; exists {
		if limitsMap, ok := limits.(map[string]interface{}); ok {
			if constraints.MaxCPU != nil {
				limitsMap["cpu"] = constraints.MaxCPU.String()
			}
			if constraints.MaxMemory != nil {
				limitsMap["memory"] = constraints.MaxMemory.String()
			}
		}
	} else {
		limitsMap := make(map[string]interface{})
		if constraints.MaxCPU != nil {
			limitsMap["cpu"] = constraints.MaxCPU.String()
		}
		if constraints.MaxMemory != nil {
			limitsMap["memory"] = constraints.MaxMemory.String()
		}
		resourcesMap["limits"] = limitsMap
	}
	
	container["resources"] = resourcesMap
}

// applyCriticalPriorityOptimizations applies optimizations for critical priority
func (bre *BlueprintRenderingEngine) applyCriticalPriorityOptimizations(rendered *RenderedBlueprint) {
	for i := range rendered.KRMResources {
		resource := &rendered.KRMResources[i]
		
		// Set priority class for critical workloads
		if resource.Kind == "Deployment" || resource.Kind == "StatefulSet" {
			if spec, ok := resource.Spec["template"].(map[string]interface{}); ok {
				if podSpec, ok := spec["spec"].(map[string]interface{}); ok {
					podSpec["priorityClassName"] = "system-cluster-critical"
				}
			}
		}
	}
}

// generateFiles generates the final files for the blueprint
func (bre *BlueprintRenderingEngine) generateFiles(ctx context.Context, rendered *RenderedBlueprint) error {
	// Generate KRM resource files
	for i, resource := range rendered.KRMResources {
		filename := fmt.Sprintf("%s-%s.yaml", strings.ToLower(resource.Kind), i)
		content, err := yaml.Marshal(resource)
		if err != nil {
			return fmt.Errorf("failed to marshal KRM resource to YAML: %w", err)
		}
		rendered.GeneratedFiles[filename] = string(content)
	}

	// Generate Helm chart files
	for i, chart := range rendered.HelmCharts {
		filename := fmt.Sprintf("helm-chart-%s-%d.yaml", chart.Name, i)
		content, err := yaml.Marshal(chart)
		if err != nil {
			return fmt.Errorf("failed to marshal Helm chart to YAML: %w", err)
		}
		rendered.GeneratedFiles[filename] = string(content)
	}

	// Generate ConfigMap files
	for i, cm := range rendered.ConfigMaps {
		configMap := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      cm.Name,
				"namespace": cm.Namespace,
				"labels":    cm.Labels,
			},
			"data": cm.Data,
		}
		
		filename := fmt.Sprintf("configmap-%s-%d.yaml", cm.Name, i)
		content, err := yaml.Marshal(configMap)
		if err != nil {
			return fmt.Errorf("failed to marshal ConfigMap to YAML: %w", err)
		}
		rendered.GeneratedFiles[filename] = string(content)
	}

	// Generate Secret files
	for i, secret := range rendered.Secrets {
		secretResource := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      secret.Name,
				"namespace": secret.Namespace,
				"labels":    secret.Labels,
			},
			"type": secret.Type,
			"data": secret.Data,
		}
		
		filename := fmt.Sprintf("secret-%s-%d.yaml", secret.Name, i)
		content, err := yaml.Marshal(secretResource)
		if err != nil {
			return fmt.Errorf("failed to marshal Secret to YAML: %w", err)
		}
		rendered.GeneratedFiles[filename] = string(content)
	}

	// Generate metadata file
	metadataFile := map[string]interface{}{
		"name":          rendered.Name,
		"version":       rendered.Version,
		"description":   rendered.Description,
		"oranCompliant": rendered.ORANCompliant,
		"metadata":      rendered.Metadata,
		"dependencies":  rendered.Dependencies,
	}
	
	content, err := yaml.Marshal(metadataFile)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata to YAML: %w", err)
	}
	rendered.GeneratedFiles["blueprint-metadata.yaml"] = string(content)

	return nil
}

// buildResourceLabels builds common resource labels
func (bre *BlueprintRenderingEngine) buildResourceLabels(context map[string]interface{}) map[string]string {
	labels := make(map[string]string)
	
	if intentName, ok := context["IntentName"].(string); ok {
		labels["nephoran.com/intent"] = intentName
	}
	
	if componentType, ok := context["ComponentType"].(v1.TargetComponent); ok {
		labels["nephoran.com/component"] = string(componentType)
	}
	
	labels["nephoran.com/managed-by"] = "nephoran-intent-operator"
	labels["nephoran.com/blueprint"] = "true"
	
	return labels
}

// createTemplateFunctionMap creates the function map for templates
func createTemplateFunctionMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()
	
	// Add custom functions
	funcMap["toYAML"] = func(v interface{}) string {
		data, _ := yaml.Marshal(v)
		return strings.TrimSuffix(string(data), "\n")
	}
	
	funcMap["fromYAML"] = func(str string) interface{} {
		var result interface{}
		yaml.Unmarshal([]byte(str), &result)
		return result
	}
	
	funcMap["include"] = func(name string, data interface{}) string {
		// Placeholder for template include functionality
		return ""
	}
	
	funcMap["required"] = func(warn string, val interface{}) interface{} {
		if val == nil {
			panic(warn)
		}
		return val
	}
	
	funcMap["tpl"] = func(tpl string, data interface{}) string {
		// Placeholder for template processing
		return tpl
	}

	return funcMap
}

// getTargetNamespace gets the target namespace with fallback
func getTargetNamespace(intent *v1.NetworkIntent) string {
	if intent.Spec.TargetNamespace != "" {
		return intent.Spec.TargetNamespace
	}
	if intent.Namespace != "" {
		return intent.Namespace
	}
	return "default"
}