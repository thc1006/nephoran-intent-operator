package service_models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
)

// RCServiceModel implements the E2SM-RC v1.0 service model.

type RCServiceModel struct {
	ServiceModelID string

	ServiceModelName string

	ServiceModelVersion string

	ServiceModelOID string
}

// NewRCServiceModel creates a new RC service model instance.

func NewRCServiceModel() *RCServiceModel {
	return &RCServiceModel{
		ServiceModelID: "1.3.6.1.4.1.53148.1.1.2.3",

		ServiceModelName: "RC",

		ServiceModelVersion: "v1.0",

		ServiceModelOID: "1.3.6.1.4.1.53148.1.1.2.3",
	}
}

// GetServiceModelID returns the service model ID.

func (rc *RCServiceModel) GetServiceModelID() string {
	return rc.ServiceModelID
}

// GetServiceModelName returns the service model name.

func (rc *RCServiceModel) GetServiceModelName() string {
	return rc.ServiceModelName
}

// GetServiceModelVersion returns the service model version.

func (rc *RCServiceModel) GetServiceModelVersion() string {
	return rc.ServiceModelVersion
}

// GetServiceModelOID returns the service model OID.

func (rc *RCServiceModel) GetServiceModelOID() string {
	return rc.ServiceModelOID
}

// RCControlType defines types of RC control.

type RCControlType string

const (

	// RCControlTypeTrafficSteering holds rccontroltypetrafficsteering value.

	RCControlTypeTrafficSteering RCControlType = "TRAFFIC_STEERING"

	// RCControlTypeQoSModification holds rccontroltypeqosmodification value.

	RCControlTypeQoSModification RCControlType = "QOS_MODIFICATION"

	// RCControlTypeHandoverControl holds rccontroltypehandovercontrol value.

	RCControlTypeHandoverControl RCControlType = "HANDOVER_CONTROL"

	// RCControlTypeDualConnectivity holds rccontroltypedualconnectivity value.

	RCControlTypeDualConnectivity RCControlType = "DUAL_CONNECTIVITY"

	// RCControlTypeRadioResourceAlloc holds rccontroltyperadioresourcealloc value.

	RCControlTypeRadioResourceAlloc RCControlType = "RADIO_RESOURCE_ALLOC"
)

// RCParameter represents a RAN parameter.

type RCParameter struct {
	ParameterID int `json:"parameter_id"`

	ParameterName string `json:"parameter_name"`

	ParameterValue interface{} `json:"parameter_value"`
}

// RCControlParams contains control parameters.

type RCControlParams struct {
	Parameters []RCParameter `json:"parameters"`
}

// RCControlResult represents control operation result.

type RCControlResult struct {
	Success bool `json:"success"`

	Cause string `json:"cause,omitempty"`

	Details json.RawMessage `json:"details,omitempty"`
}

// E2SMRCEventTriggerDefinition represents RC event trigger.

type E2SMRCEventTriggerDefinition struct {
	EventDefinitionFormats *E2SMRCEventTriggerDefinitionFormat1 `json:"event_definition_formats"`
}

// E2SMRCEventTriggerDefinitionFormat1 represents format 1.

type E2SMRCEventTriggerDefinitionFormat1 struct {
	TriggerType string `json:"trigger_type"`

	Parameters json.RawMessage `json:"parameters,omitempty"`
}

// E2SMRCActionDefinition represents RC action definition.

type E2SMRCActionDefinition struct {
	ActionDefinitionFormats *E2SMRCActionDefinitionFormat1 `json:"action_definition_formats"`
}

// E2SMRCActionDefinitionFormat1 represents format 1.

type E2SMRCActionDefinitionFormat1 struct {
	ControlActionID int `json:"control_action_id"`

	ControlStyle int `json:"control_style"`

	Parameters json.RawMessage `json:"parameters,omitempty"`
}

// E2SMRCControlHeader represents RC control header.

type E2SMRCControlHeader struct {
	ControlHeaderFormats *E2SMRCControlHeaderFormat1 `json:"control_header_formats"`
}

// E2SMRCControlHeaderFormat1 represents format 1.

type E2SMRCControlHeaderFormat1 struct {
	UEID string `json:"ue_id"`

	ControlType RCControlType `json:"control_type"`

	ControlStyle int `json:"control_style"`
}

