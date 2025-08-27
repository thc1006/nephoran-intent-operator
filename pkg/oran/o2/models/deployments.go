package models

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// Deployment Template and Management Models following O-RAN.WG6.O2ims-Interface-v01.01

// DeploymentTemplate represents a deployment template following O2 IMS specification
type DeploymentTemplate struct {
	DeploymentTemplateID string `json:"deploymentTemplateId"`
	Name                 string `json:"name"`
	Description          string `json:"description,omitempty"`
	Version              string `json:"version"`
	Provider             string `json:"provider,omitempty"`
	Category             string `json:"category,omitempty"` // VNF, CNF, PNF, NS

	// Template specifications
	TemplateSpec        *TemplateSpecification `json:"templateSpec"`
	RequiredResources   *ResourceRequirements  `json:"requiredResources"`
	SupportedParameters []*TemplateParameter   `json:"supportedParameters,omitempty"`

	// Validation and compatibility
	ValidationRules     []*ValidationRule    `json:"validationRules,omitempty"`
	CompatibilityMatrix []*CompatibilityInfo `json:"compatibilityMatrix,omitempty"`

	// Metadata and lifecycle
	Tags       map[string]string      `json:"tags,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	Status     *TemplateStatus        `json:"status"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	CreatedBy  string                 `json:"createdBy,omitempty"`
	UpdatedBy  string                 `json:"updatedBy,omitempty"`

	// Legacy fields for backward compatibility (mapped to TemplateSpec)
	Type         string                `json:"type,omitempty"`         // Maps to TemplateSpec.Type
	Content      *runtime.RawExtension `json:"content,omitempty"`      // Maps to TemplateSpec.Content
	Author       string                `json:"author,omitempty"`       // Maps to CreatedBy or Provider
	InputSchema  *runtime.RawExtension `json:"inputSchema,omitempty"`  // Maps to TemplateSpec.InputSchema
	OutputSchema *runtime.RawExtension `json:"outputSchema,omitempty"` // Maps to TemplateSpec.OutputSchema
}

// TemplateDependency represents a dependency of a deployment template
type TemplateDependency struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Type        string `json:"type"` // TEMPLATE, RESOURCE, SERVICE
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// TemplateSpecification defines the deployment template specification
type TemplateSpecification struct {
	Type            string                           `json:"type"` // HEAT, HELM, KUBERNETES, TERRAFORM
	Content         *runtime.RawExtension            `json:"content"`
	ContentType     string                           `json:"contentType"` // yaml, json, zip
	MainTemplate    string                           `json:"mainTemplate,omitempty"`
	NestedTemplates map[string]*runtime.RawExtension `json:"nestedTemplates,omitempty"`

	// Deployment configuration
	DeploymentOptions *DeploymentOptions `json:"deploymentOptions,omitempty"`
	Hooks             []*DeploymentHook  `json:"hooks,omitempty"`

	// Resource mappings
	ResourceMappings []*ResourceMapping `json:"resourceMappings,omitempty"`
	NetworkMappings  []*NetworkMapping  `json:"networkMappings,omitempty"`
	StorageMappings  []*StorageMapping  `json:"storageMappings,omitempty"`

	// Dependencies
	Dependencies  []*TemplateDependency `json:"dependencies,omitempty"`
	Prerequisites []*Prerequisite       `json:"prerequisites,omitempty"`
}

// DeploymentHook defines a deployment lifecycle hook
type DeploymentHook struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"` // PRE_CREATE, POST_CREATE, PRE_UPDATE, POST_UPDATE, PRE_DELETE, POST_DELETE
	Script        string                 `json:"script,omitempty"`
	Image         string                 `json:"image,omitempty"`
	Command       []string               `json:"command,omitempty"`
	Args          []string               `json:"args,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
	EnvVars       map[string]string      `json:"envVars,omitempty"`
	FailurePolicy string                 `json:"failurePolicy,omitempty"` // IGNORE, ABORT
	Extensions    map[string]interface{} `json:"extensions,omitempty"`
}

// ResourceMapping defines how template resources map to actual resources
type ResourceMapping struct {
	TemplateName   string                 `json:"templateName"`
	ResourceType   string                 `json:"resourceType"`
	ResourcePoolID string                 `json:"resourcePoolId,omitempty"`
	Constraints    []PlacementConstraint  `json:"constraints,omitempty"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
}

// NetworkMapping defines network mappings for deployments
type NetworkMapping struct {
	TemplateName string                 `json:"templateName"`
	NetworkType  string                 `json:"networkType"` // MANAGEMENT, DATA, CONTROL
	NetworkName  string                 `json:"networkName,omitempty"`
	SubnetCIDR   string                 `json:"subnetCIDR,omitempty"`
	VLANID       *int                   `json:"vlanId,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// StorageMapping defines storage mappings for deployments
type StorageMapping struct {
	TemplateName string                 `json:"templateName"`
	StorageType  string                 `json:"storageType"` // BLOCK, FILE, OBJECT
	StorageClass string                 `json:"storageClass,omitempty"`
	Size         string                 `json:"size,omitempty"`
	AccessModes  []string               `json:"accessModes,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// ScalingPolicy defines scaling policies for deployments
type ScalingPolicy struct {
	MinReplicas       *int32           `json:"minReplicas,omitempty"`
	MaxReplicas       *int32           `json:"maxReplicas,omitempty"`
	TargetUtilization *int32           `json:"targetUtilization,omitempty"`
	ScaleUpPolicy     *ScalePolicy     `json:"scaleUpPolicy,omitempty"`
	ScaleDownPolicy   *ScalePolicy     `json:"scaleDownPolicy,omitempty"`
	Metrics           []ScalingMetric  `json:"metrics,omitempty"`
	Behavior          *ScalingBehavior `json:"behavior,omitempty"`
	CustomTriggers    []ScalingTrigger `json:"customTriggers,omitempty"`
}

// ScalingTrigger represents a custom scaling trigger
type ScalingTrigger struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"` // metric, queue, custom
	Threshold  float64 `json:"threshold"`
	Operator   string  `json:"operator"` // gt, lt, eq, gte, lte
	MetricPath string  `json:"metricPath,omitempty"`
	Query      string  `json:"query,omitempty"`
	Enabled    bool    `json:"enabled"`
}

