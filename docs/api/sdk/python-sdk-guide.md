# Nephoran Intent Operator Python SDK Guide

## Overview

The Nephoran Intent Operator Python SDK provides a comprehensive, production-ready client library for interacting with the Nephoran Intent Operator API. Built with enterprise requirements in mind, it offers:

- **Async/await support** for high-performance applications
- **Type hints and Pydantic models** for robust development
- **OAuth2 authentication** with automatic token refresh
- **Retry mechanisms** with exponential backoff
- **Circuit breaker patterns** for fault tolerance
- **Comprehensive logging** and distributed tracing
- **Connection pooling** and resource management
- **Full OpenAPI 3.0 compliance**

## Installation

```bash
# Install via pip
pip install nephoran-intent-operator-sdk

# Install with async support
pip install "nephoran-intent-operator-sdk[async]"

# Install development version
pip install "nephoran-intent-operator-sdk[dev]"
```

## Quick Start

### Basic Synchronous Client

```python
import os
from nephoran_sdk import NephoranClient
from nephoran_sdk.models import NetworkIntent, NetworkIntentSpec, ObjectMeta

def main():
    # Initialize client with OAuth2 authentication
    client = NephoranClient(
        base_url="https://api.nephoran.com/v1",
        client_id=os.getenv("NEPHORAN_CLIENT_ID"),
        client_secret=os.getenv("NEPHORAN_CLIENT_SECRET"),
        auth_url="https://auth.nephoran.com/oauth/token",
        scopes=["intent.read", "intent.write", "intent.execute"]
    )

    # Create a network intent
    intent = NetworkIntent(
        apiVersion="nephoran.io/v1alpha1",
        kind="NetworkIntent",
        metadata=ObjectMeta(
            name="amf-production",
            namespace="telecom-5g",
            labels={
                "environment": "production",
                "component": "amf"
            }
        ),
        spec=NetworkIntentSpec(
            intent="Deploy a high-availability AMF instance for production with auto-scaling",
            priority="high",
            parameters={
                "replicas": 3,
                "enableAutoScaling": True,
                "maxConcurrentConnections": 10000
            }
        )
    )

    try:
        # Create the intent
        result = client.intents.create(intent)
        print(f"Intent created: {result.metadata.name} (UID: {result.metadata.uid})")
        
        # Monitor processing status
        status = client.intents.get_status(result.metadata.namespace, result.metadata.name)
        print(f"Status: {status.phase} - {status.message}")
        
    except Exception as e:
        print(f"Error creating intent: {e}")
    finally:
        client.close()

if __name__ == "__main__":
    main()
```

### Asynchronous Client

```python
import asyncio
import os
from nephoran_sdk.async_client import AsyncNephoranClient
from nephoran_sdk.models import NetworkIntent, NetworkIntentSpec, ObjectMeta

async def main():
    # Initialize async client
    async with AsyncNephoranClient(
        base_url="https://api.nephoran.com/v1",
        client_id=os.getenv("NEPHORAN_CLIENT_ID"),
        client_secret=os.getenv("NEPHORAN_CLIENT_SECRET"),
        auth_url="https://auth.nephoran.com/oauth/token",
        scopes=["intent.read", "intent.write", "rag.query"],
        timeout=60.0
    ) as client:
        
        # Create multiple intents concurrently
        intents = [
            NetworkIntent(
                apiVersion="nephoran.io/v1alpha1",
                kind="NetworkIntent",
                metadata=ObjectMeta(
                    name=f"amf-instance-{i}",
                    namespace="telecom-5g"
                ),
                spec=NetworkIntentSpec(
                    intent=f"Deploy AMF instance {i} with load balancing",
                    priority="medium"
                )
            )
            for i in range(5)
        ]
        
        # Create all intents concurrently
        tasks = [client.intents.create(intent) for intent in intents]
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        success_count = sum(1 for r in results if not isinstance(r, Exception))
        print(f"Successfully created {success_count}/{len(intents)} intents")

if __name__ == "__main__":
    asyncio.run(main())
```

## Advanced Client Configuration

### Production Configuration

```python
import logging
from nephoran_sdk import NephoranClient
from nephoran_sdk.config import ClientConfig, AuthConfig, RetryConfig, CircuitBreakerConfig

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def create_production_client():
    """Create a production-ready client with comprehensive configuration."""
    
    config = ClientConfig(
        base_url="https://api.nephoran.com/v1",
        auth=AuthConfig(
            client_id=os.getenv("NEPHORAN_CLIENT_ID"),
            client_secret=os.getenv("NEPHORAN_CLIENT_SECRET"),
            token_url="https://auth.nephoran.com/oauth/token",
            scopes=["intent.read", "intent.write", "intent.execute", "rag.query"],
            token_cache_ttl=3600,  # Cache tokens for 1 hour
            auto_refresh=True
        ),
        retry=RetryConfig(
            max_attempts=3,
            backoff_factor=2.0,
            max_backoff=60.0,
            retryable_status_codes=[429, 500, 502, 503, 504],
            retryable_exceptions=[
                "requests.exceptions.ConnectionError",
                "requests.exceptions.Timeout"
            ]
        ),
        circuit_breaker=CircuitBreakerConfig(
            failure_threshold=5,
            timeout=30.0,
            expected_exception=Exception
        ),
        timeout=60.0,
        max_connections=100,
        max_connections_per_host=20,
        enable_http2=True,
        verify_ssl=True,
        user_agent="MyApp/1.0.0 (Nephoran SDK/1.0.0)"
    )
    
    return NephoranClient(config=config)
```

