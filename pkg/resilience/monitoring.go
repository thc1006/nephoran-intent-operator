package resilience

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// MonitoringClient wraps Prometheus API client
type MonitoringClient struct {
	client v1.API
	logger logr.Logger
}

// NewMonitoringClient creates a new monitoring client
func NewMonitoringClient(endpoint string) (*MonitoringClient, error) {
	client, err := api.NewClient(api.Config{
		Address: endpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	return &MonitoringClient{
		client: v1.NewAPI(client),
	}, nil
}

// QueryMetric queries a Prometheus metric
func (m *MonitoringClient) QueryMetric(ctx context.Context, query string) (model.Value, error) {
	result, warnings, err := m.client.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	if len(warnings) > 0 {
		if m.logger.Enabled() {
			m.logger.Info("Prometheus query returned warnings", "warnings", warnings, "query", query)
		}
	}

	return result, nil
}

// QueryRange queries metrics over a time range
func (m *MonitoringClient) QueryRange(ctx context.Context, query string, r v1.Range) (model.Value, error) {
	result, warnings, err := m.client.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("range query failed: %w", err)
	}

	if len(warnings) > 0 {
		if m.logger.Enabled() {
			m.logger.Info("Prometheus range query returned warnings", "warnings", warnings, "query", query)
		}
	}

	return result, nil
}
