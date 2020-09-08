package statemgt

import (
	"depoy/route"
	"depoy/upstreamclient"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
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

// AddSwitchOverToRoute adds a switchover struct to the given route
func (s *StateMgt) AddSwitchOverToRoute(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

}
