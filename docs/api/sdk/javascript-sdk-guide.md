# Nephoran Intent Operator JavaScript/TypeScript SDK Guide

## Overview

The Nephoran Intent Operator JavaScript/TypeScript SDK provides a modern, type-safe client library for web applications, Node.js services, and edge functions. Built with enterprise requirements in mind, it offers:

- **Full TypeScript support** with comprehensive type definitions
- **Modern async/await** syntax with Promise-based APIs
- **OAuth2 authentication** with automatic token management
- **Built-in retry logic** and exponential backoff
- **Request/response interceptors** for custom middleware
- **Server-Sent Events** for real-time streaming
- **Node.js and browser compatibility**
- **Tree-shaking support** for optimal bundle sizes

## Installation

```bash
# npm
npm install @nephoran/intent-operator-sdk

# yarn
yarn add @nephoran/intent-operator-sdk

# pnpm
pnpm add @nephoran/intent-operator-sdk
```

### TypeScript Definitions

TypeScript definitions are included by default. For JavaScript projects, you can install type definitions separately:

```bash
npm install -D @types/nephoran-intent-operator-sdk
```

## Quick Start

### Basic Client Setup (TypeScript)

```typescript
import { NephoranClient, NetworkIntent, NetworkIntentSpec, ObjectMeta } from '@nephoran/intent-operator-sdk';

const client = new NephoranClient({
  baseURL: 'https://api.nephoran.com/v1',
  auth: {
    clientId: process.env.NEPHORAN_CLIENT_ID!,
    clientSecret: process.env.NEPHORAN_CLIENT_SECRET!,
    tokenUrl: 'https://auth.nephoran.com/oauth/token',
    scopes: ['intent.read', 'intent.write', 'intent.execute']
  },
  timeout: 30000
});

async function createNetworkIntent() {
  try {
    const intent: NetworkIntent = {
      apiVersion: 'nephoran.io/v1alpha1',
      kind: 'NetworkIntent',
      metadata: {
        name: 'amf-production',
        namespace: 'telecom-5g',
        labels: {
          environment: 'production',
          component: 'amf'
        }
      },
      spec: {
        intent: 'Deploy a high-availability AMF instance for production with auto-scaling',
        priority: 'high',
        parameters: {
          replicas: 3,
          enableAutoScaling: true,
          maxConcurrentConnections: 10000
        }
      }
    };

    const result = await client.intents.create(intent);
    console.log(`Intent created: ${result.metadata.name} (UID: ${result.metadata.uid})`);

    // Monitor status
    const status = await client.intents.getStatus(
      result.metadata.namespace!, 
      result.metadata.name!
    );
    console.log(`Status: ${status.phase} - ${status.message}`);

  } catch (error) {
    console.error('Error creating intent:', error);
  }
}

createNetworkIntent();
```

### Basic Client Setup (JavaScript/ES6)

```javascript
const { NephoranClient } = require('@nephoran/intent-operator-sdk');

const client = new NephoranClient({
  baseURL: 'https://api.nephoran.com/v1',
  auth: {
    clientId: process.env.NEPHORAN_CLIENT_ID,
    clientSecret: process.env.NEPHORAN_CLIENT_SECRET,
    tokenUrl: 'https://auth.nephoran.com/oauth/token',
    scopes: ['intent.read', 'intent.write', 'rag.query']
  }
});

async function listIntents() {
  try {
    const response = await client.intents.list({
      namespace: 'telecom-5g',
      limit: 20,
      labels: 'environment=production'
    });

    console.log(`Found ${response.items.length} intents`);
    
    response.items.forEach(intent => {
      console.log(`- ${intent.metadata.name}: ${intent.status?.phase}`);
    });

  } catch (error) {
    console.error('Error listing intents:', error);
  }
}

listIntents();
```

## Advanced Configuration

### Production Client Configuration

