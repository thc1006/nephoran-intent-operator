"""LLM-backed intent planner using Ollama (ADR-001, ADR-0004).

Converts natural-language intent text into a schema-valid IntentPlan JSON.
Uses qwen3:14b via Ollama's OpenAI-compatible API.
Falls back to stub_planner on LLM failure.
"""
from __future__ import annotations

import json
import logging
import os
import re
from datetime import datetime, timezone
from typing import Any

import requests

from llm_nephio_oran.validators.schema_validate import validate_json_instance

logger = logging.getLogger(__name__)

OLLAMA_BASE_URL = os.getenv("OLLAMA_BASE_URL", "http://localhost:11434")
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "qwen3:14b")
OLLAMA_TIMEOUT = int(os.getenv("OLLAMA_TIMEOUT", "120"))
OLLAMA_STRUCTURED = os.getenv("OLLAMA_STRUCTURED", "true").lower() == "true"
SCHEMA_PATH = "schemas/intent-plan.schema.json"

# Simplified schema for Ollama structured output (no allOf/if/then/format which Ollama can't handle)
_OLLAMA_FORMAT_SCHEMA: dict[str, Any] = {
    "type": "object",
    "required": ["intentId", "intentType", "actions", "policy"],
    "properties": {
        "intentId": {"type": "string"},
        "intentType": {"type": "string", "enum": ["slice.deploy", "slice.scale", "closedloop.act", "config.update"]},
        "description": {"type": "string"},
        "slice": {
            "type": "object",
            "required": ["sliceType", "name", "site"],
            "properties": {
                "sliceType": {"type": "string", "enum": ["eMBB", "URLLC", "mMTC", "shared"]},
                "name": {"type": "string"},
                "site": {"type": "string", "enum": ["edge01", "edge02", "regional", "central", "lab"]},
            },
        },
        "constraints": {
            "type": "object",
            "properties": {
                "maxReplicas": {"type": "integer"},
                "minReplicas": {"type": "integer"},
                "allowedNamespaces": {"type": "array", "items": {"type": "string"}},
            },
        },
        "actions": {
            "type": "array",
            "items": {
                "type": "object",
                "required": ["kind", "component"],
                "properties": {
                    "kind": {"type": "string", "enum": ["deploy", "scale", "configure", "promote", "rollback"]},
                    "component": {"type": "string", "enum": [
                        "oai-odu", "oai-ocu", "oai-cu-cp", "oai-cu-up",
                        "free5gc-upf", "free5gc-smf", "free5gc-amf",
                        "ric-kpimon", "ric-ts", "sim-e2", "trafficgen",
                    ]},
                    "replicas": {"type": "integer"},
                    "naming": {
                        "type": "object",
                        "properties": {
                            "domain": {"type": "string", "enum": ["ran", "core", "ric", "sim", "obs"]},
                            "site": {"type": "string", "enum": ["edge01", "edge02", "regional", "central", "lab"]},
                            "slice": {"type": "string", "enum": ["embb", "urllc", "mmtc", "shared"]},
                            "instance": {"type": "string"},
                        },
                    },
                },
            },
        },
        "policy": {
            "type": "object",
            "required": ["requireHumanReview", "guardrails"],
            "properties": {
                "requireHumanReview": {"type": "boolean"},
                "guardrails": {
                    "type": "object",
                    "required": ["denyClusterScoped", "denyPrivileged", "denyHostNetwork"],
                    "properties": {
                        "denyClusterScoped": {"type": "boolean"},
                        "denyPrivileged": {"type": "boolean"},
                        "denyHostNetwork": {"type": "boolean"},
                        "denyCRDChanges": {"type": "boolean"},
                        "denyRBACChanges": {"type": "boolean"},
                    },
                },
            },
        },
        "metadata": {
            "type": "object",
            "properties": {
                "createdAt": {"type": "string"},
                "createdBy": {"type": "string"},
                "source": {"type": "string", "enum": ["cli", "web", "tmf921"]},
            },
        },
    },
}

