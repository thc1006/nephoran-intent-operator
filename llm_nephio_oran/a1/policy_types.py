"""O-RAN SC A1 Policy Type definitions (O-RAN WG2, A1AP v2).

Policy types are registered in the A1 Mediator and consumed by xApps via RMR.
Integer IDs are O-RAN SC convention (not O-RAN Alliance standard).

Reference: https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-a1/en/latest/user-guide-api.html
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass(frozen=True)
class A1PolicyType:
    """A1 policy type definition for registration with A1 Mediator."""
    policy_type_id: int
    name: str
    description: str
    create_schema: dict[str, Any]


# O-RAN SC standard policy type IDs
POLICY_TYPE_TS = A1PolicyType(
    policy_type_id=20008,
    name="tsapolicy",
    description="Traffic Steering policy — threshold for handover decision",
    create_schema={
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "threshold": {
                "type": "integer",
                "default": 0,
                "minimum": 0,
                "maximum": 100,
                "description": "Throughput improvement % required to trigger handover",
            },
        },
        "additionalProperties": False,
    },
)

POLICY_TYPE_ADMISSION = A1PolicyType(
    policy_type_id=20001,
    name="admission_control_policy",
    description="Admission control policy for resource management",
    create_schema={
        "$schema": "http://json-schema.org/draft-07/schema#",
        "type": "object",
        "properties": {
            "scope": {"type": "object", "properties": {"ueId": {"type": "string"}}},
            "qosObjectives": {"type": "object", "properties": {"priorityLevel": {"type": "integer"}}},
        },
        "additionalProperties": False,
    },
)

# Registry for lookup by ID
POLICY_TYPES: dict[int, A1PolicyType] = {
    pt.policy_type_id: pt for pt in [POLICY_TYPE_TS, POLICY_TYPE_ADMISSION]
}
