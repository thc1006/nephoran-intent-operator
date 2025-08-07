#!/bin/bash

# Compliance Validation Automation Script
# Validates mTLS system against O-RAN, NIST, and TLS standards

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORTS_DIR="${SCRIPT_DIR}/reports"
LOG_DIR="${SCRIPT_DIR}/logs"

# Create directories
mkdir -p "$REPORTS_DIR" "$LOG_DIR"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

log_info "Starting compliance validation..."

# Run compliance tests
cd "$SCRIPT_DIR"
if ginkgo --focus="Compliance" ./mtls_compliance_test.go --junit-report="$REPORTS_DIR/compliance-validation.xml" > "$LOG_DIR/compliance.log" 2>&1; then
    log_success "Compliance validation passed"
else
    log_error "Compliance validation failed"
    exit 1
fi

# Generate compliance report
cat > "$REPORTS_DIR/compliance-summary.html" << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Compliance Validation Report</title></head>
<body>
<h1>mTLS Compliance Validation</h1>
<p>Compliance validation completed successfully.</p>
</body>
</html>
EOF

log_success "Compliance validation complete. Report: $REPORTS_DIR/compliance-summary.html"