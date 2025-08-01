package rag

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// ChunkingService provides intelligent document chunking for telecom specifications
type ChunkingService struct {
	config  *ChunkingConfig
	logger  *slog.Logger
	metrics *ChunkingMetrics
	mutex   sync.RWMutex
}

// ChunkingConfig holds configuration for the chunking service
type ChunkingConfig struct {
	// Basic chunking parameters
	ChunkSize        int `json:"chunk_size"`         // Target chunk size in characters
	ChunkOverlap     int `json:"chunk_overlap"`      // Overlap between chunks in characters
	MinChunkSize     int `json:"min_chunk_size"`     // Minimum viable chunk size
	MaxChunkSize     int `json:"max_chunk_size"`     // Maximum chunk size before force split
	
	// Hierarchy preservation
	PreserveHierarchy    bool `json:"preserve_hierarchy"`     // Maintain document structure
	MaxHierarchyDepth    int  `json:"max_hierarchy_depth"`    // Maximum depth to track
	IncludeParentContext bool `json:"include_parent_context"` // Include parent section context
	
	// Semantic chunking
	UseSemanticBoundaries bool `json:"use_semantic_boundaries"` // Respect semantic boundaries
	SentenceBoundaryWeight float64 `json:"sentence_boundary_weight"` // Weight for sentence boundaries
	ParagraphBoundaryWeight float64 `json:"paragraph_boundary_weight"` // Weight for paragraph boundaries
	SectionBoundaryWeight   float64 `json:"section_boundary_weight"`   // Weight for section boundaries
	
	// Telecom-specific chunking
	PreserveTechnicalTerms bool     `json:"preserve_technical_terms"` // Don't split technical terms
	TechnicalTermPatterns  []string `json:"technical_term_patterns"`  // Patterns for technical terms
	PreserveTablesAndFigures bool   `json:"preserve_tables_figures"`  // Keep tables/figures intact
	PreserveCodeBlocks      bool     `json:"preserve_code_blocks"`     // Keep code blocks intact
	
	// Quality control
	MinContentRatio    float64 `json:"min_content_ratio"`    // Minimum content vs metadata ratio
	MaxEmptyLines      int     `json:"max_empty_lines"`      // Maximum consecutive empty lines
	FilterNoiseContent bool    `json:"filter_noise_content"` // Remove headers, footers, etc.
	
	// Performance settings
	BatchSize      int `json:"batch_size"`      // Documents to process in parallel
	MaxConcurrency int `json:"max_concurrency"` // Maximum concurrent processing
	
	// Context enhancement
	AddSectionHeaders   bool `json:"add_section_headers"`   // Include section headers in chunks
	AddDocumentMetadata bool `json:"add_document_metadata"` // Include document metadata
	AddChunkMetadata    bool `json:"add_chunk_metadata"`    // Include chunk-specific metadata
}

// DocumentChunk represents a processed chunk with hierarchy and context
type DocumentChunk struct {
	// Basic chunk information
	ID              string    `json:"id"`
	DocumentID      string    `json:"document_id"`
	Content         string    `json:"content"`
	CleanContent    string    `json:"clean_content"`
	ChunkIndex      int       `json:"chunk_index"`
	StartOffset     int       `json:"start_offset"`
	EndOffset       int       `json:"end_offset"`
	
	// Size and quality metrics
	CharacterCount  int     `json:"character_count"`
	WordCount       int     `json:"word_count"`
	SentenceCount   int     `json:"sentence_count"`
	ContentRatio    float64 `json:"content_ratio"`
	QualityScore    float64 `json:"quality_score"`
	
	// Hierarchy information
	HierarchyPath   []string `json:"hierarchy_path"`   // Path from root to this chunk
	SectionTitle    string   `json:"section_title"`    // Immediate section title
	ParentSections  []string `json:"parent_sections"`  // All parent section titles
	HierarchyLevel  int      `json:"hierarchy_level"`  // Depth in document hierarchy
	
	// Context information
	PreviousContext string   `json:"previous_context"` // Overlap with previous chunk
	NextContext     string   `json:"next_context"`     // Overlap with next chunk
	ParentContext   string   `json:"parent_context"`   // Context from parent section
	
	// Technical metadata
	TechnicalTerms    []string `json:"technical_terms"`     // Identified technical terms
	ContainsTable     bool     `json:"contains_table"`      // Contains table data
	ContainsFigure    bool     `json:"contains_figure"`     // Contains figure references
	ContainsCode      bool     `json:"contains_code"`       // Contains code blocks
	ContainsFormula   bool     `json:"contains_formula"`    // Contains mathematical formulas
	ContainsReference bool     `json:"contains_reference"`  // Contains references/citations
	
	// Semantic information
	ChunkType       string   `json:"chunk_type"`       // text, table, figure, code, etc.
	SemanticTags    []string `json:"semantic_tags"`    // Semantic classification tags
	KeywordDensity  map[string]float64 `json:"keyword_density"` // Keyword frequency analysis
	
	// Processing metadata
	ProcessedAt     time.Time `json:"processed_at"`
	ProcessingTime  time.Duration `json:"processing_time"`
	ChunkingMethod  string    `json:"chunking_method"`  // Method used for chunking
	
	// Document metadata (inherited)
	DocumentMetadata *DocumentMetadata `json:"document_metadata,omitempty"`
}

