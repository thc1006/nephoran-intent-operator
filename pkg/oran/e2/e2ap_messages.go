package e2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

// E2AP Protocol Data Units (PDUs) following O-RAN.WG3.E2AP-v3.0.

// E2APMessageType represents the type of E2AP message.

type E2APMessageType int

const (

	// Initiating messages.

	E2APMessageTypeSetupRequest E2APMessageType = 1

	// E2APMessageTypeRICSubscriptionRequest holds e2apmessagetypericsubscriptionrequest value.

	E2APMessageTypeRICSubscriptionRequest E2APMessageType = 2

	// E2APMessageTypeRICControlRequest holds e2apmessagetypericcontrolrequest value.

	E2APMessageTypeRICControlRequest E2APMessageType = 3

	// E2APMessageTypeErrorIndication holds e2apmessagetypeerrorindication value.

	E2APMessageTypeErrorIndication E2APMessageType = 4

	// E2APMessageTypeRICSubscriptionDeleteRequest holds e2apmessagetypericsubscriptiondeleterequest value.

	E2APMessageTypeRICSubscriptionDeleteRequest E2APMessageType = 5

	// E2APMessageTypeResetRequest holds e2apmessagetyperesetrequest value.

	E2APMessageTypeResetRequest E2APMessageType = 6

	// Successful outcome messages.

	E2APMessageTypeSetupResponse E2APMessageType = 11

	// E2APMessageTypeRICSubscriptionResponse holds e2apmessagetypericsubscriptionresponse value.

	E2APMessageTypeRICSubscriptionResponse E2APMessageType = 12

	// E2APMessageTypeRICControlAcknowledge holds e2apmessagetypericcontrolacknowledge value.

	E2APMessageTypeRICControlAcknowledge E2APMessageType = 13

	// E2APMessageTypeRICSubscriptionDeleteResponse holds e2apmessagetypericsubscriptiondeleteresponse value.

	E2APMessageTypeRICSubscriptionDeleteResponse E2APMessageType = 14

	// E2APMessageTypeResetResponse holds e2apmessagetyperesetresponse value.

	E2APMessageTypeResetResponse E2APMessageType = 15

	// Unsuccessful outcome messages.

	E2APMessageTypeSetupFailure E2APMessageType = 21

	// E2APMessageTypeRICSubscriptionFailure holds e2apmessagetypericsubscriptionfailure value.

	E2APMessageTypeRICSubscriptionFailure E2APMessageType = 22

	// E2APMessageTypeRICControlFailure holds e2apmessagetypericcontrolfailure value.

	E2APMessageTypeRICControlFailure E2APMessageType = 23

	// E2APMessageTypeRICSubscriptionDeleteFailure holds e2apmessagetypericsubscriptiondeletefailure value.

	E2APMessageTypeRICSubscriptionDeleteFailure E2APMessageType = 24

	// E2APMessageTypeResetFailure holds e2apmessagetyperesetfailure value.

	E2APMessageTypeResetFailure E2APMessageType = 25

	// Indication messages.

	E2APMessageTypeRICIndication E2APMessageType = 31

	// E2APMessageTypeRICServiceUpdate holds e2apmessagetypericserviceupdate value.

	E2APMessageTypeRICServiceUpdate E2APMessageType = 32

	// E2APMessageTypeRICServiceUpdateAcknowledge holds e2apmessagetypericserviceupdateacknowledge value.

	E2APMessageTypeRICServiceUpdateAcknowledge E2APMessageType = 33

	// E2APMessageTypeRICServiceUpdateFailure holds e2apmessagetypericserviceupdatefailure value.

	E2APMessageTypeRICServiceUpdateFailure E2APMessageType = 34
)

// E2APMessage represents a generic E2AP message structure.

type E2APMessage struct {
	MessageType E2APMessageType `json:"message_type"`

	TransactionID int32 `json:"transaction_id"`

	ProcedureCode int32 `json:"procedure_code"`

	Criticality Criticality `json:"criticality"`

	Payload interface{} `json:"payload"`

	Timestamp time.Time `json:"timestamp"`

	CorrelationID string `json:"correlation_id,omitempty"`
}

// Criticality represents the criticality of E2AP messages.

type Criticality int

const (

	// CriticalityReject holds criticalityreject value.

	CriticalityReject Criticality = iota

	// CriticalityIgnore holds criticalityignore value.

	CriticalityIgnore

	// CriticalityNotify holds criticalitynotify value.

	CriticalityNotify
)

// E2APEncoder handles encoding/decoding of E2AP messages.

type E2APEncoder struct {
	messageRegistry map[E2APMessageType]MessageCodec

	correlationMap map[string]*PendingMessage

	mutex sync.RWMutex
}

// MessageCodec interface for encoding/decoding specific message types.

type MessageCodec interface {
	Encode(message interface{}) ([]byte, error)

	Decode(data []byte) (interface{}, error)

	GetMessageType() E2APMessageType

	Validate(message interface{}) error
}

// PendingMessage tracks pending E2AP transactions.

type PendingMessage struct {
	MessageType E2APMessageType

	TransactionID int32

	SentTime time.Time

	Timeout time.Duration

	Callback func(response *E2APMessage, err error)
}

// E2SetupRequest represents E2 Setup Request message.

type E2SetupRequest struct {
	GlobalE2NodeID GlobalE2NodeID `json:"global_e2_node_id"`

	RANFunctionsList []RANFunctionItem `json:"ran_functions_list"`

	E2NodeComponentConfigurationList []E2NodeComponentConfigurationItem `json:"e2_node_component_config_list,omitempty"`
}

