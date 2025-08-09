package e2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewE2APEncoder(t *testing.T) {
	encoder := NewE2APEncoder()

	assert.NotNil(t, encoder)
	assert.NotNil(t, encoder.messageRegistry)
	assert.NotNil(t, encoder.correlationMap)

	// Verify all codecs are registered
	expectedCodecs := []E2APMessageType{
		E2APMessageTypeSetupRequest,
		E2APMessageTypeSetupResponse,
		E2APMessageTypeSetupFailure,
		E2APMessageTypeRICSubscriptionRequest,
		E2APMessageTypeRICSubscriptionResponse,
		E2APMessageTypeRICSubscriptionFailure,
		E2APMessageTypeRICControlRequest,
		E2APMessageTypeRICControlAcknowledge,
		E2APMessageTypeRICControlFailure,
		E2APMessageTypeRICIndication,
		E2APMessageTypeRICServiceUpdate,
		E2APMessageTypeRICServiceUpdateAcknowledge,
		E2APMessageTypeRICServiceUpdateFailure,
		E2APMessageTypeRICSubscriptionDeleteRequest,
		E2APMessageTypeRICSubscriptionDeleteResponse,
		E2APMessageTypeRICSubscriptionDeleteFailure,
		E2APMessageTypeResetRequest,
		E2APMessageTypeResetResponse,
		E2APMessageTypeErrorIndication,
	}

	for _, msgType := range expectedCodecs {
		_, exists := encoder.messageRegistry[msgType]
		assert.True(t, exists, "Codec for message type %d should be registered", msgType)
	}
}

func TestE2APEncoder_EncodeDecodeE2SetupRequest(t *testing.T) {
	encoder := NewE2APEncoder()

	// Create test message
	req := &E2SetupRequest{
		GlobalE2NodeID: GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
		},
		RANFunctionsList: []RANFunctionItem{
			{
				RANFunctionID:         1,
				RANFunctionDefinition: []byte("test-function"),
				RANFunctionRevision:   1,
			},
		},
	}

	message := &E2APMessage{
		MessageType:   E2APMessageTypeSetupRequest,
		TransactionID: 123,
		ProcedureCode: 1,
		Criticality:   CriticalityReject,
		Payload:       req,
		Timestamp:     time.Now(),
	}

	// Test encoding
	data, err := encoder.EncodeMessage(message)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test decoding
	decoded, err := encoder.DecodeMessage(data)
	assert.NoError(t, err)
	assert.NotNil(t, decoded)

	// Verify message fields
	assert.Equal(t, message.MessageType, decoded.MessageType)
	assert.Equal(t, message.TransactionID, decoded.TransactionID)
	assert.Equal(t, message.ProcedureCode, decoded.ProcedureCode)
	assert.Equal(t, message.Criticality, decoded.Criticality)

	// Verify payload
	decodedReq, ok := decoded.Payload.(*E2SetupRequest)
	require.True(t, ok)
	assert.Equal(t, req.GlobalE2NodeID.PLMNIdentity.MCC, decodedReq.GlobalE2NodeID.PLMNIdentity.MCC)
	assert.Len(t, decodedReq.RANFunctionsList, 1)
}

func TestE2APEncoder_EncodeDecodeRICSubscriptionRequest(t *testing.T) {
	encoder := NewE2APEncoder()

	req := &RICSubscriptionRequest{
		RICRequestID:  RICRequestID{RICRequestorID: 1, RICInstanceID: 1},
		RANFunctionID: 1,
		RICSubscriptionDetails: RICSubscriptionDetails{
			RICEventTriggerDefinition: []byte("trigger"),
			RICActionToBeSetupList: []RICActionToBeSetupItem{
				{
					RICActionID:   1,
					RICActionType: RICActionTypeReport,
				},
			},
		},
	}

	message := &E2APMessage{
		MessageType:   E2APMessageTypeRICSubscriptionRequest,
		TransactionID: 456,
		ProcedureCode: 2,
		Criticality:   CriticalityReject,
		Payload:       req,
		Timestamp:     time.Now(),
	}

	// Encode/decode cycle
	data, err := encoder.EncodeMessage(message)
	assert.NoError(t, err)

	decoded, err := encoder.DecodeMessage(data)
	assert.NoError(t, err)

	// Verify
	assert.Equal(t, message.MessageType, decoded.MessageType)
	decodedReq, ok := decoded.Payload.(*RICSubscriptionRequest)
	require.True(t, ok)
	assert.Equal(t, req.RICRequestID, decodedReq.RICRequestID)
	assert.Equal(t, req.RANFunctionID, decodedReq.RANFunctionID)
}

