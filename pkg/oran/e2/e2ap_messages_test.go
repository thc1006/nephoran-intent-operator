package e2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2APMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		msgType  E2APMessageType
		expected string
	}{
		{"E2 Setup Request", E2APMessageTypeSetupRequest, "E2 Setup Request"},
		{"RIC Subscription Request", E2APMessageTypeRICSubscriptionRequest, "RIC Subscription Request"},
		{"RIC Control Request", E2APMessageTypeRICControlRequest, "RIC Control Request"},
		{"RIC Indication", E2APMessageTypeRICIndication, "RIC Indication"},
		{"Error Indication", E2APMessageTypeErrorIndication, "Error Indication"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEqual(t, 0, int(tt.msgType), "Message type should not be zero")
		})
	}
}

func TestCriticalityValues(t *testing.T) {
	tests := []struct {
		name        string
		criticality Criticality
		expected    int
	}{
		{"Reject", CriticalityReject, 0},
		{"Ignore", CriticalityIgnore, 1},
		{"Notify", CriticalityNotify, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.criticality))
		})
	}
}

func TestGlobalE2NodeID(t *testing.T) {
	plmnID := PLMNIdentity{
		MCC: "001",
		MNC: "01",
	}

	gnbID := "12345"
	nodeID := GlobalE2NodeID{
		PLMNIdentity: plmnID,
		E2NodeID: E2NodeID{
			GNBID: &GNBID{
				GNBIDChoice: GNBIDChoice{
					GNBID22: &gnbID,
				},
			},
		},
	}

	assert.Equal(t, "001", nodeID.PLMNIdentity.MCC)
	assert.Equal(t, "01", nodeID.PLMNIdentity.MNC)
	assert.NotNil(t, nodeID.E2NodeID.GNBID)
	assert.Equal(t, gnbID, *nodeID.E2NodeID.GNBID.GNBIDChoice.GNBID22)
}

func TestRICRequestID(t *testing.T) {
	requestID := RICRequestID{
		RICRequestorID: 1,
		RICInstanceID:  2,
	}

	assert.Equal(t, RICRequestorID(1), requestID.RICRequestorID)
	assert.Equal(t, RICInstanceID(2), requestID.RICInstanceID)
}

func TestE2APCause(t *testing.T) {
	ricCause := RICCauseRANFunctionIDInvalid
	cause := E2APCause{
		RICCause: &ricCause,
	}

	assert.NotNil(t, cause.RICCause)
	assert.Equal(t, RICCauseRANFunctionIDInvalid, *cause.RICCause)
	assert.Nil(t, cause.RICServiceCause)
	assert.Nil(t, cause.E2NodeCause)
}

func TestRANFunctionItem(t *testing.T) {
	ranFunction := RANFunctionItem{
		RANFunctionID:         1,
		RANFunctionDefinition: []byte("test-definition"),
		RANFunctionRevision:   1,
		RANFunctionOID:        stringPtr("1.2.3.4.5"),
	}

	assert.Equal(t, RANFunctionID(1), ranFunction.RANFunctionID)
	assert.Equal(t, []byte("test-definition"), ranFunction.RANFunctionDefinition)
	assert.Equal(t, RANFunctionRevision(1), ranFunction.RANFunctionRevision)
	assert.NotNil(t, ranFunction.RANFunctionOID)
	assert.Equal(t, "1.2.3.4.5", *ranFunction.RANFunctionOID)
}

func TestRICSubscriptionDetails(t *testing.T) {
	actionItem := RICActionToBeSetupItem{
		RICActionID:   1,
		RICActionType: RICActionTypeReport,
		RICActionDefinition: []byte("action-def"),
	}

	details := RICSubscriptionDetails{
		RICEventTriggerDefinition: []byte("trigger-def"),
		RICActionToBeSetupList:    []RICActionToBeSetupItem{actionItem},
	}

	assert.Equal(t, []byte("trigger-def"), details.RICEventTriggerDefinition)
	assert.Len(t, details.RICActionToBeSetupList, 1)
	assert.Equal(t, RICActionID(1), details.RICActionToBeSetupList[0].RICActionID)
	assert.Equal(t, RICActionTypeReport, details.RICActionToBeSetupList[0].RICActionType)
}

func TestRICActionTypes(t *testing.T) {
	tests := []struct {
		name       string
		actionType RICActionType
		expected   int
	}{
		{"Report", RICActionTypeReport, 0},
		{"Insert", RICActionTypeInsert, 1},
		{"Policy", RICActionTypePolicy, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.actionType))
		})
	}
}

func TestRICIndicationTypes(t *testing.T) {
	tests := []struct {
		name           string
		indicationType RICIndicationType
		expected       int
	}{
		{"Report", RICIndicationTypeReport, 0},
		{"Insert", RICIndicationTypeInsert, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.indicationType))
		})
	}
}

