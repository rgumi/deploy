package metrics

import (
	"bufio"
	"bytes"
	"depoy/conditional"
	"depoy/storage"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	defaultMetricRates = []string{
		"ContentLength",
		"ResponseTime",
		"2xxRate",
		"3xxRate",
		"4xxRate",
		"5xxRate",
		"6xxRate",
	}
)

type Storage interface {
	Write(string, uuid.UUID, map[string]float64, int64, int64, int)
	ReadAllRoutes(start, end time.Time) map[string]storage.Metric
	ReadAllBackends(start, end time.Time) map[string]map[uuid.UUID]storage.Metric
	ReadData() map[string]map[uuid.UUID]map[time.Time]storage.Metric
	ReadBackend(backend uuid.UUID, start, end time.Time) (storage.Metric, error)
	ReadRoute(route string, start, end time.Time) storage.Metric
}

type Alert struct {
	Type       string    `json:"type" yaml:"type"`
	BackendID  uuid.UUID `json:"backend_id" yaml:"backendID"`
	Metric     string    `json:"metric" yaml:"metric"`
	Threshhold float64   `json:"threshold" yaml:"treshold"`
	Value      float64   `json:"value" yaml:"value"`
	StartTime  time.Time
	EndTime    time.Time
	SendTime   time.Time
}

type Metrics struct {
	Route                string
	BackendID            uuid.UUID
	ResponseStatus       int
	RequestMethod        string
	ContentLength        int64
	UpstreamResponseTime int64
	UpstreamRequestTime  int64
	DownstreamAddr       string
}

type ScrapeMetrics struct {
	BackendID uuid.UUID
	Metrics   map[string]float64
}

type MonitoredBackend struct {
	ID                 uuid.UUID
	ScrapeURL          string
	Errors             int
	nextTimeout        time.Duration
	MetricThreshholds  []*conditional.Condition
	AlertChannel       chan Alert        `yaml:"-" json:"-"`
	stopMonitoring     chan int          `yaml:"-" json:"-"` // Channel to kill Monitor-Loop
	activeAlerts       map[string]*Alert `json:"-" yaml:"-"`
	ScrapeMetrics      []string
	ScrapeMetricPuffer map[string]float64 `yaml:"-" json:"-"`
}

type Repository struct {
	Storage              Storage                         `yaml:"-" json:"-"`
	PromMetrics          *PromMetrics                    `yaml:"-" json:"-"`
	ScrapeInterval       time.Duration                   `yaml:"scrape_interval" json:"scrapeInterval"`
	client               *http.Client                    `yaml:"-" json:"-"`
	InChannel            chan (Metrics)                  `yaml:"-" json:"-"`
	scrapeMetricsChannel chan (ScrapeMetrics)            `yaml:"-" json:"-"`
	Backends             map[uuid.UUID]*MonitoredBackend `yaml:"backends" json:"backends"`
	stopScraping         chan int                        `yaml:"-" json:"-"`
	shutdown             chan int                        `yaml:"-" json:"-"`
}

// NewMetricsRepository creates a new instance of NewMetricsRepository
// return a channel for Metrics
func NewMetricsRepository(st Storage, scrapeInterval time.Duration) (chan<- Metrics, *Repository) {
	channel := make(chan Metrics, 50)
	scrapeMetricsChannel := make(chan ScrapeMetrics, 50)
	log.Debug("Created new metricsRepository")
	return channel, &Repository{
		Storage:              st,
		PromMetrics:          NewPromMetrics(),
		ScrapeInterval:       scrapeInterval,
		client:               http.DefaultClient,
		InChannel:            channel,
		Backends:             make(map[uuid.UUID]*MonitoredBackend),
		stopScraping:         make(chan int, 2), // CHannel to kill Scraping-Loop
		shutdown:             make(chan int, 2), // Channel to kill Listen-Loop
		scrapeMetricsChannel: scrapeMetricsChannel,
	}
}

