package collector

import (
	"database/sql"
	"fmt"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

// MetricQueryMeta Metric meta information (Key and unit)
type MetricQueryMeta struct {
	Key  string
	Unit string
}

// MetricQuery would be holding information about our custom metric.
type MetricQuery struct {
	Query   string
	Metrics []MetricQueryMeta
}

// sqlMetricsCollectorDriver pulls metrics using generic SQL queries
type sqlMetricsCollectorDriver struct {
	brokerInfo brokerinfo.BrokerInfo
	queries    []MetricQuery
	driver     string
	name       string
	logger     lager.Logger
}

// NewCollector ...
func (d *sqlMetricsCollectorDriver) NewCollector(instanceInfo brokerinfo.InstanceInfo) (MetricsCollector, error) {
	url, err := d.brokerInfo.ConnectionString(instanceInfo)
	if err != nil {
		d.logger.Error("cannot compose connection string", err, lager.Data{
			"instanceInfo": instanceInfo,
		})
		return nil, err
	}

	dbConn, err := sql.Open(d.driver, url)
	if err != nil {
		d.logger.Error("cannot connect to the database", err, lager.Data{
			"instanceInfo": instanceInfo,
		})
		return nil, err
	}

	err = dbConn.Ping()
	if err != nil {
		d.logger.Error("cannot ping the database", err, lager.Data{
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

type sqlMetricsCollector struct {
	queries []MetricQuery
	dbConn  *sql.DB
	logger  lager.Logger
}

func (mc *sqlMetricsCollector) Collect() ([]metrics.Metric, error) {
	var metrics []metrics.Metric
	err := mc.dbConn.Ping()
	if err != nil {
		mc.logger.Error("connecting to db", err)
		return metrics, err
	}
	for _, q := range mc.queries {
		newMetrics, err := queryToMetrics(mc.dbConn, q)
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

// queryToMetrics Executes the given query and retunrs the result as
// a list of Metric[]
func queryToMetrics(db *sql.DB, mq MetricQuery) ([]metrics.Metric, error) {
	rows, err := db.Query(mq.Query)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %s", err)
	}
	defer rows.Close()

	rowMetrics := []metrics.Metric{}
	for rows.Next() {
		rowMap, tags, err := getRowDataAsMaps(len(mq.Metrics), rows)
		if err != nil {
			return nil, err
		}
		tags["source"] = "sql"

		for _, m := range mq.Metrics {
			v, ok := rowMap[m.Key]
			if !ok {
				return nil, fmt.Errorf("unable to find key '%s' in the query '%s'", m.Key, mq.Query)
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
