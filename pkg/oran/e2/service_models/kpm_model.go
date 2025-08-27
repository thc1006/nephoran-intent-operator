// FIXME: Adding package comment per revive linter
// Package service_models provides E2 service model implementations for KPM, RIC, and NI services
package service_models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
)

// KPMServiceModel implements the E2SM-KPM v2.0 service model
type KPMServiceModel struct {
	ServiceModelID      string
	ServiceModelName    string
	ServiceModelVersion string
	ServiceModelOID     string
}

// NewKPMServiceModel creates a new KPM service model instance
func NewKPMServiceModel() *KPMServiceModel {
	return &KPMServiceModel{
		ServiceModelID:      "1.3.6.1.4.1.53148.1.1.2.2",
		ServiceModelName:    "KPM",
		ServiceModelVersion: "v2.0",
		ServiceModelOID:     "1.3.6.1.4.1.53148.1.1.2.2",
	}
}

// GetServiceModelID returns the service model ID
func (kpm *KPMServiceModel) GetServiceModelID() string {
	return kpm.ServiceModelID
}

// GetServiceModelName returns the service model name
func (kpm *KPMServiceModel) GetServiceModelName() string {
	return kpm.ServiceModelName
}

// GetServiceModelVersion returns the service model version
func (kpm *KPMServiceModel) GetServiceModelVersion() string {
	return kpm.ServiceModelVersion
}

// GetServiceModelOID returns the service model OID
func (kpm *KPMServiceModel) GetServiceModelOID() string {
	return kpm.ServiceModelOID
}

// KPMTriggerConfig configures KPM event triggers
type KPMTriggerConfig struct {
	ReportingPeriodMs int64 `json:"reporting_period_ms"`
}

// KPMActionConfig configures KPM actions
type KPMActionConfig struct {
	Measurements      []KPMMeasurement `json:"measurements"`
	GranularityPeriod int64            `json:"granularity_period"`
	CellID            string           `json:"cell_id"`
}

// KPMMeasurement defines a KPM measurement
type KPMMeasurement struct {
	MeasName string `json:"meas_name"`
	MeasID   int    `json:"meas_id"`
}

// KPMReport represents a KPM measurement report
type KPMReport struct {
	Timestamp    time.Time          `json:"timestamp"`
	CellID       string             `json:"cell_id"`
	UECount      int                `json:"ue_count"`
	Measurements map[string]float64 `json:"measurements"`
}

// E2SMKPMEventTriggerDefinition represents KPM event trigger
type E2SMKPMEventTriggerDefinition struct {
	EventDefinitionFormats *E2SMKPMEventTriggerDefinitionFormat1 `json:"event_definition_formats"`
}

// E2SMKPMEventTriggerDefinitionFormat1 represents format 1
type E2SMKPMEventTriggerDefinitionFormat1 struct {
	ReportingPeriod int64 `json:"reporting_period"`
}

// E2SMKPMActionDefinition represents KPM action definition
type E2SMKPMActionDefinition struct {
	ActionDefinitionFormats *E2SMKPMActionDefinitionFormat1 `json:"action_definition_formats"`
}

// E2SMKPMActionDefinitionFormat1 represents format 1
type E2SMKPMActionDefinitionFormat1 struct {
	MeasInfoList      []KPMMeasurement `json:"meas_info_list"`
	GranularityPeriod int64            `json:"granularity_period"`
	CellObjectID      string           `json:"cell_object_id"`
}

// E2SMKPMIndicationHeader represents KPM indication header
type E2SMKPMIndicationHeader struct {
	IndicationHeaderFormats *E2SMKPMIndicationHeaderFormat1 `json:"indication_header_formats"`
}

// E2SMKPMIndicationHeaderFormat1 represents format 1
type E2SMKPMIndicationHeaderFormat1 struct {
	CollectionStartTime time.Time `json:"collection_start_time"`
	CellObjectID        string    `json:"cell_object_id"`
}

// E2SMKPMIndicationMessage represents KPM indication message
type E2SMKPMIndicationMessage struct {
	IndicationMessageFormat *E2SMKPMIndicationMessageFormat1 `json:"indication_message_format"`
}

// E2SMKPMIndicationMessageFormat1 represents format 1
type E2SMKPMIndicationMessageFormat1 struct {
	MeasDataList        []MeasurementData    `json:"meas_data_list"`
	UEMeasurementReport *UEMeasurementReport `json:"ue_measurement_report,omitempty"`
}

// MeasurementData represents measurement data
type MeasurementData struct {
	MeasRecordList []MeasurementRecord `json:"meas_record_list"`
}

