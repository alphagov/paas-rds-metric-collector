package collector

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/mock"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo/fakebrokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector/mocks"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("cloudwatch_collector", func() {
	Context("CloudWatchCollectorDriver", func() {
		var metricsCollectorDriver MetricsCollectorDriver

		BeforeEach(func() {
			brokerInfo := &fakebrokerinfo.FakeBrokerInfo{}
			brokerInfo.On(
				"GetInstanceName", mock.Anything,
			).Return(
				"mydb",
			)

			s := session.New()
			metricsCollectorDriver = NewCloudWatchCollectorDriver(s, brokerInfo, logger)
		})

		It("should create a NewCollector successfully", func() {

			c, err := metricsCollectorDriver.NewCollector("__TEST_INSTANCE_ID__")
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
		})

		It("shall return the name", func() {
			Expect(metricsCollectorDriver.GetName()).To(Equal("cloudwatch"))
		})
	})

	Context("CloudWatchCollector", func() {
		var fakeClient *mocks.FakeCloudWatchAPI
		var collector CloudWatchCollector

		BeforeEach(func() {
			fakeClient = &mocks.FakeCloudWatchAPI{}
			collector = CloudWatchCollector{
				client:   fakeClient,
				instance: "mydb",
				logger:   logger,
			}
		})

		It("should fail to Collect metrics due to invalid API response", func() {
			fakeClient.GetMetricStatisticsReturns(nil, fmt.Errorf("__CONTROLLED_ERROR__"))

			data, err := collector.Collect()
			Expect(err).To(HaveOccurred())
			Expect(data).To(BeNil())
		})

		It("should Collect metrics successfully", func() {
			fakeClient.GetMetricStatisticsReturns(&cloudwatch.GetMetricStatisticsOutput{
				Label: aws.String("test"),
				Datapoints: []*cloudwatch.Datapoint{
					&cloudwatch.Datapoint{
						Average: aws.Float64(1),
						Unit:    aws.String("Second"),
					},
					&cloudwatch.Datapoint{
						Average: aws.Float64(2),
						Unit:    aws.String("Second"),
					},
					&cloudwatch.Datapoint{
						Average: aws.Float64(3),
						Unit:    aws.String("Second"),
					},
				},
			}, nil)

			data, err := collector.Collect()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())
			Expect(data).To(HaveLen(3))
			Expect(data[1].Unit).To(Equal("second"))
		})
	})
})
