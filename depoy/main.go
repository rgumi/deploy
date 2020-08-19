package main

import (
	"depoy/gateway"
	_ "depoy/gateway"
	"depoy/route"
	st "depoy/statemgt"
	"depoy/upstreamclient"
	"fmt"

	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetLevel(log.InfoLevel)

	// load config

	// init gateway
	g := gateway.New(":8080")

	// start service

	r, _ := route.New(
		"/",
		"",
		[]string{"GET", "POST"},
		1,
		route.NewRoundRobin(2),
	)

	ch := make(chan upstreamclient.Metric, 20)

	r.Targets = []route.Target{
		route.Target{
			ID:         1,
			Name:       "Test1",
			Addr:       "http://localhost:7070",
			MetricChan: ch,
			Active:     true,
		},
		route.Target{
			ID:         2,
			Name:       "Test2",
			Addr:       "http://localhost:9090",
			MetricChan: ch,
			Active:     true,
		},
	}
	r.SetMetricChannel(ch)

	go func() {
		for {
			for in := range ch {
				fmt.Printf("%+v\n", in)
			}
		}
	}()

	g.Register(r)

	st.Start(":8081", "", g)
}
