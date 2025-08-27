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
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
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
	Components   []v1.NetworkTargetComponent `json:"components,omitempty" yaml:"components,omitempty"`
	IntentTypes  []v1.IntentType             `json:"intentTypes,omitempty" yaml:"intentTypes,omitempty"`
	Environments []string                    `json:"environments,omitempty" yaml:"environments,omitempty"`

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
	Namespaces []string                    `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`
	Components []v1.NetworkTargetComponent `json:"components,omitempty" yaml:"components,omitempty"`
	Labels     map[string]string           `json:"labels,omitempty" yaml:"labels,omitempty"`
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
		case v1.NetworkTargetComponentAMF:
			if err := c.customizeAMF(customCtx); err != nil {
				return fmt.Errorf("AMF customization failed: %w", err)
			}
		case v1.NetworkTargetComponentSMF:
			if err := c.customizeSMF(customCtx); err != nil {
				return fmt.Errorf("SMF customization failed: %w", err)
			}
		case v1.NetworkTargetComponentUPF:
			if err := c.customizeUPF(customCtx); err != nil {
				return fmt.Errorf("UPF customization failed: %w", err)
			}
		case "Near-RT-RIC": // Using string constant as defined in networkintent_types.go
			if err := c.customizeNearRTRIC(customCtx); err != nil {
				return fmt.Errorf("Near-RT RIC customization failed: %w", err)
			}
		case "xApp": // Using string constant as defined in networkintent_types.go
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
			c.applyResourceLimitsToContainer(resources, &ctx.Environment.ResourceLimits)
		}
	}
}

func (c *Customizer) applyResourceLimitsToContainer(resources map[interface{}]interface{}, limits *ResourceConfiguration) {
	// Apply resource limits to container resources
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

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		policies:  make(map[string]*CustomizationPolicy),
		evaluator: &PolicyEvaluator{},
	}
}

// applyResourceCustomizations applies resource-specific customizations
func (c *Customizer) applyResourceCustomizations(ctx context.Context, customCtx *CustomizationContext) error {
	c.logger.Debug("Applying resource customizations")
	
	// Apply resource customizations to all Kubernetes manifests
	for filename, content := range customCtx.Files {
		if c.isKubernetesManifest(content) {
			customized, err := c.applyResourceCustomizationsToManifest(content, customCtx)
			if err != nil {
				c.logger.Warn("Failed to apply resource customizations",
					zap.String("filename", filename),
					zap.Error(err))
				continue
			}
			customCtx.Files[filename] = customized
		}
	}
	
	return nil
}

func (c *Customizer) applyResourceCustomizationsToManifest(content string, customCtx *CustomizationContext) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil // Skip non-YAML files
	}

	// Apply resource customizations based on component type
	if kind, ok := obj["kind"].(string); ok {
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			c.applyWorkloadResourceCustomizations(obj, customCtx)
		case "Service":
			c.applyServiceResourceCustomizations(obj, customCtx)
		case "ConfigMap":
			c.applyConfigMapResourceCustomizations(obj, customCtx)
		case "Secret":
			c.applySecretResourceCustomizations(obj, customCtx)
		}
	}

	customizedYAML, err := yaml.Marshal(obj)
	if err != nil {
		return content, err
	}

	return string(customizedYAML), nil
}

func (c *Customizer) applyWorkloadResourceCustomizations(obj map[string]interface{}, customCtx *CustomizationContext) {
	// Add custom labels
	c.addCustomLabels(obj, customCtx)
	
	// Add custom annotations
	c.addCustomAnnotations(obj, customCtx)
}

func (c *Customizer) applyServiceResourceCustomizations(obj map[string]interface{}, customCtx *CustomizationContext) {
	// Apply service-specific customizations
	c.addCustomLabels(obj, customCtx)
	c.addCustomAnnotations(obj, customCtx)
}

func (c *Customizer) applyConfigMapResourceCustomizations(obj map[string]interface{}, customCtx *CustomizationContext) {
	// Apply ConfigMap-specific customizations
	c.addCustomLabels(obj, customCtx)
	c.addCustomAnnotations(obj, customCtx)
}

func (c *Customizer) applySecretResourceCustomizations(obj map[string]interface{}, customCtx *CustomizationContext) {
	// Apply Secret-specific customizations
	c.addCustomLabels(obj, customCtx)
	c.addCustomAnnotations(obj, customCtx)
}

func (c *Customizer) addCustomLabels(obj map[string]interface{}, customCtx *CustomizationContext) {
	if metadata, ok := obj["metadata"].(map[interface{}]interface{}); ok {
		var labels map[interface{}]interface{}
		if existing, exists := metadata["labels"].(map[interface{}]interface{}); exists {
			labels = existing
		} else {
			labels = make(map[interface{}]interface{})
			metadata["labels"] = labels
		}
		
		// Add standard labels
		labels["nephoran.io/intent-name"] = customCtx.Intent.Name
		labels["nephoran.io/intent-type"] = string(customCtx.Intent.Spec.IntentType)
		labels["nephoran.io/version"] = "v1"
		
		if customCtx.NetworkSlice != "" {
			labels["nephoran.io/network-slice"] = customCtx.NetworkSlice
		}
		
		if customCtx.Environment != nil {
			labels["nephoran.io/environment"] = string(customCtx.Environment.Type)
		}
	}
}

