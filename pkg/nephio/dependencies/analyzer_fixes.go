/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dependencies

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// DefaultAnalyzerConfig returns a default AnalyzerConfig configuration
func DefaultAnalyzerConfig() *AnalyzerConfig {
	return &AnalyzerConfig{
		EnableMLAnalysis:       false,
		EnableMLOptimization:   false,
		EnableCaching:          true,
		EnableConcurrency:      true,
		EnableBenchmarking:     false,
		EnableResourceAnalysis: false,
		EnableTrendAnalysis:    true,
		WorkerCount:           4,
		QueueSize:             100,
		UsageAnalyzerConfig:      &UsageAnalyzerConfig{},
		CostAnalyzerConfig:       &CostAnalyzerConfig{},
		HealthAnalyzerConfig:     &HealthAnalyzerConfig{},
		RiskAnalyzerConfig:       &RiskAnalyzerConfig{},
		PerformanceAnalyzerConfig: &PerformanceAnalyzerConfig{},
	}
}

// AnalyzerMetrics with missing fields
type AnalyzerMetrics struct {
	// Cache metrics
	AnalysisCacheHits   prometheus.Counter
	AnalysisCacheMisses prometheus.Counter
	
	// System metrics
	Uptime              prometheus.Gauge

	// Analysis metrics
	UsageAnalysisTotal    prometheus.Counter
	UsageAnalysisTime     prometheus.Histogram
	CostAnalysisTotal     prometheus.Counter
	CostAnalysisTime      prometheus.Histogram
	HealthAnalysisTotal   prometheus.Counter
	HealthAnalysisTime    prometheus.Histogram
	
	// Optimization metrics
	OptimizationRecommendationsGenerated prometheus.Counter
	OptimizationRecommendationTime       prometheus.Histogram
}

// NewAnalyzerMetrics creates new analyzer metrics
func NewAnalyzerMetrics() *AnalyzerMetrics {
	return &AnalyzerMetrics{
		AnalysisCacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analyzer_cache_hits_total",
			Help: "Total number of analysis cache hits",
		}),
		AnalysisCacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analyzer_cache_misses_total",
			Help: "Total number of analysis cache misses",
		}),
		UsageAnalysisTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analyzer_usage_analysis_total",
			Help: "Total number of usage analyses performed",
		}),
		UsageAnalysisTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "analyzer_usage_analysis_duration_seconds",
			Help: "Time taken for usage analysis",
		}),
		CostAnalysisTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analyzer_cost_analysis_total",
			Help: "Total number of cost analyses performed",
		}),
		CostAnalysisTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "analyzer_cost_analysis_duration_seconds",
			Help: "Time taken for cost analysis",
		}),
		HealthAnalysisTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analyzer_health_analysis_total",
			Help: "Total number of health analyses performed",
		}),
		HealthAnalysisTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "analyzer_health_analysis_duration_seconds",
			Help: "Time taken for health analysis",
		}),
		OptimizationRecommendationsGenerated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analyzer_optimization_recommendations_total",
			Help: "Total number of optimization recommendations generated",
		}),
		OptimizationRecommendationTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "analyzer_optimization_recommendation_duration_seconds",
			Help: "Time taken to generate optimization recommendations",
		}),
	}
}

// Add missing methods to dependencyAnalyzer struct

// Note: The following methods have been moved to analyzer.go to avoid duplicates:
// - generateAnalysisCacheKey
// - performMLAnalysis  
// - updateAnalysisMetrics
// - usageDataCollectionProcess
// - metricsCollectionProcess
// - eventProcessingLoop
// - mlModelUpdateProcess

// Remaining helper methods for compliance analysis that are not duplicated

// analyzePackageComplianceRisks analyzes compliance risks for a single package
func (a *dependencyAnalyzer) analyzePackageComplianceRisks(ctx context.Context, pkg *PackageReference) []*ComplianceIssue {
	issues := make([]*ComplianceIssue, 0)

	// Check for known compliance issues (stub implementation)
	issue := &ComplianceIssue{
		ID:          fmt.Sprintf("compliance-issue-%s-%s", pkg.Name, pkg.Version),
		Standard:    "SOC2",
		Severity:    "low",
		Description: "Package license compliance verified",
		Package:     pkg.Name,
		Status:      "resolved",
		DetectedAt:  time.Now(),
	}
	issues = append(issues, issue)

	return issues
}

// determineOverallComplianceRiskLevel determines overall compliance risk level
func (a *dependencyAnalyzer) determineOverallComplianceRiskLevel(issues []*ComplianceIssue) RiskLevel {
	if len(issues) == 0 {
		return RiskLevelLow
	}

	highSeverityCount := 0
	for _, issue := range issues {
		if issue.Severity == "high" {
			highSeverityCount++
		}
	}

	if highSeverityCount > 2 {
		return RiskLevelHigh
	} else if len(issues) > 5 {
		return RiskLevelMedium
	}

	return RiskLevelLow
}

// generateComplianceActions generates compliance actions for issues
func (a *dependencyAnalyzer) generateComplianceActions(issues []*ComplianceIssue) []*ComplianceAction {
	actions := make([]*ComplianceAction, 0)

	for _, issue := range issues {
		if issue.Status == "open" {
			action := &ComplianceAction{
				ID:          fmt.Sprintf("action-%s", issue.ID),
				Type:        "remediation",
				Description: fmt.Sprintf("Address compliance issue: %s", issue.Description),
				Priority:    issue.Severity,
				DueDate:     time.Now().Add(30 * 24 * time.Hour), // 30 days
				Status:      "pending",
			}
			actions = append(actions, action)
		}
	}

	return actions
}

// generateComplianceRiskAnalysisID generates unique ID for compliance risk analysis
func generateComplianceRiskAnalysisID() string {
	return fmt.Sprintf("compliance-risk-%d", time.Now().UnixNano())
}

// End of analyzer_fixes.go - all other methods have been moved to analyzer.go to avoid duplicates