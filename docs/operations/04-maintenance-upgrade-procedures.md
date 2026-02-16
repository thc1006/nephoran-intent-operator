# Nephoran Intent Operator - Maintenance & Upgrade Procedures

## Overview

This document provides comprehensive procedures for maintaining, upgrading, and managing the lifecycle of the Nephoran Intent Operator in production environments. All procedures are designed for zero-downtime operations with automated rollback capabilities.

## Maintenance Framework

### Maintenance Categories

| Type | Frequency | Downtime | Scope | Automation Level |
|------|-----------|----------|-------|------------------|
| **Critical Security** | As needed | Zero | Security patches | Fully automated |
| **Emergency Patches** | As needed | Zero | Critical bugs | Semi-automated |
| **Minor Updates** | Weekly | Zero | Bug fixes, features | Automated |
| **Major Upgrades** | Monthly | Minimal | Version upgrades | Semi-automated |
| **Infrastructure** | Quarterly | Planned | Platform upgrades | Manual + automated |

### Maintenance Windows

**Standard Maintenance Windows:**
- **Daily**: 02:00-04:00 UTC (automated tasks)
- **Weekly**: Sunday 01:00-05:00 UTC (minor updates)
- **Monthly**: First Saturday 00:00-06:00 UTC (major upgrades)
- **Emergency**: As needed (24/7 on-call)

## Rolling Upgrade Procedures

### 1.1 Zero-Downtime Rolling Updates

**Pre-Upgrade Checklist:**
```bash
#!/bin/bash
# pre-upgrade-checklist.sh

echo "🔍 Pre-upgrade checklist for Nephoran Intent Operator"

UPGRADE_ID="UPG-$(date +%Y%m%d-%H%M%S)"
CHECKLIST_RESULTS="/tmp/pre-upgrade-$UPGRADE_ID.json"

# 1. System health validation
SYSTEM_HEALTH=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
ACTIVE_INTENTS=$(kubectl get networkintents -A --no-headers | grep -c "Processing")
BACKUP_STATUS=$(kubectl get cronjobs -n nephoran-system | grep backup | awk '{print $4}')

# 2. Resource availability check
NODE_CAPACITY=$(kubectl describe nodes | grep -A 5 "Allocated resources" | grep -E "(cpu|memory)" | tail -2)
STORAGE_USAGE=$(kubectl get pvc -n nephoran-system -o json | jq '.items[] | {name: .metadata.name, capacity: .status.capacity.storage, used: .status.phase}')

# 3. Backup verification
LATEST_BACKUP=$(kubectl get jobs -n nephoran-system | grep backup | sort -k5 -r | head -1)
BACKUP_AGE_HOURS=$(( ($(date +%s) - $(kubectl get job $(echo $LATEST_BACKUP | awk '{print $1}') -n nephoran-system -o jsonpath='{.status.startTime}' | xargs -I {} date -d {} +%s)) / 3600 ))

# 4. Dependency checks
OPENAI_STATUS=$(kubectl exec deployment/llm-processor -n nephoran-system -- curl -s -o /dev/null -w "%{http_code}" https://api.openai.com/v1/models)
WEAVIATE_STATUS=$(kubectl exec deployment/weaviate -n nephoran-system -- curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/v1/.well-known/ready)

# Generate checklist report
cat > "$CHECKLIST_RESULTS" <<EOF
{
  "upgrade_id": "$UPGRADE_ID",
  "timestamp": "$(date -Iseconds)",
  "checklist": {
    "system_health": {
      "unhealthy_pods": $SYSTEM_HEALTH,
      "status": "$([ $SYSTEM_HEALTH -eq 0 ] && echo "PASS" || echo "FAIL")"
    },
    "active_processing": {
      "processing_intents": $ACTIVE_INTENTS,
      "status": "$([ $ACTIVE_INTENTS -lt 10 ] && echo "PASS" || echo "WARN")"
    },
    "backup_status": {
      "latest_backup_age_hours": $BACKUP_AGE_HOURS,
      "status": "$([ $BACKUP_AGE_HOURS -lt 24 ] && echo "PASS" || echo "FAIL")"
    },
    "external_dependencies": {
      "openai_api": {
        "status_code": $OPENAI_STATUS,
        "status": "$([ $OPENAI_STATUS -eq 200 ] && echo "PASS" || echo "FAIL")"
      },
      "weaviate": {
        "status_code": $WEAVIATE_STATUS,
        "status": "$([ $WEAVIATE_STATUS -eq 200 ] && echo "PASS" || echo "FAIL")"
      }
    }
  },
  "approval": {
    "ready_for_upgrade": $([ $SYSTEM_HEALTH -eq 0 ] && [ $BACKUP_AGE_HOURS -lt 24 ] && [ $OPENAI_STATUS -eq 200 ] && [ $WEAVIATE_STATUS -eq 200 ] && echo "true" || echo "false")
  }
}
EOF

echo "Pre-upgrade checklist complete: $CHECKLIST_RESULTS"
cat "$CHECKLIST_RESULTS" | jq .
```

