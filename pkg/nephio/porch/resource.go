/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package porch

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource returns a resource for the given resource name
func Resource(resource string) schema.GroupResource {
	return schema.GroupResource{
		Group:    "porch.nephoran.com",
		Resource: resource,
	}
}
