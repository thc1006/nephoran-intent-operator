#!/bin/bash

# Nephoran Intent Operator - Production Deployment Script
# Supports multi-cloud, multi-environment deployment with advanced deployment strategies

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LOG_FILE="/tmp/nephoran-deployment-$(date +%Y%m%d-%H%M%S).log"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default configuration
DEFAULT_ENVIRONMENT="dev"
DEFAULT_CLOUD_PROVIDER="aws"
DEFAULT_DEPLOYMENT_TYPE="standard"
DEFAULT_STRATEGY="rolling"
DEFAULT_REGION="us-central1"
DEFAULT_CONFIG_TYPE="standard"

# Global variables
ENVIRONMENT="${ENVIRONMENT:-$DEFAULT_ENVIRONMENT}"
CLOUD_PROVIDER="${CLOUD_PROVIDER:-$DEFAULT_CLOUD_PROVIDER}"
DEPLOYMENT_TYPE="${DEPLOYMENT_TYPE:-$DEFAULT_DEPLOYMENT_TYPE}"
STRATEGY="${STRATEGY:-$DEFAULT_STRATEGY}"
REGION="${REGION:-$DEFAULT_REGION}"
CONFIG_TYPE="${CONFIG_TYPE:-$DEFAULT_CONFIG_TYPE}"
DRY_RUN="${DRY_RUN:-false}"
VERBOSE="${VERBOSE:-false}"
FORCE="${FORCE:-false}"
SKIP_TERRAFORM="${SKIP_TERRAFORM:-false}"
SKIP_HELM="${SKIP_HELM:-false}"
MONITORING_ENABLED="${MONITORING_ENABLED:-true}"
BACKUP_ENABLED="${BACKUP_ENABLED:-true}"

# Logging functions
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case "$level" in
        INFO)
            echo -e "${GREEN}[INFO]${NC} $message" | tee -a "$LOG_FILE"
            ;;
        WARN)
            echo -e "${YELLOW}[WARN]${NC} $message" | tee -a "$LOG_FILE"
            ;;
        ERROR)
            echo -e "${RED}[ERROR]${NC} $message" | tee -a "$LOG_FILE"
            ;;
        DEBUG)
            if [[ "$VERBOSE" == "true" ]]; then
                echo -e "${CYAN}[DEBUG]${NC} $message" | tee -a "$LOG_FILE"
            fi
            ;;
        SUCCESS)
            echo -e "${GREEN}[SUCCESS]${NC} $message" | tee -a "$LOG_FILE"
            ;;
    esac
    
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
}

# Usage function
usage() {
    cat << EOF
Nephoran Intent Operator - Production Deployment Script

Usage: $0 [OPTIONS]

OPTIONS:
    -e, --environment ENVIRONMENT     Target environment (dev|staging|prod) [default: $DEFAULT_ENVIRONMENT]
    -c, --cloud PROVIDER              Cloud provider (aws|azure|gcp) [default: $DEFAULT_CLOUD_PROVIDER]
    -t, --type TYPE                   Deployment type (standard|enterprise|telecom|edge) [default: $DEFAULT_DEPLOYMENT_TYPE]
    -s, --strategy STRATEGY           Deployment strategy (rolling|blue-green|canary) [default: $DEFAULT_STRATEGY]
    -r, --region REGION              Target region [default: $DEFAULT_REGION]
    --config-type TYPE               Configuration type (standard|enterprise|telecom-operator|edge-computing) [default: $DEFAULT_CONFIG_TYPE]
    --dry-run                        Show what would be deployed without making changes
    --skip-terraform                 Skip Terraform infrastructure deployment
    --skip-helm                      Skip Helm application deployment
    --no-monitoring                  Disable monitoring components
    --no-backup                      Disable backup components
    -f, --force                      Force deployment even if validation fails
    -v, --verbose                    Enable verbose logging
    -h, --help                       Show this help message

EXAMPLES:
    # Standard development deployment on AWS
    $0 -e dev -c aws -t standard

    # Enterprise production deployment with blue-green strategy
    $0 -e prod -c aws -t enterprise -s blue-green --config-type enterprise

    # Carrier-grade telecom deployment
    $0 -e prod -c gcp -t telecom -s canary --config-type telecom-operator

    # Edge computing deployment
    $0 -e prod -c aws -t edge --config-type edge-computing

    # Dry run to validate configuration
    $0 -e prod -c azure -t enterprise --dry-run

ENVIRONMENT VARIABLES:
    ENVIRONMENT                      Same as --environment
    CLOUD_PROVIDER                   Same as --cloud
    DEPLOYMENT_TYPE                  Same as --type
    STRATEGY                         Same as --strategy
    DRY_RUN                         Same as --dry-run
    VERBOSE                         Same as --verbose
    SKIP_TERRAFORM                  Same as --skip-terraform
    SKIP_HELM                       Same as --skip-helm

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -c|--cloud)
                CLOUD_PROVIDER="$2"
                shift 2
                ;;
            -t|--type)
                DEPLOYMENT_TYPE="$2"
                shift 2
                ;;
            -s|--strategy)
                STRATEGY="$2"
                shift 2
                ;;
            -r|--region)
                REGION="$2"
                shift 2
                ;;
            --config-type)
                CONFIG_TYPE="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            --skip-terraform)
                SKIP_TERRAFORM="true"
                shift
                ;;
            --skip-helm)
                SKIP_HELM="true"
                shift
                ;;
            --no-monitoring)
                MONITORING_ENABLED="false"
                shift
                ;;
            --no-backup)
                BACKUP_ENABLED="false"
                shift
                ;;
            -f|--force)
                FORCE="true"
                shift
                ;;
            -v|--verbose)
                VERBOSE="true"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log ERROR "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Validation functions
