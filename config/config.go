package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/creasty/defaults"
	"github.com/rgumi/depoy/gateway"

	log "github.com/sirupsen/logrus"
	"gopkg.in/dealancer/validate.v2"
)

// UnmarshalFunc implements the UnmarshalFunc interface
type UnmarshalFunc func(data []byte, v interface{}) error

// ParseFromBinary uses the provided unmarshalFunc to create a new gateway object from b
func ParseFromBinary(unmarshal UnmarshalFunc, b []byte) (*gateway.Gateway, error) {
	var err error
	existingGateway := NewInputeGateway()
	err = unmarshal(b, existingGateway)
	err = validate.Validate(existingGateway)
	if err != nil {
		return nil, err
	}
	newGateway := ConvertInputGatewayToGateway(existingGateway)
	for _, existingRoute := range existingGateway.Routes {
		if err := defaults.Set(existingRoute); err != nil {
			return nil, err
		}

		log.Infof("Adding existing route %v to  new Gateway", existingRoute.Name)
		newRoute, err := ConvertInputRouteToRoute(existingRoute)
		if err != nil {
			return nil, err
		}
		if err = existingRoute.Strategy.Validate(newRoute); err != nil {
			return nil, err
		}
		err = existingRoute.Strategy.Copy(newRoute)
		if err != nil {
			return nil, err
		}
		err = newGateway.RegisterRoute(newRoute)
		if err != nil {
			return nil, err
		}
		newRoute.Reload()
		log.Warnf("Successfully reloaded route %s", newRoute.Name)
	}
	newGateway.Reload()
	log.Warnf("Successfully reloaded Gateway")
	return newGateway, nil
}

// LoadFromFile can be used at startup to read the config from a yaml-file
func LoadFromFile(file string) *gateway.Gateway {
	start := time.Now()
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	g, err := ParseFromBinary(yaml.Unmarshal, b)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Finished initialization of Gateway from file in %v", time.Since(start))
	return g
}

func WriteToFile(g *gateway.Gateway, file string) error {
	out := ConvertGatewayToInputGateway(g)
	out.Routes = make([]*InputRoute, len(g.Routes))
	i := 0
	for _, r := range g.Routes {
		outRoute := ConvertRouteToInputRoute(r)
		out.Routes[i] = outRoute
		i++
	}
	b, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, b, 0777)
	return err
}
