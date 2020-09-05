package route

import (
	"depoy/metrics"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var healthCheckInterval = 5 * time.Second

type UpstreamClient interface {
	Send(*http.Request) (*http.Response, metrics.Metrics, error)
}

type Route struct {
	Name            string                 `json:"name"`
	Prefix          string                 `json:"prefix"`
	Methods         []string               `json:"methods"`
	Host            string                 `json:"host"`
	Rewrite         string                 `json:"rewrite"`
	Backends        map[uuid.UUID]*Backend `json:"backends"`
	Client          UpstreamClient         `json:"-"`
	Handler         http.HandlerFunc       `json:"-"`
	NextTargetDistr []*Backend             `json:"-"`
	CookieTTL       time.Duration          `json:"cookie_ttl"`
	MetricsRepo     *metrics.Repository    `json:"-"`
	SwitchOver      *SwitchOver            `json:"switch"`
}

func New(name, prefix, rewrite, host string, methods []string, upstreamClient UpstreamClient) (*Route, error) {

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
	route.Handler = route.StickyHandler()
	route.Backends = make(map[uuid.UUID]*Backend)

	route.CookieTTL = 120 * time.Second

	go route.RunHealthCheckOnBackends()

	return route, nil
}

func (r *Route) GetHandler() http.HandlerFunc {
	return r.Handler
}

func (r *Route) updateWeights() {

	k := 0
	listWeights := make([]uint8, len(r.Backends))

	i := 0
	for _, backend := range r.Backends {
		if backend.Active {
			listWeights[i] = backend.Weigth
			i++
		}
	}

	// find ggt to reduce list length
	ggt := GGT(listWeights)
	log.Debugf("Current GGT of Weights is %d", ggt)

	var sum uint8
	if ggt > 0 {
		for _, weight := range listWeights {
			sum += weight / ggt
		}
	}

	var distr = make([]*Backend, sum)

	for _, backend := range r.Backends {
		if backend.Active {
			for i := uint8(0); i < backend.Weigth/ggt; i++ {
				distr[k] = backend
				k++
			}
		}
	}
	log.Warnf("Current TargetDistribution: %v", distr)

	r.NextTargetDistr = distr
}

func (r *Route) getNextBackend() (*Backend, error) {

	numberOfBackends := len(r.NextTargetDistr)

	if numberOfBackends == 0 {
		return nil, fmt.Errorf("No backend is active")
	}

	n := rand.Intn(numberOfBackends)
	backend := r.NextTargetDistr[n]

	/*
		if !backend.Active {
			r.updateWeights()
			backend = r.NextTargetDistr[n]
		}
	*/
	return backend, nil
}

func (r *Route) sendRequestToUpstream(target *Backend, req *http.Request, body []byte) (*http.Response, error) {
	url := req.URL.String()

	if r.Rewrite != "" {
		url = strings.Replace(url, r.Prefix, r.Rewrite, -1)
	}
	// Copies the downstream request into the upstream request
	// and formats it accordingly
	log.Debug(target.Addr + url)
	newReq, _ := formateRequest(req, target.Addr+url, body)
	/*
		if err != nil {
			return nil, err
		}
	*/

	// Send Request to the upstream host
	// reuses TCP-Connection
	resp, m, err := r.Client.Send(newReq)
	if err != nil {

		target.UpdateStatus(false)
		m.Route = r.Name
		m.RequestMethod = req.Method
		m.DownstreamAddr = req.RemoteAddr
		m.UpstreamAddr = target.Addr
		m.ResponseStatus = 600
		m.ContentLength = resp.ContentLength
		m.BackendID = target.ID
		r.MetricsRepo.InChannel <- m

		return nil, err
	}

	m.Route = r.Name
	m.RequestMethod = req.Method
	m.DownstreamAddr = req.RemoteAddr
	m.UpstreamAddr = target.Addr
	m.ResponseStatus = resp.StatusCode
	m.ContentLength = resp.ContentLength
	m.BackendID = target.ID
	r.MetricsRepo.InChannel <- m

	return resp, nil
}

// AddBackend adds another backend instance of the route
// A backend could be another version of the upstream application
// which then can be routed to
func (r *Route) AddBackend(name, addr, scrapeURL string, scrapeMetrics map[string]float64, weight uint8) uuid.UUID {

	backend := NewBackend(name, addr, scrapeURL, "", scrapeMetrics, weight)
	backend.updateWeigth = r.updateWeights
	log.Debugf("Added Backend %v to Router %s", backend, r.Name)
	r.Backends[backend.ID] = backend
	r.updateWeights()

	return backend.ID
}

func (r *Route) UpdateBackendWeight(id uuid.UUID, newWeigth uint8) error {
	if backend, found := r.Backends[id]; found {
		backend.Weigth = newWeigth
		r.updateWeights()
		return nil
	}
	return fmt.Errorf("Backend with ID %v does not exist", id)
}

// TODO: make this an alert in metrics.go !!!!
// otherwise status will never be updated lol

func (r *Route) RunHealthCheckOnBackends() {
	for {
		<-time.After(healthCheckInterval)

		for _, backend := range r.Backends {
			req, err := http.NewRequest("GET", backend.HealthCheckURL, nil)
			if err != nil {
				log.Error(err.Error())
				continue
			}

			resp, m, err := r.Client.Send(req)
			if err != nil {

				log.Warnf("Healthcheck for %v failed due to %s", backend.ID, err.Error())
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
				continue
			}

			m.Route = r.Name
			m.RequestMethod = req.Method
			m.DownstreamAddr = req.RemoteAddr
			m.ResponseStatus = resp.StatusCode
			m.ContentLength = resp.ContentLength
			m.BackendID = backend.ID

			r.MetricsRepo.InChannel <- m
		}
	}
}

// StartSwitchOver starts the switch over process
func (r *Route) StartSwitchOver(
	old, new *Backend,
	conditions []Condition,
	wait time.Duration,
	weightChange int) {

	switchOver := &SwitchOver{
		From:         old,
		To:           new,
		Conditions:   conditions,
		Route:        r,
		Wait:         wait,
		WeightChange: weightChange,
	}
	r.SwitchOver = switchOver
	go switchOver.Start()
}

// StopSwitchOver stops the switchover process and leaves the weights as they are last
func (r *Route) StopSwitchOver() {
	r.SwitchOver.Stop()
}
