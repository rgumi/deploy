package statemgt

import (
	"depoy/conditional"
	"depoy/route"
	"depoy/upstreamclient"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/creasty/defaults"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"gopkg.in/dealancer/validate.v2"
)

/*
	Routes
*/

// RouteRequest is a wrapper for the actual Route struct
// it replaces the map[uuid.UUID]*Backend with an array and
// avoids the required uuid.UUID when registering a new backend/route
type RouteRequest struct {
	route.Route
	NewBackends []*route.Backend `json:"backends"`
}

// GetRouteByName returns the route with given name
func (s *StateMgt) GetRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	name := ps.ByName("name")
	route := s.Gateway.GetRouteByName(name)
	if route == nil {
		w.WriteHeader(404)
		return
	}
	bJson, err := json.Marshal(route)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(bJson)
}

// GetAllRoutes returns all defined routes of the Gateway
func (s *StateMgt) GetAllRoutes(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	routes := s.Gateway.GetRoutes()
	if routes == nil {
		w.WriteHeader(404)
		return
	}
	bJson, err := json.Marshal(routes)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(bJson)
}

// CreateRoute creates a new Route. If route already exist, error
func (s *StateMgt) CreateRoute(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	myRoute := new(RouteRequest)

	if err := readBodyAndUnmarshal(req, myRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	newRoute, err := route.New(
		myRoute.Name,
		myRoute.Prefix,
		myRoute.Rewrite,
		myRoute.Host,
		myRoute.Methods,
		upstreamclient.NewDefaultClient(),
	)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	for _, myBackend := range myRoute.NewBackends {

		for _, cond := range myBackend.Metricthresholds {
			cond.IsTrue = cond.Compile()
		}

		newRoute.AddBackend(
			myBackend.Name, myBackend.Addr, myBackend.Scrapeurl, myBackend.Healthcheckurl,
			myBackend.Scrapemetrics, myBackend.Metricthresholds, myBackend.Weigth,
		)

	}

	newRoute.Reload()
	marshalAndReturn(w, req, s.Gateway.Routes[newRoute.Name])
}

// DeleteRouteByName removed the given route and all its backends
func (s *StateMgt) DeleteRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	name := ps.ByName("name")
	route := s.Gateway.RemoveRoute(name)
	if route == nil {
		w.WriteHeader(404)
		return
	}
	marshalAndReturn(w, req, route)
}

// UpdateRouteByName removed route and replaces it with new route
func (s *StateMgt) UpdateRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	myRoute := new(RouteRequest)
	routeName := ps.ByName("name")

	if err := readBodyAndUnmarshal(req, myRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	// Both routes need to have the same name otherwise the old one cant be replaced
	// CreateRoute should be used when creating a new Route
	if myRoute.Name != routeName {
		returnError(w, req, 400, fmt.Errorf("Names must be equal. Otherwise they cant be replaced"), nil)
		return
	}

	newRoute, err := route.New(
		myRoute.Name,
		myRoute.Prefix,
		myRoute.Rewrite,
		myRoute.Host,
		myRoute.Methods,
		upstreamclient.NewDefaultClient(),
	)

	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	// remove first
	s.Gateway.RemoveRoute(newRoute.Name)

	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	for _, myBackend := range myRoute.NewBackends {

		for _, cond := range myBackend.Metricthresholds {
			cond.IsTrue = cond.Compile()
		}

		newRoute.AddBackend(
			myBackend.Name, myBackend.Addr, myBackend.Scrapeurl, myBackend.Healthcheckurl,
			myBackend.Scrapemetrics, myBackend.Metricthresholds, myBackend.Weigth,
		)

	}

	newRoute.Reload()
	marshalAndReturn(w, req, s.Gateway.Routes[newRoute.Name])
}

/*
	Backends
*/

// AddNewBackendToRoute adds a new backend to the defined route
func (s *StateMgt) AddNewBackendToRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	myBackend := new(route.Backend)

	routeName := ps.ByName("name")

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(w, req, 404, fmt.Errorf("Could not find route"), nil)
		return
	}

	if err := readBodyAndUnmarshal(req, myBackend); err != nil {
		returnError(w, req, 400, err, nil)
	}

	for _, cond := range myBackend.Metricthresholds {
		cond.IsTrue = cond.Compile()
	}

	route.AddBackend(
		myBackend.Name, myBackend.Addr, myBackend.Scrapeurl, myBackend.Healthcheckurl,
		myBackend.Scrapemetrics, myBackend.Metricthresholds, myBackend.Weigth,
	)
	route.Reload()

	log.Debug("Sucessfully updated route")
	marshalAndReturn(w, req, route)
}

// RemoveBackendFromRoute remoes a backend from the defined route
// if route is not found, returns 404
func (s *StateMgt) RemoveBackendFromRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")
	id := ps.ByName("id")

	backendID, err := uuid.Parse(id)
	if err != nil {
		returnError(w, req, 400, fmt.Errorf("Invalid uuid for backendID"), nil)
		return
	}

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(w, req, 404, fmt.Errorf("Could not find route"), nil)
		return
	}
	route.RemoveBackend(backendID)

	marshalAndReturn(w, req, route)
}

/*
	Switchover
*/

// SwitchOverRequest is required to add a switchover to a route
// it is a wrapper for the actual SwitchOver struct and replaces
// the actual backends (from and to) with their corrosponding ids
type SwitchOverRequest struct {
	From         uuid.UUID                `json:"from" validate:"empty=false"`
	To           uuid.UUID                `json:"to" validate:"empty=false"`
	Conditions   []*conditional.Condition `json:"conditions" validate:"empty=false"`
	Timeout      int                      `json:"timeout" default:"2m0s"`
	WeightChange uint8                    `json:"weight_change" default:"5"`
}

// CreateSwitchover adds a switchover struct to the given route
func (s *StateMgt) CreateSwitchover(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	mySwitchOver := new(SwitchOverRequest)
	defaults.Set(mySwitchOver)

	routeName := ps.ByName("name")

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(w, req, 404, fmt.Errorf("Could not find route"), nil)
		return
	}

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		returnError(w, req, 500, err, nil)
		return
	}

	if err := json.Unmarshal(b, mySwitchOver); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	err = validate.Validate(mySwitchOver)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	timeout := time.Duration(mySwitchOver.Timeout) * time.Second

	newSwitchover, err := route.StartSwitchOver(
		mySwitchOver.From,
		mySwitchOver.To,
		mySwitchOver.Conditions,
		timeout,
		mySwitchOver.WeightChange,
	)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	marshalAndReturn(w, req, newSwitchover)

}

// GetSwitchover returns the state of the current switchover of the given route
func (s *StateMgt) GetSwitchover(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(w, req, 404, fmt.Errorf("Could not find route"), nil)
		return
	}

	if route.SwitchOver == nil {
		returnError(w, req, 404, fmt.Errorf("Route does not have a swtichover active"), nil)
		return
	}
	marshalAndReturn(w, req, route.SwitchOver)
}

// DeleteSwitchover stops and removes the switchover of the given route
// if no switchover is active, 404 is returned
func (s *StateMgt) DeleteSwitchover(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(w, req, 404, fmt.Errorf("Could not find route"), nil)
		return
	}

	if route.SwitchOver == nil {
		returnError(w, req, 404, fmt.Errorf("Route does not have a swtichover active"), nil)
		return
	}
	route.RemoveSwitchOver()
	w.WriteHeader(200)
}
