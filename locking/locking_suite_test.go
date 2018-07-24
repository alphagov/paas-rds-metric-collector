package locking_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/alphagov/paas-rds-metric-collector/testhelpers"
)

var (
	rdsMetricCollectorPath string
	mockLocketServer       testhelpers.MockLocketServer
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
