package monitoring

// ExampleTestMain demonstrates how to properly set up metrics for tests
// to prevent duplicate registration errors.
//
// Copy this pattern to your test packages that use Prometheus metrics:
//
//	func TestMain(m *testing.M) {
//		// Setup global test environment
//		setupGlobalTestMetrics()
//		
//		// Run tests
//		code := m.Run()
//		
//		// Cleanup
//		teardownGlobalTestMetrics()
//		
//		os.Exit(code)
//	}
func exampleTestMain() {
	// This is just an example - actual TestMain would take m *testing.M
	// Setup global test environment
	setupGlobalTestMetrics()
	
	// Run tests would be here: code := m.Run()
	
	// Cleanup
	teardownGlobalTestMetrics()
	
	// Exit would be here: os.Exit(code)
}

func setupGlobalTestMetrics() {
	// Enable test mode globally for all metrics
	gr := GetGlobalRegistry()
	gr.SetTestMode(true)
	gr.ResetForTest()
}

func teardownGlobalTestMetrics() {
	// Reset back to production mode and clean up
	TeardownTestMetrics()
}

// Example of how to write a test that uses metrics without duplicates
func exampleTest() {
	// This is just an example - actual test would take t *testing.T
	// Each individual test should set up its own isolated metrics
	// SetupTestMetrics(t)
	
	// Now you can safely register metrics without worrying about duplicates
	gr := GetGlobalRegistry()
	_ = gr.SafeRegister("example-component", nil) // your metric here
	
	// Test continues...
	// The t.Cleanup() registered by SetupTestMetrics will handle cleanup
}
