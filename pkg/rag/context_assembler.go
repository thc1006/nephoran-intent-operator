package rag

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
)

// ContextAssembler assembles coherent context from search results
type ContextAssembler struct {
	config *RetrievalConfig
	logger *slog.Logger
	mutex  sync.RWMutex
}

// ContextAssemblyStrategy defines how context should be assembled
type ContextAssemblyStrategy string

const (
	StrategyRankBased      ContextAssemblyStrategy = "rank_based"      // Use results in rank order
	StrategyHierarchical   ContextAssemblyStrategy = "hierarchical"    // Group by document hierarchy
	StrategyTopical        ContextAssemblyStrategy = "topical"         // Group by topic/domain
	StrategyProgressive    ContextAssemblyStrategy = "progressive"     // Build context progressively
	StrategyBalanced       ContextAssemblyStrategy = "balanced"        // Balance different sources/types
)

// ContextSection represents a section of assembled context
type ContextSection struct {
	Title       string                  `json:"title"`
	Content     string                  `json:"content"`
	Source      string                  `json:"source"`
	Relevance   float32                 `json:"relevance"`
	Results     []*EnhancedSearchResult `json:"results"`
	Metadata    map[string]interface{}  `json:"metadata"`
	StartPos    int                     `json:"start_pos"`
	EndPos      int                     `json:"end_pos"`
	SectionType string                  `json:"section_type"` // introduction, main, detail, conclusion
}

// NewContextAssembler creates a new context assembler
func NewContextAssembler(config *RetrievalConfig) *ContextAssembler {
	return &ContextAssembler{
		config: config,
		logger: slog.Default().With("component", "context-assembler"),
	}
}

// AssembleContext assembles coherent context from search results
func (ca *ContextAssembler) AssembleContext(results []*EnhancedSearchResult, request *EnhancedSearchRequest) (string, *ContextMetadata) {
	if len(results) == 0 {
		return "", &ContextMetadata{
			DocumentCount: 0,
			TotalLength:   0,
		}
	}

	ca.logger.Debug("Assembling context",
		"result_count", len(results),
		"intent_type", request.IntentType,
		"max_context_length", ca.config.MaxContextLength,
	)

	// Determine assembly strategy based on intent and results
	strategy := ca.determineAssemblyStrategy(results, request)

	// Create context sections based on strategy
	sections := ca.createContextSections(results, request, strategy)

	// Assemble final context
	assembledContext := ca.assembleContextFromSections(sections, request)

	// Create metadata
	metadata := ca.createContextMetadata(sections, results, assembledContext)

	ca.logger.Info("Context assembly completed",
		"strategy", strategy,
		"section_count", len(sections),
		"context_length", len(assembledContext),
		"documents_used", metadata.DocumentCount,
	)

	return assembledContext, metadata
}

// determineAssemblyStrategy determines the best strategy for context assembly
func (ca *ContextAssembler) determineAssemblyStrategy(results []*EnhancedSearchResult, request *EnhancedSearchRequest) ContextAssemblyStrategy {
	// Default strategy
	strategy := StrategyRankBased

	// Consider intent type
	if request.IntentType != "" {
		switch request.IntentType {
		case "configuration":
			strategy = StrategyProgressive // Build step-by-step context
		case "troubleshooting":
			strategy = StrategyTopical // Group by problem types
		case "optimization":
			strategy = StrategyBalanced // Balance different approaches
		case "monitoring":
			strategy = StrategyHierarchical // Structure by metrics hierarchy
		}
	}

	// Consider result diversity
	if ca.hasHighDiversity(results) {
		strategy = StrategyTopical // Group similar topics together
	}

	// Consider hierarchy information
	if ca.hasRichHierarchy(results) {
		strategy = StrategyHierarchical
	}

	return strategy
}

// hasHighDiversity checks if results have high topic diversity
func (ca *ContextAssembler) hasHighDiversity(results []*EnhancedSearchResult) bool {
	if len(results) < 3 {
		return false
	}

	// Check source diversity
	sources := make(map[string]bool)
	categories := make(map[string]bool)

	for _, result := range results {
		if result.Document != nil {
			sources[result.Document.Source] = true
			categories[result.Document.Category] = true
		}
	}

	// High diversity if we have multiple sources and categories
	return len(sources) >= 3 || len(categories) >= 2
}

// hasRichHierarchy checks if results have rich hierarchical information
func (ca *ContextAssembler) hasRichHierarchy(results []*EnhancedSearchResult) bool {
	hierarchyCount := 0
	for _, result := range results {
		if result.Document != nil && len(result.Document.NetworkFunction) > 0 {
			hierarchyCount++
		}
	}
	return float64(hierarchyCount)/float64(len(results)) > 0.5
}

