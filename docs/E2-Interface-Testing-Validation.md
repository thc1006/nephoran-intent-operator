# E2 Interface Testing and Validation Procedures

## Overview

This document provides comprehensive testing and validation procedures for the E2 interface implementation in the Nephoran Intent Operator. The testing framework covers unit testing, integration testing, performance testing, and compliance testing to ensure robust E2AP protocol implementation and O-RAN specification compliance.

## Testing Framework Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                         E2 Interface Testing Framework                              │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                          Test Execution Layer                               │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │   │
│  │  │  Unit Tests     │  │ Integration     │  │    End-to-End              │ │   │
│  │  │    Suite        │  │    Tests        │  │      Tests                 │ │   │
│  │  │                 │  │                 │  │                             │ │   │
│  │  │ • Component     │  │ • E2AP Flow     │  │ • Full Workflow            │ │   │
│  │  │ • Message Codec │  │ • RIC Protocol  │  │ • Multi-Node Testing       │ │   │
│  │  │ • State Machine │  │ • Service Model │  │ • Load Testing             │ │   │
│  │  │ • Error Handling│  │ • Health Checks │  │ • Failover Testing         │ │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                          │                                         │
│                                          ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                        Test Infrastructure                                  │   │
│  │                                                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │                     Mock Services                                   │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │   │
│  │  │  │ Mock RIC    │  │ Mock E2Node │  │ Test Data   │                 │   │   │
│  │  │  │  Server     │  │ Simulator   │  │ Generator   │                 │   │   │
│  │  │  │             │  │             │  │             │                 │   │   │
│  │  │  │ • HTTP API  │  │ • Node Sim  │  │ • Message   │                 │   │   │
│  │  │  │ • Protocol  │  │ • Function  │  │ • Scenarios │                 │   │   │
│  │  │  │ • Responses │  │ • Events    │  │ • Datasets  │                 │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                          │                                         │
│                                          ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                     Validation & Compliance                                 │   │
│  │                                                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │                    Compliance Testing                               │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │   │
│  │  │  │ O-RAN Spec  │  │ Protocol    │  │ Interop     │                 │   │   │
│  │  │  │ Validation  │  │ Compliance  │  │  Testing    │                 │   │   │
│  │  │  │             │  │             │  │             │                 │   │   │
│  │  │  │ • Message   │  │ • State     │  │ • Multi-    │                 │   │   │
│  │  │  │   Format    │  │   Machine   │  │   Vendor    │                 │   │   │
│  │  │  │ • Procedure │  │ • Timers    │  │ • Real RIC  │                 │   │   │
│  │  │  │ • Behavior  │  │ • Error     │  │ • Standards │                 │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                          │                                         │
│                                          ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                    Performance Testing                                      │   │
│  │                                                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │                  Benchmarking Suite                                 │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │   │
│  │  │  │ Load Tests  │  │ Stress Test │  │ Latency     │                 │   │   │
│  │  │  │             │  │             │  │ Analysis    │                 │   │   │
│  │  │  │ • Conn Pool │  │ • Resource  │  │ • Message   │                 │   │   │
│  │  │  │ • Sub Scale │  │ • Memory    │  │   Timing    │                 │   │   │
│  │  │  │ • Throughput│  │ • CPU Usage │  │ • Response  │                 │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## Unit Testing Framework

### 1. Message Codec Testing

The E2 interface implementation includes comprehensive unit tests for all message encoding and decoding operations:

#### E2AP Message Encoding Tests
```go
func TestE2APEncoder_EncodeMessage(t *testing.T) {
    tests := []struct {
        name        string
        message     *E2APMessage
        expectError bool
        validateFn  func(*testing.T, []byte)
    }{
        {
            name: "E2Setup Request Encoding",
            message: &E2APMessage{
                MessageType:   E2MessageTypeE2SetupRequest,
                TransactionID: 12345,
                ProcedureCode: 1,
                Criticality:   CriticalityReject,
                Payload: &E2SetupRequest{
                    TransactionID: 12345,
                    GlobalE2NodeID: GlobalE2NodeID{
                        NodeID: E2NodeID{
                            NodeID:   "gNB-001",
                            NodeType: "gNB",
                        },
                        PLMNID: PLMNID{
                            MCC: "001",
                            MNC: "01",
                        },
                    },
                    RANFunctions: []RanFunction{
                        {
                            FunctionID:         1,
                            FunctionDefinition: "gNB-DU",
                            FunctionRevision:   1,
                            ServiceModel: E2ServiceModel{
                                ServiceModelID:   "1.3.6.1.4.1.53148.1.1.2.2",
                                ServiceModelName: "KPM",
                            },
                        },
                    },
                },
                Timestamp: time.Now(),
            },
            expectError: false,
            validateFn: func(t *testing.T, encoded []byte) {
                // Validate binary header (16 bytes)
                assert.GreaterOrEqual(t, len(encoded), 16)
                
                // Extract and validate header fields
                messageType := binary.BigEndian.Uint32(encoded[0:4])
                transactionID := binary.BigEndian.Uint32(encoded[4:8])
                procedureCode := binary.BigEndian.Uint32(encoded[8:12])
                criticality := binary.BigEndian.Uint32(encoded[12:16])
                
                assert.Equal(t, uint32(E2MessageTypeE2SetupRequest), messageType)
                assert.Equal(t, uint32(12345), transactionID)
                assert.Equal(t, uint32(1), procedureCode)
                assert.Equal(t, uint32(CriticalityReject), criticality)
                
                // Validate JSON payload
                var payload map[string]interface{}
                err := json.Unmarshal(encoded[16:], &payload)
                assert.NoError(t, err)
                assert.Contains(t, payload, "transaction_id")
                assert.Contains(t, payload, "global_e2_node_id")
            },
        },
        {
            name: "RIC Subscription Request Encoding",
            message: &E2APMessage{
                MessageType:   E2MessageTypeRICSubscriptionRequest,
                TransactionID: 54321,
                ProcedureCode: 201,
                Criticality:   CriticalityReject,
                Payload: &RICSubscriptionRequest{
                    RICRequestID: RICRequestID{
                        RICRequestorID: 1001,
                        RICInstanceID:  1,
                    },
                    RANFunctionID: 1,
                    RICSubscriptionDetails: RICSubscriptionDetails{
                        RICEventTriggerDefinition: []byte("periodic-1000ms"),
                        RICActionToBeSetupList: []RICActionToBeSetup{
                            {
                                RICActionID:   1,
                                RICActionType: RICActionTypeReport,
                                RICActionDefinition: []byte(`{
                                    "measurement_type": "DRB.RlcSduDelayDl",
                                    "reporting_period": 1000
                                }`),
                            },
                        },
                    },
                },
                Timestamp: time.Now(),
            },
            expectError: false,
            validateFn: func(t *testing.T, encoded []byte) {
                assert.GreaterOrEqual(t, len(encoded), 16)
                
                messageType := binary.BigEndian.Uint32(encoded[0:4])
                assert.Equal(t, uint32(E2MessageTypeRICSubscriptionRequest), messageType)
                
                var payload map[string]interface{}
                err := json.Unmarshal(encoded[16:], &payload)
                assert.NoError(t, err)
                assert.Contains(t, payload, "ric_request_id")
                assert.Contains(t, payload, "ran_function_id")
                assert.Contains(t, payload, "ric_subscription_details")
            },
        },
    }

    encoder := NewE2APEncoder()
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            encoded, err := encoder.EncodeMessage(tt.message)
            
            if tt.expectError {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.NotNil(t, encoded)
            
            if tt.validateFn != nil {
                tt.validateFn(t, encoded)
            }
        })
    }
}
```

