package storage

var ()

type Metric struct {
	TotalResponses    int
	ResponseStatus200 int
	ResponseStatus300 int
	ResponseStatus400 int
	ResponseStatus500 int
	ResponseStatus600 int
	ContentLength     float64
	ResponseTime      float64
	CustomMetrics     map[string]float64
}
