//go:build tools

// +build tools



// Package tools provides a mechanism for tracking build tool dependencies.

// This ensures that all developers use the same versions of build tools.

// and that tools are included in go.sum for supply chain security.

//

// To install tools: go generate tools.go.

// To update tools: go get -u <tool>@latest && go mod tidy.


package tools



import (

	// SBOM generation tools.

	_ "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod"

	// Code generation and build tools.

	_ "github.com/golang/mock/mockgen"

	// Testing and quality tools.

	_ "github.com/onsi/ginkgo/v2/ginkgo"

	// Documentation and API tools.

	_ "github.com/swaggo/swag/cmd/swag"

	// Security and vulnerability tools.

	_ "golang.org/x/vuln/cmd/govulncheck"

	_ "k8s.io/code-generator/cmd/client-gen"

	_ "k8s.io/code-generator/cmd/deepcopy-gen"

	_ "k8s.io/code-generator/cmd/informer-gen"

	_ "k8s.io/code-generator/cmd/lister-gen"

	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"

)



// Tool versions - update these when upgrading tools.

const (

	// Core Kubernetes toolchain.

	ControllerToolsVersion = "v0.16.5"

	// CodeGeneratorVersion holds codegeneratorversion value.

	CodeGeneratorVersion = "v0.32.0"



	// Security toolchain.

	GovulncheckVersion = "v1.1.4"

	// GolangciLintVersion holds golangcilintversion value.

	GolangciLintVersion = "v1.63.4"



	// Testing toolchain.

	GinkgoVersion = "v2.22.0"

	// MockgenVersion holds mockgenversion value.

	MockgenVersion = "v1.7.0-rc.1"

	// MockeryVersion holds mockeryversion value.

	MockeryVersion = "v2.50.2"



	// SBOM and compliance.

	CycloneDXVersion = "v1.6.0"



	// Build and deployment.

	HelmVersion = "v3.16.4"

	// KoVersion holds koversion value.

	KoVersion = "v0.17.2"



	// Documentation.

	SwagVersion = "v1.16.4"

)



//go:generate go install k8s.io/code-generator/cmd/client-gen

//go:generate go install k8s.io/code-generator/cmd/deepcopy-gen

//go:generate go install k8s.io/code-generator/cmd/informer-gen

//go:generate go install k8s.io/code-generator/cmd/lister-gen

//go:generate go install sigs.k8s.io/controller-tools/cmd/controller-gen

//go:generate go install github.com/golang/mock/mockgen

//go:generate go install golang.org/x/vuln/cmd/govulncheck

//go:generate go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod

//go:generate go install github.com/onsi/ginkgo/v2/ginkgo

//go:generate go install github.com/golangci/golangci-lint/cmd/golangci-lint

//go:generate go install gotest.tools/gotestsum

//go:generate go install github.com/vektra/mockery/v2

//go:generate go install github.com/google/pprof

//go:generate go install github.com/google/ko

//go:generate go install helm.sh/helm/v3/cmd/helm

//go:generate go install github.com/swaggo/swag/cmd/swag

//go:generate go install k8s.io/kube-openapi/cmd/openapi-gen

//go:generate go install github.com/git-chglog/git-chglog/cmd/git-chglog



// Tool installation verification.

var toolVersions = map[string]string{

	"controller-gen":  ControllerToolsVersion,

	"deepcopy-gen":    CodeGeneratorVersion,

	"mockgen":         MockgenVersion,

	"govulncheck":     GovulncheckVersion,

	"golangci-lint":   GolangciLintVersion,

	"ginkgo":          GinkgoVersion,

	"mockery":         MockeryVersion,

	"cyclonedx-gomod": CycloneDXVersion,

	"helm":            HelmVersion,

	"ko":              KoVersion,

	"swag":            SwagVersion,

}



// GetToolVersion returns the expected version for a given tool.

func GetToolVersion(tool string) string {

	return toolVersions[tool]

}



// GetAllToolVersions returns all tool versions for verification.

func GetAllToolVersions() map[string]string {

	result := make(map[string]string)

	for k, v := range toolVersions {

		result[k] = v

	}

	return result

}