func (c *Customizer) addCustomAnnotations(obj map[string]interface{}, customCtx *CustomizationContext) {
	if metadata, ok := obj["metadata"].(map[interface{}]interface{}); ok {
		var annotations map[interface{}]interface{}
		if existing, exists := metadata["annotations"].(map[interface{}]interface{}); exists {
			annotations = existing
		} else {
			annotations = make(map[interface{}]interface{})
			metadata["annotations"] = annotations
		}
		
		// Add standard annotations
		annotations["nephoran.io/customized-at"] = customCtx.StartTime.Format(time.RFC3339)
		annotations["nephoran.io/target-cluster"] = customCtx.TargetCluster
		annotations["nephoran.io/target-namespace"] = customCtx.TargetNamespace
	}
}

// applySecurityCustomizations applies security-specific customizations
func (c *Customizer) applySecurityCustomizations(ctx context.Context, customCtx *CustomizationContext) error {
	c.logger.Debug("Applying security customizations")
	
	if customCtx.Environment == nil {
		return nil
	}
	
	securityLevel := customCtx.Environment.SecurityLevel
	
	for filename, content := range customCtx.Files {
		if c.isKubernetesManifest(content) {
			customized, err := c.applySecurityCustomizationsToManifest(content, securityLevel)
			if err != nil {
				c.logger.Warn("Failed to apply security customizations",
					zap.String("filename", filename),
					zap.Error(err))
				continue
			}
			customCtx.Files[filename] = customized
		}
	}
	
	return nil
}

func (c *Customizer) applySecurityCustomizationsToManifest(content string, securityLevel SecurityLevel) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	if kind, ok := obj["kind"].(string); ok {
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			c.applyWorkloadSecurityCustomizations(obj, securityLevel)
		}
	}

	customizedYAML, err := yaml.Marshal(obj)
	if err != nil {
		return content, err
	}

	return string(customizedYAML), nil
}

func (c *Customizer) applyWorkloadSecurityCustomizations(obj map[string]interface{}, securityLevel SecurityLevel) {
	// Navigate to pod template spec
	if spec, ok := obj["spec"].(map[interface{}]interface{}); ok {
		if template, ok := spec["template"].(map[interface{}]interface{}); ok {
			if podSpec, ok := template["spec"].(map[interface{}]interface{}); ok {
				// Apply security context based on security level
				switch securityLevel {
				case SecurityLevelBasic:
					c.applyBasicSecurityContext(podSpec)
				case SecurityLevelStandard:
					c.applyStandardSecurityContext(podSpec)
				case SecurityLevelEnhanced:
					c.applyEnhancedSecurityContext(podSpec)
				case SecurityLevelCritical:
					c.applyCriticalSecurityContext(podSpec)
				}
			}
		}
	}
}

func (c *Customizer) applyBasicSecurityContext(podSpec map[interface{}]interface{}) {
	// Basic security: run as non-root
	securityContext := map[interface{}]interface{}{
		"runAsNonRoot": true,
		"runAsUser":    1000,
	}
	podSpec["securityContext"] = securityContext
}

func (c *Customizer) applyStandardSecurityContext(podSpec map[interface{}]interface{}) {
	// Standard security: non-root + read-only filesystem
	securityContext := map[interface{}]interface{}{
		"runAsNonRoot":             true,
		"runAsUser":                1000,
		"readOnlyRootFilesystem":   true,
		"allowPrivilegeEscalation": false,
	}
	podSpec["securityContext"] = securityContext
}

func (c *Customizer) applyEnhancedSecurityContext(podSpec map[interface{}]interface{}) {
	// Enhanced security: standard + dropped capabilities
	securityContext := map[interface{}]interface{}{
		"runAsNonRoot":             true,
		"runAsUser":                1000,
		"readOnlyRootFilesystem":   true,
		"allowPrivilegeEscalation": false,
		"seccompProfile": map[interface{}]interface{}{
			"type": "RuntimeDefault",
		},
		"capabilities": map[interface{}]interface{}{
			"drop": []interface{}{"ALL"},
		},
	}
	podSpec["securityContext"] = securityContext
}

func (c *Customizer) applyCriticalSecurityContext(podSpec map[interface{}]interface{}) {
	// Critical security: enhanced + apparmor + selinux
	securityContext := map[interface{}]interface{}{
		"runAsNonRoot":             true,
		"runAsUser":                1000,
		"readOnlyRootFilesystem":   true,
		"allowPrivilegeEscalation": false,
		"seccompProfile": map[interface{}]interface{}{
			"type": "RuntimeDefault",
		},
		"capabilities": map[interface{}]interface{}{
			"drop": []interface{}{"ALL"},
		},
		"seLinuxOptions": map[interface{}]interface{}{
			"level": "s0:c123,c456",
		},
	}
	podSpec["securityContext"] = securityContext
	
	// Add AppArmor annotation
	if metadata, ok := podSpec["metadata"].(map[interface{}]interface{}); !ok {
		metadata = make(map[interface{}]interface{})
		podSpec["metadata"] = metadata
	}
	if annotations, ok := podSpec["metadata"].(map[interface{}]interface{})["annotations"].(map[interface{}]interface{}); !ok {
		annotations = make(map[interface{}]interface{})
		podSpec["metadata"].(map[interface{}]interface{})["annotations"] = annotations
	}
}

