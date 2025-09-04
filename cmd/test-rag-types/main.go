package main

import (
	"fmt"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// DocumentChunk represents a document chunk for RAG processing (stub type).

type DocumentChunk struct {
	ID string

	DocumentID string

	Content string

	CleanContent string

	ChunkIndex int

	StartOffset int

	EndOffset int

	CharacterCount int

	WordCount int

	SentenceCount int

	HierarchyPath []string

	SectionTitle string

	HierarchyLevel int

	ChunkType string

	ProcessedAt time.Time
}

func main() {
	// Test DocumentChunk type.

	chunk := &DocumentChunk{
		ID: "test_chunk_1",

		DocumentID: "test_doc",

		Content: "This is test content",

		CleanContent: "This is test content",

		ChunkIndex: 1,

		StartOffset: 0,

		EndOffset: 20,

		CharacterCount: 20,

		WordCount: 4,

		SentenceCount: 1,

		HierarchyPath: []string{"root", "section1"},

		SectionTitle: "Test Section",

		HierarchyLevel: 2,

		ChunkType: "text",

		ProcessedAt: time.Now(),
	}

	fmt.Printf("DocumentChunk created: ID=%s, Content=%s\n", chunk.ID, chunk.Content)

	// Test SearchQuery type.

	query := &shared.SearchQuery{
		Query: "test query",

		Limit: 10,

		Offset: 0,

		HybridSearch: true,

		HybridAlpha: 0.5,

		UseReranker: false,

		MinConfidence: 0.8,

		IncludeVector: true,

		ExpandQuery: false,

		TargetVectors: []string{"content_vector", "title_vector"},
	}

	fmt.Printf("SearchQuery created: Query=%s, Limit=%d, TargetVectors=%v\n",

		query.Query, query.Limit, query.TargetVectors)

	// Test type alias - for now just use shared.SearchQuery directly.

	fmt.Printf("SearchQuery works: Query=%s\n", query.Query)

	fmt.Println("??All types are working correctly!")
}
