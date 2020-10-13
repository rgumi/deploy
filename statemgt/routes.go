package statemgt

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rgumi/depoy/config"
	"github.com/rgumi/depoy/route"
	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

/*
	Routes
*/

// GetRouteByName returns the route with given name
func (s *StateMgt) GetRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	name := ps.ByName("name")
	route := s.Gateway.GetRoute(name)
	if route == nil {
		w.WriteHeader(404)
		return
	}

	marshalAndReturn(w, req, config.ConvertRouteToInputRoute(route))
}

// GetAllRoutes returns all defined routes of the Gateway
func (s *StateMgt) GetAllRoutes(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	routes := s.Gateway.GetRoutes()
	if routes == nil {
		w.WriteHeader(404)
		return
	}
	output := make(map[string]*config.InputRoute, len(routes))
	for idx, route := range routes {
		output[idx] = config.ConvertRouteToInputRoute(route)
	}

	b, err := json.Marshal(output)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

// CreateRoute creates a new Route. If route already exist, error
func (s *StateMgt) CreateRoute(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	myRoute := config.NewInputRoute()
	if err := readBodyAndUnmarshal(req, myRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	newRoute, err := config.ConvertInputRouteToRoute(myRoute)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	if err = myRoute.Strategy.Validate(newRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	err = myRoute.Strategy.Reset(newRoute)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	newRoute.Reload()
	s.Gateway.Reload()
	marshalAndReturn(w, req, config.ConvertRouteToInputRoute(newRoute))
}

// DeleteRouteByName removed the given route and all its backends
func (s *StateMgt) DeleteRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	name := ps.ByName("name")
	route := s.Gateway.RemoveRoute(name)
	if route == nil {
		w.WriteHeader(404)
		return
	}
	marshalAndReturn(w, req, config.ConvertRouteToInputRoute(route))
}

// UpdateRouteByName removed route and replaces it with new route
func (s *StateMgt) UpdateRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	myRoute := config.NewInputRoute()
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
	newRoute, err := config.ConvertInputRouteToRoute(myRoute)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	if err = myRoute.Strategy.Validate(newRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	err = myRoute.Strategy.Reset(newRoute)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	// remove first
	s.Gateway.RemoveRoute(newRoute.Name)
	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		returnError(w, req, 500, err, nil)
		return
	}

	newRoute.Reload()
	s.Gateway.Reload()
	marshalAndReturn(w, req, config.ConvertRouteToInputRoute(newRoute))
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
		return
	}
	for _, cond := range myBackend.Metricthresholds {
		cond.Compile()
	}
	_, err := route.AddBackend(
		myBackend.Name, myBackend.Addr, myBackend.Scrapeurl, myBackend.Healthcheckurl,
		myBackend.Scrapemetrics, myBackend.Metricthresholds, myBackend.Weigth,
	)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	route.Reload()
	log.Debug("Sucessfully updated route")
	marshalAndReturn(w, req, config.ConvertRouteToInputRoute(route))
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

	marshalAndReturn(w, req, config.ConvertRouteToInputRoute(route))
}

/*
	Switchover
*/

// CreateSwitchover adds a switchover struct to the given route
func (s *StateMgt) CreateSwitchover(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	mySwitchOver := config.NewInputSwitchover()
	routeName := ps.ByName("name")
	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(w, req, 404, fmt.Errorf("Could not find route"), nil)
		return
	}
	if err := readBodyAndUnmarshal(req, mySwitchOver); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}

	newSwitchover, err := route.StartSwitchOver(
		mySwitchOver.From,
		mySwitchOver.To,
		mySwitchOver.Conditions,
		mySwitchOver.Timeout.Duration,
		mySwitchOver.AllowedFailures,
		mySwitchOver.WeightChange,
		mySwitchOver.Force,
		mySwitchOver.Rollback,
	)
	if err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	marshalAndReturn(w, req, config.ConvertSwitchoverToInputSwitchover(newSwitchover))

}

// GetSwitchover returns the state of the current switchover of the given route
func (s *StateMgt) GetSwitchover(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(w, req, 404, fmt.Errorf("Could not find route"), nil)
		return
	}

	if route.Switchover == nil {
		returnError(w, req, 404, fmt.Errorf("Route does not have a swtichover active"), nil)
		return
	}
	marshalAndReturn(w, req, config.ConvertSwitchoverToInputSwitchover(route.Switchover))
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

	if route.Switchover == nil {
		returnError(w, req, 404, fmt.Errorf("Route does not have a swtichover active"), nil)
		return
	}
	route.RemoveSwitchOver()
	w.WriteHeader(200)
}
