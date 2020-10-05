package route

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rgumi/depoy/conditional"
	"github.com/rgumi/depoy/metrics"
	"github.com/rgumi/depoy/upstreamclient"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var (
	ServerName   = "depoy/0.1.0"
	maxIdleConns = 100
	tlsVerify    = false
)

// UpstreamClient is an interface for the http.Client
// it implements the method Send which is used to send http requests
type UpstreamClient interface {
	Send(*http.Request) (*http.Response, metrics.Metrics, error)
}

type Route struct {
	Name                string                 `json:"name" yaml:"name" validate:"empty=false"`
	Prefix              string                 `json:"prefix" yaml:"prefix" validate:"empty=false"`
	Methods             []string               `json:"methods" yaml:"methods" validate:"empty=false"`
	Host                string                 `json:"host" yaml:"host" default:"*"`
	Rewrite             string                 `json:"rewrite" yaml:"rewrite" validate:"empty=false"`
	CookieTTL           time.Duration          `json:"cookie_ttl" yaml:"cookieTTL"`
	Strategy            *Strategy              `json:"strategy" yaml:"strategy" validate:"nil=false"`
	HealthCheck         bool                   `json:"healthcheck_bool" yaml:"healthcheckBool" default:"true"`
	HealthCheckInterval time.Duration          `json:"healthcheck_interval" yaml:"healthcheckInterval"`
	MonitoringInterval  time.Duration          `json:"monitoring_interval" yaml:"monitoringInterval"`
	Timeout             time.Duration          `json:"timeout" yaml:"timeout"`
	IdleTimeout         time.Duration          `json:"idle_timeout" yaml:"idleTimeout"`
	ScrapeInterval      time.Duration          `json:"scrape_interval" yaml:"scrapeInterval"`
	Proxy               string                 `json:"proxy" yaml:"proxy" default:""`
	Backends            map[uuid.UUID]*Backend `json:"backends" yaml:"backends"`
	SwitchOver          *SwitchOver            `json:"switchover" yaml:"-"`
	Client              UpstreamClient         `json:"-" yaml:"-"`
	MetricsRepo         *metrics.Repository    `json:"-" yaml:"-"`
	NextTargetDistr     []*Backend             `json:"-" yaml:"-"`
	lenNextTargetDistr  int
	killHealthCheck     chan int
	mux                 sync.RWMutex
}

// New creates a new route-object with the provided config
func New(
	name, prefix, rewrite, host, proxy string,
	methods []string,
	timeout, idleTimeout, scrapeInterval, healthcheckInterval,
	monitoringInterval, cookieTTL time.Duration,
	doHealthCheck bool,
) (*Route, error) {

	client := upstreamclient.NewClient(maxIdleConns, timeout, idleTimeout, proxy, tlsVerify)

	// fix prefix if prefix does not end with /
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	route := &Route{
		Name:                name,
		Prefix:              prefix,
		Rewrite:             rewrite,
		Methods:             methods,
		Host:                host,
		Proxy:               proxy,
		Timeout:             timeout,
		IdleTimeout:         idleTimeout,
		ScrapeInterval:      scrapeInterval,
		HealthCheck:         doHealthCheck,
		HealthCheckInterval: healthcheckInterval,
		MonitoringInterval:  monitoringInterval,
		Strategy:            nil,
		Backends:            make(map[uuid.UUID]*Backend),
		killHealthCheck:     make(chan int, 1),
		CookieTTL:           cookieTTL,
		Client:              client,
	}

	if route.HealthCheck {
		go route.RunHealthCheckOnBackends()
	}

	return route, nil
}

func (r *Route) SetStrategy(strategy *Strategy) {
	r.Strategy = strategy
}

func (r *Route) GetHandler() http.HandlerFunc {
	if r.Strategy == nil {
		panic(fmt.Errorf("No strategy is set for %s", r.Name))
	}

	return r.Strategy.Handler
}

