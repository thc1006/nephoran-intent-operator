module github.com/thc1006/nephoran-intent-operator/api

go 1.24.6

require (
	k8s.io/apimachinery v0.34.0
	sigs.k8s.io/controller-runtime v0.22.0
)

// Consistent Kubernetes dependency versions
replace (
	k8s.io/api => k8s.io/api v0.34.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.34.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.34.0
	k8s.io/client-go => k8s.io/client-go v0.34.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.22.0
)
