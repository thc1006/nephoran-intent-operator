#!/bin/bash

# validate-docs.sh - Comprehensive Documentation Quality Validation
# 
# This script validates all documentation in the Nephoran Intent Operator project
# including link validation, API documentation accuracy, runbook testing, and
# broken link detection with detailed reporting.

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DOCS_DIR="$PROJECT_ROOT/docs"
API_DIR="$PROJECT_ROOT/api"
CONFIG_DIR="$PROJECT_ROOT/config"
DEPLOYMENTS_DIR="$PROJECT_ROOT/deployments"
SCRIPTS_DIR="$PROJECT_ROOT/scripts"
REPORTS_DIR="$PROJECT_ROOT/.excellence-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="$REPORTS_DIR/docs_validation_$TIMESTAMP.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

# Initialize report structure
init_report() {
    mkdir -p "$REPORTS_DIR"
    cat > "$REPORT_FILE" <<EOF
{
    "timestamp": "$TIMESTAMP",
    "project": "nephoran-intent-operator",
    "validation_type": "documentation_quality",
    "results": {
        "link_validation": {
            "total_links": 0,
            "valid_links": 0,
            "broken_links": 0,
            "external_links": 0,
            "internal_links": 0,
            "broken_link_details": []
        },
        "api_documentation": {
            "total_apis": 0,
            "documented_apis": 0,
            "accuracy_score": 0,
            "missing_documentation": [],
            "outdated_documentation": []
        },
        "runbook_validation": {
            "total_runbooks": 0,
            "validated_runbooks": 0,
            "failed_procedures": [],
            "success_rate": 0
        },
        "content_quality": {
            "spelling_errors": 0,
            "grammar_issues": 0,
            "formatting_issues": 0,
            "consistency_issues": []
        },
        "completeness": {
            "missing_sections": [],
            "incomplete_sections": [],
            "completeness_score": 0
        }
    },
    "overall_score": 0,
    "recommendations": []
}
EOF
}

# Find all documentation files
find_docs() {
    log_info "Finding documentation files..."
    
    # Find markdown files
    find "$PROJECT_ROOT" -type f \( -name "*.md" -o -name "*.rst" -o -name "*.txt" \) \
        ! -path "*/vendor/*" \
        ! -path "*/.git/*" \
        ! -path "*/node_modules/*" \
        ! -path "*/.excellence-reports/*" | sort
}

