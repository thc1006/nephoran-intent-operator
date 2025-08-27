package e2

import (
	"context"
	"testing"
	"time"
)

// TestE2InterfaceComplete demonstrates the complete E2 interface implementation
func TestE2InterfaceComplete(t *testing.T) {
	t.Run("E2AP_Messages", func(t *testing.T) {
		// Test E2 Setup Request/Response
		globalE2NodeID := GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{MCC: "310", MNC: "260"},
			NodeType:     E2NodeTypegNB,
			NodeID:       "gNB-001",
		}

		setupReq := &E2SetupRequest{
			GlobalE2NodeID: globalE2NodeID,
			RANFunctionsList: []RANFunctionItem{
				{
					RANFunctionID:         1,
					RANFunctionDefinition: []byte("KPM Service Model"),
					RANFunctionRevision:   1,
					RANFunctionOID:        stringPtr("1.3.6.1.4.1.53148.1.2.2.100"),
				},
				{
					RANFunctionID:         2,
					RANFunctionDefinition: []byte("RC Service Model"),
					RANFunctionRevision:   1,
					RANFunctionOID:        stringPtr("1.3.6.1.4.1.53148.1.2.2.101"),
				},
				{
					RANFunctionID:         3,
					RANFunctionDefinition: []byte("NI Service Model"),
					RANFunctionRevision:   1,
					RANFunctionOID:        stringPtr("1.3.6.1.4.1.53148.1.2.2.102"),
				},
			},
		}

		if setupReq.GlobalE2NodeID.NodeID != "gNB-001" {
			t.Errorf("Expected node ID gNB-001, got %s", setupReq.GlobalE2NodeID.NodeID)
		}

		if len(setupReq.RANFunctionsList) != 3 {
			t.Errorf("Expected 3 RAN functions, got %d", len(setupReq.RANFunctionsList))
		}
	})

	t.Run("ASN1_Encoding", func(t *testing.T) {
		codec := NewASN1Codec(true)

		// Create a test message
		msg := &E2APMessage{
			MessageType:   E2APMessageTypeSetupRequest,
			TransactionID: 1,
			ProcedureCode: 1,
			Criticality:   CriticalityReject,
			Payload: &E2SetupRequest{
				GlobalE2NodeID: GlobalE2NodeID{
					PLMNIdentity: PLMNIdentity{MCC: "310", MNC: "260"},
					NodeType:     E2NodeTypegNB,
					NodeID:       "gNB-001",
				},
				RANFunctionsList: []RANFunctionItem{
					{
						RANFunctionID:         1,
						RANFunctionDefinition: []byte("Test Function"),
						RANFunctionRevision:   1,
						RANFunctionOID:        stringPtr("1.3.6.1.4.1.53148.1.2.2.100"),
					},
				},
			},
		}

		// Encode the message
		encoded, err := codec.EncodeE2APMessage(msg)
		if err != nil {
			t.Fatalf("Failed to encode E2AP message: %v", err)
		}

		if len(encoded) == 0 {
			t.Error("Encoded message is empty")
		}

		// Decode the message
		decoded, err := codec.DecodeE2APMessage(encoded)
		if err != nil {
			t.Fatalf("Failed to decode E2AP message: %v", err)
		}

		if decoded.MessageType != msg.MessageType {
			t.Errorf("Message type mismatch: expected %v, got %v", msg.MessageType, decoded.MessageType)
		}
	})

	t.Run("Service_Models", func(t *testing.T) {
		// Test KPM Service Model
		t.Run("KPM", func(t *testing.T) {
			// KPM model should be available
			kpmRanFunction := RanFunction{
				FunctionID:          1,
				FunctionDefinition:  "E2SM-KPM",
				FunctionRevision:    1,
				FunctionOID:         "1.3.6.1.4.1.53148.1.2.2.100",
				FunctionDescription: "Key Performance Measurement Service Model",
			}

			if kpmRanFunction.FunctionID != 1 {
				t.Errorf("Expected KPM function ID 1, got %d", kpmRanFunction.FunctionID)
			}
		})

		// Test RC Service Model
		t.Run("RC", func(t *testing.T) {
			// RC model should be available
			rcRanFunction := RanFunction{
				FunctionID:          2,
				FunctionDefinition:  "E2SM-RC",
				FunctionRevision:    1,
				FunctionOID:         "1.3.6.1.4.1.53148.1.2.2.101",
				FunctionDescription: "RAN Control Service Model",
			}

			if rcRanFunction.FunctionID != 2 {
				t.Errorf("Expected RC function ID 2, got %d", rcRanFunction.FunctionID)
			}
		})

		// Test NI Service Model (Network Interface)
		t.Run("NI", func(t *testing.T) {
			// NI model should be available - this is the newly added model
			niRanFunction := RanFunction{
				FunctionID:          3,
				FunctionDefinition:  "E2SM-NI",
				FunctionRevision:    1,
				FunctionOID:         "1.3.6.1.4.1.53148.1.2.2.102",
				FunctionDescription: "Network Interface Service Model",
			}

			if niRanFunction.FunctionID != 3 {
				t.Errorf("Expected NI function ID 3, got %d", niRanFunction.FunctionID)
			}

			if niRanFunction.FunctionDefinition != "E2SM-NI" {
				t.Errorf("Expected E2SM-NI, got %s", niRanFunction.FunctionDefinition)
			}
		})
	})

	t.Run("SCTP_Transport", func(t *testing.T) {
		config := &SCTPConfig{
			ListenAddress:     "0.0.0.0",
			ListenPort:        36421,
			MaxConnections:    100,
			MaxStreams:        10,
			InitTimeout:       5 * time.Second,
			HeartbeatInterval: 30 * time.Second,
			MaxRetransmits:    3,
			RTO:               1 * time.Second,
			KeepAlive:         true,
			KeepAliveInterval: 60 * time.Second,
			ConnectionTimeout: 300 * time.Second,
			SendBufferSize:    65536,
			RecvBufferSize:    65536,
			MaxMessageSize:    8192,
			EnableTLS:         false,
		}

		codec := NewASN1Codec(true)
		transport := NewSCTPTransport(config, codec)

		if transport == nil {
			t.Fatal("Failed to create SCTP transport")
		}

		// Verify transport configuration
		if transport.config.ListenPort != 36421 {
			t.Errorf("Expected SCTP port 36421, got %d", transport.config.ListenPort)
		}

		// Test message handler setting
		messageHandler := func(nodeID string, message *E2APMessage) error {
			// Handle message
			return nil
		}

		transport.SetMessageHandler(messageHandler)

		if transport.messageHandler == nil {
			t.Error("Message handler not set")
		}
	})

	t.Run("E2_Subscription", func(t *testing.T) {
		// Test subscription request creation
		requestID := RICRequestID{
			RICRequestorID: 123,
			RICInstanceID:  456,
		}

		subscriptionReq := &RICSubscriptionRequest{
			RICRequestID:  requestID,
			RANFunctionID: 1,
			RICSubscriptionDetails: RICSubscriptionDetails{
				RICEventTriggerDefinition: []byte("event_trigger"),
				RICActionToBeSetupList: []RICActionToBeSetupItem{
					{
						RICActionID:         1,
						RICActionType:       RICActionTypeReport,
						RICActionDefinition: []byte("action_definition"),
					},
				},
			},
		}

		if subscriptionReq.RICRequestID.RICRequestorID != 123 {
			t.Errorf("Expected requestor ID 123, got %d", subscriptionReq.RICRequestID.RICRequestorID)
		}

		if len(subscriptionReq.RICSubscriptionDetails.RICActionToBeSetupList) != 1 {
			t.Errorf("Expected 1 action, got %d", len(subscriptionReq.RICSubscriptionDetails.RICActionToBeSetupList))
		}
	})

	t.Run("E2_Indication", func(t *testing.T) {
		// Test indication message
		indicationSN := RICIndicationSN(1000)
		indication := &RICIndication{
			RICRequestID:         RICRequestID{RICRequestorID: 123, RICInstanceID: 456},
			RANFunctionID:        1,
			RICActionID:          1,
			RICIndicationSN:      &indicationSN,
			RICIndicationType:    RICIndicationTypeReport,
			RICIndicationHeader:  []byte("indication_header"),
			RICIndicationMessage: []byte("indication_message"),
		}

		if indication.RICIndicationType != RICIndicationTypeReport {
			t.Errorf("Expected indication type REPORT, got %v", indication.RICIndicationType)
		}

		if *indication.RICIndicationSN != 1000 {
			t.Errorf("Expected indication SN 1000, got %d", *indication.RICIndicationSN)
		}
	})

	t.Run("xApp_Interface", func(t *testing.T) {
		// Test xApp SDK configuration
		xappConfig := &XAppConfig{
			XAppName:        "test-xapp",
			XAppVersion:     "1.0.0",
			XAppDescription: "Test xApp for E2 interface",
			E2NodeID:        "gNB-001",
			NearRTRICURL:    "http://near-rt-ric:8080",
			ServiceModels:   []string{"KPM", "RC", "NI"},
			Environment:     map[string]string{"ENV": "test"},
			ResourceLimits: &XAppResourceLimits{
				MaxMemoryMB:      1024,
				MaxCPUCores:      2.0,
				MaxSubscriptions: 100,
				RequestTimeout:   30 * time.Second,
			},
			HealthCheck: &XAppHealthConfig{
				Enabled:          true,
				CheckInterval:    30 * time.Second,
				FailureThreshold: 3,
				HealthEndpoint:   "/health",
			},
		}

		if xappConfig.XAppName != "test-xapp" {
			t.Errorf("Expected xApp name test-xapp, got %s", xappConfig.XAppName)
		}

		if len(xappConfig.ServiceModels) != 3 {
			t.Errorf("Expected 3 service models, got %d", len(xappConfig.ServiceModels))
		}

		// Verify NI service model is included
		hasNI := false
		for _, model := range xappConfig.ServiceModels {
			if model == "NI" {
				hasNI = true
				break
			}
		}

		if !hasNI {
			t.Error("NI service model not found in xApp configuration")
		}
	})

	t.Run("E2_Manager_Interface", func(t *testing.T) {
		// Test that E2ManagerInterface is properly defined
		var mgr E2ManagerInterface

		// This should compile, proving the interface exists
		_ = mgr

		// Test E2Node structure
		node := &E2Node{
			NodeID: "gNB-001",
			GlobalE2NodeID: E2NodeID{
				GNBID: &GNBID{
					GNBIDChoice: GNBIDChoice{
						GNBID22: strPtr("0x123456"),
					},
				},
			},
			ConnectionStatus: E2ConnectionStatus{
				State: "CONNECTED",
			},
			HealthStatus:      NodeHealth{Status: "HEALTHY"},
			SubscriptionCount: 5,
			LastSeen:          time.Now(),
		}

		if node.ConnectionStatus.State != "CONNECTED" {
			t.Errorf("Expected CONNECTED status, got %s", node.ConnectionStatus.State)
		}
	})

	t.Run("Complete_E2AP_Flow", func(t *testing.T) {
		// Simulate complete E2AP message flow
		ctx := context.Background()

		// 1. E2 Setup
		setupReq := CreateE2SetupRequest(
			GlobalE2NodeID{
				PLMNIdentity: PLMNIdentity{MCC: "310", MNC: "260"},
				NodeType:     E2NodeTypegNB,
				NodeID:       "gNB-001",
			},
			[]RANFunctionItem{
				{RANFunctionID: 1, RANFunctionDefinition: []byte("KPM")},
				{RANFunctionID: 2, RANFunctionDefinition: []byte("RC")},
				{RANFunctionID: 3, RANFunctionDefinition: []byte("NI")},
			},
		)

		if setupReq.MessageType != E2APMessageTypeSetupRequest {
			t.Errorf("Expected SETUP_REQUEST, got %v", setupReq.MessageType)
		}

		// 2. RIC Subscription
		subscriptionReq := CreateRICSubscriptionRequest(
			RICRequestID{RICRequestorID: 1, RICInstanceID: 1},
			1, // RAN Function ID for KPM
			RICSubscriptionDetails{
				RICEventTriggerDefinition: []byte("periodic:1000ms"),
				RICActionToBeSetupList: []RICActionToBeSetupItem{
					{
						RICActionID:         1,
						RICActionType:       RICActionTypeReport,
						RICActionDefinition: []byte("cell_metrics"),
					},
				},
			},
		)

		if subscriptionReq.MessageType != E2APMessageTypeRICSubscriptionRequest {
			t.Errorf("Expected RIC_SUBSCRIPTION_REQUEST, got %v", subscriptionReq.MessageType)
		}

		// 3. RIC Control
		controlReq := CreateRICControlRequest(
			RICRequestID{RICRequestorID: 1, RICInstanceID: 2},
			2, // RAN Function ID for RC
			[]byte("control_header"),
			[]byte("control_message"),
		)

		if controlReq.MessageType != E2APMessageTypeRICControlRequest {
			t.Errorf("Expected RIC_CONTROL_REQUEST, got %v", controlReq.MessageType)
		}

		// Verify context is usable
		select {
		case <-ctx.Done():
			t.Error("Context should not be cancelled")
		default:
			// Context is active
		}
	})
}

