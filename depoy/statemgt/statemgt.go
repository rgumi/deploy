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

// StateMgt is the struct that serves the vue web app
// and holds the configurations of the Gateway including Routes, Backends etc.
type StateMgt struct {
	Gateway *gateway.Gateway
	Addr    string
	Box     *packr.Box
}

// NewStateMgt returns a new instance of StateMgt with given parameters
func NewStateMgt(addr string, g *gateway.Gateway) *StateMgt {
	return &StateMgt{
		Gateway: g,
		Addr:    addr,
	}
}

// Start the statemgt webserver
func (s *StateMgt) Start() {
	router := httprouter.New()

	//static files
	//router.Handle("GET", "/*file", s.GetRootFiles)
	// router.Handle("GET", "/static/", s.GetStaticFiles)

	// single page app routes
	router.Handle("GET", "/help", s.GetIndexPage)
	router.Handle("GET", "/dashboard", s.GetIndexPage)
	router.Handle("GET", "/routes", s.GetIndexPage)
	router.Handle("GET", "/home", s.GetIndexPage)

	// gateway routes
	router.Handle("GET", "/v1/routes/:name", SetupHeaders(s.GetRouteByName))
	router.Handle("GET", "/v1/routes/", SetupHeaders(s.GetAllRoutes))
	router.Handle("POST", "/v1/routes/", SetupHeaders(s.CreateRoute))
	router.Handle("PUT", "/v1/routes/:name", SetupHeaders(s.UpdateRouteByName))
	router.Handle("DELETE", "/v1/routes/:name", SetupHeaders(s.DeleteRouteByName))

	// monitoring
	router.Handle("GET", "/v1/monitoring/routes", SetupHeaders(s.GetMetrics))
	router.Handle("GET", "/v1/monitoring/backend/:id", SetupHeaders(s.GetMetricsOfBackend))
	router.Handle("GET", "/v1/monitoring/route/:name", SetupHeaders(s.GetMetricsOfRoute))
	router.Handle("GET", "/v1/monitoring/all", SetupHeaders(s.GetMetricsData))

	// etc
	router.NotFound = http.FileServer(s.Box)

	server := http.Server{
		Addr:              s.Addr,
		Handler:           middleware.LogRequest(router),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Info("Starting statemgt server")
	log.Fatal(server.ListenAndServe())
}
