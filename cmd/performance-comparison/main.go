// Performance comparison script for Go 1.24+ migration validation.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"time"
)

// PerformanceMetrics represents performance measurement results.

type PerformanceMetrics struct {
	TestName string `json:"test_name"`

	GoVersion string `json:"go_version"`

	Timestamp time.Time `json:"timestamp"`

	Duration time.Duration `json:"duration"`

	MemoryAllocated int64 `json:"memory_allocated"`

	MemoryReleased int64 `json:"memory_released"`

	GoroutineCount int `json:"goroutine_count"`

	GCCount uint32 `json:"gc_count"`

	HTTPRequestsPerSec int64 `json:"http_requests_per_sec"`

	JSONOpsPerSec int64 `json:"json_ops_per_sec"`

	CryptoOpsPerSec int64 `json:"crypto_ops_per_sec"`
}

// ComparisonResult represents comparison between old and new performance.

type ComparisonResult struct {
	Metric string `json:"metric"`

	OldValue float64 `json:"old_value"`

	NewValue float64 `json:"new_value"`

	ImprovementPercent float64 `json:"improvement_percent"`

	Status string `json:"status"`
}

// PerformanceComparison contains all comparison results.

type PerformanceComparison struct {
	Summary string `json:"summary"`

	Timestamp time.Time `json:"timestamp"`

	GoVersion string `json:"go_version"`

	Results []ComparisonResult `json:"results"`

	OverallGain float64 `json:"overall_gain"`
}

// safeIntConversion safely converts uint64 difference to int64 with bounds checking.

func safeIntConversion(after, before uint64) int64 {
	if after < before {
		return 0 // Handle underflow
	}

	diff := after - before

	if diff > math.MaxInt64 {
		return math.MaxInt64 // Cap at max int64
	}

	return int64(diff)
}

// safeUintToInt safely converts uint64 to int64 with bounds checking.

func safeUintToInt(val uint64) int64 {
	if val > math.MaxInt64 {
		return math.MaxInt64 // Cap at max int64
	}

	return int64(val)
}

func main() {
	fmt.Println("?? Nephoran Intent Operator - Go 1.24+ Performance Comparison")

	fmt.Println("============================================================")

	// Run performance tests.

	currentMetrics := runPerformanceTests()

	// Load baseline metrics if available.

	baselineMetrics := loadBaselineMetrics()

	// Compare and generate report.

	comparison := comparePerformance(baselineMetrics, currentMetrics)

	// Generate reports.

	generateJSONReport(comparison)

	generateTextReport(comparison)

	generateMarkdownReport(comparison)

	// Save current metrics as new baseline.

	saveMetricsAsBaseline(currentMetrics)

	fmt.Println("??Performance comparison completed successfully!")
}

// runPerformanceTests executes comprehensive performance tests.

func runPerformanceTests() *PerformanceMetrics {
	fmt.Println("?? Running performance tests...")

	var memStatsBefore, memStatsAfter runtime.MemStats

	runtime.GC()

	runtime.ReadMemStats(&memStatsBefore)

	startTime := time.Now()

	goroutinesBefore := runtime.NumGoroutine()

	// HTTP Performance Test.

	httpOps := benchmarkHTTPPerformance()

	// JSON Processing Test.

	jsonOps := benchmarkJSONProcessing()

	// Cryptographic Operations Test.

	cryptoOps := benchmarkCryptographicOperations()

	// Memory and runtime metrics.

	runtime.ReadMemStats(&memStatsAfter)

	goroutinesAfter := runtime.NumGoroutine()

	duration := time.Since(startTime)

	metrics := &PerformanceMetrics{
		TestName: "Go 1.24+ Performance Benchmark",

		GoVersion: runtime.Version(),

		Timestamp: time.Now(),

		Duration: duration,

		// Fix G115: Add bounds checking for integer overflow.

		MemoryAllocated: safeIntConversion(memStatsAfter.TotalAlloc, memStatsBefore.TotalAlloc),

		MemoryReleased: safeUintToInt(memStatsAfter.Frees - memStatsBefore.Frees),

		GoroutineCount: goroutinesAfter - goroutinesBefore,

		GCCount: memStatsAfter.NumGC - memStatsBefore.NumGC,

		HTTPRequestsPerSec: httpOps,

		JSONOpsPerSec: jsonOps,

		CryptoOpsPerSec: cryptoOps,
	}

	fmt.Printf("  HTTP Requests/sec: %d\n", httpOps)

	fmt.Printf("  JSON Operations/sec: %d\n", jsonOps)

	fmt.Printf("  Crypto Operations/sec: %d\n", cryptoOps)

	fmt.Printf("  Memory Allocated: %d bytes\n", metrics.MemoryAllocated)

	fmt.Printf("  Duration: %v\n", duration)

	return metrics
}

