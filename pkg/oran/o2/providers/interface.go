
package providers



import (

	"context"

	"time"

)



// CloudProvider defines the interface for cloud infrastructure providers.

// This abstraction enables support for multiple cloud platforms while maintaining.

// consistent O2 IMS operations across different infrastructure layers.

type CloudProvider interface {

	// Provider identification and metadata.

	GetProviderInfo() *ProviderInfo

	GetSupportedResourceTypes() []string

	GetCapabilities() *ProviderCapabilities



	// Connection and lifecycle management.

	Connect(ctx context.Context) error

	Disconnect(ctx context.Context) error

	HealthCheck(ctx context.Context) error

	Close() error



	// Resource management operations.

	CreateResource(ctx context.Context, req *CreateResourceRequest) (*ResourceResponse, error)

	GetResource(ctx context.Context, resourceID string) (*ResourceResponse, error)

	UpdateResource(ctx context.Context, resourceID string, req *UpdateResourceRequest) (*ResourceResponse, error)

	DeleteResource(ctx context.Context, resourceID string) error

	ListResources(ctx context.Context, filter *ResourceFilter) ([]*ResourceResponse, error)



	// Deployment operations.

	Deploy(ctx context.Context, req *DeploymentRequest) (*DeploymentResponse, error)

	GetDeployment(ctx context.Context, deploymentID string) (*DeploymentResponse, error)

	UpdateDeployment(ctx context.Context, deploymentID string, req *UpdateDeploymentRequest) (*DeploymentResponse, error)

	DeleteDeployment(ctx context.Context, deploymentID string) error

	ListDeployments(ctx context.Context, filter *DeploymentFilter) ([]*DeploymentResponse, error)



	// Scaling operations.

	ScaleResource(ctx context.Context, resourceID string, req *ScaleRequest) error

	GetScalingCapabilities(ctx context.Context, resourceID string) (*ScalingCapabilities, error)



	// Monitoring and metrics.

	GetMetrics(ctx context.Context) (map[string]interface{}, error)

	GetResourceMetrics(ctx context.Context, resourceID string) (map[string]interface{}, error)

	GetResourceHealth(ctx context.Context, resourceID string) (*HealthStatus, error)



	// Network operations.

	CreateNetworkService(ctx context.Context, req *NetworkServiceRequest) (*NetworkServiceResponse, error)

	GetNetworkService(ctx context.Context, serviceID string) (*NetworkServiceResponse, error)

	DeleteNetworkService(ctx context.Context, serviceID string) error

	ListNetworkServices(ctx context.Context, filter *NetworkServiceFilter) ([]*NetworkServiceResponse, error)



	// Storage operations.

	CreateStorageResource(ctx context.Context, req *StorageResourceRequest) (*StorageResourceResponse, error)

	GetStorageResource(ctx context.Context, resourceID string) (*StorageResourceResponse, error)

	DeleteStorageResource(ctx context.Context, resourceID string) error

	ListStorageResources(ctx context.Context, filter *StorageResourceFilter) ([]*StorageResourceResponse, error)



	// Event and notification handling.

	SubscribeToEvents(ctx context.Context, callback EventCallback) error

	UnsubscribeFromEvents(ctx context.Context) error



	// Configuration management.

	ApplyConfiguration(ctx context.Context, config *ProviderConfiguration) error

	GetConfiguration(ctx context.Context) (*ProviderConfiguration, error)

	ValidateConfiguration(ctx context.Context, config *ProviderConfiguration) error

}



// ProviderInfo contains metadata about a cloud provider.

type ProviderInfo struct {

	Name        string            `json:"name"`

	Type        string            `json:"type"`

	Version     string            `json:"version"`

	Description string            `json:"description"`

	Vendor      string            `json:"vendor"`

	Region      string            `json:"region,omitempty"`

	Zone        string            `json:"zone,omitempty"`

	Endpoint    string            `json:"endpoint,omitempty"`

	Tags        map[string]string `json:"tags,omitempty"`

	LastUpdated time.Time         `json:"lastUpdated"`

}



// ProviderCapabilities describes the capabilities of a cloud provider.