## Core API Operations

### Intent Management

#### Creating Different Types of Intents

```python
from nephoran_sdk.models import (
    NetworkIntent, NetworkSlice, RANIntent,
    NetworkIntentSpec, ObjectMeta
)

def create_5g_core_intents(client):
    """Create various 5G Core network function intents."""
    
    # AMF (Access and Mobility Management Function)
    amf_intent = NetworkIntent(
        apiVersion="nephoran.io/v1alpha1",
        kind="NetworkIntent",
        metadata=ObjectMeta(
            name="production-amf-cluster",
            namespace="telecom-5g-core",
            labels={
                "environment": "production",
                "nf-type": "amf",
                "3gpp-release": "rel16",
                "deployment-tier": "critical"
            },
            annotations={
                "nephoran.io/sla-target": "99.99%",
                "nephoran.io/max-downtime": "52.56-minutes-per-year"
            }
        ),
        spec=NetworkIntentSpec(
            intent=(
                "Deploy high-availability AMF cluster with N1/N2 interface support "
                "for 500,000 concurrent UE registrations with sub-second response time"
            ),
            priority="critical",
            parameters={
                "replicas": 7,
                "enableAutoScaling": True,
                "minReplicas": 5,
                "maxReplicas": 15,
                "cpuTargetUtilization": 70,
                "maxConcurrentUEs": 500000,
                "interfaces": {
                    "n1": {"enabled": True, "port": 38412},
                    "n2": {"enabled": True, "port": 38422}
                },
                "sessionPersistence": {
                    "enabled": True,
                    "backend": "redis-cluster",
                    "replication": 3
                },
                "loadBalancing": {
                    "algorithm": "consistent-hash",
                    "healthCheck": {
                        "interval": "5s",
                        "timeout": "2s",
                        "retries": 3
                    }
                },
                "security": {
                    "mtls": True,
                    "oauth2": True,
                    "rateLimiting": {
                        "requestsPerSecond": 10000,
                        "burstSize": 20000
                    }
                }
            },
            targetClusters=["cluster-east-1", "cluster-west-1", "cluster-central"],
            oranCompliance=True
        )
    )
    
    # SMF (Session Management Function)
    smf_intent = NetworkIntent(
        apiVersion="nephoran.io/v1alpha1",
        kind="NetworkIntent", 
        metadata=ObjectMeta(
            name="production-smf-cluster",
            namespace="telecom-5g-core",
            labels={
                "nf-type": "smf",
                "deployment-tier": "critical"
            }
        ),
        spec=NetworkIntentSpec(
            intent=(
                "Deploy SMF cluster with UPF integration supporting 1M active PDU sessions "
                "with advanced QoS management and network slice awareness"
            ),
            priority="critical", 
            parameters={
                "replicas": 5,
                "maxActivePDUSessions": 1000000,
                "upfIntegration": {
                    "enabled": True,
                    "discoveryProtocol": "pfcp",
                    "maxUPFs": 50
                },
                "qosManagement": {
                    "enabled": True,
                    "flowControlEnabled": True,
                    "trafficShaping": True
                },
                "networkSlicing": {
                    "enabled": True,
                    "maxSlices": 100,
                    "sliceTypes": ["eMBB", "URLLC", "mMTC"]
                }
            }
        )
    )
    
    # UPF (User Plane Function) 
    upf_intent = NetworkIntent(
        apiVersion="nephoran.io/v1alpha1",
        kind="NetworkIntent",
        metadata=ObjectMeta(
            name="edge-upf-deployment", 
            namespace="telecom-5g-core",
            labels={"nf-type": "upf", "deployment-type": "edge"}
        ),
        spec=NetworkIntentSpec(
            intent=(
                "Deploy edge UPF instances with 100Gbps throughput capacity "
                "and low-latency data plane processing for edge computing workloads"
            ),
            priority="high",
            parameters={
                "throughputCapacity": "100Gbps",
                "latencyTarget": "1ms",
                "edgeOptimized": True,
                "dataPlane": {
                    "technology": "dpdk",
                    "acceleration": "sr-iov",
                    "cpuCores": 16
                },
                "interfaces": {
                    "n3": {"enabled": True},
                    "n4": {"enabled": True, "pfcpEnabled": True},
                    "n6": {"enabled": True},
                    "n9": {"enabled": True}
                }
            }
        )
    )
    
    # Create all intents
    intents_to_create = [amf_intent, smf_intent, upf_intent]
    created_intents = []
    
    for intent in intents_to_create:
        try:
            result = client.intents.create(intent)
            created_intents.append(result)
            print(f"✓ Created {intent.metadata.name}: {result.metadata.uid}")
        except Exception as e:
            print(f"✗ Failed to create {intent.metadata.name}: {e}")
    
    return created_intents

def create_network_slice_intent(client):
    """Create advanced network slice with comprehensive SLA requirements."""
    
    slice_intent = NetworkSlice(
        apiVersion="nephoran.io/v1alpha1",
        kind="NetworkSlice",
        metadata=ObjectMeta(
            name="automotive-urllc-slice",
            namespace="network-slicing",
            labels={
                "slice-type": "URLLC", 
                "use-case": "automotive",
                "sla-tier": "premium"
            }
        ),
        spec=NetworkIntentSpec(
            intent=(
                "Create URLLC network slice for autonomous vehicle communication "
                "with 1ms latency, 99.9999% reliability, and dedicated resources"
            ),
            priority="critical",
            parameters={
                "sliceType": "URLLC",
                "useCase": "automotive-v2x",
                "sla": {
                    "latency": "1ms",
                    "reliability": "99.9999%",
                    "availability": "99.999%",
                    "throughput": "10Gbps"
                },
                "resourceAllocation": {
                    "dedicatedSpectrum": True,
                    "cpuReservation": "50%",
                    "memoryReservation": "40%",
                    "networkBandwidth": "dedicated"
                },
                "qosProfile": {
                    "5qi": 82,  # V2X specific QoS identifier
                    "arp": {
                        "priorityLevel": 1,
                        "preemptionCapability": "may-preempt",
                        "preemptionVulnerability": "not-preemptable"
                    }
                },
                "security": {
                    "isolation": "hard",
                    "encryption": "end-to-end",
                    "authentication": "certificate-based"
                }
            },
            networkSlice="s-nssai-01-000001"
        )
    )
    
    return client.intents.create(slice_intent)

def create_oran_near_rt_ric_intent(client):
    """Create O-RAN Near-RT RIC with comprehensive xApp ecosystem."""
    
    ric_intent = RANIntent(
        apiVersion="nephoran.io/v1alpha1",
        kind="RANIntent", 
        metadata=ObjectMeta(
            name="production-near-rt-ric",
            namespace="o-ran-ric",
            labels={
                "component": "near-rt-ric",
                "oran-release": "r5",
                "deployment": "production"
            }
        ),
        spec=NetworkIntentSpec(
            intent=(
                "Deploy production Near-RT RIC with comprehensive xApp ecosystem "
                "for intelligent RAN optimization, traffic steering, and QoS management"
            ),
            priority="high",
            parameters={
                "platform": {
                    "version": "1.3.6",
                    "enableHA": True,
                    "persistence": True
                },
                "xApps": [
                    {
                        "name": "traffic-steering",
                        "version": "1.2.0",
                        "config": {
                            "algorithm": "load-based",
                            "updateInterval": "100ms"
                        }
                    },
                    {
                        "name": "qos-optimizer", 
                        "version": "1.1.5",
                        "config": {
                            "optimizationTarget": "latency",
                            "mlModel": "reinforcement-learning"
                        }
                    },
                    {
                        "name": "anomaly-detector",
                        "version": "1.0.8", 
                        "config": {
                            "detectionMethod": "statistical + ml",
                            "alertThreshold": 0.95
                        }
                    },
                    {
                        "name": "energy-optimizer",
                        "version": "1.0.3",
                        "config": {
                            "strategy": "predictive",
                            "savingsTarget": "20%"
                        }
                    }
                ],
                "e2Interfaces": [
                    {
                        "version": "E2-KPM",
                        "enabled": True,
                        "reportingPeriod": "1s"
                    },
                    {
                        "version": "E2-RC",
                        "enabled": True,
                        "controlActions": ["handover", "admission", "qos"]
                    },
                    {
                        "version": "E2-NI",
                        "enabled": True,
                        "nodeTypes": ["gNB", "eNB"]
                    }
                ],
                "a1Policies": [
                    {
                        "type": "traffic_steering_policy",
                        "version": "1.0",
                        "enforcement": "mandatory"
                    },
                    {
                        "type": "qos_management_policy", 
                        "version": "1.0",
                        "enforcement": "advisory"
                    }
                ],
                "messagingBus": {
                    "type": "rmr",
                    "instances": 5,
                    "persistence": True
                }
            },
            oranCompliance=True
        )
    )
    
    return client.intents.create(ric_intent)
```