// ChunkingMetrics tracks chunking performance and quality
type ChunkingMetrics struct {
	TotalDocuments     int64         `json:"total_documents"`
	TotalChunks        int64         `json:"total_chunks"`
	AverageChunksPerDoc float64      `json:"average_chunks_per_doc"`
	AverageChunkSize   float64       `json:"average_chunk_size"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	
	QualityMetrics struct {
		AverageQualityScore  float64 `json:"average_quality_score"`
		HighQualityChunks    int64   `json:"high_quality_chunks"`
		MediumQualityChunks  int64   `json:"medium_quality_chunks"`
		LowQualityChunks     int64   `json:"low_quality_chunks"`
	} `json:"quality_metrics"`
	
	TypeDistribution map[string]int64 `json:"type_distribution"`
	HierarchyDepthStats map[int]int64 `json:"hierarchy_depth_stats"`
	
	LastProcessedAt time.Time `json:"last_processed_at"`
	mutex          sync.RWMutex
}

// DocumentStructure represents the hierarchical structure of a document
type DocumentStructure struct {
	Sections      []Section `json:"sections"`
	TableOfContents []TOCEntry `json:"table_of_contents"`
	Figures       []Figure  `json:"figures"`
	Tables        []Table   `json:"tables"`
	References    []Reference `json:"references"`
}

// Section represents a document section with hierarchy
type Section struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Level        int       `json:"level"`
	StartOffset  int       `json:"start_offset"`
	EndOffset    int       `json:"end_offset"`
	Content      string    `json:"content"`
	Parent       *Section  `json:"parent,omitempty"`
	Children     []Section `json:"children"`
	SectionNumber string   `json:"section_number"`
}

// TOCEntry represents a table of contents entry
type TOCEntry struct {
	Title       string `json:"title"`
	Level       int    `json:"level"`
	PageNumber  int    `json:"page_number"`
	SectionNum  string `json:"section_number"`
	StartOffset int    `json:"start_offset"`
}

// Figure represents a figure reference
type Figure struct {
	ID          string `json:"id"`
	Caption     string `json:"caption"`
	Number      int    `json:"number"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
}

// Table represents a table with its structure
type Table struct {
	ID          string `json:"id"`
	Caption     string `json:"caption"`
	Number      int    `json:"number"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Headers     []string `json:"headers"`
	RowCount    int      `json:"row_count"`
}

// Reference represents a bibliographic reference
type Reference struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	StartOffset int    `json:"start_offset"`
	Type        string `json:"type"` // citation, standard, specification, etc.
}

// NewChunkingService creates a new chunking service with the specified configuration
func NewChunkingService(config *ChunkingConfig) *ChunkingService {
	if config == nil {
		config = getDefaultChunkingConfig()
	}

	return &ChunkingService{
		config:  config,
		logger:  slog.Default().With("component", "chunking-service"),
		metrics: &ChunkingMetrics{
			TypeDistribution:    make(map[string]int64),
			HierarchyDepthStats: make(map[int]int64),
			LastProcessedAt:     time.Now(),
		},
	}
}

// getDefaultChunkingConfig returns default configuration for the chunking service
func getDefaultChunkingConfig() *ChunkingConfig {
	return &ChunkingConfig{
		ChunkSize:               512,   // Optimized for telecom domain (512 tokens)
		ChunkOverlap:            50,    // Optimized overlap (50 tokens)
		MinChunkSize:            50,    // Smaller minimum for technical content
		MaxChunkSize:            1024,  // Reasonable maximum
		PreserveHierarchy:       true,
		MaxHierarchyDepth:       10,
		IncludeParentContext:    true,
		UseSemanticBoundaries:   true,
		SentenceBoundaryWeight:  1.0,
		ParagraphBoundaryWeight: 2.0,
		SectionBoundaryWeight:   3.0,
		PreserveTechnicalTerms:  true,
		TechnicalTermPatterns: []string{
			`\b[A-Z]{2,}(?:-[A-Z]{2,})*\b`,           // Acronyms like RAN, AMF, SMF
			`\b\d+G\b`,                               // Technology generations like 5G, 4G
			`\b(?:Rel|Release)[-\s]*\d+\b`,           // Release versions
			`\b[vV]\d+\.\d+(?:\.\d+)?\b`,            // Version numbers
			`\b\d+\.\d+\.\d+\b`,                     // Specification numbers
			`\b[A-Z]+\d+\b`,                         // Standards like TS38.300
		},
		PreserveTablesAndFigures: true,
		PreserveCodeBlocks:       true,
		MinContentRatio:          0.5,
		MaxEmptyLines:            3,
		FilterNoiseContent:       true,
		BatchSize:                10,
		MaxConcurrency:           5,
		AddSectionHeaders:        true,
		AddDocumentMetadata:      true,
		AddChunkMetadata:         true,
	}
}

// ChunkDocument processes a loaded document and returns intelligent chunks
func (cs *ChunkingService) ChunkDocument(ctx context.Context, doc *LoadedDocument) ([]*DocumentChunk, error) {
	startTime := time.Now()

	cs.logger.Info("Starting document chunking",
		"document_id", doc.ID,
		"content_length", len(doc.Content),
		"title", doc.Title,
	)

	// Analyze document structure
	structure, err := cs.analyzeDocumentStructure(doc.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze document structure: %w", err)
	}

	// Create chunks based on the analyzed structure
	chunks, err := cs.createIntelligentChunks(ctx, doc, structure)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunks: %w", err)
	}

	// Post-process chunks for quality and consistency
	chunks = cs.postProcessChunks(chunks, doc, structure)

	// Update metrics
	processingTime := time.Since(startTime)
	cs.updateMetrics(func(m *ChunkingMetrics) {
		m.TotalDocuments++
		m.TotalChunks += int64(len(chunks))
		m.AverageChunksPerDoc = float64(m.TotalChunks) / float64(m.TotalDocuments)
		m.AverageProcessingTime = (m.AverageProcessingTime*time.Duration(m.TotalDocuments-1) + processingTime) / time.Duration(m.TotalDocuments)
		m.LastProcessedAt = time.Now()
		
		// Update type distribution
		for _, chunk := range chunks {
			m.TypeDistribution[chunk.ChunkType]++
			m.HierarchyDepthStats[chunk.HierarchyLevel]++
		}
	})

	cs.logger.Info("Document chunking completed",
		"document_id", doc.ID,
		"chunks_created", len(chunks),
		"processing_time", processingTime,
	)

	return chunks, nil
}

// analyzeDocumentStructure analyzes the document to identify its hierarchical structure
func (cs *ChunkingService) analyzeDocumentStructure(content string) (*DocumentStructure, error) {
	structure := &DocumentStructure{
		Sections:        []Section{},
		TableOfContents: []TOCEntry{},
		Figures:         []Figure{},
		Tables:          []Table{},
		References:      []Reference{},
	}

	// Extract sections with hierarchy
	sections := cs.extractSections(content)
	structure.Sections = cs.buildSectionHierarchy(sections)

	// Extract figures
	structure.Figures = cs.extractFigures(content)

	// Extract tables
	structure.Tables = cs.extractTables(content)

	// Extract references
	structure.References = cs.extractReferences(content)

	cs.logger.Debug("Document structure analyzed",
		"sections", len(structure.Sections),
		"figures", len(structure.Figures),
		"tables", len(structure.Tables),
		"references", len(structure.References),
	)

	return structure, nil
}

// extractSections identifies section headers and their content
func (cs *ChunkingService) extractSections(content string) []Section {
	var sections []Section

	// Pattern for section headers with numbers (e.g., "1.2.3 Section Title")
	sectionPattern := regexp.MustCompile(`(?m)^(\d+(?:\.\d+)*)\s+(.+)$`)
	
	// Pattern for unnumbered section headers (e.g., "Introduction", "Abstract")
	headerPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?mi)^(abstract|introduction|summary|conclusion|references?|bibliography|appendix\s*[a-z]?)$`),
		regexp.MustCompile(`(?mi)^([A-Z][A-Z\s]+[A-Z])$`), // ALL CAPS headers
		regexp.MustCompile(`(?mi)^(\d+\.\s+[A-Z].+)$`),    // Numbered headers with dot and space
	}

	lines := strings.Split(content, "\n")
	var currentOffset int

	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines
		if line == "" {
			currentOffset += 1 // for the newline
			continue
		}

		var section Section
		var matched bool

		// Check numbered sections first
		if matches := sectionPattern.FindStringSubmatch(line); len(matches) >= 3 {
			section = Section{
				ID:            fmt.Sprintf("section_%d", len(sections)),
				Title:         strings.TrimSpace(matches[2]),
				SectionNumber: matches[1],
				Level:         strings.Count(matches[1], ".") + 1,
				StartOffset:   currentOffset,
			}
			matched = true
		} else {
			// Check unnumbered headers
			for _, pattern := range headerPatterns {
				if pattern.MatchString(line) && len(line) < 100 { // Reasonable title length
					section = Section{
						ID:            fmt.Sprintf("section_%d", len(sections)),
						Title:         line,
						SectionNumber: "",
						Level:         1, // Assume top level for unnumbered
						StartOffset:   currentOffset,
					}
					matched = true
					break
				}
			}
		}

		if matched {
			// Set end offset for previous section
			if len(sections) > 0 {
				sections[len(sections)-1].EndOffset = currentOffset
			}
			
			sections = append(sections, section)
		}

		currentOffset += len(line) + 1 // +1 for newline
	}

	// Set end offset for the last section
	if len(sections) > 0 {
		sections[len(sections)-1].EndOffset = len(content)
	}

	// Extract content for each section
	for i := range sections {
		if i < len(sections)-1 {
			sections[i].Content = content[sections[i].StartOffset:sections[i+1].StartOffset]
		} else {
			sections[i].Content = content[sections[i].StartOffset:sections[i].EndOffset]
		}
		sections[i].Content = strings.TrimSpace(sections[i].Content)
	}

	return sections
}