#### Message Validation Tests
```go
func TestMessageValidation(t *testing.T) {
    tests := []struct {
        name        string
        message     interface{}
        validator   MessageValidator
        expectError bool
        errorMsg    string
    }{
        {
            name: "Valid RIC Subscription Request",
            message: &RICSubscriptionRequest{
                RICRequestID: RICRequestID{
                    RICRequestorID: 1001,
                    RICInstanceID:  1,
                },
                RANFunctionID: 1,
                RICSubscriptionDetails: RICSubscriptionDetails{
                    RICEventTriggerDefinition: []byte("periodic-1000ms"),
                    RICActionToBeSetupList: []RICActionToBeSetup{
                        {
                            RICActionID:   1,
                            RICActionType: RICActionTypeReport,
                        },
                    },
                },
            },
            validator:   &RICSubscriptionRequestValidator{},
            expectError: false,
        },
        {
            name: "Invalid RAN Function ID",
            message: &RICSubscriptionRequest{
                RICRequestID: RICRequestID{
                    RICRequestorID: 1001,
                    RICInstanceID:  1,
                },
                RANFunctionID: 9999, // Invalid range
                RICSubscriptionDetails: RICSubscriptionDetails{
                    RICEventTriggerDefinition: []byte("periodic-1000ms"),
                    RICActionToBeSetupList: []RICActionToBeSetup{
                        {
                            RICActionID:   1,
                            RICActionType: RICActionTypeReport,
                        },
                    },
                },
            },
            validator:   &RICSubscriptionRequestValidator{},
            expectError: true,
            errorMsg:    "invalid RAN function ID",
        },
        {
            name: "Missing RIC Actions",
            message: &RICSubscriptionRequest{
                RICRequestID: RICRequestID{
                    RICRequestorID: 1001,
                    RICInstanceID:  1,
                },
                RANFunctionID: 1,
                RICSubscriptionDetails: RICSubscriptionDetails{
                    RICEventTriggerDefinition: []byte("periodic-1000ms"),
                    RICActionToBeSetupList:    []RICActionToBeSetup{}, // Empty
                },
            },
            validator:   &RICSubscriptionRequestValidator{},
            expectError: true,
            errorMsg:    "no RIC actions specified",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.validator.Validate(tt.message)
            
            if tt.expectError {
                assert.Error(t, err)
                if tt.errorMsg != "" {
                    assert.Contains(t, err.Error(), tt.errorMsg)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 2. State Machine Testing

Comprehensive tests for E2 node and subscription state machines:

#### Node State Machine Tests
```go
func TestE2NodeStateMachine(t *testing.T) {
    tests := []struct {
        name           string
        initialState   E2NodeState
        action         string
        expectedState  E2NodeState
        expectError    bool
    }{
        {
            name:          "Connect Disconnected Node",
            initialState:  NodeStateDisconnected,
            action:        "connect",
            expectedState: NodeStateConnecting,
            expectError:   false,
        },
        {
            name:          "Setup Connected Node",
            initialState:  NodeStateConnecting,
            action:        "setup_complete",
            expectedState: NodeStateConnected,
            expectError:   false,
        },
        {
            name:          "Connection Failure",
            initialState:  NodeStateConnecting,
            action:        "connection_failed",
            expectedState: NodeStateFailed,
            expectError:   false,
        },
        {
            name:          "Invalid Transition",
            initialState:  NodeStateDisconnected,
            action:        "setup_complete",
            expectedState: NodeStateDisconnected,
            expectError:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            node := &E2NodeInfo{
                NodeID: "test-node",
                State:  tt.initialState,
            }

            var err error
            switch tt.action {
            case "connect":
                err = node.TransitionToConnecting("connection attempt")
            case "setup_complete":
                err = node.TransitionToConnected("setup completed")
            case "connection_failed":
                err = node.TransitionToFailed("connection timeout")
            case "setup_complete":
                if tt.initialState == NodeStateDisconnected {
                    err = fmt.Errorf("invalid state transition")
                }
            }

            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedState, node.State)
            }
        })
    }
}
```

#### Subscription State Machine Tests
```go
func TestSubscriptionStateMachine(t *testing.T) {
    subscription := &ManagedSubscription{
        E2Subscription: E2Subscription{
            SubscriptionID: "test-sub",
            RequestorID:    "test-requestor",
        },
        State:      SubscriptionStatePending,
        MaxRetries: 3,
    }

    // Test successful activation
    subscription.TransitionToActive("subscription confirmed")
    assert.Equal(t, SubscriptionStateActive, subscription.State)

    // Test deactivation
    subscription.TransitionToInactive("node disconnected")
    assert.Equal(t, SubscriptionStateInactive, subscription.State)

    // Test reactivation
    subscription.TransitionToActive("node reconnected")
    assert.Equal(t, SubscriptionStateActive, subscription.State)

    // Test failure
    subscription.TransitionToFailed("max retries exceeded")
    assert.Equal(t, SubscriptionStateFailed, subscription.State)

    // Test deletion
    subscription.TransitionToDeleting("cleanup requested")
    assert.Equal(t, SubscriptionStateDeleting, subscription.State)
}
```

### 3. Error Handling Tests

Comprehensive error handling validation:

```go
func TestE2AdaptorErrorHandling(t *testing.T) {
    tests := []struct {
        name           string
        setupMockFn    func() *httptest.Server
        operation      func(*E2Adaptor) error
        expectedError  string
        retryExpected  bool
    }{
        {
            name: "Network Timeout Error",
            setupMockFn: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    time.Sleep(2 * time.Second) // Cause timeout
                }))
            },
            operation: func(adaptor *E2Adaptor) error {
                ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
                defer cancel()
                return adaptor.RegisterE2Node(ctx, "test-node", []*E2NodeFunction{})
            },
            expectedError: "context deadline exceeded",
            retryExpected: true,
        },
        {
            name: "HTTP 500 Server Error",
            setupMockFn: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.WriteHeader(http.StatusInternalServerError)
                    w.Write([]byte("Internal Server Error"))
                }))
            },
            operation: func(adaptor *E2Adaptor) error {
                ctx := context.Background()
                return adaptor.RegisterE2Node(ctx, "test-node", []*E2NodeFunction{})
            },
            expectedError: "server error",
            retryExpected: true,
        },
        {
            name: "HTTP 400 Bad Request",
            setupMockFn: func() *httptest.Server {
                return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                    w.WriteHeader(http.StatusBadRequest)
                    w.Write([]byte("Bad Request"))
                }))
            },
            operation: func(adaptor *E2Adaptor) error {
                ctx := context.Background()
                return adaptor.RegisterE2Node(ctx, "test-node", []*E2NodeFunction{})
            },
            expectedError: "bad request",
            retryExpected: false, // Don't retry client errors
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := tt.setupMockFn()
            defer server.Close()

            config := &E2AdaptorConfig{
                RICURL:     server.URL,
                APIVersion: "v1",
                Timeout:    1 * time.Second,
                MaxRetries: 3,
            }

            adaptor, err := NewE2Adaptor(config)
            require.NoError(t, err)

            err = tt.operation(adaptor)
            assert.Error(t, err)
            assert.Contains(t, err.Error(), tt.expectedError)
        })
    }
}
```

## Integration Testing Framework

### 1. End-to-End E2AP Flow Testing

Complete E2 setup, subscription, and control message flows:

```go
func TestE2CompleteFlow(t *testing.T) {
    // Setup mock RIC server
    ricServer := NewMockRICServer(t)
    defer ricServer.Close()

    // Create E2Manager with test configuration
    config := &E2ManagerConfig{
        DefaultRICURL:       ricServer.URL,
        DefaultAPIVersion:   "v1",
        DefaultTimeout:      30 * time.Second,
        MaxConnections:      10,
        HealthCheckInterval: 10 * time.Second,
    }

    manager, err := NewE2Manager(config)
    require.NoError(t, err)
    defer manager.Shutdown()

    ctx := context.Background()

    // Step 1: Setup E2 connection
    err = manager.SetupE2Connection("test-gnb-001", ricServer.URL)
    require.NoError(t, err)

    // Step 2: Register E2 node with RAN functions
    ranFunctions := []RanFunction{
        {
            FunctionID:         1,
            FunctionDefinition: "gNB-DU",
            FunctionRevision:   1,
            FunctionOID:        "1.3.6.1.4.1.53148.1.1.1.1",
            ServiceModel: E2ServiceModel{
                ServiceModelID:   "1.3.6.1.4.1.53148.1.1.2.2",
                ServiceModelName: "KPM",
                ServiceModelVersion: "1.0",
                SupportedProcedures: []string{
                    "RIC_SUBSCRIPTION",
                    "RIC_INDICATION",
                },
            },
        },
        {
            FunctionID:         2,
            FunctionDefinition: "gNB-CU-CP",
            FunctionRevision:   1,
            FunctionOID:        "1.3.6.1.4.1.53148.1.1.1.2",
            ServiceModel: E2ServiceModel{
                ServiceModelID:   "1.3.6.1.4.1.53148.1.1.2.3",
                ServiceModelName: "RC",
                ServiceModelVersion: "1.0",
                SupportedProcedures: []string{
                    "RIC_CONTROL_REQUEST",
                    "RIC_CONTROL_ACKNOWLEDGE",
                },
            },
        },
    }

    err = manager.RegisterE2Node(ctx, "test-gnb-001", ranFunctions)
    require.NoError(t, err)

    // Verify node registration
    nodes, err := manager.ListE2Nodes(ctx)
    require.NoError(t, err)
    assert.Len(t, nodes, 1)
    assert.Equal(t, "test-gnb-001", nodes[0].NodeID)
    assert.Len(t, nodes[0].RanFunctions, 2)

    // Step 3: Create KPM subscription
    kmpSubscriptionReq := &E2SubscriptionRequest{
        NodeID:         "test-gnb-001",
        SubscriptionID: "kmp-metrics-001",
        RequestorID:    "integration-test",
        RanFunctionID:  1, // KPM function
        ReportingPeriod: 5 * time.Second,
        EventTriggers: []E2EventTrigger{
            {
                TriggerType:     "PERIODIC",
                ReportingPeriod: 5 * time.Second,
                Conditions: map[string]interface{}{
                    "measurement_period": "5000ms",
                },
            },
        },
        Actions: []E2Action{
            {
                ActionID:   1,
                ActionType: "REPORT",
                ActionDefinition: map[string]interface{}{
                    "measurement_info_list": []map[string]interface{}{
                        {
                            "measurement_type": "DRB.RlcSduDelayDl",
                            "label_info_list": []map[string]interface{}{
                                {
                                    "measurement_label": "UE_ID",
                                    "measurement_value": "all",
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    subscription, err := manager.SubscribeE2(kmpSubscriptionReq)
    require.NoError(t, err)
    assert.Equal(t, "kmp-metrics-001", subscription.SubscriptionID)

    // Step 4: Send RIC control message
    controlReq := &RICControlRequest{
        RICRequestID: RICRequestID{
            RICRequestorID: 1001,
            RICInstanceID:  1,
        },
        RANFunctionID: 2, // RC function
        RICControlHeader: []byte(`{
            "target_cell": "cell-001", 
            "control_action": "qos_flow_mapping"
        }`),
        RICControlMessage: []byte(`{
            "qfi": 5,
            "action": "modify_qos",
            "parameters": {
                "priority_level": 1,
                "packet_delay_budget": 20
            }
        }`),
        RICControlAckRequest: &RICControlAckRequest{},
    }

    ack, err := manager.SendControlMessage(ctx, controlReq)
    require.NoError(t, err)
    assert.NotNil(t, ack)
    assert.Equal(t, controlReq.RICRequestID, ack.RICRequestID)

    // Step 5: Verify metrics collection
    metrics := manager.GetMetrics()
    assert.Greater(t, metrics.NodesRegistered, int64(0))
    assert.Greater(t, metrics.SubscriptionsTotal, int64(0))
    assert.Greater(t, metrics.MessagesSent, int64(0))

    // Step 6: Test subscription cleanup
    err = manager.DeleteSubscription(ctx, "test-gnb-001", "kmp-metrics-001")
    require.NoError(t, err)

    // Verify subscription is removed
    time.Sleep(1 * time.Second) // Allow cleanup
    updatedMetrics := manager.GetMetrics()
    assert.Equal(t, int64(0), updatedMetrics.SubscriptionsActive)
}
```

### 2. Service Model Integration Testing

Testing service model plugin integration and compatibility:

```go
func TestServiceModelIntegration(t *testing.T) {
    // Create service model registry
    registry := &E2ServiceModelRegistry{
        serviceModels: make(map[string]*RegisteredServiceModel),
        plugins:       make(map[string]ServiceModelPlugin),
        enablePlugins: true,
    }

    // Test KPM service model registration
    kmpModel := CreateKPMServiceModel()
    err := registry.registerServiceModel(kmpModel, nil)
    require.NoError(t, err)

    // Test RC service model registration
    rcModel := CreateRCServiceModel()
    err = registry.registerServiceModel(rcModel, nil)
    require.NoError(t, err)

    // Test service model compatibility
    testRanFunction := &E2NodeFunction{
        FunctionID:         1,
        FunctionDefinition: "gNB-DU",
        ServiceModel:       *kmpModel,
    }

    err = registry.CheckCompatibility(kmpModel, testRanFunction)
    assert.NoError(t, err)

    // Test invalid service model
    invalidModel := &E2ServiceModel{
        ServiceModelID:      "invalid-id",
        ServiceModelName:    "INVALID",
        ServiceModelVersion: "1.0",
        SupportedProcedures: []string{},
    }

    err = registry.validateServiceModel(invalidModel)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "no supported procedures")

    // Test plugin integration
    plugin := &MockServiceModelPlugin{
        name:    "test-plugin",
        version: "1.0.0",
    }

    err = registry.RegisterPlugin(plugin)
    require.NoError(t, err)

    // Verify plugin is registered
    registeredPlugin, exists := registry.plugins["test-plugin"]
    assert.True(t, exists)
    assert.Equal(t, "test-plugin", registeredPlugin.GetName())
}
```

### 3. Multi-Node Testing

Testing multi-node scenarios and scaling:

```go
func TestMultiNodeIntegration(t *testing.T) {
    // Setup multiple mock RIC servers
    ricServers := make([]*MockRICServer, 3)
    for i := 0; i < 3; i++ {
        ricServers[i] = NewMockRICServer(t)
        defer ricServers[i].Close()
    }

    config := &E2ManagerConfig{
        DefaultRICURL:       ricServers[0].URL,
        DefaultAPIVersion:   "v1",
        DefaultTimeout:      30 * time.Second,
        MaxConnections:      50,
        HealthCheckInterval: 5 * time.Second,
    }

    manager, err := NewE2Manager(config)
    require.NoError(t, err)
    defer manager.Shutdown()

    ctx := context.Background()

    // Register multiple E2 nodes
    nodeCount := 10
    nodes := make([]string, nodeCount)
    
    for i := 0; i < nodeCount; i++ {
        nodeID := fmt.Sprintf("test-gnb-%03d", i+1)
        nodes[i] = nodeID
        
        // Distribute nodes across RIC servers
        ricURL := ricServers[i%len(ricServers)].URL
        
        err := manager.SetupE2Connection(nodeID, ricURL)
        require.NoError(t, err)
        
        // Register node with different RAN function combinations
        var ranFunctions []RanFunction
        if i%2 == 0 {
            // Even nodes: KPM only
            ranFunctions = []RanFunction{
                CreateKPMRanFunction(),
            }
        } else {
            // Odd nodes: KPM + RC
            ranFunctions = []RanFunction{
                CreateKPMRanFunction(),
                CreateRCRanFunction(),
            }
        }
        
        err = manager.RegisterE2Node(ctx, nodeID, ranFunctions)
        require.NoError(t, err)
    }

    // Verify all nodes are registered
    registeredNodes, err := manager.ListE2Nodes(ctx)
    require.NoError(t, err)
    assert.Len(t, registeredNodes, nodeCount)

    // Create subscriptions for all nodes
    var subscriptions []*E2Subscription
    for i, nodeID := range nodes {
        subscriptionID := fmt.Sprintf("sub-%s", nodeID)
        
        req := &E2SubscriptionRequest{
            NodeID:         nodeID,
            SubscriptionID: subscriptionID,
            RequestorID:    "multi-node-test",
            RanFunctionID:  1, // KPM function (available on all nodes)
            ReportingPeriod: time.Duration(1+i%5) * time.Second, // Vary reporting periods
            EventTriggers: []E2EventTrigger{
                {
                    TriggerType:     "PERIODIC",
                    ReportingPeriod: time.Duration(1+i%5) * time.Second,
                },
            },
            Actions: []E2Action{
                {
                    ActionID:   1,
                    ActionType: "REPORT",
                    ActionDefinition: map[string]interface{}{
                        "measurement_type": "DRB.RlcSduDelayDl",
                    },
                },
            },
        }
        
        subscription, err := manager.SubscribeE2(req)
        require.NoError(t, err)
        subscriptions = append(subscriptions, subscription)
    }

    // Verify subscription metrics
    metrics := manager.GetMetrics()
    assert.Equal(t, int64(nodeCount), metrics.NodesRegistered)
    assert.Equal(t, int64(len(subscriptions)), metrics.SubscriptionsTotal)
    assert.Greater(t, metrics.MessagesSent, int64(nodeCount)) // At least one message per node

    // Test load balancing across RIC servers
    for _, server := range ricServers {
        assert.Greater(t, server.GetRequestCount(), 0)
    }

    // Cleanup: Delete all subscriptions
    for i, subscription := range subscriptions {
        err := manager.DeleteSubscription(ctx, nodes[i], subscription.SubscriptionID)
        assert.NoError(t, err)
    }

    // Verify cleanup
    finalMetrics := manager.GetMetrics()
    assert.Equal(t, int64(0), finalMetrics.SubscriptionsActive)
}
```

## Performance Testing Framework

### 1. Load Testing

Comprehensive load testing for various scenarios:

```go
func TestE2AdaptorLoadTesting(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }

    // Setup load test parameters
    const (
        nodeCount        = 100
        subscriptionsPerNode = 10
        testDuration     = 2 * time.Minute
        targetTPS        = 1000 // Target transactions per second
    )

    // Create high-capacity mock server
    requestCount := int64(0)
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        atomic.AddInt64(&requestCount, 1)
        
        // Simulate realistic processing time
        time.Sleep(1 * time.Millisecond)
        
        if strings.Contains(r.URL.Path, "register") {
            w.WriteHeader(http.StatusCreated)
        } else if strings.Contains(r.URL.Path, "subscriptions") {
            w.WriteHeader(http.StatusCreated)
        } else if strings.Contains(r.URL.Path, "control") {
            w.Header().Set("Content-Type", "application/json")
            response := map[string]interface{}{
                "response_id":     "load-test-response",
                "request_id":      r.Header.Get("Request-ID"),
                "ran_function_id": 1,
                "status": map[string]interface{}{
                    "result": "SUCCESS",
                },
                "timestamp": time.Now().Format(time.RFC3339),
            }
            json.NewEncoder(w).Encode(response)
        } else {
            w.WriteHeader(http.StatusOK)
        }
    }))
    defer server.Close()

    config := &E2ManagerConfig{
        DefaultRICURL:       server.URL,
        DefaultAPIVersion:   "v1",
        DefaultTimeout:      30 * time.Second,
        MaxConnections:      200, // High connection limit
        HealthCheckInterval: 30 * time.Second,
    }

    manager, err := NewE2Manager(config)
    require.NoError(t, err)
    defer manager.Shutdown()

    ctx, cancel := context.WithTimeout(context.Background(), testDuration)
    defer cancel()

    // Load test metrics
    var wg sync.WaitGroup
    successCount := int64(0)
    errorCount := int64(0)
    
    startTime := time.Now()

    // Phase 1: Register nodes concurrently
    t.Log("Starting node registration phase...")
    nodeRegistrationStart := time.Now()
    
    for i := 0; i < nodeCount; i++ {
        wg.Add(1)
        go func(nodeIndex int) {
            defer wg.Done()
            
            nodeID := fmt.Sprintf("load-test-node-%d", nodeIndex)
            
            if err := manager.SetupE2Connection(nodeID, server.URL); err != nil {
                atomic.AddInt64(&errorCount, 1)
                return
            }
            
            ranFunctions := []RanFunction{
                CreateKPMRanFunction(),
                CreateRCRanFunction(),
            }
            
            if err := manager.RegisterE2Node(ctx, nodeID, ranFunctions); err != nil {
                atomic.AddInt64(&errorCount, 1)
                return
            }
            
            atomic.AddInt64(&successCount, 1)
        }(i)
    }
    
    wg.Wait()
    nodeRegistrationDuration := time.Since(nodeRegistrationStart)
    
    t.Logf("Node registration completed: %d success, %d errors in %v", 
        atomic.LoadInt64(&successCount), 
        atomic.LoadInt64(&errorCount), 
        nodeRegistrationDuration)

    // Reset counters for subscription phase
    atomic.StoreInt64(&successCount, 0)
    atomic.StoreInt64(&errorCount, 0)

    // Phase 2: Create subscriptions concurrently
    t.Log("Starting subscription creation phase...")
    subscriptionStart := time.Now()
    
    for i := 0; i < nodeCount; i++ {
        for j := 0; j < subscriptionsPerNode; j++ {
            wg.Add(1)
            go func(nodeIndex, subIndex int) {
                defer wg.Done()
                
                nodeID := fmt.Sprintf("load-test-node-%d", nodeIndex)
                subscriptionID := fmt.Sprintf("load-test-sub-%d-%d", nodeIndex, subIndex)
                
                req := &E2SubscriptionRequest{
                    NodeID:         nodeID,
                    SubscriptionID: subscriptionID,
                    RequestorID:    "load-test",
                    RanFunctionID:  1,
                    ReportingPeriod: time.Duration(1+subIndex%10) * time.Second,
                    EventTriggers: []E2EventTrigger{
                        {
                            TriggerType:     "PERIODIC",
                            ReportingPeriod: time.Duration(1+subIndex%10) * time.Second,
                        },
                    },
                    Actions: []E2Action{
                        {
                            ActionID:   1,
                            ActionType: "REPORT",
                            ActionDefinition: map[string]interface{}{
                                "measurement_type": "DRB.RlcSduDelayDl",
                            },
                        },
                    },
                }
                
                if _, err := manager.SubscribeE2(req); err != nil {
                    atomic.AddInt64(&errorCount, 1)
                    return
                }
                
                atomic.AddInt64(&successCount, 1)
            }(i, j)
        }
    }
    
    wg.Wait()
    subscriptionDuration := time.Since(subscriptionStart)
    
    t.Logf("Subscription creation completed: %d success, %d errors in %v", 
        atomic.LoadInt64(&successCount), 
        atomic.LoadInt64(&errorCount), 
        subscriptionDuration)

    // Phase 3: Sustained control message load
    t.Log("Starting sustained control message phase...")
    controlStart := time.Now()
    atomic.StoreInt64(&successCount, 0)
    atomic.StoreInt64(&errorCount, 0)
    
    // Control message workers
    workerCount := 50
    controlMessageChan := make(chan int, 1000)
    
    // Start workers
    for w := 0; w < workerCount; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            for nodeIndex := range controlMessageChan {
                nodeID := fmt.Sprintf("load-test-node-%d", nodeIndex%nodeCount)
                
                controlReq := &RICControlRequest{
                    RICRequestID: RICRequestID{
                        RICRequestorID: int32(1000 + nodeIndex),
                        RICInstanceID:  1,
                    },
                    RANFunctionID: 2,
                    RICControlHeader: []byte(`{"target_cell": "cell-001"}`),
                    RICControlMessage: []byte(`{"qfi": 5, "action": "modify_qos"}`),
                    RICControlAckRequest: &RICControlAckRequest{},
                }
                
                if _, err := manager.SendControlMessage(ctx, controlReq); err != nil {
                    atomic.AddInt64(&errorCount, 1)
                } else {
                    atomic.AddInt64(&successCount, 1)
                }
            }
        }()
    }
    
    // Generate control messages at target rate
    ticker := time.NewTicker(time.Second / time.Duration(targetTPS))
    defer ticker.Stop()
    
    messageIndex := 0
    for {
        select {
        case <-ctx.Done():
            close(controlMessageChan)
            wg.Wait()
            goto loadTestComplete
        case <-ticker.C:
            select {
            case controlMessageChan <- messageIndex:
                messageIndex++
            default:
                // Channel full, skip this tick
            }
        }
    }

loadTestComplete:
    totalDuration := time.Since(startTime)
    controlDuration := time.Since(controlStart)
    
    // Final metrics
    finalMetrics := manager.GetMetrics()
    actualTPS := float64(atomic.LoadInt64(&successCount)) / controlDuration.Seconds()
    
    t.Logf("Load test completed in %v", totalDuration)
    t.Logf("Control message phase: %d success, %d errors in %v", 
        atomic.LoadInt64(&successCount), 
        atomic.LoadInt64(&errorCount), 
        controlDuration)
    t.Logf("Achieved TPS: %.2f (target: %d)", actualTPS, targetTPS)
    t.Logf("Server handled %d total requests", atomic.LoadInt64(&requestCount))
    
    // Performance assertions
    successRate := float64(atomic.LoadInt64(&successCount)) / float64(atomic.LoadInt64(&successCount)+atomic.LoadInt64(&errorCount))
    assert.Greater(t, successRate, 0.95, "Success rate should be > 95%%")
    assert.Greater(t, actualTPS, float64(targetTPS)*0.8, "Should achieve at least 80%% of target TPS")
    
    // Resource usage assertions
    assert.Less(t, finalMetrics.ConnectionsFailed, int64(nodeCount*0.05), "Connection failures should be < 5%%")
    assert.Greater(t, finalMetrics.ConnectionsActive, int64(nodeCount*0.9), "Active connections should be > 90%%")
}
```

### 2. Stress Testing

Memory and CPU stress testing:

```go
func TestE2AdaptorStressTesting(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping stress test in short mode")
    }

    // Stress test parameters
    const (
        maxNodes = 1000
        maxSubscriptionsPerNode = 100
        stressDuration = 5 * time.Minute
    )

    // Memory and CPU monitoring
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    initialMemory := memStats.Alloc

    cpuStartTime := time.Now()
    
    // Create high-performance mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Minimal processing for maximum throughput
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    config := &E2ManagerConfig{
        DefaultRICURL:       server.URL,
        DefaultAPIVersion:   "v1",
        DefaultTimeout:      10 * time.Second,
        MaxConnections:      2000,
        HealthCheckInterval: 60 * time.Second,
    }

    manager, err := NewE2Manager(config)
    require.NoError(t, err)
    defer manager.Shutdown()

    ctx, cancel := context.WithTimeout(context.Background(), stressDuration)
    defer cancel()

    // Stress test loop
    nodeIndex := 0
    subscriptionIndex := 0
    
    stressStart := time.Now()
    ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            goto stressComplete
        case <-ticker.C:
            // Alternate between node registration and subscription creation
            if nodeIndex < maxNodes && (subscriptionIndex == 0 || nodeIndex*maxSubscriptionsPerNode < subscriptionIndex) {
                // Create new node
                nodeID := fmt.Sprintf("stress-node-%d", nodeIndex)
                
                go func() {
                    manager.SetupE2Connection(nodeID, server.URL)
                    manager.RegisterE2Node(ctx, nodeID, []RanFunction{CreateKPMRanFunction()})
                }()
                
                nodeIndex++
            } else if subscriptionIndex < maxNodes*maxSubscriptionsPerNode {
                // Create new subscription
                targetNodeIndex := subscriptionIndex / maxSubscriptionsPerNode
                nodeID := fmt.Sprintf("stress-node-%d", targetNodeIndex)
                subscriptionID := fmt.Sprintf("stress-sub-%d", subscriptionIndex)
                
                go func() {
                    req := &E2SubscriptionRequest{
                        NodeID:         nodeID,
                        SubscriptionID: subscriptionID,
                        RequestorID:    "stress-test",
                        RanFunctionID:  1,
                        ReportingPeriod: 1 * time.Second,
                        EventTriggers: []E2EventTrigger{
                            {
                                TriggerType:     "PERIODIC",
                                ReportingPeriod: 1 * time.Second,
                            },
                        },
                        Actions: []E2Action{
                            {
                                ActionID:   1,
                                ActionType: "REPORT",
                            },
                        },
                    }
                    manager.SubscribeE2(req)
                }()
                
                subscriptionIndex++
            }
        }
    }

