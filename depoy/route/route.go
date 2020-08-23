package route

import (
	"bytes"
	"depoy/upstreamclient"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
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
	Prefix     string
	Methods    []string
	Host       string
	Rewrite    string
	Client     UpstreamClient
	Targets    []*Target
	MetricChan chan upstreamclient.Metric
	Strategy   Strategy
	Handler    http.HandlerFunc
}

func (r *Route) SetMetricChannel(ch chan upstreamclient.Metric) {
	r.MetricChan = ch
}

func New(prefix, rewrite, host string, methods []string, strategy Strategy) (*Route, error) {

	route := new(Route)
	client := upstreamclient.NewClient()
	route.Client = client
	route.Prefix = prefix
	route.Rewrite = rewrite
	route.Methods = methods
	route.Host = host
	route.Strategy = strategy
	route.Handler = route.GetExternalHandle()

	route.MetricChan = make(chan upstreamclient.Metric, 5)

	return route, nil
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

func (r *Route) GetExternalHandle() func(w http.ResponseWriter, req *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("Handler received downstream request")

		// Get the next target depending on the strategy selected
		// TODO: Change this somehow???!!
		currentTarget := r.Targets[r.Strategy.GetTargetIndex()]

		// Copies the downstream request into the upstream request
		// and formats it accordingly
		newReq, err := FormateRequest(req, currentTarget.Addr)
		if err != nil {
			http.Error(w, "Unable to formate new request", 500)
			return
		}

		// Send Request to the upstream host
		// reuses TCP-Connection
		resp, m, err := r.Client.Send(newReq)
		if err != nil {
			log.Errorf(err.Error())
			http.Error(w, "Unable so send request to upstream client", 503)
			return
		}

		// Send the Response to the downstream client
		sendStart := time.Now()
		SendResponse(resp, w)

		// Collect important metrics
		m.ResponseSendTime = time.Since(sendStart).Milliseconds()
		m.Method = req.Method
		m.DownstreamAddr = req.RemoteAddr

		// consume metrics
		currentTarget.MetricChan <- m
		return
	}
}

func (r *Route) AddTarget(target *Target) {
	r.Targets = append(r.Targets, target)
}
