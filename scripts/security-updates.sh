#!/bin/bash
# Security Updates Script for Nephoran (2025 Standards)
# Updates all dependencies to latest secure versions

set -euo pipefail

echo "=================================================="
echo "NEPHORAN SECURITY UPDATES - 2025 Standards"
echo "=================================================="

# Update Go dependencies to latest secure versions
echo "[1/5] Updating Go dependencies..."
go get -u github.com/golang-jwt/jwt/v5@v5.3.0
go get -u golang.org/x/crypto@latest
go get -u golang.org/x/net@latest
go get -u golang.org/x/text@latest
go get -u golang.org/x/sys@latest

# Security-critical updates
echo "[2/5] Applying security-critical updates..."
go get -u github.com/google/uuid@v1.6.0
go get -u github.com/gorilla/mux@v1.8.1
go get -u github.com/prometheus/client_golang@v1.22.0
go get -u k8s.io/apimachinery@v0.32.1
go get -u k8s.io/client-go@v0.32.1
go get -u sigs.k8s.io/controller-runtime@v0.20.2

# Remove vulnerable dependencies
echo "[3/5] Removing known vulnerable dependencies..."
go mod edit -droprequire github.com/dgrijalva/jwt-go || true
go mod edit -droprequire github.com/gorilla/websocket@v1.4.2 || true

# Update and verify
echo "[4/5] Tidying and verifying modules..."
go mod tidy
go mod verify

# Run security audit
echo "[5/5] Running security audit..."
go list -json -m all | nancy sleuth || true
govulncheck ./... || true

echo "=================================================="
echo "Security updates completed!"
echo "Please review changes and run tests before committing."
echo "==================================================">