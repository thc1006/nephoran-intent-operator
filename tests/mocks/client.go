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

package test_mocks

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockClient implements a mock Kubernetes client for testing
type MockClient struct {
	objects    map[client.ObjectKey]client.Object
	mutex      sync.RWMutex
	scheme     *runtime.Scheme
	getError   error
	listError  error
	createError error
	updateError error
	deleteError error
	patchError  error
}

// NewMockClient creates a new mock client
func NewMockClient(scheme *runtime.Scheme) *MockClient {
	return &MockClient{
		objects: make(map[client.ObjectKey]client.Object),
		scheme:  scheme,
	}
}

// SetGetError sets an error to return on Get operations
func (m *MockClient) SetGetError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.getError = err
}

// SetListError sets an error to return on List operations
func (m *MockClient) SetListError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.listError = err
}

// SetCreateError sets an error to return on Create operations
func (m *MockClient) SetCreateError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.createError = err
}

// SetUpdateError sets an error to return on Update operations
func (m *MockClient) SetUpdateError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.updateError = err
}

// SetDeleteError sets an error to return on Delete operations
func (m *MockClient) SetDeleteError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.deleteError = err
}

// SetPatchError sets an error to return on Patch operations
func (m *MockClient) SetPatchError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.patchError = err
}

// Get retrieves an object by key
func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.getError != nil {
		return m.getError
	}

	stored, exists := m.objects[key]
	if !exists {
		return errors.NewNotFound(schema.GroupResource{
			Group:    obj.GetObjectKind().GroupVersionKind().Group,
			Resource: obj.GetObjectKind().GroupVersionKind().Kind,
		}, key.Name)
	}

	// Use runtime.Object's DeepCopyObject method if available
	if copyable, ok := stored.(interface{ DeepCopyObject() runtime.Object }); ok {
		copied := copyable.DeepCopyObject()
		if copyableResult, ok := copied.(client.Object); ok {
			// Copy metadata
			if metaObj, ok := obj.(metav1.Object); ok {
				if storedMetaObj, ok := copyableResult.(metav1.Object); ok {
					metaObj.SetName(storedMetaObj.GetName())
					metaObj.SetNamespace(storedMetaObj.GetNamespace())
					metaObj.SetLabels(storedMetaObj.GetLabels())
					metaObj.SetAnnotations(storedMetaObj.GetAnnotations())
					metaObj.SetUID(storedMetaObj.GetUID())
					metaObj.SetResourceVersion(storedMetaObj.GetResourceVersion())
					metaObj.SetCreationTimestamp(storedMetaObj.GetCreationTimestamp())
				}
			}
			obj.GetObjectKind().SetGroupVersionKind(copyableResult.GetObjectKind().GroupVersionKind())
		}
	}
	return nil
}

// List retrieves a list of objects
func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.listError != nil {
		return m.listError
	}

	// For simplicity, return all objects
	// In a real mock, you'd filter based on options
	items := make([]client.Object, 0, len(m.objects))
	for _, obj := range m.objects {
		items = append(items, obj.DeepCopyObject().(client.Object))
	}

	// This is a simplified implementation
	// Real implementation would need to handle different list types
	return nil
}

// Create creates a new object
func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.createError != nil {
		return m.createError
	}

	key := client.ObjectKeyFromObject(obj)
	if _, exists := m.objects[key]; exists {
		return errors.NewAlreadyExists(schema.GroupResource{
			Group:    obj.GetObjectKind().GroupVersionKind().Group,
			Resource: obj.GetObjectKind().GroupVersionKind().Kind,
		}, key.Name)
	}

	// Set creation timestamp and UID
	now := metav1.Now()
	obj.SetCreationTimestamp(now)
	obj.SetUID(types.UID(fmt.Sprintf("uid-%s-%s", key.Namespace, key.Name)))

	m.objects[key] = obj.DeepCopyObject().(client.Object)
	return nil
}

// Update updates an existing object
func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.updateError != nil {
		return m.updateError
	}

	key := client.ObjectKeyFromObject(obj)
	if _, exists := m.objects[key]; !exists {
		return errors.NewNotFound(schema.GroupResource{
			Group:    obj.GetObjectKind().GroupVersionKind().Group,
			Resource: obj.GetObjectKind().GroupVersionKind().Kind,
		}, key.Name)
	}

	m.objects[key] = obj.DeepCopyObject().(client.Object)
	return nil
}

// Patch patches an existing object
func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.patchError != nil {
		return m.patchError
	}

	key := client.ObjectKeyFromObject(obj)
	if _, exists := m.objects[key]; !exists {
		return errors.NewNotFound(schema.GroupResource{
			Group:    obj.GetObjectKind().GroupVersionKind().Group,
			Resource: obj.GetObjectKind().GroupVersionKind().Kind,
		}, key.Name)
	}

	m.objects[key] = obj.DeepCopyObject().(client.Object)
	return nil
}

// Delete deletes an object
func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.deleteError != nil {
		return m.deleteError
	}

	key := client.ObjectKeyFromObject(obj)
	if _, exists := m.objects[key]; !exists {
		return errors.NewNotFound(schema.GroupResource{
			Group:    obj.GetObjectKind().GroupVersionKind().Group,
			Resource: obj.GetObjectKind().GroupVersionKind().Kind,
		}, key.Name)
	}

	delete(m.objects, key)
	return nil
}

// DeleteAllOf deletes all objects matching the given options
func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.deleteError != nil {
		return m.deleteError
	}

	// For simplicity, delete all objects of this type
	// In a real mock, you'd filter based on options
	toDelete := make([]client.ObjectKey, 0)
	for key, stored := range m.objects {
		if stored.GetObjectKind().GroupVersionKind() == obj.GetObjectKind().GroupVersionKind() {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		delete(m.objects, key)
	}

	return nil
}

// Scheme returns the scheme
func (m *MockClient) Scheme() *runtime.Scheme {
	return m.scheme
}

// RESTMapper returns a RESTMapper (not implemented in mock)
func (m *MockClient) RESTMapper() meta.RESTMapper {
	return nil
}

// Status returns a status writer
func (m *MockClient) Status() client.StatusWriter {
	return &mockStatusWriter{client: m}
}

// SubResource returns a subresource client
func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	return nil
}

// mockStatusWriter implements client.StatusWriter for testing
type mockStatusWriter struct {
	client *MockClient
}

func (m *mockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return m.client.Update(ctx, obj)
}

func (m *mockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return m.client.Patch(ctx, obj, patch)
}

func (m *mockStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return m.client.Create(ctx, obj)
}

// AddObject adds an object to the mock client's storage
func (m *MockClient) AddObject(obj client.Object) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	key := client.ObjectKeyFromObject(obj)
	m.objects[key] = obj.DeepCopyObject().(client.Object)
}

// GetObjectCount returns the number of objects stored
func (m *MockClient) GetObjectCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.objects)
}

// Clear removes all objects from storage
func (m *MockClient) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.objects = make(map[client.ObjectKey]client.Object)
}