#### Advanced Intent Operations

```python
from typing import List, Optional, Dict, Any
from nephoran_sdk.models import NetworkIntent, ListOptions, PaginatedResponse
import time

class IntentManager:
    """Advanced intent management with monitoring and lifecycle operations."""
    
    def __init__(self, client):
        self.client = client
        
    def list_intents_with_filters(
        self, 
        namespace: Optional[str] = None,
        labels: Optional[Dict[str, str]] = None,
        status: Optional[str] = None,
        limit: int = 20
    ) -> PaginatedResponse[NetworkIntent]:
        """List intents with comprehensive filtering options."""
        
        options = ListOptions(
            namespace=namespace,
            limit=limit,
            label_selector=self._build_label_selector(labels),
            field_selector=f"status.phase={status}" if status else None
        )
        
        return self.client.intents.list(options)
    
    def wait_for_intent_completion(
        self, 
        namespace: str, 
        name: str, 
        timeout: int = 300,
        poll_interval: int = 5
    ) -> NetworkIntent:
        """Wait for intent to reach completion with timeout."""
        
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            intent = self.client.intents.get(namespace, name)
            
            if intent.status.phase in ["Deployed", "Failed"]:
                return intent
            
            print(f"Intent {name} status: {intent.status.phase}")
            time.sleep(poll_interval)
            
        raise TimeoutError(f"Intent {name} did not complete within {timeout} seconds")
    
    def bulk_create_intents(
        self, 
        intents: List[NetworkIntent],
        max_concurrent: int = 5,
        fail_fast: bool = False
    ) -> List[Dict[str, Any]]:
        """Create multiple intents with controlled concurrency."""
        
        import concurrent.futures
        import threading
        
        results = []
        semaphore = threading.Semaphore(max_concurrent)
        
        def create_intent_with_semaphore(intent):
            with semaphore:
                try:
                    result = self.client.intents.create(intent)
                    return {
                        "success": True,
                        "intent": result,
                        "error": None
                    }
                except Exception as e:
                    if fail_fast:
                        raise
                    return {
                        "success": False,
                        "intent": intent,
                        "error": str(e)
                    }
        
        with concurrent.futures.ThreadPoolExecutor(max_workers=max_concurrent) as executor:
            future_to_intent = {
                executor.submit(create_intent_with_semaphore, intent): intent
                for intent in intents
            }
            
            for future in concurrent.futures.as_completed(future_to_intent):
                result = future.result()
                results.append(result)
                
                if not result["success"]:
                    print(f"Failed to create intent: {result['error']}")
                else:
                    print(f"Created intent: {result['intent'].metadata.name}")
        
        return results
    
    def monitor_intent_events(self, namespace: str, name: str):
        """Monitor real-time intent processing events."""
        
        try:
            event_stream = self.client.intents.watch_events(namespace, name)
            
            print(f"Monitoring events for intent {namespace}/{name}")
            
            for event in event_stream:
                timestamp = event.timestamp.isoformat()
                
                if event.event == "processing":
                    stage = event.data.get("stage", "unknown")
                    progress = event.data.get("progress", 0)
                    print(f"[{timestamp}] Processing: {stage} ({progress}%)")
                    
                elif event.event == "context_retrieved":
                    docs = event.data.get("documents", 0)
                    score = event.data.get("relevance_score", 0.0)
                    print(f"[{timestamp}] Context retrieved: {docs} documents (score: {score:.2f})")
                    
                elif event.event == "deployment_started":
                    resources = event.data.get("resources", 0)
                    estimated_time = event.data.get("estimated_time", "unknown")
                    print(f"[{timestamp}] Deployment started: {resources} resources (ETA: {estimated_time})")
                    
                elif event.event == "deployment_complete":
                    success_count = event.data.get("successful_resources", 0)
                    total_count = event.data.get("total_resources", 0)
                    print(f"[{timestamp}] Deployment complete: {success_count}/{total_count} resources")
                    break
                    
                elif event.event == "error":
                    error_msg = event.data.get("message", "Unknown error")
                    print(f"[{timestamp}] ERROR: {error_msg}")
                    break
                    
        except Exception as e:
            print(f"Error monitoring events: {e}")
    
    def _build_label_selector(self, labels: Optional[Dict[str, str]]) -> Optional[str]:
        """Build Kubernetes label selector from dictionary."""
        if not labels:
            return None
            
        selectors = []
        for key, value in labels.items():
            selectors.append(f"{key}={value}")
            
        return ",".join(selectors)
```