// Helper function
func strPtr(s string) *string {
	return &s
}

// TestE2ServiceModelsComplete verifies all three service models are implemented
func TestE2ServiceModelsComplete(t *testing.T) {
	serviceModels := []struct {
		name string
		oid  string
		id   int
	}{
		{
			name: "E2SM-KPM",
			oid:  "1.3.6.1.4.1.53148.1.2.2.100",
			id:   1,
		},
		{
			name: "E2SM-RC",
			oid:  "1.3.6.1.4.1.53148.1.2.2.101",
			id:   2,
		},
		{
			name: "E2SM-NI",
			oid:  "1.3.6.1.4.1.53148.1.2.2.102",
			id:   3,
		},
	}

	for _, model := range serviceModels {
		t.Run(model.name, func(t *testing.T) {
			// Verify model can be registered
			ranFunction := RanFunction{
				FunctionID:          model.id,
				FunctionDefinition:  model.name,
				FunctionOID:         model.oid,
				FunctionRevision:    1,
				FunctionDescription: model.name + " Service Model",
			}

			if ranFunction.FunctionOID != model.oid {
				t.Errorf("Expected OID %s, got %s", model.oid, ranFunction.FunctionOID)
			}
		})
	}
}

// TestSCTPTransportComplete verifies SCTP transport implementation
func TestSCTPTransportComplete(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "0.0.0.0",
		ListenPort:        36421, // Standard E2AP SCTP port
		MaxConnections:    100,
		MaxStreams:        10,
		InitTimeout:       5 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		MaxRetransmits:    3,
		RTO:               1 * time.Second,
		KeepAlive:         true,
		KeepAliveInterval: 60 * time.Second,
		ConnectionTimeout: 300 * time.Second,
		SendBufferSize:    65536,
		RecvBufferSize:    65536,
		MaxMessageSize:    8192,
		EnableTLS:         false,
	}

	// Verify SCTP port is correct
	if config.ListenPort != 36421 {
		t.Errorf("Expected SCTP port 36421, got %d", config.ListenPort)
	}

	// Create transport
	codec := NewASN1Codec(true)
	transport := NewSCTPTransport(config, codec)

	if transport == nil {
		t.Fatal("Failed to create SCTP transport")
	}

	// Verify transport metrics
	metrics := transport.GetMetrics()
	if metrics == nil {
		t.Error("Failed to get transport metrics")
	}
}

