"""Regression tests for bugs found during deep code review.

Each test is named after the bug ID and proves:
  1. The bug existed (would fail without the fix)
  2. The fix resolves it

Bug IDs:
  C1: metrics_text() and cache data inconsistency
  C2: ensure_reposync() missing RoleBinding creation
  C3: Pipeline APPLIED step doesn't verify intent commit
  C4: CronJob pod_spec extraction wrong in manifest_validator
  M1: Rego CRD produces duplicate violations (fixed by removing denyCRDChanges rule)
  M2: Python doesn't deny namespace-scoped Role/RoleBinding
  M4: KpimonQuery inconsistent cache freshness
  M5: delete_intent error message uses stale state (TOCTOU)
"""
from __future__ import annotations

import re
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from llm_nephio_oran.e2sim.kpm_simulator import CellConfig, E2KpmSimulator, KpmSnapshot
from llm_nephio_oran.kpimon.bridge import KpimonBridge, KpimonQuery
from llm_nephio_oran.validators.manifest_validator import (
    CLUSTER_SCOPED_KINDS,
    DENIED_RBAC_KINDS,
    validate_package,
)


class TestC1MetricsTextCacheConsistency:
    """C1: metrics_text() must use the SAME snapshots as the cache.

    Before fix: metrics_text() called sim.generate() via collect(), then
    to_prometheus() internally called sim.generate() AGAIN with different random
    values, so the Prometheus text didn't match cached snapshots.
    """

    def test_prometheus_values_match_cache(self):
        """Prove Prometheus text values equal cached snapshot values."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="cell-001", gnb_id="gnb-edge01", slices=["embb"])],
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)

        # Get Prometheus text (this also populates cache)
        text = bridge.metrics_text()

        # Now get summary from cache — these MUST match
        summary = bridge.summary_by_cell()
        cached_dl = summary["cell-001"]["total_dl_throughput"]

        # Extract the actual DL throughput from Prometheus text
        match = re.search(
            r'e2_kpm_dl_throughput_mbps\{cell_id="cell-001".*?\}\s+([0-9.]+)',
            text,
        )
        assert match, "DL throughput not found in Prometheus text"
        prom_dl = float(match.group(1))

        assert prom_dl == cached_dl, (
            f"Prometheus DL={prom_dl} != cache DL={cached_dl} — data inconsistency!"
        )

    def test_two_consecutive_calls_same_cache(self):
        """After metrics_text(), summary uses same cached data (not regenerated)."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="c1", gnb_id="g1", slices=["embb"])],
            seed=99,
        )
        bridge = KpimonBridge(simulator=sim)
        bridge.metrics_text()

        # summary_by_cell uses _ensure_cache (returns existing cache)
        s1 = bridge.summary_by_cell()
        # Call again — should be same data since we didn't collect() in between
        s2 = bridge.summary_by_cell()

        assert s1["c1"]["total_dl_throughput"] == s2["c1"]["total_dl_throughput"]
        assert s1["c1"]["total_ues"] == s2["c1"]["total_ues"]

    def test_to_prometheus_with_explicit_snapshots(self):
        """to_prometheus(snapshots=...) must use those exact snapshots, not generate new ones."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="c1", gnb_id="g1", slices=["embb"])],
            seed=1,
        )
        known = [KpmSnapshot(
            cell_id="c1", gnb_id="g1", slice_id="embb",
            dl_throughput_mbps=999.9, ul_throughput_mbps=0.0,
            prb_usage_pct=0.0, active_ues=0,
            dl_prb_used=0, ul_prb_used=0,
            cqi_average=1.0, rsrp_average=-100.0,
        )]
        text = sim.to_prometheus(snapshots=known)
        assert "999.9" in text, "to_prometheus ignored explicit snapshots"

    def test_to_prometheus_without_snapshots_generates_new(self):
        """to_prometheus() without args still works (backward compatible)."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="c1", gnb_id="g1", slices=["embb"])],
            seed=1,
        )
        text = sim.to_prometheus()
        assert "e2_kpm_dl_throughput_mbps" in text