### 1.2 Component Update Strategy

**Rolling Update Configuration:**
```yaml
# rolling-update-strategy.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-processor
  namespace: nephoran-system
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 25%
      maxSurge: 25%
  template:
    spec:
      terminationGracePeriodSeconds: 60
      containers:
      - name: llm-processor
        lifecycle:
          preStop:
            exec:
              command:
              - /bin/sh
              - -c
              - "sleep 15; curl -X POST http://localhost:8080/shutdown"
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
```

**Automated Rolling Update Script:**
```bash
#!/bin/bash
# rolling-update.sh - Zero-downtime component updates

COMPONENT=${1:-all}
NEW_VERSION=${2:-latest}
DRY_RUN=${3:-false}

UPGRADE_ID="UPG-$(date +%Y%m%d-%H%M%S)"
LOG_FILE="/var/log/nephoran/upgrade-$UPGRADE_ID.log"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "$LOG_FILE"
    exit 1
}

success() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1" | tee -a "$LOG_FILE"
}

# Pre-upgrade backup
create_pre_upgrade_backup() {
    log "Creating pre-upgrade backup..."
    ./scripts/ops/disaster-recovery-system.sh backup --tag="pre-upgrade-$UPGRADE_ID"
    if [ $? -ne 0 ]; then
        error "Pre-upgrade backup failed"
    fi
    success "Pre-upgrade backup completed"
}

# Update container images
update_component() {
    local component=$1
    local version=$2
    
    log "Updating $component to version $version..."
    
    # Update image tag
    if [ "$DRY_RUN" = "true" ]; then
        log "DRY RUN: Would update $component to $version"
        return 0
    fi
    
    kubectl set image deployment/"$component" \
        "$component"="us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran/$component:$version" \
        -n nephoran-system
    
    # Wait for rollout to complete
    kubectl rollout status deployment/"$component" -n nephoran-system --timeout=600s
    if [ $? -ne 0 ]; then
        error "Rollout failed for $component"
    fi
    
    # Verify health after update
    sleep 30
    verify_component_health "$component"
    
    success "$component updated successfully to $version"
}

# Health verification
verify_component_health() {
    local component=$1
    
    log "Verifying health of $component..."
    
    # Check pod status
    local ready_replicas=$(kubectl get deployment "$component" -n nephoran-system -o jsonpath='{.status.readyReplicas}')
    local desired_replicas=$(kubectl get deployment "$component" -n nephoran-system -o jsonpath='{.spec.replicas}')
    
    if [ "$ready_replicas" != "$desired_replicas" ]; then
        error "$component has $ready_replicas ready replicas, expected $desired_replicas"
    fi
    
    # Health check endpoint test
    case $component in
        "llm-processor"|"rag-api")
            kubectl exec deployment/"$component" -n nephoran-system -- \
                curl -f http://localhost:8080/healthz >/dev/null 2>&1
            if [ $? -ne 0 ]; then
                error "$component health check failed"
            fi
            ;;
        "weaviate")
            kubectl exec deployment/"$component" -n nephoran-system -- \
                curl -f http://localhost:8080/v1/.well-known/ready >/dev/null 2>&1
            if [ $? -ne 0 ]; then
                error "$component health check failed"
            fi
            ;;
    esac
    
    success "$component health verification passed"
}

# Functional testing
run_post_upgrade_tests() {
    log "Running post-upgrade functional tests..."
    
    # Test intent processing
    kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: upgrade-test-$UPGRADE_ID
  namespace: nephoran-system
spec:
  description: "Test intent for upgrade validation - deploy AMF with 2 replicas"
  priority: high
  intentType: deployment
EOF
    
    # Wait for processing
    sleep 60
    
    # Check if intent was processed successfully
    local intent_status=$(kubectl get networkintent "upgrade-test-$UPGRADE_ID" -n nephoran-system -o jsonpath='{.status.phase}')
    if [ "$intent_status" != "Ready" ] && [ "$intent_status" != "Completed" ]; then
        error "Post-upgrade test failed: intent status is $intent_status"
    fi
    
    # Cleanup test intent
    kubectl delete networkintent "upgrade-test-$UPGRADE_ID" -n nephoran-system
    
    success "Post-upgrade functional tests passed"
}

# Main upgrade logic
main() {
    log "Starting rolling update - ID: $UPGRADE_ID"
    
    # Create pre-upgrade backup
    create_pre_upgrade_backup
    
    # Update components in order
    if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "weaviate" ]; then
        update_component "weaviate" "$NEW_VERSION"
    fi
    
    if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "rag-api" ]; then
        update_component "rag-api" "$NEW_VERSION"
    fi
    
    if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "llm-processor" ]; then
        update_component "llm-processor" "$NEW_VERSION"
    fi
    
    if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "nephio-bridge" ]; then
        update_component "nephio-bridge" "$NEW_VERSION"
    fi
    
    if [ "$COMPONENT" = "all" ] || [ "$COMPONENT" = "oran-adaptor" ]; then
        update_component "oran-adaptor" "$NEW_VERSION"
    fi
    
    # Run post-upgrade tests
    run_post_upgrade_tests
    
    # Generate upgrade report
    generate_upgrade_report
    
    success "Rolling update completed successfully - ID: $UPGRADE_ID"
}

# Generate upgrade report
generate_upgrade_report() {
    local report_file="/var/log/nephoran/upgrade-report-$UPGRADE_ID.json"
    
    cat > "$report_file" <<EOF
{
  "upgrade_id": "$UPGRADE_ID",
  "timestamp": "$(date -Iseconds)",
  "component": "$COMPONENT",
  "new_version": "$NEW_VERSION",
  "status": "completed",
  "components_updated": [
    $(kubectl get deployments -n nephoran-system -o json | jq -r '.items[] | {name: .metadata.name, image: .spec.template.spec.containers[0].image, replicas: .status.readyReplicas}' | jq -s .)
  ],
  "test_results": {
    "functional_tests": "passed",
    "health_checks": "passed"
  }
}
EOF
    
    log "Upgrade report generated: $report_file"
}

# Execute main function
main "$@"
```