### LLM Processing

```python
from nephoran_sdk.models import LLMProcessingRequest, LLMParameters

def process_telecommunications_intents(client):
    """Process various telecommunications intents with LLM."""
    
    # 5G Core Network Function Intent
    core_nf_request = LLMProcessingRequest(
        intent=(
            "Deploy a resilient 5G core network with AMF, SMF, and UPF functions "
            "supporting 1 million subscribers with carrier-grade availability"
        ),
        context=[
            "5G core network follows service-based architecture (SBA)",
            "AMF handles access and mobility management for UEs", 
            "SMF manages PDU sessions and controls UPF",
            "UPF provides user plane packet processing",
            "Carrier-grade availability requires 99.999% uptime"
        ],
        parameters=LLMParameters(
            model="gpt-4o-mini",
            temperature=0.2,  # Lower temperature for technical precision
            max_tokens=3000,
            top_p=0.9
        )
    )
    
    # Process and analyze response
    core_response = client.llm.process(core_nf_request)
    
    print("5G Core Network Analysis:")
    print(f"Confidence: {core_response.confidence:.2f}")
    print(f"Network Functions: {', '.join(core_response.networkFunctions)}")
    print(f"Deployment Strategy: {core_response.deploymentStrategy}")
    print(f"O-RAN Compliance: {core_response.oranCompliance}")
    
    # O-RAN RIC Intent
    oran_request = LLMProcessingRequest(
        intent=(
            "Configure Near-RT RIC with intelligent xApps for autonomous network "
            "optimization including traffic steering, load balancing, and QoS management"
        ),
        context=[
            "Near-RT RIC provides real-time RAN intelligence and control",
            "xApps are microservices running on RIC platform",
            "E2 interface connects RIC to RAN nodes",
            "A1 interface provides policy-based RIC control",
            "Traffic steering optimizes user experience and network efficiency"
        ],
        parameters=LLMParameters(
            model="gpt-4o-mini", 
            temperature=0.1,  # Very low temperature for O-RAN compliance
            max_tokens=2500
        )
    )
    
    oran_response = client.llm.process(oran_request)
    
    print("\nO-RAN RIC Analysis:")
    print(f"Identified xApps: {oran_response.processedIntent.get('xApps', [])}")
    print(f"E2 Interfaces: {oran_response.processedIntent.get('e2Interfaces', [])}")
    print(f"Optimization Targets: {oran_response.processedIntent.get('optimizationTargets', [])}")
    
    return core_response, oran_response

def stream_llm_processing(client):
    """Demonstrate streaming LLM processing with real-time updates."""
    
    request = LLMProcessingRequest(
        intent=(
            "Design a comprehensive network slicing architecture for a smart city "
            "deployment supporting autonomous vehicles, IoT sensors, emergency services, "
            "and mobile broadband with distinct SLA requirements"
        ),
        streaming=True,
        parameters=LLMParameters(
            model="gpt-4o-mini",
            temperature=0.3,
            max_tokens=4000
        )
    )
    
    print("Streaming LLM Response:")
    print("=" * 50)
    
    try:
        response_stream = client.llm.process_stream(request)
        
        full_response = ""
        for chunk in response_stream:
            if chunk.error:
                print(f"Stream error: {chunk.error}")
                break
                
            print(chunk.content, end='', flush=True)
            full_response += chunk.content
            
        print("\n" + "=" * 50)
        print(f"Total tokens received: {len(full_response.split())}")
        
    except Exception as e:
        print(f"Streaming failed: {e}")
```

