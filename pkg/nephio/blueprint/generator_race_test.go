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

package blueprint

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"go.uber.org/zap"
)

// TestClientAdapterConcurrentAccess verifies that ClientAdapter is safe for concurrent use
func TestClientAdapterConcurrentAccess(t *testing.T) {
	// Create a mock client
	mockClient := llm.NewClient("http://test:8080")
	adapter := &ClientAdapter{client: mockClient}

	// Test concurrent access to ProcessIntent
	var wg sync.WaitGroup
	numGoroutines := 10
	numCallsPerGoroutine := 5

	ctx := context.Background()
	errors := make(chan error, numGoroutines*numCallsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numCallsPerGoroutine; j++ {
				request := &llm.ProcessIntentRequest{
					Intent: "test intent",
					Context: map[string]string{
						"goroutine": string(rune(id)),
						"call":      string(rune(j)),
					},
					Metadata: llm.RequestMetadata{
						RequestID: "test-request",
					},
					Timestamp: time.Now(),
				}

				// This will fail because there's no actual LLM service,
				// but we're testing for race conditions, not functionality
				_, err := adapter.ProcessIntent(ctx, request)
				if err != nil {
					// Expected to fail, but shouldn't race
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// If we get here without a race condition detected, the test passes
	t.Log("No race conditions detected in concurrent ClientAdapter access")
}

// TestGeneratorConcurrentGeneration tests concurrent blueprint generation
func TestGeneratorConcurrentGeneration(t *testing.T) {
	config := DefaultBlueprintConfig()
	logger := zap.NewNop()

	// Create generator with mock client
	generator, err := NewGenerator(config, logger)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create multiple intents
	var wg sync.WaitGroup
	numConcurrent := 5

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			intent := createTestNetworkIntent(string(rune('a' + id)))
			ctx := context.Background()

			// This will fail due to no LLM service, but tests for races
			_, _ = generator.GenerateFromNetworkIntent(ctx, intent)
		}(i)
	}

	wg.Wait()
	t.Log("No race conditions detected in concurrent blueprint generation")
}