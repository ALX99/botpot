package config

import (
	"strconv"
	"strings"

	"github.com/alx99/envcache"
)

const (
	logLevel    = "LOG_LEVEL"
	pgHost      = "PG_HOST"
	dockerHost  = "DOCKER_HOST"
	sshHostKeys = "SSH_HOST_KEYS"
	port        = "PORT"
	hostBuffer  = "HOST_BUFFER"
)

var (
	entries = make(map[string]func(string) (any, error))
)

func init() {
	entries[logLevel] = nil
	entries[pgHost] = nil
	entries[dockerHost] = nil
	entries[sshHostKeys] = func(s string) (any, error) { return strings.Split(s, ":"), nil }
	entries[port] = func(s string) (any, error) { return strconv.Atoi(s) }
	entries[hostBuffer] = func(s string) (any, error) { return strconv.Atoi(s) }
}

// Config holds all the config needed for the application
type Config struct {
	LogLevel    string
	PGHost      string
	DockerHost  string
	SSHHostKeys []string
	Port        int
	HostBuffer  int
}

// GetConfig returns the configuration
func GetConfig() (Config, error) {
	err := envcache.CacheNewEntries(entries)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		LogLevel:    envcache.Get[string](logLevel),
		PGHost:      envcache.Get[string](pgHost),
		DockerHost:  envcache.Get[string](dockerHost),
		SSHHostKeys: envcache.Get[[]string](sshHostKeys),
		Port:        envcache.Get[int](port),
		HostBuffer:  envcache.Get[int](hostBuffer),
	}

	return cfg, err
}
