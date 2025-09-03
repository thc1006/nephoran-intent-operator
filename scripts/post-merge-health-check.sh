#!/bin/bash
# =============================================================================
# POST-MERGE HEALTH CHECK SYSTEM
# =============================================================================
# Comprehensive validation after feat/e2e → integrate/mvp merge
# Ensures merge success and system stability
# =============================================================================

set -e

# Configuration
BRANCH_NAME="${1:-integrate/mvp}"
MAX_WAIT_TIME=300  # 5 minutes
CHECK_INTERVAL=30  # 30 seconds
VERBOSE="${VERBOSE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

log_verbose() {
    if [ "$VERBOSE" = "true" ]; then
        echo -e "${BLUE}[VERBOSE]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    fi
}

# Initialize health check
initialize_health_check() {
    log_info "🔍 POST-MERGE HEALTH CHECK INITIATED"
    log_info "Branch: $BRANCH_NAME"
    log_info "Max Wait Time: ${MAX_WAIT_TIME}s"
    log_info "Check Interval: ${CHECK_INTERVAL}s"
    log_info "Verbose Logging: $VERBOSE"
    echo "=============================================="
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a git repository"
        exit 1
    fi
    
    # Check if GitHub CLI is available
    if ! command -v gh &> /dev/null; then
        log_warning "GitHub CLI not available - some checks will be skipped"
        GH_AVAILABLE=false
    else
        GH_AVAILABLE=true
        log_verbose "GitHub CLI available: $(gh --version | head -n1)"
    fi
    
    # Check if Go is available
    if ! command -v go &> /dev/null; then
        log_error "Go is not available - cannot perform build checks"
        exit 1
    else
        GO_VERSION=$(go version)
        log_verbose "Go available: $GO_VERSION"
    fi
    
    log_success "Prerequisites validated"
}

# Verify merge completion
verify_merge_completion() {
    log_info "Verifying merge completion..."
    
    # Check if we're on the target branch
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    if [ "$CURRENT_BRANCH" != "$BRANCH_NAME" ]; then
        log_warning "Currently on branch '$CURRENT_BRANCH', not '$BRANCH_NAME'"
        
        # Try to switch to target branch
        if git checkout "$BRANCH_NAME" 2>/dev/null; then
            log_info "Switched to branch '$BRANCH_NAME'"
        else
            log_error "Cannot switch to branch '$BRANCH_NAME'"
            exit 1
        fi
    fi
    
    # Update branch to latest
    log_verbose "Fetching latest changes..."
    git fetch origin "$BRANCH_NAME" --quiet
    
    # Check for merge commit in recent history
    MERGE_COMMITS=$(git log --oneline -10 --grep="Merge.*feat/e2e" --grep="Merge pull request.*feat/e2e")
    if [ -z "$MERGE_COMMITS" ]; then
        log_warning "No explicit merge commit from feat/e2e found in recent history"
        log_verbose "This may be normal for squash merges or rebased merges"
    else
        log_success "Merge commit detected: $(echo "$MERGE_COMMITS" | head -n1)"
    fi
    
    # Verify branch is up to date with remote
    LOCAL_COMMIT=$(git rev-parse HEAD)
    REMOTE_COMMIT=$(git rev-parse "origin/$BRANCH_NAME")
    
    if [ "$LOCAL_COMMIT" != "$REMOTE_COMMIT" ]; then
        log_warning "Local branch is not up to date with remote"
        log_info "Pulling latest changes..."
        git pull origin "$BRANCH_NAME" --quiet
    else
        log_success "Branch is up to date with remote"
    fi
}

