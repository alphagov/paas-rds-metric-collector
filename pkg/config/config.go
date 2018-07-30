package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/locket"
	validator "gopkg.in/go-playground/validator.v9"
)

type Config struct {
	LogLevel                     string                   `json:"log_level" validate:"required"`
	AWS                          AWSConfig                `json:"aws"`
	RDSBrokerInfo                RDSBrokerInfoConfig      `json:"rds_broker"`
	Scheduler                    SchedulerConfig          `json:"scheduler"`
	LoggregatorEmitter           LoggregatorEmitterConfig `json:"loggregator_emitter"`
	locket.ClientLocketConfig
}

type AWSConfig struct {
	Region       string `json:"region" validate:"required"`
	AWSPartition string `json:"aws_partition" validate:"required"`
}

type RDSBrokerInfoConfig struct {
	DBPrefix           string `json:"db_prefix" validate:"required"`
	BrokerName         string `json:"broker_name" validate:"required"`
	MasterPasswordSeed string `json:"master_password_seed" validate:"required"`
}

type SchedulerConfig struct {
	InstanceRefreshInterval int `json:"instance_refresh_interval" validate:"required,gte=1,lte=3600"`
	MetricCollectorInterval int `json:"metrics_collector_interval" validate:"required,gte=1,lte=3600"`
}

type LoggregatorEmitterConfig struct {
	MetronURL  string `json:"url" validate:"required"`
	CACertPath string `json:"ca_cert" validate:"required"`
	CertPath   string `json:"client_cert" validate:"required"`
	KeyPath    string `json:"client_key" validate:"required"`
}

const defaultConfig = `
{
	"log_level": "INFO",
	"aws": {
		"aws_partition": "aws"
	},
	"scheduler": {
		"instance_refresh_interval": 120,
		"metrics_collector_interval": 60
	},
	"loggregator_emitter": {
		"url": "localhost:3458"
	}
}
`

func LoadConfig(configFile string) (*Config, error) {
	var config Config

	if configFile == "" {
		return &config, errors.New("Must provide a config file")
	}

	file, err := os.Open(configFile)
	if err != nil {
		return &config, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return &config, err
	}

	json.Unmarshal([]byte(defaultConfig), &config) // Parse defaults
	if err = json.Unmarshal(bytes, &config); err != nil {
		return &config, err
	}

	if err = config.Validate(); err != nil {
		return &config, fmt.Errorf("Validating config contents: %s", err)
	}

	return &config, nil
}

func (c Config) Validate() error {
	validate := validator.New()

	return validate.Struct(c)
}
