package emitter

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager/v3"
	"github.com/golang/protobuf/proto"

	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

// WithTimestamp overrides an envelope timestamp
func WithTimestamp(timestamp int64) loggregator.EmitGaugeOption {
	return func(m proto.Message) {
		switch e := m.(type) {
		case *loggregator_v2.Envelope:
			e.Timestamp = timestamp
		default:
			panic(fmt.Sprintf("unsupported Message type: %T", m))
		}
	}
}

type LoggregatorEmitter struct {
	loggregatorIngressClient *loggregator.IngressClient
	logger                   lager.Logger
}

func NewLoggregatorEmitter(
	emitterConfig config.LoggregatorEmitterConfig,
	logger lager.Logger,
) (*LoggregatorEmitter, error) {
	logger.Debug("new_loggregator_emitter", lager.Data{
		"ca_cert":     emitterConfig.CACertPath,
		"client_cert": emitterConfig.CertPath,
		"client_key":  emitterConfig.KeyPath,
		"url":         emitterConfig.MetronURL,
	})

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
		logger:                   logger,
	}, nil
}

func (e *LoggregatorEmitter) Emit(me metrics.MetricEnvelope) {
	e.logger.Debug("emit", lager.Data{
		"envelope": me,
	})
	var timestamp int64
	if me.Metric.Timestamp != 0 {
		timestamp = me.Metric.Timestamp
	} else {
		timestamp = time.Now().UnixNano()
	}
	e.loggregatorIngressClient.EmitGauge(
		loggregator.WithGaugeValue(me.Metric.Key, me.Metric.Value, me.Metric.Unit),
		loggregator.WithGaugeSourceInfo(me.InstanceGUID, "0"),
		WithTimestamp(timestamp),
		loggregator.WithEnvelopeTags(me.Metric.Tags),
	)
}
