package scheduler

import (
	"fmt"
	"time"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo/fakebrokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector"
	"github.com/alphagov/paas-rds-metric-collector/pkg/config"
	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/mock"

	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeMetricsCollectorDriver struct {
	mock.Mock
}

func (f *fakeMetricsCollectorDriver) NewCollector(instanceInfo brokerinfo.InstanceInfo) (collector.MetricsCollector, error) {
	args := f.Called(instanceInfo)
	arg0 := args.Get(0)
	if arg0 != nil {
		return arg0.(collector.MetricsCollector), args.Error(1)
	}

	return nil, args.Error(1)
}

func (f *fakeMetricsCollectorDriver) GetName() string {
	args := f.Called()
	return args.String(0)
}

func (f *fakeMetricsCollectorDriver) SupportedTypes() []string {
	args := f.Called()
	return args.Get(0).([]string)
}

func (f *fakeMetricsCollectorDriver) GetCollectInterval() int {
	args := f.Called()
	return args.Int(0)
}

type fakeMetricsCollector struct {
	mock.Mock
}

func (f *fakeMetricsCollector) Collect() ([]metrics.Metric, error) {
	args := f.Called()
	return args.Get(0).([]metrics.Metric), args.Error(1)
}

func (f *fakeMetricsCollector) Close() error {
	args := f.Called()
	return args.Error(0)
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
		signals                chan os.Signal
		ready                  chan struct{}
	)

	BeforeEach(func() {
		brokerInfo = &fakebrokerinfo.FakeBrokerInfo{}
		metricsEmitter = &fakeMetricsEmitter{}
		metricsCollectorDriver = &fakeMetricsCollectorDriver{}
		metricsCollectorDriver.On("GetName").Return("fake")
		metricsCollectorDriver.On("GetCollectInterval").Return(1)
		metricsCollectorDriver.On("SupportedTypes").Return([]string{"fake"})
		metricsCollector = &fakeMetricsCollector{}

		signals = make(chan os.Signal)
		ready = make(chan struct{})

		scheduler = NewScheduler(
			config.SchedulerConfig{
				InstanceRefreshInterval: 1,
			},
			brokerInfo,
			metricsEmitter,
			logger,
		)
		scheduler.WithDriver(metricsCollectorDriver)
	})

	It("should not start any worker and return error if fails starting the scheduler", func() {
		scheduler.instanceRefreshInterval = 0 // Force the `scheduler` library to fail

		err := scheduler.Run(signals, ready)
		Expect(err).To(HaveOccurred())

		Consistently(func() []string {
			return scheduler.ListIntanceGUIDs()
		}, 1*time.Second).Should(
			HaveLen(0),
		)
	})

	It("should not schedule any worker if brokerinfo.ListInstanceGUIDs() fails", func() {
		brokerInfo.On(
			"ListInstances", mock.Anything,
		).Return(
			[]brokerinfo.InstanceInfo{}, fmt.Errorf("Error in ListInstanceGUIDs"),
		)

		go scheduler.Run(signals, ready)

		Consistently(func() []string {
			return scheduler.ListIntanceGUIDs()
		}, 1*time.Second).Should(
			HaveLen(0),
		)
		metricsCollectorDriver.AssertNotCalled(GinkgoT(), "NewCollector")
	})

	It("should check for new instances every 1 second", func() {
		brokerInfo.On(
			"ListInstances", mock.Anything,
		).Return(
			[]brokerinfo.InstanceInfo{}, nil,
		)

		go scheduler.Run(signals, ready)

		Eventually(
			func() int { return len(brokerInfo.Calls) },
			2*time.Second,
		).Should(BeNumerically(">=", 2))
	})

	It("should not add a worker if fails creating a collector ", func() {
		brokerInfo.On(
			"ListInstances", mock.Anything,
		).Return(
			[]brokerinfo.InstanceInfo{
				{GUID: "instance-guid1", Type: "fake"},
			}, nil,
		)
		metricsCollectorDriver.On(
			"NewCollector", mock.Anything,
		).Return(
			nil, fmt.Errorf("Failed creating collector"),
		)

		go scheduler.Run(signals, ready)

		Consistently(func() []string {
			return scheduler.ListIntanceGUIDs()
		}, 1*time.Second).Should(
			HaveLen(0),
		)

	})

	It("should not send metrics if the collector returns an error", func() {
		brokerInfo.On(
			"ListInstances", mock.Anything,
		).Return(
			[]brokerinfo.InstanceInfo{
				{GUID: "instance-guid1", Type: "fake"},
			}, nil,
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

		go scheduler.Run(signals, ready)

		Consistently(func() []metrics.MetricEnvelope {
			return metricsEmitter.envelopesReceived
		}, 2*time.Second).Should(
			HaveLen(0),
		)

	})

	Context("with working collector driver", func() {

		var metricsCollectorDriverNewCollectorCall *mock.Call

		BeforeEach(func() {
			metricsCollectorDriverNewCollectorCall = metricsCollectorDriver.On(
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
			scheduler.metricsCollectorDrivers = map[string]collector.MetricsCollectorDriver{} // Force the `scheduler` library to fail
			brokerInfo.On(
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
				}, nil,
			)

			go scheduler.Run(signals, ready)

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
			}, 1*time.Second).Should(
				HaveLen(0),
			)
		})

		It("should start one worker successfully when one instance exist", func() {
			brokerInfo.On(
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
				}, nil,
			)

			go scheduler.Run(signals, ready)

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
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
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
					{GUID: "instance-guid2", Type: "fake"},
				}, nil,
			)

			go scheduler.Run(signals, ready)

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
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
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
				}, nil,
			).Once()

			go scheduler.Run(signals, ready)

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
			}, 1*time.Second).Should(
				HaveLen(1),
			)

			brokerInfo.On(
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
					{GUID: "instance-guid2", Type: "fake"},
				}, nil,
			)

			// Clear received envelopes
			metricsEmitter.envelopesReceived = metricsEmitter.envelopesReceived[:0]

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
			}, 2*time.Second).Should(
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
			metricsCollector.On(
				"Close", mock.Anything,
			).Return(
				nil,
			)
			// First loop returns 2 instances
			brokerInfo.On(
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
					{GUID: "instance-guid2", Type: "fake"},
				}, nil,
			).Once()

			// After return only one instance
			brokerInfo.On(
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
				}, nil,
			)

			go scheduler.Run(signals, ready)

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
			}, 2*time.Second).Should(
				HaveLen(2),
			)

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
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

		It("should stop the scheduler, workers and close collectors", func() {
			brokerInfo.On(
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
					{GUID: "instance-guid2", Type: "fake"},
				}, nil,
			)

			go scheduler.Run(signals, ready)

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
			}, 1*time.Second).Should(
				HaveLen(2),
			)

			scheduler.Stop()

			Eventually(func() []string {
				return scheduler.ListIntanceGUIDs()
			}, 1*time.Second).Should(
				HaveLen(0),
			)
			metricsCollector.AssertNumberOfCalls(GinkgoT(), "Close", 2)

			Consistently(func() bool {
				brokerInfo.AssertNumberOfCalls(GinkgoT(), "ListInstances", 1)
				metricsCollectorDriver.AssertNumberOfCalls(GinkgoT(), "NewCollector", 2)
				metricsCollector.AssertNumberOfCalls(GinkgoT(), "Collect", 2)
				return true
			}).Should(BeTrue())
		})

		It("should stop the scheduler without any race condition", func() {
			brokerInfo.On(
				"ListInstances", mock.Anything,
			).Return(
				[]brokerinfo.InstanceInfo{
					{GUID: "instance-guid1", Type: "fake"},
				}, nil,
			)

			metricsCollectorDriverNewCollectorCall.After(700 * time.Millisecond)

			go scheduler.Run(signals, ready)

			// Wait for the collector to collect metrics at least once
			Eventually(func() []metrics.MetricEnvelope {
				return metricsEmitter.envelopesReceived
			}, 2*time.Second).Should(
				HaveLen(1),
			)

			// Stop the scheduler
			scheduler.Stop()

			// Should not have any workers to the list
			Expect(scheduler.ListIntanceGUIDs()).To(HaveLen(0))
			// Should not send any other envelope
			Consistently(func() []metrics.MetricEnvelope {
				return metricsEmitter.envelopesReceived
			}, 2*time.Second).Should(
				HaveLen(1),
			)
		})

		Context("with two collector drivers", func() {
			var (
				metricsCollectorDriver2 *fakeMetricsCollectorDriver
				metricsCollector2       *fakeMetricsCollector
			)

			BeforeEach(func() {
				metricsCollectorDriver2 = &fakeMetricsCollectorDriver{}
				metricsCollectorDriver2.On("GetName").Return("fake2")
				metricsCollectorDriver2.On("GetCollectInterval").Return(1)
				metricsCollectorDriver2.On("SupportedTypes").Return([]string{"fake", "fake2"})
				metricsCollector2 = &fakeMetricsCollector{}

				scheduler.WithDriver(metricsCollectorDriver2)

			})

			It("should start one worker successfully when one driver works but other not", func() {
				brokerInfo.On(
					"ListInstances", mock.Anything,
				).Return(
					[]brokerinfo.InstanceInfo{
						{GUID: "instance-guid1", Type: "fake"},
					}, nil,
				)
				metricsCollectorDriver2.On(
					"NewCollector", mock.Anything,
				).Return(
					nil, fmt.Errorf("Failed creating collector"),
				)

				go scheduler.Run(signals, ready)

				Eventually(func() []string {
					return scheduler.ListIntanceGUIDs()
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

			It("Retries to create new collectors if one driver fails creating it once", func() {
				brokerInfo.On(
					"ListInstances", mock.Anything,
				).Return(
					[]brokerinfo.InstanceInfo{
						{GUID: "instance-guid1", Type: "fake"},
					}, nil,
				)

				metricsCollectorDriver2.On(
					"NewCollector", mock.Anything,
				).Return(
					nil, fmt.Errorf("Failed creating collector"),
				).Once()

				metricsCollectorDriver2.On(
					"NewCollector", mock.Anything,
				).Return(
					metricsCollector2, nil,
				)
				metricsCollector2.On(
					"Collect",
				).Return(
					[]metrics.Metric{
						metrics.Metric{Key: "bar", Value: 3, Unit: "s"},
					},
					nil,
				)
				metricsCollector2.On(
					"Close", mock.Anything,
				).Return(
					nil,
				)

				go scheduler.Run(signals, ready)

				Eventually(func() []string {
					return scheduler.ListIntanceGUIDs()
				}, 1*time.Second).Should(
					HaveLen(1),
				)
				Eventually(func() []metrics.MetricEnvelope {
					return metricsEmitter.envelopesReceived
				}, 3*time.Second).Should(
					And(
						ContainElement(
							metrics.MetricEnvelope{
								InstanceGUID: "instance-guid1",
								Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
							},
						),
						ContainElement(
							metrics.MetricEnvelope{
								InstanceGUID: "instance-guid1",
								Metric:       metrics.Metric{Key: "bar", Value: 3.0, Unit: "s"},
							},
						),
					),
				)
			})

			It("should start only one worker for the supported types", func() {
				brokerInfo.On(
					"ListInstances", mock.Anything,
				).Return(
					[]brokerinfo.InstanceInfo{
						{GUID: "instance-guid1", Type: "fake2"},
					}, nil,
				)
				metricsCollectorDriver2.On(
					"NewCollector", mock.Anything,
				).Return(
					metricsCollector2, nil,
				)
				metricsCollector2.On(
					"Collect",
				).Return(
					[]metrics.Metric{
						metrics.Metric{Key: "bar", Value: 3, Unit: "s"},
					},
					nil,
				)

				go scheduler.Run(signals, ready)

				Eventually(func() []string {
					return scheduler.ListIntanceGUIDs()
				}, 1*time.Second).Should(
					HaveLen(1),
				)
				Consistently(func() []string {
					return scheduler.ListIntanceGUIDs()
				}, 2*time.Second).Should(
					HaveLen(1),
				)
				Eventually(func() []metrics.MetricEnvelope {
					return metricsEmitter.envelopesReceived
				}, 2*time.Second).Should(
					ContainElement(
						metrics.MetricEnvelope{
							InstanceGUID: "instance-guid1",
							Metric:       metrics.Metric{Key: "bar", Value: 3.0, Unit: "s"},
						},
					),
				)
				Consistently(func() []metrics.MetricEnvelope {
					return metricsEmitter.envelopesReceived
				}, 2*time.Second).ShouldNot(
					ContainElement(
						metrics.MetricEnvelope{
							InstanceGUID: "instance-guid1",
							Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
						},
					),
				)

			})

			It("should stop workers when one instance disappears", func() {
				metricsCollectorDriver2.On(
					"NewCollector", mock.Anything,
				).Return(
					metricsCollector2, nil,
				)

				metricsCollector2.On(
					"Collect",
				).Return(
					[]metrics.Metric{
						metrics.Metric{Key: "bar", Value: 3, Unit: "s"},
					},
					nil,
				)
				metricsCollector2.On(
					"Close", mock.Anything,
				).Return(
					nil,
				)

				// First loop returns 2 instances
				brokerInfo.On(
					"ListInstances", mock.Anything,
				).Return(
					[]brokerinfo.InstanceInfo{
						{GUID: "instance-guid1", Type: "fake"},
						{GUID: "instance-guid2", Type: "fake"},
					}, nil,
				).Once()

				// After return only one instance
				brokerInfo.On(
					"ListInstances", mock.Anything,
				).Return(
					[]brokerinfo.InstanceInfo{
						{GUID: "instance-guid1", Type: "fake"},
					}, nil,
				)

				go scheduler.Run(signals, ready)

				Eventually(func() []string {
					return scheduler.ListIntanceGUIDs()
				}, 3*time.Second).Should(
					HaveLen(2),
				)

				Eventually(func() []metrics.MetricEnvelope {
					return metricsEmitter.envelopesReceived
				}, 1*time.Second).Should(
					And(
						ContainElement(
							metrics.MetricEnvelope{
								InstanceGUID: "instance-guid2",
								Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
							},
						),
						ContainElement(
							metrics.MetricEnvelope{
								InstanceGUID: "instance-guid2",
								Metric:       metrics.Metric{Key: "bar", Value: 3.0, Unit: "s"},
							},
						),
					),
				)

				Eventually(func() []string {
					return scheduler.ListIntanceGUIDs()
				}, 2*time.Second).Should(
					HaveLen(1),
				)

				// Clear received envelopes
				metricsEmitter.envelopesReceived = metricsEmitter.envelopesReceived[:0]

				Consistently(func() []metrics.MetricEnvelope {
					return metricsEmitter.envelopesReceived
				}, 2*time.Second).ShouldNot(
					Or(
						ContainElement(
							metrics.MetricEnvelope{
								InstanceGUID: "instance-guid2",
								Metric:       metrics.Metric{Key: "foo", Value: 1.0, Unit: "b"},
							},
						),
						ContainElement(
							metrics.MetricEnvelope{
								InstanceGUID: "instance-guid2",
								Metric:       metrics.Metric{Key: "bar", Value: 3.0, Unit: "s"},
							},
						),
					),
				)
			})

			It("should immediately start all drivers although one may be slow in creating a new collector", func() {
				metricsCollectorDriver2.On(
					"NewCollector", mock.Anything,
				).Return(
					metricsCollector2, nil,
				)

				metricsCollector2.On(
					"Collect",
				).Return(
					[]metrics.Metric{
						metrics.Metric{Key: "bar", Value: 3, Unit: "s"},
					},
					nil,
				)
				metricsCollector2.On(
					"Close", mock.Anything,
				).Return(
					nil,
				)

				brokerInfo.On(
					"ListInstances", mock.Anything,
				).Return(
					[]brokerinfo.InstanceInfo{
						{GUID: "instance-guid1", Type: "fake"},
					}, nil,
				)

				metricsCollectorDriverNewCollectorCall.After(700 * time.Millisecond)

				go scheduler.Run(signals, ready)

				// Wait for the collector to collect metrics at least once
				Eventually(func() []metrics.MetricEnvelope {
					return metricsEmitter.envelopesReceived
				}, 500*time.Millisecond).Should(
					ContainElement(
						metrics.MetricEnvelope{
							InstanceGUID: "instance-guid1",
							Metric:       metrics.Metric{Key: "bar", Value: 3.0, Unit: "s"},
						},
					),
				)
			})
		})
	})
})
