//go:build test




package rag



import (

	"time"



	"github.com/nephio-project/nephoran-intent-operator/pkg/shared"

)



// DocumentChunk represents a document chunk used in RAG processing.

// This test-only version provides the type when build tag 'test' is used.

type DocumentChunk struct {

	// Basic chunk information.

	ID           string `json:"id"`

	DocumentID   string `json:"document_id"`

	Content      string `json:"content"`

	CleanContent string `json:"clean_content"`

	ChunkIndex   int    `json:"chunk_index"`

	StartOffset  int    `json:"start_offset"`

	EndOffset    int    `json:"end_offset"`



	// Size and quality metrics.

	CharacterCount int     `json:"character_count"`

	WordCount      int     `json:"word_count"`

	SentenceCount  int     `json:"sentence_count"`

	ContentRatio   float64 `json:"content_ratio"`

	QualityScore   float64 `json:"quality_score"`



	// Hierarchy information.

	HierarchyPath  []string `json:"hierarchy_path"`  // Path from root to this chunk

	SectionTitle   string   `json:"section_title"`   // Immediate section title

	ParentSections []string `json:"parent_sections"` // All parent section titles

	HierarchyLevel int      `json:"hierarchy_level"` // Depth in document hierarchy



	// Context information.

	PreviousContext string `json:"previous_context"` // Overlap with previous chunk

	NextContext     string `json:"next_context"`     // Overlap with next chunk

	ParentContext   string `json:"parent_context"`   // Context from parent section



	// Technical metadata.

	TechnicalTerms    []string `json:"technical_terms"`    // Identified technical terms

	ContainsTable     bool     `json:"contains_table"`     // Contains table data

	ContainsFigure    bool     `json:"contains_figure"`    // Contains figure references

	ContainsCode      bool     `json:"contains_code"`      // Contains code blocks

	ContainsFormula   bool     `json:"contains_formula"`   // Contains mathematical formulas

	ContainsReference bool     `json:"contains_reference"` // Contains references/citations



	// Semantic information.

	ChunkType      string             `json:"chunk_type"`      // text, table, figure, code, etc.

	SemanticTags   []string           `json:"semantic_tags"`   // Semantic classification tags

	KeywordDensity map[string]float64 `json:"keyword_density"` // Keyword frequency analysis



	// Processing metadata.

	ProcessedAt    time.Time     `json:"processed_at"`

	ProcessingTime time.Duration `json:"processing_time"`

	ChunkingMethod string        `json:"chunking_method"` // Method used for chunking



	Metadata map[string]interface{} `json:"metadata,omitempty"`

}



// SearchQuery type alias to shared.SearchQuery for consistency.

type SearchQuery = shared.SearchQuery

