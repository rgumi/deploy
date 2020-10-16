package route

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rgumi/depoy/metrics"
	log "github.com/sirupsen/logrus"
)

type Strategy struct {
	Type        string                                         `json:"type" yaml:"type" validate:"empty=false"`
	HeaderName  string                                         `json:"header_name,omitempty" yaml:"headerName,omitempty"`
	HeaderValue string                                         `json:"header_value,omitempty" yaml:"headerValue,omitempty"`
	Target      string                                         `json:"target_backend,omitempty" yaml:"targetBackend,omitempty"`
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
		if newRoute == nil || s.Target == "" {
			return fmt.Errorf("Required parameter are missing")
		}

	case "header":
		if newRoute == nil || s.HeaderName == "" || s.HeaderValue == "" || s.Target == "" {
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
		strat, err := NewShadowStrategy(newRoute, s.Target)
		if err != nil {
			return err
		}
		newRoute.SetStrategy(strat)
	case "header":
		strat, err := NewHeaderStrategy(
			newRoute, s.HeaderName, s.HeaderValue, s.Target)

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

func NewHeaderStrategy(r *Route, headerName, headerValue, targetBackend string) (*Strategy, error) {
	var target *Backend

	if r == nil || headerName == "" || headerValue == "" || targetBackend == "" {
		return nil, fmt.Errorf("Required parameter are missing")
	}

	for _, backend := range r.Backends {
		if backend.Name == targetBackend {
			target = backend
		}
	}

	if target == nil {
		return nil, fmt.Errorf("Unable to find the provided backends")
	}

	return &Strategy{
		Type:        "header",
		HeaderName:  headerName,
		HeaderValue: headerValue,
		Target:      targetBackend,
		Handler:     HeaderHandler(r, headerName, headerValue, target),
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
		Target:  shadowBackend,
		Handler: ShadowHandler(r, shadow),
	}, nil
}

// SlipperyHandler uses a Canary Strategy and select a backend for forwarding
// based on its weight. Each request is handled seperately and no sessions are
// handled
func SlipperyHandler(r *Route) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()

		currentTarget, err := r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			http.Error(w, "No Upstream Host Available", 503)
			return
		}
		if err := r.httpDo(req.Context(), currentTarget, req, req.Body, r.httpReturn(w)); err != nil {
			log.Errorf("Unable to handle request (%v)", err)
			http.Error(w, err.Error(), err.Code())
		}
	}
}

// StickyHandler uses a Canary Strategy and selects a backend for forwarding
// based on its weight. StickyHandler also sets a session cookie so that all
// following requests are forwarded to the same backend
func StickyHandler(r *Route) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		var err error
		var target *Backend
		cookieName := strings.ToUpper(r.Name) + "_SESSIONCOOKIE"

		if value, expires := checkCookie(req, cookieName); value != "" {
			BackendID, err := uuid.Parse(value)
			log.Debugf("Found routeCookie for %v", BackendID)
			// check if uuid is valid and cookie is not expired
			// (browsers do not send expired cookies but w/e)
			if err == nil && expires.Before(time.Now()) {
				if t, found := r.Backends[BackendID]; found {
					if t.Active {
						target = t
						goto forward
					}
				}
			}
		}
		target, err = r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			http.Error(w, "No Upstream Host Available", 503)
			return
		}
		addCookie(w, cookieName, target.ID.String(), r.CookieTTL)

	forward:
		if err := r.httpDo(req.Context(), target, req, req.Body, r.httpReturn(w)); err != nil {
			log.Errorf("Unable to handle request (%v)", err)
			http.Error(w, err.Error(), err.Code())
		}
	}
}

// HeaderHandler is used to check the header of an downstream request
// if a routing header is found, the request is routed to the specified backend
func HeaderHandler(r *Route, headerName, headerValue string, target *Backend) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		var err error

		if value := req.Header.Get(headerName); value != "" {
			// header is set therefore new backend will be used
			// headerValue is ignored
			if err = r.httpDo(req.Context(), target, req, req.Body, r.httpReturn(w)); err != nil {
				log.Errorf("Unable to handle request (%v)", err)
				http.Error(w, err.Error(), 503)
			}
			return
		}

		target, err = r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			http.Error(w, "No Upstream Host Available", 503)
			return
		}
		if err := r.httpDo(req.Context(), target, req, req.Body, r.httpReturn(w)); err != nil {
			log.Errorf("Unable to handle request (%v)", err)
			http.Error(w, err.Error(), err.Code())
		}
	}
}

// ShadowHandler accepts requests of the downstream client and forward it to two backends
// (the new version and the old version). Only the response of the old version is
// returned. Both responses can then be compared
func ShadowHandler(r *Route, shadow *Backend) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		var err error

		// read body so that it can be send to both backends (shadow & actual)
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Errorf("Unable to read body of request (%v)", err)
			http.Error(w, err.Error(), 500)
		}

		// actual backend

		bodyReader := bytes.NewReader(body)
		target, err := r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			http.Error(w, "No Upstream Host Available", 503)
			return
		}
		if err := r.httpDo(req.Context(), target, req, ioutil.NopCloser(bodyReader), r.httpReturn(w)); err != nil {
			log.Errorf("Unable to handle request (%v)", err)
			http.Error(w, err.Error(), err.Code())
		}

		// shadow backend

		bodyReader = bytes.NewReader(body)
		go func() {
			if err := r.httpDo(
				context.TODO(), shadow, req, ioutil.NopCloser(bodyReader),
				func(resp *http.Response, m *metrics.Metrics, err error) GatewayError {
					if err != nil {
						return NewGatewayError(err)
					}
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						return NewGatewayError(err)
					}
					resp.Body.Close()
					m.ResponseStatus = resp.StatusCode
					m.ContentLength = int64(len(body))
					r.MetricsRepo.InChannel <- m

					return nil
				}); err != nil {
				log.Errorf("Unable to handle request of shadow (%v)", err)
			}
		}()
	}
}
