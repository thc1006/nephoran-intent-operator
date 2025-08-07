# Security Baseline Policy for Nephoran Intent Operator
# OPA/Rego policies for enforcing security best practices

package nephoran.security.baseline

import rego.v1

# Deny containers running as root
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    not container.securityContext.runAsNonRoot
    msg := sprintf("Container '%v' must set runAsNonRoot: true", [container.name])
}

# Deny containers without security contexts
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    not container.securityContext
    msg := sprintf("Container '%v' must define a securityContext", [container.name])
}

# Deny privileged containers
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    container.securityContext.privileged
    msg := sprintf("Container '%v' must not run in privileged mode", [container.name])
}

# Deny containers with privileged escalation
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    container.securityContext.allowPrivilegeEscalation
    msg := sprintf("Container '%v' must not allow privilege escalation", [container.name])
}

# Require resource limits
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    not container.resources.limits.memory
    msg := sprintf("Container '%v' must define memory limits", [container.name])
}

deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    not container.resources.limits.cpu
    msg := sprintf("Container '%v' must define CPU limits", [container.name])
}

# Deny containers without readiness probes
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    not container.readinessProbe
    msg := sprintf("Container '%v' must define a readinessProbe", [container.name])
}

# Deny containers without liveness probes
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    not container.livenessProbe
    msg := sprintf("Container '%v' must define a livenessProbe", [container.name])
}

# Require specific labels for security tracking
required_labels := {
    "security.nephoran.com/scanned",
    "app.kubernetes.io/name",
    "app.kubernetes.io/version"
}

deny contains msg if {
    some label in required_labels
    not input.metadata.labels[label]
    msg := sprintf("Resource must have label '%v'", [label])
}

# Deny hostNetwork usage
deny contains msg if {
    input.kind == "Pod"
    input.spec.hostNetwork
    msg := "Pod must not use hostNetwork"
}

# Deny hostPID usage
deny contains msg if {
    input.kind == "Pod"
    input.spec.hostPID
    msg := "Pod must not use hostPID"
}

# Deny hostIPC usage
deny contains msg if {
    input.kind == "Pod"
    input.spec.hostIPC
    msg := "Pod must not use hostIPC"
}

# Require read-only root filesystem
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    not container.securityContext.readOnlyRootFilesystem
    msg := sprintf("Container '%v' must set readOnlyRootFilesystem: true", [container.name])
}

# Deny dangerous capabilities
dangerous_capabilities := {
    "SYS_ADMIN",
    "NET_ADMIN",
    "SYS_TIME",
    "SYS_MODULE"
}

deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    some capability in container.securityContext.capabilities.add
    capability in dangerous_capabilities
    msg := sprintf("Container '%v' must not add dangerous capability '%v'", [container.name, capability])
}

# Require dropping ALL capabilities
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    not "ALL" in container.securityContext.capabilities.drop
    msg := sprintf("Container '%v' must drop ALL capabilities", [container.name])
}