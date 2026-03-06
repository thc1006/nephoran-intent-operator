from __future__ import annotations

from datetime import datetime
from typing import Any, Literal, Optional

from pydantic import BaseModel, Field


IntentType = Literal["slice.deploy", "slice.scale", "closedloop.act", "config.update"]
ActionKind = Literal["deploy", "scale", "configure", "promote", "rollback"]


class SliceTarget(BaseModel):
    clusters: list[str] = Field(default_factory=list)
    namespaces: list[str] = Field(default_factory=list)


class SliceSpec(BaseModel):
    sliceType: Literal["eMBB", "URLLC", "mMTC", "shared"]
    name: str
    site: Literal["edge01", "edge02", "regional", "central", "lab"]
    targets: Optional[SliceTarget] = None


class Guardrails(BaseModel):
    denyClusterScoped: bool = True
    denyPrivileged: bool = True
    denyHostNetwork: bool = True
    denyCRDChanges: bool = True
    denyRBACChanges: bool = True


class Policy(BaseModel):
    requireHumanReview: bool = True
    guardrails: Guardrails = Field(default_factory=Guardrails)


class Action(BaseModel):
    kind: ActionKind
    component: Literal[
        "oai-odu",
        "oai-ocu",
        "oai-cu-cp",
        "oai-cu-up",
        "free5gc-upf",
        "free5gc-smf",
        "free5gc-amf",
        "ric-kpimon",
        "ric-ts",
        "sim-e2",
        "trafficgen",
    ]
    replicas: Optional[int] = None
    params: dict[str, Any] = Field(default_factory=dict)
    naming: dict[str, Any] = Field(default_factory=dict)


class Metadata(BaseModel):
    createdAt: datetime = Field(default_factory=datetime.utcnow)
    createdBy: str = "unknown"
    source: Literal["cli", "web", "tmf921"] = "cli"


class IntentPlan(BaseModel):
    intentId: str
    intentType: IntentType
    actions: list[Action]
    policy: Policy
    description: Optional[str] = None
    slice: Optional[SliceSpec] = None
    constraints: dict[str, Any] = Field(default_factory=dict)
    metadata: Metadata = Field(default_factory=Metadata)
