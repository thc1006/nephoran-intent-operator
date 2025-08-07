package service_models

import (
	"encoding/json"
	"fmt"
	"time"
)

// NIServiceModel implements the E2SM-NI (Network Interface) Service Model
// Following O-RAN.WG3.E2SM-NI-v01.00 specification
type NIServiceModel struct {
	ServiceModelOID     string
	ServiceModelName    string
	ServiceModelVersion string
}

// NewNIServiceModel creates a new Network Interface Service Model instance
func NewNIServiceModel() *NIServiceModel {
	return &NIServiceModel{
		ServiceModelOID:     "1.3.6.1.4.1.53148.1.2.2.102",
		ServiceModelName:    "E2SM-NI",
		ServiceModelVersion: "v01.00",
	}
}

// NIEventTriggerDefinition defines event triggers for NI service model
type NIEventTriggerDefinition struct {
	EventDefinitionFormats NIEventDefinitionFormats `json:"event_definition_formats"`
	ReportingPeriod        int                      `json:"reporting_period_ms,omitempty"` // milliseconds
}

// NIEventDefinitionFormats defines different event formats
type NIEventDefinitionFormats struct {
	EventDefinitionFormat1 *NIEventDefinitionFormat1 `json:"format1,omitempty"`
	EventDefinitionFormat2 *NIEventDefinitionFormat2 `json:"format2,omitempty"`
}

// NIEventDefinitionFormat1 for periodic reporting
type NIEventDefinitionFormat1 struct {
	ReportingPeriod int `json:"reporting_period_ms"` // milliseconds
}

// NIEventDefinitionFormat2 for event-driven reporting
type NIEventDefinitionFormat2 struct {
	TriggerType      NITriggerType      `json:"trigger_type"`
	InterfaceType    NIInterfaceType    `json:"interface_type"`
	InterfaceID      string             `json:"interface_id"`
	MessageDirection NIMessageDirection `json:"message_direction"`
	MessageTypeList  []NIMessageType    `json:"message_type_list,omitempty"`
}

// NITriggerType defines trigger types for NI events
type NITriggerType int

const (
	NITriggerTypeUponReceive NITriggerType = iota
	NITriggerTypeUponSend
	NITriggerTypeUponChange
)

// NIInterfaceType defines network interface types
type NIInterfaceType int

const (
	NIInterfaceTypeE1 NIInterfaceType = iota
	NIInterfaceTypeF1
	NIInterfaceTypeE2
	NIInterfaceTypeXn
	NIInterfaceTypeX2
	NIInterfaceTypeNg
	NIInterfaceTypeS1
)

// NIMessageDirection defines message direction
type NIMessageDirection int

const (
	NIMessageDirectionIncoming NIMessageDirection = iota
	NIMessageDirectionOutgoing
	NIMessageDirectionBoth
)

// NIMessageType defines specific message types to monitor
type NIMessageType struct {
	ProcedureCode int    `json:"procedure_code"`
	TypeOfMessage string `json:"type_of_message"` // "initiating", "successful", "unsuccessful"
}

// NIActionDefinition defines actions for NI service model
type NIActionDefinition struct {
	ActionDefinitionFormats NIActionDefinitionFormats `json:"action_definition_formats"`
}

// NIActionDefinitionFormats defines different action formats
type NIActionDefinitionFormats struct {
	ActionDefinitionFormat1 *NIActionDefinitionFormat1 `json:"format1,omitempty"`
	ActionDefinitionFormat2 *NIActionDefinitionFormat2 `json:"format2,omitempty"`
	ActionDefinitionFormat3 *NIActionDefinitionFormat3 `json:"format3,omitempty"`
}

// NIActionDefinitionFormat1 for message capture
type NIActionDefinitionFormat1 struct {
	InterfaceList        []NIInterfaceInfo `json:"interface_list"`
	MessageCapture       bool              `json:"message_capture"`
	IncludeRawMessage    bool              `json:"include_raw_message"`
	IncludeDecodedFields bool              `json:"include_decoded_fields"`
}

