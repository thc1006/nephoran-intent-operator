#!/bin/bash

# Docker Security Scanning Script for Nephoran Intent Operator
# Comprehensive security validation for container images

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORTS_DIR="${SCRIPT_DIR}/security-reports"
DATE_STAMP=$(date +%Y%m%d-%H%M%S)

# Security scanning tools
TRIVY_SEVERITY=${TRIVY_SEVERITY:-HIGH,CRITICAL}
DOCKER_BENCH_SECURITY=${DOCKER_BENCH_SECURITY:-false}
HADOLINT_ENABLED=${HADOLINT_ENABLED:-true}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Help function
show_help() {
    cat << EOF
Docker Security Scanning Script for Nephoran Intent Operator

Usage: $0 [options] <image|dockerfile|all>

Scanning Targets:
  image:<name>         Scan specific container image
  dockerfile:<path>    Scan Dockerfile for security issues
  all                  Run comprehensive security scan

Options:
  --severity          Set Trivy severity levels (default: ${TRIVY_SEVERITY})
  --format            Output format: table|json|sarif (default: table)
  --output-dir        Output directory for reports (default: ${REPORTS_DIR})
  --docker-bench      Run Docker Bench Security test
  --no-hadolint       Skip Hadolint Dockerfile scanning
  --exit-code         Exit with non-zero code on findings (default: 1)
  --help              Show this help message

Examples:
  $0 image:ghcr.io/thc1006/nephoran-intent-operator/llm-processor:v2.0.0
  $0 dockerfile:cmd/llm-processor/Dockerfile
  $0 all --docker-bench --format json

Security Tools Used:
  - Trivy: Vulnerability and misconfiguration scanner
  - Hadolint: Dockerfile best practices linter
  - Docker Bench Security: CIS Docker Benchmark
  - Custom security checks
EOF
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking security scanning prerequisites..."
    
    local missing_tools=()
    
    # Check Trivy
    if ! command -v trivy &> /dev/null; then
        missing_tools+=("trivy")
    fi
    
    # Check Hadolint
    if [[ "$HADOLINT_ENABLED" == true ]] && ! command -v hadolint &> /dev/null; then
        log_warning "Hadolint not found. Install from: https://github.com/hadolint/hadolint"
        HADOLINT_ENABLED=false
    fi
    
    # Check Docker Bench Security
    if [[ "$DOCKER_BENCH_SECURITY" == true ]] && ! command -v docker-bench-security &> /dev/null; then
        log_warning "Docker Bench Security not found. Skipping benchmark tests."
        DOCKER_BENCH_SECURITY=false
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Install Trivy: https://aquasecurity.github.io/trivy/latest/getting-started/installation/"
        exit 1
    fi
    
    # Create reports directory
    mkdir -p "$REPORTS_DIR"
    
    log_success "Security scanning prerequisites ready"
}

# Trivy vulnerability scanning
run_trivy_scan() {
    local target="$1"
    local target_type="$2"
    local format="${3:-table}"
    
    log_info "Running Trivy security scan..."
    log_info "Target: $target"
    log_info "Type: $target_type"
    log_info "Severity: $TRIVY_SEVERITY"
    
    local output_file=""
    local trivy_args=()
    
    # Configure output based on format
    case "$format" in
        json)
            output_file="${REPORTS_DIR}/trivy-${target_type}-${DATE_STAMP}.json"
            trivy_args+=(--format json --output "$output_file")
            ;;
        sarif)
            output_file="${REPORTS_DIR}/trivy-${target_type}-${DATE_STAMP}.sarif"
            trivy_args+=(--format sarif --output "$output_file")
            ;;
        table)
            trivy_args+=(--format table)
            ;;
    esac
    
    # Add common arguments
    trivy_args+=(
        --severity "$TRIVY_SEVERITY"
        --no-progress
        --ignore-unfixed
    )
    
    # Run appropriate Trivy scan
    case "$target_type" in
        image)
            trivy image "${trivy_args[@]}" "$target"
            ;;
        filesystem)
            trivy fs "${trivy_args[@]}" "$target"
            ;;
        config)
            trivy config "${trivy_args[@]}" "$target"
            ;;
    esac
    
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "Trivy scan completed successfully"
    else
        log_warning "Trivy scan found security issues (exit code: $exit_code)"
    fi
    
    if [[ -n "$output_file" ]]; then
        log_info "Report saved to: $output_file"
    fi
    
    return $exit_code
}

