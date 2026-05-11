package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// PullProgress represents the progress of an image pull operation
type PullProgress struct {
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	Progress string `json:"progress"`
	ID       string `json:"id"`
}

// PullImage pulls a Docker image from a registry
func (c *Client) PullImage(ctx context.Context, imageName string, showProgress bool) error {
	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer func() {
		_ = reader.Close()
	}()

	if showProgress {
		return c.displayPullProgress(reader)
	}

	_, err = io.Copy(io.Discard, reader)
	return err
}

// displayPullProgress displays image pull progress to stdout
func (c *Client) displayPullProgress(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	var lastStatus string

	for {
		var progress PullProgress
		if err := decoder.Decode(&progress); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode progress: %w", err)
		}

		if progress.Status != lastStatus {
			if progress.ID != "" {
				fmt.Printf("%s: %s\n", progress.ID, progress.Status)
			} else {
				fmt.Printf("%s\n", progress.Status)
			}
			lastStatus = progress.Status
		}
	}

	return nil
}

// ImageExists checks if an image exists locally
func (c *Client) ImageExists(ctx context.Context, imageName string) (bool, error) {
	_, _, err := c.cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect image: %w", err)
	}
	return true, nil
}

// ListImages returns a list of images on the system
func (c *Client) ListImages(ctx context.Context) ([]string, error) {
	images, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var imageNames []string
	for _, img := range images {
		if len(img.RepoTags) > 0 {
			imageNames = append(imageNames, img.RepoTags...)
		}
	}

	return imageNames, nil
}

// RemoveImage removes an image from the system
func (c *Client) RemoveImage(ctx context.Context, imageName string, force bool) error {
	opts := image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	}

	_, err := c.cli.ImageRemove(ctx, imageName, opts)
	if err != nil {
		return fmt.Errorf("failed to remove image %s: %w", imageName, err)
	}

	return nil
}
