package statemgt

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/middleware"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"

	"github.com/creasty/defaults"
	"github.com/gobuffalo/packr/v2"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/dealancer/validate.v2"
)

var (
	IdleTimeout       = time.Duration(*flag.Int("statemgt.idleTimeout", 30, "idle timeout of connections in seconds")) * time.Second
	ReadTimeout       = time.Duration(*flag.Int("statemgt.readTimeout", 5, "read timeout of connections in seconds")) * time.Second
	WriteTimeout      = time.Duration(*flag.Int("statemgt.writeTimeout", 5, "write timeout of connections in seconds")) * time.Second
	ReadHeaderTimeout = time.Duration(*flag.Int("statemgt.readHeaderTimeout", 2, "read header timeout of connections in seconds")) * time.Second
	PromPath          = flag.String("statemgt.prompath", "/metrics", "path on which Prometheus metrics are served")
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

	router.Handle("GET", *PromPath, Metrics(promhttp.Handler()))
	router.Handle("GET", "/healthz", s.HealthzHandler)

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
	router.Handle("GET", "/v1/routes/:name", s.GetRouteByName)
	router.Handle("DELETE", "/v1/routes/:name", s.DeleteRouteByName)
	router.Handle("GET", "/v1/routes/", s.GetAllRoutes)
	router.Handle("POST", "/v1/routes/", s.CreateRoute)
	router.Handle("PUT", "/v1/routes/:name", s.UpdateRouteByName)

	// route backends
	router.Handle("PATCH", "/v1/routes/:name", s.AddNewBackendToRoute)
	router.Handle("DELETE", "/v1/routes/:name/backends/:id", s.RemoveBackendFromRoute)

	// route switchover
	router.Handle("POST", "/v1/routes/:name/switchover", s.CreateSwitchover)
	router.Handle("GET", "/v1/routes/:name/switchover", s.GetSwitchover)
	router.Handle("DELETE", "/v1/routes/:name/switchover", s.DeleteSwitchover)

	// monitoring
	router.Handle("GET", "/v1/monitoring/routes/", s.GetMetricsOfAllRoutes)
	router.Handle("GET", "/v1/monitoring/backends/", s.GetMetricsOfAllBackends)
	router.Handle("GET", "/v1/monitoring/backends/:id", s.GetMetricsOfBackend)
	router.Handle("GET", "/v1/monitoring/routes/:name", s.GetMetricsOfRoute)
	router.Handle("GET", "/v1/monitoring", s.GetMetricsData)
	router.Handle("GET", "/v1/monitoring/prometheus", s.GetPromMetrics)
	router.Handle("GET", "/v1/monitoring/alerts", s.GetActiveAlerts)

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
		Handler:           middleware.LogRequest(cors.Default().Handler(router)),
		IdleTimeout:       IdleTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
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
func Metrics(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		h.ServeHTTP(w, req)
	}
}

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
