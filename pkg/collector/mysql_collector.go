package collector

import (
	"fmt"

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
			  ,'Threads_created'
			  ,'Connections'
			  ,'Queries'
			  ,'Questions'
			  ,'Aborted_clients'
			  ,'Aborted_connects'
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
			{
				Key:  "threads_created",
				Unit: "conn",
			},
			{
				Key:  "queries",
				Unit: "conn",
			},
			{
				Key:  "questions",
				Unit: "conn",
			},
			{
				Key:  "aborted_clients",
				Unit: "conn",
			},
			{
				Key:  "aborted_connects",
				Unit: "conn",
			},
		},
	},
	&rowMetricQuery{
		Query: `
			SELECT *
			FROM performance_schema.global_status
			WHERE variable_name IN (
			   'Innodb_row_lock_waits'
			  ,'Innodb_row_lock_time'
			  ,'Innodb_num_open_files'
			  ,'Innodb_log_waits'
			  ,'Innodb_buffer_pool_bytes_data'
			  ,'Innodb_buffer_pool_bytes_dirty'
			  ,'Innodb_buffer_pool_pages_data'
			  ,'Innodb_buffer_pool_pages_dirty'
			  ,'Innodb_buffer_pool_pages_flushed'
			  ,'Innodb_buffer_pool_pages_free'
			  ,'Innodb_buffer_pool_pages_misc'
			  ,'Innodb_buffer_pool_pages_total'
			  ,'Innodb_buffer_pool_read_ahead'
			  ,'Innodb_buffer_pool_read_ahead_evicted'
			  ,'Innodb_buffer_pool_read_ahead_rnd'
			  ,'Innodb_buffer_pool_read_requests'
			  ,'Innodb_buffer_pool_reads'
			  ,'Innodb_buffer_pool_wait_free'
			  ,'Innodb_buffer_pool_write_requests'
			);
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "innodb_row_lock_waits",
				Unit: "guage",
			},
			{
				Key:  "innodb_row_lock_time",
				Unit: "ms",
			},
			{
				Key:  "innodb_num_open_files",
				Unit: "files",
			},
			{
				Key:  "innodb_log_waits",
				Unit: "guage",
			},
			{
				Key:  "innodb_buffer_pool_bytes_data",
				Unit: "bytes",
			},
			{
				Key:  "innodb_buffer_pool_bytes_dirty",
				Unit: "bytes",
			},
			{
				Key:  "innodb_buffer_pool_pages_data",
				Unit: "pages",
			},
			{
				Key:  "innodb_buffer_pool_pages_dirty",
				Unit: "pages",
			},
			{
				Key:  "innodb_buffer_pool_pages_flushed",
				Unit: "pages",
			},
			{
				Key:  "innodb_buffer_pool_pages_free",
				Unit: "pages",
			},
			{
				Key:  "innodb_buffer_pool_pages_misc",
				Unit: "pages",
			},
			{
				Key:  "innodb_buffer_pool_pages_total",
				Unit: "pages",
			},
			{
				Key:  "innodb_buffer_pool_read_ahead",
				Unit: "pages",
			},
			{
				Key:  "innodb_buffer_pool_read_ahead_evicted",
				Unit: "pages",
			},
			{
				Key:  "innodb_buffer_pool_read_ahead_rnd",
				Unit: "guage",
			},
			{
				Key:  "innodb_buffer_pool_read_requests",
				Unit: "conn",
			},
			{
				Key:  "innodb_buffer_pool_reads",
				Unit: "guage",
			},
			{
				Key:  "innodb_buffer_pool_wait_free",
				Unit: "guage",
			},
			{
				Key:  "innodb_buffer_pool_write_requests",
				Unit: "guage",
			},
		},
	},
	&columnMetricQuery{
		Query: `
			SELECT
					variable_value as max_connections
			FROM performance_schema.global_variables
			WHERE variable_name = 'max_connections';
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
					SUM(variable_value) as connection_errors
			FROM performance_schema.global_status
			WHERE variable_name like 'Connection_errors_%';
		`,
		Metrics: []metricQueryMeta{
			{
				Key:  "connection_errors",
				Unit: "err",
			},
		},
	},
}

type mysqlConnectionStringBuilder struct {
	ConnectionTimeout int
	TLS               string
}

func (m *mysqlConnectionStringBuilder) ConnectionString(details brokerinfo.InstanceConnectionDetails) string {
	tls := "false"
	if m.TLS != "" {
		tls = m.TLS
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?tls=%s&timeout=%ds",
		details.MasterUsername,
		details.MasterPassword,
		details.DBAddress,
		details.DBPort,
		details.DBName,
		tls,
		m.ConnectionTimeout,
	)
}

// NewMysqlMetricsCollectorDriver ...
func NewMysqlMetricsCollectorDriver(
	brokerInfo brokerinfo.BrokerInfo,
	intervalSeconds int,
	timeout int,
	TLS string,
	logger lager.Logger,
) MetricsCollectorDriver {
	return &sqlMetricsCollectorDriver{
		collectInterval: intervalSeconds,
		logger:          logger,
		queries:         mysqlMetricQueries,
		driver:          "mysql",
		brokerInfo:      brokerInfo,
		name:            "mysql",
		connectionStringBuilder: &mysqlConnectionStringBuilder{
			ConnectionTimeout: timeout,
			TLS:               TLS,
		},
	}
}