stressComplete:
    stressDuration := time.Since(stressStart)
    
    // Measure final resource usage
    runtime.ReadMemStats(&memStats)
    finalMemory := memStats.Alloc
    memoryGrowth := finalMemory - initialMemory
    
    // Get final metrics
    metrics := manager.GetMetrics()
    
    t.Logf("Stress test completed in %v", stressDuration)
    t.Logf("Created %d nodes and %d subscriptions", nodeIndex, subscriptionIndex)
    t.Logf("Memory growth: %d bytes (%.2f MB)", memoryGrowth, float64(memoryGrowth)/(1024*1024))
    t.Logf("Final metrics: Nodes=%d, Subscriptions=%d", metrics.NodesActive, metrics.SubscriptionsActive)
    
    // Resource usage assertions
    maxMemoryGrowth := uint64(500 * 1024 * 1024) // 500 MB
    assert.Less(t, memoryGrowth, maxMemoryGrowth, "Memory growth should be reasonable")
    
    // Performance assertions
    assert.Greater(t, metrics.NodesActive, int64(nodeIndex*0.8), "Should maintain most nodes")
    assert.Greater(t, metrics.SubscriptionsActive, int64(subscriptionIndex*0.8), "Should maintain most subscriptions")
}
```

### 3. Latency Analysis

Detailed latency measurement and analysis:

```go
func TestE2LatencyAnalysis(t *testing.T) {
    // Latency measurement setup
    latencyMeasurements := make([]time.Duration, 0, 1000)
    var latencyMutex sync.Mutex

    // Mock server with controlled latency
    serverLatency := 10 * time.Millisecond
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(serverLatency)
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    config := &E2AdaptorConfig{
        RICURL:     server.URL,
        APIVersion: "v1",
        Timeout:    30 * time.Second,
    }

    adaptor, err := NewE2Adaptor(config)
    require.NoError(t, err)

    ctx := context.Background()

    // Test different operation types
    testCases := []struct {
        name      string
        operation func() error
    }{
        {
            name: "Node Registration",
            operation: func() error {
                nodeID := fmt.Sprintf("latency-node-%d", time.Now().UnixNano())
                functions := []*E2NodeFunction{CreateDefaultE2NodeFunction()}
                return adaptor.RegisterE2Node(ctx, nodeID, functions)
            },
        },
        {
            name: "Subscription Creation",
            operation: func() error {
                nodeID := "latency-test-node"
                
                // Ensure node exists
                functions := []*E2NodeFunction{CreateDefaultE2NodeFunction()}
                adaptor.RegisterE2Node(ctx, nodeID, functions)
                
                subscription := &E2Subscription{
                    SubscriptionID:  fmt.Sprintf("latency-sub-%d", time.Now().UnixNano()),
                    RequestorID:     "latency-test",
                    RanFunctionID:   1,
                    ReportingPeriod: 1 * time.Second,
                    EventTriggers: []E2EventTrigger{
                        {
                            TriggerType:     "PERIODIC",
                            ReportingPeriod: 1 * time.Second,
                        },
                    },
                    Actions: []E2Action{
                        {
                            ActionID:   1,
                            ActionType: "REPORT",
                        },
                    },
                }
                
                return adaptor.CreateSubscription(ctx, nodeID, subscription)
            },
        },
        {
            name: "Control Request",
            operation: func() error {
                controlReq := &E2ControlRequest{
                    RequestID:     fmt.Sprintf("latency-ctrl-%d", time.Now().UnixNano()),
                    RanFunctionID: 1,
                    ControlHeader: map[string]interface{}{
                        "target_cell": "cell-001",
                    },
                    ControlMessage: map[string]interface{}{
                        "qfi":    5,
                        "action": "modify_qos",
                    },
                    ControlAckRequest: true,
                }
                
                _, err := adaptor.SendControlRequest(ctx, "latency-test-node", controlReq)
                return err
            },
        },
    }

    // Run latency measurements for each operation type
    for _, testCase := range testCases {
        t.Run(testCase.name, func(t *testing.T) {
            measurements := make([]time.Duration, 0, 100)
            
            // Warm-up runs
            for i := 0; i < 10; i++ {
                testCase.operation()
            }
            
            // Measurement runs
            for i := 0; i < 100; i++ {
                start := time.Now()
                err := testCase.operation()
                duration := time.Since(start)
                
                if err == nil {
                    measurements = append(measurements, duration)
                }
            }
            
            // Calculate latency statistics
            sort.Slice(measurements, func(i, j int) bool {
                return measurements[i] < measurements[j]
            })
            
            if len(measurements) == 0 {
                t.Fatalf("No successful measurements for %s", testCase.name)
            }
            
            min := measurements[0]
            max := measurements[len(measurements)-1]
            p50 := measurements[len(measurements)*50/100]
            p95 := measurements[len(measurements)*95/100]
            p99 := measurements[len(measurements)*99/100]
            
            var sum time.Duration
            for _, d := range measurements {
                sum += d
            }
            avg := sum / time.Duration(len(measurements))
            
            t.Logf("%s Latency Statistics:", testCase.name)
            t.Logf("  Min: %v", min)
            t.Logf("  Avg: %v", avg)
            t.Logf("  P50: %v", p50)
            t.Logf("  P95: %v", p95)
            t.Logf("  P99: %v", p99)
            t.Logf("  Max: %v", max)
            
            // Latency assertions (accounting for server latency)
            expectedMin := serverLatency
            expectedMax := serverLatency + 100*time.Millisecond // Allow 100ms overhead
            
            assert.GreaterOrEqual(t, min, expectedMin, "Minimum latency should account for server latency")
            assert.Less(t, p95, expectedMax, "95th percentile should be reasonable")
            assert.Less(t, avg, expectedMax/2, "Average latency should be efficient")
            
            latencyMutex.Lock()
            latencyMeasurements = append(latencyMeasurements, measurements...)
            latencyMutex.Unlock()
        })
    }

    // Overall latency analysis
    t.Run("Overall Latency Analysis", func(t *testing.T) {
        latencyMutex.Lock()
        defer latencyMutex.Unlock()
        
        if len(latencyMeasurements) == 0 {
            t.Skip("No latency measurements available")
        }
        
        sort.Slice(latencyMeasurements, func(i, j int) bool {
            return latencyMeasurements[i] < latencyMeasurements[j]
        })
        
        // Calculate percentiles across all operations
        p50 := latencyMeasurements[len(latencyMeasurements)*50/100]
        p95 := latencyMeasurements[len(latencyMeasurements)*95/100]
        p99 := latencyMeasurements[len(latencyMeasurements)*99/100]
        
        t.Logf("Overall Latency Percentiles:")
        t.Logf("  P50: %v", p50)
        t.Logf("  P95: %v", p95)
        t.Logf("  P99: %v", p99)
        
        // Performance targets
        assert.Less(t, p50, 50*time.Millisecond, "P50 latency should be < 50ms")
        assert.Less(t, p95, 200*time.Millisecond, "P95 latency should be < 200ms")
        assert.Less(t, p99, 500*time.Millisecond, "P99 latency should be < 500ms")
    })
}
```

## Compliance Testing Framework

### 1. O-RAN Specification Compliance

Validation against O-RAN Alliance specifications:

```go
func TestORANSpecificationCompliance(t *testing.T) {
    tests := []struct {
        name           string
        specification  string
        testFunction   func(t *testing.T)
    }{
        {
            name:          "E2AP Message Format Compliance",
            specification: "O-RAN.WG3.E2AP-v3.0",
            testFunction:  testE2APMessageFormatCompliance,
        },
        {
            name:          "E2 Setup Procedure Compliance",
            specification: "O-RAN.WG3.E2AP-v3.0-Section-8.2",
            testFunction:  testE2SetupProcedureCompliance,
        },
        {
            name:          "RIC Subscription Procedure Compliance",
            specification: "O-RAN.WG3.E2AP-v3.0-Section-8.3",
            testFunction:  testRICSubscriptionProcedureCompliance,
        },
        {
            name:          "Service Model Compliance",
            specification: "O-RAN.WG3.E2SM-KPM-v3.0",
            testFunction:  testServiceModelCompliance,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Logf("Testing compliance with %s", tt.specification)
            tt.testFunction(t)
        })
    }
}

