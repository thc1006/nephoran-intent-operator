package rag

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
)

// QueryEnhancer provides query enhancement capabilities for telecom domain queries
type QueryEnhancer struct {
	config            *RetrievalConfig
	logger            *slog.Logger
	telecomDictionary *TelecomDictionary
	synonymExpander   *SynonymExpander
	spellChecker      *SpellChecker
	mutex             sync.RWMutex
}

// TelecomDictionary contains telecom-specific terminology and expansions
type TelecomDictionary struct {
	Acronyms          map[string][]string            `json:"acronyms"`           // AMF -> ["Access and Mobility Management Function"]
	Synonyms          map[string][]string            `json:"synonyms"`           // "base station" -> ["gNB", "eNB", "node"]
	TechnicalTerms    map[string]TechnicalTermInfo   `json:"technical_terms"`    // Detailed term information
	DomainMapping     map[string][]string            `json:"domain_mapping"`     // "RAN" -> related terms
	IntentPatterns    map[string][]string            `json:"intent_patterns"`    // Common query patterns
	ContextualTerms   map[string]map[string][]string `json:"contextual_terms"`   // Context-specific expansions
	mutex             sync.RWMutex
}

// TechnicalTermInfo contains detailed information about technical terms
type TechnicalTermInfo struct {
	FullForm        string   `json:"full_form"`
	Abbreviations   []string `json:"abbreviations"`
	RelatedTerms    []string `json:"related_terms"`
	Domain          string   `json:"domain"`
	Definition      string   `json:"definition"`
	UsageExamples   []string `json:"usage_examples"`
	Weight          float64  `json:"weight"`        // Importance weight for boosting
	Popularity      float64  `json:"popularity"`    // How often used in domain
}

// SynonymExpander handles synonym expansion for queries
type SynonymExpander struct {
	synonymSets       [][]string           // Groups of synonymous terms
	contextualSynonyms map[string][]string  // Context-dependent synonyms
	telecomSynonyms   map[string][]string  // Telecom-specific synonyms
	mutex             sync.RWMutex
}

// SpellChecker provides spell checking and correction for telecom terms
type SpellChecker struct {
	dictionary     map[string]bool      // Valid telecom terms
	corrections    map[string]string    // Common misspellings -> corrections
	soundexMap     map[string][]string  // Phonetic similarity mapping
	editDistance   int                  // Maximum edit distance for suggestions
	mutex          sync.RWMutex
}

// NewQueryEnhancer creates a new query enhancer
func NewQueryEnhancer(config *RetrievalConfig) *QueryEnhancer {
	enhancer := &QueryEnhancer{
		config: config,
		logger: slog.Default().With("component", "query-enhancer"),
	}

	// Initialize components
	enhancer.telecomDictionary = NewTelecomDictionary()
	enhancer.synonymExpander = NewSynonymExpander()
	enhancer.spellChecker = NewSpellChecker()

	return enhancer
}