// RegisterBackend adds a new instance to the ScrapingJob
func (m *Repository) RegisterBackend(
	routeName string,
	backendID uuid.UUID,
	scrapeURL string,
	scrapeMetrics []string,
	metricsTresholds []*conditional.Condition) (<-chan Alert, error) {

	// check if backendID is already configured
	for key := range m.Backends {
		if key == backendID {
			return nil, fmt.Errorf("instance with ID %v already exists", key)
		}
	}

	newBackend := &MonitoredBackend{
		ID:                 backendID,
		ScrapeURL:          scrapeURL,
		Errors:             0,
		nextTimeout:        0,
		MetricThreshholds:  metricsTresholds,
		ScrapeMetrics:      scrapeMetrics,
		ScrapeMetricPuffer: make(map[string]float64),
		AlertChannel:       make(chan Alert),
		stopMonitoring:     make(chan int, 1),
		activeAlerts:       make(map[string]*Alert),
	}

	// add to PromMetrics
	m.PromMetrics.RegisterRouteBackend(routeName, backendID)

	// append to the list
	m.Backends[backendID] = newBackend
	return newBackend.AlertChannel, nil
}

// RemoveBackend removes the instance with backendID from the scrapeList
func (m *Repository) RemoveBackend(backendID uuid.UUID) error {

	log.Warnf("Removing MontioringBackend for BackendID: %v", backendID)

	// check if backendID is exists and delete
	for key := range m.Backends {
		if key == backendID {
			// stop monitoring job of backend
			m.Backends[key].stopMonitoring <- 1

			// Unregister backend
			// time.Sleep(2 * time.Second)
			delete(m.Backends, key)

			return nil
		}
	}

	return fmt.Errorf("Could not find instance with ID %v", backendID)
}

// Stop cancels the Listen()-Loop and channels are no longer read
func (m *Repository) Stop() {
	log.Debug("Shutting down listening loop")
	m.shutdown <- 1
	m.stopScraping <- 1

	for _, b := range m.Backends {
		b.stopMonitoring <- 1
	}
}

// Monitor stats the monitor loop which checks every $timeout interval
// if an alert needs to be sent
// $activeFor defines for how long a threshhold needs to be reached to
// send an alert
func (m *Repository) Monitor(
	backendID uuid.UUID, timeout time.Duration, activeFor time.Duration) error {
	/*
		defer func() {
			if err := recover(); err != nil {
				log.Warnf("Closing Monitor for Backend %v with Error: %v", backendID, err)
			}
		}()
	*/
	if backend, ok := m.Backends[backendID]; ok {
		log.Debugf("Starting monitoring of backend %v", backend.ID)

		for {
			select {
			case _ = <-backend.stopMonitoring:
				return nil

			default:
				// read the collected metric from the storage
				// may be an average over 1s, 10s, 1m? Configure that
				now := time.Now()
				collected, err := m.ReadRatesOfBackend(backendID, now.Add(-10*time.Second), now)

				if err != nil {
					log.Debugf("Unable to obtain rates of backend for the last 10 seconds (%v)", err)
					time.Sleep(timeout)
					continue
				}

				log.Debug(collected)
				// loop over every metric that was collected
				// collected := map[string]float64
				for _, condition := range backend.MetricThreshholds {

					// get the treshhold for this metric
					// this has to exist otherwise it would not have been collected
					isReached, err := condition.IsTrue(collected)
					currentValue := collected[condition.Metric]

					if err != nil {
						log.Errorf("Used condition is not valid (%v)", err)
						time.Sleep(timeout)
						continue
					}

					// check if an alert already exists for this metric
					if alert, ok := backend.activeAlerts[condition.Metric]; ok {
						// alert exists already
						// check if it is still active
						if isReached {
							log.Debugf("Threshhold still reached for Alert %v", alert)
							// threshhold is still reached and alert remains up
							// goto next metric

							// update value to current value
							alert.Value = currentValue

							// check if alert existed for long enough to send an alert
							now := time.Now()
							if now.After(alert.StartTime.Add(condition.GetActiveFor())) && alert.SendTime.IsZero() {
								log.Debugf("Sending alarm for %v", alert)
								alert.Type = "Alarming"
								alert.SendTime = now
								backend.AlertChannel <- *alert
								log.Debugf("Send alarm for %v", alert)
							}

							// goto next metric
							continue
						}

						// treshhold is no longer reached

						if alert.EndTime.IsZero() {
							alert.EndTime = time.Now()
						}

						if time.Now().After(alert.EndTime.Add(condition.GetActiveFor())) {
							alert.Type = "Resolved"
							alert.Value = currentValue
							backend.AlertChannel <- *alert
							delete(backend.activeAlerts, condition.Metric)

							log.Debugf("Resolved Alert for %v", alert)
						}

						continue
					}

					// new alarm for metric

					if isReached {
						alert := &Alert{
							Type:       "Pending",
							BackendID:  backend.ID,
							Metric:     condition.Metric,
							Threshhold: condition.Threshold,
							Value:      collected[condition.Metric],
							StartTime:  time.Now(),
						}
						backend.activeAlerts[condition.Metric] = alert

						log.Debugf("New alert registered: %v", alert)
					}
				}
			}
			time.Sleep(timeout)
		}
	}
	return fmt.Errorf("Could not find backend with id %v", backendID)

}

