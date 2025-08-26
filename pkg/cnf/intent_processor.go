/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cnf

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// CNFIntentProcessor processes natural language intents for CNF deployment
type CNFIntentProcessor struct {
	Client           client.Client
	LLMProcessor     llm.Processor
	RAGService       rag.Service
	KnowledgeBase    *CNFKnowledgeBase
	TemplateRegistry *CNFTemplateRegistry
	Config           *CNFIntentProcessorConfig
}

// CNFIntentProcessorConfig holds configuration for the CNF intent processor
type CNFIntentProcessorConfig struct {
	MaxProcessingTime        time.Duration
	ConfidenceThreshold      float64
	EnableContextEnrichment  bool
	EnableCostEstimation     bool
	EnableTimelineEstimation bool
	DefaultStrategy          nephoranv1.DeploymentStrategy
	EnableValidation         bool
	MaxRetries               int
}

// CNFKnowledgeBase contains CNF-specific knowledge for intent processing
type CNFKnowledgeBase struct {
	FunctionMappings        map[string]nephoranv1.CNFFunction
	ResourceRequirements    map[nephoranv1.CNFFunction]ResourceProfile
	InterfaceSpecifications map[nephoranv1.CNFFunction][]InterfaceProfile
	DeploymentPatterns      map[nephoranv1.CNFType]DeploymentPattern
	PerformanceProfiles     map[string]PerformanceProfile
	SecurityProfiles        map[string]SecurityProfile
	CostModels              map[nephoranv1.CNFFunction]CostModel
}

// ResourceProfile defines resource characteristics for CNF functions
type ResourceProfile struct {
	BaselineResources CNFResourceRequirements            `json:"baseline"`
	ScalingFactors    map[string]float64                 `json:"scaling_factors"`
	PerformanceTiers  map[string]CNFResourceRequirements `json:"performance_tiers"`
	Dependencies      []nephoranv1.CNFFunction           `json:"dependencies"`
	Characteristics   map[string]interface{}             `json:"characteristics"`
}

// CNFResourceRequirements defines resource requirements for CNFs
type CNFResourceRequirements struct {
	CPU              string            `json:"cpu"`
	Memory           string            `json:"memory"`
	Storage          string            `json:"storage"`
	NetworkBandwidth string            `json:"network_bandwidth"`
	GPU              int32             `json:"gpu"`
	DPDK             *DPDKRequirements `json:"dpdk,omitempty"`
	Hugepages        map[string]string `json:"hugepages,omitempty"`
}

// DPDKRequirements defines DPDK-specific requirements
type DPDKRequirements struct {
	Enabled bool   `json:"enabled"`
	Cores   int32  `json:"cores"`
	Memory  int32  `json:"memory"`
	Driver  string `json:"driver"`
}

// InterfaceProfile defines interface characteristics
type InterfaceProfile struct {
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Protocols        []string `json:"protocols"`
	DefaultPort      int32    `json:"default_port"`
	Mandatory        bool     `json:"mandatory"`
	BandwidthProfile string   `json:"bandwidth_profile"`
	LatencyProfile   string   `json:"latency_profile"`
}

// DeploymentPattern defines deployment patterns for CNF types
type DeploymentPattern struct {
	PreferredStrategy      nephoranv1.DeploymentStrategy   `json:"preferred_strategy"`
	SupportedStrategies    []nephoranv1.DeploymentStrategy `json:"supported_strategies"`
	DefaultReplicas        int32                           `json:"default_replicas"`
	ScalingCharacteristics map[string]interface{}          `json:"scaling_characteristics"`
	AffinityRules          []string                        `json:"affinity_rules"`
	SecurityRequirements   []string                        `json:"security_requirements"`
}

// PerformanceProfile defines performance characteristics
type PerformanceProfile struct {
	LatencyTarget      int32             `json:"latency_target"`
	ThroughputTarget   int32             `json:"throughput_target"`
	AvailabilityTarget string            `json:"availability_target"`
	QoSRequirements    map[string]string `json:"qos_requirements"`
	SLARequirements    map[string]string `json:"sla_requirements"`
}

// SecurityProfile defines security characteristics
type SecurityProfile struct {
	SecurityLevel          string   `json:"security_level"`
	EncryptionRequired     []string `json:"encryption_required"`
	AuthenticationMethods  []string `json:"authentication_methods"`
	NetworkPolicies        []string `json:"network_policies"`
	PodSecurityStandard    string   `json:"pod_security_standard"`
	ComplianceRequirements []string `json:"compliance_requirements"`
}

