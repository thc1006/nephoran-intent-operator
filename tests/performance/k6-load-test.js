import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { randomString, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const intentProcessingDuration = new Trend('intent_processing_duration');
const intentSuccessRate = new Rate('intent_success_rate');
const intentErrorRate = new Rate('intent_error_rate');
const intentThroughput = new Counter('intent_throughput');
const activeConcurrentIntents = new Gauge('active_concurrent_intents');

// Test configuration
export const options = {
  scenarios: {
    // Baseline test: 1,000 intents per minute for 10 minutes
    baseline_load: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 200,
      stages: [
        { duration: '30s', target: 17 },  // Warm-up to ~1000/min
        { duration: '10m', target: 17 },  // Sustain 1000/min (17/sec)
        { duration: '30s', target: 0 },   // Cool-down
      ],
    },
    
    // Spike test: Sudden 2x load increase
    spike_test: {
      executor: 'ramping-arrival-rate',
      startRate: 17,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 400,
      startTime: '12m',
      stages: [
        { duration: '10s', target: 17 },   // Normal load
        { duration: '5s', target: 34 },    // Spike to 2x
        { duration: '3m', target: 34 },    // Sustain spike
        { duration: '10s', target: 17 },   // Return to normal
        { duration: '2m', target: 17 },    // Observe recovery
      ],
    },
    
    // Stress test: Gradual increase to breaking point
    stress_test: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      preAllocatedVUs: 200,
      maxVUs: 1000,
      startTime: '20m',
      stages: [
        { duration: '2m', target: 20 },   // 1200/min
        { duration: '2m', target: 30 },   // 1800/min
        { duration: '2m', target: 40 },   // 2400/min
        { duration: '2m', target: 50 },   // 3000/min
        { duration: '2m', target: 60 },   // 3600/min
        { duration: '2m', target: 70 },   // 4200/min
        { duration: '2m', target: 80 },   // 4800/min
        { duration: '2m', target: 90 },   // 5400/min
        { duration: '2m', target: 100 },  // 6000/min
        { duration: '1m', target: 0 },    // Cool-down
      ],
    },
  },
  
  thresholds: {
    // SLA Requirements
    'http_req_duration{type:intent_create}': ['p(95)<2000'], // P95 < 2s
    'intent_error_rate': ['rate<0.01'],                       // Error rate < 1%
    'intent_success_rate': ['rate>0.99'],                     // Success rate > 99%
    'http_req_failed': ['rate<0.01'],                         // HTTP failures < 1%
    
    // Additional performance thresholds
    'http_req_duration{type:intent_status}': ['p(95)<500'],   // Status check < 500ms
    'http_req_duration{type:health_check}': ['p(95)<100'],    // Health check < 100ms
    'intent_processing_duration': ['p(95)<5000', 'p(99)<10000'], // Processing time
  },
};

// Test configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const NAMESPACE = __ENV.NAMESPACE || 'nephoran-system';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';

// Intent templates
const INTENT_TEMPLATES = [
  {
    type: 'amf-deployment',
    template: {
      apiVersion: 'intent.nephoran.io/v1',
      kind: 'NetworkIntent',
      metadata: {
        name: '',
        namespace: NAMESPACE,
      },
      spec: {
        intent: 'Deploy AMF with high availability and auto-scaling',
        priority: 'high',
        parameters: {
          replicas: '3',
          autoScaling: 'true',
          minReplicas: '2',
          maxReplicas: '10',
          targetCPU: '70',
        },
      },
    },
  },
  {
    type: 'smf-deployment',
    template: {
      apiVersion: 'intent.nephoran.io/v1',
      kind: 'NetworkIntent',
      metadata: {
        name: '',
        namespace: NAMESPACE,
      },
      spec: {
        intent: 'Deploy SMF for production with session management',
        priority: 'medium',
        parameters: {
          replicas: '2',
          sessionCapacity: '100000',
          pduSessionTimeout: '3600',
        },
      },
    },
  },
  {
    type: 'upf-deployment',
    template: {
      apiVersion: 'intent.nephoran.io/v1',
      kind: 'NetworkIntent',
      metadata: {
        name: '',
        namespace: NAMESPACE,
      },
      spec: {
        intent: 'Deploy UPF with high throughput configuration',
        priority: 'high',
        parameters: {
          replicas: '4',
          throughput: '10Gbps',
          latency: 'ultra-low',
        },
      },
    },
  },
  {
    type: 'network-slice',
    template: {
      apiVersion: 'intent.nephoran.io/v1',
      kind: 'NetworkIntent',
      metadata: {
        name: '',
        namespace: NAMESPACE,
      },
      spec: {
        intent: 'Create eMBB network slice for video streaming',
        priority: 'high',
        parameters: {
          sliceType: 'eMBB',
          bandwidth: '1Gbps',
          qos: '5QI-9',
          maxUsers: '10000',
        },
      },
    },
  },
  {
    type: 'ran-deployment',
    template: {
      apiVersion: 'intent.nephoran.io/v1',
      kind: 'NetworkIntent',
      metadata: {
        name: '',
        namespace: NAMESPACE,
      },
      spec: {
        intent: 'Deploy O-RAN compliant DU with carrier aggregation',
        priority: 'medium',
        parameters: {
          carriers: '3',
          bandwidth: '100MHz',
          mimo: '4x4',
        },
      },
    },
  },
];

