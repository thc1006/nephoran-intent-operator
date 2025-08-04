#!/usr/bin/env python3
"""
A/B Testing Framework for Prompt Optimization in Nephoran Intent Operator

This script provides comprehensive A/B testing capabilities for prompt variants,
with statistical analysis and automated winner selection.
"""

import argparse
import asyncio
import json
import logging
import random
import statistics
import time
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, List, Optional, Tuple, Any
import yaml

import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
import pandas as pd
from scipy import stats
import aiohttp
import asyncio


@dataclass
class PromptVariant:
    """Represents a prompt variant for A/B testing"""
    name: str
    description: str
    system_prompt: str
    user_template: str
    examples: List[Dict[str, str]]
    parameters: Dict[str, Any]


@dataclass
class TestCase:
    """Represents a test case for evaluation"""
    id: str
    intent: str
    domain: str
    context: Dict[str, Any]
    expected_elements: List[str]
    evaluation_criteria: Dict[str, float]


@dataclass
class TestResult:
    """Represents the result of a single test"""
    test_case_id: str
    variant_name: str
    response: str
    latency_ms: float
    token_count: int
    cost: float
    quality_scores: Dict[str, float]
    errors: List[str]
    timestamp: datetime


class PromptEvaluator:
    """Evaluates prompt performance using telecom-specific metrics"""
    
    def __init__(self, config_path: str):
        self.config = self._load_config(config_path)
        self.logger = self._setup_logging()
        
    def _load_config(self, config_path: str) -> Dict[str, Any]:
        """Load configuration from YAML file"""
        with open(config_path, 'r') as f:
            return yaml.safe_load(f)
    
    def _setup_logging(self) -> logging.Logger:
        """Setup logging configuration"""
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        return logging.getLogger(__name__)
    
    def evaluate_technical_accuracy(self, response: str, test_case: TestCase) -> float:
        """Evaluate technical accuracy of the response"""
        score = 0.0
        total_checks = 0
        
        # Check for required technical elements
        for element in test_case.expected_elements:
            total_checks += 1
            if element.lower() in response.lower():
                score += 1.0
        
        # Check for telecom-specific terminology
        telecom_terms = [
            'gnb', 'amf', 'smf', 'upf', 'ric', 'xapp', 'e2', 'a1', 'o1',
            'urllc', 'embb', 'mmtc', 'nssai', '5qi', 'qos', 'sla'
        ]
        
        found_terms = sum(1 for term in telecom_terms if term in response.lower())
        terminology_score = min(found_terms / 5.0, 1.0)  # Cap at 1.0
        
        # Combine scores
        if total_checks > 0:
            accuracy_score = score / total_checks
        else:
            accuracy_score = 0.5  # Neutral if no specific elements to check
        
        return (accuracy_score * 0.7) + (terminology_score * 0.3)
    
    def evaluate_yaml_compliance(self, response: str) -> float:
        """Evaluate YAML format compliance"""
        try:
            # Extract YAML content from response
            yaml_content = self._extract_yaml_content(response)
            if not yaml_content:
                return 0.0
            
            # Try to parse YAML
            parsed = yaml.safe_load(yaml_content)
            if parsed is None:
                return 0.2
            
            # Check for proper structure
            score = 0.4  # Basic parsing success
            
            # Check for required fields
            required_fields = ['apiVersion', 'kind', 'metadata', 'spec']
            found_fields = sum(1 for field in required_fields if field in parsed)
            score += (found_fields / len(required_fields)) * 0.4
            
            # Check for comments (indicated by explanatory text)
            if '#' in yaml_content or 'name:' in yaml_content:
                score += 0.2
            
            return min(score, 1.0)
            
        except yaml.YAMLError:
            return 0.1  # Some YAML-like structure but invalid
        except Exception:
            return 0.0
    
    def evaluate_standards_compliance(self, response: str) -> float:
        """Evaluate compliance with telecom standards"""
        standards_keywords = {
            '3gpp': ['3gpp', 'ts 23', 'ts 38', 'rel-', 'release'],
            'o-ran': ['o-ran', 'oran', 'alliance', 'wg'],
            'etsi': ['etsi', 'nfv', 'mano'],
            'ietf': ['ietf', 'rfc', 'internet-draft']
        }
        
        score = 0.0
        standards_found = 0
        
        response_lower = response.lower()
        
        for standard, keywords in standards_keywords.items():
            if any(keyword in response_lower for keyword in keywords):
                standards_found += 1
                score += 0.25
        
        # Bonus for multiple standards references
        if standards_found >= 2:
            score += 0.1
        
        return min(score, 1.0)
    
    def evaluate_completeness(self, response: str, test_case: TestCase) -> float:
        """Evaluate completeness of the response"""
        required_sections = [
            'configuration', 'implementation', 'validation',
            'risk', 'rollback', 'monitoring'
        ]
        
        found_sections = sum(
            1 for section in required_sections 
            if section in response.lower()
        )
        
        return found_sections / len(required_sections)
    
    def _extract_yaml_content(self, response: str) -> str:
        """Extract YAML content from response"""
        lines = response.split('\n')
        yaml_lines = []
        in_yaml_block = False
        
        for line in lines:
            if '```yaml' in line.lower():
                in_yaml_block = True
                continue
            elif '```' in line and in_yaml_block:
                break
            elif in_yaml_block:
                yaml_lines.append(line)
        
        return '\n'.join(yaml_lines)


