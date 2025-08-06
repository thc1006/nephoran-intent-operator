package telecom

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContextMatcher(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	assert.True(t, cm.IsInitialized())
	assert.NotNil(t, cm.knowledgeBase)
	assert.NotEmpty(t, cm.patterns)
	assert.NotEmpty(t, cm.synonyms)
	assert.NotEmpty(t, cm.abbreviations)
}

func TestGetRelevantContextAMF(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	intent := "Deploy AMF with high availability for production environment"
	result, err := cm.GetRelevantContext(intent)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, intent, result.Intent)
	assert.NotEmpty(t, result.NetworkFunctions)
	assert.Greater(t, result.Confidence, 0.0)
	
	// Should match AMF with high confidence
	foundAMF := false
	for _, nfMatch := range result.NetworkFunctions {
		if nfMatch.Function.Name == "AMF" {
			foundAMF = true
			assert.Greater(t, nfMatch.Confidence, 0.8) // High confidence for direct match
			assert.Contains(t, nfMatch.Reason, "direct_name_match")
			break
		}
	}
	assert.True(t, foundAMF)
	
	// Should have 3GPP context
	assert.NotNil(t, result.Context3GPP)
	assert.NotEmpty(t, result.Context3GPP.Specifications)
	assert.Contains(t, result.Context3GPP.Specifications, "TS 23.501")
}

func TestGetRelevantContextSlicing(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	intent := "Create URLLC network slice for industrial automation with 1ms latency"
	result, err := cm.GetRelevantContext(intent)
	
	require.NoError(t, err)
	assert.NotEmpty(t, result.SliceTypes)
	
	// Should match URLLC slice
	foundURLLC := false
	for _, sliceMatch := range result.SliceTypes {
		if sliceMatch.SliceType.SST == 2 { // URLLC SST
			foundURLLC = true
			assert.Greater(t, sliceMatch.Confidence, 0.5)
			break
		}
	}
	assert.True(t, foundURLLC)
	
	// Should extract latency requirement
	assert.NotNil(t, result.Requirements)
	assert.NotNil(t, result.Requirements.Latency)
	assert.Equal(t, 1.0, result.Requirements.Latency.Value)
	assert.Equal(t, "ms", result.Requirements.Latency.Unit)
}

func TestGetRelevantContextInterfaces(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	intent := "Configure N2 interface between gNB and AMF with NGAP protocol"
	result, err := cm.GetRelevantContext(intent)
	
	require.NoError(t, err)
	assert.NotEmpty(t, result.Interfaces)
	
	// Should match N2 interface
	foundN2 := false
	for _, ifaceMatch := range result.Interfaces {
		if ifaceMatch.Interface.Name == "N2" {
			foundN2 = true
			assert.Greater(t, ifaceMatch.Confidence, 0.7)
			assert.Contains(t, ifaceMatch.Reason, "interface_name_match")
			break
		}
	}
	assert.True(t, foundN2)
	
	// Should also match AMF and gNB
	foundAMF := false
	foundGNB := false
	for _, nfMatch := range result.NetworkFunctions {
		if nfMatch.Function.Name == "AMF" {
			foundAMF = true
		}
		if nfMatch.Function.Name == "gNodeB" {
			foundGNB = true
		}
	}
	assert.True(t, foundAMF)
	assert.True(t, foundGNB)
}

func TestGetRelevantContextQoS(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	intent := "Setup QCI 1 for voice calls with guaranteed bit rate"
	result, err := cm.GetRelevantContext(intent)
	
	require.NoError(t, err)
	assert.NotEmpty(t, result.QosProfiles)
	
	// Should match QCI 1 profile
	foundQCI1 := false
	for _, qosMatch := range result.QosProfiles {
		if qosMatch.Profile.QCI == 1 {
			foundQCI1 = true
			assert.Greater(t, qosMatch.Confidence, 0.8)
			assert.Contains(t, qosMatch.Reason, "qci_qfi_match")
			break
		}
	}
	assert.True(t, foundQCI1)
}