type ProviderCapabilities struct {

	// Supported resource types.

	ComputeTypes     []string `json:"computeTypes"`

	StorageTypes     []string `json:"storageTypes"`

	NetworkTypes     []string `json:"networkTypes"`

	AcceleratorTypes []string `json:"acceleratorTypes"`



	// Feature support.

	AutoScaling    bool `json:"autoScaling"`

	LoadBalancing  bool `json:"loadBalancing"`

	Monitoring     bool `json:"monitoring"`

	Logging        bool `json:"logging"`

	Networking     bool `json:"networking"`

	StorageClasses bool `json:"storageClasses"`



	// Scaling capabilities.

	HorizontalPodAutoscaling bool `json:"horizontalPodAutoscaling"`

	VerticalPodAutoscaling   bool `json:"verticalPodAutoscaling"`

	ClusterAutoscaling       bool `json:"clusterAutoscaling"`



	// Multi-tenancy support.

	Namespaces      bool `json:"namespaces"`

	ResourceQuotas  bool `json:"resourceQuotas"`

	NetworkPolicies bool `json:"networkPolicies"`

	RBAC            bool `json:"rbac"`



	// High availability features.

	MultiZone        bool `json:"multiZone"`

	MultiRegion      bool `json:"multiRegion"`

	BackupRestore    bool `json:"backupRestore"`

	DisasterRecovery bool `json:"disasterRecovery"`



	// Security features.

	Encryption       bool `json:"encryption"`

	SecretManagement bool `json:"secretManagement"`

	ImageScanning    bool `json:"imageScanning"`

	PolicyEngine     bool `json:"policyEngine"`



	// Limits and quotas.

	MaxNodes    int `json:"maxNodes,omitempty"`

	MaxPods     int `json:"maxPods,omitempty"`

	MaxServices int `json:"maxServices,omitempty"`

	MaxVolumes  int `json:"maxVolumes,omitempty"`

}



// Resource management request/response types.



// CreateResourceRequest represents a request to create a resource.

type CreateResourceRequest struct {

	Name          string                 `json:"name"`

	Type          string                 `json:"type"`

	Namespace     string                 `json:"namespace,omitempty"`

	Specification map[string]interface{} `json:"specification"`

	Labels        map[string]string      `json:"labels,omitempty"`

	Annotations   map[string]string      `json:"annotations,omitempty"`

	Timeout       time.Duration          `json:"timeout,omitempty"`

	DryRun        bool                   `json:"dryRun,omitempty"`

}



// UpdateResourceRequest represents a request to update a resource.

type UpdateResourceRequest struct {

	Specification map[string]interface{} `json:"specification,omitempty"`

	Labels        map[string]string      `json:"labels,omitempty"`

	Annotations   map[string]string      `json:"annotations,omitempty"`

	Timeout       time.Duration          `json:"timeout,omitempty"`

}



// ResourceResponse represents a resource response.

type ResourceResponse struct {

	ID            string                 `json:"id"`

	Name          string                 `json:"name"`

	Type          string                 `json:"type"`

	Namespace     string                 `json:"namespace,omitempty"`

	Status        string                 `json:"status"`

	Health        string                 `json:"health"`

	Specification map[string]interface{} `json:"specification"`

	Labels        map[string]string      `json:"labels,omitempty"`

	Annotations   map[string]string      `json:"annotations,omitempty"`

	Metrics       map[string]interface{} `json:"metrics,omitempty"`

	Events        []*ResourceEvent       `json:"events,omitempty"`

	CreatedAt     time.Time              `json:"createdAt"`

	UpdatedAt     time.Time              `json:"updatedAt"`

}



// ResourceEvent represents an event related to a resource.

type ResourceEvent struct {

	Type      string                 `json:"type"`

	Reason    string                 `json:"reason"`

	Message   string                 `json:"message"`

	Source    string                 `json:"source"`

	Timestamp time.Time              `json:"timestamp"`

	Metadata  map[string]interface{} `json:"metadata,omitempty"`

}



// ResourceFilter defines filters for resource queries.