// buildSectionHierarchy creates a hierarchical structure from flat sections
func (cs *ChunkingService) buildSectionHierarchy(sections []Section) []Section {
	if len(sections) == 0 {
		return sections
	}

	var rootSections []Section
	var stack []*Section // Stack to track parent sections

	for i := range sections {
		section := &sections[i]
		
		// Find the appropriate parent
		for len(stack) > 0 && stack[len(stack)-1].Level >= section.Level {
			stack = stack[:len(stack)-1] // Pop from stack
		}

		// Set parent if stack is not empty
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			section.Parent = parent
			parent.Children = append(parent.Children, *section)
		} else {
			// This is a root section
			rootSections = append(rootSections, *section)
		}

		// Push current section to stack
		stack = append(stack, section)
	}

	return rootSections
}

// extractFigures identifies figure references in the document
func (cs *ChunkingService) extractFigures(content string) []Figure {
	var figures []Figure

	// Pattern for figure references (e.g., "Figure 1: Caption")
	figurePattern := regexp.MustCompile(`(?i)figure\s+(\d+)(?:\s*[:\-]\s*(.+?))?(?:\n|$)`)
	
	matches := figurePattern.FindAllStringIndex(content, -1)
	submatches := figurePattern.FindAllStringSubmatch(content, -1)

	for i, match := range matches {
		if i < len(submatches) {
			submatch := submatches[i]
			figureNum := 0
			if len(submatch) > 1 {
				fmt.Sscanf(submatch[1], "%d", &figureNum)
			}
			
			caption := ""
			if len(submatch) > 2 {
				caption = strings.TrimSpace(submatch[2])
			}

			figure := Figure{
				ID:          fmt.Sprintf("figure_%d", figureNum),
				Number:      figureNum,
				Caption:     caption,
				StartOffset: match[0],
				EndOffset:   match[1],
			}
			figures = append(figures, figure)
		}
	}

	return figures
}

