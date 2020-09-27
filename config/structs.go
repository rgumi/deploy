package config

import (
	"time"

	"github.com/rgumi/depoy/route"
)

type InputGateway struct {
	Addr         string        `yaml:"addr" json:"addr" validate:"empty=false"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"readTimeout" default:"5s"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"writeTimeout" default:"5s"`
	HTTPTimeout  time.Duration `yaml:"http_timeout" json:"httpTimeout" default:"10s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idleTimeout" default:"30s"`
	Routes       []*InputRoute `yaml:"routes" json:"routes"`
}

type InputRoute struct {
	Name                string           `json:"name" yaml:"name" validate:"empty=false"`
	Prefix              string           `json:"prefix" yaml:"prefix" validate:"empty=false"`
	Methods             []string         `json:"methods" yaml:"methods" validate:"empty=false"`
	Host                string           `json:"host" yaml:"host" default:"*"`
	Rewrite             string           `json:"rewrite" yaml:"rewrite" validate:"empty=false"`
	CookieTTL           time.Duration    `json:"cookie_ttl" yaml:"cookieTTL" default:"5m0s"`
	Strategy            *route.Strategy  `json:"strategy" yaml:"strategy" validate:"nil=false"`
	HealthCheck         *bool            `json:"do_healthcheck" yaml:"doHealthcheck" validate:"nil=false"`
	HealthCheckInterval time.Duration    `json:"healthcheck_interval" yaml:"healthcheckInterval" default:"5s"`
	Timeout             time.Duration    `json:"timeout" yaml:"timeout" default:"10s"`
	IdleTimeout         time.Duration    `json:"idle_timeout" yaml:"idleTimeout" default:"30s"`
	ScrapeInterval      time.Duration    `json:"scrape_interval" yaml:"scrapeInterval" default:"5s"`
	Proxy               string           `json:"proxy" yaml:"proxy" default:""`
	Backends            []*route.Backend `json:"backends" yaml:"backends"`
}
