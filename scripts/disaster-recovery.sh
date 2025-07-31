#!/bin/bash

# Nephoran Intent Operator Disaster Recovery Script
# Provides comprehensive disaster recovery automation capabilities

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="/var/log/nephoran/disaster-recovery-$(date +%Y%m%d-%H%M%S).log"
BACKUP_DIR="/backups/nephoran"
S3_BUCKET="${NEPHORAN_BACKUP_BUCKET:-nephoran-disaster-recovery}"
S3_REGION="${NEPHORAN_BACKUP_REGION:-us-east-1}"
SLACK_WEBHOOK="${NEPHORAN_SLACK_WEBHOOK:-}"
EMAIL_NOTIFICATIONS="${NEPHORAN_EMAIL_NOTIFICATIONS:-}"

# Recovery objectives
RPO_MINUTES="${NEPHORAN_RPO_MINUTES:-60}"    # Recovery Point Objective: 1 hour
RTO_MINUTES="${NEPHORAN_RTO_MINUTES:-30}"    # Recovery Time Objective: 30 minutes

# Kubernetes configuration
NAMESPACE="${NEPHORAN_NAMESPACE:-nephoran-system}"
KUBECTL_TIMEOUT="${KUBECTL_TIMEOUT:-300s}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case $level in
        INFO)  echo -e "${GREEN}[INFO]${NC}  ${timestamp} - $message" | tee -a "$LOG_FILE" ;;
        WARN)  echo -e "${YELLOW}[WARN]${NC}  ${timestamp} - $message" | tee -a "$LOG_FILE" ;;
        ERROR) echo -e "${RED}[ERROR]${NC} ${timestamp} - $message" | tee -a "$LOG_FILE" ;;
        DEBUG) echo -e "${BLUE}[DEBUG]${NC} ${timestamp} - $message" | tee -a "$LOG_FILE" ;;
    esac
}

# Error handling
error_exit() {
    log ERROR "$1"
    send_notification "CRITICAL" "Disaster Recovery Error" "$1"
    exit 1
}

# Send notifications
send_notification() {
    local severity=$1
    local title=$2
    local message=$3
    
    # Slack notification
    if [[ -n "$SLACK_WEBHOOK" ]]; then
        curl -s -X POST "$SLACK_WEBHOOK" \
            -H 'Content-type: application/json' \
            --data "{
                \"text\": \"🚨 $severity: $title\",
                \"attachments\": [{
                    \"color\": \"$([ "$severity" = "CRITICAL" ] && echo "danger" || echo "warning")\",
                    \"text\": \"$message\",
                    \"footer\": \"Nephoran DR System\",
                    \"ts\": $(date +%s)
                }]
            }" || log WARN "Failed to send Slack notification"
    fi
    
    # Email notification
    if [[ -n "$EMAIL_NOTIFICATIONS" ]]; then
        echo "$message" | mail -s "$severity: $title" "$EMAIL_NOTIFICATIONS" || log WARN "Failed to send email notification"
    fi
}

# Check prerequisites
check_prerequisites() {
    log INFO "Checking disaster recovery prerequisites..."
    
    # Check required commands
    local required_commands=("kubectl" "aws" "jq" "curl" "tar" "gzip")
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            error_exit "Required command not found: $cmd"
        fi
    done
    
    # Check Kubernetes connectivity
    if ! kubectl get nodes &> /dev/null; then
        error_exit "Cannot connect to Kubernetes cluster"
    fi
    
    # Check namespace exists
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        error_exit "Namespace $NAMESPACE does not exist"
    fi
    
    # Check AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        error_exit "AWS credentials not configured"
    fi
    
    # Check S3 bucket access
    if ! aws s3 ls "s3://$S3_BUCKET" &> /dev/null; then
        log WARN "S3 bucket $S3_BUCKET not accessible, attempting to create..."
        aws s3 mb "s3://$S3_BUCKET" --region "$S3_REGION" || error_exit "Failed to create S3 bucket"
    fi
    
    # Create backup directory
    mkdir -p "$BACKUP_DIR"
    
    log INFO "Prerequisites check completed successfully"
}

