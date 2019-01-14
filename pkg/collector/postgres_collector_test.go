package collector

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/Kount/pq-timeouts"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
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

		mainDBConn, err := sql.Open("pq-timeouts", postgresTestDatabaseConnectionURL)
		defer mainDBConn.Close()
		Expect(err).NotTo(HaveOccurred())
		_, err = mainDBConn.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
		Expect(err).NotTo(HaveOccurred())

		testDBConn, err = sql.Open("pq-timeouts", testDBConnectionString)
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
			CREATE UNIQUE INDEX title_idx ON films (title)
		`)
		Expect(err).NotTo(HaveOccurred())

		_, err = testDBConn.Exec(`
			INSERT INTO
				films(title, date_prod, kind, len)
			VALUES
				('The Shawshaxxxxxiank Redemption', '1995-02-17', 'drama', 142)
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
		dbConn, err := sql.Open("pq-timeouts", postgresTestDatabaseConnectionURL)
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

		psqlURL, err := url.Parse(testDBConnectionString)
		Expect(err).NotTo(HaveOccurred())

		address, portStr, err := net.SplitHostPort(psqlURL.Host)
		Expect(err).NotTo(HaveOccurred())
		port, err := strconv.ParseInt(portStr, 10, 64)
		Expect(err).NotTo(HaveOccurred())
		passwd, _ := psqlURL.User.Password()

		brokerInfo.On(
			"GetInstanceConnectionDetails", mock.Anything,
		).Return(
			brokerinfo.InstanceConnectionDetails{
				DBAddress:      address,
				DBPort:         port,
				DBName:         strings.TrimLeft(psqlURL.Path, "/"),
				MasterUsername: psqlURL.User.Username(),
				MasterPassword: passwd,
			}, nil,
		)

		metricsCollectorDriver = NewPostgresMetricsCollectorDriver(
			brokerInfo,
			5,
			10,
			psqlURL.Query().Get("sslmode"),
			logger,
		)

		By("Creating a new collector")
		metricsCollector, err = metricsCollectorDriver.NewCollector(
			brokerinfo.InstanceInfo{
				GUID: "instance-guid1",
				Type: "postgres",
			},
		)
		Expect(err).NotTo(HaveOccurred())

		By("Retrieving initial metrics")
		collectedMetrics, err = metricsCollector.Collect(context.Background())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := metricsCollector.Close()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return the CollectInterval", func() {
		Expect(metricsCollectorDriver.GetCollectInterval()).To(Equal(5))
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err, _ = openMultipleDBConns(ctx, 20, "pq-timeouts", postgresTestDatabaseConnectionURL)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() float64 {
			collectedMetrics, err = metricsCollector.Collect(context.Background())
			Expect(err).NotTo(HaveOccurred())
			metric = getMetricByKey(collectedMetrics, "connections")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Unit).To(Equal("conn"))
			return metric.Value
		}, 2*time.Second).Should(
			BeNumerically(">=", 20),
		)

		By("Closing again the connections")
		cancel()

		Eventually(func() float64 {
			collectedMetrics, err = metricsCollector.Collect(context.Background())
			Expect(err).NotTo(HaveOccurred())
			metric = getMetricByKey(collectedMetrics, "connections")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Unit).To(Equal("conn"))
			return metric.Value
		}, 2*time.Second).Should(
			BeNumerically("<=", initialConnections+5),
		)
	})

	It("can collect the number of maximum connections", func() {
		metric := getMetricByKey(collectedMetrics, "max_connections")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically(">=", 10))
		Expect(metric.Unit).To(Equal("conn"))
	})

	It("can collect the database size", func() {
		metric := getMetricByKey(collectedMetrics, "dbsize")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically(">=", 1))
		Expect(metric.Unit).To(Equal("byte"))
		Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
	})

	Context("pg_stat_database and pg_locks", func() {
		It("can collect the database locks and deadlocks", func() {
			metric := getMetricByKey(collectedMetrics, "deadlocks")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("lock"))
			Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
			initialDeadLocks := metric.Value

			metric = getMetricByKey(collectedMetrics, "blocked_connections")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("conn"))
			initialLockedConns := metric.Value

			By("simulating a deadlock")
			_, err := testDBConn.Exec("CREATE TABLE z AS SELECT i FROM GENERATE_SERIES(1,2) AS i")
			ctx1, cancel1 := context.WithCancel(context.Background())
			defer cancel1()
			ctx2, cancel2 := context.WithCancel(context.Background())
			defer cancel2()

			tx1, err := testDBConn.BeginTx(ctx1, nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = tx1.Exec("DELETE FROM z WHERE i = 1")
			Expect(err).NotTo(HaveOccurred())

			tx2, err := testDBConn.BeginTx(ctx2, nil)
			Expect(err).NotTo(HaveOccurred())

			_, err = tx2.Exec("DELETE FROM z WHERE i = 2")
			Expect(err).NotTo(HaveOccurred())

			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				tx2.Exec("DELETE FROM z WHERE i = 1")
			}()

			By("detecting the locked connection")
			Eventually(func() float64 {
				collectedMetrics, err := metricsCollector.Collect(context.Background())
				Expect(err).NotTo(HaveOccurred())

				metric = getMetricByKey(collectedMetrics, "blocked_connections")
				Expect(metric).ToNot(BeNil())
				Expect(metric.Unit).To(Equal("conn"))
				return metric.Value
			},
				2*time.Second,
				500*time.Millisecond,
			).Should(BeNumerically(">", initialLockedConns))

			wg.Add(1)
			go func() {
				defer wg.Done()
				tx1.Exec("DELETE FROM z WHERE i = 2")
			}()
			wg.Wait()

			By("increasing the deadlock counter")
			Eventually(func() float64 {
				collectedMetrics, err := metricsCollector.Collect(context.Background())
				Expect(err).NotTo(HaveOccurred())

				metric = getMetricByKey(collectedMetrics, "deadlocks")
				Expect(metric).ToNot(BeNil())
				Expect(metric.Unit).To(Equal("lock"))
				Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
				return metric.Value
			},
				2*time.Second,
				500*time.Millisecond,
			).Should(BeNumerically(">", initialDeadLocks))

			By("not reporting the blocked connection")
			metric = getMetricByKey(collectedMetrics, "blocked_connections")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically("==", initialLockedConns))
			Expect(metric.Unit).To(Equal("conn"))
		})

		It("can collect number of commit and rollback transactions", func() {
			metric := getMetricByKey(collectedMetrics, "commits")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("tx"))
			Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
			metric = getMetricByKey(collectedMetrics, "rollbacks")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("tx"))
			Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
		})

		It("can collect number of blocks read/hit and write/read times", func() {
			metric := getMetricByKey(collectedMetrics, "blocks_read")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("block"))
			Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
			metric = getMetricByKey(collectedMetrics, "blocks_hit")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("block"))
			Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
			metric = getMetricByKey(collectedMetrics, "read_time")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("ms"))
			Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
			metric = getMetricByKey(collectedMetrics, "write_time")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("ms"))
			Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
		})

		It("can collect the bytes in temporary files", func() {
			metric := getMetricByKey(collectedMetrics, "temp_bytes")
			Expect(metric).ToNot(BeNil())
			Expect(metric.Value).To(BeNumerically(">=", 0))
			Expect(metric.Unit).To(Equal("byte"))
			Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))
		})
	})

	It("can collect the number of sequencial and indexed scans", func() {
		metric := getMetricByKey(collectedMetrics, "seq_scan")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically("==", 0))
		Expect(metric.Unit).To(Equal("scan"))
		Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))

		initialSeqScanValue := metric.Value

		metric = getMetricByKey(collectedMetrics, "idx_scan")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically("==", 0))
		Expect(metric.Unit).To(Equal("scan"))
		Expect(metric.Tags).To(HaveKeyWithValue("dbname", testDBName))

		initialIdxScanValue := metric.Value

		dbConn, err := sql.Open("pq-timeouts", testDBConnectionString)
		defer dbConn.Close()
		_, err = dbConn.Exec("SELECT * from films")
		Expect(err).NotTo(HaveOccurred())
		_, err = dbConn.Exec("SELECT * FROM films WHERE title = 'Code Name: K.O.Z.'")
		Expect(err).NotTo(HaveOccurred())
		dbConn.Close() // Needed for the stats collector to

		// Wait for the stat collector to write the value down
		Eventually(func() float64 {
			collectedMetrics, err = metricsCollector.Collect(context.Background())
			Expect(err).NotTo(HaveOccurred())

			metric = getMetricByKey(collectedMetrics, "seq_scan")
			Expect(metric).ToNot(BeNil())
			return metric.Value
		},
			2*time.Second,
			500*time.Millisecond,
		).Should(BeNumerically(">", initialSeqScanValue))

		metric = getMetricByKey(collectedMetrics, "idx_scan")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically(">", initialIdxScanValue))
	})

	It("can collect the maximum transaction age", func() {
		ctx1, cancel1 := context.WithCancel(context.Background())
		defer cancel1()
		_, err := testDBConn.BeginTx(ctx1, nil)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(1 * time.Second)

		collectedMetrics, err = metricsCollector.Collect(context.Background())
		metric := getMetricByKey(collectedMetrics, "max_tx_age")
		Expect(metric).ToNot(BeNil())
		Expect(metric.Value).To(BeNumerically(">=", 1))
		Expect(metric.Unit).To(Equal("s"))
	})
})