// E2SetupResponse represents E2 Setup Response message.

type E2SetupResponse struct {
	GlobalRICID GlobalRICID `json:"global_ric_id"`

	RANFunctionsAccepted []RANFunctionIDItem `json:"ran_functions_accepted,omitempty"`

	RANFunctionsRejected []RANFunctionIDCause `json:"ran_functions_rejected,omitempty"`

	E2NodeComponentConfigurationList []E2NodeComponentConfigurationItem `json:"e2_node_component_config_list,omitempty"`
}

// E2SetupFailure represents E2 Setup Failure message.

type E2SetupFailure struct {
	Cause E2APCause `json:"cause"`

	TimeToWait *TimeToWait `json:"time_to_wait,omitempty"`

	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// RICSubscriptionRequest represents RIC Subscription Request message.

type RICSubscriptionRequest struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`

	RICSubscriptionDetails RICSubscriptionDetails `json:"ric_subscription_details"`
}

// RICSubscriptionResponse represents RIC Subscription Response message.

type RICSubscriptionResponse struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`

	RICActionAdmittedList []RICActionID `json:"ric_action_admitted_list,omitempty"`

	RICActionNotAdmittedList []RICActionNotAdmittedItem `json:"ric_action_not_admitted_list,omitempty"`
}

// RICSubscriptionFailure represents RIC Subscription Failure message.

type RICSubscriptionFailure struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`

	Cause E2APCause `json:"cause"`

	RICActionNotAdmittedList []RICActionNotAdmittedItem `json:"ric_action_not_admitted_list,omitempty"`

	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// RICControlRequest represents RIC Control Request message.

type RICControlRequest struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`

	RICCallProcessID *RICCallProcessID `json:"ric_call_process_id,omitempty"`

	RICControlHeader []byte `json:"ric_control_header"`

	RICControlMessage []byte `json:"ric_control_message"`

	RICControlAckRequest *RICControlAckRequest `json:"ric_control_ack_request,omitempty"`
}

// RICControlAcknowledge represents RIC Control Acknowledge message.

type RICControlAcknowledge struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`

	RICCallProcessID *RICCallProcessID `json:"ric_call_process_id,omitempty"`

	RICControlOutcome []byte `json:"ric_control_outcome,omitempty"`
}

// RICControlFailure represents RIC Control Failure message.

type RICControlFailure struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`

	RICCallProcessID *RICCallProcessID `json:"ric_call_process_id,omitempty"`

	Cause E2APCause `json:"cause"`

	RICControlOutcome []byte `json:"ric_control_outcome,omitempty"`

	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// RICIndication represents RIC Indication message.