# Hadolint Dockerfile scanning
run_hadolint_scan() {
    local dockerfile="$1"
    
    if [[ "$HADOLINT_ENABLED" != true ]]; then
        log_info "Hadolint scanning disabled"
        return 0
    fi
    
    log_info "Running Hadolint Dockerfile scan..."
    log_info "Dockerfile: $dockerfile"
    
    local output_file="${REPORTS_DIR}/hadolint-${DATE_STAMP}.json"
    
    if hadolint --format json "$dockerfile" > "$output_file" 2>&1; then
        log_success "Hadolint scan completed successfully"
        
        # Show table format for console
        hadolint --format tty "$dockerfile" || true
    else
        log_warning "Hadolint found issues in Dockerfile"
        
        # Show issues in console
        hadolint "$dockerfile" || true
    fi
    
    log_info "Hadolint report saved to: $output_file"
}

# Docker Bench Security
run_docker_bench() {
    if [[ "$DOCKER_BENCH_SECURITY" != true ]]; then
        log_info "Docker Bench Security disabled"
        return 0
    fi
    
    log_info "Running Docker Bench Security..."
    
    local output_file="${REPORTS_DIR}/docker-bench-${DATE_STAMP}.log"
    
    if docker run --rm --net host --pid host --userns host --cap-add audit_control \
        -e DOCKER_CONTENT_TRUST=$DOCKER_CONTENT_TRUST \
        -v /var/lib:/var/lib \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -v /usr/lib/systemd:/usr/lib/systemd \
        -v /etc:/etc \
        --label docker_bench_security \
        docker/docker-bench-security > "$output_file" 2>&1; then
        log_success "Docker Bench Security completed"
    else
        log_warning "Docker Bench Security found issues"
    fi
    
    log_info "Docker Bench report saved to: $output_file"
    
    # Show summary
    grep -E "(WARN|FAIL)" "$output_file" | head -20 || true
}

# Custom security checks
run_custom_checks() {
    local image="$1"
    
    log_info "Running custom security checks..."
    
    local report_file="${REPORTS_DIR}/custom-checks-${DATE_STAMP}.txt"
    
    {
        echo "=== Custom Security Check Report ==="
        echo "Image: $image"
        echo "Date: $(date)"
        echo "======================================"
        echo
        
        # Check if image runs as root
        echo "1. User and Group Checks:"
        if docker run --rm --entrypoint="" "$image" id 2>/dev/null; then
            echo "   ✓ User information retrieved"
        else
            echo "   ✗ Could not retrieve user information"
        fi
        echo
        
        # Check file permissions
        echo "2. File Permission Checks:"
        if docker run --rm --entrypoint="" "$image" ls -la / 2>/dev/null | head -10; then
            echo "   ✓ Root directory permissions checked"
        else
            echo "   ✗ Could not check file permissions"
        fi
        echo
        
        # Check for sensitive files
        echo "3. Sensitive File Checks:"
        sensitive_files=("/etc/passwd" "/etc/shadow" "/etc/hosts" "/etc/hostname")
        for file in "${sensitive_files[@]}"; do
            if docker run --rm --entrypoint="" "$image" test -f "$file" 2>/dev/null; then
                echo "   ⚠ Found: $file"
            else
                echo "   ✓ Not found: $file"
            fi
        done
        echo
        
        # Check environment variables
        echo "4. Environment Variable Checks:"
        if docker run --rm --entrypoint="" "$image" env 2>/dev/null | grep -v "^PATH\|^HOSTNAME\|^HOME" | head -10; then
            echo "   ✓ Environment variables checked"
        else
            echo "   ✗ Could not check environment variables"
        fi
        echo
        
        # Check capabilities
        echo "5. Capability Checks:"
        if docker run --rm --entrypoint="" "$image" sh -c "capsh --print 2>/dev/null || echo 'capsh not available'"; then
            echo "   ✓ Capabilities checked"
        else
            echo "   ⚠ Could not check capabilities"
        fi
        echo
        
    } > "$report_file"
    
    log_info "Custom checks report saved to: $report_file"
    
    # Show summary
    cat "$report_file"
}