class TestC2EnsureReposyncRoleBinding:
    """C2: ensure_reposync() must create RoleBinding before RepoSync.

    Before fix: only docs[1] (RepoSync) was created; docs[0] (RoleBinding) was
    silently discarded. The namespace reconciler would fail with permission denied.
    """

    @pytest.fixture
    def mock_k8s(self):
        with patch("llm_nephio_oran.configsync.client.client") as mock_client, \
             patch("llm_nephio_oran.configsync.client.config"):
            mock_custom = MagicMock()
            mock_rbac = MagicMock()
            mock_client.CustomObjectsApi.return_value = mock_custom
            mock_client.RbacAuthorizationV1Api.return_value = mock_rbac
            # Properly mock ApiException with status attribute
            class MockApiException(Exception):
                def __init__(self, status=0):
                    self.status = status
                    super().__init__(f"status={status}")
            mock_client.ApiException = MockApiException
            mock_custom._rbac = mock_rbac
            mock_custom._ApiException = MockApiException
            yield mock_custom

    def test_rolebinding_created_before_reposync(self, mock_k8s):
        """RoleBinding must be created BEFORE RepoSync."""
        from llm_nephio_oran.configsync.client import ConfigSyncClient, RepoSyncSpec

        mock_k8s.list_namespaced_custom_object.return_value = {"items": []}
        mock_k8s.create_namespaced_custom_object.return_value = {"metadata": {"name": "s"}}

        c = ConfigSyncClient()
        spec = RepoSyncSpec(
            name="intent-sync", namespace="oran",
            repo="http://gitea/repo", branch="main", directory="d",
        )
        c.ensure_reposync(spec)

        # RoleBinding MUST have been created
        rbac = mock_k8s._rbac
        assert rbac.create_namespaced_role_binding.called, \
            "RoleBinding was NOT created — reconciler will fail with permission denied"

        # Verify it was created in the right namespace with the right body
        call_kwargs = rbac.create_namespaced_role_binding.call_args.kwargs
        assert call_kwargs["namespace"] == "oran"
        rb_body = call_kwargs["body"]
        assert rb_body["kind"] == "RoleBinding"
        assert rb_body["roleRef"]["name"] == "edit"

    def test_rolebinding_409_is_idempotent(self, mock_k8s):
        """If RoleBinding already exists (409), it should proceed without error."""
        from llm_nephio_oran.configsync.client import ConfigSyncClient, RepoSyncSpec

        mock_k8s.list_namespaced_custom_object.return_value = {"items": []}
        mock_k8s.create_namespaced_custom_object.return_value = {"metadata": {"name": "s"}}

        # RoleBinding creation returns 409 (already exists)
        mock_k8s._rbac.create_namespaced_role_binding.side_effect = \
            mock_k8s._ApiException(status=409)

        c = ConfigSyncClient()
        spec = RepoSyncSpec(
            name="intent-sync", namespace="oran",
            repo="http://gitea/repo", branch="main", directory="d",
        )
        # Should NOT raise — 409 is handled
        result = c.ensure_reposync(spec)
        assert result["metadata"]["name"] == "s"

    def test_rolebinding_other_error_propagates(self, mock_k8s):
        """Non-409 RBAC errors must propagate (e.g., 403 forbidden)."""
        from llm_nephio_oran.configsync.client import ConfigSyncClient, RepoSyncSpec

        mock_k8s.list_namespaced_custom_object.return_value = {"items": []}

        mock_k8s._rbac.create_namespaced_role_binding.side_effect = \
            mock_k8s._ApiException(status=403)

        c = ConfigSyncClient()
        spec = RepoSyncSpec(
            name="intent-sync", namespace="oran",
            repo="http://gitea/repo", branch="main", directory="d",
        )
        with pytest.raises(Exception) as exc_info:
            c.ensure_reposync(spec)
        assert exc_info.value.status == 403

    def test_existing_reposync_skips_rolebinding(self, mock_k8s):
        """If RepoSync already exists, RoleBinding should NOT be re-created."""
        from llm_nephio_oran.configsync.client import ConfigSyncClient, RepoSyncSpec

        mock_k8s.list_namespaced_custom_object.return_value = {
            "items": [{"metadata": {"name": "intent-sync"}}],
        }

        c = ConfigSyncClient()
        spec = RepoSyncSpec(
            name="intent-sync", namespace="oran",
            repo="http://gitea/repo", branch="main", directory="d",
        )
        c.ensure_reposync(spec)

        # Neither should be called since RepoSync exists
        assert not mock_k8s._rbac.create_namespaced_role_binding.called
        assert not mock_k8s.create_namespaced_custom_object.called


