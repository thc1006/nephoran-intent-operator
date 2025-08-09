package e2

import (
	"sync"
	"time"
)

// E2AP Additional Type Definitions following O-RAN.WG3.E2AP-v03.01

// E2NodeType represents the type of E2 node
type E2NodeType int

const (
	E2NodeTypegNB E2NodeType = iota
	E2NodeTypeeNB
	E2NodeTypeNgENB
	E2NodeTypeEnGNB
)

// RANFunctionItem represents a RAN function item
type RANFunctionItem struct {
	RANFunctionID         int    `json:"ran_function_id"`
	RANFunctionDefinition string `json:"ran_function_definition"`
	RANFunctionRevision   int    `json:"ran_function_revision"`
	RANFunctionOID        string `json:"ran_function_oid"`
}

// RANFunctionIDItem represents a RAN function ID item
type RANFunctionIDItem struct {
	RANFunctionID       int `json:"ran_function_id"`
	RANFunctionRevision int `json:"ran_function_revision"`
}

// RANFunctionIDCause represents a RAN function ID with cause
type RANFunctionIDCauseItem struct {
	RANFunctionID int     `json:"ran_function_id"`
	Cause         E2Cause `json:"cause"`
}

// Alias for compatibility
type RANFunctionIDCause = RANFunctionIDCauseItem

// E2NodeComponentConfigurationItem represents E2 node component configuration
type E2NodeComponentConfigurationItem struct {
	E2NodeComponentInterfaceType E2NodeComponentInterfaceType `json:"e2_node_component_interface_type"`
	E2NodeComponentID            string                       `json:"e2_node_component_id"`
	E2NodeComponentConfiguration string                       `json:"e2_node_component_configuration"`
}

// E2NodeComponentInterfaceType represents the interface type
type E2NodeComponentInterfaceType int

const (
	E2NodeComponentInterfaceTypeNG E2NodeComponentInterfaceType = iota
	E2NodeComponentInterfaceTypeXn
	E2NodeComponentInterfaceTypeE1
	E2NodeComponentInterfaceTypeF1
	E2NodeComponentInterfaceTypeW1
	E2NodeComponentInterfaceTypeS1
	E2NodeComponentInterfaceTypeX2
)

// E2NodeComponentID represents E2 node component ID
type E2NodeComponentID struct {
	InterfaceType E2NodeComponentInterfaceType `json:"interface_type"`
	ComponentID   string                       `json:"component_id"`
}

// RICSubscriptionDetails represents subscription details
type RICSubscriptionDetails struct {
	EventTriggerDefinition []byte               `json:"event_trigger_definition"`
	ActionsToBeSetupList   []RICActionToBeSetup `json:"actions_to_be_setup_list"`
}

// RICActionToBeSetup represents an action to be setup
type RICActionToBeSetup struct {
	ActionID         int                  `json:"action_id"`
	ActionType       RICActionType        `json:"action_type"`
	ActionDefinition []byte               `json:"action_definition,omitempty"`
	SubsequentAction *RICSubsequentAction `json:"subsequent_action,omitempty"`
}

// RICActionToBeSetupItem represents an action item to be setup
type RICActionToBeSetupItem struct {
	ActionID         int                  `json:"action_id"`
	ActionType       RICActionType        `json:"action_type"`
	ActionDefinition []byte               `json:"action_definition,omitempty"`
	SubsequentAction *RICSubsequentAction `json:"subsequent_action,omitempty"`
}

// RICActionType represents the type of RIC action
type RICActionType int

const (
	RICActionTypeReport RICActionType = iota
	RICActionTypeInsert
	RICActionTypePolicy
)

// RICSubsequentAction represents subsequent action
type RICSubsequentAction struct {
	SubsequentActionType RICSubsequentActionType `json:"subsequent_action_type"`
	TimeToWait           RICTimeToWait           `json:"time_to_wait"`
}

// RICSubsequentActionType represents subsequent action type
type RICSubsequentActionType int

const (
	RICSubsequentActionTypeContinue RICSubsequentActionType = iota
	RICSubsequentActionTypeWait
)

// RICTimeToWait represents time to wait
type RICTimeToWait int

