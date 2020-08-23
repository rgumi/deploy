package main

import (
	"depoy/gateway"
	"depoy/route"
	st "depoy/statemgt"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.InfoLevel)

	// load config

	// init gateway
	g := gateway.NewGateway(":8080")

	// start service

	r, _ := route.New(
		"/",
		"",
		"*",
		[]string{"GET", "POST"},
		route.NewRoundRobin(2),
	)

	r1, _ := route.NewTarget("Test1", "http://localhost:7070")
	r2, _ := route.NewTarget("Test2", "http://localhost:9090")
	r.Targets = []*route.Target{
		r1,
		r2,
	}

	if err := g.RegisterRoute(r); err != nil {
		panic(err)
	}

	st.Start(":8081", "", g)
}
