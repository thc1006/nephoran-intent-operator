"""Tests for llm_nephio_oran.validators.manifest_validator (ADR-008, 5-layer validation)."""
from __future__ import annotations

from pathlib import Path

import yaml
import pytest

from llm_nephio_oran.validators.manifest_validator import validate_package, NAMING_RE


class TestNamingRegex:
    """ADR-006: naming convention regex."""

    @pytest.mark.parametrize("name", [
        "ran-oai-odu-edge01-embb-i001",
        "core-free5gc-amf-lab-urllc-i999",
        "ric-ric-kpimon-regional-shared-i001",
        "sim-sim-e2-central-mmtc-i042",
        "sim-trafficgen-edge02-embb-i001",
    ])
    def test_valid_names(self, name):
        assert NAMING_RE.match(name), f"{name} should match"

    @pytest.mark.parametrize("name", [
        "bad-name",
        "ran-unknown-edge01-embb-i001",
        "ran-oai-odu-badsite-embb-i001",
        "ran-oai-odu-edge01-badslice-i001",
        "ran-oai-odu-edge01-embb-j001",
    ])
    def test_invalid_names(self, name):
        assert not NAMING_RE.match(name), f"{name} should NOT match"


class TestValidatePackage:
    """5-layer validation on generated packages."""

    def _write_yaml(self, path: Path, doc: dict) -> Path:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(yaml.dump(doc, default_flow_style=False))
        return path

    def test_empty_dir_returns_error(self, tmp_path):
        errors = validate_package(tmp_path)
        assert any("No YAML" in e for e in errors)

    def test_valid_deployment_passes(self, tmp_path):
        self._write_yaml(tmp_path / "deploy.yaml", {
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {
                "name": "ran-oai-odu-edge01-embb-i001",
                "namespace": "oran",
                "labels": {
                    "oran.ai/intent-id": "intent-20260305-0001",
                    "oran.ai/component": "oai-odu",
                    "app.kubernetes.io/managed-by": "nephio",
                    "app.kubernetes.io/part-of": "llm-nephio-oran",
                },
            },
            "spec": {
                "replicas": 1,
                "selector": {"matchLabels": {"app": "test"}},
                "template": {
                    "metadata": {"labels": {"app": "test"}},
                    "spec": {"containers": [{"name": "test", "image": "test:latest"}]},
                },
            },
        })
        errors = validate_package(tmp_path)
        assert errors == [], f"Expected no errors but got: {errors}"

    def test_l1_yaml_syntax_error(self, tmp_path):
        (tmp_path / "bad.yaml").write_text("{{invalid yaml")
        errors = validate_package(tmp_path)
        assert any("[L1-YAML]" in e for e in errors)

    def test_l2_missing_apiversion(self, tmp_path):
        self._write_yaml(tmp_path / "bad.yaml", {
            "kind": "Deployment",
            "metadata": {"name": "test"},
        })
        errors = validate_package(tmp_path)
        assert any("[L2-K8s]" in e and "apiVersion" in e for e in errors)

    def test_l2_missing_kind(self, tmp_path):
        self._write_yaml(tmp_path / "bad.yaml", {
            "apiVersion": "v1",
            "metadata": {"name": "test"},
        })
        errors = validate_package(tmp_path)
        assert any("[L2-K8s]" in e and "kind" in e for e in errors)

    def test_l3_cluster_scoped_denied(self, tmp_path):
        self._write_yaml(tmp_path / "cr.yaml", {
            "apiVersion": "rbac.authorization.k8s.io/v1",
            "kind": "ClusterRole",
            "metadata": {"name": "admin"},
        })
        errors = validate_package(tmp_path)
        assert any("[L3-Policy]" in e and "ClusterRole" in e for e in errors)

    def test_l3_privileged_denied(self, tmp_path):
        self._write_yaml(tmp_path / "priv.yaml", {
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {
                "name": "ran-oai-odu-edge01-embb-i001",
                "labels": {
                    "oran.ai/intent-id": "x", "oran.ai/component": "x",
                    "app.kubernetes.io/managed-by": "x", "app.kubernetes.io/part-of": "x",
                },
            },
            "spec": {
                "template": {
                    "spec": {
                        "containers": [{"name": "c", "image": "i", "securityContext": {"privileged": True}}]
                    },
                },
            },
        })
        errors = validate_package(tmp_path)
        assert any("[L3-Policy]" in e and "privileged" in e.lower() for e in errors)

    def test_l3_hostnetwork_denied(self, tmp_path):
        self._write_yaml(tmp_path / "host.yaml", {
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {
                "name": "ran-oai-odu-edge01-embb-i001",
                "labels": {
                    "oran.ai/intent-id": "x", "oran.ai/component": "x",
                    "app.kubernetes.io/managed-by": "x", "app.kubernetes.io/part-of": "x",
                },
            },
            "spec": {
                "template": {"spec": {"hostNetwork": True, "containers": [{"name": "c", "image": "i"}]}},
            },
        })
        errors = validate_package(tmp_path)
        assert any("[L3-Policy]" in e and "hostNetwork" in e for e in errors)

    def test_l4_bad_naming_detected(self, tmp_path):
        self._write_yaml(tmp_path / "bad-name.yaml", {
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {
                "name": "bad-naming-here-too",
                "labels": {
                    "oran.ai/intent-id": "x", "oran.ai/component": "x",
                    "app.kubernetes.io/managed-by": "x", "app.kubernetes.io/part-of": "x",
                },
            },
            "spec": {
                "template": {"spec": {"containers": [{"name": "c", "image": "i"}]}},
            },
        })
        errors = validate_package(tmp_path)
        assert any("[L4-Naming]" in e for e in errors)

    def test_l5_missing_labels(self, tmp_path):
        self._write_yaml(tmp_path / "no-labels.yaml", {
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {"name": "ran-oai-odu-edge01-embb-i001"},
            "spec": {
                "template": {"spec": {"containers": [{"name": "c", "image": "i"}]}},
            },
        })
        errors = validate_package(tmp_path)
        assert any("[L5-Labels]" in e for e in errors)

    def test_kptfile_skipped(self, tmp_path):
        self._write_yaml(tmp_path / "Kptfile", {
            "apiVersion": "kpt.dev/v1",
            "kind": "Kptfile",
            "metadata": {"name": "test"},
        })
        self._write_yaml(tmp_path / "deploy.yaml", {
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {
                "name": "ran-oai-odu-edge01-embb-i001",
                "namespace": "oran",
                "labels": {
                    "oran.ai/intent-id": "i", "oran.ai/component": "c",
                    "app.kubernetes.io/managed-by": "m", "app.kubernetes.io/part-of": "p",
                },
            },
            "spec": {
                "template": {"spec": {"containers": [{"name": "c", "image": "i"}]}},
            },
        })
        errors = validate_package(tmp_path)
        assert errors == []

    def test_service_validated_too(self, tmp_path):
        """Services also need naming + labels validation."""
        self._write_yaml(tmp_path / "svc.yaml", {
            "apiVersion": "v1",
            "kind": "Service",
            "metadata": {"name": "bad-svc-name-x"},
            "spec": {"ports": [{"port": 80}]},
        })
        errors = validate_package(tmp_path)
        assert any("[L5-Labels]" in e for e in errors)