# Validate links in documentation
validate_links() {
    local doc_files=("$@")
    local total_links=0
    local valid_links=0
    local broken_links=0
    local external_links=0
    local internal_links=0
    local broken_link_details=()
    
    log_info "Validating links in documentation..."
    
    for file in "${doc_files[@]}"; do
        log_info "Checking links in: $file"
        
        # Extract all links using grep and sed
        local links
        links=$(grep -oP '\[([^\]]*)\]\(([^)]*)\)' "$file" 2>/dev/null | sed -n 's/.*](\([^)]*\)).*/\1/p' || true)
        
        while IFS= read -r link; do
            [[ -z "$link" ]] && continue
            
            ((total_links++))
            
            if [[ "$link" =~ ^https?:// ]]; then
                # External link
                ((external_links++))
                
                # Check external link with timeout
                if timeout 10 curl -s --head "$link" > /dev/null 2>&1; then
                    ((valid_links++))
                    log_success "✓ Valid external link: $link"
                else
                    ((broken_links++))
                    broken_link_details+=("{\"file\": \"$file\", \"link\": \"$link\", \"type\": \"external\", \"status\": \"broken\"}")
                    log_error "✗ Broken external link: $link in $file"
                fi
            elif [[ "$link" =~ ^mailto: ]]; then
                # Email link - validate format
                ((external_links++))
                if [[ "$link" =~ ^mailto:[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
                    ((valid_links++))
                    log_success "✓ Valid email link: $link"
                else
                    ((broken_links++))
                    broken_link_details+=("{\"file\": \"$file\", \"link\": \"$link\", \"type\": \"email\", \"status\": \"invalid_format\"}")
                    log_error "✗ Invalid email format: $link in $file"
                fi
            else
                # Internal link
                ((internal_links++))
                
                # Resolve relative path
                local target_path
                if [[ "$link" =~ ^/ ]]; then
                    target_path="$PROJECT_ROOT$link"
                else
                    target_path="$(dirname "$file")/$link"
                fi
                
                # Remove anchor (#section)
                target_path="${target_path%#*}"
                
                if [[ -f "$target_path" || -d "$target_path" ]]; then
                    ((valid_links++))
                    log_success "✓ Valid internal link: $link"
                else
                    ((broken_links++))
                    broken_link_details+=("{\"file\": \"$file\", \"link\": \"$link\", \"type\": \"internal\", \"status\": \"not_found\", \"target\": \"$target_path\"}")
                    log_error "✗ Broken internal link: $link in $file (target: $target_path)"
                fi
            fi
        done <<< "$links"
    done
    
    # Update report
    jq --argjson total "$total_links" \
       --argjson valid "$valid_links" \
       --argjson broken "$broken_links" \
       --argjson external "$external_links" \
       --argjson internal "$internal_links" \
       --argjson details "[$(IFS=,; echo "${broken_link_details[*]}")]" \
       '.results.link_validation = {
           "total_links": $total,
           "valid_links": $valid,
           "broken_links": $broken,
           "external_links": $external,
           "internal_links": $internal,
           "broken_link_details": $details
       }' "$REPORT_FILE" > "$REPORT_FILE.tmp" && mv "$REPORT_FILE.tmp" "$REPORT_FILE"
}

# Validate API documentation accuracy
validate_api_documentation() {
    log_info "Validating API documentation accuracy..."
    
    local total_apis=0
    local documented_apis=0
    local missing_documentation=()
    local outdated_documentation=()
    
    # Find API definitions
    if [[ -d "$API_DIR" ]]; then
        while IFS= read -r -d '' api_file; do
            ((total_apis++))
            
            local api_name
            api_name=$(basename "$api_file" .go)
            
            # Check if API is documented
            local doc_found=false
            
            # Check in various documentation locations
            local doc_locations=(
                "$DOCS_DIR/api/$api_name.md"
                "$DOCS_DIR/reference/api-$api_name.md"
                "$PROJECT_ROOT/README.md"
                "$DOCS_DIR/README.md"
            )
            
            for doc_location in "${doc_locations[@]}"; do
                if [[ -f "$doc_location" ]] && grep -q "$api_name" "$doc_location"; then
                    doc_found=true
                    ((documented_apis++))
                    break
                fi
            done
            
            if [[ "$doc_found" == false ]]; then
                missing_documentation+=("\"$api_name\"")
                log_warn "Missing documentation for API: $api_name"
            fi
            
        done < <(find "$API_DIR" -name "*.go" -type f -print0 2>/dev/null || true)
    fi
    
    # Check for outdated documentation by comparing modification times
    if [[ -d "$DOCS_DIR" ]]; then
        while IFS= read -r -d '' doc_file; do
            local doc_name
            doc_name=$(basename "$doc_file" .md)
            
            # Find corresponding source file
            local source_file
            source_file=$(find "$PROJECT_ROOT" -name "*$doc_name*" -type f \( -name "*.go" -o -name "*.yaml" \) | head -1)
            
            if [[ -n "$source_file" && -f "$source_file" ]]; then
                if [[ "$source_file" -nt "$doc_file" ]]; then
                    outdated_documentation+=("\"$doc_file\"")
                    log_warn "Potentially outdated documentation: $doc_file (source modified more recently)"
                fi
            fi
            
        done < <(find "$DOCS_DIR" -name "*.md" -type f -print0 2>/dev/null || true)
    fi
    
    # Calculate accuracy score
    local accuracy_score=0
    if [[ $total_apis -gt 0 ]]; then
        accuracy_score=$((documented_apis * 100 / total_apis))
    fi
    
    # Update report
    jq --argjson total "$total_apis" \
       --argjson documented "$documented_apis" \
       --argjson accuracy "$accuracy_score" \
       --argjson missing "[$(IFS=,; echo "${missing_documentation[*]}")]" \
       --argjson outdated "[$(IFS=,; echo "${outdated_documentation[*]}")]" \
       '.results.api_documentation = {
           "total_apis": $total,
           "documented_apis": $documented,
           "accuracy_score": $accuracy,
           "missing_documentation": $missing,
           "outdated_documentation": $outdated
       }' "$REPORT_FILE" > "$REPORT_FILE.tmp" && mv "$REPORT_FILE.tmp" "$REPORT_FILE"
}

# Validate runbook procedures
validate_runbooks() {
    log_info "Validating runbook procedures..."
    
    local total_runbooks=0
    local validated_runbooks=0
    local failed_procedures=()
    
    # Find runbook files
    local runbook_patterns=(
        "runbook"
        "procedure"
        "playbook"
        "guide"
        "howto"
        "tutorial"
    )
    
    for pattern in "${runbook_patterns[@]}"; do
        while IFS= read -r -d '' runbook_file; do
            [[ -f "$runbook_file" ]] || continue
            ((total_runbooks++))
            
            log_info "Validating runbook: $runbook_file"
            
            # Extract code blocks and validate syntax
            local validation_passed=true
            local line_number=0
            
            while IFS= read -r line; do
                ((line_number++))
                
                # Check for shell commands
                if [[ "$line" =~ ^\`\`\`bash || "$line" =~ ^\`\`\`sh ]]; then
                    local in_code_block=true
                    local code_block=""
                    
                    while IFS= read -r code_line; do
                        ((line_number++))
                        
                        if [[ "$code_line" =~ ^\`\`\` ]]; then
                            break
                        fi
                        
                        code_block+="$code_line"$'\n'
                    done
                    
                    # Validate shell syntax (dry run)
                    if ! echo "$code_block" | bash -n 2>/dev/null; then
                        validation_passed=false
                        failed_procedures+=("{\"file\": \"$runbook_file\", \"line\": $line_number, \"error\": \"invalid_shell_syntax\"}")
                        log_error "Invalid shell syntax in $runbook_file at line $line_number"
                    fi
                fi
                
                # Check for kubectl commands
                if [[ "$line" =~ kubectl ]]; then
                    # Validate kubectl command structure
                    if ! echo "$line" | grep -qE "kubectl\s+(get|apply|delete|create|patch|scale|rollout|logs|exec|port-forward|config)"; then
                        validation_passed=false
                        failed_procedures+=("{\"file\": \"$runbook_file\", \"line\": $line_number, \"error\": \"invalid_kubectl_command\"}")
                        log_warn "Potentially invalid kubectl command in $runbook_file at line $line_number"
                    fi
                fi
                
            done < "$runbook_file"
            
            if [[ "$validation_passed" == true ]]; then
                ((validated_runbooks++))
                log_success "✓ Runbook validation passed: $runbook_file"
            else
                log_error "✗ Runbook validation failed: $runbook_file"
            fi
            
        done < <(find "$PROJECT_ROOT" -type f -name "*$pattern*" \( -name "*.md" -o -name "*.rst" \) -print0 2>/dev/null || true)
    done
    
    # Calculate success rate
    local success_rate=0
    if [[ $total_runbooks -gt 0 ]]; then
        success_rate=$((validated_runbooks * 100 / total_runbooks))
    fi
    
    # Update report
    jq --argjson total "$total_runbooks" \
       --argjson validated "$validated_runbooks" \
       --argjson rate "$success_rate" \
       --argjson failed "[$(IFS=,; echo "${failed_procedures[*]}")]" \
       '.results.runbook_validation = {
           "total_runbooks": $total,
           "validated_runbooks": $validated,
           "success_rate": $rate,
           "failed_procedures": $failed
       }' "$REPORT_FILE" > "$REPORT_FILE.tmp" && mv "$REPORT_FILE.tmp" "$REPORT_FILE"
}

# Check content quality (spelling, grammar, formatting)
check_content_quality() {
    log_info "Checking content quality..."
    
    local spelling_errors=0
    local grammar_issues=0
    local formatting_issues=0
    local consistency_issues=()
    
    # Find all documentation files
    local doc_files
    readarray -t doc_files < <(find_docs)
    
    for file in "${doc_files[@]}"; do
        log_info "Checking content quality: $file"
        
        # Check for common formatting issues
        local line_number=0
        while IFS= read -r line; do
            ((line_number++))
            
            # Check for missing spaces after punctuation
            if [[ "$line" =~ [.!?][A-Z] ]]; then
                ((formatting_issues++))
                consistency_issues+=("{\"file\": \"$file\", \"line\": $line_number, \"issue\": \"missing_space_after_punctuation\"}")
            fi
            
            # Check for inconsistent heading styles
            if [[ "$line" =~ ^#+\s*[a-z] ]]; then
                local heading_level
                heading_level=$(echo "$line" | grep -o '^#*' | wc -c)
                if [[ $heading_level -gt 1 ]]; then
                    consistency_issues+=("{\"file\": \"$file\", \"line\": $line_number, \"issue\": \"heading_should_be_title_case\"}")
                fi
            fi
            
            # Check for trailing whitespace
            if [[ "$line" =~ \s+$ ]]; then
                ((formatting_issues++))
            fi
            
            # Check for tabs instead of spaces
            if [[ "$line" =~ $'\t' ]]; then
                ((formatting_issues++))
            fi
            
        done < "$file"
        
        # Check for basic spelling using built-in word lists
        if command -v aspell >/dev/null 2>&1; then
            local spell_errors
            spell_errors=$(aspell --mode=markdown --list < "$file" | wc -l)
            ((spelling_errors += spell_errors))
        fi
    done
    
    # Update report
    jq --argjson spelling "$spelling_errors" \
       --argjson grammar "$grammar_issues" \
       --argjson formatting "$formatting_issues" \
       --argjson consistency "[$(IFS=,; echo "${consistency_issues[*]}")]" \
       '.results.content_quality = {
           "spelling_errors": $spelling,
           "grammar_issues": $grammar,
           "formatting_issues": $formatting,
           "consistency_issues": $consistency
       }' "$REPORT_FILE" > "$REPORT_FILE.tmp" && mv "$REPORT_FILE.tmp" "$REPORT_FILE"
}

# Check documentation completeness
check_completeness() {
    log_info "Checking documentation completeness..."
    
    local missing_sections=()
    local incomplete_sections=()
    
    # Required sections for a complete project
    local required_sections=(
        "README.md"
        "INSTALLATION.md:docs/installation"
        "API_REFERENCE.md:docs/api"
        "CONFIGURATION.md:docs/configuration"
        "TROUBLESHOOTING.md:docs/troubleshooting"
        "CONTRIBUTING.md"
        "CHANGELOG.md"
        "LICENSE"
    )
    
    for section_spec in "${required_sections[@]}"; do
        local section_name="${section_spec%:*}"
        local section_paths="${section_spec#*:}"
        
        if [[ "$section_spec" == "$section_name" ]]; then
            # Single file check
            if [[ ! -f "$PROJECT_ROOT/$section_name" ]]; then
                missing_sections+=("\"$section_name\"")
                log_warn "Missing required section: $section_name"
            fi
        else
            # Multiple path check
            local found=false
            IFS=':' read -ra paths <<< "$section_paths"
            for path in "${paths[@]}"; do
                if [[ -f "$PROJECT_ROOT/$path.md" || -f "$PROJECT_ROOT/$path/README.md" || -d "$PROJECT_ROOT/$path" ]]; then
                    found=true
                    break
                fi
            done
            
            if [[ "$found" == false ]]; then
                missing_sections+=("\"$section_name\"")
                log_warn "Missing required section: $section_name (checked paths: ${section_paths//:/, })"
            fi
        fi
    done
    
    # Check for incomplete sections (very short files)
    while IFS= read -r -d '' doc_file; do
        local word_count
        word_count=$(wc -w < "$doc_file")
        
        if [[ $word_count -lt 50 ]]; then
            local relative_path="${doc_file#$PROJECT_ROOT/}"
            incomplete_sections+=("\"$relative_path\"")
            log_warn "Potentially incomplete section: $relative_path ($word_count words)"
        fi
        
    done < <(find "$PROJECT_ROOT" -name "*.md" -type f -not -path "*/.git/*" -print0 2>/dev/null || true)
    
    # Calculate completeness score
    local total_required=${#required_sections[@]}
    local missing_count=${#missing_sections[@]}
    local completeness_score=0
    
    if [[ $total_required -gt 0 ]]; then
        completeness_score=$(((total_required - missing_count) * 100 / total_required))
    fi
    
    # Update report
    jq --argjson missing "[$(IFS=,; echo "${missing_sections[*]}")]" \
       --argjson incomplete "[$(IFS=,; echo "${incomplete_sections[*]}")]" \
       --argjson score "$completeness_score" \
       '.results.completeness = {
           "missing_sections": $missing,
           "incomplete_sections": $incomplete,
           "completeness_score": $score
       }' "$REPORT_FILE" > "$REPORT_FILE.tmp" && mv "$REPORT_FILE.tmp" "$REPORT_FILE"
}

# Calculate overall score and generate recommendations
calculate_overall_score() {
    log_info "Calculating overall documentation quality score..."
    
    local recommendations=()
    
    # Extract scores from report
    local link_score
    local api_score
    local runbook_score
    local completeness_score
    
    link_score=$(jq -r '.results.link_validation | if .total_links > 0 then (.valid_links * 100 / .total_links) else 100 end' "$REPORT_FILE")
    api_score=$(jq -r '.results.api_documentation.accuracy_score' "$REPORT_FILE")
    runbook_score=$(jq -r '.results.runbook_validation.success_rate' "$REPORT_FILE")
    completeness_score=$(jq -r '.results.completeness.completeness_score' "$REPORT_FILE")
    
    # Content quality score (inverse of issues)
    local content_issues
    content_issues=$(jq -r '.results.content_quality | (.spelling_errors + .grammar_issues + .formatting_issues)' "$REPORT_FILE")
    local content_score=$((100 - content_issues))
    if [[ $content_score -lt 0 ]]; then content_score=0; fi
    
    # Calculate weighted overall score
    local overall_score
    overall_score=$(echo "scale=2; ($link_score * 0.25 + $api_score * 0.25 + $runbook_score * 0.20 + $completeness_score * 0.20 + $content_score * 0.10)" | bc)
    
    # Generate recommendations based on scores
    if (( $(echo "$link_score < 90" | bc -l) )); then
        recommendations+=("\"Fix broken links to improve reliability (current: ${link_score}%)\"")
    fi
    
    if (( $(echo "$api_score < 80" | bc -l) )); then
        recommendations+=("\"Improve API documentation coverage (current: ${api_score}%)\"")
    fi
    
    if (( $(echo "$runbook_score < 85" | bc -l) )); then
        recommendations+=("\"Validate and fix runbook procedures (current: ${runbook_score}%)\"")
    fi
    
    if (( $(echo "$completeness_score < 75" | bc -l) )); then
        recommendations+=("\"Add missing documentation sections (current: ${completeness_score}%)\"")
    fi
    
    if [[ $content_issues -gt 10 ]]; then
        recommendations+=("\"Address content quality issues ($content_issues issues found)\"")
    fi
    
    # Update report with overall score and recommendations
    jq --argjson score "$overall_score" \
       --argjson recs "[$(IFS=,; echo "${recommendations[*]}")]" \
       '.overall_score = $score | .recommendations = $recs' "$REPORT_FILE" > "$REPORT_FILE.tmp" && mv "$REPORT_FILE.tmp" "$REPORT_FILE"
}

# Generate summary report
generate_summary() {
    log_info "Generating documentation validation summary..."
    
    echo ""
    echo "==============================================="
    echo "  DOCUMENTATION QUALITY VALIDATION REPORT"
    echo "==============================================="
    echo ""
    
    local overall_score
    overall_score=$(jq -r '.overall_score' "$REPORT_FILE")
    
    echo "Overall Documentation Quality Score: $overall_score%"
    echo ""
    
    # Link validation summary
    echo "📋 Link Validation:"
    jq -r '.results.link_validation | "  Total Links: \(.total_links)\n  Valid Links: \(.valid_links)\n  Broken Links: \(.broken_links)\n  External Links: \(.external_links)\n  Internal Links: \(.internal_links)"' "$REPORT_FILE"
    echo ""
    
    # API documentation summary
    echo "📚 API Documentation:"
    jq -r '.results.api_documentation | "  Total APIs: \(.total_apis)\n  Documented APIs: \(.documented_apis)\n  Accuracy Score: \(.accuracy_score)%"' "$REPORT_FILE"
    echo ""
    
    # Runbook validation summary
    echo "📖 Runbook Validation:"
    jq -r '.results.runbook_validation | "  Total Runbooks: \(.total_runbooks)\n  Validated Runbooks: \(.validated_runbooks)\n  Success Rate: \(.success_rate)%"' "$REPORT_FILE"
    echo ""
    
    # Content quality summary
    echo "✍️  Content Quality:"
    jq -r '.results.content_quality | "  Spelling Errors: \(.spelling_errors)\n  Grammar Issues: \(.grammar_issues)\n  Formatting Issues: \(.formatting_issues)\n  Consistency Issues: \(.consistency_issues | length)"' "$REPORT_FILE"
    echo ""
    
    # Completeness summary
    echo "📑 Completeness:"
    jq -r '.results.completeness | "  Missing Sections: \(.missing_sections | length)\n  Incomplete Sections: \(.incomplete_sections | length)\n  Completeness Score: \(.completeness_score)%"' "$REPORT_FILE"
    echo ""
    
    # Recommendations
    echo "💡 Recommendations:"
    jq -r '.recommendations[] | "  - \(.)"' "$REPORT_FILE"
    echo ""
    
    echo "Full report saved to: $REPORT_FILE"
    echo ""
    
    # Set exit code based on score
    if (( $(echo "$overall_score >= 85" | bc -l) )); then
        log_success "Documentation quality is excellent (>= 85%)"
        return 0
    elif (( $(echo "$overall_score >= 70" | bc -l) )); then
        log_warn "Documentation quality needs improvement (70-84%)"
        return 1
    else
        log_error "Documentation quality is poor (< 70%)"
        return 2
    fi
}

# Main execution
main() {
    log_info "Starting comprehensive documentation quality validation..."
    
    # Check dependencies
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed. Please install jq."
        exit 1
    fi
    
    if ! command -v bc >/dev/null 2>&1; then
        log_error "bc is required but not installed. Please install bc."
        exit 1
    fi
    
    # Initialize report
    init_report
    
    # Find all documentation files
    local doc_files
    readarray -t doc_files < <(find_docs)
    
    if [[ ${#doc_files[@]} -eq 0 ]]; then
        log_warn "No documentation files found"
        exit 1
    fi
    
    log_info "Found ${#doc_files[@]} documentation files"
    
    # Run all validation checks
    validate_links "${doc_files[@]}"
    validate_api_documentation
    validate_runbooks
    check_content_quality
    check_completeness
    calculate_overall_score
    
    # Generate summary
    generate_summary
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --help, -h     Show this help message"
            echo "  --report-dir   Specify custom report directory (default: .excellence-reports)"
            echo ""
            echo "This script validates documentation quality including:"
            echo "  - Link validation (internal and external)"
            echo "  - API documentation accuracy"
            echo "  - Runbook procedure testing"
            echo "  - Content quality checks"
            echo "  - Documentation completeness"
            exit 0
            ;;
        --report-dir)
            REPORTS_DIR="$2"
            REPORT_FILE="$REPORTS_DIR/docs_validation_$TIMESTAMP.json"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Execute main function
main "$@"