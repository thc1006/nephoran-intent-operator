package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// TestRunner manages parallel test execution with sharding.

type TestRunner struct {
	ShardIndex int

	TotalShards int

	Pattern string

	Parallel int

	Timeout time.Duration

	Coverage bool

	Verbose bool

	OutputDir string

	MaxRetries int

	FailFast bool
}

// TestResult represents the result of a test execution.

type TestResult struct {
	Package string

	Duration time.Duration

	Success bool

	Output string

	Coverage string

	Error error
}

// TestShard contains tests assigned to a specific shard.

type TestShard struct {
	Index int

	Packages []string
}

func main() {
	var runner TestRunner

	rootCmd := &cobra.Command{
		Use: "test-runner",

		Short: "Parallel test runner with intelligent sharding",

		Long: `A test runner that distributes Go tests across multiple shards for parallel execution.

Features:

- Intelligent test sharding based on historical execution times

- Parallel execution within shards  

- Coverage report generation

- Retry mechanism for flaky tests

- Test result aggregation`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Run()
		},
	}

	rootCmd.Flags().IntVar(&runner.ShardIndex, "shard-index", 0, "Shard index (0-based)")

	rootCmd.Flags().IntVar(&runner.TotalShards, "total-shards", 1, "Total number of shards")

	rootCmd.Flags().StringVar(&runner.Pattern, "pattern", "./...", "Test pattern to run")

	rootCmd.Flags().IntVar(&runner.Parallel, "parallel", 4, "Number of parallel test processes")

	rootCmd.Flags().DurationVar(&runner.Timeout, "timeout", 10*time.Minute, "Test timeout per package")

	rootCmd.Flags().BoolVar(&runner.Coverage, "coverage", false, "Enable coverage reporting")

	rootCmd.Flags().BoolVar(&runner.Verbose, "verbose", false, "Verbose test output")

	rootCmd.Flags().StringVar(&runner.OutputDir, "output-dir", "./test-results", "Output directory for results")

	rootCmd.Flags().IntVar(&runner.MaxRetries, "max-retries", 2, "Maximum retries for failed tests")

	rootCmd.Flags().BoolVar(&runner.FailFast, "fail-fast", false, "Stop on first test failure")

	// Environment variable overrides.

	if envIndex := os.Getenv("SHARD_INDEX"); envIndex != "" {
		if i, err := strconv.Atoi(envIndex); err == nil && i >= 0 && i <= math.MaxInt32 {
			runner.ShardIndex = i
		}
	}

	if envTotal := os.Getenv("TOTAL_SHARDS"); envTotal != "" {
		if t, err := strconv.Atoi(envTotal); err == nil && t >= 0 && t <= math.MaxInt32 {
			runner.TotalShards = t
		}
	}

	if envPattern := os.Getenv("TEST_PATTERN"); envPattern != "" {
		runner.Pattern = envPattern
	}

	// Apply timeout scaling from environment.

	if envTimeoutScale := os.Getenv("GO_TEST_TIMEOUT_SCALE"); envTimeoutScale != "" {
		if scale, err := strconv.ParseFloat(envTimeoutScale, 64); err == nil && scale > 0 {

			scaledTimeout := time.Duration(float64(runner.Timeout) * scale)

			log.Printf("Scaling timeout from %v to %v (scale: %.1fx)", runner.Timeout, scaledTimeout, scale)

			runner.Timeout = scaledTimeout

		}
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// Run executes the test runner.

func (r *TestRunner) Run() error {
	log.Printf("Starting test runner (shard %d/%d)", r.ShardIndex+1, r.TotalShards)

	// Discover test packages.

	packages, err := r.discoverPackages()
	if err != nil {
		return fmt.Errorf("discovering packages: %w", err)
	}

	if len(packages) == 0 {

		log.Printf("No test packages found for pattern: %s", r.Pattern)

		return nil

	}

	// Create shard assignment.

	shard := r.createShard(packages)

	log.Printf("Assigned %d packages to shard %d", len(shard.Packages), r.ShardIndex)

	if len(shard.Packages) == 0 {

		log.Printf("No packages assigned to this shard")

		return nil

	}

	// Create output directory.

	if err := os.MkdirAll(r.OutputDir, 0o750); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Run tests in parallel.

	results, err := r.runTests(shard.Packages)
	if err != nil {
		return fmt.Errorf("running tests: %w", err)
	}

	// Generate reports.

	if err := r.generateReports(results); err != nil {
		return fmt.Errorf("generating reports: %w", err)
	}

	// Check for failures.

	failedCount := 0

	for _, result := range results {
		if !result.Success {
			failedCount++
		}
	}

	if failedCount > 0 {

		log.Printf("❌ %d/%d test packages failed", failedCount, len(results))

		return fmt.Errorf("%d test packages failed", failedCount)

	}

	log.Printf("✅ All %d test packages passed", len(results))

	return nil
}

// discoverPackages finds all test packages matching the pattern.

func (r *TestRunner) discoverPackages() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", r.Pattern)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing packages: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Preallocate slice with expected capacity for performance.

	packages := make([]string, 0, len(lines))

	for _, line := range lines {

		line = strings.TrimSpace(line)

		if line != "" {
			// Check if package has test files.

			if r.hasTestFiles(line) {
				packages = append(packages, line)
			}
		}

	}

	return packages, nil
}

