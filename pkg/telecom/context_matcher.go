// Package telecom provides intelligent context matching and knowledge base services
// for telecommunications network intents. It includes pattern matching, semantic analysis,
// and context extraction capabilities for O-RAN network functions and configurations.
package telecom

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ContextMatcher provides intelligent context matching for telecom intents.

type ContextMatcher struct {
	knowledgeBase *TelecomKnowledgeBase

	patterns map[string]*regexp.Regexp

	synonyms map[string][]string

	abbreviations map[string]string

	contextCache map[string]*MatchResult

	initialized bool
}

// MatchResult contains the results of intent matching.

type MatchResult struct {
	Intent string `json:"intent"`

	NetworkFunctions []*NetworkFunctionMatch `json:"network_functions"`

	Interfaces []*InterfaceMatch `json:"interfaces"`

	SliceTypes []*SliceTypeMatch `json:"slice_types"`

	QosProfiles []*QosMatch `json:"qos_profiles"`

	DeploymentPatterns []*DeploymentMatch `json:"deployment_patterns"`

	KPIs []*KPIMatch `json:"kpis"`

	Requirements *RequirementExtraction `json:"requirements"`

	Context3GPP *Context3GPP `json:"context_3gpp"`

	Confidence float64 `json:"confidence"`

	Timestamp time.Time `json:"timestamp"`
}

// NetworkFunctionMatch represents a matched network function.

type NetworkFunctionMatch struct {
	Function *NetworkFunctionSpec `json:"function"`

	Confidence float64 `json:"confidence"`

	Reason string `json:"reason"`

	Context []string `json:"context"`
}

// InterfaceMatch represents a matched interface.

type InterfaceMatch struct {
	Interface *InterfaceSpec `json:"interface"`

	Confidence float64 `json:"confidence"`

	Reason string `json:"reason"`

	Context []string `json:"context"`
}

// SliceTypeMatch represents a matched slice type.

type SliceTypeMatch struct {
	SliceType *SliceTypeSpec `json:"slice_type"`

	Confidence float64 `json:"confidence"`

	Reason string `json:"reason"`

	Context []string `json:"context"`
}

// QosMatch represents a matched QoS profile.

type QosMatch struct {
	Profile *QosProfile `json:"profile"`

	Confidence float64 `json:"confidence"`

	Reason string `json:"reason"`

	Context []string `json:"context"`
}

// DeploymentMatch represents a matched deployment pattern.

type DeploymentMatch struct {
	Pattern *DeploymentPattern `json:"pattern"`

	Confidence float64 `json:"confidence"`

	Reason string `json:"reason"`

	Context []string `json:"context"`
}

// KPIMatch represents a matched KPI.

type KPIMatch struct {
	KPI *KPISpec `json:"kpi"`

	Confidence float64 `json:"confidence"`

	Reason string `json:"reason"`

	Context []string `json:"context"`
}

// RequirementExtraction contains extracted requirements from the intent.

type RequirementExtraction struct {
	Latency *LatencyRequirement `json:"latency,omitempty"`

	Throughput *ThroughputRequirement `json:"throughput,omitempty"`

	Reliability *ReliabilityRequirement `json:"reliability,omitempty"`

	Scalability *ScalabilityRequirement `json:"scalability,omitempty"`

	Security *SecurityRequirement `json:"security,omitempty"`

	Availability *AvailabilityRequirement `json:"availability,omitempty"`
}

// Context3GPP contains 3GPP specification context.

type Context3GPP struct {
	Specifications []string `json:"specifications"`

	Procedures []string `json:"procedures"`

	Interfaces []string `json:"interfaces"`

	Parameters map[string]string `json:"parameters"`

	Compliance []string `json:"compliance"`
}

// Requirement types.

type LatencyRequirement struct {
	Value float64 `json:"value"`

	Unit string `json:"unit"`

	Type string `json:"type"` // user-plane, control-plane, end-to-end
}

// ThroughputRequirement represents a throughputrequirement.

type ThroughputRequirement struct {
	Value float64 `json:"value"`

	Unit string `json:"unit"`

	Type string `json:"type"` // uplink, downlink, aggregate
}

// ReliabilityRequirement represents a reliabilityrequirement.

type ReliabilityRequirement struct {
	Value float64 `json:"value"`

	Unit string `json:"unit"`

	Type string `json:"type"` // availability, packet-loss, error-rate
}

// ScalabilityRequirement represents a scalabilityrequirement.

type ScalabilityRequirement struct {
	MinReplicas int `json:"min_replicas"`

	MaxReplicas int `json:"max_replicas"`

	Type string `json:"type"` // horizontal, vertical, auto
}

