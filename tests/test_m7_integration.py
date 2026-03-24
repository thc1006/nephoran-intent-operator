"""M7 Phase A+B integration tests — live service connections.

These tests hit REAL services (Ollama, Gitea, Porch, intentd v2).
Marked with @pytest.mark.live so they can be skipped in CI:
    pytest tests/ -m "not live"       # skip live tests
    pytest tests/ -m live             # run only live tests

TDD Cycles:
  Phase A (component verification):
    1. Ollama structured output → schema-valid IntentPlan
    2. Gitea clone → branch → push → PR
    3. Porch draft → update → propose
    4. intentd v2 full pipeline (TestClient, real backends)
    5. Closed-loop → intentd v2 → Git PR (dry_run)
  Phase B (full E2E chain):
    6. Closed-loop → intentd v2 → real Git PR + real Porch Proposed
    7. Pipeline error recovery + partial failure handling
"""
from __future__ import annotations

import json
import os
import shutil
import tempfile
import time
from pathlib import Path
from typing import Any
from unittest.mock import patch

import pytest
import requests

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

PROJECT_ROOT = Path(__file__).parent.parent
SCHEMA_PATH = str(PROJECT_ROOT / "schemas" / "intent-plan.schema.json")

OLLAMA_URL = os.getenv("OLLAMA_BASE_URL", "http://localhost:11434")
GITEA_URL = os.getenv("GITEA_URL", "http://localhost:3000")
GITEA_TOKEN = os.getenv("GITEA_TOKEN", "f94032c87b4e0b1ee1e19e472a7ebb5e1823b936")
GITEA_OWNER = "nephio"
GITEA_REPO = "nephoran-packages"


def _ollama_reachable() -> bool:
    try:
        resp = requests.get(f"{OLLAMA_URL}/api/tags", timeout=3)
        return resp.status_code == 200
    except Exception:
        return False


def _gitea_reachable() -> bool:
    try:
        resp = requests.get(
            f"{GITEA_URL}/api/v1/repos/{GITEA_OWNER}/{GITEA_REPO}",
            headers={"Authorization": f"token {GITEA_TOKEN}"},
            timeout=3,
        )
        return resp.status_code == 200
    except Exception:
        return False


def _porch_reachable() -> bool:
    try:
        from kubernetes import client, config as k8s_config
        try:
            k8s_config.load_incluster_config()
        except k8s_config.ConfigException:
            k8s_config.load_kube_config()
        api = client.CustomObjectsApi()
        api.list_namespaced_custom_object(
            group="porch.kpt.dev", version="v1alpha1",
            namespace="default", plural="packagerevisions",
        )
        return True
    except Exception:
        return False


live = pytest.mark.live
skip_no_ollama = pytest.mark.skipif(not _ollama_reachable(), reason="Ollama not reachable")
skip_no_gitea = pytest.mark.skipif(not _gitea_reachable(), reason="Gitea not reachable")
skip_no_porch = pytest.mark.skipif(not _porch_reachable(), reason="Porch not reachable")


# ===========================================================================
# Cycle 1: Ollama LLM Connection
# ===========================================================================

