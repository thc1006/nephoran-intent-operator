package o1

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientImpl_GetConfig(t *testing.T) {
	client := &ClientImpl{
		baseURL: "http://localhost:8080",
		timeout: 30 * time.Second,
	}

	t.Run("returns config response", func(t *testing.T) {
		ctx := context.Background()
		path := "/managed-element/1/config"

		resp, err := client.GetConfig(ctx, path)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, path, resp.ObjectInstance)
		assert.Equal(t, "success", resp.Status)
		assert.NotNil(t, resp.Attributes)
		assert.False(t, resp.Timestamp.IsZero())
	})

	t.Run("handles different paths", func(t *testing.T) {
		ctx := context.Background()
		paths := []string{
			"/managed-element/1",
			"/managed-element/1/gnb-du",
			"/managed-element/1/gnb-du/cell-1",
		}

		for _, path := range paths {
			resp, err := client.GetConfig(ctx, path)
			require.NoError(t, err)
			assert.Equal(t, path, resp.ObjectInstance)
		}
	})

	t.Run("returns valid JSON attributes", func(t *testing.T) {
		ctx := context.Background()

		resp, err := client.GetConfig(ctx, "/test/path")
		require.NoError(t, err)

		// Verify attributes are valid JSON
		var attrs map[string]interface{}
		err = json.Unmarshal(resp.Attributes, &attrs)
		require.NoError(t, err)
	})
}

func TestClientImpl_SetConfig(t *testing.T) {
	client := &ClientImpl{
		baseURL: "http://localhost:8080",
		timeout: 30 * time.Second,
	}

	t.Run("sets config successfully", func(t *testing.T) {
		ctx := context.Background()

		attrs := map[string]interface{}{
			"parameter1": "value1",
			"parameter2": 123,
		}
		attrsJSON, err := json.Marshal(attrs)
		require.NoError(t, err)

		configReq := &ConfigRequest{
			ObjectInstance: "/managed-element/1/config",
			Attributes:     json.RawMessage(attrsJSON),
		}

		resp, err := client.SetConfig(ctx, configReq)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, configReq.ObjectInstance, resp.ObjectInstance)
		assert.Equal(t, "success", resp.Status)
		assert.Equal(t, "Configuration updated successfully", resp.ErrorMessage)
		assert.Equal(t, configReq.Attributes, resp.Attributes)
		assert.False(t, resp.Timestamp.IsZero())
	})

	t.Run("handles complex attributes", func(t *testing.T) {
		ctx := context.Background()

		attrs := map[string]interface{}{
			"cellConfig": map[string]interface{}{
				"cellId":   "cell-001",
				"pci":      123,
				"tracking": "area-1",
			},
			"resources": map[string]interface{}{
				"cpu":    "2000m",
				"memory": "4Gi",
			},
		}
		attrsJSON, err := json.Marshal(attrs)
		require.NoError(t, err)

		configReq := &ConfigRequest{
			ObjectInstance: "/managed-element/1/gnb",
			Attributes:     json.RawMessage(attrsJSON),
		}

		resp, err := client.SetConfig(ctx, configReq)
		require.NoError(t, err)
		assert.Equal(t, "success", resp.Status)
	})
}

func TestClientImpl_GetPerformanceData(t *testing.T) {
	client := &ClientImpl{
		baseURL: "http://localhost:8080",
		timeout: 30 * time.Second,
	}

	t.Run("returns performance data", func(t *testing.T) {
		ctx := context.Background()

		perfReq := &PerformanceRequest{
			ID:         "req-001",
			MetricType: "RRCConnections",
			Target:     "/managed-element/1",
		}

		resp, err := client.GetPerformanceData(ctx, perfReq)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, "success", resp.Status)
		assert.NotEmpty(t, resp.RequestID)
		assert.Len(t, resp.PerformanceData, 2)

		// Verify first data point
		assert.Equal(t, "perf_data_1", resp.PerformanceData[0].ID)
		assert.Equal(t, "cell_001", resp.PerformanceData[0].Source)
		assert.Equal(t, "RRCConnections", resp.PerformanceData[0].DataType)
		assert.NotNil(t, resp.PerformanceData[0].Metrics)

		// Verify second data point
		assert.Equal(t, "perf_data_2", resp.PerformanceData[1].ID)
		assert.Equal(t, "Throughput", resp.PerformanceData[1].DataType)
	})

	t.Run("generates unique request IDs", func(t *testing.T) {
		ctx := context.Background()

		perfReq := &PerformanceRequest{
			ID:         "req-002",
			MetricType: "Throughput",
			Target:     "/managed-element/1",
		}

		resp1, err := client.GetPerformanceData(ctx, perfReq)
		require.NoError(t, err)

		// Delay to ensure different timestamp (IDs use Unix seconds)
		time.Sleep(1100 * time.Millisecond)

		resp2, err := client.GetPerformanceData(ctx, perfReq)
		require.NoError(t, err)

		// Request IDs should be different (timestamp-based)
		assert.NotEqual(t, resp1.RequestID, resp2.RequestID)
	})
}

