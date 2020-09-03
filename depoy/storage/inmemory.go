package storage

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type LocalStorage struct {
	mux    sync.RWMutex
	puffer map[string]map[uuid.UUID][]Metric
	data   map[string]map[uuid.UUID]map[time.Time]Metric // map of backend to metrics
}

func NewLocalStorage() *LocalStorage {
	st := new(LocalStorage)
	st.data = make(map[string]map[uuid.UUID]map[time.Time]Metric)
	st.puffer = make(map[string]map[uuid.UUID][]Metric)

	// time for averiging and saving to data
	go st.Job(5)
	return st
}

func (st *LocalStorage) readPuffer() {

	for routeName, routeData := range st.puffer {

		for backendID, backendData := range routeData {
			if _, found := st.data[routeName]; !found {
				st.data[routeName] = make(map[uuid.UUID]map[time.Time]Metric)
				st.data[routeName][backendID] = make(map[time.Time]Metric)
			} else {

				if _, found := st.data[routeName][backendID]; !found {
					st.data[routeName][backendID] = make(map[time.Time]Metric)
				}
			}
			st.data[routeName][backendID][time.Now()] = makeAverageBackend(backendData)
			// empty puffer
			st.puffer[routeName][backendID] = []Metric{}
		}
	}
}

func (st *LocalStorage) deleteOldData() {

	deletePeriod := time.Duration(10 * time.Minute)
	for _, routeData := range st.data {
		for _, backendData := range routeData {
			for timestamp := range backendData {
				if timestamp.Add(deletePeriod).Before(time.Now()) {
					delete(backendData, timestamp)
				}
			}
		}
	}
}

func (st *LocalStorage) Job(interval int) {

	for {
		st.mux.Lock()
		st.deleteOldData()
		st.readPuffer()
		st.mux.Unlock()

		// sleep n seconds
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (st *LocalStorage) Write(
	routeName string,
	backend uuid.UUID,
	customMetrics map[string]float64,
	responseTime, contentLength int64,
	responseStatus int) {

	st.mux.Lock()
	defer st.mux.Unlock()

	if _, found := st.puffer[routeName]; !found {
		st.puffer[routeName] = make(map[uuid.UUID][]Metric)
	}

	tmpMetric := Metric{
		ResponseTime:  float64(responseTime),
		ContentLength: contentLength,
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

func (st *LocalStorage) ReadData() map[string]map[uuid.UUID]map[time.Time]Metric {

	return st.data
}

func (st *LocalStorage) ReadBackend(backend uuid.UUID, start, end time.Time) Metric {

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

				return makeAverageBackend(relevantMetrics)
			}
		}
	}
	return Metric{}
}

func (st *LocalStorage) ReadAll(start, end time.Time) map[string]Metric {

	m := make(map[string]Metric)

	for name := range st.data {
		metric := st.ReadRoute(name, start, end)
		m[name] = metric
	}
	return m
}

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
	finalMetric.ContentLength = finalMetric.ContentLength / int64(length)
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

	finalMetric.ContentLength = finalMetric.ContentLength / int64(length)
	finalMetric.ResponseTime = finalMetric.ResponseTime / float64(length)

	for key, val := range finalMetric.CustomMetrics {
		finalMetric.CustomMetrics[key] = val / float64(length)
	}
}

func (st *LocalStorage) ReadRates(backend uuid.UUID, lastSeconds time.Duration) map[string]float64 {

	st.mux.RLock()
	defer st.mux.RUnlock()

	m := make(map[string]float64)

	// select average from last 10s
	current := st.ReadBackend(backend, time.Now().Add(-lastSeconds*time.Second), time.Now())

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

	return m
}