// Helper functions
function generateIntentName() {
  return `intent-${Date.now()}-${randomString(8)}`;
}

function selectIntentTemplate() {
  const template = randomItem(INTENT_TEMPLATES);
  const intent = JSON.parse(JSON.stringify(template.template));
  intent.metadata.name = generateIntentName();
  return { type: template.type, intent };
}

function getHeaders() {
  const headers = {
    'Content-Type': 'application/json',
  };
  
  if (AUTH_TOKEN) {
    headers['Authorization'] = `Bearer ${AUTH_TOKEN}`;
  }
  
  return headers;
}

// API functions
function createIntent(intent) {
  const startTime = Date.now();
  
  const response = http.post(
    `${BASE_URL}/api/v1/namespaces/${NAMESPACE}/intents`,
    JSON.stringify(intent),
    {
      headers: getHeaders(),
      tags: { type: 'intent_create' },
      timeout: '30s',
    }
  );
  
  const duration = Date.now() - startTime;
  intentProcessingDuration.add(duration);
  
  const success = check(response, {
    'intent created': (r) => r.status === 201 || r.status === 200,
    'has intent ID': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.metadata && body.metadata.uid;
      } catch (e) {
        return false;
      }
    },
  });
  
  intentSuccessRate.add(success ? 1 : 0);
  intentErrorRate.add(success ? 0 : 1);
  intentThroughput.add(1);
  
  if (success) {
    try {
      const body = JSON.parse(response.body);
      return body.metadata.uid;
    } catch (e) {
      console.error('Failed to parse response:', e);
    }
  } else {
    console.error(`Intent creation failed: ${response.status} - ${response.body}`);
  }
  
  return null;
}

function checkIntentStatus(intentId) {
  const response = http.get(
    `${BASE_URL}/api/v1/namespaces/${NAMESPACE}/intents/${intentId}/status`,
    {
      headers: getHeaders(),
      tags: { type: 'intent_status' },
      timeout: '10s',
    }
  );
  
  const success = check(response, {
    'status retrieved': (r) => r.status === 200,
    'has phase': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status && body.status.phase;
      } catch (e) {
        return false;
      }
    },
  });
  
  if (success) {
    try {
      const body = JSON.parse(response.body);
      return body.status.phase;
    } catch (e) {
      console.error('Failed to parse status response:', e);
    }
  }
  
  return 'Unknown';
}

function healthCheck() {
  const endpoints = [
    '/health',
    '/ready',
    '/metrics',
  ];
  
  for (const endpoint of endpoints) {
    const response = http.get(
      `${BASE_URL}${endpoint}`,
      {
        tags: { type: 'health_check' },
        timeout: '5s',
      }
    );
    
    check(response, {
      [`${endpoint} healthy`]: (r) => r.status === 200,
    });
  }
}

