package e2

import (
	"encoding/json"
	"fmt"
)

// Message codecs implementing the MessageCodec interface
// These provide JSON-based encoding for HTTP transport (simplified ASN.1)

// E2SetupRequestCodec handles E2 Setup Request encoding/decoding
type E2SetupRequestCodec struct{}

func (c *E2SetupRequestCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeSetupRequest
}

func (c *E2SetupRequestCodec) Encode(message interface{}) ([]byte, error) {
	req, ok := message.(*E2SetupRequest)
	if !ok {
		return nil, fmt.Errorf("invalid message type for E2SetupRequest codec")
	}
	return json.Marshal(req)
}

func (c *E2SetupRequestCodec) Decode(data []byte) (interface{}, error) {
	var req E2SetupRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (c *E2SetupRequestCodec) Validate(message interface{}) error {
	req, ok := message.(*E2SetupRequest)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if req.GlobalE2NodeID.PLMNIdentity.MCC == "" {
		return fmt.Errorf("missing PLMN MCC")
	}
	if len(req.RANFunctionsList) == 0 {
		return fmt.Errorf("no RAN functions specified")
	}
	return nil
}

// E2SetupResponseCodec handles E2 Setup Response encoding/decoding
type E2SetupResponseCodec struct{}

func (c *E2SetupResponseCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeSetupResponse
}

func (c *E2SetupResponseCodec) Encode(message interface{}) ([]byte, error) {
	resp, ok := message.(*E2SetupResponse)
	if !ok {
		return nil, fmt.Errorf("invalid message type for E2SetupResponse codec")
	}
	return json.Marshal(resp)
}

func (c *E2SetupResponseCodec) Decode(data []byte) (interface{}, error) {
	var resp E2SetupResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *E2SetupResponseCodec) Validate(message interface{}) error {
	resp, ok := message.(*E2SetupResponse)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if resp.GlobalRICID.PLMNIdentity.MCC == "" {
		return fmt.Errorf("missing RIC PLMN MCC")
	}
	return nil
}

// E2SetupFailureCodec handles E2 Setup Failure encoding/decoding
type E2SetupFailureCodec struct{}

func (c *E2SetupFailureCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeSetupFailure
}

func (c *E2SetupFailureCodec) Encode(message interface{}) ([]byte, error) {
	fail, ok := message.(*E2SetupFailure)
	if !ok {
		return nil, fmt.Errorf("invalid message type for E2SetupFailure codec")
	}
	return json.Marshal(fail)
}

func (c *E2SetupFailureCodec) Decode(data []byte) (interface{}, error) {
	var fail E2SetupFailure
	if err := json.Unmarshal(data, &fail); err != nil {
		return nil, err
	}
	return &fail, nil
}

func (c *E2SetupFailureCodec) Validate(message interface{}) error {
	fail, ok := message.(*E2SetupFailure)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	// At least one cause must be present
	if fail.Cause.RICCause == nil && fail.Cause.RICServiceCause == nil &&
		fail.Cause.E2NodeCause == nil && fail.Cause.TransportCause == nil &&
		fail.Cause.ProtocolCause == nil && fail.Cause.MiscCause == nil {
		return fmt.Errorf("no cause specified")
	}
	return nil
}

// RICSubscriptionRequestCodec handles RIC Subscription Request encoding/decoding
type RICSubscriptionRequestCodec struct{}

func (c *RICSubscriptionRequestCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICSubscriptionRequest
}

func (c *RICSubscriptionRequestCodec) Encode(message interface{}) ([]byte, error) {
	req, ok := message.(*RICSubscriptionRequest)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICSubscriptionRequest codec")
	}
	return json.Marshal(req)
}

func (c *RICSubscriptionRequestCodec) Decode(data []byte) (interface{}, error) {
	var req RICSubscriptionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (c *RICSubscriptionRequestCodec) Validate(message interface{}) error {
	req, ok := message.(*RICSubscriptionRequest)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if req.RANFunctionID < 0 || req.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", req.RANFunctionID)
	}
	if len(req.RICSubscriptionDetails.RICActionToBeSetupList) == 0 {
		return fmt.Errorf("no RIC actions specified")
	}
	return nil
}

// RICSubscriptionResponseCodec handles RIC Subscription Response encoding/decoding
type RICSubscriptionResponseCodec struct{}

func (c *RICSubscriptionResponseCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICSubscriptionResponse
}

func (c *RICSubscriptionResponseCodec) Encode(message interface{}) ([]byte, error) {
	resp, ok := message.(*RICSubscriptionResponse)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICSubscriptionResponse codec")
	}
	return json.Marshal(resp)
}