type RICIndication struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`

	RICActionID RICActionID `json:"ric_action_id"`

	RICIndicationSN *RICIndicationSN `json:"ric_indication_sn,omitempty"`

	RICIndicationType RICIndicationType `json:"ric_indication_type"`

	RICIndicationHeader []byte `json:"ric_indication_header"`

	RICIndicationMessage []byte `json:"ric_indication_message"`

	RICCallProcessID *RICCallProcessID `json:"ric_call_process_id,omitempty"`
}

// Common E2AP data types.

// GlobalE2NodeID represents the global E2 node identifier.

type GlobalE2NodeID struct {
	PLMNIdentity PLMNIdentity `json:"plmn_identity"`

	NodeType E2NodeType `json:"node_type"`

	NodeID string `json:"node_id"`

	E2NodeID E2NodeID `json:"e2_node_id"`
}

// GlobalRICID represents the global RIC identifier.

type GlobalRICID struct {
	PLMNIdentity PLMNIdentity `json:"plmn_identity"`

	RICID RICID `json:"ric_id"`
}

// PLMNIdentity represents the PLMN (Public Land Mobile Network) identity.

type PLMNIdentity struct {
	MCC string `json:"mcc"` // Mobile Country Code (3 digits)

	MNC string `json:"mnc"` // Mobile Network Code (2-3 digits)
}

// E2NodeID represents the E2 node identifier (choice).

type E2NodeID struct {
	GNBID *GNBID `json:"gnb_id,omitempty"`

	ENBID *ENBID `json:"enb_id,omitempty"`

	EnGNBID *EnGNBID `json:"en_gnb_id,omitempty"`

	NgENBID *NgENBID `json:"ng_enb_id,omitempty"`
}

// GNBID represents gNB identifier.

type GNBID struct {
	GNBIDChoice GNBIDChoice `json:"gnb_id_choice"`
}

// GNBIDChoice represents the choice of gNB ID.

type GNBIDChoice struct {
	GNBID22 *string `json:"gnb_id_22,omitempty"` // 22-bit string

	GNBID32 *string `json:"gnb_id_32,omitempty"` // 32-bit string
}

// ENBID represents eNB identifier.

type ENBID struct {
	MacroENBID *string `json:"macro_enb_id,omitempty"` // 20-bit string

	HomeENBID *string `json:"home_enb_id,omitempty"` // 28-bit string

	ShortMacro *string `json:"short_macro,omitempty"` // 18-bit string

	LongMacro *string `json:"long_macro,omitempty"` // 21-bit string
}

// EnGNBID represents en-gNB identifier.

type EnGNBID struct {
	EnGNBID string `json:"en_gnb_id"` // 22-bit string
}

// NgENBID represents ng-eNB identifier.

type NgENBID struct {
	MacroNgENBID *string `json:"macro_ng_enb_id,omitempty"` // 20-bit string

	ShortMacro *string `json:"short_macro,omitempty"` // 18-bit string

	LongMacro *string `json:"long_macro,omitempty"` // 21-bit string
}

// RICID represents RIC identifier (20-bit).

type RICID int

// RICRequestID represents RIC request identifier.

type RICRequestID struct {
	RICRequestorID RICRequestorID `json:"ric_requestor_id"`

	RICInstanceID RICInstanceID `json:"ric_instance_id"`
}

// RICRequestorID represents RIC requestor identifier (0..65535).

type RICRequestorID int32

// RICInstanceID represents RIC instance identifier (0..65535).

type RICInstanceID int32

// RANFunctionID represents RAN function identifier (0..4095).

type RANFunctionID int32

// RICActionID represents RIC action identifier (0..255).

type RICActionID int32

// RICCallProcessID represents RIC call process identifier.

type RICCallProcessID []byte

// RICIndicationSN represents RIC indication sequence number (0..65535).

type RICIndicationSN int32

// RICIndicationType represents RIC indication type.

type RICIndicationType int

const (

	// RICIndicationTypeReport holds ricindicationtypereport value.

	RICIndicationTypeReport RICIndicationType = iota

	// RICIndicationTypeInsert holds ricindicationtypeinsert value.

	RICIndicationTypeInsert
)

// RICControlAckRequest represents RIC control acknowledgment request.

type RICControlAckRequest int

const (

	// RICControlAckRequestNoAck holds riccontrolackrequestnoack value.

	RICControlAckRequestNoAck RICControlAckRequest = iota

	// RICControlAckRequestAck holds riccontrolackrequestack value.

	RICControlAckRequestAck

	// RICControlAckRequestNAck holds riccontrolackrequestnack value.

	RICControlAckRequestNAck
)

// E2APCause represents E2AP cause.

type E2APCause struct {
	RICCause *RICCause `json:"ric_cause,omitempty"`

	RICServiceCause *RICServiceCause `json:"ric_service_cause,omitempty"`

	E2NodeCause *E2NodeCause `json:"e2_node_cause,omitempty"`

	TransportCause *TransportCause `json:"transport_cause,omitempty"`

	ProtocolCause *ProtocolCause `json:"protocol_cause,omitempty"`

	MiscCause *MiscCause `json:"misc_cause,omitempty"`
}

// RICCause represents RIC-related causes.

type RICCause int

const (

	// RICCauseRANFunctionIDInvalid holds riccauseranfunctionidinvalid value.

	RICCauseRANFunctionIDInvalid RICCause = iota

	// RICCauseActionNotSupported holds riccauseactionnotsupported value.

	RICCauseActionNotSupported

	// RICCauseExcessiveActions holds riccauseexcessiveactions value.

	RICCauseExcessiveActions

	// RICCauseDuplicateAction holds riccauseduplicateaction value.

	RICCauseDuplicateAction

	// RICCauseDuplicateRICRequestID holds riccauseduplicatericrequestid value.

	RICCauseDuplicateRICRequestID

	// RICCauseFunctionResourceLimit holds riccausefunctionresourcelimit value.

	RICCauseFunctionResourceLimit

	// RICCauseRequestIDUnknown holds riccauserequestidunknown value.

	RICCauseRequestIDUnknown

	// RICCauseInconsistentActionSubsequentActionSequence holds riccauseinconsistentactionsubsequentactionsequence value.

	RICCauseInconsistentActionSubsequentActionSequence

	// RICCauseControlMessageInvalid holds riccausecontrolmessageinvalid value.

	RICCauseControlMessageInvalid

	// RICCauseRICCallProcessIDInvalid holds riccausericcallprocessidinvalid value.

	RICCauseRICCallProcessIDInvalid

	// RICCauseControlTimerExpired holds riccausecontroltimerexpired value.

	RICCauseControlTimerExpired

	// RICCauseControlFailedToExecute holds riccausecontrolfailedtoexecute value.

	RICCauseControlFailedToExecute

	// RICCauseSystemNotReady holds riccausesystemnotready value.

	RICCauseSystemNotReady

	// RICCauseUnspecified holds riccauseunspecified value.

	RICCauseUnspecified
)

// RICServiceCause represents RIC service-related causes.

type RICServiceCause int

const (

	// RICServiceCauseRANFunctionNotRequired holds ricservicecauseranfunctionnotrequired value.

	RICServiceCauseRANFunctionNotRequired RICServiceCause = iota

	// RICServiceCauseExcessiveFunctions holds ricservicecauseexcessivefunctions value.

	RICServiceCauseExcessiveFunctions

	// RICServiceCauseRICResourceLimit holds ricservicecausericresourcelimit value.

	RICServiceCauseRICResourceLimit
)

// E2NodeCause represents E2 node-related causes.

type E2NodeCause int

const (

	// E2NodeCauseE2NodeComponentUnknown holds e2nodecausee2nodecomponentunknown value.

	E2NodeCauseE2NodeComponentUnknown E2NodeCause = iota
)

// TransportCause represents transport-related causes.

type TransportCause int

const (

	// TransportCauseUnspecified holds transportcauseunspecified value.

	TransportCauseUnspecified TransportCause = iota

	// TransportCauseTransportResourceUnavailable holds transportcausetransportresourceunavailable value.

	TransportCauseTransportResourceUnavailable
)

// ProtocolCause represents protocol-related causes.

type ProtocolCause int

const (

	// ProtocolCauseTransferSyntaxError holds protocolcausetransfersyntaxerror value.

	ProtocolCauseTransferSyntaxError ProtocolCause = iota

	// ProtocolCauseAbstractSyntaxErrorReject holds protocolcauseabstractsyntaxerrorreject value.

	ProtocolCauseAbstractSyntaxErrorReject

	// ProtocolCauseAbstractSyntaxErrorIgnoreAndNotify holds protocolcauseabstractsyntaxerrorignoreandnotify value.

	ProtocolCauseAbstractSyntaxErrorIgnoreAndNotify

	// ProtocolCauseMessageNotCompatibleWithReceiverState holds protocolcausemessagenotcompatiblewithreceiverstate value.

	ProtocolCauseMessageNotCompatibleWithReceiverState

	// ProtocolCauseSemanticError holds protocolcausesemanticerror value.

	ProtocolCauseSemanticError

	// ProtocolCauseAbstractSyntaxErrorFalselyConstructedMessage holds protocolcauseabstractsyntaxerrorfalselyconstructedmessage value.

	ProtocolCauseAbstractSyntaxErrorFalselyConstructedMessage

	// ProtocolCauseUnspecified holds protocolcauseunspecified value.

	ProtocolCauseUnspecified
)

// MiscCause represents miscellaneous causes.

type MiscCause int

const (

	// MiscCauseControlProcessingOverload holds misccausecontrolprocessingoverload value.

	MiscCauseControlProcessingOverload MiscCause = iota

	// MiscCauseNotEnoughUserPlaneProcessingResources holds misccausenotenoughuserplaneprocessingresources value.

	MiscCauseNotEnoughUserPlaneProcessingResources

	// MiscCauseHardwareFailure holds misccausehardwarefailure value.

	MiscCauseHardwareFailure

	// MiscCauseOMIntervention holds misccauseomintervention value.

	MiscCauseOMIntervention

	// MiscCauseUnspecified holds misccauseunspecified value.

	MiscCauseUnspecified
)

// TimeToWait represents time to wait before retry.

type TimeToWait int

const (

	// TimeToWaitV1s holds timetowaitv1s value.

	TimeToWaitV1s TimeToWait = iota

	// TimeToWaitV2s holds timetowaitv2s value.

	TimeToWaitV2s

	// TimeToWaitV5s holds timetowaitv5s value.

	TimeToWaitV5s

	// TimeToWaitV10s holds timetowaitv10s value.

	TimeToWaitV10s

	// TimeToWaitV20s holds timetowaitv20s value.

	TimeToWaitV20s

	// TimeToWaitV60s holds timetowaitv60s value.

	TimeToWaitV60s
)

// CriticalityDiagnostics represents criticality diagnostics information.

type CriticalityDiagnostics struct {
	ProcedureCode *int32 `json:"procedure_code,omitempty"`

	TriggeringMessage *TriggeringMessage `json:"triggering_message,omitempty"`

	ProcedureCriticality *Criticality `json:"procedure_criticality,omitempty"`

	IEsCriticalityDiagnostics []CriticalityDiagnosticsIEItem `json:"ies_criticality_diagnostics,omitempty"`
}

// TriggeringMessage represents the triggering message type.

type TriggeringMessage int

const (

	// TriggeringMessageInitiatingMessage holds triggeringmessageinitiatingmessage value.

	TriggeringMessageInitiatingMessage TriggeringMessage = iota

	// TriggeringMessageSuccessfulOutcome holds triggeringmessagesuccessfuloutcome value.

	TriggeringMessageSuccessfulOutcome

	// TriggeringMessageUnsuccessfulOutcome holds triggeringmessageunsuccessfuloutcome value.

	TriggeringMessageUnsuccessfulOutcome
)

// CriticalityDiagnosticsIEItem represents IE-specific criticality diagnostics.

type CriticalityDiagnosticsIEItem struct {
	IECriticality Criticality `json:"ie_criticality"`

	IEID int32 `json:"ie_id"`

	TypeOfError TypeOfError `json:"type_of_error"`
}

// TypeOfError represents the type of error in criticality diagnostics.

type TypeOfError int

const (

	// TypeOfErrorNotUnderstood holds typeoferrornotunderstood value.

	TypeOfErrorNotUnderstood TypeOfError = iota

	// TypeOfErrorMissing holds typeoferrormissing value.

	TypeOfErrorMissing
)

// RANFunctionItem represents a RAN function item.

type RANFunctionItem struct {
	RANFunctionID RANFunctionID `json:"ran_function_id"`

	RANFunctionDefinition []byte `json:"ran_function_definition"`

	RANFunctionRevision RANFunctionRevision `json:"ran_function_revision"`

	RANFunctionOID *string `json:"ran_function_oid,omitempty"`
}

// RANFunctionRevision represents RAN function revision (0..255).

type RANFunctionRevision int32

// RANFunctionIDItem represents RAN function ID item.

type RANFunctionIDItem struct {
	RANFunctionID RANFunctionID `json:"ran_function_id"`

	RANFunctionRevision RANFunctionRevision `json:"ran_function_revision"`
}

// RANFunctionIDCause represents RAN function ID with cause.

type RANFunctionIDCause struct {
	RANFunctionID RANFunctionID `json:"ran_function_id"`

	Cause E2APCause `json:"cause"`
}

// RICSubscriptionDetails represents RIC subscription details.

type RICSubscriptionDetails struct {
	RICEventTriggerDefinition []byte `json:"ric_event_trigger_definition"`

	RICActionToBeSetupList []RICActionToBeSetupItem `json:"ric_action_to_be_setup_list"`
}

// RICActionToBeSetupItem represents a RIC action to be setup.

type RICActionToBeSetupItem struct {
	RICActionID RICActionID `json:"ric_action_id"`

	RICActionType RICActionType `json:"ric_action_type"`

	RICActionDefinition []byte `json:"ric_action_definition,omitempty"`

	RICSubsequentAction *RICSubsequentAction `json:"ric_subsequent_action,omitempty"`
}

// RICActionType represents the type of RIC action.

type RICActionType int

const (

	// RICActionTypeReport holds ricactiontypereport value.

	RICActionTypeReport RICActionType = iota

	// RICActionTypeInsert holds ricactiontypeinsert value.

	RICActionTypeInsert

	// RICActionTypePolicy holds ricactiontypepolicy value.

	RICActionTypePolicy
)

// RICSubsequentAction represents subsequent action for RIC action.

type RICSubsequentAction struct {
	RICSubsequentActionType RICSubsequentActionType `json:"ric_subsequent_action_type"`

	RICTimeToWait *RICTimeToWait `json:"ric_time_to_wait,omitempty"`
}

// RICSubsequentActionType represents the type of subsequent action.

type RICSubsequentActionType int

const (

	// RICSubsequentActionTypeContinue holds ricsubsequentactiontypecontinue value.

	RICSubsequentActionTypeContinue RICSubsequentActionType = iota

	// RICSubsequentActionTypeWait holds ricsubsequentactiontypewait value.

	RICSubsequentActionTypeWait
)

// RICTimeToWait represents RIC time to wait.

type RICTimeToWait int

const (

	// RICTimeToWaitZero holds rictimetowaitzero value.

	RICTimeToWaitZero RICTimeToWait = iota

	// RICTimeToWaitW1ms holds rictimetowaitw1ms value.

	RICTimeToWaitW1ms

	// RICTimeToWaitW2ms holds rictimetowaitw2ms value.

	RICTimeToWaitW2ms

	// RICTimeToWaitW5ms holds rictimetowaitw5ms value.

	RICTimeToWaitW5ms

	// RICTimeToWaitW10ms holds rictimetowaitw10ms value.

	RICTimeToWaitW10ms

	// RICTimeToWaitW20ms holds rictimetowaitw20ms value.

	RICTimeToWaitW20ms

	// RICTimeToWaitW50ms holds rictimetowaitw50ms value.

	RICTimeToWaitW50ms

	// RICTimeToWaitW100ms holds rictimetowaitw100ms value.

	RICTimeToWaitW100ms

	// RICTimeToWaitW200ms holds rictimetowaitw200ms value.

	RICTimeToWaitW200ms

	// RICTimeToWaitW500ms holds rictimetowaitw500ms value.

	RICTimeToWaitW500ms

	// RICTimeToWaitW1s holds rictimetowaitw1s value.

	RICTimeToWaitW1s

	// RICTimeToWaitW2s holds rictimetowaitw2s value.

	RICTimeToWaitW2s

	// RICTimeToWaitW5s holds rictimetowaitw5s value.

	RICTimeToWaitW5s

	// RICTimeToWaitW10s holds rictimetowaitw10s value.

	RICTimeToWaitW10s

	// RICTimeToWaitW20s holds rictimetowaitw20s value.

	RICTimeToWaitW20s

	// RICTimeToWaitW60s holds rictimetowaitw60s value.

	RICTimeToWaitW60s
)

// RICActionNotAdmittedItem represents a RIC action that was not admitted.

type RICActionNotAdmittedItem struct {
	RICActionID RICActionID `json:"ric_action_id"`

	Cause E2APCause `json:"cause"`
}

// E2NodeComponentConfigurationItem represents E2 node component configuration.

type E2NodeComponentConfigurationItem struct {
	E2NodeComponentInterfaceType E2NodeComponentInterfaceType `json:"e2_node_component_interface_type"`

	E2NodeComponentID E2NodeComponentID `json:"e2_node_component_id"`

	E2NodeComponentConfiguration E2NodeComponentConfiguration `json:"e2_node_component_configuration"`
}

// E2NodeComponentInterfaceType represents the interface type.

type E2NodeComponentInterfaceType int

const (

	// E2NodeComponentInterfaceTypeNG holds e2nodecomponentinterfacetypeng value.

	E2NodeComponentInterfaceTypeNG E2NodeComponentInterfaceType = iota

	// E2NodeComponentInterfaceTypeXn holds e2nodecomponentinterfacetypexn value.

	E2NodeComponentInterfaceTypeXn

	// E2NodeComponentInterfaceTypeE1 holds e2nodecomponentinterfacetypee1 value.

	E2NodeComponentInterfaceTypeE1

	// E2NodeComponentInterfaceTypeF1 holds e2nodecomponentinterfacetypef1 value.

	E2NodeComponentInterfaceTypeF1

	// E2NodeComponentInterfaceTypeW1 holds e2nodecomponentinterfacetypew1 value.

	E2NodeComponentInterfaceTypeW1

	// E2NodeComponentInterfaceTypeS1 holds e2nodecomponentinterfacetypes1 value.

	E2NodeComponentInterfaceTypeS1

	// E2NodeComponentInterfaceTypeX2 holds e2nodecomponentinterfacetypex2 value.

	E2NodeComponentInterfaceTypeX2
)

// E2NodeComponentID represents E2 node component identifier (choice).

type E2NodeComponentID struct {
	E2NodeComponentInterfaceTypeNG *E2NodeComponentInterfaceNG `json:"ng,omitempty"`

	E2NodeComponentInterfaceTypeXn *E2NodeComponentInterfaceXn `json:"xn,omitempty"`

	E2NodeComponentInterfaceTypeE1 *E2NodeComponentInterfaceE1 `json:"e1,omitempty"`

	E2NodeComponentInterfaceTypeF1 *E2NodeComponentInterfaceF1 `json:"f1,omitempty"`

	E2NodeComponentInterfaceTypeW1 *E2NodeComponentInterfaceW1 `json:"w1,omitempty"`

	E2NodeComponentInterfaceTypeS1 *E2NodeComponentInterfaceS1 `json:"s1,omitempty"`

	E2NodeComponentInterfaceTypeX2 *E2NodeComponentInterfaceX2 `json:"x2,omitempty"`
}

// Interface-specific component IDs (simplified for HTTP transport).

type E2NodeComponentInterfaceNG struct {
	AMFName string `json:"amf_name"`
}

// E2NodeComponentInterfaceXn represents a e2nodecomponentinterfacexn.

type E2NodeComponentInterfaceXn struct {
	GlobalNGRANNodeID string `json:"global_ng_ran_node_id"`
}

// E2NodeComponentInterfaceE1 represents a e2nodecomponentinterfacee1.

type E2NodeComponentInterfaceE1 struct {
	GNBCUCPID string `json:"gnb_cu_cp_id"`
}

// E2NodeComponentInterfaceF1 represents a e2nodecomponentinterfacef1.

type E2NodeComponentInterfaceF1 struct {
	GNBDUID string `json:"gnb_du_id"`
}

// E2NodeComponentInterfaceW1 represents a e2nodecomponentinterfacew1.

type E2NodeComponentInterfaceW1 struct {
	NGENBDUID string `json:"ng_enb_du_id"`
}

// E2NodeComponentInterfaceS1 represents a e2nodecomponentinterfaces1.

type E2NodeComponentInterfaceS1 struct {
	MMEName string `json:"mme_name"`
}

// E2NodeComponentInterfaceX2 represents a e2nodecomponentinterfacex2.

type E2NodeComponentInterfaceX2 struct {
	ENBID string `json:"enb_id"`
}

// E2NodeComponentConfiguration represents E2 node component configuration.

type E2NodeComponentConfiguration struct {
	E2NodeComponentRequestPart []byte `json:"e2_node_component_request_part"`

	E2NodeComponentResponsePart []byte `json:"e2_node_component_response_part"`
}

// NewE2APEncoder creates a new E2AP encoder with registered codecs.

func NewE2APEncoder() *E2APEncoder {
	encoder := &E2APEncoder{
		messageRegistry: make(map[E2APMessageType]MessageCodec),

		correlationMap: make(map[string]*PendingMessage),
	}

	// Register standard message codecs.

	encoder.registerCodec(&E2SetupRequestCodec{})

	encoder.registerCodec(&E2SetupResponseCodec{})

	encoder.registerCodec(&E2SetupFailureCodec{})

	encoder.registerCodec(&RICSubscriptionRequestCodec{})

	encoder.registerCodec(&RICSubscriptionResponseCodec{})

	encoder.registerCodec(&RICSubscriptionFailureCodec{})

	encoder.registerCodec(&RICControlRequestCodec{})

	encoder.registerCodec(&RICControlAcknowledgeCodec{})

	encoder.registerCodec(&RICControlFailureCodec{})

	encoder.registerCodec(&RICIndicationCodec{})

	encoder.registerCodec(&RICServiceUpdateCodec{})

	encoder.registerCodec(&RICServiceUpdateAcknowledgeCodec{})

	encoder.registerCodec(&RICServiceUpdateFailureCodec{})

	encoder.registerCodec(&RICSubscriptionDeleteRequestCodec{})

	encoder.registerCodec(&RICSubscriptionDeleteResponseCodec{})

	encoder.registerCodec(&RICSubscriptionDeleteFailureCodec{})

	encoder.registerCodec(&ResetRequestCodec{})

	encoder.registerCodec(&ResetResponseCodec{})

	encoder.registerCodec(&ErrorIndicationCodec{})

	return encoder
}

// registerCodec registers a message codec.

func (e *E2APEncoder) registerCodec(codec MessageCodec) {
	e.messageRegistry[codec.GetMessageType()] = codec
}

// EncodeMessage encodes an E2AP message to bytes.

func (e *E2APEncoder) EncodeMessage(message *E2APMessage) ([]byte, error) {
	codec, exists := e.messageRegistry[message.MessageType]

	if !exists {
		return nil, fmt.Errorf("no codec registered for message type %d", message.MessageType)
	}

	// Validate message.

	if err := codec.Validate(message.Payload); err != nil {
		return nil, fmt.Errorf("message validation failed: %w", err)
	}

	// Encode payload.

	payloadBytes, err := codec.Encode(message.Payload)
	if err != nil {
		return nil, fmt.Errorf("payload encoding failed: %w", err)
	}

	// Create message header.

	header := make([]byte, 16) // Fixed header size for HTTP transport

	binary.BigEndian.PutUint32(header[0:4], uint32(message.MessageType))

	binary.BigEndian.PutUint32(header[4:8], uint32(message.TransactionID))

	binary.BigEndian.PutUint32(header[8:12], uint32(message.ProcedureCode))

	binary.BigEndian.PutUint32(header[12:16], uint32(message.Criticality))

	// Combine header and payload.

	buf := bytes.NewBuffer(header)

	buf.Write(payloadBytes)

	return buf.Bytes(), nil
}

// DecodeMessage decodes bytes to an E2AP message.

func (e *E2APEncoder) DecodeMessage(data []byte) (*E2APMessage, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("message too short: expected at least 16 bytes, got %d", len(data))
	}

	// Parse header.

	messageType := E2APMessageType(binary.BigEndian.Uint32(data[0:4]))

	transactionID := int32(binary.BigEndian.Uint32(data[4:8]))

	procedureCode := int32(binary.BigEndian.Uint32(data[8:12]))

	criticality := Criticality(binary.BigEndian.Uint32(data[12:16]))

	// Get codec for message type.

	codec, exists := e.messageRegistry[messageType]

	if !exists {
		return nil, fmt.Errorf("no codec registered for message type %d", messageType)
	}

	// Decode payload.

	payload, err := codec.Decode(data[16:])
	if err != nil {
		return nil, fmt.Errorf("payload decoding failed: %w", err)
	}

	return &E2APMessage{
		MessageType: messageType,

		TransactionID: transactionID,

		ProcedureCode: procedureCode,

		Criticality: criticality,

		Payload: payload,

		Timestamp: time.Now(),
	}, nil
}

// TrackMessage tracks a message for correlation.

func (e *E2APEncoder) TrackMessage(correlationID string, message *PendingMessage) {
	e.mutex.Lock()

	defer e.mutex.Unlock()

	e.correlationMap[correlationID] = message
}

// GetPendingMessage retrieves a pending message by correlation ID.

func (e *E2APEncoder) GetPendingMessage(correlationID string) (*PendingMessage, bool) {
	e.mutex.RLock()

	defer e.mutex.RUnlock()

	msg, exists := e.correlationMap[correlationID]

	return msg, exists
}

// RemovePendingMessage removes a pending message.

func (e *E2APEncoder) RemovePendingMessage(correlationID string) {
	e.mutex.Lock()

	defer e.mutex.Unlock()

	delete(e.correlationMap, correlationID)
}

// Helper functions for creating common E2AP messages.

// CreateE2SetupRequest creates a new E2 Setup Request message.

func CreateE2SetupRequest(nodeID GlobalE2NodeID, functions []RANFunctionItem) *E2APMessage {
	return &E2APMessage{
		MessageType: E2APMessageTypeSetupRequest,

		ProcedureCode: 1,

		Criticality: CriticalityReject,

		Payload: &E2SetupRequest{
			GlobalE2NodeID: nodeID,

			RANFunctionsList: functions,
		},

		Timestamp: time.Now(),
	}
}

// CreateRICSubscriptionRequest creates a new RIC Subscription Request message.

func CreateRICSubscriptionRequest(requestID RICRequestID, functionID RANFunctionID, details RICSubscriptionDetails) *E2APMessage {
	return &E2APMessage{
		MessageType: E2APMessageTypeRICSubscriptionRequest,

		ProcedureCode: 2,

		Criticality: CriticalityReject,

		Payload: &RICSubscriptionRequest{
			RICRequestID: requestID,

			RANFunctionID: functionID,

			RICSubscriptionDetails: details,
		},

		Timestamp: time.Now(),
	}
}

// CreateRICControlRequest creates a new RIC Control Request message.

func CreateRICControlRequest(requestID RICRequestID, functionID RANFunctionID, header, message []byte) *E2APMessage {
	return &E2APMessage{
		MessageType: E2APMessageTypeRICControlRequest,

		ProcedureCode: 3,

		Criticality: CriticalityReject,

		Payload: &RICControlRequest{
			RICRequestID: requestID,

			RANFunctionID: functionID,

			RICControlHeader: header,

			RICControlMessage: message,
		},

		Timestamp: time.Now(),
	}
}

// Additional E2AP message structures for complete implementation.

// RICServiceUpdate represents RIC Service Update message.

type RICServiceUpdate struct {
	GlobalE2NodeID GlobalE2NodeID `json:"global_e2_node_id"`

	RANFunctionsAdded []RANFunctionItem `json:"ran_functions_added,omitempty"`

	RANFunctionsModified []RANFunctionItem `json:"ran_functions_modified,omitempty"`

	RANFunctionsDeleted []RANFunctionIDItem `json:"ran_functions_deleted,omitempty"`

	E2NodeComponentConfigAdditionList []E2NodeComponentConfigurationItem `json:"e2_node_component_config_addition_list,omitempty"`

	E2NodeComponentConfigUpdateList []E2NodeComponentConfigurationItem `json:"e2_node_component_config_update_list,omitempty"`

	E2NodeComponentConfigRemovalList []E2NodeComponentConfigurationItem `json:"e2_node_component_config_removal_list,omitempty"`
}

// RICServiceUpdateAcknowledge represents RIC Service Update Acknowledge message.

type RICServiceUpdateAcknowledge struct {
	RANFunctionsAccepted []RANFunctionIDItem `json:"ran_functions_accepted,omitempty"`

	RANFunctionsRejected []RANFunctionIDCause `json:"ran_functions_rejected,omitempty"`

	E2NodeComponentConfigAdditionAckList []E2NodeComponentConfigurationAck `json:"e2_node_component_config_addition_ack_list,omitempty"`

	E2NodeComponentConfigUpdateAckList []E2NodeComponentConfigurationAck `json:"e2_node_component_config_update_ack_list,omitempty"`

	E2NodeComponentConfigRemovalAckList []E2NodeComponentConfigurationAck `json:"e2_node_component_config_removal_ack_list,omitempty"`
}

// RICServiceUpdateFailure represents RIC Service Update Failure message.

type RICServiceUpdateFailure struct {
	Cause E2APCause `json:"cause"`

	TimeToWait *TimeToWait `json:"time_to_wait,omitempty"`

	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// RICSubscriptionDeleteRequest represents RIC Subscription Delete Request message.

type RICSubscriptionDeleteRequest struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`
}

