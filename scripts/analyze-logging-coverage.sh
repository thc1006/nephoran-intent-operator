#!/bin/bash
# Logging Coverage Analysis Script
# Analyzes Go codebase for logging coverage and best practices

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_FILE="${REPO_ROOT}/docs/LOGGING_COVERAGE_REPORT.md"

echo "🔍 Analyzing logging coverage for Nephoran Intent Operator..."
echo "Repository: $REPO_ROOT"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Initialize counters
total_go_files=0
files_with_logging=0
files_without_logging=0
structured_logging_files=0
plain_log_files=0

# Arrays to store file lists
declare -a files_without_logs
declare -a files_with_plain_logs
declare -a critical_files_without_logs

# Critical components that MUST have logging
critical_components=(
    "controllers/networkintent_controller.go"
    "cmd/intent-ingest/main.go"
    "internal/loop/watcher.go"
    "pkg/porch/client.go"
    "pkg/oran/a1/"
)

echo "📊 Scanning Go files..."

# Function to check if file has logging
check_logging() {
    local file=$1
    local has_log=false
    local has_structured=false
    local has_plain=false

    # Check for any logging import
    if grep -q "\"log\"" "$file" || \
       grep -q "github.com/go-logr/logr" "$file" || \
       grep -q "github.com/thc1006/nephoran-intent-operator/pkg/logging" "$file" || \
       grep -q "ctrl.Log" "$file"; then
        has_log=true
    fi

    # Check for structured logging (our new package)
    if grep -q "github.com/thc1006/nephoran-intent-operator/pkg/logging" "$file"; then
        has_structured=true
    # Check for controller-runtime logr
    elif grep -q "github.com/go-logr/logr" "$file" || grep -q "ctrl.Log" "$file"; then
        has_structured=true
    # Check for plain log
    elif grep -q "\"log\"" "$file" && ! grep -q "logr" "$file"; then
        has_plain=true
    fi

    echo "$has_log|$has_structured|$has_plain"
}

# Scan all Go files
while IFS= read -r -d '' file; do
    ((total_go_files++))

    relative_file="${file#$REPO_ROOT/}"

    # Skip vendor, test files for now
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"_test.go" ]]; then
        continue
    fi

    result=$(check_logging "$file")
    IFS='|' read -r has_log has_structured has_plain <<< "$result"

    if [ "$has_log" = "true" ]; then
        ((files_with_logging++))

        if [ "$has_structured" = "true" ]; then
            ((structured_logging_files++))
        fi

        if [ "$has_plain" = "true" ]; then
            ((plain_log_files++))
            files_with_plain_logs+=("$relative_file")
        fi
    else
        ((files_without_logging++))
        files_without_logs+=("$relative_file")

        # Check if it's a critical file
        for critical in "${critical_components[@]}"; do
            if [[ "$relative_file" == *"$critical"* ]]; then
                critical_files_without_logs+=("$relative_file")
            fi
        done
    fi

done < <(find "$REPO_ROOT" -name "*.go" -type f -not -path "*/vendor/*" -print0)

# Calculate percentages
if [ $total_go_files -gt 0 ]; then
    logging_coverage=$((files_with_logging * 100 / total_go_files))
    structured_percentage=$((structured_logging_files * 100 / total_go_files))
else
    logging_coverage=0
    structured_percentage=0
fi

# Generate report
cat > "$OUTPUT_FILE" << EOF
# Logging Coverage Report

**Generated**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
**Repository**: Nephoran Intent Operator
**Analysis Tool**: scripts/analyze-logging-coverage.sh

---

## 📊 Summary

| Metric | Count | Percentage |
|--------|-------|------------|
| **Total Go Files** | $total_go_files | 100% |
| **Files with Logging** | $files_with_logging | ${logging_coverage}% |
| **Files without Logging** | $files_without_logging | $((100 - logging_coverage))% |
| **Structured Logging Files** | $structured_logging_files | ${structured_percentage}% |
| **Plain log.* Files** | $plain_log_files | - |

