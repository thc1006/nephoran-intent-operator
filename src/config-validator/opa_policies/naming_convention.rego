package nephio.naming

import rego.v1

violation contains msg if {
    input.kind in {"Deployment", "StatefulSet", "Pod"}
    name := input.metadata.name
    not regex.match(`^(odu|ocu-cp|ocu-up|amf|smf|upf|nrf|nssf)-(embb|urllc|mmtc|custom-\d+)-\d{3}(-scaled-\d{2})?$`, name)
    msg := sprintf("Name '%s' violates CNF naming convention", [name])
}