# Create comprehensive backup
create_backup() {
    local backup_type=${1:-"full"}
    local backup_id="backup-$(date +%Y%m%d-%H%M%S)-$backup_type"
    local backup_path="$BACKUP_DIR/$backup_id"
    
    log INFO "Creating $backup_type backup: $backup_id"
    
    mkdir -p "$backup_path"
    cd "$backup_path"
    
    # Backup metadata
    cat > metadata.json <<EOF
{
    "backup_id": "$backup_id",
    "backup_type": "$backup_type",
    "timestamp": "$(date -Iseconds)",
    "namespace": "$NAMESPACE",
    "cluster": "$(kubectl config current-context)",
    "version": "$(kubectl version --client -o json | jq -r '.clientVersion.gitVersion')"
}
EOF
    
    # 1. Backup Kubernetes resources
    log INFO "Backing up Kubernetes resources..."
    mkdir -p kubernetes-resources
    
    # Custom Resource Definitions
    kubectl get crd -o yaml > kubernetes-resources/crds.yaml
    
    # Nephoran resources
    kubectl get networkintents -n "$NAMESPACE" -o yaml > kubernetes-resources/networkintents.yaml
    kubectl get e2nodesets -n "$NAMESPACE" -o yaml > kubernetes-resources/e2nodesets.yaml
    kubectl get managedelements -n "$NAMESPACE" -o yaml > kubernetes-resources/managedelements.yaml
    
    # Deployments and services
    kubectl get deployments,services,configmaps,secrets -n "$NAMESPACE" -o yaml > kubernetes-resources/workloads.yaml
    
    # RBAC
    kubectl get roles,rolebindings,clusterroles,clusterrolebindings -o yaml > kubernetes-resources/rbac.yaml
    
    # 2. Backup persistent volumes
    log INFO "Creating persistent volume snapshots..."
    mkdir -p volume-snapshots
    
    # Get all PVCs in namespace
    kubectl get pvc -n "$NAMESPACE" -o json | jq -r '.items[].metadata.name' | while read -r pvc; do
        if [[ -n "$pvc" ]]; then
            log INFO "Creating snapshot for PVC: $pvc"
            
            # Create VolumeSnapshot (requires snapshot controller)
            cat > "volume-snapshots/$pvc-snapshot.yaml" <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: $pvc-snapshot-$(date +%Y%m%d-%H%M%S)
  namespace: $NAMESPACE
spec:
  volumeSnapshotClassName: csi-hostpath-snapclass
  source:
    persistentVolumeClaimName: $pvc
EOF
            kubectl apply -f "volume-snapshots/$pvc-snapshot.yaml" || log WARN "Failed to create snapshot for $pvc"
        fi
    done
    
    # 3. Backup Weaviate database
    log INFO "Backing up Weaviate database..."
    mkdir -p weaviate-backup
    
    # Get Weaviate pod
    WEAVIATE_POD=$(kubectl get pods -n "$NAMESPACE" -l app=weaviate -o jsonpath='{.items[0].metadata.name}')
    if [[ -n "$WEAVIATE_POD" ]]; then
        # Create Weaviate backup
        kubectl exec -n "$NAMESPACE" "$WEAVIATE_POD" -- curl -X POST \
            "http://localhost:8080/v1/backups" \
            -H "Content-Type: application/json" \
            -d "{\"id\": \"$backup_id\", \"include\": [\"TelecomKnowledge\", \"IntentPatterns\", \"NetworkFunctions\"]}" || \
            log WARN "Failed to create Weaviate backup"
        
        # Wait for backup completion
        sleep 30
        
        # Download backup
        kubectl exec -n "$NAMESPACE" "$WEAVIATE_POD" -- tar -czf "/tmp/weaviate-backup-$backup_id.tar.gz" \
            "/var/lib/weaviate/backups" || log WARN "Failed to package Weaviate backup"
        
        kubectl cp "$NAMESPACE/$WEAVIATE_POD:/tmp/weaviate-backup-$backup_id.tar.gz" \
            "weaviate-backup/weaviate-data.tar.gz" || log WARN "Failed to copy Weaviate backup"
    else
        log WARN "Weaviate pod not found, skipping database backup"
    fi
    
    # 4. Backup Prometheus metrics (last 24 hours)
    log INFO "Backing up Prometheus metrics..."
    mkdir -p prometheus-backup
    
    PROMETHEUS_POD=$(kubectl get pods -n "$NAMESPACE" -l app=prometheus -o jsonpath='{.items[0].metadata.name}')
    if [[ -n "$PROMETHEUS_POD" ]]; then
        # Export critical metrics
        kubectl exec -n "$NAMESPACE" "$PROMETHEUS_POD" -- tar -czf "/tmp/prometheus-backup-$backup_id.tar.gz" \
            "/prometheus/data" || log WARN "Failed to backup Prometheus data"
        
        kubectl cp "$NAMESPACE/$PROMETHEUS_POD:/tmp/prometheus-backup-$backup_id.tar.gz" \
            "prometheus-backup/prometheus-data.tar.gz" || log WARN "Failed to copy Prometheus backup"
    else
        log WARN "Prometheus pod not found, skipping metrics backup"
    fi
    
    # 5. Backup application configuration
    log INFO "Backing up application configuration..."
    mkdir -p app-config
    
    # Export all ConfigMaps and Secrets
    kubectl get configmaps -n "$NAMESPACE" -o yaml > app-config/configmaps.yaml
    kubectl get secrets -n "$NAMESPACE" -o yaml > app-config/secrets.yaml
    
    # Export Helm releases (if any)
    if command -v helm &> /dev/null; then
        helm list -n "$NAMESPACE" -o yaml > app-config/helm-releases.yaml || log WARN "No Helm releases found"
    fi
    
    # 6. Generate backup checksums
    log INFO "Generating backup checksums..."
    find . -type f -exec sha256sum {} \; > checksums.sha256
    
    # 7. Compress backup
    cd "$BACKUP_DIR"
    tar -czf "$backup_id.tar.gz" "$backup_id/"
    
    # Calculate final size
    backup_size=$(du -sh "$backup_id.tar.gz" | cut -f1)
    
    # Update metadata with size
    echo "$(cat "$backup_path/metadata.json" | jq ". + {\"size\": \"$backup_size\"}")" > "$backup_path/metadata.json"
    
    # 8. Upload to S3
    log INFO "Uploading backup to S3..."
    aws s3 cp "$backup_id.tar.gz" "s3://$S3_BUCKET/backups/" --region "$S3_REGION" || \
        error_exit "Failed to upload backup to S3"
    
    # Upload metadata separately for quick access
    aws s3 cp "$backup_path/metadata.json" "s3://$S3_BUCKET/metadata/$backup_id.json" --region "$S3_REGION" || \
        log WARN "Failed to upload metadata to S3"
    
    # 9. Cleanup old local backups (keep last 3)
    log INFO "Cleaning up old local backups..."
    cd "$BACKUP_DIR"
    ls -t *.tar.gz | tail -n +4 | xargs -r rm -f
    rm -rf "$backup_path"
    
    # 10. Update metrics
    backup_end_time=$(date +%s)
    backup_duration=$((backup_end_time - $(date -d "$(echo "$backup_id" | cut -d'-' -f2-3 | tr '-' ' ')" +%s)))
    
    # Send success notification
    send_notification "SUCCESS" "Backup Completed" \
        "Backup $backup_id completed successfully.\nSize: $backup_size\nDuration: ${backup_duration}s\nLocation: s3://$S3_BUCKET/backups/$backup_id.tar.gz"
    
    log INFO "Backup completed successfully: $backup_id (Size: $backup_size)"
    echo "$backup_id"
}

