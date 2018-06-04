package scheduler

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo/fakebrokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeMetricsCollectorDriver struct {
	mock.Mock
}

func (f *fakeMetricsCollectorDriver) NewCollector(instanceGUID string) (collector.MetricsCollector, error) {
	args := f.Called(instanceGUID)
	arg0 := args.Get(0)
	if arg0 != nil {
		return arg0.(collector.MetricsCollector), args.Error(1)
	} else {
		return nil, args.Error(1)
	}
}

type fakeMetricsCollector struct {
	mock.Mock
}

func (f *fakeMetricsCollector) Collect() ([]metrics.Metric, error) {
	args := f.Called()
	return args.Get(0).([]metrics.Metric), args.Error(1)
}

type fakeMetricsEmitter struct {
	envelopesReceived []metrics.MetricEnvelope
}

func (f *fakeMetricsEmitter) Emit(me metrics.MetricEnvelope) {
	f.envelopesReceived = append(f.envelopesReceived, me)
}

var _ = Describe("collector scheduler", func() {
	var (
		brokerInfo             *fakebrokerinfo.FakeBrokerInfo
		metricsEmitter         *fakeMetricsEmitter
		metricsCollectorDriver *fakeMetricsCollectorDriver
		metricsCollector       *fakeMetricsCollector
		scheduler              *Scheduler
	)

	BeforeEach(func() {
		var err error
		brokerInfo = &fakebrokerinfo.FakeBrokerInfo{}
		metricsEmitter = &fakeMetricsEmitter{}
		metricsCollectorDriver = &fakeMetricsCollectorDriver{}
		metricsCollector = &fakeMetricsCollector{}

		scheduler = NewScheduler(
			brokerInfo,
			metricsEmitter,
			metricsCollectorDriver,
			logger,
		)

		scheduler.instanceRefreshInterval = 1
		scheduler.metricCollectorInterval = 1
	})

	It("should not start any worker and return error if fails starting the scheduler", func() {
		scheduler.instanceRefreshInterval = 0 // Force the `scheduler` library to fail

		err := scheduler.Start()
		Expect(err).To(HaveOccurred())

		Consistently(func() []string {
			return scheduler.ListWorkers()
		}, 1*time.Second).Should(
			HaveLen(0),
		)
	})

	It("should not schedule any worker if brokerinfo.ListInstanceGUIDs() fails", func() {
		brokerInfo.On(
			"ListInstanceGUIDs", mock.Anything,
		).Return(
			[]string{}, fmt.Errorf("Error in ListInstanceGUIDs"),
		)

		scheduler.Start()

		Consistently(func() []string {
			return scheduler.ListWorkers()
		}, 1*time.Second).Should(
			HaveLen(0),
		)
		metricsCollectorDriver.AssertNotCalled(GinkgoT(), "NewCollector")
	})

	It("should check for new instances every 1 second", func() {
		brokerInfo.On(
			"ListInstanceGUIDs", mock.Anything,
		).Return(
			[]string{}, nil,
		)

		scheduler.Start()

		Eventually(
			func() int { return len(brokerInfo.Calls) },
			2*time.Second,
		).Should(BeNumerically(">=", 2))
	})

	It("should not add a worker if fails creating a collector ", func() {
		brokerInfo.On(
			"ListInstanceGUIDs", mock.Anything,
		).Return(
			[]string{"instance-guid1"}, nil,
		)
		metricsCollectorDriver.On(
			"NewCollector", mock.Anything,
		).Return(
			nil, fmt.Errorf("Failed creating collector"),
		)

		scheduler.Start()

		Consistently(func() []string {
			return scheduler.ListWorkers()
		}, 1*time.Second).Should(
			HaveLen(0),
		)

	})

	It("should not send metrics if the collector returns an error", func() {
		brokerInfo.On(
			"ListInstanceGUIDs", mock.Anything,
		).Return(
			[]string{"instance-guid1"}, nil,
		)
		metricsCollectorDriver.On(
			"NewCollector", mock.Anything,
		).Return(
			metricsCollector, nil,
		)
		metricsCollector.On(
			"Collect",
		).Return(
			[]metrics.Metric{
				metrics.Metric{Key: "foo", Value: 1, Unit: "b"},
			},
			fmt.Errorf("error collecting metrics"),
		)

		scheduler.Start()

		Consistently(func() []metrics.MetricEnvelope {
			return metricsEmitter.envelopesReceived
		}, 2*time.Second).Should(
			HaveLen(0),
		)

	})

	Context("with working collector", func() {
		BeforeEach(func() {
			metricsCollectorDriver.On(
				"NewCollector", mock.Anything,
			).Return(
				metricsCollector, nil,
			)
			metricsCollector.On(
				"Collect",
			).Return(
				[]metrics.Metric{
					metrics.Metric{Key: "foo", Value: 1, Unit: "b"},
				},
				nil,
			)
			metricsCollector.On(
				"Close", mock.Anything,
			).Return(
				nil,
			)
		})

		It("should not add a worker if it fails scheduling the worker job", func() {
			scheduler.metricCollectorInterval = 0 // Force the `scheduler` library to fail
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1"}, nil,
			)

			scheduler.Start()

			Eventually(func() []string {
				return scheduler.ListWorkers()
			}, 1*time.Second).Should(
				HaveLen(0),
			)
		})

		It("should start one worker successfully when one instance exist", func() {
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1"}, nil,
			)

			scheduler.Start()

			Eventually(func() []string {
				return scheduler.ListWorkers()
			}, 1*time.Second).Should(
				HaveLen(1),
			)
			Eventually(func() []metrics.MetricEnvelope {
				return metricsEmitter.envelopesReceived
			}, 2*time.Second).Should(
				ContainElement(
					metrics.MetricEnvelope{
						InstanceGUID: "instance-guid1",
						Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
					},
				),
			)
		})

		It("should start multiple workers successfully when multiple instance exist", func() {
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1", "instance-guid2"}, nil,
			)

			scheduler.Start()

			Eventually(func() []string {
				return scheduler.ListWorkers()
			}, 1*time.Second).Should(
				HaveLen(2),
			)
			Eventually(func() []metrics.MetricEnvelope {
				return metricsEmitter.envelopesReceived
			}, 2*time.Second).Should(
				ContainElement(
					metrics.MetricEnvelope{
						InstanceGUID: "instance-guid1",
						Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
					},
				),
			)

			Eventually(func() []metrics.MetricEnvelope {
				return metricsEmitter.envelopesReceived
			}, 2*time.Second).Should(
				ContainElement(
					metrics.MetricEnvelope{
						InstanceGUID: "instance-guid2",
						Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
					},
				),
			)
		})

		It("should add new workers when a new instance appears", func() {
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1"}, nil,
			).Once()

			scheduler.Start()

			Eventually(func() []string {
				return scheduler.ListWorkers()
			}, 1*time.Second).Should(
				HaveLen(1),
			)

			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1", "instance-guid2"}, nil,
			)

			// Clear received envelopes
			metricsEmitter.envelopesReceived = metricsEmitter.envelopesReceived[:0]

			Eventually(func() []string {
				return scheduler.ListWorkers()
			}, 1*time.Second).Should(
				HaveLen(2),
			)

			Eventually(func() []metrics.MetricEnvelope {
				return metricsEmitter.envelopesReceived
			}, 2*time.Second).Should(
				ContainElement(
					metrics.MetricEnvelope{
						InstanceGUID: "instance-guid1",
						Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
					},
				),
			)

			Eventually(func() []metrics.MetricEnvelope {
				return metricsEmitter.envelopesReceived
			}, 2*time.Second).Should(
				ContainElement(
					metrics.MetricEnvelope{
						InstanceGUID: "instance-guid2",
						Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
					},
				),
			)
		})

		It("should stop workers when one instance disappears", func() {
			// First loop returns 2 instances
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1", "instance-guid2"}, nil,
			).Once()

			// After return only one instance
			brokerInfo.On(
				"ListInstanceGUIDs", mock.Anything,
			).Return(
				[]string{"instance-guid1"}, nil,
			)

			scheduler.Start()

			Eventually(func() []string {
				return scheduler.ListWorkers()
			}, 2*time.Second).Should(
				HaveLen(2),
			)

			Eventually(func() []string {
				return scheduler.ListWorkers()
			}, 2*time.Second).Should(
				HaveLen(1),
			)

			// Clear received envelopes
			metricsEmitter.envelopesReceived = metricsEmitter.envelopesReceived[:0]

			Consistently(func() []metrics.MetricEnvelope {
				return metricsEmitter.envelopesReceived
			}, 2*time.Second).ShouldNot(
				ContainElement(
					metrics.MetricEnvelope{
						InstanceGUID: "instance-guid2",
						Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
					},
				),
			)
		})

	})

})
