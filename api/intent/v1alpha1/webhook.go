package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// 確保型別實作了新版介面（v0.21+）.
var (
	_ admission.CustomDefaulter = &NetworkIntentDefaulter{}
	_ admission.CustomValidator = &NetworkIntentValidator{}
)

// ---- Defaulter ----.

// NetworkIntentDefaulter represents a networkintentdefaulter.
type NetworkIntentDefaulter struct{}

// Default implements webhook.CustomDefaulter (mutating).
func (d *NetworkIntentDefaulter) Default(_ context.Context, obj runtime.Object) error {
	ni, ok := obj.(*NetworkIntent)
	if !ok {
		return fmt.Errorf("expected *NetworkIntent, got %T", obj)
	}

	// 給預設值.
	if ni.Spec.Source == "" {
		ni.Spec.Source = "user"
	}

	return nil
}

// ---- Validator ----.

// NetworkIntentValidator represents a networkintentvalidator.
type NetworkIntentValidator struct{}

// ValidateCreate implements webhook.CustomValidator (validating).
func (v *NetworkIntentValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return validateNetworkIntent(obj)
}

// ValidateUpdate implements webhook.CustomValidator (validating).
func (v *NetworkIntentValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return validateNetworkIntent(newObj)
}

// ValidateDelete implements webhook.CustomValidator (validating).
func (v *NetworkIntentValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func validateNetworkIntent(obj runtime.Object) (admission.Warnings, error) {
	ni, ok := obj.(*NetworkIntent)
	if !ok {
		return nil, fmt.Errorf("expected *NetworkIntent, got %T", obj)
	}

	var allErrs field.ErrorList

	// Validate intentType - must be "scaling" per contract.
	if ni.Spec.IntentType != "scaling" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "intentType"), ni.Spec.IntentType, "only 'scaling' supported"))
	}

	// Validate replicas - must be >= 0.
	if ni.Spec.Replicas < 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "replicas"), ni.Spec.Replicas, "must be >= 0"))
	}

	// Validate target - must be non-empty.
	if ni.Spec.Target == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "target"), ni.Spec.Target, "must be non-empty"))
	}

	// Validate namespace - must be non-empty.
	if ni.Spec.Namespace == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "namespace"), ni.Spec.Namespace, "must be non-empty"))
	}

	// Validate source if provided - must be user, planner, or test per contract.
	if ni.Spec.Source != "" && ni.Spec.Source != "user" && ni.Spec.Source != "planner" && ni.Spec.Source != "test" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "source"), ni.Spec.Source, "must be 'user', 'planner', or 'test'"))
	}

	if len(allErrs) > 0 {
		return nil, allErrs.ToAggregate()
	}

	return nil, nil
}

// ---- Builder glue ----.

// SetupWebhookWithManager sets up the webhook with the manager.
// It registers both the defaulter and validator for NetworkIntent resources.
func (r *NetworkIntent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(&NetworkIntentDefaulter{}).
		WithValidator(&NetworkIntentValidator{}).
		Complete()
}
