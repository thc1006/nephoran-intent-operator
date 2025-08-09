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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"text/template"

	"github.com/thc1006/nephoran-intent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Customizer handles blueprint customization and parameterization based on NetworkIntent context
type Customizer struct {
	config *BlueprintConfig
	logger *zap.Logger

	// Customization rules and policies
	customizationRules  map[string]*CustomizationRule
	policyEngine        *PolicyEngine
	environmentProfiles map[string]*EnvironmentProfile

	// Template processing
	templateCache    sync.Map
	functionRegistry map[string]CustomFunction

	// Performance optimization
	processingPool     sync.Pool
	customizationMutex sync.RWMutex
}

// CustomizationRule defines rules for blueprint customization
type CustomizationRule struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Priority    int    `json:"priority" yaml:"priority"`

	// Targeting criteria
	Components   []v1.TargetComponent `json:"components,omitempty" yaml:"components,omitempty"`
	IntentTypes  []v1.IntentType      `json:"intentTypes,omitempty" yaml:"intentTypes,omitempty"`
	Environments []string             `json:"environments,omitempty" yaml:"environments,omitempty"`

	// Conditions for rule activation
	Conditions []RuleCondition `json:"conditions" yaml:"conditions"`

	// Transformations to apply
	Transformations []Transformation `json:"transformations" yaml:"transformations"`

	// Rule metadata
	Enabled bool     `json:"enabled" yaml:"enabled"`
	Author  string   `json:"author" yaml:"author"`
	Version string   `json:"version" yaml:"version"`
	Tags    []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// RuleCondition defines conditions for rule activation
type RuleCondition struct {
	Field       string      `json:"field" yaml:"field"`
	Operator    string      `json:"operator" yaml:"operator"`
	Value       interface{} `json:"value" yaml:"value"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
}

// Transformation defines a transformation to apply to blueprint
type Transformation struct {
	Type        TransformationType `json:"type" yaml:"type"`
	Target      string             `json:"target" yaml:"target"`
	Action      string             `json:"action" yaml:"action"`
	Value       interface{}        `json:"value,omitempty" yaml:"value,omitempty"`
	Template    string             `json:"template,omitempty" yaml:"template,omitempty"`
	Conditions  []string           `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
}

// TransformationType defines types of transformations
type TransformationType string

const (
	TransformationReplace     TransformationType = "replace"
	TransformationMerge       TransformationType = "merge"
	TransformationAppend      TransformationType = "append"
	TransformationDelete      TransformationType = "delete"
	TransformationConditional TransformationType = "conditional"
	TransformationTemplate    TransformationType = "template"
	TransformationFunction    TransformationType = "function"
)

