"""Tests for OPA Rego policy equivalence (T12, ADR-008, ADR-006).

Verifies that manifest_validator.py enforces the same rules defined in Rego:
  - security_guardrails.rego: denyClusterScoped, denyPrivileged, denyHostNetwork
  - naming_convention.rego: {domain}-{component}-{site}-{slice}-{instance}
  - required_labels.rego: oran.ai/intent-id, oran.ai/component, etc.
  - resource_limits.rego: containers must have resource limits

Also verifies the Rego files exist and have correct package declarations.
"""
from __future__ import annotations

import re
from pathlib import Path

import pytest

from llm_nephio_oran.validators.manifest_validator import (
    CLUSTER_SCOPED_KINDS,
    NAMING_RE,
    REQUIRED_LABELS,
    validate_package,
)

REGO_DIR = Path(__file__).parent.parent / "src" / "config-validator" / "opa_policies"


class TestRegoFilesExist:
    def test_naming_convention_exists(self):
        assert (REGO_DIR / "naming_convention.rego").exists()

    def test_security_guardrails_exists(self):
        assert (REGO_DIR / "security_guardrails.rego").exists()

    def test_required_labels_exists(self):
        assert (REGO_DIR / "required_labels.rego").exists()

    def test_resource_limits_exists(self):
        assert (REGO_DIR / "resource_limits.rego").exists()

    def test_rego_packages_correct(self):
        expected = {
            "naming_convention.rego": "nephoran.naming",
            "security_guardrails.rego": "nephoran.security",
            "required_labels.rego": "nephoran.labels",
            "resource_limits.rego": "nephoran.resources",
        }
        for filename, pkg in expected.items():
            content = (REGO_DIR / filename).read_text()
            assert f"package {pkg}" in content, f"{filename} missing package {pkg}"


class TestSecurityGuardrails:
    """Verify Python manifest_validator matches security_guardrails.rego."""

    def _pkg(self, tmp_path, yaml_content):
        f = tmp_path / "test.yaml"
        f.write_text(yaml_content)
        return tmp_path

    def test_deny_cluster_role(self, tmp_path):
        errors = validate_package(self._pkg(tmp_path, """
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-role
"""))
        assert any("cluster-scoped" in e or "L3-Policy" in e for e in errors)

    def test_deny_crd(self, tmp_path):
        errors = validate_package(self._pkg(tmp_path, """
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test.example.com
"""))
        assert any("cluster-scoped" in e for e in errors)

    def test_deny_privileged(self, tmp_path):
        errors = validate_package(self._pkg(tmp_path, """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ran-oai-odu-edge01-embb-i001
  labels:
    oran.ai/intent-id: "intent-20260306-0001"
    oran.ai/component: "oai-odu"
    app.kubernetes.io/managed-by: nephio
    app.kubernetes.io/part-of: llm-nephio-oran
spec:
  template:
    spec:
      containers:
      - name: odu
        securityContext:
          privileged: true
"""))
        assert any("privileged" in e for e in errors)

    def test_deny_host_network(self, tmp_path):
        errors = validate_package(self._pkg(tmp_path, """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ran-oai-odu-edge01-embb-i001
  labels:
    oran.ai/intent-id: "intent-20260306-0001"
    oran.ai/component: "oai-odu"
    app.kubernetes.io/managed-by: nephio
    app.kubernetes.io/part-of: llm-nephio-oran
spec:
  template:
    spec:
      hostNetwork: true
      containers:
      - name: odu
"""))
        assert any("hostNetwork" in e for e in errors)

    def test_allow_valid_deployment(self, tmp_path):
        errors = validate_package(self._pkg(tmp_path, """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ran-oai-odu-edge01-embb-i001
  labels:
    oran.ai/intent-id: "intent-20260306-0001"
    oran.ai/component: "oai-odu"
    app.kubernetes.io/managed-by: nephio
    app.kubernetes.io/part-of: llm-nephio-oran
spec:
  template:
    spec:
      containers:
      - name: odu
        securityContext:
          privileged: false
"""))
        # No policy errors (no L3/L4/L5 errors)
        policy_errors = [e for e in errors if "[L3-" in e or "[L4-" in e or "[L5-" in e]
        assert policy_errors == []


