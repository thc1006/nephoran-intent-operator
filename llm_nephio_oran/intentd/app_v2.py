"""intentd v2 — TMF921-aligned async Intent Management API.

Architecture fixes over v1:
  1. Single entry point: POST /tmf-api/intent/v5/intent (creates + triggers pipeline)
  2. Async pipeline: returns 201 immediately, background worker processes
  3. State machine: acknowledged → planning → ... → completed / failed
  4. GET /intent/{id} returns live state (polling by frontend)
  5. Closed-loop controller also routes through this API

Endpoints:
  POST   /tmf-api/intent/v5/intent           → create intent + trigger async pipeline
  GET    /tmf-api/intent/v5/intent           → list all intents
  GET    /tmf-api/intent/v5/intent/{id}      → get intent with live state
  DELETE /tmf-api/intent/v5/intent/{id}      → cancel intent (from acknowledged only)
  GET    /tmf-api/intent/v5/intent/{id}/report → TMF921 IntentReport
  GET    /healthz                            → health check
  GET    /api/porch/packages                 → Porch PackageRevision status
  GET    /api/scale/status                   → current CNF replica counts
"""
from __future__ import annotations

import logging
import os
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from llm_nephio_oran.intentd.store import IntentState, IntentStore, InvalidTransition

logger = logging.getLogger(__name__)

STORE_DIR = Path(".state")


class CreateIntentRequest(BaseModel):
    expression: str
    source: str = "tmf921"  # cli | web | tmf921 | closedloop
    use_llm: bool = True
    dry_run: bool = False


