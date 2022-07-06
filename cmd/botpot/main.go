package main

import (
	"os"
	"os/signal"
	"strings"

	"github.com/alx99/botpot/internal/botpot/config"
	"github.com/alx99/botpot/internal/botpot/db"
	"github.com/alx99/botpot/internal/botpot/hostprovider"
	"github.com/alx99/botpot/internal/botpot/misc"
	"github.com/alx99/botpot/internal/botpot/ssh"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := setup()

	provider := hostprovider.NewDockerProvider(
		cfg.DockerHost,
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
		cfg.HostBuffer,
	)

	db := db.NewDB(cfg.PGHost)
	sshServer := ssh.New(cfg.Port, provider, &db)

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
}

func setup() config.Config {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	switch strings.ToLower(misc.GetEnv("LOG_LEVEL")) {
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
	return config.Config{
		Port:       misc.GetEnvInt("PORT"),
		LogLevel:   misc.GetEnv("LOG_LEVEL"),
		PGHost:     misc.GetEnv("PG_HOST"),
		DockerHost: misc.GetEnv("DOCKER_HOST"),
		HostBuffer: misc.GetEnvInt("HOST_BUFFER"),
	}
}