// MeasurementRecord represents a measurement record
type MeasurementRecord struct {
	MeasName  string  `json:"meas_name"`
	MeasValue float64 `json:"meas_value"`
}

// UEMeasurementReport represents UE measurement report
type UEMeasurementReport struct {
	UECount int `json:"ue_count"`
}

// CreateEventTrigger creates a KPM event trigger definition
func (kpm *KPMServiceModel) CreateEventTrigger(config interface{}) ([]byte, error) {
	cfg, ok := config.(*KPMTriggerConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for KPM trigger")
	}

	trigger := &E2SMKPMEventTriggerDefinition{
		EventDefinitionFormats: &E2SMKPMEventTriggerDefinitionFormat1{
			ReportingPeriod: cfg.ReportingPeriodMs,
		},
	}

	return json.Marshal(trigger)
}

// CreateActionDefinition creates a KPM action definition
func (kpm *KPMServiceModel) CreateActionDefinition(config interface{}) ([]byte, error) {
	cfg, ok := config.(*KPMActionConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type for KPM action")
	}

	action := &E2SMKPMActionDefinition{
		ActionDefinitionFormats: &E2SMKPMActionDefinitionFormat1{
			MeasInfoList:      cfg.Measurements,
			GranularityPeriod: cfg.GranularityPeriod,
			CellObjectID:      cfg.CellID,
		},
	}

	return json.Marshal(action)
}

// ParseIndication parses a KPM indication message
func (kpm *KPMServiceModel) ParseIndication(header, message []byte) (interface{}, error) {
	// Parse indication header
	indHeader := &E2SMKPMIndicationHeader{}
	if err := json.Unmarshal(header, indHeader); err != nil {
		return nil, fmt.Errorf("failed to parse indication header: %w", err)
	}

	// Parse indication message
	indMessage := &E2SMKPMIndicationMessage{}
	if err := json.Unmarshal(message, indMessage); err != nil {
		return nil, fmt.Errorf("failed to parse indication message: %w", err)
	}

	// Convert to KPM report
	report := &KPMReport{
		Timestamp:    time.Now(),
		CellID:       indHeader.IndicationHeaderFormats.CellObjectID,
		Measurements: make(map[string]float64),
	}

	if indMessage.IndicationMessageFormat != nil {
		if indMessage.IndicationMessageFormat.UEMeasurementReport != nil {
			report.UECount = indMessage.IndicationMessageFormat.UEMeasurementReport.UECount
		}

		// Extract measurements
		for _, measData := range indMessage.IndicationMessageFormat.MeasDataList {
			for _, measRecord := range measData.MeasRecordList {
				report.Measurements[measRecord.MeasName] = measRecord.MeasValue
			}
		}
	}

	return report, nil
}

// CreateControlHeader is not applicable for KPM (measurement-only model)
// FIXME: Renamed 'params' to avoid unused parameter warning
func (kpm *KPMServiceModel) CreateControlHeader(_ interface{}) ([]byte, error) {
	return nil, fmt.Errorf("KPM service model does not support control procedures")
}

// CreateControlMessage is not applicable for KPM
// FIXME: Renamed 'params' to avoid unused parameter warning
func (kpm *KPMServiceModel) CreateControlMessage(_ interface{}) ([]byte, error) {
	return nil, fmt.Errorf("KPM service model does not support control procedures")
}

// ParseControlOutcome is not applicable for KPM
// FIXME: Renamed 'outcome' to avoid unused parameter warning
func (kpm *KPMServiceModel) ParseControlOutcome(_ []byte) (interface{}, error) {
	return nil, fmt.Errorf("KPM service model does not support control procedures")
}

// ValidateEventTrigger validates a KPM event trigger
func (kpm *KPMServiceModel) ValidateEventTrigger(trigger []byte) error {
	var eventTrigger E2SMKPMEventTriggerDefinition
	if err := json.Unmarshal(trigger, &eventTrigger); err != nil {
		return fmt.Errorf("invalid event trigger format: %w", err)
	}

	if eventTrigger.EventDefinitionFormats == nil {
		return fmt.Errorf("event definition formats missing")
	}

	if eventTrigger.EventDefinitionFormats.ReportingPeriod < 10 {
		return fmt.Errorf("reporting period too short: minimum 10ms")
	}

	return nil
}