const (
	RICTimeToWaitZero RICTimeToWait = iota
	RICTimeToWaitW1ms
	RICTimeToWaitW2ms
	RICTimeToWaitW5ms
	RICTimeToWaitW10ms
	RICTimeToWaitW20ms
	RICTimeToWaitW30ms
	RICTimeToWaitW40ms
	RICTimeToWaitW50ms
	RICTimeToWaitW100ms
	RICTimeToWaitW200ms
	RICTimeToWaitW500ms
	RICTimeToWaitW1s
	RICTimeToWaitW2s
	RICTimeToWaitW5s
	RICTimeToWaitW10s
	RICTimeToWaitW20s
	RICTimeToWaitW60s
)

// RICActionNotAdmittedItem represents an action not admitted
type RICActionNotAdmittedItem struct {
	ActionID int     `json:"action_id"`
	Cause    E2Cause `json:"cause"`
}

// RICActionAdmittedItem represents an admitted action
type RICActionAdmittedItem struct {
	ActionID int `json:"action_id"`
}

// E2Cause represents the cause for E2AP procedures
type E2Cause struct {
	CauseType  E2CauseType `json:"cause_type"`
	CauseValue int         `json:"cause_value"`
}

// E2CauseType represents the type of cause
type E2CauseType int

const (
	E2CauseTypeRIC E2CauseType = iota
	E2CauseTypeRICService
	E2CauseTypeE2Node
	E2CauseTypeTransport
	E2CauseTypeProtocol
	E2CauseTypeMisc
)

// RICServiceCause represents RIC service-related causes
type RICServiceCause int

const (
	RICServiceCauseUnspecified RICServiceCause = iota
	RICServiceCauseRANFunctionNotSupported
	RICServiceCauseExcessiveFunctions
	RICServiceCauseRICResourceLimit
)

// E2NodeCause represents E2 node-related causes
type E2NodeCause int

const (
	E2NodeCauseUnspecified E2NodeCause = iota
)

// TransportCause represents transport-related causes
type TransportCause int

const (
	TransportCauseUnspecified TransportCause = iota
	TransportCauseTransportResourceUnavailable
)

// ProtocolCause represents protocol-related causes
type ProtocolCause int

const (
	ProtocolCauseTransferSyntaxError ProtocolCause = iota
	ProtocolCauseAbstractSyntaxErrorReject
	ProtocolCauseAbstractSyntaxErrorIgnoreAndNotify
	ProtocolCauseMessageNotCompatibleWithReceiverState
	ProtocolCauseSemanticError
	ProtocolCauseAbstractSyntaxErrorFalselyConstructedMessage
	ProtocolCauseUnspecified
)

// MiscCause represents miscellaneous causes
type MiscCause int

const (
	MiscCauseControlProcessingOverload MiscCause = iota
	MiscCauseHardwareFailure
	MiscCauseOMIntervention
	MiscCauseUnspecified
)

// TimeToWait represents the time to wait
type TimeToWait int

const (
	TimeToWaitV1s TimeToWait = iota
	TimeToWaitV2s
	TimeToWaitV5s
	TimeToWaitV10s
	TimeToWaitV20s
	TimeToWaitV60s
)

// CriticalityDiagnostics represents criticality diagnostics
type CriticalityDiagnostics struct {
	ProcedureCode             *int32                         `json:"procedure_code,omitempty"`
	TriggeringMessage         *TriggeringMessage             `json:"triggering_message,omitempty"`
	ProcedureCriticality      *int                           `json:"procedure_criticality,omitempty"`
	RICRequestorID            *int32                         `json:"ric_requestor_id,omitempty"`
	IEsCriticalityDiagnostics []CriticalityDiagnosticsIEItem `json:"ies_criticality_diagnostics,omitempty"`
}

// TriggeringMessage represents the triggering message
type TriggeringMessage int

const (
	TriggeringMessageInitiatingMessage TriggeringMessage = iota
	TriggeringMessageSuccessfulOutcome
	TriggeringMessageUnsuccessfulOutcome
)

// CriticalityDiagnosticsIEItem represents an IE item in criticality diagnostics
type CriticalityDiagnosticsIEItem struct {
	IECriticality int         `json:"ie_criticality"`
	IEID          int32       `json:"ie_id"`
	TypeOfError   TypeOfError `json:"type_of_error"`
}

