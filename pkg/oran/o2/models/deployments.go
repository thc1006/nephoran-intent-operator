package models

import (
	"time"
	
	"k8s.io/apimachinery/pkg/runtime"
)

// Deployment Template and Management Models following O-RAN.WG6.O2ims-Interface-v01.01

// DeploymentTemplate represents a deployment template following O2 IMS specification
type DeploymentTemplate struct {
	DeploymentTemplateID string                 `json:"deploymentTemplateId"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description,omitempty"`
	Version             string                 `json:"version"`
	Provider            string                 `json:"provider,omitempty"`
	Category            string                 `json:"category,omitempty"` // VNF, CNF, PNF, NS
	
	// Template specifications
	TemplateSpec        *TemplateSpecification `json:"templateSpec"`
	RequiredResources   *ResourceRequirements  `json:"requiredResources"`
	SupportedParameters []*TemplateParameter   `json:"supportedParameters,omitempty"`
	
	// Validation and compatibility
	ValidationRules     []*ValidationRule      `json:"validationRules,omitempty"`
	CompatibilityMatrix []*CompatibilityInfo   `json:"compatibilityMatrix,omitempty"`
	
	// Metadata and lifecycle
	Tags                map[string]string      `json:"tags,omitempty"`
	Labels              map[string]string      `json:"labels,omitempty"`
	Extensions          map[string]interface{} `json:"extensions,omitempty"`
	Status              *TemplateStatus        `json:"status"`
	CreatedAt           time.Time              `json:"createdAt"`
	UpdatedAt           time.Time              `json:"updatedAt"`
	CreatedBy           string                 `json:"createdBy,omitempty"`
	UpdatedBy           string                 `json:"updatedBy,omitempty"`
}

// TemplateDependency represents a dependency of a deployment template
type TemplateDependency struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Type         string `json:"type"` // TEMPLATE, RESOURCE, SERVICE
	Required     bool   `json:"required"`
	Description  string `json:"description,omitempty"`
}

// TemplateSpecification defines the deployment template specification
type TemplateSpecification struct {
	Type                string                 `json:"type"` // HEAT, HELM, KUBERNETES, TERRAFORM
	Content             *runtime.RawExtension  `json:"content"`
	ContentType         string                 `json:"contentType"` // yaml, json, zip
	MainTemplate        string                 `json:"mainTemplate,omitempty"`
	NestedTemplates     map[string]*runtime.RawExtension `json:"nestedTemplates,omitempty"`
	
	// Deployment configuration
	DeploymentOptions   *DeploymentOptions     `json:"deploymentOptions,omitempty"`
	Hooks               []*DeploymentHook      `json:"hooks,omitempty"`
	
	// Resource mappings
	ResourceMappings    []*ResourceMapping     `json:"resourceMappings,omitempty"`
	NetworkMappings     []*NetworkMapping      `json:"networkMappings,omitempty"`
	StorageMappings     []*StorageMapping      `json:"storageMappings,omitempty"`
	
	// Dependencies
	Dependencies        []*TemplateDependency  `json:"dependencies,omitempty"`
	Prerequisites       []*Prerequisite        `json:"prerequisites,omitempty"`
}

// DeploymentOptions defines options for template deployment
type DeploymentOptions struct {
	Timeout             time.Duration          `json:"timeout"`
	RetryPolicy         *RetryPolicy           `json:"retryPolicy,omitempty"`
	RollbackPolicy      *RollbackPolicy        `json:"rollbackPolicy,omitempty"`
	ScalingPolicy       *ScalingPolicy         `json:"scalingPolicy,omitempty"`
	MonitoringConfig    *MonitoringConfig      `json:"monitoringConfig,omitempty"`
	SecurityConfig      *SecurityConfiguration `json:"securityConfig,omitempty"`
}

