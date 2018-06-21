package collector

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo/fakebrokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
	"github.com/alphagov/paas-rds-metric-collector/pkg/utils"
)

var _ = Describe("NewPostgresMetricsCollectorDriver", func() {

	var (
		brokerInfo             *fakebrokerinfo.FakeBrokerInfo
		metricsCollectorDriver MetricsCollectorDriver
		metricsCollector       MetricsCollector
		collectedMetrics       []metrics.Metric
		testDBName             string
		testDBConnectionString string
		testDBConn             *sql.DB
	)

	BeforeEach(func() {
		testDBName = utils.RandomString(10)
		testDBConnectionString = injectDBName(postgresTestDatabaseConnectionURL, testDBName)

		mainDBConn, err := sql.Open("postgres", postgresTestDatabaseConnectionURL)
		defer mainDBConn.Close()
		Expect(err).NotTo(HaveOccurred())
		_, err = mainDBConn.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
		Expect(err).NotTo(HaveOccurred())

		testDBConn, err = sql.Open("postgres", testDBConnectionString)
		Expect(err).NotTo(HaveOccurred())
		_, err = testDBConn.Exec(`
			CREATE TABLE films (
					id          SERIAL NOT NULL,
					title       varchar(40) NOT NULL,
					date_prod   date,
					kind        varchar(10),
					len         integer
			)
		`)
		Expect(err).NotTo(HaveOccurred())
		_, err = testDBConn.Exec(`
			INSERT INTO
				films(title, date_prod, kind, len)
			VALUES
				('The Shawshank Redemption', '1995-02-17', 'drama', 142)
		`)
		Expect(err).NotTo(HaveOccurred())
		_, err = testDBConn.Exec(`
			INSERT INTO
				films(title, date_prod, kind, len)
			VALUES
				('Code Name: K.O.Z.', '2015-02-13', 'crime', 114)
		`)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		testDBConn.Close()
		dbConn, err := sql.Open("postgres", postgresTestDatabaseConnectionURL)
		defer dbConn.Close()
		Expect(err).NotTo(HaveOccurred())
		// Kill all connections to this DB, as sql.DB keeps a pool and it
		// does not close all, preventing the DROP DATABASE from working.
		// FIXME: Why I cannot use a Prepare parametrized query here??
		_, err = dbConn.Query(fmt.Sprintf(`
			SELECT pg_terminate_backend(pg_stat_activity.pid)
			FROM pg_stat_activity
			WHERE datname = '%s'
		`, testDBName))
		Expect(err).NotTo(HaveOccurred())

		_, err = dbConn.Query(fmt.Sprintf("DROP DATABASE %s", testDBName))
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		var err error
		brokerInfo = &fakebrokerinfo.FakeBrokerInfo{}
		metricsCollectorDriver = NewPostgresMetricsCollectorDriver(brokerInfo, logger)

		brokerInfo.On(
			"ConnectionString", mock.Anything,
		).Return(
			testDBConnectionString, nil,
		)
		By("Creating a new collector")
		metricsCollector, err = metricsCollectorDriver.NewCollector("instance-guid1")
		Expect(err).NotTo(HaveOccurred())

		By("Retrieving initial metrics")
		collectedMetrics, err = metricsCollector.Collect()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := metricsCollector.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns the right metricsCollectorDriver name", func() {
		Expect(metricsCollectorDriver.GetName()).To(Equal("postgres"))
	})

	It("can collect the number of connections", func() {
		var err error

		By("Checking initial number of connections")
		metric := getMetricByKey(collectedMetrics, "connections")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically(">=", 1))
		Expect(metric.Unit).To(Equal("conn"))

		initialConnections := metric.Value

		By("Creating multiple new connections")
		closeDBConns := openMultipleDBConns(20, "postgres", postgresTestDatabaseConnectionURL)
		defer closeDBConns()

		Eventually(func() float64 {
			collectedMetrics, err = metricsCollector.Collect()
			Expect(err).NotTo(HaveOccurred())
			metric = getMetricByKey(collectedMetrics, "connections")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Unit).To(Equal("conn"))
			return metric.Value
		}, 2*time.Second).Should(
			BeNumerically(">=", 20),
		)

		By("Closing again the connections")
		closeDBConns()

		Eventually(func() float64 {
			collectedMetrics, err = metricsCollector.Collect()
			Expect(err).NotTo(HaveOccurred())
			metric = getMetricByKey(collectedMetrics, "connections")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Unit).To(Equal("conn"))
			return metric.Value
		}, 2*time.Second).Should(
			BeNumerically("<=", initialConnections+5),
		)
	})

	It("can collect the database size", func() {
		metric := getMetricByKey(collectedMetrics, "dbsize")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically(">=", 1))
		Expect(metric.Unit).To(Equal("byte"))
		Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
	})
})

func openMultipleDBConns(count int, driver, url string) func() {
	var dbConns []*sql.DB
	success := false

	closeDBConns := func() {
		for _, c := range dbConns {
			c.Close()
		}
	}
	defer func() {
		if success != true {
			closeDBConns()
		}
	}()

	for i := 0; i < count; i++ {
		dbConn, err := sql.Open(driver, url)
		Expect(err).ToNot(HaveOccurred())
		err = dbConn.Ping()
		Expect(err).ToNot(HaveOccurred())
		dbConns = append(dbConns, dbConn)
	}
	success = true
	return closeDBConns
}

func getMetricByKey(collectedMetrics []metrics.Metric, key string) *metrics.Metric {
	for _, metric := range collectedMetrics {
		if metric.Key == key {
			return &metric
		}
	}
	return nil
}

// Replaces the DB name in a postgres DB connection string
func injectDBName(connectionString, newDBName string) string {
	re := regexp.MustCompile("(.*:[0-9]+)[^?]*([?].*)?$")
	return re.ReplaceAllString(connectionString, fmt.Sprintf("$1/%s$2", newDBName))
}

var _ = Describe("injectDBName", func() {
	It("replaces the db name", func() {
		Expect(
			injectDBName("postgresql://postgres@localhost:5432/foo?sslmode=disable", "mydb"),
		).To(Equal(
			"postgresql://postgres@localhost:5432/mydb?sslmode=disable",
		))
		Expect(
			injectDBName("postgresql://postgres@localhost:5432?sslmode=disable", "mydb"),
		).To(Equal(
			"postgresql://postgres@localhost:5432/mydb?sslmode=disable",
		))
	})
})
