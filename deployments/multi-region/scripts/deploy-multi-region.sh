#!/bin/bash
set -euo pipefail

# Multi-Region Deployment Script for Nephoran Intent Operator
# This script orchestrates the deployment across multiple GKE regions

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ID="${PROJECT_ID:-}"
ENVIRONMENT="${ENVIRONMENT:-prod}"
TERRAFORM_DIR="../terraform"
K8S_DIR="../kubernetes"
REGIONS=("us-central1" "europe-west1" "asia-southeast1")
PRIMARY_REGION="us-central1"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validation functions
validate_prerequisites() {
    log_info "Validating prerequisites..."
    
    # Check required tools
    local required_tools=("gcloud" "kubectl" "terraform" "helm" "kustomize")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "$tool is not installed"
            exit 1
        fi
    done
    
    # Check GCP authentication
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" &> /dev/null; then
        log_error "Not authenticated to GCP. Run 'gcloud auth login'"
        exit 1
    fi
    
    # Check project ID
    if [[ -z "$PROJECT_ID" ]]; then
        PROJECT_ID=$(gcloud config get-value project)
        if [[ -z "$PROJECT_ID" ]]; then
            log_error "PROJECT_ID not set. Set it with: export PROJECT_ID=your-project-id"
            exit 1
        fi
    fi
    
    log_success "Prerequisites validated"
}

# Terraform deployment
deploy_infrastructure() {
    log_info "Deploying infrastructure with Terraform..."
    
    cd "$TERRAFORM_DIR"
    
    # Initialize Terraform
    terraform init -upgrade
    
    # Create terraform.tfvars
    cat > terraform.tfvars <<EOF
project_id = "$PROJECT_ID"
environment = "$ENVIRONMENT"
domain_name = "nephoran.${PROJECT_ID}.com"
dns_zone_name = "nephoran-zone"
enable_istio = true
enable_config_sync = true
enable_binary_authorization = true
enable_workload_identity = true
enable_network_policy = true
budget_amount = 10000
budget_alert_thresholds = [50, 80, 90, 100]
EOF
    
    # Plan deployment
    terraform plan -out=tfplan
    
    # Apply with auto-approve for CI/CD
    if [[ "${CI:-false}" == "true" ]]; then
        terraform apply -auto-approve tfplan
    else
        terraform apply tfplan
    fi
    
    log_success "Infrastructure deployed successfully"
}

# Configure kubectl contexts
configure_kubectl() {
    log_info "Configuring kubectl contexts..."
    
    for region in "${REGIONS[@]}"; do
        local cluster_name="nephoran-${region//-/_}"
        log_info "Getting credentials for $cluster_name in $region"
        
        gcloud container clusters get-credentials "$cluster_name" \
            --region="$region" \
            --project="$PROJECT_ID"
        
        # Rename context for easier management
        kubectl config rename-context \
            "gke_${PROJECT_ID}_${region}_${cluster_name}" \
            "nephoran-${region}"
    done
    
    # Set primary region as default context
    kubectl config use-context "nephoran-${PRIMARY_REGION}"
    
    log_success "kubectl contexts configured"
}

# Deploy Istio service mesh
deploy_istio() {
    log_info "Deploying Istio service mesh..."
    
    for region in "${REGIONS[@]}"; do
        log_info "Installing Istio in $region"
        kubectl config use-context "nephoran-${region}"
        
        # Install Istio with multi-cluster configuration
        istioctl install -y \
            --set values.pilot.env.EXTERNAL_ISTIOD=true \
            --set values.global.meshID=nephoran-mesh \
            --set values.global.multiCluster.clusterName="nephoran-${region}" \
            --set values.global.network="${region}"
    done
    
    # Configure multi-cluster mesh
    configure_multi_cluster_mesh
    
    log_success "Istio deployed successfully"
}