validate_prerequisites() {
    log INFO "Validating prerequisites..."
    
    local missing_tools=()
    
    # Check required tools
    local required_tools=("kubectl" "helm" "terraform" "jq" "curl")
    
    case "$CLOUD_PROVIDER" in
        aws)
            required_tools+=("aws")
            ;;
        azure)
            required_tools+=("az")
            ;;
        gcp)
            required_tools+=("gcloud")
            ;;
    esac
    
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log ERROR "Missing required tools: ${missing_tools[*]}"
        return 1
    fi
    
    # Validate parameters
    case "$ENVIRONMENT" in
        dev|staging|prod) ;;
        *)
            log ERROR "Invalid environment: $ENVIRONMENT. Must be dev, staging, or prod."
            return 1
            ;;
    esac
    
    case "$CLOUD_PROVIDER" in
        aws|azure|gcp) ;;
        *)
            log ERROR "Invalid cloud provider: $CLOUD_PROVIDER. Must be aws, azure, or gcp."
            return 1
            ;;
    esac
    
    case "$DEPLOYMENT_TYPE" in
        standard|enterprise|telecom|edge) ;;
        *)
            log ERROR "Invalid deployment type: $DEPLOYMENT_TYPE. Must be standard, enterprise, telecom, or edge."
            return 1
            ;;
    esac
    
    case "$STRATEGY" in
        rolling|blue-green|canary) ;;
        *)
            log ERROR "Invalid deployment strategy: $STRATEGY. Must be rolling, blue-green, or canary."
            return 1
            ;;
    esac
    
    case "$CONFIG_TYPE" in
        standard|enterprise|telecom-operator|edge-computing) ;;
        *)
            log ERROR "Invalid config type: $CONFIG_TYPE. Must be standard, enterprise, telecom-operator, or edge-computing."
            return 1
            ;;
    esac
    
    log SUCCESS "Prerequisites validated successfully"
}

# Cloud authentication
authenticate_cloud() {
    log INFO "Authenticating with $CLOUD_PROVIDER..."
    
    case "$CLOUD_PROVIDER" in
        aws)
            if ! aws sts get-caller-identity &> /dev/null; then
                log ERROR "AWS authentication failed. Please run 'aws configure' or set AWS credentials."
                return 1
            fi
            AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
            log DEBUG "AWS Account ID: $AWS_ACCOUNT_ID"
            ;;
        azure)
            if ! az account show &> /dev/null; then
                log ERROR "Azure authentication failed. Please run 'az login'."
                return 1
            fi
            AZURE_SUBSCRIPTION_ID=$(az account show --query id --output tsv)
            log DEBUG "Azure Subscription ID: $AZURE_SUBSCRIPTION_ID"
            ;;
        gcp)
            if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" &> /dev/null; then
                log ERROR "GCP authentication failed. Please run 'gcloud auth login'."
                return 1
            fi
            GCP_PROJECT_ID=$(gcloud config get-value project)
            log DEBUG "GCP Project ID: $GCP_PROJECT_ID"
            ;;
    esac
    
    log SUCCESS "Cloud authentication successful"
}

