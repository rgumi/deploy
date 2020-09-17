package route

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rgumi/depoy/conditional"
	"github.com/rgumi/depoy/metrics"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var (
	healthCheckInterval = 5 * time.Second
	cookieTTL           = 120 * time.Second
	monitorTimeout      = 5 * time.Second
	activeFor           = 10 * time.Second
)

// UpstreamClient is an interface for the http.Client
// it implements the method Send which is used to send http requests
type UpstreamClient interface {
	Send(*http.Request) (*http.Response, metrics.Metrics, error)
}

type Route struct {
	Name            string                 `json:"name" yaml:"name" validate:"empty=false"`
	Prefix          string                 `json:"prefix" yaml:"prefix" validate:"empty=false"`
	Methods         []string               `json:"methods" yaml:"methods" validate:"empty=false"`
	Host            string                 `json:"host" yaml:"host" default:"*"`
	Rewrite         string                 `json:"rewrite" yaml:"rewrite" validate:"empty=false"`
	CookieTTL       time.Duration          `json:"cookie_ttl" yaml:"cookieTTL"  default:"2m0s"`
	Strategy        *Strategy              `json:"strategy" yaml:"strategy"`
	Backends        map[uuid.UUID]*Backend `json:"backends" yaml:"backends"`
	Client          UpstreamClient         `json:"-" yaml:"-"`
	NextTargetDistr []*Backend             `json:"-" yaml:"-"`
	MetricsRepo     *metrics.Repository    `json:"-" yaml:"-"`
	SwitchOver      *SwitchOver            `json:"switchover" yaml:"-"`
	killHealthCheck chan int               `json:"-" yaml:"-"`
	mux             sync.RWMutex           `json:"-" yaml:"-"`
}

func New(
	name, prefix, rewrite, host string,
	methods []string,
	upstreamClient UpstreamClient,
) (*Route, error) {

	route := new(Route)
	client := upstreamClient
	route.Client = client

	// fix prefix if prefix does not end with /
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	route.Name = name
	route.Prefix = prefix
	route.Rewrite = rewrite
	route.Methods = methods
	route.Host = host
	route.Strategy = nil
	route.Backends = make(map[uuid.UUID]*Backend)
	route.killHealthCheck = make(chan int, 1)

	route.CookieTTL = cookieTTL

	go route.RunHealthCheckOnBackends()

	return route, nil
}

func (r *Route) SetStrategy(strategy *Strategy) {
	r.Strategy = strategy
}

func (r *Route) GetHandler() http.HandlerFunc {
	return r.Strategy.Handler
}

func (r *Route) updateWeights() {

	var sum uint8
	k := 0
	i := 0

	// avoid any race conditions when multiple backends are used
	r.mux.Lock()
	defer r.mux.Unlock()

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
}

func (r *Route) getNextBackend() (*Backend, error) {

	numberOfBackends := len(r.NextTargetDistr)

	if numberOfBackends == 0 {
		return nil, fmt.Errorf("No backend is active")
	}

	n := rand.Intn(numberOfBackends)
	backend := r.NextTargetDistr[n]

	return backend, nil
}