```typescript
import { 
  NephoranClient, 
  ClientConfig, 
  RetryConfig, 
  CircuitBreakerConfig 
} from '@nephoran/intent-operator-sdk';
import axios, { AxiosRequestConfig } from 'axios';

const productionConfig: ClientConfig = {
  baseURL: 'https://api.nephoran.com/v1',
  auth: {
    clientId: process.env.NEPHORAN_CLIENT_ID!,
    clientSecret: process.env.NEPHORAN_CLIENT_SECRET!,
    tokenUrl: 'https://auth.nephoran.com/oauth/token',
    scopes: ['intent.read', 'intent.write', 'intent.execute', 'rag.query'],
    tokenCacheTtl: 3600, // Cache tokens for 1 hour
    autoRefresh: true
  },
  timeout: 60000,
  retry: {
    maxAttempts: 3,
    backoffFactor: 2,
    maxBackoffDelay: 60000,
    retryableStatusCodes: [429, 500, 502, 503, 504],
    retryableErrors: ['ECONNRESET', 'ETIMEDOUT', 'ENOTFOUND']
  },
  circuitBreaker: {
    failureThreshold: 5,
    timeout: 30000,
    resetTimeout: 60000
  },
  // Custom axios instance for advanced HTTP configuration
  httpClient: axios.create({
    timeout: 60000,
    maxContentLength: 10 * 1024 * 1024, // 10MB
    maxBodyLength: 10 * 1024 * 1024,
    headers: {
      'User-Agent': 'MyApp/1.0.0 (Nephoran SDK/1.0.0)',
      'Accept-Encoding': 'gzip, deflate, br'
    }
  }),
  // Request/response interceptors
  interceptors: {
    request: [(config: AxiosRequestConfig) => {
      console.log(`Making request: ${config.method?.toUpperCase()} ${config.url}`);
      config.headers = {
        ...config.headers,
        'X-Request-ID': generateRequestId(),
        'X-Request-Timestamp': new Date().toISOString()
      };
      return config;
    }],
    response: [
      (response) => {
        console.log(`Response: ${response.status} ${response.statusText}`);
        return response;
      },
      (error) => {
        console.error('Response error:', error.response?.status, error.message);
        return Promise.reject(error);
      }
    ]
  }
};

const client = new NephoranClient(productionConfig);

function generateRequestId(): string {
  return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}
```

### Browser Configuration

```typescript
import { NephoranClient } from '@nephoran/intent-operator-sdk/browser';

// Browser-optimized configuration
const browserClient = new NephoranClient({
  baseURL: 'https://api.nephoran.com/v1',
  auth: {
    // For browser apps, use OAuth2 authorization code flow
    clientId: 'your-public-client-id',
    authUrl: 'https://auth.nephoran.com/oauth/authorize',
    tokenUrl: 'https://auth.nephoran.com/oauth/token',
    redirectUri: window.location.origin + '/auth/callback',
    scopes: ['intent.read', 'rag.query'],
    usePKCE: true // Use PKCE for security
  },
  timeout: 30000,
  enableCORS: true,
  // Browser-specific options
  browserOptions: {
    enableCaching: true,
    cachePrefix: 'nephoran_',
    enableOfflineQueue: true,
    maxOfflineRequests: 50
  }
});

// Handle authentication flow
async function authenticate() {
  try {
    await browserClient.auth.login();
    console.log('Authentication successful');
  } catch (error) {
    console.error('Authentication failed:', error);
  }
}
```

## Core API Operations

### Intent Management

#### Creating Various Network Intents