// TypeOfError represents the type of error
type TypeOfError int

const (
	TypeOfErrorNotUnderstood TypeOfError = iota
	TypeOfErrorMissing
)

// Additional types for complete E2AP implementation

// E2EventTrigger represents an E2 event trigger
type E2EventTrigger struct {
	InterfaceID        string                 `json:"interface_id"`
	InterfaceDirection InterfaceDirection     `json:"interface_direction"`
	ProcedureCode      int                    `json:"procedure_code"`
	MessageType        int                    `json:"message_type"`
	EventDefinition    map[string]interface{} `json:"event_definition,omitempty"`
}

// InterfaceDirection represents the direction of an interface
type InterfaceDirection int

const (
	InterfaceDirectionIncoming InterfaceDirection = iota
	InterfaceDirectionOutgoing
)

// E2Action represents an E2 action
type E2Action struct {
	ActionID         int                    `json:"action_id"`
	ActionType       string                 `json:"action_type"`
	ActionDefinition map[string]interface{} `json:"action_definition,omitempty"`
	SubsequentAction *E2SubsequentAction    `json:"subsequent_action,omitempty"`
}

// E2SubsequentAction represents a subsequent E2 action
type E2SubsequentAction struct {
	SubsequentActionType string        `json:"subsequent_action_type"`
	TimeToWait           time.Duration `json:"time_to_wait,omitempty"`
}

// E2ControlResponse represents an E2 control response
type E2ControlResponse struct {
	Success        bool                   `json:"success"`
	FailureCause   string                 `json:"failure_cause,omitempty"`
	ControlOutcome map[string]interface{} `json:"control_outcome,omitempty"`
}

// E2ServiceModel represents an E2 service model
type E2ServiceModel struct {
	ServiceModelID      string                 `json:"service_model_id"`
	ServiceModelName    string                 `json:"service_model_name"`
	ServiceModelVersion string                 `json:"service_model_version"`
	ServiceModelOID     string                 `json:"service_model_oid"`
	SupportedProcedures []string               `json:"supported_procedures"`
	Parameters          map[string]interface{} `json:"parameters,omitempty"`
}

// E2NodeFunction represents an E2 node function
type E2NodeFunction struct {
	FunctionID          int                    `json:"function_id"`
	FunctionName        string                 `json:"function_name"`
	FunctionDescription string                 `json:"function_description,omitempty"`
	FunctionOID         string                 `json:"function_oid"`
	FunctionRevision    int                    `json:"function_revision"`
	ServiceModels       []E2ServiceModel       `json:"service_models,omitempty"`
	Parameters          map[string]interface{} `json:"parameters,omitempty"`
}

// E2ConnectionStatus represents the connection status
type E2ConnectionStatus string

const (
	E2ConnectionStatusConnected    E2ConnectionStatus = "CONNECTED"
	E2ConnectionStatusConnecting   E2ConnectionStatus = "CONNECTING"
	E2ConnectionStatusDisconnected E2ConnectionStatus = "DISCONNECTED"
	E2ConnectionStatusError        E2ConnectionStatus = "ERROR"
)

// NodeHealth represents the health status of a node
type NodeHealth struct {
	NodeID       string                  `json:"node_id"`
	Status       string                  `json:"status"`
	LastCheck    time.Time               `json:"last_check"`
	ResponseTime time.Duration           `json:"response_time"`
	FailureCount int                     `json:"failure_count"`
	LastFailure  string                  `json:"last_failure,omitempty"`
	Functions    map[int]*FunctionHealth `json:"functions,omitempty"`
}

// FunctionHealth represents the health of a RAN function
type FunctionHealth struct {
	FunctionID   int           `json:"function_id"`
	Status       string        `json:"status"`
	LastActivity time.Time     `json:"last_activity"`
	MessageCount int64         `json:"message_count"`
	ErrorCount   int64         `json:"error_count"`
	Latency      time.Duration `json:"latency"`
}

