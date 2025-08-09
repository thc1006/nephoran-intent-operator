# Disaster Recovery Runbook - Nephoran Intent Operator

**Version:** 3.0  
**Last Updated:** January 2025  
**Audience:** Site Reliability Engineers, Operations Teams, Incident Commanders  
**Classification:** Disaster Recovery Documentation

## Table of Contents

1. [DR Strategy Overview](#dr-strategy-overview)
2. [RTO/RPO Requirements](#rtorpo-requirements)
3. [Backup Procedures](#backup-procedures)
4. [Recovery Procedures](#recovery-procedures)
5. [Failover Procedures](#failover-procedures)
6. [Data Recovery](#data-recovery)
7. [Service Restoration](#service-restoration)
8. [DR Testing](#dr-testing)
9. [Post-Recovery Validation](#post-recovery-validation)

## DR Strategy Overview

### Architecture

```yaml
DR Architecture:
  Primary Region: us-east-1
    - Production cluster
    - Active workloads
    - Primary databases
    - Real-time replication
    
  Secondary Region: us-west-2
    - Standby cluster
    - Warm standby mode
    - Replicated databases
    - Ready for failover
    
  Backup Region: eu-west-1
    - Cold storage
    - Long-term backups
    - Compliance archives
    - Disaster archives
    
  Replication Strategy:
    - Database: Streaming replication (PostgreSQL)
    - Object Storage: Cross-region replication (S3)
    - Vector Database: Snapshot replication (Weaviate)
    - Configuration: GitOps synchronization
```

### DR Scenarios

| Scenario | Impact | RTO | RPO | Response |
|----------|--------|-----|-----|----------|
| Node Failure | Single node down | 5 min | 0 | Automatic pod rescheduling |
| Zone Failure | Availability zone down | 15 min | 30 sec | Multi-zone failover |
| Cluster Failure | Complete cluster failure | 30 min | 5 min | Cross-cluster failover |
| Region Failure | Entire region down | 45 min | 15 min | Cross-region failover |
| Data Corruption | Data integrity issues | 4 hours | 1 hour | Restore from backups |
| Ransomware | Malicious encryption | 8 hours | 24 hours | Clean room recovery |
| Complete Loss | Total infrastructure loss | 24 hours | 24 hours | Full rebuild from backups |

## RTO/RPO Requirements

### Service Level Objectives

```yaml
Critical Services (Tier 1):
  Services: [llm-processor, nephoran-controller, rag-api]
  RTO: 15 minutes
  RPO: 5 minutes
  Availability: 99.99%
  
Important Services (Tier 2):
  Services: [weaviate, monitoring, oauth-proxy]
  RTO: 30 minutes
  RPO: 15 minutes
  Availability: 99.9%
  
Standard Services (Tier 3):
  Services: [grafana, jaeger, documentation]
  RTO: 4 hours
  RPO: 1 hour
  Availability: 99.5%
  
Development Services (Tier 4):
  Services: [dev-environments, test-clusters]
  RTO: 24 hours
  RPO: 24 hours
  Availability: 95%
```

### Data Classification

```yaml
Data Categories:
  Critical Data:
    - NetworkIntent CRDs
    - Production configurations
    - Authentication tokens
    - Encryption keys
    Backup: Every 5 minutes
    Retention: 90 days
    
  Important Data:
    - Vector embeddings
    - Knowledge base
    - Metrics data
    - Audit logs
    Backup: Every hour
    Retention: 30 days
    
  Standard Data:
    - Application logs
    - Temporary caches
    - Development data
    Backup: Daily
    Retention: 7 days
```

## Backup Procedures

### Automated Backup System

```yaml
# backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nephoran-backup
  namespace: backup-system
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes for critical data
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: backup-operator
          containers:
          - name: backup
            image: nephoran/backup-operator:latest
            command:
            - /bin/sh
            - -c
            - |
              # Backup script
              echo "Starting backup at $(date)"
              
              # 1. Backup Kubernetes resources
              kubectl get all,cm,secret,pvc,networkintents,e2nodesets \
                -n nephoran-system -o yaml > /backup/k8s-resources-$(date +%Y%m%d-%H%M%S).yaml
              
              # 2. Backup PostgreSQL
              PGPASSWORD=$DB_PASSWORD pg_dump \
                -h postgresql.nephoran-system \
                -U nephoran \
                -d nephoran_db \
                > /backup/postgresql-$(date +%Y%m%d-%H%M%S).sql
              
              # 3. Backup Weaviate
              curl -X POST http://weaviate.nephoran-system:8080/v1/backups \
                -H "Content-Type: application/json" \
                -d '{
                  "id": "backup-'$(date +%Y%m%d-%H%M%S)'",
                  "backend": "s3",
                  "include": ["TelecomKnowledge"]
                }'
              
              # 4. Backup Redis
              redis-cli -h redis.nephoran-system BGSAVE
              cp /data/dump.rdb /backup/redis-$(date +%Y%m%d-%H%M%S).rdb
              
              # 5. Upload to S3
              aws s3 sync /backup/ s3://nephoran-backups/$(date +%Y%m%d)/ \
                --storage-class GLACIER_IR
              
              # 6. Verify backup
              aws s3 ls s3://nephoran-backups/$(date +%Y%m%d)/ --recursive
              
              echo "Backup completed at $(date)"
          restartPolicy: OnFailure
```

### Manual Backup Procedure

```bash
#!/bin/bash
# Manual backup script for DR

perform_manual_backup() {
  echo "=== Manual DR Backup ==="
  echo "Starting at $(date)"
  
  BACKUP_ID="manual-$(date +%Y%m%d-%H%M%S)"
  BACKUP_DIR="/tmp/backup-$BACKUP_ID"
  mkdir -p $BACKUP_DIR
  
  # 1. Kubernetes resources
  echo "--- Backing up Kubernetes resources ---"
  for namespace in nephoran-system monitoring istio-system; do
    kubectl get all,cm,secret,pvc,ing,cert -n $namespace -o yaml > \
      $BACKUP_DIR/k8s-$namespace.yaml
  done
  
  # Custom resources
  kubectl get networkintents,e2nodesets --all-namespaces -o yaml > \
    $BACKUP_DIR/k8s-custom-resources.yaml
  
  # 2. Database backups
  echo "--- Backing up databases ---"
  
  # PostgreSQL
  kubectl exec -n nephoran-system postgresql-0 -- \
    pg_dumpall -U postgres > $BACKUP_DIR/postgresql-all.sql
  
  # Weaviate
  kubectl exec -n nephoran-system weaviate-0 -- \
    curl -X POST localhost:8080/v1/backups \
    -d "{\"id\": \"$BACKUP_ID\", \"backend\": \"filesystem\"}"
  
  # 3. Configuration files
  echo "--- Backing up configurations ---"
  kubectl get configmaps -n nephoran-system -o yaml > \
    $BACKUP_DIR/configmaps.yaml
  kubectl get secrets -n nephoran-system -o yaml > \
    $BACKUP_DIR/secrets.yaml
  
  # 4. Persistent volumes
  echo "--- Backing up persistent volumes ---"
  for pvc in $(kubectl get pvc -n nephoran-system -o name); do
    PVC_NAME=$(echo $pvc | cut -d'/' -f2)
    POD=$(kubectl get pods -n nephoran-system -o json | \
      jq -r ".items[] | select(.spec.volumes[].persistentVolumeClaim.claimName==\"$PVC_NAME\") | .metadata.name" | head -1)
    
    if [ ! -z "$POD" ]; then
      kubectl exec -n nephoran-system $POD -- tar czf - /data 2>/dev/null > \
        $BACKUP_DIR/pv-$PVC_NAME.tar.gz
    fi
  done
  
  # 5. Create backup manifest
  cat > $BACKUP_DIR/manifest.json <<EOF
{
  "backup_id": "$BACKUP_ID",
  "timestamp": "$(date -Iseconds)",
  "type": "manual",
  "components": [
    "kubernetes",
    "postgresql",
    "weaviate",
    "redis",
    "configurations",
    "persistent_volumes"
  ],
  "cluster": "$(kubectl config current-context)",
  "version": "$(kubectl version -o json | jq -r .serverVersion.gitVersion)"
}
EOF
  
  # 6. Compress and encrypt
  echo "--- Compressing and encrypting backup ---"
  tar czf $BACKUP_DIR.tar.gz -C /tmp backup-$BACKUP_ID
  openssl enc -aes-256-cbc -salt -in $BACKUP_DIR.tar.gz \
    -out $BACKUP_DIR.tar.gz.enc -k "$BACKUP_ENCRYPTION_KEY"
  
  # 7. Upload to multiple locations
  echo "--- Uploading backup ---"
  
  # Primary backup location
  aws s3 cp $BACKUP_DIR.tar.gz.enc \
    s3://nephoran-backups-primary/$BACKUP_ID.tar.gz.enc \
    --storage-class STANDARD_IA
  
  # Secondary backup location
  aws s3 cp $BACKUP_DIR.tar.gz.enc \
    s3://nephoran-backups-secondary/$BACKUP_ID.tar.gz.enc \
    --storage-class GLACIER_IR \
    --region us-west-2
  
  # Cleanup
  rm -rf $BACKUP_DIR $BACKUP_DIR.tar.gz $BACKUP_DIR.tar.gz.enc
  
  echo "Backup completed: $BACKUP_ID"
  echo "Primary: s3://nephoran-backups-primary/$BACKUP_ID.tar.gz.enc"
  echo "Secondary: s3://nephoran-backups-secondary/$BACKUP_ID.tar.gz.enc"
}
```

### Backup Validation

```bash
#!/bin/bash
# Backup validation script

validate_backup() {
  local BACKUP_ID=$1
  
  echo "=== Backup Validation: $BACKUP_ID ==="
  
  # Download backup
  echo "--- Downloading backup ---"
  aws s3 cp s3://nephoran-backups-primary/$BACKUP_ID.tar.gz.enc /tmp/
  
  # Decrypt
  echo "--- Decrypting backup ---"
  openssl enc -aes-256-cbc -d -in /tmp/$BACKUP_ID.tar.gz.enc \
    -out /tmp/$BACKUP_ID.tar.gz -k "$BACKUP_ENCRYPTION_KEY"
  
  # Extract
  echo "--- Extracting backup ---"
  tar xzf /tmp/$BACKUP_ID.tar.gz -C /tmp/
  
  # Validate components
  echo "--- Validating components ---"
  
  # Check manifest
  if [ -f /tmp/backup-$BACKUP_ID/manifest.json ]; then
    echo "✓ Manifest present"
    jq '.' /tmp/backup-$BACKUP_ID/manifest.json
  else
    echo "✗ Manifest missing"
    return 1
  fi
  
  # Check Kubernetes resources
  if [ -f /tmp/backup-$BACKUP_ID/k8s-nephoran-system.yaml ]; then
    echo "✓ Kubernetes resources present"
    kubectl apply --dry-run=client -f /tmp/backup-$BACKUP_ID/k8s-nephoran-system.yaml > /dev/null 2>&1 && \
      echo "✓ Kubernetes resources valid" || echo "✗ Kubernetes resources invalid"
  else
    echo "✗ Kubernetes resources missing"
  fi
  
  # Check database backup
  if [ -f /tmp/backup-$BACKUP_ID/postgresql-all.sql ]; then
    echo "✓ Database backup present"
    head -n 10 /tmp/backup-$BACKUP_ID/postgresql-all.sql | grep -q "PostgreSQL" && \
      echo "✓ Database backup valid" || echo "✗ Database backup invalid"
  else
    echo "✗ Database backup missing"
  fi
  
  # Cleanup
  rm -rf /tmp/$BACKUP_ID.tar.gz.enc /tmp/$BACKUP_ID.tar.gz /tmp/backup-$BACKUP_ID
  
  echo "Validation completed"
}
```

## Recovery Procedures

### Full System Recovery

```bash
#!/bin/bash
# Full system recovery procedure

perform_full_recovery() {
  local BACKUP_ID=$1
  local TARGET_CLUSTER=$2
  
  echo "=== FULL SYSTEM RECOVERY ==="
  echo "Backup ID: $BACKUP_ID"
  echo "Target Cluster: $TARGET_CLUSTER"
  echo "Started at $(date)"
  
  # Switch to target cluster
  kubectl config use-context $TARGET_CLUSTER
  
  # 1. Prepare cluster
  echo "--- Preparing cluster ---"
  
  # Create namespaces
  kubectl create namespace nephoran-system 2>/dev/null
  kubectl create namespace monitoring 2>/dev/null
  kubectl create namespace istio-system 2>/dev/null
  
  # 2. Download and extract backup
  echo "--- Downloading backup ---"
  aws s3 cp s3://nephoran-backups-primary/$BACKUP_ID.tar.gz.enc /tmp/
  openssl enc -aes-256-cbc -d -in /tmp/$BACKUP_ID.tar.gz.enc \
    -out /tmp/$BACKUP_ID.tar.gz -k "$BACKUP_ENCRYPTION_KEY"
  tar xzf /tmp/$BACKUP_ID.tar.gz -C /tmp/
  
  BACKUP_DIR="/tmp/backup-$BACKUP_ID"
  
  # 3. Restore configurations
  echo "--- Restoring configurations ---"
  kubectl apply -f $BACKUP_DIR/configmaps.yaml
  kubectl apply -f $BACKUP_DIR/secrets.yaml
  
  # 4. Restore CRDs
  echo "--- Restoring CRDs ---"
  kubectl apply -f deployments/crds/
  
  # 5. Restore infrastructure components
  echo "--- Restoring infrastructure ---"
  
  # Cert-manager
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
  kubectl wait --for=condition=available deployment/cert-manager -n cert-manager --timeout=300s
  
  # Istio
  istioctl install --set profile=production -y
  kubectl wait --for=condition=available deployment/istiod -n istio-system --timeout=300s
  
  # 6. Restore databases
  echo "--- Restoring databases ---"
  
  # Deploy PostgreSQL
  kubectl apply -f deployments/postgresql/
  kubectl wait --for=condition=ready pod/postgresql-0 -n nephoran-system --timeout=300s
  
  # Restore PostgreSQL data
  kubectl exec -n nephoran-system postgresql-0 -- \
    psql -U postgres < $BACKUP_DIR/postgresql-all.sql
  
  # Deploy Weaviate
  kubectl apply -f deployments/weaviate/
  kubectl wait --for=condition=ready pod/weaviate-0 -n nephoran-system --timeout=300s
  
  # 7. Restore applications
  echo "--- Restoring applications ---"
  kubectl apply -f $BACKUP_DIR/k8s-nephoran-system.yaml
  
  # 8. Restore persistent volumes
  echo "--- Restoring persistent volumes ---"
  for pv_backup in $BACKUP_DIR/pv-*.tar.gz; do
    if [ -f "$pv_backup" ]; then
      PVC_NAME=$(basename $pv_backup | sed 's/pv-//;s/.tar.gz//')
      echo "Restoring PVC: $PVC_NAME"
      
      # Wait for pod with this PVC
      sleep 30
      POD=$(kubectl get pods -n nephoran-system -o json | \
        jq -r ".items[] | select(.spec.volumes[].persistentVolumeClaim.claimName==\"$PVC_NAME\") | .metadata.name" | head -1)
      
      if [ ! -z "$POD" ]; then
        kubectl exec -n nephoran-system $POD -- tar xzf - -C / < $pv_backup
      fi
    fi
  done
  
  # 9. Restore custom resources
  echo "--- Restoring custom resources ---"
  kubectl apply -f $BACKUP_DIR/k8s-custom-resources.yaml
  
  # 10. Validate restoration
  echo "--- Validating restoration ---"
  sleep 60
  
  # Check pod status
  kubectl get pods -n nephoran-system
  UNHEALTHY=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
  
  if [ $UNHEALTHY -eq 0 ]; then
    echo "✓ All pods healthy"
  else
    echo "✗ $UNHEALTHY pods unhealthy"
    kubectl get pods -n nephoran-system --no-headers | grep -v Running
  fi
  
  # Check services
  for service in llm-processor rag-api nephoran-controller weaviate; do
    kubectl get endpoints $service -n nephoran-system > /dev/null 2>&1 && \
      echo "✓ Service $service ready" || echo "✗ Service $service not ready"
  done
  
  # Cleanup
  rm -rf /tmp/$BACKUP_ID.tar.gz.enc /tmp/$BACKUP_ID.tar.gz $BACKUP_DIR
  
  echo "Recovery completed at $(date)"
}
```

### Partial Recovery

```bash
#!/bin/bash
# Partial service recovery

recover_service() {
  local SERVICE=$1
  local BACKUP_ID=$2
  
  echo "=== Partial Recovery: $SERVICE ==="
  
  case $SERVICE in
    "database")
      echo "--- Recovering database ---"
      
      # Scale down dependent services
      kubectl scale deployment llm-processor nephoran-controller --replicas=0 -n nephoran-system
      
      # Restore database
      kubectl exec -n nephoran-system postgresql-0 -- \
        psql -U postgres -c "DROP DATABASE IF EXISTS nephoran_db;"
      kubectl exec -n nephoran-system postgresql-0 -- \
        psql -U postgres < /backup/postgresql-$BACKUP_ID.sql
      
      # Scale up services
      kubectl scale deployment llm-processor --replicas=3 -n nephoran-system
      kubectl scale deployment nephoran-controller --replicas=2 -n nephoran-system
      ;;
      
    "weaviate")
      echo "--- Recovering Weaviate ---"
      
      # Delete existing data
      kubectl delete statefulset weaviate -n nephoran-system
      kubectl delete pvc -l app=weaviate -n nephoran-system
      
      # Redeploy
      kubectl apply -f deployments/weaviate/
      kubectl wait --for=condition=ready pod/weaviate-0 -n nephoran-system --timeout=300s
      
      # Restore from backup
      curl -X POST http://weaviate.nephoran-system:8080/v1/backups/$BACKUP_ID/restore \
        -H "Content-Type: application/json" \
        -d '{"include": ["TelecomKnowledge"]}'
      ;;
      
    "controller")
      echo "--- Recovering controller ---"
      
      # Backup current state
      kubectl get networkintents --all-namespaces -o yaml > /tmp/intents-current.yaml
      
      # Redeploy controller
      kubectl delete deployment nephoran-controller -n nephoran-system
      kubectl apply -f deployments/controller/
      
      # Restore intents
      kubectl apply -f /tmp/intents-current.yaml
      ;;
  esac
  
  echo "Service $SERVICE recovered"
}
```

## Failover Procedures

### Regional Failover

```bash
#!/bin/bash
# Regional failover procedure

perform_regional_failover() {
  local SOURCE_REGION=$1
  local TARGET_REGION=$2
  
  echo "=== REGIONAL FAILOVER ==="
  echo "Source: $SOURCE_REGION → Target: $TARGET_REGION"
  echo "Started at $(date)"
  
  # 1. Pre-failover checks
  echo "--- Pre-failover checks ---"
  
  # Check target region health
  kubectl --context $TARGET_REGION get nodes > /dev/null 2>&1 || {
    echo "✗ Target region not accessible"
    exit 1
  }
  
  # Check replication lag
  REPLICATION_LAG=$(kubectl --context $TARGET_REGION exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "SELECT extract(epoch from now() - pg_last_xact_replay_timestamp());" -t)
  echo "Replication lag: ${REPLICATION_LAG}s"
  
  if (( $(echo "$REPLICATION_LAG > 300" | bc -l) )); then
    echo "⚠️  High replication lag detected"
    read -p "Continue with failover? (y/N) " -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 1
  fi
  
  # 2. Stop traffic to source region
  echo "--- Stopping traffic to source region ---"
  
  # Update DNS to maintenance page
  ./scripts/update-dns.sh --domain api.nephoran.com --target maintenance.nephoran.com
  
  # Wait for connections to drain
  echo "Waiting 30s for connection draining..."
  sleep 30
  
  # 3. Final data sync
  echo "--- Final data synchronization ---"
  
  # Force PostgreSQL checkpoint
  kubectl --context $SOURCE_REGION exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "CHECKPOINT;" 2>/dev/null || echo "Source DB unreachable"
  
  # Wait for replication to catch up
  echo "Waiting for replication to complete..."
  sleep 10
  
  # 4. Promote standby to primary
  echo "--- Promoting standby database ---"
  kubectl --context $TARGET_REGION exec -n nephoran-system postgresql-0 -- \
    pg_ctl promote -D /var/lib/postgresql/data
  
  # 5. Start services in target region
  echo "--- Starting services in target region ---"
  kubectl --context $TARGET_REGION scale deployment llm-processor --replicas=5 -n nephoran-system
  kubectl --context $TARGET_REGION scale deployment rag-api --replicas=3 -n nephoran-system
  kubectl --context $TARGET_REGION scale deployment nephoran-controller --replicas=3 -n nephoran-system
  
  # Wait for services to be ready
  kubectl --context $TARGET_REGION wait --for=condition=available \
    deployment/llm-processor deployment/rag-api deployment/nephoran-controller \
    -n nephoran-system --timeout=300s
  
  # 6. Update DNS to target region
  echo "--- Updating DNS ---"
  ./scripts/update-dns.sh --domain api.nephoran.com --target api-$TARGET_REGION.nephoran.com
  
  # 7. Validate failover
  echo "--- Validating failover ---"
  
  # Test API endpoint
  curl -s https://api.nephoran.com/health | jq '.status' | grep -q "healthy" && \
    echo "✓ API healthy" || echo "✗ API unhealthy"
  
  # Test intent processing
  kubectl --context $TARGET_REGION apply -f - <<EOF
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: failover-test-$(date +%s)
  namespace: default
spec:
  description: "Failover validation test"
  priority: high
EOF
  
  # 8. Update monitoring
  echo "--- Updating monitoring ---"
  kubectl --context $TARGET_REGION patch configmap prometheus-config -n monitoring \
    --type merge -p '{"data":{"region":"'$TARGET_REGION'"}}'
  
  echo "Regional failover completed at $(date)"
  echo "New primary region: $TARGET_REGION"
}
```

### Cluster Failover

```bash
#!/bin/bash
# Cluster failover within same region

perform_cluster_failover() {
  local SOURCE_CLUSTER=$1
  local TARGET_CLUSTER=$2
  
  echo "=== CLUSTER FAILOVER ==="
  echo "Source: $SOURCE_CLUSTER → Target: $TARGET_CLUSTER"
  
  # 1. Redirect traffic
  echo "--- Redirecting traffic ---"
  kubectl --context $SOURCE_CLUSTER patch service nephoran-api -n nephoran-system \
    -p '{"spec":{"selector":{"cluster":"maintenance"}}}'
  
  # 2. Sync final state
  echo "--- Syncing state ---"
  kubectl --context $SOURCE_CLUSTER get networkintents --all-namespaces -o yaml | \
    kubectl --context $TARGET_CLUSTER apply -f -
  
  # 3. Switch load balancer
  echo "--- Switching load balancer ---"
  aws elbv2 modify-target-group \
    --target-group-arn $TARGET_GROUP_ARN \
    --targets Id=$TARGET_CLUSTER_IP
  
  # 4. Validate
  echo "--- Validating failover ---"
  curl -s https://api.nephoran.com/health
  
  echo "Cluster failover completed"
}
```

## Data Recovery

### Point-in-Time Recovery

```bash
#!/bin/bash
# Point-in-time recovery procedure

perform_pitr() {
  local TARGET_TIME=$1
  
  echo "=== Point-in-Time Recovery ==="
  echo "Target time: $TARGET_TIME"
  
  # 1. Find appropriate backup
  echo "--- Finding backup ---"
  BACKUP_TIME=$(date -d "$TARGET_TIME" +%Y%m%d-%H%M%S)
  BACKUP_ID=$(aws s3 ls s3://nephoran-backups-primary/ | \
    grep -E "backup-.*$BACKUP_TIME" | \
    awk '{print $4}' | head -1)
  
  if [ -z "$BACKUP_ID" ]; then
    echo "No backup found for $TARGET_TIME"
    exit 1
  fi
  
  echo "Using backup: $BACKUP_ID"
  
  # 2. Restore base backup
  echo "--- Restoring base backup ---"
  perform_full_recovery $BACKUP_ID production
  
  # 3. Apply WAL logs to target time
  echo "--- Applying WAL logs ---"
  kubectl exec -n nephoran-system postgresql-0 -- \
    pg_wal_replay --target-time="$TARGET_TIME" --target-action=promote
  
  # 4. Validate recovery
  echo "--- Validating recovery ---"
  kubectl exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "SELECT max(created_at) FROM audit_log;"
  
  echo "Point-in-time recovery completed"
}
```

### Data Consistency Verification

```bash
#!/bin/bash
# Verify data consistency after recovery

verify_data_consistency() {
  echo "=== Data Consistency Verification ==="
  
  # 1. Database consistency
  echo "--- Database consistency ---"
  kubectl exec -n nephoran-system postgresql-0 -- \
    psql -U postgres -c "SELECT schemaname, tablename, 
      pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
      n_live_tup AS rows
      FROM pg_stat_user_tables 
      ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"
  
  # 2. Vector database consistency
  echo "--- Vector database consistency ---"
  curl -s http://weaviate.nephoran-system:8080/v1/schema | \
    jq '.classes[] | {class: .class, vectorizer: .vectorizer, propertyCount: .properties | length}'
  
  # 3. CRD consistency
  echo "--- CRD consistency ---"
  kubectl get networkintents --all-namespaces | wc -l
  kubectl get e2nodesets --all-namespaces | wc -l
  
  # 4. Configuration consistency
  echo "--- Configuration consistency ---"
  kubectl get configmaps -n nephoran-system -o name | wc -l
  kubectl get secrets -n nephoran-system -o name | wc -l
  
  # 5. Generate consistency report
  cat > /tmp/consistency-report.md <<EOF
# Data Consistency Report
Date: $(date)

## Database Status
- Tables: $(kubectl exec -n nephoran-system postgresql-0 -- psql -U postgres -c "\dt" -t | wc -l)
- Total Size: $(kubectl exec -n nephoran-system postgresql-0 -- psql -U postgres -c "SELECT pg_database_size('nephoran_db');" -t)

## Vector Database Status
- Collections: $(curl -s http://weaviate.nephoran-system:8080/v1/schema | jq '.classes | length')
- Documents: $(curl -s http://weaviate.nephoran-system:8080/v1/objects | jq '.totalResults')

## Kubernetes Resources
- NetworkIntents: $(kubectl get networkintents --all-namespaces --no-headers | wc -l)
- Pods Running: $(kubectl get pods -n nephoran-system --no-headers | grep Running | wc -l)

## Validation Results
- Database: $(kubectl exec -n nephoran-system postgresql-0 -- pg_isready > /dev/null 2>&1 && echo "✓ Healthy" || echo "✗ Unhealthy")
- Weaviate: $(curl -s http://weaviate.nephoran-system:8080/v1/.well-known/ready | grep -q true && echo "✓ Ready" || echo "✗ Not Ready")
- Services: $(kubectl get endpoints -n nephoran-system | grep -v "^NAME" | wc -l) active endpoints

EOF
  
  echo "Report saved to /tmp/consistency-report.md"
}
```

## Service Restoration

### Service Restoration Order

```yaml
Restoration Sequence:
  Phase 1 - Infrastructure (0-15 min):
    1. Kubernetes cluster
    2. Networking (CNI, DNS)
    3. Storage (CSI, PV)
    4. Certificate management
    
  Phase 2 - Data Layer (15-30 min):
    1. PostgreSQL database
    2. Redis cache
    3. Weaviate vector DB
    4. Persistent volumes
    
  Phase 3 - Core Services (30-45 min):
    1. Nephoran controller
    2. LLM processor
    3. RAG API
    4. O-RAN adaptors
    
  Phase 4 - Supporting Services (45-60 min):
    1. Monitoring stack
    2. Service mesh
    3. Ingress controllers
    4. OAuth proxy
    
  Phase 5 - Validation (60-75 min):
    1. Health checks
    2. Integration tests
    3. Performance validation
    4. Security scanning
```

### Restoration Runbook

```bash
#!/bin/bash
# Service restoration procedure

restore_services() {
  echo "=== Service Restoration Procedure ==="
  echo "Started at $(date)"
  
  # Phase 1: Infrastructure
  echo "--- Phase 1: Infrastructure ---"
  
  # Verify cluster
  kubectl cluster-info
  kubectl get nodes
  
  # Verify networking
  kubectl get pods -n kube-system | grep -E "coredns|calico|cilium"
  
  # Verify storage
  kubectl get storageclasses
  kubectl get pv
  
  # Phase 2: Data Layer
  echo "--- Phase 2: Data Layer ---"
  
  # Start databases
  kubectl apply -f deployments/postgresql/
  kubectl apply -f deployments/redis/
  kubectl apply -f deployments/weaviate/
  
  # Wait for databases
  kubectl wait --for=condition=ready pod -l app=postgresql -n nephoran-system --timeout=300s
  kubectl wait --for=condition=ready pod -l app=redis -n nephoran-system --timeout=300s
  kubectl wait --for=condition=ready pod -l app=weaviate -n nephoran-system --timeout=300s
  
  # Phase 3: Core Services
  echo "--- Phase 3: Core Services ---"
  
  # Deploy core services
  kubectl apply -f deployments/controller/
  kubectl apply -f deployments/llm-processor/
  kubectl apply -f deployments/rag-api/
  
  # Wait for services
  kubectl wait --for=condition=available deployment --all -n nephoran-system --timeout=300s
  
  # Phase 4: Supporting Services
  echo "--- Phase 4: Supporting Services ---"
  
  # Deploy monitoring
  kubectl apply -f deployments/monitoring/
  
  # Deploy service mesh
  kubectl apply -f deployments/istio/
  
  # Deploy ingress
  kubectl apply -f deployments/ingress/
  
  # Phase 5: Validation
  echo "--- Phase 5: Validation ---"
  
  # Health checks
  ./scripts/health-check-all.sh
  
  # Integration tests
  ./scripts/run-integration-tests.sh
  
  # Performance validation
  ./scripts/validate-performance.sh
  
  echo "Service restoration completed at $(date)"
}
```

## DR Testing

### DR Test Plan

```yaml
DR Test Schedule:
  Monthly:
    - Backup validation
    - Restore to test environment
    - Service health checks
    
  Quarterly:
    - Cluster failover drill
    - Partial recovery test
    - Performance validation
    
  Semi-Annually:
    - Regional failover drill
    - Full recovery test
    - RTO/RPO validation
    
  Annually:
    - Complete disaster simulation
    - Multi-region failure test
    - Ransomware recovery drill
```

### DR Test Execution

```bash
#!/bin/bash
# DR test execution script

execute_dr_test() {
  local TEST_TYPE=$1
  
  echo "=== DR Test: $TEST_TYPE ==="
  echo "Test started at $(date)"
  
  case $TEST_TYPE in
    "backup-restore")
      echo "--- Backup/Restore Test ---"
      
      # Create test backup
      TEST_BACKUP_ID="test-$(date +%Y%m%d-%H%M%S)"
      perform_manual_backup
      
      # Restore to test cluster
      perform_full_recovery $TEST_BACKUP_ID test-cluster
      
      # Validate
      kubectl --context test-cluster get pods -n nephoran-system
      ;;
      
    "cluster-failover")
      echo "--- Cluster Failover Test ---"
      
      # Failover to backup cluster
      perform_cluster_failover production-primary production-backup
      
      # Run for 10 minutes
      sleep 600
      
      # Fail back
      perform_cluster_failover production-backup production-primary
      ;;
      
    "regional-failover")
      echo "--- Regional Failover Test ---"
      
      # Document current state
      kubectl get all -n nephoran-system > /tmp/pre-failover-state.txt
      
      # Perform failover
      perform_regional_failover us-east-1 us-west-2
      
      # Validate operations
      sleep 300
      ./scripts/validate-operations.sh
      
      # Fail back
      perform_regional_failover us-west-2 us-east-1
      ;;
      
    "ransomware-recovery")
      echo "--- Ransomware Recovery Test ---"
      
      # Simulate encryption (in test environment)
      kubectl --context test-cluster exec -n nephoran-system deployment/llm-processor -- \
        find /data -name "*.dat" -exec sh -c 'dd if=/dev/urandom of={} bs=1024 count=1' \;
      
      # Perform recovery
      perform_full_recovery "clean-backup-20250101" test-cluster
      
      # Validate
      verify_data_consistency
      ;;
  esac
  
  # Generate test report
  cat > /tmp/dr-test-report-$(date +%Y%m%d).md <<EOF
# DR Test Report

## Test Details
- Type: $TEST_TYPE
- Date: $(date)
- Duration: $(($(date +%s) - START_TIME)) seconds

## Results
- Status: $([ $? -eq 0 ] && echo "✓ Passed" || echo "✗ Failed")
- RTO Achieved: $(calculate_rto)
- RPO Achieved: $(calculate_rpo)

## Observations
$(cat /tmp/dr-test-observations.txt 2>/dev/null)

## Recommendations
- Review and update procedures based on test results
- Address any issues identified during testing
- Schedule follow-up test if needed

EOF
  
  echo "Test completed at $(date)"
  echo "Report: /tmp/dr-test-report-$(date +%Y%m%d).md"
}
```

## Post-Recovery Validation

### Validation Checklist

```bash
#!/bin/bash
# Post-recovery validation script

validate_recovery() {
  echo "=== Post-Recovery Validation ==="
  echo "Starting validation at $(date)"
  
  VALIDATION_PASSED=true
  
  # 1. Infrastructure validation
  echo "--- Infrastructure Validation ---"
  
  # Cluster health
  kubectl get nodes | grep -v Ready && VALIDATION_PASSED=false || echo "✓ All nodes ready"
  
  # Namespace existence
  for ns in nephoran-system monitoring istio-system; do
    kubectl get namespace $ns > /dev/null 2>&1 && \
      echo "✓ Namespace $ns exists" || \
      { echo "✗ Namespace $ns missing"; VALIDATION_PASSED=false; }
  done
  
  # 2. Service validation
  echo "--- Service Validation ---"
  
  # Pod health
  UNHEALTHY=$(kubectl get pods -n nephoran-system --no-headers | grep -v Running | wc -l)
  if [ $UNHEALTHY -eq 0 ]; then
    echo "✓ All pods healthy"
  else
    echo "✗ $UNHEALTHY pods unhealthy"
    kubectl get pods -n nephoran-system --no-headers | grep -v Running
    VALIDATION_PASSED=false
  fi
  
  # Service endpoints
  for svc in llm-processor rag-api nephoran-controller weaviate postgresql redis; do
    ENDPOINTS=$(kubectl get endpoints $svc -n nephoran-system -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null | wc -w)
    if [ $ENDPOINTS -gt 0 ]; then
      echo "✓ Service $svc has $ENDPOINTS endpoints"
    else
      echo "✗ Service $svc has no endpoints"
      VALIDATION_PASSED=false
    fi
  done
  
  # 3. Data validation
  echo "--- Data Validation ---"
  
  # Database connectivity
  kubectl exec -n nephoran-system postgresql-0 -- pg_isready > /dev/null 2>&1 && \
    echo "✓ Database accessible" || \
    { echo "✗ Database not accessible"; VALIDATION_PASSED=false; }
  
  # Database content
  INTENT_COUNT=$(kubectl get networkintents --all-namespaces --no-headers 2>/dev/null | wc -l)
  echo "NetworkIntents found: $INTENT_COUNT"
  [ $INTENT_COUNT -gt 0 ] && echo "✓ Intent data present" || echo "⚠️  No intent data found"
  
  # Vector database
  curl -s http://weaviate.nephoran-system:8080/v1/.well-known/ready | grep -q true && \
    echo "✓ Weaviate ready" || \
    { echo "✗ Weaviate not ready"; VALIDATION_PASSED=false; }
  
  # 4. Functional validation
  echo "--- Functional Validation ---"
  
  # API health check
  kubectl port-forward -n nephoran-system service/llm-processor 8080:8080 &
  PF_PID=$!
  sleep 5
  
  curl -s http://localhost:8080/health | grep -q healthy && \
    echo "✓ API health check passed" || \
    { echo "✗ API health check failed"; VALIDATION_PASSED=false; }
  
  kill $PF_PID 2>/dev/null
  
  # Create test intent
  kubectl apply -f - <<EOF
apiVersion: nephoran.io/v1
kind: NetworkIntent
metadata:
  name: recovery-validation-$(date +%s)
  namespace: default
spec:
  description: "Post-recovery validation test"
  priority: low
EOF
  
  sleep 30
  
  # Check intent processing
  kubectl get networkintent recovery-validation-* -n default -o jsonpath='{.status.phase}' | \
    grep -q "Succeeded" && echo "✓ Intent processing working" || \
    { echo "✗ Intent processing failed"; VALIDATION_PASSED=false; }
  
  # 5. Performance validation
  echo "--- Performance Validation ---"
  
  # Response time check
  RESPONSE_TIME=$(curl -o /dev/null -s -w '%{time_total}' http://localhost:8080/health)
  echo "API response time: ${RESPONSE_TIME}s"
  
  # 6. Security validation
  echo "--- Security Validation ---"
  
  # Certificate validity
  kubectl get certificates -n nephoran-system | grep -v True | grep -v READY && \
    { echo "✗ Certificate issues detected"; VALIDATION_PASSED=false; } || \
    echo "✓ All certificates valid"
  
  # Secret presence
  for secret in openai-key jwt-keys database-credentials; do
    kubectl get secret $secret -n nephoran-system > /dev/null 2>&1 && \
      echo "✓ Secret $secret exists" || \
      { echo "✗ Secret $secret missing"; VALIDATION_PASSED=false; }
  done
  
  # Generate validation report
  if [ "$VALIDATION_PASSED" = true ]; then
    echo "================================"
    echo "✓ RECOVERY VALIDATION PASSED"
    echo "================================"
  else
    echo "================================"
    echo "✗ RECOVERY VALIDATION FAILED"
    echo "================================"
    echo "Please review failures above and take corrective action"
  fi
  
  echo "Validation completed at $(date)"
}

# Run validation
validate_recovery
```

## Related Documentation

- [Master Operational Runbook](./operational-runbook-master.md) - Daily operations
- [Incident Response Runbook](./incident-response-runbook.md) - Incident procedures
- [Security Operations Runbook](./security-operations-runbook.md) - Security procedures
- [Backup Recovery Guide](../operations/BACKUP-RECOVERY.md) - Detailed backup procedures
- [Business Continuity Plan](../enterprise/business-continuity-disaster-recovery.md) - BC/DR strategy

---

**Note:** This disaster recovery runbook is critical for business continuity. Regular testing and updates are mandatory. All DR procedures must be validated quarterly and after any significant infrastructure changes.