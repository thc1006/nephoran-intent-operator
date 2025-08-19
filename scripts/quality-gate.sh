#!/bin/bash
# Comprehensive code quality gate enforcement script
# Maintains 90%+ test coverage and enforces quality standards

set -euo pipefail

# Configuration
COVERAGE_THRESHOLD=90
QUALITY_THRESHOLD=8.0
REPORTS_DIR=".quality-reports"
TEMP_DIR=$(mktemp -d)
EXIT_CODE=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${CYAN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

info() {
    echo -e "${BLUE}[INFO] $1${NC}"
}

# Cleanup function
cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Usage information
show_help() {
    cat << EOF
Usage: $0 [OPTIONS]

Comprehensive code quality gate enforcement for Nephoran Intent Operator

OPTIONS:
    --coverage-threshold=N    Minimum test coverage percentage (default: 90)
    --quality-threshold=N     Minimum quality score (default: 8.0)
    --reports-dir=DIR         Directory for reports (default: .quality-reports)
    --fix                     Attempt to automatically fix issues
    --ci                      Run in CI mode (stricter checks, fail fast)
    --skip-tests              Skip test execution (use existing coverage)
    --skip-lint               Skip linting checks
    --skip-security           Skip security scanning
    --verbose                 Enable verbose output
    --help                    Show this help message

EXAMPLES:
    # Run all quality checks with default thresholds
    $0

    # Run in CI mode with custom coverage threshold
    $0 --ci --coverage-threshold=95

    # Fix issues automatically and generate reports
    $0 --fix --verbose

    # Quick check skipping time-intensive tests
    $0 --skip-tests --skip-security
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --coverage-threshold=*)
                COVERAGE_THRESHOLD="${1#*=}"
                shift
                ;;
            --quality-threshold=*)
                QUALITY_THRESHOLD="${1#*=}"
                shift
                ;;
            --reports-dir=*)
                REPORTS_DIR="${1#*=}"
                shift
                ;;
            --fix)
                FIX_ISSUES=true
                shift
                ;;
            --ci)
                CI_MODE=true
                shift
                ;;
            --skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            --skip-lint)
                SKIP_LINT=true
                shift
                ;;
            --skip-security)
                SKIP_SECURITY=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Initialize environment
init_environment() {
    log "Initializing quality gate environment..."
    
    # Create reports directory
    mkdir -p "$REPORTS_DIR"
    mkdir -p "$REPORTS_DIR/coverage"
    mkdir -p "$REPORTS_DIR/lint"
    mkdir -p "$REPORTS_DIR/security"
    mkdir -p "$REPORTS_DIR/metrics"
    
    # Verify required tools
    local required_tools=("go" "git" "jq")
    
    if [[ "${SKIP_LINT:-false}" != "true" ]]; then
        required_tools+=("golangci-lint")
    fi
    
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "Required tool '$tool' not found"
            if [[ "$tool" == "golangci-lint" ]]; then
                info "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
            elif [[ "$tool" == "jq" ]]; then
                info "Install jq: https://jqlang.github.io/jq/download/"
                info "Ubuntu/Debian: apt-get install jq"
                info "macOS: brew install jq"
                info "Windows: choco install jq"
            fi
            exit 1
        fi
    done
    
    success "Environment initialized successfully"
}