// extractTables identifies table structures in the document
func (cs *ChunkingService) extractTables(content string) []Table {
	var tables []Table

	// Pattern for table headers (e.g., "Table 1: Caption")
	tablePattern := regexp.MustCompile(`(?i)table\s+(\d+)(?:\s*[:\-]\s*(.+?))?(?:\n|$)`)
	
	matches := tablePattern.FindAllStringIndex(content, -1)
	submatches := tablePattern.FindAllStringSubmatch(content, -1)

	for i, match := range matches {
		if i < len(submatches) {
			submatch := submatches[i]
			tableNum := 0
			if len(submatch) > 1 {
				fmt.Sscanf(submatch[1], "%d", &tableNum)
			}
			
			caption := ""
			if len(submatch) > 2 {
				caption = strings.TrimSpace(submatch[2])
			}

			// Try to identify table structure
			tableContent := cs.extractTableContent(content, match[1])
			headers, rowCount := cs.analyzeTableStructure(tableContent)

			table := Table{
				ID:          fmt.Sprintf("table_%d", tableNum),
				Number:      tableNum,
				Caption:     caption,
				StartOffset: match[0],
				EndOffset:   match[1] + len(tableContent),
				Headers:     headers,
				RowCount:    rowCount,
			}
			tables = append(tables, table)
		}
	}

	return tables
}

// extractTableContent extracts table content following a table header
func (cs *ChunkingService) extractTableContent(content string, startPos int) string {
	if startPos >= len(content) {
		return ""
	}

	// Look for tabular data patterns in the next few hundred characters
	searchRange := 1000
	if startPos+searchRange > len(content) {
		searchRange = len(content) - startPos
	}

	searchText := content[startPos : startPos+searchRange]
	
	// Simple heuristic: look for lines with multiple columns (separated by tabs or multiple spaces)
	lines := strings.Split(searchText, "\n")
	var tableLines []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(tableLines) > 0 {
				break // End of table
			}
			continue
		}
		
		// Check if line looks like table data (has multiple columns)
		if cs.looksLikeTableRow(line) {
			tableLines = append(tableLines, line)
		} else if len(tableLines) > 0 {
			break // End of table
		}
	}

	return strings.Join(tableLines, "\n")
}

// looksLikeTableRow determines if a line looks like a table row
func (cs *ChunkingService) looksLikeTableRow(line string) bool {
	// Check for tab separators
	if strings.Count(line, "\t") >= 2 {
		return true
	}
	
	// Check for multiple space separators (3+ spaces)
	multiSpacePattern := regexp.MustCompile(`\s{3,}`)
	if len(multiSpacePattern.FindAllString(line, -1)) >= 2 {
		return true
	}
	
	// Check for pipe separators
	if strings.Count(line, "|") >= 2 {
		return true
	}
	
	return false
}

// analyzeTableStructure analyzes table content to extract headers and row count
func (cs *ChunkingService) analyzeTableStructure(tableContent string) ([]string, int) {
	lines := strings.Split(tableContent, "\n")
	if len(lines) == 0 {
		return nil, 0
	}

	// Try to identify headers from the first line
	firstLine := strings.TrimSpace(lines[0])
	var headers []string
	
	if strings.Contains(firstLine, "\t") {
		headers = strings.Split(firstLine, "\t")
	} else if strings.Contains(firstLine, "|") {
		headers = strings.Split(firstLine, "|")
	} else {
		// Split by multiple spaces
		multiSpacePattern := regexp.MustCompile(`\s{3,}`)
		headers = multiSpacePattern.Split(firstLine, -1)
	}

	// Clean up headers
	for i, header := range headers {
		headers[i] = strings.TrimSpace(header)
	}

	return headers, len(lines)
}

// extractReferences identifies references and citations
func (cs *ChunkingService) extractReferences(content string) []Reference {
	var references []Reference

	// Pattern for reference sections
	refSectionPattern := regexp.MustCompile(`(?i)(?:^|\n)\s*(references?|bibliography)\s*\n`)
	refMatches := refSectionPattern.FindAllStringIndex(content, -1)

	if len(refMatches) > 0 {
		// Extract references from reference section
		refStart := refMatches[0][1]
		refSection := content[refStart:]
		
		// Pattern for numbered references [1], [2], etc.
		refPattern := regexp.MustCompile(`(?m)^\s*\[(\d+)\]\s*(.+?)(?=^\s*\[\d+\]|\z)`)
		refSubmatches := refPattern.FindAllStringSubmatch(refSection, -1)
		
		for _, submatch := range refSubmatches {
			if len(submatch) >= 3 {
				ref := Reference{
					ID:          fmt.Sprintf("ref_%s", submatch[1]),
					Text:        strings.TrimSpace(submatch[2]),
					StartOffset: refStart + strings.Index(refSection, submatch[0]),
					Type:        "citation",
				}
				references = append(references, ref)
			}
		}
	}

	// Also look for inline citations throughout the document
	inlineCitationPattern := regexp.MustCompile(`\[(\d+(?:\s*,\s*\d+)*)\]`)
	citationMatches := inlineCitationPattern.FindAllStringIndex(content, -1)
	
	for _, match := range citationMatches {
		citation := content[match[0]:match[1]]
		ref := Reference{
			ID:          fmt.Sprintf("inline_cite_%d", len(references)),
			Text:        citation,
			StartOffset: match[0],
			Type:        "inline_citation",
		}
		references = append(references, ref)
	}

	return references
}

// createIntelligentChunks creates chunks based on the document structure
func (cs *ChunkingService) createIntelligentChunks(ctx context.Context, doc *LoadedDocument, structure *DocumentStructure) ([]*DocumentChunk, error) {
	var chunks []*DocumentChunk

	if cs.config.PreserveHierarchy && len(structure.Sections) > 0 {
		// Create chunks based on sections
		chunks = cs.createSectionBasedChunks(doc, structure)
	} else {
		// Create chunks using semantic boundaries
		chunks = cs.createSemanticChunks(doc, structure)
	}

	// Add chunk metadata and enhance with context
	for i, chunk := range chunks {
		chunk.ChunkIndex = i
		chunk.ProcessedAt = time.Now()
		chunk.ChunkingMethod = cs.determineChunkingMethod()
		
		if cs.config.AddDocumentMetadata {
			chunk.DocumentMetadata = doc.Metadata
		}
		
		// Calculate quality score
		chunk.QualityScore = cs.calculateChunkQuality(chunk)
		
		// Add technical analysis
		cs.analyzeChunkTechnicalContent(chunk)
		
		// Add semantic tags
		chunk.SemanticTags = cs.extractSemanticTags(chunk.Content)
		
		// Calculate keyword density
		chunk.KeywordDensity = cs.calculateKeywordDensity(chunk.Content)
	}

	// Add context information between chunks
	cs.addChunkContext(chunks)

	return chunks, nil
}

