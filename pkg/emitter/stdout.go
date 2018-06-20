package emitter

import (
	"fmt"

	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

// StdOutEmitter ...
type StdOutEmitter struct{}

// Emit ...
func (s *StdOutEmitter) Emit(m metrics.MetricEnvelope) {
	fmt.Println("============>", m)
}