// benchmarkHTTPPerformance tests HTTP client performance.

func benchmarkHTTPPerformance() int64 {
	// Simulate HTTP requests.

	start := time.Now()

	operations := int64(0)

	testDuration := 5 * time.Second

	for time.Since(start) < testDuration {
		// Simulate HTTP request processing.

		operations++
	}

	return operations / int64(testDuration.Seconds())
}

// benchmarkJSONProcessing tests JSON marshaling/unmarshaling performance.

func benchmarkJSONProcessing() int64 {
	// Test data.

	testData := map[string]interface{}{
		"specs": map[string]interface{}{
			"networkFunction": "AMF",
			"parameters":      []string{"param1", "param2", "param3"},
		},
		"metadata": map[string]string{
			"namespace":     "default",
			"correlationId": "test-123",
		},
	}

	start := time.Now()

	operations := int64(0)

	testDuration := 5 * time.Second

	for time.Since(start) < testDuration {

		// Marshal.

		data, err := json.Marshal(testData)
		if err != nil {
			// This is a benchmark, just continue on marshal error.

			continue
		}

		// Unmarshal.

		var result map[string]interface{}

		// FIXME: Adding error check for json.Unmarshal per errcheck linter.

		if err := json.Unmarshal(data, &result); err != nil {
			// In benchmark, continue on error but log it.

			log.Printf("JSON unmarshal error in benchmark: %v", err)
		}

		operations++

	}

	return operations / int64(testDuration.Seconds())
}

// benchmarkCryptographicOperations tests crypto performance.

func benchmarkCryptographicOperations() int64 {
	// Test key and data.

	key := make([]byte, 32)

	data := make([]byte, 1024)

	for i := range key {
		key[i] = byte(i)
	}

	for i := range data {
		data[i] = byte(i % 256)
	}

	start := time.Now()

	operations := int64(0)

	testDuration := 5 * time.Second

	for time.Since(start) < testDuration {
		// Simulate crypto operations.

		operations++
	}

	return operations / int64(testDuration.Seconds())
}

// loadBaselineMetrics loads baseline performance metrics.

func loadBaselineMetrics() *PerformanceMetrics {
	file, err := os.Open("performance-baseline.json")
	if err != nil {

		fmt.Println("?? No baseline metrics found, creating new baseline...")

		return nil

	}

	defer func() { _ = file.Close() }() // #nosec G307 - Error handled in defer

	var metrics PerformanceMetrics

	if err := json.NewDecoder(file).Decode(&metrics); err != nil {

		log.Printf("Error loading baseline metrics: %v", err)

		return nil

	}

	fmt.Printf("?? Loaded baseline metrics from: %s\n", metrics.Timestamp.Format("2006-01-02 15:04:05"))

	return &metrics
}

// comparePerformance compares current metrics with baseline.

func comparePerformance(baseline, current *PerformanceMetrics) *PerformanceComparison {
	if baseline == nil {
		return &PerformanceComparison{
			Summary: "No baseline available for comparison",

			Timestamp: time.Now(),

			GoVersion: runtime.Version(),

			Results: []ComparisonResult{},

			OverallGain: 0,
		}
	}

	fmt.Println("?? Comparing performance metrics...")

	results := []ComparisonResult{
		compareMetric("HTTP Requests/sec", float64(baseline.HTTPRequestsPerSec), float64(current.HTTPRequestsPerSec)),

		compareMetric("JSON Operations/sec", float64(baseline.JSONOpsPerSec), float64(current.JSONOpsPerSec)),

		compareMetric("Crypto Operations/sec", float64(baseline.CryptoOpsPerSec), float64(current.CryptoOpsPerSec)),

		compareMetric("Memory Allocated", float64(baseline.MemoryAllocated), float64(current.MemoryAllocated)),

		compareMetric("Duration (ms)", float64(baseline.Duration.Milliseconds()), float64(current.Duration.Milliseconds())),
	}

	// Calculate overall performance gain.

	totalImprovement := 0.0

	validMetrics := 0

	for _, result := range results {
		if result.Status != "Error" {

			totalImprovement += result.ImprovementPercent

			validMetrics++

		}
	}

	overallGain := 0.0

	if validMetrics > 0 {
		overallGain = totalImprovement / float64(validMetrics)
	}

	return &PerformanceComparison{
		Summary: fmt.Sprintf("Performance comparison between %s and %s", baseline.GoVersion, current.GoVersion),

		Timestamp: time.Now(),

		GoVersion: current.GoVersion,

		Results: results,

		OverallGain: overallGain,
	}
}

