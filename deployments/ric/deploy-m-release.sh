#!/bin/bash
################################################################################
# O-RAN SC RIC M Release Deployment Script with Enhanced Logging
# Environment: K8s 1.35.1, Single-Node, Ubuntu 22.04
# Created: 2026-02-15
################################################################################

set -euo pipefail

# ============================================================================
# Logger Configuration
# ============================================================================
LOG_DIR="/home/thc1006/dev/nephoran-intent-operator/deployments/ric/logs"
LOG_FILE="${LOG_DIR}/ric-deploy-$(date +%Y%m%d-%H%M%S).log"
SUMMARY_FILE="${LOG_DIR}/deployment-summary.txt"

mkdir -p "${LOG_DIR}"

# Color codes for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging functions
log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # Write to log file
    echo "[${timestamp}] [${level}] ${message}" >> "${LOG_FILE}"

    # Write to console with colors
    case "${level}" in
        INFO)
            echo -e "${BLUE}[INFO]${NC} ${message}"
            ;;
        SUCCESS)
            echo -e "${GREEN}[SUCCESS]${NC} ${message}"
            ;;
        WARN)
            echo -e "${YELLOW}[WARN]${NC} ${message}"
            ;;
        ERROR)
            echo -e "${RED}[ERROR]${NC} ${message}"
            ;;
        STEP)
            echo -e "${MAGENTA}[STEP]${NC} ${message}"
            ;;
        DEBUG)
            echo -e "${CYAN}[DEBUG]${NC} ${message}"
            ;;
        *)
            echo "[${level}] ${message}"
            ;;
    esac
}

log_separator() {
    local char="${1:--}"
    local line=$(printf '%*s' 80 | tr ' ' "${char}")
    log INFO "${line}"
}

log_step() {
    log_separator "="
    log STEP "$1"
    log_separator "="
}

log_command() {
    local cmd="$@"
    log DEBUG "Executing: ${cmd}"

    # Execute command and capture output
    if output=$($cmd 2>&1); then
        log DEBUG "Command succeeded"
        echo "${output}" >> "${LOG_FILE}"
        return 0
    else
        local exit_code=$?
        log ERROR "Command failed with exit code ${exit_code}"
        echo "${output}" >> "${LOG_FILE}"
        return ${exit_code}
    fi
}

# ============================================================================
# Error Handling
# ============================================================================
cleanup_on_error() {
    log ERROR "Deployment failed! Check logs at: ${LOG_FILE}"
    log INFO "Collecting diagnostic information..."

    {
        echo "=== Pod Status ==="
        kubectl get pods -A
        echo ""
        echo "=== Recent Events ==="
        kubectl get events -A --sort-by='.lastTimestamp' | tail -50
        echo ""
        echo "=== Failed Pods Logs ==="
        for ns in ricplt ricinfra ricxapp ricaux; do
            kubectl get pods -n ${ns} 2>/dev/null | grep -v Running | awk 'NR>1 {print $1}' | while read pod; do
                echo "--- Logs for ${ns}/${pod} ---"
                kubectl logs -n ${ns} ${pod} --tail=100 2>&1 || echo "Could not get logs"
            done
        done
    } >> "${LOG_FILE}"

    exit 1
}

trap cleanup_on_error ERR

# ============================================================================
# Pre-flight Checks
# ============================================================================
preflight_checks() {
    log_step "Running Pre-flight Checks"

    # Check Kubernetes
    log INFO "Checking Kubernetes connection..."
    if ! kubectl cluster-info &>/dev/null; then
        log ERROR "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    local k8s_version=$(kubectl version --short 2>/dev/null | grep "Server Version" | awk '{print $3}')
    log SUCCESS "Kubernetes version: ${k8s_version}"

    # Check Helm
    log INFO "Checking Helm installation..."
    if ! command -v helm &>/dev/null; then
        log ERROR "Helm is not installed"
        exit 1
    fi

    local helm_version=$(helm version --short)
    log SUCCESS "Helm version: ${helm_version}"

    # Check disk space
    log INFO "Checking disk space..."
    local available_space=$(df -h /home/thc1006 | awk 'NR==2 {print $4}')
    log INFO "Available disk space: ${available_space}"

    # Check if ric-common is installed
    log INFO "Checking ric-common Helm package..."
    if helm show chart ric-common &>/dev/null; then
        log SUCCESS "ric-common package found"
    else
        log WARN "ric-common package not found, will install"
    fi

    log SUCCESS "Pre-flight checks completed"
}

# ============================================================================
# Namespace Setup
# ============================================================================
setup_namespaces() {
    log_step "Setting up Namespaces"

    local namespaces=("ricplt" "ricxapp" "ricinfra" "ricaux")

    for ns in "${namespaces[@]}"; do
        if kubectl get namespace ${ns} &>/dev/null; then
            log INFO "Namespace ${ns} already exists"
        else
            log INFO "Creating namespace ${ns}..."
            kubectl create namespace ${ns}
            log SUCCESS "Namespace ${ns} created"
        fi
    done
}