func (c *RICSubscriptionResponseCodec) Decode(data []byte) (interface{}, error) {
	var resp RICSubscriptionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *RICSubscriptionResponseCodec) Validate(message interface{}) error {
	resp, ok := message.(*RICSubscriptionResponse)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if resp.RANFunctionID < 0 || resp.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", resp.RANFunctionID)
	}
	return nil
}

// RICSubscriptionFailureCodec handles RIC Subscription Failure encoding/decoding
type RICSubscriptionFailureCodec struct{}

func (c *RICSubscriptionFailureCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICSubscriptionFailure
}

func (c *RICSubscriptionFailureCodec) Encode(message interface{}) ([]byte, error) {
	fail, ok := message.(*RICSubscriptionFailure)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICSubscriptionFailure codec")
	}
	return json.Marshal(fail)
}

func (c *RICSubscriptionFailureCodec) Decode(data []byte) (interface{}, error) {
	var fail RICSubscriptionFailure
	if err := json.Unmarshal(data, &fail); err != nil {
		return nil, err
	}
	return &fail, nil
}

func (c *RICSubscriptionFailureCodec) Validate(message interface{}) error {
	fail, ok := message.(*RICSubscriptionFailure)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if fail.RANFunctionID < 0 || fail.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", fail.RANFunctionID)
	}
	return nil
}

// RICControlRequestCodec handles RIC Control Request encoding/decoding
type RICControlRequestCodec struct{}

func (c *RICControlRequestCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICControlRequest
}

func (c *RICControlRequestCodec) Encode(message interface{}) ([]byte, error) {
	req, ok := message.(*RICControlRequest)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICControlRequest codec")
	}
	return json.Marshal(req)
}

func (c *RICControlRequestCodec) Decode(data []byte) (interface{}, error) {
	var req RICControlRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (c *RICControlRequestCodec) Validate(message interface{}) error {
	req, ok := message.(*RICControlRequest)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if req.RANFunctionID < 0 || req.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", req.RANFunctionID)
	}
	if len(req.RICControlHeader) == 0 {
		return fmt.Errorf("empty control header")
	}
	if len(req.RICControlMessage) == 0 {
		return fmt.Errorf("empty control message")
	}
	return nil
}

// RICControlAcknowledgeCodec handles RIC Control Acknowledge encoding/decoding
type RICControlAcknowledgeCodec struct{}

func (c *RICControlAcknowledgeCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICControlAcknowledge
}

func (c *RICControlAcknowledgeCodec) Encode(message interface{}) ([]byte, error) {
	ack, ok := message.(*RICControlAcknowledge)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICControlAcknowledge codec")
	}
	return json.Marshal(ack)
}

func (c *RICControlAcknowledgeCodec) Decode(data []byte) (interface{}, error) {
	var ack RICControlAcknowledge
	if err := json.Unmarshal(data, &ack); err != nil {
		return nil, err
	}
	return &ack, nil
}

func (c *RICControlAcknowledgeCodec) Validate(message interface{}) error {
	ack, ok := message.(*RICControlAcknowledge)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if ack.RANFunctionID < 0 || ack.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", ack.RANFunctionID)
	}
	return nil
}

// RICControlFailureCodec handles RIC Control Failure encoding/decoding
type RICControlFailureCodec struct{}

func (c *RICControlFailureCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICControlFailure
}

func (c *RICControlFailureCodec) Encode(message interface{}) ([]byte, error) {
	fail, ok := message.(*RICControlFailure)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICControlFailure codec")
	}
	return json.Marshal(fail)
}

func (c *RICControlFailureCodec) Decode(data []byte) (interface{}, error) {
	var fail RICControlFailure
	if err := json.Unmarshal(data, &fail); err != nil {
		return nil, err
	}
	return &fail, nil
}

func (c *RICControlFailureCodec) Validate(message interface{}) error {
	fail, ok := message.(*RICControlFailure)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if fail.RANFunctionID < 0 || fail.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", fail.RANFunctionID)
	}
	return nil
}

// RICIndicationCodec handles RIC Indication encoding/decoding
type RICIndicationCodec struct{}

func (c *RICIndicationCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICIndication
}

func (c *RICIndicationCodec) Encode(message interface{}) ([]byte, error) {
	ind, ok := message.(*RICIndication)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICIndication codec")
	}
	return json.Marshal(ind)
}

func (c *RICIndicationCodec) Decode(data []byte) (interface{}, error) {
	var ind RICIndication
	if err := json.Unmarshal(data, &ind); err != nil {
		return nil, err
	}
	return &ind, nil
}

func (c *RICIndicationCodec) Validate(message interface{}) error {
	ind, ok := message.(*RICIndication)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if ind.RANFunctionID < 0 || ind.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", ind.RANFunctionID)
	}
	if ind.RICActionID < 0 || ind.RICActionID > 255 {
		return fmt.Errorf("invalid RIC action ID: %d", ind.RICActionID)
	}
	if len(ind.RICIndicationHeader) == 0 {
		return fmt.Errorf("empty indication header")
	}
	if len(ind.RICIndicationMessage) == 0 {
		return fmt.Errorf("empty indication message")
	}
	return nil
}

