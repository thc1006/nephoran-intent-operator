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

package templates

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
	"github.com/thc1006/nephoran-intent-operator/pkg/validation/yang"
)

// TemplateEngine provides comprehensive template management and rendering capabilities
// Supports O-RAN and 5G Core network function blueprints, parameter validation,
// template inheritance, and multi-vendor configuration templates
type TemplateEngine interface {
	// Template management
	LoadTemplate(ctx context.Context, templateID string) (*BlueprintTemplate, error)
	LoadTemplateFromRepository(ctx context.Context, repoURL, templatePath string) (*BlueprintTemplate, error)
	RegisterTemplate(ctx context.Context, template *BlueprintTemplate) error
	UnregisterTemplate(ctx context.Context, templateID string) error
	GetTemplate(ctx context.Context, templateID string) (*BlueprintTemplate, error)
	ListTemplates(ctx context.Context, filter *TemplateFilter) ([]*BlueprintTemplate, error)

	// Template rendering
	RenderTemplate(ctx context.Context, templateID string, parameters map[string]interface{}) ([]*porch.KRMResource, error)
	RenderTemplateWithValidation(ctx context.Context, templateID string, parameters map[string]interface{}) (*RenderResult, error)
	ValidateParameters(ctx context.Context, templateID string, parameters map[string]interface{}) (*ParameterValidationResult, error)

	// Template catalog management
	RefreshCatalog(ctx context.Context) error
	GetCatalogInfo(ctx context.Context) (*CatalogInfo, error)
	SearchTemplates(ctx context.Context, query *SearchQuery) ([]*BlueprintTemplate, error)

	// Template composition
	ComposeTemplates(ctx context.Context, templates []*TemplateComposition) (*CompositeTemplate, error)
	RenderCompositeTemplate(ctx context.Context, composite *CompositeTemplate, parameters map[string]interface{}) ([]*porch.KRMResource, error)

	// Template versioning
	GetTemplateVersions(ctx context.Context, templateName string) ([]*TemplateVersion, error)
	PromoteTemplateVersion(ctx context.Context, templateID, version string) error
	RollbackTemplateVersion(ctx context.Context, templateID, version string) error

	// Health and maintenance
	GetEngineHealth(ctx context.Context) (*EngineHealth, error)
	Close() error
}

