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

		TestProvisionGetMetricsDeprovision := func(serviceID string) {
			var (
				instanceID string
				planID     string
			)

			BeforeEach(func() {
				instanceID = uuid.NewV4().String()
				planID = fmt.Sprintf("%s-micro-without-snapshot", serviceID)
			})

			AfterEach(func() {
				deprovisionInstance(instanceID, serviceID, planID)
			})

			It("Should send metrics for the created instances", func() {
				By(fmt.Sprintf("creating a %s instance", serviceID))
				provisionInstance(instanceID, serviceID, planID)

				By("checking that collector discovers the instance and emits metrics")
				Eventually(rdsMetricsCollectorSession, 60*time.Second).Should(
					gbytes.Say(
						fmt.Sprintf(`scheduler.started_worker.*"driver":"%s"`, serviceID),
					),
				)

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
				cloudwatchEnvelopes := filterEnvelopesBySourceAndTag(envelopes, instanceID, "source", "sql")
				Expect(cloudwatchEnvelopes).ToNot(BeEmpty())

				cloudwatchEnvelopes = filterEnvelopesBySourceAndTag(envelopes, instanceID, "source", "cloudwatch")
				Expect(cloudwatchEnvelopes).ToNot(BeEmpty())

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
		}

		Describe("Postgres", func() {
			TestProvisionGetMetricsDeprovision("postgres")
		})

		Describe("MySQL", func() {
			TestProvisionGetMetricsDeprovision("mysql")
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

func filterEnvelopesBySourceAndTag(
	envelopes []*loggregator_v2.Envelope,
	sourceId string,
	tagKey string,
	tagValue string,
) []*loggregator_v2.Envelope {
	filteredEnvelopes := []*loggregator_v2.Envelope{}
	for _, e := range envelopes {
		if e.GetSourceId() == sourceId {
			if v, ok := e.GetTags()[tagKey]; ok && v == tagValue {
				filteredEnvelopes = append(filteredEnvelopes, e)
			}
		}
	}
	return filteredEnvelopes
}
