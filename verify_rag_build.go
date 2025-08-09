package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

func main() {
	fmt.Println("=== RAG Build Verification ===")
	
	config := &rag.RAGClientConfig{
		Enabled:          true,
		MaxSearchResults: 5,
		MinConfidence:    0.7,
	}
	
	client := rag.NewRAGClient(config)
	ctx := context.Background()
	
	// Test Initialize
	if err := client.Initialize(ctx); err != nil {
		log.Printf("Initialize error: %v", err)
	} else {
		fmt.Println("✓ Initialize succeeded")
	}
	
	// Test ProcessIntent
	result, err := client.ProcessIntent(ctx, "test intent")
	if err != nil {
		if strings.Contains(err.Error(), "RAG support is not enabled") {
			fmt.Println("✓ Running with NO-OP implementation (build without -tags=rag)")
		} else {
			log.Printf("ProcessIntent error: %v", err)
		}
	} else {
		fmt.Printf("✓ Running with Weaviate implementation (build with -tags=rag)\n")
		fmt.Printf("  Result: %s\n", result)
	}
	
	// Test IsHealthy
	healthy := client.IsHealthy()
	fmt.Printf("✓ IsHealthy: %v\n", healthy)
	
	// Test Shutdown
	if err := client.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	} else {
		fmt.Println("✓ Shutdown succeeded")
	}
	
	fmt.Println("\n=== Build verification complete ===")
}