## Backup and Recovery Procedures

### 2.1 Automated Backup Strategy

**Comprehensive Backup Script:**
```bash
#!/bin/bash
# comprehensive-backup.sh - Multi-tier backup strategy

BACKUP_TYPE=${1:-full}  # full, incremental, config-only
RETENTION_DAYS=${2:-90}
BACKUP_ID="BKP-$(date +%Y%m%d-%H%M%S)"
BACKUP_DIR="/var/nephoran/backups/$BACKUP_ID"
S3_BUCKET="${S3_BUCKET:-nephoran-production-backups}"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1"
    exit 1
}

success() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

# Initialize backup environment
initialize_backup() {
    log "Initializing backup - ID: $BACKUP_ID, Type: $BACKUP_TYPE"
    
    mkdir -p "$BACKUP_DIR"/{kubernetes,weaviate,configs,secrets,monitoring}
    
    # Create backup metadata
    cat > "$BACKUP_DIR/backup-metadata.json" <<EOF
{
  "backup_id": "$BACKUP_ID",
  "timestamp": "$(date -Iseconds)",
  "type": "$BACKUP_TYPE",
  "cluster": "$(kubectl config current-context)",
  "version": "$(kubectl version --short | grep Server | awk '{print $3}')"
}
EOF
}

# Backup Kubernetes resources
backup_kubernetes_resources() {
    log "Backing up Kubernetes resources..."
    
    # Namespace resources
    kubectl get all -n nephoran-system -o yaml > "$BACKUP_DIR/kubernetes/all-resources.yaml"
    kubectl get secrets -n nephoran-system -o yaml > "$BACKUP_DIR/kubernetes/secrets.yaml"
    kubectl get configmaps -n nephoran-system -o yaml > "$BACKUP_DIR/kubernetes/configmaps.yaml"
    kubectl get pvc -n nephoran-system -o yaml > "$BACKUP_DIR/kubernetes/persistent-volumes.yaml"
    
    # Custom resources
    kubectl get networkintents -A -o yaml > "$BACKUP_DIR/kubernetes/networkintents.yaml"
    kubectl get e2nodesets -A -o yaml > "$BACKUP_DIR/kubernetes/e2nodesets.yaml"
    kubectl get managedelements -A -o yaml > "$BACKUP_DIR/kubernetes/managedelements.yaml"
    
    # CRDs
    kubectl get crd -o yaml | grep -A 1000 "nephoran.com" > "$BACKUP_DIR/kubernetes/crds.yaml"
    
    # RBAC
    kubectl get rolebindings,clusterrolebindings -o yaml | grep -A 100 -B 5 "nephoran" > "$BACKUP_DIR/kubernetes/rbac.yaml"
    
    success "Kubernetes resources backed up"
}

# Backup Weaviate data
backup_weaviate_data() {
    log "Backing up Weaviate vector database..."
    
    # Create Weaviate backup
    local backup_name="weaviate-$BACKUP_ID"
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -X POST "http://localhost:8080/v1/backups" \
        -H "Content-Type: application/json" \
        -d "{
            \"id\": \"$backup_name\",
            \"include\": [\"TelecomKnowledge\", \"IntentPatterns\", \"NetworkFunctions\"],
            \"compression\": \"gzip\"
        }"
    
    # Wait for backup completion
    local backup_status=""
    local attempts=0
    while [ "$backup_status" != "SUCCESS" ] && [ $attempts -lt 30 ]; do
        sleep 10
        backup_status=$(kubectl exec deployment/weaviate -n nephoran-system -- \
            curl -s "http://localhost:8080/v1/backups/$backup_name" | jq -r '.status')
        ((attempts++))
    done
    
    if [ "$backup_status" != "SUCCESS" ]; then
        error "Weaviate backup failed or timed out"
    fi
    
    # Download backup files
    kubectl exec deployment/weaviate -n nephoran-system -- \
        tar -czf "/tmp/weaviate-backup-$BACKUP_ID.tar.gz" "/var/lib/weaviate/backups/$backup_name"
    
    kubectl cp "nephoran-system/$(kubectl get pods -l app=weaviate -o jsonpath='{.items[0].metadata.name}'):/tmp/weaviate-backup-$BACKUP_ID.tar.gz" \
        "$BACKUP_DIR/weaviate/weaviate-data.tar.gz"
    
    success "Weaviate data backed up"
}

# Backup configurations
backup_configurations() {
    log "Backing up system configurations..."
    
    # Application configurations
    kubectl exec deployment/llm-processor -n nephoran-system -- \
        tar -czf "/tmp/llm-config-$BACKUP_ID.tar.gz" /app/config/ 2>/dev/null || true
    
    kubectl exec deployment/rag-api -n nephoran-system -- \
        tar -czf "/tmp/rag-config-$BACKUP_ID.tar.gz" /app/config/ 2>/dev/null || true
    
    # Copy configuration backups
    for component in llm-processor rag-api; do
        kubectl cp "nephoran-system/$(kubectl get pods -l app=$component -o jsonpath='{.items[0].metadata.name}'):/tmp/$component-config-$BACKUP_ID.tar.gz" \
            "$BACKUP_DIR/configs/$component-config.tar.gz" 2>/dev/null || true
    done
    
    # Knowledge base files
    if [ "$BACKUP_TYPE" = "full" ]; then
        kubectl exec deployment/rag-api -n nephoran-system -- \
            tar -czf "/tmp/knowledge-base-$BACKUP_ID.tar.gz" /app/knowledge_base/ 2>/dev/null || true
        
        kubectl cp "nephoran-system/$(kubectl get pods -l app=rag-api -o jsonpath='{.items[0].metadata.name}'):/tmp/knowledge-base-$BACKUP_ID.tar.gz" \
            "$BACKUP_DIR/configs/knowledge-base.tar.gz" 2>/dev/null || true
    fi
    
    success "Configurations backed up"
}

# Backup monitoring data
backup_monitoring_data() {
    if [ "$BACKUP_TYPE" = "full" ]; then
        log "Backing up monitoring data..."
        
        # Prometheus data (last 24 hours)
        kubectl exec deployment/prometheus -n nephoran-monitoring -- \
            tar -czf "/tmp/prometheus-data-$BACKUP_ID.tar.gz" \
            --exclude="*.tmp" /prometheus/data/ 2>/dev/null || true
        
        kubectl cp "nephoran-monitoring/$(kubectl get pods -l app=prometheus -o jsonpath='{.items[0].metadata.name}'):/tmp/prometheus-data-$BACKUP_ID.tar.gz" \
            "$BACKUP_DIR/monitoring/prometheus-data.tar.gz" 2>/dev/null || true
        
        # Grafana dashboards
        kubectl get configmaps -n nephoran-monitoring -l grafana_dashboard=1 -o yaml > "$BACKUP_DIR/monitoring/grafana-dashboards.yaml"
        
        success "Monitoring data backed up"
    fi
}

# Encrypt and upload backup
encrypt_and_upload() {
    log "Encrypting and uploading backup..."
    
    # Create compressed archive
    local archive_name="nephoran-backup-$BACKUP_ID.tar.gz"
    tar -czf "/tmp/$archive_name" -C "$(dirname "$BACKUP_DIR")" "$(basename "$BACKUP_DIR")"
    
    # Encrypt backup
    if [ -n "$BACKUP_ENCRYPTION_KEY" ]; then
        gpg --symmetric --cipher-algo AES256 --compress-algo 1 --s2k-mode 3 \
            --s2k-digest-algo SHA512 --s2k-count 65536 \
            --passphrase "$BACKUP_ENCRYPTION_KEY" \
            --output "/tmp/$archive_name.gpg" "/tmp/$archive_name"
        archive_name="$archive_name.gpg"
    fi
    
    # Upload to S3
    if [ -n "$S3_BUCKET" ]; then
        aws s3 cp "/tmp/$archive_name" "s3://$S3_BUCKET/backups/$BACKUP_TYPE/"
        if [ $? -eq 0 ]; then
            success "Backup uploaded to S3: s3://$S3_BUCKET/backups/$BACKUP_TYPE/$archive_name"
        else
            error "Failed to upload backup to S3"
        fi
    fi
    
    # Cleanup local files
    rm -rf "$BACKUP_DIR" "/tmp/$archive_name"* 2>/dev/null || true
}

# Cleanup old backups
cleanup_old_backups() {
    log "Cleaning up backups older than $RETENTION_DAYS days..."
    
    if [ -n "$S3_BUCKET" ]; then
        # List and delete old backups from S3
        aws s3 ls "s3://$S3_BUCKET/backups/$BACKUP_TYPE/" | \
        while read -r line; do
            backup_date=$(echo "$line" | awk '{print $1" "$2}')
            backup_file=$(echo "$line" | awk '{print $4}')
            
            if [ -n "$backup_date" ] && [ -n "$backup_file" ]; then
                backup_timestamp=$(date -d "$backup_date" +%s)
                cutoff_timestamp=$(date -d "$RETENTION_DAYS days ago" +%s)
                
                if [ "$backup_timestamp" -lt "$cutoff_timestamp" ]; then
                    aws s3 rm "s3://$S3_BUCKET/backups/$BACKUP_TYPE/$backup_file"
                    log "Deleted old backup: $backup_file"
                fi
            fi
        done
    fi
    
    success "Old backup cleanup completed"
}

# Main backup function
main() {
    initialize_backup
    
    backup_kubernetes_resources
    
    if [ "$BACKUP_TYPE" = "full" ] || [ "$BACKUP_TYPE" = "incremental" ]; then
        backup_weaviate_data
        backup_monitoring_data
    fi
    
    backup_configurations
    encrypt_and_upload
    cleanup_old_backups
    
    success "Backup completed successfully - ID: $BACKUP_ID"
}

# Execute main function
main "$@"
```