func TestGetRelevantContextDeployment(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	intent := "Deploy with high availability and multi-region redundancy for production"
	result, err := cm.GetRelevantContext(intent)
	
	require.NoError(t, err)
	assert.NotEmpty(t, result.DeploymentPatterns)
	
	// Should match high availability pattern
	foundHA := false
	for _, depMatch := range result.DeploymentPatterns {
		if depMatch.Pattern.Name == "high-availability" {
			foundHA = true
			assert.Greater(t, depMatch.Confidence, 0.6)
			break
		}
	}
	assert.True(t, foundHA)
}

func TestExtractLatencyRequirement(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	testCases := []struct {
		intent   string
		expected *LatencyRequirement
	}{
		{
			intent: "Need 5ms latency for real-time applications",
			expected: &LatencyRequirement{
				Value: 5.0,
				Unit:  "ms",
				Type:  "user-plane",
			},
		},
		{
			intent: "Control plane latency should be 50ms maximum",
			expected: &LatencyRequirement{
				Value: 50.0,
				Unit:  "ms",
				Type:  "control-plane",
			},
		},
		{
			intent: "End-to-end delay budget of 100 milliseconds",
			expected: &LatencyRequirement{
				Value: 100.0,
				Unit:  "millisecond",
				Type:  "end-to-end",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.intent, func(t *testing.T) {
			result := cm.extractLatencyRequirement(strings.ToLower(tc.intent))
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expected.Value, result.Value)
				assert.Equal(t, tc.expected.Unit, result.Unit)
				assert.Equal(t, tc.expected.Type, result.Type)
			}
		})
	}
}

func TestExtractThroughputRequirement(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	testCases := []struct {
		intent   string
		expected *ThroughputRequirement
	}{
		{
			intent: "Need 1Gbps throughput for streaming",
			expected: &ThroughputRequirement{
				Value: 1.0,
				Unit:  "gbps",
				Type:  "aggregate",
			},
		},
		{
			intent: "Uplink bandwidth of 100Mbps required",
			expected: &ThroughputRequirement{
				Value: 100.0,
				Unit:  "mbps",
				Type:  "uplink",
			},
		},
		{
			intent: "Downlink speed should be 500 Mbps",
			expected: &ThroughputRequirement{
				Value: 500.0,
				Unit:  "mbps",
				Type:  "downlink",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.intent, func(t *testing.T) {
			result := cm.extractThroughputRequirement(strings.ToLower(tc.intent))
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expected.Value, result.Value)
				assert.Equal(t, tc.expected.Unit, result.Unit)
				assert.Equal(t, tc.expected.Type, result.Type)
			}
		})
	}
}

func TestExtractReliabilityRequirement(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	testCases := []struct {
		intent   string
		expected *ReliabilityRequirement
	}{
		{
			intent: "Need 99.99% availability for critical services",
			expected: &ReliabilityRequirement{
				Value: 99.99,
				Unit:  "percentage",
				Type:  "availability",
			},
		},
		{
			intent: "Packet loss should be less than 0.1%",
			expected: &ReliabilityRequirement{
				Value: 0.1,
				Unit:  "percentage",
				Type:  "packet-loss",
			},
		},
		{
			intent: "System uptime of 99.9% required",
			expected: &ReliabilityRequirement{
				Value: 99.9,
				Unit:  "percentage",
				Type:  "availability",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.intent, func(t *testing.T) {
			result := cm.extractReliabilityRequirement(strings.ToLower(tc.intent))
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expected.Value, result.Value)
				assert.Equal(t, tc.expected.Unit, result.Unit)
				assert.Equal(t, tc.expected.Type, result.Type)
			}
		})
	}
}

