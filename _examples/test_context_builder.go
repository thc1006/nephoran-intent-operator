//go:build examples
// +build examples

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nephio-project/nephoran-intent-operator/pkg/llm"
	"github.com/nephio-project/nephoran-intent-operator/pkg/rag"
)

func main() {
	fmt.Println("Testing ContextBuilder implementation...")

	// Create a ContextBuilder without a connection pool to test graceful handling
	cb := llm.NewContextBuilder()

	// Test BuildContext with no pool (should return empty context)
	ctx := context.Background()
	intent := "Deploy a 5G AMF function with high availability"
	maxDocs := 5

	contextDocs, err := cb.BuildContext(ctx, intent, maxDocs)
	if err != nil {
		log.Printf("Error building context: %v", err)
	} else {
		log.Printf("Successfully built context with %d documents", len(contextDocs))
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
