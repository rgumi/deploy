package metrics

import (
	"sync"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

// PromMetric stores all metrics of a Backend for the runtime
// it is cumulative
// it is used by Prometheus to expose metrics
type PromMetric struct {
	TotalResponses    int64
	ResponseStatus200 int64
	ResponseStatus300 int64
	ResponseStatus400 int64
	ResponseStatus500 int64
	ResponseStatus600 int64
	ContentLength     float64
	ResponseTime      float64
	GetRequest        int64
	PostRequest       int64
	DeleteRequest     int64
}

type PromMetrics struct {
	mux     sync.RWMutex
	Metrics map[string]map[uuid.UUID]*PromMetric
}

var (
	// TotalHTTPRequests is the total amount of http requests that were received
	TotalHTTPRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "total_http_requests",
			Help: "the total amount of http requests that were received",
		},
		[]string{"route", "backend", "code", "method"},
	)

	// AvgResponseTime is the average response time of the backend
	AvgResponseTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "average_response_time",
			Help: "the average response time of the backend",
		},
		[]string{"route", "backend"},
	)

	// AvgContentLength is the average content length of requests
	AvgContentLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "average_content_length",
			Help: "the average content length of requests",
		},
		[]string{"route", "backend"},
	)
)

func init() {
	prometheus.MustRegister(TotalHTTPRequests)
	prometheus.MustRegister(AvgResponseTime)
	prometheus.MustRegister(AvgContentLength)
}

func (p *PromMetrics) GetCurrentMetrics() map[string]map[uuid.UUID]*PromMetric {
	p.mux.RLock()
	defer p.mux.RUnlock()

	return p.Metrics
}

func NewPromMetrics() *PromMetrics {
	return &PromMetrics{
		Metrics: make(map[string]map[uuid.UUID]*PromMetric),
	}
}

func (p *PromMetrics) RegisterRouteBackend(routeName string, backend uuid.UUID) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if _, found := p.Metrics[routeName]; !found {
		p.Metrics[routeName] = make(map[uuid.UUID]*PromMetric)

	}
	p.Metrics[routeName][backend] = new(PromMetric)

	// already exist
}

func (p *PromMetrics) RemoveRouteBackend(routeName string, backend uuid.UUID) {
	p.mux.Lock()
	defer p.mux.Unlock()

	if _, found := p.Metrics[routeName]; found {
		delete(p.Metrics, routeName)
	}
}

func (p *PromMetrics) Update(
	responseTime, contentLength float64,
	responseStatus int, requestMethod string, routeName string, backend uuid.UUID) {

	p.mux.Lock()
	defer p.mux.Unlock()

	var newMetric *PromMetric
	var found bool

	if newMetric, found = p.Metrics[routeName][backend]; !found {
		// not registered
		return
	}

	newMetric.TotalResponses++

	switch status := responseStatus; {
	case status < 300:
		newMetric.ResponseStatus200++
	case status < 400:
		newMetric.ResponseStatus300++
	case status < 500:
		newMetric.ResponseStatus400++
	case status < 600:
		newMetric.ResponseStatus500++
	default:
		newMetric.ResponseStatus600++
	}

	switch method := requestMethod; {
	case method == "GET":
		newMetric.GetRequest++
	case method == "POST":
		newMetric.PostRequest++
	case method == "DELETE":
		newMetric.DeleteRequest++
	}

	newMetric.ResponseTime = floatingAverage(newMetric.ResponseTime, responseTime, float64(newMetric.TotalResponses))
	newMetric.ContentLength = floatingAverage(newMetric.ContentLength, contentLength, float64(newMetric.TotalResponses))
}

// GetAvgResponseTime returns the average response time of the given route/backend
// if no route/backend is found, -1 is returned
func (p *PromMetrics) GetAvgResponseTime(routeName string, backend uuid.UUID) float64 {
	if val, found := p.Metrics[routeName][backend]; found {
		return val.ResponseTime
	}
	return -1
}

// GetAvgContentLength returns the average response time of the given route/backend
// if no route/backend is found, -1 is returned
func (p *PromMetrics) GetAvgContentLength(routeName string, backend uuid.UUID) float64 {
	if val, found := p.Metrics[routeName][backend]; found {
		return val.ContentLength
	}
	return -1
}

/*

	Helper functions

*/

// https://math.stackexchange.com/questions/106700/incremental-averageing
func floatingAverage(a, x, k float64) float64 {
	if a == 0 {
		return x
	}
	return a + (x-a)/k
}
