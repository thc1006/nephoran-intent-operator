#!/usr/bin/env python3
"""
Telecom-Specific Prompt Evaluation Metrics

This script provides comprehensive metrics for evaluating prompts in the telecom domain,
specifically for O-RAN and 5G network configurations.
"""

import json
import re
import yaml
from typing import Dict, List, Any, Optional, Tuple, Set
from dataclasses import dataclass, field
from enum import Enum
import logging
from collections import defaultdict
import numpy as np
from datetime import datetime
import pandas as pd
from pathlib import Path

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class ComplianceStandard(Enum):
    """Telecom compliance standards"""
    THREEGPP_R15 = "3GPP-R15"
    THREEGPP_R16 = "3GPP-R16"
    THREEGPP_R17 = "3GPP-R17"
    ORAN_WG1 = "O-RAN.WG1"
    ORAN_WG2 = "O-RAN.WG2"
    ORAN_WG3 = "O-RAN.WG3"
    ORAN_WG4 = "O-RAN.WG4"
    ORAN_WG5 = "O-RAN.WG5"
    ETSI_NFV = "ETSI-NFV"
    ETSI_MEC = "ETSI-MEC"


@dataclass
class TelecomMetrics:
    """Comprehensive metrics for telecom prompt evaluation"""
    # Technical Accuracy Metrics
    interface_correctness: float = 0.0
    protocol_accuracy: float = 0.0
    port_mapping_validity: float = 0.0
    network_function_accuracy: float = 0.0
    configuration_completeness: float = 0.0
    
    # O-RAN Specific Metrics
    oran_interface_compliance: float = 0.0
    ric_configuration_validity: float = 0.0
    xapp_compatibility: float = 0.0
    e2_node_configuration: float = 0.0
    a1_policy_correctness: float = 0.0
    o1_management_compliance: float = 0.0
    
    # 5G Core Metrics
    nf_type_accuracy: float = 0.0
    sbi_interface_validity: float = 0.0
    nrf_registration_config: float = 0.0
    slice_configuration_validity: float = 0.0
    qos_parameter_accuracy: float = 0.0
    
    # Network Slicing Metrics
    sst_validity: float = 0.0
    sd_format_correctness: float = 0.0
    slice_isolation_score: float = 0.0
    resource_allocation_efficiency: float = 0.0
    
    # Performance Metrics
    latency_optimization_score: float = 0.0
    throughput_configuration_score: float = 0.0
    scalability_readiness: float = 0.0
    ha_configuration_score: float = 0.0
    
    # Kubernetes/Cloud-Native Metrics
    k8s_manifest_validity: float = 0.0
    resource_specification_accuracy: float = 0.0
    probe_configuration_completeness: float = 0.0
    security_context_compliance: float = 0.0
    
    # Compliance Metrics
    standards_compliance_score: float = 0.0
    security_compliance_score: float = 0.0
    regulatory_compliance_score: float = 0.0
    
    # Overall Scores
    technical_score: float = 0.0
    domain_score: float = 0.0
    compliance_score: float = 0.0
    overall_score: float = 0.0
    
    def calculate_overall_scores(self):
        """Calculate aggregate scores"""
        # Technical score
        tech_metrics = [
            self.interface_correctness,
            self.protocol_accuracy,
            self.port_mapping_validity,
            self.network_function_accuracy,
            self.configuration_completeness,
            self.k8s_manifest_validity,
            self.resource_specification_accuracy
        ]
        self.technical_score = np.mean([m for m in tech_metrics if m > 0])
        
        # Domain score (O-RAN + 5G)
        domain_metrics = [
            self.oran_interface_compliance,
            self.ric_configuration_validity,
            self.xapp_compatibility,
            self.nf_type_accuracy,
            self.sbi_interface_validity,
            self.slice_configuration_validity
        ]
        self.domain_score = np.mean([m for m in domain_metrics if m > 0])
        
        # Compliance score
        compliance_metrics = [
            self.standards_compliance_score,
            self.security_compliance_score,
            self.regulatory_compliance_score
        ]
        self.compliance_score = np.mean([m for m in compliance_metrics if m > 0])
        
        # Overall score with weights
        self.overall_score = (
            0.4 * self.technical_score +
            0.35 * self.domain_score +
            0.25 * self.compliance_score
        )


