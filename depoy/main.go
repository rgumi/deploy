package main

import (
	"depoy/gateway"
	"depoy/route"
	"depoy/statemgt"
	"depoy/upstreamclient"

	log "github.com/sirupsen/logrus"
)

// DefaultMetricThreshholds insert var with the default metric threshholds for response time, status etc.
var DefaultMetricThreshholds = map[string]float64{
	"ResponseTime": 1000,
	"6xxRate":      0.1,
	"5xxRate":      0.2,
	"4xxRate":      0.6,
}

func main() {
	log.SetLevel(log.DebugLevel)

	// load config

	// init gateway
	g := gateway.NewGateway(":8080")

	// start service

	r, _ := route.New(
		"Route1",
		"/",
		"/",
		"*",
		[]string{"GET", "POST"},
		upstreamclient.NewDefaultClient(),
	)

	r2, _ := route.New(
		"Route2",
		"/grafana/",
		"/grafana/",
		"*",
		[]string{"GET", "POST"},
		upstreamclient.NewDefaultClient(),
	)

	//r.AddBackend("Test1", "http://localhost:7070", "", DefaultMetricThreshholds, 75)
	r.AddBackend("Test2", "https://k8s-irrp-ewu-cc.telekom.de", "", DefaultMetricThreshholds, 25)
	r2.AddBackend("Test2", "https://k8s-irrp-ewu-cc.telekom.de", "", DefaultMetricThreshholds, 25)

	if err := g.RegisterRoute(r); err != nil {
		panic(err)
	}

	if err := g.RegisterRoute(r2); err != nil {
		panic(err)
	}

	st := statemgt.NewStateMgt(":8081", g)
	st.Start()
}
