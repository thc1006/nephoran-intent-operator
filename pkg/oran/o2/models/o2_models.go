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

package models

import (
	"time"
)

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

// CreateSubscriptionRequest represents a subscription creation request
type CreateSubscriptionRequest struct {
	ConsumerSubscriptionID string                 `json:"consumerSubscriptionId"`
	Filter                 string                 `json:"filter,omitempty"`
	Callback               string                 `json:"callback"`
	ConsumerInfo           *ConsumerInfo          `json:"consumerInfo,omitempty"`
	EventTypes             []string               `json:"eventTypes"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
}

// ConsumerInfo represents subscription consumer information
type ConsumerInfo struct {
	ConsumerID   string `json:"consumerId"`
	ConsumerName string `json:"consumerName"`
	Description  string `json:"description,omitempty"`
}

// Subscription represents an O2 subscription
type Subscription struct {
	SubscriptionID         string                 `json:"subscriptionId"`
	ConsumerSubscriptionID string                 `json:"consumerSubscriptionId"`
	Filter                 string                 `json:"filter,omitempty"`
	Callback               string                 `json:"callback"`
	ConsumerInfo           *ConsumerInfo          `json:"consumerInfo,omitempty"`
	EventTypes             []string               `json:"eventTypes"`
	Status                 string                 `json:"status"`
	CreatedAt              time.Time              `json:"createdAt"`
	UpdatedAt              time.Time              `json:"updatedAt"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
}

// SubscriptionFilter represents filter criteria for subscriptions
type SubscriptionFilter struct {
	SubscriptionID         string    `json:"subscriptionId,omitempty"`
	ConsumerSubscriptionID string    `json:"consumerSubscriptionId,omitempty"`
	ConsumerID             string    `json:"consumerId,omitempty"`
	Status                 string    `json:"status,omitempty"`
	EventTypes             []string  `json:"eventTypes,omitempty"`
	CreatedAfter           *time.Time `json:"createdAfter,omitempty"`
	CreatedBefore          *time.Time `json:"createdBefore,omitempty"`
}

