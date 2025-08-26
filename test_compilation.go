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

// Package main is a simple compilation test
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/parallel"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/resilience"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

func main() {
	fmt.Println("Testing compilation of fixed components...")

	// Test ErrorTracker
	errorConfig := &monitoring.ErrorTrackingConfig{
		EnablePrometheus: true,
	}
	logger := logr.Discard()
	tracker, err := monitoring.NewErrorTracker(errorConfig, logger)
	if err != nil {
		fmt.Printf("Error creating tracker: %v\n", err)
		return
	}

	// Test ResilienceManager
	resilienceConfig := &resilience.ResilienceConfig{
		DefaultTimeout:          30 * time.Second,
		MaxConcurrentOperations: 100,
	}
	resMgr := resilience.NewResilienceManager(resilienceConfig, logger)

	// Test ParallelProcessingEngine
	engineConfig := &parallel.ParallelProcessingConfig{
		MaxConcurrentIntents: 50,
		IntentPoolSize:       10,
		TaskQueueSize:        1000,
	}
	engine, err := parallel.NewParallelProcessingEngine(engineConfig, resMgr, tracker, logger)
	if err != nil {
		fmt.Printf("Error creating engine: %v\n", err)
		return
	}

	// Test NetworkIntent creation
	intent := &v1.NetworkIntent{
		Spec: v1.NetworkIntentSpec{
			Intent:   "Test intent for compilation",
			Priority: v1.NetworkPriorityNormal,
		},
	}

	fmt.Printf("Created components successfully:\n")
	fmt.Printf("- ErrorTracker: %p\n", tracker)
	fmt.Printf("- ResilienceManager: %p\n", resMgr)
	fmt.Printf("- ParallelProcessingEngine: %p\n", engine)
	fmt.Printf("- NetworkIntent: %s\n", intent.Spec.Intent)

	// Test basic operations
	ctx := context.Background()
	if err := resMgr.Start(ctx); err != nil {
		fmt.Printf("Error starting resilience manager: %v\n", err)
		return
	}

	if err := engine.Start(ctx); err != nil {
		fmt.Printf("Error starting engine: %v\n", err)
		return
	}

	// Clean shutdown
	engine.Stop()
	resMgr.Stop()

	fmt.Println("All components compiled and basic operations completed successfully!")
}