---

## 🎯 Coverage Goal

- **Current Coverage**: ${logging_coverage}%
- **Target Coverage**: 80%+ for production code
- **Critical Components Coverage**: ${#critical_files_without_logs[@]} critical files missing logs

EOF

# Add status indicator
if [ $logging_coverage -ge 80 ]; then
    echo "- **Status**: ✅ **EXCELLENT** (>80%)" >> "$OUTPUT_FILE"
elif [ $logging_coverage -ge 60 ]; then
    echo "- **Status**: ⚠️ **GOOD** (60-80%)" >> "$OUTPUT_FILE"
elif [ $logging_coverage -ge 40 ]; then
    echo "- **Status**: ⚠️ **NEEDS IMPROVEMENT** (40-60%)" >> "$OUTPUT_FILE"
else
    echo "- **Status**: ❌ **POOR** (<40%)" >> "$OUTPUT_FILE"
fi

cat >> "$OUTPUT_FILE" << EOF

---

## 🔴 Critical Files Without Logging

These files are in critical paths and MUST have logging:

EOF

if [ ${#critical_files_without_logs[@]} -eq 0 ]; then
    echo "✅ All critical files have logging!" >> "$OUTPUT_FILE"
else
    for file in "${critical_files_without_logs[@]}"; do
        echo "- ❌ \`$file\`" >> "$OUTPUT_FILE"
    done
fi

cat >> "$OUTPUT_FILE" << EOF

---

## ⚠️ Files Using Plain log Package

These files should migrate to structured logging:

EOF

if [ ${#files_with_plain_logs[@]} -eq 0 ]; then
    echo "✅ No files using plain log package!" >> "$OUTPUT_FILE"
else
    # Limit to first 20
    count=0
    for file in "${files_with_plain_logs[@]}"; do
        if [ $count -lt 20 ]; then
            echo "- \`$file\`" >> "$OUTPUT_FILE"
            ((count++))
        fi
    done
    if [ ${#files_with_plain_logs[@]} -gt 20 ]; then
        echo "" >> "$OUTPUT_FILE"
        echo "... and $((${#files_with_plain_logs[@]} - 20)) more files" >> "$OUTPUT_FILE"
    fi
fi

cat >> "$OUTPUT_FILE" << EOF

### Migration Priority

1. **High Priority**: Controllers, main entry points
2. **Medium Priority**: Internal packages, utilities
3. **Low Priority**: Example code, test utilities

**Migration Guide**: See \`docs/LOGGING_BEST_PRACTICES.md\`

---

## 📝 Files Without Any Logging

EOF

if [ ${#files_without_logs[@]} -eq 0 ]; then
    echo "✅ All files have logging!" >> "$OUTPUT_FILE"
else
    # Group by directory
    echo "### By Directory" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    current_dir=""
    count=0
    for file in "${files_without_logs[@]}"; do
        if [ $count -lt 30 ]; then
            dir=$(dirname "$file")
            if [ "$dir" != "$current_dir" ]; then
                echo "" >> "$OUTPUT_FILE"
                echo "**$dir/**" >> "$OUTPUT_FILE"
                current_dir="$dir"
            fi
            echo "- \`$(basename "$file")\`" >> "$OUTPUT_FILE"
            ((count++))
        fi
    done

    if [ ${#files_without_logs[@]} -gt 30 ]; then
        echo "" >> "$OUTPUT_FILE"
        echo "... and $((${#files_without_logs[@]} - 30)) more files" >> "$OUTPUT_FILE"
    fi
fi

cat >> "$OUTPUT_FILE" << EOF

---

## 📈 Recommendations

### Immediate Actions

1. **Add logging to critical files**:
   - All controllers MUST have structured logging
   - All HTTP handlers MUST log requests
   - All CLI commands MUST log operations

2. **Migrate from plain log to structured logging**:
   - Replace \`log.Printf()\` with \`logger.InfoEvent()\`
   - Add context fields (namespace, resource name)
   - Use appropriate log levels

3. **Add logging to key operations**:
   - Reconciliation start/end with duration
   - HTTP requests with status codes
   - File processing with filenames
   - External API calls (A1, Porch, RAG)

### Best Practices

1. **Always use structured logging**
   \`\`\`go
   logger := logging.NewLogger(logging.ComponentController)
   logger.InfoEvent("event", "key", "value")
   \`\`\`

2. **Add context to loggers**
   \`\`\`go
   logger = logger.WithNamespace(namespace).WithRequestID(requestID)
   \`\`\`

3. **Use specialized event methods**
   \`\`\`go
   logger.ReconcileStart(namespace, name)
   logger.A1PolicyCreated(policyID, intentType)
   logger.ScalingExecuted(deployment, namespace, from, to)
   \`\`\`

4. **Record durations**
   \`\`\`go
   start := time.Now()
   // ... operation ...
   logger.InfoEvent("operation completed", "durationSeconds", time.Since(start).Seconds())
   \`\`\`

### Long-term Goals

- [ ] Achieve 80%+ logging coverage
- [ ] Migrate all files to structured logging
- [ ] Add logging to all error paths
- [ ] Add logging to all external API calls
- [ ] Implement log sampling for high-frequency logs
- [ ] Set up log aggregation (Loki + Grafana)
- [ ] Create Grafana dashboards for log analytics

---

## 🔍 How to Use This Report

### 1. Review Critical Files

Check the "Critical Files Without Logging" section and add logging immediately.

### 2. Migrate Plain log Usage

For each file in "Files Using Plain log Package":
\`\`\`bash
# Edit the file
vim path/to/file.go

# Follow migration guide in docs/LOGGING_BEST_PRACTICES.md
\`\`\`

### 3. Add Logging to New Code

When writing new code, always:
\`\`\`go
import "github.com/thc1006/nephoran-intent-operator/pkg/logging"

logger := logging.NewLogger(logging.ComponentXXX)
logger.InfoEvent("operation started", "param", value)
\`\`\`

### 4. Re-run Analysis

After making changes:
\`\`\`bash
./scripts/analyze-logging-coverage.sh
\`\`\`

---

## 📚 References

- [Logging Best Practices](./LOGGING_BEST_PRACTICES.md)
- [pkg/logging Documentation](../pkg/logging/)
- [Kubernetes Logging Best Practices](https://kubernetes.io/docs/concepts/cluster-administration/logging/)

---

**Next Analysis**: Run \`./scripts/analyze-logging-coverage.sh\` after adding logging
**Target**: 80%+ coverage for production code
EOF

echo ""
echo "✅ Analysis complete!"
echo ""
echo "📊 Results:"
echo "   Total Go files: $total_go_files"
echo "   Files with logging: $files_with_logging (${logging_coverage}%)"
echo "   Structured logging: $structured_logging_files (${structured_percentage}%)"
echo "   Plain log usage: $plain_log_files files"
echo ""

if [ ${#critical_files_without_logs[@]} -gt 0 ]; then
    echo -e "${RED}❌ ${#critical_files_without_logs[@]} critical files missing logging${NC}"
else
    echo -e "${GREEN}✅ All critical files have logging${NC}"
fi

echo ""
echo "📄 Full report: $OUTPUT_FILE"
echo ""

# Exit with error if coverage is too low
if [ $logging_coverage -lt 40 ]; then
    echo -e "${RED}⚠️  Warning: Logging coverage is below 40%${NC}"
    exit 1
elif [ $logging_coverage -lt 60 ]; then
    echo -e "${YELLOW}⚠️  Warning: Logging coverage is below 60%${NC}"
fi

echo "✅ Logging coverage analysis complete!"
