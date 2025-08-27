package e2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2SetupRequestCodec(t *testing.T) {
	codec := &E2SetupRequestCodec{}

	// Test GetMessageType
	assert.Equal(t, E2APMessageTypeSetupRequest, codec.GetMessageType())

	// Test valid message
	req := &E2SetupRequest{
		GlobalE2NodeID: GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
		},
		RANFunctionsList: []RANFunctionItem{
			{
				RANFunctionID:         1,
				RANFunctionDefinition: []byte("test"),
				RANFunctionRevision:   1,
			},
		},
	}

	// Test Validate
	err := codec.Validate(req)
	assert.NoError(t, err)

	// Test Encode
	data, err := codec.Encode(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test Decode
	decoded, err := codec.Decode(data)
	assert.NoError(t, err)

	decodedReq, ok := decoded.(*E2SetupRequest)
	require.True(t, ok)
	assert.Equal(t, req.GlobalE2NodeID.PLMNIdentity.MCC, decodedReq.GlobalE2NodeID.PLMNIdentity.MCC)
	assert.Equal(t, req.GlobalE2NodeID.PLMNIdentity.MNC, decodedReq.GlobalE2NodeID.PLMNIdentity.MNC)
	assert.Len(t, decodedReq.RANFunctionsList, 1)
}

