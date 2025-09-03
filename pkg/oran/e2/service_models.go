package e2

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Enhanced Service Model Support with Plugin Architecture.

// ServiceModelRegistry methods for E2ServiceModelRegistry.

// registerServiceModel registers a service model with optional plugin.

func (r *E2ServiceModelRegistry) registerServiceModel(serviceModel *E2ServiceModel, plugin ServiceModelPlugin) error {
	r.mutex.Lock()

	defer r.mutex.Unlock()

	// Validate service model.

	if err := r.validateServiceModelInternal(serviceModel); err != nil {
		return fmt.Errorf("service model validation failed: %w", err)
	}

	// Create registered service model.

	registered := &RegisteredServiceModel{
		E2ServiceModel: *serviceModel,

		RegistrationTime: time.Now(),

		Version: serviceModel.ServiceModelVersion,

		Plugin: plugin,

		ValidationRules: r.getValidationRules(serviceModel.ServiceModelName),

		Compatibility: r.getCompatibilityList(serviceModel.ServiceModelName),
	}

	r.serviceModels[serviceModel.ServiceModelID] = registered

	// Register plugin if provided.

	if plugin != nil {
		r.plugins[serviceModel.ServiceModelID] = plugin
	}

	return nil
}

// validateServiceModel validates a service model configuration.

func (r *E2ServiceModelRegistry) validateServiceModel(serviceModel *E2ServiceModel) error {
	r.mutex.RLock()

	defer r.mutex.RUnlock()

	return r.validateServiceModelInternal(serviceModel)
}

// validateServiceModelInternal performs internal validation (assumes lock held).

func (r *E2ServiceModelRegistry) validateServiceModelInternal(serviceModel *E2ServiceModel) error {
	if serviceModel.ServiceModelID == "" {
		return fmt.Errorf("service model ID is required")
	}

	if serviceModel.ServiceModelName == "" {
		return fmt.Errorf("service model name is required")
	}

	if serviceModel.ServiceModelVersion == "" {
		return fmt.Errorf("service model version is required")
	}

	if serviceModel.ServiceModelOID == "" {
		return fmt.Errorf("service model OID is required")
	}

	if len(serviceModel.SupportedProcedures) == 0 {
		return fmt.Errorf("at least one supported procedure is required")
	}

	// Validate procedures.

	validProcedures := map[string]bool{
		"RIC_SUBSCRIPTION": true,

		"RIC_SUBSCRIPTION_DELETE": true,

		"RIC_INDICATION": true,

		"RIC_CONTROL_REQUEST": true,

		"RIC_CONTROL_ACKNOWLEDGE": true,

		"RIC_CONTROL_FAILURE": true,

		"RIC_SERVICE_UPDATE": true,
	}

	for _, procedure := range serviceModel.SupportedProcedures {
		if !validProcedures[procedure] {
			return fmt.Errorf("invalid procedure: %s", procedure)
		}
	}

	return nil
}

// getValidationRules returns validation rules for a service model.

func (r *E2ServiceModelRegistry) getValidationRules(modelName string) []ValidationRule {
	switch modelName {

	case "KPM":

		return []ValidationRule{
			{
				Name: "measurement_types_validation",

				Description: "Validate KPM measurement types",

				Validate: r.validateKPMMeasurementTypes,
			},

			{
				Name: "granularity_period_validation",

				Description: "Validate KPM granularity period",

				Validate: r.validateKPMGranularityPeriod,
			},
		}

	case "RC":

		return []ValidationRule{
			{
				Name: "control_actions_validation",

				Description: "Validate RC control actions",

				Validate: r.validateRCControlActions,
			},

			{
				Name: "control_outcomes_validation",

				Description: "Validate RC control outcomes",

				Validate: r.validateRCControlOutcomes,
			},
		}

	default:

		return []ValidationRule{}

	}
}

// getCompatibilityList returns compatibility information for a service model.

func (r *E2ServiceModelRegistry) getCompatibilityList(modelName string) []string {
	switch modelName {

	case "KPM":

		return []string{"v1.0", "v1.1", "v2.0"}

	case "RC":

		return []string{"v1.0", "v1.1"}

	default:

		return []string{}

	}
}

// Service model validation functions.