// createSectionBasedChunks creates chunks respecting document section boundaries
func (cs *ChunkingService) createSectionBasedChunks(doc *LoadedDocument, structure *DocumentStructure) []*DocumentChunk {
	var chunks []*DocumentChunk

	// Process sections recursively
	cs.processSectionsRecursively(structure.Sections, doc, []string{}, &chunks)

	return chunks
}

// processSectionsRecursively processes sections and their children
func (cs *ChunkingService) processSectionsRecursively(sections []Section, doc *LoadedDocument, parentPath []string, chunks *[]*DocumentChunk) {
	for _, section := range sections {
		currentPath := append(parentPath, section.Title)
		
		// Create chunks for this section
		sectionChunks := cs.chunkSectionContent(section, doc, currentPath)
		*chunks = append(*chunks, sectionChunks...)
		
		// Process child sections
		if len(section.Children) > 0 {
			cs.processSectionsRecursively(section.Children, doc, currentPath, chunks)
		}
	}
}

// chunkSectionContent creates chunks from a single section
func (cs *ChunkingService) chunkSectionContent(section Section, doc *LoadedDocument, hierarchyPath []string) []*DocumentChunk {
	var chunks []*DocumentChunk

	content := section.Content
	if len(content) <= cs.config.ChunkSize {
		// Section fits in one chunk
		chunk := cs.createChunk(content, doc.ID, section.StartOffset, len(content), hierarchyPath, section.Title, section.Level)
		chunks = append(chunks, chunk)
	} else {
		// Split section into multiple chunks
		sectionChunks := cs.splitContentIntoChunks(content, doc.ID, section.StartOffset, hierarchyPath, section.Title, section.Level)
		chunks = append(chunks, sectionChunks...)
	}

	return chunks
}

// createSemanticChunks creates chunks using semantic boundary detection
func (cs *ChunkingService) createSemanticChunks(doc *LoadedDocument, structure *DocumentStructure) []*DocumentChunk {
	var chunks []*DocumentChunk

	content := doc.Content
	offset := 0

	for offset < len(content) {
		chunkEnd := cs.findOptimalChunkBoundary(content, offset)
		if chunkEnd <= offset {
			chunkEnd = offset + cs.config.ChunkSize
		}
		if chunkEnd > len(content) {
			chunkEnd = len(content)
		}

		chunkContent := content[offset:chunkEnd]
		chunk := cs.createChunk(chunkContent, doc.ID, offset, chunkEnd-offset, []string{"document"}, "", 0)
		chunks = append(chunks, chunk)

		// Move to next chunk with overlap
		offset = chunkEnd - cs.config.ChunkOverlap
		if offset < 0 {
			offset = 0
		}
	}

	return chunks
}

// findOptimalChunkBoundary finds the best place to end a chunk
func (cs *ChunkingService) findOptimalChunkBoundary(content string, start int) int {
	targetEnd := start + cs.config.ChunkSize
	if targetEnd >= len(content) {
		return len(content)
	}

	// Search window around target end
	searchStart := targetEnd - cs.config.ChunkOverlap
	searchEnd := targetEnd + cs.config.ChunkOverlap
	if searchStart < start {
		searchStart = start
	}
	if searchEnd > len(content) {
		searchEnd = len(content)
	}

	bestBoundary := targetEnd
	bestScore := 0.0

	// Look for optimal boundaries in the search window
	for i := searchStart; i < searchEnd; i++ {
		score := cs.calculateBoundaryScore(content, i)
		if score > bestScore {
			bestScore = score
			bestBoundary = i
		}
	}

	return bestBoundary
}

// calculateBoundaryScore calculates how good a position is for a chunk boundary
func (cs *ChunkingService) calculateBoundaryScore(content string, pos int) float64 {
	if pos >= len(content) {
		return 0.0
	}

	score := 0.0
	char := rune(content[pos])

	// Sentence boundary (period, exclamation, question mark followed by space/newline)
	if pos > 0 && pos < len(content)-1 {
		prevChar := rune(content[pos-1])
		nextChar := rune(content[pos])
		
		if (prevChar == '.' || prevChar == '!' || prevChar == '?') && (nextChar == ' ' || nextChar == '\n') {
			score += cs.config.SentenceBoundaryWeight
		}
	}

	// Paragraph boundary (double newline)
	if pos > 1 && pos < len(content)-1 {
		if content[pos-1] == '\n' && char == '\n' {
			score += cs.config.ParagraphBoundaryWeight
		}
	}

	// Section boundary (detect section headers)
	if char == '\n' && pos < len(content)-50 {
		lineStart := pos + 1
		lineEnd := strings.Index(content[lineStart:], "\n")
		if lineEnd == -1 {
			lineEnd = len(content) - lineStart
		} else {
			lineEnd += lineStart
		}
		
		line := strings.TrimSpace(content[lineStart:lineEnd])
		if cs.looksLikeSectionHeader(line) {
			score += cs.config.SectionBoundaryWeight
		}
	}

	// Avoid breaking technical terms
	if cs.config.PreserveTechnicalTerms {
		if cs.isInsideTechnicalTerm(content, pos) {
			score -= 2.0 // Penalty for breaking technical terms
		}
	}

	return score
}

