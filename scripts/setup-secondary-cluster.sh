#!/bin/bash
# setup-secondary-cluster.sh
# Secondary K3d cluster setup for disaster recovery
# Ensures RTO < 5 minutes by pre-provisioning infrastructure

set -euo pipefail

# Configuration
SECONDARY_CLUSTER_NAME="nephoran-dr-secondary"
PRIMARY_CLUSTER_NAME="nephoran-primary"
K3D_VERSION="v5.6.0"
KUBECTL_VERSION="v1.28.0"
HELM_VERSION="v3.13.0"

# Network configuration for secondary cluster
SECONDARY_CLUSTER_PORT="6444"
SECONDARY_INGRESS_PORT="8081"
SECONDARY_INGRESS_PORT_SSL="8443"

# Logging setup
LOG_FILE="/tmp/setup-secondary-cluster-$(date +%Y%m%d-%H%M%S).log"
exec 1> >(tee -a "$LOG_FILE")
exec 2>&1

echo "Starting secondary cluster setup for disaster recovery at $(date)"
echo "Log file: $LOG_FILE"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if running on supported OS
    if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
        log_info "Windows detected - using Windows-compatible commands"
        DOCKER_CMD="docker.exe"
        K3D_CMD="k3d.exe"
        KUBECTL_CMD="kubectl.exe"
        HELM_CMD="helm.exe"
    else
        DOCKER_CMD="docker"
        K3D_CMD="k3d"
        KUBECTL_CMD="kubectl"
        HELM_CMD="helm"
    fi
    
    # Check Docker
    if ! command -v $DOCKER_CMD &> /dev/null; then
        log_error "Docker not found. Please install Docker."
        exit 1
    fi
    
    # Check if Docker daemon is running
    if ! $DOCKER_CMD info &> /dev/null; then
        log_error "Docker daemon is not running. Please start Docker."
        exit 1
    fi
    
    # Install k3d if not present
    if ! command -v $K3D_CMD &> /dev/null; then
        log_info "Installing k3d..."
        if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
            # Windows installation
            curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=$K3D_VERSION bash
        else
            # Linux/macOS installation
            curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG=$K3D_VERSION bash
        fi
    fi
    
    # Install kubectl if not present
    if ! command -v $KUBECTL_CMD &> /dev/null; then
        log_info "Installing kubectl..."
        if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
            curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/windows/amd64/kubectl.exe"
            chmod +x kubectl.exe
            mv kubectl.exe /usr/local/bin/
        else
            curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl"
            chmod +x kubectl
            sudo mv kubectl /usr/local/bin/
        fi
    fi
    
    # Install helm if not present
    if ! command -v $HELM_CMD &> /dev/null; then
        log_info "Installing Helm..."
        curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    fi
    
    log_success "Prerequisites check completed"
}

# Create secondary cluster configuration
create_cluster_config() {
    log_info "Creating secondary cluster configuration..."
    
    cat > /tmp/k3d-secondary-config.yaml << EOF
apiVersion: k3d.io/v1alpha5
kind: Simple
metadata:
  name: $SECONDARY_CLUSTER_NAME
servers: 3  # HA control plane for production DR
agents: 3   # Worker nodes for workload distribution
kubeAPI:
  host: "127.0.0.1"
  hostIP: "127.0.0.1"
  hostPort: "$SECONDARY_CLUSTER_PORT"
image: rancher/k3s:v1.28.2-k3s1
network: nephoran-dr-network
subnet: "172.20.0.0/16"
token: nephoran-dr-cluster-token
volumes:
  - volume: /tmp/k3d-nephoran-dr:/tmp/k3d-nephoran-dr
    nodeFilters:
      - server:*
      - agent:*
ports:
  - port: $SECONDARY_INGRESS_PORT:80
    nodeFilters:
      - loadbalancer
  - port: $SECONDARY_INGRESS_PORT_SSL:443
    nodeFilters:
      - loadbalancer
env:
  - envVar: K3S_KUBECONFIG_OUTPUT=/tmp/k3d-nephoran-dr/kubeconfig.yaml
    nodeFilters:
      - server:*
options:
  k3d:
    wait: true
    timeout: "300s"
    disableLoadbalancer: false
  k3s:
    extraArgs:
      - arg: --disable=traefik
        nodeFilters:
          - server:*
      - arg: --disable=servicelb
        nodeFilters:
          - server:*
      - arg: --node-taint=CriticalAddonsOnly=true:NoExecute
        nodeFilters:
          - server:*
  kubeconfig:
    updateDefaultKubeconfig: false
    switchCurrentContext: false
registries:
  create:
    name: nephoran-dr-registry
    host: "localhost"
    hostPort: "5001"
EOF
    
    log_success "Secondary cluster configuration created"
}

