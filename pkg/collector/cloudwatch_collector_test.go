package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/mock"

	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo/fakebrokerinfo"
	"github.com/alphagov/paas-rds-metric-collector/pkg/collector/mocks"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	. "github.com/onsi/ginkgo/v2"
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
			metricsCollectorDriver = NewCloudWatchCollectorDriver(5, s, brokerInfo, logger)
		})

		It("should create a NewCollector successfully", func() {

			c, err := metricsCollectorDriver.NewCollector(brokerinfo.InstanceInfo{GUID: "__TEST_INSTANCE_ID__"})
			Expect(err).NotTo(HaveOccurred())
			Expect(c).NotTo(BeNil())
		})

		It("shall return the name", func() {
			Expect(metricsCollectorDriver.GetName()).To(Equal("cloudwatch"))
		})

		It("should return the CollectInterval", func() {
			Expect(metricsCollectorDriver.GetCollectInterval()).To(Equal(5))
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

		It("should Collect metrics successfully", func() {
			now := time.Now()
			fakeClient.GetMetricStatisticsWithContextReturns(&cloudwatch.GetMetricStatisticsOutput{
				Label: aws.String("test"),
				Datapoints: []*cloudwatch.Datapoint{
					{
						Timestamp: aws.Time(now.Add(-3 * time.Second)),
						Average:   aws.Float64(1),
						Unit:      aws.String("Second"),
					},
					{
						Timestamp: aws.Time(now.Add(-2 * time.Second)),
						Average:   aws.Float64(2),
						Unit:      aws.String("Second"),
					},
					{
						Timestamp: aws.Time(now.Add(-1 * time.Second)),
						Average:   aws.Float64(3),
						Unit:      aws.String("Second"),
					},
				},
			}, nil)

			data, err := collector.Collect(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())
			Expect(data).NotTo(BeEmpty())
			Expect(data[0].Unit).To(Equal("second"))
			Expect(data[0].Value).To(Equal(3.0))
			Expect(data[0].Tags).To(HaveKeyWithValue("source", "cloudwatch"))
		})

		It("should preserve the timestamp", func() {
			metricTime := time.Now().Add(-1 * time.Hour)

			fakeClient.GetMetricStatisticsWithContextReturns(&cloudwatch.GetMetricStatisticsOutput{
				Label: aws.String("test"),
				Datapoints: []*cloudwatch.Datapoint{
					{
						Timestamp: aws.Time(metricTime),
						Average:   aws.Float64(1),
						Unit:      aws.String("Second"),
					},
				},
			}, nil)

			data, err := collector.Collect(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())
			Expect(data).NotTo(BeEmpty())
			Expect(data[0].Timestamp).To(Equal(metricTime.UnixNano()))
		})

		It("should continue to collect metrics when it hits an error", func() {
			fakeClient.GetMetricStatisticsWithContextReturnsOnCall(0, nil, fmt.Errorf("__CONTROLLED_ERROR__"))
			fakeClient.GetMetricStatisticsWithContextReturns(&cloudwatch.GetMetricStatisticsOutput{
				Label: aws.String("test"),
				Datapoints: []*cloudwatch.Datapoint{
					{
						Timestamp: aws.Time(time.Now()),
						Average:   aws.Float64(1),
						Unit:      aws.String("Second"),
					},
				},
			}, nil)
			data, err := collector.Collect(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())
			Expect(data).NotTo(BeEmpty())
			Expect(data[0].Unit).To(Equal("second"))
			Expect(data[0].Value).To(Equal(1.0))
			Expect(data[0].Tags).To(HaveKeyWithValue("source", "cloudwatch"))
		})

		It("should not fail if there are no datapoints", func() {
			fakeClient.GetMetricStatisticsWithContextReturns(&cloudwatch.GetMetricStatisticsOutput{
				Label:      aws.String("test"),
				Datapoints: []*cloudwatch.Datapoint{},
			}, nil)

			_, err := collector.Collect(context.Background())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
