package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// ContainerConfig represents configuration for creating a container
type ContainerConfig struct {
	Name        string
	Image       string
	Env         []string
	Cmd         []string
	Entrypoint  []string
	Memory      int64  // in bytes
	CPU         int64  // CPU shares
	PortMap     map[string]string // containerPort -> hostPort
}

// CreateContainer creates a new container with the specified configuration
func (c *Client) CreateContainer(ctx context.Context, config ContainerConfig) (string, error) {
	// Build port bindings
	portBindings := nat.PortMap{}
	exposedPorts := nat.PortSet{}
	
	for containerPort, hostPort := range config.PortMap {
		natPort, err := nat.NewPort("tcp", containerPort)
		if err != nil {
			return "", fmt.Errorf("invalid port %s: %w", containerPort, err)
		}
		
		exposedPorts[natPort] = struct{}{}
		portBindings[natPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			},
		}
	}

	// Build container config
	containerCfg := &container.Config{
		Image:        config.Image,
		Env:          config.Env,
		ExposedPorts: exposedPorts,
	}

	if len(config.Cmd) > 0 {
		containerCfg.Cmd = config.Cmd
	}

	if len(config.Entrypoint) > 0 {
		containerCfg.Entrypoint = config.Entrypoint
	}

	// Build host config with resource limits
	hostCfg := &container.HostConfig{
		PortBindings: portBindings,
	}

	if config.Memory > 0 {
		hostCfg.Memory = config.Memory
	}

	if config.CPU > 0 {
		hostCfg.CPUShares = config.CPU
	}

	// Create the container
	resp, err := c.cli.ContainerCreate(
		ctx,
		containerCfg,
		hostCfg,
		&network.NetworkingConfig{},
		nil,
		config.Name,
	)

	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// StartContainer starts a container by ID
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer stops a running container
func (c *Client) StopContainer(ctx context.Context, containerID string, timeoutSeconds int) error {
	timeout := timeoutSeconds
	err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

// RemoveContainer removes a container
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: force})
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	return nil
}

// ListContainers returns a list of containers
func (c *Client) ListContainers(ctx context.Context, all bool) ([]types.Container, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	return containers, nil
}

// GetContainerStatus returns the status of a container
func (c *Client) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}
	return inspect.State.Status, nil
}

// GetContainerInspect returns full container inspection details
func (c *Client) GetContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("failed to inspect container: %w", err)
	}
	return inspect, nil
}
