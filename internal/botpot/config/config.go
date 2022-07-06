package config

// https://zhwt.github.io/yaml-to-go/

// Config holds all the config needed for the application
type Config struct {
	Port       int
	LogLevel   string
	PGHost     string
	DockerHost string
	HostBuffer int
}
