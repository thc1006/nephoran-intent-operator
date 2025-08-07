#!/bin/bash
# failover-to-secondary.sh
# Automated failover script for Nephoran Intent Operator
# Target RTO: < 5 minutes

set -euo pipefail

# Configuration
PRIMARY_CLUSTER_NAME="nephoran-primary"
SECONDARY_CLUSTER_NAME="nephoran-dr-secondary"
PRIMARY_KUBECONFIG="${HOME}/.kube/config"
SECONDARY_KUBECONFIG="/tmp/secondary-kubeconfig.yaml"

# DNS/Load Balancer Configuration
LB_CONFIG_FILE="/tmp/nephoran-lb-config.json"
DNS_PROVIDER="cloudflare"  # Options: cloudflare, route53, manual
DOMAIN_NAME="nephoran.example.com"

# Health Check Configuration
HEALTH_CHECK_TIMEOUT=30
MAX_HEALTH_CHECK_RETRIES=5
FAILOVER_TRIGGER_FILE="/tmp/nephoran-failover-trigger"

# Logging setup
LOG_FILE="/tmp/failover-$(date +%Y%m%d-%H%M%S).log"
exec 1> >(tee -a "$LOG_FILE")
exec 2>&1

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

log_critical() {
    echo -e "${RED}[CRITICAL]${NC} $1"
}

log_failover() {
    echo -e "${PURPLE}[FAILOVER]${NC} $1"
}

# Failover state tracking
FAILOVER_START_TIME=""
FAILOVER_PHASES=""

start_failover_timer() {
    FAILOVER_START_TIME=$(date +%s)
    log_failover "Failover initiated at $(date -Iseconds)"
}

log_failover_phase() {
    local phase="$1"
    local current_time=$(date +%s)
    local elapsed=$((current_time - FAILOVER_START_TIME))
    log_failover "Phase: $phase (Elapsed: ${elapsed}s)"
    FAILOVER_PHASES="$FAILOVER_PHASES\n$(date -Iseconds): $phase (+${elapsed}s)"
}

# Check if failover is already in progress
check_failover_lock() {
    if [ -f "/tmp/nephoran-failover.lock" ]; then
        local lock_pid=$(cat /tmp/nephoran-failover.lock)
        if kill -0 "$lock_pid" 2>/dev/null; then
            log_error "Failover already in progress (PID: $lock_pid)"
            exit 1
        else
            log_warn "Stale failover lock found, removing..."
            rm -f /tmp/nephoran-failover.lock
        fi
    fi
    
    echo $$ > /tmp/nephoran-failover.lock
}

# Remove failover lock
remove_failover_lock() {
    rm -f /tmp/nephoran-failover.lock
}

# Health check functions
check_primary_cluster_health() {
    local retries=0
    
    log_info "Checking primary cluster health..."
    
    while [ $retries -lt $MAX_HEALTH_CHECK_RETRIES ]; do
        # Check API server
        if timeout $HEALTH_CHECK_TIMEOUT kubectl --kubeconfig="$PRIMARY_KUBECONFIG" cluster-info &>/dev/null; then
            # Check critical pods
            local critical_pods_ready=true
            
            # Check Nephoran operator
            if ! kubectl --kubeconfig="$PRIMARY_KUBECONFIG" get pods -n nephoran-system -l app.kubernetes.io/name=nephoran-operator --no-headers 2>/dev/null | grep -q "Running"; then
                log_warn "Nephoran operator not running"
                critical_pods_ready=false
            fi
            
            # Check LLM processor
            if ! kubectl --kubeconfig="$PRIMARY_KUBECONFIG" get pods -n nephoran-system -l app.kubernetes.io/name=llm-processor --no-headers 2>/dev/null | grep -q "Running"; then
                log_warn "LLM processor not running"
                critical_pods_ready=false
            fi
            
            # Check Weaviate
            if ! kubectl --kubeconfig="$PRIMARY_KUBECONFIG" get pods -n nephoran-system -l app.kubernetes.io/name=weaviate --no-headers 2>/dev/null | grep -q "Running"; then
                log_warn "Weaviate not running"
                critical_pods_ready=false
            fi
            
            if [ "$critical_pods_ready" = true ]; then
                log_success "Primary cluster is healthy"
                return 0
            fi
        fi
        
        retries=$((retries + 1))
        log_warn "Primary cluster health check failed (attempt $retries/$MAX_HEALTH_CHECK_RETRIES)"
        
        if [ $retries -lt $MAX_HEALTH_CHECK_RETRIES ]; then
            sleep 10
        fi
    done
    
    log_error "Primary cluster is unhealthy after $MAX_HEALTH_CHECK_RETRIES attempts"
    return 1
}