// SecurityRequirement represents a securityrequirement.

type SecurityRequirement struct {
	Level string `json:"level"` // basic, enhanced, high

	Features []string `json:"features"` // encryption, authentication, etc.

	Compliance []string `json:"compliance"` // GDPR, HIPAA, etc.

	Certificates []string `json:"certificates"` // TLS, mTLS, etc.
}

// AvailabilityRequirement represents a availabilityrequirement.

type AvailabilityRequirement struct {
	Value float64 `json:"value"`

	Unit string `json:"unit"`

	Type string `json:"type"` // service, system, network
}

// NewContextMatcher creates a new context matcher with initialized patterns.

func NewContextMatcher(kb *TelecomKnowledgeBase) *ContextMatcher {
	cm := &ContextMatcher{
		knowledgeBase: kb,

		patterns: make(map[string]*regexp.Regexp),

		synonyms: make(map[string][]string),

		abbreviations: make(map[string]string),

		contextCache: make(map[string]*MatchResult),
	}

	cm.initializePatterns()

	cm.initializeSynonyms()

	cm.initializeAbbreviations()

	cm.initialized = true

	return cm
}

// GetRelevantContext analyzes intent and returns relevant telecom context.

func (cm *ContextMatcher) GetRelevantContext(intent string) (*MatchResult, error) {
	if !cm.initialized {
		return nil, fmt.Errorf("context matcher not initialized")
	}

	// Check cache first.

	cacheKey := strings.ToLower(strings.TrimSpace(intent))

	if cached, exists := cm.contextCache[cacheKey]; exists {

		// Update timestamp.

		cached.Timestamp = time.Now()

		return cached, nil

	}

	result := &MatchResult{
		Intent: intent,

		Timestamp: time.Now(),
	}

	// Normalize intent.

	normalizedIntent := cm.normalizeIntent(intent)

	// Extract requirements.

	result.Requirements = cm.extractRequirements(normalizedIntent)

	// Match network functions.

	result.NetworkFunctions = cm.matchNetworkFunctions(normalizedIntent)

	// Match interfaces.

	result.Interfaces = cm.matchInterfaces(normalizedIntent)

	// Match slice types.

	result.SliceTypes = cm.matchSliceTypes(normalizedIntent)

	// Match QoS profiles.

	result.QosProfiles = cm.matchQosProfiles(normalizedIntent)

	// Match deployment patterns.

	result.DeploymentPatterns = cm.matchDeploymentPatterns(normalizedIntent)

	// Match KPIs.

	result.KPIs = cm.matchKPIs(normalizedIntent)

	// Build 3GPP context.

	result.Context3GPP = cm.build3GPPContext(result)

	// Calculate overall confidence.

	result.Confidence = cm.calculateConfidence(result)

	// Cache the result.

	cm.contextCache[cacheKey] = result

	return result, nil
}

// normalizeIntent cleans and normalizes the input intent.

func (cm *ContextMatcher) normalizeIntent(intent string) string {
	// Convert to lowercase.

	normalized := strings.ToLower(intent)

	// Expand abbreviations.

	for abbr, expansion := range cm.abbreviations {
		normalized = strings.ReplaceAll(normalized, abbr, expansion)
	}

	// Remove extra whitespace.

	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	normalized = strings.TrimSpace(normalized)

	return normalized
}

// matchNetworkFunctions finds relevant network functions based on intent.

func (cm *ContextMatcher) matchNetworkFunctions(intent string) []*NetworkFunctionMatch {
	var matches []*NetworkFunctionMatch

	for name, nf := range cm.knowledgeBase.NetworkFunctions {

		confidence := 0.0

		var reasons []string

		var context []string

		// Direct name match.

		if cm.containsWord(intent, name) || cm.containsWord(intent, nf.Name) {

			confidence += 0.9

			reasons = append(reasons, "direct_name_match")

			context = append(context, fmt.Sprintf("Found '%s' in intent", name))

		}

		// Type match.

		if cm.containsWord(intent, nf.Type) {

			confidence += 0.3

			reasons = append(reasons, "type_match")

		}

		// Description keywords.

		descWords := strings.Fields(strings.ToLower(nf.Description))

		for _, word := range descWords {
			if len(word) > 3 && cm.containsWord(intent, word) {

				confidence += 0.1

				reasons = append(reasons, "description_keyword")

			}
		}

		// Interface mentions.

		for _, iface := range nf.Interfaces {
			if cm.containsWord(intent, iface) {

				confidence += 0.4

				reasons = append(reasons, "interface_match")

				context = append(context, fmt.Sprintf("Interface %s mentioned", iface))

			}
		}

		// Use case patterns.

		confidence += cm.matchUseCasePatterns(intent, nf)

		// Synonym matching.

		for _, synonyms := range cm.synonyms {
			for _, synonym := range synonyms {
				if synonym == name && cm.containsAny(intent, synonyms) {

					confidence += 0.6

					reasons = append(reasons, "synonym_match")

					break

				}
			}
		}

		if confidence > 0.1 {
			matches = append(matches, &NetworkFunctionMatch{
				Function: nf,

				Confidence: confidence,

				Reason: strings.Join(reasons, ","),

				Context: context,
			})
		}

	}

	// Sort by confidence.

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	// Limit to top 5 matches.

	if len(matches) > 5 {
		matches = matches[:5]
	}

	return matches
}