// looksLikeSectionHeader determines if a line looks like a section header
func (cs *ChunkingService) looksLikeSectionHeader(line string) bool {
	if len(line) == 0 || len(line) > 100 {
		return false
	}

	// Check for numbered sections
	if matched, _ := regexp.MatchString(`^\d+(?:\.\d+)*\s+\w+`, line); matched {
		return true
	}

	// Check for common section titles
	commonHeaders := []string{
		"introduction", "abstract", "summary", "conclusion", "background",
		"methodology", "implementation", "results", "discussion", "references",
		"appendix", "overview", "architecture", "requirements", "specifications",
	}

	lowerLine := strings.ToLower(line)
	for _, header := range commonHeaders {
		if strings.Contains(lowerLine, header) {
			return true
		}
	}

	// Check if line is all caps (potential header)
	if len(line) > 3 && strings.ToUpper(line) == line && strings.Contains(line, " ") {
		return true
	}

	return false
}

// isInsideTechnicalTerm checks if a position is inside a technical term
func (cs *ChunkingService) isInsideTechnicalTerm(content string, pos int) bool {
	// Look at a small window around the position
	windowSize := 20
	start := pos - windowSize
	if start < 0 {
		start = 0
	}
	end := pos + windowSize
	if end > len(content) {
		end = len(content)
	}

	window := content[start:end]
	relativePos := pos - start

	// Check against technical term patterns
	for _, pattern := range cs.config.TechnicalTermPatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringIndex(window, -1)
		
		for _, match := range matches {
			if relativePos >= match[0] && relativePos <= match[1] {
				return true
			}
		}
	}

	return false
}

// splitContentIntoChunks splits content into multiple chunks with overlap
func (cs *ChunkingService) splitContentIntoChunks(content, docID string, baseOffset int, hierarchyPath []string, sectionTitle string, level int) []*DocumentChunk {
	var chunks []*DocumentChunk

	offset := 0

	for offset < len(content) {
		chunkEnd := offset + cs.config.ChunkSize
		if chunkEnd > len(content) {
			chunkEnd = len(content)
		}

		// Try to find a better boundary
		if cs.config.UseSemanticBoundaries && chunkEnd < len(content) {
			betterEnd := cs.findOptimalChunkBoundary(content, offset)
			if betterEnd > offset && betterEnd <= len(content) {
				chunkEnd = betterEnd
			}
		}

		chunkContent := content[offset:chunkEnd]
		chunk := cs.createChunk(chunkContent, docID, baseOffset+offset, chunkEnd-offset, hierarchyPath, sectionTitle, level)
		chunks = append(chunks, chunk)

		// Move to next chunk with overlap
		nextOffset := chunkEnd - cs.config.ChunkOverlap
		if nextOffset <= offset {
			nextOffset = offset + 1 // Ensure progress
		}
		offset = nextOffset
	}

	return chunks
}

// createChunk creates a single document chunk with metadata
func (cs *ChunkingService) createChunk(content, docID string, startOffset, length int, hierarchyPath []string, sectionTitle string, level int) *DocumentChunk {
	chunkID := fmt.Sprintf("%s_chunk_%d", docID, startOffset)
	
	chunk := &DocumentChunk{
		ID:             chunkID,
		DocumentID:     docID,
		Content:        content,
		CleanContent:   cs.cleanChunkContent(content),
		StartOffset:    startOffset,
		EndOffset:      startOffset + length,
		CharacterCount: len(content),
		WordCount:      cs.countWords(content),
		SentenceCount:  cs.countSentences(content),
		HierarchyPath:  hierarchyPath,
		SectionTitle:   sectionTitle,
		HierarchyLevel: level,
		ChunkType:      "text", // Default, will be updated by analysis
	}

	// Add parent sections (all but the last element of hierarchy path)
	if len(hierarchyPath) > 1 {
		chunk.ParentSections = hierarchyPath[:len(hierarchyPath)-1]
	}

	// Calculate content ratio (text vs whitespace/formatting)
	chunk.ContentRatio = cs.calculateContentRatio(content)

	return chunk
}

// cleanChunkContent cleans and normalizes chunk content
func (cs *ChunkingService) cleanChunkContent(content string) string {
	// Remove excessive whitespace
	cleaned := regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	
	// Remove common artifacts if noise filtering is enabled
	if cs.config.FilterNoiseContent {
		// Remove page numbers
		cleaned = regexp.MustCompile(`(?i)page\s+\d+(?:\s+of\s+\d+)?`).ReplaceAllString(cleaned, "")
		
		// Remove excessive dashes/underscores (likely formatting)
		cleaned = regexp.MustCompile(`[-_]{5,}`).ReplaceAllString(cleaned, "")
		
		// Remove copyright notices
		cleaned = regexp.MustCompile(`(?i)©\s*\d{4}.*?(?:\n|$)`).ReplaceAllString(cleaned, "")
	}

	return strings.TrimSpace(cleaned)
}

// countWords counts the number of words in the content
func (cs *ChunkingService) countWords(content string) int {
	words := strings.Fields(content)
	return len(words)
}

// countSentences counts the number of sentences in the content
func (cs *ChunkingService) countSentences(content string) int {
	// Simple sentence counting based on sentence-ending punctuation
	sentencePattern := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentencePattern.Split(content, -1)
	
	// Filter out very short "sentences" (likely abbreviations)
	count := 0
	for _, sentence := range sentences {
		if len(strings.TrimSpace(sentence)) > 10 {
			count++
		}
	}
	
	return count
}

// calculateContentRatio calculates the ratio of meaningful content to total characters
func (cs *ChunkingService) calculateContentRatio(content string) float64 {
	if len(content) == 0 {
		return 0.0
	}

	meaningfulChars := 0
	for _, char := range content {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			meaningfulChars++
		}
	}

	return float64(meaningfulChars) / float64(len(content))
}