var _ = Describe("postgresConnectionStringBuilder.ConnectionString()", func() {
	It("returns the proper connection string for postgres", func() {
		details := brokerinfo.InstanceConnectionDetails{
			DBAddress:      "endpoint-address.example.com",
			DBPort:         5432,
			DBName:         "dbprefix-db",
			MasterUsername: "master-username",
			MasterPassword: "9Fs6CWnuwf0BAY3rDFAels3OXANSo0-M",
		}
		builder := postgresConnectionStringBuilder{
			ConnectionTimeout: 10,
			ReadTimeout:       11,
			WriteTimeout:      12,
			SSLMode:           "require",
		}
		connectionString := builder.ConnectionString(details)
		Expect(connectionString).To(Equal("postgresql://master-username:9Fs6CWnuwf0BAY3rDFAels3OXANSo0-M@endpoint-address.example.com:5432/dbprefix-db?sslmode=require&connect_timeout=10&read_timeout=11000&write_timeout=12000"))
	})

	It("should timeout postgres connection", func() {
		details := brokerinfo.InstanceConnectionDetails{
			DBAddress:      "1.2.3.4",
			DBPort:         5678,
			DBName:         "dbprefix-db",
			MasterUsername: "master-username",
			MasterPassword: "9Fs6CWnuwf0BAY3rDFAels3OXANSo0-M",
		}
		builder := postgresConnectionStringBuilder{ConnectionTimeout: 1}
		connectionString := builder.ConnectionString(details)

		startTime := time.Now()

		dbConn, err := sql.Open("pq-timeouts", connectionString)
		defer dbConn.Close()
		Expect(err).NotTo(HaveOccurred())

		err = dbConn.Ping()
		Expect(err).To(HaveOccurred())

		endTime := time.Now()

		Expect(endTime).To(BeTemporally("~", startTime, 2*time.Second))
	})
})