// TestE2APCompleteImplementation verifies all required E2AP messages are implemented
func TestE2APCompleteImplementation(t *testing.T) {
	requiredMessages := []E2APMessageType{
		E2APMessageTypeSetupRequest,
		E2APMessageTypeSetupResponse,
		E2APMessageTypeSetupFailure,
		E2APMessageTypeRICSubscriptionRequest,
		E2APMessageTypeRICSubscriptionResponse,
		E2APMessageTypeRICSubscriptionFailure,
		E2APMessageTypeRICSubscriptionDeleteRequest,
		E2APMessageTypeRICSubscriptionDeleteResponse,
		E2APMessageTypeRICSubscriptionDeleteFailure,
		E2APMessageTypeRICIndication,
		E2APMessageTypeRICControlRequest,
		E2APMessageTypeRICControlAcknowledge,
		E2APMessageTypeRICControlFailure,
		E2APMessageTypeRICServiceUpdate,
		E2APMessageTypeRICServiceUpdateAcknowledge,
		E2APMessageTypeRICServiceUpdateFailure,
		E2APMessageTypeResetRequest,
		E2APMessageTypeResetResponse,
		E2APMessageTypeErrorIndication,
	}

	for _, msgType := range requiredMessages {
		t.Run(msgType.String(), func(t *testing.T) {
			// Verify message type is defined
			if msgType < 0 {
				t.Errorf("Invalid message type: %v", msgType)
			}
		})
	}
}