// EnhanceQuery enhances a query using various techniques
func (qe *QueryEnhancer) EnhanceQuery(ctx context.Context, request *EnhancedSearchRequest) (string, *QueryEnhancements, error) {
	originalQuery := request.Query
	currentQuery := originalQuery

	qe.logger.Debug("Starting query enhancement",
		"original_query", originalQuery,
		"intent_type", request.IntentType,
	)

	enhancements := &QueryEnhancements{
		OriginalQuery:       originalQuery,
		ExpandedTerms:       []string{},
		SynonymReplacements: make(map[string]string),
		SpellingCorrections: make(map[string]string),
		EnhancementApplied:  []string{},
	}

	// Step 1: Spell correction
	if qe.config.EnableSpellCorrection {
		corrected, corrections := qe.spellChecker.CorrectQuery(currentQuery)
		if len(corrections) > 0 {
			currentQuery = corrected
			enhancements.SpellingCorrections = corrections
			enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "spell_correction")
			qe.logger.Debug("Applied spell corrections", "corrections", corrections)
		}
	}

	// Step 2: Telecom-specific term expansion
	if qe.config.EnableQueryExpansion {
		expanded, expandedTerms := qe.expandTelecomTerms(currentQuery, request.IntentType)
		if len(expandedTerms) > 0 {
			currentQuery = expanded
			enhancements.ExpandedTerms = expandedTerms
			enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "term_expansion")
			qe.logger.Debug("Applied term expansion", "expanded_terms", expandedTerms)
		}
	}

	// Step 3: Synonym expansion
	if qe.config.EnableSynonymExpansion {
		synonymized, synonyms := qe.synonymExpander.ExpandSynonyms(currentQuery, request.IntentType)
		if len(synonyms) > 0 {
			currentQuery = synonymized
			enhancements.SynonymReplacements = synonyms
			enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "synonym_expansion")
			qe.logger.Debug("Applied synonym expansion", "synonyms", synonyms)
		}
	}

	// Step 4: Query rewriting for intent optimization
	if qe.config.EnableQueryRewriting {
		rewritten := qe.rewriteForIntent(currentQuery, request)
		if rewritten != currentQuery {
			currentQuery = rewritten
			enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "intent_rewriting")
			qe.logger.Debug("Applied intent-based rewriting")
		}
	}

	// Step 5: Context-aware enhancement
	if len(request.ConversationHistory) > 0 {
		contextEnhanced := qe.enhanceWithContext(currentQuery, request.ConversationHistory)
		if contextEnhanced != currentQuery {
			currentQuery = contextEnhanced
			enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "context_enhancement")
			qe.logger.Debug("Applied context enhancement")
		}
	}

	enhancements.RewrittenQuery = currentQuery

	qe.logger.Info("Query enhancement completed",
		"original", originalQuery,
		"enhanced", currentQuery,
		"enhancements_applied", enhancements.EnhancementApplied,
	)

	return currentQuery, enhancements, nil
}

// expandTelecomTerms expands technical terms with their full forms and related terms
func (qe *QueryEnhancer) expandTelecomTerms(query, intentType string) (string, []string) {
	var expandedTerms []string
	expandedQuery := query

	// Find acronyms and expand them
	acronymPattern := regexp.MustCompile(`\b[A-Z]{2,}\b`)
	acronyms := acronymPattern.FindAllString(query, -1)

	for _, acronym := range acronyms {
		if expansions, exists := qe.telecomDictionary.getAcronymExpansions(acronym); exists {
			// Add the full form as additional search terms
			for _, expansion := range expansions {
				if !strings.Contains(expandedQuery, expansion) {
					expandedQuery += " " + expansion
					expandedTerms = append(expandedTerms, expansion)
				}
			}
		}
	}

	// Add domain-specific related terms
	if intentType != "" {
		if relatedTerms, exists := qe.telecomDictionary.getDomainTerms(intentType); exists {
			for _, term := range relatedTerms {
				if strings.Contains(strings.ToLower(query), strings.ToLower(term)) {
					continue // Already present
				}
				
				// Add related terms with lower weight
				expandedQuery += " " + term
				expandedTerms = append(expandedTerms, term)
				
				// Limit expansion to avoid over-expansion
				if len(expandedTerms) >= qe.config.QueryExpansionTerms {
					break
				}
			}
		}
	}

	return expandedQuery, expandedTerms
}

// rewriteForIntent rewrites the query to be more effective for specific intent types
func (qe *QueryEnhancer) rewriteForIntent(query string, request *EnhancedSearchRequest) string {
	intentType := request.IntentType
	if intentType == "" {
		return query
	}

	lowerQuery := strings.ToLower(query)
	
	switch intentType {
	case "configuration":
		// Add configuration-specific terms
		if !strings.Contains(lowerQuery, "config") && !strings.Contains(lowerQuery, "parameter") {
			query += " configuration parameter setting"
		}
		if !strings.Contains(lowerQuery, "how to") && !strings.Contains(lowerQuery, "procedure") {
			query += " procedure setup"
		}

	case "troubleshooting":
		// Add troubleshooting-specific terms
		if !strings.Contains(lowerQuery, "problem") && !strings.Contains(lowerQuery, "issue") && !strings.Contains(lowerQuery, "error") {
			query += " problem issue troubleshoot"
		}
		if !strings.Contains(lowerQuery, "solution") && !strings.Contains(lowerQuery, "fix") {
			query += " solution fix resolve"
		}

	case "optimization":
		// Add optimization-specific terms
		if !strings.Contains(lowerQuery, "optimize") && !strings.Contains(lowerQuery, "performance") {
			query += " optimize performance improve"
		}
		if !strings.Contains(lowerQuery, "tuning") && !strings.Contains(lowerQuery, "enhancement") {
			query += " tuning enhancement"
		}

	case "monitoring":
		// Add monitoring-specific terms
		if !strings.Contains(lowerQuery, "monitor") && !strings.Contains(lowerQuery, "metric") {
			query += " monitoring metrics measurement"
		}
		if !strings.Contains(lowerQuery, "kpi") && !strings.Contains(lowerQuery, "alarm") {
			query += " KPI alarm notification"
		}
	}

	// Add network domain context if specified
	if request.NetworkDomain != "" {
		domainLower := strings.ToLower(request.NetworkDomain)
		if !strings.Contains(lowerQuery, domainLower) {
			query += " " + request.NetworkDomain
		}
	}

	return query
}

