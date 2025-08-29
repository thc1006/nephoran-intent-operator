//go:build tools

// +build tools



// Package tools manages tool dependencies for security scanning and supply chain validation.


package tools



import (

	// Security scanning tools.

	_ "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod"

	_ "golang.org/x/vuln/cmd/govulncheck"

	// Code quality and complexity.

	_ "honnef.co/go/tools/cmd/staticcheck"

)