```typescript
import { 
  NetworkIntent, 
  NetworkSlice, 
  RANIntent,
  NetworkIntentSpec,
  ObjectMeta 
} from '@nephoran/intent-operator-sdk';

class IntentManager {
  constructor(private client: NephoranClient) {}

  async create5GCoreIntent(): Promise<NetworkIntent> {
    const intent: NetworkIntent = {
      apiVersion: 'nephoran.io/v1alpha1',
      kind: 'NetworkIntent',
      metadata: {
        name: 'production-amf-cluster',
        namespace: 'telecom-5g-core',
        labels: {
          environment: 'production',
          'nf-type': 'amf',
          '3gpp-release': 'rel16',
          'deployment-tier': 'critical'
        },
        annotations: {
          'nephoran.io/sla-target': '99.99%',
          'nephoran.io/max-downtime': '52.56-minutes-per-year'
        }
      },
      spec: {
        intent: 'Deploy high-availability AMF cluster with N1/N2 interface support for 500,000 concurrent UE registrations with sub-second response time',
        priority: 'critical',
        parameters: {
          replicas: 7,
          enableAutoScaling: true,
          minReplicas: 5,
          maxReplicas: 15,
          cpuTargetUtilization: 70,
          maxConcurrentUEs: 500000,
          interfaces: {
            n1: { enabled: true, port: 38412 },
            n2: { enabled: true, port: 38422 }
          },
          sessionPersistence: {
            enabled: true,
            backend: 'redis-cluster',
            replication: 3
          },
          loadBalancing: {
            algorithm: 'consistent-hash',
            healthCheck: {
              interval: '5s',
              timeout: '2s',
              retries: 3
            }
          },
          security: {
            mtls: true,
            oauth2: true,
            rateLimiting: {
              requestsPerSecond: 10000,
              burstSize: 20000
            }
          }
        },
        targetClusters: ['cluster-east-1', 'cluster-west-1', 'cluster-central'],
        oranCompliance: true
      }
    };

    return this.client.intents.create(intent);
  }

  async createNetworkSlice(): Promise<NetworkSlice> {
    const slice: NetworkSlice = {
      apiVersion: 'nephoran.io/v1alpha1',
      kind: 'NetworkSlice',
      metadata: {
        name: 'automotive-urllc-slice',
        namespace: 'network-slicing',
        labels: {
          'slice-type': 'URLLC',
          'use-case': 'automotive',
          'sla-tier': 'premium'
        }
      },
      spec: {
        intent: 'Create URLLC network slice for autonomous vehicle communication with 1ms latency, 99.9999% reliability, and dedicated resources',
        priority: 'critical',
        parameters: {
          sliceType: 'URLLC',
          useCase: 'automotive-v2x',
          sla: {
            latency: '1ms',
            reliability: '99.9999%',
            availability: '99.999%',
            throughput: '10Gbps'
          },
          resourceAllocation: {
            dedicatedSpectrum: true,
            cpuReservation: '50%',
            memoryReservation: '40%',
            networkBandwidth: 'dedicated'
          },
          qosProfile: {
            '5qi': 82, // V2X specific QoS identifier
            arp: {
              priorityLevel: 1,
              preemptionCapability: 'may-preempt',
              preemptionVulnerability: 'not-preemptable'
            }
          },
          security: {
            isolation: 'hard',
            encryption: 'end-to-end',
            authentication: 'certificate-based'
          }
        },
        networkSlice: 's-nssai-01-000001'
      }
    };

    return this.client.intents.create(slice);
  }

  async createORANIntent(): Promise<RANIntent> {
    const ranIntent: RANIntent = {
      apiVersion: 'nephoran.io/v1alpha1',
      kind: 'RANIntent',
      metadata: {
        name: 'production-near-rt-ric',
        namespace: 'o-ran-ric',
        labels: {
          component: 'near-rt-ric',
          'oran-release': 'r5',
          deployment: 'production'
        }
      },
      spec: {
        intent: 'Deploy production Near-RT RIC with comprehensive xApp ecosystem for intelligent RAN optimization, traffic steering, and QoS management',
        priority: 'high',
        parameters: {
          platform: {
            version: '1.3.6',
            enableHA: true,
            persistence: true
          },
          xApps: [
            {
              name: 'traffic-steering',
              version: '1.2.0',
              config: {
                algorithm: 'load-based',
                updateInterval: '100ms'
              }
            },
            {
              name: 'qos-optimizer',
              version: '1.1.5',
              config: {
                optimizationTarget: 'latency',
                mlModel: 'reinforcement-learning'
              }
            },
            {
              name: 'anomaly-detector',
              version: '1.0.8',
              config: {
                detectionMethod: 'statistical + ml',
                alertThreshold: 0.95
              }
            }
          ],
          e2Interfaces: [
            {
              version: 'E2-KPM',
              enabled: true,
              reportingPeriod: '1s'
            },
            {
              version: 'E2-RC',
              enabled: true,
              controlActions: ['handover', 'admission', 'qos']
            }
          ],
          a1Policies: [
            {
              type: 'traffic_steering_policy',
              version: '1.0',
              enforcement: 'mandatory'
            }
          ]
        },
        oranCompliance: true
      }
    };

    return this.client.intents.create(ranIntent);
  }

  async batchCreateIntents(intents: NetworkIntent[]): Promise<NetworkIntent[]> {
    const promises = intents.map(intent => 
      this.client.intents.create(intent).catch(error => ({
        error,
        intent
      }))
    );

    const results = await Promise.allSettled(promises);
    const successful: NetworkIntent[] = [];
    const failed: { intent: NetworkIntent; error: any }[] = [];

    results.forEach((result, index) => {
      if (result.status === 'fulfilled' && !('error' in result.value)) {
        successful.push(result.value);
      } else {
        const value = result.status === 'fulfilled' ? result.value : { error: result.reason, intent: intents[index] };
        failed.push(value);
        console.error(`Failed to create intent ${value.intent.metadata.name}:`, value.error);
      }
    });

    console.log(`Batch operation completed: ${successful.length} succeeded, ${failed.length} failed`);
    return successful;
  }
}
```

#### Real-time Intent Monitoring