// enhanceWithContext enhances query using conversation history
func (qe *QueryEnhancer) enhanceWithContext(query string, history []string) string {
	if len(history) == 0 {
		return query
	}

	// Extract important terms from recent conversation
	contextTerms := qe.extractContextTerms(history)
	
	// Add relevant context terms that aren't already in the query
	enhanced := query
	lowerQuery := strings.ToLower(query)
	
	for _, term := range contextTerms {
		lowerTerm := strings.ToLower(term)
		if !strings.Contains(lowerQuery, lowerTerm) && len(enhanced.split(" ")) < 20 { // Limit total query length
			enhanced += " " + term
		}
	}

	return enhanced
}

// extractContextTerms extracts important technical terms from conversation history
func (qe *QueryEnhancer) extractContextTerms(history []string) []string {
	var contextTerms []string
	termFreq := make(map[string]int)

	// Combine recent history (last 3 messages)
	recentHistory := history
	if len(history) > 3 {
		recentHistory = history[len(history)-3:]
	}

	text := strings.Join(recentHistory, " ")
	
	// Extract technical terms using patterns
	patterns := []string{
		`\b[A-Z]{2,}\b`,                    // Acronyms
		`\b\d+G\b`,                        // Technology generations
		`\b(?:gNB|eNB|AMF|SMF|UPF)\b`,     // Network functions
		`\b\d+\.\d+\.\d+\b`,               // Specification numbers
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllString(text, -1)
		for _, match := range matches {
			termFreq[match]++
		}
	}

	// Sort by frequency and take top terms
	type termCount struct {
		term  string
		count int
	}
	
	var terms []termCount
	for term, count := range termFreq {
		if count >= 2 { // Must appear at least twice
			terms = append(terms, termCount{term, count})
		}
	}

	// Sort by frequency
	for i := 0; i < len(terms)-1; i++ {
		for j := i + 1; j < len(terms); j++ {
			if terms[i].count < terms[j].count {
				terms[i], terms[j] = terms[j], terms[i]
			}
		}
	}

	// Return top terms
	maxTerms := 3
	for i, tc := range terms {
		if i >= maxTerms {
			break
		}
		contextTerms = append(contextTerms, tc.term)
	}

	return contextTerms
}

// TelecomDictionary implementation

// NewTelecomDictionary creates a new telecom dictionary
func NewTelecomDictionary() *TelecomDictionary {
	td := &TelecomDictionary{
		Acronyms:        make(map[string][]string),
		Synonyms:        make(map[string][]string),
		TechnicalTerms:  make(map[string]TechnicalTermInfo),
		DomainMapping:   make(map[string][]string),
		IntentPatterns:  make(map[string][]string),
		ContextualTerms: make(map[string]map[string][]string),
	}

	// Initialize with common telecom terms
	td.initializeAcronyms()
	td.initializeSynonyms()
	td.initializeTechnicalTerms()
	td.initializeDomainMappings()
	td.initializeIntentPatterns()

	return td
}

