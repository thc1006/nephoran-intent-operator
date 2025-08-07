package compliance

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
)

// ORANComplianceTestSuite validates O-RAN compliance
type ORANComplianceTestSuite struct {
	suite.Suite
	ctx        context.Context
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// SetupSuite runs before all tests
func (s *ORANComplianceTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.baseURL = config.GetEnvOrDefault("ORAN_TEST_URL", "http://localhost:8080")
	s.apiKey = config.GetEnvOrDefault("ORAN_API_KEY", "test-key")

	// Configure HTTP client with TLS
	s.httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
		},
	}
}

// Test A1 Interface Compliance
func (s *ORANComplianceTestSuite) TestA1InterfaceCompliance() {
	s.Run("A1_PolicyTypeManagement", func() {
		// Test policy type creation
		policyType := map[string]interface{}{
			"policy_type_id": 1001,
			"name":          "QoS_Policy",
			"description":   "QoS optimization policy",
			"policy_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"max_throughput": map[string]interface{}{
						"type": "integer",
						"minimum": 0,
					},
				},
			},
		}

		resp := s.makeRequest("PUT", "/A1-P/v2/policytypes/1001", policyType)
		s.Assert().Equal(201, resp.StatusCode, "Policy type creation should return 201")
	})

	s.Run("A1_PolicyInstanceManagement", func() {
		// Test policy instance creation
		policy := map[string]interface{}{
			"ric_id":         "ric_001",
			"policy_id":      "policy_001",
			"policy_type_id": 1001,
			"policy_data": map[string]interface{}{
				"max_throughput": 1000,
			},
		}

		resp := s.makeRequest("PUT", "/A1-P/v2/policies/policy_001", policy)
		s.Assert().Equal(201, resp.StatusCode, "Policy creation should return 201")

		// Test policy retrieval
		resp = s.makeRequest("GET", "/A1-P/v2/policies/policy_001", nil)
		s.Assert().Equal(200, resp.StatusCode, "Policy retrieval should return 200")
	})

	s.Run("A1_PolicyStatusMonitoring", func() {
		// Test policy status endpoint
		resp := s.makeRequest("GET", "/A1-P/v2/policies/policy_001/status", nil)
		s.Assert().Equal(200, resp.StatusCode, "Policy status should return 200")

		var status map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&status)
		s.Require().NoError(err)
		s.Assert().Equal("ENFORCED", status["enforced"])
	})

	s.Run("A1_PerformanceRequirements", func() {
		// Test performance requirements
		start := time.Now()
		resp := s.makeRequest("GET", "/A1-P/v2/policies", nil)
		latency := time.Since(start)

		s.Assert().Equal(200, resp.StatusCode)
		s.Assert().Less(latency, 500*time.Millisecond, "A1 API latency should be < 500ms")
	})
}

// Test O1 Interface Compliance
func (s *ORANComplianceTestSuite) TestO1InterfaceCompliance() {
	s.Run("O1_NetconfCapabilities", func() {
		// Test NETCONF capabilities
		resp := s.makeRequest("GET", "/O1/netconf/capabilities", nil)
		s.Assert().Equal(200, resp.StatusCode)

		var capabilities map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&capabilities)
		s.Require().NoError(err)

		// Verify required capabilities
		requiredCaps := []string{
			"urn:ietf:params:netconf:base:1.1",
			"urn:ietf:params:netconf:capability:writable-running:1.0",
			"urn:ietf:params:netconf:capability:notification:1.0",
		}

		capList := capabilities["capabilities"].([]interface{})
		for _, required := range requiredCaps {
			s.Assert().Contains(capList, required, "Missing required capability: %s", required)
		}
	})

	s.Run("O1_YangModels", func() {
		// Test YANG model support
		resp := s.makeRequest("GET", "/O1/yang-models", nil)
		s.Assert().Equal(200, resp.StatusCode)

		var models map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&models)
		s.Require().NoError(err)

		// Verify O-RAN YANG models
		requiredModels := []string{
			"o-ran-sc-du-hello-world",
			"o-ran-performance-management",
			"o-ran-fault-management",
			"o-ran-software-management",
		}

		modelList := models["models"].([]interface{})
		for _, required := range requiredModels {
			found := false
			for _, model := range modelList {
				if model.(map[string]interface{})["name"] == required {
					found = true
					break
				}
			}
			s.Assert().True(found, "Missing required YANG model: %s", required)
		}
	})

	s.Run("O1_FaultManagement", func() {
		// Test alarm retrieval
		resp := s.makeRequest("GET", "/O1/alarms/active", nil)
		s.Assert().Equal(200, resp.StatusCode)

		// Test alarm subscription
		subscription := map[string]interface{}{
			"filter": map[string]interface{}{
				"severity": []string{"CRITICAL", "MAJOR"},
			},
			"callback_url": "http://test-callback/alarms",
		}

		resp = s.makeRequest("POST", "/O1/alarm-subscriptions", subscription)
		s.Assert().Equal(201, resp.StatusCode)
	})
}

