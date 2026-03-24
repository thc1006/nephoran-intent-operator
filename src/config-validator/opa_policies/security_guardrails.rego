# Security guardrails per CLAUDE.md Section 5 (IntentPlan.policy.guardrails).
#
# Denies:
#   - Cluster-scoped resources (denyClusterScoped) — includes CRDs, ClusterRole, etc.
#   - Privileged containers (denyPrivileged)
#   - hostNetwork usage (denyHostNetwork)
#   - Namespace-scoped RBAC mutations (denyRBACChanges) — Role, RoleBinding

package nephoran.security

import rego.v1

# Cluster-scoped kinds that are denied
cluster_scoped_kinds := {
    "ClusterRole", "ClusterRoleBinding", "Namespace",
    "PersistentVolume", "StorageClass", "CustomResourceDefinition",
    "PriorityClass", "ValidatingWebhookConfiguration", "MutatingWebhookConfiguration",
}

workload_kinds := {"Deployment", "StatefulSet", "DaemonSet", "Pod", "Job", "CronJob"}

# Deny cluster-scoped resources
violation contains msg if {
    input.kind in cluster_scoped_kinds
    msg := sprintf("[L3-Policy] cluster-scoped resource '%s' not allowed (denyClusterScoped)", [input.kind])
}

# Deny privileged containers
violation contains msg if {
    input.kind in workload_kinds
    containers := _pod_spec.containers
    container := containers[_]
    container.securityContext.privileged == true
    msg := sprintf("[L3-Policy] container '%s' is privileged (denyPrivileged)", [container.name])
}

# Deny hostNetwork
violation contains msg if {
    input.kind in workload_kinds
    _pod_spec.hostNetwork == true
    msg := sprintf("[L3-Policy] hostNetwork=true not allowed (denyHostNetwork)", [])
}

# Deny namespace-scoped RBAC changes (cluster-scoped RBAC already caught by denyClusterScoped)
violation contains msg if {
    input.kind in {"Role", "RoleBinding"}
    msg := sprintf("[L3-Policy] RBAC resource '%s' not allowed (denyRBACChanges)", [input.kind])
}

# Helper: extract pod spec from workload
_pod_spec := input.spec if {
    input.kind == "Pod"
}

_pod_spec := input.spec.template.spec if {
    input.kind in {"Deployment", "StatefulSet", "DaemonSet", "Job"}
}

_pod_spec := input.spec.jobTemplate.spec.template.spec if {
    input.kind == "CronJob"
}