func (r *E2ServiceModelRegistry) validateKPMMeasurementTypes(serviceModel *E2ServiceModel) error {
	if len(serviceModel.Configuration) == 0 {
		return fmt.Errorf("KPM service model requires configuration")
	}

	var config map[string]interface{}
	if err := json.Unmarshal(serviceModel.Configuration, &config); err != nil {
		return fmt.Errorf("invalid configuration format: %w", err)
	}

	measurementTypes, exists := config["measurement_types"]

	if !exists {
		return fmt.Errorf("KPM service model requires measurement_types")
	}

	types, ok := measurementTypes.([]string)

	if !ok {
		// Try to convert from []interface{}.

		if interfaceSlice, ok := measurementTypes.([]interface{}); ok {

			types = make([]string, len(interfaceSlice))

			for i, v := range interfaceSlice {
				if str, ok := v.(string); ok {
					types[i] = str
				} else {
					return fmt.Errorf("invalid measurement type at index %d", i)
				}
			}

		} else {
			return fmt.Errorf("measurement_types must be a string array")
		}
	}

	// Validate known measurement types.

	validTypes := map[string]bool{
		"DRB.RlcSduDelayDl": true,

		"DRB.RlcSduDelayUl": true,

		"DRB.RlcSduVolumeDl": true,

		"DRB.RlcSduVolumeUl": true,

		"DRB.UEThpDl": true,

		"DRB.UEThpUl": true,

		"RRU.PrbTotDl": true,

		"RRU.PrbTotUl": true,

		"RRU.PrbUsedDl": true,

		"RRU.PrbUsedUl": true,

		"TB.TotNbrDl": true,

		"TB.TotNbrUl": true,

		"TB.ErrTotNbrDl": true,

		"TB.ErrTotNbrUl": true,

		"MAC.UESchedDlInitialTx": true,

		"MAC.UESchedUlInitialTx": true,

		"CARR.PDSCHMCSDist": true,

		"CARR.PUSCHMCSDist": true,
	}

	for _, measurementType := range types {
		if !validTypes[measurementType] {
			return fmt.Errorf("unknown measurement type: %s", measurementType)
		}
	}

	return nil
}

func (r *E2ServiceModelRegistry) validateKPMGranularityPeriod(serviceModel *E2ServiceModel) error {
	if len(serviceModel.Configuration) == 0 {
		return nil // Optional configuration
	}

	var config map[string]interface{}
	if err := json.Unmarshal(serviceModel.Configuration, &config); err != nil {
		return fmt.Errorf("invalid configuration format: %w", err)
	}

	granularityPeriod, exists := config["granularity_period"]

	if !exists {
		return nil // Optional field
	}

	periodStr, ok := granularityPeriod.(string)

	if !ok {
		return fmt.Errorf("granularity_period must be a string")
	}

	// Validate period format (e.g., "1000ms", "1s", "5s").

	validPeriods := []string{"100ms", "200ms", "500ms", "1000ms", "1s", "2s", "5s", "10s"}

	for _, validPeriod := range validPeriods {
		if periodStr == validPeriod {
			return nil
		}
	}

	return fmt.Errorf("invalid granularity_period: %s", periodStr)
}

func (r *E2ServiceModelRegistry) validateRCControlActions(serviceModel *E2ServiceModel) error {
	if len(serviceModel.Configuration) == 0 {
		return fmt.Errorf("RC service model requires configuration")
	}

	var config map[string]interface{}
	if err := json.Unmarshal(serviceModel.Configuration, &config); err != nil {
		return fmt.Errorf("invalid configuration format: %w", err)
	}

	controlActions, exists := config["control_actions"]

	if !exists {
		return fmt.Errorf("RC service model requires control_actions")
	}

	actions, ok := controlActions.([]string)

	if !ok {
		// Try to convert from []interface{}.

		if interfaceSlice, ok := controlActions.([]interface{}); ok {

			actions = make([]string, len(interfaceSlice))

			for i, v := range interfaceSlice {
				if str, ok := v.(string); ok {
					actions[i] = str
				} else {
					return fmt.Errorf("invalid control action at index %d", i)
				}
			}

		} else {
			return fmt.Errorf("control_actions must be a string array")
		}
	}

	// Validate known control actions.

	validActions := map[string]bool{
		"QoS_flow_mapping": true,

		"Traffic_steering": true,

		"Dual_connectivity": true,

		"Mobility_control": true,

		"Carrier_aggregation": true,

		"RRM_optimization": true,

		"Beamforming_control": true,
	}

	for _, action := range actions {
		if !validActions[action] {
			return fmt.Errorf("unknown control action: %s", action)
		}
	}

	return nil
}

