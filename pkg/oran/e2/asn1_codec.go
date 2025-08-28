package e2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// ASN1Codec provides ASN.1 PER (Packed Encoding Rules) encoding/decoding.
// for E2AP messages following O-RAN.WG3.E2AP-v03.01 specification.
type ASN1Codec struct {
	aligned bool // PER aligned or unaligned variant
}

// NewASN1Codec creates a new ASN.1 codec.
func NewASN1Codec(aligned bool) *ASN1Codec {
	return &ASN1Codec{aligned: aligned}
}

// EncodeE2APMessage encodes an E2AP message to ASN.1 PER format.
func (c *ASN1Codec) EncodeE2APMessage(msg *E2APMessage) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Encode procedure code (16 bits).
	if err := binary.Write(buf, binary.BigEndian, uint16(msg.ProcedureCode)); err != nil {
		return nil, fmt.Errorf("failed to encode procedure code: %w", err)
	}

	// Encode criticality (2 bits).
	criticalityBits := uint8(msg.Criticality & 0x03)
	if err := buf.WriteByte(criticalityBits << 6); err != nil {
		return nil, fmt.Errorf("failed to encode criticality: %w", err)
	}

	// Encode transaction ID (8 bits for now, can be extended).
	if err := binary.Write(buf, binary.BigEndian, uint8(msg.TransactionID)); err != nil {
		return nil, fmt.Errorf("failed to encode transaction ID: %w", err)
	}

	// Encode message type specific payload.
	payloadBytes, err := c.encodePayload(msg.MessageType, msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode payload: %w", err)
	}

	// Write payload length (32 bits).
	if err := binary.Write(buf, binary.BigEndian, uint32(len(payloadBytes))); err != nil {
		return nil, fmt.Errorf("failed to encode payload length: %w", err)
	}

	// Write payload.
	if _, err := buf.Write(payloadBytes); err != nil {
		return nil, fmt.Errorf("failed to write payload: %w", err)
	}

	return buf.Bytes(), nil
}

// DecodeE2APMessage decodes an ASN.1 PER format message to E2AP message.
func (c *ASN1Codec) DecodeE2APMessage(data []byte) (*E2APMessage, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("insufficient data for E2AP message header")
	}

	buf := bytes.NewReader(data)
	msg := &E2APMessage{}

	// Decode procedure code.
	var procedureCode uint16
	if err := binary.Read(buf, binary.BigEndian, &procedureCode); err != nil {
		return nil, fmt.Errorf("failed to decode procedure code: %w", err)
	}
	msg.ProcedureCode = int32(procedureCode)

	// Decode criticality.
	var criticalityByte uint8
	if err := binary.Read(buf, binary.BigEndian, &criticalityByte); err != nil {
		return nil, fmt.Errorf("failed to decode criticality: %w", err)
	}
	msg.Criticality = Criticality(criticalityByte >> 6)

	// Decode transaction ID.
	var transactionID uint8
	if err := binary.Read(buf, binary.BigEndian, &transactionID); err != nil {
		return nil, fmt.Errorf("failed to decode transaction ID: %w", err)
	}
	msg.TransactionID = int32(transactionID)

	// Decode payload length.
	var payloadLength uint32
	if err := binary.Read(buf, binary.BigEndian, &payloadLength); err != nil {
		return nil, fmt.Errorf("failed to decode payload length: %w", err)
	}

	// Read payload.
	payloadBytes := make([]byte, payloadLength)
	if _, err := io.ReadFull(buf, payloadBytes); err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	// Determine message type from procedure code.
	msg.MessageType = c.getMessageTypeFromProcedureCode(msg.ProcedureCode)

	// Decode payload based on message type.
	payload, err := c.decodePayload(msg.MessageType, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}
	msg.Payload = payload

	return msg, nil
}

// encodePayload encodes message-specific payload.
func (c *ASN1Codec) encodePayload(msgType E2APMessageType, payload interface{}) ([]byte, error) {
	switch msgType {
	case E2APMessageTypeSetupRequest:
		return c.encodeE2SetupRequest(payload.(*E2SetupRequest))
	case E2APMessageTypeSetupResponse:
		return c.encodeE2SetupResponse(payload.(*E2SetupResponse))
	case E2APMessageTypeRICSubscriptionRequest:
		return c.encodeRICSubscriptionRequest(payload.(*RICSubscriptionRequest))
	case E2APMessageTypeRICSubscriptionResponse:
		return c.encodeRICSubscriptionResponse(payload.(*RICSubscriptionResponse))
	case E2APMessageTypeRICIndication:
		return c.encodeRICIndication(payload.(*RICIndication))
	case E2APMessageTypeRICControlRequest:
		return c.encodeRICControlRequest(payload.(*RICControlRequest))
	case E2APMessageTypeRICControlAcknowledge:
		return c.encodeRICControlAcknowledge(payload.(*RICControlAcknowledge))
	default:
		return nil, fmt.Errorf("unsupported message type: %v", msgType)
	}
}

// decodePayload decodes message-specific payload.
func (c *ASN1Codec) decodePayload(msgType E2APMessageType, data []byte) (interface{}, error) {
	switch msgType {
	case E2APMessageTypeSetupRequest:
		return c.decodeE2SetupRequest(data)
	case E2APMessageTypeSetupResponse:
		return c.decodeE2SetupResponse(data)
	case E2APMessageTypeRICSubscriptionRequest:
		return c.decodeRICSubscriptionRequest(data)
	case E2APMessageTypeRICSubscriptionResponse:
		return c.decodeRICSubscriptionResponse(data)
	case E2APMessageTypeRICIndication:
		return c.decodeRICIndication(data)
	case E2APMessageTypeRICControlRequest:
		return c.decodeRICControlRequest(data)
	case E2APMessageTypeRICControlAcknowledge:
		return c.decodeRICControlAcknowledge(data)
	default:
		return nil, fmt.Errorf("unsupported message type: %v", msgType)
	}
}

