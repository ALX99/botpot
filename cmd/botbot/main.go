package main

import (
	"os"
	"os/signal"

	"github.com/alx99/botpot/internal/botpot/ssh"
	"github.com/alx99/botpot/internal/hostprovider"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	provider := hostprovider.NewDockerProvider(
		"unix:///var/run/docker.sock",
		"linuxserver/openssh-server:latest",
		[]string{
			"PUID=1000",
			"PGID=1000",
			"TZ=Europe/London",
			"SUDO_ACCESS=true",
			"PASSWORD_ACCESS=true",
			"USER_PASSWORD=password",
			"USER_NAME=panda",
		},
		1,
	)
	err := provider.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start provider")
	}

	server := ssh.New(2000, provider)
	err = server.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start SSH Server")
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c

	err = server.Stop()
	if err != nil {
		log.Err(err).Msg("Could not stop SSH Server")
	}

	err = provider.Stop()
	if err != nil {
		log.Err(err).Msg("Could not stop provider")
	}
}