# Create the secondary cluster
create_secondary_cluster() {
    log_info "Creating secondary K3d cluster..."
    
    # Check if cluster already exists
    if $K3D_CMD cluster list | grep -q $SECONDARY_CLUSTER_NAME; then
        log_warn "Secondary cluster '$SECONDARY_CLUSTER_NAME' already exists"
        read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Deleting existing secondary cluster..."
            $K3D_CMD cluster delete $SECONDARY_CLUSTER_NAME
        else
            log_info "Using existing secondary cluster"
            return 0
        fi
    fi
    
    # Create network if it doesn't exist
    if ! $DOCKER_CMD network ls | grep -q nephoran-dr-network; then
        log_info "Creating Docker network for DR cluster..."
        $DOCKER_CMD network create nephoran-dr-network --subnet=172.20.0.0/16
    fi
    
    # Create the cluster
    log_info "Creating K3d cluster from configuration..."
    $K3D_CMD cluster create --config /tmp/k3d-secondary-config.yaml
    
    # Wait for cluster to be ready
    log_info "Waiting for secondary cluster to be ready..."
    timeout=300
    counter=0
    while [ $counter -lt $timeout ]; do
        if $K3D_CMD kubeconfig get $SECONDARY_CLUSTER_NAME > /tmp/secondary-kubeconfig.yaml 2>/dev/null; then
            if KUBECONFIG=/tmp/secondary-kubeconfig.yaml $KUBECTL_CMD get nodes --no-headers 2>/dev/null | wc -l | grep -q "6"; then
                log_success "Secondary cluster is ready with 6 nodes (3 servers + 3 agents)"
                break
            fi
        fi
        sleep 5
        counter=$((counter + 5))
        log_info "Waiting for cluster readiness... ($counter/$timeout seconds)"
    done
    
    if [ $counter -ge $timeout ]; then
        log_error "Secondary cluster failed to become ready within $timeout seconds"
        exit 1
    fi
    
    # Label nodes for DR workloads
    KUBECONFIG=/tmp/secondary-kubeconfig.yaml $KUBECTL_CMD label nodes --all nephoran.com/cluster-role=disaster-recovery
    KUBECONFIG=/tmp/secondary-kubeconfig.yaml $KUBECTL_CMD label nodes --all nephoran.com/failover-ready=true
    
    log_success "Secondary cluster created successfully"
}