// analyzeChunkTechnicalContent analyzes chunk for technical content
func (cs *ChunkingService) analyzeChunkTechnicalContent(chunk *DocumentChunk) {
	content := strings.ToLower(chunk.Content)

	// Check for tables
	chunk.ContainsTable = cs.containsTableData(chunk.Content)

	// Check for figures
	chunk.ContainsFigure = regexp.MustCompile(`(?i)\bfigure\s+\d+`).MatchString(chunk.Content)

	// Check for code blocks
	chunk.ContainsCode = cs.containsCodeBlock(chunk.Content)

	// Check for formulas
	chunk.ContainsFormula = cs.containsFormula(chunk.Content)

	// Check for references
	chunk.ContainsReference = regexp.MustCompile(`\[\d+\]|\b(?:ref|reference)\b`).MatchString(content)

	// Extract technical terms
	chunk.TechnicalTerms = cs.extractTechnicalTermsFromChunk(chunk.Content)

	// Determine chunk type
	chunk.ChunkType = cs.determineChunkType(chunk)
}

// containsTableData checks if chunk contains tabular data
func (cs *ChunkingService) containsTableData(content string) bool {
	lines := strings.Split(content, "\n")
	tableRowCount := 0

	for _, line := range lines {
		if cs.looksLikeTableRow(line) {
			tableRowCount++
		}
	}

	return tableRowCount >= 2 // At least 2 rows that look like table data
}

// containsCodeBlock checks if chunk contains code blocks
func (cs *ChunkingService) containsCodeBlock(content string) bool {
	// Check for common code indicators
	codePatterns := []string{
		"```", // Markdown code blocks
		"```", // Alternative code blocks
		"function ", "class ", "def ", "void ", "int ", "string ",
		"import ", "include ", "#include",
		"{\n", "}\n", // Brace patterns
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range codePatterns {
		if strings.Contains(lowerContent, pattern) {
			return true
		}
	}

	return false
}

// containsFormula checks if chunk contains mathematical formulas
func (cs *ChunkingService) containsFormula(content string) bool {
	// Check for mathematical notation
	formulaPatterns := []string{
		"=", "+", "-", "*", "/", "^", "√",
		"∑", "∫", "π", "α", "β", "γ", "θ", "λ", "μ", "σ",
		"log", "ln", "sin", "cos", "tan",
	}

	mathSymbolCount := 0
	for _, pattern := range formulaPatterns {
		mathSymbolCount += strings.Count(content, pattern)
	}

	// Also check for LaTeX-style formulas
	if strings.Contains(content, "$") || strings.Contains(content, "\\") {
		mathSymbolCount += 10 // Higher weight for LaTeX
	}

	return mathSymbolCount >= 3 // Threshold for formula detection
}

// extractTechnicalTermsFromChunk extracts technical terms from a chunk
func (cs *ChunkingService) extractTechnicalTermsFromChunk(content string) []string {
	var terms []string
	termSet := make(map[string]bool)

	for _, pattern := range cs.config.TechnicalTermPatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllString(content, -1)
		
		for _, match := range matches {
			term := strings.TrimSpace(match)
			if !termSet[term] && len(term) > 1 {
				terms = append(terms, term)
				termSet[term] = true
			}
		}
	}

	return terms
}

// determineChunkType determines the primary type of content in the chunk
func (cs *ChunkingService) determineChunkType(chunk *DocumentChunk) string {
	if chunk.ContainsTable {
		return "table"
	}
	if chunk.ContainsFigure {
		return "figure"
	}
	if chunk.ContainsCode {
		return "code"
	}
	if chunk.ContainsFormula {
		return "formula"
	}
	if len(chunk.TechnicalTerms) > 5 {
		return "technical"
	}
	if chunk.ContainsReference {
		return "reference"
	}
	
	return "text"
}

// extractSemanticTags extracts semantic classification tags
func (cs *ChunkingService) extractSemanticTags(content string) []string {
	var tags []string
	lowerContent := strings.ToLower(content)

	// Define semantic categories and their keywords
	semanticCategories := map[string][]string{
		"architecture": {"architecture", "design", "structure", "component", "interface"},
		"protocol": {"protocol", "message", "procedure", "signaling", "handshake"},
		"configuration": {"configuration", "parameter", "setting", "option", "value"},
		"performance": {"performance", "throughput", "latency", "efficiency", "optimization"},
		"security": {"security", "authentication", "encryption", "key", "certificate"},
		"requirements": {"requirement", "shall", "must", "should", "mandatory"},
		"specification": {"specification", "standard", "defined", "described", "specified"},
		"implementation": {"implementation", "algorithm", "method", "approach", "solution"},
	}

	for category, keywords := range semanticCategories {
		for _, keyword := range keywords {
			if strings.Contains(lowerContent, keyword) {
				tags = append(tags, category)
				break // Only add category once
			}
		}
	}

	return tags
}

// calculateKeywordDensity calculates keyword frequency density
func (cs *ChunkingService) calculateKeywordDensity(content string) map[string]float64 {
	words := strings.Fields(strings.ToLower(content))
	wordCount := make(map[string]int)
	totalWords := len(words)

	// Count word frequencies
	for _, word := range words {
		// Clean word (remove punctuation)
		word = regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		if len(word) > 2 { // Ignore very short words
			wordCount[word]++
		}
	}

	// Calculate density for significant words
	density := make(map[string]float64)
	for word, count := range wordCount {
		if count > 1 { // Only include words that appear more than once
			density[word] = float64(count) / float64(totalWords)
		}
	}

	return density
}

