#!/bin/bash
# Nephoran Intent Operator - Complete Monitoring Stack Setup
# TRL 9 Production-Ready Monitoring Infrastructure Deployment

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly MONITORING_DIR="${SCRIPT_DIR}/../../monitoring"
readonly PROMETHEUS_NAMESPACE="nephoran-monitoring"
readonly GRAFANA_NAMESPACE="nephoran-monitoring"
readonly JAEGER_NAMESPACE="nephoran-tracing"
readonly NWDAF_NAMESPACE="nwdaf-system"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Create monitoring namespaces
create_namespaces() {
    log "Creating monitoring namespaces"
    
    kubectl create namespace "$PROMETHEUS_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace "$GRAFANA_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace "$JAEGER_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace "$NWDAF_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Label namespaces for monitoring
    kubectl label namespace "$PROMETHEUS_NAMESPACE" monitoring=enabled --overwrite
    kubectl label namespace "$GRAFANA_NAMESPACE" monitoring=enabled --overwrite
    kubectl label namespace "$JAEGER_NAMESPACE" tracing=enabled --overwrite
    kubectl label namespace "$NWDAF_NAMESPACE" analytics=enabled --overwrite
}

# Deploy Prometheus stack
deploy_prometheus() {
    log "Deploying Prometheus monitoring stack"
    
    # Apply Prometheus configuration
    kubectl apply -f "$MONITORING_DIR/comprehensive-monitoring-stack.yaml"
    
    # Deploy Prometheus operator
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: $PROMETHEUS_NAMESPACE
  labels:
    app: prometheus
spec:
  replicas: 2
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      serviceAccountName: prometheus
      containers:
      - name: prometheus
        image: prom/prometheus:v2.47.0
        ports:
        - containerPort: 9090
        args:
          - --config.file=/etc/prometheus/prometheus.yml
          - --storage.tsdb.path=/prometheus
          - --storage.tsdb.retention.time=180d
          - --web.console.libraries=/usr/share/prometheus/console_libraries
          - --web.console.templates=/usr/share/prometheus/consoles
          - --web.enable-lifecycle
          - --web.enable-admin-api
          - --storage.tsdb.max-block-duration=2h
          - --storage.tsdb.min-block-duration=2h
        resources:
          requests:
            cpu: 500m
            memory: 2Gi
          limits:
            cpu: 2000m
            memory: 8Gi
        volumeMounts:
        - name: prometheus-config
          mountPath: /etc/prometheus
        - name: prometheus-storage
          mountPath: /prometheus
        - name: prometheus-rules
          mountPath: /etc/prometheus/rules
        livenessProbe:
          httpGet:
            path: /-/healthy
            port: 9090
          initialDelaySeconds: 30
          timeoutSeconds: 10
        readinessProbe:
          httpGet:
            path: /-/ready
            port: 9090
          initialDelaySeconds: 30
          timeoutSeconds: 10
      volumes:
      - name: prometheus-config
        configMap:
          name: prometheus-production-config
      - name: prometheus-storage
        persistentVolumeClaim:
          claimName: prometheus-storage
      - name: prometheus-rules
        configMap:
          name: prometheus-rules
---
apiVersion: v1
kind: Service
metadata:
  name: prometheus
  namespace: $PROMETHEUS_NAMESPACE
  labels:
    app: prometheus
spec:
  selector:
    app: prometheus
  ports:
  - port: 9090
    targetPort: 9090
  type: ClusterIP
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: prometheus-storage
  namespace: $PROMETHEUS_NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
  storageClassName: fast-ssd
EOF
    
    # Apply alerting rules
    kubectl create configmap prometheus-rules \
        --from-file="$MONITORING_DIR/prometheus/rules/" \
        -n "$PROMETHEUS_NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    log "Prometheus deployment completed"
}