// ResourceRequirements defines resource requirements for deployment templates
type ResourceRequirements struct {
	MinCPU              string                 `json:"minCpu,omitempty"`
	MinMemory           string                 `json:"minMemory,omitempty"`
	MinStorage          string                 `json:"minStorage,omitempty"`
	MinNodes            int                    `json:"minNodes,omitempty"`
	MaxCPU              string                 `json:"maxCpu,omitempty"`
	MaxMemory           string                 `json:"maxMemory,omitempty"`
	MaxStorage          string                 `json:"maxStorage,omitempty"`
	MaxNodes            int                    `json:"maxNodes,omitempty"`
	RequiredFeatures    []string               `json:"requiredFeatures,omitempty"`
	SupportedArch       []string               `json:"supportedArchitectures,omitempty"`
	NetworkRequirements []*NetworkRequirement  `json:"networkRequirements,omitempty"`
	StorageRequirements []*StorageRequirement  `json:"storageRequirements,omitempty"`
	AcceleratorReq      []*AcceleratorRequirement `json:"acceleratorRequirements,omitempty"`
}

// AffinityRules represents affinity and anti-affinity rules
type AffinityRules struct {
	NodeAffinity    *NodeAffinity    `json:"nodeAffinity,omitempty"`
	PodAffinity     *PodAffinity     `json:"podAffinity,omitempty"`
	PodAntiAffinity *PodAffinity     `json:"podAntiAffinity,omitempty"`
}

// NodeAffinity represents node affinity rules
type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []*NodeSelectorTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []*PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAffinity represents pod affinity rules
type PodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []*PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
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
	MatchLabels      map[string]string        `json:"matchLabels,omitempty"`
	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// LabelSelectorRequirement represents a label selector requirement
type LabelSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"` // In, NotIn, Exists, DoesNotExist
	Values   []string `json:"values,omitempty"`
}

// Deployment represents a deployment instance created from a template
type Deployment struct {
	DeploymentManagerID string                 `json:"deploymentManagerId"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description,omitempty"`
	ParentDeploymentID string                 `json:"parentDeploymentId,omitempty"`
	Extensions         map[string]interface{} `json:"extensions,omitempty"`
	
	// Deployment specification
	TemplateID         string                 `json:"templateId"`
	TemplateVersion    string                 `json:"templateVersion,omitempty"`
	InputParameters    *runtime.RawExtension  `json:"inputParameters,omitempty"`
	OutputValues       *runtime.RawExtension  `json:"outputValues,omitempty"`
	ResourcePoolID     string                 `json:"resourcePoolId"`
	
	// Deployment status and lifecycle
	Status             *DeploymentStatus      `json:"status"`
	Resources          []*DeployedResource    `json:"resources,omitempty"`
	Services           []*DeployedService     `json:"services,omitempty"`
	
	// Lifecycle information
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
	CreatedBy          string                 `json:"createdBy,omitempty"`
	UpdatedBy          string                 `json:"updatedBy,omitempty"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	State              string                 `json:"state"` // PENDING, RUNNING, FAILED, SUCCEEDED, DELETING
	Phase              string                 `json:"phase"` // CREATING, UPDATING, SCALING, TERMINATING
	Health             string                 `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN
	Progress           *DeploymentProgress    `json:"progress,omitempty"`
	Conditions         []DeploymentCondition  `json:"conditions,omitempty"`
	ErrorMessage       string                 `json:"errorMessage,omitempty"`
	Events             []*DeploymentEvent     `json:"events,omitempty"`
	LastStateChange    time.Time              `json:"lastStateChange"`
	LastHealthCheck    time.Time              `json:"lastHealthCheck"`
	Metrics            map[string]interface{} `json:"metrics,omitempty"`
}

// DeploymentProgress represents the progress of a deployment operation
type DeploymentProgress struct {
	TotalSteps       int32     `json:"totalSteps"`
	CompletedSteps   int32     `json:"completedSteps"`
	CurrentStep      string    `json:"currentStep,omitempty"`
	PercentComplete  float64   `json:"percentComplete"`
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
	ID                 string                 `json:"id"`
	Type               string                 `json:"type"` // NORMAL, WARNING, ERROR
	Reason             string                 `json:"reason"`
	Message            string                 `json:"message"`
	Component          string                 `json:"component,omitempty"`
	Source             string                 `json:"source,omitempty"`
	FirstTimestamp     time.Time              `json:"firstTimestamp"`
	LastTimestamp      time.Time              `json:"lastTimestamp"`
	Count              int32                  `json:"count"`
	AdditionalData     map[string]interface{} `json:"additionalData,omitempty"`
}

