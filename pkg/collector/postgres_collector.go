package collector

import (
	"code.cloudfoundry.org/lager"

	// Used in the SQL driver.
	_ "github.com/lib/pq"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
)

var postgresMetricQueries = []MetricQuery{
	MetricQuery{
		Query: `
			SELECT
				SUM(numbackends) AS connections
			FROM pg_stat_database
		`,
		Metrics: []MetricQueryMeta{
			{
				Key:  "connections",
				Unit: "conn",
			},
		},
	},
	MetricQuery{
		Query: `
			SELECT
				pg_database_size(pg_database.datname) as dbsize,
				current_database() as dbname
			FROM pg_database
			WHERE datname = current_database()
		`,
		Metrics: []MetricQueryMeta{
			{
				Key:  "dbsize",
				Unit: "byte",
			},
		},
	},
	MetricQuery{
		Query: `
			SELECT
				deadlocks,
				current_database() as dbname
			FROM pg_stat_database
			WHERE datname = current_database()
		`,
		Metrics: []MetricQueryMeta{
			{
				Key:  "deadlocks",
				Unit: "lock",
			},
		},
	},
	MetricQuery{
		Query: `
			SELECT
				COALESCE(seq_scan, 0) as seq_scan,
				COALESCE(idx_scan, 0) as idx_scan,
				relname as table_name,
				current_database() as dbname
			FROM pg_stat_user_tables
		`,
		Metrics: []MetricQueryMeta{
			{
				Key:  "seq_scan",
				Unit: "scan",
			},
			{
				Key:  "idx_scan",
				Unit: "scan",
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
