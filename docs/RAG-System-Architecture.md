# Enhanced RAG System Architecture for Nephoran Intent Operator

## Overview

The Nephoran Intent Operator features a comprehensive Retrieval-Augmented Generation (RAG) system specifically optimized for telecommunications domain knowledge. This production-ready system enables natural language intent processing with domain-specific context retrieval, resulting in more accurate and relevant network function deployments.

## System Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    Enhanced RAG System                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────┐ │
│  │   Weaviate      │    │  Enhanced RAG   │    │ Document    │ │
│  │ Vector Database │◄───┤   Pipeline      │◄───┤ Processor   │ │
│  │                 │    │                 │    │             │ │
│  │ • Production    │    │ • Async Proc.   │    │ • Telecom   │ │
│  │   Scaling       │    │ • Caching       │    │   Keywords  │ │
│  │ • Monitoring    │    │ • Metrics       │    │ • Multi-fmt │ │
│  │ • Clustering    │    │ • Error Handling│    │ • Batch Proc│ │
│  └─────────────────┘    └─────────────────┘    └─────────────┘ │
│           │                       │                      │     │
│           ▼                       ▼                      │     │
│  ┌─────────────────┐    ┌─────────────────┐              │     │
│  │ Flask RAG API   │    │ Knowledge Base  │              │     │
│  │                 │    │ Manager         │◄─────────────┘     │
│  │ • REST Endpoints│    │                 │                    │
│  │ • Health Checks │    │ • Schema Mgmt   │                    │
│  │ • File Upload   │    │ • Batch Ingest  │                    │
│  │ • Statistics    │    │ • Validation    │                    │
│  └─────────────────┘    └─────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘
```

### Integration with Existing System

```
NetworkIntent CRD → NetworkIntent Controller → LLM Processor Service
                                                        │
                                                        ▼
                                              Enhanced RAG API
                                                        │
                                                        ▼
                                           ┌─────────────────────┐
                                           │    Weaviate DB      │
                                           │ ┌─────────────────┐ │
                                           │ │ TelecomKnowledge│ │
                                           │ │     Class       │ │
                                           │ │                 │ │
                                           │ │ • 3GPP Specs    │ │
                                           │ │ • O-RAN Specs   │ │
                                           │ │ • Use Cases     │ │
                                           │ │ • Configs       │ │
                                           │ └─────────────────┘ │
                                           └─────────────────────┘
```

## Component Details

### 1. Weaviate Vector Database

**Production-Ready Features:**
- **Horizontal Scaling**: 2-5 replica auto-scaling based on CPU/memory usage
- **Persistent Storage**: 100GB persistent volume for knowledge base
- **Authentication**: API key-based access control
- **Monitoring**: Prometheus metrics integration with Grafana dashboards
- **High Availability**: Pod anti-affinity rules for replica distribution

**Configuration:**
```yaml
# Key configuration from deployments/weaviate/values.yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 70

persistence:
  enabled: true
  size: 100Gi

env:
  DEFAULT_VECTORIZER_MODULE: "text2vec-openai"
  ENABLE_MODULES: "text2vec-openai,generative-openai,qna-openai"
```

### 2. Enhanced RAG Pipeline

**Advanced Features:**
- **Asynchronous Processing**: Concurrent intent processing with asyncio
- **Intelligent Caching**: LRU cache with TTL for repeated intents
- **Metrics Collection**: Comprehensive processing metrics and performance tracking
- **Error Recovery**: Robust error handling with fallback mechanisms
- **Response Validation**: Schema validation for structured outputs

**Key Capabilities:**
```python
class EnhancedTelecomRAGPipeline:
    async def process_intent_async(self, intent: str) -> ProcessedIntent:
        # Cache check
        # Async LLM processing
        # Metrics calculation
        # Response validation
        # Cache storage
