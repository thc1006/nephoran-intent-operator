# API Testing and Validation Guide

## Overview

This comprehensive guide covers testing strategies, validation procedures, and automated testing approaches for the Nephoran Intent Operator API. As a production-ready (TRL 9) system, rigorous testing ensures reliability, performance, and compliance with telecommunications standards.

## Testing Strategy

### Testing Pyramid

```
        ┌─────────────────┐
        │   E2E Tests     │  ← 10% (Critical user journeys)
        │                 │
        ├─────────────────┤
        │ Integration     │  ← 20% (API contracts, services)
        │ Tests           │
        ├─────────────────┤
        │                 │
        │   Unit Tests    │  ← 70% (Individual components)
        │                 │
        └─────────────────┘
```

### Test Categories

1. **Functional Testing**
   - API endpoint validation
   - Request/response schema validation
   - Business logic verification
   - Error handling validation

2. **Performance Testing**
   - Load testing (normal traffic)
   - Stress testing (peak traffic)
   - Volume testing (large data sets)
   - Endurance testing (sustained load)

3. **Security Testing**
   - Authentication and authorization
   - Input validation and sanitization
   - Rate limiting verification
   - Penetration testing

4. **Integration Testing**
   - Service-to-service communication
   - Database integration
   - External API dependencies
   - Event streaming validation

5. **Contract Testing**
   - OpenAPI specification compliance
   - Schema validation
   - API versioning compatibility
   - Backward compatibility

## Automated Testing Framework

### Jest + Supertest Setup

```javascript
// package.json
{
  "devDependencies": {
    "@types/jest": "^29.5.0",
    "@types/supertest": "^2.0.12",
    "jest": "^29.5.0",
    "supertest": "^6.3.0",
    "nock": "^13.3.0",
    "swagger-parser": "^10.0.3"
  },
  "scripts": {
    "test": "jest",
    "test:integration": "jest --testPathPattern=integration",
    "test:performance": "jest --testPathPattern=performance",
    "test:security": "jest --testPathPattern=security",
    "test:coverage": "jest --coverage",
    "test:watch": "jest --watch"
  }
}
```

### Test Configuration

```javascript
// jest.config.js
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/tests'],
  testMatch: [
    '**/__tests__/**/*.+(ts|tsx|js)',
    '**/*.(test|spec).+(ts|tsx|js)'
  ],
  transform: {
    '^.+\\.(ts|tsx)$': 'ts-jest'
  },
  collectCoverageFrom: [
    'src/**/*.{js,ts}',
    '!src/**/*.d.ts',
    '!src/tests/**'
  ],
  coverageThreshold: {
    global: {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80
    }
  },
  setupFilesAfterEnv: ['<rootDir>/tests/setup.js'],
  testTimeout: 30000,
  maxWorkers: 4
};
```

### Test Setup and Utilities

```javascript
// tests/setup.js
const { NephoranClient } = require('@nephoran/intent-operator-sdk');
const nock = require('nock');

// Global test configuration
global.testConfig = {
  baseURL: process.env.TEST_API_URL || 'http://localhost:8080/v1',
  authURL: process.env.TEST_AUTH_URL || 'http://localhost:8080/auth',
  clientId: 'test-client',
  clientSecret: 'test-secret'
};

// Setup global mocks
beforeAll(() => {
  // Mock authentication endpoint
  nock(global.testConfig.authURL)
    .persist()
    .post('/oauth/token')
    .reply(200, {
      access_token: 'mock-access-token',
      token_type: 'Bearer',
      expires_in: 3600,
      scope: 'intent.read intent.write rag.query'
    });
});

afterAll(() => {
  nock.cleanAll();
});

// Test utilities
global.TestUtils = {
  createMockClient() {
    return new NephoranClient({
      baseURL: global.testConfig.baseURL,
      auth: {
        clientId: global.testConfig.clientId,
        clientSecret: global.testConfig.clientSecret,
        tokenUrl: `${global.testConfig.authURL}/oauth/token`
      },
      timeout: 5000
    });
  },

  generateTestIntent(overrides = {}) {
    return {
      apiVersion: 'nephoran.io/v1alpha1',
      kind: 'NetworkIntent',
      metadata: {
        name: `test-intent-${Date.now()}`,
        namespace: 'test',
        labels: {
          'test': 'true',
          'environment': 'testing'
        }
      },
      spec: {
        intent: 'Deploy test AMF instance with basic configuration',
        priority: 'low',
        parameters: {
          replicas: 1,
          enableAutoScaling: false
        }
      },
      ...overrides
    };
  },

  async waitForCondition(conditionFn, timeout = 30000, interval = 1000) {
    const start = Date.now();
    
    while (Date.now() - start < timeout) {
      if (await conditionFn()) {
        return true;
      }
      await new Promise(resolve => setTimeout(resolve, interval));
    }
    
    throw new Error(`Condition not met within ${timeout}ms`);
  }
};
```

