"""Tests for llm_nephio_oran.generator.kpt_generator (ADR-003, ADR-006, ADR-0003, ADR-0007)."""
from __future__ import annotations

from pathlib import Path

import yaml
import pytest

from llm_nephio_oran.generator.kpt_generator import (
    generate_kpt_packages,
    _resource_name,
    _labels,
    COMPONENT_DEFAULTS,
    NAMESPACE_MAP,
)


class TestResourceName:
    """ADR-006 / ADR-0007: naming convention {domain}-{component}-{site}-{slice}-{instance}."""

    def test_basic_naming(self):
        action = {
            "component": "oai-odu",
            "naming": {"domain": "ran", "site": "edge01", "slice": "urllc", "instance": "i001"},
        }
        assert _resource_name(action) == "ran-oai-odu-edge01-urllc-i001"

    def test_defaults_when_naming_missing(self):
        action = {"component": "free5gc-amf"}
        name = _resource_name(action)
        assert name.startswith("core-free5gc-amf-")
        assert name.endswith("-i001")

    def test_all_11_components_have_domain(self):
        for comp, defaults in COMPONENT_DEFAULTS.items():
            assert "domain" in defaults, f"{comp} missing domain in COMPONENT_DEFAULTS"
            action = {"component": comp}
            name = _resource_name(action)
            assert name.startswith(defaults["domain"] + "-" + comp)


class TestLabels:
    """CLAUDE.md §5: 7 required labels."""

    def test_all_required_labels_present(self):
        labels = _labels("intent-20260305-0001", {
            "component": "ric-kpimon",
            "naming": {"slice": "embb", "site": "lab"},
        })
        required = [
            "oran.ai/intent-id", "oran.ai/slice", "oran.ai/site", "oran.ai/component",
            "app.kubernetes.io/managed-by", "app.kubernetes.io/part-of", "app.kubernetes.io/name",
        ]
        for lbl in required:
            assert lbl in labels, f"Missing label: {lbl}"

    def test_intent_id_propagated(self):
        labels = _labels("intent-20260305-9999", {"component": "sim-e2"})
        assert labels["oran.ai/intent-id"] == "intent-20260305-9999"

    def test_managed_by_nephio(self):
        labels = _labels("intent-20260305-0001", {"component": "trafficgen"})
        assert labels["app.kubernetes.io/managed-by"] == "nephio"
        assert labels["app.kubernetes.io/part-of"] == "llm-nephio-oran"


