package locking_test

import (
	"github.com/alphagov/paas-rds-metric-collector/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"os/exec"
	"time"
)

var _ = Describe("Locket / rds-metric-collector process group", func() {
	var (
		mockLocketServerSession *gexec.Session
		err                     error
	)

	AfterEach(func() {
		mockLocketServerSession.Kill()
	})

	It("Should start cleanly if it can acquire a lock", func() {
		mockLocketServerSession = mockLocketServer.Run("../fixtures", "alwaysGrantLock")
		Expect(err).ToNot(HaveOccurred())
		configFilePath := testhelpers.BuildTempConfigFile(mockLocketServer.ListenAddress, "../fixtures")

		Eventually(mockLocketServerSession.Buffer).Should(gbytes.Say("grpc.grpc-server.started"))

		rdsMetricCollectorSession, err := gexec.Start(exec.Command(rdsMetricCollectorPath, "-config="+configFilePath), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(rdsMetricCollectorSession.Buffer).Should(gbytes.Say("rds-metric-collector.locket-lock.acquired-lock"))
		Eventually(rdsMetricCollectorSession.Buffer).Should(gbytes.Say("rds-metric-collector.scheduler.scheduler-started"))
	})

	It("Should hang if it cannot acquire a lock", func() {
		mockLocketServerSession = mockLocketServer.Run("../fixtures", "neverGrantLock")
		Expect(err).ToNot(HaveOccurred())
		configFilePath := testhelpers.BuildTempConfigFile(mockLocketServer.ListenAddress, "../fixtures")

		Eventually(mockLocketServerSession.Buffer).Should(gbytes.Say("grpc.grpc-server.started"))

		rdsMetricCollectorSession, err := gexec.Start(exec.Command(rdsMetricCollectorPath, "-config="+configFilePath), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(rdsMetricCollectorSession.Buffer).Should(gbytes.Say("rds-metric-collector.locket-lock.failed-to-acquire-lock"))
		Consistently(rdsMetricCollectorSession.Buffer, 5*time.Second).ShouldNot(gbytes.Say("tps-watcher.started"))
	})

	It("Should hang until it acquires a lock, then start", func() {
		mockLocketServerSession = mockLocketServer.Run("../fixtures", "grantLockAfterFiveAttempts")
		Expect(err).ToNot(HaveOccurred())
		configFilePath := testhelpers.BuildTempConfigFile(mockLocketServer.ListenAddress, "../fixtures")

		Eventually(mockLocketServerSession.Buffer).Should(gbytes.Say("grpc.grpc-server.started"))

		rdsMetricCollectorSession, err := gexec.Start(exec.Command(rdsMetricCollectorPath, "-config="+configFilePath), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(rdsMetricCollectorSession.Buffer).Should(gbytes.Say("rds-metric-collector.locket-lock.failed-to-acquire-lock"))
		// By default locketClient retries every one second with a TTL of 15 seconds.
		// The mock server is set to release the lock after 5 attempts, so we need to wait more than 5 seconds
		Eventually(rdsMetricCollectorSession.Buffer, 10*time.Second).Should(gbytes.Say("rds-metric-collector.locket-lock.acquired-lock"))
		Eventually(rdsMetricCollectorSession.Buffer).Should(gbytes.Say("rds-metric-collector.scheduler.scheduler-started"))
	})

	It("Should crash if it loses the lock", func() {
		mockLocketServerSession = mockLocketServer.Run("../fixtures", "grantLockOnceThenFail")
		Expect(err).ToNot(HaveOccurred())
		configFilePath := testhelpers.BuildTempConfigFile(mockLocketServer.ListenAddress, "../fixtures")

		Eventually(mockLocketServerSession.Buffer).Should(gbytes.Say("grpc.grpc-server.started"))

		rdsMetricCollectorSession, err := gexec.Start(exec.Command(rdsMetricCollectorPath, "-config="+configFilePath), GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(rdsMetricCollectorSession.Buffer).Should(gbytes.Say("rds-metric-collector.locket-lock.acquired-lock"))
		Eventually(rdsMetricCollectorSession.Buffer, 10*time.Second).Should(gbytes.Say("rds-metric-collector.locket-lock.lost-lock"))
		Eventually(rdsMetricCollectorSession.Buffer, 30*time.Second).Should(gbytes.Say("rds-metric-collector.process-group-stopped-with-error"))
		Eventually(rdsMetricCollectorSession, 30*time.Second).Should(gexec.Exit())
	})
})
