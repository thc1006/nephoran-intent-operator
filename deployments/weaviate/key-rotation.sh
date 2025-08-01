#!/bin/bash

# Weaviate Encryption Key Rotation Script
# Automates the rotation of API keys and encryption keys for enhanced security

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="${NAMESPACE:-nephoran-system}"
WEAVIATE_SECRET_NAME="${WEAVIATE_SECRET_NAME:-weaviate-api-key}"
OPENAI_SECRET_NAME="${OPENAI_SECRET_NAME:-openai-api-key}"
BACKUP_ENCRYPTION_SECRET="${BACKUP_ENCRYPTION_SECRET:-weaviate-backup-encryption}"
KEY_LENGTH="${KEY_LENGTH:-32}"
ROTATION_LOG_FILE="${ROTATION_LOG_FILE:-/tmp/key-rotation.log}"

# Rotation schedule configuration
ROTATION_INTERVAL_DAYS="${ROTATION_INTERVAL_DAYS:-90}"
WARNING_DAYS="${WARNING_DAYS:-7}"

# Logging functions
log_info() {
    local message="[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo -e "${GREEN}$message${NC}"
    echo "$message" >> "$ROTATION_LOG_FILE"
}

log_warn() {
    local message="[WARN] $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo -e "${YELLOW}$message${NC}"
    echo "$message" >> "$ROTATION_LOG_FILE"
}

log_error() {
    local message="[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo -e "${RED}$message${NC}"
    echo "$message" >> "$ROTATION_LOG_FILE"
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        local message="[DEBUG] $(date '+%Y-%m-%d %H:%M:%S') - $1"
        echo -e "${BLUE}$message${NC}"
        echo "$message" >> "$ROTATION_LOG_FILE"
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites for key rotation..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check openssl
    if ! command -v openssl &> /dev/null; then
        log_error "openssl is not installed or not in PATH"
        exit 1
    fi
    
    # Check base64
    if ! command -v base64 &> /dev/null; then
        log_error "base64 is not installed or not in PATH"
        exit 1
    fi
    
    # Check jq
    if ! command -v jq &> /dev/null; then
        log_error "jq is not installed or not in PATH"
        exit 1
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Namespace $NAMESPACE does not exist"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Generate cryptographically secure key
generate_secure_key() {
    local key_length="$1"
    local key_type="${2:-alphanumeric}"
    
    case "$key_type" in
        "alphanumeric")
            # Generate alphanumeric key with symbols
            openssl rand -base64 $((key_length * 3 / 4)) | tr -d "=+/" | cut -c1-${key_length}
            ;;
        "hex")
            # Generate hexadecimal key
            openssl rand -hex $((key_length / 2))
            ;;
        "uuid")
            # Generate UUID-based key
            uuidgen | tr -d '-'
            ;;
        *)
            log_error "Unknown key type: $key_type"
            return 1
            ;;
    esac
}

# Get secret creation timestamp
get_secret_age() {
    local secret_name="$1"
    local namespace="$2"
    
    if ! kubectl get secret "$secret_name" -n "$namespace" &> /dev/null; then
        echo "0"
        return
    fi
    
    local creation_timestamp
    creation_timestamp=$(kubectl get secret "$secret_name" -n "$namespace" -o jsonpath='{.metadata.creationTimestamp}')
    
    if [[ -z "$creation_timestamp" ]]; then
        echo "0"
        return
    fi
    
    # Convert to epoch time
    local creation_epoch
    creation_epoch=$(date -d "$creation_timestamp" +%s 2>/dev/null || echo "0")
    
    local current_epoch
    current_epoch=$(date +%s)
    
    # Calculate age in days
    local age_seconds=$((current_epoch - creation_epoch))
    local age_days=$((age_seconds / 86400))
    
    echo "$age_days"
}

# Check if key rotation is needed
check_rotation_needed() {
    local secret_name="$1"
    local namespace="$2"
    
    local age_days
    age_days=$(get_secret_age "$secret_name" "$namespace")
    
    if [[ "$age_days" -ge "$ROTATION_INTERVAL_DAYS" ]]; then
        log_info "Key rotation needed for $secret_name (age: $age_days days, threshold: $ROTATION_INTERVAL_DAYS days)"
        return 0
    elif [[ "$age_days" -ge $((ROTATION_INTERVAL_DAYS - WARNING_DAYS)) ]]; then
        log_warn "Key rotation warning for $secret_name (age: $age_days days, will need rotation in $((ROTATION_INTERVAL_DAYS - age_days)) days)"
        return 1
    else
        log_info "Key rotation not needed for $secret_name (age: $age_days days)"
        return 1
    fi
}