// matchInterfaces finds relevant interfaces based on intent.

func (cm *ContextMatcher) matchInterfaces(intent string) []*InterfaceMatch {
	var matches []*InterfaceMatch

	for name, iface := range cm.knowledgeBase.Interfaces {

		confidence := 0.0

		var reasons []string

		var context []string

		// Direct interface name match.

		if cm.containsWord(intent, name) || cm.containsWord(intent, iface.Name) {

			confidence += 0.8

			reasons = append(reasons, "interface_name_match")

			context = append(context, fmt.Sprintf("Interface %s mentioned", iface.Name))

		}

		// Protocol match.

		for _, protocol := range iface.Protocol {
			if cm.containsWord(intent, protocol) {

				confidence += 0.5

				reasons = append(reasons, "protocol_match")

				context = append(context, fmt.Sprintf("Protocol %s mentioned", protocol))

			}
		}

		// Description keywords.

		if cm.matchDescription(intent, iface.Description) {

			confidence += 0.2

			reasons = append(reasons, "description_match")

		}

		if confidence > 0.1 {
			matches = append(matches, &InterfaceMatch{
				Interface: iface,

				Confidence: confidence,

				Reason: strings.Join(reasons, ","),

				Context: context,
			})
		}

	}

	// Sort by confidence.

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches
}

// matchSliceTypes finds relevant slice types based on intent.

func (cm *ContextMatcher) matchSliceTypes(intent string) []*SliceTypeMatch {
	var matches []*SliceTypeMatch

	for name, slice := range cm.knowledgeBase.SliceTypes {

		confidence := 0.0

		var reasons []string

		var context []string

		// Direct slice type match.

		if cm.containsWord(intent, name) || cm.containsWord(intent, slice.Description) {

			confidence += 0.9

			reasons = append(reasons, "slice_type_match")

		}

		// Use case match.

		if cm.containsWord(intent, slice.UseCase) {

			confidence += 0.7

			reasons = append(reasons, "use_case_match")

			context = append(context, fmt.Sprintf("Use case: %s", slice.UseCase))

		}

		// SST match.

		sstStr := fmt.Sprintf("sst %d", slice.SST)

		if strings.Contains(intent, sstStr) || strings.Contains(intent, fmt.Sprintf("sst=%d", slice.SST)) {

			confidence += 0.8

			reasons = append(reasons, "sst_match")

		}

		// Requirement-based matching.

		confidence += cm.matchSliceRequirements(intent, slice)

		if confidence > 0.1 {
			matches = append(matches, &SliceTypeMatch{
				SliceType: slice,

				Confidence: confidence,

				Reason: strings.Join(reasons, ","),

				Context: context,
			})
		}

	}

	// Sort by confidence.

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches
}

// matchQosProfiles finds relevant QoS profiles based on intent.

func (cm *ContextMatcher) matchQosProfiles(intent string) []*QosMatch {
	var matches []*QosMatch

	for _, qos := range cm.knowledgeBase.QosProfiles {

		confidence := 0.0

		var reasons []string

		var context []string

		// QCI/5QI match.

		qciStr := fmt.Sprintf("qci %d", qos.QCI)

		qfiStr := fmt.Sprintf("qfi %d", qos.QFI)

		if strings.Contains(intent, qciStr) || strings.Contains(intent, qfiStr) {

			confidence += 0.9

			reasons = append(reasons, "qci_qfi_match")

		}

		// Resource type match.

		if cm.containsWord(intent, strings.ToLower(qos.Resource)) {

			confidence += 0.6

			reasons = append(reasons, "resource_type_match")

		}

		// Priority-based matching.

		if cm.matchPriorityRequirements(intent, qos.Priority) {

			confidence += 0.4

			reasons = append(reasons, "priority_match")

		}

		// Delay budget matching.

		if cm.matchLatencyRequirements(intent, float64(qos.DelayBudget)) {

			confidence += 0.5

			reasons = append(reasons, "delay_budget_match")

		}

		if confidence > 0.1 {
			matches = append(matches, &QosMatch{
				Profile: qos,

				Confidence: confidence,

				Reason: strings.Join(reasons, ","),

				Context: context,
			})
		}

	}

	// Sort by confidence.

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches
}

