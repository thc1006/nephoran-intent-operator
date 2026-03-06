"""Prometheus-backed observability layer (ADR-010).

Queries Prometheus for K8s pod metrics and component health.
Provides metrics snapshots for the closed-loop controller's Observe phase.
"""
from __future__ import annotations

import logging
import os
from datetime import datetime, timezone
from typing import Any

import requests

logger = logging.getLogger(__name__)

PROMETHEUS_URL = os.getenv("PROMETHEUS_URL", "http://prometheus-kube-prometheus-prometheus.monitoring:9090")
PROMETHEUS_TIMEOUT = int(os.getenv("PROMETHEUS_TIMEOUT", "10"))


def _prom_query(query: str) -> list[dict[str, Any]]:
    """Execute a PromQL instant query."""
    try:
        resp = requests.get(
            f"{PROMETHEUS_URL}/api/v1/query",
            params={"query": query},
            timeout=PROMETHEUS_TIMEOUT,
        )
        resp.raise_for_status()
        data = resp.json()
        if data["status"] != "success":
            logger.warning("Prometheus query failed: %s", data.get("error"))
            return []
        return data["data"]["result"]
    except requests.RequestException:
        logger.exception("Prometheus query error for: %s", query)
        return []


def _scalar(results: list[dict[str, Any]], default: float = 0.0) -> float:
    """Extract first scalar value from PromQL result."""
    if results and results[0].get("value"):
        try:
            return float(results[0]["value"][1])
        except (IndexError, ValueError):
            pass
    return default


def snapshot_metrics(namespace: str = "", component: str = "") -> dict[str, Any]:
    """Collect a metrics snapshot from Prometheus.

    Args:
        namespace: Filter by K8s namespace (optional).
        component: Filter by oran.ai/component label (optional).

    Returns:
        Dict with timestamp, cpu_usage, memory_usage, pod_count, component details.
    """
    ns_filter = f'namespace="{namespace}"' if namespace else ""
    comp_filter = f'label_oran_ai_component="{component}"' if component else ""
    filters = ", ".join(f for f in [ns_filter, comp_filter] if f)

    # Pod CPU usage (cores)
    cpu_query = f'sum(rate(container_cpu_usage_seconds_total{{{filters}}}[5m]))'
    cpu_results = _prom_query(cpu_query)
    cpu_usage = _scalar(cpu_results)

    # Pod memory usage (bytes)
    mem_query = f'sum(container_memory_working_set_bytes{{{filters}}})'
    mem_results = _prom_query(mem_query)
    mem_usage = _scalar(mem_results)

    # Pod count
    pod_query = f'count(kube_pod_info{{{ns_filter}}})'
    pod_results = _prom_query(pod_query)
    pod_count = int(_scalar(pod_results))

    # Ready pods
    ready_query = f'count(kube_pod_status_ready{{condition="true", {ns_filter}}})'
    ready_results = _prom_query(ready_query)
    ready_count = int(_scalar(ready_results))

    return {
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "namespace": namespace or "all",
        "component": component or "all",
        "k8s": {
            "cpu_usage_cores": round(cpu_usage, 4),
            "memory_usage_mb": round(mem_usage / 1024 / 1024, 1) if mem_usage else 0.0,
            "pod_count": pod_count,
            "ready_count": ready_count,
        },
    }


def snapshot_all_components() -> dict[str, Any]:
    """Collect metrics for all Nephoran-managed components."""
    components = {
        "ric": {"namespace": "ricplt"},
        "core": {"namespace": "free5gc"},
        "monitoring": {"namespace": "monitoring"},
    }

    result = {
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "components": {},
    }

    for name, cfg in components.items():
        result["components"][name] = snapshot_metrics(namespace=cfg["namespace"])

    return result


def check_scaling_thresholds(
    namespace: str,
    cpu_threshold: float = 0.8,
    memory_threshold: float = 0.8,
) -> dict[str, Any]:
    """Check if any component exceeds scaling thresholds.

    Returns scaling recommendations for the Analyze phase of the closed loop.
    """
    metrics = snapshot_metrics(namespace=namespace)
    recommendations: list[dict[str, str]] = []

    # CPU-based scaling check
    cpu_util_query = (
        f'sum(rate(container_cpu_usage_seconds_total{{namespace="{namespace}"}}[5m])) '
        f'/ sum(kube_pod_container_resource_requests{{namespace="{namespace}", resource="cpu"}})'
    )
    cpu_util_results = _prom_query(cpu_util_query)
    cpu_util = _scalar(cpu_util_results)

    if cpu_util > cpu_threshold:
        recommendations.append({
            "type": "scale_out",
            "reason": f"CPU utilization {cpu_util:.1%} exceeds threshold {cpu_threshold:.0%}",
            "namespace": namespace,
        })

    # Memory-based scaling check
    mem_util_query = (
        f'sum(container_memory_working_set_bytes{{namespace="{namespace}"}}) '
        f'/ sum(kube_pod_container_resource_limits{{namespace="{namespace}", resource="memory"}})'
    )
    mem_util_results = _prom_query(mem_util_query)
    mem_util = _scalar(mem_util_results)

    if mem_util > memory_threshold:
        recommendations.append({
            "type": "scale_out",
            "reason": f"Memory utilization {mem_util:.1%} exceeds threshold {memory_threshold:.0%}",
            "namespace": namespace,
        })

    return {
        "metrics": metrics,
        "cpu_utilization": round(cpu_util, 4),
        "memory_utilization": round(mem_util, 4),
        "thresholds_exceeded": len(recommendations) > 0,
        "recommendations": recommendations,
    }
