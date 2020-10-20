package statemgt

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp"

	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/middleware"
	"github.com/rgumi/depoy/router"
	log "github.com/sirupsen/logrus"

	"github.com/creasty/defaults"
	"github.com/gobuffalo/packr/v2"
	"gopkg.in/dealancer/validate.v2"
)

var (
	Prefix, Addr, PromPath, PromAddr                          string
	IdleTimeout, ReadTimeout, WriteTimeout, ReadHeaderTimeout time.Duration
	ServerName                                                = "Depoy"
)

func init() {
	flag.StringVar(&Prefix, "statemgt.prefix", "/", "prefix for all requests")
	flag.StringVar(&Addr, "statemgt.addr", ":8081", "The address that the statemgt listens on")
	flag.StringVar(&PromAddr, "statemgt.promaddr", ":8090", "The address that exposes prometheus metrics")
	flag.StringVar(&PromPath, "statemgt.prompath", "/metrics", "path on which Prometheus metrics are served")
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
	server  *fasthttp.Server
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

func (s *StateMgt) Start() {
	router := router.NewRouter()

	go func() {
		http.Handle(PromPath, promhttp.Handler())
		http.ListenAndServe(PromAddr, nil)
	}()
	if s.Prefix != "/" {
		router.Handle("GET", "/healthz", s.HealthzHandler) // Kubernetes static healthcheck path
	}
	router.Handle("GET", s.Prefix+"v1/", func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("application/json")
		ctx.SetStatusCode(200)
		ctx.SetBody([]byte("Depoy API v1 that enabled access to the Gateway"))
	})
	router.Handle("GET", s.Prefix+"healthz", s.HealthzHandler)

	// webpage
	router.Handle("GET", s.Prefix+"", middleware.LogRequest(serveFiles(s.Box, s.Prefix)))

	// Config
	router.Handle("GET", s.Prefix+"v1/config", middleware.LogRequest(s.GetCurrentConfig))
	router.Handle("POST", s.Prefix+"v1/config", middleware.LogRequest(s.SetCurrentConfig))

	// gateway routes
	router.Handle("GET", s.Prefix+"v1/routes", middleware.LogRequest(s.GetRouteByName))
	router.Handle("DELETE", s.Prefix+"v1/routes", middleware.LogRequest(s.DeleteRouteByName))
	router.Handle("GET", s.Prefix+"v1/routes", middleware.LogRequest(s.GetAllRoutes))
	router.Handle("POST", s.Prefix+"v1/routes", middleware.LogRequest(s.CreateRoute))
	router.Handle("PUT", s.Prefix+"v1/routes", middleware.LogRequest(s.UpdateRouteByName))

	// route backends
	router.Handle("PATCH", s.Prefix+"v1/routes/backends", middleware.LogRequest(s.AddNewBackendToRoute))
	router.Handle("DELETE", s.Prefix+"v1/routes/backends", middleware.LogRequest(s.RemoveBackendFromRoute))

	// route switchover
	router.Handle("POST", s.Prefix+"v1/routes/switchover", middleware.LogRequest(s.CreateSwitchover))
	router.Handle("GET", s.Prefix+"v1/routes/switchover", middleware.LogRequest(s.GetSwitchover))
	router.Handle("DELETE", s.Prefix+"v1/routes/switchover", middleware.LogRequest(s.DeleteSwitchover))

	// monitoring
	router.Handle("GET", s.Prefix+"v1/monitoring", middleware.LogRequest(s.GetMetricsData))
	router.Handle("GET", s.Prefix+"v1/monitoring/routes", middleware.LogRequest(s.GetMetricsOfAllRoutes))
	router.Handle("GET", s.Prefix+"v1/monitoring/backends", middleware.LogRequest(s.GetMetricsOfBackend))
	router.Handle("GET", s.Prefix+"v1/monitoring/routes", middleware.LogRequest(s.GetMetricsOfRoute))
	router.Handle("GET", s.Prefix+"v1/monitoring/prometheus", middleware.LogRequest(s.GetPromMetrics))
	router.Handle("GET", s.Prefix+"v1/monitoring/alerts", middleware.LogRequest(s.GetActiveAlerts))

	if err := updateBaseUrl(s.Box, s.Prefix); err != nil {
		log.Fatal(err)
	}

	s.server = &fasthttp.Server{
		Handler:                       router.ServeHTTP,
		Name:                          ServerName,
		Concurrency:                   256 * 1024,
		DisableKeepalive:              false,
		ReadTimeout:                   ReadTimeout,
		WriteTimeout:                  WriteTimeout,
		IdleTimeout:                   IdleTimeout,
		MaxConnsPerIP:                 0,
		MaxRequestsPerConn:            0,
		TCPKeepalive:                  false,
		DisableHeaderNamesNormalizing: false,
		NoDefaultServerHeader:         false,
	}

	go func() {
		if err := s.server.ListenAndServe(s.Addr); err != nil {
			log.Fatalf("statemgt server listen failed with %v\n", err)
		}
		log.Debug("Successfully shutdown statemgt server")
	}()
}

func (s *StateMgt) Stop() {
	if err := s.server.Shutdown(); err != nil {
		log.Fatalf("statemgt server shutdown failed: %v\n", err)
	}
}

/*

	Helper functions

*/

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

func returnError(ctx *fasthttp.RequestCtx, errCode int, err error, details []string) {
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
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(errCode)
	ctx.SetBody(b)
}

func marshalAndReturn(ctx *fasthttp.RequestCtx, in interface{}) {
	b, err := json.Marshal(in)

	if err != nil {
		log.Errorf(err.Error())
		returnError(ctx, 500, err, nil)
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(200)
	ctx.SetBody(b)
}

func readBodyAndUnmarshal(ctx *fasthttp.RequestCtx, out interface{}) error {
	var err error
	defaults.Set(out)
	if err = json.Unmarshal(ctx.Request.Body(), out); err != nil {
		return err
	}
	err = validate.Validate(out)
	if err != nil {
		return err
	}
	return nil
}

func serveFiles(fs http.FileSystem, prefix string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.RequestURI())
		// remove uri prefix from filesystem path
		if prefix != "" {
			path = strings.Replace(path, prefix, "", 1)
		}
		// rewrite to index.html
		if path == "" || path == "/" {
			path = "index.html"
		}

	open:
		file, err := fs.Open(path)
		if err != nil {
			// not found
			path = "index.html"
			goto open
		}

		fileInfo, _ := file.Stat()
		// do not return dirs
		if fileInfo.IsDir() {
			ctx.SetStatusCode(404)
			return
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			ctx.SetStatusCode(500)
			return
		}
		mimeType := mime.TypeByExtension(filepath.Ext(path))
		ctx.Response.Header.Set("Content-Type", mimeType)
		ctx.Response.Header.Set("Accept-Ranges", "bytes")
		ctx.SetStatusCode(200)
		ctx.SetBody(content)
	}
}