// EnvironmentProfile defines environment-specific configurations
type EnvironmentProfile struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`

	// Environment characteristics
	Type          EnvironmentType  `json:"type" yaml:"type"`
	Scale         EnvironmentScale `json:"scale" yaml:"scale"`
	SecurityLevel SecurityLevel    `json:"securityLevel" yaml:"securityLevel"`

	// Resource configurations
	ResourceLimits  ResourceConfiguration   `json:"resourceLimits" yaml:"resourceLimits"`
	NetworkPolicies []NetworkPolicyTemplate `json:"networkPolicies,omitempty" yaml:"networkPolicies,omitempty"`

	// Deployment configurations
	ReplicaCounts map[string]int       `json:"replicaCounts,omitempty" yaml:"replicaCounts,omitempty"`
	NodeSelectors map[string]string    `json:"nodeSelectors,omitempty" yaml:"nodeSelectors,omitempty"`
	Tolerations   []TolerationTemplate `json:"tolerations,omitempty" yaml:"tolerations,omitempty"`
	Affinity      *AffinityTemplate    `json:"affinity,omitempty" yaml:"affinity,omitempty"`

	// Service mesh configuration
	ServiceMesh ServiceMeshProfile `json:"serviceMesh" yaml:"serviceMesh"`

	// Monitoring configuration
	Monitoring MonitoringProfile `json:"monitoring" yaml:"monitoring"`

	// Custom parameters
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Environment classification enums
type EnvironmentType string
type EnvironmentScale string
type SecurityLevel string

const (
	EnvironmentDevelopment EnvironmentType = "development"
	EnvironmentTesting     EnvironmentType = "testing"
	EnvironmentStaging     EnvironmentType = "staging"
	EnvironmentProduction  EnvironmentType = "production"
	EnvironmentEdge        EnvironmentType = "edge"
)

const (
	ScaleSmall  EnvironmentScale = "small"
	ScaleMedium EnvironmentScale = "medium"
	ScaleLarge  EnvironmentScale = "large"
	ScaleXLarge EnvironmentScale = "xlarge"
)

const (
	SecurityLevelBasic    SecurityLevel = "basic"
	SecurityLevelStandard SecurityLevel = "standard"
	SecurityLevelEnhanced SecurityLevel = "enhanced"
	SecurityLevelCritical SecurityLevel = "critical"
)

// Configuration templates
type ResourceConfiguration struct {
	CPU        string `json:"cpu" yaml:"cpu"`
	Memory     string `json:"memory" yaml:"memory"`
	Storage    string `json:"storage" yaml:"storage"`
	MaxCPU     string `json:"maxCpu" yaml:"maxCpu"`
	MaxMemory  string `json:"maxMemory" yaml:"maxMemory"`
	MaxStorage string `json:"maxStorage" yaml:"maxStorage"`
}

type NetworkPolicyTemplate struct {
	Name        string   `json:"name" yaml:"name"`
	Ingress     []string `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	Egress      []string `json:"egress,omitempty" yaml:"egress,omitempty"`
	PodSelector string   `json:"podSelector" yaml:"podSelector"`
}

type TolerationTemplate struct {
	Key      string `json:"key" yaml:"key"`
	Operator string `json:"operator" yaml:"operator"`
	Value    string `json:"value,omitempty" yaml:"value,omitempty"`
	Effect   string `json:"effect" yaml:"effect"`
}

type AffinityTemplate struct {
	NodeAffinity    *NodeAffinityTemplate `json:"nodeAffinity,omitempty" yaml:"nodeAffinity,omitempty"`
	PodAffinity     *PodAffinityTemplate  `json:"podAffinity,omitempty" yaml:"podAffinity,omitempty"`
	PodAntiAffinity *PodAffinityTemplate  `json:"podAntiAffinity,omitempty" yaml:"podAntiAffinity,omitempty"`
}

type NodeAffinityTemplate struct {
	RequiredDuringScheduling  []NodeSelectorTerm `json:"requiredDuringScheduling,omitempty" yaml:"requiredDuringScheduling,omitempty"`
	PreferredDuringScheduling []NodeSelectorTerm `json:"preferredDuringScheduling,omitempty" yaml:"preferredDuringScheduling,omitempty"`
}

type PodAffinityTemplate struct {
	RequiredDuringScheduling  []PodAffinityTerm `json:"requiredDuringScheduling,omitempty" yaml:"requiredDuringScheduling,omitempty"`
	PreferredDuringScheduling []PodAffinityTerm `json:"preferredDuringScheduling,omitempty" yaml:"preferredDuringScheduling,omitempty"`
}

type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty" yaml:"matchExpressions,omitempty"`
}

type NodeSelectorRequirement struct {
	Key      string   `json:"key" yaml:"key"`
	Operator string   `json:"operator" yaml:"operator"`
	Values   []string `json:"values,omitempty" yaml:"values,omitempty"`
}

type PodAffinityTerm struct {
	LabelSelector map[string]string `json:"labelSelector" yaml:"labelSelector"`
	TopologyKey   string            `json:"topologyKey" yaml:"topologyKey"`
}

