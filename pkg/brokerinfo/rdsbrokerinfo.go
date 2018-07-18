package brokerinfo

import (
	"fmt"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-rds-broker/awsrds"
	"github.com/alphagov/paas-rds-broker/utils"

	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
)

type RDSBrokerInfo struct {
	brokerName         string
	dbPrefix           string
	masterPasswordSeed string
	dbInstance         awsrds.DBInstance
	logger             lager.Logger
}

func NewRDSBrokerInfo(
	brokerInfoConfig config.RDSBrokerInfoConfig,
	dbInstance awsrds.DBInstance,
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
		if dbDetails.Engine == "postgres" {
			instanceInfo := InstanceInfo{
				GUID: r.dbInstanceIdentifierToServiceInstanceID(dbDetails.Identifier),
				Type: "postgres",
			}
			serviceInstances = append(serviceInstances, instanceInfo)
		}
	}
	return serviceInstances, nil
}

func (r *RDSBrokerInfo) ConnectionString(instanceInfo InstanceInfo) (string, error) {
	if instanceInfo.Type != "postgres" {
		return "", fmt.Errorf("invalid instance type: %s", instanceInfo.Type)
	}
	dbInstanceDetails, err := r.dbInstance.Describe(r.dbInstanceIdentifier(instanceInfo.GUID))
	if err != nil {
		r.logger.Error("obtaining instances details", err, lager.Data{"brokerName": r.brokerName, "instanceInfo": instanceInfo})
		return "", err
	}

	dbAddress := dbInstanceDetails.Address
	dbPort := dbInstanceDetails.Port
	masterUsername := dbInstanceDetails.MasterUsername
	masterPassword := r.generateMasterPassword(instanceInfo.GUID)
	dbName := dbInstanceDetails.DBName

	ConnectionString := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=require", masterUsername, masterPassword, dbAddress, dbPort, dbName)

	return ConnectionString, nil

}

func (r *RDSBrokerInfo) GetInstanceName(instanceInfo InstanceInfo) string {
	return r.dbInstanceIdentifier(instanceInfo.GUID)
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