class TelecomEvaluator:
    """Evaluator for telecom-specific prompt responses"""
    
    def __init__(self):
        self.oran_interfaces = {
            'A1': {'protocols': ['REST', 'HTTP'], 'ports': [9090, 8080]},
            'E2': {'protocols': ['SCTP', 'E2AP'], 'ports': [36421, 36422]},
            'O1': {'protocols': ['NETCONF', 'YANG'], 'ports': [830, 831]},
            'O2': {'protocols': ['REST'], 'ports': [8080, 443]},
            'R1': {'protocols': ['REST'], 'ports': [8888]},
            'Open Fronthaul': {'protocols': ['eCPRI', 'IEEE-1914.3'], 'ports': []}
        }
        
        self.nf_types = {
            'AMF': {'interfaces': ['N1', 'N2', 'N8', 'N11', 'N12', 'N14', 'N15', 'N22', 'N26']},
            'SMF': {'interfaces': ['N4', 'N7', 'N10', 'N11', 'N16', 'N28', 'N29']},
            'UPF': {'interfaces': ['N3', 'N4', 'N6', 'N9', 'N19']},
            'PCF': {'interfaces': ['N5', 'N7', 'N15', 'N28', 'N30', 'N31', 'N36']},
            'UDM': {'interfaces': ['N8', 'N10', 'N13', 'N21', 'N35']},
            'UDR': {'interfaces': ['N35', 'N36', 'N37']},
            'AUSF': {'interfaces': ['N12', 'N13']},
            'NRF': {'interfaces': ['Nnrf']},
            'NSSF': {'interfaces': ['N22', 'N31']},
            'NEF': {'interfaces': ['N29', 'N30', 'N33']}
        }
        
        self.slice_types = {
            1: 'eMBB',  # Enhanced Mobile Broadband
            2: 'URLLC', # Ultra-Reliable Low Latency Communications
            3: 'mMTC',  # Massive Machine Type Communications
            4: 'V2X'    # Vehicle-to-Everything
        }
        
        self.qos_5qi_values = {
            1: {'priority': 2, 'delay': 100, 'loss_rate': 1e-2, 'type': 'GBR'},
            2: {'priority': 4, 'delay': 150, 'loss_rate': 1e-3, 'type': 'GBR'},
            3: {'priority': 3, 'delay': 50, 'loss_rate': 1e-3, 'type': 'GBR'},
            5: {'priority': 1, 'delay': 100, 'loss_rate': 1e-6, 'type': 'Non-GBR'},
            6: {'priority': 6, 'delay': 300, 'loss_rate': 1e-6, 'type': 'Non-GBR'},
            7: {'priority': 7, 'delay': 100, 'loss_rate': 1e-3, 'type': 'Non-GBR'},
            8: {'priority': 8, 'delay': 300, 'loss_rate': 1e-6, 'type': 'Non-GBR'},
            9: {'priority': 9, 'delay': 300, 'loss_rate': 1e-6, 'type': 'Non-GBR'},
            65: {'priority': 0.7, 'delay': 75, 'loss_rate': 1e-2, 'type': 'GBR'},
            66: {'priority': 2, 'delay': 100, 'loss_rate': 1e-2, 'type': 'GBR'},
            67: {'priority': 1.5, 'delay': 100, 'loss_rate': 1e-3, 'type': 'GBR'},
            82: {'priority': 1.9, 'delay': 10, 'loss_rate': 1e-4, 'type': 'Delay Critical GBR'},
            83: {'priority': 2.2, 'delay': 10, 'loss_rate': 1e-4, 'type': 'Delay Critical GBR'},
            84: {'priority': 2.4, 'delay': 30, 'loss_rate': 1e-5, 'type': 'Delay Critical GBR'},
            85: {'priority': 2.1, 'delay': 5, 'loss_rate': 1e-5, 'type': 'Delay Critical GBR'}
        }
    
    def evaluate_response(self, response: Dict[str, Any], intent_type: str, 
                         domain: str) -> TelecomMetrics:
        """Evaluate a response and calculate metrics"""
        metrics = TelecomMetrics()
        
        # Evaluate based on domain
        if domain == "O-RAN":
            self._evaluate_oran_response(response, metrics)
        elif domain == "5G-Core":
            self._evaluate_5g_core_response(response, metrics)
        elif domain == "Network-Slicing":
            self._evaluate_network_slice_response(response, metrics)
        
        # Evaluate Kubernetes configuration
        self._evaluate_k8s_configuration(response, metrics)
        
        # Evaluate compliance
        self._evaluate_compliance(response, metrics)
        
        # Calculate overall scores
        metrics.calculate_overall_scores()
        
        return metrics
    
    def _evaluate_oran_response(self, response: Dict[str, Any], metrics: TelecomMetrics):
        """Evaluate O-RAN specific aspects"""
        # Check O-RAN interfaces
        interfaces_found = set()
        correct_interfaces = 0
        total_interfaces = 0
        
        if 'spec' in response and 'template' in response['spec']:
            containers = response['spec']['template'].get('spec', {}).get('containers', [])
            for container in containers:
                ports = container.get('ports', [])
                for port in ports:
                    port_name = port.get('name', '')
                    port_number = port.get('containerPort', 0)
                    protocol = port.get('protocol', '')
                    
                    # Check against O-RAN interfaces
                    for iface, details in self.oran_interfaces.items():
                        if any(p in port_name.lower() for p in [iface.lower(), 'e2ap', 'a1', 'o1']):
                            interfaces_found.add(iface)
                            total_interfaces += 1
                            
                            # Verify port and protocol
                            if (port_number in details['ports'] or not details['ports']) and \
                               (protocol in details['protocols'] or not protocol):
                                correct_interfaces += 1
        
        metrics.oran_interface_compliance = correct_interfaces / total_interfaces if total_interfaces > 0 else 0
        
        # Check RIC configuration
        if 'ric' in json.dumps(response).lower():
            metrics.ric_configuration_validity = self._check_ric_configuration(response)
        
        # Check xApp compatibility
        env_vars = self._extract_env_vars(response)
        if any('xapp' in var['name'].lower() for var in env_vars):
            metrics.xapp_compatibility = 1.0
        
        # Check E2 configuration
        if 'e2' in json.dumps(response).lower():
            metrics.e2_node_configuration = self._check_e2_configuration(response)
        
        # Check A1 policy configuration
        if 'a1_policy' in response:
            metrics.a1_policy_correctness = self._evaluate_a1_policy(response['a1_policy'])
        
        # Check O1 configuration
        if 'o1_config' in response:
            metrics.o1_management_compliance = self._evaluate_o1_config(response['o1_config'])
    
    def _evaluate_5g_core_response(self, response: Dict[str, Any], metrics: TelecomMetrics):
        """Evaluate 5G Core specific aspects"""
        # Check NF type
        labels = response.get('metadata', {}).get('labels', {})
        nf_type = labels.get('nf.5gc.3gpp.org/type', '').upper()
        
        if nf_type in self.nf_types:
            metrics.nf_type_accuracy = 1.0
            
            # Check interfaces for this NF type
            expected_interfaces = self.nf_types[nf_type]['interfaces']
            # This would need more detailed checking in production
        else:
            metrics.nf_type_accuracy = 0.0
        
        # Check SBI interface
        if self._check_sbi_interface(response):
            metrics.sbi_interface_validity = 1.0
        
        # Check NRF registration
        env_vars = self._extract_env_vars(response)
        if any('nrf' in var['name'].lower() for var in env_vars):
            metrics.nrf_registration_config = 1.0
        
        # Check slice configuration
        if 'network_slice' in response:
            metrics.slice_configuration_validity = self._evaluate_slice_config(response['network_slice'])
        
        # Check QoS parameters
        if 'qos_parameters' in response:
            metrics.qos_parameter_accuracy = self._evaluate_qos_parameters(response['qos_parameters'])
    
    def _evaluate_network_slice_response(self, response: Dict[str, Any], metrics: TelecomMetrics):
        """Evaluate network slice configuration"""
        if 'network_slice' not in response:
            return
        
        slice_config = response['network_slice']
        
        # Check SST validity
        if 'sst' in slice_config:
            try:
                sst = int(slice_config['sst'])
                if sst in self.slice_types:
                    metrics.sst_validity = 1.0
                elif 1 <= sst <= 255:  # Valid range
                    metrics.sst_validity = 0.8
            except:
                metrics.sst_validity = 0.0
        
        # Check SD format
        if 'sd' in slice_config:
            sd = slice_config['sd']
            # SD should be 3 bytes (6 hex chars)
            if isinstance(sd, str) and len(sd) == 6 and all(c in '0123456789abcdefABCDEF' for c in sd):
                metrics.sd_format_correctness = 1.0
            else:
                metrics.sd_format_correctness = 0.5
        
        # Check slice isolation
        if 'isolation_level' in slice_config:
            isolation = slice_config['isolation_level']
            if isolation in ['physical', 'logical', 'no_isolation']:
                metrics.slice_isolation_score = {'physical': 1.0, 'logical': 0.7, 'no_isolation': 0.3}[isolation]
        
        # Check resource allocation
        if 'resource_allocation' in slice_config:
            metrics.resource_allocation_efficiency = self._evaluate_resource_allocation(
                slice_config['resource_allocation']
            )
    
    def _evaluate_k8s_configuration(self, response: Dict[str, Any], metrics: TelecomMetrics):
        """Evaluate Kubernetes configuration aspects"""
        # Check manifest validity
        required_fields = ['apiVersion', 'kind', 'metadata', 'spec']
        if all(field in response for field in required_fields):
            metrics.k8s_manifest_validity = 1.0
        else:
            present = sum(1 for field in required_fields if field in response)
            metrics.k8s_manifest_validity = present / len(required_fields)
        
        # Check resource specifications
        if 'spec' in response and 'template' in response['spec']:
            containers = response['spec']['template'].get('spec', {}).get('containers', [])
            if containers:
                container = containers[0]
                resources = container.get('resources', {})
                
                # Check for requests and limits
                has_requests = 'requests' in resources
                has_limits = 'limits' in resources
                has_cpu = has_requests and 'cpu' in resources['requests']
                has_memory = has_requests and 'memory' in resources['requests']
                
                metrics.resource_specification_accuracy = sum([has_requests, has_limits, has_cpu, has_memory]) / 4
                
                # Check probes
                has_liveness = 'livenessProbe' in container
                has_readiness = 'readinessProbe' in container
                metrics.probe_configuration_completeness = sum([has_liveness, has_readiness]) / 2
                
                # Check security context
                if 'securityContext' in container or 'securityContext' in response['spec']['template']['spec']:
                    metrics.security_context_compliance = 1.0
        
        # Check high availability configuration
        if 'spec' in response and 'replicas' in response['spec']:
            replicas = response['spec']['replicas']
            if replicas >= 3:
                metrics.ha_configuration_score = 1.0
            elif replicas == 2:
                metrics.ha_configuration_score = 0.7
            else:
                metrics.ha_configuration_score = 0.3
    
    def _evaluate_compliance(self, response: Dict[str, Any], metrics: TelecomMetrics):
        """Evaluate compliance with standards"""
        response_str = json.dumps(response).lower()
        
        # Check for standards references
        standards_found = 0
        standards_checked = ['3gpp', 'o-ran', 'etsi', 'ietf']
        for standard in standards_checked:
            if standard in response_str:
                standards_found += 1
        
        metrics.standards_compliance_score = standards_found / len(standards_checked)
        
        # Check security compliance
        security_features = 0
        if 'tls' in response_str or 'https' in response_str:
            security_features += 1
        if 'oauth' in response_str or 'authentication' in response_str:
            security_features += 1
        if 'encryption' in response_str or 'encrypted' in response_str:
            security_features += 1
        
        metrics.security_compliance_score = min(security_features / 3, 1.0)
        
        # Check regulatory compliance (simplified)
        if any(reg in response_str for reg in ['gdpr', 'regulatory', 'compliance', 'lawful']):
            metrics.regulatory_compliance_score = 0.8
    
    def _check_ric_configuration(self, response: Dict[str, Any]) -> float:
        """Check RIC configuration validity"""
        score = 0.0
        checks = 0
        
        # Check for RIC-specific environment variables
        env_vars = self._extract_env_vars(response)
        ric_vars = ['RIC_ID', 'E2_TERM_INIT', 'XAPP_ONBOARDER_ENABLED', 'CONFLICT_MITIGATION']
        
        for var in ric_vars:
            checks += 1
            if any(v['name'] == var for v in env_vars):
                score += 1
        
        return score / checks if checks > 0 else 0.0
    
    def _check_e2_configuration(self, response: Dict[str, Any]) -> float:
        """Check E2 interface configuration"""
        score = 0.0
        
        # Check for E2AP port
        if self._has_port(response, 36421, 'SCTP'):
            score += 0.5
        
        # Check for E2 related config
        env_vars = self._extract_env_vars(response)
        if any('e2' in var['name'].lower() for var in env_vars):
            score += 0.5
        
        return score
    
    def _evaluate_a1_policy(self, a1_policy: Dict[str, Any]) -> float:
        """Evaluate A1 policy configuration"""
        required_fields = ['policy_type_id', 'policy_data']
        score = 0.0
        
        for field in required_fields:
            if field in a1_policy:
                score += 0.5
        
        # Check policy data structure
        if 'policy_data' in a1_policy:
            policy_data = a1_policy['policy_data']
            if 'scope' in policy_data and 'qos_parameters' in policy_data:
                score = min(score + 0.5, 1.0)
        
        return score
    
    def _evaluate_o1_config(self, o1_config: str) -> float:
        """Evaluate O1 interface configuration"""
        try:
            # Check if it's valid XML (O1 uses NETCONF/YANG which is XML-based)
            if o1_config.startswith('<?xml'):
                score = 0.5
                
                # Check for O-RAN namespace
                if 'urn:o-ran' in o1_config:
                    score += 0.3
                
                # Check for proper structure
                if '<config' in o1_config and '</config>' in o1_config:
                    score += 0.2
                
                return score
        except:
            pass
        
        return 0.0
    
    def _check_sbi_interface(self, response: Dict[str, Any]) -> bool:
        """Check for Service Based Interface configuration"""
        # Look for SBI port (typically 8080 for HTTP/2)
        return self._has_port(response, 8080, 'TCP') or self._has_port(response, 443, 'TCP')
    
    def _evaluate_slice_config(self, slice_config: Dict[str, Any]) -> float:
        """Evaluate network slice configuration"""
        score = 0.0
        checks = 0
        
        # Check SST
        if 'sst' in slice_config:
            checks += 1
            try:
                sst = int(slice_config['sst'])
                if sst in self.slice_types:
                    score += 1
            except:
                pass
        
        # Check PLMN ID format
        if 'plmn_id' in slice_config:
            checks += 1
            plmn = slice_config['plmn_id']
            if isinstance(plmn, str) and len(plmn) in [5, 6]:
                score += 1
        
        return score / checks if checks > 0 else 0.0
    
    def _evaluate_qos_parameters(self, qos_params: Dict[str, Any]) -> float:
        """Evaluate QoS parameter configuration"""
        score = 0.0
        checks = 0
        
        # Check 5QI value
        if '5qi' in qos_params or 'qi' in qos_params:
            checks += 1
            qi_value = qos_params.get('5qi', qos_params.get('qi', 0))
            if qi_value in self.qos_5qi_values:
                score += 1
        
        # Check delay budget
        if 'delay_budget' in qos_params or 'packet_delay_budget' in qos_params:
            checks += 1
            delay = qos_params.get('delay_budget', qos_params.get('packet_delay_budget', ''))
            if isinstance(delay, (int, float)) or 'ms' in str(delay):
                score += 1
        
        # Check guaranteed bit rate
        if 'guaranteed_bitrate' in qos_params or 'gbr' in qos_params:
            checks += 1
            bitrate = qos_params.get('guaranteed_bitrate', qos_params.get('gbr', ''))
            if self._is_valid_bitrate(bitrate):
                score += 1
        
        return score / checks if checks > 0 else 0.0
    
    def _evaluate_resource_allocation(self, resource_alloc: Dict[str, Any]) -> float:
        """Evaluate resource allocation configuration"""
        score = 0.0
        checks = 0
        
        # Check CPU allocation
        if 'cpu' in resource_alloc or 'cpu_allocation' in resource_alloc:
            checks += 1
            cpu = resource_alloc.get('cpu', resource_alloc.get('cpu_allocation', ''))
            if self._is_valid_cpu_spec(cpu):
                score += 1
        
        # Check memory allocation
        if 'memory' in resource_alloc or 'memory_allocation' in resource_alloc:
            checks += 1
            memory = resource_alloc.get('memory', resource_alloc.get('memory_allocation', ''))
            if self._is_valid_memory_spec(memory):
                score += 1
        
        # Check bandwidth allocation
        if 'bandwidth' in resource_alloc or 'bandwidth_allocation' in resource_alloc:
            checks += 1
            bandwidth = resource_alloc.get('bandwidth', resource_alloc.get('bandwidth_allocation', ''))
            if self._is_valid_bitrate(bandwidth):
                score += 1
        
        return score / checks if checks > 0 else 0.0
    
    def _extract_env_vars(self, response: Dict[str, Any]) -> List[Dict[str, str]]:
        """Extract environment variables from response"""
        env_vars = []
        
        if 'spec' in response and 'template' in response['spec']:
            containers = response['spec']['template'].get('spec', {}).get('containers', [])
            for container in containers:
                env = container.get('env', [])
                env_vars.extend(env)
        
        return env_vars
    
    def _has_port(self, response: Dict[str, Any], port_number: int, protocol: str) -> bool:
        """Check if response has a specific port configuration"""
        if 'spec' in response and 'template' in response['spec']:
            containers = response['spec']['template'].get('spec', {}).get('containers', [])
            for container in containers:
                ports = container.get('ports', [])
                for port in ports:
                    if port.get('containerPort') == port_number and \
                       port.get('protocol', 'TCP') == protocol:
                        return True
        return False
    
    def _is_valid_bitrate(self, bitrate: Any) -> bool:
        """Check if bitrate specification is valid"""
        if isinstance(bitrate, (int, float)):
            return True
        
        if isinstance(bitrate, str):
            # Check for common bitrate formats
            patterns = [r'\d+\s*[KMG]?bps', r'\d+\s*[KMG]?b/s', r'\d+\s*[KMG]?bit/s']
            return any(re.match(pattern, bitrate, re.IGNORECASE) for pattern in patterns)
        
        return False
    
    def _is_valid_cpu_spec(self, cpu: Any) -> bool:
        """Check if CPU specification is valid"""
        if isinstance(cpu, (int, float)):
            return True
        
        if isinstance(cpu, str):
            # Check for millicores or percentage
            return bool(re.match(r'\d+m?$', cpu)) or bool(re.match(r'\d+%$', cpu))
        
        return False
    
    def _is_valid_memory_spec(self, memory: Any) -> bool:
        """Check if memory specification is valid"""
        if isinstance(memory, (int, float)):
            return True
        
        if isinstance(memory, str):
            # Check for memory formats
            return bool(re.match(r'\d+[KMGTPE]i?$', memory))
        
        return False


