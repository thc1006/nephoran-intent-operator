//go:build tools
// +build tools

// Package tools manages tool dependencies for security scanning and supply chain validation.

package tools

import (
	_ "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod"
	_ "golang.org/x/vuln/cmd/govulncheck"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