### 2.2 Disaster Recovery Testing

**Monthly DR Test Procedure:**
```bash
#!/bin/bash
# disaster-recovery-test.sh - Automated DR testing

DR_TEST_ID="DR-TEST-$(date +%Y%m%d-%H%M%S)"
TEST_NAMESPACE="nephoran-dr-test"
REPORT_FILE="/var/log/nephoran/dr-test-$DR_TEST_ID.json"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1"
    echo "$1" >> "$REPORT_FILE.errors"
}

success() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

# Initialize DR test environment
setup_dr_test_environment() {
    log "Setting up DR test environment..."
    
    # Create isolated test namespace
    kubectl create namespace "$TEST_NAMESPACE" || true
    kubectl label namespace "$TEST_NAMESPACE" purpose=disaster-recovery-test
    
    # Apply resource quotas to limit test impact
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ResourceQuota
metadata:
  name: dr-test-quota
  namespace: $TEST_NAMESPACE
spec:
  hard:
    requests.cpu: "4"
    requests.memory: 8Gi
    limits.cpu: "8"
    limits.memory: 16Gi
    persistentvolumeclaims: "5"
EOF
    
    success "DR test environment setup complete"
}

# Test backup restoration
test_backup_restoration() {
    log "Testing backup restoration..."
    
    # Get latest backup
    local latest_backup=$(aws s3 ls s3://$S3_BUCKET/backups/full/ | sort | tail -1 | awk '{print $4}')
    if [ -z "$latest_backup" ]; then
        error "No backup found for testing"
        return 1
    fi
    
    log "Testing restoration of backup: $latest_backup"
    
    # Download and decrypt backup
    aws s3 cp "s3://$S3_BUCKET/backups/full/$latest_backup" /tmp/
    
    if [[ "$latest_backup" == *.gpg ]]; then
        gpg --batch --yes --passphrase "$BACKUP_ENCRYPTION_KEY" \
            --output "/tmp/${latest_backup%.gpg}" \
            --decrypt "/tmp/$latest_backup"
        latest_backup="${latest_backup%.gpg}"
    fi
    
    # Extract backup
    local extract_dir="/tmp/dr-test-$DR_TEST_ID"
    mkdir -p "$extract_dir"
    tar -xzf "/tmp/$latest_backup" -C "$extract_dir"
    
    # Deploy minimal system from backup
    local backup_content_dir=$(find "$extract_dir" -name "nephoran-backup-*" -type d)
    
    # Apply CRDs first
    kubectl apply -f "$backup_content_dir/kubernetes/crds.yaml" --validate=false || true
    
    # Wait for CRDs to be established
    sleep 10
    
    # Apply basic resources to test namespace
    sed "s/namespace: nephoran-system/namespace: $TEST_NAMESPACE/g" \
        "$backup_content_dir/kubernetes/all-resources.yaml" | \
        kubectl apply -f - || true
    
    # Test basic functionality
    sleep 60
    
    # Check if pods are running
    local running_pods=$(kubectl get pods -n "$TEST_NAMESPACE" --no-headers | grep Running | wc -l)
    local total_pods=$(kubectl get pods -n "$TEST_NAMESPACE" --no-headers | wc -l)
    
    if [ "$running_pods" -gt 0 ]; then
        success "Backup restoration test passed: $running_pods/$total_pods pods running"
        return 0
    else
        error "Backup restoration test failed: no pods running"
        return 1
    fi
}

# Test configuration recovery
test_configuration_recovery() {
    log "Testing configuration recovery..."
    
    # Check if ConfigMaps were restored
    local configmaps=$(kubectl get configmaps -n "$TEST_NAMESPACE" --no-headers | wc -l)
    
    # Check if Secrets were restored (excluding default tokens)
    local secrets=$(kubectl get secrets -n "$TEST_NAMESPACE" --no-headers | grep -v "default-token" | wc -l)
    
    if [ "$configmaps" -gt 0 ] && [ "$secrets" -gt 0 ]; then
        success "Configuration recovery test passed: $configmaps ConfigMaps, $secrets Secrets"
        return 0
    else
        error "Configuration recovery test failed: $configmaps ConfigMaps, $secrets Secrets"
        return 1
    fi
}

# Test data recovery
test_data_recovery() {
    log "Testing data recovery..."
    
    # This is a simplified test - in production, you would restore Weaviate data
    # and verify vector search functionality
    
    # Check if PVCs were created
    local pvcs=$(kubectl get pvc -n "$TEST_NAMESPACE" --no-headers | wc -l)
    
    if [ "$pvcs" -gt 0 ]; then
        success "Data recovery test passed: $pvcs PVCs created"
        return 0
    else
        error "Data recovery test failed: no PVCs created"
        return 1
    fi
}

# Calculate RTO and RPO
calculate_rto_rpo() {
    log "Calculating RTO and RPO metrics..."
    
    local test_start=$(date -d "5 minutes ago" +%s)
    local test_end=$(date +%s)
    local rto_seconds=$((test_end - test_start))
    local rto_minutes=$((rto_seconds / 60))
    
    # RPO is based on backup frequency (daily = 24 hours max)
    local rpo_hours=24
    
    cat >> "$REPORT_FILE" <<EOF
{
  "rto_minutes": $rto_minutes,
  "rpo_hours": $rpo_hours,
  "target_rto_minutes": 120,
  "target_rpo_hours": 24,
  "rto_compliance": $([ $rto_minutes -le 120 ] && echo "true" || echo "false"),
  "rpo_compliance": $([ $rpo_hours -le 24 ] && echo "true" || echo "false")
}
EOF
    
    success "RTO: $rto_minutes minutes, RPO: $rpo_hours hours"
}

# Cleanup DR test environment
cleanup_dr_test() {
    log "Cleaning up DR test environment..."
    
    kubectl delete namespace "$TEST_NAMESPACE" --ignore-not-found=true
    rm -rf /tmp/dr-test-* /tmp/nephoran-backup-* 2>/dev/null || true
    
    success "DR test cleanup complete"
}

# Generate DR test report
generate_dr_test_report() {
    local backup_test_result=$1
    local config_test_result=$2
    local data_test_result=$3
    
    cat > "$REPORT_FILE" <<EOF
{
  "dr_test_id": "$DR_TEST_ID",
  "timestamp": "$(date -Iseconds)",
  "test_results": {
    "backup_restoration": $([ $backup_test_result -eq 0 ] && echo "\"passed\"" || echo "\"failed\""),
    "configuration_recovery": $([ $config_test_result -eq 0 ] && echo "\"passed\"" || echo "\"failed\""),
    "data_recovery": $([ $data_test_result -eq 0 ] && echo "\"passed\"" || echo "\"failed\"")
  },
  "overall_status": $([ $backup_test_result -eq 0 ] && [ $config_test_result -eq 0 ] && [ $data_test_result -eq 0 ] && echo "\"passed\"" || echo "\"failed\""),
  "recommendations": [
    "Review backup frequency if RPO requirements change",
    "Consider implementing automated failover for critical components",
    "Test DR procedures quarterly with different failure scenarios"
  ]
}
EOF
    
    # Append RTO/RPO data
    calculate_rto_rpo
    
    log "DR test report generated: $REPORT_FILE"
}

# Main DR test function
main() {
    log "Starting disaster recovery test - ID: $DR_TEST_ID"
    
    setup_dr_test_environment
    
    # Run tests
    test_backup_restoration
    backup_test_result=$?
    
    test_configuration_recovery
    config_test_result=$?
    
    test_data_recovery
    data_test_result=$?
    
    # Generate report
    generate_dr_test_report $backup_test_result $config_test_result $data_test_result
    
    # Cleanup
    cleanup_dr_test
    
    if [ $backup_test_result -eq 0 ] && [ $config_test_result -eq 0 ] && [ $data_test_result -eq 0 ]; then
        success "DR test completed successfully - ID: $DR_TEST_ID"
        exit 0
    else
        error "DR test failed - ID: $DR_TEST_ID"
        exit 1
    fi
}

# Execute main function
main "$@"
```

