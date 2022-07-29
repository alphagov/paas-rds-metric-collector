package emitter_test

import (
	"testing"

	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var logger lager.Logger

var _ = BeforeSuite(func() {
	logger = lager.NewLogger("tests")
	logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))
})

func TestCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Emitter Suite")
}