// E2SMRCControlMessage represents RC control message.

type E2SMRCControlMessage struct {
	ControlMessageFormats *E2SMRCControlMessageFormat1 `json:"control_message_formats"`
}

// E2SMRCControlMessageFormat1 represents format 1.

type E2SMRCControlMessageFormat1 struct {
	RANParameters []RCParameter `json:"ran_parameters"`
}

// E2SMRCControlOutcome represents RC control outcome.

type E2SMRCControlOutcome struct {
	ControlOutcomeFormats *E2SMRCControlOutcomeFormat1 `json:"control_outcome_formats"`
}

// E2SMRCControlOutcomeFormat1 represents format 1.

type E2SMRCControlOutcomeFormat1 struct {
	ControlStyle int `json:"control_style"`

	Result string `json:"result"`

	Cause string `json:"cause,omitempty"`
}

// CreateEventTrigger creates an RC event trigger definition.

// FIXME: Renamed 'config' to avoid unused parameter warning.

func (rc *RCServiceModel) CreateEventTrigger(_ interface{}) ([]byte, error) {
	// RC typically uses on-demand control rather than periodic triggers.

	trigger := &E2SMRCEventTriggerDefinition{
		EventDefinitionFormats: &E2SMRCEventTriggerDefinitionFormat1{
			TriggerType: "ON_DEMAND",

			Parameters: make(map[string]interface{}),
		},
	}

	return json.Marshal(trigger)
}

// CreateActionDefinition creates an RC action definition.

// FIXME: Renamed 'config' to avoid unused parameter warning.

func (rc *RCServiceModel) CreateActionDefinition(_ interface{}) ([]byte, error) {
	action := &E2SMRCActionDefinition{
		ActionDefinitionFormats: &E2SMRCActionDefinitionFormat1{
			ControlActionID: 1,

			ControlStyle: 1,

			Parameters: make(map[string]interface{}),
		},
	}

	return json.Marshal(action)
}

// ParseIndication parses an RC indication message (if applicable).

// FIXME: Renamed unused parameters to avoid warnings.

func (rc *RCServiceModel) ParseIndication(_, _ []byte) (interface{}, error) {
	// RC primarily uses control procedures, but may have indications for control results.

	return nil, fmt.Errorf("RC indication parsing not implemented")
}

// CreateControlHeader creates an RC control header.

func (rc *RCServiceModel) CreateControlHeader(params interface{}) ([]byte, error) {
	headerParams, ok := params.(*E2SMRCControlHeaderFormat1)

	if !ok {
		return nil, fmt.Errorf("invalid params type for RC control header")
	}

	header := &E2SMRCControlHeader{
		ControlHeaderFormats: headerParams,
	}

	return json.Marshal(header)
}

// CreateControlMessage creates an RC control message.

func (rc *RCServiceModel) CreateControlMessage(params interface{}) ([]byte, error) {
	controlParams, ok := params.(*RCControlParams)

	if !ok {
		return nil, fmt.Errorf("invalid params type for RC control message")
	}

	message := &E2SMRCControlMessage{
		ControlMessageFormats: &E2SMRCControlMessageFormat1{
			RANParameters: controlParams.Parameters,
		},
	}

	return json.Marshal(message)
}

// ParseControlOutcome parses an RC control outcome.

func (rc *RCServiceModel) ParseControlOutcome(outcome []byte) (interface{}, error) {
	var outcomeMsg E2SMRCControlOutcome

	if err := json.Unmarshal(outcome, &outcomeMsg); err != nil {
		return nil, fmt.Errorf("failed to parse control outcome: %w", err)
	}

	if outcomeMsg.ControlOutcomeFormats == nil {
		return nil, fmt.Errorf("control outcome formats missing")
	}

	result := &RCControlResult{
		Success: outcomeMsg.ControlOutcomeFormats.Result == "SUCCESS",

		Cause: outcomeMsg.ControlOutcomeFormats.Cause,

		Details: make(map[string]interface{}),
	}

	return result, nil
}

// ValidateEventTrigger validates an RC event trigger.

