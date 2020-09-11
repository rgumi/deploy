package config

import (
	"io/ioutil"

	"github.com/creasty/defaults"
	"gopkg.in/dealancer/validate.v2"
	_ "gopkg.in/dealancer/validate.v2"
	"gopkg.in/yaml.v2"
)

type GeneralConfig struct {
	LogLevel     int    `yaml:"logLevel" default:"3"`
	PromAddr     string `yaml:"prometheusAddr" default:":8084"`
	StateMgtAddr string `yaml:"stateMgtAddr" default:"8081"`
}

type GatewayConfig struct {
	Addr         string `yaml:"addr" default:":8080"`
	ReadTimeout  int    `yaml:"readTimeout" default:"2000"`
	WriteTimeout int    `yaml:"writeTimeout" default:"2000"`
}

type RouteConfig struct {
	Name     string          `yaml:"name" validate:"empty=false"`
	Prefix   string          `yaml:"prefix" validate:"empty=false"`
	Rewrite  string          `yaml:"rewrite" validate:"empty=false"`
	Host     string          `yaml:"host" validate:"empty=false"`
	Methods  []string        `yaml:"methods" validate:"empty=false"`
	Timeout  int             `yaml:"timeout" default:"2000"`
	Backends []BackendConfig `yaml:"backends" validate:"empty=false"`
}

type BackendConfig struct {
	Name           string   `yaml:"name" validate:"empty=false"`
	UpstreamURL    string   `yaml:"upstreamUrl" validate:"empty=false & format=url"`
	ScrapeURL      string   `yaml:"scrapeUrl" validate:"empty=true | format=url"`
	HealthCheckURL string   `yaml:"healthCheckUrl" validate:"empty=true | format=url"`
	ScrapeMetrics  []string `yaml:"scrapeMetrics"`
}

// Config saves all configs for the application
// will be filled by a config file or env
// with env > file
type Config struct {
	GeneralConfig
	GatewayConfig
	Routes []RouteConfig `yaml:"routes"`
}

func New() *Config {
	return new(Config)
}

func (c *Config) PrintToFile(filepath string) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath, b, 0777)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) Read() ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func ReadFromFile(filepath string) (*Config, error) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var newConfig = &Config{}
	defaults.Set(newConfig)

	err = yaml.Unmarshal(b, newConfig)
	if err != nil {
		return nil, err
	}

	err = validate.Validate(newConfig)
	if err != nil {
		return newConfig, err
	}

	return newConfig, nil
}