# Run test coverage analysis
run_coverage_analysis() {
    if [[ "${SKIP_TESTS:-false}" == "true" ]]; then
        log "Skipping test coverage analysis"
        return 0
    fi
    
    log "Running comprehensive test coverage analysis..."
    
    local coverage_file="$REPORTS_DIR/coverage/coverage.out"
    local coverage_html="$REPORTS_DIR/coverage/coverage.html"
    local coverage_json="$REPORTS_DIR/coverage/coverage.json"
    
    # Run tests with coverage
    if [[ "${CI_MODE:-false}" == "true" ]]; then
        info "Running tests in CI mode (parallel execution disabled for stability)"
        go test -v -race -coverprofile="$coverage_file" -covermode=atomic -timeout=30m ./... 2>&1 | tee "$REPORTS_DIR/coverage/test-output.log"
    else
        info "Running tests with race detection and coverage"
        go test -v -race -coverprofile="$coverage_file" -covermode=atomic -parallel=4 ./... 2>&1 | tee "$REPORTS_DIR/coverage/test-output.log"
    fi
    
    if [[ $? -ne 0 ]]; then
        error "Tests failed"
        EXIT_CODE=1
        return 1
    fi
    
    # Generate HTML coverage report
    go tool cover -html="$coverage_file" -o "$coverage_html"
    
    # Calculate coverage percentage
    local coverage_percent
    coverage_percent=$(go tool cover -func="$coverage_file" | tail -1 | awk '{print $3}' | sed 's/%//')
    
    # Generate detailed coverage analysis
    cat > "$REPORTS_DIR/coverage/analysis.md" << EOF
# Test Coverage Analysis Report

## Summary
- **Coverage**: ${coverage_percent}%
- **Threshold**: ${COVERAGE_THRESHOLD}%
- **Status**: $([ "${coverage_percent%.*}" -ge "$COVERAGE_THRESHOLD" ] && echo "✅ PASSED" || echo "❌ FAILED")
- **Generated**: $(date -u '+%Y-%m-%d %H:%M:%S UTC')

## Detailed Coverage by Package
EOF
    
    # Add per-package coverage
    go tool cover -func="$coverage_file" | head -n -1 | awk '{print "- " $1 ": " $3}' >> "$REPORTS_DIR/coverage/analysis.md"
    
    # Generate JSON report for programmatic access
    cat > "$coverage_json" << EOF
{
  "timestamp": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "coverage_percent": ${coverage_percent},
  "threshold": ${COVERAGE_THRESHOLD},
  "status": "$([ "${coverage_percent%.*}" -ge "$COVERAGE_THRESHOLD" ] && echo "PASSED" || echo "FAILED")",
  "reports": {
    "html": "coverage.html",
    "text": "analysis.md",
    "raw": "coverage.out"
  }
}
EOF
    
    # Check coverage threshold
    if [[ "${coverage_percent%.*}" -ge "$COVERAGE_THRESHOLD" ]]; then
        success "Coverage check PASSED: ${coverage_percent}% >= ${COVERAGE_THRESHOLD}%"
    else
        error "Coverage check FAILED: ${coverage_percent}% < ${COVERAGE_THRESHOLD}%"
        EXIT_CODE=1
    fi
    
    info "Coverage reports generated in $REPORTS_DIR/coverage/"
}

