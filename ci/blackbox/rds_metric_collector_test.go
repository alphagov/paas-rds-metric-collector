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
						fmt.Sprintf(`scheduler.start_worker.*"driver":"%s"`, serviceID),
					),
				)

				By("receiving some envelopes in the fake loggregator server")
				envelopes := receiveEnvelopes(30, 60*time.Second)
				Expect(envelopes).ToNot(BeEmpty())

				By("checking we received the expected metrics")
				cloudwatchEnvelopes := filterEnvelopesBySourceAndTag(envelopes, instanceID, "source", "sql")
				Expect(cloudwatchEnvelopes).ToNot(BeEmpty())

				cloudwatchEnvelopes = filterEnvelopesBySourceAndTag(envelopes, instanceID, "source", "cloudwatch")
				Expect(cloudwatchEnvelopes).ToNot(BeEmpty())

				By("reboot the instance")
				rebootInstance(instanceID, serviceID, planID)

				By("receiving some envelopes in the fake loggregator server")
				// Flush the channel
				for len(fakeLoggregatorServer.ReceivedEnvelopes) > 0 {
					<-fakeLoggregatorServer.ReceivedEnvelopes
				}
				envelopes = receiveEnvelopes(30, 60*time.Second)

				cloudwatchEnvelopes = filterEnvelopesBySourceAndTag(envelopes, instanceID, "source", "sql")
				Expect(cloudwatchEnvelopes).ToNot(BeEmpty())

				cloudwatchEnvelopes = filterEnvelopesBySourceAndTag(envelopes, instanceID, "source", "cloudwatch")
				Expect(cloudwatchEnvelopes).ToNot(BeEmpty())

				By("deprovision the instance")
				deprovisionInstance(instanceID, serviceID, planID)

				By("checking that collector stops collecting metrics from the instance")
				Eventually(rdsMetricsCollectorSession, 60*time.Second).Should(gbytes.Say("scheduler.stop_worker"))

				By("not receiving more metrics in the fake loggregator server")
				// Flush the channel for 60 seconds (wait for the collector to stop the workers)
				_ = receiveEnvelopes(10000, 60*time.Second)

				Consistently(fakeLoggregatorServer.ReceivedEnvelopes, 30*time.Second).ShouldNot(Receive())
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

func receiveEnvelopes(max int, timeout time.Duration) []*loggregator_v2.Envelope {
	envelopes := []*loggregator_v2.Envelope{}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
loop:
	for {
		select {
		case e := <-fakeLoggregatorServer.ReceivedEnvelopes:
			fmt.Fprintf(GinkgoWriter, "Received envelope: %v\n", e)
			envelopes = append(envelopes, e)
			if len(envelopes) >= max {
				break
			}
		case <-timer.C:
			break loop
		}
	}

	return envelopes
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