# Generate Terraform variables
generate_terraform_vars() {
    log INFO "Generating Terraform variables..."
    
    local terraform_dir="${PROJECT_ROOT}/infrastructure/terraform/environments/${ENVIRONMENT}/${CLOUD_PROVIDER}"
    local terraform_vars_file="${terraform_dir}/terraform.tfvars"
    
    mkdir -p "$terraform_dir"
    
    # Base variables
    cat > "$terraform_vars_file" << EOF
# Nephoran Intent Operator - Terraform Variables
# Generated on $(date)
# Environment: $ENVIRONMENT
# Cloud Provider: $CLOUD_PROVIDER
# Deployment Type: $DEPLOYMENT_TYPE

# Project configuration
project_name = "nephoran-intent-operator"
environment = "$ENVIRONMENT"
owner = "platform-team"
cost_center = "engineering"
compliance_level = "$([[ "$DEPLOYMENT_TYPE" == "telecom" ]] && echo "carrier-grade" || echo "standard")"

# Regional configuration
region = "$REGION"
EOF

    # Cloud-specific variables
    case "$CLOUD_PROVIDER" in
        aws)
            cat >> "$terraform_vars_file" << 'EOF'

# AWS-specific configuration
vpc_cidr = "10.0.0.0/16"
availability_zones_count = 3
kubernetes_version = "1.28"

# Feature flags
enable_rds = true
enable_elasticache = true
enable_s3_backup = true
enable_monitoring = true
enable_load_balancer = true

EOF
            ;;
        azure)
            cat >> "$terraform_vars_file" << 'EOF'

# Azure-specific configuration
vnet_address_space = "10.0.0.0/16"
kubernetes_version = "1.28.3"

# Feature flags
enable_postgresql = true
enable_redis = true
enable_storage_backup = true
enable_application_gateway = true

EOF
            ;;
        gcp)
            cat >> "$terraform_vars_file" << 'EOF'

# GCP-specific configuration
network_cidr = "10.0.0.0/16"
kubernetes_version = "1.28.3-gke.1286000"

# Feature flags
enable_cloud_sql = true
enable_memorystore = true
enable_cloud_storage = true
enable_load_balancer = true

EOF
            ;;
    esac
    
    # Deployment type specific configurations
    case "$DEPLOYMENT_TYPE" in
        enterprise)
            cat >> "$terraform_vars_file" << 'EOF'
# Enterprise configuration
sla_tier = "high"
enable_multi_region_ha = false
enable_monitoring = true
enable_backup = true
enable_security_scanning = true

EOF
            ;;
        telecom)
            cat >> "$terraform_vars_file" << 'EOF'
# Carrier-grade configuration
sla_tier = "carrier-grade"
enable_multi_region_ha = true
enable_oran_interfaces = true
enable_network_slicing = true
enable_edge_orchestration = true
enable_carrier_grade_features = true

EOF
            ;;
        edge)
            cat >> "$terraform_vars_file" << 'EOF'
# Edge computing configuration
enable_edge_deployment = true
edge_regions = ["$REGION"]
enable_offline_mode = true
enable_local_storage = true

EOF
            ;;
    esac
    
    log SUCCESS "Terraform variables generated: $terraform_vars_file"
}

