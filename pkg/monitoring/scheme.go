// Package monitoring - Kubernetes scheme and runtime registration
package monitoring

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// GroupVersion is group version used to register these objects
var GroupVersion = schema.GroupVersion{Group: "monitoring.nephoran.io", Version: "v1alpha1"}

// SchemeBuilder is used to add go types to the GroupVersionKind scheme
var SchemeBuilder = &runtime.SchemeBuilder{}

// Scheme is the runtime scheme for the monitoring package
var Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(AddToScheme(Scheme))
}

// AddToScheme adds the types in this group-version to the given scheme.
func AddToScheme(s *runtime.Scheme) error {
	return SchemeBuilder.AddToScheme(s)
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return GroupVersion.WithResource(resource).GroupResource()
}

// Kind takes an unqualified kind and returns back a Group qualified GroupVersionKind
func Kind(kind string) schema.GroupVersionKind {
	return GroupVersion.WithKind(kind)
}
