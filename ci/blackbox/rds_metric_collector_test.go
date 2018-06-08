package integration_rds_metric_collector_test

import (
	"time"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("RDS Metrics Collector", func() {
	Describe("Instance Provision/get metrics/Deprovision", func() {
		const planID = "micro-without-snapshot"
		const serviceID = "postgres"

		var (
			instanceID string
		)

		BeforeEach(func() {
			instanceID = uuid.NewV4().String()
		})

		AfterEach(func() {
			deprovisionInstance(instanceID, serviceID, planID)
		})

		It("Should send metrics for the created instances", func() {
			By("creating a postgresql instance")
			provisionInstance(instanceID, serviceID, planID)

			By("checking that collector discovers the instance and emits metrics")
			Eventually(rdsMetricsCollectorSession, 30*time.Second).Should(gbytes.Say("scheduler.start_worker"))
			Eventually(rdsMetricsCollectorSession, 30*time.Second).Should(gbytes.Say("loggregator_emitter.emit"))

			By("receiving a metric in the fake loggregator server")
			var recv loggregator_v2.Ingress_BatchSenderServer
			Eventually(fakeLoggregator.Receivers, 30*time.Second).Should(Receive(&recv))
			envBatch, err := recv.Recv()
			Expect(err).ToNot(HaveOccurred())

			envelopes := envBatch.Batch
			connectionEnvelopes := filterEnvelopesBySourceAndMetric(envelopes, instanceID, "connections")
			Expect(connectionEnvelopes).ToNot(BeEmpty())

			Expect(envelopes[0].GetGauge()).NotTo(BeNil())
			Expect(envelopes[0].GetGauge().GetMetrics()).NotTo(BeNil())
			Expect(envelopes[0].GetGauge().GetMetrics()).To(HaveKey("connections"))
			Expect(envelopes[0].GetGauge().GetMetrics()["connections"].Value).To(BeNumerically(">=", 1))

			By("deprovision the instance")
			deprovisionInstance(instanceID, serviceID, planID)

			By("checking that collector stops collecting metrics from the instance")
			Eventually(rdsMetricsCollectorSession, 60*time.Second).Should(gbytes.Say("scheduler.stop_worker"))

			By("not receiving more metrics in the fake loggregator server")
			// Flush the channel
			for len(fakeLoggregator.Receivers) > 0 {
				<-fakeLoggregator.Receivers
			}
			Consistently(fakeLoggregator.Receivers).ShouldNot(Receive())
		})
	})
})

func filterEnvelopesBySourceAndMetric(
	envelopes []*loggregator_v2.Envelope,
	sourceId string,
	metricKey string,
) []*loggregator_v2.Envelope {
	filteredEnvelopes := []*loggregator_v2.Envelope{}
	for _, e := range envelopes {
		if e.GetSourceId() == sourceId && e.GetGauge() != nil {
			if _, ok := e.GetGauge().GetMetrics()[metricKey]; ok {
				filteredEnvelopes = append(filteredEnvelopes, e)
			}
		}
	}
	return filteredEnvelopes
}