type ServiceMeshProfile struct {
	Enabled        bool   `json:"enabled" yaml:"enabled"`
	InjectSidecar  bool   `json:"injectSidecar" yaml:"injectSidecar"`
	MTLSMode       string `json:"mtlsMode" yaml:"mtlsMode"`
	TrafficPolicy  string `json:"trafficPolicy" yaml:"trafficPolicy"`
	CircuitBreaker bool   `json:"circuitBreaker" yaml:"circuitBreaker"`
	RetryPolicy    string `json:"retryPolicy" yaml:"retryPolicy"`
	TimeoutPolicy  string `json:"timeoutPolicy" yaml:"timeoutPolicy"`
}

type MonitoringProfile struct {
	Enabled               bool     `json:"enabled" yaml:"enabled"`
	MetricsEnabled        bool     `json:"metricsEnabled" yaml:"metricsEnabled"`
	LoggingEnabled        bool     `json:"loggingEnabled" yaml:"loggingEnabled"`
	TracingEnabled        bool     `json:"tracingEnabled" yaml:"tracingEnabled"`
	AlertingEnabled       bool     `json:"alertingEnabled" yaml:"alertingEnabled"`
	MetricsScrapeInterval string   `json:"metricsScrapeInterval" yaml:"metricsScrapeInterval"`
	LogLevel              string   `json:"logLevel" yaml:"logLevel"`
	AlertRules            []string `json:"alertRules,omitempty" yaml:"alertRules,omitempty"`
}

// CustomFunction represents a custom function for blueprint customization
type CustomFunction func(context.Context, interface{}) (interface{}, error)

// PolicyEngine handles policy-based customization
type PolicyEngine struct {
	policies  map[string]*CustomizationPolicy
	evaluator *PolicyEvaluator
	mutex     sync.RWMutex
}

type CustomizationPolicy struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Rules       []PolicyRule      `json:"rules" yaml:"rules"`
	Priority    int               `json:"priority" yaml:"priority"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Scope       PolicyScope       `json:"scope" yaml:"scope"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type PolicyRule struct {
	If   PolicyCondition `json:"if" yaml:"if"`
	Then PolicyAction    `json:"then" yaml:"then"`
	Else *PolicyAction   `json:"else,omitempty" yaml:"else,omitempty"`
}

type PolicyCondition struct {
	Field    string      `json:"field" yaml:"field"`
	Operator string      `json:"operator" yaml:"operator"`
	Value    interface{} `json:"value" yaml:"value"`
}

type PolicyAction struct {
	Action     string                 `json:"action" yaml:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type PolicyScope struct {
	Namespaces []string             `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`
	Components []v1.TargetComponent `json:"components,omitempty" yaml:"components,omitempty"`
	Labels     map[string]string    `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type PolicyEvaluator struct {
	// Implementation for policy evaluation logic
}

// NewCustomizer creates a new blueprint customizer
func NewCustomizer(config *BlueprintConfig, logger *zap.Logger) (*Customizer, error) {
	if config == nil {
		config = DefaultBlueprintConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	customizer := &Customizer{
		config:              config,
		logger:              logger,
		customizationRules:  make(map[string]*CustomizationRule),
		policyEngine:        NewPolicyEngine(),
		environmentProfiles: make(map[string]*EnvironmentProfile),
		functionRegistry:    make(map[string]CustomFunction),
		processingPool: sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{})
			},
		},
	}

	// Initialize default rules and profiles
	if err := customizer.initializeDefaults(); err != nil {
		return nil, fmt.Errorf("failed to initialize defaults: %w", err)
	}

	// Register built-in functions
	customizer.registerBuiltinFunctions()

	logger.Info("Blueprint customizer initialized",
		zap.Int("customization_rules", len(customizer.customizationRules)),
		zap.Int("environment_profiles", len(customizer.environmentProfiles)),
		zap.Int("custom_functions", len(customizer.functionRegistry)))

	return customizer, nil
}