# Backup current secret
backup_secret() {
    local secret_name="$1"
    local namespace="$2"
    local backup_suffix="backup-$(date +%Y%m%d-%H%M%S)"
    
    log_info "Creating backup of secret $secret_name..."
    
    if ! kubectl get secret "$secret_name" -n "$namespace" &> /dev/null; then
        log_warn "Secret $secret_name does not exist, skipping backup"
        return 0
    fi
    
    # Export current secret
    kubectl get secret "$secret_name" -n "$namespace" -o yaml > "/tmp/${secret_name}-${backup_suffix}.yaml"
    
    # Create backup secret in cluster
    kubectl get secret "$secret_name" -n "$namespace" -o json | \
        jq --arg new_name "${secret_name}-${backup_suffix}" '.metadata.name = $new_name | del(.metadata.resourceVersion) | del(.metadata.uid) | del(.metadata.creationTimestamp)' | \
        kubectl apply -f -
    
    log_info "Secret backup created: ${secret_name}-${backup_suffix}"
    echo "${secret_name}-${backup_suffix}"
}

# Rotate Weaviate API key
rotate_weaviate_api_key() {
    local force_rotation="${1:-false}"
    
    log_info "Starting Weaviate API key rotation..."
    
    # Check if rotation is needed
    if [[ "$force_rotation" != "true" ]] && ! check_rotation_needed "$WEAVIATE_SECRET_NAME" "$NAMESPACE"; then
        log_info "Weaviate API key rotation not needed"
        return 0
    fi
    
    # Backup current secret
    local backup_name
    backup_name=$(backup_secret "$WEAVIATE_SECRET_NAME" "$NAMESPACE")
    
    # Generate new API key
    local new_api_key
    new_api_key=$(generate_secure_key "$KEY_LENGTH" "alphanumeric")
    
    if [[ -z "$new_api_key" ]]; then
        log_error "Failed to generate new Weaviate API key"
        return 1
    fi
    
    log_info "Generated new Weaviate API key"
    
    # Create new secret
    kubectl create secret generic "$WEAVIATE_SECRET_NAME-new" \
        --from-literal=api-key="$new_api_key" \
        -n "$NAMESPACE" \
        --dry-run=client -o yaml | \
        kubectl apply -f -
    
    # Test new key by attempting to connect to Weaviate (if running)
    local weaviate_host="weaviate.$NAMESPACE.svc.cluster.local:8080"
    if kubectl get deployment weaviate -n "$NAMESPACE" &> /dev/null; then
        log_info "Testing new API key..."
        
        # Wait a moment for potential DNS propagation
        sleep 2
        
        # Simple connectivity test using port-forward
        local port_forward_pid=""
        kubectl port-forward svc/weaviate 18080:8080 -n "$NAMESPACE" &
        port_forward_pid=$!
        
        # Give port-forward time to establish
        sleep 3
        
        # Test the new key
        local test_result=false
        if curl -s -H "Authorization: Bearer $new_api_key" "http://localhost:18080/v1/.well-known/ready" | grep -q "true"; then
            test_result=true
        fi
        
        # Cleanup port-forward
        if [[ -n "$port_forward_pid" ]]; then
            kill "$port_forward_pid" 2>/dev/null || true
        fi
        
        if [[ "$test_result" == "true" ]]; then
            log_info "New API key test passed"
        else
            log_warn "New API key test failed, but proceeding with rotation"
        fi
    else
        log_info "Weaviate not running, skipping API key test"
    fi
    
    # Replace the original secret
    kubectl delete secret "$WEAVIATE_SECRET_NAME" -n "$NAMESPACE" --ignore-not-found=true
    kubectl create secret generic "$WEAVIATE_SECRET_NAME" \
        --from-literal=api-key="$new_api_key" \
        -n "$NAMESPACE"
    
    # Add metadata
    kubectl annotate secret "$WEAVIATE_SECRET_NAME" \
        "nephoran.com/rotated-at=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        "nephoran.com/rotated-by=key-rotation-script" \
        "nephoran.com/backup-secret=$backup_name" \
        -n "$NAMESPACE"
    
    # Cleanup temp secret
    kubectl delete secret "$WEAVIATE_SECRET_NAME-new" -n "$NAMESPACE" --ignore-not-found=true
    
    log_info "Weaviate API key rotation completed successfully"
    
    # Restart Weaviate deployment to pick up new key
    if kubectl get deployment weaviate -n "$NAMESPACE" &> /dev/null; then
        log_info "Restarting Weaviate deployment to pick up new API key..."
        kubectl rollout restart deployment/weaviate -n "$NAMESPACE"
        kubectl rollout status deployment/weaviate -n "$NAMESPACE" --timeout=300s
        log_info "Weaviate deployment restarted successfully"
    fi
    
    return 0
}

