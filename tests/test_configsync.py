"""Tests for ConfigSync client — RootSync/RepoSync management (T4, ADR-002, ADR-0002).

Verifies:
  - RootSync status query (sync commit, errors)
  - RepoSync generation for namespace-scoped delivery
  - RepoSync YAML structure matches configsync.gke.io/v1beta1
  - Sync-wait logic (poll until synced or timeout)
  - Integration: pipeline → Porch Published → ConfigSync sync check
"""
from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest

from llm_nephio_oran.configsync.client import (
    ConfigSyncClient,
    RepoSyncSpec,
    SyncStatus,
    generate_reposync_yaml,
)


class TestSyncStatus:
    def test_synced_status(self):
        s = SyncStatus(synced=True, commit="abc123", errors=[])
        assert s.synced is True
        assert s.commit == "abc123"
        assert s.errors == []

    def test_error_status(self):
        s = SyncStatus(synced=False, commit="", errors=["source error"])
        assert s.synced is False
        assert len(s.errors) == 1


class TestRepoSyncSpec:
    def test_defaults(self):
        spec = RepoSyncSpec(
            name="intent-sync",
            namespace="oran",
            repo="http://gitea-http.gitea.svc.cluster.local:3000/nephio/nephoran-packages",
            branch="main",
            directory="packages/instances/intent-20260306-0001",
        )
        assert spec.name == "intent-sync"
        assert spec.namespace == "oran"
        assert spec.auth == "token"
        assert spec.secret_ref == "git-creds"


class TestGenerateRepoSyncYaml:
    def test_basic_structure(self):
        spec = RepoSyncSpec(
            name="intent-0001",
            namespace="oran",
            repo="http://gitea/nephio/nephoran-packages",
            branch="main",
            directory="packages/instances/intent-20260306-0001",
        )
        docs = generate_reposync_yaml(spec)
        assert len(docs) == 2  # Namespace binding + RepoSync

        reposync = docs[1]
        assert reposync["apiVersion"] == "configsync.gke.io/v1beta1"
        assert reposync["kind"] == "RepoSync"
        assert reposync["metadata"]["name"] == "intent-0001"
        assert reposync["metadata"]["namespace"] == "oran"
        assert reposync["spec"]["sourceFormat"] == "unstructured"

    def test_git_section(self):
        spec = RepoSyncSpec(
            name="test-sync",
            namespace="free5gc",
            repo="http://gitea/nephio/nephoran-packages",
            branch="main",
            directory="packages/instances/test",
        )
        docs = generate_reposync_yaml(spec)
        reposync = docs[1]
        git = reposync["spec"]["git"]
        assert git["repo"] == "http://gitea/nephio/nephoran-packages"
        assert git["branch"] == "main"
        assert git["dir"] == "packages/instances/test"
        assert git["auth"] == "token"
        assert git["secretRef"]["name"] == "git-creds"

    def test_rolebinding_for_namespace(self):
        spec = RepoSyncSpec(
            name="intent-sync",
            namespace="oran",
            repo="http://gitea/repo",
            branch="main",
            directory="packages/instances/x",
        )
        docs = generate_reposync_yaml(spec)
        rb = docs[0]
        assert rb["kind"] == "RoleBinding"
        assert rb["metadata"]["namespace"] == "oran"
        assert rb["roleRef"]["name"] == "edit"
        assert any(s["name"].startswith("ns-reconciler-oran") for s in rb["subjects"])

    def test_custom_auth(self):
        spec = RepoSyncSpec(
            name="s",
            namespace="ns",
            repo="http://repo",
            branch="main",
            directory="d",
            auth="ssh",
            secret_ref="ssh-key",
        )
        docs = generate_reposync_yaml(spec)
        git = docs[1]["spec"]["git"]
        assert git["auth"] == "ssh"
        assert git["secretRef"]["name"] == "ssh-key"


