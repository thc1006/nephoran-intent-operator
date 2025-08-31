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

package fixtures

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// NetworkIntentBuilder provides a fluent interface for building NetworkIntent test objects
type NetworkIntentBuilder struct {
	intent *nephoranv1.NetworkIntent
}

// NewNetworkIntentBuilder creates a new NetworkIntentBuilder
func NewNetworkIntentBuilder() *NetworkIntentBuilder {
	return &NetworkIntentBuilder{
		intent: &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-intent",
				Namespace: "default",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "default test intent",
			},
		},
	}
}

// WithName sets the name
func (b *NetworkIntentBuilder) WithName(name string) *NetworkIntentBuilder {
	b.intent.ObjectMeta.Name = name
	return b
}

// WithNamespace sets the namespace
func (b *NetworkIntentBuilder) WithNamespace(namespace string) *NetworkIntentBuilder {
	b.intent.ObjectMeta.Namespace = namespace
	return b
}

// WithIntent sets the intent string
func (b *NetworkIntentBuilder) WithIntent(intent string) *NetworkIntentBuilder {
	b.intent.Spec.Intent = intent
	return b
}

// WithUID sets the UID
func (b *NetworkIntentBuilder) WithUID(uid string) *NetworkIntentBuilder {
	b.intent.ObjectMeta.UID = types.UID(uid)
	return b
}

// WithResourceVersion sets the resource version
func (b *NetworkIntentBuilder) WithResourceVersion(version string) *NetworkIntentBuilder {
	b.intent.ObjectMeta.ResourceVersion = version
	return b
}

// WithGeneration sets the generation
func (b *NetworkIntentBuilder) WithGeneration(generation int64) *NetworkIntentBuilder {
	b.intent.ObjectMeta.Generation = generation
	return b
}

// WithLabels sets labels
func (b *NetworkIntentBuilder) WithLabels(labels map[string]string) *NetworkIntentBuilder {
	b.intent.ObjectMeta.Labels = labels
	return b
}

// WithAnnotations sets annotations
func (b *NetworkIntentBuilder) WithAnnotations(annotations map[string]string) *NetworkIntentBuilder {
	b.intent.ObjectMeta.Annotations = annotations
	return b
}

// WithFinalizers sets finalizers
func (b *NetworkIntentBuilder) WithFinalizers(finalizers []string) *NetworkIntentBuilder {
	b.intent.ObjectMeta.Finalizers = finalizers
	return b
}

// WithDeletionTimestamp sets the deletion timestamp
func (b *NetworkIntentBuilder) WithDeletionTimestamp(timestamp *metav1.Time) *NetworkIntentBuilder {
	b.intent.ObjectMeta.DeletionTimestamp = timestamp
	return b
}

// WithStatus sets the status
func (b *NetworkIntentBuilder) WithStatus(status nephoranv1.NetworkIntentStatus) *NetworkIntentBuilder {
	b.intent.Status = status
	return b
}

// Build returns the constructed NetworkIntent
func (b *NetworkIntentBuilder) Build() *nephoranv1.NetworkIntent {
	return b.intent.DeepCopy()
}

// Predefined fixture functions for common test scenarios

// SimpleNetworkIntent returns a basic NetworkIntent for testing
func SimpleNetworkIntent() *nephoranv1.NetworkIntent {
	return NewNetworkIntentBuilder().
		WithName("simple-intent").
		WithIntent("scale up network function").
		Build()
}

// ScaleUpIntent returns a NetworkIntent with scale-up intent
func ScaleUpIntent() *nephoranv1.NetworkIntent {
	return NewNetworkIntentBuilder().
		WithName("scale-up-intent").
		WithIntent("scale up network function to 5 replicas").
		Build()
}

// ScaleDownIntent returns a NetworkIntent with scale-down intent
func ScaleDownIntent() *nephoranv1.NetworkIntent {
	return NewNetworkIntentBuilder().
		WithName("scale-down-intent").
		WithIntent("scale down network function to 1 replica").
		Build()
}

// ComplexIntent returns a NetworkIntent with complex intent
func ComplexIntent() *nephoranv1.NetworkIntent {
	return NewNetworkIntentBuilder().
		WithName("complex-intent").
		WithIntent("scale network function based on CPU utilization above 80% to maximum 10 replicas with minimum 2 replicas").
		Build()
}

// EmptyIntent returns a NetworkIntent with empty intent
func EmptyIntent() *nephoranv1.NetworkIntent {
	return NewNetworkIntentBuilder().
		WithName("empty-intent").
		WithIntent("").
		Build()
}

// LongIntent returns a NetworkIntent with very long intent string
func LongIntent() *nephoranv1.NetworkIntent {
	longIntentText := "This is a very long intent string that contains many words and describes a complex scaling scenario. " +
		"The intent should handle network function scaling based on multiple metrics including CPU utilization, " +
		"memory usage, network throughput, and user-defined custom metrics. The scaling should be performed " +
		"gradually to avoid service disruption and should consider dependencies between different network functions. " +
		"Additionally, the scaling should respect resource quotas and cluster capacity constraints while " +
		"maintaining high availability and optimal performance characteristics."

	return NewNetworkIntentBuilder().
		WithName("long-intent").
		WithIntent(longIntentText).
		Build()
}

