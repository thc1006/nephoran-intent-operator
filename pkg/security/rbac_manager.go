// Package security provides comprehensive security management for the Nephoran Intent Operator
package security

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RBACManager manages RBAC policies with least privilege principle
type RBACManager struct {
	client    client.Client
	clientset *kubernetes.Clientset
	namespace string
}

// NewRBACManager creates a new RBAC manager instance
func NewRBACManager(client client.Client, clientset *kubernetes.Clientset, namespace string) *RBACManager {
	return &RBACManager{
		client:    client,
		clientset: clientset,
		namespace: namespace,
	}
}

// OperatorRole defines different operator personas with specific permissions
type OperatorRole string

const (
	// RoleNetworkOperator can view and manage network intents
	RoleNetworkOperator OperatorRole = "network-operator"
	// RoleNetworkViewer can only view network intents and status
	RoleNetworkViewer OperatorRole = "network-viewer"
	// RoleSecurityAuditor can view all resources and audit logs
	RoleSecurityAuditor OperatorRole = "security-auditor"
	// RoleClusterAdmin has full administrative privileges
	RoleClusterAdmin OperatorRole = "cluster-admin"
	// RoleServiceOperator can manage specific services
	RoleServiceOperator OperatorRole = "service-operator"
)

// RoleDefinition contains RBAC rules for a specific role
type RoleDefinition struct {
	Name        string
	Rules       []rbacv1.PolicyRule
	ClusterRole bool
	Labels      map[string]string
}

// GetRoleDefinitions returns role definitions following least privilege principle
func (m *RBACManager) GetRoleDefinitions() map[OperatorRole]RoleDefinition {
	return map[OperatorRole]RoleDefinition{
		RoleNetworkOperator: {
			Name:        "nephoran-network-operator",
			ClusterRole: false,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "rbac",
				"security.nephoran.io/level":  "operator",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"nephoran.io"},
					Resources: []string{"networkintents", "networkintents/status"},
					Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
				},
				{
					APIGroups: []string{"porch.kpt.dev"},
					Resources: []string{"packagerevisions", "packagerevisionresources"},
					Verbs:     []string{"get", "list", "watch", "create", "update", "patch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "secrets"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs:     []string{"create", "patch"},
				},
			},
		},
		RoleNetworkViewer: {
			Name:        "nephoran-network-viewer",
			ClusterRole: false,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "rbac",
				"security.nephoran.io/level":  "viewer",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"nephoran.io"},
					Resources: []string{"networkintents", "networkintents/status"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"porch.kpt.dev"},
					Resources: []string{"packagerevisions", "packagerevisionresources"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get", "list"},
				},
			},
		},
		RoleSecurityAuditor: {
			Name:        "nephoran-security-auditor",
			ClusterRole: true,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "rbac",
				"security.nephoran.io/level":  "auditor",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"audit.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"admissionregistration.k8s.io"},
					Resources: []string{"validatingwebhookconfigurations", "mutatingwebhookconfigurations"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
		RoleServiceOperator: {
			Name:        "nephoran-service-operator",
			ClusterRole: false,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "rbac",
				"security.nephoran.io/level":  "service",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apps"},
					Resources: []string{"deployments", "statefulsets", "daemonsets"},
					Verbs:     []string{"get", "list", "watch", "update", "patch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"services", "endpoints", "pods"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods/log"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{"autoscaling"},
					Resources: []string{"horizontalpodautoscalers"},
					Verbs:     []string{"get", "list", "watch", "create", "update", "patch"},
				},
			},
		},
	}
}

// CreateRole creates a Role or ClusterRole with least privilege permissions
func (m *RBACManager) CreateRole(ctx context.Context, role OperatorRole) error {
	logger := log.FromContext(ctx)
	definitions := m.GetRoleDefinitions()
	
	def, exists := definitions[role]
	if !exists {
		return fmt.Errorf("unknown role: %s", role)
	}

	if def.ClusterRole {
		clusterRole := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:   def.Name,
				Labels: def.Labels,
			},
			Rules: def.Rules,
		}
		
		if err := m.client.Create(ctx, clusterRole); err != nil {
			logger.Error(err, "Failed to create ClusterRole", "role", def.Name)
			return fmt.Errorf("failed to create ClusterRole %s: %w", def.Name, err)
		}
	} else {
		role := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      def.Name,
				Namespace: m.namespace,
				Labels:    def.Labels,
			},
			Rules: def.Rules,
		}
		
		if err := m.client.Create(ctx, role); err != nil {
			logger.Error(err, "Failed to create Role", "role", def.Name)
			return fmt.Errorf("failed to create Role %s: %w", def.Name, err)
		}
	}

	logger.Info("Created RBAC role", "role", def.Name, "clusterRole", def.ClusterRole)
	return nil
}