```typescript
import { EventEmitter } from 'events';

interface IntentEvent {
  event: string;
  timestamp: string;
  data: Record<string, any>;
}

class IntentMonitor extends EventEmitter {
  private eventSources: Map<string, EventSource> = new Map();

  constructor(private client: NephoranClient) {
    super();
  }

  async watchIntent(namespace: string, name: string): Promise<void> {
    const key = `${namespace}/${name}`;
    
    if (this.eventSources.has(key)) {
      console.warn(`Already watching intent ${key}`);
      return;
    }

    try {
      // Get the streaming endpoint URL
      const streamUrl = await this.client.intents.getEventStreamUrl(namespace, name);
      
      const eventSource = new EventSource(streamUrl, {
        headers: {
          Authorization: `Bearer ${await this.client.auth.getAccessToken()}`
        }
      });

      this.eventSources.set(key, eventSource);

      eventSource.onopen = () => {
        console.log(`Started watching intent ${key}`);
        this.emit('connected', { namespace, name });
      };

      eventSource.onmessage = (event) => {
        try {
          const intentEvent: IntentEvent = JSON.parse(event.data);
          this.handleIntentEvent(namespace, name, intentEvent);
        } catch (error) {
          console.error('Error parsing event:', error);
        }
      };

      eventSource.onerror = (error) => {
        console.error(`EventSource error for ${key}:`, error);
        this.emit('error', { namespace, name, error });
        
        // Attempt to reconnect after a delay
        setTimeout(() => {
          if (this.eventSources.has(key)) {
            this.reconnectEventSource(namespace, name);
          }
        }, 5000);
      };

    } catch (error) {
      console.error(`Failed to start watching intent ${key}:`, error);
      throw error;
    }
  }

  private handleIntentEvent(namespace: string, name: string, event: IntentEvent): void {
    const intentKey = `${namespace}/${name}`;
    
    switch (event.event) {
      case 'processing':
        const stage = event.data.stage || 'unknown';
        const progress = event.data.progress || 0;
        console.log(`[${event.timestamp}] ${intentKey}: Processing ${stage} (${progress}%)`);
        this.emit('processing', { namespace, name, stage, progress });
        break;

      case 'context_retrieved':
        const docs = event.data.documents || 0;
        const score = event.data.relevance_score || 0;
        console.log(`[${event.timestamp}] ${intentKey}: Context retrieved (${docs} docs, score: ${score.toFixed(2)})`);
        this.emit('context_retrieved', { namespace, name, documents: docs, score });
        break;

      case 'deployment_started':
        const resources = event.data.resources || 0;
        const estimatedTime = event.data.estimated_time || 'unknown';
        console.log(`[${event.timestamp}] ${intentKey}: Deployment started (${resources} resources, ETA: ${estimatedTime})`);
        this.emit('deployment_started', { namespace, name, resources, estimatedTime });
        break;

      case 'deployment_complete':
        const successful = event.data.successful_resources || 0;
        const total = event.data.total_resources || 0;
        console.log(`[${event.timestamp}] ${intentKey}: Deployment complete (${successful}/${total} resources)`);
        this.emit('deployment_complete', { namespace, name, successful, total });
        break;

      case 'error':
        const errorMessage = event.data.message || 'Unknown error';
        console.error(`[${event.timestamp}] ${intentKey}: ERROR - ${errorMessage}`);
        this.emit('intent_error', { namespace, name, message: errorMessage });
        break;

      default:
        console.log(`[${event.timestamp}] ${intentKey}: ${event.event}`, event.data);
        this.emit('unknown_event', { namespace, name, event, data: event.data });
    }
  }

  private async reconnectEventSource(namespace: string, name: string): Promise<void> {
    const key = `${namespace}/${name}`;
    
    // Close existing connection
    const existingSource = this.eventSources.get(key);
    if (existingSource) {
      existingSource.close();
      this.eventSources.delete(key);
    }

    // Attempt to reconnect
    try {
      await this.watchIntent(namespace, name);
      console.log(`Reconnected to intent ${key}`);
    } catch (error) {
      console.error(`Failed to reconnect to intent ${key}:`, error);
    }
  }

  stopWatching(namespace: string, name: string): void {
    const key = `${namespace}/${name}`;
    const eventSource = this.eventSources.get(key);
    
    if (eventSource) {
      eventSource.close();
      this.eventSources.delete(key);
      console.log(`Stopped watching intent ${key}`);
      this.emit('disconnected', { namespace, name });
    }
  }

  stopAll(): void {
    this.eventSources.forEach((eventSource, key) => {
      eventSource.close();
      console.log(`Stopped watching intent ${key}`);
    });
    
    this.eventSources.clear();
    this.emit('all_disconnected');
  }
}

// Usage example
async function monitorIntentLifecycle() {
  const client = new NephoranClient(/* config */);
  const monitor = new IntentMonitor(client);

  // Set up event listeners
  monitor.on('processing', (event) => {
    console.log(`Progress update: ${event.stage} - ${event.progress}%`);
  });

  monitor.on('deployment_complete', (event) => {
    console.log(`Deployment successful: ${event.successful}/${event.total} resources`);
    monitor.stopWatching(event.namespace, event.name);
  });

  monitor.on('intent_error', (event) => {
    console.error(`Intent failed: ${event.message}`);
    monitor.stopWatching(event.namespace, event.name);
  });

  // Start monitoring
  await monitor.watchIntent('telecom-5g', 'my-intent');
}
```

### LLM Processing

