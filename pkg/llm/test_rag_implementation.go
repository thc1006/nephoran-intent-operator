//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

func main() {
	// Create a RAG-enhanced processor with default config.
	processor := llm.NewRAGEnhancedProcessor()

	// Test intent that should use RAG.
	testIntents := []string{
		"Deploy a 5G AMF with high availability",
		"How to configure O-RAN E2 interface",
		"Scale UPF to handle increased traffic",
		"Create nginx deployment", // Should not use RAG
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, intent := range testIntents {
		fmt.Printf("\n=== Testing intent: %s ===\n", intent)

		result, err := processor.ProcessIntent(ctx, intent)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Success! Result length: %d characters\n", len(result))
			if len(result) > 200 {
				fmt.Printf("Result preview: %s...\n", result[:200])
			} else {
				fmt.Printf("Result: %s\n", result)
			}
		}
	}
}
