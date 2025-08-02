#!/bin/bash

# Storage Class Detection Script for Multi-Cloud Weaviate Deployment
# Detects available storage classes and configures appropriate values for Weaviate

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m' 
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if kubectl can connect to cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig"
        exit 1
    fi
    
    log_info "Successfully connected to Kubernetes cluster"
}

# Detect cloud provider based on available storage classes and nodes
detect_cloud_provider() {
    local storage_classes
    local nodes_info
    
    # Get all storage classes
    storage_classes=$(kubectl get storageclass -o json 2>/dev/null || echo '{"items":[]}')
    
    # Get node information
    nodes_info=$(kubectl get nodes -o json 2>/dev/null || echo '{"items":[]}')
    
    # Check for AWS indicators
    if echo "$storage_classes" | jq -r '.items[].provisioner' | grep -q "ebs.csi.aws.com\|kubernetes.io/aws-ebs"; then
        echo "aws"
        return 0
    fi
    
    if echo "$nodes_info" | jq -r '.items[].spec.providerID' | grep -q "aws://"; then
        echo "aws"
        return 0
    fi
    
    # Check for GCP indicators
    if echo "$storage_classes" | jq -r '.items[].provisioner' | grep -q "pd.csi.storage.gke.io\|kubernetes.io/gce-pd"; then
        echo "gcp"
        return 0
    fi
    
    if echo "$nodes_info" | jq -r '.items[].spec.providerID' | grep -q "gce://"; then
        echo "gcp"
        return 0
    fi
    
    # Check for Azure indicators
    if echo "$storage_classes" | jq -r '.items[].provisioner' | grep -q "disk.csi.azure.com\|kubernetes.io/azure-disk"; then
        echo "azure"
        return 0
    fi
    
    if echo "$nodes_info" | jq -r '.items[].spec.providerID' | grep -q "azure://"; then
        echo "azure"
        return 0
    fi
    
    # Default to generic if no specific provider detected
    echo "generic"
}

# Get available storage classes for a provider
get_available_storage_classes() {
    local provider="$1"
    local storage_classes
    
    storage_classes=$(kubectl get storageclass -o json 2>/dev/null || echo '{"items":[]}')
    
    case "$provider" in
        "aws")
            # Look for AWS storage classes
            echo "$storage_classes" | jq -r '.items[] | select(.provisioner | contains("ebs.csi.aws.com") or contains("kubernetes.io/aws-ebs")) | .metadata.name'
            ;;
        "gcp")
            # Look for GCP storage classes
            echo "$storage_classes" | jq -r '.items[] | select(.provisioner | contains("pd.csi.storage.gke.io") or contains("kubernetes.io/gce-pd")) | .metadata.name'
            ;;
        "azure")
            # Look for Azure storage classes
            echo "$storage_classes" | jq -r '.items[] | select(.provisioner | contains("disk.csi.azure.com") or contains("kubernetes.io/azure-disk")) | .metadata.name'
            ;;
        "generic")
            # Get default storage class or first available
            local default_sc
            default_sc=$(echo "$storage_classes" | jq -r '.items[] | select(.metadata.annotations["storageclass.kubernetes.io/is-default-class"] == "true") | .metadata.name' | head -1)
            if [[ -n "$default_sc" ]]; then
                echo "$default_sc"
            else
                echo "$storage_classes" | jq -r '.items[0].metadata.name // "default"'
            fi
            ;;
    esac
}

# Map storage classes to performance tiers
map_storage_classes() {
    local provider="$1"
    local available_classes="$2"
    
    declare -A class_mapping
    
    case "$provider" in
        "aws")
            # Preferred mapping for AWS
            for class in $available_classes; do
                case "$class" in
                    *"gp3"*|*"gp2"*)
                        class_mapping["primary"]="$class"
                        if [[ "$class" == *"gp3"* ]]; then
                            class_mapping["fast"]="$class"
                        fi
                        ;;
                    *"io1"*|*"io2"*)
                        class_mapping["fast"]="$class"
                        ;;
                    *"sc1"*|*"st1"*)
                        class_mapping["backup"]="$class"
                        ;;
                esac
            done
            ;;
        "gcp")
            # Preferred mapping for GCP
            for class in $available_classes; do
                case "$class" in
                    *"ssd"*)
                        class_mapping["primary"]="$class"
                        class_mapping["fast"]="$class"
                        ;;
                    *"standard"*)
                        class_mapping["backup"]="$class"
                        if [[ -z "${class_mapping["primary"]:-}" ]]; then
                            class_mapping["primary"]="$class"
                        fi
                        ;;
                esac
            done
            ;;
        "azure")
            # Preferred mapping for Azure
            for class in $available_classes; do
                case "$class" in
                    *"premium"*)
                        class_mapping["primary"]="$class"
                        class_mapping["fast"]="$class"
                        ;;
                    *"standard"*|*"default"*)
                        class_mapping["backup"]="$class"
                        if [[ -z "${class_mapping["primary"]:-}" ]]; then
                            class_mapping["primary"]="$class"
                        fi
                        ;;
                esac
            done
            ;;
        "generic")
            # Generic mapping - use first available for all tiers
            local default_class
            default_class=$(echo "$available_classes" | head -1)
            class_mapping["primary"]="$default_class"
            class_mapping["backup"]="$default_class"
            class_mapping["fast"]="$default_class"
            ;;
    esac
    
    # Set defaults if not found
    local first_available
    first_available=$(echo "$available_classes" | head -1)
    class_mapping["primary"]="${class_mapping["primary"]:-$first_available}"
    class_mapping["backup"]="${class_mapping["backup"]:-${class_mapping["primary"]}}"
    class_mapping["fast"]="${class_mapping["fast"]:-${class_mapping["primary"]}}"
    
    echo "primary=${class_mapping["primary"]}"
    echo "backup=${class_mapping["backup"]}"
    echo "fast=${class_mapping["fast"]}"
}

