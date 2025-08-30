// Package istio provides metrics for Istio service mesh operations.

package istio

import (
	"github.com/prometheus/client_golang/prometheus"
)

// IstioMetrics contains Prometheus metrics for Istio operations.

type IstioMetrics struct {
	policiesApplied prometheus.Counter

	servicesRegistered prometheus.Gauge

	mtlsConnectionsTotal prometheus.Counter

	certificateRotations prometheus.Counter

	policyValidations prometheus.Counter

	meshHealthStatus prometheus.Gauge
}

// NewIstioMetrics creates new Istio metrics.

func NewIstioMetrics() *IstioMetrics {

	return &IstioMetrics{

		policiesApplied: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "istio_policies_applied_total",

			Help: "Total number of Istio policies applied",
		}),

		servicesRegistered: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "istio_services_registered",

			Help: "Number of services registered with Istio mesh",
		}),

		mtlsConnectionsTotal: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "istio_mtls_connections_total",

			Help: "Total number of mTLS connections established",
		}),

		certificateRotations: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "istio_certificate_rotations_total",

			Help: "Total number of certificate rotations performed",
		}),

		policyValidations: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "istio_policy_validations_total",

			Help: "Total number of policy validations performed",
		}),

		meshHealthStatus: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "istio_mesh_health_status",

			Help: "Health status of the Istio mesh (1 = healthy, 0 = unhealthy)",
		}),
	}

}

// GetCollectors returns all Prometheus collectors.

func (m *IstioMetrics) GetCollectors() []prometheus.Collector {

	return []prometheus.Collector{

		m.policiesApplied,

		m.servicesRegistered,

		m.mtlsConnectionsTotal,

		m.certificateRotations,

		m.policyValidations,

		m.meshHealthStatus,
	}

}

// Inc increments a counter metric.

func (m *IstioMetrics) Inc(metric string) {

	switch metric {

	case "policies_applied":

		m.policiesApplied.Inc()

	case "mtls_connections":

		m.mtlsConnectionsTotal.Inc()

	case "certificate_rotations":

		m.certificateRotations.Inc()

	case "policy_validations":

		m.policyValidations.Inc()

	}

}

// Set sets a gauge metric value.

func (m *IstioMetrics) Set(metric string, value float64) {

	switch metric {

	case "services_registered":

		m.servicesRegistered.Set(value)

	case "mesh_health":

		m.meshHealthStatus.Set(value)

	}

}

// Add adds to a gauge metric.

func (m *IstioMetrics) Add(metric string, value float64) {

	switch metric {

	case "services_registered":

		m.servicesRegistered.Add(value)

	}

}
