package main

import (
	"depoy/conditional"
	"depoy/config"
	"depoy/gateway"
	"depoy/statemgt"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

var (
	PersistConfigOnExit = true
	ConfigFilename      = "gateway-config.yaml"
)

// DefaultMetricsThresholds Repalce with Condition ???
var DefaultMetricsThresholds = []*conditional.Condition{
	conditional.NewCondition("ResponseTime", ">", 1000, 10),
	conditional.NewCondition("6xxRate", ">", 0.1, 10),
	conditional.NewCondition("5xxRate", ">", 0.3, 10),
	conditional.NewCondition("4xxRate", ">", 0.6, 10),
}

func main() {
	log.SetLevel(log.WarnLevel)
	promPort := ":8084"
	promPath := "/metrics"
	var g *gateway.Gateway

	// init prometheus
	http.Handle(promPath, promhttp.Handler())
	go http.ListenAndServe(promPort, nil)

	newGateway := config.LoadFromFile(ConfigFilename)
	if newGateway != nil {
		g = newGateway
		go g.Run()

	} else {
		panic(fmt.Errorf("Unable to recreate the Gateway specified in file"))

		// g = gateway.NewGateway(":8080", 5000, 5000)
	}

	/*
		Start app

	*/

	st := statemgt.NewStateMgt(":8081", g)

	distFilepath := "../vue/dist"
	// package static files into binary
	box := packr.New("files", distFilepath)
	st.Box = box

	go st.Start()

	// sys singal
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	sig := <-signalChannel
	switch sig {
	case os.Interrupt:
		log.Warnf("Received Signal%v", sig)

	case syscall.SIGTERM:
		log.Warnf("Received Signal%v", sig)

	case syscall.SIGKILL:
		log.Warnf("Received Signal%v", sig)
	}

	if PersistConfigOnExit {
		st.Gateway.SaveConfigToFile(ConfigFilename)
	}
	st.Stop()
	st.Gateway.Stop()
}