func (rc *RCServiceModel) ValidateEventTrigger(trigger []byte) error {
	var eventTrigger E2SMRCEventTriggerDefinition

	if err := json.Unmarshal(trigger, &eventTrigger); err != nil {
		return fmt.Errorf("invalid event trigger format: %w", err)
	}

	return nil
}

// ValidateActionDefinition validates an RC action definition.

func (rc *RCServiceModel) ValidateActionDefinition(action []byte) error {
	var actionDef E2SMRCActionDefinition

	if err := json.Unmarshal(action, &actionDef); err != nil {
		return fmt.Errorf("invalid action definition format: %w", err)
	}

	return nil
}

// ValidateControlMessage validates an RC control message.

func (rc *RCServiceModel) ValidateControlMessage(message []byte) error {
	var controlMsg E2SMRCControlMessage

	if err := json.Unmarshal(message, &controlMsg); err != nil {
		return fmt.Errorf("invalid control message format: %w", err)
	}

	if controlMsg.ControlMessageFormats == nil {
		return fmt.Errorf("control message formats missing")
	}

	return nil
}

// Traffic Steering Control Implementation.

// CreateTrafficSteeringControl creates a traffic steering control request.

func (rc *RCServiceModel) CreateTrafficSteeringControl(ueID, targetCellID string) (*e2.E2ControlRequest, error) {
	// Create control header.

	headerParams := &E2SMRCControlHeaderFormat1{
		UEID: ueID,

		ControlType: RCControlTypeTrafficSteering,

		ControlStyle: 1,
	}

	header, err := rc.CreateControlHeader(headerParams)
	if err != nil {
		return nil, err
	}

	// Create control message with parameters.

	params := &RCControlParams{
		Parameters: []RCParameter{
			{
				ParameterID: 1,

				ParameterName: "target_cell_id",

				ParameterValue: targetCellID,
			},

			{
				ParameterID: 2,

				ParameterName: "handover_cause",

				ParameterValue: "load-balancing",
			},

			{
				ParameterID: 3,

				ParameterName: "force_handover",

				ParameterValue: false,
			},
		},
	}

	message, err := rc.CreateControlMessage(params)
	if err != nil {
		return nil, err
	}

	// Build control request.

	request := &e2.E2ControlRequest{
		RequestID: fmt.Sprintf("rc-traffic-%s-%d", ueID, time.Now().Unix()),

		RanFunctionID: 2, // RC function ID

		CallProcessID: "", // Empty string instead of nil

		ControlHeader: json.RawMessage("{}"),

		ControlMessage: json.RawMessage("{}"),

		ControlAckRequest: true,
	}

	return request, nil
}

// QoS Control Implementation.

// CreateQoSControl creates a QoS modification control request.

func (rc *RCServiceModel) CreateQoSControl(ueID string, bearerID int, qosParams map[string]interface{}) (*e2.E2ControlRequest, error) {
	// Create control header.

	headerParams := &E2SMRCControlHeaderFormat1{
		UEID: ueID,

		ControlType: RCControlTypeQoSModification,

		ControlStyle: 2,
	}

	header, err := rc.CreateControlHeader(headerParams)
	if err != nil {
		return nil, err
	}

	// Create control message with QoS parameters.

	params := &RCControlParams{
		Parameters: []RCParameter{
			{
				ParameterID: 1,

				ParameterName: "bearer_id",

				ParameterValue: bearerID,
			},

			{
				ParameterID: 2,

				ParameterName: "qci",

				ParameterValue: qosParams["qci"],
			},

			{
				ParameterID: 3,

				ParameterName: "priority_level",

				ParameterValue: qosParams["priority_level"],
			},

			{
				ParameterID: 4,

				ParameterName: "preemption_capability",

				ParameterValue: qosParams["preemption_capability"],
			},

			{
				ParameterID: 5,

				ParameterName: "preemption_vulnerability",

				ParameterValue: qosParams["preemption_vulnerability"],
			},

			{
				ParameterID: 6,

				ParameterName: "gbr_dl",

				ParameterValue: qosParams["gbr_dl"],
			},

			{
				ParameterID: 7,

				ParameterName: "gbr_ul",

				ParameterValue: qosParams["gbr_ul"],
			},
		},
	}

	message, err := rc.CreateControlMessage(params)
	if err != nil {
		return nil, err
	}

	// Build control request.

	request := &e2.E2ControlRequest{
		RequestID: fmt.Sprintf("rc-qos-%s-%d", ueID, time.Now().Unix()),

		RanFunctionID: 2, // RC function ID

		CallProcessID: "", // Empty string instead of nil

		ControlHeader: json.RawMessage("{}"),

		ControlMessage: json.RawMessage("{}"),

		ControlAckRequest: true,
	}

	return request, nil
}

