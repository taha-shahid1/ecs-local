package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// ExecConfig represents configuration for executing a command in a container
type ExecConfig struct {
	ContainerID string
	Cmd         []string
	Interactive bool
	Tty         bool
	WorkingDir  string
	Env         []string
	User        string
}

// ExecInteractive executes a command in a container with interactive terminal
func (c *Client) ExecInteractive(ctx context.Context, config ExecConfig) error {
	execConfig := container.ExecOptions{
		AttachStdin:  config.Interactive,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          config.Tty,
		Cmd:          config.Cmd,
		WorkingDir:   config.WorkingDir,
		Env:          config.Env,
		User:         config.User,
	}

	execIDResp, err := c.cli.ContainerExecCreate(ctx, config.ContainerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}

	resp, err := c.cli.ContainerExecAttach(ctx, execIDResp.ID, container.ExecStartOptions{
		Tty: config.Tty,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer resp.Close()

	if config.Tty {
		_, err = io.Copy(os.Stdout, resp.Reader)
	} else {
		_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader)
	}

	if err != nil && err != io.EOF {
		return fmt.Errorf("error during exec: %w", err)
	}

	inspect, err := c.cli.ContainerExecInspect(ctx, execIDResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec instance: %w", err)
	}

	if inspect.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d", inspect.ExitCode)
	}

	return nil
}

// ExecStart executes a command in a container and returns output
func (c *Client) ExecStart(ctx context.Context, containerID string, cmd []string) (string, error) {
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	}

	execIDResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec instance: %w", err)
	}

	resp, err := c.cli.ContainerExecAttach(ctx, execIDResp.ID, container.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer resp.Close()

	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	return string(output), nil
}

// ContainerExecAttachOptions represents options for attaching to exec with stdin
type ContainerExecAttachOptions struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// ExecWithIO executes a command with custom IO streams
func (c *Client) ExecWithIO(ctx context.Context, containerID string, cmd []string, opts ContainerExecAttachOptions) (int, error) {
	execConfig := container.ExecOptions{
		AttachStdin:  opts.Stdin != nil,
		AttachStdout: opts.Stdout != nil,
		AttachStderr: opts.Stderr != nil,
		Tty:          false,
		Cmd:          cmd,
	}

	execIDResp, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return -1, fmt.Errorf("failed to create exec instance: %w", err)
	}

	resp, err := c.cli.ContainerExecAttach(ctx, execIDResp.ID, container.ExecStartOptions{})
	if err != nil {
		return -1, fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer resp.Close()

	if opts.Stdin != nil {
		go func() {
			_, _ = io.Copy(resp.Conn, opts.Stdin)
			_ = resp.CloseWrite()
		}()
	}

	if opts.Stdout != nil && opts.Stderr != nil {
		_, err = stdcopy.StdCopy(opts.Stdout, opts.Stderr, resp.Reader)
	} else if opts.Stdout != nil {
		_, err = io.Copy(opts.Stdout, resp.Reader)
	}

	if err != nil && err != io.EOF {
		return -1, fmt.Errorf("error during exec: %w", err)
	}

	inspect, err := c.cli.ContainerExecInspect(ctx, execIDResp.ID)
	if err != nil {
		return -1, fmt.Errorf("failed to inspect exec instance: %w", err)
	}

	return inspect.ExitCode, nil
}

// GetContainerID returns container ID from container name
func (c *Client) GetContainerID(ctx context.Context, containerName string) (string, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cont := range containers {
		for _, name := range cont.Names {
			if name == "/"+containerName || name == containerName {
				return cont.ID, nil
			}
		}
	}

	return "", fmt.Errorf("container not found: %s", containerName)
}