// CostModel defines cost estimation parameters
type CostModel struct {
	BaseCostPerHour     float64            `json:"base_cost_per_hour"`
	ResourceMultipliers map[string]float64 `json:"resource_multipliers"`
	ScalingCostFactor   float64            `json:"scaling_cost_factor"`
	StorageCostPerGB    float64            `json:"storage_cost_per_gb"`
	NetworkCostPerGB    float64            `json:"network_cost_per_gb"`
}

// CNFIntentContext holds context information for intent processing
type CNFIntentContext struct {
	Intent              string
	RequestID           string
	UserID              string
	Namespace           string
	TargetCluster       string
	Priority            nephoranv1.NetworkPriority
	ProcessingStartTime time.Time
	RAGContext          map[string]interface{}
	LLMResponse         *llm.ProcessingResponse
	EnrichedContext     map[string]interface{}
}

// NewCNFIntentProcessor creates a new CNF intent processor
func NewCNFIntentProcessor(client client.Client, llmProcessor llm.Processor, ragService rag.Service) *CNFIntentProcessor {
	processor := &CNFIntentProcessor{
		Client:       client,
		LLMProcessor: llmProcessor,
		RAGService:   ragService,
		Config: &CNFIntentProcessorConfig{
			MaxProcessingTime:        5 * time.Minute,
			ConfidenceThreshold:      0.7,
			EnableContextEnrichment:  true,
			EnableCostEstimation:     true,
			EnableTimelineEstimation: true,
			DefaultStrategy:          "Helm",
			EnableValidation:         true,
			MaxRetries:               3,
		},
	}

	processor.initializeKnowledgeBase()
	return processor
}

// ProcessCNFIntent processes a natural language intent for CNF deployment
func (p *CNFIntentProcessor) ProcessCNFIntent(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) (*nephoranv1.CNFIntentProcessingResult, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing CNF intent", "intent", networkIntent.Name)

	// Create processing context
	intentContext := &CNFIntentContext{
		Intent:              networkIntent.Spec.Intent,
		RequestID:           string(networkIntent.UID),
		Namespace:           networkIntent.Namespace,
		TargetCluster:       networkIntent.Spec.TargetCluster,
		Priority:            networkIntent.Spec.Priority,
		ProcessingStartTime: time.Now(),
	}

	// Validate intent contains CNF-related keywords
	if !p.containsCNFKeywords(networkIntent.Spec.Intent) {
		return &nephoranv1.CNFIntentProcessingResult{
			ConfidenceScore: 0.0,
			Errors:          []string{"Intent does not contain CNF deployment keywords"},
		}, fmt.Errorf("intent does not appear to be CNF-related")
	}

	// Enrich context with RAG
	if err := p.enrichContextWithRAG(ctx, intentContext); err != nil {
		logger.Error(err, "Failed to enrich context with RAG")
		// Continue processing with reduced context
	}

	// Process with LLM
	llmResponse, err := p.processWithLLM(ctx, intentContext)
	if err != nil {
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}
	intentContext.LLMResponse = llmResponse

	// Extract CNF functions and requirements
	result, err := p.extractCNFDeploymentSpecs(ctx, intentContext)
	if err != nil {
		return nil, fmt.Errorf("failed to extract CNF deployment specs: %w", err)
	}

	// Validate extracted specifications
	if p.Config.EnableValidation {
		if err := p.validateCNFSpecs(result); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Validation warnings: %v", err))
		}
	}

	// Estimate resources and costs
	if p.Config.EnableCostEstimation {
		p.estimateResourcesAndCosts(result)
	}

	// Estimate deployment timeline
	if p.Config.EnableTimelineEstimation {
		p.estimateDeploymentTimeline(result)
	}

	// Calculate confidence score
	result.ConfidenceScore = p.calculateConfidenceScore(intentContext, result)

	logger.Info("CNF intent processing completed",
		"functions", len(result.DetectedFunctions),
		"deployments", len(result.CNFDeployments),
		"confidence", result.ConfidenceScore)

	return result, nil
}