// createContextSections creates context sections based on the chosen strategy
func (ca *ContextAssembler) createContextSections(results []*EnhancedSearchResult, request *EnhancedSearchRequest, strategy ContextAssemblyStrategy) []*ContextSection {
	switch strategy {
	case StrategyHierarchical:
		return ca.createHierarchicalSections(results, request)
	case StrategyTopical:
		return ca.createTopicalSections(results, request)
	case StrategyProgressive:
		return ca.createProgressiveSections(results, request)
	case StrategyBalanced:
		return ca.createBalancedSections(results, request)
	default:
		return ca.createRankBasedSections(results, request)
	}
}

// createRankBasedSections creates sections in rank order
func (ca *ContextAssembler) createRankBasedSections(results []*EnhancedSearchResult, request *EnhancedSearchRequest) []*ContextSection {
	var sections []*ContextSection

	// Group results into sections to avoid too many small sections
	sectionSize := 2 // Results per section
	if len(results) > 10 {
		sectionSize = 3
	}

	for i := 0; i < len(results); i += sectionSize {
		end := i + sectionSize
		if end > len(results) {
			end = len(results)
		}

		sectionResults := results[i:end]
		section := ca.createSectionFromResults(sectionResults, fmt.Sprintf("Relevant Information %d", (i/sectionSize)+1), "main")
		sections = append(sections, section)
	}

	return sections
}

// createHierarchicalSections creates sections based on document hierarchy
func (ca *ContextAssembler) createHierarchicalSections(results []*EnhancedSearchResult, request *EnhancedSearchRequest) []*ContextSection {
	// Group results by category and subcategory
	categoryGroups := make(map[string][]*EnhancedSearchResult)

	for _, result := range results {
		if result.Document == nil {
			continue
		}

		category := result.Document.Category
		if category == "" {
			category = "General"
		}

		categoryGroups[category] = append(categoryGroups[category], result)
	}

	var sections []*ContextSection

	// Create sections for each category
	categoryOrder := []string{"RAN", "Core", "Transport", "Management", "General"}
	for _, category := range categoryOrder {
		if groupResults, exists := categoryGroups[category]; exists {
			// Sort by relevance within category
			sort.Slice(groupResults, func(i, j int) bool {
				return groupResults[i].CombinedScore > groupResults[j].CombinedScore
			})

			section := ca.createSectionFromResults(groupResults, category, ca.determineSectionType(category, len(sections)))
			sections = append(sections, section)
		}
	}

	// Add any remaining categories
	for category, groupResults := range categoryGroups {
		found := false
		for _, orderedCat := range categoryOrder {
			if category == orderedCat {
				found = true
				break
			}
		}
		if !found {
			section := ca.createSectionFromResults(groupResults, category, "detail")
			sections = append(sections, section)
		}
	}

	return sections
}

// createTopicalSections creates sections grouped by topic
func (ca *ContextAssembler) createTopicalSections(results []*EnhancedSearchResult, request *EnhancedSearchRequest) []*ContextSection {
	// Cluster results by semantic similarity
	clusters := ca.clusterResultsByTopic(results)

	var sections []*ContextSection

	for i, cluster := range clusters {
		// Generate topic title from cluster
		topicTitle := ca.generateTopicTitle(cluster, request)
		sectionType := ca.determineSectionType(topicTitle, i)

		section := ca.createSectionFromResults(cluster, topicTitle, sectionType)
		sections = append(sections, section)
	}

	return sections
}

// createProgressiveSections creates sections that build context progressively
func (ca *ContextAssembler) createProgressiveSections(results []*EnhancedSearchResult, request *EnhancedSearchRequest) []*ContextSection {
	var sections []*ContextSection

	if len(results) == 0 {
		return sections
	}

	// Introduction section - highest relevance overview
	if len(results) > 0 {
		introResults := []*EnhancedSearchResult{results[0]}
		if len(results) > 1 && results[1].CombinedScore > 0.8 {
			introResults = append(introResults, results[1])
		}
		section := ca.createSectionFromResults(introResults, "Introduction", "introduction")
		sections = append(sections, section)
	}

	// Main content sections - group remaining results
	remaining := results[len(sections):]
	if len(remaining) > 0 {
		// Create main sections
		sectionSize := 3
		for i := 0; i < len(remaining); i += sectionSize {
			end := i + sectionSize
			if end > len(remaining) {
				end = len(remaining)
			}

			sectionResults := remaining[i:end]
			title := fmt.Sprintf("Detailed Information %d", (i/sectionSize)+1)
			section := ca.createSectionFromResults(sectionResults, title, "main")
			sections = append(sections, section)
		}
	}

	return sections
}

