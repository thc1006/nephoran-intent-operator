# RAG Configuration Templates and Examples

This document provides comprehensive configuration templates, examples, and best practices for deploying and configuring the Nephoran Intent Operator's RAG pipeline across different environments.

## Table of Contents

1. [Environment-Specific Configurations](#environment-specific-configurations)
2. [Component Configuration Templates](#component-configuration-templates)
3. [Advanced Configuration Examples](#advanced-configuration-examples)
4. [Security Configuration Templates](#security-configuration-templates)
5. [Performance Tuning Templates](#performance-tuning-templates)
6. [Monitoring and Observability](#monitoring-and-observability)
7. [Backup and Recovery Configuration](#backup-and-recovery-configuration)
8. [Multi-Environment Management](#multi-environment-management)

## Environment-Specific Configurations

### Development Environment

```yaml
# config/environments/development.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-config-development
  namespace: nephoran-system
data:
  environment: "development"
  
  # Weaviate Configuration
  weaviate.yaml: |
    weaviate:
      url: "http://localhost:8080"
      replicas: 1
      resources:
        requests:
          memory: "2Gi"
          cpu: "500m"
        limits:
          memory: "4Gi"
          cpu: "1000m"
      persistence:
        size: "20Gi"
        storageClass: "local-path"
      autoscaling:
        enabled: false
        
  # RAG Pipeline Configuration
  rag-pipeline.yaml: |
    rag:
      llm:
        provider: "openai"
        model: "gpt-3.5-turbo"
        temperature: 0.2
        max_tokens: 1024
        timeout: 15s
        
      embedding:
        provider: "openai"
        model: "text-embedding-ada-002"
        dimensions: 1536
        batch_size: 50
        
      processing:
        chunk_size: 800
        chunk_overlap: 150
        max_chunks_processed: 5
        confidence_threshold: 0.6
        
      cache:
        enabled: true
        ttl: 1800s  # 30 minutes
        max_size: 100
        
      metrics:
        enabled: false  # Disable detailed metrics in dev
        
  # Document Loader Configuration
  document-loader.yaml: |
    document_loader:
      batch_size: 10
      max_file_size: "50MB"
      supported_formats:
        - "pdf"
        - "txt"
        - "md"
      processing:
        ocr_enabled: false
        language_detection: false
        
  # Redis Cache Configuration (Optional in dev)
  redis.yaml: |
    redis:
      enabled: false  # Use in-memory cache for development
      
---
# Development secrets template
apiVersion: v1
kind: Secret
metadata:
  name: rag-secrets-development
  namespace: nephoran-system
type: Opaque
stringData:
  openai-api-key: "sk-dev-your-development-api-key-here"
  weaviate-api-key: "dev-weaviate-key-12345"
```

### Staging Environment

```yaml
# config/environments/staging.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-config-staging
  namespace: nephoran-system
data:
  environment: "staging"
  
  # Weaviate Configuration
  weaviate.yaml: |
    weaviate:
      url: "http://weaviate.nephoran-system.svc.cluster.local:8080"
      replicas: 2
      resources:
        requests:
          memory: "4Gi"
          cpu: "1000m"
        limits:
          memory: "8Gi"
          cpu: "2000m"
      persistence:
        size: "200Gi"
        storageClass: "gp3-encrypted"
      autoscaling:
        enabled: true
        minReplicas: 2
        maxReplicas: 4
        targetCPUUtilizationPercentage: 70
        
  # RAG Pipeline Configuration
  rag-pipeline.yaml: |
    rag:
      llm:
        provider: "openai"
        model: "gpt-4o-mini"
        temperature: 0.1
        max_tokens: 1536
        timeout: 25s
        rate_limiting:
          requests_per_minute: 1000
          
      embedding:
        provider: "openai"
        model: "text-embedding-3-large"
        dimensions: 3072
        batch_size: 75
        rate_limiting:
          requests_per_minute: 2000
          
      processing:
        chunk_size: 1000
        chunk_overlap: 200
        max_chunks_processed: 8
        confidence_threshold: 0.7
        parallel_processing: true
        worker_count: 3
        
      cache:
        enabled: true
        ttl: 3600s  # 1 hour
        max_size: 500
        
      metrics:
        enabled: true
        export_interval: 30s
        
  # Document Loader Configuration
  document-loader.yaml: |
    document_loader:
      batch_size: 25
      max_file_size: "100MB"
      parallel_processing: true
      worker_count: 3
      supported_formats:
        - "pdf"
        - "txt"
        - "md"
        - "docx"
      processing:
        ocr_enabled: true
        language_detection: true
        quality_validation: true
        
  # Redis Cache Configuration
  redis.yaml: |
    redis:
      enabled: true
      replicas: 1
      resources:
        requests:
          memory: "1Gi"
          cpu: "250m"
        limits:
          memory: "2Gi"
          cpu: "500m"
      persistence:
        enabled: true
        size: "10Gi"
        
---
# Staging secrets template
apiVersion: v1
kind: Secret
metadata:
  name: rag-secrets-staging
  namespace: nephoran-system
type: Opaque
stringData:
  openai-api-key: "sk-staging-your-staging-api-key-here"
  weaviate-api-key: "staging-weaviate-key-67890"
  redis-password: "staging-redis-password-abc123"
```

### Production Environment

```yaml
# config/environments/production.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-config-production
  namespace: nephoran-system
data:
  environment: "production"
  
  # Weaviate Configuration
  weaviate.yaml: |
    weaviate:
      url: "http://weaviate.nephoran-system.svc.cluster.local:8080"
      replicas: 3
      resources:
        requests:
          memory: "8Gi"
          cpu: "2000m"
        limits:
          memory: "32Gi"
          cpu: "8000m"
      persistence:
        size: "1Ti"
        storageClass: "gp3-encrypted"
        backup:
          enabled: true
          schedule: "0 2 * * *"
          retention: "30d"
      autoscaling:
        enabled: true
        minReplicas: 3
        maxReplicas: 10
        targetCPUUtilizationPercentage: 60
        targetMemoryUtilizationPercentage: 70
        customMetrics:
          - name: "weaviate_query_latency_p99"
            targetValue: "500m"
          - name: "weaviate_concurrent_queries"
            targetValue: "100"
      
      # High Availability Configuration
      podDisruptionBudget:
        enabled: true
        minAvailable: 2
      
      # Anti-affinity rules
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - weaviate
              topologyKey: kubernetes.io/hostname
        
  # RAG Pipeline Configuration
  rag-pipeline.yaml: |
    rag:
      llm:
        provider: "openai"
        model: "gpt-4o"
        temperature: 0.05
        max_tokens: 2048
        timeout: 30s
        rate_limiting:
          requests_per_minute: 5000
          burst_capacity: 100
        retry_policy:
          max_attempts: 3
          backoff: "exponential"
          
      embedding:
        provider: "openai"
        model: "text-embedding-3-large"
        dimensions: 3072
        batch_size: 100
        rate_limiting:
          requests_per_minute: 10000
        retry_policy:
          max_attempts: 3
          backoff: "exponential"
          
      processing:
        chunk_size: 1000
        chunk_overlap: 200
        max_chunks_processed: 10
        confidence_threshold: 0.75
        parallel_processing: true
        worker_count: 5
        queue_size: 1000
        
      cache:
        enabled: true
        ttl: 7200s  # 2 hours
        max_size: 2000
        eviction_policy: "lru"
        
      metrics:
        enabled: true
        export_interval: 15s
        detailed_metrics: true
        custom_metrics:
          - "rag_pipeline_accuracy"
          - "rag_pipeline_latency_p95"
          - "rag_pipeline_throughput"
        
  # Document Loader Configuration
  document-loader.yaml: |
    document_loader:
      batch_size: 50
      max_file_size: "500MB"
      parallel_processing: true
      worker_count: 8
      supported_formats:
        - "pdf"
        - "txt"
        - "md"
        - "docx"
        - "html"
        - "xml"
      processing:
        ocr_enabled: true
        language_detection: true
        quality_validation: true
        duplicate_detection: true
        metadata_extraction: true
      telecom_specific:
        3gpp_parser: true
        oran_parser: true
        etsi_parser: true
        
  # Redis Cache Configuration
  redis.yaml: |
    redis:
      enabled: true
      replicas: 3
      sentinel:
        enabled: true
        replicas: 3
      resources:
        requests:
          memory: "4Gi"
          cpu: "1000m"
        limits:
          memory: "8Gi"
          cpu: "2000m"
      persistence:
        enabled: true
        size: "100Gi"
        storageClass: "gp3-encrypted"
      security:
        auth_enabled: true
        tls_enabled: true
        
---
# Production secrets template (use external secret management)
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: rag-secrets-production
  namespace: nephoran-system
spec:
  refreshInterval: 15m
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: rag-secrets-production
    creationPolicy: Owner
  data:
  - secretKey: openai-api-key
    remoteRef:
      key: nephoran/rag/openai-api-key
  - secretKey: weaviate-api-key
    remoteRef:
      key: nephoran/rag/weaviate-api-key
  - secretKey: redis-password
    remoteRef:
      key: nephoran/rag/redis-password
```

## Component Configuration Templates

### RAG API Service Configuration

```yaml
# config/components/rag-api.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-api-config
  namespace: nephoran-system
data:
  server.yaml: |
    server:
      port: 8080
      host: "0.0.0.0"
      timeout:
        read: 30s
        write: 30s
        idle: 60s
      
      # Rate limiting
      rate_limiting:
        enabled: true
        requests_per_minute: 1000
        burst_capacity: 50
        
      # CORS configuration
      cors:
        enabled: true
        allowed_origins:
          - "https://nephoran-ui.com"
          - "https://staging.nephoran-ui.com"
        allowed_methods:
          - "GET"
          - "POST"
          - "OPTIONS"
        allowed_headers:
          - "Content-Type"
          - "Authorization"
          
      # Request/Response logging
      logging:
        enabled: true
        level: "info"
        format: "json"
        log_requests: true
        log_responses: false  # Avoid logging sensitive content
        
  # API endpoints configuration
  endpoints.yaml: |
    endpoints:
      # Document processing endpoints
      documents:
        upload: "/api/v1/documents/upload"
        process: "/api/v1/documents/process"
        status: "/api/v1/documents/{id}/status"
        delete: "/api/v1/documents/{id}"
        
      # Query processing endpoints
      queries:
        search: "/api/v1/query/search"
        intent: "/api/v1/query/intent"
        explain: "/api/v1/query/explain"
        
      # Management endpoints
      management:
        health: "/health"
        ready: "/ready"
        metrics: "/metrics"
        config: "/api/v1/config"
        
      # Batch processing endpoints
      batch:
        submit: "/api/v1/batch/submit"
        status: "/api/v1/batch/{id}/status"
        results: "/api/v1/batch/{id}/results"
```

### LLM Processor Configuration

```yaml
# config/components/llm-processor.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-processor-config
  namespace: nephoran-system
data:
  processor.yaml: |
    processor:
      # Model configuration
      models:
        primary:
          provider: "openai"
          model: "gpt-4o"
          temperature: 0.1
          max_tokens: 2048
          
        fallback:
          provider: "openai"
          model: "gpt-4o-mini"
          temperature: 0.1
          max_tokens: 1536
          
        embedding:
          provider: "openai"
          model: "text-embedding-3-large"
          dimensions: 3072
          
      # Processing configuration
      processing:
        max_concurrent_requests: 10
        timeout: 45s
        retry_attempts: 3
        
        # Context management
        context:
          max_length: 16000  # tokens
          truncation_strategy: "sliding_window"
          
        # Response validation
        validation:
          enabled: true
          min_confidence: 0.7
          max_response_length: 4000
          
      # Integration settings
      integration:
        weaviate:
          timeout: 30s
          max_results: 50
          
        controller:
          webhook_url: "http://nephoran-controller:8080/webhooks/llm"
          timeout: 15s
          
  # Prompt templates
  prompts.yaml: |
    prompts:
      # Intent processing prompts
      intent_analysis: |
        You are a telecommunications expert analyzing network operator intents.
        
        Context: {context}
        
        User Intent: {user_intent}
        
        Please analyze the intent and provide:
        1. Structured interpretation of the request
        2. Required network function configurations
        3. Relevant 3GPP/O-RAN specifications
        4. Potential risks or considerations
        
        Response format: JSON with fields: interpretation, configurations, specifications, risks
        
      configuration_generation: |
        Generate Kubernetes network function configuration based on:
        
        Intent: {intent}
        Context: {context}
        Network Function: {network_function}
        
        Requirements:
        - Follow 3GPP specifications
        - Include proper resource limits
        - Add monitoring and health checks
        - Ensure security best practices
        
        Provide valid YAML configuration.
        
      troubleshooting: |
        Analyze this network issue and provide troubleshooting guidance:
        
        Issue: {issue_description}
        Context: {context}
        Logs: {logs}
        
        Provide:
        1. Root cause analysis
        2. Step-by-step troubleshooting guide
        3. Preventive measures
        4. Relevant documentation references
```

### Enhanced Retrieval Service Configuration

```yaml
# config/components/enhanced-retrieval.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: enhanced-retrieval-config
  namespace: nephoran-system
data:
  retrieval.yaml: |
    retrieval:
      # Search configuration
      search:
        default_limit: 25
        max_limit: 1000
        timeout: 15s
        
        # Hybrid search settings
        hybrid:
          enabled: true
          alpha: 0.75  # Vector search weight
          
        # Similarity thresholds
        similarity:
          minimum: 0.7
          high_confidence: 0.9
          
      # Query enhancement
      enhancement:
        enabled: true
        
        # Query expansion
        expansion:
          enabled: true
          max_terms: 5
          telecom_synonyms: true
          acronym_expansion: true
          
        # Spell correction
        spelling:
          enabled: true
          confidence_threshold: 0.8
          
        # Query classification
        classification:
          enabled: true
          categories:
            - "configuration"
            - "troubleshooting"
            - "optimization"
            - "monitoring"
            
      # Result processing
      results:
        # Reranking
        reranking:
          enabled: true
          model: "cross-encoder/ms-marco-MiniLM-L-12-v2"
          top_k: 100
          
        # Deduplication
        deduplication:
          enabled: true
          similarity_threshold: 0.95
          
        # Context assembly
        context:
          max_length: 8000  # characters
          chunk_separator: "\n---\n"
          include_metadata: true
          
  # Telecom-specific configuration
  telecom.yaml: |
    telecom:
      # Domain-specific processing
      domains:
        ran:
          weight: 1.2
          keywords:
            - "gNB"
            - "eNB"
            - "RAN"
            - "Radio"
            - "Antenna"
        core:
          weight: 1.1
          keywords:
            - "AMF"
            - "SMF"
            - "UPF"
            - "5GC"
            - "EPC"
        transport:
          weight: 1.0
          keywords:
            - "Transport"
            - "Backhaul"
            - "Fronthaul"
            - "Network"
            
      # Specification priorities
      specifications:
        "3GPP":
          priority: 10
          releases:
            "Rel-17": 1.2
            "Rel-16": 1.1
            "Rel-15": 1.0
        "O-RAN":
          priority: 9
          versions:
            "v1.8": 1.2
            "v1.7": 1.1
        "ETSI":
          priority: 8
        "ITU":
          priority: 7
```

## Advanced Configuration Examples

### Multi-Model LLM Configuration

```yaml
# config/advanced/multi-model-llm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: multi-model-llm-config
  namespace: nephoran-system
data:
  models.yaml: |
    models:
      # Intent classification model
      intent_classifier:
        provider: "openai"
        model: "gpt-4o-mini"
        temperature: 0.0
        max_tokens: 100
        purpose: "intent_classification"
        
      # Configuration generation model
      config_generator:
        provider: "openai"
        model: "gpt-4o"
        temperature: 0.1
        max_tokens: 2048
        purpose: "configuration_generation"
        
      # Explanation model
      explainer:
        provider: "openai"
        model: "gpt-4o"
        temperature: 0.2
        max_tokens: 1536
        purpose: "explanation"
        
      # Code generation model (for automation scripts)
      code_generator:
        provider: "openai"
        model: "gpt-4o"
        temperature: 0.0
        max_tokens: 2048
        purpose: "code_generation"
        
    # Model routing configuration
    routing:
      rules:
        - condition: "intent_type == 'configuration'"
          model: "config_generator"
        - condition: "intent_type == 'explanation'"
          model: "explainer"
        - condition: "intent_type == 'automation'"
          model: "code_generator"
        - condition: "default"
          model: "intent_classifier"
          
    # Fallback strategy
    fallback:
      enabled: true
      strategy: "cascade"
      models:
        - "gpt-4o"
        - "gpt-4o-mini"
        - "gpt-3.5-turbo"
```

### Advanced Caching Configuration

```yaml
# config/advanced/advanced-caching.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: advanced-caching-config
  namespace: nephoran-system
data:
  caching.yaml: |
    caching:
      # Multi-tier caching
      tiers:
        l1:  # In-memory cache
          type: "memory"
          size: "500MB"
          ttl: "5m"
          eviction: "lru"
          
        l2:  # Redis cache
          type: "redis"
          size: "10GB"
          ttl: "1h"
          eviction: "allkeys-lru"
          
        l3:  # Persistent cache
          type: "filesystem"
          size: "100GB"
          ttl: "24h"
          path: "/var/cache/rag"
          
      # Cache strategies
      strategies:
        query_results:
          enabled: true
          tiers: ["l1", "l2"]
          ttl: "30m"
          key_pattern: "query:{hash}"
          
        embeddings:
          enabled: true
          tiers: ["l2", "l3"]
          ttl: "24h"
          key_pattern: "embedding:{hash}"
          
        documents:
          enabled: true
          tiers: ["l3"]
          ttl: "7d"
          key_pattern: "document:{id}"
          
      # Cache warming
      warming:
        enabled: true
        schedule: "0 1 * * *"  # Daily at 1 AM
        strategies:
          - "popular_queries"
          - "new_documents"
          
      # Cache analytics
      analytics:
        enabled: true
        metrics:
          - "hit_rate"
          - "miss_rate"
          - "eviction_rate"
          - "memory_usage"
```

### Performance Optimization Configuration

```yaml
# config/advanced/performance-optimization.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: performance-optimization-config
  namespace: nephoran-system
data:
  optimization.yaml: |
    optimization:
      # Connection pooling
      connection_pools:
        weaviate:
          max_connections: 50
          max_idle_connections: 10
          connection_timeout: "30s"
          idle_timeout: "5m"
          
        openai:
          max_connections: 100
          max_idle_connections: 20
          connection_timeout: "30s"
          idle_timeout: "2m"
          
        redis:
          max_connections: 25
          max_idle_connections: 5
          connection_timeout: "5s"
          idle_timeout: "10m"
          
      # Batch processing
      batching:
        embeddings:
          enabled: true
          batch_size: 100
          max_wait_time: "500ms"
          
        documents:
          enabled: true
          batch_size: 25
          max_wait_time: "2s"
          
      # Parallel processing
      parallelism:
        document_processing:
          workers: 8
          queue_size: 1000
          
        query_processing:
          workers: 4
          queue_size: 500
          
        embedding_generation:
          workers: 6
          queue_size: 2000
          
      # Resource limits
      resources:
        cpu:
          request: "2000m"
          limit: "8000m"
          
        memory:
          request: "4Gi"
          limit: "16Gi"
          
        disk:
          cache_size: "50Gi"
          temp_size: "10Gi"
```

## Security Configuration Templates

### Authentication and Authorization

```yaml
# config/security/auth.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-auth-config
  namespace: nephoran-system
data:
  authentication.yaml: |
    authentication:
      # API key authentication
      api_key:
        enabled: true
        header: "X-API-Key"
        validation:
          min_length: 32
          required_chars: ["upper", "lower", "digit", "special"]
          
      # JWT authentication
      jwt:
        enabled: true
        issuer: "https://auth.nephoran.com"
        audience: "rag-api"
        algorithm: "RS256"
        public_key_url: "https://auth.nephoran.com/.well-known/jwks.json"
        
      # mTLS authentication
      mtls:
        enabled: true
        ca_cert_path: "/etc/ssl/ca.crt"
        client_cert_path: "/etc/ssl/client.crt"
        client_key_path: "/etc/ssl/client.key"
        
  authorization.yaml: |
    authorization:
      # Role-based access control
      rbac:
        enabled: true
        roles:
          admin:
            permissions:
              - "documents:*"
              - "queries:*"
              - "config:*"
              - "metrics:*"
              
          operator:
            permissions:
              - "documents:read"
              - "documents:upload"
              - "queries:*"
              - "metrics:read"
              
          readonly:
            permissions:
              - "documents:read"
              - "queries:search"
              - "metrics:read"
              
      # Attribute-based access control
      abac:
        enabled: true
        policies:
          - name: "tenant_isolation"
            condition: "user.tenant == resource.tenant"
            effect: "allow"
            
          - name: "time_based_access"
            condition: "time.hour >= 8 && time.hour <= 18"
            effect: "allow"
            
      # API rate limiting per user
      rate_limiting:
        enabled: true
        per_user: true
        limits:
          admin: 10000  # requests per hour
          operator: 5000
          readonly: 1000
```

### Network Security Configuration

```yaml
# config/security/network.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: rag-network-policy
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app: rag-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Allow traffic from authorized namespaces
  - from:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 9090  # Metrics
  egress:
  # Allow DNS
  - to: []
    ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53
  # Allow HTTPS for external APIs
  - to: []
    ports:
    - protocol: TCP
      port: 443
  # Allow internal service communication
  - to:
    - podSelector:
        matchLabels:
          app: weaviate
    ports:
    - protocol: TCP
      port: 8080
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379

---
# TLS configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: tls-config
  namespace: nephoran-system
data:
  tls.yaml: |
    tls:
      # Server TLS configuration
      server:
        enabled: true
        cert_file: "/etc/tls/tls.crt"
        key_file: "/etc/tls/tls.key"
        min_version: "1.3"
        cipher_suites:
          - "TLS_AES_256_GCM_SHA384"
          - "TLS_AES_128_GCM_SHA256"
          - "TLS_CHACHA20_POLY1305_SHA256"
          
      # Client TLS configuration
      client:
        enabled: true
        ca_file: "/etc/tls/ca.crt"
        verify_server_cert: true
        server_name: "weaviate.nephoran-system.svc.cluster.local"
```

## Performance Tuning Templates

### Vector Database Optimization

```yaml
# config/performance/vector-optimization.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vector-optimization-config
  namespace: nephoran-system
data:
  vector_config.yaml: |
    vector_optimization:
      # HNSW index configuration
      hnsw:
        max_connections: 32      # Optimized for high-dimensional vectors
        ef_construction: 256     # Higher quality index construction
        ef: 128                  # Search accuracy vs speed balance
        max_m: 16               # Maximum bidirectional links
        ml: 1.0                 # Level generation factor
        
      # Quantization for memory efficiency
      quantization:
        enabled: true
        type: "pq"  # Product Quantization
        segments: 256
        centroids: 256
        
      # Index caching
      cache:
        vector_cache_max_objects: 2000000
        query_cache_max_objects: 1000000
        
      # Parallel processing
      parallel:
        indexing_workers: 8
        search_workers: 4
        
  memory_optimization.yaml: |
    memory:
      # JVM settings for Weaviate
      jvm:
        heap_size: "24g"
        gc_algorithm: "G1GC"
        gc_options:
          - "-XX:MaxPauseTimeMillis=200"
          - "-XX:G1HeapRegionSize=32m"
          - "-XX:G1ReservePercent=25"
          
      # Memory allocation
      allocation:
        vector_index: "16Gi"
        query_cache: "4Gi"
        metadata_cache: "2Gi"
        buffer_cache: "2Gi"
        
      # Memory monitoring
      monitoring:
        gc_logging: true
        heap_dumps: true
        threshold_alerts: "90%"
```

### Query Performance Configuration

```yaml
# config/performance/query-optimization.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: query-optimization-config
  namespace: nephoran-system
data:
  query_optimization.yaml: |
    query_optimization:
      # Query planning
      planning:
        cost_based_optimization: true
        statistics_enabled: true
        cardinality_estimation: true
        
      # Index hints
      indexing:
        force_vector_index: true
        use_inverted_index: true
        enable_filtering_optimization: true
        
      # Result caching
      result_caching:
        enabled: true
        cache_size: "1Gi"
        ttl: "30m"
        key_pattern: "query_result:{hash}"
        
      # Query rewriting
      rewriting:
        enabled: true
        rules:
          - name: "telecom_acronym_expansion"
            pattern: "AMF"
            replacement: "Access and Mobility Management Function OR AMF"
          - name: "3gpp_spec_normalization"
            pattern: "TS(\\d+\\.\\d+)"
            replacement: "3GPP TS $1"
            
      # Execution optimization
      execution:
        timeout: "30s"
        max_results: 1000
        early_termination: true
        result_streaming: true
```

## Monitoring and Observability

### Comprehensive Monitoring Configuration

```yaml
# config/monitoring/monitoring.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-monitoring-config
  namespace: nephoran-system
data:
  prometheus.yaml: |
    monitoring:
      prometheus:
        enabled: true
        scrape_interval: "15s"
        evaluation_interval: "15s"
        
        # Custom metrics
        custom_metrics:
          - name: "rag_pipeline_accuracy"
            type: "gauge"
            help: "RAG pipeline accuracy score"
            
          - name: "rag_query_latency"
            type: "histogram"
            help: "Query processing latency"
            buckets: [0.1, 0.5, 1.0, 2.0, 5.0, 10.0]
            
          - name: "rag_document_processing_rate"
            type: "counter"
            help: "Documents processed per second"
            
          - name: "rag_cache_hit_rate"
            type: "gauge"
            help: "Cache hit rate percentage"
            
        # Alerting rules
        rules:
          - name: "RAG High Latency"
            condition: "rag_query_latency_p95 > 5"
            severity: "warning"
            description: "RAG query latency is above 5 seconds"
            
          - name: "RAG Low Accuracy"
            condition: "rag_pipeline_accuracy < 0.8"
            severity: "critical"
            description: "RAG pipeline accuracy below 80%"
            
          - name: "RAG Cache Miss Rate High"
            condition: "rag_cache_hit_rate < 0.7"
            severity: "warning"
            description: "Cache hit rate below 70%"
            
  jaeger.yaml: |
    tracing:
      jaeger:
        enabled: true
        agent_host: "jaeger-agent"
        agent_port: 6831
        
        # Sampling configuration
        sampling:
          type: "probabilistic"
          param: 0.1  # Sample 10% of traces
          
        # Span configuration
        spans:
          max_tag_value_length: 1024
          max_logs_per_span: 128
          
        # Trace tags
        tags:
          - key: "component"
            value: "rag-pipeline"
          - key: "version"
            value: "${APP_VERSION}"
          - key: "environment"
            value: "${ENVIRONMENT}"
```

### Custom Dashboard Configuration

```yaml
# config/monitoring/dashboards.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-dashboards-config
  namespace: nephoran-system
data:
  grafana-dashboard.json: |
    {
      "dashboard": {
        "title": "Nephoran RAG Pipeline Dashboard",
        "tags": ["nephoran", "rag", "telecom"],
        "panels": [
          {
            "title": "Query Processing Latency",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, rate(rag_query_latency_bucket[5m]))",
                "legendFormat": "95th percentile"
              },
              {
                "expr": "histogram_quantile(0.50, rate(rag_query_latency_bucket[5m]))",
                "legendFormat": "50th percentile"
              }
            ]
          },
          {
            "title": "Pipeline Accuracy",
            "type": "singlestat",
            "targets": [
              {
                "expr": "avg(rag_pipeline_accuracy)",
                "legendFormat": "Accuracy"
              }
            ],
            "thresholds": "0.7,0.8"
          },
          {
            "title": "Document Processing Rate",
            "type": "graph",
            "targets": [
              {
                "expr": "rate(rag_document_processing_rate[5m])",
                "legendFormat": "Documents/sec"
              }
            ]
          },
          {
            "title": "Cache Performance",
            "type": "graph",
            "targets": [
              {
                "expr": "rag_cache_hit_rate",
                "legendFormat": "Hit Rate"
              },
              {
                "expr": "1 - rag_cache_hit_rate",
                "legendFormat": "Miss Rate"
              }
            ]
          }
        ]
      }
    }
```

## Backup and Recovery Configuration

### Automated Backup Configuration

```yaml
# config/backup/backup-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: backup-config
  namespace: nephoran-system
data:
  backup.yaml: |
    backup:
      # Backup schedules
      schedules:
        incremental:
          enabled: true
          schedule: "0 */4 * * *"  # Every 4 hours
          retention: "72h"
          
        full:
          enabled: true
          schedule: "0 2 * * 0"   # Weekly full backup
          retention: "4w"
          
      # Storage configuration
      storage:
        type: "s3"
        bucket: "nephoran-rag-backups"
        region: "us-east-1"
        encryption: true
        compression: true
        
      # Backup validation
      validation:
        enabled: true
        checksum_verification: true
        restore_test: true
        
      # Notification
      notifications:
        enabled: true
        channels:
          - type: "slack"
            webhook: "${SLACK_WEBHOOK_URL}"
          - type: "email"
            recipients: ["ops@nephoran.com"]

---
# Backup CronJob
apiVersion: batch/v1
kind: CronJob
metadata:
  name: rag-backup
  namespace: nephoran-system
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: nephoran/backup-tool:latest
            env:
            - name: BACKUP_TYPE
              value: "incremental"
            - name: WEAVIATE_URL
              value: "http://weaviate:8080"
            - name: S3_BUCKET
              valueFrom:
                configMapKeyRef:
                  name: backup-config
                  key: s3_bucket
            volumeMounts:
            - name: backup-config
              mountPath: /etc/backup
          volumes:
          - name: backup-config
            configMap:
              name: backup-config
          restartPolicy: OnFailure
```

## Multi-Environment Management

### Environment-Specific Kustomization

```yaml
# config/environments/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../base
- rag-config-production.yaml
- secrets.yaml

patchesStrategicMerge:
- production-patches.yaml

images:
- name: nephoran/rag-api
  newTag: v1.2.3
- name: nephoran/llm-processor
  newTag: v1.2.3

configMapGenerator:
- name: environment-config
  literals:
  - ENVIRONMENT=production
  - LOG_LEVEL=info
  - METRICS_ENABLED=true

secretGenerator:
- name: api-keys
  env: production.env
```

### Configuration Validation

```yaml
# config/validation/validation.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-validation
  namespace: nephoran-system
data:
  validate.sh: |
    #!/bin/bash
    set -e
    
    echo "Validating RAG configuration..."
    
    # Validate required environment variables
    required_vars=("OPENAI_API_KEY" "WEAVIATE_URL" "ENVIRONMENT")
    for var in "${required_vars[@]}"; do
      if [[ -z "${!var}" ]]; then
        echo "Error: Required environment variable $var is not set"
        exit 1
      fi
    done
    
    # Validate Weaviate connectivity
    echo "Testing Weaviate connectivity..."
    if ! curl -f "${WEAVIATE_URL}/v1/.well-known/ready" > /dev/null 2>&1; then
      echo "Error: Cannot connect to Weaviate at ${WEAVIATE_URL}"
      exit 1
    fi
    
    # Validate OpenAI API key
    echo "Testing OpenAI API key..."
    if ! curl -f -H "Authorization: Bearer ${OPENAI_API_KEY}" \
         "https://api.openai.com/v1/models" > /dev/null 2>&1; then
      echo "Error: Invalid OpenAI API key"
      exit 1
    fi
    
    # Validate configuration schema
    echo "Validating configuration schema..."
    python3 /scripts/validate_config.py
    
    echo "Configuration validation completed successfully!"

---
# Configuration validation job
apiVersion: batch/v1
kind: Job
metadata:
  name: config-validation
  namespace: nephoran-system
spec:
  template:
    spec:
      containers:
      - name: validator
        image: curlimages/curl:8.5.0
        command: ["/bin/sh", "/scripts/validate.sh"]
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: rag-secrets
              key: openai-api-key
        - name: WEAVIATE_URL
          valueFrom:
            configMapKeyRef:
              name: rag-config
              key: weaviate_url
        - name: ENVIRONMENT
          valueFrom:
            configMapKeyRef:
              name: rag-config
              key: environment
        volumeMounts:
        - name: validation-scripts
          mountPath: /scripts
      volumes:
      - name: validation-scripts
        configMap:
          name: config-validation
          defaultMode: 0755
      restartPolicy: Never
```

This comprehensive configuration template collection provides production-ready configurations for all aspects of the RAG pipeline deployment, from basic environment setup to advanced performance tuning and security hardening. Each template is designed to be customizable and can be adapted to specific deployment requirements and organizational policies.