// containsCNFKeywords checks if intent contains CNF-related keywords
func (p *CNFIntentProcessor) containsCNFKeywords(intent string) bool {
	intentLower := strings.ToLower(intent)

	cnfKeywords := []string{
		"cnf", "cloud native function", "containerized",
		"helm", "operator", "kubernetes",
		"deploy", "deployment", "scale", "scaling",
		"microservice", "container", "pod",
		"5g core", "o-ran", "network function",
		"amf", "smf", "upf", "nrf", "ausf", "udm",
		"o-du", "o-cu", "ric", "near-rt", "non-rt",
	}

	for _, keyword := range cnfKeywords {
		if strings.Contains(intentLower, keyword) {
			return true
		}
	}

	return false
}

// enrichContextWithRAG enriches the processing context using RAG
func (p *CNFIntentProcessor) enrichContextWithRAG(ctx context.Context, intentContext *CNFIntentContext) error {
	if p.RAGService == nil {
		return fmt.Errorf("RAG service not configured")
	}

	// Query RAG for CNF-related information
	ragQueries := p.generateRAGQueries(intentContext.Intent)
	ragContext := make(map[string]interface{})

	for _, query := range ragQueries {
		response, err := p.RAGService.Query(ctx, query)
		if err != nil {
			continue // Skip failed queries
		}
		ragContext[query] = response
	}

	intentContext.RAGContext = ragContext
	return nil
}

// generateRAGQueries generates relevant queries for RAG service
func (p *CNFIntentProcessor) generateRAGQueries(intent string) []string {
	queries := []string{
		"CNF deployment best practices",
		"5G Core network function requirements",
		"O-RAN network function specifications",
		"Kubernetes CNF orchestration",
		"Helm charts for telecommunications",
		"CNF resource requirements",
		"Network function scaling policies",
		"Service mesh for CNFs",
		"CNF monitoring and observability",
		"Telecommunications security requirements",
	}

	// Extract specific terms from intent and create targeted queries
	intentWords := strings.Fields(strings.ToLower(intent))
	for _, word := range intentWords {
		if p.isRelevantTerm(word) {
			queries = append(queries, fmt.Sprintf("%s CNF deployment", word))
			queries = append(queries, fmt.Sprintf("%s configuration requirements", word))
		}
	}

	return queries
}

// isRelevantTerm checks if a term is relevant for RAG queries
func (p *CNFIntentProcessor) isRelevantTerm(term string) bool {
	relevantTerms := map[string]bool{
		"amf": true, "smf": true, "upf": true, "nrf": true,
		"ausf": true, "udm": true, "pcf": true, "nssf": true,
		"o-du": true, "o-cu": true, "ric": true,
		"near-rt": true, "non-rt": true, "smo": true,
		"high-availability": true, "scaling": true, "monitoring": true,
		"security": true, "performance": true, "latency": true,
	}
	return relevantTerms[term]
}

// processWithLLM processes the intent using the LLM service
func (p *CNFIntentProcessor) processWithLLM(ctx context.Context, intentContext *CNFIntentContext) (*llm.ProcessingResponse, error) {
	if p.LLMProcessor == nil {
		return nil, fmt.Errorf("LLM processor not configured")
	}

	// Prepare LLM request with CNF-specific prompts
	request := p.prepareLLMRequest(intentContext)

	// Process with LLM
	response, err := p.LLMProcessor.ProcessIntent(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM processing failed: %w", err)
	}

	return response, nil
}

// prepareLLMRequest prepares the LLM request with CNF-specific context
func (p *CNFIntentProcessor) prepareLLMRequest(intentContext *CNFIntentContext) *llm.ProcessingRequest {
	systemPrompt := `You are an expert in Cloud Native Functions (CNFs) for telecommunications networks. 
Your task is to analyze natural language intents for CNF deployment and extract structured specifications.

Focus on:
1. Identifying specific CNF functions (AMF, SMF, UPF, O-DU, O-CU, RIC, etc.)
2. Determining resource requirements
3. Identifying scaling and performance requirements
4. Extracting high availability and security requirements
5. Determining deployment strategy preferences
6. Identifying network slicing requirements

Respond with structured JSON containing the extracted specifications.`

	userPrompt := fmt.Sprintf(`Analyze this CNF deployment intent and extract structured specifications:

Intent: "%s"

Consider the following context:
- Target Namespace: %s
- Target Cluster: %s
- Priority: %s

Extract:
1. CNF functions to deploy
2. Resource requirements
3. Scaling preferences
4. Performance requirements
5. Security requirements
6. High availability requirements
7. Recommended deployment strategy
8. Any network slicing requirements

Provide confidence scores for each extracted element.`,
		intentContext.Intent,
		intentContext.Namespace,
		intentContext.TargetCluster,
		intentContext.Priority)

	return &llm.ProcessingRequest{
		Intent:            intentContext.Intent,
		SystemPrompt:      systemPrompt,
		UserPrompt:        userPrompt,
		Context:           intentContext.RAGContext,
		MaxTokens:         2000,
		Temperature:       0.3,
		RequestID:         intentContext.RequestID,
		ProcessingTimeout: p.Config.MaxProcessingTime,
	}
}