// matchDeploymentPatterns finds relevant deployment patterns.

func (cm *ContextMatcher) matchDeploymentPatterns(intent string) []*DeploymentMatch {
	var matches []*DeploymentMatch

	for name, pattern := range cm.knowledgeBase.DeploymentTypes {

		confidence := 0.0

		var reasons []string

		var context []string

		// Direct pattern name match.

		if cm.containsWord(intent, name) || cm.containsWord(intent, pattern.Name) {

			confidence += 0.8

			reasons = append(reasons, "pattern_name_match")

		}

		// Use case match.

		for _, useCase := range pattern.UseCase {
			if cm.containsWord(intent, useCase) {

				confidence += 0.6

				reasons = append(reasons, "use_case_match")

				context = append(context, fmt.Sprintf("Use case: %s", useCase))

			}
		}

		// Architecture keywords.

		if cm.matchArchitectureKeywords(intent, &pattern.Architecture) {

			confidence += 0.4

			reasons = append(reasons, "architecture_match")

		}

		if confidence > 0.1 {
			matches = append(matches, &DeploymentMatch{
				Pattern: pattern,

				Confidence: confidence,

				Reason: strings.Join(reasons, ","),

				Context: context,
			})
		}

	}

	// Sort by confidence.

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches
}

// matchKPIs finds relevant KPIs based on intent.

func (cm *ContextMatcher) matchKPIs(intent string) []*KPIMatch {
	var matches []*KPIMatch

	for _, kpi := range cm.knowledgeBase.PerformanceKPIs {

		confidence := 0.0

		var reasons []string

		var context []string

		// Direct KPI name match.

		kpiWords := strings.Fields(strings.ToLower(kpi.Name))

		matchCount := 0

		for _, word := range kpiWords {
			if cm.containsWord(intent, word) {
				matchCount++
			}
		}

		if matchCount > 0 {

			confidence += float64(matchCount) * 0.3

			reasons = append(reasons, "kpi_name_match")

		}

		// Category match.

		if cm.containsWord(intent, kpi.Category) {

			confidence += 0.4

			reasons = append(reasons, "category_match")

		}

		// Type match.

		if cm.containsWord(intent, kpi.Type) {

			confidence += 0.2

			reasons = append(reasons, "type_match")

		}

		if confidence > 0.1 {
			matches = append(matches, &KPIMatch{
				KPI: kpi,

				Confidence: confidence,

				Reason: strings.Join(reasons, ","),

				Context: context,
			})
		}

	}

	// Sort by confidence.

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	return matches
}

// extractRequirements extracts quantified requirements from intent.

func (cm *ContextMatcher) extractRequirements(intent string) *RequirementExtraction {
	req := &RequirementExtraction{}

	// Extract latency requirements.

	if latency := cm.extractLatencyRequirement(intent); latency != nil {
		req.Latency = latency
	}

	// Extract throughput requirements.

	if throughput := cm.extractThroughputRequirement(intent); throughput != nil {
		req.Throughput = throughput
	}

	// Extract reliability requirements.

	if reliability := cm.extractReliabilityRequirement(intent); reliability != nil {
		req.Reliability = reliability
	}

	// Extract scalability requirements.

	if scalability := cm.extractScalabilityRequirement(intent); scalability != nil {
		req.Scalability = scalability
	}

	// Extract security requirements.

	if security := cm.extractSecurityRequirement(intent); security != nil {
		req.Security = security
	}

	// Extract availability requirements.

	if availability := cm.extractAvailabilityRequirement(intent); availability != nil {
		req.Availability = availability
	}

	return req
}

// build3GPPContext builds relevant 3GPP specification context.

func (cm *ContextMatcher) build3GPPContext(result *MatchResult) *Context3GPP {
	context := &Context3GPP{
		Specifications: make([]string, 0, 10),

		Procedures: make([]string, 0, 5),

		Interfaces: make([]string, 0, 8),

		Parameters: make(map[string]string),

		Compliance: make([]string, 0, 5),
	}

	specSet := make(map[string]bool)

	// Collect specifications from matched network functions.

	for _, nfMatch := range result.NetworkFunctions {
		for _, spec := range nfMatch.Function.Specification3GPP {
			if !specSet[spec] {

				context.Specifications = append(context.Specifications, spec)

				specSet[spec] = true

			}
		}
	}

	// Collect interface specifications.

	for _, ifaceMatch := range result.Interfaces {
		if ifaceMatch.Interface.Specification != "" && !specSet[ifaceMatch.Interface.Specification] {

			context.Specifications = append(context.Specifications, ifaceMatch.Interface.Specification)

			specSet[ifaceMatch.Interface.Specification] = true

		}
	}

	// Add relevant procedures based on context.

	context.Procedures = cm.getRelevantProcedures(result)

	// Add compliance standards.

	context.Compliance = []string{"3GPP Release 17", "O-RAN Alliance", "ETSI NFV"}

	// Sort specifications.

	sort.Strings(context.Specifications)

	return context
}