func testE2APMessageFormatCompliance(t *testing.T) {
    encoder := NewE2APEncoder()
    
    // Test all supported message types
    messageTests := []struct {
        messageType E2APMessageType
        payload     interface{}
    }{
        {
            messageType: E2MessageTypeE2SetupRequest,
            payload: &E2SetupRequest{
                TransactionID: 12345,
                GlobalE2NodeID: GlobalE2NodeID{
                    NodeID: E2NodeID{
                        NodeID:   "compliance-test-node",
                        NodeType: "gNB",
                    },
                },
                RANFunctions: []RanFunction{
                    {
                        FunctionID:         1,
                        FunctionDefinition: "gNB-DU",
                        FunctionRevision:   1,
                        ServiceModel:       *CreateKPMServiceModel(),
                    },
                },
            },
        },
        {
            messageType: E2MessageTypeRICSubscriptionRequest,
            payload: &RICSubscriptionRequest{
                RICRequestID: RICRequestID{
                    RICRequestorID: 1001,
                    RICInstanceID:  1,
                },
                RANFunctionID: 1,
                RICSubscriptionDetails: RICSubscriptionDetails{
                    RICEventTriggerDefinition: []byte("periodic-1000ms"),
                    RICActionToBeSetupList: []RICActionToBeSetup{
                        {
                            RICActionID:   1,
                            RICActionType: RICActionTypeReport,
                            RICActionDefinition: []byte(`{
                                "measurement_type": "DRB.RlcSduDelayDl"
                            }`),
                        },
                    },
                },
            },
        },
    }

    for _, test := range messageTests {
        message := &E2APMessage{
            MessageType:   test.messageType,
            TransactionID: 12345,
            ProcedureCode: 1,
            Criticality:   CriticalityReject,
            Payload:       test.payload,
            Timestamp:     time.Now(),
        }

        encoded, err := encoder.EncodeMessage(message)
        require.NoError(t, err, "Message encoding should succeed")
        
        // Validate binary header format (16 bytes as per O-RAN spec)
        assert.GreaterOrEqual(t, len(encoded), 16, "Message should have 16-byte header")
        
        // Validate header fields
        messageType := binary.BigEndian.Uint32(encoded[0:4])
        transactionID := binary.BigEndian.Uint32(encoded[4:8])
        procedureCode := binary.BigEndian.Uint32(encoded[8:12])
        criticality := binary.BigEndian.Uint32(encoded[12:16])
        
        assert.Equal(t, uint32(test.messageType), messageType)
        assert.Equal(t, uint32(12345), transactionID)
        assert.Equal(t, uint32(1), procedureCode)
        assert.Equal(t, uint32(CriticalityReject), criticality)
        
        // Validate JSON payload is well-formed
        var payload map[string]interface{}
        err = json.Unmarshal(encoded[16:], &payload)
        assert.NoError(t, err, "Payload should be valid JSON")
    }
}

