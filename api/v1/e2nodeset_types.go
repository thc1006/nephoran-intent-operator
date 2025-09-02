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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RANFunction defines a RAN function supported by E2 nodes.

type RANFunction struct {
	// FunctionID is the unique identifier for the RAN function (0-4095).

	// +kubebuilder:validation:Minimum=0

	// +kubebuilder:validation:Maximum=4095

	FunctionID int32 `json:"functionID"`

	// Revision is the revision of the RAN function (0-255).

	// +kubebuilder:validation:Minimum=0

	// +kubebuilder:validation:Maximum=255

	Revision int32 `json:"revision"`

	// Description provides a human-readable description of the RAN function.

	// +kubebuilder:validation:MaxLength=256

	Description string `json:"description"`

	// OID is the ASN.1 Object Identifier for the RAN function.

	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)*$`

	OID string `json:"oid"`
}

// E2NodeSpec defines the specification for an individual E2 node.

type E2NodeSpec struct {
	// NodeID is the unique identifier for the E2 node.

	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`

	NodeID string `json:"nodeID"`

	// E2InterfaceVersion specifies the E2 interface version.

	// +kubebuilder:validation:Enum=v10;v11;v20;v2.1;v3.0

	E2InterfaceVersion string `json:"e2InterfaceVersion"`

	// SupportedRANFunctions lists the RAN functions supported by this E2 node.

	// +kubebuilder:validation:MinItems=1

	// +kubebuilder:validation:MaxItems=256

	SupportedRANFunctions []RANFunction `json:"supportedRANFunctions"`
}

// E2NodeTemplate defines the template for creating E2 nodes.

type E2NodeTemplate struct {
	// Metadata for the E2 node template.

	// +optional

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the E2 node specification.

	Spec E2NodeSpec `json:"spec"`
}

// TrafficProfile defines the traffic generation profile.

// +kubebuilder:validation:Enum=low;medium;high;burst

type TrafficProfile string

const (

	// TrafficProfileLow holds trafficprofilelow value.

	TrafficProfileLow TrafficProfile = "low"

	// TrafficProfileMedium holds trafficprofilemedium value.

	TrafficProfileMedium TrafficProfile = "medium"

	// TrafficProfileHigh holds trafficprofilehigh value.

	TrafficProfileHigh TrafficProfile = "high"

	// TrafficProfileBurst holds trafficprofileburst value.

	TrafficProfileBurst TrafficProfile = "burst"
)

// SimulationConfig defines configuration for E2 node simulation.

type SimulationConfig struct {
	// UECount specifies the number of UEs to simulate per E2 node.

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=10000

	// +kubebuilder:default=100

	UECount int32 `json:"ueCount,omitempty"`

	// TrafficGeneration enables traffic generation simulation.

	// +kubebuilder:default=false

	TrafficGeneration bool `json:"trafficGeneration,omitempty"`

	// MetricsInterval specifies the interval for metrics reporting.

	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`

	// +kubebuilder:default="30s"

	MetricsInterval string `json:"metricsInterval,omitempty"`

	// TrafficProfile defines the traffic generation profile.

	// +kubebuilder:default=low

	TrafficProfile TrafficProfile `json:"trafficProfile,omitempty"`
}

// RetryConfig defines retry configuration for RIC connections.

type RetryConfig struct {
	// MaxAttempts specifies the maximum number of retry attempts.

	// +kubebuilder:validation:Minimum=1

	// +kubebuilder:validation:Maximum=10

	// +kubebuilder:default=3

	MaxAttempts int32 `json:"maxAttempts,omitempty"`

	// BackoffInterval specifies the backoff interval between retries.

	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`

	// +kubebuilder:default="5s"

	BackoffInterval string `json:"backoffInterval,omitempty"`
}

// RICConfiguration defines configuration for RIC connectivity.

type RICConfiguration struct {
	// RICEndpoint specifies the RIC endpoint URL.

	// +kubebuilder:validation:Pattern=`^https?://[a-zA-Z0-9-]+:[0-9]+$`

	// +kubebuilder:default="http://near-rt-ric:38080"

	RICEndpoint string `json:"ricEndpoint,omitempty"`

	// ConnectionTimeout specifies the timeout for establishing connections.

	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`

	// +kubebuilder:default="30s"

	ConnectionTimeout string `json:"connectionTimeout,omitempty"`

	// HeartbeatInterval specifies the interval for heartbeat messages.

	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$`

	// +kubebuilder:default="10s"

	HeartbeatInterval string `json:"heartbeatInterval,omitempty"`

	// RetryConfig defines retry behavior for failed connections.

	// +optional

	RetryConfig *RetryConfig `json:"retryConfig,omitempty"`
}

