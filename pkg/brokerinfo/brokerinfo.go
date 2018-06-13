package brokerinfo

// BrokerInfo ...
type BrokerInfo interface {
	ListInstanceGUIDs() ([]string, error)
	ConnectionString(instanceGUID string) (string, error)
}
