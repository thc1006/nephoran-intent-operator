// Package v1alpha1 contains API Schema definitions for the intent v1alpha1 API group
// +kubebuilder:object:generate=true
<<<<<<< HEAD
// +groupName=intent.nephio.org
=======
// +groupName=intent.nephoran.com
>>>>>>> 6835433495e87288b95961af7173d866977175ff
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
<<<<<<< HEAD
	GroupVersion = schema.GroupVersion{Group: "intent.nephio.org", Version: "v1alpha1"}
=======
	GroupVersion = schema.GroupVersion{Group: "intent.nephoran.com", Version: "v1alpha1"}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