def create_app(store_dir: Path | None = None) -> FastAPI:
    """Factory: create FastAPI app with its own IntentStore."""
    from contextlib import asynccontextmanager

    _store = IntentStore(state_dir=store_dir or STORE_DIR)
    _executor = ThreadPoolExecutor(max_workers=4, thread_name_prefix="pipeline")

    @asynccontextmanager
    async def lifespan(app: FastAPI):  # type: ignore[no-untyped-def]
        yield
        _executor.shutdown(wait=True, cancel_futures=False)

    app = FastAPI(
        title="intentd v2 (TMF921 async)",
        version="2.0.0",
        lifespan=lifespan,
    )
    cors_origins = os.getenv("CORS_ALLOWED_ORIGINS", "*").split(",")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=cors_origins,
        allow_methods=["GET", "POST", "DELETE"],
        allow_headers=["Content-Type", "Authorization"],
    )

    # ── TMF921 endpoints ───────────────────────────────────────────────

    @app.get("/healthz")
    def healthz() -> dict[str, str]:
        return {"status": "ok"}

    @app.get("/metrics")
    def metrics():
        from llm_nephio_oran.observability.pipeline_metrics import generate_metrics_text
        from fastapi.responses import PlainTextResponse
        return PlainTextResponse(generate_metrics_text(), media_type="text/plain")

    @app.post("/tmf-api/intent/v5/intent", status_code=201)
    def create_intent(req: CreateIntentRequest) -> dict[str, Any]:
        """Create intent → acknowledged immediately, pipeline runs async."""
        from llm_nephio_oran.observability import pipeline_metrics as _pm
        _pm.intent_created.labels(source=req.source).inc()
        rec = _store.create(expression=req.expression, source=req.source)

        # Submit pipeline to bounded thread pool (max 4 concurrent pipelines)
        _executor.submit(_run_pipeline, _store, rec["id"], req.use_llm, req.dry_run)

        return rec

    @app.get("/tmf-api/intent/v5/intent")
    def list_intents() -> list[dict[str, Any]]:
        return _store.list_all()

    @app.get("/tmf-api/intent/v5/intent/{intent_id}")
    def get_intent(intent_id: str) -> dict[str, Any]:
        rec = _store.get(intent_id)
        if rec is None:
            raise HTTPException(status_code=404, detail="Intent not found")
        return rec

    @app.delete("/tmf-api/intent/v5/intent/{intent_id}")
    def delete_intent(intent_id: str) -> dict[str, Any]:
        """Cancel an intent (TMF921 DELETE). Only valid from acknowledged state."""
        rec = _store.get(intent_id)
        if rec is None:
            raise HTTPException(status_code=404, detail="Intent not found")
        try:
            return _store.cancel(intent_id)
        except InvalidTransition:
            raise HTTPException(
                status_code=409,
                detail=f"Cannot cancel intent in '{rec['state']}' state (only acknowledged)",
            )

    @app.get("/tmf-api/intent/v5/intent/{intent_id}/report")
    def get_intent_report(intent_id: str) -> dict[str, Any]:
        """TMF921 IntentReport — compliance summary for an intent."""
        report = _store.build_report(intent_id)
        if report is None:
            raise HTTPException(status_code=404, detail="Intent not found")
        return report

    @app.get("/api/metrics/json")
    def metrics_json() -> dict[str, Any]:
        """Structured JSON metrics snapshot for the React Closed-Loop dashboard."""
        from llm_nephio_oran.observability.pipeline_metrics import collect_json_snapshot
        return collect_json_snapshot()

    # ── Frontend helper endpoints ──────────────────────────────────────

    @app.get("/api/porch/packages")
    def list_porch_packages() -> list[dict[str, Any]]:
        try:
            from llm_nephio_oran.porch.client import PorchClient
            pc = PorchClient()
            pkgs = pc.list_packages("nephoran-packages")
            return [
                {
                    "name": p["metadata"]["name"],
                    "package": p["spec"]["packageName"],
                    "lifecycle": p["spec"]["lifecycle"],
                    "workspace": p["spec"].get("workspaceName", ""),
                    "repository": p["spec"]["repository"],
                }
                for p in pkgs
            ]
        except Exception as e:
            logger.warning("Cannot list Porch packages: %s", e)
            return []

    import threading
    _k8s_apps: list[Any] = []  # cached K8s client [AppsV1Api]
    _k8s_lock = threading.Lock()

    def _get_apps_v1() -> Any:
        if not _k8s_apps:
            with _k8s_lock:
                if not _k8s_apps:  # double-check after acquiring lock
                    from kubernetes import client, config as k8s_config
                    try:
                        k8s_config.load_incluster_config()
                    except k8s_config.ConfigException:
                        k8s_config.load_kube_config()
                    _k8s_apps.append(client.AppsV1Api())
        return _k8s_apps[0]

    @app.get("/api/scale/status")
    def get_scale_status() -> dict[str, Any]:
        try:
            apps_v1 = _get_apps_v1()
            namespaces = {
                "oran": "ran", "free5gc": "core", "ricplt": "ric",
                "ricxapp": "ric", "sim": "sim", "nephoran-system": "obs",
            }
            components = []
            for ns, domain in namespaces.items():
                try:
                    deps = apps_v1.list_namespaced_deployment(namespace=ns)
                    for d in deps.items:
                        labels = d.metadata.labels or {}
                        components.append({
                            "name": d.metadata.name,
                            "namespace": ns,
                            "domain": domain,
                            "replicas": d.spec.replicas or 0,
                            "ready": d.status.ready_replicas or 0,
                            "available": d.status.available_replicas or 0,
                            "intentId": labels.get("oran.ai/intent-id"),
                            "slice": labels.get("oran.ai/slice"),
                            "site": labels.get("oran.ai/site"),
                        })
                except Exception as e:
                    logger.debug("Namespace %s not accessible: %s", ns, e)
            return {"components": components, "timestamp": datetime.now(timezone.utc).isoformat()}
        except Exception as e:
            logger.warning("Cannot get scale status: %s", e)
            return {"components": [], "error": str(e)}

    return app


# ── ConfigSync delivery check (pure function, testable) ───────────────────

def check_configsync_delivery(
    intent_commit: str,
    sync_status: Any,
) -> dict[str, Any]:
    """Compare intent commit with ConfigSync sync status.

    Pure function extracted from pipeline for testability.
    Aligns with TMF921 TR292B compliance state determination:
      - commit_match + synced + no errors → delivered (Compliant)
      - commit_match + errors → partial (Degraded)
      - no match or empty commit → not delivered (NonCompliant)

    Args:
        intent_commit: The git commit SHA produced by the pipeline's git push.
        sync_status: A SyncStatus object with .synced, .commit, .errors fields.

    Returns:
        Dict with: delivered, synced, commit, intent_commit, commit_match, errors.
    """
    if not intent_commit or sync_status is None:
        return {
            "delivered": False,
            "synced": False,
            "commit": sync_status.commit if sync_status else "",
            "intent_commit": intent_commit or "",
            "commit_match": False,
            "errors": (sync_status.errors if sync_status else []),
        }

    commit_match = sync_status.commit == intent_commit
    delivered = sync_status.synced and commit_match and len(sync_status.errors) == 0
    return {
        "delivered": delivered,
        "synced": sync_status.synced,
        "commit": sync_status.commit,
        "intent_commit": intent_commit,
        "commit_match": commit_match,
        "errors": sync_status.errors,
    }


