// Package o2 - Helper types for O2 IMS implementation.

// This file contains additional types referenced by the main O2 IMS components.

package o2

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
)

// Additional types referenced by ResourcePoolManager and other components.

// DeploymentTemplateFilter defines filters for deployment template queries.

type DeploymentTemplateFilter struct {
	Names []string `json:"names,omitempty"`

	Categories []string `json:"categories,omitempty"`

	Types []string `json:"types,omitempty"`

	Versions []string `json:"versions,omitempty"`

	Authors []string `json:"authors,omitempty"`

	Keywords []string `json:"keywords,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	CreatedAfter *time.Time `json:"createdAfter,omitempty"`

	CreatedBefore *time.Time `json:"createdBefore,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`

	SortBy string `json:"sortBy,omitempty"`

	SortOrder string `json:"sortOrder,omitempty"`
}

// DeploymentTemplate represents a deployment template following O2 IMS specification.

type DeploymentTemplate struct {
	DeploymentTemplateID string `json:"deploymentTemplateId"`

	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	Version string `json:"version"`

	Provider string `json:"provider,omitempty"`

	Category string `json:"category,omitempty"` // VNF, CNF, PNF, NS

	// Template specifications.

	TemplateSpec *TemplateSpecification `json:"templateSpec"`

	RequiredResources *ResourceRequirements `json:"requiredResources"`

	SupportedParameters []*TemplateParameter `json:"supportedParameters,omitempty"`

	// Validation and compatibility.

	ValidationRules []*ValidationRule `json:"validationRules,omitempty"`

	CompatibilityMatrix []*CompatibilityInfo `json:"compatibilityMatrix,omitempty"`

	// Metadata and lifecycle.

	Tags map[string]string `json:"tags,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`

	Status *TemplateStatus `json:"status"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`

	CreatedBy string `json:"createdBy,omitempty"`

	UpdatedBy string `json:"updatedBy,omitempty"`
}

// TemplateSpecification defines the deployment template specification.

type TemplateSpecification struct {
	Type string `json:"type"` // HEAT, HELM, KUBERNETES, TERRAFORM

	Content *runtime.RawExtension `json:"content"`

	ContentType string `json:"contentType"` // yaml, json, zip

	MainTemplate string `json:"mainTemplate,omitempty"`

	NestedTemplates map[string]*runtime.RawExtension `json:"nestedTemplates,omitempty"`

	// Deployment configuration.

	DeploymentOptions *DeploymentOptions `json:"deploymentOptions,omitempty"`

	Hooks []*DeploymentHook `json:"hooks,omitempty"`

	// Resource mappings.

	ResourceMappings []*ResourceMapping `json:"resourceMappings,omitempty"`

	NetworkMappings []*NetworkMapping `json:"networkMappings,omitempty"`

	StorageMappings []*StorageMapping `json:"storageMappings,omitempty"`

	// Dependencies.

	Dependencies []*TemplateDependency `json:"dependencies,omitempty"`

	Prerequisites []*Prerequisite `json:"prerequisites,omitempty"`
}

// DeploymentOptions defines options for template deployment.

