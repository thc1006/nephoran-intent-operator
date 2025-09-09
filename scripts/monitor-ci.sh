#!/bin/bash

# Ultra-Fast CI Monitoring Script for PR 177
# Continuous monitoring with 30-minute timeout and 30-second intervals

set -euo pipefail

PR_NUMBER=177
TIMEOUT_MINUTES=30
CHECK_INTERVAL=30
MAX_CHECKS=$((TIMEOUT_MINUTES * 60 / CHECK_INTERVAL))
CHECK_COUNT=0

echo "🚀 ULTRA-FAST CI MONITORING STARTED FOR PR #${PR_NUMBER}"
echo "⏱️  Timeout: ${TIMEOUT_MINUTES} minutes"
echo "🔄 Check interval: ${CHECK_INTERVAL} seconds"
echo "📊 Max checks: ${MAX_CHECKS}"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

log_with_timestamp() {
    echo -e "[$(date '+%H:%M:%S')] $1"
}

get_ci_status() {
    gh pr checks $PR_NUMBER --json name,status,conclusion,detailsUrl
}

format_status() {
    local status="$1"
    local conclusion="$2"
    
    case "$status" in
        "completed")
            case "$conclusion" in
                "success") echo -e "${GREEN}✅ SUCCESS${NC}" ;;
                "failure") echo -e "${RED}❌ FAILURE${NC}" ;;
                "cancelled") echo -e "${YELLOW}⚠️  CANCELLED${NC}" ;;
                "skipped") echo -e "${BLUE}⏭️  SKIPPED${NC}" ;;
                "neutral") echo -e "${PURPLE}➖ NEUTRAL${NC}" ;;
                *) echo -e "${YELLOW}❓ $conclusion${NC}" ;;
            esac
            ;;
        "in_progress") echo -e "${BLUE}🔄 IN PROGRESS${NC}" ;;
        "queued") echo -e "${YELLOW}⏳ QUEUED${NC}" ;;
        "pending") echo -e "${YELLOW}⏳ PENDING${NC}" ;;
        *) echo -e "${PURPLE}❓ $status${NC}" ;;
    esac
}

check_ci_status() {
    local checks_json=$(get_ci_status)
    local total_checks=$(echo "$checks_json" | jq length)
    local completed_checks=0
    local success_checks=0
    local failed_checks=0
    local in_progress_checks=0
    local queued_checks=0
    
    log_with_timestamp "📋 CI STATUS CHECK #$((CHECK_COUNT + 1))"
    echo "────────────────────────────────────────────────"
    
    if [ "$total_checks" -eq 0 ]; then
        log_with_timestamp "${YELLOW}⚠️  No CI checks found yet...${NC}"
        return 1
    fi
    
    # Process each check
    echo "$checks_json" | jq -r '.[] | "\(.name)|\(.status)|\(.conclusion // "null")|\(.detailsUrl)"' | while IFS='|' read -r name status conclusion url; do
        local formatted_status=$(format_status "$status" "$conclusion")
        printf "%-40s %s\n" "$name" "$formatted_status"
        
        # Track stats
        case "$status" in
            "completed")
                completed_checks=$((completed_checks + 1))
                case "$conclusion" in
                    "success") success_checks=$((success_checks + 1)) ;;
                    "failure"|"cancelled") failed_checks=$((failed_checks + 1)) ;;
                esac
                ;;
            "in_progress") in_progress_checks=$((in_progress_checks + 1)) ;;
            "queued"|"pending") queued_checks=$((queued_checks + 1)) ;;
        esac
    done
    
    # Calculate stats from the JSON (since while loop runs in subshell)
    completed_checks=$(echo "$checks_json" | jq '[.[] | select(.status == "completed")] | length')
    success_checks=$(echo "$checks_json" | jq '[.[] | select(.status == "completed" and .conclusion == "success")] | length')
    failed_checks=$(echo "$checks_json" | jq '[.[] | select(.status == "completed" and (.conclusion == "failure" or .conclusion == "cancelled"))] | length')
    in_progress_checks=$(echo "$checks_json" | jq '[.[] | select(.status == "in_progress")] | length')
    queued_checks=$(echo "$checks_json" | jq '[.[] | select(.status == "queued" or .status == "pending")] | length')
    
    echo "────────────────────────────────────────────────"
    log_with_timestamp "📊 SUMMARY: ${GREEN}${success_checks}✅${NC} ${RED}${failed_checks}❌${NC} ${BLUE}${in_progress_checks}🔄${NC} ${YELLOW}${queued_checks}⏳${NC} (${completed_checks}/${total_checks} complete)"
    
    # Check for failures
    if [ "$failed_checks" -gt 0 ]; then
        log_with_timestamp "${RED}🚨 ALERT: $failed_checks job(s) FAILED! Immediate action required!${NC}"
        echo "$checks_json" | jq -r '.[] | select(.status == "completed" and (.conclusion == "failure" or .conclusion == "cancelled")) | "❌ FAILED: \(.name) - \(.detailsUrl)"'
        return 2
    fi
    
    # Check if all completed successfully
    if [ "$completed_checks" -eq "$total_checks" ] && [ "$success_checks" -eq "$total_checks" ]; then
        log_with_timestamp "${GREEN}🎉 SUCCESS: ALL CI JOBS PASSED! 🎉${NC}"
        return 0
    fi
    
    # Still in progress
    return 1
}

# Main monitoring loop
log_with_timestamp "🔍 Starting continuous CI monitoring..."

while [ $CHECK_COUNT -lt $MAX_CHECKS ]; do
    CHECK_COUNT=$((CHECK_COUNT + 1))
    
    if check_ci_status; then
        # All checks passed
        log_with_timestamp "${GREEN}🏆 MONITORING COMPLETE: ALL CI JOBS SUCCESSFUL! 🏆${NC}"
        exit 0
    elif [ $? -eq 2 ]; then
        # Failures detected
        log_with_timestamp "${RED}💥 FAILURES DETECTED - CONTINUING MONITORING FOR FIXES...${NC}"
    fi
    
    # Calculate remaining time
    remaining_checks=$((MAX_CHECKS - CHECK_COUNT))
    remaining_minutes=$((remaining_checks * CHECK_INTERVAL / 60))
    
    if [ $remaining_checks -gt 0 ]; then
        log_with_timestamp "⏰ Next check in ${CHECK_INTERVAL}s (${remaining_minutes}m remaining, check ${CHECK_COUNT}/${MAX_CHECKS})"
        sleep $CHECK_INTERVAL
    fi
done

log_with_timestamp "${YELLOW}⏰ TIMEOUT REACHED: Monitoring stopped after ${TIMEOUT_MINUTES} minutes${NC}"
log_with_timestamp "📋 Final status check..."
check_ci_status || true

exit 1