class MetricsAggregator:
    """Aggregates and analyzes metrics across multiple evaluations"""
    
    def __init__(self):
        self.evaluations = []
        
    def add_evaluation(self, prompt_id: str, test_case_id: str, 
                      metrics: TelecomMetrics, metadata: Dict[str, Any] = None):
        """Add an evaluation result"""
        self.evaluations.append({
            'prompt_id': prompt_id,
            'test_case_id': test_case_id,
            'metrics': metrics,
            'metadata': metadata or {},
            'timestamp': datetime.now()
        })
    
    def get_summary_statistics(self) -> Dict[str, Any]:
        """Get summary statistics for all evaluations"""
        if not self.evaluations:
            return {}
        
        # Extract all metrics
        metrics_data = defaultdict(list)
        
        for eval in self.evaluations:
            metrics_dict = vars(eval['metrics'])
            for metric_name, value in metrics_dict.items():
                if isinstance(value, (int, float)):
                    metrics_data[metric_name].append(value)
        
        # Calculate statistics
        summary = {}
        for metric_name, values in metrics_data.items():
            if values:
                summary[metric_name] = {
                    'mean': np.mean(values),
                    'std': np.std(values),
                    'min': np.min(values),
                    'max': np.max(values),
                    'median': np.median(values),
                    'p25': np.percentile(values, 25),
                    'p75': np.percentile(values, 75)
                }
        
        return summary
    
    def get_prompt_comparison(self) -> pd.DataFrame:
        """Compare metrics across different prompts"""
        data = []
        
        # Group by prompt_id
        prompt_groups = defaultdict(list)
        for eval in self.evaluations:
            prompt_groups[eval['prompt_id']].append(eval)
        
        # Calculate average metrics per prompt
        for prompt_id, evals in prompt_groups.items():
            metrics_sum = defaultdict(float)
            count = len(evals)
            
            for eval in evals:
                metrics_dict = vars(eval['metrics'])
                for metric_name, value in metrics_dict.items():
                    if isinstance(value, (int, float)):
                        metrics_sum[metric_name] += value
            
            # Calculate averages
            row = {'prompt_id': prompt_id}
            for metric_name, total in metrics_sum.items():
                row[metric_name] = total / count
            
            data.append(row)
        
        return pd.DataFrame(data)
    
    def get_domain_analysis(self) -> Dict[str, Any]:
        """Analyze metrics by domain (O-RAN, 5G-Core, etc.)"""
        domain_metrics = defaultdict(lambda: defaultdict(list))
        
        for eval in self.evaluations:
            domain = eval['metadata'].get('domain', 'unknown')
            metrics_dict = vars(eval['metrics'])
            
            for metric_name, value in metrics_dict.items():
                if isinstance(value, (int, float)):
                    domain_metrics[domain][metric_name].append(value)
        
        # Calculate statistics per domain
        analysis = {}
        for domain, metrics in domain_metrics.items():
            domain_stats = {}
            for metric_name, values in metrics.items():
                if values:
                    domain_stats[metric_name] = {
                        'mean': np.mean(values),
                        'std': np.std(values),
                        'count': len(values)
                    }
            analysis[domain] = domain_stats
        
        return analysis
    
    def identify_weaknesses(self, threshold: float = 0.7) -> List[Dict[str, Any]]:
        """Identify metrics that consistently score below threshold"""
        metrics_scores = defaultdict(list)
        
        for eval in self.evaluations:
            metrics_dict = vars(eval['metrics'])
            for metric_name, value in metrics_dict.items():
                if isinstance(value, (int, float)):
                    metrics_scores[metric_name].append(value)
        
        weaknesses = []
        for metric_name, scores in metrics_scores.items():
            if scores:
                avg_score = np.mean(scores)
                if avg_score < threshold:
                    weaknesses.append({
                        'metric': metric_name,
                        'average_score': avg_score,
                        'std_dev': np.std(scores),
                        'samples': len(scores),
                        'improvement_needed': threshold - avg_score
                    })
        
        # Sort by improvement needed
        weaknesses.sort(key=lambda x: x['improvement_needed'], reverse=True)
        
        return weaknesses
    
    def export_metrics(self, filepath: Path):
        """Export metrics to file for further analysis"""
        # Convert to DataFrame
        data = []
        for eval in self.evaluations:
            row = {
                'prompt_id': eval['prompt_id'],
                'test_case_id': eval['test_case_id'],
                'timestamp': eval['timestamp'].isoformat()
            }
            
            # Add metrics
            metrics_dict = vars(eval['metrics'])
            row.update(metrics_dict)
            
            # Add metadata
            for key, value in eval['metadata'].items():
                row[f'meta_{key}'] = value
            
            data.append(row)
        
        df = pd.DataFrame(data)
        
        # Export based on file extension
        if filepath.suffix == '.csv':
            df.to_csv(filepath, index=False)
        elif filepath.suffix == '.json':
            df.to_json(filepath, orient='records', indent=2)
        elif filepath.suffix in ['.xlsx', '.xls']:
            df.to_excel(filepath, index=False)
        else:
            raise ValueError(f"Unsupported file format: {filepath.suffix}")


