// Package observability provides observability integration for service mesh
package observability

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/thc1006/nephoran-intent-operator/pkg/servicemesh/abstraction"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ObservabilityIntegration manages observability for service mesh
type ObservabilityIntegration struct {
	mesh            abstraction.ServiceMeshInterface
	metricsRegistry *prometheus.Registry
	tracer          trace.Tracer
	meter           metric.Meter
	logger          logr.Logger

	// Metrics
	mtlsConnections      metric.Int64Counter
	certificateRotations metric.Int64Counter
	policyApplications   metric.Int64Counter
	meshHealthGauge      metric.Float64ObservableGauge
	serviceLatency       metric.Float64Histogram
	errorRate            metric.Float64ObservableGauge
}

// NewObservabilityIntegration creates new observability integration
func NewObservabilityIntegration(mesh abstraction.ServiceMeshInterface) (*ObservabilityIntegration, error) {
	tracer := otel.Tracer("service-mesh")
	meter := otel.Meter("service-mesh")

	integration := &ObservabilityIntegration{
		mesh:            mesh,
		metricsRegistry: prometheus.NewRegistry(),
		tracer:          tracer,
		meter:           meter,
		logger:          log.Log.WithName("mesh-observability"),
	}

	// Initialize metrics
	if err := integration.initializeMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// Register mesh-specific metrics
	for _, collector := range mesh.GetMetrics() {
		integration.metricsRegistry.MustRegister(collector)
	}

	return integration, nil
}

// initializeMetrics initializes OpenTelemetry metrics
func (o *ObservabilityIntegration) initializeMetrics() error {
	var err error

	// mTLS connections counter
	o.mtlsConnections, err = o.meter.Int64Counter(
		"service_mesh_mtls_connections_total",
		metric.WithDescription("Total number of mTLS connections established"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create mtls connections metric: %w", err)
	}

	// Certificate rotations counter
	o.certificateRotations, err = o.meter.Int64Counter(
		"service_mesh_certificate_rotations_total",
		metric.WithDescription("Total number of certificate rotations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create certificate rotations metric: %w", err)
	}

	// Policy applications counter
	o.policyApplications, err = o.meter.Int64Counter(
		"service_mesh_policy_applications_total",
		metric.WithDescription("Total number of policy applications"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create policy applications metric: %w", err)
	}

	// Service latency histogram
	o.serviceLatency, err = o.meter.Float64Histogram(
		"service_mesh_latency_milliseconds",
		metric.WithDescription("Service-to-service latency in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return fmt.Errorf("failed to create service latency metric: %w", err)
	}

	// Mesh health gauge
	o.meshHealthGauge, err = o.meter.Float64ObservableGauge(
		"service_mesh_health_score",
		metric.WithDescription("Service mesh health score (0-100)"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create mesh health metric: %w", err)
	}

	// Error rate gauge
	o.errorRate, err = o.meter.Float64ObservableGauge(
		"service_mesh_error_rate",
		metric.WithDescription("Service mesh error rate percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return fmt.Errorf("failed to create error rate metric: %w", err)
	}

	// Register callbacks for observable gauges
	_, err = o.meter.RegisterCallback(o.observeMetrics,
		o.meshHealthGauge,
		o.errorRate,
	)
	if err != nil {
		return fmt.Errorf("failed to register metric callbacks: %w", err)
	}

	return nil
}

// observeMetrics is the callback for observable metrics
func (o *ObservabilityIntegration) observeMetrics(ctx context.Context, observer metric.Observer) error {
	// Get mesh health status
	if err := o.mesh.IsHealthy(ctx); err == nil {
		observer.ObserveFloat64(o.meshHealthGauge, 100.0)
	} else {
		observer.ObserveFloat64(o.meshHealthGauge, 0.0)
	}

	// Calculate error rate (simplified - would need actual data)
	observer.ObserveFloat64(o.errorRate, 0.01) // 1% error rate

	return nil
}

// RecordMTLSConnection records an mTLS connection
func (o *ObservabilityIntegration) RecordMTLSConnection(ctx context.Context, source, destination string) {
	_, span := o.tracer.Start(ctx, "mtls_connection",
		trace.WithAttributes(
			attribute.String("source", source),
			attribute.String("destination", destination),
		),
	)
	defer span.End()

	o.mtlsConnections.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("source", source),
			attribute.String("destination", destination),
		),
	)
}

// RecordCertificateRotation records a certificate rotation
func (o *ObservabilityIntegration) RecordCertificateRotation(ctx context.Context, service, namespace string) {
	_, span := o.tracer.Start(ctx, "certificate_rotation",
		trace.WithAttributes(
			attribute.String("service", service),
			attribute.String("namespace", namespace),
		),
	)
	defer span.End()

	o.certificateRotations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("service", service),
			attribute.String("namespace", namespace),
		),
	)
}

// RecordPolicyApplication records a policy application
func (o *ObservabilityIntegration) RecordPolicyApplication(ctx context.Context, policyType, policyName string, success bool) {
	_, span := o.tracer.Start(ctx, "policy_application",
		trace.WithAttributes(
			attribute.String("policy_type", policyType),
			attribute.String("policy_name", policyName),
			attribute.Bool("success", success),
		),
	)
	defer span.End()

	o.policyApplications.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("policy_type", policyType),
			attribute.Bool("success", success),
		),
	)
}

