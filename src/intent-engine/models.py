"""Pydantic models aligned with TMF 921."""
import uuid
from datetime import datetime
from enum import Enum
from pydantic import BaseModel, Field


class IntentCategory(str, Enum):
    DEPLOY = "deploy"
    SCALE_OUT = "scale_out"
    SCALE_IN = "scale_in"
    UPDATE = "update"
    DELETE = "delete"
    OBSERVE = "observe"


class IntentStatus(str, Enum):
    PENDING = "pending"
    VALIDATING = "validating"
    APPROVED = "approved"
    EXECUTING = "executing"
    COMPLETED = "completed"
    FAILED = "failed"
    ROLLED_BACK = "rolled_back"


class IntentRequest(BaseModel):
    original_text: str
    priority: str = "medium"
    approval_required: bool = True


class IntentResponse(BaseModel):
    intent_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    status: IntentStatus = IntentStatus.PENDING
    created_at: datetime = Field(default_factory=datetime.utcnow)
    original_text: str
    plan: dict | None = None
    errors: list[str] = []