class ABTestRunner:
    """Runs A/B tests for prompt variants"""
    
    def __init__(self, config_path: str):
        self.config = self._load_config(config_path)
        self.evaluator = PromptEvaluator(config_path)
        self.logger = logging.getLogger(__name__)
        self.session = None
        
    def _load_config(self, config_path: str) -> Dict[str, Any]:
        """Load configuration from YAML file"""
        with open(config_path, 'r') as f:
            return yaml.safe_load(f)
    
    async def __aenter__(self):
        """Async context manager entry"""
        self.session = aiohttp.ClientSession()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        if self.session:
            await self.session.close()
    
    async def run_ab_test(
        self,
        variants: List[PromptVariant],
        test_cases: List[TestCase],
        runs_per_variant: int = 10
    ) -> Dict[str, List[TestResult]]:
        """Run A/B test across all variants and test cases"""
        results = {variant.name: [] for variant in variants}
        
        # Create test plan
        test_plan = []
        for test_case in test_cases:
            for variant in variants:
                for run in range(runs_per_variant):
                    test_plan.append((test_case, variant, run))
        
        # Randomize test order to avoid bias
        random.shuffle(test_plan)
        
        # Execute tests
        for i, (test_case, variant, run) in enumerate(test_plan):
            self.logger.info(
                f"Running test {i+1}/{len(test_plan)}: "
                f"{test_case.id} with {variant.name} (run {run+1})"
            )
            
            try:
                result = await self._execute_single_test(test_case, variant)
                results[variant.name].append(result)
                
                # Add small delay to avoid overwhelming the API
                await asyncio.sleep(0.1)
                
            except Exception as e:
                self.logger.error(f"Test failed: {e}")
                # Create error result
                error_result = TestResult(
                    test_case_id=test_case.id,
                    variant_name=variant.name,
                    response="",
                    latency_ms=0.0,
                    token_count=0,
                    cost=0.0,
                    quality_scores={},
                    errors=[str(e)],
                    timestamp=datetime.now()
                )
                results[variant.name].append(error_result)
        
        return results
    
    async def _execute_single_test(
        self,
        test_case: TestCase,
        variant: PromptVariant
    ) -> TestResult:
        """Execute a single test case with a prompt variant"""
        start_time = time.time()
        
        # Build prompt from variant and test case
        prompt = self._build_prompt(variant, test_case)
        
        # Make API call
        response_text, token_count = await self._call_llm_api(prompt, variant)
        
        end_time = time.time()
        latency_ms = (end_time - start_time) * 1000
        
        # Calculate cost
        cost = self._calculate_cost(token_count, variant.parameters.get('model', 'gpt-4'))
        
        # Evaluate quality
        quality_scores = {
            'technical_accuracy': self.evaluator.evaluate_technical_accuracy(response_text, test_case),
            'yaml_compliance': self.evaluator.evaluate_yaml_compliance(response_text),
            'standards_compliance': self.evaluator.evaluate_standards_compliance(response_text),
            'completeness': self.evaluator.evaluate_completeness(response_text, test_case)
        }
        
        return TestResult(
            test_case_id=test_case.id,
            variant_name=variant.name,
            response=response_text,
            latency_ms=latency_ms,
            token_count=token_count,
            cost=cost,
            quality_scores=quality_scores,
            errors=[],
            timestamp=datetime.now()
        )
    
    def _build_prompt(self, variant: PromptVariant, test_case: TestCase) -> str:
        """Build complete prompt from variant and test case"""
        prompt_parts = []
        
        # System prompt
        if variant.system_prompt:
            prompt_parts.append(f"## System Instructions\n{variant.system_prompt}")
        
        # Examples
        if variant.examples:
            prompt_parts.append("## Examples\n")
            for i, example in enumerate(variant.examples):
                prompt_parts.append(f"### Example {i+1}")
                prompt_parts.append(f"**Input:** {example['input']}")
                prompt_parts.append(f"**Output:**\n```yaml\n{example['output']}\n```")
                if 'explanation' in example:
                    prompt_parts.append(f"**Explanation:** {example['explanation']}")
        
        # Context
        if test_case.context:
            prompt_parts.append("## Current Network Context")
            context_lines = []
            for key, value in test_case.context.items():
                if isinstance(value, list):
                    context_lines.append(f"**{key.title()}:**")
                    for item in value:
                        if isinstance(item, dict):
                            context_lines.append(f"- {item.get('name', 'Unknown')}: {item}")
                        else:
                            context_lines.append(f"- {item}")
                else:
                    context_lines.append(f"**{key.title()}:** {value}")
            prompt_parts.append('\n'.join(context_lines))
        
        # User intent
        prompt_parts.append(f"## User Intent\n{test_case.intent}")
        
        # Response requirements
        prompt_parts.append("""## Response Requirements
Please provide your response in valid YAML format with:
1. Detailed configuration parameters
2. Explanatory comments for each major section
3. Compliance references to relevant standards
4. Risk assessment and mitigation strategies
5. Validation and testing recommendations""")
        
        return '\n\n'.join(prompt_parts)
    
    async def _call_llm_api(self, prompt: str, variant: PromptVariant) -> Tuple[str, int]:
        """Make API call to LLM service"""
        # This would be replaced with actual API calls
        # For now, simulate response
        
        api_config = self.config.get('api', {})
        model = variant.parameters.get('model', 'gpt-4')
        
        # Simulate API call delay
        await asyncio.sleep(random.uniform(0.5, 2.0))
        
        # For demonstration, return a mock response
        mock_response = f"""apiVersion: networking.nephoran.io/v1
kind: NetworkIntent
metadata:
  name: mock-intent-{variant.name}
  namespace: nephoran
spec:
  # Mock configuration generated for variant: {variant.name}
  intentType: {variant.parameters.get('intent_type', 'network_optimization')}
  parameters:
    optimization_target: throughput
    constraints:
      latency: 10ms
      reliability: 99.9%
  validation:
    enabled: true
    criteria:
      - technical_compliance
      - performance_targets
"""
        
        # Estimate token count (rough approximation)
        token_count = len(prompt.split()) + len(mock_response.split())
        
        return mock_response, token_count
    
    def _calculate_cost(self, token_count: int, model: str) -> float:
        """Calculate cost based on token count and model"""
        cost_per_token = {
            'gpt-4': 0.00003,
            'gpt-4-32k': 0.00006,
            'gpt-3.5-turbo': 0.0000015,
            'claude-3': 0.000008
        }
        
        return token_count * cost_per_token.get(model, 0.00003)