// extractCNFDeploymentSpecs extracts CNF deployment specifications from LLM response
func (p *CNFIntentProcessor) extractCNFDeploymentSpecs(ctx context.Context, intentContext *CNFIntentContext) (*nephoranv1.CNFIntentProcessingResult, error) {
	result := &nephoranv1.CNFIntentProcessingResult{
		DetectedFunctions:  []nephoranv1.CNFFunction{},
		CNFDeployments:     []nephoranv1.CNFDeploymentIntent{},
		Warnings:           []string{},
		Errors:             []string{},
	}

	if intentContext.LLMResponse == nil {
		return result, fmt.Errorf("no LLM response available")
	}

	// Parse LLM response
	var llmResult map[string]interface{}
	if err := json.Unmarshal([]byte(intentContext.LLMResponse.ProcessedParameters), &llmResult); err != nil {
		// Fallback to pattern matching if JSON parsing fails
		return p.extractUsingPatterns(intentContext.Intent), nil
	}

	// Extract CNF functions
	if functions, ok := llmResult["cnf_functions"].([]interface{}); ok {
		for _, f := range functions {
			if funcStr, ok := f.(string); ok {
				if cnfFunc := p.mapToCNFFunction(funcStr); cnfFunc != "" {
					result.DetectedFunctions = append(result.DetectedFunctions, cnfFunc)
				}
			}
		}
	}

	// Extract deployment specifications for each function
	for _, function := range result.DetectedFunctions {
		cnfDeployment := p.createCNFDeploymentIntent(function, llmResult, intentContext)
		result.CNFDeployments = append(result.CNFDeployments, cnfDeployment)
	}

	// Extract recommended strategy
	if strategy, ok := llmResult["deployment_strategy"].(string); ok {
		result.RecommendedStrategy = p.mapToDeploymentStrategy(strategy)
	} else {
		result.RecommendedStrategy = p.Config.DefaultStrategy
	}

	return result, nil
}

// extractUsingPatterns extracts CNF specs using pattern matching (fallback)
func (p *CNFIntentProcessor) extractUsingPatterns(intent string) *nephoranv1.CNFIntentProcessingResult {
	result := &nephoranv1.CNFIntentProcessingResult{
		DetectedFunctions:   []nephoranv1.CNFFunction{},
		CNFDeployments:      []nephoranv1.CNFDeploymentIntent{},
		RecommendedStrategy: p.Config.DefaultStrategy,
		ConfidenceScore:     0.5, // Lower confidence for pattern matching
	}

	intentLower := strings.ToLower(intent)

	// Pattern matching for CNF functions
	functionPatterns := map[string]nephoranv1.CNFFunction{
		`\bamf\b`:         nephoranv1.CNFFunctionAMF,
		`\bsmf\b`:         nephoranv1.CNFFunctionSMF,
		`\bupf\b`:         nephoranv1.CNFFunctionUPF,
		`\bnrf\b`:         nephoranv1.CNFFunctionNRF,
		`\bausf\b`:        nephoranv1.CNFFunctionAUSF,
		`\budm\b`:         nephoranv1.CNFFunctionUDM,
		`\bpcf\b`:         nephoranv1.CNFFunctionPCF,
		`\bnssf\b`:        nephoranv1.CNFFunctionNSSF,
		`\bo-du\b`:        nephoranv1.CNFFunctionODU,
		`\bo-cu\b`:        nephoranv1.CNFFunctionOCUCP,
		`\bnear-rt-ric\b`: nephoranv1.CNFFunctionNearRTRIC,
		`\bnon-rt-ric\b`:  nephoranv1.CNFFunctionNonRTRIC,
	}

	for pattern, function := range functionPatterns {
		if matched, _ := regexp.MatchString(pattern, intentLower); matched {
			result.DetectedFunctions = append(result.DetectedFunctions, function)

			// Create basic deployment intent
			cnfDeployment := nephoranv1.CNFDeploymentIntent{
				Function:           function,
				CNFType:            p.determineCNFType(function),
				DeploymentStrategy: result.RecommendedStrategy,
				Replicas:           p.getDefaultReplicas(function),
			}
			result.CNFDeployments = append(result.CNFDeployments, cnfDeployment)
		}
	}

	return result
}

