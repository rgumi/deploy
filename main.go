package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rgumi/depoy/config"
	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/statemgt"

	packr "github.com/gobuffalo/packr/v2"

	log "github.com/sirupsen/logrus"
)

const (
	// web application files
	distFilepath = "webapp/dist"
	signalMsg    = "Received Signal%v"
)

func main() {
	var g *gateway.Gateway

	// set global config
	flag.Parse()
	log.SetLevel(log.Level(config.LogLevel))

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

		g = gateway.NewGateway(":8080", 5*time.Second, 5*time.Second, 5*time.Second)
	}

	/*
		Start app

	*/
	go g.Run()

	st := statemgt.NewStateMgt(":8081", config.PromPath, g)

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
		log.Warnf(signalMsg, sig)

	case syscall.SIGTERM:
		log.Warnf(signalMsg, sig)

	case syscall.SIGKILL:
		log.Warnf(signalMsg, sig)
	}

	if config.PersistConfigOnExit && config.ConfigFile != "" {
		st.Gateway.SaveConfigToFile(config.ConfigFile)
	}
	st.Stop()
	st.Gateway.Stop()
}
