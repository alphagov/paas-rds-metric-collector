package main_test

import (
	"fmt"
	"os/exec"
	"time"

	_ "github.com/lib/pq"

	"github.com/alphagov/paas-rds-metric-collector/testhelpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("collector", func() {
	var (
		rdsMetricsCollectorSession *gexec.Session
	)

	It("fails to start if the config is missing", func() {
		var err error
		command := exec.Command(rdsMetricCollectorPath,
			fmt.Sprintf("-config=unknown.json"),
		)
		rdsMetricsCollectorSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(rdsMetricsCollectorSession).Should(gexec.Exit(1))
		Expect(rdsMetricsCollectorSession.Err).To(gbytes.Say("Error loading config file"))
		Expect(rdsMetricsCollectorSession.Err).To(gbytes.Say("no such file or directory"))
	})

	It("fails to start if the config is invalid", func() {
		var err error
		command := exec.Command(rdsMetricCollectorPath,
			fmt.Sprintf("-config=./fixtures/invalid_collector_config.json"),
		)
		rdsMetricsCollectorSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(rdsMetricsCollectorSession).Should(gexec.Exit(1))
		Expect(rdsMetricsCollectorSession.Err).To(gbytes.Say("Error loading config file"))
		Expect(rdsMetricsCollectorSession.Err).To(gbytes.Say("invalid character"))
	})

	Context("with valid configuration", func() {
		BeforeEach(func() {
			var err error
			command := exec.Command(rdsMetricCollectorPath,
				"-config="+testhelpers.BuildTempConfigFile(mockLocketServer.ListenAddress, "./fixtures"),
			)
			rdsMetricsCollectorSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			if rdsMetricsCollectorSession != nil {
				rdsMetricsCollectorSession.Kill()
			}
		})

		It("starts the collector process and keeps running for a while", func() {
			Eventually(rdsMetricsCollectorSession, 10*time.Second).Should(
				gbytes.Say("rds-metric-collector.scheduler.scheduler-started"),
			)
			Consistently(rdsMetricsCollectorSession, 2*time.Second).ShouldNot(gexec.Exit(0))
		})
		It("terminates (Ctrl+C) the process", func() {
			Eventually(rdsMetricsCollectorSession, 10*time.Second).Should(
				gbytes.Say("rds-metric-collector.scheduler.scheduler-started"),
			)
			rdsMetricsCollectorSession.Terminate()
			Eventually(rdsMetricsCollectorSession, 30*time.Second).Should(gexec.Exit())
		})
	})

})