// CustomizeBlueprint customizes blueprint files based on NetworkIntent context
func (c *Customizer) CustomizeBlueprint(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (map[string]string, error) {
	startTime := time.Now()

	c.logger.Info("Customizing blueprint",
		zap.String("intent_name", intent.Name),
		zap.String("intent_type", string(intent.Spec.IntentType)),
		zap.Int("files", len(files)))

	// Create customization context
	customCtx := &CustomizationContext{
		Intent:          intent,
		Files:           make(map[string]string),
		Parameters:      c.extractParameters(intent),
		Environment:     c.determineEnvironment(intent),
		TargetCluster:   intent.Spec.TargetCluster,
		TargetNamespace: intent.Spec.TargetNamespace,
		NetworkSlice:    intent.Spec.NetworkSlice,
		StartTime:       startTime,
	}

	// Copy original files to customization context
	for k, v := range files {
		customCtx.Files[k] = v
	}

	// Apply environment-specific customizations
	if err := c.applyEnvironmentCustomizations(ctx, customCtx); err != nil {
		return nil, fmt.Errorf("environment customization failed: %w", err)
	}

	// Apply component-specific customizations
	if err := c.applyComponentCustomizations(ctx, customCtx); err != nil {
		return nil, fmt.Errorf("component customization failed: %w", err)
	}

	// Apply resource customizations
	if err := c.applyResourceCustomizations(ctx, customCtx); err != nil {
		return nil, fmt.Errorf("resource customization failed: %w", err)
	}

	// Apply security customizations
	if err := c.applySecurityCustomizations(ctx, customCtx); err != nil {
		return nil, fmt.Errorf("security customization failed: %w", err)
	}

	// Apply network slice customizations
	if customCtx.NetworkSlice != "" {
		if err := c.applyNetworkSliceCustomizations(ctx, customCtx); err != nil {
			return nil, fmt.Errorf("network slice customization failed: %w", err)
		}
	}

	// Apply policy-based customizations
	if err := c.applyPolicyCustomizations(ctx, customCtx); err != nil {
		return nil, fmt.Errorf("policy customization failed: %w", err)
	}

	// Apply rule-based customizations
	if err := c.applyRuleBasedCustomizations(ctx, customCtx); err != nil {
		return nil, fmt.Errorf("rule-based customization failed: %w", err)
	}

	// Perform final validations
	if err := c.validateCustomizedBlueprint(ctx, customCtx); err != nil {
		return nil, fmt.Errorf("customized blueprint validation failed: %w", err)
	}

	duration := time.Since(startTime)
	c.logger.Info("Blueprint customization completed",
		zap.String("intent_name", intent.Name),
		zap.Duration("duration", duration),
		zap.Int("customized_files", len(customCtx.Files)))

	return customCtx.Files, nil
}

// CustomizationContext holds context for blueprint customization
type CustomizationContext struct {
	Intent             *v1.NetworkIntent
	Files              map[string]string
	Parameters         map[string]interface{}
	Environment        *EnvironmentProfile
	TargetCluster      string
	TargetNamespace    string
	NetworkSlice       string
	SecurityProfile    string
	ResourceProfile    string
	ServiceMeshEnabled bool
	StartTime          time.Time
	Metadata           map[string]interface{}
}

// applyEnvironmentCustomizations applies environment-specific customizations
func (c *Customizer) applyEnvironmentCustomizations(ctx context.Context, customCtx *CustomizationContext) error {
	if customCtx.Environment == nil {
		return nil
	}

	env := customCtx.Environment

	// Apply resource limits
	if err := c.applyResourceLimits(customCtx, &env.ResourceLimits); err != nil {
		return fmt.Errorf("failed to apply resource limits: %w", err)
	}

	// Apply replica counts
	if err := c.applyReplicaCounts(customCtx, env.ReplicaCounts); err != nil {
		return fmt.Errorf("failed to apply replica counts: %w", err)
	}

	// Apply node selectors and affinity
	if err := c.applySchedulingConstraints(customCtx, env); err != nil {
		return fmt.Errorf("failed to apply scheduling constraints: %w", err)
	}

	// Apply network policies
	if err := c.applyNetworkPolicies(customCtx, env.NetworkPolicies); err != nil {
		return fmt.Errorf("failed to apply network policies: %w", err)
	}

	// Apply service mesh configuration
	if env.ServiceMesh.Enabled {
		if err := c.applyServiceMeshConfiguration(customCtx, &env.ServiceMesh); err != nil {
			return fmt.Errorf("failed to apply service mesh configuration: %w", err)
		}
	}

	// Apply monitoring configuration
	if env.Monitoring.Enabled {
		if err := c.applyMonitoringConfiguration(customCtx, &env.Monitoring); err != nil {
			return fmt.Errorf("failed to apply monitoring configuration: %w", err)
		}
	}

	c.logger.Debug("Applied environment customizations",
		zap.String("environment", env.Name),
		zap.String("type", string(env.Type)),
		zap.String("scale", string(env.Scale)),
		zap.String("security_level", string(env.SecurityLevel)))

	return nil
}

// applyComponentCustomizations applies component-specific customizations
func (c *Customizer) applyComponentCustomizations(ctx context.Context, customCtx *CustomizationContext) error {
	for _, component := range customCtx.Intent.Spec.TargetComponents {
		switch component {
		case v1.TargetComponentAMF:
			if err := c.customizeAMF(customCtx); err != nil {
				return fmt.Errorf("AMF customization failed: %w", err)
			}
		case v1.TargetComponentSMF:
			if err := c.customizeSMF(customCtx); err != nil {
				return fmt.Errorf("SMF customization failed: %w", err)
			}
		case v1.TargetComponentUPF:
			if err := c.customizeUPF(customCtx); err != nil {
				return fmt.Errorf("UPF customization failed: %w", err)
			}
		case v1.TargetComponentNearRTRIC:
			if err := c.customizeNearRTRIC(customCtx); err != nil {
				return fmt.Errorf("Near-RT RIC customization failed: %w", err)
			}
		case v1.TargetComponentXApp:
			if err := c.customizeXApp(customCtx); err != nil {
				return fmt.Errorf("xApp customization failed: %w", err)
			}
		default:
			c.logger.Debug("Using generic customization for component",
				zap.String("component", string(component)))
		}
	}

	return nil
}

// Component-specific customization methods

func (c *Customizer) customizeAMF(ctx *CustomizationContext) error {
	// Apply AMF-specific configurations
	for filename, content := range ctx.Files {
		if strings.Contains(filename, "amf") {
			customized, err := c.applyAMFCustomizations(content, ctx)
			if err != nil {
				return err
			}
			ctx.Files[filename] = customized
		}
	}
	return nil
}

func (c *Customizer) applyAMFCustomizations(content string, ctx *CustomizationContext) (string, error) {
	// Parse YAML content
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil // Skip non-YAML files
	}

	// Apply AMF-specific customizations
	if deployment, ok := obj["spec"].(map[interface{}]interface{}); ok {
		if template, ok := deployment["template"].(map[interface{}]interface{}); ok {
			if spec, ok := template["spec"].(map[interface{}]interface{}); ok {
				if containers, ok := spec["containers"].([]interface{}); ok {
					for i, container := range containers {
						if containerMap, ok := container.(map[interface{}]interface{}); ok {
							// Customize AMF container configuration
							c.customizeAMFContainer(containerMap, ctx)
							containers[i] = containerMap
						}
					}
				}
			}
		}
	}

	// Convert back to YAML
	customizedYAML, err := yaml.Marshal(obj)
	if err != nil {
		return content, err
	}

	return string(customizedYAML), nil
}

