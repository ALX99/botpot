package main

import (
	"os"
	"os/signal"
	"strings"

	"github.com/alx99/botpot/internal/botpot/config"
	"github.com/alx99/botpot/internal/botpot/db"
	"github.com/alx99/botpot/internal/botpot/ssh"
	"github.com/alx99/botpot/internal/hostprovider"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

func main() {
	cfg := setup()

	provider := hostprovider.NewDockerProvider(
		cfg.Botpot.Dockerprovider.Host,
		container.Config{Image: "linuxserver/openssh-server:latest",
			Env: []string{
				"PUID=1000",
				"PGID=1000",
				"TZ=Europe/London",
				"SUDO_ACCESS=true",
				"PASSWORD_ACCESS=true",
				"USER_PASSWORD=password",
				"USER_NAME=panda",
			},
		},
		container.HostConfig{Privileged: false, PublishAllPorts: true},
		network.NetworkingConfig{},
		specs.Platform{},
		cfg.Botpot.Dockerprovider.HostBuffer,
	)

	db := db.NewDB(cfg.Botpot.Postgres.URI)
	sshServer := ssh.New(cfg.Botpot.Port, provider, &db)

	err := db.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start database")
	}

	err = provider.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start provider")
	}

	err = sshServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start SSH Server")
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c

	err = sshServer.Stop()
	if err != nil {
		log.Err(err).Msg("Could not stop SSH Server")
	}

	err = provider.Stop()
	if err != nil {
		log.Err(err).Msg("Could not stop provider")
	}

	err = db.Stop()
	if err != nil {
		log.Err(err).Msg("Could not stop database")
	}
	log.Info().Msg("lolol")
}

func setup() config.Config {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	var cfg config.Config

	b, err := os.ReadFile("./botpot.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not read config file")
	}

	err = yaml.Unmarshal(b, &cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse config yaml file")
	}
	switch strings.ToLower(cfg.Botpot.LogLevel) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "info":
		fallthrough
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	return cfg
}
