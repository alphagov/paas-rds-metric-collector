package testhelpers

import (
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"code.cloudfoundry.org/locket"
	"io/ioutil"
	"encoding/json"
	. "github.com/onsi/gomega"
	"path"
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
			CACertPath: path.Join(fixturesPath, "ca.cert.pem"),
			CertPath:   path.Join(fixturesPath, "client.cert.pem"),
			KeyPath:    path.Join(fixturesPath, "client.key.pem"),
		},
		ClientLocketConfig: locket.ClientLocketConfig{
			LocketCACertFile:     path.Join(fixturesPath, "ca.cert.pem"),
			LocketClientCertFile: path.Join(fixturesPath, "client.cert.pem"),
			LocketClientKeyFile:  path.Join(fixturesPath, "client.key.pem"),
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

