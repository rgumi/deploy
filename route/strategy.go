package route

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type Strategy struct {
	Type        string                         `json:"type" yaml:"type" validate:"empty=false"`
	HeaderName  string                         `json:"header_name,omitempty" yaml:"headerName,omitempty"`
	HeaderValue string                         `json:"header_value,omitempty" yaml:"headerValue,omitempty"`
	Target      string                         `json:"target_backend,omitempty" yaml:"targetBackend,omitempty"`
	Handler     func(ctx *fasthttp.RequestCtx) `json:"-" yaml:"-"`
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

func NewStickyStrategy(r *Route) (*Strategy, error) {
	st := &Strategy{
		Type:    "sticky",
		Handler: StickyHandler(r),
	}
	return st, st.Validate(r)
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
	target.Weigth = 0

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

	shadow.Weigth = 0

	return &Strategy{
		Type:    "shadow",
		Target:  shadowBackend,
		Handler: ShadowHandler(r, shadow),
	}, nil
}

// StickyHandler uses a Canary Strategy and selects a backend for forwarding
// based on its weight. StickyHandler also sets a session cookie so that all
// following requests are forwarded to the same backend
func StickyHandler(r *Route) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		var err error
		var target *Backend
		c := fasthttp.AcquireCookie()

		if value := string(ctx.Request.Header.Cookie(r.cookieName)); value != "" {
			BackendID, err := uuid.Parse(value)
			log.Debugf("Found routeCookie for %v", BackendID)
			if err == nil {
				if t, found := r.Backends[BackendID]; found {
					if t.Active {
						target = t
						fasthttp.ReleaseCookie(c)
						c = nil
						goto forward
					}
				}
			}
		}
		target, err = r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			ctx.Error("No Upstream Host Available", 503)
			return
		}
		log.Debugf("Setting new routeCookie for %v", target.ID)
		c.SetKey(r.cookieName)
		c.SetValue(target.ID.String())
		defer fasthttp.ReleaseCookie(c)

	forward:
		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)
		ctx.Request.CopyTo(req)
		appendXForwardForHeader(req, ctx.RemoteAddr().String())
		delRequestHopHeader(req)
		r.HTTPDo(req, target, HTTPReturn(ctx, c))
	}
}

// HeaderHandler is used to check the header of an downstream request
// if a routing header is found, the request is routed to the specified backend
func HeaderHandler(r *Route, headerName, headerValue string, target *Backend) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		var err error

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)
		ctx.Request.CopyTo(req)
		delRequestHopHeader(req)
		appendXForwardForHeader(req, ctx.RemoteAddr().String())

		if len(ctx.Request.Header.Peek(headerName)) > 0 {
			r.HTTPDo(req, target, HTTPReturn(ctx, nil))
			return
		}

		target, err = r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			ctx.Error("No Upstream Host Available", 503)
			return
		}
		r.HTTPDo(req, target, HTTPReturn(ctx, nil))
	}
}

// ShadowHandler accepts requests of the downstream client and forward it to two backends
// (the new version and the old version). Only the response of the old version is
// returned. Both responses can then be compared
func ShadowHandler(r *Route, shadow *Backend) func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		target, err := r.getNextBackend()
		if err != nil {
			log.Debugf("Could not get next backend: %v", err)
			ctx.Error("No Upstream Host Available", 503)
			return
		}

		req1 := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req1)
		ctx.Request.CopyTo(req1)
		delRequestHopHeader(req1)
		appendXForwardForHeader(req1, ctx.RemoteAddr().String())

		req2 := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req2)
		req2.SetBody(req1.Body())
		req1.Header.CopyTo(&req2.Header)

		if err = r.HTTPDo(req1, target, HTTPReturn(ctx, nil)); err != nil {
			ctx.Error(handleNetError(err))
		}

		go func() {
			if err = r.HTTPDo(req2, shadow, func(resp *fasthttp.Response) {
				return
			}); err != nil {
				log.Infof("Shadow Request failed with %s", err.Error())
			}
		}()
	}
}