func (r *E2ServiceModelRegistry) validateRCControlOutcomes(serviceModel *E2ServiceModel) error {
	if len(serviceModel.Configuration) == 0 {
		return nil // Optional configuration
	}

	var config map[string]interface{}
	if err := json.Unmarshal(serviceModel.Configuration, &config); err != nil {
		return fmt.Errorf("invalid configuration format: %w", err)
	}

	controlOutcomes, exists := config["control_outcomes"]

	if !exists {
		return nil // Optional field
	}

	outcomes, ok := controlOutcomes.([]string)

	if !ok {
		// Try to convert from []interface{}.

		if interfaceSlice, ok := controlOutcomes.([]interface{}); ok {

			outcomes = make([]string, len(interfaceSlice))

			for i, v := range interfaceSlice {
				if str, ok := v.(string); ok {
					outcomes[i] = str
				} else {
					return fmt.Errorf("invalid control outcome at index %d", i)
				}
			}

		} else {
			return fmt.Errorf("control_outcomes must be a string array")
		}
	}

	// Validate known control outcomes.

	validOutcomes := map[string]bool{
		"successful": true,

		"rejected": true,

		"failed": true,

		"partial": true,

		"timeout": true,
	}

	for _, outcome := range outcomes {
		if !validOutcomes[outcome] {
			return fmt.Errorf("unknown control outcome: %s", outcome)
		}
	}

	return nil
}

// Enhanced service model creation functions.

// CreateEnhancedKPMServiceModel creates an enhanced KPM service model with full configuration.

func CreateEnhancedKPMServiceModel() *E2ServiceModel {
	return &E2ServiceModel{
		ServiceModelID: "1.3.6.1.4.1.53148.1.1.2.2",

		ServiceModelName: "KPM",

		ServiceModelVersion: "2.0",

		ServiceModelOID: "1.3.6.1.4.1.53148.1.1.2.2",

		SupportedProcedures: []string{
			"RIC_SUBSCRIPTION",

			"RIC_SUBSCRIPTION_DELETE",

			"RIC_INDICATION",
		},

		Configuration: json.RawMessage(`{
			"granularity_period": "1000ms",
			"collection_start_time": "2025-07-29T10:00:00Z",
			"collection_duration": "continuous",
			"reporting_format": "CHOICE",
			"supported_cell_types": ["NR", "LTE"],
			"supported_aggregation_levels": ["cell", "ue", "slice"],
			"max_concurrent_measurements": 100,
			"measurement_filtering": {}
		}`),
	}
}

// CreateEnhancedRCServiceModel creates an enhanced RC service model with full configuration.

func CreateEnhancedRCServiceModel() *E2ServiceModel {
	return &E2ServiceModel{
		ServiceModelID: "1.3.6.1.4.1.53148.1.1.2.3",

		ServiceModelName: "RC",

		ServiceModelVersion: "1.1",

		ServiceModelOID: "1.3.6.1.4.1.53148.1.1.2.3",

		SupportedProcedures: []string{
			"RIC_CONTROL_REQUEST",

			"RIC_CONTROL_ACKNOWLEDGE",

			"RIC_CONTROL_FAILURE",
		},

		Configuration: json.RawMessage(`{
			"control_outcomes": ["successful", "rejected", "failed", "partial", "timeout"],
			"control_granularity": ["cell", "ue", "slice", "beam"],
			"control_frequency": "on_demand",
			"supported_ran_types": ["gNB", "ng-eNB", "en-gNB"],
			"max_concurrent_controls": 50,
			"control_timeout": "5s",
			"acknowledgment_required": true,
			"rollback_supported": true,
			"batch_control_supported": false
		}`),
	}
}

// CreateReportServiceModel creates a generic Report service model for custom data structures.

