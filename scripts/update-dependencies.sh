#!/bin/bash

# update-dependencies.sh - Automated dependency update script
# Part of Phase 1: Dependency Management Fixes

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Nephoran Intent Operator Dependency Update ==="
echo "Project Root: $PROJECT_ROOT"
echo "Update Date: $(date)"
echo

cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BACKUP_DIR="$PROJECT_ROOT/.dependency-backups"
DRY_RUN=${1:-false}

# Ensure backup directory exists
mkdir -p "$BACKUP_DIR"

# Function to create backup
create_backup() {
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local backup_file="$BACKUP_DIR/go.mod.backup.$timestamp"
    
    echo "Creating backup of go.mod..."
    cp go.mod "$backup_file"
    cp go.sum "$backup_file.sum" 2>/dev/null || true
    
    echo -e "${GREEN}✓ Backup created: $backup_file${NC}"
}

# Function to check Go version compatibility
check_go_version() {
    echo "=== Go Version Check ==="
    
    local go_version=$(go version | cut -d' ' -f3 | sed 's/go//')
    local mod_version=$(grep "^go " go.mod | cut -d' ' -f2)
    
    echo "System Go version: $go_version"
    echo "go.mod requires: $mod_version"
    
    # Simple version comparison (assumes semantic versioning)
    if [[ "$go_version" < "$mod_version" ]]; then
        echo -e "${YELLOW}⚠ Warning: System Go version ($go_version) is older than go.mod requirement ($mod_version)${NC}"
        return 1
    else
        echo -e "${GREEN}✓ Go version is compatible${NC}"
        return 0
    fi
    echo
}

# Function to update security-critical dependencies
update_security_deps() {
    echo "=== Updating Security-Critical Dependencies ==="
    
    local security_deps=(
        "golang.org/x/crypto"
        "github.com/golang-jwt/jwt/v5"
        "golang.org/x/oauth2"
        "google.golang.org/grpc"
        "google.golang.org/protobuf"
        "golang.org/x/net"
        "golang.org/x/sys"
    )
    
    for dep in "${security_deps[@]}"; do
        echo "Checking updates for $dep..."
        
        if go list -m -u "$dep" >/dev/null 2>&1; then
            local current_version=$(go list -m "$dep" 2>/dev/null | cut -d' ' -f2 || echo "not found")
            local latest_info=$(go list -m -u "$dep" 2>/dev/null || echo "")
            
            if [[ "$latest_info" == *"["* ]]; then
                local latest_version=$(echo "$latest_info" | grep -o '\[.*\]' | tr -d '[]')
                echo -e "${YELLOW}  Current: $current_version → Available: $latest_version${NC}"
                
                if [[ "$DRY_RUN" != "true" ]]; then
                    echo "  Updating $dep to $latest_version..."
                    if go get "$dep@$latest_version"; then
                        echo -e "${GREEN}  ✓ Updated successfully${NC}"
                    else
                        echo -e "${RED}  ✗ Update failed${NC}"
                    fi
                else
                    echo -e "${BLUE}  [DRY RUN] Would update to $latest_version${NC}"
                fi
            else
                echo -e "${GREEN}  ✓ Already at latest version: $current_version${NC}"
            fi
        else
            echo -e "${YELLOW}  ⚠ Dependency not found in current module${NC}"
        fi
        echo
    done
}

# Function to update Kubernetes dependencies
update_k8s_deps() {
    echo "=== Updating Kubernetes Dependencies ==="
    
    local k8s_deps=(
        "k8s.io/api"
        "k8s.io/apimachinery"
        "k8s.io/client-go"
        "sigs.k8s.io/controller-runtime"
    )
    
    # Get current k8s version from go.mod
    local current_k8s_version=$(go list -m k8s.io/api 2>/dev/null | cut -d' ' -f2 || echo "")
    
    if [[ -n "$current_k8s_version" ]]; then
        echo "Current Kubernetes version: $current_k8s_version"
        
        # Check for available updates (this is simplified - in practice you'd check k8s release cycles)
        echo "Checking for Kubernetes updates..."
        
        for dep in "${k8s_deps[@]}"; do
            echo "Checking $dep..."
            
            if go list -m -u "$dep" >/dev/null 2>&1; then
                local current=$(go list -m "$dep" 2>/dev/null | cut -d' ' -f2 || echo "not found")
                local update_info=$(go list -m -u "$dep" 2>/dev/null || echo "")
                
                if [[ "$update_info" == *"["* ]]; then
                    local available=$(echo "$update_info" | grep -o '\[.*\]' | tr -d '[]')
                    echo -e "${YELLOW}  Current: $current → Available: $available${NC}"
                    
                    if [[ "$DRY_RUN" != "true" ]]; then
                        echo "  Note: Kubernetes dependencies should be updated together"
                        echo "  Consider updating all k8s deps to the same version"
                    else
                        echo -e "${BLUE}  [DRY RUN] Would consider updating to $available${NC}"
                    fi
                else
                    echo -e "${GREEN}  ✓ Already at current version: $current${NC}"
                fi
            fi
        done
    else
        echo -e "${YELLOW}⚠ No Kubernetes dependencies found${NC}"
    fi
    echo
}