```

### 3. Document Processor

**Telecom-Specific Processing:**
- **Multi-Format Support**: PDF, Markdown, JSON, YAML, text documents
- **Keyword Extraction**: Automated extraction of telecom domain keywords
- **Technical References**: Pattern matching for 3GPP, O-RAN, RFC specifications
- **Chunk Optimization**: Intelligent text chunking with overlap for context preservation
- **Batch Processing**: Concurrent document processing with configurable limits

**Supported Document Types:**
- 3GPP Technical Specifications (TS documents)
- O-RAN Working Group specifications
- YAML/JSON configuration files
- Markdown documentation
- PDF specifications

### 4. Knowledge Base Management

**Schema Design:**
```json
{
  "class": "TelecomKnowledge",
  "properties": [
    {"name": "content", "dataType": ["text"]},
    {"name": "source", "dataType": ["text"]},
    {"name": "category", "dataType": ["text"]},
    {"name": "keywords", "dataType": ["text[]"]},
    {"name": "confidence", "dataType": ["number"]}
  ]
}
```

**Categories:**
- `5g_core`: AMF, SMF, UPF, NSSF, etc.
- `ran`: gNB, CU, DU, RIC components
- `network_slice`: S-NSSAI, NSI, NSSI concepts
- `interfaces`: A1, O1, O2, E2 specifications
- `mgmt`: FCAPS, OAM, policy management
- `protocols`: HTTP/2, gRPC, NETCONF

## API Endpoints

### Enhanced RAG API

**Core Endpoints:**
- `GET /healthz` - Basic health check
- `GET /readyz` - Readiness with dependency checks
- `POST /process_intent` - Process natural language intents
- `GET /stats` - System statistics and metrics

**Knowledge Management:**
- `POST /knowledge/upload` - Upload and process documents
- `POST /knowledge/populate` - Populate from directory
- `GET /knowledge/stats` - Knowledge base statistics

**Example Usage:**
```bash
# Process an intent
curl -X POST http://rag-api:5001/process_intent \
  -H "Content-Type: application/json" \
  -d '{"intent": "Deploy AMF with 3 replicas for network slice eMBB"}'

# Upload documents
curl -X POST http://rag-api:5001/knowledge/upload \
  -F "files=@3gpp_ts_23_501.pdf" \
  -F "files=@oran_spec.md"

# Get system stats
curl http://rag-api:5001/stats
```

## Deployment Guide

### Prerequisites

1. **Kubernetes Cluster** with minimum 8GB RAM and 4 CPU cores
2. **OpenAI API Key** for LLM and embedding services
3. **kubectl** and **kustomize** tools installed
4. **Storage Class** for persistent volumes

### Quick Deployment

```bash
# Set required environment variable
export OPENAI_API_KEY="sk-your-api-key-here"

# Deploy complete RAG system
make deploy-rag

# Verify deployment
make verify-rag

# Populate knowledge base
make populate-kb-enhanced

# Check status
make rag-status
```

### Manual Deployment Steps

1. **Deploy Weaviate:**
```bash
kubectl apply -f deployments/weaviate/
```

2. **Create Secrets:**
```bash
kubectl create secret generic openai-api-key --from-literal=api-key="$OPENAI_API_KEY"
kubectl create secret generic weaviate-api-key --from-literal=api-key="nephoran-rag-key"
```

3. **Deploy RAG API:**
```bash
kubectl apply -f deployments/kustomize/base/rag-api/
```

4. **Populate Knowledge Base:**
```bash
python3 scripts/populate_vector_store_enhanced.py knowledge_base/
```

## Configuration Options

### Environment Variables

**Weaviate Configuration:**
- `WEAVIATE_URL`: Vector database URL (default: `http://weaviate:8080`)
- `WEAVIATE_API_KEY`: Authentication key

**LLM Configuration:**
- `OPENAI_API_KEY`: Required OpenAI API key
- `OPENAI_MODEL`: Model to use (default: `gpt-4o-mini`)

**Processing Configuration:**
- `CACHE_MAX_SIZE`: Max cached intents (default: `1000`)
- `CACHE_TTL_SECONDS`: Cache TTL (default: `3600`)
- `CHUNK_SIZE`: Text chunk size (default: `1000`)
- `CHUNK_OVERLAP`: Chunk overlap (default: `200`)

### Scaling Configuration

**Weaviate Scaling:**
```yaml
autoscaling:
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
```

**RAG API Scaling:**
```yaml
replicas: 2  # Adjust based on load
resources:
  limits:
    memory: "2Gi"
    cpu: "1000m"
```

## Monitoring and Observability

### Metrics Collection

**Weaviate Metrics:**
- Object count and storage usage
- Query performance and latency
- Memory and CPU utilization
- Error rates and availability

