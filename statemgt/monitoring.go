package statemgt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
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

func getTimeDurationFromURLQuery(paramName string, req *http.Request, defaultValue time.Duration) (time.Duration, error) {
	queryValues := req.URL.Query()
	queryTimeframe := queryValues.Get(paramName)
	if queryTimeframe != "" {
		timeframe, err := strconv.Atoi(queryTimeframe)
		if err != nil {
			// timeframe must be int
			return 0, err
		}
		return time.Duration(timeframe) * time.Second, nil
	}

	// key not found
	// not error but default needs to be used
	return defaultValue, nil
}

func (s *StateMgt) GetMetricsOfAllRoutes(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	timeframe, err := getTimeDurationFromURLQuery("timeframe", req, DefaultTimeframe)
	granularity, err := getTimeDurationFromURLQuery("granularity", req, timeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
		return
	}

	if timeframe < granularity {
		returnError(w, req, 400, fmt.Errorf("Timeframe must be greater than granularity"), nil)
		return
	}

	data, err := s.Gateway.MetricsRepo.ReadAllRoutes(time.Now().Add(-timeframe), time.Now(), granularity)
	if err != nil {
		log.Errorf(err.Error())
		returnError(w, req, 400, err, nil)
		return
	}
	marshalAndReturn(w, req, data)
}

func (s *StateMgt) GetMetricsOfAllBackends(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	timeframe, err := getTimeDurationFromURLQuery("timeframe", req, DefaultTimeframe)
	granularity, err := getTimeDurationFromURLQuery("granularity", req, timeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
		return
	}
	if timeframe < granularity {
		returnError(w, req, 400, fmt.Errorf("Timeframe must be greater than granularity"), nil)
		return
	}

	data, err := s.Gateway.MetricsRepo.ReadAllBackends(time.Now().Add(-timeframe), time.Now(), granularity)
	if err != nil {
		log.Errorf(err.Error())
		returnError(w, req, 400, err, nil)
		return
	}
	marshalAndReturn(w, req, data)
}

// GetMetricsData is used solely for debugging. Returns the data-map of Storage
func (s *StateMgt) GetMetricsData(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	data := s.Gateway.MetricsRepo.Storage.ReadData()

	b, err := json.Marshal(data)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func (s *StateMgt) GetMetricsOfBackend(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	id := ps.ByName("id")

	timeframe, err := getTimeDurationFromURLQuery("timeframe", req, DefaultTimeframe)
	granularity, err := getTimeDurationFromURLQuery("granularity", req, timeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
		return
	}

	if timeframe < granularity {
		returnError(w, req, 400, fmt.Errorf("Timeframe must be greater than granularity"), nil)
		return
	}

	// parse uuid
	backendID, err := uuid.Parse(id)
	if err != nil {
		// failure in uuid.Parse
		returnError(w, req, 400, err, nil)
		return
	}

	data, err := s.Gateway.MetricsRepo.ReadBackend(backendID, time.Now().Add(-timeframe), time.Now(), granularity)

	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	marshalAndReturn(w, req, data)

}

// GetMetricsOfRoute returns all metrics for the route
func (s *StateMgt) GetMetricsOfRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")

	timeframe, err := getTimeDurationFromURLQuery("timeframe", req, DefaultTimeframe)
	granularity, err := getTimeDurationFromURLQuery("granularity", req, timeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
		return
	}

	if timeframe < granularity {
		returnError(w, req, 400, fmt.Errorf("Timeframe must be greater than granularity"), nil)
		return
	}

	data, err := s.Gateway.MetricsRepo.ReadRoute(routeName, time.Now().Add(-timeframe), time.Now(), granularity)

	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	marshalAndReturn(w, req, data)
}

func (s *StateMgt) GetPromMetrics(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	var metrics interface{}

	route := req.URL.Query().Get("route")
	backend := req.URL.Query().Get("backend")

	if route != "" {
		if metricsOfRoute, found := s.Gateway.MetricsRepo.PromMetrics.Metrics[route]; found {
			metrics = metricsOfRoute
		}
	} else if backend != "" {

		backendID, err := uuid.Parse(backend)
		if err != nil {
			err := fmt.Errorf("Unable to parse backendID from query parameter (%v)", err)
			log.Error(err)
			returnError(w, req, 400, err, nil)
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
	marshalAndReturn(w, req, metrics)
}

func (s *StateMgt) GetActiveAlerts(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	alerts := s.Gateway.MetricsRepo.GetActiveAlerts()

	marshalAndReturn(w, req, alerts)
}
