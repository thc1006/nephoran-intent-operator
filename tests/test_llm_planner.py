"""Tests for llm_nephio_oran.planner.llm_planner (ADR-001, ADR-0004)."""
from __future__ import annotations

import json
from unittest.mock import patch, MagicMock

import pytest

from llm_nephio_oran.planner.llm_planner import (
    plan_from_text,
    _extract_json,
    _fix_metadata,
    _build_prompt,
)
from llm_nephio_oran.validators.schema_validate import validate_json_instance


SCHEMA_PATH = "schemas/intent-plan.schema.json"


class TestExtractJson:
    def test_plain_json(self):
        result = _extract_json('{"key": "value"}')
        assert result == {"key": "value"}

    def test_fenced_json(self):
        result = _extract_json('```json\n{"key": "value"}\n```')
        assert result == {"key": "value"}

    def test_fenced_no_lang(self):
        result = _extract_json('```\n{"key": "value"}\n```')
        assert result == {"key": "value"}

    def test_invalid_json_raises(self):
        with pytest.raises(json.JSONDecodeError):
            _extract_json("not valid json")


class TestFixMetadata:
    def test_assigns_unique_intent_id(self):
        plan = {"intentId": "intent-20260101-0001", "metadata": {}}
        result = _fix_metadata(plan)
        assert result["intentId"].startswith("intent-2026")
        assert result["intentId"] != "intent-20260101-0001"

    def test_sets_created_at(self):
        plan = {"intentId": "intent-20260101-0001"}
        result = _fix_metadata(plan)
        assert result["metadata"]["createdAt"].endswith("Z")

    def test_preserves_existing_created_by(self):
        plan = {"intentId": "x", "metadata": {"createdBy": "custom"}}
        result = _fix_metadata(plan)
        assert result["metadata"]["createdBy"] == "custom"


class TestBuildPrompt:
    def test_contains_system_prompt(self):
        messages = _build_prompt("Deploy eMBB slice")
        assert messages[0]["role"] == "system"
        assert "O-RAN" in messages[0]["content"]

    def test_contains_few_shot_examples(self):
        messages = _build_prompt("test")
        # System + 2 examples (user+assistant each) + final user = 6
        assert len(messages) == 6

    def test_user_intent_is_last(self):
        messages = _build_prompt("my intent text")
        assert messages[-1]["role"] == "user"
        assert messages[-1]["content"] == "my intent text"


class TestPlanFromText:
    def test_stub_mode_returns_valid_plan(self):
        plan = plan_from_text("Deploy eMBB slice", use_llm=False)
        validate_json_instance(SCHEMA_PATH, plan)
        assert plan["intentType"] == "slice.deploy"

    def test_stub_mode_scale_intent(self):
        plan = plan_from_text("Scale AMF to 3", use_llm=False)
        validate_json_instance(SCHEMA_PATH, plan)
        assert plan["intentType"] == "slice.scale"

    def test_llm_mode_with_mock(self, valid_plan, mock_ollama_response):
        """Mock Ollama and verify LLM path produces valid plan."""
        mock_resp = mock_ollama_response(valid_plan)
        with patch("llm_nephio_oran.planner.llm_planner.requests.post", return_value=mock_resp):
            plan = plan_from_text("Deploy eMBB slice", use_llm=True)
        validate_json_instance(SCHEMA_PATH, plan)

    def test_llm_failure_falls_back_to_stub(self):
        """When LLM call fails, should fall back to stub planner."""
        with patch("llm_nephio_oran.planner.llm_planner.requests.post", side_effect=Exception("connection refused")):
            plan = plan_from_text("Deploy eMBB slice", use_llm=True)
        # Should still return a valid plan (from stub)
        validate_json_instance(SCHEMA_PATH, plan)

    def test_llm_invalid_json_falls_back(self, mock_ollama_response):
        """When LLM returns invalid JSON, should fall back to stub."""
        mock_resp = MagicMock()
        mock_resp.raise_for_status = MagicMock()
        mock_resp.json.return_value = {"choices": [{"message": {"content": "not json {{"}}]}
        with patch("llm_nephio_oran.planner.llm_planner.requests.post", return_value=mock_resp):
            plan = plan_from_text("Deploy eMBB slice", use_llm=True)
        validate_json_instance(SCHEMA_PATH, plan)

    def test_llm_schema_invalid_falls_back(self, mock_ollama_response):
        """When LLM returns JSON that fails schema, should fall back to stub."""
        bad_plan = {"intentId": "bad", "actions": []}  # Invalid
        mock_resp = mock_ollama_response(bad_plan)
        with patch("llm_nephio_oran.planner.llm_planner.requests.post", return_value=mock_resp):
            plan = plan_from_text("Deploy eMBB slice", use_llm=True)
        # Should be a valid stub plan
        validate_json_instance(SCHEMA_PATH, plan)