func (r *Route) updateWeights() {
	r.mux.Lock()
	defer r.mux.Unlock()

	var sum uint8
	k, i := 0, 0
	listWeights := make([]uint8, len(r.Backends))
	activeBackends := []*Backend{}

	for _, backend := range r.Backends {
		if backend.Active {
			listWeights[i] = backend.Weigth
			activeBackends = append(activeBackends, backend)
			i++
		}
	}
	// find ggt to reduce list length
	ggt := GGT(listWeights) // if 0, return 0
	log.Debugf("Current GGT of Weights is %d", ggt)

	if ggt > 0 {
		for _, weight := range listWeights {
			sum += weight / ggt
		}
		distr := make([]*Backend, sum)

		for _, backend := range activeBackends {
			for i := uint8(0); i < backend.Weigth/ggt; i++ {
				distr[k] = backend
				k++
			}
		}
		log.Debugf("Current TargetDistribution of %s: %v", r.Name, distr)
		r.NextTargetDistr = distr
	} else {
		// no active backend
		r.NextTargetDistr = make([]*Backend, 0)
	}
	r.lenNextTargetDistr = len(r.NextTargetDistr)
}

func (r *Route) getNextBackend() (*Backend, error) {

	if r.lenNextTargetDistr == 0 {
		return nil, fmt.Errorf("No backend is active")
	}

	backend := r.NextTargetDistr[rand.Intn(r.lenNextTargetDistr)]
	return backend, nil
}

// Reload is required if the route is changed (reload config).
// when a new backend is registerd reload handles the initial tasks
// like monitoring and healthcheck
func (r *Route) Reload() {
	log.Infof("Reloading %v", r.Name)
	if !r.HealthCheck {
		log.Warnf("Healthcheck of %s is not active", r.Name)
	}
	if r.MetricsRepo == nil {
		panic(fmt.Errorf("MetricsRepo of %s cannot be nil", r.Name))
	}
	for _, backend := range r.Backends {
		if backend.AlertChan == nil {
			if r.HealthCheck {
				mustHaveCondition := conditional.NewCondition(
					"6xxRate", ">", 0.1, 5*time.Second, 10*time.Second)
				mustHaveCondition.Compile()
				backend.Metricthresholds = append(backend.Metricthresholds, mustHaveCondition)
			}

			log.Infof("Registering %v of %s to MetricsRepository", backend.ID, r.Name)
			backend.AlertChan, _ = r.MetricsRepo.RegisterBackend(
				r.Name, backend.ID, backend.Scrapeurl, backend.Scrapemetrics,
				r.ScrapeInterval, backend.Metricthresholds,
			)
			// start monitoring the registered backend
			go r.MetricsRepo.Monitor(backend.ID, r.MonitoringInterval)
			// starts listening on alertChan
			go backend.Monitor()
		}

		if r.HealthCheck {
			go r.validateStatus(backend)
		} else {
			r.updateWeights()
		}
	}
}

func (r *Route) validateStatus(backend *Backend) {
	log.Debugf("Executing validateStatus on %v", backend.ID)
	if r.healthCheck(backend) {
		log.Debugf("Finished healtcheck of %v successfully", backend.ID)
		backend.UpdateStatus(true)
		return
	}

	// It sometimes is possible that when a new backend is added and while the
	// backend is registered, the upstream application is just starting (Conn refused),
	// the status does not get updated when the upstream application is healthy again as an
	// alarm has not been registered in the MetricsRepo due to an activeFor which can then be
	// resolved
	if r.MetricsRepo != nil {
		r.MetricsRepo.RegisterAlert(backend.ID, "Pending", "6xxRate", 0, 1)
	}

}

