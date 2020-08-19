package route

import (
	"bytes"
	"depoy/upstreamclient"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// Request to get total message time (request to gateway --> answer to downstream client)
type Connection struct {
	Request  *http.Request
	Response http.ResponseWriter
	ID       uuid.UUID
}

type UpstreamClient interface {
	Send(*http.Request) (*http.Response, upstreamclient.Metric, error)
}

type Route struct {
	Prefix         string
	Methods        []string
	Rewrite        string
	Client         UpstreamClient
	Targets        []Target
	ConnectionPool uint
	ShutdownChan   chan int
	ConnectionChan chan Connection
	MetricChan     chan upstreamclient.Metric
	Strategy       Strategy
	Handler        httprouter.Handle
}

func (r *Route) SetMetricChannel(ch chan upstreamclient.Metric) {
	r.MetricChan = ch
}

func New(prefix, rewrite string, methods []string, connPool uint, strategy Strategy) (*Route, error) {

	route := new(Route)
	client := upstreamclient.New()
	route.Client = client
	route.Prefix = prefix
	route.Rewrite = rewrite
	route.Methods = methods
	route.Strategy = strategy
	route.Handler = route.GetExternalHandle()

	route.ConnectionChan = make(chan Connection, 5)
	route.MetricChan = make(chan upstreamclient.Metric, 5)
	route.ShutdownChan = make(chan int)

	if connPool <= 0 {
		return nil, fmt.Errorf("ConnectionPool needs to be atleast 1")
	}
	route.ConnectionPool = connPool
	return route, nil
}

// Init activates the route by starting the goroutines
func (r *Route) Init() {
	// Start connection pool
	var i uint
	for i = 0; i < r.ConnectionPool; i++ {
		go r.upstreamClient()
	}

	//
}

func (r *Route) upstreamClient() {
	for {
		select {
		case conn := <-r.ConnectionChan:
			log.Debugf("%+v", conn.Request)

			// steuere die Verteilung falls zwei Versionen hinter der Route sind!
			currentTarget := r.Targets[r.Strategy.GetTargetIndex()]
			log.Debug("Creating new request for upstream client")

			newReq, err := FormateRequest(conn.Request, currentTarget.Addr)
			if err != nil {
				log.Error("Failed to formate request")
				log.Error(err)
				conn.Response.WriteHeader(503)

			} else {
				// why is the response written here????

				resp, metrics, err := r.Client.Send(newReq)

				if err != nil {
					log.Error("Failed to send request to upstream host")
					log.Error(err)
					conn.Response.WriteHeader(503)

				} else {
					log.Debug("Sending response to downstream client")
					SendResponse(resp, conn.Response)
					r.MetricChan <- metrics
				}
			}

		case _ = <-r.ShutdownChan:
			log.Debug("[UpstreamClient] Received shutdown signal")
			return
		}
	}
}

// Shutdown the Route and all its goroutines
func (r *Route) Shutdown() {
	log.Debug("Shutting down go routes of Route")
	var i uint
	for i = 0; i < r.ConnectionPool; i++ {
		r.ShutdownChan <- 1
	}
}

func SendResponse(resp *http.Response, w http.ResponseWriter) {
	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		log.Error("Failed to read response body of upstream response")
		log.Error(err)
		http.Error(w, "Could not return response", 500)

		return
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(b)
}

func FormateRequest(old *http.Request, addr string) (*http.Request, error) {
	b, err := ioutil.ReadAll(old.Body)

	if err != nil {
		log.Error("Unable ro read body of downstream request")
		log.Error(err)
		return nil, err
	}
	defer old.Body.Close()
	reader := bytes.NewReader(b)

	new, err := http.NewRequest(old.Method, addr, reader)
	if err != nil {
		return nil, err
	}

	// setup the X-Forwarded-For header
	if clientIP, _, err := net.SplitHostPort(old.RemoteAddr); err == nil {
		appendHostToXForwardHeader(new.Header, clientIP)
	}

	// Copy all headers from the downstream request to the new upstream request
	copyHeaders(old.Header, new.Header)
	// Remove all headers from the new upstream request that are bound to downstream connection
	delHopHeaders(new.Header)

	return new, nil
}

func (r *Route) GetHandle() func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	return func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		uuid := uuid.New()
		log.Debugf("[%s] Received request on route", uuid.String())

		r.ConnectionChan <- Connection{
			Request:  req,
			Response: w,
			ID:       uuid,
		}
		log.Debugf("[%s] Forwarded request to route", uuid.String())
		return
	}
}

func (r *Route) GetExternalHandle() func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	return func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		currentTarget := r.Targets[r.Strategy.GetTargetIndex()]
		newReq, err := FormateRequest(req, currentTarget.Addr)
		if err != nil {
			http.Error(w, "Unable to formate new request", 500)
			return
		}
		resp, m, err := r.Client.Send(newReq)
		if err != nil {
			http.Error(w, "Unable so send request to upstream client", 503)
			return
		}
		SendResponse(resp, w)
		r.MetricChan <- m

		return
	}
}
