package main

import (
	"depoy/gateway"
	"depoy/route"
	"depoy/statemgt"
	"depoy/upstreamclient"
	"net/http"
	"time"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

// DefaultMetricThreshholds insert var with the default metric threshholds for response time, status etc.
// replace this with []Condition{} ??
var ScrapeMetrics = []string{
	"go_goroutines",
}

var DefaultMetricsThresholds = map[string]float64{
	"go_goroutines": 20,
	"ResponseTime":  1000,
	"6xxRate":       0.1,
	"5xxRate":       0.2,
	"4xxRate":       0.6,
}

func main() {
	log.SetLevel(log.DebugLevel)
	promPort := ":8084"
	// load config

	// init prometheus
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(promPort, nil)

	// init gateway
	g := gateway.NewGateway(":8080")
	// setup complete. Start servers
	log.Info("Starting gateway")
	go g.Run()

	// start service

	r, _ := route.New(
		"Route1",
		"/",
		"/",
		"*",
		[]string{"GET", "POST"},
		upstreamclient.NewDefaultClient(),
	)
	// old
	r.AddBackend("Test2", "http://localhost:9090", "http://localhost:9091/metrics",
		"http://localhost:9090/", ScrapeMetrics, DefaultMetricsThresholds, 100)

	// new
	r.AddBackend(
		"Test2", "http://localhost:7070", "http://localhost:7071/metrics",
		"http://localhost:7070/", ScrapeMetrics, DefaultMetricsThresholds, 0)

	if err := g.RegisterRoute(r); err != nil {
		panic(err)
	}

	go func() {
		// test switchover
		time.Sleep(10 * time.Second)

		cond1, _ := route.NewCondition(
			"ResponseTime",
			"<",
			1000,
			2*time.Minute,
		)

		conditionsList := []*route.Condition{
			cond1,
		}

		backendIds := []uuid.UUID{}
		for key := range r.Backends {
			backendIds = append(backendIds, key)
		}
		if route, found := g.Routes["Route1"]; found {
			route.StartSwitchOver(backendIds[0], backendIds[1], conditionsList, 10*time.Second, 5)
		}
	}()

	/*
		r2, _ := route.New(
			"Route2",
			"/test/",
			"/",
			"*",
			[]string{"GET", "POST"},
			upstreamclient.NewDefaultClient(),
		)
		r2.AddBackend("Test2", "http://ip-172-31-38-69.eu-central-1.compute.internal:9090", "", "", DefaultMetricThreshholds, 25)

		if err := g.RegisterRoute(r2); err != nil {
			panic(err)
		}

		r3, _ := route.New(
			"Route3",
			"/foo/",
			"/",
			"*",
			[]string{"GET", "POST"},
			upstreamclient.NewDefaultClient(),
		)
		r3.AddBackend("Test2", "http://localhost:9090", "", "", DefaultMetricThreshholds, 25)

		if err := g.RegisterRoute(r3); err != nil {
			panic(err)
		}

		r4, _ := route.New(
			"Route4",
			"/bar/",
			"/",
			"*",
			[]string{"GET", "POST"},
			upstreamclient.NewDefaultClient(),
		)
		r4.AddBackend("Test2", "http://localhost:9090", "", "", DefaultMetricThreshholds, 25)

		if err := g.RegisterRoute(r4); err != nil {
			panic(err)
		}
	*/

	/*
		Start app

	*/

	st := statemgt.NewStateMgt(":8081", g)

	distFilepath := "../vue/dist"
	// package static files into binary
	box := packr.New("files", distFilepath)

	st.Box = box
	st.Start()
}
