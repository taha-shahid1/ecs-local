package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
)

// LogOptions configures container log streaming
type LogOptions struct {
	Follow     bool
	Timestamps bool
	Tail       string
	Since      string
}

// StreamLogs streams container logs to stdout
func (c *Client) StreamLogs(ctx context.Context, containerID string, opts LogOptions) error {
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
	}

	if opts.Tail != "" {
		logOpts.Tail = opts.Tail
	}

	if opts.Since != "" {
		logOpts.Since = opts.Since
	}

	reader, err := c.cli.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	_, err = io.Copy(os.Stdout, reader)
	return err
}

// GetContainerName returns the name of a container by ID
func (c *Client) GetContainerName(ctx context.Context, containerID string) (string, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	if len(inspect.Name) > 0 && inspect.Name[0] == '/' {
		return inspect.Name[1:], nil
	}

	return inspect.Name, nil
}

// LogReader wraps a log reader with metadata
type LogReader struct {
	ContainerName string
	ContainerID   string
	Reader        io.ReadCloser
	Color         string
}

// StreamMultipleLogs streams logs from multiple containers with colored output
func (c *Client) StreamMultipleLogs(ctx context.Context, containerIDs []string, follow bool) error {
	if len(containerIDs) == 0 {
		return fmt.Errorf("no containers specified")
	}

	colors := []string{
		"\033[36m", // Cyan
		"\033[33m", // Yellow
		"\033[35m", // Magenta
		"\033[32m", // Green
		"\033[34m", // Blue
		"\033[31m", // Red
	}
	reset := "\033[0m"

	readers := make([]LogReader, 0, len(containerIDs))

	for i, containerID := range containerIDs {
		name, err := c.GetContainerName(ctx, containerID)
		if err != nil {
			name = containerID[:12]
		}

		logOpts := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     follow,
			Timestamps: true,
		}

		reader, err := c.cli.ContainerLogs(ctx, containerID, logOpts)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to get logs for %s: %v\n", name, err)
			continue
		}

		readers = append(readers, LogReader{
			ContainerName: name,
			ContainerID:   containerID,
			Reader:        reader,
			Color:         colors[i%len(colors)],
		})
	}

	if len(readers) == 0 {
		return fmt.Errorf("failed to start log streaming for any container")
	}

	// For multiple containers, read from each in goroutines
	done := make(chan struct{})
	for _, lr := range readers {
		go func(logReader LogReader) {
			defer func() {
				_ = logReader.Reader.Close()
			}()

			buffer := make([]byte, 8192)
			for {
				n, err := logReader.Reader.Read(buffer)
				if n > 0 {
					timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")
					fmt.Printf("%s[%s]%s %s | %s",
						logReader.Color,
						logReader.ContainerName,
						reset,
						timestamp,
						string(buffer[:n]))
				}

				if err != nil {
					if err != io.EOF {
						_, _ = fmt.Fprintf(os.Stderr, "\nError reading logs from %s: %v\n", logReader.ContainerName, err)
					}
					break
				}
			}
		}(lr)
	}

	if follow {
		<-done // Block forever if following
	} else {
		// Wait a bit for logs to finish
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
