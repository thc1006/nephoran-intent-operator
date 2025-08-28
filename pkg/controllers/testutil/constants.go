package testutil

import "time"

const (
	// Test timeouts and intervals.
	TestTimeout = time.Second * 30
	// TestInterval holds testinterval value.
	TestInterval = time.Millisecond * 250

	// Default test values.
	DefaultReplicas = int32(3)
	// DefaultTestEndpoint holds defaulttestendpoint value.
	DefaultTestEndpoint = "http://localhost:8080"
	// DefaultGitRepoURL holds defaultgitrepourl value.
	DefaultGitRepoURL = "https://github.com/test/deployments.git"
	// DefaultGitBranch holds defaultgitbranch value.
	DefaultGitBranch = "main"
	// DefaultGitDeployPath holds defaultgitdeploypath value.
	DefaultGitDeployPath = "networkintents"
	// DefaultMaxRetries holds defaultmaxretries value.
	DefaultMaxRetries = 3
	// DefaultRetryDelay holds defaultretrydelay value.
	DefaultRetryDelay = time.Second * 1
	// DefaultHTTPTimeout holds defaulthttptimeout value.
	DefaultHTTPTimeout = 30 * time.Second
	// DefaultRequeueAfter holds defaultrequeueafter value.
	DefaultRequeueAfter = 30 * time.Second

	// Test resource names.
	TestE2NodeSetName = "test-e2nodeset"
	// TestNetworkIntentName holds testnetworkintentname value.
	TestNetworkIntentName = "test-networkintent"
	// TestNamespacePrefix holds testnamespaceprefix value.
	TestNamespacePrefix = "test-ns"

	// Test labels and selectors.
	TestAppLabel = "test-app"
	// TestAppName holds testappname value.
	TestAppName = "test-e2node"

	// Test function parameters.
	TestFunctionID = 1
	// TestFunctionOID holds testfunctionoid value.
	TestFunctionOID = "1.3.6.1.4.1.1.1.1.1"
	// TestServiceModel holds testservicemodel value.
	TestServiceModel = "kpm"
	// TestFunctionDesc holds testfunctiondesc value.
	TestFunctionDesc = "Test function"

	// Test network intent parameters.
	TestBandwidth = "100Mbps"
	// TestLatency holds testlatency value.
	TestLatency = "10ms"
	// TestIntentType holds testintenttype value.
	TestIntentType = "qos"
	// TestPriority holds testpriority value.
	TestPriority = 1
)