# Rotate backup encryption key
rotate_backup_encryption_key() {
    local force_rotation="${1:-false}"
    
    log_info "Starting backup encryption key rotation..."
    
    # Check if rotation is needed
    if [[ "$force_rotation" != "true" ]] && ! check_rotation_needed "$BACKUP_ENCRYPTION_SECRET" "$NAMESPACE"; then
        log_info "Backup encryption key rotation not needed"
        return 0
    fi
    
    # Backup current secret
    local backup_name
    backup_name=$(backup_secret "$BACKUP_ENCRYPTION_SECRET" "$NAMESPACE")
    
    # Generate new encryption key (256-bit for AES-256)
    local new_encryption_key
    new_encryption_key=$(generate_secure_key 64 "hex")
    
    if [[ -z "$new_encryption_key" ]]; then
        log_error "Failed to generate new backup encryption key"
        return 1
    fi
    
    log_info "Generated new backup encryption key"
    
    # Create new secret
    kubectl create secret generic "$BACKUP_ENCRYPTION_SECRET" \
        --from-literal=encryption-key="$new_encryption_key" \
        --from-literal=key-id="nephoran-backup-$(date +%Y%m%d-%H%M%S)" \
        -n "$NAMESPACE" \
        --dry-run=client -o yaml | \
        kubectl apply -f -
    
    # Add metadata
    kubectl annotate secret "$BACKUP_ENCRYPTION_SECRET" \
        "nephoran.com/rotated-at=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        "nephoran.com/rotated-by=key-rotation-script" \
        "nephoran.com/backup-secret=$backup_name" \
        "nephoran.com/key-algorithm=AES-256" \
        -n "$NAMESPACE"
    
    log_info "Backup encryption key rotation completed successfully"
    
    return 0
}

# Generate rotation report
generate_rotation_report() {
    local output_file="${1:-key-rotation-report.json}"
    
    log_info "Generating key rotation report: $output_file"
    
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # Get current key ages
    local weaviate_key_age
    local backup_key_age
    weaviate_key_age=$(get_secret_age "$WEAVIATE_SECRET_NAME" "$NAMESPACE")
    backup_key_age=$(get_secret_age "$BACKUP_ENCRYPTION_SECRET" "$NAMESPACE")
    
    # Get rotation annotations
    local weaviate_last_rotation
    local backup_last_rotation
    weaviate_last_rotation=$(kubectl get secret "$WEAVIATE_SECRET_NAME" -n "$NAMESPACE" -o jsonpath='{.metadata.annotations.nephoran\.com/rotated-at}' 2>/dev/null || echo "unknown")
    backup_last_rotation=$(kubectl get secret "$BACKUP_ENCRYPTION_SECRET" -n "$NAMESPACE" -o jsonpath='{.metadata.annotations.nephoran\.com/rotated-at}' 2>/dev/null || echo "unknown")
    
    # Create report
    cat > "$output_file" << EOF
{
    "key_rotation_report": {
        "timestamp": "$timestamp",
        "namespace": "$NAMESPACE",
        "rotation_interval_days": $ROTATION_INTERVAL_DAYS,
        "warning_days": $WARNING_DAYS,
        "keys": {
            "weaviate_api_key": {
                "secret_name": "$WEAVIATE_SECRET_NAME",
                "age_days": $weaviate_key_age,
                "last_rotation": "$weaviate_last_rotation",
                "needs_rotation": $(if [[ $weaviate_key_age -ge $ROTATION_INTERVAL_DAYS ]]; then echo "true"; else echo "false"; fi),
                "rotation_due_in_days": $((ROTATION_INTERVAL_DAYS - weaviate_key_age))
            },
            "backup_encryption_key": {
                "secret_name": "$BACKUP_ENCRYPTION_SECRET",
                "age_days": $backup_key_age,
                "last_rotation": "$backup_last_rotation",
                "needs_rotation": $(if [[ $backup_key_age -ge $ROTATION_INTERVAL_DAYS ]]; then echo "true"; else echo "false"; fi),
                "rotation_due_in_days": $((ROTATION_INTERVAL_DAYS - backup_key_age))
            }
        },
        "recommendations": [
            "Monitor key age regularly using this script",
            "Test backup restoration after encryption key rotation",
            "Keep backup secrets for disaster recovery scenarios",
            "Coordinate key rotation with maintenance windows"
        ]
    }
}
EOF
    
    log_info "Key rotation report generated: $output_file"
}

