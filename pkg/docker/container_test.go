package docker

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestClient_CreateContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	// Ensure image exists
	err = client.PullImage(ctx, "alpine:latest", false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	config := ContainerConfig{
		Name:  "test-container",
		Image: "alpine:latest",
		Cmd:   []string{"echo", "hello"},
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		t.Errorf("failed to create container: %v", err)
		return
	}
	defer client.RemoveContainer(ctx, containerID, true)

	if containerID == "" {
		t.Error("expected non-empty container ID")
	}

	t.Logf("Created container: %s", containerID)
}

func TestClient_CreateContainer_WithResources(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	err = client.PullImage(ctx, "alpine:latest", false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	config := ContainerConfig{
		Name:   "test-container-resources",
		Image:  "alpine:latest",
		Cmd:    []string{"sleep", "1"},
		Memory: 128 * 1024 * 1024, // 128MB
		CPU:    512,               // CPU shares
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		t.Errorf("failed to create container with resources: %v", err)
		return
	}
	defer client.RemoveContainer(ctx, containerID, true)

	t.Logf("Created container with resources: %s", containerID)
}

func TestClient_CreateContainer_WithEnv(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	err = client.PullImage(ctx, "alpine:latest", false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	config := ContainerConfig{
		Name:  "test-container-env",
		Image: "alpine:latest",
		Env:   []string{"TEST_VAR=test_value", "ANOTHER=value"},
		Cmd:   []string{"env"},
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		t.Errorf("failed to create container with env: %v", err)
		return
	}
	defer client.RemoveContainer(ctx, containerID, true)

	t.Logf("Created container with env vars: %s", containerID)
}

func TestClient_CreateContainer_WithPorts(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	err = client.PullImage(ctx, "nginx:alpine", false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	config := ContainerConfig{
		Name:  "test-container-ports",
		Image: "nginx:alpine",
		PortMap: map[string]string{
			"80": "8080",
		},
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		t.Errorf("failed to create container with ports: %v", err)
		return
	}
	defer client.RemoveContainer(ctx, containerID, true)

	t.Logf("Created container with port mapping: %s", containerID)
}

func TestClient_StartContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	err = client.PullImage(ctx, "alpine:latest", false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	config := ContainerConfig{
		Name:  "test-container-start",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "2"},
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		t.Skipf("failed to create container: %v", err)
	}
	defer client.RemoveContainer(ctx, containerID, true)

	err = client.StartContainer(ctx, containerID)
	if err != nil {
		t.Errorf("failed to start container: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	status, err := client.GetContainerStatus(ctx, containerID)
	if err != nil {
		t.Errorf("failed to get container status: %v", err)
	}

	if status != "running" && status != "exited" {
		t.Errorf("unexpected status: %s", status)
	}

	t.Logf("Container status: %s", status)
}

func TestClient_StopContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	err = client.PullImage(ctx, "alpine:latest", false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	config := ContainerConfig{
		Name:  "test-container-stop",
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "30"},
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		t.Skipf("failed to create container: %v", err)
	}
	defer client.RemoveContainer(ctx, containerID, true)

	err = client.StartContainer(ctx, containerID)
	if err != nil {
		t.Skipf("failed to start container: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = client.StopContainer(ctx, containerID, 5)
	if err != nil {
		t.Errorf("failed to stop container: %v", err)
	}

	status, err := client.GetContainerStatus(ctx, containerID)
	if err != nil {
		t.Errorf("failed to get container status: %v", err)
	}

	if status != "exited" {
		t.Errorf("expected exited status, got: %s", status)
	}
}

func TestClient_ListContainers(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	containers, err := client.ListContainers(ctx, true)
	if err != nil {
		t.Errorf("failed to list containers: %v", err)
	}

	t.Logf("Found %d containers", len(containers))
}

func TestClient_ContainerLifecycle(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker daemon not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	err = client.PullImage(ctx, "alpine:latest", false)
	if err != nil {
		t.Skipf("failed to pull test image: %v", err)
	}

	// Create
	config := ContainerConfig{
		Name:  fmt.Sprintf("test-lifecycle-%d", time.Now().Unix()),
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "5"},
	}

	containerID, err := client.CreateContainer(ctx, config)
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	t.Logf("Created: %s", containerID)

	// Start
	err = client.StartContainer(ctx, containerID)
	if err != nil {
		t.Errorf("failed to start: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Check status
	status, err := client.GetContainerStatus(ctx, containerID)
	if err != nil {
		t.Errorf("failed to get status: %v", err)
	}
	t.Logf("Status: %s", status)

	// Stop
	err = client.StopContainer(ctx, containerID, 5)
	if err != nil {
		t.Errorf("failed to stop: %v", err)
	}

	// Remove
	err = client.RemoveContainer(ctx, containerID, false)
	if err != nil {
		t.Errorf("failed to remove: %v", err)
	}

	t.Log("Lifecycle complete")
}
