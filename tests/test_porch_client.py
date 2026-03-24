"""Tests for llm_nephio_oran.porch.client (ADR-003, ADR-0002, ADR-0003).

Porch lifecycle: draft → proposed → published.
Uses kubernetes.client to interact with Porch CRDs via K8s API.
"""
from __future__ import annotations

import json
from unittest.mock import patch, MagicMock, PropertyMock

import pytest


class TestPorchClient:
    """Test the PorchClient class methods."""

    @pytest.fixture
    def mock_k8s_api(self):
        """Mock kubernetes CustomObjectsApi."""
        mock_api = MagicMock()
        with patch("llm_nephio_oran.porch.client.client") as mock_client, \
             patch("llm_nephio_oran.porch.client.config") as mock_config:
            mock_config.load_incluster_config = MagicMock()
            mock_config.load_kube_config = MagicMock()
            mock_client.CustomObjectsApi.return_value = mock_api
            yield mock_api

    @pytest.fixture
    def porch_client(self, mock_k8s_api):
        from llm_nephio_oran.porch.client import PorchClient
        return PorchClient()

    def test_list_packages(self, porch_client, mock_k8s_api):
        mock_k8s_api.list_namespaced_custom_object.return_value = {
            "items": [
                {
                    "metadata": {"name": "pkg1"},
                    "spec": {"lifecycle": "Published", "packageName": "catalog/test", "repository": "nephoran-packages"},
                },
            ]
        }
        packages = porch_client.list_packages("nephoran-packages")
        assert len(packages) == 1
        assert packages[0]["metadata"]["name"] == "pkg1"

    def test_create_draft(self, porch_client, mock_k8s_api):
        mock_k8s_api.create_namespaced_custom_object.return_value = {
            "metadata": {"name": "nephoran-packages.instances.test-pkg.v1"},
            "spec": {"lifecycle": "Draft"},
        }
        result = porch_client.create_draft(
            repo="nephoran-packages",
            package_name="instances/test-pkg",
            workspace="v1",
        )
        assert result["spec"]["lifecycle"] == "Draft"
        # Verify the API was called with correct group/version/kind
        call_args = mock_k8s_api.create_namespaced_custom_object.call_args
        assert call_args[1]["group"] == "porch.kpt.dev"
        assert call_args[1]["version"] == "v1alpha1"
        assert call_args[1]["plural"] == "packagerevisions"

    def test_propose_package(self, porch_client, mock_k8s_api):
        """Transition from Draft → Proposed."""
        mock_k8s_api.get_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg", "resourceVersion": "12345"},
            "spec": {"lifecycle": "Draft", "packageName": "test", "repository": "repo"},
        }
        mock_k8s_api.replace_namespaced_custom_object.return_value = {
            "spec": {"lifecycle": "Proposed"},
        }
        result = porch_client.propose("test-pkg")
        assert result["spec"]["lifecycle"] == "Proposed"

    def test_approve_package(self, porch_client, mock_k8s_api):
        """Transition from Proposed → Published."""
        mock_k8s_api.get_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg", "resourceVersion": "12345"},
            "spec": {"lifecycle": "Proposed", "packageName": "test", "repository": "repo"},
        }
        mock_k8s_api.replace_namespaced_custom_object.return_value = {
            "spec": {"lifecycle": "Published"},
        }
        result = porch_client.approve("test-pkg")
        assert result["spec"]["lifecycle"] == "Published"

    def test_full_lifecycle_flow(self, porch_client, mock_k8s_api):
        """Draft → Proposed → Published full cycle."""
        # Create draft
        mock_k8s_api.create_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg"},
            "spec": {"lifecycle": "Draft"},
        }
        draft = porch_client.create_draft("repo", "pkg", "v1")
        assert draft["spec"]["lifecycle"] == "Draft"

        # Propose
        mock_k8s_api.get_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg", "resourceVersion": "1"},
            "spec": {"lifecycle": "Draft"},
        }
        mock_k8s_api.replace_namespaced_custom_object.return_value = {
            "spec": {"lifecycle": "Proposed"},
        }
        proposed = porch_client.propose("test-pkg")
        assert proposed["spec"]["lifecycle"] == "Proposed"

        # Approve
        mock_k8s_api.get_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg", "resourceVersion": "2"},
            "spec": {"lifecycle": "Proposed"},
        }
        mock_k8s_api.replace_namespaced_custom_object.return_value = {
            "spec": {"lifecycle": "Published"},
        }
        published = porch_client.approve("test-pkg")
        assert published["spec"]["lifecycle"] == "Published"

    def test_update_resources(self, porch_client, mock_k8s_api):
        """Push KRM content into a draft PackageRevision."""
        mock_k8s_api.get_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg", "resourceVersion": "1"},
            "spec": {"resources": {}},
        }
        mock_k8s_api.replace_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg"},
            "spec": {"resources": {"deployment.yaml": "apiVersion: apps/v1"}},
        }
        resources = {"deployment.yaml": "apiVersion: apps/v1\nkind: Deployment"}
        result = porch_client.update_resources("test-pkg", resources)
        assert "deployment.yaml" in result["spec"]["resources"]

    def test_update_resources_preserves_kptfile(self, porch_client, mock_k8s_api):
        """update_resources merges new resources with existing ones (e.g. Kptfile)."""
        existing_kptfile = "apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: test\n"
        mock_k8s_api.get_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg", "resourceVersion": "1"},
            "spec": {"resources": {"Kptfile": existing_kptfile}},
        }
        mock_k8s_api.replace_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg"},
            "spec": {"resources": {"Kptfile": existing_kptfile, "configmap.yaml": "data: test"}},
        }

        porch_client.update_resources("test-pkg", {"configmap.yaml": "data: test"})

        # Verify the body sent to replace includes both Kptfile and new resource
        call_args = mock_k8s_api.replace_namespaced_custom_object.call_args
        body = call_args[1]["body"]
        assert "Kptfile" in body["spec"]["resources"], "Kptfile should be preserved"
        assert "configmap.yaml" in body["spec"]["resources"], "New resource should be added"

    def test_get_package(self, porch_client, mock_k8s_api):
        mock_k8s_api.get_namespaced_custom_object.return_value = {
            "metadata": {"name": "test-pkg"},
            "spec": {"lifecycle": "Published"},
        }
        result = porch_client.get_package("test-pkg")
        assert result["spec"]["lifecycle"] == "Published"