// Helper methods for pattern matching.

func (cm *ContextMatcher) containsWord(text, word string) bool {
	if word == "" {
		return false
	}

	pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(strings.ToLower(word)))

	matched, _ := regexp.MatchString(pattern, strings.ToLower(text))

	return matched
}

func (cm *ContextMatcher) containsAny(text string, words []string) bool {
	for _, word := range words {
		if cm.containsWord(text, word) {
			return true
		}
	}

	return false
}

func (cm *ContextMatcher) matchDescription(intent, description string) bool {
	words := strings.Fields(strings.ToLower(description))

	matchCount := 0

	for _, word := range words {
		if len(word) > 3 && cm.containsWord(intent, word) {
			matchCount++
		}
	}

	return matchCount > 2
}

func (cm *ContextMatcher) matchUseCasePatterns(intent string, nf *NetworkFunctionSpec) float64 {
	confidence := 0.0

	// AMF specific patterns.

	if nf.Name == "AMF" {

		patterns := []string{"registration", "authentication", "mobility", "access"}

		for _, pattern := range patterns {
			if cm.containsWord(intent, pattern) {
				confidence += 0.2
			}
		}

	}

	// SMF specific patterns.

	if nf.Name == "SMF" {

		patterns := []string{"session", "pdu", "ip allocation", "routing"}

		for _, pattern := range patterns {
			if cm.containsWord(intent, pattern) {
				confidence += 0.2
			}
		}

	}

	// UPF specific patterns.

	if nf.Name == "UPF" {

		patterns := []string{"forwarding", "qos", "user plane", "packet"}

		for _, pattern := range patterns {
			if cm.containsWord(intent, pattern) {
				confidence += 0.2
			}
		}

	}

	return confidence
}

func (cm *ContextMatcher) matchSliceRequirements(intent string, slice *SliceTypeSpec) float64 {
	confidence := 0.0

	// Check latency requirements.

	if cm.matchLatencyRequirements(intent, slice.Requirements.Latency.UserPlane) {
		confidence += 0.3
	}

	// Check throughput requirements.

	var throughputValue float64

	fmt.Sscanf(slice.Requirements.Throughput.Typical, "%f", &throughputValue)

	if cm.matchThroughputRequirements(intent, throughputValue) {
		confidence += 0.3
	}

	// Check reliability requirements.

	if cm.matchReliabilityRequirements(intent, slice.Requirements.Reliability.Availability) {
		confidence += 0.2
	}

	return confidence
}

func (cm *ContextMatcher) matchArchitectureKeywords(intent string, arch *DeploymentArchitecture) bool {
	keywords := []string{arch.Type, arch.Redundancy, arch.DataPlane, arch.ControlPlane}

	for _, keyword := range keywords {
		if cm.containsWord(intent, keyword) {
			return true
		}
	}

	return false
}

func (cm *ContextMatcher) matchLatencyRequirements(intent string, latency float64) bool {
	// Look for latency-related keywords and values.

	patterns := []string{
		`(\d+(?:\.\d+)?)\s*(?:ms|millisecond|msec)`,

		`latency.*?(\d+(?:\.\d+)?)`,

		`delay.*?(\d+(?:\.\d+)?)`,

		`response.*?time.*?(\d+(?:\.\d+)?)`,
	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(intent)

		if len(matches) > 1 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return value <= latency*1.5 // Allow 50% tolerance
			}
		}

	}

	return false
}

func (cm *ContextMatcher) matchThroughputRequirements(intent string, throughput float64) bool {
	patterns := []string{
		`(\d+(?:\.\d+)?)\s*(?:mbps|gbps|kbps)`,

		`throughput.*?(\d+(?:\.\d+)?)`,

		`bandwidth.*?(\d+(?:\.\d+)?)`,

		`speed.*?(\d+(?:\.\d+)?)`,
	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(intent)

		if len(matches) > 1 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return value >= throughput*0.5 // Allow for lower bounds
			}
		}

	}

	return false
}