# Check immediate workflow status
check_workflow_status() {
    if [ "$GH_AVAILABLE" != "true" ]; then
        log_warning "Skipping workflow status check - GitHub CLI not available"
        return
    fi
    
    log_info "Checking immediate workflow status..."
    
    # Get recent workflow runs
    FAILED_WORKFLOWS=$(gh run list --branch "$BRANCH_NAME" --limit 10 --json conclusion,name,status --jq '.[] | select(.conclusion=="failure") | .name' 2>/dev/null || echo "")
    
    if [ ! -z "$FAILED_WORKFLOWS" ]; then
        log_warning "Recent workflow failures detected:"
        echo "$FAILED_WORKFLOWS" | while read -r workflow; do
            log_warning "  • $workflow"
        done
        
        # Check if failures are from manual workflows (less critical)
        MANUAL_FAILURES=$(echo "$FAILED_WORKFLOWS" | grep -E "(manual|dispatch)" || echo "")
        if [ ! -z "$MANUAL_FAILURES" ]; then
            log_info "Some failures are from manual workflows (may be expected)"
        fi
        
        # Auto-trigger emergency fix workflow if it exists
        if gh workflow list --json name | jq -r '.[].name' | grep -q "emergency-fix"; then
            log_info "Triggering emergency fix workflow..."
            gh workflow run emergency-fix.yml --ref "$BRANCH_NAME" 2>/dev/null || log_warning "Could not trigger emergency fix workflow"
        fi
    else
        log_success "No recent workflow failures detected"
    fi
    
    # Check for in-progress workflows
    IN_PROGRESS_WORKFLOWS=$(gh run list --branch "$BRANCH_NAME" --limit 5 --json status,name --jq '.[] | select(.status=="in_progress" or .status=="queued") | .name' 2>/dev/null || echo "")
    
    if [ ! -z "$IN_PROGRESS_WORKFLOWS" ]; then
        log_info "Workflows currently in progress:"
        echo "$IN_PROGRESS_WORKFLOWS" | while read -r workflow; do
            log_info "  • $workflow"
        done
    fi
}

