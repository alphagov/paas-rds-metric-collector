package metrics

// Metric ...
type Metric struct {
	Key       string
	Timestamp int64
	Value     float64
	Unit      string
	Tags      map[string]string
}

// MetricEnvelope ...
type MetricEnvelope struct {
	InstanceGUID string
	Metric       Metric
}
