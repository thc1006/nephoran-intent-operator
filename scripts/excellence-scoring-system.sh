#!/bin/bash

# excellence-scoring-system.sh - Overall Excellence Scoring System
# 
# This script aggregates all excellence validation results and provides
# an overall excellence score with executive dashboard, trend analysis,
# and automated recommendations for improvement.

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORTS_DIR="$PROJECT_ROOT/.excellence-reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
DATE=$(date +"%Y-%m-%d")
EXCELLENCE_REPORT="$REPORTS_DIR/excellence_score_$TIMESTAMP.json"
DASHBOARD_FILE="$REPORTS_DIR/excellence_dashboard.json"
TRENDS_FILE="$REPORTS_DIR/excellence_trends.json"
HTML_DASHBOARD="$REPORTS_DIR/excellence_dashboard.html"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Score weights for different categories
DOCS_WEIGHT=20
SECURITY_WEIGHT=25
PERFORMANCE_WEIGHT=20
API_WEIGHT=15
COMMUNITY_WEIGHT=20

# Excellence thresholds
EXCELLENT_THRESHOLD=90
GOOD_THRESHOLD=75
FAIR_THRESHOLD=60

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

log_header() {
    echo -e "${PURPLE}[EXCELLENCE]${NC} $1" >&2
}

# Initialize excellence report structure
init_excellence_report() {
    mkdir -p "$REPORTS_DIR"
    cat > "$EXCELLENCE_REPORT" <<EOF
{
    "timestamp": "$TIMESTAMP",
    "date": "$DATE",
    "project": "nephoran-intent-operator",
    "report_type": "excellence_scoring",
    "version": "1.0",
    "scores": {
        "documentation": {
            "score": 0,
            "weight": $DOCS_WEIGHT,
            "weighted_score": 0,
            "status": "unknown",
            "details": {}
        },
        "security": {
            "score": 0,
            "weight": $SECURITY_WEIGHT,
            "weighted_score": 0,
            "status": "unknown",
            "details": {}
        },
        "performance": {
            "score": 0,
            "weight": $PERFORMANCE_WEIGHT,
            "weighted_score": 0,
            "status": "unknown",
            "details": {}
        },
        "api_specification": {
            "score": 0,
            "weight": $API_WEIGHT,
            "weighted_score": 0,
            "status": "unknown",
            "details": {}
        },
        "community": {
            "score": 0,
            "weight": $COMMUNITY_WEIGHT,
            "weighted_score": 0,
            "status": "unknown",
            "details": {}
        }
    },
    "overall": {
        "total_score": 0,
        "grade": "F",
        "status": "unknown",
        "improvement_needed": 0
    },
    "trends": {
        "score_history": [],
        "trend_direction": "stable",
        "improvement_rate": 0
    },
    "recommendations": [],
    "action_items": [],
    "next_review_date": "$(date -d '+1 week' --iso-8601)"
}
EOF

    # Initialize trends file if it doesn't exist
    if [[ ! -f "$TRENDS_FILE" ]]; then
        cat > "$TRENDS_FILE" <<EOF
{
    "excellence_scores": [],
    "category_scores": {
        "documentation": [],
        "security": [],
        "performance": [],
        "api_specification": [],
        "community": []
    },
    "milestones": [],
    "last_updated": "$TIMESTAMP"
}
EOF
    fi
}