def evaluate_prompt_response(response: Dict[str, Any], test_case: Dict[str, Any]) -> TelecomMetrics:
    """Evaluate a single prompt response"""
    evaluator = TelecomEvaluator()
    
    metrics = evaluator.evaluate_response(
        response=response,
        intent_type=test_case.get('intent_type', 'unknown'),
        domain=test_case.get('domain', 'unknown')
    )
    
    return metrics


def generate_evaluation_report(aggregator: MetricsAggregator, output_dir: Path):
    """Generate comprehensive evaluation report"""
    output_dir.mkdir(parents=True, exist_ok=True)
    
    # Get summary statistics
    summary = aggregator.get_summary_statistics()
    
    # Create report
    report = {
        'evaluation_summary': {
            'total_evaluations': len(aggregator.evaluations),
            'evaluation_period': {
                'start': min(e['timestamp'] for e in aggregator.evaluations).isoformat(),
                'end': max(e['timestamp'] for e in aggregator.evaluations).isoformat()
            }
        },
        'overall_metrics': {},
        'domain_analysis': aggregator.get_domain_analysis(),
        'identified_weaknesses': aggregator.identify_weaknesses(),
        'recommendations': []
    }
    
    # Add overall metrics
    key_metrics = ['overall_score', 'technical_score', 'domain_score', 'compliance_score']
    for metric in key_metrics:
        if metric in summary:
            report['overall_metrics'][metric] = summary[metric]
    
    # Generate recommendations
    weaknesses = aggregator.identify_weaknesses()
    for weakness in weaknesses[:5]:  # Top 5 weaknesses
        metric_name = weakness['metric']
        score = weakness['average_score']
        
        if 'oran' in metric_name:
            report['recommendations'].append(
                f"Improve O-RAN compliance: {metric_name} scores {score:.2f}, "
                f"enhance O-RAN interface configurations and standards compliance"
            )
        elif '5g' in metric_name or 'nf' in metric_name:
            report['recommendations'].append(
                f"Enhance 5G Core accuracy: {metric_name} scores {score:.2f}, "
                f"improve network function configurations and SBI interfaces"
            )
        elif 'slice' in metric_name:
            report['recommendations'].append(
                f"Improve network slicing: {metric_name} scores {score:.2f}, "
                f"enhance S-NSSAI configuration and slice isolation"
            )
    
    # Save report
    report_path = output_dir / 'evaluation_report.json'
    with open(report_path, 'w') as f:
        json.dump(report, f, indent=2)
    
    # Export detailed metrics
    aggregator.export_metrics(output_dir / 'detailed_metrics.csv')
    
    # Create comparison DataFrame
    comparison_df = aggregator.get_prompt_comparison()
    comparison_df.to_csv(output_dir / 'prompt_comparison.csv', index=False)
    
    logger.info(f"Evaluation report generated at {output_dir}")


