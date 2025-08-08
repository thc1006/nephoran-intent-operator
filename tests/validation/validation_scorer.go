// Package validation provides automated scoring system for comprehensive validation
package validation

import (
	"fmt"
	"strings"
	"sync"

	"github.com/onsi/ginkgo/v2"
)

// ValidationScorer provides automated scoring for validation results
type ValidationScorer struct {
	config  *ValidationConfig
	metrics map[string]*ScoreMetric
	mu      sync.RWMutex
}

// ScoreMetric represents a specific scoring criterion
type ScoreMetric struct {
	Name        string
	Category    string
	MaxPoints   int
	Threshold   float64
	Unit        string
	Weight      float64
	ActualValue float64
	Achieved    bool
}

// NewValidationScorer creates a new validation scorer
func NewValidationScorer(config *ValidationConfig) *ValidationScorer {
	scorer := &ValidationScorer{
		config:  config,
		metrics: make(map[string]*ScoreMetric),
	}
	scorer.initializeMetrics()
	return scorer
}

// initializeMetrics sets up all scoring metrics and thresholds
func (vs *ValidationScorer) initializeMetrics() {
	// Functional Completeness Metrics (50 points total)
	vs.metrics["intent_processing"] = &ScoreMetric{
		Name: "Intent Processing Pipeline", Category: "functional", MaxPoints: 15, 
		Threshold: 100.0, Unit: "% completion", Weight: 1.0,
	}
	vs.metrics["llm_rag_integration"] = &ScoreMetric{
		Name: "LLM/RAG Integration", Category: "functional", MaxPoints: 10,
		Threshold: 95.0, Unit: "% accuracy", Weight: 1.0,
	}
	vs.metrics["porch_integration"] = &ScoreMetric{
		Name: "Porch Package Management", Category: "functional", MaxPoints: 10,
		Threshold: 100.0, Unit: "% functionality", Weight: 1.0,
	}
	vs.metrics["multi_cluster"] = &ScoreMetric{
		Name: "Multi-cluster Deployment", Category: "functional", MaxPoints: 8,
		Threshold: 100.0, Unit: "% success", Weight: 1.0,
	}
	vs.metrics["oran_compliance"] = &ScoreMetric{
		Name: "O-RAN Interface Compliance", Category: "functional", MaxPoints: 7,
		Threshold: 85.0, Unit: "% compliance", Weight: 1.0,
	}
	
	// Performance Benchmarks (25 points total)
	vs.metrics["latency_p95"] = &ScoreMetric{
		Name: "P95 Latency Performance", Category: "performance", MaxPoints: 8,
		Threshold: 2000.0, Unit: "milliseconds", Weight: -1.0, // Lower is better
	}
	vs.metrics["throughput"] = &ScoreMetric{
		Name: "Throughput Performance", Category: "performance", MaxPoints: 8,
		Threshold: 45.0, Unit: "intents/minute", Weight: 1.0,
	}
	vs.metrics["scalability"] = &ScoreMetric{
		Name: "Scalability", Category: "performance", MaxPoints: 5,
		Threshold: 200.0, Unit: "concurrent intents", Weight: 1.0,
	}
	vs.metrics["resource_efficiency"] = &ScoreMetric{
		Name: "Resource Efficiency", Category: "performance", MaxPoints: 4,
		Threshold: 80.0, Unit: "% efficiency", Weight: 1.0,
	}
	
	// Security Compliance (15 points total)
	vs.metrics["authentication"] = &ScoreMetric{
		Name: "Authentication & Authorization", Category: "security", MaxPoints: 5,
		Threshold: 100.0, Unit: "% compliance", Weight: 1.0,
	}
	vs.metrics["encryption"] = &ScoreMetric{
		Name: "Data Encryption", Category: "security", MaxPoints: 4,
		Threshold: 100.0, Unit: "% coverage", Weight: 1.0,
	}
	vs.metrics["network_security"] = &ScoreMetric{
		Name: "Network Security", Category: "security", MaxPoints: 3,
		Threshold: 95.0, Unit: "% secure", Weight: 1.0,
	}
	vs.metrics["vulnerability_scan"] = &ScoreMetric{
		Name: "Vulnerability Scanning", Category: "security", MaxPoints: 3,
		Threshold: 100.0, Unit: "% clean", Weight: 1.0,
	}
	
	// Production Readiness (10 points total)
	vs.metrics["high_availability"] = &ScoreMetric{
		Name: "High Availability", Category: "production", MaxPoints: 3,
		Threshold: 99.95, Unit: "% uptime", Weight: 1.0,
	}
	vs.metrics["fault_tolerance"] = &ScoreMetric{
		Name: "Fault Tolerance", Category: "production", MaxPoints: 3,
		Threshold: 100.0, Unit: "% recovery", Weight: 1.0,
	}
	vs.metrics["observability"] = &ScoreMetric{
		Name: "Monitoring & Observability", Category: "production", MaxPoints: 2,
		Threshold: 95.0, Unit: "% coverage", Weight: 1.0,
	}
	vs.metrics["disaster_recovery"] = &ScoreMetric{
		Name: "Disaster Recovery", Category: "production", MaxPoints: 2,
		Threshold: 100.0, Unit: "% tested", Weight: 1.0,
	}
}

