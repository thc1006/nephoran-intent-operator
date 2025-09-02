module github.com/thc1006/nephoran-intent-operator

go 1.24.6

// Core Kubernetes Dependencies
require (
	k8s.io/api v0.28.9
	k8s.io/apimachinery v0.28.9
	k8s.io/client-go v0.28.9
	k8s.io/metrics v0.28.9
	sigs.k8s.io/controller-runtime v0.16.3
)

// Test Framework Dependencies
require (
	github.com/onsi/ginkgo/v2 v2.25.1
	github.com/onsi/gomega v1.38.1
)

// Project Core Dependencies
require (

	// Additional Core Packages
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/aws/aws-sdk-go-v2 v1.38.3
	github.com/go-logr/logr v1.4.3
	github.com/prometheus/client_golang v1.23.0
	github.com/spiffe/go-spiffe/v2 v2.5.0
	go.opentelemetry.io/otel/trace v1.38.0
	go.uber.org/zap v1.27.0
)

// OpenTelemetry and Monitoring
require (
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.32.0
	go.opentelemetry.io/otel/exporters/prometheus v0.54.0
)

// Force Test Framework Compatibility
replace (
	github.com/onsi/ginkgo/v2 => github.com/onsi/ginkgo/v2 v2.22.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.38.1
)

// Remove version constraints to let Go resolve naturally
replace go.yaml.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.1
