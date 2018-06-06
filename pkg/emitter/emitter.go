package emitter

import "github.com/alphagov/paas-rds-metric-collector/pkg/metrics"

type MetricsEmitter interface {
	Emit(metrics.MetricEnvelope)
}