# ── Async Pipeline Worker ──────────────────────────────────────────────────

def _run_pipeline(
    store: IntentStore,
    intent_id: str,
    use_llm: bool,
    dry_run: bool,
) -> None:
    """Background pipeline: plan → validate → generate → git → porch.

    Each step transitions the intent state. On failure, transitions to FAILED.
    """
    from llm_nephio_oran.planner.llm_planner import plan_from_text as _plan_from_text
    from llm_nephio_oran.observability import pipeline_metrics as pm

    pm.active_intents.inc()
    try:
        # Step 1: Planning
        store.transition(intent_id, IntentState.PLANNING)
        rec = store.get(intent_id)
        with pm.instrument_stage("planning"):
            plan = _plan_from_text(rec["expression"], use_llm=use_llm)

        # Step 2: Validating
        store.transition(intent_id, IntentState.VALIDATING, data={"plan": plan})
        from llm_nephio_oran.validators.schema_validate import validate_json_instance
        with pm.instrument_stage("validating"):
            validate_json_instance("schemas/intent-plan.schema.json", plan)

        # Step 3: Generating
        store.transition(intent_id, IntentState.GENERATING)
        from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages
        out_dir = Path("packages/instances")
        with pm.instrument_stage("generating"):
            pkg_dir = generate_kpt_packages(plan, out_dir)

        # Step 3b: Validate manifests
        from llm_nephio_oran.validators.manifest_validator import validate_package
        manifest_errors = validate_package(pkg_dir)
        if manifest_errors:
            store.transition(intent_id, IntentState.FAILED, data={
                "errors": manifest_errors,
                "report": {"stage": "manifest_validation", "detail": manifest_errors},
            })
            pm.intent_completed.labels(result="failed").inc()
            pm.active_intents.dec()
            return

        if dry_run:
            # Dry run: skip git+porch, mark completed
            store.transition(intent_id, IntentState.EXECUTING, data={"package_dir": str(pkg_dir)})
            store.transition(intent_id, IntentState.PROPOSED)
            store.transition(intent_id, IntentState.APPLIED)
            store.transition(intent_id, IntentState.COMPLETED, data={
                "report": {"mode": "dry_run", "package_dir": str(pkg_dir)},
            })
            pm.intent_completed.labels(result="completed").inc()
            pm.active_intents.dec()
            return

        # Step 4: Executing (git push + PR)
        store.transition(intent_id, IntentState.EXECUTING, data={"package_dir": str(pkg_dir)})
        from llm_nephio_oran.gitops.git_ops import push_to_gitops
        with pm.instrument_stage("executing"):
            raw_git = push_to_gitops(pkg_dir, plan)
        pm.git_commits.inc()
        intent_id_from_plan = plan.get("intentId", intent_id)

        # Normalize git result to match frontend IntentRecord.git contract
        pr_raw = raw_git.get("pr")
        pr_normalized = None
        if isinstance(pr_raw, dict) and pr_raw.get("number"):
            pr_normalized = {
                "number": pr_raw["number"],
                "url": pr_raw.get("html_url") or pr_raw.get("url", ""),
            }
        git_result: dict[str, Any] = {
            "branch": raw_git.get("branch", ""),
            "commit_sha": raw_git.get("commit_sha", ""),
        }
        if pr_normalized:
            git_result["pr"] = pr_normalized

        # Step 5: Proposed (Porch lifecycle)
        porch_data: dict[str, Any] = {}
        porch_failed = False
        try:
            from llm_nephio_oran.porch.client import PorchClient
            porch = PorchClient()
            pkg_name = f"instances/{intent_id_from_plan}"
            draft = porch.create_draft(repo="nephoran-packages", package_name=pkg_name)
            draft_name = draft["metadata"]["name"]
            resources = {}
            for f_path in pkg_dir.rglob("*.yaml"):
                resources[str(f_path.relative_to(pkg_dir))] = f_path.read_text()
            porch.update_resources(draft_name, resources)
            porch.propose(draft_name)
            porch_data = {"name": draft_name, "lifecycle": "Proposed", "files": len(resources)}
            pm.porch_lifecycle.labels(lifecycle="Proposed").inc()
        except Exception as e:
            logger.warning("Porch integration failed: %s", e)
            porch_data = {"error": str(e)}
            porch_failed = True

        # Step 5b: Create per-intent RepoSync for namespace-scoped delivery
        reposync_data: dict[str, Any] = {}
        try:
            from llm_nephio_oran.configsync.client import ConfigSyncClient, RepoSyncSpec
            cs = ConfigSyncClient()
            # Determine target namespace from plan actions
            # Actions use naming.domain, not a direct namespace field.
            # Map domain → K8s namespace (same mapping as kpt_generator).
            _domain_ns_map = {
                "ran": "oran", "core": "free5gc", "ric": "ricplt",
                "sim": "sim", "obs": "monitoring",
            }
            target_ns = "default"
            for action in plan.get("actions", []):
                ns = action.get("namespace")
                if ns:
                    target_ns = ns
                    break
                domain = (action.get("naming") or {}).get("domain")
                if domain and domain in _domain_ns_map:
                    target_ns = _domain_ns_map[domain]
                    break
            reposync_spec = RepoSyncSpec(
                name=f"intent-{intent_id_from_plan}",
                namespace=target_ns,
                repo=os.getenv("GITEA_CLONE_URL", "http://gitea-http.gitea.svc.cluster.local:3000/nephio/nephoran-packages.git"),
                branch="main",
                directory=f"packages/instances/{intent_id_from_plan}",
            )
            cs.ensure_reposync(reposync_spec)
            reposync_data = {
                "name": reposync_spec.name,
                "namespace": target_ns,
                "directory": reposync_spec.directory,
            }
        except Exception as e:
            logger.warning("Per-intent RepoSync creation failed (non-fatal): %s", e)
            reposync_data = {"error": str(e)}

        store.transition(intent_id, IntentState.PROPOSED, data={
            "git": git_result,
            "porch": porch_data,
            "reposync": reposync_data,
        })

        # Step 6: Applied — verify ConfigSync has synced this intent's commit
        configsync_data: dict[str, Any] = {}
        intent_commit = git_result.get("commit_sha", "")
        try:
            from llm_nephio_oran.configsync.client import ConfigSyncClient
            cs = ConfigSyncClient()
            sync_status = cs.get_rootsync_status("nephoran-packages")
            configsync_data = check_configsync_delivery(intent_commit, sync_status)
            if not configsync_data["commit_match"] and intent_commit:
                logger.info(
                    "ConfigSync commit %s does not match intent commit %s (may need time to sync)",
                    sync_status.commit[:8] if sync_status.commit else "none",
                    intent_commit[:8],
                )
        except Exception as e:
            logger.warning("ConfigSync status check failed: %s", e)
            configsync_data = {"error": str(e)}

        store.transition(intent_id, IntentState.APPLIED, data={
            "configsync": configsync_data,
        })

        # Step 7: Completed
        store.transition(intent_id, IntentState.COMPLETED, data={
            "report": {
                "git": git_result,
                "porch": porch_data,
                "configsync": configsync_data,
                "package_dir": str(pkg_dir),
                "porch_failed": porch_failed,
            },
        })
        pm.intent_completed.labels(result="completed").inc()
        pm.active_intents.dec()

        # Cleanup: remove per-intent RepoSync (delivery is done)
        _cleanup_reposync(store, intent_id)

    except Exception as e:
        logger.exception("Pipeline failed for %s", intent_id)
        try:
            store.transition(intent_id, IntentState.FAILED, data={
                "errors": [str(e)],
                "report": {"stage": "pipeline", "error": str(e)},
            })
        except Exception:
            logger.exception(
                "CRITICAL: Failed to record failure for %s — "
                "intent state and metrics may be inconsistent", intent_id,
            )
        pm.intent_completed.labels(result="failed").inc()
        pm.active_intents.dec()
        _cleanup_reposync(store, intent_id)


def _cleanup_reposync(store: IntentStore, intent_id: str) -> None:
    """Delete per-intent RepoSync when intent reaches terminal state."""
    rec = store.get(intent_id)
    if rec is None:
        return
    reposync = rec.get("reposync")
    if not reposync or reposync.get("error") or not reposync.get("name"):
        return
    try:
        from llm_nephio_oran.configsync.client import ConfigSyncClient
        cs = ConfigSyncClient()
        cs.delete_reposync(
            name=reposync["name"],
            namespace=reposync["namespace"],
        )
        logger.info("Cleaned up RepoSync %s/%s for intent %s",
                     reposync["namespace"], reposync["name"], intent_id)
    except Exception:
        logger.warning("RepoSync cleanup failed for %s (non-fatal)", intent_id, exc_info=True)


# Lazy singleton for uvicorn — avoids side effects on import
_default_app: FastAPI | None = None


def get_app() -> FastAPI:
    """Get or create the default app instance (for uvicorn)."""
    global _default_app
    if _default_app is None:
        _default_app = create_app()
    return _default_app