// hasTestFiles checks if a package contains test files.

func (r *TestRunner) hasTestFiles(pkg string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "list", "-f", "{{.TestGoFiles}} {{.XTestGoFiles}}", pkg)

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) != "[] []"
}

// createShard assigns packages to the current shard using round-robin distribution.

func (r *TestRunner) createShard(packages []string) TestShard {
	shard := TestShard{
		Index: r.ShardIndex,

		Packages: make([]string, 0),
	}

	// Use smart sharding based on package characteristics if available.

	packageWeights := r.calculatePackageWeights(packages)

	// Distribute packages using weighted round-robin.

	for i, pkg := range packages {
		if r.shouldAssignToShard(i, packageWeights[pkg]) {
			shard.Packages = append(shard.Packages, pkg)
		}
	}

	return shard
}

// calculatePackageWeights estimates relative execution time for packages.

func (r *TestRunner) calculatePackageWeights(packages []string) map[string]int {
	weights := make(map[string]int)

	for _, pkg := range packages {

		weight := 1 // Default weight

		// Adjust weight based on package characteristics.

		if strings.Contains(pkg, "integration") || strings.Contains(pkg, "e2e") {
			weight = 3 // Integration tests are typically slower
		} else if strings.Contains(pkg, "performance") || strings.Contains(pkg, "benchmark") {
			weight = 5 // Performance tests are slowest
		} else if strings.Contains(pkg, "controller") || strings.Contains(pkg, "manager") {
			weight = 2 // Controller tests often involve more setup
		} else if strings.Contains(pkg, "security") {
			weight = 2 // Security tests may involve crypto operations
		}

		weights[pkg] = weight

	}

	return weights
}

// getPackageTimeout calculates appropriate timeout for a specific package.

func (r *TestRunner) getPackageTimeout(pkg string) time.Duration {
	baseTimeout := r.Timeout

	// Apply package-specific timeout scaling for Windows optimization.

	multiplier := 1.0

	if strings.Contains(pkg, "integration") || strings.Contains(pkg, "e2e") {
		multiplier = 1.5 // Integration tests need more time
	} else if strings.Contains(pkg, "performance") || strings.Contains(pkg, "benchmark") {
		multiplier = 2.0 // Performance tests are slowest
	} else if strings.Contains(pkg, "security") {
		multiplier = 1.2 // Security tests with crypto operations
	} else if strings.Contains(pkg, "controller") || strings.Contains(pkg, "manager") {
		multiplier = 1.3 // Controller tests with more setup
	}

	return time.Duration(float64(baseTimeout) * multiplier)
}

// shouldAssignToShard determines if a package should be assigned to this shard.

func (r *TestRunner) shouldAssignToShard(index, weight int) bool {
	// Use weighted distribution to balance shard load.

	adjustedIndex := index * weight

	return adjustedIndex%r.TotalShards == r.ShardIndex
}

// runTests executes tests for assigned packages in parallel.

