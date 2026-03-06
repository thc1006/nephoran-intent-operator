"""Closed-loop Analyze phase — metrics evaluation and scaling recommendations (ADR-010).

Guard rails:
- Max replicas: configurable per component (default 5)
- Cooldown: minimum 5 minutes between same-component actions
- Human approval gate: configurable (default enabled for scale-out > 3)
- Auto rollback if new replicas fail health checks within 2 minutes

Thresholds:
- Scale-out: CPU or memory utilization > 80%
- Scale-in: CPU and memory utilization < 20%
"""
from __future__ import annotations

import logging
import time
from typing import Any

logger = logging.getLogger(__name__)

SCALE_OUT_THRESHOLD = 0.80
SCALE_IN_THRESHOLD = 0.20
HUMAN_APPROVAL_REPLICA_THRESHOLD = 3


class MetricsAnalyzer:
    """Analyzes component metrics and produces scaling recommendations."""

    def __init__(
        self,
        max_replicas: int = 5,
        cooldown_seconds: int = 300,
        scale_out_threshold: float = SCALE_OUT_THRESHOLD,
        scale_in_threshold: float = SCALE_IN_THRESHOLD,
    ):
        self.max_replicas = max_replicas
        self.cooldown_seconds = cooldown_seconds
        self.scale_out_threshold = scale_out_threshold
        self.scale_in_threshold = scale_in_threshold
        self._last_action: dict[str, float] = {}  # component → timestamp

    def record_action(self, component: str, action: str) -> None:
        """Record that an action was taken for cooldown tracking."""
        self._last_action[component] = time.monotonic()
        logger.info("Recorded action %s for %s", action, component)

    def _check_cooldown(self, component: str) -> bool:
        """Return True if cooldown is still active (should NOT act)."""
        last = self._last_action.get(component)
        if last is None:
            return False
        elapsed = time.monotonic() - last
        return elapsed < self.cooldown_seconds

    def analyze(self, metrics: dict[str, Any]) -> dict[str, Any]:
        """Analyze metrics and return a scaling recommendation.

        Args:
            metrics: Dict with keys: namespace, component, cpu_utilization,
                     memory_utilization, current_replicas.

        Returns:
            Dict with: action (scale_out|scale_in|none), recommended_replicas,
                      reason, requires_human_approval.
        """
        component = metrics["component"]
        cpu_util = metrics["cpu_utilization"]
        mem_util = metrics["memory_utilization"]
        current = metrics["current_replicas"]

        # Check cooldown
        if self._check_cooldown(component):
            return {
                "action": "none",
                "recommended_replicas": current,
                "reason": f"Cooldown active for {component} ({self.cooldown_seconds}s)",
                "requires_human_approval": False,
            }

        needs_scale_out = cpu_util > self.scale_out_threshold or mem_util > self.scale_out_threshold
        needs_scale_in = cpu_util < self.scale_in_threshold and mem_util < self.scale_in_threshold

        # Scale-out
        if needs_scale_out:
            if current >= self.max_replicas:
                return {
                    "action": "none",
                    "recommended_replicas": current,
                    "reason": f"Max replicas ({self.max_replicas}) reached for {component}",
                    "requires_human_approval": False,
                }
            new_replicas = min(current + 1, self.max_replicas)
            requires_approval = new_replicas > HUMAN_APPROVAL_REPLICA_THRESHOLD
            reason_parts = []
            if cpu_util > self.scale_out_threshold:
                reason_parts.append(f"CPU {cpu_util:.0%} > {self.scale_out_threshold:.0%}")
            if mem_util > self.scale_out_threshold:
                reason_parts.append(f"Memory {mem_util:.0%} > {self.scale_out_threshold:.0%}")
            return {
                "action": "scale_out",
                "recommended_replicas": new_replicas,
                "reason": f"Scale out {component}: {', '.join(reason_parts)}",
                "requires_human_approval": requires_approval,
            }

        # Scale-in
        if needs_scale_in and current > 1:
            new_replicas = max(current - 1, 1)
            return {
                "action": "scale_in",
                "recommended_replicas": new_replicas,
                "reason": f"Scale in {component}: CPU {cpu_util:.0%}, Memory {mem_util:.0%} both below {self.scale_in_threshold:.0%}",
                "requires_human_approval": False,
            }

        # No action needed
        return {
            "action": "none",
            "recommended_replicas": current,
            "reason": f"Utilization within bounds for {component}",
            "requires_human_approval": False,
        }