class StatisticalAnalyzer:
    """Performs statistical analysis of A/B test results"""
    
    def __init__(self):
        self.logger = logging.getLogger(__name__)
    
    def analyze_results(
        self,
        results: Dict[str, List[TestResult]],
        significance_level: float = 0.05
    ) -> Dict[str, Any]:
        """Perform comprehensive statistical analysis"""
        analysis = {
            'summary_stats': {},
            'significance_tests': {},
            'recommendations': {},
            'detailed_metrics': {}
        }
        
        # Calculate summary statistics for each variant
        for variant_name, variant_results in results.items():
            analysis['summary_stats'][variant_name] = self._calculate_summary_stats(variant_results)
        
        # Perform pairwise significance tests
        variant_names = list(results.keys())
        for i, variant_a in enumerate(variant_names):
            for variant_b in variant_names[i+1:]:
                analysis['significance_tests'][f"{variant_a}_vs_{variant_b}"] = \
                    self._compare_variants(
                        results[variant_a],
                        results[variant_b],
                        significance_level
                    )
        
        # Generate recommendations
        analysis['recommendations'] = self._generate_recommendations(analysis)
        
        # Detailed metrics breakdown
        analysis['detailed_metrics'] = self._calculate_detailed_metrics(results)
        
        return analysis
    
    def _calculate_summary_stats(self, results: List[TestResult]) -> Dict[str, Any]:
        """Calculate summary statistics for a variant"""
        if not results:
            return {}
        
        # Extract metrics
        latencies = [r.latency_ms for r in results if r.latency_ms > 0]
        costs = [r.cost for r in results if r.cost > 0]
        token_counts = [r.token_count for r in results if r.token_count > 0]
        
        # Quality scores
        quality_metrics = {}
        for metric in ['technical_accuracy', 'yaml_compliance', 'standards_compliance', 'completeness']:
            scores = [r.quality_scores.get(metric, 0) for r in results if r.quality_scores.get(metric, 0) > 0]
            if scores:
                quality_metrics[metric] = {
                    'mean': statistics.mean(scores),
                    'std': statistics.stdev(scores) if len(scores) > 1 else 0,
                    'min': min(scores),
                    'max': max(scores),
                    'median': statistics.median(scores)
                }
        
        # Overall composite score
        composite_scores = []
        for result in results:
            if result.quality_scores:
                score = sum(result.quality_scores.values()) / len(result.quality_scores)
                composite_scores.append(score)
        
        return {
            'count': len(results),
            'success_rate': len([r for r in results if not r.errors]) / len(results),
            'latency': {
                'mean': statistics.mean(latencies) if latencies else 0,
                'std': statistics.stdev(latencies) if len(latencies) > 1 else 0,
                'p95': np.percentile(latencies, 95) if latencies else 0,
                'p99': np.percentile(latencies, 99) if latencies else 0
            },
            'cost': {
                'mean': statistics.mean(costs) if costs else 0,
                'total': sum(costs) if costs else 0,
                'std': statistics.stdev(costs) if len(costs) > 1 else 0
            },
            'tokens': {
                'mean': statistics.mean(token_counts) if token_counts else 0,
                'total': sum(token_counts) if token_counts else 0
            },
            'quality_metrics': quality_metrics,
            'composite_score': {
                'mean': statistics.mean(composite_scores) if composite_scores else 0,
                'std': statistics.stdev(composite_scores) if len(composite_scores) > 1 else 0
            }
        }
    
    def _compare_variants(
        self,
        results_a: List[TestResult],
        results_b: List[TestResult],
        significance_level: float
    ) -> Dict[str, Any]:
        """Compare two variants using statistical tests"""
        comparison = {
            'sample_sizes': {'a': len(results_a), 'b': len(results_b)},
            'tests': {}
        }
        
        # Compare composite quality scores
        scores_a = [
            sum(r.quality_scores.values()) / len(r.quality_scores)
            for r in results_a if r.quality_scores
        ]
        scores_b = [
            sum(r.quality_scores.values()) / len(r.quality_scores)
            for r in results_b if r.quality_scores
        ]
        
        if len(scores_a) > 1 and len(scores_b) > 1:
            t_stat, p_value = stats.ttest_ind(scores_a, scores_b)
            comparison['tests']['quality_ttest'] = {
                't_statistic': t_stat,
                'p_value': p_value,
                'significant': p_value < significance_level,
                'mean_diff': statistics.mean(scores_a) - statistics.mean(scores_b)
            }
        
        # Compare latencies
        latencies_a = [r.latency_ms for r in results_a if r.latency_ms > 0]
        latencies_b = [r.latency_ms for r in results_b if r.latency_ms > 0]
        
        if len(latencies_a) > 1 and len(latencies_b) > 1:
            t_stat, p_value = stats.ttest_ind(latencies_a, latencies_b)
            comparison['tests']['latency_ttest'] = {
                't_statistic': t_stat,
                'p_value': p_value,
                'significant': p_value < significance_level,
                'mean_diff': statistics.mean(latencies_a) - statistics.mean(latencies_b)
            }
        
        # Compare costs
        costs_a = [r.cost for r in results_a if r.cost > 0]
        costs_b = [r.cost for r in results_b if r.cost > 0]
        
        if len(costs_a) > 1 and len(costs_b) > 1:
            t_stat, p_value = stats.ttest_ind(costs_a, costs_b)
            comparison['tests']['cost_ttest'] = {
                't_statistic': t_stat,
                'p_value': p_value,
                'significant': p_value < significance_level,
                'mean_diff': statistics.mean(costs_a) - statistics.mean(costs_b)
            }
        
        return comparison
    
    def _generate_recommendations(self, analysis: Dict[str, Any]) -> Dict[str, Any]:
        """Generate recommendations based on analysis"""
        recommendations = {
            'winner': None,
            'confidence': 'low',
            'reasons': [],
            'considerations': []
        }
        
        # Find variant with highest composite score
        best_variant = None
        best_score = 0
        
        for variant_name, stats in analysis['summary_stats'].items():
            composite_mean = stats.get('composite_score', {}).get('mean', 0)
            if composite_mean > best_score:
                best_score = composite_mean
                best_variant = variant_name
        
        if best_variant:
            recommendations['winner'] = best_variant
            recommendations['reasons'].append(f"Highest composite quality score: {best_score:.3f}")
            
            # Check for statistical significance
            significant_comparisons = []
            for comparison_name, comparison_data in analysis['significance_tests'].items():
                if best_variant in comparison_name:
                    quality_test = comparison_data.get('tests', {}).get('quality_ttest', {})
                    if quality_test.get('significant', False):
                        significant_comparisons.append(comparison_name)
            
            if significant_comparisons:
                recommendations['confidence'] = 'high'
                recommendations['reasons'].append(
                    f"Statistically significant improvements in {len(significant_comparisons)} comparisons"
                )
            else:
                recommendations['confidence'] = 'medium'
                recommendations['considerations'].append(
                    "No statistically significant differences found - may need more test runs"
                )
        
        return recommendations
    
    def _calculate_detailed_metrics(self, results: Dict[str, List[TestResult]]) -> Dict[str, Any]:
        """Calculate detailed metrics breakdown"""
        detailed = {}
        
        for variant_name, variant_results in results.items():
            # Error analysis
            error_types = {}
            for result in variant_results:
                for error in result.errors:
                    error_types[error] = error_types.get(error, 0) + 1
            
            # Quality score distribution
            quality_distributions = {}
            for metric in ['technical_accuracy', 'yaml_compliance', 'standards_compliance', 'completeness']:
                scores = [r.quality_scores.get(metric, 0) for r in variant_results if r.quality_scores.get(metric, 0) >= 0]
                if scores:
                    quality_distributions[metric] = {
                        'histogram': np.histogram(scores, bins=10),
                        'quartiles': [np.percentile(scores, q) for q in [25, 50, 75]]
                    }
            
            detailed[variant_name] = {
                'error_analysis': error_types,
                'quality_distributions': quality_distributions,
                'performance_trend': self._calculate_performance_trend(variant_results)
            }
        
        return detailed
    
    def _calculate_performance_trend(self, results: List[TestResult]) -> Dict[str, Any]:
        """Calculate performance trend over time"""
        if len(results) < 3:
            return {'trend': 'insufficient_data'}
        
        # Sort by timestamp
        sorted_results = sorted(results, key=lambda r: r.timestamp)
        
        # Calculate moving average of composite scores
        window_size = max(3, len(results) // 5)
        moving_averages = []
        
        for i in range(len(sorted_results) - window_size + 1):
            window_results = sorted_results[i:i + window_size]
            window_scores = [
                sum(r.quality_scores.values()) / len(r.quality_scores)
                for r in window_results if r.quality_scores
            ]
            if window_scores:
                moving_averages.append(statistics.mean(window_scores))
        
        if len(moving_averages) >= 2:
            # Simple linear trend
            x = list(range(len(moving_averages)))
            slope, intercept, r_value, p_value, std_err = stats.linregress(x, moving_averages)
            
            trend_direction = 'improving' if slope > 0.001 else 'declining' if slope < -0.001 else 'stable'
            
            return {
                'trend': trend_direction,
                'slope': slope,
                'r_squared': r_value ** 2,
                'p_value': p_value,
                'moving_averages': moving_averages
            }
        
        return {'trend': 'insufficient_data'}


class Visualizer:
    """Creates visualizations for A/B test results"""
    
    def __init__(self, output_dir: str):
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(exist_ok=True)
        
        # Set style
        plt.style.use('seaborn-v0_8')
        sns.set_palette("husl")
    
    def create_comprehensive_report(
        self,
        results: Dict[str, List[TestResult]],
        analysis: Dict[str, Any],
        output_file: str = "ab_test_report.html"
    ):
        """Create comprehensive HTML report with all visualizations"""
        # Create individual plots
        plots = {
            'quality_comparison': self.plot_quality_comparison(results),
            'performance_metrics': self.plot_performance_metrics(results),
            'cost_analysis': self.plot_cost_analysis(results),
            'distribution_analysis': self.plot_score_distributions(results),
            'timeline_analysis': self.plot_timeline_analysis(results)
        }
        
        # Generate HTML report
        html_content = self._generate_html_report(analysis, plots)
        
        # Save report
        report_path = self.output_dir / output_file
        with open(report_path, 'w') as f:
            f.write(html_content)
        
        print(f"Comprehensive report saved to: {report_path}")
    
    def plot_quality_comparison(self, results: Dict[str, List[TestResult]]) -> str:
        """Create quality comparison plots"""
        fig, axes = plt.subplots(2, 2, figsize=(15, 12))
        fig.suptitle('Quality Metrics Comparison', fontsize=16, fontweight='bold')
        
        metrics = ['technical_accuracy', 'yaml_compliance', 'standards_compliance', 'completeness']
        
        for idx, metric in enumerate(metrics):
            ax = axes[idx // 2, idx % 2]
            
            # Prepare data
            data = []
            labels = []
            
            for variant_name, variant_results in results.items():
                scores = [r.quality_scores.get(metric, 0) for r in variant_results if r.quality_scores.get(metric, 0) >= 0]
                if scores:
                    data.append(scores)
                    labels.append(variant_name)
            
            # Create box plot
            if data:
                bp = ax.boxplot(data, labels=labels, patch_artist=True)
                
                # Color boxes
                colors = sns.color_palette("husl", len(data))
                for patch, color in zip(bp['boxes'], colors):
                    patch.set_facecolor(color)
                    patch.set_alpha(0.7)
            
            ax.set_title(metric.replace('_', ' ').title())
            ax.set_ylabel('Score')
            ax.grid(True, alpha=0.3)
            ax.set_ylim(0, 1)
        
        plt.tight_layout()
        
        # Save plot
        plot_path = self.output_dir / 'quality_comparison.png'
        plt.savefig(plot_path, dpi=300, bbox_inches='tight')
        plt.close()
        
        return str(plot_path)
    
    def plot_performance_metrics(self, results: Dict[str, List[TestResult]]) -> str:
        """Create performance metrics visualization"""
        fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(15, 6))
        fig.suptitle('Performance Metrics', fontsize=16, fontweight='bold')
        
        # Latency comparison
        latency_data = []
        labels = []
        
        for variant_name, variant_results in results.items():
            latencies = [r.latency_ms for r in variant_results if r.latency_ms > 0]
            if latencies:
                latency_data.append(latencies)
                labels.append(variant_name)
        
        if latency_data:
            bp1 = ax1.boxplot(latency_data, labels=labels, patch_artist=True)
            colors = sns.color_palette("husl", len(latency_data))
            for patch, color in zip(bp1['boxes'], colors):
                patch.set_facecolor(color)
                patch.set_alpha(0.7)
        
        ax1.set_title('Response Latency')
        ax1.set_ylabel('Latency (ms)')
        ax1.grid(True, alpha=0.3)
        
        # Token usage comparison
        token_data = []
        for variant_name, variant_results in results.items():
            tokens = [r.token_count for r in variant_results if r.token_count > 0]
            if tokens:
                token_data.append(tokens)
        
        if token_data:
            bp2 = ax2.boxplot(token_data, labels=labels, patch_artist=True)
            for patch, color in zip(bp2['boxes'], colors):
                patch.set_facecolor(color)
                patch.set_alpha(0.7)
        
        ax2.set_title('Token Usage')
        ax2.set_ylabel('Token Count')
        ax2.grid(True, alpha=0.3)
        
        plt.tight_layout()
        
        # Save plot
        plot_path = self.output_dir / 'performance_metrics.png'
        plt.savefig(plot_path, dpi=300, bbox_inches='tight')
        plt.close()
        
        return str(plot_path)
    
    def plot_cost_analysis(self, results: Dict[str, List[TestResult]]) -> str:
        """Create cost analysis visualization"""
        fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(15, 6))
        fig.suptitle('Cost Analysis', fontsize=16, fontweight='bold')
        
        # Cost per request
        variant_names = []
        mean_costs = []
        total_costs = []
        
        for variant_name, variant_results in results.items():
            costs = [r.cost for r in variant_results if r.cost > 0]
            if costs:
                variant_names.append(variant_name)
                mean_costs.append(statistics.mean(costs))
                total_costs.append(sum(costs))
        
        if variant_names:
            # Bar plot for mean cost per request
            colors = sns.color_palette("husl", len(variant_names))
            bars1 = ax1.bar(variant_names, mean_costs, color=colors, alpha=0.7)
            ax1.set_title('Mean Cost per Request')
            ax1.set_ylabel('Cost ($)')
            ax1.grid(True, alpha=0.3)
            
            # Add value labels on bars
            for bar, cost in zip(bars1, mean_costs):
                ax1.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 0.0001,
                        f'${cost:.4f}', ha='center', va='bottom')
            
            # Bar plot for total cost
            bars2 = ax2.bar(variant_names, total_costs, color=colors, alpha=0.7)
            ax2.set_title('Total Cost')
            ax2.set_ylabel('Total Cost ($)')
            ax2.grid(True, alpha=0.3)
            
            # Add value labels on bars
            for bar, cost in zip(bars2, total_costs):
                ax2.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 0.001,
                        f'${cost:.3f}', ha='center', va='bottom')
        
        plt.xticks(rotation=45)
        plt.tight_layout()
        
        # Save plot
        plot_path = self.output_dir / 'cost_analysis.png'
        plt.savefig(plot_path, dpi=300, bbox_inches='tight')
        plt.close()
        
        return str(plot_path)
    
    def plot_score_distributions(self, results: Dict[str, List[TestResult]]) -> str:
        """Create score distribution plots"""
        fig, axes = plt.subplots(2, 2, figsize=(15, 12))
        fig.suptitle('Score Distributions', fontsize=16, fontweight='bold')
        
        metrics = ['technical_accuracy', 'yaml_compliance', 'standards_compliance', 'completeness']
        
        for idx, metric in enumerate(metrics):
            ax = axes[idx // 2, idx % 2]
            
            for variant_name, variant_results in results.items():
                scores = [r.quality_scores.get(metric, 0) for r in variant_results if r.quality_scores.get(metric, 0) >= 0]
                if scores:
                    ax.hist(scores, alpha=0.6, label=variant_name, bins=20, density=True)
            
            ax.set_title(metric.replace('_', ' ').title())
            ax.set_xlabel('Score')
            ax.set_ylabel('Density')
            ax.legend()
            ax.grid(True, alpha=0.3)
            ax.set_xlim(0, 1)
        
        plt.tight_layout()
        
        # Save plot
        plot_path = self.output_dir / 'score_distributions.png'
        plt.savefig(plot_path, dpi=300, bbox_inches='tight')
        plt.close()
        
        return str(plot_path)
    
    def plot_timeline_analysis(self, results: Dict[str, List[TestResult]]) -> str:
        """Create timeline analysis plots"""
        fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(15, 10))
        fig.suptitle('Timeline Analysis', fontsize=16, fontweight='bold')
        
        for variant_name, variant_results in results.items():
            # Sort by timestamp
            sorted_results = sorted(variant_results, key=lambda r: r.timestamp)
            
            # Extract timestamps and composite scores
            timestamps = [r.timestamp for r in sorted_results]
            composite_scores = []
            latencies = []
            
            for result in sorted_results:
                if result.quality_scores:
                    score = sum(result.quality_scores.values()) / len(result.quality_scores)
                    composite_scores.append(score)
                    latencies.append(result.latency_ms)
                else:
                    composite_scores.append(0)
                    latencies.append(result.latency_ms)
            
            if timestamps:
                # Plot composite scores over time
                ax1.plot(timestamps, composite_scores, marker='o', label=variant_name, alpha=0.7)
                
                # Plot latencies over time
                ax2.plot(timestamps, latencies, marker='s', label=variant_name, alpha=0.7)
        
        ax1.set_title('Quality Scores Over Time')
        ax1.set_ylabel('Composite Score')
        ax1.legend()
        ax1.grid(True, alpha=0.3)
        ax1.set_ylim(0, 1)
        
        ax2.set_title('Latency Over Time')
        ax2.set_xlabel('Time')
        ax2.set_ylabel('Latency (ms)')
        ax2.legend()
        ax2.grid(True, alpha=0.3)
        
        # Format x-axis
        for ax in [ax1, ax2]:
            ax.tick_params(axis='x', rotation=45)
        
        plt.tight_layout()
        
        # Save plot
        plot_path = self.output_dir / 'timeline_analysis.png'
        plt.savefig(plot_path, dpi=300, bbox_inches='tight')
        plt.close()
        
        return str(plot_path)
    
    def _generate_html_report(self, analysis: Dict[str, Any], plots: Dict[str, str]) -> str:
        """Generate comprehensive HTML report"""
        html_template = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Prompt A/B Test Report</title>
    <style>
        body {{
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }}
        .container {{
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 0 20px rgba(0,0,0,0.1);
        }}
        h1, h2, h3 {{
            color: #333;
        }}
        .summary {{
            background: #e8f4fd;
            padding: 20px;
            border-radius: 8px;
            margin: 20px 0;
        }}
        .recommendation {{
            background: #d4edda;
            border: 1px solid #c3e6cb;
            padding: 15px;
            border-radius: 5px;
            margin: 15px 0;
        }}
        .warning {{
            background: #fff3cd;
            border: 1px solid #ffeaa7;
            padding: 15px;
            border-radius: 5px;
            margin: 15px 0;
        }}
        .plot {{
            text-align: center;
            margin: 30px 0;
        }}
        .plot img {{
            max-width: 100%;
            height: auto;
            border: 1px solid #ddd;
            border-radius: 5px;
        }}
        table {{
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }}
        th, td {{
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }}
        th {{
            background-color: #f2f2f2;
            font-weight: bold;
        }}
        .metric {{
            display: inline-block;
            margin: 10px;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 5px;
            text-align: center;
        }}
        .metric-value {{
            font-size: 24px;
            font-weight: bold;
            color: #007bff;
        }}
        .metric-label {{
            font-size: 14px;
            color: #6c757d;
        }}
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Prompt A/B Test Report</h1>
        <p><strong>Generated:</strong> {timestamp}</p>
        
        <div class="summary">
            <h2>📊 Executive Summary</h2>
            <p><strong>Winner:</strong> {winner}</p>
            <p><strong>Confidence:</strong> {confidence}</p>
            <div>
                <strong>Key Findings:</strong>
                <ul>
                    {findings}
                </ul>
            </div>
        </div>
        
        {recommendations_section}
        
        <h2>📈 Performance Visualizations</h2>
        
        <div class="plot">
            <h3>Quality Metrics Comparison</h3>
            <img src="{quality_plot}" alt="Quality Comparison">
        </div>
        
        <div class="plot">
            <h3>Performance Metrics</h3>
            <img src="{performance_plot}" alt="Performance Metrics">
        </div>
        
        <div class="plot">
            <h3>Cost Analysis</h3>
            <img src="{cost_plot}" alt="Cost Analysis">
        </div>
        
        <div class="plot">
            <h3>Score Distributions</h3>
            <img src="{distribution_plot}" alt="Score Distributions">
        </div>
        
        <div class="plot">
            <h3>Timeline Analysis</h3>
            <img src="{timeline_plot}" alt="Timeline Analysis">
        </div>
        
        <h2>📋 Detailed Statistics</h2>
        {statistics_table}
        
        <h2>🔬 Statistical Tests</h2>
        {significance_tests}
        
        <footer style="margin-top: 50px; padding-top: 20px; border-top: 1px solid #eee; color: #666;">
            <p>Generated by Nephoran Intent Operator Prompt A/B Testing Framework</p>
        </footer>
    </div>