// NIActionDefinitionFormat2 for statistics collection
type NIActionDefinitionFormat2 struct {
	InterfaceList   []NIInterfaceInfo `json:"interface_list"`
	MeasurementList []NIMeasurement   `json:"measurement_list"`
	GranularityPeriod int             `json:"granularity_period_ms"` // milliseconds
}

// NIActionDefinitionFormat3 for interface monitoring
type NIActionDefinitionFormat3 struct {
	InterfaceList     []NIInterfaceInfo       `json:"interface_list"`
	MonitoringType    NIMonitoringType        `json:"monitoring_type"`
	ThresholdList     []NIThreshold           `json:"threshold_list,omitempty"`
	NotificationMode  NINotificationMode      `json:"notification_mode"`
}

// NIInterfaceInfo describes a network interface
type NIInterfaceInfo struct {
	InterfaceType     NIInterfaceType `json:"interface_type"`
	InterfaceID       string          `json:"interface_id"`
	InterfaceEndpoint string          `json:"interface_endpoint,omitempty"`
}

// NIMeasurement defines a measurement for statistics
type NIMeasurement struct {
	MeasurementID   int    `json:"measurement_id"`
	MeasurementName string `json:"measurement_name"`
	MeasurementUnit string `json:"measurement_unit,omitempty"`
}

// NIMonitoringType defines monitoring types
type NIMonitoringType int

const (
	NIMonitoringTypeStatus NIMonitoringType = iota
	NIMonitoringTypeLoad
	NIMonitoringTypeLatency
	NIMonitoringTypeErrors
)

// NIThreshold defines threshold for monitoring
type NIThreshold struct {
	ThresholdType  string  `json:"threshold_type"`
	ThresholdValue float64 `json:"threshold_value"`
	Direction      string  `json:"direction"` // "above", "below"
}

// NINotificationMode defines how notifications are sent
type NINotificationMode int

const (
	NINotificationModeImmediate NINotificationMode = iota
	NINotificationModePeriodic
	NINotificationModeOnChange
)

// NIIndicationHeader defines indication header for NI
type NIIndicationHeader struct {
	InterfaceID      string             `json:"interface_id"`
	InterfaceType    NIInterfaceType    `json:"interface_type"`
	MessageDirection NIMessageDirection `json:"message_direction,omitempty"`
	Timestamp        time.Time          `json:"timestamp"`
}

// NIIndicationMessage defines indication message for NI
type NIIndicationMessage struct {
	IndicationMessageFormats NIIndicationMessageFormats `json:"indication_message_formats"`
}

// NIIndicationMessageFormats defines different indication formats
type NIIndicationMessageFormats struct {
	IndicationMessageFormat1 *NIIndicationMessageFormat1 `json:"format1,omitempty"`
	IndicationMessageFormat2 *NIIndicationMessageFormat2 `json:"format2,omitempty"`
	IndicationMessageFormat3 *NIIndicationMessageFormat3 `json:"format3,omitempty"`
}

// NIIndicationMessageFormat1 for captured messages
type NIIndicationMessageFormat1 struct {
	CapturedMessage     []byte                 `json:"captured_message"`
	MessageType         NIMessageType          `json:"message_type"`
	DecodedFields       map[string]interface{} `json:"decoded_fields,omitempty"`
	CaptureTimestamp    time.Time              `json:"capture_timestamp"`
	MessageSize         int                    `json:"message_size"`
}

// NIIndicationMessageFormat2 for statistics report
type NIIndicationMessageFormat2 struct {
	InterfaceStatsList []NIInterfaceStats `json:"interface_stats_list"`
	ReportPeriod       NIReportPeriod     `json:"report_period"`
}

// NIIndicationMessageFormat3 for interface status
type NIIndicationMessageFormat3 struct {
	InterfaceStatusList []NIInterfaceStatus `json:"interface_status_list"`
	NotificationCause   string              `json:"notification_cause,omitempty"`
}

