<<<<<<< HEAD
// Package porch provides client interfaces for interacting with Porch package management.
=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
package porch

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

<<<<<<< HEAD
// CreateDraftPackageRevision performs createdraftpackagerevision operation.

func CreateDraftPackageRevision(
	ctx context.Context,

	restcfg *rest.Config,

	namespace string,

	repository string,

	packageName string,

	workspace string,

	labels map[string]string,

	annotations map[string]string,
) (*unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

=======
func CreateDraftPackageRevision(
	ctx context.Context,
	restcfg *rest.Config,
	namespace string,
	repository string,
	packageName string,
	workspace string,
	labels map[string]string,
	annotations map[string]string,
) (*unstructured.Unstructured, error) {

	gvr := schema.GroupVersionResource{
		Group:    "porch.kpt.dev",
		Version:  "v1alpha1",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		Resource: "packagerevisions",
	}

	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
<<<<<<< HEAD
			"metadata": map[string]interface{}{
				"generateName": fmt.Sprintf("%s.%s.", repository, packageName),
				"namespace": namespace,
				"labels": labels,
				"annotations": annotations,
			},
			"spec": map[string]interface{}{},
=======
			"apiVersion": "porch.kpt.dev/v1alpha1",
			"kind":       "PackageRevision",
			"metadata": map[string]interface{}{
				"generateName": fmt.Sprintf("%s.%s.", repository, packageName),
				"namespace":    namespace,
				"labels":       labels,
				"annotations":  annotations,
			},
			"spec": map[string]interface{}{
				"lifecycle":     "Draft",
				"repository":    repository,
				"packageName":   packageName,
				"workspaceName": workspace,
				"revision":      0,
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
	}

	dc, err := dynamic.NewForConfig(restcfg)
	if err != nil {
		return nil, err
	}
<<<<<<< HEAD

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	return dc.Resource(gvr).Namespace(namespace).Create(ctx, u, metav1.CreateOptions{})
}
