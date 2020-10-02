package metrics

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rgumi/depoy/conditional"
	"github.com/rgumi/depoy/storage"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (

	// DefaultMetrics are the default metrics that are offered
	DefaultMetrics = []string{
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
	ReadData() map[string]map[uuid.UUID]map[time.Time]storage.Metric
	ReadBackend(backend uuid.UUID, start, end time.Time) (storage.Metric, error)
	ReadRoute(route string, start, end time.Time) (storage.Metric, error)
	Stop()
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
	Route              string
	ScrapeURL          string
	Errors             int
	nextTimeout        time.Duration
	MetricThreshholds  []*conditional.Condition
	AlertChannel       chan Alert
	stopMonitoring     chan int // Channel to kill Monitor-Loop
	stopScraping       chan int
	activeAlerts       map[string]*Alert
	ScrapeMetrics      []string
	ScrapeInterval     time.Duration
	ScrapeMetricPuffer map[string]float64
}

type Repository struct {
	Storage              Storage                         `yaml:"-" json:"-"`
	PromMetrics          *PromMetrics                    `yaml:"-" json:"-"`
	InChannel            chan (Metrics)                  `yaml:"-" json:"-"`
	Backends             map[uuid.UUID]*MonitoredBackend `yaml:"backends" json:"backends"`
	Granularity          time.Duration
	client               *http.Client
	scrapeMetricsChannel chan (ScrapeMetrics)
	shutdown             chan int
}

// NewMetricsRepository creates a new instance of NewMetricsRepository
// return a channel for Metrics
func NewMetricsRepository(
	st Storage, granularity time.Duration,
	metricChannelPuffersize, scrapeMetricChannelPuffersize int) (chan<- Metrics, *Repository) {

	channel := make(chan Metrics, metricChannelPuffersize)
	scrapeMetricsChannel := make(chan ScrapeMetrics, scrapeMetricChannelPuffersize)
	log.Info("Created new metricsRepository")
	repo := &Repository{
		Storage:              st,
		PromMetrics:          NewPromMetrics(),
		client:               http.DefaultClient,
		Granularity:          granularity,
		InChannel:            channel,
		Backends:             make(map[uuid.UUID]*MonitoredBackend),
		shutdown:             make(chan int, 1), // Channel to kill Listen-Loop
		scrapeMetricsChannel: scrapeMetricsChannel,
	}
	go repo.Listen()

	return channel, repo
}

// RegisterBackend adds a new instance to the ScrapingJob
func (m *Repository) RegisterBackend(
	routeName string,
	backendID uuid.UUID,
	scrapeURL string,
	scrapeMetrics []string,
	scrapeInterval time.Duration,
	metricsTresholds []*conditional.Condition) (<-chan Alert, error) {

	// check if backendID is already configured
	for key := range m.Backends {
		if key == backendID {
			return nil, fmt.Errorf("instance with ID %v already exists", key)
		}
	}

	newBackend := &MonitoredBackend{
		ID:                 backendID,
		Route:              routeName,
		ScrapeURL:          scrapeURL,
		Errors:             0,
		nextTimeout:        0,
		MetricThreshholds:  metricsTresholds,
		ScrapeInterval:     scrapeInterval,
		ScrapeMetrics:      scrapeMetrics,
		ScrapeMetricPuffer: make(map[string]float64),
		AlertChannel:       make(chan Alert),
		stopMonitoring:     make(chan int, 1),
		stopScraping:       make(chan int, 1),
		activeAlerts:       make(map[string]*Alert),
	}

	// add to PromMetrics
	m.PromMetrics.RegisterRouteBackend(routeName, backendID)
	go m.jobLoop(newBackend)

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
			m.Backends[key].stopScraping <- 1
			// Unregister backend
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

	for _, b := range m.Backends {
		b.stopMonitoring <- 1
		b.stopScraping <- 1
	}
	m.Storage.Stop()
}

// RegisterAlert adds an Alert to the backend for the provided metric
func (m *Repository) RegisterAlert(backendID uuid.UUID, alertType, metric string, threshold, value float64) {
	alert := &Alert{
		Type:       alertType,
		BackendID:  backendID,
		Metric:     metric,
		Threshhold: threshold,
		Value:      value,
		StartTime:  time.Now(),
		SendTime:   time.Time{},
		EndTime:    time.Time{},
	}
	if backend, found := m.Backends[backendID]; found {
		backend.activeAlerts[metric] = alert
		backend.AlertChannel <- *alert
	}
}

// Monitor starts the monitoring-loop of a Backend which checks every interval
// if an alert needs to be sent
// activeFor defines for how long a threshhold needs to be reached to
// send an alert
// resolveFor defines for how long a alert has to be inactive before resolving it
func (m *Repository) Monitor(backendID uuid.UUID, interval time.Duration) error {
	if backend, ok := m.Backends[backendID]; ok {
		log.Debugf("Starting monitoring of backend %v", backend.ID)
		for {
			select {
			case _ = <-backend.stopMonitoring:
				return nil
			default:
				now := time.Now()
				collected, _ := m.ReadRatesOfBackend(backendID, now.Add(-2*interval), now)
				log.Debugf("Rates of Backend %v: %v", backendID, collected)
				// loop over every metric that was collected
				for _, condition := range backend.MetricThreshholds {
					// get the treshhold for this metric
					// this has to exist otherwise it would not have been collected
					isReached := condition.IsTrue(collected)
					currentValue := collected[condition.Metric]
					// check if an alert already exists for this metric
					if alert, ok := backend.activeAlerts[condition.Metric]; ok {
						// check if it is still active
						if isReached {
							log.Debugf("Threshhold still reached for Alert %v", alert)
							alert.EndTime = time.Time{}
							// threshhold is still reached and alert remains up
							alert.Value = currentValue
							// Update the Prometheus-Gauge with the current number
							// of active alerts of the backend
							ActiveAlerts.With(
								prometheus.Labels{
									"route":   backend.Route,
									"backend": backend.ID.String(),
								},
							).Set(float64(len(backend.activeAlerts)))
							// check if alert existed for long enough to send an alert
							now := time.Now()
							if now.After(alert.StartTime.Add(condition.GetActiveFor())) && alert.SendTime.IsZero() {
								alert.Type = "Alarming"
								alert.SendTime = now
								backend.AlertChannel <- *alert
							}
							// goto next metric
							continue
						}
						// treshhold is no longer reached
						if alert.EndTime.IsZero() {
							alert.EndTime = time.Now()
						}
						// 0 is interpreted as indefinitely and therefore once an alarm is active,
						// the Backend will never be resolved again
						if condition.GetResolveIn() == 0 {
							continue
						}
						if time.Now().After(alert.EndTime.Add(condition.GetResolveIn())) {
							alert.Type = "Resolved"
							alert.Value = currentValue
							backend.AlertChannel <- *alert
							delete(backend.activeAlerts, condition.Metric)

							log.Debugf("Resolved Alert for %v", alert)
						}
						// goto next metric
						continue
					}
					// new alarm for metric aka not yet in backend.activeAlerts
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
						// sending pending alarming to backend
						backend.AlertChannel <- *alert
						log.Debugf("New alert registered: %v", alert)
					}
				}
			}
			time.Sleep(interval)
		}
	}
	return fmt.Errorf("Could not find backend with id %v", backendID)

}