# Run all excellence validation checks
run_excellence_validations() {
    log_header "Running comprehensive excellence validations..."
    
    local validation_results=()
    local failed_validations=()
    
    # Documentation validation
    log_info "Running documentation quality validation..."
    if "$SCRIPT_DIR/validate-docs.sh" --report-dir "$REPORTS_DIR" > /dev/null 2>&1; then
        validation_results+=("docs:success")
        log_success "Documentation validation completed"
    else
        validation_results+=("docs:partial")
        failed_validations+=("Documentation validation had issues")
        log_warn "Documentation validation completed with warnings"
    fi
    
    # Security compliance check
    log_info "Running security compliance check..."
    if "$SCRIPT_DIR/daily-compliance-check.sh" --report-dir "$REPORTS_DIR" --security-only > /dev/null 2>&1; then
        validation_results+=("security:success")
        log_success "Security compliance check completed"
    else
        validation_results+=("security:partial")
        failed_validations+=("Security compliance check had issues")
        log_warn "Security compliance check completed with warnings"
    fi
    
    # Community metrics collection
    log_info "Running community engagement analysis..."
    if "$SCRIPT_DIR/community-metrics.sh" --report-dir "$REPORTS_DIR" > /dev/null 2>&1; then
        validation_results+=("community:success")
        log_success "Community metrics collection completed"
    else
        validation_results+=("community:partial")
        failed_validations+=("Community metrics collection had issues")
        log_warn "Community metrics collection completed with warnings"
    fi
    
    # Run test suite validations if available
    log_info "Running excellence test suite..."
    if command -v go >/dev/null 2>&1 && [[ -d "$PROJECT_ROOT/tests/excellence" ]]; then
        local test_results
        if test_results=$(cd "$PROJECT_ROOT" && go test ./tests/excellence/... -v 2>&1); then
            validation_results+=("tests:success")
            log_success "Excellence test suite passed"
        else
            validation_results+=("tests:partial")
            failed_validations+=("Excellence test suite had failures")
            log_warn "Excellence test suite completed with some failures"
        fi
    else
        log_info "Skipping Go test suite (go not available or tests not found)"
        validation_results+=("tests:skipped")
    fi
    
    if [[ ${#failed_validations[@]} -gt 0 ]]; then
        log_warn "Some validations completed with issues:"
        for issue in "${failed_validations[@]}"; do
            log_warn "  - $issue"
        done
    fi
    
    log_success "All excellence validations completed"
}

# Collect and aggregate scores from reports
collect_scores() {
    log_info "Collecting scores from validation reports..."
    
    local docs_score=0
    local security_score=0
    local performance_score=0
    local api_score=0
    local community_score=0
    
    # Documentation score from latest docs validation report
    local latest_docs_report
    latest_docs_report=$(find "$REPORTS_DIR" -name "docs_validation_*.json" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2- || echo "")
    
    if [[ -n "$latest_docs_report" && -f "$latest_docs_report" ]]; then
        docs_score=$(jq -r '.overall_score // 0' "$latest_docs_report" 2>/dev/null || echo 0)
        log_info "Documentation score: $docs_score from $latest_docs_report"
    else
        log_warn "No documentation validation report found, using default score"
        docs_score=70  # Default reasonable score
    fi
    
    # Security score from latest compliance report
    local latest_compliance_report
    latest_compliance_report=$(find "$REPORTS_DIR" -name "compliance_report_*.json" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2- || echo "")
    
    if [[ -n "$latest_compliance_report" && -f "$latest_compliance_report" ]]; then
        security_score=$(jq -r '.compliance.security.overall_score // 0' "$latest_compliance_report" 2>/dev/null || echo 0)
        log_info "Security score: $security_score from $latest_compliance_report"
        
        # Also extract performance score from compliance report
        local availability
        availability=$(jq -r '.compliance.performance.sla_compliance.availability_percent // 99' "$latest_compliance_report" 2>/dev/null || echo 99)
        performance_score=$availability
        log_info "Performance score: $performance_score (availability-based)"
    else
        log_warn "No compliance report found, using default scores"
        security_score=75
        performance_score=85
    fi
    
    # Community score from latest community metrics report
    local latest_community_report
    latest_community_report=$(find "$REPORTS_DIR" -name "community_metrics_*.json" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2- || echo "")
    
    if [[ -n "$latest_community_report" && -f "$latest_community_report" ]]; then
        # Calculate community score from multiple metrics
        local stars forks contributors
        stars=$(jq -r '.metrics.github_analytics.repository.stars // 0' "$latest_community_report" 2>/dev/null || echo 0)
        forks=$(jq -r '.metrics.github_analytics.repository.forks // 0' "$latest_community_report" 2>/dev/null || echo 0)
        contributors=$(jq -r '.metrics.github_analytics.repository.contributors // 0' "$latest_community_report" 2>/dev/null || echo 0)
        
        # Simple community scoring algorithm
        community_score=$((
            (stars > 50 ? 30 : stars * 30 / 50) +
            (forks > 20 ? 25 : forks * 25 / 20) +
            (contributors > 10 ? 25 : contributors * 25 / 10) +
            20  # Base score for having community metrics
        ))
        
        if [[ $community_score -gt 100 ]]; then community_score=100; fi
        log_info "Community score: $community_score (calculated from engagement metrics)"
    else
        log_warn "No community metrics report found, using default score"
        community_score=65
    fi
    
    # API score from test artifacts
    local latest_api_report
    latest_api_report=$(find "$REPORTS_DIR" -name "api_specification_validation_report.json" -type f -printf '%T@ %p\n' | sort -n | tail -1 | cut -d' ' -f2- 2>/dev/null || echo "")
    
    if [[ -n "$latest_api_report" && -f "$latest_api_report" ]]; then
        api_score=$(jq -r '.results.overall_score // 0' "$latest_api_report" 2>/dev/null || echo 0)
        log_info "API specification score: $api_score from $latest_api_report"
    else
        log_warn "No API specification report found, using default score"
        api_score=80
    fi
    
    # Update excellence report with collected scores
    jq --argjson docs "$docs_score" \
       --argjson security "$security_score" \
       --argjson performance "$performance_score" \
       --argjson api "$api_score" \
       --argjson community "$community_score" \
       --argjson docs_weight "$DOCS_WEIGHT" \
       --argjson security_weight "$SECURITY_WEIGHT" \
       --argjson performance_weight "$PERFORMANCE_WEIGHT" \
       --argjson api_weight "$API_WEIGHT" \
       --argjson community_weight "$COMMUNITY_WEIGHT" \
       '.scores.documentation.score = $docs |
        .scores.documentation.weighted_score = ($docs * $docs_weight / 100) |
        .scores.security.score = $security |
        .scores.security.weighted_score = ($security * $security_weight / 100) |
        .scores.performance.score = $performance |
        .scores.performance.weighted_score = ($performance * $performance_weight / 100) |
        .scores.api_specification.score = $api |
        .scores.api_specification.weighted_score = ($api * $api_weight / 100) |
        .scores.community.score = $community |
        .scores.community.weighted_score = ($community * $community_weight / 100)' \
       "$EXCELLENCE_REPORT" > "$EXCELLENCE_REPORT.tmp" && mv "$EXCELLENCE_REPORT.tmp" "$EXCELLENCE_REPORT"
}

# Calculate overall excellence score and grade
calculate_overall_score() {
    log_info "Calculating overall excellence score..."
    
    # Extract weighted scores
    local docs_weighted security_weighted performance_weighted api_weighted community_weighted
    docs_weighted=$(jq -r '.scores.documentation.weighted_score' "$EXCELLENCE_REPORT")
    security_weighted=$(jq -r '.scores.security.weighted_score' "$EXCELLENCE_REPORT")
    performance_weighted=$(jq -r '.scores.performance.weighted_score' "$EXCELLENCE_REPORT")
    api_weighted=$(jq -r '.scores.api_specification.weighted_score' "$EXCELLENCE_REPORT")
    community_weighted=$(jq -r '.scores.community.weighted_score' "$EXCELLENCE_REPORT")
    
    # Calculate total weighted score
    local total_score
    total_score=$(echo "scale=2; $docs_weighted + $security_weighted + $performance_weighted + $api_weighted + $community_weighted" | bc)
    
    # Determine grade and status
    local grade="F"
    local status="poor"
    local improvement_needed=0
    
    if (( $(echo "$total_score >= $EXCELLENT_THRESHOLD" | bc -l) )); then
        grade="A"
        status="excellent"
        improvement_needed=0
    elif (( $(echo "$total_score >= $GOOD_THRESHOLD" | bc -l) )); then
        grade="B"
        status="good"
        improvement_needed=$(echo "scale=0; $EXCELLENT_THRESHOLD - $total_score" | bc)
    elif (( $(echo "$total_score >= $FAIR_THRESHOLD" | bc -l) )); then
        grade="C"
        status="fair"
        improvement_needed=$(echo "scale=0; $GOOD_THRESHOLD - $total_score" | bc)
    else
        grade="D"
        status="poor"
        improvement_needed=$(echo "scale=0; $FAIR_THRESHOLD - $total_score" | bc)
    fi
    
    # Update excellence report
    jq --argjson total "$total_score" \
       --arg grade "$grade" \
       --arg status "$status" \
       --argjson improvement "$improvement_needed" \
       '.overall.total_score = $total |
        .overall.grade = $grade |
        .overall.status = $status |
        .overall.improvement_needed = $improvement' \
       "$EXCELLENCE_REPORT" > "$EXCELLENCE_REPORT.tmp" && mv "$EXCELLENCE_REPORT.tmp" "$EXCELLENCE_REPORT"
    
    log_success "Overall excellence score calculated: $total_score ($grade - $status)"
}

# Update trends and historical data
update_trends() {
    log_info "Updating excellence trends..."
    
    local total_score
    total_score=$(jq -r '.overall.total_score' "$EXCELLENCE_REPORT")
    
    # Extract individual category scores
    local docs_score security_score performance_score api_score community_score
    docs_score=$(jq -r '.scores.documentation.score' "$EXCELLENCE_REPORT")
    security_score=$(jq -r '.scores.security.score' "$EXCELLENCE_REPORT")
    performance_score=$(jq -r '.scores.performance.score' "$EXCELLENCE_REPORT")
    api_score=$(jq -r '.scores.api_specification.score' "$EXCELLENCE_REPORT")
    community_score=$(jq -r '.scores.community.score' "$EXCELLENCE_REPORT")
    
    # Update trends file
    jq --argjson total "$total_score" \
       --argjson docs "$docs_score" \
       --argjson security "$security_score" \
       --argjson performance "$performance_score" \
       --argjson api "$api_score" \
       --argjson community "$community_score" \
       --arg date "$DATE" \
       --arg timestamp "$TIMESTAMP" \
       '.excellence_scores += [{"date": $date, "timestamp": $timestamp, "score": $total}] |
        .category_scores.documentation += [{"date": $date, "score": $docs}] |
        .category_scores.security += [{"date": $date, "score": $security}] |
        .category_scores.performance += [{"date": $date, "score": $performance}] |
        .category_scores.api_specification += [{"date": $date, "score": $api}] |
        .category_scores.community += [{"date": $date, "score": $community}] |
        .last_updated = $timestamp |
        .excellence_scores = (.excellence_scores | sort_by(.date) | .[-30:]) |
        .category_scores.documentation = (.category_scores.documentation | sort_by(.date) | .[-30:]) |
        .category_scores.security = (.category_scores.security | sort_by(.date) | .[-30:]) |
        .category_scores.performance = (.category_scores.performance | sort_by(.date) | .[-30:]) |
        .category_scores.api_specification = (.category_scores.api_specification | sort_by(.date) | .[-30:]) |
        .category_scores.community = (.category_scores.community | sort_by(.date) | .[-30:])' \
       "$TRENDS_FILE" > "$TRENDS_FILE.tmp" && mv "$TRENDS_FILE.tmp" "$TRENDS_FILE"
    
    # Calculate trend direction and improvement rate
    local trend_direction="stable"
    local improvement_rate=0
    
    local score_history
    score_history=$(jq -r '.excellence_scores[-5:] | map(.score) | @json' "$TRENDS_FILE" 2>/dev/null || echo "[]")
    
    if [[ "$score_history" != "[]" ]]; then
        local scores_array
        scores_array=$(echo "$score_history" | jq -r '.[]' 2>/dev/null || echo "")
        
        if [[ -n "$scores_array" ]]; then
            local score_count
            score_count=$(echo "$score_history" | jq '. | length' 2>/dev/null || echo 0)
            
            if [[ $score_count -gt 1 ]]; then
                local first_score last_score
                first_score=$(echo "$score_history" | jq -r '.[0]' 2>/dev/null || echo 0)
                last_score=$(echo "$score_history" | jq -r '.[-1]' 2>/dev/null || echo 0)
                
                local score_diff
                score_diff=$(echo "scale=2; $last_score - $first_score" | bc 2>/dev/null || echo 0)
                
                if (( $(echo "$score_diff > 5" | bc -l) )); then
                    trend_direction="improving"
                    improvement_rate=$(echo "scale=2; $score_diff / ($score_count - 1)" | bc 2>/dev/null || echo 0)
                elif (( $(echo "$score_diff < -5" | bc -l) )); then
                    trend_direction="declining"
                    improvement_rate=$(echo "scale=2; $score_diff / ($score_count - 1)" | bc 2>/dev/null || echo 0)
                fi
            fi
        fi
    fi
    
    # Update excellence report with trend info
    jq --arg trend "$trend_direction" \
       --argjson rate "$improvement_rate" \
       '.trends.trend_direction = $trend |
        .trends.improvement_rate = $rate |
        .trends.score_history = (.trends.score_history + [{"date": "'$DATE'", "score": .overall.total_score}] | .[-10:])' \
       "$EXCELLENCE_REPORT" > "$EXCELLENCE_REPORT.tmp" && mv "$EXCELLENCE_REPORT.tmp" "$EXCELLENCE_REPORT"
    
    log_success "Excellence trends updated: $trend_direction (rate: $improvement_rate)"
}

# Generate recommendations and action items
generate_recommendations() {
    log_info "Generating improvement recommendations..."
    
    local recommendations=()
    local action_items=()
    
    # Extract scores for analysis
    local docs_score security_score performance_score api_score community_score
    docs_score=$(jq -r '.scores.documentation.score' "$EXCELLENCE_REPORT")
    security_score=$(jq -r '.scores.security.score' "$EXCELLENCE_REPORT")
    performance_score=$(jq -r '.scores.performance.score' "$EXCELLENCE_REPORT")
    api_score=$(jq -r '.scores.api_specification.score' "$EXCELLENCE_REPORT")
    community_score=$(jq -r '.scores.community.score' "$EXCELLENCE_REPORT")
    
    # Documentation recommendations
    if [[ $docs_score -lt $GOOD_THRESHOLD ]]; then
        if [[ $docs_score -lt $FAIR_THRESHOLD ]]; then
            action_items+=("\"URGENT: Improve documentation quality - current score: $docs_score%\"")
        fi
        recommendations+=("\"Enhance documentation completeness and quality (current: $docs_score%)\"")
        recommendations+=("\"Fix broken links and update outdated documentation sections\"")
        recommendations+=("\"Add more comprehensive API documentation and examples\"")
    fi
    
    # Security recommendations
    if [[ $security_score -lt $GOOD_THRESHOLD ]]; then
        if [[ $security_score -lt $FAIR_THRESHOLD ]]; then
            action_items+=("\"CRITICAL: Address security vulnerabilities - current score: $security_score%\"")
        fi
        recommendations+=("\"Improve security compliance measures (current: $security_score%)\"")
        recommendations+=("\"Conduct security audit and vulnerability assessment\"")
        recommendations+=("\"Implement additional security controls and monitoring\"")
    fi
    
    # Performance recommendations
    if [[ $performance_score -lt $GOOD_THRESHOLD ]]; then
        if [[ $performance_score -lt $FAIR_THRESHOLD ]]; then
            action_items+=("\"HIGH: Improve performance to meet SLA requirements - current: $performance_score%\"")
        fi
        recommendations+=("\"Optimize performance to improve SLA compliance (current: $performance_score%)\"")
        recommendations+=("\"Implement performance monitoring and alerting\"")
        recommendations+=("\"Review resource allocation and scaling policies\"")
    fi
    
    # API recommendations
    if [[ $api_score -lt $GOOD_THRESHOLD ]]; then
        if [[ $api_score -lt $FAIR_THRESHOLD ]]; then
            action_items+=("\"HIGH: Improve API specification quality - current score: $api_score%\"")
        fi
        recommendations+=("\"Enhance API specification and documentation (current: $api_score%)\"")
        recommendations+=("\"Validate API schemas and ensure consistency\"")
        recommendations+=("\"Add comprehensive API testing and examples\"")
    fi
    
    # Community recommendations
    if [[ $community_score -lt $GOOD_THRESHOLD ]]; then
        if [[ $community_score -lt $FAIR_THRESHOLD ]]; then
            action_items+=("\"MEDIUM: Improve community engagement - current score: $community_score%\"")
        fi
        recommendations+=("\"Enhance community engagement and documentation (current: $community_score%)\"")
        recommendations+=("\"Create issue templates and contribution guidelines\"")
        recommendations+=("\"Improve project visibility and outreach efforts\"")
    fi
    
    # Overall excellence recommendations
    local total_score
    total_score=$(jq -r '.overall.total_score' "$EXCELLENCE_REPORT")
    
    if (( $(echo "$total_score < $FAIR_THRESHOLD" | bc -l) )); then
        action_items+=("\"URGENT: Overall project excellence needs immediate attention (score: $total_score)\"")
        recommendations+=("\"Focus on foundational improvements across all categories\"")
        recommendations+=("\"Establish regular excellence review and improvement cycles\"")
    elif (( $(echo "$total_score < $GOOD_THRESHOLD" | bc -l) )); then
        recommendations+=("\"Continue steady improvements to achieve good excellence level\"")
        recommendations+=("\"Focus on the lowest-scoring categories first\"")
    elif (( $(echo "$total_score < $EXCELLENT_THRESHOLD" | bc -l) )); then
        recommendations+=("\"Polish remaining areas to achieve excellence status\"")
        recommendations+=("\"Maintain current high standards while addressing minor gaps\"")
    else
        recommendations+=("\"Maintain excellent standards across all categories\"")
        recommendations+=("\"Share best practices with other projects\"")
    fi
    
    # Trend-based recommendations
    local trend_direction
    trend_direction=$(jq -r '.trends.trend_direction' "$EXCELLENCE_REPORT")
    
    case "$trend_direction" in
        "declining")
            action_items+=("\"ATTENTION: Excellence score is declining - investigate root causes\"")
            recommendations+=("\"Identify and address factors causing excellence decline\"")
            ;;
        "improving")
            recommendations+=("\"Continue current improvement trajectory\"")
            recommendations+=("\"Document successful practices for future reference\"")
            ;;
        *)
            recommendations+=("\"Establish improvement initiatives to drive excellence forward\"")
            ;;
    esac
    
    # Update excellence report with recommendations
    jq --argjson recs "[$(IFS=,; echo "${recommendations[*]}")]" \
       --argjson actions "[$(IFS=,; echo "${action_items[*]}")]" \
       '.recommendations = $recs | .action_items = $actions' \
       "$EXCELLENCE_REPORT" > "$EXCELLENCE_REPORT.tmp" && mv "$EXCELLENCE_REPORT.tmp" "$EXCELLENCE_REPORT"
    
    log_success "Generated ${#recommendations[@]} recommendations and ${#action_items[@]} action items"
}