// Listen listens on all channels and adds Metrics to the storage
// alarms when a treshhold is reached
func (m *Repository) Listen() {

	// start the scraping Loop
	go m.jobLoop()

	for {
		select {
		// received the metrics struct with backendID and all metrics
		case metrics := <-m.InChannel:

			log.Debug(metrics)

			// update PromMetrics

			go m.PromMetrics.Update(
				float64(metrics.UpstreamResponseTime), float64(metrics.ContentLength),
				metrics.ResponseStatus, metrics.RequestMethod, metrics.Route, metrics.BackendID)

			// update Prometheus Metrics

			TotalHttpRequests.With(
				prometheus.Labels{
					"route":   metrics.Route,
					"backend": metrics.BackendID.String(),
					"code":    strconv.Itoa(metrics.ResponseStatus),
					"method":  metrics.RequestMethod}).Inc()

			AvgResponseTime.With(
				prometheus.Labels{
					"route":   metrics.Route,
					"backend": metrics.BackendID.String()}).Set(m.PromMetrics.GetAvgResponseTime(metrics.Route, metrics.BackendID))

			AvgContentLength.With(
				prometheus.Labels{
					"route":   metrics.Route,
					"backend": metrics.BackendID.String()}).Set(m.PromMetrics.GetAvgContentLength(metrics.Route, metrics.BackendID))

			// Get Scrape Metrics and persist to Storage

			scrapeMetrics := m.Backends[metrics.BackendID].ScrapeMetricPuffer
			if scrapeMetrics == nil {

				m.Storage.Write(
					metrics.Route, metrics.BackendID, nil, metrics.UpstreamResponseTime,
					metrics.ContentLength, metrics.ResponseStatus)
			} else {

				m.Storage.Write(
					metrics.Route, metrics.BackendID, scrapeMetrics, metrics.UpstreamResponseTime,
					metrics.ContentLength, metrics.ResponseStatus)
			}

		case scrapeMetrics := <-m.scrapeMetricsChannel:

			log.Trace(scrapeMetrics)
			m.Backends[scrapeMetrics.BackendID].ScrapeMetricPuffer = scrapeMetrics.Metrics

		case _ = <-m.shutdown:
			return
		}
	}
}

// scrapeJob scraped the given instance, extracts the defined metrics
// and pushes them into the scrapeMetricsChannel
func (m *Repository) scrapeJob(instance *MonitoredBackend) {

	// timeout if last scrape was an error
	time.Sleep(instance.nextTimeout)

	req, err := http.NewRequest("GET", instance.ScrapeURL, nil)
	if err != nil {
		// should never happen
		panic(err)
	}
	log.Tracef("Scraping instance %v", instance.ID)
	resp, err := m.client.Do(req)
	if err != nil {
		instance.Errors++
		instance.nextTimeout = time.Duration(instance.Errors) * time.Second
		return
	}

	// reset errors counter
	instance.Errors = 0
	instance.nextTimeout = 0

	// got response therefore extract metricValues
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}
	defer resp.Body.Close()

	metrics := ScrapeMetrics{
		BackendID: instance.ID,
		Metrics:   map[string]float64{},
	}

	for _, name := range instance.ScrapeMetrics {
		bodyReader := bytes.NewReader(body)
		value, err := getRowFromBody(bodyReader, name)
		if err != nil {
			log.Error(err)
		}
		metrics.Metrics[name] = value
	}

	// finished extracting metric values from scrape
	m.scrapeMetricsChannel <- metrics
}

// jobLoop is a loop which executes all ScrapeInstances and waits ScrapeInterval
// for each ScrapeInstance a goroutine scrapeJob is started
func (m *Repository) jobLoop() {

	// loop over all scrapeInstances, get metrics and then sleep
	for {
		select {
		case _ = <-m.stopScraping:
			return

		default:
			for _, instance := range m.Backends {
				if instance.ScrapeURL != "" {
					go m.scrapeJob(instance)
				}
			}
			time.Sleep(m.ScrapeInterval)
		}
	}

}

