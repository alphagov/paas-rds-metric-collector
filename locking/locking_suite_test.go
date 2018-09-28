package locking_test

import (
	"testing"

	"github.com/alphagov/paas-rds-metric-collector/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	rdsMetricCollectorPath string

	// MockLocketServer is now available as in in-process server in an external library:
	// https://github.com/alphagov/paas-go
	// Consider refactoring next time this is updated.
	mockLocketServer testhelpers.MockLocketServer
)

func TestLocking(t *testing.T) {
	BeforeSuite(func() {
		var err error

		rdsMetricCollectorPath, err = gexec.Build("github.com/alphagov/paas-rds-metric-collector")
		Expect(err).ToNot(HaveOccurred())

		mockLocketServer = testhelpers.MockLocketServer{}
		mockLocketServer.Build()
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "Locking Suite")
}