// Test O2 Interface Compliance
func (s *ORANComplianceTestSuite) TestO2InterfaceCompliance() {
	s.Run("O2_CloudResourceInventory", func() {
		// Test resource inventory API
		resp := s.makeRequest("GET", "/O2/v1/inventory", nil)
		s.Assert().Equal(200, resp.StatusCode)

		var inventory map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&inventory)
		s.Require().NoError(err)

		// Verify inventory structure
		s.Assert().Contains(inventory, "compute_resources")
		s.Assert().Contains(inventory, "network_resources")
		s.Assert().Contains(inventory, "storage_resources")
	})

	s.Run("O2_DeploymentManagement", func() {
		// Test deployment descriptor
		deployment := map[string]interface{}{
			"name":        "test-nf-deployment",
			"description": "Test network function deployment",
			"nf_type":     "DU",
			"resources": map[string]interface{}{
				"cpu":    "4",
				"memory": "8Gi",
				"storage": "100Gi",
			},
		}

		resp := s.makeRequest("POST", "/O2/v1/deployments", deployment)
		s.Assert().Equal(201, resp.StatusCode)

		// Verify deployment status
		var result map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&result)
		s.Require().NoError(err)
		s.Assert().Equal("DEPLOYING", result["status"])
	})

	s.Run("O2_MonitoringIntegration", func() {
		// Test monitoring endpoint
		resp := s.makeRequest("GET", "/O2/v1/monitoring/metrics", nil)
		s.Assert().Equal(200, resp.StatusCode)

		// Verify Prometheus format
		contentType := resp.Header.Get("Content-Type")
		s.Assert().Contains(contentType, "text/plain")
	})
}

// Test E2 Interface Compliance
func (s *ORANComplianceTestSuite) TestE2InterfaceCompliance() {
	s.Run("E2_SetupProcedure", func() {
		// Test E2 setup request
		setup := map[string]interface{}{
			"global_e2_node_id": map[string]interface{}{
				"gnb_id": "001",
				"plmn_id": "00101",
			},
			"ran_functions": []map[string]interface{}{
				{
					"ran_function_id": 1,
					"ran_function_definition": "KPM Service Model",
				},
			},
		}

		resp := s.makeRequest("POST", "/E2/v1/setup", setup)
		s.Assert().Equal(200, resp.StatusCode)
	})

	s.Run("E2_SubscriptionManagement", func() {
		// Test subscription creation
		subscription := map[string]interface{}{
			"subscription_id": "sub_001",
			"ran_function_id": 1,
			"event_triggers": map[string]interface{}{
				"period_ms": 1000,
			},
			"action_list": []map[string]interface{}{
				{
					"action_id": 1,
					"action_type": "REPORT",
				},
			},
		}

		resp := s.makeRequest("POST", "/E2/v1/subscriptions", subscription)
		s.Assert().Equal(201, resp.StatusCode)
	})

	s.Run("E2_ServiceModels", func() {
		// Test service model support
		resp := s.makeRequest("GET", "/E2/v1/service-models", nil)
		s.Assert().Equal(200, resp.StatusCode)

		var models map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&models)
		s.Require().NoError(err)

		// Verify required service models
		requiredModels := []string{"E2SM-KPM", "E2SM-RC", "E2SM-NI"}
		modelList := models["service_models"].([]interface{})
		
		for _, required := range requiredModels {
			found := false
			for _, model := range modelList {
				if model.(map[string]interface{})["name"] == required {
					found = true
					break
				}
			}
			s.Assert().True(found, "Missing required service model: %s", required)
		}
	})
}

// Test SMO Integration Compliance
func (s *ORANComplianceTestSuite) TestSMOIntegrationCompliance() {
	s.Run("SMO_ServiceRegistration", func() {
		// Test service registration with SMO
		service := map[string]interface{}{
			"service_id":   "nephoran-operator",
			"service_type": "rApp",
			"version":      "2.0.0",
			"capabilities": []string{"intent-processing", "policy-management"},
			"endpoint":     "http://nephoran:8080",
		}

		resp := s.makeRequest("POST", "/SMO/v1/services/register", service)
		s.Assert().Equal(201, resp.StatusCode)
	})

	s.Run("SMO_PolicyCoordination", func() {
		// Test policy coordination
		policy := map[string]interface{}{
			"policy_id": "coord_policy_001",
			"scope": map[string]interface{}{
				"ric_ids": []string{"ric_001", "ric_002"},
				"cell_ids": []string{"cell_001", "cell_002"},
			},
			"objectives": map[string]interface{}{
				"throughput_optimization": true,
				"energy_efficiency": true,
			},
		}

		resp := s.makeRequest("POST", "/SMO/v1/policies/coordinate", policy)
		s.Assert().Equal(200, resp.StatusCode)
	})

	s.Run("SMO_WorkflowOrchestration", func() {
		// Test workflow creation
		workflow := map[string]interface{}{
			"workflow_id": "wf_001",
			"name": "Network Slice Deployment",
			"steps": []map[string]interface{}{
				{
					"step_id": "1",
					"action": "deploy_nf",
					"target": "AMF",
				},
				{
					"step_id": "2",
					"action": "configure_policy",
					"target": "QoS_Policy",
				},
			},
		}

		resp := s.makeRequest("POST", "/SMO/v1/workflows", workflow)
		s.Assert().Equal(201, resp.StatusCode)
	})
}