func (c *Customizer) customizeAMFContainer(container map[interface{}]interface{}, ctx *CustomizationContext) {
	// Apply AMF-specific environment variables
	if env, ok := container["env"].([]interface{}); ok {
		// Add network slice configuration if available
		if ctx.NetworkSlice != "" {
			env = append(env, map[interface{}]interface{}{
				"name":  "NETWORK_SLICE_ID",
				"value": ctx.NetworkSlice,
			})
		}

		// Add environment-specific configurations
		if ctx.Environment != nil {
			env = append(env, map[interface{}]interface{}{
				"name":  "DEPLOYMENT_ENV",
				"value": string(ctx.Environment.Type),
			})
		}

		container["env"] = env
	}

	// Customize resource requirements
	if ctx.Environment != nil {
		if resources, ok := container["resources"].(map[interface{}]interface{}); ok {
			c.applyResourceCustomizations(resources, &ctx.Environment.ResourceLimits)
		}
	}
}

func (c *Customizer) customizeSMF(ctx *CustomizationContext) error {
	// Similar implementation for SMF customization
	return nil
}

func (c *Customizer) customizeUPF(ctx *CustomizationContext) error {
	// UPF needs special networking customizations
	for filename, content := range ctx.Files {
		if strings.Contains(filename, "upf") {
			customized, err := c.applyUPFCustomizations(content, ctx)
			if err != nil {
				return err
			}
			ctx.Files[filename] = customized
		}
	}
	return nil
}

