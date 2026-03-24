"""ConfigSync client — RootSync/RepoSync management (T4, ADR-002, ADR-0002).

Manages ConfigSync resources for automated delivery:
  - Query RootSync/RepoSync sync status (commit, errors)
  - Generate RepoSync YAML for namespace-scoped delivery
  - Wait for sync completion (poll until commit matches)
  - Ensure idempotent RepoSync creation

Architecture:
  Porch Published → ConfigSync picks up from Git → applies to cluster
  This client verifies the last mile: ConfigSync has synced the target commit.
"""
from __future__ import annotations

import logging
import time
from dataclasses import dataclass, field
from typing import Any

from kubernetes import client, config

logger = logging.getLogger(__name__)

CONFIGSYNC_GROUP = "configsync.gke.io"
CONFIGSYNC_VERSION = "v1beta1"
ROOTSYNC_PLURAL = "rootsyncs"
REPOSYNC_PLURAL = "reposyncs"
CONFIGSYNC_NAMESPACE = "config-management-system"


@dataclass
class SyncStatus:
    """Parsed sync status from a RootSync or RepoSync."""
    synced: bool
    commit: str
    errors: list[str] = field(default_factory=list)


@dataclass
class RepoSyncSpec:
    """Specification for generating a RepoSync resource."""
    name: str
    namespace: str
    repo: str
    branch: str
    directory: str
    auth: str = "token"
    secret_ref: str = "git-creds"
    source_format: str = "unstructured"


def generate_reposync_yaml(spec: RepoSyncSpec) -> list[dict[str, Any]]:
    """Generate RepoSync + RoleBinding YAML dicts for a namespace.

    Returns a list of two dicts:
      [0] RoleBinding — grants reconciler access to the namespace
      [1] RepoSync — configsync.gke.io/v1beta1 resource

    The RoleBinding is required because namespace-scoped reconcilers
    need explicit permissions via a RoleBinding in the target namespace.
    """
    # Google ConfigSync SA naming convention:
    #   name == "repo-sync" → ns-reconciler-{NAMESPACE}
    #   otherwise           → ns-reconciler-{NAMESPACE}-{NAME}-{len(NAME)}
    # See: https://cloud.google.com/kubernetes-engine/enterprise/config-sync/docs/how-to/multiple-repositories
    if spec.name == "repo-sync":
        sa_name = f"ns-reconciler-{spec.namespace}"
    else:
        sa_name = f"ns-reconciler-{spec.namespace}-{spec.name}-{len(spec.name)}"

    rolebinding: dict[str, Any] = {
        "apiVersion": "rbac.authorization.k8s.io/v1",
        "kind": "RoleBinding",
        "metadata": {
            "name": f"syncs-{spec.name}",
            "namespace": spec.namespace,
        },
        "roleRef": {
            "apiGroup": "rbac.authorization.k8s.io",
            "kind": "ClusterRole",
            "name": "edit",
        },
        "subjects": [
            {
                "kind": "ServiceAccount",
                "name": sa_name,
                "namespace": CONFIGSYNC_NAMESPACE,
            },
        ],
    }

    reposync: dict[str, Any] = {
        "apiVersion": f"{CONFIGSYNC_GROUP}/{CONFIGSYNC_VERSION}",
        "kind": "RepoSync",
        "metadata": {
            "name": spec.name,
            "namespace": spec.namespace,
        },
        "spec": {
            "sourceFormat": spec.source_format,
            "git": {
                "repo": spec.repo,
                "branch": spec.branch,
                "dir": spec.directory,
                "auth": spec.auth,
                "secretRef": {
                    "name": spec.secret_ref,
                },
            },
        },
    }

    return [rolebinding, reposync]


