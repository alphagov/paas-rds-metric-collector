package collector

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

// MetricQuery has a method to get the metrics for a query
type metricQuery interface {
	getMetrics(ctx context.Context, db *sql.DB) ([]metrics.Metric, error)
}

// Used to get the right connection string for each DB type
type sqlConnectionStringBuilder interface {
	ConnectionString(brokerinfo.InstanceConnectionDetails) string
}

// sqlMetricsCollectorDriver pulls metrics using generic SQL queries
type sqlMetricsCollectorDriver struct {
	collectInterval int
	brokerInfo      brokerinfo.BrokerInfo
	queries         []metricQuery
	driver          string
	name            string
	logger          lager.Logger

	connectionStringBuilder sqlConnectionStringBuilder
}

// NewCollector ...
func (d *sqlMetricsCollectorDriver) NewCollector(instanceInfo brokerinfo.InstanceInfo) (MetricsCollector, error) {
	details, err := d.brokerInfo.GetInstanceConnectionDetails(instanceInfo)
	if err != nil {
		d.logger.Error("cannot compose connection string", err, lager.Data{
			"instanceInfo": instanceInfo,
		})
		return nil, err
	}

	url := d.connectionStringBuilder.ConnectionString(details)

	dbConn, err := sql.Open(d.driver, url)
	if err != nil {
		d.logger.Error("cannot connect to the database", err, lager.Data{
			"instanceInfo": instanceInfo,
		})
		return nil, err
	}

	sqlMetricsCollector := &sqlMetricsCollector{
		logger:  d.logger,
		queries: d.queries,
		dbConn:  dbConn,
	}

	return sqlMetricsCollector, nil
}

func (d *sqlMetricsCollectorDriver) GetName() string {
	return d.name
}

func (d *sqlMetricsCollectorDriver) SupportedTypes() []string {
	return []string{d.name}
}

func (d *sqlMetricsCollectorDriver) GetCollectInterval() int {
	return d.collectInterval
}

type sqlMetricsCollector struct {
	queries []metricQuery
	dbConn  *sql.DB
	logger  lager.Logger
}

func (mc *sqlMetricsCollector) Collect(ctx context.Context) ([]metrics.Metric, error) {
	var metrics []metrics.Metric
	err := mc.dbConn.PingContext(ctx)
	if err != nil {
		mc.logger.Error("connecting to db", err)
		return metrics, err
	}
	for _, q := range mc.queries {
		newMetrics, err := q.getMetrics(ctx, mc.dbConn)
		if err != nil {
			mc.logger.Error("querying metrics", err, lager.Data{"query": q})
		}

		metrics = append(metrics, newMetrics...)
	}
	return metrics, nil
}

func (mc *sqlMetricsCollector) Close() error {
	return mc.dbConn.Close()
}

// MetricQueryMeta Metric meta information (Key and unit)
type metricQueryMeta struct {
	Key  string
	Unit string
}

// The query retuns one metric per column in the form:
//
// mysql> SELECT
//     ->     variable_value as connections
//     -> FROM
//     ->     performance_schema.global_status
//     -> WHERE
//     ->     variable_name = 'Threads_connected';
// +-------------+
// | connections |
// +-------------+
// | 1           |
// +-------------+
// 1 row in set (0.01 sec)
type columnMetricQuery struct {
	Query   string
	Metrics []metricQueryMeta
}

// queryToMetrics Executes the given query and retunrs the result as
// a list of Metric[]
func (q *columnMetricQuery) getMetrics(ctx context.Context, db *sql.DB) ([]metrics.Metric, error) {
	rows, err := db.QueryContext(ctx, q.Query)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %s", err)
	}
	defer rows.Close()

	rowMetrics := []metrics.Metric{}
	for rows.Next() {
		rowMap, tags, err := getRowDataAsMaps(len(q.Metrics), rows)
		if err != nil {
			return nil, err
		}
		tags["source"] = "sql"

		for _, m := range q.Metrics {
			v, ok := rowMap[m.Key]
			if !ok {
				return nil, fmt.Errorf("unable to find key '%s' in the query '%s'", m.Key, q.Query)
			}

			rowMetrics = append(rowMetrics, metrics.Metric{
				Key:   m.Key,
				Unit:  m.Unit,
				Value: v,
				Tags:  tags,
			})
		}
	}

	return rowMetrics, nil
}