// ScalePolicy defines scale up/down policies
type ScalePolicy struct {
	Type          string        `json:"type"` // PODS, PERCENT
	Value         int32         `json:"value"`
	PeriodSeconds *int32        `json:"periodSeconds,omitempty"`
	Stabilization time.Duration `json:"stabilizationWindowSeconds,omitempty"`
}

// ScalingMetric defines metrics used for scaling decisions
type ScalingMetric struct {
	Type     string                `json:"type"` // RESOURCE, PODS, OBJECT, EXTERNAL
	Resource *ResourceMetricSource `json:"resource,omitempty"`
	Pods     *PodsMetricSource     `json:"pods,omitempty"`
	Object   *ObjectMetricSource   `json:"object,omitempty"`
	External *ExternalMetricSource `json:"external,omitempty"`
}

// ResourceMetricSource defines resource-based scaling metrics
type ResourceMetricSource struct {
	Name   string        `json:"name"`
	Target *MetricTarget `json:"target"`
}

// PodsMetricSource defines pod-based scaling metrics
type PodsMetricSource struct {
	Metric *MetricIdentifier `json:"metric"`
	Target *MetricTarget     `json:"target"`
}

// ObjectMetricSource defines object-based scaling metrics
type ObjectMetricSource struct {
	DescribedObject *CrossVersionObjectReference `json:"describedObject"`
	Metric          *MetricIdentifier            `json:"metric"`
	Target          *MetricTarget                `json:"target"`
}

// ExternalMetricSource defines external scaling metrics
type ExternalMetricSource struct {
	Metric *MetricIdentifier `json:"metric"`
	Target *MetricTarget     `json:"target"`
}

// MetricIdentifier defines a metric identifier
type MetricIdentifier struct {
	Name     string            `json:"name"`
	Selector map[string]string `json:"selector,omitempty"`
}

// MetricTarget defines target values for metrics
type MetricTarget struct {
	Type               string  `json:"type"` // UTILIZATION, VALUE, AVERAGE_VALUE
	Value              *string `json:"value,omitempty"`
	AverageValue       *string `json:"averageValue,omitempty"`
	AverageUtilization *int32  `json:"averageUtilization,omitempty"`
}

// CrossVersionObjectReference identifies an object
type CrossVersionObjectReference struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// ScalingBehavior defines scaling behavior policies
type ScalingBehavior struct {
	ScaleUp   *HPAScalingRules `json:"scaleUp,omitempty"`
	ScaleDown *HPAScalingRules `json:"scaleDown,omitempty"`
}

// HPAScalingRules defines scaling rules
type HPAScalingRules struct {
	StabilizationWindowSeconds *int32             `json:"stabilizationWindowSeconds,omitempty"`
	SelectPolicy               *string            `json:"selectPolicy,omitempty"`
	Policies                   []HPAScalingPolicy `json:"policies,omitempty"`
}

// HPAScalingPolicy defines a single scaling policy
type HPAScalingPolicy struct {
	Type          string `json:"type"` // PODS, PERCENT
	Value         int32  `json:"value"`
	PeriodSeconds int32  `json:"periodSeconds"`
}

// MonitoringConfig defines monitoring configuration for deployments
type MonitoringConfig struct {
	Enabled         bool                 `json:"enabled"`
	MetricsEnabled  bool                 `json:"metricsEnabled"`
	LogsEnabled     bool                 `json:"logsEnabled"`
	TracingEnabled  bool                 `json:"tracingEnabled"`
	Prometheus      *PrometheusConfig    `json:"prometheus,omitempty"`
	Jaeger          *JaegerConfig        `json:"jaeger,omitempty"`
	CustomExporters []MonitoringExporter `json:"customExporters,omitempty"`
	HealthChecks    []HealthCheckConfig  `json:"healthChecks,omitempty"`
	Alerts          []AlertConfig        `json:"alerts,omitempty"`
}

// PrometheusConfig defines Prometheus monitoring configuration
type PrometheusConfig struct {
	Enabled     bool              `json:"enabled"`
	Path        string            `json:"path,omitempty"`
	Port        int32             `json:"port,omitempty"`
	Interval    time.Duration     `json:"interval,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// JaegerConfig defines Jaeger tracing configuration
type JaegerConfig struct {
	Enabled      bool    `json:"enabled"`
	Endpoint     string  `json:"endpoint,omitempty"`
	ServiceName  string  `json:"serviceName,omitempty"`
	SamplingRate float64 `json:"samplingRate,omitempty"`
}

// MonitoringExporter defines a custom monitoring exporter
type MonitoringExporter struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Endpoint string                 `json:"endpoint"`
	Format   string                 `json:"format,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	Name                string   `json:"name"`
	Type                string   `json:"type"` // HTTP, TCP, EXEC
	Path                string   `json:"path,omitempty"`
	Port                int32    `json:"port,omitempty"`
	Command             []string `json:"command,omitempty"`
	InitialDelaySeconds int32    `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32    `json:"periodSeconds,omitempty"`
	TimeoutSeconds      int32    `json:"timeoutSeconds,omitempty"`
	SuccessThreshold    int32    `json:"successThreshold,omitempty"`
	FailureThreshold    int32    `json:"failureThreshold,omitempty"`
}

// AlertConfig defines alert configuration
type AlertConfig struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Severity    string            `json:"severity"` // CRITICAL, WARNING, INFO
	Condition   string            `json:"condition"`
	Duration    time.Duration     `json:"duration,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Actions     []AlertAction     `json:"actions,omitempty"`
}

