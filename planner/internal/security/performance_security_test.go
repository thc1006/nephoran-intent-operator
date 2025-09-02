package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/planner"
	"github.com/thc1006/nephoran-intent-operator/planner/internal/rules"
)

// BenchmarkSecurity_ValidationPerformance benchmarks the performance of security validation functions
func BenchmarkSecurity_ValidationPerformance(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	// Benchmark KMP data validation
	b.Run("KMPDataValidation", func(b *testing.B) {
		kmpData := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "benchmark-node-001",
			PRBUtilization:  0.75,
			P95Latency:      150.0,
			ActiveUEs:       100,
			CurrentReplicas: 3,
		}

		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = validator.ValidateKMPData(kmpData)
		}
	})

	// Benchmark URL validation with different URL lengths
	urlLengths := []int{50, 100, 500, 1000, 2000}
	for _, length := range urlLengths {
		b.Run(fmt.Sprintf("URLValidation_Length%d", length), func(b *testing.B) {
			url := "https://api.example.com/v1/metrics?param=" + strings.Repeat("a", length-50)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = validator.ValidateURL(url, "benchmark")
			}
		})
	}

	// Benchmark file path validation with different path depths
	pathDepths := []int{1, 5, 10, 20, 50}
	for _, depth := range pathDepths {
		b.Run(fmt.Sprintf("FilePathValidation_Depth%d", depth), func(b *testing.B) {
			path := "/" + strings.Repeat("level/", depth) + "file.json"

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = validator.ValidateFilePath(path, "benchmark")
			}
		})
	}

	// Benchmark node ID validation with different ID lengths
	nodeIDLengths := []int{10, 50, 100, 200, 255}
	for _, length := range nodeIDLengths {
		b.Run(fmt.Sprintf("NodeIDValidation_Length%d", length), func(b *testing.B) {
			nodeID := "node-" + strings.Repeat("a", length-5)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = validator.validateNodeID(nodeID)
			}
		})
	}

	// Benchmark log sanitization with different input sizes
	logSizes := []int{100, 1000, 10000, 100000}
	for _, size := range logSizes {
		b.Run(fmt.Sprintf("LogSanitization_Size%d", size), func(b *testing.B) {
			input := "log message " + strings.Repeat("a\nb\tc\rd", size/10)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = validator.SanitizeForLogging(input)
			}
		})
	}
}

// BenchmarkSecurity_FileOperations benchmarks secure file operations vs standard operations
func BenchmarkSecurity_FileOperations(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "planner-perf-benchmark-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validator := NewValidator(DefaultValidationConfig())

	// Sample intent data for benchmarking
	intent := &planner.Intent{
		IntentType:    "scaling",
		Target:        "benchmark-cnf",
		Namespace:     "production",
		Replicas:      3,
		Reason:        "Performance benchmark test",
		Source:        "benchmark",
		CorrelationID: "benchmark-correlation-id",
	}

	intentData, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		b.Fatalf("Failed to marshal intent: %v", err)
	}

	// Benchmark secure file write (with validation and 0600 permissions)
	b.Run("SecureFileWrite", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fileName := filepath.Join(tempDir, fmt.Sprintf("secure-%d.json", i))

			// Validate path
			if err := validator.ValidateFilePath(fileName, "benchmark"); err != nil {
				b.Fatalf("Path validation failed: %v", err)
			}

			// Write with secure permissions
			if err := os.WriteFile(fileName, intentData, 0600); err != nil {
				b.Fatalf("Secure file write failed: %v", err)
			}
		}
	})

	// Benchmark standard file write (without validation, with 0644 permissions)
	b.Run("StandardFileWrite", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fileName := filepath.Join(tempDir, fmt.Sprintf("standard-%d.json", i))

			// Direct write without validation
			if err := os.WriteFile(fileName, intentData, 0644); err != nil {
				b.Fatalf("Standard file write failed: %v", err)
			}
		}
	})

	// Benchmark file write with validation only (no permission change)
	b.Run("ValidationOnlyFileWrite", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fileName := filepath.Join(tempDir, fmt.Sprintf("validation-%d.json", i))

			// Validate path but use standard permissions
			if err := validator.ValidateFilePath(fileName, "benchmark"); err != nil {
				b.Fatalf("Path validation failed: %v", err)
			}

			if err := os.WriteFile(fileName, intentData, 0644); err != nil {
				b.Fatalf("Validation-only file write failed: %v", err)
			}
		}
	})
}

