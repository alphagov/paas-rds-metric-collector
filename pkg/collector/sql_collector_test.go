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
		Query: `
			SELECT
				1::integer as foo,
				'2'::varchar as bar,
				3::double precision as baz,
				'val1' as tag1,
				'val2' as tag2
		`,
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
		Query: "SELECT 'Hello World' as foo2",
		Metrics: []MetricQueryMeta{
			{Key: "foo2", Unit: "gauge"},
		},
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
				postgresTestDatabaseConnectionURL, nil,
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
				postgresTestDatabaseConnectionURL, nil,
			)

			collector, err = metricsCollectorDriver.NewCollector("instance-guid1")
			Expect(err).NotTo(HaveOccurred())
		})

		It("can collect all metrics from multiple queries", func() {
			collectedMetrics, err := collector.Collect()
			Expect(err).NotTo(HaveOccurred())
			expectedTags1 := map[string]string{"source": "sql", "tag1": "val1", "tag2": "val2"}
			expectedTags2 := map[string]string{"source": "sql"}
			Expect(collectedMetrics).To(ConsistOf(
				metrics.Metric{Key: "foo", Value: 1, Unit: "b", Tags: expectedTags1},
				metrics.Metric{Key: "bar", Value: 2, Unit: "s", Tags: expectedTags1},
				metrics.Metric{Key: "baz", Value: 3, Unit: "conn", Tags: expectedTags1},
				metrics.Metric{Key: "foo2", Value: 1, Unit: "gauge", Tags: expectedTags2},
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
		dbConn, err = sql.Open("postgres", postgresTestDatabaseConnectionURL)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		dbConn.Close()
	})

	Context("getRowDataAsMaps()", func() {
		It("should error when unexpected type from database", func() {
			rows, err := dbConn.Query("SELECT 'Hello World'")

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				_, _, err = getRowDataAsMaps(1, rows)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("converting driver.Value type .+")))
			}
		})

		It("should error when no rows returned", func() {
			rows, err := dbConn.Query("SELECT 1::integer")

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				rows.Close() // Close the rows
				_, _, err = getRowDataAsMaps(1, rows)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("Rows are closed")))
			}
		})

		It("should succeed when returning values", func() {
			rows, err := dbConn.Query(`
				SELECT
					1::integer as foo,
					'2'::varchar as bar,
					3::double precision as baz
			`)

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				data, _, err := getRowDataAsMaps(3, rows)
				Expect(err).NotTo(HaveOccurred())
				Expect(data).To(Equal(map[string]float64{"foo": 1.0, "bar": 2.0, "baz": 3.0}))
			}
		})

		It("should returning only the number of values indicated", func() {
			rows, err := dbConn.Query(`
				SELECT
					1::integer as foo,
					'2'::varchar as bar,
					3::double precision as baz
			`)

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				data, _, err := getRowDataAsMaps(1, rows)
				Expect(err).NotTo(HaveOccurred())
				Expect(data).To(Equal(map[string]float64{"foo": 1.0}))
			}
		})

		It("should fail if the query does not have enough values", func() {
			rows, err := dbConn.Query("SELECT 1::integer as foo")

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				_, _, err := getRowDataAsMaps(4, rows)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp("Expected 4 values but the row only has 1 columns")))
			}
		})

		It("should succeed when returning values and tags", func() {
			rows, err := dbConn.Query(`
				SELECT
					1::integer as foo,
					'2'::varchar as bar,
					3::double precision as baz,
					'val1' as tag1,
					'val2' as tag2
			`)

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				data, tags, err := getRowDataAsMaps(3, rows)
				Expect(err).NotTo(HaveOccurred())
				Expect(data).To(Equal(map[string]float64{"foo": 1.0, "bar": 2.0, "baz": 3.0}))
				Expect(tags).To(Equal(map[string]string{"tag1": "val1", "tag2": "val2"}))
			}
		})
		It("should succeed tags is not a string", func() {
			rows, err := dbConn.Query("SELECT 1::integer as foo, 1 as tag1")

			Expect(err).NotTo(HaveOccurred())

			for rows.Next() {
				data, tags, err := getRowDataAsMaps(1, rows)
				Expect(err).NotTo(HaveOccurred())
				Expect(data).To(Equal(map[string]float64{"foo": 1.0}))
				Expect(tags).To(Equal(map[string]string{"tag1": "1"}))
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
			expectedTags := map[string]string{"source": "sql", "tag1": "val1", "tag2": "val2"}
			Expect(rowMetrics).To(Equal([]metrics.Metric{
				{Key: "foo", Value: 1, Unit: "b", Tags: expectedTags},
				{Key: "bar", Value: 2, Unit: "s", Tags: expectedTags},
				{Key: "baz", Value: 3, Unit: "conn", Tags: expectedTags},
			}))
		})
	})
})