// AlertAction defines actions to take when an alert fires
type AlertAction struct {
	Type     string                 `json:"type"` // WEBHOOK, EMAIL, SLACK
	Endpoint string                 `json:"endpoint,omitempty"`
	Template string                 `json:"template,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// SecurityConfiguration defines security configuration for deployments
type SecurityConfiguration struct {
	PodSecurityContext       *PodSecurityContext       `json:"podSecurityContext,omitempty"`
	ContainerSecurityContext *ContainerSecurityContext `json:"containerSecurityContext,omitempty"`
	NetworkPolicy            *NetworkPolicyConfig      `json:"networkPolicy,omitempty"`
	ServiceMesh              *ServiceMeshConfig        `json:"serviceMesh,omitempty"`
	Encryption               *EncryptionConfig         `json:"encryption,omitempty"`
	Authentication           *AuthenticationConfig     `json:"authentication,omitempty"`
	Authorization            *AuthorizationConfig      `json:"authorization,omitempty"`
	Compliance               []ComplianceRequirement   `json:"compliance,omitempty"`
}

// PodSecurityContext defines pod-level security settings
type PodSecurityContext struct {
	RunAsUser          *int64          `json:"runAsUser,omitempty"`
	RunAsGroup         *int64          `json:"runAsGroup,omitempty"`
	RunAsNonRoot       *bool           `json:"runAsNonRoot,omitempty"`
	FSGroup            *int64          `json:"fsGroup,omitempty"`
	SELinuxOptions     *SELinuxOptions `json:"seLinuxOptions,omitempty"`
	SeccompProfile     *SeccompProfile `json:"seccompProfile,omitempty"`
	SupplementalGroups []int64         `json:"supplementalGroups,omitempty"`
	Sysctls            []Sysctl        `json:"sysctls,omitempty"`
}

// ContainerSecurityContext defines container-level security settings
type ContainerSecurityContext struct {
	RunAsUser                *int64          `json:"runAsUser,omitempty"`
	RunAsGroup               *int64          `json:"runAsGroup,omitempty"`
	RunAsNonRoot             *bool           `json:"runAsNonRoot,omitempty"`
	ReadOnlyRootFilesystem   *bool           `json:"readOnlyRootFilesystem,omitempty"`
	AllowPrivilegeEscalation *bool           `json:"allowPrivilegeEscalation,omitempty"`
	Privileged               *bool           `json:"privileged,omitempty"`
	Capabilities             *Capabilities   `json:"capabilities,omitempty"`
	SELinuxOptions           *SELinuxOptions `json:"seLinuxOptions,omitempty"`
	SeccompProfile           *SeccompProfile `json:"seccompProfile,omitempty"`
}

// SELinuxOptions defines SELinux options
type SELinuxOptions struct {
	Level string `json:"level,omitempty"`
	Role  string `json:"role,omitempty"`
	Type  string `json:"type,omitempty"`
	User  string `json:"user,omitempty"`
}

// SeccompProfile defines seccomp profile
type SeccompProfile struct {
	Type             string  `json:"type"` // RuntimeDefault, Localhost, Unconfined
	LocalhostProfile *string `json:"localhostProfile,omitempty"`
}

// Capabilities defines Linux capabilities
type Capabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

// Sysctl defines a sysctl and its value
type Sysctl struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// NetworkPolicyConfig defines network policy configuration
type NetworkPolicyConfig struct {
	Enabled     bool                `json:"enabled"`
	Ingress     []NetworkPolicyRule `json:"ingress,omitempty"`
	Egress      []NetworkPolicyRule `json:"egress,omitempty"`
	PolicyTypes []string            `json:"policyTypes,omitempty"` // Ingress, Egress
}

// NetworkPolicyRule defines a network policy rule
type NetworkPolicyRule struct {
	From  []NetworkPolicyPeer `json:"from,omitempty"`
	To    []NetworkPolicyPeer `json:"to,omitempty"`
	Ports []NetworkPolicyPort `json:"ports,omitempty"`
}

// NetworkPolicyPeer defines a network policy peer
type NetworkPolicyPeer struct {
	PodSelector       *LabelSelector `json:"podSelector,omitempty"`
	NamespaceSelector *LabelSelector `json:"namespaceSelector,omitempty"`
	IPBlock           *IPBlock       `json:"ipBlock,omitempty"`
}

// NetworkPolicyPort defines a network policy port
type NetworkPolicyPort struct {
	Protocol *string `json:"protocol,omitempty"`
	Port     *string `json:"port,omitempty"`
	EndPort  *int32  `json:"endPort,omitempty"`
}

// IPBlock defines an IP block for network policies
type IPBlock struct {
	CIDR   string   `json:"cidr"`
	Except []string `json:"except,omitempty"`
}

// ServiceMeshConfig defines service mesh configuration
type ServiceMeshConfig struct {
	Enabled       bool                 `json:"enabled"`
	Provider      string               `json:"provider"` // ISTIO, LINKERD, CONSUL_CONNECT
	Injection     bool                 `json:"injection"`
	MTLS          *MTLSConfig          `json:"mtls,omitempty"`
	TrafficPolicy *TrafficPolicyConfig `json:"trafficPolicy,omitempty"`
	Retries       *RetryConfig         `json:"retries,omitempty"`
	Timeout       *TimeoutConfig       `json:"timeout,omitempty"`
}

// MTLSConfig defines mutual TLS configuration
type MTLSConfig struct {
	Mode string `json:"mode"` // STRICT, PERMISSIVE, DISABLE
}

// TrafficPolicyConfig defines traffic policy configuration
type TrafficPolicyConfig struct {
	LoadBalancer     *TrafficLoadBalancerConfig `json:"loadBalancer,omitempty"`
	CircuitBreaker   *CircuitBreakerConfig      `json:"circuitBreaker,omitempty"`
	OutlierDetection *OutlierDetectionConfig    `json:"outlierDetection,omitempty"`
}

// TrafficLoadBalancerConfig defines load balancer configuration for traffic policies
type TrafficLoadBalancerConfig struct {
	Simple string `json:"simple,omitempty"` // ROUND_ROBIN, LEAST_CONN, RANDOM, PASSTHROUGH
}

// CircuitBreakerConfig defines circuit breaker configuration
type CircuitBreakerConfig struct {
	MaxConnections     *int32 `json:"maxConnections,omitempty"`
	MaxPendingRequests *int32 `json:"maxPendingRequests,omitempty"`
	MaxRequests        *int32 `json:"maxRequests,omitempty"`
	MaxRetries         *int32 `json:"maxRetries,omitempty"`
}

// OutlierDetectionConfig defines outlier detection configuration
type OutlierDetectionConfig struct {
	ConsecutiveErrors  *int32        `json:"consecutiveErrors,omitempty"`
	Interval           time.Duration `json:"interval,omitempty"`
	BaseEjectionTime   time.Duration `json:"baseEjectionTime,omitempty"`
	MaxEjectionPercent *int32        `json:"maxEjectionPercent,omitempty"`
	MinHealthPercent   *int32        `json:"minHealthPercent,omitempty"`
}

// RetryConfig defines retry configuration
type RetryConfig struct {
	Attempts      int32         `json:"attempts"`
	PerTryTimeout time.Duration `json:"perTryTimeout,omitempty"`
	RetryOn       string        `json:"retryOn,omitempty"`
}

// TimeoutConfig defines timeout configuration
type TimeoutConfig struct {
	Request time.Duration `json:"request,omitempty"`
}

// EncryptionConfig defines encryption configuration
type EncryptionConfig struct {
	InTransit *InTransitEncryption `json:"inTransit,omitempty"`
	AtRest    *AtRestEncryption    `json:"atRest,omitempty"`
}

// InTransitEncryption defines encryption in transit
type InTransitEncryption struct {
	Enabled      bool     `json:"enabled"`
	TLSVersion   string   `json:"tlsVersion,omitempty"`
	CipherSuites []string `json:"cipherSuites,omitempty"`
}

// AtRestEncryption defines encryption at rest
type AtRestEncryption struct {
	Enabled   bool   `json:"enabled"`
	Algorithm string `json:"algorithm,omitempty"`
	KeySource string `json:"keySource,omitempty"`
}

// AuthenticationConfig defines authentication configuration
type AuthenticationConfig struct {
	Type   string                 `json:"type"` // NONE, BASIC, BEARER, OIDC, MTLS
	OIDC   *OIDCConfig            `json:"oidc,omitempty"`
	MTLS   *MTLSAuthConfig        `json:"mtls,omitempty"`
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// OIDCConfig defines OpenID Connect configuration
type OIDCConfig struct {
	Issuer       string   `json:"issuer"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	Scopes       []string `json:"scopes,omitempty"`
	RedirectURI  string   `json:"redirectUri,omitempty"`
}

