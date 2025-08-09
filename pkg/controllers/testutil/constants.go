package testutil

import "time"

const (
	// Test timeouts and intervals
	TestTimeout  = time.Second * 30
	TestInterval = time.Millisecond * 250

	// Default test values
	DefaultReplicas      = int32(3)
	DefaultTestEndpoint  = "http://localhost:8080"
	DefaultGitRepoURL    = "https://github.com/test/deployments.git"
	DefaultGitBranch     = "main"
	DefaultGitDeployPath = "networkintents"
	DefaultMaxRetries    = 3
	DefaultRetryDelay    = time.Second * 1
	DefaultHTTPTimeout   = 30 * time.Second
	DefaultRequeueAfter  = 30 * time.Second

	// Test resource names
	TestE2NodeSetName     = "test-e2nodeset"
	TestNetworkIntentName = "test-networkintent"
	TestNamespacePrefix   = "test-ns"

	// Test labels and selectors
	TestAppLabel = "test-app"
	TestAppName  = "test-e2node"

	// Test function parameters
	TestFunctionID   = 1
	TestFunctionOID  = "1.3.6.1.4.1.1.1.1.1"
	TestServiceModel = "kpm"
	TestFunctionDesc = "Test function"

	// Test network intent parameters
	TestBandwidth  = "100Mbps"
	TestLatency    = "10ms"
	TestIntentType = "qos"
	TestPriority   = 1
)