# Deploy AlertManager
deploy_alertmanager() {
    log "Deploying AlertManager"
    
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertmanager
  namespace: $PROMETHEUS_NAMESPACE
  labels:
    app: alertmanager
spec:
  replicas: 2
  selector:
    matchLabels:
      app: alertmanager
  template:
    metadata:
      labels:
        app: alertmanager
    spec:
      containers:
      - name: alertmanager
        image: prom/alertmanager:v0.26.0
        ports:
        - containerPort: 9093
        args:
          - --config.file=/etc/alertmanager/alertmanager.yml
          - --storage.path=/alertmanager
          - --web.external-url=http://alertmanager.monitoring.local
          - --cluster.listen-address=0.0.0.0:9094
          - --cluster.peer=alertmanager-0.alertmanager.monitoring.svc.cluster.local:9094
          - --cluster.peer=alertmanager-1.alertmanager.monitoring.svc.cluster.local:9094
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        volumeMounts:
        - name: alertmanager-config
          mountPath: /etc/alertmanager
        - name: alertmanager-storage
          mountPath: /alertmanager
        livenessProbe:
          httpGet:
            path: /-/healthy
            port: 9093
          initialDelaySeconds: 30
          timeoutSeconds: 10
        readinessProbe:
          httpGet:
            path: /-/ready
            port: 9093
          initialDelaySeconds: 30
          timeoutSeconds: 10
      volumes:
      - name: alertmanager-config
        configMap:
          name: alertmanager-config
      - name: alertmanager-storage
        persistentVolumeClaim:
          claimName: alertmanager-storage
---
apiVersion: v1
kind: Service
metadata:
  name: alertmanager
  namespace: $PROMETHEUS_NAMESPACE
  labels:
    app: alertmanager
spec:
  selector:
    app: alertmanager
  ports:
  - port: 9093
    targetPort: 9093
  type: ClusterIP
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: alertmanager-storage
  namespace: $PROMETHEUS_NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: fast-ssd
EOF
    
    log "AlertManager deployment completed"
}

# Deploy Grafana
deploy_grafana() {
    log "Deploying Grafana dashboards"
    
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: $GRAFANA_NAMESPACE
  labels:
    app: grafana
spec:
  replicas: 2
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana-enterprise:10.1.0
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_PASSWORD
          valueFrom:
            secretKeyRef:
              name: grafana-secrets
              key: admin-password
        - name: GF_DATABASE_TYPE
          value: postgres
        - name: GF_DATABASE_HOST
          value: grafana-db:5432
        - name: GF_DATABASE_NAME
          value: grafana
        - name: GF_DATABASE_USER
          value: grafana
        - name: GF_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: grafana-secrets
              key: db-password
        - name: GF_INSTALL_PLUGINS
          value: grafana-piechart-panel,grafana-worldmap-panel,grafana-clock-panel
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 1000m
            memory: 2Gi
        volumeMounts:
        - name: grafana-storage
          mountPath: /var/lib/grafana
        - name: grafana-dashboards
          mountPath: /etc/grafana/provisioning/dashboards
        - name: grafana-datasources
          mountPath: /etc/grafana/provisioning/datasources
        livenessProbe:
          httpGet:
            path: /api/health
            port: 3000
          initialDelaySeconds: 30
          timeoutSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/health
            port: 3000
          initialDelaySeconds: 30
          timeoutSeconds: 10
      volumes:
      - name: grafana-storage
        persistentVolumeClaim:
          claimName: grafana-storage
      - name: grafana-dashboards
        configMap:
          name: grafana-dashboards
      - name: grafana-datasources
        configMap:
          name: grafana-datasources
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: $GRAFANA_NAMESPACE
  labels:
    app: grafana
spec:
  selector:
    app: grafana
  ports:
  - port: 3000
    targetPort: 3000
  type: LoadBalancer
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: grafana-storage
  namespace: $GRAFANA_NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
  storageClassName: standard
EOF
    
    # Create Grafana dashboard ConfigMaps
    kubectl create configmap grafana-dashboards \
        --from-file="$MONITORING_DIR/grafana/dashboards/" \
        -n "$GRAFANA_NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create datasource configuration
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-datasources
  namespace: $GRAFANA_NAMESPACE
data:
  datasources.yaml: |
    apiVersion: 1
    datasources:
    - name: Prometheus
      type: prometheus
      access: proxy
      url: http://prometheus.$PROMETHEUS_NAMESPACE:9090
      isDefault: true
      jsonData:
        timeInterval: "30s"
    - name: Jaeger
      type: jaeger
      access: proxy
      url: http://jaeger-query.$JAEGER_NAMESPACE:16686
    - name: Loki
      type: loki
      access: proxy
      url: http://loki.$PROMETHEUS_NAMESPACE:3100
EOF
    
    log "Grafana deployment completed"
}