// MTLSAuthConfig defines mutual TLS authentication configuration
type MTLSAuthConfig struct {
	CACert     string `json:"caCert"`
	ClientCert string `json:"clientCert"`
	ClientKey  string `json:"clientKey"`
}

// AuthorizationConfig defines authorization configuration
type AuthorizationConfig struct {
	Type     string              `json:"type"` // RBAC, ABAC, WEBHOOK
	RBAC     *RBACConfig         `json:"rbac,omitempty"`
	Webhook  *WebhookAuthzConfig `json:"webhook,omitempty"`
	Policies []AuthzPolicy       `json:"policies,omitempty"`
}

// RBACConfig defines RBAC configuration
type RBACConfig struct {
	Enabled  bool              `json:"enabled"`
	Roles    []RBACRole        `json:"roles,omitempty"`
	Bindings []RBACRoleBinding `json:"bindings,omitempty"`
}

// RBACRole defines an RBAC role
type RBACRole struct {
	Name  string       `json:"name"`
	Rules []PolicyRule `json:"rules"`
}

// RBACRoleBinding defines an RBAC role binding
type RBACRoleBinding struct {
	Name     string    `json:"name"`
	RoleRef  RoleRef   `json:"roleRef"`
	Subjects []Subject `json:"subjects"`
}