// CreateServiceAccount creates a ServiceAccount with appropriate labels
func (m *RBACManager) CreateServiceAccount(ctx context.Context, name string, role OperatorRole) error {
	logger := log.FromContext(ctx)
	
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "rbac",
				"security.nephoran.io/role":   string(role),
			},
			Annotations: map[string]string{
				"security.nephoran.io/created-by": "rbac-manager",
				"security.nephoran.io/purpose":    "operator-authentication",
			},
		},
	}

	if err := m.client.Create(ctx, sa); err != nil {
		logger.Error(err, "Failed to create ServiceAccount", "name", name)
		return fmt.Errorf("failed to create ServiceAccount %s: %w", name, err)
	}

	logger.Info("Created ServiceAccount", "name", name, "role", role)
	return nil
}

// BindRoleToServiceAccount creates RoleBinding or ClusterRoleBinding
func (m *RBACManager) BindRoleToServiceAccount(ctx context.Context, saName string, role OperatorRole) error {
	logger := log.FromContext(ctx)
	definitions := m.GetRoleDefinitions()
	
	def, exists := definitions[role]
	if !exists {
		return fmt.Errorf("unknown role: %s", role)
	}

	bindingName := fmt.Sprintf("%s-binding", saName)
	
	if def.ClusterRole {
		binding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: bindingName,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "nephoran-intent-operator",
					"app.kubernetes.io/component": "rbac",
					"security.nephoran.io/role":   string(role),
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     def.Name,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      saName,
					Namespace: m.namespace,
				},
			},
		}
		
		if err := m.client.Create(ctx, binding); err != nil {
			logger.Error(err, "Failed to create ClusterRoleBinding", "binding", bindingName)
			return fmt.Errorf("failed to create ClusterRoleBinding %s: %w", bindingName, err)
		}
	} else {
		binding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      bindingName,
				Namespace: m.namespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "nephoran-intent-operator",
					"app.kubernetes.io/component": "rbac",
					"security.nephoran.io/role":   string(role),
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     def.Name,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      saName,
					Namespace: m.namespace,
				},
			},
		}
		
		if err := m.client.Create(ctx, binding); err != nil {
			logger.Error(err, "Failed to create RoleBinding", "binding", bindingName)
			return fmt.Errorf("failed to create RoleBinding %s: %w", bindingName, err)
		}
	}

	logger.Info("Created role binding", "binding", bindingName, "serviceAccount", saName, "role", def.Name)
	return nil
}

// ValidatePermissions validates that permissions follow least privilege principle
func (m *RBACManager) ValidatePermissions(ctx context.Context, rules []rbacv1.PolicyRule) error {
	for _, rule := range rules {
		// Check for wildcard permissions (violates least privilege)
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "*" {
				return fmt.Errorf("wildcard API group not allowed: use specific API groups")
			}
		}
		
		for _, resource := range rule.Resources {
			if resource == "*" {
				return fmt.Errorf("wildcard resource not allowed: use specific resources")
			}
		}
		
		for _, verb := range rule.Verbs {
			if verb == "*" {
				return fmt.Errorf("wildcard verb not allowed: use specific verbs")
			}
		}
		
		// Check for dangerous verb combinations
		hasDelete := contains(rule.Verbs, "delete")
		hasDeleteCollection := contains(rule.Verbs, "deletecollection")
		if hasDelete && hasDeleteCollection {
			return fmt.Errorf("both delete and deletecollection verbs present: consider separating concerns")
		}
		
		// Check for escalation permissions
		if contains(rule.Resources, "clusterroles") || contains(rule.Resources, "clusterrolebindings") {
			if contains(rule.Verbs, "create") || contains(rule.Verbs, "update") || contains(rule.Verbs, "patch") {
				return fmt.Errorf("privilege escalation risk: modifying cluster RBAC requires additional validation")
			}
		}
	}
	
	return nil
}