// createBalancedSections creates balanced sections across sources and types
func (ca *ContextAssembler) createBalancedSections(results []*EnhancedSearchResult, request *EnhancedSearchRequest) []*ContextSection {
	// Group by source first to ensure diversity
	sourceGroups := make(map[string][]*EnhancedSearchResult)

	for _, result := range results {
		source := "Unknown"
		if result.Document != nil && result.Document.Source != "" {
			source = result.Document.Source
		}
		sourceGroups[source] = append(sourceGroups[source], result)
	}

	var sections []*ContextSection

	// Create balanced sections by interleaving sources
	sourceOrder := []string{"3GPP", "O-RAN", "ETSI", "ITU", "Unknown"}
	maxSourceResults := 2 // Maximum results per source per section

	sectionCount := 0
	for {
		var sectionResults []*EnhancedSearchResult
		hasMore := false

		// Take results from each source in order
		for _, source := range sourceOrder {
			if groupResults, exists := sourceGroups[source]; exists && len(groupResults) > 0 {
				// Take up to maxSourceResults from this source
				takeCount := min(maxSourceResults, len(groupResults))
				sectionResults = append(sectionResults, groupResults[:takeCount]...)
				sourceGroups[source] = groupResults[takeCount:]

				if len(sourceGroups[source]) > 0 {
					hasMore = true
				}
			}
		}

		if len(sectionResults) == 0 {
			break
		}

		// Sort section results by combined score
		sort.Slice(sectionResults, func(i, j int) bool {
			return sectionResults[i].CombinedScore > sectionResults[j].CombinedScore
		})

		title := fmt.Sprintf("Balanced Information %d", sectionCount+1)
		sectionType := ca.determineSectionType(title, sectionCount)
		section := ca.createSectionFromResults(sectionResults, title, sectionType)
		sections = append(sections, section)

		sectionCount++
		if !hasMore || sectionCount >= 5 { // Limit number of sections
			break
		}
	}

	return sections
}

