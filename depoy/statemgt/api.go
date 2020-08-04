package statemgt

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
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

func GetTestDataset(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

func GetIndexPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Println("Serving index")
	fmt.Println(r.URL.Path)
	http.ServeFile(w, r, "./public/")
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

func GetRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Println(ps.ByName("id"))
	i, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(datasets[i])
}