type DeploymentOptions struct {
	Timeout time.Duration `json:"timeout"`

	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`

	RollbackPolicy *RollbackPolicy `json:"rollbackPolicy,omitempty"`

	ScalingPolicy *ScalingPolicy `json:"scalingPolicy,omitempty"`

	MonitoringConfig *MonitoringConfig `json:"monitoringConfig,omitempty"`

	SecurityConfig *SecurityConfiguration `json:"securityConfig,omitempty"`
}

// ResourceRequirements defines resource requirements for deployment templates.

type ResourceRequirements struct {
	MinCPU string `json:"minCpu,omitempty"`

	MinMemory string `json:"minMemory,omitempty"`

	MinStorage string `json:"minStorage,omitempty"`

	MinNodes int `json:"minNodes,omitempty"`

	MaxCPU string `json:"maxCpu,omitempty"`

	MaxMemory string `json:"maxMemory,omitempty"`

	MaxStorage string `json:"maxStorage,omitempty"`

	MaxNodes int `json:"maxNodes,omitempty"`

	RequiredFeatures []string `json:"requiredFeatures,omitempty"`

	SupportedArch []string `json:"supportedArchitectures,omitempty"`

	NetworkRequirements []*NetworkRequirement `json:"networkRequirements,omitempty"`

	StorageRequirements []*StorageRequirement `json:"storageRequirements,omitempty"`

	AcceleratorReq []*AcceleratorRequirement `json:"acceleratorRequirements,omitempty"`
}

// Deployment represents a deployment instance created from a template.

type Deployment struct {
	DeploymentManagerID string `json:"deploymentManagerId"`

	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	ParentDeploymentID string `json:"parentDeploymentId,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`

	// Deployment specification.

	TemplateID string `json:"templateId"`

	TemplateVersion string `json:"templateVersion,omitempty"`

	InputParameters *runtime.RawExtension `json:"inputParameters,omitempty"`

	OutputValues *runtime.RawExtension `json:"outputValues,omitempty"`

	ResourcePoolID string `json:"resourcePoolId"`

	// Deployment status and lifecycle.

	Status *DeploymentStatus `json:"status"`

	Resources []*DeployedResource `json:"resources,omitempty"`

	Services []*DeployedService `json:"services,omitempty"`

	// Lifecycle information.

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`

	CreatedBy string `json:"createdBy,omitempty"`

	UpdatedBy string `json:"updatedBy,omitempty"`
}

// DeploymentFilter defines filters for deployment queries.

type DeploymentFilter struct {
	Names []string `json:"names,omitempty"`

	TemplateIDs []string `json:"templateIds,omitempty"`

	ResourcePoolIDs []string `json:"resourcePoolIds,omitempty"`

	States []string `json:"states,omitempty"`

	Phases []string `json:"phases,omitempty"`

	HealthStates []string `json:"healthStates,omitempty"`

	ParentDeploymentIDs []string `json:"parentDeploymentIds,omitempty"`

	CreatedBy []string `json:"createdBy,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	CreatedAfter *time.Time `json:"createdAfter,omitempty"`

	CreatedBefore *time.Time `json:"createdBefore,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`

	SortBy string `json:"sortBy,omitempty"`

	SortOrder string `json:"sortOrder,omitempty"`
}

// CreateDeploymentRequest represents a request to create a deployment.

type CreateDeploymentRequest struct {
	Name string `json:"name"`

	Description string `json:"description,omitempty"`

	TemplateID string `json:"templateId"`

	TemplateVersion string `json:"templateVersion,omitempty"`

	ResourcePoolID string `json:"resourcePoolId"`

	InputParameters *runtime.RawExtension `json:"inputParameters,omitempty"`

	ParentDeploymentID string `json:"parentDeploymentId,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	// Deployment options.

	DryRun bool `json:"dryRun,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`

	RollbackPolicy *RollbackPolicy `json:"rollbackPolicy,omitempty"`
}

// UpdateDeploymentRequest represents a request to update a deployment.

