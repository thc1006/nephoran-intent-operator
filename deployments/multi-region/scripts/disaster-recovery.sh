#!/bin/bash
set -euo pipefail

# Disaster Recovery Script for Nephoran Intent Operator
# Handles failover, recovery, and backup operations

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
PROJECT_ID="${PROJECT_ID:-}"
REGIONS=("us-central1" "europe-west1" "asia-southeast1")
PRIMARY_REGION="${PRIMARY_REGION:-us-central1}"
BACKUP_BUCKET="${PROJECT_ID}-nephoran-backup"
RECOVERY_MODE="${1:-check}"  # check, failover, recover, backup

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Check region health
check_region_health() {
    local region=$1
    log_info "Checking health of region: $region"
    
    kubectl config use-context "nephoran-${region}" 2>/dev/null || {
        log_error "Cannot connect to cluster in $region"
        return 1
    }
    
    # Check cluster connectivity
    if ! kubectl cluster-info &>/dev/null; then
        log_error "Cluster in $region is not accessible"
        return 1
    fi
    
    # Check critical deployments
    local unhealthy_deployments=$(kubectl get deployments -n nephoran-system \
        -o json | jq -r '.items[] | select(.status.replicas != .status.readyReplicas) | .metadata.name' | wc -l)
    
    if [[ $unhealthy_deployments -gt 0 ]]; then
        log_warning "$unhealthy_deployments unhealthy deployments in $region"
        return 1
    fi
    
    # Check Weaviate health
    local weaviate_ready=$(kubectl get pods -n nephoran-system -l app=weaviate \
        -o json | jq -r '.items[] | select(.status.phase == "Running") | .metadata.name' | wc -l)
    
    if [[ $weaviate_ready -lt 2 ]]; then
        log_warning "Weaviate cluster unhealthy in $region (only $weaviate_ready pods ready)"
        return 1
    fi
    
    log_success "Region $region is healthy"
    return 0
}