// mapToCNFFunction maps string to CNFFunction enum
func (p *CNFIntentProcessor) mapToCNFFunction(funcStr string) nephoranv1.CNFFunction {
	if p.KnowledgeBase != nil && p.KnowledgeBase.FunctionMappings != nil {
		if cnfFunc, exists := p.KnowledgeBase.FunctionMappings[strings.ToLower(funcStr)]; exists {
			return cnfFunc
		}
	}

	// Default mapping
	mappings := map[string]nephoranv1.CNFFunction{
		"amf":         nephoranv1.CNFFunctionAMF,
		"smf":         nephoranv1.CNFFunctionSMF,
		"upf":         nephoranv1.CNFFunctionUPF,
		"nrf":         nephoranv1.CNFFunctionNRF,
		"ausf":        nephoranv1.CNFFunctionAUSF,
		"udm":         nephoranv1.CNFFunctionUDM,
		"pcf":         nephoranv1.CNFFunctionPCF,
		"nssf":        nephoranv1.CNFFunctionNSSF,
		"o-du":        nephoranv1.CNFFunctionODU,
		"o-cu-cp":     nephoranv1.CNFFunctionOCUCP,
		"o-cu-up":     nephoranv1.CNFFunctionOCUUP,
		"near-rt-ric": nephoranv1.CNFFunctionNearRTRIC,
		"non-rt-ric":  nephoranv1.CNFFunctionNonRTRIC,
	}

	return mappings[strings.ToLower(funcStr)]
}

// mapToDeploymentStrategy maps string to deployment strategy enum
func (p *CNFIntentProcessor) mapToDeploymentStrategy(strategy string) nephoranv1.DeploymentStrategy {
	mappings := map[string]nephoranv1.DeploymentStrategy{
		"helm":     "Helm",
		"operator": "Operator", 
		"direct":   "Direct",
		"gitops":   "GitOps",
	}

	if strategy, exists := mappings[strings.ToLower(strategy)]; exists {
		return strategy
	}

	return p.Config.DefaultStrategy
}

// createCNFDeploymentIntent creates a CNF deployment intent from LLM result
func (p *CNFIntentProcessor) createCNFDeploymentIntent(function nephoranv1.CNFFunction, llmResult map[string]interface{}, intentContext *CNFIntentContext) nephoranv1.CNFDeploymentIntent {
	deployment := nephoranv1.CNFDeploymentIntent{
		Function:           function,
		CNFType:            p.determineCNFType(function),
		DeploymentStrategy: p.Config.DefaultStrategy,
		Replicas:           p.getDefaultReplicas(function),
	}

	// Extract resource requirements
	if resources, ok := llmResult["resource_requirements"].(map[string]interface{}); ok {
		deployment.Resources = p.parseResourceRequirements(resources, function)
	} else {
		deployment.Resources = p.getDefaultResources(function)
	}

	// Extract auto-scaling requirements
	if scaling, ok := llmResult["auto_scaling"].(map[string]interface{}); ok {
		deployment.AutoScaling = p.parseAutoScalingRequirements(scaling)
	}

	// Extract service mesh requirements
	if serviceMesh, ok := llmResult["service_mesh"].(map[string]interface{}); ok {
		deployment.ServiceMesh = p.parseServiceMeshRequirements(serviceMesh)
	}

	// Extract monitoring requirements
	if monitoring, ok := llmResult["monitoring"].(map[string]interface{}); ok {
		deployment.Monitoring = p.parseMonitoringRequirements(monitoring)
	}

	// Extract security requirements
	if security, ok := llmResult["security"].(map[string]interface{}); ok {
		deployment.Security = p.parseSecurityRequirements(security)
	}

	// Extract high availability requirements
	if ha, ok := llmResult["high_availability"].(map[string]interface{}); ok {
		deployment.HighAvailability = p.parseHighAvailabilityRequirements(ha)
	}

	// Extract performance requirements
	if performance, ok := llmResult["performance"].(map[string]interface{}); ok {
		deployment.Performance = p.parsePerformanceRequirements(performance)
	}

	return deployment
}

// Helper methods for parsing different requirement types