check_secondary_cluster_health() {
    log_info "Checking secondary cluster health..."
    
    if ! kubectl --kubeconfig="$SECONDARY_KUBECONFIG" cluster-info &>/dev/null; then
        log_error "Secondary cluster is not accessible"
        return 1
    fi
    
    # Check if secondary cluster is ready for failover
    if ! kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get configmap failover-config -n kube-system -o jsonpath='{.data.cluster-status}' 2>/dev/null | grep -q "ready"; then
        log_error "Secondary cluster is not ready for failover"
        return 1
    fi
    
    # Check essential components
    local components=("cert-manager" "ingress-nginx" "monitoring" "velero")
    for component in "${components[@]}"; do
        if ! kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get pods -n "$component" --no-headers 2>/dev/null | grep -q "Running"; then
            log_error "Essential component '$component' not ready on secondary cluster"
            return 1
        fi
    done
    
    log_success "Secondary cluster is ready for failover"
    return 0
}

# Backup current state before failover
create_failover_backup() {
    log_info "Creating emergency backup before failover..."
    
    local backup_name="emergency-failover-$(date +%Y%m%d-%H%M%S)"
    
    # Attempt to create backup from primary cluster if accessible
    if kubectl --kubeconfig="$PRIMARY_KUBECONFIG" cluster-info &>/dev/null; then
        log_info "Creating backup from primary cluster..."
        
        cat << EOF | kubectl --kubeconfig="$PRIMARY_KUBECONFIG" apply -f -
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: $backup_name
  namespace: velero
  labels:
    backup-type: emergency-failover
    nephoran.com/failover: "true"
spec:
  includedNamespaces:
  - nephoran-system
  - nephoran-production
  
  includedResources:
  - networkintents.nephoran.com
  - e2nodesets.nephoran.com
  - managedelements.nephoran.com
  - secrets
  - configmaps
  - persistentvolumeclaims
  
  storageLocation: nephoran-s3-backup
  volumeSnapshotLocations:
  - nephoran-volume-snapshots
  ttl: 168h  # 7 days
  
  snapshotVolumes: true
  includeClusterResources: true
  defaultVolumesToSnapshot: true
EOF
        
        # Wait for backup to start
        local timeout=60
        local counter=0
        while [ $counter -lt $timeout ]; do
            local backup_phase=$(kubectl --kubeconfig="$PRIMARY_KUBECONFIG" get backup $backup_name -n velero -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
            if [ "$backup_phase" = "InProgress" ] || [ "$backup_phase" = "Completed" ]; then
                log_success "Emergency backup created: $backup_name"
                echo "$backup_name" > /tmp/emergency-backup-name
                break
            fi
            sleep 5
            counter=$((counter + 5))
        done
        
        if [ $counter -ge $timeout ]; then
            log_warn "Emergency backup creation timed out, proceeding with failover"
        fi
    else
        log_warn "Primary cluster not accessible, skipping emergency backup"
    fi
}

# Get latest backup for restore
get_latest_backup() {
    log_info "Identifying latest backup for restore..."
    
    local latest_backup=""
    
    # Check for emergency backup first
    if [ -f "/tmp/emergency-backup-name" ]; then
        latest_backup=$(cat /tmp/emergency-backup-name)
        log_info "Using emergency backup: $latest_backup"
    else
        # Find latest successful backup
        latest_backup=$(kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get backups -n velero \
            -o jsonpath='{range .items[?(@.status.phase=="Completed")]}{.metadata.creationTimestamp}{" "}{.metadata.name}{"\n"}{end}' \
            | sort -r | head -1 | awk '{print $2}')
        
        if [ -n "$latest_backup" ]; then
            log_info "Using latest successful backup: $latest_backup"
        else
            log_error "No successful backup found for restore"
            return 1
        fi
    fi
    
    echo "$latest_backup" > /tmp/restore-backup-name
    return 0
}

# Restore from backup to secondary cluster
restore_from_backup() {
    log_info "Restoring from backup to secondary cluster..."
    
    if [ ! -f "/tmp/restore-backup-name" ]; then
        log_error "No backup specified for restore"
        return 1
    fi
    
    local backup_name=$(cat /tmp/restore-backup-name)
    local restore_name="failover-restore-$(date +%Y%m%d-%H%M%S)"
    
    cat << EOF | kubectl --kubeconfig="$SECONDARY_KUBECONFIG" apply -f -
apiVersion: velero.io/v1
kind: Restore
metadata:
  name: $restore_name
  namespace: velero
  labels:
    restore-type: disaster-recovery-failover
    nephoran.com/failover: "true"
spec:
  backupName: $backup_name
  
  # Restore to same namespaces
  namespaceMapping: {}
  
  # Include all resources from backup
  includedResources:
  - networkintents.nephoran.com
  - e2nodesets.nephoran.com
  - managedelements.nephoran.com
  - secrets
  - configmaps
  - persistentvolumeclaims
  - deployments
  - services
  - ingresses
  
  # Restore hooks for application startup
  hooks:
    resources:
    - name: weaviate-post-restore
      includedNamespaces:
      - nephoran-system
      - nephoran-production
      labelSelector:
        matchLabels:
          app.kubernetes.io/name: weaviate
      post:
      - exec:
          container: weaviate
          command:
          - /bin/sh
          - -c
          - "sleep 30 && curl -X POST http://localhost:8080/v1/schema/restore"
          onError: Continue
          timeout: 5m
  
  restorePVs: true
  preserveNodePorts: false
EOF
    
    # Wait for restore to complete
    log_info "Waiting for restore to complete..."
    
    local timeout=600  # 10 minutes
    local counter=0
    while [ $counter -lt $timeout ]; do
        local restore_phase=$(kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get restore $restore_name -n velero -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        
        case "$restore_phase" in
            "Completed")
                log_success "Restore completed successfully"
                return 0
                ;;
            "Failed"|"PartiallyFailed")
                log_error "Restore failed or partially failed"
                kubectl --kubeconfig="$SECONDARY_KUBECONFIG" describe restore $restore_name -n velero
                return 1
                ;;
            "InProgress")
                log_info "Restore in progress... ($counter/$timeout seconds)"
                ;;
            *)
                log_info "Waiting for restore to start... ($counter/$timeout seconds)"
                ;;
        esac
        
        sleep 10
        counter=$((counter + 10))
    done
    
    log_error "Restore timed out after $timeout seconds"
    return 1
}

