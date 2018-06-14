package collector

import (
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-rds-metric-collector/pkg/brokerinfo"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"

	"github.com/alphagov/paas-rds-metric-collector/pkg/metrics"
)

var metricNamesToLabels = map[string]string{
	"CPUUtilization": "cpu",
}

// NewCloudWatchCollectorDriver ...
func NewCloudWatchCollectorDriver(session client.ConfigProvider, brokerInfo brokerinfo.BrokerInfo, logger lager.Logger) *CloudWatchCollectorDriver {
	return &CloudWatchCollectorDriver{
		session:    session,
		brokerInfo: brokerInfo,
		logger:     logger,
	}
}

// CloudWatchCollectorDriver ...
type CloudWatchCollectorDriver struct {
	session    client.ConfigProvider
	brokerInfo brokerinfo.BrokerInfo
	logger     lager.Logger
}

// NewCollector ...
func (cw *CloudWatchCollectorDriver) NewCollector(instanceID string) (MetricsCollector, error) {
	return &CloudWatchCollector{
		client:   cloudwatch.New(cw.session),
		instance: cw.brokerInfo.GetInstanceName(instanceID),
		logger:   cw.logger,
	}, nil
}

// GetName ...
func (cw *CloudWatchCollectorDriver) GetName() string {
	return "cloudwatch"
}

// CloudWatchCollector ...
type CloudWatchCollector struct {
	client   cloudwatchiface.CloudWatchAPI
	instance string
	logger   lager.Logger
}

// Collect ...
func (cw *CloudWatchCollector) Collect() ([]metrics.Metric, error) {
	m := []metrics.Metric{}

	for metricName, label := range metricNamesToLabels {
		input := &cloudwatch.GetMetricStatisticsInput{
			Dimensions: []*cloudwatch.Dimension{
				&cloudwatch.Dimension{
					Name:  aws.String("DBInstanceIdentifier"),
					Value: aws.String(cw.instance),
				},
			},
			MetricName: aws.String(metricName),
			Namespace:  aws.String("AWS/RDS"),
			Period:     aws.Int64(60),
			StartTime:  aws.Time(time.Now().Add(-2 * time.Minute)),
			EndTime:    aws.Time(time.Now().Add(-1 * time.Minute)),
			Statistics: []*string{aws.String("Average")},
		}

		cw.logger.Debug("GetMetricStatistics", lager.Data{
			"GetMetricStatisticsInput": *input,
		})
		data, err := cw.client.GetMetricStatistics(input)
		if err != nil {
			return nil, err
		}

		cw.logger.Debug("GetMetricStatistics", lager.Data{
			"GetMetricStatisticsOutput": *data,
		})
		for _, d := range data.Datapoints {
			m = append(m, metrics.Metric{
				Key:       label,
				Timestamp: aws.TimeValue(d.Timestamp).UnixNano(),
				Value:     aws.Float64Value(d.Average),
				Unit:      strings.ToLower(*d.Unit),
			})
		}
	}

	return m, nil
}

// Close ...
func (cw *CloudWatchCollector) Close() error {
	return nil
}
