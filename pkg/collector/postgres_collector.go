package collector

import (
	"fmt"

	"code.cloudfoundry.org/lager"

	// Used in the SQL driver.
	_ "github.com/Kount/pq-timeouts"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
)

var postgresMetricQueries = []metricQuery{
	&columnMetricQuery{
		Query: `
			SELECT
				SUM(numbackends) AS connections
			FROM pg_stat_database
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "connections",
				Unit: "conn",
			},
		},
	},
	&columnMetricQuery{
		Query: `
			SELECT
					setting::float as max_connections
			FROM pg_settings
			WHERE name = 'max_connections'
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "max_connections",
				Unit: "conn",
			},
		},
	},
	&columnMetricQuery{
		Query: `
			SELECT
				pg_database_size(pg_database.datname) as dbsize,
				current_database() as dbname
			FROM pg_database
			WHERE datname = current_database()
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "dbsize",
				Unit: "byte",
			},
		},
	},
	&columnMetricQuery{
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
		Metrics: []metricQueryMeta{
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
	&columnMetricQuery{
		Query: `
			SELECT
				COALESCE(SUM(seq_scan), 0) as seq_scan,
				COALESCE(SUM(idx_scan), 0) as idx_scan,
				current_database() as dbname
			FROM pg_stat_user_tables
		`,
		Metrics: []metricQueryMeta{
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
	&columnMetricQuery{
		Query: `
			SELECT
				count(distinct pid) as blocked_connections
			FROM pg_locks
			WHERE granted = false
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "blocked_connections",
				Unit: "conn",
			},
		},
	},
	&columnMetricQuery{
		Query: `
			SELECT
				COALESCE(EXTRACT(epoch FROM MAX(now() - a.xact_start))::INT, 0) as max_tx_age
			FROM pg_stat_activity a
			INNER JOIN pg_roles r ON r.rolname = a.usename
			INNER JOIN pg_group g ON r.oid = ANY (g.grolist)
			WHERE
				g.groname LIKE 'rdsbroker_%_manager'
			AND state IN ('idle in transaction', 'active')
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "max_tx_age",
				Unit: "s",
			},
		},
	},
	&columnMetricQuery{
		Query: `
			SELECT
				COALESCE(EXTRACT(epoch FROM MAX(now() - xact_start))::INT, 0) as max_system_tx_age
			FROM pg_stat_activity a
			INNER JOIN pg_roles r ON r.rolname = a.usename
			LEFT JOIN pg_group g ON r.oid = ANY (g.grolist)
			WHERE
				(g.groname NOT LIKE 'rdsbroker_%_manager' OR g.groname IS NULL)
			AND state IN ('idle in transaction', 'active')
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "max_system_tx_age",
				Unit: "s",
			},
		},
	},
}

type postgresConnectionStringBuilder struct {
	ConnectionTimeout int
	ReadTimeout       int
	WriteTimeout      int
	SSLMode           string
}

func (m *postgresConnectionStringBuilder) ConnectionString(details brokerinfo.InstanceConnectionDetails) string {
	sslMode := "disable"
	if m.SSLMode != "" {
		sslMode = m.SSLMode
	}
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s&connect_timeout=%d&read_timeout=%d&write_timeout=%d",
		details.MasterUsername,
		details.MasterPassword,
		details.DBAddress,
		details.DBPort,
		details.DBName,
		sslMode,
		m.ConnectionTimeout,
		m.ReadTimeout*1000,
		m.WriteTimeout*1000,
	)
}

// NewPostgresMetricsCollectorDriver ...
func NewPostgresMetricsCollectorDriver(
	brokerInfo brokerinfo.BrokerInfo,
	intervalSeconds int,
	timeout int,
	SSLMode string,
	logger lager.Logger,
) MetricsCollectorDriver {
	return &sqlMetricsCollectorDriver{
		collectInterval: intervalSeconds,
		logger:          logger,
		queries:         postgresMetricQueries,
		driver:          "pq-timeouts",
		brokerInfo:      brokerInfo,
		name:            "postgres",
		connectionStringBuilder: &postgresConnectionStringBuilder{
			ConnectionTimeout: timeout,
			ReadTimeout:       timeout,
			WriteTimeout:      timeout,
			SSLMode:           SSLMode,
		},
	}
}