# Run code quality linting
run_lint_analysis() {
    if [[ "${SKIP_LINT:-false}" == "true" ]]; then
        log "Skipping lint analysis"
        return 0
    fi
    
    log "Running comprehensive code quality analysis..."
    
    local lint_report="$REPORTS_DIR/lint/golangci-lint-report.json"
    local lint_summary="$REPORTS_DIR/lint/summary.md"
    
    # Run golangci-lint with comprehensive configuration
    if golangci-lint run --out-format=json --issues-exit-code=0 > "$lint_report" 2>&1; then
        local issue_count
        issue_count=$(jq '.Issues | length' "$lint_report" 2>/dev/null || echo "0")
        
        # Generate summary report
        cat > "$lint_summary" << EOF
# Code Quality Lint Analysis

## Summary
- **Issues Found**: ${issue_count}
- **Status**: $([ "$issue_count" -eq 0 ] && echo "✅ PASSED" || echo "⚠️  NEEDS ATTENTION")
- **Generated**: $(date -u '+%Y-%m-%d %H:%M:%S UTC')

## Linter Configuration
- Configuration file: \`.golangci.yml\`
- Enabled linters: $(golangci-lint linters | grep -E "^\s*\w+:" | wc -l) linters
- Quality threshold: Strict mode enabled

EOF
        
        if [[ "$issue_count" -gt 0 ]]; then
            echo "## Issues by Category" >> "$lint_summary"
            jq -r '.Issues | group_by(.FromLinter) | .[] | "- " + .[0].FromLinter + ": " + (length | tostring) + " issues"' "$lint_report" >> "$lint_summary" 2>/dev/null || true
            
            warning "Found $issue_count code quality issues"
            if [[ "${FIX_ISSUES:-false}" == "true" ]]; then
                info "Attempting to auto-fix issues..."
                golangci-lint run --fix 2>&1 | tee "$REPORTS_DIR/lint/autofix.log" || true
            fi
            
            if [[ "${CI_MODE:-false}" == "true" ]]; then
                error "Code quality issues found in CI mode"
                EXIT_CODE=1
            fi
        else
            success "No code quality issues found"
        fi
        
    else
        error "golangci-lint execution failed"
        EXIT_CODE=1
    fi
    
    info "Lint reports generated in $REPORTS_DIR/lint/"
}

# Run security analysis
run_security_analysis() {
    if [[ "${SKIP_SECURITY:-false}" == "true" ]]; then
        log "Skipping security analysis"
        return 0
    fi
    
    log "Running security analysis..."
    
    local security_dir="$REPORTS_DIR/security"
    local vuln_report="$security_dir/vulnerabilities.json"
    local gosec_report="$security_dir/gosec-report.json"
    
    # Vulnerability scanning with govulncheck
    if command -v govulncheck &> /dev/null; then
        info "Running vulnerability scan..."
        if govulncheck -json ./... > "$vuln_report" 2>&1; then
            local vuln_count
            vuln_count=$(jq '[.Vulns[]? // empty] | length' "$vuln_report" 2>/dev/null || echo "0")
            
            if [[ "$vuln_count" -gt 0 ]]; then
                warning "Found $vuln_count vulnerabilities"
                jq -r '.Vulns[]? | "- " + .Symbol + " (" + .ID + "): " + .Description' "$vuln_report" > "$security_dir/vulnerabilities.txt" 2>/dev/null || true
                if [[ "${CI_MODE:-false}" == "true" ]]; then
                    EXIT_CODE=1
                fi
            else
                success "No vulnerabilities found"
            fi
        else
            warning "Vulnerability scan failed or no vulnerabilities detected"
        fi
    else
        warning "govulncheck not available, installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi
    
    # Security analysis with gosec
    if command -v gosec &> /dev/null; then
        info "Running gosec security analysis..."
        gosec -fmt=json -out="$gosec_report" ./... 2>/dev/null || true
    else
        info "Installing gosec for security analysis..."
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        gosec -fmt=json -out="$gosec_report" ./... 2>/dev/null || true
    fi
    
    # Generate security summary
    cat > "$security_dir/summary.md" << EOF
# Security Analysis Report

## Vulnerability Scan
- **Tool**: govulncheck
- **Vulnerabilities**: $(jq '[.Vulns[]? // empty] | length' "$vuln_report" 2>/dev/null || echo "N/A")
- **Status**: $([ "$(jq '[.Vulns[]? // empty] | length' "$vuln_report" 2>/dev/null || echo "1")" -eq 0 ] && echo "✅ CLEAN" || echo "⚠️  ISSUES FOUND")

## Static Security Analysis
- **Tool**: gosec
- **Issues**: $(jq '.Issues | length' "$gosec_report" 2>/dev/null || echo "N/A")
- **Confidence Level**: High

## Generated Reports
- Vulnerabilities: \`vulnerabilities.json\`
- Security Issues: \`gosec-report.json\`
- Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
EOF
    
    info "Security reports generated in $security_dir/"
}

# Calculate code quality metrics
calculate_quality_metrics() {
    log "Calculating comprehensive quality metrics..."
    
    local metrics_dir="$REPORTS_DIR/metrics"
    local metrics_json="$metrics_dir/quality-metrics.json"
    
    # Cyclomatic complexity analysis
    info "Analyzing cyclomatic complexity..."
    local complexity_data
    
    # Ensure gocyclo is available
    if ! command -v gocyclo &> /dev/null; then
        info "Installing gocyclo for cyclomatic complexity analysis..."
        go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
    fi
    
    complexity_data=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" -not -name "*_test.go" -not -name "*generated*" | \
        xargs gocyclo -over 10 2>/dev/null | \
        awk '{complexity+=$1; files++} END {if(files>0) print complexity/files; else print 0}')
    complexity_data=${complexity_data:-0}
    
    # Line count analysis
    local total_lines
    local code_lines
    local test_lines
    local comment_lines
    
    total_lines=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs wc -l | tail -1 | awk '{print $1}')
    code_lines=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" -not -name "*_test.go" | xargs wc -l | tail -1 | awk '{print $1}')
    test_lines=$(find . -name "*_test.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs wc -l | tail -1 | awk '{print $1}')
    comment_lines=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs grep -c "^[[:space:]]*//\|^[[:space:]]*\*" | awk -F: '{sum+=$2} END {print sum}')
    
    total_lines=${total_lines:-0}
    code_lines=${code_lines:-0}
    test_lines=${test_lines:-0}
    comment_lines=${comment_lines:-0}
    
    # Package count
    local package_count
    package_count=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | xargs grep -l "^package " | xargs grep "^package " | cut -d: -f2 | sort -u | wc -l)
    
    # Test coverage from previous analysis
    local coverage_percent=0
    if [[ -f "$REPORTS_DIR/coverage/coverage.json" ]]; then
        coverage_percent=$(jq -r '.coverage_percent' "$REPORTS_DIR/coverage/coverage.json" 2>/dev/null || echo "0")
    fi
    
    # Code duplication analysis (basic)
    local duplication_percent=0
    if command -v duplo &> /dev/null || go install github.com/maruel/duplo@latest; then
        duplication_percent=$(duplo -d . 2>/dev/null | grep -o "Duplication: [0-9.]*%" | head -1 | grep -o "[0-9.]*" || echo "0")
    fi
    
    # Calculate overall quality score (0-10 scale)
    local quality_score
    quality_score=$(awk "BEGIN {
        coverage_score = ($coverage_percent >= 90) ? 3 : ($coverage_percent * 3 / 90);
        complexity_score = ($complexity_data <= 5) ? 2 : (2 - (($complexity_data - 5) * 2 / 10));
        if (complexity_score < 0) complexity_score = 0;
        duplication_score = ($duplication_percent <= 5) ? 2 : (2 - (($duplication_percent - 5) * 2 / 15));
        if (duplication_score < 0) duplication_score = 0;
        test_ratio_score = (($test_lines / ($code_lines + 1)) >= 0.5) ? 2 : (($test_lines / ($code_lines + 1)) * 2 / 0.5);
        comment_ratio_score = (($comment_lines / ($code_lines + 1)) >= 0.2) ? 1 : (($comment_lines / ($code_lines + 1)) * 1 / 0.2);
        total = coverage_score + complexity_score + duplication_score + test_ratio_score + comment_ratio_score;
        printf \"%.1f\", total;
    }")
    
    # Generate comprehensive metrics JSON
    cat > "$metrics_json" << EOF
{
  "timestamp": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "overall_quality_score": $quality_score,
  "quality_threshold": $QUALITY_THRESHOLD,
  "metrics": {
    "test_coverage": {
      "percentage": $coverage_percent,
      "status": "$([ "${coverage_percent%.*}" -ge "$COVERAGE_THRESHOLD" ] && echo "PASSED" || echo "FAILED")"
    },
    "cyclomatic_complexity": {
      "average": $complexity_data,
      "threshold": 10,
      "status": "$(awk "BEGIN {print ($complexity_data <= 10) ? \"PASSED\" : \"FAILED\"}")"
    },
    "code_duplication": {
      "percentage": $duplication_percent,
      "threshold": 5,
      "status": "$(awk "BEGIN {print ($duplication_percent <= 5) ? \"PASSED\" : \"FAILED\"}")"
    },
    "code_statistics": {
      "total_lines": $total_lines,
      "code_lines": $code_lines,
      "test_lines": $test_lines,
      "comment_lines": $comment_lines,
      "packages": $package_count,
      "test_to_code_ratio": $(awk "BEGIN {printf \"%.2f\", ($test_lines / ($code_lines + 1))}")
    }
  },
  "status": "$(awk "BEGIN {print ($quality_score >= $QUALITY_THRESHOLD) ? \"PASSED\" : \"FAILED\"}"))"
}
EOF
    
    # Generate human-readable summary
    cat > "$metrics_dir/summary.md" << EOF
# Code Quality Metrics Summary

## Overall Quality Score: ${quality_score}/10.0
**Status**: $(awk "BEGIN {print ($quality_score >= $QUALITY_THRESHOLD) ? \"✅ PASSED\" : \"❌ FAILED\"}")
**Threshold**: ${QUALITY_THRESHOLD}/10.0

## Detailed Metrics

### Test Coverage
- **Percentage**: ${coverage_percent}%
- **Status**: $([ "${coverage_percent%.*}" -ge "$COVERAGE_THRESHOLD" ] && echo "✅ PASSED" || echo "❌ FAILED") (threshold: ${COVERAGE_THRESHOLD}%)

### Code Complexity
- **Average Cyclomatic Complexity**: ${complexity_data}
- **Status**: $(awk "BEGIN {print ($complexity_data <= 10) ? \"✅ PASSED\" : \"❌ FAILED\"}") (threshold: ≤10)

### Code Duplication
- **Duplication Percentage**: ${duplication_percent}%
- **Status**: $(awk "BEGIN {print ($duplication_percent <= 5) ? \"✅ PASSED\" : \"❌ FAILED\"}") (threshold: ≤5%)

### Code Statistics
- **Total Lines**: $(printf "%'d" $total_lines)
- **Code Lines**: $(printf "%'d" $code_lines)
- **Test Lines**: $(printf "%'d" $test_lines)
- **Comment Lines**: $(printf "%'d" $comment_lines)
- **Packages**: $package_count
- **Test-to-Code Ratio**: $(awk "BEGIN {printf \"%.2f\", ($test_lines / ($code_lines + 1))}")

## Quality Score Breakdown
- **Test Coverage Score**: $(awk "BEGIN {coverage_score = ($coverage_percent >= 90) ? 3 : ($coverage_percent * 3 / 90); printf \"%.1f/3.0\", coverage_score}")
- **Complexity Score**: $(awk "BEGIN {complexity_score = ($complexity_data <= 5) ? 2 : (2 - (($complexity_data - 5) * 2 / 10)); if (complexity_score < 0) complexity_score = 0; printf \"%.1f/2.0\", complexity_score}")
- **Duplication Score**: $(awk "BEGIN {duplication_score = ($duplication_percent <= 5) ? 2 : (2 - (($duplication_percent - 5) * 2 / 15)); if (duplication_score < 0) duplication_score = 0; printf \"%.1f/2.0\", duplication_score}")
- **Test Ratio Score**: $(awk "BEGIN {test_ratio_score = (($test_lines / ($code_lines + 1)) >= 0.5) ? 2 : (($test_lines / ($code_lines + 1)) * 2 / 0.5); printf \"%.1f/2.0\", test_ratio_score}")
- **Documentation Score**: $(awk "BEGIN {comment_ratio_score = (($comment_lines / ($code_lines + 1)) >= 0.2) ? 1 : (($comment_lines / ($code_lines + 1)) * 1 / 0.2); printf \"%.1f/1.0\", comment_ratio_score}")

Generated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')
EOF
    
    # Check quality threshold
    if awk "BEGIN {exit !($quality_score >= $QUALITY_THRESHOLD)}"; then
        success "Quality score check PASSED: ${quality_score} >= ${QUALITY_THRESHOLD}"
    else
        error "Quality score check FAILED: ${quality_score} < ${QUALITY_THRESHOLD}"
        EXIT_CODE=1
    fi
    
    info "Quality metrics generated in $metrics_dir/"
}

# Generate comprehensive quality dashboard
generate_quality_dashboard() {
    log "Generating quality dashboard..."
    
    local dashboard_html="$REPORTS_DIR/quality-dashboard.html"
    local dashboard_json="$REPORTS_DIR/quality-dashboard.json"
    
    # Aggregate all metrics
    local coverage_percent=0
    local quality_score=0
    local lint_issues=0
    local vulnerabilities=0
    
    if [[ -f "$REPORTS_DIR/coverage/coverage.json" ]]; then
        coverage_percent=$(jq -r '.coverage_percent' "$REPORTS_DIR/coverage/coverage.json" 2>/dev/null || echo "0")
    fi
    
    if [[ -f "$REPORTS_DIR/metrics/quality-metrics.json" ]]; then
        quality_score=$(jq -r '.overall_quality_score' "$REPORTS_DIR/metrics/quality-metrics.json" 2>/dev/null || echo "0")
    fi
    
    if [[ -f "$REPORTS_DIR/lint/golangci-lint-report.json" ]]; then
        lint_issues=$(jq '.Issues | length' "$REPORTS_DIR/lint/golangci-lint-report.json" 2>/dev/null || echo "0")
    fi
    
    if [[ -f "$REPORTS_DIR/security/vulnerabilities.json" ]]; then
        vulnerabilities=$(jq '[.Vulns[]? // empty] | length' "$REPORTS_DIR/security/vulnerabilities.json" 2>/dev/null || echo "0")
    fi
    
    # Generate JSON dashboard data
    cat > "$dashboard_json" << EOF
{
  "timestamp": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "project": "Nephoran Intent Operator",
  "version": "$(git describe --tags --always --dirty 2>/dev/null || echo "unknown")",
  "commit": "$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")",
  "summary": {
    "overall_status": "$([ $EXIT_CODE -eq 0 ] && echo "PASSED" || echo "FAILED")",
    "quality_score": $quality_score,
    "coverage_percent": $coverage_percent,
    "lint_issues": $lint_issues,
    "vulnerabilities": $vulnerabilities
  },
  "thresholds": {
    "coverage": $COVERAGE_THRESHOLD,
    "quality_score": $QUALITY_THRESHOLD
  },
  "reports": {
    "coverage": {
      "html": "coverage/coverage.html",
      "analysis": "coverage/analysis.md",
      "json": "coverage/coverage.json"
    },
    "lint": {
      "summary": "lint/summary.md",
      "json": "lint/golangci-lint-report.json"
    },
    "security": {
      "summary": "security/summary.md",
      "vulnerabilities": "security/vulnerabilities.json",
      "gosec": "security/gosec-report.json"
    },
    "metrics": {
      "summary": "metrics/summary.md",
      "json": "metrics/quality-metrics.json"
    }
  }
}
EOF
    
    # Generate HTML dashboard
    cat > "$dashboard_html" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Code Quality Dashboard - Nephoran Intent Operator</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #f5f7fa; color: #333; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 30px; padding: 20px; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header h1 { color: #2c3e50; margin-bottom: 10px; }
        .header p { color: #7f8c8d; }
        .metrics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .metric-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); text-align: center; }
        .metric-value { font-size: 2.5em; font-weight: bold; margin: 10px 0; }
        .metric-label { color: #7f8c8d; font-size: 0.9em; text-transform: uppercase; letter-spacing: 1px; }
        .status-passed { color: #27ae60; }
        .status-failed { color: #e74c3c; }
        .status-warning { color: #f39c12; }
        .progress-bar { width: 100%; height: 8px; background: #ecf0f1; border-radius: 4px; overflow: hidden; margin: 10px 0; }
        .progress-fill { height: 100%; transition: width 0.3s ease; }
        .progress-success { background: #27ae60; }
        .progress-warning { background: #f39c12; }
        .progress-danger { background: #e74c3c; }
        .reports-section { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .reports-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-top: 20px; }
        .report-link { display: block; padding: 15px; background: #ecf0f1; border-radius: 6px; text-decoration: none; color: #2c3e50; transition: all 0.3s ease; }
        .report-link:hover { background: #d5dbdb; transform: translateY(-2px); }
        .timestamp { text-align: center; margin-top: 30px; color: #7f8c8d; font-size: 0.9em; }
        .badge { display: inline-block; padding: 4px 12px; border-radius: 20px; font-size: 0.8em; font-weight: bold; text-transform: uppercase; }
        .badge-success { background: #d5f4e6; color: #27ae60; }
        .badge-danger { background: #fadbd8; color: #e74c3c; }
        .badge-warning { background: #fef9e7; color: #f39c12; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Code Quality Dashboard</h1>
            <p>Nephoran Intent Operator - Comprehensive Quality Metrics</p>
            <div class="badge" id="overall-status">Loading...</div>
        </div>
        
        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">Test Coverage</div>
                <div class="metric-value" id="coverage-value">--</div>
                <div class="progress-bar">
                    <div class="progress-fill" id="coverage-progress"></div>
                </div>
                <div id="coverage-status" class="badge">--</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-label">Quality Score</div>
                <div class="metric-value" id="quality-value">--</div>
                <div class="progress-bar">
                    <div class="progress-fill" id="quality-progress"></div>
                </div>
                <div id="quality-status" class="badge">--</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-label">Lint Issues</div>
                <div class="metric-value" id="lint-value">--</div>
                <div id="lint-status" class="badge">--</div>
            </div>
            
            <div class="metric-card">
                <div class="metric-label">Security Issues</div>
                <div class="metric-value" id="security-value">--</div>
                <div id="security-status" class="badge">--</div>
            </div>
        </div>
        
        <div class="reports-section">
            <h2>Detailed Reports</h2>
            <div class="reports-grid" id="reports-grid">
                <!-- Reports will be populated by JavaScript -->
            </div>
        </div>
        
        <div class="timestamp" id="timestamp">
            Loading...
        </div>
    </div>

    <script>
        // Load dashboard data
        fetch('./quality-dashboard.json')
            .then(response => response.json())
            .then(data => {
                updateDashboard(data);
            })
            .catch(error => {
                console.error('Error loading dashboard data:', error);
                document.getElementById('overall-status').textContent = 'Error Loading Data';
                document.getElementById('overall-status').className = 'badge badge-danger';
            });

        function updateDashboard(data) {
            // Update overall status
            const overallStatus = document.getElementById('overall-status');
            overallStatus.textContent = data.summary.overall_status;
            overallStatus.className = `badge ${data.summary.overall_status === 'PASSED' ? 'badge-success' : 'badge-danger'}`;

            // Update coverage
            const coverageValue = document.getElementById('coverage-value');
            const coverageProgress = document.getElementById('coverage-progress');
            const coverageStatus = document.getElementById('coverage-status');
            
            coverageValue.textContent = data.summary.coverage_percent + '%';
            coverageProgress.style.width = data.summary.coverage_percent + '%';
            
            if (data.summary.coverage_percent >= data.thresholds.coverage) {
                coverageProgress.className = 'progress-fill progress-success';
                coverageStatus.textContent = 'PASSED';
                coverageStatus.className = 'badge badge-success';
            } else {
                coverageProgress.className = 'progress-fill progress-danger';
                coverageStatus.textContent = 'FAILED';
                coverageStatus.className = 'badge badge-danger';
            }

            // Update quality score
            const qualityValue = document.getElementById('quality-value');
            const qualityProgress = document.getElementById('quality-progress');
            const qualityStatus = document.getElementById('quality-status');
            
            qualityValue.textContent = data.summary.quality_score + '/10';
            qualityProgress.style.width = (data.summary.quality_score / 10 * 100) + '%';
            
            if (data.summary.quality_score >= data.thresholds.quality_score) {
                qualityProgress.className = 'progress-fill progress-success';
                qualityStatus.textContent = 'PASSED';
                qualityStatus.className = 'badge badge-success';
            } else {
                qualityProgress.className = 'progress-fill progress-danger';
                qualityStatus.textContent = 'FAILED';
                qualityStatus.className = 'badge badge-danger';
            }

            // Update lint issues
            const lintValue = document.getElementById('lint-value');
            const lintStatus = document.getElementById('lint-status');
            
            lintValue.textContent = data.summary.lint_issues;
            if (data.summary.lint_issues === 0) {
                lintStatus.textContent = 'CLEAN';
                lintStatus.className = 'badge badge-success';
            } else {
                lintStatus.textContent = 'ISSUES';
                lintStatus.className = 'badge badge-warning';
            }

            // Update security issues
            const securityValue = document.getElementById('security-value');
            const securityStatus = document.getElementById('security-status');
            
            securityValue.textContent = data.summary.vulnerabilities;
            if (data.summary.vulnerabilities === 0) {
                securityStatus.textContent = 'SECURE';
                securityStatus.className = 'badge badge-success';
            } else {
                securityStatus.textContent = 'VULNERABLE';
                securityStatus.className = 'badge badge-danger';
            }

            // Update reports
            const reportsGrid = document.getElementById('reports-grid');
            reportsGrid.innerHTML = '';
            
            Object.entries(data.reports).forEach(([category, reports]) => {
                Object.entries(reports).forEach(([type, path]) => {
                    const link = document.createElement('a');
                    link.href = path;
                    link.className = 'report-link';
                    link.textContent = `${category.charAt(0).toUpperCase() + category.slice(1)} - ${type.charAt(0).toUpperCase() + type.slice(1)}`;
                    reportsGrid.appendChild(link);
                });
            });

            // Update timestamp
            document.getElementById('timestamp').textContent = `Generated: ${new Date(data.timestamp).toLocaleString()}`;
        }
    </script>
</body>
</html>
EOF
    
    success "Quality dashboard generated: $dashboard_html"
}

# Main execution function
main() {
    local start_time=$(date +%s)
    
    log "Starting comprehensive quality gate analysis..."
    log "Coverage threshold: ${COVERAGE_THRESHOLD}%"
    log "Quality threshold: ${QUALITY_THRESHOLD}/10.0"
    
    # Initialize environment
    init_environment
    
    # Run all quality checks
    run_coverage_analysis
    run_lint_analysis
    run_security_analysis
    calculate_quality_metrics
    generate_quality_dashboard
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Final report
    echo ""
    echo "============================================================================="
    echo "                    QUALITY GATE ANALYSIS COMPLETE"
    echo "============================================================================="
    echo ""
    
    if [[ $EXIT_CODE -eq 0 ]]; then
        success "✅ ALL QUALITY GATES PASSED"
    else
        error "❌ QUALITY GATE FAILURES DETECTED"
        
        if [[ "${CI_MODE:-false}" == "true" ]]; then
            error "Failing CI build due to quality gate failures"
        fi
    fi
    
    echo ""
    echo "Duration: ${duration}s"
    echo "Reports: $REPORTS_DIR/"
    echo "Dashboard: $REPORTS_DIR/quality-dashboard.html"
    echo "============================================================================="
    
    exit $EXIT_CODE
}

# Parse arguments and run main function
FIX_ISSUES=false
CI_MODE=false
SKIP_TESTS=false
SKIP_LINT=false
SKIP_SECURITY=false
VERBOSE=false

parse_args "$@"
main