func (c *Customizer) applyUPFCustomizations(content string, ctx *CustomizationContext) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	// Apply UPF-specific networking configurations
	if deployment, ok := obj["spec"].(map[interface{}]interface{}); ok {
		if template, ok := deployment["template"].(map[interface{}]interface{}); ok {
			if spec, ok := template["spec"].(map[interface{}]interface{}); ok {
				// Enable host networking for UPF if required
				if ctx.Environment != nil && ctx.Environment.Type == EnvironmentProduction {
					spec["hostNetwork"] = true
					spec["dnsPolicy"] = "ClusterFirstWithHostNet"
				}

				// Add privileged security context
				if containers, ok := spec["containers"].([]interface{}); ok {
					for i, container := range containers {
						if containerMap, ok := container.(map[interface{}]interface{}); ok {
							securityContext := map[interface{}]interface{}{
								"privileged": true,
								"capabilities": map[interface{}]interface{}{
									"add": []interface{}{"NET_ADMIN", "SYS_ADMIN"},
								},
							}
							containerMap["securityContext"] = securityContext
							containers[i] = containerMap
						}
					}
				}
			}
		}
	}

	customizedYAML, err := yaml.Marshal(obj)
	if err != nil {
		return content, err
	}

	return string(customizedYAML), nil
}

func (c *Customizer) customizeNearRTRIC(ctx *CustomizationContext) error {
	// Apply Near-RT RIC specific customizations
	return nil
}

func (c *Customizer) customizeXApp(ctx *CustomizationContext) error {
	// Apply xApp specific customizations
	return nil
}

// Helper methods for applying various customizations

func (c *Customizer) applyResourceLimits(ctx *CustomizationContext, limits *ResourceConfiguration) error {
	for filename, content := range ctx.Files {
		if c.isKubernetesManifest(content) {
			customized, err := c.applyResourceLimitsToManifest(content, limits)
			if err != nil {
				c.logger.Warn("Failed to apply resource limits to manifest",
					zap.String("filename", filename),
					zap.Error(err))
				continue
			}
			ctx.Files[filename] = customized
		}
	}
	return nil
}

func (c *Customizer) applyResourceLimitsToManifest(content string, limits *ResourceConfiguration) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	// Navigate to container resources
	if c.navigateToContainerResources(obj, func(resources map[interface{}]interface{}) {
		// Apply resource limits
		requests := make(map[interface{}]interface{})
		limitsMap := make(map[interface{}]interface{})

		if limits.CPU != "" {
			requests["cpu"] = limits.CPU
		}
		if limits.Memory != "" {
			requests["memory"] = limits.Memory
		}
		if limits.MaxCPU != "" {
			limitsMap["cpu"] = limits.MaxCPU
		}
		if limits.MaxMemory != "" {
			limitsMap["memory"] = limits.MaxMemory
		}

		if len(requests) > 0 {
			resources["requests"] = requests
		}
		if len(limitsMap) > 0 {
			resources["limits"] = limitsMap
		}
	}) {
		customizedYAML, err := yaml.Marshal(obj)
		if err != nil {
			return content, err
		}
		return string(customizedYAML), nil
	}

	return content, nil
}

