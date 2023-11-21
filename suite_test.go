package main_test

import (
	"testing"

	"github.com/alphagov/paas-rds-metric-collector/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	rdsMetricCollectorPath  string
	mockLocketServerSession *gexec.Session
	mockLocketServer        testhelpers.MockLocketServer
)

func TestPaasPostgresMetricCollector(t *testing.T) {
	BeforeSuite(func() {
		var err error
		rdsMetricCollectorPath, err = gexec.Build("github.com/alphagov/paas-rds-metric-collector")
		Expect(err).ToNot(HaveOccurred())
		mockLocketServer = testhelpers.MockLocketServer{}
		mockLocketServer.Build()
		mockLocketServerSession = mockLocketServer.Run("./fixtures", "alwaysGrantLock")
		Eventually(mockLocketServerSession.Buffer, "5s").Should(gbytes.Say("grpc.grpc-server.started"))
	})

	AfterSuite(func() {
		mockLocketServerSession.Kill()
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "RDS Metric Collector Suite")
}