```typescript
import { 
  LLMProcessingRequest, 
  LLMProcessingResponse, 
  LLMParameters,
  StreamingLLMResponse 
} from '@nephoran/intent-operator-sdk';

class LLMProcessor {
  constructor(private client: NephoranClient) {}

  async processTelecomIntent(
    intent: string, 
    context?: string[], 
    options?: Partial<LLMParameters>
  ): Promise<LLMProcessingResponse> {
    const request: LLMProcessingRequest = {
      intent,
      context: context || [
        '5G Core network functions require high availability',
        'Auto-scaling based on connection count and CPU utilization',
        'Production deployments need comprehensive monitoring'
      ],
      parameters: {
        model: 'gpt-4o-mini',
        temperature: 0.3,
        maxTokens: 2000,
        ...options
      }
    };

    const response = await this.client.llm.process(request);
    
    console.log(`LLM Processing Results:`);
    console.log(`- Confidence: ${response.confidence.toFixed(2)}`);
    console.log(`- Network Functions: ${response.networkFunctions.join(', ')}`);
    console.log(`- Deployment Strategy: ${response.deploymentStrategy}`);
    console.log(`- Processing Time: ${response.processingMetrics.processingTimeMs}ms`);
    console.log(`- Token Usage: ${response.processingMetrics.tokenUsage}`);

    return response;
  }

  async streamLLMProcessing(
    intent: string,
    onChunk: (chunk: string) => void,
    onComplete: (fullResponse: string) => void,
    onError: (error: Error) => void
  ): Promise<void> {
    const request: LLMProcessingRequest = {
      intent,
      streaming: true,
      parameters: {
        model: 'gpt-4o-mini',
        temperature: 0.2,
        maxTokens: 3000
      }
    };

    try {
      const stream = await this.client.llm.processStream(request);
      let fullResponse = '';

      for await (const chunk of stream) {
        if (chunk.error) {
          onError(new Error(chunk.error));
          return;
        }

        onChunk(chunk.content);
        fullResponse += chunk.content;
      }

      onComplete(fullResponse);
    } catch (error) {
      onError(error as Error);
    }
  }

  async processMultipleIntents(intents: string[]): Promise<LLMProcessingResponse[]> {
    const requests = intents.map(intent => ({
      intent,
      parameters: {
        model: 'gpt-4o-mini',
        temperature: 0.3,
        maxTokens: 1500
      }
    }));

    // Process with controlled concurrency
    const semaphore = new Semaphore(3); // Max 3 concurrent requests
    const results: LLMProcessingResponse[] = [];

    await Promise.all(requests.map(async (request, index) => {
      await semaphore.acquire();
      
      try {
        const response = await this.client.llm.process(request);
        results[index] = response;
        console.log(`Processed intent ${index + 1}/${requests.length}`);
      } catch (error) {
        console.error(`Failed to process intent ${index + 1}:`, error);
        throw error;
      } finally {
        semaphore.release();
      }
    }));

    return results;
  }
}

// Semaphore implementation for concurrency control
class Semaphore {
  private permits: number;
  private waitQueue: Array<() => void> = [];

  constructor(permits: number) {
    this.permits = permits;
  }

  async acquire(): Promise<void> {
    return new Promise<void>((resolve) => {
      if (this.permits > 0) {
        this.permits--;
        resolve();
      } else {
        this.waitQueue.push(resolve);
      }
    });
  }

  release(): void {
    this.permits++;
    const next = this.waitQueue.shift();
    if (next) {
      this.permits--;
      next();
    }
  }
}

// Usage examples
async function demonstrateLLMProcessing() {
  const client = new NephoranClient(/* config */);
  const processor = new LLMProcessor(client);

  // Basic processing
  const response = await processor.processTelecomIntent(
    'Deploy a resilient 5G core network with AMF, SMF, and UPF functions supporting 1 million subscribers'
  );

  // Streaming processing
  await processor.streamLLMProcessing(
    'Design a comprehensive network slicing architecture for smart city deployment',
    (chunk) => process.stdout.write(chunk),
    (fullResponse) => console.log('\n\nProcessing complete!'),
    (error) => console.error('Streaming error:', error)
  );

  // Batch processing
  const intents = [
    'Deploy AMF with high availability',
    'Create URLLC network slice for industrial IoT',
    'Configure O-RAN Near-RT RIC with traffic steering'
  ];

  const results = await processor.processMultipleIntents(intents);
  console.log(`Processed ${results.length} intents`);
}
```

### RAG Knowledge Retrieval