func (c *Customizer) applyReplicaCounts(ctx *CustomizationContext, replicaCounts map[string]int) error {
	if len(replicaCounts) == 0 {
		return nil
	}

	for filename, content := range ctx.Files {
		if c.isDeploymentManifest(content) {
			customized, err := c.applyReplicaCountToDeployment(content, replicaCounts)
			if err != nil {
				c.logger.Warn("Failed to apply replica count to deployment",
					zap.String("filename", filename),
					zap.Error(err))
				continue
			}
			ctx.Files[filename] = customized
		}
	}

	return nil
}

func (c *Customizer) applyReplicaCountToDeployment(content string, replicaCounts map[string]int) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	// Get deployment name to find matching replica count
	if metadata, ok := obj["metadata"].(map[interface{}]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			// Check for direct name match or pattern match
			for pattern, replicas := range replicaCounts {
				if matched, _ := regexp.MatchString(pattern, name); matched || pattern == name {
					if spec, ok := obj["spec"].(map[interface{}]interface{}); ok {
						spec["replicas"] = replicas
						c.logger.Debug("Applied replica count",
							zap.String("deployment", name),
							zap.Int("replicas", replicas))
						break
					}
				}
			}
		}
	}

	customizedYAML, err := yaml.Marshal(obj)
	if err != nil {
		return content, err
	}

	return string(customizedYAML), nil
}

// Utility methods

func (c *Customizer) extractParameters(intent *v1.NetworkIntent) map[string]interface{} {
	params := make(map[string]interface{})

	// Extract from raw parameters
	if intent.Spec.Parameters.Raw != nil {
		var rawParams map[string]interface{}
		if err := json.Unmarshal(intent.Spec.Parameters.Raw, &rawParams); err == nil {
			for k, v := range rawParams {
				params[k] = v
			}
		}
	}

	// Extract from parameters map
	for k, v := range intent.Spec.ParametersMap {
		params[k] = v
	}

	// Add intent metadata
	params["intent_name"] = intent.Name
	params["intent_namespace"] = intent.Namespace
	params["intent_type"] = string(intent.Spec.IntentType)
	params["priority"] = string(intent.Spec.Priority)

	return params
}

func (c *Customizer) determineEnvironment(intent *v1.NetworkIntent) *EnvironmentProfile {
	// Default to production environment
	envName := "production"

	// Extract environment from various sources
	if intent.Spec.TargetCluster != "" {
		if strings.Contains(strings.ToLower(intent.Spec.TargetCluster), "dev") {
			envName = "development"
		} else if strings.Contains(strings.ToLower(intent.Spec.TargetCluster), "test") {
			envName = "testing"
		} else if strings.Contains(strings.ToLower(intent.Spec.TargetCluster), "staging") {
			envName = "staging"
		} else if strings.Contains(strings.ToLower(intent.Spec.TargetCluster), "edge") {
			envName = "edge"
		}
	}

	if profile, ok := c.environmentProfiles[envName]; ok {
		return profile
	}

	return c.environmentProfiles["production"] // fallback
}

func (c *Customizer) isKubernetesManifest(content string) bool {
	return strings.Contains(content, "apiVersion:") && strings.Contains(content, "kind:")
}

func (c *Customizer) isDeploymentManifest(content string) bool {
	return c.isKubernetesManifest(content) && strings.Contains(content, "kind: Deployment")
}

