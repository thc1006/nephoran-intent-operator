#!/bin/bash
# =============================================================================
# Migration Script: Consolidate Fragmented CI/CD Workflows
# =============================================================================
# This script helps migrate from 37 fragmented workflows to the single
# consolidated pipeline, providing safe migration with rollback capability
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKFLOWS_DIR="$REPO_ROOT/.github/workflows"
ARCHIVE_DIR="$WORKFLOWS_DIR/archive"
BACKUP_DIR="$WORKFLOWS_DIR/backup-$(date +%Y%m%d-%H%M%S)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Main consolidated workflow file
CONSOLIDATED_WORKFLOW="nephoran-ci-2025-consolidated.yml"

# Fragmented workflows to be disabled/archived
FRAGMENTED_WORKFLOWS=(
    "branch-protection-setup.yml"
    "cache-recovery-system.yml"
    "ci-2025.yml"
    "ci-optimized.yml"
    "ci-production.yml"
    "container-build-2025.yml"
    "ci-ultra-optimized-2025.yml"
    "debug-ghcr-auth.yml"
    "ci-stability-orchestrator.yml"
    "ci-timeout-fix.yml"
    "dev-fast-fixed.yml"
    "emergency-disable.yml"
    "dev-fast.yml"
    "final-integration-validation.yml"
    "emergency-merge.yml"
    "go-module-cache.yml"
    "kubernetes-operator-deployment.yml"
    "pr-ci-fast.yml"
    "parallel-tests.yml"
    "oran-telecom-validation.yml"
    "nephoran-master-orchestrator.yml"
    "main-ci-optimized.yml"
    "main-ci-optimized-2025.yml"
    "security-enhanced-ci.yml"
    "security-scan-config.yml"
    "production-ci.yml"
    "pr-validation.yml"
    "security-scan.yml"
    "telecom-security-compliance.yml"
    "security-scan-ultra-reliable.yml"
    "security-scan-optimized.yml"
    "security-scan-optimized-2025.yml"
    "ubuntu-ci.yml"
    "timeout-management.yml"
    "ci-reliability-optimized.yml"
    "k8s-operator-ci-2025.yml"
)

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a git repository"
        exit 1
    fi
    
    # Check if workflows directory exists
    if [[ ! -d "$WORKFLOWS_DIR" ]]; then
        log_error "Workflows directory not found: $WORKFLOWS_DIR"
        exit 1
    fi
    
    # Check if consolidated workflow exists
    if [[ ! -f "$WORKFLOWS_DIR/$CONSOLIDATED_WORKFLOW" ]]; then
        log_error "Consolidated workflow not found: $WORKFLOWS_DIR/$CONSOLIDATED_WORKFLOW"
        log_info "Please ensure the consolidated pipeline file is in place first"
        exit 1
    fi
    
    # Check git status
    if ! git diff --quiet --exit-code; then
        log_warning "Working directory has uncommitted changes"
        log_info "It's recommended to commit changes before migration"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    log_success "Prerequisites check passed"
}

