//go:build tools
// +build tools

// Package tools manages tool dependencies for security scanning and supply chain validation
package tools

import (
	// Security scanning tools
	_ "github.com/securego/gosec/v2/cmd/gosec"
	_ "golang.org/x/vuln/cmd/govulncheck"
	_ "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod"
	_ "github.com/anchore/syft/cmd/syft"
	_ "github.com/aquasecurity/trivy/cmd/trivy"
	
	// Supply chain security
	_ "github.com/sigstore/cosign/v2/cmd/cosign"
	_ "github.com/in-toto/in-toto-golang/in-toto"
	
	// License compliance
	_ "github.com/uw-labs/lichen"
	_ "github.com/google/go-licenses"
	
	// Code quality and complexity
	_ "github.com/fzipp/gocyclo/cmd/gocyclo"
	_ "honnef.co/go/tools/cmd/staticcheck"
	
	// Secret detection
	_ "github.com/trufflesecurity/trufflehog/v3/cmd/trufflehog"
	_ "github.com/zricethezav/gitleaks/v8/cmd/gitleaks"
)