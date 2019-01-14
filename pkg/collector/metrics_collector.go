package collector

import (
	"context"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

// MetricsCollectorDriver ...
type MetricsCollectorDriver interface {
	NewCollector(instanceInfo brokerinfo.InstanceInfo) (MetricsCollector, error)
	GetName() string
	SupportedTypes() []string
	GetCollectInterval() int
}

// MetricsCollector ...
type MetricsCollector interface {
	Collect(ctx context.Context) ([]metrics.Metric, error)
	Close() error
}