// The query retuns one metric per row in the format:
//
// mysql> SHOW STATUS WHERE variable_name = 'Threads_connected';
// +-------------------+-------+
// | Variable_name     | Value |
// +-------------------+-------+
// | Threads_connected | 1     |
// +-------------------+-------+
// 1 row in set (0.09 sec)
//
type rowMetricQuery struct {
	Query   string
	Metrics []metricQueryMeta
}

// queryToMetrics Executes the given query and returns the result as
// a list of Metric[]
func (q *rowMetricQuery) getMetrics(ctx context.Context, db *sql.DB) (resultMetrics []metrics.Metric, err error) {
	rows, err := db.QueryContext(ctx, q.Query)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %s", err)
	}
	defer rows.Close()

	acumMetrics := map[string]metrics.Metric{}
	for rows.Next() {
		columnNames, err := rows.Columns()
		if err != nil {
			return resultMetrics, err
		}
		if len(columnNames) < 2 {
			return resultMetrics, fmt.Errorf("query '%s' must return at least 2 columns", q.Query)
		}

		var metricKey string
		var metricValue float64
		scanArgs := []interface{}{
			&metricKey, &metricValue,
		}

		tagsData := make([]string, len(columnNames)-2)
		for i := range tagsData {
			scanArgs = append(scanArgs, &tagsData[i])
		}

		err = rows.Scan(scanArgs...)
		if err != nil {
			return resultMetrics, err
		}

		tags := make(map[string]string, len(tagsData)+1)
		for i, v := range tagsData {
			tags[columnNames[i+2]] = v
		}
		tags["source"] = "sql"

		acumMetrics[strings.ToLower(metricKey)] = metrics.Metric{
			Value: metricValue,
			Tags:  tags,
		}
	}

	for _, m := range q.Metrics {
		v, ok := acumMetrics[m.Key]
		if !ok {
			return resultMetrics, fmt.Errorf("unable to find key '%s' in the query '%s'", m.Key, q.Query)
		}

		resultMetrics = append(resultMetrics, metrics.Metric{
			Key:   m.Key,
			Unit:  m.Unit,
			Value: v.Value,
			Tags:  v.Tags,
		})
	}

	return resultMetrics, nil
}

// Helpers

// getRowDataAsMaps Returns a sql.Rows row and returns two maps with values
// as map[string]float64 or tags as map[string]string.
//
// Values should be returned as the first columns. You must pass the expected number
// of values in the query.
//
func getRowDataAsMaps(numberOfValues int, rows *sql.Rows) (valuesMap map[string]float64, tagsMap map[string]string, err error) {
	valuesMap = make(map[string]float64)
	tagsMap = make(map[string]string)

	columnNames, err := rows.Columns()
	if err != nil {
		return valuesMap, tagsMap, err
	}

	if len(columnNames) < numberOfValues {
		return valuesMap, tagsMap, fmt.Errorf("Expected %d values but the row only has %v columns", numberOfValues, len(columnNames))
	}

	valuesData := make([]float64, numberOfValues)
	tagsData := make([]string, len(columnNames)-numberOfValues)
	var scanArgs = make([]interface{}, len(columnNames))
	for i := range scanArgs {
		if i < numberOfValues {
			scanArgs[i] = &valuesData[i]
		} else {
			scanArgs[i] = &tagsData[i-numberOfValues]
		}

	}
	err = rows.Scan(scanArgs...)
	if err != nil {
		return valuesMap, tagsMap, err
	}

	for i, v := range valuesData {
		valuesMap[columnNames[i]] = v
	}
	for i, v := range tagsData {
		tagsMap[columnNames[numberOfValues+i]] = v
	}

	return valuesMap, tagsMap, nil
}
