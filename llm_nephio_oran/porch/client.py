"""Porch lifecycle client — manages PackageRevision draft→proposed→published (ADR-003, ADR-0002).

Uses kubernetes.client to interact with Porch CRDs:
  - porch.kpt.dev/v1alpha1/PackageRevision — package lifecycle
  - porch.kpt.dev/v1alpha1/PackageRevisionResources — package content

Lifecycle flow:
  1. create_draft() — submit new PackageRevision in Draft state
  2. update_resources() — push KRM YAML content into draft
  3. propose() — transition Draft → Proposed (triggers validation)
  4. approve() — transition Proposed → Published (triggers ConfigSync delivery)
"""
from __future__ import annotations

import logging
import os
from typing import Any

from kubernetes import client, config

logger = logging.getLogger(__name__)

PORCH_GROUP = "porch.kpt.dev"
PORCH_VERSION = "v1alpha1"
PORCH_PR_PLURAL = "packagerevisions"
PORCH_PRR_PLURAL = "packagerevisionresources"
PORCH_NAMESPACE = os.getenv("PORCH_NAMESPACE", "default")


class PorchClient:
    """Client for Porch PackageRevision lifecycle management."""

    def __init__(self, namespace: str = PORCH_NAMESPACE):
        self.namespace = namespace
        try:
            config.load_incluster_config()
        except config.ConfigException:
            config.load_kube_config()
        self.api = client.CustomObjectsApi()

    def list_packages(self, repository: str | None = None) -> list[dict[str, Any]]:
        """List PackageRevisions, optionally filtered by repository."""
        result = self.api.list_namespaced_custom_object(
            group=PORCH_GROUP,
            version=PORCH_VERSION,
            namespace=self.namespace,
            plural=PORCH_PR_PLURAL,
        )
        items = result.get("items", [])
        if repository:
            items = [i for i in items if i.get("spec", {}).get("repository") == repository]
        return items

    def get_package(self, name: str) -> dict[str, Any]:
        """Get a single PackageRevision by name."""
        return self.api.get_namespaced_custom_object(
            group=PORCH_GROUP,
            version=PORCH_VERSION,
            namespace=self.namespace,
            plural=PORCH_PR_PLURAL,
            name=name,
        )

    def create_draft(
        self,
        repo: str,
        package_name: str,
        workspace: str = "v1",
    ) -> dict[str, Any]:
        """Create a new PackageRevision in Draft state.

        Args:
            repo: Repository name (e.g., "nephoran-packages").
            package_name: Package path (e.g., "instances/intent-20260305-0001").
            workspace: Workspace name (e.g., "v1").

        Returns:
            Created PackageRevision dict.
        """
        body = {
            "apiVersion": f"{PORCH_GROUP}/{PORCH_VERSION}",
            "kind": "PackageRevision",
            "metadata": {
                "namespace": self.namespace,
            },
            "spec": {
                "lifecycle": "Draft",
                "repository": repo,
                "packageName": package_name,
                "workspaceName": workspace,
                "tasks": [
                    {"type": "init", "init": {"description": f"Init package {package_name}"}},
                ],
            },
        }
        result = self.api.create_namespaced_custom_object(
            group=PORCH_GROUP,
            version=PORCH_VERSION,
            namespace=self.namespace,
            plural=PORCH_PR_PLURAL,
            body=body,
        )
        logger.info("Created draft PackageRevision: %s", result["metadata"]["name"])
        return result

    def update_resources(self, name: str, resources: dict[str, str]) -> dict[str, Any]:
        """Update resources in a PackageRevisionResources object.

        Args:
            name: PackageRevision name.
            resources: Dict of filename → YAML content.

        Returns:
            Updated PackageRevisionResources dict.
        """
        prr = self.api.get_namespaced_custom_object(
            group=PORCH_GROUP,
            version=PORCH_VERSION,
            namespace=self.namespace,
            plural=PORCH_PRR_PLURAL,
            name=name,
        )
        # Merge: preserve existing resources (e.g. Kptfile from init) and add/overwrite new ones
        existing = prr["spec"].get("resources") or {}
        existing.update(resources)
        prr["spec"]["resources"] = existing
        result = self.api.replace_namespaced_custom_object(
            group=PORCH_GROUP,
            version=PORCH_VERSION,
            namespace=self.namespace,
            plural=PORCH_PRR_PLURAL,
            name=name,
            body=prr,
        )
        logger.info("Updated resources for %s: %d files", name, len(resources))
        return result

    def _transition_lifecycle(self, name: str, target_lifecycle: str) -> dict[str, Any]:
        """Generic lifecycle transition for a PackageRevision."""
        pr = self.api.get_namespaced_custom_object(
            group=PORCH_GROUP,
            version=PORCH_VERSION,
            namespace=self.namespace,
            plural=PORCH_PR_PLURAL,
            name=name,
        )
        pr["spec"]["lifecycle"] = target_lifecycle
        result = self.api.replace_namespaced_custom_object(
            group=PORCH_GROUP,
            version=PORCH_VERSION,
            namespace=self.namespace,
            plural=PORCH_PR_PLURAL,
            name=name,
            body=pr,
        )
        logger.info("Transitioned %s to %s", name, target_lifecycle)
        return result

    def propose(self, name: str) -> dict[str, Any]:
        """Transition Draft → Proposed."""
        return self._transition_lifecycle(name, "Proposed")

    def approve(self, name: str) -> dict[str, Any]:
        """Transition Proposed → Published."""
        return self._transition_lifecycle(name, "Published")
