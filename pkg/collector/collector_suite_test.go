package collector

import (
	"fmt"
	"os"
	"testing"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var postgresTestDatabaseConnectionURL string
var logger lager.Logger

var _ = BeforeSuite(func() {
	logger = lager.NewLogger("tests")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

	postgresTestDatabaseConnectionURL = os.Getenv("TEST_DATABASE_URL")
	if postgresTestDatabaseConnectionURL == "" {
		postgresTestDatabaseConnectionURL = "postgresql://postgres@localhost:5432?sslmode=disable"
		fmt.Fprintf(GinkgoWriter, "$TEST_DATABASE_URL not defined, using default: %s\n", postgresTestDatabaseConnectionURL)
	}
})

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}
