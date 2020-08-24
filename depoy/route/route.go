package route

import (
	"bytes"
	"depoy/upstreamclient"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
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
	Prefix   string         `json:"prefix"`
	Methods  []string       `json:"methods"`
	Host     string         `json:"host"`
	Rewrite  string         `json:"rewrite"`
	Client   UpstreamClient `json:"-"`
	Targets  []*Target      `json:"targets"`
	Strategy Strategy       `json:"-"`
	handler  http.HandlerFunc
}

func New(prefix, rewrite, host string, methods []string, strategy Strategy) (*Route, error) {

	route := new(Route)
	client := upstreamclient.NewClient()
	route.Client = client

	// fix prefix if prefix does not end with /
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}

	route.Prefix = prefix
	route.Rewrite = rewrite
	route.Methods = methods
	route.Host = host
	route.Strategy = strategy
	route.handler = route.GetExternalHandle()

	return route, nil
}

func (r *Route) GetHandler() http.HandlerFunc {
	return r.handler
}

func sendResponse(resp *http.Response, w http.ResponseWriter) {
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

func formateRequest(old *http.Request, addr string) (*http.Request, error) {
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

		// rewrite the url
		// rewrite == "" => no rewrite
		// rewrite == "/" => replace prefix with /
		url := req.URL.String()

		if r.Rewrite != "" {
			url = strings.Replace(url, r.Prefix, r.Rewrite, -1)
			log.Debugf("Rewriting upstream URL from %s to %s", req.URL.String(), url)
		}
		url = currentTarget.Addr + url

		// Copies the downstream request into the upstream request
		// and formats it accordingly
		newReq, err := formateRequest(req, url)
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
		sendResponse(resp, w)

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