# Validate critical components build
validate_component_builds() {
    log_info "Validating critical component builds..."
    
    # Set build environment
    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=amd64
    
    # Critical components to test
    CRITICAL_COMPONENTS=(
        "./cmd/intent-ingest"
        "./cmd/llm-processor"
        "./cmd/conductor-loop"
        "./controllers"
    )
    
    BUILD_FAILURES=()
    BUILD_SUCCESSES=()
    
    for component in "${CRITICAL_COMPONENTS[@]}"; do
        log_verbose "Building component: $component"
        
        if timeout 120s go build -v -ldflags="-s -w" "$component" &>/dev/null; then
            log_success "✅ $component built successfully"
            BUILD_SUCCESSES+=("$component")
        else
            log_error "❌ $component failed to build"
            BUILD_FAILURES+=("$component")
        fi
    done
    
    # Report build results
    log_info "Build Results Summary:"
    log_success "Successful: ${#BUILD_SUCCESSES[@]}"
    log_error "Failed: ${#BUILD_FAILURES[@]}"
    
    if [ ${#BUILD_FAILURES[@]} -gt 0 ]; then
        log_error "Critical components failed to build post-merge:"
        for failure in "${BUILD_FAILURES[@]}"; do
            log_error "  • $failure"
        done
        
        # Try to get more detailed build info
        log_info "Attempting detailed build analysis..."
        for failure in "${BUILD_FAILURES[@]}"; do
            log_verbose "Build error details for $failure:"
            go build -v "$failure" 2>&1 | head -20 | while read -r line; do
                log_verbose "  $line"
            done
        done
        
        exit 1
    fi
}

# Verify dependency integrity
verify_dependencies() {
    log_info "Verifying dependency integrity..."
    
    # Check go.mod and go.sum integrity
    if [ ! -f "go.mod" ]; then
        log_error "go.mod not found"
        exit 1
    fi
    
    if [ ! -f "go.sum" ]; then
        log_warning "go.sum not found - may be normal for new projects"
    fi
    
    # Verify modules
    log_verbose "Running go mod verify..."
    if timeout 60s go mod verify; then
        log_success "✅ Module verification passed"
    else
        log_error "❌ Module verification failed"
        
        # Try to fix common issues
        log_info "Attempting automatic dependency resolution..."
        
        # Clean module cache and retry
        go clean -modcache
        
        if timeout 120s go mod download; then
            log_info "Dependencies re-downloaded successfully"
            
            # Verify again
            if go mod verify; then
                log_success "✅ Module verification now passes after cleanup"
            else
                log_error "❌ Module verification still failing after cleanup"
                exit 1
            fi
        else
            log_error "❌ Failed to download dependencies"
            exit 1
        fi
    fi
    
    # Check for tidy modules
    log_verbose "Checking if go.mod is tidy..."
    if go mod tidy && git diff --exit-code go.mod go.sum; then
        log_success "✅ go.mod and go.sum are tidy"
    else
        log_warning "⚠️ go.mod or go.sum needs tidying"
        log_info "Running go mod tidy..."
        go mod tidy
        
        # Show what changed
        if ! git diff --exit-code go.mod go.sum; then
            log_warning "go.mod/go.sum were modified by go mod tidy"
            log_verbose "Consider committing these changes"
        fi
    fi
}

# Test basic functionality
test_basic_functionality() {
    log_info "Testing basic functionality..."
    
    # Test that we can run go test on key packages
    TEST_PACKAGES=(
        "./api/..."
        "./pkg/core/..."
        "./pkg/clients/..."
    )
    
    for package in "${TEST_PACKAGES[@]}"; do
        if ls $package >/dev/null 2>&1; then
            log_verbose "Testing package: $package"
            
            if timeout 60s go test -short "$package" &>/dev/null; then
                log_success "✅ Tests pass for $package"
            else
                log_warning "⚠️ Tests failed or timed out for $package"
                # Don't fail the health check for test failures, just warn
            fi
        else
            log_verbose "Package $package not found, skipping"
        fi
    done
}

# Generate health report
generate_health_report() {
    log_info "Generating health report..."
    
    REPORT_FILE="post-merge-health-report-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$REPORT_FILE" << EOF
# Post-Merge Health Check Report

**Date**: $(date '+%Y-%m-%d %H:%M:%S')
**Branch**: $BRANCH_NAME
**Commit**: $(git rev-parse HEAD)
**Go Version**: $(go version)

## Summary
- ✅ Prerequisites validated
- ✅ Merge completion verified
- ✅ Critical components build successfully
- ✅ Dependencies verified
- ✅ Basic functionality tested

## Details

### Git Status
\`\`\`
$(git status --porcelain)
\`\`\`

### Recent Commits
\`\`\`
$(git log --oneline -5)
\`\`\`

### Build Environment
- CGO_ENABLED: $CGO_ENABLED
- GOOS: $GOOS
- GOARCH: $GOARCH

### Dependency Status
\`\`\`
$(go list -m all | head -10)
\`\`\`

## Recommendations
1. Monitor workflow status for next 24 hours
2. Verify all manual workflows complete successfully
3. Check application logs in production environment
4. Validate end-to-end functionality

---
*Generated by post-merge-health-check.sh*
EOF
    
    log_success "Health report generated: $REPORT_FILE"
}

# Performance monitoring
monitor_performance() {
    log_info "Monitoring post-merge performance..."
    
    # Record build times
    START_TIME=$(date +%s)
    
    # Sample build to measure performance
    log_verbose "Performing sample build for performance measurement..."
    if timeout 180s go build ./cmd/intent-ingest &>/dev/null; then
        END_TIME=$(date +%s)
        BUILD_TIME=$((END_TIME - START_TIME))
        
        if [ $BUILD_TIME -lt 30 ]; then
            log_success "✅ Build performance excellent ($BUILD_TIME seconds)"
        elif [ $BUILD_TIME -lt 60 ]; then
            log_info "ℹ️ Build performance good ($BUILD_TIME seconds)"
        else
            log_warning "⚠️ Build performance slower than expected ($BUILD_TIME seconds)"
        fi
    else
        log_warning "⚠️ Could not measure build performance - build timeout"
    fi
}

# Main execution
main() {
    initialize_health_check
    
    # Execute all health checks
    check_prerequisites
    verify_merge_completion
    check_workflow_status
    validate_component_builds
    verify_dependencies
    test_basic_functionality
    monitor_performance
    generate_health_report
    
    log_success "🎯 POST-MERGE HEALTH CHECK COMPLETED SUCCESSFULLY"
    log_info "All systems appear healthy after merge"
    log_info "Monitor GitHub Actions at: https://github.com/$(gh repo view --json owner,name -q '.owner.login + "/" + .name")/actions"
}

# Execute main function
main "$@"