// NIInterfaceStats contains statistics for an interface
type NIInterfaceStats struct {
	InterfaceID    string                   `json:"interface_id"`
	InterfaceType  NIInterfaceType          `json:"interface_type"`
	Measurements   []NIMeasurementValue     `json:"measurements"`
	StartTime      time.Time                `json:"start_time"`
	EndTime        time.Time                `json:"end_time"`
}

// NIMeasurementValue contains a measurement value
type NIMeasurementValue struct {
	MeasurementID    int     `json:"measurement_id"`
	MeasurementName  string  `json:"measurement_name"`
	MeasurementValue float64 `json:"measurement_value"`
	MeasurementUnit  string  `json:"measurement_unit,omitempty"`
}

// NIInterfaceStatus contains status information for an interface
type NIInterfaceStatus struct {
	InterfaceID     string                 `json:"interface_id"`
	InterfaceType   NIInterfaceType        `json:"interface_type"`
	OperationalState NIOperationalState    `json:"operational_state"`
	AdminState      NIAdminState           `json:"admin_state"`
	LoadLevel       float64                `json:"load_level,omitempty"`
	ErrorRate       float64                `json:"error_rate,omitempty"`
	Latency         float64                `json:"latency_ms,omitempty"`
	AdditionalInfo  map[string]interface{} `json:"additional_info,omitempty"`
}

// NIOperationalState defines operational state of interface
type NIOperationalState int

const (
	NIOperationalStateUp NIOperationalState = iota
	NIOperationalStateDown
	NIOperationalStateDegraded
	NIOperationalStateUnknown
)

// NIAdminState defines administrative state of interface
type NIAdminState int

const (
	NIAdminStateEnabled NIAdminState = iota
	NIAdminStateDisabled
	NIAdminStateShuttingDown
)

// NIReportPeriod defines the reporting period
type NIReportPeriod struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// NIControlHeader defines control header for NI
type NIControlHeader struct {
	InterfaceID   string          `json:"interface_id"`
	InterfaceType NIInterfaceType `json:"interface_type"`
	ControlAction NIControlAction `json:"control_action"`
}

// NIControlMessage defines control message for NI
type NIControlMessage struct {
	ControlMessageFormats NIControlMessageFormats `json:"control_message_formats"`
}

// NIControlMessageFormats defines different control message formats
type NIControlMessageFormats struct {
	ControlMessageFormat1 *NIControlMessageFormat1 `json:"format1,omitempty"`
	ControlMessageFormat2 *NIControlMessageFormat2 `json:"format2,omitempty"`
}

// NIControlMessageFormat1 for interface control
type NIControlMessageFormat1 struct {
	TargetInterface  NIInterfaceInfo        `json:"target_interface"`
	ControlAction    NIControlAction        `json:"control_action"`
	ControlParameter map[string]interface{} `json:"control_parameter,omitempty"`
}

// NIControlMessageFormat2 for message injection
type NIControlMessageFormat2 struct {
	TargetInterface   NIInterfaceInfo    `json:"target_interface"`
	MessageToInject   []byte             `json:"message_to_inject"`
	MessageType       NIMessageType      `json:"message_type"`
	InjectionTiming   NIInjectionTiming  `json:"injection_timing"`
}

// NIControlAction defines control actions
type NIControlAction int

const (
	NIControlActionEnable NIControlAction = iota
	NIControlActionDisable
	NIControlActionReset
	NIControlActionModify
	NIControlActionInject
)

// NIInjectionTiming defines when to inject a message
type NIInjectionTiming struct {
	ImmediateInjection bool          `json:"immediate_injection"`
	ScheduledTime      *time.Time    `json:"scheduled_time,omitempty"`
	DelayMs            int           `json:"delay_ms,omitempty"`
}