// RecordServiceLatency records service-to-service latency
func (o *ObservabilityIntegration) RecordServiceLatency(ctx context.Context, source, destination string, latencyMs float64) {
	_, span := o.tracer.Start(ctx, "service_communication",
		trace.WithAttributes(
			attribute.String("source", source),
			attribute.String("destination", destination),
			attribute.Float64("latency_ms", latencyMs),
		),
	)
	defer span.End()

	o.serviceLatency.Record(ctx, latencyMs,
		metric.WithAttributes(
			attribute.String("source", source),
			attribute.String("destination", destination),
		),
	)
}

// GenerateMTLSReport generates a comprehensive mTLS status report
func (o *ObservabilityIntegration) GenerateMTLSReport(ctx context.Context) (*MTLSReport, error) {
	ctx, span := o.tracer.Start(ctx, "generate_mtls_report")
	defer span.End()

	report := &MTLSReport{
		Timestamp: time.Now(),
		Provider:  o.mesh.GetProvider(),
	}

	// Get mTLS status from mesh
	status, err := o.mesh.GetMTLSStatus(ctx, "")
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get mTLS status: %w", err)
	}

	report.TotalServices = status.TotalServices
	report.MTLSEnabledServices = status.MTLSEnabledCount
	report.Coverage = status.Coverage
	report.CertificateStatus = make([]CertificateMetrics, 0)

	// Analyze certificate status
	for _, certStatus := range status.CertificateStatus {
		metrics := CertificateMetrics{
			Service:         certStatus.Service,
			Namespace:       certStatus.Namespace,
			Valid:           certStatus.Valid,
			ExpiryDate:      certStatus.ExpiryDate,
			DaysUntilExpiry: int(time.Until(certStatus.ExpiryDate).Hours() / 24),
			NeedsRotation:   certStatus.NeedsRotation,
		}
		report.CertificateStatus = append(report.CertificateStatus, metrics)
	}

	// Calculate health score
	report.HealthScore = o.calculateHealthScore(status)

	// Add recommendations
	report.Recommendations = o.generateRecommendations(status)

	span.SetAttributes(
		attribute.Float64("coverage", report.Coverage),
		attribute.Float64("health_score", report.HealthScore),
		attribute.Int("total_services", report.TotalServices),
	)

	return report, nil
}

// calculateHealthScore calculates the overall health score
func (o *ObservabilityIntegration) calculateHealthScore(status *abstraction.MTLSStatusReport) float64 {
	score := 0.0
	weights := 0.0

	// mTLS coverage (weight: 40%)
	score += status.Coverage * 0.4
	weights += 40.0

	// Certificate validity (weight: 30%)
	validCerts := 0
	for _, cert := range status.CertificateStatus {
		if cert.Valid && !cert.NeedsRotation {
			validCerts++
		}
	}
	if len(status.CertificateStatus) > 0 {
		certScore := float64(validCerts) / float64(len(status.CertificateStatus)) * 100
		score += certScore * 0.3
		weights += 30.0
	}

	// Policy compliance (weight: 30%)
	if len(status.PolicyStatus) > 0 {
		compliantPolicies := 0
		for _, policy := range status.PolicyStatus {
			if policy.MTLSMode == "STRICT" {
				compliantPolicies++
			}
		}
		policyScore := float64(compliantPolicies) / float64(len(status.PolicyStatus)) * 100
		score += policyScore * 0.3
		weights += 30.0
	}

	if weights > 0 {
		return score * (100.0 / weights)
	}
	return 0
}

// generateRecommendations generates recommendations based on status
func (o *ObservabilityIntegration) generateRecommendations(status *abstraction.MTLSStatusReport) []string {
	recommendations := []string{}

	// Check mTLS coverage
	if status.Coverage < 100 {
		recommendations = append(recommendations,
			fmt.Sprintf("Enable mTLS for remaining %.0f%% of services", 100-status.Coverage))
	}

	// Check certificate expiry
	expiringCerts := 0
	for _, cert := range status.CertificateStatus {
		if cert.NeedsRotation {
			expiringCerts++
		}
	}
	if expiringCerts > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Rotate %d expiring certificates", expiringCerts))
	}

	// Check for permissive mTLS mode
	permissiveCount := 0
	for _, policy := range status.PolicyStatus {
		if policy.MTLSMode == "PERMISSIVE" {
			permissiveCount++
		}
	}
	if permissiveCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Switch %d services from PERMISSIVE to STRICT mTLS mode", permissiveCount))
	}

	// Check for issues
	for _, issue := range status.Issues {
		if issue.Severity == "critical" {
			recommendations = append(recommendations,
				fmt.Sprintf("Address critical issue: %s", issue.Message))
		}
	}

	return recommendations
}

