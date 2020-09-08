package route

import (
	"depoy/metrics"
	"fmt"
	"net/url"
	"sync"

	"github.com/google/uuid"

	log "github.com/sirupsen/logrus"
)

type Backend struct {
	ID               uuid.UUID            `json:"id"`
	Name             string               `json:"name"`
	Addr             string               `json:"addr"`
	Weigth           uint8                `json:"weight"`
	Active           bool                 `json:"active"` // in % (100 max)
	ScrapeURL        string               `json:"scrape_url"`
	ScrapeMetrics    []string             `json:"scrape_metrics"`
	MetricThresholds map[string]float64   `json:"metrics_tresholds"`
	HealthCheckURL   string               `json:"healthcheck_url"`
	AlertChan        <-chan metrics.Alert `json:"-"`
	updateWeigth     func()               `json:"-"`
	mux              sync.Mutex           `json:"-"`
}

// NewBackend returns a new base Target
// it has the minimum required configs and misses configs for Scraping
func NewBackend(
	name, addr, scrapeURL, healthCheckPath string,
	scrapeMetrics []string,
	metricThresholds map[string]float64,
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
		ScrapeURL:        scrapeURL,
		ScrapeMetrics:    scrapeMetrics,    // can be nil
		MetricThresholds: metricThresholds, // can be nil
		HealthCheckURL:   healthCheckPath,
	}

	return backend
}

func (b *Backend) UpdateWeight(weight uint8) {
	b.mux.Lock()
	defer b.mux.Unlock()

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
		log.Warn("Backend %v has no AlertChan set", b.ID)
		return
	}
	log.Debugf("Listening for alert on %v", b)
	for {

		alert := <-b.AlertChan
		log.Warnf("Backend %v received %v", b.ID, alert)
		if alert.Type == "Alarming" {
			b.UpdateStatus(false)
			continue
		}
		b.UpdateStatus(true)
	}
}
