package config

import (
	"flag"
	"time"
)

/*
	CLI flags that can be used to configure the application on startup

*/
var (

	// global
	PersistConfigOnExit bool
	ConfigFile          string
	LogLevel            int

	// statemgt
	StateMgtAddr string

	// gateway
	GatewayAddr  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	HTTPTimeout  time.Duration
	IdleTimeout  time.Duration

	// metrics
	// MetricsChannelPuffersize defines the maximal puffer size of the
	// Metric Channel. This can be increased by there are too many concurrent
	// requests and the Storage Job cannot keep up
	MetricsChannelPuffersize int
	// ScrapeMetricsChannelPuffersize defines the maximal puffer size of
	// the Scrape Metric Channel. This should never be a problem
	ScrapeMetricsChannelPuffersize int
	// Granulartiy defines the granularity of the metrics that are evaluated
	// in the Monitoring-Job. The higher the value, the more historic data will be used
	Granulartiy     time.Duration
	RetentionPeriod time.Duration
)

func init() {
	// global config
	flag.BoolVar(&PersistConfigOnExit, "global.persistconfig", true, "defines if configs of gateway are stored on exit")
	flag.StringVar(&ConfigFile, "global.configfile", "", "configfile to get and store config of gateway")
	flag.IntVar(&LogLevel, "global.loglevel", 3, "loglevel of the application (default=warn)")

	// statemgt
	flag.StringVar(&StateMgtAddr, "statemgt.addr", ":8081", "The address that the statemgt listens on")

	// gateway defaults (overwritten by configfile)
	flag.StringVar(&GatewayAddr, "gateway.addr", ":8080", "The address that the gateway listens on (overwritten by configfile)")
	ReadTimeout = time.Duration(*flag.Int("gateway.readtimeout", 5, "read timeout of in seconds (overwritten by configfile)")) * time.Second
	WriteTimeout = time.Duration(*flag.Int("gateway.writeTimeout", 5, "write timeout in seconds (overwritten by configfile)")) * time.Second
	HTTPTimeout = time.Duration(*flag.Int("gateway.httpTimeout", 10, "read timeout of in seconds (overwritten by configfile)")) * time.Second
	IdleTimeout = time.Duration(*flag.Int("gateway.idleTimeout", 30, "write timeout in seconds (overwritten by configfile)")) * time.Second

	// metrics defaults
	flag.IntVar(&MetricsChannelPuffersize, "metrics.metricsPuffersize", 100, "Size of the puffer for the metric channel")
	flag.IntVar(&ScrapeMetricsChannelPuffersize, "metrics.scrapePuffersize", 50, "Size of the puffer for the scrapeMetric channel")
	RetentionPeriod = time.Duration(*flag.Int("metrics.retentionPeriod", 10, "number of minutes after a collected metric is deleted")) * time.Minute
	Granulartiy = time.Duration(*flag.Int("metrics.granulartiy", 5, "number of second that define the granularity of stored metrics")) * time.Second
}
