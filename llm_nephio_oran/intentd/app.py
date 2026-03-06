"""intentd — TMF921-aligned Intent Management API (ADR-007).

Endpoints:
  GET  /healthz                            → health check
  POST /tmf-api/intent/v5/intent           → create intent (triggers pipeline)
  GET  /tmf-api/intent/v5/intent/{id}      → get intent status
  GET  /api/intents                        → list all intents (frontend)
  POST /internal/validate/intent-plan      → validate plan against schema
  POST /internal/pipeline                  → full pipeline (plan→generate→git→PR→Porch)
  GET  /api/porch/packages                 → Porch PackageRevision status
  GET  /api/scale/status                   → current CNF replica counts
"""
from __future__ import annotations

import json
import logging
import os
import tempfile
import traceback
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from llm_nephio_oran.validators.schema_validate import validate_json_instance

logger = logging.getLogger(__name__)

APP_STATE_DIR = Path(os.environ.get("INTENTD_STATE_DIR", ".state"))
APP_STATE_DIR.mkdir(parents=True, exist_ok=True)
INTENTS_FILE = APP_STATE_DIR / "intents.jsonl"

app = FastAPI(title="intentd (TMF921 subset)", version="0.3.0")

# CORS for frontend
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


class CreateIntentRequest(BaseModel):
    payload: dict[str, Any] | None = None
    text: str | None = None
    use_llm: bool = True


class PipelineRequest(BaseModel):
    text: str
    use_llm: bool = True
    dry_run: bool = False
    source: str = "cli"  # cli | web | tmf921


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/tmf-api/intent/v5/intent", status_code=201)
def create_intent(req: CreateIntentRequest) -> dict[str, Any]:
    """Create an intent. If 'text' is provided, run planner first."""
    if req.text:
        from llm_nephio_oran.planner.llm_planner import plan_from_text
        plan = plan_from_text(req.text, use_llm=req.use_llm)
        intent_id = plan["intentId"]
        # Tag source as tmf921 for direct API callers
        if "metadata" not in plan:
            plan["metadata"] = {}
        plan["metadata"]["source"] = "tmf921"
    elif req.payload:
        plan = req.payload
        intent_id = plan.get("intentId") or plan.get("id") or "intent-UNASSIGNED"
    else:
        raise HTTPException(status_code=400, detail="Must provide 'text' or 'payload'")

    rec = {
        "id": intent_id,
        "status": "planned",
        "plan": plan,
        "created_at": datetime.now(timezone.utc).isoformat(),
    }
    with INTENTS_FILE.open("a", encoding="utf-8") as f:
        f.write(json.dumps(rec, ensure_ascii=False) + "\n")

    return {"id": intent_id, "status": "planned", "plan": plan}


@app.get("/tmf-api/intent/v5/intent/{intent_id}")
def get_intent(intent_id: str) -> dict[str, Any]:
    if not INTENTS_FILE.exists():
        raise HTTPException(status_code=404, detail="No intents stored yet")
    with INTENTS_FILE.open("r", encoding="utf-8") as f:
        for line in f:
            obj = json.loads(line)
            if obj.get("id") == intent_id:
                return obj
    raise HTTPException(status_code=404, detail="Intent not found")


@app.post("/internal/validate/intent-plan")
def validate_intent_plan(plan: dict[str, Any]) -> dict[str, Any]:
    validate_json_instance(
        schema_path="schemas/intent-plan.schema.json",
        instance=plan,
    )
    return {"valid": True}


