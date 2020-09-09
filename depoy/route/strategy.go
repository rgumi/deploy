package route

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// SlipperyHandler uses a Canary Strategy and select a backend for forwarding
// based on its weight. Each request is handled seperately and no sessions are
// handled
func (r *Route) SlipperyHandler() func(w http.ResponseWriter, req *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		//var err error
		var b []byte

		currentTarget, err := r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			http.Error(w, "No Upstream Host Available", 503)
			return
		}

		b, _ = ioutil.ReadAll(req.Body)

		defer req.Body.Close()

		resp, m, err := r.sendRequestToUpstream(currentTarget, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "Unable to send request to upstream client", 503)
			return
		}
		m.ContentLength = int64(sendResponse(resp, w))
		r.MetricsRepo.InChannel <- m
	}
}

// StickyHandler uses a Canary Strategy and selects a backend for forwarding
// based on its weight. StickyHandler also sets a session cookie so that all
// following requests are forwarded to the same backend
func (r *Route) StickyHandler() func(w http.ResponseWriter, req *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {
		//var err error
		var b []byte
		var err error
		var currentTarget *Backend
		cookieName := strings.ToUpper(r.Name) + "_SESSIONCOOKIE"

		if value, expires := checkCookie(req, cookieName); value != "" {

			backendName, err := uuid.Parse(value)
			log.Debugf("Found routeCookie for %v", backendName)

			// check if uuid is valid and cookie is not expired
			if err == nil && expires.Before(time.Now()) {
				if target, found := r.Backends[backendName]; found {
					if target.Active {
						currentTarget = r.Backends[backendName]
						goto forward
					}
				}
			}
		}

		currentTarget, err = r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			http.Error(w, "No Upstream Host Available", 503)
			return
		}

		log.Debugf("Setting routeCookie for %v", currentTarget.ID)
		addCookie(w, cookieName, currentTarget.ID.String(), r.CookieTTL)

	forward:

		b, _ = ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		resp, m, err := r.sendRequestToUpstream(currentTarget, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "Unable to send request to upstream client", 503)
			return
		}
		m.ContentLength = int64(sendResponse(resp, w))
		r.MetricsRepo.InChannel <- m
	}
}

// HeaderHandler is used to check the header of an downstream request
// if a routing header is found, the request is routed to the specified backend
func (r *Route) HeaderHandler(headerName, headerValue string, old, new *Backend) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var b []byte

		// check if header exist
		if value := req.Header.Get(headerName); value != "" {
			// header is set therefore new backend will be used
			// headerValue is ignored
			b, _ = ioutil.ReadAll(req.Body)
			defer req.Body.Close()

			resp, m, err := r.sendRequestToUpstream(new, req, b)
			if err != nil {
				log.Warnf("Unable to send request to upstream client: %v", err)
				http.Error(w, "Unable to send request to upstream client", 503)
				return
			}
			m.ContentLength = int64(sendResponse(resp, w))
			r.MetricsRepo.InChannel <- m
			return
		}

		// header is not set therefore old backend is used
		b, _ = ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		resp, m, err := r.sendRequestToUpstream(old, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "Unable to send request to upstream client", 503)
			return
		}
		m.ContentLength = int64(sendResponse(resp, w))
		r.MetricsRepo.InChannel <- m
		return
	}
}

// Shadow accepts requests of the downstream client and forward it to two backends
// (the new version and the old version). Only the response of the old version is
// returned. Both responses can then be compared
func (r *Route) Shadow(old, new *Backend) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var b []byte

		// send request to old backend
		b, _ = ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		resp, m, err := r.sendRequestToUpstream(old, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "Unable to send request to upstream client", 503)
			return
		}

		// return only old response
		m.ContentLength = int64(sendResponse(resp, w))
		r.MetricsRepo.InChannel <- m

		// send request to new backend
		b, _ = ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		resp, m, err = r.sendRequestToUpstream(new, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			return
		}
		b, _ = ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		m.ContentLength = int64(len(b))
		r.MetricsRepo.InChannel <- m
	}
}
