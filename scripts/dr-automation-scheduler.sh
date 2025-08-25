#!/bin/bash
# Nephoran Intent Operator DR Automation Scheduler
# Automated disaster recovery operations and testing
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-nephoran-system}"
DR_SCHEDULE_CONFIG="${DR_SCHEDULE_CONFIG:-/etc/nephoran/dr-schedule.yaml}"
LOG_FILE="${LOG_FILE:-/var/log/nephoran/dr-scheduler.log}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

log() {
    local message="[$(date +'%Y-%m-%d %H:%M:%S')] $1"
    echo -e "${BLUE}${message}${NC}"
    echo "$message" >> "$LOG_FILE"
}

warn() {
    local message="[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1"
    echo -e "${YELLOW}${message}${NC}"
    echo "$message" >> "$LOG_FILE"
}

error() {
    local message="[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1"
    echo -e "${RED}${message}${NC}"
    echo "$message" >> "$LOG_FILE"
}

success() {
    local message="[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
    echo -e "${GREEN}${message}${NC}"
    echo "$message" >> "$LOG_FILE"
}

# Initialize DR scheduler
initialize_scheduler() {
    log "Initializing DR automation scheduler..."
    
    # Create log directory
    mkdir -p "$(dirname "$LOG_FILE")"
    
    # Create default schedule configuration if it doesn't exist
    if [[ ! -f "$DR_SCHEDULE_CONFIG" ]]; then
        mkdir -p "$(dirname "$DR_SCHEDULE_CONFIG")"
        cat > "$DR_SCHEDULE_CONFIG" <<EOF
# Nephoran Intent Operator DR Schedule Configuration
schedule:
  # Full backup schedule (daily at 2 AM)
  full_backup:
    enabled: true
    cron: "0 2 * * *"
    retention_days: 30
    
  # Incremental backup schedule (every 6 hours)
  incremental_backup:
    enabled: true
    cron: "0 */6 * * *"
    retention_days: 7
    
  # DR testing schedule (monthly on first Sunday at 3 AM)
  dr_test:
    enabled: true
    cron: "0 3 1-7 * 0"
    test_type: "partial"
    
  # Backup verification (daily at 3 AM)
  backup_verification:
    enabled: true
    cron: "0 3 * * *"
    
  # Cleanup old backups (weekly on Sunday at 4 AM)
  cleanup:
    enabled: true
    cron: "0 4 * * 0"
    keep_days: 90

notifications:
  # Slack webhook for notifications
  slack:
    enabled: false
    webhook_url: ""
    channel: "#nephoran-ops"
    
  # Email notifications
  email:
    enabled: false
    smtp_server: ""
    smtp_port: 587
    username: ""
    password: ""
    to_addresses: ["ops@nephoran.com"]
    
  # PagerDuty integration
  pagerduty:
    enabled: false
    integration_key: ""

monitoring:
  # Prometheus metrics
  prometheus:
    enabled: true
    pushgateway_url: "http://prometheus-pushgateway:9091"
    
  # Health check endpoint
  health_check:
    enabled: true
    port: 8090
    path: "/health"
EOF
        success "Default DR schedule configuration created: $DR_SCHEDULE_CONFIG"
    fi
}

# Parse schedule configuration
parse_schedule() {
    if [[ ! -f "$DR_SCHEDULE_CONFIG" ]]; then
        error "DR schedule configuration not found: $DR_SCHEDULE_CONFIG"
        return 1
    fi
    
    # For simplicity, we'll extract basic values using grep/awk
    # In production, consider using yq or a proper YAML parser
    
    FULL_BACKUP_ENABLED=$(grep -A 3 "full_backup:" "$DR_SCHEDULE_CONFIG" | grep "enabled:" | awk '{print $2}')
    FULL_BACKUP_CRON=$(grep -A 3 "full_backup:" "$DR_SCHEDULE_CONFIG" | grep "cron:" | awk '{print $2}' | tr -d '"')
    
    INCREMENTAL_BACKUP_ENABLED=$(grep -A 3 "incremental_backup:" "$DR_SCHEDULE_CONFIG" | grep "enabled:" | awk '{print $2}')
    INCREMENTAL_BACKUP_CRON=$(grep -A 3 "incremental_backup:" "$DR_SCHEDULE_CONFIG" | grep "cron:" | awk '{print $2}' | tr -d '"')
    
    DR_TEST_ENABLED=$(grep -A 3 "dr_test:" "$DR_SCHEDULE_CONFIG" | grep "enabled:" | awk '{print $2}')
    DR_TEST_CRON=$(grep -A 3 "dr_test:" "$DR_SCHEDULE_CONFIG" | grep "cron:" | awk '{print $2}' | tr -d '"')
    
    VERIFICATION_ENABLED=$(grep -A 3 "backup_verification:" "$DR_SCHEDULE_CONFIG" | grep "enabled:" | awk '{print $2}')
    VERIFICATION_CRON=$(grep -A 3 "backup_verification:" "$DR_SCHEDULE_CONFIG" | grep "cron:" | awk '{print $2}' | tr -d '"')
    
    CLEANUP_ENABLED=$(grep -A 3 "cleanup:" "$DR_SCHEDULE_CONFIG" | grep "enabled:" | awk '{print $2}')
    CLEANUP_CRON=$(grep -A 3 "cleanup:" "$DR_SCHEDULE_CONFIG" | grep "cron:" | awk '{print $2}' | tr -d '"')
    
    log "Schedule configuration parsed successfully"
}

