package integration_rds_metric_collector_test

import (
	"fmt"
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
			Eventually(rdsMetricsCollectorSession, 240*time.Second).Should(gbytes.Say("cloudwatch_metrics_collector.retrieved_metric"))

			By("receiving several seconds of metrics in the fake loggregator server")

			envelopes := []*loggregator_v2.Envelope{}

			timer := time.NewTimer(60 * time.Second)
			defer timer.Stop()
		loop:
			for {
				select {
				case e := <-fakeLoggregator.ReceivedEnvelopes:
					fmt.Fprintf(GinkgoWriter, "Received envelope: %v\n", e)
					envelopes = append(envelopes, e)
				case <-timer.C:
					break loop
				}
			}

			By("checking we received the expected metrics")
			connectionEnvelopes := filterEnvelopesBySourceAndMetric(envelopes, instanceID, "connections")
			Expect(connectionEnvelopes).ToNot(BeEmpty())
			Expect(connectionEnvelopes[0].GetGauge()).NotTo(BeNil())
			Expect(connectionEnvelopes[0].GetGauge().GetMetrics()).NotTo(BeNil())
			Expect(connectionEnvelopes[0].GetGauge().GetMetrics()).To(HaveKey("connections"))
			Expect(connectionEnvelopes[0].GetGauge().GetMetrics()["connections"].Value).To(BeNumerically(">=", 1))

			cpuEnvelopes := filterEnvelopesBySourceAndMetric(envelopes, instanceID, "cpu")
			Expect(cpuEnvelopes).ToNot(BeEmpty())
			Expect(cpuEnvelopes[0].GetGauge()).NotTo(BeNil())
			Expect(cpuEnvelopes[0].GetGauge().GetMetrics()).NotTo(BeNil())
			Expect(cpuEnvelopes[0].GetGauge().GetMetrics()).To(HaveKey("cpu"))
			Expect(cpuEnvelopes[0].GetGauge().GetMetrics()["cpu"].Value).To(BeNumerically(">=", 0))

			By("deprovision the instance")
			deprovisionInstance(instanceID, serviceID, planID)

			By("checking that collector stops collecting metrics from the instance")
			Eventually(rdsMetricsCollectorSession, 60*time.Second).Should(gbytes.Say("scheduler.stop_worker"))

			By("not receiving more metrics in the fake loggregator server")
			// Flush the channel
			for len(fakeLoggregator.ReceivedEnvelopes) > 0 {
				<-fakeLoggregator.ReceivedEnvelopes
			}
			Consistently(fakeLoggregator.ReceivedEnvelopes, 30*time.Second).ShouldNot(Receive())
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