@live
@skip_no_ollama
class TestOllamaConnection:
    """Verify Ollama can produce schema-valid IntentPlans."""

    def test_ollama_health(self):
        resp = requests.get(f"{OLLAMA_URL}/api/tags", timeout=5)
        assert resp.status_code == 200
        models = [m["name"] for m in resp.json().get("models", [])]
        assert any("qwen3" in m for m in models), f"qwen3 not found in {models}"

    def test_ollama_structured_simple(self):
        """Ollama returns valid JSON with format schema constraint."""
        resp = requests.post(
            f"{OLLAMA_URL}/api/chat",
            json={
                "model": "qwen3:14b",
                "messages": [{"role": "user", "content": "Return JSON: {\"ok\": true}"}],
                "format": {"type": "object", "properties": {"ok": {"type": "boolean"}}, "required": ["ok"]},
                "stream": False,
                "options": {"num_predict": 512, "temperature": 0.1},
            },
            timeout=30,
        )
        resp.raise_for_status()
        content = resp.json()["message"]["content"]
        assert content, "Ollama returned empty content"
        data = json.loads(content)
        assert data["ok"] is True

    def test_llm_planner_scale_intent(self):
        """plan_from_text(use_llm=True) produces schema-valid plan for scale intent."""
        from llm_nephio_oran.planner.llm_planner import plan_from_text
        from llm_nephio_oran.validators.schema_validate import validate_json_instance

        plan = plan_from_text("Scale oai-odu to 3 replicas", use_llm=True)

        # Must pass full schema validation
        validate_json_instance(SCHEMA_PATH, plan)

        # Verify semantic correctness
        assert plan["intentType"] == "slice.scale"
        assert len(plan["actions"]) >= 1
        scale_action = plan["actions"][0]
        assert scale_action["kind"] == "scale"
        assert scale_action["component"] == "oai-odu"
        assert scale_action["replicas"] == 3

    def test_llm_planner_deploy_intent(self):
        """plan_from_text produces valid plan for deploy intent."""
        from llm_nephio_oran.planner.llm_planner import plan_from_text
        from llm_nephio_oran.validators.schema_validate import validate_json_instance

        plan = plan_from_text("Deploy KPIMON and traffic generator for eMBB", use_llm=True)
        validate_json_instance(SCHEMA_PATH, plan)
        assert plan["intentType"] in ("slice.deploy", "config.update")

    def test_llm_planner_ts_intent(self):
        """plan_from_text produces valid plan for traffic steering intent."""
        from llm_nephio_oran.planner.llm_planner import plan_from_text
        from llm_nephio_oran.validators.schema_validate import validate_json_instance

        plan = plan_from_text("Enable traffic steering with 15% threshold for edge01", use_llm=True)
        validate_json_instance(SCHEMA_PATH, plan)
        assert plan["intentType"] in ("closedloop.act", "config.update")
        # Should have a configure action for ric-ts
        ts_actions = [a for a in plan["actions"] if a.get("component") == "ric-ts"]
        assert len(ts_actions) >= 1, f"No ric-ts action found in {plan['actions']}"

    def test_llm_planner_fallback_to_stub(self):
        """If Ollama returns invalid JSON, planner falls back to stub."""
        from llm_nephio_oran.planner.llm_planner import plan_from_text
        from llm_nephio_oran.validators.schema_validate import validate_json_instance

        # Patch Ollama to return garbage — should fallback to stub
        with patch("llm_nephio_oran.planner.llm_planner._call_structured", return_value=None), \
             patch("llm_nephio_oran.planner.llm_planner._call_freeform", return_value=None):
            plan = plan_from_text("Scale free5gc-amf to 2 replicas", use_llm=True)

        validate_json_instance(SCHEMA_PATH, plan)
        assert plan["actions"][0]["component"] == "free5gc-amf"
        assert plan["actions"][0]["replicas"] == 2


# ===========================================================================
# Cycle 2: Gitea Git Operations
# ===========================================================================

@live
@skip_no_gitea
class TestGiteaConnection:
    """Verify Gitea clone → branch → push → PR workflow."""

    @pytest.fixture(autouse=True)
    def _setup_gitea_env(self, tmp_path):
        """Set up env vars for Gitea and use temp work dir."""
        self._work_dir = tmp_path / "gitops"
        self._branch = f"test/m7-integration-{int(time.time())}"
        self._env = {
            "GITEA_URL": GITEA_URL,
            "GITEA_TOKEN": GITEA_TOKEN,
            "GITEA_OWNER": GITEA_OWNER,
            "GITEA_REPO": GITEA_REPO,
            "GITEA_CLONE_URL": f"{GITEA_URL}/{GITEA_OWNER}/{GITEA_REPO}.git",
            "GITOPS_WORK_DIR": str(self._work_dir),
        }
        yield
        # Cleanup: delete test branch via API
        try:
            requests.delete(
                f"{GITEA_URL}/api/v1/repos/{GITEA_OWNER}/{GITEA_REPO}/branches/{self._branch}",
                headers={"Authorization": f"token {GITEA_TOKEN}"},
                timeout=5,
            )
        except Exception:
            pass

    def test_gitea_clone(self):
        """Can clone nephoran-packages from Gitea."""
        with patch.dict(os.environ, self._env):
            # Re-import to pick up new env vars
            import importlib
            import llm_nephio_oran.gitops.git_ops as git_mod
            importlib.reload(git_mod)

            repo = git_mod._ensure_repo()
            assert (Path(repo.working_dir) / ".git").exists()
            assert repo.active_branch.name == "main"

    def test_gitea_push_to_gitops(self):
        """Full push_to_gitops: clone → branch → copy → commit → push → PR."""
        from llm_nephio_oran.planner.stub_planner import plan_from_text
        from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages

        plan = plan_from_text("Scale oai-odu to 2 replicas")
        intent_id = plan["intentId"]

        # Generate packages in temp dir
        pkg_out = self._work_dir / "pkg_out"
        pkg_dir = generate_kpt_packages(plan, pkg_out)
        assert pkg_dir.exists()

        with patch.dict(os.environ, self._env):
            import importlib
            import llm_nephio_oran.gitops.git_ops as git_mod
            importlib.reload(git_mod)

            result = git_mod.push_to_gitops(pkg_dir, plan)

        assert result["intent_id"] == intent_id
        assert result["branch"].startswith("intent/")
        assert result["commit_sha"], "No commit SHA returned"
        assert result["files_count"] > 0
        # PR should be created (or dry_run if no token)
        assert result["pr"] is not None

        # Store branch name for cleanup
        self._branch = result["branch"]