// GetMetricsForBackend retrieves the metrics for $backend and the given timeframe ($end-$start)
// with the average over $granularity seconds
// e. g. retrieve for backend1 last 1min of metrics with granularity 10s would return 6 values with each
// representing the granularity over 10s
func (m *Repository) GetMetricsForBackend(
	backend uuid.UUID, start, end time.Time, granularity time.Duration) map[time.Time]storage.Metric {

	// get the delta of start and end
	timeDelta := end.Sub(start)

	dataPoints := int(math.Abs(float64(timeDelta / granularity)))
	log.Warnf("Using %v datapoints", dataPoints)

	metricData := make(map[time.Time]storage.Metric, dataPoints)
	start = start

	for idx := 0; idx < dataPoints; idx++ {
		end := start.Add(time.Duration((dataPoints-idx)*int(granularity)) * time.Second)

		val, err := m.Storage.ReadBackend(backend, start, end)
		log.Warn(val)
		if err != nil {
			log.Errorf("Unable to get requested metrics for backend %v (%v)", backend, err)
			return nil
		}
		metricData[end] = val
		start = end
	}

	return metricData
}

// GetMetricsForRoute retrieves the metrics for $route and the given timeframe ($end-$start)
// with the average over $avg seconds
// e. g. retrieve for route1 last 1min of metrics with avg 10s would return 6 values with each
// representing the avg over 10s
func (m *Repository) GetMetricsForRoute(route string, start, end time.Time, avg time.Duration) map[string]float64 {
	return nil
}

// ReadRatesOfBackend makes rates (average) of all metrics of the backend within the given timeframe
func (m *Repository) ReadRatesOfBackend(backend uuid.UUID, start, end time.Time) (map[string]float64, error) {

	metricRates := make(map[string]float64)

	current, err := m.Storage.ReadBackend(backend, start, end)

	if err != nil {
		return nil, err
	}
	// there were no responses yet
	if current.TotalResponses == 0 {
		current.TotalResponses = 1
	}

	metricRates["2xxRate"] = float64(current.ResponseStatus200 / current.TotalResponses)
	metricRates["3xxRate"] = float64(current.ResponseStatus300 / current.TotalResponses)
	metricRates["4xxRate"] = float64(current.ResponseStatus400 / current.TotalResponses)
	metricRates["5xxRate"] = float64(current.ResponseStatus500 / current.TotalResponses)
	metricRates["6xxRate"] = float64(current.ResponseStatus600 / current.TotalResponses)
	metricRates["ResponseTime"] = current.ResponseTime
	metricRates["ContentLength"] = float64(current.ContentLength)

	for customScrapeMetricName, customScrapeMetricValue := range current.CustomMetrics {
		metricRates[customScrapeMetricName] = customScrapeMetricValue
	}
	return metricRates, nil
}

/*

	Helper functions

*/

// Source: https://gist.github.com/yyscamper/5657c360fadd6701580f3c0bcca9f63a
func parseFloat(str string) (float64, error) {
	val, err := strconv.ParseFloat(str, 64)
	if err == nil {
		return val, nil
	}

	//Some number may be seperated by comma, for example, 23,120,123, so remove the comma firstly
	str = strings.Replace(str, ",", "", -1)

	//Some number is specifed in scientific notation
	pos := strings.IndexAny(str, "eE")
	if pos < 0 {
		return strconv.ParseFloat(str, 64)
	}

	var baseVal float64
	var expVal int64

	baseStr := str[0:pos]
	baseVal, err = strconv.ParseFloat(baseStr, 64)
	if err != nil {
		return 0, err
	}

	expStr := str[(pos + 1):]
	expVal, err = strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return baseVal * math.Pow10(int(expVal)), nil
}

// getRowFromBody reads the body line by line (sep=\n) and checks if the given pattern
// exists. Returns the value that indeicated by the pattern
// Prometheus format: pattern *space* value
func getRowFromBody(body io.Reader, pattern string) (float64, error) {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {

		// Prometheus scrape format is metricName space metricValue
		substrings := strings.Split(scanner.Text(), " ")

		// Comment rows start with #
		if substrings[0] == "#" {
			continue
		}
		if substrings[0] == pattern {
			i, err := parseFloat(substrings[1])
			if err != nil {
				return -1, err
			}
			return i, nil
		}

	}
	return -1, fmt.Errorf("Could not find value for given pattern %s", pattern)
}

func appendToMap(puffer map[string][]float64, input map[string]float64) {
	for key, val := range input {
		puffer[key] = append(puffer[key], val)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