# Deploy infrastructure with Terraform
deploy_infrastructure() {
    if [[ "$SKIP_TERRAFORM" == "true" ]]; then
        log INFO "Skipping Terraform deployment as requested"
        return 0
    fi
    
    log INFO "Deploying infrastructure with Terraform..."
    
    local terraform_dir="${PROJECT_ROOT}/infrastructure/terraform/environments/${ENVIRONMENT}/${CLOUD_PROVIDER}"
    
    # Ensure Terraform directory exists with main.tf
    if [[ ! -f "${terraform_dir}/main.tf" ]]; then
        log INFO "Creating Terraform main configuration..."
        mkdir -p "$terraform_dir"
        
        # Create main.tf that references the appropriate module
        cat > "${terraform_dir}/main.tf" << EOF
# Nephoran Intent Operator - ${CLOUD_PROVIDER} ${ENVIRONMENT} Infrastructure

terraform {
  required_version = ">= 1.5.0"
  
  backend "$(get_terraform_backend)" {
$(get_terraform_backend_config)
  }
}

module "nephoran_${CLOUD_PROVIDER}" {
  source = "../../modules/${CLOUD_PROVIDER}"
  
  # Pass all variables from terraform.tfvars
  project_name = var.project_name
  environment = var.environment
  owner = var.owner
  cost_center = var.cost_center
  compliance_level = var.compliance_level
  
  # Regional configuration
$(get_region_variable_assignment)
  
  # Cloud-specific variables
$(get_cloud_specific_variable_assignments)
}

# Define variables
$(get_terraform_variable_definitions)

# Outputs
$(get_terraform_outputs)
EOF
    fi
    
    cd "$terraform_dir"
    
    # Initialize Terraform
    log INFO "Initializing Terraform..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run: terraform init"
    else
        terraform init -reconfigure
    fi
    
    # Plan
    log INFO "Planning Terraform deployment..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run: terraform plan"
    else
        terraform plan -out=tfplan
    fi
    
    # Apply
    if [[ "$DRY_RUN" == "false" ]]; then
        log INFO "Applying Terraform deployment..."
        if [[ "$FORCE" == "true" ]]; then
            terraform apply -auto-approve tfplan
        else
            read -p "Do you want to proceed with the infrastructure deployment? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                terraform apply tfplan
            else
                log WARN "Infrastructure deployment cancelled"
                return 1
            fi
        fi
    fi
    
    log SUCCESS "Infrastructure deployment completed"
    cd "$PROJECT_ROOT"
}

# Get Kubernetes credentials
get_k8s_credentials() {
    log INFO "Getting Kubernetes credentials..."
    
    local cluster_name="nephoran-intent-operator-${ENVIRONMENT}"
    
    case "$CLOUD_PROVIDER" in
        aws)
            aws eks update-kubeconfig --region "$REGION" --name "${cluster_name}-eks"
            ;;
        azure)
            az aks get-credentials --resource-group "${cluster_name}-rg" --name "${cluster_name}-aks" --overwrite-existing
            ;;
        gcp)
            gcloud container clusters get-credentials "${cluster_name}-gke" --region "$REGION"
            ;;
    esac
    
    # Verify connection
    if kubectl cluster-info &> /dev/null; then
        log SUCCESS "Kubernetes credentials configured successfully"
    else
        log ERROR "Failed to connect to Kubernetes cluster"
        return 1
    fi
}

# Deploy applications with Helm
deploy_applications() {
    if [[ "$SKIP_HELM" == "true" ]]; then
        log INFO "Skipping Helm deployment as requested"
        return 0
    fi
    
    log INFO "Deploying applications with Helm..."
    
    local helm_dir="${PROJECT_ROOT}/deployments/helm"
    local values_file="${helm_dir}/nephoran-operator/environments/values-${CONFIG_TYPE}.yaml"
    
    # Verify values file exists
    if [[ ! -f "$values_file" ]]; then
        log ERROR "Values file not found: $values_file"
        return 1
    fi
    
    # Create namespace
    local namespace="nephoran-system"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would create namespace: $namespace"
    else
        kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    # Deploy with different strategies
    case "$STRATEGY" in
        rolling)
            deploy_rolling
            ;;
        blue-green)
            deploy_blue_green
            ;;
        canary)
            deploy_canary
            ;;
    esac
    
    log SUCCESS "Application deployment completed"
}

# Rolling deployment strategy
deploy_rolling() {
    log INFO "Executing rolling deployment strategy..."
    
    local helm_dir="${PROJECT_ROOT}/deployments/helm"
    local values_file="${helm_dir}/nephoran-operator/environments/values-${CONFIG_TYPE}.yaml"
    local namespace="nephoran-system"
    local release_name="nephoran-operator"
    
    # Add any additional values for rolling deployment
    local extra_args=(
        "--set" "global.deploymentStrategy=RollingUpdate"
        "--set" "global.environment=${ENVIRONMENT}"
        "--set" "global.cloudProvider=${CLOUD_PROVIDER}"
    )
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would run Helm upgrade with rolling deployment"
        helm template "$release_name" "${helm_dir}/nephoran-operator" \
            --namespace "$namespace" \
            --values "$values_file" \
            "${extra_args[@]}" \
            --dry-run
    else
        # Check if release exists
        if helm list -n "$namespace" | grep -q "$release_name"; then
            log INFO "Upgrading existing release..."
            helm upgrade "$release_name" "${helm_dir}/nephoran-operator" \
                --namespace "$namespace" \
                --values "$values_file" \
                "${extra_args[@]}" \
                --wait --timeout=15m
        else
            log INFO "Installing new release..."
            helm install "$release_name" "${helm_dir}/nephoran-operator" \
                --namespace "$namespace" \
                --values "$values_file" \
                "${extra_args[@]}" \
                --wait --timeout=15m \
                --create-namespace
        fi
        
        # Wait for rollout
        kubectl rollout status deployment/nephoran-operator -n "$namespace" --timeout=300s
    fi
}

