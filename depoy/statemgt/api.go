package statemgt

import (
	"depoy/route"
	"depoy/upstreamclient"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

func (s *StateMgt) GetStaticFiles(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fileServer := http.FileServer(s.Box)

	fileServer.ServeHTTP(w, r)
}

func (s *StateMgt) GetFavicon(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(200)
	html, err := s.Box.Find("favicon.ico")
	if err != nil {
		http.Error(w, "", 500)
	}
	w.Write(html)
}

func (s *StateMgt) GetIndexPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	w.WriteHeader(200)
	html, err := s.Box.Find("index.html")
	if err != nil {
		http.Error(w, "", 500)
	}
	w.Write(html)
}

func (s *StateMgt) NotFound(w http.ResponseWriter, r *http.Request) {

	// check if call was directly to the api
	if r.URL.Path[:3] != "/v1" {
		//if no direct request to the api then return singlepage app
		s.GetIndexPage(w, r, nil)
		return
	}
	// if reqeusts is a direct call to the api return 404
	w.WriteHeader(404)
}

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
	newRoute, err := route.New(
		incRoute.Name, incRoute.Prefix, incRoute.Rewrite, incRoute.Host, incRoute.Methods,
		upstreamclient.NewDefaultClient())

	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable toreplace route", 400)
		return
	}

	newRoute.AddBackend("Test1", "http://localhost:7070", "", nil, 25)
	newRoute.AddBackend("Test2", "http://localhost:9090", "", nil, 75)

	if r := s.Gateway.RemoveRoute(name); r == nil {
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

func (s *StateMgt) GetMetrics(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// default is 10 seconds
	var timeframe = 10
	var err error

	queryValues := req.URL.Query()
	queryTimeframe := queryValues.Get("timeframe")
	if queryTimeframe != "" {
		timeframe, err = strconv.Atoi(queryTimeframe)
		if err != nil {
			log.Errorf(err.Error())
			http.Error(w, "Parameter timeframe must be int", 400)
			return
		}
	}

	b, err := json.Marshal(s.Gateway.MetricsRepo.Storage.ReadAll(time.Now().Add(
		time.Duration(-timeframe)*time.Second), time.Now()))
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func (s *StateMgt) GetMetricsData(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	data := s.Gateway.MetricsRepo.Storage.ReadData()

	b, err := json.Marshal(data)
	if err != nil {
		log.Errorf(err.Error())
		http.Error(w, "Unable to marshal route", 500)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

/*

	Helper functions

*/
func SetupHeaders(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Methods, Content-Type")

		h(w, r, ps)
	}
}