func (cm *ContextMatcher) matchReliabilityRequirements(intent string, reliability float64) bool {
	patterns := []string{
		`(\d+(?:\.\d+)?)\s*%.*?(?:availability|uptime|reliability)`,

		`(?:availability|uptime|reliability).*?(\d+(?:\.\d+)?)\s*%`,

		`nine.*?(\d+)`,
	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(intent)

		if len(matches) > 1 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {
				return value >= reliability*0.95 // Allow 5% tolerance
			}
		}

	}

	return false
}

func (cm *ContextMatcher) matchPriorityRequirements(intent string, priority int) bool {
	patterns := []string{
		`priority\s*(\d+)`,

		`high.*?priority`,

		`low.*?priority`,

		`critical`,

		`urgent`,
	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		if re.MatchString(intent) {

			matches := re.FindStringSubmatch(intent)

			if len(matches) > 1 {
				if value, err := strconv.Atoi(matches[1]); err == nil {
					return value <= priority+2 // Allow some tolerance
				}
			}

			// Handle qualitative priorities.

			if strings.Contains(intent, "high") || strings.Contains(intent, "critical") {
				return priority <= 3
			}

			if strings.Contains(intent, "low") {
				return priority >= 7
			}

		}

	}

	return false
}

// Requirement extraction methods.

func (cm *ContextMatcher) extractLatencyRequirement(intent string) *LatencyRequirement {
	patterns := []string{
		`(?:latency|delay|response.*?time).*?(\d+(?:\.\d+)?)\s*(ms|millisecond|msec|μs|microsecond|s|second)`,

		`(\d+(?:\.\d+)?)\s*(ms|millisecond|msec|μs|microsecond|s|second).*?(?:latency|delay)`,
	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(intent)

		if len(matches) > 2 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {

				unit := strings.ToLower(matches[2])

				reqType := "user-plane"

				if strings.Contains(intent, "control") {
					reqType = "control-plane"
				} else if strings.Contains(intent, "end-to-end") {
					reqType = "end-to-end"
				}

				return &LatencyRequirement{
					Value: value,

					Unit: unit,

					Type: reqType,
				}

			}
		}

	}

	return nil
}

func (cm *ContextMatcher) extractThroughputRequirement(intent string) *ThroughputRequirement {
	patterns := []string{
		`(?i)(?:throughput|bandwidth|speed).*?(\d+(?:\.\d+)?)\s*(mbps|gbps|kbps|mb/s|gb/s|kb/s)`,

		`(?i)(\d+(?:\.\d+)?)\s*(mbps|gbps|kbps|mb/s|gb/s|kb/s).*?(?:throughput|bandwidth)`,
	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(intent)

		if len(matches) > 2 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {

				unit := strings.ToLower(matches[2])

				reqType := "aggregate"

				lowerIntent := strings.ToLower(intent)

				if strings.Contains(lowerIntent, "uplink") || strings.Contains(lowerIntent, "ul") {
					reqType = "uplink"
				} else if strings.Contains(lowerIntent, "downlink") || strings.Contains(lowerIntent, "dl") {
					reqType = "downlink"
				}

				return &ThroughputRequirement{
					Value: value,

					Unit: unit,

					Type: reqType,
				}

			}
		}

	}

	return nil
}

func (cm *ContextMatcher) extractReliabilityRequirement(intent string) *ReliabilityRequirement {
	patterns := []string{
		`(?:availability|uptime|reliability).*?(\d+(?:\.\d+)?)\s*%`,

		`(\d+(?:\.\d+)?)\s*%.*?(?:availability|uptime|reliability)`,

		`(?:packet.*?loss).*?(\d+(?:\.\d+)?)\s*%`,

		`(\d+(?:\.\d+)?)\s*%.*?(?:packet.*?loss)`,
	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(intent)

		if len(matches) > 1 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {

				reqType := "availability"

				if strings.Contains(pattern, "packet") {
					reqType = "packet-loss"
				}

				return &ReliabilityRequirement{
					Value: value,

					Unit: "percentage",

					Type: reqType,
				}

			}
		}

	}

	return nil
}

func (cm *ContextMatcher) extractScalabilityRequirement(intent string) *ScalabilityRequirement {
	req := &ScalabilityRequirement{
		Type: "horizontal",
	}

	// Extract replica counts.

	patterns := []string{
		`(?:min.*?replica).*?(\d+)`,

		`(?:max.*?replica).*?(\d+)`,

		`(?:replica).*?(\d+)`,

		`(?:instance).*?(\d+)`,
	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(intent)

		if len(matches) > 1 {
			if value, err := strconv.Atoi(matches[1]); err == nil {
				if strings.Contains(pattern, "min") {
					req.MinReplicas = value
				} else if strings.Contains(pattern, "max") {
					req.MaxReplicas = value
				} else {
					req.MaxReplicas = value
				}
			}
		}

	}

	// Determine scaling type.

	if strings.Contains(intent, "vertical") {
		req.Type = "vertical"
	} else if strings.Contains(intent, "auto") {
		req.Type = "auto"
	}

	// Only return if we found some scaling information.

	if req.MinReplicas > 0 || req.MaxReplicas > 0 {
		return req
	}

	return nil
}

