package statemgt

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

func (s *StateMgt) GetStaticFiles(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	log.Info("Returning static files")

	fileServer := http.FileServer(s.Box)
	fileServer.ServeHTTP(w, r)
}

func (s *StateMgt) GetIndexPage(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(200)
	html, err := s.Box.Find("index.html")
	if err != nil {
		http.Error(w, "", 500)
	}
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
	cfg, err := s.Gateway.ReadConfig()
	if err != nil {
		returnError(w, r, 500, err, nil)
		return
	}
	w.Header().Set("Content-Type", "text/yaml")
	w.WriteHeader(200)
	w.Write(cfg)
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