// NIControlOutcome defines control outcome for NI
type NIControlOutcome struct {
	ControlOutcomeFormats NIControlOutcomeFormats `json:"control_outcome_formats"`
}

// NIControlOutcomeFormats defines different control outcome formats
type NIControlOutcomeFormats struct {
	ControlOutcomeFormat1 *NIControlOutcomeFormat1 `json:"format1,omitempty"`
}

// NIControlOutcomeFormat1 for control result
type NIControlOutcomeFormat1 struct {
	ControlResult     NIControlResult        `json:"control_result"`
	ResultDescription string                 `json:"result_description,omitempty"`
	ResultDetails     map[string]interface{} `json:"result_details,omitempty"`
}

// NIControlResult defines control result status
type NIControlResult int

const (
	NIControlResultSuccess NIControlResult = iota
	NIControlResultPartialSuccess
	NIControlResultFailure
)

// EncodeEventTriggerDefinition encodes NI event trigger definition
func (m *NIServiceModel) EncodeEventTriggerDefinition(definition *NIEventTriggerDefinition) ([]byte, error) {
	return json.Marshal(definition)
}

// DecodeEventTriggerDefinition decodes NI event trigger definition
func (m *NIServiceModel) DecodeEventTriggerDefinition(data []byte) (*NIEventTriggerDefinition, error) {
	var definition NIEventTriggerDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, err
	}
	return &definition, nil
}

// EncodeActionDefinition encodes NI action definition
func (m *NIServiceModel) EncodeActionDefinition(definition *NIActionDefinition) ([]byte, error) {
	return json.Marshal(definition)
}

// DecodeActionDefinition decodes NI action definition
func (m *NIServiceModel) DecodeActionDefinition(data []byte) (*NIActionDefinition, error) {
	var definition NIActionDefinition
	if err := json.Unmarshal(data, &definition); err != nil {
		return nil, err
	}
	return &definition, nil
}

// EncodeIndicationHeader encodes NI indication header
func (m *NIServiceModel) EncodeIndicationHeader(header *NIIndicationHeader) ([]byte, error) {
	return json.Marshal(header)
}

// DecodeIndicationHeader decodes NI indication header
func (m *NIServiceModel) DecodeIndicationHeader(data []byte) (*NIIndicationHeader, error) {
	var header NIIndicationHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}
	return &header, nil
}

// EncodeIndicationMessage encodes NI indication message
func (m *NIServiceModel) EncodeIndicationMessage(message *NIIndicationMessage) ([]byte, error) {
	return json.Marshal(message)
}

// DecodeIndicationMessage decodes NI indication message
func (m *NIServiceModel) DecodeIndicationMessage(data []byte) (*NIIndicationMessage, error) {
	var message NIIndicationMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// EncodeControlHeader encodes NI control header
func (m *NIServiceModel) EncodeControlHeader(header *NIControlHeader) ([]byte, error) {
	return json.Marshal(header)
}

// DecodeControlHeader decodes NI control header
func (m *NIServiceModel) DecodeControlHeader(data []byte) (*NIControlHeader, error) {
	var header NIControlHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}
	return &header, nil
}

// EncodeControlMessage encodes NI control message
func (m *NIServiceModel) EncodeControlMessage(message *NIControlMessage) ([]byte, error) {
	return json.Marshal(message)
}