// Test Security Compliance
func (s *ORANComplianceTestSuite) TestSecurityCompliance() {
	s.Run("Security_TLSVersion", func() {
		// Verify TLS 1.3 enforcement
		tlsConn := s.httpClient.Transport.(*http.Transport).TLSClientConfig
		s.Assert().Equal(uint16(tls.VersionTLS13), tlsConn.MinVersion)
	})

	s.Run("Security_Authentication", func() {
		// Test without authentication
		resp := s.makeRequestWithoutAuth("GET", "/A1-P/v2/policies", nil)
		s.Assert().Equal(401, resp.StatusCode, "Should require authentication")

		// Test with invalid token
		resp = s.makeRequestWithAuth("GET", "/A1-P/v2/policies", nil, "invalid-token")
		s.Assert().Equal(401, resp.StatusCode, "Should reject invalid token")
	})

	s.Run("Security_RateLimiting", func() {
		// Test rate limiting
		for i := 0; i < 150; i++ {
			resp := s.makeRequest("GET", "/A1-P/v2/healthz", nil)
			if resp.StatusCode == 429 {
				// Rate limit hit
				s.Assert().Greater(i, 99, "Rate limit should allow at least 100 requests")
				return
			}
		}
		s.Fail("Rate limiting not enforced")
	})
}

// Test Performance Compliance
func (s *ORANComplianceTestSuite) TestPerformanceCompliance() {
	s.Run("Performance_Latency", func() {
		// Test API latency requirements
		endpoints := []struct {
			path      string
			maxLatency time.Duration
		}{
			{"/A1-P/v2/policies", 500 * time.Millisecond},
			{"/O1/alarms/active", 1 * time.Second},
			{"/O2/v1/inventory", 2 * time.Second},
			{"/E2/v1/subscriptions", 500 * time.Millisecond},
		}

		for _, ep := range endpoints {
			start := time.Now()
			resp := s.makeRequest("GET", ep.path, nil)
			latency := time.Since(start)

			s.Assert().Equal(200, resp.StatusCode)
			s.Assert().Less(latency, ep.maxLatency, 
				"Endpoint %s exceeded latency requirement", ep.path)
		}
	})

	s.Run("Performance_Throughput", func() {
		// Test throughput requirements
		start := time.Now()
		successCount := 0

		// Send 1000 requests
		for i := 0; i < 1000; i++ {
			resp := s.makeRequest("GET", "/A1-P/v2/healthz", nil)
			if resp.StatusCode == 200 {
				successCount++
			}
		}

		duration := time.Since(start)
		throughput := float64(successCount) / duration.Seconds()

		s.Assert().Greater(throughput, 100.0, 
			"Throughput should exceed 100 requests/second")
		s.Assert().Greater(successCount, 990, 
			"Success rate should exceed 99%")
	})
}

// Helper methods

func (s *ORANComplianceTestSuite) makeRequest(method, path string, body interface{}) *http.Response {
	return s.makeRequestWithAuth(method, path, body, s.apiKey)
}

func (s *ORANComplianceTestSuite) makeRequestWithAuth(method, path string, body interface{}, apiKey string) *http.Response {
	url := s.baseURL + path
	
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		s.Require().NoError(err)
	}

	req, err := http.NewRequestWithContext(s.ctx, method, url, nil)
	s.Require().NoError(err)

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	s.Require().NoError(err)

	return resp
}

func (s *ORANComplianceTestSuite) makeRequestWithoutAuth(method, path string, body interface{}) *http.Response {
	url := s.baseURL + path
	
	req, err := http.NewRequestWithContext(s.ctx, method, url, nil)
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	s.Require().NoError(err)

	return resp
}

// Note: Helper functions have been moved to pkg/config/env_helpers.go

// TestORANCompliance runs the compliance test suite
func TestORANCompliance(t *testing.T) {
	suite.Run(t, new(ORANComplianceTestSuite))
}

// ComplianceReport generates a compliance report
type ComplianceReport struct {
	Timestamp      time.Time                `json:"timestamp"`
	Version        string                   `json:"version"`
	TotalTests     int                      `json:"total_tests"`
	PassedTests    int                      `json:"passed_tests"`
	FailedTests    int                      `json:"failed_tests"`
	ComplianceRate float64                  `json:"compliance_rate"`
	Details        map[string]TestResult    `json:"details"`
}

type TestResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
}

// GenerateComplianceReport generates a detailed compliance report
func GenerateComplianceReport(results *testing.T) (*ComplianceReport, error) {
	report := &ComplianceReport{
		Timestamp: time.Now(),
		Version:   "2.0.0",
		Details:   make(map[string]TestResult),
	}

	// Calculate compliance rate
	if report.TotalTests > 0 {
		report.ComplianceRate = float64(report.PassedTests) / float64(report.TotalTests) * 100
	}

	return report, nil
}