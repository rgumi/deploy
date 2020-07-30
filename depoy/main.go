package main

import (
	"depoy/configserver"
	"depoy/middleware"

	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	router := httprouter.New()

	router.Handle("GET", "/v1/info", configserver.SetupHeaders(configserver.GetTestDataset))
	router.Handle("GET", "/", configserver.GetIndexPage)
	router.Handle("GET", "/favicon.ico", configserver.GetFavicon)
	router.ServeFiles("/static/*filepath", http.Dir("public/static"))

	router.Handle("GET", "/v1/routes/:id", configserver.SetupHeaders(configserver.GetRoute))
	router.NotFound = http.HandlerFunc(configserver.NotFound)

	router.HandleMethodNotAllowed = false
	log.Fatal(http.ListenAndServe("0.0.0.0:9090", middleware.LogRequest(router)))
}
