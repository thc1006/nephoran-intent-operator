module github.com/nephio-project/nephoran-intent-operator

go 1.24.6

// Core Kubernetes dependencies
require (
	k8s.io/api v0.29.0
	k8s.io/apimachinery v0.29.0
	k8s.io/client-go v0.29.0
	sigs.k8s.io/controller-runtime v0.17.0
)

// Performance and Monitoring
require (
	github.com/prometheus/client_golang v1.18.0
	go.opentelemetry.io/otel v1.19.0
	go.uber.org/zap v1.26.0
)

// Utility libraries
require (
	github.com/go-logr/logr v1.4.1
	golang.org/x/sync v0.5.0
	golang.org/x/time v0.3.0
)

// Add compatibility requirements
require (
	github.com/golang/protobuf v1.5.3
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v3 v3.0.1
)

// Restrict dependency versions for consistency
replace (
	k8s.io/api => k8s.io/api v0.29.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.29.0
	k8s.io/client-go => k8s.io/client-go v0.29.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.17.0
)