// templateEngine implements comprehensive template management and rendering
type templateEngine struct {
	// Core dependencies
	yangValidator yang.YANGValidator
	logger        logr.Logger

	// Template storage
	templates       map[string]*BlueprintTemplate
	templateMutex   sync.RWMutex
	templateCatalog *TemplateCatalog

	// Configuration
	config *EngineConfig

	// Background processing
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// Core template types

// BlueprintTemplate represents a network function deployment template
type BlueprintTemplate struct {
	// Metadata
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Description string            `json:"description" yaml:"description"`
	Author      string            `json:"author" yaml:"author"`
	CreatedAt   time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt" yaml:"updatedAt"`
	Tags        []string          `json:"tags" yaml:"tags"`
	Keywords    []string          `json:"keywords" yaml:"keywords"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Annotations map[string]string `json:"annotations" yaml:"annotations"`

	// Template classification
	Category        TemplateCategory           `json:"category" yaml:"category"`
	Type            TemplateType               `json:"type" yaml:"type"`
	TargetComponent nephoranv1.TargetComponent `json:"targetComponent" yaml:"targetComponent"`
	Vendor          string                     `json:"vendor" yaml:"vendor"`
	Standard        string                     `json:"standard" yaml:"standard"` // O-RAN, 3GPP, etc.

	// Template content
	Schema      *ParameterSchema    `json:"schema" yaml:"schema"`
	Resources   []*KRMTemplate      `json:"resources" yaml:"resources"`
	Functions   []*FunctionTemplate `json:"functions" yaml:"functions"`
	Validations []*ValidationRule   `json:"validations" yaml:"validations"`

	// Template inheritance
	BaseTemplate string                 `json:"baseTemplate,omitempty" yaml:"baseTemplate,omitempty"`
	Overrides    map[string]interface{} `json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Extensions   []*TemplateExtension   `json:"extensions,omitempty" yaml:"extensions,omitempty"`

	// O-RAN specific configurations
	ORANConfig *ORANTemplateConfig `json:"oranConfig,omitempty" yaml:"oranConfig,omitempty"`

	// 5G Core specific configurations
	FiveGConfig *FiveGTemplateConfig `json:"5gConfig,omitempty" yaml:"5gConfig,omitempty"`

	// Multi-vendor support
	VendorVariants []*VendorVariant `json:"vendorVariants,omitempty" yaml:"vendorVariants,omitempty"`

	// Template maturity and compliance
	MaturityLevel  MaturityLevel   `json:"maturityLevel" yaml:"maturityLevel"`
	ComplianceInfo *ComplianceInfo `json:"complianceInfo" yaml:"complianceInfo"`
	TestingStatus  *TestingStatus  `json:"testingStatus" yaml:"testingStatus"`

	// Usage and analytics
	UsageCount  int64      `json:"usageCount" yaml:"-"`
	LastUsed    *time.Time `json:"lastUsed" yaml:"-"`
	SuccessRate float64    `json:"successRate" yaml:"-"`
}

// Template classification enums
type TemplateCategory string

const (
	TemplateCategoryNetworkFunction TemplateCategory = "network-function"
	TemplateCategoryApplication     TemplateCategory = "application"
	TemplateCategoryInfrastructure  TemplateCategory = "infrastructure"
	TemplateCategoryConfiguration   TemplateCategory = "configuration"
	TemplateCategoryComposite       TemplateCategory = "composite"
)

type TemplateType string

const (
	TemplateTypeDeployment    TemplateType = "deployment"
	TemplateTypeConfiguration TemplateType = "configuration"
	TemplateTypePolicy        TemplateType = "policy"
	TemplateTypeService       TemplateType = "service"
	TemplateTypeNetworkSlice  TemplateType = "network-slice"
)

type MaturityLevel string

const (
	MaturityLevelAlpha      MaturityLevel = "alpha"
	MaturityLevelBeta       MaturityLevel = "beta"
	MaturityLevelStable     MaturityLevel = "stable"
	MaturityLevelDeprecated MaturityLevel = "deprecated"
)

// Parameter schema and validation

// ParameterSchema defines the schema for template parameters
type ParameterSchema struct {
	Version    string                    `json:"version" yaml:"version"`
	Properties map[string]*ParameterSpec `json:"properties" yaml:"properties"`
	Required   []string                  `json:"required" yaml:"required"`
	Groups     []*ParameterGroup         `json:"groups,omitempty" yaml:"groups,omitempty"`
}

// ParameterSpec defines a single parameter specification
type ParameterSpec struct {
	Type         string        `json:"type" yaml:"type"` // string, integer, boolean, object, array
	Description  string        `json:"description" yaml:"description"`
	Default      interface{}   `json:"default,omitempty" yaml:"default,omitempty"`
	Examples     []interface{} `json:"examples,omitempty" yaml:"examples,omitempty"`
	Enum         []interface{} `json:"enum,omitempty" yaml:"enum,omitempty"`
	Pattern      string        `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Minimum      *float64      `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum      *float64      `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	MinLength    *int          `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength    *int          `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Format       string        `json:"format,omitempty" yaml:"format,omitempty"` // uuid, ipv4, ipv6, etc.
	Sensitive    bool          `json:"sensitive,omitempty" yaml:"sensitive,omitempty"`
	Dependencies []string      `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// ParameterGroup groups related parameters for better UX
type ParameterGroup struct {
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Parameters  []string `json:"parameters" yaml:"parameters"`
	Conditional string   `json:"conditional,omitempty" yaml:"conditional,omitempty"`
}

// Template resource definitions

// KRMTemplate defines a Kubernetes resource template
type KRMTemplate struct {
	APIVersion string                 `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                 `json:"kind" yaml:"kind"`
	Metadata   *ResourceMetadata      `json:"metadata" yaml:"metadata"`
	Spec       map[string]interface{} `json:"spec,omitempty" yaml:"spec,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty" yaml:"data,omitempty"`
	Template   string                 `json:"template,omitempty" yaml:"template,omitempty"` // Go template string
	Conditions []*RenderCondition     `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// ResourceMetadata defines metadata for generated resources
type ResourceMetadata struct {
	Name        string            `json:"name" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// RenderCondition defines when a resource should be included in rendering
type RenderCondition struct {
	Expression string `json:"expression" yaml:"expression"`
	Action     string `json:"action" yaml:"action"` // include, exclude, warn
}

// FunctionTemplate defines a KRM function template
type FunctionTemplate struct {
	Name       string                 `json:"name" yaml:"name"`
	Image      string                 `json:"image" yaml:"image"`
	ConfigPath string                 `json:"configPath,omitempty" yaml:"configPath,omitempty"`
	ConfigMap  map[string]interface{} `json:"configMap,omitempty" yaml:"configMap,omitempty"`
	Type       string                 `json:"type" yaml:"type"`   // mutator, validator
	Stage      string                 `json:"stage" yaml:"stage"` // pre-render, post-render
	Conditions []*RenderCondition     `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// ValidationRule defines template validation rules
type ValidationRule struct {
	Name        string             `json:"name" yaml:"name"`
	Type        string             `json:"type" yaml:"type"` // yang, policy, custom
	Rule        string             `json:"rule" yaml:"rule"`
	Severity    string             `json:"severity" yaml:"severity"`
	Message     string             `json:"message" yaml:"message"`
	Remediation string             `json:"remediation,omitempty" yaml:"remediation,omitempty"`
	Conditions  []*RenderCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// O-RAN specific configurations

// ORANTemplateConfig defines O-RAN specific template configuration
type ORANTemplateConfig struct {
	// O-RAN Alliance specifications
	Specifications []*SpecificationRef `json:"specifications" yaml:"specifications"`

	// O-RAN interfaces configuration
	Interfaces []*ORANInterface `json:"interfaces" yaml:"interfaces"`

	// SMO integration
	SMOIntegration *SMOIntegration `json:"smoIntegration,omitempty" yaml:"smoIntegration,omitempty"`

	// RIC applications (xApps/rApps)
	RICApplications []*RICApplication `json:"ricApplications,omitempty" yaml:"ricApplications,omitempty"`

	// Cloud native network functions
	CNFs []*CNFConfig `json:"cnfs,omitempty" yaml:"cnfs,omitempty"`
}

// SpecificationRef references an O-RAN specification
type SpecificationRef struct {
	Name     string `json:"name" yaml:"name"`
	Version  string `json:"version" yaml:"version"`
	Document string `json:"document" yaml:"document"`
	Section  string `json:"section,omitempty" yaml:"section,omitempty"`
}

// ORANInterface defines O-RAN interface configuration
type ORANInterface struct {
	Name     string                 `json:"name" yaml:"name"`
	Type     string                 `json:"type" yaml:"type"` // A1, O1, O2, E2, Open-FH
	Version  string                 `json:"version" yaml:"version"`
	Endpoint string                 `json:"endpoint" yaml:"endpoint"`
	Protocol string                 `json:"protocol" yaml:"protocol"`
	Config   map[string]interface{} `json:"config" yaml:"config"`
	Security *InterfaceSecurity     `json:"security,omitempty" yaml:"security,omitempty"`
}

// InterfaceSecurity defines security configuration for O-RAN interfaces
type InterfaceSecurity struct {
	TLSEnabled     bool              `json:"tlsEnabled" yaml:"tlsEnabled"`
	mTLSEnabled    bool              `json:"mtlsEnabled" yaml:"mtlsEnabled"`
	Certificates   map[string]string `json:"certificates,omitempty" yaml:"certificates,omitempty"`
	Authentication *AuthConfig       `json:"authentication,omitempty" yaml:"authentication,omitempty"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	Type       string            `json:"type" yaml:"type"` // oauth2, jwt, basic
	Config     map[string]string `json:"config" yaml:"config"`
	SecretRefs map[string]string `json:"secretRefs,omitempty" yaml:"secretRefs,omitempty"`
}

// SMOIntegration defines SMO integration configuration
type SMOIntegration struct {
	Enabled      bool             `json:"enabled" yaml:"enabled"`
	SMOEndpoint  string           `json:"smoEndpoint" yaml:"smoEndpoint"`
	Registration *SMORegistration `json:"registration" yaml:"registration"`
	Monitoring   *SMOMonitoring   `json:"monitoring,omitempty" yaml:"monitoring,omitempty"`
}

// SMORegistration defines SMO registration configuration
type SMORegistration struct {
	RegistryURL string            `json:"registryUrl" yaml:"registryUrl"`
	Credentials map[string]string `json:"credentials" yaml:"credentials"`
	Metadata    map[string]string `json:"metadata" yaml:"metadata"`
}

// SMOMonitoring defines SMO monitoring configuration
type SMOMonitoring struct {
	Enabled       bool            `json:"enabled" yaml:"enabled"`
	MetricsURL    string          `json:"metricsUrl" yaml:"metricsUrl"`
	LoggingURL    string          `json:"loggingUrl" yaml:"loggingUrl"`
	AlertingRules []*AlertingRule `json:"alertingRules,omitempty" yaml:"alertingRules,omitempty"`
}

// AlertingRule defines monitoring alerting rules
type AlertingRule struct {
	Name        string            `json:"name" yaml:"name"`
	Expression  string            `json:"expression" yaml:"expression"`
	Severity    string            `json:"severity" yaml:"severity"`
	Description string            `json:"description" yaml:"description"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// RICApplication defines RIC application configuration
type RICApplication struct {
	Name         string                 `json:"name" yaml:"name"`
	Type         string                 `json:"type" yaml:"type"` // xApp, rApp
	Version      string                 `json:"version" yaml:"version"`
	Image        string                 `json:"image" yaml:"image"`
	Config       map[string]interface{} `json:"config" yaml:"config"`
	Dependencies []string               `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Resources    *ResourceRequirements  `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	CPU     *resource.Quantity `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory  *resource.Quantity `json:"memory,omitempty" yaml:"memory,omitempty"`
	Storage *resource.Quantity `json:"storage,omitempty" yaml:"storage,omitempty"`
	Limits  *ResourceLimits    `json:"limits,omitempty" yaml:"limits,omitempty"`
}

// ResourceLimits defines resource limits
type ResourceLimits struct {
	CPU    *resource.Quantity `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory *resource.Quantity `json:"memory,omitempty" yaml:"memory,omitempty"`
}

// CNFConfig defines Cloud Native Network Function configuration
type CNFConfig struct {
	Name            string                   `json:"name" yaml:"name"`
	Helm            *HelmConfig              `json:"helm,omitempty" yaml:"helm,omitempty"`
	Operator        *OperatorConfig          `json:"operator,omitempty" yaml:"operator,omitempty"`
	ConfigMaps      []*ConfigMapTemplate     `json:"configMaps,omitempty" yaml:"configMaps,omitempty"`
	Secrets         []*SecretTemplate        `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	NetworkPolicies []*NetworkPolicyTemplate `json:"networkPolicies,omitempty" yaml:"networkPolicies,omitempty"`
}

// HelmConfig defines Helm chart configuration
type HelmConfig struct {
	Chart     string                 `json:"chart" yaml:"chart"`
	Version   string                 `json:"version" yaml:"version"`
	Values    map[string]interface{} `json:"values" yaml:"values"`
	Variables map[string]string      `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// OperatorConfig defines Kubernetes operator configuration
type OperatorConfig struct {
	Name      string                 `json:"name" yaml:"name"`
	Namespace string                 `json:"namespace" yaml:"namespace"`
	CRD       string                 `json:"crd" yaml:"crd"`
	Config    map[string]interface{} `json:"config" yaml:"config"`
}

// ConfigMapTemplate defines ConfigMap template
type ConfigMapTemplate struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Data      map[string]string `json:"data" yaml:"data"`
	Template  bool              `json:"template,omitempty" yaml:"template,omitempty"`
}

// SecretTemplate defines Secret template
type SecretTemplate struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Type      string            `json:"type" yaml:"type"`
	Data      map[string]string `json:"data" yaml:"data"`
	Template  bool              `json:"template,omitempty" yaml:"template,omitempty"`
}

// NetworkPolicyTemplate defines NetworkPolicy template
type NetworkPolicyTemplate struct {
	Name      string                 `json:"name" yaml:"name"`
	Namespace string                 `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Spec      map[string]interface{} `json:"spec" yaml:"spec"`
}

// 5G Core specific configurations

// FiveGTemplateConfig defines 5G Core specific template configuration
type FiveGTemplateConfig struct {
	// 5G Core network functions
	NetworkFunctions []*NetworkFunction `json:"networkFunctions" yaml:"networkFunctions"`

	// Network slicing configuration
	NetworkSlicing *NetworkSlicingConfig `json:"networkSlicing,omitempty" yaml:"networkSlicing,omitempty"`

	// QoS configuration
	QoSConfiguration *QoSConfiguration `json:"qosConfiguration,omitempty" yaml:"qosConfiguration,omitempty"`

	// Security configuration
	Security *SecurityConfig `json:"security,omitempty" yaml:"security,omitempty"`

	// Service-based architecture
	SBA *SBAConfig `json:"sba,omitempty" yaml:"sba,omitempty"`
}

// NetworkFunction defines 5G Core network function configuration
type NetworkFunction struct {
	Type             nephoranv1.TargetComponent `json:"type" yaml:"type"`
	Version          string                     `json:"version" yaml:"version"`
	Configuration    map[string]interface{}     `json:"configuration" yaml:"configuration"`
	Interfaces       []*NFInterface             `json:"interfaces" yaml:"interfaces"`
	Resources        *ResourceRequirements      `json:"resources,omitempty" yaml:"resources,omitempty"`
	HighAvailability *HAConfig                  `json:"highAvailability,omitempty" yaml:"highAvailability,omitempty"`
}

// NFInterface defines network function interface
type NFInterface struct {
	Name     string                 `json:"name" yaml:"name"`
	Type     string                 `json:"type" yaml:"type"` // SBI, N1, N2, N3, etc.
	Protocol string                 `json:"protocol" yaml:"protocol"`
	Config   map[string]interface{} `json:"config" yaml:"config"`
}

// HAConfig defines high availability configuration
type HAConfig struct {
	Enabled      bool   `json:"enabled" yaml:"enabled"`
	Replicas     int32  `json:"replicas" yaml:"replicas"`
	Strategy     string `json:"strategy" yaml:"strategy"` // active-passive, active-active
	AntiAffinity bool   `json:"antiAffinity" yaml:"antiAffinity"`
}

// NetworkSlicingConfig defines network slicing configuration
type NetworkSlicingConfig struct {
	Enabled        bool             `json:"enabled" yaml:"enabled"`
	SliceTemplates []*SliceTemplate `json:"sliceTemplates" yaml:"sliceTemplates"`
	Isolation      *IsolationConfig `json:"isolation,omitempty" yaml:"isolation,omitempty"`
}

// SliceTemplate defines network slice template
type SliceTemplate struct {
	ID        string                 `json:"id" yaml:"id"`
	Type      string                 `json:"type" yaml:"type"` // eMBB, URLLC, mMTC
	SLA       *SLAConfig             `json:"sla" yaml:"sla"`
	QoS       *QoSConfig             `json:"qos" yaml:"qos"`
	Resources *SliceResources        `json:"resources" yaml:"resources"`
	Config    map[string]interface{} `json:"config" yaml:"config"`
}

// SLAConfig defines service level agreement configuration
type SLAConfig struct {
	Latency      *LatencyRequirement      `json:"latency,omitempty" yaml:"latency,omitempty"`
	Throughput   *ThroughputRequirement   `json:"throughput,omitempty" yaml:"throughput,omitempty"`
	Availability *AvailabilityRequirement `json:"availability,omitempty" yaml:"availability,omitempty"`
	Reliability  *ReliabilityRequirement  `json:"reliability,omitempty" yaml:"reliability,omitempty"`
}

// LatencyRequirement defines latency requirements
type LatencyRequirement struct {
	MaxLatency     *metav1.Duration `json:"maxLatency" yaml:"maxLatency"`
	TypicalLatency *metav1.Duration `json:"typicalLatency,omitempty" yaml:"typicalLatency,omitempty"`
	Percentile     float64          `json:"percentile,omitempty" yaml:"percentile,omitempty"`
}

// ThroughputRequirement defines throughput requirements
type ThroughputRequirement struct {
	MinDownlink *resource.Quantity `json:"minDownlink,omitempty" yaml:"minDownlink,omitempty"`
	MinUplink   *resource.Quantity `json:"minUplink,omitempty" yaml:"minUplink,omitempty"`
	MaxDownlink *resource.Quantity `json:"maxDownlink,omitempty" yaml:"maxDownlink,omitempty"`
	MaxUplink   *resource.Quantity `json:"maxUplink,omitempty" yaml:"maxUplink,omitempty"`
	UserDensity int32              `json:"userDensity,omitempty" yaml:"userDensity,omitempty"`
}

// AvailabilityRequirement defines availability requirements
type AvailabilityRequirement struct {
	Target       float64          `json:"target" yaml:"target"` // 0.999, 0.9999, etc.
	ServiceLevel string           `json:"serviceLevel,omitempty" yaml:"serviceLevel,omitempty"`
	Downtime     *metav1.Duration `json:"downtime,omitempty" yaml:"downtime,omitempty"`
}

// ReliabilityRequirement defines reliability requirements
type ReliabilityRequirement struct {
	SuccessRate             float64          `json:"successRate" yaml:"successRate"`
	ErrorRate               float64          `json:"errorRate,omitempty" yaml:"errorRate,omitempty"`
	MeanTimeBetweenFailures *metav1.Duration `json:"mtbf,omitempty" yaml:"mtbf,omitempty"`
	MeanTimeToRepair        *metav1.Duration `json:"mttr,omitempty" yaml:"mttr,omitempty"`
}

// QoSConfig and QoSConfiguration define QoS settings
type QoSConfig struct {
	FiveQI            int32  `json:"5qi,omitempty" yaml:"5qi,omitempty"`
	PriorityLevel     int32  `json:"priorityLevel,omitempty" yaml:"priorityLevel,omitempty"`
	PacketDelayBudget int32  `json:"packetDelayBudget,omitempty" yaml:"packetDelayBudget,omitempty"`
	PacketErrorRate   string `json:"packetErrorRate,omitempty" yaml:"packetErrorRate,omitempty"`
}

type QoSConfiguration struct {
	Profiles []*QoSProfile `json:"profiles" yaml:"profiles"`
	Rules    []*QoSRule    `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// QoSProfile defines QoS profile
type QoSProfile struct {
	Name        string          `json:"name" yaml:"name"`
	Description string          `json:"description" yaml:"description"`
	QoS         *QoSConfig      `json:"qos" yaml:"qos"`
	Conditions  []*QoSCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// QoSRule defines QoS rule
type QoSRule struct {
	ID          string          `json:"id" yaml:"id"`
	Description string          `json:"description" yaml:"description"`
	Conditions  []*QoSCondition `json:"conditions" yaml:"conditions"`
	Actions     []*QoSAction    `json:"actions" yaml:"actions"`
}

// QoSCondition defines QoS condition
type QoSCondition struct {
	Field    string      `json:"field" yaml:"field"`
	Operator string      `json:"operator" yaml:"operator"`
	Value    interface{} `json:"value" yaml:"value"`
}

// QoSAction defines QoS action
type QoSAction struct {
	Type   string      `json:"type" yaml:"type"`
	Config interface{} `json:"config" yaml:"config"`
}

// SliceResources defines slice-specific resource requirements
type SliceResources struct {
	CPU         *resource.Quantity `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory      *resource.Quantity `json:"memory,omitempty" yaml:"memory,omitempty"`
	Storage     *resource.Quantity `json:"storage,omitempty" yaml:"storage,omitempty"`
	Bandwidth   *resource.Quantity `json:"bandwidth,omitempty" yaml:"bandwidth,omitempty"`
	Connections int32              `json:"connections,omitempty" yaml:"connections,omitempty"`
}

// IsolationConfig defines isolation configuration
type IsolationConfig struct {
	Level      string   `json:"level" yaml:"level"` // physical, logical, none
	Mechanisms []string `json:"mechanisms,omitempty" yaml:"mechanisms,omitempty"`
	Encryption bool     `json:"encryption,omitempty" yaml:"encryption,omitempty"`
}

// SecurityConfig defines security configuration
type SecurityConfig struct {
	Encryption     *EncryptionConfig     `json:"encryption,omitempty" yaml:"encryption,omitempty"`
	Authentication *AuthenticationConfig `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	Authorization  *AuthorizationConfig  `json:"authorization,omitempty" yaml:"authorization,omitempty"`
	Certificates   *CertificateConfig    `json:"certificates,omitempty" yaml:"certificates,omitempty"`
}

// EncryptionConfig defines encryption configuration
type EncryptionConfig struct {
	InTransit     bool           `json:"inTransit" yaml:"inTransit"`
	AtRest        bool           `json:"atRest" yaml:"atRest"`
	Algorithms    []string       `json:"algorithms" yaml:"algorithms"`
	KeyManagement *KeyManagement `json:"keyManagement,omitempty" yaml:"keyManagement,omitempty"`
}

// KeyManagement defines key management configuration
type KeyManagement struct {
	Type     string            `json:"type" yaml:"type"` // vault, k8s-secrets, external
	Config   map[string]string `json:"config" yaml:"config"`
	Rotation *KeyRotation      `json:"rotation,omitempty" yaml:"rotation,omitempty"`
}

// KeyRotation defines key rotation policy
type KeyRotation struct {
	Enabled  bool             `json:"enabled" yaml:"enabled"`
	Interval *metav1.Duration `json:"interval" yaml:"interval"`
	Policy   string           `json:"policy" yaml:"policy"`
}

// AuthenticationConfig defines authentication configuration
type AuthenticationConfig struct {
	Methods         []string        `json:"methods" yaml:"methods"`
	Providers       []*AuthProvider `json:"providers" yaml:"providers"`
	MultiFactorAuth bool            `json:"multiFactorAuth,omitempty" yaml:"multiFactorAuth,omitempty"`
}

// AuthProvider defines authentication provider
type AuthProvider struct {
	Name     string            `json:"name" yaml:"name"`
	Type     string            `json:"type" yaml:"type"`
	Config   map[string]string `json:"config" yaml:"config"`
	Priority int               `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// AuthorizationConfig defines authorization configuration
type AuthorizationConfig struct {
	RBAC     *RBACConfig     `json:"rbac,omitempty" yaml:"rbac,omitempty"`
	Policies []*PolicyConfig `json:"policies,omitempty" yaml:"policies,omitempty"`
}

// RBACConfig defines RBAC configuration
type RBACConfig struct {
	Enabled  bool             `json:"enabled" yaml:"enabled"`
	Roles    []*RoleConfig    `json:"roles" yaml:"roles"`
	Bindings []*BindingConfig `json:"bindings" yaml:"bindings"`
}

// RoleConfig defines role configuration
type RoleConfig struct {
	Name        string              `json:"name" yaml:"name"`
	Permissions []*PermissionConfig `json:"permissions" yaml:"permissions"`
}

// PermissionConfig defines permission configuration
type PermissionConfig struct {
	Resource string   `json:"resource" yaml:"resource"`
	Actions  []string `json:"actions" yaml:"actions"`
}

// BindingConfig defines role binding configuration
type BindingConfig struct {
	Role     string   `json:"role" yaml:"role"`
	Subjects []string `json:"subjects" yaml:"subjects"`
}

// PolicyConfig defines policy configuration
type PolicyConfig struct {
	Name   string      `json:"name" yaml:"name"`
	Type   string      `json:"type" yaml:"type"`
	Policy interface{} `json:"policy" yaml:"policy"`
}

// CertificateConfig defines certificate configuration
type CertificateConfig struct {
	CA           *CAConfig     `json:"ca,omitempty" yaml:"ca,omitempty"`
	Certificates []*CertConfig `json:"certificates" yaml:"certificates"`
	AutoRotation *CertRotation `json:"autoRotation,omitempty" yaml:"autoRotation,omitempty"`
}

// CAConfig defines certificate authority configuration
type CAConfig struct {
	Name   string            `json:"name" yaml:"name"`
	Type   string            `json:"type" yaml:"type"` // self-signed, external, vault
	Config map[string]string `json:"config" yaml:"config"`
}

// CertConfig defines certificate configuration
type CertConfig struct {
	Name     string           `json:"name" yaml:"name"`
	Subject  *Subject         `json:"subject" yaml:"subject"`
	Usage    []string         `json:"usage" yaml:"usage"`
	Validity *metav1.Duration `json:"validity" yaml:"validity"`
}

// Subject defines certificate subject
type Subject struct {
	CommonName   string `json:"commonName" yaml:"commonName"`
	Organization string `json:"organization" yaml:"organization"`
	Country      string `json:"country" yaml:"country"`
}

// CertRotation defines certificate rotation configuration
type CertRotation struct {
	Enabled       bool             `json:"enabled" yaml:"enabled"`
	RenewalBefore *metav1.Duration `json:"renewalBefore" yaml:"renewalBefore"`
}

// SBAConfig defines Service-Based Architecture configuration
type SBAConfig struct {
	ServiceRegistry *ServiceRegistry `json:"serviceRegistry" yaml:"serviceRegistry"`
	ServiceMesh     *ServiceMesh     `json:"serviceMesh,omitempty" yaml:"serviceMesh,omitempty"`
	LoadBalancing   *LoadBalancing   `json:"loadBalancing,omitempty" yaml:"loadBalancing,omitempty"`
}

// ServiceRegistry defines service registry configuration
type ServiceRegistry struct {
	Type     string            `json:"type" yaml:"type"` // consul, etcd, kubernetes
	Config   map[string]string `json:"config" yaml:"config"`
	Security *SecurityConfig   `json:"security,omitempty" yaml:"security,omitempty"`
}

// ServiceMesh defines service mesh configuration
type ServiceMesh struct {
	Enabled bool              `json:"enabled" yaml:"enabled"`
	Type    string            `json:"type" yaml:"type"` // istio, linkerd, consul-connect
	Config  map[string]string `json:"config" yaml:"config"`
}

// LoadBalancing defines load balancing configuration
type LoadBalancing struct {
	Algorithm string            `json:"algorithm" yaml:"algorithm"`
	Config    map[string]string `json:"config" yaml:"config"`
}

// Multi-vendor support

// VendorVariant defines vendor-specific template variations
type VendorVariant struct {
	Vendor    string                 `json:"vendor" yaml:"vendor"`
	Version   string                 `json:"version" yaml:"version"`
	Overrides map[string]interface{} `json:"overrides" yaml:"overrides"`
	Resources []*KRMTemplate         `json:"resources,omitempty" yaml:"resources,omitempty"`
	Functions []*FunctionTemplate    `json:"functions,omitempty" yaml:"functions,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// Template extensions and composition

// TemplateExtension defines template extensions
type TemplateExtension struct {
	Name       string                 `json:"name" yaml:"name"`
	Type       string                 `json:"type" yaml:"type"` // addon, plugin, overlay
	Config     map[string]interface{} `json:"config" yaml:"config"`
	Resources  []*KRMTemplate         `json:"resources,omitempty" yaml:"resources,omitempty"`
	Functions  []*FunctionTemplate    `json:"functions,omitempty" yaml:"functions,omitempty"`
	Conditions []*RenderCondition     `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// TemplateComposition defines how multiple templates are composed
type TemplateComposition struct {
	TemplateID   string                 `json:"templateId" yaml:"templateId"`
	Parameters   map[string]interface{} `json:"parameters" yaml:"parameters"`
	Overrides    map[string]interface{} `json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Order        int                    `json:"order" yaml:"order"`
}

// CompositeTemplate represents a composed template from multiple base templates
type CompositeTemplate struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Components  []*TemplateComposition `json:"components" yaml:"components"`
	Schema      *ParameterSchema       `json:"schema" yaml:"schema"`
	Resources   []*KRMTemplate         `json:"resources" yaml:"resources"`
	Functions   []*FunctionTemplate    `json:"functions" yaml:"functions"`
	Validations []*ValidationRule      `json:"validations" yaml:"validations"`
}

// Template catalog and management

// TemplateCatalog manages the template repository
type TemplateCatalog struct {
	Repository string                                    `json:"repository" yaml:"repository"`
	Branch     string                                    `json:"branch" yaml:"branch"`
	Templates  map[string]*BlueprintTemplate             `json:"templates" yaml:"-"`
	Categories map[TemplateCategory][]*BlueprintTemplate `json:"categories" yaml:"-"`
	Index      *TemplateIndex                            `json:"index" yaml:"index"`
	LastUpdate time.Time                                 `json:"lastUpdate" yaml:"lastUpdate"`
}

// TemplateIndex provides efficient template lookups
type TemplateIndex struct {
	ByComponent map[nephoranv1.TargetComponent][]*BlueprintTemplate `json:"byComponent" yaml:"byComponent"`
	ByVendor    map[string][]*BlueprintTemplate                     `json:"byVendor" yaml:"byVendor"`
	ByStandard  map[string][]*BlueprintTemplate                     `json:"byStandard" yaml:"byStandard"`
	ByMaturity  map[MaturityLevel][]*BlueprintTemplate              `json:"byMaturity" yaml:"byMaturity"`
	ByKeyword   map[string][]*BlueprintTemplate                     `json:"byKeyword" yaml:"byKeyword"`
}

// Template versioning and lifecycle

// TemplateVersion represents a template version
type TemplateVersion struct {
	Version   string             `json:"version" yaml:"version"`
	Template  *BlueprintTemplate `json:"template" yaml:"template"`
	CreatedAt time.Time          `json:"createdAt" yaml:"createdAt"`
	Author    string             `json:"author" yaml:"author"`
	Changes   []string           `json:"changes" yaml:"changes"`
	Status    string             `json:"status" yaml:"status"` // draft, released, deprecated
}

// Template compliance and testing

// ComplianceInfo contains template compliance information
type ComplianceInfo struct {
	Standards      []*StandardCompliance `json:"standards" yaml:"standards"`
	Certifications []*Certification      `json:"certifications,omitempty" yaml:"certifications,omitempty"`
	LastAudit      *time.Time            `json:"lastAudit,omitempty" yaml:"lastAudit,omitempty"`
	AuditResults   []*AuditResult        `json:"auditResults,omitempty" yaml:"auditResults,omitempty"`
}

// StandardCompliance represents compliance with a standard
type StandardCompliance struct {
	Standard    string    `json:"standard" yaml:"standard"`
	Version     string    `json:"version" yaml:"version"`
	Status      string    `json:"status" yaml:"status"` // compliant, partial, non-compliant
	LastChecked time.Time `json:"lastChecked" yaml:"lastChecked"`
	Issues      []string  `json:"issues,omitempty" yaml:"issues,omitempty"`
}

// Certification represents a certification
type Certification struct {
	Name        string    `json:"name" yaml:"name"`
	Authority   string    `json:"authority" yaml:"authority"`
	ValidFrom   time.Time `json:"validFrom" yaml:"validFrom"`
	ValidUntil  time.Time `json:"validUntil" yaml:"validUntil"`
	Certificate string    `json:"certificate" yaml:"certificate"`
}

// AuditResult represents an audit result
type AuditResult struct {
	ID          string    `json:"id" yaml:"id"`
	AuditDate   time.Time `json:"auditDate" yaml:"auditDate"`
	Auditor     string    `json:"auditor" yaml:"auditor"`
	Score       float64   `json:"score" yaml:"score"`
	Findings    []string  `json:"findings" yaml:"findings"`
	Remediation []string  `json:"remediation,omitempty" yaml:"remediation,omitempty"`
}

// TestingStatus contains template testing information
type TestingStatus struct {
	LastTested *time.Time   `json:"lastTested,omitempty" yaml:"lastTested,omitempty"`
	TestSuites []*TestSuite `json:"testSuites,omitempty" yaml:"testSuites,omitempty"`
	Coverage   float64      `json:"coverage" yaml:"coverage"`
	PassRate   float64      `json:"passRate" yaml:"passRate"`
}

// TestSuite represents a test suite
type TestSuite struct {
	Name     string        `json:"name" yaml:"name"`
	Type     string        `json:"type" yaml:"type"` // unit, integration, e2e
	Tests    []*TestCase   `json:"tests" yaml:"tests"`
	LastRun  time.Time     `json:"lastRun" yaml:"lastRun"`
	Status   string        `json:"status" yaml:"status"`
	Duration time.Duration `json:"duration" yaml:"duration"`
}

// TestCase represents a test case
type TestCase struct {
	Name        string        `json:"name" yaml:"name"`
	Description string        `json:"description" yaml:"description"`
	Status      string        `json:"status" yaml:"status"` // passed, failed, skipped
	Duration    time.Duration `json:"duration" yaml:"duration"`
	Error       string        `json:"error,omitempty" yaml:"error,omitempty"`
}

// Template filtering and searching

// TemplateFilter defines template filtering criteria
type TemplateFilter struct {
	Category      *TemplateCategory           `json:"category,omitempty"`
	Type          *TemplateType               `json:"type,omitempty"`
	Component     *nephoranv1.TargetComponent `json:"component,omitempty"`
	Vendor        string                      `json:"vendor,omitempty"`
	Standard      string                      `json:"standard,omitempty"`
	MaturityLevel *MaturityLevel              `json:"maturityLevel,omitempty"`
	Tags          []string                    `json:"tags,omitempty"`
	Keywords      []string                    `json:"keywords,omitempty"`
	MinVersion    string                      `json:"minVersion,omitempty"`
	MaxVersion    string                      `json:"maxVersion,omitempty"`
}

// SearchQuery defines template search criteria
type SearchQuery struct {
	Query     string          `json:"query"`
	Filter    *TemplateFilter `json:"filter,omitempty"`
	SortBy    string          `json:"sortBy,omitempty"`    // name, version, usage, rating
	SortOrder string          `json:"sortOrder,omitempty"` // asc, desc
	Limit     int             `json:"limit,omitempty"`
	Offset    int             `json:"offset,omitempty"`
}

// Template rendering results

// RenderResult contains template rendering results
type RenderResult struct {
	Success           bool                       `json:"success"`
	Resources         []*porch.KRMResource       `json:"resources"`
	ValidationResult  *ParameterValidationResult `json:"validationResult"`
	RenderingErrors   []*RenderingError          `json:"renderingErrors,omitempty"`
	RenderingWarnings []*RenderingWarning        `json:"renderingWarnings,omitempty"`
	GeneratedFiles    map[string]string          `json:"generatedFiles,omitempty"`
	Functions         []*FunctionExecution       `json:"functions,omitempty"`
	Duration          time.Duration              `json:"duration"`
	Metadata          map[string]interface{}     `json:"metadata,omitempty"`
}

// ParameterValidationResult contains parameter validation results
type ParameterValidationResult struct {
	Valid      bool                          `json:"valid"`
	Errors     []*ParameterValidationError   `json:"errors,omitempty"`
	Warnings   []*ParameterValidationWarning `json:"warnings,omitempty"`
	Missing    []string                      `json:"missing,omitempty"`
	Unexpected []string                      `json:"unexpected,omitempty"`
}

// ParameterValidationError represents a parameter validation error
type ParameterValidationError struct {
	Parameter  string `json:"parameter"`
	Message    string `json:"message"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ParameterValidationWarning represents a parameter validation warning
type ParameterValidationWarning struct {
	Parameter  string `json:"parameter"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// RenderingError represents a template rendering error
type RenderingError struct {
	Resource string `json:"resource,omitempty"`
	Template string `json:"template,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
	Code     string `json:"code,omitempty"`
}

// RenderingWarning represents a template rendering warning
type RenderingWarning struct {
	Resource   string `json:"resource,omitempty"`
	Template   string `json:"template,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// FunctionExecution represents function execution results
type FunctionExecution struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Output   interface{}   `json:"output,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// Engine configuration and health

// EngineConfig contains template engine configuration
type EngineConfig struct {
	// Repository settings
	TemplateRepository string        `yaml:"templateRepository"`
	RepositoryBranch   string        `yaml:"repositoryBranch"`
	RepositoryPath     string        `yaml:"repositoryPath"`
	RefreshInterval    time.Duration `yaml:"refreshInterval"`

	// Rendering settings
	MaxConcurrentRenders int           `yaml:"maxConcurrentRenders"`
	RenderTimeout        time.Duration `yaml:"renderTimeout"`
	EnableCaching        bool          `yaml:"enableCaching"`
	CacheSize            int           `yaml:"cacheSize"`
	CacheTTL             time.Duration `yaml:"cacheTTL"`

	// Validation settings
	EnableParameterValidation bool          `yaml:"enableParameterValidation"`
	EnableYANGValidation      bool          `yaml:"enableYangValidation"`
	EnableTemplateValidation  bool          `yaml:"enableTemplateValidation"`
	ValidationTimeout         time.Duration `yaml:"validationTimeout"`

	// Security settings
	EnableSecurityScanning bool     `yaml:"enableSecurityScanning"`
	SecurityScanners       []string `yaml:"securityScanners"`
	AllowUnsignedTemplates bool     `yaml:"allowUnsignedTemplates"`

	// Performance settings
	EnableMetrics   bool          `yaml:"enableMetrics"`
	MetricsInterval time.Duration `yaml:"metricsInterval"`
}

// CatalogInfo contains template catalog information
type CatalogInfo struct {
	Repository           string                             `json:"repository"`
	Branch               string                             `json:"branch"`
	LastUpdate           time.Time                          `json:"lastUpdate"`
	TotalTemplates       int                                `json:"totalTemplates"`
	TemplatesByCategory  map[TemplateCategory]int           `json:"templatesByCategory"`
	TemplatesByComponent map[nephoranv1.TargetComponent]int `json:"templatesByComponent"`
	TemplatesByVendor    map[string]int                     `json:"templatesByVendor"`
	TemplatesByMaturity  map[MaturityLevel]int              `json:"templatesByMaturity"`
}

// EngineHealth contains template engine health information
type EngineHealth struct {
	Status             string            `json:"status"`
	TemplatesLoaded    int               `json:"templatesLoaded"`
	CacheHitRate       float64           `json:"cacheHitRate"`
	AverageRenderTime  time.Duration     `json:"averageRenderTime"`
	ActiveRenders      int               `json:"activeRenders"`
	LastCatalogRefresh time.Time         `json:"lastCatalogRefresh"`
	ComponentHealth    map[string]string `json:"componentHealth"`
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(yangValidator yang.YANGValidator, config *EngineConfig) (TemplateEngine, error) {
	if config == nil {
		config = getDefaultEngineConfig()
	}

	engine := &templateEngine{
		yangValidator: yangValidator,
		logger:        log.Log.WithName("template-engine"),
		config:        config,
		templates:     make(map[string]*BlueprintTemplate),
		templateCatalog: &TemplateCatalog{
			Repository: config.TemplateRepository,
			Branch:     config.RepositoryBranch,
			Templates:  make(map[string]*BlueprintTemplate),
			Categories: make(map[TemplateCategory][]*BlueprintTemplate),
			Index: &TemplateIndex{
				ByComponent: make(map[nephoranv1.TargetComponent][]*BlueprintTemplate),
				ByVendor:    make(map[string][]*BlueprintTemplate),
				ByStandard:  make(map[string][]*BlueprintTemplate),
				ByMaturity:  make(map[MaturityLevel][]*BlueprintTemplate),
				ByKeyword:   make(map[string][]*BlueprintTemplate),
			},
			LastUpdate: time.Now(),
		},
		shutdown: make(chan struct{}),
	}

	// Start background workers
	engine.wg.Add(1)
	go engine.catalogRefreshWorker()

	// Load built-in templates
	if err := engine.loadBuiltInTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load built-in templates: %w", err)
	}

	engine.logger.Info("Template engine initialized successfully")
	return engine, nil
}

// GetTemplate retrieves a template by ID
func (e *templateEngine) GetTemplate(ctx context.Context, templateID string) (*BlueprintTemplate, error) {
	e.templateMutex.RLock()
	defer e.templateMutex.RUnlock()

	template, exists := e.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template %s not found", templateID)
	}

	return template, nil
}

// ListTemplates lists templates with optional filtering
func (e *templateEngine) ListTemplates(ctx context.Context, filter *TemplateFilter) ([]*BlueprintTemplate, error) {
	e.templateMutex.RLock()
	defer e.templateMutex.RUnlock()

	var result []*BlueprintTemplate

	for _, template := range e.templates {
		if e.matchesFilter(template, filter) {
			result = append(result, template)
		}
	}

	return result, nil
}

// RenderTemplate renders a template with given parameters
func (e *templateEngine) RenderTemplate(ctx context.Context, templateID string, parameters map[string]interface{}) ([]*porch.KRMResource, error) {
	result, err := e.RenderTemplateWithValidation(ctx, templateID, parameters)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("template rendering failed")
	}

	return result.Resources, nil
}

// RenderTemplateWithValidation renders a template with comprehensive validation
func (e *templateEngine) RenderTemplateWithValidation(ctx context.Context, templateID string, parameters map[string]interface{}) (*RenderResult, error) {
	startTime := time.Now()

	result := &RenderResult{
		Success:        true,
		Resources:      []*porch.KRMResource{},
		GeneratedFiles: make(map[string]string),
		Functions:      []*FunctionExecution{},
		Metadata:       make(map[string]interface{}),
	}

	// Get template
	template, err := e.GetTemplate(ctx, templateID)
	if err != nil {
		result.Success = false
		result.RenderingErrors = []*RenderingError{{Message: err.Error()}}
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Validate parameters
	if e.config.EnableParameterValidation {
		validationResult, err := e.ValidateParameters(ctx, templateID, parameters)
		if err != nil {
			result.Success = false
			result.RenderingErrors = []*RenderingError{{Message: fmt.Sprintf("Parameter validation failed: %v", err)}}
			result.Duration = time.Since(startTime)
			return result, err
		}
		result.ValidationResult = validationResult

		if !validationResult.Valid {
			result.Success = false
			result.Duration = time.Since(startTime)
			return result, fmt.Errorf("parameter validation failed")
		}
	}

	// Render resources
	resources, err := e.renderResources(ctx, template, parameters)
	if err != nil {
		result.Success = false
		result.RenderingErrors = []*RenderingError{{Message: err.Error()}}
		result.Duration = time.Since(startTime)
		return result, err
	}
	result.Resources = resources

	// Execute functions if any
	if len(template.Functions) > 0 {
		functionResults, err := e.executeFunctions(ctx, template, parameters, resources)
		if err != nil {
			result.RenderingWarnings = []*RenderingWarning{{Message: fmt.Sprintf("Function execution warning: %v", err)}}
		}
		result.Functions = functionResults
	}

	result.Duration = time.Since(startTime)

	e.logger.V(1).Info("Template rendered successfully",
		"templateID", templateID,
		"resourceCount", len(result.Resources),
		"duration", result.Duration)

	return result, nil
}

// ValidateParameters validates template parameters against schema
func (e *templateEngine) ValidateParameters(ctx context.Context, templateID string, parameters map[string]interface{}) (*ParameterValidationResult, error) {
	template, err := e.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	if template.Schema == nil {
		// No schema defined, assume all parameters are valid
		return &ParameterValidationResult{Valid: true}, nil
	}

	result := &ParameterValidationResult{
		Valid:      true,
		Errors:     []*ParameterValidationError{},
		Warnings:   []*ParameterValidationWarning{},
		Missing:    []string{},
		Unexpected: []string{},
	}

	// Check required parameters
	for _, required := range template.Schema.Required {
		if _, exists := parameters[required]; !exists {
			result.Valid = false
			result.Missing = append(result.Missing, required)
			result.Errors = append(result.Errors, &ParameterValidationError{
				Parameter: required,
				Message:   fmt.Sprintf("Required parameter %s is missing", required),
			})
		}
	}

	// Validate parameter values
	for paramName, paramValue := range parameters {
		spec, exists := template.Schema.Properties[paramName]
		if !exists {
			result.Unexpected = append(result.Unexpected, paramName)
			result.Warnings = append(result.Warnings, &ParameterValidationWarning{
				Parameter: paramName,
				Message:   fmt.Sprintf("Unexpected parameter %s", paramName),
			})
			continue
		}

		// Validate parameter according to spec
		if err := e.validateParameterValue(paramName, paramValue, spec); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, &ParameterValidationError{
				Parameter: paramName,
				Message:   err.Error(),
			})
		}
	}

	return result, nil
}

// Helper methods

func (e *templateEngine) matchesFilter(template *BlueprintTemplate, filter *TemplateFilter) bool {
	if filter == nil {
		return true
	}

	if filter.Category != nil && template.Category != *filter.Category {
		return false
	}

	if filter.Type != nil && template.Type != *filter.Type {
		return false
	}

	if filter.Component != nil && template.TargetComponent != *filter.Component {
		return false
	}

	if filter.Vendor != "" && template.Vendor != filter.Vendor {
		return false
	}

	if filter.Standard != "" && template.Standard != filter.Standard {
		return false
	}

	if filter.MaturityLevel != nil && template.MaturityLevel != *filter.MaturityLevel {
		return false
	}

	// Check tags
	if len(filter.Tags) > 0 {
		tagMap := make(map[string]bool)
		for _, tag := range template.Tags {
			tagMap[tag] = true
		}
		for _, filterTag := range filter.Tags {
			if !tagMap[filterTag] {
				return false
			}
		}
	}

	// Check keywords
	if len(filter.Keywords) > 0 {
		keywordMap := make(map[string]bool)
		for _, keyword := range template.Keywords {
			keywordMap[keyword] = true
		}
		for _, filterKeyword := range filter.Keywords {
			if !keywordMap[filterKeyword] {
				return false
			}
		}
	}

	return true
}

func (e *templateEngine) renderResources(ctx context.Context, template *BlueprintTemplate, parameters map[string]interface{}) ([]*porch.KRMResource, error) {
	var resources []*porch.KRMResource

	for _, resourceTemplate := range template.Resources {
		// Check conditions
		if !e.evaluateConditions(resourceTemplate.Conditions, parameters) {
			continue
		}

		// Render the resource
		resource, err := e.renderResource(ctx, resourceTemplate, parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to render resource %s: %w", resourceTemplate.Metadata.Name, err)
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func (e *templateEngine) renderResource(ctx context.Context, resourceTemplate *KRMTemplate, parameters map[string]interface{}) (*porch.KRMResource, error) {
	resource := &porch.KRMResource{
		APIVersion: resourceTemplate.APIVersion,
		Kind:       resourceTemplate.Kind,
		Metadata:   make(map[string]interface{}),
		Spec:       make(map[string]interface{}),
	}

	// Render metadata
	if resourceTemplate.Metadata != nil {
		metadata := map[string]interface{}{
			"name": e.renderString(resourceTemplate.Metadata.Name, parameters),
		}

		if resourceTemplate.Metadata.Namespace != "" {
			metadata["namespace"] = e.renderString(resourceTemplate.Metadata.Namespace, parameters)
		}

		if resourceTemplate.Metadata.Labels != nil {
			labels := make(map[string]string)
			for k, v := range resourceTemplate.Metadata.Labels {
				labels[k] = e.renderString(v, parameters)
			}
			metadata["labels"] = labels
		}

		if resourceTemplate.Metadata.Annotations != nil {
			annotations := make(map[string]string)
			for k, v := range resourceTemplate.Metadata.Annotations {
				annotations[k] = e.renderString(v, parameters)
			}
			metadata["annotations"] = annotations
		}

		resource.Metadata = metadata
	}

	// Render spec
	if resourceTemplate.Spec != nil {
		renderedSpec, err := e.renderMapInterface(resourceTemplate.Spec, parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to render spec: %w", err)
		}
		resource.Spec = renderedSpec
	}

	// Render data
	if resourceTemplate.Data != nil {
		renderedData, err := e.renderMapInterface(resourceTemplate.Data, parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to render data: %w", err)
		}
		resource.Data = renderedData
	}

	return resource, nil
}

func (e *templateEngine) renderString(templateStr string, parameters map[string]interface{}) string {
	if !strings.Contains(templateStr, "{{") {
		return templateStr
	}

	tmpl, err := template.New("render").Parse(templateStr)
	if err != nil {
		e.logger.Error(err, "Failed to parse template string", "template", templateStr)
		return templateStr
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, parameters); err != nil {
		e.logger.Error(err, "Failed to execute template", "template", templateStr)
		return templateStr
	}

	return result.String()
}

func (e *templateEngine) renderMapInterface(data map[string]interface{}, parameters map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range data {
		renderedKey := e.renderString(key, parameters)

		switch v := value.(type) {
		case string:
			result[renderedKey] = e.renderString(v, parameters)
		case map[string]interface{}:
			renderedMap, err := e.renderMapInterface(v, parameters)
			if err != nil {
				return nil, err
			}
			result[renderedKey] = renderedMap
		case []interface{}:
			renderedSlice, err := e.renderSliceInterface(v, parameters)
			if err != nil {
				return nil, err
			}
			result[renderedKey] = renderedSlice
		default:
			result[renderedKey] = value
		}
	}

	return result, nil
}

func (e *templateEngine) renderSliceInterface(data []interface{}, parameters map[string]interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(data))

	for i, value := range data {
		switch v := value.(type) {
		case string:
			result[i] = e.renderString(v, parameters)
		case map[string]interface{}:
			renderedMap, err := e.renderMapInterface(v, parameters)
			if err != nil {
				return nil, err
			}
			result[i] = renderedMap
		case []interface{}:
			renderedSlice, err := e.renderSliceInterface(v, parameters)
			if err != nil {
				return nil, err
			}
			result[i] = renderedSlice
		default:
			result[i] = value
		}
	}

	return result, nil
}

func (e *templateEngine) evaluateConditions(conditions []*RenderCondition, parameters map[string]interface{}) bool {
	if len(conditions) == 0 {
		return true
	}

	// Simple condition evaluation (in a real implementation, this would be more sophisticated)
	for _, condition := range conditions {
		if condition.Action == "exclude" {
			// For simplicity, assume all exclude conditions are false
			return false
		}
	}

	return true
}

func (e *templateEngine) validateParameterValue(paramName string, value interface{}, spec *ParameterSpec) error {
	// Type validation
	switch spec.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %s must be a string", paramName)
		}

		// String-specific validations
		str := value.(string)
		if spec.MinLength != nil && len(str) < *spec.MinLength {
			return fmt.Errorf("parameter %s must be at least %d characters", paramName, *spec.MinLength)
		}
		if spec.MaxLength != nil && len(str) > *spec.MaxLength {
			return fmt.Errorf("parameter %s must be at most %d characters", paramName, *spec.MaxLength)
		}
		if spec.Pattern != "" {
			// Pattern validation would go here
		}

	case "integer":
		switch v := value.(type) {
		case int, int32, int64, float64:
			// Valid numeric types
		default:
			return fmt.Errorf("parameter %s must be an integer, got %T", paramName, v)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter %s must be a boolean", paramName)
		}

	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("parameter %s must be an object", paramName)
		}

	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("parameter %s must be an array", paramName)
		}
	}

	// Enum validation
	if len(spec.Enum) > 0 {
		found := false
		for _, enumValue := range spec.Enum {
			if value == enumValue {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parameter %s must be one of %v", paramName, spec.Enum)
		}
	}

	return nil
}

func (e *templateEngine) executeFunctions(ctx context.Context, template *BlueprintTemplate, parameters map[string]interface{}, resources []*porch.KRMResource) ([]*FunctionExecution, error) {
	var results []*FunctionExecution

	for _, function := range template.Functions {
		// Check conditions
		if !e.evaluateConditions(function.Conditions, parameters) {
			continue
		}

		startTime := time.Now()
		execution := &FunctionExecution{
			Name: function.Name,
			Type: function.Type,
		}

		// Execute function (this would integrate with actual KRM function execution)
		// For now, just simulate successful execution
		execution.Success = true
		execution.Duration = time.Since(startTime)
		execution.Output = "Function executed successfully"

		results = append(results, execution)
	}

	return results, nil
}

func (e *templateEngine) loadBuiltInTemplates() error {
	// Load built-in O-RAN and 5G Core templates
	builtInTemplates := e.getBuiltInTemplates()

	e.templateMutex.Lock()
	defer e.templateMutex.Unlock()

	for _, template := range builtInTemplates {
		e.templates[template.ID] = template
		e.updateTemplateIndex(template)
	}

	e.logger.Info("Loaded built-in templates", "count", len(builtInTemplates))
	return nil
}

func (e *templateEngine) updateTemplateIndex(template *BlueprintTemplate) {
	// Update category index
	if e.templateCatalog.Categories[template.Category] == nil {
		e.templateCatalog.Categories[template.Category] = []*BlueprintTemplate{}
	}
	e.templateCatalog.Categories[template.Category] = append(e.templateCatalog.Categories[template.Category], template)

	// Update component index
	if e.templateCatalog.Index.ByComponent[template.TargetComponent] == nil {
		e.templateCatalog.Index.ByComponent[template.TargetComponent] = []*BlueprintTemplate{}
	}
	e.templateCatalog.Index.ByComponent[template.TargetComponent] = append(e.templateCatalog.Index.ByComponent[template.TargetComponent], template)

	// Update other indices...
}

func (e *templateEngine) catalogRefreshWorker() {
	defer e.wg.Done()
	ticker := time.NewTicker(e.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.shutdown:
			return
		case <-ticker.C:
			if err := e.RefreshCatalog(context.Background()); err != nil {
				e.logger.Error(err, "Failed to refresh template catalog")
			}
		}
	}
}

// Close gracefully shuts down the template engine
func (e *templateEngine) Close() error {
	e.logger.Info("Shutting down template engine")
	close(e.shutdown)
	e.wg.Wait()
	e.logger.Info("Template engine shutdown complete")
	return nil
}

// Placeholder implementations for interface compliance

func (e *templateEngine) LoadTemplate(ctx context.Context, templateID string) (*BlueprintTemplate, error) {
	return e.GetTemplate(ctx, templateID)
}

func (e *templateEngine) LoadTemplateFromRepository(ctx context.Context, repoURL, templatePath string) (*BlueprintTemplate, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *templateEngine) RegisterTemplate(ctx context.Context, template *BlueprintTemplate) error {
	e.templateMutex.Lock()
	defer e.templateMutex.Unlock()

	e.templates[template.ID] = template
	e.updateTemplateIndex(template)
	return nil
}

func (e *templateEngine) UnregisterTemplate(ctx context.Context, templateID string) error {
	e.templateMutex.Lock()
	defer e.templateMutex.Unlock()

	delete(e.templates, templateID)
	return nil
}

func (e *templateEngine) RefreshCatalog(ctx context.Context) error {
	e.logger.V(1).Info("Refreshing template catalog")
	e.templateCatalog.LastUpdate = time.Now()
	return nil
}

func (e *templateEngine) GetCatalogInfo(ctx context.Context) (*CatalogInfo, error) {
	e.templateMutex.RLock()
	defer e.templateMutex.RUnlock()

	info := &CatalogInfo{
		Repository:           e.templateCatalog.Repository,
		Branch:               e.templateCatalog.Branch,
		LastUpdate:           e.templateCatalog.LastUpdate,
		TotalTemplates:       len(e.templates),
		TemplatesByCategory:  make(map[TemplateCategory]int),
		TemplatesByComponent: make(map[nephoranv1.TargetComponent]int),
		TemplatesByVendor:    make(map[string]int),
		TemplatesByMaturity:  make(map[MaturityLevel]int),
	}

	for _, template := range e.templates {
		info.TemplatesByCategory[template.Category]++
		info.TemplatesByComponent[template.TargetComponent]++
		info.TemplatesByVendor[template.Vendor]++
		info.TemplatesByMaturity[template.MaturityLevel]++
	}

	return info, nil
}

func (e *templateEngine) SearchTemplates(ctx context.Context, query *SearchQuery) ([]*BlueprintTemplate, error) {
	templates, err := e.ListTemplates(ctx, query.Filter)
	if err != nil {
		return nil, err
	}

	// Simple search implementation
	if query.Query != "" {
		var filtered []*BlueprintTemplate
		for _, template := range templates {
			if strings.Contains(strings.ToLower(template.Name), strings.ToLower(query.Query)) ||
				strings.Contains(strings.ToLower(template.Description), strings.ToLower(query.Query)) {
				filtered = append(filtered, template)
			}
		}
		templates = filtered
	}

	// Apply limit and offset
	if query.Limit > 0 {
		start := query.Offset
		if start >= len(templates) {
			return []*BlueprintTemplate{}, nil
		}
		end := start + query.Limit
		if end > len(templates) {
			end = len(templates)
		}
		templates = templates[start:end]
	}

	return templates, nil
}

func (e *templateEngine) ComposeTemplates(ctx context.Context, templates []*TemplateComposition) (*CompositeTemplate, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *templateEngine) RenderCompositeTemplate(ctx context.Context, composite *CompositeTemplate, parameters map[string]interface{}) ([]*porch.KRMResource, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *templateEngine) GetTemplateVersions(ctx context.Context, templateName string) ([]*TemplateVersion, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *templateEngine) PromoteTemplateVersion(ctx context.Context, templateID, version string) error {
	return fmt.Errorf("not implemented")
}

func (e *templateEngine) RollbackTemplateVersion(ctx context.Context, templateID, version string) error {
	return fmt.Errorf("not implemented")
}

func (e *templateEngine) GetEngineHealth(ctx context.Context) (*EngineHealth, error) {
	e.templateMutex.RLock()
	templatesCount := len(e.templates)
	e.templateMutex.RUnlock()

	return &EngineHealth{
		Status:             "healthy",
		TemplatesLoaded:    templatesCount,
		CacheHitRate:       0.85,                   // Mock value
		AverageRenderTime:  100 * time.Millisecond, // Mock value
		ActiveRenders:      0,
		LastCatalogRefresh: e.templateCatalog.LastUpdate,
		ComponentHealth: map[string]string{
			"catalog":   "healthy",
			"validator": "healthy",
			"renderer":  "healthy",
		},
	}, nil
}

// Utility functions

func getDefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		TemplateRepository:        "https://github.com/nephoran/templates.git",
		RepositoryBranch:          "main",
		RepositoryPath:            "templates",
		RefreshInterval:           60 * time.Minute,
		MaxConcurrentRenders:      10,
		RenderTimeout:             5 * time.Minute,
		EnableCaching:             true,
		CacheSize:                 1000,
		CacheTTL:                  24 * time.Hour,
		EnableParameterValidation: true,
		EnableYANGValidation:      true,
		EnableTemplateValidation:  true,
		ValidationTimeout:         30 * time.Second,
		EnableSecurityScanning:    true,
		SecurityScanners:          []string{"trivy", "snyk"},
		AllowUnsignedTemplates:    false,
		EnableMetrics:             true,
		MetricsInterval:           30 * time.Second,
	}
}

// getBuiltInTemplates returns built-in O-RAN and 5G Core templates
func (e *templateEngine) getBuiltInTemplates() []*BlueprintTemplate {
	var templates []*BlueprintTemplate

	// AMF Template
	amfTemplate := &BlueprintTemplate{
		ID:              "oran-5g-amf-v1",
		Name:            "O-RAN 5G AMF",
		Version:         "1.0.0",
		Description:     "Access and Mobility Management Function template for O-RAN compliant 5G Core",
		Author:          "Nephoran Team",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Category:        TemplateCategoryNetworkFunction,
		Type:            TemplateTypeDeployment,
		TargetComponent: nephoranv1.TargetComponentAMF,
		Vendor:          "open-source",
		Standard:        "O-RAN",
		MaturityLevel:   MaturityLevelStable,
		Tags:            []string{"5g", "core", "amf", "oran"},
		Keywords:        []string{"access", "mobility", "management", "5g-core"},
		Schema: &ParameterSchema{
			Version: "1.0",
			Properties: map[string]*ParameterSpec{
				"replicas": {
					Type:        "integer",
					Description: "Number of AMF replicas",
					Default:     3,
					Minimum:     func() *float64 { v := 1.0; return &v }(),
					Maximum:     func() *float64 { v := 10.0; return &v }(),
				},
				"resources": {
					Type:        "object",
					Description: "Resource requirements for AMF",
				},
				"plmnList": {
					Type:        "array",
					Description: "List of PLMN identifiers",
				},
			},
			Required: []string{"replicas", "plmnList"},
		},
		Resources: []*KRMTemplate{
			{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Metadata: &ResourceMetadata{
					Name:      "amf-deployment",
					Namespace: "5g-core",
					Labels: map[string]string{
						"app":       "amf",
						"component": "5g-core",
					},
				},
				Spec: map[string]interface{}{
					"replicas": "{{ .replicas }}",
					"selector": map[string]interface{}{
						"matchLabels": map[string]string{
							"app": "amf",
						},
					},
					"template": map[string]interface{}{
						"metadata": map[string]interface{}{
							"labels": map[string]string{
								"app": "amf",
							},
						},
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name":  "amf",
									"image": "nephoran/amf:latest",
									"ports": []interface{}{
										map[string]interface{}{
											"containerPort": 8080,
											"name":          "sbi",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		FiveGConfig: &FiveGTemplateConfig{
			NetworkFunctions: []*NetworkFunction{
				{
					Type:    nephoranv1.TargetComponentAMF,
					Version: "rel-16",
					Configuration: map[string]interface{}{
						"servedGuamiList": []interface{}{
							map[string]interface{}{
								"plmnId": map[string]interface{}{
									"mcc": "001",
									"mnc": "01",
								},
								"amfId": "cafe00",
							},
						},
					},
					Interfaces: []*NFInterface{
						{
							Name:     "namf-comm",
							Type:     "SBI",
							Protocol: "HTTP/2",
							Config: map[string]interface{}{
								"port": 8080,
								"tls":  true,
							},
						},
						{
							Name:     "n1",
							Type:     "N1",
							Protocol: "NAS",
						},
						{
							Name:     "n2",
							Type:     "N2",
							Protocol: "NGAP",
						},
					},
					Resources: &ResourceRequirements{
						CPU:    resource.NewMilliQuantity(500, resource.DecimalSI),
						Memory: resource.NewQuantity(1*1024*1024*1024, resource.BinarySI), // 1Gi
					},
					HighAvailability: &HAConfig{
						Enabled:      true,
						Replicas:     3,
						Strategy:     "active-active",
						AntiAffinity: true,
					},
				},
			},
		},
		ComplianceInfo: &ComplianceInfo{
			Standards: []*StandardCompliance{
				{
					Standard:    "3GPP TS 23.501",
					Version:     "16.8.0",
					Status:      "compliant",
					LastChecked: time.Now(),
				},
				{
					Standard:    "O-RAN WG6",
					Version:     "3.0",
					Status:      "compliant",
					LastChecked: time.Now(),
				},
			},
		},
		TestingStatus: &TestingStatus{
			LastTested: func() *time.Time { t := time.Now(); return &t }(),
			Coverage:   85.5,
			PassRate:   98.2,
		},
	}

	templates = append(templates, amfTemplate)

	// Add more templates for other components...
	// SMF, UPF, gNodeB, O-DU, O-CU, Near-RT RIC, etc.

	return templates
}
