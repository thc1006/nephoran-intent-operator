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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var networkintentlog = logf.Log.WithName("networkintent-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *NetworkIntent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-intent-nephio-org-v1alpha1-networkintent,mutating=true,failurePolicy=fail,sideEffects=None,groups=intent.nephio.org,resources=networkintents,verbs=create;update,versions=v1alpha1,name=mnetworkintent.kb.io,admissionReviewVersions=v1

var _ admission.CustomDefaulter = &NetworkIntent{}

// Default implements admission.CustomDefaulter so a webhook will be registered for the type
func (r *NetworkIntent) Default(ctx context.Context, obj runtime.Object) error {
	ni, ok := obj.(*NetworkIntent)
	if !ok {
		return fmt.Errorf("expected *NetworkIntent, got %T", obj)
	}

	networkintentlog.Info("default", "name", ni.Name)

	// Set default source if not provided
	if ni.Spec.Source == "" {
		ni.Spec.Source = "user"
	}

	// Set default intentType if not provided
	if ni.Spec.IntentType == "" {
		ni.Spec.IntentType = "scaling"
	}

	return nil
}

//+kubebuilder:webhook:path=/validate-intent-nephio-org-v1alpha1-networkintent,mutating=false,failurePolicy=fail,sideEffects=None,groups=intent.nephio.org,resources=networkintents,verbs=create;update,versions=v1alpha1,name=vnetworkintent.kb.io,admissionReviewVersions=v1

var _ admission.CustomValidator = &NetworkIntent{}

// ValidateCreate implements admission.CustomValidator so a webhook will be registered for the type
func (r *NetworkIntent) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	ni, ok := obj.(*NetworkIntent)
	if !ok {
		return nil, fmt.Errorf("expected *NetworkIntent, got %T", obj)
	}

	networkintentlog.Info("validate create", "name", ni.Name)
	return ni.validateNetworkIntent()
}

// ValidateUpdate implements admission.CustomValidator so a webhook will be registered for the type
func (r *NetworkIntent) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	ni, ok := newObj.(*NetworkIntent)
	if !ok {
		return nil, fmt.Errorf("expected *NetworkIntent, got %T", newObj)
	}

	networkintentlog.Info("validate update", "name", ni.Name)
	return ni.validateNetworkIntent()
}

// ValidateDelete implements admission.CustomValidator so a webhook will be registered for the type
func (r *NetworkIntent) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	ni, ok := obj.(*NetworkIntent)
	if !ok {
		return nil, fmt.Errorf("expected *NetworkIntent, got %T", obj)
	}

	networkintentlog.Info("validate delete", "name", ni.Name)

	// No validation needed for deletion
	return nil, nil
}

// validateNetworkIntent performs validation for NetworkIntent
func (r *NetworkIntent) validateNetworkIntent() (admission.Warnings, error) {
	var allErrs []error
	var warnings admission.Warnings

	// Validate intentType
	if r.Spec.IntentType != "scaling" {
		allErrs = append(allErrs, fmt.Errorf("spec.intentType must be 'scaling', got: %s", r.Spec.IntentType))
	}

	// Validate target is not empty
	if r.Spec.Target == "" {
		allErrs = append(allErrs, fmt.Errorf("spec.target cannot be empty"))
	}

	// Validate namespace is not empty
	if r.Spec.Namespace == "" {
		allErrs = append(allErrs, fmt.Errorf("spec.namespace cannot be empty"))
	}

	// Validate replicas is non-negative
	if r.Spec.Replicas < 0 {
		allErrs = append(allErrs, fmt.Errorf("spec.replicas must be non-negative, got: %d", r.Spec.Replicas))
	}

	// Validate source
	validSources := []string{"user", "planner", "test"}
	validSource := false
	for _, validSrc := range validSources {
		if r.Spec.Source == validSrc {
			validSource = true
			break
		}
	}
	if !validSource {
		allErrs = append(allErrs, fmt.Errorf("spec.source must be 'user', 'planner', or 'test', got: %s", r.Spec.Source))
	}


	// Add warning if replicas is very high
	if r.Spec.Replicas > 100 {
		warnings = append(warnings, "spec.replicas is set to a very high value, consider reviewing resource requirements")
	}

	if len(allErrs) > 0 {
		return nil, fmt.Errorf("intent validation failed: %v", allErrs)
	}

	return warnings, nil
}