@app.post("/internal/pipeline")
def trigger_pipeline(req: PipelineRequest) -> dict[str, Any]:
    """Full pipeline: NL text → plan → validate → generate → git → PR."""
    # Step 1: Plan
    from llm_nephio_oran.planner.llm_planner import plan_from_text
    plan = plan_from_text(req.text, use_llm=req.use_llm)
    intent_id = plan["intentId"]

    # Tag source so we know where the intent came from (cli/web/tmf921)
    if "metadata" not in plan:
        plan["metadata"] = {}
    plan["metadata"]["source"] = req.source

    # Step 2: Validate
    validate_json_instance("schemas/intent-plan.schema.json", plan)

    # Step 3: Generate packages
    from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages
    out_dir = Path("packages/instances")
    pkg_dir = generate_kpt_packages(plan, out_dir)

    # Step 4: Validate manifests
    from llm_nephio_oran.validators.manifest_validator import validate_package
    manifest_errors = validate_package(pkg_dir)
    if manifest_errors:
        return {
            "id": intent_id,
            "status": "validation_failed",
            "errors": manifest_errors,
        }

    # Step 5: Git + PR (unless dry run)
    result: dict[str, Any] = {
        "id": intent_id,
        "status": "packages_generated",
        "package_dir": str(pkg_dir),
        "plan": plan,
    }

    if not req.dry_run:
        from llm_nephio_oran.gitops.git_ops import push_to_gitops
        git_result = push_to_gitops(pkg_dir, plan)
        result["status"] = "pr_created"
        result["git"] = git_result

    # Step 6: Porch lifecycle (if not dry run)
    porch_result = None
    if not req.dry_run:
        try:
            from llm_nephio_oran.porch.client import PorchClient
            porch = PorchClient()
            pkg_name = f"instances/{intent_id}"
            draft = porch.create_draft(repo="nephoran-packages", package_name=pkg_name)
            draft_name = draft["metadata"]["name"]
            # Push resources into draft
            resources = {}
            for f_path in pkg_dir.rglob("*.yaml"):
                resources[str(f_path.relative_to(pkg_dir))] = f_path.read_text()
            porch.update_resources(draft_name, resources)
            # Propose for review
            porch.propose(draft_name)
            porch_result = {"name": draft_name, "lifecycle": "Proposed", "files": len(resources)}
            result["porch"] = porch_result
            result["status"] = "proposed"
        except Exception as e:
            logger.warning("Porch integration skipped: %s", e)
            porch_result = {"error": str(e)}
            result["porch"] = porch_result

    # Store in ledger
    rec = {
        "id": intent_id,
        "status": result["status"],
        "plan": plan,
        "created_at": datetime.now(timezone.utc).isoformat(),
    }
    if porch_result:
        rec["porch"] = porch_result
    with INTENTS_FILE.open("a", encoding="utf-8") as f:
        f.write(json.dumps(rec, ensure_ascii=False) + "\n")

    return result


# ─── Frontend API endpoints ───────────────────────────────────────────────


@app.get("/api/intents")
def list_intents() -> list[dict[str, Any]]:
    """List all intents (most recent first) for the frontend dashboard."""
    if not INTENTS_FILE.exists():
        return []
    intents = []
    with INTENTS_FILE.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                intents.append(json.loads(line))
    # Deduplicate by id (keep last entry per id)
    seen: dict[str, dict] = {}
    for intent in intents:
        seen[intent["id"]] = intent
    return list(reversed(seen.values()))


@app.get("/api/porch/packages")
def list_porch_packages() -> list[dict[str, Any]]:
    """List Porch PackageRevisions for the frontend."""
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


@app.get("/api/scale/status")
def get_scale_status() -> dict[str, Any]:
    """Get current CNF replica counts from cluster for the scale dashboard."""
    try:
        from kubernetes import client, config
        try:
            config.load_incluster_config()
        except config.ConfigException:
            config.load_kube_config()
        apps_v1 = client.AppsV1Api()

        namespaces = {"oran": "ran", "free5gc": "core", "ricplt": "ric", "sim": "sim"}
        components = []
        for ns, domain in namespaces.items():
            try:
                deps = apps_v1.list_namespaced_deployment(namespace=ns)
                for d in deps.items:
                    components.append({
                        "name": d.metadata.name,
                        "namespace": ns,
                        "domain": domain,
                        "replicas": d.spec.replicas or 0,
                        "ready": d.status.ready_replicas or 0,
                        "available": d.status.available_replicas or 0,
                    })
            except Exception:
                pass
        return {"components": components, "timestamp": datetime.now(timezone.utc).isoformat()}
    except Exception as e:
        logger.warning("Cannot get scale status: %s", e)
        return {"components": [], "error": str(e)}
