package statemgt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/rgumi/depoy/config"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

func (s *StateMgt) GetStaticFiles(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Info("Returning static files")

	fileServer := http.FileServer(s.Box)
	fileServer.ServeHTTP(w, r)
}

func (s *StateMgt) GetIndexPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	html, err := s.Box.Find("index.html")
	if err != nil {
		log.Error(err)
		returnError(w, r, 500, err, nil)
		return
	}
	w.WriteHeader(200)
	w.Write(html)
}

func (s *StateMgt) GetRootFiles(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Info("Returning root files")

	fileServer := http.FileServer(s.Box)
	fileServer.ServeHTTP(w, r)
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

func (s *StateMgt) GetCurrentConfig(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	cfg, err := s.Gateway.ReadConfig()
	if err != nil {
		returnError(w, r, 500, err, nil)
		return
	}
	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(200)
	w.Write(cfg)
}

func (s *StateMgt) SetCurrentConfig(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	// Read input as YAML
	// Parse to Gateway
	// Remove old Gateway
	// Start new Gateway

	if r.Header.Get("Content-Type") != "application/json" {
		returnError(w, r, 400, fmt.Errorf("Content-Type application/json is required"), nil)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		returnError(w, r, 400, err, nil)
		return
	}

	newGateway, err := config.ParseFromBinary(json.Unmarshal, b)
	if err != nil {
		log.Error(err)
		returnError(w, r, 400, err, nil)
		return
	}

	w.WriteHeader(201)

	go func() {
		s.Gateway.Stop()

		// wait for old server to shutdown
		time.Sleep(2 * time.Second)

		s.Gateway = newGateway

		go s.Gateway.Run()
	}()
}