// GenerateServiceDependencyMap generates a service dependency visualization
func (o *ObservabilityIntegration) GenerateServiceDependencyMap(ctx context.Context, namespace string) (*DependencyVisualization, error) {
	ctx, span := o.tracer.Start(ctx, "generate_dependency_map")
	defer span.End()

	// Get dependency graph from mesh
	graph, err := o.mesh.GetServiceDependencies(ctx, namespace)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get service dependencies: %w", err)
	}

	viz := &DependencyVisualization{
		Namespace: namespace,
		Timestamp: time.Now(),
		Nodes:     make([]VisualizationNode, 0),
		Edges:     make([]VisualizationEdge, 0),
	}

	// Convert nodes
	for _, node := range graph.Nodes {
		vizNode := VisualizationNode{
			ID:          node.ID,
			Name:        node.Name,
			Type:        node.Type,
			MTLSEnabled: node.MTLSEnabled,
			Metrics: NodeMetrics{
				RequestRate: 0, // Would need actual metrics
				ErrorRate:   0,
				P99Latency:  0,
			},
		}
		viz.Nodes = append(viz.Nodes, vizNode)
	}

	// Convert edges
	for _, edge := range graph.Edges {
		vizEdge := VisualizationEdge{
			Source:      edge.Source,
			Target:      edge.Target,
			Protocol:    edge.Protocol,
			MTLSEnabled: edge.MTLSEnabled,
			Metrics: EdgeMetrics{
				RequestRate: 0, // Would need actual metrics
				ErrorRate:   edge.ErrorRate,
				Latency:     edge.Latency,
			},
		}
		viz.Edges = append(viz.Edges, vizEdge)
	}

	// Detect cycles
	if len(graph.Cycles) > 0 {
		viz.HasCycles = true
		viz.Cycles = graph.Cycles
	}

	span.SetAttributes(
		attribute.Int("node_count", len(viz.Nodes)),
		attribute.Int("edge_count", len(viz.Edges)),
		attribute.Bool("has_cycles", viz.HasCycles),
	)

	return viz, nil
}

// ExportMetrics exports metrics in Prometheus format
func (o *ObservabilityIntegration) ExportMetrics() ([]byte, error) {
	// Use Prometheus registry to gather metrics
	metricFamilies, err := o.metricsRegistry.Gather()
	if err != nil {
		return nil, fmt.Errorf("failed to gather metrics: %w", err)
	}

	// Convert to text format
	var buf bytes.Buffer
	for _, mf := range metricFamilies {
		if _, err := expfmt.MetricFamilyToText(&buf, mf); err != nil {
			return nil, fmt.Errorf("failed to export metric family: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// StartContinuousMonitoring starts continuous monitoring
func (o *ObservabilityIntegration) StartContinuousMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("Stopping continuous monitoring")
			return
		case <-ticker.C:
			o.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck performs a health check
func (o *ObservabilityIntegration) performHealthCheck(ctx context.Context) {
	ctx, span := o.tracer.Start(ctx, "health_check")
	defer span.End()

	// Check mesh health
	if err := o.mesh.IsHealthy(ctx); err != nil {
		o.logger.Error(err, "Mesh health check failed")
		span.RecordError(err)
		// Record unhealthy state
		o.recordHealthMetric(ctx, false)
		return
	}

	// Check mTLS status
	status, err := o.mesh.GetMTLSStatus(ctx, "")
	if err != nil {
		o.logger.Error(err, "Failed to get mTLS status")
		span.RecordError(err)
		return
	}

	// Log status
	o.logger.Info("Health check completed",
		"mtls_coverage", status.Coverage,
		"total_services", status.TotalServices,
		"issues", len(status.Issues))

	// Record healthy state
	o.recordHealthMetric(ctx, true)

	span.SetAttributes(
		attribute.Float64("mtls_coverage", status.Coverage),
		attribute.Int("total_services", status.TotalServices),
		attribute.Int("issue_count", len(status.Issues)),
	)
}

// recordHealthMetric records health metric
func (o *ObservabilityIntegration) recordHealthMetric(ctx context.Context, healthy bool) {
	healthValue := 0.0
	if healthy {
		healthValue = 1.0
	}

	// Record to Prometheus metric
	healthGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "service_mesh_health",
		Help: "Service mesh health status (1 = healthy, 0 = unhealthy)",
	})
	healthGauge.Set(healthValue)
	o.metricsRegistry.MustRegister(healthGauge)
}
