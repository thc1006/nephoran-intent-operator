module github.com/nephio-project/nephoran-intent-operator

go 1.24.6

// Core Kubernetes dependencies
require (
	k8s.io/api v0.29.0
	k8s.io/apiextensions-apiserver v0.29.0
	k8s.io/apimachinery v0.29.0
	k8s.io/client-go v0.29.0
	sigs.k8s.io/controller-runtime v0.17.0
	sigs.k8s.io/kustomize/api v0.17.1
	sigs.k8s.io/kustomize/kyaml v0.17.0
)

// Monitoring and Observability
require (
	github.com/prometheus/client_golang v1.18.0
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.19.0
	go.uber.org/zap v1.26.0
)

// Utility Libraries
require (
	github.com/go-logr/logr v1.4.1
	golang.org/x/sync v0.5.0
	golang.org/x/time v0.3.0
)

// Compatibility and Compliance
require (
	github.com/golang/protobuf v1.5.3
	google.golang.org/protobuf v1.33.0
	gopkg.in/yaml.v3 v3.0.1
)

// Replace local modules to use local filesystem
replace (
	github.com/nephio-project/nephoran-intent-operator/api => ./api
	github.com/nephio-project/nephoran-intent-operator/controllers => ./controllers
	github.com/nephio-project/nephoran-intent-operator/pkg => ./pkg
	github.com/nephio-project/nephoran-intent-operator/pkg/audit => ./pkg/audit
	github.com/nephio-project/nephoran-intent-operator/pkg/cnf => ./pkg/cnf
	github.com/nephio-project/nephoran-intent-operator/pkg/monitoring/availability => ./pkg/monitoring/availability
	github.com/nephio-project/nephoran-intent-operator/pkg/oran/o2 => ./pkg/oran/o2
	github.com/nephio-project/nephoran-intent-operator/pkg/security/ca => ./pkg/security/ca
	github.com/nephio-project/nephoran-intent-operator/pkg/security/mtls => ./pkg/security/mtls
)

// Restrict and pin dependency versions
replace (
	k8s.io/api => k8s.io/api v0.29.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.29.0
	k8s.io/client-go => k8s.io/client-go v0.29.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.17.0
	sigs.k8s.io/kustomize/api => sigs.k8s.io/kustomize/api v0.17.1
	sigs.k8s.io/kustomize/kyaml => sigs.k8s.io/kustomize/kyaml v0.16.0
)
