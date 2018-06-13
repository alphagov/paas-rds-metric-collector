package fakebrokerinfo

import "github.com/stretchr/testify/mock"

type FakeBrokerInfo struct {
	mock.Mock
}

func (b *FakeBrokerInfo) ConnectionString(instanceGUID string) (string, error) {
	args := b.Called(instanceGUID)
	return args.String(0), args.Error(1)
}

func (b *FakeBrokerInfo) ListInstanceGUIDs() ([]string, error) {
	args := b.Called()
	return args.Get(0).([]string), args.Error(1)
}