// IntentWithFinalizers returns a NetworkIntent with finalizers
func IntentWithFinalizers() *nephoranv1.NetworkIntent {
	return NewNetworkIntentBuilder().
		WithName("intent-with-finalizers").
		WithIntent("scale network function").
		WithFinalizers([]string{"nephoran.io/finalizer"}).
		Build()
}

// IntentWithLabelsAndAnnotations returns a NetworkIntent with labels and annotations
func IntentWithLabelsAndAnnotations() *nephoranv1.NetworkIntent {
	labels := map[string]string{
		"app":     "network-function",
		"tier":    "control-plane",
		"version": "v1.0.0",
	}
	annotations := map[string]string{
		"nephoran.io/processed":   "true",
		"nephoran.io/llm-enabled": "true",
		"kubectl.kubernetes.io/last-applied-configuration": `{"apiVersion":"nephoran.io/v1","kind":"NetworkIntent"}`,
	}

	return NewNetworkIntentBuilder().
		WithName("intent-with-metadata").
		WithIntent("scale network function with metadata").
		WithLabels(labels).
		WithAnnotations(annotations).
		Build()
}

// IntentBeingDeleted returns a NetworkIntent that is being deleted
func IntentBeingDeleted() *nephoranv1.NetworkIntent {
	now := metav1.Now()
	return NewNetworkIntentBuilder().
		WithName("intent-being-deleted").
		WithIntent("scale network function").
		WithDeletionTimestamp(&now).
		WithFinalizers([]string{"nephoran.io/finalizer"}).
		Build()
}

// IntentInDifferentNamespace returns a NetworkIntent in a non-default namespace
func IntentInDifferentNamespace(namespace string) *nephoranv1.NetworkIntent {
	return NewNetworkIntentBuilder().
		WithName("intent-different-namespace").
		WithNamespace(namespace).
		WithIntent("scale network function in different namespace").
		Build()
}

// IntentWithStatus returns a NetworkIntent with status set
func IntentWithStatus() *nephoranv1.NetworkIntent {
	status := nephoranv1.NetworkIntentStatus{
		Phase:       "Processing",
		LastMessage: "Intent is being processed",
		Conditions: []metav1.Condition{
			{
				Type:    "Ready",
				Status:  metav1.ConditionTrue,
				Reason:  "IntentProcessed",
				Message: "NetworkIntent has been successfully processed",
			},
		},
	}

	return NewNetworkIntentBuilder().
		WithName("intent-with-status").
		WithIntent("scale network function").
		WithStatus(status).
		Build()
}

// MultipleIntents returns a slice of multiple NetworkIntent objects for bulk testing
func MultipleIntents(count int) []*nephoranv1.NetworkIntent {
	intents := make([]*nephoranv1.NetworkIntent, count)
	for i := 0; i < count; i++ {
		intents[i] = NewNetworkIntentBuilder().
			WithName(fmt.Sprintf("intent-%d", i)).
			WithIntent(fmt.Sprintf("scale network function %d", i)).
			Build()
	}
	return intents
}

// IntentsForLoadTesting returns a large number of NetworkIntent objects for load testing
func IntentsForLoadTesting() []*nephoranv1.NetworkIntent {
	return MultipleIntents(100)
}

// IntentsForConcurrencyTesting returns NetworkIntent objects designed for concurrency testing
func IntentsForConcurrencyTesting() []*nephoranv1.NetworkIntent {
	intents := make([]*nephoranv1.NetworkIntent, 10)
	for i := 0; i < 10; i++ {
		intents[i] = NewNetworkIntentBuilder().
			WithName(fmt.Sprintf("concurrent-intent-%d", i)).
			WithNamespace("concurrent-test").
			WithIntent(fmt.Sprintf("concurrent scaling intent %d", i)).
			WithLabels(map[string]string{
				"test-type": "concurrency",
				"batch-id":  "concurrent-test-1",
			}).
			Build()
	}
	return intents
}

// InvalidIntents returns NetworkIntent objects with invalid data for negative testing
func InvalidIntents() []*nephoranv1.NetworkIntent {
	return []*nephoranv1.NetworkIntent{
		// Intent with invalid characters in name (should be handled by Kubernetes validation)
		NewNetworkIntentBuilder().
			WithName("invalid-name-with-UPPERCASE").
			WithIntent("scale network function").
			Build(),
		
		// Intent with extremely long name
		NewNetworkIntentBuilder().
			WithName("this-is-an-extremely-long-name-that-exceeds-kubernetes-limits-and-should-cause-validation-errors").
			WithIntent("scale network function").
			Build(),
	}
}