func CreateReportServiceModel() *E2ServiceModel {
	return &E2ServiceModel{
		ServiceModelID: "1.3.6.1.4.1.53148.1.1.2.4",

		ServiceModelName: "REPORT",

		ServiceModelVersion: "1.0",

		ServiceModelOID: "1.3.6.1.4.1.53148.1.1.2.4",

		SupportedProcedures: []string{
			"RIC_SUBSCRIPTION",

			"RIC_SUBSCRIPTION_DELETE",

			"RIC_INDICATION",
		},

		Configuration: json.RawMessage(`{
			"data_formats": ["JSON", "XML", "BINARY", "ASN1_PER"],
			"reporting_frequency": ["on_demand", "periodic", "event_driven"],
			"aggregation_supported": true,
			"filtering_supported": true,
			"compression_supported": true,
			"encryption_supported": false,
			"max_report_size": "1MB",
			"max_concurrent_reports": 200
		}`),
	}
}

// Service model plugin interface implementations.

// KMPServiceModelPlugin implements KMP-specific processing.

type KMPServiceModelPlugin struct {
	name string

	version string

	processors map[string]func(context.Context, interface{}) (interface{}, error)

	mutex sync.RWMutex
}

// NewKMPServiceModelPlugin creates a new KMP service model plugin.

func NewKMPServiceModelPlugin() *KMPServiceModelPlugin {
	plugin := &KMPServiceModelPlugin{
		name: "KMP",

		version: "2.0",

		processors: make(map[string]func(context.Context, interface{}) (interface{}, error)),
	}

	// Register processors.

	plugin.processors["RIC_SUBSCRIPTION"] = plugin.processSubscription

	plugin.processors["RIC_INDICATION"] = plugin.processIndication

	return plugin
}

// GetName performs getname operation.

func (p *KMPServiceModelPlugin) GetName() string {
	return p.name
}

// GetVersion performs getversion operation.

func (p *KMPServiceModelPlugin) GetVersion() string {
	return p.version
}

// Validate performs validate operation.

func (p *KMPServiceModelPlugin) Validate(serviceModel *E2ServiceModel) error {
	// KMP-specific validation logic.

	if serviceModel.ServiceModelName != "KPM" {
		return fmt.Errorf("invalid service model for KMP plugin")
	}

	return nil
}

// Process performs process operation.

func (p *KMPServiceModelPlugin) Process(ctx context.Context, request interface{}) (interface{}, error) {
	p.mutex.RLock()

	defer p.mutex.RUnlock()

	// Determine request type and route to appropriate processor.

	switch req := request.(type) {

	case *RICSubscriptionRequest:

		if processor, exists := p.processors["RIC_SUBSCRIPTION"]; exists {
			return processor(ctx, req)
		}

	case *RICIndication:

		if processor, exists := p.processors["RIC_INDICATION"]; exists {
			return processor(ctx, req)
		}

	default:

		return nil, fmt.Errorf("unsupported request type for KMP plugin")

	}

	return nil, fmt.Errorf("no processor found for request")
}

// GetSupportedProcedures performs getsupportedprocedures operation.

func (p *KMPServiceModelPlugin) GetSupportedProcedures() []string {
	return []string{"RIC_SUBSCRIPTION", "RIC_SUBSCRIPTION_DELETE", "RIC_INDICATION"}
}

func (p *KMPServiceModelPlugin) processSubscription(ctx context.Context, request interface{}) (interface{}, error) {
	req, ok := request.(*RICSubscriptionRequest)

	if !ok {
		return nil, fmt.Errorf("invalid request type for subscription processor")
	}

	// KMP-specific subscription processing.

	// Parse event trigger definition for KMP measurements.

	var eventTrigger map[string]interface{}

	if err := json.Unmarshal(req.RICSubscriptionDetails.RICEventTriggerDefinition, &eventTrigger); err != nil {
		return nil, fmt.Errorf("failed to parse event trigger: %w", err)
	}

	// Validate KMP measurement configuration.

	measurementTypes, ok := eventTrigger["measurement_types"].([]interface{})

	if !ok {
		return nil, fmt.Errorf("invalid measurement_types in event trigger")
	}

	// Process measurement types.

	processedMeasurements := make([]string, 0, len(measurementTypes))

	for _, mt := range measurementTypes {
		if measurementType, ok := mt.(string); ok {
			processedMeasurements = append(processedMeasurements, measurementType)
		}
	}

	// Return processed subscription response.

	return &RICSubscriptionResponse{
		RICRequestID: req.RICRequestID,

		RANFunctionID: req.RANFunctionID,

		RICActionAdmittedList: []RICActionID{1}, // Assume action 1 is admitted

	}, nil
}

