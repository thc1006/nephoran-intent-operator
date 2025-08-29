package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
)

// TestEvent represents a single test event from go test -json output.
type TestEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed,omitempty"`
	Output  string  `json:"Output,omitempty"`
}

// TestResult represents the outcome of a single test run.
type TestResult struct {
	Package string
	Test    string
	Pass    bool
	Skip    bool
	Elapsed float64
	Output  []string
}

// FlakeStats contains flakiness statistics for a test.
type FlakeStats struct {
	TestName        string   `json:"test_name"`
	Package         string   `json:"package"`
	TotalRuns       int      `json:"total_runs"`
	PassCount       int      `json:"pass_count"`
	FailCount       int      `json:"fail_count"`
	SkipCount       int      `json:"skip_count"`
	FlakeRate       float64  `json:"flake_rate"`
	IsFlaky         bool     `json:"is_flaky"`
	AvgElapsed      float64  `json:"avg_elapsed_seconds"`
	MinElapsed      float64  `json:"min_elapsed_seconds"`
	MaxElapsed      float64  `json:"max_elapsed_seconds"`
	FailurePatterns []string `json:"failure_patterns,omitempty"`
}

// FlakeReport contains the complete flakiness analysis.
type FlakeReport struct {
	TotalTests  int                   `json:"total_tests"`
	FlakyTests  int                   `json:"flaky_tests"`
	FlakeRate   float64               `json:"overall_flake_rate"`
	TestResults map[string]FlakeStats `json:"test_results"`
	Summary     Summary               `json:"summary"`
}

// Summary provides high-level statistics.
type Summary struct {
	HighFlakeTests      []string `json:"high_flake_tests"`     // > 50% flake rate
	MediumFlakeTests    []string `json:"medium_flake_tests"`   // 10-50% flake rate
	LowFlakeTests       []string `json:"low_flake_tests"`      // < 10% flake rate
	ConsistentlyFailing []string `json:"consistently_failing"` // Always fail
	ConsistentlyPassing []string `json:"consistently_passing"` // Always pass
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <test-result-file1.json> [test-result-file2.json] ...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nThis tool analyzes go test -json output to detect flaky tests.\n")
		fmt.Fprintf(os.Stderr, "Example: go test -json -count=5 ./... > results.json && flaky-detector results.json\n")
		os.Exit(1)
	}

	allResults := make(map[string][]TestResult)

	// Parse all result files.
	for _, filename := range os.Args[1:] {
		results, err := parseTestResults(filename)
		if err != nil {
			log.Fatalf("Error parsing %s: %v", filename, err)
		}

		// Group results by test name.
		for _, result := range results {
			testKey := result.Package + "." + result.Test
			allResults[testKey] = append(allResults[testKey], result)
		}
	}

	// Generate flake report.
	report := analyzeFlakiness(allResults)

	// Output report as JSON.
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Error generating report: %v", err)
	}

	fmt.Println(string(reportJSON))

	// Exit with error code if flaky tests found.
	if report.FlakyTests > 0 {
		fmt.Fprintf(os.Stderr, "\n❌ Found %d flaky tests (%.2f%% flake rate)\n",
			report.FlakyTests, report.FlakeRate*100)
		os.Exit(1)
	}
	
	fmt.Fprintf(os.Stderr, "\n✅ No flaky tests detected across %d tests\n", report.TotalTests)
}

// parseTestResults parses go test -json output from a file.
func parseTestResults(filename string) ([]TestResult, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	results := make(map[string]*TestResult)

	for decoder.More() {
		var event TestEvent
		if err := decoder.Decode(&event); err != nil {
			continue // Skip malformed lines
		}

		// Skip non-test events.
		if event.Test == "" || event.Package == "" {
			continue
		}

		testKey := event.Package + "." + event.Test

		if results[testKey] == nil {
			results[testKey] = &TestResult{
				Package: event.Package,
				Test:    event.Test,
				Output:  make([]string, 0, 100),
			}
		}

		result := results[testKey]

		switch event.Action {
		case "pass":
			result.Pass = true
			result.Elapsed = event.Elapsed
		case "fail":
			result.Pass = false
			result.Elapsed = event.Elapsed
		case "skip":
			result.Skip = true
			result.Elapsed = event.Elapsed
		case "output":
			if strings.TrimSpace(event.Output) != "" {
				result.Output = append(result.Output, strings.TrimSpace(event.Output))
			}
		}
	}

	// Convert map to slice.
	var testResults []TestResult
	for _, result := range results {
		testResults = append(testResults, *result)
	}

	return testResults, nil
}

