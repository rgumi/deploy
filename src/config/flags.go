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
	PromAddr            string
	LogLevel            int
	ReadTimeout         int
	WriteTimeout        int
)

func init() {
	flag.BoolVar(&PersistConfigOnExit, "persistconfig", true, "defines if configs of gateway are stored on exit")
	flag.StringVar(&ConfigFile, "configfile", "", "configfile to get and store config of gateway")
	flag.StringVar(&PromPath, "prompath", "/metrics", "path on which Prometheus metrics are served")
	flag.StringVar(&PromAddr, "promaddr", ":8084", "address on which the Prometheus server listens")
	flag.IntVar(&LogLevel, "loglevel", 3, "loglevel of the application (default=warn)")
	flag.IntVar(&ReadTimeout, "readtimeout", 5000, "read timeout of the stateMgt")
	flag.IntVar(&WriteTimeout, "writeTimeout", 5000, "wr9te timeout of the stateMgt")
}