### RAG Knowledge Retrieval

```python
from nephoran_sdk.models import RAGQuery, RAGContext

class KnowledgeRetriever:
    """Advanced RAG knowledge retrieval with caching and optimization."""
    
    def __init__(self, client):
        self.client = client
        self._cache = {}  # Simple in-memory cache
        
    def query_telecom_knowledge(
        self,
        query: str,
        domain: str,
        max_results: int = 5,
        threshold: float = 0.8,
        use_cache: bool = True
    ):
        """Query telecommunications knowledge base with caching."""
        
        cache_key = f"{query}:{domain}:{max_results}:{threshold}"
        
        if use_cache and cache_key in self._cache:
            print("Using cached result")
            return self._cache[cache_key]
        
        rag_query = RAGQuery(
            query=query,
            context=RAGContext(
                domain=domain,
                technology=self._infer_technology(query)
            ),
            maxResults=max_results,
            threshold=threshold,
            includeMetadata=True
        )
        
        response = self.client.rag.query(rag_query)
        
        if use_cache:
            self._cache[cache_key] = response
            
        return response
    
    def query_oran_specifications(self):
        """Specialized queries for O-RAN specifications and standards."""
        
        queries = [
            ("Near-RT RIC architecture and xApp development framework", "o-ran"),
            ("E2 interface protocol specifications and service models", "o-ran"), 
            ("A1 policy management and RIC control procedures", "o-ran"),
            ("O-RAN Security architecture and threat model", "o-ran"),
            ("Open Fronthaul interface and functional split options", "o-ran")
        ]
        
        results = {}
        
        for query, domain in queries:
            print(f"\nQuerying: {query}")
            response = self.query_telecom_knowledge(query, domain)
            
            print(f"Found {len(response.results)} results in {response.metadata.processingTimeMs}ms")
            
            for i, result in enumerate(response.results, 1):
                print(f"\n--- Result {i} (Score: {result.score:.3f}) ---")
                print(f"Source: {result.source}")
                print(f"Preview: {result.text[:200]}...")
                
                if result.metadata:
                    print(f"Document: {result.metadata.documentType}, Section: {result.metadata.section}")
            
            results[query] = response
            
        return results
    
    def query_5g_core_architecture(self):
        """Comprehensive 5G Core network architecture knowledge retrieval."""
        
        # Multi-faceted query for comprehensive understanding
        queries = [
            "5G core network service-based architecture components and interfaces",
            "AMF access and mobility management procedures and N1/N2 interfaces",
            "SMF session management and PDU session establishment procedures", 
            "UPF user plane function and data packet processing architecture",
            "Network slicing implementation in 5G core with NSSF and slice selection",
            "5G QoS framework with QoS flows and guaranteed bit rates"
        ]
        
        comprehensive_results = []
        
        for query in queries:
            response = self.query_telecom_knowledge(
                query=query,
                domain="5g-core", 
                max_results=3,
                threshold=0.85
            )
            comprehensive_results.extend(response.results)
        
        # Deduplicate and rank results
        unique_results = {}
        for result in comprehensive_results:
            if result.source not in unique_results or result.score > unique_results[result.source].score:
                unique_results[result.source] = result
        
        # Sort by relevance score
        sorted_results = sorted(unique_results.values(), key=lambda x: x.score, reverse=True)
        
        print(f"\n5G Core Architecture Knowledge Summary:")
        print(f"Total unique sources: {len(sorted_results)}")
        
        for i, result in enumerate(sorted_results[:10], 1):  # Top 10 results
            print(f"\n{i}. {result.source} (Score: {result.score:.3f})")
            print(f"   {result.text[:150]}...")
        
        return sorted_results
    
    def query_with_context_enhancement(self, primary_query: str, domain: str):
        """Enhanced querying with automatic context building."""
        
        # First, get initial results
        initial_response = self.query_telecom_knowledge(
            query=primary_query,
            domain=domain,
            max_results=3
        )
        
        # Extract key terms and build enhanced context
        context_terms = self._extract_key_terms(initial_response)
        
        # Build enhanced query with context
        enhanced_query = f"{primary_query} {' '.join(context_terms)}"
        
        # Query again with enhanced context
        enhanced_response = self.query_telecom_knowledge(
            query=enhanced_query,
            domain=domain,
            max_results=7,
            use_cache=False  # Don't cache enhanced queries
        )
        
        print(f"Enhanced query found {len(enhanced_response.results)} results")
        return enhanced_response
    
    def _infer_technology(self, query: str) -> str:
        """Infer specific technology from query text."""
        
        technology_keywords = {
            "near-rt-ric": ["near-rt", "ric", "xapp", "e2"],
            "5g-core": ["amf", "smf", "upf", "nssf", "sba"],
            "network-slicing": ["slice", "nssai", "sst", "sd"],
            "mec": ["edge", "mec", "multi-access"],
            "ran": ["ran", "gnb", "enb", "cell"]
        }
        
        query_lower = query.lower()
        
        for tech, keywords in technology_keywords.items():
            if any(keyword in query_lower for keyword in keywords):
                return tech
                
        return "general"
    
    def _extract_key_terms(self, response) -> List[str]:
        """Extract key technical terms from RAG response for context enhancement."""
        
        # This is a simplified implementation
        # In production, you might use NLP libraries for better term extraction
        
        key_terms = set()
        
        for result in response.results:
            text = result.text.lower()
            
            # Extract telecommunications acronyms (3-5 uppercase letters)
            import re
            acronyms = re.findall(r'\b[A-Z]{3,5}\b', result.text)
            key_terms.update(acronyms)
            
            # Extract technical terms
            technical_terms = [
                "interface", "protocol", "function", "service",
                "architecture", "deployment", "configuration"
            ]
            
            for term in technical_terms:
                if term in text:
                    key_terms.add(term)
        
        return list(key_terms)[:5]  # Limit to 5 key terms
```