// E2Metrics contains E2 interface metrics
type E2Metrics struct {
	NodesConnected      int                     `json:"nodes_connected"`
	TotalMessages       int64                   `json:"total_messages"`
	MessagesPerSecond   float64                 `json:"messages_per_second"`
	SubscriptionsActive int                     `json:"subscriptions_active"`
	IndicationsReceived int64                   `json:"indications_received"`
	ControlRequestsSent int64                   `json:"control_requests_sent"`
	ErrorRate           float64                 `json:"error_rate"`
	AverageLatency      time.Duration           `json:"average_latency"`
	NodeMetrics         map[string]*NodeMetrics `json:"node_metrics,omitempty"`
}

// NodeMetrics contains per-node metrics
type NodeMetrics struct {
	NodeID              string        `json:"node_id"`
	MessagesSent        int64         `json:"messages_sent"`
	MessagesReceived    int64         `json:"messages_received"`
	BytesSent           int64         `json:"bytes_sent"`
	BytesReceived       int64         `json:"bytes_received"`
	ErrorCount          int64         `json:"error_count"`
	LastActivity        time.Time     `json:"last_activity"`
	AverageResponseTime time.Duration `json:"average_response_time"`
}

// E2Indication represents an E2 indication
type E2Indication struct {
	NodeID            string                 `json:"node_id"`
	SubscriptionID    string                 `json:"subscription_id"`
	IndicationType    string                 `json:"indication_type"`
	IndicationHeader  map[string]interface{} `json:"indication_header"`
	IndicationMessage map[string]interface{} `json:"indication_message"`
	Timestamp         time.Time              `json:"timestamp"`
}

// E2Subscription represents an E2 subscription
type E2Subscription struct {
	SubscriptionID string                 `json:"subscription_id"`
	NodeID         string                 `json:"node_id"`
	RequestorID    string                 `json:"requestor_id"`
	InstanceID     int                    `json:"instance_id"`
	RanFunctionID  int                    `json:"ran_function_id"`
	EventTriggers  []E2EventTrigger       `json:"event_triggers"`
	Actions        []E2Action             `json:"actions"`
	Status         string                 `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

// SubscriptionState represents the state of a subscription
type SubscriptionState string

const (
	SubscriptionStatePending   SubscriptionState = "PENDING"
	SubscriptionStateActive    SubscriptionState = "ACTIVE"
	SubscriptionStateSuspended SubscriptionState = "SUSPENDED"
	SubscriptionStateFailed    SubscriptionState = "FAILED"
	SubscriptionStateDeleting  SubscriptionState = "DELETING"
	SubscriptionStateDeleted   SubscriptionState = "DELETED"
)

// SubscriptionHealth represents the health of a subscription
type SubscriptionHealth struct {
	SubscriptionID    string            `json:"subscription_id"`
	State             SubscriptionState `json:"state"`
	LastIndication    time.Time         `json:"last_indication"`
	IndicationCount   int64             `json:"indication_count"`
	ErrorCount        int64             `json:"error_count"`
	ConsecutiveErrors int               `json:"consecutive_errors"`
	HealthScore       float64           `json:"health_score"`
}

// SubscriptionStateTracker tracks subscription states
type SubscriptionStateTracker struct {
	states      map[string]SubscriptionState
	transitions map[string][]StateTransition
	mutex       sync.RWMutex
}

// StateTransition represents a state transition
type StateTransition struct {
	FromState SubscriptionState `json:"from_state"`
	ToState   SubscriptionState `json:"to_state"`
	Timestamp time.Time         `json:"timestamp"`
	Reason    string            `json:"reason,omitempty"`
}

// SubscriptionNotifier manages subscription notifications
type SubscriptionNotifier struct {
	listeners []SubscriptionListener
	mutex     sync.RWMutex
}

// SubscriptionListener interface for subscription events
type SubscriptionListener interface {
	OnSubscriptionStateChange(nodeID, subscriptionID string, oldState, newState SubscriptionState)
	OnSubscriptionError(nodeID, subscriptionID string, err error)
	OnSubscriptionMessage(nodeID, subscriptionID string, indication *E2Indication)
}

// E2HealthMonitor monitors E2 node health
type E2HealthMonitor struct {
	nodeHealth         map[string]*NodeHealth
	subscriptionHealth map[string]map[string]*SubscriptionHealth
	checkInterval      time.Duration
	mutex              sync.RWMutex
	stopChan           chan struct{}
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	CAFile             string `json:"ca_file,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
}