## Functional Tests

### Intent Management API Tests

```javascript
// tests/functional/intents.test.js
const request = require('supertest');
const nock = require('nock');

describe('Intent Management API', () => {
  let app;
  let client;

  beforeAll(() => {
    client = TestUtils.createMockClient();
  });

  describe('POST /intents', () => {
    test('should create a network intent successfully', async () => {
      const intentData = TestUtils.generateTestIntent();

      const response = await request(global.testConfig.baseURL)
        .post('/intents')
        .set('Authorization', 'Bearer mock-access-token')
        .set('Content-Type', 'application/json')
        .send(intentData)
        .expect(201);

      expect(response.body).toMatchObject({
        apiVersion: 'nephoran.io/v1alpha1',
        kind: 'NetworkIntent',
        metadata: {
          name: expect.stringContaining('test-intent'),
          namespace: 'test',
          uid: expect.any(String)
        },
        spec: expect.objectContaining({
          intent: expect.any(String),
          priority: 'low'
        }),
        status: {
          phase: 'Pending'
        }
      });

      // Validate response time
      expect(response.get('X-Response-Time')).toBeDefined();
      const responseTime = parseFloat(response.get('X-Response-Time'));
      expect(responseTime).toBeLessThan(2000); // < 2 seconds SLA
    });

    test('should validate required fields', async () => {
      const invalidIntent = {
        apiVersion: 'nephoran.io/v1alpha1',
        kind: 'NetworkIntent'
        // Missing metadata and spec
      };

      const response = await request(global.testConfig.baseURL)
        .post('/intents')
        .set('Authorization', 'Bearer mock-access-token')
        .set('Content-Type', 'application/json')
        .send(invalidIntent)
        .expect(400);

      expect(response.body).toMatchObject({
        code: 'INTENT_VALIDATION_FAILED',
        message: expect.stringContaining('validation'),
        details: expect.objectContaining({
          field: expect.any(String),
          reason: expect.any(String)
        })
      });
    });

    test('should enforce intent text length limits', async () => {
      const intentWithShortText = TestUtils.generateTestIntent({
        spec: {
          intent: 'Short', // Too short (< 10 characters)
          priority: 'low'
        }
      });

      await request(global.testConfig.baseURL)
        .post('/intents')
        .set('Authorization', 'Bearer mock-access-token')
        .send(intentWithShortText)
        .expect(400);

      const intentWithLongText = TestUtils.generateTestIntent({
        spec: {
          intent: 'A'.repeat(1001), // Too long (> 1000 characters)
          priority: 'low'
        }
      });

      await request(global.testConfig.baseURL)
        .post('/intents')
        .set('Authorization', 'Bearer mock-access-token')
        .send(intentWithLongText)
        .expect(400);
    });

    test('should handle different intent types', async () => {
      const intentTypes = [
        {
          kind: 'NetworkIntent',
          intent: 'Deploy 5G Core AMF with high availability'
        },
        {
          kind: 'NetworkSlice',
          intent: 'Create eMBB network slice for mobile broadband'
        },
        {
          kind: 'RANIntent',
          intent: 'Deploy O-RAN Near-RT RIC with traffic steering xApp'
        }
      ];

      for (const type of intentTypes) {
        const intentData = TestUtils.generateTestIntent({
          kind: type.kind,
          spec: {
            intent: type.intent,
            priority: 'medium'
          }
        });

        const response = await request(global.testConfig.baseURL)
          .post('/intents')
          .set('Authorization', 'Bearer mock-access-token')
          .send(intentData)
          .expect(201);

        expect(response.body.kind).toBe(type.kind);
      }
    });
  });

  describe('GET /intents', () => {
    test('should list intents with pagination', async () => {
      const response = await request(global.testConfig.baseURL)
        .get('/intents')
        .query({
          limit: 10,
          offset: 0,
          namespace: 'test'
        })
        .set('Authorization', 'Bearer mock-access-token')
        .expect(200);

      expect(response.body).toMatchObject({
        items: expect.any(Array),
        metadata: {
          totalItems: expect.any(Number),
          itemsPerPage: 10,
          currentPage: expect.any(Number),
          hasNext: expect.any(Boolean),
          hasPrevious: expect.any(Boolean)
        }
      });
    });

    test('should filter intents by status', async () => {
      const response = await request(global.testConfig.baseURL)
        .get('/intents')
        .query({
          status: 'Deployed',
          limit: 20
        })
        .set('Authorization', 'Bearer mock-access-token')
        .expect(200);

      response.body.items.forEach(intent => {
        expect(intent.status.phase).toBe('Deployed');
      });
    });

    test('should filter intents by labels', async () => {
      const response = await request(global.testConfig.baseURL)
        .get('/intents')
        .query({
          labels: 'environment=production,component=amf'
        })
        .set('Authorization', 'Bearer mock-access-token')
        .expect(200);

      response.body.items.forEach(intent => {
        expect(intent.metadata.labels).toMatchObject({
          environment: 'production',
          component: 'amf'
        });
      });
    });
  });

  describe('GET /intents/{namespace}/{name}', () => {
    test('should retrieve specific intent', async () => {
      const response = await request(global.testConfig.baseURL)
        .get('/intents/test/test-intent-123')
        .set('Authorization', 'Bearer mock-access-token')
        .expect(200);

      expect(response.body).toMatchObject({
        metadata: {
          name: 'test-intent-123',
          namespace: 'test'
        }
      });
    });

    test('should return 404 for non-existent intent', async () => {
      const response = await request(global.testConfig.baseURL)
        .get('/intents/test/non-existent-intent')
        .set('Authorization', 'Bearer mock-access-token')
        .expect(404);

      expect(response.body.code).toBe('RESOURCE_NOT_FOUND');
    });
  });

  describe('Intent Status Updates', () => {
    test('should track intent processing lifecycle', async () => {
      // Create intent
      const intentData = TestUtils.generateTestIntent();
      const createResponse = await request(global.testConfig.baseURL)
        .post('/intents')
        .set('Authorization', 'Bearer mock-access-token')
        .send(intentData)
        .expect(201);

      const { namespace, name } = createResponse.body.metadata;

      // Wait for processing to start
      await TestUtils.waitForCondition(async () => {
        const statusResponse = await request(global.testConfig.baseURL)
          .get(`/intents/${namespace}/${name}/status`)
          .set('Authorization', 'Bearer mock-access-token');
        
        return statusResponse.body.phase !== 'Pending';
      });

      // Verify status progression
      const finalStatus = await request(global.testConfig.baseURL)
        .get(`/intents/${namespace}/${name}/status`)
        .set('Authorization', 'Bearer mock-access-token')
        .expect(200);

      expect(finalStatus.body.phase).toMatch(/^(Processing|Deployed|Failed)$/);
      expect(finalStatus.body.processingMetrics).toBeDefined();
      expect(finalStatus.body.processingMetrics.totalProcessingTimeMs).toBeGreaterThan(0);
    });
  });
});
```

