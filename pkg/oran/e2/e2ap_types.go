package e2

import (
	"time"
)

// E2AP Additional Type Definitions following O-RAN.WG3.E2AP-v03.01

// E2NodeType and constants are now defined in e2ap_messages.go to avoid duplication

// RANFunctionItem is now defined in e2ap_messages.go to avoid duplication

// RANFunctionIDItem is now defined in e2ap_messages.go to avoid duplication

// RANFunctionIDCause is now defined in e2ap_messages.go to avoid duplication

// E2NodeComponentConfigurationItem is now defined in e2ap_messages.go to avoid duplication

// E2NodeComponentInterfaceType and constants are now defined in e2ap_messages.go to avoid duplication

// E2NodeComponentID is now defined in e2ap_messages.go to avoid duplication

// RICSubscriptionDetails is now defined in e2ap_messages.go to avoid duplication

// RICActionToBeSetup represents an action to be setup
type RICActionToBeSetup struct {
	ActionID         int                  `json:"action_id"`
	ActionType       RICActionType        `json:"action_type"`
	ActionDefinition []byte               `json:"action_definition,omitempty"`
	SubsequentAction *RICSubsequentAction `json:"subsequent_action,omitempty"`
}

// RICActionToBeSetupItem is now defined in e2ap_messages.go to avoid duplication

// RICActionType and constants are now defined in e2ap_messages.go to avoid duplication

// RICSubsequentAction is now defined in e2ap_messages.go to avoid duplication

// RICSubsequentActionType and constants are now defined in e2ap_messages.go to avoid duplication

// RICTimeToWait and constants are now defined in e2ap_messages.go to avoid duplication

// RICActionNotAdmittedItem is now defined in e2ap_messages.go to avoid duplication

// RICActionAdmittedItem is now defined in e2ap_messages.go to avoid duplication

// E2Cause is now defined in e2ap_messages.go to avoid duplication

// E2CauseType and constants are now defined in e2ap_messages.go to avoid duplication

// RICServiceCause and constants are now defined in e2ap_messages.go to avoid duplication

// E2NodeCause and constants are now defined in e2ap_messages.go to avoid duplication

// TransportCause and constants are now defined in e2ap_messages.go to avoid duplication

// ProtocolCause and constants are now defined in e2ap_messages.go to avoid duplication

// MiscCause and constants are now defined in e2ap_messages.go to avoid duplication

// TimeToWait and constants are now defined in e2ap_messages.go to avoid duplication

// CriticalityDiagnostics is now defined in e2ap_messages.go to avoid duplication

// TriggeringMessage and constants are now defined in e2ap_messages.go to avoid duplication

// CriticalityDiagnosticsIEItem is now defined in e2ap_messages.go to avoid duplication

// TypeOfError and constants are now defined in e2ap_messages.go to avoid duplication

// Additional types for complete E2AP implementation

// E2EventTrigger is defined in e2_adaptor.go to avoid duplication

// InterfaceDirection is defined in e2_adaptor.go to avoid duplication

// E2Action is defined in e2_adaptor.go to avoid duplication

// E2SubsequentAction is defined in e2_adaptor.go to avoid duplication

// E2ControlResponse is defined in e2_adaptor.go to avoid duplication

// E2ServiceModel is defined in e2_adaptor.go to avoid duplication

// E2NodeFunction is defined in e2_adaptor.go to avoid duplication

// E2ConnectionStatus is defined in e2_adaptor.go to avoid duplication

// NodeHealth is defined in e2_manager.go to avoid duplication

// FunctionHealth is defined in e2_manager.go to avoid duplication

// E2Metrics is defined in e2_manager.go to avoid duplication

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

// E2Indication is defined in e2_adaptor.go to avoid duplication

// E2Subscription is defined in e2_adaptor.go to avoid duplication

// SubscriptionState is defined in e2_manager.go to avoid duplication

// SubscriptionHealth is defined in e2_manager.go to avoid duplication

// SubscriptionStateTracker is defined in e2_manager.go to avoid duplication

// StateTransition is defined in e2_manager.go to avoid duplication

// SubscriptionNotifier is defined in e2_manager.go to avoid duplication

// SubscriptionListener is defined in e2_manager.go to avoid duplication

// E2HealthMonitor is defined in e2_manager.go to avoid duplication

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled            bool   `json:"enabled"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
	CAFile             string `json:"ca_file,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
}