# Example usage
if __name__ == "__main__":
    # Create sample response
    sample_response = {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "metadata": {
            "name": "near-rt-ric",
            "namespace": "oran-system",
            "labels": {
                "oran.component": "ric",
                "oran.type": "near-rt"
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
                        ],
                        "env": [
                            {"name": "RIC_ID", "value": "ric-001"},
                            {"name": "E2_TERM_INIT", "value": "true"}
                        ],
                        "resources": {
                            "requests": {"cpu": "2", "memory": "4Gi"},
                            "limits": {"cpu": "4", "memory": "8Gi"}
                        }
                    }]
                }
            }
        }
    }
    
    # Create test case
    test_case = {
        'intent_type': 'NetworkFunctionDeployment',
        'domain': 'O-RAN'
    }
    
    # Evaluate
    metrics = evaluate_prompt_response(sample_response, test_case)
    
    # Print metrics
    print(f"Overall Score: {metrics.overall_score:.3f}")
    print(f"Technical Score: {metrics.technical_score:.3f}")
    print(f"Domain Score: {metrics.domain_score:.3f}")
    print(f"Compliance Score: {metrics.compliance_score:.3f}")
    
    # Create aggregator and add evaluation
    aggregator = MetricsAggregator()
    aggregator.add_evaluation("test_prompt", "test_case_1", metrics, {'domain': 'O-RAN'})
    
    # Generate report
    generate_evaluation_report(aggregator, Path("evaluation_results"))