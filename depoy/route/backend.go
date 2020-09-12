package route

import (
	"depoy/conditional"
	"depoy/metrics"
	"fmt"
	"net/url"
	"sync"

	"github.com/google/uuid"

	log "github.com/sirupsen/logrus"
)

type Backend struct {
	ID               uuid.UUID                `json:"id" yaml:"id" validate:"empty=false"`
	Name             string                   `json:"name" yaml:"name" validate:"empty=false"`
	Addr             string                   `json:"addr" yaml:"addr" validate:"empty=true | format=url"`
	Weigth           uint8                    `json:"weight" yaml:"weight"`
	Active           bool                     `json:"active" yaml:"active"`
	Scrapeurl        string                   `json:"scrape_url" yaml:"scrape_url" validate:"empty=true | format=url"`
	Scrapemetrics    []string                 `json:"scrape_metrics" yaml:"scrape_metrics"`
	Metricthresholds []*conditional.Condition `json:"metric_thresholds" yaml:"metric_thresholds"`
	Healthcheckurl   string                   `json:"healthcheck_url" yaml:"healthcheck_url" validate:"empty=true | format=url"`
	AlertChan        <-chan metrics.Alert     `json:"-" yaml:"-"`
	updateWeigth     func()                   `json:"-" yaml:"-"`
	mux              sync.Mutex               `json:"-" yaml:"-"`
	ActiveAlerts     map[string]metrics.Alert `json:"active_alerts" yaml:"-"`
	killChan         chan int                 `json:"-" yaml:"-"`
}

// NewBackend returns a new base Target
// it has the minimum required configs and misses configs for Scraping
func NewBackend(
	name, addr, scrapeURL, healthCheckPath string,
	scrapeMetrics []string,
	metricThresholds []*conditional.Condition,
	weight uint8) *Backend {

	if name == "" {
		panic("name cannot be null")
	}

	if healthCheckPath == "" {
		healthCheckPath = addr + "/"
	}

	if weight > 100 {
		panic(fmt.Errorf("Weight cannot be larger than 100(%)"))
	}

	url, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}

	backend := &Backend{
		ID:               uuid.New(),
		Name:             name,
		Addr:             url.String(),
		Weigth:           weight,
		Active:           true,
		Scrapeurl:        scrapeURL,
		Scrapemetrics:    scrapeMetrics,    // can be nil
		Metricthresholds: metricThresholds, // can be nil
		Healthcheckurl:   healthCheckPath,
		ActiveAlerts:     make(map[string]metrics.Alert),
		killChan:         make(chan int, 1),
	}

	return backend
}

func (b *Backend) UpdateWeight(weight uint8) {
	b.mux.Lock()
	defer b.mux.Unlock()

	log.Warnf("Updating Weight of Backend %v from %d to %d", b.ID, b.Weigth, weight)
	b.Weigth = weight
}

func (b *Backend) UpdateStatus(status bool) {
	b.mux.Lock()
	defer b.mux.Unlock()

	if b.Active == status {
		return
	}

	b.Active = status
	b.updateWeigth()

	if status {
		log.Warnf("Enabling backend %v: %v", b.ID, b.Active)
	} else {
		log.Warnf("Disabling backend %v: %v", b.ID, b.Active)
	}

}

func (b *Backend) Monitor() {

	if b.AlertChan == nil {
		log.Warnf("Backend %v has no AlertChan set", b.ID)
		return
	}

	log.Debugf("Listening for alert on %v", b)
	for {
		select {
		case _ = <-b.killChan:
			return
		case alert := <-b.AlertChan:
			log.Warnf("Backend %v received %v", b.ID, alert)
			if alert.Type == "Alarming" {

				b.ActiveAlerts[alert.Metric] = alert
				b.UpdateStatus(false)
				continue
			}

			// there can only be one active alert per metric
			delete(b.ActiveAlerts, alert.Metric)

			// if no alert is currently active, set active to true
			if len(b.ActiveAlerts) == 0 {
				b.UpdateStatus(true)
			}
		}

	}
}

func (b *Backend) Stop() {
	b.killChan <- 1
	log.Debugf("Killed Backend %v", b.ID)
}