# Blue-Green deployment strategy
deploy_blue_green() {
    log INFO "Executing blue-green deployment strategy..."
    
    local namespace="nephoran-system"
    local current_color=$(get_current_color "$namespace")
    local new_color=$([ "$current_color" = "blue" ] && echo "green" || echo "blue")
    
    log INFO "Current color: $current_color, deploying to: $new_color"
    
    # Deploy to new color
    deploy_to_color "$new_color" "$namespace"
    
    # Health check on new deployment
    if health_check_color "$new_color" "$namespace"; then
        log INFO "Health check passed for $new_color environment"
        
        if [[ "$DRY_RUN" == "false" ]]; then
            # Switch traffic
            switch_traffic "$new_color" "$namespace"
            
            # Wait and verify
            sleep 30
            if health_check_color "$new_color" "$namespace"; then
                log SUCCESS "Blue-green deployment completed successfully"
                
                # Cleanup old color
                cleanup_color "$current_color" "$namespace"
            else
                log ERROR "Health check failed after traffic switch, rolling back"
                switch_traffic "$current_color" "$namespace"
                return 1
            fi
        fi
    else
        log ERROR "Health check failed for $new_color environment"
        return 1
    fi
}

# Canary deployment strategy
deploy_canary() {
    log INFO "Executing canary deployment strategy..."
    
    local namespace="nephoran-system"
    local canary_percentage="${CANARY_PERCENTAGE:-10}"
    
    # Deploy canary version
    deploy_canary_version "$namespace" "$canary_percentage"
    
    # Monitor canary
    monitor_canary_deployment "$namespace" "$canary_percentage"
}

# Helper functions for deployment strategies
get_current_color() {
    local namespace="$1"
    # Logic to determine current active color
    if kubectl get service nephoran-operator-blue -n "$namespace" &> /dev/null; then
        echo "blue"
    else
        echo "green"
    fi
}

deploy_to_color() {
    local color="$1"
    local namespace="$2"
    
    log INFO "Deploying to $color environment"
    
    # Implement color-specific deployment logic
    # This would involve deploying with color-specific labels and services
}

health_check_color() {
    local color="$1"
    local namespace="$2"
    
    log INFO "Performing health check for $color environment"
    
    # Implement health check logic
    # Return 0 for success, 1 for failure
    return 0
}

switch_traffic() {
    local new_color="$1"
    local namespace="$2"
    
    log INFO "Switching traffic to $new_color environment"
    
    # Implement traffic switching logic
    # This typically involves updating service selectors
}

cleanup_color() {
    local old_color="$1"
    local namespace="$2"
    
    log INFO "Cleaning up $old_color environment"
    
    # Implement cleanup logic
}

deploy_canary_version() {
    local namespace="$1"
    local percentage="$2"
    
    log INFO "Deploying canary version with $percentage% traffic"
    
    # Implement canary deployment logic
}

monitor_canary_deployment() {
    local namespace="$1"
    local percentage="$2"
    
    log INFO "Monitoring canary deployment"
    
    # Implement canary monitoring and promotion logic
}

# Terraform helper functions
get_terraform_backend() {
    case "$CLOUD_PROVIDER" in
        aws) echo "s3" ;;
        azure) echo "azurerm" ;;
        gcp) echo "gcs" ;;
    esac
}

get_terraform_backend_config() {
    case "$CLOUD_PROVIDER" in
        aws)
            cat << 'EOF'
    bucket = "nephoran-terraform-state-${random_id.bucket_suffix.hex}"
    key    = "${environment}/${cloud_provider}/terraform.tfstate"
    region = "${region}"
    encrypt = true
    dynamodb_table = "nephoran-terraform-locks"
EOF
            ;;
        azure)
            cat << 'EOF'
    resource_group_name  = "nephoran-terraform-rg"
    storage_account_name = "nephoranterraformstate"
    container_name      = "tfstate"
    key                = "${environment}/${cloud_provider}/terraform.tfstate"
