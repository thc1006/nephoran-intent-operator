package e2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

func TestNewE2Adaptor(t *testing.T) {
	tests := []struct {
		name   string
		config *E2AdaptorConfig
		want   *E2AdaptorConfig
	}{
		{
			name:   "default config",
			config: nil,
			want: &E2AdaptorConfig{
				RICURL:            "http://near-rt-ric:38080",
				APIVersion:        "v1",
				Timeout:           30 * time.Second,
				HeartbeatInterval: 30 * time.Second,
				MaxRetries:        3,
			},
		},
		{
			name: "custom config",
			config: &E2AdaptorConfig{
				RICURL:            "http://custom-ric:8080",
				APIVersion:        "v2",
				Timeout:           60 * time.Second,
				HeartbeatInterval: 10 * time.Second,
				MaxRetries:        5,
			},
			want: &E2AdaptorConfig{
				RICURL:            "http://custom-ric:8080",
				APIVersion:        "v2",
				Timeout:           60 * time.Second,
				HeartbeatInterval: 10 * time.Second,
				MaxRetries:        5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adaptor, err := NewE2Adaptor(tt.config)
			require.NoError(t, err)
			assert.NotNil(t, adaptor)
			assert.Equal(t, tt.want.RICURL, adaptor.ricURL)
			assert.Equal(t, tt.want.APIVersion, adaptor.apiVersion)
			assert.Equal(t, tt.want.Timeout, adaptor.timeout)
		})
	}
}

func TestE2Adaptor_RegisterE2Node(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/e2ap/v1/nodes/test-node/register")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(t, err)

	// Test data
	functions := []*E2NodeFunction{
		{
			FunctionID:          1,
			FunctionDefinition:  "gNB-DU",
			FunctionRevision:    1,
			FunctionOID:         "1.3.6.1.4.1.53148.1.1.1.1",
			FunctionDescription: "gNB Distributed Unit",
			ServiceModel:        *CreateKPMServiceModel(),
		},
	}

	ctx := context.Background()
	err = adaptor.RegisterE2Node(ctx, "test-node", functions)
	require.NoError(t, err)

	// Verify node is registered locally
	nodeInfo, err := adaptor.GetE2Node(ctx, "test-node")
	require.NoError(t, err)
	assert.Equal(t, "test-node", nodeInfo.NodeID)
	assert.Len(t, nodeInfo.RANFunctions, 1)
	assert.Equal(t, "CONNECTED", nodeInfo.ConnectionStatus.State)
}

