# O-RAN Nephio R5 Complete Monitoring Stack Deployment
# Deploys Prometheus, Grafana, Jaeger, ELK, VES Collector, and AlertManager

param(
    [string]$KubeConfig = $env:KUBECONFIG,
    [switch]$SkipCRDs,
    [switch]$WaitForReady = $true
)

Write-Host "🚀 Deploying O-RAN Nephio R5 Complete Monitoring Stack..." -ForegroundColor Cyan

# Function to check if command exists
function Test-CommandExists {
    param($Command)
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

# Validate prerequisites
Write-Host "✅ Checking prerequisites..." -ForegroundColor Yellow
if (-not (Test-CommandExists "kubectl")) {
    throw "kubectl is not installed or not in PATH"
}
if (-not (Test-CommandExists "helm")) {
    throw "helm is not installed or not in PATH"
}

# Create monitoring namespace
Write-Host "📁 Creating monitoring namespace..." -ForegroundColor Green
kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -

# Install Helm repositories
Write-Host "📦 Adding Helm repositories..." -ForegroundColor Green
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add elastic https://helm.elastic.co
helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
helm repo update

# Install Elastic Cloud Operator (for ELK stack)
if (-not $SkipCRDs) {
    Write-Host "🔧 Installing Elastic Cloud Operator..." -ForegroundColor Green
    kubectl apply -f https://download.elastic.co/downloads/eck/2.10.0/crds.yaml
    kubectl apply -f https://download.elastic.co/downloads/eck/2.10.0/operator.yaml
    
    # Install Jaeger Operator
    Write-Host "🔧 Installing Jaeger Operator..." -ForegroundColor Green
    kubectl create namespace observability --dry-run=client -o yaml | kubectl apply -f -
    kubectl apply -f https://github.com/jaegertracing/jaeger-operator/releases/download/v1.53.0/jaeger-operator.yaml -n observability
}

# Deploy Prometheus Stack with O-RAN customizations
Write-Host "📊 Installing Prometheus Stack with O-RAN configuration..." -ForegroundColor Green
helm upgrade --install monitoring prometheus-community/kube-prometheus-stack `
    --namespace monitoring `
    --values prometheus-values.yaml `
    --set grafana.persistence.enabled=true `
    --set grafana.persistence.size=10Gi `
    --set prometheus.prometheusSpec.retention=30d `
    --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=100Gi `
    --timeout 600s

# Create ORAN namespace if it doesn't exist
kubectl create namespace oran --dry-run=client -o yaml | kubectl apply -f -

# Deploy VES Collector
Write-Host "📡 Deploying VES 7.3 Collector..." -ForegroundColor Green
kubectl apply -f ves-collector-config.yaml

# Deploy ELK Stack
Write-Host "📋 Deploying Elasticsearch and Kibana..." -ForegroundColor Green
kubectl apply -f elk-stack.yaml

# Deploy Jaeger for distributed tracing
Write-Host "🔍 Deploying Jaeger distributed tracing..." -ForegroundColor Green
kubectl apply -f jaeger-config.yaml

# Apply O-RAN specific monitoring rules and ServiceMonitors
Write-Host "📏 Applying O-RAN KPI rules and ServiceMonitors..." -ForegroundColor Green
kubectl apply -f oran-monitoring-rules.yaml
kubectl apply -f servicemonitors.yaml

# Wait for deployments to be ready
if ($WaitForReady) {
    Write-Host "⏳ Waiting for monitoring components to be ready..." -ForegroundColor Yellow
    
    # Wait for Prometheus stack
    kubectl wait --for=condition=Ready pods --all -n monitoring --timeout=600s
    
    # Wait for VES collector
    kubectl wait --for=condition=Ready pods -l app=ves-collector -n oran --timeout=300s
    
    # Wait for Elasticsearch
    kubectl wait --for=condition=Ready pods -l elasticsearch.k8s.elastic.co/cluster-name=oran-elasticsearch -n monitoring --timeout=600s
    
    # Wait for Jaeger
    kubectl wait --for=condition=Ready pods -l app.kubernetes.io/name=jaeger -n monitoring --timeout=300s
}

# Import Grafana dashboard
Write-Host "📈 Importing O-RAN SLA dashboard to Grafana..." -ForegroundColor Green
$grafanaPassword = kubectl get secret --namespace monitoring monitoring-grafana -o jsonpath="{.data.admin-password}" | ForEach-Object { [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($_)) }

# Start port-forward in background for Grafana import
$grafanaJob = Start-Job -ScriptBlock {
    kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80
}
Start-Sleep 10

try {
    # Import dashboard via REST API
    $dashboardJson = Get-Content -Raw grafana-dashboards.json
    $headers = @{
        "Content-Type" = "application/json"
        "Authorization" = "Basic " + [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes("admin:$grafanaPassword"))
    }
    
    Invoke-RestMethod -Uri "http://localhost:3000/api/dashboards/db" -Method POST -Body $dashboardJson -Headers $headers -ErrorAction SilentlyContinue
    Write-Host "✅ Dashboard imported successfully!" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Dashboard import failed, you can import manually later: $($_.Exception.Message)" -ForegroundColor Yellow
} finally {
    Stop-Job $grafanaJob -Force
    Remove-Job $grafanaJob -Force
}

# Label services for monitoring
Write-Host "🏷️  Labeling services for monitoring discovery..." -ForegroundColor Green
kubectl label svc ves-collector -n oran monitor=true --overwrite=true
# Add more service labels as services are deployed
kubectl label svc oai-cu -n oran monitor=true --overwrite=true -ErrorAction SilentlyContinue
kubectl label svc oai-du -n oran monitor=true --overwrite=true -ErrorAction SilentlyContinue
kubectl label svc ric-e2term -n ricplt monitor=true --overwrite=true -ErrorAction SilentlyContinue

# Get access information
Write-Host "`n🎯 Monitoring Stack Access Information:" -ForegroundColor Cyan
Write-Host "════════════════════════════════════════" -ForegroundColor Cyan

$grafanaPassword = kubectl get secret --namespace monitoring monitoring-grafana -o jsonpath="{.data.admin-password}" | ForEach-Object { [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($_)) }

Write-Host "📊 Prometheus:" -ForegroundColor Green
Write-Host "   kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-prometheus 9090:9090"
Write-Host "   Access: http://localhost:9090"

Write-Host "`n📈 Grafana:" -ForegroundColor Green
Write-Host "   kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80"
Write-Host "   Access: http://localhost:3000"
Write-Host "   Username: admin"
Write-Host "   Password: $grafanaPassword"

Write-Host "`n🔍 Jaeger:" -ForegroundColor Green
Write-Host "   kubectl port-forward -n monitoring svc/oran-tracing-query 16686:16686"
Write-Host "   Access: http://localhost:16686"

Write-Host "`n📋 Kibana:" -ForegroundColor Green
Write-Host "   kubectl port-forward -n monitoring svc/oran-kibana-kb-http 5601:5601"
Write-Host "   Access: http://localhost:5601"
$elasticPassword = kubectl get secret oran-elasticsearch-es-elastic-user -o jsonpath="{.data.elastic}" -n monitoring | ForEach-Object { [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($_)) } -ErrorAction SilentlyContinue
if ($elasticPassword) {
    Write-Host "   Username: elastic"
    Write-Host "   Password: $elasticPassword"
}

Write-Host "`n🚨 AlertManager:" -ForegroundColor Green
Write-Host "   kubectl port-forward -n monitoring svc/monitoring-kube-prometheus-alertmanager 9093:9093"
Write-Host "   Access: http://localhost:9093"

Write-Host "`n📡 VES Collector:" -ForegroundColor Green
Write-Host "   kubectl port-forward -n oran svc/ves-collector 8443:8443"
Write-Host "   VES Events: https://localhost:8443/eventListener/v7"
Write-Host "   Metrics: http://localhost:8080/metrics"

Write-Host "`n✅ Monitoring stack deployment completed!" -ForegroundColor Green
Write-Host "🔄 Run the following to check status:" -ForegroundColor Yellow
Write-Host "   kubectl get pods -n monitoring"
Write-Host "   kubectl get pods -n oran"
Write-Host "   kubectl get servicemonitors -n monitoring"
Write-Host "   kubectl get prometheusrules -n monitoring"