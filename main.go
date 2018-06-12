package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"

	_ "github.com/lib/pq"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-rds-broker/awsrds"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/emitter"
	"github.com/alphagov/paas-rds-metric-collector/pkg/scheduler"
)

var (
	configFilePath string

	logLevels = map[string]lager.LogLevel{
		"DEBUG": lager.DEBUG,
		"INFO":  lager.INFO,
		"ERROR": lager.ERROR,
		"FATAL": lager.FATAL,
	}
)

func init() {
	flag.StringVar(&configFilePath, "config", "", "Location of the config file")
}

func buildDBInstance(region string, logger lager.Logger) awsrds.DBInstance {
	awsConfig := aws.NewConfig().WithRegion(region)
	awsSession := session.New(awsConfig)
	rdssvc := rds.New(awsSession)
	return awsrds.NewRDSDBInstance(region, "aws", rdssvc, logger)
}

func stopOnSignal(metricsScheduler *scheduler.Scheduler) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	<-signalChan
	metricsScheduler.Stop()
	os.Exit(1)
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

	dbInstance := buildDBInstance(cfg.AWS.Region, logger)

	rdsBrokerInfo := brokerinfo.NewRDSBrokerInfo(
		cfg.RDSBrokerInfo,
		dbInstance,
		logger.Session("brokerinfo", lager.Data{"broker_name": cfg.RDSBrokerInfo.BrokerName}),
	)

	loggregatorEmitter, err := emitter.NewLoggregatorEmitter(
		cfg.LoggregatorEmitter,
		logger.Session("loggregator_emitter", lager.Data{"url": cfg.LoggregatorEmitter.MetronURL}),
	)
	if err != nil {
		logger.Error("connecting to loggregator", err)
		os.Exit(1)
	}

	postgresMetricsCollectorDriver := collector.NewPostgresMetricsCollectorDriver(
		rdsBrokerInfo,
		logger.Session("postgres_metrics_collector"),
	)

	scheduler := scheduler.NewScheduler(
		cfg.Scheduler,
		rdsBrokerInfo,
		loggregatorEmitter,
		logger.Session("scheduler"),
	)

	go stopOnSignal(scheduler)

	err = scheduler.Start()
	if err != nil {
		logger.Error("starting scheduler", err)
		os.Exit(1)
	}

	logger.Info("start")

	runtime.Goexit()
}
