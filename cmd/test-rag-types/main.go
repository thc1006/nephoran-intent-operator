package main

import (
	"fmt"
	"time"
	
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

func main() {
	// Test DocumentChunk type
	chunk := &rag.DocumentChunk{
		ID:           "test_chunk_1",
		DocumentID:   "test_doc",
		Content:      "This is test content",
		CleanContent: "This is test content",
		ChunkIndex:   1,
		StartOffset:  0,
		EndOffset:    20,
		CharacterCount: 20,
		WordCount:    4,
		SentenceCount: 1,
		HierarchyPath:  []string{"root", "section1"},
		SectionTitle:   "Test Section",
		HierarchyLevel: 2,
		ChunkType:      "text",
		ProcessedAt:    time.Now(),
	}
	
	fmt.Printf("DocumentChunk created: ID=%s, Content=%s\n", chunk.ID, chunk.Content)
	
	// Test SearchQuery type
	query := &shared.SearchQuery{
		Query:         "test query",
		Limit:         10,
		Offset:        0,
		HybridSearch:  true,
		HybridAlpha:   0.5,
		UseReranker:   false,
		MinConfidence: 0.8,
		IncludeVector: true,
		ExpandQuery:   false,
		TargetVectors: []string{"content_vector", "title_vector"},
	}
	
	fmt.Printf("SearchQuery created: Query=%s, Limit=%d, TargetVectors=%v\n", 
		query.Query, query.Limit, query.TargetVectors)
	
	// Test type alias in RAG package
	var ragQuery rag.SearchQuery = *query
	fmt.Printf("RAG SearchQuery alias works: Query=%s\n", ragQuery.Query)
	
	fmt.Println("✅ All RAG package types are working correctly!")
}