func TestE2Adaptor_DeregisterE2Node(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Contains(t, r.URL.Path, "/e2ap/v1/nodes/test-node/deregister")

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(t, err)

	// First register a node
	functions := []*E2NodeFunction{CreateDefaultE2NodeFunction()}
	ctx := context.Background()
	err = adaptor.RegisterE2Node(ctx, "test-node", functions)
	require.NoError(t, err)

	// Then deregister it
	err = adaptor.DeregisterE2Node(ctx, "test-node")
	require.NoError(t, err)

	// Verify node is removed locally
	_, err = adaptor.GetE2Node(ctx, "test-node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "E2 node not found")
}

func TestE2Adaptor_CreateSubscription(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/e2ap/v1/nodes/test-node/subscriptions")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(t, err)

	// First register a node
	functions := []*E2NodeFunction{CreateDefaultE2NodeFunction()}
	ctx := context.Background()
	err = adaptor.RegisterE2Node(ctx, "test-node", functions)
	require.NoError(t, err)

	// Create subscription
	subscription := &E2Subscription{
		SubscriptionID:  "sub-001",
		RequestorID:     "requester-001",
		RanFunctionID:   1,
		ReportingPeriod: 1000 * time.Millisecond,
		EventTriggers: []E2EventTrigger{
			{
				TriggerType:     "PERIODIC",
				ReportingPeriod: 1000 * time.Millisecond,
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

	err = adaptor.CreateSubscription(ctx, "test-node", subscription)
	require.NoError(t, err)

	// Verify subscription is created locally
	retrievedSub, err := adaptor.GetSubscription(ctx, "test-node", "sub-001")
	require.NoError(t, err)
	assert.Equal(t, "sub-001", retrievedSub.SubscriptionID)
	assert.Equal(t, "ACTIVE", retrievedSub.Status.State)
}

func TestE2Adaptor_SendControlRequest(t *testing.T) {
	// Expected response
	expectedResponse := &E2ControlResponse{
		ResponseID:    "resp-001",
		RequestID:     "req-001",
		RanFunctionID: 1,
		Status: E2ControlStatus{
			Result: "SUCCESS",
		},
		Timestamp: time.Now(),
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/e2ap/v1/nodes/test-node/control")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(t, err)

	// Test control request
	controlRequest := &E2ControlRequest{
		RequestID:     "req-001",
		RanFunctionID: 1,
		ControlHeader: map[string]interface{}{
			"target_cell": "cell-001",
		},
		ControlMessage: map[string]interface{}{
			"qos_flow_id": 5,
			"action":      "modify_qos",
		},
		ControlAckRequest: true,
	}

	ctx := context.Background()
	response, err := adaptor.SendControlRequest(ctx, "test-node", controlRequest)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse.ResponseID, response.ResponseID)
	assert.Equal(t, expectedResponse.RequestID, response.RequestID)
	assert.Equal(t, "SUCCESS", response.Status.Result)
}

func TestE2Adaptor_ConfigureE2Interface(t *testing.T) {
	// Create mock server for registration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/e2ap/v1/nodes/test-me/register" {
			w.WriteHeader(http.StatusCreated)
		} else if r.Method == "POST" && r.URL.Path == "/e2ap/v1/nodes/test-me/subscriptions" {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(t, err)

	// Create ManagedElement with E2 configuration
	e2Config := map[string]interface{}{
		"node_id": "test-me",
		"ran_functions": []interface{}{
			map[string]interface{}{
				"function_id":          1,
				"function_definition":  "gNB-DU",
				"function_revision":    1,
				"function_oid":         "1.3.6.1.4.1.53148.1.1.1.1",
				"function_description": "gNB Distributed Unit",
				"service_model": map[string]interface{}{
					"service_model_id":      "1.3.6.1.4.1.53148.1.1.2.2",
					"service_model_name":    "KPM",
					"service_model_version": "1.0",
					"service_model_oid":     "1.3.6.1.4.1.53148.1.1.2.2",
					"supported_procedures":  []interface{}{"RIC_SUBSCRIPTION", "RIC_INDICATION"},
				},
			},
		},
		"default_subscriptions": []interface{}{
			map[string]interface{}{
				"subscription_id":     "default-sub-001",
				"requestor_id":        "nephoran-controller",
				"ran_function_id":     1.0,
				"reporting_period_ms": 1000.0,
				"event_triggers": []interface{}{
					map[string]interface{}{
						"trigger_type":        "PERIODIC",
						"reporting_period_ms": 1000.0,
					},
				},
				"actions": []interface{}{
					map[string]interface{}{
						"action_id":   1.0,
						"action_type": "REPORT",
						"action_definition": map[string]interface{}{
							"measurement_type": "DRB.RlcSduDelayDl",
						},
					},
				},
			},
		},
	}

	e2ConfigRaw, err := json.Marshal(e2Config)
	require.NoError(t, err)

	me := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-me",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			E2Configuration: runtime.RawExtension{
				Raw: e2ConfigRaw,
			},
		},
	}

	ctx := context.Background()
	err = adaptor.ConfigureE2Interface(ctx, me)
	require.NoError(t, err)

	// Verify node is registered
	nodeInfo, err := adaptor.GetE2Node(ctx, "test-me")
	require.NoError(t, err)
	assert.Equal(t, "test-me", nodeInfo.NodeID)
	assert.Len(t, nodeInfo.RANFunctions, 1)

	// Verify subscription is created
	subscriptions, err := adaptor.ListSubscriptions(ctx, "test-me")
	require.NoError(t, err)
	assert.Len(t, subscriptions, 1)
	assert.Equal(t, "default-sub-001", subscriptions[0].SubscriptionID)
}

func TestE2Adaptor_RemoveE2Interface(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/e2ap/v1/nodes/test-me/register" {
			w.WriteHeader(http.StatusCreated)
		} else if r.Method == "DELETE" && r.URL.Path == "/e2ap/v1/nodes/test-me/deregister" {
			w.WriteHeader(http.StatusNoContent)
		} else if r.Method == "DELETE" && r.URL.Path == "/e2ap/v1/nodes/test-me/subscriptions/test-sub" {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(t, err)

	// First configure E2 interface
	e2Config := map[string]interface{}{
		"node_id": "test-me",
		"ran_functions": []interface{}{
			map[string]interface{}{
				"function_id":          1,
				"function_definition":  "gNB-DU",
				"function_revision":    1,
				"function_oid":         "1.3.6.1.4.1.53148.1.1.1.1",
				"function_description": "gNB Distributed Unit",
			},
		},
	}

	e2ConfigRaw, err := json.Marshal(e2Config)
	require.NoError(t, err)

	me := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-me",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			E2Configuration: runtime.RawExtension{
				Raw: e2ConfigRaw,
			},
		},
	}

	ctx := context.Background()
	err = adaptor.ConfigureE2Interface(ctx, me)
	require.NoError(t, err)

	// Add a subscription manually for testing cleanup
	subscription := &E2Subscription{
		SubscriptionID: "test-sub",
		RequestorID:    "test-requestor",
		RanFunctionID:  1,
	}
	adaptor.mutex.Lock()
	adaptor.subscriptions["test-me"]["test-sub"] = subscription
	adaptor.mutex.Unlock()

	// Now remove E2 interface
	err = adaptor.RemoveE2Interface(ctx, me)
	require.NoError(t, err)

	// Verify node is deregistered
	_, err = adaptor.GetE2Node(ctx, "test-me")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "E2 node not found")

	// Verify subscriptions are cleaned up
	subscriptions, err := adaptor.ListSubscriptions(ctx, "test-me")
	require.NoError(t, err)
	assert.Len(t, subscriptions, 0)
}

