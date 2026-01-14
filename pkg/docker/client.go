package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK
type Client struct {
	cli *client.Client
}

// NewClient initializes a Docker client and verifies daemon connection
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &Client{cli: cli}, nil
}

// Close closes the Docker client connection
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

// Ping checks if the Docker daemon is responsive
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	return err
}

// Info returns Docker daemon information
func (c *Client) Info(ctx context.Context) (string, error) {
	info, err := c.cli.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get Docker info: %w", err)
	}
	return fmt.Sprintf("Docker %s (%s)", info.ServerVersion, info.OperatingSystem), nil
}

// Version returns Docker daemon version
func (c *Client) Version(ctx context.Context) (string, error) {
	version, err := c.cli.ServerVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get Docker version: %w", err)
	}
	return version.Version, nil
}