// applyNetworkSliceCustomizations applies network slice specific customizations
func (c *Customizer) applyNetworkSliceCustomizations(ctx context.Context, customCtx *CustomizationContext) error {
	c.logger.Debug("Applying network slice customizations",
		zap.String("network_slice", customCtx.NetworkSlice))
	
	if customCtx.NetworkSlice == "" {
		return nil
	}
	
	for filename, content := range customCtx.Files {
		if c.isKubernetesManifest(content) {
			customized, err := c.applyNetworkSliceCustomizationsToManifest(content, customCtx)
			if err != nil {
				c.logger.Warn("Failed to apply network slice customizations",
					zap.String("filename", filename),
					zap.Error(err))
				continue
			}
			customCtx.Files[filename] = customized
		}
	}
	
	return nil
}

func (c *Customizer) applyNetworkSliceCustomizationsToManifest(content string, customCtx *CustomizationContext) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	// Add network slice labels and annotations
	c.addNetworkSliceLabels(obj, customCtx)
	
	// Configure network slice specific environment variables
	if kind, ok := obj["kind"].(string); ok {
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			c.configureNetworkSliceEnvironment(obj, customCtx)
		}
	}

	customizedYAML, err := yaml.Marshal(obj)
	if err != nil {
		return content, err
	}

	return string(customizedYAML), nil
}

func (c *Customizer) addNetworkSliceLabels(obj map[string]interface{}, customCtx *CustomizationContext) {
	if metadata, ok := obj["metadata"].(map[interface{}]interface{}); ok {
		var labels map[interface{}]interface{}
		if existing, exists := metadata["labels"].(map[interface{}]interface{}); exists {
			labels = existing
		} else {
			labels = make(map[interface{}]interface{})
			metadata["labels"] = labels
		}
		
		labels["nephoran.io/network-slice"] = customCtx.NetworkSlice
		labels["nephoran.io/slice-priority"] = string(customCtx.Intent.Spec.Priority)
	}
}

func (c *Customizer) configureNetworkSliceEnvironment(obj map[string]interface{}, customCtx *CustomizationContext) {
	// Navigate to containers and add network slice environment variables
	c.navigateToContainerResources(obj, func(resources map[interface{}]interface{}) {
		// This callback is for resources, we need a different navigation for env vars
	})
	
	// Navigate to container env vars
	if spec, ok := obj["spec"].(map[interface{}]interface{}); ok {
		if template, ok := spec["template"].(map[interface{}]interface{}); ok {
			if podSpec, ok := template["spec"].(map[interface{}]interface{}); ok {
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					for i, container := range containers {
						if containerMap, ok := container.(map[interface{}]interface{}); ok {
							var env []interface{}
							if existing, exists := containerMap["env"].([]interface{}); exists {
								env = existing
							} else {
								env = make([]interface{}, 0)
							}
							
							// Add network slice environment variables
							env = append(env, map[interface{}]interface{}{
								"name":  "NEPHORAN_NETWORK_SLICE",
								"value": customCtx.NetworkSlice,
							})
							
							env = append(env, map[interface{}]interface{}{
								"name":  "NEPHORAN_SLICE_PRIORITY",
								"value": string(customCtx.Intent.Spec.Priority),
							})
							
							containerMap["env"] = env
							containers[i] = containerMap
						}
					}
				}
			}
		}
	}
}

// applyPolicyCustomizations applies policy-based customizations
func (c *Customizer) applyPolicyCustomizations(ctx context.Context, customCtx *CustomizationContext) error {
	c.logger.Debug("Applying policy customizations")
	
	if c.policyEngine == nil {
		return fmt.Errorf("policy engine not initialized")
	}
	
	// Evaluate policies against the customization context
	applicablePolicies := c.policyEngine.EvaluatePolicies(customCtx)
	
	for _, policy := range applicablePolicies {
		if err := c.applyPolicy(ctx, customCtx, policy); err != nil {
			c.logger.Warn("Failed to apply policy",
				zap.String("policy_id", policy.ID),
				zap.Error(err))
		}
	}
	
	return nil
}

func (c *Customizer) applyPolicy(ctx context.Context, customCtx *CustomizationContext, policy *CustomizationPolicy) error {
	for _, rule := range policy.Rules {
		if c.evaluatePolicyCondition(rule.If, customCtx) {
			if err := c.executePolicyAction(rule.Then, customCtx); err != nil {
				return err
			}
		} else if rule.Else != nil {
			if err := c.executePolicyAction(*rule.Else, customCtx); err != nil {
				return err
			}
		}
	}
	
	return nil
}