class TestNamingConvention:
    """Verify Python NAMING_RE matches naming_convention.rego pattern."""

    @pytest.mark.parametrize("name", [
        "ran-oai-odu-edge01-embb-i001",
        "core-free5gc-amf-central-shared-i003",
        "ric-ric-kpimon-lab-shared-i001",
        "sim-sim-e2-edge02-urllc-i010",
        "obs-trafficgen-regional-mmtc-i005",
    ])
    def test_valid_names(self, name):
        assert NAMING_RE.match(name), f"Should match: {name}"

    @pytest.mark.parametrize("name", [
        "invalid-name",
        "oai-odu-edge01-embb-i001",  # missing domain
        "ran-unknown-edge01-embb-i001",  # unknown component
        "ran-oai-odu-unknown-embb-i001",  # unknown site
        "ran-oai-odu-edge01-5g-i001",  # unknown slice
        "ran-oai-odu-edge01-embb-001",  # missing 'i' prefix
    ])
    def test_invalid_names(self, name):
        assert not NAMING_RE.match(name), f"Should NOT match: {name}"

    def test_rego_pattern_matches_python(self):
        """Verify Rego pattern in naming_convention.rego matches Python NAMING_RE."""
        rego = (REGO_DIR / "naming_convention.rego").read_text()
        # Extract the pattern from Rego
        match = re.search(r'naming_pattern := `([^`]+)`', rego)
        assert match, "Could not find naming_pattern in Rego"
        rego_pattern = match.group(1)
        # Both should accept/reject the same names
        test_names = [
            ("ran-oai-odu-edge01-embb-i001", True),
            ("invalid", False),
            ("ran-unknown-edge01-embb-i001", False),
        ]
        for name, expected in test_names:
            py_result = bool(NAMING_RE.match(name))
            rego_result = bool(re.match(rego_pattern, name))
            assert py_result == rego_result == expected, f"Mismatch for {name}"


class TestRequiredLabels:
    """Verify Python REQUIRED_LABELS matches required_labels.rego."""

    def test_missing_labels_detected(self, tmp_path):
        f = tmp_path / "test.yaml"
        f.write_text("""
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ran-oai-odu-edge01-embb-i001
  labels: {}
spec:
  template:
    spec:
      containers:
      - name: odu
""")
        errors = validate_package(tmp_path)
        label_errors = [e for e in errors if "[L5-Labels]" in e]
        assert len(label_errors) == len(REQUIRED_LABELS)

    def test_all_labels_present_no_errors(self, tmp_path):
        f = tmp_path / "test.yaml"
        f.write_text("""
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ran-oai-odu-edge01-embb-i001
  labels:
    oran.ai/intent-id: "intent-20260306-0001"
    oran.ai/component: "oai-odu"
    app.kubernetes.io/managed-by: nephio
    app.kubernetes.io/part-of: llm-nephio-oran
spec:
  template:
    spec:
      containers:
      - name: odu
""")
        errors = validate_package(tmp_path)
        label_errors = [e for e in errors if "[L5-Labels]" in e]
        assert label_errors == []

    def test_rego_required_labels_match_python(self):
        """Verify Rego required_labels set matches Python REQUIRED_LABELS."""
        rego = (REGO_DIR / "required_labels.rego").read_text()
        for label in REQUIRED_LABELS:
            assert label in rego, f"Label '{label}' not found in required_labels.rego"


class TestClusterScopedKinds:
    """Verify cluster-scoped kinds in Python match security_guardrails.rego."""

    def test_rego_has_all_cluster_scoped_kinds(self):
        rego = (REGO_DIR / "security_guardrails.rego").read_text()
        for kind in CLUSTER_SCOPED_KINDS:
            assert kind in rego, f"Kind '{kind}' not found in security_guardrails.rego"