// RICSubscriptionDeleteResponse represents RIC Subscription Delete Response message.

type RICSubscriptionDeleteResponse struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`
}

// RICSubscriptionDeleteFailure represents RIC Subscription Delete Failure message.

type RICSubscriptionDeleteFailure struct {
	RICRequestID RICRequestID `json:"ric_request_id"`

	RANFunctionID RANFunctionID `json:"ran_function_id"`

	Cause E2APCause `json:"cause"`

	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// ResetRequest represents Reset Request message.

type ResetRequest struct {
	TransactionID int32 `json:"transaction_id"`

	Cause E2APCause `json:"cause"`
}

// ResetResponse represents Reset Response message.

type ResetResponse struct {
	TransactionID int32 `json:"transaction_id"`
}

// ErrorIndication represents Error Indication message.

type ErrorIndication struct {
	TransactionID *int32 `json:"transaction_id,omitempty"`

	RICRequestID *RICRequestID `json:"ric_request_id,omitempty"`

	RANFunctionID *RANFunctionID `json:"ran_function_id,omitempty"`

	Cause E2APCause `json:"cause"`

	CriticalityDiagnostics *CriticalityDiagnostics `json:"criticality_diagnostics,omitempty"`
}

// E2NodeComponentConfigurationAck represents E2 node component configuration acknowledgment.

type E2NodeComponentConfigurationAck struct {
	E2NodeComponentInterfaceType E2NodeComponentInterfaceType `json:"e2_node_component_interface_type"`

	E2NodeComponentID E2NodeComponentID `json:"e2_node_component_id"`

	E2NodeComponentConfigurationAck E2NodeComponentConfigurationStatus `json:"e2_node_component_configuration_ack"`
}

// E2NodeComponentConfigurationStatus represents configuration status.

type E2NodeComponentConfigurationStatus struct {
	UpdateOutcome E2NodeComponentUpdateOutcome `json:"update_outcome"`

	FailureCause *E2APCause `json:"failure_cause,omitempty"`
}

// E2NodeComponentUpdateOutcome represents update outcome.

type E2NodeComponentUpdateOutcome int

const (

	// E2NodeComponentUpdateOutcomeSuccess holds e2nodecomponentupdateoutcomesuccess value.

	E2NodeComponentUpdateOutcomeSuccess E2NodeComponentUpdateOutcome = iota

	// E2NodeComponentUpdateOutcomeFailure holds e2nodecomponentupdateoutcomefailure value.

	E2NodeComponentUpdateOutcomeFailure
)

// Helper functions for creating additional E2AP messages.

// CreateRICServiceUpdate creates a new RIC Service Update message.

func CreateRICServiceUpdate(nodeID GlobalE2NodeID) *E2APMessage {
	return &E2APMessage{
		MessageType: E2APMessageTypeRICServiceUpdate,

		ProcedureCode: 7,

		Criticality: CriticalityReject,

		Payload: &RICServiceUpdate{
			GlobalE2NodeID: nodeID,
		},

		Timestamp: time.Now(),
	}
}

// CreateRICSubscriptionDeleteRequest creates a new RIC Subscription Delete Request message.

func CreateRICSubscriptionDeleteRequest(requestID RICRequestID, functionID RANFunctionID) *E2APMessage {
	return &E2APMessage{
		MessageType: E2APMessageTypeRICSubscriptionDeleteRequest,

		ProcedureCode: 8,

		Criticality: CriticalityReject,

		Payload: &RICSubscriptionDeleteRequest{
			RICRequestID: requestID,

			RANFunctionID: functionID,
		},

		Timestamp: time.Now(),
	}
}

// CreateResetRequest creates a new Reset Request message.

func CreateResetRequest(transactionID int32, cause E2APCause) *E2APMessage {
	return &E2APMessage{
		MessageType: E2APMessageTypeResetRequest,

		TransactionID: transactionID,

		ProcedureCode: 9,

		Criticality: CriticalityReject,

		Payload: &ResetRequest{
			TransactionID: transactionID,

			Cause: cause,
		},

		Timestamp: time.Now(),
	}
}

// CreateErrorIndication creates a new Error Indication message.

func CreateErrorIndication(cause E2APCause) *E2APMessage {
	return &E2APMessage{
		MessageType: E2APMessageTypeErrorIndication,

		ProcedureCode: 10,

		Criticality: CriticalityIgnore,

		Payload: &ErrorIndication{
			Cause: cause,
		},

		Timestamp: time.Now(),
	}
}
