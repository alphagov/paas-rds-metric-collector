package metrics

// Metric ...
type Metric struct {
	Key   string
	Value float64
	Unit  string
}

// MetricEnvelope ...
type MetricEnvelope struct {
	InstanceGUID string
	Metric       Metric
}
