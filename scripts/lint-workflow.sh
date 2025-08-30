#!/bin/bash
# Automated workflow for lint fixing with fast feedback
# Usage: ./lint-workflow.sh [workflow-type] [options]

set -euo pipefail

WORKFLOW_TYPE="${1:-full}"
TARGET_PATH="${2:-.}"
LINTER_NAME="${3:-}"

# Configuration
TEMP_DIR="/tmp/lint-workflow-$$"
BACKUP_DIR="$TEMP_DIR/backup"
MAX_ITERATIONS=5
BATCH_SIZE=10

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
BOLD='\033[1m'
NC='\033[0m'

usage() {
    echo "Usage: $0 [workflow-type] [target-path] [linter-name]"
    echo ""
    echo "Workflow types:"
    echo "  full      - Complete lint fixing workflow (default)"
    echo "  quick     - Quick iteration for specific issues"
    echo "  targeted  - Target specific linter only"
    echo "  batch     - Process files in batches"
    echo "  safe      - Safe mode with backups and rollback"
    echo ""
    echo "Examples:"
    echo "  $0 full ./pkg"
    echo "  $0 targeted ./internal revive"
    echo "  $0 quick ./cmd/conductor-loop"
    echo "  $0 batch ./pkg/controllers"
    exit 1
}

log() {
    echo -e "${GRAY}[$(date +'%H:%M:%S')] $1${NC}" >&2
}

log_info() {
    echo -e "${CYAN}[INFO] $1${NC}" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}" >&2
}

log_error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}" >&2
}

cleanup() {
    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
}
trap cleanup EXIT