// E2NodeSetSpec defines the desired state of E2NodeSet.

type E2NodeSetSpec struct {
	// Replicas is the number of simulated E2 Nodes to run.

	// +kubebuilder:validation:Minimum=0

	// +kubebuilder:validation:Maximum=1000

	// +kubebuilder:default=1

	Replicas int32 `json:"replicas"`

	// Template defines the template for creating E2 nodes.

	Template E2NodeTemplate `json:"template"`

	// SimulationConfig defines simulation parameters.

	// +optional

	SimulationConfig *SimulationConfig `json:"simulationConfig,omitempty"`

	// RICConfiguration defines RIC connectivity settings.

	// +optional

	RICConfiguration *RICConfiguration `json:"ricConfiguration,omitempty"`

	// RicEndpoint is the Near-RT RIC endpoint for E2 connections.

	// Deprecated: Use ricConfiguration.ricEndpoint instead.

	// If not specified, defaults to "http://near-rt-ric:38080".

	// +kubebuilder:validation:Pattern=`^https?://[a-zA-Z0-9-]+:[0-9]+$`

	// +optional

	RicEndpoint string `json:"ricEndpoint,omitempty"`
}

// E2NodeLifecycleState represents the lifecycle state of E2 nodes.

// +kubebuilder:validation:Enum=Pending;Initializing;Connected;Disconnected;Error;Terminating

type E2NodeLifecycleState string

const (

	// E2NodeLifecycleStatePending holds e2nodelifecyclestatepending value.

	E2NodeLifecycleStatePending E2NodeLifecycleState = "Pending"

	// E2NodeLifecycleStateInitializing holds e2nodelifecyclestateinitializing value.

	E2NodeLifecycleStateInitializing E2NodeLifecycleState = "Initializing"

	// E2NodeLifecycleStateConnected holds e2nodelifecyclestateconnected value.

	E2NodeLifecycleStateConnected E2NodeLifecycleState = "Connected"

	// E2NodeLifecycleStateDisconnected holds e2nodelifecyclestatedisconnected value.

	E2NodeLifecycleStateDisconnected E2NodeLifecycleState = "Disconnected"

	// E2NodeLifecycleStateError holds e2nodelifecyclestateerror value.

	E2NodeLifecycleStateError E2NodeLifecycleState = "Error"

	// E2NodeLifecycleStateTerminating holds e2nodelifecyclestateterminating value.

	E2NodeLifecycleStateTerminating E2NodeLifecycleState = "Terminating"
)

// E2NodeStatus represents the status of an individual E2 node.

