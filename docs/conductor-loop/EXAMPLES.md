# Conductor Loop - Examples

This document provides comprehensive examples and use cases for the conductor-loop component.

## Table of Contents

- [Simple Scaling Examples](#simple-scaling-examples)
- [Complex O-RAN Deployment](#complex-o-ran-deployment)
- [CI/CD Integration](#cicd-integration)
- [Windows Deployment](#windows-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Docker Compose Setup](#docker-compose-setup)
- [Monitoring Integration](#monitoring-integration)
- [Error Handling Examples](#error-handling-examples)

## Simple Scaling Examples

### Basic Scale-Up Operation

**Intent File**: `intent-scale-up-basic.json`
```json
{
  "intent_type": "scaling",
  "target": "web-service",
  "namespace": "production",
  "replicas": 5,
  "reason": "Increased traffic load",
  "source": "user",
  "correlation_id": "scale-2024-08-15-001"
}
```

**Usage**:
```bash
# Copy intent file to handoff directory
cp intent-scale-up-basic.json /opt/conductor-loop/data/handoff/

# Monitor processing
tail -f /opt/conductor-loop/logs/conductor-loop.log

# Check status
cat /opt/conductor-loop/data/handoff/status/intent-scale-up-basic.json.status
```

**Expected Output**:
```json
{
  "intent_file": "intent-scale-up-basic.json",
  "status": "success",
  "message": "Successfully scaled web-service to 5 replicas",
  "timestamp": "2024-08-15T14:30:15Z",
  "processed_by": "conductor-loop",
  "worker_id": 1,
  "processing_duration": "2.1s",
  "mode": "direct",
  "correlation_id": "scale-2024-08-15-001"
}
```

### Scale-Down to Zero

**Intent File**: `intent-scale-down-zero.json`
```json
{
  "intent_type": "scaling",
  "target": "batch-processor",
  "namespace": "batch-jobs",
  "replicas": 0,
  "reason": "Scheduled maintenance window",
  "source": "automation",
  "priority": "low",
  "correlation_id": "maintenance-2024-08-15"
}
```

### Emergency Scale-Up

**Intent File**: `intent-emergency-scale.json`
```json
{
  "intent_type": "scaling",
  "target": "api-gateway",
  "namespace": "production",
  "replicas": 20,
  "reason": "Traffic spike detected - emergency response",
  "source": "automation",
  "priority": "critical",
  "timeout": "10s",
  "correlation_id": "emergency-spike-001",
  "metadata": {
    "alert_id": "ALERT-12345",
    "trigger": "cpu_utilization_high",
    "threshold_exceeded": "85%"
  }
}
```

## Complex O-RAN Deployment

### O-RAN CNF Scaling Scenario

This example demonstrates scaling O-RAN network functions based on KPI metrics.

**Intent File**: `intent-oran-cnf-scale.json`
```json
{
  "intent_type": "scaling",
  "target": "oran-du-simulator",
  "namespace": "oran-ran",
  "replicas": 8,
  "reason": "High UE connection rate detected",
  "source": "planner",
  "priority": "high",
  "correlation_id": "oran-scale-2024-08-15-001",
  "metadata": {
    "oran_component": "DU",
    "cell_id": "001-002-003",
    "kpi_trigger": "ue_connection_rate",
    "current_value": "450/min",
    "threshold": "400/min",
    "expected_improvement": "30%"
  }
}
```

**Processing Script**:
```bash
#!/bin/bash
# oran-scaling-example.sh

set -euo pipefail

HANDOFF_DIR="/opt/conductor-loop/data/handoff"
INTENT_FILE="intent-oran-cnf-scale.json"

echo "=== O-RAN CNF Scaling Example ==="

# 1. Create intent file
cat > "${HANDOFF_DIR}/${INTENT_FILE}" << 'EOF'
{
  "intent_type": "scaling",
  "target": "oran-du-simulator",
  "namespace": "oran-ran",
  "replicas": 8,
  "reason": "High UE connection rate detected",
  "source": "planner",
  "priority": "high",
  "correlation_id": "oran-scale-2024-08-15-001",
  "metadata": {
    "oran_component": "DU",
    "cell_id": "001-002-003",
    "kpi_trigger": "ue_connection_rate",
    "current_value": "450/min",
    "threshold": "400/min",
    "expected_improvement": "30%"
  }
}
EOF

echo "✓ Created intent file: ${INTENT_FILE}"

# 2. Monitor processing
echo "Monitoring processing..."
timeout 60 bash -c "
while [ ! -f '${HANDOFF_DIR}/status/${INTENT_FILE}.status' ]; do
  echo -n '.'
  sleep 1
done
echo
"

# 3. Check result
if [ -f "${HANDOFF_DIR}/status/${INTENT_FILE}.status" ]; then
  echo "✓ Processing completed"
  echo "Status:"
  cat "${HANDOFF_DIR}/status/${INTENT_FILE}.status" | jq .
  
  # 4. Verify scaling result
  if command -v kubectl >/dev/null; then
    echo "Verifying deployment scaling..."
    kubectl get deployment oran-du-simulator -n oran-ran -o jsonpath='{.spec.replicas}' || true
  fi
else
  echo "✗ Processing timeout"
  exit 1
fi

echo "=== O-RAN CNF Scaling Example Complete ==="
```

### Multi-Component O-RAN Scaling

**Batch Intent**: Multiple related components scaled together

```bash
#!/bin/bash
# oran-multi-component-scale.sh

HANDOFF_DIR="/opt/conductor-loop/data/handoff"
CORRELATION_ID="oran-multi-scale-$(date +%s)"

# DU Scaling
cat > "${HANDOFF_DIR}/intent-oran-du-scale.json" << EOF
{
  "intent_type": "scaling",
  "target": "oran-du-simulator",
  "namespace": "oran-ran",
  "replicas": 4,
  "reason": "Multi-component scaling for load balancing",
  "source": "planner",
  "correlation_id": "${CORRELATION_ID}",
  "metadata": {
    "batch_operation": true,
    "component_group": "oran-ran-functions"
  }
}
EOF

# CU Scaling
cat > "${HANDOFF_DIR}/intent-oran-cu-scale.json" << EOF
{
  "intent_type": "scaling",
  "target": "oran-cu-simulator",
  "namespace": "oran-ran",
  "replicas": 2,
  "reason": "Multi-component scaling for load balancing",
  "source": "planner",
  "correlation_id": "${CORRELATION_ID}",
  "metadata": {
    "batch_operation": true,
    "component_group": "oran-ran-functions"
  }
}
EOF

echo "Created batch scaling intents with correlation ID: ${CORRELATION_ID}"
```

## CI/CD Integration

### GitHub Actions Integration

**.github/workflows/deploy-scaling.yml**:
```yaml
name: Deploy Scaling Intent

on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Target environment'
        required: true
        default: 'staging'
        type: choice
        options:
        - staging
        - production
      replicas:
        description: 'Number of replicas'
        required: true
        default: '3'
        type: string
      reason:
        description: 'Reason for scaling'
        required: true
        type: string

jobs:
  deploy-intent:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      
    - name: Setup environment
      run: |
        echo "CORRELATION_ID=ci-deploy-$(date +%s)" >> $GITHUB_ENV
        echo "INTENT_FILE=intent-ci-deploy-${{ github.run_number }}.json" >> $GITHUB_ENV
        
    - name: Create scaling intent
      run: |
        cat > $INTENT_FILE << EOF
        {
          "intent_type": "scaling",
          "target": "web-app",
          "namespace": "${{ github.event.inputs.environment }}",
          "replicas": ${{ github.event.inputs.replicas }},
          "reason": "${{ github.event.inputs.reason }}",
          "source": "automation",
          "priority": "normal",
          "correlation_id": "$CORRELATION_ID",
          "metadata": {
            "ci_cd": true,
            "workflow_run": "${{ github.run_number }}",
            "triggered_by": "${{ github.actor }}",
            "repository": "${{ github.repository }}"
          }
        }
        EOF
        
    - name: Validate intent
      run: |
        # Install ajv-cli for validation
        npm install -g ajv-cli
        
        # Validate against schema
        ajv validate -s docs/contracts/intent.schema.json -d $INTENT_FILE
        
    - name: Deploy intent to conductor-loop
      run: |
        # Copy to conductor-loop handoff directory
        # This could be via kubectl cp, scp, or API call
        kubectl cp $INTENT_FILE conductor-loop/conductor-loop-0:/data/handoff/
        
    - name: Wait for processing
      run: |
        echo "Waiting for intent processing..."
        timeout 120 bash -c "
          while ! kubectl exec -n conductor-loop conductor-loop-0 -- \
            test -f /data/handoff/status/${INTENT_FILE}.status; do
            echo 'Waiting for processing...'
            sleep 5
          done
        "
        
    - name: Check processing result
      run: |
        # Get status file
        kubectl cp conductor-loop/conductor-loop-0:/data/handoff/status/${INTENT_FILE}.status status.json
        
        # Parse result
        STATUS=$(cat status.json | jq -r '.status')
        MESSAGE=$(cat status.json | jq -r '.message')
        
        echo "Processing status: $STATUS"
        echo "Message: $MESSAGE"
        
        if [ "$STATUS" != "success" ]; then
          echo "Intent processing failed"
          cat status.json | jq .
          exit 1
        fi
        
    - name: Verify deployment
      run: |
        # Verify the actual scaling occurred
        ACTUAL_REPLICAS=$(kubectl get deployment web-app \
          -n ${{ github.event.inputs.environment }} \
          -o jsonpath='{.spec.replicas}')
          
        if [ "$ACTUAL_REPLICAS" != "${{ github.event.inputs.replicas }}" ]; then
          echo "Scaling verification failed: expected ${{ github.event.inputs.replicas }}, got $ACTUAL_REPLICAS"
          exit 1
        fi
        
        echo "✓ Scaling verified: $ACTUAL_REPLICAS replicas"
```

### Jenkins Pipeline Integration

**Jenkinsfile**:
```groovy
pipeline {
    agent any
    
    parameters {
        choice(
            name: 'ENVIRONMENT',
            choices: ['staging', 'production'],
            description: 'Target environment'
        )
        string(
            name: 'REPLICAS',
            defaultValue: '3',
            description: 'Number of replicas'
        )
        string(
            name: 'REASON',
            defaultValue: 'CI/CD deployment',
            description: 'Reason for scaling'
        )
    }
    
    environment {
        CORRELATION_ID = "jenkins-${env.BUILD_NUMBER}-${env.BUILD_TIMESTAMP}"
        INTENT_FILE = "intent-jenkins-${env.BUILD_NUMBER}.json"
    }
    
    stages {
        stage('Create Intent') {
            steps {
                script {
                    def intent = [
                        intent_type: "scaling",
                        target: "web-app",
                        namespace: params.ENVIRONMENT,
                        replicas: params.REPLICAS.toInteger(),
                        reason: params.REASON,
                        source: "automation",
                        correlation_id: env.CORRELATION_ID,
                        metadata: [
                            ci_cd: true,
                            jenkins_job: env.JOB_NAME,
                            build_number: env.BUILD_NUMBER,
                            triggered_by: env.BUILD_USER
                        ]
                    ]
                    
                    writeJSON file: env.INTENT_FILE, json: intent
                }
            }
        }
        
        stage('Validate Intent') {
            steps {
                sh '''
                    # Validate JSON syntax
                    python3 -m json.tool $INTENT_FILE > /dev/null
                    
                    # Additional validation could be added here
                    echo "Intent validation passed"
                '''
            }
        }
        
        stage('Deploy Intent') {
            steps {
                sh '''
                    # Deploy to conductor-loop
                    kubectl cp $INTENT_FILE conductor-loop/conductor-loop-0:/data/handoff/
                    echo "Intent deployed: $INTENT_FILE"
                '''
            }
        }
        
        stage('Monitor Processing') {
            steps {
                timeout(time: 5, unit: 'MINUTES') {
                    sh '''
                        while ! kubectl exec -n conductor-loop conductor-loop-0 -- \
                            test -f /data/handoff/status/${INTENT_FILE}.status; do
                            echo "Waiting for processing..."
                            sleep 10
                        done
                        
                        echo "Processing completed"
                    '''
                }
            }
        }
        
        stage('Verify Result') {
            steps {
                script {
                    sh '''
                        # Get processing result
                        kubectl cp conductor-loop/conductor-loop-0:/data/handoff/status/${INTENT_FILE}.status result.json
                        
                        # Check status
                        STATUS=$(cat result.json | jq -r '.status')
                        
                        if [ "$STATUS" = "success" ]; then
                            echo "✓ Intent processed successfully"
                        else
                            echo "✗ Intent processing failed"
                            cat result.json
                            exit 1
                        fi
                    '''
                }
            }
        }
    }
    
    post {
        always {
            archiveArtifacts artifacts: '*.json', allowEmptyArchive: true
        }
        success {
            echo "Scaling deployment completed successfully"
        }
        failure {
            echo "Scaling deployment failed"
        }
    }
}
```

## Windows Deployment

### Windows Service Installation

**install-service.ps1**:
```powershell
# PowerShell script to install conductor-loop as Windows service

param(
    [string]$ServiceName = "ConductorLoop",
    [string]$InstallPath = "C:\Program Files\ConductorLoop",
    [string]$DataPath = "C:\ProgramData\ConductorLoop",
    [string]$PorchPath = "C:\Tools\porch.exe"
)

# Check if running as administrator
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script must be run as Administrator"
    exit 1
}

Write-Host "Installing Conductor Loop Windows Service..."

# Create directories
$Directories = @($InstallPath, $DataPath, "$DataPath\handoff", "$DataPath\logs")
foreach ($Dir in $Directories) {
    if (!(Test-Path $Dir)) {
        New-Item -ItemType Directory -Path $Dir -Force
        Write-Host "Created directory: $Dir"
    }
}

# Copy executable
Copy-Item "conductor-loop.exe" "$InstallPath\conductor-loop.exe" -Force
Write-Host "Copied executable to: $InstallPath"

# Create configuration file
$Config = @{
    handoff_dir = "$DataPath\handoff"
    porch_path = $PorchPath
    mode = "structured"
    out_dir = "$DataPath\output"
    debounce_duration = "500ms"
    max_workers = 4
    log_level = "info"
    log_format = "json"
} | ConvertTo-Json -Depth 10

$Config | Out-File -FilePath "$InstallPath\config.json" -Encoding UTF8
Write-Host "Created configuration file"

# Create service
$ServiceArgs = @(
    "--config=`"$InstallPath\config.json`"",
    "--handoff=`"$DataPath\handoff`"",
    "--out=`"$DataPath\output`""
)

$ServiceArgsString = $ServiceArgs -join " "

New-Service -Name $ServiceName `
           -BinaryPathName "`"$InstallPath\conductor-loop.exe`" $ServiceArgsString" `
           -DisplayName "Conductor Loop Service" `
           -Description "Nephoran Intent Operator Conductor Loop" `
           -StartupType Automatic

Write-Host "Service created: $ServiceName"

# Set service to restart on failure
sc.exe failure $ServiceName reset=86400 actions=restart/5000/restart/10000/restart/30000

# Start service
Start-Service -Name $ServiceName
Write-Host "Service started"

# Verify service status
$Service = Get-Service -Name $ServiceName
Write-Host "Service Status: $($Service.Status)"

Write-Host "Installation completed successfully!"
Write-Host "Data directory: $DataPath"
Write-Host "Configuration: $InstallPath\config.json"
Write-Host "To manage service: Services.msc or 'sc' command"
```

### Windows Configuration Example

**config-windows.json**:
```json
{
  "handoff_dir": "C:\\ProgramData\\ConductorLoop\\handoff",
  "porch_path": "C:\\Tools\\porch.exe",
  "mode": "structured",
  "out_dir": "C:\\ProgramData\\ConductorLoop\\output",
  "debounce_duration": "1s",
  "max_workers": 2,
  "cleanup_after": "168h",
  "log_level": "info",
  "log_format": "json",
  "metrics": {
    "enabled": true,
    "port": 9090
  },
  "health": {
    "enabled": true,
    "port": 8080
  }
}
```

### Windows PowerShell Usage Examples

**deploy-intent.ps1**:
```powershell
# PowerShell script to deploy scaling intent

param(
    [Parameter(Mandatory=$true)]
    [string]$Target,
    
    [Parameter(Mandatory=$true)]
    [string]$Namespace,
    
    [Parameter(Mandatory=$true)]
    [int]$Replicas,
    
    [string]$Reason = "PowerShell deployment",
    [string]$HandoffDir = "C:\ProgramData\ConductorLoop\handoff"
)

# Create intent object
$Intent = @{
    intent_type = "scaling"
    target = $Target
    namespace = $Namespace
    replicas = $Replicas
    reason = $Reason
    source = "user"
    correlation_id = "ps-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
}

# Generate filename
$FileName = "intent-ps-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
$FilePath = Join-Path $HandoffDir $FileName

# Convert to JSON and save
$Intent | ConvertTo-Json -Depth 10 | Out-File -FilePath $FilePath -Encoding UTF8

Write-Host "Intent deployed: $FilePath"
Write-Host "Correlation ID: $($Intent.correlation_id)"

# Monitor for completion
$StatusFile = Join-Path $HandoffDir "status\$FileName.status"
$Timeout = 60
$Elapsed = 0

Write-Host "Monitoring processing..."
while (!(Test-Path $StatusFile) -and $Elapsed -lt $Timeout) {
    Start-Sleep -Seconds 2
    $Elapsed += 2
    Write-Host "." -NoNewline
}

Write-Host ""

if (Test-Path $StatusFile) {
    $Status = Get-Content $StatusFile | ConvertFrom-Json
    Write-Host "Processing completed!"
    Write-Host "Status: $($Status.status)"
    Write-Host "Message: $($Status.message)"
    
    if ($Status.status -eq "success") {
        Write-Host "✓ Scaling successful" -ForegroundColor Green
    } else {
        Write-Host "✗ Scaling failed" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "✗ Processing timeout" -ForegroundColor Red
    exit 1
}
```

## Kubernetes Deployment

### Complete Kubernetes Example

**conductor-loop-complete.yaml**:
```yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: conductor-loop
  labels:
    name: conductor-loop
    istio-injection: enabled

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: conductor-loop
  namespace: conductor-loop

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: conductor-loop-reader
rules:
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: conductor-loop-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: conductor-loop-reader
subjects:
- kind: ServiceAccount
  name: conductor-loop
  namespace: conductor-loop

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: conductor-loop-config
  namespace: conductor-loop
data:
  config.json: |
    {
      "handoff_dir": "/data/handoff",
      "porch_path": "/usr/local/bin/porch",
      "mode": "structured",
      "out_dir": "/data/output",
      "debounce_duration": "200ms",
      "max_workers": 4,
      "cleanup_after": "72h",
      "log_level": "info",
      "log_format": "json",
      "metrics": {
        "enabled": true,
        "port": 9090,
        "path": "/metrics"
      },
      "health": {
        "enabled": true,
        "port": 8080
      },
      "processing": {
        "timeout": "30s",
        "retry_attempts": 3,
        "priority_queues": true
      }
    }

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: conductor-loop-data
  namespace: conductor-loop
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: nfs-client
  resources:
    requests:
      storage: 10Gi

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: conductor-loop
  namespace: conductor-loop
  labels:
    app.kubernetes.io/name: conductor-loop
    app.kubernetes.io/version: "1.0.0"
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app.kubernetes.io/name: conductor-loop
  template:
    metadata:
      labels:
        app.kubernetes.io/name: conductor-loop
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: conductor-loop
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: conductor-loop
        image: ghcr.io/thc1006/conductor-loop:latest
        imagePullPolicy: IfNotPresent
        args:
          - "--config=/config/config.json"
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: CONDUCTOR_LOG_LEVEL
          value: "info"
        ports:
        - name: http-health
          containerPort: 8080
        - name: http-metrics
          containerPort: 9090
        resources:
          limits:
            cpu: "1000m"
            memory: "1Gi"
          requests:
            cpu: "200m"
            memory: "256Mi"
        volumeMounts:
        - name: config-volume
          mountPath: /config
          readOnly: true
        - name: data-volume
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /livez
            port: http-health
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: http-health
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
      volumes:
      - name: config-volume
        configMap:
          name: conductor-loop-config
      - name: data-volume
        persistentVolumeClaim:
          claimName: conductor-loop-data

---
apiVersion: v1
kind: Service
metadata:
  name: conductor-loop
  namespace: conductor-loop
  labels:
    app.kubernetes.io/name: conductor-loop
spec:
  type: ClusterIP
  ports:
  - name: http-health
    port: 8080
    targetPort: http-health
  - name: http-metrics
    port: 9090
    targetPort: http-metrics
  selector:
    app.kubernetes.io/name: conductor-loop

---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: conductor-loop-hpa
  namespace: conductor-loop
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: conductor-loop
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Kubernetes Usage Examples

**kubectl-deploy-intent.sh**:
```bash
#!/bin/bash
# Deploy intent via kubectl

set -euo pipefail

TARGET="${1:-web-app}"
NAMESPACE="${2:-default}"
REPLICAS="${3:-3}"
REASON="${4:-kubectl deployment}"

CORRELATION_ID="kubectl-$(date +%s)"
INTENT_FILE="intent-kubectl-$(date +%s).json"

# Create intent
cat > "/tmp/${INTENT_FILE}" << EOF
{
  "intent_type": "scaling",
  "target": "${TARGET}",
  "namespace": "${NAMESPACE}",
  "replicas": ${REPLICAS},
  "reason": "${REASON}",
  "source": "user",
  "correlation_id": "${CORRELATION_ID}"
}
EOF

echo "Created intent: ${INTENT_FILE}"

# Deploy to conductor-loop
kubectl cp "/tmp/${INTENT_FILE}" conductor-loop/conductor-loop-0:/data/handoff/

echo "Intent deployed to conductor-loop"

# Monitor processing
echo "Monitoring processing..."
timeout 60 bash -c "
while ! kubectl exec -n conductor-loop conductor-loop-0 -- \
    test -f /data/handoff/status/${INTENT_FILE}.status; do
    echo -n '.'
    sleep 2
done
echo
"

# Get result
kubectl exec -n conductor-loop conductor-loop-0 -- \
    cat "/data/handoff/status/${INTENT_FILE}.status" | jq .

echo "Processing completed"
```

## Docker Compose Setup

### Complete Docker Compose Example

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  conductor-loop:
    image: ghcr.io/thc1006/conductor-loop:latest
    container_name: conductor-loop
    restart: unless-stopped
    
    environment:
      - CONDUCTOR_LOG_LEVEL=info
      - CONDUCTOR_METRICS_ENABLED=true
      - CONDUCTOR_HEALTH_ENABLED=true
    
    command: [
      "--handoff=/data/handoff",
      "--out=/data/output",
      "--porch=/usr/local/bin/porch",
      "--mode=structured",
      "--debounce=200ms",
      "--max-workers=4"
    ]
    
    ports:
      - "8080:8080"  # Health checks
      - "9090:9090"  # Metrics
    
    volumes:
      - conductor_data:/data
      - conductor_logs:/logs
      - ./config:/config:ro
    
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.conductor-loop.rule=Host(`conductor-loop.local`)"
      - "traefik.http.services.conductor-loop.loadbalancer.server.port=8080"
    
    networks:
      - conductor-network
  
  # Mock Porch service for testing
  porch-mock:
    image: alpine:latest
    container_name: porch-mock
    restart: unless-stopped
    
    command: >
      sh -c "
        apk add --no-cache curl jq &&
        mkdir -p /usr/local/bin &&
        cat > /usr/local/bin/porch << 'EOF' &&
        #!/bin/sh
        echo \"Mock porch execution: \$*\"
        echo \"Intent processed successfully\"
        exit 0
        EOF
        chmod +x /usr/local/bin/porch &&
        tail -f /dev/null
      "
    
    volumes:
      - porch_bin:/usr/local/bin
    
    networks:
      - conductor-network
  
  # Prometheus for metrics collection
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    restart: unless-stopped
    
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'
    
    ports:
      - "9091:9090"
    
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    
    networks:
      - conductor-network
  
  # Grafana for visualization
  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    restart: unless-stopped
    
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_INSTALL_PLUGINS=grafana-piechart-panel
    
    ports:
      - "3000:3000"
    
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./grafana/datasources:/etc/grafana/provisioning/datasources:ro
    
    networks:
      - conductor-network

volumes:
  conductor_data:
    driver: local
  conductor_logs:
    driver: local
  porch_bin:
    driver: local
  prometheus_data:
    driver: local
  grafana_data:
    driver: local

networks:
  conductor-network:
    driver: bridge
```

**prometheus.yml**:
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'conductor-loop'
    static_configs:
      - targets: ['conductor-loop:9090']
    scrape_interval: 10s
    metrics_path: /metrics
    
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
```

**Usage Commands**:
```bash
# Start the complete stack
docker-compose up -d

# View logs
docker-compose logs -f conductor-loop

# Deploy an intent
echo '{
  "intent_type": "scaling",
  "target": "test-app",
  "namespace": "default",
  "replicas": 5,
  "reason": "Docker Compose test",
  "source": "user"
}' > /tmp/intent-test.json

docker cp /tmp/intent-test.json conductor-loop:/data/handoff/

# Check status
docker exec conductor-loop ls -la /data/handoff/status/

# View metrics
curl http://localhost:9090/metrics

# Access Grafana
# http://localhost:3000 (admin/admin)

# Clean up
docker-compose down -v
```

## Monitoring Integration

### Prometheus + Grafana Setup

**grafana-dashboard.json**:
```json
{
  "dashboard": {
    "id": null,
    "title": "Conductor Loop Dashboard",
    "tags": ["conductor-loop", "nephoran"],
    "timezone": "browser",
    "refresh": "30s",
    "panels": [
      {
        "id": 1,
        "title": "File Processing Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(conductor_files_processed_total[5m])",
            "legendFormat": "{{status}} files/sec"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "short",
            "min": 0
          }
        }
      },
      {
        "id": 2,
        "title": "Processing Duration",
        "type": "heatmap",
        "targets": [
          {
            "expr": "conductor_files_processing_duration_seconds_bucket",
            "format": "heatmap",
            "legendFormat": "{{le}}"
          }
        ]
      },
      {
        "id": 3,
        "title": "Queue Status",
        "type": "graph",
        "targets": [
          {
            "expr": "conductor_queue_size",
            "legendFormat": "Queue Size"
          },
          {
            "expr": "conductor_active_workers",
            "legendFormat": "Active Workers"
          },
          {
            "expr": "conductor_max_workers",
            "legendFormat": "Max Workers"
          }
        ]
      },
      {
        "id": 4,
        "title": "Success Rate",
        "type": "singlestat",
        "targets": [
          {
            "expr": "rate(conductor_files_processed_total{status=\"success\"}[5m]) / rate(conductor_files_processed_total[5m]) * 100",
            "legendFormat": "Success Rate %"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percent",
            "min": 0,
            "max": 100,
            "thresholds": {
              "steps": [
                {"color": "red", "value": 0},
                {"color": "yellow", "value": 90},
                {"color": "green", "value": 95}
              ]
            }
          }
        }
      }
    ],
    "time": {
      "from": "now-1h",
      "to": "now"
    }
  }
}
```

### ELK Stack Integration

**filebeat.yml**:
```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /opt/conductor-loop/logs/*.log
  fields:
    service: conductor-loop
    environment: production
  fields_under_root: true
  json.keys_under_root: true
  json.add_error_key: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "conductor-loop-%{+yyyy.MM.dd}"

setup.template.name: "conductor-loop"
setup.template.pattern: "conductor-loop-*"
```

**logstash.conf**:
```ruby
input {
  beats {
    port => 5044
  }
}

filter {
  if [service] == "conductor-loop" {
    if [message] =~ /^\{/ {
      json {
        source => "message"
      }
    }
    
    # Parse correlation IDs
    if [correlation_id] {
      mutate {
        add_tag => ["correlated"]
      }
    }
    
    # Parse processing duration
    if [processing_duration] {
      ruby {
        code => "
          duration = event.get('processing_duration')
          if duration =~ /^(\d+\.?\d*)(s|ms|m)$/
            value = $1.to_f
            unit = $2
            case unit
            when 'ms'
              event.set('processing_duration_seconds', value / 1000.0)
            when 'm'
              event.set('processing_duration_seconds', value * 60.0)
            else
              event.set('processing_duration_seconds', value)
            end
          end
        "
      }
    }
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
    index => "conductor-loop-%{+YYYY.MM.dd}"
  }
}
```

## Error Handling Examples

### Retry Logic Example

**intent-with-retry.json**:
```json
{
  "intent_type": "scaling",
  "target": "unreliable-service",
  "namespace": "test",
  "replicas": 3,
  "reason": "Testing retry logic",
  "source": "test",
  "priority": "normal",
  "timeout": "10s",
  "correlation_id": "retry-test-001",
  "metadata": {
    "max_retries": 3,
    "retry_delay": "30s",
    "failure_expected": true
  }
}
```

### Error Simulation Script

**error-simulation.sh**:
```bash
#!/bin/bash
# Script to simulate various error conditions

HANDOFF_DIR="/opt/conductor-loop/data/handoff"

echo "=== Error Handling Simulation ==="

# 1. Invalid JSON format
echo "1. Testing invalid JSON..."
cat > "${HANDOFF_DIR}/intent-invalid-json.json" << 'EOF'
{
  "intent_type": "scaling",
  "target": "test-service"
  "replicas": 3,  // Invalid JSON comment
  "namespace": "test"
EOF

# 2. Missing required fields
echo "2. Testing missing required fields..."
cat > "${HANDOFF_DIR}/intent-missing-fields.json" << 'EOF'
{
  "intent_type": "scaling",
  "target": "test-service"
}
EOF

# 3. Invalid replica count
echo "3. Testing invalid replica count..."
cat > "${HANDOFF_DIR}/intent-invalid-replicas.json" << 'EOF'
{
  "intent_type": "scaling",
  "target": "test-service",
  "namespace": "test",
  "replicas": -5,
  "reason": "Testing negative replicas"
}
EOF

# 4. Non-existent deployment
echo "4. Testing non-existent deployment..."
cat > "${HANDOFF_DIR}/intent-nonexistent-target.json" << 'EOF'
{
  "intent_type": "scaling",
  "target": "does-not-exist",
  "namespace": "test",
  "replicas": 3,
  "reason": "Testing non-existent target"
}
EOF

# Monitor results
echo "Monitoring error handling..."
sleep 30

echo "=== Results ==="
for status_file in "${HANDOFF_DIR}/status"/intent-*.json.status; do
  if [ -f "$status_file" ]; then
    echo "File: $(basename "$status_file")"
    jq -r '.status' "$status_file"
    jq -r '.message' "$status_file"
    echo "---"
  fi
done
```

### Recovery Procedures

**recovery-example.sh**:
```bash
#!/bin/bash
# Example recovery procedures

set -euo pipefail

HANDOFF_DIR="/opt/conductor-loop/data/handoff"
BACKUP_DIR="/backup/conductor-loop"

echo "=== Conductor Loop Recovery Example ==="

# 1. Check service status
echo "1. Checking service status..."
if systemctl is-active --quiet conductor-loop; then
    echo "✓ Service is running"
else
    echo "✗ Service is not running"
    echo "Attempting to start service..."
    systemctl start conductor-loop
    sleep 5
    
    if systemctl is-active --quiet conductor-loop; then
        echo "✓ Service started successfully"
    else
        echo "✗ Failed to start service"
        echo "Checking logs..."
        journalctl -u conductor-loop --lines=20
        exit 1
    fi
fi

# 2. Check file system health
echo "2. Checking file system health..."
if [ -w "$HANDOFF_DIR" ]; then
    echo "✓ Handoff directory is writable"
else
    echo "✗ Handoff directory is not writable"
    echo "Fixing permissions..."
    sudo chown -R conductor-loop:conductor-loop "$HANDOFF_DIR"
    sudo chmod -R 755 "$HANDOFF_DIR"
fi

# 3. Validate state file
echo "3. Validating state file..."
STATE_FILE="${HANDOFF_DIR}/.conductor-state.json"
if [ -f "$STATE_FILE" ]; then
    if jq empty "$STATE_FILE" 2>/dev/null; then
        echo "✓ State file is valid"
    else
        echo "✗ State file is corrupted"
        echo "Backing up corrupted state file..."
        cp "$STATE_FILE" "${BACKUP_DIR}/corrupted-state-$(date +%s).json"
        
        echo "Restoring from latest backup..."
        LATEST_BACKUP=$(ls -t "${BACKUP_DIR}"/conductor-loop_*.tar.gz | head -1)
        if [ -n "$LATEST_BACKUP" ]; then
            tar -xzf "$LATEST_BACKUP" -O .conductor-state.json > "$STATE_FILE"
            echo "✓ State file restored"
        else
            echo "No backup found, creating empty state file..."
            echo '{"version":"1.0","states":{}}' > "$STATE_FILE"
        fi
    fi
else
    echo "State file not found, creating new one..."
    echo '{"version":"1.0","states":{}}' > "$STATE_FILE"
fi

# 4. Check processing backlog
echo "4. Checking processing backlog..."
PENDING_COUNT=$(find "$HANDOFF_DIR" -name "intent-*.json" | wc -l)
echo "Pending intents: $PENDING_COUNT"

if [ "$PENDING_COUNT" -gt 0 ]; then
    echo "Processing pending intents..."
    # Optionally trigger once mode to clear backlog
    # conductor-loop --once --handoff="$HANDOFF_DIR"
fi

# 5. Verify recovery
echo "5. Verifying recovery..."
sleep 10

# Check health endpoint if available
if curl -f http://localhost:8080/healthz >/dev/null 2>&1; then
    echo "✓ Health check passed"
else
    echo "Health check not available or failed"
fi

# Check recent logs for errors
RECENT_ERRORS=$(journalctl -u conductor-loop --since "5 minutes ago" --priority=err | wc -l)
if [ "$RECENT_ERRORS" -eq 0 ]; then
    echo "✓ No recent errors in logs"
else
    echo "⚠ $RECENT_ERRORS recent errors found in logs"
    journalctl -u conductor-loop --since "5 minutes ago" --priority=err
fi

echo "=== Recovery completed ==="
```

This comprehensive examples document provides practical usage scenarios, deployment configurations, and real-world integration patterns for the conductor-loop component across different platforms and environments.