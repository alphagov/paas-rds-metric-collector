package collector

import (
	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

// MetricsCollectorDriver ...
type MetricsCollectorDriver interface {
	NewCollector(instanceInfo brokerinfo.InstanceInfo) (MetricsCollector, error)
	GetName() string
	SupportedTypes() []string
}

// MetricsCollector ...
type MetricsCollector interface {
	Collect() ([]metrics.Metric, error)
	Close() error
}
