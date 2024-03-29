package fakebrokerinfo

import (
	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/stretchr/testify/mock"
)

type FakeBrokerInfo struct {
	mock.Mock
}

func (b *FakeBrokerInfo) GetInstanceConnectionDetails(
	instanceInfo brokerinfo.InstanceInfo,
) (brokerinfo.InstanceConnectionDetails, error) {
	args := b.Called(instanceInfo)
	return args.Get(0).(brokerinfo.InstanceConnectionDetails), args.Error(1)
}

func (b *FakeBrokerInfo) ListInstances() ([]brokerinfo.InstanceInfo, error) {
	args := b.Called()
	return args.Get(0).([]brokerinfo.InstanceInfo), args.Error(1)
}

func (b *FakeBrokerInfo) GetInstanceName(instanceInfo brokerinfo.InstanceInfo) string {
	args := b.Called(instanceInfo)
	return args.String(0)
}