**RAG Pipeline Metrics:**
- Intent processing time
- Cache hit rates
- Token usage and costs
- Success/failure rates

### Grafana Dashboard

The system includes a pre-configured Grafana dashboard (`deployments/weaviate/monitoring.yaml`) with:
- Weaviate instance status
- Vector database size
- Query performance metrics
- Memory usage graphs
- Vector operations per second

### Alerts

**Critical Alerts:**
- `WeaviateDown`: Instance unavailable
- `WeaviateHighErrorRate`: >10% query error rate
- `WeaviateSlowQueries`: >5s 95th percentile latency
- `WeaviateHighMemoryUsage`: >6GB memory usage

## Security Features

### Network Policies

**Ingress Rules:**
- RAG API pods can access Weaviate
- LLM processor can access RAG API
- Health check traffic allowed

**Egress Rules:**
- HTTPS for OpenAI API calls
- DNS resolution allowed
- Inter-pod communication for clustering

### Authentication

**API Key Authentication:**
- Weaviate secured with API key
- OpenAI API key stored in Kubernetes secrets
- Internal service communication secured

### RBAC

**Service Account Permissions:**
- Minimal required permissions for each component
- Separate service accounts for different components
- ClusterRole for necessary cluster-wide access

## Performance Characteristics

### Benchmarks

**Intent Processing:**
- Single intent: ~2-5 seconds (including retrieval)
- Concurrent processing: 10+ intents/second
- Cache hit processing: <100ms

**Knowledge Base:**
- Index capacity: 1M+ document chunks
- Query latency: <500ms for semantic search
- Ingestion rate: 1000+ documents/hour

### Resource Requirements

**Minimum Configuration:**
- Weaviate: 2GB RAM, 1 CPU core
- RAG API: 1GB RAM, 0.5 CPU core
- Storage: 100GB persistent volume

**Recommended Production:**
- Weaviate: 8GB RAM, 2 CPU cores, 3 replicas
- RAG API: 2GB RAM, 1 CPU core, 2 replicas
- Storage: 500GB with backup strategy

## Troubleshooting Guide

### Common Issues

**1. Weaviate Connection Failed:**
```bash
# Check Weaviate status
kubectl get pods -l app=weaviate
kubectl logs deployment/weaviate

# Verify service connectivity
kubectl exec -it deployment/rag-api -- curl http://weaviate:8080/v1/.well-known/ready
```

**2. OpenAI API Issues:**
```bash
# Check secret configuration
kubectl get secret openai-api-key -o yaml

# Verify API key in pod
kubectl exec -it deployment/rag-api -- env | grep OPENAI
```

**3. Knowledge Base Empty:**
```bash
# Check object count
kubectl exec -it deployment/rag-api -- curl http://localhost:5001/stats

# Repopulate if needed
make populate-kb-enhanced
```

### Debug Commands

```bash
# Check all RAG system components
make rag-status

# View logs from all components
make rag-logs

# Test RAG API endpoints
kubectl port-forward svc/rag-api 5001:5001
curl http://localhost:5001/healthz
curl http://localhost:5001/stats

# Test Weaviate directly
kubectl port-forward svc/weaviate 8080:8080
curl http://localhost:8080/v1/.well-known/ready
```

## Future Enhancements

### Planned Features

1. **Multi-Modal Support**: Image and diagram processing for technical specifications
2. **Federated Search**: Integration with external telecom knowledge bases
3. **Real-time Updates**: Continuous knowledge base synchronization
4. **Advanced Analytics**: Intent pattern analysis and optimization suggestions
5. **Multi-Language**: Support for non-English telecom specifications

### Integration Roadmap

1. **Nephio Porch Integration**: Automated KRM package generation from processed intents
2. **O-RAN Interface Implementation**: Direct integration with O-RAN components
3. **Policy Engine**: Advanced A1 policy generation and validation
4. **Service Mesh**: Istio integration for advanced traffic management

## Conclusion

The Enhanced RAG System provides a production-ready, scalable foundation for intelligent telecom intent processing. With comprehensive monitoring, security features, and telecom-specific optimizations, it enables the Nephoran Intent Operator to deliver accurate, context-aware network function deployments from natural language descriptions.

The system is designed for enterprise deployment with high availability, security, and performance characteristics suitable for production telecommunications environments.