// clusterResultsByTopic clusters results by semantic topic similarity
func (ca *ContextAssembler) clusterResultsByTopic(results []*EnhancedSearchResult) [][]*EnhancedSearchResult {
	if len(results) <= 2 {
		return [][]*EnhancedSearchResult{results}
	}

	// Simple clustering based on keyword similarity
	var clusters [][]*EnhancedSearchResult
	used := make(map[int]bool)

	for i, result := range results {
		if used[i] {
			continue
		}

		cluster := []*EnhancedSearchResult{result}
		used[i] = true

		// Find similar results
		for j := i + 1; j < len(results); j++ {
			if used[j] {
				continue
			}

			if ca.areTopicallySimilar(result, results[j]) {
				cluster = append(cluster, results[j])
				used[j] = true
			}
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}

// areTopicallySimilar checks if two results are topically similar
func (ca *ContextAssembler) areTopicallySimilar(result1, result2 *EnhancedSearchResult) bool {
	if result1.Document == nil || result2.Document == nil {
		return false
	}

	doc1, doc2 := result1.Document, result2.Document

	// Check category similarity
	if doc1.Category != "" && doc2.Category != "" && doc1.Category == doc2.Category {
		return true
	}

	// Check technology overlap
	if len(doc1.Technology) > 0 && len(doc2.Technology) > 0 {
		for _, tech1 := range doc1.Technology {
			for _, tech2 := range doc2.Technology {
				if tech1 == tech2 {
					return true
				}
			}
		}
	}

	// Check network function overlap
	if len(doc1.NetworkFunction) > 0 && len(doc2.NetworkFunction) > 0 {
		for _, nf1 := range doc1.NetworkFunction {
			for _, nf2 := range doc2.NetworkFunction {
				if nf1 == nf2 {
					return true
				}
			}
		}
	}

	// Check keyword similarity
	if len(doc1.Keywords) > 0 && len(doc2.Keywords) > 0 {
		commonKeywords := 0
		for _, kw1 := range doc1.Keywords {
			for _, kw2 := range doc2.Keywords {
				if strings.EqualFold(kw1, kw2) {
					commonKeywords++
				}
			}
		}
		// Similar if they share more than 30% of keywords
		totalKeywords := len(doc1.Keywords) + len(doc2.Keywords)
		if float64(commonKeywords*2)/float64(totalKeywords) > 0.3 {
			return true
		}
	}

	return false
}

// generateTopicTitle generates a title for a topic cluster
func (ca *ContextAssembler) generateTopicTitle(cluster []*EnhancedSearchResult, request *EnhancedSearchRequest) string {
	if len(cluster) == 0 {
		return "Information"
	}

	// Find common themes
	categories := make(map[string]int)
	technologies := make(map[string]int)
	networkFunctions := make(map[string]int)

	for _, result := range cluster {
		if result.Document == nil {
			continue
		}

		doc := result.Document
		if doc.Category != "" {
			categories[doc.Category]++
		}

		for _, tech := range doc.Technology {
			technologies[tech]++
		}

		for _, nf := range doc.NetworkFunction {
			networkFunctions[nf]++
		}
	}

	// Find the most common theme
	if mostCommon := findMostCommon(categories); mostCommon != "" {
		return mostCommon
	}

	if mostCommon := findMostCommon(technologies); mostCommon != "" {
		return mostCommon + " Technology"
	}

	if mostCommon := findMostCommon(networkFunctions); mostCommon != "" {
		return mostCommon + " Function"
	}

	return "Related Information"
}

// findMostCommon finds the most common item in a frequency map
func findMostCommon(freq map[string]int) string {
	maxCount := 0
	mostCommon := ""

	for item, count := range freq {
		if count > maxCount {
			maxCount = count
			mostCommon = item
		}
	}

	return mostCommon
}

// createSectionFromResults creates a context section from results
func (ca *ContextAssembler) createSectionFromResults(results []*EnhancedSearchResult, title, sectionType string) *ContextSection {
	if len(results) == 0 {
		return &ContextSection{
			Title:       title,
			Content:     "",
			SectionType: sectionType,
			Results:     results,
			Metadata:    make(map[string]interface{}),
		}
	}

	var contentBuilder strings.Builder
	var sources []string
	totalRelevance := float32(0.0)

	// Add section header if configured
	if ca.config.IncludeHierarchyInfo && title != "" {
		contentBuilder.WriteString(fmt.Sprintf("=== %s ===\n\n", title))
	}

	for i, result := range results {
		if result.Document == nil {
			continue
		}

		doc := result.Document
		totalRelevance += result.CombinedScore

		// Add document header with metadata
		if ca.config.IncludeSourceMetadata {
			var headerParts []string
			if doc.Source != "" && doc.Source != "Unknown" {
				headerParts = append(headerParts, fmt.Sprintf("Source: %s", doc.Source))
			}
			if doc.Title != "" {
				headerParts = append(headerParts, fmt.Sprintf("Title: %s", doc.Title))
			}
			if doc.Version != "" {
				headerParts = append(headerParts, fmt.Sprintf("Version: %s", doc.Version))
			}

			if len(headerParts) > 0 {
				contentBuilder.WriteString(fmt.Sprintf("[%s]\n", strings.Join(headerParts, " | ")))
			}
		}

		// Add the content
		content := doc.Content
		if len(content) > 2000 { // Limit individual document content
			content = content[:2000] + "..."
		}

		contentBuilder.WriteString(content)

		// Add spacing between documents
		if i < len(results)-1 {
			contentBuilder.WriteString("\n\n---\n\n")
		}

		// Track sources
		if doc.Source != "" && doc.Source != "Unknown" {
			sources = append(sources, doc.Source)
		}
	}

	// Calculate average relevance
	avgRelevance := float32(0.0)
	if len(results) > 0 {
		avgRelevance = totalRelevance / float32(len(results))
	}

	// Create metadata
	metadata := map[string]interface{}{
		"result_count":      len(results),
		"average_relevance": avgRelevance,
		"sources":          ca.uniqueStrings(sources),
		"section_type":     sectionType,
	}

	section := &ContextSection{
		Title:       title,
		Content:     contentBuilder.String(),
		Source:      strings.Join(ca.uniqueStrings(sources), ", "),
		Relevance:   avgRelevance,
		Results:     results,
		Metadata:    metadata,
		SectionType: sectionType,
	}

	return section
}

// determineSectionType determines the type of section based on position and content
func (ca *ContextAssembler) determineSectionType(title string, position int) string {
	titleLower := strings.ToLower(title)

	if strings.Contains(titleLower, "introduction") || strings.Contains(titleLower, "overview") {
		return "introduction"
	}

	if strings.Contains(titleLower, "conclusion") || strings.Contains(titleLower, "summary") {
		return "conclusion"
	}

	if position == 0 {
		return "introduction"
	}

	if strings.Contains(titleLower, "detailed") || strings.Contains(titleLower, "specific") {
		return "detail"
	}

	return "main"
}

// assembleContextFromSections assembles the final context from sections
func (ca *ContextAssembler) assembleContextFromSections(sections []*ContextSection, request *EnhancedSearchRequest) string {
	if len(sections) == 0 {
		return ""
	}

	var contextBuilder strings.Builder
	currentLength := 0
	maxLength := ca.config.MaxContextLength

	// Sort sections by type and relevance
	ca.sortSectionsByPriority(sections)

	for i, section := range sections {
		sectionContent := section.Content
		
		// Check if adding this section would exceed the limit
		if currentLength+len(sectionContent) > maxLength {
			// Try to include partial content
			remainingSpace := maxLength - currentLength
			if remainingSpace > 200 { // Only include if we have meaningful space
				truncated := ca.truncateAtSentenceBoundary(sectionContent, remainingSpace)
				if truncated != "" {
					contextBuilder.WriteString(truncated)
					contextBuilder.WriteString("\n\n[Content truncated...]\n")
				}
			}
			break
		}

		contextBuilder.WriteString(sectionContent)
		currentLength += len(sectionContent)

		// Add section separator (except for the last section)
		if i < len(sections)-1 {
			separator := "\n\n"
			contextBuilder.WriteString(separator)
			currentLength += len(separator)
		}
	}

	return contextBuilder.String()
}

// sortSectionsByPriority sorts sections by type priority and relevance
func (ca *ContextAssembler) sortSectionsByPriority(sections []*ContextSection) {
	typePriority := map[string]int{
		"introduction": 0,
		"main":        1,
		"detail":      2,
		"conclusion":  3,
	}

	sort.Slice(sections, func(i, j int) bool {
		// First sort by type priority
		priI := typePriority[sections[i].SectionType]
		priJ := typePriority[sections[j].SectionType]
		
		if priI != priJ {
			return priI < priJ
		}

		// Then by relevance (descending)
		return sections[i].Relevance > sections[j].Relevance
	})
}

// truncateAtSentenceBoundary truncates content at a sentence boundary
func (ca *ContextAssembler) truncateAtSentenceBoundary(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}

	// Find the last sentence boundary before maxLength
	truncated := content[:maxLength]
	
	// Look for sentence endings
	lastDot := strings.LastIndex(truncated, ".")
	lastExclamation := strings.LastIndex(truncated, "!")
	lastQuestion := strings.LastIndex(truncated, "?")
	
	lastSentenceEnd := max(lastDot, lastExclamation, lastQuestion)
	
	if lastSentenceEnd > maxLength/2 { // Only truncate if we keep at least half
		return content[:lastSentenceEnd+1]
	}

	// Fallback: truncate at word boundary
	words := strings.Fields(truncated)
	if len(words) > 1 {
		words = words[:len(words)-1] // Remove last potentially incomplete word
		return strings.Join(words, " ")
	}

	return truncated
}

// createContextMetadata creates metadata about the assembled context
func (ca *ContextAssembler) createContextMetadata(sections []*ContextSection, results []*EnhancedSearchResult, context string) *ContextMetadata {
	metadata := &ContextMetadata{
		TotalLength:        len(context),
		DocumentCount:      len(results),
		SourceDistribution: make(map[string]int),
		HierarchyLevels:    []int{},
	}

	// Count sources
	for _, result := range results {
		if result.Document != nil && result.Document.Source != "" {
			metadata.SourceDistribution[result.Document.Source]++
		}
	}

	// Calculate average quality
	totalQuality := float32(0.0)
	qualityCount := 0
	technicalTermCount := 0

	for _, result := range results {
		if result.QualityScore > 0 {
			totalQuality += result.QualityScore
			qualityCount++
		}
		if result.Document != nil {
			technicalTermCount += len(result.Document.Keywords)
		}
	}

	if qualityCount > 0 {
		metadata.AverageQuality = totalQuality / float32(qualityCount)
	}

	metadata.TechnicalTermCount = technicalTermCount

	// Check if content was truncated
	requestedLength := 0
	for _, section := range sections {
		requestedLength += len(section.Content)
	}
	
	if len(context) < requestedLength {
		metadata.TruncatedAt = len(context)
	}

	return metadata
}

// uniqueStrings removes duplicate strings from a slice
func (ca *ContextAssembler) uniqueStrings(strings []string) []string {
	keys := make(map[string]bool)
	var unique []string

	for _, str := range strings {
		if !keys[str] {
			keys[str] = true
			unique = append(unique, str)
		}
	}

	return unique
}

// Helper functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of three integers
func max(a, b, c int) int {
	if a > b {
		if a > c {
			return a
		}
		return c
	}
	if b > c {
		return b
	}
	return c
}