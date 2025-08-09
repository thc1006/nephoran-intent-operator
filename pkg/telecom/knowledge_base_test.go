package telecom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTelecomKnowledgeBase(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	assert.True(t, kb.IsInitialized())
	assert.Equal(t, "1.0.0-3GPP-R17", kb.GetVersion())
	assert.NotEmpty(t, kb.NetworkFunctions)
	assert.NotEmpty(t, kb.Interfaces)
	assert.NotEmpty(t, kb.QosProfiles)
	assert.NotEmpty(t, kb.SliceTypes)
}

func TestGetNetworkFunction(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Test AMF
	amf, exists := kb.GetNetworkFunction("amf")
	require.True(t, exists)
	assert.Equal(t, "AMF", amf.Name)
	assert.Equal(t, "5gc-control-plane", amf.Type)
	assert.Contains(t, amf.Interfaces, "N1")
	assert.Contains(t, amf.Interfaces, "N2")
	assert.Contains(t, amf.Dependencies, "AUSF")
	assert.Contains(t, amf.Dependencies, "UDM")

	// Test performance baseline
	assert.Equal(t, 10000, amf.Performance.MaxThroughputRPS)
	assert.Equal(t, 50.0, amf.Performance.AvgLatencyMs)
	assert.Equal(t, 100000, amf.Performance.MaxConcurrentSessions)

	// Test scaling parameters
	assert.Equal(t, 3, amf.Scaling.MinReplicas)
	assert.Equal(t, 20, amf.Scaling.MaxReplicas)
	assert.Equal(t, 70, amf.Scaling.TargetCPU)

	// Test case insensitive lookup
	amf2, exists2 := kb.GetNetworkFunction("AMF")
	assert.True(t, exists2)
	assert.Equal(t, amf, amf2)

	// Test non-existent function
	_, exists3 := kb.GetNetworkFunction("nonexistent")
	assert.False(t, exists3)
}

func TestGetNetworkFunctionsByType(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	controlPlaneFunctions := kb.GetNetworkFunctionsByType("5gc-control-plane")
	assert.NotEmpty(t, controlPlaneFunctions)

	// Verify all returned functions are control plane
	for _, nf := range controlPlaneFunctions {
		assert.Equal(t, "5gc-control-plane", nf.Type)
	}

	// Should include AMF, SMF, PCF, UDM, AUSF, NRF, NSSF
	functionNames := make(map[string]bool)
	for _, nf := range controlPlaneFunctions {
		functionNames[nf.Name] = true
	}
	assert.True(t, functionNames["AMF"])
	assert.True(t, functionNames["SMF"])
	assert.True(t, functionNames["PCF"])

	userPlaneFunctions := kb.GetNetworkFunctionsByType("5gc-user-plane")
	assert.NotEmpty(t, userPlaneFunctions)
	assert.Equal(t, "UPF", userPlaneFunctions[0].Name)
}

func TestGetInterfacesForFunction(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Test AMF interfaces
	interfaces := kb.GetInterfacesForFunction("amf")
	assert.NotEmpty(t, interfaces)

	// Should include N1 and N2 interfaces
	interfaceNames := make(map[string]bool)
	for _, iface := range interfaces {
		interfaceNames[iface.Name] = true
	}
	assert.True(t, interfaceNames["N1"])
	assert.True(t, interfaceNames["N2"])

	// Test non-existent function
	interfaces2 := kb.GetInterfacesForFunction("nonexistent")
	assert.Nil(t, interfaces2)
}

func TestGetDependenciesForFunction(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	required, optional := kb.GetDependenciesForFunction("amf")

	// AMF should have required dependencies
	assert.NotEmpty(t, required)
	depNames := make(map[string]bool)
	for _, dep := range required {
		depNames[dep.Name] = true
	}
	assert.True(t, depNames["AUSF"])
	assert.True(t, depNames["UDM"])

	// AMF should have optional dependencies
	assert.NotEmpty(t, optional)
	optDepNames := make(map[string]bool)
	for _, dep := range optional {
		optDepNames[dep.Name] = true
	}
	assert.True(t, optDepNames["NSSF"])
}