// AuditRBACCompliance performs comprehensive RBAC audit
func (m *RBACManager) AuditRBACCompliance(ctx context.Context) (*RBACAuditReport, error) {
	logger := log.FromContext(ctx)
	report := &RBACAuditReport{
		Timestamp: metav1.Now(),
		Namespace: m.namespace,
		Issues:    []string{},
		Warnings:  []string{},
	}

	// Audit ClusterRoles
	clusterRoles := &rbacv1.ClusterRoleList{}
	if err := m.client.List(ctx, clusterRoles); err != nil {
		return nil, fmt.Errorf("failed to list ClusterRoles: %w", err)
	}

	for _, cr := range clusterRoles.Items {
		if err := m.ValidatePermissions(ctx, cr.Rules); err != nil {
			report.Issues = append(report.Issues, fmt.Sprintf("ClusterRole %s: %s", cr.Name, err.Error()))
		}
		
		// Check for system:masters binding (critical security issue)
		if cr.Name == "cluster-admin" {
			report.Warnings = append(report.Warnings, "cluster-admin role detected: ensure minimal usage")
		}
	}

	// Audit Roles in namespace
	roles := &rbacv1.RoleList{}
	if err := m.client.List(ctx, roles, client.InNamespace(m.namespace)); err != nil {
		return nil, fmt.Errorf("failed to list Roles: %w", err)
	}

	for _, r := range roles.Items {
		if err := m.ValidatePermissions(ctx, r.Rules); err != nil {
			report.Issues = append(report.Issues, fmt.Sprintf("Role %s: %s", r.Name, err.Error()))
		}
	}

	// Audit ServiceAccounts
	serviceAccounts := &corev1.ServiceAccountList{}
	if err := m.client.List(ctx, serviceAccounts, client.InNamespace(m.namespace)); err != nil {
		return nil, fmt.Errorf("failed to list ServiceAccounts: %w", err)
	}

	report.ServiceAccountCount = len(serviceAccounts.Items)
	
	// Check for default service account usage
	for _, sa := range serviceAccounts.Items {
		if sa.Name == "default" && len(sa.Secrets) > 0 {
			report.Warnings = append(report.Warnings, "default ServiceAccount has mounted secrets: consider using dedicated ServiceAccounts")
		}
	}

	report.Compliant = len(report.Issues) == 0
	logger.Info("RBAC audit completed", "compliant", report.Compliant, "issues", len(report.Issues), "warnings", len(report.Warnings))
	
	return report, nil
}

// RBACAuditReport contains RBAC compliance audit results
type RBACAuditReport struct {
	Timestamp           metav1.Time
	Namespace           string
	Compliant           bool
	Issues              []string
	Warnings            []string
	ServiceAccountCount int
}

// EnforceMinimalPermissions ensures minimal required permissions for operation
func (m *RBACManager) EnforceMinimalPermissions(ctx context.Context) error {
	logger := log.FromContext(ctx)
	
	// Create minimal operator role
	minimalRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nephoran-minimal-operator",
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"security.nephoran.io/minimal": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"nephoran.io"},
				Resources: []string{"networkintents"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create"},
			},
		},
	}
	
	if err := m.client.Create(ctx, minimalRole); err != nil {
		logger.Error(err, "Failed to create minimal role")
		return fmt.Errorf("failed to create minimal role: %w", err)
	}
	
	logger.Info("Enforced minimal permissions")
	return nil
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// GetServiceAccountToken retrieves a token for a ServiceAccount
func (m *RBACManager) GetServiceAccountToken(ctx context.Context, saName string) (string, error) {
	sa := &corev1.ServiceAccount{}
	if err := m.client.Get(ctx, client.ObjectKey{
		Name:      saName,
		Namespace: m.namespace,
	}, sa); err != nil {
		return "", fmt.Errorf("failed to get ServiceAccount: %w", err)
	}

	if len(sa.Secrets) == 0 {
		return "", fmt.Errorf("no secrets found for ServiceAccount %s", saName)
	}

	// Get the first secret (usually the token)
	secret := &corev1.Secret{}
	if err := m.client.Get(ctx, client.ObjectKey{
		Name:      sa.Secrets[0].Name,
		Namespace: m.namespace,
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	token, exists := secret.Data["token"]
	if !exists {
		return "", fmt.Errorf("token not found in secret %s", secret.Name)
	}

	return string(token), nil
}

// ValidateServiceAccountPermissions validates a ServiceAccount has expected permissions
func (m *RBACManager) ValidateServiceAccountPermissions(ctx context.Context, saName string, expectedRole OperatorRole) error {
	// Get RoleBindings for the ServiceAccount
	roleBindings := &rbacv1.RoleBindingList{}
	if err := m.client.List(ctx, roleBindings, client.InNamespace(m.namespace)); err != nil {
		return fmt.Errorf("failed to list RoleBindings: %w", err)
	}

	definitions := m.GetRoleDefinitions()
	expectedDef, exists := definitions[expectedRole]
	if !exists {
		return fmt.Errorf("unknown role: %s", expectedRole)
	}

	found := false
	for _, rb := range roleBindings.Items {
		for _, subject := range rb.Subjects {
			if subject.Kind == "ServiceAccount" && subject.Name == saName {
				if rb.RoleRef.Name == expectedDef.Name {
					found = true
					break
				}
			}
		}
	}

	if !found {
		return fmt.Errorf("ServiceAccount %s does not have expected role %s", saName, expectedRole)
	}

	return nil
}