func TestE2SetupRequestCodecValidation(t *testing.T) {
	codec := &E2SetupRequestCodec{}

	tests := []struct {
		name    string
		req     interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "wrong type",
			req:     "not a request",
			wantErr: true,
			errMsg:  "invalid message type",
		},
		{
			name: "missing MCC",
			req: &E2SetupRequest{
				GlobalE2NodeID: GlobalE2NodeID{
					PLMNIdentity: PLMNIdentity{MNC: "01"},
				},
				RANFunctionsList: []RANFunctionItem{{RANFunctionID: 1}},
			},
			wantErr: true,
			errMsg:  "missing PLMN MCC",
		},
		{
			name: "no RAN functions",
			req: &E2SetupRequest{
				GlobalE2NodeID: GlobalE2NodeID{
					PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
				},
				RANFunctionsList: []RANFunctionItem{},
			},
			wantErr: true,
			errMsg:  "no RAN functions specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := codec.Validate(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestE2SetupResponseCodec(t *testing.T) {
	codec := &E2SetupResponseCodec{}

	assert.Equal(t, E2APMessageTypeSetupResponse, codec.GetMessageType())

	resp := &E2SetupResponse{
		GlobalRICID: GlobalRICID{
			PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
			RICID:        RICID(12345),
		},
		RANFunctionsAccepted: []RANFunctionIDItem{
			{RANFunctionID: 1, RANFunctionRevision: 1},
		},
	}

	// Test validation
	err := codec.Validate(resp)
	assert.NoError(t, err)

	// Test encode/decode cycle
	data, err := codec.Encode(resp)
	assert.NoError(t, err)

	decoded, err := codec.Decode(data)
	assert.NoError(t, err)

	decodedResp, ok := decoded.(*E2SetupResponse)
	require.True(t, ok)
	assert.Equal(t, resp.GlobalRICID.PLMNIdentity.MCC, decodedResp.GlobalRICID.PLMNIdentity.MCC)
}

func TestE2SetupFailureCodec(t *testing.T) {
	codec := &E2SetupFailureCodec{}

	assert.Equal(t, E2APMessageTypeSetupFailure, codec.GetMessageType())

	cause := RICCauseSystemNotReady
	fail := &E2SetupFailure{
		Cause: E2APCause{
			RICCause: &cause,
		},
	}

	// Test validation
	err := codec.Validate(fail)
	assert.NoError(t, err)

	// Test encode/decode cycle
	data, err := codec.Encode(fail)
	assert.NoError(t, err)

	decoded, err := codec.Decode(data)
	assert.NoError(t, err)

	decodedFail, ok := decoded.(*E2SetupFailure)
	require.True(t, ok)
	assert.NotNil(t, decodedFail.Cause.RICCause)
	assert.Equal(t, cause, *decodedFail.Cause.RICCause)
}

func TestRICSubscriptionRequestCodec(t *testing.T) {
	codec := &RICSubscriptionRequestCodec{}

	assert.Equal(t, E2APMessageTypeRICSubscriptionRequest, codec.GetMessageType())

	req := &RICSubscriptionRequest{
		RICRequestID:  RICRequestID{RICRequestorID: 1, RICInstanceID: 1},
		RANFunctionID: 10,
		RICSubscriptionDetails: RICSubscriptionDetails{
			RICEventTriggerDefinition: []byte("trigger"),
			RICActionToBeSetupList: []RICActionToBeSetupItem{
				{RICActionID: 1, RICActionType: RICActionTypeReport},
			},
		},
	}

	// Test validation
	err := codec.Validate(req)
	assert.NoError(t, err)

	// Test encode/decode cycle
	data, err := codec.Encode(req)
	assert.NoError(t, err)

	decoded, err := codec.Decode(data)
	assert.NoError(t, err)

	decodedReq, ok := decoded.(*RICSubscriptionRequest)
	require.True(t, ok)
	assert.Equal(t, req.RICRequestID, decodedReq.RICRequestID)
	assert.Equal(t, req.RANFunctionID, decodedReq.RANFunctionID)
}

func TestRICSubscriptionRequestCodecValidation(t *testing.T) {
	codec := &RICSubscriptionRequestCodec{}

	tests := []struct {
		name    string
		req     interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "wrong type",
			req:     123,
			wantErr: true,
			errMsg:  "invalid message type",
		},
		{
			name: "invalid RAN function ID - negative",
			req: &RICSubscriptionRequest{
				RANFunctionID: -1,
				RICSubscriptionDetails: RICSubscriptionDetails{
					RICActionToBeSetupList: []RICActionToBeSetupItem{{RICActionID: 1}},
				},
			},
			wantErr: true,
			errMsg:  "invalid RAN function ID",
		},
		{
			name: "invalid RAN function ID - too large",
			req: &RICSubscriptionRequest{
				RANFunctionID: 5000,
				RICSubscriptionDetails: RICSubscriptionDetails{
					RICActionToBeSetupList: []RICActionToBeSetupItem{{RICActionID: 1}},
				},
			},
			wantErr: true,
			errMsg:  "invalid RAN function ID",
		},
		{
			name: "no RIC actions",
			req: &RICSubscriptionRequest{
				RANFunctionID: 1,
				RICSubscriptionDetails: RICSubscriptionDetails{
					RICActionToBeSetupList: []RICActionToBeSetupItem{},
				},
			},
			wantErr: true,
			errMsg:  "no RIC actions specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := codec.Validate(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRICControlRequestCodec(t *testing.T) {
	codec := &RICControlRequestCodec{}

	assert.Equal(t, E2APMessageTypeRICControlRequest, codec.GetMessageType())

	req := &RICControlRequest{
		RICRequestID:      RICRequestID{RICRequestorID: 1, RICInstanceID: 1},
		RANFunctionID:     1,
		RICControlHeader:  []byte("header"),
		RICControlMessage: []byte("message"),
	}

	// Test validation
	err := codec.Validate(req)
	assert.NoError(t, err)

	// Test encode/decode cycle
	data, err := codec.Encode(req)
	assert.NoError(t, err)

	decoded, err := codec.Decode(data)
	assert.NoError(t, err)

	decodedReq, ok := decoded.(*RICControlRequest)
	require.True(t, ok)
	assert.Equal(t, req.RICRequestID, decodedReq.RICRequestID)
	assert.Equal(t, req.RANFunctionID, decodedReq.RANFunctionID)
	assert.Equal(t, req.RICControlHeader, decodedReq.RICControlHeader)
	assert.Equal(t, req.RICControlMessage, decodedReq.RICControlMessage)
}

func TestRICControlRequestCodecValidation(t *testing.T) {
	codec := &RICControlRequestCodec{}

	tests := []struct {
		name    string
		req     interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty control header",
			req: &RICControlRequest{
				RANFunctionID:     1,
				RICControlHeader:  []byte{},
				RICControlMessage: []byte("message"),
			},
			wantErr: true,
			errMsg:  "empty control header",
		},
		{
			name: "empty control message",
			req: &RICControlRequest{
				RANFunctionID:     1,
				RICControlHeader:  []byte("header"),
				RICControlMessage: []byte{},
			},
			wantErr: true,
			errMsg:  "empty control message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := codec.Validate(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRICIndicationCodec(t *testing.T) {
	codec := &RICIndicationCodec{}

	assert.Equal(t, E2APMessageTypeRICIndication, codec.GetMessageType())

	ind := &RICIndication{
		RICRequestID:         RICRequestID{RICRequestorID: 1, RICInstanceID: 1},
		RANFunctionID:        1,
		RICActionID:          1,
		RICIndicationType:    RICIndicationTypeReport,
		RICIndicationHeader:  []byte("header"),
		RICIndicationMessage: []byte("message"),
	}

	// Test validation
	err := codec.Validate(ind)
	assert.NoError(t, err)

	// Test encode/decode cycle
	data, err := codec.Encode(ind)
	assert.NoError(t, err)

	decoded, err := codec.Decode(data)
	assert.NoError(t, err)

	decodedInd, ok := decoded.(*RICIndication)
	require.True(t, ok)
	assert.Equal(t, ind.RICRequestID, decodedInd.RICRequestID)
	assert.Equal(t, ind.RANFunctionID, decodedInd.RANFunctionID)
	assert.Equal(t, ind.RICActionID, decodedInd.RICActionID)
	assert.Equal(t, ind.RICIndicationType, decodedInd.RICIndicationType)
}

func TestRICIndicationCodecValidation(t *testing.T) {
	codec := &RICIndicationCodec{}

	tests := []struct {
		name    string
		ind     interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid RIC action ID - negative",
			ind: &RICIndication{
				RANFunctionID:        1,
				RICActionID:          -1,
				RICIndicationHeader:  []byte("header"),
				RICIndicationMessage: []byte("message"),
			},
			wantErr: true,
			errMsg:  "invalid RIC action ID",
		},
		{
			name: "invalid RIC action ID - too large",
			ind: &RICIndication{
				RANFunctionID:        1,
				RICActionID:          300,
				RICIndicationHeader:  []byte("header"),
				RICIndicationMessage: []byte("message"),
			},
			wantErr: true,
			errMsg:  "invalid RIC action ID",
		},
		{
			name: "empty indication header",
			ind: &RICIndication{
				RANFunctionID:        1,
				RICActionID:          1,
				RICIndicationHeader:  []byte{},
				RICIndicationMessage: []byte("message"),
			},
			wantErr: true,
			errMsg:  "empty indication header",
		},
		{
			name: "empty indication message",
			ind: &RICIndication{
				RANFunctionID:        1,
				RICActionID:          1,
				RICIndicationHeader:  []byte("header"),
				RICIndicationMessage: []byte{},
			},
			wantErr: true,
			errMsg:  "empty indication message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := codec.Validate(tt.ind)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRICServiceUpdateCodec(t *testing.T) {
	codec := &RICServiceUpdateCodec{}

	assert.Equal(t, E2APMessageTypeRICServiceUpdate, codec.GetMessageType())

	update := &RICServiceUpdate{
		GlobalE2NodeID: GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
		},
		RANFunctionsAdded: []RANFunctionItem{
			{RANFunctionID: 1, RANFunctionRevision: 1},
		},
	}

	// Test validation
	err := codec.Validate(update)
	assert.NoError(t, err)

	// Test encode/decode cycle
	data, err := codec.Encode(update)
	assert.NoError(t, err)

	decoded, err := codec.Decode(data)
	assert.NoError(t, err)

	decodedUpdate, ok := decoded.(*RICServiceUpdate)
	require.True(t, ok)
	assert.Equal(t, update.GlobalE2NodeID.PLMNIdentity.MCC, decodedUpdate.GlobalE2NodeID.PLMNIdentity.MCC)
}

func TestResetRequestCodec(t *testing.T) {
	codec := &ResetRequestCodec{}

	assert.Equal(t, E2APMessageTypeResetRequest, codec.GetMessageType())

	cause := RICCauseSystemNotReady
	req := &ResetRequest{
		TransactionID: 123,
		Cause: E2APCause{
			RICCause: &cause,
		},
	}

	// Test validation
	err := codec.Validate(req)
	assert.NoError(t, err)

	// Test encode/decode cycle
	data, err := codec.Encode(req)
	assert.NoError(t, err)

	decoded, err := codec.Decode(data)
	assert.NoError(t, err)

	decodedReq, ok := decoded.(*ResetRequest)
	require.True(t, ok)
	assert.Equal(t, req.TransactionID, decodedReq.TransactionID)
	assert.NotNil(t, decodedReq.Cause.RICCause)
}

func TestErrorIndicationCodec(t *testing.T) {
	codec := &ErrorIndicationCodec{}

	assert.Equal(t, E2APMessageTypeErrorIndication, codec.GetMessageType())

	protocolCause := ProtocolCauseSemanticError
	err := &ErrorIndication{
		Cause: E2APCause{
			ProtocolCause: &protocolCause,
		},
	}

	// Test validation
	validationErr := codec.Validate(err)
	assert.NoError(t, validationErr)

	// Test encode/decode cycle
	data, encodeErr := codec.Encode(err)
	assert.NoError(t, encodeErr)

	decoded, decodeErr := codec.Decode(data)
	assert.NoError(t, decodeErr)

	decodedErr, ok := decoded.(*ErrorIndication)
	require.True(t, ok)
	assert.NotNil(t, decodedErr.Cause.ProtocolCause)
	assert.Equal(t, protocolCause, *decodedErr.Cause.ProtocolCause)
}

func TestCodecInvalidTypes(t *testing.T) {
	codecs := []MessageCodec{
		&E2SetupRequestCodec{},
		&RICSubscriptionRequestCodec{},
		&RICControlRequestCodec{},
		&RICIndicationCodec{},
	}

	for _, codec := range codecs {
		t.Run(codec.GetMessageType().String(), func(t *testing.T) {
			// Test encoding with wrong type
			_, err := codec.Encode("wrong type")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid message type")

			// Test validation with wrong type
			err = codec.Validate("wrong type")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid message type")
		})
	}
}

func TestCodecDecodeInvalidJSON(t *testing.T) {
	codecs := []MessageCodec{
		&E2SetupRequestCodec{},
		&E2SetupResponseCodec{},
		&RICSubscriptionRequestCodec{},
		&RICControlRequestCodec{},
		&RICIndicationCodec{},
	}

	invalidJSON := []byte(`{"invalid": json}`)

	for _, codec := range codecs {
		t.Run(codec.GetMessageType().String(), func(t *testing.T) {
			_, err := codec.Decode(invalidJSON)
			assert.Error(t, err)
		})
	}
}

// Note: String() method for E2APMessageType is defined in e2_interface_test.go
