# Naming convention enforcement per CLAUDE.md Section 6.
# Pattern: {domain}-{component}-{site}-{slice}-{instance}
#
# Applies to: Deployment, StatefulSet, DaemonSet, Pod, Service

package nephoran.naming

import rego.v1

domains := {"ran", "core", "ric", "sim", "obs"}

components := {
    "oai-odu", "oai-ocu", "oai-cu-cp", "oai-cu-up",
    "free5gc-upf", "free5gc-smf", "free5gc-amf",
    "ric-kpimon", "ric-ts", "sim-e2", "trafficgen",
}

sites := {"edge01", "edge02", "regional", "central", "lab"}

slices := {"embb", "urllc", "mmtc", "shared"}

workload_kinds := {"Deployment", "StatefulSet", "DaemonSet", "Pod", "Job", "CronJob", "Service"}

# Full naming regex: domain-component-site-slice-instance
naming_pattern := `^(ran|core|ric|sim|obs)-(oai-odu|oai-ocu|oai-cu-cp|oai-cu-up|free5gc-upf|free5gc-smf|free5gc-amf|ric-kpimon|ric-ts|sim-e2|trafficgen)-(edge01|edge02|regional|central|lab)-(embb|urllc|mmtc|shared)-(i\d{3})$`

violation contains msg if {
    input.kind in workload_kinds
    name := input.metadata.name
    count(split(name, "-")) >= 4
    not regex.match(naming_pattern, name)
    msg := sprintf("[L4-Naming] name '%s' violates naming convention {domain}-{component}-{site}-{slice}-{instance}", [name])
}