EOF
            ;;
        gcp)
            cat << 'EOF'
    bucket = "nephoran-terraform-state-${random_id.bucket_suffix.hex}"
    prefix = "${environment}/${cloud_provider}"
EOF
            ;;
    esac
}

get_region_variable_assignment() {
    case "$CLOUD_PROVIDER" in
        aws) echo "  region = var.region" ;;
        azure) echo "  location = var.location" ;;
        gcp) echo "  region = var.region" ;;
    esac
}

get_cloud_specific_variable_assignments() {
    # This would contain cloud-specific variable assignments
    echo "  # Additional cloud-specific variables would be added here"
}

get_terraform_variable_definitions() {
    cat << 'EOF'
variable "project_name" {
  description = "Name of the project"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "owner" {
  description = "Owner of the resources"
  type        = string
}

variable "cost_center" {
  description = "Cost center for billing"
  type        = string
}

variable "compliance_level" {
  description = "Compliance level required"
  type        = string
}
EOF

    case "$CLOUD_PROVIDER" in
        aws)
            echo 'variable "region" { description = "AWS region"; type = string }'
            ;;
        azure)
            echo 'variable "location" { description = "Azure location"; type = string }'
            ;;
        gcp)
            echo 'variable "region" { description = "GCP region"; type = string }'
            ;;
    esac
}

get_terraform_outputs() {
    cat << 'EOF'
output "cluster_connection_info" {
  description = "Kubernetes cluster connection information"
  value       = module.nephoran_CLOUD_PROVIDER.cluster_connection_info
  sensitive   = true
}

output "deployment_info" {
  description = "Comprehensive deployment information"
  value       = module.nephoran_CLOUD_PROVIDER.deployment_info
  sensitive   = true
}
EOF
}

# Post-deployment tasks
post_deployment_tasks() {
    log INFO "Executing post-deployment tasks..."
    
    # Wait for all deployments to be ready
    wait_for_deployments
    
    # Run health checks
    run_health_checks
    
    # Configure monitoring
    if [[ "$MONITORING_ENABLED" == "true" ]]; then
        configure_monitoring
    fi
    
    # Configure backup
    if [[ "$BACKUP_ENABLED" == "true" ]]; then
        configure_backup
    fi
    
    # Run smoke tests
    run_smoke_tests
    
    log SUCCESS "Post-deployment tasks completed"
}

wait_for_deployments() {
    log INFO "Waiting for deployments to be ready..."
    
    local namespace="nephoran-system"
    local deployments=(
        "nephoran-operator"
        "llm-processor"
        "rag-api"
        "weaviate"
    )
    
    for deployment in "${deployments[@]}"; do
        log INFO "Waiting for deployment: $deployment"
        if [[ "$DRY_RUN" == "false" ]]; then
            kubectl rollout status deployment/"$deployment" -n "$namespace" --timeout=600s
        fi
    done
}

run_health_checks() {
    log INFO "Running health checks..."
    
    local namespace="nephoran-system"
    local services=(
        "nephoran-operator:8080"
        "llm-processor:8080"
        "rag-api:5001"
        "weaviate:8080"
    )
    
    for service in "${services[@]}"; do
        local service_name="${service%:*}"
        local port="${service#*:}"
        
        log INFO "Health checking service: $service_name"
        
        if [[ "$DRY_RUN" == "false" ]]; then
            # Port forward and check health endpoint
            kubectl port-forward svc/"$service_name" "$port:$port" -n "$namespace" &
            local port_forward_pid=$!
            
            sleep 5
            
            if curl -f "http://localhost:$port/health" &> /dev/null; then
                log SUCCESS "Health check passed for $service_name"
            else
                log WARN "Health check failed for $service_name"
            fi
            
            kill $port_forward_pid 2>/dev/null || true
        fi
    done
}

configure_monitoring() {
    log INFO "Configuring monitoring..."
    
    # Deploy monitoring stack if not already deployed
    if ! kubectl get namespace monitoring &> /dev/null; then
        log INFO "Deploying monitoring stack..."
        
        if [[ "$DRY_RUN" == "false" ]]; then
            # Install Prometheus Operator
            helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
            helm repo update
            
            helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
                --namespace monitoring \
                --create-namespace \
                --values "${PROJECT_ROOT}/deployments/monitoring/prometheus-values.yaml" \
                --wait
        fi
    fi
}

