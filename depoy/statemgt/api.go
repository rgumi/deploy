package statemgt

import (
	"depoy/route"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

type Dataset struct {
	ID     int
	Value  string
	From   string
	To     string
	Status string
}

var datasets []Dataset

func GetFavicon(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, "public/favicon.ico")
}

func GetTestDataset(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	datasets = nil
	for i := 0; i < 10; i++ {
		ds := Dataset{
			ID:     rand.Intn(100),
			Value:  "HelloWorldRouting",
			From:   "/hello",
			To:     "localhost:8080",
			Status: "running",
		}
		datasets = append(datasets, ds)
	}

	for i := 10; i < 20; i++ {
		ds := Dataset{
			ID:     rand.Intn(100),
			Value:  "HelloWorldRouting",
			From:   "/hello/world",
			To:     "http://qde9dp.de.telekom.de:8080",
			Status: "idle",
		}
		datasets = append(datasets, ds)
	}
	fmt.Println("Received TestDataset Request")

	time.Sleep(20000 * time.Millisecond)
	fmt.Println("Finished Sleeping")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(datasets)
}

func GetIndexPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, "./public/")
}

func SetupHeaders(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h(w, r, ps)
	}
}

func NotFound(w http.ResponseWriter, r *http.Request) {

	// check if call was directly to the api
	if r.URL.Path[:3] != "/v1" {
		//if no direct request to the api then return singlepage app
		http.ServeFile(w, r, "./public/index.html")
		return
	}
	// if reqeusts is a direct call to the api return 404
	w.WriteHeader(404)
}

func (s *StateMgt) GetRouteByID(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id, err := strconv.ParseUint(ps.ByName("id"), 10, 32)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "id is invalid", 400)
		return
	}
	route := s.Gateway.GetRouteByID(uint32(id))
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

func (s *StateMgt) CreateRoute(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	w.WriteHeader(200)
}

func (s *StateMgt) UpdateRouteByID(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id, err := strconv.ParseUint(ps.ByName("id"), 10, 32)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "id is invalid", 400)
		return
	}

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
	newRoute, err := route.New(
		incRoute.Prefix, incRoute.Rewrite, incRoute.Host, incRoute.Methods,
		route.NewRoundRobin(2))

	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable toreplace route", 400)
		return
	}

	newRoute.AddBackend(route.NewBackend("Test1", "http://localhost:7070"))
	newRoute.AddBackend(route.NewBackend("Test2", "http://localhost:9090"))
	if r := s.Gateway.RemoveRoute(uint32(id)); r == nil {
		w.WriteHeader(200)
		return
	}

	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable toreplace route", 400)
		return
	}
	w.WriteHeader(200)
}

func (s *StateMgt) DeleteRouteByID(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id, err := strconv.ParseUint(ps.ByName("id"), 10, 32)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "id is invalid", 400)
		return
	}
	route := s.Gateway.RemoveRoute(uint32(id))
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
