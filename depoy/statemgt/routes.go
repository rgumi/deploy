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

	var myRoute *route.Route

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		returnError(w, req, 500, err, nil)
		return
	}

	if err := json.Unmarshal(b, myRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	err = validate.Validate(myRoute)
	if err != nil {
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
	marshalAndReturn(w, req, s.Gateway.Routes[newRoute.Name])
}

// UpdateRouteByName removed route and replaces it with new route
func (s *StateMgt) UpdateRouteByName(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var myRoute *route.Route

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		returnError(w, req, 500, err, nil)
		return
	}

	if err := json.Unmarshal(b, myRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	err = validate.Validate(myRoute)
	if err != nil {
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

	// remove first
	s.Gateway.RemoveRoute(newRoute.Name)

	// replace with new one
	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		returnError(w, req, 400, err, nil)
		return
	}
	marshalAndReturn(w, req, s.Gateway.Routes[newRoute.Name])
}

func (s *StateMgt) AddNewBackendToRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	routeName := ps.ByName("name")

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(w, req, 404, fmt.Errorf("Could not find route"), nil)
		return
	}

	//	route.AddBackend()

	log.Debug("Sucessfully updated route")
	w.WriteHeader(200)
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

/*
	Switchover
*/

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