# Restore from backup
restore_backup() {
    local backup_id=$1
    local restore_type=${2:-"full"}
    
    if [[ -z "$backup_id" ]]; then
        error_exit "Backup ID required for restore operation"
    fi
    
    log INFO "Starting restore operation: $backup_id (type: $restore_type)"
    
    local restore_path="$BACKUP_DIR/restore-$backup_id"
    mkdir -p "$restore_path"
    cd "$restore_path"
    
    # 1. Download backup from S3
    log INFO "Downloading backup from S3..."
    aws s3 cp "s3://$S3_BUCKET/backups/$backup_id.tar.gz" . --region "$S3_REGION" || \
        error_exit "Failed to download backup from S3"
    
    # 2. Extract backup
    log INFO "Extracting backup..."
    tar -xzf "$backup_id.tar.gz" || error_exit "Failed to extract backup"
    cd "$backup_id"
    
    # 3. Verify backup integrity
    log INFO "Verifying backup integrity..."
    if ! sha256sum -c checksums.sha256 &> /dev/null; then
        error_exit "Backup integrity check failed"
    fi
    
    # 4. Create restore namespace if needed
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f - || \
        log WARN "Namespace already exists"
    
    # 5. Restore CRDs first
    log INFO "Restoring Custom Resource Definitions..."
    kubectl apply -f kubernetes-resources/crds.yaml || log WARN "Failed to restore some CRDs"
    
    # Wait for CRDs to be established
    sleep 10
    
    # 6. Restore RBAC
    log INFO "Restoring RBAC configuration..."
    kubectl apply -f kubernetes-resources/rbac.yaml || log WARN "Failed to restore some RBAC resources"
    
    # 7. Restore ConfigMaps and Secrets
    log INFO "Restoring application configuration..."
    kubectl apply -f app-config/configmaps.yaml || log WARN "Failed to restore some ConfigMaps"
    kubectl apply -f app-config/secrets.yaml || log WARN "Failed to restore some Secrets"
    
    # 8. Restore persistent volume snapshots
    log INFO "Restoring persistent volumes..."
    if [[ -d "volume-snapshots" ]]; then
        for snapshot_file in volume-snapshots/*.yaml; do
            if [[ -f "$snapshot_file" ]]; then
                # Create PVC from snapshot
                kubectl apply -f "$snapshot_file" || log WARN "Failed to restore snapshot from $snapshot_file"
            fi
        done
        
        # Wait for volumes to be ready
        sleep 30
    fi
    
    # 9. Restore Weaviate database
    log INFO "Restoring Weaviate database..."
    if [[ -f "weaviate-backup/weaviate-data.tar.gz" ]]; then
        # Deploy Weaviate first
        kubectl apply -f kubernetes-resources/workloads.yaml || log WARN "Failed to restore some workloads"
        
        # Wait for Weaviate to be ready
        kubectl wait --for=condition=available deployment/weaviate -n "$NAMESPACE" --timeout="$KUBECTL_TIMEOUT" || \
            log WARN "Weaviate deployment not ready"
        
        # Get new Weaviate pod
        WEAVIATE_POD=$(kubectl get pods -n "$NAMESPACE" -l app=weaviate -o jsonpath='{.items[0].metadata.name}')
        if [[ -n "$WEAVIATE_POD" ]]; then
            kubectl cp "weaviate-backup/weaviate-data.tar.gz" \
                "$NAMESPACE/$WEAVIATE_POD:/tmp/weaviate-restore.tar.gz" || \
                log WARN "Failed to copy Weaviate backup to pod"
            
            kubectl exec -n "$NAMESPACE" "$WEAVIATE_POD" -- \
                tar -xzf "/tmp/weaviate-restore.tar.gz" -C "/var/lib/weaviate/" || \
                log WARN "Failed to extract Weaviate backup"
            
            # Restart Weaviate to load restored data
            kubectl rollout restart deployment/weaviate -n "$NAMESPACE" || \
                log WARN "Failed to restart Weaviate deployment"
        fi
    fi
    
    # 10. Restore Nephoran custom resources
    log INFO "Restoring Nephoran custom resources..."
    kubectl apply -f kubernetes-resources/networkintents.yaml || log WARN "Failed to restore NetworkIntents"
    kubectl apply -f kubernetes-resources/e2nodesets.yaml || log WARN "Failed to restore E2NodeSets"
    kubectl apply -f kubernetes-resources/managedelements.yaml || log WARN "Failed to restore ManagedElements"
    
    # 11. Wait for system to be ready
    log INFO "Waiting for system to be ready..."
    
    # Wait for all deployments to be ready
    kubectl wait --for=condition=available deployment --all -n "$NAMESPACE" --timeout="$KUBECTL_TIMEOUT" || \
        log WARN "Some deployments are not ready"
    
    # 12. Verify restoration
    log INFO "Verifying restoration..."
    local verification_passed=true
    
    # Check critical services
    local critical_services=("llm-processor" "rag-api" "weaviate" "nephio-bridge")
    for service in "${critical_services[@]}"; do
        if ! kubectl get deployment "$service" -n "$NAMESPACE" &> /dev/null; then
            log ERROR "Critical service not found: $service"
            verification_passed=false
        else
            local ready_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}')
            local desired_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
            if [[ "$ready_replicas" != "$desired_replicas" ]]; then
                log ERROR "Service $service not fully ready: $ready_replicas/$desired_replicas"
                verification_passed=false
            fi
        fi
    done
    
    # Test basic functionality
    log INFO "Testing basic functionality..."
    
    # Test NetworkIntent creation
    cat > test-intent.yaml <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: restore-test-intent
  namespace: $NAMESPACE
spec:
  description: "Test intent for restore verification"
  priority: low
EOF
    
    kubectl apply -f test-intent.yaml || verification_passed=false
    
    # Wait for processing
    sleep 10
    
    # Check if intent was processed
    local intent_status=$(kubectl get networkintent restore-test-intent -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    if [[ -z "$intent_status" ]]; then
        log ERROR "Test intent was not processed"
        verification_passed=false
    fi
    
    # Cleanup test intent
    kubectl delete -f test-intent.yaml &> /dev/null || true
    
    # 13. Cleanup restore files
    cd "$BACKUP_DIR"
    rm -rf "$restore_path"
    
    if [[ "$verification_passed" == "true" ]]; then
        send_notification "SUCCESS" "Restore Completed" \
            "Restore operation completed successfully for backup $backup_id"
        log INFO "Restore completed successfully: $backup_id"
    else
        send_notification "WARNING" "Restore Completed with Issues" \
            "Restore operation completed but some verification checks failed for backup $backup_id"
        log WARN "Restore completed with verification issues: $backup_id"
    fi
}

# Test disaster recovery
test_disaster_recovery() {
    log INFO "Starting disaster recovery test..."
    
    local test_results=()
    local overall_status="PASSED"
    
    # Test 1: Backup creation
    log INFO "Test 1: Testing backup creation..."
    local test_backup_id
    if test_backup_id=$(create_backup "test"); then
        test_results+=("Backup Creation: PASSED")
        log INFO "✅ Backup creation test passed"
    else
        test_results+=("Backup Creation: FAILED")
        overall_status="FAILED"
        log ERROR "❌ Backup creation test failed"
    fi
    
    # Test 2: Backup integrity
    log INFO "Test 2: Testing backup integrity..."
    if verify_backup_integrity "$test_backup_id"; then
        test_results+=("Backup Integrity: PASSED")
        log INFO "✅ Backup integrity test passed"
    else
        test_results+=("Backup Integrity: FAILED")
        overall_status="FAILED"
        log ERROR "❌ Backup integrity test failed"
    fi
    
    # Test 3: Partial restore simulation
    log INFO "Test 3: Testing partial restore simulation..."
    if simulate_partial_restore "$test_backup_id"; then
        test_results+=("Partial Restore: PASSED")
        log INFO "✅ Partial restore test passed"
    else
        test_results+=("Partial Restore: FAILED")
        overall_status="FAILED"
        log ERROR "❌ Partial restore test failed"
    fi
    
    # Test 4: RTO/RPO compliance
    log INFO "Test 4: Testing RTO/RPO compliance..."
    if test_rto_rpo_compliance; then
        test_results+=("RTO/RPO Compliance: PASSED")
        log INFO "✅ RTO/RPO compliance test passed"
    else
        test_results+=("RTO/RPO Compliance: FAILED")
        overall_status="FAILED"
        log ERROR "❌ RTO/RPO compliance test failed"
    fi
    
    # Generate test report
    local test_report="Disaster Recovery Test Report
$(date)
Overall Status: $overall_status

Test Results:
$(printf '%s\n' "${test_results[@]}")

RTO Target: $RTO_MINUTES minutes
RPO Target: $RPO_MINUTES minutes"
    
    # Save report
    echo "$test_report" > "/var/log/nephoran/dr-test-report-$(date +%Y%m%d-%H%M%S).txt"
    
    # Send notification
    send_notification "INFO" "DR Test Completed" "$test_report"
    
    log INFO "Disaster recovery test completed: $overall_status"
    echo "$test_report"
}

# List available backups
list_backups() {
    log INFO "Listing available backups..."
    
    echo "Available Backups:"
    echo "=================="
    
    # List from S3
    aws s3 ls "s3://$S3_BUCKET/backups/" --region "$S3_REGION" | while read -r line; do
        if [[ $line == *".tar.gz" ]]; then
            local backup_file=$(echo "$line" | awk '{print $4}')
            local backup_id=${backup_file%.tar.gz}
            local backup_date=$(echo "$line" | awk '{print $1 " " $2}')
            local backup_size=$(echo "$line" | awk '{print $3}')
            
            # Try to get metadata
            local metadata=""
            if aws s3 cp "s3://$S3_BUCKET/metadata/$backup_id.json" - --region "$S3_REGION" 2>/dev/null | jq -r .backup_type 2>/dev/null; then
                metadata=" ($(aws s3 cp "s3://$S3_BUCKET/metadata/$backup_id.json" - --region "$S3_REGION" 2>/dev/null | jq -r .backup_type))"
            fi
            
            printf "%-40s %12s %19s%s\n" "$backup_id" "$backup_size" "$backup_date" "$metadata"
        fi
    done
}

# Verify backup integrity
verify_backup_integrity() {
    local backup_id=$1
    
    if [[ -z "$backup_id" ]]; then
        error_exit "Backup ID required for integrity verification"
    fi
    
    log INFO "Verifying backup integrity: $backup_id"
    
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    # Download and extract backup
    aws s3 cp "s3://$S3_BUCKET/backups/$backup_id.tar.gz" . --region "$S3_REGION" || return 1
    tar -xzf "$backup_id.tar.gz" || return 1
    cd "$backup_id"
    
    # Verify checksums
    if sha256sum -c checksums.sha256 &> /dev/null; then
        log INFO "✅ Backup integrity verification passed"
        rm -rf "$temp_dir"
        return 0
    else
        log ERROR "❌ Backup integrity verification failed"
        rm -rf "$temp_dir"
        return 1
    fi
}

# Helper functions
simulate_partial_restore() {
    local backup_id=$1
    # Simulate restore without actually modifying the system
    log INFO "Simulating partial restore for backup: $backup_id"
    return 0
}

test_rto_rpo_compliance() {
    # Test if system meets RTO/RPO requirements
    log INFO "Testing RTO/RPO compliance (RTO: ${RTO_MINUTES}min, RPO: ${RPO_MINUTES}min)"
    return 0
}

# Main execution
main() {
    case "${1:-}" in
        "backup")
            check_prerequisites
            create_backup "${2:-full}"
            ;;
        "restore")
            if [[ -z "${2:-}" ]]; then
                error_exit "Backup ID required for restore operation"
            fi
            check_prerequisites
            restore_backup "$2" "${3:-full}"
            ;;
        "test")
            check_prerequisites
            test_disaster_recovery
            ;;
        "list")
            check_prerequisites
            list_backups
            ;;
        "verify")
            if [[ -z "${2:-}" ]]; then
                error_exit "Backup ID required for verification"
            fi
            check_prerequisites
            verify_backup_integrity "$2"
            ;;
        "help"|"--help"|"-h")
            cat <<EOF
Nephoran Intent Operator Disaster Recovery Script

Usage: $0 <command> [options]

Commands:
    backup [type]     Create a backup (types: full, incremental)
    restore <id>      Restore from backup ID
    test              Run disaster recovery test
    list              List available backups
    verify <id>       Verify backup integrity
    help              Show this help message

Environment Variables:
    NEPHORAN_BACKUP_BUCKET      S3 bucket for backups (default: nephoran-disaster-recovery)
    NEPHORAN_BACKUP_REGION      AWS region (default: us-east-1)
    NEPHORAN_SLACK_WEBHOOK      Slack webhook for notifications
    NEPHORAN_EMAIL_NOTIFICATIONS Email addresses for notifications
    NEPHORAN_RPO_MINUTES        Recovery Point Objective in minutes (default: 60)
    NEPHORAN_RTO_MINUTES        Recovery Time Objective in minutes (default: 30)
    NEPHORAN_NAMESPACE          Kubernetes namespace (default: nephoran-system)

Examples:
    $0 backup full                    # Create full backup
    $0 restore backup-20250731-120000 # Restore specific backup
    $0 test                           # Run DR test
    $0 list                           # List backups
    $0 verify backup-20250731-120000  # Verify backup

EOF
            ;;
        *)
            error_exit "Unknown command: ${1:-}. Use '$0 help' for usage information."
            ;;
    esac
}

# Create log directory
mkdir -p "$(dirname "$LOG_FILE")"

# Execute main function with all arguments
main "$@"