package route

import (
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/rgumi/depoy/upstreamclient"

	"github.com/rgumi/depoy/conditional"
	"github.com/rgumi/depoy/metrics"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Route struct {
	Name                string
	Prefix              string
	Methods             []string
	Host                string
	Rewrite             string
	CookieTTL           time.Duration
	Strategy            *Strategy
	HealthCheck         bool
	HealthCheckInterval time.Duration
	MonitoringInterval  time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	ScrapeInterval      time.Duration
	Proxy               string
	cookieName          string
	Backends            map[uuid.UUID]*Backend
	Switchover          *Switchover
	Client              *upstreamclient.Upstreamclient
	MetricsRepo         *metrics.Repository
	NextTargetDistr     []*Backend
	lenNextTargetDistr  int
	killHealthCheck     chan int
	mux                 sync.RWMutex
}

// New creates a new route-object with the provided config
func New(
	name, prefix, rewrite, host, proxy string,
	methods []string,
	readTimeout, writeTimeout, idleTimeout, scrapeInterval, healthcheckInterval,
	monitoringInterval, cookieTTL time.Duration,
	doHealthCheck bool,
) (*Route, error) {

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
		ReadTimeout:         readTimeout,
		WriteTimeout:        writeTimeout,
		IdleTimeout:         idleTimeout,
		ScrapeInterval:      scrapeInterval,
		HealthCheck:         doHealthCheck,
		HealthCheckInterval: healthcheckInterval,
		MonitoringInterval:  monitoringInterval,
		cookieName:          strings.ToUpper(name) + "_SESSIONCOOKIE",
		Strategy:            nil,
		Backends:            make(map[uuid.UUID]*Backend),
		killHealthCheck:     make(chan int, 1),
		CookieTTL:           cookieTTL,
		Client: upstreamclient.NewUpstreamclient(readTimeout, writeTimeout, idleTimeout,
			upstreamclient.MaxIdleConnsPerHost, upstreamclient.SkipTLSVerify,
		),
	}

	if route.HealthCheck {
		go route.RunHealthCheckOnBackends()
	}
	return route, nil
}

func (r *Route) SetStrategy(strategy *Strategy) {
	r.Strategy = strategy
}

func (r *Route) GetHandler() fasthttp.RequestHandler {
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
					"6xxRate", ">", 0, 5*time.Second, 2*time.Second)
				mustHaveCondition.Compile()
				backend.Metricthresholds = append(backend.Metricthresholds, mustHaveCondition)
			}

			log.Debugf("Registering %v of %s to MetricsRepository", backend.ID, r.Name)
			backend.AlertChan, _ = r.MetricsRepo.RegisterBackend(
				r.Name, backend.ID, backend.Scrapeurl, backend.Scrapemetrics,
				r.ScrapeInterval, backend.Metricthresholds,
			)

			// start monitoring the registered backend
			log.Debugf("Starting monitoring goroutine of %v of %s", backend.ID, r.Name)
			go r.MetricsRepo.Monitor(backend.ID, r.MonitoringInterval)
			// starts listening on alertChan
			log.Debugf("Starting listening goroutine of %v of %s", backend.ID, r.Name)
			go backend.Monitor()
		}

		log.Debugf("Starting initial healthcheck of %v of %s", backend.ID, r.Name)
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

// AddBackend adds another backend instance of the route
// A backend could be another version of the upstream application
// which then can be routed to
func (r *Route) AddBackend(
	name string, addr, scrapeURL, healthCheckURL *url.URL,
	scrapeMetrics []string,
	metricsThresholds []*conditional.Condition,
	weight uint8) (uuid.UUID, error) {

	backend, err := NewBackend(
		name, addr, scrapeURL, healthCheckURL, scrapeMetrics, metricsThresholds, weight)
	if err != nil {
		return uuid.UUID{}, err
	}
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

	newBackend, err := NewBackend(
		backend.Name, backend.Addr, backend.Scrapeurl, backend.Healthcheckurl, backend.Scrapemetrics,
		backend.Metricthresholds, backend.Weigth,
	)
	if err != nil {
		return uuid.UUID{}, err
	}

	if backend.ID != uuid.Nil {
		newBackend.ID = backend.ID
	} else {
		log.Infof("Registered backend (ID: %v) does not have a valid ID. Creating new one.", newBackend.ID)
	}

	for _, existingBackend := range r.Backends {
		if existingBackend.Name == newBackend.Name {
			return uuid.UUID{}, fmt.Errorf("Backend with given name already exists")
		}
	}

	// status will be set by first healthcheck
	if r.HealthCheck {
		newBackend.Active = false
	} else {
		newBackend.Active = true
	}

	newBackend.updateWeigth = r.updateWeights
	newBackend.ActiveAlerts = make(map[string]metrics.Alert)
	newBackend.killChan = make(chan int, 1)

	log.Warnf("Added Backend %v to Route %s", newBackend.ID, r.Name)
	r.Backends[newBackend.ID] = newBackend
	return newBackend.ID, nil
}

