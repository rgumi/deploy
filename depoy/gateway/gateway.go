package gateway

import (
	"depoy/metrics"
	"depoy/route"
	"depoy/router"
	"fmt"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	//AllowedMethods are supported by the Gateway
	AllowedMethods = []string{"GET", "POST", "PATCH"}
)

//Gateway has a HTTP-Server which has Routes configured for it
type Gateway struct {
	mux          sync.Mutex
	Addr         string
	ReadTimeout  int
	WriteTimeout int
	Routes       map[string]*route.Route // name => route. Name is unique
	Router       map[string]*router.Router
	MetricsRepo  *metrics.Repository
}

//NewGateway returns a new instance of Gateway
func NewGateway(addr string) *Gateway {
	g := new(Gateway)
	_, g.MetricsRepo = metrics.NewMetricsRepository(nil, 5*time.Second)

	go g.MetricsRepo.Listen()

	g.Addr = addr
	g.Routes = make(map[string]*route.Route)

	// map for each HOST
	g.Router = make(map[string]*router.Router)

	// any HOST router
	g.Router["*"] = router.NewRouter()

	// set defaults
	g.ReadTimeout = 10000
	g.WriteTimeout = 10000

	return g
}

func (g *Gateway) Reload() {

	log.Info("Reloading backend")

	newRouter := make(map[string]*router.Router)
	newRouter["*"] = router.NewRouter()

	for _, routeItem := range g.Routes {

		if _, found := newRouter[routeItem.Host]; !found {
			// host does not exist
			newRouter[routeItem.Host] = router.NewRouter()
		}

		// add all routes
		for _, method := range routeItem.Methods {
			newRouter[routeItem.Host].AddHandler(method, routeItem.Prefix, routeItem.GetHandler())
		}
	}

	// overwrite existing tree with new
	g.Router = newRouter
}

// Run starts the HTTP-Server of the Gateway
func (g *Gateway) Run() {
	server := http.Server{
		Addr:              g.Addr,
		Handler:           http.TimeoutHandler(g, 10*time.Second, "HTTP Handling Timeout"),
		WriteTimeout:      time.Duration(g.WriteTimeout) * time.Millisecond,
		ReadTimeout:       time.Duration(g.ReadTimeout) * time.Millisecond,
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func (g *Gateway) checkIfExists(newRoute *route.Route) error {

	for routeName, route := range g.Routes {
		if routeName == newRoute.Name {
			return fmt.Errorf("Route with name %s already exists", routeName)
		}
		if route.Prefix == newRoute.Prefix && route.Host == newRoute.Host {
			return fmt.Errorf(
				"Route with combination of prefix (%s) and host (%s) already exist. Existing Route: %s",
				route.Prefix, route.Host, routeName)
		}
	}
	return nil
}

func (g *Gateway) RegisterRoute(route *route.Route) error {
	var err error

	g.mux.Lock()
	defer g.mux.Unlock()

	if route.Name == "" {
		return fmt.Errorf("Route.Name cannot be empty")
	}

	// create id of route
	if err = g.checkIfExists(route); err != nil {
		return err
	}

	route.MetricsChan = g.MetricsRepo.InChannel

	if len(route.Backends) == 0 {
		log.Warnf("Route %s has no backends configured for it", route.Name)

	} else {
		for _, backend := range route.Backends {
			log.Debugf("Registering %v of Route %s to MetricsRepository", backend.ID, route.Name)

			// register the backend
			backend.AlertChan, err = g.MetricsRepo.RegisterBackend(
				backend.ID, backend.ScrapeURL, backend.ScrapeMetrics)

			if err != nil {
				return err
			}

			// start monitoring the registered backend
			go g.MetricsRepo.Monitor(backend.ID, 5*time.Second, 10*time.Second)
			if err != nil {
				return err
			}

			// starts listening on alertChan
			go backend.Monitor()
		}
	}

	g.Routes[route.Name] = route
	g.Reload()
	return nil
}

func (g *Gateway) RemoveRoute(name string) *route.Route {

	g.mux.Lock()
	defer g.mux.Unlock()

	if route, exists := g.Routes[name]; exists {
		log.Debugf("Removing %s from Gateway.Routes", name)

		delete(g.Routes, name)

		g.Reload()
		return route
	}
	return nil
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	if router, found := g.Router[req.Host]; found {
		log.Debugf("Found explicit match for %s", req.Host)
		router.ServeHTTP(w, req)

	} else {
		log.Debugf("Did not find explicit match for %s. Using any Host", req.Host)
		g.Router["*"].ServeHTTP(w, req)
	}
}

func (g *Gateway) GetRouteByName(name string) *route.Route {
	return g.Routes[name]
}

func (g *Gateway) GetRoutes() map[string]*route.Route {
	return g.Routes
}
