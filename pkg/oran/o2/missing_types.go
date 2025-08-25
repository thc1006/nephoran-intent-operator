//go:build o2_stub
// +build o2_stub

// Package o2 implements missing type definitions for O2 IMS complete implementation  
package o2

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
)

// DeploymentTemplate represents a deployment template in the O2 IMS system
type DeploymentTemplate struct {
	DeploymentTemplateID string                `json:"deploymentTemplateId"`
	Name                 string                `json:"name"`
	Description          string                `json:"description,omitempty"`
	Category             string                `json:"category"`
	Type                 string                `json:"type"` // helm, kubernetes, terraform, ansible
	Version              string                `json:"version"`
	Author               string                `json:"author,omitempty"`
	Content              *runtime.RawExtension `json:"content"`
	InputSchema          *runtime.RawExtension `json:"inputSchema,omitempty"`
	OutputSchema         *runtime.RawExtension `json:"outputSchema,omitempty"`
	Parameters           []TemplateParameter   `json:"parameters,omitempty"`
	Dependencies         []string              `json:"dependencies,omitempty"`
	Tags                 []string              `json:"tags,omitempty"`
	Labels               map[string]string     `json:"labels,omitempty"`
	Annotations          map[string]string     `json:"annotations,omitempty"`
	CreatedAt            time.Time             `json:"createdAt"`
	UpdatedAt            time.Time             `json:"updatedAt"`
}

// TemplateParameter represents a parameter in a deployment template
type TemplateParameter struct {
	Name          string               `json:"name"`
	Type          string               `json:"type"`
	Description   string               `json:"description"`
	Required      bool                 `json:"required"`
	DefaultValue  interface{}          `json:"defaultValue,omitempty"`
	AllowedValues []interface{}        `json:"allowedValues,omitempty"`
	Validation    *ParameterValidation `json:"validation,omitempty"`
}

// ParameterValidation represents validation rules for template parameters
type ParameterValidation struct {
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	MinValue  *float64 `json:"minValue,omitempty"`
	MaxValue  *float64 `json:"maxValue,omitempty"`
}