# Create executive dashboard
create_dashboard() {
    log_info "Creating excellence executive dashboard..."
    
    local total_score grade status
    total_score=$(jq -r '.overall.total_score' "$EXCELLENCE_REPORT")
    grade=$(jq -r '.overall.grade' "$EXCELLENCE_REPORT")
    status=$(jq -r '.overall.status' "$EXCELLENCE_REPORT")
    
    # Create dashboard data structure
    cat > "$DASHBOARD_FILE" <<EOF
{
    "timestamp": "$TIMESTAMP",
    "date": "$DATE",
    "project": "nephoran-intent-operator",
    "dashboard_version": "1.0",
    "summary": {
        "overall_score": $total_score,
        "grade": "$grade",
        "status": "$status",
        "trend": "$(jq -r '.trends.trend_direction' "$EXCELLENCE_REPORT")",
        "last_assessment": "$DATE"
    },
    "categories": {
        "documentation": {
            "score": $(jq -r '.scores.documentation.score' "$EXCELLENCE_REPORT"),
            "weight": $(jq -r '.scores.documentation.weight' "$EXCELLENCE_REPORT"),
            "status": "$(jq -r 'if .scores.documentation.score >= 75 then "good" elif .scores.documentation.score >= 60 then "fair" else "needs_attention" end' "$EXCELLENCE_REPORT")"
        },
        "security": {
            "score": $(jq -r '.scores.security.score' "$EXCELLENCE_REPORT"),
            "weight": $(jq -r '.scores.security.weight' "$EXCELLENCE_REPORT"),
            "status": "$(jq -r 'if .scores.security.score >= 75 then "good" elif .scores.security.score >= 60 then "fair" else "needs_attention" end' "$EXCELLENCE_REPORT")"
        },
        "performance": {
            "score": $(jq -r '.scores.performance.score' "$EXCELLENCE_REPORT"),
            "weight": $(jq -r '.scores.performance.weight' "$EXCELLENCE_REPORT"),
            "status": "$(jq -r 'if .scores.performance.score >= 75 then "good" elif .scores.performance.score >= 60 then "fair" else "needs_attention" end' "$EXCELLENCE_REPORT")"
        },
        "api_specification": {
            "score": $(jq -r '.scores.api_specification.score' "$EXCELLENCE_REPORT"),
            "weight": $(jq -r '.scores.api_specification.weight' "$EXCELLENCE_REPORT"),
            "status": "$(jq -r 'if .scores.api_specification.score >= 75 then "good" elif .scores.api_specification.score >= 60 then "fair" else "needs_attention" end' "$EXCELLENCE_REPORT")"
        },
        "community": {
            "score": $(jq -r '.scores.community.score' "$EXCELLENCE_REPORT"),
            "weight": $(jq -r '.scores.community.weight' "$EXCELLENCE_REPORT"),
            "status": "$(jq -r 'if .scores.community.score >= 75 then "good" elif .scores.community.score >= 60 then "fair" else "needs_attention" end' "$EXCELLENCE_REPORT")"
        }
    },
    "alerts": {
        "critical": $(jq '.action_items | [.[] | select(contains("CRITICAL") or contains("URGENT"))] | length' "$EXCELLENCE_REPORT"),
        "high": $(jq '.action_items | [.[] | select(contains("HIGH"))] | length' "$EXCELLENCE_REPORT"),
        "medium": $(jq '.action_items | [.[] | select(contains("MEDIUM"))] | length' "$EXCELLENCE_REPORT")
    },
    "improvements": {
        "needed_for_good": $(jq -r 'if .overall.total_score < 75 then (75 - .overall.total_score) else 0 end' "$EXCELLENCE_REPORT"),
        "needed_for_excellent": $(jq -r 'if .overall.total_score < 90 then (90 - .overall.total_score) else 0 end' "$EXCELLENCE_REPORT")
    },
    "next_actions": $(jq '.action_items[0:5]' "$EXCELLENCE_REPORT"),
    "trends": $(jq '.trends' "$EXCELLENCE_REPORT")
}
EOF
    
    log_success "Executive dashboard created: $DASHBOARD_FILE"
}

