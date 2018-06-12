package collector

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo/fakebrokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

var testQueries = []MetricQuery{
	{
		Query: "SELECT 1::integer as foo, '2'::varchar as bar, 3::double precision as baz",
		Metrics: []MetricQueryMeta{
			{Key: "foo", Unit: "b"},
			{Key: "bar", Unit: "s"},
			{Key: "baz", Unit: "conn"},
		},
	},
	{
		Query: "SELECT 1::integer as foo2",
		Metrics: []MetricQueryMeta{
			{Key: "foo2", Unit: "gauge"},
		},
	},
	{
		Query: "SELECT 1::integer as foo",
		Metrics: []MetricQueryMeta{
			{Key: "powah", Unit: "gauge"},
		},
	},
	{
		Query: "SELECT * FROM hell",
	},
	{
		Query: "SELECT 'Hello World'",
	},
}

var _ = Describe("sql_collector", func() {

	var (
		brokerInfo             *fakebrokerinfo.FakeBrokerInfo
		metricsCollectorDriver *sqlMetricsCollectorDriver
	)
	BeforeEach(func() {
		brokerInfo = &fakebrokerinfo.FakeBrokerInfo{}
		metricsCollectorDriver = &sqlMetricsCollectorDriver{
			queries:    testQueries,
			driver:     "postgres", // valid driver for testing
			brokerInfo: brokerInfo,
			name:       "sql",
			logger:     logger,
		}
	})

	Context("sqlMetricsCollectorDriver", func() {
		It("fails on error creating the connection string", func() {
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1"}, nil,
			)
			brokerInfo.On(
				"ConnectionString", mock.Anything,
			).Return(
				"", fmt.Errorf("failure"),
			)

			_, err := metricsCollectorDriver.NewCollector("instance-guid1")
			Expect(err).To(HaveOccurred())
		})

		It("should fail to start the collector due to invalid sql driver", func() {
			metricsCollectorDriver.driver = "invalid"
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1"}, nil,
			)
			brokerInfo.On(
				"ConnectionString", mock.Anything,
			).Return(
				"dummy", nil,
			)

			_, err := metricsCollectorDriver.NewCollector("instance-guid1")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(MatchRegexp("sql: unknown driver")))
		})

		It("should fail to start the collector due to database being unavailable", func() {
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1"}, nil,
			)
			brokerInfo.On(
				"ConnectionString", mock.Anything,
			).Return(
				"postgresql://postgres@localhost:3000?sslmode=disable", nil,
			)

			_, err := metricsCollectorDriver.NewCollector("instance-guid1")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(MatchRegexp("connect")))
		})

		It("can create a new sqlMetricsCollector", func() {
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1"}, nil,
			)
			brokerInfo.On(
				"ConnectionString", mock.Anything,
			).Return(
				postgresTestDatabaseConnectionUrl, nil,
			)

			_, err := metricsCollectorDriver.NewCollector("instance-guid1")
			Expect(err).NotTo(HaveOccurred())
		})

		It("shall return the name", func() {
			Expect(metricsCollectorDriver.GetName()).To(Equal("sql"))
		})
	})

	Context("sqlMetricsCollector", func() {

		var collector MetricsCollector

		BeforeEach(func() {
			var err error
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return([]string{"instance-guid1"}, nil)
			brokerInfo.On(
				"ConnectionString", mock.Anything,
			).Return(
				postgresTestDatabaseConnectionUrl, nil,
			)

			collector, err = metricsCollectorDriver.NewCollector("instance-guid1")
			Expect(err).NotTo(HaveOccurred())
		})

		It("can collect all metrics from multiple queries", func() {
			collectedMetrics, err := collector.Collect()
			Expect(err).NotTo(HaveOccurred())
			Expect(collectedMetrics).To(ConsistOf(
				metrics.Metric{Key: "foo", Value: 1, Unit: "b"},
				metrics.Metric{Key: "bar", Value: 2, Unit: "s"},
				metrics.Metric{Key: "baz", Value: 3, Unit: "conn"},
				metrics.Metric{Key: "foo2", Value: 1, Unit: "gauge"},
			))
		})

		It("closes the connection and retuns error after", func() {
			err := collector.Close()
			Expect(err).ToNot(HaveOccurred())
			_, err = collector.Collect()
			Expect(err).To(HaveOccurred())
		})
	})

})

var _ = Describe("helpers", func() {
	var dbConn *sql.DB

	BeforeEach(func() {
		var err error
		dbConn, err = sql.Open("postgres", postgresTestDatabaseConnectionUrl)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		dbConn.Close()
	})

	Context("getRowDataAsMap()", func() {
		It("should error when unexpected type from database", func() {
			rows, err := dbConn.Query("SELECT 'Hello World'")

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				_, err = getRowDataAsMap(rows)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("converting driver.Value type .+")))
			}
		})

		It("should error when no rows returned", func() {
			rows, err := dbConn.Query("SELECT 1::integer")

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				rows.Close()
				_, err = getRowDataAsMap(rows)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("Rows are closed")))
			}
		})

		It("should succeed when returning a values", func() {
			rows, err := dbConn.Query("SELECT 1::integer as foo, '2'::varchar as bar, 3::double precision as baz")

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				data, err := getRowDataAsMap(rows)
				Expect(err).NotTo(HaveOccurred())
				Expect(data).To(Equal(map[string]float64{"foo": 1.0, "bar": 2.0, "baz": 3.0}))
			}
		})
	})

	Context("queryToMetrics()", func() {
		It("should error when query is missing a required key", func() {
			_, err := queryToMetrics(dbConn, testQueries[2])

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(MatchRegexp("unable to find key")))
		})

		It("should error when query has syntax error", func() {
			_, err := queryToMetrics(dbConn, testQueries[3])

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(MatchRegexp("unable to execute query")))
		})

		It("should error when query doesn't record float", func() {
			_, err := queryToMetrics(dbConn, testQueries[4])

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(MatchRegexp("converting driver.Value type")))
		})

		It("should succeed to obtain metrics from query", func() {
			rowMetrics, err := queryToMetrics(dbConn, testQueries[0])

			Expect(err).NotTo(HaveOccurred())
			Expect(len(rowMetrics)).To(Equal(3))
			Expect(rowMetrics).To(Equal([]metrics.Metric{
				{Key: "foo", Value: 1, Unit: "b"},
				{Key: "bar", Value: 2, Unit: "s"},
				{Key: "baz", Value: 3, Unit: "conn"},
			}))
		})
	})
})