### LLM Processing API Tests

```javascript
// tests/functional/llm.test.js
describe('LLM Processing API', () => {
  describe('POST /llm/process', () => {
    test('should process 5G network intent', async () => {
      const request_data = {
        intent: 'Deploy a high-availability AMF cluster with auto-scaling for 100,000 concurrent UE connections',
        context: [
          'AMF handles access and mobility management in 5G core',
          'High availability requires N+1 redundancy and state replication',
          'Auto-scaling should be based on connection count and CPU utilization'
        ],
        parameters: {
          model: 'gpt-4o-mini',
          temperature: 0.3,
          maxTokens: 2000
        }
      };

      const response = await request(global.testConfig.baseURL)
        .post('/llm/process')
        .set('Authorization', 'Bearer mock-access-token')
        .send(request_data)
        .expect(200);

      expect(response.body).toMatchObject({
        processedIntent: expect.any(Object),
        networkFunctions: expect.arrayContaining(['AMF']),
        deploymentStrategy: expect.any(String),
        confidence: expect.any(Number),
        oranCompliance: expect.any(Object),
        processingMetrics: {
          tokenUsage: expect.any(Number),
          processingTimeMs: expect.any(Number),
          ragContextUsed: expect.any(Boolean)
        }
      });

      expect(response.body.confidence).toBeGreaterThan(0.7);
      expect(response.body.processingMetrics.processingTimeMs).toBeLessThan(10000);
    });

    test('should process O-RAN intent', async () => {
      const request_data = {
        intent: 'Configure Near-RT RIC with traffic steering and QoS optimization xApps for intelligent RAN control',
        parameters: {
          model: 'gpt-4o-mini',
          temperature: 0.1, // Low temperature for O-RAN compliance
          maxTokens: 1500
        }
      };

      const response = await request(global.testConfig.baseURL)
        .post('/llm/process')
        .set('Authorization', 'Bearer mock-access-token')
        .send(request_data)
        .expect(200);

      expect(response.body.networkFunctions).toContain('Near-RT RIC');
      expect(response.body.processedIntent).toHaveProperty('xApps');
      expect(response.body.processedIntent.xApps).toEqual(
        expect.arrayContaining([
          expect.stringContaining('traffic-steering'),
          expect.stringContaining('qos')
        ])
      );
      expect(response.body.oranCompliance.compliant).toBe(true);
    });

    test('should validate request parameters', async () => {
      const invalidRequests = [
        {
          // Missing intent
          parameters: { model: 'gpt-4o-mini' }
        },
        {
          intent: 'Short', // Too short
          parameters: { model: 'gpt-4o-mini' }
        },
        {
          intent: 'Valid intent text',
          parameters: {
            model: 'invalid-model', // Invalid model
            temperature: 3.0 // Invalid temperature
          }
        }
      ];

      for (const invalidRequest of invalidRequests) {
        await request(global.testConfig.baseURL)
          .post('/llm/process')
          .set('Authorization', 'Bearer mock-access-token')
          .send(invalidRequest)
          .expect(400);
      }
    });

    test('should handle streaming responses', async () => {
      const request_data = {
        intent: 'Design network slicing architecture for smart city deployment',
        streaming: true,
        parameters: {
          model: 'gpt-4o-mini',
          temperature: 0.3
        }
      };

      const response = await request(global.testConfig.baseURL)
        .post('/llm/stream')
        .set('Authorization', 'Bearer mock-access-token')
        .set('Accept', 'text/event-stream')
        .send(request_data)
        .expect(200);

      expect(response.headers['content-type']).toContain('text/event-stream');
      expect(response.headers['cache-control']).toBe('no-cache');
      expect(response.headers['connection']).toBe('keep-alive');
    });
  });
});
```