# Deploy Jaeger tracing
deploy_jaeger() {
    log "Deploying Jaeger distributed tracing"
    
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger-collector
  namespace: $JAEGER_NAMESPACE
  labels:
    app: jaeger-collector
spec:
  replicas: 3
  selector:
    matchLabels:
      app: jaeger-collector
  template:
    metadata:
      labels:
        app: jaeger-collector
    spec:
      containers:
      - name: jaeger-collector
        image: jaegertracing/jaeger-collector:1.49.0
        ports:
        - containerPort: 14267
        - containerPort: 14268
        - containerPort: 9411
        env:
        - name: SPAN_STORAGE_TYPE
          value: elasticsearch
        - name: ES_SERVER_URLS
          value: http://elasticsearch:9200
        resources:
          requests:
            cpu: 200m
            memory: 512Mi
          limits:
            cpu: 500m
            memory: 1Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger-query
  namespace: $JAEGER_NAMESPACE
  labels:
    app: jaeger-query
spec:
  replicas: 2
  selector:
    matchLabels:
      app: jaeger-query
  template:
    metadata:
      labels:
        app: jaeger-query
    spec:
      containers:
      - name: jaeger-query
        image: jaegertracing/jaeger-query:1.49.0
        ports:
        - containerPort: 16686
        env:
        - name: SPAN_STORAGE_TYPE
          value: elasticsearch
        - name: ES_SERVER_URLS
          value: http://elasticsearch:9200
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger-collector
  namespace: $JAEGER_NAMESPACE
  labels:
    app: jaeger-collector
spec:
  selector:
    app: jaeger-collector
  ports:
  - port: 14267
    targetPort: 14267
    name: tchannel
  - port: 14268
    targetPort: 14268
    name: http
  - port: 9411
    targetPort: 9411
    name: zipkin
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger-query
  namespace: $JAEGER_NAMESPACE
  labels:
    app: jaeger-query
spec:
  selector:
    app: jaeger-query
  ports:
  - port: 16686
    targetPort: 16686
  type: LoadBalancer
EOF
    
    log "Jaeger deployment completed"
}

# Deploy NWDAF analytics
deploy_nwdaf_analytics() {
    log "Deploying NWDAF analytics components"
    
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nwdaf-collector
  namespace: $NWDAF_NAMESPACE
  labels:
    app: nwdaf-collector
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nwdaf-collector
  template:
    metadata:
      labels:
        app: nwdaf-collector
    spec:
      containers:
      - name: nwdaf-collector
        image: nephoran/nwdaf-collector:v2.1.0
        ports:
        - containerPort: 8080
        - containerPort: 9090
        env:
        - name: KAFKA_BROKERS
          value: kafka:9092
        - name: PROMETHEUS_URL
          value: http://prometheus.$PROMETHEUS_NAMESPACE:9090
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 1000m
            memory: 2Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 30
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nwdaf-analytics
  namespace: $NWDAF_NAMESPACE
  labels:
    app: nwdaf-analytics
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nwdaf-analytics
  template:
    metadata:
      labels:
        app: nwdaf-analytics
    spec:
      containers:
      - name: nwdaf-analytics
        image: nephoran/nwdaf-analytics:v2.1.0
        ports:
        - containerPort: 8080
        - containerPort: 9090
        env:
        - name: ML_MODEL_PATH
          value: /models/nwdaf-model.pkl
        - name: KAFKA_BROKERS
          value: kafka:9092
        resources:
          requests:
            cpu: 1000m
            memory: 2Gi
          limits:
            cpu: 2000m
            memory: 4Gi
        volumeMounts:
        - name: ml-models
          mountPath: /models
      volumes:
      - name: ml-models
        persistentVolumeClaim:
          claimName: nwdaf-models
---
apiVersion: v1
kind: Service
metadata:
  name: nwdaf-collector
  namespace: $NWDAF_NAMESPACE
  labels:
    app: nwdaf-collector
spec:
  selector:
    app: nwdaf-collector
  ports:
  - port: 8080
    targetPort: 8080
    name: http
  - port: 9090
    targetPort: 9090
    name: metrics
---
apiVersion: v1
kind: Service
metadata:
  name: nwdaf-analytics
  namespace: $NWDAF_NAMESPACE
  labels:
    app: nwdaf-analytics
spec:
  selector:
    app: nwdaf-analytics
  ports:
  - port: 8080
    targetPort: 8080
    name: http
  - port: 9090
    targetPort: 9090
    name: metrics
EOF
    
    log "NWDAF analytics deployment completed"
}