</body>
</html>
        """
        
        # Extract data for template
        recommendations = analysis.get('recommendations', {})
        winner = recommendations.get('winner', 'No clear winner')
        confidence = recommendations.get('confidence', 'Unknown').title()
        
        findings_list = recommendations.get('reasons', []) + recommendations.get('considerations', [])
        findings_html = ''.join(f'<li>{finding}</li>' for finding in findings_list)
        
        # Generate recommendations section
        recommendations_html = ""
        if recommendations.get('reasons'):
            recommendations_html += '<div class="recommendation"><h3>✅ Recommendations</h3><ul>'
            for reason in recommendations.get('reasons', []):
                recommendations_html += f'<li>{reason}</li>'
            recommendations_html += '</ul></div>'
        
        if recommendations.get('considerations'):
            recommendations_html += '<div class="warning"><h3>⚠️ Considerations</h3><ul>'
            for consideration in recommendations.get('considerations', []):
                recommendations_html += f'<li>{consideration}</li>'
            recommendations_html += '</ul></div>'
        
        # Generate statistics table
        stats_html = '<table><thead><tr><th>Variant</th><th>Success Rate</th><th>Avg Quality</th><th>Avg Latency (ms)</th><th>Avg Cost ($)</th></tr></thead><tbody>'
        
        for variant_name, stats in analysis.get('summary_stats', {}).items():
            success_rate = f"{stats.get('success_rate', 0)*100:.1f}%"
            avg_quality = f"{stats.get('composite_score', {}).get('mean', 0):.3f}"
            avg_latency = f"{stats.get('latency', {}).get('mean', 0):.1f}"
            avg_cost = f"{stats.get('cost', {}).get('mean', 0):.4f}"
            
            stats_html += f'<tr><td>{variant_name}</td><td>{success_rate}</td><td>{avg_quality}</td><td>{avg_latency}</td><td>{avg_cost}</td></tr>'
        
        stats_html += '</tbody></table>'
        
        # Generate significance tests
        sig_tests_html = ""
        for test_name, test_data in analysis.get('significance_tests', {}).items():
            quality_test = test_data.get('tests', {}).get('quality_ttest', {})
            if quality_test:
                is_significant = quality_test.get('significant', False)
                p_value = quality_test.get('p_value', 1.0)
                mean_diff = quality_test.get('mean_diff', 0)
                
                status_class = "recommendation" if is_significant else "warning"
                status_text = "Significant" if is_significant else "Not Significant"
                
                sig_tests_html += f"""
                <div class="{status_class}">
                    <h4>{test_name.replace('_', ' vs ').title()}</h4>
                    <p><strong>Status:</strong> {status_text} (p = {p_value:.4f})</p>
                    <p><strong>Mean Difference:</strong> {mean_diff:.4f}</p>
                </div>
                """
        
        # Fill template
        return html_template.format(
            timestamp=datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            winner=winner,
            confidence=confidence,
            findings=findings_html,
            recommendations_section=recommendations_html,
            quality_plot=Path(plots['quality_comparison']).name,
            performance_plot=Path(plots['performance_metrics']).name,
            cost_plot=Path(plots['cost_analysis']).name,
            distribution_plot=Path(plots['distribution_analysis']).name,
            timeline_plot=Path(plots['timeline_analysis']).name,
            statistics_table=stats_html,
            significance_tests=sig_tests_html
        )


async def main():
    """Main function to run A/B tests"""
    parser = argparse.ArgumentParser(description='Run A/B tests for prompt variants')
    parser.add_argument('--config', required=True, help='Configuration file path')
    parser.add_argument('--variants', required=True, help='Prompt variants file path')
    parser.add_argument('--test-cases', required=True, help='Test cases file path')
    parser.add_argument('--runs', type=int, default=10, help='Number of runs per variant')
    parser.add_argument('--output-dir', default='./ab_test_results', help='Output directory')
    
    args = parser.parse_args()
    
    # Load variants and test cases
    with open(args.variants, 'r') as f:
        variants_data = yaml.safe_load(f)
        variants = [PromptVariant(**variant) for variant in variants_data['variants']]
    
    with open(args.test_cases, 'r') as f:
        test_cases_data = yaml.safe_load(f)
        test_cases = [TestCase(**case) for case in test_cases_data['test_cases']]
    
    # Create output directory
    output_dir = Path(args.output_dir)
    output_dir.mkdir(exist_ok=True)
    
    # Run A/B tests
    async with ABTestRunner(args.config) as runner:
        print(f"Running A/B tests with {len(variants)} variants and {len(test_cases)} test cases...")
        print(f"Total test executions: {len(variants) * len(test_cases) * args.runs}")
        
        results = await runner.run_ab_test(variants, test_cases, args.runs)
        
        # Analyze results
        analyzer = StatisticalAnalyzer()
        analysis = analyzer.analyze_results(results)
        
        # Create visualizations
        visualizer = Visualizer(str(output_dir))
        visualizer.create_comprehensive_report(results, analysis)
        
        # Save raw results
        results_file = output_dir / 'raw_results.json'
        serializable_results = {}
        for variant_name, variant_results in results.items():
            serializable_results[variant_name] = [asdict(result) for result in variant_results]
            # Convert datetime objects to strings
            for result_dict in serializable_results[variant_name]:
                result_dict['timestamp'] = result_dict['timestamp'].isoformat()
        
        with open(results_file, 'w') as f:
            json.dump(serializable_results, f, indent=2)
        
        # Save analysis
        analysis_file = output_dir / 'analysis.json'
        with open(analysis_file, 'w') as f:
            json.dump(analysis, f, indent=2, default=str)
        
        # Print summary
        print(f"\n🎉 A/B Testing Complete!")
        print(f"Results saved to: {output_dir}")
        print(f"Winner: {analysis['recommendations']['winner']}")
        print(f"Confidence: {analysis['recommendations']['confidence']}")
        
        if analysis['recommendations']['reasons']:
            print("Key findings:")
            for reason in analysis['recommendations']['reasons']:
                print(f"  • {reason}")


if __name__ == "__main__":
    asyncio.run(main())