# Generate summary report
generate_summary() {
    log_info "Generating security scan summary..."
    
    local summary_file="${REPORTS_DIR}/security-summary-${DATE_STAMP}.md"
    
    {
        echo "# Nephoran Intent Operator Security Scan Summary"
        echo
        echo "**Date:** $(date)"
        echo "**Scan ID:** ${DATE_STAMP}"
        echo
        echo "## Scan Configuration"
        echo "- **Trivy Severity:** $TRIVY_SEVERITY"
        echo "- **Hadolint Enabled:** $HADOLINT_ENABLED"
        echo "- **Docker Bench Security:** $DOCKER_BENCH_SECURITY"
        echo
        echo "## Reports Generated"
        find "$REPORTS_DIR" -name "*${DATE_STAMP}*" -type f | while read -r file; do
            echo "- [$(basename "$file")]($file)"
        done
        echo
        echo "## Recommendations"
        echo "1. Review all HIGH and CRITICAL vulnerabilities"
        echo "2. Update base images to latest secure versions"
        echo "3. Implement security best practices from reports"
        echo "4. Regular security scanning in CI/CD pipeline"
        echo "5. Monitor for new vulnerabilities in production"
        echo
        echo "## Next Steps"
        echo "1. Address critical security findings"
        echo "2. Update Dockerfile security configurations"
        echo "3. Implement runtime security monitoring"
        echo "4. Schedule regular security reviews"
        echo
    } > "$summary_file"
    
    log_success "Security summary generated: $summary_file"
}

# Main scanning function
main() {
    local target="$1"
    local format="${2:-table}"
    local exit_on_findings="${3:-1}"
    
    log_info "Starting comprehensive security scan"
    log_info "Target: $target"
    
    check_prerequisites
    
    local has_findings=false
    
    case "$target" in
        image:*)
            local image_name="${target#image:}"
            log_info "Scanning container image: $image_name"
            
            if ! run_trivy_scan "$image_name" "image" "$format"; then
                has_findings=true
            fi
            
            run_custom_checks "$image_name"
            ;;
            
        dockerfile:*)
            local dockerfile_path="${target#dockerfile:}"
            log_info "Scanning Dockerfile: $dockerfile_path"
            
            if [[ ! -f "$dockerfile_path" ]]; then
                log_error "Dockerfile not found: $dockerfile_path"
                exit 1
            fi
            
            run_hadolint_scan "$dockerfile_path"
            
            # Also scan the directory for config issues
            if ! run_trivy_scan "$(dirname "$dockerfile_path")" "config" "$format"; then
                has_findings=true
            fi
            ;;
            
        all)
            log_info "Running comprehensive security scan"
            
            # Scan all Dockerfiles
            find "$SCRIPT_DIR" -name "Dockerfile*" -type f | while read -r dockerfile; do
                log_info "Scanning: $dockerfile"
                run_hadolint_scan "$dockerfile"
            done
            
            # Scan filesystem
            if ! run_trivy_scan "$SCRIPT_DIR" "filesystem" "$format"; then
                has_findings=true
            fi
            
            # Run Docker Bench Security
            run_docker_bench
            ;;
            
        *)
            log_error "Unknown target type: $target"
            show_help
            exit 1
            ;;
    esac
    
    # Generate summary
    generate_summary
    
    log_success "Security scanning completed"
    log_info "Reports available in: $REPORTS_DIR"
    
    # Exit with appropriate code
    if [[ "$has_findings" == true && "$exit_on_findings" == "1" ]]; then
        log_warning "Security findings detected. Review reports and address issues."
        exit 1
    fi
}

# Parse command line arguments
FORMAT="table"
EXIT_ON_FINDINGS="1"
TARGET=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --severity)
            TRIVY_SEVERITY="$2"
            shift 2
            ;;
        --format)
            FORMAT="$2"
            shift 2
            ;;
        --output-dir)
            REPORTS_DIR="$2"
            shift 2
            ;;
        --docker-bench)
            DOCKER_BENCH_SECURITY=true
            shift
            ;;
        --no-hadolint)
            HADOLINT_ENABLED=false
            shift
            ;;
        --exit-code)
            EXIT_ON_FINDINGS="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        image:*|dockerfile:*|all)
            TARGET="$1"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

if [[ -z "$TARGET" ]]; then
    log_error "Target is required"
    show_help
    exit 1
fi

# Run main function
main "$TARGET" "$FORMAT" "$EXIT_ON_FINDINGS"