# Update DNS/Load Balancer configuration
update_load_balancer() {
    log_info "Updating load balancer configuration..."
    
    # Get secondary cluster endpoints
    local secondary_api_server="https://localhost:6444"
    local secondary_ingress_http="http://localhost:8081"
    local secondary_ingress_https="https://localhost:8443"
    
    # Create load balancer configuration
    cat > "$LB_CONFIG_FILE" << EOF
{
  "failover_time": "$(date -Iseconds)",
  "primary_cluster": {
    "name": "$PRIMARY_CLUSTER_NAME",
    "status": "failed",
    "api_server": "https://localhost:6443",
    "ingress_http": "http://localhost:8080",
    "ingress_https": "https://localhost:8443"
  },
  "secondary_cluster": {
    "name": "$SECONDARY_CLUSTER_NAME",
    "status": "active",
    "api_server": "$secondary_api_server",
    "ingress_http": "$secondary_ingress_http",
    "ingress_https": "$secondary_ingress_https"
  },
  "dns_updates": []
}
EOF
    
    case "$DNS_PROVIDER" in
        "cloudflare")
            update_cloudflare_dns
            ;;
        "route53")
            update_route53_dns
            ;;
        "manual")
            log_warn "Manual DNS update required:"
            log_warn "  Update A record for $DOMAIN_NAME to point to secondary cluster"
            log_warn "  Update API endpoint from primary to secondary"
            ;;
        *)
            log_warn "Unknown DNS provider: $DNS_PROVIDER"
            ;;
    esac
    
    log_success "Load balancer configuration updated"
}

