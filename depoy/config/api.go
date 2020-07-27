package config

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Dataset struct {
	ID    int ``
	Value string
}

var datasets []Dataset

func init() {
	for i := 0; i < 10; i++ {
		ds := Dataset{
			ID:    i,
			Value: "Hello World",
		}
		datasets = append(datasets, ds)
	}
}

func GetFavicon(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, "public/favicon.ico")
}

func GetTestDataset(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(datasets)
}

func GetIndexPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Println("Serving index")
	fmt.Print("path: " + r.URL.Path[:1])
	http.ServeFile(w, r, "./public/"+r.URL.Path[1:])
}

func SetupHeaders(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h(w, r, ps)
	}
}

func ServeStaticFiles(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Print("path: " + r.URL.Path[:1])
	http.ServeFile(w, r, "./public/"+r.URL.Path[1:])
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