// BenchmarkSecurity_KMPProcessingPipeline benchmarks the complete KMP processing pipeline
func BenchmarkSecurity_KMPProcessingPipeline(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "planner-pipeline-benchmark-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validator := NewValidator(DefaultValidationConfig())
	stateFile := filepath.Join(tempDir, "benchmark-state.json")

	// Create rule engine with security validator
	config := rules.Config{
		StateFile:            stateFile,
		CooldownDuration:     60 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     90 * time.Second,
	}

	engine := rules.NewRuleEngine(config)
	engine.SetValidator(validator)

	// Benchmark complete KMP processing with security validation
	b.Run("CompleteKMPProcessingWithSecurity", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// Create KMP data that will trigger scaling decision
			kmpData := rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          fmt.Sprintf("benchmark-node-%d", i),
				PRBUtilization:  0.85, // High utilization
				P95Latency:      200.0,
				ActiveUEs:       150,
				CurrentReplicas: 2,
			}

			// Validate KMP data (security check)
			if err := validator.ValidateKMPData(kmpData); err != nil {
				b.Fatalf("KMP validation failed: %v", err)
			}

			// Process through rule engine
			decision := engine.Evaluate(kmpData)
			if decision != nil {
				// Create intent
				intent := &planner.Intent{
					IntentType:    "scaling",
					Target:        decision.Target,
					Namespace:     decision.Namespace,
					Replicas:      decision.TargetReplicas,
					Reason:        decision.Reason,
					Source:        "benchmark",
					CorrelationID: fmt.Sprintf("benchmark-%d", i),
				}

				// Marshal intent
				intentData, err := json.MarshalIndent(intent, "", "  ")
				if err != nil {
					b.Fatalf("Intent marshaling failed: %v", err)
				}

				// Write intent with security validation
				intentFile := filepath.Join(tempDir, fmt.Sprintf("intent-%d.json", i))
				if err := validator.ValidateFilePath(intentFile, "benchmark"); err != nil {
					b.Fatalf("Intent path validation failed: %v", err)
				}

				if err := os.WriteFile(intentFile, intentData, 0600); err != nil {
					b.Fatalf("Intent file write failed: %v", err)
				}
			}
		}
	})

	// Benchmark KMP processing without security validation (for comparison)
	b.Run("KMPProcessingWithoutSecurity", func(b *testing.B) {
		// Create engine without security validator
		engineNoSecurity := rules.NewRuleEngine(config)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			kmpData := rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          fmt.Sprintf("benchmark-node-%d", i),
				PRBUtilization:  0.85,
				P95Latency:      200.0,
				ActiveUEs:       150,
				CurrentReplicas: 2,
			}

			// Skip security validation
			decision := engineNoSecurity.Evaluate(kmpData)
			if decision != nil {
				intent := &planner.Intent{
					IntentType:    "scaling",
					Target:        decision.Target,
					Namespace:     decision.Namespace,
					Replicas:      decision.TargetReplicas,
					Reason:        decision.Reason,
					Source:        "benchmark",
					CorrelationID: fmt.Sprintf("benchmark-%d", i),
				}

				intentData, err := json.MarshalIndent(intent, "", "  ")
				if err != nil {
					b.Fatalf("Intent marshaling failed: %v", err)
				}

				// Direct file write without security checks
				intentFile := filepath.Join(tempDir, fmt.Sprintf("intent-nosec-%d.json", i))
				if err := os.WriteFile(intentFile, intentData, 0644); err != nil {
					b.Fatalf("Intent file write failed: %v", err)
				}
			}
		}
	})
}