# Send notification
send_notification() {
    local subject="$1"
    local message="$2"
    local severity="${3:-info}"
    
    # Check if Slack notifications are enabled
    local slack_enabled=$(grep -A 5 "slack:" "$DR_SCHEDULE_CONFIG" | grep "enabled:" | awk '{print $2}')
    local slack_webhook=$(grep -A 5 "slack:" "$DR_SCHEDULE_CONFIG" | grep "webhook_url:" | awk '{print $2}' | tr -d '"')
    
    if [[ "$slack_enabled" == "true" ]] && [[ -n "$slack_webhook" ]]; then
        local color="good"
        case $severity in
            "error") color="danger" ;;
            "warning") color="warning" ;;
            "info") color="good" ;;
        esac
        
        local payload=$(cat <<EOF
{
    "channel": "#nephoran-ops",
    "username": "Nephoran DR Bot",
    "icon_emoji": ":gear:",
    "attachments": [{
        "color": "$color",
        "title": "$subject",
        "text": "$message",
        "footer": "Nephoran DR Scheduler",
        "ts": $(date +%s)
    }]
}
EOF
)
        
        curl -X POST -H 'Content-type: application/json' \
            --data "$payload" "$slack_webhook" >/dev/null 2>&1 || \
            warn "Failed to send Slack notification"
    fi
    
    log "Notification sent: $subject"
}

# Execute full backup
execute_full_backup() {
    log "Executing scheduled full backup..."
    
    local backup_start=$(date +%s)
    local backup_result=0
    
    # Run the main DR system script
    if [[ -f "$SCRIPT_DIR/disaster-recovery-system.sh" ]]; then
        "$SCRIPT_DIR/disaster-recovery-system.sh" >> "$LOG_FILE" 2>&1
        backup_result=$?
    else
        error "Disaster recovery system script not found"
        backup_result=1
    fi
    
    local backup_end=$(date +%s)
    local backup_duration=$((backup_end - backup_start))
    
    if [[ $backup_result -eq 0 ]]; then
        success "Full backup completed successfully in ${backup_duration}s"
        send_notification "✅ Full Backup Successful" \
            "Nephoran full backup completed successfully in ${backup_duration} seconds." "info"
    else
        error "Full backup failed after ${backup_duration}s"
        send_notification "❌ Full Backup Failed" \
            "Nephoran full backup failed after ${backup_duration} seconds. Check logs for details." "error"
    fi
    
    # Update Prometheus metrics if enabled
    update_prometheus_metrics "full_backup" $backup_result $backup_duration
    
    return $backup_result
}