// Handover Control Implementation.

// CreateHandoverControl creates a handover control request.

func (rc *RCServiceModel) CreateHandoverControl(ueID, targetCellID, handoverType string) (*e2.E2ControlRequest, error) {
	// Create control header.

	headerParams := &E2SMRCControlHeaderFormat1{
		UEID: ueID,

		ControlType: RCControlTypeHandoverControl,

		ControlStyle: 3,
	}

	header, err := rc.CreateControlHeader(headerParams)
	if err != nil {
		return nil, err
	}

	// Create control message with handover parameters.

	params := &RCControlParams{
		Parameters: []RCParameter{
			{
				ParameterID: 1,

				ParameterName: "target_cell_id",

				ParameterValue: targetCellID,
			},

			{
				ParameterID: 2,

				ParameterName: "handover_type",

				ParameterValue: handoverType, // "intra-freq", "inter-freq", "inter-rat"

			},

			{
				ParameterID: 3,

				ParameterName: "preparation_timer",

				ParameterValue: 5000, // milliseconds

			},

			{
				ParameterID: 4,

				ParameterName: "completion_timer",

				ParameterValue: 10000, // milliseconds

			},
		},
	}

	message, err := rc.CreateControlMessage(params)
	if err != nil {
		return nil, err
	}

	// Build control request.

	request := &e2.E2ControlRequest{
		RequestID: fmt.Sprintf("rc-handover-%s-%d", ueID, time.Now().Unix()),

		RanFunctionID: 2, // RC function ID

		CallProcessID: "", // Empty string instead of nil

		ControlHeader: json.RawMessage("{}"),

		ControlMessage: json.RawMessage("{}"),

		ControlAckRequest: true,
	}

	return request, nil
}

// Dual Connectivity Control Implementation.

// CreateDualConnectivityControl creates a dual connectivity control request.

func (rc *RCServiceModel) CreateDualConnectivityControl(ueID, operation, secondaryCellID string) (*e2.E2ControlRequest, error) {
	// Create control header.

	headerParams := &E2SMRCControlHeaderFormat1{
		UEID: ueID,

		ControlType: RCControlTypeDualConnectivity,

		ControlStyle: 4,
	}

	header, err := rc.CreateControlHeader(headerParams)
	if err != nil {
		return nil, err
	}

	// Create control message with dual connectivity parameters.

	params := &RCControlParams{
		Parameters: []RCParameter{
			{
				ParameterID: 1,

				ParameterName: "operation",

				ParameterValue: operation, // "add", "modify", "release"

			},

			{
				ParameterID: 2,

				ParameterName: "secondary_cell_id",

				ParameterValue: secondaryCellID,
			},

			{
				ParameterID: 3,

				ParameterName: "split_bearer_option",

				ParameterValue: "mcg-split", // "mcg-split", "scg-split"

			},

			{
				ParameterID: 4,

				ParameterName: "data_forwarding",

				ParameterValue: true,
			},
		},
	}

	message, err := rc.CreateControlMessage(params)
	if err != nil {
		return nil, err
	}

	// Build control request.

	request := &e2.E2ControlRequest{
		RequestID: fmt.Sprintf("rc-dc-%s-%d", ueID, time.Now().Unix()),

		RanFunctionID: 2, // RC function ID

		CallProcessID: "", // Empty string instead of nil

		ControlHeader: json.RawMessage("{}"),

		ControlMessage: json.RawMessage("{}"),

		ControlAckRequest: true,
	}

	return request, nil
}

// GetSupportedControlTypes returns supported RC control types.

func (rc *RCServiceModel) GetSupportedControlTypes() []RCControlType {
	return []RCControlType{
		RCControlTypeTrafficSteering,

		RCControlTypeQoSModification,

		RCControlTypeHandoverControl,

		RCControlTypeDualConnectivity,

		RCControlTypeRadioResourceAlloc,
	}
}
