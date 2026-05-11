package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
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
	Memory      int64
	CPU         int64
	PortMap     map[string]string
	HealthCheck *HealthCheckConfig
	Mounts      []MountConfig
}

// MountConfig represents a volume mount
type MountConfig struct {
	Source   string
	Target   string
	ReadOnly bool
	Type     string
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Test        []string
	Interval    int64
	Timeout     int64
	Retries     int
	StartPeriod int64
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

	if config.HealthCheck != nil {
		containerCfg.Healthcheck = &container.HealthConfig{
			Test:        config.HealthCheck.Test,
			Interval:    time.Duration(config.HealthCheck.Interval),
			Timeout:     time.Duration(config.HealthCheck.Timeout),
			Retries:     config.HealthCheck.Retries,
			StartPeriod: time.Duration(config.HealthCheck.StartPeriod),
		}
	}

	// Build mounts
	var mounts []mount.Mount
	for _, m := range config.Mounts {
		mountType := mount.TypeBind
		if m.Type == "volume" {
			mountType = mount.TypeVolume
		}

		mounts = append(mounts, mount.Mount{
			Type:     mountType,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// Build host config with resource limits
	hostCfg := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
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

// GetContainerHealth returns the health status of a container
func (c *Client) GetContainerHealth(ctx context.Context, containerID string) (string, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	if inspect.State.Health == nil {
		return "none", nil
	}

	return inspect.State.Health.Status, nil
}
