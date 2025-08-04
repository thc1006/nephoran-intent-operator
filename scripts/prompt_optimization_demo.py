#!/usr/bin/env python3
"""
Prompt Optimization Demo for Nephoran Intent Operator

This script demonstrates how to use the prompt optimization framework
to improve LLM responses for telecom network configurations.
"""

import asyncio
import json
import logging
from pathlib import Path
from typing import Dict, List, Any
import matplotlib.pyplot as plt
import numpy as np
from datetime import datetime

# Import our modules (in production, these would be proper imports)
from prompt_ab_testing import (
    PromptVariant, TestCase, PromptABTestRunner,
    create_telecom_test_cases, PromptVariantType
)
from prompt_evaluation_metrics import (
    TelecomEvaluator, TelecomMetrics, MetricsAggregator,
    evaluate_prompt_response
)

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class PromptOptimizationDemo:
    """Demonstrates prompt optimization workflow"""
    
    def __init__(self):
        self.evaluator = TelecomEvaluator()
        self.aggregator = MetricsAggregator()
        self.test_runner = PromptABTestRunner(
            llm_client=None,  # Mock client
            config={'timeout': 30}
        )
        
    async def demonstrate_optimization_workflow(self):
        """Run complete optimization workflow"""
        logger.info("Starting Prompt Optimization Demo")
        
        # Step 1: Create baseline and optimized prompts
        prompts = self.create_prompt_variants()
        
        # Step 2: Create test cases
        test_cases = self.create_comprehensive_test_cases()
        
        # Step 3: Run initial evaluation
        logger.info("Step 1: Running baseline evaluation...")
        baseline_results = await self.evaluate_prompts(prompts[:1], test_cases)
        self.display_results("Baseline Results", baseline_results)
        
        # Step 4: Identify weaknesses
        logger.info("\nStep 2: Identifying weaknesses...")
        weaknesses = self.identify_optimization_opportunities(baseline_results)
        
        # Step 5: Create optimized variants
        logger.info("\nStep 3: Creating optimized variants...")
        optimized_prompts = self.create_optimized_variants(prompts[0], weaknesses)
        
        # Step 6: Run A/B test
        logger.info("\nStep 4: Running A/B test...")
        ab_results = await self.run_ab_test_demo(
            [prompts[0]] + optimized_prompts,
            test_cases
        )
        
        # Step 7: Analyze improvements
        logger.info("\nStep 5: Analyzing improvements...")
        self.analyze_improvements(baseline_results, ab_results)
        
        # Step 8: Generate recommendations
        logger.info("\nStep 6: Generating recommendations...")
        recommendations = self.generate_optimization_recommendations(ab_results)
        
        # Step 9: Create final optimized prompt
        logger.info("\nStep 7: Creating final optimized prompt...")
        final_prompt = self.create_final_optimized_prompt(
            prompts[0], ab_results, recommendations
        )
        
        # Step 10: Validate final prompt
        logger.info("\nStep 8: Validating final prompt...")
        validation_results = await self.validate_final_prompt(final_prompt, test_cases)
        
        # Display final results
        self.display_final_summary(baseline_results, validation_results)
        
        logger.info("\nOptimization workflow completed!")
        
    def create_prompt_variants(self) -> List[PromptVariant]:
        """Create different prompt variants for testing"""
        variants = []
        
        # Baseline prompt
        baseline = PromptVariant(
            id="baseline",
            name="Baseline Telecom Prompt",
            type=PromptVariantType.BASELINE,
            system_prompt="""You are a telecom network engineer. Generate configurations for network functions.""",
            user_prompt_template="""Create a configuration for: {intent}
            
Output format: Kubernetes YAML or JSON""",
            metadata={"version": "1.0", "domain": "telecom"}
        )
        variants.append(baseline)
        
        # O-RAN specialized prompt
        oran_specialized = PromptVariant(
            id="oran_specialized",
            name="O-RAN Specialized",
            type=PromptVariantType.TECHNICAL,
            system_prompt="""You are an O-RAN architecture expert with comprehensive knowledge of:

**O-RAN Components:**
- Near-RT RIC: Hosts xApps (10ms-1s control), E2/A1 interfaces
- Non-RT RIC: Hosts rApps (>1s control), policy management
- O-CU/O-DU/O-RU: Disaggregated RAN components
- SMO: Service Management and Orchestration

**Key Interfaces:**
- A1: Policy guidance (Non-RT RIC → Near-RT RIC)
- E2: Real-time control (Near-RT RIC → RAN)
- O1: FCAPS management (SMO → all components)
- O2: Cloud infrastructure (SMO → O-Cloud)
- Open Fronthaul: O-DU ↔ O-RU communication

**Configuration Requirements:**
- Multi-vendor interoperability
- Cloud-native deployment patterns
- Proper interface definitions
- Security and isolation""",
            user_prompt_template="""Intent: {intent}

Generate a production-ready Kubernetes configuration including:
1. Proper namespace and labels (O-RAN compliant)
2. Container specifications with correct images
3. Network interfaces (E2, A1, O1 as needed)
4. Resource allocation (CPU, memory)
5. High availability setup (replicas, anti-affinity)
6. Environment variables for O-RAN components
7. Config maps for O1 management

Ensure compliance with O-RAN Alliance specifications.""",
            examples=[{
                "intent": "Deploy Near-RT RIC",
                "response": """{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "near-rt-ric",
    "namespace": "oran-system",
    "labels": {
      "oran.alliance/component": "near-rt-ric",
      "oran.alliance/version": "g-release"
    }
  },
  "spec": {
    "replicas": 3,
    "template": {
      "spec": {
        "containers": [{
          "name": "ric",
          "ports": [
            {"containerPort": 36421, "name": "e2ap", "protocol": "SCTP"},
            {"containerPort": 9090, "name": "a1", "protocol": "TCP"}
          ]
        }]
      }
    }
  }
}"""
            }],
            metadata={"version": "2.0", "domain": "O-RAN", "optimized": True}
        )
        variants.append(oran_specialized)
        
        # 5G Core specialized prompt
        fiveg_specialized = PromptVariant(
            id="5g_core_specialized",
            name="5G Core Specialized",
            type=PromptVariantType.TECHNICAL,
            system_prompt="""You are a 5G Core Network expert specializing in cloud-native deployments.

**5G Core Architecture (SBA):**
- Control Plane: AMF, SMF, PCF, UDM, UDR, AUSF, NSSF, NEF, NRF
- User Plane: UPF
- Service-based interfaces using HTTP/2 + JSON

**Key Concepts:**
- Network slicing (eMBB, URLLC, mMTC)
- QoS flows and 5QI values
- PDU sessions
- PLMN and S-NSSAI

Always ensure 3GPP compliance (R15/R16/R17).""",
            user_prompt_template="""Intent: {intent}

Create a 5G Core network function deployment with:
- 3GPP-compliant configuration
- Service mesh readiness
- Proper SBI exposure
- Network slice support
- Database requirements
- Scaling and redundancy""",
            metadata={"version": "2.0", "domain": "5G-Core", "optimized": True}
        )
        variants.append(fiveg_specialized)
        
        return variants
    
    def create_comprehensive_test_cases(self) -> List[TestCase]:
        """Create comprehensive test cases"""
        test_cases = []
        
        # O-RAN test cases
        test_cases.append(TestCase(
            id="oran_ric_basic",
            intent="Deploy Near-RT RIC for traffic optimization",
            intent_type="NetworkFunctionDeployment",
            domain="O-RAN",
            expected_fields=[
                "apiVersion", "kind", "metadata.name",
                "metadata.namespace", "spec.replicas"
            ],
            validation_rules={
                "min_replicas": 2,
                "required_interfaces": ["E2", "A1"],
                "namespace_pattern": "oran.*"
            },
            complexity="moderate",
            tags=["oran", "ric", "basic"]
        ))
        
        test_cases.append(TestCase(
            id="oran_xapp_advanced",
            intent="Deploy Near-RT RIC with xApp platform for ML-based optimization in multi-vendor environment",
            intent_type="NetworkFunctionDeployment",
            domain="O-RAN",
            expected_fields=[
                "apiVersion", "kind", "metadata.name",
                "metadata.labels", "spec.template.spec.containers",
                "spec.template.spec.volumes"
            ],
            validation_rules={
                "min_replicas": 3,
                "required_interfaces": ["E2", "A1", "R1"],
                "required_env_vars": ["XAPP_ONBOARDER_ENABLED", "RIC_ID"],
                "ha_required": True
            },
            complexity="complex",
            tags=["oran", "ric", "xapp", "advanced"]
        ))
        
        # 5G Core test cases
        test_cases.append(TestCase(
            id="5gc_smf_urllc",
            intent="Deploy SMF with URLLC support for industrial IoT with <1ms latency",
            intent_type="NetworkFunctionDeployment",
            domain="5G-Core",
            expected_fields=[
                "apiVersion", "kind", "metadata.labels",
                "spec.template.spec.containers"
            ],
            validation_rules={
                "nf_type": "SMF",
                "slice_type": "URLLC",
                "max_latency": "1ms",
                "sbi_required": True
            },
            complexity="complex",
            tags=["5g-core", "smf", "urllc", "low-latency"]
        ))
        
        # Network slicing test cases
        test_cases.append(TestCase(
            id="slice_embb_video",
            intent="Configure eMBB network slice for 4K video streaming with 100Mbps guaranteed bandwidth",
            intent_type="NetworkSliceConfiguration",
            domain="Network-Slicing",
            expected_fields=[
                "network_slice.sst", "network_slice.sd",
                "qos_parameters"
            ],
            validation_rules={
                "sst": 1,  # eMBB
                "min_bandwidth": "100Mbps",
                "qos_required": True
            },
            complexity="moderate",
            tags=["slice", "embb", "video", "qos"]
        ))
        
        return test_cases
    
    async def evaluate_prompts(self, prompts: List[PromptVariant], 
                              test_cases: List[TestCase]) -> Dict[str, Any]:
        """Evaluate prompts against test cases"""
        results = {
            'evaluations': [],
            'summary': {}
        }
        
        for prompt in prompts:
            prompt_results = []
            
            for test_case in test_cases:
                # Generate prompt
                full_prompt = prompt.generate_prompt(
                    test_case.intent,
                    test_case.context
                )
                
                # Simulate LLM response
                response = await self._simulate_llm_response(test_case)
                
                # Evaluate response
                metrics = self.evaluator.evaluate_response(
                    response,
                    test_case.intent_type,
                    test_case.domain
                )
                
                # Store in aggregator
                self.aggregator.add_evaluation(
                    prompt.id,
                    test_case.id,
                    metrics,
                    {'domain': test_case.domain, 'complexity': test_case.complexity}
                )
                
                prompt_results.append({
                    'test_case': test_case.id,
                    'metrics': metrics,
                    'success': metrics.overall_score > 0.7
                })
            
            results['evaluations'].append({
                'prompt_id': prompt.id,
                'results': prompt_results
            })
        
        # Generate summary
        results['summary'] = self.aggregator.get_summary_statistics()
        
        return results
    
    async def _simulate_llm_response(self, test_case: TestCase) -> Dict[str, Any]:
        """Simulate LLM response based on test case"""
        await asyncio.sleep(0.1)  # Simulate API call
        
        if test_case.domain == "O-RAN":
            if "ric" in test_case.id:
                return {
                    "apiVersion": "apps/v1",
                    "kind": "Deployment",
                    "metadata": {
                        "name": "near-rt-ric",
                        "namespace": "oran-system",
                        "labels": {
                            "oran.alliance/component": "ric",
                            "oran.alliance/type": "near-rt"
                        }
                    },
                    "spec": {
                        "replicas": 3,
                        "selector": {
                            "matchLabels": {"app": "near-rt-ric"}
                        },
                        "template": {
                            "spec": {
                                "containers": [{
                                    "name": "ric",
                                    "image": "oran/near-rt-ric:g-release",
                                    "ports": [
                                        {"containerPort": 36421, "name": "e2ap", "protocol": "SCTP"},
                                        {"containerPort": 9090, "name": "a1-rest", "protocol": "TCP"},
                                        {"containerPort": 8888, "name": "r1", "protocol": "TCP"}
                                    ],
                                    "env": [
                                        {"name": "RIC_ID", "value": "ric-001"},
                                        {"name": "XAPP_ONBOARDER_ENABLED", "value": "true"},
                                        {"name": "E2_TERM_INIT", "value": "true"}
                                    ],
                                    "resources": {
                                        "requests": {"cpu": "2", "memory": "4Gi"},
                                        "limits": {"cpu": "4", "memory": "8Gi"}
                                    },
                                    "livenessProbe": {
                                        "httpGet": {"path": "/health", "port": 8080},
                                        "periodSeconds": 10
                                    }
                                }],
                                "volumes": [{
                                    "name": "xapp-registry",
                                    "persistentVolumeClaim": {"claimName": "xapp-pvc"}
                                }]
                            }
                        }
                    },
                    "o1_config": '<?xml version="1.0"?><config xmlns="urn:o-ran:config:1.0"></config>'
                }
        
        elif test_case.domain == "5G-Core":
            return {
                "apiVersion": "apps/v1",
                "kind": "StatefulSet",
                "metadata": {
                    "name": "smf-urllc",
                    "namespace": "5gc",
                    "labels": {
                        "nf.5gc.3gpp.org/type": "smf",
                        "nf.5gc.3gpp.org/slice": "urllc"
                    }
                },
                "spec": {
                    "replicas": 2,
                    "template": {
                        "spec": {
                            "containers": [{
                                "name": "smf",
                                "ports": [
                                    {"containerPort": 8080, "name": "sbi", "protocol": "TCP"}
                                ],
                                "env": [
                                    {"name": "SLICE_TYPE", "value": "URLLC"},
                                    {"name": "LATENCY_TARGET", "value": "1ms"}
                                ]
                            }]
                        }
                    }
                }
            }
        
        else:  # Network slicing
            return {
                "network_slice": {
                    "sst": 1,
                    "sd": "000001",
                    "plmn_id": "00101"
                },
                "qos_parameters": {
                    "5qi": 5,
                    "guaranteed_bitrate": "100Mbps",
                    "packet_delay_budget": "100ms"
                }
            }
    
    def identify_optimization_opportunities(self, results: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Identify areas for improvement"""
        weaknesses = self.aggregator.identify_weaknesses(threshold=0.8)
        
        opportunities = []
        for weakness in weaknesses[:5]:  # Top 5 weaknesses
            opportunity = {
                'metric': weakness['metric'],
                'current_score': weakness['average_score'],
                'target_score': 0.8,
                'improvement_needed': weakness['improvement_needed'],
                'recommendation': self._generate_improvement_recommendation(weakness)
            }
            opportunities.append(opportunity)
            
            logger.info(f"Weakness found: {weakness['metric']} "
                       f"(score: {weakness['average_score']:.2f})")
        
        return opportunities
    
    def _generate_improvement_recommendation(self, weakness: Dict[str, Any]) -> str:
        """Generate specific improvement recommendation"""
        metric = weakness['metric']
        
        recommendations = {
            'oran_interface_compliance': "Add detailed O-RAN interface specifications and port mappings",
            'nf_type_accuracy': "Include explicit 5G network function type labels",
            'slice_configuration_validity': "Ensure proper S-NSSAI configuration with SST and SD",
            'resource_specification_accuracy': "Add comprehensive resource requests and limits",
            'ha_configuration_score': "Increase replicas and add pod anti-affinity rules",
            'probe_configuration_completeness': "Add both liveness and readiness probes"
        }
        
        return recommendations.get(metric, f"Improve {metric} configuration")
    
    def create_optimized_variants(self, base_prompt: PromptVariant, 
                                 weaknesses: List[Dict[str, Any]]) -> List[PromptVariant]:
        """Create optimized prompt variants based on weaknesses"""
        optimized_variants = []
        
        # Create targeted optimization for each major weakness
        for i, weakness in enumerate(weaknesses[:3]):  # Top 3 weaknesses
            metric = weakness['metric']
            
            # Create optimized system prompt
            optimized_system = base_prompt.system_prompt + f"\n\nIMPORTANT: {weakness['recommendation']}"
            
            # Add specific instructions
            if 'oran' in metric:
                optimized_system += "\nEnsure all O-RAN interfaces (E2, A1, O1) are properly configured with correct ports and protocols."
            elif 'resource' in metric:
                optimized_system += "\nAlways specify resource requests and limits for CPU and memory."
            elif 'ha' in metric or 'replica' in metric:
                optimized_system += "\nConfigure high availability with at least 3 replicas and anti-affinity rules."
            
            variant = PromptVariant(
                id=f"optimized_{metric}",
                name=f"Optimized for {metric}",
                type=PromptVariantType.ENHANCED,
                system_prompt=optimized_system,
                user_prompt_template=base_prompt.user_prompt_template,
                examples=base_prompt.examples,
                metadata={
                    'optimization_target': metric,
                    'base_prompt': base_prompt.id
                }
            )
            
            optimized_variants.append(variant)
        
        return optimized_variants
    
    async def run_ab_test_demo(self, prompts: List[PromptVariant], 
                               test_cases: List[TestCase]) -> Dict[str, Any]:
        """Run A/B test demonstration"""
        results = await self.test_runner.run_ab_test(
            variants=prompts,
            test_cases=test_cases,
            iterations=3  # Reduced for demo
        )
        
        # Display A/B test results
        winner = results['analysis']['winner']
        logger.info(f"\nA/B Test Winner: {winner['variant_id']} "
                   f"(score: {winner['score']:.3f})")
        
        return results
    
    def analyze_improvements(self, baseline_results: Dict[str, Any], 
                           ab_results: Dict[str, Any]):
        """Analyze improvements from optimization"""
        baseline_stats = self.aggregator.get_summary_statistics()
        
        # Compare key metrics
        improvements = {}
        key_metrics = ['overall_score', 'technical_score', 'domain_score', 'compliance_score']
        
        for metric in key_metrics:
            if metric in baseline_stats:
                baseline_value = baseline_stats[metric]['mean']
                
                # Find best optimized value
                best_value = baseline_value
                for variant_id, stats in ab_results['analysis']['variant_statistics'].items():
                    if variant_id != 'baseline' and metric in stats:
                        variant_value = stats[metric]['mean']
                        if variant_value > best_value:
                            best_value = variant_value
                
                improvement = ((best_value - baseline_value) / baseline_value) * 100
                improvements[metric] = {
                    'baseline': baseline_value,
                    'optimized': best_value,
                    'improvement_pct': improvement
                }
                
                logger.info(f"{metric}: {baseline_value:.3f} → {best_value:.3f} "
                           f"({improvement:+.1f}%)")
        
        return improvements
    
    def generate_optimization_recommendations(self, ab_results: Dict[str, Any]) -> List[str]:
        """Generate specific optimization recommendations"""
        recommendations = []
        
        # Analyze variant performance
        variant_stats = ab_results['analysis']['variant_statistics']
        
        # Find best performers for each metric
        metric_winners = {}
        for variant_id, stats in variant_stats.items():
            for metric, values in stats.items():
                if metric != 'overall_score' and 'mean' in values:
                    if metric not in metric_winners or values['mean'] > metric_winners[metric]['score']:
                        metric_winners[metric] = {
                            'variant': variant_id,
                            'score': values['mean']
                        }
        
        # Generate recommendations
        for metric, winner in metric_winners.items():
            if winner['score'] > 0.8:
                recommendations.append(
                    f"Use techniques from {winner['variant']} for {metric} "
                    f"(achieves {winner['score']:.2f} score)"
                )
        
        # Add general recommendations
        if ab_results['analysis']['winner']['variant_id'] != 'baseline':
            recommendations.append(
                f"Adopt {ab_results['analysis']['winner']['variant_id']} "
                f"as the new baseline prompt"
            )
        
        return recommendations
    
    def create_final_optimized_prompt(self, base_prompt: PromptVariant,
                                    ab_results: Dict[str, Any],
                                    recommendations: List[str]) -> PromptVariant:
        """Create final optimized prompt combining best practices"""
        # Get winner variant
        winner_id = ab_results['analysis']['winner']['variant_id']
        
        # Find winner prompt
        winner_prompt = base_prompt  # Default to base
        for result in ab_results['raw_results']:
            if result['variant_id'] == winner_id:
                # In real implementation, we'd retrieve the actual variant
                break
        
        # Create final optimized prompt
        final_prompt = PromptVariant(
            id="final_optimized",
            name="Final Optimized Prompt",
            type=PromptVariantType.ENHANCED,
            system_prompt=winner_prompt.system_prompt + "\n\n" + "\n".join([
                "Optimization Guidelines:",
                "- Always validate O-RAN interface compliance",
                "- Ensure proper resource specifications",
                "- Configure high availability by default",
                "- Include comprehensive health checks",
                "- Follow 3GPP and O-RAN standards strictly"
            ]),
            user_prompt_template=winner_prompt.user_prompt_template,
            examples=winner_prompt.examples,
            metadata={
                'optimization_date': datetime.now().isoformat(),
                'based_on': winner_id,
                'improvements': recommendations
            }
        )
        
        return final_prompt
    
    async def validate_final_prompt(self, prompt: PromptVariant, 
                                   test_cases: List[TestCase]) -> Dict[str, Any]:
        """Validate the final optimized prompt"""
        logger.info("Validating final optimized prompt...")
        
        validation_results = await self.evaluate_prompts([prompt], test_cases)
        
        # Check if all metrics meet threshold
        summary = validation_results['summary']
        validation_passed = True
        
        thresholds = {
            'overall_score': 0.8,
            'technical_score': 0.75,
            'domain_score': 0.75,
            'compliance_score': 0.8
        }
        
        for metric, threshold in thresholds.items():
            if metric in summary:
                score = summary[metric]['mean']
                passed = score >= threshold
                validation_passed &= passed
                
                logger.info(f"  {metric}: {score:.3f} "
                           f"({'PASS' if passed else 'FAIL'} - threshold: {threshold})")
        
        validation_results['validation_passed'] = validation_passed
        
        return validation_results
    
    def display_results(self, title: str, results: Dict[str, Any]):
        """Display evaluation results"""
        print(f"\n{'='*60}")
        print(f"{title}")
        print(f"{'='*60}")
        
        if 'summary' in results:
            summary = results['summary']
            
            print("\nKey Metrics:")
            for metric in ['overall_score', 'technical_score', 'domain_score', 'compliance_score']:
                if metric in summary:
                    stats = summary[metric]
                    print(f"  {metric:20s}: {stats['mean']:.3f} "
                          f"(±{stats['std']:.3f})")
    
    def display_final_summary(self, baseline_results: Dict[str, Any],
                            final_results: Dict[str, Any]):
        """Display final optimization summary"""
        print(f"\n{'='*60}")
        print("OPTIMIZATION SUMMARY")
        print(f"{'='*60}")
        
        # Calculate improvements
        baseline_summary = baseline_results['summary']
        final_summary = final_results['summary']
        
        print("\nMetric Improvements:")
        print(f"{'Metric':<25} {'Baseline':>10} {'Optimized':>10} {'Change':>10}")
        print("-" * 60)
        
        for metric in ['overall_score', 'technical_score', 'domain_score', 'compliance_score']:
            if metric in baseline_summary and metric in final_summary:
                baseline_val = baseline_summary[metric]['mean']
                final_val = final_summary[metric]['mean']
                change = ((final_val - baseline_val) / baseline_val) * 100
                
                print(f"{metric:<25} {baseline_val:>10.3f} {final_val:>10.3f} "
                      f"{change:>+9.1f}%")
        
        # Validation status
        print(f"\nValidation Status: "
              f"{'PASSED' if final_results.get('validation_passed', False) else 'FAILED'}")
        
        print("\nRecommendations:")
        print("- Deploy the optimized prompt to production")
        print("- Monitor performance metrics continuously")
        print("- Run A/B tests monthly to maintain optimization")
        print("- Update prompts when new standards are released")
    
    def visualize_optimization_progress(self, results_history: List[Dict[str, Any]]):
        """Create visualization of optimization progress"""
        # This would create charts showing improvement over iterations
        pass


async def main():
    """Main demonstration function"""
    demo = PromptOptimizationDemo()
    
    # Run the complete optimization workflow
    await demo.demonstrate_optimization_workflow()
    
    print("\n" + "="*60)
    print("Prompt Optimization Demo Complete!")
    print("="*60)
    
    print("\nKey Takeaways:")
    print("1. Baseline evaluation identifies weaknesses")
    print("2. Targeted optimizations address specific metrics")
    print("3. A/B testing validates improvements")
    print("4. Final prompt combines best practices")
    print("5. Continuous monitoring ensures quality")
    
    print("\nNext Steps:")
    print("- Integrate with production LLM service")
    print("- Set up automated testing pipeline")
    print("- Configure monitoring and alerts")
    print("- Schedule regular optimization cycles")


if __name__ == "__main__":
    asyncio.run(main())