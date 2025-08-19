/*
Copyright 2024 Nephoran.

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

package controllers

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/thc1006/nephoran-intent-operator/pkg/security/ca"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nephv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// CertificateAutomationReconciler reconciles CertificateAutomation objects
type CertificateAutomationReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Log              logr.Logger
	AutomationEngine *ca.AutomationEngine
	CAManager        *ca.CAManager
}

// CertificateAutomationSpec defines the desired state of CertificateAutomation
type CertificateAutomationSpec struct {
	// ServiceName specifies the target service name
	ServiceName string `json:"serviceName"`

	// Namespace specifies the target namespace
	Namespace string `json:"namespace"`

	// DNSNames specifies additional DNS names for the certificate
	DNSNames []string `json:"dnsNames,omitempty"`

	// IPAddresses specifies IP addresses for the certificate
	IPAddresses []string `json:"ipAddresses,omitempty"`

	// Template specifies the certificate template to use
	Template string `json:"template,omitempty"`

	// AutoRenewal enables automatic certificate renewal
	AutoRenewal bool `json:"autoRenewal,omitempty"`

	// ValidityDuration specifies certificate validity period
	ValidityDuration *metav1.Duration `json:"validityDuration,omitempty"`

	// SecretName specifies the target secret name
	SecretName string `json:"secretName,omitempty"`

	// Priority specifies provisioning priority
	Priority string `json:"priority,omitempty"`
}

// CertificateAutomationStatus defines the observed state of CertificateAutomation
type CertificateAutomationStatus struct {
	// Phase represents the current phase of certificate automation
	Phase CertificateAutomationPhase `json:"phase,omitempty"`

	// Conditions represents the current conditions
	Conditions []CertificateAutomationCondition `json:"conditions,omitempty"`

	// CertificateSerialNumber contains the serial number of the issued certificate
	CertificateSerialNumber string `json:"certificateSerialNumber,omitempty"`

	// ExpiresAt contains the certificate expiration time
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// LastRenewalTime contains the last renewal time
	LastRenewalTime *metav1.Time `json:"lastRenewalTime,omitempty"`

	// NextRenewalTime contains the next scheduled renewal time
	NextRenewalTime *metav1.Time `json:"nextRenewalTime,omitempty"`

	// SecretRef contains reference to the created secret
	SecretRef *v1.LocalObjectReference `json:"secretRef,omitempty"`

	// ValidationStatus contains certificate validation status
	ValidationStatus *CertificateValidationStatus `json:"validationStatus,omitempty"`

	// RevocationStatus contains certificate revocation status
	RevocationStatus string `json:"revocationStatus,omitempty"`

	// ObservedGeneration represents the generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// CertificateAutomationPhase represents the phase of certificate automation
type CertificateAutomationPhase string

const (
	// CertificateAutomationPhasePending indicates the request is pending
	CertificateAutomationPhasePending CertificateAutomationPhase = "Pending"

	// CertificateAutomationPhaseProvisioning indicates certificate is being provisioned
	CertificateAutomationPhaseProvisioning CertificateAutomationPhase = "Provisioning"

	// CertificateAutomationPhaseReady indicates certificate is ready
	CertificateAutomationPhaseReady CertificateAutomationPhase = "Ready"

	// CertificateAutomationPhaseRenewing indicates certificate is being renewed
	CertificateAutomationPhaseRenewing CertificateAutomationPhase = "Renewing"

	// CertificateAutomationPhaseExpired indicates certificate has expired
	CertificateAutomationPhaseExpired CertificateAutomationPhase = "Expired"

	// CertificateAutomationPhaseRevoked indicates certificate has been revoked
	CertificateAutomationPhaseRevoked CertificateAutomationPhase = "Revoked"

	// CertificateAutomationPhaseFailed indicates an error occurred
	CertificateAutomationPhaseFailed CertificateAutomationPhase = "Failed"
)

// CertificateAutomationCondition describes the state of certificate automation
type CertificateAutomationCondition struct {
	// Type of condition
	Type CertificateAutomationConditionType `json:"type"`

	// Status of the condition
	Status metav1.ConditionStatus `json:"status"`

	// Last time the condition transitioned
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Reason for the condition's last transition
	Reason string `json:"reason,omitempty"`

	// Message providing details about the transition
	Message string `json:"message,omitempty"`
}

// CertificateAutomationConditionType represents condition types
type CertificateAutomationConditionType string

const (
	// CertificateAutomationConditionReady indicates the certificate is ready
	CertificateAutomationConditionReady CertificateAutomationConditionType = "Ready"

	// CertificateAutomationConditionIssued indicates the certificate has been issued
	CertificateAutomationConditionIssued CertificateAutomationConditionType = "Issued"

	// CertificateAutomationConditionValidated indicates the certificate has been validated
	CertificateAutomationConditionValidated CertificateAutomationConditionType = "Validated"

	// CertificateAutomationConditionRenewed indicates the certificate has been renewed
	CertificateAutomationConditionRenewed CertificateAutomationConditionType = "Renewed"

	// CertificateAutomationConditionRevoked indicates the certificate has been revoked
	CertificateAutomationConditionRevoked CertificateAutomationConditionType = "Revoked"
)

// CertificateValidationStatus contains certificate validation details
type CertificateValidationStatus struct {
	// Valid indicates if the certificate is valid
	Valid bool `json:"valid"`

	// ChainValid indicates if the certificate chain is valid
	ChainValid bool `json:"chainValid,omitempty"`

	// CTLogVerified indicates if the certificate is in CT logs
	CTLogVerified bool `json:"ctLogVerified,omitempty"`

	// LastValidationTime contains the last validation time
	LastValidationTime *metav1.Time `json:"lastValidationTime,omitempty"`

	// ValidationErrors contains validation errors
	ValidationErrors []string `json:"validationErrors,omitempty"`

	// ValidationWarnings contains validation warnings
	ValidationWarnings []string `json:"validationWarnings,omitempty"`
}

//+kubebuilder:rbac:groups=nephoran.io,resources=certificateautomations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.io,resources=certificateautomations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.io,resources=certificateautomations/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *CertificateAutomationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("certificateautomation", req.NamespacedName)

	// Fetch the CertificateAutomation instance
	var certAutomation nephv1alpha1.CertificateAutomation
	if err := r.Get(ctx, req.NamespacedName, &certAutomation); err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted
			log.Info("CertificateAutomation resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		log.Error(err, "Failed to get CertificateAutomation")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !certAutomation.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &certAutomation)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&certAutomation, "certificateautomation.nephoran.io/finalizer") {
		controllerutil.AddFinalizer(&certAutomation, "certificateautomation.nephoran.io/finalizer")
		if err := r.Update(ctx, &certAutomation); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Reconcile based on current phase
	switch certAutomation.Status.Phase {
	case "", CertificateAutomationPhasePending:
		return r.reconcileProvisioning(ctx, &certAutomation)
	case CertificateAutomationPhaseProvisioning:
		return r.reconcileProvisioningStatus(ctx, &certAutomation)
	case CertificateAutomationPhaseReady:
		return r.reconcileReady(ctx, &certAutomation)
	case CertificateAutomationPhaseRenewing:
		return r.reconcileRenewalStatus(ctx, &certAutomation)
	case CertificateAutomationPhaseExpired:
		return r.reconcileExpired(ctx, &certAutomation)
	case CertificateAutomationPhaseFailed:
		return r.reconcileFailed(ctx, &certAutomation)
	default:
		log.Info("Unknown phase, resetting to pending", "phase", certAutomation.Status.Phase)
		return r.updateStatus(ctx, &certAutomation, CertificateAutomationPhasePending, "Unknown phase, resetting")
	}
}

// reconcileDelete handles deletion of CertificateAutomation
func (r *CertificateAutomationReconciler) reconcileDelete(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) (ctrl.Result, error) {
	log := r.Log.WithValues("certificateautomation", certAutomation.Name, "namespace", certAutomation.Namespace)

	// Revoke certificate if it exists
	if certAutomation.Status.CertificateSerialNumber != "" {
		log.Info("Revoking certificate", "serial_number", certAutomation.Status.CertificateSerialNumber)

		if err := r.CAManager.RevokeCertificate(ctx, certAutomation.Status.CertificateSerialNumber, 1, "default"); err != nil {
			log.Error(err, "Failed to revoke certificate", "serial_number", certAutomation.Status.CertificateSerialNumber)
			// Continue with deletion even if revocation fails
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(certAutomation, "certificateautomation.nephoran.io/finalizer")
	if err := r.Update(ctx, certAutomation); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("CertificateAutomation deleted successfully")
	return ctrl.Result{}, nil
}

// reconcileProvisioning initiates certificate provisioning
func (r *CertificateAutomationReconciler) reconcileProvisioning(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) (ctrl.Result, error) {
	log := r.Log.WithValues("certificateautomation", certAutomation.Name, "namespace", certAutomation.Namespace)

	// Build DNS names
	dnsNames := []string{
		fmt.Sprintf("%s.%s.svc.cluster.local", certAutomation.Spec.ServiceName, certAutomation.Spec.Namespace),
		fmt.Sprintf("%s.%s", certAutomation.Spec.ServiceName, certAutomation.Spec.Namespace),
		certAutomation.Spec.ServiceName,
	}

	// Add additional DNS names
	dnsNames = append(dnsNames, certAutomation.Spec.DNSNames...)

	// Create provisioning request
	priority := ca.PriorityNormal
	switch certAutomation.Spec.Priority {
	case "low":
		priority = ca.PriorityLow
	case "high":
		priority = ca.PriorityHigh
	case "critical":
		priority = ca.PriorityCritical
	}

	req := &ca.ProvisioningRequest{
		ID:          string(certAutomation.UID),
		ServiceName: certAutomation.Spec.ServiceName,
		Namespace:   certAutomation.Spec.Namespace,
		Template:    certAutomation.Spec.Template,
		DNSNames:    dnsNames,
		IPAddresses: certAutomation.Spec.IPAddresses,
		Priority:    priority,
		Metadata: map[string]string{
			"kubernetes_managed": "true",
			"resource_name":      certAutomation.Name,
			"resource_namespace": certAutomation.Namespace,
			"auto_renew":         fmt.Sprintf("%v", certAutomation.Spec.AutoRenewal),
		},
	}

	// Submit provisioning request
	if err := r.AutomationEngine.RequestProvisioning(req); err != nil {
		log.Error(err, "Failed to request certificate provisioning")
		return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseFailed, fmt.Sprintf("Provisioning request failed: %v", err))
	}

	log.Info("Certificate provisioning requested")
	return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseProvisioning, "Certificate provisioning in progress")
}

// reconcileProvisioningStatus checks provisioning status
func (r *CertificateAutomationReconciler) reconcileProvisioningStatus(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) (ctrl.Result, error) {
	log := r.Log.WithValues("certificateautomation", certAutomation.Name, "namespace", certAutomation.Namespace)

	// Check if secret exists (indicates successful provisioning)
	secretName := certAutomation.Spec.SecretName
	if secretName == "" {
		secretName = fmt.Sprintf("%s-tls", certAutomation.Spec.ServiceName)
	}

	secret := &v1.Secret{}
	secretKey := types.NamespacedName{
		Name:      secretName,
		Namespace: certAutomation.Spec.Namespace,
	}

	if err := r.Get(ctx, secretKey, secret); err != nil {
		if apierrors.IsNotFound(err) {
			// Secret not yet created, check if provisioning has failed
			// This is a simplified check - real implementation would track provisioning status
			log.Info("Certificate secret not found, continuing to wait")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		log.Error(err, "Failed to get certificate secret")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Extract certificate information from secret
	certPEM, exists := secret.Data["tls.crt"]
	if !exists {
		log.Error(nil, "Certificate data not found in secret")
		return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseFailed, "Certificate data not found in secret")
	}

	// Parse certificate to get details
	cert, err := r.parseCertificateFromPEM(certPEM)
	if err != nil {
		log.Error(err, "Failed to parse certificate")
		return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseFailed, fmt.Sprintf("Failed to parse certificate: %v", err))
	}

	// Update status with certificate information
	certAutomation.Status.CertificateSerialNumber = cert.SerialNumber.String()
	certAutomation.Status.ExpiresAt = &metav1.Time{Time: cert.NotAfter}
	certAutomation.Status.SecretRef = &v1.LocalObjectReference{Name: secretName}

	// Calculate next renewal time
	if certAutomation.Spec.AutoRenewal {
		renewalThreshold := time.Duration(30 * 24 * time.Hour) // 30 days before expiry
		nextRenewal := cert.NotAfter.Add(-renewalThreshold)
		certAutomation.Status.NextRenewalTime = &metav1.Time{Time: nextRenewal}
	}

	// Validate certificate
	if r.AutomationEngine != nil {
		validationResult, err := r.AutomationEngine.ValidateCertificate(cert)
		if err != nil {
			log.Warn("Certificate validation failed", "error", err)
		} else {
			certAutomation.Status.ValidationStatus = &CertificateValidationStatus{
				Valid:              validationResult.Valid,
				ChainValid:         validationResult.ChainValid,
				CTLogVerified:      validationResult.CTLogVerified,
				LastValidationTime: &metav1.Time{Time: validationResult.ValidationTime},
				ValidationErrors:   validationResult.Errors,
				ValidationWarnings: validationResult.Warnings,
			}
		}

		// Check revocation status
		revocationStatus, err := r.AutomationEngine.CheckRevocationStatus(cert)
		if err != nil {
			log.Warn("Revocation check failed", "error", err)
		} else {
			certAutomation.Status.RevocationStatus = string(revocationStatus)
		}
	}

	log.Info("Certificate provisioned successfully",
		"serial_number", certAutomation.Status.CertificateSerialNumber,
		"expires_at", certAutomation.Status.ExpiresAt.Time)

	return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseReady, "Certificate ready")
}

// reconcileReady handles ready state monitoring
func (r *CertificateAutomationReconciler) reconcileReady(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) (ctrl.Result, error) {
	log := r.Log.WithValues("certificateautomation", certAutomation.Name, "namespace", certAutomation.Namespace)

	now := time.Now()

	// Check if certificate has expired
	if certAutomation.Status.ExpiresAt != nil && certAutomation.Status.ExpiresAt.Time.Before(now) {
		log.Info("Certificate has expired")
		return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseExpired, "Certificate expired")
	}

	// Check if renewal is needed
	if certAutomation.Spec.AutoRenewal &&
		certAutomation.Status.NextRenewalTime != nil &&
		certAutomation.Status.NextRenewalTime.Time.Before(now) {
		log.Info("Initiating certificate renewal")
		return r.initiateRenewal(ctx, certAutomation)
	}

	// Periodic validation if enabled
	if certAutomation.Status.ValidationStatus == nil ||
		(certAutomation.Status.ValidationStatus.LastValidationTime != nil &&
			now.Sub(certAutomation.Status.ValidationStatus.LastValidationTime.Time) > 24*time.Hour) {
		go r.performPeriodicValidation(ctx, certAutomation)
	}

	// Requeue for next check
	var requeueAfter time.Duration
	if certAutomation.Status.NextRenewalTime != nil {
		requeueAfter = time.Until(certAutomation.Status.NextRenewalTime.Time)
		if requeueAfter <= 0 {
			requeueAfter = 1 * time.Hour // Minimum requeue time
		}
	} else {
		requeueAfter = 24 * time.Hour // Daily check
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// reconcileRenewalStatus checks renewal status
func (r *CertificateAutomationReconciler) reconcileRenewalStatus(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) (ctrl.Result, error) {
	// Similar to reconcileProvisioningStatus but for renewal
	return r.reconcileProvisioningStatus(ctx, certAutomation)
}

// reconcileExpired handles expired certificates
func (r *CertificateAutomationReconciler) reconcileExpired(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) (ctrl.Result, error) {
	log := r.Log.WithValues("certificateautomation", certAutomation.Name, "namespace", certAutomation.Namespace)

	if certAutomation.Spec.AutoRenewal {
		log.Info("Attempting renewal of expired certificate")
		return r.initiateRenewal(ctx, certAutomation)
	}

	// Manual intervention required
	log.Warn("Certificate expired and auto-renewal is disabled")
	return ctrl.Result{RequeueAfter: 24 * time.Hour}, nil
}

// reconcileFailed handles failed state
func (r *CertificateAutomationReconciler) reconcileFailed(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) (ctrl.Result, error) {
	// Implement retry logic with exponential backoff
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// initiateRenewal starts certificate renewal process
func (r *CertificateAutomationReconciler) initiateRenewal(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) (ctrl.Result, error) {
	log := r.Log.WithValues("certificateautomation", certAutomation.Name, "namespace", certAutomation.Namespace)

	if certAutomation.Status.CertificateSerialNumber == "" {
		return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseFailed, "Cannot renew: no certificate serial number")
	}

	// Create renewal request
	req := &ca.RenewalRequest{
		SerialNumber:    certAutomation.Status.CertificateSerialNumber,
		CurrentExpiry:   certAutomation.Status.ExpiresAt.Time,
		ServiceName:     certAutomation.Spec.ServiceName,
		Namespace:       certAutomation.Spec.Namespace,
		GracefulRenewal: true,
		Metadata: map[string]string{
			"kubernetes_managed": "true",
			"resource_name":      certAutomation.Name,
			"resource_namespace": certAutomation.Namespace,
		},
	}

	if err := r.AutomationEngine.RequestRenewal(req); err != nil {
		log.Error(err, "Failed to request certificate renewal")
		return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseFailed, fmt.Sprintf("Renewal request failed: %v", err))
	}

	log.Info("Certificate renewal requested")
	return r.updateStatus(ctx, certAutomation, CertificateAutomationPhaseRenewing, "Certificate renewal in progress")
}

// performPeriodicValidation performs background certificate validation
func (r *CertificateAutomationReconciler) performPeriodicValidation(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation) {
	// This would run in background to update validation status
	// Implementation would fetch the certificate and validate it
}

// updateStatus updates the CertificateAutomation status
func (r *CertificateAutomationReconciler) updateStatus(ctx context.Context, certAutomation *nephv1alpha1.CertificateAutomation, phase CertificateAutomationPhase, message string) (ctrl.Result, error) {
	certAutomation.Status.Phase = phase
	certAutomation.Status.ObservedGeneration = certAutomation.Generation

	// Update conditions
	condition := CertificateAutomationCondition{
		Type:               CertificateAutomationConditionReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             string(phase),
		Message:            message,
	}

	if phase == CertificateAutomationPhaseReady {
		condition.Status = metav1.ConditionTrue
	}

	// Update or add condition
	r.setCondition(&certAutomation.Status, condition)

	if err := r.Status().Update(ctx, certAutomation); err != nil {
		r.Log.Error(err, "Failed to update CertificateAutomation status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// setCondition sets or updates a condition
func (r *CertificateAutomationReconciler) setCondition(status *CertificateAutomationStatus, condition CertificateAutomationCondition) {
	for i, existingCondition := range status.Conditions {
		if existingCondition.Type == condition.Type {
			if existingCondition.Status != condition.Status {
				condition.LastTransitionTime = metav1.Now()
			} else {
				condition.LastTransitionTime = existingCondition.LastTransitionTime
			}
			status.Conditions[i] = condition
			return
		}
	}

	// Condition not found, add it
	status.Conditions = append(status.Conditions, condition)
}

// Helper function to parse certificate from PEM data
func (r *CertificateAutomationReconciler) parseCertificateFromPEM(pemData []byte) (*x509.Certificate, error) {
	// This would implement PEM parsing and certificate extraction
	// Placeholder implementation
	return nil, fmt.Errorf("not implemented")
}

// SetupWithManager sets up the controller with the Manager
func (r *CertificateAutomationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephv1alpha1.CertificateAutomation{}).
		Owns(&v1.Secret{}).
		Watches(
			&source.Kind{Type: &v1.Service{}},
			handler.EnqueueRequestsFromMapFunc(r.findCertificateAutomationsForService),
		).
		Complete(r)
}

// findCertificateAutomationsForService finds CertificateAutomations that reference a Service
func (r *CertificateAutomationReconciler) findCertificateAutomationsForService(obj client.Object) []reconcile.Request {
	service := obj.(*v1.Service)

	var certAutomationList nephv1alpha1.CertificateAutomationList
	if err := r.List(context.Background(), &certAutomationList, client.InNamespace(service.Namespace)); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, certAutomation := range certAutomationList.Items {
		if certAutomation.Spec.ServiceName == service.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      certAutomation.Name,
					Namespace: certAutomation.Namespace,
				},
			})
		}
	}

	return requests
}