// initializeAcronyms sets up common telecom acronyms
func (td *TelecomDictionary) initializeAcronyms() {
	td.Acronyms = map[string][]string{
		"5G":    {"Fifth Generation", "5th Generation"},
		"4G":    {"Fourth Generation", "LTE"},
		"AMF":   {"Access and Mobility Management Function"},
		"SMF":   {"Session Management Function"},
		"UPF":   {"User Plane Function"},
		"PCF":   {"Policy Control Function"},
		"UDM":   {"Unified Data Management"},
		"UDR":   {"Unified Data Repository"},
		"AUSF":  {"Authentication Server Function"},
		"NRF":   {"Network Repository Function"},
		"NSSF":  {"Network Slice Selection Function"},
		"NEF":   {"Network Exposure Function"},
		"gNB":   {"Next Generation NodeB", "5G Base Station"},
		"eNB":   {"Evolved NodeB", "4G Base Station"},
		"DU":    {"Distributed Unit"},
		"CU":    {"Centralized Unit", "Central Unit"},
		"RU":    {"Radio Unit"},
		"RAN":   {"Radio Access Network"},
		"EPC":   {"Evolved Packet Core"},
		"5GC":   {"5G Core Network"},
		"QoS":   {"Quality of Service"},
		"SLA":   {"Service Level Agreement"},
		"KPI":   {"Key Performance Indicator"},
		"OAM":   {"Operations, Administration and Maintenance"},
		"SON":   {"Self-Organizing Network"},
		"NFV":   {"Network Function Virtualization"},
		"SDN":   {"Software Defined Network"},
		"MANO":  {"Management and Orchestration"},
		"VNF":   {"Virtual Network Function"},
		"CNF":   {"Cloud Native Function"},
		"URLLC": {"Ultra-Reliable Low Latency Communication"},
		"eMBB":  {"Enhanced Mobile Broadband"},
		"mMTC":  {"Massive Machine Type Communication"},
		"IoT":   {"Internet of Things"},
		"V2X":   {"Vehicle to Everything"},
		"AR":    {"Augmented Reality"},
		"VR":    {"Virtual Reality"},
		"AI":    {"Artificial Intelligence"},
		"ML":    {"Machine Learning"},
	}
}

// initializeSynonyms sets up synonym mappings
func (td *TelecomDictionary) initializeSynonyms() {
	td.Synonyms = map[string][]string{
		"base station":     {"node", "cell", "site", "tower"},
		"mobile device":    {"UE", "user equipment", "handset", "phone"},
		"network":          {"system", "infrastructure", "platform"},
		"configuration":    {"config", "setup", "settings", "parameters"},
		"optimization":     {"tuning", "improvement", "enhancement"},
		"monitoring":       {"surveillance", "tracking", "observation"},
		"performance":      {"throughput", "capacity", "efficiency"},
		"latency":          {"delay", "response time", "lag"},
		"bandwidth":        {"capacity", "data rate", "speed"},
		"coverage":         {"signal strength", "reach", "footprint"},
		"interference":     {"noise", "disturbance", "crosstalk"},
		"handover":         {"handoff", "mobility", "transition"},
		"authentication":   {"auth", "verification", "validation"},
		"encryption":       {"security", "protection", "cipher"},
		"protocol":         {"standard", "specification", "procedure"},
		"interface":        {"connection", "link", "API"},
		"deployment":       {"installation", "setup", "rollout"},
		"maintenance":      {"upkeep", "service", "support"},
		"troubleshooting":  {"debugging", "diagnosis", "problem solving"},
		"fault":            {"error", "failure", "issue", "problem"},
	}
}