// UpdateSubscriptionRequest represents a subscription update request
type UpdateSubscriptionRequest struct {
	Filter     string                 `json:"filter,omitempty"`
	Callback   string                 `json:"callback,omitempty"`
	EventTypes []string               `json:"eventTypes,omitempty"`
	Status     string                 `json:"status,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationEventType represents supported notification event types
type NotificationEventType struct {
	EventType   string `json:"eventType"`
	Description string `json:"description"`
	Schema      string `json:"schema,omitempty"`
	Version     string `json:"version"`
}

// Node represents an infrastructure inventory node
type Node struct {
	NodeID      string                 `json:"nodeId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Location    *NodeLocation          `json:"location,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Resources   []*NodeResource        `json:"resources,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// NodeLocation represents geographical location of a node
type NodeLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
	Site      string  `json:"site,omitempty"`
	Zone      string  `json:"zone,omitempty"`
}

// NodeResource represents resources available on a node
type NodeResource struct {
	ResourceType string                 `json:"resourceType"`
	Capacity     map[string]interface{} `json:"capacity"`
	Available    map[string]interface{} `json:"available"`
	Reserved     map[string]interface{} `json:"reserved,omitempty"`
}

// NodeFilter represents filter criteria for inventory nodes
type NodeFilter struct {
	NodeID      string   `json:"nodeId,omitempty"`
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	Status      []string `json:"status,omitempty"`
	Site        string   `json:"site,omitempty"`
	Zone        string   `json:"zone,omitempty"`
	HasResource string   `json:"hasResource,omitempty"`
}

// Alarm represents an O2 alarm
type Alarm struct {
	AlarmID            string                 `json:"alarmId"`
	ResourceID         string                 `json:"resourceId"`
	ResourceType       string                 `json:"resourceType"`
	AlarmEventRecordId string                 `json:"alarmEventRecordId,omitempty"`
	AlarmDefinitionID  string                 `json:"alarmDefinitionId"`
	AlarmRaisedTime    time.Time              `json:"alarmRaisedTime"`
	AlarmChangedTime   time.Time              `json:"alarmChangedTime,omitempty"`
	AlarmAckTime       *time.Time             `json:"alarmAckTime,omitempty"`
	AlarmClearTime     *time.Time             `json:"alarmClearTime,omitempty"`
	PerceivedSeverity  string                 `json:"perceivedSeverity"`
	ProbableCause      string                 `json:"probableCause"`
	SpecificProblem    string                 `json:"specificProblem"`
	AdditionalText     string                 `json:"additionalText,omitempty"`
	AlarmType          string                 `json:"alarmType"`
	AlarmState         string                 `json:"alarmState"`
	AckState           string                 `json:"ackState"`
	AckUser            string                 `json:"ackUser,omitempty"`
	AckSystemId        string                 `json:"ackSystemId,omitempty"`
	Extensions         map[string]interface{} `json:"extensions,omitempty"`
}

// AlarmFilter represents filter criteria for alarms
type AlarmFilter struct {
	AlarmID           string     `json:"alarmId,omitempty"`
	ResourceID        string     `json:"resourceId,omitempty"`
	ResourceType      string     `json:"resourceType,omitempty"`
	AlarmDefinitionID string     `json:"alarmDefinitionId,omitempty"`
	PerceivedSeverity []string   `json:"perceivedSeverity,omitempty"`
	ProbableCause     string     `json:"probableCause,omitempty"`
	AlarmType         []string   `json:"alarmType,omitempty"`
	AlarmState        []string   `json:"alarmState,omitempty"`
	AckState          []string   `json:"ackState,omitempty"`
	RaisedAfter       *time.Time `json:"raisedAfter,omitempty"`
	RaisedBefore      *time.Time `json:"raisedBefore,omitempty"`
}

// AlarmAcknowledgementRequest represents an alarm acknowledgement request
type AlarmAcknowledgementRequest struct {
	AckUser      string                 `json:"ackUser"`
	AckSystemId  string                 `json:"ackSystemId,omitempty"`
	AckComments  string                 `json:"ackComments,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AlarmClearRequest represents an alarm clear request
type AlarmClearRequest struct {
	ClearUser    string                 `json:"clearUser"`
	ClearSystemId string                `json:"clearSystemId,omitempty"`
	ClearReason  string                 `json:"clearReason,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ResourcePoolFilter represents filter criteria for resource pools
type ResourcePoolFilter struct {
	PoolID       string   `json:"poolId,omitempty"`
	Name         string   `json:"name,omitempty"`
	Type         string   `json:"type,omitempty"`
	Status       []string `json:"status,omitempty"`
	Location     string   `json:"location,omitempty"`
	HasResources bool     `json:"hasResources,omitempty"`
}

// CreateResourcePoolRequest represents a resource pool creation request
type CreateResourcePoolRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Type         string                 `json:"type"`
	Location     *PoolLocation          `json:"location,omitempty"`
	Capacity     map[string]interface{} `json:"capacity,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateResourcePoolRequest represents a resource pool update request
type UpdateResourcePoolRequest struct {
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Capacity     map[string]interface{} `json:"capacity,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PoolLocation represents geographical location of a resource pool
type PoolLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
	Site      string  `json:"site,omitempty"`
	Zone      string  `json:"zone,omitempty"`
	Region    string  `json:"region,omitempty"`
}

// ResourceTypeFilter represents filter criteria for resource types
type ResourceTypeFilter struct {
	TypeID      string   `json:"typeId,omitempty"`
	Name        string   `json:"name,omitempty"`
	Category    string   `json:"category,omitempty"`
	Vendor      string   `json:"vendor,omitempty"`
	Version     string   `json:"version,omitempty"`
	Deployable  *bool    `json:"deployable,omitempty"`
}

// ResourceFilter represents filter criteria for resources
type ResourceFilter struct {
	ResourceID   string   `json:"resourceId,omitempty"`
	Name         string   `json:"name,omitempty"`
	Type         string   `json:"type,omitempty"`
	Status       []string `json:"status,omitempty"`
	PoolID       string   `json:"poolId,omitempty"`
	ParentID     string   `json:"parentId,omitempty"`
	HasChildren  *bool    `json:"hasChildren,omitempty"`
}

// CreateResourceRequest represents a resource creation request
type CreateResourceRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Type         string                 `json:"type"`
	PoolID       string                 `json:"poolId"`
	ParentID     string                 `json:"parentId,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateResourceRequest represents a resource update request
type UpdateResourceRequest struct {
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// DeploymentTemplateFilter represents filter criteria for deployment templates
type DeploymentTemplateFilter struct {
	TemplateID  string   `json:"templateId,omitempty"`
	Name        string   `json:"name,omitempty"`
	Type        string   `json:"type,omitempty"`
	Version     string   `json:"version,omitempty"`
	Status      []string `json:"status,omitempty"`
	Category    string   `json:"category,omitempty"`
}

// CreateDeploymentTemplateRequest represents a deployment template creation request
type CreateDeploymentTemplateRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Type         string                 `json:"type"`
	Version      string                 `json:"version"`
	Category     string                 `json:"category,omitempty"`
	Template     map[string]interface{} `json:"template"`
	Parameters   []*TemplateParameter   `json:"parameters,omitempty"`
	Requirements *TemplateRequirements  `json:"requirements,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateDeploymentTemplateRequest represents a deployment template update request
type UpdateDeploymentTemplateRequest struct {
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Template     map[string]interface{} `json:"template,omitempty"`
	Parameters   []*TemplateParameter   `json:"parameters,omitempty"`
	Requirements *TemplateRequirements  `json:"requirements,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateParameter represents a template parameter
type TemplateParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description,omitempty"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	Required     bool        `json:"required"`
	Constraints  *ParamConstraints `json:"constraints,omitempty"`
}

// ParamConstraints represents parameter constraints
type ParamConstraints struct {
	MinValue    *float64 `json:"minValue,omitempty"`
	MaxValue    *float64 `json:"maxValue,omitempty"`
	MinLength   *int     `json:"minLength,omitempty"`
	MaxLength   *int     `json:"maxLength,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	AllowedValues []interface{} `json:"allowedValues,omitempty"`
}

// TemplateRequirements represents template resource requirements
type TemplateRequirements struct {
	MinCPU          *float64           `json:"minCpu,omitempty"`
	MinMemory       *int64             `json:"minMemory,omitempty"`
	MinStorage      *int64             `json:"minStorage,omitempty"`
	RequiredLabels  map[string]string  `json:"requiredLabels,omitempty"`
	RequiredFeatures []string          `json:"requiredFeatures,omitempty"`
	NetworkRequirements *NetworkReqs   `json:"networkRequirements,omitempty"`
}

// NetworkReqs represents network requirements
type NetworkReqs struct {
	MinBandwidth    *int64    `json:"minBandwidth,omitempty"`
	MaxLatency      *int      `json:"maxLatency,omitempty"`
	RequiredPorts   []int     `json:"requiredPorts,omitempty"`
	NetworkPolicies []string  `json:"networkPolicies,omitempty"`
}

// DeploymentFilter represents filter criteria for deployments
type DeploymentFilter struct {
	DeploymentID string   `json:"deploymentId,omitempty"`
	Name         string   `json:"name,omitempty"`
	TemplateID   string   `json:"templateId,omitempty"`
	Status       []string `json:"status,omitempty"`
	Environment  string   `json:"environment,omitempty"`
	PoolID       string   `json:"poolId,omitempty"`
	CreatedBy    string   `json:"createdBy,omitempty"`
}

// CreateDeploymentRequest represents a deployment creation request
type CreateDeploymentRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	TemplateID   string                 `json:"templateId"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	PoolID       string                 `json:"poolId"`
	Environment  string                 `json:"environment,omitempty"`
	AutoStart    bool                   `json:"autoStart"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateDeploymentRequest represents a deployment update request
type UpdateDeploymentRequest struct {
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}