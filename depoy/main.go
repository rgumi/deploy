package main

import (
	"depoy/config"
	"depoy/middleware"

	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	router := httprouter.New()

	router.Handle("GET", "/v1/info", config.SetupHeaders(config.GetTestDataset))
	router.Handle("GET", "/", config.GetIndexPage)
	router.Handle("GET", "/favicon.ico", config.GetFavicon)
	router.ServeFiles("/static/*filepath", http.Dir("public/static"))
	router.NotFound = http.HandlerFunc(config.NotFound)

	router.HandleMethodNotAllowed = false
	log.Fatal(http.ListenAndServe(":9090", middleware.LogRequest(router)))
}