SYSTEM_PROMPT = """\
You are an O-RAN/5G network intent planner. Given a natural-language intent, produce a JSON object conforming EXACTLY to the IntentPlan schema.

## Schema constraints (STRICT)
- intentId: "intent-YYYYMMDD-NNNN" (today's date, sequential number)
- intentType: one of "slice.deploy", "slice.scale", "closedloop.act", "config.update"
- actions[]: array of 1+ action objects, each with:
  - kind: one of "deploy", "scale", "configure", "promote", "rollback"
  - component: one of "oai-odu", "oai-ocu", "oai-cu-cp", "oai-cu-up", "free5gc-upf", "free5gc-smf", "free5gc-amf", "ric-kpimon", "ric-ts", "sim-e2", "trafficgen"
  - replicas: integer 0-200 (REQUIRED if kind is "scale")
  - params: optional object
  - naming: optional object with domain (ran|core|ric|sim|obs), site (edge01|edge02|regional|central|lab), slice (embb|urllc|mmtc|shared), instance (i001-i999)
- slice: object with sliceType (eMBB|URLLC|mMTC|shared), name (lowercase-kebab), site
- constraints: object with minReplicas, maxReplicas (1-200), allowedNamespaces, forbiddenKinds
- policy: object with requireHumanReview (bool), guardrails (denyClusterScoped, denyPrivileged, denyHostNetwork, denyCRDChanges, denyRBACChanges)
- metadata: createdAt (ISO8601), createdBy, source (cli|web|tmf921)

## Component-to-domain mapping
- oai-odu, oai-ocu, oai-cu-cp, oai-cu-up → domain: "ran"
- free5gc-upf, free5gc-smf, free5gc-amf → domain: "core"
- ric-kpimon, ric-ts → domain: "ric"
- sim-e2, trafficgen → domain: "sim"

## Rules
1. If the intent mentions "scale", use intentType "slice.scale" with kind "scale".
2. If the intent mentions "deploy" or "create", use intentType "slice.deploy" with kind "deploy".
3. If the intent mentions "configure" or "update", use intentType "config.update" with kind "configure".
4. If the intent mentions "rollback" or "revert", use intentType "config.update" with kind "rollback".
5. Default site is "lab", default slice is "embb", default instance is "i001".
6. Always set policy.requireHumanReview=true and all guardrails=true for safety.
7. Set maxReplicas to 5 unless explicitly stated (ADR-010 guard rail).
8. If the intent mentions "traffic steering", "handover", "steer traffic", or "load balancing between cells", use intentType "closedloop.act" with kind "configure" and component "ric-ts". Include params with "threshold" (integer 0-100, default 5).

Output ONLY the JSON object. No markdown fences, no explanation, no extra text.
"""