# Setup automated remediation
setup_automated_remediation() {
    log "Setting up automated remediation"
    
    # Create webhook service for AlertManager
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: automated-remediation
  namespace: $PROMETHEUS_NAMESPACE
  labels:
    app: automated-remediation
spec:
  replicas: 1
  selector:
    matchLabels:
      app: automated-remediation
  template:
    metadata:
      labels:
        app: automated-remediation
    spec:
      serviceAccountName: automated-remediation
      containers:
      - name: webhook-receiver
        image: nginx:alpine
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: remediation-script
          mountPath: /scripts
        - name: webhook-config
          mountPath: /etc/nginx/conf.d
        command: ["/bin/sh"]
        args:
        - -c
        - |
          while true; do
            nc -l -p 8080 -e /scripts/automated-remediation.sh webhook
          done
      volumes:
      - name: remediation-script
        configMap:
          name: automated-remediation-script
          defaultMode: 0755
      - name: webhook-config
        configMap:
          name: webhook-nginx-config
---
apiVersion: v1
kind: Service
metadata:
  name: automated-remediation
  namespace: $PROMETHEUS_NAMESPACE
  labels:
    app: automated-remediation
spec:
  selector:
    app: automated-remediation
  ports:
  - port: 8080
    targetPort: 8080
EOF
    
    # Create ConfigMap with remediation script
    kubectl create configmap automated-remediation-script \
        --from-file="$SCRIPT_DIR/automated-remediation.sh" \
        -n "$PROMETHEUS_NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Setup CronJob for periodic health checks
    cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: automated-health-checks
  namespace: $PROMETHEUS_NAMESPACE
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: automated-remediation
          containers:
          - name: health-check
            image: bitnami/kubectl:latest
            command: ["/bin/bash"]
            args:
            - -c
            - |
              /scripts/automated-remediation.sh health-check
            volumeMounts:
            - name: remediation-script
              mountPath: /scripts
          volumes:
          - name: remediation-script
            configMap:
              name: automated-remediation-script
              defaultMode: 0755
          restartPolicy: OnFailure
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: automated-optimization
  namespace: $PROMETHEUS_NAMESPACE
spec:
  schedule: "0 */2 * * *"  # Every 2 hours
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: automated-remediation
          containers:
          - name: optimization
            image: bitnami/kubectl:latest
            command: ["/bin/bash"]
            args:
            - -c
            - |
              /scripts/automated-remediation.sh full-optimization
            volumeMounts:
            - name: remediation-script
              mountPath: /scripts
          volumes:
          - name: remediation-script
            configMap:
              name: automated-remediation-script
              defaultMode: 0755
          restartPolicy: OnFailure
EOF
    
    log "Automated remediation setup completed"
}

# Create RBAC permissions
create_rbac() {
    log "Creating RBAC permissions for monitoring components"
    
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: $PROMETHEUS_NAMESPACE
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: automated-remediation
  namespace: $PROMETHEUS_NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus
rules:
- apiGroups: [""]
  resources: ["nodes", "nodes/metrics", "services", "endpoints", "pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "daemonsets"]
  verbs: ["get", "list", "watch"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: automated-remediation
rules:
- apiGroups: [""]
  resources: ["pods", "services", "configmaps"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete", "scale"]
- apiGroups: ["batch"]
  resources: ["jobs", "cronjobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["nephoran.io"]
  resources: ["networkintents", "e2nodes", "e2subscriptions", "a1policies"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus
subjects:
- kind: ServiceAccount
  name: prometheus
  namespace: $PROMETHEUS_NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: automated-remediation
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: automated-remediation
subjects:
- kind: ServiceAccount
  name: automated-remediation
  namespace: $PROMETHEUS_NAMESPACE
EOF
    
    log "RBAC permissions created"
}

# Setup log aggregation
setup_log_aggregation() {
    log "Setting up log aggregation with ELK stack"
    
    # Deploy Elasticsearch
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: elasticsearch
  namespace: $PROMETHEUS_NAMESPACE
spec:
  serviceName: elasticsearch
  replicas: 3
  selector:
    matchLabels:
      app: elasticsearch
  template:
    metadata:
      labels:
        app: elasticsearch
    spec:
      containers:
      - name: elasticsearch
        image: docker.elastic.co/elasticsearch/elasticsearch:8.9.0
        ports:
        - containerPort: 9200
        - containerPort: 9300
        env:
        - name: discovery.type
          value: zen
        - name: cluster.name
          value: nephoran-logs
        - name: node.name
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: discovery.seed_hosts
          value: elasticsearch-0.elasticsearch,elasticsearch-1.elasticsearch,elasticsearch-2.elasticsearch
        - name: cluster.initial_master_nodes
          value: elasticsearch-0,elasticsearch-1,elasticsearch-2
        - name: ES_JAVA_OPTS
          value: "-Xms2g -Xmx2g"
        resources:
          requests:
            cpu: 1000m
            memory: 4Gi
          limits:
            cpu: 2000m
            memory: 8Gi
        volumeMounts:
        - name: elasticsearch-data
          mountPath: /usr/share/elasticsearch/data
  volumeClaimTemplates:
  - metadata:
      name: elasticsearch-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
      storageClassName: fast-ssd
---
apiVersion: v1
kind: Service
metadata:
  name: elasticsearch
  namespace: $PROMETHEUS_NAMESPACE
spec:
  selector:
    app: elasticsearch
  ports:
  - port: 9200
    targetPort: 9200
  clusterIP: None
EOF
    
    # Deploy Kibana
    cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kibana
  namespace: $PROMETHEUS_NAMESPACE
spec:
  replicas: 2
  selector:
    matchLabels:
      app: kibana
  template:
    metadata:
      labels:
        app: kibana
    spec:
      containers:
      - name: kibana
        image: docker.elastic.co/kibana/kibana:8.9.0
        ports:
        - containerPort: 5601
        env:
        - name: ELASTICSEARCH_HOSTS
          value: http://elasticsearch:9200
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 1000m
            memory: 2Gi
---
apiVersion: v1
kind: Service
metadata:
  name: kibana
  namespace: $PROMETHEUS_NAMESPACE
spec:
  selector:
    app: kibana
  ports:
  - port: 5601
    targetPort: 5601
  type: LoadBalancer
EOF
    
    log "Log aggregation setup completed"
}

# Verify deployment
verify_deployment() {
    log "Verifying monitoring stack deployment"
    
    # Wait for deployments to be ready
    local deployments=(
        "prometheus:$PROMETHEUS_NAMESPACE"
        "alertmanager:$PROMETHEUS_NAMESPACE"
        "grafana:$GRAFANA_NAMESPACE"
        "jaeger-collector:$JAEGER_NAMESPACE"
        "jaeger-query:$JAEGER_NAMESPACE"
        "nwdaf-collector:$NWDAF_NAMESPACE"
        "nwdaf-analytics:$NWDAF_NAMESPACE"
    )
    
    for deployment in "${deployments[@]}"; do
        local dep_name="${deployment%:*}"
        local dep_namespace="${deployment#*:}"
        
        log "Waiting for deployment $dep_name in namespace $dep_namespace"
        kubectl rollout status deployment/"$dep_name" -n "$dep_namespace" --timeout=300s
    done
    
    # Verify services are accessible
    log "Verifying service endpoints"
    
    if kubectl port-forward -n "$PROMETHEUS_NAMESPACE" svc/prometheus 9090:9090 &
    then
        local pf_pid=$!
        sleep 5
        if curl -f http://localhost:9090/-/healthy; then
            log "Prometheus health check: OK"
        else
            error "Prometheus health check failed"
        fi
        kill $pf_pid
    fi
    
    log "Monitoring stack verification completed"
}

# Main execution
main() {
    local action="${1:-deploy-all}"
    
    log "Starting monitoring stack setup with action: $action"
    
    case "$action" in
        "deploy-all")
            create_namespaces
            create_rbac
            deploy_prometheus
            deploy_alertmanager
            deploy_grafana
            deploy_jaeger
            deploy_nwdaf_analytics
            setup_log_aggregation
            setup_automated_remediation
            verify_deployment
            ;;
        "prometheus")
            create_namespaces
            create_rbac
            deploy_prometheus
            ;;
        "grafana")
            deploy_grafana
            ;;
        "jaeger")
            deploy_jaeger
            ;;
        "nwdaf")
            deploy_nwdaf_analytics
            ;;
        "verify")
            verify_deployment
            ;;
        *)
            error "Unknown action: $action"
            echo "Usage: $0 {deploy-all|prometheus|grafana|jaeger|nwdaf|verify}"
            exit 1
            ;;
    esac
    
    log "Monitoring stack setup completed successfully"
    log "Access points:"
    log "  - Prometheus: kubectl port-forward -n $PROMETHEUS_NAMESPACE svc/prometheus 9090:9090"
    log "  - Grafana: kubectl port-forward -n $GRAFANA_NAMESPACE svc/grafana 3000:3000"
    log "  - AlertManager: kubectl port-forward -n $PROMETHEUS_NAMESPACE svc/alertmanager 9093:9093"
    log "  - Jaeger: kubectl port-forward -n $JAEGER_NAMESPACE svc/jaeger-query 16686:16686"
    log "  - Kibana: kubectl port-forward -n $PROMETHEUS_NAMESPACE svc/kibana 5601:5601"
}

main "$@"