class TestConfigSyncClient:
    @pytest.fixture
    def mock_api(self):
        with patch("llm_nephio_oran.configsync.client.client") as mock_client, \
             patch("llm_nephio_oran.configsync.client.config"):
            mock_custom = MagicMock()
            mock_rbac = MagicMock()
            mock_client.CustomObjectsApi.return_value = mock_custom
            mock_client.RbacAuthorizationV1Api.return_value = mock_rbac
            mock_client.ApiException = type("ApiException", (Exception,), {"status": 409})
            mock_custom._rbac = mock_rbac  # expose for assertions
            yield mock_custom

    def test_get_rootsync_status_synced(self, mock_api):
        mock_api.get_namespaced_custom_object.return_value = {
            "status": {
                "sync": {
                    "commit": "abc123",
                    "errorSummary": {},
                },
                "source": {
                    "commit": "abc123",
                    "errorSummary": {},
                },
                "conditions": [
                    {"type": "Syncing", "status": "False", "message": "Sync Completed"},
                ],
            },
        }
        c = ConfigSyncClient()
        status = c.get_rootsync_status("nephoran-packages")
        assert status.synced is True
        assert status.commit == "abc123"
        assert status.errors == []

    def test_get_rootsync_status_with_errors(self, mock_api):
        mock_api.get_namespaced_custom_object.return_value = {
            "status": {
                "sync": {
                    "commit": "def456",
                    "errorSummary": {"totalCount": 1},
                },
                "source": {
                    "commit": "def456",
                    "errorSummary": {},
                    "errors": [{"errorMessage": "parse error"}],
                },
                "conditions": [
                    {"type": "Syncing", "status": "True", "message": "Syncing"},
                ],
            },
        }
        c = ConfigSyncClient()
        status = c.get_rootsync_status("nephoran-packages")
        assert status.synced is False
        assert len(status.errors) >= 1

    def test_create_reposync(self, mock_api):
        mock_api.create_namespaced_custom_object.return_value = {"metadata": {"name": "intent-sync"}}
        mock_api.list_namespaced_custom_object.return_value = {"items": []}

        c = ConfigSyncClient()
        spec = RepoSyncSpec(
            name="intent-sync",
            namespace="oran",
            repo="http://gitea/repo",
            branch="main",
            directory="d",
        )
        result = c.ensure_reposync(spec)
        assert result["metadata"]["name"] == "intent-sync"
        assert mock_api.create_namespaced_custom_object.called
        # Verify RoleBinding was also created
        assert mock_api._rbac.create_namespaced_role_binding.called
        rb_call = mock_api._rbac.create_namespaced_role_binding.call_args
        assert rb_call.kwargs["namespace"] == "oran"

    def test_ensure_reposync_already_exists(self, mock_api):
        mock_api.list_namespaced_custom_object.return_value = {
            "items": [{"metadata": {"name": "intent-sync"}}],
        }
        c = ConfigSyncClient()
        spec = RepoSyncSpec(
            name="intent-sync",
            namespace="oran",
            repo="http://gitea/repo",
            branch="main",
            directory="d",
        )
        result = c.ensure_reposync(spec)
        assert result["metadata"]["name"] == "intent-sync"
        assert not mock_api.create_namespaced_custom_object.called

    def test_wait_for_sync_success(self, mock_api):
        """wait_for_sync returns True when commit matches."""
        call_count = {"n": 0}

        def side_effect(*args, **kwargs):
            call_count["n"] += 1
            if call_count["n"] < 3:
                return {
                    "status": {
                        "sync": {"commit": "old", "errorSummary": {}},
                        "source": {"commit": "new123", "errorSummary": {}},
                        "conditions": [{"type": "Syncing", "status": "True"}],
                    }
                }
            return {
                "status": {
                    "sync": {"commit": "new123", "errorSummary": {}},
                    "source": {"commit": "new123", "errorSummary": {}},
                    "conditions": [{"type": "Syncing", "status": "False", "message": "Sync Completed"}],
                }
            }

        mock_api.get_namespaced_custom_object.side_effect = side_effect
        c = ConfigSyncClient()
        synced = c.wait_for_sync("nephoran-packages", "new123", timeout=5, poll_interval=0.1)
        assert synced is True

    def test_wait_for_sync_timeout(self, mock_api):
        """wait_for_sync returns False on timeout."""
        mock_api.get_namespaced_custom_object.return_value = {
            "status": {
                "sync": {"commit": "old", "errorSummary": {}},
                "source": {"commit": "old", "errorSummary": {}},
                "conditions": [{"type": "Syncing", "status": "True"}],
            }
        }
        c = ConfigSyncClient()
        synced = c.wait_for_sync("nephoran-packages", "new123", timeout=0.3, poll_interval=0.1)
        assert synced is False