// Reload is required if the route is changed (reload config).
// when a new backend is registerd reload handles the initial tasks
// like monitoring and healthcheck
func (r *Route) Reload() {
	var err error

	log.Warnf("Reloading %v", r.Name)

	if r.MetricsRepo == nil {
		panic(fmt.Errorf("MetricsRepo of %s cannot be nil", r.Name))
	}

	for _, backend := range r.Backends {

		// Check if backend is eligible for monitoring: (1) has metrics defined, (2) is not yet registered
		if (backend.Metricthresholds != nil ||
			(backend.Scrapemetrics != nil && backend.Scrapeurl != "")) && backend.AlertChan == nil {

			log.Warnf("Registering %v of %s to MetricsRepository", backend.ID, r.Name)
			backend.AlertChan, err = r.MetricsRepo.RegisterBackend(
				r.Name, backend.ID, backend.Scrapeurl, backend.Scrapemetrics, backend.Metricthresholds)

			if err != nil {
				panic(err)
			}

			// start monitoring the registered backend
			go r.MetricsRepo.Monitor(backend.ID, monitorTimeout, activeFor)

			// starts listening on alertChan
			go backend.Monitor()
		}

		// if this fails an alert will be registered
		// when the backend is ready again, the alarm
		// will eventually be resolved and the backend
		// is set to active
		go r.validateStatus(backend)
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
	r.MetricsRepo.RegisterAlert(backend.ID, "Pending", "6xxRate", 0, 1)
}
func (r *Route) sendRequestToUpstream(
	target *Backend,
	req *http.Request,
	body []byte) (*http.Response, metrics.Metrics, error) {

	url := req.URL.String()

	if r.Rewrite != "" {
		url = strings.Replace(url, r.Prefix, r.Rewrite, -1)
	}

	// Copies the downstream request into the upstream request
	// and formats it accordingly
	log.Debug(target.Addr + url)
	newReq, _ := formateRequest(req, target.Addr+url, body)

	// Send Request to the upstream host
	// reuses TCP-Connection
	resp, m, err := r.Client.Send(newReq)
	if err != nil {

		target.UpdateStatus(false)
		m.Route = r.Name
		m.RequestMethod = req.Method
		m.DownstreamAddr = req.RemoteAddr
		m.ResponseStatus = 600
		m.ContentLength = -1
		m.BackendID = target.ID

		return nil, m, err
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
	backend.Active = false

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

	if backend.Addr == "" || backend.Name == "" || backend.ID.String() == "" {
		return uuid.UUID{}, fmt.Errorf("Required Parameters of backend are missing")
	}

	for _, existingBackend := range r.Backends {
		if existingBackend.Name == backend.Name {
			return uuid.UUID{}, fmt.Errorf("Backend with given name already exists")
		}
	}

	backend.updateWeigth = r.updateWeights
	backend.Active = false
	backend.ActiveAlerts = make(map[string]metrics.Alert)
	backend.killChan = make(chan int, 1)

	// compile conditions to prevent nil-pointers
	for _, cond := range backend.Metricthresholds {
		cond.IsTrue = cond.Compile()
	}

	if backend.Healthcheckurl == "" {
		backend.Healthcheckurl = backend.Addr + "/"
	}

	log.Warnf("Added Backend %v to Route %s", backend.ID, r.Name)
	r.Backends[backend.ID] = backend

	return backend.ID, nil
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

func (r *Route) StopAll() {
	r.killHealthCheck <- 1
	r.RemoveSwitchOver()

	for backendID := range r.Backends {
		r.RemoveBackend(backendID)
	}

}

func (r *Route) healthCheck(backend *Backend) bool {
	req, err := http.NewRequest("GET", backend.Healthcheckurl, nil)
	if err != nil {
		log.Error(err.Error())
		return false
	}

	resp, m, err := r.Client.Send(req)
	if err != nil {

		log.Infof("Healthcheck for %v failed due to %v", backend.ID, err)
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
		time.Sleep(healthCheckInterval)
	}

}

// StartSwitchOver starts the switch over process
func (r *Route) StartSwitchOver(
	from, to string,
	conditions []*conditional.Condition,
	timeout time.Duration,
	weightChange uint8, force bool) (*SwitchOver, error) {

	var fromBackend, toBackend *Backend

	// check if a switchover is already active
	// only one switchover is allowed per route at a time
	if r.SwitchOver != nil {
		return nil, fmt.Errorf("Only one switchover can be active per route")
	}

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
		fromBackend.Weigth = 100
		toBackend.Weigth = 0

		r.updateWeights()

	} else {
		// The Strategy must be canary (sticky or slippery) because otherwise
		// the traffic cannot be increased/switched-over
		if strings.ToLower(r.Strategy.Type) != "sticky" || strings.ToLower(r.Strategy.Type) != "slippery" {
			return nil, fmt.Errorf(
				"Switchover is only supported with Strategy \"sticky\" or \"slippery\"")
		}
	}

	switchOver, err := NewSwitchOver(
		fromBackend, toBackend, r, conditions, timeout, weightChange)

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
