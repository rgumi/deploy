package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rgumi/depoy/config"
	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/metrics"
	"github.com/rgumi/depoy/statemgt"
	"github.com/rgumi/depoy/storage"
	log "github.com/sirupsen/logrus"

	"net/http"
	_ "net/http/pprof" // profiling and debugging TODO delete

	packr "github.com/gobuffalo/packr/v2"
)

const (
	// web application files
	distFilepath = "webapp/dist"
	signalMsg    = "Received Signal %v"
)

var (
	gw *gateway.Gateway
)

func main() {

	// profiling and debugging TODO delete
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	// set global config
	flag.Parse()
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.Level(config.LogLevel))
	// read config from file if configured
	if config.ConfigFile != "" {
		gw = config.LoadFromFile(config.ConfigFile)
		if gw == nil {
			log.Fatal(fmt.Errorf("Unable to recreate the Gateway specified in file"))
		}
		log.Info("Using configured Gateway")
	} else {
		// if no config file is configured, a new instance will be started
		_, newMetricsRepo := metrics.NewMetricsRepository(
			storage.NewLocalStorage(config.RetentionPeriod, config.Granulartiy),
			config.Granulartiy, config.MetricsChannelPuffersize, config.ScrapeMetricsChannelPuffersize,
		)
		gw = gateway.NewGateway(config.GatewayAddr, newMetricsRepo,
			config.ReadTimeout, config.WriteTimeout, config.IdleTimeout,
		)
	}
	go gw.Run()
	log.Warnf("Gateway listening on Addr %s", config.GatewayAddr)
	st := statemgt.NewStateMgt(statemgt.Addr, gw, statemgt.Prefix)

	// package static files into binary
	box := packr.New("files", distFilepath)
	st.Box = box

	go st.Start()
	log.Warnf("StateMgt listening on Addr %s with prefix %s", statemgt.Addr, statemgt.Prefix)

	// sys signal
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
		config.WriteToFile(st.Gateway, config.ConfigFile)
	}
	st.Stop()
	st.Gateway.Stop()
}
