package emitter_test

import (
	"github.com/alphagov/paas-rds-metric-collector/pkg/emitter"
	"github.com/alphagov/paas-rds-metric-collector/pkg/helpers"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IngressClient", func() {
	var (
		server             *helpers.FakeLoggregatorIngressServer
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

		loggregatorEmitter, err = emitter.NewLoggregatorEmitter(
			server.addr,
			"./fixtures/CA.crt",
			"./fixtures/client.crt",
			"./fixtures/client.key",
			logger,
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		server.Stop()
	})

	// FIXME: OR should it??? Fails coverage
	It("should not fail if the loggregator servers is down", func() {
		_, err := emitter.NewLoggregatorEmitter(
			"bananas://localhost:123",
			"./fixtures/CA.crt",
			"./fixtures/client.crt",
			"./fixtures/client.key",
			logger,
		)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fail if any of the cert files is missing", func() {
		var err error
		_, err = emitter.NewLoggregatorEmitter(
			"localhost:123",
			"missing",
			"./fixtures/client.crt",
			"./fixtures/client.key",
			logger,
		)
		Expect(err).To(HaveOccurred())
		_, err = emitter.NewLoggregatorEmitter(
			"localhost:123",
			"./fixtures/CA.crt",
			"missing",
			"./fixtures/client.key",
			logger,
		)
		Expect(err).To(HaveOccurred())
		_, err = emitter.NewLoggregatorEmitter(
			"localhost:123",
			"./fixtures/CA.crt",
			"./fixtures/client.crt",
			"missing",
			logger,
		)
		Expect(err).To(HaveOccurred())
	})

	It("should fail if any of the cert files is invalid", func() {
		var err error
		_, err = emitter.NewLoggregatorEmitter(
			"localhost:123",
			"./fixtures/invalid-cert.data",
			"./fixtures/client.crt",
			"./fixtures/client.key",
			logger,
		)
		Expect(err).To(HaveOccurred())
		_, err = emitter.NewLoggregatorEmitter(
			"localhost:123",
			"./fixtures/CA.crt",
			"./fixtures/invalid-cert.data",
			"./fixtures/client.key",
			logger,
		)
		Expect(err).To(HaveOccurred())
		_, err = emitter.NewLoggregatorEmitter(
			"localhost:123",
			"./fixtures/CA.crt",
			"./fixtures/client.crt",
			"./fixtures/invalid-cert.data",
			logger,
		)
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
