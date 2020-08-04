package gateway

import (
	"depoy/middleware"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
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
	Router       *httprouter.Router
	Routes       map[string]Route
}

//New returns a new instance of Gateway
func New(addr string) *Gateway {
	g := new(Gateway)
	g.Addr = addr
	g.Router = httprouter.New()

	// set defaults
	g.ReadTimeout = 10000
	g.WriteTimeout = 10000

	return g
}

func contains(array []string, sub string) bool {
	for _, item := range array {
		if item == sub {
			return true
		}
	}
	return false
}

func formateID(method, path string) string {
	// remove unwanted "/" to allow for comparison
	cleanPath := strings.Trim(path, "/")
	if !contains(AllowedMethods, method) {
		return ""
	}
	return method + cleanPath
}

func (g *Gateway) routeExists(id string) (bool, error) {
	for key := range g.Routes {
		if key == id {
			return true, nil
		}
	}
	return false, nil
}

// RouteExists checks if a Route matching the matching and path of
// the parameters is already configured in the Gateway
// This is done by checking the Key (method+path) in the Gateway.Routes property
func (g *Gateway) RouteExists(method, path string) (bool, error) {
	// get id
	var id string
	if id := formateID(method, path); id == "" {
		return false, fmt.Errorf("Unable to formate ID")
	}
	return g.routeExists(id)
}

// Removes the route from the Gateway.Routes and reloads the HTTP-Server
func (g *Gateway) remove(id string) error {
	exists, err := g.routeExists(id)
	if err != nil {
		return errors.Wrap(err, "Unable to find route")
	}
	if !exists {
		return fmt.Errorf("Unable to find route")
	}
	delete(g.Routes, id)
	g.reload()
	return nil
}

// registers handles in the router
// requests will be forwarded to another service, the method does not change the handle
// therefore the same handle can be used with multiple methods
func register(router *httprouter.Router, methods []string, path string, handle httprouter.Handle) {
	for _, method := range methods {
		router.Handle(method, path, handle)
	}
}

// hot reload the Gateway's HTTP Server
// initialize a new httprouter with the routes of the Gateway
// and replace it with the router of the Gateway
func (g *Gateway) reload() {
	newRouter := httprouter.New()
	for _, item := range g.Routes {
		register(newRouter, item.Methods, item.Src, item.Chain)
	}
	g.Router = newRouter
}

// Run starts the HTTP-Server of the Gateway
func (g *Gateway) Run() {
	server := http.Server{
		Addr:    g.Addr,
		Handler: middleware.LogRequest(g.Router),
	}
	log.Fatal(server.ListenAndServe())
}

// Register a new route to the gateway
// A route defines the mapping of an downstream URI to the
// upstream service
func (g *Gateway) Register(route *Route) error {
	return nil
}