```typescript
import { 
  RAGQuery, 
  RAGResponse, 
  RAGResult,
  RAGContext,
  KnowledgeBaseDocument
} from '@nephoran/intent-operator-sdk';

class KnowledgeRetriever {
  private cache = new Map<string, { data: RAGResponse; timestamp: number }>();
  private readonly CACHE_TTL = 300000; // 5 minutes

  constructor(private client: NephoranClient) {}

  async queryTelecomKnowledge(
    query: string,
    domain: string,
    options: {
      maxResults?: number;
      threshold?: number;
      useCache?: boolean;
      includeMetadata?: boolean;
    } = {}
  ): Promise<RAGResponse> {
    const {
      maxResults = 5,
      threshold = 0.8,
      useCache = true,
      includeMetadata = true
    } = options;

    const cacheKey = `${query}:${domain}:${maxResults}:${threshold}`;

    // Check cache
    if (useCache && this.cache.has(cacheKey)) {
      const cached = this.cache.get(cacheKey)!;
      if (Date.now() - cached.timestamp < this.CACHE_TTL) {
        console.log('Using cached RAG result');
        return cached.data;
      }
    }

    const ragQuery: RAGQuery = {
      query,
      context: {
        domain,
        technology: this.inferTechnology(query)
      },
      maxResults,
      threshold,
      includeMetadata
    };

    const response = await this.client.rag.query(ragQuery);

    // Cache the result
    if (useCache) {
      this.cache.set(cacheKey, {
        data: response,
        timestamp: Date.now()
      });
    }

    console.log(`RAG Query Results:`);
    console.log(`- Query: "${query}"`);
    console.log(`- Domain: ${domain}`);
    console.log(`- Results: ${response.results.length}`);
    console.log(`- Processing Time: ${response.metadata.processingTimeMs}ms`);

    return response;
  }

  async queryORANSpecifications(): Promise<Record<string, RAGResponse>> {
    const oranQueries = [
      'Near-RT RIC architecture and xApp development framework',
      'E2 interface protocol specifications and service models',
      'A1 policy management and RIC control procedures',
      'O-RAN Security architecture and threat model',
      'Open Fronthaul interface and functional split options'
    ];

    const results: Record<string, RAGResponse> = {};

    // Execute queries in parallel
    await Promise.all(oranQueries.map(async (query) => {
      try {
        const response = await this.queryTelecomKnowledge(query, 'o-ran', {
          maxResults: 3,
          threshold: 0.85
        });
        results[query] = response;
        
        console.log(`\n=== ${query} ===`);
        response.results.forEach((result, index) => {
          console.log(`${index + 1}. ${result.source} (Score: ${result.score.toFixed(3)})`);
          console.log(`   ${result.text.substring(0, 150)}...`);
        });
      } catch (error) {
        console.error(`Failed to query: ${query}`, error);
      }
    }));

    return results;
  }

  async query5GCoreArchitecture(): Promise<RAGResult[]> {
    const coreQueries = [
      '5G core network service-based architecture components and interfaces',
      'AMF access and mobility management procedures and N1/N2 interfaces',
      'SMF session management and PDU session establishment procedures',
      'UPF user plane function and data packet processing architecture',
      'Network slicing implementation in 5G core with NSSF and slice selection'
    ];

    const allResults: RAGResult[] = [];

    for (const query of coreQueries) {
      const response = await this.queryTelecomKnowledge(query, '5g-core', {
        maxResults: 3,
        threshold: 0.8
      });
      allResults.push(...response.results);
    }

    // Deduplicate and sort by relevance
    const uniqueResults = new Map<string, RAGResult>();
    allResults.forEach(result => {
      const existing = uniqueResults.get(result.source);
      if (!existing || result.score > existing.score) {
        uniqueResults.set(result.source, result);
      }
    });

    const sortedResults = Array.from(uniqueResults.values())
      .sort((a, b) => b.score - a.score)
      .slice(0, 10); // Top 10 results

    console.log(`\n5G Core Architecture Knowledge Summary:`);
    console.log(`Total unique sources: ${sortedResults.length}`);
    
    sortedResults.forEach((result, index) => {
      console.log(`\n${index + 1}. ${result.source} (Score: ${result.score.toFixed(3)})`);
      console.log(`   ${result.text.substring(0, 200)}...`);
      if (result.metadata) {
        console.log(`   Type: ${result.metadata.documentType}, Section: ${result.metadata.section}`);
      }
    });

    return sortedResults;
  }

  async getKnowledgeBaseInfo(domain?: string): Promise<KnowledgeBaseDocument[]> {
    const params: any = { limit: 50 };
    if (domain) params.domain = domain;

    const response = await this.client.rag.listDocuments(params);
    
    console.log(`\nKnowledge Base Documents${domain ? ` (${domain})` : ''}:`);
    response.documents.forEach(doc => {
      console.log(`- ${doc.title} (${doc.documentType})`);
      console.log(`  Chunks: ${doc.chunkCount}, Updated: ${doc.lastUpdated}`);
    });

    return response.documents;
  }

  async enhancedQuery(
    primaryQuery: string,
    domain: string,
    contextQueries: string[] = []
  ): Promise<RAGResponse> {
    // Build enhanced context by querying related topics
    const contextResults: RAGResult[] = [];

    if (contextQueries.length > 0) {
      for (const contextQuery of contextQueries) {
        const contextResponse = await this.queryTelecomKnowledge(
          contextQuery,
          domain,
          { maxResults: 2, threshold: 0.7 }
        );
        contextResults.push(...contextResponse.results);
      }
    }

    // Use context to enhance the primary query
    const enhancedQuery = contextResults.length > 0
      ? `${primaryQuery} ${contextResults.map(r => r.text.split('.')[0]).join(' ')}`
      : primaryQuery;

    return this.queryTelecomKnowledge(enhancedQuery, domain, {
      maxResults: 7,
      threshold: 0.75,
      useCache: false
    });
  }

  private inferTechnology(query: string): string {
    const technologyKeywords: Record<string, string[]> = {
      'near-rt-ric': ['near-rt', 'ric', 'xapp', 'e2'],
      '5g-core': ['amf', 'smf', 'upf', 'nssf', 'sba'],
      'network-slicing': ['slice', 'nssai', 'sst', 'sd'],
      'mec': ['edge', 'mec', 'multi-access'],
      'ran': ['ran', 'gnb', 'enb', 'cell']
    };

    const queryLower = query.toLowerCase();

    for (const [tech, keywords] of Object.entries(technologyKeywords)) {
      if (keywords.some(keyword => queryLower.includes(keyword))) {
        return tech;
      }
    }

    return 'general';
  }

  clearCache(): void {
    this.cache.clear();
    console.log('RAG cache cleared');
  }
}
```

