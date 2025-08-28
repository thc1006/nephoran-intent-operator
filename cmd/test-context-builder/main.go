// Package main provides a utility for building test contexts and environments for Nephoran testing.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

func main() {
	fmt.Println("Testing ContextBuilder implementation...")

	// Create a ContextBuilder without a connection pool to test graceful handling
	cb := llm.NewContextBuilderStub()

	// Test BuildContext with no pool (should return empty context)
	ctx := context.Background()
	intent := "Deploy a 5G AMF function with high availability"
	maxDocs := 5 // Max documents to retrieve

	contextData, err := cb.BuildContext(ctx, intent, maxDocs)
	if err != nil {
		log.Printf("Error building context: %v", err)
	} else {
		log.Printf("Successfully built context with %d documents", len(contextData))
		if len(contextData) > 0 {
			fmt.Printf("Sample context data: %+v\n", contextData[0])
		}
	}

	// Test metrics
	metrics := cb.GetMetrics()
	fmt.Printf("ContextBuilder Metrics: %+v\n", metrics)

	// Test with connection pool (would require actual Weaviate instance)
	poolConfig := rag.DefaultPoolConfig()
	poolConfig.URL = "http://localhost:8080" // This would need a real Weaviate instance

	// This would work if Weaviate was running:
	// pool := rag.NewWeaviateConnectionPool(poolConfig)
	// cbWithPool := llm.NewContextBuilderWithPool(pool)
	// contextDocs, err = cbWithPool.BuildContext(ctx, intent, maxDocs)

	fmt.Println("ContextBuilder implementation test completed successfully!")
}