func (cm *ContextMatcher) extractSecurityRequirement(intent string) *SecurityRequirement {
	req := &SecurityRequirement{
		Features: make([]string, 0, 8),

		Compliance: make([]string, 0, 5),

		Certificates: make([]string, 0, 3),
	}

	// Determine security level.

	if strings.Contains(intent, "high security") || strings.Contains(intent, "maximum security") {
		req.Level = "high"
	} else if strings.Contains(intent, "enhanced security") {
		req.Level = "enhanced"
	} else if strings.Contains(intent, "security") {
		req.Level = "basic"
	} else {
		return nil
	}

	// Extract security features.

	securityKeywords := []string{"encryption", "authentication", "authorization", "tls", "mtls", "oauth", "rbac"}

	for _, keyword := range securityKeywords {
		if cm.containsWord(intent, keyword) {

			req.Features = append(req.Features, keyword)

			if strings.Contains(keyword, "tls") {
				req.Certificates = append(req.Certificates, keyword)
			}

		}
	}

	// Extract compliance requirements.

	complianceKeywords := []string{"gdpr", "hipaa", "pci", "sox", "iso27001"}

	for _, keyword := range complianceKeywords {
		if cm.containsWord(intent, keyword) {
			req.Compliance = append(req.Compliance, strings.ToUpper(keyword))
		}
	}

	return req
}

func (cm *ContextMatcher) extractAvailabilityRequirement(intent string) *AvailabilityRequirement {
	patterns := []string{
		`(?:availability|uptime).*?(\d+(?:\.\d+)?)\s*%`,

		`(\d+(?:\.\d+)?)\s*%.*?(?:availability|uptime)`,

		`(?:nine|9).*?(\d+)`, // "five nines" = 99.999%

	}

	for _, pattern := range patterns {

		re := regexp.MustCompile(pattern)

		matches := re.FindStringSubmatch(intent)

		if len(matches) > 1 {
			if value, err := strconv.ParseFloat(matches[1], 64); err == nil {

				// Handle "nines" notation.

				if strings.Contains(pattern, "nine") {

					nines := value

					value = 100 - (100 / pow10(int(nines)))

				}

				reqType := "service"

				if strings.Contains(intent, "system") {
					reqType = "system"
				} else if strings.Contains(intent, "network") {
					reqType = "network"
				}

				return &AvailabilityRequirement{
					Value: value,

					Unit: "percentage",

					Type: reqType,
				}

			}
		}

	}

	return nil
}

func (cm *ContextMatcher) getRelevantProcedures(result *MatchResult) []string {
	procedures := make([]string, 0, 10)

	// Add procedures based on matched network functions.

	for _, nfMatch := range result.NetworkFunctions {
		switch nfMatch.Function.Name {

		case "AMF":

			procedures = append(procedures, "UE Registration", "Authentication", "Mobility Management")

		case "SMF":

			procedures = append(procedures, "PDU Session Establishment", "PDU Session Modification")

		case "UPF":

			procedures = append(procedures, "User Plane Setup", "QoS Flow Management")

		case "PCF":

			procedures = append(procedures, "Policy Control", "Charging Control")

		}
	}

	return procedures
}

func (cm *ContextMatcher) calculateConfidence(result *MatchResult) float64 {
	totalConfidence := 0.0

	matchCount := 0

	// Weight network function matches highly.

	for _, match := range result.NetworkFunctions {

		totalConfidence += match.Confidence * 0.4

		matchCount++

	}

	// Weight interface matches.

	for _, match := range result.Interfaces {

		totalConfidence += match.Confidence * 0.2

		matchCount++

	}

	// Weight slice type matches.

	for _, match := range result.SliceTypes {

		totalConfidence += match.Confidence * 0.3

		matchCount++

	}

	// Add other matches with lower weights.

	for _, match := range result.QosProfiles {

		totalConfidence += match.Confidence * 0.1

		matchCount++

	}

	if matchCount == 0 {
		return 0.0
	}

	return totalConfidence / float64(matchCount)
}

// Helper function for power of 10.

func pow10(n int) float64 {
	result := 1.0

	for range n {
		result *= 10
	}

	return result
}

