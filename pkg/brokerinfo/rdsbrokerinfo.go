package brokerinfo

// BrokerInfo ...
type RDSBrokerInfo struct{}

func (r *RDSBrokerInfo) ListInstanceGUIDs() ([]string, error) {
	panic("Not Implemented")
}
func (r *RDSBrokerInfo) ConnectionString(instanceGUID string) (string, error) {
	panic("Not Implemented")
}