func (c *Customizer) evaluatePolicyCondition(condition PolicyCondition, customCtx *CustomizationContext) bool {
	// Simple condition evaluation - can be extended
	switch condition.Field {
	case "intent.type":
		return string(customCtx.Intent.Spec.IntentType) == condition.Value
	case "intent.priority":
		return string(customCtx.Intent.Spec.Priority) == condition.Value
	case "environment.type":
		if customCtx.Environment != nil {
			return string(customCtx.Environment.Type) == condition.Value
		}
	case "network.slice":
		return customCtx.NetworkSlice == condition.Value
	}
	
	return false
}

func (c *Customizer) executePolicyAction(action PolicyAction, customCtx *CustomizationContext) error {
	switch action.Action {
	case "set_replicas":
		if replicas, ok := action.Parameters["replicas"].(int); ok {
			return c.setReplicasForAllDeployments(customCtx, replicas)
		}
	case "set_resource_limits":
		if limits, ok := action.Parameters["limits"].(map[string]interface{}); ok {
			return c.setResourceLimitsForAllDeployments(customCtx, limits)
		}
	case "add_labels":
		if labels, ok := action.Parameters["labels"].(map[string]string); ok {
			return c.addLabelsToAllResources(customCtx, labels)
		}
	}
	
	return nil
}

func (c *Customizer) setReplicasForAllDeployments(customCtx *CustomizationContext, replicas int) error {
	for filename, content := range customCtx.Files {
		if c.isDeploymentManifest(content) {
			customized, err := c.setReplicasInManifest(content, replicas)
			if err != nil {
				continue
			}
			customCtx.Files[filename] = customized
		}
	}
	return nil
}

func (c *Customizer) setReplicasInManifest(content string, replicas int) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	if spec, ok := obj["spec"].(map[interface{}]interface{}); ok {
		spec["replicas"] = replicas
	}

	customizedYAML, err := yaml.Marshal(obj)
	if err != nil {
		return content, err
	}

	return string(customizedYAML), nil
}

func (c *Customizer) setResourceLimitsForAllDeployments(customCtx *CustomizationContext, limits map[string]interface{}) error {
	// Implementation for setting resource limits
	return nil
}

func (c *Customizer) addLabelsToAllResources(customCtx *CustomizationContext, labels map[string]string) error {
	// Implementation for adding labels
	return nil
}

// applyRuleBasedCustomizations applies rule-based customizations
func (c *Customizer) applyRuleBasedCustomizations(ctx context.Context, customCtx *CustomizationContext) error {
	c.logger.Debug("Applying rule-based customizations")
	
	// Get applicable rules based on context
	applicableRules := c.getApplicableRules(customCtx)
	
	// Sort rules by priority
	for i := 0; i < len(applicableRules)-1; i++ {
		for j := 0; j < len(applicableRules)-i-1; j++ {
			if applicableRules[j].Priority < applicableRules[j+1].Priority {
				applicableRules[j], applicableRules[j+1] = applicableRules[j+1], applicableRules[j]
			}
		}
	}
	
	// Apply rules in priority order
	for _, rule := range applicableRules {
		if err := c.applyCustomizationRule(ctx, customCtx, rule); err != nil {
			c.logger.Warn("Failed to apply customization rule",
				zap.String("rule_id", rule.ID),
				zap.Error(err))
		}
	}
	
	return nil
}

func (c *Customizer) getApplicableRules(customCtx *CustomizationContext) []*CustomizationRule {
	var applicableRules []*CustomizationRule
	
	for _, rule := range c.customizationRules {
		if !rule.Enabled {
			continue
		}
		
		if c.ruleApplies(rule, customCtx) {
			applicableRules = append(applicableRules, rule)
		}
	}
	
	return applicableRules
}

