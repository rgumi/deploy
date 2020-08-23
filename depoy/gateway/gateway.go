package gateway

import (
	"depoy/route"
	"depoy/router"
	"fmt"
	"hash/fnv"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	//AllowedMethods are supported by the Gateway
	AllowedMethods = []string{"GET", "POST", "PATCH"}
)

//Gateway has a HTTP-Server which has Routes configured for it
type Gateway struct {
	Addr         string
	ReadTimeout  int
	WriteTimeout int
	Routes       map[uint32]*route.Route
	Router       map[string]*router.Router
}

//NewGateway returns a new instance of Gateway
func NewGateway(addr string) *Gateway {
	g := new(Gateway)
	g.Addr = addr
	g.Routes = make(map[uint32]*route.Route)

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

	log.Debug("Creating new Router")

	newRouter := make(map[string]*router.Router)
	newRouter["*"] = router.NewRouter()

	for _, routeItem := range g.Routes {

		if _, found := newRouter[routeItem.Host]; !found {
			// host does not exist
			newRouter[routeItem.Host] = router.NewRouter()
		}

		// add all routes
		for _, method := range routeItem.Methods {
			newRouter[routeItem.Host].AddHandler(method, routeItem.Prefix, routeItem.Handler)
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

func getID(route *route.Route) uint32 {
	rawID := fmt.Sprintf("%s%s", route.Host, route.Prefix)
	log.Debugf("Created new ID for %v: %s", route, rawID)

	hash := fnv.New32a()
	hash.Write([]byte(rawID))
	log.Debugf("Created new hash for %v: %d", route, hash.Sum32())
	return hash.Sum32()
}

func (g *Gateway) checkIfExists(route *route.Route) bool {
	routeID := getID(route)

	if _, exists := g.Routes[routeID]; exists {
		return true
	}
	return false
}

func (g *Gateway) RegisterRoute(route *route.Route) error {

	// create id of route
	if g.checkIfExists(route) {
		return fmt.Errorf("Route with given host and prefix already exists")
	}

	// check if route exists
	g.Routes[getID(route)] = route
	g.Reload()

	return nil
}

func (g *Gateway) RemoveRoute(id uint32) {
	if _, exists := g.Routes[id]; exists {
		log.Debugf("Removing %d from Gateway.Routes", id)

		delete(g.Routes, id)

		g.Reload()
	}
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
