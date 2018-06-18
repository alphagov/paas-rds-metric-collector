package collector

import (
	"code.cloudfoundry.org/lager"

	// Used in the SQL driver.
	_ "github.com/lib/pq"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
)

var postgresMetricQueries = []MetricQuery{
	MetricQuery{
		Query: "SELECT CAST (SUM(numbackends) AS DOUBLE PRECISION) AS connections FROM pg_stat_database",
		Metrics: []MetricQueryMeta{
			{
				Key:  "connections",
				Unit: "conn",
			},
		},
	},
}

// NewPostgresMetricsCollectorDriver ...
func NewPostgresMetricsCollectorDriver(brokerInfo brokerinfo.BrokerInfo, logger lager.Logger) MetricsCollectorDriver {
	return &sqlMetricsCollectorDriver{
		logger:     logger,
		queries:    postgresMetricQueries,
		driver:     "postgres",
		brokerInfo: brokerInfo,
		name:       "postgres",
	}
}
