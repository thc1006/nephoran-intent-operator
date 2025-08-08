// Package validation provides baseline management for regression testing
package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"crypto/sha256"

	"github.com/onsi/ginkgo/v2"
)

// BaselineManager handles creation, storage, and retrieval of validation baselines
type BaselineManager struct {
	config      *RegressionConfig
	storagePath string
}

// NewBaselineManager creates a new baseline manager
func NewBaselineManager(config *RegressionConfig) *BaselineManager {
	return &BaselineManager{
		config:      config,
		storagePath: config.BaselineStoragePath,
	}
}

// CreateBaseline creates a new baseline from validation results
func (bm *BaselineManager) CreateBaseline(results *ValidationResults, context string) (*BaselineSnapshot, error) {
	ginkgo.By(fmt.Sprintf("Creating baseline with context: %s", context))
	
	// Generate baseline ID
	baselineID := bm.generateBaselineID(results, context)
	
	// Create baseline snapshot
	baseline := &BaselineSnapshot{
		ID:          baselineID,
		Timestamp:   time.Now(),
		Version:     bm.getSystemVersion(),
		CommitHash:  bm.getCurrentCommitHash(),
		Environment: bm.getEnvironmentInfo(),
		Results:     results,
		PerformanceBaselines: bm.createPerformanceBaselines(results),
		SecurityBaselines:    bm.createSecurityBaseline(results),
		ProductionBaselines:  bm.createProductionBaseline(results),
		Statistics:          bm.calculateBaselineStatistics(results),
	}
	
	// Store baseline
	if err := bm.storeBaseline(baseline); err != nil {
		return nil, fmt.Errorf("failed to store baseline: %w", err)
	}
	
	// Clean up old baselines if retention limit exceeded
	if err := bm.cleanupOldBaselines(); err != nil {
		ginkgo.By(fmt.Sprintf("Warning: Failed to cleanup old baselines: %v", err))
	}
	
	ginkgo.By(fmt.Sprintf("Baseline created successfully: %s", baselineID))
	return baseline, nil
}

