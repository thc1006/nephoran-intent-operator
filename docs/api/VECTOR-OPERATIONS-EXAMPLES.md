# Vector Operations Examples and Best Practices
## Comprehensive Guide for Enhanced RAG System in Nephoran Intent Operator

### Overview

This document provides practical examples and best practices for vector operations in the enhanced Weaviate deployment for the Nephoran Intent Operator. It covers semantic search patterns, hybrid queries, intent matching, circuit breaker integration, and performance optimization techniques specifically tailored for telecommunications domain operations.

### 🚀 **Latest Enhancements (July 2025)**

**Circuit Breaker Integration:**
- Automatic failure detection and recovery for vector operations
- Rate limiting with exponential backoff for OpenAI API calls
- Local embedding model fallback (sentence-transformers/all-mpnet-base-v2)
- Multi-provider embedding strategy with seamless failover

**Enhanced Chunking Service:**
- Section-aware 512-token chunking optimized for telecom specifications
- Technical term protection preventing splitting of compound terms
- Hierarchy-aware processing maintaining document structure
- 50-token overlap for optimal context preservation

**Performance Optimizations:**
- HNSW parameter tuning (ef=64, efConstruction=128, maxConnections=16)
- Resource optimization (50% memory reduction, maintained performance)
- Query latency improvements (P95 <200ms for hybrid search operations)
- 85%+ cache hit rate for frequently accessed content

### Table of Contents

