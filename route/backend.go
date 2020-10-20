package route

import (
	"fmt"
	"net/url"
	"sync"

	"gopkg.in/dealancer/validate.v2"

	"github.com/rgumi/depoy/conditional"
	"github.com/rgumi/depoy/metrics"
	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

type Backend struct {
	ID               uuid.UUID                `json:"id" yaml:"id" validate:"empty=false"`
	Name             string                   `json:"name" yaml:"name" validate:"empty=false"`
	Addr             *url.URL                 `json:"addr" yaml:"addr"`
	Weigth           uint8                    `json:"weight" yaml:"weight"`
	Active           bool                     `json:"active" yaml:"active"`
	Scrapeurl        *url.URL                 `json:"scrape_url" yaml:"scrapeUrl"`
	Scrapemetrics    []string                 `json:"scrape_metrics" yaml:"scrapeMetrics"`
	Metricthresholds []*conditional.Condition `json:"metric_thresholds" yaml:"metricThresholds"`
	Healthcheckurl   *url.URL                 `json:"healthcheck_url" yaml:"healthcheckUrl"`
	ActiveAlerts     map[string]metrics.Alert `json:"active_alerts" yaml:"-"`
	AlertChan        <-chan metrics.Alert     `json:"-" yaml:"-"`
	updateWeigth     func()
	mux              sync.Mutex
	killChan         chan int
}

// NewBackend returns a new base Target
// it has the minimum required configs and misses configs for Scraping
func NewBackend(
	name string, addr, scrapeURL, healthCheckAddr *url.URL,
	scrapeMetrics []string,
	metricThresholds []*conditional.Condition,
	weight uint8) (*Backend, error) {

	id := uuid.New()
	if name == "" {
		name = id.String()
	}

	if healthCheckAddr.Host == "" {
		healthCheckAddr.Scheme = addr.Scheme
		healthCheckAddr.Host = addr.Host
		healthCheckAddr.Path = "/"
	}

	if weight > 100 {
		return nil, fmt.Errorf("Weight cannot be larger than 100")
	}

	backend := &Backend{
		ID:               id,
		Name:             name,
		Addr:             addr,
		Weigth:           weight,
		Active:           true,
		Scrapeurl:        scrapeURL,
		Scrapemetrics:    scrapeMetrics,    // can be nil
		Metricthresholds: metricThresholds, // can be nil
		Healthcheckurl:   healthCheckAddr,
		ActiveAlerts:     make(map[string]metrics.Alert),
		killChan:         make(chan int, 1),
	}

	if err := validate.Validate(backend); err != nil {
		return nil, err
	}

	// compile conditions to prevent nil-pointers
	for _, cond := range backend.Metricthresholds {
		cond.Compile()
	}

	return backend, nil
}

func (b *Backend) UpdateWeight(weight uint8) {
	b.mux.Lock()
	defer b.mux.Unlock()

	log.Debugf("Updating Weight of Backend %v from %d to %d", b.ID, b.Weigth, weight)
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
		log.Infof("Enabling backend %v: %v", b.ID, b.Active)
	} else {
		log.Infof("Disabling backend %v: %v", b.ID, b.Active)
	}
}

func (b *Backend) Monitor() {
	if b.AlertChan == nil {
		panic(fmt.Errorf("Backend %v has no AlertChan set", b.ID))
	}
	log.Debugf("Listening for alert on Backend %v", b.ID)
	for {
		select {
		case _ = <-b.killChan:
			return
		case alert := <-b.AlertChan:
			log.Debugf("Backend %v received %v", b.ID, alert.Type)
			if alert.Type == "Alarming" {
				// Alarm condition was active for long enought => alarming
				b.ActiveAlerts[alert.Metric] = alert
				b.UpdateStatus(false)
			} else if alert.Type == "Pending" {
				// Alarm condition was reached initially
				b.ActiveAlerts[alert.Metric] = alert
			} else {
				// alert.Type == "Resolved"
				delete(b.ActiveAlerts, alert.Metric)
				// if no alert is currently active, set active to true
				if len(b.ActiveAlerts) == 0 {
					b.UpdateStatus(true)
				}
			}
		}
	}
}

func (b *Backend) Stop() {
	b.killChan <- 1
	log.Debugf("Killed Backend %v", b.ID)
}