// RICServiceUpdateCodec handles RIC Service Update encoding/decoding
type RICServiceUpdateCodec struct{}

func (c *RICServiceUpdateCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICServiceUpdate
}

func (c *RICServiceUpdateCodec) Encode(message interface{}) ([]byte, error) {
	update, ok := message.(*RICServiceUpdate)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICServiceUpdate codec")
	}
	return json.Marshal(update)
}

func (c *RICServiceUpdateCodec) Decode(data []byte) (interface{}, error) {
	var update RICServiceUpdate
	if err := json.Unmarshal(data, &update); err != nil {
		return nil, err
	}
	return &update, nil
}

func (c *RICServiceUpdateCodec) Validate(message interface{}) error {
	update, ok := message.(*RICServiceUpdate)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if update.GlobalE2NodeID.PLMNIdentity.MCC == "" {
		return fmt.Errorf("missing PLMN MCC")
	}
	return nil
}

// RICServiceUpdateAcknowledgeCodec handles RIC Service Update Acknowledge encoding/decoding
type RICServiceUpdateAcknowledgeCodec struct{}

func (c *RICServiceUpdateAcknowledgeCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICServiceUpdateAcknowledge
}

func (c *RICServiceUpdateAcknowledgeCodec) Encode(message interface{}) ([]byte, error) {
	ack, ok := message.(*RICServiceUpdateAcknowledge)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICServiceUpdateAcknowledge codec")
	}
	return json.Marshal(ack)
}

func (c *RICServiceUpdateAcknowledgeCodec) Decode(data []byte) (interface{}, error) {
	var ack RICServiceUpdateAcknowledge
	if err := json.Unmarshal(data, &ack); err != nil {
		return nil, err
	}
	return &ack, nil
}

func (c *RICServiceUpdateAcknowledgeCodec) Validate(message interface{}) error {
	_, ok := message.(*RICServiceUpdateAcknowledge)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	return nil
}

// RICServiceUpdateFailureCodec handles RIC Service Update Failure encoding/decoding
type RICServiceUpdateFailureCodec struct{}

func (c *RICServiceUpdateFailureCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICServiceUpdateFailure
}

func (c *RICServiceUpdateFailureCodec) Encode(message interface{}) ([]byte, error) {
	fail, ok := message.(*RICServiceUpdateFailure)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICServiceUpdateFailure codec")
	}
	return json.Marshal(fail)
}

func (c *RICServiceUpdateFailureCodec) Decode(data []byte) (interface{}, error) {
	var fail RICServiceUpdateFailure
	if err := json.Unmarshal(data, &fail); err != nil {
		return nil, err
	}
	return &fail, nil
}

func (c *RICServiceUpdateFailureCodec) Validate(message interface{}) error {
	fail, ok := message.(*RICServiceUpdateFailure)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	// At least one cause must be present
	if fail.Cause.RICCause == nil && fail.Cause.RICServiceCause == nil &&
		fail.Cause.E2NodeCause == nil && fail.Cause.TransportCause == nil &&
		fail.Cause.ProtocolCause == nil && fail.Cause.MiscCause == nil {
		return fmt.Errorf("no cause specified")
	}
	return nil
}

// RICSubscriptionDeleteRequestCodec handles RIC Subscription Delete Request encoding/decoding
type RICSubscriptionDeleteRequestCodec struct{}

func (c *RICSubscriptionDeleteRequestCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICSubscriptionDeleteRequest
}

func (c *RICSubscriptionDeleteRequestCodec) Encode(message interface{}) ([]byte, error) {
	req, ok := message.(*RICSubscriptionDeleteRequest)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICSubscriptionDeleteRequest codec")
	}
	return json.Marshal(req)
}

func (c *RICSubscriptionDeleteRequestCodec) Decode(data []byte) (interface{}, error) {
	var req RICSubscriptionDeleteRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (c *RICSubscriptionDeleteRequestCodec) Validate(message interface{}) error {
	req, ok := message.(*RICSubscriptionDeleteRequest)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if req.RANFunctionID < 0 || req.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", req.RANFunctionID)
	}
	return nil
}

// RICSubscriptionDeleteResponseCodec handles RIC Subscription Delete Response encoding/decoding
type RICSubscriptionDeleteResponseCodec struct{}

func (c *RICSubscriptionDeleteResponseCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICSubscriptionDeleteResponse
}

