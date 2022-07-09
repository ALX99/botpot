package config

import "github.com/caarlos0/env/v6"

// Config holds all the config needed for the application
type Config struct {
	LogLevel    string   `env:"LOG_LEVEL"`
	PGHost      string   `env:"PG_HOST"`
	DockerHost  string   `env:"DOCKER_HOST"`
	SSHHostKeys []string `env:"SSH_HOST_KEYS" envSeparator:":"`
	Port        int      `env:"PORT"`
	HostBuffer  int      `env:"HOST_BUFFER"`
}

// GetConfig returns the configuration
func GetConfig() (Config, error) {
	cfg := Config{}
	err := env.Parse(&cfg)
	return cfg, err
}