func (c *Customizer) ruleApplies(rule *CustomizationRule, customCtx *CustomizationContext) bool {
	// Check if rule applies to intent type
	if len(rule.IntentTypes) > 0 {
		found := false
		for _, intentType := range rule.IntentTypes {
			if intentType == customCtx.Intent.Spec.IntentType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check if rule applies to components
	if len(rule.Components) > 0 {
		found := false
		for _, ruleComponent := range rule.Components {
			for _, intentComponent := range customCtx.Intent.Spec.TargetComponents {
				if ruleComponent == intentComponent {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check conditions
	for _, condition := range rule.Conditions {
		if !c.evaluateRuleCondition(condition, customCtx) {
			return false
		}
	}
	
	return true
}

func (c *Customizer) evaluateRuleCondition(condition RuleCondition, customCtx *CustomizationContext) bool {
	// Simple condition evaluation
	switch condition.Field {
	case "intent.name":
		return c.matchesOperator(customCtx.Intent.Name, condition.Operator, condition.Value)
	case "intent.namespace":
		return c.matchesOperator(customCtx.Intent.Namespace, condition.Operator, condition.Value)
	case "target.cluster":
		return c.matchesOperator(customCtx.TargetCluster, condition.Operator, condition.Value)
	}
	
	return true
}

func (c *Customizer) matchesOperator(fieldValue string, operator string, conditionValue interface{}) bool {
	switch operator {
	case "equals":
		return fieldValue == fmt.Sprintf("%v", conditionValue)
	case "contains":
		return strings.Contains(fieldValue, fmt.Sprintf("%v", conditionValue))
	case "matches":
		if matched, err := regexp.MatchString(fmt.Sprintf("%v", conditionValue), fieldValue); err == nil {
			return matched
		}
	}
	
	return false
}

func (c *Customizer) applyCustomizationRule(ctx context.Context, customCtx *CustomizationContext, rule *CustomizationRule) error {
	for _, transformation := range rule.Transformations {
		if err := c.applyTransformation(ctx, customCtx, transformation); err != nil {
			return err
		}
	}
	
	return nil
}

func (c *Customizer) applyTransformation(ctx context.Context, customCtx *CustomizationContext, transformation Transformation) error {
	switch transformation.Type {
	case TransformationReplace:
		return c.applyReplaceTransformation(customCtx, transformation)
	case TransformationMerge:
		return c.applyMergeTransformation(customCtx, transformation)
	case TransformationAppend:
		return c.applyAppendTransformation(customCtx, transformation)
	case TransformationDelete:
		return c.applyDeleteTransformation(customCtx, transformation)
	}
	
	return nil
}

func (c *Customizer) applyReplaceTransformation(customCtx *CustomizationContext, transformation Transformation) error {
	// Implementation for replace transformation
	return nil
}

func (c *Customizer) applyMergeTransformation(customCtx *CustomizationContext, transformation Transformation) error {
	// Implementation for merge transformation
	return nil
}

func (c *Customizer) applyAppendTransformation(customCtx *CustomizationContext, transformation Transformation) error {
	// Implementation for append transformation
	return nil
}

func (c *Customizer) applyDeleteTransformation(customCtx *CustomizationContext, transformation Transformation) error {
	// Implementation for delete transformation
	return nil
}

// validateCustomizedBlueprint validates the customized blueprint
func (c *Customizer) validateCustomizedBlueprint(ctx context.Context, customCtx *CustomizationContext) error {
	c.logger.Debug("Validating customized blueprint")
	
	var validationErrors []string
	
	// Validate each file
	for filename, content := range customCtx.Files {
		if c.isKubernetesManifest(content) {
			if err := c.validateKubernetesManifest(filename, content); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", filename, err))
			}
		}
	}
	
	// Check for required resources
	if err := c.validateRequiredResources(customCtx); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("required resources: %v", err))
	}
	
	// Validate resource relationships
	if err := c.validateResourceRelationships(customCtx); err != nil {
		validationErrors = append(validationErrors, fmt.Sprintf("resource relationships: %v", err))
	}
	
	if len(validationErrors) > 0 {
		return fmt.Errorf("blueprint validation failed: %v", validationErrors)
	}
	
	c.logger.Debug("Blueprint validation completed successfully")
	return nil
}

func (c *Customizer) validateKubernetesManifest(filename, content string) error {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	
	// Check required fields
	if _, ok := obj["apiVersion"]; !ok {
		return fmt.Errorf("missing apiVersion")
	}
	
	if _, ok := obj["kind"]; !ok {
		return fmt.Errorf("missing kind")
	}
	
	if metadata, ok := obj["metadata"].(map[interface{}]interface{}); ok {
		if _, ok := metadata["name"]; !ok {
			return fmt.Errorf("missing metadata.name")
		}
	} else {
		return fmt.Errorf("missing metadata")
	}
	
	return nil
}

func (c *Customizer) validateRequiredResources(customCtx *CustomizationContext) error {
	// Check if required components have corresponding resources
	requiredKinds := make(map[string]bool)
	
	for _, component := range customCtx.Intent.Spec.TargetComponents {
		switch component {
		case v1.NetworkTargetComponentAMF, v1.NetworkTargetComponentSMF, v1.NetworkTargetComponentUPF,
			 "Near-RT-RIC", "xApp":
			requiredKinds["Deployment"] = true
		}
	}
	
	foundKinds := make(map[string]bool)
	for _, content := range customCtx.Files {
		if c.isKubernetesManifest(content) {
			var obj map[string]interface{}
			if err := yaml.Unmarshal([]byte(content), &obj); err == nil {
				if kind, ok := obj["kind"].(string); ok {
					foundKinds[kind] = true
				}
			}
		}
	}
	
	for kind := range requiredKinds {
		if !foundKinds[kind] {
			return fmt.Errorf("missing required resource kind: %s", kind)
		}
	}
	
	return nil
}

func (c *Customizer) validateResourceRelationships(customCtx *CustomizationContext) error {
	// Validate that services have matching deployments, etc.
	deployments := make(map[string]bool)
	services := make(map[string]bool)
	
	for _, content := range customCtx.Files {
		if c.isKubernetesManifest(content) {
			var obj map[string]interface{}
			if err := yaml.Unmarshal([]byte(content), &obj); err == nil {
				if kind, ok := obj["kind"].(string); ok {
					if metadata, ok := obj["metadata"].(map[interface{}]interface{}); ok {
						if name, ok := metadata["name"].(string); ok {
							switch kind {
							case "Deployment":
								deployments[name] = true
							case "Service":
								services[name] = true
							}
						}
					}
				}
			}
		}
	}
	
	// Check that each service has a corresponding deployment (basic check)
	for serviceName := range services {
		if !deployments[serviceName] {
			c.logger.Warn("Service without matching deployment",
				zap.String("service", serviceName))
		}
	}
	
	return nil
}

// applySchedulingConstraints applies scheduling constraints
func (c *Customizer) applySchedulingConstraints(customCtx *CustomizationContext, env *EnvironmentProfile) error {
	c.logger.Debug("Applying scheduling constraints")
	
	for filename, content := range customCtx.Files {
		if c.isKubernetesManifest(content) {
			customized, err := c.applySchedulingConstraintsToManifest(content, env)
			if err != nil {
				c.logger.Warn("Failed to apply scheduling constraints",
					zap.String("filename", filename),
					zap.Error(err))
				continue
			}
			customCtx.Files[filename] = customized
		}
	}
	
	return nil
}

func (c *Customizer) applySchedulingConstraintsToManifest(content string, env *EnvironmentProfile) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	if kind, ok := obj["kind"].(string); ok {
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			c.applySchedulingToPodTemplate(obj, env)
		}
	}

	customizedYAML, err := yaml.Marshal(obj)
	if err != nil {
		return content, err
	}

	return string(customizedYAML), nil
}

func (c *Customizer) applySchedulingToPodTemplate(obj map[string]interface{}, env *EnvironmentProfile) {
	if spec, ok := obj["spec"].(map[interface{}]interface{}); ok {
		if template, ok := spec["template"].(map[interface{}]interface{}); ok {
			if podSpec, ok := template["spec"].(map[interface{}]interface{}); ok {
				// Apply node selectors
				if len(env.NodeSelectors) > 0 {
					podSpec["nodeSelector"] = env.NodeSelectors
				}
				
				// Apply tolerations
				if len(env.Tolerations) > 0 {
					tolerations := make([]interface{}, len(env.Tolerations))
					for i, t := range env.Tolerations {
						tolerations[i] = map[interface{}]interface{}{
							"key":    t.Key,
							"operator": t.Operator,
							"value":  t.Value,
							"effect": t.Effect,
						}
					}
					podSpec["tolerations"] = tolerations
				}
				
				// Apply affinity
				if env.Affinity != nil {
					affinity := make(map[interface{}]interface{})
					
					if env.Affinity.NodeAffinity != nil {
						nodeAffinity := make(map[interface{}]interface{})
						// Convert NodeAffinity
						affinity["nodeAffinity"] = nodeAffinity
					}
					
					if env.Affinity.PodAffinity != nil {
						podAffinity := make(map[interface{}]interface{})
						// Convert PodAffinity
						affinity["podAffinity"] = podAffinity
					}
					
					if env.Affinity.PodAntiAffinity != nil {
						podAntiAffinity := make(map[interface{}]interface{})
						// Convert PodAntiAffinity
						affinity["podAntiAffinity"] = podAntiAffinity
					}
					
					if len(affinity) > 0 {
						podSpec["affinity"] = affinity
					}
				}
			}
		}
	}
}

// applyNetworkPolicies applies network policies
func (c *Customizer) applyNetworkPolicies(customCtx *CustomizationContext, networkPolicies []NetworkPolicyTemplate) error {
	c.logger.Debug("Applying network policies", zap.Int("policies", len(networkPolicies)))
	
	for _, policyTemplate := range networkPolicies {
		networkPolicy := c.createNetworkPolicyManifest(policyTemplate, customCtx)
		filename := fmt.Sprintf("networkpolicy-%s.yaml", policyTemplate.Name)
		customCtx.Files[filename] = networkPolicy
	}
	
	return nil
}

func (c *Customizer) createNetworkPolicyManifest(template NetworkPolicyTemplate, customCtx *CustomizationContext) string {
	policy := map[string]interface{}{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "NetworkPolicy",
		"metadata": map[string]interface{}{
			"name":      template.Name,
			"namespace": customCtx.TargetNamespace,
			"labels": map[string]interface{}{
				"nephoran.io/intent-name": customCtx.Intent.Name,
				"nephoran.io/managed":     "true",
			},
		},
		"spec": map[string]interface{}{
			"podSelector": map[string]interface{}{
				"matchLabels": c.parsePodSelector(template.PodSelector),
			},
			"policyTypes": []interface{}{"Ingress", "Egress"},
		},
	}
	
	spec := policy["spec"].(map[string]interface{})
	
	// Add ingress rules
	if len(template.Ingress) > 0 {
		ingress := make([]interface{}, len(template.Ingress))
		for i, rule := range template.Ingress {
			ingress[i] = c.parseNetworkPolicyRule(rule)
		}
		spec["ingress"] = ingress
	}
	
	// Add egress rules
	if len(template.Egress) > 0 {
		egress := make([]interface{}, len(template.Egress))
		for i, rule := range template.Egress {
			egress[i] = c.parseNetworkPolicyRule(rule)
		}
		spec["egress"] = egress
	}
	
	yamlBytes, _ := yaml.Marshal(policy)
	return string(yamlBytes)
}

func (c *Customizer) parsePodSelector(selector string) map[string]interface{} {
	// Simple selector parsing - can be extended
	if selector == "" {
		return map[string]interface{}{}
	}
	
	return map[string]interface{}{
		"app": selector,
	}
}

func (c *Customizer) parseNetworkPolicyRule(rule string) map[string]interface{} {
	// Simple rule parsing - can be extended
	return map[string]interface{}{
		"from": []interface{}{
			map[string]interface{}{
				"podSelector": map[string]interface{}{},
			},
		},
	}
}

// applyServiceMeshConfiguration applies service mesh configuration
func (c *Customizer) applyServiceMeshConfiguration(customCtx *CustomizationContext, serviceMesh *ServiceMeshProfile) error {
	c.logger.Debug("Applying service mesh configuration")
	
	if !serviceMesh.Enabled {
		return nil
	}
	
	// Apply sidecar injection annotation
	if serviceMesh.InjectSidecar {
		for filename, content := range customCtx.Files {
			if c.isKubernetesManifest(content) {
				customized, err := c.addSidecarInjectionAnnotation(content)
				if err != nil {
					continue
				}
				customCtx.Files[filename] = customized
			}
		}
	}
	
	// Create service mesh specific resources
	if err := c.createServiceMeshResources(customCtx, serviceMesh); err != nil {
		return err
	}
	
	return nil
}

func (c *Customizer) addSidecarInjectionAnnotation(content string) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	if kind, ok := obj["kind"].(string); ok {
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			if spec, ok := obj["spec"].(map[interface{}]interface{}); ok {
				if template, ok := spec["template"].(map[interface{}]interface{}); ok {
					if metadata, ok := template["metadata"].(map[interface{}]interface{}); ok {
						var annotations map[interface{}]interface{}
						if existing, exists := metadata["annotations"].(map[interface{}]interface{}); exists {
							annotations = existing
						} else {
							annotations = make(map[interface{}]interface{})
							metadata["annotations"] = annotations
						}
						annotations["sidecar.istio.io/inject"] = "true"
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

func (c *Customizer) createServiceMeshResources(customCtx *CustomizationContext, serviceMesh *ServiceMeshProfile) error {
	// Create VirtualService
	if serviceMesh.TrafficPolicy != "" {
		virtualService := c.createVirtualService(customCtx, serviceMesh)
		customCtx.Files["virtualservice.yaml"] = virtualService
	}
	
	// Create DestinationRule
	if serviceMesh.MTLSMode != "" || serviceMesh.CircuitBreaker {
		destinationRule := c.createDestinationRule(customCtx, serviceMesh)
		customCtx.Files["destinationrule.yaml"] = destinationRule
	}
	
	return nil
}

func (c *Customizer) createVirtualService(customCtx *CustomizationContext, serviceMesh *ServiceMeshProfile) string {
	vs := map[string]interface{}{
		"apiVersion": "networking.istio.io/v1beta1",
		"kind":       "VirtualService",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-vs", customCtx.Intent.Name),
			"namespace": customCtx.TargetNamespace,
		},
		"spec": map[string]interface{}{
			"hosts": []interface{}{"*"},
			"http": []interface{}{
				map[string]interface{}{
					"route": []interface{}{
						map[string]interface{}{
							"destination": map[string]interface{}{
								"host": customCtx.Intent.Name,
							},
						},
					},
				},
			},
		},
	}
	
	yamlBytes, _ := yaml.Marshal(vs)
	return string(yamlBytes)
}

func (c *Customizer) createDestinationRule(customCtx *CustomizationContext, serviceMesh *ServiceMeshProfile) string {
	dr := map[string]interface{}{
		"apiVersion": "networking.istio.io/v1beta1",
		"kind":       "DestinationRule",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-dr", customCtx.Intent.Name),
			"namespace": customCtx.TargetNamespace,
		},
		"spec": map[string]interface{}{
			"host": customCtx.Intent.Name,
			"trafficPolicy": map[string]interface{}{
				"tls": map[string]interface{}{
					"mode": serviceMesh.MTLSMode,
				},
			},
		},
	}
	
	if serviceMesh.CircuitBreaker {
		trafficPolicy := dr["spec"].(map[string]interface{})["trafficPolicy"].(map[string]interface{})
		trafficPolicy["outlierDetection"] = map[string]interface{}{
			"consecutive5xxErrors": 3,
			"interval":             "30s",
			"baseEjectionTime":     "30s",
		}
	}
	
	yamlBytes, _ := yaml.Marshal(dr)
	return string(yamlBytes)
}

// applyMonitoringConfiguration applies monitoring configuration
func (c *Customizer) applyMonitoringConfiguration(customCtx *CustomizationContext, monitoring *MonitoringProfile) error {
	c.logger.Debug("Applying monitoring configuration")
	
	if !monitoring.Enabled {
		return nil
	}
	
	// Add monitoring annotations to workloads
	for filename, content := range customCtx.Files {
		if c.isKubernetesManifest(content) {
			customized, err := c.addMonitoringAnnotations(content, monitoring)
			if err != nil {
				continue
			}
			customCtx.Files[filename] = customized
		}
	}
	
	// Create monitoring resources
	if err := c.createMonitoringResources(customCtx, monitoring); err != nil {
		return err
	}
	
	return nil
}

func (c *Customizer) addMonitoringAnnotations(content string, monitoring *MonitoringProfile) (string, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
		return content, nil
	}

	if kind, ok := obj["kind"].(string); ok {
		switch kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			if spec, ok := obj["spec"].(map[interface{}]interface{}); ok {
				if template, ok := spec["template"].(map[interface{}]interface{}); ok {
					if metadata, ok := template["metadata"].(map[interface{}]interface{}); ok {
						var annotations map[interface{}]interface{}
						if existing, exists := metadata["annotations"].(map[interface{}]interface{}); exists {
							annotations = existing
						} else {
							annotations = make(map[interface{}]interface{})
							metadata["annotations"] = annotations
						}
						
						if monitoring.MetricsEnabled {
							annotations["prometheus.io/scrape"] = "true"
							annotations["prometheus.io/port"] = "8080"
							annotations["prometheus.io/path"] = "/metrics"
						}
						
						if monitoring.TracingEnabled {
							annotations["sidecar.jaegertracing.io/inject"] = "true"
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

func (c *Customizer) createMonitoringResources(customCtx *CustomizationContext, monitoring *MonitoringProfile) error {
	// Create ServiceMonitor for Prometheus
	if monitoring.MetricsEnabled {
		serviceMonitor := c.createServiceMonitor(customCtx, monitoring)
		customCtx.Files["servicemonitor.yaml"] = serviceMonitor
	}
	
	// Create PrometheusRule for alerting
	if monitoring.AlertingEnabled && len(monitoring.AlertRules) > 0 {
		prometheusRule := c.createPrometheusRule(customCtx, monitoring)
		customCtx.Files["prometheusrule.yaml"] = prometheusRule
	}
	
	return nil
}

func (c *Customizer) createServiceMonitor(customCtx *CustomizationContext, monitoring *MonitoringProfile) string {
	sm := map[string]interface{}{
		"apiVersion": "monitoring.coreos.com/v1",
		"kind":       "ServiceMonitor",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-monitor", customCtx.Intent.Name),
			"namespace": customCtx.TargetNamespace,
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"nephoran.io/intent-name": customCtx.Intent.Name,
				},
			},
			"endpoints": []interface{}{
				map[string]interface{}{
					"port":     "http",
					"interval": monitoring.MetricsScrapeInterval,
				},
			},
		},
	}
	
	yamlBytes, _ := yaml.Marshal(sm)
	return string(yamlBytes)
}

func (c *Customizer) createPrometheusRule(customCtx *CustomizationContext, monitoring *MonitoringProfile) string {
	pr := map[string]interface{}{
		"apiVersion": "monitoring.coreos.com/v1",
		"kind":       "PrometheusRule",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-alerts", customCtx.Intent.Name),
			"namespace": customCtx.TargetNamespace,
		},
		"spec": map[string]interface{}{
			"groups": []interface{}{
				map[string]interface{}{
					"name": fmt.Sprintf("%s.rules", customCtx.Intent.Name),
					"rules": c.convertAlertRules(monitoring.AlertRules),
				},
			},
		},
	}
	
	yamlBytes, _ := yaml.Marshal(pr)
	return string(yamlBytes)
}

func (c *Customizer) convertAlertRules(alertRules []string) []interface{} {
	rules := make([]interface{}, len(alertRules))
	for i, rule := range alertRules {
		// Simple rule conversion - can be extended
		rules[i] = map[string]interface{}{
			"alert": fmt.Sprintf("Rule%d", i+1),
			"expr":  rule,
			"for":   "5m",
			"labels": map[string]interface{}{
				"severity": "warning",
			},
			"annotations": map[string]interface{}{
				"summary": fmt.Sprintf("Alert rule %d triggered", i+1),
			},
		}
	}
	return rules
}

// PolicyEngine methods
func (pe *PolicyEngine) EvaluatePolicies(customCtx *CustomizationContext) []*CustomizationPolicy {
	pe.mutex.RLock()
	defer pe.mutex.RUnlock()
	
	var applicablePolicies []*CustomizationPolicy
	
	for _, policy := range pe.policies {
		if !policy.Enabled {
			continue
		}
		
		if pe.policyApplies(policy, customCtx) {
			applicablePolicies = append(applicablePolicies, policy)
		}
	}
	
	return applicablePolicies
}

func (pe *PolicyEngine) policyApplies(policy *CustomizationPolicy, customCtx *CustomizationContext) bool {
	// Check scope
	if len(policy.Scope.Namespaces) > 0 {
		found := false
		for _, ns := range policy.Scope.Namespaces {
			if ns == customCtx.TargetNamespace {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	if len(policy.Scope.Components) > 0 {
		found := false
		for _, policyComponent := range policy.Scope.Components {
			for _, intentComponent := range customCtx.Intent.Spec.TargetComponents {
				if policyComponent == intentComponent {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}
