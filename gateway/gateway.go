package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rgumi/depoy/metrics"
	"github.com/rgumi/depoy/route"
	"github.com/rgumi/depoy/router"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// defaults
var (
	ReadHeaderTimeout = 2 * time.Second
)

//Gateway has a HTTP-Server which has Routes configured for it
type Gateway struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	HTTPTimeout  time.Duration
	IdleTimeout  time.Duration
	Routes       map[string]*route.Route
	Router       map[string]*router.Router
	MetricsRepo  *metrics.Repository
	server       http.Server
	mux          sync.Mutex
}

//NewGateway returns a new instance of Gateway
func NewGateway(
	addr string, metricsRepo *metrics.Repository,
	readTimeout, writeTimeout, httpTimeout, idleTimeout time.Duration) *Gateway {

	g := new(Gateway)

	// initialize a new MetricsRepo for metric collection and evaluation
	g.MetricsRepo = metricsRepo

	g.Addr = addr
	// initialize the map for storing the routes
	g.Routes = make(map[string]*route.Route)

	// map for each HOST
	g.Router = make(map[string]*router.Router)

	// any HOST router
	g.Router["*"] = router.NewRouter()

	// set timeouts
	g.ReadTimeout = readTimeout
	g.WriteTimeout = writeTimeout
	g.HTTPTimeout = httpTimeout
	g.IdleTimeout = idleTimeout

	return g
}

// Reload the entire config of the Gateway
// initializes a new router which is then configured based on the
// defined Routes in the Gateway. Then the old Router is replaced by
// a pointer to the new Router
func (g *Gateway) Reload() {
	log.Info("Reloading Gateway")
	newRouter := make(map[string]*router.Router)
	// any host router
	newRouter["*"] = router.NewRouter()
	for _, routeItem := range g.Routes {
		// Each host has its own router
		if _, found := newRouter[routeItem.Host]; !found {
			// host does not exist, create its router
			newRouter[routeItem.Host] = router.NewRouter()
		}
		// add all routes to the router
		for _, method := range routeItem.Methods {
			// for each http-method add a handler to the router
			newRouter[routeItem.Host].AddHandler(method, routeItem.Prefix, routeItem.GetHandler())
		}
	}
	// overwrite existing tree with new
	g.Router = newRouter
}

// Run starts the HTTP-Server of the Gateway
func (g *Gateway) Run() {
	g.server = http.Server{
		Addr:              g.Addr,
		Handler:           http.TimeoutHandler(g, g.HTTPTimeout, "HTTP Handling Timeout"),
		WriteTimeout:      g.WriteTimeout,
		ReadTimeout:       g.ReadTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
		IdleTimeout:       g.IdleTimeout,
	}

	go func() {
		log.Info("Starting gateway server")
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("gateway server listen failed with %v\n", err)
		}
		log.Debug("Successfully shutdown gateway server")
	}()
}

// checkIfExists checks if the newRoute is already present on the Gateway
func (g *Gateway) checkIfExists(newRoute *route.Route) error {

	for routeName, route := range g.Routes {
		// if name is already defined, return error
		if routeName == newRoute.Name {
			return fmt.Errorf("Route with name %s already exists", routeName)
		}

		// if name is not taken, check if other configs are taken
		// if combination of prefix/host is already taken, return error
		if route.Prefix == newRoute.Prefix && route.Host == newRoute.Host {
			return fmt.Errorf(
				"Route with combination of prefix (%s) and host (%s) already exist. Existing Route: %s",
				route.Prefix, route.Host, routeName)
		}
	}
	// no error
	return nil
}

// GetRoute returns the Route, if it exists. Otherwise nil
func (g *Gateway) GetRoute(routeName string) *route.Route {

	if route, found := g.Routes[routeName]; found {
		return route
	}
	return nil
}

// RegisterRoute registers a new Route to the Gateway
// if Route is valid, the route is added to the Gateway
// and the Gateway is reloaded
func (g *Gateway) RegisterRoute(newRoute *route.Route) error {
	g.mux.Lock()
	defer g.mux.Unlock()

	var err error
	log.Debugf("Trying to register new route %s", newRoute.Name)

	if newRoute.Name == "" {
		return fmt.Errorf("Route.Name cannot be empty")
	}
	if err = g.checkIfExists(newRoute); err != nil {
		return err
	}
	if g.MetricsRepo == nil {
		panic(fmt.Errorf("Gateway MetricsRepo is nil"))
	}
	log.Debugf("Setting up MetricsRepo for %s", newRoute.Name)
	newRoute.MetricsRepo = g.MetricsRepo

	g.Routes[newRoute.Name] = newRoute
	return nil
}

// RemoveRoute removes the Route from the Gateway and stops/removes
// all children of the Route (backends, monitoring components)
// then reload the Gateway
func (g *Gateway) RemoveRoute(name string) *route.Route {

	g.mux.Lock()
	defer g.mux.Unlock()

	if route, exists := g.Routes[name]; exists {
		log.Warnf("Removing %s from Gateway.Routes", name)

		route.StopAll()

		delete(g.Routes, name)

		log.Debugf("Deleted Route %s from Gateway", route.Name)
		g.Reload()

		return route
	}

	return nil
}

// ServeHTTP is the required interface to quality as http.Handler
// so the Gateway can be executed as a http.Server
func (g *Gateway) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if router, found := g.Router[req.Host]; found {
		router.ServeHTTP(w, req)
		return

	}
	g.Router["*"].ServeHTTP(w, req)
}

// GetRoutes returns all Routes that are configured for the Gateway
func (g *Gateway) GetRoutes() map[string]*route.Route {
	return g.Routes
}

// Stop executes a shutdown of the Gateway server and removes all
// routes of the Gateway
func (g *Gateway) Stop() {

	for routeName := range g.Routes {
		g.RemoveRoute(routeName)
	}
	g.MetricsRepo.Stop()

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer func() {
		cancel()
	}()

	if err := g.server.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("gateway server shutdown failed: %v\n", err)
	}
}

// ReadConfig reads the current config of the Gateway and returns a []byte
func (g *Gateway) ReadConfig() ([]byte, error) {
	b, err := yaml.Marshal(g)
	if err != nil {
		return nil, err
	}
	return b, nil
}
