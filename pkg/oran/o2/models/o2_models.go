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

// SystemInfo represents O2 IMS system information.

type SystemInfo struct {
	Name string `json:"name"`

	Description string `json:"description"`

	Version string `json:"version"`

	APIVersions []string `json:"apiVersions"`

	SupportedResourceTypes []string `json:"supportedResourceTypes"`

	Extensions map[string]interface{} `json:"extensions"`

	Timestamp time.Time `json:"timestamp"`
}

// CreateSubscriptionRequest represents a subscription creation request.

type CreateSubscriptionRequest struct {
	ConsumerSubscriptionID string `json:"consumerSubscriptionId"`

	Filter string `json:"filter,omitempty"`

	Callback string `json:"callback"`

	ConsumerInfo *ConsumerInfo `json:"consumerInfo,omitempty"`

	EventTypes []string `json:"eventTypes"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ConsumerInfo represents subscription consumer information.

type ConsumerInfo struct {
	ConsumerID string `json:"consumerId"`

	ConsumerName string `json:"consumerName"`

	Description string `json:"description,omitempty"`
}

// UpdateSubscriptionRequest represents a subscription update request.

type UpdateSubscriptionRequest struct {
	Filter string `json:"filter,omitempty"`

	Callback string `json:"callback,omitempty"`

	EventTypes []string `json:"eventTypes,omitempty"`

	Status string `json:"status,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationEventType represents supported notification event types.

type NotificationEventType struct {
	EventType string `json:"eventType"`

	Description string `json:"description"`

	Schema string `json:"schema,omitempty"`

	Version string `json:"version"`
}

// Alarm represents an O2 alarm.

type Alarm struct {
	AlarmID string `json:"alarmId"`

	ResourceID string `json:"resourceId"`

	ResourceType string `json:"resourceType"`

	AlarmEventRecordId string `json:"alarmEventRecordId,omitempty"`

	AlarmDefinitionID string `json:"alarmDefinitionId"`

	AlarmRaisedTime time.Time `json:"alarmRaisedTime"`

	AlarmChangedTime time.Time `json:"alarmChangedTime,omitempty"`

	AlarmAckTime *time.Time `json:"alarmAckTime,omitempty"`

	AlarmClearTime *time.Time `json:"alarmClearTime,omitempty"`

	PerceivedSeverity string `json:"perceivedSeverity"`

	ProbableCause string `json:"probableCause"`

	SpecificProblem string `json:"specificProblem"`

	AdditionalText string `json:"additionalText,omitempty"`

	AlarmType string `json:"alarmType"`

	AlarmState string `json:"alarmState"`

	AckState string `json:"ackState"`

	AckUser string `json:"ackUser,omitempty"`

	AckSystemId string `json:"ackSystemId,omitempty"`

	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// AlarmFilter represents filter criteria for alarms.

type AlarmFilter struct {
	AlarmID string `json:"alarmId,omitempty"`

	ResourceID string `json:"resourceId,omitempty"`

	ResourceType string `json:"resourceType,omitempty"`

	AlarmDefinitionID string `json:"alarmDefinitionId,omitempty"`

	PerceivedSeverity []string `json:"perceivedSeverity,omitempty"`

	ProbableCause string `json:"probableCause,omitempty"`

	AlarmType []string `json:"alarmType,omitempty"`

	AlarmState []string `json:"alarmState,omitempty"`

	AckState []string `json:"ackState,omitempty"`

	RaisedAfter *time.Time `json:"raisedAfter,omitempty"`

	RaisedBefore *time.Time `json:"raisedBefore,omitempty"`
}

// AlarmAcknowledgementRequest represents an alarm acknowledgement request.

type AlarmAcknowledgementRequest struct {
	AckUser string `json:"ackUser"`

	AckSystemId string `json:"ackSystemId,omitempty"`

	AckComments string `json:"ackComments,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AlarmClearRequest represents an alarm clear request.

type AlarmClearRequest struct {
	ClearUser string `json:"clearUser"`

	ClearSystemId string `json:"clearSystemId,omitempty"`

	ClearReason string `json:"clearReason,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PoolLocation represents geographical location of a resource pool.

type PoolLocation struct {
	Latitude float64 `json:"latitude"`

	Longitude float64 `json:"longitude"`

	Address string `json:"address,omitempty"`

	Site string `json:"site,omitempty"`

	Zone string `json:"zone,omitempty"`

	Region string `json:"region,omitempty"`
}

// TemplateRequirements represents template resource requirements.

type TemplateRequirements struct {
	MinCPU *float64 `json:"minCpu,omitempty"`

	MinMemory *int64 `json:"minMemory,omitempty"`

	MinStorage *int64 `json:"minStorage,omitempty"`

	RequiredLabels map[string]string `json:"requiredLabels,omitempty"`

	RequiredFeatures []string `json:"requiredFeatures,omitempty"`

	NetworkRequirements *NetworkReqs `json:"networkRequirements,omitempty"`
}

// NetworkReqs represents network requirements.

type NetworkReqs struct {
	MinBandwidth *int64 `json:"minBandwidth,omitempty"`

	MaxLatency *int `json:"maxLatency,omitempty"`

	RequiredPorts []int `json:"requiredPorts,omitempty"`

	NetworkPolicies []string `json:"networkPolicies,omitempty"`
}
