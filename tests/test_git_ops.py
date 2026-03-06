"""Tests for llm_nephio_oran.gitops.git_ops (ADR-002, ADR-0002, ADR-0004)."""
from __future__ import annotations

import json
import os
from pathlib import Path
from unittest.mock import patch, MagicMock

import pytest
from git import Repo

from llm_nephio_oran.gitops.git_ops import (
    push_to_gitops,
    _ensure_repo,
    _create_branch,
    _copy_packages,
    _commit_and_push,
    _open_pr,
)


@pytest.fixture
def mock_git_repo(tmp_path):
    """Create a real but local git repo for testing."""
    repo_dir = tmp_path / "nephoran-packages"
    repo_dir.mkdir()
    repo = Repo.init(repo_dir)
    # Create initial commit on main
    (repo_dir / "README.md").write_text("# test repo")
    repo.index.add(["README.md"])
    repo.index.commit("init")
    repo.create_head("main")
    repo.heads.main.checkout()
    return repo


@pytest.fixture
def sample_pkg_dir(tmp_path):
    """Create sample generated packages."""
    pkg = tmp_path / "pkg" / "intent-20260305-0001"
    (pkg / "ric-ric-kpimon-lab-embb-i001").mkdir(parents=True)
    (pkg / "ric-ric-kpimon-lab-embb-i001" / "Kptfile").write_text("apiVersion: kpt.dev/v1")
    (pkg / "ric-ric-kpimon-lab-embb-i001" / "deployment.yaml").write_text("apiVersion: apps/v1")
    (pkg / "intent-plan.json").write_text(json.dumps({"intentId": "intent-20260305-0001"}))
    return pkg


class TestCreateBranch:
    def test_creates_new_branch(self, mock_git_repo):
        _create_branch(mock_git_repo, "intent/test-001")
        assert "intent/test-001" in [b.name for b in mock_git_repo.branches]

    def test_checkout_existing_branch(self, mock_git_repo):
        mock_git_repo.create_head("intent/existing")
        _create_branch(mock_git_repo, "intent/existing")
        assert mock_git_repo.active_branch.name == "intent/existing"


class TestCopyPackages:
    def test_copies_files_to_repo(self, mock_git_repo, sample_pkg_dir):
        files = _copy_packages(mock_git_repo, sample_pkg_dir, "intent-20260305-0001")
        assert len(files) >= 2
        dest = Path(mock_git_repo.working_dir) / "packages" / "instances" / "intent-20260305-0001"
        assert dest.exists()

    def test_overwrites_existing(self, mock_git_repo, sample_pkg_dir):
        """Re-copying should replace existing files."""
        _copy_packages(mock_git_repo, sample_pkg_dir, "intent-20260305-0001")
        files = _copy_packages(mock_git_repo, sample_pkg_dir, "intent-20260305-0001")
        assert len(files) >= 2


class TestOpenPr:
    def test_dry_run_without_token(self):
        with patch.dict(os.environ, {"GITEA_TOKEN": ""}, clear=False):
            # Re-import to pick up env var
            from llm_nephio_oran.gitops import git_ops
            git_ops.GITEA_TOKEN = ""
            result = git_ops._open_pr("test-id", "intent/test-id", {"intentType": "slice.deploy"})
        assert result["dry_run"] is True

    def test_pr_creation_success(self):
        mock_resp = MagicMock()
        mock_resp.status_code = 201
        mock_resp.raise_for_status = MagicMock()
        mock_resp.json.return_value = {"number": 42, "html_url": "http://test/pr/42"}

        with patch("llm_nephio_oran.gitops.git_ops.GITEA_TOKEN", "test-token"), \
             patch("llm_nephio_oran.gitops.git_ops.requests.post", return_value=mock_resp):
            result = _open_pr("test-id", "intent/test-id", {
                "intentType": "slice.deploy",
                "actions": [{"kind": "deploy", "component": "ric-kpimon"}],
            })
        assert result["number"] == 42

    def test_pr_409_conflict_handled(self):
        mock_resp = MagicMock()
        mock_resp.status_code = 409

        with patch("llm_nephio_oran.gitops.git_ops.GITEA_TOKEN", "test-token"), \
             patch("llm_nephio_oran.gitops.git_ops.requests.post", return_value=mock_resp):
            result = _open_pr("test-id", "intent/test-id", {"intentType": "x", "actions": []})
        assert result["existing"] is True

    def test_pr_network_error_returns_none(self):
        import requests as req_lib
        with patch("llm_nephio_oran.gitops.git_ops.GITEA_TOKEN", "test-token"), \
             patch("llm_nephio_oran.gitops.git_ops.requests.post",
                   side_effect=req_lib.ConnectionError("network error")):
            result = _open_pr("test-id", "intent/test-id", {"intentType": "x", "actions": []})
        assert result is None
