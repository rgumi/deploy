package config

import (
	"net/url"

	"github.com/creasty/defaults"
	"github.com/google/uuid"
	"github.com/rgumi/depoy/conditional"
	"github.com/rgumi/depoy/gateway"
	"github.com/rgumi/depoy/metrics"
	"github.com/rgumi/depoy/route"
	"github.com/rgumi/depoy/storage"
	"github.com/rgumi/depoy/util"
	log "github.com/sirupsen/logrus"
)

type InputBackend struct {
	ID               uuid.UUID                `json:"id" yaml:"id" validate:"empty=false"`
	Name             string                   `json:"name" yaml:"name" validate:"empty=false"`
	Addr             string                   `json:"addr" yaml:"addr"`
	Weigth           uint8                    `json:"weight" yaml:"weight"`
	Active           bool                     `json:"active" yaml:"active"`
	Scrapeurl        string                   `json:"scrape_url" yaml:"scrapeUrl"`
	Scrapemetrics    []string                 `json:"scrape_metrics" yaml:"scrapeMetrics"`
	Metricthresholds []*conditional.Condition `json:"metric_thresholds" yaml:"metricThresholds"`
	Healthcheckurl   string                   `json:"healthcheck_url" yaml:"healthcheckUrl"`
	ActiveAlerts     map[string]metrics.Alert `json:"active_alerts" yaml:"-"`
}

type InputGateway struct {
	Addr         string              `yaml:"addr" json:"addr" default:":8080"`
	ReadTimeout  util.ConfigDuration `yaml:"read_timeout" json:"readTimeout" default:"\"5s\""`
	WriteTimeout util.ConfigDuration `yaml:"write_timeout" json:"writeTimeout" default:"\"5s\""`
	HTTPTimeout  util.ConfigDuration `yaml:"http_timeout" json:"httpTimeout" default:"\"5s\""`
	IdleTimeout  util.ConfigDuration `yaml:"idle_timeout" json:"idleTimeout" default:"\"5s\""`
	Routes       []*InputRoute       `yaml:"routes" json:"routes"`
}

type InputRoute struct {
	Name                string              `json:"name" yaml:"name" validate:"empty=false"`
	Prefix              string              `json:"prefix" yaml:"prefix" validate:"empty=false"`
	Methods             []string            `json:"methods" yaml:"methods" default:"[\"GET\", \"POST\", \"PUT\", \"DELETE\", \"PATCH\", \"HEAD\", \"OPTIONS\", \"TRACE\"]"`
	Host                string              `json:"host" yaml:"host" default:"*"`
	Rewrite             string              `json:"rewrite" yaml:"rewrite" validate:"empty=false"`
	CookieTTL           util.ConfigDuration `json:"cookie_ttl" yaml:"cookieTTL" default:"\"5m\""`
	Strategy            *route.Strategy     `json:"strategy" yaml:"strategy" validate:"nil=false"`
	Switchover          *InputSwitchover    `json:"switchover" yaml:"-"`
	HealthCheck         bool                `json:"healthcheck_bool" yaml:"healthcheckBool" default:"true"`
	HealthCheckInterval util.ConfigDuration `json:"healthcheck_interval" yaml:"healthcheckInterval" default:"\"5s\""`
	MonitoringInterval  util.ConfigDuration `json:"monitoring_interval" yaml:"monitoringInterval" default:"\"5s\""`
	Timeout             util.ConfigDuration `json:"timeout" yaml:"timeout" default:"\"5s\""`
	IdleTimeout         util.ConfigDuration `json:"idle_timeout" yaml:"idleTimeout" default:"\"5s\""`
	ScrapeInterval      util.ConfigDuration `json:"scrape_interval" yaml:"scrapeInterval" default:"\"5s\""`
	Proxy               string              `json:"proxy" yaml:"proxy" default:""`
	Backends            []*InputBackend     `json:"backends" yaml:"backends"`
}

// InputSwitchover is required to add a switchover to a route
// it is a wrapper for the actual SwitchOver struct and replaces
// the actual backends (from and to) with their corrosponding ids
type InputSwitchover struct {
	Status       string                   `json:"status"`
	From         string                   `json:"from"`
	To           string                   `json:"to" validate:"empty=false"`
	Conditions   []*conditional.Condition `json:"conditions" validate:"empty=false"`
	Timeout      util.ConfigDuration      `json:"timeout" default:"\"2m\""`
	WeightChange uint8                    `json:"weight_change" default:"5"`
	// Force overwrites the current config of the backends to enable switchover (if required)
	Force bool `json:"force,omitempty" default:"false"`
	// If switchover fails, rollback all changes to the weights and stop switchover
	Rollback bool `json:"rollback,omitempty" default:"true"`
	// The amount of times a cycle is allowed to fail before switchover is stopped
	AllowedFailures int `json:"allowed_failures" default:"5"`
	FailureCounter  int `json:"failure_counter"`
}

func NewInputBackend() *InputBackend {
	backend := new(InputBackend)
	defaults.Set(backend)
	return backend
}

func NewInputSwitchover() *InputSwitchover {
	switchover := new(InputSwitchover)
	defaults.Set(switchover)
	return switchover
}

func NewInputRoute() *InputRoute {
	route := new(InputRoute)
	defaults.Set(route)
	return route
}

func NewInputeGateway() *InputGateway {
	g := new(InputGateway)
	defaults.Set(g)
	return g
}

// Backend

