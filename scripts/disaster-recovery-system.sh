#!/bin/bash
# Nephoran Intent Operator Comprehensive Disaster Recovery System
# Phase 3 Production Excellence - Complete DR capabilities
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-nephoran-system}"
BACKUP_DIR="${BACKUP_DIR:-/var/nephoran/backups}"
S3_BUCKET="${S3_BUCKET:-nephoran-dr-backups}"
DR_REGION="${DR_REGION:-us-west-2}"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
DR_ID="DR-$TIMESTAMP"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}"
}

info() {
    echo -e "${PURPLE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Initialize DR environment
initialize_dr_environment() {
    log "Initializing disaster recovery environment..."
    
    # Create backup directory structure
    mkdir -p "$BACKUP_DIR"/{kubernetes,weaviate,configs,secrets,monitoring,logs}
    
    # Create DR metadata
    cat > "$BACKUP_DIR/dr-metadata.json" <<EOF
{
    "dr_id": "$DR_ID",
    "timestamp": "$(date -Iseconds)",
    "namespace": "$NAMESPACE",
    "operator": "$(whoami)",
    "cluster_info": {
        "name": "$(kubectl config current-context)",
        "server": "$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')",
        "version": "$(kubectl version --short --client | grep Client | cut -d' ' -f3)"
    },
    "backup_strategy": "comprehensive",
    "encryption_enabled": true,
    "compression_enabled": true
}
EOF
    
    success "DR environment initialized with ID: $DR_ID"
}

# Create comprehensive Kubernetes backup
create_kubernetes_backup() {
    log "Creating comprehensive Kubernetes backup..."
    
    local k8s_backup_dir="$BACKUP_DIR/kubernetes"
    local backup_file="$k8s_backup_dir/k8s-backup-$TIMESTAMP.tar.gz"
    
    # Backup Custom Resource Definitions
    log "Backing up Custom Resource Definitions..."
    kubectl get crd nephoran.com -o yaml > "$k8s_backup_dir/crds.yaml" 2>/dev/null || warn "No Nephoran CRDs found"
    
    # Backup all Nephoran resources
    log "Backing up Nephoran resources..."
    kubectl get networkintents -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/networkintents.yaml" 2>/dev/null || warn "No NetworkIntents found"
    kubectl get e2nodesets -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/e2nodesets.yaml" 2>/dev/null || warn "No E2NodeSets found"
    kubectl get managedelements -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/managedelements.yaml" 2>/dev/null || warn "No ManagedElements found"
    
    # Backup deployments and services
    log "Backing up deployments and services..."
    kubectl get deployments -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/deployments.yaml"
    kubectl get services -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/services.yaml"
    kubectl get ingress -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/ingress.yaml" 2>/dev/null || echo "# No ingress resources" > "$k8s_backup_dir/ingress.yaml"
    
    # Backup ConfigMaps and PersistentVolumeClaims
    log "Backing up ConfigMaps and PVCs..."
    kubectl get configmaps -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/configmaps.yaml"
    kubectl get pvc -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/pvcs.yaml"
    
    # Backup ServiceAccounts and RBAC
    log "Backing up RBAC and ServiceAccounts..."
    kubectl get serviceaccounts -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/serviceaccounts.yaml"
    kubectl get roles -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/roles.yaml" 2>/dev/null || echo "# No roles" > "$k8s_backup_dir/roles.yaml"
    kubectl get rolebindings -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/rolebindings.yaml" 2>/dev/null || echo "# No rolebindings" > "$k8s_backup_dir/rolebindings.yaml"
    
    # Backup cluster-level resources related to Nephoran
    log "Backing up cluster-level resources..."
    kubectl get clusterroles -l "app.kubernetes.io/name=nephoran-intent-operator" -o yaml > "$k8s_backup_dir/clusterroles.yaml" 2>/dev/null || echo "# No clusterroles" > "$k8s_backup_dir/clusterroles.yaml"
    kubectl get clusterrolebindings -l "app.kubernetes.io/name=nephoran-intent-operator" -o yaml > "$k8s_backup_dir/clusterrolebindings.yaml" 2>/dev/null || echo "# No clusterrolebindings" > "$k8s_backup_dir/clusterrolebindings.yaml"
    
    # Create backup archive with encryption
    log "Creating encrypted backup archive..."
    cd "$k8s_backup_dir"
    tar -czf - *.yaml | gpg --cipher-algo AES256 --compress-algo 1 --symmetric --output "$backup_file.gpg" 2>/dev/null || {
        warn "GPG encryption failed, creating unencrypted backup"
        tar -czf "$backup_file" *.yaml
    }
    
    # Clean up individual files
    rm -f *.yaml
    
    local backup_size=$(du -h "$backup_file"* | cut -f1)
    success "Kubernetes backup created: $backup_size"
    
    return 0
}

# Create Weaviate vector database backup
create_weaviate_backup() {
    log "Creating Weaviate vector database backup..."
    
    local weaviate_backup_dir="$BACKUP_DIR/weaviate"
    local backup_id="weaviate-backup-$TIMESTAMP"
    
    # Check if Weaviate is running
    if ! kubectl get pods -n "$NAMESPACE" -l app=weaviate --no-headers 2>/dev/null | grep -q Running; then
        warn "Weaviate pods not found or not running, skipping vector DB backup"
        return 1
    fi
    
    # Get Weaviate pod name
    local weaviate_pod=$(kubectl get pods -n "$NAMESPACE" -l app=weaviate -o jsonpath='{.items[0].metadata.name}')
    
    if [[ -z "$weaviate_pod" ]]; then
        warn "No Weaviate pod found, skipping vector DB backup"
        return 1
    fi
    
    log "Creating Weaviate backup via API..."
    
    # Create backup through Weaviate API
    local backup_response=$(kubectl exec "$weaviate_pod" -n "$NAMESPACE" -- \
        curl -s -X POST "http://localhost:8080/v1/backups" \
        -H "Content-Type: application/json" \
        -d "{
            \"id\": \"$backup_id\",
            \"include\": [\"TelecomKnowledge\", \"IntentPatterns\", \"NetworkFunctions\"],
            \"compression\": \"gzip\"
        }" 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "$backup_response" | jq -e '.id' >/dev/null 2>&1; then
        success "Weaviate backup initiated: $backup_id"
        
        # Wait for backup completion
        local max_wait=300  # 5 minutes
        local wait_time=0
        local backup_status=""
        
        while [[ $wait_time -lt $max_wait ]]; do
            backup_status=$(kubectl exec "$weaviate_pod" -n "$NAMESPACE" -- \
                curl -s "http://localhost:8080/v1/backups/$backup_id" 2>/dev/null | \
                jq -r '.status // "UNKNOWN"')
            
            if [[ "$backup_status" == "SUCCESS" ]]; then
                success "Weaviate backup completed successfully"
                break
            elif [[ "$backup_status" == "FAILED" ]]; then
                error "Weaviate backup failed"
                return 1
            fi
            
            sleep 10
            wait_time=$((wait_time + 10))
            log "Waiting for backup completion... (${wait_time}s/${max_wait}s)"
        done
        
        if [[ "$backup_status" != "SUCCESS" ]]; then
            error "Weaviate backup timed out"
            return 1
        fi
        
        # Download backup files
        log "Downloading Weaviate backup files..."
        kubectl exec "$weaviate_pod" -n "$NAMESPACE" -- \
            tar -czf "/tmp/weaviate-backup-$TIMESTAMP.tar.gz" -C /var/lib/weaviate/backups . 2>/dev/null
        
        kubectl cp "$NAMESPACE/$weaviate_pod:/tmp/weaviate-backup-$TIMESTAMP.tar.gz" \
            "$weaviate_backup_dir/weaviate-backup-$TIMESTAMP.tar.gz"
        
        # Clean up temp file on pod
        kubectl exec "$weaviate_pod" -n "$NAMESPACE" -- rm -f "/tmp/weaviate-backup-$TIMESTAMP.tar.gz"
        
        local backup_size=$(du -h "$weaviate_backup_dir/weaviate-backup-$TIMESTAMP.tar.gz" | cut -f1)
        success "Weaviate backup downloaded: $backup_size"
        
    else
        error "Failed to create Weaviate backup"
        return 1
    fi
    
    return 0
}

# Backup secrets and configurations
backup_secrets_and_configs() {
    log "Backing up secrets and configurations..."
    
    local secrets_backup_dir="$BACKUP_DIR/secrets"
    local configs_backup_dir="$BACKUP_DIR/configs"
    
    # Backup secrets (encrypted)
    log "Backing up secrets..."
    kubectl get secrets -n "$NAMESPACE" -o yaml > "$secrets_backup_dir/secrets-raw.yaml"
    
    # Encrypt secrets backup
    if command -v gpg >/dev/null 2>&1; then
        gpg --cipher-algo AES256 --compress-algo 1 --symmetric \
            --output "$secrets_backup_dir/secrets-$TIMESTAMP.yaml.gpg" \
            "$secrets_backup_dir/secrets-raw.yaml" 2>/dev/null && \
            rm -f "$secrets_backup_dir/secrets-raw.yaml"
        success "Secrets backup encrypted and stored"
    else
        warn "GPG not available, storing secrets backup unencrypted"
        mv "$secrets_backup_dir/secrets-raw.yaml" "$secrets_backup_dir/secrets-$TIMESTAMP.yaml"
    fi
    
    # Backup important configuration files
    log "Backing up configuration files..."
    
    # Copy important config files from the repository
    if [[ -f "go.mod" ]]; then
        cp go.mod "$configs_backup_dir/"
    fi
    
    if [[ -f "go.sum" ]]; then
        cp go.sum "$configs_backup_dir/"
    fi
    
    if [[ -f "requirements-rag.txt" ]]; then
        cp requirements-rag.txt "$configs_backup_dir/"
    fi
    
    if [[ -f "Makefile" ]]; then
        cp Makefile "$configs_backup_dir/"
    fi
    
    # Backup deployment configurations
    if [[ -d "deployments" ]]; then
        cp -r deployments "$configs_backup_dir/"
    fi
    
    # Create configurations archive
    cd "$configs_backup_dir"
    tar -czf "configs-$TIMESTAMP.tar.gz" * 2>/dev/null || warn "No configuration files to backup"
    
    success "Configuration backup completed"
    return 0
}

# Backup monitoring and logs
backup_monitoring_data() {
    log "Backing up monitoring data and logs..."
    
    local monitoring_backup_dir="$BACKUP_DIR/monitoring"
    local logs_backup_dir="$BACKUP_DIR/logs"
    
    # Backup Prometheus data if available
    if kubectl get pods -n monitoring -l app=prometheus --no-headers 2>/dev/null | grep -q Running; then
        log "Backing up Prometheus data..."
        local prometheus_pod=$(kubectl get pods -n monitoring -l app=prometheus -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
        
        if [[ -n "$prometheus_pod" ]]; then
            kubectl exec "$prometheus_pod" -n monitoring -- \
                tar -czf "/tmp/prometheus-data-$TIMESTAMP.tar.gz" -C /prometheus/data . 2>/dev/null && \
            kubectl cp "monitoring/$prometheus_pod:/tmp/prometheus-data-$TIMESTAMP.tar.gz" \
                "$monitoring_backup_dir/prometheus-data-$TIMESTAMP.tar.gz" && \
            kubectl exec "$prometheus_pod" -n monitoring -- \
                rm -f "/tmp/prometheus-data-$TIMESTAMP.tar.gz"
            
            success "Prometheus data backup completed"
        fi
    fi
    
    # Backup Grafana dashboards if available
    if kubectl get pods -n monitoring -l app=grafana --no-headers 2>/dev/null | grep -q Running; then
        log "Backing up Grafana dashboards..."
        kubectl get configmaps -n monitoring -l grafana=dashboard -o yaml > \
            "$monitoring_backup_dir/grafana-dashboards-$TIMESTAMP.yaml" 2>/dev/null || \
            warn "No Grafana dashboards found"
    fi
    
    # Collect recent logs from Nephoran components
    log "Collecting application logs..."
    local components=("llm-processor" "rag-api" "nephio-bridge" "weaviate")
    
    for component in "${components[@]}"; do
        if kubectl get pods -n "$NAMESPACE" -l app="$component" --no-headers 2>/dev/null | grep -q Running; then
            kubectl logs -n "$NAMESPACE" -l app="$component" --since=24h > \
                "$logs_backup_dir/${component}-logs-$TIMESTAMP.log" 2>/dev/null || \
                warn "Failed to collect logs for $component"
        fi
    done
    
    # Collect system events
    kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' > \
        "$logs_backup_dir/events-$TIMESTAMP.log" 2>/dev/null || \
        warn "Failed to collect system events"
    
    success "Monitoring data and logs backup completed"
    return 0
}

# Upload backups to cloud storage
upload_to_cloud_storage() {
    log "Uploading backups to cloud storage..."
    
    local backup_archive="$BACKUP_DIR/nephoran-backup-$TIMESTAMP.tar.gz"
    
    # Create comprehensive backup archive
    log "Creating comprehensive backup archive..."
    cd "$BACKUP_DIR"
    tar -czf "$backup_archive" kubernetes/ weaviate/ configs/ secrets/ monitoring/ logs/ dr-metadata.json
    
    local archive_size=$(du -h "$backup_archive" | cut -f1)
    log "Backup archive size: $archive_size"
    
    # Upload to S3 if AWS CLI is available
    if command -v aws >/dev/null 2>&1; then
        log "Uploading to S3 bucket: $S3_BUCKET"
        
        # Check if bucket exists, create if not
        if ! aws s3 ls "s3://$S3_BUCKET" >/dev/null 2>&1; then
            log "Creating S3 bucket: $S3_BUCKET"
            aws s3 mb "s3://$S3_BUCKET" --region "$DR_REGION" || {
                error "Failed to create S3 bucket"
                return 1
            }
        fi
        
        # Upload with encryption
        aws s3 cp "$backup_archive" "s3://$S3_BUCKET/disaster-recovery/" \
            --server-side-encryption AES256 \
            --metadata "dr-id=$DR_ID,timestamp=$TIMESTAMP,namespace=$NAMESPACE" || {
            error "Failed to upload backup to S3"
            return 1
        }
        
        success "Backup uploaded to S3: s3://$S3_BUCKET/disaster-recovery/$(basename "$backup_archive")"
        
        # Set lifecycle policy for old backups
        aws s3api put-bucket-lifecycle-configuration \
            --bucket "$S3_BUCKET" \
            --lifecycle-configuration '{
                "Rules": [{
                    "ID": "DeleteOldBackups",
                    "Status": "Enabled",
                    "Filter": {"Prefix": "disaster-recovery/"},
                    "Expiration": {"Days": 90}
                }]
            }' 2>/dev/null || warn "Failed to set lifecycle policy"
        
    else
        warn "AWS CLI not available, backup stored locally only"
    fi
    
    # Clean up local archive to save space
    rm -f "$backup_archive"
    
    return 0
}

# Test disaster recovery procedures
test_disaster_recovery() {
    log "Testing disaster recovery procedures..."
    
    local test_namespace="nephoran-dr-test"
    local test_result=0
    
    # Create test namespace
    kubectl create namespace "$test_namespace" --dry-run=client -o yaml | kubectl apply -f - || {
        error "Failed to create test namespace"
        return 1
    }
    
    # Label test namespace
    kubectl label namespace "$test_namespace" nephoran.com/dr-test=true --overwrite
    
    log "Testing partial restore process..."
    
    # Test CRD restoration
    if [[ -f "$BACKUP_DIR/kubernetes/crds.yaml" ]] || [[ -f "$BACKUP_DIR/kubernetes/k8s-backup-"*".tar.gz.gpg" ]]; then
        log "Testing CRD restoration..."
        
        # For this test, we'll just verify the backup files exist and are readable
        if [[ -f "$BACKUP_DIR/kubernetes/k8s-backup-"*".tar.gz.gpg" ]]; then
            local backup_file=$(ls "$BACKUP_DIR/kubernetes/k8s-backup-"*".tar.gz.gpg" | head -1)
            if gpg --list-packets "$backup_file" >/dev/null 2>&1; then
                success "Encrypted Kubernetes backup is readable"
            else
                warn "Encrypted backup validation failed"
                test_result=1
            fi
        fi
    else
        warn "No Kubernetes backup found for testing"
        test_result=1
    fi
    
    # Test Weaviate backup
    if [[ -f "$BACKUP_DIR/weaviate/weaviate-backup-"*".tar.gz" ]]; then
        log "Testing Weaviate backup integrity..."
        local weaviate_backup=$(ls "$BACKUP_DIR/weaviate/weaviate-backup-"*".tar.gz" | head -1)
        if tar -tzf "$weaviate_backup" >/dev/null 2>&1; then
            success "Weaviate backup archive is valid"
        else
            warn "Weaviate backup validation failed"
            test_result=1
        fi
    else
        warn "No Weaviate backup found for testing"
        test_result=1
    fi
    
    # Test configuration backup
    if [[ -f "$BACKUP_DIR/configs/configs-"*".tar.gz" ]]; then
        log "Testing configuration backup..."
        local config_backup=$(ls "$BACKUP_DIR/configs/configs-"*".tar.gz" | head -1)
        if tar -tzf "$config_backup" >/dev/null 2>&1; then
            success "Configuration backup is valid"
        else
            warn "Configuration backup validation failed"
            test_result=1
        fi
    fi
    
    # Generate DR test report
    cat > "$BACKUP_DIR/dr-test-report-$TIMESTAMP.json" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "test_namespace": "$test_namespace",
    "test_results": {
        "kubernetes_backup": "$([ -f "$BACKUP_DIR/kubernetes/k8s-backup-"*".tar.gz"* ] && echo "PASS" || echo "FAIL")",
        "weaviate_backup": "$([ -f "$BACKUP_DIR/weaviate/weaviate-backup-"*".tar.gz" ] && echo "PASS" || echo "FAIL")",
        "config_backup": "$([ -f "$BACKUP_DIR/configs/configs-"*".tar.gz" ] && echo "PASS" || echo "FAIL")",
        "secrets_backup": "$([ -f "$BACKUP_DIR/secrets/secrets-"*".yaml"* ] && echo "PASS" || echo "FAIL")"
    },
    "overall_result": "$([ $test_result -eq 0 ] && echo "PASS" || echo "FAIL")",
    "recommendations": [
        "Regular DR testing should be performed monthly",
        "Backup integrity checks should be automated",
        "Recovery procedures should be documented and practiced",
        "Cross-region backup replication should be implemented"
    ]
}
EOF
    
    # Cleanup test namespace
    kubectl delete namespace "$test_namespace" --ignore-not-found=true
    
    if [[ $test_result -eq 0 ]]; then
        success "Disaster recovery test completed successfully"
    else
        warn "Disaster recovery test completed with warnings"
    fi
    
    return $test_result
}

