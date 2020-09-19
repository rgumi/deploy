package route

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type Strategy struct {
	Type        string                                         `json:"type" yaml:"type" validate:"empty=false"`
	HeaderName  string                                         `json:"header_name,omitempty" yaml:"header_name,omitempty"`
	HeaderValue string                                         `json:"header_value,omitempty" yaml:"header_value,omitempty"`
	Old         string                                         `json:"old_backend,omitempty" yaml:"old_backend,omitempty"`
	New         string                                         `json:"new_backend,omitempty" yaml:"new_backend,omitempty"`
	Shadow      string                                         `json:"shadow_backend,omitempty" yaml:"shadow_backend,omitempty"`
	Handler     func(w http.ResponseWriter, req *http.Request) `json:"-" yaml:"-"`
}

func (s *Strategy) Validate(newRoute *Route) (err error) {

	switch t := strings.ToLower(s.Type); t {

	case "sticky":
		if newRoute == nil {
			return fmt.Errorf("Parameter route cannot be nil")
		}

	case "slippery":
		if newRoute == nil {
			return fmt.Errorf("Parameter route cannot be nil")
		}

	case "shadow":
		if newRoute == nil || s.Shadow == "" {
			return fmt.Errorf("Required parameter are missing")
		}

	case "header":
		if newRoute == nil || s.HeaderName == "" || s.HeaderValue == "" || s.Old == "" || s.New == "" {
			return fmt.Errorf("Required parameter are missing")
		}

	default:
		return fmt.Errorf("Unsupported strategy type (%s)", t)
	}
	return nil
}

// Reset converts an old strategy into a new strategy-instance
// and sets it for the provided route
func (s *Strategy) Reset(newRoute *Route) error {
	switch t := strings.ToLower(s.Type); t {
	case "sticky":
		strat, err := NewStickyStrategy(newRoute)
		if err != nil {
			return err
		}
		newRoute.SetStrategy(strat)
	case "slippery":
		strat, err := NewSlipperyStrategy(newRoute)
		if err != nil {
			return err
		}
		newRoute.SetStrategy(strat)
	case "shadow":
		strat, err := NewShadowStrategy(newRoute, s.Shadow)
		if err != nil {
			return err
		}
		newRoute.SetStrategy(strat)
	case "header":
		strat, err := NewHeaderStrategy(
			newRoute, s.HeaderName, s.HeaderValue, s.Old, s.New)

		if err != nil {
			return err
		}
		newRoute.SetStrategy(strat)
	default:
		return fmt.Errorf("Unsupported strategy type (%s)", t)
	}
	return nil
}

func NewSlipperyStrategy(r *Route) (*Strategy, error) {
	if r == nil {
		return nil, fmt.Errorf("router cannot be nil")
	}

	return &Strategy{
		Type:    "slippery",
		Handler: SlipperyHandler(r),
	}, nil
}

func NewStickyStrategy(r *Route) (*Strategy, error) {
	if r == nil {
		return nil, fmt.Errorf("router cannot be nil")
	}

	return &Strategy{
		Type:    "sticky",
		Handler: StickyHandler(r),
	}, nil
}

func NewHeaderStrategy(r *Route, headerName, headerValue, oldBackend, newBackend string) (*Strategy, error) {
	var old, new *Backend

	if r == nil || headerName == "" || headerValue == "" || oldBackend == "" || newBackend == "" {
		return nil, fmt.Errorf("Required parameter are missing")
	}

	for _, backend := range r.Backends {
		if backend.Name == oldBackend {
			old = backend
		} else if backend.Name == newBackend {
			new = backend
		}
	}

	if old == nil || new == nil {
		return nil, fmt.Errorf("Unable to find the provided backends")
	}

	return &Strategy{
		Type:        "header",
		HeaderName:  headerName,
		HeaderValue: headerValue,
		Old:         oldBackend,
		New:         newBackend,
		Handler:     HeaderHandler(r, headerName, headerValue, old, new),
	}, nil
}