// initializeTechnicalTerms sets up detailed technical term information
func (td *TelecomDictionary) initializeTechnicalTerms() {
	td.TechnicalTerms = map[string]TechnicalTermInfo{
		"AMF": {
			FullForm:        "Access and Mobility Management Function",
			Abbreviations:   []string{"AMF"},
			RelatedTerms:    []string{"5GC", "authentication", "mobility", "registration"},
			Domain:          "Core",
			Definition:      "5G Core network function responsible for access and mobility management",
			UsageExamples:   []string{"AMF handles UE registration", "AMF manages mobility procedures"},
			Weight:          1.2,
			Popularity:      0.9,
		},
		"gNB": {
			FullForm:        "Next Generation NodeB",
			Abbreviations:   []string{"gNB", "gNodeB"},
			RelatedTerms:    []string{"5G", "NR", "base station", "RAN"},
			Domain:          "RAN",
			Definition:      "5G base station that provides radio access",
			UsageExamples:   []string{"gNB connects UEs to 5G network", "gNB handles radio resources"},
			Weight:          1.3,
			Popularity:      0.95,
		},
		"URLLC": {
			FullForm:        "Ultra-Reliable Low Latency Communication",
			Abbreviations:   []string{"URLLC"},
			RelatedTerms:    []string{"5G", "latency", "reliability", "use case"},
			Domain:          "Service",
			Definition:      "5G use case requiring ultra-low latency and high reliability",
			UsageExamples:   []string{"URLLC for industrial automation", "URLLC latency requirements"},
			Weight:          1.1,
			Popularity:      0.7,
		},
	}
}

// initializeDomainMappings sets up domain-specific term mappings  
func (td *TelecomDictionary) initializeDomainMappings() {
	td.DomainMapping = map[string][]string{
		"RAN": {
			"radio", "antenna", "RF", "spectrum", "coverage", "interference",
			"handover", "beamforming", "MIMO", "scheduling", "resource allocation",
			"gNB", "eNB", "base station", "cell", "sector",
		},
		"Core": {
			"routing", "switching", "authentication", "authorization", "policy",
			"session management", "user plane", "control plane", "signaling",
			"AMF", "SMF", "UPF", "PCF", "UDM", "AUSF",
		},
		"Transport": {
			"backhaul", "fronthaul", "xhaul", "fiber", "ethernet", "IP",
			"MPLS", "VPN", "QoS", "bandwidth", "latency", "jitter",
		},
		"Management": {
			"orchestration", "automation", "configuration", "monitoring",
			"performance", "fault", "security", "analytics", "OAM",
			"SON", "AI", "ML", "optimization",
		},
		"configuration": {
			"parameter", "setting", "value", "option", "policy", "rule",
			"template", "profile", "configuration management", "provisioning",
		},
		"troubleshooting": {
			"problem", "issue", "error", "fault", "failure", "diagnosis",
			"debug", "trace", "log", "alarm", "event", "symptom",
		},
		"optimization": {
			"performance", "efficiency", "tuning", "improvement", "enhancement",
			"throughput", "capacity", "utilization", "resource allocation",
		},
		"monitoring": {
			"metrics", "KPI", "measurement", "statistics", "analytics",
			"dashboard", "alert", "notification", "threshold", "trending",
		},
	}
}

// initializeIntentPatterns sets up common query patterns for different intents
func (td *TelecomDictionary) initializeIntentPatterns() {
	td.IntentPatterns = map[string][]string{
		"configuration": {
			"how to configure", "how to set", "configuration of", "setting up",
			"parameters for", "configuring", "setup procedure", "configuration guide",
		},
		"troubleshooting": {
			"problem with", "issue with", "error in", "not working", "failing",
			"troubleshoot", "debug", "fix", "resolve", "solve problem",
		},
		"optimization": {
			"optimize", "improve performance", "enhance", "tuning", "better",
			"increase throughput", "reduce latency", "maximize", "efficiency",
		},
		"monitoring": {
			"monitor", "measure", "track", "observe", "metrics for",
			"KPI", "performance indicators", "statistics", "analytics",
		},
	}
}

// getAcronymExpansions returns expansions for an acronym
func (td *TelecomDictionary) getAcronymExpansions(acronym string) ([]string, bool) {
	td.mutex.RLock()
	defer td.mutex.RUnlock()
	
	expansions, exists := td.Acronyms[strings.ToUpper(acronym)]
	return expansions, exists
}

// getDomainTerms returns terms related to a domain
func (td *TelecomDictionary) getDomainTerms(domain string) ([]string, bool) {
	td.mutex.RLock()
	defer td.mutex.RUnlock()
	
	terms, exists := td.DomainMapping[strings.ToLower(domain)]
	return terms, exists
}

// SynonymExpander implementation