type E2NodeStatus struct {
	// NodeID is the identifier of the E2 node.

	NodeID string `json:"nodeID"`

	// State represents the current lifecycle state.

	State E2NodeLifecycleState `json:"state"`

	// LastHeartbeat is the timestamp of the last heartbeat.

	// +optional

	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`

	// ConnectedSince is the timestamp when the node connected.

	// +optional

	ConnectedSince *metav1.Time `json:"connectedSince,omitempty"`

	// ActiveSubscriptions is the number of active E2 subscriptions.

	// +optional

	ActiveSubscriptions int32 `json:"activeSubscriptions,omitempty"`

	// ErrorMessage provides details about any error state.

	// +optional

	ErrorMessage string `json:"errorMessage,omitempty"`
}

// E2NodeSetConditionType represents the type of E2NodeSet condition.

type E2NodeSetConditionType string

const (

	// E2NodeSetConditionAvailable holds e2nodesetconditionavailable value.

	E2NodeSetConditionAvailable E2NodeSetConditionType = "Available"

	// E2NodeSetConditionProgressing holds e2nodesetconditionprogressing value.

	E2NodeSetConditionProgressing E2NodeSetConditionType = "Progressing"

	// E2NodeSetConditionDegraded holds e2nodesetconditiondegraded value.

	E2NodeSetConditionDegraded E2NodeSetConditionType = "Degraded"
)

// E2NodeSetCondition represents a condition of the E2NodeSet.

type E2NodeSetCondition struct {
	// Type of the condition.

	Type E2NodeSetConditionType `json:"type"`

	// Status of the condition.

	Status metav1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time the condition transitioned.

	// +optional

	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last transition.

	// +optional

	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the condition.

	// +optional

	Message string `json:"message,omitempty"`
}

// E2NodeSetPhase represents the phase of E2NodeSet lifecycle.

// +kubebuilder:validation:Enum=Pending;Initializing;Ready;Scaling;Degraded;Failed;Terminating

type E2NodeSetPhase string

const (

	// E2NodeSetPhasePending holds e2nodesetphasepending value.

	E2NodeSetPhasePending E2NodeSetPhase = "Pending"

	// E2NodeSetPhaseInitializing holds e2nodesetphaseinitializing value.

	E2NodeSetPhaseInitializing E2NodeSetPhase = "Initializing"

	// E2NodeSetPhaseReady holds e2nodesetphaseready value.

	E2NodeSetPhaseReady E2NodeSetPhase = "Ready"

	// E2NodeSetPhaseScaling holds e2nodesetphasescaling value.

	E2NodeSetPhaseScaling E2NodeSetPhase = "Scaling"

	// E2NodeSetPhaseDegraded holds e2nodesetphasedegraded value.

	E2NodeSetPhaseDegraded E2NodeSetPhase = "Degraded"

	// E2NodeSetPhaseFailed holds e2nodesetphasefailed value.

	E2NodeSetPhaseFailed E2NodeSetPhase = "Failed"

	// E2NodeSetPhaseTerminating holds e2nodesetphaseterminating value.

	E2NodeSetPhaseTerminating E2NodeSetPhase = "Terminating"
)

// E2NodeSetStatus defines the observed state of E2NodeSet.

type E2NodeSetStatus struct {
	// Phase represents the current phase of the E2NodeSet lifecycle.

	// +optional

	Phase E2NodeSetPhase `json:"phase,omitempty"`

	// ReadyReplicas is the number of E2 Nodes that are ready and connected.

	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// CurrentReplicas is the current number of E2 Node replicas.

	CurrentReplicas int32 `json:"currentReplicas,omitempty"`

	// UpdatedReplicas is the number of E2 Node replicas updated to the latest template.

	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// AvailableReplicas is the number of available E2 Node replicas.

	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// E2NodeStatuses provides detailed status for each E2 node.

	// +optional

	E2NodeStatuses []E2NodeStatus `json:"e2NodeStatuses,omitempty"`

	// Conditions represents the latest available observations of the E2NodeSet's state.

	// +optional

	// +patchMergeKey=type

	// +patchStrategy=merge

	Conditions []E2NodeSetCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration reflects the generation of the most recently observed E2NodeSet.

	// +optional

	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// TotalSubscriptions is the total number of active E2 subscriptions across all nodes.

	// +optional

	TotalSubscriptions int32 `json:"totalSubscriptions,omitempty"`

	// LastUpdateTime is the last time the status was updated.

	// +optional

	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+kubebuilder:object:root=true

//+kubebuilder:subresource:status

//+kubebuilder:subresource:scale:specpath=spec.replicas,statuspath=status.replicas,selectorpath=status.selector

//+kubebuilder:printcolumn:name="Desired",type="integer",JSONPath=".spec.replicas",description="Number of desired E2 nodes"

//+kubebuilder:printcolumn:name="Current",type="integer",JSONPath=".status.currentReplicas",description="Number of current E2 nodes"

//+kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas",description="Number of ready E2 nodes"

//+kubebuilder:printcolumn:name="Available",type="integer",JSONPath=".status.availableReplicas",description="Number of available E2 nodes"

//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Current phase of the E2NodeSet"

//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the E2NodeSet"

//+kubebuilder:printcolumn:name="E2Version",type="string",JSONPath=".spec.template.spec.e2InterfaceVersion",description="E2 Interface Version"

// E2NodeSet is the Schema for the e2nodesets API.

type E2NodeSet struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec E2NodeSetSpec `json:"spec,omitempty"`

	Status E2NodeSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// E2NodeSetList contains a list of E2NodeSet.

type E2NodeSetList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`

	Items []E2NodeSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&E2NodeSet{}, &E2NodeSetList{})
}