# Workflow implementations
workflow_full() {
    log_info "Starting full lint fixing workflow"
    
    mkdir -p "$TEMP_DIR"
    
    # Step 1: Baseline check
    log_info "Step 1: Establishing baseline"
    if ! golangci-lint run --disable-all --enable=revive --timeout=30s "$TARGET_PATH" > "$TEMP_DIR/baseline.log" 2>&1; then
        log_warning "Baseline check found issues"
    else
        log_success "No issues found - code is already clean!"
        return 0
    fi
    
    # Step 2: Run all linters individually to identify problem areas
    log_info "Step 2: Identifying problem linters"
    local linters=(revive staticcheck govet ineffassign errcheck gocritic misspell unparam unconvert prealloc gosec)
    local problem_linters=()
    
    for linter in "${linters[@]}"; do
        if ! golangci-lint run --disable-all --enable="$linter" --timeout=30s "$TARGET_PATH" > "$TEMP_DIR/${linter}.log" 2>&1; then
            problem_linters+=("$linter")
            log_warning "  $linter: has issues"
        else
            log_success "  $linter: clean"
        fi
    done
    
    if [[ ${#problem_linters[@]} -eq 0 ]]; then
        log_success "All individual linters pass!"
        return 0
    fi
    
    # Step 3: Fix linters one by one
    log_info "Step 3: Fixing linters individually"
    for linter in "${problem_linters[@]}"; do
        log_info "Fixing $linter issues..."
        
        local iteration=1
        while [[ $iteration -le $MAX_ITERATIONS ]]; do
            log "  Iteration $iteration for $linter"
            
            # Try auto-fix first
            if golangci-lint run --disable-all --enable="$linter" --fix --timeout=60s "$TARGET_PATH" > "$TEMP_DIR/${linter}_fix_${iteration}.log" 2>&1; then
                log_success "  $linter fixed successfully"
                break
            else
                log_warning "  $linter iteration $iteration failed"
                if [[ $iteration -eq $MAX_ITERATIONS ]]; then
                    log_error "  $linter could not be auto-fixed after $MAX_ITERATIONS attempts"
                fi
            fi
            
            ((iteration++))
        done
        
        # Verify fix
        if golangci-lint run --disable-all --enable="$linter" --timeout=30s "$TARGET_PATH" > "$TEMP_DIR/${linter}_verify.log" 2>&1; then
            log_success "  $linter verification passed"
        else
            log_error "  $linter verification failed - manual intervention needed"
        fi
    done
    
    # Step 4: Final verification
    log_info "Step 4: Final verification"
    if golangci-lint run --timeout=120s "$TARGET_PATH" > "$TEMP_DIR/final_check.log" 2>&1; then
        log_success "All linters pass! Workflow completed successfully."
        return 0
    else
        log_error "Some issues remain after automated fixing"
        log "Check detailed logs in: $TEMP_DIR/"
        return 1
    fi
}

workflow_quick() {
    log_info "Starting quick iteration workflow"
    
    local iteration=1
    while [[ $iteration -le 3 ]]; do
        log_info "Quick iteration $iteration"
        
        if golangci-lint run --fix --timeout=60s "$TARGET_PATH" > "$TEMP_DIR/quick_${iteration}.log" 2>&1; then
            log_success "Quick fix successful on iteration $iteration"
            return 0
        fi
        
        ((iteration++))
        sleep 1
    done
    
    log_error "Quick workflow failed after 3 iterations"
    return 1
}

workflow_targeted() {
    if [[ -z "$LINTER_NAME" ]]; then
        log_error "Targeted workflow requires a linter name"
        return 1
    fi
    
    log_info "Starting targeted workflow for $LINTER_NAME"
    
    local iteration=1
    while [[ $iteration -le $MAX_ITERATIONS ]]; do
        log_info "Targeted iteration $iteration for $LINTER_NAME"
        
        if golangci-lint run --disable-all --enable="$LINTER_NAME" --fix --timeout=60s "$TARGET_PATH" > "$TEMP_DIR/targeted_${iteration}.log" 2>&1; then
            log_success "Targeted fix for $LINTER_NAME successful on iteration $iteration"
            return 0
        fi
        
        log_warning "Iteration $iteration failed, trying again..."
        ((iteration++))
        sleep 2
    done
    
    log_error "Targeted workflow for $LINTER_NAME failed after $MAX_ITERATIONS iterations"
    return 1
}

workflow_batch() {
    log_info "Starting batch processing workflow"
    
    # Find Go files
    mapfile -t go_files < <(find "$TARGET_PATH" -name "*.go" -type f | grep -v -E '_test\.go$|_mock\.go$|\.mock\.go$|_generated\.go$|\.pb\.go$|zz_generated\.')
    
    if [[ ${#go_files[@]} -eq 0 ]]; then
        log_warning "No Go files found in $TARGET_PATH"
        return 0
    fi
    
    log_info "Found ${#go_files[@]} Go files to process"
    
    # Process in batches
    local batch_num=1
    local total_batches=$(( (${#go_files[@]} + BATCH_SIZE - 1) / BATCH_SIZE ))
    
    for ((i=0; i<${#go_files[@]}; i+=BATCH_SIZE)); do
        local batch=("${go_files[@]:$i:$BATCH_SIZE}")
        log_info "Processing batch $batch_num of $total_batches (${#batch[@]} files)"
        
        # Create temporary batch file list
        printf '%s\n' "${batch[@]}" > "$TEMP_DIR/batch_${batch_num}.txt"
        
        # Run linter on batch
        if xargs -a "$TEMP_DIR/batch_${batch_num}.txt" golangci-lint run --fix --timeout=60s > "$TEMP_DIR/batch_${batch_num}.log" 2>&1; then
            log_success "  Batch $batch_num completed successfully"
        else
            log_warning "  Batch $batch_num had issues (continuing with next batch)"
        fi
        
        ((batch_num++))
    done
    
    # Final verification
    log_info "Running final verification on all processed files"
    if golangci-lint run --timeout=120s "$TARGET_PATH" > "$TEMP_DIR/batch_final.log" 2>&1; then
        log_success "Batch processing completed successfully"
        return 0
    else
        log_error "Some issues remain after batch processing"
        return 1
    fi
}

workflow_safe() {
    log_info "Starting safe workflow with backups"
    
    # Create backup
    mkdir -p "$BACKUP_DIR"
    log_info "Creating backup in $BACKUP_DIR"
    
    if ! cp -r "$TARGET_PATH" "$BACKUP_DIR/"; then
        log_error "Failed to create backup"
        return 1
    fi
    
    # Run full workflow
    if workflow_full; then
        log_success "Safe workflow completed successfully"
        return 0
    else
        log_error "Safe workflow failed, restoring from backup"
        
        # Restore from backup
        if rm -rf "${TARGET_PATH:?}" && cp -r "$BACKUP_DIR/$(basename "$TARGET_PATH")" "$TARGET_PATH"; then
            log_success "Successfully restored from backup"
        else
            log_error "Failed to restore from backup!"
        fi
        
        return 1
    fi
}

# Main execution
mkdir -p "$TEMP_DIR"

case "$WORKFLOW_TYPE" in
    full)
        workflow_full
        ;;
    quick)
        workflow_quick
        ;;
    targeted)
        workflow_targeted
        ;;
    batch)
        workflow_batch
        ;;
    safe)
        workflow_safe
        ;;
    *)
        log_error "Unknown workflow type: $WORKFLOW_TYPE"
        usage
        ;;
esac

exit_code=$?

# Generate summary report
log_info "Generating workflow summary"
{
    echo "Lint Workflow Summary"
    echo "===================="
    echo "Workflow Type: $WORKFLOW_TYPE"
    echo "Target Path: $TARGET_PATH"
    echo "Linter: ${LINTER_NAME:-all}"
    echo "Exit Code: $exit_code"
    echo "Timestamp: $(date)"
    echo ""
    echo "Log files available in: $TEMP_DIR"
    
    if [[ -d "$TEMP_DIR" ]]; then
        echo ""
        echo "Generated files:"
        find "$TEMP_DIR" -name "*.log" -exec basename {} \; | sort
    fi
} > "$TEMP_DIR/summary.txt"

# Copy summary to project directory if successful
if [[ $exit_code -eq 0 ]]; then
    mkdir -p "./test-results"
    cp "$TEMP_DIR/summary.txt" "./test-results/lint-workflow-$(date +%Y%m%d_%H%M%S).txt"
fi

exit $exit_code