func (c *RICSubscriptionDeleteResponseCodec) Encode(message interface{}) ([]byte, error) {
	resp, ok := message.(*RICSubscriptionDeleteResponse)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICSubscriptionDeleteResponse codec")
	}
	return json.Marshal(resp)
}

func (c *RICSubscriptionDeleteResponseCodec) Decode(data []byte) (interface{}, error) {
	var resp RICSubscriptionDeleteResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *RICSubscriptionDeleteResponseCodec) Validate(message interface{}) error {
	resp, ok := message.(*RICSubscriptionDeleteResponse)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if resp.RANFunctionID < 0 || resp.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", resp.RANFunctionID)
	}
	return nil
}

// RICSubscriptionDeleteFailureCodec handles RIC Subscription Delete Failure encoding/decoding
type RICSubscriptionDeleteFailureCodec struct{}

func (c *RICSubscriptionDeleteFailureCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeRICSubscriptionDeleteFailure
}

func (c *RICSubscriptionDeleteFailureCodec) Encode(message interface{}) ([]byte, error) {
	fail, ok := message.(*RICSubscriptionDeleteFailure)
	if !ok {
		return nil, fmt.Errorf("invalid message type for RICSubscriptionDeleteFailure codec")
	}
	return json.Marshal(fail)
}

func (c *RICSubscriptionDeleteFailureCodec) Decode(data []byte) (interface{}, error) {
	var fail RICSubscriptionDeleteFailure
	if err := json.Unmarshal(data, &fail); err != nil {
		return nil, err
	}
	return &fail, nil
}

func (c *RICSubscriptionDeleteFailureCodec) Validate(message interface{}) error {
	fail, ok := message.(*RICSubscriptionDeleteFailure)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	if fail.RANFunctionID < 0 || fail.RANFunctionID > 4095 {
		return fmt.Errorf("invalid RAN function ID: %d", fail.RANFunctionID)
	}
	return nil
}

// ResetRequestCodec handles Reset Request encoding/decoding
type ResetRequestCodec struct{}

func (c *ResetRequestCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeResetRequest
}

func (c *ResetRequestCodec) Encode(message interface{}) ([]byte, error) {
	req, ok := message.(*ResetRequest)
	if !ok {
		return nil, fmt.Errorf("invalid message type for ResetRequest codec")
	}
	return json.Marshal(req)
}

func (c *ResetRequestCodec) Decode(data []byte) (interface{}, error) {
	var req ResetRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (c *ResetRequestCodec) Validate(message interface{}) error {
	req, ok := message.(*ResetRequest)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	// At least one cause must be present
	if req.Cause.RICCause == nil && req.Cause.RICServiceCause == nil &&
		req.Cause.E2NodeCause == nil && req.Cause.TransportCause == nil &&
		req.Cause.ProtocolCause == nil && req.Cause.MiscCause == nil {
		return fmt.Errorf("no cause specified")
	}
	return nil
}

// ResetResponseCodec handles Reset Response encoding/decoding
type ResetResponseCodec struct{}

func (c *ResetResponseCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeResetResponse
}

func (c *ResetResponseCodec) Encode(message interface{}) ([]byte, error) {
	resp, ok := message.(*ResetResponse)
	if !ok {
		return nil, fmt.Errorf("invalid message type for ResetResponse codec")
	}
	return json.Marshal(resp)
}

func (c *ResetResponseCodec) Decode(data []byte) (interface{}, error) {
	var resp ResetResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *ResetResponseCodec) Validate(message interface{}) error {
	_, ok := message.(*ResetResponse)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	return nil
}

// ErrorIndicationCodec handles Error Indication encoding/decoding
type ErrorIndicationCodec struct{}

func (c *ErrorIndicationCodec) GetMessageType() E2APMessageType {
	return E2APMessageTypeErrorIndication
}

func (c *ErrorIndicationCodec) Encode(message interface{}) ([]byte, error) {
	err, ok := message.(*ErrorIndication)
	if !ok {
		return nil, fmt.Errorf("invalid message type for ErrorIndication codec")
	}
	return json.Marshal(err)
}

func (c *ErrorIndicationCodec) Decode(data []byte) (interface{}, error) {
	var err ErrorIndication
	if err := json.Unmarshal(data, &err); err != nil {
		return nil, err
	}
	return &err, nil
}

func (c *ErrorIndicationCodec) Validate(message interface{}) error {
	errInd, ok := message.(*ErrorIndication)
	if !ok {
		return fmt.Errorf("invalid message type")
	}
	// At least one cause must be present
	if errInd.Cause.RICCause == nil && errInd.Cause.RICServiceCause == nil &&
		errInd.Cause.E2NodeCause == nil && errInd.Cause.TransportCause == nil &&
		errInd.Cause.ProtocolCause == nil && errInd.Cause.MiscCause == nil {
		return fmt.Errorf("no cause specified")
	}
	return nil
}
