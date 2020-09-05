package statemgt

import (
	"depoy/gateway"
	"depoy/middleware"
	"net/http"
	"time"

	"github.com/gobuffalo/packr/v2"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

type StateMgt struct {
	Gateway *gateway.Gateway
	Addr    string
	Box     *packr.Box
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
	router.Handle("GET", "/", s.GetIndexPage)
	router.Handle("GET", "/favicon.ico", s.GetFavicon)
	router.Handle("GET", "/static/*filepath", s.GetStaticFiles)

	// gateway routes
	router.Handle("GET", "/v1/routes/:name", SetupHeaders(s.GetRouteByName))
	router.Handle("GET", "/v1/routes/", SetupHeaders(s.GetAllRoutes))
	router.Handle("POST", "/v1/routes/", SetupHeaders(s.CreateRoute))
	router.Handle("PUT", "/v1/routes/:name", SetupHeaders(s.UpdateRouteByName))
	router.Handle("DELETE", "/v1/routes/:name", SetupHeaders(s.DeleteRouteByName))

	// monitoring
	router.Handle("GET", "/v1/monitoring", SetupHeaders(s.GetMetrics))
	router.Handle("GET", "/v1/monitoring/all", SetupHeaders(s.GetMetricsData))

	// etc
	router.NotFound = http.HandlerFunc(s.NotFound)
	server := http.Server{
		Addr:              s.Addr,
		Handler:           middleware.LogRequest(router),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	// setup complete. Start servers
	log.Info("Starting gateway")
	go s.Gateway.Run()

	log.Info("Starting statemgt server")
	log.Fatal(server.ListenAndServe())
}
