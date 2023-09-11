package testhelpers

import (
	"encoding/json"
	"os"
	"path"

	"code.cloudfoundry.org/locket"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
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
			InstanceRefreshInterval:    30,
			SQLMetricCollectorInterval: 5,
			CWMetricCollectorInterval:  5,
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
	temporaryConfigFile, err := os.CreateTemp("", "rds-metrics-collector-config-")
	Expect(err).ToNot(HaveOccurred())
	configJSON, err := json.Marshal(rdsMetricCollectorConfig)
	Expect(err).ToNot(HaveOccurred())
	configFilePath = temporaryConfigFile.Name()
	err = os.WriteFile(configFilePath, configJSON, 0644)
	Expect(err).ToNot(HaveOccurred())
	return configFilePath
}