func TestGetSliceType(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Test eMBB slice
	embb, exists := kb.GetSliceType("embb")
	require.True(t, exists)
	assert.Equal(t, 1, embb.SST)
	assert.Equal(t, "Enhanced Mobile Broadband", embb.Description)
	assert.Equal(t, 10.0, embb.Requirements.Latency.UserPlane)
	assert.Equal(t, "1000", embb.Requirements.Throughput.Typical)
	assert.Equal(t, 99.9, embb.Requirements.Reliability.Availability)

	// Test URLLC slice
	urllc, exists2 := kb.GetSliceType("urllc")
	require.True(t, exists2)
	assert.Equal(t, 2, urllc.SST)
	assert.Equal(t, 1.0, urllc.Requirements.Latency.UserPlane)
	assert.Equal(t, 99.999, urllc.Requirements.Reliability.Availability)
	assert.True(t, urllc.ResourceProfile.Network.LatencySensitive)
	assert.True(t, urllc.ResourceProfile.Network.EdgePlacement)

	// Test mMTC slice
	mmtc, exists3 := kb.GetSliceType("mmtc")
	require.True(t, exists3)
	assert.Equal(t, 3, mmtc.SST)
	assert.Equal(t, 1000.0, mmtc.Requirements.Latency.UserPlane)
	assert.Equal(t, 1000000, mmtc.Requirements.Density.ConnectedDevices)
}

func TestGetQosProfile(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Test 5QI 1 (Conversational Voice)
	qos1, exists := kb.GetQosProfile("5qi_1")
	require.True(t, exists)
	assert.Equal(t, 1, qos1.QCI)
	assert.Equal(t, "GBR", qos1.Resource)
	assert.Equal(t, 2, qos1.Priority)
	assert.Equal(t, 100, qos1.DelayBudget)

	// Test 5QI 82 (URLLC)
	qos82, exists2 := kb.GetQosProfile("5qi_82")
	require.True(t, exists2)
	assert.Equal(t, 82, qos82.QCI)
	assert.Equal(t, 1, qos82.Priority)
	assert.Equal(t, 10, qos82.DelayBudget)
}

func TestValidateSliceConfiguration(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Valid configuration
	issues := kb.ValidateSliceConfiguration("embb", "5qi_9", []string{"amf", "smf", "upf"})
	assert.Empty(t, issues)

	// Invalid slice type
	issues2 := kb.ValidateSliceConfiguration("invalid", "", []string{})
	assert.NotEmpty(t, issues2)
	assert.Contains(t, issues2[0], "Unknown slice type")

	// Invalid QoS profile
	issues3 := kb.ValidateSliceConfiguration("embb", "invalid_qos", []string{})
	assert.Contains(t, issues3, "Unknown QoS profile: invalid_qos")

	// Missing required network function
	issues4 := kb.ValidateSliceConfiguration("embb", "", []string{"amf"})
	assert.NotEmpty(t, issues4)
	foundMissingSMF := false
	foundMissingUPF := false
	for _, issue := range issues4 {
		if contains(issue, "smf") || contains(issue, "SMF") {
			foundMissingSMF = true
		}
		if contains(issue, "upf") || contains(issue, "UPF") {
			foundMissingUPF = true
		}
	}
	assert.True(t, foundMissingSMF)
	assert.True(t, foundMissingUPF)
}

func TestEstimateResourceRequirements(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Test single function
	total, err := kb.EstimateResourceRequirements([]string{"amf"}, map[string]int{"amf": 1})
	assert.NoError(t, err)
	assert.Equal(t, "8", total.MaxCPU)
	assert.Equal(t, "16Gi", total.MaxMemory)
	assert.Equal(t, "100Gi", total.Storage)

	// Test multiple functions with replicas
	total2, err2 := kb.EstimateResourceRequirements(
		[]string{"amf", "smf"},
		map[string]int{"amf": 2, "smf": 3},
	)
	assert.NoError(t, err2)
	// AMF: 8*2 + SMF: 6*3 = 34 cores
	assert.Equal(t, "34", total2.MaxCPU)

	// Test unknown function
	_, err3 := kb.EstimateResourceRequirements([]string{"unknown"}, nil)
	assert.Error(t, err3)
	assert.Contains(t, err3.Error(), "unknown network function")
}

func TestGetCompatibleSliceTypes(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Look for low latency, high reliability slices (URLLC criteria)
	compatible := kb.GetCompatibleSliceTypes(2.0, 5, 99.99)
	assert.NotEmpty(t, compatible)

	// Should include URLLC
	foundURLLC := false
	for _, slice := range compatible {
		if slice.SST == 2 { // URLLC SST
			foundURLLC = true
			break
		}
	}
	assert.True(t, foundURLLC)

	// Very strict requirements should only match URLLC
	strictCompatible := kb.GetCompatibleSliceTypes(1.0, 1, 99.999)
	assert.NotEmpty(t, strictCompatible)

	// Very loose requirements should match multiple slice types
	looseCompatible := kb.GetCompatibleSliceTypes(1000.0, 1, 90.0)
	assert.Greater(t, len(looseCompatible), 1)
}