func TestClientImpl_SubscribePerformanceData(t *testing.T) {
	client := &ClientImpl{
		baseURL: "http://localhost:8080",
		timeout: 30 * time.Second,
	}

	t.Run("creates subscription channel", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		subscription := &PerformanceSubscription{
			ID:              "sub-001",
			MetricType:      "RRCConnections",
			Callback:        "http://callback/endpoint",
			ReportingPeriod: 100 * time.Millisecond,
			Status:          "active",
			CreatedAt:       time.Now(),
		}

		ch, err := client.SubscribePerformanceData(ctx, subscription)
		require.NoError(t, err)
		require.NotNil(t, ch)
	})

	t.Run("subscription channel is buffered", func(t *testing.T) {
		ctx := context.Background()

		subscription := &PerformanceSubscription{
			ID:              "sub-002",
			MetricType:      "Throughput",
			Callback:        "http://callback/endpoint",
			ReportingPeriod: 100 * time.Millisecond,
			Status:          "active",
			CreatedAt:       time.Now(),
		}

		ch, err := client.SubscribePerformanceData(ctx, subscription)
		require.NoError(t, err)
		require.NotNil(t, ch)

		// Channel should be buffered (size 100)
		assert.Equal(t, 100, cap(ch))
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		subscription := &PerformanceSubscription{
			ID:              "sub-003",
			MetricType:      "Latency",
			Callback:        "http://callback/endpoint",
			ReportingPeriod: 50 * time.Millisecond,
			Status:          "active",
			CreatedAt:       time.Now(),
		}

		ch, err := client.SubscribePerformanceData(ctx, subscription)
		require.NoError(t, err)

		// Cancel context immediately
		cancel()

		// Channel should eventually close
		timeout := time.After(200 * time.Millisecond)
		select {
		case _, ok := <-ch:
			if !ok {
				// Channel closed as expected
				return
			}
		case <-timeout:
			t.Fatal("channel did not close after context cancellation")
		}
	})
}

func TestConfigResponse_Marshaling(t *testing.T) {
	t.Run("marshals to valid JSON", func(t *testing.T) {
		attrs := map[string]interface{}{
			"key": "value",
		}
		attrsJSON, _ := json.Marshal(attrs)

		resp := &ConfigResponse{
			ID:             "cfg-001",
			RequestID:      "req-001",
			Status:         "success",
			ObjectInstance: "/test/path",
			Attributes:     json.RawMessage(attrsJSON),
			Timestamp:      time.Now(),
			ProcessedAt:    time.Now(),
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		// Unmarshal to verify structure
		var unmarshaled map[string]interface{}
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		// Note: JSON tags use snake_case
		assert.Equal(t, "/test/path", unmarshaled["object_instance"])
		assert.Equal(t, "success", unmarshaled["status"])
		assert.Equal(t, "cfg-001", unmarshaled["id"])
	})
}

func TestPerformanceData_Structure(t *testing.T) {
	t.Run("contains required fields", func(t *testing.T) {
		metrics := map[string]interface{}{
			"value": 100,
			"unit":  "connections",
		}
		metricsJSON, _ := json.Marshal(metrics)

		perfData := PerformanceData{
			ID:        "perf_001",
			Timestamp: time.Now(),
			Metrics:   json.RawMessage(metricsJSON),
			Source:    "cell_001",
			DataType:  "RRCConnections",
		}

		// Marshal and unmarshal to verify
		data, err := json.Marshal(perfData)
		require.NoError(t, err)

		var unmarshaled PerformanceData
		err = json.Unmarshal(data, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, perfData.ID, unmarshaled.ID)
		assert.Equal(t, perfData.Source, unmarshaled.Source)
		assert.Equal(t, perfData.DataType, unmarshaled.DataType)
	})
}

func TestClientImpl_Initialization(t *testing.T) {
	t.Run("creates client with baseURL and timeout", func(t *testing.T) {
		client := &ClientImpl{
			baseURL: "https://oran-o1.example.com",
			timeout: 45 * time.Second,
		}

		assert.Equal(t, "https://oran-o1.example.com", client.baseURL)
		assert.Equal(t, 45*time.Second, client.timeout)
	})

	t.Run("client fields are accessible", func(t *testing.T) {
		client := &ClientImpl{}

		// Should be able to set fields
		client.baseURL = "http://test"
		client.timeout = 10 * time.Second

		assert.NotEmpty(t, client.baseURL)
		assert.Greater(t, client.timeout, time.Duration(0))
	})
}

func TestPerformanceRequest_Validation(t *testing.T) {
	t.Run("creates valid request", func(t *testing.T) {
		req := &PerformanceRequest{
			ID:         "req-valid",
			MetricType: "RRCConnections",
			Target:     "/managed-element/1",
		}

		assert.NotEmpty(t, req.ID)
		assert.NotEmpty(t, req.MetricType)
		assert.NotEmpty(t, req.Target)
	})
}

func TestConfigRequest_ComplexScenarios(t *testing.T) {
	t.Run("handles nested configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"network": map[string]interface{}{
				"mcc": "001",
				"mnc": "01",
				"tac": "0001",
			},
			"slice": map[string]interface{}{
				"sst": 1,
				"sd":  "000001",
			},
		}

		configJSON, err := json.Marshal(config)
		require.NoError(t, err)

		req := &ConfigRequest{
			ObjectInstance: "/managed-element/1/plmn",
			Attributes:     json.RawMessage(configJSON),
		}

		assert.NotEmpty(t, req.ObjectInstance)
		assert.NotNil(t, req.Attributes)

		// Verify can unmarshal back
		var unmarshaled map[string]interface{}
		err = json.Unmarshal(req.Attributes, &unmarshaled)
		require.NoError(t, err)
		assert.Contains(t, unmarshaled, "network")
		assert.Contains(t, unmarshaled, "slice")
	})
}