## Error Handling and Resilience

```typescript
import { 
  NephoranAPIError, 
  AuthenticationError, 
  ValidationError,
  RateLimitError,
  ResourceNotFoundError,
  CircuitBreakerError
} from '@nephoran/intent-operator-sdk';

class ResilientOperations {
  constructor(private client: NephoranClient) {}

  async executeWithRetry<T>(
    operation: () => Promise<T>,
    options: {
      maxAttempts?: number;
      baseDelay?: number;
      maxDelay?: number;
      backoffFactor?: number;
      jitter?: boolean;
    } = {}
  ): Promise<T> {
    const {
      maxAttempts = 3,
      baseDelay = 1000,
      maxDelay = 60000,
      backoffFactor = 2,
      jitter = true
    } = options;

    let lastError: Error;

    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      try {
        return await operation();
      } catch (error) {
        lastError = error as Error;

        // Don't retry on certain errors
        if (error instanceof ValidationError || 
            error instanceof AuthenticationError ||
            error instanceof ResourceNotFoundError) {
          throw error;
        }

        if (attempt === maxAttempts) {
          break;
        }

        // Calculate delay with exponential backoff
        let delay = Math.min(baseDelay * Math.pow(backoffFactor, attempt - 1), maxDelay);
        
        if (jitter) {
          delay *= (0.5 + Math.random() * 0.5); // Add ±50% jitter
        }

        // Handle rate limiting
        if (error instanceof RateLimitError && error.retryAfter) {
          delay = Math.min(error.retryAfter * 1000, maxDelay);
        }

        console.warn(`Attempt ${attempt} failed, retrying in ${delay}ms:`, error.message);
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }

    throw lastError!;
  }

  async handleAPIError<T>(operation: () => Promise<T>): Promise<T> {
    try {
      return await operation();
    } catch (error) {
      if (error instanceof NephoranAPIError) {
        this.logAPIError(error);
        
        switch (error.constructor) {
          case ValidationError:
            console.error('Validation Error:', error.message);
            if (error.details) {
              Object.entries(error.details).forEach(([field, details]) => {
                console.error(`  ${field}: ${details}`);
              });
            }
            break;

          case AuthenticationError:
            console.error('Authentication Error:', error.message);
            // Attempt token refresh if available
            if (this.client.auth.canRefresh()) {
              console.log('Attempting to refresh token...');
              await this.client.auth.refresh();
              return operation(); // Retry once after refresh
            }
            break;

          case RateLimitError:
            console.warn('Rate Limit Exceeded:', error.message);
            if (error.retryAfter) {
              console.log(`Retry after: ${error.retryAfter} seconds`);
            }
            break;

          case ResourceNotFoundError:
            console.error('Resource Not Found:', error.message);
            break;

          case CircuitBreakerError:
            console.error('Circuit Breaker Open:', error.message);
            break;

          default:
            console.error('API Error:', error.message);
        }
      } else {
        console.error('Unexpected Error:', error);
      }
      
      throw error;
    }
  }

  private logAPIError(error: NephoranAPIError): void {
    const logData = {
      timestamp: new Date().toISOString(),
      error: error.constructor.name,
      message: error.message,
      statusCode: error.statusCode,
      requestId: error.requestId,
      endpoint: error.endpoint
    };

    console.error('API Error Details:', JSON.stringify(logData, null, 2));
  }

  async createIntentWithFullErrorHandling(intent: any): Promise<any> {
    const operation = () => this.client.intents.create(intent);
    const operationWithErrorHandling = () => this.handleAPIError(operation);

    return this.executeWithRetry(operationWithErrorHandling, {
      maxAttempts: 3,
      baseDelay: 1000
    });
  }
}

// Usage example
async function demonstrateErrorHandling() {
  const client = new NephoranClient(/* config */);
  const resilientOps = new ResilientOperations(client);

  const intent = {
    apiVersion: 'nephoran.io/v1alpha1',
    kind: 'NetworkIntent',
    metadata: { name: 'test-intent', namespace: 'default' },
    spec: { intent: 'Test intent' }
  };

  try {
    const result = await resilientOps.createIntentWithFullErrorHandling(intent);
    console.log('Intent created successfully:', result.metadata.name);
  } catch (error) {
    console.error('Failed to create intent after all retries:', error);
  }
}
```

## Testing