# Function to create backup
create_backup() {
    log_info "Creating backup of existing workflows..."
    
    mkdir -p "$BACKUP_DIR"
    
    # Copy all workflow files to backup
    cp "$WORKFLOWS_DIR"/*.yml "$BACKUP_DIR/" 2>/dev/null || true
    cp "$WORKFLOWS_DIR"/*.yaml "$BACKUP_DIR/" 2>/dev/null || true
    
    # Create backup manifest
    cat > "$BACKUP_DIR/BACKUP_MANIFEST.md" <<EOF
# Workflow Backup - $(date)

## Backup Information
- **Backup Date**: $(date)
- **Git Commit**: $(git rev-parse HEAD 2>/dev/null || echo "unknown")
- **Git Branch**: $(git branch --show-current 2>/dev/null || echo "unknown")
- **Migration Script**: migrate-to-consolidated-pipeline.sh

## Backed Up Files
$(ls -la "$BACKUP_DIR"/*.y*ml 2>/dev/null | wc -l || echo "0") workflow files backed up

## Restoration Instructions
To restore these workflows:
1. Copy files from this backup directory to .github/workflows/
2. Remove or disable the consolidated pipeline
3. Commit and push changes

## Backup Location
$BACKUP_DIR
EOF
    
    log_success "Backup created: $BACKUP_DIR"
}

# Function to analyze current workflows
analyze_workflows() {
    log_info "Analyzing current workflow configuration..."
    
    total_workflows=$(ls -1 "$WORKFLOWS_DIR"/*.yml "$WORKFLOWS_DIR"/*.yaml 2>/dev/null | wc -l || echo "0")
    fragmented_count=0
    
    echo ""
    echo "📊 Current Workflow Analysis"
    echo "============================"
    echo "Total workflow files: $total_workflows"
    echo ""
    echo "Fragmented workflows to be disabled:"
    
    for workflow in "${FRAGMENTED_WORKFLOWS[@]}"; do
        if [[ -f "$WORKFLOWS_DIR/$workflow" ]]; then
            echo "  ✓ $workflow (found)"
            ((fragmented_count++))
        else
            echo "  - $workflow (not found)"
        fi
    done
    
    echo ""
    echo "Summary:"
    echo "  - Fragmented workflows found: $fragmented_count"
    echo "  - Consolidated workflow: $([ -f "$WORKFLOWS_DIR/$CONSOLIDATED_WORKFLOW" ] && echo "ready" || echo "missing")"
    echo "  - Reduction: $fragmented_count → 1 workflow ($(( (fragmented_count - 1) * 100 / fragmented_count ))% reduction)"
    echo ""
}

# Function to disable fragmented workflows
disable_workflows() {
    log_info "Disabling fragmented workflows..."
    
    mkdir -p "$ARCHIVE_DIR"
    disabled_count=0
    
    for workflow in "${FRAGMENTED_WORKFLOWS[@]}"; do
        workflow_path="$WORKFLOWS_DIR/$workflow"
        archive_path="$ARCHIVE_DIR/$workflow"
        disabled_path="$WORKFLOWS_DIR/${workflow%.yml}.disabled.yml"
        
        if [[ -f "$workflow_path" ]]; then
            # Copy to archive
            cp "$workflow_path" "$archive_path"
            
            # Disable by renaming
            mv "$workflow_path" "$disabled_path"
            
            log_info "Disabled: $workflow → ${workflow%.yml}.disabled.yml"
            ((disabled_count++))
        fi
    done
    
    log_success "Disabled $disabled_count fragmented workflows"
}

# Function to validate consolidated workflow
validate_consolidated_workflow() {
    log_info "Validating consolidated workflow..."
    
    workflow_path="$WORKFLOWS_DIR/$CONSOLIDATED_WORKFLOW"
    
    # Basic YAML syntax validation
    if command -v python3 >/dev/null 2>&1; then
        if ! python3 -c "
import yaml
try:
    with open('$workflow_path', 'r') as f:
        yaml.safe_load(f)
    print('✓ YAML syntax valid')
except Exception as e:
    print(f'✗ YAML syntax error: {e}')
    exit(1)
" 2>/dev/null; then
            log_error "Consolidated workflow has YAML syntax errors"
            return 1
        fi
    else
        log_warning "Python3 not available - skipping YAML validation"
    fi
    
    # Check required sections
    required_sections=("name" "on" "jobs")
    for section in "${required_sections[@]}"; do
        if ! grep -q "^$section:" "$workflow_path"; then
            log_error "Missing required section: $section"
            return 1
        fi
    done
    
    # Check for security configurations
    if grep -q "permissions:" "$workflow_path"; then
        log_success "Security permissions configured"
    else
        log_warning "No explicit permissions found"
    fi
    
    # Check for concurrency control
    if grep -q "concurrency:" "$workflow_path"; then
        log_success "Concurrency control configured"
    else
        log_warning "No concurrency control found"
    fi
    
    log_success "Consolidated workflow validation passed"
}

# Function to create migration summary
create_migration_summary() {
    log_info "Creating migration summary..."
    
    summary_file="$REPO_ROOT/MIGRATION_SUMMARY_$(date +%Y%m%d_%H%M%S).md"
    
    cat > "$summary_file" <<EOF
# CI/CD Pipeline Migration Summary

## Migration Details
- **Date**: $(date)
- **Git Commit Before**: $(git rev-parse HEAD)
- **Git Branch**: $(git branch --show-current 2>/dev/null || echo "unknown")
- **Migration Script**: migrate-to-consolidated-pipeline.sh

## Changes Made

### ✅ Consolidated Pipeline
- **New Workflow**: \`.github/workflows/$CONSOLIDATED_WORKFLOW\`
- **Features**: 
  - 25-minute maximum timeouts with intelligent stage-level timeouts
  - Multi-layer intelligent caching with fallback strategies
  - 8-way parallel job execution with proper dependency management
  - Comprehensive security scanning and container builds
  - 4 build modes: fast, full, debug, security

### 📦 Archived Workflows
Fragmented workflows moved to: \`.github/workflows/archive/\`

$(for workflow in "${FRAGMENTED_WORKFLOWS[@]}"; do
    if [[ -f "$WORKFLOWS_DIR/${workflow%.yml}.disabled.yml" ]]; then
        echo "- $workflow → ${workflow%.yml}.disabled.yml"
    fi
done)

### 💾 Backup Location
Complete backup created at: \`$BACKUP_DIR\`

## Verification Steps

### 1. Test the New Pipeline
\`\`\`bash
# Push a test commit to trigger the pipeline
git commit --allow-empty -m "test: trigger consolidated pipeline"
git push
\`\`\`

### 2. Monitor Initial Runs
- Check GitHub Actions tab for pipeline execution
- Verify all build targets complete successfully
- Confirm security scans execute properly
- Validate artifacts are generated correctly

### 3. Performance Comparison
- **Before**: ~37 parallel workflows with potential conflicts
- **After**: 1 optimized workflow with intelligent scheduling
- **Expected**: 40% reduction in total CI time

## Rollback Instructions

If issues arise, restore the previous configuration:

\`\`\`bash
# 1. Disable consolidated workflow
mv .github/workflows/$CONSOLIDATED_WORKFLOW .github/workflows/$CONSOLIDATED_WORKFLOW.disabled

# 2. Restore fragmented workflows
cp $BACKUP_DIR/*.yml .github/workflows/

# 3. Commit and push
git add .github/workflows/
git commit -m "rollback: restore fragmented workflows"
git push
\`\`\`

## Next Steps

1. **Monitor Performance**: Track build times and success rates
2. **Update Documentation**: Update CI/CD references in project docs
3. **Team Training**: Familiarize team with new workflow parameters
4. **Cleanup**: After 30 days of stable operation, remove disabled workflows

## Support

For issues with the consolidated pipeline:
1. Check the troubleshooting guide in \`docs/ci-cd-consolidated-pipeline-2025.md\`
2. Review pipeline logs in GitHub Actions
3. Contact the maintainers with specific error details

---
*Migration completed by migrate-to-consolidated-pipeline.sh on $(date)*
EOF
    
    log_success "Migration summary created: $summary_file"
}

# Function to perform post-migration checks
post_migration_checks() {
    log_info "Performing post-migration checks..."
    
    # Check if consolidated workflow is the only active one
    active_workflows=$(find "$WORKFLOWS_DIR" -name "*.yml" -not -path "*/archive/*" | wc -l)
    if [[ $active_workflows -eq 1 ]]; then
        log_success "Only 1 active workflow remaining (consolidated pipeline)"
    else
        log_warning "$active_workflows active workflows found (expected: 1)"
        echo "Active workflows:"
        find "$WORKFLOWS_DIR" -name "*.yml" -not -path "*/archive/*" -exec basename {} \;
    fi
    
    # Check archive directory
    archived_count=$(ls -1 "$ARCHIVE_DIR"/*.yml 2>/dev/null | wc -l || echo "0")
    log_info "Archived workflows: $archived_count"
    
    # Check backup integrity
    backup_count=$(ls -1 "$BACKUP_DIR"/*.yml 2>/dev/null | wc -l || echo "0")
    log_info "Backup contains: $backup_count workflows"
    
    # Validate git status
    if git diff --quiet --exit-code; then
        log_warning "No changes staged - you may need to commit the migration"
    else
        log_info "Changes ready to commit"
        echo ""
        echo "Suggested commit message:"
        echo "feat(ci): consolidate fragmented workflows into unified pipeline"
        echo ""
        echo "- Replace 37 fragmented workflows with single optimized pipeline"
        echo "- Add intelligent caching and parallel execution"
        echo "- Include comprehensive security scanning"
        echo "- Support 4 build modes: fast/full/debug/security"
        echo "- Archive old workflows for potential rollback"
        echo ""
    fi
}

# Main execution function
main() {
    echo "🚀 Nephoran CI/CD Pipeline Migration Tool"
    echo "=========================================="
    echo ""
    
    # Parse command line arguments
    FORCE=false
    DRY_RUN=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                FORCE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help)
                cat <<EOF
Usage: $0 [OPTIONS]

Migrate from fragmented CI/CD workflows to consolidated pipeline

OPTIONS:
    --force     Skip confirmation prompts
    --dry-run   Show what would be done without making changes
    --help      Show this help message

EXAMPLES:
    $0                    # Interactive migration
    $0 --force           # Automatic migration without prompts
    $0 --dry-run         # Preview changes without executing

EOF
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Perform checks
    check_prerequisites
    analyze_workflows
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "DRY RUN MODE - No changes will be made"
        echo ""
        echo "Would perform these actions:"
        echo "1. Create backup in: $BACKUP_DIR"
        echo "2. Move $(ls -1 "$WORKFLOWS_DIR"/*.yml 2>/dev/null | wc -l) workflows to archive"
        echo "3. Disable fragmented workflows by renaming"
        echo "4. Validate consolidated workflow"
        echo "5. Create migration summary"
        exit 0
    fi
    
    # Confirmation prompt
    if [[ "$FORCE" != "true" ]]; then
        echo ""
        log_warning "This will disable $(echo "${FRAGMENTED_WORKFLOWS[@]}" | wc -w) fragmented workflows"
        log_info "The consolidated pipeline will become the primary CI/CD system"
        echo ""
        read -p "Continue with migration? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Migration cancelled by user"
            exit 0
        fi
    fi
    
    # Execute migration steps
    log_info "Starting migration process..."
    
    create_backup
    validate_consolidated_workflow
    disable_workflows
    create_migration_summary
    post_migration_checks
    
    echo ""
    echo "🎉 Migration completed successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Review the changes: git status"
    echo "2. Commit the migration: git add . && git commit -m 'feat(ci): consolidate workflows'"
    echo "3. Push to trigger first consolidated pipeline run: git push"
    echo "4. Monitor the pipeline execution in GitHub Actions"
    echo ""
    echo "📖 Documentation: docs/ci-cd-consolidated-pipeline-2025.md"
    echo "🔄 Rollback backup: $BACKUP_DIR"
    echo ""
}

# Execute main function with all arguments
main "$@"