// Listen listens on all channels and adds Metrics to the storage
// alarms when a treshhold is reached
func (m *Repository) Listen() {

	for {
		select {
		// received the metrics struct with backendID and all metrics
		case metrics := <-m.InChannel:

			log.Trace(metrics)

			// update PromMetrics

			go m.PromMetrics.Update(
				float64(metrics.UpstreamResponseTime), float64(metrics.ContentLength),
				metrics.ResponseStatus, metrics.RequestMethod, metrics.Route, metrics.BackendID)

			// update Prometheus Metrics

			TotalHTTPRequests.With(
				prometheus.Labels{
					"route":   metrics.Route,
					"backend": metrics.BackendID.String(),
					"code":    strconv.Itoa(metrics.ResponseStatus),
					"method":  metrics.RequestMethod},
			).Inc()

			AvgResponseTime.With(
				prometheus.Labels{
					"route":   metrics.Route,
					"backend": metrics.BackendID.String(),
					"code":    strconv.Itoa(metrics.ResponseStatus),
					"method":  metrics.RequestMethod},
			).Set(m.PromMetrics.GetAvgResponseTime(metrics.Route, metrics.BackendID))

			AvgContentLength.With(
				prometheus.Labels{
					"route":   metrics.Route,
					"backend": metrics.BackendID.String(),
					"code":    strconv.Itoa(metrics.ResponseStatus),
					"method":  metrics.RequestMethod},
			).Set(m.PromMetrics.GetAvgContentLength(metrics.Route, metrics.BackendID))

			// Get Scrape Metrics and persist to Storage
			backend, found := m.Backends[metrics.BackendID]

			// check if backend exists (to avoid nil pointer exc)
			if !found {
				continue
			}
			scrapeMetrics := backend.ScrapeMetricPuffer
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

			backend, found := m.Backends[scrapeMetrics.BackendID]

			// check if backend exists (to avoid nil pointer exc)
			if !found {
				continue
			}
			log.Trace(scrapeMetrics)
			backend.ScrapeMetricPuffer = scrapeMetrics.Metrics

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
func (m *Repository) jobLoop(b *MonitoredBackend) {

	// loop over all scrapeInstances, get metrics and then sleep
	for {
		select {
		case _ = <-b.stopScraping:
			return

		default:
			m.scrapeJob(b)
			time.Sleep(b.ScrapeInterval)
		}
	}

}

// ReadRatesOfBackend makes rates (average) of all metrics of the backend within the given timeframe
func (m *Repository) ReadRatesOfBackend(backend uuid.UUID, start, end time.Time) (map[string]float64, error) {
	metricRates := make(map[string]float64)
	current, err := m.Storage.ReadBackend(backend, start, end)

	// there were no responses yet => avoid divison by 0
	if current.TotalResponses == 0 {
		current.TotalResponses = 1
	}
	metricRates["2xxRate"] = float64(current.ResponseStatus200) / float64(current.TotalResponses)
	metricRates["3xxRate"] = float64(current.ResponseStatus300) / float64(current.TotalResponses)
	metricRates["4xxRate"] = float64(current.ResponseStatus400) / float64(current.TotalResponses)
	metricRates["5xxRate"] = float64(current.ResponseStatus500) / float64(current.TotalResponses)
	metricRates["6xxRate"] = float64(current.ResponseStatus600) / float64(current.TotalResponses)
	metricRates["ResponseTime"] = current.ResponseTime
	metricRates["ContentLength"] = float64(current.ContentLength)
	for customScrapeMetricName, customScrapeMetricValue := range current.CustomMetrics {
		metricRates[customScrapeMetricName] = customScrapeMetricValue
	}
	return metricRates, err
}

func (m *Repository) GetActiveAlerts() map[uuid.UUID]map[string]*Alert {
	alertMap := make(map[uuid.UUID]map[string]*Alert)
	for id, backend := range m.Backends {
		alertMap[id] = backend.activeAlerts
	}
	return alertMap
}

// ReadAllBackends returns all metrics by backend that are withing the given timeframe
func (m *Repository) ReadAllBackends(start, end time.Time, granularity time.Duration) (map[string]map[uuid.UUID]map[time.Time]storage.Metric, error) {

	metricsByBackends := make(map[string]map[uuid.UUID]map[time.Time]storage.Metric)
	for backendID, backend := range m.Backends {

		if _, found := metricsByBackends[backend.Route]; !found {
			metricsByBackends[backend.Route] = make(map[uuid.UUID]map[time.Time]storage.Metric)
		}

		metrics, err := m.ReadBackend(backendID, start, end, granularity)
		if err != nil {
			return nil, err
		}
		metricsByBackends[backend.Route][backendID] = metrics
	}

	return metricsByBackends, nil
}

// ReadAllRoutes returns all metrics in data that are within the given timeframe
func (m *Repository) ReadAllRoutes(start, end time.Time, granularity time.Duration) (map[string]map[time.Time]storage.Metric, error) {

	var err error

	metricsByRoute := make(map[string]map[time.Time]storage.Metric)

	for _, backend := range m.Backends {
		if _, found := metricsByRoute[backend.Route]; !found {
			metricsByRoute[backend.Route], err = m.ReadRoute(backend.Route, start, end, granularity)
		}
	}
	return metricsByRoute, err
}

func (m *Repository) ReadBackend(backendID uuid.UUID, start, end time.Time, granularity time.Duration) (map[time.Time]storage.Metric, error) {
	var err error
	if _, found := m.Backends[backendID]; !found {
		return nil, fmt.Errorf("Could not find backend with ID %v", backendID)
	}
	if granularity == 0 {
		granularity = m.Granularity
	}
	timeframe := end.Sub(start)
	// granualrity < MonitoringGranularity is ignored
	if timeframe < granularity {
		return nil, fmt.Errorf("Timeframe must be greater than granulartiy (%v > %v)", timeframe, granularity)
	}
	if timeframe == granularity {
		// only return avg over the timeframe
		data := make(map[time.Time]storage.Metric, 1)
		data[time.Now()], err = m.Storage.ReadBackend(backendID, start, end)
		return data, err
	}
	// number of timestamped entries
	steps := int(timeframe / granularity)
	data := make(map[time.Time]storage.Metric, steps)

	maxTime := start
	for i := 0; i < steps; i++ {
		maxTime = maxTime.Add(granularity)
		data[maxTime], err = m.Storage.ReadBackend(backendID, start, maxTime)
		if err != nil {
			// errors are ignored and just empty metrics are returned instead
			data[maxTime] = storage.Metric{}
		}
		start = maxTime
	}
	return data, nil
}

func (m *Repository) ReadRoute(routeName string, start, end time.Time, granularity time.Duration) (map[time.Time]storage.Metric, error) {
	var err error
	if granularity == 0 {
		granularity = m.Granularity
	}
	timeframe := end.Sub(start)
	// granualrity < MonitoringGranularity is ignored
	if timeframe < granularity {
		return nil, fmt.Errorf("Timeframe must be greater than granulartiy (%v must be larger than %v)", timeframe, granularity)
	}
	// only return avg over the timeframe
	if timeframe == granularity {
		data := make(map[time.Time]storage.Metric, 1)
		data[end], err = m.Storage.ReadRoute(routeName, start, end)
		return data, err
	}
	// number of timestamped entries
	steps := int(timeframe / granularity)
	data := make(map[time.Time]storage.Metric, steps)

	maxTime := start
	for i := 0; i < steps; i++ {
		maxTime = maxTime.Add(granularity)
		data[maxTime], err = m.Storage.ReadRoute(routeName, start, maxTime)
		if err != nil {
			// errors are ignored and just empty metrics are returned instead
			data[maxTime] = storage.Metric{}
		}
		start = maxTime
	}
	return data, nil
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