func TestCreateKPMServiceModel(t *testing.T) {
	serviceModel := CreateKPMServiceModel()

	assert.Equal(t, "1.3.6.1.4.1.53148.1.1.2.2", serviceModel.ServiceModelID)
	assert.Equal(t, "KPM", serviceModel.ServiceModelName)
	assert.Equal(t, "1.0", serviceModel.ServiceModelVersion)
	assert.Contains(t, serviceModel.SupportedProcedures, "RIC_SUBSCRIPTION")
	assert.Contains(t, serviceModel.SupportedProcedures, "RIC_INDICATION")

	// Verify configuration contains expected measurement types
	config := serviceModel.Configuration
	assert.NotNil(t, config)

	if measurementTypes, exists := config["measurement_types"]; exists {
		types := measurementTypes.([]string)
		assert.Contains(t, types, "DRB.RlcSduDelayDl")
		assert.Contains(t, types, "DRB.UEThpDl")
	}
}

func TestCreateRCServiceModel(t *testing.T) {
	serviceModel := CreateRCServiceModel()

	assert.Equal(t, "1.3.6.1.4.1.53148.1.1.2.3", serviceModel.ServiceModelID)
	assert.Equal(t, "RC", serviceModel.ServiceModelName)
	assert.Equal(t, "1.0", serviceModel.ServiceModelVersion)
	assert.Contains(t, serviceModel.SupportedProcedures, "RIC_CONTROL_REQUEST")
	assert.Contains(t, serviceModel.SupportedProcedures, "RIC_CONTROL_ACKNOWLEDGE")

	// Verify configuration contains expected control actions
	config := serviceModel.Configuration
	assert.NotNil(t, config)

	if controlActions, exists := config["control_actions"]; exists {
		actions := controlActions.([]string)
		assert.Contains(t, actions, "QoS_flow_mapping")
		assert.Contains(t, actions, "Traffic_steering")
	}
}

func TestCreateDefaultE2NodeFunction(t *testing.T) {
	function := CreateDefaultE2NodeFunction()

	assert.Equal(t, 1, function.FunctionID)
	assert.Equal(t, "gNB-DU", function.FunctionDefinition)
	assert.Equal(t, 1, function.FunctionRevision)
	assert.Equal(t, "ACTIVE", function.Status.State)

	// Verify service model is KPM
	assert.Equal(t, "KPM", function.ServiceModel.ServiceModelName)
	assert.Equal(t, "1.3.6.1.4.1.53148.1.1.2.2", function.ServiceModel.ServiceModelID)
}