func (r *Route) sendRequestToUpstream(
	target *Backend,
	req *http.Request,
	readCloser io.ReadCloser) (*http.Response, metrics.Metrics, GatewayError) {

	url := req.URL.String()

	if r.Rewrite != "" {
		url = strings.Replace(url, r.Prefix, r.Rewrite, -1)
	}

	// Copies the downstream request into the upstream request
	// and formats it accordingly
	log.Debug(target.Addr + url)
	newReq, _ := formateRequest(req, target.Addr+url, readCloser)

	// Send Request to the upstream host
	// reuses TCP-Connection
	resp, m, err := r.Client.Send(newReq)
	if err != nil {
		gErr := NewGatewayError(err)
		//target.UpdateStatus(false)
		m.Route = r.Name
		m.RequestMethod = req.Method
		m.DownstreamAddr = req.RemoteAddr
		m.ResponseStatus = gErr.Code()
		m.ContentLength = -1
		m.BackendID = target.ID
		return nil, m, gErr
	}
	m.Route = r.Name
	m.RequestMethod = req.Method
	m.DownstreamAddr = req.RemoteAddr
	m.ResponseStatus = resp.StatusCode
	m.ContentLength = resp.ContentLength
	m.BackendID = target.ID
	return resp, m, nil
}

// AddBackend adds another backend instance of the route
// A backend could be another version of the upstream application
// which then can be routed to
func (r *Route) AddBackend(
	name, addr, scrapeURL, healthCheckURL string,
	scrapeMetrics []string,
	metricsThresholds []*conditional.Condition,
	weight uint8) (uuid.UUID, error) {

	backend := NewBackend(
		name, addr, scrapeURL, healthCheckURL, scrapeMetrics, metricsThresholds, weight)

	backend.updateWeigth = r.updateWeights

	if r.HealthCheck {
		backend.Active = false
	} else {
		backend.Active = true
	}

	for _, backend := range r.Backends {
		if backend.Name == name {
			return uuid.UUID{}, fmt.Errorf("Backend with given name already exists")
		}
	}

	log.Warnf("Added Backend %v to Route %s", backend.ID, r.Name)
	r.Backends[backend.ID] = backend

	return backend.ID, nil
}

// AddExistingBackend can be used to add an existing backend to a route
func (r *Route) AddExistingBackend(backend *Backend) (uuid.UUID, error) {

	if backend.Addr == "" {
		return uuid.UUID{}, fmt.Errorf("Required Parameters of backend are missing")
	}

	if backend.ID == uuid.Nil {
		backend.ID = uuid.New()
	}
	if backend.Name == "" {
		backend.Name = backend.ID.String()
	}

	for _, existingBackend := range r.Backends {
		if existingBackend.Name == backend.Name {
			return uuid.UUID{}, fmt.Errorf("Backend with given name already exists")
		}
	}

	// status will be set by first healthcheck
	if r.HealthCheck {
		backend.Active = false
	} else {
		backend.Active = true
	}

	backend.updateWeigth = r.updateWeights
	backend.ActiveAlerts = make(map[string]metrics.Alert)
	backend.killChan = make(chan int, 1)

	// compile conditions to prevent nil-pointers
	for _, cond := range backend.Metricthresholds {
		cond.Compile()
	}

	if backend.Healthcheckurl == "" {
		backend.Healthcheckurl = backend.Addr + "/"
	}

	log.Warnf("Added Backend %v to Route %s", backend.ID, r.Name)
	r.Backends[backend.ID] = backend

	return backend.ID, nil
}

func (r *Route) StopAll() {
	r.killHealthCheck <- 1
	r.RemoveSwitchOver()

	for backendID := range r.Backends {
		r.RemoveBackend(backendID)
	}

}
func (r *Route) RemoveBackend(backendID uuid.UUID) {
	log.Warnf("Removing %s from %s", backendID, r.Name)

	if r.MetricsRepo != nil {
		r.MetricsRepo.RemoveBackend(backendID)
	}

	r.Backends[backendID].Stop()
	delete(r.Backends, backendID)
}

func (r *Route) UpdateBackendWeight(id uuid.UUID, newWeigth uint8) error {
	if backend, found := r.Backends[id]; found {
		backend.Weigth = newWeigth
		r.updateWeights()
		return nil
	}
	return fmt.Errorf("Backend with ID %v does not exist", id)
}

