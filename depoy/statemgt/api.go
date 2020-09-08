package statemgt

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
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
