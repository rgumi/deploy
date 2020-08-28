package route

import (
	"depoy/metrics"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

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
	MetricsChan     chan metrics.Metrics   `json:"-"`
	WeightSum       int                    `json:"-"`
	NextTargetDistr []*Backend             `json:"-"`
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
	route.Handler = route.GetExternalHandle()
	route.Backends = make(map[uuid.UUID]*Backend)

	go route.RunHealthCheckOnBackends()

	return route, nil
}

func (r *Route) GetHandler() http.HandlerFunc {
	return r.Handler
}

func (r *Route) updateWeights() {
	sum, k := 0, 0

	for _, backend := range r.Backends {
		if backend.Active {
			sum += backend.Weigth
		}
	}

	var distr = make([]*Backend, sum)

	for _, backend := range r.Backends {
		if backend.Active {
			sum += backend.Weigth

			for i := 0; i < backend.Weigth; i++ {
				distr[k] = backend
				k++
			}
		}
	}
	log.Debugf("Current TargetDistribution: %v", distr)

	r.WeightSum = sum
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
		m.RequestMethod = req.Method
		m.DownstreamAddr = req.RemoteAddr
		m.UpstreamAddr = target.Addr
		m.ResponseStatus = 600
		m.BackendID = target.ID
		r.MetricsChan <- m

		return nil, err
	}

	m.RequestMethod = req.Method
	m.DownstreamAddr = req.RemoteAddr
	m.UpstreamAddr = target.Addr
	m.ResponseStatus = resp.StatusCode
	m.BackendID = target.ID
	r.MetricsChan <- m

	return resp, nil
}

func (r *Route) GetExternalHandle() func(w http.ResponseWriter, req *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		//var err error
		var b []byte

		currentTarget, _ := r.getNextBackend()
		/*
			if err != nil {
				log.Errorf("Could not get next backend: %v", err)
				http.Error(w, "", 503)

				return
			}
		*/
		b, _ = ioutil.ReadAll(req.Body)
		/*
			if err != nil {
				log.Errorf("Unable to read body of downstream request: %v", err)
				http.Error(w, "", 500)
				return
			}
		*/
		defer req.Body.Close()

		resp, err := r.sendRequestToUpstream(currentTarget, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "", 503)
			return
		}
		sendResponse(resp, w)
	}
}

// AddBackend adds another backend instance of the route
// A backend could be another version of the upstream application
// which then can be routed to
func (r *Route) AddBackend(name, addr, scrapeURL string, scrapeMetrics map[string]float64, weight int) uuid.UUID {

	backend := NewBackend(name, addr, scrapeURL, "", scrapeMetrics, weight)
	backend.updateWeigth = r.updateWeights
	log.Debugf("Added Backend %v to Router %s", backend, r.Name)
	r.Backends[backend.ID] = backend
	r.updateWeights()

	return backend.ID
}

func (r *Route) UpdateBackendWeight(id uuid.UUID, newWeigth int) error {
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
		for _, backend := range r.Backends {
			req, err := http.NewRequest("GET", backend.HealthCheckURL, nil)
			if err != nil {
				log.Error(err.Error())
				continue
			}

			resp, m, err := r.Client.Send(req)
			if err != nil {
				// not worth printing healthcheck error as error
				log.Debug(err.Error())
				if backend.Active {
					backend.UpdateStatus(false)
				}

				m.RequestMethod = req.Method
				m.DownstreamAddr = req.RemoteAddr
				m.ResponseStatus = 600
				m.BackendID = backend.ID
				r.MetricsChan <- m
				continue
			}

			m.RequestMethod = req.Method
			m.DownstreamAddr = req.RemoteAddr
			m.ResponseStatus = resp.StatusCode
			m.BackendID = backend.ID

			r.MetricsChan <- m
		}
		time.Sleep(5 * time.Second)
	}
}
