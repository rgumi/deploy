package config

import (
	"io/ioutil"

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
	myGateway := &gateway.Gateway{}
	defaults.Set(myGateway)

	err = unmarshal(b, myGateway)
	if err != nil {
		return nil, err
	}

	err = validate.Validate(myGateway)
	if err != nil {
		return nil, err
	}

	newGateway := gateway.NewGateway(
		myGateway.Addr,
		myGateway.ReadTimeout,
		myGateway.WriteTimeout,
		myGateway.ScrapeInterval,
	)

	for routeName, existingRoute := range myGateway.Routes {
		log.Debugf("Adding existing route %v to  new Gateway", routeName)

		newRoute, err := route.New(
			existingRoute.Name,
			existingRoute.Prefix,
			existingRoute.Rewrite,
			existingRoute.Host,
			existingRoute.Proxy,
			existingRoute.Methods,
			existingRoute.Timeout,
			existingRoute.IdleTimeout,
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

		for backendID, backend := range existingRoute.Backends {

			log.Debugf("Adding existing backend %v to Route %v", backendID, routeName)

			for _, cond := range backend.Metricthresholds {
				cond.IsTrue = cond.Compile()
			}

			_, err := newGateway.Routes[routeName].AddExistingBackend(backend)

			if err != nil {
				return nil, err
			}
		}

		err = existingRoute.Strategy.Reset(newRoute)
		// should never return an err
		if err != nil {
			// rollback
			newGateway.RemoveRoute(routeName)
			return nil, err
		}

		newGateway.Routes[routeName].Reload()
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