// PolicyRule defines a policy rule
type PolicyRule struct {
	APIGroups []string `json:"apiGroups,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Verbs     []string `json:"verbs"`
}

// RoleRef defines a role reference
type RoleRef struct {
	APIGroup string `json:"apiGroup"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

// Subject defines a subject
type Subject struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// WebhookAuthzConfig defines webhook authorization configuration
type WebhookAuthzConfig struct {
	URL           string        `json:"url"`
	Timeout       time.Duration `json:"timeout,omitempty"`
	FailurePolicy string        `json:"failurePolicy,omitempty"` // ALLOW, DENY
	CACert        string        `json:"caCert,omitempty"`
}

// AuthzPolicy defines an authorization policy
type AuthzPolicy struct {
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	Resource  string `json:"resource"`
	Action    string `json:"action"`
	Condition string `json:"condition,omitempty"`
	Effect    string `json:"effect"` // ALLOW, DENY
}

// ComplianceRequirement defines a compliance requirement
type ComplianceRequirement struct {
	Name        string   `json:"name"`
	Framework   string   `json:"framework"` // SOC2, PCI-DSS, HIPAA, ISO27001
	Controls    []string `json:"controls"`
	Description string   `json:"description,omitempty"`
	Mandatory   bool     `json:"mandatory"`
}

// DeploymentOptions defines options for template deployment
type DeploymentOptions struct {
	Timeout          time.Duration          `json:"timeout"`
	RetryPolicy      *RetryPolicy           `json:"retryPolicy,omitempty"`
	RollbackPolicy   *RollbackPolicy        `json:"rollbackPolicy,omitempty"`
	ScalingPolicy    *ScalingPolicy         `json:"scalingPolicy,omitempty"`
	MonitoringConfig *MonitoringConfig      `json:"monitoringConfig,omitempty"`
	SecurityConfig   *SecurityConfiguration `json:"securityConfig,omitempty"`
}

// ResourceRequirements defines resource requirements for deployment templates
type ResourceRequirements struct {
	MinCPU              string                    `json:"minCpu,omitempty"`
	MinMemory           string                    `json:"minMemory,omitempty"`
	MinStorage          string                    `json:"minStorage,omitempty"`
	MinNodes            int                       `json:"minNodes,omitempty"`
	MaxCPU              string                    `json:"maxCpu,omitempty"`
	MaxMemory           string                    `json:"maxMemory,omitempty"`
	MaxStorage          string                    `json:"maxStorage,omitempty"`
	MaxNodes            int                       `json:"maxNodes,omitempty"`
	RequiredFeatures    []string                  `json:"requiredFeatures,omitempty"`
	SupportedArch       []string                  `json:"supportedArchitectures,omitempty"`
	NetworkRequirements []*NetworkRequirement     `json:"networkRequirements,omitempty"`
	StorageRequirements []*StorageRequirement     `json:"storageRequirements,omitempty"`
	AcceleratorReq      []*AcceleratorRequirement `json:"acceleratorRequirements,omitempty"`
}

// AffinityRules represents affinity and anti-affinity rules
type AffinityRules struct {
	NodeAffinity    *NodeAffinity `json:"nodeAffinity,omitempty"`
	PodAffinity     *PodAffinity  `json:"podAffinity,omitempty"`
	PodAntiAffinity *PodAffinity  `json:"podAntiAffinity,omitempty"`
}

// PodAffinity represents pod affinity rules
type PodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []*PodAffinityTerm         `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []*WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeSelectorTerm represents a node selector term
type NodeSelectorTerm struct {
	MatchExpressions []*NodeSelectorRequirement `json:"matchExpressions,omitempty"`
	MatchFields      []*NodeSelectorRequirement `json:"matchFields,omitempty"`
}

// NodeSelectorRequirement represents a node selector requirement
type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"` // In, NotIn, Exists, DoesNotExist, Gt, Lt
	Values   []string `json:"values,omitempty"`
}

// PreferredSchedulingTerm represents a preferred scheduling term
type PreferredSchedulingTerm struct {
	Weight     int32             `json:"weight"`
	Preference *NodeSelectorTerm `json:"preference"`
}

// PodAffinityTerm represents a pod affinity term
type PodAffinityTerm struct {
	LabelSelector *LabelSelector `json:"labelSelector,omitempty"`
	Namespaces    []string       `json:"namespaces,omitempty"`
	TopologyKey   string         `json:"topologyKey"`
}

// WeightedPodAffinityTerm represents a weighted pod affinity term
type WeightedPodAffinityTerm struct {
	Weight          int32            `json:"weight"`
	PodAffinityTerm *PodAffinityTerm `json:"podAffinityTerm"`
}

// LabelSelector represents a label selector
type LabelSelector struct {
	MatchLabels      map[string]string           `json:"matchLabels,omitempty"`
	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// LabelSelectorRequirement represents a label selector requirement
type LabelSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"` // In, NotIn, Exists, DoesNotExist
	Values   []string `json:"values,omitempty"`
}

// DeploymentInstance represents a deployment instance created from a template
type DeploymentInstance struct {
	DeploymentManagerID string                 `json:"deploymentManagerId"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	ParentDeploymentID  string                 `json:"parentDeploymentId,omitempty"`
	Extensions          map[string]interface{} `json:"extensions,omitempty"`

	// Deployment specification
	TemplateID      string                `json:"templateId"`
	TemplateVersion string                `json:"templateVersion,omitempty"`
	InputParameters *runtime.RawExtension `json:"inputParameters,omitempty"`
	OutputValues    *runtime.RawExtension `json:"outputValues,omitempty"`
	ResourcePoolID  string                `json:"resourcePoolId"`

	// Deployment status and lifecycle
	Status    *DeploymentStatus   `json:"status"`
	Resources []*DeployedResource `json:"resources,omitempty"`
	Services  []*DeployedService  `json:"services,omitempty"`

	// Lifecycle information
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string    `json:"createdBy,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	State           string                 `json:"state"`  // PENDING, RUNNING, FAILED, SUCCEEDED, DELETING
	Phase           string                 `json:"phase"`  // CREATING, UPDATING, SCALING, TERMINATING
	Health          string                 `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN
	Progress        *DeploymentProgress    `json:"progress,omitempty"`
	Conditions      []DeploymentCondition  `json:"conditions,omitempty"`
	ErrorMessage    string                 `json:"errorMessage,omitempty"`
	Events          []*DeploymentEvent     `json:"events,omitempty"`
	LastStateChange time.Time              `json:"lastStateChange"`
	LastHealthCheck time.Time              `json:"lastHealthCheck"`
	Metrics         map[string]interface{} `json:"metrics,omitempty"`
}

// DeploymentProgress represents the progress of a deployment operation
type DeploymentProgress struct {
	TotalSteps             int32         `json:"totalSteps"`
	CompletedSteps         int32         `json:"completedSteps"`
	CurrentStep            string        `json:"currentStep,omitempty"`
	PercentComplete        float64       `json:"percentComplete"`
	EstimatedTimeRemaining time.Duration `json:"estimatedTimeRemaining,omitempty"`
}

// DeploymentCondition represents a condition of the deployment
type DeploymentCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"` // True, False, Unknown
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	LastUpdateTime     time.Time `json:"lastUpdateTime,omitempty"`
}