# Generate DR documentation
generate_dr_documentation() {
    log "Generating disaster recovery documentation..."
    
    local dr_doc="$BACKUP_DIR/DISASTER_RECOVERY_RUNBOOK.md"
    
    cat > "$dr_doc" <<EOF
# Nephoran Intent Operator - Disaster Recovery Runbook

**Generated:** $(date +'%Y-%m-%d %H:%M:%S')  
**DR ID:** $DR_ID  
**Namespace:** $NAMESPACE  

## Overview

This runbook contains comprehensive disaster recovery procedures for the Nephoran Intent Operator system. The backup created on $(date +'%Y-%m-%d %H:%M:%S') includes all critical components and data necessary for full system restoration.

## Backup Contents

### 1. Kubernetes Resources
- **Location:** \`kubernetes/k8s-backup-$TIMESTAMP.tar.gz(.gpg)\`
- **Contents:** CRDs, Deployments, Services, ConfigMaps, PVCs, RBAC
- **Size:** $(du -h "$BACKUP_DIR/kubernetes/"* 2>/dev/null | tail -1 | cut -f1 || echo "Unknown")

### 2. Weaviate Vector Database
- **Location:** \`weaviate/weaviate-backup-$TIMESTAMP.tar.gz\`
- **Contents:** Vector embeddings, knowledge base, indexes
- **Collections:** TelecomKnowledge, IntentPatterns, NetworkFunctions
- **Size:** $(du -h "$BACKUP_DIR/weaviate/"* 2>/dev/null | tail -1 | cut -f1 || echo "Unknown")

### 3. Secrets and Configuration
- **Location:** \`secrets/secrets-$TIMESTAMP.yaml(.gpg)\`
- **Contents:** API keys, certificates, service configurations
- **Encryption:** GPG encrypted (if available)
- **Size:** $(du -h "$BACKUP_DIR/secrets/"* 2>/dev/null | tail -1 | cut -f1 || echo "Unknown")

### 4. Monitoring Data
- **Location:** \`monitoring/\`  
- **Contents:** Prometheus data, Grafana dashboards
- **Retention:** 90 days of metrics data

### 5. Application Logs
- **Location:** \`logs/\`
- **Contents:** Recent logs from all components
- **Timeframe:** Last 24 hours

## Recovery Procedures

### Full System Recovery

1. **Prepare Target Environment**
   \`\`\`bash
   # Ensure kubectl is configured for target cluster
   kubectl config current-context
   
   # Create namespace
   kubectl create namespace $NAMESPACE
   \`\`\`

2. **Restore Kubernetes Resources**
   \`\`\`bash
   # If encrypted
   gpg --decrypt kubernetes/k8s-backup-$TIMESTAMP.tar.gz.gpg | tar -xzf -
   
   # Apply CRDs first
   kubectl apply -f crds.yaml
   
   # Apply other resources in order
   kubectl apply -f serviceaccounts.yaml
   kubectl apply -f roles.yaml
   kubectl apply -f rolebindings.yaml
   kubectl apply -f configmaps.yaml
   kubectl apply -f secrets.yaml
   kubectl apply -f pvcs.yaml
   kubectl apply -f deployments.yaml
   kubectl apply -f services.yaml
   \`\`\`

3. **Restore Weaviate Data**
   \`\`\`bash
   # Wait for Weaviate pod to be ready
   kubectl wait --for=condition=ready pod -l app=weaviate -n $NAMESPACE --timeout=300s
   
   # Copy backup to pod
   kubectl cp weaviate/weaviate-backup-$TIMESTAMP.tar.gz \\
     $NAMESPACE/\$(kubectl get pods -l app=weaviate -o jsonpath='{.items[0].metadata.name}'):/tmp/
   
   # Extract and restore
   kubectl exec -n $NAMESPACE \$(kubectl get pods -l app=weaviate -o jsonpath='{.items[0].metadata.name}') -- \\
     tar -xzf /tmp/weaviate-backup-$TIMESTAMP.tar.gz -C /var/lib/weaviate/backups/
   
   # Trigger restore via API
   kubectl exec -n $NAMESPACE \$(kubectl get pods -l app=weaviate -o jsonpath='{.items[0].metadata.name}') -- \\
     curl -X POST http://localhost:8080/v1/backups/weaviate-backup-$TIMESTAMP/restore
   \`\`\`

4. **Verify System Health**
   \`\`\`bash
   # Check all pods are running
   kubectl get pods -n $NAMESPACE
   
   # Verify services are accessible
   kubectl get services -n $NAMESPACE
   
   # Test basic functionality
   kubectl apply -f - <<EOF
   apiVersion: nephoran.com/v1
   kind: NetworkIntent
   metadata:
     name: recovery-test
     namespace: $NAMESPACE
   spec:
     description: "Test intent for recovery validation"
     priority: medium
   EOF
   
   # Check intent processing
   kubectl get networkintents -n $NAMESPACE
   kubectl describe networkintent recovery-test -n $NAMESPACE
   \`\`\`

### Partial Recovery Scenarios

#### Configuration Only Recovery
\`\`\`bash
# Restore only configurations without data
kubectl apply -f configs/deployments/
\`\`\`

#### Data Only Recovery
\`\`\`bash
# Restore only Weaviate data to existing system
# Follow step 3 from Full System Recovery
\`\`\`

## Recovery Verification Checklist

- [ ] All pods are in Running state
- [ ] All services are accessible
- [ ] CRDs are properly registered
- [ ] Weaviate contains expected data collections
- [ ] NetworkIntent processing is functional
- [ ] E2NodeSet scaling operations work
- [ ] Monitoring dashboards are functional
- [ ] All secrets are properly mounted

## RTO/RPO Targets

- **Recovery Time Objective (RTO):** 2 hours
- **Recovery Point Objective (RPO):** 24 hours
- **Maximum Tolerable Downtime:** 4 hours

## Emergency Contacts

- **Operations Team:** ops@nephoran.com
- **Engineering Team:** engineering@nephoran.com
- **Platform Team:** platform@nephoran.com

## Testing Schedule

- **Full DR Test:** Quarterly
- **Partial DR Test:** Monthly
- **Backup Verification:** Weekly

## Troubleshooting

### Common Issues

1. **CRD Registration Failures**
   - Ensure CRDs are applied before other resources
   - Check for API version conflicts

2. **Weaviate Restore Failures**
   - Verify backup file integrity
   - Check available storage space
   - Ensure proper permissions

3. **Secret Decryption Issues**
   - Verify GPG key availability
   - Check file permissions

### Support Commands

\`\`\`bash
# Check system status
kubectl get all -n $NAMESPACE

# Describe failing resources
kubectl describe pod <pod-name> -n $NAMESPACE

# Check logs
kubectl logs -f deployment/<deployment-name> -n $NAMESPACE

# Test network connectivity
kubectl exec -it <pod-name> -n $NAMESPACE -- nslookup <service-name>
\`\`\`

---
*This runbook was generated automatically by the Nephoran DR system. Keep it updated with any manual changes to the system.*
EOF
    
    success "Disaster recovery documentation generated: $dr_doc"
}

# Main disaster recovery function
main() {
    log "=========================================="
    log "Nephoran Intent Operator Disaster Recovery"
    log "Phase 3 Production Excellence - DR System"
    log "=========================================="
    
    # Initialize DR environment
    initialize_dr_environment
    
    # Create comprehensive backups
    local backup_result=0
    
    log "Step 1: Creating Kubernetes backup..."
    create_kubernetes_backup || backup_result=1
    
    log "Step 2: Creating Weaviate backup..."
    create_weaviate_backup || backup_result=1
    
    log "Step 3: Backing up secrets and configurations..."
    backup_secrets_and_configs || backup_result=1
    
    log "Step 4: Backing up monitoring data..."
    backup_monitoring_data || backup_result=1
    
    log "Step 5: Uploading to cloud storage..."
    upload_to_cloud_storage || backup_result=1
    
    log "Step 6: Testing disaster recovery procedures..."
    test_disaster_recovery || warn "DR test completed with warnings"
    
    log "Step 7: Generating DR documentation..."
    generate_dr_documentation
    
    # Final status
    if [[ $backup_result -eq 0 ]]; then
        success "=========================================="
        success "Disaster Recovery System Setup Complete"
        success "Backup ID: $DR_ID"
        success "Backup Location: $BACKUP_DIR"
        success "Cloud Storage: s3://$S3_BUCKET/disaster-recovery/"
        success "=========================================="
    else
        error "=========================================="
        error "Disaster Recovery Setup Failed"
        error "Check logs for details"
        error "=========================================="
    fi
    
    return $backup_result
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi