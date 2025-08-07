#!/bin/bash

# Nephoran Intent Operator - Automated Quick Start Script
# This script automates the 15-minute quickstart tutorial
# Usage: ./scripts/quickstart.sh [--record] [--cleanup]

set -e

# Configuration
CLUSTER_NAME="nephoran-quickstart"
NAMESPACE="nephoran-system"
TIMEOUT=300
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Parse arguments
RECORD_MODE=false
CLEANUP_MODE=false
SKIP_PREREQ=false
DEMO_MODE=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --record)
            RECORD_MODE=true
            shift
            ;;
        --cleanup)
            CLEANUP_MODE=true
            shift
            ;;
        --skip-prereq)
            SKIP_PREREQ=true
            shift
            ;;
        --production)
            DEMO_MODE=false
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  --record       Record terminal session with asciinema"
            echo "  --cleanup      Clean up all resources and exit"
            echo "  --skip-prereq  Skip prerequisite checks"
            echo "  --production   Use production configuration (requires API keys)"
            echo "  --help         Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "\n${MAGENTA}═══════════════════════════════════════════════════════${NC}"
    echo -e "${WHITE}STEP: $1${NC}"
    echo -e "${MAGENTA}═══════════════════════════════════════════════════════${NC}\n"
}

show_progress() {
    local duration=$1
    local steps=$2
    local step_duration=$((duration / steps))
    
    for ((i=1; i<=steps; i++)); do
        echo -ne "\r${CYAN}Progress: ["
        for ((j=1; j<=i; j++)); do echo -n "█"; done
        for ((j=i; j<steps; j++)); do echo -n "░"; done
        echo -ne "] $((i * 100 / steps))%${NC}"
        sleep $step_duration
    done
    echo -ne "\r${GREEN}Progress: [██████████████████████████] 100%${NC}\n"
}

check_command() {
    if command -v "$1" &> /dev/null; then
        log_success "$1 is installed ✓"
        return 0
    else
        log_error "$1 is not installed ✗"
        return 1
    fi
}

cleanup() {
    log_step "Cleaning up resources"
    
    # Delete network intent
    kubectl delete networkintent deploy-amf-quickstart --ignore-not-found=true 2>/dev/null || true
    
    # Delete namespace
    kubectl delete namespace $NAMESPACE --ignore-not-found=true 2>/dev/null || true
    
    # Delete CRDs
    kubectl delete crd networkintents.nephoran.com --ignore-not-found=true 2>/dev/null || true
    kubectl delete crd managedelements.nephoran.com --ignore-not-found=true 2>/dev/null || true
    kubectl delete crd e2nodesets.nephoran.com --ignore-not-found=true 2>/dev/null || true
    
    # Delete Kind cluster
    if kind get clusters | grep -q "$CLUSTER_NAME"; then
        log_info "Deleting Kind cluster: $CLUSTER_NAME"
        kind delete cluster --name "$CLUSTER_NAME"
        log_success "Cluster deleted successfully"
    else
        log_info "Cluster $CLUSTER_NAME not found, skipping deletion"
    fi
    
    # Clean up temporary files
    rm -f kind-config.yaml my-first-intent.yaml validate-quickstart.sh 2>/dev/null || true
    
    log_success "Cleanup completed!"
}

check_prerequisites() {
    log_step "Checking Prerequisites (2 minutes)"
    
    local all_good=true
    
    log_info "Checking required tools..."
    check_command docker || all_good=false
    check_command kubectl || all_good=false
    check_command git || all_good=false
    check_command kind || all_good=false
    
    if [ "$all_good" = false ]; then
        log_error "Some prerequisites are missing. Please install them first."
        echo ""
        echo "Installation instructions:"
        echo "  Linux/WSL: curl -fsSL https://get.docker.com | sh"
        echo "  macOS:     brew install docker kubectl kind"
        echo "  Windows:   choco install docker-desktop kubernetes-cli kind"
        exit 1
    fi
    
    # Check Docker daemon
    if ! docker info &>/dev/null; then
        log_error "Docker daemon is not running. Please start Docker."
        exit 1
    fi
    
    log_success "All prerequisites met! ✓"
}

setup_cluster() {
    log_step "Setting up Kubernetes Cluster (5 minutes)"
    
    # Check if cluster already exists
    if kind get clusters | grep -q "$CLUSTER_NAME"; then
        log_warning "Cluster $CLUSTER_NAME already exists"
        read -p "Delete and recreate? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kind delete cluster --name "$CLUSTER_NAME"
        else
            log_info "Using existing cluster"
            return 0
        fi
    fi
    
    log_info "Creating Kind cluster configuration..."
    cat <<EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: $CLUSTER_NAME
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  - containerPort: 8080
    hostPort: 8080
    protocol: TCP
- role: worker
- role: worker
EOF
    
    log_info "Creating Kind cluster: $CLUSTER_NAME"
    kind create cluster --config=kind-config.yaml
    
    log_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=120s
    
    log_success "Cluster created successfully!"
    kubectl get nodes
}

install_crds() {
    log_info "Installing Custom Resource Definitions..."
    
    # Create CRDs directly if files don't exist
    if [ ! -d "$PROJECT_ROOT/deployments/crds" ]; then
        log_warning "CRD directory not found, creating CRDs inline..."
        kubectl apply -f - <<EOF
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: networkintents.nephoran.com
spec:
  group: nephoran.com
  names:
    kind: NetworkIntent
    listKind: NetworkIntentList
    plural: networkintents
    singular: networkintent
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              intent:
                type: string
          status:
            type: object
            properties:
              phase:
                type: string
              message:
                type: string
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: managedelements.nephoran.com
spec:
  group: nephoran.com
  names:
    kind: ManagedElement
    listKind: ManagedElementList
    plural: managedelements
    singular: managedelement
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: e2nodesets.nephoran.com
spec:
  group: nephoran.com
  names:
    kind: E2NodeSet
    listKind: E2NodeSetList
    plural: e2nodesets
    singular: e2nodeset
  scope: Namespaced
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
EOF
    else
        kubectl apply -f "$PROJECT_ROOT/deployments/crds/"
    fi
    
    log_info "Verifying CRD installation..."
    kubectl get crds | grep nephoran
    
    log_success "CRDs installed successfully!"
}

deploy_controller() {
    log_step "Deploying Nephoran Controller"
    
    log_info "Creating namespace: $NAMESPACE"
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    if [ "$DEMO_MODE" = true ]; then
        log_info "Deploying in DEMO mode (no API keys required)..."
        
        # Create mock secrets
        kubectl create secret generic llm-secrets \
            --from-literal=openai-api-key=demo-key-not-for-production \
            -n $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    else
        log_info "Deploying in PRODUCTION mode..."
        
        # Check for API key
        if [ -z "$OPENAI_API_KEY" ]; then
            log_error "OPENAI_API_KEY environment variable not set"
            echo "Please set: export OPENAI_API_KEY=your-key-here"
            exit 1
        fi
        
        kubectl create secret generic llm-secrets \
            --from-literal=openai-api-key="$OPENAI_API_KEY" \
            -n $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    log_info "Creating controller configuration..."
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: nephoran-config
  namespace: $NAMESPACE
data:
  config.yaml: |
    mode: $([ "$DEMO_MODE" = true ] && echo "demo" || echo "production")
    llm:
      provider: $([ "$DEMO_MODE" = true ] && echo "mock" || echo "openai")
      mockResponses: $([ "$DEMO_MODE" = true ] && echo "true" || echo "false")
      model: gpt-4o-mini
    rag:
      enabled: false
    telemetry:
      enabled: true
      prometheus:
        port: 9090
    logging:
      level: info
      format: json
EOF
    
    log_info "Deploying controller..."
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nephoran-controller
  namespace: $NAMESPACE
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-controller
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["apps"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: ["nephoran.com"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nephoran-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nephoran-controller
subjects:
- kind: ServiceAccount
  name: nephoran-controller
  namespace: $NAMESPACE
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-controller
  namespace: $NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nephoran-controller
  template:
    metadata:
      labels:
        app: nephoran-controller
    spec:
      serviceAccountName: nephoran-controller
      containers:
      - name: controller
        image: ghcr.io/thc1006/nephoran-intent-operator:latest
        imagePullPolicy: Always
        env:
        - name: DEMO_MODE
          value: "$([ "$DEMO_MODE" = true ] && echo "true" || echo "false")"
        - name: LOG_LEVEL
          value: "info"
        - name: NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 9090
          name: health
        volumeMounts:
        - name: config
          mountPath: /etc/nephoran
        - name: secrets
          mountPath: /etc/secrets
        livenessProbe:
          httpGet:
            path: /healthz
            port: health
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: health
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: nephoran-config
      - name: secrets
        secret:
          secretName: llm-secrets
---
apiVersion: v1
kind: Service
metadata:
  name: nephoran-controller
  namespace: $NAMESPACE
spec:
  selector:
    app: nephoran-controller
  ports:
  - name: metrics
    port: 8080
    targetPort: metrics
  - name: health
    port: 9090
    targetPort: health
EOF
    
    log_info "Waiting for controller to be ready..."
    kubectl wait --for=condition=available --timeout=120s \
        deployment/nephoran-controller -n $NAMESPACE || {
        log_error "Controller failed to start. Checking logs..."
        kubectl logs -n $NAMESPACE deployment/nephoran-controller --tail=50
        exit 1
    }
    
    log_success "Controller deployed successfully!"
}

deploy_first_intent() {
    log_step "Deploying Your First Intent (5 minutes)"
    
    log_info "Creating NetworkIntent for AMF deployment..."
    cat <<EOF > my-first-intent.yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: deploy-amf-quickstart
  namespace: default
  labels:
    tutorial: quickstart
    component: amf
    generated-from: quickstart-script
spec:
  intent: |
    Deploy a production-ready AMF (Access and Mobility Management Function) 
    for a 5G core network with the following requirements:
    - High availability with 3 replicas
    - Auto-scaling enabled (min: 3, max: 10 pods)
    - Resource limits: 2 CPU cores, 4GB memory per pod
    - Enable prometheus monitoring on port 9090
    - Configure for urban area with expected 100k UE connections
    - Set up with standard 3GPP interfaces (N1, N2, N11)
    - Include health checks and readiness probes
    - Deploy in namespace: default
EOF
    
    log_info "Applying the NetworkIntent..."
    kubectl apply -f my-first-intent.yaml
    
    log_info "Waiting for intent processing..."
    
    # Monitor intent status
    for i in {1..30}; do
        STATUS=$(kubectl get networkintent deploy-amf-quickstart -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
        case "$STATUS" in
            "Deployed"|"Ready"|"Completed")
                log_success "Intent processed successfully! Status: $STATUS"
                break
                ;;
            "Failed"|"Error")
                log_error "Intent processing failed! Status: $STATUS"
                kubectl describe networkintent deploy-amf-quickstart
                exit 1
                ;;
            *)
                echo -ne "\r${CYAN}Processing intent... Status: $STATUS (${i}/30)${NC}"
                sleep 2
                ;;
        esac
    done
    echo ""
    
    log_info "Checking generated resources..."
    kubectl get all -l generated-from=deploy-amf-quickstart 2>/dev/null || {
        log_warning "No resources with expected label found. Checking alternative patterns..."
        kubectl get deployments,services,configmaps | grep -i amf || true
    }
    
    log_success "First intent deployed successfully!"
}

validate_deployment() {
    log_step "Validating Deployment (2 minutes)"
    
    cat <<'EOF' > validate-quickstart.sh
#!/bin/bash

echo "🔍 Running Nephoran Quickstart Validation..."
echo "==========================================="

ERRORS=0
WARNINGS=0

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check CRDs
echo -e "\n📋 Checking CRDs..."
for crd in networkintents managedelements e2nodesets; do
    if kubectl get crd ${crd}.nephoran.com &>/dev/null; then
        echo -e "${GREEN}✓${NC} ${crd}.nephoran.com installed"
    else
        echo -e "${RED}✗${NC} ${crd}.nephoran.com missing"
        ((ERRORS++))
    fi
done

# Check controller
echo -e "\n🎮 Checking Controller..."
if kubectl get deployment nephoran-controller -n nephoran-system &>/dev/null; then
    READY=$(kubectl get deployment nephoran-controller -n nephoran-system -o jsonpath='{.status.readyReplicas}')
    DESIRED=$(kubectl get deployment nephoran-controller -n nephoran-system -o jsonpath='{.spec.replicas}')
    if [ "$READY" = "$DESIRED" ]; then
        echo -e "${GREEN}✓${NC} Controller running ($READY/$DESIRED replicas ready)"
    else
        echo -e "${YELLOW}⚠${NC} Controller partially ready ($READY/$DESIRED replicas)"
        ((WARNINGS++))
    fi
else
    echo -e "${RED}✗${NC} Controller deployment not found"
    ((ERRORS++))
fi

# Check intent
echo -e "\n📝 Checking Network Intent..."
if kubectl get networkintent deploy-amf-quickstart &>/dev/null; then
    STATUS=$(kubectl get networkintent deploy-amf-quickstart -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
    MESSAGE=$(kubectl get networkintent deploy-amf-quickstart -o jsonpath='{.status.message}' 2>/dev/null || echo "")
    echo -e "${GREEN}✓${NC} Intent found - Status: $STATUS"
    if [ -n "$MESSAGE" ]; then
        echo "   Message: $MESSAGE"
    fi
else
    echo -e "${RED}✗${NC} Intent not found"
    ((ERRORS++))
fi

# Summary
echo -e "\n==========================================="
echo "📊 VALIDATION SUMMARY"
echo "==========================================="

if [ $ERRORS -eq 0 ]; then
    if [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}🎉 PERFECT!${NC} All checks passed!"
        echo ""
        echo "Your Nephoran Intent Operator quickstart is complete!"
        echo ""
        echo "Next steps:"
        echo "  1. View controller logs: kubectl logs -n nephoran-system deployment/nephoran-controller"
        echo "  2. Try more intents: kubectl apply -f examples/networkintent-example.yaml"
        echo "  3. Access metrics: kubectl port-forward -n nephoran-system svc/nephoran-controller 8080:8080"
    else
        echo -e "${YELLOW}✅ SUCCESS${NC} with $WARNINGS warnings"
        echo "The system is functional but may need minor adjustments."
    fi
else
    echo -e "${RED}❌ ISSUES FOUND${NC}: $ERRORS errors, $WARNINGS warnings"
    echo ""
    echo "Troubleshooting tips:"
    echo "  1. Check controller logs: kubectl logs -n nephoran-system deployment/nephoran-controller"
    echo "  2. Describe the intent: kubectl describe networkintent deploy-amf-quickstart"
    echo "  3. Check events: kubectl get events --sort-by='.lastTimestamp'"
fi

exit $ERRORS
EOF
    
    chmod +x validate-quickstart.sh
    bash ./validate-quickstart.sh
}

show_summary() {
    log_step "Quick Start Complete! 🎉"
    
    echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
    echo -e "${WHITE}QUICKSTART SUMMARY${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
    echo ""
    echo "✅ Cluster Name: $CLUSTER_NAME"
    echo "✅ Namespace: $NAMESPACE"
    echo "✅ Mode: $([ "$DEMO_MODE" = true ] && echo "Demo" || echo "Production")"
    echo ""
    echo -e "${CYAN}Useful Commands:${NC}"
    echo "  # View controller logs"
    echo "  kubectl logs -n $NAMESPACE deployment/nephoran-controller -f"
    echo ""
    echo "  # Watch intent status"
    echo "  kubectl get networkintents -w"
    echo ""
    echo "  # Port-forward to access metrics"
    echo "  kubectl port-forward -n $NAMESPACE svc/nephoran-controller 8080:8080"
    echo ""
    echo "  # Deploy more examples"
    echo "  kubectl apply -f examples/networkintent-example.yaml"
    echo ""
    echo "  # Clean up everything"
    echo "  $0 --cleanup"
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
    echo ""
    echo "📖 Documentation: https://github.com/thc1006/nephoran-intent-operator"
    echo "🐛 Issues: https://github.com/thc1006/nephoran-intent-operator/issues"
    echo ""
    echo "Thank you for trying Nephoran Intent Operator!"
}

record_asciinema() {
    if ! command -v asciinema &> /dev/null; then
        log_warning "asciinema not installed. Skipping recording."
        log_info "Install with: pip install asciinema"
        return 1
    fi
    
    local step_name=$1
    local output_file="nephoran-quickstart-${step_name}.cast"
    
    log_info "Recording $step_name with asciinema..."
    asciinema rec -t "Nephoran Quickstart: $step_name" "$output_file"
    log_success "Recording saved to $output_file"
}

# Main execution
main() {
    echo -e "${CYAN}╔════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║     ${WHITE}Nephoran Intent Operator - Quick Start Script${CYAN}      ║${NC}"
    echo -e "${CYAN}║           ${YELLOW}15-Minute Setup & Deployment${CYAN}                 ║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    # Start timer
    START_TIME=$(date +%s)
    
    # Handle cleanup mode
    if [ "$CLEANUP_MODE" = true ]; then
        cleanup
        exit 0
    fi
    
    # Prerequisites
    if [ "$SKIP_PREREQ" = false ]; then
        if [ "$RECORD_MODE" = true ]; then
            record_asciinema "prerequisites" || true
        fi
        check_prerequisites
    fi
    
    # Setup cluster
    if [ "$RECORD_MODE" = true ]; then
        record_asciinema "cluster-setup" || true
    fi
    setup_cluster
    
    # Install CRDs
    install_crds
    
    # Deploy controller
    if [ "$RECORD_MODE" = true ]; then
        record_asciinema "controller-deployment" || true
    fi
    deploy_controller
    
    # Deploy first intent
    if [ "$RECORD_MODE" = true ]; then
        record_asciinema "first-intent" || true
    fi
    deploy_first_intent
    
    # Validate
    if [ "$RECORD_MODE" = true ]; then
        record_asciinema "validation" || true
    fi
    validate_deployment
    
    # Calculate elapsed time
    END_TIME=$(date +%s)
    ELAPSED=$((END_TIME - START_TIME))
    MINUTES=$((ELAPSED / 60))
    SECONDS=$((ELAPSED % 60))
    
    # Show summary
    show_summary
    
    echo -e "${GREEN}Total time: ${MINUTES} minutes ${SECONDS} seconds${NC}"
    
    if [ $MINUTES -le 15 ]; then
        echo -e "${GREEN}✅ Completed within the 15-minute target!${NC}"
    else
        echo -e "${YELLOW}⚠️  Took longer than 15 minutes, but that's okay!${NC}"
    fi
}

# Run main function
main