# Validate storage class capabilities
validate_storage_class() {
    local class_name="$1"
    local provider="$2"
    
    if [[ -z "$class_name" || "$class_name" == "null" ]]; then
        return 1
    fi
    
    # Check if storage class exists
    if ! kubectl get storageclass "$class_name" &> /dev/null; then
        log_warn "Storage class '$class_name' not found"
        return 1
    fi
    
    log_info "Validated storage class: $class_name"
    return 0
}

# Generate values override file
generate_values_override() {
    local provider="$1"
    local primary_class="$2"
    local backup_class="$3"
    local fast_class="$4"
    local output_file="${5:-weaviate-values-override.yaml}"
    
    cat > "$output_file" << EOF
# Auto-generated storage class configuration for $provider
# Generated on: $(date -u +"%Y-%m-%d %H:%M:%S UTC")

global:
  detectedProvider: "$provider"
  storageClass:
    override:
      primary: "$primary_class"
      backup: "$backup_class"
      fast: "$fast_class"

# Provider-specific optimizations
weaviate:
  persistence:
    storageClass: "$primary_class"
    backup:
      storageClass: "$backup_class"
    tiers:
      hot:
        storageClass: "$fast_class"
      warm:
        storageClass: "$primary_class"

# Backup configurations with detected storage classes
backup:
  schedules:
    hourly:
      storageClass: "$fast_class"
    daily:
      storageClass: "$primary_class"
    weekly:
      storageClass: "$backup_class"
    monthly:
      storageClass: "$backup_class"
EOF

    log_info "Generated values override file: $output_file"
}

# Main function
main() {
    local output_file="weaviate-values-override.yaml"
    local dry_run=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -o|--output)
                output_file="$2"
                shift 2
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [-o|--output OUTPUT_FILE] [--dry-run] [-h|--help]"
                echo "  -o, --output    Output file for values override (default: weaviate-values-override.yaml)"
                echo "  --dry-run       Show detected configuration without writing file"
                echo "  -h, --help      Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    log_info "Starting storage class detection for Weaviate deployment..."
    
    # Check prerequisites
    check_kubectl
    
    # Detect cloud provider
    log_info "Detecting cloud provider..."
    local provider
    provider=$(detect_cloud_provider)
    log_info "Detected cloud provider: $provider"
    
    # Get available storage classes
    log_info "Scanning available storage classes..."
    local available_classes
    available_classes=$(get_available_storage_classes "$provider")
    
    if [[ -z "$available_classes" ]]; then
        log_error "No suitable storage classes found for provider: $provider"
        exit 1
    fi
    
    log_info "Available storage classes:"
    echo "$available_classes" | while read -r class; do
        if [[ -n "$class" ]]; then
            log_info "  - $class"
        fi
    done
    
    # Map storage classes to tiers
    log_info "Mapping storage classes to performance tiers..."
    local class_mappings
    class_mappings=$(map_storage_classes "$provider" "$available_classes")
    
    # Extract individual mappings
    local primary_class backup_class fast_class
    primary_class=$(echo "$class_mappings" | grep "^primary=" | cut -d'=' -f2)
    backup_class=$(echo "$class_mappings" | grep "^backup=" | cut -d'=' -f2)
    fast_class=$(echo "$class_mappings" | grep "^fast=" | cut -d'=' -f2)
    
    # Validate mappings
    local validation_failed=false
    for tier in "primary" "backup" "fast"; do
        local class_var="${tier}_class"
        local class_name="${!class_var}"
        if ! validate_storage_class "$class_name" "$provider"; then
            log_error "Validation failed for $tier storage class: $class_name"
            validation_failed=true
        fi
    done
    
    if [[ "$validation_failed" == "true" ]]; then
        log_error "Storage class validation failed. Please check your cluster configuration."
        exit 1
    fi
    
    # Display configuration
    log_info "Storage class configuration:"
    log_info "  Provider: $provider"
    log_info "  Primary (general use): $primary_class"
    log_info "  Backup (cost-effective): $backup_class"
    log_info "  Fast (high-performance): $fast_class"
    
    # Generate or display configuration
    if [[ "$dry_run" == "true" ]]; then
        log_info "Dry run mode - configuration would be:"
        generate_values_override "$provider" "$primary_class" "$backup_class" "$fast_class" "/tmp/dry-run-values.yaml"
        cat "/tmp/dry-run-values.yaml"
        rm -f "/tmp/dry-run-values.yaml"
    else
        generate_values_override "$provider" "$primary_class" "$backup_class" "$fast_class" "$output_file"
        log_info "Storage class detection completed successfully!"
        log_info "Use the generated file with: helm install weaviate -f $output_file"
    fi
}

# Run main function with all arguments
main "$@"