// NewSynonymExpander creates a new synonym expander
func NewSynonymExpander() *SynonymExpander {
	se := &SynonymExpander{
		synonymSets:       [][]string{},
		contextualSynonyms: make(map[string][]string),
		telecomSynonyms:   make(map[string][]string),
	}

	se.initializeSynonymSets()
	se.initializeTelecomSynonyms()

	return se
}

// initializeSynonymSets sets up synonym groups
func (se *SynonymExpander) initializeSynonymSets() {
	se.synonymSets = [][]string{
		{"base station", "gNB", "eNB", "node", "cell site"},
		{"mobile device", "UE", "user equipment", "handset", "smartphone"},
		{"network", "system", "infrastructure", "platform"},
		{"latency", "delay", "response time", "lag"},
		{"bandwidth", "capacity", "data rate", "throughput"},
		{"coverage", "signal strength", "reach", "footprint"},
		{"configuration", "config", "setup", "settings"},
		{"optimization", "tuning", "improvement", "enhancement"},
		{"monitoring", "surveillance", "tracking", "observation"},
		{"authentication", "auth", "verification", "validation"},
		{"performance", "efficiency", "throughput", "capacity"},
	}
}

// initializeTelecomSynonyms sets up telecom-specific synonyms
func (se *SynonymExpander) initializeTelecomSynonyms() {
	se.telecomSynonyms = map[string][]string{
		"5G": {"NR", "New Radio", "fifth generation"},
		"4G": {"LTE", "LTE-A", "fourth generation"},
		"handover": {"handoff", "mobility", "cell change"},
		"QoS": {"quality of service", "service quality"},
		"beamforming": {"beam steering", "antenna beamforming"},
		"MIMO": {"multiple input multiple output", "spatial multiplexing"},
		"interference": {"noise", "crosstalk", "signal degradation"},
		"spectrum": {"frequency", "radio frequency", "RF"},
	}
}

// ExpandSynonyms expands query with synonyms
func (se *SynonymExpander) ExpandSynonyms(query, intentType string) (string, map[string]string) {
	se.mutex.RLock()
	defer se.mutex.RUnlock()

	synonymReplacements := make(map[string]string)
	expandedQuery := query
	queryLower := strings.ToLower(query)

	// Check telecom-specific synonyms first
	for original, synonyms := range se.telecomSynonyms {
		originalLower := strings.ToLower(original)
		if strings.Contains(queryLower, originalLower) {
			// Add the first synonym as expansion
			if len(synonyms) > 0 {
				synonym := synonyms[0]
				if !strings.Contains(queryLower, strings.ToLower(synonym)) {
					expandedQuery += " " + synonym
					synonymReplacements[original] = synonym
				}
			}
		}
	}

	// Check synonym sets
	for _, synonymSet := range se.synonymSets {
		for _, term := range synonymSet {
			termLower := strings.ToLower(term)
			if strings.Contains(queryLower, termLower) {
				// Add other terms from the same set
				for _, synonym := range synonymSet {
					synonymLower := strings.ToLower(synonym)
					if synonym != term && !strings.Contains(queryLower, synonymLower) {
						expandedQuery += " " + synonym
						synonymReplacements[term] = synonym
						break // Add only one synonym to avoid over-expansion
					}
				}
				break
			}
		}
	}

	return expandedQuery, synonymReplacements
}

// SpellChecker implementation

// NewSpellChecker creates a new spell checker
func NewSpellChecker() *SpellChecker {
	sc := &SpellChecker{
		dictionary:   make(map[string]bool),
		corrections:  make(map[string]string),
		soundexMap:   make(map[string][]string),
		editDistance: 2,
	}

	sc.initializeDictionary()
	sc.initializeCorrections()

	return sc
}

