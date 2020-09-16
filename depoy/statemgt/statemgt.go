package statemgt

import (
	"context"
	"depoy/gateway"
	"depoy/middleware"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/creasty/defaults"
	"github.com/gobuffalo/packr/v2"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"gopkg.in/dealancer/validate.v2"
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
	router.Handle("DELETE", "/v1/routes/:name", SetupHeaders(s.DeleteRouteByName))
	router.Handle("GET", "/v1/routes/", SetupHeaders(s.GetAllRoutes))
	router.Handle("POST", "/v1/routes/", SetupHeaders(s.CreateRoute))
	router.Handle("PUT", "/v1/routes/:name", SetupHeaders(s.UpdateRouteByName))

	// route backends
	router.Handle("PATCH", "/v1/routes/:name", SetupHeaders(s.AddNewBackendToRoute))
	router.Handle("DELETE", "/v1/routes/:name/backends/:id", SetupHeaders(s.RemoveBackendFromRoute))

	// route switchover
	router.Handle("POST", "/v1/routes/:name/switchover", SetupHeaders(s.CreateSwitchover))
	router.Handle("GET", "/v1/routes/:name/switchover", SetupHeaders(s.GetSwitchover))
	router.Handle("DELETE", "/v1/routes/:name/switchover", SetupHeaders(s.DeleteSwitchover))

	// monitoring
	router.Handle("GET", "/v1/monitoring/routes", SetupHeaders(s.GetMetricsOfAllRoutes))
	router.Handle("GET", "/v1/monitoring/backends", SetupHeaders(s.GetMetricsOfAllBackends))
	router.Handle("GET", "/v1/monitoring/backends/:id", SetupHeaders(s.GetMetricsOfBackend))
	router.Handle("GET", "/v1/monitoring/routes/:name", SetupHeaders(s.GetMetricsOfRoute))
	router.Handle("GET", "/v1/monitoring", SetupHeaders(s.GetMetricsData))
	router.Handle("GET", "/v1/monitoring/prometheus", SetupHeaders(s.GetPromMetrics))
	router.Handle("GET", "/v1/monitoring/alerts", SetupHeaders(s.GetActiveAlerts))

	// etc
	router.NotFound = http.FileServer(s.Box)

	router.PanicHandler = func(w http.ResponseWriter, req *http.Request, err interface{}) {
		returnError(
			w, req, 500,
			fmt.Errorf("Unexpected error occured while handling the request (%v)", err.(error)),
			nil)

		return
	}

	s.server = http.Server{
		Addr:              s.Addr,
		Handler:           middleware.LogRequest(router),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Info("Starting statemgt server")

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("statemgt server listen failed with %v\n", err)
		}
		log.Debug("Successfully shutdown statemgt server")
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

/*

	Helper functions

*/

func returnError(w http.ResponseWriter, req *http.Request, errCode int, err error, details []string) {
	msg := ErrorMessage{
		Timestamp:  time.Now(),
		Message:    err.Error(),
		Details:    details,
		StatusCode: errCode,
		StatusText: http.StatusText(errCode),
	}

	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(errCode)
	w.Write(b)
}

func marshalAndReturn(w http.ResponseWriter, req *http.Request, in interface{}) {
	b, err := json.Marshal(in)

	if err != nil {
		log.Errorf(err.Error())
		returnError(w, req, 500, err, nil)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func readBodyAndUnmarshal(req *http.Request, out interface{}) error {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}

	defaults.Set(out)

	if err := json.Unmarshal(b, out); err != nil {
		return err
	}
	err = validate.Validate(out)
	if err != nil {
		return err
	}

	return nil
}
