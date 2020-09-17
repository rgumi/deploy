package config

import (
	"encoding/json"
	"io/ioutil"

	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/route"
	"github.com/rgumi/depoy/upstreamclient"

	"github.com/creasty/defaults"
	"github.com/prometheus/common/log"
	"gopkg.in/dealancer/validate.v2"
	"gopkg.in/yaml.v2"
)

func ParseFromBinaryJSON(b []byte) (*gateway.Gateway, error) {
	var err error
	myGateway := &gateway.Gateway{}
	defaults.Set(myGateway)

	err = json.Unmarshal(b, myGateway)
	if err != nil {
		return nil, err
	}

	err = validate.Validate(myGateway)
	if err != nil {
		return nil, err
	}

	newGateway := gateway.NewGateway(myGateway.Addr, myGateway.ReadTimeout, myGateway.WriteTimeout)

	for routeName, existingRoute := range myGateway.Routes {
		log.Debugf("Adding existing route %v to  new Gateway", routeName)

		newRoute, err := route.New(
			existingRoute.Name, existingRoute.Prefix, existingRoute.Rewrite, existingRoute.Host,
			existingRoute.Methods, upstreamclient.NewDefaultClient())

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

func ParseFromBinaryYAML(b []byte) (*gateway.Gateway, error) {
	var err error
	myGateway := &gateway.Gateway{}
	defaults.Set(myGateway)

	err = yaml.Unmarshal(b, myGateway)
	if err != nil {
		return nil, err
	}

	err = validate.Validate(myGateway)
	if err != nil {
		return nil, err
	}

	newGateway := gateway.NewGateway(
		myGateway.Addr, myGateway.ReadTimeout, myGateway.WriteTimeout)

	for routeName, existingRoute := range myGateway.Routes {
		log.Debugf("Adding existing route %v to  new Gateway", routeName)

		newRoute, err := route.New(
			existingRoute.Name, existingRoute.Prefix, existingRoute.Rewrite, existingRoute.Host,
			existingRoute.Methods, upstreamclient.NewDefaultClient())

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

func LoadFromFile(file string) *gateway.Gateway {

	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error(err)
		return nil
	}
	g, err := ParseFromBinaryYAML(b)
	if err != nil {
		log.Error(err)
		return nil
	}
	return g
}
