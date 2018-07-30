package emitter_test

import (
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/emitter"
	"github.com/alphagov/paas-rds-metric-collector/pkg/helpers"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IngressClient", func() {
	var (
		server             *helpers.FakeLoggregatorIngressServer
		emitterConfig      config.LoggregatorEmitterConfig
		loggregatorEmitter *emitter.LoggregatorEmitter
	)

	BeforeEach(func() {
		var err error
		server, err = helpers.NewFakeLoggregatorIngressServer(
			"../../fixtures/loggregator-server.cert.pem",
			"../../fixtures/loggregator-server.key.pem",
			"../../fixtures/ca.cert.pem",
		)
		Expect(err).NotTo(HaveOccurred())

		err = server.Start()
		Expect(err).NotTo(HaveOccurred())

		emitterConfig = config.LoggregatorEmitterConfig{
			MetronURL:  server.Addr,
			CACertPath: "../../fixtures/ca.cert.pem",
			CertPath:   "../../fixtures/client.cert.pem",
			KeyPath:    "../../fixtures/client.key.pem",
		}
		loggregatorEmitter, err = emitter.NewLoggregatorEmitter(
			emitterConfig,
			logger,
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Stop()
	})

	// FIXME: OR should it??? Fails coverage
	It("should not fail if the loggregator servers is down", func() {
		emitterConfig.MetronURL = "bananas://localhost:123"
		_, err := emitter.NewLoggregatorEmitter(emitterConfig, logger)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fail if any of the cert files is missing", func() {
		var err error

		newEmitterConfig := emitterConfig
		newEmitterConfig.CACertPath = "missing"
		_, err = emitter.NewLoggregatorEmitter(newEmitterConfig, logger)
		Expect(err).To(HaveOccurred())

		newEmitterConfig = emitterConfig
		newEmitterConfig.CertPath = "missing"
		_, err = emitter.NewLoggregatorEmitter(newEmitterConfig, logger)
		Expect(err).To(HaveOccurred())

		newEmitterConfig = emitterConfig
		newEmitterConfig.KeyPath = "missing"
		_, err = emitter.NewLoggregatorEmitter(newEmitterConfig, logger)
		Expect(err).To(HaveOccurred())
	})

	It("should fail if any of the cert files is invalid", func() {
		var err error

		newEmitterConfig := emitterConfig
		newEmitterConfig.CACertPath = "./fixtures/invalid-cert.data"
		_, err = emitter.NewLoggregatorEmitter(newEmitterConfig, logger)
		Expect(err).To(HaveOccurred())

		newEmitterConfig = emitterConfig
		newEmitterConfig.CertPath = "./fixtures/invalid-cert.data"
		_, err = emitter.NewLoggregatorEmitter(newEmitterConfig, logger)
		Expect(err).To(HaveOccurred())

		newEmitterConfig = emitterConfig
		newEmitterConfig.KeyPath = "./fixtures/invalid-cert.data"
		_, err = emitter.NewLoggregatorEmitter(newEmitterConfig, logger)
		Expect(err).To(HaveOccurred())
	})

	It("should emit one metric as gauge", func() {
		loggregatorEmitter.Emit(
			metrics.MetricEnvelope{
				InstanceGUID: "instance-guid",
				Metric:       metrics.Metric{Key: "a_key", Value: 1, Unit: "bytes"},
			},
		)

		var envelope *loggregator_v2.Envelope
		Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))
		Expect(envelope.GetTimestamp()).To(
			BeNumerically(">=", time.Now().Add(-1*time.Minute).UnixNano()),
		)
		Expect(envelope.GetSourceId()).To(Equal("instance-guid"))
		Expect(envelope.GetGauge()).NotTo(BeNil())
		Expect(envelope.GetGauge().GetMetrics()).NotTo(BeNil())
		Expect(envelope.GetGauge().GetMetrics()).To(HaveKey("a_key"))
		Expect(envelope.GetGauge().GetMetrics()["a_key"].Value).To(Equal(1.0))
		Expect(envelope.GetGauge().GetMetrics()["a_key"].Unit).To(Equal("bytes"))
	})

	It("should emit multiple metrics from different souces as gauges", func() {
		loggregatorEmitter.Emit(
			metrics.MetricEnvelope{
				InstanceGUID: "instance-guid-0",
				Metric:       metrics.Metric{Key: "size", Value: 1, Unit: "bytes"},
			},
		)
		loggregatorEmitter.Emit(
			metrics.MetricEnvelope{
				InstanceGUID: "instance-guid-1",
				Metric:       metrics.Metric{Key: "time", Value: 2, Unit: "ms"},
			},
		)
		loggregatorEmitter.Emit(
			metrics.MetricEnvelope{
				InstanceGUID: "instance-guid-2",
				Metric:       metrics.Metric{Key: "connections", Value: 3, Unit: "conn"},
			},
		)

		var envelope *loggregator_v2.Envelope
		Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))
		Expect(envelope.GetTimestamp()).To(
			BeNumerically(">=", time.Now().Add(-1*time.Minute).UnixNano()),
		)
		Expect(envelope.GetSourceId()).To(Equal("instance-guid-0"))
		Expect(envelope.GetGauge()).NotTo(BeNil())
		Expect(envelope.GetGauge().GetMetrics()).NotTo(BeNil())
		Expect(envelope.GetGauge().GetMetrics()).To(HaveKey("size"))
		Expect(envelope.GetGauge().GetMetrics()["size"].Value).To(Equal(1.0))
		Expect(envelope.GetGauge().GetMetrics()["size"].Unit).To(Equal("bytes"))

		Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))
		Expect(envelope.GetTimestamp()).To(
			BeNumerically(">=", time.Now().Add(-1*time.Minute).UnixNano()),
		)
		Expect(envelope.GetSourceId()).To(Equal("instance-guid-1"))
		Expect(envelope.GetGauge()).NotTo(BeNil())
		Expect(envelope.GetGauge().GetMetrics()).NotTo(BeNil())
		Expect(envelope.GetGauge().GetMetrics()).To(HaveKey("time"))
		Expect(envelope.GetGauge().GetMetrics()["time"].Value).To(Equal(2.0))
		Expect(envelope.GetGauge().GetMetrics()["time"].Unit).To(Equal("ms"))

		Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))
		Expect(envelope.GetTimestamp()).To(
			BeNumerically(">=", time.Now().Add(-1*time.Minute).UnixNano()),
		)
		Expect(envelope.GetSourceId()).To(Equal("instance-guid-2"))
		Expect(envelope.GetGauge()).NotTo(BeNil())
		Expect(envelope.GetGauge().GetMetrics()).NotTo(BeNil())
		Expect(envelope.GetGauge().GetMetrics()).To(HaveKey("connections"))
		Expect(envelope.GetGauge().GetMetrics()["connections"].Value).To(Equal(3.0))
		Expect(envelope.GetGauge().GetMetrics()["connections"].Unit).To(Equal("conn"))
	})

	It("should preserve the metric timestamp if it is not 0", func() {
		metricTime := time.Now().Add(-1 * time.Hour)

		loggregatorEmitter.Emit(
			metrics.MetricEnvelope{
				InstanceGUID: "instance-guid",
				Metric: metrics.Metric{
					Key:       "a_key",
					Timestamp: metricTime.UnixNano(),
					Value:     1,
					Unit:      "bytes",
				},
			},
		)

		var envelope *loggregator_v2.Envelope
		Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))

		Expect(envelope.GetTimestamp()).To(And(
			BeNumerically(">=", metricTime.Add(-1*time.Minute).UnixNano()),
			BeNumerically("<", time.Now().Add(-2*time.Minute).UnixNano()),
		))
	})

	It("should send tags if the metric has tags", func() {
		metricTime := time.Now().Add(-1 * time.Hour)

		loggregatorEmitter.Emit(
			metrics.MetricEnvelope{
				InstanceGUID: "instance-guid",
				Metric: metrics.Metric{
					Key:       "a_key",
					Timestamp: metricTime.UnixNano(),
					Value:     1,
					Unit:      "bytes",
					Tags: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
				},
			},
		)

		var envelope *loggregator_v2.Envelope
		Eventually(server.ReceivedEnvelopes, 1*time.Second).Should(Receive(&envelope))

		Expect(envelope.GetTags()).To(HaveKeyWithValue("key1", "val1"))
		Expect(envelope.GetTags()).To(HaveKeyWithValue("key2", "val2"))
	})
})
