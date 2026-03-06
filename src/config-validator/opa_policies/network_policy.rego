package nephio.network

import rego.v1

# Ensure CNF namespaces have NetworkPolicy
violation contains msg if {
    input.kind == "Namespace"
    input.metadata.labels["nephio.org/managed"] == "true"
    msg := "Managed namespace should have an associated NetworkPolicy"
}