# ===========================================================================
# Cycle 3: Porch Lifecycle
# ===========================================================================

@live
@skip_no_porch
class TestPorchConnection:
    """Verify Porch draft → update → propose lifecycle."""

    def test_porch_list_packages(self):
        """Can list PackageRevisions from Porch."""
        from llm_nephio_oran.porch.client import PorchClient
        pc = PorchClient()
        pkgs = pc.list_packages()
        # Should return a list (possibly empty)
        assert isinstance(pkgs, list)

    def test_porch_list_nephoran_packages(self):
        """Can list packages in nephoran-packages repository."""
        from llm_nephio_oran.porch.client import PorchClient
        pc = PorchClient()
        pkgs = pc.list_packages("nephoran-packages")
        assert isinstance(pkgs, list)
        # Log what we found
        for p in pkgs[:5]:
            name = p["metadata"]["name"]
            lifecycle = p["spec"]["lifecycle"]
            pkg_name = p["spec"]["packageName"]
            print(f"  {name}: {pkg_name} ({lifecycle})")

    def test_porch_create_draft_and_update(self):
        """Create draft + update resources."""
        from llm_nephio_oran.porch.client import PorchClient
        from kubernetes.client.exceptions import ApiException
        pc = PorchClient()

        test_pkg = f"test/m7-integration-{int(time.time())}"
        draft_name = None

        try:
            draft = pc.create_draft(
                repo="nephoran-packages",
                package_name=test_pkg,
                workspace="v1",
            )
            draft_name = draft["metadata"]["name"]
            assert draft["spec"]["lifecycle"] == "Draft"

            resources = {
                "test-configmap.yaml": (
                    "apiVersion: v1\nkind: ConfigMap\n"
                    "metadata:\n  name: m7-test\n  namespace: default\n"
                    "data:\n  test: integration\n"
                ),
            }
            # Porch needs time to create PackageRevisionResources after draft
            updated = None
            for attempt in range(10):
                try:
                    updated = pc.update_resources(draft_name, resources)
                    break
                except ApiException as e:
                    if e.status == 403 and "not yet ready" in str(e.body):
                        time.sleep(2)
                    else:
                        raise
            assert updated is not None, "update_resources failed after retries"

        finally:
            if draft_name:
                try:
                    from kubernetes import client, config as k8s_config
                    try:
                        k8s_config.load_incluster_config()
                    except k8s_config.ConfigException:
                        k8s_config.load_kube_config()
                    api = client.CustomObjectsApi()
                    api.delete_namespaced_custom_object(
                        group="porch.kpt.dev", version="v1alpha1",
                        namespace="default", plural="packagerevisions",
                        name=draft_name,
                    )
                except Exception:
                    pass

    def test_porch_propose(self):
        """Draft → Proposed lifecycle (requires Porch→Gitea auth configured)."""
        from llm_nephio_oran.porch.client import PorchClient
        from kubernetes.client.exceptions import ApiException
        pc = PorchClient()

        test_pkg = f"test/m7-propose-{int(time.time())}"
        draft_name = None

        try:
            draft = pc.create_draft(
                repo="nephoran-packages",
                package_name=test_pkg,
                workspace="v1",
            )
            draft_name = draft["metadata"]["name"]

            resources = {
                "test-configmap.yaml": (
                    "apiVersion: v1\nkind: ConfigMap\n"
                    "metadata:\n  name: m7-propose-test\n  namespace: default\n"
                    "data:\n  test: propose\n"
                ),
            }
            # Retry until PackageRevisionResources is ready
            for attempt in range(10):
                try:
                    pc.update_resources(draft_name, resources)
                    break
                except ApiException as e:
                    if e.status == 403 and "not yet ready" in str(e.body):
                        time.sleep(2)
                    else:
                        raise

            proposed = pc.propose(draft_name)
            assert proposed["spec"]["lifecycle"] == "Proposed"

        finally:
            if draft_name:
                try:
                    from kubernetes import client, config as k8s_config
                    try:
                        k8s_config.load_incluster_config()
                    except k8s_config.ConfigException:
                        k8s_config.load_kube_config()
                    api = client.CustomObjectsApi()
                    api.delete_namespaced_custom_object(
                        group="porch.kpt.dev", version="v1alpha1",
                        namespace="default", plural="packagerevisions",
                        name=draft_name,
                    )
                except Exception:
                    pass