// DeploymentEvent represents an event that occurred during deployment lifecycle
type DeploymentEvent struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"` // NORMAL, WARNING, ERROR
	Reason         string                 `json:"reason"`
	Message        string                 `json:"message"`
	Component      string                 `json:"component,omitempty"`
	Source         string                 `json:"source,omitempty"`
	FirstTimestamp time.Time              `json:"firstTimestamp"`
	LastTimestamp  time.Time              `json:"lastTimestamp"`
	Count          int32                  `json:"count"`
	AdditionalData map[string]interface{} `json:"additionalData,omitempty"`
}

// DeployedResource represents a resource that has been deployed
type DeployedResource struct {
	ResourceID    string                 `json:"resourceId"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Kind          string                 `json:"kind,omitempty"`
	Namespace     string                 `json:"namespace,omitempty"`
	Status        *ResourceStatus        `json:"status"`
	Configuration *runtime.RawExtension  `json:"configuration,omitempty"`
	Dependencies  []string               `json:"dependencies,omitempty"`
	Endpoints     []*ResourceEndpoint    `json:"endpoints,omitempty"`
	Metrics       map[string]interface{} `json:"metrics,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// DeployedService represents a service that has been deployed
type DeployedService struct {
	ServiceID     string                `json:"serviceId"`
	Name          string                `json:"name"`
	Type          string                `json:"type"` // ClusterIP, NodePort, LoadBalancer, ExternalName
	Namespace     string                `json:"namespace,omitempty"`
	Ports         []*ServicePort        `json:"ports,omitempty"`
	Endpoints     []*ServiceEndpoint    `json:"endpoints,omitempty"`
	Status        *ServiceStatus        `json:"status"`
	Configuration *runtime.RawExtension `json:"configuration,omitempty"`
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`
}

// ResourceEndpoint represents an endpoint exposed by a deployed resource
type ResourceEndpoint struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int32  `json:"port"`
	Path     string `json:"path,omitempty"`
	Scheme   string `json:"scheme,omitempty"`
	Type     string `json:"type"` // HTTP, HTTPS, TCP, UDP, GRPC
}

// ServicePort represents a port exposed by a service
type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort string `json:"targetPort,omitempty"`
	NodePort   int32  `json:"nodePort,omitempty"`
}

// ServiceEndpoint represents an endpoint of a service
type ServiceEndpoint struct {
	Address    string            `json:"address"`
	Port       int32             `json:"port"`
	Protocol   string            `json:"protocol"`
	Ready      bool              `json:"ready"`
	Conditions map[string]string `json:"conditions,omitempty"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Type                string             `json:"type"`
	ClusterIP           string             `json:"clusterIP,omitempty"`
	ExternalIPs         []string           `json:"externalIPs,omitempty"`
	LoadBalancerIP      string             `json:"loadBalancerIP,omitempty"`
	LoadBalancerIngress []string           `json:"loadBalancerIngress,omitempty"`
	Conditions          []ServiceCondition `json:"conditions,omitempty"`
	Health              string             `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY
	LastHealthCheck     time.Time          `json:"lastHealthCheck"`
}

// ServiceCondition represents a condition of the service
type ServiceCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
}

// Filter types for deployment queries

// DeploymentTemplateFilter defines filters for querying deployment templates
type DeploymentTemplateFilter struct {
	Names         []string          `json:"names,omitempty"`
	Categories    []string          `json:"categories,omitempty"`
	Types         []string          `json:"types,omitempty"`
	Versions      []string          `json:"versions,omitempty"`
	Authors       []string          `json:"authors,omitempty"`
	Keywords      []string          `json:"keywords,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	CreatedAfter  *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore *time.Time        `json:"createdBefore,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
	SortBy        string            `json:"sortBy,omitempty"`
	SortOrder     string            `json:"sortOrder,omitempty"`
}

// DeploymentFilter defines filters for querying deployments
type DeploymentFilter struct {
	Names               []string          `json:"names,omitempty"`
	TemplateIDs         []string          `json:"templateIds,omitempty"`
	ResourcePoolIDs     []string          `json:"resourcePoolIds,omitempty"`
	States              []string          `json:"states,omitempty"`
	Phases              []string          `json:"phases,omitempty"`
	HealthStates        []string          `json:"healthStates,omitempty"`
	ParentDeploymentIDs []string          `json:"parentDeploymentIds,omitempty"`
	CreatedBy           []string          `json:"createdBy,omitempty"`
	Labels              map[string]string `json:"labels,omitempty"`
	CreatedAfter        *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore       *time.Time        `json:"createdBefore,omitempty"`
	Limit               int               `json:"limit,omitempty"`
	Offset              int               `json:"offset,omitempty"`
	SortBy              string            `json:"sortBy,omitempty"`
	SortOrder           string            `json:"sortOrder,omitempty"`
}

// Request types for deployment management operations

// CreateDeploymentTemplateRequest represents a request to create a deployment template
type CreateDeploymentTemplateRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Version      string                 `json:"version"`
	Category     string                 `json:"category"`
	Type         string                 `json:"type"`
	Content      *runtime.RawExtension  `json:"content"`
	InputSchema  *runtime.RawExtension  `json:"inputSchema,omitempty"`
	OutputSchema *runtime.RawExtension  `json:"outputSchema,omitempty"`
	Author       string                 `json:"author,omitempty"`
	License      string                 `json:"license,omitempty"`
	Keywords     []string               `json:"keywords,omitempty"`
	Dependencies []*TemplateDependency  `json:"dependencies,omitempty"`
	Requirements *ResourceRequirements  `json:"requirements,omitempty"`
	Extensions   map[string]interface{} `json:"extensions,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

// UpdateDeploymentTemplateRequest represents a request to update a deployment template
type UpdateDeploymentTemplateRequest struct {
	Name         *string                `json:"name,omitempty"`
	Description  *string                `json:"description,omitempty"`
	Version      *string                `json:"version,omitempty"`
	Content      *runtime.RawExtension  `json:"content,omitempty"`
	InputSchema  *runtime.RawExtension  `json:"inputSchema,omitempty"`
	OutputSchema *runtime.RawExtension  `json:"outputSchema,omitempty"`
	Keywords     []string               `json:"keywords,omitempty"`
	Dependencies []*TemplateDependency  `json:"dependencies,omitempty"`
	Requirements *ResourceRequirements  `json:"requirements,omitempty"`
	Extensions   map[string]interface{} `json:"extensions,omitempty"`
	Metadata     map[string]string      `json:"metadata,omitempty"`
}

// CreateDeploymentRequest represents a request to create a deployment
type CreateDeploymentRequest struct {
	Name               string                 `json:"name"`
	Description        string                 `json:"description,omitempty"`
	TemplateID         string                 `json:"templateId"`
	TemplateVersion    string                 `json:"templateVersion,omitempty"`
	ResourcePoolID     string                 `json:"resourcePoolId"`
	InputParameters    *runtime.RawExtension  `json:"inputParameters,omitempty"`
	ParentDeploymentID string                 `json:"parentDeploymentId,omitempty"`
	Extensions         map[string]interface{} `json:"extensions,omitempty"`
	Metadata           map[string]string      `json:"metadata,omitempty"`

	// Deployment options
	DryRun         bool            `json:"dryRun,omitempty"`
	Timeout        time.Duration   `json:"timeout,omitempty"`
	RetryPolicy    *RetryPolicy    `json:"retryPolicy,omitempty"`
	RollbackPolicy *RollbackPolicy `json:"rollbackPolicy,omitempty"`
}

// UpdateDeploymentRequest represents a request to update a deployment
type UpdateDeploymentRequest struct {
	Description     *string                `json:"description,omitempty"`
	InputParameters *runtime.RawExtension  `json:"inputParameters,omitempty"`
	Extensions      map[string]interface{} `json:"extensions,omitempty"`
	Metadata        map[string]string      `json:"metadata,omitempty"`

	// Update options
	UpdateStrategy *UpdateStrategy `json:"updateStrategy,omitempty"`
	Timeout        time.Duration   `json:"timeout,omitempty"`
	RetryPolicy    *RetryPolicy    `json:"retryPolicy,omitempty"`
}

// RetryPolicy defines retry behavior for deployment operations
type RetryPolicy struct {
	MaxRetries      int           `json:"maxRetries"`
	RetryDelay      time.Duration `json:"retryDelay"`
	BackoffFactor   float64       `json:"backoffFactor"`
	MaxRetryDelay   time.Duration `json:"maxRetryDelay"`
	RetryConditions []string      `json:"retryConditions,omitempty"`
}

// RollbackPolicy defines rollback behavior for failed deployments
type RollbackPolicy struct {
	Enabled            bool          `json:"enabled"`
	AutoRollback       bool          `json:"autoRollback"`
	RollbackDelay      time.Duration `json:"rollbackDelay,omitempty"`
	MaxRollbacks       int           `json:"maxRollbacks"`
	RollbackConditions []string      `json:"rollbackConditions,omitempty"`
}

// UpdateStrategy defines how deployment updates should be performed
type UpdateStrategy struct {
	Type            string        `json:"type"` // RECREATE, ROLLING_UPDATE, BLUE_GREEN, CANARY
	MaxUnavailable  string        `json:"maxUnavailable,omitempty"`
	MaxSurge        string        `json:"maxSurge,omitempty"`
	Timeout         time.Duration `json:"timeout,omitempty"`
	PauseConditions []string      `json:"pauseConditions,omitempty"`
}

// Additional helper types for deployments

// TemplateParameter defines a parameter for deployment templates
type TemplateParameter struct {
	Name          string               `json:"name"`
	Type          string               `json:"type"` // STRING, INTEGER, BOOLEAN, ARRAY, OBJECT
	Description   string               `json:"description,omitempty"`
	Required      bool                 `json:"required"`
	DefaultValue  interface{}          `json:"defaultValue,omitempty"`
	MinValue      interface{}          `json:"minValue,omitempty"`
	MaxValue      interface{}          `json:"maxValue,omitempty"`
	AllowedValues []interface{}        `json:"allowedValues,omitempty"`
	Pattern       string               `json:"pattern,omitempty"`
	Validation    *ParameterValidation `json:"validation,omitempty"`
}

// ParameterValidation defines validation rules for template parameters
type ParameterValidation struct {
	Script       string `json:"script,omitempty"`
	Expression   string `json:"expression,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// ValidationRule defines validation rules for deployment templates
type ValidationRule struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // SYNTAX, SEMANTIC, COMPATIBILITY
	Description  string `json:"description,omitempty"`
	Script       string `json:"script,omitempty"`
	Expression   string `json:"expression,omitempty"`
	Severity     string `json:"severity"` // ERROR, WARNING, INFO
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// CompatibilityInfo defines compatibility information for templates
type CompatibilityInfo struct {
	Platform           string   `json:"platform"`
	Version            string   `json:"version"`
	Architectures      []string `json:"architectures,omitempty"`
	KubernetesVersions []string `json:"kubernetesVersions,omitempty"`
	Compatible         bool     `json:"compatible"`
	Limitations        []string `json:"limitations,omitempty"`
}

// TemplateStatus represents the status of a deployment template
type TemplateStatus struct {
	State         string    `json:"state"`      // DRAFT, ACTIVE, DEPRECATED, OBSOLETE
	Validation    string    `json:"validation"` // PENDING, VALID, INVALID
	LastValidated time.Time `json:"lastValidated,omitempty"`
	Errors        []string  `json:"errors,omitempty"`
	Warnings      []string  `json:"warnings,omitempty"`
}

// NetworkRequirement defines network requirements for deployments
type NetworkRequirement struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // MANAGEMENT, DATA, CONTROL, EXTERNAL
	Bandwidth   string   `json:"bandwidth,omitempty"`
	Latency     string   `json:"latency,omitempty"`
	Protocols   []string `json:"protocols,omitempty"`
	Ports       []int32  `json:"ports,omitempty"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
}

// StorageRequirement defines storage requirements for deployments
type StorageRequirement struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // BLOCK, FILE, OBJECT
	Size        string   `json:"size"`
	IOPS        *int32   `json:"iops,omitempty"`
	Throughput  string   `json:"throughput,omitempty"`
	AccessModes []string `json:"accessModes,omitempty"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
}