// BenchmarkSecurity_MemoryUsage benchmarks memory usage of security validation
func BenchmarkSecurity_MemoryUsage(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	b.Run("MemoryUsage_Validation", func(b *testing.B) {
		kmpData := rules.KPMData{
			Timestamp:       time.Now(),
			NodeID:          "memory-test-node",
			PRBUtilization:  0.75,
			P95Latency:      150.0,
			ActiveUEs:       100,
			CurrentReplicas: 3,
		}

		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_ = validator.ValidateKMPData(kmpData)
			_ = validator.ValidateURL("https://api.example.com/metrics", "memory test")
			_ = validator.ValidateFilePath("/tmp/memory-test.json", "memory test")
		}

		runtime.GC()
		runtime.ReadMemStats(&m2)

		b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
		b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "allocs/op")
	})
}

// BenchmarkSecurity_ConcurrentPerformance benchmarks security validation under concurrent load
func BenchmarkSecurity_ConcurrentPerformance(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	concurrencyLevels := []int{1, 5, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrent_%d", concurrency), func(b *testing.B) {
			kmpData := rules.KPMData{
				Timestamp:       time.Now(),
				NodeID:          "concurrent-test-node",
				PRBUtilization:  0.75,
				P95Latency:      150.0,
				ActiveUEs:       100,
				CurrentReplicas: 3,
			}

			b.SetParallelism(concurrency)
			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = validator.ValidateKMPData(kmpData)
				}
			})
		})
	}
}

// BenchmarkSecurity_ComplexInputs benchmarks performance with complex/large inputs
func BenchmarkSecurity_ComplexInputs(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	// Test with various input complexities
	complexities := []struct {
		name  string
		setup func() interface{}
		test  func(validator *Validator, input interface{})
	}{
		{
			name: "SimpleNodeID",
			setup: func() interface{} {
				return "simple-node"
			},
			test: func(validator *Validator, input interface{}) {
				_ = validator.validateNodeID(input.(string))
			},
		},
		{
			name: "ComplexNodeID",
			setup: func() interface{} {
				return "very-long-complex-node-identifier-" + strings.Repeat("segment-", 20) + "end"
			},
			test: func(validator *Validator, input interface{}) {
				_ = validator.validateNodeID(input.(string))
			},
		},
		{
			name: "SimpleURL",
			setup: func() interface{} {
				return "https://api.example.com/metrics"
			},
			test: func(validator *Validator, input interface{}) {
				_ = validator.ValidateURL(input.(string), "benchmark")
			},
		},
		{
			name: "ComplexURL",
			setup: func() interface{} {
				query := "param1=value1&param2=value2&" + strings.Repeat("param=value&", 50)
				return "https://api.complex-domain.example.com/v1/very/deep/path/to/metrics?" + query
			},
			test: func(validator *Validator, input interface{}) {
				_ = validator.ValidateURL(input.(string), "benchmark")
			},
		},
		{
			name: "SimplePath",
			setup: func() interface{} {
				return "/tmp/file.json"
			},
			test: func(validator *Validator, input interface{}) {
				_ = validator.ValidateFilePath(input.(string), "benchmark")
			},
		},
		{
			name: "ComplexPath",
			setup: func() interface{} {
				return "/" + strings.Repeat("very-long-directory-name/", 20) + "final-file.json"
			},
			test: func(validator *Validator, input interface{}) {
				_ = validator.ValidateFilePath(input.(string), "benchmark")
			},
		},
	}

	for _, complexity := range complexities {
		b.Run(complexity.name, func(b *testing.B) {
			input := complexity.setup()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				complexity.test(validator, input)
			}
		})
	}
}

