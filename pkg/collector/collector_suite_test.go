package collector

import (
	"fmt"
	"os"
	"testing"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var postgresTestDatabaseConnectionUrl string
var logger lager.Logger

var _ = BeforeSuite(func() {
	logger = lager.NewLogger("tests")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

	postgresTestDatabaseConnectionUrl = os.Getenv("TEST_DATABASE_URL")
	if postgresTestDatabaseConnectionUrl == "" {
		postgresTestDatabaseConnectionUrl = "postgresql://postgres@localhost:5432?sslmode=disable"
		fmt.Fprintf(GinkgoWriter, "$TEST_DATABASE_URL not defined, using default: %s\n", postgresTestDatabaseConnectionUrl)
	}
})

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}