func NewShadowStrategy(r *Route, shadowBackend string) (*Strategy, error) {
	var shadow *Backend

	if r == nil || shadowBackend == "" {
		return nil, fmt.Errorf("Required parameter are missing")
	}

	for _, backend := range r.Backends {
		if backend.Name == shadowBackend {
			shadow = backend
		}
	}

	if shadow == nil {
		return nil, fmt.Errorf("Unable to find the provided backend")
	}

	return &Strategy{
		Type:    "shadow",
		Shadow:  shadowBackend,
		Handler: ShadowHandler(r, shadow),
	}, nil
}

// SlipperyHandler uses a Canary Strategy and select a backend for forwarding
// based on its weight. Each request is handled seperately and no sessions are
// handled
func SlipperyHandler(r *Route) func(w http.ResponseWriter, req *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {

		defer req.Body.Close()

		//var err error
		var b []byte

		currentTarget, err := r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			http.Error(w, "No Upstream Host Available", 503)
			return
		}

		b, _ = ioutil.ReadAll(req.Body)

		resp, m, err := r.sendRequestToUpstream(currentTarget, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "Unable to send request to upstream client", 503)
			return
		}
		defer resp.Body.Close()
		m.ContentLength = int64(sendResponse(resp, w))
		r.MetricsRepo.InChannel <- m
	}
}

// StickyHandler uses a Canary Strategy and selects a backend for forwarding
// based on its weight. StickyHandler also sets a session cookie so that all
// following requests are forwarded to the same backend
func StickyHandler(r *Route) func(w http.ResponseWriter, req *http.Request) {

	return func(w http.ResponseWriter, req *http.Request) {

		defer req.Body.Close()

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

		resp, m, err := r.sendRequestToUpstream(currentTarget, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "Unable to send request to upstream client", 503)
			return
		}
		defer resp.Body.Close()

		m.ContentLength = int64(sendResponse(resp, w))
		r.MetricsRepo.InChannel <- m
	}
}

// HeaderHandler is used to check the header of an downstream request
// if a routing header is found, the request is routed to the specified backend
func HeaderHandler(r *Route, headerName, headerValue string, old, new *Backend) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		var b []byte

		// check if header exist
		if value := req.Header.Get(headerName); value != "" {
			// header is set therefore new backend will be used
			// headerValue is ignored
			b, _ = ioutil.ReadAll(req.Body)

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

		resp, m, err := r.sendRequestToUpstream(old, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "Unable to send request to upstream client", 503)
			return
		}
		defer resp.Body.Close()

		m.ContentLength = int64(sendResponse(resp, w))
		r.MetricsRepo.InChannel <- m
		return
	}
}

// ShadowHandler accepts requests of the downstream client and forward it to two backends
// (the new version and the old version). Only the response of the old version is
// returned. Both responses can then be compared
func ShadowHandler(r *Route, shadow *Backend) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		var b []byte

		// send request to old backend
		b, _ = ioutil.ReadAll(req.Body)

		currentTarget, err := r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			http.Error(w, "No Upstream Host Available", 503)
			return
		}

		resp, m, err := r.sendRequestToUpstream(currentTarget, req, b)
		if err != nil {
			log.Warnf("Unable to send request to upstream client: %v", err)
			http.Error(w, "Unable to send request to upstream client", 503)
			return
		}
		defer resp.Body.Close()

		m.ContentLength = int64(sendResponse(resp, w))
		r.MetricsRepo.InChannel <- m

		// send request to shadowed backend
		go func() {
			b, _ = ioutil.ReadAll(req.Body)
			defer req.Body.Close()

			respShadow, m, err := r.sendRequestToUpstream(shadow, req, b)
			if err != nil {
				log.Warnf("Unable to send request to upstream client: %v", err)
				return
			}
			b, _ = ioutil.ReadAll(respShadow.Body)
			defer respShadow.Body.Close()
			m.ContentLength = int64(len(b))
			r.MetricsRepo.InChannel <- m
		}()
	}
}
