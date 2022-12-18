package hostprovider

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alx99/botpot/internal/botpot/host"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
)

// DockerProvider provides docker containers that
// run SSH servers that can serve attackers
type DockerProvider struct {
	sync.RWMutex
	running       atomic.Bool
	hostConfig    container.HostConfig
	client        *client.Client
	containers    map[string]*host.DHost
	networkConfig network.NetworkingConfig
	shutdown      <-chan any
	plaform       specs.Platform
	host          string
	config        container.Config
	hostBuffer    int
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
		shutdown:      make(<-chan any),
	}
}

func (d *DockerProvider) Start(ctx context.Context) error {
	log.Info().Msg("Starting DockerProvider")
	var err error
	d.client, err = client.NewClientWithOpts(client.WithHost(d.host))
	if err != nil {
		return err
	}
	list, err := d.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return err
	}

	found := false
	for _, image := range list {
		for _, tag := range image.RepoTags {
			found = tag == d.config.Image
			if found {
				break
			}
		}
		if found {
			break
		}
	}

	if !found { // Pull image
		readCloser, err := d.client.ImagePull(ctx, d.config.Image, types.ImagePullOptions{})
		if err != nil {
			return err
		}
		defer readCloser.Close()
		io.Copy(os.Stdout, readCloser) // this needs to be handled for whatever reason
	}

	d.running.Store(true)
	go d.monitorHostBuf(ctx)
	return nil
}

func (d *DockerProvider) monitorHostBuf(ctx context.Context) {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
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
				_, err := d.createAndRunContainer(ctx)
				if err != nil {
					log.Err(err).Msg("Error while creating&running container")
				}
			}

		case <-d.shutdown:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (d *DockerProvider) createAndRunContainer(ctx context.Context) (*host.DHost, error) {
	t := time.Now()
	res, err := d.client.ContainerCreate(ctx, &d.config, &d.hostConfig, &d.networkConfig, &d.plaform, "")
	if err != nil {
		return nil, err
	}

	d.Lock()
	d.containers[res.ID] = host.NewDHost(res.ID)
	d.Unlock()

	err = d.client.ContainerStart(ctx, res.ID, types.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}
	log.Debug().Dur("timeSinceCreation", time.Since(t)).Msg("Container started")

	d.Lock()
	h := d.containers[res.ID]
	h.SetRunning(true)
	d.Unlock()

	return h, nil
}

func (d *DockerProvider) Stop(ctx context.Context) error {
	log.Info().Msg("Stopping DockerProvider")
	d.running.Store(false)

	var errs error
	for ID := range d.containers {
		err := d.deleteContainer(ctx, ID)
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

func (d *DockerProvider) stopContainer(ctx context.Context, ID string) error {
	d.RLock()
	h, ok := d.containers[ID]
	d.RUnlock()
	if !ok {
		return fmt.Errorf("container with ID %s not found", ID)
	}

	timeout := 10 * time.Second
	err := d.client.ContainerStop(ctx, ID, &timeout)
	if err != nil {
		return err
	}

	h.SetRunning(false)
	return nil
}

// deleteContainer will optionally stop and delete a container
func (d *DockerProvider) deleteContainer(ctx context.Context, ID string) error {
	d.RLock()
	h, ok := d.containers[ID]
	d.RUnlock()
	if !ok {
		return fmt.Errorf("container with ID %s not found", ID)
	}

	if h.Running() {
		err := d.stopContainer(ctx, ID)
		if err != nil {
			return err
		}
	}

	err := d.client.ContainerRemove(ctx, ID, types.ContainerRemoveOptions{
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
func (d *DockerProvider) GetHost(ctx context.Context) (string, string, error) {
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
		H, err = d.createAndRunContainer(ctx)
		if err != nil {
			return "", "", err
		}
	}

	res, err := d.client.ContainerInspect(ctx, H.ID())
	if err != nil {
		return "", "", err
	}
	H.SetOccupied(true)

	// Obtain network name
	networkName := ""
	for k := range d.networkConfig.EndpointsConfig {
		networkName = k
		break
	}

	if networkName == "" {
		return "", "", errors.New("could not obtain network name")
	}

	endPointSettings, ok := res.NetworkSettings.Networks[networkName]
	if !ok {
		return "", "", errors.New("could not find network name")
	}

	// TODO this has to be fixed not to always return localhost
	// and not always assume that 22/tcp is the ssh port
	return fmt.Sprintf("%s:22", endPointSettings.IPAddress), H.ID(), err
}

// GetScriptOutput gets the script output and timing files
func (d *DockerProvider) GetScriptOutput(ctx context.Context, ID string) (string, string, error) {
	r, _, err := d.client.CopyFromContainer(ctx, ID, "/tmp/l")
	if err != nil {
		// Not really something that we should treat as an error for now
		if strings.Contains(err.Error(), "No such container:path") {
			return "", "", nil
		}
		return "", "", err
	}

	stdout, err := readTar(r)
	if err != nil {
		return "", "", err
	}

	r, _, err = d.client.CopyFromContainer(context.TODO(), ID, "/tmp/t")
	if err != nil {
		return "", "", err
	}

	timing, err := readTar(r)
	if err != nil {
		return "", "", err
	}

	return stdout, timing, nil
}

// StopHost stops a managed host
func (d *DockerProvider) StopHost(ctx context.Context, ID string) error {
	return d.deleteContainer(ctx, ID)
}

func readTar(r io.ReadCloser) (string, error) {
	defer r.Close()
	tr := tar.NewReader(r)

	_, err := tr.Next()
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(tr)

	return buf.String(), nil
}