// DeployedResource represents a resource that has been deployed
type DeployedResource struct {
	ResourceID         string                 `json:"resourceId"`
	Name               string                 `json:"name"`
	Type               string                 `json:"type"`
	Kind               string                 `json:"kind,omitempty"`
	Namespace          string                 `json:"namespace,omitempty"`
	Status             *ResourceStatus        `json:"status"`
	Configuration      *runtime.RawExtension  `json:"configuration,omitempty"`
	Dependencies       []string               `json:"dependencies,omitempty"`
	Endpoints          []*ResourceEndpoint    `json:"endpoints,omitempty"`
	Metrics            map[string]interface{} `json:"metrics,omitempty"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

// DeployedService represents a service that has been deployed
type DeployedService struct {
	ServiceID          string                 `json:"serviceId"`
	Name               string                 `json:"name"`
	Type               string                 `json:"type"` // ClusterIP, NodePort, LoadBalancer, ExternalName
	Namespace          string                 `json:"namespace,omitempty"`
	Ports              []*ServicePort         `json:"ports,omitempty"`
	Endpoints          []*ServiceEndpoint     `json:"endpoints,omitempty"`
	Status             *ServiceStatus         `json:"status"`
	Configuration      *runtime.RawExtension  `json:"configuration,omitempty"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

// ResourceEndpoint represents an endpoint exposed by a deployed resource
type ResourceEndpoint struct {
	Name               string `json:"name"`
	Protocol           string `json:"protocol"`
	Address            string `json:"address"`
	Port               int32  `json:"port"`
	Path               string `json:"path,omitempty"`
	Scheme             string `json:"scheme,omitempty"`
	Type               string `json:"type"` // HTTP, HTTPS, TCP, UDP, GRPC
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
	Type               string                 `json:"type"`
	ClusterIP          string                 `json:"clusterIP,omitempty"`
	ExternalIPs        []string               `json:"externalIPs,omitempty"`
	LoadBalancerIP     string                 `json:"loadBalancerIP,omitempty"`
	LoadBalancerIngress []string              `json:"loadBalancerIngress,omitempty"`
	Conditions         []ServiceCondition     `json:"conditions,omitempty"`
	Health             string                 `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY
	LastHealthCheck    time.Time              `json:"lastHealthCheck"`
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
	Names            []string          `json:"names,omitempty"`
	Categories       []string          `json:"categories,omitempty"`
	Types            []string          `json:"types,omitempty"`
	Versions         []string          `json:"versions,omitempty"`
	Authors          []string          `json:"authors,omitempty"`
	Keywords         []string          `json:"keywords,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	CreatedAfter     *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore    *time.Time        `json:"createdBefore,omitempty"`
	Limit            int               `json:"limit,omitempty"`
	Offset           int               `json:"offset,omitempty"`
	SortBy           string            `json:"sortBy,omitempty"`
	SortOrder        string            `json:"sortOrder,omitempty"`
}

// DeploymentFilter defines filters for querying deployments
type DeploymentFilter struct {
	Names              []string          `json:"names,omitempty"`
	TemplateIDs        []string          `json:"templateIds,omitempty"`
	ResourcePoolIDs    []string          `json:"resourcePoolIds,omitempty"`
	States             []string          `json:"states,omitempty"`
	Phases             []string          `json:"phases,omitempty"`
	HealthStates       []string          `json:"healthStates,omitempty"`
	ParentDeploymentIDs []string         `json:"parentDeploymentIds,omitempty"`
	CreatedBy          []string          `json:"createdBy,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	CreatedAfter       *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore      *time.Time        `json:"createdBefore,omitempty"`
	Limit              int               `json:"limit,omitempty"`
	Offset             int               `json:"offset,omitempty"`
	SortBy             string            `json:"sortBy,omitempty"`
	SortOrder          string            `json:"sortOrder,omitempty"`
}

// Request types for deployment management operations

// CreateDeploymentTemplateRequest represents a request to create a deployment template
type CreateDeploymentTemplateRequest struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	Version         string                 `json:"version"`
	Category        string                 `json:"category"`
	Type            string                 `json:"type"`
	Content         *runtime.RawExtension  `json:"content"`
	InputSchema     *runtime.RawExtension  `json:"inputSchema,omitempty"`
	OutputSchema    *runtime.RawExtension  `json:"outputSchema,omitempty"`
	Author          string                 `json:"author,omitempty"`
	License         string                 `json:"license,omitempty"`
	Keywords        []string               `json:"keywords,omitempty"`
	Dependencies    []*TemplateDependency  `json:"dependencies,omitempty"`
	Requirements    *ResourceRequirements  `json:"requirements,omitempty"`
	Extensions      map[string]interface{} `json:"extensions,omitempty"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
}

// UpdateDeploymentTemplateRequest represents a request to update a deployment template
type UpdateDeploymentTemplateRequest struct {
	Name            *string                `json:"name,omitempty"`
	Description     *string                `json:"description,omitempty"`
	Version         *string                `json:"version,omitempty"`
	Content         *runtime.RawExtension  `json:"content,omitempty"`
	InputSchema     *runtime.RawExtension  `json:"inputSchema,omitempty"`
	OutputSchema    *runtime.RawExtension  `json:"outputSchema,omitempty"`
	Keywords        []string               `json:"keywords,omitempty"`
	Dependencies    []*TemplateDependency  `json:"dependencies,omitempty"`
	Requirements    *ResourceRequirements  `json:"requirements,omitempty"`
	Extensions      map[string]interface{} `json:"extensions,omitempty"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
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
	DryRun             bool                   `json:"dryRun,omitempty"`
	Timeout            time.Duration          `json:"timeout,omitempty"`
	RetryPolicy        *RetryPolicy           `json:"retryPolicy,omitempty"`
	RollbackPolicy     *RollbackPolicy        `json:"rollbackPolicy,omitempty"`
}

// UpdateDeploymentRequest represents a request to update a deployment
type UpdateDeploymentRequest struct {
	Description        *string                `json:"description,omitempty"`
	InputParameters    *runtime.RawExtension  `json:"inputParameters,omitempty"`
	Extensions         map[string]interface{} `json:"extensions,omitempty"`
	Metadata           map[string]string      `json:"metadata,omitempty"`
	
	// Update options
	UpdateStrategy     *UpdateStrategy        `json:"updateStrategy,omitempty"`
	Timeout            time.Duration          `json:"timeout,omitempty"`
	RetryPolicy        *RetryPolicy           `json:"retryPolicy,omitempty"`
}

// RetryPolicy defines retry behavior for deployment operations
type RetryPolicy struct {
	MaxRetries     int           `json:"maxRetries"`
	RetryDelay     time.Duration `json:"retryDelay"`
	BackoffFactor  float64       `json:"backoffFactor"`
	MaxRetryDelay  time.Duration `json:"maxRetryDelay"`
	RetryConditions []string      `json:"retryConditions,omitempty"`
}

// RollbackPolicy defines rollback behavior for failed deployments
type RollbackPolicy struct {
	Enabled        bool          `json:"enabled"`
	AutoRollback   bool          `json:"autoRollback"`
	RollbackDelay  time.Duration `json:"rollbackDelay,omitempty"`
	MaxRollbacks   int           `json:"maxRollbacks"`
	RollbackConditions []string  `json:"rollbackConditions,omitempty"`
}

// UpdateStrategy defines how deployment updates should be performed
type UpdateStrategy struct {
	Type           string        `json:"type"` // RECREATE, ROLLING_UPDATE, BLUE_GREEN, CANARY
	MaxUnavailable string        `json:"maxUnavailable,omitempty"`
	MaxSurge       string        `json:"maxSurge,omitempty"`
	Timeout        time.Duration `json:"timeout,omitempty"`
	PauseConditions []string     `json:"pauseConditions,omitempty"`
}

// System Information Model

// SystemInfo represents O2 IMS system information
type SystemInfo struct {
	Name                   string                 `json:"name"`
	Description            string                 `json:"description"`
	Version                string                 `json:"version"`
	APIVersions            []string               `json:"apiVersions"`
	SupportedResourceTypes []string               `json:"supportedResourceTypes"`
	Extensions             map[string]interface{} `json:"extensions"`
	Timestamp              time.Time              `json:"timestamp"`
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
)