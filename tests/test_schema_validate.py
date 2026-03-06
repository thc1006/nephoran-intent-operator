"""Tests for llm_nephio_oran.validators.schema_validate (ADR-008 Layer 1)."""
from __future__ import annotations

import json
from pathlib import Path

import pytest

from llm_nephio_oran.validators.schema_validate import validate_json_instance, validate_json_file


class TestValidateJsonInstance:
    """Test schema validation against intent-plan.schema.json."""

    def test_valid_plan_passes(self, valid_plan, schema_path):
        # Should not raise
        validate_json_instance(schema_path, valid_plan)

    def test_valid_scale_plan_passes(self, valid_plan_scale, schema_path):
        validate_json_instance(schema_path, valid_plan_scale)

    def test_missing_intentId_fails(self, valid_plan, schema_path):
        del valid_plan["intentId"]
        with pytest.raises(ValueError, match="intentId"):
            validate_json_instance(schema_path, valid_plan)

    def test_missing_actions_fails(self, valid_plan, schema_path):
        del valid_plan["actions"]
        with pytest.raises(ValueError, match="actions"):
            validate_json_instance(schema_path, valid_plan)

    def test_missing_policy_fails(self, valid_plan, schema_path):
        del valid_plan["policy"]
        with pytest.raises(ValueError, match="policy"):
            validate_json_instance(schema_path, valid_plan)

    def test_invalid_intentId_pattern_fails(self, valid_plan, schema_path):
        valid_plan["intentId"] = "bad-id"
        with pytest.raises(ValueError):
            validate_json_instance(schema_path, valid_plan)

    def test_invalid_intentType_fails(self, valid_plan, schema_path):
        valid_plan["intentType"] = "invalid.type"
        with pytest.raises(ValueError):
            validate_json_instance(schema_path, valid_plan)

    def test_invalid_component_fails(self, valid_plan, schema_path):
        valid_plan["actions"][0]["component"] = "not-a-valid-component"
        with pytest.raises(ValueError):
            validate_json_instance(schema_path, valid_plan)

    def test_scale_without_replicas_fails(self, valid_plan, schema_path):
        valid_plan["actions"] = [{"kind": "scale", "component": "free5gc-amf"}]
        with pytest.raises(ValueError, match="replicas"):
            validate_json_instance(schema_path, valid_plan)

    def test_deploy_without_replicas_passes(self, valid_plan, schema_path):
        """deploy kind does NOT require replicas."""
        valid_plan["actions"] = [{"kind": "deploy", "component": "ric-kpimon"}]
        validate_json_instance(schema_path, valid_plan)

    def test_empty_actions_fails(self, valid_plan, schema_path):
        valid_plan["actions"] = []
        with pytest.raises(ValueError):
            validate_json_instance(schema_path, valid_plan)

    def test_additional_properties_rejected(self, valid_plan, schema_path):
        valid_plan["unknownField"] = "should fail"
        with pytest.raises(ValueError):
            validate_json_instance(schema_path, valid_plan)

    def test_all_11_components_valid(self, valid_plan, schema_path):
        """Verify all 11 allowlisted components pass validation."""
        components = [
            "oai-odu", "oai-ocu", "oai-cu-cp", "oai-cu-up",
            "free5gc-upf", "free5gc-smf", "free5gc-amf",
            "ric-kpimon", "ric-ts", "sim-e2", "trafficgen",
        ]
        for comp in components:
            valid_plan["actions"] = [{"kind": "deploy", "component": comp}]
            validate_json_instance(schema_path, valid_plan)

    def test_all_intent_types_valid(self, valid_plan, schema_path):
        for it in ["slice.deploy", "slice.scale", "closedloop.act", "config.update"]:
            valid_plan["intentType"] = it
            validate_json_instance(schema_path, valid_plan)

    def test_replicas_max_200(self, valid_plan, schema_path):
        valid_plan["actions"] = [{"kind": "scale", "component": "free5gc-amf", "replicas": 201}]
        with pytest.raises(ValueError):
            validate_json_instance(schema_path, valid_plan)


class TestValidateJsonFile:
    def test_valid_file(self, valid_plan, schema_path, tmp_path):
        f = tmp_path / "test.json"
        f.write_text(json.dumps(valid_plan))
        validate_json_file(schema_path, f)

    def test_invalid_file_raises(self, schema_path, tmp_path):
        f = tmp_path / "bad.json"
        f.write_text(json.dumps({"bad": "data"}))
        with pytest.raises(ValueError):
            validate_json_file(schema_path, f)
