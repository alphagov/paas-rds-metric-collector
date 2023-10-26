package collector

import (
	"fmt"
	"os"
	"testing"

	"code.cloudfoundry.org/lager/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var postgresTestDatabaseConnectionURL string
var mysqlTestDatabaseConnectionURL string
var logger lager.Logger

var _ = BeforeSuite(func() {
	logger = lager.NewLogger("tests")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))

	postgresTestDatabaseConnectionURL = os.Getenv("TEST_POSTGRES_URL")
	if postgresTestDatabaseConnectionURL == "" {
		postgresTestDatabaseConnectionURL = "postgresql://postgres@localhost:5432?sslmode=disable"
		fmt.Fprintf(GinkgoWriter, "$TEST_POSTGRES_URL not defined, using default: %s\n", postgresTestDatabaseConnectionURL)
	}

	mysqlTestDatabaseConnectionURL = os.Getenv("TEST_MYSQL_URL")
	if mysqlTestDatabaseConnectionURL == "" {
		mysqlTestDatabaseConnectionURL = "root:@tcp(localhost:3306)/mysql?tls=false"
		fmt.Fprintf(GinkgoWriter, "$TEST_MYSQL_URL not defined, using default: %s\n", mysqlTestDatabaseConnectionURL)
	}
})

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Collector Suite")
}