func TestExtractScalabilityRequirement(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	testCases := []struct {
		intent   string
		expected *ScalabilityRequirement
	}{
		{
			intent: "Scale from 3 to 10 replicas based on load",
			expected: &ScalabilityRequirement{
				MinReplicas: 3,
				MaxReplicas: 10,
				Type:        "horizontal",
			},
		},
		{
			intent: "Auto-scale up to 5 instances when needed",
			expected: &ScalabilityRequirement{
				MaxReplicas: 5,
				Type:        "auto",
			},
		},
		{
			intent: "Vertical scaling for increased CPU and memory",
			expected: &ScalabilityRequirement{
				Type: "vertical",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.intent, func(t *testing.T) {
			result := cm.extractScalabilityRequirement(strings.ToLower(tc.intent))
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				if tc.expected.MinReplicas > 0 {
					assert.Equal(t, tc.expected.MinReplicas, result.MinReplicas)
				}
				if tc.expected.MaxReplicas > 0 {
					assert.Equal(t, tc.expected.MaxReplicas, result.MaxReplicas)
				}
				assert.Equal(t, tc.expected.Type, result.Type)
			}
		})
	}
}

func TestExtractSecurityRequirement(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	testCases := []struct {
		intent   string
		expected *SecurityRequirement
	}{
		{
			intent: "Deploy with high security and encryption",
			expected: &SecurityRequirement{
				Level:    "high",
				Features: []string{"encryption"},
			},
		},
		{
			intent: "Enable TLS and OAuth2 authentication",
			expected: &SecurityRequirement{
				Level:        "basic",
				Features:     []string{"tls", "oauth"},
				Certificates: []string{"tls"},
			},
		},
		{
			intent: "Enhanced security with mTLS and RBAC",
			expected: &SecurityRequirement{
				Level:        "enhanced",
				Features:     []string{"mtls", "rbac"},
				Certificates: []string{"mtls"},
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.intent, func(t *testing.T) {
			result := cm.extractSecurityRequirement(strings.ToLower(tc.intent))
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expected.Level, result.Level)
				
				// Check that expected features are present
				for _, expectedFeature := range tc.expected.Features {
					assert.Contains(t, result.Features, expectedFeature)
				}
				
				// Check certificates if expected
				if len(tc.expected.Certificates) > 0 {
					for _, expectedCert := range tc.expected.Certificates {
						assert.Contains(t, result.Certificates, expectedCert)
					}
				}
			}
		})
	}
}