// LoadLatestBaseline loads the most recent baseline
func (bm *BaselineManager) LoadLatestBaseline() (*BaselineSnapshot, error) {
	baselines, err := bm.listBaselines()
	if err != nil {
		return nil, err
	}
	
	if len(baselines) == 0 {
		return nil, os.ErrNotExist
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(baselines, func(i, j int) bool {
		return baselines[i].Timestamp.After(baselines[j].Timestamp)
	})
	
	return bm.loadBaseline(baselines[0].ID)
}

// LoadBaseline loads a specific baseline by ID
func (bm *BaselineManager) LoadBaseline(id string) (*BaselineSnapshot, error) {
	return bm.loadBaseline(id)
}

// LoadAllBaselines loads all available baselines
func (bm *BaselineManager) LoadAllBaselines() ([]*BaselineSnapshot, error) {
	baselineInfos, err := bm.listBaselines()
	if err != nil {
		return nil, err
	}
	
	baselines := make([]*BaselineSnapshot, len(baselineInfos))
	for i, info := range baselineInfos {
		baseline, err := bm.loadBaseline(info.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load baseline %s: %w", info.ID, err)
		}
		baselines[i] = baseline
	}
	
	// Sort by timestamp (newest first)
	sort.Slice(baselines, func(i, j int) bool {
		return baselines[i].Timestamp.After(baselines[j].Timestamp)
	})
	
	return baselines, nil
}

// GetRegressionHistory returns regression detection history
func (bm *BaselineManager) GetRegressionHistory(days int) ([]*RegressionDetection, error) {
	since := time.Now().AddDate(0, 0, -days)
	
	// List regression reports
	reportFiles, err := filepath.Glob(filepath.Join(bm.storagePath, "regression-report-*.json"))
	if err != nil {
		return nil, err
	}
	
	var history []*RegressionDetection
	for _, file := range reportFiles {
		// Check file modification time
		stat, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		if stat.ModTime().Before(since) {
			continue
		}
		
		// Load regression detection
		detection, err := bm.loadRegressionReport(file)
		if err != nil {
			ginkgo.By(fmt.Sprintf("Warning: Failed to load regression report %s: %v", file, err))
			continue
		}
		
		history = append(history, detection)
	}
	
	// Sort by comparison time (newest first)
	sort.Slice(history, func(i, j int) bool {
		return history[i].ComparisonTime.After(history[j].ComparisonTime)
	})
	
	return history, nil
}

// CompareBaselines compares two baselines and returns differences
func (bm *BaselineManager) CompareBaselines(baselineA, baselineB *BaselineSnapshot) *BaselineComparison {
	comparison := &BaselineComparison{
		BaselineA:      baselineA,
		BaselineB:      baselineB,
		ComparisonTime: time.Now(),
		Differences:    make(map[string]*BaselineDifference),
	}
	
	// Compare overall scores
	comparison.Differences["overall_score"] = &BaselineDifference{
		Category:    "Overall Score",
		ValueA:      float64(baselineA.Results.TotalScore),
		ValueB:      float64(baselineB.Results.TotalScore),
		Difference:  float64(baselineB.Results.TotalScore - baselineA.Results.TotalScore),
		PercentChange: calculatePercentChange(float64(baselineA.Results.TotalScore), float64(baselineB.Results.TotalScore)),
	}
	
	// Compare category scores
	categories := map[string][2]int{
		"functional":  {baselineA.Results.FunctionalScore, baselineB.Results.FunctionalScore},
		"performance": {baselineA.Results.PerformanceScore, baselineB.Results.PerformanceScore},
		"security":    {baselineA.Results.SecurityScore, baselineB.Results.SecurityScore},
		"production":  {baselineA.Results.ProductionScore, baselineB.Results.ProductionScore},
	}
	
	for category, scores := range categories {
		comparison.Differences[category] = &BaselineDifference{
			Category:      category,
			ValueA:        float64(scores[0]),
			ValueB:        float64(scores[1]),
			Difference:    float64(scores[1] - scores[0]),
			PercentChange: calculatePercentChange(float64(scores[0]), float64(scores[1])),
		}
	}
	
	// Compare performance metrics
	if baselineA.Results.P95Latency > 0 && baselineB.Results.P95Latency > 0 {
		comparison.Differences["p95_latency"] = &BaselineDifference{
			Category:      "P95 Latency",
			ValueA:        float64(baselineA.Results.P95Latency.Nanoseconds()),
			ValueB:        float64(baselineB.Results.P95Latency.Nanoseconds()),
			Difference:    float64(baselineB.Results.P95Latency.Nanoseconds() - baselineA.Results.P95Latency.Nanoseconds()),
			PercentChange: calculatePercentChange(float64(baselineA.Results.P95Latency.Nanoseconds()), float64(baselineB.Results.P95Latency.Nanoseconds())),
		}
	}
	
	if baselineA.Results.ThroughputAchieved > 0 && baselineB.Results.ThroughputAchieved > 0 {
		comparison.Differences["throughput"] = &BaselineDifference{
			Category:      "Throughput",
			ValueA:        baselineA.Results.ThroughputAchieved,
			ValueB:        baselineB.Results.ThroughputAchieved,
			Difference:    baselineB.Results.ThroughputAchieved - baselineA.Results.ThroughputAchieved,
			PercentChange: calculatePercentChange(baselineA.Results.ThroughputAchieved, baselineB.Results.ThroughputAchieved),
		}
	}
	
	if baselineA.Results.AvailabilityAchieved > 0 && baselineB.Results.AvailabilityAchieved > 0 {
		comparison.Differences["availability"] = &BaselineDifference{
			Category:      "Availability",
			ValueA:        baselineA.Results.AvailabilityAchieved,
			ValueB:        baselineB.Results.AvailabilityAchieved,
			Difference:    baselineB.Results.AvailabilityAchieved - baselineA.Results.AvailabilityAchieved,
			PercentChange: calculatePercentChange(baselineA.Results.AvailabilityAchieved, baselineB.Results.AvailabilityAchieved),
		}
	}
	
	return comparison
}

// BaselineComparison holds the result of comparing two baselines
type BaselineComparison struct {
	BaselineA      *BaselineSnapshot                `json:"baseline_a"`
	BaselineB      *BaselineSnapshot                `json:"baseline_b"`
	ComparisonTime time.Time                        `json:"comparison_time"`
	Differences    map[string]*BaselineDifference   `json:"differences"`
}

// BaselineDifference represents a difference between two baseline values
type BaselineDifference struct {
	Category      string  `json:"category"`
	ValueA        float64 `json:"value_a"`
	ValueB        float64 `json:"value_b"`
	Difference    float64 `json:"difference"`
	PercentChange float64 `json:"percent_change"`
}

// generateBaselineID creates a unique identifier for a baseline
func (bm *BaselineManager) generateBaselineID(results *ValidationResults, context string) string {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", timestamp, context, results.TotalScore)))
	return fmt.Sprintf("%s-%x", timestamp, hash[:8])
}

// createPerformanceBaselines extracts performance baselines from results
func (bm *BaselineManager) createPerformanceBaselines(results *ValidationResults) map[string]*PerformanceBaseline {
	baselines := make(map[string]*PerformanceBaseline)
	
	// P95 Latency baseline
	if results.P95Latency > 0 {
		baselines["p95_latency"] = &PerformanceBaseline{
			MetricName:      "P95 Latency",
			AverageValue:    float64(results.P95Latency.Nanoseconds()),
			MedianValue:     float64(results.P95Latency.Nanoseconds()),
			P95Value:        float64(results.P95Latency.Nanoseconds()),
			P99Value:        float64(results.P99Latency.Nanoseconds()),
			AcceptableRange: [2]float64{0, float64(2 * time.Second.Nanoseconds())},
			Unit:            "nanoseconds",
			Threshold:       float64(2 * time.Second.Nanoseconds()),
		}
	}
	
	// Throughput baseline
	if results.ThroughputAchieved > 0 {
		baselines["throughput"] = &PerformanceBaseline{
			MetricName:      "Throughput",
			AverageValue:    results.ThroughputAchieved,
			MedianValue:     results.ThroughputAchieved,
			P95Value:        results.ThroughputAchieved,
			P99Value:        results.ThroughputAchieved,
			AcceptableRange: [2]float64{45.0, 1000.0},
			Unit:            "intents/minute",
			Threshold:       45.0,
		}
	}
	
	// Availability baseline
	if results.AvailabilityAchieved > 0 {
		baselines["availability"] = &PerformanceBaseline{
			MetricName:      "Availability",
			AverageValue:    results.AvailabilityAchieved,
			MedianValue:     results.AvailabilityAchieved,
			P95Value:        results.AvailabilityAchieved,
			P99Value:        results.AvailabilityAchieved,
			AcceptableRange: [2]float64{99.95, 100.0},
			Unit:            "percentage",
			Threshold:       99.95,
		}
	}
	
	return baselines
}

// createSecurityBaseline extracts security baseline from results
func (bm *BaselineManager) createSecurityBaseline(results *ValidationResults) *SecurityBaseline {
	baseline := &SecurityBaseline{
		SecurityScore:           results.SecurityScore,
		VulnerabilityCount:      len(results.SecurityFindings),
		CriticalVulnerabilities: 0,
		HighVulnerabilities:     0,
		ComplianceFindings:      make(map[string]*SecurityFinding),
		EncryptionCoverage:      100.0, // Default assumption
		AuthenticationScore:     5,     // Default assumption from max auth score
	}
	
	// Count vulnerabilities by severity
	for _, finding := range results.SecurityFindings {
		baseline.ComplianceFindings[finding.Type] = finding
		switch strings.ToLower(finding.Severity) {
		case "critical":
			baseline.CriticalVulnerabilities++
		case "high":
			baseline.HighVulnerabilities++
		}
	}
	
	return baseline
}

// createProductionBaseline extracts production readiness baseline from results
func (bm *BaselineManager) createProductionBaseline(results *ValidationResults) *ProductionBaseline {
	baseline := &ProductionBaseline{
		AvailabilityTarget:     results.AvailabilityAchieved,
		MTBF:                  24 * time.Hour, // Default assumption
		MTTR:                  5 * time.Minute, // Default assumption
		ErrorRate:             0.05, // Default assumption (0.05% error rate)
		FaultToleranceScore:   3,    // Default assumption from max fault tolerance score
		MonitoringCoverage:    90.0, // Default assumption
		DisasterRecoveryScore: 2,    // Default assumption from max disaster recovery score
	}
	
	// Extract from reliability metrics if available
	if results.ReliabilityMetrics != nil {
		baseline.MTBF = results.ReliabilityMetrics.MTBF
		baseline.MTTR = results.ReliabilityMetrics.MTTR
		baseline.ErrorRate = results.ReliabilityMetrics.ErrorRate
		baseline.AvailabilityTarget = results.ReliabilityMetrics.Availability
	}
	
	return baseline
}

// calculateBaselineStatistics calculates statistical metrics for the baseline
func (bm *BaselineManager) calculateBaselineStatistics(results *ValidationResults) *BaselineStatistics {
	return &BaselineStatistics{
		SampleSize:       1, // Single sample for new baseline
		ConfidenceLevel:  95.0,
		VariabilityIndex: 0.0, // No variability with single sample
		TrendDirection:   "stable",
		SeasonalPatterns: false,
	}
}

// storeBaseline saves a baseline to storage
func (bm *BaselineManager) storeBaseline(baseline *BaselineSnapshot) error {
	filename := fmt.Sprintf("baseline-%s.json", baseline.ID)
	path := filepath.Join(bm.storagePath, filename)
	
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}
	
	return os.WriteFile(path, data, 0644)
}

