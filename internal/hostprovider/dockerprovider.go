package hostprovider

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/alx99/botpot/internal/host"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
)

type DockerProvider struct {
	sync.RWMutex
	client        *client.Client
	containers    map[string]*host.DHost
	host          string
	hostBuffer    int
	running       bool
	config        container.Config
	hostConfig    container.HostConfig
	networkConfig network.NetworkingConfig
	plaform       specs.Platform
}

// NewDockerProvider creates a new docker provider
func NewDockerProvider(hostt string, config container.Config, hostConfig container.HostConfig, networkConfig network.NetworkingConfig, platform specs.Platform, hostBuffer int) *DockerProvider {
	return &DockerProvider{
		host:          hostt,
		config:        config,
		hostConfig:    hostConfig,
		networkConfig: networkConfig,
		plaform:       platform,
		containers:    make(map[string]*host.DHost),
		hostBuffer:    hostBuffer,
	}
}

func (d *DockerProvider) Start() error {
	log.Info().Msg("Starting DockerProvider")
	var err error
	d.client, err = client.NewClientWithOpts(client.WithHost(d.host))
	if err != nil {
		return err
	}

	// Pull image
	readCloser, err := d.client.ImagePull(context.TODO(), d.config.Image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer readCloser.Close()
	io.Copy(os.Stdout, readCloser) // this needs to be handled for whatever reason

	d.running = true
	go d.monitorHostBuf()
	return nil
}

func (d *DockerProvider) monitorHostBuf() {
	for d.running {
		occupiedCount := 0
		d.RLock()
		hostCount := len(d.containers)
		for _, h := range d.containers {
			if h.Occupied() {
				occupiedCount++
			}
		}
		d.RUnlock()

		for i := 0; i < d.hostBuffer-(hostCount-occupiedCount); i++ {
			_, err := d.createAndRunContainer()
			if err != nil {
				log.Err(err).Msg("Error while creating&running container")
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (d *DockerProvider) createAndRunContainer() (*host.DHost, error) {
	res, err := d.client.ContainerCreate(context.TODO(), &d.config, &d.hostConfig, &d.networkConfig, &d.plaform, "")
	if err != nil {
		return nil, err
	}

	d.Lock()
	d.containers[res.ID] = host.NewDHost(res.ID)
	d.Unlock()

	err = d.client.ContainerStart(context.TODO(), res.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	d.Lock()
	h := d.containers[res.ID]
	h.SetRunning(true)
	d.Unlock()

	return h, nil
}

func (d *DockerProvider) Stop() error {
	log.Info().Msg("Stopping DockerProvider")
	d.running = false
	var errs error
	for ID := range d.containers {
		err := d.deleteContainer(ID)
		if err != nil {
			if errs != nil {
				errs = fmt.Errorf("%w %s", err, errs)
			} else {
				errs = err
			}
		}
	}
	return errs
}

func (d *DockerProvider) stopContainer(ID string) error {
	d.RLock()
	h, ok := d.containers[ID]
	d.RUnlock()
	if !ok {
		return fmt.Errorf("container with ID %s not found", ID)
	}

	timeout := 10 * time.Second
	err := d.client.ContainerStop(context.TODO(), ID, &timeout)
	if err != nil {
		return err
	}

	h.SetRunning(false)
	return nil
}

// deleteContainer will optionally stop and delete a container
func (d *DockerProvider) deleteContainer(ID string) error {
	d.RLock()
	h, ok := d.containers[ID]
	d.RUnlock()
	if !ok {
		return fmt.Errorf("container with ID %s not found", ID)
	}

	if h.Running() {
		err := d.stopContainer(ID)
		if err != nil {
			return err
		}
	}

	err := d.client.ContainerRemove(context.TODO(), ID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         true,
	})
	if err != nil {
		return err
	}

	d.Lock()
	delete(d.containers, ID)
	d.Unlock()

	return nil
}

// GetHost returns an available host in the format IP:PORT
// to connect to
func (d *DockerProvider) GetHost() (string, string, error) {
	var H *host.DHost
	for _, h := range d.containers {
		if h.Running() && !h.Occupied() {
			H = h
			break
		}
	}

	// In case no available containers
	if H == nil {
		var err error
		H, err = d.createAndRunContainer()
		if err != nil {
			return "", "", err
		}
	}

	res, err := d.client.ContainerInspect(context.TODO(), H.ID())
	if err != nil {
		return "", "", err
	}
	H.SetOccupied(true)
	// TODO this has to be fixed not to always return localhost
	// and not always assume that 2222/tcp is the ssh port
	return fmt.Sprintf("127.0.0.1:%s", res.NetworkSettings.Ports["2222/tcp"][0].HostPort), H.ID(), err
}

// StopHost stops a managed host
func (d *DockerProvider) StopHost(ID string) error {
	return d.deleteContainer(ID)
}