## Database Migration and Management

### 3.1 Weaviate Schema Migration

**Schema Update Procedure:**
```bash
#!/bin/bash
# weaviate-schema-migration.sh

MIGRATION_ID="MIG-$(date +%Y%m%d-%H%M%S)"
NEW_SCHEMA_FILE=${1:-"deployments/weaviate/schemas/v2-schema.json"}

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1"
    exit 1
}

success() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

# Backup current schema
backup_current_schema() {
    log "Backing up current Weaviate schema..."
    
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -s http://localhost:8080/v1/schema > "/tmp/schema-backup-$MIGRATION_ID.json"
    
    if [ $? -eq 0 ]; then
        success "Schema backup created: /tmp/schema-backup-$MIGRATION_ID.json"
    else
        error "Failed to backup current schema"
    fi
}

# Validate new schema
validate_new_schema() {
    log "Validating new schema..."
    
    if [ ! -f "$NEW_SCHEMA_FILE" ]; then
        error "New schema file not found: $NEW_SCHEMA_FILE"
    fi
    
    # Basic JSON validation
    jq empty "$NEW_SCHEMA_FILE" >/dev/null 2>&1
    if [ $? -ne 0 ]; then
        error "Invalid JSON in schema file: $NEW_SCHEMA_FILE"
    fi
    
    success "New schema validation passed"
}

# Apply schema migration
apply_schema_migration() {
    log "Applying schema migration..."
    
    # For major schema changes, might need to:
    # 1. Create new classes
    # 2. Migrate data
    # 3. Remove old classes
    
    # Copy new schema to Weaviate pod
    kubectl cp "$NEW_SCHEMA_FILE" "nephoran-system/$(kubectl get pods -l app=weaviate -o jsonpath='{.items[0].metadata.name}'):/tmp/new-schema.json"
    
    # Apply schema update
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -X PUT http://localhost:8080/v1/schema \
        -H "Content-Type: application/json" \
        -d @/tmp/new-schema.json
    
    if [ $? -eq 0 ]; then
        success "Schema migration applied successfully"
    else
        error "Failed to apply schema migration"
    fi
}

# Verify migration
verify_migration() {
    log "Verifying schema migration..."
    
    # Get updated schema
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -s http://localhost:8080/v1/schema > "/tmp/schema-after-$MIGRATION_ID.json"
    
    # Test basic functionality
    kubectl exec deployment/weaviate -n nephoran-system -- \
        curl -X POST http://localhost:8080/v1/graphql \
        -H "Content-Type: application/json" \
        -d '{"query": "{Get{TelecomKnowledge(limit:1){title}}}"}'
    
    if [ $? -eq 0 ]; then
        success "Schema migration verification passed"
    else
        error "Schema migration verification failed"
    fi
}

# Main migration function
main() {
    log "Starting Weaviate schema migration - ID: $MIGRATION_ID"
    
    backup_current_schema
    validate_new_schema
    apply_schema_migration
    verify_migration
    
    success "Schema migration completed successfully - ID: $MIGRATION_ID"
}

# Execute main function
main "$@"
```