func (p *KMPServiceModelPlugin) processIndication(ctx context.Context, request interface{}) (interface{}, error) {
	ind, ok := request.(*RICIndication)

	if !ok {
		return nil, fmt.Errorf("invalid request type for indication processor")
	}

	// KMP-specific indication processing.

	// Parse indication message for KMP measurements.

	var indicationData map[string]interface{}

	if err := json.Unmarshal(ind.RICIndicationMessage, &indicationData); err != nil {
		return nil, fmt.Errorf("failed to parse indication message: %w", err)
	}

	// Process measurement data.

	processedData := json.RawMessage(`{}`)

	return processedData, nil
}

// RCServiceModelPlugin implements RC-specific processing.

type RCServiceModelPlugin struct {
	name string

	version string

	processors map[string]func(context.Context, interface{}) (interface{}, error)

	mutex sync.RWMutex
}

// NewRCServiceModelPlugin creates a new RC service model plugin.

func NewRCServiceModelPlugin() *RCServiceModelPlugin {
	plugin := &RCServiceModelPlugin{
		name: "RC",

		version: "1.1",

		processors: make(map[string]func(context.Context, interface{}) (interface{}, error)),
	}

	// Register processors.

	plugin.processors["RIC_CONTROL_REQUEST"] = plugin.processControlRequest

	return plugin
}

// GetName performs getname operation.

func (p *RCServiceModelPlugin) GetName() string {
	return p.name
}

// GetVersion performs getversion operation.

func (p *RCServiceModelPlugin) GetVersion() string {
	return p.version
}

// Validate performs validate operation.

func (p *RCServiceModelPlugin) Validate(serviceModel *E2ServiceModel) error {
	if serviceModel.ServiceModelName != "RC" {
		return fmt.Errorf("invalid service model for RC plugin")
	}

	return nil
}

// Process performs process operation.

func (p *RCServiceModelPlugin) Process(ctx context.Context, request interface{}) (interface{}, error) {
	p.mutex.RLock()

	defer p.mutex.RUnlock()

	switch req := request.(type) {

	case *RICControlRequest:

		if processor, exists := p.processors["RIC_CONTROL_REQUEST"]; exists {
			return processor(ctx, req)
		}

	default:

		return nil, fmt.Errorf("unsupported request type for RC plugin")

	}

	return nil, fmt.Errorf("no processor found for request")
}

// GetSupportedProcedures performs getsupportedprocedures operation.

func (p *RCServiceModelPlugin) GetSupportedProcedures() []string {
	return []string{"RIC_CONTROL_REQUEST", "RIC_CONTROL_ACKNOWLEDGE", "RIC_CONTROL_FAILURE"}
}

func (p *RCServiceModelPlugin) processControlRequest(ctx context.Context, request interface{}) (interface{}, error) {
	req, ok := request.(*RICControlRequest)

	if !ok {
		return nil, fmt.Errorf("invalid request type for control processor")
	}

	// RC-specific control processing.

	// Parse control header and message.

	var controlHeader map[string]interface{}

	if err := json.Unmarshal(req.RICControlHeader, &controlHeader); err != nil {
		return nil, fmt.Errorf("failed to parse control header: %w", err)
	}

	var controlMessage map[string]interface{}

	if err := json.Unmarshal(req.RICControlMessage, &controlMessage); err != nil {
		return nil, fmt.Errorf("failed to parse control message: %w", err)
	}

	// Process control action.

	controlAction, ok := controlMessage["action"].(string)

	if !ok {
		return nil, fmt.Errorf("missing or invalid control action")
	}

	// Simulate control execution.

	var outcome string

	switch controlAction {

	case "QoS_flow_mapping", "Traffic_steering":

		outcome = "successful"

	case "Dual_connectivity":

		outcome = "partial"

	default:

		outcome = "rejected"

	}

	// Return control acknowledge.
	outcomeMap := map[string]interface{}{
		"result": outcome,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	outcomeData, _ := json.Marshal(outcomeMap)

	return &RICControlAcknowledge{
		RICRequestID: req.RICRequestID,

		RANFunctionID: req.RANFunctionID,

		RICCallProcessID: req.RICCallProcessID,

		RICControlOutcome: outcomeData,
	}, nil
}