### RAG API Tests

```javascript
// tests/functional/rag.test.js
describe('RAG Knowledge Retrieval API', () => {
  describe('POST /rag/query', () => {
    test('should retrieve O-RAN knowledge', async () => {
      const query = {
        query: 'O-RAN Near-RT RIC architecture components and E2 interface',
        context: {
          domain: 'o-ran',
          technology: 'near-rt-ric'
        },
        maxResults: 5,
        threshold: 0.8,
        includeMetadata: true
      };

      const response = await request(global.testConfig.baseURL)
        .post('/rag/query')
        .set('Authorization', 'Bearer mock-access-token')
        .send(query)
        .expect(200);

      expect(response.body).toMatchObject({
        results: expect.any(Array),
        metadata: {
          totalResults: expect.any(Number),
          processingTimeMs: expect.any(Number),
          queryEnhanced: expect.any(String)
        }
      });

      expect(response.body.results.length).toBeGreaterThan(0);
      expect(response.body.results.length).toBeLessThanOrEqual(5);

      response.body.results.forEach(result => {
        expect(result).toMatchObject({
          text: expect.any(String),
          score: expect.any(Number),
          source: expect.any(String),
          metadata: expect.any(Object)
        });
        expect(result.score).toBeGreaterThanOrEqual(0.8);
      });

      expect(response.body.metadata.processingTimeMs).toBeLessThan(1000);
    });

    test('should retrieve 5G Core knowledge', async () => {
      const query = {
        query: 'AMF and SMF deployment configuration for high availability',
        context: {
          domain: '5g-core',
          technology: 'service-based-architecture'
        },
        maxResults: 3,
        threshold: 0.75
      };

      const response = await request(global.testConfig.baseURL)
        .post('/rag/query')
        .set('Authorization', 'Bearer mock-access-token')
        .send(query)
        .expect(200);

      expect(response.body.results).toHaveLength(3);
      
      const hasAMFContent = response.body.results.some(result => 
        result.text.toLowerCase().includes('amf')
      );
      const hasSMFContent = response.body.results.some(result => 
        result.text.toLowerCase().includes('smf')
      );

      expect(hasAMFContent || hasSMFContent).toBe(true);
    });

    test('should validate query parameters', async () => {
      const invalidQueries = [
        {
          // Missing query
          maxResults: 5
        },
        {
          query: 'Test', // Too short
          maxResults: 5
        },
        {
          query: 'Valid query text',
          maxResults: 25, // Too many results
          threshold: 1.5 // Invalid threshold
        }
      ];

      for (const invalidQuery of invalidQueries) {
        await request(global.testConfig.baseURL)
          .post('/rag/query')
          .set('Authorization', 'Bearer mock-access-token')
          .send(invalidQuery)
          .expect(400);
      }
    });

    test('should handle empty results gracefully', async () => {
      const query = {
        query: 'completely unrelated topic like cooking recipes',
        context: {
          domain: 'o-ran'
        },
        maxResults: 5,
        threshold: 0.95 // Very high threshold
      };

      const response = await request(global.testConfig.baseURL)
        .post('/rag/query')
        .set('Authorization', 'Bearer mock-access-token')
        .send(query)
        .expect(200);

      expect(response.body.results).toHaveLength(0);
      expect(response.body.metadata.totalResults).toBe(0);
    });
  });

  describe('GET /rag/knowledge-base', () => {
    test('should list knowledge base documents', async () => {
      const response = await request(global.testConfig.baseURL)
        .get('/rag/knowledge-base')
        .query({
          domain: 'o-ran',
          limit: 10
        })
        .set('Authorization', 'Bearer mock-access-token')
        .expect(200);

      expect(response.body).toMatchObject({
        documents: expect.any(Array)
      });

      response.body.documents.forEach(doc => {
        expect(doc).toMatchObject({
          id: expect.any(String),
          title: expect.any(String),
          source: expect.any(String),
          documentType: expect.any(String),
          domain: expect.any(String),
          lastUpdated: expect.any(String),
          chunkCount: expect.any(Number)
        });
      });
    });
  });
});
```

