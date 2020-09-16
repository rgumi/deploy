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

	// storage for all data that is stored longterm with increasing counter for e. g. Prometheus
	// for each route/backend
	// TotalRequests, 2xx, 3xx, 4xx, 5xx, 6xx Requests, avg Responsetime
	overview map[string]map[uuid.UUID]Metric
}

func NewLocalStorage() *LocalStorage {
	st := new(LocalStorage)
	st.data = make(map[string]map[uuid.UUID]map[time.Time]Metric)
	st.puffer = make(map[string]map[uuid.UUID][]Metric)
	st.overview = make(map[string]map[uuid.UUID]Metric)
	st.killChan = make(chan int, 1)

	st.RetentionPeriod = 10 * time.Minute
	st.Granularity = 5 * time.Second

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

		default:
			time.Sleep(st.Granularity)
			go func() {
				// Lock & Unlock data
				st.mux.Lock()
				defer st.mux.Unlock()

				// Lock & Unlock puffer
				st.pufferMux.Lock()
				defer st.pufferMux.Unlock()

				// execute operations
				st.deleteOldData()
				st.readPuffer()
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
		st.overview[routeName] = make(map[uuid.UUID]Metric)
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

// ReadAllRoutes returns all metrics in data that are within the given timeframe
func (st *LocalStorage) ReadAllRoutes(start, end time.Time) map[string]Metric {

	st.mux.RLock()
	defer st.mux.RUnlock()

	m := make(map[string]Metric)

	for name := range st.data {
		metric := st.ReadRoute(name, start, end)
		m[name] = metric
	}
	return m
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
	return Metric{}, fmt.Errorf("Could not find provided backend %v", backend)
}

// ReadAllBackends returns all metrics by backend that are withing the given timeframe
func (st *LocalStorage) ReadAllBackends(start, end time.Time) map[string]map[uuid.UUID]Metric {

	metricsByBackends := make(map[string]map[uuid.UUID]Metric)
	for routeName, backends := range st.data {

		metricsByBackends[routeName] = make(map[uuid.UUID]Metric)
		for backendID := range backends {

			m, err := st.ReadBackend(backendID, start, end)
			if err != nil {
				panic(err)
			}
			metricsByBackends[routeName][backendID] = m
		}
	}

	return metricsByBackends
}

// ReadRoute returns all metrics for the route that are within the given timeframe
func (st *LocalStorage) ReadRoute(route string, start, end time.Time) Metric {

	st.mux.RLock()
	defer st.mux.RUnlock()

	if routeData, found := st.data[route]; found {
		// get the averages for this route
		finalMetric := Metric{}
		finalMetric.CustomMetrics = make(map[string]float64)

		relevantMetrics := []Metric{}

		// this is wrong!!!! TODO: Change this so its works
		// die avg response time ist verfälscht, da das ein Backend, was kaputt ist
		// eine ResponseTime von 0 hat und somit den avg der anderen beiden runterzieht
		// Gewichtung beachten!!!

		for _, backend := range routeData {
			for time, metric := range backend {
				if time.After(start) && time.Before(end) {
					relevantMetrics = append(relevantMetrics, metric)
				}
			}
		}

		return makeAverageBackend(relevantMetrics)

	}

	// not found
	return Metric{}
}

// ReadRatesOfBackend makes rates (average) of all metrics of the backend within the given timeframe
func (st *LocalStorage) ReadRatesOfBackend(backend uuid.UUID, start, end time.Time) (map[string]float64, error) {

	m := make(map[string]float64)

	current, err := st.ReadBackend(backend, start, end)

	if err != nil {
		return nil, err
	}
	// there were no responses yet
	if current.TotalResponses == 0 {
		current.TotalResponses = 1
	}

	m["2xxRate"] = float64(current.ResponseStatus200 / current.TotalResponses)
	m["3xxRate"] = float64(current.ResponseStatus300 / current.TotalResponses)
	m["4xxRate"] = float64(current.ResponseStatus400 / current.TotalResponses)
	m["5xxRate"] = float64(current.ResponseStatus500 / current.TotalResponses)
	m["6xxRate"] = float64(current.ResponseStatus600 / current.TotalResponses)
	m["ResponseTime"] = current.ResponseTime
	m["ContentLength"] = float64(current.ContentLength)

	for customScrapeMetricName, customScrapeMetricValue := range current.CustomMetrics {
		m[customScrapeMetricName] = customScrapeMetricValue
	}
	return m, nil
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

func makeAverage(in map[uuid.UUID]Metric, finalMetric *Metric) {

	length := len(in)
	for _, metric := range in {
		//		length += len(metrics)
		//		for _, metric := range metrics {
		finalMetric.ContentLength += metric.ContentLength
		finalMetric.ResponseTime += metric.ResponseTime
		finalMetric.TotalResponses += metric.TotalResponses
		finalMetric.ResponseStatus200 += metric.ResponseStatus200
		finalMetric.ResponseStatus300 += metric.ResponseStatus300
		finalMetric.ResponseStatus400 += metric.ResponseStatus400
		finalMetric.ResponseStatus500 += metric.ResponseStatus500
		finalMetric.ResponseStatus600 += metric.ResponseStatus600

		for key, val := range metric.CustomMetrics {
			finalMetric.CustomMetrics[key] += val
		}
		//		}
	}

	finalMetric.ContentLength = finalMetric.ContentLength / float64(length)
	finalMetric.ResponseTime = finalMetric.ResponseTime / float64(length)

	for key, val := range finalMetric.CustomMetrics {
		finalMetric.CustomMetrics[key] = val / float64(length)
	}
}

func (st *LocalStorage) readPuffer() {

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
			st.data[routeName][backendID][time.Now()] = makeAverageBackend(backendData)
			// empty puffer
			st.puffer[routeName][backendID] = []Metric{}
		}
	}
}

func (st *LocalStorage) deleteOldData() {

	deletePeriod := st.RetentionPeriod

	// for each route
	for _, routeData := range st.data {

		//outer:
		// for each backend of route
		for _, backendData := range routeData {

			// for each timestamped, averaged metric of backend
			for timestamp := range backendData {
				if timestamp.Add(deletePeriod).Before(time.Now()) {
					// metric is out of retention period => delete it
					delete(backendData, timestamp)
				}
				// continue
			}
		}
	}
}
