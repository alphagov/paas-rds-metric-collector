package collector

import "github.com/alphagov/paas-rds-metric-collector/pkg/metrics"

// MetricsCollectorDriver ...
type MetricsCollectorDriver interface {
	NewCollector(instanceID string) (MetricsCollector, error)
	GetName() string
}

// MetricsCollector ...
type MetricsCollector interface {
	Collect() ([]metrics.Metric, error)
	Close() error
}