```typescript
import { jest } from '@jest/globals';
import { NephoranClient, NetworkIntent } from '@nephoran/intent-operator-sdk';

// Mock client for testing
class MockNephoranClient {
  intents = {
    create: jest.fn(),
    list: jest.fn(),
    get: jest.fn(),
    update: jest.fn(),
    delete: jest.fn()
  };

  llm = {
    process: jest.fn(),
    processStream: jest.fn()
  };

  rag = {
    query: jest.fn(),
    listDocuments: jest.fn()
  };

  health = {
    check: jest.fn()
  };
}

describe('Nephoran SDK Tests', () => {
  let mockClient: MockNephoranClient;

  beforeEach(() => {
    mockClient = new MockNephoranClient();
  });

  describe('Intent Management', () => {
    test('should create network intent successfully', async () => {
      const mockIntent: NetworkIntent = {
        apiVersion: 'nephoran.io/v1alpha1',
        kind: 'NetworkIntent',
        metadata: {
          name: 'test-intent',
          namespace: 'test',
          uid: '12345'
        },
        spec: {
          intent: 'Deploy test AMF instance',
          priority: 'low'
        },
        status: {
          phase: 'Pending',
          message: 'Intent created'
        }
      };

      mockClient.intents.create.mockResolvedValue(mockIntent);

      const result = await mockClient.intents.create({
        apiVersion: 'nephoran.io/v1alpha1',
        kind: 'NetworkIntent',
        metadata: { name: 'test-intent', namespace: 'test' },
        spec: { intent: 'Deploy test AMF instance' }
      });

      expect(result.metadata.name).toBe('test-intent');
      expect(result.status?.phase).toBe('Pending');
      expect(mockClient.intents.create).toHaveBeenCalledTimes(1);
    });

    test('should handle validation errors', async () => {
      const validationError = new Error('Intent validation failed');
      mockClient.intents.create.mockRejectedValue(validationError);

      await expect(mockClient.intents.create({})).rejects.toThrow('Intent validation failed');
    });
  });

  describe('RAG Operations', () => {
    test('should query knowledge base successfully', async () => {
      const mockRAGResponse = {
        results: [
          {
            text: 'O-RAN Near-RT RIC provides real-time control...',
            score: 0.95,
            source: 'oran-arch-spec-v1.2',
            metadata: { documentType: 'specification' }
          }
        ],
        metadata: {
          totalResults: 1,
          processingTimeMs: 150
        }
      };

      mockClient.rag.query.mockResolvedValue(mockRAGResponse);

      const result = await mockClient.rag.query({
        query: 'O-RAN Near-RT RIC architecture',
        maxResults: 5
      });

      expect(result.results).toHaveLength(1);
      expect(result.results[0].score).toBe(0.95);
      expect(result.metadata.processingTimeMs).toBe(150);
    });
  });

  describe('Error Handling', () => {
    test('should handle rate limiting correctly', async () => {
      const rateLimitError = new Error('Rate limit exceeded');
      (rateLimitError as any).retryAfter = 30;
      
      mockClient.intents.create
        .mockRejectedValueOnce(rateLimitError)
        .mockResolvedValueOnce({ metadata: { name: 'success' } });

      // In a real implementation, this would include retry logic
      await expect(mockClient.intents.create({})).rejects.toThrow('Rate limit exceeded');
    });
  });
});

// Integration test setup
describe('Integration Tests', () => {
  let client: NephoranClient;

  beforeAll(() => {
    if (!process.env.NEPHORAN_CLIENT_ID || !process.env.NEPHORAN_CLIENT_SECRET) {
      console.log('Skipping integration tests - credentials not available');
      return;
    }

    client = new NephoranClient({
      baseURL: process.env.NEPHORAN_TEST_URL || 'https://staging-api.nephoran.com/v1',
      auth: {
        clientId: process.env.NEPHORAN_CLIENT_ID!,
        clientSecret: process.env.NEPHORAN_CLIENT_SECRET!,
        tokenUrl: 'https://auth.nephoran.com/oauth/token',
        scopes: ['intent.read', 'rag.query']
      }
    });
  });

  test('should check API health', async () => {
    if (!client) return;

    const health = await client.health.check();
    expect(health.status).toMatch(/^(healthy|degraded)$/);
    expect(health.components).toHaveProperty('llm-processor');
    expect(health.components).toHaveProperty('rag-service');
  });

  test('should perform RAG query', async () => {
    if (!client) return;

    const response = await client.rag.query({
      query: 'O-RAN architecture overview',
      maxResults: 3,
      threshold: 0.7
    });

    expect(response.results.length).toBeGreaterThan(0);
    expect(response.results[0].score).toBeGreaterThanOrEqual(0.7);
    expect(response.metadata.processingTimeMs).toBeGreaterThan(0);
  });
});
```

This comprehensive JavaScript/TypeScript SDK guide provides production-ready code examples, advanced error handling, real-time monitoring capabilities, comprehensive testing patterns, and enterprise-grade features for both browser and Node.js environments.