// ScoreMetric updates a metric with actual value and calculates score
func (vs *ValidationScorer) ScoreMetric(metricName string, actualValue float64) int {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	
	metric, exists := vs.metrics[metricName]
	if !exists {
		ginkgo.By(fmt.Sprintf("Warning: Unknown metric %s", metricName))
		return 0
	}
	
	metric.ActualValue = actualValue
	
	// Calculate score based on threshold and weight
	var score int
	if metric.Weight > 0 {
		// Higher values are better
		if actualValue >= metric.Threshold {
			score = metric.MaxPoints
			metric.Achieved = true
		} else {
			// Partial scoring based on how close to threshold
			ratio := actualValue / metric.Threshold
			score = int(float64(metric.MaxPoints) * ratio)
			if score < 0 {
				score = 0
			}
		}
	} else {
		// Lower values are better (e.g., latency)
		if actualValue <= metric.Threshold {
			score = metric.MaxPoints
			metric.Achieved = true
		} else {
			// Partial scoring - worse performance = lower score
			ratio := metric.Threshold / actualValue
			score = int(float64(metric.MaxPoints) * ratio)
			if score < 0 {
				score = 0
			}
		}
	}
	
	ginkgo.By(fmt.Sprintf("Metric '%s': %.2f %s (threshold: %.2f) = %d/%d points", 
		metric.Name, actualValue, metric.Unit, metric.Threshold, score, metric.MaxPoints))
	
	return score
}

// ScoreBinary provides binary pass/fail scoring
func (vs *ValidationScorer) ScoreBinary(metricName string, passed bool) int {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	
	metric, exists := vs.metrics[metricName]
	if !exists {
		ginkgo.By(fmt.Sprintf("Warning: Unknown metric %s", metricName))
		return 0
	}
	
	var score int
	if passed {
		score = metric.MaxPoints
		metric.Achieved = true
		metric.ActualValue = 100.0
	} else {
		score = 0
		metric.ActualValue = 0.0
	}
	
	ginkgo.By(fmt.Sprintf("Metric '%s': %t = %d/%d points", 
		metric.Name, passed, score, metric.MaxPoints))
	
	return score
}

// GetCategoryScore calculates total score for a category
func (vs *ValidationScorer) GetCategoryScore(category string) (int, int) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	totalScore := 0
	maxScore := 0
	
	for _, metric := range vs.metrics {
		if metric.Category == category {
			maxScore += metric.MaxPoints
			if metric.Achieved {
				totalScore += metric.MaxPoints
			} else {
				// Calculate partial score
				ratio := metric.ActualValue / metric.Threshold
				if metric.Weight < 0 {
					ratio = metric.Threshold / metric.ActualValue
				}
				if ratio > 1.0 {
					ratio = 1.0
				}
				partialScore := int(float64(metric.MaxPoints) * ratio)
				totalScore += partialScore
			}
		}
	}
	
	return totalScore, maxScore
}

