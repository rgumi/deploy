package statemgt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/prometheus/common/log"
	"github.com/rgumi/depoy/storage"
)

var (
	defaultTimeframe = 10 * time.Second
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

	timeframe, err := getTimeDurationFromURLQuery("timeframe", req, defaultTimeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
	}

	b, err := json.Marshal(s.Gateway.MetricsRepo.Storage.ReadAllRoutes(
		time.Now().Add(-timeframe), time.Now()))

	if err != nil {
		log.Errorf(err.Error())
		returnError(w, req, 500, err, nil)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func (s *StateMgt) GetMetricsOfAllBackends(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	timeframe, err := getTimeDurationFromURLQuery("timeframe", req, defaultTimeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
	}

	b, err := json.Marshal(s.Gateway.MetricsRepo.Storage.ReadAllBackends(
		time.Now().Add(-timeframe), time.Now()))

	if err != nil {
		log.Errorf(err.Error())
		returnError(w, req, 500, err, nil)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

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

	backendID := ps.ByName("id")

	timeframe, err := getTimeDurationFromURLQuery("timeframe", req, defaultTimeframe)
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
	id, err := uuid.Parse(backendID)
	if err != nil {
		// failure in uuid.Parse
		returnError(w, req, 400, err, nil)
		return
	}
	if _, found := s.Gateway.MetricsRepo.Backends[id]; !found {
		returnError(w, req, 404, fmt.Errorf("Could not find backend with id %v", id), nil)
		return
	}

	if timeframe == granularity {
		data, err := s.Gateway.MetricsRepo.Storage.ReadBackend(id, time.Now().Add(-timeframe), time.Now())
		if err != nil {
			// failure in ReadBackend, wrong timeframe or backend not found
			data = storage.Metric{}
		}

		marshalAndReturn(w, req, data)

	} else {
		steps := int(timeframe / granularity)
		data := make(map[time.Time]storage.Metric, steps)

		for i := 0; i < steps; i++ {
			maxTime := time.Now().Add(-timeframe + granularity)
			data[maxTime], err = s.Gateway.MetricsRepo.Storage.ReadBackend(id, time.Now().Add(-timeframe), maxTime)
			if err != nil {
				// errors are ignored and just empty metrics are returned instead
				data[maxTime] = storage.Metric{}
			}

			timeframe = timeframe - granularity
		}
		marshalAndReturn(w, req, data)
	}

}

// GetMetricsOfRoute returns all metrics for the route
func (s *StateMgt) GetMetricsOfRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")

	timeframe, err := getTimeDurationFromURLQuery("timeframe", req, defaultTimeframe)
	granularity, err := getTimeDurationFromURLQuery("granularity", req, timeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
		return
	}

	if timeframe == granularity {
		data := s.Gateway.MetricsRepo.Storage.ReadRoute(routeName, time.Now().Add(-timeframe), time.Now())
		if err != nil {
			data = storage.Metric{}
		}
		marshalAndReturn(w, req, data)

	} else {
		steps := int(timeframe / granularity)
		data := make(map[time.Time]storage.Metric, steps)

		for i := 0; i < steps; i++ {
			maxTime := time.Now().Add(-timeframe + granularity)
			data[maxTime] = s.Gateway.MetricsRepo.Storage.ReadRoute(routeName, time.Now().Add(-timeframe), maxTime)
			timeframe = timeframe - granularity
			if err != nil {
				// errors are ignored and just empty metrics are returned instead
				data[maxTime] = storage.Metric{}
			}

		}
		marshalAndReturn(w, req, data)
	}
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