# Perform health check on all regions
health_check() {
    log_info "Performing health check on all regions..."
    
    local healthy_regions=()
    local unhealthy_regions=()
    
    for region in "${REGIONS[@]}"; do
        if check_region_health "$region"; then
            healthy_regions+=("$region")
        else
            unhealthy_regions+=("$region")
        fi
    done
    
    echo -e "\n${GREEN}=== Health Check Summary ===${NC}"
    echo -e "${GREEN}Healthy regions:${NC} ${healthy_regions[*]:-None}"
    echo -e "${RED}Unhealthy regions:${NC} ${unhealthy_regions[*]:-None}"
    
    if [[ ${#unhealthy_regions[@]} -gt 0 ]]; then
        log_warning "Action required for unhealthy regions"
        return 1
    else
        log_success "All regions are healthy"
        return 0
    fi
}

# Backup Weaviate data
backup_weaviate() {
    local region=$1
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local backup_name="weaviate-backup-${region}-${timestamp}"
    
    log_info "Creating Weaviate backup for $region..."
    
    kubectl config use-context "nephoran-${region}"
    
    # Trigger Weaviate backup
    kubectl exec -n nephoran-system weaviate-0 -- curl -X POST \
        "http://localhost:8080/v1/backups" \
        -H "Content-Type: application/json" \
        -d "{
            \"id\": \"${backup_name}\",
            \"backend\": \"gcs\",
            \"config\": {
                \"bucket\": \"${BACKUP_BUCKET}\",
                \"path\": \"weaviate/${region}\"
            }
        }"
    
    log_success "Weaviate backup initiated: $backup_name"
}

# Backup all critical data
perform_backup() {
    log_info "Performing backup of all critical data..."
    
    for region in "${REGIONS[@]}"; do
        if check_region_health "$region"; then
            # Backup Weaviate
            backup_weaviate "$region"
            
            # Backup Kubernetes resources
            backup_k8s_resources "$region"
            
            # Backup persistent volumes
            backup_persistent_volumes "$region"
        else
            log_warning "Skipping backup for unhealthy region: $region"
        fi
    done
    
    log_success "Backup completed"
}

# Backup Kubernetes resources
backup_k8s_resources() {
    local region=$1
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    local backup_file="k8s-resources-${region}-${timestamp}.yaml"
    
    log_info "Backing up Kubernetes resources for $region..."
    
    kubectl config use-context "nephoran-${region}"
    
    # Export all resources
    kubectl get all,cm,secret,pvc,pv,ing,netpol,sa,role,rolebinding \
        -n nephoran-system \
        -o yaml > "/tmp/${backup_file}"
    
    # Upload to GCS
    gsutil cp "/tmp/${backup_file}" "gs://${BACKUP_BUCKET}/k8s-resources/${region}/"
    
    rm -f "/tmp/${backup_file}"
    
    log_success "Kubernetes resources backed up"
}

# Backup persistent volumes
backup_persistent_volumes() {
    local region=$1
    local timestamp=$(date '+%Y%m%d_%H%M%S')
    
    log_info "Backing up persistent volumes for $region..."
    
    kubectl config use-context "nephoran-${region}"
    
    # Get all PVCs
    local pvcs=$(kubectl get pvc -n nephoran-system -o json | jq -r '.items[].metadata.name')
    
    for pvc in $pvcs; do
        local pv=$(kubectl get pvc "$pvc" -n nephoran-system -o jsonpath='{.spec.volumeName}')
        local disk=$(kubectl get pv "$pv" -o jsonpath='{.spec.gcePersistentDisk.pdName}')
        
        if [[ -n "$disk" ]]; then
            log_info "Creating snapshot of disk: $disk"
            gcloud compute disks snapshot "$disk" \
                --snapshot-names="${disk}-${timestamp}" \
                --zone="${region}-a" \
                --project="$PROJECT_ID"
        fi
    done
    
    log_success "Persistent volumes backed up"
}

# Perform failover
perform_failover() {
    local failed_region=$1
    local target_region=$2
    
    log_warning "Initiating failover from $failed_region to $target_region"
    
    # Update DNS records
    update_dns_records "$failed_region" "$target_region"
    
    # Promote Weaviate replica
    promote_weaviate_replica "$target_region"
    
    # Update service endpoints
    update_service_endpoints "$failed_region" "$target_region"
    
    # Scale up services in target region
    scale_services "$target_region" "up"
    
    # Update monitoring alerts
    update_monitoring_alerts "$failed_region" "$target_region"
    
    log_success "Failover completed to $target_region"
}

# Update DNS records
update_dns_records() {
    local failed_region=$1
    local target_region=$2
    
    log_info "Updating DNS records..."
    
    # Get target region IP
    kubectl config use-context "nephoran-${target_region}"
    local target_ip=$(kubectl get svc -n nephoran-system multi-region-gateway \
        -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    
    # Update Cloud DNS
    gcloud dns record-sets update "api.nephoran.${PROJECT_ID}.com." \
        --type=A \
        --ttl=60 \
        --rrdatas="$target_ip" \
        --zone="nephoran-zone" \
        --project="$PROJECT_ID"
    
    log_success "DNS records updated"
}

# Promote Weaviate replica
promote_weaviate_replica() {
    local region=$1
    
    log_info "Promoting Weaviate replica in $region to master..."
    
    kubectl config use-context "nephoran-${region}"
    
    # Update Weaviate configuration
    kubectl patch configmap weaviate-config -n nephoran-system \
        --type merge \
        -p '{"data":{"WEAVIATE_ROLE":"master"}}'
    
    # Restart Weaviate pods
    kubectl rollout restart statefulset weaviate -n nephoran-system
    
    # Wait for rollout
    kubectl rollout status statefulset weaviate -n nephoran-system --timeout=300s
    
    log_success "Weaviate replica promoted"
}

# Update service endpoints
update_service_endpoints() {
    local failed_region=$1
    local target_region=$2
    
    log_info "Updating service endpoints..."
    
    kubectl config use-context "nephoran-${target_region}"
    
    # Update region endpoints ConfigMap
    kubectl patch configmap region-endpoints -n nephoran-system \
        --type json \
        -p "[{\"op\": \"replace\", \"path\": \"/data/primary_region\", \"value\": \"$target_region\"}]"
    
    # Restart dependent services
    for deployment in llm-processor rag-api intent-controller edge-controller; do
        kubectl rollout restart deployment "$deployment" -n nephoran-system
    done
    
    log_success "Service endpoints updated"
}

# Scale services
scale_services() {
    local region=$1
    local direction=$2  # up or down
    
    log_info "Scaling services $direction in $region..."
    
    kubectl config use-context "nephoran-${region}"
    
    if [[ "$direction" == "up" ]]; then
        # Scale up for increased load
        kubectl scale deployment llm-processor -n nephoran-system --replicas=10
        kubectl scale deployment rag-api -n nephoran-system --replicas=8
        kubectl scale deployment intent-controller -n nephoran-system --replicas=5
        kubectl scale deployment edge-controller -n nephoran-system --replicas=5
    else
        # Scale down to normal levels
        kubectl scale deployment llm-processor -n nephoran-system --replicas=3
        kubectl scale deployment rag-api -n nephoran-system --replicas=3
        kubectl scale deployment intent-controller -n nephoran-system --replicas=2
        kubectl scale deployment edge-controller -n nephoran-system --replicas=2
    fi
    
    log_success "Services scaled $direction"
}

# Update monitoring alerts
update_monitoring_alerts() {
    local failed_region=$1
    local target_region=$2
    
    log_info "Updating monitoring alerts..."
    
    # Send alert about failover
    curl -X POST "${SLACK_WEBHOOK_URL}" \
        -H 'Content-Type: application/json' \
        -d "{
            \"text\": \"🚨 Disaster Recovery Alert\",
            \"attachments\": [{
                \"color\": \"warning\",
                \"title\": \"Regional Failover Executed\",
                \"fields\": [
                    {\"title\": \"Failed Region\", \"value\": \"$failed_region\", \"short\": true},
                    {\"title\": \"Target Region\", \"value\": \"$target_region\", \"short\": true},
                    {\"title\": \"Timestamp\", \"value\": \"$(date)\", \"short\": false}
                ]
            }]
        }"
    
    log_success "Monitoring alerts updated"
}

# Perform recovery
perform_recovery() {
    local region=$1
    
    log_info "Performing recovery for region: $region"
    
    # Check if region is now healthy
    if ! check_region_health "$region"; then
        log_error "Region $region is still unhealthy. Cannot proceed with recovery."
        return 1
    fi
    
    # Restore from latest backup
    restore_from_backup "$region"
    
    # Resync data
    resync_data "$region"
    
    # Rebalance traffic
    rebalance_traffic "$region"
    
    log_success "Recovery completed for $region"
}

# Restore from backup
restore_from_backup() {
    local region=$1
    
    log_info "Restoring from backup for $region..."
    
    # Find latest backup
    local latest_backup=$(gsutil ls "gs://${BACKUP_BUCKET}/weaviate/${region}/" | \
        grep "weaviate-backup-" | sort -r | head -1)
    
    if [[ -z "$latest_backup" ]]; then
        log_warning "No backup found for $region"
        return 1
    fi
    
    log_info "Restoring from backup: $latest_backup"
    
    kubectl config use-context "nephoran-${region}"
    
    # Trigger Weaviate restore
    kubectl exec -n nephoran-system weaviate-0 -- curl -X POST \
        "http://localhost:8080/v1/backups/restore" \
        -H "Content-Type: application/json" \
        -d "{
            \"id\": \"$(basename "$latest_backup")\",
            \"backend\": \"gcs\",
            \"config\": {
                \"bucket\": \"${BACKUP_BUCKET}\",
                \"path\": \"weaviate/${region}\"
            }
        }"
    
    log_success "Restore initiated"
}