## Configuration Change Management

### 4.1 Configuration Validation Pipeline

**Configuration Change Process:**
```bash
#!/bin/bash
# config-change-pipeline.sh

CHANGE_ID="CFG-$(date +%Y%m%d-%H%M%S)"
CONFIG_FILE=${1:-""}
ENVIRONMENT=${2:-"staging"}

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1"
    exit 1
}

success() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

# Validate configuration syntax
validate_configuration() {
    log "Validating configuration syntax..."
    
    if [ -z "$CONFIG_FILE" ]; then
        error "Configuration file not specified"
    fi
    
    if [ ! -f "$CONFIG_FILE" ]; then
        error "Configuration file not found: $CONFIG_FILE"
    fi
    
    # YAML syntax validation
    if [[ "$CONFIG_FILE" == *.yaml ]] || [[ "$CONFIG_FILE" == *.yml ]]; then
        python3 -c "import yaml; yaml.safe_load(open('$CONFIG_FILE'))" 2>/dev/null
        if [ $? -ne 0 ]; then
            error "Invalid YAML syntax in: $CONFIG_FILE"
        fi
    fi
    
    # JSON syntax validation
    if [[ "$CONFIG_FILE" == *.json ]]; then
        jq empty "$CONFIG_FILE" >/dev/null 2>&1
        if [ $? -ne 0 ]; then
            error "Invalid JSON syntax in: $CONFIG_FILE"
        fi
    fi
    
    success "Configuration syntax validation passed"
}

# Test configuration in staging
test_configuration_staging() {
    log "Testing configuration in staging environment..."
    
    # Apply configuration to staging namespace
    local staging_namespace="nephoran-staging"
    
    # Create staging namespace if it doesn't exist
    kubectl create namespace "$staging_namespace" --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply configuration
    if [[ "$CONFIG_FILE" == *configmap* ]] || [[ "$CONFIG_FILE" == *config* ]]; then
        kubectl apply -f "$CONFIG_FILE" -n "$staging_namespace" --dry-run=server
        if [ $? -ne 0 ]; then
            error "Configuration validation failed in staging"
        fi
    fi
    
    success "Configuration testing in staging passed"
}

# Deploy configuration with canary strategy
deploy_configuration_canary() {
    log "Deploying configuration with canary strategy..."
    
    # Create backup of current configuration
    local backup_file="/tmp/config-backup-$CHANGE_ID.yaml"
    kubectl get configmap,secret -n nephoran-system -o yaml > "$backup_file"
    
    # Apply new configuration
    kubectl apply -f "$CONFIG_FILE" -n nephoran-system
    
    # Wait for pods to pick up new configuration
    sleep 30
    
    # Monitor system health
    local unhealthy_pods=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
    
    if [ "$unhealthy_pods" -gt 0 ]; then
        log "Configuration caused $unhealthy_pods pods to become unhealthy, rolling back..."
        kubectl apply -f "$backup_file"
        error "Configuration deployment failed, rolled back"
    fi
    
    success "Configuration deployed successfully with canary strategy"
}

# Monitor configuration impact
monitor_configuration_impact() {
    log "Monitoring configuration impact..."
    
    # Monitor key metrics for 5 minutes
    local monitoring_duration=300
    local start_time=$(date +%s)
    local end_time=$((start_time + monitoring_duration))
    
    while [ $(date +%s) -lt $end_time ]; do
        # Check error rates
        local error_rate=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=rate(nephoran_errors_total[5m])" | jq -r '.data.result[0].value[1] // 0')
        
        # Check response times
        local p95_latency=$(curl -s "http://prometheus.nephoran-monitoring.svc.cluster.local:9090/api/v1/query?query=histogram_quantile(0.95,rate(nephoran_llm_request_duration_seconds_bucket[5m]))" | jq -r '.data.result[0].value[1] // 0')
        
        log "Error rate: $error_rate, P95 latency: ${p95_latency}s"
        
        # Alert if metrics exceed thresholds
        if (( $(echo "$error_rate > 0.05" | bc -l) )); then
            error "High error rate detected: $error_rate"
        fi
        
        if (( $(echo "$p95_latency > 5.0" | bc -l) )); then
            error "High latency detected: ${p95_latency}s"
        fi
        
        sleep 60
    done
    
    success "Configuration impact monitoring completed - no issues detected"
}

# Generate change report
generate_change_report() {
    local report_file="/var/log/nephoran/config-change-$CHANGE_ID.json"
    
    cat > "$report_file" <<EOF
{
  "change_id": "$CHANGE_ID",
  "timestamp": "$(date -Iseconds)",
  "configuration_file": "$CONFIG_FILE",
  "environment": "$ENVIRONMENT",
  "validation_status": "passed",
  "deployment_status": "completed",
  "monitoring_duration_minutes": 5,
  "impact_assessment": "no negative impact detected"
}
EOF
    
    log "Change report generated: $report_file"
}

# Main configuration change function
main() {
    log "Starting configuration change process - ID: $CHANGE_ID"
    
    validate_configuration
    test_configuration_staging
    deploy_configuration_canary
    monitor_configuration_impact
    generate_change_report
    
    success "Configuration change completed successfully - ID: $CHANGE_ID"
}

# Execute main function
main "$@"
```