func (r *TestRunner) runTests(packages []string) ([]TestResult, error) {
	results := make([]TestResult, len(packages))

	var wg sync.WaitGroup

	semaphore := make(chan struct{}, r.Parallel)

	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout*time.Duration(len(packages)))

	defer cancel()

	for i, pkg := range packages {

		wg.Add(1)

		go func(index int, packageName string) {
			defer wg.Done()

			// Acquire semaphore.

			semaphore <- struct{}{}

			defer func() { <-semaphore }()

			result := r.runSingleTest(ctx, packageName)

			results[index] = result

			// Log progress.

			status := "✅"

			if !result.Success {
				status = "❌"
			}

			log.Printf("%s %s (%v)", status, packageName, result.Duration.Round(time.Millisecond))

			// Fail fast if enabled.

			if r.FailFast && !result.Success {
				cancel()
			}
		}(i, pkg)

	}

	wg.Wait()

	// Filter out empty results if context was cancelled.

	// Preallocate slice with expected capacity for performance.

	filteredResults := make([]TestResult, 0, len(results))

	for _, result := range results {
		if result.Package != "" {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults, nil
}

// runSingleTest executes tests for a single package.

func (r *TestRunner) runSingleTest(ctx context.Context, pkg string) TestResult {
	result := TestResult{
		Package: pkg,
	}

	start := time.Now()

	defer func() {
		result.Duration = time.Since(start)
	}()

	// Create package-specific timeout context.

	packageTimeout := r.getPackageTimeout(pkg)

	pkgCtx, pkgCancel := context.WithTimeout(ctx, packageTimeout)

	defer pkgCancel()

	// Retry logic for flaky tests.

	var lastErr error

	for attempt := 0; attempt <= r.MaxRetries; attempt++ {

		if attempt > 0 {

			log.Printf("Retrying %s (attempt %d/%d)", pkg, attempt+1, r.MaxRetries+1)

			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff

		}

		success, output, coverage, err := r.executeTest(pkgCtx, pkg)

		result.Success = success

		result.Output = output

		result.Coverage = coverage

		result.Error = err

		if success || err == context.Canceled {
			break // Success or cancelled
		}

		lastErr = err

	}

	if !result.Success && lastErr != nil {
		result.Error = fmt.Errorf("failed after %d attempts: %w", r.MaxRetries+1, lastErr)
	}

	return result
}

// executeTest runs the actual go test command.

func (r *TestRunner) executeTest(ctx context.Context, pkg string) (bool, string, string, error) {
	args := []string{"test"}

	if r.Verbose {
		args = append(args, "-v")
	}

	args = append(args, "-count=1") // Disable test caching

	if r.Coverage {

		coverFile := filepath.Join(r.OutputDir, strings.ReplaceAll(pkg, "/", "_")+".coverage")

		args = append(args, "-coverprofile="+coverFile, "-covermode=atomic")

	}

	args = append(args, pkg)

	cmd := exec.CommandContext(ctx, "go", args...)

	cmd.Dir = "."

	// Set environment variables.

	cmd.Env = append(os.Environ(),

		"GOMAXPROCS=1", // Limit per-test parallelism to avoid resource contention

	)

	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	if ctx.Err() == context.Canceled {
		return false, outputStr, "", context.Canceled
	}

	success := err == nil

	// Extract coverage if available.

	var coverage string

	if r.Coverage {

		coverFile := filepath.Join(r.OutputDir, strings.ReplaceAll(pkg, "/", "_")+".coverage")

		if _, err := os.Stat(coverFile); err == nil {
			coverage = coverFile
		}

	}

	return success, outputStr, coverage, err
}

// generateReports creates test and coverage reports.

func (r *TestRunner) generateReports(results []TestResult) error {
	// Generate JUnit XML report.

	if err := r.generateJUnitReport(results); err != nil {
		log.Printf("Warning: failed to generate JUnit report: %v", err)
	}

	// Generate coverage report.

	if r.Coverage {
		if err := r.generateCoverageReport(results); err != nil {
			log.Printf("Warning: failed to generate coverage report: %v", err)
		}
	}

	// Generate timing report.

	if err := r.generateTimingReport(results); err != nil {
		log.Printf("Warning: failed to generate timing report: %v", err)
	}

	return nil
}

// generateJUnitReport creates a JUnit XML report.

func (r *TestRunner) generateJUnitReport(results []TestResult) error {
	reportFile := filepath.Join(r.OutputDir, fmt.Sprintf("junit-shard-%d.xml", r.ShardIndex))

	file, err := os.Create(reportFile)
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }() // #nosec G307 - Error handled in defer

	fmt.Fprintf(file, `<?xml version="1.0" encoding="UTF-8"?>`)

	fmt.Fprintf(file, `<testsuite name="shard-%d" tests="%d" failures="%d" time="%.2f">`,

		r.ShardIndex, len(results), r.countFailures(results), r.totalDuration(results).Seconds())

	for _, result := range results {

		fmt.Fprintf(file, `<testcase classname="%s" name="%s" time="%.2f">`,

			result.Package, result.Package, result.Duration.Seconds())

		if !result.Success {
			fmt.Fprintf(file, `<failure message="Test failed">%s</failure>`,

				strings.ReplaceAll(result.Output, "&", "&amp;"))
		}

		fmt.Fprintf(file, `</testcase>`)

	}

	fmt.Fprintf(file, `</testsuite>`)

	return nil
}