# ===========================================================================
# Cycle 4: intentd v2 Full Pipeline (TestClient, real backends)
# ===========================================================================

@live
@skip_no_ollama
class TestIntentdV2Pipeline:
    """Verify intentd v2 async pipeline with real Ollama backend."""

    @pytest.fixture
    def app_client(self, tmp_path):
        """Create intentd v2 app with TestClient and temp state dir."""
        from llm_nephio_oran.intentd.app_v2 import create_app
        from fastapi.testclient import TestClient

        app = create_app(store_dir=tmp_path / ".state")
        yield TestClient(app)

    def test_healthz(self, app_client):
        resp = app_client.get("/healthz")
        assert resp.status_code == 200
        assert resp.json()["status"] == "ok"

    def test_pipeline_dry_run_with_stub(self, app_client):
        """Dry-run pipeline with stub planner (no LLM) completes."""
        resp = app_client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Scale oai-odu to 2 replicas",
            "source": "test",
            "use_llm": False,
            "dry_run": True,
        })
        assert resp.status_code == 201
        intent_id = resp.json()["id"]

        # Poll until terminal
        for _ in range(30):
            time.sleep(0.2)
            rec = app_client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            if rec["state"] in ("completed", "failed"):
                break

        assert rec["state"] == "completed", f"Pipeline ended in {rec['state']}: {rec.get('errors')}"
        assert rec["report"]["mode"] == "dry_run"

    def test_pipeline_dry_run_with_llm(self, app_client):
        """Dry-run pipeline with REAL LLM (Ollama) completes."""
        resp = app_client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Scale free5gc-amf to 3 replicas",
            "source": "test",
            "use_llm": True,
            "dry_run": True,
        })
        assert resp.status_code == 201
        intent_id = resp.json()["id"]

        # LLM call may take up to 60s on CPU
        for _ in range(120):
            time.sleep(0.5)
            rec = app_client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            if rec["state"] in ("completed", "failed"):
                break

        assert rec["state"] == "completed", f"Pipeline ended in {rec['state']}: {rec.get('errors')}"

        # Verify plan was generated
        assert rec["plan"] is not None
        assert rec["plan"]["intentType"] == "slice.scale"

    def test_intent_report(self, app_client):
        """IntentReport reflects correct TMF921 compliance state."""
        resp = app_client.post("/tmf-api/intent/v5/intent", json={
            "expression": "Deploy ric-kpimon",
            "source": "test",
            "use_llm": False,
            "dry_run": True,
        })
        intent_id = resp.json()["id"]

        for _ in range(30):
            time.sleep(0.2)
            rec = app_client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
            if rec["state"] in ("completed", "failed"):
                break

        report = app_client.get(f"/tmf-api/intent/v5/intent/{intent_id}/report").json()
        assert report["complianceState"] == "compliant"
        assert report["lifecycleStatus"] == "Active"
        assert report["terminal"] is True