FEW_SHOT_EXAMPLES = [
    {
        "input": "Deploy eMBB slice with KPIMON and traffic generator",
        "output": {
            "intentId": "intent-20260305-0001",
            "intentType": "slice.deploy",
            "description": "Deploy eMBB slice with KPIMON and traffic generator",
            "slice": {"sliceType": "eMBB", "name": "embb-lab-001", "site": "lab"},
            "constraints": {"minReplicas": 1, "maxReplicas": 5, "allowedNamespaces": ["oran", "ric", "sim"]},
            "actions": [
                {"kind": "deploy", "component": "ric-kpimon", "replicas": 1, "naming": {"domain": "ric", "site": "lab", "slice": "embb", "instance": "i001"}},
                {"kind": "deploy", "component": "trafficgen", "replicas": 1, "naming": {"domain": "sim", "site": "lab", "slice": "embb", "instance": "i001"}},
            ],
            "policy": {"requireHumanReview": True, "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True, "denyCRDChanges": True, "denyRBACChanges": True}},
            "metadata": {"createdAt": "2026-03-05T00:00:00Z", "createdBy": "intentctl", "source": "cli"},
        },
    },
    {
        "input": "Scale AMF to 3 replicas",
        "output": {
            "intentId": "intent-20260305-0002",
            "intentType": "slice.scale",
            "description": "Scale AMF to 3 replicas",
            "slice": {"sliceType": "eMBB", "name": "embb-lab-001", "site": "lab"},
            "constraints": {"minReplicas": 1, "maxReplicas": 5, "allowedNamespaces": ["core"]},
            "actions": [
                {"kind": "scale", "component": "free5gc-amf", "replicas": 3, "naming": {"domain": "core", "site": "lab", "slice": "embb", "instance": "i001"}},
            ],
            "policy": {"requireHumanReview": True, "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True, "denyCRDChanges": True, "denyRBACChanges": True}},
            "metadata": {"createdAt": "2026-03-05T00:00:00Z", "createdBy": "intentctl", "source": "cli"},
        },
    },
    {
        "input": "Enable traffic steering with 10% threshold for edge01",
        "output": {
            "intentId": "intent-20260305-0003",
            "intentType": "closedloop.act",
            "description": "Enable traffic steering with 10% threshold for edge01",
            "slice": {"sliceType": "eMBB", "name": "embb-edge01-001", "site": "edge01"},
            "constraints": {"minReplicas": 1, "maxReplicas": 5, "allowedNamespaces": ["ric"]},
            "actions": [
                {"kind": "configure", "component": "ric-ts", "params": {"threshold": 10}, "naming": {"domain": "ric", "site": "edge01", "slice": "embb", "instance": "i001"}},
            ],
            "policy": {"requireHumanReview": True, "guardrails": {"denyClusterScoped": True, "denyPrivileged": True, "denyHostNetwork": True, "denyCRDChanges": True, "denyRBACChanges": True}},
            "metadata": {"createdAt": "2026-03-05T00:00:00Z", "createdBy": "intentctl", "source": "cli"},
        },
    },
]


def _build_prompt(text: str) -> list[dict[str, str]]:
    """Build chat messages with system prompt + few-shot examples + user intent."""
    messages = [{"role": "system", "content": SYSTEM_PROMPT}]
    for ex in FEW_SHOT_EXAMPLES:
        messages.append({"role": "user", "content": ex["input"]})
        messages.append({"role": "assistant", "content": json.dumps(ex["output"], indent=2)})
    messages.append({"role": "user", "content": text})
    return messages


def _extract_json(raw: str) -> dict[str, Any]:
    """Extract JSON from LLM response, handling markdown fences and think tags."""
    raw = raw.strip()
    # Strip Qwen3-style <think>...</think> reasoning blocks if present
    raw = re.sub(r"<think>.*?</think>", "", raw, flags=re.DOTALL).strip()
    if raw.startswith("```"):
        raw = re.sub(r"^```(?:json)?\s*\n?", "", raw)
        raw = re.sub(r"\n?```\s*$", "", raw)
    return json.loads(raw)


_seq_counter = 0

def _next_seq() -> str:
    """Generate a unique 4-digit sequence number based on time."""
    global _seq_counter
    _seq_counter += 1
    # Use seconds-of-day + counter for uniqueness
    now = datetime.now(timezone.utc)
    base = (now.hour * 3600 + now.minute * 60 + now.second) % 10000
    return f"{(base + _seq_counter) % 10000:04d}"


def _fix_metadata(plan: dict[str, Any]) -> dict[str, Any]:
    """Ensure intentId uses today's date with unique seq and metadata has correct timestamps."""
    today = datetime.now(timezone.utc).strftime("%Y%m%d")
    # Always replace intentId with a unique one
    plan["intentId"] = f"intent-{today}-{_next_seq()}"

    if "metadata" not in plan:
        plan["metadata"] = {}
    plan["metadata"]["createdAt"] = datetime.now(timezone.utc).replace(microsecond=0).isoformat() + "Z"
    if "createdBy" not in plan["metadata"]:
        plan["metadata"]["createdBy"] = "intentctl"
    if "source" not in plan["metadata"]:
        plan["metadata"]["source"] = "cli"
    return plan


def _call_structured(text: str) -> dict[str, Any] | None:
    """Call Ollama native API with structured output (JSON schema constraint).

    Returns parsed plan dict, or None on failure.
    """
    messages = _build_prompt(text)
    # Prepend /no_think to user message to disable Qwen3 thinking mode for structured output
    messages[-1] = {"role": "user", "content": f"/no_think\n{messages[-1]['content']}"}

    try:
        resp = requests.post(
            f"{OLLAMA_BASE_URL}/api/chat",
            json={
                "model": OLLAMA_MODEL,
                "messages": messages,
                "format": _OLLAMA_FORMAT_SCHEMA,
                "stream": False,
                "options": {"num_predict": 4096, "temperature": 0.1},
            },
            timeout=OLLAMA_TIMEOUT,
        )
        resp.raise_for_status()
        content = resp.json()["message"]["content"]
        if not content:
            logger.warning("Structured output returned empty content")
            return None
        return json.loads(content)
    except Exception:
        logger.warning("Structured output call failed", exc_info=True)
        return None


def _call_freeform(text: str) -> dict[str, Any] | None:
    """Call Ollama via OpenAI-compatible API (freeform text, then extract JSON).

    Returns parsed plan dict, or None on failure.
    """
    messages = _build_prompt(text)

    try:
        resp = requests.post(
            f"{OLLAMA_BASE_URL}/v1/chat/completions",
            json={
                "model": OLLAMA_MODEL,
                "messages": messages,
                "temperature": 0.1,
                "max_tokens": 4096,
            },
            timeout=OLLAMA_TIMEOUT,
        )
        resp.raise_for_status()
        raw = resp.json()["choices"][0]["message"]["content"]
        logger.debug("LLM raw response: %s", raw[:500])
    except Exception:
        logger.warning("Freeform LLM call failed", exc_info=True)
        return None

    try:
        return _extract_json(raw)
    except json.JSONDecodeError:
        logger.warning("LLM returned invalid JSON in freeform mode")
        return None


def plan_from_text(text: str, use_llm: bool = True) -> dict[str, Any]:
    """Convert natural-language intent to IntentPlan JSON.

    Args:
        text: Natural language intent string.
        use_llm: If True, call Ollama LLM. If False, use deterministic stub.

    Returns:
        Schema-valid IntentPlan dict.

    Raises:
        ValueError: If the plan fails schema validation after all attempts.
    """
    if not use_llm:
        from llm_nephio_oran.planner.stub_planner import plan_from_text as stub_plan
        return stub_plan(text)

    logger.info("Calling Ollama (%s) model=%s structured=%s for: %s",
                OLLAMA_BASE_URL, OLLAMA_MODEL, OLLAMA_STRUCTURED, text[:80])

    plan = None
    if OLLAMA_STRUCTURED:
        plan = _call_structured(text)
        if plan is not None:
            logger.info("Structured output produced plan, validating...")

    if plan is None:
        plan = _call_freeform(text)

    if plan is None:
        logger.error("Both structured and freeform calls failed, falling back to stub")
        from llm_nephio_oran.planner.stub_planner import plan_from_text as stub_plan
        return stub_plan(text)

    plan = _fix_metadata(plan)

    try:
        validate_json_instance(schema_path=SCHEMA_PATH, instance=plan)
        logger.info("LLM plan passed schema validation: %s", plan["intentId"])
        return plan
    except ValueError as e:
        logger.warning("LLM plan failed schema validation: %s — falling back to stub", e)
        from llm_nephio_oran.planner.stub_planner import plan_from_text as stub_plan
        return stub_plan(text)
