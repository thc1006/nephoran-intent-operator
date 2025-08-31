package o1

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
	api "github.com/prometheus/client_golang/api/prometheus/v1"
)

// Export required prometheus types for O-RAN O1 interface
var (
	_ prometheus.Counter = (*prometheus.CounterVec)(nil).WithLabelValues("")
	_ prometheus.Gauge   = promauto.NewGauge(prometheus.GaugeOpts{})
	_ api.Client         = (*api.API)(nil) 
)