# ===========================================================================
# Cycle 5: Closed Loop → intentd v2
# ===========================================================================

@live
@skip_no_ollama
class TestClosedLoopToIntentd:
    """Verify E2 KPM closed-loop triggers intent creation via intentd v2."""

    def test_closed_loop_creates_intent_via_api(self):
        """High PRB triggers scale_out → POST to intentd v2 → pipeline completes."""
        from llm_nephio_oran.intentd.app_v2 import create_app
        from llm_nephio_oran.e2sim.kpm_simulator import E2KpmSimulator, CellConfig
        from llm_nephio_oran.kpimon.bridge import KpimonBridge
        from llm_nephio_oran.closedloop.e2_closed_loop import ClosedLoopE2EController
        from fastapi.testclient import TestClient

        with tempfile.TemporaryDirectory() as tmp:
            app = create_app(store_dir=Path(tmp) / ".state")
            client = TestClient(app)

            sim = E2KpmSimulator(
                cells=[CellConfig(cell_id="cell-e2e", gnb_id="gnb-edge01", slices=["embb"])],
                load_factor=0.95,
                seed=42,
            )
            bridge = KpimonBridge(sim)
            controller = ClosedLoopE2EController(
                bridge=bridge,
                prb_threshold=80.0,
                dry_run=True,
                max_replicas=5,
            )
            controller._client = client

            results = controller.run_cycle()
            assert len(results) >= 1, "No intents created from high-load cell"

            result = results[0]
            assert "id" in result, f"No intent ID in result: {result}"
            intent_id = result["id"]

            # Poll until pipeline completes
            for _ in range(30):
                time.sleep(0.2)
                rec = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
                if rec["state"] in ("completed", "failed"):
                    break

            assert rec["state"] == "completed", f"State={rec['state']}, errors={rec.get('errors')}"

    def test_low_load_no_intent(self):
        """Low PRB → no intent created (no false positives)."""
        from llm_nephio_oran.e2sim.kpm_simulator import E2KpmSimulator, CellConfig
        from llm_nephio_oran.kpimon.bridge import KpimonBridge
        from llm_nephio_oran.closedloop.e2_closed_loop import ClosedLoopE2EController

        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="cell-low", gnb_id="gnb-edge01", slices=["embb"])],
            load_factor=0.2,
            seed=42,
        )
        bridge = KpimonBridge(sim)
        controller = ClosedLoopE2EController(
            bridge=bridge,
            prb_threshold=80.0,
            dry_run=True,
        )

        results = controller.run_cycle()
        assert len(results) == 0, f"Unexpected intents from low-load: {results}"


# ===========================================================================
# Cycle 6: Phase B — Full E2E Chain (non-dry-run, real Git + Porch)
# ===========================================================================

def _all_services_reachable() -> bool:
    """Check if ALL live services (Ollama, Gitea, Porch) are reachable."""
    return _ollama_reachable() and _gitea_reachable() and _porch_reachable()


skip_no_all = pytest.mark.skipif(
    not _all_services_reachable(), reason="Not all services reachable (need Ollama+Gitea+Porch)"
)