# Execute incremental backup
execute_incremental_backup() {
    log "Executing scheduled incremental backup..."
    
    local backup_start=$(date +%s)
    local backup_result=0
    
    # For incremental backup, we'll backup only changed resources
    local backup_dir="/var/nephoran/backups/incremental/$(date +%Y%m%d-%H%M%S)"
    mkdir -p "$backup_dir"
    
    # Backup recent changes (last 6 hours)
    local since="6h"
    
    # Backup NetworkIntents modified in the last 6 hours
    kubectl get networkintents -n "$NAMESPACE" \
        --field-selector="metadata.creationTimestamp>$(date -d "$since ago" -Iseconds)" \
        -o yaml > "$backup_dir/recent-networkintents.yaml" 2>/dev/null || true
    
    # Backup E2NodeSets modified in the last 6 hours
    kubectl get e2nodesets -n "$NAMESPACE" \
        --field-selector="metadata.creationTimestamp>$(date -d "$since ago" -Iseconds)" \
        -o yaml > "$backup_dir/recent-e2nodesets.yaml" 2>/dev/null || true
    
    # Backup recent ConfigMaps
    kubectl get configmaps -n "$NAMESPACE" \
        --field-selector="metadata.creationTimestamp>$(date -d "$since ago" -Iseconds)" \
        -o yaml > "$backup_dir/recent-configmaps.yaml" 2>/dev/null || true
    
    # Create archive
    cd "$(dirname "$backup_dir")"
    tar -czf "$(basename "$backup_dir").tar.gz" "$(basename "$backup_dir")"
    rm -rf "$backup_dir"
    
    local backup_end=$(date +%s)
    local backup_duration=$((backup_end - backup_start))
    
    success "Incremental backup completed in ${backup_duration}s"
    
    # Update Prometheus metrics
    update_prometheus_metrics "incremental_backup" 0 $backup_duration
    
    return 0
}

# Execute DR test
execute_dr_test() {
    log "Executing scheduled DR test..."
    
    local test_start=$(date +%s)
    local test_result=0
    
    # Create test namespace
    local test_namespace="nephoran-dr-test-$(date +%s)"
    kubectl create namespace "$test_namespace" || {
        error "Failed to create test namespace"
        return 1
    }
    
    # Label test namespace
    kubectl label namespace "$test_namespace" nephoran.com/dr-test=true
    
    # Perform partial restore test
    local latest_backup=$(find /var/nephoran/backups -name "nephoran-backup-*.tar.gz" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2-)
    
    if [[ -n "$latest_backup" ]]; then
        log "Testing restore from backup: $latest_backup"
        
        # Extract backup for testing
        local test_dir="/tmp/dr-test-$(date +%s)"
        mkdir -p "$test_dir"
        cd "$test_dir"
        tar -xzf "$latest_backup" || {
            error "Failed to extract backup for testing"
            test_result=1
        }
        
        # Test CRD restoration
        if [[ $test_result -eq 0 ]] && [[ -f "kubernetes/crds.yaml" ]]; then
            kubectl apply -f "kubernetes/crds.yaml" --dry-run=client || {
                error "CRD restoration test failed"
                test_result=1
            }
        fi
        
        # Cleanup test files
        rm -rf "$test_dir"
    else
        warn "No backup found for DR testing"
        test_result=1
    fi
    
    # Cleanup test namespace
    kubectl delete namespace "$test_namespace" --ignore-not-found=true
    
    local test_end=$(date +%s)
    local test_duration=$((test_end - test_start))
    
    if [[ $test_result -eq 0 ]]; then
        success "DR test completed successfully in ${test_duration}s"
        send_notification "✅ DR Test Successful" \
            "Nephoran DR test completed successfully in ${test_duration} seconds." "info"
    else
        error "DR test failed after ${test_duration}s"
        send_notification "❌ DR Test Failed" \
            "Nephoran DR test failed after ${test_duration} seconds. Check logs for details." "error"
    fi
    
    # Update Prometheus metrics
    update_prometheus_metrics "dr_test" $test_result $test_duration
    
    return $test_result
}

# Verify backup integrity
verify_backups() {
    log "Executing scheduled backup verification..."
    
    local verification_start=$(date +%s)
    local verification_result=0
    local verified_backups=0
    local failed_backups=0
    
    # Find all backup files from the last 24 hours
    local backup_files=$(find /var/nephoran/backups -name "*.tar.gz" -newermt "24 hours ago" -type f)
    
    for backup_file in $backup_files; do
        log "Verifying backup: $(basename "$backup_file")"
        
        if tar -tzf "$backup_file" >/dev/null 2>&1; then
            verified_backups=$((verified_backups + 1))
            log "  ✓ Archive integrity verified"
        else
            failed_backups=$((failed_backups + 1))
            warn "  ✗ Archive integrity check failed"
            verification_result=1
        fi
    done
    
    local verification_end=$(date +%s)
    local verification_duration=$((verification_end - verification_start))
    
    if [[ $verification_result -eq 0 ]]; then
        success "Backup verification completed: $verified_backups verified, $failed_backups failed (${verification_duration}s)"
        if [[ $failed_backups -gt 0 ]]; then
            send_notification "⚠️ Backup Verification Issues" \
                "Backup verification completed with $failed_backups failed backups out of $((verified_backups + failed_backups)) total." "warning"
        fi
    else
        error "Backup verification failed: $verified_backups verified, $failed_backups failed (${verification_duration}s)"
        send_notification "❌ Backup Verification Failed" \
            "Backup verification failed with $failed_backups failed backups." "error"
    fi
    
    # Update Prometheus metrics
    update_prometheus_metrics "backup_verification" $verification_result $verification_duration
    
    return $verification_result
}

