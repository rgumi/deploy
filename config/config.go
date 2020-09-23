package config

import (
	"io/ioutil"

	"github.com/rgumi/depoy/storage"

	"github.com/rgumi/depoy/metrics"

	"gopkg.in/yaml.v3"

	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/route"

	"github.com/creasty/defaults"
	"github.com/prometheus/common/log"
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
			existingRoute.CookieTTL,
			existingRoute.HealthCheck,
		)

		if err != nil {
			return nil, err
		}

		if err = existingRoute.Strategy.Validate(newRoute); err != nil {
			return nil, err
		}

		err = newGateway.RegisterRoute(newRoute)
		if err != nil {
			return nil, err
		}

		for _, backend := range existingRoute.Backends {

			log.Debugf("Adding existing backend %v to Route %v", backend.ID, existingRoute.Name)

			for _, cond := range backend.Metricthresholds {
				cond.IsTrue = cond.Compile()
			}

			_, err := newGateway.Routes[existingRoute.Name].AddExistingBackend(backend)

			if err != nil {
				return nil, err
			}
		}

		err = existingRoute.Strategy.Reset(newRoute)
		// should never return an err
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
	out := new(InputGateway)
	out.Addr = g.Addr
	out.ReadTimeout = g.ReadTimeout
	out.WriteTimeout = g.WriteTimeout
	out.HTTPTimeout = g.HTTPTimeout
	out.IdleTimeout = g.IdleTimeout
	out.Routes = []*InputRoute{}

	for _, r := range g.Routes {
		outRoute := new(InputRoute)
		outRoute.Backends = []*route.Backend{}
		outRoute.CookieTTL = r.CookieTTL
		outRoute.HealthCheck = r.HealthCheck
		outRoute.Host = r.Host
		outRoute.IdleTimeout = r.IdleTimeout
		outRoute.Methods = r.Methods
		outRoute.Name = r.Name
		outRoute.Prefix = r.Prefix
		outRoute.Proxy = r.Proxy
		outRoute.Rewrite = r.Rewrite
		outRoute.Strategy = r.Strategy
		outRoute.Timeout = r.Timeout
		outRoute.ScrapeInterval = r.ScrapeInterval

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