func TestGetRecommendedDeploymentConfig(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Test production deployment
	config, err := kb.GetRecommendedDeploymentConfig("amf", "production")
	assert.NoError(t, err)
	assert.Equal(t, "production", config.Name)
	assert.Equal(t, 3, config.Replicas)
	assert.True(t, config.AntiAffinity)
	assert.Equal(t, "large", config.ResourceProfile)

	// Test development deployment
	config2, err2 := kb.GetRecommendedDeploymentConfig("amf", "development")
	assert.NoError(t, err2)
	assert.Equal(t, "development", config2.Name)
	assert.Equal(t, 1, config2.Replicas)
	assert.False(t, config2.AntiAffinity)

	// Test unknown environment (should fallback to production)
	config3, err3 := kb.GetRecommendedDeploymentConfig("amf", "unknown")
	assert.NoError(t, err3)
	assert.Equal(t, "production", config3.Name)

	// Test unknown function
	_, err4 := kb.GetRecommendedDeploymentConfig("unknown", "production")
	assert.Error(t, err4)
	assert.Contains(t, err4.Error(), "unknown network function")
}

func TestGetPerformanceBaseline(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Test AMF performance baseline
	baseline, exists := kb.GetPerformanceBaseline("amf")
	require.True(t, exists)
	assert.Equal(t, 10000, baseline.MaxThroughputRPS)
	assert.Equal(t, 50.0, baseline.AvgLatencyMs)
	assert.Equal(t, 100.0, baseline.P95LatencyMs)
	assert.Equal(t, 200.0, baseline.P99LatencyMs)

	// Test UPF performance baseline (should be much higher throughput, lower latency)
	upfBaseline, exists2 := kb.GetPerformanceBaseline("upf")
	require.True(t, exists2)
	assert.Equal(t, 100000, upfBaseline.MaxThroughputRPS)
	assert.Equal(t, 10.0, upfBaseline.AvgLatencyMs)
	assert.Greater(t, upfBaseline.MaxThroughputRPS, baseline.MaxThroughputRPS)
	assert.Less(t, upfBaseline.AvgLatencyMs, baseline.AvgLatencyMs)

	// Test non-existent function
	_, exists3 := kb.GetPerformanceBaseline("nonexistent")
	assert.False(t, exists3)
}

func TestListNetworkFunctions(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	functions := kb.ListNetworkFunctions()
	assert.NotEmpty(t, functions)

	// Should include all expected core functions
	expectedFunctions := []string{"amf", "smf", "upf", "pcf", "udm", "ausf", "nrf", "nssf", "gnb"}
	for _, expected := range expectedFunctions {
		assert.Contains(t, functions, expected)
	}

	// Should be sorted
	for i := 1; i < len(functions); i++ {
		assert.True(t, functions[i-1] <= functions[i])
	}
}

func TestGetDeploymentPattern(t *testing.T) {
	kb := NewTelecomKnowledgeBase()

	// Test high availability pattern
	pattern, exists := kb.GetDeploymentPattern("high-availability")
	require.True(t, exists)
	assert.Equal(t, "high-availability", pattern.Name)
	assert.Equal(t, "multi-region", pattern.Architecture.Type)
	assert.Equal(t, "active-active", pattern.Architecture.Redundancy)
	assert.True(t, pattern.Scaling.Horizontal)
	assert.True(t, pattern.Resilience.CircuitBreaker)

	// Test edge optimized pattern
	edgePattern, exists2 := kb.GetDeploymentPattern("edge-optimized")
	require.True(t, exists2)
	assert.Equal(t, "single-zone", edgePattern.Architecture.Type)
	assert.Contains(t, edgePattern.UseCase, "urllc")

	// Test non-existent pattern
	_, exists3 := kb.GetDeploymentPattern("nonexistent")
	assert.False(t, exists3)
}

// Helper function to check if string contains substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkGetNetworkFunction(b *testing.B) {
	kb := NewTelecomKnowledgeBase()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kb.GetNetworkFunction("amf")
	}
}

func BenchmarkGetNetworkFunctionsByType(b *testing.B) {
	kb := NewTelecomKnowledgeBase()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kb.GetNetworkFunctionsByType("5gc-control-plane")
	}
}

func BenchmarkValidateSliceConfiguration(b *testing.B) {
	kb := NewTelecomKnowledgeBase()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kb.ValidateSliceConfiguration("embb", "5qi_9", []string{"amf", "smf", "upf"})
	}
}