# Cleanup old backup secrets
cleanup_old_backups() {
    local retention_days="${1:-30}"
    
    log_info "Cleaning up backup secrets older than $retention_days days..."
    
    # Find old backup secrets
    local old_secrets
    old_secrets=$(kubectl get secrets -n "$NAMESPACE" -o json | \
        jq -r --arg threshold "$(date -d "$retention_days days ago" -u +%Y-%m-%dT%H:%M:%SZ)" \
        '.items[] | select(.metadata.name | test(".*-backup-[0-9]{8}-[0-9]{6}$")) | select(.metadata.creationTimestamp < $threshold) | .metadata.name')
    
    if [[ -z "$old_secrets" ]]; then
        log_info "No old backup secrets found for cleanup"
        return 0
    fi
    
    local count=0
    while IFS= read -r secret_name; do
        if [[ -n "$secret_name" ]]; then
            log_info "Deleting old backup secret: $secret_name"
            kubectl delete secret "$secret_name" -n "$NAMESPACE"
            ((count++))
        fi
    done <<< "$old_secrets"
    
    log_info "Cleaned up $count old backup secrets"
}

# Main function
main() {
    local command="${1:-check}"
    local force_rotation="${2:-false}"
    
    case "$command" in
        "check")
            log_info "Checking key rotation status..."
            check_prerequisites
            
            check_rotation_needed "$WEAVIATE_SECRET_NAME" "$NAMESPACE" || true
            check_rotation_needed "$BACKUP_ENCRYPTION_SECRET" "$NAMESPACE" || true
            
            generate_rotation_report
            ;;
            
        "rotate")
            log_info "Starting key rotation process..."
            check_prerequisites
            
            rotate_weaviate_api_key "$force_rotation"
            rotate_backup_encryption_key "$force_rotation"
            
            generate_rotation_report
            log_info "Key rotation process completed"
            ;;
            
        "rotate-weaviate")
            log_info "Rotating Weaviate API key only..."
            check_prerequisites
            rotate_weaviate_api_key "$force_rotation"
            ;;
            
        "rotate-backup")
            log_info "Rotating backup encryption key only..."
            check_prerequisites
            rotate_backup_encryption_key "$force_rotation"
            ;;
            
        "cleanup")
            local retention_days="${2:-30}"
            log_info "Cleaning up old backup secrets..."
            check_prerequisites
            cleanup_old_backups "$retention_days"
            ;;
            
        "report")
            check_prerequisites
            generate_rotation_report "${2:-key-rotation-report.json}"
            ;;
            
        "help"|"--help"|"-h")
            cat << EOF
Weaviate Encryption Key Rotation Script

USAGE:
    $0 [COMMAND] [OPTIONS]

COMMANDS:
    check                 - Check key rotation status
    rotate [force]        - Rotate all keys (use 'force' to force rotation)
    rotate-weaviate [force] - Rotate only Weaviate API key
    rotate-backup [force]   - Rotate only backup encryption key
    cleanup [DAYS]        - Cleanup old backup secrets (default: 30 days)
    report [FILE]         - Generate rotation status report
    help                  - Show this help message

ENVIRONMENT VARIABLES:
    NAMESPACE             - Kubernetes namespace (default: nephoran-system)
    WEAVIATE_SECRET_NAME  - Weaviate API key secret name (default: weaviate-api-key)
    BACKUP_ENCRYPTION_SECRET - Backup encryption secret name (default: weaviate-backup-encryption)
    ROTATION_INTERVAL_DAYS - Days between rotations (default: 90)
    WARNING_DAYS          - Days before expiry to warn (default: 7)
    KEY_LENGTH            - API key length (default: 32)
    DEBUG                 - Enable debug logging (default: false)

EXAMPLES:
    $0 check                    # Check rotation status
    $0 rotate                   # Rotate keys if needed
    $0 rotate force             # Force key rotation
    $0 rotate-weaviate          # Rotate only Weaviate API key
    $0 cleanup 14               # Cleanup backups older than 14 days
    $0 report my-report.json    # Generate custom report

EOF
            ;;
            
        *)
            log_error "Unknown command: $command"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Setup logging
mkdir -p "$(dirname "$ROTATION_LOG_FILE")"
log_info "Key rotation script started with command: ${1:-check}"

# Run main function with all arguments
main "$@"

log_info "Key rotation script completed"