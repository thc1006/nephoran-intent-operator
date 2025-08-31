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

package v1

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var networkintentlog = logf.Log.WithName("networkintent-v1-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *NetworkIntent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-nephoran-io-v1-networkintent,mutating=true,failurePolicy=fail,sideEffects=None,groups=nephoran.io,resources=networkintents,verbs=create;update,versions=v1,name=mnetworkintent-v1.kb.io,admissionReviewVersions=v1

var _ admission.CustomDefaulter = &NetworkIntent{}

// Default implements admission.CustomDefaulter so a webhook will be registered for the type
func (r *NetworkIntent) Default(ctx context.Context, obj runtime.Object) error {
	ni, ok := obj.(*NetworkIntent)
	if !ok {
		return fmt.Errorf("expected *NetworkIntent, got %T", obj)
	}

	networkintentlog.Info("default", "name", ni.Name)

	// Set default intent type if not provided
	if ni.Spec.IntentType == "" {
		ni.Spec.IntentType = IntentTypeDeployment
	}

	// Set default priority if not provided
	if ni.Spec.Priority == "" {
		ni.Spec.Priority = PriorityMedium
	}

	return nil
}

//+kubebuilder:webhook:path=/validate-nephoran-io-v1-networkintent,mutating=false,failurePolicy=fail,sideEffects=None,groups=nephoran.io,resources=networkintents,verbs=create;update,versions=v1,name=vnetworkintent-v1.kb.io,admissionReviewVersions=v1

var _ admission.CustomValidator = &NetworkIntent{}

// ValidateCreate implements admission.CustomValidator so a webhook will be registered for the type
func (r *NetworkIntent) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	ni, ok := obj.(*NetworkIntent)
	if !ok {
		return nil, fmt.Errorf("expected *NetworkIntent, got %T", obj)
	}

	networkintentlog.Info("validate create", "name", ni.Name)
	return r.validateNetworkIntent()
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
	var allErrs field.ErrorList
	var warnings admission.Warnings

	// Validate intent field
	if err := r.validateIntent(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Validate intent type
	if err := r.validateIntentType(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Validate priority
	if err := r.validatePriority(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Validate target components
	if err := r.validateTargetComponents(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Add warnings for resource constraints
	if r.Spec.ResourceConstraints != nil && r.Spec.ResourceConstraints.CPU != nil && r.Spec.ResourceConstraints.CPU.Max != nil {
		// Parse the string and check if it's a high value
		if *r.Spec.ResourceConstraints.CPU.Max == "32" || strings.Contains(*r.Spec.ResourceConstraints.CPU.Max, "32") {
			warnings = append(warnings, "spec.resourceConstraints.cpu.max is set to a very high value")
		}
	}

	if len(allErrs) > 0 {
		return warnings, allErrs.ToAggregate()
	}

	return warnings, nil
}

// validateIntent validates the intent field according to security requirements
func (r *NetworkIntent) validateIntent() *field.Error {
	if r.Spec.Intent == "" {
		return field.Required(field.NewPath("spec", "intent"), "intent cannot be empty")
	}

	if len(r.Spec.Intent) > 1000 {
		return field.TooLong(field.NewPath("spec", "intent"), r.Spec.Intent, 1000)
	}

	// Security validation - check for dangerous patterns
	dangerous := []string{"<script", "javascript:", "data:", "vbscript:", "onload=", "onerror=", "eval(", "setTimeout(", "setInterval("}
	intentLower := strings.ToLower(r.Spec.Intent)
	for _, pattern := range dangerous {
		if strings.Contains(intentLower, pattern) {
			return field.Invalid(field.NewPath("spec", "intent"), r.Spec.Intent, "intent contains potentially dangerous content")
		}
	}

	return nil
}

// validateIntentType validates the intent type
func (r *NetworkIntent) validateIntentType() *field.Error {
	validTypes := []IntentType{IntentTypeDeployment, IntentTypeOptimization, IntentTypeScaling, IntentTypeMaintenance}
	for _, validType := range validTypes {
		if r.Spec.IntentType == validType {
			return nil
		}
	}
	return field.Invalid(field.NewPath("spec", "intentType"), r.Spec.IntentType, "must be one of: deployment, optimization, scaling, maintenance")
}

// validatePriority validates the priority
func (r *NetworkIntent) validatePriority() *field.Error {
	if r.Spec.Priority == "" {
		return nil // Priority is optional
	}

	validPriorities := []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical}
	for _, validPriority := range validPriorities {
		if r.Spec.Priority == validPriority {
			return nil
		}
	}
	return field.Invalid(field.NewPath("spec", "priority"), r.Spec.Priority, "must be one of: low, medium, high, critical")
}

// validateTargetComponents validates the target components
func (r *NetworkIntent) validateTargetComponents() *field.Error {
	if len(r.Spec.TargetComponents) == 0 {
		return nil // Target components are optional
	}

	validComponents := []ORANComponent{
		ORANComponentNearRTRIC, ORANComponentSMO, ORANComponentO1, ORANComponentE2,
		ORANComponentA1, ORANComponentXApp, ORANComponentGNodeB, ORANComponentAMF,
		ORANComponentSMF, ORANComponentUPF,
	}

	for i, component := range r.Spec.TargetComponents {
		valid := false
		for _, validComponent := range validComponents {
			if component == validComponent {
				valid = true
				break
			}
		}
		if !valid {
			return field.Invalid(field.NewPath("spec", "targetComponents").Index(i), component, "invalid O-RAN component")
		}
	}

	return nil
}