# Generate HTML dashboard
generate_html_dashboard() {
    log_info "Generating HTML excellence dashboard..."
    
    local total_score grade status trend
    total_score=$(jq -r '.summary.overall_score' "$DASHBOARD_FILE")
    grade=$(jq -r '.summary.grade' "$DASHBOARD_FILE")
    status=$(jq -r '.summary.status' "$DASHBOARD_FILE")
    trend=$(jq -r '.summary.trend' "$DASHBOARD_FILE")
    
    # Determine status color
    local status_color="#dc3545"  # Red
    case "$status" in
        "excellent") status_color="#28a745" ;;  # Green
        "good") status_color="#ffc107" ;;       # Yellow
        "fair") status_color="#fd7e14" ;;       # Orange
    esac
    
    cat > "$HTML_DASHBOARD" <<EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Excellence Dashboard - Nephoran Intent Operator</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f8f9fa; }
        .dashboard { max-width: 1200px; margin: 0 auto; }
        .header { text-align: center; margin-bottom: 30px; }
        .score-circle { width: 150px; height: 150px; border-radius: 50%; display: inline-flex; align-items: center; justify-content: center; color: white; font-size: 36px; font-weight: bold; background: $status_color; margin: 20px; }
        .grade { font-size: 48px; }
        .status { text-transform: uppercase; letter-spacing: 2px; }
        .categories { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin: 30px 0; }
        .category { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .category-header { display: flex; justify-content: between; align-items: center; margin-bottom: 15px; }
        .category-title { font-weight: 600; font-size: 16px; }
        .category-score { font-size: 24px; font-weight: bold; }
        .progress-bar { width: 100%; height: 8px; background: #e9ecef; border-radius: 4px; overflow: hidden; }
        .progress-fill { height: 100%; transition: width 0.3s ease; }
        .good { background: #28a745; }
        .fair { background: #ffc107; }
        .needs_attention { background: #dc3545; }
        .alerts { background: white; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .alert-item { display: inline-block; margin: 5px 10px; padding: 5px 12px; border-radius: 20px; color: white; font-size: 14px; }
        .critical { background: #dc3545; }
        .high { background: #fd7e14; }
        .medium { background: #ffc107; }
        .recommendations { background: white; padding: 20px; border-radius: 8px; margin: 20px 0; }
        .recommendation { padding: 10px; border-left: 4px solid #007bff; margin: 10px 0; background: #f8f9fa; }
        .footer { text-align: center; margin-top: 40px; color: #6c757d; font-size: 14px; }
    </style>
</head>
<body>
    <div class="dashboard">
        <div class="header">
            <h1>🏆 Excellence Dashboard</h1>
            <h2>Nephoran Intent Operator</h2>
            <div class="score-circle">
                <div>
                    <div class="grade">$grade</div>
                    <div>$total_score</div>
                </div>
            </div>
            <div class="status">$status</div>
            <div>Trend: $trend</div>
            <div style="margin-top: 10px; color: #6c757d;">Last Updated: $DATE</div>
        </div>
        
        <div class="categories">
EOF
    
    # Add category sections
    local categories=("documentation" "security" "performance" "api_specification" "community")
    local category_titles=("📚 Documentation" "🔒 Security" "⚡ Performance" "🔧 API Specification" "👥 Community")
    
    for i in "${!categories[@]}"; do
        local category="${categories[i]}"
        local title="${category_titles[i]}"
        local score weight status_class
        score=$(jq -r ".categories.$category.score" "$DASHBOARD_FILE")
        weight=$(jq -r ".categories.$category.weight" "$DASHBOARD_FILE")
        status_class=$(jq -r ".categories.$category.status" "$DASHBOARD_FILE")
        
        cat >> "$HTML_DASHBOARD" <<EOF
            <div class="category">
                <div class="category-header">
                    <div class="category-title">$title</div>
                    <div class="category-score">$score</div>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill $status_class" style="width: ${score}%"></div>
                </div>
                <div style="margin-top: 10px; font-size: 14px; color: #6c757d;">Weight: ${weight}%</div>
            </div>
EOF
    done
    
    # Add alerts section
    local critical_count high_count medium_count
    critical_count=$(jq -r '.alerts.critical' "$DASHBOARD_FILE")
    high_count=$(jq -r '.alerts.high' "$DASHBOARD_FILE")
    medium_count=$(jq -r '.alerts.medium' "$DASHBOARD_FILE")
    
    cat >> "$HTML_DASHBOARD" <<EOF
        </div>
        
        <div class="alerts">
            <h3>🚨 Alerts & Action Items</h3>
            <span class="alert-item critical">$critical_count Critical</span>
            <span class="alert-item high">$high_count High</span>
            <span class="alert-item medium">$medium_count Medium</span>
        </div>
        
        <div class="recommendations">
            <h3>💡 Top Recommendations</h3>
EOF
    
    # Add top 5 recommendations
    jq -r '.next_actions[]' "$DASHBOARD_FILE" 2>/dev/null | head -5 | while IFS= read -r recommendation; do
        if [[ -n "$recommendation" && "$recommendation" != "null" ]]; then
            echo "            <div class=\"recommendation\">$recommendation</div>" >> "$HTML_DASHBOARD"
        fi
    done
    
    cat >> "$HTML_DASHBOARD" <<EOF
        </div>
        
        <div class="footer">
            <p>Generated by Nephoran Excellence Scoring System on $DATE</p>
            <p>Next assessment scheduled for $(jq -r '.next_review_date' "$EXCELLENCE_REPORT")</p>
        </div>
    </div>
</body>
</html>
EOF
    
    log_success "HTML dashboard generated: $HTML_DASHBOARD"
}

# Generate comprehensive summary report
generate_summary() {
    log_header "Excellence Assessment Summary Report"
    echo ""
    echo "================================================================="
    echo "           🏆 NEPHORAN INTENT OPERATOR EXCELLENCE REPORT"
    echo "================================================================="
    echo ""
    echo "Assessment Date: $DATE"
    echo "Report Generated: $(date)"
    echo ""
    
    local total_score grade status
    total_score=$(jq -r '.overall.total_score' "$EXCELLENCE_REPORT")
    grade=$(jq -r '.overall.grade' "$EXCELLENCE_REPORT")
    status=$(jq -r '.overall.status' "$EXCELLENCE_REPORT")
    
    # Overall Score Display
    echo "🎯 OVERALL EXCELLENCE SCORE"
    echo "─────────────────────────────────────────────────────────────────"
    printf "Score: %.1f/100  |  Grade: %s  |  Status: %s\n" "$total_score" "$grade" "$status"
    echo ""
    
    # Category Breakdown
    echo "📊 CATEGORY BREAKDOWN"
    echo "─────────────────────────────────────────────────────────────────"
    printf "%-20s %8s %8s %12s %10s\n" "Category" "Score" "Weight" "Weighted" "Status"
    echo "─────────────────────────────────────────────────────────────────"
    
    local categories=("documentation" "security" "performance" "api_specification" "community")
    local category_names=("Documentation" "Security" "Performance" "API Specification" "Community")
    
    for i in "${!categories[@]}"; do
        local category="${categories[i]}"
        local name="${category_names[i]}"
        local score weight weighted_score status_icon
        score=$(jq -r ".scores.$category.score" "$EXCELLENCE_REPORT")
        weight=$(jq -r ".scores.$category.weight" "$EXCELLENCE_REPORT")
        weighted_score=$(jq -r ".scores.$category.weighted_score" "$EXCELLENCE_REPORT")
        
        # Status icon
        if (( $(echo "$score >= 75" | bc -l) )); then
            status_icon="✅"
        elif (( $(echo "$score >= 60" | bc -l) )); then
            status_icon="⚠️"
        else
            status_icon="❌"
        fi
        
        printf "%-20s %7.1f %7d%% %10.1f %8s\n" "$name" "$score" "$weight" "$weighted_score" "$status_icon"
    done
    
    echo ""
    
    # Trend Analysis
    echo "📈 TREND ANALYSIS"
    echo "─────────────────────────────────────────────────────────────────"
    local trend_direction improvement_rate
    trend_direction=$(jq -r '.trends.trend_direction' "$EXCELLENCE_REPORT")
    improvement_rate=$(jq -r '.trends.improvement_rate' "$EXCELLENCE_REPORT")
    
    case "$trend_direction" in
        "improving")
            echo "Trend: 📈 Improving (rate: +$improvement_rate points/assessment)"
            ;;
        "declining")  
            echo "Trend: 📉 Declining (rate: $improvement_rate points/assessment)"
            ;;
        *)
            echo "Trend: ➡️ Stable (no significant change)"
            ;;
    esac
    echo ""
    
    # Critical Action Items
    local action_items_count
    action_items_count=$(jq '.action_items | length' "$EXCELLENCE_REPORT")
    
    if [[ $action_items_count -gt 0 ]]; then
        echo "🚨 CRITICAL ACTION ITEMS"
        echo "─────────────────────────────────────────────────────────────────"
        jq -r '.action_items[] | "  • \(.)"' "$EXCELLENCE_REPORT"
        echo ""
    fi
    
    # Key Recommendations
    local recommendations_count
    recommendations_count=$(jq '.recommendations | length' "$EXCELLENCE_REPORT")
    
    if [[ $recommendations_count -gt 0 ]]; then
        echo "💡 KEY RECOMMENDATIONS"
        echo "─────────────────────────────────────────────────────────────────"
        jq -r '.recommendations[0:5][] | "  • \(.)"' "$EXCELLENCE_REPORT"
        echo ""
    fi
    
    # Excellence Status
    echo "🎖️  EXCELLENCE STATUS"
    echo "─────────────────────────────────────────────────────────────────"
    case "$grade" in
        "A")
            echo -e "${GREEN}🌟 EXCELLENT! ${NC}The project demonstrates exceptional quality across all dimensions."
            echo "Continue maintaining these high standards and consider sharing best practices."
            ;;
        "B")
            echo -e "${YELLOW}👍 GOOD WORK! ${NC}The project shows strong quality with room for improvement."
            local needed_for_excellent
            needed_for_excellent=$(jq -r '.overall.improvement_needed' "$EXCELLENCE_REPORT")
            echo "Focus on key areas to achieve excellence (need +$needed_for_excellent points)."
            ;;
        "C")
            echo -e "${YELLOW}⚠️  FAIR QUALITY ${NC}The project meets basic standards but needs attention."
            echo "Prioritize improvements in the lowest-scoring categories."
            ;;
        *)
            echo -e "${RED}❌ NEEDS IMPROVEMENT ${NC}The project requires significant quality improvements."
            echo "Focus on foundational issues before advancing to enhancement features."
            ;;
    esac
    echo ""
    
    # Next Steps
    echo "🎯 NEXT STEPS"
    echo "─────────────────────────────────────────────────────────────────"
    echo "  1. Review detailed reports in: $REPORTS_DIR"
    echo "  2. Address critical action items first"
    echo "  3. Focus on lowest-scoring categories"
    echo "  4. Schedule next assessment for: $(jq -r '.next_review_date' "$EXCELLENCE_REPORT")"
    echo "  5. View HTML dashboard: $HTML_DASHBOARD"
    echo ""
    
    # File Locations
    echo "📁 REPORT FILES"
    echo "─────────────────────────────────────────────────────────────────"
    echo "  Excellence Report: $EXCELLENCE_REPORT"
    echo "  Executive Dashboard: $DASHBOARD_FILE"
    echo "  HTML Dashboard: $HTML_DASHBOARD"
    echo "  Trends Data: $TRENDS_FILE"
    echo ""
    
    echo "================================================================="
    echo "          Excellence Assessment Complete - $(date)"
    echo "================================================================="
    
    # Set exit code based on score
    if (( $(echo "$total_score >= $EXCELLENT_THRESHOLD" | bc -l) )); then
        return 0  # Excellent
    elif (( $(echo "$total_score >= $GOOD_THRESHOLD" | bc -l) )); then
        return 1  # Good (warning)
    elif (( $(echo "$total_score >= $FAIR_THRESHOLD" | bc -l) )); then
        return 2  # Fair (needs attention)
    else
        return 3  # Poor (critical)
    fi
}

# Main execution
main() {
    log_header "Starting Comprehensive Excellence Assessment..."
    
    # Check dependencies
    local missing_deps=()
    for cmd in jq bc; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_deps+=("$cmd")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install missing dependencies and try again."
        exit 1
    fi
    
    # Initialize report structures
    init_excellence_report
    
    # Run all validations
    run_excellence_validations
    
    # Collect and calculate scores  
    collect_scores
    calculate_overall_score
    
    # Update trends and generate insights
    update_trends
    generate_recommendations
    
    # Create dashboards
    create_dashboard
    generate_html_dashboard
    
    # Generate final summary
    generate_summary
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --help, -h         Show this help message"
            echo "  --report-dir DIR   Specify custom report directory (default: .excellence-reports)"
            echo "  --skip-validation  Skip running validation checks (use existing reports)"
            echo "  --html-only        Generate only HTML dashboard"
            echo ""
            echo "Excellence Scoring System for Nephoran Intent Operator"
            echo ""
            echo "This comprehensive system evaluates project excellence across multiple dimensions:"
            echo "  • Documentation Quality (20% weight)"
            echo "  • Security Compliance (25% weight)"  
            echo "  • Performance SLA (20% weight)"
            echo "  • API Specification (15% weight)"
            echo "  • Community Engagement (20% weight)"
            echo ""
            echo "Scoring Scale:"
            echo "  A (90-100): Excellent - Industry-leading quality"
            echo "  B (75-89):  Good - High quality with minor improvements needed"
            echo "  C (60-74):  Fair - Acceptable quality, focus on key areas"
            echo "  D (0-59):   Poor - Significant improvements required"
            echo ""
            echo "Output Files:"
            echo "  • excellence_score_TIMESTAMP.json - Detailed scoring report"
            echo "  • excellence_dashboard.json - Executive dashboard data"
            echo "  • excellence_dashboard.html - Visual HTML dashboard"
            echo "  • excellence_trends.json - Historical trend data"
            exit 0
            ;;
        --report-dir)
            REPORTS_DIR="$2"
            EXCELLENCE_REPORT="$REPORTS_DIR/excellence_score_$TIMESTAMP.json"
            DASHBOARD_FILE="$REPORTS_DIR/excellence_dashboard.json"
            TRENDS_FILE="$REPORTS_DIR/excellence_trends.json"
            HTML_DASHBOARD="$REPORTS_DIR/excellence_dashboard.html"
            shift 2
            ;;
        --skip-validation)
            SKIP_VALIDATION=true
            shift
            ;;
        --html-only)
            HTML_ONLY=true
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Execute main function
if [[ "${HTML_ONLY:-false}" == "true" ]]; then
    log_info "Generating HTML dashboard only..."
    init_excellence_report
    collect_scores
    calculate_overall_score
    create_dashboard
    generate_html_dashboard
    log_success "HTML dashboard generated: $HTML_DASHBOARD"
else
    main "$@"
fi