// ValidateActionDefinition validates a KPM action definition
func (kpm *KPMServiceModel) ValidateActionDefinition(action []byte) error {
	var actionDef E2SMKPMActionDefinition
	if err := json.Unmarshal(action, &actionDef); err != nil {
		return fmt.Errorf("invalid action definition format: %w", err)
	}

	if actionDef.ActionDefinitionFormats == nil {
		return fmt.Errorf("action definition formats missing")
	}

	if len(actionDef.ActionDefinitionFormats.MeasInfoList) == 0 {
		return fmt.Errorf("no measurements specified")
	}

	return nil
}

// ValidateControlMessage is not applicable for KPM
// FIXME: Renamed 'message' to avoid unused parameter warning
func (kpm *KPMServiceModel) ValidateControlMessage(_ []byte) error {
	return fmt.Errorf("KPM service model does not support control procedures")
}

// GetSupportedMeasurements returns supported KPM measurements
func (kpm *KPMServiceModel) GetSupportedMeasurements() []KPMMeasurement {
	return []KPMMeasurement{
		// RRC measurements
		{MeasName: "RRC.ConnEstabAtt", MeasID: 1},
		{MeasName: "RRC.ConnEstabSucc", MeasID: 2},
		{MeasName: "RRC.ConnMean", MeasID: 3},
		{MeasName: "RRC.ConnMax", MeasID: 4},

		// DRB measurements
		{MeasName: "DRB.PdcpSduVolumeDL", MeasID: 10},
		{MeasName: "DRB.PdcpSduVolumeUL", MeasID: 11},
		{MeasName: "DRB.RlcSduDelayDl", MeasID: 12},
		{MeasName: "DRB.UEThpDl", MeasID: 13},
		{MeasName: "DRB.UEThpUl", MeasID: 14},

		// PRB measurements
		{MeasName: "RRU.PrbTotDl", MeasID: 20},
		{MeasName: "RRU.PrbTotUl", MeasID: 21},
		{MeasName: "RRU.PrbUsedDl", MeasID: 22},
		{MeasName: "RRU.PrbUsedUl", MeasID: 23},

		// QoS measurements
		{MeasName: "QosFlow.PdcpPduVolumeDL_Filter", MeasID: 30},
		{MeasName: "QosFlow.PdcpPduVolumeUL_Filter", MeasID: 31},
		{MeasName: "QosFlow.PdcpSduDelayDL", MeasID: 32},

		// Slice measurements
		{MeasName: "SNSSAI.DrbNumMean", MeasID: 40},
		{MeasName: "SNSSAI.PdcpSduVolumeDL", MeasID: 41},
		{MeasName: "SNSSAI.PdcpSduVolumeUL", MeasID: 42},
	}
}

// CreateKPMSubscription creates a complete KPM subscription
func (kpm *KPMServiceModel) CreateKPMSubscription(nodeID string, cellID string, measurements []string) (*e2.E2Subscription, error) {
	// Create event trigger for periodic reporting
	triggerConfig := &KPMTriggerConfig{
		ReportingPeriodMs: 1000, // 1 second
	}

	eventTrigger, err := kpm.CreateEventTrigger(triggerConfig)
	if err != nil {
		return nil, err
	}

	// Create action for measurements
	measList := make([]KPMMeasurement, 0, len(measurements))
	supportedMeas := kpm.GetSupportedMeasurements()

	for _, measName := range measurements {
		for _, supported := range supportedMeas {
			if supported.MeasName == measName {
				measList = append(measList, supported)
				break
			}
		}
	}

	actionConfig := &KPMActionConfig{
		Measurements:      measList,
		GranularityPeriod: 1000, // 1 second
		CellID:            cellID,
	}

	actionDef, err := kpm.CreateActionDefinition(actionConfig)
	if err != nil {
		return nil, err
	}

	// Create subscription
	subscription := &e2.E2Subscription{
		SubscriptionID: fmt.Sprintf("kpm-%s-%d", nodeID, time.Now().Unix()),
		RequestorID:    "kpm-xapp",
		RanFunctionID:  1, // KPM function ID
		EventTriggers: []e2.E2EventTrigger{
			{
				TriggerType:     "PERIODIC",
				ReportingPeriod: time.Second,
			},
		},
		Actions: []e2.E2Action{
			{
				ActionID:   1,
				ActionType: "REPORT",
				ActionDefinition: map[string]interface{}{
					"event_trigger": json.RawMessage(eventTrigger),
					"action_def":    json.RawMessage(actionDef),
				},
			},
		},
		ReportingPeriod: time.Second,
		Status: e2.E2SubscriptionStatus{
			State: "PENDING",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return subscription, nil
}
