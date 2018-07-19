package collector

import (
	"code.cloudfoundry.org/lager"

	// Used in the SQL driver.
	_ "github.com/go-sql-driver/mysql"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
)

var mysqlMetricQueries = []metricQuery{
	&rowMetricQuery{
		Query: `
			SELECT *
			FROM performance_schema.global_status
			WHERE variable_name IN (
				 'Threads_connected'
				,'Threads_running'
			);
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "threads_connected",
				Unit: "conn",
			},
			{
				Key:  "threads_running",
				Unit: "conn",
			},
		},
	},
}

// NewMysqlMetricsCollectorDriver ...
func NewMysqlMetricsCollectorDriver(brokerInfo brokerinfo.BrokerInfo, logger lager.Logger) MetricsCollectorDriver {
	return &sqlMetricsCollectorDriver{
		logger:     logger,
		queries:    mysqlMetricQueries,
		driver:     "mysql",
		brokerInfo: brokerInfo,
		name:       "mysql",
	}
}
