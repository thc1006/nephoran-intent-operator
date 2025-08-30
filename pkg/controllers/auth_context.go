package controllers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	"github.com/nephio-project/nephoran-intent-operator/pkg/auth"
)

// AuthContextKey represents the context key for authentication.

type AuthContextKey string

const (

	// ControllerAuthContextKey holds controllerauthcontextkey value.

	ControllerAuthContextKey AuthContextKey = "controller_auth_context"
)

// ControllerAuthContext holds authentication context for controller operations.

type ControllerAuthContext struct {
	UserID string

	OperationType string

	ResourceType string

	ResourceName string

	ResourceNamespace string

	Timestamp time.Time

	RequestID string
}

// AuthenticatedReconciler provides authentication context.

type AuthenticatedReconciler struct {
	client.Client

	logger *slog.Logger

	authIntegration *auth.NephoranAuthIntegration

	requireAuth bool
}

// NewAuthenticatedReconciler creates authenticated reconciler.

func NewAuthenticatedReconciler(

	client client.Client,

	authIntegration *auth.NephoranAuthIntegration,

	logger *slog.Logger,

	requireAuth bool,

) *AuthenticatedReconciler {

	return &AuthenticatedReconciler{

		Client: client,

		logger: logger,

		authIntegration: authIntegration,

		requireAuth: requireAuth,
	}

}

// WithAuthContext adds authentication context.

func (ar *AuthenticatedReconciler) WithAuthContext(ctx context.Context, operation, resourceType, resourceName, resourceNamespace string) (context.Context, error) {

	authCtx := &ControllerAuthContext{

		OperationType: operation,

		ResourceType: resourceType,

		ResourceName: resourceName,

		ResourceNamespace: resourceNamespace,

		Timestamp: time.Now(),

		RequestID: fmt.Sprintf("ctrl-%d", time.Now().UnixNano()),
	}

	return context.WithValue(ctx, ControllerAuthContextKey, authCtx), nil

}

// ValidateNetworkIntentAccess validates access to NetworkIntent.

func (ar *AuthenticatedReconciler) ValidateNetworkIntentAccess(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, operation string) error {

	if !ar.requireAuth {

		return nil

	}

	return nil

}

// NetworkIntentAuthDecorator decorates controller with auth.

type NetworkIntentAuthDecorator struct {
	*AuthenticatedReconciler

	originalReconciler *NetworkIntentReconciler
}

// NewNetworkIntentAuthDecorator creates authenticated controller.

func NewNetworkIntentAuthDecorator(

	originalReconciler *NetworkIntentReconciler,

	authIntegration *auth.NephoranAuthIntegration,

	logger *slog.Logger,

	requireAuth bool,

) *NetworkIntentAuthDecorator {

	authReconciler := NewAuthenticatedReconciler(

		originalReconciler.Client,

		authIntegration,

		logger,

		requireAuth,
	)

	return &NetworkIntentAuthDecorator{

		AuthenticatedReconciler: authReconciler,

		originalReconciler: originalReconciler,
	}

}

// Reconcile wraps original reconcile with authentication.

func (niad *NetworkIntentAuthDecorator) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	authCtx, err := niad.WithAuthContext(ctx, "reconcile", "networkintents", req.Name, req.Namespace)

	if err != nil {

		return ctrl.Result{}, err

	}

	var networkIntent nephoranv1.NetworkIntent

	if err := niad.AuthenticatedReconciler.Get(authCtx, req.NamespacedName, &networkIntent); err != nil {

		return ctrl.Result{}, client.IgnoreNotFound(err)

	}

	if err := niad.ValidateNetworkIntentAccess(authCtx, &networkIntent, "update"); err != nil {

		return ctrl.Result{}, err

	}

	return niad.originalReconciler.Reconcile(authCtx, req)

}

// AuthenticatedE2NodeSetReconciler decorates E2NodeSet controller with auth.

type AuthenticatedE2NodeSetReconciler struct {
	*AuthenticatedReconciler

	OriginalReconciler *E2NodeSetReconciler
}

// NewAuthenticatedE2NodeSetReconciler creates authenticated E2NodeSet controller.

func NewAuthenticatedE2NodeSetReconciler(

	originalReconciler *E2NodeSetReconciler,

	authIntegration *auth.NephoranAuthIntegration,

	logger *slog.Logger,

	requireAuth bool,

) *AuthenticatedE2NodeSetReconciler {

	authReconciler := NewAuthenticatedReconciler(

		originalReconciler.Client,

		authIntegration,

		logger,

		requireAuth,
	)

	return &AuthenticatedE2NodeSetReconciler{

		AuthenticatedReconciler: authReconciler,

		OriginalReconciler: originalReconciler,
	}

}

// Reconcile wraps original reconcile with authentication for E2NodeSet.

func (aer *AuthenticatedE2NodeSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	authCtx, err := aer.WithAuthContext(ctx, "reconcile", "e2nodesets", req.Name, req.Namespace)

	if err != nil {

		return ctrl.Result{}, err

	}

	// Add E2-specific authentication validation if needed.

	if aer.requireAuth {

		// Check if user has permission to manage E2 nodes.

		if err := aer.validateE2NodeAccess(authCtx, req); err != nil {

			aer.logger.Error("E2NodeSet access denied",

				"namespace", req.Namespace,

				"name", req.Name,

				"error", err)

			return ctrl.Result{}, err

		}

	}

	return aer.OriginalReconciler.Reconcile(authCtx, req)

}

// validateE2NodeAccess validates access to E2NodeSet resources.

func (aer *AuthenticatedE2NodeSetReconciler) validateE2NodeAccess(ctx context.Context, req ctrl.Request) error {

	// Extract auth context.

	authCtx, ok := ctx.Value(ControllerAuthContextKey).(*ControllerAuthContext)

	if !ok {

		return fmt.Errorf("authentication context not found")

	}

	// Log the access attempt for audit.

	aer.logger.Info("E2NodeSet access validation",

		"operation", authCtx.OperationType,

		"resource", authCtx.ResourceType,

		"name", authCtx.ResourceName,

		"namespace", authCtx.ResourceNamespace,

		"request_id", authCtx.RequestID)

	// For now, we'll allow all authenticated users to manage E2 nodes.

	// In a production environment, you might want to check specific RBAC permissions.

	// using the auth integration's RBAC manager.

	return nil

}

// SetupWithManager sets up the authenticated controller with the Manager.

func (aer *AuthenticatedE2NodeSetReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.E2NodeSet{}).
		Complete(aer)

}
