package brokerinfo

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/lager/v3"
	"github.com/alphagov/paas-rds-broker/awsrds"
	"github.com/alphagov/paas-rds-broker/utils"

	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/aws/aws-sdk-go/service/rds"
)

type RDSBrokerInfo struct {
	brokerName         string
	dbPrefix           string
	masterPasswordSeed string
	dbInstance         awsrds.RDSInstance
	logger             lager.Logger
}

func NewRDSBrokerInfo(
	brokerInfoConfig config.RDSBrokerInfoConfig,
	dbInstance awsrds.RDSInstance,
	logger lager.Logger,
) *RDSBrokerInfo {
	return &RDSBrokerInfo{
		brokerName:         brokerInfoConfig.BrokerName,
		dbPrefix:           brokerInfoConfig.DBPrefix,
		masterPasswordSeed: brokerInfoConfig.MasterPasswordSeed,
		dbInstance:         dbInstance,
		logger:             logger,
	}
}

func (r *RDSBrokerInfo) ListInstances() ([]InstanceInfo, error) {
	serviceInstances := []InstanceInfo{}

	dbInstanceDetailsList, err := r.dbInstance.DescribeByTag("Broker Name", r.brokerName)
	if err != nil {
		r.logger.Error("retriving list of AWS instances", err, lager.Data{"brokerName": r.brokerName})
		return serviceInstances, err
	}

	for _, dbDetails := range dbInstanceDetailsList {
		engine := stringValue(dbDetails.Engine)
		if engine != "postgres" && engine != "mysql" {
			continue
		}
		instanceInfo := InstanceInfo{
			GUID: r.dbInstanceIdentifierToServiceInstanceID(stringValue(dbDetails.DBInstanceIdentifier)),
			Type: engine,
		}
		serviceInstances = append(serviceInstances, instanceInfo)
	}
	return serviceInstances, nil
}

func (r *RDSBrokerInfo) GetInstanceConnectionDetails(instanceInfo InstanceInfo) (InstanceConnectionDetails, error) {
	if instanceInfo.Type != "postgres" && instanceInfo.Type != "mysql" {
		return InstanceConnectionDetails{}, fmt.Errorf("invalid instance type: %s", instanceInfo.Type)
	}
	dbInstanceDetails, err := r.dbInstance.Describe(r.dbInstanceIdentifier(instanceInfo.GUID))
	if err != nil {
		r.logger.Error("obtaining instances details", err, lager.Data{"brokerName": r.brokerName, "instanceInfo": instanceInfo})
		return InstanceConnectionDetails{}, err
	}

	return InstanceConnectionDetails{
		DBAddress:      getEndpointAddress(dbInstanceDetails.Endpoint),
		DBPort:         getEndpointPort(dbInstanceDetails.Endpoint),
		MasterUsername: stringValue(dbInstanceDetails.MasterUsername),
		MasterPassword: r.generateMasterPassword(instanceInfo.GUID),
		DBName:         stringValue(dbInstanceDetails.DBName),
	}, nil
}

func (r *RDSBrokerInfo) GetInstanceName(instanceInfo InstanceInfo) string {
	return r.dbInstanceIdentifier(instanceInfo.GUID)
}

func stringValue(pointer *string) string {
	if pointer == nil {
		return ""
	} else {
		return *pointer
	}
}

func int64Value(pointer *int64) int64 {
	if pointer == nil {
		return 0
	} else {
		return *pointer
	}
}

func getEndpointPort(endpoint *rds.Endpoint) int64 {
	if endpoint != nil {
		return int64Value(endpoint.Port)
	}
	return 0
}

func getEndpointAddress(endpoint *rds.Endpoint) string {
	if endpoint != nil {
		return stringValue(endpoint.Address)
	}
	return ""
}

// FIXME: Following code has been copied from
// https://github.com/alphagov/paas-rds-broker/blob/eee2df8257264e9afdbe9bc1b942174882e5d0d5/rdsbroker/broker.go#L666-L669
// We shall refactor paas-rds-broker to extract this to a module that can be imported
const MasterPasswordLength = 32

func (r *RDSBrokerInfo) dbInstanceIdentifier(instanceGUID string) string {
	return fmt.Sprintf("%s-%s", strings.Replace(r.dbPrefix, "_", "-", -1), strings.Replace(instanceGUID, "_", "-", -1))
}

func (r *RDSBrokerInfo) dbInstanceIdentifierToServiceInstanceID(serviceInstanceID string) string {
	return strings.TrimPrefix(serviceInstanceID, strings.Replace(r.dbPrefix, "_", "-", -1)+"-")
}

func (r *RDSBrokerInfo) generateMasterPassword(instanceGUID string) string {
	return utils.GenerateHash(r.masterPasswordSeed+instanceGUID, MasterPasswordLength)
}
