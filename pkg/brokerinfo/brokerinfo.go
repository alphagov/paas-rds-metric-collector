package brokerinfo

type InstanceInfo struct {
	GUID string
	Type string
}

type InstanceConnectionDetails struct {
	DBAddress      string
	DBPort         int64
	DBName         string
	MasterUsername string
	MasterPassword string
}

// BrokerInfo ...
type BrokerInfo interface {
	ListInstances() ([]InstanceInfo, error)
	GetInstanceConnectionDetails(instanceInfo InstanceInfo) (InstanceConnectionDetails, error)
	GetInstanceName(instanceInfo InstanceInfo) string
}
