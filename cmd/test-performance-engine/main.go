//go:build testperformance

package main

import (
	"context"
	"log"
	"time"

	"github.com/go-logr/logr"
	"github.com/nephio-project/nephoran-intent-operator/pkg/optimization"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// Test that the PerformanceAnalysisEngine compiles and runs.

func main() {

	logger := logr.Discard()

	var prometheusClient v1.API

	config := optimization.GetDefaultAnalysisConfig()

	engine := optimization.NewPerformanceAnalysisEngine(config, prometheusClient, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	result, err := engine.AnalyzePerformance(ctx)

	if err != nil {

		log.Printf("Analysis failed: %v", err)

	} else {

		log.Printf("Analysis completed: %s, System Health: %s", result.AnalysisID, result.SystemHealth)

	}

}