// postProcessChunks performs final processing and quality checks on chunks
func (cs *ChunkingService) postProcessChunks(chunks []*DocumentChunk, doc *LoadedDocument, structure *DocumentStructure) []*DocumentChunk {
	var processedChunks []*DocumentChunk

	for _, chunk := range chunks {
		// Filter out low-quality chunks
		if chunk.ContentRatio < cs.config.MinContentRatio {
			cs.logger.Debug("Filtering out low-quality chunk", "chunk_id", chunk.ID, "content_ratio", chunk.ContentRatio)
			continue
		}

		// Ensure minimum chunk size
		if len(chunk.CleanContent) < cs.config.MinChunkSize {
			cs.logger.Debug("Filtering out too-small chunk", "chunk_id", chunk.ID, "size", len(chunk.CleanContent))
			continue
		}

		// Add section headers if configured
		if cs.config.AddSectionHeaders && chunk.SectionTitle != "" {
			chunk.Content = fmt.Sprintf("Section: %s\n\n%s", chunk.SectionTitle, chunk.Content)
			chunk.CleanContent = fmt.Sprintf("Section: %s\n\n%s", chunk.SectionTitle, chunk.CleanContent)
		}

		processedChunks = append(processedChunks, chunk)
	}

	return processedChunks
}

// addChunkContext adds contextual information between chunks
func (cs *ChunkingService) addChunkContext(chunks []*DocumentChunk) {
	for i, chunk := range chunks {
		// Add previous context (overlap with previous chunk)
		if i > 0 && cs.config.ChunkOverlap > 0 {
			prevChunk := chunks[i-1]
			overlapSize := cs.config.ChunkOverlap
			if len(prevChunk.Content) < overlapSize {
				overlapSize = len(prevChunk.Content)
			}
			chunk.PreviousContext = prevChunk.Content[len(prevChunk.Content)-overlapSize:]
		}

		// Add next context (overlap with next chunk)
		if i < len(chunks)-1 && cs.config.ChunkOverlap > 0 {
			nextChunk := chunks[i+1]
			overlapSize := cs.config.ChunkOverlap
			if len(nextChunk.Content) < overlapSize {
				overlapSize = len(nextChunk.Content)
			}
			chunk.NextContext = nextChunk.Content[:overlapSize]
		}

		// Add parent context if configured
		if cs.config.IncludeParentContext && len(chunk.ParentSections) > 0 {
			parentContext := strings.Join(chunk.ParentSections, " > ")
			chunk.ParentContext = parentContext
		}
	}
}

// calculateChunkQuality calculates a quality score for the chunk
func (cs *ChunkingService) calculateChunkQuality(chunk *DocumentChunk) float64 {
	score := 0.0

	// Content ratio (meaningful characters vs total)
	score += chunk.ContentRatio * 40

	// Length appropriateness (not too short, not too long)
	lengthScore := 0.0
	if chunk.CharacterCount >= cs.config.MinChunkSize && chunk.CharacterCount <= cs.config.MaxChunkSize {
		lengthScore = 20.0
	} else if chunk.CharacterCount < cs.config.MinChunkSize {
		lengthScore = float64(chunk.CharacterCount) / float64(cs.config.MinChunkSize) * 20.0
	} else {
		lengthScore = 20.0 - (float64(chunk.CharacterCount-cs.config.MaxChunkSize) / float64(cs.config.MaxChunkSize) * 10.0)
	}
	if lengthScore < 0 {
		lengthScore = 0
	}
	score += lengthScore

	// Technical content bonus
	if len(chunk.TechnicalTerms) > 0 {
		score += float64(len(chunk.TechnicalTerms)) * 2
		if score > 100 {
			score = 100
		}
	}

	// Sentence structure (well-formed sentences)
	if chunk.SentenceCount > 0 {
		avgWordsPerSentence := float64(chunk.WordCount) / float64(chunk.SentenceCount)
		if avgWordsPerSentence >= 5 && avgWordsPerSentence <= 30 {
			score += 20
		} else {
			score += 10
		}
	}

	// Hierarchy context bonus
	if len(chunk.HierarchyPath) > 0 {
		score += 10
	}

	// Semantic tags bonus
	score += float64(len(chunk.SemanticTags)) * 2

	// Cap the score at 100
	if score > 100 {
		score = 100
	}

	return score
}

// determineChunkingMethod returns a description of the chunking method used
func (cs *ChunkingService) determineChunkingMethod() string {
	methods := []string{}

	if cs.config.PreserveHierarchy {
		methods = append(methods, "hierarchy-aware")
	}
	if cs.config.UseSemanticBoundaries {
		methods = append(methods, "semantic-boundaries")
	}
	if cs.config.PreserveTechnicalTerms {
		methods = append(methods, "technical-preserving")
	}

	if len(methods) == 0 {
		return "basic-chunking"
	}

	return strings.Join(methods, "+")
}

// updateMetrics safely updates the chunking metrics
func (cs *ChunkingService) updateMetrics(updater func(*ChunkingMetrics)) {
	cs.metrics.mutex.Lock()
	defer cs.metrics.mutex.Unlock()
	updater(cs.metrics)
}

// GetMetrics returns the current chunking metrics
func (cs *ChunkingService) GetMetrics() *ChunkingMetrics {
	cs.metrics.mutex.RLock()
	defer cs.metrics.mutex.RUnlock()
	
	// Return a copy
	metrics := *cs.metrics
	return &metrics
}

// ChunkDocuments processes multiple documents in parallel
func (cs *ChunkingService) ChunkDocuments(ctx context.Context, docs []*LoadedDocument) ([]*DocumentChunk, error) {
	var allChunks []*DocumentChunk
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create a semaphore to limit concurrency
	semaphore := make(chan struct{}, cs.config.MaxConcurrency)

	// Process documents in batches
	for i := 0; i < len(docs); i += cs.config.BatchSize {
		end := i + cs.config.BatchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := docs[i:end]

		for _, doc := range batch {
			wg.Add(1)
			go func(d *LoadedDocument) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				chunks, err := cs.ChunkDocument(ctx, d)
				if err != nil {
					cs.logger.Error("Failed to chunk document", "doc_id", d.ID, "error", err)
					return
				}

				mu.Lock()
				allChunks = append(allChunks, chunks...)
				mu.Unlock()
			}(doc)
		}
	}

	wg.Wait()

	cs.logger.Info("Batch chunking completed",
		"documents", len(docs),
		"total_chunks", len(allChunks),
	)

	return allChunks, nil
}