func (c *Customizer) navigateToContainerResources(obj map[string]interface{}, callback func(map[interface{}]interface{})) bool {
	// Navigate through Kubernetes deployment structure to find container resources
	if spec, ok := obj["spec"].(map[interface{}]interface{}); ok {
		if template, ok := spec["template"].(map[interface{}]interface{}); ok {
			if podSpec, ok := template["spec"].(map[interface{}]interface{}); ok {
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					for _, container := range containers {
						if containerMap, ok := container.(map[interface{}]interface{}); ok {
							if resources, ok := containerMap["resources"].(map[interface{}]interface{}); ok {
								callback(resources)
								return true
							} else {
								// Create resources section if it doesn't exist
								resources = make(map[interface{}]interface{})
								containerMap["resources"] = resources
								callback(resources)
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

// initializeDefaults initializes default customization rules and environment profiles
func (c *Customizer) initializeDefaults() error {
	// Initialize default environment profiles
	c.environmentProfiles["development"] = &EnvironmentProfile{
		Name:          "development",
		Description:   "Development environment profile",
		Type:          EnvironmentDevelopment,
		Scale:         ScaleSmall,
		SecurityLevel: SecurityLevelBasic,
		ResourceLimits: ResourceConfiguration{
			CPU:       "100m",
			Memory:    "256Mi",
			MaxCPU:    "500m",
			MaxMemory: "512Mi",
		},
		ReplicaCounts: map[string]int{
			".*": 1, // Single replica for all deployments
		},
		ServiceMesh: ServiceMeshProfile{
			Enabled:  false,
			MTLSMode: "PERMISSIVE",
		},
		Monitoring: MonitoringProfile{
			Enabled:               true,
			MetricsEnabled:        true,
			LoggingEnabled:        true,
			TracingEnabled:        false,
			AlertingEnabled:       false,
			MetricsScrapeInterval: "30s",
			LogLevel:              "debug",
		},
	}

	c.environmentProfiles["production"] = &EnvironmentProfile{
		Name:          "production",
		Description:   "Production environment profile",
		Type:          EnvironmentProduction,
		Scale:         ScaleLarge,
		SecurityLevel: SecurityLevelEnhanced,
		ResourceLimits: ResourceConfiguration{
			CPU:       "500m",
			Memory:    "1Gi",
			MaxCPU:    "2",
			MaxMemory: "4Gi",
		},
		ReplicaCounts: map[string]int{
			"amf": 3,
			"smf": 3,
			"upf": 2,
			".*":  2, // Default 2 replicas
		},
		ServiceMesh: ServiceMeshProfile{
			Enabled:        true,
			InjectSidecar:  true,
			MTLSMode:       "STRICT",
			CircuitBreaker: true,
		},
		Monitoring: MonitoringProfile{
			Enabled:               true,
			MetricsEnabled:        true,
			LoggingEnabled:        true,
			TracingEnabled:        true,
			AlertingEnabled:       true,
			MetricsScrapeInterval: "15s",
			LogLevel:              "info",
		},
	}

	return nil
}

func (c *Customizer) registerBuiltinFunctions() {
	c.functionRegistry["formatResource"] = c.formatResourceFunction
	c.functionRegistry["calculateReplicas"] = c.calculateReplicasFunction
	c.functionRegistry["generateLabels"] = c.generateLabelsFunction
}

// Built-in custom functions
func (c *Customizer) formatResourceFunction(ctx context.Context, input interface{}) (interface{}, error) {
	// Implementation for resource formatting function
	return input, nil
}

func (c *Customizer) calculateReplicasFunction(ctx context.Context, input interface{}) (interface{}, error) {
	// Implementation for replica calculation function
	return input, nil
}

func (c *Customizer) generateLabelsFunction(ctx context.Context, input interface{}) (interface{}, error) {
	// Implementation for label generation function
	return input, nil
}

// HealthCheck performs health check on the customizer
func (c *Customizer) HealthCheck(ctx context.Context) bool {
	// Check if we have environment profiles loaded
	if len(c.environmentProfiles) == 0 {
		c.logger.Warn("No environment profiles loaded")
		return false
	}

	// Check if we have customization rules loaded
	if len(c.customizationRules) == 0 {
		c.logger.Warn("No customization rules loaded")
		return false
	}

	return true
}

// Additional helper methods would be implemented for:
// - applySchedulingConstraints
// - applyNetworkPolicies
// - applyServiceMeshConfiguration
// - applyMonitoringConfiguration
// - applyResourceCustomizations
// - applySecurityCustomizations
// - applyNetworkSliceCustomizations
// - applyPolicyCustomizations
// - applyRuleBasedCustomizations
// - validateCustomizedBlueprint
// - NewPolicyEngine and related policy methods

// These would follow similar patterns to the methods shown above