# Configure multi-cluster mesh
configure_multi_cluster_mesh() {
    log_info "Configuring multi-cluster mesh..."
    
    # Create remote secrets for each cluster
    for region in "${REGIONS[@]}"; do
        kubectl config use-context "nephoran-${region}"
        
        for remote_region in "${REGIONS[@]}"; do
            if [[ "$region" != "$remote_region" ]]; then
                istioctl x create-remote-secret \
                    --context="nephoran-${remote_region}" \
                    --name="nephoran-${remote_region}" | \
                    kubectl apply -f -
            fi
        done
    done
    
    log_success "Multi-cluster mesh configured"
}

# Deploy Weaviate clusters
deploy_weaviate() {
    log_info "Deploying Weaviate vector database..."
    
    for region in "${REGIONS[@]}"; do
        log_info "Deploying Weaviate in $region"
        kubectl config use-context "nephoran-${region}"
        
        # Apply Weaviate configuration
        kubectl apply -k "${K8S_DIR}/overlays/${region}/weaviate"
        
        # Wait for Weaviate to be ready
        kubectl wait --for=condition=ready pod \
            -l app=weaviate \
            -n nephoran-system \
            --timeout=300s
    done
    
    # Configure cross-region replication
    configure_weaviate_replication
    
    log_success "Weaviate deployed successfully"
}