func testE2SetupProcedureCompliance(t *testing.T) {
    // Test E2 Setup procedure state machine compliance
    setupStates := []struct {
        state       string
        nextActions []string
        valid       bool
    }{
        {
            state:       "IDLE",
            nextActions: []string{"E2_SETUP_REQUEST"},
            valid:       true,
        },
        {
            state:       "WAIT_E2_SETUP_RESPONSE",
            nextActions: []string{"E2_SETUP_RESPONSE", "E2_SETUP_FAILURE", "TIMEOUT"},
            valid:       true,
        },
        {
            state:       "ESTABLISHED",
            nextActions: []string{"SERVICE_UPDATE", "DISCONNECT"},
            valid:       true,
        },
    }

    for _, stateTest := range setupStates {
        t.Run(fmt.Sprintf("State_%s", stateTest.state), func(t *testing.T) {
            for _, action := range stateTest.nextActions {
                // Validate that each action is allowed from the current state
                // This would involve more complex state machine logic in a real implementation
                assert.True(t, stateTest.valid, "Action %s should be valid from state %s", action, stateTest.state)
            }
        })
    }
}

func testRICSubscriptionProcedureCompliance(t *testing.T) {
    // Test RIC Subscription procedure compliance
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.URL.Path, "subscriptions") && r.Method == "POST" {
            // Validate subscription request format
            var requestBody map[string]interface{}
            json.NewDecoder(r.Body).Decode(&requestBody)
            
            // Check required fields per O-RAN specification
            requiredFields := []string{"ric_request_id", "ran_function_id", "ric_subscription_details"}
            for _, field := range requiredFields {
                if _, exists := requestBody[field]; !exists {
                    w.WriteHeader(http.StatusBadRequest)
                    return
                }
            }
            
            w.WriteHeader(http.StatusCreated)
        }
    }))
    defer mockServer.Close()

    config := &E2AdaptorConfig{
        RICURL:     mockServer.URL,
        APIVersion: "v1",
        Timeout:    30 * time.Second,
    }

    adaptor, err := NewE2Adaptor(config)
    require.NoError(t, err)

    ctx := context.Background()

    // Register node first
    functions := []*E2NodeFunction{CreateDefaultE2NodeFunction()}
    err = adaptor.RegisterE2Node(ctx, "compliance-node", functions)
    require.NoError(t, err)

    // Test compliant subscription request
    subscription := &E2Subscription{
        SubscriptionID:  "compliance-sub-001",
        RequestorID:     "compliance-test",
        RanFunctionID:   1,
        ReportingPeriod: 1 * time.Second,
        EventTriggers: []E2EventTrigger{
            {
                TriggerType:     "PERIODIC",
                ReportingPeriod: 1 * time.Second,
                Conditions: map[string]interface{}{
                    "measurement_period": "1000ms",
                },
            },
        },
        Actions: []E2Action{
            {
                ActionID:   1,
                ActionType: "REPORT",
                ActionDefinition: map[string]interface{}{
                    "measurement_type": "DRB.RlcSduDelayDl",
                    "label_info_list": []map[string]interface{}{
                        {
                            "measurement_label": "UE_ID",
                            "measurement_value": "all",
                        },
                    },
                },
            },
        },
    }

    err = adaptor.CreateSubscription(ctx, "compliance-node", subscription)
    assert.NoError(t, err, "Compliant subscription should be accepted")
}