type UpdateDeploymentRequest struct {
	Description *string `json:"description,omitempty"`

	InputParameters *runtime.RawExtension `json:"inputParameters,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	// Update options.

	UpdateStrategy *UpdateStrategy `json:"updateStrategy,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// Subscription represents an event subscription in O2 IMS.

type Subscription = models.Subscription

// SubscriptionFilter defines filters for subscription queries.

type SubscriptionFilter struct {
	Names []string `json:"names,omitempty"`

	EventTypes []string `json:"eventTypes,omitempty"`

	States []string `json:"states,omitempty"`

	ResourceTypes []string `json:"resourceTypes,omitempty"`

	ResourcePoolIDs []string `json:"resourcePoolIds,omitempty"`

	CreatedBy []string `json:"createdBy,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	CreatedAfter *time.Time `json:"createdAfter,omitempty"`

	CreatedBefore *time.Time `json:"createdBefore,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`

	SortBy string `json:"sortBy,omitempty"`

	SortOrder string `json:"sortOrder,omitempty"`
}

// InfrastructureEvent represents an infrastructure event.

type InfrastructureEvent = models.InfrastructureEvent

// Infrastructure monitoring and health types.

// ResourceHealth represents the health status of a resource.

type ResourceHealth struct {
	ResourceID string `json:"resourceId"`

	Status string `json:"status"` // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN

	LastCheck time.Time `json:"lastCheck"`

	Checks []*HealthCheckResult `json:"checks,omitempty"`

	Metrics map[string]interface{} `json:"metrics,omitempty"`

	Alerts []interface{} `json:"alerts,omitempty"`

	OverallScore float64 `json:"overallScore,omitempty"`
}

// HealthCheckResult represents the result of a health check.

type HealthCheckResult struct {
	CheckName string `json:"checkName"`

	Status string `json:"status"`

	Message string `json:"message,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Duration time.Duration `json:"duration"`

	Details map[string]interface{} `json:"details,omitempty"`
}

// Alarm represents an alarm in the system.

type Alarm struct {
	AlarmID string `json:"alarmId"`

	ResourceID string `json:"resourceId"`

	AlarmType string `json:"alarmType"`

	Severity string `json:"severity"`

	Status string `json:"status"` // ACTIVE, CLEARED, ACKNOWLEDGED

	Message string `json:"message"`

	Description string `json:"description,omitempty"`

	Source string `json:"source"`

	RaisedAt time.Time `json:"raisedAt"`

	ClearedAt *time.Time `json:"clearedAt,omitempty"`

	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`

	AcknowledgedBy string `json:"acknowledgedBy,omitempty"`

	AdditionalInfo map[string]interface{} `json:"additionalInfo,omitempty"`
}

// AlarmFilter defines filters for alarm queries.

type AlarmFilter struct {
	ResourceIDs []string `json:"resourceIds,omitempty"`

	AlarmTypes []string `json:"alarmTypes,omitempty"`

	Severities []string `json:"severities,omitempty"`

	Status []string `json:"status,omitempty"`

	Sources []string `json:"sources,omitempty"`

	RaisedAfter *time.Time `json:"raisedAfter,omitempty"`

	RaisedBefore *time.Time `json:"raisedBefore,omitempty"`

	ClearedAfter *time.Time `json:"clearedAfter,omitempty"`

	ClearedBefore *time.Time `json:"clearedBefore,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`

	SortBy string `json:"sortBy,omitempty"`

	SortOrder string `json:"sortOrder,omitempty"`
}

// MetricsData represents collected metrics.

type MetricsData struct {
	ResourceID string `json:"resourceId"`

	Timestamp time.Time `json:"timestamp"`

	Metrics map[string]interface{} `json:"metrics"`

	Labels map[string]string `json:"labels,omitempty"`

	CollectorID string `json:"collectorId,omitempty"`
}

// MetricsFilter defines filters for metrics queries.

type MetricsFilter struct {
	MetricNames []string `json:"metricNames,omitempty"`

	StartTime *time.Time `json:"startTime,omitempty"`

	EndTime *time.Time `json:"endTime,omitempty"`

	Interval string `json:"interval,omitempty"`

	Aggregation string `json:"aggregation,omitempty"` // avg, min, max, sum, count

	Labels map[string]string `json:"labels,omitempty"`

	Limit int `json:"limit,omitempty"`
}

// Infrastructure discovery and inventory types.

// InfrastructureDiscovery represents discovered infrastructure.

type InfrastructureDiscovery struct {
	Provider providers.CloudProvider `json:"provider"`

	Region string `json:"region,omitempty"`

	DiscoveryID string `json:"discoveryId"`

	Timestamp time.Time `json:"timestamp"`

	ResourcePools []*models.ResourcePool `json:"resourcePools,omitempty"`

	Resources []*models.Resource `json:"resources,omitempty"`

	Capabilities []string `json:"capabilities,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// InventoryUpdate represents an update to the infrastructure inventory.

type InventoryUpdate struct {
	UpdateID string `json:"updateId"`

	UpdateType string `json:"updateType"` // ADD, UPDATE, DELETE

	ResourceID string `json:"resourceId"`

	ResourceType string `json:"resourceType"`

	Timestamp time.Time `json:"timestamp"`

	Changes map[string]interface{} `json:"changes,omitempty"`

	Source string `json:"source,omitempty"`
}

// CapacityPrediction represents predicted capacity requirements.

type CapacityPrediction struct {
	ResourcePoolID string `json:"resourcePoolId"`

	PredictionID string `json:"predictionId"`

	Horizon time.Duration `json:"horizon"`

	PredictedCapacity *models.ResourceCapacity `json:"predictedCapacity"`

	Confidence float64 `json:"confidence"`

	Factors []string `json:"factors,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Model string `json:"model,omitempty"`
}

// Resource management operation types.

// ProvisionResourceRequest represents a request to provision a resource.

type ProvisionResourceRequest struct {
	Name string `json:"name"`

	ResourceType string `json:"resourceType"`

	ResourcePoolID string `json:"resourcePoolId"`

	Provider string `json:"provider"`

	Configuration *runtime.RawExtension `json:"configuration"`

	Requirements *ResourceRequirements `json:"requirements,omitempty"`

	Tags map[string]string `json:"tags,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ScaleResourceRequest represents a request to scale a resource.

type ScaleResourceRequest struct {
	ScaleType string `json:"scaleType"` // UP, DOWN, OUT, IN

	TargetSize int `json:"targetSize,omitempty"`

	TargetReplicas int `json:"targetReplicas,omitempty"`

	Percentage int `json:"percentage,omitempty"`

	Resources *ResourceRequirements `json:"resources,omitempty"`

	Options map[string]interface{} `json:"options,omitempty"`
}

// MigrateResourceRequest represents a request to migrate a resource.

type MigrateResourceRequest struct {
	TargetPoolID string `json:"targetPoolId"`

	SourceProvider string `json:"sourceProvider"`

	TargetProvider string `json:"targetProvider"`

	MigrationType string `json:"migrationType"` // LIVE, COLD, HOT

	PreservePersistence bool `json:"preservePersistence"`

	Options map[string]interface{} `json:"options,omitempty"`
}

// BackupResourceRequest represents a request to backup a resource.

type BackupResourceRequest struct {
	BackupType string `json:"backupType"` // FULL, INCREMENTAL, DIFFERENTIAL

	StorageLocation string `json:"storageLocation,omitempty"`

	Retention time.Duration `json:"retention,omitempty"`

	Compression bool `json:"compression"`

	Options map[string]interface{} `json:"options,omitempty"`
}

// BackupInfo represents information about a backup.

type BackupInfo struct {
	BackupID string `json:"backupId"`

	ResourceID string `json:"resourceId"`

	BackupType string `json:"backupType"`

	Status string `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED

	Size int64 `json:"size,omitempty"`

	Location string `json:"location,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	CompletedAt *time.Time `json:"completedAt,omitempty"`

	ExpiresAt *time.Time `json:"expiresAt,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceHistoryEntry represents a history entry for a resource.

type ResourceHistoryEntry struct {
	EntryID string `json:"entryId"`

	ResourceID string `json:"resourceId"`

	EventType string `json:"eventType"` // CREATED, UPDATED, DELETED, STATE_CHANGED

	Timestamp time.Time `json:"timestamp"`

	Changes map[string]interface{} `json:"changes,omitempty"`

	User string `json:"user,omitempty"`

	Reason string `json:"reason,omitempty"`

	PreviousState *runtime.RawExtension `json:"previousState,omitempty"`

	CurrentState *runtime.RawExtension `json:"currentState,omitempty"`
}

// Monitoring service types.

// HealthCallback defines a callback function for health monitoring.

type HealthCallback func(resourceID string, health *ResourceHealth) error

// MetricsCollectionConfig configures metrics collection.

type MetricsCollectionConfig struct {
	MetricNames []string `json:"metricNames"`

	Interval time.Duration `json:"interval"`

	Aggregation string `json:"aggregation,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	StorageRetention time.Duration `json:"storageRetention,omitempty"`
}

// Supporting types for deployment templates and resources.

// TemplateParameter represents a parameter for a deployment template.

type TemplateParameter struct {
	Name string `json:"name"`

	Type string `json:"type"` // string, integer, boolean, array, object

	Description string `json:"description,omitempty"`

	Required bool `json:"required"`

	DefaultValue *runtime.RawExtension `json:"defaultValue,omitempty"`

	Constraints *ParameterConstraints `json:"constraints,omitempty"`
}

// ParameterConstraints defines constraints for template parameters.

type ParameterConstraints struct {
	MinValue *float64 `json:"minValue,omitempty"`

	MaxValue *float64 `json:"maxValue,omitempty"`

	MinLength *int `json:"minLength,omitempty"`

	MaxLength *int `json:"maxLength,omitempty"`

	Pattern string `json:"pattern,omitempty"`

	AllowedValues []interface{} `json:"allowedValues,omitempty"`

	Validation map[string]interface{} `json:"validation,omitempty"`
}

// ValidationRule defines validation rules for templates.

type ValidationRule struct {
	RuleID string `json:"ruleId"`

	Name string `json:"name"`

	Type string `json:"type"` // SYNTAX, SEMANTIC, RESOURCE, POLICY

	Expression string `json:"expression"`

	Message string `json:"message,omitempty"`

	Severity string `json:"severity"` // ERROR, WARNING, INFO

	Context map[string]interface{} `json:"context,omitempty"`
}

// CompatibilityInfo defines compatibility information.

type CompatibilityInfo struct {
	Platform string `json:"platform"`

	Versions []string `json:"versions"`

	Dependencies []string `json:"dependencies,omitempty"`

	Constraints []string `json:"constraints,omitempty"`
}

// TemplateStatus represents the status of a deployment template.

type TemplateStatus struct {
	State string `json:"state"` // DRAFT, ACTIVE, DEPRECATED, ARCHIVED

	ValidationStatus string `json:"validationStatus"` // PENDING, VALID, INVALID

	LastValidated *time.Time `json:"lastValidated,omitempty"`

	ValidationErrors []string `json:"validationErrors,omitempty"`

	Usage *TemplateUsageStats `json:"usage,omitempty"`

	Health string `json:"health,omitempty"`
}

// TemplateUsageStats provides usage statistics for templates.

type TemplateUsageStats struct {
	TotalDeployments int `json:"totalDeployments"`

	ActiveDeployments int `json:"activeDeployments"`

	FailedDeployments int `json:"failedDeployments"`

	SuccessRate float64 `json:"successRate"`

	AverageDeployTime time.Duration `json:"averageDeployTime"`

	LastUsed *time.Time `json:"lastUsed,omitempty"`
}

// DeploymentStatus represents the status of a deployment.

type DeploymentStatus struct {
	State string `json:"state"` // PENDING, RUNNING, FAILED, SUCCEEDED, DELETING

	Phase string `json:"phase"` // CREATING, UPDATING, SCALING, TERMINATING

	Health string `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN

	Progress *DeploymentProgress `json:"progress,omitempty"`

	Conditions []DeploymentCondition `json:"conditions,omitempty"`

	ErrorMessage string `json:"errorMessage,omitempty"`

	Events []*DeploymentEvent `json:"events,omitempty"`

	LastStateChange time.Time `json:"lastStateChange"`

	LastHealthCheck time.Time `json:"lastHealthCheck"`

	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// DeploymentProgress represents the progress of a deployment operation.

type DeploymentProgress struct {
	TotalSteps int32 `json:"totalSteps"`

	CompletedSteps int32 `json:"completedSteps"`

	CurrentStep string `json:"currentStep,omitempty"`

	PercentComplete float64 `json:"percentComplete"`

	EstimatedTimeRemaining time.Duration `json:"estimatedTimeRemaining,omitempty"`
}

// DeploymentCondition represents a condition of the deployment.

type DeploymentCondition struct {
	Type string `json:"type"`

	Status string `json:"status"` // True, False, Unknown

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`

	LastTransitionTime time.Time `json:"lastTransitionTime"`

	LastUpdateTime time.Time `json:"lastUpdateTime,omitempty"`
}

// DeploymentEvent represents an event that occurred during deployment lifecycle.

type DeploymentEvent struct {
	ID string `json:"id"`

	Type string `json:"type"` // NORMAL, WARNING, ERROR

	Reason string `json:"reason"`

	Message string `json:"message"`

	Component string `json:"component,omitempty"`

	Source string `json:"source,omitempty"`

	FirstTimestamp time.Time `json:"firstTimestamp"`

	LastTimestamp time.Time `json:"lastTimestamp"`

	Count int32 `json:"count"`

	AdditionalData map[string]interface{} `json:"additionalData,omitempty"`
}

// DeployedResource represents a resource that has been deployed.

type DeployedResource struct {
	ResourceID string `json:"resourceId"`

	Name string `json:"name"`

	Type string `json:"type"`

	Kind string `json:"kind,omitempty"`

	Namespace string `json:"namespace,omitempty"`

	Status *ResourceStatus `json:"status"`

	Configuration *runtime.RawExtension `json:"configuration,omitempty"`

	Dependencies []string `json:"dependencies,omitempty"`

	Endpoints []*ResourceEndpoint `json:"endpoints,omitempty"`

	Metrics map[string]interface{} `json:"metrics,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// ResourceStatus represents the status of a resource.

type ResourceStatus struct {
	State string `json:"state"` // PENDING, ACTIVE, INACTIVE, FAILED, TERMINATING

	Health string `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY, UNKNOWN

	Message string `json:"message,omitempty"`

	Reason string `json:"reason,omitempty"`

	LastTransition time.Time `json:"lastTransition"`

	LastHealthCheck time.Time `json:"lastHealthCheck"`

	Conditions []ResourceCondition `json:"conditions,omitempty"`

	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// ResourceCondition represents a condition of the resource.

type ResourceCondition struct {
	Type string `json:"type"`

	Status string `json:"status"` // True, False, Unknown

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`

	LastTransitionTime time.Time `json:"lastTransitionTime"`

	LastUpdateTime time.Time `json:"lastUpdateTime,omitempty"`
}

// ResourceEndpoint represents an endpoint exposed by a deployed resource.

type ResourceEndpoint struct {
	Name string `json:"name"`

	Protocol string `json:"protocol"`

	Address string `json:"address"`

	Port int32 `json:"port"`

	Path string `json:"path,omitempty"`

	Scheme string `json:"scheme,omitempty"`

	Type string `json:"type"` // HTTP, HTTPS, TCP, UDP, GRPC
}

// DeployedService represents a service that has been deployed.

type DeployedService struct {
	ServiceID string `json:"serviceId"`

	Name string `json:"name"`

	Type string `json:"type"` // ClusterIP, NodePort, LoadBalancer, ExternalName

	Namespace string `json:"namespace,omitempty"`

	Ports []*ServicePort `json:"ports,omitempty"`

	Endpoints []*ServiceEndpoint `json:"endpoints,omitempty"`

	Status *ServiceStatus `json:"status"`

	Configuration *runtime.RawExtension `json:"configuration,omitempty"`

	CreatedAt time.Time `json:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt"`
}

// ServicePort represents a port exposed by a service.

type ServicePort struct {
	Name string `json:"name,omitempty"`

	Protocol string `json:"protocol"`

	Port int32 `json:"port"`

	TargetPort string `json:"targetPort,omitempty"`

	NodePort int32 `json:"nodePort,omitempty"`
}

// ServiceEndpoint represents an endpoint of a service.

type ServiceEndpoint struct {
	Address string `json:"address"`

	Port int32 `json:"port"`

	Protocol string `json:"protocol"`

	Ready bool `json:"ready"`

	Conditions map[string]string `json:"conditions,omitempty"`
}

// ServiceStatus represents the status of a service.

type ServiceStatus struct {
	Type string `json:"type"`

	ClusterIP string `json:"clusterIP,omitempty"`

	ExternalIPs []string `json:"externalIPs,omitempty"`

	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`

	LoadBalancerIngress []string `json:"loadBalancerIngress,omitempty"`

	Conditions []ServiceCondition `json:"conditions,omitempty"`

	Health string `json:"health"` // HEALTHY, DEGRADED, UNHEALTHY

	LastHealthCheck time.Time `json:"lastHealthCheck"`
}

// ServiceCondition represents a condition of the service.

type ServiceCondition struct {
	Type string `json:"type"`

	Status string `json:"status"`

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`

	LastTransitionTime time.Time `json:"lastTransitionTime"`
}

// Supporting types for template specifications.

// DeploymentHook represents hooks for deployment lifecycle events.

type DeploymentHook struct {
	Name string `json:"name"`

	Phase string `json:"phase"` // PRE_DEPLOY, POST_DEPLOY, PRE_UPDATE, POST_UPDATE, PRE_DELETE, POST_DELETE

	Type string `json:"type"` // SCRIPT, HTTP, KUBERNETES_JOB

	Config *runtime.RawExtension `json:"config"`

	Timeout time.Duration `json:"timeout,omitempty"`

	OnFailure string `json:"onFailure,omitempty"` // CONTINUE, ABORT, RETRY

	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// ResourceMapping defines how template resources map to infrastructure.

type ResourceMapping struct {
	TemplateResource string `json:"templateResource"`

	InfrastructureType string `json:"infrastructureType"`

	Mapping map[string]interface{} `json:"mapping"`

	Constraints []string `json:"constraints,omitempty"`
}

// NetworkMapping defines network resource mappings.

type NetworkMapping struct {
	NetworkName string `json:"networkName"`

	NetworkType string `json:"networkType"` // INTERNAL, EXTERNAL, MANAGEMENT

	CIDR string `json:"cidr,omitempty"`

	VLAN *int `json:"vlan,omitempty"`

	Properties map[string]interface{} `json:"properties,omitempty"`
}

// StorageMapping defines storage resource mappings.

type StorageMapping struct {
	StorageName string `json:"storageName"`

	StorageType string `json:"storageType"` // BLOCK, OBJECT, FILE

	Size string `json:"size"`

	Performance string `json:"performance,omitempty"` // HIGH, MEDIUM, LOW

	Properties map[string]interface{} `json:"properties,omitempty"`
}

// TemplateDependency represents a dependency of a deployment template.

type TemplateDependency struct {
	Name string `json:"name"`

	Version string `json:"version"`

	Type string `json:"type"` // TEMPLATE, RESOURCE, SERVICE

	Required bool `json:"required"`

	Description string `json:"description,omitempty"`
}

// Prerequisite represents a prerequisite for template deployment.

type Prerequisite struct {
	Type string `json:"type"` // RESOURCE, SERVICE, CAPABILITY, POLICY

	Name string `json:"name"`

	Version string `json:"version,omitempty"`

	Description string `json:"description,omitempty"`

	Validation *runtime.RawExtension `json:"validation,omitempty"`

	Required bool `json:"required"`
}

// ScalingPolicy defines scaling behavior.

type ScalingPolicy struct {
	Enabled bool `json:"enabled"`

	MinInstances int `json:"minInstances"`

	MaxInstances int `json:"maxInstances"`

	TargetCPU *int `json:"targetCpu,omitempty"`

	TargetMemory *int `json:"targetMemory,omitempty"`

	ScaleUpPolicy *ScalePolicy `json:"scaleUpPolicy,omitempty"`

	ScaleDownPolicy *ScalePolicy `json:"scaleDownPolicy,omitempty"`

	Metrics []string `json:"metrics,omitempty"`
}

// ScalePolicy defines scaling policy parameters.

type ScalePolicy struct {
	Type string `json:"type"` // PODS, PERCENT

	Value int `json:"value"`

	PeriodSeconds int `json:"periodSeconds"`

	StabilizationWindow int `json:"stabilizationWindow"`
}

// AlertRule defines an alert rule.

type AlertRule struct {
	Name string `json:"name"`

	Expression string `json:"expression"`

	Duration time.Duration `json:"duration"`

	Severity string `json:"severity"`

	Labels map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
}

// SecurityConfiguration defines security settings.

type SecurityConfiguration struct {
	NetworkPolicies []*NetworkPolicy `json:"networkPolicies,omitempty"`

	PodSecurityPolicy *PodSecurityPolicy `json:"podSecurityPolicy,omitempty"`

	ServiceAccount *ServiceAccountConfig `json:"serviceAccount,omitempty"`

	RBAC *RBACConfig `json:"rbac,omitempty"`

	SecretManagement *SecretConfig `json:"secretManagement,omitempty"`
}

// NetworkPolicy defines network security policies.

type NetworkPolicy struct {
	Name string `json:"name"`

	PodSelector map[string]string `json:"podSelector"`

	Ingress []*NetworkPolicyRule `json:"ingress,omitempty"`

	Egress []*NetworkPolicyRule `json:"egress,omitempty"`
}

// NetworkPolicyPeer defines network policy peer.

type NetworkPolicyPeer struct {
	PodSelector map[string]string `json:"podSelector,omitempty"`

	NamespaceSelector map[string]string `json:"namespaceSelector,omitempty"`

	IPBlock *IPBlock `json:"ipBlock,omitempty"`
}

// NetworkPolicyPort defines network policy port.

type NetworkPolicyPort struct {
	Protocol string `json:"protocol,omitempty"`

	Port string `json:"port,omitempty"`
}

// IPBlock defines IP block for network policies.

type IPBlock struct {
	CIDR string `json:"cidr"`

	Except []string `json:"except,omitempty"`
}

// PodSecurityPolicy defines pod security policies.

type PodSecurityPolicy struct {
	RunAsUser *int64 `json:"runAsUser,omitempty"`

	RunAsGroup *int64 `json:"runAsGroup,omitempty"`

	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`

	ReadOnlyRootFS *bool `json:"readOnlyRootFS,omitempty"`

	AllowedCapabilities []string `json:"allowedCapabilities,omitempty"`

	RequiredDropCapabilities []string `json:"requiredDropCapabilities,omitempty"`
}

// ServiceAccountConfig defines service account configuration.

type ServiceAccountConfig struct {
	Name string `json:"name"`

	Annotations map[string]string `json:"annotations,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`
}

// RBACConfig defines RBAC configuration.

type RBACConfig struct {
	Roles []*Role `json:"roles,omitempty"`

	ClusterRoles []*ClusterRole `json:"clusterRoles,omitempty"`

	RoleBindings []*RoleBinding `json:"roleBindings,omitempty"`
}

// Role defines a role.

type Role struct {
	Name string `json:"name"`

	Rules []*RoleRule `json:"rules"`
}

// ClusterRole defines a cluster role.

type ClusterRole struct {
	Name string `json:"name"`

	Rules []*RoleRule `json:"rules"`
}

// RoleRule defines role rules.

type RoleRule struct {
	APIGroups []string `json:"apiGroups"`

	Resources []string `json:"resources"`

	Verbs []string `json:"verbs"`
}

// RoleBinding defines role bindings.

type RoleBinding struct {
	Name string `json:"name"`

	RoleRef *RoleRef `json:"roleRef"`

	Subjects []*Subject `json:"subjects"`
}

// RoleRef defines role reference.

type RoleRef struct {
	Kind string `json:"kind"`

	Name string `json:"name"`

	APIGroup string `json:"apiGroup"`
}

// Subject defines RBAC subject.

type Subject struct {
	Kind string `json:"kind"`

	Name string `json:"name"`

	Namespace string `json:"namespace,omitempty"`
}

// SecretConfig defines secret management configuration.

type SecretConfig struct {
	Secrets []*SecretSpec `json:"secrets,omitempty"`

	Provider string `json:"provider,omitempty"` // KUBERNETES, VAULT, AWS_SECRETS_MANAGER
}

// SecretSpec defines secret specification.

type SecretSpec struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Data map[string]string `json:"data,omitempty"`

	StringData map[string]string `json:"stringData,omitempty"`
}

// RetryPolicy defines retry behavior for operations.

type RetryPolicy struct {
	MaxRetries int `json:"maxRetries"`

	RetryDelay time.Duration `json:"retryDelay"`

	BackoffFactor float64 `json:"backoffFactor"`

	MaxRetryDelay time.Duration `json:"maxRetryDelay"`

	RetryConditions []string `json:"retryConditions,omitempty"`
}

// RollbackPolicy defines rollback behavior for failed operations.

type RollbackPolicy struct {
	Enabled bool `json:"enabled"`

	AutoRollback bool `json:"autoRollback"`

	RollbackDelay time.Duration `json:"rollbackDelay,omitempty"`

	MaxRollbacks int `json:"maxRollbacks"`

	RollbackConditions []string `json:"rollbackConditions,omitempty"`
}

// UpdateStrategy defines how updates should be performed.

type UpdateStrategy struct {
	Type string `json:"type"` // RECREATE, ROLLING_UPDATE, BLUE_GREEN, CANARY

	MaxUnavailable string `json:"maxUnavailable,omitempty"`

	MaxSurge string `json:"maxSurge,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	PauseConditions []string `json:"pauseConditions,omitempty"`
}

// Resource requirement types.

// NetworkRequirement defines network requirements.

type NetworkRequirement struct {
	Name string `json:"name"`

	Type string `json:"type"` // INTERNAL, EXTERNAL, MANAGEMENT

	Bandwidth string `json:"bandwidth,omitempty"`

	Latency time.Duration `json:"latency,omitempty"`

	Properties map[string]interface{} `json:"properties,omitempty"`
}

// StorageRequirement defines storage requirements.

type StorageRequirement struct {
	Name string `json:"name"`

	Type string `json:"type"` // BLOCK, OBJECT, FILE

	Size string `json:"size"`

	IOPS *int `json:"iops,omitempty"`

	Throughput string `json:"throughput,omitempty"`

	AccessModes []string `json:"accessModes,omitempty"`

	StorageClass string `json:"storageClass,omitempty"`

	Properties map[string]interface{} `json:"properties,omitempty"`
}

// AcceleratorRequirement defines accelerator requirements (GPU, FPGA, etc.).

type AcceleratorRequirement struct {
	Type string `json:"type"` // GPU, FPGA, TPU

	Count int `json:"count"`

	Model string `json:"model,omitempty"`

	Memory string `json:"memory,omitempty"`

	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ResourceSpec defines resource specifications for provisioning.

type ResourceSpec struct {
	CPU string `json:"cpu,omitempty"`

	Memory string `json:"memory,omitempty"`

	Storage string `json:"storage,omitempty"`

	GPU string `json:"gpu,omitempty"`
}

// Constants for resource lifecycle operations.

// Scale types.

const (

	// ScaleTypeHorizontal holds scaletypehorizontal value.

	ScaleTypeHorizontal = "horizontal"

	// ScaleTypeVertical holds scaletypevertical value.

	ScaleTypeVertical = "vertical"
)

// Backup types.

const (

	// BackupTypeFull holds backuptypefull value.

	BackupTypeFull = "full"

	// BackupTypeIncremental holds backuptypeincremental value.

	BackupTypeIncremental = "incremental"

	// BackupTypeDifferential holds backuptypedifferential value.

	BackupTypeDifferential = "differential"
)