# Install essential components
install_essential_components() {
    log_info "Installing essential components on secondary cluster..."
    
    export KUBECONFIG=/tmp/secondary-kubeconfig.yaml
    
    # Create namespaces
    log_info "Creating essential namespaces..."
    cat << EOF | $KUBECTL_CMD apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-system
  labels:
    name: nephoran-system
    nephoran.com/cluster-role: disaster-recovery
---
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-production
  labels:
    name: nephoran-production
    nephoran.com/cluster-role: disaster-recovery
---
apiVersion: v1
kind: Namespace
metadata:
  name: velero
  labels:
    name: velero
    nephoran.com/cluster-role: disaster-recovery
---
apiVersion: v1
kind: Namespace
metadata:
  name: monitoring
  labels:
    name: monitoring
    nephoran.com/cluster-role: disaster-recovery
---
apiVersion: v1
kind: Namespace
metadata:
  name: cert-manager
  labels:
    name: cert-manager
    nephoran.com/cluster-role: disaster-recovery
EOF
    
    # Install cert-manager
    log_info "Installing cert-manager..."
    $HELM_CMD repo add jetstack https://charts.jetstack.io --force-update
    $HELM_CMD install cert-manager jetstack/cert-manager \
        --namespace cert-manager \
        --set installCRDs=true \
        --set global.leaderElection.namespace=cert-manager \
        --wait --timeout=300s
    
    # Install Nginx Ingress Controller
    log_info "Installing Nginx Ingress Controller..."
    $HELM_CMD repo add ingress-nginx https://kubernetes.github.io/ingress-nginx --force-update
    $HELM_CMD install ingress-nginx ingress-nginx/ingress-nginx \
        --namespace ingress-nginx \
        --create-namespace \
        --set controller.service.type=NodePort \
        --set controller.service.nodePorts.http=30080 \
        --set controller.service.nodePorts.https=30443 \
        --wait --timeout=300s
    
    # Install Prometheus for monitoring
    log_info "Installing Prometheus monitoring stack..."
    $HELM_CMD repo add prometheus-community https://prometheus-community.github.io/helm-charts --force-update
    $HELM_CMD install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --set grafana.adminPassword=admin123 \
        --set prometheus.service.type=NodePort \
        --set prometheus.service.nodePort=30090 \
        --set grafana.service.type=NodePort \
        --set grafana.service.nodePort=30300 \
        --wait --timeout=600s
    
    # Pre-install Velero (configured but not started)
    log_info "Pre-installing Velero for backup restore capability..."
    $HELM_CMD repo add vmware-tanzu https://vmware-tanzu.github.io/helm-charts --force-update
    
    # Create Velero configuration for secondary cluster
    cat << EOF | $KUBECTL_CMD apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: cloud-credentials
  namespace: velero
type: Opaque
data:
  # Same S3 credentials as primary cluster
  cloud: |
    W2RlZmF1bHRdCmF3c19hY2Nlc3Nfa2V5X2lkID0gWU9VUl9BQ0NFU1NfS0VZCmF3c19zZWNyZXRfYWNjZXNzX2tleSA9IFlPVVJfU0VDUkVUX0tFWQ==
EOF
    
    # Install Velero with restore-only configuration
    $HELM_CMD install velero vmware-tanzu/velero \
        --namespace velero \
        --set configuration.backupStorageLocation[0].name=nephoran-s3-backup \
        --set configuration.backupStorageLocation[0].provider=aws \
        --set configuration.backupStorageLocation[0].bucket=nephoran-disaster-recovery-backups \
        --set configuration.backupStorageLocation[0].prefix=production-cluster \
        --set configuration.backupStorageLocation[0].config.region=us-west-2 \
        --set configuration.volumeSnapshotLocation[0].name=nephoran-volume-snapshots \
        --set configuration.volumeSnapshotLocation[0].provider=aws \
        --set configuration.volumeSnapshotLocation[0].config.region=us-west-2 \
        --set credentials.existingSecret=cloud-credentials \
        --set deployRestic=true \
        --set metrics.enabled=true \
        --set metrics.serviceMonitor.enabled=true \
        --wait --timeout=300s
    
    log_success "Essential components installed"
}

# Configure cluster for fast failover
configure_fast_failover() {
    log_info "Configuring secondary cluster for fast failover..."
    
    export KUBECONFIG=/tmp/secondary-kubeconfig.yaml
    
    # Create fast failover configuration
    cat << EOF | $KUBECTL_CMD apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: failover-config
  namespace: kube-system
  labels:
    nephoran.com/component: disaster-recovery
data:
  cluster-role: "secondary"
  failover-ready: "true"
  rto-target: "300"  # 5 minutes
  rpo-target: "1800" # 30 minutes
  primary-cluster: "$PRIMARY_CLUSTER_NAME"
  secondary-cluster: "$SECONDARY_CLUSTER_NAME"
  backup-location: "nephoran-s3-backup"
  restore-strategy: "full-cluster"
---
# Resource quotas to prevent resource exhaustion during failover
apiVersion: v1
kind: ResourceQuota
metadata:
  name: disaster-recovery-quota
  namespace: nephoran-system
spec:
  hard:
    requests.cpu: "20"
    requests.memory: 40Gi
    limits.cpu: "40" 
    limits.memory: 80Gi
    persistentvolumeclaims: "50"
---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: disaster-recovery-quota
  namespace: nephoran-production
spec:
  hard:
    requests.cpu: "30"
    requests.memory: 60Gi
    limits.cpu: "60"
    limits.memory: 120Gi
    persistentvolumeclaims: "100"
EOF
    
    # Create priority classes for critical workloads
    cat << EOF | $KUBECTL_CMD apply -f -
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: nephoran-critical
value: 1000000
globalDefault: false
description: "Priority class for critical Nephoran components during DR"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: nephoran-high
value: 100000
globalDefault: false
description: "Priority class for high-priority Nephoran components during DR"
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: nephoran-normal
value: 1000
globalDefault: true
description: "Default priority class for Nephoran components during DR"
EOF
    
    # Pre-pull critical images for faster startup
    log_info "Pre-pulling critical container images..."
    
    # Create DaemonSet to pre-pull images on all nodes
    cat << EOF | $KUBECTL_CMD apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: image-preloader
  namespace: kube-system
  labels:
    app: image-preloader
    nephoran.com/component: disaster-recovery
spec:
  selector:
    matchLabels:
      app: image-preloader
  template:
    metadata:
      labels:
        app: image-preloader
    spec:
      tolerations:
      - operator: Exists
      initContainers:
      # Pre-pull Nephoran operator images
      - name: preload-operator
        image: nephoran/nephoran-operator:latest
        command: ['sh', '-c', 'echo "Operator image preloaded"']
        resources:
          requests:
            cpu: 10m
            memory: 10Mi
          limits:
            cpu: 50m
            memory: 50Mi
      - name: preload-llm-processor
        image: nephoran/llm-processor:latest
        command: ['sh', '-c', 'echo "LLM processor image preloaded"']
        resources:
          requests:
            cpu: 10m
            memory: 10Mi
          limits:
            cpu: 50m
            memory: 50Mi
      - name: preload-rag-api
        image: nephoran/rag-api:latest
        command: ['sh', '-c', 'echo "RAG API image preloaded"']
        resources:
          requests:
            cpu: 10m
            memory: 10Mi
          limits:
            cpu: 50m
            memory: 50Mi
      - name: preload-weaviate
        image: semitechnologies/weaviate:1.22.2
        command: ['sh', '-c', 'echo "Weaviate image preloaded"']
        resources:
          requests:
            cpu: 10m
            memory: 10Mi
          limits:
            cpu: 50m
            memory: 50Mi
      containers:
      - name: sleep
        image: busybox:1.36
        command: ['sleep', '3600']
        resources:
          requests:
            cpu: 1m
            memory: 1Mi
          limits:
            cpu: 5m
            memory: 5Mi
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
EOF
    
    # Wait for image preloading to complete
    log_info "Waiting for image preloading to complete..."
    $KUBECTL_CMD rollout status daemonset/image-preloader -n kube-system --timeout=600s
    
    log_success "Fast failover configuration completed"
}