# Configure Weaviate replication
configure_weaviate_replication() {
    log_info "Configuring Weaviate cross-region replication..."
    
    # Get Weaviate endpoints
    local endpoints=()
    for region in "${REGIONS[@]}"; do
        kubectl config use-context "nephoran-${region}"
        local endpoint=$(kubectl get svc weaviate-lb -n nephoran-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
        endpoints+=("${region}:${endpoint}")
    done
    
    # Configure replication on primary cluster
    kubectl config use-context "nephoran-${PRIMARY_REGION}"
    
    # Apply replication configuration
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: weaviate-replication-config
  namespace: nephoran-system
data:
  replication.json: |
    {
      "replication_factor": 3,
      "regions": {
$(for i in "${!endpoints[@]}"; do
    IFS=':' read -r region endpoint <<< "${endpoints[$i]}"
    echo "        \"$region\": {"
    echo "          \"endpoint\": \"http://$endpoint:8080\","
    echo "          \"priority\": $((100 - i * 10))"
    echo "        }$([[ $i -lt $((${#endpoints[@]} - 1)) ]] && echo ",")"
done)
      }
    }
EOF
    
    log_success "Weaviate replication configured"
}

# Deploy application services
deploy_services() {
    log_info "Deploying application services..."
    
    for region in "${REGIONS[@]}"; do
        log_info "Deploying services in $region"
        kubectl config use-context "nephoran-${region}"
        
        # Apply region-specific configuration
        kustomize build "${K8S_DIR}/overlays/${region}" | \
            sed "s/PROJECT_ID/${PROJECT_ID}/g" | \
            kubectl apply -f -
        
        # Wait for deployments to be ready
        kubectl wait --for=condition=available deployment \
            -l app.kubernetes.io/part-of=nephoran-intent-operator \
            -n nephoran-system \
            --timeout=600s
    done
    
    log_success "Application services deployed"
}

# Configure global load balancing
configure_global_load_balancing() {
    log_info "Configuring global load balancing..."
    
    # Reserve global IP address
    gcloud compute addresses create nephoran-global-ip \
        --global \
        --project="$PROJECT_ID" || true
    
    # Get the reserved IP
    GLOBAL_IP=$(gcloud compute addresses describe nephoran-global-ip \
        --global \
        --project="$PROJECT_ID" \
        --format="value(address)")
    
    # Create SSL certificate
    gcloud compute ssl-certificates create nephoran-ssl-cert \
        --domains="nephoran.${PROJECT_ID}.com,*.nephoran.${PROJECT_ID}.com" \
        --global \
        --project="$PROJECT_ID" || true
    
    # Apply multi-cluster ingress
    kubectl config use-context "nephoran-${PRIMARY_REGION}"
    kubectl apply -f "${K8S_DIR}/base/multi-region-services.yaml"
    
    log_success "Global load balancing configured with IP: $GLOBAL_IP"
}

# Setup monitoring
setup_monitoring() {
    log_info "Setting up monitoring and observability..."
    
    # Deploy Prometheus and Grafana
    for region in "${REGIONS[@]}"; do
        kubectl config use-context "nephoran-${region}"
        
        # Create monitoring namespace
        kubectl create namespace monitoring || true
        
        # Install Prometheus Operator
        helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
        helm repo update
        
        helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
            --namespace monitoring \
            --set prometheus.prometheusSpec.replicas=2 \
            --set prometheus.prometheusSpec.retention=30d \
            --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=100Gi \
            --set grafana.persistence.enabled=true \
            --set grafana.persistence.size=10Gi \
            --wait
    done
    
    # Deploy Thanos for global metrics
    deploy_thanos
    
    log_success "Monitoring setup complete"
}

# Deploy Thanos for global metrics
deploy_thanos() {
    log_info "Deploying Thanos for global metrics..."
    
    kubectl config use-context "nephoran-${PRIMARY_REGION}"
    
    # Install Thanos
    helm repo add bitnami https://charts.bitnami.com/bitnami
    helm upgrade --install thanos bitnami/thanos \
        --namespace monitoring \
        --set query.replicaCount=2 \
        --set queryFrontend.replicaCount=2 \
        --set storegateway.replicaCount=2 \
        --set compactor.retentionResolution5m=30d \
        --set compactor.retentionResolution1h=90d \
        --set objstoreConfig.type=GCS \
        --set objstoreConfig.config.bucket="${PROJECT_ID}-thanos-metrics" \
        --wait
    
    log_success "Thanos deployed"
}

# Perform health checks
perform_health_checks() {
    log_info "Performing health checks..."
    
    local all_healthy=true
    
    for region in "${REGIONS[@]}"; do
        log_info "Checking health in $region"
        kubectl config use-context "nephoran-${region}"
        
        # Check deployments
        if ! kubectl get deployments -n nephoran-system -o json | \
            jq -e '.items[] | select(.status.replicas != .status.readyReplicas) | empty' &> /dev/null; then
            log_warning "Some deployments are not ready in $region"
            all_healthy=false
        fi
        
        # Check pods
        if kubectl get pods -n nephoran-system -o json | \
            jq -e '.items[] | select(.status.phase != "Running") | length > 0' &> /dev/null; then
            log_warning "Some pods are not running in $region"
            all_healthy=false
        fi
        
        # Check services
        if ! kubectl get endpoints -n nephoran-system -o json | \
            jq -e '.items[] | select(.subsets == null or .subsets[].addresses == null) | empty' &> /dev/null; then
            log_warning "Some services have no endpoints in $region"
            all_healthy=false
        fi
    done
    
    if $all_healthy; then
        log_success "All health checks passed"
    else
        log_error "Some health checks failed"
        exit 1
    fi
}

# Main deployment flow
main() {
    log_info "Starting multi-region deployment for Nephoran Intent Operator"
    log_info "Project: $PROJECT_ID, Environment: $ENVIRONMENT"
    
    validate_prerequisites
    
    # Infrastructure deployment
    if [[ "${SKIP_TERRAFORM:-false}" != "true" ]]; then
        deploy_infrastructure
    fi
    
    # Kubernetes configuration
    configure_kubectl
    
    # Service mesh deployment
    if [[ "${SKIP_ISTIO:-false}" != "true" ]]; then
        deploy_istio
    fi
    
    # Application deployment
    deploy_weaviate
    deploy_services
    
    # Global load balancing
    configure_global_load_balancing
    
    # Monitoring setup
    if [[ "${SKIP_MONITORING:-false}" != "true" ]]; then
        setup_monitoring
    fi
    
    # Health checks
    perform_health_checks
    
    log_success "Multi-region deployment completed successfully!"
    log_info "Global endpoint: https://nephoran.${PROJECT_ID}.com"
}

# Run main function
main "$@"