class TestC4CronJobPodSpecExtraction:
    """C4: manifest_validator must extract CronJob pod spec from
    spec.jobTemplate.spec.template.spec, not spec.template.spec.

    This was a Rego/Python divergence — Rego had correct CronJob handling
    but Python used the Deployment path for all workloads.
    """

    def test_cronjob_privileged_detected(self, tmp_path):
        """A privileged container inside a CronJob must be detected."""
        f = tmp_path / "test.yaml"
        f.write_text("""\
apiVersion: batch/v1
kind: CronJob
metadata:
  name: ran-oai-odu-edge01-embb-i001
  labels:
    oran.ai/intent-id: "intent-20260306-0001"
    oran.ai/component: "oai-odu"
    app.kubernetes.io/managed-by: nephio
    app.kubernetes.io/part-of: llm-nephio-oran
spec:
  schedule: "0 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: odu
            securityContext:
              privileged: true
""")
        errors = validate_package(tmp_path)
        privileged_errors = [e for e in errors if "privileged" in e]
        assert len(privileged_errors) >= 1, (
            f"CronJob with privileged container was NOT detected! Errors: {errors}"
        )

    def test_cronjob_hostnetwork_detected(self, tmp_path):
        """hostNetwork inside a CronJob must be detected."""
        f = tmp_path / "test.yaml"
        f.write_text("""\
apiVersion: batch/v1
kind: CronJob
metadata:
  name: ran-oai-odu-edge01-embb-i001
  labels:
    oran.ai/intent-id: "intent-20260306-0001"
    oran.ai/component: "oai-odu"
    app.kubernetes.io/managed-by: nephio
    app.kubernetes.io/part-of: llm-nephio-oran
spec:
  schedule: "0 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          hostNetwork: true
          containers:
          - name: odu
""")
        errors = validate_package(tmp_path)
        hostnet_errors = [e for e in errors if "hostNetwork" in e]
        assert len(hostnet_errors) >= 1, (
            f"CronJob with hostNetwork was NOT detected! Errors: {errors}"
        )


class TestM1RegoNoDuplicateViolations:
    """M1: CRD should produce exactly ONE violation, not two.

    Before fix: Rego had both denyClusterScoped (matching CRD in cluster_scoped_kinds)
    AND a separate denyCRDChanges rule, producing duplicate messages.
    """

    def test_crd_single_violation_in_python(self, tmp_path):
        """CRD should produce exactly 1 L3-Policy error, not 2."""
        f = tmp_path / "test.yaml"
        f.write_text("""\
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test.example.com
""")
        errors = validate_package(tmp_path)
        l3_errors = [e for e in errors if "[L3-Policy]" in e]
        # Should be exactly 1 (denyClusterScoped), not 2 (denyClusterScoped + denyCRDChanges)
        assert len(l3_errors) == 1, f"Expected 1 L3 error for CRD, got {len(l3_errors)}: {l3_errors}"
        assert "denyClusterScoped" in l3_errors[0]

    def test_rego_no_deny_crd_changes_rule(self):
        """security_guardrails.rego should NOT have a separate denyCRDChanges rule."""
        rego_path = Path(__file__).parent.parent / "src" / "config-validator" / "opa_policies" / "security_guardrails.rego"
        content = rego_path.read_text()
        # Count actual violation rules that match CRD — should be exactly 1 (denyClusterScoped)
        # The string "denyCRDChanges" must not appear as a rule identifier
        lines = [ln.strip() for ln in content.splitlines() if not ln.strip().startswith("#")]
        active_crd_refs = [ln for ln in lines if "denyCRDChanges" in ln]
        assert len(active_crd_refs) == 0, \
            f"denyCRDChanges rule still exists in non-comment lines — will cause duplicate violations: {active_crd_refs}"


