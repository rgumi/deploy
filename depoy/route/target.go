package route

import (
	"bufio"
	"bytes"
	"depoy/upstreamclient"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// ScrapeMetric is the object which defines what metrics should be evaluated
// after a scrape is executed succesfully
// this metric is collected by matching ScrapeMetric.Name with scraped Prometheus
// string row by row and saving the corrosponding value
type ScrapeMetric struct {
	Name          string
	Threshhold    float64
	ScrapeCounter int
	ScrapeSum     float64
}

type Target struct {
	ID                uuid.UUID
	Name              string
	Addr              string
	MetricChan        chan upstreamclient.Metric
	KillChan          chan int
	Active            bool
	Scrape            bool
	ScrapeFailCounter int
	ScrapeAddr        string
	ScrapeMetrics     []*ScrapeMetric
	ScrapeInterval    time.Duration
}

func NewTarget(name, addr string) (*Target, error) {

	if name == "" {
		return nil, fmt.Errorf("Name cannot be null")
	}
	url, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	target := &Target{
		ID:         uuid.New(),
		Name:       name,
		Addr:       url.String(),
		MetricChan: make(chan upstreamclient.Metric),
		KillChan:   make(chan int),
		Active:     true,
	}

	go target.MetricListener()
	return target, nil
}

// AddScrapeMetric adds a new scrape metric to the target which will be scraped
// by the scrapejob
func (t *Target) AddScrapeMetric(name string, threshhold float64) error {
	if name == "" {
		return fmt.Errorf("Name cannot null")
	}
	scrapeMetrics := &ScrapeMetric{
		Name:          name,
		Threshhold:    threshhold,
		ScrapeCounter: 0,
		ScrapeSum:     0,
	}
	t.ScrapeMetrics = append(t.ScrapeMetrics, scrapeMetrics)

	return nil
}

// MetricListener changes the state of target depending on its application state
func (t *Target) MetricListener() {

	for {
		select {
		case _ = <-t.KillChan:
			return

		case m := <-t.MetricChan:
			log.Info(m)
		}

	}
}

func (t *Target) SetupScrape(addr string, interval time.Duration, scrapeMetrics []*ScrapeMetric) error {
	url, err := url.Parse(addr)
	if err != nil {
		return err
	}
	t.ScrapeAddr = url.String()
	t.Scrape = true
	t.ScrapeInterval = interval
	t.ScrapeMetrics = scrapeMetrics

	go t.scrapeJob()
	return nil
}

func (t *Target) handleError(err error) {
	// do something with error
	panic(err)
}

func (t *Target) scrapeJob() {
	for {
		select {
		case _ = <-t.KillChan:
			return

		default:
			if t.Scrape {
				t.DoScrape()
			}
			time.Sleep(t.ScrapeInterval)
		}
	}
}

// DoScrape executes a scrape on the target and writes the collectes
// metrics to the ScrapeMetric object
func (t *Target) DoScrape() {

	client := http.DefaultClient

	resp, err := client.Get(t.ScrapeAddr)
	if err != nil {
		t.handleError(err)
		return
	}
	defer resp.Body.Close()

	// get the body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.handleError(err)
		return
	}

	for _, scrapeMetric := range t.ScrapeMetrics {
		bodyReader := bytes.NewReader(body)
		value, err := getRowFromBody(bodyReader, scrapeMetric.Name)
		if err != nil {
			t.handleError(err)
			return
		}

		scrapeMetric.ScrapeCounter++
		scrapeMetric.ScrapeSum += value
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
