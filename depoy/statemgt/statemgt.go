package statemgt

import (
	"context"
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
	server  http.Server
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

	// single page app routes
	router.Handle("GET", "/help", s.GetIndexPage)
	router.Handle("GET", "/dashboard", s.GetIndexPage)
	router.Handle("GET", "/routes", s.GetIndexPage)
	router.Handle("GET", "/home", s.GetIndexPage)
	router.Handle("GET", "/login", s.GetIndexPage)

	// Config
	router.Handle("GET", "/v1/config", s.GetCurrentConfig)
	router.Handle("POST", "/v1/config", s.SetCurrentConfig)

	// gateway routes
	router.Handle("GET", "/v1/routes/:name", SetupHeaders(s.GetRouteByName))
	router.Handle("GET", "/v1/routes/", SetupHeaders(s.GetAllRoutes))
	router.Handle("POST", "/v1/routes/", SetupHeaders(s.CreateRoute))
	router.Handle("PUT", "/v1/routes/:name", SetupHeaders(s.UpdateRouteByName))
	router.Handle("DELETE", "/v1/routes/:name", SetupHeaders(s.DeleteRouteByName))

	// route switchover
	router.Handle("POST", "/v1/routes/:name/switchover", SetupHeaders(s.CreateSwitchover))
	router.Handle("GET", "/v1/routes/:name/switchover", SetupHeaders(s.GetSwitchover))
	router.Handle("DELETE", "/v1/routes/:name/switchover", SetupHeaders(s.DeleteSwitchover))

	// monitoring
	router.Handle("GET", "/v1/monitoring/routes", SetupHeaders(s.GetMetricsOfAllRoutes))
	router.Handle("GET", "/v1/monitoring/backends", SetupHeaders(s.GetMetricsOfAllBackends))
	router.Handle("GET", "/v1/monitoring/backend/:id", SetupHeaders(s.GetMetricsOfBackend))
	router.Handle("GET", "/v1/monitoring/route/:name", SetupHeaders(s.GetMetricsOfRoute))
	router.Handle("GET", "/v1/monitoring/all", SetupHeaders(s.GetMetricsData))
	router.Handle("GET", "/v1/monitoring/prometheus", SetupHeaders(s.GetPromMetrics))

	// etc
	router.NotFound = http.FileServer(s.Box)

	s.server = http.Server{
		Addr:              s.Addr,
		Handler:           middleware.LogRequest(router),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Info("Starting statemgt server")

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("statemgt server listen failed with %v\n", err)
		}
	}()
}

func (s *StateMgt) Stop() {
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := s.server.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("statemgt server shutdown failed: %v\n", err)
	}
}