class ConfigSyncClient:
    """Client for querying and managing ConfigSync RootSync/RepoSync resources."""

    def __init__(self):
        try:
            config.load_incluster_config()
        except config.ConfigException:
            config.load_kube_config()
        self.api = client.CustomObjectsApi()

    def get_rootsync_status(
        self,
        name: str,
        namespace: str = CONFIGSYNC_NAMESPACE,
    ) -> SyncStatus:
        """Get the sync status of a RootSync resource."""
        obj = self.api.get_namespaced_custom_object(
            group=CONFIGSYNC_GROUP,
            version=CONFIGSYNC_VERSION,
            namespace=namespace,
            plural=ROOTSYNC_PLURAL,
            name=name,
        )
        return self._parse_sync_status(obj)

    def get_reposync_status(self, name: str, namespace: str) -> SyncStatus:
        """Get the sync status of a RepoSync resource."""
        obj = self.api.get_namespaced_custom_object(
            group=CONFIGSYNC_GROUP,
            version=CONFIGSYNC_VERSION,
            namespace=namespace,
            plural=REPOSYNC_PLURAL,
            name=name,
        )
        return self._parse_sync_status(obj)

    def ensure_reposync(self, spec: RepoSyncSpec) -> dict[str, Any]:
        """Create a RepoSync if it doesn't exist, or return existing one.

        Idempotent: if a RepoSync with the same name already exists
        in the namespace, returns it without modification.
        """
        existing = self.api.list_namespaced_custom_object(
            group=CONFIGSYNC_GROUP,
            version=CONFIGSYNC_VERSION,
            namespace=spec.namespace,
            plural=REPOSYNC_PLURAL,
        )
        for item in existing.get("items", []):
            if item["metadata"]["name"] == spec.name:
                logger.info("RepoSync %s/%s already exists", spec.namespace, spec.name)
                return item

        docs = generate_reposync_yaml(spec)
        rolebinding_body = docs[0]  # RoleBinding for namespace reconciler
        reposync_body = docs[1]     # The RepoSync resource

        # Create RoleBinding first (reconciler needs permissions)
        try:
            rbac_api = client.RbacAuthorizationV1Api()
            rbac_api.create_namespaced_role_binding(
                namespace=spec.namespace,
                body=rolebinding_body,
            )
            logger.info("Created RoleBinding %s/%s", spec.namespace, rolebinding_body["metadata"]["name"])
        except client.ApiException as e:
            if e.status == 409:
                logger.info("RoleBinding %s/%s already exists", spec.namespace, rolebinding_body["metadata"]["name"])
            else:
                raise

        result = self.api.create_namespaced_custom_object(
            group=CONFIGSYNC_GROUP,
            version=CONFIGSYNC_VERSION,
            namespace=spec.namespace,
            plural=REPOSYNC_PLURAL,
            body=reposync_body,
        )
        logger.info("Created RepoSync %s/%s", spec.namespace, spec.name)
        return result

    def delete_reposync(self, name: str, namespace: str) -> bool:
        """Delete a RepoSync and its associated RoleBinding.

        Returns True if deleted, False if not found.
        """
        deleted = False

        # Delete RepoSync
        try:
            self.api.delete_namespaced_custom_object(
                group=CONFIGSYNC_GROUP,
                version=CONFIGSYNC_VERSION,
                namespace=namespace,
                plural=REPOSYNC_PLURAL,
                name=name,
            )
            logger.info("Deleted RepoSync %s/%s", namespace, name)
            deleted = True
        except client.ApiException as e:
            if e.status == 404:
                logger.debug("RepoSync %s/%s not found (already deleted)", namespace, name)
            else:
                raise

        # Delete associated RoleBinding
        rb_name = f"syncs-{name}"
        try:
            rbac_api = client.RbacAuthorizationV1Api()
            rbac_api.delete_namespaced_role_binding(
                name=rb_name,
                namespace=namespace,
            )
            logger.info("Deleted RoleBinding %s/%s", namespace, rb_name)
        except client.ApiException as e:
            if e.status == 404:
                logger.debug("RoleBinding %s/%s not found", namespace, rb_name)
            else:
                raise

        return deleted

    def wait_for_sync(
        self,
        name: str,
        target_commit: str,
        namespace: str = CONFIGSYNC_NAMESPACE,
        timeout: float = 120,
        poll_interval: float = 5,
        sync_type: str = "rootsync",
    ) -> bool:
        """Poll until the sync commit matches target_commit, or timeout.

        Args:
            name: RootSync or RepoSync name.
            target_commit: Expected commit SHA to be synced.
            namespace: Namespace of the sync resource.
            timeout: Max seconds to wait.
            poll_interval: Seconds between polls.
            sync_type: "rootsync" or "reposync".

        Returns:
            True if sync completed with target commit, False on timeout.
        """
        deadline = time.monotonic() + timeout
        plural = ROOTSYNC_PLURAL if sync_type == "rootsync" else REPOSYNC_PLURAL

        while time.monotonic() < deadline:
            obj = self.api.get_namespaced_custom_object(
                group=CONFIGSYNC_GROUP,
                version=CONFIGSYNC_VERSION,
                namespace=namespace,
                plural=plural,
                name=name,
            )
            status = self._parse_sync_status(obj)
            if status.synced and status.commit == target_commit:
                logger.info("Sync %s/%s reached commit %s", namespace, name, target_commit[:8])
                return True
            if status.errors:
                logger.warning("Sync %s/%s has errors: %s", namespace, name, status.errors)
            time.sleep(poll_interval)

        logger.warning("Sync %s/%s timed out waiting for %s", namespace, name, target_commit[:8])
        return False

    @staticmethod
    def _parse_sync_status(obj: dict[str, Any]) -> SyncStatus:
        """Parse status from a RootSync or RepoSync object."""
        status = obj.get("status", {})
        sync_status = status.get("sync", {})
        source_status = status.get("source", {})
        conditions = status.get("conditions", [])

        sync_commit = sync_status.get("commit", "")
        source_commit = source_status.get("commit", "")

        # Collect errors from source and sync
        errors: list[str] = []
        for section in (source_status, sync_status):
            error_summary = section.get("errorSummary", {})
            if error_summary.get("totalCount", 0) > 0:
                errors.append(f"errorCount={error_summary['totalCount']}")
            for err in section.get("errors", []):
                msg = err.get("errorMessage", str(err))
                errors.append(msg)

        # Synced when: sync commit == source commit AND no Syncing condition active
        syncing = False
        for cond in conditions:
            if cond.get("type") == "Syncing" and cond.get("status") == "True":
                syncing = True

        synced = (
            sync_commit == source_commit
            and sync_commit != ""
            and not syncing
            and len(errors) == 0
        )

        return SyncStatus(synced=synced, commit=sync_commit, errors=errors)
