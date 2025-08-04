#!/usr/bin/env python3
"""
Prompt A/B Testing Framework for Nephoran Intent Operator

This script provides a comprehensive framework for testing and evaluating
different prompt variations in the telecom domain.
"""

import json
import time
import asyncio
import logging
import statistics
from datetime import datetime
from typing import Dict, List, Any, Optional, Tuple
from dataclasses import dataclass, field, asdict
from enum import Enum
import random
import hashlib
from collections import defaultdict
import numpy as np
from concurrent.futures import ThreadPoolExecutor, as_completed
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
from pathlib import Path
import yaml
import re

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class PromptVariantType(Enum):
    """Types of prompt variants for testing"""
    BASELINE = "baseline"
    ENHANCED = "enhanced"
    COMPRESSED = "compressed"
    TECHNICAL = "technical"
    CONVERSATIONAL = "conversational"
    STRUCTURED = "structured"
    MINIMAL = "minimal"


@dataclass
class PromptVariant:
    """Represents a prompt variant for A/B testing"""
    id: str
    name: str
    type: PromptVariantType
    system_prompt: str
    user_prompt_template: str
    examples: List[Dict[str, str]] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)
    token_estimate: int = 0
    
    def generate_prompt(self, intent: str, context: Dict[str, Any]) -> str:
        """Generate a complete prompt with the given intent and context"""
        user_prompt = self.user_prompt_template.format(
            intent=intent,
            **context
        )
        
        full_prompt = f"{self.system_prompt}\n\n{user_prompt}"
        
        if self.examples:
            examples_text = "\n\n## Examples:\n"
            for i, example in enumerate(self.examples[:2]):  # Limit to 2 examples
                examples_text += f"\n### Example {i+1}:\n"
                examples_text += f"Intent: {example['intent']}\n"
                examples_text += f"Response: {example['response']}\n"
            full_prompt += examples_text
        
        return full_prompt


@dataclass
class TestCase:
    """Represents a test case for prompt evaluation"""
    id: str
    intent: str
    intent_type: str
    domain: str
    expected_fields: List[str]
    validation_rules: Dict[str, Any]
    context: Dict[str, Any] = field(default_factory=dict)
    ground_truth: Optional[Dict[str, Any]] = None
    complexity: str = "moderate"  # simple, moderate, complex
    tags: List[str] = field(default_factory=list)


@dataclass
class EvaluationMetrics:
    """Metrics for evaluating prompt performance"""
    response_time: float
    token_count: int
    technical_accuracy: float
    compliance_score: float
    completeness_score: float
    format_validity: bool
    error_rate: float
    latency_p50: float
    latency_p95: float
    latency_p99: float
    cost_estimate: float
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)
    
    @property
    def overall_score(self) -> float:
        """Calculate overall quality score"""
        weights = {
            'technical_accuracy': 0.35,
            'compliance_score': 0.25,
            'completeness_score': 0.20,
            'format_validity': 0.10,
            'error_rate': 0.10  # Lower is better
        }
        
        score = (
            weights['technical_accuracy'] * self.technical_accuracy +
            weights['compliance_score'] * self.compliance_score +
            weights['completeness_score'] * self.completeness_score +
            weights['format_validity'] * (1.0 if self.format_validity else 0.0) +
            weights['error_rate'] * (1.0 - self.error_rate)
        )
        
        return score


