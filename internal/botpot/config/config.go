package config

// https://zhwt.github.io/yaml-to-go/

// Config holds all the config needed for the application
type Config struct {
	Botpot struct {
		Port     int    `yaml:"port"`
		LogLevel string `yaml:"logLevel"`
		Postgres struct {
			URI string `yaml:"uri"`
		} `yaml:"postgres"`
		Dockerprovider struct {
			Host       string `yaml:"host"`
			HostBuffer int    `yaml:"hostBuffer"`
		} `yaml:"dockerprovider"`
	} `yaml:"botpot"`
}