func TestNormalizeIntent(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "Deploy 5GC network functions",
			expected: "deploy 5g core network functions",
		},
		{
			input:    "Setup   AMF   with   high    availability",
			expected: "setup amf with high availability",
		},
		{
			input:    "Configure RAN optimization",
			expected: "configure radio access network optimization",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := cm.normalizeIntent(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMatchUseCasePatterns(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	// Get AMF function for testing
	amf, _ := kb.GetNetworkFunction("amf")
	
	testCases := []struct {
		intent              string
		expectedConfidence  float64
	}{
		{
			intent:             "setup user registration and authentication",
			expectedConfidence: 0.4, // Should match "registration" and "authentication"
		},
		{
			intent:             "configure mobility management",
			expectedConfidence: 0.2, // Should match "mobility"
		},
		{
			intent:             "unrelated network task",
			expectedConfidence: 0.0, // No matches
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.intent, func(t *testing.T) {
			result := cm.matchUseCasePatterns(tc.intent, amf)
			assert.Equal(t, tc.expectedConfidence, result)
		})
	}
}

func TestBuild3GPPContext(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	// Create a mock result with AMF match
	amf, _ := kb.GetNetworkFunction("amf")
	n2, _ := kb.GetInterface("n2")
	
	result := &MatchResult{
		NetworkFunctions: []*NetworkFunctionMatch{
			{Function: amf, Confidence: 0.9},
		},
		Interfaces: []*InterfaceMatch{
			{Interface: n2, Confidence: 0.8},
		},
	}
	
	context := cm.build3GPPContext(result)
	
	assert.NotNil(t, context)
	assert.NotEmpty(t, context.Specifications)
	assert.Contains(t, context.Specifications, "TS 23.501")
	assert.Contains(t, context.Specifications, "TS 38.413") // N2 interface spec
	assert.Contains(t, context.Compliance, "3GPP Release 17")
	assert.Contains(t, context.Compliance, "O-RAN Alliance")
}

func TestCachefunctionality(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	intent := "Deploy AMF for 5G network"
	
	// First call should populate cache
	result1, err1 := cm.GetRelevantContext(intent)
	require.NoError(t, err1)
	assert.Equal(t, 1, cm.GetCacheSize())
	
	// Second call should use cache
	result2, err2 := cm.GetRelevantContext(intent)
	require.NoError(t, err2)
	assert.Equal(t, 1, cm.GetCacheSize()) // Still 1 entry
	
	// Results should be equivalent (same pointer due to cache)
	assert.Equal(t, result1.Intent, result2.Intent)
	assert.Equal(t, len(result1.NetworkFunctions), len(result2.NetworkFunctions))
	
	// Clear cache
	cm.ClearCache()
	assert.Equal(t, 0, cm.GetCacheSize())
}

func TestAbbreviationExpansion(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	testCases := []struct {
		input    string
		contains string // What the normalized string should contain
	}{
		{
			input:    "Deploy 5GC functions",
			contains: "5g core",
		},
		{
			input:    "Configure QoS policies",
			contains: "quality of service",
		},
		{
			input:    "Setup VNF instances",
			contains: "virtual network function",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			normalized := cm.normalizeIntent(tc.input)
			assert.Contains(t, normalized, tc.contains)
		})
	}
}

func TestComplexIntentScenarios(t *testing.T) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	
	complexIntents := []struct {
		intent               string
		expectedNFs          []string
		expectedSliceTypes   []int    // SST values
		minConfidence       float64
	}{
		{
			intent: "Deploy eMBB slice with AMF, SMF, and UPF for mobile broadband services with 100Mbps throughput",
			expectedNFs: []string{"AMF", "SMF", "UPF"},
			expectedSliceTypes: []int{1}, // eMBB SST
			minConfidence: 0.5,
		},
		{
			intent: "Setup URLLC network slice for industrial automation with 1ms latency and 99.999% reliability",
			expectedSliceTypes: []int{2}, // URLLC SST
			minConfidence: 0.4,
		},
		{
			intent: "Configure N4 interface between SMF and UPF with PFCP protocol for session management",
			expectedNFs: []string{"SMF", "UPF"},
			minConfidence: 0.6,
		},
	}
	
	for _, scenario := range complexIntents {
		t.Run(scenario.intent, func(t *testing.T) {
			result, err := cm.GetRelevantContext(scenario.intent)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, result.Confidence, scenario.minConfidence)
			
			// Check expected network functions
			if len(scenario.expectedNFs) > 0 {
				assert.NotEmpty(t, result.NetworkFunctions)
				foundNFs := make(map[string]bool)
				for _, nfMatch := range result.NetworkFunctions {
					foundNFs[nfMatch.Function.Name] = true
				}
				
				for _, expectedNF := range scenario.expectedNFs {
					assert.True(t, foundNFs[expectedNF], "Expected to find NF: %s", expectedNF)
				}
			}
			
			// Check expected slice types
			if len(scenario.expectedSliceTypes) > 0 {
				assert.NotEmpty(t, result.SliceTypes)
				foundSSTs := make(map[int]bool)
				for _, sliceMatch := range result.SliceTypes {
					foundSSTs[sliceMatch.SliceType.SST] = true
				}
				
				for _, expectedSST := range scenario.expectedSliceTypes {
					assert.True(t, foundSSTs[expectedSST], "Expected to find slice SST: %d", expectedSST)
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkGetRelevantContext(b *testing.B) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	intent := "Deploy AMF with high availability for production environment"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.GetRelevantContext(intent)
	}
}

func BenchmarkGetRelevantContextCached(b *testing.B) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	intent := "Deploy AMF with high availability for production environment"
	
	// Prime the cache
	cm.GetRelevantContext(intent)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.GetRelevantContext(intent)
	}
}

func BenchmarkNormalizeIntent(b *testing.B) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	intent := "Deploy 5GC AMF with high availability and QoS configuration"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.normalizeIntent(intent)
	}
}

func BenchmarkExtractRequirements(b *testing.B) {
	kb := NewTelecomKnowledgeBase()
	cm := NewContextMatcher(kb)
	intent := "deploy with 5ms latency, 1gbps throughput, 99.99% availability and auto-scaling from 3 to 10 replicas"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cm.extractRequirements(intent)
	}
}