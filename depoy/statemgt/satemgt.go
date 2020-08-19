package statemgt

import (
	"depoy/gateway"
	"depoy/middleware"
	"net/http"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

func Start(addr, prefix string, g *gateway.Gateway) {
	router := httprouter.New()

	router.Handle("GET", prefix+"/v1/info", SetupHeaders(GetTestDataset))
	router.Handle("GET", prefix+"/", GetIndexPage)
	router.Handle("GET", prefix+"/favicon.ico", GetFavicon)
	router.ServeFiles(prefix+"/static/*filepath", http.Dir("public/static"))

	router.Handle("GET", prefix+"/v1/routes/:id", SetupHeaders(GetRoute))

	router.NotFound = http.HandlerFunc(NotFound)
	router.HandleMethodNotAllowed = false

	server := http.Server{
		Addr:    addr,
		Handler: middleware.LogRequest(router),
	}
	log.Debug("Starting gateway")
	go g.Run()

	log.Debug("Starting statemgt server")
	log.Fatal(server.ListenAndServe())
}