func TestE2APEncoder_EncodeDecodeRICControlRequest(t *testing.T) {
	encoder := NewE2APEncoder()

	req := &RICControlRequest{
		RICRequestID:      RICRequestID{RICRequestorID: 1, RICInstanceID: 1},
		RANFunctionID:     1,
		RICControlHeader:  []byte("control-header"),
		RICControlMessage: []byte("control-message"),
	}

	message := &E2APMessage{
		MessageType:   E2APMessageTypeRICControlRequest,
		TransactionID: 789,
		ProcedureCode: 3,
		Criticality:   CriticalityReject,
		Payload:       req,
		Timestamp:     time.Now(),
	}

	// Encode/decode cycle
	data, err := encoder.EncodeMessage(message)
	assert.NoError(t, err)

	decoded, err := encoder.DecodeMessage(data)
	assert.NoError(t, err)

	// Verify
	assert.Equal(t, message.MessageType, decoded.MessageType)
	decodedReq, ok := decoded.Payload.(*RICControlRequest)
	require.True(t, ok)
	assert.Equal(t, req.RICRequestID, decodedReq.RICRequestID)
	assert.Equal(t, req.RICControlHeader, decodedReq.RICControlHeader)
	assert.Equal(t, req.RICControlMessage, decodedReq.RICControlMessage)
}

func TestE2APEncoder_EncodeDecodeRICIndication(t *testing.T) {
	encoder := NewE2APEncoder()

	ind := &RICIndication{
		RICRequestID:         RICRequestID{RICRequestorID: 1, RICInstanceID: 1},
		RANFunctionID:        1,
		RICActionID:          1,
		RICIndicationType:    RICIndicationTypeReport,
		RICIndicationHeader:  []byte("indication-header"),
		RICIndicationMessage: []byte("indication-message"),
	}

	message := &E2APMessage{
		MessageType:   E2APMessageTypeRICIndication,
		TransactionID: 999,
		ProcedureCode: 4,
		Criticality:   CriticalityIgnore,
		Payload:       ind,
		Timestamp:     time.Now(),
	}

	// Encode/decode cycle
	data, err := encoder.EncodeMessage(message)
	assert.NoError(t, err)

	decoded, err := encoder.DecodeMessage(data)
	assert.NoError(t, err)

	// Verify
	assert.Equal(t, message.MessageType, decoded.MessageType)
	decodedInd, ok := decoded.Payload.(*RICIndication)
	require.True(t, ok)
	assert.Equal(t, ind.RICRequestID, decodedInd.RICRequestID)
	assert.Equal(t, ind.RICIndicationType, decodedInd.RICIndicationType)
	assert.Equal(t, ind.RICIndicationHeader, decodedInd.RICIndicationHeader)
	assert.Equal(t, ind.RICIndicationMessage, decodedInd.RICIndicationMessage)
}