# Function to clean up and tidy modules
cleanup_modules() {
    echo "=== Module Cleanup ==="
    
    if [[ "$DRY_RUN" != "true" ]]; then
        echo "Running go mod tidy..."
        if go mod tidy; then
            echo -e "${GREEN}✓ go mod tidy completed successfully${NC}"
        else
            echo -e "${RED}✗ go mod tidy failed${NC}"
            return 1
        fi
        
        echo "Running go mod download..."
        if go mod download; then
            echo -e "${GREEN}✓ Dependencies downloaded successfully${NC}"
        else
            echo -e "${RED}✗ Dependency download failed${NC}"
            return 1
        fi
    else
        echo -e "${BLUE}[DRY RUN] Would run go mod tidy and go mod download${NC}"
    fi
    echo
}

# Function to verify after updates
verify_updates() {
    echo "=== Verification ==="
    
    if [[ "$DRY_RUN" != "true" ]]; then
        echo "Verifying module integrity..."
        if go mod verify; then
            echo -e "${GREEN}✓ All modules verified successfully${NC}"
        else
            echo -e "${RED}✗ Module verification failed${NC}"
            return 1
        fi
        
        echo "Testing basic build..."
        if go list ./... >/dev/null 2>&1; then
            echo -e "${GREEN}✓ All packages can be loaded${NC}"
        else
            echo -e "${RED}✗ Some packages failed to load${NC}"
            return 1
        fi
    else
        echo -e "${BLUE}[DRY RUN] Would verify modules and test build${NC}"
    fi
    echo
}

# Function to generate update report
generate_report() {
    echo "=== Update Report ==="
    
    local report_file="$PROJECT_ROOT/dependency-update-$(date +%Y%m%d-%H%M%S).log"
    
    {
        echo "Nephoran Intent Operator Dependency Update Report"
        echo "Generated: $(date)"
        echo "Mode: $([ "$DRY_RUN" = "true" ] && echo "DRY RUN" || echo "LIVE UPDATE")"
        echo
        echo "=== Security-Critical Dependencies ==="
        go list -m all | grep -E "(crypto|jwt|oauth|protobuf|grpc)" || echo "None found"
        echo
        echo "=== Kubernetes Dependencies ==="
        go list -m all | grep -E "k8s.io|sigs.k8s.io" || echo "None found"
        echo
        echo "=== Dependency Summary ==="
        echo "Total dependencies: $(go list -m all | tail -n +2 | wc -l)"
        echo "Indirect dependencies: $(go list -m all | grep "// indirect" | wc -l)"
        echo
    } > "$report_file"
    
    echo "Update report saved to: $report_file"
}

# Function to show help
show_help() {
    cat << EOF
Usage: $0 [dry-run]

Options:
  dry-run    Run in dry-run mode (show what would be updated without making changes)

Description:
  This script automates the update of Go dependencies for the Nephoran Intent Operator.
  It focuses on security-critical dependencies and maintains compatibility.

Safety Features:
  - Creates automatic backups of go.mod and go.sum
  - Runs in dry-run mode by default for safety
  - Verifies all modules after updates
  - Generates detailed update reports

Examples:
  $0 dry-run    # Show what would be updated (safe)
  $0 live       # Actually perform updates
EOF
}

# Main execution
main() {
    # Parse arguments
    case "${1:-dry-run}" in
        "help"|"-h"|"--help")
            show_help
            exit 0
            ;;
        "live"|"update"|"run")
            DRY_RUN=false
            echo -e "${YELLOW}⚠ Running in LIVE UPDATE mode${NC}"
            ;;
        "dry-run"|"dry"|"test"|*)
            DRY_RUN=true
            echo -e "${BLUE}ℹ Running in DRY RUN mode${NC}"
            ;;
    esac
    
    echo
    
    # Pre-flight checks
    if ! check_go_version; then
        echo -e "${YELLOW}⚠ Go version compatibility issues detected${NC}"
        echo "Consider updating Go or adjusting go.mod requirements"
        echo
    fi
    
    # Create backup only in live mode
    if [[ "$DRY_RUN" != "true" ]]; then
        create_backup
    fi
    
    # Update dependencies
    update_security_deps
    update_k8s_deps
    
    # Cleanup and verify
    cleanup_modules
    verify_updates
    
    # Generate report
    generate_report
    
    echo -e "${GREEN}Dependency update process completed${NC}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo
        echo "To actually perform updates, run:"
        echo "  $0 live"
    else
        echo
        echo "Updates applied. Consider running tests to verify functionality."
        echo "Backups are available in: $BACKUP_DIR"
    fi
}

# Run main function
main "$@"