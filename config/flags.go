package config

import (
	"flag"
)

/*
	CLI flags that can be used to configure the application on startup

*/
var (
	PersistConfigOnExit bool
	ConfigFile          string
	PromPath            string
	LogLevel            int
	ReadTimeout         int
	WriteTimeout        int
)

func init() {
	flag.BoolVar(&PersistConfigOnExit, "global.persistconfig", true, "defines if configs of gateway are stored on exit")
	flag.StringVar(&ConfigFile, "global.configfile", "", "configfile to get and store config of gateway")
	flag.IntVar(&LogLevel, "global.loglevel", 3, "loglevel of the application (default=warn)")

	flag.StringVar(&PromPath, "statemgt.prompath", "/metrics", "path on which Prometheus metrics are served")
	flag.IntVar(&ReadTimeout, "statemgt.readtimeout", 5000, "read timeout of the stateMgt")
	flag.IntVar(&WriteTimeout, "statemgt.writeTimeout", 5000, "write timeout of the stateMgt")
}
