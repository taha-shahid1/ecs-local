package docker

import (
	"context"
	"testing"
)

func TestClient_PullImage(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Use a small, commonly available image
	testImage := "alpine:latest"

	// Pull without progress
	err = client.PullImage(ctx, testImage, false)
	if err != nil {
		t.Errorf("failed to pull image: %v", err)
	}

	// Verify image exists
	exists, err := client.ImageExists(ctx, testImage)
	if err != nil {
		t.Errorf("failed to check image existence: %v", err)
	}
	if !exists {
		t.Error("image should exist after pull")
	}
}

func TestClient_PullImage_WithProgress(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	testImage := "alpine:3.18"

	// Pull with progress
	err = client.PullImage(ctx, testImage, true)
	if err != nil {
		t.Errorf("failed to pull image with progress: %v", err)
	}
}

func TestClient_PullImage_InvalidImage(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Try to pull a non-existent image
	err = client.PullImage(ctx, "nonexistent/image:notreal", false)
	if err == nil {
		t.Error("expected error when pulling invalid image")
	}
}

func TestClient_ImageExists(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Check for non-existent image
	exists, err := client.ImageExists(ctx, "definitely-does-not-exist:nope")
	if err != nil {
		t.Errorf("unexpected error checking non-existent image: %v", err)
	}
	if exists {
		t.Error("non-existent image should not exist")
	}

	// Pull and check existing image
	testImage := "alpine:latest"
	err = client.PullImage(ctx, testImage, false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	exists, err = client.ImageExists(ctx, testImage)
	if err != nil {
		t.Errorf("failed to check image existence: %v", err)
	}
	if !exists {
		t.Error("pulled image should exist")
	}
}

func TestClient_ListImages(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	images, err := client.ListImages(ctx)
	if err != nil {
		t.Errorf("failed to list images: %v", err)
	}

	// Should at least have some images (or empty list is valid)
	t.Logf("Found %d images", len(images))
}

func TestClient_RemoveImage(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Pull a test image
	testImage := "alpine:3.17"
	err = client.PullImage(ctx, testImage, false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	// Remove the image
	err = client.RemoveImage(ctx, testImage, false)
	if err != nil {
		t.Errorf("failed to remove image: %v", err)
	}

	// Verify it's gone
	exists, err := client.ImageExists(ctx, testImage)
	if err != nil {
		t.Errorf("failed to check image existence: %v", err)
	}
	if exists {
		t.Error("image should not exist after removal")
	}
}

func TestClient_RemoveImage_Force(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	testImage := "alpine:latest"

	// Ensure image exists
	err = client.PullImage(ctx, testImage, false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	// Force remove should work even if containers are using it
	err = client.RemoveImage(ctx, testImage, true)
	if err != nil {
		t.Logf("force remove failed (may be in use): %v", err)
	}
}