func TestE2APEncoder_EncodeUnregisteredMessageType(t *testing.T) {
	encoder := NewE2APEncoder()

	// Create message with unregistered type
	message := &E2APMessage{
		MessageType:   E2APMessageType(9999), // Non-existent type
		TransactionID: 123,
		ProcedureCode: 1,
		Criticality:   CriticalityReject,
		Payload:       &E2SetupRequest{},
		Timestamp:     time.Now(),
	}

	_, err := encoder.EncodeMessage(message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no codec registered for message type")
}

func TestE2APEncoder_DecodeInvalidHeader(t *testing.T) {
	encoder := NewE2APEncoder()

	// Test with data too short
	shortData := []byte{1, 2, 3}
	_, err := encoder.DecodeMessage(shortData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message too short")

	// Test with header containing unregistered message type
	invalidHeader := make([]byte, 16)
	// Set invalid message type (9999)
	invalidHeader[0] = 0x00
	invalidHeader[1] = 0x00
	invalidHeader[2] = 0x27
	invalidHeader[3] = 0x0F // 9999 in bytes

	_, err = encoder.DecodeMessage(invalidHeader)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no codec registered for message type")
}

func TestE2APEncoder_TrackPendingMessage(t *testing.T) {
	encoder := NewE2APEncoder()

	correlationID := "test-correlation-123"
	pendingMsg := &PendingMessage{
		MessageType:   E2APMessageTypeSetupRequest,
		TransactionID: 123,
		SentTime:      time.Now(),
		Timeout:       30 * time.Second,
	}

	// Track message
	encoder.TrackMessage(correlationID, pendingMsg)

	// Retrieve message
	retrieved, exists := encoder.GetPendingMessage(correlationID)
	assert.True(t, exists)
	assert.Equal(t, pendingMsg.MessageType, retrieved.MessageType)
	assert.Equal(t, pendingMsg.TransactionID, retrieved.TransactionID)

	// Remove message
	encoder.RemovePendingMessage(correlationID)
	_, exists = encoder.GetPendingMessage(correlationID)
	assert.False(t, exists)
}

func TestE2APEncoder_ValidationFailure(t *testing.T) {
	encoder := NewE2APEncoder()

	// Create invalid message (missing required fields)
	req := &E2SetupRequest{
		GlobalE2NodeID: GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{}, // Missing MCC
		},
		RANFunctionsList: []RANFunctionItem{}, // Empty list
	}

	message := &E2APMessage{
		MessageType:   E2APMessageTypeSetupRequest,
		TransactionID: 123,
		ProcedureCode: 1,
		Criticality:   CriticalityReject,
		Payload:       req,
		Timestamp:     time.Now(),
	}

	_, err := encoder.EncodeMessage(message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message validation failed")
}

func TestE2APEncoder_MessageHeaderFormat(t *testing.T) {
	encoder := NewE2APEncoder()

	req := &E2SetupRequest{
		GlobalE2NodeID: GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
		},
		RANFunctionsList: []RANFunctionItem{
			{RANFunctionID: 1, RANFunctionRevision: 1},
		},
	}

	message := &E2APMessage{
		MessageType:   E2APMessageTypeSetupRequest,
		TransactionID: 0x12345678,
		ProcedureCode: 0x87654321,
		Criticality:   CriticalityReject,
		Payload:       req,
		Timestamp:     time.Now(),
	}

	data, err := encoder.EncodeMessage(message)
	assert.NoError(t, err)
	assert.True(t, len(data) >= 16, "Message should have at least 16-byte header")

	// Decode and verify header fields were preserved
	decoded, err := encoder.DecodeMessage(data)
	assert.NoError(t, err)
	assert.Equal(t, message.MessageType, decoded.MessageType)
	assert.Equal(t, message.TransactionID, decoded.TransactionID)
	assert.Equal(t, message.ProcedureCode, decoded.ProcedureCode)
	assert.Equal(t, message.Criticality, decoded.Criticality)
}

func TestE2APEncoder_ConcurrentAccess(t *testing.T) {
	encoder := NewE2APEncoder()

	// Test concurrent access to correlation map
	correlationIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		correlationIDs[i] = fmt.Sprintf("correlation-%d", i)
	}

	// Concurrent writes
	done := make(chan bool, 100)
	for _, id := range correlationIDs {
		go func(correlationID string) {
			pendingMsg := &PendingMessage{
				MessageType:   E2APMessageTypeSetupRequest,
				TransactionID: 123,
				SentTime:      time.Now(),
				Timeout:       30 * time.Second,
			}
			encoder.TrackMessage(correlationID, pendingMsg)
			done <- true
		}(id)
	}

	// Wait for all writes to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// Concurrent reads
	for _, id := range correlationIDs {
		go func(correlationID string) {
			_, exists := encoder.GetPendingMessage(correlationID)
			assert.True(t, exists)
			done <- true
		}(id)
	}

	// Wait for all reads to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestE2APEncoder_AllMessageTypes(t *testing.T) {
	encoder := NewE2APEncoder()

	testCases := []struct {
		name        string
		messageType E2APMessageType
		payload     interface{}
	}{
		{
			name:        "E2SetupRequest",
			messageType: E2APMessageTypeSetupRequest,
			payload: &E2SetupRequest{
				GlobalE2NodeID:   GlobalE2NodeID{PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"}},
				RANFunctionsList: []RANFunctionItem{{RANFunctionID: 1}},
			},
		},
		{
			name:        "E2SetupResponse",
			messageType: E2APMessageTypeSetupResponse,
			payload: &E2SetupResponse{
				GlobalRICID: GlobalRICID{PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"}},
			},
		},
		{
			name:        "RICServiceUpdate",
			messageType: E2APMessageTypeRICServiceUpdate,
			payload: &RICServiceUpdate{
				GlobalE2NodeID: GlobalE2NodeID{PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"}},
			},
		},
		{
			name:        "ResetRequest",
			messageType: E2APMessageTypeResetRequest,
			payload: &ResetRequest{
				TransactionID: 123,
				Cause:         E2APCause{RICCause: &[]RICCause{RICCauseSystemNotReady}[0]},
			},
		},
		{
			name:        "ErrorIndication",
			messageType: E2APMessageTypeErrorIndication,
			payload: &ErrorIndication{
				Cause: E2APCause{ProtocolCause: &[]ProtocolCause{ProtocolCauseSemanticError}[0]},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message := &E2APMessage{
				MessageType:   tc.messageType,
				TransactionID: 123,
				ProcedureCode: 1,
				Criticality:   CriticalityReject,
				Payload:       tc.payload,
				Timestamp:     time.Now(),
			}

			// Encode
			data, err := encoder.EncodeMessage(message)
			assert.NoError(t, err, "Failed to encode %s", tc.name)

			// Decode
			decoded, err := encoder.DecodeMessage(data)
			assert.NoError(t, err, "Failed to decode %s", tc.name)

			// Verify message type
			assert.Equal(t, tc.messageType, decoded.MessageType, "Message type mismatch for %s", tc.name)
			assert.NotNil(t, decoded.Payload, "Payload should not be nil for %s", tc.name)
		})
	}
}

// Helper function for fmt.Sprintf in concurrent test
func formatCorrelationID(i int) string {
	return fmt.Sprintf("correlation-%d", i)
}
