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

	w.WriteHeader(200)
}

// UpdateRouteByName removed route and replaces it with new route
func (s *StateMgt) UpdateRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	name := ps.ByName("name")

	incRoute := new(route.Route)
	bJSON, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to read body", 400)
		return
	}

	req.Body.Close()

	if err = json.Unmarshal(bJSON, incRoute); err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to unmarshal body into struct", 400)
		return
	}
	log.Debugf("Successfully unmarshaled json into %v", incRoute)

	newRoute, err := route.New(
		incRoute.Name, incRoute.Prefix, incRoute.Rewrite, incRoute.Host, incRoute.Methods,
		upstreamclient.NewDefaultClient())

	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable toreplace route", 400)
		return
	}

	log.Debugf("Successfully created new Route from Reuqest: %v", newRoute)

	if r := s.Gateway.RemoveRoute(name); r == nil {
		w.WriteHeader(404)
		return
	}

	newRoute.AddBackend("Test1", "http://localhost:7070", "", "", nil, nil, 25)
	newRoute.AddBackend("Test2", "http://localhost:9090", "", "", nil, nil, 75)

	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to replace route", 400)
		return
	}

	log.Debug("Sucessfully updated route")
	w.WriteHeader(200)
}

// DeleteRouteByName removed the given route and all its backends
func (s *StateMgt) DeleteRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	name := ps.ByName("name")
	route := s.Gateway.RemoveRoute(name)
	if route == nil {
		w.WriteHeader(200)
		return
	}
	bJSON, err := json.Marshal(route)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(bJSON)
}

// RemoveSwitchOverOfRoute stops the switchover and removed it from the route
// takes in new weights for from- and to-Backends
func (s *StateMgt) RemoveSwitchOverOfRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

}

type SwitchOverRequest struct {
	From         uuid.UUID                `json:"from" validate:"empty=false"`
	To           uuid.UUID                `json:"to" validate:"empty=false"`
	Conditions   []*conditional.Condition `json:"conditions" validate:"empty=false"`
	Timeout      int                      `json:"timeout" default:"2m0s"`
	WeightChange uint8                    `json:"weight_change" default:"5"`
}

// AddSwitchOverToRoute adds a switchover struct to the given route
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

func marshalAndReturn(w http.ResponseWriter, req *http.Request, in interface{}) {
	b, err := json.Marshal(in)

	if err != nil {
		log.Errorf(err.Error())
		returnError(w, req, 500, err, nil)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}