## Error Handling and Resilience

```python
from nephoran_sdk.exceptions import (
    NephoranAPIError, AuthenticationError, RateLimitError,
    ValidationError, ResourceNotFoundError, CircuitBreakerError
)
import time
import random
from typing import Callable, Any, Type
import logging

logger = logging.getLogger(__name__)

class ResilientOperations:
    """Resilient operations with comprehensive error handling and retry logic."""
    
    def __init__(self, client):
        self.client = client
        
    def retry_with_exponential_backoff(
        self,
        operation: Callable,
        max_attempts: int = 3,
        base_delay: float = 1.0,
        max_delay: float = 60.0,
        exponential_base: float = 2.0,
        jitter: bool = True
    ) -> Any:
        """Execute operation with exponential backoff retry logic."""
        
        for attempt in range(max_attempts):
            try:
                return operation()
                
            except RateLimitError as e:
                if attempt == max_attempts - 1:
                    raise
                    
                # Use rate limit header if available
                retry_after = getattr(e, 'retry_after', None)
                if retry_after:
                    delay = min(float(retry_after), max_delay)
                else:
                    delay = min(base_delay * (exponential_base ** attempt), max_delay)
                
                if jitter:
                    delay *= (0.5 + random.random() * 0.5)  # Add ±50% jitter
                
                logger.warning(f"Rate limited, retrying in {delay:.2f}s (attempt {attempt + 1}/{max_attempts})")
                time.sleep(delay)
                
            except (ConnectionError, TimeoutError) as e:
                if attempt == max_attempts - 1:
                    raise
                    
                delay = min(base_delay * (exponential_base ** attempt), max_delay)
                if jitter:
                    delay *= (0.5 + random.random() * 0.5)
                
                logger.warning(f"Connection error, retrying in {delay:.2f}s: {e}")
                time.sleep(delay)
                
            except CircuitBreakerError:
                logger.error("Circuit breaker is open, operation not allowed")
                raise
                
        raise RuntimeError(f"Operation failed after {max_attempts} attempts")
    
    def handle_api_errors(self, operation: Callable) -> Any:
        """Comprehensive API error handling with specific error type handling."""
        
        try:
            return operation()
            
        except ValidationError as e:
            logger.error(f"Validation failed: {e.message}")
            if hasattr(e, 'details') and e.details:
                for field, error in e.details.items():
                    logger.error(f"  {field}: {error}")
            raise
            
        except AuthenticationError as e:
            logger.error(f"Authentication failed: {e.message}")
            # Attempt to refresh token if auto-refresh is enabled
            if hasattr(self.client, '_refresh_token'):
                try:
                    self.client._refresh_token()
                    logger.info("Token refreshed, retrying operation")
                    return operation()
                except Exception:
                    logger.error("Token refresh failed")
            raise
            
        except ResourceNotFoundError as e:
            logger.error(f"Resource not found: {e.message}")
            raise
            
        except RateLimitError as e:
            logger.warning(f"Rate limit exceeded: {e.message}")
            if hasattr(e, 'retry_after'):
                logger.info(f"Retry after: {e.retry_after} seconds")
            raise
            
        except NephoranAPIError as e:
            logger.error(f"API error [{e.status_code}]: {e.message}")
            if hasattr(e, 'request_id'):
                logger.error(f"Request ID: {e.request_id}")
            raise
            
        except Exception as e:
            logger.error(f"Unexpected error: {e}", exc_info=True)
            raise

def create_resilient_intent(client, intent_data):
    """Example of creating an intent with full error handling and retry logic."""
    
    resilient_ops = ResilientOperations(client)
    
    def create_operation():
        return client.intents.create(intent_data)
    
    def create_with_error_handling():
        return resilient_ops.handle_api_errors(create_operation)
    
    try:
        result = resilient_ops.retry_with_exponential_backoff(
            create_with_error_handling,
            max_attempts=3,
            base_delay=1.0
        )
        
        logger.info(f"Intent created successfully: {result.metadata.name}")
        return result
        
    except ValidationError as e:
        logger.error("Intent validation failed - check intent specification")
        # Handle validation errors specifically
        return None
        
    except RateLimitError:
        logger.error("Rate limit exceeded - reduce request frequency") 
        return None
        
    except Exception as e:
        logger.error(f"Failed to create intent after all retries: {e}")
        return None
```