// DecodeControlMessage decodes NI control message
func (m *NIServiceModel) DecodeControlMessage(data []byte) (*NIControlMessage, error) {
	var message NIControlMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// EncodeControlOutcome encodes NI control outcome
func (m *NIServiceModel) EncodeControlOutcome(outcome *NIControlOutcome) ([]byte, error) {
	return json.Marshal(outcome)
}

// DecodeControlOutcome decodes NI control outcome
func (m *NIServiceModel) DecodeControlOutcome(data []byte) (*NIControlOutcome, error) {
	var outcome NIControlOutcome
	if err := json.Unmarshal(data, &outcome); err != nil {
		return nil, err
	}
	return &outcome, nil
}

// ValidateEventTriggerDefinition validates NI event trigger definition
func (m *NIServiceModel) ValidateEventTriggerDefinition(definition *NIEventTriggerDefinition) error {
	if definition == nil {
		return fmt.Errorf("event trigger definition is nil")
	}

	// Validate based on format
	if definition.EventDefinitionFormats.EventDefinitionFormat1 != nil {
		if definition.EventDefinitionFormats.EventDefinitionFormat1.ReportingPeriod <= 0 {
			return fmt.Errorf("reporting period must be positive")
		}
	}

	if definition.EventDefinitionFormats.EventDefinitionFormat2 != nil {
		format2 := definition.EventDefinitionFormats.EventDefinitionFormat2
		if format2.InterfaceID == "" {
			return fmt.Errorf("interface ID is required for format 2")
		}
	}

	return nil
}

// ValidateActionDefinition validates NI action definition
func (m *NIServiceModel) ValidateActionDefinition(definition *NIActionDefinition) error {
	if definition == nil {
		return fmt.Errorf("action definition is nil")
	}

	// Validate based on format
	if definition.ActionDefinitionFormats.ActionDefinitionFormat1 != nil {
		format1 := definition.ActionDefinitionFormats.ActionDefinitionFormat1
		if len(format1.InterfaceList) == 0 {
			return fmt.Errorf("interface list cannot be empty for format 1")
		}
	}

	if definition.ActionDefinitionFormats.ActionDefinitionFormat2 != nil {
		format2 := definition.ActionDefinitionFormats.ActionDefinitionFormat2
		if len(format2.InterfaceList) == 0 {
			return fmt.Errorf("interface list cannot be empty for format 2")
		}
		if len(format2.MeasurementList) == 0 {
			return fmt.Errorf("measurement list cannot be empty for format 2")
		}
		if format2.GranularityPeriod <= 0 {
			return fmt.Errorf("granularity period must be positive")
		}
	}

	if definition.ActionDefinitionFormats.ActionDefinitionFormat3 != nil {
		format3 := definition.ActionDefinitionFormats.ActionDefinitionFormat3
		if len(format3.InterfaceList) == 0 {
			return fmt.Errorf("interface list cannot be empty for format 3")
		}
	}

	return nil
}

// GetSupportedInterfaces returns the list of supported network interfaces
func (m *NIServiceModel) GetSupportedInterfaces() []string {
	return []string{"E1", "F1", "E2", "Xn", "X2", "Ng", "S1"}
}

// GetSupportedMeasurements returns the list of supported measurements
func (m *NIServiceModel) GetSupportedMeasurements() []NIMeasurement {
	return []NIMeasurement{
		{MeasurementID: 1, MeasurementName: "Messages Sent", MeasurementUnit: "count"},
		{MeasurementID: 2, MeasurementName: "Messages Received", MeasurementUnit: "count"},
		{MeasurementID: 3, MeasurementName: "Message Errors", MeasurementUnit: "count"},
		{MeasurementID: 4, MeasurementName: "Average Latency", MeasurementUnit: "ms"},
		{MeasurementID: 5, MeasurementName: "Max Latency", MeasurementUnit: "ms"},
		{MeasurementID: 6, MeasurementName: "Throughput", MeasurementUnit: "Mbps"},
		{MeasurementID: 7, MeasurementName: "Interface Load", MeasurementUnit: "percent"},
		{MeasurementID: 8, MeasurementName: "Queue Length", MeasurementUnit: "count"},
		{MeasurementID: 9, MeasurementName: "Drop Rate", MeasurementUnit: "percent"},
		{MeasurementID: 10, MeasurementName: "Retransmissions", MeasurementUnit: "count"},
	}
}

// GetSupportedControlActions returns the list of supported control actions
func (m *NIServiceModel) GetSupportedControlActions() []string {
	return []string{"Enable", "Disable", "Reset", "Modify", "Inject"}
}