# Resync data between regions
resync_data() {
    local region=$1
    
    log_info "Resyncing data for $region..."
    
    # Get primary region data
    local primary_endpoint=$(kubectl get svc weaviate-lb -n nephoran-system \
        --context="nephoran-${PRIMARY_REGION}" \
        -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    
    # Trigger data sync
    kubectl exec -n nephoran-system weaviate-0 --context="nephoran-${region}" -- \
        curl -X POST "http://localhost:8080/v1/replication/sync" \
        -H "Content-Type: application/json" \
        -d "{
            \"source\": \"http://${primary_endpoint}:8080\",
            \"mode\": \"full\"
        }"
    
    log_success "Data resync initiated"
}

# Rebalance traffic
rebalance_traffic() {
    local region=$1
    
    log_info "Rebalancing traffic to include $region..."
    
    # Update backend service weights
    gcloud compute backend-services update nephoran-backend-service \
        --global \
        --project="$PROJECT_ID" \
        --update-backend="instance-group=${region}-ig,balancing-mode=UTILIZATION,max-utilization=0.8"
    
    log_success "Traffic rebalanced"
}

# Main function
main() {
    case "$RECOVERY_MODE" in
        check)
            health_check
            ;;
        backup)
            perform_backup
            ;;
        failover)
            if [[ $# -lt 3 ]]; then
                log_error "Usage: $0 failover <failed_region> <target_region>"
                exit 1
            fi
            perform_failover "$2" "$3"
            ;;
        recover)
            if [[ $# -lt 2 ]]; then
                log_error "Usage: $0 recover <region>"
                exit 1
            fi
            perform_recovery "$2"
            ;;
        *)
            log_error "Invalid mode: $RECOVERY_MODE"
            echo "Usage: $0 {check|backup|failover|recover}"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"