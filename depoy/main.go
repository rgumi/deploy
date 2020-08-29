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
	"UpstreamResponseTime": 100,
	"6xxResponse":          0,
	"5xxResponse":          0.1,
}

func main() {
	log.SetLevel(log.WarnLevel)

	// load config

	// init gateway
	g := gateway.NewGateway(":8080")

	// start service

	r, _ := route.New(
		"TestRoute",
		"/test",
		"/",
		"*",
		[]string{"GET", "POST"},
		upstreamclient.NewClient(),
	)

	r.AddBackend("Test1", "http://localhost:7070", "", DefaultMetricThreshholds, 75)
	r.AddBackend("Test2", "http://localhost:9090", "", DefaultMetricThreshholds, 25)
	r.AddBackend("Test3", "http://localhost:7070", "", DefaultMetricThreshholds, 75)

	if err := g.RegisterRoute(r); err != nil {
		panic(err)
	}

	st := statemgt.NewStateMgt(":8081", g)
	st.Start()
}
