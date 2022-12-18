package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alx99/botpot/internal/botpot/config"
	"github.com/alx99/botpot/internal/botpot/db"
	"github.com/alx99/botpot/internal/botpot/hostprovider"
	"github.com/alx99/botpot/internal/botpot/ssh"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Variables set by linker
var (
	commitHash      = ""
	compilationDate = ""
)

func init() {
	// Did they really not fix it for postgres lol?
	// https://github.com/grafana/grafana/issues/18120
	os.Setenv("TZ", "UTC")
}

func main() {
	cfg := setup()
	log.Info().Str("commitHash", commitHash).Str("compilationDate", compilationDate).Msgf("Botpot started!")

	provider := hostprovider.NewDockerProvider(
		cfg.DockerHost,
		container.Config{Image: cfg.HoneypotImage,
			Env: []string{},
		},
		container.HostConfig{
			NetworkMode:     container.NetworkMode(cfg.DockerNetwork),
			AutoRemove:      true,
			Privileged:      false,
			PublishAllPorts: false,
			ReadonlyRootfs:  false,
		},
		network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{cfg.DockerNetwork: {}}},
		specs.Platform{},
		cfg.HostBuffer,
	)

	db := db.NewDB(cfg.PGHost)
	sshServer := ssh.New(cfg.Port, cfg.SSHHostKeys, provider, &db)

	err := db.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start database")
	}

	err = provider.Start(context.TODO())
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start provider")
	}

	err = sshServer.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start SSH Server")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	err = sshServer.Stop()
	if err != nil {
		log.Err(err).Msg("Could not stop SSH Server")
	}

	err = provider.Stop(context.TODO())
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

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not get config")
	}

	switch strings.ToLower(cfg.LogLevel) {
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