type ResourceFilter struct {

	Names       []string          `json:"names,omitempty"`

	Types       []string          `json:"types,omitempty"`

	Namespaces  []string          `json:"namespaces,omitempty"`

	Statuses    []string          `json:"statuses,omitempty"`

	Labels      map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	Limit       int               `json:"limit,omitempty"`

	Offset      int               `json:"offset,omitempty"`

}



// Deployment request/response types.



// DeploymentRequest represents a deployment request.

type DeploymentRequest struct {

	Name         string                 `json:"name"`

	Template     string                 `json:"template"`

	TemplateType string                 `json:"templateType"` // helm, kubernetes, terraform

	Parameters   map[string]interface{} `json:"parameters,omitempty"`

	Namespace    string                 `json:"namespace,omitempty"`

	Labels       map[string]string      `json:"labels,omitempty"`

	Annotations  map[string]string      `json:"annotations,omitempty"`

	Timeout      time.Duration          `json:"timeout,omitempty"`

	DryRun       bool                   `json:"dryRun,omitempty"`

}



// UpdateDeploymentRequest represents a deployment update request.

type UpdateDeploymentRequest struct {

	Template    string                 `json:"template,omitempty"`

	Parameters  map[string]interface{} `json:"parameters,omitempty"`

	Labels      map[string]string      `json:"labels,omitempty"`

	Annotations map[string]string      `json:"annotations,omitempty"`

	Timeout     time.Duration          `json:"timeout,omitempty"`

}



// DeploymentResponse represents a deployment response.

type DeploymentResponse struct {

	ID           string                 `json:"id"`

	Name         string                 `json:"name"`

	Status       string                 `json:"status"`

	Phase        string                 `json:"phase"`

	Template     string                 `json:"template"`

	TemplateType string                 `json:"templateType"`

	Parameters   map[string]interface{} `json:"parameters,omitempty"`

	Namespace    string                 `json:"namespace,omitempty"`

	Resources    []*ResourceResponse    `json:"resources,omitempty"`

	Services     []*ServiceResponse     `json:"services,omitempty"`

	Events       []*DeploymentEvent     `json:"events,omitempty"`

	Labels       map[string]string      `json:"labels,omitempty"`

	Annotations  map[string]string      `json:"annotations,omitempty"`

	CreatedAt    time.Time              `json:"createdAt"`

	UpdatedAt    time.Time              `json:"updatedAt"`

}



// DeploymentEvent represents an event during deployment.

type DeploymentEvent struct {

	Type      string                 `json:"type"`

	Phase     string                 `json:"phase"`

	Message   string                 `json:"message"`

	Timestamp time.Time              `json:"timestamp"`

	Metadata  map[string]interface{} `json:"metadata,omitempty"`

}



// DeploymentFilter defines filters for deployment queries.

type DeploymentFilter struct {

	Names         []string          `json:"names,omitempty"`

	Statuses      []string          `json:"statuses,omitempty"`

	Phases        []string          `json:"phases,omitempty"`

	TemplateTypes []string          `json:"templateTypes,omitempty"`

	Namespaces    []string          `json:"namespaces,omitempty"`

	Labels        map[string]string `json:"labels,omitempty"`

	Annotations   map[string]string `json:"annotations,omitempty"`

	Limit         int               `json:"limit,omitempty"`

	Offset        int               `json:"offset,omitempty"`

}



// Scaling request/response types.



// ScaleRequest represents a scaling request.

type ScaleRequest struct {

	Type        string                 `json:"type"`      // horizontal, vertical

	Direction   string                 `json:"direction"` // up, down

	Amount      int32                  `json:"amount,omitempty"`

	Target      map[string]interface{} `json:"target,omitempty"`

	MinReplicas *int32                 `json:"minReplicas,omitempty"`

	MaxReplicas *int32                 `json:"maxReplicas,omitempty"`

	Metrics     []string               `json:"metrics,omitempty"`

	Timeout     time.Duration          `json:"timeout,omitempty"`

}



// ScalingCapabilities represents the scaling capabilities of a resource.