## Testing and Development

```python
import pytest
from unittest.mock import Mock, patch, MagicMock
from nephoran_sdk import NephoranClient
from nephoran_sdk.models import NetworkIntent, NetworkIntentSpec, ObjectMeta

class TestNephoranSDK:
    """Comprehensive test suite for Nephoran SDK."""
    
    @pytest.fixture
    def mock_client(self):
        """Create mock client for testing."""
        with patch('nephoran_sdk.client.NephoranClient') as mock:
            yield mock.return_value
    
    @pytest.fixture
    def sample_intent(self):
        """Sample network intent for testing."""
        return NetworkIntent(
            apiVersion="nephoran.io/v1alpha1",
            kind="NetworkIntent",
            metadata=ObjectMeta(
                name="test-intent",
                namespace="test",
                labels={"env": "test"}
            ),
            spec=NetworkIntentSpec(
                intent="Deploy test AMF instance",
                priority="low"
            )
        )
    
    def test_client_initialization(self):
        """Test client initialization with various configurations."""
        
        # Test basic initialization
        client = NephoranClient(
            base_url="https://api.test.com/v1",
            client_id="test-client",
            client_secret="test-secret"
        )
        
        assert client.base_url == "https://api.test.com/v1"
        assert client.auth.client_id == "test-client"
    
    def test_intent_creation_success(self, mock_client, sample_intent):
        """Test successful intent creation."""
        
        # Mock successful response
        mock_response = Mock()
        mock_response.metadata.name = "test-intent"
        mock_response.metadata.uid = "12345"
        mock_response.status.phase = "Pending"
        
        mock_client.intents.create.return_value = mock_response
        
        # Test intent creation
        result = mock_client.intents.create(sample_intent)
        
        assert result.metadata.name == "test-intent"
        assert result.metadata.uid == "12345"
        mock_client.intents.create.assert_called_once_with(sample_intent)
    
    def test_intent_creation_validation_error(self, mock_client, sample_intent):
        """Test intent creation with validation error."""
        
        from nephoran_sdk.exceptions import ValidationError
        
        # Mock validation error
        mock_client.intents.create.side_effect = ValidationError(
            "Intent validation failed",
            details={"field": "spec.intent", "reason": "too short"}
        )
        
        # Test that validation error is raised
        with pytest.raises(ValidationError) as exc_info:
            mock_client.intents.create(sample_intent)
        
        assert "validation failed" in str(exc_info.value).lower()
    
    def test_rag_query(self, mock_client):
        """Test RAG knowledge query functionality."""
        
        from nephoran_sdk.models import RAGQuery, RAGResponse, RAGResult
        
        # Mock RAG response
        mock_result = RAGResult(
            text="O-RAN Near-RT RIC provides real-time control...",
            score=0.95,
            source="oran-arch-spec-v1.2",
            metadata={"documentType": "specification"}
        )
        
        mock_response = RAGResponse(
            results=[mock_result],
            metadata={"totalResults": 1, "processingTimeMs": 150}
        )
        
        mock_client.rag.query.return_value = mock_response
        
        # Test query
        query = RAGQuery(
            query="O-RAN Near-RT RIC architecture",
            maxResults=5
        )
        
        result = mock_client.rag.query(query)
        
        assert len(result.results) == 1
        assert result.results[0].score == 0.95
        assert "oran" in result.results[0].source.lower()
    
    @pytest.mark.asyncio
    async def test_async_operations(self):
        """Test asynchronous client operations."""
        
        from nephoran_sdk.async_client import AsyncNephoranClient
        
        with patch('nephoran_sdk.async_client.AsyncNephoranClient') as mock_async_client:
            # Mock async intent creation
            mock_intent = Mock()
            mock_intent.metadata.name = "async-intent"
            
            mock_async_client.return_value.__aenter__.return_value.intents.create = Mock(
                return_value=mock_intent
            )
            
            async with AsyncNephoranClient(
                base_url="https://api.test.com/v1",
                client_id="test",
                client_secret="secret"
            ) as client:
                
                result = await client.intents.create(sample_intent)
                assert result.metadata.name == "async-intent"

# Integration tests (require actual API access)
class TestIntegration:
    """Integration tests with real API (requires credentials)."""
    
    @pytest.fixture
    def integration_client(self):
        """Create client for integration testing."""
        import os
        
        # Skip if no credentials available
        if not all([
            os.getenv("NEPHORAN_CLIENT_ID"),
            os.getenv("NEPHORAN_CLIENT_SECRET")
        ]):
            pytest.skip("Integration test credentials not available")
        
        return NephoranClient(
            base_url=os.getenv("NEPHORAN_TEST_URL", "https://staging-api.nephoran.com/v1"),
            client_id=os.getenv("NEPHORAN_CLIENT_ID"),
            client_secret=os.getenv("NEPHORAN_CLIENT_SECRET"),
            auth_url=os.getenv("NEPHORAN_AUTH_URL", "https://auth.nephoran.com/oauth/token"),
            scopes=["intent.read", "intent.write", "rag.query"]
        )
    
    def test_health_check(self, integration_client):
        """Test API health check."""
        
        health = integration_client.health.check()
        assert health.status in ["healthy", "degraded"]
        assert "llm-processor" in health.components
        assert "rag-service" in health.components
    
    def test_rag_query_integration(self, integration_client):
        """Test actual RAG query against knowledge base."""
        
        from nephoran_sdk.models import RAGQuery
        
        query = RAGQuery(
            query="O-RAN architecture components",
            maxResults=3,
            threshold=0.7
        )
        
        response = integration_client.rag.query(query)
        
        assert len(response.results) > 0
        assert all(result.score >= 0.7 for result in response.results)
        assert response.metadata.processingTimeMs > 0

# Performance tests
def test_performance_metrics():
    """Test SDK performance characteristics."""
    
    import time
    
    with patch('nephoran_sdk.client.requests.Session') as mock_session:
        # Mock fast response
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"status": "healthy"}
        mock_response.elapsed.total_seconds.return_value = 0.1
        
        mock_session.return_value.request.return_value = mock_response
        
        client = NephoranClient(
            base_url="https://api.test.com/v1",
            client_id="test",
            client_secret="secret"
        )
        
        # Measure response time
        start_time = time.time()
        health = client.health.check()
        elapsed = time.time() - start_time
        
        # Assert performance characteristics
        assert elapsed < 1.0  # Should complete within 1 second
        assert health.status == "healthy"

if __name__ == "__main__":
    # Run tests
    pytest.main([__file__, "-v", "--tb=short"])
```

This comprehensive Python SDK guide provides production-ready code examples, advanced error handling, asynchronous support, comprehensive testing patterns, and enterprise-grade features for integrating with the Nephoran Intent Operator API.