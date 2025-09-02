// Package porch provides client interfaces for interacting with Porch package management.
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

		Resource: "packagerevisions",
	}

	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "porch.kpt.dev/v1alpha1",

			"kind": "PackageRevision",

			"metadata": map[string]interface{}{
				"generateName": fmt.Sprintf("%s.%s.", repository, packageName),

				"namespace": namespace,

				"labels": labels,

				"annotations": annotations,
			},

			"spec": map[string]interface{}{
				"lifecycle": "Draft",

				"repository": repository,

				"packageName": packageName,

				"workspaceName": workspace,

				"revision": 0,
			},
		},
	}

	dc, err := dynamic.NewForConfig(restcfg)
	if err != nil {
		return nil, err
	}

	return dc.Resource(gvr).Namespace(namespace).Create(ctx, u, metav1.CreateOptions{})
}
