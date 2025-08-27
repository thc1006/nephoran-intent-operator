package rag

import (
	"context"
	"fmt"
	"log"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
)

// ExampleWeaviateV4Query demonstrates the updated v4 API patterns
func ExampleWeaviateV4Query() {
	// Initialize Weaviate client
	cfg := weaviate.Config{
		Host:   "localhost:8080",
		Scheme: "http",
	}
	client, err := weaviate.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Example 1: Hybrid Search with Named Vectors
	hybridQuery := &SearchQuery{
		Query:         "O-RAN network slicing configuration",
		HybridSearch:  true,
		HybridAlpha:   0.75, // 75% vector, 25% keyword
		Limit:         10,
		TargetVectors: []string{"title_vector", "content_vector"}, // Named vectors
		IncludeVector: true,
	}

	// Build hybrid search with v4 API
	hybridArg := (&graphql.HybridArgumentBuilder{}).
		WithQuery(hybridQuery.Query).
		WithAlpha(hybridQuery.HybridAlpha).
		WithTargetVectors(hybridQuery.TargetVectors...)

	// Example 2: Near Text Search with Certainty
	nearTextQuery := &SearchQuery{
		Query:         "5G RAN optimization techniques",
		MinConfidence: 0.8,
		Limit:         5,
		TargetVectors: []string{"content_vector"},
	}

	// Build near text search with v4 API
	nearTextArg := (&graphql.NearTextArgumentBuilder{}).
		WithConcepts([]string{nearTextQuery.Query}).
		WithCertainty(float32(nearTextQuery.MinConfidence)).
		WithTargetVectors(nearTextQuery.TargetVectors...)

	// Example 3: Multi-Target Vector Search with Manual Weights
	multiTargetArg := (&graphql.MultiTargetArgumentBuilder{}).
		ManualWeights(map[string]float32{
			"title_vector":   0.3,
			"content_vector": 0.7,
		})

	hybridWithMultiTarget := (&graphql.HybridArgumentBuilder{}).
		WithQuery("network function virtualization").
		WithTargets(multiTargetArg)

	// Execute queries
	ctx := context.Background()

	// Query 1: Hybrid search
	result1, err := client.GraphQL().Get().
		WithClassName("TelecomKnowledge").
		WithHybrid(hybridArg).
		WithFields(
			graphql.Field{Name: "content"},
			graphql.Field{Name: "title"},
			graphql.Field{Name: "_additional", Fields: []graphql.Field{
				{Name: "id"},
				{Name: "score"},
				{Name: "distance"},
				{Name: "vector"}, // Vector is now stored in metadata
			}},
		).
		WithLimit(10).
		Do(ctx)

	if err != nil {
		log.Printf("Hybrid search failed: %v", err)
	} else {
		fmt.Printf("Hybrid search returned %d results\n", len(result1.Data))
	}

	// Query 2: Near text search
	result2, err := client.GraphQL().Get().
		WithClassName("TelecomKnowledge").
		WithNearText(nearTextArg).
		WithFields(
			graphql.Field{Name: "content"},
			graphql.Field{Name: "title"},
		).
		WithLimit(5).
		Do(ctx)

	if err != nil {
		log.Printf("Near text search failed: %v", err)
	} else {
		fmt.Printf("Near text search returned %d results\n", len(result2.Data))
	}

	// Query 3: Multi-target hybrid search
	result3, err := client.GraphQL().Get().
		WithClassName("TelecomKnowledge").
		WithHybrid(hybridWithMultiTarget).
		WithFields(
			graphql.Field{Name: "content"},
			graphql.Field{Name: "title"},
		).
		WithLimit(10).
		Do(ctx)

	if err != nil {
		log.Printf("Multi-target search failed: %v", err)
	} else {
		fmt.Printf("Multi-target search returned %d results\n", len(result3.Data))
	}
}

// Key v4 API Changes:
// 1. GraphQL builders are instantiated directly: graphql.HybridArgumentBuilder{}
//    instead of client.GraphQL().HybridArgumentBuilder()
// 2. WithTargetVectors() for named vector support
// 3. WithTargets() for multi-target vector queries with weights
// 4. SearchResult.Vector field removed - vectors stored in metadata
// 5. WithCertainty() instead of MinConfidence in where filters
// 6. Simplified filter building - complex WHERE filters need proper v4 construction