func (p *CNFIntentProcessor) determineCNFType(function nephoranv1.CNFFunction) nephoranv1.CNFType {
	coreTypes := []nephoranv1.CNFFunction{
		nephoranv1.CNFFunctionAMF, nephoranv1.CNFFunctionSMF, nephoranv1.CNFFunctionUPF,
		nephoranv1.CNFFunctionNRF, nephoranv1.CNFFunctionAUSF, nephoranv1.CNFFunctionUDM,
		nephoranv1.CNFFunctionPCF, nephoranv1.CNFFunctionNSSF,
	}

	oranTypes := []nephoranv1.CNFFunction{
		nephoranv1.CNFFunctionODU, nephoranv1.CNFFunctionOCUCP, nephoranv1.CNFFunctionOCUUP,
		nephoranv1.CNFFunctionNearRTRIC, nephoranv1.CNFFunctionNonRTRIC,
	}

	for _, coreFunc := range coreTypes {
		if function == coreFunc {
			return nephoranv1.CNF5GCore
		}
	}

	for _, oranFunc := range oranTypes {
		if function == oranFunc {
			return nephoranv1.CNFORAN
		}
	}

	return nephoranv1.CNFCustom
}

func (p *CNFIntentProcessor) getDefaultReplicas(function nephoranv1.CNFFunction) *int32 {
	defaultReplicas := map[nephoranv1.CNFFunction]int32{
		nephoranv1.CNFFunctionAMF:  3,
		nephoranv1.CNFFunctionSMF:  3,
		nephoranv1.CNFFunctionUPF:  2,
		nephoranv1.CNFFunctionNRF:  2,
		nephoranv1.CNFFunctionAUSF: 2,
		nephoranv1.CNFFunctionUDM:  2,
	}

	if replicas, exists := defaultReplicas[function]; exists {
		return &replicas
	}

	defaultVal := int32(1)
	return &defaultVal
}

func (p *CNFIntentProcessor) getDefaultResources(function nephoranv1.CNFFunction) *nephoranv1.CNFResourceIntent {
	// Get default resources from knowledge base if available
	if p.KnowledgeBase != nil && p.KnowledgeBase.ResourceRequirements != nil {
		if profile, exists := p.KnowledgeBase.ResourceRequirements[function]; exists {
			return &nephoranv1.CNFResourceIntent{
				CPU:             parseQuantity(profile.BaselineResources.CPU),
				Memory:          parseQuantity(profile.BaselineResources.Memory),
				Storage:         parseQuantity(profile.BaselineResources.Storage),
				PerformanceTier: "medium",
			}
		}
	}

	// Fallback to hardcoded defaults
	defaults := map[nephoranv1.CNFFunction]nephoranv1.CNFResourceIntent{
		nephoranv1.CNFFunctionAMF: {
			CPU:             parseQuantity("1000m"),
			Memory:          parseQuantity("2Gi"),
			Storage:         parseQuantity("10Gi"),
			PerformanceTier: "medium",
		},
		nephoranv1.CNFFunctionSMF: {
			CPU:             parseQuantity("1000m"),
			Memory:          parseQuantity("2Gi"),
			Storage:         parseQuantity("10Gi"),
			PerformanceTier: "medium",
		},
		nephoranv1.CNFFunctionUPF: {
			CPU:             parseQuantity("2000m"),
			Memory:          parseQuantity("4Gi"),
			Storage:         parseQuantity("20Gi"),
			PerformanceTier: "high",
			DPDK:            &nephoranv1.DPDKIntent{Enabled: true, Cores: &[]int32{4}[0]},
		},
	}

	if defaultResource, exists := defaults[function]; exists {
		return &defaultResource
	}

	// Generic default
	return &nephoranv1.CNFResourceIntent{
		CPU:             parseQuantity("500m"),
		Memory:          parseQuantity("1Gi"),
		Storage:         parseQuantity("5Gi"),
		PerformanceTier: "medium",
	}
}

func parseQuantity(s string) *resource.Quantity {
	if q, err := resource.ParseQuantity(s); err == nil {
		return &q
	}
	return nil
}

// Additional parsing methods would be implemented here
func (p *CNFIntentProcessor) parseResourceRequirements(resources map[string]interface{}, function nephoranv1.CNFFunction) *nephoranv1.CNFResourceIntent {
	// Implementation for parsing resource requirements from LLM response
	return p.getDefaultResources(function)
}