# ============================================================================
# ric-common Installation
# ============================================================================
install_ric_common() {
    log_step "Installing ric-common Helm Package"

    local ric_common_dir="/home/thc1006/dev/nephoran-intent-operator/deployments/ric/ric-dep/ric-common/Common-Template/helm/ric-common"
    local helm_dir="/home/thc1006/dev/nephoran-intent-operator/deployments/ric/ric-dep/helm"

    if [ ! -d "${ric_common_dir}" ]; then
        log ERROR "ric-common directory not found at ${ric_common_dir}"
        exit 1
    fi

    cd "${ric_common_dir}"
    log INFO "Building ric-common Helm chart..."

    # Package the chart
    if helm package .; then
        log SUCCESS "ric-common packaged successfully"

        local package=$(ls -t ric-common-*.tgz 2>/dev/null | head -1)
        if [ -n "${package}" ]; then
            log INFO "Copying ${package} to Helm charts directory..."

            # For Helm 4, we copy the packaged chart to each component's charts/ directory
            for component in appmgr dbaas e2mgr e2term rtmgr submgr a1mediator alarmmanager jaegeradapter o1mediator vespamgr xapp-onboarder; do
                if [ -d "${helm_dir}/${component}" ]; then
                    mkdir -p "${helm_dir}/${component}/charts"
                    cp "${package}" "${helm_dir}/${component}/charts/" 2>/dev/null || true
                    log INFO "Copied to ${component}/charts/"
                fi
            done

            log SUCCESS "ric-common distributed to component charts"
        else
            log ERROR "No ric-common package file found"
            exit 1
        fi
    else
        log ERROR "Failed to package ric-common"
        exit 1
    fi
}

# ============================================================================
# Recipe Preparation
# ============================================================================
prepare_recipe() {
    log_step "Preparing M Release Recipe"

    local recipe_template="/home/thc1006/dev/nephoran-intent-operator/deployments/ric/ric-dep/RECIPE_EXAMPLE/example_recipe_oran_m_release.yaml"
    local recipe_file="/home/thc1006/dev/nephoran-intent-operator/deployments/ric/recipe-m-release-k135.yaml"

    log INFO "Copying M Release recipe template..."
    cp "${recipe_template}" "${recipe_file}"

    # Get node IP
    local node_ip=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
    log INFO "Detected node IP: ${node_ip}"

    # Update recipe with correct IPs
    log INFO "Updating recipe with node IP..."
    sed -i "s/ricip: \"10.0.0.1\"/ricip: \"${node_ip}\"/" "${recipe_file}"
    sed -i "s/auxip: \"10.0.0.1\"/auxip: \"${node_ip}\"/" "${recipe_file}"

    log SUCCESS "Recipe prepared at: ${recipe_file}"

    # Show recipe summary
    log INFO "Recipe summary:"
    log INFO "  - e2mgr: $(grep -A2 'e2mgr:' ${recipe_file} | grep 'tag:' | awk '{print $2}')"
    log INFO "  - rtmgr: $(grep -A4 'rtmgr:' ${recipe_file} | grep 'tag:' | awk '{print $2}')"
    log INFO "  - appmgr: $(grep -A6 'appmgr:' ${recipe_file} | grep 'tag:' | head -1 | awk '{print $2}')"
    log INFO "  - dbaas: $(grep -A2 'dbaas:' ${recipe_file} | grep 'tag:' | awk '{print $2}')"
}

# ============================================================================
# RIC Platform Deployment
# ============================================================================
deploy_ric_platform() {
    log_step "Deploying RIC Platform (M Release)"

    local recipe_file="/home/thc1006/dev/nephoran-intent-operator/deployments/ric/recipe-m-release-k135.yaml"
    local install_script="/home/thc1006/dev/nephoran-intent-operator/deployments/ric/ric-dep/bin/install"

    if [ ! -f "${recipe_file}" ]; then
        log ERROR "Recipe file not found: ${recipe_file}"
        exit 1
    fi

    if [ ! -x "${install_script}" ]; then
        log ERROR "Install script not found or not executable: ${install_script}"
        exit 1
    fi

    cd /home/thc1006/dev/nephoran-intent-operator/deployments/ric/ric-dep

    log INFO "Starting RIC platform deployment..."
    log INFO "This may take 10-15 minutes..."

    # Run install script with logging
    if "${install_script}" -f "${recipe_file}" 2>&1 | tee -a "${LOG_FILE}"; then
        log SUCCESS "RIC platform deployment script completed"
    else
        log ERROR "RIC platform deployment script failed"
        exit 1
    fi
}

