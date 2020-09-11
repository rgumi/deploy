package main

import (
	"depoy/conditional"
	"depoy/gateway"
	"depoy/route"
	"depoy/statemgt"
	"depoy/upstreamclient"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/yaml"

	log "github.com/sirupsen/logrus"
)

var ScrapeMetrics = []string{
	"go_goroutines",
}

// DefaultMetricsThresholds Repalce with Condition ???
var DefaultMetricsThresholds = []*conditional.Condition{
	conditional.NewCondition("ResponseTime", ">", 1000, 10),
	conditional.NewCondition("6xxRate", ">", 0.1, 10),
	conditional.NewCondition("5xxRate", ">", 0.3, 10),
	conditional.NewCondition("4xxRate", ">", 0.6, 10),
}

func loadFromFile(file string) *gateway.Gateway {

	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error(err)
		return nil
	}
	myGateway := &gateway.Gateway{}
	err = yaml.Unmarshal(b, myGateway)
	if err != nil {
		log.Error(err)
		return nil
	}

	newGateway := gateway.NewGateway(myGateway.Addr, myGateway.ReadTimeout, myGateway.WriteTimeout)

	for routeName, existingRoute := range myGateway.Routes {
		log.Debugf("Adding existing route %v to  new Gateway", routeName)
		newRoute, err := route.New(existingRoute.Name, existingRoute.Prefix, existingRoute.Rewrite, existingRoute.Host,
			existingRoute.Methods, upstreamclient.NewDefaultClient())

		if err != nil {
			log.Error(err)
			return nil
		}

		err = newGateway.RegisterRoute(newRoute)
		if err != nil {
			log.Error(err)
			return nil
		}

		for backendID, backend := range existingRoute.Backends {
			log.Debugf("Adding existing backend %v to Route %v", backendID, routeName)

			for _, cond := range backend.Metricthresholds {
				cond.IsTrue = cond.Compile()
			}
			newGateway.Routes[routeName].AddExistingBackend(backend)
		}

		newGateway.Routes[routeName].Reload()
	}

	newGateway.Reload()
	return newGateway
}

func main() {
	log.SetLevel(log.WarnLevel)
	promPort := ":8084"
	promPath := "/metrics"
	var g *gateway.Gateway
	// load confi

	// init prometheus
	http.Handle(promPath, promhttp.Handler())
	go http.ListenAndServe(promPort, nil)

	configFile := "gateway-config.yaml"

	newGateway := loadFromFile(configFile)
	if newGateway != nil {
		g = newGateway
		go g.Run()

	} else {
		panic(fmt.Errorf("Unable to recreate the Gateway specified in file"))

		/*
			g = gateway.NewGateway(":8080", 5000, 5000)
			// setup complete. Start servers
			log.Info("Starting gateway")


			r, _ := route.New(
				"Route1",
				"/",
				"/",
				"*",
				[]string{"GET", "POST"},
				upstreamclient.NewDefaultClient(),
			)

			if err := g.RegisterRoute(r); err != nil {
				panic(err)
			}

			// old
			id1 := r.AddBackend("Test2", "http://localhost:9090", "http://localhost:9091/metrics",
				"http://localhost:9090/", ScrapeMetrics, DefaultMetricsThresholds, 100)

			// new
			id2 := r.AddBackend(
				"Test2", "http://localhost:7070", "http://localhost:7071/metrics",
				"http://localhost:7070/", ScrapeMetrics, DefaultMetricsThresholds, 0)

			r.Reload()

			go func() {
				// test switchover
				time.Sleep(10 * time.Second)

				cond1 := conditional.NewCondition(
					"ResponseTime",
					"<",
					1000,
					2*time.Minute,
				)

				conditionsList := []*conditional.Condition{
					cond1,
				}

				backendIds := []uuid.UUID{}
				for key := range r.Backends {
					backendIds = append(backendIds, key)
				}
				if route, found := g.Routes["Route1"]; found {
					route.StartSwitchOver(id1, id2, conditionsList, 10*time.Second, 5)
				}
			}()

			r2, _ := route.New(
				"Route2",
				"/test/",
				"/",
				"*",
				[]string{"GET", "POST"},
				upstreamclient.NewDefaultClient(),
			)

			r2.AddBackend(
				"Test2", "http://localhost:7070", "http://localhost:7071/metrics",
				"http://localhost:7070/", ScrapeMetrics, DefaultMetricsThresholds, 100)

			if err := g.RegisterRoute(r2); err != nil {
				panic(err)
			}
		*/
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
	signalChannel := make(chan os.Signal, 2)
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
	g.SaveConfigToFile("gateway-config.yaml")
	g.Stop()
}
