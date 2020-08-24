package main

import (
	"depoy/gateway"
	"depoy/route"
	"depoy/statemgt"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.DebugLevel)

	// load config

	// init gateway
	g := gateway.NewGateway(":8080")

	// start service

	r, _ := route.New(
		"/test",
		"/",
		"*",
		[]string{"GET", "POST"},
		route.NewRoundRobin(2),
	)

	r1 := route.NewTarget("Test1", "http://localhost:7070")
	r2 := route.NewTarget("Test2", "http://localhost:9090")
	r.AddTarget(r1)
	r.AddTarget(r2)

	if err := g.RegisterRoute(r); err != nil {
		panic(err)
	}

	st := statemgt.NewStateMgt(":8081", g)
	st.Start()
}
