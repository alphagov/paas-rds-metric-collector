package collector

import (
	"sort"
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
	"CPUUtilization":   "cpu",
	"CPUCreditUsage":   "cpu_credit_usage",
	"CPUCreditBalance": "cpu_credit_balance",
	"FreeableMemory":   "freeable_memory",
	"FreeStorageSpace": "free_storage_space",
	"SwapUsage":        "swap_usage",
	"ReadIOPS":         "read_iops",
	"WriteIOPS":        "write_iops",
}

// NewCloudWatchCollectorDriver ...
func NewCloudWatchCollectorDriver(intervalSeconds int, session client.ConfigProvider, brokerInfo brokerinfo.BrokerInfo, logger lager.Logger) MetricsCollectorDriver {
	return &CloudWatchCollectorDriver{
		collectInterval: intervalSeconds,
		session:         session,
		brokerInfo:      brokerInfo,
		logger:          logger,
	}
}

// CloudWatchCollectorDriver ...
type CloudWatchCollectorDriver struct {
	collectInterval int
	session         client.ConfigProvider
	brokerInfo      brokerinfo.BrokerInfo
	logger          lager.Logger
}

// NewCollector ...
func (cw *CloudWatchCollectorDriver) NewCollector(instanceInfo brokerinfo.InstanceInfo) (MetricsCollector, error) {
	return &CloudWatchCollector{
		client:   cloudwatch.New(cw.session),
		instance: cw.brokerInfo.GetInstanceName(instanceInfo),
		logger:   cw.logger,
	}, nil
}

// GetName ...
func (cw *CloudWatchCollectorDriver) GetName() string {
	return "cloudwatch"
}

func (cw *CloudWatchCollectorDriver) SupportedTypes() []string {
	return []string{"postgres", "mysql"}
}

func (cw *CloudWatchCollectorDriver) GetCollectInterval() int {
	return cw.collectInterval
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
			StartTime:  aws.Time(time.Now().Add(-10 * time.Minute)),
			EndTime:    aws.Time(time.Now()),
			Statistics: []*string{aws.String("Average")},
		}

		cw.logger.Debug("GetMetricStatistics", lager.Data{
			"GetMetricStatisticsInput": *input,
		})
		data, err := cw.client.GetMetricStatistics(input)
		if err != nil {
			cw.logger.Error("querying cloudwatch metrics", err, lager.Data{
				"metricName":   metricName,
				"instanceGUID": cw.instance,
			})
			continue
		}

		cw.logger.Debug("GetMetricStatistics", lager.Data{
			"GetMetricStatisticsOutput": *data,
		})

		if len(data.Datapoints) > 0 {
			cw.logger.Debug("retrieved_metric", lager.Data{
				"metric_name": metricName,
			})

			// Get latest datapoint for this metric type
			sort.Slice(data.Datapoints, func(i, j int) bool {
				a := aws.TimeValue(data.Datapoints[i].Timestamp).UnixNano()
				b := aws.TimeValue(data.Datapoints[j].Timestamp).UnixNano()
				return a < b
			})
			d := data.Datapoints[len(data.Datapoints)-1]

			m = append(m, metrics.Metric{
				Key:       label,
				Timestamp: aws.TimeValue(d.Timestamp).UnixNano(),
				Value:     aws.Float64Value(d.Average),
				Unit:      strings.ToLower(*d.Unit),
				Tags: map[string]string{
					"source": "cloudwatch",
				},
			})
		} else {
			cw.logger.Debug("no_metrics_retrieved")
		}
	}

	return m, nil
}

// Close ...
func (cw *CloudWatchCollector) Close() error {
	return nil
}
