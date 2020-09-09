package statemgt

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/prometheus/common/log"
)

var defaultTimeframe = 10 * time.Second

type ErrorMessage struct {
	StatusCode int       `json:"status_code"`
	StatusText string    `json:"status_text"`
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
	Details    []string  `json:"details"`
}

func returnError(w http.ResponseWriter, req *http.Request, errCode int, err error, details []string) {
	msg := ErrorMessage{
		Timestamp:  time.Now(),
		Message:    err.Error(),
		Details:    details,
		StatusCode: errCode,
		StatusText: http.StatusText(errCode),
	}

	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(errCode)
	w.Write(b)
}

func getTimeframeFromURLQuery(req *http.Request, defaultValue time.Duration) (time.Duration, error) {
	queryValues := req.URL.Query()
	queryTimeframe := queryValues.Get("timeframe")
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

	timeframe, err := getTimeframeFromURLQuery(req, defaultTimeframe)
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

	timeframe, err := getTimeframeFromURLQuery(req, defaultTimeframe)
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

	timeframe, err := getTimeframeFromURLQuery(req, defaultTimeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
		return
	}
	// parse uuid
	id, err := uuid.Parse(backendID)
	if err != nil {
		// failure in uuid.Parse
		returnError(w, req, 400, err, nil)
		return
	}

	// get data
	data, err := s.Gateway.MetricsRepo.Storage.ReadBackend(id, time.Now().Add(-timeframe), time.Now())
	if err != nil {
		// failure in ReadBackend, wrong timeframe or backend not found
		returnError(w, req, 400, err, nil)
		return
	}

	// marshal data
	b, err := json.Marshal(data)

	if err != nil {
		log.Errorf(err.Error())
		returnError(w, req, 500, err, nil)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

// GetMetricsOfRoute returns all metrics for the route
func (s *StateMgt) GetMetricsOfRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")

	timeframe, err := getTimeframeFromURLQuery(req, defaultTimeframe)
	if err != nil {
		// failure in atoi
		returnError(w, req, 400, err, nil)
		return
	}

	// marshal data
	b, err := json.Marshal(s.Gateway.MetricsRepo.Storage.ReadRoute(routeName, time.Now().Add(-timeframe), time.Now()))

	if err != nil {
		log.Errorf(err.Error())
		returnError(w, req, 500, err, nil)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func (s *StateMgt) GetPromMetrics(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	b, err := json.Marshal(s.Gateway.MetricsRepo.PromMetrics.Metrics)
	if err != nil {
		returnError(w, req, 500, err, nil)
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}
