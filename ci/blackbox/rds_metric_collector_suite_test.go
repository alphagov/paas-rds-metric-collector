package integration_rds_metric_collector_test

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"

	rdsconfig "github.com/alphagov/paas-rds-broker/config"
	collectorconfig "github.com/alphagov/paas-rds-metric-collector/pkg/config"

	. "github.com/alphagov/paas-rds-broker/ci/helpers"
	"github.com/alphagov/paas-rds-metric-collector/pkg/helpers"
)

var (
	rdsSubnetGroupName *string
	ec2SecurityGroupID *string

	rdsBrokerPath    string
	rdsBrokerConfig  *rdsconfig.Config
	rdsBrokerSession *gexec.Session
	brokerAPIClient  *BrokerAPIClient
	rdsClient        *RDSClient

	rdsMetricCollectorPath     string
	rdsMetricCollectorConfig   *collectorconfig.Config
	rdsMetricsCollectorSession *gexec.Session

	fakeLoggregator *helpers.FakeLoggregatorIngressServer
)

func TestSuite(t *testing.T) {
	BeforeSuite(func() {
		var err error

		// Compile the broker
		rdsBrokerPath, err = gexec.Build("github.com/alphagov/paas-rds-broker")
		Expect(err).ShouldNot(HaveOccurred())

		// Update config
		rdsBrokerConfig, err = rdsconfig.LoadConfig("../../fixtures/broker_config.json")
		Expect(err).ToNot(HaveOccurred())
		err = rdsBrokerConfig.Validate()
		Expect(err).ToNot(HaveOccurred())

		rdsBrokerConfig.RDSConfig.BrokerName = fmt.Sprintf("%s-%s",
			rdsBrokerConfig.RDSConfig.BrokerName,
			uuid.NewV4().String(),
		)

		awsSession := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(rdsBrokerConfig.RDSConfig.Region)},
		))
		rdsSubnetGroupName, err = CreateSubnetGroup(rdsBrokerConfig.RDSConfig.DBPrefix, awsSession)
		Expect(err).ToNot(HaveOccurred())
		ec2SecurityGroupID, err = CreateSecurityGroup(rdsBrokerConfig.RDSConfig.DBPrefix, awsSession)
		Expect(err).ToNot(HaveOccurred())

		for serviceIndex := range rdsBrokerConfig.RDSConfig.Catalog.Services {
			for planIndex := range rdsBrokerConfig.RDSConfig.Catalog.Services[serviceIndex].Plans {
				plan := &rdsBrokerConfig.RDSConfig.Catalog.Services[serviceIndex].Plans[planIndex]
				plan.RDSProperties.DBSubnetGroupName = *rdsSubnetGroupName
				plan.RDSProperties.VpcSecurityGroupIds = []string{*ec2SecurityGroupID}
			}
		}

		// Start a fake server for loggregator
		fakeLoggregator, err = helpers.NewFakeLoggregatorIngressServer(
			"../../fixtures/loggregator-server.cert.pem",
			"../../fixtures/loggregator-server.key.pem",
			"../../fixtures/ca.cert.pem")
		Expect(err).ShouldNot(HaveOccurred())
		err = fakeLoggregator.Start()
		Expect(err).ShouldNot(HaveOccurred())

		// Compile the rds collector
		rdsMetricCollectorPath, err = gexec.Build("github.com/alphagov/paas-rds-metric-collector")
		Expect(err).ShouldNot(HaveOccurred())

		// Update config
		rdsMetricCollectorConfig, err = collectorconfig.LoadConfig("../../fixtures/collector_config.json")
		Expect(err).ToNot(HaveOccurred())
		rdsMetricCollectorConfig.RDSBrokerInfo.BrokerName = rdsBrokerConfig.RDSConfig.BrokerName
		rdsMetricCollectorConfig.LoggregatorEmitter.MetronURL = fakeLoggregator.Addr
		rdsMetricCollectorConfig.LoggregatorEmitter.CACertPath = "../../fixtures/ca.cert.pem"
		rdsMetricCollectorConfig.LoggregatorEmitter.CertPath = "../../fixtures/client.cert.pem"
		rdsMetricCollectorConfig.LoggregatorEmitter.KeyPath = "../../fixtures/client.key.pem"

		// Start the services
		rdsBrokerSession, brokerAPIClient, rdsClient = startNewBroker(rdsBrokerConfig)
		rdsMetricsCollectorSession = startNewCollector(rdsMetricCollectorConfig)
	})

	AfterSuite(func() {
		if fakeLoggregator != nil {
			fakeLoggregator.Stop()
		}
		if rdsBrokerSession != nil {
			rdsBrokerSession.Kill()
		}
		if rdsMetricsCollectorSession != nil {
			rdsMetricsCollectorSession.Kill()
		}

		awsSession := session.New(&aws.Config{
			Region: aws.String(rdsBrokerConfig.RDSConfig.Region)},
		)
		if ec2SecurityGroupID != nil {
			Expect(DestroySecurityGroup(ec2SecurityGroupID, awsSession)).To(Succeed())
		}
		if rdsSubnetGroupName != nil {
			Expect(DestroySubnetGroup(rdsSubnetGroupName, awsSession)).To(Succeed())
		}
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "RDS Metric Collector Suite")
}
