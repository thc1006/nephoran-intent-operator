"""Tests for llm_nephio_oran.porch.package_variant (ADR-003 — Nephio CD).

PackageVariant: single-site delivery from upstream → downstream repo.
PackageVariantSet: multi-site delivery with per-site specialization.
"""
from __future__ import annotations

import yaml
import pytest

from llm_nephio_oran.porch.package_variant import (
    generate_package_variant,
    generate_package_variant_set,
)


class TestGeneratePackageVariant:
    """PackageVariant: upstream → single downstream repo with specialization."""

    def test_basic_structure(self):
        pv = generate_package_variant(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/oai-odu",
            downstream_repo="edge01-packages",
            site="edge01",
            slice_type="embb",
        )
        assert pv["apiVersion"] == "config.porch.kpt.dev/v1alpha1"
        assert pv["kind"] == "PackageVariant"

    def test_upstream_reference(self):
        pv = generate_package_variant(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/oai-odu",
            downstream_repo="edge01-packages",
            site="edge01",
            slice_type="embb",
        )
        assert pv["spec"]["upstream"]["repo"] == "nephoran-packages"
        assert pv["spec"]["upstream"]["package"] == "catalog/oai-odu"

    def test_downstream_reference(self):
        pv = generate_package_variant(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/oai-odu",
            downstream_repo="edge01-packages",
            site="edge01",
            slice_type="embb",
        )
        assert pv["spec"]["downstream"]["repo"] == "edge01-packages"

    def test_labels_injection(self):
        """ADR-006/ADR-0007: site/slice labels injected."""
        pv = generate_package_variant(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/oai-odu",
            downstream_repo="edge01-packages",
            site="edge01",
            slice_type="urllc",
        )
        injections = pv["spec"].get("injectors", [])
        label_injector = next((i for i in injections if i.get("name") == "set-labels"), None)
        assert label_injector is not None
        assert label_injector["configMap"]["oran.ai/site"] == "edge01"
        assert label_injector["configMap"]["oran.ai/slice"] == "urllc"

    def test_naming_convention(self):
        """Name should follow pattern: pv-{component}-{site}-{slice}."""
        pv = generate_package_variant(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/oai-odu",
            downstream_repo="edge01-packages",
            site="edge01",
            slice_type="embb",
        )
        assert pv["metadata"]["name"] == "pv-oai-odu-edge01-embb"

    def test_valid_yaml_output(self):
        pv = generate_package_variant(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/free5gc-amf",
            downstream_repo="central-packages",
            site="central",
            slice_type="shared",
        )
        yaml_str = yaml.dump(pv, default_flow_style=False)
        parsed = yaml.safe_load(yaml_str)
        assert parsed["kind"] == "PackageVariant"


class TestGeneratePackageVariantSet:
    """PackageVariantSet: multi-site deployment with per-site specialization."""

    def test_basic_structure(self):
        pvs = generate_package_variant_set(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/oai-odu",
            sites=["edge01", "edge02"],
            slice_type="embb",
        )
        assert pvs["apiVersion"] == "config.porch.kpt.dev/v1alpha2"
        assert pvs["kind"] == "PackageVariantSet"

    def test_multi_site_targets(self):
        pvs = generate_package_variant_set(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/oai-odu",
            sites=["edge01", "edge02", "regional"],
            slice_type="embb",
        )
        targets = pvs["spec"]["targets"]
        assert len(targets) == 3
        repos = [t["repo"] for t in targets]
        assert "edge01-packages" in repos
        assert "edge02-packages" in repos
        assert "regional-packages" in repos

    def test_per_site_labels(self):
        pvs = generate_package_variant_set(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/free5gc-upf",
            sites=["edge01", "edge02"],
            slice_type="urllc",
        )
        for target in pvs["spec"]["targets"]:
            injections = target.get("injectors", [])
            label_injector = next((i for i in injections if i.get("name") == "set-labels"), None)
            assert label_injector is not None
            assert label_injector["configMap"]["oran.ai/slice"] == "urllc"

    def test_naming(self):
        pvs = generate_package_variant_set(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/ric-kpimon",
            sites=["edge01"],
            slice_type="embb",
        )
        assert pvs["metadata"]["name"] == "pvs-ric-kpimon-embb"

    def test_valid_yaml_output(self):
        pvs = generate_package_variant_set(
            upstream_pkg="nephoran-packages",
            upstream_path="catalog/oai-odu",
            sites=["edge01"],
            slice_type="embb",
        )
        yaml_str = yaml.dump(pvs, default_flow_style=False)
        parsed = yaml.safe_load(yaml_str)
        assert parsed["kind"] == "PackageVariantSet"