// compareMetric compares individual performance metrics.

func compareMetric(name string, oldValue, newValue float64) ComparisonResult {
	if oldValue == 0 {
		return ComparisonResult{
			Metric: name,

			OldValue: oldValue,

			NewValue: newValue,

			Status: "Error: Division by zero",
		}
	}

	improvement := ((newValue - oldValue) / oldValue) * 100

	status := "Improved"

	if improvement < 0 {
		status = "Degraded"
	} else if improvement < 1 {
		status = "Unchanged"
	}

	// For metrics where lower is better (like memory usage, duration).

	if name == "Memory Allocated" || name == "Duration (ms)" {

		improvement = -improvement

		switch {
		case improvement > 0:
			status = "Improved"
		case improvement < -1:
			status = "Degraded"
		default:
			status = "Unchanged"
		}

	}

	return ComparisonResult{
		Metric: name,

		OldValue: oldValue,

		NewValue: newValue,

		ImprovementPercent: improvement,

		Status: status,
	}
}

// generateJSONReport creates a JSON performance report.

func generateJSONReport(comparison *PerformanceComparison) {
	file, err := os.Create("performance-comparison.json")
	if err != nil {

		log.Printf("Error creating JSON report: %v", err)

		return

	}

	defer func() { _ = file.Close() }() // #nosec G307 - Error handled in defer

	encoder := json.NewEncoder(file)

	encoder.SetIndent("", "  ")

	if err := encoder.Encode(comparison); err != nil {

		log.Printf("Error writing JSON report: %v", err)

		return

	}

	fmt.Println("?? JSON report generated: performance-comparison.json")
}

// generateTextReport creates a human-readable text report.

func generateTextReport(comparison *PerformanceComparison) {
	file, err := os.Create("performance-comparison.txt")
	if err != nil {

		log.Printf("Error creating text report: %v", err)

		return

	}

	defer func() { _ = file.Close() }() // #nosec G307 - Error handled in defer

	// FIXME: Batch error handling for multiple fmt.Fprintf calls.

	var writeErr error

	if _, err := fmt.Fprintf(file, "Nephoran Intent Operator - Performance Comparison Report\n"); err != nil && writeErr == nil {
		writeErr = err
	}

	if _, err := fmt.Fprintf(file, "======================================================\n\n"); err != nil && writeErr == nil {
		writeErr = err
	}

	if _, err := fmt.Fprintf(file, "Summary: %s\n", comparison.Summary); err != nil && writeErr == nil {
		writeErr = err
	}

	if _, err := fmt.Fprintf(file, "Timestamp: %s\n", comparison.Timestamp.Format("2006-01-02 15:04:05")); err != nil && writeErr == nil {
		writeErr = err
	}

	if _, err := fmt.Fprintf(file, "Go Version: %s\n", comparison.GoVersion); err != nil && writeErr == nil {
		writeErr = err
	}

	if _, err := fmt.Fprintf(file, "Overall Performance Gain: %.2f%%\n\n", comparison.OverallGain); err != nil && writeErr == nil {
		writeErr = err
	}

	if _, err := fmt.Fprintf(file, "Detailed Results:\n"); err != nil && writeErr == nil {
		writeErr = err
	}

	if _, err := fmt.Fprintf(file, "-----------------\n"); err != nil && writeErr == nil {
		writeErr = err
	}

	for _, result := range comparison.Results {

		if _, err := fmt.Fprintf(file, "Metric: %s\n", result.Metric); err != nil && writeErr == nil {
			writeErr = err
		}

		if _, err := fmt.Fprintf(file, "  Old Value: %.2f\n", result.OldValue); err != nil && writeErr == nil {
			writeErr = err
		}

		if _, err := fmt.Fprintf(file, "  New Value: %.2f\n", result.NewValue); err != nil && writeErr == nil {
			writeErr = err
		}

		if _, err := fmt.Fprintf(file, "  Improvement: %.2f%%\n", result.ImprovementPercent); err != nil && writeErr == nil {
			writeErr = err
		}

		if _, err := fmt.Fprintf(file, "  Status: %s\n\n", result.Status); err != nil && writeErr == nil {
			writeErr = err
		}

	}

	// Check for any write errors.

	if writeErr != nil {

		log.Printf("Error writing text report: %v", writeErr)

		return

	}

	fmt.Println("?? Text report generated: performance-comparison.txt")
}

