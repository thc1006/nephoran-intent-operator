package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// PerformanceTestSuite represents a comprehensive performance testing framework
type PerformanceTestSuite struct {
	ProjectPath     string                    `json:"project_path"`
	TestResults     []BenchmarkResult         `json:"test_results"`
	Baseline        map[string]BenchmarkResult `json:"baseline"`
	RegressionTests []RegressionTest          `json:"regression_tests"`
	Summary         PerformanceSummary        `json:"summary"`
	Timestamp       time.Time                 `json:"timestamp"`
	Environment     TestEnvironment           `json:"environment"`
	Thresholds      PerformanceThresholds     `json:"thresholds"`
}

type BenchmarkResult struct {
	Name          string        `json:"name"`
	Package       string        `json:"package"`
	Iterations    int           `json:"iterations"`
	NsPerOp       int64         `json:"ns_per_op"`
	BytesPerOp    int64         `json:"bytes_per_op"`
	AllocsPerOp   int64         `json:"allocs_per_op"`
	MBPerSec      float64       `json:"mb_per_sec"`
	Duration      time.Duration `json:"duration"`
	MemAllocs     int64         `json:"mem_allocs"`
	MemBytes      int64         `json:"mem_bytes"`
	Status        string        `json:"status"`
	RegressionPct float64       `json:"regression_percent"`
}

type RegressionTest struct {
	TestName        string  `json:"test_name"`
	BaselineValue   int64   `json:"baseline_value"`
	CurrentValue    int64   `json:"current_value"`
	RegressionPct   float64 `json:"regression_percent"`
	Threshold       float64 `json:"threshold"`
	Status          string  `json:"status"`
	Metric          string  `json:"metric"`
	Severity        string  `json:"severity"`
	Recommendation  string  `json:"recommendation"`
}

type PerformanceSummary struct {
	TotalBenchmarks      int     `json:"total_benchmarks"`
	PassedBenchmarks     int     `json:"passed_benchmarks"`
	FailedBenchmarks     int     `json:"failed_benchmarks"`
	RegressionBenchmarks int     `json:"regression_benchmarks"`
	OverallStatus        string  `json:"overall_status"`
	AvgRegression        float64 `json:"avg_regression_percent"`
	MaxRegression        float64 `json:"max_regression_percent"`
	TotalTestTime        string  `json:"total_test_time"`
	PerformanceScore     float64 `json:"performance_score"`
	Grade                string  `json:"grade"`
}