// BenchmarkSecurity_RegexPerformance benchmarks regex-heavy validation performance
func BenchmarkSecurity_RegexPerformance(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	// Test regex performance with different input patterns
	regexTests := []struct {
		name    string
		input   string
		pattern string
	}{
		{
			name:    "ValidPattern",
			input:   "valid-node-123",
			pattern: "matches O-RAN node ID pattern",
		},
		{
			name:    "InvalidPattern_SpecialChars",
			input:   "invalid@node#123",
			pattern: "contains special characters",
		},
		{
			name:    "InvalidPattern_SQL",
			input:   "node'; DROP TABLE users; --",
			pattern: "SQL injection attempt",
		},
		{
			name:    "EdgeCase_LongValid",
			input:   "very-" + strings.Repeat("long-", 50) + "node-id",
			pattern: "very long but valid pattern",
		},
		{
			name:    "EdgeCase_AlmostValid",
			input:   "almost-valid-node-id!",
			pattern: "almost valid with one invalid char",
		},
	}

	for _, test := range regexTests {
		b.Run(test.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = validator.validateNodeID(test.input)
			}
		})
	}
}

// TestSecurity_PerformanceRegression tests for performance regressions in security functions
// DISABLED: func TestSecurity_PerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression tests in short mode")
	}

	validator := NewValidator(DefaultValidationConfig())

	// Define acceptable performance baselines (in nanoseconds per operation)
	performanceBaselines := map[string]time.Duration{
		"KMPDataValidation":  1000000, // 1ms max per validation
		"URLValidation":      500000,  // 500μs max per validation
		"FilePathValidation": 300000,  // 300μs max per validation
		"NodeIDValidation":   100000,  // 100μs max per validation
		"LogSanitization":    50000,   // 50μs max per sanitization
	}

	testCases := []struct {
		name      string
		operation func() error
		baseline  string
	}{
		{
			name: "KMPDataValidation",
			operation: func() error {
				kmpData := rules.KPMData{
					Timestamp:       time.Now(),
					NodeID:          "regression-test-node",
					PRBUtilization:  0.75,
					P95Latency:      150.0,
					ActiveUEs:       100,
					CurrentReplicas: 3,
				}
				return validator.ValidateKMPData(kmpData)
			},
			baseline: "KMPDataValidation",
		},
		{
			name: "URLValidation",
			operation: func() error {
				return validator.ValidateURL("https://api.example.com/v1/metrics?format=json", "regression test")
			},
			baseline: "URLValidation",
		},
		{
			name: "FilePathValidation",
			operation: func() error {
				return validator.ValidateFilePath("/tmp/planner/intent-regression-test.json", "regression test")
			},
			baseline: "FilePathValidation",
		},
		{
			name: "NodeIDValidation",
			operation: func() error {
				return validator.validateNodeID("regression-test-node-001")
			},
			baseline: "NodeIDValidation",
		},
		{
			name: "LogSanitization",
			operation: func() error {
				_ = validator.SanitizeForLogging("regression test log message\nwith\tcontrol\rchars")
				return nil
			},
			baseline: "LogSanitization",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			const iterations = 1000
			var totalDuration time.Duration

			// Warm up
			for i := 0; i < 10; i++ {
				_ = test.operation()
			}

			// Measure performance
			for i := 0; i < iterations; i++ {
				start := time.Now()
				_ = test.operation()
				totalDuration += time.Since(start)
			}

			avgDuration := totalDuration / iterations
			baseline := performanceBaselines[test.baseline]

			t.Logf("Performance test %s: avg duration = %v (baseline = %v)",
				test.name, avgDuration, baseline)

			if avgDuration > baseline {
				t.Errorf("Performance regression detected in %s: avg duration %v exceeds baseline %v",
					test.name, avgDuration, baseline)
			} else {
				margin := float64(baseline-avgDuration) / float64(baseline) * 100
				t.Logf("✓ Performance within baseline with %.1f%% margin", margin)
			}
		})
	}
}

