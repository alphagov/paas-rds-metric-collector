package emitter

import (
	"code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

type LoggregatorEmitter struct {
	loggregatorIngressClient *loggregator.IngressClient
	logger                   lager.Logger
}

func NewLoggregatorEmitter(
	emitterConfig config.LoggregatorEmitterConfig,
	logger lager.Logger,
) (*LoggregatorEmitter, error) {
	tlsConfig, err := loggregator.NewIngressTLSConfig(
		emitterConfig.CACertPath,
		emitterConfig.CertPath,
		emitterConfig.KeyPath,
	)
	if err != nil {
		logger.Error("creating loggregator TLS config", err)
		return nil, err
	}

	client, err := loggregator.NewIngressClient(
		tlsConfig,
		loggregator.WithAddr(emitterConfig.MetronURL),
		loggregator.WithTag("origin", "rds-metrics-collector"),
	)
	if err != nil {
		logger.Error("Could not create loggregator client", err, lager.Data{"metron_url": emitterConfig.MetronURL})
		return nil, err
	}

	return &LoggregatorEmitter{
		loggregatorIngressClient: client,
		logger: logger,
	}, nil
}

func (e *LoggregatorEmitter) Emit(me metrics.MetricEnvelope) {
	e.loggregatorIngressClient.EmitGauge(
		loggregator.WithGaugeValue(me.Metric.Key, me.Metric.Value, me.Metric.Unit),
		loggregator.WithGaugeSourceInfo(me.InstanceGUID, "0"),
	)
}
