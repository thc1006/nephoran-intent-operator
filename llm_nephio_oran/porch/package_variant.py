"""PackageVariant and PackageVariantSet generators (ADR-003 — Nephio CD).

PackageVariant: creates a single downstream package from an upstream template,
with per-site/slice label injection using standard Nephio hydration functions.

PackageVariantSet: creates multiple downstream packages across sites,
each with site-specific specialization (labels, namespace, resource limits).

CRDs:
  - config.porch.kpt.dev/v1alpha1 PackageVariant
  - config.porch.kpt.dev/v1alpha2 PackageVariantSet
"""
from __future__ import annotations

from typing import Any


def _component_from_path(upstream_path: str) -> str:
    """Extract component name from upstream package path (e.g., 'catalog/oai-odu' → 'oai-odu')."""
    return upstream_path.rsplit("/", 1)[-1]


def _make_label_injector(site: str, slice_type: str) -> dict[str, Any]:
    """Create a label injection mutator for Nephio hydration."""
    return {
        "name": "set-labels",
        "image": "gcr.io/kpt-fn/set-labels:v0.2.0",
        "configMap": {
            "oran.ai/site": site,
            "oran.ai/slice": slice_type,
            "app.kubernetes.io/managed-by": "nephio",
            "app.kubernetes.io/part-of": "llm-nephio-oran",
        },
    }


def generate_package_variant(
    upstream_pkg: str,
    upstream_path: str,
    downstream_repo: str,
    site: str,
    slice_type: str,
) -> dict[str, Any]:
    """Generate a PackageVariant CR for single-site package delivery.

    Args:
        upstream_pkg: Upstream repository name (e.g., "nephoran-packages").
        upstream_path: Package path in upstream repo (e.g., "catalog/oai-odu").
        downstream_repo: Target repo for the specialized package.
        site: Target site (edge01, edge02, regional, central, lab).
        slice_type: Slice type (embb, urllc, mmtc, shared).

    Returns:
        PackageVariant dict ready for kubectl apply or yaml.dump.
    """
    component = _component_from_path(upstream_path)

    return {
        "apiVersion": "config.porch.kpt.dev/v1alpha1",
        "kind": "PackageVariant",
        "metadata": {
            "name": f"pv-{component}-{site}-{slice_type}",
        },
        "spec": {
            "upstream": {
                "repo": upstream_pkg,
                "package": upstream_path,
                "revision": "main",
            },
            "downstream": {
                "repo": downstream_repo,
                "package": f"{component}-{site}-{slice_type}",
            },
            "injectors": [
                _make_label_injector(site, slice_type),
            ],
        },
    }


def generate_package_variant_set(
    upstream_pkg: str,
    upstream_path: str,
    sites: list[str],
    slice_type: str,
) -> dict[str, Any]:
    """Generate a PackageVariantSet CR for multi-site package delivery.

    Creates one target entry per site, each with site-specific label injection.

    Args:
        upstream_pkg: Upstream repository name.
        upstream_path: Package path in upstream repo.
        sites: List of target sites.
        slice_type: Slice type for all targets.

    Returns:
        PackageVariantSet dict ready for kubectl apply or yaml.dump.
    """
    component = _component_from_path(upstream_path)

    targets = []
    for site in sites:
        targets.append({
            "repo": f"{site}-packages",
            "package": f"{component}-{site}-{slice_type}",
            "injectors": [
                _make_label_injector(site, slice_type),
            ],
        })

    return {
        "apiVersion": "config.porch.kpt.dev/v1alpha2",
        "kind": "PackageVariantSet",
        "metadata": {
            "name": f"pvs-{component}-{slice_type}",
        },
        "spec": {
            "upstream": {
                "repo": upstream_pkg,
                "package": upstream_path,
                "revision": "main",
            },
            "targets": targets,
        },
    }
