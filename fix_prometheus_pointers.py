#!/usr/bin/env python3
"""Fix Prometheus metric fields to use pointer types."""

import re
import os

files_to_fix = [
    ("pkg/chaos/engine.go", [
        ("experimentsTotal    prometheus.CounterVec", "experimentsTotal    *prometheus.CounterVec"),
        ("experimentsDuration prometheus.HistogramVec", "experimentsDuration *prometheus.HistogramVec"),
        ("slaViolations       prometheus.CounterVec", "slaViolations       *prometheus.CounterVec"),
        ("recoveryTime        prometheus.HistogramVec", "recoveryTime        *prometheus.HistogramVec"),
        ("rollbacksTotal      prometheus.CounterVec", "rollbacksTotal      *prometheus.CounterVec"),
    ]),
    ("pkg/controllers/e2nodeset_controller.go", [
        ("nodesTotal      prometheus.GaugeVec", "nodesTotal      *prometheus.GaugeVec"),
        ("nodesReady      prometheus.GaugeVec", "nodesReady      *prometheus.GaugeVec"),
        ("reconcilesTotal prometheus.CounterVec", "reconcilesTotal *prometheus.CounterVec"),
        ("reconcileErrors prometheus.CounterVec", "reconcileErrors *prometheus.CounterVec"),
        ("heartbeatsTotal prometheus.CounterVec", "heartbeatsTotal *prometheus.CounterVec"),
    ]),
    ("pkg/nephio/workload_cluster_registry.go", [
        ("ClusterRegistrations prometheus.CounterVec", "ClusterRegistrations *prometheus.CounterVec"),
        ("ClusterHealth        prometheus.GaugeVec", "ClusterHealth        *prometheus.GaugeVec"),
        ("HealthCheckDuration  prometheus.HistogramVec", "HealthCheckDuration  *prometheus.HistogramVec"),
        ("ClusterCapabilities  prometheus.GaugeVec", "ClusterCapabilities  *prometheus.GaugeVec"),
        ("RegistrationErrors   prometheus.CounterVec", "RegistrationErrors   *prometheus.CounterVec"),
        ("ClusterEvents        prometheus.CounterVec", "ClusterEvents        *prometheus.CounterVec"),
    ]),
    ("pkg/nephio/workflow_orchestrator.go", [
        ("WorkflowExecutions   prometheus.CounterVec", "WorkflowExecutions   *prometheus.CounterVec"),
        ("WorkflowDuration     prometheus.HistogramVec", "WorkflowDuration     *prometheus.HistogramVec"),
        ("WorkflowPhases       prometheus.GaugeVec", "WorkflowPhases       *prometheus.GaugeVec"),
        ("PackageVariants      prometheus.CounterVec", "PackageVariants      *prometheus.CounterVec"),
        ("ClusterDeployments   prometheus.CounterVec", "ClusterDeployments   *prometheus.CounterVec"),
        ("ConfigSyncOperations prometheus.CounterVec", "ConfigSyncOperations *prometheus.CounterVec"),
        ("WorkflowErrors       prometheus.CounterVec", "WorkflowErrors       *prometheus.CounterVec"),
    ]),
    ("pkg/nephio/workflow_engine.go", [
        ("WorkflowRegistrations prometheus.CounterVec", "WorkflowRegistrations *prometheus.CounterVec"),
        ("WorkflowExecutions    prometheus.CounterVec", "WorkflowExecutions    *prometheus.CounterVec"),
        ("ExecutionDuration     prometheus.HistogramVec", "ExecutionDuration     *prometheus.HistogramVec"),
        ("WorkflowErrors        prometheus.CounterVec", "WorkflowErrors        *prometheus.CounterVec"),
    ]),
    ("pkg/nephio/package_catalog.go", [
        ("BlueprintQueries    prometheus.CounterVec", "BlueprintQueries    *prometheus.CounterVec"),
        ("VariantCreations    prometheus.CounterVec", "VariantCreations    *prometheus.CounterVec"),
        ("CatalogOperations   prometheus.CounterVec", "CatalogOperations   *prometheus.CounterVec"),
        ("BlueprintLoadTime   prometheus.HistogramVec", "BlueprintLoadTime   *prometheus.HistogramVec"),
        ("VariantCreationTime prometheus.HistogramVec", "VariantCreationTime *prometheus.HistogramVec"),
        ("CatalogSize         prometheus.GaugeVec", "CatalogSize         *prometheus.GaugeVec"),
    ]),
    ("pkg/optimization/optimization_dashboard.go", [
        ("optimizationCounter  prometheus.CounterVec", "optimizationCounter  *prometheus.CounterVec"),
        ("optimizationDuration prometheus.HistogramVec", "optimizationDuration *prometheus.HistogramVec"),
        ("optimizationGauge    prometheus.GaugeVec", "optimizationGauge    *prometheus.GaugeVec"),
    ]),
    ("pkg/controllers/optimized/performance_metrics.go", [
        ("ReconcileDuration prometheus.HistogramVec", "ReconcileDuration *prometheus.HistogramVec"),
        ("ReconcileTotal    prometheus.CounterVec", "ReconcileTotal    *prometheus.CounterVec"),
        ("ReconcileErrors   prometheus.CounterVec", "ReconcileErrors   *prometheus.CounterVec"),
        ("ReconcileRequeue  prometheus.CounterVec", "ReconcileRequeue  *prometheus.CounterVec"),
        ("BackoffDelay   prometheus.HistogramVec", "BackoffDelay   *prometheus.HistogramVec"),
        ("BackoffRetries prometheus.HistogramVec", "BackoffRetries *prometheus.HistogramVec"),
        ("BackoffResets  prometheus.CounterVec", "BackoffResets  *prometheus.CounterVec"),
        ("StatusBatchSize      prometheus.HistogramVec", "StatusBatchSize      *prometheus.HistogramVec"),
        ("StatusBatchDuration  prometheus.HistogramVec", "StatusBatchDuration  *prometheus.HistogramVec"),
        ("StatusUpdatesQueued  prometheus.CounterVec", "StatusUpdatesQueued  *prometheus.CounterVec"),
        ("StatusUpdatesDropped prometheus.CounterVec", "StatusUpdatesDropped *prometheus.CounterVec"),
        ("StatusUpdatesFailed  prometheus.CounterVec", "StatusUpdatesFailed  *prometheus.CounterVec"),
        ("StatusQueueSize      prometheus.GaugeVec", "StatusQueueSize      *prometheus.GaugeVec"),
        ("ApiCallDuration prometheus.HistogramVec", "ApiCallDuration *prometheus.HistogramVec"),
        ("ApiCallTotal    prometheus.CounterVec", "ApiCallTotal    *prometheus.CounterVec"),
        ("ApiCallErrors   prometheus.CounterVec", "ApiCallErrors   *prometheus.CounterVec"),
        ("ActiveReconcilers prometheus.GaugeVec", "ActiveReconcilers *prometheus.GaugeVec"),
        ("MemoryUsage       prometheus.GaugeVec", "MemoryUsage       *prometheus.GaugeVec"),
        ("GoroutineCount    prometheus.GaugeVec", "GoroutineCount    *prometheus.GaugeVec"),
    ]),
    ("pkg/nephio/configsync_client.go", [
        ("SyncOperations     prometheus.CounterVec", "SyncOperations     *prometheus.CounterVec"),
        ("SyncDuration       prometheus.HistogramVec", "SyncDuration       *prometheus.HistogramVec"),
        ("SyncErrors         prometheus.CounterVec", "SyncErrors         *prometheus.CounterVec"),
        ("GitOperations      prometheus.CounterVec", "GitOperations      *prometheus.CounterVec"),
        ("RepositoryHealth   prometheus.GaugeVec", "RepositoryHealth   *prometheus.GaugeVec"),
        ("PackageDeployments prometheus.CounterVec", "PackageDeployments *prometheus.CounterVec"),
        ("GitCommands     prometheus.CounterVec", "GitCommands     *prometheus.CounterVec"),
        ("CommandDuration prometheus.HistogramVec", "CommandDuration *prometheus.HistogramVec"),
        ("GitErrors       prometheus.CounterVec", "GitErrors       *prometheus.CounterVec"),
    ]),
    ("pkg/nephio/krm/registry.go", [
        ("FunctionCount       prometheus.GaugeVec", "FunctionCount       *prometheus.GaugeVec"),
        ("DiscoveryDuration   prometheus.HistogramVec", "DiscoveryDuration   *prometheus.HistogramVec"),
        ("HealthCheckFailures prometheus.CounterVec", "HealthCheckFailures *prometheus.CounterVec"),
    ]),
    ("pkg/nephio/krm/porch_integration.go", [
        ("PackageCreations   prometheus.CounterVec", "PackageCreations   *prometheus.CounterVec"),
        ("PackageRevisions   prometheus.CounterVec", "PackageRevisions   *prometheus.CounterVec"),
        ("FunctionExecutions prometheus.CounterVec", "FunctionExecutions *prometheus.CounterVec"),
        ("PipelineExecutions prometheus.CounterVec", "PipelineExecutions *prometheus.CounterVec"),
        ("ExecutionDuration  prometheus.HistogramVec", "ExecutionDuration  *prometheus.HistogramVec"),
        ("PackageSize        prometheus.HistogramVec", "PackageSize        *prometheus.HistogramVec"),
        ("ErrorRate          prometheus.CounterVec", "ErrorRate          *prometheus.CounterVec"),
    ]),
    ("pkg/nephio/krm/pipeline_orchestrator.go", [
        ("PipelineExecutions   prometheus.CounterVec", "PipelineExecutions   *prometheus.CounterVec"),
        ("ExecutionDuration    prometheus.HistogramVec", "ExecutionDuration    *prometheus.HistogramVec"),
        ("StageExecutions      prometheus.CounterVec", "StageExecutions      *prometheus.CounterVec"),
        ("StageDuration        prometheus.HistogramVec", "StageDuration        *prometheus.HistogramVec"),
        ("DependencyResolution prometheus.HistogramVec", "DependencyResolution *prometheus.HistogramVec"),
        ("ErrorRate            prometheus.CounterVec", "ErrorRate            *prometheus.CounterVec"),
        ("ResourceUtilization  prometheus.GaugeVec", "ResourceUtilization  *prometheus.GaugeVec"),
    ]),
    ("pkg/nephio/krm/pipeline.go", [
        ("PipelineExecutions   prometheus.CounterVec", "PipelineExecutions   *prometheus.CounterVec"),
        ("ExecutionDuration    prometheus.HistogramVec", "ExecutionDuration    *prometheus.HistogramVec"),
        ("StageExecutions      prometheus.CounterVec", "StageExecutions      *prometheus.CounterVec"),
        ("StageDuration        prometheus.HistogramVec", "StageDuration        *prometheus.HistogramVec"),
        ("FunctionExecutions   prometheus.CounterVec", "FunctionExecutions   *prometheus.CounterVec"),
        ("FunctionDuration     prometheus.HistogramVec", "FunctionDuration     *prometheus.HistogramVec"),
        ("ErrorRate            prometheus.CounterVec", "ErrorRate            *prometheus.CounterVec"),
        ("RetryCount           prometheus.CounterVec", "RetryCount           *prometheus.CounterVec"),
        ("CheckpointOperations prometheus.CounterVec", "CheckpointOperations *prometheus.CounterVec"),
    ]),
    ("pkg/nephio/krm/function_manager.go", [
        ("FunctionExecutions    prometheus.CounterVec", "FunctionExecutions    *prometheus.CounterVec"),
        ("ExecutionDuration     prometheus.HistogramVec", "ExecutionDuration     *prometheus.HistogramVec"),
        ("ExecutionErrors       prometheus.CounterVec", "ExecutionErrors       *prometheus.CounterVec"),
        ("CachePerformance      prometheus.HistogramVec", "CachePerformance      *prometheus.HistogramVec"),
        ("SecurityViolations    prometheus.CounterVec", "SecurityViolations    *prometheus.CounterVec"),
        ("ResourceUtilization   prometheus.GaugeVec", "ResourceUtilization   *prometheus.GaugeVec"),
    ]),
]

for filepath, replacements in files_to_fix:
    if os.path.exists(filepath):
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original_content = content
        for old, new in replacements:
            content = content.replace(old, new)
        
        if content != original_content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"Fixed {filepath}")
        else:
            print(f"No changes needed in {filepath}")
    else:
        print(f"File not found: {filepath}")

print("\nDone! All Prometheus metric fields have been changed to pointer types.")