## Performance Testing

### Load Testing with Artillery

```yaml
# tests/performance/load-test.yml
config:
  target: 'https://api.nephoran.com/v1'
  phases:
    - duration: 60
      arrivalRate: 10
      name: "Warm up"
    - duration: 300  
      arrivalRate: 50
      name: "Normal load"
    - duration: 120
      arrivalRate: 100
      name: "Peak load"
  variables:
    clientId: "{{ $randomString() }}"
  processor: "./load-test-processor.js"

scenarios:
  - name: "Intent Creation Flow"
    weight: 40
    flow:
      - post:
          url: "/auth/oauth/token"
          form:
            grant_type: "client_credentials"
            client_id: "{{ clientId }}"
            client_secret: "{{ $randomString() }}"
            scope: "intent.write"
          capture:
            - json: "$.access_token"
              as: "accessToken"
      - post:
          url: "/intents"
          headers:
            Authorization: "Bearer {{ accessToken }}"
            Content-Type: "application/json"
          json:
            apiVersion: "nephoran.io/v1alpha1"
            kind: "NetworkIntent"
            metadata:
              name: "load-test-{{ $randomInt(1000, 9999) }}"
              namespace: "load-test"
            spec:
              intent: "Deploy AMF instance for load testing with {{ $randomInt(100, 1000) }} concurrent connections"
              priority: "medium"
          expect:
            - statusCode: 201
            - hasHeader: "X-Response-Time"
          capture:
            - json: "$.metadata.name"
              as: "intentName"
      - get:
          url: "/intents/load-test/{{ intentName }}/status"
          headers:
            Authorization: "Bearer {{ accessToken }}"
          expect:
            - statusCode: 200

  - name: "RAG Query Flow"
    weight: 30
    flow:
      - post:
          url: "/auth/oauth/token"
          form:
            grant_type: "client_credentials"
            client_id: "{{ clientId }}"
            client_secret: "{{ $randomString() }}"
            scope: "rag.query"
          capture:
            - json: "$.access_token"
              as: "accessToken"
      - post:
          url: "/rag/query"
          headers:
            Authorization: "Bearer {{ accessToken }}"
            Content-Type: "application/json"
          json:
            query: "{{ $randomElement(['O-RAN Near-RT RIC architecture', '5G AMF deployment', 'Network slicing configuration', 'UPF high availability']) }}"
            context:
              domain: "{{ $randomElement(['o-ran', '5g-core', 'network-slicing']) }}"
            maxResults: 5
            threshold: 0.7
          expect:
            - statusCode: 200
            - hasProperty: "results"

  - name: "Health Check"
    weight: 20
    flow:
      - get:
          url: "/health"
          expect:
            - statusCode: 200
            - contentType: json

  - name: "LLM Processing"
    weight: 10
    flow:
      - post:
          url: "/auth/oauth/token"
          form:
            grant_type: "client_credentials"
            client_id: "{{ clientId }}"
            client_secret: "{{ $randomString() }}"
            scope: "intent.write"
          capture:
            - json: "$.access_token"
              as: "accessToken"
      - post:
          url: "/llm/process"
          headers:
            Authorization: "Bearer {{ accessToken }}"
            Content-Type: "application/json"
          json:
            intent: "Deploy {{ $randomElement(['AMF', 'SMF', 'UPF']) }} with {{ $randomElement(['high availability', 'auto-scaling', 'load balancing']) }}"
            parameters:
              model: "gpt-4o-mini"
              temperature: 0.3
              maxTokens: 1000
          expect:
            - statusCode: 200
            - hasProperty: "confidence"
```

### Performance Test Processor

