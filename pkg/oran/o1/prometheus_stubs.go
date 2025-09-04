package o1

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
)

// Export required prometheus types for O-RAN O1 interface
var (
	// Create a dummy CounterVec and use it properly instead of nil pointer
	_dummyCounterVec                    = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "dummy"}, []string{"label"})
	_                prometheus.Counter = _dummyCounterVec.WithLabelValues("")
	_                prometheus.Gauge   = promauto.NewGauge(prometheus.GaugeOpts{Name: "dummy_gauge"})
	_                model.Metric       = model.Metric{}
)