func (p *CNFIntentProcessor) parseAutoScalingRequirements(scaling map[string]interface{}) *nephoranv1.AutoScalingIntent {
	// Implementation for parsing auto-scaling requirements
	return &nephoranv1.AutoScalingIntent{
		Enabled:                 true,
		MinReplicas:             &[]int32{1}[0],
		MaxReplicas:             &[]int32{10}[0],
		TargetCPUUtilization:    &[]int32{70}[0],
		TargetMemoryUtilization: &[]int32{80}[0],
		ScalingPolicy:           "moderate",
	}
}

func (p *CNFIntentProcessor) parseServiceMeshRequirements(serviceMesh map[string]interface{}) *nephoranv1.ServiceMeshIntent {
	// Implementation for parsing service mesh requirements
	return &nephoranv1.ServiceMeshIntent{
		Enabled: true,
		Type:    "istio",
		MTLS:    &nephoranv1.MTLSIntent{Enabled: true, Mode: "strict"},
	}
}

func (p *CNFIntentProcessor) parseMonitoringRequirements(monitoring map[string]interface{}) *nephoranv1.MonitoringIntent {
	// Implementation for parsing monitoring requirements
	return &nephoranv1.MonitoringIntent{
		Enabled:        true,
		LogLevel:       "info",
		TracingEnabled: true,
	}
}

func (p *CNFIntentProcessor) parseSecurityRequirements(security map[string]interface{}) *nephoranv1.SecurityIntent {
	// Implementation for parsing security requirements
	return &nephoranv1.SecurityIntent{
		Level:               "standard",
		PodSecurityStandard: "baseline",
		RBACEnabled:         true,
	}
}

func (p *CNFIntentProcessor) parseHighAvailabilityRequirements(ha map[string]interface{}) *nephoranv1.HighAvailabilityIntent {
	// Implementation for parsing HA requirements
	return &nephoranv1.HighAvailabilityIntent{
		Enabled:           true,
		AvailabilityLevel: "99.9%",
		MultiZone:         true,
		AntiAffinity:      true,
	}
}

func (p *CNFIntentProcessor) parsePerformanceRequirements(performance map[string]interface{}) *nephoranv1.PerformanceIntent {
	// Implementation for parsing performance requirements
	return &nephoranv1.PerformanceIntent{
		LatencyRequirement:    &[]int32{10}[0],
		ThroughputRequirement: &[]int32{1000}[0],
		Tier:                  "standard",
	}
}

// Additional methods for validation, estimation, and knowledge base initialization

func (p *CNFIntentProcessor) validateCNFSpecs(result *nephoranv1.CNFIntentProcessingResult) error {
	// Implementation for validating CNF specifications
	return nil
}

func (p *CNFIntentProcessor) estimateResourcesAndCosts(result *nephoranv1.CNFIntentProcessingResult) {
	// Implementation for resource and cost estimation
	totalCPU := 0.0
	totalMemory := 0.0
	totalCost := 0.0

	for _, deployment := range result.CNFDeployments {
		if deployment.Resources != nil {
			if deployment.Resources.CPU != nil {
				totalCPU += float64(deployment.Resources.CPU.MilliValue()) / 1000.0
			}
			if deployment.Resources.Memory != nil {
				totalMemory += float64(deployment.Resources.Memory.Value()) / (1024 * 1024 * 1024) // Convert to GB
			}
		}

		// Estimate cost based on function type
		if p.KnowledgeBase != nil && p.KnowledgeBase.CostModels != nil {
			if costModel, exists := p.KnowledgeBase.CostModels[deployment.Function]; exists {
				totalCost += costModel.BaseCostPerHour * 24 * 30 // Monthly estimate
			}
		}
	}

	// Convert map to RawExtension
	resourcesBytes, _ := json.Marshal(map[string]interface{}{
		"total_cpu":              totalCPU,
		"total_memory":           totalMemory,
		"estimated_monthly_cost": totalCost,
	})
	result.EstimatedResources = runtime.RawExtension{Raw: resourcesBytes}
	result.EstimatedCost = totalCost
}

func (p *CNFIntentProcessor) estimateDeploymentTimeline(result *nephoranv1.CNFIntentProcessingResult) {
	// Base deployment time per CNF (in minutes)
	baseTime := int32(5)

	// Add time based on complexity
	totalTime := baseTime * int32(len(result.CNFDeployments))

	// Add additional time for complex deployments
	for _, deployment := range result.CNFDeployments {
		if deployment.AutoScaling != nil && deployment.AutoScaling.Enabled {
			totalTime += 2
		}
		if deployment.ServiceMesh != nil && deployment.ServiceMesh.Enabled {
			totalTime += 3
		}
		if deployment.HighAvailability != nil && deployment.HighAvailability.Enabled {
			totalTime += 5
		}
	}

	result.EstimatedDeploymentTime = totalTime
}