func TestE2Adaptor_ListE2Nodes(t *testing.T) {
	config := &E2AdaptorConfig{
		RICURL:     "http://mock-ric:8080",
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Initially should be empty
	nodes, err := adaptor.ListE2Nodes(ctx)
	require.NoError(t, err)
	assert.Len(t, nodes, 0)

	// Add nodes to local registry manually for testing
	adaptor.mutex.Lock()
	adaptor.nodeRegistry["node1"] = &E2NodeInfo{
		NodeID: "node1",
		GlobalE2NodeID: GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
			NodeType:     E2NodeTypegNB,
			NodeID:       "node1",
		},
		ConnectionStatus: E2ConnectionStatus{
			State: "CONNECTED",
		},
	}
	adaptor.nodeRegistry["node2"] = &E2NodeInfo{
		NodeID: "node2",
		GlobalE2NodeID: GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{MCC: "001", MNC: "01"},
			NodeType:     E2NodeTypeeNB,
			NodeID:       "node2",
		},
		ConnectionStatus: E2ConnectionStatus{
			State: "DISCONNECTED",
		},
	}
	adaptor.mutex.Unlock()

	// Now should return both nodes
	nodes, err = adaptor.ListE2Nodes(ctx)
	require.NoError(t, err)
	assert.Len(t, nodes, 2)

	// Verify node details
	nodeMap := make(map[string]*E2NodeInfo)
	for _, node := range nodes {
		nodeMap[node.NodeID] = node
	}

	assert.Equal(t, "gNB", nodeMap["node1"].GlobalE2NodeID.NodeType)
	assert.Equal(t, "CONNECTED", nodeMap["node1"].ConnectionStatus.State)
	assert.Equal(t, "eNB", nodeMap["node2"].GlobalE2NodeID.NodeType)
	assert.Equal(t, "DISCONNECTED", nodeMap["node2"].ConnectionStatus.State)
}

func TestE2Adaptor_GetServiceModel(t *testing.T) {
	// Expected service model
	expectedServiceModel := CreateKPMServiceModel()

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/e2ap/v1/service-models/")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedServiceModel)
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(t, err)

	ctx := context.Background()
	serviceModel, err := adaptor.GetServiceModel(ctx, "1.3.6.1.4.1.53148.1.1.2.2")
	require.NoError(t, err)

	assert.Equal(t, expectedServiceModel.ServiceModelID, serviceModel.ServiceModelID)
	assert.Equal(t, expectedServiceModel.ServiceModelName, serviceModel.ServiceModelName)
	assert.Equal(t, expectedServiceModel.ServiceModelVersion, serviceModel.ServiceModelVersion)
}

// Benchmark tests for performance validation

func BenchmarkE2Adaptor_RegisterE2Node(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(b, err)

	functions := []*E2NodeFunction{CreateDefaultE2NodeFunction()}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := "bench-node-" + string(rune(i))
		err := adaptor.RegisterE2Node(ctx, nodeID, functions)
		require.NoError(b, err)
	}
}

func BenchmarkE2Adaptor_CreateSubscription(b *testing.B) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &E2AdaptorConfig{
		RICURL:     server.URL,
		APIVersion: "v1",
		Timeout:    10 * time.Second,
	}

	adaptor, err := NewE2Adaptor(config)
	require.NoError(b, err)

	// Register a node first
	functions := []*E2NodeFunction{CreateDefaultE2NodeFunction()}
	ctx := context.Background()
	err = adaptor.RegisterE2Node(ctx, "bench-node", functions)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subscription := &E2Subscription{
			SubscriptionID:  "bench-sub-" + string(rune(i)),
			RequestorID:     "bench-requestor",
			RanFunctionID:   1,
			ReportingPeriod: 1000 * time.Millisecond,
			EventTriggers: []E2EventTrigger{
				{
					TriggerType:     "PERIODIC",
					ReportingPeriod: 1000 * time.Millisecond,
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

		err := adaptor.CreateSubscription(ctx, "bench-node", subscription)
		require.NoError(b, err)
	}
}
