package config

import (
	"strings"

	"github.com/Netflix/go-env"
)

// Config holds all the config needed for the application
type Config struct {
	LogLevel          string `env:"LOG_LEVEL`
	PGHost            string `env:"PG_HOST"`
	DockerHost        string `env:"DOCKER_HOST"`
	DockerNetwork     string `env:"DOCKER_NETWORK_NAME"`
	HoneypotImage     string `env:"HONEYPOT_IMAGE"`
	SSHHostKeys       []string
	SSHHostKeysString string `env:"SSH_HOST_KEYS"`
	Port              int    `env:"PORT"`
	HostBuffer        int    `env:"HOST_BUFFER"`
}

// GetConfig returns the configuration
func GetConfig() (Config, error) {
	cfg := Config{}

	_, err := env.UnmarshalFromEnviron(&cfg)
	if err != nil {
		return cfg, err
	}
	cfg.SSHHostKeys = strings.Split(cfg.SSHHostKeysString, ":")

	return cfg, err
}
