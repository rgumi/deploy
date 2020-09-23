package config

import (
	"time"

	"github.com/rgumi/depoy/route"
)

type InputGateway struct {
	Addr         string        `yaml:"addr" json:"addr" validate:"empty=false"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout" default:"5s"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout" default:"5s"`
	HTTPTimeout  time.Duration `yaml:"http_timeout" json:"http_timeout" default:"10s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout" default:"30s"`
	Routes       []*InputRoute `yaml:"routes" json:"routes"`
}

type InputRoute struct {
	Name           string           `json:"name" yaml:"name" validate:"empty=false"`
	Prefix         string           `json:"prefix" yaml:"prefix" validate:"empty=false"`
	Methods        []string         `json:"methods" yaml:"methods" validate:"empty=false"`
	Host           string           `json:"host" yaml:"host" default:"*"`
	Rewrite        string           `json:"rewrite" yaml:"rewrite" validate:"empty=false"`
	CookieTTL      time.Duration    `json:"cookie_ttl" yaml:"cookieTTL" default:"5m0s"`
	Strategy       *route.Strategy  `json:"strategy" yaml:"strategy"`
	HealthCheck    bool             `json:"healthcheck" yaml:"healthcheck" default:"true"`
	Timeout        time.Duration    `json:"timeout" yaml:"timeout" default:"10s"`
	IdleTimeout    time.Duration    `json:"idle_timeout" yaml:"idle_timeout" default:"30s"`
	ScrapeInterval time.Duration    `json:"scrape_interval" yaml:"scrape_interval" default:"5s"`
	Proxy          string           `json:"proxy" yaml:"proxy" default:""`
	Backends       []*route.Backend `json:"backends" yaml:"backends"`
}