1. [Enhanced Chunking Service API](#enhanced-chunking-service-api)
2. [Circuit Breaker Integration](#circuit-breaker-integration)
3. [Basic Vector Operations](#basic-vector-operations)
4. [Semantic Search Patterns](#semantic-search-patterns)
5. [Hybrid Search Strategies](#hybrid-search-strategies)
6. [Intent Pattern Matching](#intent-pattern-matching)
7. [Cross-Reference Queries](#cross-reference-queries)
8. [Batch Operations](#batch-operations)
9. [Performance Optimization](#performance-optimization)
10. [Error Handling](#error-handling)
11. [Best Practices](#best-practices)
12. [Advanced Techniques](#advanced-techniques)

## Enhanced Chunking Service API

### Overview

The Enhanced Chunking Service provides intelligent document segmentation optimized for telecommunications specifications. It implements section-aware chunking with technical term protection and hierarchy preservation.

### Core API Endpoints

#### POST /v1/chunk/document

Process a document into optimized chunks for vector storage.

**Request:**
```http
POST /v1/chunk/document HTTP/1.1
Host: rag-api:5000
Content-Type: application/json
Authorization: Bearer <RAG_API_KEY>

{
  "document": {
    "content": "3GPP TS 23.501 defines the 5G System Architecture. The Access and Mobility Management Function (AMF) provides registration management...",
    "title": "5G System Architecture Overview",
    "source": "3GPP",
    "specification": "TS 23.501",
    "document_type": "technical_specification"
  },
  "chunking_config": {
    "chunk_size": 512,
    "overlap_size": 50,
    "preserve_hierarchy": true,
    "protect_technical_terms": true,
    "section_boundary_detection": true
  }
}
```

**Response:**
```json
{
  "status": "success",
  "document_id": "3gpp-ts-23501-overview",
  "total_chunks": 15,
  "processing_time_ms": 245,
  "chunks": [
    {
      "chunk_id": "3gpp-ts-23501-overview-chunk-001",
      "content": "3GPP TS 23.501 defines the 5G System Architecture. The Access and Mobility Management Function (AMF) provides registration management, connection management, and mobility management services...",
      "metadata": {
        "chunk_index": 1,
        "token_count": 487,
        "technical_terms": ["3GPP", "AMF", "5G"],
        "section_hierarchy": ["1", "1.1", "1.1.1"],
        "section_title": "AMF Functions",
        "confidence_score": 0.92,
        "overlap_with_previous": 50,
        "overlap_with_next": 50
      },
      "embeddings": {
        "primary_provider": "openai",
        "model": "text-embedding-3-large",
        "dimensions": 3072,
        "generation_time_ms": 156
      }
    }
  ],
  "document_metadata": {
    "total_tokens": 7234,
    "sections_detected": 8,
    "technical_terms_preserved": 45,
    "hierarchy_levels": 4
  }
}
```

#### POST /v1/chunk/batch

Process multiple documents in batch for efficient chunking.

**Request:**
```http
POST /v1/chunk/batch HTTP/1.1
Host: rag-api:5000
Content-Type: application/json
Authorization: Bearer <RAG_API_KEY>

{
  "documents": [
    {
      "document_id": "3gpp-ts-23501",
      "content": "3GPP TS 23.501 content...",
      "title": "5G System Architecture",
      "source": "3GPP"
    },
    {
      "document_id": "oran-wg1-use-cases",
      "content": "O-RAN WG1 use cases...",
      "title": "O-RAN Use Cases",
      "source": "O-RAN"
    }
  ],
  "batch_config": {
    "max_concurrent": 5,
    "chunk_size": 512,
    "overlap_size": 50,
    "circuit_breaker_enabled": true
  }
}
```

**Response:**
```json
{
  "status": "success",
  "batch_id": "batch-20240728-103045",
  "total_documents": 2,
  "total_chunks": 28,
  "processing_time_ms": 1234,
  "results": [
    {
      "document_id": "3gpp-ts-23501",
      "status": "success",
      "chunks_created": 15,
      "processing_time_ms": 678
    },
    {
      "document_id": "oran-wg1-use-cases", 
      "status": "success",
      "chunks_created": 13,
      "processing_time_ms": 556
    }
  ],
  "circuit_breaker_status": {
    "state": "closed",
    "failure_count": 0,
    "success_rate": 1.0
  }
}
```

#### GET /v1/chunk/status/{batch_id}

Check the status of a batch chunking operation.

**Request:**
```http
GET /v1/chunk/status/batch-20240728-103045 HTTP/1.1
Host: rag-api:5000
Authorization: Bearer <RAG_API_KEY>
```

**Response:**
```json
{
  "batch_id": "batch-20240728-103045",
  "status": "completed",
  "progress": {
    "total_documents": 2,
    "processed_documents": 2,
    "failed_documents": 0,
    "percentage_complete": 100
  },
  "performance_metrics": {
    "average_processing_time_ms": 617,
    "chunks_per_second": 22.7,
    "embedding_generation_rate": 18.3
  },
  "circuit_breaker_metrics": {
    "total_requests": 28,
    "successful_requests": 28,
    "failed_requests": 0,
    "fallback_used": 0
  }
}
```

### Data Models

#### ChunkingConfig

```typescript
interface ChunkingConfig {
  chunk_size: number;                    // Target chunk size in tokens (default: 512)
  overlap_size: number;                  // Overlap between chunks in tokens (default: 50)
  preserve_hierarchy: boolean;           // Maintain document section hierarchy
  protect_technical_terms: boolean;      // Prevent splitting technical terms
  section_boundary_detection: boolean;   // Detect and respect section boundaries
  min_chunk_size: number;               // Minimum chunk size (default: 100)
  max_chunk_size: number;               // Maximum chunk size (default: 1000)
  technical_term_dictionary?: string[]; // Custom technical terms to protect
}
```

#### ChunkMetadata

```typescript
interface ChunkMetadata {
  chunk_index: number;                   // Sequential chunk number
  token_count: number;                   // Actual token count in chunk
  technical_terms: string[];             // Technical terms found in chunk
  section_hierarchy: string[];           // Document section path (e.g., ["1", "1.1"])
  section_title: string;                 // Title of the containing section
  confidence_score: number;              // Quality/completeness score (0.0-1.0)
  overlap_with_previous: number;         // Tokens overlapping with previous chunk
  overlap_with_next: number;             // Tokens overlapping with next chunk
  boundary_type: "natural" | "forced";  // How chunk boundary was determined
}
```

#### EmbeddingInfo

```typescript
interface EmbeddingInfo {
  primary_provider: "openai" | "local";  // Which embedding provider was used
  model: string;                         // Specific model name
  dimensions: number;                    // Vector dimensions
  generation_time_ms: number;           // Time taken to generate embedding
  fallback_used: boolean;               // Whether fallback model was used
  circuit_breaker_state: string;       // Circuit breaker state during generation
}
```

## Circuit Breaker Integration

### Overview

The Circuit Breaker pattern is integrated throughout the RAG system to provide resilience against failures and automatically manage fallback strategies.

### Circuit Breaker States

- **CLOSED**: Normal operation, requests flow through
- **OPEN**: Failures detected, requests are blocked/redirected to fallback
- **HALF_OPEN**: Testing recovery, limited requests allowed

### API Endpoints

#### GET /v1/circuit-breaker/status

Get the current status of all circuit breakers in the system.

**Request:**
```http
GET /v1/circuit-breaker/status HTTP/1.1
Host: rag-api:5000
Authorization: Bearer <RAG_API_KEY>
```

**Response:**
```json
{
  "timestamp": "2024-07-28T10:30:00Z",
  "circuit_breakers": {
    "openai_embedding": {
      "state": "closed",
      "failure_count": 0,
      "success_count": 1247,
      "failure_threshold": 3,
      "timeout_duration_ms": 60000,
      "last_failure": null,
      "success_rate": 1.0,
      "average_response_time_ms": 156
    },
    "weaviate_vector_search": {
      "state": "closed",
      "failure_count": 1,
      "success_count": 892,
      "failure_threshold": 3,
      "timeout_duration_ms": 60000,
      "last_failure": "2024-07-28T09:15:23Z",
      "success_rate": 0.998,
      "average_response_time_ms": 89
    },
    "document_chunking": {
      "state": "closed",
      "failure_count": 0,
      "success_count": 345,
      "failure_threshold": 3,
      "timeout_duration_ms": 30000,
      "last_failure": null,
      "success_rate": 1.0,
      "average_response_time_ms": 234
    }
  },
  "fallback_status": {
    "local_embedding_model": {
      "available": true,
      "model": "sentence-transformers/all-mpnet-base-v2",
      "dimensions": 768,
      "last_used": "2024-07-28T08:45:12Z",
      "success_rate": 0.97
    }
  }
}
```

#### POST /v1/circuit-breaker/reset

Reset a specific circuit breaker to closed state.

**Request:**
```http
POST /v1/circuit-breaker/reset HTTP/1.1
Host: rag-api:5000
Content-Type: application/json
Authorization: Bearer <RAG_API_KEY>

{
  "circuit_breaker_name": "openai_embedding",
  "force_reset": true
}
```

**Response:**
```json
{
  "status": "success",
  "circuit_breaker_name": "openai_embedding",
  "previous_state": "open",
  "new_state": "closed",
  "reset_timestamp": "2024-07-28T10:30:15Z",
  "reset_by": "manual_intervention"
}
```

### Integration Examples

#### Python Client with Circuit Breaker Awareness

```python
import requests
import time
from typing import Dict, List, Any, Optional

class EnhancedRAGClient:
    """RAG API client with circuit breaker awareness"""
    
    def __init__(self, base_url: str, api_key: str):
        self.base_url = base_url
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {api_key}',
            'Content-Type': 'application/json'
        })
    
    def chunk_document_with_fallback(
        self, 
        document: Dict[str, Any],
        chunking_config: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """
        Chunk document with automatic circuit breaker handling
        """
        
        chunking_config = chunking_config or {
            "chunk_size": 512,
            "overlap_size": 50,
            "preserve_hierarchy": True,
            "protect_technical_terms": True
        }
        
        request_payload = {
            "document": document,
            "chunking_config": chunking_config
        }
        
        try:
            response = self.session.post(
                f"{self.base_url}/v1/chunk/document",
                json=request_payload,
                timeout=30
            )
            
            if response.status_code == 200:
                return response.json()
            
            elif response.status_code == 503:
                # Service unavailable - circuit breaker likely open
                error_data = response.json()
                if error_data.get("circuit_breaker_open"):
                    return self._handle_circuit_breaker_open(document, chunking_config)
            
            else:
                response.raise_for_status()
        
        except requests.exceptions.RequestException as e:
            print(f"Request failed: {e}")
            return self._handle_circuit_breaker_open(document, chunking_config)
    
    def _handle_circuit_breaker_open(
        self, 
        document: Dict[str, Any], 
        chunking_config: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Handle circuit breaker open state with fallback chunking
        """
        
        print("Circuit breaker open - using fallback chunking strategy")
        
        # Simple text-based chunking as fallback
        content = document["content"]
        chunk_size = chunking_config.get("chunk_size", 512)
        overlap_size = chunking_config.get("overlap_size", 50)
        
        # Approximate token count (4 chars per token)
        approx_char_per_token = 4
        chunk_char_size = chunk_size * approx_char_per_token
        overlap_char_size = overlap_size * approx_char_per_token
        
        chunks = []
        chunk_index = 1
        start_pos = 0
        
        while start_pos < len(content):
            end_pos = min(start_pos + chunk_char_size, len(content))
            
            # Try to break at sentence boundary
            if end_pos < len(content):
                sentence_end = content.rfind('.', start_pos, end_pos)
                if sentence_end > start_pos + chunk_char_size // 2:
                    end_pos = sentence_end + 1
            
            chunk_content = content[start_pos:end_pos].strip()
            
            if chunk_content:
                chunks.append({
                    "chunk_id": f"{document.get('title', 'doc')}-fallback-{chunk_index:03d}",
                    "content": chunk_content,
                    "metadata": {
                        "chunk_index": chunk_index,
                        "token_count": len(chunk_content) // approx_char_per_token,
                        "technical_terms": [],
                        "section_hierarchy": [],
                        "section_title": "Unknown",
                        "confidence_score": 0.5,  # Lower confidence for fallback
                        "fallback_method": "simple_text_splitting"
                    },
                    "embeddings": {
                        "primary_provider": "fallback",
                        "model": "none",
                        "dimensions": 0,
                        "generation_time_ms": 0
                    }
                })
                
                chunk_index += 1
            
            # Move start position with overlap
            start_pos = end_pos - overlap_char_size
            if start_pos >= end_pos:
                break
        
        return {
            "status": "success_fallback",
            "document_id": document.get("title", "unknown"),
            "total_chunks": len(chunks),
            "processing_time_ms": 50,  # Fast fallback processing
            "chunks": chunks,
            "circuit_breaker_used": True,
            "fallback_method": "simple_text_splitting"
        }
    
    def get_circuit_breaker_status(self) -> Dict[str, Any]:
        """Get current circuit breaker status"""
        
        try:
            response = self.session.get(f"{self.base_url}/v1/circuit-breaker/status")
            response.raise_for_status()
            return response.json()
        
        except requests.exceptions.RequestException as e:
            print(f"Failed to get circuit breaker status: {e}")
            return {"error": str(e)}
    
    def monitor_circuit_breakers(self, interval_seconds: int = 60):
        """Monitor circuit breaker status continuously"""
        
        print(f"Starting circuit breaker monitoring (interval: {interval_seconds}s)")
        
        while True:
            status = self.get_circuit_breaker_status()
            
            if "error" not in status:
                print(f"\n=== Circuit Breaker Status at {status.get('timestamp')} ===")
                
                for name, cb_info in status.get("circuit_breakers", {}).items():
                    state = cb_info["state"]
                    success_rate = cb_info["success_rate"]
                    failure_count = cb_info["failure_count"]
                    
                    status_icon = "🟢" if state == "closed" else "🟡" if state == "half_open" else "🔴"
                    
                    print(f"{status_icon} {name}: {state.upper()} (success: {success_rate:.3f}, failures: {failure_count})")
                
                # Check for fallback usage
                fallback_status = status.get("fallback_status", {})
                for name, fb_info in fallback_status.items():
                    if fb_info.get("last_used"):
                        print(f"⚡ {name}: Available (last used: {fb_info['last_used']})")
            
            time.sleep(interval_seconds)

# Example Usage
client = EnhancedRAGClient("http://rag-api:5000", "your-api-key")

# Test document chunking with circuit breaker handling
test_document = {
    "content": "The Access and Mobility Management Function (AMF) is a key component of the 5G Core Network...",
    "title": "AMF Overview",
    "source": "3GPP",
    "specification": "TS 23.501"
}

result = client.chunk_document_with_fallback(test_document)
print(f"Chunking result: {result['status']}")
print(f"Chunks created: {result['total_chunks']}")

if result.get("circuit_breaker_used"):
    print("⚠️  Circuit breaker was activated - fallback method used")

# Monitor circuit breakers (runs continuously)
# client.monitor_circuit_breakers(interval_seconds=30)
```

## Basic Vector Operations

### Simple Similarity Search

Find documents similar to a given query using semantic vector search:

```python
import weaviate
from typing import List, Dict, Any

def basic_similarity_search(client: weaviate.Client, query: str, limit: int = 10) -> List[Dict[str, Any]]:
    """
    Basic semantic similarity search using nearText
    
    Args:
        client: Weaviate client instance
        query: Search query text
        limit: Maximum number of results
        
    Returns:
        List of matching documents with metadata
    """
    result = client.query.get("TelecomKnowledge", [
        "content", 
        "title", 
        "source", 
        "specification",
        "networkFunctions",
        "confidence"
    ]).with_near_text({
        "concepts": [query],
        "certainty": 0.7  # Similarity threshold
    }).with_limit(limit).with_additional([
        "certainty", 
        "distance"
    ]).do()
    
    return result["data"]["Get"]["TelecomKnowledge"]

# Example usage
client = weaviate.Client("http://weaviate:8080")
results = basic_similarity_search(client, "AMF registration procedure")

for doc in results:
    print(f"Title: {doc['title']}")
    print(f"Certainty: {doc['_additional']['certainty']:.3f}")
    print(f"Content: {doc['content'][:200]}...")
    print("---")
```

### Vector Search with Filtering

Combine semantic search with metadata filtering:

```python
def filtered_vector_search(
    client: weaviate.Client, 
    query: str, 
    source: str = None,
    network_functions: List[str] = None,
    min_confidence: float = 0.8
) -> List[Dict[str, Any]]:
    """
    Vector search with multiple filters
    
    Args:
        client: Weaviate client instance
        query: Search query
        source: Filter by source (3GPP, O-RAN, etc.)
        network_functions: Filter by network functions
        min_confidence: Minimum confidence threshold
    """
    
    # Build where filter
    where_filter = {"operator": "And", "operands": []}
    
    if source:
        where_filter["operands"].append({
            "path": ["source"],
            "operator": "Equal",
            "valueText": source
        })
    
    if network_functions:
        where_filter["operands"].append({
            "path": ["networkFunctions"],
            "operator": "ContainsAny",
            "valueTextArray": network_functions
        })
    
    if min_confidence:
        where_filter["operands"].append({
            "path": ["confidence"],
            "operator": "GreaterThan",
            "valueNumber": min_confidence
        })
    
    # Execute query
    query_builder = client.query.get("TelecomKnowledge", [
        "content", "title", "source", "specification", 
        "networkFunctions", "confidence", "release"
    ]).with_near_text({
        "concepts": [query],
        "certainty": 0.7
    }).with_limit(20)
    
    if where_filter["operands"]:
        query_builder = query_builder.with_where(where_filter)
    
    result = query_builder.with_additional(["certainty", "distance"]).do()
    return result["data"]["Get"]["TelecomKnowledge"]

# Example usage
results = filtered_vector_search(
    client, 
    "network slice configuration",
    source="3GPP",
    network_functions=["AMF", "NSSF"],
    min_confidence=0.9
)
```

## Semantic Search Patterns

### Intent-Based Semantic Search

Search for telecommunications procedures based on natural language intent:

```python
def intent_based_search(client: weaviate.Client, intent: str) -> Dict[str, Any]:
    """
    Search for procedures and configurations based on user intent
    
    Args:
        client: Weaviate client instance
        intent: Natural language intent description
        
    Returns:
        Structured results with procedures, configurations, and related NFs
    """
    
    # Search for relevant knowledge
    knowledge_results = client.query.get("TelecomKnowledge", [
        "content", "title", "procedures", "networkFunctions", 
        "interfaces", "category", "confidence"
    ]).with_near_text({
        "concepts": [intent],
        "certainty": 0.75
    }).with_limit(10).with_additional(["certainty"]).do()
    
    # Search for matching intent patterns
    pattern_results = client.query.get("IntentPatterns", [
        "pattern", "intentType", "parameters", "confidence", "examples"
    ]).with_near_text({
        "concepts": [intent],
        "certainty": 0.7
    }).with_limit(5).with_additional(["certainty"]).do()
    
    # Search for relevant network functions
    nf_results = client.query.get("NetworkFunctions", [
        "name", "description", "category", "deploymentOptions",
        "interfaces", "resourceRequirements"
    ]).with_near_text({
        "concepts": [intent],
        "certainty": 0.7
    }).with_limit(5).with_additional(["certainty"]).do()
    
    return {
        "knowledge": knowledge_results["data"]["Get"]["TelecomKnowledge"],
        "patterns": pattern_results["data"]["Get"]["IntentPatterns"],
        "network_functions": nf_results["data"]["Get"]["NetworkFunctions"]
    }

# Example usage
intent = "I need to configure QoS for eMBB traffic with guaranteed bit rate"
results = intent_based_search(client, intent)

print("Relevant Knowledge:")
for doc in results["knowledge"][:3]:
    print(f"- {doc['title']} (certainty: {doc['_additional']['certainty']:.3f})")

print("\nMatching Intent Patterns:")
for pattern in results["patterns"]:
    print(f"- {pattern['pattern']} ({pattern['intentType']})")

print("\nRelevant Network Functions:")
for nf in results["network_functions"]:
    print(f"- {nf['name']}: {nf['description'][:100]}...")
```

### Multi-Vector Section Search

Search within specific document sections using targeted vectors:

```python
def section_based_search(
    client: weaviate.Client, 
    query: str, 
    section_types: List[str] = None
) -> Dict[str, List[Dict[str, Any]]]:
    """
    Search within specific document sections
    
    Args:
        client: Weaviate client instance
        query: Search query
        section_types: Types of sections to search (procedures, interfaces, etc.)
        
    Returns:
        Results organized by section type
    """
    
    section_types = section_types or ["procedures", "interfaces", "parameters", "architecture"]
    results = {}
    
    for section_type in section_types:
        # Modify query to target specific section content
        section_query = f"{section_type} {query}"
        
        result = client.query.get("TelecomKnowledge", [
            "content", "title", "source", "specification",
            "procedures", "interfaces", "category"
        ]).with_near_text({
            "concepts": [section_query],
            "certainty": 0.7
        }).with_where({
            "path": ["category"],
            "operator": "Like", 
            "valueText": f"*{section_type.capitalize()}*"
        }).with_limit(5).with_additional(["certainty", "distance"]).do()
        
        results[section_type] = result["data"]["Get"]["TelecomKnowledge"]
    
    return results

# Example usage
results = section_based_search(
    client, 
    "authentication handover", 
    ["procedures", "interfaces"]
)

for section_type, docs in results.items():
    print(f"\n{section_type.capitalize()} Results:")
    for doc in docs:
        print(f"- {doc['title']} (certainty: {doc['_additional']['certainty']:.3f})")
```

## Hybrid Search Strategies

### Balanced Hybrid Query

Combine keyword and semantic search with optimized alpha values:

```python
def balanced_hybrid_search(
    client: weaviate.Client,
    query: str,
    query_type: str = "balanced"
) -> List[Dict[str, Any]]:
    """
    Hybrid search with query-type optimized alpha values
    
    Args:
        client: Weaviate client instance
        query: Search query
        query_type: Type of query (exact_match, semantic_search, balanced, fuzzy_match)
    """
    
    # Alpha values optimized for different query types
    alpha_values = {
        "exact_match": 0.9,      # Favor keyword search
        "semantic_search": 0.3,   # Favor vector search
        "balanced": 0.7,         # Balanced approach
        "fuzzy_match": 0.5       # Equal weighting
    }
    
    alpha = alpha_values.get(query_type, 0.7)
    
    result = client.query.get("TelecomKnowledge", [
        "content", "title", "source", "specification",
        "networkFunctions", "interfaces", "procedures",
        "priority", "confidence"
    ]).with_hybrid(
        query=query,
        alpha=alpha,
        properties=["content", "title", "procedures"]  # Target specific properties
    ).with_limit(15).with_additional([
        "score", "explainScore"
    ]).do()
    
    return result["data"]["Get"]["TelecomKnowledge"]

# Example usage
queries = [
    ("TS 23.501", "exact_match"),
    ("how to configure network slicing", "semantic_search"),
    ("AMF session management procedure", "balanced"),
    ("UPF data forwarding", "fuzzy_match")
]

for query, query_type in queries:
    print(f"\nQuery: '{query}' (Type: {query_type})")
    results = balanced_hybrid_search(client, query, query_type)
    
    for doc in results[:3]:
        score = doc.get('_additional', {}).get('score', 0)
        print(f"- {doc['title']} (score: {score:.3f})")
```

### Domain-Specific Hybrid Search

Hybrid search optimized for telecommunications terminology:

```python
def telecom_hybrid_search(
    client: weaviate.Client,
    query: str,
    domain: str = "5G",
    use_case: str = None
) -> List[Dict[str, Any]]:
    """
    Telecommunications domain-optimized hybrid search
    
    Args:
        client: Weaviate client instance
        query: Search query
        domain: Technology domain (5G, O-RAN, Edge)
        use_case: Specific use case (eMBB, URLLC, mMTC)
    """
    
    # Enhance query with domain-specific context
    enhanced_query = f"{domain} {query}"
    if use_case:
        enhanced_query = f"{use_case} {enhanced_query}"
    
    # Build domain-specific filter
    where_filter = {"operator": "And", "operands": []}
    
    # Filter by domain-relevant sources
    domain_sources = {
        "5G": ["3GPP"],
        "O-RAN": ["O-RAN"],
        "Edge": ["ETSI", "O-RAN"]
    }
    
    if domain in domain_sources:
        where_filter["operands"].append({
            "path": ["source"],
            "operator": "ContainsAny",
            "valueTextArray": domain_sources[domain]
        })
    
    # Filter by use case if specified
    if use_case:
        where_filter["operands"].append({
            "path": ["useCase"],
            "operator": "Equal",
            "valueText": use_case
        })
    
    # Execute hybrid search
    query_builder = client.query.get("TelecomKnowledge", [
        "content", "title", "source", "specification", "release",
        "networkFunctions", "useCase", "technicalLevel", "confidence"
    ]).with_hybrid(
        query=enhanced_query,
        alpha=0.6,  # Slightly favor vector search for technical content
        properties=["content", "title", "networkFunctions", "procedures"]
    ).with_limit(20)
    
    if where_filter["operands"]:
        query_builder = query_builder.with_where(where_filter)
    
    result = query_builder.with_additional([
        "score", "explainScore"
    ]).do()
    
    return result["data"]["Get"]["TelecomKnowledge"]

# Example usage
results = telecom_hybrid_search(
    client,
    "session establishment authentication",
    domain="5G",
    use_case="eMBB"
)

print("5G eMBB Session Establishment Results:")
for doc in results[:5]:
    print(f"- {doc['title']} ({doc['source']} {doc.get('release', 'N/A')})")
    print(f"  Score: {doc['_additional']['score']:.3f}")
    print(f"  Functions: {', '.join(doc.get('networkFunctions', []))}")
    print()
```

## Intent Pattern Matching

### Natural Language Intent Recognition

Match user input to predefined intent patterns:

```python
def match_intent_patterns(
    client: weaviate.Client,
    user_input: str,
    confidence_threshold: float = 0.8
) -> Dict[str, Any]:
    """
    Match user input against intent patterns
    
    Args:
        client: Weaviate client instance
        user_input: User's natural language input
        confidence_threshold: Minimum confidence for pattern matching
        
    Returns:
        Matched patterns with extracted parameters
    """
    
    # Search for matching intent patterns
    pattern_results = client.query.get("IntentPatterns", [
        "pattern", "intentType", "parameters", "examples", 
        "confidence", "accuracy", "frequency"
    ]).with_near_text({
        "concepts": [user_input],
        "certainty": 0.7
    }).with_where({
        "path": ["confidence"],
        "operator": "GreaterThan",
        "valueNumber": confidence_threshold
    }).with_limit(10).with_additional([
        "certainty", "distance"
    ]).do()
    
    patterns = pattern_results["data"]["Get"]["IntentPatterns"]
    
    # Extract parameters from user input for each pattern
    matched_patterns = []
    for pattern in patterns:
        match_score = pattern["_additional"]["certainty"]
        
        # Simple parameter extraction (in practice, use NLP library)
        extracted_params = extract_parameters(user_input, pattern["parameters"])
        
        matched_patterns.append({
            "pattern": pattern["pattern"],
            "intent_type": pattern["intentType"],
            "match_score": match_score,
            "required_params": pattern["parameters"],
            "extracted_params": extracted_params,
            "confidence": pattern["confidence"],
            "examples": pattern.get("examples", [])
        })
    
    # Sort by match score
    matched_patterns.sort(key=lambda x: x["match_score"], reverse=True)
    
    return {
        "user_input": user_input,
        "matched_patterns": matched_patterns,
        "best_match": matched_patterns[0] if matched_patterns else None
    }

def extract_parameters(text: str, param_names: List[str]) -> Dict[str, str]:
    """
    Simple parameter extraction from text
    (In production, use more sophisticated NLP)
    """
    import re
    
    extracted = {}
    
    # Common patterns for telecom parameters
    patterns = {
        "networkFunction": r"\b(AMF|SMF|UPF|gNB|CU|DU|RU|AUSF|UDM|PCF|NRF|NSSF)\b",
        "replicas": r"\b(\d+)\s*(?:replica|instance|copy|copies)s?\b",
        "namespace": r"(?:namespace|ns)\s+([a-z0-9-]+)",
        "slice": r"(?:slice|s-nssai)\s+([0-9-]+)",
        "interface": r"\b(N[1-9]|E[1-9]|F[1-9]|A[1-9]|O[1-9])\b"
    }
    
    for param in param_names:
        if param in patterns:
            match = re.search(patterns[param], text, re.IGNORECASE)
            if match:
                extracted[param] = match.group(1)
    
    return extracted

# Example usage
user_inputs = [
    "Deploy AMF with 3 replicas in production namespace",
    "Scale SMF to 5 instances for eMBB slice",
    "Configure UPF with high throughput for URLLC"
]

for user_input in user_inputs:
    print(f"\nInput: '{user_input}'")
    results = match_intent_patterns(client, user_input)
    
    if results["best_match"]:
        best = results["best_match"]
        print(f"Best Match: {best['pattern']} ({best['intent_type']})")
        print(f"Confidence: {best['match_score']:.3f}")
        print(f"Extracted Parameters: {best['extracted_params']}")
    else:
        print("No matching patterns found")
```

### Intent-to-Action Mapping

Convert matched intents to executable actions:

```python
def intent_to_action(
    client: weaviate.Client,
    matched_intent: Dict[str, Any]
) -> Dict[str, Any]:
    """
    Convert matched intent to actionable deployment configuration
    
    Args:
        client: Weaviate client instance
        matched_intent: Result from match_intent_patterns
        
    Returns:
        Executable action configuration
    """
    
    if not matched_intent["best_match"]:
        return {"error": "No intent match found"}
    
    best_match = matched_intent["best_match"]
    intent_type = best_match["intent_type"]
    params = best_match["extracted_params"]
    
    # Look up network function details if needed
    if "networkFunction" in params:
        nf_name = params["networkFunction"]
        nf_results = client.query.get("NetworkFunctions", [
            "name", "description", "deploymentOptions", 
            "resourceRequirements", "configurationTemplates",
            "interfaces", "scalingOptions"
        ]).with_where({
            "path": ["name"],
            "operator": "Equal",
            "valueText": nf_name
        }).with_limit(1).do()
        
        nf_info = nf_results["data"]["Get"]["NetworkFunctions"]
        nf_details = nf_info[0] if nf_info else {}
    else:
        nf_details = {}
    
    # Generate action based on intent type
    if intent_type == "NetworkFunctionDeployment":
        action = {
            "action_type": "deploy",
            "resource": "NetworkFunction",
            "spec": {
                "name": params.get("networkFunction", "unknown"),
                "replicas": int(params.get("replicas", 1)),
                "namespace": params.get("namespace", "default"),
                "deployment_options": nf_details.get("deploymentOptions", []),
                "resource_requirements": nf_details.get("resourceRequirements", "standard"),
                "interfaces": nf_details.get("interfaces", [])
            }
        }
    
    elif intent_type == "NetworkFunctionScale":
        action = {
            "action_type": "scale",
            "resource": "NetworkFunction", 
            "spec": {
                "name": params.get("networkFunction", "unknown"),
                "target_replicas": int(params.get("replicas", 1)),
                "scaling_options": nf_details.get("scalingOptions", "horizontal")
            }
        }
    
    elif intent_type == "NetworkSliceConfiguration":
        action = {
            "action_type": "configure",
            "resource": "NetworkSlice",
            "spec": {
                "slice_id": params.get("slice", "default"),
                "network_functions": [params.get("networkFunction", "AMF")],
                "sla_requirements": extract_sla_from_intent(matched_intent["user_input"])
            }
        }
    
    else:
        action = {
            "action_type": "unknown",
            "error": f"Unsupported intent type: {intent_type}"
        }
    
    return {
        "original_intent": matched_intent["user_input"],
        "matched_pattern": best_match["pattern"],
        "action": action,
        "confidence": best_match["match_score"],
        "network_function_details": nf_details
    }

def extract_sla_from_intent(text: str) -> Dict[str, Any]:
    """Extract SLA requirements from natural language"""
    import re
    
    sla = {}
    
    # Throughput patterns
    throughput_match = re.search(r"(\d+(?:\.\d+)?)\s*(Gbps|Mbps|kbps)", text, re.IGNORECASE)
    if throughput_match:
        value = float(throughput_match.group(1))
        unit = throughput_match.group(2).lower()
        sla["throughput"] = f"{value}{unit}"
    
    # Latency patterns
    latency_match = re.search(r"(\d+(?:\.\d+)?)\s*(ms|μs|us)", text, re.IGNORECASE)
    if latency_match:
        value = float(latency_match.group(1))
        unit = latency_match.group(2).lower()
        sla["latency"] = f"{value}{unit}"
    
    # Reliability patterns
    if re.search(r"high.*reliability|99\.9", text, re.IGNORECASE):
        sla["reliability"] = "99.999%"
    
    # Use case detection
    if re.search(r"eMBB|enhanced.*mobile", text, re.IGNORECASE):
        sla["use_case"] = "eMBB"
    elif re.search(r"URLLC|ultra.*reliable", text, re.IGNORECASE):
        sla["use_case"] = "URLLC"
    elif re.search(r"mMTC|massive.*IoT", text, re.IGNORECASE):
        sla["use_case"] = "mMTC"
    
    return sla

# Example usage
user_input = "Deploy AMF with 3 replicas for high throughput eMBB slice"
intent_match = match_intent_patterns(client, user_input)
action_config = intent_to_action(client, intent_match)

print("Generated Action Configuration:")
print(f"Action Type: {action_config['action']['action_type']}")
print(f"Resource: {action_config['action']['resource']}")
print(f"Spec: {action_config['action']['spec']}")
print(f"Confidence: {action_config['confidence']:.3f}")
```

## Cross-Reference Queries

### Knowledge-to-Intent Mapping

Find intent patterns related to specific knowledge articles:

```python
def find_related_intents(
    client: weaviate.Client,
    knowledge_id: str = None,
    knowledge_query: str = None
) -> Dict[str, Any]:
    """
    Find intent patterns related to specific knowledge
    
    Args:
        client: Weaviate client instance
        knowledge_id: Specific knowledge document ID
        knowledge_query: Query to find knowledge first
        
    Returns:
        Related intents and network functions
    """
    
    # Get knowledge document
    if knowledge_id:
        knowledge_result = client.query.get("TelecomKnowledge", [
            "content", "title", "networkFunctions", "procedures", "interfaces"
        ]).with_where({
            "path": ["id"],
            "operator": "Equal",
            "valueText": knowledge_id
        }).with_limit(1).do()
        
        knowledge_docs = knowledge_result["data"]["Get"]["TelecomKnowledge"]
    
    elif knowledge_query:
        knowledge_result = client.query.get("TelecomKnowledge", [
            "content", "title", "networkFunctions", "procedures", "interfaces"
        ]).with_near_text({
            "concepts": [knowledge_query],
            "certainty": 0.8
        }).with_limit(1).do()
        
        knowledge_docs = knowledge_result["data"]["Get"]["TelecomKnowledge"]
    
    else:
        return {"error": "Either knowledge_id or knowledge_query must be provided"}
    
    if not knowledge_docs:
        return {"error": "No knowledge document found"}
    
    knowledge_doc = knowledge_docs[0]
    
    # Find related intent patterns using network functions and procedures
    search_terms = []
    search_terms.extend(knowledge_doc.get("networkFunctions", []))
    search_terms.extend(knowledge_doc.get("procedures", []))
    
    related_intents = []
    for term in search_terms:
        intent_result = client.query.get("IntentPatterns", [
            "pattern", "intentType", "parameters", "confidence", "examples"
        ]).with_near_text({
            "concepts": [term],
            "certainty": 0.7
        }).with_limit(3).with_additional(["certainty"]).do()
        
        related_intents.extend(intent_result["data"]["Get"]["IntentPatterns"])
    
    # Find related network functions
    nf_search_query = f"{knowledge_doc['title']} {' '.join(knowledge_doc.get('procedures', []))}"
    nf_result = client.query.get("NetworkFunctions", [
        "name", "description", "category", "interfaces", "deploymentOptions"
    ]).with_near_text({
        "concepts": [nf_search_query],
        "certainty": 0.7
    }).with_limit(5).with_additional(["certainty"]).do()
    
    related_nfs = nf_result["data"]["Get"]["NetworkFunctions"]
    
    # Remove duplicates and sort by relevance
    unique_intents = {}
    for intent in related_intents:
        pattern = intent["pattern"]
        if pattern not in unique_intents or intent["_additional"]["certainty"] > unique_intents[pattern]["_additional"]["certainty"]:
            unique_intents[pattern] = intent
    
    sorted_intents = sorted(unique_intents.values(), key=lambda x: x["_additional"]["certainty"], reverse=True)
    sorted_nfs = sorted(related_nfs, key=lambda x: x["_additional"]["certainty"], reverse=True)
    
    return {
        "knowledge_document": knowledge_doc,
        "related_intents": sorted_intents[:10],
        "related_network_functions": sorted_nfs[:5],
        "recommendations": generate_intent_recommendations(knowledge_doc, sorted_intents, sorted_nfs)
    }

def generate_intent_recommendations(
    knowledge_doc: Dict[str, Any],
    intents: List[Dict[str, Any]], 
    network_functions: List[Dict[str, Any]]
) -> List[str]:
    """Generate actionable recommendations based on cross-references"""
    
    recommendations = []
    
    # Deployment recommendations
    nf_names = [nf["name"] for nf in network_functions[:3]]
    if nf_names:
        recommendations.append(f"Consider deploying: {', '.join(nf_names)}")
    
    # Configuration recommendations
    config_intents = [i for i in intents if "Configuration" in i["intentType"]]
    if config_intents:
        recommendations.append(f"Relevant configurations: {config_intents[0]['pattern']}")
    
    # Procedure recommendations
    procedures = knowledge_doc.get("procedures", [])
    if procedures:
        recommendations.append(f"Related procedures: {', '.join(procedures[:3])}")
    
    return recommendations

# Example usage
results = find_related_intents(
    client, 
    knowledge_query="AMF authentication and key agreement procedure"
)

print("Knowledge Document:", results["knowledge_document"]["title"])
print("\nRelated Intent Patterns:")
for intent in results["related_intents"][:3]:
    print(f"- {intent['pattern']} ({intent['intentType']})")
    print(f"  Certainty: {intent['_additional']['certainty']:.3f}")

print("\nRelated Network Functions:")
for nf in results["related_network_functions"][:3]:
    print(f"- {nf['name']}: {nf['description'][:100]}...")

print("\nRecommendations:")
for rec in results["recommendations"]:
    print(f"- {rec}")
```

### Multi-Class Aggregation Queries

Aggregate information across all three schema classes:

```python
def comprehensive_domain_analysis(
    client: weaviate.Client,
    domain_query: str,
    analysis_depth: str = "standard"
) -> Dict[str, Any]:
    """
    Comprehensive analysis across all schema classes
    
    Args:
        client: Weaviate client instance
        domain_query: Domain or topic to analyze
        analysis_depth: Level of analysis (basic, standard, deep)
        
    Returns:
        Comprehensive domain analysis
    """
    
    limits = {
        "basic": {"knowledge": 5, "patterns": 3, "nfs": 3},
        "standard": {"knowledge": 15, "patterns": 10, "nfs": 10},
        "deep": {"knowledge": 50, "patterns": 25, "nfs": 20}
    }
    
    limit = limits.get(analysis_depth, limits["standard"])
    
    # Knowledge analysis
    knowledge_analysis = client.query.get("TelecomKnowledge", [
        "title", "source", "specification", "release", "category",
        "networkFunctions", "interfaces", "procedures", "useCase", "confidence"
    ]).with_near_text({
        "concepts": [domain_query],
        "certainty": 0.7
    }).with_limit(limit["knowledge"]).with_additional(["certainty"]).do()
    
    knowledge_docs = knowledge_analysis["data"]["Get"]["TelecomKnowledge"]
    
    # Intent pattern analysis
    pattern_analysis = client.query.get("IntentPatterns", [
        "pattern", "intentType", "parameters", "confidence", "accuracy", "frequency"
    ]).with_near_text({
        "concepts": [domain_query],
        "certainty": 0.7
    }).with_limit(limit["patterns"]).with_additional(["certainty"]).do()
    
    intent_patterns = pattern_analysis["data"]["Get"]["IntentPatterns"]
    
    # Network function analysis
    nf_analysis = client.query.get("NetworkFunctions", [
        "name", "description", "category", "interfaces", "deploymentOptions",
        "resourceRequirements", "scalingOptions", "standardsCompliance"
    ]).with_near_text({
        "concepts": [domain_query],
        "certainty": 0.7
    }).with_limit(limit["nfs"]).with_additional(["certainty"]).do()
    
    network_functions = nf_analysis["data"]["Get"]["NetworkFunctions"]
    
    # Aggregate statistics
    stats = calculate_domain_statistics(knowledge_docs, intent_patterns, network_functions)
    
    # Generate insights
    insights = generate_domain_insights(knowledge_docs, intent_patterns, network_functions, domain_query)
    
    return {
        "domain_query": domain_query,
        "analysis_depth": analysis_depth,
        "knowledge_documents": knowledge_docs,
        "intent_patterns": intent_patterns,
        "network_functions": network_functions,
        "statistics": stats,
        "insights": insights,
        "recommendations": generate_domain_recommendations(stats, insights)
    }

def calculate_domain_statistics(knowledge_docs, intent_patterns, network_functions):
    """Calculate aggregate statistics across all classes"""
    
    # Knowledge statistics
    sources = {}
    categories = {}
    nf_mentions = {}
    
    for doc in knowledge_docs:
        source = doc.get("source", "unknown")
        sources[source] = sources.get(source, 0) + 1
        
        category = doc.get("category", "unknown")  
        categories[category] = categories.get(category, 0) + 1
        
        for nf in doc.get("networkFunctions", []):
            nf_mentions[nf] = nf_mentions.get(nf, 0) + 1
    
    # Intent statistics
    intent_types = {}
    for pattern in intent_patterns:
        intent_type = pattern.get("intentType", "unknown")
        intent_types[intent_type] = intent_types.get(intent_type, 0) + 1
    
    # Network function statistics
    nf_categories = {}
    for nf in network_functions:
        category = nf.get("category", "unknown")
        nf_categories[category] = nf_categories.get(category, 0) + 1
    
    return {
        "knowledge_sources": sources,
        "knowledge_categories": categories, 
        "network_function_mentions": nf_mentions,
        "intent_types": intent_types,
        "nf_categories": nf_categories,
        "total_knowledge_docs": len(knowledge_docs),
        "total_intent_patterns": len(intent_patterns),
        "total_network_functions": len(network_functions)
    }

def generate_domain_insights(knowledge_docs, intent_patterns, network_functions, domain_query):
    """Generate insights from the aggregated data"""
    
    insights = []
    
    # Knowledge insights
    if knowledge_docs:
        avg_confidence = sum(doc.get("confidence", 0) for doc in knowledge_docs) / len(knowledge_docs)
        insights.append(f"Average knowledge confidence: {avg_confidence:.2f}")
        
        main_source = max(set(doc.get("source", "") for doc in knowledge_docs), 
                         key=lambda x: sum(1 for doc in knowledge_docs if doc.get("source") == x))
        insights.append(f"Primary knowledge source: {main_source}")
    
    # Intent insights
    if intent_patterns:
        main_intent_type = max(set(p.get("intentType", "") for p in intent_patterns),
                              key=lambda x: sum(1 for p in intent_patterns if p.get("intentType") == x))
        insights.append(f"Primary intent type: {main_intent_type}")
    
    # Network function insights
    if network_functions:
        main_nf_category = max(set(nf.get("category", "") for nf in network_functions),
                              key=lambda x: sum(1 for nf in network_functions if nf.get("category") == x))
        insights.append(f"Primary NF category: {main_nf_category}")
    
    # Cross-domain insights
    common_nfs = set()
    for doc in knowledge_docs:
        common_nfs.update(doc.get("networkFunctions", []))
    
    nf_names = set(nf.get("name", "") for nf in network_functions)
    overlap = common_nfs.intersection(nf_names)
    
    if overlap:
        insights.append(f"Common network functions: {', '.join(list(overlap)[:5])}")
    
    return insights

def generate_domain_recommendations(stats, insights):
    """Generate actionable recommendations based on analysis"""
    
    recommendations = []
    
    # Knowledge recommendations
    if stats["total_knowledge_docs"] > 10:
        recommendations.append("Rich knowledge base available - consider implementing advanced semantic search")
    
    # Intent recommendations  
    if stats["total_intent_patterns"] > 5:
        recommendations.append("Multiple intent patterns identified - implement pattern-based automation")
    
    # Network function recommendations
    if stats["total_network_functions"] > 3:
        recommendations.append("Multiple NF options available - consider deployment optimization")
    
    # Cross-domain recommendations
    if stats["total_knowledge_docs"] > 0 and stats["total_intent_patterns"] > 0:
        recommendations.append("Enable knowledge-to-intent mapping for automated operations")
    
    return recommendations

# Example usage
analysis = comprehensive_domain_analysis(
    client,
    "network slicing and QoS management",
    analysis_depth="standard"
)

print(f"Domain Analysis: {analysis['domain_query']}")
print(f"\nStatistics:")
print(f"- Knowledge documents: {analysis['statistics']['total_knowledge_docs']}")
print(f"- Intent patterns: {analysis['statistics']['total_intent_patterns']}")
print(f"- Network functions: {analysis['statistics']['total_network_functions']}")

print(f"\nKey Insights:")
for insight in analysis['insights']:
    print(f"- {insight}")
    
print(f"\nRecommendations:")
for rec in analysis['recommendations']:
    print(f"- {rec}")
```

## Batch Operations

### Bulk Vector Operations

Efficient batch processing for large-scale operations:

```python
def bulk_knowledge_ingestion(
    client: weaviate.Client,
    documents: List[Dict[str, Any]],
    batch_size: int = 100
) -> Dict[str, Any]:
    """
    Bulk ingest knowledge documents with optimized batching
    
    Args:
        client: Weaviate client instance
        documents: List of documents to ingest
        batch_size: Number of documents per batch
        
    Returns:
        Ingestion results and statistics
    """
    
    import time
    from datetime import datetime
    
    total_docs = len(documents)
    successful = 0
    failed = 0
    batches_processed = 0
    errors = []
    
    print(f"Starting bulk ingestion of {total_docs} documents...")
    start_time = time.time()
    
    # Configure batch settings for optimal performance
    client.batch.configure(
        batch_size=batch_size,
        dynamic=True,
        timeout_retries=3,
        callback=lambda batch_results: print(f"Batch {batches_processed + 1} completed")
    )
    
    with client.batch as batch:
        for i, doc in enumerate(documents):
            try:
                # Validate document structure
                if not validate_document_structure(doc):
                    failed += 1
                    errors.append(f"Document {i}: Invalid structure")
                    continue
                
                # Enhance document with metadata
                enhanced_doc = enhance_document_metadata(doc)
                
                # Add to batch
                batch.add_data_object(
                    data_object=enhanced_doc,
                    class_name="TelecomKnowledge",
                    uuid=enhanced_doc.get("id")  # Use provided ID if available
                )
                
                successful += 1
                
                # Progress reporting
                if (i + 1) % 500 == 0:
                    progress = ((i + 1) / total_docs) * 100
                    elapsed = time.time() - start_time
                    rate = (i + 1) / elapsed
                    print(f"Progress: {progress:.1f}% ({i+1}/{total_docs}) - Rate: {rate:.1f} docs/sec")
                
            except Exception as e:
                failed += 1
                errors.append(f"Document {i}: {str(e)}")
                continue
    
    end_time = time.time()
    duration = end_time - start_time
    
    results = {
        "total_documents": total_docs,
        "successful": successful,
        "failed": failed,
        "errors": errors[:10],  # Show first 10 errors
        "duration_seconds": duration,
        "rate_docs_per_second": successful / duration if duration > 0 else 0,
        "timestamp": datetime.now().isoformat()
    }
    
    print(f"Bulk ingestion completed:")
    print(f"- Total: {total_docs}")
    print(f"- Successful: {successful}")
    print(f"- Failed: {failed}")
    print(f"- Duration: {duration:.2f} seconds")
    print(f"- Rate: {results['rate_docs_per_second']:.1f} docs/sec")
    
    return results

def validate_document_structure(doc: Dict[str, Any]) -> bool:
    """Validate document has required fields"""
    required_fields = ["content", "title", "source"]
    return all(field in doc for field in required_fields)

def enhance_document_metadata(doc: Dict[str, Any]) -> Dict[str, Any]:
    """Enhance document with additional metadata"""
    from datetime import datetime
    import hashlib
    
    enhanced = doc.copy()
    
    # Add ingestion timestamp
    enhanced["lastUpdated"] = datetime.now().isoformat()
    
    # Generate content hash for deduplication
    content_hash = hashlib.md5(doc["content"].encode()).hexdigest()
    enhanced["contentHash"] = content_hash
    
    # Set default values for optional fields
    enhanced.setdefault("confidence", 0.8)
    enhanced.setdefault("priority", 5)
    enhanced.setdefault("language", "en")
    enhanced.setdefault("technicalLevel", "Intermediate")
    
    # Extract technical terms (simplified)
    enhanced["keywords"] = extract_technical_terms(doc["content"])
    
    return enhanced

def extract_technical_terms(content: str) -> List[str]:
    """Extract telecommunications technical terms"""
    import re
    
    # Common telecom term patterns
    patterns = [
        r'\b(?:AMF|SMF|UPF|gNB|CU|DU|RU|AUSF|UDM|PCF|NRF|NSSF)\b',
        r'\b(?:5G|LTE|NR|SA|NSA|eMBB|URLLC|mMTC)\b',
        r'\b(?:N[1-9]|E[1-9]|F[1-9]|A[1-9]|O[1-9])\b',
        r'\b(?:PDU|QoS|SLA|KPI|RAN|CN|MEC)\b'
    ]
    
    terms = set()
    for pattern in patterns:
        matches = re.findall(pattern, content, re.IGNORECASE)
        terms.update(matches)
    
    return list(terms)[:10]  # Limit to top 10 terms

# Example usage
sample_documents = [
    {
        "content": "The AMF (Access and Mobility Management Function) handles registration procedures...",
        "title": "AMF Registration Procedures",
        "source": "3GPP",
        "specification": "TS 23.501",
        "version": "17.9.0",
        "release": "Rel-17",
        "category": "Procedures",
        "networkFunctions": ["AMF"],
        "procedures": ["Registration", "Authentication"]
    },
    # ... more documents
]

# Run bulk ingestion
results = bulk_knowledge_ingestion(client, sample_documents, batch_size=50)
```

### Batch Query Operations

Execute multiple queries efficiently:

```python
def batch_semantic_queries(
    client: weaviate.Client,
    queries: List[str],
    query_type: str = "hybrid"
) -> Dict[str, List[Dict[str, Any]]]:
    """
    Execute multiple semantic queries in parallel
    
    Args:
        client: Weaviate client instance
        queries: List of query strings
        query_type: Type of query (vector, hybrid, keyword)
        
    Returns:
        Results for each query
    """
    
    import concurrent.futures
    import time
    
    def execute_single_query(query: str) -> List[Dict[str, Any]]:
        """Execute a single query"""
        try:
            if query_type == "vector":
                result = client.query.get("TelecomKnowledge", [
                    "content", "title", "source", "networkFunctions", "confidence"
                ]).with_near_text({
                    "concepts": [query],
                    "certainty": 0.7
                }).with_limit(10).with_additional(["certainty"]).do()
            
            elif query_type == "hybrid":
                result = client.query.get("TelecomKnowledge", [
                    "content", "title", "source", "networkFunctions", "confidence"
                ]).with_hybrid(
                    query=query,
                    alpha=0.7
                ).with_limit(10).with_additional(["score"]).do()
            
            elif query_type == "keyword":
                result = client.query.get("TelecomKnowledge", [
                    "content", "title", "source", "networkFunctions", "confidence"
                ]).with_bm25(
                    query=query
                ).with_limit(10).with_additional(["score"]).do()
            
            else:
                raise ValueError(f"Unsupported query type: {query_type}")
            
            return result["data"]["Get"]["TelecomKnowledge"]
        
        except Exception as e:
            print(f"Query failed: {query} - Error: {str(e)}")
            return []
    
    start_time = time.time()
    results = {}
    
    # Execute queries in parallel
    with concurrent.futures.ThreadPoolExecutor(max_workers=5) as executor:
        future_to_query = {
            executor.submit(execute_single_query, query): query 
            for query in queries
        }
        
        for future in concurrent.futures.as_completed(future_to_query):
            query = future_to_query[future]
            try:
                query_results = future.result()
                results[query] = query_results
                print(f"Completed query: '{query}' - {len(query_results)} results")
            except Exception as e:
                print(f"Query failed: '{query}' - Error: {str(e)}")
                results[query] = []
    
    duration = time.time() - start_time
    print(f"Batch query completed in {duration:.2f} seconds")
    
    return results

# Example usage
test_queries = [
    "AMF registration procedure",
    "SMF session management", 
    "UPF data forwarding",
    "Network slicing configuration",
    "QoS policy enforcement"
]

batch_results = batch_semantic_queries(client, test_queries, query_type="hybrid")

for query, results in batch_results.items():
    print(f"\nQuery: '{query}' - {len(results)} results")
    for i, doc in enumerate(results[:3]):
        score_key = "score" if "_additional" in doc and "score" in doc["_additional"] else "certainty"
        score = doc.get("_additional", {}).get(score_key, 0)
        print(f"  {i+1}. {doc['title']} (score: {score:.3f})")
```

## Performance Optimization

### Query Performance Tuning

Optimize queries for different performance requirements:

```python
def optimized_query_builder(
    client: weaviate.Client,
    query: str,
    performance_mode: str = "balanced",
    result_limit: int = 10
) -> List[Dict[str, Any]]:
    """
    Build optimized queries based on performance requirements
    
    Args:
        client: Weaviate client instance
        query: Search query
        performance_mode: fast, balanced, accurate, comprehensive
        result_limit: Maximum number of results
        
    Returns:
        Optimized query results
    """
    
    # Performance mode configurations
    configs = {
        "fast": {
            "certainty": 0.6,
            "alpha": 0.8,  # Favor keyword search
            "properties": ["title", "networkFunctions"],
            "additional": ["score"],
            "limit": min(result_limit, 20)
        },
        "balanced": {
            "certainty": 0.7,
            "alpha": 0.7,
            "properties": ["content", "title", "networkFunctions", "procedures"],
            "additional": ["score", "certainty"],
            "limit": result_limit
        },
        "accurate": {
            "certainty": 0.8,
            "alpha": 0.3,  # Favor vector search
            "properties": ["content", "title", "source", "networkFunctions", "procedures", "interfaces"],
            "additional": ["score", "certainty", "distance"],
            "limit": result_limit * 2  # Get more results for better accuracy
        },
        "comprehensive": {
            "certainty": 0.6,
            "alpha": 0.5,
            "properties": [
                "content", "title", "source", "specification", "version",
                "networkFunctions", "procedures", "interfaces", "category",
                "confidence", "priority", "technicalLevel"
            ],
            "additional": ["score", "certainty", "distance", "explainScore"],
            "limit": result_limit * 3
        }
    }
    
    config = configs.get(performance_mode, configs["balanced"])
    
    # Build optimized query
    result = client.query.get("TelecomKnowledge", config["properties"]).with_hybrid(
        query=query,
        alpha=config["alpha"],
        properties=config["properties"][:4]  # Limit vectorized properties
    ).with_limit(config["limit"]).with_additional(config["additional"]).do()
    
    documents = result["data"]["Get"]["TelecomKnowledge"]
    
    # Post-process results based on performance mode
    if performance_mode == "accurate":
        # Re-rank by certainty for accuracy mode
        documents = sorted(documents, 
                         key=lambda x: x.get("_additional", {}).get("certainty", 0), 
                         reverse=True)[:result_limit]
    
    elif performance_mode == "comprehensive":
        # Add relevance scoring for comprehensive mode
        documents = add_relevance_scoring(documents, query)[:result_limit]
    
    return documents

def add_relevance_scoring(documents: List[Dict[str, Any]], query: str) -> List[Dict[str, Any]]:
    """Add custom relevance scoring to documents"""
    
    query_terms = set(query.lower().split())
    
    for doc in documents:
        relevance_score = 0.0
        
        # Title match bonus
        title_terms = set(doc.get("title", "").lower().split())
        title_overlap = len(query_terms.intersection(title_terms))
        relevance_score += title_overlap * 0.3
        
        # Network function match bonus
        nf_terms = set(nf.lower() for nf in doc.get("networkFunctions", []))
        nf_overlap = len(query_terms.intersection(nf_terms))
        relevance_score += nf_overlap * 0.4
        
        # Confidence bonus
        confidence = doc.get("confidence", 0.5)
        relevance_score += confidence * 0.2
        
        # Priority bonus
        priority = doc.get("priority", 5)
        relevance_score += (priority / 10) * 0.1
        
        # Store relevance score
        if "_additional" not in doc:
            doc["_additional"] = {}
        doc["_additional"]["relevance_score"] = relevance_score
    
    # Sort by relevance score
    return sorted(documents, 
                 key=lambda x: x.get("_additional", {}).get("relevance_score", 0), 
                 reverse=True)

# Example usage and performance comparison
test_query = "AMF authentication procedure for network slicing"
performance_modes = ["fast", "balanced", "accurate", "comprehensive"]

print("Performance Mode Comparison:")
print("=" * 50)

for mode in performance_modes:
    import time
    start_time = time.time()
    
    results = optimized_query_builder(client, test_query, performance_mode=mode, result_limit=10)
    
    duration = time.time() - start_time
    
    print(f"\n{mode.capitalize()} Mode:")
    print(f"Duration: {duration:.3f} seconds")
    print(f"Results: {len(results)}")
    
    if results:
        best_result = results[0]
        additional = best_result.get("_additional", {})
        score = additional.get("score", additional.get("certainty", 0))
        print(f"Best match: {best_result['title']} (score: {score:.3f})")
        
        if "relevance_score" in additional:
            print(f"Relevance: {additional['relevance_score']:.3f}")
```

### Caching and Memoization

Implement intelligent caching for frequently used queries:

```python
import functools
import hashlib
import time
from typing import Any, Callable

class QueryCache:
    """Intelligent query cache with TTL and LRU eviction"""
    
    def __init__(self, max_size: int = 1000, ttl_seconds: int = 3600):
        self.max_size = max_size
        self.ttl_seconds = ttl_seconds
        self.cache = {}
        self.access_times = {}
        self.creation_times = {}
    
    def _generate_key(self, query: str, **kwargs) -> str:
        """Generate cache key from query and parameters"""
        key_data = f"{query}_{sorted(kwargs.items())}"
        return hashlib.md5(key_data.encode()).hexdigest()
    
    def _is_expired(self, key: str) -> bool:
        """Check if cache entry is expired"""
        if key not in self.creation_times:
            return True
        return time.time() - self.creation_times[key] > self.ttl_seconds
    
    def _evict_lru(self):
        """Evict least recently used entries"""
        if len(self.cache) < self.max_size:
            return
        
        # Remove expired entries first
        expired_keys = [k for k in self.cache.keys() if self._is_expired(k)]
        for key in expired_keys:
            self._remove_entry(key)
        
        # If still over capacity, remove LRU entries
        while len(self.cache) >= self.max_size:
            lru_key = min(self.access_times.keys(), key=self.access_times.get)
            self._remove_entry(lru_key)
    
    def _remove_entry(self, key: str):
        """Remove cache entry and metadata"""
        self.cache.pop(key, None)
        self.access_times.pop(key, None)
        self.creation_times.pop(key, None)
    
    def get(self, key: str) -> Any:
        """Get cached result"""
        if key in self.cache and not self._is_expired(key):
            self.access_times[key] = time.time()
            return self.cache[key]
        return None
    
    def set(self, key: str, value: Any):
        """Cache result"""
        self._evict_lru()
        
        current_time = time.time()
        self.cache[key] = value
        self.access_times[key] = current_time
        self.creation_times[key] = current_time
    
    def stats(self) -> Dict[str, Any]:
        """Get cache statistics"""
        current_time = time.time()
        expired_count = sum(1 for k in self.creation_times.keys() if self._is_expired(k))
        
        return {
            "size": len(self.cache),
            "max_size": self.max_size,
            "expired_entries": expired_count,
            "hit_rate": getattr(self, "_hits", 0) / max(getattr(self, "_total", 1), 1),
            "avg_age": sum(current_time - t for t in self.creation_times.values()) / max(len(self.creation_times), 1)
        }

# Global cache instance
query_cache = QueryCache(max_size=500, ttl_seconds=1800)  # 30 minutes TTL

def cached_query(cache_key_func: Callable = None):
    """Decorator for caching query results"""
    
    def decorator(func):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            # Generate cache key
            if cache_key_func:
                cache_key = cache_key_func(*args, **kwargs)
            else:
                # Default key generation
                key_parts = [str(arg) for arg in args[1:]]  # Skip client object
                key_parts.extend(f"{k}={v}" for k, v in sorted(kwargs.items()))
                cache_key = query_cache._generate_key("_".join(key_parts))
            
            # Check cache
            cached_result = query_cache.get(cache_key)
            if cached_result is not None:
                if not hasattr(query_cache, "_hits"):
                    query_cache._hits = 0
                    query_cache._total = 0
                query_cache._hits += 1
                query_cache._total += 1
                return cached_result
            
            # Execute query
            result = func(*args, **kwargs)
            
            # Cache result
            query_cache.set(cache_key, result)
            
            if not hasattr(query_cache, "_total"):
                query_cache._total = 0
            query_cache._total += 1
            
            return result
        return wrapper
    return decorator

def semantic_search_key(client, query: str, **kwargs) -> str:
    """Generate cache key for semantic search"""
    return query_cache._generate_key(query, **kwargs)

@cached_query(cache_key_func=semantic_search_key)
def cached_semantic_search(
    client: weaviate.Client,
    query: str,
    certainty: float = 0.7,
    limit: int = 10
) -> List[Dict[str, Any]]:
    """Cached semantic search"""
    
    result = client.query.get("TelecomKnowledge", [
        "content", "title", "source", "networkFunctions", "confidence"
    ]).with_near_text({
        "concepts": [query],
        "certainty": certainty
    }).with_limit(limit).with_additional(["certainty"]).do()
    
    return result["data"]["Get"]["TelecomKnowledge"]

# Example usage with caching
print("Testing cached queries:")

# First call - will hit the database
start_time = time.time()
results1 = cached_semantic_search(client, "AMF registration procedure", certainty=0.7, limit=10)
duration1 = time.time() - start_time
print(f"First call: {duration1:.3f} seconds - {len(results1)} results")

# Second call - should hit cache
start_time = time.time()
results2 = cached_semantic_search(client, "AMF registration procedure", certainty=0.7, limit=10)
duration2 = time.time() - start_time
print(f"Second call: {duration2:.3f} seconds - {len(results2)} results")

# Cache statistics
print(f"Cache stats: {query_cache.stats()}")
```

## Error Handling

### Robust Error Handling Patterns

Implement comprehensive error handling for production use:

```python
import logging
import time
from typing import Optional, Union
from enum import Enum

class QueryErrorType(Enum):
    CONNECTION_ERROR = "connection_error"
    TIMEOUT_ERROR = "timeout_error"
    RATE_LIMIT_ERROR = "rate_limit_error"
    VALIDATION_ERROR = "validation_error"
    VECTORIZER_ERROR = "vectorizer_error"
    UNKNOWN_ERROR = "unknown_error"

class QueryError(Exception):
    """Custom exception for query errors"""
    
    def __init__(self, message: str, error_type: QueryErrorType, retry_after: Optional[int] = None):
        super().__init__(message)
        self.error_type = error_type
        self.retry_after = retry_after
        self.timestamp = time.time()

def robust_query_executor(
    client: weaviate.Client,
    query_func: Callable,
    max_retries: int = 3,
    backoff_factor: float = 2.0,
    timeout: int = 30
) -> Union[List[Dict[str, Any]], Dict[str, Any]]:
    """
    Execute queries with comprehensive error handling and retry logic
    
    Args:
        client: Weaviate client instance
        query_func: Function that executes the query
        max_retries: Maximum number of retry attempts
        backoff_factor: Exponential backoff multiplier
        timeout: Query timeout in seconds
        
    Returns:
        Query results or error information
    """
    
    logger = logging.getLogger(__name__)
    
    for attempt in range(max_retries + 1):
        try:
            # Set timeout for the query
            start_time = time.time()
            
            # Execute the query function
            result = query_func(client)
            
            # Check if query took too long
            duration = time.time() - start_time
            if duration > timeout:
                raise QueryError(
                    f"Query timeout after {duration:.2f} seconds",
                    QueryErrorType.TIMEOUT_ERROR
                )
            
            logger.info(f"Query successful in {duration:.3f} seconds")
            return result
        
        except Exception as e:
            error_type = classify_error(e)
            
            # Log the error
            logger.warning(f"Query attempt {attempt + 1} failed: {str(e)} (Type: {error_type.value})")
            
            # Check if we should retry
            if attempt >= max_retries:
                logger.error(f"Query failed after {max_retries + 1} attempts")
                return {
                    "error": True,
                    "error_type": error_type.value,
                    "message": str(e),
                    "attempts": attempt + 1,
                    "timestamp": time.time()
                }
            
            # Calculate backoff delay
            if error_type == QueryErrorType.RATE_LIMIT_ERROR:
                # For rate limits, wait longer
                delay = min(60, backoff_factor ** attempt * 5)
            else:
                delay = backoff_factor ** attempt
            
            logger.info(f"Retrying in {delay:.1f} seconds...")
            time.sleep(delay)
    
    return {
        "error": True,
        "error_type": QueryErrorType.UNKNOWN_ERROR.value,
        "message": "Max retries exceeded",
        "attempts": max_retries + 1
    }

def classify_error(error: Exception) -> QueryErrorType:
    """Classify error type for appropriate handling"""
    
    error_message = str(error).lower()
    
    if "connection" in error_message or "network" in error_message:
        return QueryErrorType.CONNECTION_ERROR
    
    elif "timeout" in error_message or "timed out" in error_message:
        return QueryErrorType.TIMEOUT_ERROR
    
    elif "rate limit" in error_message or "429" in error_message:
        return QueryErrorType.RATE_LIMIT_ERROR
    
    elif "vectorizer" in error_message or "embedding" in error_message:
        return QueryErrorType.VECTORIZER_ERROR
    
    elif "validation" in error_message or "invalid" in error_message:
        return QueryErrorType.VALIDATION_ERROR
    
    else:
        return QueryErrorType.UNKNOWN_ERROR

def safe_semantic_search(
    client: weaviate.Client,
    query: str,
    certainty: float = 0.7,
    limit: int = 10,
    fallback_strategy: str = "keyword"
) -> Dict[str, Any]:
    """
    Safe semantic search with fallback strategies
    
    Args:
        client: Weaviate client instance
        query: Search query
        certainty: Similarity threshold
        limit: Maximum results
        fallback_strategy: Strategy when vector search fails (keyword, cached, none)
        
    Returns:
        Search results with error handling information
    """
    
    def vector_search(client):
        return client.query.get("TelecomKnowledge", [
            "content", "title", "source", "networkFunctions", "confidence"
        ]).with_near_text({
            "concepts": [query],
            "certainty": certainty
        }).with_limit(limit).with_additional(["certainty"]).do()
    
    # Try vector search first
    result = robust_query_executor(client, vector_search)
    
    if not isinstance(result, dict) or not result.get("error"):
        return {
            "results": result["data"]["Get"]["TelecomKnowledge"],
            "search_type": "vector",
            "query": query,
            "success": True
        }
    
    # Vector search failed, try fallback strategies
    logger = logging.getLogger(__name__)
    logger.warning(f"Vector search failed: {result.get('message')}. Trying fallback strategy: {fallback_strategy}")
    
    if fallback_strategy == "keyword":
        def keyword_search(client):
            return client.query.get("TelecomKnowledge", [
                "content", "title", "source", "networkFunctions", "confidence"
            ]).with_bm25(
                query=query
            ).with_limit(limit).with_additional(["score"]).do()
        
        fallback_result = robust_query_executor(client, keyword_search)
        
        if not isinstance(fallback_result, dict) or not fallback_result.get("error"):
            return {
                "results": fallback_result["data"]["Get"]["TelecomKnowledge"],
                "search_type": "keyword_fallback",
                "query": query,
                "success": True,
                "original_error": result
            }
    
    elif fallback_strategy == "cached":
        # Try to get cached results
        cached_result = query_cache.get(query_cache._generate_key(query))
        if cached_result:
            return {
                "results": cached_result,
                "search_type": "cached_fallback", 
                "query": query,
                "success": True,
                "original_error": result
            }
    
    # All strategies failed
    return {
        "results": [],
        "search_type": "failed",
        "query": query,
        "success": False,
        "error": result,
        "fallback_attempted": fallback_strategy
    }

# Example usage with error handling
test_queries = [
    "AMF registration procedure",
    "invalid query with special chars @#$%",
    "network slicing configuration"
]

for query in test_queries:
    print(f"\nTesting query: '{query}'")
    result = safe_semantic_search(client, query, fallback_strategy="keyword")
    
    if result["success"]:
        print(f"✅ Success - Search type: {result['search_type']}")
        print(f"Results: {len(result['results'])}")
        if result['results']:
            print(f"Top result: {result['results'][0]['title']}")
    else:
        print(f"❌ Failed - Error: {result.get('error', {}).get('message', 'Unknown error')}")
```

## Best Practices

### Query Optimization Guidelines

```python
class QueryBestPractices:
    """Collection of best practices for Weaviate queries"""
    
    @staticmethod
    def optimize_vector_search(query: str, context: Dict[str, Any] = None) -> Dict[str, Any]:
        """
        Generate optimized vector search parameters
        
        Args:
            query: Search query
            context: Additional context (domain, use_case, etc.)
            
        Returns:
            Optimized search parameters
        """
        
        context = context or {}
        
        # Analyze query characteristics
        query_length = len(query.split())
        has_technical_terms = any(term.upper() in query.upper() 
                                for term in ["AMF", "SMF", "UPF", "gNB", "3GPP", "O-RAN"])
        has_procedural_language = any(word in query.lower() 
                                    for word in ["how", "configure", "deploy", "setup", "procedure"])
        
        # Base configuration
        config = {
            "certainty": 0.7,
            "limit": 10,
            "properties": ["content", "title", "networkFunctions"]
        }
        
        # Adjust based on query characteristics
        if query_length <= 3:
            # Short queries - increase certainty for precision
            config["certainty"] = 0.8
            config["properties"].append("procedures")
        
        elif query_length > 10:
            # Long queries - decrease certainty for recall
            config["certainty"] = 0.6
            config["limit"] = 15
        
        if has_technical_terms:
            # Technical queries - focus on technical properties
            config["properties"].extend(["interfaces", "specification"])
            config["certainty"] = 0.75
        
        if has_procedural_language:
            # Procedural queries - emphasize procedures and examples
            config["properties"].extend(["procedures", "category"])
            if context.get("domain") == "operations":
                config["certainty"] = 0.65  # More permissive for operational queries
        
        # Domain-specific adjustments
        if context.get("domain") == "5G":
            config["source_filter"] = ["3GPP"]
            config["properties"].append("networkFunctions")
        
        elif context.get("domain") == "O-RAN":
            config["source_filter"] = ["O-RAN"]
            config["properties"].append("interfaces")
        
        return config
    
    @staticmethod
    def build_effective_filters(requirements: Dict[str, Any]) -> Dict[str, Any]:
        """
        Build effective where filters for common requirements
        
        Args:
            requirements: Filter requirements
            
        Returns:
            Optimized where filter
        """
        
        operands = []
        
        # Source filtering
        if "sources" in requirements:
            sources = requirements["sources"]
            if len(sources) == 1:
                operands.append({
                    "path": ["source"],
                    "operator": "Equal",
                    "valueText": sources[0]
                })
            else:
                operands.append({
                    "path": ["source"],
                    "operator": "ContainsAny",
                    "valueTextArray": sources
                })
        
        # Network function filtering
        if "network_functions" in requirements:
            nfs = requirements["network_functions"]
            operands.append({
                "path": ["networkFunctions"],
                "operator": "ContainsAny",
                "valueTextArray": nfs
            })
        
        # Quality thresholds
        if "min_confidence" in requirements:
            operands.append({
                "path": ["confidence"],
                "operator": "GreaterThan",
                "valueNumber": requirements["min_confidence"]
            })
        
        if "min_priority" in requirements:
            operands.append({
                "path": ["priority"],
                "operator": "GreaterThan",
                "valueNumber": requirements["min_priority"]
            })
        
        # Date range filtering
        if "date_range" in requirements:
            date_range = requirements["date_range"]
            if "start" in date_range:
                operands.append({
                    "path": ["lastUpdated"],
                    "operator": "GreaterThan",
                    "valueDate": date_range["start"]
                })
            if "end" in date_range:
                operands.append({
                    "path": ["lastUpdated"],
                    "operator": "LessThan", 
                    "valueDate": date_range["end"]
                })
        
        # Use case filtering
        if "use_cases" in requirements:
            use_cases = requirements["use_cases"]
            operands.append({
                "path": ["useCase"],
                "operator": "ContainsAny",
                "valueTextArray": use_cases
            })
        
        # Build final filter
        if not operands:
            return {}
        
        if len(operands) == 1:
            return operands[0]
        
        return {
            "operator": "And",
            "operands": operands
        }
    
    @staticmethod
    def post_process_results(
        results: List[Dict[str, Any]], 
        query: str,
        processing_options: Dict[str, Any] = None
    ) -> List[Dict[str, Any]]:
        """
        Post-process query results for better relevance and presentation
        
        Args:
            results: Raw query results
            query: Original query
            processing_options: Post-processing options
            
        Returns:
            Enhanced and reordered results
        """
        
        processing_options = processing_options or {}
        
        # Add relevance scoring
        query_terms = set(query.lower().split())
        
        for result in results:
            relevance_factors = {
                "title_match": 0.0,
                "nf_match": 0.0, 
                "procedure_match": 0.0,
                "confidence_boost": 0.0,
                "priority_boost": 0.0
            }
            
            # Title matching
            title_terms = set(result.get("title", "").lower().split())
            title_overlap = len(query_terms.intersection(title_terms))
            relevance_factors["title_match"] = title_overlap / max(len(query_terms), 1)
            
            # Network function matching
            nf_terms = set(nf.lower() for nf in result.get("networkFunctions", []))
            nf_overlap = len(query_terms.intersection(nf_terms))
            relevance_factors["nf_match"] = nf_overlap / max(len(query_terms), 1)
            
            # Procedure matching
            procedure_terms = set(proc.lower() for proc in result.get("procedures", []))
            proc_overlap = len(query_terms.intersection(procedure_terms))
            relevance_factors["procedure_match"] = proc_overlap / max(len(query_terms), 1)
            
            # Quality boosts
            relevance_factors["confidence_boost"] = result.get("confidence", 0.5)
            relevance_factors["priority_boost"] = result.get("priority", 5) / 10.0
            
            # Calculate composite relevance score
            weights = {
                "title_match": 0.3,
                "nf_match": 0.25,
                "procedure_match": 0.2,
                "confidence_boost": 0.15,
                "priority_boost": 0.1
            }
            
            composite_score = sum(
                relevance_factors[factor] * weights[factor]
                for factor in weights
            )
            
            # Add to additional data
            if "_additional" not in result:
                result["_additional"] = {}
            result["_additional"]["relevance_score"] = composite_score
            result["_additional"]["relevance_factors"] = relevance_factors
        
        # Re-rank by composite relevance if requested
        if processing_options.get("rerank_by_relevance", True):
            results = sorted(results, 
                           key=lambda x: x.get("_additional", {}).get("relevance_score", 0),
                           reverse=True)
        
        # Add result explanations if requested
        if processing_options.get("add_explanations", False):
            for i, result in enumerate(results):
                explanation = generate_result_explanation(result, query, i + 1)
                result["_additional"]["explanation"] = explanation
        
        # Limit results if specified
        if "max_results" in processing_options:
            results = results[:processing_options["max_results"]]
        
        return results

def generate_result_explanation(result: Dict[str, Any], query: str, rank: int) -> str:
    """Generate human-readable explanation for why result is relevant"""
    
    factors = result.get("_additional", {}).get("relevance_factors", {})
    explanations = []
    
    if factors.get("title_match", 0) > 0.3:
        explanations.append("strong title match")
    
    if factors.get("nf_match", 0) > 0.2:
        nfs = result.get("networkFunctions", [])[:2]
        explanations.append(f"matches network functions: {', '.join(nfs)}")
    
    if factors.get("procedure_match", 0) > 0.2:
        procedures = result.get("procedures", [])[:2] 
        explanations.append(f"contains relevant procedures: {', '.join(procedures)}")
    
    if result.get("confidence", 0) > 0.8:
        explanations.append("high confidence content")
    
    if not explanations:
        explanations.append("semantic similarity to query")
    
    base_explanation = f"Ranked #{rank}: {'; '.join(explanations)}"
    
    # Add score information
    relevance_score = result.get("_additional", {}).get("relevance_score", 0)
    base_explanation += f" (relevance: {relevance_score:.2f})"
    
    return base_explanation

# Example usage of best practices
query = "how to configure AMF for network slicing"
context = {"domain": "5G", "use_case": "operations"}

# Get optimized parameters
config = QueryBestPractices.optimize_vector_search(query, context)
print(f"Optimized config: {config}")

# Build filters
requirements = {
    "sources": ["3GPP"],
    "network_functions": ["AMF", "NSSF"],
    "min_confidence": 0.8,
    "use_cases": ["eMBB", "URLLC"]
}

where_filter = QueryBestPractices.build_effective_filters(requirements)
print(f"Generated filter: {where_filter}")

# Execute optimized query
result = client.query.get("TelecomKnowledge", config["properties"]).with_near_text({
    "concepts": [query],
    "certainty": config["certainty"]
}).with_where(where_filter).with_limit(config["limit"]).with_additional(["certainty"]).do()

documents = result["data"]["Get"]["TelecomKnowledge"]

# Post-process results
enhanced_results = QueryBestPractices.post_process_results(
    documents, 
    query,
    {"rerank_by_relevance": True, "add_explanations": True, "max_results": 5}
)

print(f"\nEnhanced Results ({len(enhanced_results)}):")
for doc in enhanced_results:
    relevance = doc["_additional"]["relevance_score"]
    explanation = doc["_additional"]["explanation"]
    print(f"- {doc['title']} (relevance: {relevance:.3f})")
    print(f"  {explanation}")
    print()
```

This comprehensive guide provides practical examples and best practices for vector operations in the Weaviate deployment for the Nephoran Intent Operator, covering everything from basic operations to advanced optimization techniques specifically tailored for telecommunications domain operations.