# Prompt Engineering Guide for Nephoran Intent Operator

## Overview

This guide provides comprehensive documentation for the enhanced LLM prompt system in the Nephoran Intent Operator, focusing on telecom domain expertise, dynamic context injection, and budget control mechanisms.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Telecom Domain Prompts](#telecom-domain-prompts)
3. [Dynamic Context Injection](#dynamic-context-injection)
4. [Token Budget Management](#token-budget-management)
5. [A/B Testing Framework](#ab-testing-framework)
6. [Quality Metrics](#quality-metrics)
7. [Best Practices](#best-practices)
8. [Usage Examples](#usage-examples)

## Architecture Overview

The enhanced prompt system consists of several key components:

### 1. Telecom Prompt Templates (`telecom_prompt_templates.go`)

Provides specialized prompt templates for different telecom domains:
- **O-RAN Network Functions**: Near-RT RIC, xApps, O-CU/DU/RU
- **5G Core Functions**: AMF, SMF, UPF, PCF, etc.
- **Network Slicing**: eMBB, URLLC, mMTC configurations
- **RAN Optimization**: Coverage, capacity, and quality optimization

### 2. Dynamic Context Manager (`dynamic_context_manager.go`)

Intelligently manages context injection with:
- **Token Budget Control**: Ensures prompts stay within model limits
- **Multi-tier Strategy**: Minimal, standard, and comprehensive context levels
- **Compression**: Reduces token usage while maintaining quality
- **Network State Integration**: Includes real-time network information

### 3. Evaluation Framework

Python scripts for comprehensive prompt evaluation:
- **A/B Testing** (`prompt_ab_testing.py`): Compare prompt variants
- **Quality Metrics** (`prompt_evaluation_metrics.py`): Telecom-specific evaluation

## Telecom Domain Prompts

### O-RAN Prompts

```go
// Example: O-RAN Network Function Deployment
template := &TelecomPromptTemplate{
    ID: "oran-nf-deployment",
    Domain: "O-RAN",
    SystemPrompt: `You are an O-RAN architecture expert...`,
    ComplianceStandards: []string{"O-RAN.WG1", "O-RAN.WG2", ...},
}
```

Key features:
- Comprehensive O-RAN component knowledge
- Interface specifications (A1, E2, O1, O2, R1)
- Multi-vendor interoperability focus
- xApp/rApp deployment patterns

### 5G Core Prompts

Specialized for Service Based Architecture (SBA):
- Network function expertise (AMF, SMF, UPF, etc.)
- SBI interface configuration
- Network slicing integration
- 3GPP standards compliance (R15, R16, R17)

### Network Slicing Prompts

Focus on:
- S-NSSAI configuration (SST and SD)
- Slice types (eMBB, URLLC, mMTC, V2X)
- Resource isolation strategies
- QoS parameter mapping

## Dynamic Context Injection

### Context Tiers

1. **Minimal Tier** (2000 tokens)
   - Essential information only
   - Heavy compression
   - No examples
   - Quick responses

2. **Standard Tier** (4000 tokens)
   - Balanced information
   - Moderate compression
   - Selected examples
   - Good quality/speed ratio

3. **Comprehensive Tier** (8000 tokens)
   - Detailed context
   - Light compression
   - Multiple examples
   - Highest quality

### Context Selection Algorithm

```go
func (dcm *DynamicContextManager) selectContextTier(request *ContextInjectionRequest) *ContextTier {
    // Analyze intent complexity
    complexity := dcm.analyzeIntentComplexity(request)
    
    // Consider token budget
    if request.MaxTokenBudget > 0 {
        // Select tier that fits budget
    }
    
    // Adaptive selection based on:
    // - Intent complexity
    // - Network state
    // - Compliance requirements
}
```

### Network State Integration

Includes real-time network information:
- Active network slices
- Deployed network functions
- Current load and performance
- Active alarms and issues

## Token Budget Management

### Budget Calculation

```go
type TokenBudget struct {
    MaxContextTokens     int  // Total available
    OptimalContextTokens int  // Recommended
    SystemPromptTokens   int  // System prompt usage
    UserPromptTokens     int  // User prompt usage
    ContextTokens        int  // Context usage
    ExampleTokens        int  // Examples usage
}
```

### Optimization Strategies

1. **Compression Levels**
   - Light: Remove whitespace, formatting
   - Moderate: Remove redundancy, structure
   - Heavy: Summarize, extract key points

2. **Document Prioritization**
   - Relevance scoring
   - Diversity constraints
   - Source authority weighting

3. **Dynamic Allocation**
   - Adjust based on intent complexity
   - Reserve tokens for response
   - Balance quality vs. efficiency

## A/B Testing Framework

### Test Configuration

```python
# Create prompt variants
variants = [
    PromptVariant(
        id="baseline",
        type=PromptVariantType.BASELINE,
        system_prompt="...",
    ),
    PromptVariant(
        id="technical_enhanced",
        type=PromptVariantType.TECHNICAL,
        system_prompt="Enhanced telecom expert...",
    ),
]

# Run A/B test
results = await test_runner.run_ab_test(
    variants=variants,
    test_cases=test_cases,
    iterations=10
)
```

### Test Cases

Comprehensive test cases covering:
- O-RAN deployments
- 5G Core configurations
- Network slice setups
- Edge computing scenarios

### Result Analysis

- Performance comparison
- Statistical significance
- Winner determination
- Weakness identification

## Quality Metrics

### Telecom-Specific Metrics

```python
@dataclass
class TelecomMetrics:
    # O-RAN Metrics
    oran_interface_compliance: float
    ric_configuration_validity: float
    xapp_compatibility: float
    
    # 5G Core Metrics
    nf_type_accuracy: float
    sbi_interface_validity: float
    slice_configuration_validity: float
    
    # Compliance Metrics
    standards_compliance_score: float
    security_compliance_score: float
```

### Evaluation Process

1. **Technical Accuracy**
   - Interface correctness
   - Protocol accuracy
   - Configuration completeness

2. **Domain Compliance**
   - O-RAN standards
   - 3GPP compliance
   - Security requirements

3. **Operational Readiness**
   - High availability
   - Scalability
   - Performance optimization

## Best Practices

### 1. Prompt Design

**DO:**
- Include specific domain expertise
- Specify output format clearly
- Add relevant examples
- Set clear constraints
- Use consistent terminology

**DON'T:**
- Use vague instructions
- Omit compliance requirements
- Ignore token limits
- Mix terminology
- Skip validation

### 2. Context Management

**Effective Context Injection:**
```go
// Good: Prioritized and compressed
context := dcm.gatherAndInjectContext(
    ctx, request, tier, maxTokens,
)

// Include network state when relevant
if request.NetworkState != nil {
    networkContext := dcm.buildNetworkStateContext(
        request.NetworkState, modelName,
    )
}
```

### 3. Token Optimization

**Strategies:**
- Pre-calculate token usage
- Implement caching
- Use compression wisely
- Batch similar requests
- Monitor usage patterns

### 4. Quality Assurance

**Continuous Improvement:**
1. Regular A/B testing
2. Metric monitoring
3. User feedback integration
4. Standards updates
5. Performance optimization

## Usage Examples

### Example 1: O-RAN Deployment

```go
// Create request
request := &ContextInjectionRequest{
    Intent: "Deploy Near-RT RIC with xApp support for traffic optimization",
    IntentType: "NetworkFunctionDeployment",
    Domain: "O-RAN",
    ModelName: "gpt-4o",
    NetworkState: &NetworkStateContext{
        ActiveSlices: []NetworkSliceInfo{...},
        NetworkFunctions: []NetworkFunctionInfo{...},
    },
}

// Inject context
result, err := contextManager.InjectContext(ctx, request)

// Use the optimized prompt
response := llmClient.Generate(result.FinalPrompt)
```

### Example 2: 5G Core with Slicing

```go
// Template selection
template := promptRegistry.GetTemplate("5g-core-nf")

// Build prompt with variables
prompt, err := promptRegistry.BuildPrompt(template.ID, map[string]interface{}{
    "intent": "Deploy SMF with URLLC support",
    "slice_requirements": []string{"SST=2", "latency<1ms"},
    "qos_parameters": map[string]interface{}{
        "5qi": 82,
        "priority": 1,
    },
})
```

### Example 3: A/B Testing

```python
# Define test case
test_case = TestCase(
    id="tc_oran_001",
    intent="Deploy Near-RT RIC with xApp support",
    domain="O-RAN",
    expected_fields=["metadata.name", "spec.replicas"],
    complexity="complex"
)

# Run evaluation
metrics = evaluator.evaluate_response(
    response=llm_response,
    intent_type=test_case.intent_type,
    domain=test_case.domain
)

# Analyze results
print(f"Overall Score: {metrics.overall_score:.3f}")
print(f"O-RAN Compliance: {metrics.oran_interface_compliance:.3f}")
```

## Configuration

Key configuration in `prompt_optimization_config.yaml`:

```yaml
prompt_optimization:
  token_budgets:
    minimal:
      max_tokens: 2000
    standard:
      max_tokens: 4000
    comprehensive:
      max_tokens: 8000
      
  domain_priorities:
    O-RAN:
      weight: 1.0
    5G-Core:
      weight: 0.95
      
  quality_thresholds:
    technical_accuracy: 0.8
    compliance_score: 0.75
```

## Monitoring and Maintenance

### Key Metrics to Track

1. **Performance Metrics**
   - Response time (p50, p95, p99)
   - Token usage efficiency
   - Cache hit rate
   - Error rate

2. **Quality Metrics**
   - Technical accuracy scores
   - Compliance scores
   - User satisfaction
   - Intent success rate

3. **Cost Metrics**
   - Tokens per request
   - Cost per operation
   - Budget utilization

### Maintenance Tasks

**Weekly:**
- Review performance metrics
- Analyze failed requests
- Update prompt templates

**Monthly:**
- Run comprehensive A/B tests
- Update domain knowledge
- Review compliance standards

**Quarterly:**
- Major prompt optimizations
- Model upgrades evaluation
- Architecture improvements

## Troubleshooting

### Common Issues

1. **Token Budget Exceeded**
   - Check context tier selection
   - Review compression settings
   - Analyze document selection

2. **Low Quality Scores**
   - Review prompt templates
   - Check domain knowledge
   - Validate examples

3. **Poor Performance**
   - Check cache configuration
   - Review parallel processing
   - Optimize context building

### Debug Tools

```go
// Enable detailed logging
logger := slog.Default().With("component", "prompt-system")

// Get metrics
metrics := contextManager.GetMetrics()

// Analyze prompt performance
analysis := promptRegistry.AnalyzePromptPerformance(promptID)
```

## Future Enhancements

1. **Advanced Compression**
   - LLM-based summarization
   - Semantic compression
   - Domain-specific encoding

2. **Multi-Model Support**
   - Model-specific optimizations
   - Automatic model selection
   - Ensemble approaches

3. **Real-time Learning**
   - Online prompt optimization
   - Feedback integration
   - Adaptive context selection

4. **Enhanced Compliance**
   - Automated standards tracking
   - Compliance validation
   - Audit trail generation

## Conclusion

The enhanced prompt system provides a robust foundation for telecom-specific LLM operations in the Nephoran Intent Operator. By following these guidelines and best practices, you can achieve high-quality, compliant, and efficient network configurations through natural language intents.

For questions or contributions, please refer to the main project documentation and contribution guidelines.