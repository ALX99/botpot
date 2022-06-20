package main

import (
	"os"
	"os/signal"

	"github.com/alx99/botpot/internal/botpot/db"
	"github.com/alx99/botpot/internal/botpot/ssh"
	"github.com/alx99/botpot/internal/hostprovider"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	provider := hostprovider.NewDockerProvider(
		"unix:///var/run/docker.sock",
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
		1,
	)

	db := db.NewDB("postgres://postgres:example@localhost:5432/postgres")
	sshServer := ssh.New(2000, provider, &db)

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