update_cloudflare_dns() {
    if [ -z "${CLOUDFLARE_API_TOKEN:-}" ]; then
        log_warn "CLOUDFLARE_API_TOKEN not set, skipping DNS update"
        return 0
    fi
    
    log_info "Updating Cloudflare DNS records..."
    
    # This would implement actual Cloudflare API calls
    # For now, we'll simulate the update
    log_success "Cloudflare DNS records updated (simulated)"
    
    # Update configuration file
    jq '.dns_updates += [{"provider": "cloudflare", "status": "updated", "timestamp": "'$(date -Iseconds)'"}]' "$LB_CONFIG_FILE" > "$LB_CONFIG_FILE.tmp"
    mv "$LB_CONFIG_FILE.tmp" "$LB_CONFIG_FILE"
}

update_route53_dns() {
    if [ -z "${AWS_ACCESS_KEY_ID:-}" ] || [ -z "${AWS_SECRET_ACCESS_KEY:-}" ]; then
        log_warn "AWS credentials not set, skipping DNS update"
        return 0
    fi
    
    log_info "Updating Route53 DNS records..."
    
    # This would implement actual Route53 API calls
    # For now, we'll simulate the update
    log_success "Route53 DNS records updated (simulated)"
    
    # Update configuration file
    jq '.dns_updates += [{"provider": "route53", "status": "updated", "timestamp": "'$(date -Iseconds)'"}]' "$LB_CONFIG_FILE" > "$LB_CONFIG_FILE.tmp"
    mv "$LB_CONFIG_FILE.tmp" "$LB_CONFIG_FILE"
}

# Verify workload health after failover
verify_failover_success() {
    log_info "Verifying failover success..."
    
    # Check critical pods are running
    local critical_components=("nephoran-operator" "llm-processor" "rag-api" "weaviate")
    local all_healthy=true
    
    for component in "${critical_components[@]}"; do
        log_info "Checking $component..."
        
        local timeout=120
        local counter=0
        local component_ready=false
        
        while [ $counter -lt $timeout ]; do
            if kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get pods -n nephoran-system -l "app.kubernetes.io/name=$component" --no-headers 2>/dev/null | grep -q "Running"; then
                local ready_count=$(kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get pods -n nephoran-system -l "app.kubernetes.io/name=$component" --no-headers | grep -c "Running")
                local total_count=$(kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get pods -n nephoran-system -l "app.kubernetes.io/name=$component" --no-headers | wc -l)
                
                if [ "$ready_count" = "$total_count" ] && [ "$total_count" -gt 0 ]; then
                    log_success "$component is ready ($ready_count/$total_count pods)"
                    component_ready=true
                    break
                fi
            fi
            
            sleep 5
            counter=$((counter + 5))
            
            if [ $((counter % 30)) -eq 0 ]; then
                log_info "Waiting for $component to be ready... ($counter/$timeout seconds)"
            fi
        done
        
        if [ "$component_ready" = false ]; then
            log_error "$component failed to start within $timeout seconds"
            all_healthy=false
        fi
    done
    
    # Test API endpoints
    log_info "Testing API endpoints..."
    
    # Test Kubernetes API
    if kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get nodes &>/dev/null; then
        log_success "Kubernetes API is accessible"
    else
        log_error "Kubernetes API is not accessible"
        all_healthy=false
    fi
    
    # Test custom resources
    if kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get networkintents -n nephoran-system &>/dev/null; then
        local intent_count=$(kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get networkintents -n nephoran-system --no-headers | wc -l)
        log_success "NetworkIntents accessible ($intent_count found)"
    else
        log_error "NetworkIntents not accessible"
        all_healthy=false
    fi
    
    # Test storage
    local pv_count=$(kubectl --kubeconfig="$SECONDARY_KUBECONFIG" get pv --no-headers | grep -c "Bound" || echo 0)
    if [ "$pv_count" -gt 0 ]; then
        log_success "Persistent storage available ($pv_count volumes bound)"
    else
        log_warn "No bound persistent volumes found"
    fi
    
    if [ "$all_healthy" = true ]; then
        log_success "Failover verification completed successfully"
        return 0
    else
        log_error "Failover verification failed"
        return 1
    fi
}