class TestD1bDeleteRepoSync:
    """D1b: Per-intent RepoSync lifecycle — deletion tests."""

    @pytest.fixture
    def mock_api(self):
        with patch("llm_nephio_oran.configsync.client.client") as mock_client, \
             patch("llm_nephio_oran.configsync.client.config"):
            mock_custom = MagicMock()
            mock_rbac = MagicMock()
            mock_client.CustomObjectsApi.return_value = mock_custom
            mock_client.RbacAuthorizationV1Api.return_value = mock_rbac
            mock_client.ApiException = type("ApiException", (Exception,), {"status": 404})
            mock_custom._rbac = mock_rbac
            yield mock_custom

    def test_delete_reposync_success(self, mock_api):
        """delete_reposync removes both RepoSync and RoleBinding."""
        c = ConfigSyncClient()
        result = c.delete_reposync("intent-20260306-0001", "oran")
        assert result is True
        mock_api.delete_namespaced_custom_object.assert_called_once()
        call_kwargs = mock_api.delete_namespaced_custom_object.call_args.kwargs
        assert call_kwargs["name"] == "intent-20260306-0001"
        assert call_kwargs["namespace"] == "oran"
        # RoleBinding also deleted
        mock_api._rbac.delete_namespaced_role_binding.assert_called_once()
        rb_kwargs = mock_api._rbac.delete_namespaced_role_binding.call_args.kwargs
        assert rb_kwargs["name"] == "syncs-intent-20260306-0001"
        assert rb_kwargs["namespace"] == "oran"

    def test_delete_reposync_not_found(self, mock_api):
        """delete_reposync returns False when RepoSync does not exist."""
        from llm_nephio_oran.configsync.client import client as k8s_client
        exc = k8s_client.ApiException()
        exc.status = 404
        mock_api.delete_namespaced_custom_object.side_effect = exc
        c = ConfigSyncClient()
        result = c.delete_reposync("nonexistent", "oran")
        assert result is False

    def test_delete_reposync_rolebinding_not_found(self, mock_api):
        """delete_reposync succeeds even if RoleBinding is already gone."""
        from llm_nephio_oran.configsync.client import client as k8s_client
        exc = k8s_client.ApiException()
        exc.status = 404
        mock_api._rbac.delete_namespaced_role_binding.side_effect = exc
        c = ConfigSyncClient()
        result = c.delete_reposync("intent-0001", "oran")
        assert result is True  # RepoSync was deleted, RoleBinding 404 is fine


class TestD1bRepoSyncNaming:
    """D1b: RepoSync naming follows ConfigSync SA convention."""

    def test_per_intent_reposync_name(self):
        """RepoSync name is intent-{intentId}."""
        spec = RepoSyncSpec(
            name="intent-intent-20260306-0001",
            namespace="oran",
            repo="http://gitea/repo",
            branch="main",
            directory="packages/instances/intent-20260306-0001",
        )
        docs = generate_reposync_yaml(spec)
        reposync = docs[1]
        assert reposync["metadata"]["name"] == "intent-intent-20260306-0001"
        # SA name follows custom naming (not "repo-sync")
        rb = docs[0]
        sa_name = rb["subjects"][0]["name"]
        assert sa_name.startswith("ns-reconciler-oran-")
        assert "intent-intent-20260306-0001" in sa_name