# Cleanup old backups
cleanup_old_backups() {
    log "Executing scheduled backup cleanup..."
    
    local cleanup_start=$(date +%s)
    local keep_days=90
    local deleted_count=0
    local deleted_size=0
    
    # Clean up local backups older than specified days
    local old_backups=$(find /var/nephoran/backups -name "*.tar.gz" -type f -mtime +$keep_days)
    
    for backup_file in $old_backups; do
        local file_size=$(du -b "$backup_file" | cut -f1)
        deleted_size=$((deleted_size + file_size))
        deleted_count=$((deleted_count + 1))
        
        log "Deleting old backup: $(basename "$backup_file")"
        rm -f "$backup_file"
    done
    
    # Clean up S3 backups if AWS CLI is available
    if command -v aws >/dev/null 2>&1; then
        local s3_bucket="${S3_BUCKET:-nephoran-dr-backups}"
        aws s3 ls "s3://$s3_bucket/disaster-recovery/" | \
        awk '{print $4}' | \
        while read -r s3_file; do
            if [[ -n "$s3_file" ]]; then
                local file_date=$(echo "$s3_file" | grep -o '[0-9]\{8\}-[0-9]\{6\}' | head -1)
                if [[ -n "$file_date" ]]; then
                    local file_epoch=$(date -d "${file_date:0:8} ${file_date:9:2}:${file_date:11:2}:${file_date:13:2}" +%s 2>/dev/null || echo "0")
                    local cutoff_epoch=$(date -d "$keep_days days ago" +%s)
                    
                    if [[ $file_epoch -lt $cutoff_epoch ]]; then
                        log "Deleting old S3 backup: $s3_file"
                        aws s3 rm "s3://$s3_bucket/disaster-recovery/$s3_file" || warn "Failed to delete S3 backup: $s3_file"
                    fi
                fi
            fi
        done
    fi
    
    local cleanup_end=$(date +%s)
    local cleanup_duration=$((cleanup_end - cleanup_start))
    local deleted_size_mb=$((deleted_size / 1024 / 1024))
    
    success "Cleanup completed: $deleted_count files deleted, ${deleted_size_mb}MB freed (${cleanup_duration}s)"
    
    if [[ $deleted_count -gt 0 ]]; then
        send_notification "🧹 Backup Cleanup Completed" \
            "Deleted $deleted_count old backup files, freed ${deleted_size_mb}MB of storage." "info"
    fi
    
    # Update Prometheus metrics
    update_prometheus_metrics "cleanup" 0 $cleanup_duration
    
    return 0
}

