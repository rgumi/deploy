package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type LocalStorage struct {
	pufferMux       sync.RWMutex
	mux             sync.RWMutex                      // concurrent rw on maps is not possible
	puffer          map[string]map[uuid.UUID][]Metric // puffer storage until the averaging job is executed
	RetentionPeriod time.Duration                     // time after which an entry is deleted from storage
	Granularity     time.Duration                     // time after which the puffer is read and averages are saved in data
	killChan        chan int

	data map[string]map[uuid.UUID]map[time.Time]Metric // map of backend to metrics
}

func NewLocalStorage(retentionPeriod, granularity time.Duration) *LocalStorage {
	st := new(LocalStorage)
	st.data = make(map[string]map[uuid.UUID]map[time.Time]Metric)
	st.puffer = make(map[string]map[uuid.UUID][]Metric)
	st.killChan = make(chan int, 1)

	st.RetentionPeriod = retentionPeriod
	st.Granularity = granularity

	// time for averiging and saving to data
	go st.Job()
	return st
}

// Stop stops the job loop
func (st *LocalStorage) Stop() {
	log.Warn("Shutting down storage")
	st.killChan <- 1
}

// Job loops over puffer and makes an average of all metrics
// that were collected in $interval
// the averages are then written to data with the current time
// Also old entries in data are removed periodicly
func (st *LocalStorage) Job() {
	for {
		select {
		case _ = <-st.killChan:
			// exit loop
			return
		case _ = <-time.After(st.Granularity):
			go func() {
				// Lock & Unlock data
				st.mux.Lock()
				st.pufferMux.Lock()
				defer st.mux.Unlock()

				st.readPuffer()
				st.pufferMux.Unlock()
				st.deleteOldData()
			}()
		}
	}
}

func (st *LocalStorage) Write(
	routeName string,
	backend uuid.UUID,
	customMetrics map[string]float64,
	responseTime, contentLength int64,
	responseStatus int) {

	// this only writes to putter. Therefore, lock pufferMux
	st.pufferMux.Lock()
	defer st.pufferMux.Unlock()

	if _, found := st.puffer[routeName]; !found {
		st.puffer[routeName] = make(map[uuid.UUID][]Metric)
	}

	tmpMetric := Metric{
		ResponseTime:  float64(responseTime),
		ContentLength: float64(contentLength),
		CustomMetrics: customMetrics,
	}
	tmpMetric.TotalResponses++

	switch status := responseStatus; {
	case status < 300:
		tmpMetric.ResponseStatus200++
	case status < 400:
		tmpMetric.ResponseStatus300++
	case status < 500:
		tmpMetric.ResponseStatus400++
	case status < 600:
		tmpMetric.ResponseStatus500++
	default:
		tmpMetric.ResponseStatus600++
	}

	st.puffer[routeName][backend] = append(st.puffer[routeName][backend], tmpMetric)
}

// ReadData returns the whole data map
func (st *LocalStorage) ReadData() map[string]map[uuid.UUID]map[time.Time]Metric {
	st.mux.RLock()
	defer st.mux.RUnlock()
	return st.data
}

// ReadBackend returns all metrics for the backend that are within the given timeframe
func (st *LocalStorage) ReadBackend(backend uuid.UUID, start, end time.Time) (Metric, error) {
	st.mux.RLock()
	defer st.mux.RUnlock()

	for _, backendMap := range st.data {
		for id, metrics := range backendMap {
			if id == backend {
				relevantMetrics := []Metric{}
				for time, metric := range metrics {
					if time.After(start) && time.Before(end) {
						relevantMetrics = append(relevantMetrics, metric)
					}
				}
				if len(relevantMetrics) == 0 {
					return Metric{}, fmt.Errorf("Could not find relevant metrics for provided timeframe")
				}
				return makeAverageBackend(relevantMetrics), nil
			}
		}
	}
	// not found
	return Metric{}, fmt.Errorf("Could not find provided backend %v", backend)
}

// ReadRoute returns all metrics for the route that are within the given timeframe
func (st *LocalStorage) ReadRoute(route string, start, end time.Time) (Metric, error) {
	st.mux.RLock()
	defer st.mux.RUnlock()

	if routeData, found := st.data[route]; found {
		// get the averages for this route
		finalMetric := Metric{}
		finalMetric.CustomMetrics = make(map[string]float64)
		relevantMetrics := []Metric{}

		for _, backend := range routeData {
			for time, metric := range backend {
				if time.After(start) && time.Before(end) {
					relevantMetrics = append(relevantMetrics, metric)
				}
			}
		}
		if len(relevantMetrics) == 0 {
			return Metric{}, fmt.Errorf("Could not find relevant metrics for provided timeframe")
		}
		return makeAverageBackend(relevantMetrics), nil
	}
	// not found
	return Metric{}, fmt.Errorf("Could not find provided route %v", route)
}

/*
	Helper functions

*/
func makeAverageBackend(in []Metric) Metric {
	finalMetric := Metric{}
	finalMetric.CustomMetrics = make(map[string]float64)
	length := len(in)

	if length == 0 {
		return Metric{}
	}

	for _, metric := range in {
		finalMetric.ContentLength += metric.ContentLength

		// do not count failed requests with responsetime 0
		if metric.ResponseTime > 0 {
			finalMetric.ResponseTime += metric.ResponseTime
		}

		finalMetric.TotalResponses += metric.TotalResponses
		finalMetric.ResponseStatus200 += metric.ResponseStatus200
		finalMetric.ResponseStatus300 += metric.ResponseStatus300
		finalMetric.ResponseStatus400 += metric.ResponseStatus400
		finalMetric.ResponseStatus500 += metric.ResponseStatus500
		finalMetric.ResponseStatus600 += metric.ResponseStatus600

		for key, val := range metric.CustomMetrics {
			finalMetric.CustomMetrics[key] += val
		}
	}
	finalMetric.ContentLength = finalMetric.ContentLength / float64(length)
	finalMetric.ResponseTime = finalMetric.ResponseTime / float64(length)

	for key, val := range finalMetric.CustomMetrics {
		finalMetric.CustomMetrics[key] = val / float64(length)
	}

	return finalMetric
}

func (st *LocalStorage) readPuffer() {
	now := time.Now()
	for routeName, routeData := range st.puffer {
		for backendID, backendData := range routeData {
			// no new data
			if len(st.puffer[routeName][backendID]) == 0 {
				continue
			}
			if _, found := st.data[routeName]; !found {
				st.data[routeName] = make(map[uuid.UUID]map[time.Time]Metric)
				st.data[routeName][backendID] = make(map[time.Time]Metric)
			} else {
				if _, found := st.data[routeName][backendID]; !found {
					st.data[routeName][backendID] = make(map[time.Time]Metric)
				}
			}
			// write pufferdata to data
			st.data[routeName][backendID][now] = makeAverageBackend(backendData)
			// empty puffer
			st.puffer[routeName][backendID] = []Metric{}
		}
	}
}
func (st *LocalStorage) deleteOldData() {
	now := time.Now()
	// for each route
	for _, routeData := range st.data {
		// for each backend of route
		for _, backendData := range routeData {
			// for each timestamped, averaged metric of backend
			for timestamp := range backendData {
				// "full table scan" as go maps are not sorted
				if timestamp.Add(st.RetentionPeriod).Before(now) {
					// metric is out of retention period => delete it
					delete(backendData, timestamp)
				}
			}
		}
	}
}