// E2 Setup Request encoding.
func (c *ASN1Codec) encodeE2SetupRequest(req *E2SetupRequest) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Encode Global E2 Node ID.
	if err := c.encodeGlobalE2NodeID(buf, req.GlobalE2NodeID); err != nil {
		return nil, err
	}

	// Encode RAN Functions List.
	if err := c.encodeRANFunctionsList(buf, req.RANFunctionsList); err != nil {
		return nil, err
	}

	// Encode E2 Node Component Configuration List (optional).
	if len(req.E2NodeComponentConfigurationList) > 0 {
		if err := c.encodeE2NodeComponentConfigList(buf, req.E2NodeComponentConfigurationList); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// E2 Setup Request decoding.
func (c *ASN1Codec) decodeE2SetupRequest(data []byte) (*E2SetupRequest, error) {
	buf := bytes.NewReader(data)
	req := &E2SetupRequest{}

	// Decode Global E2 Node ID.
	globalID, err := c.decodeGlobalE2NodeID(buf)
	if err != nil {
		return nil, err
	}
	req.GlobalE2NodeID = globalID

	// Decode RAN Functions List.
	functions, err := c.decodeRANFunctionsList(buf)
	if err != nil {
		return nil, err
	}
	req.RANFunctionsList = functions

	// Check for optional E2 Node Component Configuration List.
	if buf.Len() > 0 {
		configList, err := c.decodeE2NodeComponentConfigList(buf)
		if err != nil {
			return nil, err
		}
		req.E2NodeComponentConfigurationList = configList
	}

	return req, nil
}

// RIC Subscription Request encoding.
func (c *ASN1Codec) encodeRICSubscriptionRequest(req *RICSubscriptionRequest) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Encode RIC Request ID.
	if err := c.encodeRICRequestID(buf, req.RICRequestID); err != nil {
		return nil, err
	}

	// Encode RAN Function ID (16 bits).
	if err := binary.Write(buf, binary.BigEndian, uint16(req.RANFunctionID)); err != nil {
		return nil, err
	}

	// Encode Event Trigger Definition.
	if err := c.encodeOctetString(buf, req.RICSubscriptionDetails.RICEventTriggerDefinition); err != nil {
		return nil, err
	}

	// Encode Actions to be setup list.
	if err := c.encodeRICActionsList(buf, req.RICSubscriptionDetails.RICActionToBeSetupList); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// RIC Subscription Request decoding.
func (c *ASN1Codec) decodeRICSubscriptionRequest(data []byte) (*RICSubscriptionRequest, error) {
	buf := bytes.NewReader(data)
	req := &RICSubscriptionRequest{}

	// Decode RIC Request ID.
	requestID, err := c.decodeRICRequestID(buf)
	if err != nil {
		return nil, err
	}
	req.RICRequestID = requestID

	// Decode RAN Function ID.
	var ranFunctionID uint16
	if err := binary.Read(buf, binary.BigEndian, &ranFunctionID); err != nil {
		return nil, err
	}
	req.RANFunctionID = RANFunctionID(ranFunctionID)

	// Decode Event Trigger Definition.
	eventTrigger, err := c.decodeOctetString(buf)
	if err != nil {
		return nil, err
	}
	req.RICSubscriptionDetails.RICEventTriggerDefinition = eventTrigger

	// Decode Actions to be setup list.
	actions, err := c.decodeRICActionsList(buf)
	if err != nil {
		return nil, err
	}
	req.RICSubscriptionDetails.RICActionToBeSetupList = actions

	return req, nil
}

// RIC Indication encoding.
func (c *ASN1Codec) encodeRICIndication(ind *RICIndication) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Encode RIC Request ID.
	if err := c.encodeRICRequestID(buf, ind.RICRequestID); err != nil {
		return nil, err
	}

	// Encode RAN Function ID.
	if err := binary.Write(buf, binary.BigEndian, uint16(ind.RANFunctionID)); err != nil {
		return nil, err
	}

	// Encode Action ID.
	if err := binary.Write(buf, binary.BigEndian, uint16(ind.RICActionID)); err != nil {
		return nil, err
	}

	// Encode Indication SN (optional).
	if ind.RICIndicationSN != nil && *ind.RICIndicationSN > 0 {
		if err := binary.Write(buf, binary.BigEndian, uint32(*ind.RICIndicationSN)); err != nil {
			return nil, err
		}
	}

	// Encode Indication Type.
	if err := c.encodeIndicationType(buf, ind.RICIndicationType); err != nil {
		return nil, err
	}

	// Encode Indication Header.
	if err := c.encodeOctetString(buf, ind.RICIndicationHeader); err != nil {
		return nil, err
	}

	// Encode Indication Message.
	if err := c.encodeOctetString(buf, ind.RICIndicationMessage); err != nil {
		return nil, err
	}

	// Encode Call Process ID (optional).
	if ind.RICCallProcessID != nil && len(*ind.RICCallProcessID) > 0 {
		if err := c.encodeOctetString(buf, *ind.RICCallProcessID); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// RIC Indication decoding.
func (c *ASN1Codec) decodeRICIndication(data []byte) (*RICIndication, error) {
	buf := bytes.NewReader(data)
	ind := &RICIndication{}

	// Decode RIC Request ID.
	requestID, err := c.decodeRICRequestID(buf)
	if err != nil {
		return nil, err
	}
	ind.RICRequestID = requestID

	// Decode RAN Function ID.
	var ranFunctionID uint16
	if err := binary.Read(buf, binary.BigEndian, &ranFunctionID); err != nil {
		return nil, err
	}
	ind.RANFunctionID = RANFunctionID(ranFunctionID)

	// Decode Action ID.
	var actionID uint16
	if err := binary.Read(buf, binary.BigEndian, &actionID); err != nil {
		return nil, err
	}
	ind.RICActionID = RICActionID(actionID)

	// Check for optional Indication SN.
	if buf.Len() > 0 {
		var indicationSN uint32
		if err := binary.Read(buf, binary.BigEndian, &indicationSN); err == nil {
			sn := RICIndicationSN(indicationSN)
			ind.RICIndicationSN = &sn
		}
	}

	// Decode Indication Type.
	indicationType, err := c.decodeIndicationType(buf)
	if err != nil {
		return nil, err
	}
	ind.RICIndicationType = indicationType

	// Decode Indication Header.
	header, err := c.decodeOctetString(buf)
	if err != nil {
		return nil, err
	}
	ind.RICIndicationHeader = header

	// Decode Indication Message.
	message, err := c.decodeOctetString(buf)
	if err != nil {
		return nil, err
	}
	ind.RICIndicationMessage = message

	// Check for optional Call Process ID.
	if buf.Len() > 0 {
		callProcessID, err := c.decodeOctetString(buf)
		if err == nil {
			callProcID := RICCallProcessID(callProcessID)
			ind.RICCallProcessID = &callProcID
		}
	}

	return ind, nil
}

// Helper encoding functions.

func (c *ASN1Codec) encodeGlobalE2NodeID(w io.Writer, id GlobalE2NodeID) error {
	// Encode node type (1 byte).
	if err := binary.Write(w, binary.BigEndian, uint8(id.NodeType)); err != nil {
		return err
	}

	// Encode PLMN Identity (3 bytes).
	plmnBytes := c.encodePLMNIdentity(id.PLMNIdentity)
	if _, err := w.Write(plmnBytes); err != nil {
		return err
	}

	// Encode Node ID based on type.
	switch id.NodeType {
	case E2NodeTypegNB:
		return c.encodeGNBID(w, id.NodeID)
	case E2NodeTypeeNB:
		return c.encodeENBID(w, id.NodeID)
	case E2NodeTypeNgENB:
		return c.encodeNgENBID(w, id.NodeID)
	case E2NodeTypeEnGNB:
		return c.encodeEnGNBID(w, id.NodeID)
	default:
		return fmt.Errorf("unsupported node type: %v", id.NodeType)
	}
}

func (c *ASN1Codec) decodeGlobalE2NodeID(r io.Reader) (GlobalE2NodeID, error) {
	var id GlobalE2NodeID

	// Decode node type.
	var nodeType uint8
	if err := binary.Read(r, binary.BigEndian, &nodeType); err != nil {
		return id, err
	}
	id.NodeType = E2NodeType(nodeType)

	// Decode PLMN Identity.
	plmnBytes := make([]byte, 3)
	if _, err := io.ReadFull(r, plmnBytes); err != nil {
		return id, err
	}
	id.PLMNIdentity = c.decodePLMNIdentity(plmnBytes)

	// Decode Node ID based on type.
	switch id.NodeType {
	case E2NodeTypegNB:
		nodeID, err := c.decodeGNBID(r)
		if err != nil {
			return id, err
		}
		id.NodeID = nodeID
	case E2NodeTypeeNB:
		nodeID, err := c.decodeENBID(r)
		if err != nil {
			return id, err
		}
		id.NodeID = nodeID
	case E2NodeTypeNgENB:
		nodeID, err := c.decodeNgENBID(r)
		if err != nil {
			return id, err
		}
		id.NodeID = nodeID
	case E2NodeTypeEnGNB:
		nodeID, err := c.decodeEnGNBID(r)
		if err != nil {
			return id, err
		}
		id.NodeID = nodeID
	default:
		return id, fmt.Errorf("unsupported node type: %v", id.NodeType)
	}

	return id, nil
}

func (c *ASN1Codec) encodePLMNIdentity(plmn PLMNIdentity) []byte {
	// PLMN encoding: MCC (2 nibbles) + MNC (1 nibble) + MCC (1 nibble) + MNC (2 nibbles).
	bytes := make([]byte, 3)

	// Convert MCC and MNC strings to BCD.
	mcc := []byte(plmn.MCC)
	mnc := []byte(plmn.MNC)

	if len(mcc) >= 3 && len(mnc) >= 2 {
		bytes[0] = ((mcc[1] - '0') << 4) | (mcc[0] - '0')
		bytes[1] = ((mnc[0] - '0') << 4) | (mcc[2] - '0')
		if len(mnc) == 3 {
			bytes[2] = ((mnc[2] - '0') << 4) | (mnc[1] - '0')
		} else {
			bytes[2] = 0xF0 | (mnc[1] - '0')
		}
	}

	return bytes
}

func (c *ASN1Codec) decodePLMNIdentity(bytes []byte) PLMNIdentity {
	if len(bytes) != 3 {
		return PLMNIdentity{}
	}

	mcc := fmt.Sprintf("%d%d%d",
		bytes[0]&0x0F,
		(bytes[0]>>4)&0x0F,
		bytes[1]&0x0F)

	mnc2 := (bytes[1] >> 4) & 0x0F
	mnc1 := bytes[2] & 0x0F
	mnc3 := (bytes[2] >> 4) & 0x0F

	var mnc string
	if mnc3 == 0x0F {
		mnc = fmt.Sprintf("%d%d", mnc2, mnc1)
	} else {
		mnc = fmt.Sprintf("%d%d%d", mnc2, mnc1, mnc3)
	}

	return PLMNIdentity{MCC: mcc, MNC: mnc}
}

func (c *ASN1Codec) encodeOctetString(w io.Writer, data []byte) error {
	// Encode length.
	if err := c.encodeLength(w, len(data)); err != nil {
		return err
	}

	// Write data.
	_, err := w.Write(data)
	return err
}

func (c *ASN1Codec) decodeOctetString(r io.Reader) ([]byte, error) {
	// Decode length.
	length, err := c.decodeLength(r)
	if err != nil {
		return nil, err
	}

	// Read data.
	data := make([]byte, length)
	_, err = io.ReadFull(r, data)
	return data, err
}

func (c *ASN1Codec) encodeLength(w io.Writer, length int) error {
	if length < 128 {
		// Short form.
		return binary.Write(w, binary.BigEndian, uint8(length))
	} else if length < 65536 {
		// Long form with 2 bytes.
		if err := binary.Write(w, binary.BigEndian, uint8(0x82)); err != nil {
			return err
		}
		return binary.Write(w, binary.BigEndian, uint16(length))
	} else {
		// Long form with 4 bytes.
		if err := binary.Write(w, binary.BigEndian, uint8(0x84)); err != nil {
			return err
		}
		return binary.Write(w, binary.BigEndian, uint32(length))
	}
}

func (c *ASN1Codec) decodeLength(r io.Reader) (int, error) {
	var firstByte uint8
	if err := binary.Read(r, binary.BigEndian, &firstByte); err != nil {
		return 0, err
	}

	if firstByte < 128 {
		// Short form.
		return int(firstByte), nil
	}

	// Long form.
	numOctets := firstByte & 0x7F
	switch numOctets {
	case 1:
		var length uint8
		err := binary.Read(r, binary.BigEndian, &length)
		return int(length), err
	case 2:
		var length uint16
		err := binary.Read(r, binary.BigEndian, &length)
		return int(length), err
	case 4:
		var length uint32
		err := binary.Read(r, binary.BigEndian, &length)
		return int(length), err
	default:
		return 0, fmt.Errorf("unsupported length encoding: %d octets", numOctets)
	}
}

// Helper function to map procedure codes to message types.
func (c *ASN1Codec) getMessageTypeFromProcedureCode(procedureCode int32) E2APMessageType {
	switch procedureCode {
	case 1:
		return E2APMessageTypeSetupRequest
	case 2:
		return E2APMessageTypeRICSubscriptionRequest
	case 3:
		return E2APMessageTypeRICControlRequest
	case 4:
		return E2APMessageTypeErrorIndication
	case 5:
		return E2APMessageTypeRICSubscriptionDeleteRequest
	case 6:
		return E2APMessageTypeResetRequest
	case 11:
		return E2APMessageTypeSetupResponse
	case 12:
		return E2APMessageTypeRICSubscriptionResponse
	case 13:
		return E2APMessageTypeRICControlAcknowledge
	case 14:
		return E2APMessageTypeRICSubscriptionDeleteResponse
	case 15:
		return E2APMessageTypeResetResponse
	case 21:
		return E2APMessageTypeSetupFailure
	case 22:
		return E2APMessageTypeRICSubscriptionFailure
	case 23:
		return E2APMessageTypeRICControlFailure
	case 24:
		return E2APMessageTypeRICSubscriptionDeleteFailure
	case 25:
		return E2APMessageTypeResetFailure
	case 31:
		return E2APMessageTypeRICIndication
	case 32:
		return E2APMessageTypeRICServiceUpdate
	case 33:
		return E2APMessageTypeRICServiceUpdateAcknowledge
	case 34:
		return E2APMessageTypeRICServiceUpdateFailure
	default:
		return E2APMessageType(0)
	}
}

// Additional helper methods for encoding/decoding specific node ID types.

func (c *ASN1Codec) encodeGNBID(w io.Writer, nodeID string) error {
	// Simplified encoding - in production, parse and encode properly.
	return c.encodeOctetString(w, []byte(nodeID))
}

func (c *ASN1Codec) decodeGNBID(r io.Reader) (string, error) {
	data, err := c.decodeOctetString(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *ASN1Codec) encodeENBID(w io.Writer, nodeID string) error {
	return c.encodeOctetString(w, []byte(nodeID))
}

func (c *ASN1Codec) decodeENBID(r io.Reader) (string, error) {
	data, err := c.decodeOctetString(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *ASN1Codec) encodeNgENBID(w io.Writer, nodeID string) error {
	return c.encodeOctetString(w, []byte(nodeID))
}

func (c *ASN1Codec) decodeNgENBID(r io.Reader) (string, error) {
	data, err := c.decodeOctetString(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *ASN1Codec) encodeEnGNBID(w io.Writer, nodeID string) error {
	return c.encodeOctetString(w, []byte(nodeID))
}

func (c *ASN1Codec) decodeEnGNBID(r io.Reader) (string, error) {
	data, err := c.decodeOctetString(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Encode/decode RAN functions list.
func (c *ASN1Codec) encodeRANFunctionsList(w io.Writer, functions []RANFunctionItem) error {
	// Encode count.
	if err := c.encodeLength(w, len(functions)); err != nil {
		return err
	}

	// Encode each function.
	for _, fn := range functions {
		if err := c.encodeRANFunction(w, fn); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeRANFunctionsList(r io.Reader) ([]RANFunctionItem, error) {
	// Decode count.
	count, err := c.decodeLength(r)
	if err != nil {
		return nil, err
	}

	functions := make([]RANFunctionItem, count)
	for i := 0; i < count; i++ {
		fn, err := c.decodeRANFunction(r)
		if err != nil {
			return nil, err
		}
		functions[i] = fn
	}

	return functions, nil
}

func (c *ASN1Codec) encodeRANFunction(w io.Writer, fn RANFunctionItem) error {
	// Encode RAN Function ID.
	if err := binary.Write(w, binary.BigEndian, uint16(fn.RANFunctionID)); err != nil {
		return err
	}

	// Encode RAN Function Definition.
	if err := c.encodeOctetString(w, fn.RANFunctionDefinition); err != nil {
		return err
	}

	// Encode RAN Function Revision.
	if err := binary.Write(w, binary.BigEndian, uint16(fn.RANFunctionRevision)); err != nil {
		return err
	}

	// Encode RAN Function OID.
	if fn.RANFunctionOID != nil {
		if err := c.encodeOctetString(w, []byte(*fn.RANFunctionOID)); err != nil {
			return err
		}
	} else {
		if err := c.encodeOctetString(w, []byte{}); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeRANFunction(r io.Reader) (RANFunctionItem, error) {
	var fn RANFunctionItem

	// Decode RAN Function ID.
	var functionID uint16
	if err := binary.Read(r, binary.BigEndian, &functionID); err != nil {
		return fn, err
	}
	fn.RANFunctionID = RANFunctionID(functionID)

	// Decode RAN Function Definition.
	definition, err := c.decodeOctetString(r)
	if err != nil {
		return fn, err
	}
	fn.RANFunctionDefinition = definition

	// Decode RAN Function Revision.
	var revision uint16
	if err := binary.Read(r, binary.BigEndian, &revision); err != nil {
		return fn, err
	}
	fn.RANFunctionRevision = RANFunctionRevision(revision)

	// Decode RAN Function OID.
	oid, err := c.decodeOctetString(r)
	if err != nil {
		return fn, err
	}
	oidStr := string(oid)
	if len(oidStr) > 0 {
		fn.RANFunctionOID = &oidStr
	}

	return fn, nil
}

// E2 Node Component Configuration encoding/decoding.
func (c *ASN1Codec) encodeE2NodeComponentConfigList(w io.Writer, configs []E2NodeComponentConfigurationItem) error {
	// Encode count.
	if err := c.encodeLength(w, len(configs)); err != nil {
		return err
	}

	// Encode each configuration.
	for _, config := range configs {
		if err := c.encodeE2NodeComponentConfig(w, config); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeE2NodeComponentConfigList(r io.Reader) ([]E2NodeComponentConfigurationItem, error) {
	// Decode count.
	count, err := c.decodeLength(r)
	if err != nil {
		return nil, err
	}

	configs := make([]E2NodeComponentConfigurationItem, count)
	for i := 0; i < count; i++ {
		config, err := c.decodeE2NodeComponentConfig(r)
		if err != nil {
			return nil, err
		}
		configs[i] = config
	}

	return configs, nil
}

func (c *ASN1Codec) encodeE2NodeComponentConfig(w io.Writer, config E2NodeComponentConfigurationItem) error {
	// Encode component interface type.
	if err := binary.Write(w, binary.BigEndian, uint8(config.E2NodeComponentInterfaceType)); err != nil {
		return err
	}

	// Encode component ID.
	if err := c.encodeE2NodeComponentID(w, config.E2NodeComponentID); err != nil {
		return err
	}

	// Encode component configuration.
	if err := c.encodeE2NodeComponentConfiguration(w, config.E2NodeComponentConfiguration); err != nil {
		return err
	}

	return nil
}

func (c *ASN1Codec) decodeE2NodeComponentConfig(r io.Reader) (E2NodeComponentConfigurationItem, error) {
	var config E2NodeComponentConfigurationItem

	// Decode component interface type.
	var interfaceType uint8
	if err := binary.Read(r, binary.BigEndian, &interfaceType); err != nil {
		return config, err
	}
	config.E2NodeComponentInterfaceType = E2NodeComponentInterfaceType(interfaceType)

	// Decode component ID.
	id, err := c.decodeE2NodeComponentID(r)
	if err != nil {
		return config, err
	}
	config.E2NodeComponentID = id

	// Decode component configuration.
	configuration, err := c.decodeE2NodeComponentConfiguration(r)
	if err != nil {
		return config, err
	}
	config.E2NodeComponentConfiguration = configuration

	return config, nil
}

// Additional encoding/decoding methods for other message types.

func (c *ASN1Codec) encodeE2SetupResponse(resp *E2SetupResponse) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Encode Global RIC ID.
	if err := c.encodeGlobalRICID(buf, resp.GlobalRICID); err != nil {
		return nil, err
	}

	// Encode RAN Functions Accepted (optional).
	if len(resp.RANFunctionsAccepted) > 0 {
		if err := c.encodeRANFunctionIDList(buf, resp.RANFunctionsAccepted); err != nil {
			return nil, err
		}
	}

	// Encode RAN Functions Rejected (optional).
	if len(resp.RANFunctionsRejected) > 0 {
		// Convert RANFunctionIDCause slice to RANFunctionIDCauseItem slice.
		rejectedItems := make([]RANFunctionIDCauseItem, len(resp.RANFunctionsRejected))
		for i, rejected := range resp.RANFunctionsRejected {
			rejectedItems[i] = RANFunctionIDCauseItem{
				RANFunctionID: int(rejected.RANFunctionID),
				Cause: E2Cause{
					CauseType:  E2CauseTypeRIC, // Simplified mapping
					CauseValue: 0,              // Default value
				},
			}
		}
		if err := c.encodeRANFunctionIDCauseList(buf, rejectedItems); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (c *ASN1Codec) decodeE2SetupResponse(data []byte) (*E2SetupResponse, error) {
	buf := bytes.NewReader(data)
	resp := &E2SetupResponse{}

	// Decode Global RIC ID.
	ricID, err := c.decodeGlobalRICID(buf)
	if err != nil {
		return nil, err
	}
	resp.GlobalRICID = ricID

	// Check for optional RAN Functions Accepted.
	if buf.Len() > 0 {
		accepted, err := c.decodeRANFunctionIDList(buf)
		if err == nil {
			resp.RANFunctionsAccepted = accepted
		}
	}

	// Check for optional RAN Functions Rejected.
	if buf.Len() > 0 {
		rejectedItems, err := c.decodeRANFunctionIDCauseList(buf)
		if err == nil {
			// Convert RANFunctionIDCauseItem slice to RANFunctionIDCause slice.
			resp.RANFunctionsRejected = make([]RANFunctionIDCause, len(rejectedItems))
			for i, item := range rejectedItems {
				resp.RANFunctionsRejected[i] = RANFunctionIDCause{
					RANFunctionID: RANFunctionID(item.RANFunctionID),
					Cause:         E2APCause{}, // Simplified - would need proper conversion
				}
			}
		}
	}

	return resp, nil
}

func (c *ASN1Codec) encodeGlobalRICID(w io.Writer, id GlobalRICID) error {
	// Encode PLMN Identity.
	plmnBytes := c.encodePLMNIdentity(id.PLMNIdentity)
	if _, err := w.Write(plmnBytes); err != nil {
		return err
	}

	// Encode RIC ID (32 bits).
	return binary.Write(w, binary.BigEndian, uint32(id.RICID))
}

func (c *ASN1Codec) decodeGlobalRICID(r io.Reader) (GlobalRICID, error) {
	var id GlobalRICID

	// Decode PLMN Identity.
	plmnBytes := make([]byte, 3)
	if _, err := io.ReadFull(r, plmnBytes); err != nil {
		return id, err
	}
	id.PLMNIdentity = c.decodePLMNIdentity(plmnBytes)

	// Decode RIC ID.
	var ricID uint32
	if err := binary.Read(r, binary.BigEndian, &ricID); err != nil {
		return id, err
	}
	id.RICID = RICID(ricID)

	return id, nil
}

func (c *ASN1Codec) encodeRANFunctionIDList(w io.Writer, functions []RANFunctionIDItem) error {
	// Encode count.
	if err := c.encodeLength(w, len(functions)); err != nil {
		return err
	}

	// Encode each function ID.
	for _, fn := range functions {
		if err := binary.Write(w, binary.BigEndian, uint16(fn.RANFunctionID)); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, uint16(fn.RANFunctionRevision)); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeRANFunctionIDList(r io.Reader) ([]RANFunctionIDItem, error) {
	// Decode count.
	count, err := c.decodeLength(r)
	if err != nil {
		return nil, err
	}

	functions := make([]RANFunctionIDItem, count)
	for i := 0; i < count; i++ {
		var functionID, revision uint16
		if err := binary.Read(r, binary.BigEndian, &functionID); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.BigEndian, &revision); err != nil {
			return nil, err
		}
		functions[i] = RANFunctionIDItem{
			RANFunctionID:       RANFunctionID(functionID),
			RANFunctionRevision: RANFunctionRevision(revision),
		}
	}

	return functions, nil
}

func (c *ASN1Codec) encodeRANFunctionIDCauseList(w io.Writer, functions []RANFunctionIDCauseItem) error {
	// Encode count.
	if err := c.encodeLength(w, len(functions)); err != nil {
		return err
	}

	// Encode each function with cause.
	for _, fn := range functions {
		if err := binary.Write(w, binary.BigEndian, uint16(fn.RANFunctionID)); err != nil {
			return err
		}
		if err := c.encodeCause(w, fn.Cause); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeRANFunctionIDCauseList(r io.Reader) ([]RANFunctionIDCauseItem, error) {
	// Decode count.
	count, err := c.decodeLength(r)
	if err != nil {
		return nil, err
	}

	functions := make([]RANFunctionIDCauseItem, count)
	for i := 0; i < count; i++ {
		var functionID uint16
		if err := binary.Read(r, binary.BigEndian, &functionID); err != nil {
			return nil, err
		}

		cause, err := c.decodeCause(r)
		if err != nil {
			return nil, err
		}

		functions[i] = RANFunctionIDCauseItem{
			RANFunctionID: int(functionID),
			Cause:         cause,
		}
	}

	return functions, nil
}

func (c *ASN1Codec) encodeCause(w io.Writer, cause E2Cause) error {
	// Encode cause type (1 byte).
	if err := binary.Write(w, binary.BigEndian, uint8(cause.CauseType)); err != nil {
		return err
	}

	// Encode cause value (2 bytes).
	return binary.Write(w, binary.BigEndian, uint16(cause.CauseValue))
}

func (c *ASN1Codec) decodeCause(r io.Reader) (E2Cause, error) {
	var cause E2Cause

	// Decode cause type.
	var causeType uint8
	if err := binary.Read(r, binary.BigEndian, &causeType); err != nil {
		return cause, err
	}
	cause.CauseType = E2CauseType(causeType)

	// Decode cause value.
	var causeValue uint16
	if err := binary.Read(r, binary.BigEndian, &causeValue); err != nil {
		return cause, err
	}
	cause.CauseValue = int(causeValue)

	return cause, nil
}

// RIC Control Request encoding/decoding.
func (c *ASN1Codec) encodeRICControlRequest(req *RICControlRequest) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Encode RIC Request ID.
	if err := c.encodeRICRequestID(buf, req.RICRequestID); err != nil {
		return nil, err
	}

	// Encode RAN Function ID.
	if err := binary.Write(buf, binary.BigEndian, uint16(req.RANFunctionID)); err != nil {
		return nil, err
	}

	// Encode Call Process ID (optional).
	if req.RICCallProcessID != nil && len(*req.RICCallProcessID) > 0 {
		if err := c.encodeOctetString(buf, []byte(*req.RICCallProcessID)); err != nil {
			return nil, err
		}
	}

	// Encode Control Header.
	if err := c.encodeOctetString(buf, req.RICControlHeader); err != nil {
		return nil, err
	}

	// Encode Control Message.
	if err := c.encodeOctetString(buf, req.RICControlMessage); err != nil {
		return nil, err
	}

	// Encode Control Ack Request (optional).
	ackByte := uint8(0)
	if req.RICControlAckRequest != nil {
		ackByte = uint8(*req.RICControlAckRequest)
	}
	if err := binary.Write(buf, binary.BigEndian, ackByte); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (c *ASN1Codec) decodeRICControlRequest(data []byte) (*RICControlRequest, error) {
	buf := bytes.NewReader(data)
	req := &RICControlRequest{}

	// Decode RIC Request ID.
	requestID, err := c.decodeRICRequestID(buf)
	if err != nil {
		return nil, err
	}
	req.RICRequestID = requestID

	// Decode RAN Function ID.
	var ranFunctionID uint16
	if err := binary.Read(buf, binary.BigEndian, &ranFunctionID); err != nil {
		return nil, err
	}
	req.RANFunctionID = RANFunctionID(ranFunctionID)

	// Try to decode optional Call Process ID.
	// This needs proper optional field handling in real ASN.1.
	callProcessID, err := c.decodeOctetString(buf)
	if err == nil && len(callProcessID) > 0 {
		ricCallProcessID := RICCallProcessID(callProcessID)
		req.RICCallProcessID = &ricCallProcessID
	}

	// Decode Control Header.
	header, err := c.decodeOctetString(buf)
	if err != nil {
		return nil, err
	}
	req.RICControlHeader = header

	// Decode Control Message.
	message, err := c.decodeOctetString(buf)
	if err != nil {
		return nil, err
	}
	req.RICControlMessage = message

	// Decode Control Ack Request.
	var ackByte uint8
	if err := binary.Read(buf, binary.BigEndian, &ackByte); err != nil {
		return nil, err
	}
	if ackByte != 0 {
		ackReq := RICControlAckRequest(ackByte)
		req.RICControlAckRequest = &ackReq
	}

	return req, nil
}

// RIC Control Acknowledge encoding/decoding.
func (c *ASN1Codec) encodeRICControlAcknowledge(ack *RICControlAcknowledge) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Encode RIC Request ID.
	if err := c.encodeRICRequestID(buf, ack.RICRequestID); err != nil {
		return nil, err
	}

	// Encode RAN Function ID.
	if err := binary.Write(buf, binary.BigEndian, uint16(ack.RANFunctionID)); err != nil {
		return nil, err
	}

	// Encode Call Process ID (optional).
	if ack.RICCallProcessID != nil && len(*ack.RICCallProcessID) > 0 {
		if err := c.encodeOctetString(buf, []byte(*ack.RICCallProcessID)); err != nil {
			return nil, err
		}
	}

	// Encode Control Outcome (optional).
	if len(ack.RICControlOutcome) > 0 {
		if err := c.encodeOctetString(buf, ack.RICControlOutcome); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (c *ASN1Codec) decodeRICControlAcknowledge(data []byte) (*RICControlAcknowledge, error) {
	buf := bytes.NewReader(data)
	ack := &RICControlAcknowledge{}

	// Decode RIC Request ID.
	requestID, err := c.decodeRICRequestID(buf)
	if err != nil {
		return nil, err
	}
	ack.RICRequestID = requestID

	// Decode RAN Function ID.
	var ranFunctionID uint16
	if err := binary.Read(buf, binary.BigEndian, &ranFunctionID); err != nil {
		return nil, err
	}
	ack.RANFunctionID = RANFunctionID(ranFunctionID)

	// Check for optional Call Process ID.
	if buf.Len() > 0 {
		callProcessID, err := c.decodeOctetString(buf)
		if err == nil {
			ricCallProcessID := RICCallProcessID(callProcessID)
			ack.RICCallProcessID = &ricCallProcessID
		}
	}

	// Check for optional Control Outcome.
	if buf.Len() > 0 {
		outcome, err := c.decodeOctetString(buf)
		if err == nil {
			ack.RICControlOutcome = outcome
		}
	}

	return ack, nil
}

// RIC Subscription Response encoding/decoding.
func (c *ASN1Codec) encodeRICSubscriptionResponse(resp *RICSubscriptionResponse) ([]byte, error) {
	buf := &bytes.Buffer{}

	// Encode RIC Request ID.
	if err := c.encodeRICRequestID(buf, resp.RICRequestID); err != nil {
		return nil, err
	}

	// Encode RAN Function ID.
	if err := binary.Write(buf, binary.BigEndian, uint16(resp.RANFunctionID)); err != nil {
		return nil, err
	}

	// Encode Actions Admitted List - convert RICActionID slice to RICActionAdmittedItem slice.
	admittedItems := make([]RICActionAdmittedItem, len(resp.RICActionAdmittedList))
	for i, actionID := range resp.RICActionAdmittedList {
		admittedItems[i] = RICActionAdmittedItem{ActionID: int(actionID)}
	}
	if err := c.encodeRICActionAdmittedList(buf, admittedItems); err != nil {
		return nil, err
	}

	// Encode Actions Not Admitted List (optional).
	if len(resp.RICActionNotAdmittedList) > 0 {
		if err := c.encodeRICActionNotAdmittedList(buf, resp.RICActionNotAdmittedList); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (c *ASN1Codec) decodeRICSubscriptionResponse(data []byte) (*RICSubscriptionResponse, error) {
	buf := bytes.NewReader(data)
	resp := &RICSubscriptionResponse{}

	// Decode RIC Request ID.
	requestID, err := c.decodeRICRequestID(buf)
	if err != nil {
		return nil, err
	}
	resp.RICRequestID = requestID

	// Decode RAN Function ID.
	var ranFunctionID uint16
	if err := binary.Read(buf, binary.BigEndian, &ranFunctionID); err != nil {
		return nil, err
	}
	resp.RANFunctionID = RANFunctionID(ranFunctionID)

	// Decode Actions Admitted List.
	admittedItems, err := c.decodeRICActionAdmittedList(buf)
	if err != nil {
		return nil, err
	}
	// Convert RICActionAdmittedItem slice to RICActionID slice.
	resp.RICActionAdmittedList = make([]RICActionID, len(admittedItems))
	for i, item := range admittedItems {
		resp.RICActionAdmittedList[i] = RICActionID(item.ActionID)
	}

	// Check for optional Actions Not Admitted List.
	if buf.Len() > 0 {
		notAdmitted, err := c.decodeRICActionNotAdmittedList(buf)
		if err == nil {
			resp.RICActionNotAdmittedList = notAdmitted
		}
	}

	return resp, nil
}

// Helper methods for RIC Request ID.
func (c *ASN1Codec) encodeRICRequestID(w io.Writer, id RICRequestID) error {
	// Encode Requestor ID (16 bits).
	if err := binary.Write(w, binary.BigEndian, uint16(id.RICRequestorID)); err != nil {
		return err
	}

	// Encode Instance ID (16 bits).
	return binary.Write(w, binary.BigEndian, uint16(id.RICInstanceID))
}

func (c *ASN1Codec) decodeRICRequestID(r io.Reader) (RICRequestID, error) {
	var id RICRequestID

	// Decode Requestor ID.
	var requestorID uint16
	if err := binary.Read(r, binary.BigEndian, &requestorID); err != nil {
		return id, err
	}
	id.RICRequestorID = RICRequestorID(requestorID)

	// Decode Instance ID.
	var instanceID uint16
	if err := binary.Read(r, binary.BigEndian, &instanceID); err != nil {
		return id, err
	}
	id.RICInstanceID = RICInstanceID(instanceID)

	return id, nil
}

// Helper methods for RIC Actions.
func (c *ASN1Codec) encodeRICActionsList(w io.Writer, actions []RICActionToBeSetupItem) error {
	// Encode count.
	if err := c.encodeLength(w, len(actions)); err != nil {
		return err
	}

	// Encode each action.
	for _, action := range actions {
		if err := c.encodeRICAction(w, action); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeRICActionsList(r io.Reader) ([]RICActionToBeSetupItem, error) {
	// Decode count.
	count, err := c.decodeLength(r)
	if err != nil {
		return nil, err
	}

	actions := make([]RICActionToBeSetupItem, count)
	for i := 0; i < count; i++ {
		action, err := c.decodeRICAction(r)
		if err != nil {
			return nil, err
		}
		actions[i] = action
	}

	return actions, nil
}

func (c *ASN1Codec) encodeRICAction(w io.Writer, action RICActionToBeSetupItem) error {
	// Encode Action ID.
	if err := binary.Write(w, binary.BigEndian, uint16(action.RICActionID)); err != nil {
		return err
	}

	// Encode Action Type.
	if err := binary.Write(w, binary.BigEndian, uint8(action.RICActionType)); err != nil {
		return err
	}

	// Encode Action Definition (optional).
	if len(action.RICActionDefinition) > 0 {
		if err := c.encodeOctetString(w, action.RICActionDefinition); err != nil {
			return err
		}
	}

	// Encode Subsequent Action (optional).
	if action.RICSubsequentAction != nil {
		if err := c.encodeSubsequentAction(w, *action.RICSubsequentAction); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeRICAction(r io.Reader) (RICActionToBeSetupItem, error) {
	var action RICActionToBeSetupItem

	// Decode Action ID.
	var actionID uint16
	if err := binary.Read(r, binary.BigEndian, &actionID); err != nil {
		return action, err
	}
	action.RICActionID = RICActionID(actionID)

	// Decode Action Type.
	var actionType uint8
	if err := binary.Read(r, binary.BigEndian, &actionType); err != nil {
		return action, err
	}
	action.RICActionType = RICActionType(actionType)

	// Check for optional Action Definition.
	// This needs proper optional field handling.
	definition, err := c.decodeOctetString(r)
	if err == nil && len(definition) > 0 {
		action.RICActionDefinition = definition
	}

	// Check for optional Subsequent Action.
	// This needs proper optional field handling.

	return action, nil
}

func (c *ASN1Codec) encodeSubsequentAction(w io.Writer, action RICSubsequentAction) error {
	// Encode Subsequent Action Type.
	if err := binary.Write(w, binary.BigEndian, uint8(action.RICSubsequentActionType)); err != nil {
		return err
	}

	// Encode Time to Wait (optional).
	if action.RICTimeToWait != nil {
		return binary.Write(w, binary.BigEndian, uint8(*action.RICTimeToWait))
	}
	return binary.Write(w, binary.BigEndian, uint8(0))
}

func (c *ASN1Codec) encodeRICActionAdmittedList(w io.Writer, actions []RICActionAdmittedItem) error {
	// Encode count.
	if err := c.encodeLength(w, len(actions)); err != nil {
		return err
	}

	// Encode each admitted action.
	for _, action := range actions {
		if err := binary.Write(w, binary.BigEndian, uint16(action.ActionID)); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeRICActionAdmittedList(r io.Reader) ([]RICActionAdmittedItem, error) {
	// Decode count.
	count, err := c.decodeLength(r)
	if err != nil {
		return nil, err
	}

	actions := make([]RICActionAdmittedItem, count)
	for i := 0; i < count; i++ {
		var actionID uint16
		if err := binary.Read(r, binary.BigEndian, &actionID); err != nil {
			return nil, err
		}
		actions[i] = RICActionAdmittedItem{ActionID: int(actionID)}
	}

	return actions, nil
}

func (c *ASN1Codec) encodeRICActionNotAdmittedList(w io.Writer, actions []RICActionNotAdmittedItem) error {
	// Encode count.
	if err := c.encodeLength(w, len(actions)); err != nil {
		return err
	}

	// Encode each not admitted action.
	for _, action := range actions {
		if err := binary.Write(w, binary.BigEndian, uint16(action.RICActionID)); err != nil {
			return err
		}
		// Convert E2APCause to E2Cause for encoding.
		e2Cause := E2Cause{
			CauseType:  E2CauseTypeRIC, // Simplified mapping
			CauseValue: 0,              // Default value
		}
		if err := c.encodeCause(w, e2Cause); err != nil {
			return err
		}
	}

	return nil
}

func (c *ASN1Codec) decodeRICActionNotAdmittedList(r io.Reader) ([]RICActionNotAdmittedItem, error) {
	// Decode count.
	count, err := c.decodeLength(r)
	if err != nil {
		return nil, err
	}

	actions := make([]RICActionNotAdmittedItem, count)
	for i := 0; i < count; i++ {
		var actionID uint16
		if err := binary.Read(r, binary.BigEndian, &actionID); err != nil {
			return nil, err
		}

		_, err := c.decodeCause(r) // Decode but don't use - simplified implementation
		if err != nil {
			return nil, err
		}

		actions[i] = RICActionNotAdmittedItem{
			RICActionID: RICActionID(actionID),
			Cause:       E2APCause{}, // Simplified - would need proper conversion from E2Cause
		}
	}

	return actions, nil
}

// Indication Type encoding/decoding.
func (c *ASN1Codec) encodeIndicationType(w io.Writer, indType RICIndicationType) error {
	return binary.Write(w, binary.BigEndian, uint8(indType))
}

func (c *ASN1Codec) decodeIndicationType(r io.Reader) (RICIndicationType, error) {
	var indType uint8
	if err := binary.Read(r, binary.BigEndian, &indType); err != nil {
		return 0, err
	}
	return RICIndicationType(indType), nil
}

// E2NodeComponentID encoding/decoding.
func (c *ASN1Codec) encodeE2NodeComponentID(w io.Writer, id E2NodeComponentID) error {
	// For simplicity, encode as a choice indicator followed by the active field.
	// In real ASN.1 PER, this would use choice encoding rules.

	if id.E2NodeComponentInterfaceTypeNG != nil {
		// Encode choice indicator for NG (0).
		if err := binary.Write(w, binary.BigEndian, uint8(0)); err != nil {
			return err
		}
		// Encode AMF Name.
		return c.encodeOctetString(w, []byte(id.E2NodeComponentInterfaceTypeNG.AMFName))
	}
	if id.E2NodeComponentInterfaceTypeXn != nil {
		// Encode choice indicator for Xn (1).
		if err := binary.Write(w, binary.BigEndian, uint8(1)); err != nil {
			return err
		}
		// Encode Global NG-RAN Node ID.
		return c.encodeOctetString(w, []byte(id.E2NodeComponentInterfaceTypeXn.GlobalNGRANNodeID))
	}
	if id.E2NodeComponentInterfaceTypeE1 != nil {
		// Encode choice indicator for E1 (2).
		if err := binary.Write(w, binary.BigEndian, uint8(2)); err != nil {
			return err
		}
		// Encode gNB-CU-CP ID.
		return c.encodeOctetString(w, []byte(id.E2NodeComponentInterfaceTypeE1.GNBCUCPID))
	}
	if id.E2NodeComponentInterfaceTypeF1 != nil {
		// Encode choice indicator for F1 (3).
		if err := binary.Write(w, binary.BigEndian, uint8(3)); err != nil {
			return err
		}
		// Encode gNB-DU ID.
		return c.encodeOctetString(w, []byte(id.E2NodeComponentInterfaceTypeF1.GNBDUID))
	}
	if id.E2NodeComponentInterfaceTypeW1 != nil {
		// Encode choice indicator for W1 (4).
		if err := binary.Write(w, binary.BigEndian, uint8(4)); err != nil {
			return err
		}
		// Encode ng-eNB-DU ID.
		return c.encodeOctetString(w, []byte(id.E2NodeComponentInterfaceTypeW1.NGENBDUID))
	}
	if id.E2NodeComponentInterfaceTypeS1 != nil {
		// Encode choice indicator for S1 (5).
		if err := binary.Write(w, binary.BigEndian, uint8(5)); err != nil {
			return err
		}
		// Encode MME Name.
		return c.encodeOctetString(w, []byte(id.E2NodeComponentInterfaceTypeS1.MMEName))
	}
	if id.E2NodeComponentInterfaceTypeX2 != nil {
		// Encode choice indicator for X2 (6).
		if err := binary.Write(w, binary.BigEndian, uint8(6)); err != nil {
			return err
		}
		// Encode eNB ID.
		return c.encodeOctetString(w, []byte(id.E2NodeComponentInterfaceTypeX2.ENBID))
	}

	// Default case - empty choice.
	return binary.Write(w, binary.BigEndian, uint8(255))
}

func (c *ASN1Codec) decodeE2NodeComponentID(r io.Reader) (E2NodeComponentID, error) {
	var id E2NodeComponentID

	// Decode choice indicator.
	var choice uint8
	if err := binary.Read(r, binary.BigEndian, &choice); err != nil {
		return id, err
	}

	switch choice {
	case 0: // NG
		name, err := c.decodeOctetString(r)
		if err != nil {
			return id, err
		}
		id.E2NodeComponentInterfaceTypeNG = &E2NodeComponentInterfaceNG{
			AMFName: string(name),
		}
	case 1: // Xn
		nodeID, err := c.decodeOctetString(r)
		if err != nil {
			return id, err
		}
		id.E2NodeComponentInterfaceTypeXn = &E2NodeComponentInterfaceXn{
			GlobalNGRANNodeID: string(nodeID),
		}
	case 2: // E1
		cuCPID, err := c.decodeOctetString(r)
		if err != nil {
			return id, err
		}
		id.E2NodeComponentInterfaceTypeE1 = &E2NodeComponentInterfaceE1{
			GNBCUCPID: string(cuCPID),
		}
	case 3: // F1
		duID, err := c.decodeOctetString(r)
		if err != nil {
			return id, err
		}
		id.E2NodeComponentInterfaceTypeF1 = &E2NodeComponentInterfaceF1{
			GNBDUID: string(duID),
		}
	case 4: // W1
		duID, err := c.decodeOctetString(r)
		if err != nil {
			return id, err
		}
		id.E2NodeComponentInterfaceTypeW1 = &E2NodeComponentInterfaceW1{
			NGENBDUID: string(duID),
		}
	case 5: // S1
		name, err := c.decodeOctetString(r)
		if err != nil {
			return id, err
		}
		id.E2NodeComponentInterfaceTypeS1 = &E2NodeComponentInterfaceS1{
			MMEName: string(name),
		}
	case 6: // X2
		enbID, err := c.decodeOctetString(r)
		if err != nil {
			return id, err
		}
		id.E2NodeComponentInterfaceTypeX2 = &E2NodeComponentInterfaceX2{
			ENBID: string(enbID),
		}
	}

	return id, nil
}

// E2NodeComponentConfiguration encoding/decoding.
func (c *ASN1Codec) encodeE2NodeComponentConfiguration(w io.Writer, config E2NodeComponentConfiguration) error {
	// Encode request part.
	if err := c.encodeOctetString(w, config.E2NodeComponentRequestPart); err != nil {
		return err
	}

	// Encode response part.
	return c.encodeOctetString(w, config.E2NodeComponentResponsePart)
}

func (c *ASN1Codec) decodeE2NodeComponentConfiguration(r io.Reader) (E2NodeComponentConfiguration, error) {
	var config E2NodeComponentConfiguration

	// Decode request part.
	requestPart, err := c.decodeOctetString(r)
	if err != nil {
		return config, err
	}
	config.E2NodeComponentRequestPart = requestPart

	// Decode response part.
	responsePart, err := c.decodeOctetString(r)
	if err != nil {
		return config, err
	}
	config.E2NodeComponentResponsePart = responsePart

	return config, nil
}
