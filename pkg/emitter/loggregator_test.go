package emitter_test

import (
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
			"./fixtures/server.crt",
			"./fixtures/server.key",
			"./fixtures/CA.crt",
		)
		Expect(err).NotTo(HaveOccurred())

		err = server.Start()
		Expect(err).NotTo(HaveOccurred())

		emitterConfig = config.LoggregatorEmitterConfig{
			MetronURL:  server.Addr,
			CACertPath: "./fixtures/CA.crt",
			CertPath:   "./fixtures/client.crt",
			KeyPath:    "./fixtures/client.key",
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

		envelopes, err := server.GetEnvelopes()
		Expect(err).NotTo(HaveOccurred())
		Expect(envelopes).To(HaveLen(1))
		Expect(envelopes[0].GetSourceId()).To(Equal("instance-guid"))
		Expect(envelopes[0].GetGauge()).NotTo(BeNil())
		Expect(envelopes[0].GetGauge().GetMetrics()).NotTo(BeNil())
		Expect(envelopes[0].GetGauge().GetMetrics()).To(HaveKey("a_key"))
		Expect(envelopes[0].GetGauge().GetMetrics()["a_key"].Value).To(Equal(1.0))
		Expect(envelopes[0].GetGauge().GetMetrics()["a_key"].Unit).To(Equal("bytes"))
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

		envelopes, err := server.GetEnvelopes()
		Expect(err).NotTo(HaveOccurred())
		Expect(envelopes).To(HaveLen(3))

		Expect(envelopes[0].GetSourceId()).To(Equal("instance-guid-0"))
		Expect(envelopes[0].GetGauge()).NotTo(BeNil())
		Expect(envelopes[0].GetGauge().GetMetrics()).NotTo(BeNil())
		Expect(envelopes[0].GetGauge().GetMetrics()).To(HaveKey("size"))
		Expect(envelopes[0].GetGauge().GetMetrics()["size"].Value).To(Equal(1.0))
		Expect(envelopes[0].GetGauge().GetMetrics()["size"].Unit).To(Equal("bytes"))

		Expect(envelopes[1].GetSourceId()).To(Equal("instance-guid-1"))
		Expect(envelopes[1].GetGauge()).NotTo(BeNil())
		Expect(envelopes[1].GetGauge().GetMetrics()).NotTo(BeNil())
		Expect(envelopes[1].GetGauge().GetMetrics()).To(HaveKey("time"))
		Expect(envelopes[1].GetGauge().GetMetrics()["time"].Value).To(Equal(2.0))
		Expect(envelopes[1].GetGauge().GetMetrics()["time"].Unit).To(Equal("ms"))

		Expect(envelopes[2].GetSourceId()).To(Equal("instance-guid-2"))
		Expect(envelopes[2].GetGauge()).NotTo(BeNil())
		Expect(envelopes[2].GetGauge().GetMetrics()).NotTo(BeNil())
		Expect(envelopes[2].GetGauge().GetMetrics()).To(HaveKey("connections"))
		Expect(envelopes[2].GetGauge().GetMetrics()["connections"].Value).To(Equal(3.0))
		Expect(envelopes[2].GetGauge().GetMetrics()["connections"].Unit).To(Equal("conn"))
	})
})