// GetDetailedScoreReport provides detailed scoring breakdown
func (vs *ValidationScorer) GetDetailedScoreReport() string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	report := "DETAILED SCORING BREAKDOWN:\n"
	report += "==========================\n\n"
	
	categories := []string{"functional", "performance", "security", "production"}
	categoryNames := map[string]string{
		"functional":  "Functional Completeness",
		"performance": "Performance Benchmarks", 
		"security":    "Security Compliance",
		"production":  "Production Readiness",
	}
	
	totalScore := 0
	totalMaxScore := 0
	
	for _, category := range categories {
		report += fmt.Sprintf("%s:\n", categoryNames[category])
		report += strings.Repeat("-", len(categoryNames[category])+1) + "\n"
		
		categoryScore := 0
		categoryMaxScore := 0
		
		for _, metric := range vs.metrics {
			if metric.Category == category {
				actualScore := 0
				if metric.Achieved {
					actualScore = metric.MaxPoints
				} else {
					ratio := metric.ActualValue / metric.Threshold
					if metric.Weight < 0 {
						ratio = metric.Threshold / metric.ActualValue
					}
					if ratio > 1.0 {
						ratio = 1.0
					}
					if ratio < 0 {
						ratio = 0.0
					}
					actualScore = int(float64(metric.MaxPoints) * ratio)
				}
				
				status := "✓"
				if !metric.Achieved {
					status = "✗"
				}
				
				report += fmt.Sprintf("  %s %-30s: %2d/%2d points (%.1f %s)\n",
					status, metric.Name, actualScore, metric.MaxPoints, 
					metric.ActualValue, metric.Unit)
				
				categoryScore += actualScore
				categoryMaxScore += metric.MaxPoints
			}
		}
		
		report += fmt.Sprintf("  Category Total: %d/%d points\n\n", categoryScore, categoryMaxScore)
		totalScore += categoryScore
		totalMaxScore += categoryMaxScore
	}
	
	report += fmt.Sprintf("OVERALL TOTAL: %d/%d POINTS\n", totalScore, totalMaxScore)
	
	return report
}

// ValidateScoreThresholds checks if scores meet minimum requirements
func (vs *ValidationScorer) ValidateScoreThresholds() error {
	funcScore, _ := vs.GetCategoryScore("functional")
	perfScore, _ := vs.GetCategoryScore("performance")
	secScore, _ := vs.GetCategoryScore("security")
	prodScore, _ := vs.GetCategoryScore("production")
	
	if funcScore < vs.config.FunctionalTarget {
		return fmt.Errorf("functional score %d below target %d", funcScore, vs.config.FunctionalTarget)
	}
	if perfScore < vs.config.PerformanceTarget {
		return fmt.Errorf("performance score %d below target %d", perfScore, vs.config.PerformanceTarget)
	}
	if secScore < vs.config.SecurityTarget {
		return fmt.Errorf("security score %d below target %d", secScore, vs.config.SecurityTarget)
	}
	if prodScore < vs.config.ProductionTarget {
		return fmt.Errorf("production score %d below target %d", prodScore, vs.config.ProductionTarget)
	}
	
	totalScore := funcScore + perfScore + secScore + prodScore
	if totalScore < vs.config.TotalTarget {
		return fmt.Errorf("total score %d below target %d", totalScore, vs.config.TotalTarget)
	}
	
	return nil
}

// GetScoreMetrics returns all metrics for external analysis
func (vs *ValidationScorer) GetScoreMetrics() map[string]*ScoreMetric {
	vs.mu.RLock()
	defer vs.mu.RUnlock()
	
	// Return copy to prevent external modification
	metrics := make(map[string]*ScoreMetric)
	for k, v := range vs.metrics {
		metricCopy := *v
		metrics[k] = &metricCopy
	}
	
	return metrics
}

// ResetMetrics resets all metrics for a new validation run
func (vs *ValidationScorer) ResetMetrics() {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	
	for _, metric := range vs.metrics {
		metric.ActualValue = 0.0
		metric.Achieved = false
	}
}