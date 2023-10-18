package integration_rds_metric_collector_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/phayes/freeport"

	. "github.com/alphagov/paas-rds-broker/ci/helpers"

	rdsconfig "github.com/alphagov/paas-rds-broker/config"
	collectorconfig "github.com/alphagov/paas-rds-metric-collector/pkg/config"
)

func startNewBroker(rdsBrokerConfig *rdsconfig.Config) (*gexec.Session, *BrokerAPIClient, *RDSClient) {
	configFile, err := ioutil.TempFile("", "rds-broker")
	Expect(err).ToNot(HaveOccurred())
	defer os.Remove(configFile.Name())

	// start the broker in a random port
	rdsBrokerPort := freeport.GetPort()
	rdsBrokerConfig.Port = rdsBrokerPort

	configJSON, err := json.Marshal(rdsBrokerConfig)
	Expect(err).ToNot(HaveOccurred())
	Expect(ioutil.WriteFile(configFile.Name(), configJSON, 0644)).To(Succeed())
	Expect(configFile.Close()).To(Succeed())

	command := exec.Command("paas-rds-broker",
		fmt.Sprintf("-config=%s", configFile.Name()),
	)
	rdsBrokerSession, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())

	// Wait for it to be listening
	Eventually(rdsBrokerSession, 10*time.Second).Should(And(
		gbytes.Say(fmt.Sprintf(`{"port":%d}`, rdsBrokerPort)),
	))

	Consistently(rdsBrokerSession, 3*time.Second).ShouldNot(gexec.Exit())

	rdsBrokerUrl := fmt.Sprintf("http://localhost:%d", rdsBrokerPort)

	brokerAPIClient := NewBrokerAPIClient(rdsBrokerUrl, rdsBrokerConfig.Username, rdsBrokerConfig.Password)
	brokerAPIClient.AcceptsIncomplete = true
	rdsClient, err := NewRDSClient(rdsBrokerConfig.RDSConfig.Region, rdsBrokerConfig.RDSConfig.DBPrefix)
	Expect(err).ToNot(HaveOccurred())

	return rdsBrokerSession, brokerAPIClient, rdsClient
}

func startNewCollector(rdsMetricCollectorConfig *collectorconfig.Config) *gexec.Session {
	configFile, err := ioutil.TempFile("", "rds-collector")
	Expect(err).ToNot(HaveOccurred())
	defer os.Remove(configFile.Name())

	configJSON, err := json.Marshal(rdsMetricCollectorConfig)
	Expect(err).ToNot(HaveOccurred())
	Expect(ioutil.WriteFile(configFile.Name(), configJSON, 0644)).To(Succeed())
	Expect(configFile.Close()).To(Succeed())

	// start the collector
	command := exec.Command(rdsMetricCollectorPath,
		fmt.Sprintf("-config=%s", configFile.Name()),
	)
	rdsMetricsCollectorSession, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())

	// Wait for it to start
	Eventually(rdsMetricsCollectorSession, 10*time.Second).Should(And(
		gbytes.Say("rds-metric-collector.scheduler.scheduler-started"),
	))

	Consistently(rdsMetricsCollectorSession, 3*time.Second).ShouldNot(gexec.Exit())

	return rdsMetricsCollectorSession
}

const (
	INSTANCE_CREATE_TIMEOUT = 30 * time.Minute
)

// Keep track of provisioned instances to not deprovision instances twice
var provisionedInstances = map[string]bool{}

func provisionInstance(instanceID, serviceID, planID string) {
	code, operation, err := brokerAPIClient.ProvisionInstance(instanceID, serviceID, planID, "{}")
	Expect(err).ToNot(HaveOccurred())
	Expect(code).To(Equal(202))
	pollForOperationCompletion(brokerAPIClient, instanceID, serviceID, planID, operation)
	provisionedInstances[instanceID] = true
}

func deprovisionInstance(instanceID, serviceID, planID string) {
	if provisionedInstances[instanceID] {
		brokerAPIClient.AcceptsIncomplete = true
		code, operation, err := brokerAPIClient.DeprovisionInstance(instanceID, serviceID, planID)
		Expect(err).ToNot(HaveOccurred())
		Expect(code).To(SatisfyAny(Equal(200), Equal(202), Equal(401)))
		state := pollForOperationCompletion(brokerAPIClient, instanceID, serviceID, planID, operation)
		Expect(state).To(Equal("gone"))

		provisionedInstances[instanceID] = false
	}
}

func rebootInstance(instanceID, serviceID, planID string) {
	code, operation, description, err := brokerAPIClient.UpdateInstance(instanceID, serviceID, planID, planID, `{ "reboot": true }`)
	Expect(description).NotTo(BeEmpty())
	Expect(err).ToNot(HaveOccurred())
	Expect(code).To(Equal(202))
	pollForOperationCompletion(brokerAPIClient, instanceID, serviceID, planID, operation)
	provisionedInstances[instanceID] = true
}

func pollForOperationCompletion(brokerAPIClient *BrokerAPIClient, instanceID, serviceID, planID, operation string) string {
	var state string
	var err error

	fmt.Fprint(GinkgoWriter, "Polling for Instance Operation to complete")
	Eventually(
		func() string {
			fmt.Fprint(GinkgoWriter, ".")
			state, err = brokerAPIClient.GetLastOperationState(instanceID, serviceID, planID, operation)
			Expect(err).ToNot(HaveOccurred())
			return state
		},
		INSTANCE_CREATE_TIMEOUT,
		15*time.Second,
	).Should(
		SatisfyAny(
			Equal("succeeded"),
			Equal("failed"),
			Equal("gone"),
		),
	)

	fmt.Fprintf(GinkgoWriter, "done. Final state: %s.\n", state)
	return state
}