// Initialize patterns, synonyms, and abbreviations.

func (cm *ContextMatcher) initializePatterns() {
	// Compile common patterns for better performance.

	cm.patterns["latency"] = regexp.MustCompile(`(?:latency|delay|response.*?time).*?(\d+(?:\.\d+)?)\s*(ms|millisecond|msec|μs|microsecond|s|second)`)

	cm.patterns["throughput"] = regexp.MustCompile(`(?:throughput|bandwidth|speed).*?(\d+(?:\.\d+)?)\s*(mbps|gbps|kbps)`)

	cm.patterns["availability"] = regexp.MustCompile(`(?:availability|uptime).*?(\d+(?:\.\d+)?)\s*%`)

	cm.patterns["replicas"] = regexp.MustCompile(`(?:replica|instance).*?(\d+)`)
}

func (cm *ContextMatcher) initializeSynonyms() {
	cm.synonyms["amf"] = []string{"amf", "access mobility function", "access and mobility management function"}

	cm.synonyms["smf"] = []string{"smf", "session management function"}

	cm.synonyms["upf"] = []string{"upf", "user plane function"}

	cm.synonyms["pcf"] = []string{"pcf", "policy control function"}

	cm.synonyms["udm"] = []string{"udm", "unified data management"}

	cm.synonyms["ausf"] = []string{"ausf", "authentication server function"}

	cm.synonyms["nrf"] = []string{"nrf", "network repository function"}

	cm.synonyms["nssf"] = []string{"nssf", "network slice selection function"}

	cm.synonyms["gnb"] = []string{"gnb", "gnodeb", "5g base station", "base station"}

	// Slice types.

	cm.synonyms["embb"] = []string{"embb", "enhanced mobile broadband", "broadband"}

	cm.synonyms["urllc"] = []string{"urllc", "ultra reliable low latency", "low latency", "ultra reliable"}

	cm.synonyms["mmtc"] = []string{"mmtc", "massive machine type communications", "iot", "machine type"}

	// Interfaces.

	cm.synonyms["n1"] = []string{"n1", "ue amf interface", "nas"}

	cm.synonyms["n2"] = []string{"n2", "gnb amf interface", "ngap"}

	cm.synonyms["n3"] = []string{"n3", "gnb upf interface", "user plane"}

	cm.synonyms["n4"] = []string{"n4", "smf upf interface", "pfcp"}
}

func (cm *ContextMatcher) initializeAbbreviations() {
	// Common telecom abbreviations.

	cm.abbreviations["5gc"] = "5g core"

	cm.abbreviations["ran"] = "radio access network"

	cm.abbreviations["sba"] = "service based architecture"

	cm.abbreviations["nf"] = "network function"

	cm.abbreviations["vnf"] = "virtual network function"

	cm.abbreviations["cnf"] = "cloud native function"

	cm.abbreviations["pdu"] = "packet data unit"

	cm.abbreviations["qos"] = "quality of service"

	cm.abbreviations["sla"] = "service level agreement"

	cm.abbreviations["kpi"] = "key performance indicator"

	cm.abbreviations["rps"] = "requests per second"

	cm.abbreviations["tps"] = "transactions per second"

	cm.abbreviations["cpu"] = "central processing unit"

	cm.abbreviations["ram"] = "random access memory"

	cm.abbreviations["ssd"] = "solid state drive"

	cm.abbreviations["hdd"] = "hard disk drive"

	cm.abbreviations["nvme"] = "non volatile memory express"

	cm.abbreviations["iops"] = "input output operations per second"

	cm.abbreviations["tls"] = "transport layer security"

	cm.abbreviations["mtls"] = "mutual transport layer security"

	cm.abbreviations["rbac"] = "role based access control"

	cm.abbreviations["jwt"] = "json web token"

	cm.abbreviations["oauth"] = "open authorization"

	cm.abbreviations["http"] = "hypertext transfer protocol"

	cm.abbreviations["https"] = "hypertext transfer protocol secure"

	cm.abbreviations["tcp"] = "transmission control protocol"

	cm.abbreviations["udp"] = "user datagram protocol"

	cm.abbreviations["ip"] = "internet protocol"

	cm.abbreviations["dns"] = "domain name system"
}

// IsInitialized returns whether the context matcher is initialized.

func (cm *ContextMatcher) IsInitialized() bool {
	return cm.initialized
}

// ClearCache clears the context matching cache.

func (cm *ContextMatcher) ClearCache() {
	cm.contextCache = make(map[string]*MatchResult)
}

// GetCacheSize returns the number of cached results.

func (cm *ContextMatcher) GetCacheSize() int {
	return len(cm.contextCache)
}
