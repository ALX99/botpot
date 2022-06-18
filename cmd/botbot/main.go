package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/alx99/botpot/internal/botpot/ssh"
	"github.com/alx99/botpot/internal/hostprovider"
)

func main() {
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
		log.Println(err)
	}

	server := ssh.New(2000, provider)
	err = server.Start()
	if err != nil {
		log.Println(err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c

	err = server.Stop()
	if err != nil {
		log.Println(err)
	}

	err = provider.Stop()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
