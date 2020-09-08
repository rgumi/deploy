package statemgt

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

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

func (s *StateMgt) GetMetrics(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	timeframe, err := getTimeframeFromURLQuery(req, 10)
	if err != nil {
		// failure in atoi
		http.Error(w, err.Error(), 400)
	}

	b, err := json.Marshal(s.Gateway.MetricsRepo.Storage.ReadAll(
		time.Now().Add(-timeframe), time.Now()))

	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
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

	timeframe, err := getTimeframeFromURLQuery(req, 10)
	if err != nil {
		// failure in atoi
		http.Error(w, err.Error(), 400)
		return
	}
	// parse uuid
	id, err := uuid.Parse(backendID)
	if err != nil {
		// failure in uuid.Parse
		http.Error(w, err.Error(), 400)
		return
	}

	// get data
	data, err := s.Gateway.MetricsRepo.Storage.ReadBackend(id, time.Now().Add(-timeframe), time.Now())
	if err != nil {
		// failure in ReadBackend
		http.Error(w, err.Error(), 500)
		return
	}

	// marshal data
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

// GetMetricsOfRoute returns all metrics for the route
func (s *StateMgt) GetMetricsOfRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")

	timeframe, err := getTimeframeFromURLQuery(req, 10)
	if err != nil {
		// failure in atoi
		http.Error(w, err.Error(), 400)
		return
	}

	// marshal data
	b, err := json.Marshal(s.Gateway.MetricsRepo.Storage.ReadRoute(routeName, time.Now().Add(-timeframe), time.Now()))

	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}