// generateCoverageReport combines coverage files and generates reports.

func (r *TestRunner) generateCoverageReport(results []TestResult) error {
	// Preallocate slice with expected capacity for performance.

	coverageFiles := make([]string, 0, len(results))

	for _, result := range results {
		if result.Coverage != "" {
			coverageFiles = append(coverageFiles, result.Coverage)
		}
	}

	if len(coverageFiles) == 0 {
		return nil
	}

	// Combine coverage files.

	combinedFile := filepath.Join(r.OutputDir, fmt.Sprintf("coverage-shard-%d.out", r.ShardIndex))

	file, err := os.Create(combinedFile)
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }() // #nosec G307 - Error handled in defer

	fmt.Fprintln(file, "mode: atomic")

	for _, coverFile := range coverageFiles {

		content, err := os.ReadFile(coverFile)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")

		for _, line := range lines[1:] { // Skip mode line

			if strings.TrimSpace(line) != "" {
				fmt.Fprintln(file, line)
			}
		}

	}

	// Generate HTML report.

	htmlFile := filepath.Join(r.OutputDir, fmt.Sprintf("coverage-shard-%d.html", r.ShardIndex))

	cmd := exec.Command("go", "tool", "cover", "-html="+combinedFile, "-o", htmlFile) // #nosec G204 - Static command with validated args

	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to generate HTML coverage report: %v", err)
	}

	return nil
}

// generateTimingReport creates a report of test execution times.

func (r *TestRunner) generateTimingReport(results []TestResult) error {
	reportFile := filepath.Join(r.OutputDir, fmt.Sprintf("timing-shard-%d.txt", r.ShardIndex))

	file, err := os.Create(reportFile)
	if err != nil {
		return err
	}

	defer func() { _ = file.Close() }() // #nosec G307 - Error handled in defer

	fmt.Fprintf(file, "Test Timing Report - Shard %d\n", r.ShardIndex)

	fmt.Fprintf(file, "=====================================\n\n")

	// Sort by duration (slowest first).

	sortedResults := make([]TestResult, len(results))

	copy(sortedResults, results)

	for i := 0; i < len(sortedResults)-1; i++ {
		for j := i + 1; j < len(sortedResults); j++ {
			if sortedResults[i].Duration < sortedResults[j].Duration {
				sortedResults[i], sortedResults[j] = sortedResults[j], sortedResults[i]
			}
		}
	}

	for _, result := range sortedResults {

		status := "PASS"

		if !result.Success {
			status = "FAIL"
		}

		fmt.Fprintf(file, "%-6s %8s %s\n", status, result.Duration.Round(time.Millisecond), result.Package)

	}

	fmt.Fprintf(file, "\nTotal: %v\n", r.totalDuration(results).Round(time.Millisecond))

	return nil
}

// Helper functions.

func (r *TestRunner) countFailures(results []TestResult) int {
	count := 0

	for _, result := range results {
		if !result.Success {
			count++
		}
	}

	return count
}

func (r *TestRunner) totalDuration(results []TestResult) time.Duration {
	var total time.Duration

	for _, result := range results {
		total += result.Duration
	}

	return total
}