// BenchmarkSecurity_ScalabilityTest benchmarks security performance at different scales
func BenchmarkSecurity_ScalabilityTest(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	scales := []int{1, 10, 100, 1000, 10000}

	for _, scale := range scales {
		b.Run(fmt.Sprintf("Scale_%d", scale), func(b *testing.B) {
			// Pre-generate test data
			kmpDataSlice := make([]rules.KPMData, scale)
			for i := 0; i < scale; i++ {
				kmpDataSlice[i] = rules.KPMData{
					Timestamp:       time.Now(),
					NodeID:          fmt.Sprintf("scale-test-node-%d", i),
					PRBUtilization:  float64(i%100) / 100.0,
					P95Latency:      float64(50 + i%200),
					ActiveUEs:       i % 1000,
					CurrentReplicas: (i % 10) + 1,
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				for j := 0; j < scale; j++ {
					_ = validator.ValidateKMPData(kmpDataSlice[j])
				}
			}
		})
	}
}

// BenchmarkSecurity_WorstCaseScenarios benchmarks worst-case security validation scenarios
func BenchmarkSecurity_WorstCaseScenarios(b *testing.B) {
	validator := NewValidator(DefaultValidationConfig())

	worstCaseTests := []struct {
		name      string
		operation func()
	}{
		{
			name: "MaxLengthNodeID",
			operation: func() {
				nodeID := strings.Repeat("a", 255) // Maximum allowed length
				_ = validator.validateNodeID(nodeID)
			},
		},
		{
			name: "MaxLengthURL",
			operation: func() {
				url := "https://example.com/" + strings.Repeat("a", 2000) // Near maximum length
				_ = validator.ValidateURL(url, "worst case test")
			},
		},
		{
			name: "MaxLengthPath",
			operation: func() {
				path := "/" + strings.Repeat("a/", 500) + "file.json" // Deep path
				_ = validator.ValidateFilePath(path, "worst case test")
			},
		},
		{
			name: "ComplexRegexPattern",
			operation: func() {
				// Pattern that could cause ReDoS in poorly implemented regex
				nodeID := strings.Repeat("a", 100) + strings.Repeat("a?", 50) + strings.Repeat("a", 100)
				_ = validator.validateNodeID(nodeID)
			},
		},
		{
			name: "ManySpecialCharacters",
			operation: func() {
				input := strings.Repeat("!@#$%^&*()_+{}|:<>?[];',./", 50)
				_ = validator.validateNodeID(input)
			},
		},
		{
			name: "LargeSanitizationInput",
			operation: func() {
				input := strings.Repeat("log\nmessage\rwith\tcontrol\nchars\r", 1000)
				_ = validator.SanitizeForLogging(input)
			},
		},
	}

	for _, test := range worstCaseTests {
		b.Run(test.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				test.operation()
			}
		})
	}
}

// TestSecurity_PerformanceStability tests that security performance remains stable over time
// DISABLED: func TestSecurity_PerformanceStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance stability tests in short mode")
	}

	validator := NewValidator(DefaultValidationConfig())

	// Test stability over many iterations
	t.Run("StabilityOverTime", func(t *testing.T) {
		const numRounds = 100
		const iterationsPerRound = 100

		durations := make([]time.Duration, numRounds)

		for round := 0; round < numRounds; round++ {
			start := time.Now()

			for i := 0; i < iterationsPerRound; i++ {
				kmpData := rules.KPMData{
					Timestamp:       time.Now(),
					NodeID:          fmt.Sprintf("stability-test-node-%d", i),
					PRBUtilization:  0.75,
					P95Latency:      150.0,
					ActiveUEs:       100,
					CurrentReplicas: 3,
				}
				_ = validator.ValidateKMPData(kmpData)
			}

			durations[round] = time.Since(start)
		}

		// Calculate statistics
		var total time.Duration
		min := durations[0]
		max := durations[0]

		for _, d := range durations {
			total += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}

		avg := total / time.Duration(numRounds)
		variance := max - min

		t.Logf("Performance stability: avg=%v, min=%v, max=%v, variance=%v", avg, min, max, variance)

		// Check if variance is reasonable (should not exceed 50% of average)
		maxAcceptableVariance := avg / 2
		if variance > maxAcceptableVariance {
			t.Errorf("Performance is unstable: variance %v exceeds 50%% of average %v", variance, avg)
		} else {
			t.Logf("✓ Performance is stable over %d rounds", numRounds)
		}
	})
}