// Deployment represents a deployment instance in the O2 IMS system
type Deployment struct {
	DeploymentID    string                 `json:"deploymentId"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	TemplateID      string                 `json:"templateId"`
	TemplateName    string                 `json:"templateName"`
	TemplateVersion string                 `json:"templateVersion"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	Status          string                 `json:"status"`
	Phase           string                 `json:"phase"`
	Provider        string                 `json:"provider"`
	ResourcePoolID  string                 `json:"resourcePoolId,omitempty"`
	Resources       []DeploymentResource   `json:"resources,omitempty"`
	Services        []DeploymentService    `json:"services,omitempty"`
	Events          []DeploymentEvent      `json:"events,omitempty"`
	Labels          map[string]string      `json:"labels,omitempty"`
	Annotations     map[string]string      `json:"annotations,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	StartedAt       *time.Time             `json:"startedAt,omitempty"`
	CompletedAt     *time.Time             `json:"completedAt,omitempty"`
}

// DeploymentResource represents a resource created by a deployment
type DeploymentResource struct {
	ResourceID  string            `json:"resourceId"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// DeploymentService represents a service exposed by a deployment
type DeploymentService struct {
	ServiceID string            `json:"serviceId"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Endpoints []ServiceEndpoint `json:"endpoints,omitempty"`
	Status    string            `json:"status"`
}

// ServiceEndpoint represents an endpoint of a service
type ServiceEndpoint struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

// DeploymentEvent represents an event during deployment
type DeploymentEvent struct {
	EventID   string                 `json:"eventId"`
	Type      string                 `json:"type"`
	Phase     string                 `json:"phase"`
	Message   string                 `json:"message"`
	Reason    string                 `json:"reason,omitempty"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Subscription represents a subscription to O2 IMS events
type Subscription struct {
	SubscriptionID   string            `json:"subscriptionId"`
	Name             string            `json:"name,omitempty"`
	Description      string            `json:"description,omitempty"`
	EventTypes       []string          `json:"eventTypes"`
	ResourceTypes    []string          `json:"resourceTypes,omitempty"`
	ResourceIDs      []string          `json:"resourceIds,omitempty"`
	CallbackURL      string            `json:"callbackUrl"`
	AuthConfig       *SubscriptionAuth `json:"authConfig,omitempty"`
	FilterExpression string            `json:"filterExpression,omitempty"`
	Status           string            `json:"status"`
	Labels           map[string]string `json:"labels,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
	LastTriggered    *time.Time        `json:"lastTriggered,omitempty"`
	TriggerCount     int64             `json:"triggerCount"`
}

// SubscriptionAuth represents authentication configuration for subscription callbacks
type SubscriptionAuth struct {
	Type     string            `json:"type"` // none, basic, bearer, oauth2
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Token    string            `json:"token,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// Request types

// CreateDeploymentRequest represents a request to create a new deployment
type CreateDeploymentRequest struct {
	Name           string                 `json:"name" validate:"required"`
	Description    string                 `json:"description,omitempty"`
	TemplateID     string                 `json:"templateId" validate:"required"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	Provider       string                 `json:"provider" validate:"required"`
	ResourcePoolID string                 `json:"resourcePoolId,omitempty"`
	Labels         map[string]string      `json:"labels,omitempty"`
	Annotations    map[string]string      `json:"annotations,omitempty"`
	Timeout        time.Duration          `json:"timeout,omitempty"`
	DryRun         bool                   `json:"dryRun,omitempty"`
}

// UpdateDeploymentRequest represents a request to update a deployment
type UpdateDeploymentRequest struct {
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
}

// Template types constants
const (
	TemplateTypeHelm       = "helm"
	TemplateTypeKubernetes = "kubernetes"
	TemplateTypeTerraform  = "terraform"
	TemplateTypeAnsible    = "ansible"
)

// Deployment status constants
const (
	DeploymentStatusPending   = "PENDING"
	DeploymentStatusRunning   = "RUNNING"
	DeploymentStatusCompleted = "COMPLETED"
	DeploymentStatusFailed    = "FAILED"
	DeploymentStatusCancelled = "CANCELLED"
)

// Deployment phases
const (
	DeploymentPhaseInitializing = "INITIALIZING"
	DeploymentPhaseProvisioning = "PROVISIONING"
	DeploymentPhaseConfiguring  = "CONFIGURING"
	DeploymentPhaseDeploying    = "DEPLOYING"
	DeploymentPhaseCompleted    = "COMPLETED"
	DeploymentPhaseFailed       = "FAILED"
)

// Subscription status constants
const (
	SubscriptionStatusActive   = "ACTIVE"
	SubscriptionStatusInactive = "INACTIVE"
	SubscriptionStatusError    = "ERROR"
)

// Event types for subscriptions
const (
	EventTypeResourceCreated       = "RESOURCE_CREATED"
	EventTypeResourceUpdated       = "RESOURCE_UPDATED"
	EventTypeResourceDeleted       = "RESOURCE_DELETED"
	EventTypeResourceHealthChanged = "RESOURCE_HEALTH_CHANGED"
	EventTypeDeploymentCreated     = "DEPLOYMENT_CREATED"
	EventTypeDeploymentUpdated     = "DEPLOYMENT_UPDATED"
	EventTypeDeploymentCompleted   = "DEPLOYMENT_COMPLETED"
	EventTypeDeploymentFailed      = "DEPLOYMENT_FAILED"
	EventTypeAlarmRaised           = "ALARM_RAISED"
	EventTypeAlarmCleared          = "ALARM_CLEARED"
)

// Additional supporting types for complete implementation

// LabelSelector represents a label selector
type LabelSelector struct {
	MatchLabels      map[string]string           `json:"matchLabels,omitempty"`
	MatchExpressions []*LabelSelectorRequirement `json:"matchExpressions,omitempty"`
}

// LabelSelectorRequirement represents a label selector requirement
type LabelSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// ResourceMetric represents a resource metric with total, available, and used values
type ResourceMetric struct {
	Total     string `json:"total"`
	Available string `json:"available"`
	Used      string `json:"used"`
	Unit      string `json:"unit"`
}