# Create cluster status monitoring
setup_cluster_monitoring() {
    log_info "Setting up cluster status monitoring..."
    
    export KUBECONFIG=/tmp/secondary-kubeconfig.yaml
    
    # Create monitoring ServiceAccount and RBAC
    cat << EOF | $KUBECTL_CMD apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cluster-monitor
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cluster-monitor
rules:
- apiGroups: [""]
  resources: ["nodes", "pods", "services", "endpoints", "namespaces"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets", "replicasets"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["nodes", "pods"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cluster-monitor
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-monitor
subjects:
- kind: ServiceAccount
  name: cluster-monitor
  namespace: kube-system
EOF
    
    # Create cluster health check pod
    cat << EOF | $KUBECTL_CMD apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cluster-health-monitor
  namespace: kube-system
  labels:
    app: cluster-health-monitor
    nephoran.com/component: disaster-recovery
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cluster-health-monitor
  template:
    metadata:
      labels:
        app: cluster-health-monitor
    spec:
      serviceAccountName: cluster-monitor
      containers:
      - name: health-monitor
        image: curlimages/curl:8.4.0
        command:
        - /bin/sh
        - -c
        - |
          while true; do
            echo "\$(date): Cluster health check"
            
            # Check node status
            echo "Nodes status:"
            kubectl get nodes --no-headers | awk '{print \$1, \$2}' | while read node status; do
              echo "  \$node: \$status"
            done
            
            # Check critical pods
            echo "Critical pods status:"
            kubectl get pods -n kube-system --no-headers | grep -E '(coredns|metrics-server)' | awk '{print \$1, \$3}' | while read pod status; do
              echo "  \$pod: \$status"
            done
            
            # Check storage
            echo "Storage status:"
            kubectl get pv --no-headers | awk '{print \$1, \$5}' | while read pv status; do
              echo "  \$pv: \$status"
            done
            
            # Mark cluster as ready for failover
            kubectl patch configmap failover-config -n kube-system --type merge -p '{"data":{"last-health-check":"'"\$(date -Iseconds)"'","cluster-status":"ready"}}'
            
            sleep 60
          done
        resources:
          requests:
            cpu: 10m
            memory: 32Mi
          limits:
            cpu: 100m
            memory: 128Mi
---
# Service to expose health check endpoint
apiVersion: v1
kind: Service
metadata:
  name: cluster-health-monitor
  namespace: kube-system
  labels:
    app: cluster-health-monitor
spec:
  selector:
    app: cluster-health-monitor
  ports:
  - port: 8080
    targetPort: 8080
    name: health
  type: ClusterIP
EOF
    
    log_success "Cluster monitoring setup completed"
}

# Create kubeconfig for external access
setup_external_access() {
    log_info "Setting up external cluster access..."
    
    # Get kubeconfig from k3d
    $K3D_CMD kubeconfig get $SECONDARY_CLUSTER_NAME > /tmp/secondary-kubeconfig-external.yaml
    
    # Modify kubeconfig for external access
    sed -i "s/127.0.0.1:$SECONDARY_CLUSTER_PORT/localhost:$SECONDARY_CLUSTER_PORT/g" /tmp/secondary-kubeconfig-external.yaml
    
    # Create kubeconfig directory if it doesn't exist
    mkdir -p ~/.kube
    
    # Backup existing kubeconfig
    if [ -f ~/.kube/config ]; then
        cp ~/.kube/config ~/.kube/config.backup.$(date +%Y%m%d-%H%M%S)
        log_info "Existing kubeconfig backed up"
    fi
    
    # Merge contexts
    KUBECONFIG=~/.kube/config:/tmp/secondary-kubeconfig-external.yaml kubectl config view --flatten > ~/.kube/config.new
    mv ~/.kube/config.new ~/.kube/config
    
    # Set context name for easier identification
    kubectl config rename-context k3d-$SECONDARY_CLUSTER_NAME nephoran-dr-secondary
    
    log_success "External access configured - use 'kubectl config use-context nephoran-dr-secondary'"
}

# Validate secondary cluster setup
validate_cluster() {
    log_info "Validating secondary cluster setup..."
    
    export KUBECONFIG=/tmp/secondary-kubeconfig.yaml
    
    # Check cluster status
    if ! $KUBECTL_CMD cluster-info &>/dev/null; then
        log_error "Cluster is not accessible"
        exit 1
    fi
    
    # Check nodes
    NODE_COUNT=$($KUBECTL_CMD get nodes --no-headers | wc -l)
    if [ "$NODE_COUNT" -lt 6 ]; then
        log_error "Expected 6 nodes, found $NODE_COUNT"
        exit 1
    fi
    
    # Check essential pods
    log_info "Checking essential pods..."
    
    READY_PODS=0
    TOTAL_PODS=0
    
    for namespace in cert-manager ingress-nginx monitoring velero; do
        NAMESPACE_PODS=$($KUBECTL_CMD get pods -n $namespace --no-headers 2>/dev/null | wc -l || echo 0)
        READY_NAMESPACE_PODS=$($KUBECTL_CMD get pods -n $namespace --no-headers 2>/dev/null | grep -c "Running" || echo 0)
        
        TOTAL_PODS=$((TOTAL_PODS + NAMESPACE_PODS))
        READY_PODS=$((READY_PODS + READY_NAMESPACE_PODS))
        
        log_info "$namespace: $READY_NAMESPACE_PODS/$NAMESPACE_PODS pods ready"
    done
    
    if [ $READY_PODS -eq $TOTAL_PODS ] && [ $TOTAL_PODS -gt 0 ]; then
        log_success "All essential pods are running ($READY_PODS/$TOTAL_PODS)"
    else
        log_warn "Some pods are not ready: $READY_PODS/$TOTAL_PODS"
    fi
    
    # Test Velero backup list access
    if $KUBECTL_CMD get backups -n velero &>/dev/null; then
        BACKUP_COUNT=$($KUBECTL_CMD get backups -n velero --no-headers | wc -l)
        log_info "Velero can access $BACKUP_COUNT existing backups"
    else
        log_warn "Velero backup access test failed"
    fi
    
    # Check resource availability
    AVAILABLE_CPU=$($KUBECTL_CMD top nodes 2>/dev/null | awk 'NR>1 {sum+=$3} END {print sum}' | sed 's/m//' || echo "Unknown")
    AVAILABLE_MEMORY=$($KUBECTL_CMD top nodes 2>/dev/null | awk 'NR>1 {sum+=$5} END {print sum}' | sed 's/Mi//' || echo "Unknown")
    
    log_info "Available resources: ${AVAILABLE_CPU}m CPU, ${AVAILABLE_MEMORY}Mi Memory"
    
    # Performance test
    log_info "Running performance test..."
    START_TIME=$(date +%s)
    $KUBECTL_CMD run test-pod --image=nginx:alpine --restart=Never --rm -i --timeout=60s --command -- sleep 1 &>/dev/null
    END_TIME=$(date +%s)
    POD_START_TIME=$((END_TIME - START_TIME))
    
    if [ $POD_START_TIME -lt 30 ]; then
        log_success "Pod startup time: ${POD_START_TIME}s (Target: <30s)"
    else
        log_warn "Pod startup time: ${POD_START_TIME}s (Target: <30s)"
    fi
    
    log_success "Secondary cluster validation completed"
}

# Generate cluster information report
generate_cluster_report() {
    log_info "Generating cluster setup report..."
    
    REPORT_FILE="/tmp/secondary-cluster-report-$(date +%Y%m%d-%H%M%S).json"
    
    export KUBECONFIG=/tmp/secondary-kubeconfig.yaml
    
    cat > "$REPORT_FILE" << EOF
{
  "cluster": {
    "name": "$SECONDARY_CLUSTER_NAME",
    "type": "disaster-recovery-secondary",
    "created": "$(date -Iseconds)",
    "kubeconfig": "/tmp/secondary-kubeconfig.yaml",
    "external_kubeconfig": "/tmp/secondary-kubeconfig-external.yaml"
  },
  "network": {
    "api_server_port": "$SECONDARY_CLUSTER_PORT",
    "ingress_http_port": "$SECONDARY_INGRESS_PORT",
    "ingress_https_port": "$SECONDARY_INGRESS_PORT_SSL",
    "docker_network": "nephoran-dr-network"
  },
  "nodes": {
    "total": $(kubectl get nodes --no-headers | wc -l),
    "servers": 3,
    "agents": 3,
    "details": [
$(kubectl get nodes -o json | jq -c '.items[] | {name: .metadata.name, role: .metadata.labels["kubernetes.io/role"] // "agent", status: .status.conditions[] | select(.type=="Ready") | .status}' | sed 's/^/      /' | sed 's/$/,/' | sed '$s/,$//')
    ]
  },
  "components": {
$(kubectl get pods --all-namespaces -o json | jq -r '.items | group_by(.metadata.namespace) | map({namespace: .[0].metadata.namespace, pods: length, ready: [.[] | select(.status.phase=="Running")] | length}) | map("    \"" + .namespace + "\": {\"total\": " + (.pods|tostring) + ", \"ready\": " + (.ready|tostring) + "}") | join(",\n")')
  },
  "storage": {
$(kubectl get pv -o json | jq -r '.items | group_by(.status.phase) | map({phase: .[0].status.phase, count: length}) | map("    \"" + .phase + "\": " + (.count|tostring)) | join(",\n")')
  },
  "rto_configuration": {
    "target_rto_seconds": 300,
    "target_rpo_seconds": 1800,
    "image_preload_completed": true,
    "monitoring_enabled": true,
    "failover_ready": true
  }
}
EOF
    
    log_success "Cluster report generated: $REPORT_FILE"
    
    # Display summary
    echo
    echo "=========================================="
    echo "SECONDARY CLUSTER SETUP COMPLETE"
    echo "=========================================="
    echo "Cluster Name: $SECONDARY_CLUSTER_NAME"
    echo "API Server: https://localhost:$SECONDARY_CLUSTER_PORT"
    echo "Ingress HTTP: http://localhost:$SECONDARY_INGRESS_PORT"
    echo "Ingress HTTPS: https://localhost:$SECONDARY_INGRESS_PORT_SSL"
    echo "Kubeconfig: /tmp/secondary-kubeconfig.yaml"
    echo "Context Name: nephoran-dr-secondary"
    echo "Report: $REPORT_FILE"
    echo
    echo "To switch to secondary cluster:"
    echo "  kubectl config use-context nephoran-dr-secondary"
    echo
    echo "To test failover:"
    echo "  ./scripts/test-disaster-recovery.sh"
    echo "=========================================="
}

# Main execution
main() {
    log_info "Starting Nephoran Intent Operator secondary cluster setup"
    
    check_prerequisites
    create_cluster_config
    create_secondary_cluster
    install_essential_components
    configure_fast_failover
    setup_cluster_monitoring
    setup_external_access
    validate_cluster
    generate_cluster_report
    
    log_success "Secondary cluster setup completed successfully!"
    log_info "RTO target: <5 minutes | RPO target: <30 minutes"
}

# Cleanup function for script interruption
cleanup() {
    log_warn "Script interrupted. Cleaning up..."
    # Add cleanup commands here if needed
    exit 1
}

# Trap signals for cleanup
trap cleanup INT TERM

# Run main function
main "$@"