func (r *Route) Delete() {
	r.killHealthCheck <- 1
	r.RemoveSwitchOver()
	for backendID := range r.Backends {
		r.RemoveBackend(backendID)
	}
}
func (r *Route) RemoveBackend(backendID uuid.UUID) error {
	log.Warnf("Removing %s from %s", backendID, r.Name)

	if r.Switchover != nil {
		if r.Switchover.From.ID == backendID || r.Switchover.To.ID == backendID {
			return fmt.Errorf("Cannot deleted backend %v with switchover %d associated with it",
				backendID, r.Switchover.ID,
			)
		}
	}
	if r.MetricsRepo != nil {
		r.MetricsRepo.RemoveBackend(backendID)
	}
	r.Backends[backendID].Stop()
	delete(r.Backends, backendID)
	return nil
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
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(backend.Healthcheckurl.String())
	req.Header.SetMethod("GET")
	m := metrics.MetricsPool.Get().(*metrics.Metrics)
	m.BackendID = backend.ID
	m.Route = r.Name
	m.RequestMethod = string(req.Header.Method())
	m.DownstreamAddr = "depoy-healthcheck"
	resp, err := r.Client.Send(req, m)
	fasthttp.ReleaseRequest(req)
	if err != nil {
		log.Debugf("Healthcheck for %v failed due to %v", backend.ID, err)
		if backend.Active {
			backend.UpdateStatus(false)
		}
		m.ResponseStatus = 600
		m.ContentLength = 0
		r.MetricsRepo.InChannel <- m
		return false
	}
	m.ResponseStatus = resp.Header.StatusCode()
	m.ContentLength = int64(resp.Header.ContentLength())
	r.MetricsRepo.InChannel <- m
	fasthttp.ReleaseResponse(resp)
	return true
}

func (r *Route) RunHealthCheckOnBackends() {
	for {
		select {
		case _ = <-r.killHealthCheck:
			log.Warnf("Stopping healthcheck-loop of %s", r.Name)
			return
		case _ = <-time.After(r.HealthCheckInterval):
			if r.MetricsRepo == nil || r.Client == nil {
				continue
			}
			for _, backend := range r.Backends {
				go r.healthCheck(backend)
			}
		}
	}

}

// StartSwitchOver starts the switch over process
func (r *Route) StartSwitchOver(
	from, to string,
	conditions []*conditional.Condition,
	timeout time.Duration, allowedFailures int,
	weightChange uint8, force, rollback bool) (*Switchover, error) {

	var fromBackend, toBackend *Backend

	// check if a switchover is already active
	// only one switchover is allowed per route at a time
	if r.Switchover != nil {
		if r.Switchover.Status == "Running" {
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
		// Overwrite the current Strategy with CanaryStrategy
		strategy, err := NewCanaryStrategy(r)
		if err != nil {
			return nil, err
		}
		r.SetStrategy(strategy)

		// set initial weights
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

	switchover, err := NewSwitchover(
		fromBackend, toBackend, r, conditions, timeout, allowedFailures, weightChange, rollback)

	if err != nil {
		return nil, err
	}

	r.Switchover = switchover
	go switchover.Start()

	return switchover, nil
}

// RemoveSwitchOver stops the switchover process and leaves the weights as they are last
func (r *Route) RemoveSwitchOver() {
	if r.Switchover != nil {
		log.Warnf("Stopping Switchover of %s", r.Name)
		r.Switchover.Stop()
		r.Switchover = nil
	}
}

// HTTPDo accepts a request, target and the return-function
// it sends the request to the target and
// the response of the target is then handed to the return-function
func (r *Route) HTTPDo(
	req *fasthttp.Request,
	target *Backend,
	returnResp func(*fasthttp.Response)) error {

	m := metrics.AcquireMetrics()
	m.Route = r.Name
	m.BackendID = target.ID
	m.RequestMethod = string(req.Header.Method())
	m.DSContentLength = int64(req.Header.ContentLength())

	uri := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(uri)
	req.URI().CopyTo(uri)
	r.formateURI(uri, target)
	req.SetRequestURI(uri.String())
	resp, err := r.Client.Send(req, m)
	if err != nil {
		m.ResponseStatus = 600
		m.ContentLength = -1
		r.MetricsRepo.InChannel <- m
		return err
	}
	defer fasthttp.ReleaseResponse(resp)
	returnResp(resp)
	m.ResponseStatus = resp.StatusCode()
	m.ContentLength = int64(resp.Header.ContentLength())
	r.MetricsRepo.InChannel <- m
	return nil
}

// HTTPReturn takes a ctx and returns a functions that accepts an upstream response
// which is then copied to the ctx response
func HTTPReturn(
	ctx *fasthttp.RequestCtx,
	c *fasthttp.Cookie) func(resp *fasthttp.Response) {

	return func(resp *fasthttp.Response) {
		resp.Header.CopyTo(&ctx.Response.Header)
		if c != nil {
			ctx.Response.Header.SetCookie(c)
		}
		ctx.SetStatusCode(resp.StatusCode())
		delResponseHopHeader(resp)
		ctx.Response.SetBody(resp.Body())
	}
}

func (r *Route) formateURI(uri *fasthttp.URI, backend *Backend) {
	uri.SetScheme(backend.Addr.Scheme)
	uri.SetHost(backend.Addr.Host)
	if r.Rewrite != "" {
		uri.SetPath(strings.Replace(string(uri.Path()), r.Prefix, r.Rewrite, 1))
	}
}

func handleNetError(err error) (string, int) {
	netErr, ok := err.(net.Error)
	if !ok {
		return err.Error(), 500
	}
	if netErr.Timeout() {
		return netErr.Error(), 504
	}
	return netErr.Error(), 502
}