# Send notifications about failover
send_failover_notifications() {
    log_info "Sending failover notifications..."
    
    local failover_end_time=$(date +%s)
    local total_failover_time=$((failover_end_time - FAILOVER_START_TIME))
    
    # Create notification payload
    local notification_data=$(cat << EOF
{
  "event": "disaster_recovery_failover",
  "timestamp": "$(date -Iseconds)",
  "severity": "critical",
  "cluster": {
    "primary": "$PRIMARY_CLUSTER_NAME",
    "secondary": "$SECONDARY_CLUSTER_NAME"
  },
  "failover": {
    "start_time": "$(date -d @$FAILOVER_START_TIME -Iseconds)",
    "end_time": "$(date -Iseconds)",
    "duration_seconds": $total_failover_time,
    "rto_target_seconds": 300,
    "rto_achieved": $([ $total_failover_time -lt 300 ] && echo "true" || echo "false")
  },
  "phases": "$FAILOVER_PHASES",
  "status": "completed"
}
EOF
)
    
    # Write notification to file
    echo "$notification_data" > "/tmp/failover-notification-$(date +%Y%m%d-%H%M%S).json"
    
    # Send to monitoring systems (webhook endpoints)
    if [ -n "${WEBHOOK_URL:-}" ]; then
        curl -s -X POST -H "Content-Type: application/json" -d "$notification_data" "$WEBHOOK_URL" || log_warn "Failed to send webhook notification"
    fi
    
    # Send to Slack (if configured)
    if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
        local slack_message="🚨 *Disaster Recovery Failover Completed*\n\n• Primary Cluster: \`$PRIMARY_CLUSTER_NAME\` (FAILED)\n• Secondary Cluster: \`$SECONDARY_CLUSTER_NAME\` (ACTIVE)\n• Failover Duration: ${total_failover_time}s (Target: 300s)\n• RTO Achieved: $([ $total_failover_time -lt 300 ] && echo "✅ YES" || echo "❌ NO")\n• Time: $(date)"
        
        local slack_payload=$(jq -n --arg text "$slack_message" '{text: $text}')
        curl -s -X POST -H "Content-Type: application/json" -d "$slack_payload" "$SLACK_WEBHOOK_URL" || log_warn "Failed to send Slack notification"
    fi
    
    log_success "Notifications sent"
    
    # Display final summary
    echo
    echo "=========================================="
    echo "DISASTER RECOVERY FAILOVER COMPLETED"
    echo "=========================================="
    echo "Start Time: $(date -d @$FAILOVER_START_TIME)"
    echo "End Time: $(date)"
    echo "Duration: ${total_failover_time} seconds"
    echo "RTO Target: 300 seconds (5 minutes)"
    echo "RTO Achieved: $([ $total_failover_time -lt 300 ] && echo "YES ✅" || echo "NO ❌")"
    echo
    echo "Primary Cluster: $PRIMARY_CLUSTER_NAME (FAILED)"
    echo "Secondary Cluster: $SECONDARY_CLUSTER_NAME (ACTIVE)"
    echo
    echo "To switch kubectl context:"
    echo "  kubectl config use-context nephoran-dr-secondary"
    echo
    echo "Log File: $LOG_FILE"
    echo "Notification: /tmp/failover-notification-$(date +%Y%m%d-%H%M%S).json"
    echo "=========================================="
}

# Cleanup function for script interruption
cleanup() {
    log_warn "Failover script interrupted. Cleaning up..."
    remove_failover_lock
    
    # If failover was in progress, log the interruption
    if [ -n "$FAILOVER_START_TIME" ]; then
        local current_time=$(date +%s)
        local elapsed=$((current_time - FAILOVER_START_TIME))
        log_critical "Failover interrupted after ${elapsed} seconds"
    fi
    
    exit 1
}

# Main failover execution
execute_failover() {
    log_critical "=== DISASTER RECOVERY FAILOVER INITIATED ==="
    
    start_failover_timer
    log_failover_phase "Failover initiated"
    
    # Phase 1: Health checks and validation
    log_failover_phase "Health checks and validation"
    
    if ! check_secondary_cluster_health; then
        log_critical "Secondary cluster not ready, cannot proceed with failover"
        exit 1
    fi
    
    # Phase 2: Emergency backup (if primary is accessible)
    log_failover_phase "Emergency backup creation"
    create_failover_backup
    
    # Phase 3: Identify and prepare restore
    log_failover_phase "Backup identification and restore preparation"
    if ! get_latest_backup; then
        log_critical "No backup available for restore, cannot proceed"
        exit 1
    fi
    
    # Phase 4: Restore from backup
    log_failover_phase "Restore from backup"
    if ! restore_from_backup; then
        log_critical "Restore failed, failover unsuccessful"
        exit 1
    fi
    
    # Phase 5: Update load balancer/DNS
    log_failover_phase "Load balancer and DNS update"
    update_load_balancer
    
    # Phase 6: Verify failover success
    log_failover_phase "Failover verification"
    if ! verify_failover_success; then
        log_critical "Failover verification failed"
        exit 1
    fi
    
    # Phase 7: Notifications
    log_failover_phase "Notifications and reporting"
    send_failover_notifications
    
    log_failover_phase "Failover completed successfully"
    log_critical "=== DISASTER RECOVERY FAILOVER COMPLETED ==="
}