// Main test scenario
export default function () {
  // Track active concurrent intents
  activeConcurrentIntents.add(__VU);
  
  group('Intent Creation and Processing', () => {
    // Select and create intent
    const { type, intent } = selectIntentTemplate();
    const intentId = createIntent(intent.intent);
    
    if (intentId) {
      // Wait a bit before checking status
      sleep(randomItem([1, 2, 3]));
      
      // Check intent status multiple times
      let phase = 'Pending';
      let attempts = 0;
      const maxAttempts = 10;
      
      while (phase !== 'Succeeded' && phase !== 'Failed' && attempts < maxAttempts) {
        phase = checkIntentStatus(intentId);
        attempts++;
        
        if (phase !== 'Succeeded' && phase !== 'Failed') {
          sleep(2);
        }
      }
      
      // Validate final status
      check(phase, {
        'intent succeeded': (p) => p === 'Succeeded',
        'intent not failed': (p) => p !== 'Failed',
      });
      
      // Log if intent failed or timed out
      if (phase === 'Failed') {
        console.error(`Intent ${intentId} failed after ${attempts} checks`);
      } else if (attempts >= maxAttempts) {
        console.warn(`Intent ${intentId} timed out after ${attempts} checks, last phase: ${phase}`);
      }
    }
  });
  
  // Periodic health checks (10% of requests)
  if (Math.random() < 0.1) {
    group('Health Checks', () => {
      healthCheck();
    });
  }
  
  // Reduce active concurrent count
  activeConcurrentIntents.add(-__VU);
}

// Setup function - runs once per VU
export function setup() {
  console.log('Starting Nephoran Intent Operator load test');
  console.log(`Target URL: ${BASE_URL}`);
  console.log(`Namespace: ${NAMESPACE}`);
  
  // Verify connectivity
  const response = http.get(`${BASE_URL}/health`);
  if (response.status !== 200) {
    throw new Error(`Health check failed: ${response.status}`);
  }
  
  return {
    startTime: Date.now(),
  };
}

// Teardown function - runs once after all iterations
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log(`Test completed in ${duration} seconds`);
}

// Handle test lifecycle events
export function handleSummary(data) {
  const summary = {
    timestamp: new Date().toISOString(),
    duration: data.state.testRunDurationMs,
    scenarios: {},
    metrics: {
      intent_success_rate: data.metrics.intent_success_rate,
      intent_error_rate: data.metrics.intent_error_rate,
      intent_throughput: data.metrics.intent_throughput,
      http_req_duration_p95: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(95)'] : null,
      http_req_duration_p99: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(99)'] : null,
    },
    thresholds: data.root_group.checks,
  };
  
  // Generate summary report
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'summary.json': JSON.stringify(summary, null, 2),
    'summary.html': htmlReport(data),
  };
}

// Helper function to generate text summary
function textSummary(data, options) {
  // This would normally use k6's built-in textSummary
  // For now, return a simple summary
  return `
Load Test Summary
=================
Duration: ${data.state.testRunDurationMs}ms
VUs: ${data.state.vus}
Iterations: ${data.state.iterations}

Key Metrics:
- Success Rate: ${(data.metrics.intent_success_rate.values.rate * 100).toFixed(2)}%
- Error Rate: ${(data.metrics.intent_error_rate.values.rate * 100).toFixed(2)}%
- P95 Latency: ${data.metrics.http_req_duration.values['p(95)']}ms
- P99 Latency: ${data.metrics.http_req_duration.values['p(99)']}ms
- Throughput: ${data.metrics.intent_throughput.values.count} intents

Thresholds:
${Object.entries(data.metrics).map(([key, value]) => 
  `- ${key}: ${value.thresholds ? (value.thresholds.passes ? 'PASS' : 'FAIL') : 'N/A'}`
).join('\n')}
`;
}

// Helper function to generate HTML report
function htmlReport(data) {
  return `
<!DOCTYPE html>
<html>
<head>
  <title>Nephoran Load Test Report</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 20px; }
    h1 { color: #333; }
    .metric { margin: 10px 0; padding: 10px; background: #f5f5f5; }
    .pass { color: green; }
    .fail { color: red; }
  </style>
</head>
<body>
  <h1>Nephoran Intent Operator Load Test Report</h1>
  <div class="summary">
    <h2>Test Summary</h2>
    <p>Duration: ${data.state.testRunDurationMs}ms</p>
    <p>Total Iterations: ${data.state.iterations}</p>
  </div>
  <div class="metrics">
    <h2>Key Metrics</h2>
    <div class="metric">Success Rate: ${(data.metrics.intent_success_rate.values.rate * 100).toFixed(2)}%</div>
    <div class="metric">Error Rate: ${(data.metrics.intent_error_rate.values.rate * 100).toFixed(2)}%</div>
    <div class="metric">P95 Latency: ${data.metrics.http_req_duration.values['p(95)']}ms</div>
    <div class="metric">P99 Latency: ${data.metrics.http_req_duration.values['p(99)']}ms</div>
  </div>
</body>
</html>
`;
}