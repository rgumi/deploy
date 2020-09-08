package storage

type Metric struct {
	TotalResponses    int
	ResponseStatus200 int
	ResponseStatus300 int
	ResponseStatus400 int
	ResponseStatus500 int
	ResponseStatus600 int
	ContentLength     int64
	ResponseTime      float64
	CustomMetrics     map[string]float64
}

type CustomMetrics struct {
}
