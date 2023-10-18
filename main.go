package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	_ "github.com/lib/pq"

	"code.cloudfoundry.org/lager/v3"

	"github.com/alphagov/paas-rds-broker/awsrds"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/locket"
	"code.cloudfoundry.org/locket/lock"
	locketmodels "code.cloudfoundry.org/locket/models"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/emitter"
	"github.com/alphagov/paas-rds-metric-collector/pkg/scheduler"
	uuid "github.com/satori/go.uuid"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

var (
	configFilePath   string
	useStdoutEmitter bool

	logLevels = map[string]lager.LogLevel{
		"DEBUG": lager.DEBUG,
		"INFO":  lager.INFO,
		"ERROR": lager.ERROR,
		"FATAL": lager.FATAL,
	}
)

const (
	ConnectionTimeout = 10
	MysqlTLS          = "skip-verify"
	PostgresSSLMode   = "require"
)

func init() {
	flag.StringVar(&configFilePath, "config", "", "Location of the config file")
	flag.BoolVar(&useStdoutEmitter, "stdoutEmitter", false, "Print metrics to stdout rather than send to loggregator")
}

var logger = lager.NewLogger("rds-metric-collector")

func initLogger(logLevel string) lager.Logger {
	laggerLogLevel, ok := logLevels[strings.ToUpper(logLevel)]
	if !ok {
		log.Fatal("Invalid log level: ", logLevel)
	}

	logger.RegisterSink(lager.NewWriterSink(os.Stdout, laggerLogLevel))

	return logger
}

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatal(fmt.Sprintf("Error loading config file: '%s'. ", configFilePath), err)
	}
	initLogger(cfg.LogLevel)

	awsConfig := aws.NewConfig().WithRegion(cfg.AWS.Region)
	awsSession := session.New(awsConfig)
	rdssvc := rds.New(awsSession)
	dbInstance := awsrds.NewRDSDBInstance(cfg.AWS.Region, "aws", rdssvc, logger, time.Hour, nil)

	rdsBrokerInfo := brokerinfo.NewRDSBrokerInfo(
		cfg.RDSBrokerInfo,
		dbInstance,
		logger.Session("brokerinfo", lager.Data{"broker_name": cfg.RDSBrokerInfo.BrokerName}),
	)

	var metricsEmitter emitter.MetricsEmitter
	if useStdoutEmitter {
		metricsEmitter = &emitter.StdOutEmitter{}
	} else {
		metricsEmitter, err = emitter.NewLoggregatorEmitter(
			cfg.LoggregatorEmitter,
			logger.Session("loggregator_emitter", lager.Data{"url": cfg.LoggregatorEmitter.MetronURL}),
		)
		if err != nil {
			logger.Error("connecting to loggregator", err)
			os.Exit(1)
		}
	}

	postgresMetricsCollectorDriver := collector.NewPostgresMetricsCollectorDriver(
		rdsBrokerInfo,
		cfg.Scheduler.SQLMetricCollectorInterval,
		ConnectionTimeout,
		PostgresSSLMode,
		logger.Session("postgres_metrics_collector"),
	)

	mysqlMetricsCollectorDriver := collector.NewMysqlMetricsCollectorDriver(
		rdsBrokerInfo,
		cfg.Scheduler.SQLMetricCollectorInterval,
		ConnectionTimeout,
		MysqlTLS,
		logger.Session("mysql_metrics_collector"),
	)

	cloudWatchMetricsCollectorDriver := collector.NewCloudWatchCollectorDriver(
		cfg.Scheduler.CWMetricCollectorInterval,
		awsSession,
		rdsBrokerInfo,
		logger.Session("cloudwatch_metrics_collector"),
	)

	scheduler := scheduler.NewScheduler(
		cfg.Scheduler,
		rdsBrokerInfo,
		metricsEmitter,
		logger.Session("scheduler"),
	)
	scheduler.WithDriver(postgresMetricsCollectorDriver)
	scheduler.WithDriver(mysqlMetricsCollectorDriver)
	scheduler.WithDriver(cloudWatchMetricsCollectorDriver)

	members := []grouper.Member{}
	locketRunner := createLocketRunner(logger, cfg)

	members = append(members, grouper.Member{Name: "locketRunner", Runner: locketRunner})
	members = append(members, grouper.Member{Name: "scheduleRunner", Runner: scheduler})

	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))
	err = <-monitor.Wait()

	if err != nil {
		logger.Error("process-group-stopped-with-error", err)
		os.Exit(1)
	}
}

func createLocketRunner(logger lager.Logger, locketConfig *config.Config) ifrit.Runner {
	var (
		err          error
		locketClient locketmodels.LocketClient
	)
	logger.Debug("connecting-to-locket")
	locketClient, err = locket.NewClient(logger, locketConfig.ClientLocketConfig)
	if err != nil {
		logger.Fatal("Failed to initialize locket client", err)
	}
	logger.Debug("connected-to-locket")
	id := uuid.NewV4()

	lockIdentifier := &locketmodels.Resource{
		Key:   "rds-metrics-collector",
		Owner: id.String(),
		Type:  locketmodels.LockType,
	}

	return lock.NewLockRunner(
		logger,
		locketClient,
		lockIdentifier,
		locket.DefaultSessionTTLInSeconds,
		clock.NewClock(),
		locket.SQLRetryInterval,
	)
}