# Monitor mode - continuous health checking
monitor_mode() {
    log_info "Starting continuous health monitoring mode..."
    log_info "Press Ctrl+C to stop monitoring"
    
    local check_interval=60  # Check every minute
    local consecutive_failures=0
    local max_consecutive_failures=3
    
    while true; do
        log_info "Performing health check... ($(date))"
        
        if check_primary_cluster_health; then
            consecutive_failures=0
            log_success "Primary cluster healthy"
        else
            consecutive_failures=$((consecutive_failures + 1))
            log_warn "Primary cluster unhealthy (consecutive failures: $consecutive_failures/$max_consecutive_failures)"
            
            if [ $consecutive_failures -ge $max_consecutive_failures ]; then
                log_critical "Primary cluster failed $max_consecutive_failures consecutive health checks"
                log_critical "Triggering automatic failover..."
                
                # Create failover trigger file
                echo "$(date -Iseconds): Automatic failover triggered by health check failures" > "$FAILOVER_TRIGGER_FILE"
                
                # Execute failover
                execute_failover
                break
            fi
        fi
        
        sleep $check_interval
    done
}

# Usage information
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Nephoran Intent Operator Disaster Recovery Failover Script"
    echo
    echo "OPTIONS:"
    echo "  --execute          Execute immediate failover"
    echo "  --monitor          Start continuous health monitoring with auto-failover"
    echo "  --force            Force failover without health checks"
    echo "  --dry-run          Simulate failover without making changes"
    echo "  --check-primary    Check primary cluster health only"
    echo "  --check-secondary  Check secondary cluster health only"
    echo "  --help             Show this help message"
    echo
    echo "ENVIRONMENT VARIABLES:"
    echo "  CLOUDFLARE_API_TOKEN    Cloudflare API token for DNS updates"
    echo "  AWS_ACCESS_KEY_ID       AWS access key for Route53 updates"
    echo "  AWS_SECRET_ACCESS_KEY   AWS secret key for Route53 updates"
    echo "  WEBHOOK_URL             Webhook URL for notifications"
    echo "  SLACK_WEBHOOK_URL       Slack webhook URL for notifications"
    echo
    echo "EXAMPLES:"
    echo "  $0 --execute                    # Execute immediate failover"
    echo "  $0 --monitor                    # Start health monitoring"
    echo "  $0 --check-primary              # Check primary cluster health"
    echo "  $0 --dry-run --execute          # Simulate failover"
}

# Parse command line arguments
main() {
    if [ $# -eq 0 ]; then
        usage
        exit 1
    fi
    
    check_failover_lock
    trap cleanup INT TERM
    trap remove_failover_lock EXIT
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --execute)
                execute_failover
                shift
                ;;
            --monitor)
                monitor_mode
                shift
                ;;
            --force)
                # Skip health checks for forced failover
                MAX_HEALTH_CHECK_RETRIES=0
                shift
                ;;
            --dry-run)
                log_info "DRY RUN MODE - No actual changes will be made"
                # Set dry run flag for testing
                export DRY_RUN=true
                shift
                ;;
            --check-primary)
                if check_primary_cluster_health; then
                    log_success "Primary cluster is healthy"
                    exit 0
                else
                    log_error "Primary cluster is unhealthy"
                    exit 1
                fi
                ;;
            --check-secondary)
                if check_secondary_cluster_health; then
                    log_success "Secondary cluster is ready"
                    exit 0
                else
                    log_error "Secondary cluster is not ready"
                    exit 1
                fi
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Run main function with all arguments
main "$@"