// initializeDictionary sets up the telecom term dictionary
func (sc *SpellChecker) initializeDictionary() {
	// Common telecom terms that should be in the dictionary
	terms := []string{
		// Acronyms
		"5G", "4G", "3G", "2G", "AMF", "SMF", "UPF", "PCF", "UDM", "UDR",
		"AUSF", "NRF", "NSSF", "NEF", "gNB", "eNB", "DU", "CU", "RU",
		"RAN", "EPC", "5GC", "QoS", "SLA", "KPI", "OAM", "SON",
		"NFV", "SDN", "MANO", "VNF", "CNF", "URLLC", "eMBB", "mMTC",
		
		// Full terms
		"beamforming", "handover", "throughput", "latency", "bandwidth",
		"coverage", "interference", "spectrum", "authentication",
		"configuration", "optimization", "monitoring", "troubleshooting",
		"performance", "deployment", "maintenance", "orchestration",
		
		// Technology terms
		"MIMO", "OFDM", "LTE", "NR", "WiFi", "Bluetooth", "Ethernet",
		"fiber", "backhaul", "fronthaul", "xhaul", "virtualization",
	}

	for _, term := range terms {
		sc.dictionary[strings.ToLower(term)] = true
	}
}

// initializeCorrections sets up common misspellings and their corrections
func (sc *SpellChecker) initializeCorrections() {
	sc.corrections = map[string]string{
		// Common misspellings of telecom terms
		"5g":           "5G",
		"4g":           "4G",
		"gnb":          "gNB",
		"enb":          "eNB",
		"amf":          "AMF",
		"smf":          "SMF",
		"upf":          "UPF",
		"pcf":          "PCF",
		"udm":          "UDM",
		"udr":          "UDR",
		"ausf":         "AUSF",
		"nrf":          "NRF",
		"nssf":         "NSSF",
		"nef":          "NEF",
		"ran":          "RAN",
		"epc":          "EPC",
		"qos":          "QoS",
		"sla":          "SLA",
		"kpi":          "KPI",
		"oam":          "OAM",
		"son":          "SON",
		"nfv":          "NFV",
		"sdn":          "SDN",
		"mano":         "MANO",
		"vnf":          "VNF",
		"cnf":          "CNF",
		"urllc":        "URLLC",
		"embb":         "eMBB",
		"mmtc":         "mMTC",
		
		// Common typos
		"confguration": "configuration",
		"optimizaton":  "optimization",
		"performace":   "performance",
		"throuhgput":   "throughput",
		"lateny":       "latency",
		"bandwith":     "bandwidth",
		"interferance": "interference",
		"authentifcation": "authentication",
		"troubleshoot": "troubleshooting",
		"deployement":  "deployment",
		"maintainence": "maintenance",
	}
}

// CorrectQuery corrects spelling errors in the query
func (sc *SpellChecker) CorrectQuery(query string) (string, map[string]string) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	corrections := make(map[string]string)
	words := strings.Fields(query)
	correctedWords := make([]string, len(words))

	for i, word := range words {
		// Clean word (remove punctuation for checking)
		cleanWord := regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		cleanWordLower := strings.ToLower(cleanWord)

		// Check if there's a direct correction
		if correction, exists := sc.corrections[cleanWordLower]; exists {
			correctedWords[i] = strings.Replace(word, cleanWord, correction, 1)
			corrections[cleanWord] = correction
		} else if !sc.dictionary[cleanWordLower] && len(cleanWord) > 2 {
			// Try to find a suggestion
			if suggestion := sc.findBestSuggestion(cleanWordLower); suggestion != "" {
				correctedWords[i] = strings.Replace(word, cleanWord, suggestion, 1)
				corrections[cleanWord] = suggestion
			} else {
				correctedWords[i] = word
			}
		} else {
			correctedWords[i] = word
		}
	}

	correctedQuery := strings.Join(correctedWords, " ")
	return correctedQuery, corrections
}

// findBestSuggestion finds the best spelling suggestion for a word
func (sc *SpellChecker) findBestSuggestion(word string) string {
	bestSuggestion := ""
	minDistance := sc.editDistance + 1

	// Check all dictionary words
	for dictWord := range sc.dictionary {
		distance := sc.editDistance(word, dictWord)
		if distance <= sc.editDistance && distance < minDistance {
			minDistance = distance
			bestSuggestion = dictWord
		}
	}

	return bestSuggestion
}

// editDistance calculates the edit distance between two strings
func (sc *SpellChecker) editDistance(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)
	matrix := make([][]int, len1+1)
	
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
		matrix[i][0] = i
	}
	
	for j := 1; j <= len2; j++ {
		matrix[0][j] = j
	}
	
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}
	
	return matrix[len1][len2]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}