// analyzeFlakiness analyzes test results to identify flaky tests.
func analyzeFlakiness(allResults map[string][]TestResult) FlakeReport {
	report := FlakeReport{
		TestResults: make(map[string]FlakeStats),
		Summary: Summary{
			HighFlakeTests:      make([]string, 0, 20),
			MediumFlakeTests:    make([]string, 0, 50),
			LowFlakeTests:       make([]string, 0, 100),
			ConsistentlyFailing: make([]string, 0, 30),
			ConsistentlyPassing: make([]string, 0, 200),
		},
	}

	totalFlakyTests := 0

	for testName, results := range allResults {
		if len(results) < 2 {
			continue // Need multiple runs to detect flakiness
		}

		stats := calculateTestStats(testName, results)
		report.TestResults[testName] = stats

		if stats.IsFlaky {
			totalFlakyTests++

			// Categorize by flake rate.
			switch {
			case stats.FlakeRate > 0.5:
				report.Summary.HighFlakeTests = append(report.Summary.HighFlakeTests, testName)
			case stats.FlakeRate >= 0.1:
				report.Summary.MediumFlakeTests = append(report.Summary.MediumFlakeTests, testName)
			default:
				report.Summary.LowFlakeTests = append(report.Summary.LowFlakeTests, testName)
			}
		} else {
			// Categorize consistent tests.
			if stats.PassCount == 0 {
				report.Summary.ConsistentlyFailing = append(report.Summary.ConsistentlyFailing, testName)
			} else if stats.FailCount == 0 && stats.SkipCount == 0 {
				report.Summary.ConsistentlyPassing = append(report.Summary.ConsistentlyPassing, testName)
			}
		}
	}

	report.TotalTests = len(allResults)
	report.FlakyTests = totalFlakyTests
	if report.TotalTests > 0 {
		report.FlakeRate = float64(totalFlakyTests) / float64(report.TotalTests)
	}

	// Sort summary lists for consistent output.
	sort.Strings(report.Summary.HighFlakeTests)
	sort.Strings(report.Summary.MediumFlakeTests)
	sort.Strings(report.Summary.LowFlakeTests)
	sort.Strings(report.Summary.ConsistentlyFailing)
	sort.Strings(report.Summary.ConsistentlyPassing)

	return report
}

// calculateTestStats calculates flakiness statistics for a single test.
func calculateTestStats(testName string, results []TestResult) FlakeStats {
	stats := FlakeStats{
		TestName:        testName,
		TotalRuns:       len(results),
		FailurePatterns: make([]string, 0, 10),
	}

	if len(results) > 0 {
		stats.Package = results[0].Package
	}

	var totalElapsed float64
	minElapsed := float64(99999)
	maxElapsed := float64(0)
	failureOutputs := make(map[string]int)

	for _, result := range results {
		if result.Skip {
			stats.SkipCount++
		} else if result.Pass {
			stats.PassCount++
		} else {
			stats.FailCount++

			// Collect failure patterns.
			for _, output := range result.Output {
				if strings.Contains(output, "FAIL") ||
					strings.Contains(output, "ERROR") ||
					strings.Contains(output, "panic") {
					failureOutputs[output]++
				}
			}
		}

		totalElapsed += result.Elapsed
		if result.Elapsed < minElapsed {
			minElapsed = result.Elapsed
		}
		if result.Elapsed > maxElapsed {
			maxElapsed = result.Elapsed
		}
	}

	// Calculate timing statistics.
	if stats.TotalRuns > 0 {
		stats.AvgElapsed = totalElapsed / float64(stats.TotalRuns)
	}
	if minElapsed != 99999 {
		stats.MinElapsed = minElapsed
	}
	stats.MaxElapsed = maxElapsed

	// Calculate flake rate (non-deterministic outcomes).
	nonSkippedRuns := stats.TotalRuns - stats.SkipCount
	if nonSkippedRuns > 1 {
		// A test is flaky if it has both passes and failures.
		if stats.PassCount > 0 && stats.FailCount > 0 {
			stats.IsFlaky = true
			stats.FlakeRate = float64(min(stats.PassCount, stats.FailCount)) / float64(nonSkippedRuns)
		}
	}

	// Extract common failure patterns.
	for pattern, count := range failureOutputs {
		if count > 1 && len(pattern) < 200 { // Only include recurring, concise patterns
			stats.FailurePatterns = append(stats.FailurePatterns, pattern)
		}
	}

	// Sort failure patterns by frequency (most common first).
	sort.Slice(stats.FailurePatterns, func(i, j int) bool {
		return failureOutputs[stats.FailurePatterns[i]] > failureOutputs[stats.FailurePatterns[j]]
	})

	// Limit to top 5 patterns.
	if len(stats.FailurePatterns) > 5 {
		stats.FailurePatterns = stats.FailurePatterns[:5]
	}

	return stats
}

// Helper function for min (Go < 1.21 compatibility).
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