class TelecomDomainValidator:
    """Validates responses for telecom domain compliance"""
    
    def __init__(self):
        self.oran_interfaces = ["A1", "E2", "O1", "O2", "R1"]
        self.network_functions = ["AMF", "SMF", "UPF", "PCF", "UDM", "UDR", "AUSF", "NRF", "NSSF", "NEF"]
        self.slice_types = {"eMBB": 1, "URLLC": 2, "mMTC": 3, "V2X": 4}
        self.standards = ["3GPP", "O-RAN", "ETSI", "IETF"]
        
    def validate_technical_accuracy(self, response: Dict[str, Any], test_case: TestCase) -> float:
        """Validate technical accuracy of the response"""
        score = 0.0
        checks = 0
        
        # Check for required fields
        for field in test_case.expected_fields:
            checks += 1
            if self._field_exists(response, field):
                score += 1.0
        
        # Domain-specific validation
        if test_case.domain == "O-RAN":
            score += self._validate_oran_response(response)
            checks += 1
        elif test_case.domain == "5G-Core":
            score += self._validate_5g_core_response(response)
            checks += 1
        elif test_case.domain == "Network-Slicing":
            score += self._validate_network_slice_response(response)
            checks += 1
        
        return score / checks if checks > 0 else 0.0
    
    def _field_exists(self, obj: Dict[str, Any], field_path: str) -> bool:
        """Check if a field exists in nested dictionary"""
        parts = field_path.split('.')
        current = obj
        
        for part in parts:
            if isinstance(current, dict) and part in current:
                current = current[part]
            else:
                return False
        return True
    
    def _validate_oran_response(self, response: Dict[str, Any]) -> float:
        """Validate O-RAN specific requirements"""
        score = 0.0
        checks = 0
        
        # Check for O-RAN interfaces
        if 'spec' in response and 'template' in response['spec']:
            containers = response['spec']['template'].get('spec', {}).get('containers', [])
            if containers:
                ports = containers[0].get('ports', [])
                interface_found = False
                for port in ports:
                    if port.get('name') in ['e2ap', 'a1-rest', 'o1-netconf']:
                        interface_found = True
                        break
                if interface_found:
                    score += 1.0
                checks += 1
        
        # Check for O-RAN specific labels
        labels = response.get('metadata', {}).get('labels', {})
        if any('oran' in k.lower() for k in labels.keys()):
            score += 1.0
        checks += 1
        
        return score / checks if checks > 0 else 0.0
    
    def _validate_5g_core_response(self, response: Dict[str, Any]) -> float:
        """Validate 5G Core specific requirements"""
        score = 0.0
        checks = 0
        
        # Check for valid network function type
        labels = response.get('metadata', {}).get('labels', {})
        nf_type = labels.get('nf.5gc.3gpp.org/type', '')
        if nf_type.upper() in self.network_functions:
            score += 1.0
        checks += 1
        
        # Check for SBI interface
        if 'spec' in response and 'template' in response['spec']:
            containers = response['spec']['template'].get('spec', {}).get('containers', [])
            if containers:
                ports = containers[0].get('ports', [])
                sbi_found = any(p.get('name') == 'sbi' for p in ports)
                if sbi_found:
                    score += 1.0
                checks += 1
        
        return score / checks if checks > 0 else 0.0
    
    def _validate_network_slice_response(self, response: Dict[str, Any]) -> float:
        """Validate network slice configuration"""
        score = 0.0
        checks = 0
        
        # Check for S-NSSAI
        if 'network_slice' in response:
            slice_config = response['network_slice']
            if 'sst' in slice_config and isinstance(slice_config['sst'], (int, str)):
                try:
                    sst = int(slice_config['sst'])
                    if 1 <= sst <= 255:  # Valid SST range
                        score += 1.0
                except:
                    pass
            checks += 1
            
            # Check for slice differentiator
            if 'sd' in slice_config:
                score += 0.5
            checks += 0.5
        
        return score / checks if checks > 0 else 0.0
    
    def validate_compliance(self, response: Dict[str, Any], test_case: TestCase) -> float:
        """Validate compliance with standards"""
        score = 0.0
        checks = 0
        
        # Check for required compliance annotations
        annotations = response.get('metadata', {}).get('annotations', {})
        
        # Check for standard references
        response_str = json.dumps(response)
        for standard in self.standards:
            if standard in response_str:
                score += 0.25
                checks += 0.25
        
        # Check for version compliance
        if test_case.domain == "O-RAN":
            if 'g-release' in response_str.lower() or 'f-release' in response_str.lower():
                score += 0.5
            checks += 0.5
        
        return score / checks if checks > 0 else 0.0