// AcceleratorRequirement defines accelerator requirements for deployments
type AcceleratorRequirement struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // GPU, FPGA, TPU
	Model       string `json:"model,omitempty"`
	Count       int32  `json:"count"`
	Memory      string `json:"memory,omitempty"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// Prerequisite defines a prerequisite for deployment templates
type Prerequisite struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // SERVICE, RESOURCE, CONFIGURATION
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Validation  string `json:"validation,omitempty"`
}

// EventLocation represents the location where an event occurred
type EventLocation struct {
	DataCenter string  `json:"dataCenter,omitempty"`
	Rack       string  `json:"rack,omitempty"`
	Node       string  `json:"node,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
}

// Constants for deployment management

const (
	// Deployment Template Categories
	TemplateCategoryVNF = "VNF"
	TemplateCategoryCNF = "CNF"
	TemplateCategoryPNF = "PNF"
	TemplateCategoryNS  = "NS"

	// Deployment Template Types
	TemplateTypeHelm       = "HELM"
	TemplateTypeKubernetes = "KUBERNETES"
	TemplateTypeTerraform  = "TERRAFORM"
	TemplateTypeAnsible    = "ANSIBLE"

	// Deployment States
	DeploymentStatePending   = "PENDING"
	DeploymentStateRunning   = "RUNNING"
	DeploymentStateFailed    = "FAILED"
	DeploymentStateSucceeded = "SUCCEEDED"
	DeploymentStateDeleting  = "DELETING"

	// Deployment Phases
	DeploymentPhaseCreating    = "CREATING"
	DeploymentPhaseUpdating    = "UPDATING"
	DeploymentPhaseScaling     = "SCALING"
	DeploymentPhaseTerminating = "TERMINATING"

	// Update Strategy Types
	UpdateStrategyRecreate      = "RECREATE"
	UpdateStrategyRollingUpdate = "ROLLING_UPDATE"
	UpdateStrategyBlueGreen     = "BLUE_GREEN"
	UpdateStrategyCanary        = "CANARY"

	// Event Types
	EventTypeNormal  = "NORMAL"
	EventTypeWarning = "WARNING"
	EventTypeError   = "ERROR"

	// Service Types
	ServiceTypeClusterIP    = "ClusterIP"
	ServiceTypeNodePort     = "NodePort"
	ServiceTypeLoadBalancer = "LoadBalancer"
	ServiceTypeExternalName = "ExternalName"

	// Endpoint Types
	EndpointTypeHTTP  = "HTTP"
	EndpointTypeHTTPS = "HTTPS"
	EndpointTypeTCP   = "TCP"
	EndpointTypeUDP   = "UDP"
	EndpointTypeGRPC  = "GRPC"

	// Template Parameter Types
	ParameterTypeString  = "STRING"
	ParameterTypeInteger = "INTEGER"
	ParameterTypeBoolean = "BOOLEAN"
	ParameterTypeArray   = "ARRAY"
	ParameterTypeObject  = "OBJECT"

	// Template States
	TemplateStateDraft      = "DRAFT"
	TemplateStateActive     = "ACTIVE"
	TemplateStateDeprecated = "DEPRECATED"
	TemplateStateObsolete   = "OBSOLETE"

	// Validation States
	ValidationStatePending = "PENDING"
	ValidationStateValid   = "VALID"
	ValidationStateInvalid = "INVALID"

	// Hook Types
	HookTypePreCreate  = "PRE_CREATE"
	HookTypePostCreate = "POST_CREATE"
	HookTypePreUpdate  = "PRE_UPDATE"
	HookTypePostUpdate = "POST_UPDATE"
	HookTypePreDelete  = "PRE_DELETE"
	HookTypePostDelete = "POST_DELETE"

	// Hook Failure Policies
	HookFailurePolicyIgnore = "IGNORE"
	HookFailurePolicyAbort  = "ABORT"
)
