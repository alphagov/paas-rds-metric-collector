package scheduler

import (
	"testing"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var logger lager.Logger

var _ = BeforeSuite(func() {
	logger = lager.NewLogger("tests")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
})

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scheduler Suite")
}
