package statemgt

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

var (
	// DefaultTimeframe is the timeframe that will be set when the query-param is empty
	DefaultTimeframe = 10 * time.Second
)

type ErrorMessage struct {
	StatusCode int       `json:"status_code"`
	StatusText string    `json:"status_text"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
	Details    []string  `json:"details"`
}

func getTimeDurationFromURLQuery(paramName string, ctx *fasthttp.RequestCtx, defaultValue time.Duration) time.Duration {
	queryValue := ctx.QueryArgs().GetUfloatOrZero(paramName)
	if queryValue > 0 {
		return time.Duration(queryValue) * time.Second
	}
	// key not found
	// not error but default needs to be used
	return defaultValue
}

func (s *StateMgt) GetMetricsOfAllRoutes(ctx *fasthttp.RequestCtx) {
	timeframe := getTimeDurationFromURLQuery("timeframe", ctx, DefaultTimeframe)
	granularity := getTimeDurationFromURLQuery("granularity", ctx, timeframe)
	if timeframe < granularity {
		returnError(ctx, 400, fmt.Errorf("Timeframe must be greater than granularity"), nil)
		return
	}

	data, err := s.Gateway.MetricsRepo.ReadAllRoutes(time.Now().Add(-timeframe), time.Now(), granularity)
	if err != nil {
		log.Errorf(err.Error())
		returnError(ctx, 400, err, nil)
		return
	}
	marshalAndReturn(ctx, data)
}

func (s *StateMgt) GetMetricsOfAllBackends(ctx *fasthttp.RequestCtx) {

	timeframe := getTimeDurationFromURLQuery("timeframe", ctx, DefaultTimeframe)
	granularity := getTimeDurationFromURLQuery("granularity", ctx, timeframe)

	if timeframe < granularity {
		returnError(ctx, 400, fmt.Errorf("Timeframe must be greater than granularity"), nil)
		return
	}

	data, err := s.Gateway.MetricsRepo.ReadAllBackends(time.Now().Add(-timeframe), time.Now(), granularity)
	if err != nil {
		log.Errorf(err.Error())
		returnError(ctx, 400, err, nil)
		return
	}
	marshalAndReturn(ctx, data)
}

// GetMetricsData is used solely for debugging. Returns the data-map of Storage
func (s *StateMgt) GetMetricsData(ctx *fasthttp.RequestCtx) {
	data := s.Gateway.MetricsRepo.Storage.ReadData()
	marshalAndReturn(ctx, data)
}

func (s *StateMgt) GetMetricsOfBackend(ctx *fasthttp.RequestCtx) {

	id := string(ctx.QueryArgs().Peek("id"))

	timeframe := getTimeDurationFromURLQuery("timeframe", ctx, DefaultTimeframe)
	granularity := getTimeDurationFromURLQuery("granularity", ctx, timeframe)

	if timeframe < granularity {
		returnError(ctx, 400, fmt.Errorf("Timeframe must be greater than granularity"), nil)
		return
	}

	// parse uuid
	backendID, err := uuid.Parse(id)
	if err != nil {
		// failure in uuid.Parse
		returnError(ctx, 400, err, nil)
		return
	}

	data, err := s.Gateway.MetricsRepo.ReadBackend(backendID, time.Now().Add(-timeframe), time.Now(), granularity)

	if err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	marshalAndReturn(ctx, data)

}

// GetMetricsOfRoute returns all metrics for the route
func (s *StateMgt) GetMetricsOfRoute(ctx *fasthttp.RequestCtx) {
	routeName := string(ctx.QueryArgs().Peek("route"))

	timeframe := getTimeDurationFromURLQuery("timeframe", ctx, DefaultTimeframe)
	granularity := getTimeDurationFromURLQuery("granularity", ctx, timeframe)

	if timeframe < granularity {
		returnError(ctx, 400, fmt.Errorf("Timeframe must be greater than granularity"), nil)
		return
	}

	data, err := s.Gateway.MetricsRepo.ReadRoute(routeName, time.Now().Add(-timeframe), time.Now(), granularity)

	if err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	marshalAndReturn(ctx, data)
}

func (s *StateMgt) GetPromMetrics(ctx *fasthttp.RequestCtx) {

	var metrics interface{}

	route := string(ctx.QueryArgs().Peek("route"))
	backend := string(ctx.QueryArgs().Peek("backend"))
	if route != "" {
		if metricsOfRoute, found := s.Gateway.MetricsRepo.PromMetrics.Metrics[route]; found {
			metrics = metricsOfRoute
		}
	} else if backend != "" {

		backendID, err := uuid.Parse(backend)
		if err != nil {
			err := fmt.Errorf("Unable to parse backendID from query parameter (%v)", err)
			log.Error(err)
			returnError(ctx, 400, err, nil)
			return
		}

		for _, b := range s.Gateway.MetricsRepo.PromMetrics.Metrics {
			for id := range b {
				if id == backendID {
					metrics = b[id]
				}
			}
		}
	} else {
		metrics = s.Gateway.MetricsRepo.PromMetrics.Metrics
	}
	marshalAndReturn(ctx, metrics)
}

func (s *StateMgt) GetActiveAlerts(ctx *fasthttp.RequestCtx) {
	alerts := s.Gateway.MetricsRepo.GetActiveAlerts()
	marshalAndReturn(ctx, alerts)
}