class TestM2PythonDeniesNamespaceScopedRbac:
    """M2: Python manifest_validator must deny Role/RoleBinding (namespace-scoped RBAC).

    Before fix: Only CLUSTER_SCOPED_KINDS was checked. Role and RoleBinding
    (namespace-scoped) were allowed through, but Rego denied them.
    """

    def test_role_denied(self, tmp_path):
        f = tmp_path / "test.yaml"
        f.write_text("""\
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-role
  namespace: default
""")
        errors = validate_package(tmp_path)
        rbac_errors = [e for e in errors if "denyRBACChanges" in e]
        assert len(rbac_errors) >= 1, f"Role was NOT denied! Errors: {errors}"

    def test_rolebinding_denied(self, tmp_path):
        f = tmp_path / "test.yaml"
        f.write_text("""\
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: test-binding
  namespace: default
""")
        errors = validate_package(tmp_path)
        rbac_errors = [e for e in errors if "denyRBACChanges" in e]
        assert len(rbac_errors) >= 1, f"RoleBinding was NOT denied! Errors: {errors}"

    def test_denied_rbac_kinds_set_exists(self):
        """DENIED_RBAC_KINDS must exist as a distinct set."""
        assert "Role" in DENIED_RBAC_KINDS
        assert "RoleBinding" in DENIED_RBAC_KINDS
        # These should NOT be in CLUSTER_SCOPED_KINDS (they're namespace-scoped)
        assert "Role" not in CLUSTER_SCOPED_KINDS
        assert "RoleBinding" not in CLUSTER_SCOPED_KINDS