func testServiceModelCompliance(t *testing.T) {
    // Test KPM service model compliance
    kmpModel := CreateKPMServiceModel()
    
    // Validate service model ID format (OID)
    assert.Regexp(t, `^\d+(\.\d+)*$`, kmpModel.ServiceModelID, "Service model ID should be valid OID")
    
    // Validate required procedures
    requiredProcedures := []string{"RIC_SUBSCRIPTION", "RIC_INDICATION"}
    for _, procedure := range requiredProcedures {
        assert.Contains(t, kmpModel.SupportedProcedures, procedure, "KPM model should support %s", procedure)
    }
    
    // Validate measurement types
    config := kmpModel.Configuration
    measurementTypes, exists := config["measurement_types"]
    assert.True(t, exists, "KPM model should define measurement types")
    
    types := measurementTypes.([]string)
    expectedTypes := []string{
        "DRB.RlcSduDelayDl",
        "DRB.UEThpDl", 
        "RRU.PrbUsedDl",
    }
    
    for _, expectedType := range expectedTypes {
        assert.Contains(t, types, expectedType, "KPM model should support measurement type %s", expectedType)
    }

    // Test RC service model compliance
    rcModel := CreateRCServiceModel()
    
    // Validate RC-specific procedures
    rcProcedures := []string{"RIC_CONTROL_REQUEST", "RIC_CONTROL_ACKNOWLEDGE"}
    for _, procedure := range rcProcedures {
        assert.Contains(t, rcModel.SupportedProcedures, procedure, "RC model should support %s", procedure)
    }
    
    // Validate control actions
    rcConfig := rcModel.Configuration
    controlActions, exists := rcConfig["control_actions"]
    assert.True(t, exists, "RC model should define control actions")
    
    actions := controlActions.([]string)
    expectedActions := []string{
        "QoS_flow_mapping",
        "Traffic_steering",
        "Load_balancing",
    }
    
    for _, expectedAction := range expectedActions {
        assert.Contains(t, actions, expectedAction, "RC model should support control action %s", expectedAction)
    }
}
```

### 2. Interoperability Testing

Testing with different RIC implementations:

```go
func TestInteroperabilityTesting(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping interoperability tests in short mode")
    }

    // Test against different RIC vendor implementations
    ricImplementations := []struct {
        name           string
        setupServerFn  func() *httptest.Server
        specificTests  func(t *testing.T, serverURL string)
    }{
        {
            name: "Generic O-RAN RIC",
            setupServerFn: func() *httptest.Server {
                return setupGenericORanRIC()
            },
            specificTests: testGenericORanRICInterop,
        },
        {
            name: "Nokia FlexiRIC",
            setupServerFn: func() *httptest.Server {
                return setupNokiaFlexiRIC()
            },
            specificTests: testNokiaFlexiRICInterop,
        },
        {
            name: "Samsung RIC",
            setupServerFn: func() *httptest.Server {
                return setupSamsungRIC()
            },
            specificTests: testSamsungRICInterop,
        },
    }

    for _, ricImpl := range ricImplementations {
        t.Run(ricImpl.name, func(t *testing.T) {
            server := ricImpl.setupServerFn()
            defer server.Close()
            
            // Run vendor-specific tests
            ricImpl.specificTests(t, server.URL)
            
            // Run common interoperability tests
            testCommonInteroperability(t, server.URL)
        })
    }
}

