package statemgt

import (
	"depoy/gateway"
	"depoy/middleware"
	"net/http"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

type StateMgt struct {
	Gateway *gateway.Gateway
	Addr    string
}

func NewStateMgt(addr string, g *gateway.Gateway) *StateMgt {
	return &StateMgt{
		Gateway: g,
		Addr:    addr,
	}
}

func (s *StateMgt) Start() {
	router := httprouter.New()

	//static files
	router.Handle("GET", "/favicon.ico", GetFavicon)
	router.ServeFiles("/static/*filepath", http.Dir("public/"))
	router.Handle("GET", "/", GetIndexPage)

	// testing to-be-removed
	router.Handle("GET", "/v1/info", SetupHeaders(GetTestDataset))

	// gateway routes
	router.Handle("GET", "/v1/routes/:name", SetupHeaders(s.GetRouteByName))
	router.Handle("GET", "/v1/routes/", SetupHeaders(s.GetAllRoutes))
	router.Handle("POST", "/v1/routes/", SetupHeaders(s.CreateRoute))
	router.Handle("PUT", "/v1/routes/:name", SetupHeaders(s.UpdateRouteByName))
	router.Handle("DELETE", "/v1/routes/:name", SetupHeaders(s.DeleteRouteByName))

	// etc
	router.NotFound = http.HandlerFunc(NotFound)
	server := http.Server{
		Addr:    s.Addr,
		Handler: middleware.LogRequest(router),
	}

	// setup complete. Start servers
	log.Info("Starting gateway")
	go s.Gateway.Run()

	log.Info("Starting statemgt server")
	log.Fatal(server.ListenAndServe())
}
