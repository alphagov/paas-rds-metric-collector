package testhelpers

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/phayes/freeport"
)

type MockLocketServer struct {
	compiledPath  string
	ListenAddress string
}

func (server *MockLocketServer) Build() {
	compiledPath, err := gexec.Build("github.com/alphagov/paas-rds-metric-collector/testhelpers/mock_locket_server")
	Expect(err).NotTo(HaveOccurred())
	server.compiledPath = compiledPath
}

func (server *MockLocketServer) Run(fixturesPath, lockingMode string) *gexec.Session {
	port, err := freeport.GetFreePort()
	Expect(err).NotTo(HaveOccurred())
	server.ListenAddress = fmt.Sprintf("127.0.0.1:%d", port)
	command := exec.Command(server.compiledPath,
		"-fixturesPath="+fixturesPath,
		"-mode="+lockingMode,
		"-listenAddress="+server.ListenAddress)

	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}
