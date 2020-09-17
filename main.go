package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rgumi/depoy/config"
	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/statemgt"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

const (
	// web application files
	distFilepath = "vue/dist"
)

func main() {
	var g *gateway.Gateway

	// set global config
	flag.Parse()
	log.SetLevel(log.Level(config.LogLevel))

	// init prometheus
	http.Handle(config.PromPath, promhttp.Handler())
	go http.ListenAndServe(config.PromAddr, nil)

	// read config from file if configured
	if config.ConfigFile != "" {
		newGateway := config.LoadFromFile(config.ConfigFile)
		if newGateway != nil {
			g = newGateway

		} else {
			panic(fmt.Errorf("Unable to recreate the Gateway specified in file"))

		}

	} else {
		// if no config file is configured, a new instance will be started

		g = gateway.NewGateway(":8080", 5000, 5000)
	}

	/*
		Start app

	*/
	go g.Run()

	st := statemgt.NewStateMgt(":8081", g)

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

	if config.PersistConfigOnExit && config.ConfigFile != "" {
		st.Gateway.SaveConfigToFile(config.ConfigFile)
	}
	st.Stop()
	st.Gateway.Stop()
}