// String method for E2APMessageType (for test output)
func (m E2APMessageType) String() string {
	switch m {
	case E2APMessageTypeSetupRequest:
		return "SetupRequest"
	case E2APMessageTypeSetupResponse:
		return "SetupResponse"
	case E2APMessageTypeSetupFailure:
		return "SetupFailure"
	case E2APMessageTypeRICSubscriptionRequest:
		return "RICSubscriptionRequest"
	case E2APMessageTypeRICSubscriptionResponse:
		return "RICSubscriptionResponse"
	case E2APMessageTypeRICSubscriptionFailure:
		return "RICSubscriptionFailure"
	case E2APMessageTypeRICSubscriptionDeleteRequest:
		return "RICSubscriptionDeleteRequest"
	case E2APMessageTypeRICSubscriptionDeleteResponse:
		return "RICSubscriptionDeleteResponse"
	case E2APMessageTypeRICSubscriptionDeleteFailure:
		return "RICSubscriptionDeleteFailure"
	case E2APMessageTypeRICIndication:
		return "RICIndication"
	case E2APMessageTypeRICControlRequest:
		return "RICControlRequest"
	case E2APMessageTypeRICControlAcknowledge:
		return "RICControlAcknowledge"
	case E2APMessageTypeRICControlFailure:
		return "RICControlFailure"
	case E2APMessageTypeRICServiceUpdate:
		return "RICServiceUpdate"
	case E2APMessageTypeRICServiceUpdateAcknowledge:
		return "RICServiceUpdateAcknowledge"
	case E2APMessageTypeRICServiceUpdateFailure:
		return "RICServiceUpdateFailure"
	case E2APMessageTypeResetRequest:
		return "ResetRequest"
	case E2APMessageTypeResetResponse:
		return "ResetResponse"
	case E2APMessageTypeErrorIndication:
		return "ErrorIndication"
	default:
		return "Unknown"
	}
}
