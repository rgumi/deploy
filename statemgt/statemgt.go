package statemgt

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
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
	Prefix, Addr, PromPath                                    string
	IdleTimeout, ReadTimeout, WriteTimeout, ReadHeaderTimeout time.Duration
)

func init() {
	flag.StringVar(&Prefix, "statemgt.prefix", "/", "prefix for all requests")
	flag.StringVar(&Addr, "statemgt.addr", ":8081", "The address that the statemgt listens on")
	flag.StringVar(&PromPath, "statemgt.prompath", "metrics", "path on which Prometheus metrics are served")
	IdleTimeout = time.Duration(*flag.Int("statemgt.idleTimeout", 30, "idle timeout of connections in seconds")) * time.Second
	ReadTimeout = time.Duration(*flag.Int("statemgt.readTimeout", 5, "read timeout of connections in seconds")) * time.Second
	WriteTimeout = time.Duration(*flag.Int("statemgt.writeTimeout", 5, "write timeout of connections in seconds")) * time.Second
	ReadHeaderTimeout = time.Duration(*flag.Int("statemgt.readHeaderTimeout", 2, "read header timeout of connections in seconds")) * time.Second
}

// StateMgt is the struct that serves the vue web app
// and holds the configurations of the Gateway including Routes, Backends etc.
type StateMgt struct {
	Gateway *gateway.Gateway
	Addr    string
	Prefix  string
	server  http.Server
	Box     *packr.Box
}

// NewStateMgt returns a new instance of StateMgt with given parameters
func NewStateMgt(addr string, g *gateway.Gateway, prefix string) *StateMgt {
	return &StateMgt{
		Gateway: g,
		Addr:    addr,
		Prefix:  prefix,
	}
}

// Start the statemgt webserver
func (s *StateMgt) Start() {
	router := httprouter.New()

	router.Handle("GET", s.Prefix+PromPath, Metrics(promhttp.Handler()))
	if s.Prefix != "/" {
		router.Handle("GET", "/healthz", s.HealthzHandler)
	}
	router.Handle("GET", s.Prefix+"healthz", s.HealthzHandler)

	// single page app routes
	router.Handle("GET", s.Prefix+"help", s.GetIndexPage)
	router.Handle("GET", s.Prefix+"dashboard", s.GetIndexPage)
	router.Handle("GET", s.Prefix+"routes/*filepath", s.GetIndexPage)
	router.Handle("GET", s.Prefix+"home", s.GetIndexPage)
	router.Handle("GET", s.Prefix+"login", s.GetIndexPage)

	// Config
	router.Handle("GET", s.Prefix+"v1/config", s.GetCurrentConfig)
	router.Handle("POST", s.Prefix+"v1/config", s.SetCurrentConfig)

	// gateway routes
	router.Handle("GET", s.Prefix+"v1/routes/:name", s.GetRouteByName)
	router.Handle("DELETE", s.Prefix+"/v1routes/:name", s.DeleteRouteByName)
	router.Handle("GET", s.Prefix+"v1/routes/", s.GetAllRoutes)
	router.Handle("POST", s.Prefix+"v1/routes/", s.CreateRoute)
	router.Handle("PUT", s.Prefix+"v1/routes/:name", s.UpdateRouteByName)

	// route backends
	router.Handle("PATCH", s.Prefix+"v1/routes/:name", s.AddNewBackendToRoute)
	router.Handle("DELETE", s.Prefix+"v1/routes/:name/backends/:id", s.RemoveBackendFromRoute)

	// route switchover
	router.Handle("POST", s.Prefix+"v1/routes/:name/switchover", s.CreateSwitchover)
	router.Handle("GET", s.Prefix+"v1/routes/:name/switchover", s.GetSwitchover)
	router.Handle("DELETE", s.Prefix+"v1/routes/:name/switchover", s.DeleteSwitchover)

	// monitoring
	router.Handle("GET", s.Prefix+"v1/monitoring/routes/", s.GetMetricsOfAllRoutes)
	router.Handle("GET", s.Prefix+"v1/monitoring/backends/", s.GetMetricsOfAllBackends)
	router.Handle("GET", s.Prefix+"v1/monitoring/backends/:id", s.GetMetricsOfBackend)
	router.Handle("GET", s.Prefix+"v1/monitoring/routes/:name", s.GetMetricsOfRoute)
	router.Handle("GET", s.Prefix+"v1/monitoring", s.GetMetricsData)
	router.Handle("GET", s.Prefix+"v1/monitoring/prometheus", s.GetPromMetrics)
	router.Handle("GET", s.Prefix+"v1/monitoring/alerts", s.GetActiveAlerts)

	// etc
	if err := updateBaseUrl(s.Box, s.Prefix); err != nil {
		log.Fatal(err)
	}
	router.NotFound = &StaticFileHandler{Prefix: s.Prefix, Fileserver: http.FileServer(s.Box)}
	router.PanicHandler = func(w http.ResponseWriter, req *http.Request, e interface{}) {
		err := fmt.Errorf("Unexpected error occured while handling the request (%v)", e.(error))
		log.Error(err)
		returnError(
			w, req, 500,
			err,
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

// StaticFileHandler is used to route requests for static files
// to the FileServer after replacing the prefix of the Path
type StaticFileHandler struct {
	Prefix     string
	Fileserver http.Handler
}

func (sfh *StaticFileHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req.URL.Path = strings.Replace(req.URL.Path, sfh.Prefix, "", 1)
	sfh.Fileserver.ServeHTTP(w, req)
}

func updateBaseUrl(box *packr.Box, baseUrl string) error {
	match := "{{baseUrl}}"
	content, err := box.FindString("index.html")
	if err != nil {
		return fmt.Errorf("Unable to open file in box (%v)", err)
	}
	if strings.Contains(content, match) {
		newContent := strings.ReplaceAll(content, match, baseUrl)
		if err := box.AddString("index.html", newContent); err != nil {
			return fmt.Errorf("Unable to replace file in box (%v)", err)
		}
		log.Infof("Replaced baseUrl")
		return nil
	}
	return fmt.Errorf("Unable to replace baseUrl in file in box")
}

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