type ScalingCapabilities struct {

	HorizontalScaling bool          `json:"horizontalScaling"`

	VerticalScaling   bool          `json:"verticalScaling"`

	MinReplicas       int32         `json:"minReplicas"`

	MaxReplicas       int32         `json:"maxReplicas"`

	SupportedMetrics  []string      `json:"supportedMetrics"`

	ScaleUpCooldown   time.Duration `json:"scaleUpCooldown"`

	ScaleDownCooldown time.Duration `json:"scaleDownCooldown"`

}



// Health and monitoring types.



// HealthStatus represents the health status of a resource.

type HealthStatus struct {

	Status      string                 `json:"status"` // healthy, unhealthy, unknown

	Message     string                 `json:"message,omitempty"`

	Checks      []*HealthCheck         `json:"checks,omitempty"`

	LastUpdated time.Time              `json:"lastUpdated"`

	Metadata    map[string]interface{} `json:"metadata,omitempty"`

}



// HealthCheck represents an individual health check.

type HealthCheck struct {

	Name        string    `json:"name"`

	Type        string    `json:"type"`

	Status      string    `json:"status"`

	Message     string    `json:"message,omitempty"`

	LastChecked time.Time `json:"lastChecked"`

}



// Network service types.



// NetworkServiceRequest represents a network service request.

type NetworkServiceRequest struct {

	Name        string            `json:"name"`

	Type        string            `json:"type"` // service, ingress, networkpolicy

	Namespace   string            `json:"namespace,omitempty"`

	Selector    map[string]string `json:"selector,omitempty"`

	Ports       []*NetworkPort    `json:"ports,omitempty"`

	Rules       []*NetworkRule    `json:"rules,omitempty"`

	Labels      map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	Timeout     time.Duration     `json:"timeout,omitempty"`

}



// NetworkServiceResponse represents a network service response.

type NetworkServiceResponse struct {

	ID          string             `json:"id"`

	Name        string             `json:"name"`

	Type        string             `json:"type"`

	Namespace   string             `json:"namespace,omitempty"`

	Status      string             `json:"status"`

	ClusterIP   string             `json:"clusterIP,omitempty"`

	ExternalIP  string             `json:"externalIP,omitempty"`

	Ports       []*NetworkPort     `json:"ports,omitempty"`

	Endpoints   []*NetworkEndpoint `json:"endpoints,omitempty"`

	Labels      map[string]string  `json:"labels,omitempty"`

	Annotations map[string]string  `json:"annotations,omitempty"`

	CreatedAt   time.Time          `json:"createdAt"`

	UpdatedAt   time.Time          `json:"updatedAt"`

}



// NetworkPort represents a network port configuration.

type NetworkPort struct {

	Name       string `json:"name,omitempty"`

	Port       int32  `json:"port"`

	TargetPort string `json:"targetPort,omitempty"`

	Protocol   string `json:"protocol"`

	NodePort   int32  `json:"nodePort,omitempty"`

}



// NetworkEndpoint represents a network endpoint.

type NetworkEndpoint struct {

	Address string `json:"address"`

	Port    int32  `json:"port"`

	Ready   bool   `json:"ready"`

}



// NetworkRule represents a network rule for policies.

type NetworkRule struct {

	From   []*NetworkRuleSelector `json:"from,omitempty"`

	To     []*NetworkRuleSelector `json:"to,omitempty"`

	Ports  []*NetworkPort         `json:"ports,omitempty"`

	Action string                 `json:"action"` // allow, deny

}



// NetworkRuleSelector represents a selector for network rules.

type NetworkRuleSelector struct {

	PodSelector       *LabelSelector `json:"podSelector,omitempty"`

	NamespaceSelector *LabelSelector `json:"namespaceSelector,omitempty"`

	IPBlock           *IPBlock       `json:"ipBlock,omitempty"`

}



// LabelSelector represents a label selector.

type LabelSelector struct {

	MatchLabels      map[string]string           `json:"matchLabels,omitempty"`

	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions,omitempty"`

}



// LabelSelectorRequirement represents a label selector requirement.

type LabelSelectorRequirement struct {

	Key      string   `json:"key"`

	Operator string   `json:"operator"`

	Values   []string `json:"values,omitempty"`

}