```javascript
// tests/performance/load-test-processor.js
module.exports = {
  // Custom functions for Artillery
  setCustomHeaders,
  validateResponseTime,
  trackMetrics
};

function setCustomHeaders(requestParams, context, ee, next) {
  requestParams.headers = requestParams.headers || {};
  requestParams.headers['X-Load-Test'] = 'true';
  requestParams.headers['X-Request-ID'] = `load-test-${Date.now()}-${Math.random()}`;
  return next();
}

function validateResponseTime(requestParams, response, context, ee, next) {
  const responseTime = response.timings.response;
  
  // SLA validation: 95% of requests should complete within 2 seconds
  if (responseTime > 2000) {
    ee.emit('counter', 'sla.violation.response_time', 1);
  }

  // Track response time percentiles
  ee.emit('histogram', 'response_time', responseTime);
  
  return next();
}

function trackMetrics(requestParams, response, context, ee, next) {
  // Track business metrics
  if (response.statusCode === 201 && requestParams.url.includes('/intents')) {
    ee.emit('counter', 'business.intents.created', 1);
  }
  
  if (response.statusCode === 200 && requestParams.url.includes('/rag/query')) {
    const results = response.body ? JSON.parse(response.body).results : [];
    ee.emit('histogram', 'rag.results.count', results.length);
    ee.emit('histogram', 'rag.processing.time', 
      response.body ? JSON.parse(response.body).metadata.processingTimeMs : 0);
  }

  return next();
}
```

### Stress Testing

```javascript
// tests/performance/stress-test.js
const { performance } = require('perf_hooks');

describe('Stress Testing', () => {
  const STRESS_DURATION = 300000; // 5 minutes
  const MAX_CONCURRENT_REQUESTS = 200;
  
  test('should handle high concurrent intent creation load', async () => {
    const client = TestUtils.createMockClient();
    const startTime = performance.now();
    const requests = [];
    const results = {
      successful: 0,
      failed: 0,
      responseTimeSum: 0,
      maxResponseTime: 0
    };

    // Generate load for specified duration
    while (performance.now() - startTime < STRESS_DURATION) {
      // Maintain concurrent request level
      while (requests.length < MAX_CONCURRENT_REQUESTS) {
        const requestStart = performance.now();
        const promise = client.intents.create(TestUtils.generateTestIntent())
          .then(result => {
            const responseTime = performance.now() - requestStart;
            results.successful++;
            results.responseTimeSum += responseTime;
            results.maxResponseTime = Math.max(results.maxResponseTime, responseTime);
            return result;
          })
          .catch(error => {
            results.failed++;
            console.error('Request failed:', error.message);
          });

        requests.push(promise);
      }

      // Wait for some requests to complete
      await Promise.race(requests);
      
      // Remove completed requests
      const completedRequests = [];
      for (let i = 0; i < requests.length; i++) {
        const request = requests[i];
        if (await Promise.resolve(request).then(() => true).catch(() => true)) {
          completedRequests.push(i);
        }
      }
      
      // Remove completed requests in reverse order
      completedRequests.reverse().forEach(index => {
        requests.splice(index, 1);
      });
    }

    // Wait for remaining requests to complete
    await Promise.allSettled(requests);

    const totalRequests = results.successful + results.failed;
    const averageResponseTime = results.responseTimeSum / results.successful;
    const successRate = (results.successful / totalRequests) * 100;

    console.log('Stress Test Results:');
    console.log(`- Total Requests: ${totalRequests}`);
    console.log(`- Successful: ${results.successful}`);
    console.log(`- Failed: ${results.failed}`);
    console.log(`- Success Rate: ${successRate.toFixed(2)}%`);
    console.log(`- Average Response Time: ${averageResponseTime.toFixed(2)}ms`);
    console.log(`- Max Response Time: ${results.maxResponseTime.toFixed(2)}ms`);

    // Stress test assertions
    expect(successRate).toBeGreaterThan(95); // 95% success rate
    expect(averageResponseTime).toBeLessThan(5000); // 5 second average
    expect(results.maxResponseTime).toBeLessThan(30000); // 30 second max
  });

  test('should maintain performance under memory pressure', async () => {
    const client = TestUtils.createMockClient();
    const memoryUsage = [];
    const startMemory = process.memoryUsage();

    // Create memory pressure by generating large intents
    for (let i = 0; i < 1000; i++) {
      const largeIntent = TestUtils.generateTestIntent({
        spec: {
          intent: 'Deploy comprehensive 5G network with detailed configuration: ' + 
                 'A'.repeat(500), // Large intent text
          parameters: {
            ...Array(100).fill(0).reduce((acc, _, idx) => {
              acc[`param_${idx}`] = `value_${idx}_${'X'.repeat(50)}`;
              return acc;
            }, {})
          }
        }
      });

      try {
        await client.intents.create(largeIntent);
      } catch (error) {
        // Expected to fail under memory pressure
      }

      // Sample memory usage every 100 iterations
      if (i % 100 === 0) {
        memoryUsage.push(process.memoryUsage());
      }
    }

    const endMemory = process.memoryUsage();
    const memoryGrowth = endMemory.heapUsed - startMemory.heapUsed;

    console.log('Memory Usage Analysis:');
    console.log(`- Start Memory: ${(startMemory.heapUsed / 1024 / 1024).toFixed(2)} MB`);
    console.log(`- End Memory: ${(endMemory.heapUsed / 1024 / 1024).toFixed(2)} MB`);
    console.log(`- Memory Growth: ${(memoryGrowth / 1024 / 1024).toFixed(2)} MB`);

    // Memory usage should not grow excessively
    expect(memoryGrowth).toBeLessThan(500 * 1024 * 1024); // < 500MB growth
  });
});
```