# Update Prometheus metrics
update_prometheus_metrics() {
    local operation="$1"
    local result="$2"
    local duration="$3"
    
    local prometheus_enabled=$(grep -A 5 "prometheus:" "$DR_SCHEDULE_CONFIG" | grep "enabled:" | awk '{print $2}')
    local pushgateway_url=$(grep -A 5 "prometheus:" "$DR_SCHEDULE_CONFIG" | grep "pushgateway_url:" | awk '{print $2}' | tr -d '"')
    
    if [[ "$prometheus_enabled" == "true" ]] && [[ -n "$pushgateway_url" ]] && command -v curl >/dev/null 2>&1; then
        local metrics="nephoran_dr_operation_duration_seconds{operation=\"$operation\",namespace=\"$NAMESPACE\"} $duration
nephoran_dr_operation_result{operation=\"$operation\",namespace=\"$NAMESPACE\"} $result
nephoran_dr_operation_timestamp{operation=\"$operation\",namespace=\"$NAMESPACE\"} $(date +%s)"
        
        echo "$metrics" | curl -X POST --data-binary @- \
            "$pushgateway_url/metrics/job/nephoran-dr-scheduler/instance/$(hostname)" \
            >/dev/null 2>&1 || warn "Failed to push metrics to Prometheus"
    fi
}

# Start health check server
start_health_check_server() {
    local health_enabled=$(grep -A 5 "health_check:" "$DR_SCHEDULE_CONFIG" | grep "enabled:" | awk '{print $2}')
    local health_port=$(grep -A 5 "health_check:" "$DR_SCHEDULE_CONFIG" | grep "port:" | awk '{print $2}')
    
    if [[ "$health_enabled" == "true" ]] && [[ -n "$health_port" ]]; then
        log "Starting health check server on port $health_port"
        
        # Simple health check server using nc (netcat)
        while true; do
            echo -e "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK" | nc -l -p "$health_port" -q 1 >/dev/null 2>&1
            sleep 1
        done &
        
        local health_pid=$!
        echo $health_pid > /var/run/nephoran-dr-health.pid
        log "Health check server started with PID: $health_pid"
    fi
}

# Install cron jobs
install_cron_jobs() {
    log "Installing DR automation cron jobs..."
    
    parse_schedule
    
    # Create temporary crontab file
    local temp_crontab="/tmp/nephoran-dr-crontab-$$"
    
    # Get existing crontab (if any) excluding Nephoran DR entries
    crontab -l 2>/dev/null | grep -v "# Nephoran DR" > "$temp_crontab" || true
    
    # Add Nephoran DR jobs
    echo "# Nephoran DR automation jobs" >> "$temp_crontab"
    
    if [[ "$FULL_BACKUP_ENABLED" == "true" ]]; then
        echo "$FULL_BACKUP_CRON $SCRIPT_DIR/dr-automation-scheduler.sh full_backup # Nephoran DR" >> "$temp_crontab"
    fi
    
    if [[ "$INCREMENTAL_BACKUP_ENABLED" == "true" ]]; then
        echo "$INCREMENTAL_BACKUP_CRON $SCRIPT_DIR/dr-automation-scheduler.sh incremental_backup # Nephoran DR" >> "$temp_crontab"
    fi
    
    if [[ "$DR_TEST_ENABLED" == "true" ]]; then
        echo "$DR_TEST_CRON $SCRIPT_DIR/dr-automation-scheduler.sh dr_test # Nephoran DR" >> "$temp_crontab"
    fi
    
    if [[ "$VERIFICATION_ENABLED" == "true" ]]; then
        echo "$VERIFICATION_CRON $SCRIPT_DIR/dr-automation-scheduler.sh verify_backups # Nephoran DR" >> "$temp_crontab"
    fi
    
    if [[ "$CLEANUP_ENABLED" == "true" ]]; then
        echo "$CLEANUP_CRON $SCRIPT_DIR/dr-automation-scheduler.sh cleanup # Nephoran DR" >> "$temp_crontab"
    fi
    
    # Install the new crontab
    crontab "$temp_crontab"
    rm -f "$temp_crontab"
    
    success "DR automation cron jobs installed successfully"
    crontab -l | grep "# Nephoran DR"
}

# Main function
main() {
    local command="${1:-help}"
    
    case $command in
        "init"|"initialize")
            initialize_scheduler
            install_cron_jobs
            start_health_check_server
            ;;
        "full_backup")
            initialize_scheduler
            execute_full_backup
            ;;
        "incremental_backup")
            initialize_scheduler
            execute_incremental_backup
            ;;
        "dr_test")
            initialize_scheduler
            execute_dr_test
            ;;
        "verify_backups")
            initialize_scheduler
            verify_backups
            ;;
        "cleanup")
            initialize_scheduler
            cleanup_old_backups
            ;;
        "status")
            log "Nephoran DR Scheduler Status:"
            crontab -l | grep "# Nephoran DR" || log "No DR cron jobs found"
            if [[ -f /var/run/nephoran-dr-health.pid ]]; then
                local health_pid=$(cat /var/run/nephoran-dr-health.pid)
                if ps -p "$health_pid" >/dev/null 2>&1; then
                    log "Health check server running (PID: $health_pid)"
                else
                    log "Health check server not running"
                fi
            fi
            ;;
        "help"|*)
            echo "Nephoran DR Automation Scheduler"
            echo ""
            echo "Usage: $0 <command>"
            echo ""
            echo "Commands:"
            echo "  init              Initialize DR scheduler and install cron jobs"
            echo "  full_backup       Execute full backup"
            echo "  incremental_backup Execute incremental backup"
            echo "  dr_test           Execute DR test"
            echo "  verify_backups    Verify backup integrity"
            echo "  cleanup           Clean up old backups"
            echo "  status            Show scheduler status"
            echo "  help              Show this help message"
            echo ""
            echo "Configuration file: $DR_SCHEDULE_CONFIG"
            echo "Log file: $LOG_FILE"
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi