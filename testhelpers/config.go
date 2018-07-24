package testhelpers

import (
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"code.cloudfoundry.org/locket"
	"io/ioutil"
	"encoding/json"
	. "github.com/onsi/gomega"
)

func BuildTempConfigFile(locketAddress, fixturesPath string) (configFilePath string) {
	rdsMetricCollectorConfig := config.Config{
		LogLevel: "debug",
		AWS: config.AWSConfig{
			Region:       "eu-west-1",
			AWSPartition: "aws",
		},
		RDSBrokerInfo: config.RDSBrokerInfoConfig{
			BrokerName:         "mybroker",
			DBPrefix:           "build-test",
			MasterPasswordSeed: "something-secret",
		},
		Scheduler: config.SchedulerConfig{
			InstanceRefreshInterval: 30,
			MetricCollectorInterval: 5,
		},
		LoggregatorEmitter: config.LoggregatorEmitterConfig{
			MetronURL:  "localhost:3458",
			CACertPath: fixturesPath + "/ca.cert.pem",
			CertPath:   fixturesPath + "/client.cert.pem",
			KeyPath:    fixturesPath + "/client.key.pem",
		},
		ClientLocketConfig: locket.ClientLocketConfig{
			LocketCACertFile:     fixturesPath + "/ca.cert.pem",
			LocketClientCertFile: fixturesPath + "/client.cert.pem",
			LocketClientKeyFile:  fixturesPath + "/client.key.pem",
			LocketAddress:        locketAddress,
		},
	}
	temporaryConfigFile, err := ioutil.TempFile("", "rds-metrics-collector-config-")
	Expect(err).ToNot(HaveOccurred())
	configJSON, err := json.Marshal(rdsMetricCollectorConfig)
	Expect(err).ToNot(HaveOccurred())
	configFilePath = temporaryConfigFile.Name()
	err = ioutil.WriteFile(configFilePath, configJSON, 0644)
	Expect(err).ToNot(HaveOccurred())
	return configFilePath
}

