package config

import (
	"io/ioutil"

	"github.com/google/uuid"
	"github.com/rgumi/depoy/storage"

	"github.com/rgumi/depoy/metrics"

	"gopkg.in/yaml.v3"

	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/route"

	"github.com/creasty/defaults"
	log "github.com/sirupsen/logrus"
	"gopkg.in/dealancer/validate.v2"
)

// UnmarshalFunc implements the UnmarshalFunc interface
type UnmarshalFunc func(data []byte, v interface{}) error

// ParseFromBinary uses the provided unmarshalFunc to create a new gateway object from b
func ParseFromBinary(unmarshal UnmarshalFunc, b []byte) (*gateway.Gateway, error) {
	var err error
	myGateway := new(InputGateway)
	if err := defaults.Set(myGateway); err != nil {
		panic(err)
	}

	err = unmarshal(b, myGateway)
	err = validate.Validate(myGateway)

	if err != nil {
		return nil, err
	}

	_, newMetricsRepo := metrics.NewMetricsRepository(
		storage.NewLocalStorage(RetentionPeriod, Granulartiy),
		Granulartiy, MetricsChannelPuffersize, ScrapeMetricsChannelPuffersize,
	)

	newGateway := gateway.NewGateway(
		myGateway.Addr,
		newMetricsRepo,
		myGateway.ReadTimeout,
		myGateway.WriteTimeout,
		myGateway.HTTPTimeout,
		myGateway.IdleTimeout,
	)

	for _, existingRoute := range myGateway.Routes {
		defaults.Set(existingRoute)

		log.Debugf("Adding existing route %v to  new Gateway", existingRoute.Name)

		log.Warn(existingRoute.HealthCheck)
		newRoute, err := route.New(
			existingRoute.Name,
			existingRoute.Prefix,
			existingRoute.Rewrite,
			existingRoute.Host,
			existingRoute.Proxy,
			existingRoute.Methods,
			existingRoute.Timeout,
			existingRoute.IdleTimeout,
			existingRoute.ScrapeInterval,
			existingRoute.HealthCheckInterval,
			existingRoute.CookieTTL,
			*existingRoute.HealthCheck,
		)

		if err != nil {
			return nil, err
		}

		if err = existingRoute.Strategy.Validate(newRoute); err != nil {
			return nil, err
		}

		for _, backend := range existingRoute.Backends {
			if backend.ID == uuid.Nil {
				log.Debugf("Setting new uuid for %s", existingRoute.Name)
				backend.ID = uuid.New()
			}
			for _, cond := range backend.Metricthresholds {
				cond.Compile()
			}
			log.Debugf("Adding existing backend %v to Route %v", backend.ID, existingRoute.Name)
			_, err = newRoute.AddExistingBackend(backend)
			if err != nil {
				return nil, err
			}
		}
		err = existingRoute.Strategy.Reset(newRoute)

		err = newGateway.RegisterRoute(newRoute)
		if err != nil {
			return nil, err
		}

		if err != nil {
			// rollback
			newGateway.RemoveRoute(existingRoute.Name)
			return nil, err
		}

		newGateway.Routes[existingRoute.Name].Reload()
	}

	newGateway.Reload()
	return newGateway, nil
}

// LoadFromFile can be used at startup to read the config from a yaml-file
func LoadFromFile(file string) *gateway.Gateway {

	b, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	g, err := ParseFromBinary(yaml.Unmarshal, b)
	if err != nil {
		panic(err)
	}
	return g
}

func WriteToFile(g *gateway.Gateway, file string) error {
	out := &InputGateway{
		Addr:         g.Addr,
		ReadTimeout:  g.ReadTimeout,
		WriteTimeout: g.WriteTimeout,
		HTTPTimeout:  g.HTTPTimeout,
		IdleTimeout:  g.IdleTimeout,
		Routes:       []*InputRoute{},
	}

	for _, r := range g.Routes {
		outRoute := &InputRoute{
			Name:           r.Name,
			Prefix:         r.Prefix,
			Rewrite:        r.Rewrite,
			Strategy:       r.Strategy,
			Proxy:          r.Proxy,
			Timeout:        r.Timeout,
			ScrapeInterval: r.ScrapeInterval,
			Backends:       []*route.Backend{},
			CookieTTL:      r.CookieTTL,
			HealthCheck:    &r.HealthCheck,
			Host:           r.Host,
			IdleTimeout:    r.IdleTimeout,
			Methods:        r.Methods,
		}
		for _, backend := range r.Backends {

			outRoute.Backends = append(outRoute.Backends, backend)
		}
		out.Routes = append(out.Routes, outRoute)
	}

	b, err := yaml.Marshal(out)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, b, 0777)
	if err != nil {
		return err
	}

	return nil
}