// loadBaseline loads a baseline from storage
func (bm *BaselineManager) loadBaseline(id string) (*BaselineSnapshot, error) {
	filename := fmt.Sprintf("baseline-%s.json", id)
	path := filepath.Join(bm.storagePath, filename)
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var baseline BaselineSnapshot
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal baseline: %w", err)
	}
	
	return &baseline, nil
}

// listBaselines returns a list of all available baselines
func (bm *BaselineManager) listBaselines() ([]*BaselineSnapshot, error) {
	pattern := filepath.Join(bm.storagePath, "baseline-*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	var baselines []*BaselineSnapshot
	for _, file := range files {
		// Extract baseline ID from filename
		filename := filepath.Base(file)
		id := strings.TrimSuffix(strings.TrimPrefix(filename, "baseline-"), ".json")
		
		// Load minimal baseline info (could be optimized to load only metadata)
		baseline, err := bm.loadBaseline(id)
		if err != nil {
			ginkgo.By(fmt.Sprintf("Warning: Failed to load baseline %s: %v", id, err))
			continue
		}
		
		baselines = append(baselines, baseline)
	}
	
	return baselines, nil
}

// cleanupOldBaselines removes baselines beyond retention limit
func (bm *BaselineManager) cleanupOldBaselines() error {
	if bm.config.BaselineRetention <= 0 {
		return nil // No retention limit
	}
	
	baselines, err := bm.listBaselines()
	if err != nil {
		return err
	}
	
	if len(baselines) <= bm.config.BaselineRetention {
		return nil // Within retention limit
	}
	
	// Sort by timestamp (oldest first for deletion)
	sort.Slice(baselines, func(i, j int) bool {
		return baselines[i].Timestamp.Before(baselines[j].Timestamp)
	})
	
	// Delete excess baselines
	excessCount := len(baselines) - bm.config.BaselineRetention
	for i := 0; i < excessCount; i++ {
		filename := fmt.Sprintf("baseline-%s.json", baselines[i].ID)
		path := filepath.Join(bm.storagePath, filename)
		
		if err := os.Remove(path); err != nil {
			ginkgo.By(fmt.Sprintf("Warning: Failed to remove old baseline %s: %v", baselines[i].ID, err))
		} else {
			ginkgo.By(fmt.Sprintf("Cleaned up old baseline: %s", baselines[i].ID))
		}
	}
	
	return nil
}

// loadRegressionReport loads a regression report from file
func (bm *BaselineManager) loadRegressionReport(filename string) (*RegressionDetection, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	var detection RegressionDetection
	if err := json.Unmarshal(data, &detection); err != nil {
		return nil, fmt.Errorf("failed to unmarshal regression report: %w", err)
	}
	
	return &detection, nil
}

// getSystemVersion returns the current system version
func (bm *BaselineManager) getSystemVersion() string {
	// This would typically read from VERSION file or build info
	return "v1.0.0" // Placeholder
}

// getCurrentCommitHash returns the current git commit hash
func (bm *BaselineManager) getCurrentCommitHash() string {
	// This would typically execute: git rev-parse HEAD
	return "main" // Placeholder
}

// getEnvironmentInfo returns information about the test environment
func (bm *BaselineManager) getEnvironmentInfo() string {
	env := os.Getenv("TEST_ENVIRONMENT")
	if env == "" {
		return "development"
	}
	return env
}

// calculatePercentChange calculates the percentage change between two values
func calculatePercentChange(oldValue, newValue float64) float64 {
	if oldValue == 0 {
		if newValue == 0 {
			return 0
		}
		return 100 // 100% change from 0 to any value
	}
	
	return ((newValue - oldValue) / oldValue) * 100
}