#!/bin/bash
# Script: 03-send-intent.sh
# Purpose: Send scaling intent via HTTP POST or drop to handoff directory

set -e

METHOD="${METHOD:-handoff}"  # "http" or "handoff"
INTENT_URL="${INTENT_URL:-http://localhost:8080/intent}"
HANDOFF_DIR="${HANDOFF_DIR:-../../handoff}"
TARGET="${TARGET:-nf-sim}"
NAMESPACE="${NAMESPACE:-mvp-demo}"
REPLICAS="${REPLICAS:-3}"
SOURCE="${SOURCE:-test}"
REASON="${REASON:-MVP demo scaling test}"

echo "==== MVP Demo: Send Scaling Intent ===="
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Method: $METHOD"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Generate correlation ID
CORRELATION_ID="mvp-demo-$(date +%Y%m%d%H%M%S)"

# Create intent JSON according to schema
INTENT_JSON=$(cat <<EOF
{
  "intent_type": "scaling",
  "target": "$TARGET",
  "namespace": "$NAMESPACE",
  "replicas": $REPLICAS,
  "reason": "$REASON",
  "source": "$SOURCE",
  "correlation_id": "$CORRELATION_ID"
}
EOF
)

echo ""
echo "Intent JSON:"
echo "$INTENT_JSON"

# Validate JSON structure if jq is available
if command -v jq >/dev/null 2>&1; then
    echo ""
    echo "Validating intent structure..."
    if echo "$INTENT_JSON" | jq -e '.intent_type == "scaling" and .target and .namespace and .replicas' >/dev/null; then
        echo "Intent structure is valid"
    else
        echo "Warning: Intent structure validation failed, but continuing..."
    fi
fi

# Send intent based on method
if [ "$METHOD" = "http" ]; then
    # Send via HTTP POST
    echo ""
    echo "Sending intent via HTTP POST to: $INTENT_URL"
    
    if command -v curl >/dev/null 2>&1; then
        HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
            -H "Content-Type: application/json" \
            -H "X-Correlation-Id: $CORRELATION_ID" \
            -d "$INTENT_JSON" \
            "$INTENT_URL" 2>/dev/null || true)
        
        HTTP_CODE=$(echo "$HTTP_RESPONSE" | tail -n1)
        RESPONSE_BODY=$(echo "$HTTP_RESPONSE" | head -n-1)
        
        if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "202" ]; then
            echo "Intent sent successfully! (HTTP $HTTP_CODE)"
            if [ -n "$RESPONSE_BODY" ]; then
                echo "Response:"
                echo "$RESPONSE_BODY"
            fi
        else
            echo "Warning: HTTP request failed with status code: $HTTP_CODE"
            if [ -n "$RESPONSE_BODY" ]; then
                echo "Error details: $RESPONSE_BODY"
            fi
            echo ""
            echo "Falling back to handoff directory method..."
            METHOD="handoff"
        fi
    elif command -v wget >/dev/null 2>&1; then
        if wget -q -O- --method=POST \
            --header="Content-Type: application/json" \
            --header="X-Correlation-Id: $CORRELATION_ID" \
            --body-data="$INTENT_JSON" \
            "$INTENT_URL" 2>/dev/null; then
            echo "Intent sent successfully!"
        else
            echo "Warning: HTTP request failed"
            echo "Falling back to handoff directory method..."
            METHOD="handoff"
        fi
    else
        echo "Warning: Neither curl nor wget found. Using handoff method instead."
        METHOD="handoff"
    fi
fi

if [ "$METHOD" = "handoff" ]; then
    # Drop to handoff directory
    HANDOFF_PATH="$SCRIPT_DIR/$HANDOFF_DIR"
    
    # Resolve to absolute path
    HANDOFF_PATH=$(cd "$HANDOFF_PATH" 2>/dev/null && pwd) || {
        # Create handoff directory if it doesn't exist
        mkdir -p "$HANDOFF_PATH"
        HANDOFF_PATH=$(cd "$HANDOFF_PATH" && pwd)
    }
    
    echo ""
    echo "Dropping intent to handoff directory: $HANDOFF_PATH"
    
    # Generate filename with timestamp
    TIMESTAMP=$(date +%Y%m%dT%H%M%SZ)
    FILE_NAME="intent-$TIMESTAMP.json"
    FILE_PATH="$HANDOFF_PATH/$FILE_NAME"
    
    # Write intent to file
    echo "$INTENT_JSON" > "$FILE_PATH"
    
    if [ -f "$FILE_PATH" ]; then
        echo "Intent written successfully!"
        echo "File: $FILE_NAME"
        echo "Full path: $FILE_PATH"
        
        # Verify file contents
        echo ""
        echo "Verifying file contents:"
        cat "$FILE_PATH"
    else
        echo "Error: Failed to write intent file"
        exit 1
    fi
fi

# Check if conductor is running (optional)
echo ""
echo "Checking for conductor process..."
if pgrep -f "conductor" >/dev/null 2>&1; then
    CONDUCTOR_PID=$(pgrep -f "conductor" | head -n1)
    echo "Conductor process is running (PID: $CONDUCTOR_PID)"
else
    echo "Conductor process not found. Make sure it's running to process the intent."
    echo "Start conductor with: go run ./cmd/conductor"
fi

# Summary
echo ""
echo "==== Intent Summary ===="
echo "✓ Target: $TARGET"
echo "✓ Namespace: $NAMESPACE"
echo "✓ Desired replicas: $REPLICAS"
echo "✓ Correlation ID: $CORRELATION_ID"
echo "✓ Method: $METHOD"

echo ""
echo "Intent sent! Next step: Run ./04-porch-apply.sh"
echo "Monitor the conductor logs for processing status"