# Image Security Policy for Nephoran Intent Operator
# OPA/Rego policies for container image security

package nephoran.security.images

import rego.v1

# Allowed registries for production workloads
allowed_registries := {
    "us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran",
    "gcr.io/distroless",
    "registry.k8s.io",
    "docker.io/library/alpine"
}

# Blocked registries (known vulnerabilities or policy violations)
blocked_registries := {
    "docker.io/library/node:12",
    "docker.io/library/ubuntu:18.04",
    "docker.io/library/centos:7"
}

# Deny images from blocked registries
deny contains msg if {
    input.kind == "Pod"
    some container in input.spec.containers
    image := container.image
    some blocked_registry in blocked_registries
    startswith(image, blocked_registry)
    msg := sprintf("Container '%v' uses blocked image '%v'", [container.name, image])
}

# Require images from allowed registries (in production namespace)
deny contains msg if {
    input.kind == "Pod"
    input.metadata.namespace == "nephoran-production"
    some container in input.spec.containers
    image := container.image
    not image_from_allowed_registry(image)
    msg := sprintf("Container '%v' must use image from allowed registry, got '%v'", [container.name, image])
}

image_from_allowed_registry(image) if {
    some allowed_registry in allowed_registries
    startswith(image, allowed_registry)
}

# Deny latest tags in production
deny contains msg if {
    input.kind == "Pod"
    input.metadata.namespace == "nephoran-production"
    some container in input.spec.containers
    image := container.image
    endswith(image, ":latest")
    msg := sprintf("Container '%v' must not use 'latest' tag in production", [container.name])
}

# Require vulnerability scan labels
required_scan_labels := {
    "security.nephoran.com/scanned",
    "security.nephoran.com/scan-date",
    "security.nephoran.com/trivy-version"
}

deny contains msg if {
    input.kind == "Pod"
    input.metadata.namespace == "nephoran-production"
    some label in required_scan_labels
    not input.metadata.labels[label]
    msg := sprintf("Pod must have vulnerability scan label '%v'", [label])
}

# Check scan date is recent (within 7 days)
deny contains msg if {
    input.kind == "Pod"
    input.metadata.namespace == "nephoran-production"
    scan_date := input.metadata.labels["security.nephoran.com/scan-date"]
    scan_date_parsed := time.parse_rfc3339_ns(scan_date)
    now := time.now_ns()
    age_days := (now - scan_date_parsed) / (24 * 60 * 60 * 1000000000)
    age_days > 7
    msg := sprintf("Pod vulnerability scan is outdated (>7 days old): %v", [scan_date])
}

# Deny images without proper signatures (future enhancement)
# deny contains msg if {
#     input.kind == "Pod"
#     input.metadata.namespace == "nephoran-production"
#     some container in input.spec.containers
#     image := container.image
#     not image_is_signed(image)
#     msg := sprintf("Container '%v' image '%v' must be signed", [container.name, image])
# }

# Require specific image digest for critical components
critical_components := {
    "nephoran-operator",
    "llm-processor",
    "nephio-bridge"
}

deny contains msg if {
    input.kind == "Pod"
    input.metadata.namespace == "nephoran-production"
    component := input.metadata.labels["app.kubernetes.io/name"]
    component in critical_components
    some container in input.spec.containers
    image := container.image
    not contains(image, "@sha256:")
    msg := sprintf("Critical component '%v' must use image digest, not tag", [component])
}

# Telecommunications-specific image requirements
telecom_namespaces := {
    "nephoran-production",
    "nephoran-5g",
    "nephoran-oran"
}

# Require SBOM for telecommunications components
deny contains msg if {
    input.kind == "Pod"
    input.metadata.namespace in telecom_namespaces
    not input.metadata.labels["security.nephoran.com/sbom-available"]
    msg := "Telecommunications components must have SBOM available"
}

# Require FIPS compliance for crypto operations
deny contains msg if {
    input.kind == "Pod"
    input.metadata.namespace in telecom_namespaces
    input.metadata.labels["security.nephoran.com/crypto-operations"] == "true"
    not input.metadata.labels["security.nephoran.com/fips-compliant"]
    msg := "Components with cryptographic operations must be FIPS compliant"
}