func (r *Route) healthCheck(backend *Backend) bool {
	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()
	req, err := http.NewRequest("GET", backend.Healthcheckurl, nil)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	resp, m, err := r.Client.Send(req)
	if err != nil {

		log.Tracef("Healthcheck for %v failed due to %v", backend.ID, err)
		if backend.Active {
			backend.UpdateStatus(false)
		}

		m.Route = r.Name
		m.RequestMethod = req.Method
		m.DownstreamAddr = req.RemoteAddr
		m.ResponseStatus = 600
		m.ContentLength = 0
		m.BackendID = backend.ID
		r.MetricsRepo.InChannel <- m
		return false
	}
	resp.Body.Close()

	m.Route = r.Name
	m.RequestMethod = req.Method
	m.DownstreamAddr = req.RemoteAddr
	m.ResponseStatus = resp.StatusCode
	m.ContentLength = resp.ContentLength
	m.BackendID = backend.ID

	r.MetricsRepo.InChannel <- m
	return true

}

func (r *Route) RunHealthCheckOnBackends() {
	for {
		select {
		case _ = <-r.killHealthCheck:
			log.Warnf("Stopping healthcheck-loop of %s", r.Name)
			return
		default:
			for _, backend := range r.Backends {
				// could be a go-routine
				go r.healthCheck(backend)
			}
		}
		time.Sleep(r.HealthCheckInterval)
	}

}

// StartSwitchOver starts the switch over process
func (r *Route) StartSwitchOver(
	from, to string,
	conditions []*conditional.Condition,
	timeout time.Duration, allowedFailures int,
	weightChange uint8, force, rollback bool) (*SwitchOver, error) {

	var fromBackend, toBackend *Backend

	// check if a switchover is already active
	// only one switchover is allowed per route at a time
	if r.SwitchOver != nil {
		if r.SwitchOver.Status == "Running" {
			return nil, fmt.Errorf("Only one switchover can be active per route")
		}
	}

	if from == "" {
		// select an existing backend
		for _, backend := range r.Backends {
			if backend.Name != to && backend.Weigth == 100 {
				from = backend.Name
				goto forward
			}
		}
		return nil, fmt.Errorf("from was empty and no backend of route could be selected")
	}

forward:
	for _, backend := range r.Backends {
		if backend.Name == from {
			fromBackend = backend
		} else if backend.Name == to {
			toBackend = backend
		}
	}

	if fromBackend == nil {
		return nil, fmt.Errorf("Cannot find backend with Name %v", from)
	}

	if toBackend == nil {
		return nil, fmt.Errorf("Cannot find backend with Name %v", to)
	}

	if force {
		// Overwrite the current Strategy with StickyStrategy
		strategy, err := NewStickyStrategy(r)
		if err != nil {
			return nil, err
		}
		r.SetStrategy(strategy)

		// set weights
		fromBackend.Weigth = 100 - weightChange
		toBackend.Weigth = weightChange

		r.updateWeights()

	} else {
		// The Strategy must be canary (sticky or slippery) because otherwise
		// the traffic cannot be increased/switched-over
		if strings.ToLower(r.Strategy.Type) != "sticky" && strings.ToLower(r.Strategy.Type) != "slippery" {
			return nil, fmt.Errorf(
				"Switchover is only supported with Strategy \"sticky\" or \"slippery\"")
		}
	}

	switchOver, err := NewSwitchOver(
		fromBackend, toBackend, r, conditions, timeout, allowedFailures, weightChange, rollback)

	if err != nil {
		return nil, err
	}

	r.SwitchOver = switchOver
	go switchOver.Start()

	return switchOver, nil
}

// RemoveSwitchOver stops the switchover process and leaves the weights as they are last
func (r *Route) RemoveSwitchOver() {
	if r.SwitchOver != nil {
		log.Warnf("Stopping Switchover of %s", r.Name)
		r.SwitchOver.Stop()
		r.SwitchOver = nil
	}
}