func TestTimeToWaitValues(t *testing.T) {
	tests := []struct {
		name     string
		timeWait TimeToWait
		expected int
	}{
		{"1 second", TimeToWaitV1s, 0},
		{"2 seconds", TimeToWaitV2s, 1},
		{"5 seconds", TimeToWaitV5s, 2},
		{"10 seconds", TimeToWaitV10s, 3},
		{"20 seconds", TimeToWaitV20s, 4},
		{"60 seconds", TimeToWaitV60s, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.timeWait))
		})
	}
}

func TestE2NodeComponentInterfaceTypes(t *testing.T) {
	tests := []struct {
		name          string
		interfaceType E2NodeComponentInterfaceType
		expected      int
	}{
		{"NG", E2NodeComponentInterfaceTypeNG, 0},
		{"Xn", E2NodeComponentInterfaceTypeXn, 1},
		{"E1", E2NodeComponentInterfaceTypeE1, 2},
		{"F1", E2NodeComponentInterfaceTypeF1, 3},
		{"W1", E2NodeComponentInterfaceTypeW1, 4},
		{"S1", E2NodeComponentInterfaceTypeS1, 5},
		{"X2", E2NodeComponentInterfaceTypeX2, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.interfaceType))
		})
	}
}

func TestE2NodeComponentConfiguration(t *testing.T) {
	config := E2NodeComponentConfiguration{
		E2NodeComponentRequestPart:  []byte("request-part"),
		E2NodeComponentResponsePart: []byte("response-part"),
	}

	assert.Equal(t, []byte("request-part"), config.E2NodeComponentRequestPart)
	assert.Equal(t, []byte("response-part"), config.E2NodeComponentResponsePart)
}

func TestCreateE2SetupRequest(t *testing.T) {
	nodeID := GlobalE2NodeID{
		PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
		E2NodeID: E2NodeID{
			GNBID: &GNBID{
				GNBIDChoice: GNBIDChoice{
					GNBID22: stringPtr("12345"),
				},
			},
		},
	}

	functions := []RANFunctionItem{
		{
			RANFunctionID:         1,
			RANFunctionDefinition: []byte("test-function"),
			RANFunctionRevision:   1,
		},
	}

	message := CreateE2SetupRequest(nodeID, functions)

	assert.Equal(t, E2APMessageTypeSetupRequest, message.MessageType)
	assert.Equal(t, int32(1), message.ProcedureCode)
	assert.Equal(t, CriticalityReject, message.Criticality)
	assert.NotNil(t, message.Payload)

	payload, ok := message.Payload.(*E2SetupRequest)
	require.True(t, ok, "Payload should be E2SetupRequest")
	assert.Equal(t, nodeID, payload.GlobalE2NodeID)
	assert.Equal(t, functions, payload.RANFunctionsList)
}

func TestCreateRICSubscriptionRequest(t *testing.T) {
	requestID := RICRequestID{
		RICRequestorID: 1,
		RICInstanceID:  1,
	}

	details := RICSubscriptionDetails{
		RICEventTriggerDefinition: []byte("trigger"),
		RICActionToBeSetupList: []RICActionToBeSetupItem{
			{
				RICActionID:   1,
				RICActionType: RICActionTypeReport,
			},
		},
	}

	message := CreateRICSubscriptionRequest(requestID, 1, details)

	assert.Equal(t, E2APMessageTypeRICSubscriptionRequest, message.MessageType)
	assert.Equal(t, int32(2), message.ProcedureCode)
	assert.Equal(t, CriticalityReject, message.Criticality)

	payload, ok := message.Payload.(*RICSubscriptionRequest)
	require.True(t, ok, "Payload should be RICSubscriptionRequest")
	assert.Equal(t, requestID, payload.RICRequestID)
	assert.Equal(t, RANFunctionID(1), payload.RANFunctionID)
	assert.Equal(t, details, payload.RICSubscriptionDetails)
}

func TestCreateRICControlRequest(t *testing.T) {
	requestID := RICRequestID{
		RICRequestorID: 1,
		RICInstanceID:  1,
	}

	header := []byte("control-header")
	controlMsg := []byte("control-message")

	message := CreateRICControlRequest(requestID, 1, header, controlMsg)

	assert.Equal(t, E2APMessageTypeRICControlRequest, message.MessageType)
	assert.Equal(t, int32(3), message.ProcedureCode)
	assert.Equal(t, CriticalityReject, message.Criticality)

	payload, ok := message.Payload.(*RICControlRequest)
	require.True(t, ok, "Payload should be RICControlRequest")
	assert.Equal(t, requestID, payload.RICRequestID)
	assert.Equal(t, RANFunctionID(1), payload.RANFunctionID)
	assert.Equal(t, header, payload.RICControlHeader)
	assert.Equal(t, controlMsg, payload.RICControlMessage)
}

