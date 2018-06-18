package collector

import (
	"fmt"
	"time"

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
			now := time.Now()
			fakeClient.GetMetricStatisticsReturns(&cloudwatch.GetMetricStatisticsOutput{
				Label: aws.String("test"),
				Datapoints: []*cloudwatch.Datapoint{
					&cloudwatch.Datapoint{
						Timestamp: aws.Time(now.Add(-3 * time.Second)),
						Average:   aws.Float64(1),
						Unit:      aws.String("Second"),
					},
					&cloudwatch.Datapoint{
						Timestamp: aws.Time(now.Add(-2 * time.Second)),
						Average:   aws.Float64(2),
						Unit:      aws.String("Second"),
					},
					&cloudwatch.Datapoint{
						Timestamp: aws.Time(now.Add(-1 * time.Second)),
						Average:   aws.Float64(3),
						Unit:      aws.String("Second"),
					},
				},
			}, nil)

			data, err := collector.Collect()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())
			Expect(data).To(HaveLen(1))
			Expect(data[0].Unit).To(Equal("second"))
			Expect(data[0].Value).To(Equal(3.0))
		})
		It("should preserve the timestamp", func() {
			metricTime := time.Now().Add(-1 * time.Hour)

			fakeClient.GetMetricStatisticsReturns(&cloudwatch.GetMetricStatisticsOutput{
				Label: aws.String("test"),
				Datapoints: []*cloudwatch.Datapoint{
					&cloudwatch.Datapoint{
						Timestamp: aws.Time(metricTime),
						Average:   aws.Float64(1),
						Unit:      aws.String("Second"),
					},
				},
			}, nil)

			data, err := collector.Collect()
			Expect(err).NotTo(HaveOccurred())
			Expect(data).NotTo(BeNil())
			Expect(data).To(HaveLen(1))
			Expect(data[0].Timestamp).To(Equal(metricTime.UnixNano()))
		})
		It("should not fail if there are no datapoints", func() {
			fakeClient.GetMetricStatisticsReturns(&cloudwatch.GetMetricStatisticsOutput{
				Label:      aws.String("test"),
				Datapoints: []*cloudwatch.Datapoint{},
			}, nil)

			_, err := collector.Collect()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