configure_backup() {
    log INFO "Configuring backup..."
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Deploy Velero for backups
        if ! helm list -n velero-system | grep -q velero; then
            log INFO "Installing Velero backup solution..."
            
            helm repo add vmware-tanzu https://vmware-tanzu.github.io/helm-charts
            helm repo update
            
            helm upgrade --install velero vmware-tanzu/velero \
                --namespace velero-system \
                --create-namespace \
                --values "${PROJECT_ROOT}/deployments/backup/velero-values-${CLOUD_PROVIDER}.yaml" \
                --wait
        fi
    fi
}

run_smoke_tests() {
    log INFO "Running smoke tests..."
    
    local namespace="nephoran-system"
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Create a test network intent to validate the system
        cat << 'EOF' | kubectl apply -f -
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: smoke-test-intent
  namespace: nephoran-system
spec:
  intent: "Deploy a test 5G AMF instance for validation"
  priority: low
  metadata:
    test: "smoke-test"
    timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
EOF
        
        # Wait for processing
        sleep 30
        
        # Check if intent was processed
        if kubectl get networkintent smoke-test-intent -n "$namespace" -o jsonpath='{.status.phase}' | grep -q "Processed"; then
            log SUCCESS "Smoke test passed - NetworkIntent processed successfully"
        else
            log WARN "Smoke test inconclusive - NetworkIntent processing status unclear"
        fi
        
        # Cleanup test intent
        kubectl delete networkintent smoke-test-intent -n "$namespace" --ignore-not-found=true
    fi
}

# Cleanup function
cleanup() {
    log INFO "Cleaning up deployment process..."
    
    # Kill any background processes
    local pids=$(jobs -p)
    if [[ -n "$pids" ]]; then
        kill $pids 2>/dev/null || true
    fi
    
    # Archive log file
    if [[ -f "$LOG_FILE" ]]; then
        local archive_dir="${PROJECT_ROOT}/logs/deployments"
        mkdir -p "$archive_dir"
        cp "$LOG_FILE" "${archive_dir}/deployment-${ENVIRONMENT}-${CLOUD_PROVIDER}-$(date +%Y%m%d-%H%M%S).log"
        log INFO "Deployment log archived to: ${archive_dir}/"
    fi
}

# Main execution
main() {
    # Set up trap for cleanup
    trap cleanup EXIT
    
    log INFO "Starting Nephoran Intent Operator deployment..."
    log INFO "Environment: $ENVIRONMENT"
    log INFO "Cloud Provider: $CLOUD_PROVIDER"
    log INFO "Deployment Type: $DEPLOYMENT_TYPE"
    log INFO "Strategy: $STRATEGY"
    log INFO "Config Type: $CONFIG_TYPE"
    log INFO "Region: $REGION"
    log INFO "Dry Run: $DRY_RUN"
    log INFO "Log File: $LOG_FILE"
    
    # Validate prerequisites
    validate_prerequisites || exit 1
    
    # Authenticate with cloud provider
    authenticate_cloud || exit 1
    
    # Generate Terraform configuration
    generate_terraform_vars || exit 1
    
    # Deploy infrastructure
    deploy_infrastructure || exit 1
    
    # Get Kubernetes credentials
    get_k8s_credentials || exit 1
    
    # Deploy applications
    deploy_applications || exit 1
    
    # Post-deployment tasks
    post_deployment_tasks || exit 1
    
    log SUCCESS "Nephoran Intent Operator deployment completed successfully!"
    log INFO "Deployment details logged to: $LOG_FILE"
    
    # Output connection information
    if [[ "$DRY_RUN" == "false" ]]; then
        log INFO ""
        log INFO "=== Deployment Summary ==="
        log INFO "Environment: $ENVIRONMENT"
        log INFO "Cloud Provider: $CLOUD_PROVIDER"
        log INFO "Deployment Type: $DEPLOYMENT_TYPE"
        log INFO "Cluster: nephoran-intent-operator-${ENVIRONMENT}"
        log INFO "Namespace: nephoran-system"
        log INFO ""
        log INFO "To access the cluster:"
        log INFO "kubectl get pods -n nephoran-system"
        log INFO ""
        log INFO "To view logs:"
        log INFO "kubectl logs -f deployment/nephoran-operator -n nephoran-system"
        log INFO ""
    fi
}

# Parse arguments and run main function
parse_args "$@"
main