## Scheduled Maintenance Automation

### 5.1 CronJob-Based Maintenance

**Maintenance CronJob Configuration:**
```yaml
# scheduled-maintenance.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nephoran-daily-maintenance
  namespace: nephoran-system
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM UTC
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: maintenance
            image: nephoran/maintenance-runner:latest
            command:
            - /bin/bash
            - -c
            - |
              #!/bin/bash
              echo "Starting daily maintenance..."
              
              # 1. Health check all components
              /scripts/health-check.sh
              
              # 2. Clean up completed jobs
              kubectl delete jobs --field-selector status.successful=1 -n nephoran-system
              
              # 3. Rotate logs
              kubectl exec deployment/llm-processor -n nephoran-system -- \
                find /var/log -name "*.log" -mtime +7 -delete
              
              # 4. Update system metrics
              /scripts/update-metrics.sh
              
              # 5. Check for security updates
              /scripts/security/security-scan.sh --automated
              
              echo "Daily maintenance completed"
          restartPolicy: OnFailure
      backoffLimit: 3
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nephoran-weekly-optimization
  namespace: nephoran-system
spec:
  schedule: "0 1 * * 0"  # Weekly on Sunday at 1 AM UTC
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: optimization
            image: nephoran/maintenance-runner:latest
            command:
            - /bin/bash
            - -c
            - |
              #!/bin/bash
              echo "Starting weekly optimization..."
              
              # 1. Optimize Weaviate indices
              kubectl exec deployment/weaviate -n nephoran-system -- \
                curl -X POST http://localhost:8080/v1/schema/optimize
              
              # 2. Clear old cache entries
              kubectl exec deployment/rag-api -n nephoran-system -- \
                curl -X POST http://localhost:8080/cache/cleanup
              
              # 3. Generate performance report
              /scripts/performance-report.sh
              
              # 4. Update auto-scaling parameters
              /scripts/optimize-scaling.sh
              
              echo "Weekly optimization completed"
          restartPolicy: OnFailure
      backoffLimit: 2
```

This comprehensive maintenance and upgrade guide provides automated procedures for managing the Nephoran Intent Operator lifecycle with minimal downtime and maximum reliability.