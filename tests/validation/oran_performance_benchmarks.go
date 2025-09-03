// Package test_validation provides performance benchmarking for O-RAN interfaces.
package test_validation

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/onsi/ginkgo/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// [Previous code remains the same]

// Fixed method with corrected syntax
func (opb *ORANPerformanceBenchmarker) validatePerformanceTargets() {
	ginkgo.By("Validating Performance Targets")

	targets := map[string]struct {
		minThroughput  float64
		maxLatency     time.Duration
		maxErrorRate   float64
	}{
		"A1": {minThroughput: 20.0, maxLatency: 100 * time.Millisecond, maxErrorRate: 2.0},
		"E2": {minThroughput: 50.0, maxLatency: 50 * time.Millisecond, maxErrorRate: 1.5},
		"O1": {minThroughput: 10.0, maxLatency: 200 * time.Millisecond, maxErrorRate: 1.0},
		"O2": {minThroughput: 2.0, maxLatency: 5 * time.Second, maxErrorRate: 3.0},
	}

	allTargetsMet := true

	for interfaceName, target := range targets {
		// Fixed syntax error here: removed extra parenthesis
		if result, exists := opb.benchmarkResults[interfaceName]); exists {
			targetsMet := true

			if result.ThroughputRPS < target.minThroughput {
				ginkgo.By(fmt.Sprintf("[❌] %s Throughput: %.2f RPS < %.2f RPS (target)",
					interfaceName, result.ThroughputRPS, target.minThroughput))
				targetsMet = false
			} else {
				ginkgo.By(fmt.Sprintf("[✅] %s Throughput: %.2f RPS >= %.2f RPS (target)",
					interfaceName, result.ThroughputRPS, target.minThroughput))
			}

			if result.AverageLatency > target.maxLatency {
				ginkgo.By(fmt.Sprintf("[❌] %s Latency: %.2fms > %.2fms (target)",
					interfaceName, 
					float64(result.AverageLatency.Nanoseconds())/1e6,
					float64(target.maxLatency.Nanoseconds())/1e6))
				targetsMet = false
			} else {
				ginkgo.By(fmt.Sprintf("[✅] %s Latency: %.2fms <= %.2fms (target)",
					interfaceName, 
					float64(result.AverageLatency.Nanoseconds())/1e6,
					float64(target.maxLatency.Nanoseconds())/1e6))
			}

			if result.ErrorRate > target.maxErrorRate {
				ginkgo.By(fmt.Sprintf("[❌] %s Error Rate: %.1f%% > %.1f%% (target)",
					interfaceName, result.ErrorRate, target.maxErrorRate))
				targetsMet = false
			} else {
				ginkgo.By(fmt.Sprintf("[✅] %s Error Rate: %.1f%% <= %.1f%% (target)",
					interfaceName, result.ErrorRate, target.maxErrorRate))
			}

			if !targetsMet {
				allTargetsMet = false
			}
		}
	}

	if allTargetsMet {
		ginkgo.By("✅ All Performance Targets Met!")
	} else {
		ginkgo.By("🚨 Some Performance Targets Not Met")
	}
}