class TestGenerateKptPackages:
    """Full package generation tests."""

    def test_generates_files_for_deploy(self, valid_plan, tmp_pkg_dir):
        pkg_dir = generate_kpt_packages(valid_plan, tmp_pkg_dir)
        assert pkg_dir.exists()
        assert (pkg_dir / "intent-plan.json").exists()
        assert (pkg_dir / "Kptfile").exists()

    def test_generates_per_action_subdirs(self, valid_plan, tmp_pkg_dir):
        valid_plan["actions"] = [
            {"kind": "deploy", "component": "oai-odu", "replicas": 1,
             "naming": {"domain": "ran", "site": "edge01", "slice": "embb", "instance": "i001"}},
            {"kind": "deploy", "component": "free5gc-amf", "replicas": 1,
             "naming": {"domain": "core", "site": "edge01", "slice": "embb", "instance": "i001"}},
        ]
        pkg_dir = generate_kpt_packages(valid_plan, tmp_pkg_dir)
        subdirs = [d.name for d in pkg_dir.iterdir() if d.is_dir()]
        assert "ran-oai-odu-edge01-embb-i001" in subdirs
        assert "core-free5gc-amf-edge01-embb-i001" in subdirs

    def test_deployment_yaml_valid(self, valid_plan, tmp_pkg_dir):
        pkg_dir = generate_kpt_packages(valid_plan, tmp_pkg_dir)
        action_name = "ric-ric-kpimon-lab-embb-i001"
        deploy_yaml = pkg_dir / action_name / "deployment.yaml"
        assert deploy_yaml.exists()
        doc = yaml.safe_load(deploy_yaml.read_text())
        assert doc["apiVersion"] == "apps/v1"
        assert doc["kind"] == "Deployment"
        assert doc["metadata"]["name"] == action_name

    def test_deployment_security_context(self, valid_plan, tmp_pkg_dir):
        """ADR-008: runAsNonRoot, seccompProfile, drop ALL caps."""
        pkg_dir = generate_kpt_packages(valid_plan, tmp_pkg_dir)
        for deploy_yaml in pkg_dir.rglob("deployment.yaml"):
            doc = yaml.safe_load(deploy_yaml.read_text())
            pod_spec = doc["spec"]["template"]["spec"]
            assert pod_spec["securityContext"]["runAsNonRoot"] is True
            assert pod_spec["securityContext"]["seccompProfile"]["type"] == "RuntimeDefault"
            container_sec = pod_spec["containers"][0]["securityContext"]
            assert container_sec["allowPrivilegeEscalation"] is False
            assert "ALL" in container_sec["capabilities"]["drop"]

    def test_namespace_mapping(self, valid_plan, tmp_pkg_dir):
        """Verify domain→namespace mapping per CLAUDE.md §5."""
        valid_plan["actions"] = [
            {"kind": "deploy", "component": "oai-odu",
             "naming": {"domain": "ran", "site": "lab", "slice": "embb", "instance": "i001"}},
        ]
        pkg_dir = generate_kpt_packages(valid_plan, tmp_pkg_dir)
        deploy_yaml = next(pkg_dir.rglob("deployment.yaml"))
        doc = yaml.safe_load(deploy_yaml.read_text())
        assert doc["metadata"]["namespace"] == "oran"  # ran → oran

    def test_replicas_honored(self, valid_plan_scale, tmp_pkg_dir):
        pkg_dir = generate_kpt_packages(valid_plan_scale, tmp_pkg_dir)
        deploy_yaml = next(pkg_dir.rglob("deployment.yaml"))
        doc = yaml.safe_load(deploy_yaml.read_text())
        assert doc["spec"]["replicas"] == 3

    def test_service_yaml_generated(self, valid_plan, tmp_pkg_dir):
        pkg_dir = generate_kpt_packages(valid_plan, tmp_pkg_dir)
        svc_files = list(pkg_dir.rglob("service.yaml"))
        assert len(svc_files) >= 1
        doc = yaml.safe_load(svc_files[0].read_text())
        assert doc["apiVersion"] == "v1"
        assert doc["kind"] == "Service"

    def test_kptfile_has_pipeline(self, valid_plan, tmp_pkg_dir):
        pkg_dir = generate_kpt_packages(valid_plan, tmp_pkg_dir)
        for kptfile in pkg_dir.rglob("Kptfile"):
            doc = yaml.safe_load(kptfile.read_text())
            assert doc["apiVersion"] == "kpt.dev/v1"
            assert doc["kind"] == "Kptfile"
            assert "pipeline" in doc

    def test_params_become_env_vars(self, tmp_pkg_dir):
        plan = {
            "intentId": "intent-20260305-0001",
            "intentType": "slice.deploy",
            "actions": [
                {"kind": "deploy", "component": "trafficgen", "replicas": 1,
                 "params": {"mode": "iperf3", "durationSeconds": 600},
                 "naming": {"domain": "sim", "site": "lab", "slice": "embb", "instance": "i001"}},
            ],
            "policy": {"requireHumanReview": True,
                       "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True}},
        }
        pkg_dir = generate_kpt_packages(plan, tmp_pkg_dir)
        deploy_yaml = next(pkg_dir.rglob("deployment.yaml"))
        doc = yaml.safe_load(deploy_yaml.read_text())
        env = doc["spec"]["template"]["spec"]["containers"][0].get("env", [])
        env_names = [e["name"] for e in env]
        assert "MODE" in env_names
        assert "DURATIONSECONDS" in env_names