# ============================================================================
# Deployment Verification
# ============================================================================
wait_for_pods() {
    local namespace=$1
    local timeout=${2:-600}  # Default 10 minutes
    local interval=10
    local elapsed=0

    log INFO "Waiting for pods in namespace ${namespace} (timeout: ${timeout}s)..."

    while [ ${elapsed} -lt ${timeout} ]; do
        local total=$(kubectl get pods -n ${namespace} --no-headers 2>/dev/null | wc -l)
        local running=$(kubectl get pods -n ${namespace} --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l)
        local pending=$(kubectl get pods -n ${namespace} --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l)
        local failed=$(kubectl get pods -n ${namespace} --field-selector=status.phase=Failed --no-headers 2>/dev/null | wc -l)

        log INFO "[${namespace}] Total: ${total}, Running: ${running}, Pending: ${pending}, Failed: ${failed}"

        if [ ${total} -gt 0 ] && [ ${running} -eq ${total} ]; then
            log SUCCESS "All pods in ${namespace} are running"
            return 0
        fi

        if [ ${failed} -gt 0 ]; then
            log WARN "Some pods in ${namespace} are in Failed state"
            kubectl get pods -n ${namespace} --field-selector=status.phase=Failed
        fi

        sleep ${interval}
        elapsed=$((elapsed + interval))
    done

    log ERROR "Timeout waiting for pods in ${namespace}"
    kubectl get pods -n ${namespace}
    return 1
}

verify_deployment() {
    log_step "Verifying RIC Deployment"

    local namespaces=("ricinfra" "ricplt")

    for ns in "${namespaces[@]}"; do
        if kubectl get namespace ${ns} &>/dev/null; then
            wait_for_pods ${ns} 600
        else
            log WARN "Namespace ${ns} not found"
        fi
    done

    log INFO "Collecting final deployment status..."

    {
        echo "=== Final Pod Status ==="
        kubectl get pods -n ricinfra
        echo ""
        kubectl get pods -n ricplt
        echo ""
        echo "=== Services ==="
        kubectl get svc -n ricplt
        echo ""
        echo "=== ConfigMaps ==="
        kubectl get cm -n ricplt
    } | tee -a "${LOG_FILE}"

    log SUCCESS "Deployment verification completed"
}

# ============================================================================
# Summary Report
# ============================================================================
generate_summary() {
    log_step "Generating Deployment Summary"

    local end_time=$(date '+%Y-%m-%d %H:%M:%S')

    cat > "${SUMMARY_FILE}" << EOF
O-RAN SC RIC M Release Deployment Summary
==========================================
Deployment Date: ${end_time}
Environment: K8s 1.35.1, Single-Node
Release: O-RAN SC M Release (2025-12-20)

Component Versions:
- e2mgr: 6.0.7
- rtmgr: 0.9.7
- appmgr: 0.5.9
- submgr: 0.10.3
- e2term: 6.0.7
- dbaas: 0.6.5
- a1mediator: 3.2.3

Namespaces:
$(kubectl get ns | grep ric)

Pods Status:
=== ricinfra ===
$(kubectl get pods -n ricinfra 2>/dev/null || echo "Namespace not found")

=== ricplt ===
$(kubectl get pods -n ricplt 2>/dev/null || echo "Namespace not found")

Services:
$(kubectl get svc -n ricplt 2>/dev/null || echo "No services found")

Logs Location: ${LOG_FILE}

Next Steps:
1. Check pod logs if any pods are not Running
2. Verify E2 termination endpoint
3. Test A1 mediator connectivity
4. Deploy xApps if needed

EOF

    cat "${SUMMARY_FILE}"
    log SUCCESS "Summary report saved to: ${SUMMARY_FILE}"
}

# ============================================================================
# Main Execution
# ============================================================================
main() {
    log_step "O-RAN SC RIC M Release Deployment Starting"
    log INFO "Log file: ${LOG_FILE}"
    log INFO "Environment: K8s 1.35.1, Single-Node, Ubuntu 22.04"
    log INFO "Release: O-RAN SC M Release (2025-12-20)"
    log_separator

    local start_time=$(date +%s)

    # Execute deployment steps
    preflight_checks
    setup_namespaces
    install_ric_common
    prepare_recipe
    deploy_ric_platform
    verify_deployment
    generate_summary

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    local duration_min=$((duration / 60))
    local duration_sec=$((duration % 60))

    log_separator "="
    log SUCCESS "RIC Deployment Completed Successfully!"
    log INFO "Total time: ${duration_min}m ${duration_sec}s"
    log INFO "Log file: ${LOG_FILE}"
    log INFO "Summary: ${SUMMARY_FILE}"
    log_separator "="
}

# Run main function
main "$@"
