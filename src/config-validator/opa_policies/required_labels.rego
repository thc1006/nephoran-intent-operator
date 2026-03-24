# Required labels enforcement per CLAUDE.md Section 6.
#
# All managed workloads and services must have:
#   - oran.ai/intent-id
#   - oran.ai/component
#   - app.kubernetes.io/managed-by
#   - app.kubernetes.io/part-of

package nephoran.labels

import rego.v1

workload_kinds := {"Deployment", "StatefulSet", "DaemonSet", "Pod", "Job", "CronJob", "Service"}

required_labels := {
    "oran.ai/intent-id",
    "oran.ai/component",
    "app.kubernetes.io/managed-by",
    "app.kubernetes.io/part-of",
}

violation contains msg if {
    input.kind in workload_kinds
    label := required_labels[_]
    not input.metadata.labels[label]
    msg := sprintf("[L5-Labels] missing required label '%s' on %s/%s", [label, input.kind, input.metadata.name])
}