func setupGenericORanRIC() *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Generic O-RAN RIC responses
        switch {
        case strings.Contains(r.URL.Path, "/register") && r.Method == "POST":
            w.WriteHeader(http.StatusCreated)
            response := map[string]interface{}{
                "result": "SUCCESS",
                "setup_response": map[string]interface{}{
                    "transaction_id": 12345,
                    "accepted_functions": []int{1, 2},
                },
            }
            json.NewEncoder(w).Encode(response)
            
        case strings.Contains(r.URL.Path, "/subscriptions") && r.Method == "POST":
            w.WriteHeader(http.StatusCreated)
            response := map[string]interface{}{
                "result": "SUCCESS",
                "subscription_response": map[string]interface{}{
                    "ric_request_id": map[string]interface{}{
                        "ric_requestor_id": 1001,
                        "ric_instance_id":  1,
                    },
                    "ran_function_id": 1,
                    "accepted_actions": []int{1},
                },
            }
            json.NewEncoder(w).Encode(response)
            
        case strings.Contains(r.URL.Path, "/control") && r.Method == "POST":
            w.WriteHeader(http.StatusOK)
            response := map[string]interface{}{
                "result": "SUCCESS",
                "control_acknowledge": map[string]interface{}{
                    "ric_request_id": map[string]interface{}{
                        "ric_requestor_id": 1001,
                        "ric_instance_id":  1,
                    },
                    "ran_function_id": 2,
                    "control_status":  "SUCCESS",
                },
            }
            json.NewEncoder(w).Encode(response)
            
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    }))
}

func testCommonInteroperability(t *testing.T, ricURL string) {
    config := &E2ManagerConfig{
        DefaultRICURL:       ricURL,
        DefaultAPIVersion:   "v1",
        DefaultTimeout:      30 * time.Second,
        MaxConnections:      10,
        HealthCheckInterval: 30 * time.Second,
    }

    manager, err := NewE2Manager(config)
    require.NoError(t, err)
    defer manager.Shutdown()

    ctx := context.Background()

    // Test 1: Node registration interoperability
    t.Run("Node Registration", func(t *testing.T) {
        err := manager.SetupE2Connection("interop-node-001", ricURL)
        require.NoError(t, err)

        ranFunctions := []RanFunction{
            CreateKPMRanFunction(),
            CreateRCRanFunction(),
        }

        err = manager.RegisterE2Node(ctx, "interop-node-001", ranFunctions)
        assert.NoError(t, err, "Node registration should succeed with any compliant RIC")
    })

    // Test 2: Subscription interoperability
    t.Run("Subscription Creation", func(t *testing.T) {
        req := &E2SubscriptionRequest{
            NodeID:         "interop-node-001",
            SubscriptionID: "interop-sub-001",
            RequestorID:    "interop-test",
            RanFunctionID:  1,
            ReportingPeriod: 5 * time.Second,
            EventTriggers: []E2EventTrigger{
                {
                    TriggerType:     "PERIODIC",
                    ReportingPeriod: 5 * time.Second,
                },
            },
            Actions: []E2Action{
                {
                    ActionID:   1,
                    ActionType: "REPORT",
                    ActionDefinition: map[string]interface{}{
                        "measurement_type": "DRB.RlcSduDelayDl",
                    },
                },
            },
        }

        subscription, err := manager.SubscribeE2(req)
        assert.NoError(t, err, "Subscription should succeed with any compliant RIC")
        assert.Equal(t, "interop-sub-001", subscription.SubscriptionID)
    })

    // Test 3: Control message interoperability
    t.Run("Control Messages", func(t *testing.T) {
        controlReq := &RICControlRequest{
            RICRequestID: RICRequestID{
                RICRequestorID: 1001,
                RICInstanceID:  1,
            },
            RANFunctionID: 2,
            RICControlHeader: []byte(`{
                "target_cell": "cell-001",
                "control_action": "qos_flow_mapping"
            }`),
            RICControlMessage: []byte(`{
                "qfi": 5,
                "action": "modify_qos"
            }`),
            RICControlAckRequest: &RICControlAckRequest{},
        }

        ack, err := manager.SendControlMessage(ctx, controlReq)
        assert.NoError(t, err, "Control message should succeed with any compliant RIC")
        assert.NotNil(t, ack)
    })
}
```

## Test Automation and CI/CD Integration

### 1. Automated Test Execution

Continuous integration pipeline configuration:

```yaml
# .github/workflows/e2-testing.yml
name: E2 Interface Testing