func (p *CNFIntentProcessor) calculateConfidenceScore(intentContext *CNFIntentContext, result *nephoranv1.CNFIntentProcessingResult) float64 {
	score := 0.5 // Base score

	// Increase score based on detected functions
	if len(result.DetectedFunctions) > 0 {
		score += 0.2
	}

	// Increase score based on LLM response quality
	if intentContext.LLMResponse != nil && len(intentContext.LLMResponse.ProcessedParameters) > 0 {
		score += 0.2
	}

	// Increase score based on RAG context
	if len(intentContext.RAGContext) > 0 {
		score += 0.1
	}

	// Decrease score for warnings and errors
	score -= float64(len(result.Warnings)) * 0.05
	score -= float64(len(result.Errors)) * 0.1

	// Ensure score is within bounds
	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (p *CNFIntentProcessor) initializeKnowledgeBase() {
	// Initialize with basic knowledge base
	p.KnowledgeBase = &CNFKnowledgeBase{
		FunctionMappings:     make(map[string]nephoranv1.CNFFunction),
		ResourceRequirements: make(map[nephoranv1.CNFFunction]ResourceProfile),
		CostModels:           make(map[nephoranv1.CNFFunction]CostModel),
	}

	// Initialize function mappings
	functionMappings := map[string]nephoranv1.CNFFunction{
		"amf":                nephoranv1.CNFFunctionAMF,
		"access mobility":    nephoranv1.CNFFunctionAMF,
		"smf":                nephoranv1.CNFFunctionSMF,
		"session management": nephoranv1.CNFFunctionSMF,
		"upf":                nephoranv1.CNFFunctionUPF,
		"user plane":         nephoranv1.CNFFunctionUPF,
		"nrf":                nephoranv1.CNFFunctionNRF,
		"repository":         nephoranv1.CNFFunctionNRF,
	}
	p.KnowledgeBase.FunctionMappings = functionMappings

	// Initialize resource profiles
	p.initializeResourceProfiles()

	// Initialize cost models
	p.initializeCostModels()
}

func (p *CNFIntentProcessor) initializeResourceProfiles() {
	// AMF resource profile
	p.KnowledgeBase.ResourceRequirements[nephoranv1.CNFFunctionAMF] = ResourceProfile{
		BaselineResources: CNFResourceRequirements{
			CPU:              "1000m",
			Memory:           "2Gi",
			Storage:          "10Gi",
			NetworkBandwidth: "1Gbps",
		},
		ScalingFactors: map[string]float64{
			"cpu":    1.5,
			"memory": 1.2,
		},
		PerformanceTiers: map[string]CNFResourceRequirements{
			"low": {
				CPU:     "500m",
				Memory:  "1Gi",
				Storage: "5Gi",
			},
			"high": {
				CPU:     "2000m",
				Memory:  "4Gi",
				Storage: "20Gi",
			},
		},
	}

	// UPF resource profile (more intensive)
	p.KnowledgeBase.ResourceRequirements[nephoranv1.CNFFunctionUPF] = ResourceProfile{
		BaselineResources: CNFResourceRequirements{
			CPU:              "2000m",
			Memory:           "4Gi",
			Storage:          "20Gi",
			NetworkBandwidth: "10Gbps",
			DPDK: &DPDKRequirements{
				Enabled: true,
				Cores:   4,
				Memory:  2048,
				Driver:  "vfio-pci",
			},
		},
		ScalingFactors: map[string]float64{
			"cpu":    2.0,
			"memory": 1.8,
		},
	}
}

func (p *CNFIntentProcessor) initializeCostModels() {
	// Basic cost models
	p.KnowledgeBase.CostModels[nephoranv1.CNFFunctionAMF] = CostModel{
		BaseCostPerHour: 0.50,
		ResourceMultipliers: map[string]float64{
			"cpu":    0.1,
			"memory": 0.05,
		},
		ScalingCostFactor: 1.2,
	}

	p.KnowledgeBase.CostModels[nephoranv1.CNFFunctionUPF] = CostModel{
		BaseCostPerHour: 1.00,
		ResourceMultipliers: map[string]float64{
			"cpu":    0.15,
			"memory": 0.08,
		},
		ScalingCostFactor: 1.5,
	}
}
