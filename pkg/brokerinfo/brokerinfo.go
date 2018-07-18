package brokerinfo

type InstanceInfo struct {
	GUID string
}

// BrokerInfo ...
type BrokerInfo interface {
	ListInstances() ([]InstanceInfo, error)
	ConnectionString(instanceInfo InstanceInfo) (string, error)
	GetInstanceName(instanceInfo InstanceInfo) string
}
