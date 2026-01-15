package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
)

// CreateVolume creates a new Docker volume
func (c *Client) CreateVolume(ctx context.Context, volumeName string) (string, error) {
	vol, err := c.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name: volumeName,
		Labels: map[string]string{
			"com.ecs-local.managed": "true",
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to create volume %s: %w", volumeName, err)
	}

	return vol.Name, nil
}

// RemoveVolume removes a Docker volume
func (c *Client) RemoveVolume(ctx context.Context, volumeName string, force bool) error {
	err := c.cli.VolumeRemove(ctx, volumeName, force)
	if err != nil {
		return fmt.Errorf("failed to remove volume %s: %w", volumeName, err)
	}
	return nil
}