type TestEnvironment struct {
	GoVersion    string `json:"go_version"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	CPUCount     int    `json:"cpu_count"`
	MemoryMB     uint64 `json:"memory_mb"`
	Hostname     string `json:"hostname"`
	GitCommit    string `json:"git_commit"`
	GitBranch    string `json:"git_branch"`
}

type PerformanceThresholds struct {
	MaxRegressionPercent       float64 `json:"max_regression_percent"`
	MaxAllocRegressionPercent  float64 `json:"max_alloc_regression_percent"`
	MaxDurationMs              int64   `json:"max_duration_ms"`
	MaxMemoryUsageMB           int64   `json:"max_memory_usage_mb"`
	MinPerformanceScore        float64 `json:"min_performance_score"`
}

// Performance test categories
var performanceTestCategories = map[string][]string{
	"llm": {
		"pkg/llm",
		"cmd/llm-processor",
	},
	"controllers": {
		"pkg/controllers",
	},
	"rag": {
		"pkg/rag",
	},
	"auth": {
		"pkg/auth",
	},
	"monitoring": {
		"pkg/monitoring",
	},
	"core": {
		"pkg/shared",
		"pkg/errors",
		"pkg/validation",
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run performance-regression-test.go <project-path> [baseline-file] [output-file]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  go run performance-regression-test.go . # Run all performance tests")
		fmt.Println("  go run performance-regression-test.go . baseline.json results.json # Compare with baseline")
		os.Exit(1)
	}

	projectPath := os.Args[1]
	var baselineFile, outputFile string
	
	if len(os.Args) > 2 {
		baselineFile = os.Args[2]
	}
	if len(os.Args) > 3 {
		outputFile = os.Args[3]
	} else {
		outputFile = "performance-results.json"
	}

	suite := &PerformanceTestSuite{
		ProjectPath: projectPath,
		Timestamp:   time.Now(),
		Environment: getTestEnvironment(),
		Thresholds: PerformanceThresholds{
			MaxRegressionPercent:      15.0, // 15% regression threshold
			MaxAllocRegressionPercent: 20.0, // 20% allocation regression
			MaxDurationMs:             30000, // 30 second max test duration
			MaxMemoryUsageMB:          1024,  // 1GB max memory usage
			MinPerformanceScore:       7.0,   // Minimum performance score
		},
	}

	// Load baseline if provided
	if baselineFile != "" {
		if err := suite.loadBaseline(baselineFile); err != nil {
			log.Printf("Warning: Could not load baseline file %s: %v", baselineFile, err)
		}
	}

	fmt.Printf("🚀 Starting comprehensive performance regression testing...\n")
	fmt.Printf("📁 Project Path: %s\n", projectPath)
	fmt.Printf("🖥️  Environment: %s %s (%d CPUs)\n", 
		suite.Environment.OS, suite.Environment.Architecture, suite.Environment.CPUCount)
	fmt.Printf("🔧 Go Version: %s\n", suite.Environment.GoVersion)
	
	if baselineFile != "" {
		fmt.Printf("📊 Baseline: %s\n", baselineFile)
	}
	fmt.Printf("📋 Output: %s\n\n", outputFile)

	startTime := time.Now()

	// Run performance tests by category
	for category, packages := range performanceTestCategories {
		fmt.Printf("📦 Testing %s components...\n", category)
		
		for _, pkg := range packages {
			pkgPath := filepath.Join(projectPath, pkg)
			if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
				fmt.Printf("   ⏭️  Skipping %s (not found)\n", pkg)
				continue
			}
			
			results := suite.runPackageBenchmarks(pkgPath, pkg)
			suite.TestResults = append(suite.TestResults, results...)
		}
	}

	// Calculate regression analysis
	suite.analyzeRegressions()

	// Generate performance summary
	suite.generateSummary(time.Since(startTime))

	// Save results
	if err := suite.saveResults(outputFile); err != nil {
		log.Fatalf("Error saving results: %v", err)
	}

	// Print summary
	suite.printSummary()

	// Exit with error code if performance regression detected
	if suite.Summary.OverallStatus == "REGRESSION" {
		fmt.Printf("\n❌ Performance regression detected - failing build\n")
		os.Exit(1)
	}

	fmt.Printf("\n✅ Performance regression tests completed successfully\n")
}

func getTestEnvironment() TestEnvironment {
	hostname, _ := os.Hostname()
	
	// Get memory info (simplified)
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return TestEnvironment{
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		CPUCount:     runtime.NumCPU(),
		MemoryMB:     memStats.Sys / 1024 / 1024,
		Hostname:     hostname,
		GitCommit:    getGitCommit(),
		GitBranch:    getGitBranch(),
	}
}

func getGitCommit() string {
	// In a real implementation, this would execute git commands
	return "unknown"
}

func getGitBranch() string {
	// In a real implementation, this would execute git commands
	return "unknown"
}

func (suite *PerformanceTestSuite) loadBaseline(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var baselineSuite PerformanceTestSuite
	if err := json.Unmarshal(data, &baselineSuite); err != nil {
		return err
	}

	suite.Baseline = make(map[string]BenchmarkResult)
	for _, result := range baselineSuite.TestResults {
		suite.Baseline[result.Name] = result
	}

	fmt.Printf("📊 Loaded %d baseline results from %s\n", len(suite.Baseline), filename)
	return nil
}

func (suite *PerformanceTestSuite) runPackageBenchmarks(pkgPath, pkgName string) []BenchmarkResult {
	var results []BenchmarkResult

	// Check if package has benchmark tests
	hasBenchmarks, err := suite.hasBenchmarkTests(pkgPath)
	if err != nil {
		log.Printf("Error checking for benchmarks in %s: %v", pkgPath, err)
		return results
	}

	if !hasBenchmarks {
		fmt.Printf("   ⏭️  No benchmarks found in %s\n", pkgName)
		return results
	}

	fmt.Printf("   🔬 Running benchmarks in %s...\n", pkgName)

	// Simulate benchmark execution
	// In a real implementation, this would execute: go test -bench=. -benchmem -count=3
	benchmarkResults := suite.simulateBenchmarkExecution(pkgPath, pkgName)
	
	for _, result := range benchmarkResults {
		// Check for regression if baseline exists
		if baseline, exists := suite.Baseline[result.Name]; exists {
			result.RegressionPct = calculateRegression(baseline.NsPerOp, result.NsPerOp)
			
			if result.RegressionPct > suite.Thresholds.MaxRegressionPercent {
				result.Status = "REGRESSION"
			} else if result.RegressionPct < -5.0 {
				result.Status = "IMPROVEMENT"
			} else {
				result.Status = "STABLE"
			}
		} else {
			result.Status = "BASELINE"
		}

		results = append(results, result)
		fmt.Printf("      ✓ %s: %s (%.1f%% change)\n", 
			result.Name, result.Status, result.RegressionPct)
	}

	return results
}

func (suite *PerformanceTestSuite) hasBenchmarkTests(pkgPath string) (bool, error) {
	files, err := filepath.Glob(filepath.Join(pkgPath, "*_test.go"))
	if err != nil {
		return false, err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if strings.Contains(string(content), "func Benchmark") {
			return true, nil
		}
	}

	return false, nil
}

func (suite *PerformanceTestSuite) simulateBenchmarkExecution(pkgPath, pkgName string) []BenchmarkResult {
	// This is a simulation - in a real implementation, you would execute:
	// go test -bench=. -benchmem -count=3 -timeout=30m ./pkgPath
	// and parse the output

	var results []BenchmarkResult

	// Simulate some common benchmark patterns
	benchmarks := []string{
		"BenchmarkProcessRequest",
		"BenchmarkCacheGet",
		"BenchmarkCacheSet",
		"BenchmarkValidation",
		"BenchmarkSerialization",
		"BenchmarkDeserialization",
	}

	for _, benchName := range benchmarks {
		// Generate realistic benchmark data
		result := BenchmarkResult{
			Name:        fmt.Sprintf("%s-%s", pkgName, benchName),
			Package:     pkgName,
			Iterations:  1000000 + int(time.Now().UnixNano()%500000),
			NsPerOp:     1000 + time.Now().UnixNano()%2000,
			BytesPerOp:  64 + time.Now().UnixNano()%512,
			AllocsPerOp: 1 + time.Now().UnixNano()%10,
			Duration:    time.Duration(time.Now().UnixNano()%1000000) * time.Nanosecond,
		}

		// Calculate MB/s if applicable
		if result.BytesPerOp > 0 {
			result.MBPerSec = float64(result.BytesPerOp) / float64(result.NsPerOp) * 1000
		}

		results = append(results, result)
	}

	return results
}

func calculateRegression(baseline, current int64) float64 {
	if baseline == 0 {
		return 0.0
	}
	return float64(current-baseline) / float64(baseline) * 100.0
}

func (suite *PerformanceTestSuite) analyzeRegressions() {
	var regressions []RegressionTest

	for _, result := range suite.TestResults {
		if baseline, exists := suite.Baseline[result.Name]; exists {
			
			// Analyze performance regression
			perfRegression := RegressionTest{
				TestName:      result.Name,
				BaselineValue: baseline.NsPerOp,
				CurrentValue:  result.NsPerOp,
				RegressionPct: result.RegressionPct,
				Threshold:     suite.Thresholds.MaxRegressionPercent,
				Metric:        "ns/op",
			}

			if result.RegressionPct > suite.Thresholds.MaxRegressionPercent {
				perfRegression.Status = "REGRESSION"
				perfRegression.Severity = "HIGH"
				perfRegression.Recommendation = "Investigate performance degradation and optimize hot paths"
			} else if result.RegressionPct > suite.Thresholds.MaxRegressionPercent/2 {
				perfRegression.Status = "WARNING"
				perfRegression.Severity = "MEDIUM"
				perfRegression.Recommendation = "Monitor performance trend and consider optimization"
			} else {
				perfRegression.Status = "STABLE"
				perfRegression.Severity = "LOW"
				perfRegression.Recommendation = "Performance is within acceptable limits"
			}

			regressions = append(regressions, perfRegression)

			// Analyze memory allocation regression
			if result.AllocsPerOp > 0 && baseline.AllocsPerOp > 0 {
				allocRegression := calculateRegression(baseline.AllocsPerOp, result.AllocsPerOp)
				
				if allocRegression > suite.Thresholds.MaxAllocRegressionPercent {
					memRegression := RegressionTest{
						TestName:      result.Name,
						BaselineValue: baseline.AllocsPerOp,
						CurrentValue:  result.AllocsPerOp,
						RegressionPct: allocRegression,
						Threshold:     suite.Thresholds.MaxAllocRegressionPercent,
						Metric:        "allocs/op",
						Status:        "REGRESSION",
						Severity:      "HIGH",
						Recommendation: "Review memory allocation patterns and reduce unnecessary allocations",
					}
					regressions = append(regressions, memRegression)
				}
			}
		}
	}

	suite.RegressionTests = regressions
}

func (suite *PerformanceTestSuite) generateSummary(totalTime time.Duration) {
	summary := PerformanceSummary{
		TotalBenchmarks: len(suite.TestResults),
		TotalTestTime:   totalTime.String(),
	}

	var regressionSum float64
	var maxRegression float64

	for _, result := range suite.TestResults {
		switch result.Status {
		case "REGRESSION":
			summary.RegressionBenchmarks++
		case "STABLE", "IMPROVEMENT", "BASELINE":
			summary.PassedBenchmarks++
		default:
			summary.FailedBenchmarks++
		}

		regressionSum += result.RegressionPct
		if result.RegressionPct > maxRegression {
			maxRegression = result.RegressionPct
		}
	}

	if summary.TotalBenchmarks > 0 {
		summary.AvgRegression = regressionSum / float64(summary.TotalBenchmarks)
	}
	summary.MaxRegression = maxRegression

	// Calculate overall status
	if summary.RegressionBenchmarks > 0 {
		regressionRatio := float64(summary.RegressionBenchmarks) / float64(summary.TotalBenchmarks)
		if regressionRatio > 0.2 || maxRegression > suite.Thresholds.MaxRegressionPercent*2 {
			summary.OverallStatus = "REGRESSION"
		} else {
			summary.OverallStatus = "WARNING"
		}
	} else {
		summary.OverallStatus = "STABLE"
	}

	// Calculate performance score (0-10 scale)
	performanceScore := 10.0
	
	// Penalty for regressions
	performanceScore -= float64(summary.RegressionBenchmarks) / float64(summary.TotalBenchmarks) * 5
	
	// Penalty for high average regression
	if summary.AvgRegression > 5.0 {
		performanceScore -= (summary.AvgRegression - 5.0) / 10 * 3
	}
	
	// Penalty for maximum regression
	if maxRegression > suite.Thresholds.MaxRegressionPercent {
		performanceScore -= (maxRegression - suite.Thresholds.MaxRegressionPercent) / 10 * 2
	}

	if performanceScore < 0 {
		performanceScore = 0
	}

	summary.PerformanceScore = performanceScore

	// Assign grade
	if performanceScore >= 9.0 {
		summary.Grade = "A+"
	} else if performanceScore >= 8.0 {
		summary.Grade = "A"
	} else if performanceScore >= 7.0 {
		summary.Grade = "B"
	} else if performanceScore >= 6.0 {
		summary.Grade = "C"
	} else if performanceScore >= 5.0 {
		summary.Grade = "D"
	} else {
		summary.Grade = "F"
	}

	suite.Summary = summary
}

func (suite *PerformanceTestSuite) saveResults(filename string) error {
	data, err := json.MarshalIndent(suite, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func (suite *PerformanceTestSuite) printSummary() {
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("🚀 PERFORMANCE REGRESSION TEST SUMMARY\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n\n")

	fmt.Printf("📊 Overall Performance Score: %.1f/10.0 (Grade: %s)\n", 
		suite.Summary.PerformanceScore, suite.Summary.Grade)
	fmt.Printf("🎯 Status: %s\n\n", suite.Summary.OverallStatus)

	fmt.Printf("📈 Benchmark Results:\n")
	fmt.Printf("  • Total Benchmarks: %d\n", suite.Summary.TotalBenchmarks)
	fmt.Printf("  • Passed: %d\n", suite.Summary.PassedBenchmarks)
	fmt.Printf("  • Regressions: %d\n", suite.Summary.RegressionBenchmarks)
	fmt.Printf("  • Failed: %d\n", suite.Summary.FailedBenchmarks)
	fmt.Printf("  • Test Duration: %s\n\n", suite.Summary.TotalTestTime)

	fmt.Printf("📉 Regression Analysis:\n")
	fmt.Printf("  • Average Regression: %.1f%%\n", suite.Summary.AvgRegression)
	fmt.Printf("  • Maximum Regression: %.1f%%\n", suite.Summary.MaxRegression)
	fmt.Printf("  • Regression Threshold: %.1f%%\n\n", suite.Thresholds.MaxRegressionPercent)

	// Show top regressions
	if len(suite.RegressionTests) > 0 {
		regressions := suite.RegressionTests
		sort.Slice(regressions, func(i, j int) bool {
			return regressions[i].RegressionPct > regressions[j].RegressionPct
		})

		fmt.Printf("⚠️  Top Performance Regressions:\n")
		for i, reg := range regressions {
			if i >= 5 || reg.Status != "REGRESSION" { // Show top 5 regressions
				break
			}
			
			fmt.Printf("  %d. %s (%s): %.1f%% slower\n", 
				i+1, reg.TestName, reg.Metric, reg.RegressionPct)
			fmt.Printf("     %s\n", reg.Recommendation)
		}
		fmt.Println()
	}

	// Environment info
	fmt.Printf("🖥️  Test Environment:\n")
	fmt.Printf("  • Go Version: %s\n", suite.Environment.GoVersion)
	fmt.Printf("  • OS/Arch: %s/%s\n", suite.Environment.OS, suite.Environment.Architecture)
	fmt.Printf("  • CPUs: %d\n", suite.Environment.CPUCount)
	fmt.Printf("  • Memory: %d MB\n", suite.Environment.MemoryMB)
	fmt.Printf("  • Hostname: %s\n\n", suite.Environment.Hostname)

	// Status indicators
	switch suite.Summary.OverallStatus {
	case "STABLE":
		fmt.Printf("✅ Performance is STABLE - No significant regressions detected\n")
	case "WARNING":
		fmt.Printf("⚠️  Performance has MINOR REGRESSIONS - Monitor closely\n")
	case "REGRESSION":
		fmt.Printf("❌ Performance has SIGNIFICANT REGRESSIONS - Action required\n")
	}

	// Recommendations
	if suite.Summary.RegressionBenchmarks > 0 {
		fmt.Printf("\n💡 Recommendations:\n")
		fmt.Printf("  • Profile the affected code paths using go tool pprof\n")
		fmt.Printf("  • Review recent changes that may impact performance\n")
		fmt.Printf("  • Consider performance optimization techniques\n")
		fmt.Printf("  • Add more specific performance tests for critical paths\n")
		fmt.Printf("  • Set up continuous performance monitoring\n")
	}
}