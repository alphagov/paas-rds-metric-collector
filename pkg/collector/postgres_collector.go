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
					setting::float as max_connections
			FROM pg_settings
			WHERE name = 'max_connections'
		`,
		Metrics: []MetricQueryMeta{
			{
				Key:  "max_connections",
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
				deadlocks as deadlocks,
				xact_commit as commits,
				xact_rollback as rollbacks,
				blks_read as blocks_read,
				blks_hit as blocks_hit,
				blk_read_time as read_time,
				blk_write_time as write_time,
				temp_bytes as temp_bytes,
				current_database() as dbname
			FROM pg_stat_database
			WHERE datname = current_database()
		`,
		Metrics: []MetricQueryMeta{
			{
				Key:  "deadlocks",
				Unit: "lock",
			},
			{
				Key:  "commits",
				Unit: "tx",
			},
			{
				Key:  "rollbacks",
				Unit: "tx",
			},
			{
				Key:  "blocks_read",
				Unit: "block",
			},
			{
				Key:  "blocks_hit",
				Unit: "block",
			},
			{
				Key:  "read_time",
				Unit: "ms",
			},
			{
				Key:  "write_time",
				Unit: "ms",
			},
			{
				Key:  "temp_bytes",
				Unit: "byte",
			},
		},
	},
	MetricQuery{
		Query: `
			SELECT
				COALESCE(SUM(seq_scan), 0) as seq_scan,
				COALESCE(SUM(idx_scan), 0) as idx_scan,
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
	MetricQuery{
		Query: `
			SELECT
				count(distinct pid) as blocked_connections
			FROM pg_locks
			WHERE granted = false
		`,
		Metrics: []MetricQueryMeta{
			{
				Key:  "blocked_connections",
				Unit: "conn",
			},
		},
	},
	MetricQuery{
		Query: `
			SELECT
				EXTRACT(epoch FROM MAX(now() - xact_start))::INT as max_tx_age
			FROM pg_stat_activity
      WHERE state IN ('idle in transaction', 'active')
		`,
		Metrics: []MetricQueryMeta{
			{
				Key:  "max_tx_age",
				Unit: "s",
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