// IPBlock represents an IP block for network rules.

type IPBlock struct {

	CIDR   string   `json:"cidr"`

	Except []string `json:"except,omitempty"`

}



// NetworkServiceFilter defines filters for network service queries.

type NetworkServiceFilter struct {

	Names       []string          `json:"names,omitempty"`

	Types       []string          `json:"types,omitempty"`

	Namespaces  []string          `json:"namespaces,omitempty"`

	Statuses    []string          `json:"statuses,omitempty"`

	Labels      map[string]string `json:"labels,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`

	Limit       int               `json:"limit,omitempty"`

	Offset      int               `json:"offset,omitempty"`

}



// Storage types.



// StorageResourceRequest represents a storage resource request.

type StorageResourceRequest struct {

	Name         string                 `json:"name"`

	Type         string                 `json:"type"` // persistentvolume, persistentvolumeclaim, storageclass

	Size         string                 `json:"size"`

	StorageClass string                 `json:"storageClass,omitempty"`

	AccessModes  []string               `json:"accessModes,omitempty"`

	Selector     map[string]string      `json:"selector,omitempty"`

	Namespace    string                 `json:"namespace,omitempty"`

	Labels       map[string]string      `json:"labels,omitempty"`

	Annotations  map[string]string      `json:"annotations,omitempty"`

	Parameters   map[string]interface{} `json:"parameters,omitempty"`

	Timeout      time.Duration          `json:"timeout,omitempty"`

}



// StorageResourceResponse represents a storage resource response.

type StorageResourceResponse struct {

	ID           string            `json:"id"`

	Name         string            `json:"name"`

	Type         string            `json:"type"`

	Status       string            `json:"status"`

	Size         string            `json:"size"`

	StorageClass string            `json:"storageClass,omitempty"`

	AccessModes  []string          `json:"accessModes,omitempty"`

	Namespace    string            `json:"namespace,omitempty"`

	MountPath    string            `json:"mountPath,omitempty"`

	Labels       map[string]string `json:"labels,omitempty"`

	Annotations  map[string]string `json:"annotations,omitempty"`

	CreatedAt    time.Time         `json:"createdAt"`

	UpdatedAt    time.Time         `json:"updatedAt"`

}



// StorageResourceFilter defines filters for storage resource queries.

type StorageResourceFilter struct {

	Names          []string          `json:"names,omitempty"`

	Types          []string          `json:"types,omitempty"`

	Statuses       []string          `json:"statuses,omitempty"`

	StorageClasses []string          `json:"storageClasses,omitempty"`

	Namespaces     []string          `json:"namespaces,omitempty"`

	Labels         map[string]string `json:"labels,omitempty"`

	Annotations    map[string]string `json:"annotations,omitempty"`

	Limit          int               `json:"limit,omitempty"`

	Offset         int               `json:"offset,omitempty"`

}



// ServiceResponse represents a service response.

type ServiceResponse struct {

	ID          string             `json:"id"`

	Name        string             `json:"name"`

	Type        string             `json:"type"`

	Namespace   string             `json:"namespace,omitempty"`

	Status      string             `json:"status"`

	ClusterIP   string             `json:"clusterIP,omitempty"`

	ExternalIP  string             `json:"externalIP,omitempty"`

	Ports       []*NetworkPort     `json:"ports,omitempty"`

	Selector    map[string]string  `json:"selector,omitempty"`

	Labels      map[string]string  `json:"labels,omitempty"`

	Annotations map[string]string  `json:"annotations,omitempty"`

	Endpoints   []*NetworkEndpoint `json:"endpoints,omitempty"`

	CreatedAt   time.Time          `json:"createdAt"`

	UpdatedAt   time.Time          `json:"updatedAt"`

}



// Event handling types.



// EventCallback represents a callback function for provider events.

type EventCallback func(event *ProviderEvent)



// ProviderEvent represents an event from a cloud provider.