@live
@skip_no_all
class TestPhaseB_FullE2EChain:
    """Phase B: Full closed-loop E2E chain with REAL Git push + Porch lifecycle.

    This is the M7 demo scenario:
      E2KpmSim(0.95) → PRB>80% → scale_out → POST intent
        → pipeline: plan → validate → generate → Git push → PR → Porch propose
        → intent state: acknowledged → ... → completed
    """

    @pytest.fixture(autouse=True)
    def _setup_gitea_and_cleanup(self, tmp_path):
        """Set Gitea env vars, track resources for cleanup."""
        self._work_dir = tmp_path / "gitops-e2e"
        self._gitea_env = {
            "GITEA_URL": GITEA_URL,
            "GITEA_TOKEN": GITEA_TOKEN,
            "GITEA_OWNER": GITEA_OWNER,
            "GITEA_REPO": GITEA_REPO,
            "GITEA_CLONE_URL": f"{GITEA_URL}/{GITEA_OWNER}/{GITEA_REPO}.git",
            "GITOPS_WORK_DIR": str(self._work_dir),
        }
        self._created_branches: list[str] = []
        self._created_porch_pkgs: list[str] = []

        yield

        # Cleanup Gitea branches + PRs
        for branch in self._created_branches:
            try:
                requests.delete(
                    f"{GITEA_URL}/api/v1/repos/{GITEA_OWNER}/{GITEA_REPO}/branches/{branch}",
                    headers={"Authorization": f"token {GITEA_TOKEN}"},
                    timeout=5,
                )
            except Exception:
                pass
        # Cleanup Porch PackageRevisions
        for pkg_name in self._created_porch_pkgs:
            try:
                from kubernetes import client, config as k8s_config
                try:
                    k8s_config.load_incluster_config()
                except k8s_config.ConfigException:
                    k8s_config.load_kube_config()
                api = client.CustomObjectsApi()
                api.delete_namespaced_custom_object(
                    group="porch.kpt.dev", version="v1alpha1",
                    namespace="default", plural="packagerevisions",
                    name=pkg_name,
                )
            except Exception:
                pass

    def test_e2e_full_chain_high_prb_to_git_pr_and_porch(self):
        """M7 demo: E2 KPM high PRB → closed-loop → Git PR + Porch Proposed."""
        import importlib
        from llm_nephio_oran.intentd.app_v2 import create_app
        from llm_nephio_oran.e2sim.kpm_simulator import E2KpmSimulator, CellConfig
        from llm_nephio_oran.kpimon.bridge import KpimonBridge
        from llm_nephio_oran.closedloop.e2_closed_loop import ClosedLoopE2EController
        from fastapi.testclient import TestClient

        with patch.dict(os.environ, self._gitea_env):
            # Reload git_ops so module-level constants pick up env vars
            import llm_nephio_oran.gitops.git_ops as git_mod
            importlib.reload(git_mod)

            with tempfile.TemporaryDirectory() as tmp:
                app = create_app(store_dir=Path(tmp) / ".state")
                client = TestClient(app)

                sim = E2KpmSimulator(
                    cells=[CellConfig(cell_id="cell-e2e-b", gnb_id="gnb-edge01", slices=["embb"])],
                    load_factor=0.95,
                    seed=99,
                )
                bridge = KpimonBridge(sim)
                controller = ClosedLoopE2EController(
                    bridge=bridge,
                    prb_threshold=80.0,
                    dry_run=False,  # ← KEY: non-dry-run = real Git + Porch
                    max_replicas=5,
                )
                controller._client = client

                # ── ACT: run closed-loop cycle ──
                results = controller.run_cycle()
                assert len(results) >= 1, "No intents created from high-load cell"

                result = results[0]
                assert "id" in result, f"No intent ID in result: {result}"
                intent_id = result["id"]

                # ── POLL: wait for async pipeline to complete ──
                rec = None
                for _ in range(90):  # up to 45s (git clone/push can be slow)
                    time.sleep(0.5)
                    rec = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
                    if rec["state"] in ("completed", "failed"):
                        break

                assert rec is not None
                assert rec["state"] == "completed", (
                    f"Pipeline ended in {rec['state']}: errors={rec.get('errors')}, "
                    f"report={rec.get('report')}"
                )

                # ── VERIFY: Git artifacts ──
                report = rec.get("report", {})
                git_info = report.get("git") or rec.get("git", {})
                assert git_info.get("branch"), f"No Git branch in result: {rec}"
                assert git_info.get("commit_sha"), f"No commit SHA: {rec}"
                self._created_branches.append(git_info["branch"])

                # Verify PR was created on Gitea
                pr_info = git_info.get("pr")
                if pr_info and pr_info.get("number"):
                    # Check PR exists via Gitea API
                    resp = requests.get(
                        f"{GITEA_URL}/api/v1/repos/{GITEA_OWNER}/{GITEA_REPO}/pulls/{pr_info['number']}",
                        headers={"Authorization": f"token {GITEA_TOKEN}"},
                        timeout=5,
                    )
                    assert resp.status_code == 200, f"PR #{pr_info['number']} not found"

                # ── VERIFY: Porch artifacts ──
                porch_info = report.get("porch") or rec.get("porch", {})
                assert not porch_info.get("error"), f"Porch failed: {porch_info}"
                assert porch_info.get("name"), f"No Porch package name: {porch_info}"
                assert porch_info.get("lifecycle") == "Proposed", (
                    f"Expected Proposed, got: {porch_info}"
                )
                self._created_porch_pkgs.append(porch_info["name"])

    def test_e2e_pipeline_records_all_stages(self):
        """Verify completed intent has data from every pipeline stage."""
        import importlib
        from llm_nephio_oran.intentd.app_v2 import create_app
        from fastapi.testclient import TestClient

        with patch.dict(os.environ, self._gitea_env):
            import llm_nephio_oran.gitops.git_ops as git_mod
            importlib.reload(git_mod)

            with tempfile.TemporaryDirectory() as tmp:
                app = create_app(store_dir=Path(tmp) / ".state")
                client = TestClient(app)

                # Direct POST (not via closed-loop) with dry_run=False
                resp = client.post("/tmf-api/intent/v5/intent", json={
                    "expression": "Scale oai-odu to 2 replicas",
                    "source": "test-phase-b",
                    "use_llm": False,
                    "dry_run": False,
                })
                assert resp.status_code == 201
                intent_id = resp.json()["id"]

                # Poll until terminal
                rec = None
                for _ in range(90):
                    time.sleep(0.5)
                    rec = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
                    if rec["state"] in ("completed", "failed"):
                        break

                assert rec["state"] == "completed", (
                    f"State={rec['state']}, errors={rec.get('errors')}"
                )

                # Track for cleanup
                report = rec.get("report", {})
                git_info = report.get("git", {})
                if git_info.get("branch"):
                    self._created_branches.append(git_info["branch"])
                porch_info = report.get("porch", {})
                if porch_info.get("name"):
                    self._created_porch_pkgs.append(porch_info["name"])

                # Verify all pipeline stages produced data
                assert rec.get("plan") is not None, "No plan recorded"
                assert rec["plan"]["intentType"] == "slice.scale"
                assert rec["plan"]["actions"][0]["component"] == "oai-odu"
                assert rec["plan"]["actions"][0]["replicas"] == 2

                assert git_info.get("commit_sha"), "No commit SHA"
                assert git_info.get("branch"), "No branch"

                assert porch_info.get("name"), "No Porch package"
                assert porch_info.get("lifecycle") == "Proposed"
                assert porch_info.get("files", 0) > 0, "No files in Porch package"

                # ConfigSync delivery should have been checked
                cs_info = report.get("configsync", {})
                assert "delivered" in cs_info or "error" in cs_info, (
                    f"No ConfigSync status: {cs_info}"
                )

    def test_e2e_porch_failure_does_not_block_pipeline(self):
        """If Porch is unavailable, pipeline still completes (porch_failed=True)."""
        import importlib
        from llm_nephio_oran.intentd.app_v2 import create_app
        from fastapi.testclient import TestClient

        with patch.dict(os.environ, self._gitea_env):
            import llm_nephio_oran.gitops.git_ops as git_mod
            importlib.reload(git_mod)

            with tempfile.TemporaryDirectory() as tmp:
                app = create_app(store_dir=Path(tmp) / ".state")
                client = TestClient(app)

                # Patch PorchClient at source (imported inside _run_pipeline)
                with patch(
                    "llm_nephio_oran.porch.client.PorchClient",
                    side_effect=Exception("Porch connection refused"),
                ):
                    resp = client.post("/tmf-api/intent/v5/intent", json={
                        "expression": "Scale trafficgen to 2 replicas",
                        "source": "test-phase-b",
                        "use_llm": False,
                        "dry_run": False,
                    })
                    assert resp.status_code == 201
                    intent_id = resp.json()["id"]

                    rec = None
                    for _ in range(90):
                        time.sleep(0.5)
                        rec = client.get(f"/tmf-api/intent/v5/intent/{intent_id}").json()
                        if rec["state"] in ("completed", "failed"):
                            break

                    assert rec["state"] == "completed", (
                        f"Pipeline should complete even if Porch fails: {rec['state']}"
                    )
                    report = rec.get("report", {})
                    assert report.get("porch_failed") is True, "Should flag porch_failed"

                    # Git should still have succeeded
                    git_info = report.get("git", {})
                    assert git_info.get("commit_sha"), "Git should succeed even if Porch fails"
                    if git_info.get("branch"):
                        self._created_branches.append(git_info["branch"])
