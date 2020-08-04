package config

// Config saves all configs for the application
// will be filled by a config file or env
// with env > file
type Config struct {
}

func New() *Config {
	return new(Config)
}