func TestCreateRICServiceUpdate(t *testing.T) {
	nodeID := GlobalE2NodeID{
		PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
	}

	message := CreateRICServiceUpdate(nodeID)

	assert.Equal(t, E2APMessageTypeRICServiceUpdate, message.MessageType)
	assert.Equal(t, int32(7), message.ProcedureCode)
	assert.Equal(t, CriticalityReject, message.Criticality)

	payload, ok := message.Payload.(*RICServiceUpdate)
	require.True(t, ok, "Payload should be RICServiceUpdate")
	assert.Equal(t, nodeID, payload.GlobalE2NodeID)
}

func TestCreateRICSubscriptionDeleteRequest(t *testing.T) {
	requestID := RICRequestID{
		RICRequestorID: 1,
		RICInstanceID:  1,
	}

	message := CreateRICSubscriptionDeleteRequest(requestID, 1)

	assert.Equal(t, E2APMessageTypeRICSubscriptionDeleteRequest, message.MessageType)
	assert.Equal(t, int32(8), message.ProcedureCode)
	assert.Equal(t, CriticalityReject, message.Criticality)

	payload, ok := message.Payload.(*RICSubscriptionDeleteRequest)
	require.True(t, ok, "Payload should be RICSubscriptionDeleteRequest")
	assert.Equal(t, requestID, payload.RICRequestID)
	assert.Equal(t, RANFunctionID(1), payload.RANFunctionID)
}

func TestCreateResetRequest(t *testing.T) {
	transactionID := int32(123)
	cause := E2APCause{
		RICCause: &[]RICCause{RICCauseSystemNotReady}[0],
	}

	message := CreateResetRequest(transactionID, cause)

	assert.Equal(t, E2APMessageTypeResetRequest, message.MessageType)
	assert.Equal(t, transactionID, message.TransactionID)
	assert.Equal(t, int32(9), message.ProcedureCode)
	assert.Equal(t, CriticalityReject, message.Criticality)

	payload, ok := message.Payload.(*ResetRequest)
	require.True(t, ok, "Payload should be ResetRequest")
	assert.Equal(t, transactionID, payload.TransactionID)
	assert.Equal(t, cause, payload.Cause)
}

func TestCreateErrorIndication(t *testing.T) {
	cause := E2APCause{
		ProtocolCause: &[]ProtocolCause{ProtocolCauseSemanticError}[0],
	}

	message := CreateErrorIndication(cause)

	assert.Equal(t, E2APMessageTypeErrorIndication, message.MessageType)
	assert.Equal(t, int32(10), message.ProcedureCode)
	assert.Equal(t, CriticalityIgnore, message.Criticality)

	payload, ok := message.Payload.(*ErrorIndication)
	require.True(t, ok, "Payload should be ErrorIndication")
	assert.Equal(t, cause, payload.Cause)
}

func TestE2APMessageTimestamp(t *testing.T) {
	before := time.Now()
	
	message := CreateE2SetupRequest(GlobalE2NodeID{}, []RANFunctionItem{})
	
	after := time.Now()

	assert.True(t, message.Timestamp.After(before) || message.Timestamp.Equal(before))
	assert.True(t, message.Timestamp.Before(after) || message.Timestamp.Equal(after))
}

func TestRICActionNotAdmittedItem(t *testing.T) {
	cause := E2APCause{
		RICCause: &[]RICCause{RICCauseActionNotSupported}[0],
	}

	item := RICActionNotAdmittedItem{
		RICActionID: 5,
		Cause:       cause,
	}

	assert.Equal(t, RICActionID(5), item.RICActionID)
	assert.Equal(t, cause, item.Cause)
	assert.NotNil(t, item.Cause.RICCause)
	assert.Equal(t, RICCauseActionNotSupported, *item.Cause.RICCause)
}

func TestCriticalityDiagnostics(t *testing.T) {
	procedureCode := int32(1)
	triggeringMsg := TriggeringMessageInitiatingMessage
	criticality := CriticalityReject

	diagnostics := CriticalityDiagnostics{
		ProcedureCode:        &procedureCode,
		TriggeringMessage:    &triggeringMsg,
		ProcedureCriticality: &criticality,
		IEsCriticalityDiagnostics: []CriticalityDiagnosticsIEItem{
			{
				IECriticality: CriticalityReject,
				IEID:          1,
				TypeOfError:   TypeOfErrorMissing,
			},
		},
	}

	assert.Equal(t, int32(1), *diagnostics.ProcedureCode)
	assert.Equal(t, TriggeringMessageInitiatingMessage, *diagnostics.TriggeringMessage)
	assert.Equal(t, CriticalityReject, *diagnostics.ProcedureCriticality)
	assert.Len(t, diagnostics.IEsCriticalityDiagnostics, 1)
	assert.Equal(t, CriticalityReject, diagnostics.IEsCriticalityDiagnostics[0].IECriticality)
	assert.Equal(t, int32(1), diagnostics.IEsCriticalityDiagnostics[0].IEID)
	assert.Equal(t, TypeOfErrorMissing, diagnostics.IEsCriticalityDiagnostics[0].TypeOfError)
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}