class PromptABTestRunner:
    """Runs A/B tests for prompt variants"""
    
    def __init__(self, llm_client: Any, config: Dict[str, Any]):
        self.llm_client = llm_client
        self.config = config
        self.validator = TelecomDomainValidator()
        self.results = defaultdict(list)
        
    async def run_test(self, variant: PromptVariant, test_case: TestCase) -> Dict[str, Any]:
        """Run a single test with a prompt variant"""
        start_time = time.time()
        
        try:
            # Generate prompt
            prompt = variant.generate_prompt(test_case.intent, test_case.context)
            
            # Call LLM (mock or real)
            response = await self._call_llm(prompt, variant.metadata.get('model', 'gpt-4'))
            
            # Parse response
            parsed_response = self._parse_response(response)
            
            # Calculate metrics
            response_time = time.time() - start_time
            metrics = self._evaluate_response(parsed_response, test_case, response_time)
            
            result = {
                'variant_id': variant.id,
                'test_case_id': test_case.id,
                'response': parsed_response,
                'metrics': metrics,
                'success': metrics.format_validity,
                'timestamp': datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error(f"Test failed for variant {variant.id} on test case {test_case.id}: {e}")
            result = {
                'variant_id': variant.id,
                'test_case_id': test_case.id,
                'error': str(e),
                'success': False,
                'timestamp': datetime.now().isoformat()
            }
        
        return result
    
    async def _call_llm(self, prompt: str, model: str) -> str:
        """Call LLM with prompt (mock implementation)"""
        # In production, this would call actual LLM API
        await asyncio.sleep(random.uniform(0.1, 0.5))  # Simulate API latency
        
        # Return mock response based on prompt content
        if "O-RAN" in prompt:
            return json.dumps({
                "apiVersion": "apps/v1",
                "kind": "Deployment",
                "metadata": {
                    "name": "near-rt-ric",
                    "namespace": "oran-system",
                    "labels": {
                        "oran.component": "ric"
                    }
                },
                "spec": {
                    "replicas": 3,
                    "template": {
                        "spec": {
                            "containers": [{
                                "name": "near-rt-ric",
                                "image": "oran/near-rt-ric:v1.0",
                                "ports": [
                                    {"containerPort": 36421, "name": "e2ap", "protocol": "SCTP"},
                                    {"containerPort": 9090, "name": "a1-rest", "protocol": "TCP"}
                                ]
                            }]
                        }
                    }
                }
            })
        else:
            return json.dumps({
                "type": "NetworkFunctionDeployment",
                "name": "test-nf",
                "spec": {
                    "replicas": 1
                }
            })
    
    def _parse_response(self, response: str) -> Dict[str, Any]:
        """Parse LLM response"""
        try:
            return json.loads(response)
        except json.JSONDecodeError:
            # Try to extract JSON from markdown code blocks
            json_match = re.search(r'```json\s*(.*?)\s*```', response, re.DOTALL)
            if json_match:
                return json.loads(json_match.group(1))
            raise ValueError("Could not parse response as JSON")
    
    def _evaluate_response(self, response: Dict[str, Any], test_case: TestCase, response_time: float) -> EvaluationMetrics:
        """Evaluate response and calculate metrics"""
        # Validate response format
        format_validity = isinstance(response, dict) and len(response) > 0
        
        # Calculate domain-specific scores
        technical_accuracy = self.validator.validate_technical_accuracy(response, test_case)
        compliance_score = self.validator.validate_compliance(response, test_case)
        
        # Calculate completeness
        expected_fields = set(test_case.expected_fields)
        present_fields = set()
        for field in expected_fields:
            if self.validator._field_exists(response, field):
                present_fields.add(field)
        completeness_score = len(present_fields) / len(expected_fields) if expected_fields else 1.0
        
        # Estimate token count (simple estimation)
        token_count = len(json.dumps(response)) // 4
        
        # Calculate cost estimate (simplified)
        cost_estimate = token_count * 0.00002  # Rough estimate
        
        return EvaluationMetrics(
            response_time=response_time,
            token_count=token_count,
            technical_accuracy=technical_accuracy,
            compliance_score=compliance_score,
            completeness_score=completeness_score,
            format_validity=format_validity,
            error_rate=0.0 if format_validity else 1.0,
            latency_p50=response_time,
            latency_p95=response_time * 1.2,
            latency_p99=response_time * 1.5,
            cost_estimate=cost_estimate
        )
    
    async def run_ab_test(self, variants: List[PromptVariant], test_cases: List[TestCase], 
                         iterations: int = 10) -> Dict[str, Any]:
        """Run A/B test with multiple variants and test cases"""
        logger.info(f"Starting A/B test with {len(variants)} variants and {len(test_cases)} test cases")
        
        all_results = []
        
        for iteration in range(iterations):
            logger.info(f"Running iteration {iteration + 1}/{iterations}")
            
            # Create all test combinations
            tests = [(variant, test_case) for variant in variants for test_case in test_cases]
            
            # Run tests concurrently
            tasks = [self.run_test(variant, test_case) for variant, test_case in tests]
            results = await asyncio.gather(*tasks)
            
            all_results.extend(results)
        
        # Analyze results
        analysis = self._analyze_results(all_results)
        
        return {
            'raw_results': all_results,
            'analysis': analysis,
            'test_config': {
                'variants': len(variants),
                'test_cases': len(test_cases),
                'iterations': iterations,
                'total_tests': len(all_results)
            }
        }
    
    def _analyze_results(self, results: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Analyze A/B test results"""
        variant_metrics = defaultdict(lambda: defaultdict(list))
        
        for result in results:
            if 'metrics' in result and result['success']:
                variant_id = result['variant_id']
                metrics = result['metrics']
                
                for metric_name, value in metrics.to_dict().items():
                    if isinstance(value, (int, float)):
                        variant_metrics[variant_id][metric_name].append(value)
        
        # Calculate statistics for each variant
        variant_stats = {}
        for variant_id, metrics in variant_metrics.items():
            stats = {}
            for metric_name, values in metrics.items():
                if values:
                    stats[metric_name] = {
                        'mean': statistics.mean(values),
                        'median': statistics.median(values),
                        'std': statistics.stdev(values) if len(values) > 1 else 0,
                        'min': min(values),
                        'max': max(values),
                        'count': len(values)
                    }
            
            # Calculate overall scores
            overall_scores = []
            for result in results:
                if result['variant_id'] == variant_id and 'metrics' in result:
                    overall_scores.append(result['metrics'].overall_score)
            
            if overall_scores:
                stats['overall_score'] = {
                    'mean': statistics.mean(overall_scores),
                    'median': statistics.median(overall_scores),
                    'std': statistics.stdev(overall_scores) if len(overall_scores) > 1 else 0,
                    'min': min(overall_scores),
                    'max': max(overall_scores),
                    'count': len(overall_scores)
                }
            
            variant_stats[variant_id] = stats
        
        # Determine winner
        winner = self._determine_winner(variant_stats)
        
        return {
            'variant_statistics': variant_stats,
            'winner': winner,
            'summary': self._generate_summary(variant_stats)
        }
    
    def _determine_winner(self, variant_stats: Dict[str, Any]) -> Dict[str, Any]:
        """Determine the winning variant based on overall score"""
        best_variant = None
        best_score = -1
        
        for variant_id, stats in variant_stats.items():
            if 'overall_score' in stats:
                mean_score = stats['overall_score']['mean']
                if mean_score > best_score:
                    best_score = mean_score
                    best_variant = variant_id
        
        return {
            'variant_id': best_variant,
            'score': best_score,
            'margin': self._calculate_confidence_margin(variant_stats, best_variant)
        }
    
    def _calculate_confidence_margin(self, variant_stats: Dict[str, Any], winner_id: str) -> float:
        """Calculate confidence margin for the winner"""
        if winner_id not in variant_stats or 'overall_score' not in variant_stats[winner_id]:
            return 0.0
        
        winner_score = variant_stats[winner_id]['overall_score']['mean']
        margins = []
        
        for variant_id, stats in variant_stats.items():
            if variant_id != winner_id and 'overall_score' in stats:
                other_score = stats['overall_score']['mean']
                margins.append(winner_score - other_score)
        
        return min(margins) if margins else 0.0
    
    def _generate_summary(self, variant_stats: Dict[str, Any]) -> Dict[str, Any]:
        """Generate summary of A/B test results"""
        summary = {
            'total_variants': len(variant_stats),
            'key_findings': [],
            'recommendations': []
        }
        
        # Analyze key metrics
        for variant_id, stats in variant_stats.items():
            if 'overall_score' in stats:
                score = stats['overall_score']['mean']
                if score > 0.8:
                    summary['key_findings'].append(
                        f"Variant {variant_id} shows excellent performance with score {score:.3f}"
                    )
                elif score < 0.5:
                    summary['key_findings'].append(
                        f"Variant {variant_id} underperforms with score {score:.3f}"
                    )
        
        # Add recommendations
        best_latency = float('inf')
        best_latency_variant = None
        
        for variant_id, stats in variant_stats.items():
            if 'response_time' in stats:
                latency = stats['response_time']['mean']
                if latency < best_latency:
                    best_latency = latency
                    best_latency_variant = variant_id
        
        if best_latency_variant:
            summary['recommendations'].append(
                f"Use variant {best_latency_variant} for latency-critical applications"
            )
        
        return summary


class PromptOptimizer:
    """Optimizes prompts based on A/B test results"""
    
    def __init__(self, test_runner: PromptABTestRunner):
        self.test_runner = test_runner
        
    def generate_optimized_prompt(self, base_variant: PromptVariant, 
                                test_results: Dict[str, Any]) -> PromptVariant:
        """Generate an optimized prompt based on test results"""
        # Analyze what worked well
        analysis = test_results['analysis']
        winner_stats = analysis['variant_statistics'].get(analysis['winner']['variant_id'], {})
        
        # Create optimized variant
        optimized = PromptVariant(
            id=f"{base_variant.id}_optimized",
            name=f"{base_variant.name} (Optimized)",
            type=base_variant.type,
            system_prompt=self._optimize_system_prompt(base_variant.system_prompt, winner_stats),
            user_prompt_template=base_variant.user_prompt_template,
            examples=base_variant.examples,
            metadata={**base_variant.metadata, 'optimized': True}
        )
        
        return optimized
    
    def _optimize_system_prompt(self, prompt: str, stats: Dict[str, Any]) -> str:
        """Optimize system prompt based on performance statistics"""
        # This is a simplified optimization - in production, you'd use more sophisticated methods
        
        # If technical accuracy is low, add more technical details
        if 'technical_accuracy' in stats and stats['technical_accuracy']['mean'] < 0.7:
            prompt += "\n\nIMPORTANT: Ensure all technical details are accurate and complete."
        
        # If compliance score is low, emphasize standards
        if 'compliance_score' in stats and stats['compliance_score']['mean'] < 0.7:
            prompt += "\n\nIMPORTANT: Strictly follow 3GPP and O-RAN standards."
        
        return prompt


def create_telecom_test_cases() -> List[TestCase]:
    """Create comprehensive test cases for telecom domain"""
    test_cases = [
        TestCase(
            id="tc_oran_001",
            intent="Deploy Near-RT RIC with xApp support for traffic optimization",
            intent_type="NetworkFunctionDeployment",
            domain="O-RAN",
            expected_fields=[
                "metadata.name",
                "metadata.namespace",
                "spec.replicas",
                "spec.template.spec.containers"
            ],
            validation_rules={
                "min_replicas": 2,
                "required_ports": ["e2ap", "a1-rest"],
                "required_labels": ["oran.component"]
            },
            complexity="complex",
            tags=["o-ran", "ric", "xapp"]
        ),
        TestCase(
            id="tc_5gc_001",
            intent="Deploy SMF with URLLC support for industrial IoT",
            intent_type="NetworkFunctionDeployment",
            domain="5G-Core",
            expected_fields=[
                "metadata.name",
                "metadata.labels",
                "spec.replicas",
                "spec.template.spec.containers"
            ],
            validation_rules={
                "nf_type": "smf",
                "slice_support": ["URLLC"],
                "latency_target": "1ms"
            },
            complexity="complex",
            tags=["5g-core", "smf", "urllc"]
        ),
        TestCase(
            id="tc_slice_001",
            intent="Configure eMBB network slice for video streaming",
            intent_type="NetworkSliceConfiguration",
            domain="Network-Slicing",
            expected_fields=[
                "network_slice.sst",
                "network_slice.sd",
                "qos_parameters"
            ],
            validation_rules={
                "sst": 1,  # eMBB
                "min_bandwidth": "100Mbps"
            },
            complexity="moderate",
            tags=["network-slice", "embb", "video"]
        )
    ]
    
    return test_cases


def create_prompt_variants() -> List[PromptVariant]:
    """Create different prompt variants for testing"""
    variants = []
    
    # Baseline variant
    baseline = PromptVariant(
        id="baseline",
        name="Baseline Prompt",
        type=PromptVariantType.BASELINE,
        system_prompt="""You are a telecom network engineer. 
Generate Kubernetes configurations for network functions based on user requirements.""",
        user_prompt_template="Generate configuration for: {intent}",
        metadata={"model": "gpt-4"}
    )
    variants.append(baseline)
    
    # Enhanced technical variant
    technical = PromptVariant(
        id="technical_enhanced",
        name="Technical Enhanced",
        type=PromptVariantType.TECHNICAL,
        system_prompt="""You are an expert telecommunications engineer with deep knowledge of:
- O-RAN Architecture (Near-RT RIC, Non-RT RIC, O-CU, O-DU, O-RU)
- 5G Core Network Functions (AMF, SMF, UPF, PCF, etc.)
- Network Slicing (eMBB, URLLC, mMTC)
- 3GPP Standards (R15, R16, R17)

When generating configurations:
1. Ensure technical accuracy
2. Follow industry standards
3. Include proper resource allocation
4. Consider high availability
5. Add relevant annotations and labels""",
        user_prompt_template="""Intent: {intent}

Generate a production-ready Kubernetes configuration that includes:
- Proper namespace and labels
- Resource requests and limits
- Health checks
- Security context
- Network policies if needed""",
        metadata={"model": "gpt-4", "temperature": 0.3}
    )
    variants.append(technical)
    
    # Compressed variant
    compressed = PromptVariant(
        id="compressed",
        name="Compressed Prompt",
        type=PromptVariantType.COMPRESSED,
        system_prompt="""Telecom engineer. Generate K8s configs for network functions.
Focus: O-RAN, 5GC, slicing. Follow 3GPP/O-RAN standards.""",
        user_prompt_template="Config for: {intent}. Include: namespace, replicas, resources, ports.",
        metadata={"model": "gpt-4", "max_tokens": 1000}
    )
    variants.append(compressed)
    
    # Structured variant with examples
    structured = PromptVariant(
        id="structured",
        name="Structured with Examples",
        type=PromptVariantType.STRUCTURED,
        system_prompt="""You are a telecom network engineer specializing in cloud-native deployments.

Your expertise includes:
- O-RAN components and interfaces
- 5G Core network functions
- Kubernetes best practices
- Network slicing configuration

Always output valid JSON/YAML configurations.""",
        user_prompt_template="""Task: {intent}

Requirements:
- Follow Kubernetes best practices
- Include proper labels and annotations
- Set appropriate resource limits
- Configure health checks

Output format: Kubernetes manifest (JSON or YAML)""",
        examples=[
            {
                "intent": "Deploy Near-RT RIC",
                "response": """{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "near-rt-ric",
    "namespace": "oran"
  }
}"""
            }
        ],
        metadata={"model": "gpt-4", "temperature": 0.5}
    )
    variants.append(structured)
    
    return variants


def visualize_results(results: Dict[str, Any], output_dir: Path):
    """Create visualizations for A/B test results"""
    output_dir.mkdir(parents=True, exist_ok=True)
    
    # Extract data for visualization
    variant_stats = results['analysis']['variant_statistics']
    
    # Create performance comparison chart
    plt.figure(figsize=(12, 8))
    
    # Metrics to compare
    metrics = ['overall_score', 'technical_accuracy', 'compliance_score', 
               'completeness_score', 'response_time']
    
    variant_names = list(variant_stats.keys())
    x = np.arange(len(metrics))
    width = 0.8 / len(variant_names)
    
    for i, variant in enumerate(variant_names):
        values = []
        for metric in metrics:
            if metric in variant_stats[variant]:
                values.append(variant_stats[variant][metric]['mean'])
            else:
                values.append(0)
        
        plt.bar(x + i * width, values, width, label=variant)
    
    plt.xlabel('Metrics')
    plt.ylabel('Score / Value')
    plt.title('Prompt Variant Performance Comparison')
    plt.xticks(x + width * (len(variant_names) - 1) / 2, metrics, rotation=45)
    plt.legend()
    plt.tight_layout()
    plt.savefig(output_dir / 'performance_comparison.png')
    plt.close()
    
    # Create latency distribution plot
    plt.figure(figsize=(10, 6))
    
    for variant in variant_names:
        if 'response_time' in variant_stats[variant]:
            latencies = [r['metrics'].response_time for r in results['raw_results'] 
                        if r['variant_id'] == variant and 'metrics' in r]
            if latencies:
                plt.hist(latencies, bins=20, alpha=0.5, label=variant)
    
    plt.xlabel('Response Time (s)')
    plt.ylabel('Frequency')
    plt.title('Response Time Distribution by Variant')
    plt.legend()
    plt.tight_layout()
    plt.savefig(output_dir / 'latency_distribution.png')
    plt.close()
    
    # Create quality score heatmap
    quality_data = []
    test_case_ids = list(set(r['test_case_id'] for r in results['raw_results'] if 'test_case_id' in r))
    
    for variant in variant_names:
        variant_scores = []
        for test_case in test_case_ids:
            scores = [r['metrics'].overall_score for r in results['raw_results']
                     if r['variant_id'] == variant and r['test_case_id'] == test_case 
                     and 'metrics' in r]
            variant_scores.append(statistics.mean(scores) if scores else 0)
        quality_data.append(variant_scores)
    
    plt.figure(figsize=(10, 8))
    sns.heatmap(quality_data, 
                xticklabels=test_case_ids, 
                yticklabels=variant_names,
                annot=True, 
                fmt='.3f',
                cmap='YlOrRd')
    plt.title('Quality Scores by Variant and Test Case')
    plt.tight_layout()
    plt.savefig(output_dir / 'quality_heatmap.png')
    plt.close()


def generate_report(results: Dict[str, Any], output_file: Path):
    """Generate comprehensive A/B test report"""
    report = {
        'test_summary': {
            'timestamp': datetime.now().isoformat(),
            'total_tests': results['test_config']['total_tests'],
            'variants_tested': results['test_config']['variants'],
            'test_cases': results['test_config']['test_cases'],
            'iterations': results['test_config']['iterations']
        },
        'winner': results['analysis']['winner'],
        'variant_performance': {},
        'recommendations': []
    }
    
    # Add variant performance details
    for variant_id, stats in results['analysis']['variant_statistics'].items():
        if 'overall_score' in stats:
            report['variant_performance'][variant_id] = {
                'overall_score': stats['overall_score']['mean'],
                'technical_accuracy': stats.get('technical_accuracy', {}).get('mean', 0),
                'compliance_score': stats.get('compliance_score', {}).get('mean', 0),
                'average_latency': stats.get('response_time', {}).get('mean', 0),
                'error_rate': stats.get('error_rate', {}).get('mean', 0)
            }
    
    # Generate recommendations
    winner_id = results['analysis']['winner']['variant_id']
    winner_score = results['analysis']['winner']['score']
    
    report['recommendations'].append(
        f"Implement {winner_id} variant for production use (score: {winner_score:.3f})"
    )
    
    # Check for specific strengths
    for variant_id, stats in results['analysis']['variant_statistics'].items():
        if 'response_time' in stats and stats['response_time']['mean'] < 0.5:
            report['recommendations'].append(
                f"Consider {variant_id} for latency-critical applications"
            )
        
        if 'technical_accuracy' in stats and stats['technical_accuracy']['mean'] > 0.9:
            report['recommendations'].append(
                f"Use {variant_id} for complex technical configurations"
            )
    
    # Save report
    with open(output_file, 'w') as f:
        json.dump(report, f, indent=2)
    
    # Also create markdown report
    markdown_file = output_file.with_suffix('.md')
    with open(markdown_file, 'w') as f:
        f.write("# Prompt A/B Testing Report\n\n")
        f.write(f"Generated: {report['test_summary']['timestamp']}\n\n")
        
        f.write("## Test Summary\n\n")
        f.write(f"- Total tests: {report['test_summary']['total_tests']}\n")
        f.write(f"- Variants tested: {report['test_summary']['variants_tested']}\n")
        f.write(f"- Test cases: {report['test_summary']['test_cases']}\n")
        f.write(f"- Iterations: {report['test_summary']['iterations']}\n\n")
        
        f.write("## Winner\n\n")
        f.write(f"**{report['winner']['variant_id']}** with score: {report['winner']['score']:.3f}\n")
        f.write(f"Confidence margin: {report['winner']['margin']:.3f}\n\n")
        
        f.write("## Variant Performance\n\n")
        f.write("| Variant | Overall Score | Technical Accuracy | Compliance | Avg Latency | Error Rate |\n")
        f.write("|---------|---------------|-------------------|------------|-------------|------------|\n")
        
        for variant_id, perf in report['variant_performance'].items():
            f.write(f"| {variant_id} | {perf['overall_score']:.3f} | "
                   f"{perf['technical_accuracy']:.3f} | {perf['compliance_score']:.3f} | "
                   f"{perf['average_latency']:.3f}s | {perf['error_rate']:.3f} |\n")
        
        f.write("\n## Recommendations\n\n")
        for rec in report['recommendations']:
            f.write(f"- {rec}\n")


async def main():
    """Main function to run A/B testing"""
    # Configure logging
    logging.basicConfig(level=logging.INFO)
    
    # Create test components
    test_runner = PromptABTestRunner(
        llm_client=None,  # Mock client for demo
        config={'timeout': 30, 'max_retries': 3}
    )
    
    # Create test cases and variants
    test_cases = create_telecom_test_cases()
    variants = create_prompt_variants()
    
    # Run A/B test
    logger.info("Starting A/B test...")
    results = await test_runner.run_ab_test(variants, test_cases, iterations=5)
    
    # Generate visualizations and report
    output_dir = Path("prompt_test_results")
    visualize_results(results, output_dir)
    generate_report(results, output_dir / "test_report.json")
    
    logger.info(f"Test completed. Results saved to {output_dir}")
    
    # Print summary
    winner = results['analysis']['winner']
    print(f"\nWinner: {winner['variant_id']} with score {winner['score']:.3f}")
    print(f"Confidence margin: {winner['margin']:.3f}")
    
    # Create optimized prompt
    optimizer = PromptOptimizer(test_runner)
    baseline_variant = next(v for v in variants if v.id == "baseline")
    optimized = optimizer.generate_optimized_prompt(baseline_variant, results)
    
    print(f"\nOptimized prompt created: {optimized.id}")


if __name__ == "__main__":
    asyncio.run(main())