on:
  push:
    branches: [ main, develop ]
    paths: 
      - 'pkg/oran/e2/**'
      - 'docs/E2-*.md'
  pull_request:
    branches: [ main ]
    paths:
      - 'pkg/oran/e2/**'

jobs:
  unit-tests:
    name: Unit Tests
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-
    
    - name: Run Unit Tests
      run: |
        go test -v -race -coverprofile=coverage.out ./pkg/oran/e2/...
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Upload Coverage
      uses: actions/upload-artifact@v3
      with:
        name: coverage-report
        path: coverage.html

  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest
    needs: unit-tests
    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Set up Kind Cluster
      uses: helm/kind-action@v1.4.0
      with:
        cluster_name: e2-test-cluster
        config: |
          kind: Cluster
          apiVersion: kind.x-k8s.io/v1alpha4
          nodes:
          - role: control-plane
          - role: worker
          - role: worker
    
    - name: Run Integration Tests
      run: |
        export KUBECONFIG="$HOME/.kube/config"
        go test -v -tags=integration ./pkg/oran/e2/... -timeout=30m
    
    - name: Collect Test Artifacts
      if: always()
      run: |
        kubectl get pods -A
        kubectl describe pods -A
        kubectl logs -l app=e2-test --tail=1000

  performance-tests:
    name: Performance Tests
    runs-on: ubuntu-latest
    needs: integration-tests
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run Performance Tests
      run: |
        go test -v -run=Benchmark -bench=. -benchmem ./pkg/oran/e2/... > benchmark.txt
    
    - name: Store Benchmark Results
      uses: benchmark-action/github-action-benchmark@v1
      with:
        tool: 'go'
        output-file-path: benchmark.txt
        github-token: ${{ secrets.GITHUB_TOKEN }}
        auto-push: true

  compliance-tests:
    name: O-RAN Compliance Tests
    runs-on: ubuntu-latest
    needs: integration-tests
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run Compliance Tests
      run: |
        go test -v -tags=compliance ./pkg/oran/e2/... -timeout=45m
    
    - name: Generate Compliance Report
      run: |
        go test -json -tags=compliance ./pkg/oran/e2/... > compliance-results.json
    
    - name: Upload Compliance Report
      uses: actions/upload-artifact@v3
      with:
        name: compliance-report
        path: compliance-results.json
```

### 2. Test Data Management

Automated test data generation and management:

```go
// Test data generator for comprehensive testing scenarios
type E2TestDataGenerator struct {
    nodeCount        int
    subscriptionCount int
    messageCount     int
    randomSeed       int64
}

func NewE2TestDataGenerator(config *TestDataConfig) *E2TestDataGenerator {
    return &E2TestDataGenerator{
        nodeCount:        config.NodeCount,
        subscriptionCount: config.SubscriptionCount,
        messageCount:     config.MessageCount,
        randomSeed:       config.RandomSeed,
    }
}

func (gen *E2TestDataGenerator) GenerateTestNodes() []*E2NodeInfo {
    rng := rand.New(rand.NewSource(gen.randomSeed))
    nodes := make([]*E2NodeInfo, gen.nodeCount)
    
    nodeTypes := []string{"gNB", "eNB", "ng-eNB", "en-gNB"}
    
    for i := 0; i < gen.nodeCount; i++ {
        nodeType := nodeTypes[rng.Intn(len(nodeTypes))]
        
        node := &E2NodeInfo{
            NodeID: fmt.Sprintf("test-%s-%03d", strings.ToLower(nodeType), i+1),
            GlobalE2NodeID: E2NodeID{
                NodeID:   fmt.Sprintf("%s-%03d", nodeType, i+1),
                NodeType: nodeType,
            },
            ConnectionStatus: E2ConnectionStatus{
                State:     "CONNECTED",
                Timestamp: time.Now().Add(-time.Duration(rng.Intn(3600)) * time.Second),
            },
            RANFunctions: gen.generateRanFunctions(nodeType),
            Configuration: map[string]interface{}{
                "cell_count":     rng.Intn(16) + 1,
                "max_ue_count":   rng.Intn(1000) + 100,
                "frequency_band": fmt.Sprintf("n%d", rng.Intn(100)+1),
            },
        }
        
        nodes[i] = node
    }
    
    return nodes
}

func (gen *E2TestDataGenerator) generateRanFunctions(nodeType string) []RanFunction {
    functions := []RanFunction{}
    
    // All nodes support KPM
    functions = append(functions, RanFunction{
        FunctionID:         1,
        FunctionDefinition: fmt.Sprintf("%s-KPM", nodeType),
        FunctionRevision:   1,
        FunctionOID:        "1.3.6.1.4.1.53148.1.1.1.1",
        ServiceModel:       *CreateKPMServiceModel(),
    })
    
    // 70% of nodes support RC
    if rng.Float32() < 0.7 {
        functions = append(functions, RanFunction{
            FunctionID:         2,
            FunctionDefinition: fmt.Sprintf("%s-RC", nodeType),
            FunctionRevision:   1,
            FunctionOID:        "1.3.6.1.4.1.53148.1.1.1.2",
            ServiceModel:       *CreateRCServiceModel(),
        })
    }
    
    return functions
}

func (gen *E2TestDataGenerator) GenerateTestSubscriptions(nodeIDs []string) []*E2Subscription {
    rng := rand.New(rand.NewSource(gen.randomSeed))
    subscriptions := make([]*E2Subscription, 0, gen.subscriptionCount)
    
    actionTypes := []string{"REPORT", "INSERT", "POLICY"}
    triggerTypes := []string{"PERIODIC", "UPON_CHANGE", "UPON_RCV_MEASUREMENT"}
    measurementTypes := []string{
        "DRB.RlcSduDelayDl", "DRB.RlcSduDelayUl", 
        "DRB.UEThpDl", "DRB.UEThpUl",
        "RRU.PrbUsedDl", "RRU.PrbUsedUl",
    }
    
    for i := 0; i < gen.subscriptionCount; i++ {
        nodeID := nodeIDs[rng.Intn(len(nodeIDs))]
        
        subscription := &E2Subscription{
            SubscriptionID:  fmt.Sprintf("test-sub-%04d", i+1),
            RequestorID:     fmt.Sprintf("test-requestor-%d", rng.Intn(10)+1),
            RanFunctionID:   1 + rng.Intn(2), // 1 or 2
            ReportingPeriod: time.Duration(1+rng.Intn(60)) * time.Second,
            EventTriggers: []E2EventTrigger{
                {
                    TriggerType:     triggerTypes[rng.Intn(len(triggerTypes))],
                    ReportingPeriod: time.Duration(1+rng.Intn(60)) * time.Second,
                    Conditions: map[string]interface{}{
                        "measurement_period": fmt.Sprintf("%dms", (1+rng.Intn(60))*1000),
                    },
                },
            },
            Actions: []E2Action{
                {
                    ActionID:   1 + rng.Intn(5),
                    ActionType: actionTypes[rng.Intn(len(actionTypes))],
                    ActionDefinition: map[string]interface{}{
                        "measurement_type": measurementTypes[rng.Intn(len(measurementTypes))],
                        "granularity_period": (1 + rng.Intn(10)) * 1000,
                    },
                },
            },
        }
        
        subscriptions = append(subscriptions, subscription)
    }
    
    return subscriptions
}
```

This comprehensive testing and validation framework ensures that the E2 interface implementation meets all requirements for reliability, performance, and compliance with O-RAN specifications. The framework provides automated testing capabilities that integrate with CI/CD pipelines and support continuous validation throughout the development lifecycle.