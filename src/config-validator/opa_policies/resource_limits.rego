# Resource limits enforcement per ADR-008.
# All containers must have resource limits defined.

package nephoran.resources

import rego.v1

workload_kinds := {"Deployment", "StatefulSet", "DaemonSet"}

violation contains msg if {
    input.kind in workload_kinds
    container := input.spec.template.spec.containers[_]
    not container.resources.limits
    msg := sprintf("[L3-Resources] container '%s' missing resource limits", [container.name])
}

violation contains msg if {
    input.kind in workload_kinds
    container := input.spec.template.spec.containers[_]
    not container.resources.requests
    msg := sprintf("[L3-Resources] container '%s' missing resource requests", [container.name])
}
