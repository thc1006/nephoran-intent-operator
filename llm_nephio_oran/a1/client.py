"""A1 Policy Management client for O-RAN SC Near-RT RIC A1 Mediator.

REST client for the /A1-P/v2/ API. Handles policy type registration,
policy instance CRUD, and status queries.

API reference: https://docs.o-ran-sc.org/projects/o-ran-sc-ric-plt-a1/en/latest/user-guide-api.html

Note: xApps receive policies via RMR (msg type 20010), not HTTP.
This client is for the northbound API (SMO/rApp → A1 Mediator).
"""
from __future__ import annotations

import json
import logging
import os
from typing import Any

import requests

from llm_nephio_oran.a1.policy_types import A1PolicyType, POLICY_TYPES

logger = logging.getLogger(__name__)

A1_BASE_URL = os.getenv(
    "A1_MEDIATOR_URL",
    "http://service-ricplt-a1mediator-http.ricplt.svc.cluster.local:10000",
)
A1_TIMEOUT = int(os.getenv("A1_TIMEOUT", "10"))


class A1Client:
    """REST client for A1 Mediator /A1-P/v2/ API."""

    def __init__(self, base_url: str | None = None, timeout: int | None = None):
        self._base = (base_url or A1_BASE_URL).rstrip("/")
        self._timeout = timeout if timeout is not None else A1_TIMEOUT

    def _url(self, path: str) -> str:
        return f"{self._base}/A1-P/v2{path}"

    # ── Health ─────────────────────────────────────────────────────────

    def healthcheck(self) -> bool:
        """Check A1 Mediator health."""
        try:
            resp = requests.get(self._url("/healthcheck"), timeout=self._timeout)
            return resp.status_code == 200
        except requests.RequestException:
            return False

    # ── Policy Types ───────────────────────────────────────────────────

    def list_policy_types(self) -> list[int]:
        """List registered policy type IDs."""
        resp = requests.get(self._url("/policytypes"), timeout=self._timeout)
        resp.raise_for_status()
        return resp.json()

    def get_policy_type(self, policy_type_id: int) -> dict[str, Any]:
        """Get a policy type definition."""
        resp = requests.get(
            self._url(f"/policytypes/{policy_type_id}"), timeout=self._timeout,
        )
        resp.raise_for_status()
        return resp.json()

    def register_policy_type(self, pt: A1PolicyType) -> None:
        """Register a policy type (PUT). Idempotent."""
        body = {
            "name": pt.name,
            "description": pt.description,
            "policy_type_id": pt.policy_type_id,
            "create_schema": pt.create_schema,
        }
        resp = requests.put(
            self._url(f"/policytypes/{pt.policy_type_id}"),
            json=body,
            timeout=self._timeout,
        )
        resp.raise_for_status()
        logger.info("Registered policy type %d (%s)", pt.policy_type_id, pt.name)

    def delete_policy_type(self, policy_type_id: int) -> None:
        """Delete a policy type (only if no instances exist)."""
        resp = requests.delete(
            self._url(f"/policytypes/{policy_type_id}"), timeout=self._timeout,
        )
        resp.raise_for_status()

    # ── Policy Instances ───────────────────────────────────────────────

    def list_policies(self, policy_type_id: int) -> list[str]:
        """List policy instance IDs for a type."""
        resp = requests.get(
            self._url(f"/policytypes/{policy_type_id}/policies"),
            timeout=self._timeout,
        )
        resp.raise_for_status()
        return resp.json()

    def get_policy(self, policy_type_id: int, policy_id: str) -> dict[str, Any]:
        """Get a policy instance."""
        resp = requests.get(
            self._url(f"/policytypes/{policy_type_id}/policies/{policy_id}"),
            timeout=self._timeout,
        )
        resp.raise_for_status()
        return resp.json()

    def create_policy(
        self,
        policy_type_id: int,
        policy_id: str,
        policy_data: dict[str, Any],
    ) -> None:
        """Create or update a policy instance (PUT). Idempotent.

        The A1 Mediator will forward this to xApps via RMR message type 20010.
        """
        resp = requests.put(
            self._url(f"/policytypes/{policy_type_id}/policies/{policy_id}"),
            json=policy_data,
            timeout=self._timeout,
        )
        resp.raise_for_status()
        logger.info(
            "Created policy %s (type %d): %s",
            policy_id, policy_type_id, json.dumps(policy_data),
        )

    def delete_policy(self, policy_type_id: int, policy_id: str) -> None:
        """Delete a policy instance."""
        resp = requests.delete(
            self._url(f"/policytypes/{policy_type_id}/policies/{policy_id}"),
            timeout=self._timeout,
        )
        resp.raise_for_status()
        logger.info("Deleted policy %s (type %d)", policy_id, policy_type_id)

    def get_policy_status(
        self, policy_type_id: int, policy_id: str,
    ) -> dict[str, Any]:
        """Get policy enforcement status (set by xApp via RMR 20011)."""
        resp = requests.get(
            self._url(f"/policytypes/{policy_type_id}/policies/{policy_id}/status"),
            timeout=self._timeout,
        )
        resp.raise_for_status()
        return resp.json()

    # ── Convenience ────────────────────────────────────────────────────

    def ensure_ts_policy_type(self) -> None:
        """Ensure Traffic Steering policy type 20008 is registered."""
        existing = self.list_policy_types()
        if 20008 not in existing:
            self.register_policy_type(POLICY_TYPES[20008])
        else:
            logger.debug("Policy type 20008 already registered")

    def create_ts_policy(
        self, policy_id: str, threshold: int = 0,
    ) -> None:
        """Create a Traffic Steering policy with the given threshold.

        Args:
            policy_id: Unique policy instance ID.
            threshold: Throughput improvement % required to trigger handover (0-100).
        """
        self.ensure_ts_policy_type()
        self.create_policy(20008, policy_id, {"threshold": threshold})