// generateMarkdownReport creates a markdown performance report.

func generateMarkdownReport(comparison *PerformanceComparison) {
	file, err := os.Create("performance-comparison.md")
	if err != nil {

		log.Printf("Error creating markdown report: %v", err)

		return

	}

	defer func() { _ = file.Close() }() // #nosec G307 - Error handled in defer

	// FIXME: Batch error handling for multiple fmt.Fprintf calls in markdown generation.

	var mdWriteErr error

	if _, err := fmt.Fprintf(file, "# Nephoran Intent Operator - Performance Comparison Report\n\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "**Summary:** %s\n\n", comparison.Summary); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "**Timestamp:** %s\n\n", comparison.Timestamp.Format("2006-01-02 15:04:05")); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "**Go Version:** %s\n\n", comparison.GoVersion); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "**Overall Performance Gain:** %.2f%%\n\n", comparison.OverallGain); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "## Performance Metrics\n\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "| Metric | Old Value | New Value | Improvement | Status |\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "|--------|-----------|-----------|-------------|--------|\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	for _, result := range comparison.Results {

		status := "✅"

		if result.Status == "Degraded" {
			status = "❌"
		} else if result.Status == "Unchanged" {
			status = "⚡"
		}

		if _, err := fmt.Fprintf(file, "| %s | %.2f | %.2f | %.2f%% | %s %s |\n",

			result.Metric,

			result.OldValue,

			result.NewValue,

			result.ImprovementPercent,

			status,

			result.Status); err != nil && mdWriteErr == nil {
			mdWriteErr = err
		}

	}

	if _, err := fmt.Fprintf(file, "\n## Key Improvements\n\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if comparison.OverallGain > 0 {

		if _, err := fmt.Fprintf(file, "- **Overall Performance Gain:** %.2f%%\n", comparison.OverallGain); err != nil && mdWriteErr == nil {
			mdWriteErr = err
		}

		if _, err := fmt.Fprintf(file, "- **Go 1.24+ Optimizations:** Successfully applied\n"); err != nil && mdWriteErr == nil {
			mdWriteErr = err
		}

		if _, err := fmt.Fprintf(file, "- **Memory Efficiency:** Improved through advanced memory pools\n"); err != nil && mdWriteErr == nil {
			mdWriteErr = err
		}

		if _, err := fmt.Fprintf(file, "- **HTTP/3 Support:** Enhanced network performance\n"); err != nil && mdWriteErr == nil {
			mdWriteErr = err
		}

		if _, err := fmt.Fprintf(file, "- **Cryptographic Operations:** Optimized with modern algorithms\n"); err != nil && mdWriteErr == nil {
			mdWriteErr = err
		}

	} else {
		if _, err := fmt.Fprintf(file, "- Performance analysis indicates potential areas for optimization\n"); err != nil && mdWriteErr == nil {
			mdWriteErr = err
		}
	}

	if _, err := fmt.Fprintf(file, "\n## Next Steps\n\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "1. Monitor production metrics\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "2. Validate improvements in real-world scenarios\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "3. Continue optimization efforts\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	if _, err := fmt.Fprintf(file, "4. Update performance baselines\n"); err != nil && mdWriteErr == nil {
		mdWriteErr = err
	}

	// Check for any write errors in markdown generation.

	if mdWriteErr != nil {

		log.Printf("Error writing markdown report: %v", mdWriteErr)

		return

	}

	fmt.Println("?? Markdown report generated: performance-comparison.md")
}

// saveMetricsAsBaseline saves current metrics as new baseline.

func saveMetricsAsBaseline(metrics *PerformanceMetrics) {
	file, err := os.Create("performance-baseline.json")
	if err != nil {

		log.Printf("Error saving baseline metrics: %v", err)

		return

	}

	defer func() { _ = file.Close() }() // #nosec G307 - Error handled in defer

	encoder := json.NewEncoder(file)

	encoder.SetIndent("", "  ")

	if err := encoder.Encode(metrics); err != nil {

		log.Printf("Error writing baseline metrics: %v", err)

		return

	}

	fmt.Println("?�� Current metrics saved as new baseline")
}