## Security Testing

### Authentication and Authorization Tests

```javascript
// tests/security/auth.test.js
describe('Authentication & Authorization', () => {
  test('should reject requests without authentication', async () => {
    await request(global.testConfig.baseURL)
      .get('/intents')
      .expect(401);

    await request(global.testConfig.baseURL)
      .post('/intents')
      .send(TestUtils.generateTestIntent())
      .expect(401);
  });

  test('should reject invalid tokens', async () => {
    const invalidTokens = [
      'invalid-token',
      'Bearer invalid-token',
      'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature',
      '' // Empty token
    ];

    for (const token of invalidTokens) {
      await request(global.testConfig.baseURL)
        .get('/intents')
        .set('Authorization', token)
        .expect(401);
    }
  });

  test('should enforce scope-based authorization', async () => {
    // Token with only read scope
    const readOnlyToken = 'read-only-token';
    
    // Should allow read operations
    await request(global.testConfig.baseURL)
      .get('/intents')
      .set('Authorization', `Bearer ${readOnlyToken}`)
      .expect(200);

    // Should reject write operations
    await request(global.testConfig.baseURL)
      .post('/intents')
      .set('Authorization', `Bearer ${readOnlyToken}`)
      .send(TestUtils.generateTestIntent())
      .expect(403);
  });

  test('should handle token expiration', async () => {
    const expiredToken = 'expired-token';
    
    await request(global.testConfig.baseURL)
      .get('/intents')
      .set('Authorization', `Bearer ${expiredToken}`)
      .expect(401);
  });
});
```

### Input Validation and Sanitization Tests

```javascript
// tests/security/input-validation.test.js
describe('Input Validation & Sanitization', () => {
  test('should prevent SQL injection attempts', async () => {
    const maliciousInputs = [
      "'; DROP TABLE intents; --",
      "1' OR '1'='1",
      "1'; DELETE FROM intents WHERE 1=1; --",
      "admin'/*",
      "' UNION SELECT * FROM users --"
    ];

    for (const maliciousInput of maliciousInputs) {
      const intent = TestUtils.generateTestIntent({
        metadata: { name: maliciousInput },
        spec: { intent: maliciousInput }
      });

      const response = await request(global.testConfig.baseURL)
        .post('/intents')
        .set('Authorization', 'Bearer mock-access-token')
        .send(intent);

      // Should either be rejected or sanitized
      expect([400, 422]).toContain(response.status);
    }
  });

  test('should prevent script injection in intent text', async () => {
    const scriptInjections = [
      '<script>alert("XSS")</script>',
      'javascript:alert("XSS")',
      '<img src="x" onerror="alert(\'XSS\')">',
      '${7*7}', // Template injection
      '{{constructor.constructor("alert(\\"XSS\\")")()}}'
    ];

    for (const injection of scriptInjections) {
      const intent = TestUtils.generateTestIntent({
        spec: { intent: `Deploy AMF with ${injection} configuration` }
      });

      const response = await request(global.testConfig.baseURL)
        .post('/intents')
        .set('Authorization', 'Bearer mock-access-token')
        .send(intent);

      if (response.status === 201) {
        // If accepted, ensure script content is sanitized
        expect(response.body.spec.intent).not.toContain('<script>');
        expect(response.body.spec.intent).not.toContain('javascript:');
        expect(response.body.spec.intent).not.toContain('onerror');
      }
    }
  });

  test('should validate input lengths and formats', async () => {
    const testCases = [
      {
        description: 'extremely long intent name',
        data: TestUtils.generateTestIntent({
          metadata: { name: 'a'.repeat(1000) }
        })
      },
      {
        description: 'invalid characters in name',
        data: TestUtils.generateTestIntent({
          metadata: { name: 'invalid@name#with$special%chars' }
        })
      },
      {
        description: 'deeply nested parameters',
        data: TestUtils.generateTestIntent({
          spec: {
            intent: 'Valid intent',
            parameters: {
              level1: {
                level2: {
                  level3: {
                    level4: {
                      level5: {
                        // Very deeply nested
                        data: 'test'
                      }
                    }
                  }
                }
              }
            }
          }
        })
      }
    ];

    for (const testCase of testCases) {
      const response = await request(global.testConfig.baseURL)
        .post('/intents')
        .set('Authorization', 'Bearer mock-access-token')
        .send(testCase.data);

      // Should validate and reject invalid inputs
      expect([400, 422]).toContain(response.status);
    }
  });
});
```