type ProviderEvent struct {

	ID        string                 `json:"id"`

	Type      string                 `json:"type"`

	Source    string                 `json:"source"`

	Subject   string                 `json:"subject"`

	Data      map[string]interface{} `json:"data"`

	Timestamp time.Time              `json:"timestamp"`

	Metadata  map[string]interface{} `json:"metadata,omitempty"`

}



// Configuration types.



// ProviderConfiguration represents provider-specific configuration.

type ProviderConfiguration struct {

	Name        string                 `json:"name"`

	Type        string                 `json:"type"`

	Version     string                 `json:"version"`

	Enabled     bool                   `json:"enabled"`

	Region      string                 `json:"region,omitempty"`

	Zone        string                 `json:"zone,omitempty"`

	Endpoint    string                 `json:"endpoint,omitempty"`

	Credentials map[string]string      `json:"credentials,omitempty"`

	Parameters  map[string]interface{} `json:"parameters,omitempty"`

	Limits      map[string]interface{} `json:"limits,omitempty"`

	Features    map[string]bool        `json:"features,omitempty"`

	Metadata    map[string]string      `json:"metadata,omitempty"`

}



// Constants for provider operations.



const (

	// Provider Types.

	ProviderTypeKubernetes = "kubernetes"

	// ProviderTypeOpenStack holds providertypeopenstack value.

	ProviderTypeOpenStack = "openstack"

	// ProviderTypeVMware holds providertypevmware value.

	ProviderTypeVMware = "vmware"

	// ProviderTypeAWS holds providertypeaws value.

	ProviderTypeAWS = "aws"

	// ProviderTypeAzure holds providertypeazure value.

	ProviderTypeAzure = "azure"

	// ProviderTypeGCP holds providertypegcp value.

	ProviderTypeGCP = "gcp"



	// Resource Types.

	ResourceTypeCompute = "compute"

	// ResourceTypeStorage holds resourcetypestorage value.

	ResourceTypeStorage = "storage"

	// ResourceTypeNetwork holds resourcetypenetwork value.

	ResourceTypeNetwork = "network"

	// ResourceTypeAccelerator holds resourcetypeaccelerator value.

	ResourceTypeAccelerator = "accelerator"

	// ResourceTypeService holds resourcetypeservice value.

	ResourceTypeService = "service"



	// Resource Statuses.

	ResourceStatusPending = "pending"

	// ResourceStatusRunning holds resourcestatusrunning value.

	ResourceStatusRunning = "running"

	// ResourceStatusSucceeded holds resourcestatussucceeded value.

	ResourceStatusSucceeded = "succeeded"

	// ResourceStatusFailed holds resourcestatusfailed value.

	ResourceStatusFailed = "failed"

	// ResourceStatusTerminating holds resourcestatusterminating value.

	ResourceStatusTerminating = "terminating"



	// Health Statuses.

	HealthStatusHealthy = "healthy"

	// HealthStatusUnhealthy holds healthstatusunhealthy value.

	HealthStatusUnhealthy = "unhealthy"

	// HealthStatusUnknown holds healthstatusunknown value.

	HealthStatusUnknown = "unknown"



	// Scale Types.

	ScaleTypeHorizontal = "horizontal"

	// ScaleTypeVertical holds scaletypevertical value.

	ScaleTypeVertical = "vertical"



	// Scale Directions.

	ScaleDirectionUp = "up"

	// ScaleDirectionDown holds scaledirectiondown value.

	ScaleDirectionDown = "down"



	// Network Service Types.

	NetworkServiceTypeService = "service"

	// NetworkServiceTypeIngress holds networkservicetypeingress value.

	NetworkServiceTypeIngress = "ingress"

	// NetworkServiceTypeNetworkPolicy holds networkservicetypenetworkpolicy value.

	NetworkServiceTypeNetworkPolicy = "networkpolicy"



	// Storage Types.

	StorageTypePersistentVolume = "persistentvolume"

	// StorageTypePersistentVolumeClaim holds storagetypepersistentvolumeclaim value.

	StorageTypePersistentVolumeClaim = "persistentvolumeclaim"

	// StorageTypeStorageClass holds storagetypestorageclass value.

	StorageTypeStorageClass = "storageclass"

)

