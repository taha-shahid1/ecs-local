package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

// CreateNetwork creates a new Docker network for a task
func (c *Client) CreateNetwork(ctx context.Context, networkName string) (string, error) {
	enableIPv6 := false
	resp, err := c.cli.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Driver:     "bridge",
		EnableIPv6: &enableIPv6,
		Internal:   false,
		Attachable: true,
		Labels: map[string]string{
			"com.ecs-local.managed": "true",
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to create network %s: %w", networkName, err)
	}

	return resp.ID, nil
}

// RemoveNetwork removes a Docker network
func (c *Client) RemoveNetwork(ctx context.Context, networkID string) error {
	err := c.cli.NetworkRemove(ctx, networkID)
	if err != nil {
		return fmt.Errorf("failed to remove network %s: %w", networkID, err)
	}
	return nil
}

// ConnectContainerToNetwork connects a container to a network with optional aliases
func (c *Client) ConnectContainerToNetwork(ctx context.Context, networkID, containerID string, aliases []string) error {
	endpointConfig := &network.EndpointSettings{
		Aliases: aliases,
	}

	err := c.cli.NetworkConnect(ctx, networkID, containerID, endpointConfig)
	if err != nil {
		return fmt.Errorf("failed to connect container %s to network %s: %w", containerID, networkID, err)
	}

	return nil
}

// DisconnectContainerFromNetwork disconnects a container from a network
func (c *Client) DisconnectContainerFromNetwork(ctx context.Context, networkID, containerID string, force bool) error {
	err := c.cli.NetworkDisconnect(ctx, networkID, containerID, force)
	if err != nil {
		return fmt.Errorf("failed to disconnect container %s from network %s: %w", containerID, networkID, err)
	}

	return nil
}
