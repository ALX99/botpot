package hostprovider

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type DockerProvider struct {
	sync.RWMutex
	client            *client.Client
	managedContainers map[string]bool
	host              string
	image             string
	env               []string
}

// NewDockerProvider creates a new docker provider
func NewDockerProvider(host, image string, env []string) *DockerProvider {
	return &DockerProvider{
		host:              host,
		image:             image,
		env:               env,
		managedContainers: make(map[string]bool),
	}
}

func (d *DockerProvider) Start() error {
	var err error
	d.client, err = client.NewClientWithOpts(client.WithHost(d.host))
	if err != nil {
		return err
	}

	// Pull image
	readCloser, err := d.client.ImagePull(context.TODO(), d.image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer readCloser.Close()
	io.Copy(os.Stdout, readCloser) // this needs to be handled for whatever reason

	return nil
}

func (d *DockerProvider) createAndRunContainer() (string, error) {

	res, err := d.client.ContainerCreate(context.TODO(),
		&container.Config{Image: d.image, Env: d.env},
		&container.HostConfig{Privileged: false, PublishAllPorts: true},
		&network.NetworkingConfig{},
		&v1.Platform{},
		"",
	)
	if err != nil {
		return "", err
	}

	log.Printf("Container %s created", res.ID)
	d.Lock()
	d.managedContainers[res.ID] = false
	d.Unlock()

	err = d.client.ContainerStart(context.TODO(), res.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", err
	}
	log.Printf("Container %s started", res.ID)

	d.Lock()
	d.managedContainers[res.ID] = true
	d.Unlock()

	return res.ID, nil
}

func (d *DockerProvider) Stop() error {
	var errs error
	for ID := range d.managedContainers {
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
	timeout := 10 * time.Second
	err := d.client.ContainerStop(context.TODO(), ID, &timeout)
	if err != nil {
		return err
	}
	d.Lock()
	d.managedContainers[ID] = false
	d.Unlock()

	return nil
}

// deleteContainer will optionally stop and delete a container
func (d *DockerProvider) deleteContainer(ID string) error {
	d.RLock()
	found, running := d.managedContainers[ID]
	d.RUnlock()
	if !found {
		return fmt.Errorf("container with ID %s not found", ID)
	}
	if running {
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
	delete(d.managedContainers, ID)
	d.Unlock()

	return nil
}

// GetHost returns an available host in the format IP:PORT
// to connect to
func (d *DockerProvider) GetHost() (string, string, error) {
	ID, err := d.createAndRunContainer()
	if err != nil {
		return "", "", err
	}
	res, err := d.client.ContainerInspect(context.TODO(), ID)
	if err != nil {
		return "", "", err
	}
	// TODO this has to be fixed not to always return localhost
	// and not always assume that 2222/tcp is the ssh port
	return fmt.Sprintf("127.0.0.1:%s", res.NetworkSettings.Ports["2222/tcp"][0].HostPort), ID, err
}

// StopHost stops a managed host
func (d *DockerProvider) StopHost(ID string) error {
	return d.deleteContainer(ID)
}
