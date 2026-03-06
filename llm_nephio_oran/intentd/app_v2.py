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
import threading
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
    _store = IntentStore(state_dir=store_dir or STORE_DIR)

    app = FastAPI(title="intentd v2 (TMF921 async)", version="2.0.0")
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # ── TMF921 endpoints ───────────────────────────────────────────────

    @app.get("/healthz")
    def healthz() -> dict[str, str]:
        return {"status": "ok"}

    @app.post("/tmf-api/intent/v5/intent", status_code=201)
    def create_intent(req: CreateIntentRequest) -> dict[str, Any]:
        """Create intent → acknowledged immediately, pipeline runs async."""
        rec = _store.create(expression=req.expression, source=req.source)

        # Launch pipeline in background thread
        thread = threading.Thread(
            target=_run_pipeline,
            args=(_store, rec["id"], req.use_llm, req.dry_run),
            daemon=True,
        )
        thread.start()

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

    _k8s_apps: list[Any] = []  # cached K8s client [AppsV1Api]

    def _get_apps_v1() -> Any:
        if not _k8s_apps:
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

    try:
        # Step 1: Planning
        store.transition(intent_id, IntentState.PLANNING)
        rec = store.get(intent_id)
        plan = _plan_from_text(rec["expression"], use_llm=use_llm)

        # Step 2: Validating
        store.transition(intent_id, IntentState.VALIDATING, data={"plan": plan})
        from llm_nephio_oran.validators.schema_validate import validate_json_instance
        validate_json_instance("schemas/intent-plan.schema.json", plan)

        # Step 3: Generating
        store.transition(intent_id, IntentState.GENERATING)
        from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages
        out_dir = Path("packages/instances")
        pkg_dir = generate_kpt_packages(plan, out_dir)

        # Step 3b: Validate manifests
        from llm_nephio_oran.validators.manifest_validator import validate_package
        manifest_errors = validate_package(pkg_dir)
        if manifest_errors:
            store.transition(intent_id, IntentState.FAILED, data={
                "errors": manifest_errors,
                "report": {"stage": "manifest_validation", "detail": manifest_errors},
            })
            return

        if dry_run:
            # Dry run: skip git+porch, mark completed
            store.transition(intent_id, IntentState.EXECUTING, data={"package_dir": str(pkg_dir)})
            store.transition(intent_id, IntentState.PROPOSED)
            store.transition(intent_id, IntentState.APPLIED)
            store.transition(intent_id, IntentState.COMPLETED, data={
                "report": {"mode": "dry_run", "package_dir": str(pkg_dir)},
            })
            return

        # Step 4: Executing (git push + PR)
        store.transition(intent_id, IntentState.EXECUTING, data={"package_dir": str(pkg_dir)})
        from llm_nephio_oran.gitops.git_ops import push_to_gitops
        raw_git = push_to_gitops(pkg_dir, plan)
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
        except Exception as e:
            logger.warning("Porch integration failed: %s", e)
            porch_data = {"error": str(e)}
            porch_failed = True

        store.transition(intent_id, IntentState.PROPOSED, data={
            "git": git_result,
            "porch": porch_data,
        })

        # Steps 6-7: applied + completed
        # If Porch failed, still complete but record warning in report
        store.transition(intent_id, IntentState.APPLIED)
        store.transition(intent_id, IntentState.COMPLETED, data={
            "report": {
                "git": git_result,
                "porch": porch_data,
                "package_dir": str(pkg_dir),
                "porch_failed": porch_failed,
            },
        })

    except Exception as e:
        logger.exception("Pipeline failed for %s", intent_id)
        try:
            store.transition(intent_id, IntentState.FAILED, data={
                "errors": [str(e)],
                "report": {"stage": "pipeline", "error": str(e)},
            })
        except Exception:
            logger.exception("Failed to record failure for %s", intent_id)


# Lazy singleton for uvicorn — avoids side effects on import
_default_app: FastAPI | None = None


def get_app() -> FastAPI:
    """Get or create the default app instance (for uvicorn)."""
    global _default_app
    if _default_app is None:
        _default_app = create_app()
    return _default_app
