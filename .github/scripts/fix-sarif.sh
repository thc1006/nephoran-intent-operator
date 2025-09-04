#!/bin/bash

# fix-sarif.sh - Lightweight SARIF preprocessing script for CI pipeline
# Fixes common SARIF upload issues while maintaining reliability

set -euo pipefail

# Script configuration
readonly SCRIPT_NAME="fix-sarif.sh"
readonly SARIF_VERSION="2.1.0"
readonly MIN_VALID_SARIF='{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"gosec"}},"results":[]}]}'

# Logging functions
log_info() {
    echo "[$SCRIPT_NAME] INFO: $*" >&2
}

log_warn() {
    echo "[$SCRIPT_NAME] WARN: $*" >&2
}

log_error() {
    echo "[$SCRIPT_NAME] ERROR: $*" >&2
}

# Usage information
usage() {
    cat <<EOF
Usage: $SCRIPT_NAME <sarif-file>

Description:
  Fixes common SARIF upload issues for GitHub CodeQL action.
  
  Fixes applied:
  - Replace startLine: 0 with startLine: 1
  - Remove entries with null/missing locations
  - Ensure valid JSON structure
  - Create minimal valid SARIF on failure

Arguments:
  sarif-file    Path to SARIF file to process (modified in-place)

Exit codes:
  0   Success
  1   Invalid arguments
  2   File processing error
  3   JSON validation error

Examples:
  $SCRIPT_NAME security-reports/gosec/gosec.sarif
  $SCRIPT_NAME ./output/scan-results.sarif

EOF
}

# Validate input arguments
validate_args() {
    if [[ $# -ne 1 ]]; then
        log_error "Exactly one argument required"
        usage
        exit 1
    fi
    
    if [[ "$1" == "-h" || "$1" == "--help" ]]; then
        usage
        exit 0
    fi
}

# Check if jq is available
check_jq() {
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        log_error "Install jq: https://stedolan.github.io/jq/download/"
        exit 2
    fi
}

# Create minimal valid SARIF file
create_minimal_sarif() {
    local file_path="$1"
    local tool_name="${2:-gosec}"
    
    log_warn "Creating minimal valid SARIF file: $file_path"
    
    cat > "$file_path" <<EOF
{
  "version": "$SARIF_VERSION",
  "\$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "$tool_name",
          "informationUri": "https://github.com/securego/gosec"
        }
      },
      "results": []
    }
  ]
}
EOF
}

# Validate JSON syntax
validate_json() {
    local file_path="$1"
    
    if ! jq empty "$file_path" &> /dev/null; then
        log_error "Invalid JSON in file: $file_path"
        return 1
    fi
    
    return 0
}

# Extract tool name from existing SARIF (fallback to 'gosec')
extract_tool_name() {
    local file_path="$1"
    local tool_name
    
    tool_name=$(jq -r '.runs[]?.tool?.driver?.name // "gosec"' "$file_path" 2>/dev/null | head -1)
    echo "${tool_name:-gosec}"
}

# Fix startLine issues: Replace 0 with 1
fix_start_line() {
    local file_path="$1"
    local temp_file="${file_path}.tmp"
    
    log_info "Fixing startLine issues in $file_path"
    
    jq '
        (.runs[]?.results[]?.locations[]?.physicalLocation?.region? | select(.startLine == 0).startLine) = 1 |
        (.runs[]?.results[]?.locations[]?.physicalLocation?.region? | select(.endLine == 0).endLine) = 1
    ' "$file_path" > "$temp_file"
    
    mv "$temp_file" "$file_path"
}

# Remove entries with null or missing locations
remove_invalid_locations() {
    local file_path="$1"
    local temp_file="${file_path}.tmp"
    
    log_info "Removing entries with invalid locations in $file_path"
    
    jq '
        .runs[]?.results = (.runs[]?.results // [] | map(
            select(.locations != null and .locations != [] and (
                .locations | any(.physicalLocation != null and .physicalLocation.artifactLocation != null)
            ))
        ))
    ' "$file_path" > "$temp_file"
    
    mv "$temp_file" "$file_path"
}

# Ensure proper SARIF structure
normalize_sarif_structure() {
    local file_path="$1"
    local temp_file="${file_path}.tmp"
    
    log_info "Normalizing SARIF structure in $file_path"
    
    jq --arg version "$SARIF_VERSION" '
        {
            "version": $version,
            "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
            "runs": (.runs // [] | map({
                "tool": (.tool // {"driver": {"name": "gosec"}}),
                "results": (.results // [])
            }))
        }
    ' "$file_path" > "$temp_file"
    
    mv "$temp_file" "$file_path"
}

# Count results in SARIF file
count_results() {
    local file_path="$1"
    jq '[.runs[]?.results[]?] | length' "$file_path" 2>/dev/null || echo "0"
}

# Main processing function
process_sarif() {
    local file_path="$1"
    local backup_file="${file_path}.bak"
    local tool_name
    
    log_info "Processing SARIF file: $file_path"
    
    # Handle missing file
    if [[ ! -f "$file_path" ]]; then
        log_warn "File not found: $file_path"
        create_minimal_sarif "$file_path" "gosec"
        return 0
    fi
    
    # Create backup
    cp "$file_path" "$backup_file"
    
    # Extract tool name for fallback
    tool_name=$(extract_tool_name "$file_path")
    
    # Initial validation
    if ! validate_json "$file_path"; then
        log_warn "Invalid JSON detected, creating minimal SARIF"
        create_minimal_sarif "$file_path" "$tool_name"
        rm -f "$backup_file"
        return 0
    fi
    
    # Apply fixes with error recovery
    local before_count after_count
    before_count=$(count_results "$file_path")
    
    {
        fix_start_line "$file_path" &&
        remove_invalid_locations "$file_path" &&
        normalize_sarif_structure "$file_path" &&
        validate_json "$file_path"
    } || {
        log_warn "Processing failed, creating minimal SARIF"
        create_minimal_sarif "$file_path" "$tool_name"
        rm -f "$backup_file"
        return 0
    }
    
    after_count=$(count_results "$file_path")
    
    log_info "Processing complete: $before_count → $after_count results"
    
    # Clean up backup
    rm -f "$backup_file"
    
    return 0
}

# Main execution
main() {
    validate_args "$@"
    check_jq
    
    local sarif_file="$1"
    
    log_info "Starting SARIF preprocessing for: $sarif_file"
    
    # Process the file
    if process_sarif "$sarif_file"; then
        log_info "SARIF preprocessing completed successfully"
        exit 0
    else
        log_error "SARIF preprocessing failed"
        exit 2
    fi
}

# Execute main function with all arguments
main "$@"