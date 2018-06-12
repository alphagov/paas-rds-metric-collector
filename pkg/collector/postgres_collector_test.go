package collector

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"

	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo/fakebrokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

var _ = Describe("NewPostgresMetricsCollectorDriver", func() {

	var (
		brokerInfo             *fakebrokerinfo.FakeBrokerInfo
		metricsCollectorDriver MetricsCollectorDriver
		metricsCollector       MetricsCollector
	)
	BeforeEach(func() {
		brokerInfo = &fakebrokerinfo.FakeBrokerInfo{}
		metricsCollectorDriver = NewPostgresMetricsCollectorDriver(brokerInfo, logger)
	})

	It("returns the right metricsCollectorDriver name", func() {
		Expect(metricsCollectorDriver.GetName()).To(Equal("postgres"))
	})

	It("can collect the number of connections", func() {
		var err error
		brokerInfo.On(
			"ConnectionString", mock.Anything,
		).Return(
			postgresTestDatabaseConnectionUrl, nil,
		)

		By("Creating a new collector")
		metricsCollector, err = metricsCollectorDriver.NewCollector("instance-guid1")
		Expect(err).NotTo(HaveOccurred())

		collectedMetrics, err := metricsCollector.Collect()
		Expect(err).NotTo(HaveOccurred())

		By("Retrieving initial metrics")
		metric := getMetricByKey(collectedMetrics, "connections")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically(">=", 1))
		Expect(metric.Unit).To(Equal("conn"))

		initialConnections := metric.Value

		By("Creating multiple new connections")
		closeDBConns := openMultipleDBConns(20, "postgres", postgresTestDatabaseConnectionUrl)
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