func ConvertBackendToInputBackend(b *route.Backend) *InputBackend {
	inputBackend := &InputBackend{
		ID:               b.ID,
		Name:             b.Name,
		Addr:             b.Addr.String(),
		Weigth:           b.Weigth,
		Active:           b.Active,
		Scrapeurl:        b.Scrapeurl.String(),
		Scrapemetrics:    b.Scrapemetrics,
		Metricthresholds: b.Metricthresholds,
		Healthcheckurl:   b.Healthcheckurl.String(),
		ActiveAlerts:     b.ActiveAlerts,
	}
	return inputBackend
}

func ConvertInputBackendToBackend(b *InputBackend) (*route.Backend, error) {
	addr, err := url.Parse(b.Addr)
	if err != nil {
		return nil, err
	}
	healthcheckURL, err := url.Parse(b.Healthcheckurl)
	if err != nil {
		return nil, err
	}
	scrapeURL, err := url.Parse(b.Scrapeurl)
	if err != nil {
		return nil, err
	}
	backend, err := route.NewBackend(
		b.Name,
		addr,
		scrapeURL,
		healthcheckURL,
		b.Scrapemetrics,
		b.Metricthresholds,
		b.Weigth,
	)
	if err != nil {
		return nil, err
	}
	backend.ID = b.ID
	return backend, nil
}

// Route

func ConvertRouteToInputRoute(r *route.Route) *InputRoute {
	inputRoute := &InputRoute{
		Name:                r.Name,
		Prefix:              r.Prefix,
		Rewrite:             r.Rewrite,
		Strategy:            r.Strategy,
		Proxy:               r.Proxy,
		Timeout:             util.ConfigDuration{r.Timeout},
		ScrapeInterval:      util.ConfigDuration{r.ScrapeInterval},
		Backends:            []*InputBackend{},
		CookieTTL:           util.ConfigDuration{r.CookieTTL},
		HealthCheck:         r.HealthCheck,
		HealthCheckInterval: util.ConfigDuration{r.HealthCheckInterval},
		MonitoringInterval:  util.ConfigDuration{r.MonitoringInterval},
		Host:                r.Host,
		IdleTimeout:         util.ConfigDuration{r.IdleTimeout},
		Methods:             r.Methods,
	}
	inputRoute.Backends = make([]*InputBackend, len(r.Backends))
	i := 0
	for _, backend := range r.Backends {
		inputRoute.Backends[i] = ConvertBackendToInputBackend(backend)
		i++
	}
	if r.Switchover != nil {
		inputRoute.Switchover = ConvertSwitchoverToInputSwitchover(r.Switchover)
	}

	return inputRoute
}

func ConvertInputRouteToRoute(r *InputRoute) (*route.Route, error) {
	newRoute, err := route.New(
		r.Name,
		r.Prefix,
		r.Rewrite,
		r.Host,
		r.Proxy,
		r.Methods,
		r.Timeout.Duration,
		r.IdleTimeout.Duration,
		r.ScrapeInterval.Duration,
		r.HealthCheckInterval.Duration,
		r.MonitoringInterval.Duration,
		r.CookieTTL.Duration,
		r.HealthCheck,
	)

	for _, backend := range r.Backends {
		if backend.ID == uuid.Nil {
			log.Debugf("Setting new uuid for %s", r.Name)
			backend.ID = uuid.New()
		}
		for _, cond := range backend.Metricthresholds {
			cond.Compile()
		}
		log.Debugf("Adding existing backend %v to Route %v", backend.ID, r.Name)
		newBackend, err := ConvertInputBackendToBackend(backend)
		if err != nil {
			return nil, err
		}
		_, err = newRoute.AddExistingBackend(newBackend)
		if err != nil {
			return nil, err
		}
	}
	return newRoute, err
}

// Gateway

func ConvertInputGatewayToGateway(g *InputGateway) *gateway.Gateway {
	_, newMetricsRepo := metrics.NewMetricsRepository(
		storage.NewLocalStorage(RetentionPeriod, Granulartiy),
		Granulartiy, MetricsChannelPuffersize, ScrapeMetricsChannelPuffersize,
	)
	newGateway := gateway.NewGateway(
		g.Addr,
		newMetricsRepo,
		g.ReadTimeout.Duration,
		g.WriteTimeout.Duration,
		g.HTTPTimeout.Duration,
		g.IdleTimeout.Duration,
	)
	return newGateway
}
func ConvertGatewayToInputGateway(g *gateway.Gateway) *InputGateway {
	inputGateway := &InputGateway{
		Addr:         g.Addr,
		ReadTimeout:  util.ConfigDuration{g.ReadTimeout},
		WriteTimeout: util.ConfigDuration{g.WriteTimeout},
		HTTPTimeout:  util.ConfigDuration{g.HTTPTimeout},
		IdleTimeout:  util.ConfigDuration{g.IdleTimeout},
		Routes:       []*InputRoute{},
	}
	inputGateway.Routes = make([]*InputRoute, len(g.Routes))
	i := 0
	for _, r := range g.Routes {
		inputGateway.Routes[i] = ConvertRouteToInputRoute(r)
		i++
	}
	return inputGateway
}

// Switchover

func ConvertSwitchoverToInputSwitchover(s *route.Switchover) *InputSwitchover {
	inputRoute := &InputSwitchover{
		Status:          s.Status,
		From:            s.From.Name,
		To:              s.To.Name,
		FailureCounter:  s.FailureCounter,
		AllowedFailures: s.AllowedFailures,
		WeightChange:    s.WeightChange,
		Timeout:         util.ConfigDuration{s.Timeout},
		Conditions:      s.Conditions,
		Rollback:        s.Rollback,
	}
	return inputRoute
}
