package collector

import "github.com/alphagov/paas-rds-metric-collector/pkg/metrics"

type MetricsCollectorDriver interface {
	NewCollector(instanceID string) (MetricsCollector, error)
}

type MetricsCollector interface {
	Collect() ([]metrics.Metric, error)
	Close() error
}
