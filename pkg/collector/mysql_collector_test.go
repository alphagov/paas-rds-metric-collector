package collector

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo/fakebrokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
	"github.com/alphagov/paas-rds-metric-collector/pkg/utils"
)

var _ = Describe("NewMysqlMetricsCollectorDriver", func() {

	var (
		brokerInfo             *fakebrokerinfo.FakeBrokerInfo
		metricsCollectorDriver MetricsCollectorDriver
		metricsCollector       MetricsCollector
		collectedMetrics       []metrics.Metric
		testDBName             string
		testDBConnectionString string
	)

	BeforeEach(func() {
		testDBName = utils.RandomString(10)
		testDBConnectionString = injectDBName(mysqlTestDatabaseConnectionURL, testDBName)
		mainDBConn, err := sql.Open("mysql", mysqlTestDatabaseConnectionURL)
		defer mainDBConn.Close()
		Expect(err).NotTo(HaveOccurred())
		_, err = mainDBConn.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		dbConn, err := sql.Open("mysql", mysqlTestDatabaseConnectionURL)
		defer dbConn.Close()
		Expect(err).NotTo(HaveOccurred())

		_, err = dbConn.Query(fmt.Sprintf("DROP DATABASE %s", testDBName))
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		var err error
		brokerInfo = &fakebrokerinfo.FakeBrokerInfo{}
		metricsCollectorDriver = NewMysqlMetricsCollectorDriver(brokerInfo, logger)

		brokerInfo.On(
			"ConnectionString", mock.Anything,
		).Return(
			testDBConnectionString, nil,
		)
		By("Creating a new collector")
		metricsCollector, err = metricsCollectorDriver.NewCollector(
			brokerinfo.InstanceInfo{
				GUID: "instance-guid1",
				Type: "mysql",
			},
		)
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
		Expect(metricsCollectorDriver.GetName()).To(Equal("mysql"))
	})

	It("can collect the number of threads connected", func() {
		var err error

		By("Checking initial number of threads connected")
		metric := getMetricByKey(collectedMetrics, "threads_connected")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically(">=", 1))
		Expect(metric.Unit).To(Equal("conn"))

		initialConnections := metric.Value

		By("Creating multiple new threads_connected")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		openMultipleDBConns(ctx, 20, "mysql", mysqlTestDatabaseConnectionURL)

		Eventually(func() float64 {
			collectedMetrics, err = metricsCollector.Collect()
			Expect(err).NotTo(HaveOccurred())
			metric = getMetricByKey(collectedMetrics, "threads_connected")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Unit).To(Equal("conn"))
			return metric.Value
		}, 2*time.Second).Should(
			BeNumerically(">=", 20),
		)

		By("Closing again the threads_connected")
		cancel()

		Eventually(func() float64 {
			collectedMetrics, err = metricsCollector.Collect()
			Expect(err).NotTo(HaveOccurred())
			metric = getMetricByKey(collectedMetrics, "threads_connected")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Unit).To(Equal("conn"))
			return metric.Value
		}, 2*time.Second).Should(
			BeNumerically("<=", initialConnections+5),
		)
	})
})