### Rate Limiting Tests

```javascript
// tests/security/rate-limiting.test.js
describe('Rate Limiting', () => {
  test('should enforce rate limits for authenticated users', async () => {
    const token = 'mock-access-token';
    const requests = [];

    // Send requests rapidly to trigger rate limiting
    for (let i = 0; i < 150; i++) { // Exceed 100/minute limit
      requests.push(
        request(global.testConfig.baseURL)
          .get('/health')
          .set('Authorization', `Bearer ${token}`)
      );
    }

    const responses = await Promise.allSettled(requests);
    
    const rateLimitedResponses = responses.filter(
      response => response.value?.status === 429
    );

    expect(rateLimitedResponses.length).toBeGreaterThan(0);
    
    // Check rate limit headers
    const rateLimitResponse = rateLimitedResponses[0].value;
    expect(rateLimitResponse.headers).toHaveProperty('x-ratelimit-limit');
    expect(rateLimitResponse.headers).toHaveProperty('x-ratelimit-remaining');
    expect(rateLimitResponse.headers).toHaveProperty('x-ratelimit-reset');
  });

  test('should have stricter limits for expensive operations', async () => {
    const token = 'mock-access-token';
    const llmRequests = [];

    // LLM processing has lower rate limit (10/minute)
    for (let i = 0; i < 15; i++) {
      llmRequests.push(
        request(global.testConfig.baseURL)
          .post('/llm/process')
          .set('Authorization', `Bearer ${token}`)
          .send({
            intent: 'Deploy AMF for rate limit testing',
            parameters: { model: 'gpt-4o-mini' }
          })
      );
    }

    const responses = await Promise.allSettled(llmRequests);
    const rateLimited = responses.filter(
      response => response.value?.status === 429
    ).length;

    expect(rateLimited).toBeGreaterThan(0);
  });
});
```

## Contract Testing with OpenAPI

```javascript
// tests/contract/openapi-validation.test.js
const SwaggerParser = require('swagger-parser');
const path = require('path');

describe('OpenAPI Contract Validation', () => {
  let apiSpec;

  beforeAll(async () => {
    const specPath = path.join(__dirname, '../../docs/api/openapi/nephoran-intent-operator-v1.yaml');
    apiSpec = await SwaggerParser.validate(specPath);
  });

  test('should have valid OpenAPI specification', () => {
    expect(apiSpec).toBeDefined();
    expect(apiSpec.openapi).toBe('3.0.3');
    expect(apiSpec.info.version).toBe('1.0.0');
  });

  test('should validate response schemas', async () => {
    // Test intent creation response
    const response = await request(global.testConfig.baseURL)
      .post('/intents')
      .set('Authorization', 'Bearer mock-access-token')
      .send(TestUtils.generateTestIntent());

    const intentSchema = apiSpec.components.schemas.NetworkIntent;
    
    // Basic schema validation
    expect(response.body).toHaveProperty('apiVersion');
    expect(response.body).toHaveProperty('kind');
    expect(response.body).toHaveProperty('metadata');
    expect(response.body).toHaveProperty('spec');
    expect(response.body).toHaveProperty('status');
  });

  test('should validate error response schemas', async () => {
    const response = await request(global.testConfig.baseURL)
      .get('/intents/nonexistent/intent')
      .set('Authorization', 'Bearer mock-access-token')
      .expect(404);

    expect(response.body).toMatchObject({
      code: expect.any(String),
      message: expect.any(String),
      requestId: expect.any(String),
      timestamp: expect.any(String)
    });
  });
});
```

## Continuous Integration Pipeline

```yaml
# .github/workflows/api-testing.yml
name: API Testing Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm ci
      - run: npm run test:unit
      - uses: codecov/codecov-action@v3

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm ci
      - run: npm run test:integration
        env:
          DATABASE_URL: postgresql://postgres:test@localhost:5432/test

  contract-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm ci
      - run: npm run test:contract

  performance-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm ci
      - run: npm install -g artillery
      - run: artillery run tests/performance/load-test.yml
      - uses: actions/upload-artifact@v3
        with:
          name: performance-results
          path: artillery-report.html

  security-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - run: npm ci
      - run: npm run test:security
      - run: npm audit --audit-level=high
```

This comprehensive API testing guide provides structured approaches for validating the Nephoran Intent Operator API across functional, performance, security, and contract dimensions, ensuring production readiness and compliance with telecommunications industry standards.