class TestM4QueryCacheConsistency:
    """M4: KpimonQuery methods must use consistent cache.

    Before fix: cell_metrics(), slice_metrics(), etc. each called collect()
    (generating new random data), while summary_by_cell() used _ensure_cache()
    (stale). Consecutive queries saw different data.
    """

    def test_consecutive_queries_return_same_data(self):
        """cell_metrics and slice_metrics must see the same snapshot data."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="c1", gnb_id="g1", slices=["embb", "urllc"])],
            seed=42,
        )
        bridge = KpimonBridge(simulator=sim)
        query = KpimonQuery(bridge=bridge)

        # Populate cache once
        query.refresh()

        # All queries should see the same cached data
        cell = query.cell_metrics("c1")
        embb = query.slice_metrics("embb")
        total = query.total_active_ues()

        # cell_metrics returns all slices for c1
        assert len(cell) == 2  # embb + urllc
        cell_embb = [s for s in cell if s.slice_id == "embb"][0]

        # slice_metrics("embb") should return the same snapshot object
        assert len(embb) == 1
        assert embb[0].dl_throughput_mbps == cell_embb.dl_throughput_mbps
        assert embb[0].active_ues == cell_embb.active_ues

        # Total UEs should be sum of all snapshots
        expected_total = sum(s.active_ues for s in cell)
        assert total == expected_total

    def test_refresh_updates_cache(self):
        """refresh() should produce new data; subsequent queries use that new data."""
        sim = E2KpmSimulator(
            cells=[CellConfig(cell_id="c1", gnb_id="g1", slices=["embb"])],
            # NO seed — each generate() call produces different results
        )
        bridge = KpimonBridge(simulator=sim)
        query = KpimonQuery(bridge=bridge)

        snap1 = query.refresh()
        dl1 = snap1[0].dl_throughput_mbps

        # Without refresh, cache should return same value
        same = query.cell_metrics("c1")
        assert same[0].dl_throughput_mbps == dl1

        # After refresh, new data
        snap2 = query.refresh()
        # (may or may not differ due to randomness, but the mechanism is tested)
        new = query.cell_metrics("c1")
        assert new[0].dl_throughput_mbps == snap2[0].dl_throughput_mbps


class TestD1ConfigSyncSANaming:
    """D1: ConfigSync SA name must match Google convention.

    Convention: ns-reconciler-{NS}-{NAME}-{len(NAME)}
    Special case: name == "repo-sync" → ns-reconciler-{NS}
    """

    def test_custom_name_includes_length_suffix(self):
        from llm_nephio_oran.configsync.client import RepoSyncSpec, generate_reposync_yaml
        spec = RepoSyncSpec(
            name="intent-sync", namespace="oran",
            repo="http://gitea/repo", branch="main", directory="d",
        )
        docs = generate_reposync_yaml(spec)
        rolebinding = docs[0]
        sa_name = rolebinding["subjects"][0]["name"]
        # "intent-sync" has 11 characters → suffix must be "-11"
        assert sa_name == "ns-reconciler-oran-intent-sync-11", f"Got: {sa_name}"

    def test_repo_sync_default_name_no_suffix(self):
        from llm_nephio_oran.configsync.client import RepoSyncSpec, generate_reposync_yaml
        spec = RepoSyncSpec(
            name="repo-sync", namespace="free5gc",
            repo="http://gitea/repo", branch="main", directory="d",
        )
        docs = generate_reposync_yaml(spec)
        sa_name = docs[0]["subjects"][0]["name"]
        assert sa_name == "ns-reconciler-free5gc", f"Got: {sa_name}"

    def test_clusterrole_is_edit_not_admin(self):
        from llm_nephio_oran.configsync.client import RepoSyncSpec, generate_reposync_yaml
        spec = RepoSyncSpec(
            name="intent-sync", namespace="oran",
            repo="http://gitea/repo", branch="main", directory="d",
        )
        docs = generate_reposync_yaml(spec)
        role_ref = docs[0]["roleRef"]
        assert role_ref["name"] == "edit", f"Got: {role_ref['name']} (should be 'edit', not 'cluster-admin')"

    def test_sa_namespace_is_config_management_system(self):
        from llm_nephio_oran.configsync.client import RepoSyncSpec, generate_reposync_yaml
        spec = RepoSyncSpec(
            name="test", namespace="oran",
            repo="http://gitea/repo", branch="main", directory="d",
        )
        docs = generate_reposync_yaml(spec)
        sa_ns = docs[0]["subjects"][0]["namespace"]
        assert sa_ns == "config-management-system"
        # "test" has 4 characters
        sa_name = docs[0]["subjects"][0]["name"]
        assert sa_name == "ns-reconciler-oran-test-4"


class TestD2ConfigSyncDeliveryCheck:
    """D2: check_configsync_delivery() must correctly compare commits.

    Extracted pure function from pipeline APPLIED step.
    """

    def test_matching_commits_synced(self):
        from llm_nephio_oran.intentd.app_v2 import check_configsync_delivery
        from llm_nephio_oran.configsync.client import SyncStatus
        status = SyncStatus(synced=True, commit="abc123", errors=[])
        result = check_configsync_delivery("abc123", status)
        assert result["delivered"] is True
        assert result["commit_match"] is True
        assert result["synced"] is True

    def test_mismatched_commits(self):
        from llm_nephio_oran.intentd.app_v2 import check_configsync_delivery
        from llm_nephio_oran.configsync.client import SyncStatus
        status = SyncStatus(synced=True, commit="old_commit", errors=[])
        result = check_configsync_delivery("new_commit", status)
        assert result["delivered"] is False
        assert result["commit_match"] is False

    def test_empty_intent_commit(self):
        from llm_nephio_oran.intentd.app_v2 import check_configsync_delivery
        from llm_nephio_oran.configsync.client import SyncStatus
        status = SyncStatus(synced=True, commit="abc123", errors=[])
        result = check_configsync_delivery("", status)
        assert result["delivered"] is False
        assert result["commit_match"] is False

    def test_sync_errors_prevent_delivery(self):
        from llm_nephio_oran.intentd.app_v2 import check_configsync_delivery
        from llm_nephio_oran.configsync.client import SyncStatus
        status = SyncStatus(synced=False, commit="abc123", errors=["error: timeout"])
        result = check_configsync_delivery("abc123", status)
        assert result["delivered"] is False
        assert result["commit_match"] is True  # commits match but sync has errors
        assert result["errors"] == ["error: timeout"]

    def test_not_synced_yet(self):
        from llm_nephio_oran.intentd.app_v2 import check_configsync_delivery
        from llm_nephio_oran.configsync.client import SyncStatus
        status = SyncStatus(synced=False, commit="abc123", errors=[])
        result = check_configsync_delivery("abc123", status)
        assert result["delivered"] is False  # commit matches but not synced yet


class TestD1bCleanupRepoSync:
    """D1b: Per-intent RepoSync cleanup on terminal states."""

    def test_cleanup_with_reposync_data(self, tmp_path):
        """_cleanup_reposync calls delete when reposync data is present."""
        from llm_nephio_oran.intentd.app_v2 import _cleanup_reposync
        from llm_nephio_oran.intentd.store import IntentStore, IntentState

        store = IntentStore(state_dir=tmp_path)
        rec = store.create("test", "cli")
        intent_id = rec["id"]
        store.transition(intent_id, IntentState.PLANNING)
        store.transition(intent_id, IntentState.VALIDATING)
        store.transition(intent_id, IntentState.GENERATING)
        store.transition(intent_id, IntentState.EXECUTING)
        store.transition(intent_id, IntentState.PROPOSED, data={
            "reposync": {"name": "intent-test", "namespace": "oran", "directory": "d"},
        })
        store.transition(intent_id, IntentState.APPLIED)
        store.transition(intent_id, IntentState.COMPLETED)

        with patch("llm_nephio_oran.configsync.client.config"), \
             patch("llm_nephio_oran.configsync.client.client") as mock_k8s:
            mock_custom = MagicMock()
            mock_rbac = MagicMock()
            mock_k8s.CustomObjectsApi.return_value = mock_custom
            mock_k8s.RbacAuthorizationV1Api.return_value = mock_rbac
            mock_k8s.ApiException = type("ApiException", (Exception,), {"status": 404})
            _cleanup_reposync(store, intent_id)
            mock_custom.delete_namespaced_custom_object.assert_called_once()
            call_kwargs = mock_custom.delete_namespaced_custom_object.call_args.kwargs
            assert call_kwargs["name"] == "intent-test"
            assert call_kwargs["namespace"] == "oran"

    def test_cleanup_skips_when_no_reposync(self, tmp_path):
        """_cleanup_reposync is a no-op when reposync data is absent."""
        from llm_nephio_oran.intentd.app_v2 import _cleanup_reposync
        from llm_nephio_oran.intentd.store import IntentStore, IntentState

        store = IntentStore(state_dir=tmp_path)
        rec = store.create("test", "cli")
        intent_id = rec["id"]
        store.transition(intent_id, IntentState.PLANNING)
        store.transition(intent_id, IntentState.FAILED)

        # No reposync data → early return, no K8s API calls
        # Verify no exception is raised (function guards on empty reposync)
        _cleanup_reposync(store, intent_id)
        # If we got here, no import of ConfigSyncClient was attempted

    def test_cleanup_skips_when_reposync_had_error(self, tmp_path):
        """_cleanup_reposync is a no-op when reposync creation had an error."""
        from llm_nephio_oran.intentd.app_v2 import _cleanup_reposync
        from llm_nephio_oran.intentd.store import IntentStore, IntentState

        store = IntentStore(state_dir=tmp_path)
        rec = store.create("test", "cli")
        intent_id = rec["id"]
        store.transition(intent_id, IntentState.PLANNING)
        store.transition(intent_id, IntentState.VALIDATING)
        store.transition(intent_id, IntentState.GENERATING)
        store.transition(intent_id, IntentState.EXECUTING)
        store.transition(intent_id, IntentState.PROPOSED, data={
            "reposync": {"error": "connection refused"},
        })
        store.transition(intent_id, IntentState.APPLIED)
        store.transition(intent_id, IntentState.COMPLETED)

        # reposync has "error" key → early return, no delete attempted
        _cleanup_reposync(store, intent_id)

    def test_reposync_field_stored_in_intent(self, tmp_path):
        """reposync data is persisted in intent record through transitions."""
        from llm_nephio_oran.intentd.store import IntentStore, IntentState

        store = IntentStore(state_dir=tmp_path)
        rec = store.create("test", "cli")
        intent_id = rec["id"]
        store.transition(intent_id, IntentState.PLANNING)
        store.transition(intent_id, IntentState.VALIDATING)
        store.transition(intent_id, IntentState.GENERATING)
        store.transition(intent_id, IntentState.EXECUTING)
        store.transition(intent_id, IntentState.PROPOSED, data={
            "reposync": {"name": "intent-x", "namespace": "oran", "directory": "d"},
        })

        rec = store.get(intent_id)
        assert rec["reposync"]["name"] == "intent-x"
        assert rec["reposync"]["namespace"] == "oran"

    def test_reposync_field_in_report(self, tmp_path):
        """TMF921 IntentReport includes reposync data."""
        from llm_nephio_oran.intentd.store import IntentStore, IntentState

        store = IntentStore(state_dir=tmp_path)
        rec = store.create("test", "cli")
        intent_id = rec["id"]
        store.transition(intent_id, IntentState.PLANNING)
        store.transition(intent_id, IntentState.VALIDATING)
        store.transition(intent_id, IntentState.GENERATING)
        store.transition(intent_id, IntentState.EXECUTING)
        store.transition(intent_id, IntentState.PROPOSED, data={
            "reposync": {"name": "intent-x", "namespace": "oran"},
        })

        report = store.build_report(intent_id)
        assert report["reposync"]["name"] == "intent-x"
