"""Tests for A1 Policy Management client."""
from __future__ import annotations

import json
from unittest.mock import MagicMock, patch

import pytest
import requests

from llm_nephio_oran.a1.client import A1Client
from llm_nephio_oran.a1.policy_types import (
    POLICY_TYPE_TS, POLICY_TYPE_ADMISSION, POLICY_TYPES, A1PolicyType,
)


# ── Policy Types ──────────────────────────────────────────────────────


class TestPolicyTypes:
    def test_ts_policy_type_id(self):
        assert POLICY_TYPE_TS.policy_type_id == 20008

    def test_ts_policy_name(self):
        assert POLICY_TYPE_TS.name == "tsapolicy"

    def test_ts_schema_has_threshold(self):
        props = POLICY_TYPE_TS.create_schema["properties"]
        assert "threshold" in props
        assert props["threshold"]["type"] == "integer"
        assert props["threshold"]["minimum"] == 0
        assert props["threshold"]["maximum"] == 100

    def test_ts_schema_no_additional_properties(self):
        assert POLICY_TYPE_TS.create_schema["additionalProperties"] is False

    def test_admission_policy_type_id(self):
        assert POLICY_TYPE_ADMISSION.policy_type_id == 20001

    def test_policy_types_registry(self):
        assert 20008 in POLICY_TYPES
        assert 20001 in POLICY_TYPES
        assert POLICY_TYPES[20008] is POLICY_TYPE_TS

    def test_policy_type_is_frozen(self):
        with pytest.raises(AttributeError):
            POLICY_TYPE_TS.policy_type_id = 99999  # type: ignore


# ── A1Client ──────────────────────────────────────────────────────────


class TestA1Client:
    def setup_method(self):
        self.client = A1Client(base_url="http://a1-test:10000", timeout=5)

    def test_url_construction(self):
        assert self.client._url("/healthcheck") == "http://a1-test:10000/A1-P/v2/healthcheck"
        assert self.client._url("/policytypes") == "http://a1-test:10000/A1-P/v2/policytypes"

    def test_url_strips_trailing_slash(self):
        c = A1Client(base_url="http://a1-test:10000/")
        assert c._url("/healthcheck") == "http://a1-test:10000/A1-P/v2/healthcheck"

    @patch("llm_nephio_oran.a1.client.requests.get")
    def test_healthcheck_ok(self, mock_get):
        mock_get.return_value = MagicMock(status_code=200)
        assert self.client.healthcheck() is True

    @patch("llm_nephio_oran.a1.client.requests.get")
    def test_healthcheck_fail(self, mock_get):
        mock_get.side_effect = requests.RequestException("refused")
        assert self.client.healthcheck() is False

    @patch("llm_nephio_oran.a1.client.requests.get")
    def test_list_policy_types(self, mock_get):
        mock_get.return_value = MagicMock(status_code=200)
        mock_get.return_value.json.return_value = [100, 20008]
        result = self.client.list_policy_types()
        assert result == [100, 20008]
        mock_get.assert_called_once_with(
            "http://a1-test:10000/A1-P/v2/policytypes", timeout=5,
        )

    @patch("llm_nephio_oran.a1.client.requests.put")
    def test_register_policy_type(self, mock_put):
        mock_put.return_value = MagicMock(status_code=201)
        self.client.register_policy_type(POLICY_TYPE_TS)
        call_kwargs = mock_put.call_args
        body = call_kwargs.kwargs["json"]
        assert body["policy_type_id"] == 20008
        assert body["name"] == "tsapolicy"

    @patch("llm_nephio_oran.a1.client.requests.put")
    def test_create_policy(self, mock_put):
        mock_put.return_value = MagicMock(status_code=201)
        self.client.create_policy(20008, "ts-cell-001", {"threshold": 10})
        url = mock_put.call_args.args[0]
        assert "/policytypes/20008/policies/ts-cell-001" in url
        body = mock_put.call_args.kwargs["json"]
        assert body == {"threshold": 10}

    @patch("llm_nephio_oran.a1.client.requests.delete")
    def test_delete_policy(self, mock_delete):
        mock_delete.return_value = MagicMock(status_code=202)
        self.client.delete_policy(20008, "ts-cell-001")
        url = mock_delete.call_args.args[0]
        assert "/policytypes/20008/policies/ts-cell-001" in url

    @patch("llm_nephio_oran.a1.client.requests.get")
    def test_get_policy_status(self, mock_get):
        mock_get.return_value = MagicMock(status_code=200)
        mock_get.return_value.json.return_value = {"instance_status": "ENFORCED"}
        status = self.client.get_policy_status(20008, "ts-cell-001")
        assert status["instance_status"] == "ENFORCED"

    @patch("llm_nephio_oran.a1.client.requests.get")
    @patch("llm_nephio_oran.a1.client.requests.put")
    def test_create_ts_policy_registers_type_first(self, mock_put, mock_get):
        mock_get.return_value = MagicMock(status_code=200)
        mock_get.return_value.json.return_value = [100]  # 20008 not registered
        mock_put.return_value = MagicMock(status_code=201)
        self.client.create_ts_policy("ts-cell-001", threshold=15)
        # Two PUTs: register type + create instance
        assert mock_put.call_count == 2

    @patch("llm_nephio_oran.a1.client.requests.get")
    @patch("llm_nephio_oran.a1.client.requests.put")
    def test_create_ts_policy_skips_registration_if_exists(self, mock_put, mock_get):
        mock_get.return_value = MagicMock(status_code=200)
        mock_get.return_value.json.return_value = [100, 20008]  # already registered
        mock_put.return_value = MagicMock(status_code=201)
        self.client.create_ts_policy("ts-cell-001", threshold=5)
        # One PUT: create instance only
        assert mock_put.call_count == 1

    @patch("llm_nephio_oran.a1.client.requests.get")
    def test_list_policies(self, mock_get):
        mock_get.return_value = MagicMock(status_code=200)
        mock_get.return_value.json.return_value = ["ts-cell-001", "ts-cell-002"]
        result = self.client.list_policies(20008)
        assert len(result) == 2
