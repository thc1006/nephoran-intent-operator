package nephio.resources

import rego.v1

violation contains msg if {
    container := input.spec.template.spec.containers[_]
    not container.resources.limits
    msg := sprintf("Container '%s' missing resource limits", [container.name])
}
