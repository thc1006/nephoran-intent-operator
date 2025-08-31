package o1

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
)

// Export required prometheus types for O-RAN O1 interface
var (
	_ prometheus.Counter = (*prometheus.CounterVec)(nil).WithLabelValues("")
	_ prometheus.Gauge   = promauto.NewGauge(prometheus.GaugeOpts{})
	_ model.Metric      = model.Metric{}
)