package metrics

import (
	"bufio"
	"bytes"
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
	log "github.com/sirupsen/logrus"
)

type Storage interface {
	Write(string, uuid.UUID, map[string]float64, int64, int64, int)
	ReadRates(backend uuid.UUID, lastSeconds time.Duration) map[string]float64
	ReadAll(start, end time.Time) map[string]storage.Metric
	ReadData() map[string]map[uuid.UUID]map[time.Time]storage.Metric
}

type Alert struct {
	Type       string
	BackendID  uuid.UUID
	Metric     string
	Threshhold float64
	Value      float64
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
	UpstreamAddr         string
	DownstreamAddr       string
}

type ScrapeMetrics struct {
	BackendID uuid.UUID
	Metrics   map[string]float64
}

type Backend struct {
	ID                  uuid.UUID
	ScrapeURL           string
	Errors              int
	nextTimeout         time.Duration
	MetricThreshholds   map[string]float64 // metricName threshhold
	AlertChannel        chan Alert
	stopMonitoring      chan int
	activeAlerts        map[string]*Alert
	pufferMetricStorage map[string][]float64
}

func (b *Backend) QuitMonitoring() {
	b.stopMonitoring <- 1
}

type Repository struct {
	Storage              Storage
	ScrapeInterval       time.Duration
	client               *http.Client
	InChannel            chan (Metrics)
	scrapeMetricsChannel chan (ScrapeMetrics)
	Backends             map[uuid.UUID]*Backend
	stopScraping         chan int
	shutdown             chan int
}

// NewMetricsRepository creates a new instance of NewMetricsRepository
// return a channel for Metrics
func NewMetricsRepository(storage Storage, scrapeInterval time.Duration) (chan<- Metrics, *Repository) {
	channel := make(chan Metrics, 10)
	scrapeMetricsChannel := make(chan ScrapeMetrics, 10)
	return channel, &Repository{
		Storage:              storage,
		ScrapeInterval:       scrapeInterval,
		client:               http.DefaultClient,
		InChannel:            channel,
		Backends:             map[uuid.UUID]*Backend{},
		stopScraping:         make(chan int, 2),
		shutdown:             make(chan int, 2),
		scrapeMetricsChannel: scrapeMetricsChannel,
	}
}

// RegisterBackend adds a new instance to the ScrapingJob
func (m *Repository) RegisterBackend(
	backendID uuid.UUID,
	scrapeURL string,
	metricThreshholds map[string]float64) (<-chan Alert, error) {

	// check if backendID is already configured
	for key := range m.Backends {
		if key == backendID {
			return nil, fmt.Errorf("instance with ID %v already exists", key)
		}
	}

	newBackend := &Backend{
		ID:                  backendID,
		ScrapeURL:           scrapeURL,
		Errors:              0,
		nextTimeout:         0,
		MetricThreshholds:   metricThreshholds,
		AlertChannel:        make(chan Alert),
		stopMonitoring:      make(chan int, 2),
		activeAlerts:        make(map[string]*Alert),
		pufferMetricStorage: map[string][]float64{},
	}

	// append to the list
	m.Backends[backendID] = newBackend
	return newBackend.AlertChannel, nil
}

// RemoveBackend removes the instance with backendID from the scrapeList
func (m *Repository) RemoveBackend(backendID uuid.UUID) error {

	log.Debugf("Received RemoveScrapeInstance for ID: %v", backendID)

	// check if backendID is exists and delete
	for key := range m.Backends {
		if key == backendID {
			// remove
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Monitor stats the monitor loop which checks every $timeout interval
// if an alert needs to be sent
// $activeFor defines for how long a threshhold needs to be reached to
// send an alert
func (m *Repository) Monitor(
	backendID uuid.UUID, timeout time.Duration, activeFor time.Duration) error {

	defer func() {
		if err := recover(); err != nil {
			log.Warnf("Closing Monitor for Backend %v with Error: %v", backendID, err)
		}
	}()

	if backend, ok := m.Backends[backendID]; ok {
		log.Debugf("Starting monitoring of backend %v", backend.ID)

		for {
			select {
			case _ = <-backend.stopMonitoring:
				return nil

			default:
				// read the collected metric from the storage
				// may be an average over 1s, 10s, 1m? Configure that
				collected := m.Storage.ReadRates(backendID, 10)

				// loop over every metric that was collected
				// collected := map[string]float64
				for metricName, threshhold := range backend.MetricThreshholds {

					// get the treshhold for this metric
					// this has to exist otherwise it would not have been collected
					currentValue := collected[metricName]

					// check if an alert already exists for this metric
					if alert, ok := backend.activeAlerts[metricName]; ok {
						// alert exists already
						// check if it is still active
						if currentValue > threshhold {
							log.Debugf("Threshhold still reached for Alert %v", alert)
							// threshhold is still reached and alert remains up
							// goto next metric

							// update value to current value
							alert.Value = currentValue

							// check if alert existed for long enough to send an alert
							now := time.Now()
							if now.After(alert.StartTime.Add(activeFor)) && alert.SendTime.IsZero() {
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

						if time.Now().After(alert.EndTime.Add(activeFor)) {
							alert.Type = "Resolved"
							alert.Value = currentValue
							backend.AlertChannel <- *alert
							delete(backend.activeAlerts, metricName)

							log.Debugf("Resolved Alert for %v", alert)
						}

						continue
					}

					// new alarm for metric

					if currentValue > threshhold {
						alert := &Alert{
							Type:       "Pending",
							BackendID:  backend.ID,
							Metric:     metricName,
							Threshhold: threshhold,
							Value:      currentValue,
							StartTime:  time.Now(),
						}
						backend.activeAlerts[metricName] = alert

						log.Debugf("New alert registered: %v", alert)
					}
				}

				// timeout after first round of alarming
				time.Sleep(timeout)
			}
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

			m.Storage.Write(
				metrics.Route, metrics.BackendID, nil, metrics.UpstreamResponseTime,
				metrics.ContentLength, metrics.ResponseStatus)

		case scrapeMetrics := <-m.scrapeMetricsChannel:

			log.Debug(scrapeMetrics)

		case _ = <-m.shutdown:
			return
		}

	}
}

// scrapeJob scraped the given instance, extracts the defined metrics
// and pushes them into the scrapeMetricsChannel
func (m *Repository) scrapeJob(instance *Backend) {

	// timeout if last scrape was an error
	time.Sleep(instance.nextTimeout)

	req, err := http.NewRequest("GET", instance.ScrapeURL, nil)
	if err != nil {
		// should never happen
		panic(err)
	}
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

	for name := range instance.MetricThreshholds {
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
			time.Sleep(m.ScrapeInterval)
			for _, instance := range m.Backends {
				if instance.ScrapeURL != "" {
					go m.scrapeJob(instance)
				}
			}
		}
	}

}

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
